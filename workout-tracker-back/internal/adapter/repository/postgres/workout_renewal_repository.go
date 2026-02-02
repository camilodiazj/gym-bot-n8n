package postgres

import (
	"context"
	"database/sql"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// WorkoutRenewalRepository implements repository.WorkoutRenewalRepository
type WorkoutRenewalRepository struct {
	conn *Connection
}

// Ensure WorkoutRenewalRepository implements the interface
var _ repository.WorkoutRenewalRepository = (*WorkoutRenewalRepository)(nil)

// NewWorkoutRenewalRepository creates a new WorkoutRenewalRepository
func NewWorkoutRenewalRepository(conn *Connection) *WorkoutRenewalRepository {
	return &WorkoutRenewalRepository{conn: conn}
}

// GetUserWorkoutExercises retrieves all workout exercises for a user with exercise metadata
func (r *WorkoutRenewalRepository) GetUserWorkoutExercises(ctx context.Context, userID string) ([]*entity.WorkoutExercise, error) {
	query := `
		SELECT
			w.id,
			w.user_id,
			w.week,
			w.day_name,
			w.exercise_id,
			w.sets,
			w.reps,
			w.rir,
			w."rest-seconds",
			w.tempo,
			w.exercise_order,
			e.role,
			e.pattern
		FROM workouts w
		LEFT JOIN exercises e ON w.exercise_id = e.exercise_id
		WHERE w.user_id = $1
		ORDER BY w.week, w.day_name, w.exercise_order
	`

	rows, err := r.conn.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, apperror.NewInternalError("failed to query workouts", err)
	}
	defer rows.Close()

	workouts := make([]*entity.WorkoutExercise, 0)
	for rows.Next() {
		var w entity.WorkoutExercise
		var rir sql.NullString
		var restSeconds sql.NullInt64
		var tempo sql.NullString
		var role sql.NullString
		var pattern sql.NullString

		if err := rows.Scan(
			&w.ID,
			&w.UserID,
			&w.Week,
			&w.DayName,
			&w.ExerciseID,
			&w.Sets,
			&w.Reps,
			&rir,
			&restSeconds,
			&tempo,
			&w.ExerciseOrder,
			&role,
			&pattern,
		); err != nil {
			return nil, apperror.NewInternalError("failed to scan workout", err)
		}

		w.RIR = rir.String
		w.RestSeconds = int(restSeconds.Int64)
		w.Tempo = tempo.String
		w.Role = role.String
		w.Pattern = pattern.String
		workouts = append(workouts, &w)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.NewInternalError("error iterating workouts", err)
	}

	return workouts, nil
}

// GetDistinctExerciseIDs returns distinct exercise IDs for a user's workouts
func (r *WorkoutRenewalRepository) GetDistinctExerciseIDs(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT DISTINCT exercise_id
		FROM workouts
		WHERE user_id = $1
	`

	rows, err := r.conn.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, apperror.NewInternalError("failed to query distinct exercises", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, apperror.NewInternalError("failed to scan exercise id", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.NewInternalError("error iterating exercise ids", err)
	}

	return ids, nil
}

// DeleteUserWorkouts deletes all workouts for a user
func (r *WorkoutRenewalRepository) DeleteUserWorkouts(ctx context.Context, userID string) error {
	query := `
		DELETE FROM workouts
		WHERE user_id = $1
	`

	_, err := r.conn.DB.ExecContext(ctx, query, userID)
	if err != nil {
		return apperror.NewInternalError("failed to delete workouts", err)
	}

	return nil
}

// UpdateExerciseID updates the exercise_id for a specific workout
func (r *WorkoutRenewalRepository) UpdateExerciseID(ctx context.Context, workoutID, newExerciseID string) error {
	query := `
		UPDATE workouts
		SET exercise_id = $2
		WHERE id = $1
	`

	result, err := r.conn.DB.ExecContext(ctx, query, workoutID, newExerciseID)
	if err != nil {
		return apperror.NewInternalError("failed to update exercise", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return apperror.NewNotFoundError("workout not found")
	}

	return nil
}

// BatchUpdateExercises applies multiple exercise rotations in a transaction
func (r *WorkoutRenewalRepository) BatchUpdateExercises(ctx context.Context, rotations []*entity.ExerciseRotation) error {
	if len(rotations) == 0 {
		return nil
	}

	tx, err := r.conn.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.NewInternalError("failed to begin transaction", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE workouts
		SET exercise_id = $2
		WHERE id = $1
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return apperror.NewInternalError("failed to prepare statement", err)
	}
	defer stmt.Close()

	for _, rotation := range rotations {
		_, err := stmt.ExecContext(ctx, rotation.WorkoutID, rotation.NewExerciseID)
		if err != nil {
			return apperror.NewInternalError("failed to update exercise rotation", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return apperror.NewInternalError("failed to commit transaction", err)
	}

	return nil
}

// ResetWeeksToOne updates all workouts to week=1 for MANTENER flow
func (r *WorkoutRenewalRepository) ResetWeeksToOne(ctx context.Context, userID string) error {
	query := `
		UPDATE workouts
		SET week = 1
		WHERE user_id = $1
	`

	_, err := r.conn.DB.ExecContext(ctx, query, userID)
	if err != nil {
		return apperror.NewInternalError("failed to reset weeks", err)
	}

	return nil
}
