package postgres

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
	"github.com/lib/pq"
)

// ExerciseRepository implements repository.ExerciseRepository using PostgreSQL
type ExerciseRepository struct {
	conn *Connection
}

// Ensure ExerciseRepository implements the interface
var _ repository.ExerciseRepository = (*ExerciseRepository)(nil)

// NewExerciseRepository creates a new ExerciseRepository
func NewExerciseRepository(conn *Connection) *ExerciseRepository {
	return &ExerciseRepository{conn: conn}
}

// GetByID retrieves an exercise by its ID
func (r *ExerciseRepository) GetByID(ctx context.Context, exerciseID string) (*entity.ExerciseCatalog, error) {
	query := `
		SELECT
			exercise_id,
			spanish_name,
			pattern,
			role,
			main_muscle,
			secondary_muscles,
			level,
			link,
			equipment
		FROM exercises
		WHERE exercise_id = $1
	`

	var exercise entity.ExerciseCatalog
	var secondaryMuscles pq.StringArray
	var link, equipment sql.NullString

	err := r.conn.DB.QueryRowContext(ctx, query, exerciseID).Scan(
		&exercise.ExerciseID,
		&exercise.SpanishName,
		&exercise.Pattern,
		&exercise.Role,
		&exercise.MainMuscle,
		&secondaryMuscles,
		&exercise.Level,
		&link,
		&equipment,
	)

	if err == sql.ErrNoRows {
		return nil, apperror.NewNotFoundError("exercise not found")
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query exercise", err)
	}

	exercise.SecondaryMuscles = []string(secondaryMuscles)
	exercise.Link = link.String
	exercise.Equipment = equipment.String

	return &exercise, nil
}

// FindAlternatives finds alternative exercises matching criteria
func (r *ExerciseRepository) FindAlternatives(
	ctx context.Context,
	pattern, role string,
	excludeIDs, excludeMuscles []string,
	limit int,
) ([]*entity.ExerciseCatalog, error) {
	// Build dynamic query
	query := `
		SELECT
			exercise_id,
			spanish_name,
			pattern,
			role,
			main_muscle,
			secondary_muscles,
			level,
			link,
			equipment
		FROM exercises
		WHERE pattern = $1 AND role = $2
	`
	args := []interface{}{pattern, role}
	argIndex := 3

	// Exclude specific exercise IDs
	if len(excludeIDs) > 0 {
		query += " AND exercise_id != ALL($" + strconv.Itoa(argIndex) + ")"
		args = append(args, pq.StringArray(excludeIDs))
		argIndex++
	}

	// Exclude disliked muscles
	if len(excludeMuscles) > 0 {
		query += " AND main_muscle != ALL($" + strconv.Itoa(argIndex) + ")"
		args = append(args, pq.StringArray(excludeMuscles))
		argIndex++
	}

	query += " ORDER BY RANDOM() LIMIT $" + strconv.Itoa(argIndex)
	args = append(args, limit)

	rows, err := r.conn.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperror.NewInternalError("failed to find alternatives", err)
	}
	defer rows.Close()

	exercises := make([]*entity.ExerciseCatalog, 0)
	for rows.Next() {
		var exercise entity.ExerciseCatalog
		var secondaryMuscles pq.StringArray
		var link, equipment sql.NullString

		if err := rows.Scan(
			&exercise.ExerciseID,
			&exercise.SpanishName,
			&exercise.Pattern,
			&exercise.Role,
			&exercise.MainMuscle,
			&secondaryMuscles,
			&exercise.Level,
			&link,
			&equipment,
		); err != nil {
			return nil, apperror.NewInternalError("failed to scan exercise", err)
		}

		exercise.SecondaryMuscles = []string(secondaryMuscles)
		exercise.Link = link.String
		exercise.Equipment = equipment.String
		exercises = append(exercises, &exercise)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.NewInternalError("error iterating exercises", err)
	}

	return exercises, nil
}

// FindByPatternAndRole finds exercises by pattern and role
func (r *ExerciseRepository) FindByPatternAndRole(ctx context.Context, pattern, role string) ([]*entity.ExerciseCatalog, error) {
	query := `
		SELECT
			exercise_id,
			spanish_name,
			pattern,
			role,
			main_muscle,
			secondary_muscles,
			level,
			link,
			equipment
		FROM exercises
		WHERE pattern = $1 AND role = $2
		ORDER BY spanish_name
	`

	rows, err := r.conn.DB.QueryContext(ctx, query, pattern, role)
	if err != nil {
		return nil, apperror.NewInternalError("failed to find exercises", err)
	}
	defer rows.Close()

	exercises := make([]*entity.ExerciseCatalog, 0)
	for rows.Next() {
		var exercise entity.ExerciseCatalog
		var secondaryMuscles pq.StringArray
		var link, equipment sql.NullString

		if err := rows.Scan(
			&exercise.ExerciseID,
			&exercise.SpanishName,
			&exercise.Pattern,
			&exercise.Role,
			&exercise.MainMuscle,
			&secondaryMuscles,
			&exercise.Level,
			&link,
			&equipment,
		); err != nil {
			return nil, apperror.NewInternalError("failed to scan exercise", err)
		}

		exercise.SecondaryMuscles = []string(secondaryMuscles)
		exercise.Link = link.String
		exercise.Equipment = equipment.String
		exercises = append(exercises, &exercise)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.NewInternalError("error iterating exercises", err)
	}

	return exercises, nil
}
