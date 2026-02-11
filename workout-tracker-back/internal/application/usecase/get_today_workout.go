package usecase

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// GetTodayWorkoutUseCase handles the business logic for getting today's workout
type GetTodayWorkoutUseCase struct {
	workoutRepo repository.WorkoutReader
}

// NewGetTodayWorkoutUseCase creates a new GetTodayWorkoutUseCase
// Dependency Injection - depends on abstraction (interface), not concrete implementation
func NewGetTodayWorkoutUseCase(workoutRepo repository.WorkoutReader) *GetTodayWorkoutUseCase {
	return &GetTodayWorkoutUseCase{
		workoutRepo: workoutRepo,
	}
}

// Execute retrieves today's workout for a user and maps it to a DTO
func (uc *GetTodayWorkoutUseCase) Execute(ctx context.Context, userID string) (*dto.TodayWorkoutResponse, error) {
	if userID == "" {
		return nil, apperror.NewValidationError("user_id is required")
	}

	workout, err := uc.workoutRepo.GetTodayWorkout(ctx, userID)
	if err != nil {
		return nil, err
	}

	if workout == nil {
		return nil, apperror.NewNotFoundError("no workout scheduled for today")
	}

	// Map domain entity to DTO
	response := &dto.TodayWorkoutResponse{
		SessionID:   workout.ID,
		SessionName: workout.SessionName,
		Week:        workout.Week,
		DayName:     workout.DayName,
		Exercises:   make([]dto.ExerciseDTO, 0, len(workout.Exercises)),
	}

	for _, exercise := range workout.Exercises {
		exerciseDTO := dto.ExerciseDTO{
			ID:          exercise.ID,
			Name:        exercise.Name,
			BadgeColor:  exercise.BadgeColor,
			RIR:         exercise.RIR,
			RestSeconds: exercise.RestSeconds,
			VideoLink:   exercise.Link,
			Sets:        make([]dto.SetDTO, 0, len(exercise.Sets)),
			Tips:        make([]dto.TipDTO, 0, len(exercise.Tips)),
			Steps:       make([]dto.StepDTO, 0, len(exercise.Steps)),
		}

		for _, set := range exercise.Sets {
			exerciseDTO.Sets = append(exerciseDTO.Sets, dto.SetDTO{
				ID:        set.ID,
				SetNumber: set.SetNumber,
				Reps:      set.Reps,
				Kg:        set.Weight,
				Completed: set.Completed,
			})
		}

		for _, tip := range exercise.Tips {
			exerciseDTO.Tips = append(exerciseDTO.Tips, dto.TipDTO{Text: tip.Text})
		}

		for _, step := range exercise.Steps {
			exerciseDTO.Steps = append(exerciseDTO.Steps, dto.StepDTO{Text: step.Text})
		}

		if len(exercise.Alternatives) > 0 {
			altDTOs := make([]dto.AlternativeExerciseDTO, 0, len(exercise.Alternatives))
			for _, alt := range exercise.Alternatives {
				altDTO := dto.AlternativeExerciseDTO{
					Name:        alt.Name,
					RIR:         alt.RIR,
					RestSeconds: alt.RestSeconds,
					VideoLink:   alt.Link,
					Sets:        make([]dto.SetDTO, 0, len(alt.Sets)),
				}
				for _, set := range alt.Sets {
					altDTO.Sets = append(altDTO.Sets, dto.SetDTO{
						ID:        set.ID,
						SetNumber: set.SetNumber,
						Reps:      set.Reps,
						Kg:        set.Weight,
						Completed: set.Completed,
					})
				}
				altDTOs = append(altDTOs, altDTO)
			}
			exerciseDTO.AlternativeExercises = altDTOs
		}

		response.Exercises = append(response.Exercises, exerciseDTO)
	}

	return response, nil
}
