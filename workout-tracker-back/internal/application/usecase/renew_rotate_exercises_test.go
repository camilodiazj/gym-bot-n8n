package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorkoutRenewalRepository is a mock for repository.WorkoutRenewalRepository
type MockWorkoutRenewalRepository struct {
	mock.Mock
}

func (m *MockWorkoutRenewalRepository) GetUserWorkoutExercises(ctx context.Context, userID string) ([]*entity.WorkoutExercise, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.WorkoutExercise), args.Error(1)
}

func (m *MockWorkoutRenewalRepository) GetDistinctExerciseIDs(ctx context.Context, userID string) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockWorkoutRenewalRepository) DeleteUserWorkouts(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockWorkoutRenewalRepository) UpdateExerciseID(ctx context.Context, workoutID, newExerciseID string) error {
	args := m.Called(ctx, workoutID, newExerciseID)
	return args.Error(0)
}

func (m *MockWorkoutRenewalRepository) BatchUpdateExercises(ctx context.Context, rotations []*entity.ExerciseRotation) error {
	args := m.Called(ctx, rotations)
	return args.Error(0)
}

func (m *MockWorkoutRenewalRepository) ResetWeeksToOne(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockProfileReader is a mock for repository.ProfileReader
type MockProfileReader struct {
	mock.Mock
}

func (m *MockProfileReader) GetByUserID(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UserGymProfile), args.Error(1)
}

func (m *MockProfileReader) GetByWhatsAppID(ctx context.Context, whatsappID string) (*entity.UserGymProfile, error) {
	args := m.Called(ctx, whatsappID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UserGymProfile), args.Error(1)
}

// MockExerciseReader is a mock for repository.ExerciseReader
type MockExerciseReader struct {
	mock.Mock
}

func (m *MockExerciseReader) GetByID(ctx context.Context, exerciseID string) (*entity.ExerciseCatalog, error) {
	args := m.Called(ctx, exerciseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ExerciseCatalog), args.Error(1)
}

func (m *MockExerciseReader) FindAlternatives(ctx context.Context, pattern, role string, excludeIDs, excludeMuscles []string, limit int) ([]*entity.ExerciseCatalog, error) {
	args := m.Called(ctx, pattern, role, excludeIDs, excludeMuscles, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ExerciseCatalog), args.Error(1)
}

func (m *MockExerciseReader) FindByPatternAndRole(ctx context.Context, pattern, role string) ([]*entity.ExerciseCatalog, error) {
	args := m.Called(ctx, pattern, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ExerciseCatalog), args.Error(1)
}

