package repository

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// WorkoutReader defines read operations for workouts
type WorkoutReader interface {
	// GetTodayWorkout retrieves the workout scheduled for today for a user
	GetTodayWorkout(ctx context.Context, userID string) (*entity.Workout, error)
	// GetByID retrieves a workout by its ID
	GetByID(ctx context.Context, workoutID string) (*entity.Workout, error)
}

// WorkoutWriter defines write operations for workouts
type WorkoutWriter interface {
	// MarkComplete marks a workout as completed
	MarkComplete(ctx context.Context, workoutID string) error
}

// WorkoutRepository combines all workout repository operations
// Following Interface Segregation Principle (ISP)
type WorkoutRepository interface {
	WorkoutReader
	WorkoutWriter
}
