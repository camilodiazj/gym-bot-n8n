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

func TestRenewChangeDays_Success_SameFrequency(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	// User has 3 days/week and wants to keep 3 days
	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	// No DeleteUserWorkouts because frequency is the same
	mockPlanRepo.On("UpdateWeekSchedule", mock.Anything, "user-123", "fb_3").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(nil)

	req := &dto.RenewChangeDaysRequest{NewDaysPerWeek: 3}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "CAMBIAR_DIAS", result.RenewalType)
	assert.Equal(t, 2, result.NewMesocycleNumber)
	assert.Equal(t, 3, result.OldDaysPerWeek)
	assert.Equal(t, 3, result.NewDaysPerWeek)
	assert.False(t, result.RequiresRegeneration)
	assert.Equal(t, "fb_3", result.NewWeekSchedule)

	mockPlanRepo.AssertExpectations(t)
	mockScheduleRepo.AssertExpectations(t)
	mockWorkoutRepo.AssertExpectations(t)
}

func TestRenewChangeDays_Success_DifferentFrequency(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	// User has 3 days/week and wants to change to 4 days
	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("DeleteUserWorkouts", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("UpdateWeekSchedule", mock.Anything, "user-123", "ul_4").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(nil)

	req := &dto.RenewChangeDaysRequest{NewDaysPerWeek: 4}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 3, result.OldDaysPerWeek)
	assert.Equal(t, 4, result.NewDaysPerWeek)
	assert.True(t, result.RequiresRegeneration)
	assert.Equal(t, "ul_4", result.NewWeekSchedule)
	assert.Contains(t, result.Message, "4 dias")

	mockPlanRepo.AssertExpectations(t)
	mockScheduleRepo.AssertExpectations(t)
	mockWorkoutRepo.AssertExpectations(t)
}

func TestRenewChangeDays_AllScheduleMappings(t *testing.T) {
	tests := []struct {
		name             string
		days             int
		expectedSchedule string
	}{
		{"2 days", 2, "fb_2"},
		{"3 days", 3, "fb_3"},
		{"4 days", 4, "ul_4"},
		{"5 days", 5, "ppl_5"},
		{"6 days", 6, "ppl_6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPlanRepo := new(MockPlanRepository)
			mockScheduleRepo := new(MockScheduleWriter)
			mockWorkoutRepo := new(MockWorkoutRenewalWriter)

			uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

			status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

			mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
			mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
			if tt.days != 3 { // Frequency changes
				mockWorkoutRepo.On("DeleteUserWorkouts", mock.Anything, "user-123").Return(nil)
			}
			mockPlanRepo.On("UpdateWeekSchedule", mock.Anything, "user-123", tt.expectedSchedule).Return(nil)
			mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(nil)

			req := &dto.RenewChangeDaysRequest{NewDaysPerWeek: tt.days}
			result, err := uc.Execute(context.Background(), "user-123", req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSchedule, result.NewWeekSchedule)
		})
	}
}

func TestRenewChangeDays_EmptyUserID(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	req := &dto.RenewChangeDaysRequest{NewDaysPerWeek: 4}
	result, err := uc.Execute(context.Background(), "", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user_id is required")
}

func TestRenewChangeDays_NilRequest(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "request is required")
}

func TestRenewChangeDays_InvalidDays_TooLow(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	req := &dto.RenewChangeDaysRequest{NewDaysPerWeek: 1}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "between 2 and 6")
}

func TestRenewChangeDays_InvalidDays_TooHigh(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	req := &dto.RenewChangeDaysRequest{NewDaysPerWeek: 7}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "between 2 and 6")
}

func TestRenewChangeDays_MesocycleNotComplete(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	// Mesocycle incomplete (2/3)
	status := entity.NewMesocycleStatus("user-123", 1, 3, 2, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)

	req := &dto.RenewChangeDaysRequest{NewDaysPerWeek: 4}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "mesocycle is not yet complete")
}

func TestRenewChangeDays_NoPlanFound(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(nil, nil)

	req := &dto.RenewChangeDaysRequest{NewDaysPerWeek: 4}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no active plan found")
}

func TestRenewChangeDays_DeleteWorkoutsError(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("DeleteUserWorkouts", mock.Anything, "user-123").Return(errors.New("db error"))

	req := &dto.RenewChangeDaysRequest{NewDaysPerWeek: 4}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to delete workouts")
}

func TestRenewChangeDays_UpdateWeekScheduleError(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalWriter)

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("DeleteUserWorkouts", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("UpdateWeekSchedule", mock.Anything, "user-123", "ul_4").Return(errors.New("db error"))

	req := &dto.RenewChangeDaysRequest{NewDaysPerWeek: 4}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to update week schedule")
}

func TestWeekScheduleMapping(t *testing.T) {
	// Verify the mapping is correct
	assert.Equal(t, "fb_2", WeekScheduleMapping[2])
	assert.Equal(t, "fb_3", WeekScheduleMapping[3])
	assert.Equal(t, "ul_4", WeekScheduleMapping[4])
	assert.Equal(t, "ppl_5", WeekScheduleMapping[5])
	assert.Equal(t, "ppl_6", WeekScheduleMapping[6])

	// Verify invalid values don't exist
	_, exists := WeekScheduleMapping[0]
	assert.False(t, exists)
	_, exists = WeekScheduleMapping[1]
	assert.False(t, exists)
	_, exists = WeekScheduleMapping[7]
	assert.False(t, exists)
}