func TestRenewRotateExercises_Success(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalRepository)
	mockProfileRepo := new(MockProfileReader)
	mockExerciseRepo := new(MockExerciseReader)

	prefProcessor := service.NewPreferenceProcessor()
	exerciseRotationSvc := service.NewExerciseRotationServiceWithSeed(mockExerciseRepo, prefProcessor, 42)

	uc := NewRenewRotateExercisesUseCase(
		mockPlanRepo,
		mockScheduleRepo,
		mockWorkoutRepo,
		mockProfileRepo,
		exerciseRotationSvc,
	)

	// Complete mesocycle
	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

	// User profile
	profile := &entity.UserGymProfile{
		WhatsAppID:       "573001234567",
		PriorityMuscles:  "Pecho, Espalda",
		DislikedExercises: "Pantorrillas",
	}

	// Current workouts
	workouts := []*entity.WorkoutExercise{
		{ID: "workout-1", UserID: "user-123", ExerciseID: "ex-1", DayName: "Dia 1"},
	}

	// Current exercise details
	currentExercise := &entity.ExerciseCatalog{
		ExerciseID:  "ex-1",
		SpanishName: "Press de Banca",
		Pattern:     "horizontal_push",
		Role:        "compound",
		MainMuscle:  "Chest",
	}

	// Alternative exercise
	alternativeExercise := &entity.ExerciseCatalog{
		ExerciseID:  "ex-2",
		SpanishName: "Press Inclinado",
		Pattern:     "horizontal_push",
		Role:        "compound",
		MainMuscle:  "Chest",
	}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(profile, nil)
	mockWorkoutRepo.On("GetUserWorkoutExercises", mock.Anything, "user-123").Return(workouts, nil)
	mockExerciseRepo.On("GetByID", mock.Anything, "ex-1").Return(currentExercise, nil)
	mockExerciseRepo.On("FindAlternatives", mock.Anything, "horizontal_push", "compound", mock.Anything, mock.Anything, mock.Anything).
		Return([]*entity.ExerciseCatalog{alternativeExercise}, nil)
	mockWorkoutRepo.On("BatchUpdateExercises", mock.Anything, mock.Anything).Return(nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("ResetWeeksToOne", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(nil)

	req := &dto.RenewRotateExercisesRequest{
		RotateCompounds: true,
		RotateIsolation: true,
		RotateCore:      true,
	}
	result, err := uc.Execute(context.Background(), "user-123", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "ROTAR_EJERCICIOS", result.RenewalType)
	assert.Equal(t, 2, result.NewMesocycleNumber)
	assert.True(t, result.ScheduleCleared)
	assert.GreaterOrEqual(t, result.ExercisesRotated, 0)
}

func TestRenewRotateExercises_NoProfile(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalRepository)
	mockProfileRepo := new(MockProfileReader)
	mockExerciseRepo := new(MockExerciseReader)

	prefProcessor := service.NewPreferenceProcessor()
	exerciseRotationSvc := service.NewExerciseRotationServiceWithSeed(mockExerciseRepo, prefProcessor, 42)

	uc := NewRenewRotateExercisesUseCase(
		mockPlanRepo,
		mockScheduleRepo,
		mockWorkoutRepo,
		mockProfileRepo,
		exerciseRotationSvc,
	)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	workouts := []*entity.WorkoutExercise{}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(nil, nil) // No profile
	mockWorkoutRepo.On("GetUserWorkoutExercises", mock.Anything, "user-123").Return(workouts, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("ResetWeeksToOne", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(nil)

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	// Should still work with empty profile
}

func TestRenewRotateExercises_EmptyUserID(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalRepository)
	mockProfileRepo := new(MockProfileReader)
	mockExerciseRepo := new(MockExerciseReader)

	prefProcessor := service.NewPreferenceProcessor()
	exerciseRotationSvc := service.NewExerciseRotationServiceWithSeed(mockExerciseRepo, prefProcessor, 42)

	uc := NewRenewRotateExercisesUseCase(
		mockPlanRepo,
		mockScheduleRepo,
		mockWorkoutRepo,
		mockProfileRepo,
		exerciseRotationSvc,
	)

	result, err := uc.Execute(context.Background(), "", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user_id is required")
}

func TestRenewRotateExercises_MesocycleNotComplete(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalRepository)
	mockProfileRepo := new(MockProfileReader)
	mockExerciseRepo := new(MockExerciseReader)

	prefProcessor := service.NewPreferenceProcessor()
	exerciseRotationSvc := service.NewExerciseRotationServiceWithSeed(mockExerciseRepo, prefProcessor, 42)

	uc := NewRenewRotateExercisesUseCase(
		mockPlanRepo,
		mockScheduleRepo,
		mockWorkoutRepo,
		mockProfileRepo,
		exerciseRotationSvc,
	)

	// Incomplete mesocycle
	status := entity.NewMesocycleStatus("user-123", 1, 3, 2, "fb_3")
	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "mesocycle is not yet complete")
}

func TestRenewRotateExercises_NoPlanFound(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalRepository)
	mockProfileRepo := new(MockProfileReader)
	mockExerciseRepo := new(MockExerciseReader)

	prefProcessor := service.NewPreferenceProcessor()
	exerciseRotationSvc := service.NewExerciseRotationServiceWithSeed(mockExerciseRepo, prefProcessor, 42)

	uc := NewRenewRotateExercisesUseCase(
		mockPlanRepo,
		mockScheduleRepo,
		mockWorkoutRepo,
		mockProfileRepo,
		exerciseRotationSvc,
	)

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(nil, nil)

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no active plan found")
}

func TestRenewRotateExercises_ProfileRepoError(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalRepository)
	mockProfileRepo := new(MockProfileReader)
	mockExerciseRepo := new(MockExerciseReader)

	prefProcessor := service.NewPreferenceProcessor()
	exerciseRotationSvc := service.NewExerciseRotationServiceWithSeed(mockExerciseRepo, prefProcessor, 42)

	uc := NewRenewRotateExercisesUseCase(
		mockPlanRepo,
		mockScheduleRepo,
		mockWorkoutRepo,
		mockProfileRepo,
		exerciseRotationSvc,
	)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(nil, errors.New("db error"))

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get user profile")
}

