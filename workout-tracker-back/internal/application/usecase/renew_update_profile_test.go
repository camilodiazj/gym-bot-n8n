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

// MockProfileRepository is a mock implementation of repository.ProfileRepository
type MockProfileRepository struct {
	mock.Mock
}

func (m *MockProfileRepository) GetByUserID(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UserGymProfile), args.Error(1)
}

func (m *MockProfileRepository) GetByWhatsAppID(ctx context.Context, whatsappID string) (*entity.UserGymProfile, error) {
	args := m.Called(ctx, whatsappID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UserGymProfile), args.Error(1)
}

func (m *MockProfileRepository) UpdatePreferences(ctx context.Context, whatsappID string, priorityMuscles, healthStatus, sessionDuration string) error {
	args := m.Called(ctx, whatsappID, priorityMuscles, healthStatus, sessionDuration)
	return args.Error(0)
}

func TestRenewUpdateProfile_Success_WithProfileUpdates(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	// Mesocycle is complete (3/3)
	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	profile := &entity.UserGymProfile{
		WhatsAppID:     "573001234567@s.whatsapp.net",
		FullName:       "Test User",
		PriorityMuscles: "Pecho",
		HealthStatus:   "A",
	}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(profile, nil)
	mockProfileRepo.On("UpdatePreferences", mock.Anything, "573001234567@s.whatsapp.net", "Gluteo, pierna", "B", "45-60 min").Return(nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("DeleteUserWorkouts", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(nil)

	priorityMuscles := "Gluteo, pierna"
	healthStatus := "B"
	sessionDuration := "45-60 min"
	req := &dto.UpdateProfileRequest{
		PriorityMuscles: &priorityMuscles,
		HealthStatus:    &healthStatus,
		SessionDuration: &sessionDuration,
	}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 2, result.NewMesocycleNumber)
	assert.True(t, result.RequiresRegeneration)
	assert.True(t, result.ProfileUpdated)
	assert.True(t, result.WorkoutsDeleted)
	assert.True(t, result.ScheduleCleared)
	assert.Contains(t, result.Message, "actualizado")

	mockPlanRepo.AssertExpectations(t)
	mockScheduleRepo.AssertExpectations(t)
	mockWorkoutRepo.AssertExpectations(t)
	mockProfileRepo.AssertExpectations(t)
}

func TestRenewUpdateProfile_Success_NoProfileUpdates(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	profile := &entity.UserGymProfile{
		WhatsAppID: "573001234567@s.whatsapp.net",
		FullName:   "Test User",
	}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(profile, nil)
	// No UpdatePreferences call since no updates provided
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("DeleteUserWorkouts", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(nil)

	// Empty request - no profile updates
	req := &dto.UpdateProfileRequest{}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.False(t, result.ProfileUpdated) // No profile updates
	assert.True(t, result.WorkoutsDeleted)
	assert.True(t, result.ScheduleCleared)

	mockPlanRepo.AssertExpectations(t)
	mockScheduleRepo.AssertExpectations(t)
	mockWorkoutRepo.AssertExpectations(t)
	mockProfileRepo.AssertExpectations(t)
}

func TestRenewUpdateProfile_Success_PartialProfileUpdate(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	profile := &entity.UserGymProfile{
		WhatsAppID: "573001234567@s.whatsapp.net",
	}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(profile, nil)
	// Only priority muscles provided, other fields are empty strings
	mockProfileRepo.On("UpdatePreferences", mock.Anything, "573001234567@s.whatsapp.net", "Espalda", "", "").Return(nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("DeleteUserWorkouts", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(nil)

	priorityMuscles := "Espalda"
	req := &dto.UpdateProfileRequest{
		PriorityMuscles: &priorityMuscles,
	}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.ProfileUpdated)

	mockProfileRepo.AssertExpectations(t)
}

func TestRenewUpdateProfile_EmptyUserID(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	result, err := uc.Execute(context.Background(), "", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user_id is required")
}

func TestRenewUpdateProfile_NoPlanFound(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(nil, nil)

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no active plan found")
	mockPlanRepo.AssertExpectations(t)
}

func TestRenewUpdateProfile_MesocycleNotComplete(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	// Mesocycle incomplete (2/3)
	status := entity.NewMesocycleStatus("user-123", 1, 3, 2, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "mesocycle is not yet complete")
	mockPlanRepo.AssertExpectations(t)
}

func TestRenewUpdateProfile_ProfileNotFound(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(nil, nil)

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user profile not found")
	mockPlanRepo.AssertExpectations(t)
	mockProfileRepo.AssertExpectations(t)
}

func TestRenewUpdateProfile_UpdatePreferencesError(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	profile := &entity.UserGymProfile{
		WhatsAppID: "573001234567@s.whatsapp.net",
	}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(profile, nil)
	mockProfileRepo.On("UpdatePreferences", mock.Anything, "573001234567@s.whatsapp.net", "Gluteo", "", "").Return(errors.New("db error"))

	priorityMuscles := "Gluteo"
	req := &dto.UpdateProfileRequest{
		PriorityMuscles: &priorityMuscles,
	}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to update profile preferences")
	mockProfileRepo.AssertExpectations(t)
}

func TestRenewUpdateProfile_ClearScheduleError(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	profile := &entity.UserGymProfile{
		WhatsAppID: "573001234567@s.whatsapp.net",
	}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(profile, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(errors.New("db error"))

	req := &dto.UpdateProfileRequest{}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to clear schedule")
	mockScheduleRepo.AssertExpectations(t)
}

func TestRenewUpdateProfile_DeleteWorkoutsError(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	profile := &entity.UserGymProfile{
		WhatsAppID: "573001234567@s.whatsapp.net",
	}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(profile, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("DeleteUserWorkouts", mock.Anything, "user-123").Return(errors.New("db error"))

	req := &dto.UpdateProfileRequest{}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to delete workouts")
	mockWorkoutRepo.AssertExpectations(t)
}

func TestRenewUpdateProfile_IncrementMesocycleError(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	profile := &entity.UserGymProfile{
		WhatsAppID: "573001234567@s.whatsapp.net",
	}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(profile, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("DeleteUserWorkouts", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(errors.New("db error"))

	req := &dto.UpdateProfileRequest{}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to increment mesocycle")
	mockPlanRepo.AssertExpectations(t)
}

func TestRenewUpdateProfile_NilRequest(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)
	mockProfileRepo := new(MockProfileRepository)

	uc := NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	profile := &entity.UserGymProfile{
		WhatsAppID: "573001234567@s.whatsapp.net",
	}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(profile, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("DeleteUserWorkouts", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(nil)

	// Nil request should work - no profile updates
	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.False(t, result.ProfileUpdated)
}
