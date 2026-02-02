package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPlanReader is a mock implementation of repository.PlanReader
type MockPlanReader struct {
	mock.Mock
}

func (m *MockPlanReader) GetByUserID(ctx context.Context, userID string) (*entity.Plan, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Plan), args.Error(1)
}

func (m *MockPlanReader) GetMesocycleStatus(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.MesocycleStatus), args.Error(1)
}

func TestCheckMesocycleStatus_CompleteMesocycle(t *testing.T) {
	mockRepo := new(MockPlanReader)
	uc := NewCheckMesocycleStatusUseCase(mockRepo)

	expectedStatus := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	expectedStatus.Goal = "Ganar masa muscular"
	expectedStatus.Level = "Intermedio"

	mockRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(expectedStatus, nil)

	result, err := uc.Execute(context.Background(), "user-123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsComplete)
	assert.True(t, result.CanRenew)
	assert.Equal(t, 100.0, result.CompletionRate)
	assert.Equal(t, 3, result.Week4Completed)
	assert.Equal(t, 3, result.Week4Total)
	assert.Equal(t, "Mesociclo completado. Puedes renovar tu rutina.", result.Message)
	mockRepo.AssertExpectations(t)
}

func TestCheckMesocycleStatus_IncompleteMesocycle(t *testing.T) {
	mockRepo := new(MockPlanReader)
	uc := NewCheckMesocycleStatusUseCase(mockRepo)

	// 2 out of 3 sessions completed
	expectedStatus := entity.NewMesocycleStatus("user-123", 1, 3, 2, "fb_3")
	mockRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(expectedStatus, nil)

	result, err := uc.Execute(context.Background(), "user-123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsComplete)
	assert.False(t, result.CanRenew)
	assert.Equal(t, "Te falta 1 sesion para completar el mesociclo.", result.Message)
	mockRepo.AssertExpectations(t)
}

func TestCheckMesocycleStatus_MultipleSessionsRemaining(t *testing.T) {
	mockRepo := new(MockPlanReader)
	uc := NewCheckMesocycleStatusUseCase(mockRepo)

	// 1 out of 4 sessions completed
	expectedStatus := entity.NewMesocycleStatus("user-456", 2, 4, 1, "ul_4")
	mockRepo.On("GetMesocycleStatus", mock.Anything, "user-456").Return(expectedStatus, nil)

	result, err := uc.Execute(context.Background(), "user-456")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsComplete)
	assert.Equal(t, "Te faltan 3 sesiones para completar el mesociclo.", result.Message)
	mockRepo.AssertExpectations(t)
}

func TestCheckMesocycleStatus_EmptyUserID(t *testing.T) {
	mockRepo := new(MockPlanReader)
	uc := NewCheckMesocycleStatusUseCase(mockRepo)

	result, err := uc.Execute(context.Background(), "")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user_id is required")
}

func TestCheckMesocycleStatus_NoPlanFound(t *testing.T) {
	mockRepo := new(MockPlanReader)
	uc := NewCheckMesocycleStatusUseCase(mockRepo)

	mockRepo.On("GetMesocycleStatus", mock.Anything, "user-nonexistent").Return(nil, nil)

	result, err := uc.Execute(context.Background(), "user-nonexistent")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no active plan found")
	mockRepo.AssertExpectations(t)
}

func TestCheckMesocycleStatus_RepositoryError(t *testing.T) {
	mockRepo := new(MockPlanReader)
	uc := NewCheckMesocycleStatusUseCase(mockRepo)

	mockRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(nil, errors.New("database error"))

	result, err := uc.Execute(context.Background(), "user-123")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
	mockRepo.AssertExpectations(t)
}

func TestCheckMesocycleStatus_WithLastRenewalDate(t *testing.T) {
	mockRepo := new(MockPlanReader)
	uc := NewCheckMesocycleStatusUseCase(mockRepo)

	renewalDate := time.Now().Add(-24 * time.Hour)
	expectedStatus := entity.NewMesocycleStatus("user-123", 2, 3, 3, "fb_3")
	expectedStatus.LastRenewalDate = &renewalDate

	mockRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(expectedStatus, nil)

	result, err := uc.Execute(context.Background(), "user-123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.LastRenewalDate)
	assert.Equal(t, 2, result.MesocycleNumber)
	mockRepo.AssertExpectations(t)
}

func TestBuildIncompleteMessage(t *testing.T) {
	tests := []struct {
		name      string
		remaining int
		expected  string
	}{
		{"single session", 1, "Te falta 1 sesion para completar el mesociclo."},
		{"two sessions", 2, "Te faltan 2 sesiones para completar el mesociclo."},
		{"three sessions", 3, "Te faltan 3 sesiones para completar el mesociclo."},
		{"five sessions", 5, "Te faltan 5 sesiones para completar el mesociclo."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildIncompleteMessage(tt.remaining)
			assert.Equal(t, tt.expected, result)
		})
	}
}
