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
	workoutRepo      repository.WorkoutWriter
	magicLinkInvalid repository.MagicLinkInvalidator
}

// NewCompleteWorkoutUseCase creates a new CompleteWorkoutUseCase
func NewCompleteWorkoutUseCase(workoutRepo repository.WorkoutWriter, magicLinkInvalid repository.MagicLinkInvalidator) *CompleteWorkoutUseCase {
	return &CompleteWorkoutUseCase{
		workoutRepo:      workoutRepo,
		magicLinkInvalid: magicLinkInvalid,
	}
}

// Execute marks a workout as complete and invalidates user's magic links
func (uc *CompleteWorkoutUseCase) Execute(ctx context.Context, workoutID string, userID string, req *dto.CompleteWorkoutRequest) (*dto.CompleteWorkoutResponse, error) {
	if workoutID == "" {
		return nil, apperror.NewValidationError("workout_id is required")
	}

	if userID == "" {
		return nil, apperror.NewValidationError("user_id is required")
	}

	// Mark the workout as complete
	err := uc.workoutRepo.MarkComplete(ctx, workoutID)
	if err != nil {
		return nil, err
	}

	// Invalidate all magic links for this user to prevent reuse
	if err := uc.magicLinkInvalid.InvalidateByUser(ctx, userID); err != nil {
		// Log the error but don't fail the request - workout is already marked complete
		// In production, this should be logged properly
	}

	return &dto.CompleteWorkoutResponse{
		Success:     true,
		WorkoutID:   workoutID,
		CompletedAt: time.Now().Format(time.RFC3339),
	}, nil
}
