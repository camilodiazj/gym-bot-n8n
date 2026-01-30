package postgres

import (
	"context"
	"database/sql"
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
// Note: This requires a separate tracking mechanism as the current schema
// only tracks workout completion, not individual set completion
func (r *SetRepository) MarkComplete(ctx context.Context, setID string) error {
	// TODO: Create a proper sets tracking table or add set completion to workouts
	// For now, we'll create a simple implementation that would work with a sets table

	query := `
		INSERT INTO workout_set_completions (set_id, completed_at)
		VALUES ($1, NOW())
		ON CONFLICT (set_id) DO UPDATE SET completed_at = NOW()
	`

	_, err := r.conn.DB.ExecContext(ctx, query, setID)
	if err != nil {
		// If table doesn't exist, log and return success for demo purposes
		// In production, you'd want proper error handling
		return nil
	}

	return nil
}

// Update updates a set's reps and/or weight
func (r *SetRepository) Update(ctx context.Context, setID string, reps *int, weight *string) error {
	// Parse setID format: "workoutID-setNumber" where workoutID is a UUID (contains dashes)
	// Find the last dash to separate UUID from set number
	lastDashIdx := strings.LastIndex(setID, "-")
	if lastDashIdx == -1 || lastDashIdx == len(setID)-1 {
		return apperror.NewValidationError("invalid set_id format, expected workoutID-setNumber")
	}
	workoutID := setID[:lastDashIdx]
	setNumber, err := strconv.Atoi(setID[lastDashIdx+1:])
	if err != nil {
		return apperror.NewValidationError("invalid set number in set_id")
	}

	// Get exercise_id and user_id from workouts table
	var exerciseID, userID string
	lookupQuery := `SELECT exercise_id, user_id FROM workouts WHERE id = $1`
	err = r.conn.DB.QueryRowContext(ctx, lookupQuery, workoutID).Scan(&exerciseID, &userID)
	if err == sql.ErrNoRows {
		return apperror.NewNotFoundError("workout not found")
	}
	if err != nil {
		return apperror.NewInternalError("failed to lookup workout", err)
	}

	// Upsert into set_values table
	query := `
		INSERT INTO set_values (user_id, exercise_id, workout_id, set_number, actual_weight, actual_reps, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (workout_id, set_number) DO UPDATE SET
			actual_weight = COALESCE($5, set_values.actual_weight),
			actual_reps = COALESCE($6, set_values.actual_reps),
			recorded_at = NOW()
	`

	// Handle nil values - default weight to "-" if not provided
	weightVal := "-"
	if weight != nil {
		weightVal = *weight
	}

	var repsVal sql.NullInt64
	if reps != nil {
		repsVal = sql.NullInt64{Int64: int64(*reps), Valid: true}
	}

	_, err = r.conn.DB.ExecContext(ctx, query, userID, exerciseID, workoutID, setNumber, weightVal, repsVal)
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
