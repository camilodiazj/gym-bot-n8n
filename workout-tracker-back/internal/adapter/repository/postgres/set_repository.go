package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// SetRepository implements repository.SetRepository using PostgreSQL
type SetRepository struct {
	conn *Connection
}

// Ensure SetRepository implements the interface
var _ repository.SetRepository = (*SetRepository)(nil)

// NewSetRepository creates a new SetRepository
func NewSetRepository(conn *Connection) *SetRepository {
	return &SetRepository{conn: conn}
}

type parsedSetID struct {
	workoutID     string
	exerciseID    string // empty for primary sets
	setNumber     int
	isAlternative bool
}

func parseSetID(setID string) (parsedSetID, error) {
	if strings.Contains(setID, ":") {
		parts := strings.SplitN(setID, ":", 3)
		if len(parts) != 3 {
			return parsedSetID{}, fmt.Errorf("invalid alternative set_id format")
		}
		setNum, err := strconv.Atoi(parts[2])
		if err != nil {
			return parsedSetID{}, fmt.Errorf("invalid set number in alternative set_id")
		}
		return parsedSetID{
			workoutID:     parts[0],
			exerciseID:    parts[1],
			setNumber:     setNum,
			isAlternative: true,
		}, nil
	}

	lastDash := strings.LastIndex(setID, "-")
	if lastDash == -1 || lastDash == len(setID)-1 {
		return parsedSetID{}, fmt.Errorf("invalid set_id format")
	}
	setNum, err := strconv.Atoi(setID[lastDash+1:])
	if err != nil {
		return parsedSetID{}, fmt.Errorf("invalid set number")
	}
	return parsedSetID{
		workoutID:     setID[:lastDash],
		exerciseID:    "",
		setNumber:     setNum,
		isAlternative: false,
	}, nil
}

// GetByID retrieves a set by its ID
// Note: In the current schema, sets are part of the workouts table
// This implementation assumes sets are tracked separately or derived from workout data
func (r *SetRepository) GetByID(ctx context.Context, setID string) (*entity.Set, error) {
	// For now, we'll parse the setID which is in format "workoutID-setNumber"
	// In a real implementation, you might want a separate sets table

	// This is a placeholder implementation
	// TODO: Implement proper set tracking in database
	return nil, apperror.NewNotFoundError("set not found")
}

// MarkComplete marks a set as completed
func (r *SetRepository) MarkComplete(ctx context.Context, setID string) error {
	parsed, err := parseSetID(setID)
	if err != nil {
		return apperror.NewValidationError(err.Error())
	}

	var exerciseID, userID string
	if parsed.isAlternative {
		exerciseID = parsed.exerciseID
		if exerciseID == "" {
			return apperror.NewValidationError("alternative set_id missing exercise_id")
		}
		err = r.conn.DB.QueryRowContext(ctx,
			`SELECT user_id FROM workouts WHERE id = $1`,
			parsed.workoutID).Scan(&userID)
	} else {
		err = r.conn.DB.QueryRowContext(ctx,
			`SELECT exercise_id, user_id FROM workouts WHERE id = $1`,
			parsed.workoutID).Scan(&exerciseID, &userID)
	}
	if err == sql.ErrNoRows {
		return apperror.NewNotFoundError("workout not found")
	}
	if err != nil {
		return apperror.NewInternalError("failed to lookup workout", err)
	}

	// Upsert a completed marker into set_values
	query := `
		INSERT INTO set_values (user_id, exercise_id, workout_id, set_number, actual_weight, actual_reps, recorded_at)
		VALUES ($1, $2, $3, $4, '-', NULL, NOW())
		ON CONFLICT (workout_id, exercise_id, set_number) DO UPDATE SET
			recorded_at = NOW()
	`

	_, err = r.conn.DB.ExecContext(ctx, query, userID, exerciseID, parsed.workoutID, parsed.setNumber)
	if err != nil {
		return apperror.NewInternalError("failed to mark set complete", err)
	}

	return nil
}

// Update updates a set's reps and/or weight
func (r *SetRepository) Update(ctx context.Context, setID string, reps *int, weight *string) error {
	parsed, err := parseSetID(setID)
	if err != nil {
		return apperror.NewValidationError(err.Error())
	}

	var exerciseID, userID string
	if parsed.isAlternative {
		exerciseID = parsed.exerciseID
		if exerciseID == "" {
			return apperror.NewValidationError("alternative set_id missing exercise_id")
		}
		err = r.conn.DB.QueryRowContext(ctx,
			`SELECT user_id FROM workouts WHERE id = $1`,
			parsed.workoutID).Scan(&userID)
	} else {
		err = r.conn.DB.QueryRowContext(ctx,
			`SELECT exercise_id, user_id FROM workouts WHERE id = $1`,
			parsed.workoutID).Scan(&exerciseID, &userID)
	}
	if err == sql.ErrNoRows {
		return apperror.NewNotFoundError("workout not found")
	}
	if err != nil {
		return apperror.NewInternalError("failed to lookup workout", err)
	}

	// Upsert into set_values table
	// ON CONFLICT uses the 3-column constraint (workout_id, exercise_id, set_number)
	query := `
		INSERT INTO set_values (user_id, exercise_id, workout_id, set_number, actual_weight, actual_reps, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (workout_id, exercise_id, set_number) DO UPDATE SET
			actual_weight = COALESCE($5, set_values.actual_weight),
			actual_reps = COALESCE($6, set_values.actual_reps),
			recorded_at = NOW()
	`

	weightVal := "-"
	if weight != nil {
		weightVal = *weight
	}

	var repsVal sql.NullInt64
	if reps != nil {
		repsVal = sql.NullInt64{Int64: int64(*reps), Valid: true}
	}

	_, err = r.conn.DB.ExecContext(ctx, query, userID, exerciseID, parsed.workoutID, parsed.setNumber, weightVal, repsVal)
	if err != nil {
		return apperror.NewInternalError("failed to update set values", err)
	}

	return nil
}

// GetLastWeightsForExercise returns the last recorded weight for each set number
// This allows pre-filling weights when user does the same exercise in a future week
func (r *SetRepository) GetLastWeightsForExercise(ctx context.Context, userID, exerciseID string) (map[int]string, error) {
	query := `
		SELECT DISTINCT ON (set_number)
			set_number,
			actual_weight
		FROM set_values
		WHERE user_id = $1
		  AND exercise_id = $2
		  AND actual_weight IS NOT NULL
		  AND actual_weight != '-'
		ORDER BY set_number, recorded_at DESC
	`

	rows, err := r.conn.DB.QueryContext(ctx, query, userID, exerciseID)
	if err != nil {
		return nil, apperror.NewInternalError("failed to query last weights", err)
	}
	defer rows.Close()

	weights := make(map[int]string)
	for rows.Next() {
		var setNumber int
		var weight string
		if err := rows.Scan(&setNumber, &weight); err != nil {
			return nil, apperror.NewInternalError("failed to scan weight row", err)
		}
		weights[setNumber] = weight
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.NewInternalError("error iterating weight rows", err)
	}

	return weights, nil
}
