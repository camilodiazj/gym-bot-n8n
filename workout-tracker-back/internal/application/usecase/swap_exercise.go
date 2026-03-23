package usecase

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// SwapExerciseUseCase handles the business logic for swapping an exercise in a draft
type SwapExerciseUseCase struct {
	draftRepo repository.DraftRepository
}

// NewSwapExerciseUseCase creates a new SwapExerciseUseCase
func NewSwapExerciseUseCase(draftRepo repository.DraftRepository) *SwapExerciseUseCase {
	return &SwapExerciseUseCase{draftRepo: draftRepo}
}

// Execute swaps a primary exercise with one of its alternatives and persists the change
func (uc *SwapExerciseUseCase) Execute(ctx context.Context, code string, req *dto.SwapExerciseRequest) (*dto.SwapExerciseResponse, error) {
	if code == "" {
		return nil, apperror.NewValidationError("code is required")
	}

	draft, err := uc.draftRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Locate the exercise at the specified position
	_, exercise := draft.DraftData.FindExerciseAt(req.DayNumber, req.ExerciseOrder)
	if exercise == nil {
		return nil, apperror.NewNotFoundError("exercise not found at the specified position")
	}

	// Attempt the swap (validates that new_exercise_id is in alternatives)
	if !exercise.SwapExercise(req.NewExerciseID) {
		return nil, apperror.NewValidationError("new_exercise_id is not a valid alternative for this exercise")
	}

	// Persist the updated draft data
	if err := uc.draftRepo.UpdateDraftData(ctx, code, draft.DraftData); err != nil {
		return nil, err
	}

	// Enrich the swapped exercise with video_link and main_muscle
	exerciseIDs := []string{exercise.ExerciseID}
	for _, alt := range exercise.Alternatives {
		exerciseIDs = append(exerciseIDs, alt.ExerciseID)
	}

	meta, err := uc.draftRepo.EnrichExercises(ctx, exerciseIDs)
	if err != nil {
		return nil, err
	}

	// Apply enrichment
	if m, ok := meta[exercise.ExerciseID]; ok {
		exercise.VideoLink = m.VideoLink
		exercise.MainMuscle = m.MainMuscle
	}
	for i := range exercise.Alternatives {
		if m, ok := meta[exercise.Alternatives[i].ExerciseID]; ok {
			exercise.Alternatives[i].VideoLink = m.VideoLink
			exercise.Alternatives[i].MainMuscle = m.MainMuscle
		}
	}

	// Map to DTO
	altDTOs := make([]dto.DraftAlternativeDTO, 0, len(exercise.Alternatives))
	for _, alt := range exercise.Alternatives {
		altDTOs = append(altDTOs, dto.DraftAlternativeDTO{
			ExerciseID:  alt.ExerciseID,
			SpanishName: alt.SpanishName,
			MainMuscle:  alt.MainMuscle,
			VideoLink:   alt.VideoLink,
		})
	}

	return &dto.SwapExerciseResponse{
		Exercise: dto.DraftExerciseDTO{
			ExerciseID:    exercise.ExerciseID,
			SpanishName:   exercise.SpanishName,
			Pattern:       exercise.Pattern,
			Role:          exercise.Role,
			MainMuscle:    exercise.MainMuscle,
			VideoLink:     exercise.VideoLink,
			Sets:          exercise.Sets,
			Reps:          exercise.Reps,
			RIR:           exercise.RIR,
			RestSeconds:   exercise.RestSeconds,
			ExerciseOrder: exercise.ExerciseOrder,
			Alternatives:  altDTOs,
		},
	}, nil
}
