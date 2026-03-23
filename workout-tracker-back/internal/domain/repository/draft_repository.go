package repository

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// DraftReader defines read operations for draft routines
type DraftReader interface {
	// GetByCode retrieves a pending, non-expired draft by its short code
	GetByCode(ctx context.Context, code string) (*entity.DraftRoutine, error)
	// GetPhoneByUserID retrieves the user's full phone number for Kairos API callback
	GetPhoneByUserID(ctx context.Context, userID string) (string, error)
}

// DraftWriter defines write operations for draft routines
type DraftWriter interface {
	// UpdateDraftData persists the modified JSONB draft_data for a pending draft
	UpdateDraftData(ctx context.Context, code string, data entity.DraftData) error
	// MarkApproved sets draft status to 'approved'
	MarkApproved(ctx context.Context, code string) error
}

// ExerciseEnricher defines read operations for enriching exercise metadata
type ExerciseEnricher interface {
	// EnrichExercises fetches video_link and main_muscle for a batch of exercise IDs
	EnrichExercises(ctx context.Context, exerciseIDs []string) (map[string]ExerciseMeta, error)
}

// ExerciseMeta holds enrichment data fetched from the exercises table
type ExerciseMeta struct {
	VideoLink  string
	MainMuscle string
}

// DraftRepository combines all draft repository operations
type DraftRepository interface {
	DraftReader
	DraftWriter
	ExerciseEnricher
}
