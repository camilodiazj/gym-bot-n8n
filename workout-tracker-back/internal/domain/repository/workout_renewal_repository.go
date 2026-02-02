package repository

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// WorkoutRenewalReader defines read operations for workout renewal
type WorkoutRenewalReader interface {
	// GetUserWorkoutExercises retrieves all workout exercises for a user
	GetUserWorkoutExercises(ctx context.Context, userID string) ([]*entity.WorkoutExercise, error)
	// GetDistinctExerciseIDs returns distinct exercise IDs for a user's workouts
	GetDistinctExerciseIDs(ctx context.Context, userID string) ([]string, error)
}

// WorkoutRenewalWriter defines write operations for workout renewal
type WorkoutRenewalWriter interface {
	// DeleteUserWorkouts deletes all workouts for a user
	DeleteUserWorkouts(ctx context.Context, userID string) error
	// UpdateExerciseID updates the exercise_id for a specific workout
	UpdateExerciseID(ctx context.Context, workoutID, newExerciseID string) error
	// BatchUpdateExercises applies multiple exercise rotations in a single transaction
	BatchUpdateExercises(ctx context.Context, rotations []*entity.ExerciseRotation) error
	// ResetWeeksToOne updates all workouts to week=1 (for MANTENER flow)
	ResetWeeksToOne(ctx context.Context, userID string) error
}

// WorkoutRenewalRepository combines all workout renewal operations
// Following Interface Segregation Principle (ISP)
type WorkoutRenewalRepository interface {
	WorkoutRenewalReader
	WorkoutRenewalWriter
}
