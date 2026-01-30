package repository

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// SetReader defines read operations for sets
type SetReader interface {
	// GetByID retrieves a set by its ID
	GetByID(ctx context.Context, setID string) (*entity.Set, error)
	// GetLastWeightsForExercise returns the last recorded weight for each set number of an exercise
	GetLastWeightsForExercise(ctx context.Context, userID, exerciseID string) (map[int]string, error)
}

// SetWriter defines write operations for sets
type SetWriter interface {
	// MarkComplete marks a set as completed
	MarkComplete(ctx context.Context, setID string) error
	// Update updates a set's reps and/or weight
	Update(ctx context.Context, setID string, reps *int, weight *string) error
}

// SetRepository combines all set repository operations
// Following Interface Segregation Principle (ISP)
type SetRepository interface {
	SetReader
	SetWriter
}
