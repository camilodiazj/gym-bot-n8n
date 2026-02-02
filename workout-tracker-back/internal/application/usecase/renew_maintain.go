package usecase

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// RenewMaintainUseCase handles renewing a mesocycle while keeping the same routine
type RenewMaintainUseCase struct {
	planRepo           repository.PlanRepository
	scheduleRepo       repository.ScheduleWriter
	workoutRenewalRepo repository.WorkoutRenewalWriter
}

// NewRenewMaintainUseCase creates a new RenewMaintainUseCase
func NewRenewMaintainUseCase(
	planRepo repository.PlanRepository,
	scheduleRepo repository.ScheduleWriter,
	workoutRenewalRepo repository.WorkoutRenewalWriter,
) *RenewMaintainUseCase {
	return &RenewMaintainUseCase{
		planRepo:           planRepo,
		scheduleRepo:       scheduleRepo,
		workoutRenewalRepo: workoutRenewalRepo,
	}
}

// Execute renews the mesocycle while maintaining the same exercises
func (uc *RenewMaintainUseCase) Execute(ctx context.Context, userID string, req *dto.RenewMaintainRequest) (*dto.RenewalResponse, error) {
	if userID == "" {
		return nil, apperror.NewValidationError("user_id is required")
	}

	// 1. Verify mesocycle is complete
	status, err := uc.planRepo.GetMesocycleStatus(ctx, userID)
	if err != nil {
		return nil, err
	}

	if status == nil {
		return nil, apperror.NewNotFoundError("no active plan found for user")
	}

	if !status.IsComplete {
		return nil, apperror.NewValidationError("mesocycle is not yet complete")
	}

	// 2. Clear the user's schedule
	if err := uc.scheduleRepo.ClearSchedule(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to clear schedule", err)
	}

	// 3. Reset workout weeks to 1 (keeps same exercises, restarts progression)
	if err := uc.workoutRenewalRepo.ResetWeeksToOne(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to reset workout weeks", err)
	}

	// 4. Increment mesocycle number
	if err := uc.planRepo.IncrementMesocycle(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to increment mesocycle", err)
	}

	return &dto.RenewalResponse{
		Success:            true,
		RenewalType:        "MANTENER_RUTINA",
		NewMesocycleNumber: status.MesocycleNumber + 1,
		Message:            "Tu rutina se ha renovado. Continuas con los mismos ejercicios con progresion de carga.",
		ScheduleCleared:    true,
	}, nil
}