func TestRenewRotateExercises_GetWorkoutsError(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalRepository)
	mockProfileRepo := new(MockProfileReader)
	mockExerciseRepo := new(MockExerciseReader)

	prefProcessor := service.NewPreferenceProcessor()
	exerciseRotationSvc := service.NewExerciseRotationServiceWithSeed(mockExerciseRepo, prefProcessor, 42)

	uc := NewRenewRotateExercisesUseCase(
		mockPlanRepo,
		mockScheduleRepo,
		mockWorkoutRepo,
		mockProfileRepo,
		exerciseRotationSvc,
	)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	profile := &entity.UserGymProfile{}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(profile, nil)
	mockWorkoutRepo.On("GetUserWorkoutExercises", mock.Anything, "user-123").Return(nil, errors.New("db error"))

	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get workouts")
}

func TestRenewRotateExercises_DefaultRequest(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockScheduleRepo := new(MockScheduleWriter)
	mockWorkoutRepo := new(MockWorkoutRenewalRepository)
	mockProfileRepo := new(MockProfileReader)
	mockExerciseRepo := new(MockExerciseReader)

	prefProcessor := service.NewPreferenceProcessor()
	exerciseRotationSvc := service.NewExerciseRotationServiceWithSeed(mockExerciseRepo, prefProcessor, 42)

	uc := NewRenewRotateExercisesUseCase(
		mockPlanRepo,
		mockScheduleRepo,
		mockWorkoutRepo,
		mockProfileRepo,
		exerciseRotationSvc,
	)

	status := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")
	profile := &entity.UserGymProfile{}
	workouts := []*entity.WorkoutExercise{}

	mockPlanRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(status, nil)
	mockProfileRepo.On("GetByUserID", mock.Anything, "user-123").Return(profile, nil)
	mockWorkoutRepo.On("GetUserWorkoutExercises", mock.Anything, "user-123").Return(workouts, nil)
	mockScheduleRepo.On("ClearSchedule", mock.Anything, "user-123").Return(nil)
	mockWorkoutRepo.On("ResetWeeksToOne", mock.Anything, "user-123").Return(nil)
	mockPlanRepo.On("IncrementMesocycle", mock.Anything, "user-123").Return(nil)

	// Call with nil request - should use defaults
	result, err := uc.Execute(context.Background(), "user-123", nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestFilterWorkoutsByRole(t *testing.T) {
	workouts := []*entity.WorkoutExercise{
		{ID: "1", ExerciseID: "ex-1", Role: "compound"},
		{ID: "2", ExerciseID: "ex-2", Role: "isolation"},
		{ID: "3", ExerciseID: "ex-3", Role: "core"},
		{ID: "4", ExerciseID: "ex-4", Role: "compound"},
	}

	// All options true - should return all
	req := &dto.RenewRotateExercisesRequest{
		RotateCompounds: true,
		RotateIsolation: true,
		RotateCore:      true,
	}
	result := filterWorkoutsByRole(workouts, req)
	assert.Equal(t, 4, len(result))

	// Only compounds - should return 2
	req2 := &dto.RenewRotateExercisesRequest{
		RotateCompounds: true,
		RotateIsolation: false,
		RotateCore:      false,
	}
	result2 := filterWorkoutsByRole(workouts, req2)
	assert.Equal(t, 2, len(result2))
	assert.Equal(t, "compound", result2[0].Role)
	assert.Equal(t, "compound", result2[1].Role)

	// Compounds and isolation - should return 3
	req3 := &dto.RenewRotateExercisesRequest{
		RotateCompounds: true,
		RotateIsolation: true,
		RotateCore:      false,
	}
	result3 := filterWorkoutsByRole(workouts, req3)
	assert.Equal(t, 3, len(result3))

	// Only core - should return 1
	req4 := &dto.RenewRotateExercisesRequest{
		RotateCompounds: false,
		RotateIsolation: false,
		RotateCore:      true,
	}
	result4 := filterWorkoutsByRole(workouts, req4)
	assert.Equal(t, 1, len(result4))
	assert.Equal(t, "core", result4[0].Role)

	// None selected - should return empty
	req5 := &dto.RenewRotateExercisesRequest{
		RotateCompounds: false,
		RotateIsolation: false,
		RotateCore:      false,
	}
	result5 := filterWorkoutsByRole(workouts, req5)
	assert.Equal(t, 0, len(result5))
}
