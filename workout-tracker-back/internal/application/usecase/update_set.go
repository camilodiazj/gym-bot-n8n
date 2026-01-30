package usecase

import (
	"context"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// UpdateSetUseCase handles the business logic for updating a set
type UpdateSetUseCase struct {
	setRepo repository.SetWriter
}

// NewUpdateSetUseCase creates a new UpdateSetUseCase
func NewUpdateSetUseCase(setRepo repository.SetWriter) *UpdateSetUseCase {
	return &UpdateSetUseCase{
		setRepo: setRepo,
	}
}

// Execute updates a set's reps and/or weight
func (uc *UpdateSetUseCase) Execute(ctx context.Context, setID string, req *dto.UpdateSetRequest) (*dto.UpdateSetResponse, error) {
	if setID == "" {
		return nil, apperror.NewValidationError("set_id is required")
	}

	if req == nil || (req.Reps == nil && req.Kg == nil) {
		return nil, apperror.NewValidationError("at least one of reps or kg must be provided")
	}

	// Validate reps if provided
	if req.Reps != nil && *req.Reps < 0 {
		return nil, apperror.NewValidationError("reps must be a positive number")
	}

	err := uc.setRepo.Update(ctx, setID, req.Reps, req.Kg)
	if err != nil {
		return nil, err
	}

	return &dto.UpdateSetResponse{
		Success:   true,
		SetID:     setID,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}, nil
}
