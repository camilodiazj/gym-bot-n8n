package usecase

import (
	"context"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// MarkSetCompleteUseCase handles the business logic for marking a set as complete
type MarkSetCompleteUseCase struct {
	setRepo repository.SetWriter
}

// NewMarkSetCompleteUseCase creates a new MarkSetCompleteUseCase
func NewMarkSetCompleteUseCase(setRepo repository.SetWriter) *MarkSetCompleteUseCase {
	return &MarkSetCompleteUseCase{
		setRepo: setRepo,
	}
}

// Execute marks a set as complete
func (uc *MarkSetCompleteUseCase) Execute(ctx context.Context, setID string) (*dto.MarkSetCompleteResponse, error) {
	if setID == "" {
		return nil, apperror.NewValidationError("set_id is required")
	}

	err := uc.setRepo.MarkComplete(ctx, setID)
	if err != nil {
		return nil, err
	}

	return &dto.MarkSetCompleteResponse{
		Success:     true,
		SetID:       setID,
		CompletedAt: time.Now().Format(time.RFC3339),
	}, nil
}
