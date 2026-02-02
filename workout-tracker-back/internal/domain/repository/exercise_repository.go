package repository

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// ExerciseReader defines read operations for the exercise catalog
type ExerciseReader interface {
	// GetByID retrieves an exercise by its ID
	GetByID(ctx context.Context, exerciseID string) (*entity.ExerciseCatalog, error)
	// FindAlternatives finds alternative exercises matching criteria
	// excludeIDs: exercise IDs to exclude from results
	// excludeMuscles: muscles to avoid (disliked by user)
	// limit: maximum number of results
	FindAlternatives(ctx context.Context, pattern, role string, excludeIDs, excludeMuscles []string, limit int) ([]*entity.ExerciseCatalog, error)
	// FindByPatternAndRole finds exercises by pattern and role
	FindByPatternAndRole(ctx context.Context, pattern, role string) ([]*entity.ExerciseCatalog, error)
}

// ExerciseRepository combines all exercise catalog operations
// Following Interface Segregation Principle (ISP)
type ExerciseRepository interface {
	ExerciseReader
}
