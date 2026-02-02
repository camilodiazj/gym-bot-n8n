package usecase

import (
	"context"
	"strconv"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// CheckMesocycleStatusUseCase handles checking if a user's mesocycle is complete
type CheckMesocycleStatusUseCase struct {
	planRepo repository.PlanReader
}

// NewCheckMesocycleStatusUseCase creates a new CheckMesocycleStatusUseCase
func NewCheckMesocycleStatusUseCase(planRepo repository.PlanReader) *CheckMesocycleStatusUseCase {
	return &CheckMesocycleStatusUseCase{
		planRepo: planRepo,
	}
}

// Execute checks the mesocycle status for a user
func (uc *CheckMesocycleStatusUseCase) Execute(ctx context.Context, userID string) (*dto.MesocycleStatusResponse, error) {
	if userID == "" {
		return nil, apperror.NewValidationError("user_id is required")
	}

	status, err := uc.planRepo.GetMesocycleStatus(ctx, userID)
	if err != nil {
		return nil, err
	}

	if status == nil {
		return nil, apperror.NewNotFoundError("no active plan found for user")
	}

	// Determine message based on completion status
	var message string
	if status.IsComplete {
		message = "Mesociclo completado. Puedes renovar tu rutina."
	} else {
		remaining := status.Week4Total - status.Week4Completed
		message = buildIncompleteMessage(remaining)
	}

	return &dto.MesocycleStatusResponse{
		UserID:          status.UserID,
		MesocycleNumber: status.MesocycleNumber,
		DaysPerWeek:     status.DaysPerWeek,
		WeekSchedule:    status.WeekSchedule,
		Week4Completed:  status.Week4Completed,
		Week4Total:      status.Week4Total,
		IsComplete:      status.IsComplete,
		CompletionRate:  status.CompletionRate,
		LastRenewalDate: status.LastRenewalDate,
		Goal:            status.Goal,
		Level:           status.Level,
		CanRenew:        status.IsComplete,
		Message:         message,
	}, nil
}

// buildIncompleteMessage builds a message for incomplete mesocycles
func buildIncompleteMessage(remaining int) string {
	if remaining == 1 {
		return "Te falta 1 sesion para completar el mesociclo."
	}
	return "Te faltan " + strconv.Itoa(remaining) + " sesiones para completar el mesociclo."
}
