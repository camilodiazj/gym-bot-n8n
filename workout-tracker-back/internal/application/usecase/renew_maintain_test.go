package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPlanRepository is a mock implementation of repository.PlanRepository
type MockPlanRepository struct {
	mock.Mock
}

func (m *MockPlanRepository) GetByUserID(ctx context.Context, userID string) (*entity.Plan, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Plan), args.Error(1)
}

func (m *MockPlanRepository) GetMesocycleStatus(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.MesocycleStatus), args.Error(1)
}

func (m *MockPlanRepository) IncrementMesocycle(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockPlanRepository) UpdateWeekSchedule(ctx context.Context, userID, weekSchedule string) error {
	args := m.Called(ctx, userID, weekSchedule)
	return args.Error(0)
}

// MockScheduleWriter is a mock implementation of repository.ScheduleWriter
type MockScheduleWriter struct {
	mock.Mock
}

func (m *MockScheduleWriter) ClearSchedule(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockScheduleWriter) ClearScheduleForWeeks(ctx context.Context, userID string, weeks []int) error {
	args := m.Called(ctx, userID, weeks)
	return args.Error(0)
}

// MockWorkoutRenewalWriter is a mock implementation of repository.WorkoutRenewalWriter
type MockWorkoutRenewalWriter struct {
	mock.Mock
}

func (m *MockWorkoutRenewalWriter) DeleteUserWorkouts(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockWorkoutRenewalWriter) UpdateExerciseID(ctx context.Context, workoutID, newExerciseID string) error {
	args := m.Called(ctx, workoutID, newExerciseID)
	return args.Error(0)
}

func (m *MockWorkoutRenewalWriter) BatchUpdateExercises(ctx context.Context, rotations []*entity.ExerciseRotation) error {
	args := m.Called(ctx, rotations)
	return args.Error(0)
}

func (m *MockWorkoutRenewalWriter) ResetWeeksToOne(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestRenewMaintain_Success(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewMaintainUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	// Mesocycle is complete (3/3)
	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("ResetWeeksToOne", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(nil)

	req := &dto.RenewMaintainRequest{}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "MANTENER_RUTINA", result.RenewalType)
	assert.Equal(t, 2, result.NewMesocycleNumber)
	assert.True(t, result.ScheduleCleared)
	assert.Contains(t, result.Message, "renovado")

	mockPlanRepo.AssertExpectations(t)
	mockScheduleRepo.AssertExpectations(t)
	mockWorkoutRepo.AssertExpectations(t)
}

func TestRenewMaintain_EmptyUserID(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewMaintainUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	result, err := uc.Execute(context.Background(), "", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user_id is required")
}

func TestRenewMaintain_NoPlanFound(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewMaintainUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(nil, nil)

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no active plan found")
	mockPlanRepo.AssertExpectations(t)
}

func TestRenewMaintain_MesocycleNotComplete(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewMaintainUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	// Mesocycle incomplete (2/3)
	status := entity.NewMesocycleStatus("user-123", 1, 3, 2, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "mesocycle is not yet complete")
	mockPlanRepo.AssertExpectations(t)
}

func TestRenewMaintain_ClearScheduleError(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewMaintainUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(errors.New("db error"))

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to clear schedule")
	mockPlanRepo.AssertExpectations(t)
	mockScheduleRepo.AssertExpectations(t)
}

func TestRenewMaintain_ResetWeeksError(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewMaintainUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("ResetWeeksToOne", mock.Anything, "user-123").Return(errors.New("db error"))

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to reset workout weeks")
}

func TestRenewMaintain_IncrementMesocycleError(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewMaintainUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("ResetWeeksToOne", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(errors.New("db error"))

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to increment mesocycle")
}
