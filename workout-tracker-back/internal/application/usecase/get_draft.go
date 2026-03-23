package usecase

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// GetDraftUseCase handles the business logic for retrieving a draft routine
type GetDraftUseCase struct {
	draftRepo repository.DraftRepository
}

// NewGetDraftUseCase creates a new GetDraftUseCase
func NewGetDraftUseCase(draftRepo repository.DraftRepository) *GetDraftUseCase {
	return &GetDraftUseCase{draftRepo: draftRepo}
}

// Execute retrieves a draft routine by code, enriches exercise data, and maps to DTO
func (uc *GetDraftUseCase) Execute(ctx context.Context, code string) (*dto.GetDraftResponse, error) {
	if code == "" {
		return nil, apperror.NewValidationError("code is required")
	}

	draft, err := uc.draftRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Collect all exercise IDs (primaries + alternatives) for batch enrichment
	exerciseIDs := collectExerciseIDs(&draft.DraftData)

	meta, err := uc.draftRepo.EnrichExercises(ctx, exerciseIDs)
	if err != nil {
		return nil, err
	}

	// Apply enrichment to the draft data
	enrichDraftData(&draft.DraftData, meta)

	// Map to DTO
	response := &dto.GetDraftResponse{
		DraftID:      draft.DraftID,
		Code:         draft.Code,
		WeekSchedule: draft.DraftData.WeekSchedule,
		Goal:         draft.DraftData.Goal,
		Level:        draft.DraftData.Level,
		Days:         mapDraftDays(draft.DraftData.Days),
	}

	return response, nil
}

// collectExerciseIDs gathers all unique exercise IDs from the draft (primaries + alternatives)
func collectExerciseIDs(data *entity.DraftData) []string {
	seen := make(map[string]bool)
	var ids []string

	for _, day := range data.Days {
		for _, ex := range day.Exercises {
			if !seen[ex.ExerciseID] {
				seen[ex.ExerciseID] = true
				ids = append(ids, ex.ExerciseID)
			}
			for _, alt := range ex.Alternatives {
				if !seen[alt.ExerciseID] {
					seen[alt.ExerciseID] = true
					ids = append(ids, alt.ExerciseID)
				}
			}
		}
	}

	return ids
}

// enrichDraftData applies video_link and main_muscle from the exercises table
func enrichDraftData(data *entity.DraftData, meta map[string]repository.ExerciseMeta) {
	for i := range data.Days {
		for j := range data.Days[i].Exercises {
			ex := &data.Days[i].Exercises[j]
			if m, ok := meta[ex.ExerciseID]; ok {
				ex.VideoLink = m.VideoLink
				ex.MainMuscle = m.MainMuscle
			}
			for k := range ex.Alternatives {
				alt := &ex.Alternatives[k]
				if m, ok := meta[alt.ExerciseID]; ok {
					alt.VideoLink = m.VideoLink
					alt.MainMuscle = m.MainMuscle
				}
			}
		}
	}
}

// mapDraftDays converts entity days to DTO days
func mapDraftDays(days []entity.DraftDay) []dto.DraftDayDTO {
	result := make([]dto.DraftDayDTO, 0, len(days))
	for _, day := range days {
		dayDTO := dto.DraftDayDTO{
			DayNumber: day.DayNumber,
			Title:     day.Title,
			Exercises: mapDraftExercises(day.Exercises),
		}
		result = append(result, dayDTO)
	}
	return result
}

// mapDraftExercises converts entity exercises to DTO exercises
func mapDraftExercises(exercises []entity.DraftExercise) []dto.DraftExerciseDTO {
	result := make([]dto.DraftExerciseDTO, 0, len(exercises))
	for _, ex := range exercises {
		exDTO := dto.DraftExerciseDTO{
			ExerciseID:    ex.ExerciseID,
			SpanishName:   ex.SpanishName,
			Pattern:       ex.Pattern,
			Role:          ex.Role,
			MainMuscle:    ex.MainMuscle,
			VideoLink:     ex.VideoLink,
			Sets:          ex.Sets,
			Reps:          ex.Reps,
			RIR:           ex.RIR,
			RestSeconds:   ex.RestSeconds,
			ExerciseOrder: ex.ExerciseOrder,
			Alternatives:  mapDraftAlternatives(ex.Alternatives),
		}
		result = append(result, exDTO)
	}
	return result
}

// mapDraftAlternatives converts entity alternatives to DTO alternatives
func mapDraftAlternatives(alts []entity.DraftAlternative) []dto.DraftAlternativeDTO {
	result := make([]dto.DraftAlternativeDTO, 0, len(alts))
	for _, alt := range alts {
		result = append(result, dto.DraftAlternativeDTO{
			ExerciseID:  alt.ExerciseID,
			SpanishName: alt.SpanishName,
			MainMuscle:  alt.MainMuscle,
			VideoLink:   alt.VideoLink,
		})
	}
	return result
}
