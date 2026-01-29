package usecase

import (
	"context"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// CompleteWorkoutUseCase handles the business logic for completing a workout
type CompleteWorkoutUseCase struct {
	workoutRepo repository.WorkoutWriter
}

// NewCompleteWorkoutUseCase creates a new CompleteWorkoutUseCase
func NewCompleteWorkoutUseCase(workoutRepo repository.WorkoutWriter) *CompleteWorkoutUseCase {
	return &CompleteWorkoutUseCase{
		workoutRepo: workoutRepo,
	}
}

// Execute marks a workout as complete
func (uc *CompleteWorkoutUseCase) Execute(ctx context.Context, workoutID string, req *dto.CompleteWorkoutRequest) (*dto.CompleteWorkoutResponse, error) {
	if workoutID == "" {
		return nil, apperror.NewValidationError("workout_id is required")
	}

	err := uc.workoutRepo.MarkComplete(ctx, workoutID)
	if err != nil {
		return nil, err
	}

	return &dto.CompleteWorkoutResponse{
		Success:     true,
		WorkoutID:   workoutID,
		CompletedAt: time.Now().Format(time.RFC3339),
	}, nil
}
