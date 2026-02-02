# Mesocycle Renewal - Test Plan

## Overview

This document provides a comprehensive test plan for the mesocycle renewal feature, covering backend unit tests (Go), E2E test cases (n8n), fixture data (SQL), integration tests (cURL), and manual verification checklists.

**Reserved Phone Range**: `570000000010-570000000019` for renewal test users.

---

## 1. Backend Unit Tests (Go)

All tests follow the existing hexagonal architecture patterns in `workout-tracker-back/`.

### 1.1 Mock Interfaces

```go
// File: workout-tracker-back/internal/application/usecase/renewal_mocks_test.go

package usecase

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// MockPlanRepository implements domain.PlanRepository for tests
type MockPlanRepository struct {
	GetByUserIDFunc      func(ctx context.Context, userID string) (*entity.Plan, error)
	UpdateMesocycleFunc  func(ctx context.Context, userID string, mesocycle int) error
	UpdateWeekScheduleFunc func(ctx context.Context, userID string, schedule string) error
}

func (m *MockPlanRepository) GetByUserID(ctx context.Context, userID string) (*entity.Plan, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockPlanRepository) UpdateMesocycle(ctx context.Context, userID string, mesocycle int) error {
	if m.UpdateMesocycleFunc != nil {
		return m.UpdateMesocycleFunc(ctx, userID, mesocycle)
	}
	return nil
}

func (m *MockPlanRepository) UpdateWeekSchedule(ctx context.Context, userID string, schedule string) error {
	if m.UpdateWeekScheduleFunc != nil {
		return m.UpdateWeekScheduleFunc(ctx, userID, schedule)
	}
	return nil
}

// MockScheduleRepository implements domain.ScheduleRepository for tests
type MockScheduleRepository struct {
	GetWeek4CompletionFunc func(ctx context.Context, userID string) (int, int, error)
	ClearScheduleFunc      func(ctx context.Context, userID string) error
}

func (m *MockScheduleRepository) GetWeek4Completion(ctx context.Context, userID string) (int, int, error) {
	if m.GetWeek4CompletionFunc != nil {
		return m.GetWeek4CompletionFunc(ctx, userID)
	}
	return 0, 0, nil
}

func (m *MockScheduleRepository) ClearSchedule(ctx context.Context, userID string) error {
	if m.ClearScheduleFunc != nil {
		return m.ClearScheduleFunc(ctx, userID)
	}
	return nil
}

// MockExerciseRepository implements domain.ExerciseRepository for tests
type MockExerciseRepository struct {
	GetCurrentExercisesFunc  func(ctx context.Context, userID string) ([]entity.UserExercise, error)
	FindAlternativeFunc      func(ctx context.Context, pattern, role string, excludeIDs []string, dislikedMuscles []string) (*entity.Exercise, error)
	GetExercisesByPatternFunc func(ctx context.Context, pattern string) ([]entity.Exercise, error)
}

func (m *MockExerciseRepository) GetCurrentExercises(ctx context.Context, userID string) ([]entity.UserExercise, error) {
	if m.GetCurrentExercisesFunc != nil {
		return m.GetCurrentExercisesFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockExerciseRepository) FindAlternative(ctx context.Context, pattern, role string, excludeIDs []string, dislikedMuscles []string) (*entity.Exercise, error) {
	if m.FindAlternativeFunc != nil {
		return m.FindAlternativeFunc(ctx, pattern, role, excludeIDs, dislikedMuscles)
	}
	return nil, nil
}

func (m *MockExerciseRepository) GetExercisesByPattern(ctx context.Context, pattern string) ([]entity.Exercise, error) {
	if m.GetExercisesByPatternFunc != nil {
		return m.GetExercisesByPatternFunc(ctx, pattern)
	}
	return nil, nil
}

// MockWorkoutRenewalRepository implements domain.WorkoutRenewalRepository for tests
type MockWorkoutRenewalRepository struct {
	DeleteAllWorkoutsFunc    func(ctx context.Context, userID string) error
	CopyWorkoutsToWeek1Func  func(ctx context.Context, userID string) error
	RotateExerciseFunc       func(ctx context.Context, userID string, oldExerciseID, newExerciseID string) error
}

func (m *MockWorkoutRenewalRepository) DeleteAllWorkouts(ctx context.Context, userID string) error {
	if m.DeleteAllWorkoutsFunc != nil {
		return m.DeleteAllWorkoutsFunc(ctx, userID)
	}
	return nil
}

func (m *MockWorkoutRenewalRepository) CopyWorkoutsToWeek1(ctx context.Context, userID string) error {
	if m.CopyWorkoutsToWeek1Func != nil {
		return m.CopyWorkoutsToWeek1Func(ctx, userID)
	}
	return nil
}

func (m *MockWorkoutRenewalRepository) RotateExercise(ctx context.Context, userID string, oldExerciseID, newExerciseID string) error {
	if m.RotateExerciseFunc != nil {
		return m.RotateExerciseFunc(ctx, userID, oldExerciseID, newExerciseID)
	}
	return nil
}

// MockUserProfileRepository implements domain.UserProfileRepository for tests
type MockUserProfileRepository struct {
	GetByUserIDFunc func(ctx context.Context, userID string) (*entity.UserGymProfile, error)
}

func (m *MockUserProfileRepository) GetByUserID(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return nil, nil
}
```

---

### 1.2 TestPreferenceProcessor_MapMuscles

```go
// File: workout-tracker-back/internal/domain/service/preference_processor_test.go

package service

import (
	"reflect"
	"testing"
)

func TestPreferenceProcessor_MapMuscles_SingleMuscle(t *testing.T) {
	pp := NewPreferenceProcessor()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Gluteo maps to Glutes",
			input:    "Glúteo",
			expected: []string{"Glutes"},
		},
		{
			name:     "Pierna maps to multiple muscles",
			input:    "Pierna",
			expected: []string{"Quads", "Hamstrings", "Calves"},
		},
		{
			name:     "Pecho maps to Chest",
			input:    "Pecho",
			expected: []string{"Chest"},
		},
		{
			name:     "Espalda maps to Back and Lats",
			input:    "Espalda",
			expected: []string{"Back", "Lats"},
		},
		{
			name:     "Pantorrillas maps to Calves",
			input:    "Pantorrillas",
			expected: []string{"Calves"},
		},
		{
			name:     "Brazos maps to arms",
			input:    "Brazos",
			expected: []string{"Biceps", "Triceps", "Forearms"},
		},
		{
			name:     "Hombros maps to Shoulders",
			input:    "Hombros",
			expected: []string{"Shoulders"},
		},
		{
			name:     "Core/Abdomen maps to Core",
			input:    "Abdomen",
			expected: []string{"Core", "Abs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pp.MapMuscleSpanishToEnglish(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("MapMuscleSpanishToEnglish(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPreferenceProcessor_MapMuscles_CommaSeparatedList(t *testing.T) {
	pp := NewPreferenceProcessor()

	input := "Glúteo, Pierna, Pecho"
	result := pp.MapMusclesListToEnglish(input)

	expected := []string{"Glutes", "Quads", "Hamstrings", "Calves", "Chest"}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("MapMusclesListToEnglish(%q) = %v, want %v", input, result, expected)
	}
}

func TestPreferenceProcessor_MapMuscles_EmptyInput(t *testing.T) {
	pp := NewPreferenceProcessor()

	result := pp.MapMusclesListToEnglish("")

	if result == nil {
		t.Error("Expected empty slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %v", result)
	}
}

func TestPreferenceProcessor_MapMuscles_UnknownMuscle(t *testing.T) {
	pp := NewPreferenceProcessor()

	result := pp.MapMuscleSpanishToEnglish("MúsculoInventado")

	// Should return empty slice for unknown muscles
	if len(result) != 0 {
		t.Errorf("Expected empty slice for unknown muscle, got %v", result)
	}
}

func TestPreferenceProcessor_MapMuscles_CaseInsensitive(t *testing.T) {
	pp := NewPreferenceProcessor()

	tests := []struct {
		input    string
		expected []string
	}{
		{"GLUTEO", []string{"Glutes"}},
		{"gluteo", []string{"Glutes"}},
		{"Gluteo", []string{"Glutes"}},
		{"glúteo", []string{"Glutes"}},
	}

	for _, tt := range tests {
		result := pp.MapMuscleSpanishToEnglish(tt.input)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("MapMuscleSpanishToEnglish(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
```

---

### 1.3 TestPreferenceProcessor_ExperienceTier

```go
func TestPreferenceProcessor_ExperienceTier(t *testing.T) {
	pp := NewPreferenceProcessor()

	tests := []struct {
		name            string
		trainingExp     string
		expectedTier    string
		expectedYears   int
	}{
		{
			name:         "Beginner - less than 6 months",
			trainingExp:  "Menos de 6 meses",
			expectedTier: "beginner",
			expectedYears: 0,
		},
		{
			name:         "Beginner - 6 months to 1 year",
			trainingExp:  "6 meses a 1 año",
			expectedTier: "beginner",
			expectedYears: 1,
		},
		{
			name:         "Intermediate - 1 to 2 years",
			trainingExp:  "1 a 2 años",
			expectedTier: "intermediate",
			expectedYears: 2,
		},
		{
			name:         "Intermediate - 2 to 3 years",
			trainingExp:  "2 a 3 años",
			expectedTier: "intermediate",
			expectedYears: 3,
		},
		{
			name:         "Advanced - more than 3 years",
			trainingExp:  "Más de 3 años",
			expectedTier: "advanced",
			expectedYears: 4,
		},
		{
			name:         "Empty input defaults to beginner",
			trainingExp:  "",
			expectedTier: "beginner",
			expectedYears: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier, years := pp.GetExperienceTier(tt.trainingExp)
			if tier != tt.expectedTier {
				t.Errorf("GetExperienceTier(%q) tier = %q, want %q", tt.trainingExp, tier, tt.expectedTier)
			}
			if years != tt.expectedYears {
				t.Errorf("GetExperienceTier(%q) years = %d, want %d", tt.trainingExp, years, tt.expectedYears)
			}
		})
	}
}
```

---

### 1.4 TestPreferenceProcessor_VolumeModifier

```go
func TestPreferenceProcessor_VolumeModifier(t *testing.T) {
	pp := NewPreferenceProcessor()

	tests := []struct {
		name            string
		sessionDuration string
		expectedMod     float64
	}{
		{
			name:            "Short session - 30-45 min",
			sessionDuration: "30-45 min",
			expectedMod:     0.70, // Reduce volume by 30%
		},
		{
			name:            "Medium session - 45-60 min",
			sessionDuration: "45-60 min",
			expectedMod:     0.85, // Reduce volume by 15%
		},
		{
			name:            "Standard session - 60-75 min",
			sessionDuration: "60-75 min",
			expectedMod:     1.00, // Full volume
		},
		{
			name:            "Long session - 75-90 min",
			sessionDuration: "75-90 min",
			expectedMod:     1.10, // Increase volume by 10%
		},
		{
			name:            "Extended session - 90+ min",
			sessionDuration: "Más de 90 min",
			expectedMod:     1.20, // Increase volume by 20%
		},
		{
			name:            "Empty input defaults to standard",
			sessionDuration: "",
			expectedMod:     1.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mod := pp.GetVolumeModifier(tt.sessionDuration)
			if mod != tt.expectedMod {
				t.Errorf("GetVolumeModifier(%q) = %v, want %v", tt.sessionDuration, mod, tt.expectedMod)
			}
		})
	}
}
```

---

### 1.5 TestPreferenceProcessor_HealthRestrictions

```go
func TestPreferenceProcessor_HealthRestrictions(t *testing.T) {
	pp := NewPreferenceProcessor()

	tests := []struct {
		name         string
		healthStatus string
		expected     entity.HealthRestrictions
	}{
		{
			name:         "Status A - No restrictions",
			healthStatus: "A",
			expected: entity.HealthRestrictions{
				AvoidUpperBodyOverhead: false,
				AvoidLowerBodyImpact:   false,
				AvoidAxialLoading:      false,
				PreferMachines:         false,
			},
		},
		{
			name:         "Status B - Lower body issues",
			healthStatus: "B",
			expected: entity.HealthRestrictions{
				AvoidUpperBodyOverhead: false,
				AvoidLowerBodyImpact:   true,
				AvoidAxialLoading:      false,
				PreferMachines:         false,
			},
		},
		{
			name:         "Status C - Upper body issues",
			healthStatus: "C",
			expected: entity.HealthRestrictions{
				AvoidUpperBodyOverhead: true,
				AvoidLowerBodyImpact:   false,
				AvoidAxialLoading:      false,
				PreferMachines:         false,
			},
		},
		{
			name:         "Status D - Spine issues",
			healthStatus: "D",
			expected: entity.HealthRestrictions{
				AvoidUpperBodyOverhead: false,
				AvoidLowerBodyImpact:   false,
				AvoidAxialLoading:      true,
				PreferMachines:         false,
			},
		},
		{
			name:         "Status E - Special condition",
			healthStatus: "E",
			expected: entity.HealthRestrictions{
				AvoidUpperBodyOverhead: true,
				AvoidLowerBodyImpact:   true,
				AvoidAxialLoading:      true,
				PreferMachines:         true,
			},
		},
		{
			name:         "Unknown status defaults to no restrictions",
			healthStatus: "X",
			expected: entity.HealthRestrictions{
				AvoidUpperBodyOverhead: false,
				AvoidLowerBodyImpact:   false,
				AvoidAxialLoading:      false,
				PreferMachines:         false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pp.GetHealthRestrictions(tt.healthStatus)
			if result != tt.expected {
				t.Errorf("GetHealthRestrictions(%q) = %+v, want %+v", tt.healthStatus, result, tt.expected)
			}
		})
	}
}
```

---

### 1.6 TestExerciseRotationService_FindAlternatives

```go
// File: workout-tracker-back/internal/domain/service/exercise_rotation_service_test.go

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

func TestExerciseRotationService_FindAlternatives_Success(t *testing.T) {
	mockExerciseRepo := &MockExerciseRepository{
		FindAlternativeFunc: func(ctx context.Context, pattern, role string, excludeIDs []string, dislikedMuscles []string) (*entity.Exercise, error) {
			return &entity.Exercise{
				ID:          "new-exercise-123",
				SpanishName: "Press Inclinado con Barra",
				Pattern:     "push",
				Role:        "compound",
				MainMuscle:  "Pecho",
			}, nil
		},
	}

	svc := NewExerciseRotationService(mockExerciseRepo)

	currentExercise := entity.UserExercise{
		ExerciseID:  "old-exercise-456",
		Pattern:     "push",
		Role:        "compound",
		MainMuscle:  "Pecho",
	}
	dislikedMuscles := []string{"Calves"}

	result, err := svc.FindAlternative(context.Background(), currentExercise, []string{"old-exercise-456"}, dislikedMuscles)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.ID == currentExercise.ExerciseID {
		t.Error("Expected different exercise ID")
	}

	if result.Pattern != currentExercise.Pattern {
		t.Errorf("Expected same pattern %q, got %q", currentExercise.Pattern, result.Pattern)
	}
}

func TestExerciseRotationService_FindAlternatives_NoAlternativeFound(t *testing.T) {
	mockExerciseRepo := &MockExerciseRepository{
		FindAlternativeFunc: func(ctx context.Context, pattern, role string, excludeIDs []string, dislikedMuscles []string) (*entity.Exercise, error) {
			return nil, nil // No alternative found
		},
	}

	svc := NewExerciseRotationService(mockExerciseRepo)

	currentExercise := entity.UserExercise{
		ExerciseID: "old-exercise-456",
		Pattern:    "push",
		Role:       "compound",
	}

	result, err := svc.FindAlternative(context.Background(), currentExercise, []string{"old-exercise-456"}, nil)

	if err == nil {
		t.Fatal("Expected error when no alternative found")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestExerciseRotationService_FindAlternatives_RespectsDislikedMuscles(t *testing.T) {
	var capturedDisliked []string

	mockExerciseRepo := &MockExerciseRepository{
		FindAlternativeFunc: func(ctx context.Context, pattern, role string, excludeIDs []string, dislikedMuscles []string) (*entity.Exercise, error) {
			capturedDisliked = dislikedMuscles
			return &entity.Exercise{ID: "new-123", Pattern: pattern, Role: role}, nil
		},
	}

	svc := NewExerciseRotationService(mockExerciseRepo)

	dislikedMuscles := []string{"Calves", "Shoulders"}
	currentExercise := entity.UserExercise{ExerciseID: "old-456", Pattern: "push", Role: "compound"}

	_, _ = svc.FindAlternative(context.Background(), currentExercise, []string{"old-456"}, dislikedMuscles)

	if len(capturedDisliked) != 2 {
		t.Errorf("Expected 2 disliked muscles passed, got %d", len(capturedDisliked))
	}
}

func TestExerciseRotationService_FindAlternatives_RepoError(t *testing.T) {
	mockExerciseRepo := &MockExerciseRepository{
		FindAlternativeFunc: func(ctx context.Context, pattern, role string, excludeIDs []string, dislikedMuscles []string) (*entity.Exercise, error) {
			return nil, errors.New("database error")
		},
	}

	svc := NewExerciseRotationService(mockExerciseRepo)

	currentExercise := entity.UserExercise{ExerciseID: "old-456", Pattern: "push", Role: "compound"}

	result, err := svc.FindAlternative(context.Background(), currentExercise, nil, nil)

	if err == nil {
		t.Fatal("Expected error from repository")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestExerciseRotationService_RotateByFrequency(t *testing.T) {
	// Test that compounds rotate less frequently than isolation
	tests := []struct {
		name           string
		role           string
		mesocycleNum   int
		shouldRotate   bool
	}{
		{
			name:         "Compound at mesocycle 2 - no rotation",
			role:         "compound",
			mesocycleNum: 2,
			shouldRotate: false,
		},
		{
			name:         "Compound at mesocycle 3 - may rotate",
			role:         "compound",
			mesocycleNum: 3,
			shouldRotate: true,
		},
		{
			name:         "Isolation at mesocycle 2 - may rotate",
			role:         "isolation",
			mesocycleNum: 2,
			shouldRotate: true,
		},
		{
			name:         "Core at mesocycle 2 - may rotate",
			role:         "core",
			mesocycleNum: 2,
			shouldRotate: true,
		},
	}

	svc := NewExerciseRotationService(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.ShouldRotate(tt.role, tt.mesocycleNum)
			if result != tt.shouldRotate {
				t.Errorf("ShouldRotate(%q, %d) = %v, want %v", tt.role, tt.mesocycleNum, result, tt.shouldRotate)
			}
		})
	}
}
```

---

### 1.7 TestCheckMesocycleStatusUseCase

```go
// File: workout-tracker-back/internal/application/usecase/check_mesocycle_status_test.go

package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

func TestCheckMesocycleStatusUseCase_Execute_Complete(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{
				PlanID:          "plan-123",
				UserID:          userID,
				MesocycleNumber: 1,
				DaysPerWeek:     3,
				WeekSchedule:    "fb_3",
			}, nil
		},
	}

	mockScheduleRepo := &MockScheduleRepository{
		GetWeek4CompletionFunc: func(ctx context.Context, userID string) (int, int, error) {
			return 3, 3, nil // 3 completed out of 3 required
		},
	}

	uc := NewCheckMesocycleStatusUseCase(mockPlanRepo, mockScheduleRepo)
	result, err := uc.Execute(context.Background(), "user-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsComplete {
		t.Error("Expected IsComplete to be true")
	}

	if result.MesocycleNumber != 1 {
		t.Errorf("Expected MesocycleNumber 1, got %d", result.MesocycleNumber)
	}

	if result.CompletedSessions != 3 {
		t.Errorf("Expected CompletedSessions 3, got %d", result.CompletedSessions)
	}

	if result.RequiredSessions != 3 {
		t.Errorf("Expected RequiredSessions 3, got %d", result.RequiredSessions)
	}
}

func TestCheckMesocycleStatusUseCase_Execute_Incomplete(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{
				PlanID:          "plan-123",
				UserID:          userID,
				MesocycleNumber: 1,
				DaysPerWeek:     4,
				WeekSchedule:    "ul_4",
			}, nil
		},
	}

	mockScheduleRepo := &MockScheduleRepository{
		GetWeek4CompletionFunc: func(ctx context.Context, userID string) (int, int, error) {
			return 2, 4, nil // 2 completed out of 4 required
		},
	}

	uc := NewCheckMesocycleStatusUseCase(mockPlanRepo, mockScheduleRepo)
	result, err := uc.Execute(context.Background(), "user-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsComplete {
		t.Error("Expected IsComplete to be false")
	}

	if result.CompletedSessions != 2 {
		t.Errorf("Expected CompletedSessions 2, got %d", result.CompletedSessions)
	}
}

func TestCheckMesocycleStatusUseCase_Execute_EmptyUserID(t *testing.T) {
	uc := NewCheckMesocycleStatusUseCase(nil, nil)
	result, err := uc.Execute(context.Background(), "")

	if err == nil {
		t.Fatal("Expected error for empty user_id")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestCheckMesocycleStatusUseCase_Execute_NoPlanFound(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return nil, nil
		},
	}

	uc := NewCheckMesocycleStatusUseCase(mockPlanRepo, nil)
	result, err := uc.Execute(context.Background(), "user-123")

	if err == nil {
		t.Fatal("Expected error when no plan found")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestCheckMesocycleStatusUseCase_Execute_PlanRepoError(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return nil, errors.New("database error")
		},
	}

	uc := NewCheckMesocycleStatusUseCase(mockPlanRepo, nil)
	result, err := uc.Execute(context.Background(), "user-123")

	if err == nil {
		t.Fatal("Expected error from repository")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestCheckMesocycleStatusUseCase_Execute_Week2NotComplete(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{
				PlanID:          "plan-123",
				UserID:          userID,
				MesocycleNumber: 1,
				DaysPerWeek:     3,
			}, nil
		},
	}

	// User is still at week 2 (no week 4 sessions)
	mockScheduleRepo := &MockScheduleRepository{
		GetWeek4CompletionFunc: func(ctx context.Context, userID string) (int, int, error) {
			return 0, 3, nil // 0 completed week 4 sessions
		},
	}

	uc := NewCheckMesocycleStatusUseCase(mockPlanRepo, mockScheduleRepo)
	result, err := uc.Execute(context.Background(), "user-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.IsComplete {
		t.Error("Expected IsComplete to be false for user at week 2")
	}
}
```

---

### 1.8 TestRenewMaintainUseCase

```go
// File: workout-tracker-back/internal/application/usecase/renew_maintain_test.go

package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

func TestRenewMaintainUseCase_Execute_Success(t *testing.T) {
	var capturedMesocycle int
	var clearedSchedule bool

	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{
				PlanID:          "plan-123",
				UserID:          userID,
				MesocycleNumber: 1,
			}, nil
		},
		UpdateMesocycleFunc: func(ctx context.Context, userID string, mesocycle int) error {
			capturedMesocycle = mesocycle
			return nil
		},
	}

	mockScheduleRepo := &MockScheduleRepository{
		ClearScheduleFunc: func(ctx context.Context, userID string) error {
			clearedSchedule = true
			return nil
		},
	}

	mockWorkoutRepo := &MockWorkoutRenewalRepository{
		CopyWorkoutsToWeek1Func: func(ctx context.Context, userID string) error {
			return nil
		},
	}

	uc := NewRenewMaintainUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)
	result, err := uc.Execute(context.Background(), "user-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if capturedMesocycle != 2 {
		t.Errorf("Expected mesocycle to be incremented to 2, got %d", capturedMesocycle)
	}

	if !clearedSchedule {
		t.Error("Expected schedule to be cleared")
	}

	if result.NewMesocycleNumber != 2 {
		t.Errorf("Expected NewMesocycleNumber 2, got %d", result.NewMesocycleNumber)
	}

	if result.RenewalType != "maintain" {
		t.Errorf("Expected RenewalType 'maintain', got %q", result.RenewalType)
	}
}

func TestRenewMaintainUseCase_Execute_EmptyUserID(t *testing.T) {
	uc := NewRenewMaintainUseCase(nil, nil, nil)
	result, err := uc.Execute(context.Background(), "")

	if err == nil {
		t.Fatal("Expected error for empty user_id")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestRenewMaintainUseCase_Execute_PlanNotFound(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return nil, nil
		},
	}

	uc := NewRenewMaintainUseCase(mockPlanRepo, nil, nil)
	result, err := uc.Execute(context.Background(), "user-123")

	if err == nil {
		t.Fatal("Expected error when plan not found")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestRenewMaintainUseCase_Execute_UpdateMesocycleError(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{PlanID: "plan-123", MesocycleNumber: 1}, nil
		},
		UpdateMesocycleFunc: func(ctx context.Context, userID string, mesocycle int) error {
			return errors.New("database error")
		},
	}

	uc := NewRenewMaintainUseCase(mockPlanRepo, nil, nil)
	result, err := uc.Execute(context.Background(), "user-123")

	if err == nil {
		t.Fatal("Expected error from update")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestRenewMaintainUseCase_Execute_SetsLastRenewalDate(t *testing.T) {
	var renewalDateSet time.Time

	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{PlanID: "plan-123", MesocycleNumber: 1}, nil
		},
		UpdateMesocycleFunc: func(ctx context.Context, userID string, mesocycle int) error {
			renewalDateSet = time.Now()
			return nil
		},
	}

	mockScheduleRepo := &MockScheduleRepository{
		ClearScheduleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}

	mockWorkoutRepo := &MockWorkoutRenewalRepository{
		CopyWorkoutsToWeek1Func: func(ctx context.Context, userID string) error {
			return nil
		},
	}

	uc := NewRenewMaintainUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)
	result, err := uc.Execute(context.Background(), "user-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that renewal date was set (within last second)
	if time.Since(renewalDateSet) > time.Second {
		t.Error("Expected last_renewal_date to be set to current time")
	}

	if result.RenewedAt.IsZero() {
		t.Error("Expected RenewedAt to be set")
	}
}
```

---

### 1.9 TestRenewRotateExercisesUseCase

```go
// File: workout-tracker-back/internal/application/usecase/renew_rotate_exercises_test.go

package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/service"
)

func TestRenewRotateExercisesUseCase_Execute_Success(t *testing.T) {
	rotatedCount := 0

	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{
				PlanID:          "plan-123",
				MesocycleNumber: 3, // Mesocycle 3 allows rotation
			}, nil
		},
		UpdateMesocycleFunc: func(ctx context.Context, userID string, mesocycle int) error {
			return nil
		},
	}

	mockScheduleRepo := &MockScheduleRepository{
		ClearScheduleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}

	mockExerciseRepo := &MockExerciseRepository{
		GetCurrentExercisesFunc: func(ctx context.Context, userID string) ([]entity.UserExercise, error) {
			return []entity.UserExercise{
				{ExerciseID: "ex-1", Pattern: "push", Role: "compound"},
				{ExerciseID: "ex-2", Pattern: "pull", Role: "compound"},
				{ExerciseID: "ex-3", Pattern: "legs", Role: "isolation"},
			}, nil
		},
		FindAlternativeFunc: func(ctx context.Context, pattern, role string, excludeIDs []string, dislikedMuscles []string) (*entity.Exercise, error) {
			return &entity.Exercise{ID: "new-ex-" + pattern, Pattern: pattern, Role: role}, nil
		},
	}

	mockWorkoutRepo := &MockWorkoutRenewalRepository{
		RotateExerciseFunc: func(ctx context.Context, userID string, oldExerciseID, newExerciseID string) error {
			rotatedCount++
			return nil
		},
	}

	mockUserProfileRepo := &MockUserProfileRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
			return &entity.UserGymProfile{
				DislikedExercises: "Pantorrillas",
			}, nil
		},
	}

	rotationSvc := service.NewExerciseRotationService(mockExerciseRepo)

	uc := NewRenewRotateExercisesUseCase(
		mockPlanRepo, mockScheduleRepo, mockExerciseRepo, mockWorkoutRepo, mockUserProfileRepo, rotationSvc,
	)

	result, err := uc.Execute(context.Background(), "user-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// At mesocycle 3: compounds should rotate, isolation should rotate
	if rotatedCount == 0 {
		t.Error("Expected at least one exercise to be rotated")
	}

	if result.RenewalType != "rotate_exercises" {
		t.Errorf("Expected RenewalType 'rotate_exercises', got %q", result.RenewalType)
	}

	if len(result.RotatedExercises) == 0 {
		t.Error("Expected RotatedExercises to have entries")
	}
}

func TestRenewRotateExercisesUseCase_Execute_EmptyUserID(t *testing.T) {
	uc := NewRenewRotateExercisesUseCase(nil, nil, nil, nil, nil, nil)
	result, err := uc.Execute(context.Background(), "")

	if err == nil {
		t.Fatal("Expected error for empty user_id")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestRenewRotateExercisesUseCase_Execute_NoExercisesFound(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{PlanID: "plan-123", MesocycleNumber: 3}, nil
		},
	}

	mockExerciseRepo := &MockExerciseRepository{
		GetCurrentExercisesFunc: func(ctx context.Context, userID string) ([]entity.UserExercise, error) {
			return []entity.UserExercise{}, nil
		},
	}

	uc := NewRenewRotateExercisesUseCase(mockPlanRepo, nil, mockExerciseRepo, nil, nil, nil)
	result, err := uc.Execute(context.Background(), "user-123")

	if err == nil {
		t.Fatal("Expected error when no exercises found")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestRenewRotateExercisesUseCase_Execute_RespectsDislikedMuscles(t *testing.T) {
	var capturedDisliked []string

	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{PlanID: "plan-123", MesocycleNumber: 3}, nil
		},
		UpdateMesocycleFunc: func(ctx context.Context, userID string, mesocycle int) error {
			return nil
		},
	}

	mockScheduleRepo := &MockScheduleRepository{
		ClearScheduleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}

	mockExerciseRepo := &MockExerciseRepository{
		GetCurrentExercisesFunc: func(ctx context.Context, userID string) ([]entity.UserExercise, error) {
			return []entity.UserExercise{
				{ExerciseID: "ex-1", Pattern: "push", Role: "compound"},
			}, nil
		},
		FindAlternativeFunc: func(ctx context.Context, pattern, role string, excludeIDs []string, dislikedMuscles []string) (*entity.Exercise, error) {
			capturedDisliked = dislikedMuscles
			return &entity.Exercise{ID: "new-ex", Pattern: pattern, Role: role}, nil
		},
	}

	mockWorkoutRepo := &MockWorkoutRenewalRepository{
		RotateExerciseFunc: func(ctx context.Context, userID string, oldExerciseID, newExerciseID string) error {
			return nil
		},
	}

	mockUserProfileRepo := &MockUserProfileRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
			return &entity.UserGymProfile{
				DislikedExercises: "Pantorrillas, Hombros", // Spanish
			}, nil
		},
	}

	rotationSvc := service.NewExerciseRotationService(mockExerciseRepo)

	uc := NewRenewRotateExercisesUseCase(
		mockPlanRepo, mockScheduleRepo, mockExerciseRepo, mockWorkoutRepo, mockUserProfileRepo, rotationSvc,
	)

	_, _ = uc.Execute(context.Background(), "user-123")

	// Should have translated Spanish muscles to English
	if len(capturedDisliked) == 0 {
		t.Error("Expected disliked muscles to be passed")
	}
}

func TestRenewRotateExercisesUseCase_Execute_SkipsWhenNoAlternative(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{PlanID: "plan-123", MesocycleNumber: 3}, nil
		},
		UpdateMesocycleFunc: func(ctx context.Context, userID string, mesocycle int) error {
			return nil
		},
	}

	mockScheduleRepo := &MockScheduleRepository{
		ClearScheduleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}

	mockExerciseRepo := &MockExerciseRepository{
		GetCurrentExercisesFunc: func(ctx context.Context, userID string) ([]entity.UserExercise, error) {
			return []entity.UserExercise{
				{ExerciseID: "ex-1", Pattern: "push", Role: "compound"},
			}, nil
		},
		FindAlternativeFunc: func(ctx context.Context, pattern, role string, excludeIDs []string, dislikedMuscles []string) (*entity.Exercise, error) {
			return nil, nil // No alternative found
		},
	}

	mockWorkoutRepo := &MockWorkoutRenewalRepository{}

	mockUserProfileRepo := &MockUserProfileRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
			return &entity.UserGymProfile{}, nil
		},
	}

	rotationSvc := service.NewExerciseRotationService(mockExerciseRepo)

	uc := NewRenewRotateExercisesUseCase(
		mockPlanRepo, mockScheduleRepo, mockExerciseRepo, mockWorkoutRepo, mockUserProfileRepo, rotationSvc,
	)

	result, err := uc.Execute(context.Background(), "user-123")

	// Should not fail - just skip exercises with no alternatives
	if err != nil {
		t.Fatalf("Expected no error when no alternatives, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Rotated should be empty since no alternatives found
	if len(result.RotatedExercises) != 0 {
		t.Error("Expected no rotated exercises when no alternatives available")
	}
}
```

---

### 1.10 TestRenewChangeDaysUseCase

```go
// File: workout-tracker-back/internal/application/usecase/renew_change_days_test.go

package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

func TestRenewChangeDaysUseCase_Execute_Success(t *testing.T) {
	var capturedSchedule string
	var deletedWorkouts bool

	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{
				PlanID:          "plan-123",
				MesocycleNumber: 1,
				WeekSchedule:    "fb_3",
				DaysPerWeek:     3,
			}, nil
		},
		UpdateMesocycleFunc: func(ctx context.Context, userID string, mesocycle int) error {
			return nil
		},
		UpdateWeekScheduleFunc: func(ctx context.Context, userID string, schedule string) error {
			capturedSchedule = schedule
			return nil
		},
	}

	mockScheduleRepo := &MockScheduleRepository{
		ClearScheduleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}

	mockWorkoutRepo := &MockWorkoutRenewalRepository{
		DeleteAllWorkoutsFunc: func(ctx context.Context, userID string) error {
			deletedWorkouts = true
			return nil
		},
	}

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)
	result, err := uc.Execute(context.Background(), "user-123", 4) // Change from 3 to 4 days

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if capturedSchedule != "ul_4" {
		t.Errorf("Expected week_schedule 'ul_4', got %q", capturedSchedule)
	}

	if !deletedWorkouts {
		t.Error("Expected workouts to be deleted for regeneration")
	}

	if result.RenewalType != "change_days" {
		t.Errorf("Expected RenewalType 'change_days', got %q", result.RenewalType)
	}

	if result.NewDaysPerWeek != 4 {
		t.Errorf("Expected NewDaysPerWeek 4, got %d", result.NewDaysPerWeek)
	}

	if !result.RequiresRegeneration {
		t.Error("Expected RequiresRegeneration to be true")
	}
}

func TestRenewChangeDaysUseCase_Execute_ScheduleMapping(t *testing.T) {
	tests := []struct {
		name             string
		newDays          int
		expectedSchedule string
	}{
		{"2 days -> fb_2", 2, "fb_2"},
		{"3 days -> fb_3", 3, "fb_3"},
		{"4 days -> ul_4", 4, "ul_4"},
		{"5 days -> ppl_5", 5, "ppl_5"},
		{"6 days -> ppl_6", 6, "ppl_6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedSchedule string

			mockPlanRepo := &MockPlanRepository{
				GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
					return &entity.Plan{PlanID: "plan-123", MesocycleNumber: 1, DaysPerWeek: 3}, nil
				},
				UpdateMesocycleFunc: func(ctx context.Context, userID string, mesocycle int) error {
					return nil
				},
				UpdateWeekScheduleFunc: func(ctx context.Context, userID string, schedule string) error {
					capturedSchedule = schedule
					return nil
				},
			}

			mockScheduleRepo := &MockScheduleRepository{
				ClearScheduleFunc: func(ctx context.Context, userID string) error {
					return nil
				},
			}

			mockWorkoutRepo := &MockWorkoutRenewalRepository{
				DeleteAllWorkoutsFunc: func(ctx context.Context, userID string) error {
					return nil
				},
			}

			uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)
			_, err := uc.Execute(context.Background(), "user-123", tt.newDays)

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if capturedSchedule != tt.expectedSchedule {
				t.Errorf("For %d days, expected schedule %q, got %q", tt.newDays, tt.expectedSchedule, capturedSchedule)
			}
		})
	}
}

func TestRenewChangeDaysUseCase_Execute_EmptyUserID(t *testing.T) {
	uc := NewRenewChangeDaysUseCase(nil, nil, nil)
	result, err := uc.Execute(context.Background(), "", 4)

	if err == nil {
		t.Fatal("Expected error for empty user_id")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestRenewChangeDaysUseCase_Execute_InvalidDays(t *testing.T) {
	tests := []struct {
		name    string
		days    int
		wantErr bool
	}{
		{"0 days - invalid", 0, true},
		{"1 day - invalid", 1, true},
		{"2 days - valid", 2, false},
		{"6 days - valid", 6, false},
		{"7 days - invalid", 7, true},
		{"8 days - invalid", 8, true},
		{"-1 days - invalid", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPlanRepo := &MockPlanRepository{
				GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
					return &entity.Plan{PlanID: "plan-123", MesocycleNumber: 1}, nil
				},
				UpdateMesocycleFunc: func(ctx context.Context, userID string, mesocycle int) error {
					return nil
				},
				UpdateWeekScheduleFunc: func(ctx context.Context, userID string, schedule string) error {
					return nil
				},
			}

			mockScheduleRepo := &MockScheduleRepository{
				ClearScheduleFunc: func(ctx context.Context, userID string) error {
					return nil
				},
			}

			mockWorkoutRepo := &MockWorkoutRenewalRepository{
				DeleteAllWorkoutsFunc: func(ctx context.Context, userID string) error {
					return nil
				},
			}

			uc := NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)
			_, err := uc.Execute(context.Background(), "user-123", tt.days)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() with %d days: error = %v, wantErr = %v", tt.days, err, tt.wantErr)
			}
		})
	}
}

func TestRenewChangeDaysUseCase_Execute_SameDaysNoChange(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{
				PlanID:       "plan-123",
				DaysPerWeek:  3,
				WeekSchedule: "fb_3",
			}, nil
		},
	}

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, nil, nil)
	result, err := uc.Execute(context.Background(), "user-123", 3) // Same days

	if err == nil {
		t.Fatal("Expected error when days unchanged")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestRenewChangeDaysUseCase_Execute_PlanNotFound(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return nil, nil
		},
	}

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, nil, nil)
	result, err := uc.Execute(context.Background(), "user-123", 4)

	if err == nil {
		t.Fatal("Expected error when plan not found")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestRenewChangeDaysUseCase_Execute_UpdateScheduleError(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.Plan, error) {
			return &entity.Plan{PlanID: "plan-123", DaysPerWeek: 3}, nil
		},
		UpdateWeekScheduleFunc: func(ctx context.Context, userID string, schedule string) error {
			return errors.New("database error")
		},
	}

	uc := NewRenewChangeDaysUseCase(mockPlanRepo, nil, nil)
	result, err := uc.Execute(context.Background(), "user-123", 4)

	if err == nil {
		t.Fatal("Expected error from update")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}
```

---

## 2. E2E Test Cases (GymRatFlow_E2E_TestRunner)

Add these test cases to the existing E2E test runner workflow.

### Test Cases Matrix

| ID | Name | Usuario | Categoria | Prioridad | Tipo |
|----|------|---------|-----------|-----------|------|
| TC_REN_001 | Auto-detect week 4 completion | Test_RenewalComplete | RENEWAL | CRITICAL | SINGLE |
| TC_REN_002 | Manual "quiero renovar" request | Test_RenewalComplete | RENEWAL | HIGH | SINGLE |
| TC_REN_003 | MANTENER_RUTINA path | Test_RenewalComplete | RENEWAL | CRITICAL | SINGLE |
| TC_REN_004 | CAMBIAR_DIAS:3 path | Test_RenewalChangeDays | RENEWAL | HIGH | SINGLE |
| TC_REN_005 | ROTAR_EJERCICIOS path | Test_RenewalRotate | RENEWAL | HIGH | SINGLE |
| TC_REN_006 | MODIFICAR_PERFIL path | Test_RenewalProfile | RENEWAL | MEDIUM | SINGLE |
| TC_REN_007 | Invalid days (1 or 7+) handling | Test_RenewalComplete | RENEWAL | MEDIUM | SINGLE |

---

### TC_REN_001: Auto-detect week 4 completion

**Prioridad:** CRITICAL | **Tipo:** SINGLE | **Usuario:** `570000000010`

**Objetivo:** When a user with all week 4 sessions completed sends any message, the system should automatically detect completion and offer renewal options.

**Preconditions:**
- User has all week 4 sessions marked as `Completed = true`
- User has no future scheduled workouts

**Input:** `"Hola, que hay para hoy?"`

**Expected Output:** Message containing renewal options:
- "completado" or "mesociclo" or "renovar"
- Options: MANTENER, CAMBIAR, ROTAR

**Metrica:**
```javascript
output.includes('completado') || output.includes('mesociclo') || output.includes('renovar') || output.includes('felicitaciones')
```

---

### TC_REN_002: Manual "quiero renovar" request

**Prioridad:** HIGH | **Tipo:** SINGLE | **Usuario:** `570000000010`

**Objetivo:** User can explicitly request renewal even without completion detection.

**Input:** `"Quiero renovar mi rutina"`

**Expected Output:** Renewal options presented

**Metrica:**
```javascript
output.includes('renovar') || output.includes('opciones') || output.includes('mantener')
```

---

### TC_REN_003: MANTENER_RUTINA path

**Prioridad:** CRITICAL | **Tipo:** SINGLE | **Usuario:** `570000000010`

**Objetivo:** User selects to maintain their current routine with load progression.

**Cleanup Automatico:**
```sql
-- Reset to week 4 complete state
UPDATE users_plans SET mesocycle_number = 1 WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
UPDATE user_weekly_schedule SET "Completed" = true WHERE user_id = 'e2e00010-0000-0000-0000-000000000010' AND week = 4;
DELETE FROM user_weekly_schedule WHERE user_id = 'e2e00010-0000-0000-0000-000000000010' AND week < 4;
```

**Input:** `"Quiero mantener mi rutina actual"`

**DB Verification (Ground Truth):**
```sql
SELECT mesocycle_number FROM users_plans WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
-- Expected: 2 (incremented from 1)

SELECT COUNT(*) FROM user_weekly_schedule WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
-- Expected: 0 (schedule cleared for re-scheduling)
```

**Metrica:**
```javascript
dbPassed === true && (output.includes('renovado') || output.includes('progresión') || output.includes('listo'))
```

---

### TC_REN_004: CAMBIAR_DIAS:3 path

**Prioridad:** HIGH | **Tipo:** SINGLE | **Usuario:** `570000000011`

**Objetivo:** User changes from 4 days to 3 days per week.

**Preconditions:**
- User currently has 4 days/week schedule (`ul_4`)

**Cleanup Automatico:**
```sql
UPDATE users_plans SET week_schedule = 'ul_4', mesocycle_number = 1 WHERE user_id = 'e2e00011-0000-0000-0000-000000000011';
```

**Input:** `"Quiero cambiar a 3 días por semana"`

**DB Verification (Ground Truth):**
```sql
SELECT week_schedule FROM users_plans WHERE user_id = 'e2e00011-0000-0000-0000-000000000011';
-- Expected: 'fb_3'

SELECT COUNT(*) FROM workouts WHERE user_id = 'e2e00011-0000-0000-0000-000000000011';
-- Expected: 0 (workouts deleted for regeneration by GymRatForm)
```

**Metrica:**
```javascript
dbPassed === true && (output.includes('3 días') || output.includes('nueva rutina') || output.includes('generando'))
```

---

### TC_REN_005: ROTAR_EJERCICIOS path

**Prioridad:** HIGH | **Tipo:** SINGLE | **Usuario:** `570000000012`

**Objetivo:** User requests new exercises while keeping the same frequency.

**Input:** `"Quiero nuevos ejercicios"`

**DB Verification (Ground Truth):**
```sql
-- Check that exercises changed but pattern preserved
SELECT w.exercise_id, e.pattern, e.role
FROM workouts w
JOIN exercises e USING (exercise_id)
WHERE w.user_id = 'e2e00012-0000-0000-0000-000000000012' AND w.week = 1;
-- Verify: Same patterns as before, different exercise_ids
```

**Metrica:**
```javascript
output.includes('ejercicios') && (output.includes('rotado') || output.includes('nuevo') || output.includes('actualizado'))
```

---

### TC_REN_006: MODIFICAR_PERFIL path

**Prioridad:** MEDIUM | **Tipo:** SINGLE | **Usuario:** `570000000013`

**Objetivo:** User wants to update their profile preferences before renewal.

**Input:** `"Quiero cambiar mis prioridades musculares"`

**Expected Output:** AI agent asks about new preferences

**Metrica:**
```javascript
output.includes('prioridad') || output.includes('músculo') || output.includes('preferencias') || output.includes('cambiar')
```

---

### TC_REN_007: Invalid days handling

**Prioridad:** MEDIUM | **Tipo:** SINGLE | **Usuario:** `570000000010`

**Input 1:** `"Quiero entrenar 1 día a la semana"`

**Expected Output 1:** Error or clarification that minimum is 2 days

**Metrica 1:**
```javascript
output.includes('mínimo') || output.includes('2') || output.includes('al menos')
```

**Input 2:** `"Quiero entrenar 7 días a la semana"`

**Expected Output 2:** Error or clarification that maximum is 6 days

**Metrica 2:**
```javascript
output.includes('máximo') || output.includes('6') || output.includes('descanso')
```

---

## 3. Test Fixture Data (SQL)

```sql
-- ============================================
-- MESOCYCLE RENEWAL TEST DATA
-- Reserved phones: 570000000010-570000000019
-- ============================================
--
-- USUARIOS DUMMY (Fixtures):
-- 570000000010 - Test_RenewalComplete (TC_REN_001, TC_REN_002, TC_REN_003, TC_REN_007)
-- 570000000011 - Test_RenewalChangeDays (TC_REN_004)
-- 570000000012 - Test_RenewalRotate (TC_REN_005)
-- 570000000013 - Test_RenewalProfile (TC_REN_006)
--
-- ============================================

-- ============================================
-- SECTION 1: TEARDOWN
-- ============================================

-- Delete pending_tasks for renewal test users
DELETE FROM pending_tasks
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000010', '570000000011', '570000000012', '570000000013')
);

-- Delete workouts for renewal test users
DELETE FROM workouts
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000010', '570000000011', '570000000012', '570000000013')
);

-- Delete schedules for renewal test users
DELETE FROM user_weekly_schedule
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000010', '570000000011', '570000000012', '570000000013')
);

-- Delete plans for renewal test users
DELETE FROM users_plans
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000010', '570000000011', '570000000012', '570000000013')
);

-- Delete users
DELETE FROM users
WHERE full_phone_number::text IN ('570000000010', '570000000011', '570000000012', '570000000013');

-- Delete gym profiles (whatsapp_id is BIGINT)
DELETE FROM users_gym_profile
WHERE whatsapp_id IN (570000000010, 570000000011, 570000000012, 570000000013);

-- Clean chat histories
DELETE FROM n8n_chat_histories
WHERE session_id LIKE '%570000000010%'
   OR session_id LIKE '%570000000011%'
   OR session_id LIKE '%570000000012%'
   OR session_id LIKE '%570000000013%'
   OR session_id LIKE '%e2e00010%'
   OR session_id LIKE '%e2e00011%'
   OR session_id LIKE '%e2e00012%'
   OR session_id LIKE '%e2e00013%';

-- ============================================
-- SECTION 2: CREATE USERS
-- ============================================

-- User 10: Test_RenewalComplete (week 4 all completed)
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00010-0000-0000-0000-000000000010',
    'Test RenewalComplete',
    'test_renewal_complete@gymbot.test',
    '570000000010',
    0000000010,
    57,
    'America/Bogota',
    NOW()
);

-- User 11: Test_RenewalChangeDays (for changing frequency)
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00011-0000-0000-0000-000000000011',
    'Test RenewalChangeDays',
    'test_renewal_changedays@gymbot.test',
    '570000000011',
    0000000011,
    57,
    'America/Bogota',
    NOW()
);

-- User 12: Test_RenewalRotate (for exercise rotation)
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00012-0000-0000-0000-000000000012',
    'Test RenewalRotate',
    'test_renewal_rotate@gymbot.test',
    '570000000012',
    0000000012,
    57,
    'America/Bogota',
    NOW()
);

-- User 13: Test_RenewalProfile (for profile modification)
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00013-0000-0000-0000-000000000013',
    'Test RenewalProfile',
    'test_renewal_profile@gymbot.test',
    '570000000013',
    0000000013,
    57,
    'America/Bogota',
    NOW()
);

-- ============================================
-- SECTION 3: CREATE GYM PROFILES
-- ============================================

-- Profile for User 12 (rotation test - needs disliked muscles)
INSERT INTO users_gym_profile (
    whatsapp_id, full_name, email, sex, age, weight, height,
    training_experience, session_duration_mins, gym_equipment_access,
    goal, priority_muscles, disliked_exercises, health_status,
    days_available, observation, collected_at, level
)
VALUES (
    570000000012,
    'Test RenewalRotate',
    'test_renewal_rotate@gymbot.test',
    'M',
    30,
    75,
    175,
    'Más de 3 años',
    '60-75 min',
    'Gimnasio completo',
    'Hipertrofia',
    'Pecho, Espalda',
    'Pantorrillas, Hombros',  -- Disliked muscles for testing rotation exclusion
    'A',
    3,
    'Test user for rotation testing',
    NOW(),
    'Intermedio'
);

-- Profile for User 13 (profile modification test)
INSERT INTO users_gym_profile (
    whatsapp_id, full_name, email, sex, age, weight, height,
    training_experience, session_duration_mins, gym_equipment_access,
    goal, priority_muscles, disliked_exercises, health_status,
    days_available, observation, collected_at, level
)
VALUES (
    570000000013,
    'Test RenewalProfile',
    'test_renewal_profile@gymbot.test',
    'F',
    28,
    60,
    165,
    '1 a 2 años',
    '45-60 min',
    'Gimnasio completo',
    'Tonificación',
    'Glúteo, Pierna',
    '',
    'A',
    4,
    'Test user for profile modification testing',
    NOW(),
    'Intermedio'
);

-- ============================================
-- SECTION 4: CREATE PLANS
-- ============================================

-- Plan for User 10 (week 4 complete, 3 days)
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date, mesocycle_number, last_renewal_date)
SELECT
    'e2e00010-0000-0000-0001-000000000010',
    'e2e00010-0000-0000-0000-000000000010',
    template_id,
    'fb_3',
    goal,
    level,
    'active',
    NOW() - INTERVAL '28 days',  -- Started 4 weeks ago
    1,
    NULL
FROM routine_templates
WHERE week_schedule = 'fb_3' AND level = 'Intermedio'
LIMIT 1;

-- Plan for User 11 (4 days, for changing to 3)
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date, mesocycle_number, last_renewal_date)
SELECT
    'e2e00011-0000-0000-0001-000000000011',
    'e2e00011-0000-0000-0000-000000000011',
    template_id,
    'ul_4',
    goal,
    level,
    'active',
    NOW() - INTERVAL '28 days',
    1,
    NULL
FROM routine_templates
WHERE week_schedule = 'ul_4' AND level = 'Intermedio'
LIMIT 1;

-- Plan for User 12 (3 days, for rotation)
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date, mesocycle_number, last_renewal_date)
SELECT
    'e2e00012-0000-0000-0001-000000000012',
    'e2e00012-0000-0000-0000-000000000012',
    template_id,
    'fb_3',
    goal,
    level,
    'active',
    NOW() - INTERVAL '28 days',
    3,  -- Mesocycle 3 for rotation eligibility
    NULL
FROM routine_templates
WHERE week_schedule = 'fb_3' AND level = 'Intermedio'
LIMIT 1;

-- Plan for User 13 (4 days, for profile modification)
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date, mesocycle_number, last_renewal_date)
SELECT
    'e2e00013-0000-0000-0001-000000000013',
    'e2e00013-0000-0000-0000-000000000013',
    template_id,
    'ul_4',
    goal,
    level,
    'active',
    NOW() - INTERVAL '28 days',
    1,
    NULL
FROM routine_templates
WHERE week_schedule = 'ul_4' AND level = 'Intermedio'
LIMIT 1;

-- ============================================
-- SECTION 5: CREATE WEEK 4 SCHEDULES (COMPLETED)
-- User 10: All week 4 sessions completed (triggers renewal)
-- ============================================

-- User 10: Week 4 completed (3 days)
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES
    ('e2e00010-0000-0000-0000-000000000010', 4, 'Lunes', 'Dia 1 - Full Body A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00010-0000-0000-0000-000000000010', 4, 'Miercoles', 'Dia 2 - Full Body B', (NOW() - INTERVAL '5 days')::date::text, true),
    ('e2e00010-0000-0000-0000-000000000010', 4, 'Viernes', 'Dia 3 - Full Body C', (NOW() - INTERVAL '3 days')::date::text, true);

-- User 11: Week 4 completed (4 days)
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Lunes', 'Dia 1 - Upper A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Martes', 'Dia 2 - Lower A', (NOW() - INTERVAL '6 days')::date::text, true),
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Jueves', 'Dia 3 - Upper B', (NOW() - INTERVAL '4 days')::date::text, true),
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Viernes', 'Dia 4 - Lower B', (NOW() - INTERVAL '3 days')::date::text, true);

-- User 12: Week 4 completed (3 days - for rotation)
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES
    ('e2e00012-0000-0000-0000-000000000012', 4, 'Lunes', 'Dia 1 - Full Body A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00012-0000-0000-0000-000000000012', 4, 'Miercoles', 'Dia 2 - Full Body B', (NOW() - INTERVAL '5 days')::date::text, true),
    ('e2e00012-0000-0000-0000-000000000012', 4, 'Viernes', 'Dia 3 - Full Body C', (NOW() - INTERVAL '3 days')::date::text, true);

-- User 13: Week 4 completed (4 days - for profile mod)
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Lunes', 'Dia 1 - Upper A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Martes', 'Dia 2 - Lower A', (NOW() - INTERVAL '6 days')::date::text, true),
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Jueves', 'Dia 3 - Upper B', (NOW() - INTERVAL '4 days')::date::text, true),
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Viernes', 'Dia 4 - Lower B', (NOW() - INTERVAL '3 days')::date::text, true);

-- ============================================
-- SECTION 6: CREATE WORKOUTS FOR ROTATION TEST
-- User 12 needs existing workouts to rotate
-- ============================================

-- Insert workout exercises for User 12 (week 1 - will be rotated)
INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, created_at, notes, exercise_order)
SELECT
    'e2e00012-0000-0000-0000-000000000012' as user_id,
    1 as week,
    'Dia 1 - Full Body A' as day_name,
    exercise_id,
    '3' as sets,
    '8-10' as reps,
    '2' as rir,
    120 as "rest-seconds",
    '2-0-2-0' as tempo,
    NOW() as created_at,
    '' as notes,
    ROW_NUMBER() OVER (ORDER BY
        CASE role
            WHEN 'compound' THEN 1
            WHEN 'core' THEN 2
            ELSE 3
        END
    ) as exercise_order
FROM exercises
WHERE pattern IN ('push', 'pull', 'legs')
AND role IN ('compound', 'isolation')
AND level IN ('Intermedio', 'Principiante')
AND main_muscle NOT IN ('Hombros', 'Pantorrillas')  -- Respect disliked muscles
LIMIT 6;

-- ============================================
-- SECTION 7: VERIFICATION QUERIES
-- ============================================

-- Verify users created
SELECT user_id, full_name, full_phone_number
FROM users
WHERE full_phone_number::text LIKE '5700000001%'
ORDER BY full_phone_number;

-- Verify plans with mesocycle info
SELECT u.full_name, up.week_schedule, up.mesocycle_number, up.goal, up.level
FROM users_plans up
JOIN users u USING (user_id)
WHERE u.full_phone_number::text LIKE '5700000001%';

-- Verify week 4 completion status
SELECT u.full_name,
       COUNT(*) as total_sessions,
       COUNT(*) FILTER (WHERE "Completed" = true) as completed_sessions
FROM user_weekly_schedule uws
JOIN users u USING (user_id)
WHERE u.full_phone_number::text LIKE '5700000001%'
AND uws.week = 4
GROUP BY u.full_name;

-- Verify gym profiles for rotation test
SELECT whatsapp_id, priority_muscles, disliked_exercises, health_status
FROM users_gym_profile
WHERE whatsapp_id IN (570000000012, 570000000013);

-- Verify workouts for rotation test user
SELECT u.full_name, w.day_name, e.spanish_name, e.pattern, e.role, w.exercise_order
FROM workouts w
JOIN users u USING (user_id)
JOIN exercises e USING (exercise_id)
WHERE u.full_phone_number = '570000000012'
ORDER BY w.exercise_order;

-- ============================================
-- SUMMARY OF STATES
-- ============================================
--
-- | Usuario              | Phone        | Plan  | Week4   | Meso# | Workouts | Purpose                    |
-- |----------------------|--------------|-------|---------|-------|----------|----------------------------|
-- | Test_RenewalComplete | 570000000010 | fb_3  | DONE    | 1     | NO       | TC_REN_001-003, 007        |
-- | Test_RenewalChangeDays| 570000000011| ul_4  | DONE    | 1     | NO       | TC_REN_004 (change to 3d)  |
-- | Test_RenewalRotate   | 570000000012 | fb_3  | DONE    | 3     | YES      | TC_REN_005 (rotate exs)    |
-- | Test_RenewalProfile  | 570000000013 | ul_4  | DONE    | 1     | NO       | TC_REN_006 (modify profile)|
--
-- ============================================
```

---

## 4. Integration Test Scripts (cURL)

### 4.1 Check Mesocycle Status

```bash
#!/bin/bash
# test_check_mesocycle_status.sh

API_URL="https://workout-api-148665080566.us-central1.run.app/api/v1"
USER_ID="e2e00010-0000-0000-0000-000000000010"

echo "=== TC_REN_001: Check Mesocycle Status ==="
echo "Testing user: $USER_ID"

response=$(curl -s -w "\n%{http_code}" \
  -X GET "$API_URL/plans/$USER_ID/mesocycle-status" \
  -H "Content-Type: application/json")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

echo "HTTP Status: $http_code"
echo "Response: $body"

# Expected response for complete mesocycle:
# {
#   "is_complete": true,
#   "mesocycle_number": 1,
#   "completed_sessions": 3,
#   "required_sessions": 3,
#   "days_per_week": 3,
#   "week_schedule": "fb_3"
# }

if [ "$http_code" = "200" ]; then
    is_complete=$(echo "$body" | jq -r '.is_complete')
    if [ "$is_complete" = "true" ]; then
        echo "PASS: Mesocycle detected as complete"
    else
        echo "FAIL: Expected is_complete=true"
    fi
else
    echo "FAIL: Expected HTTP 200, got $http_code"
fi
```

---

### 4.2 Renew Maintain

```bash
#!/bin/bash
# test_renew_maintain.sh

API_URL="https://workout-api-148665080566.us-central1.run.app/api/v1"
USER_ID="e2e00010-0000-0000-0000-000000000010"

echo "=== TC_REN_003: Renew Maintain ==="
echo "Testing user: $USER_ID"

response=$(curl -s -w "\n%{http_code}" \
  -X POST "$API_URL/plans/$USER_ID/renew/maintain" \
  -H "Content-Type: application/json")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

echo "HTTP Status: $http_code"
echo "Response: $body"

# Expected response:
# {
#   "success": true,
#   "renewal_type": "maintain",
#   "new_mesocycle_number": 2,
#   "renewed_at": "2026-02-01T..."
# }

if [ "$http_code" = "200" ]; then
    new_meso=$(echo "$body" | jq -r '.new_mesocycle_number')
    if [ "$new_meso" = "2" ]; then
        echo "PASS: Mesocycle incremented to 2"
    else
        echo "FAIL: Expected new_mesocycle_number=2, got $new_meso"
    fi
else
    echo "FAIL: Expected HTTP 200, got $http_code"
fi
```

---

### 4.3 Renew Change Days

```bash
#!/bin/bash
# test_renew_change_days.sh

API_URL="https://workout-api-148665080566.us-central1.run.app/api/v1"
USER_ID="e2e00011-0000-0000-0000-000000000011"

echo "=== TC_REN_004: Renew Change Days (4 -> 3) ==="
echo "Testing user: $USER_ID"

response=$(curl -s -w "\n%{http_code}" \
  -X POST "$API_URL/plans/$USER_ID/renew/change-days" \
  -H "Content-Type: application/json" \
  -d '{"new_days_per_week": 3}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

echo "HTTP Status: $http_code"
echo "Response: $body"

# Expected response:
# {
#   "success": true,
#   "renewal_type": "change_days",
#   "new_mesocycle_number": 2,
#   "new_days_per_week": 3,
#   "new_week_schedule": "fb_3",
#   "requires_regeneration": true
# }

if [ "$http_code" = "200" ]; then
    new_schedule=$(echo "$body" | jq -r '.new_week_schedule')
    requires_regen=$(echo "$body" | jq -r '.requires_regeneration')
    if [ "$new_schedule" = "fb_3" ] && [ "$requires_regen" = "true" ]; then
        echo "PASS: Schedule changed to fb_3, regeneration required"
    else
        echo "FAIL: Unexpected values - schedule=$new_schedule, regen=$requires_regen"
    fi
else
    echo "FAIL: Expected HTTP 200, got $http_code"
fi
```

---

### 4.4 Renew Change Days - Invalid (1 day)

```bash
#!/bin/bash
# test_renew_change_days_invalid.sh

API_URL="https://workout-api-148665080566.us-central1.run.app/api/v1"
USER_ID="e2e00010-0000-0000-0000-000000000010"

echo "=== TC_REN_007: Invalid Days (1 day) ==="
echo "Testing user: $USER_ID"

response=$(curl -s -w "\n%{http_code}" \
  -X POST "$API_URL/plans/$USER_ID/renew/change-days" \
  -H "Content-Type: application/json" \
  -d '{"new_days_per_week": 1}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

echo "HTTP Status: $http_code"
echo "Response: $body"

# Expected: HTTP 400 Bad Request
# {
#   "error": "invalid_days",
#   "message": "Days per week must be between 2 and 6"
# }

if [ "$http_code" = "400" ]; then
    echo "PASS: Correctly rejected 1 day request"
else
    echo "FAIL: Expected HTTP 400, got $http_code"
fi

echo ""
echo "=== TC_REN_007: Invalid Days (7 days) ==="

response=$(curl -s -w "\n%{http_code}" \
  -X POST "$API_URL/plans/$USER_ID/renew/change-days" \
  -H "Content-Type: application/json" \
  -d '{"new_days_per_week": 7}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

echo "HTTP Status: $http_code"
echo "Response: $body"

if [ "$http_code" = "400" ]; then
    echo "PASS: Correctly rejected 7 days request"
else
    echo "FAIL: Expected HTTP 400, got $http_code"
fi
```

---

### 4.5 Renew Rotate Exercises

```bash
#!/bin/bash
# test_renew_rotate_exercises.sh

API_URL="https://workout-api-148665080566.us-central1.run.app/api/v1"
USER_ID="e2e00012-0000-0000-0000-000000000012"

echo "=== TC_REN_005: Renew Rotate Exercises ==="
echo "Testing user: $USER_ID"

# First, get current exercises
echo "Current exercises:"
current_exercises=$(curl -s "$API_URL/workouts/today?user_id=$USER_ID" | jq -r '.exercises[].name')
echo "$current_exercises"

response=$(curl -s -w "\n%{http_code}" \
  -X POST "$API_URL/plans/$USER_ID/renew/rotate-exercises" \
  -H "Content-Type: application/json")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

echo ""
echo "HTTP Status: $http_code"
echo "Response: $body"

# Expected response:
# {
#   "success": true,
#   "renewal_type": "rotate_exercises",
#   "new_mesocycle_number": 4,
#   "rotated_exercises": [
#     {"old_id": "...", "new_id": "...", "pattern": "push"},
#     ...
#   ]
# }

if [ "$http_code" = "200" ]; then
    rotated_count=$(echo "$body" | jq '.rotated_exercises | length')
    if [ "$rotated_count" -gt "0" ]; then
        echo "PASS: $rotated_count exercises rotated"
    else
        echo "INFO: No exercises rotated (may be expected if no alternatives)"
    fi
else
    echo "FAIL: Expected HTTP 200, got $http_code"
fi
```

---

## 5. Manual Test Checklist

### 5.1 WhatsApp Message Flow Verification

| Step | Action | Expected Result | Pass? |
|------|--------|-----------------|-------|
| 1 | Send "Hola" as user at week 4 complete | Receive renewal options message | [ ] |
| 2 | Reply "Mantener" | Receive confirmation, mesocycle incremented | [ ] |
| 3 | Check DB: `mesocycle_number` | Value = previous + 1 | [ ] |
| 4 | Check DB: `user_weekly_schedule` | All rows deleted (ready for re-scheduling) | [ ] |
| 5 | Send "Quiero cambiar a 5 días" | Receive confirmation, routine regeneration starts | [ ] |
| 6 | Wait 30 seconds, check DB: `workouts` | New workouts created with 5 days/week pattern | [ ] |
| 7 | Send "Quiero nuevos ejercicios" | Receive confirmation with rotated exercises | [ ] |
| 8 | Check DB: `workouts.exercise_id` | Different IDs but same patterns | [ ] |
| 9 | Send "Quiero 1 día" | Receive error message about minimum 2 days | [ ] |
| 10 | Send "Quiero 8 días" | Receive error message about maximum 6 days | [ ] |

---

### 5.2 Database State Verification Queries

```sql
-- Run after each renewal test to verify state

-- 1. Check mesocycle increment after MANTENER
SELECT user_id, mesocycle_number, last_renewal_date
FROM users_plans
WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
-- Expected: mesocycle_number = 2, last_renewal_date = today

-- 2. Check schedule cleared after any renewal
SELECT COUNT(*) as schedule_count
FROM user_weekly_schedule
WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
-- Expected: 0

-- 3. Check week_schedule changed after CAMBIAR_DIAS
SELECT week_schedule, mesocycle_number
FROM users_plans
WHERE user_id = 'e2e00011-0000-0000-0000-000000000011';
-- Expected: fb_3 (changed from ul_4)

-- 4. Check workouts deleted for regeneration after CAMBIAR_DIAS
SELECT COUNT(*) as workout_count
FROM workouts
WHERE user_id = 'e2e00011-0000-0000-0000-000000000011';
-- Expected: 0 (deleted, waiting for GymRatForm regeneration)

-- 5. Check exercise rotation after ROTAR_EJERCICIOS
WITH old_exercises AS (
    SELECT exercise_id, day_name, exercise_order
    FROM workouts_backup  -- Would need to capture before
    WHERE user_id = 'e2e00012-0000-0000-0000-000000000012'
),
new_exercises AS (
    SELECT w.exercise_id, w.day_name, w.exercise_order, e.pattern, e.role
    FROM workouts w
    JOIN exercises e USING (exercise_id)
    WHERE w.user_id = 'e2e00012-0000-0000-0000-000000000012'
)
SELECT n.day_name, n.exercise_order, n.pattern, n.role,
       CASE WHEN o.exercise_id != n.exercise_id THEN 'ROTATED' ELSE 'SAME' END as status
FROM new_exercises n
LEFT JOIN old_exercises o ON n.day_name = o.day_name AND n.exercise_order = o.exercise_order;
-- Verify: Patterns same, some exercise_ids different

-- 6. Verify disliked muscles excluded in rotation
SELECT w.exercise_id, e.spanish_name, e.main_muscle
FROM workouts w
JOIN exercises e USING (exercise_id)
WHERE w.user_id = 'e2e00012-0000-0000-0000-000000000012'
AND e.main_muscle IN ('Hombros', 'Pantorrillas');
-- Expected: 0 rows (disliked muscles excluded)
```

---

## 6. Test Data Cleanup Script

```sql
-- ============================================
-- CLEANUP SCRIPT - Run after testing
-- Resets renewal test users to initial state
-- ============================================

-- ============================================
-- OPTION 1: FULL RESET (delete and recreate)
-- ============================================

-- Delete all data for renewal test users
DELETE FROM pending_tasks WHERE user_id IN (
    SELECT user_id FROM users WHERE full_phone_number::text LIKE '5700000001%'
);

DELETE FROM workouts WHERE user_id IN (
    SELECT user_id FROM users WHERE full_phone_number::text LIKE '5700000001%'
);

DELETE FROM user_weekly_schedule WHERE user_id IN (
    SELECT user_id FROM users WHERE full_phone_number::text LIKE '5700000001%'
);

DELETE FROM users_plans WHERE user_id IN (
    SELECT user_id FROM users WHERE full_phone_number::text LIKE '5700000001%'
);

DELETE FROM users WHERE full_phone_number::text LIKE '5700000001%';

DELETE FROM users_gym_profile WHERE whatsapp_id BETWEEN 570000000010 AND 570000000019;

DELETE FROM n8n_chat_histories WHERE session_id LIKE '%5700000001%';

-- Then re-run the fixture SQL from Section 3

-- ============================================
-- OPTION 2: SOFT RESET (restore initial state)
-- ============================================

-- Reset mesocycle numbers to 1
UPDATE users_plans
SET mesocycle_number = 1, last_renewal_date = NULL
WHERE user_id IN (
    SELECT user_id FROM users WHERE full_phone_number::text LIKE '5700000001%'
);

-- Restore week schedules to original
UPDATE users_plans SET week_schedule = 'fb_3' WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
UPDATE users_plans SET week_schedule = 'ul_4' WHERE user_id = 'e2e00011-0000-0000-0000-000000000011';
UPDATE users_plans SET week_schedule = 'fb_3' WHERE user_id = 'e2e00012-0000-0000-0000-000000000012';
UPDATE users_plans SET week_schedule = 'ul_4' WHERE user_id = 'e2e00013-0000-0000-0000-000000000013';

-- Clear schedules (they will be recreated)
DELETE FROM user_weekly_schedule WHERE user_id IN (
    SELECT user_id FROM users WHERE full_phone_number::text LIKE '5700000001%'
);

-- Recreate week 4 completed schedules
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES
    ('e2e00010-0000-0000-0000-000000000010', 4, 'Lunes', 'Dia 1 - Full Body A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00010-0000-0000-0000-000000000010', 4, 'Miercoles', 'Dia 2 - Full Body B', (NOW() - INTERVAL '5 days')::date::text, true),
    ('e2e00010-0000-0000-0000-000000000010', 4, 'Viernes', 'Dia 3 - Full Body C', (NOW() - INTERVAL '3 days')::date::text, true),
    -- User 11
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Lunes', 'Dia 1 - Upper A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Martes', 'Dia 2 - Lower A', (NOW() - INTERVAL '6 days')::date::text, true),
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Jueves', 'Dia 3 - Upper B', (NOW() - INTERVAL '4 days')::date::text, true),
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Viernes', 'Dia 4 - Lower B', (NOW() - INTERVAL '3 days')::date::text, true),
    -- User 12
    ('e2e00012-0000-0000-0000-000000000012', 4, 'Lunes', 'Dia 1 - Full Body A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00012-0000-0000-0000-000000000012', 4, 'Miercoles', 'Dia 2 - Full Body B', (NOW() - INTERVAL '5 days')::date::text, true),
    ('e2e00012-0000-0000-0000-000000000012', 4, 'Viernes', 'Dia 3 - Full Body C', (NOW() - INTERVAL '3 days')::date::text, true),
    -- User 13
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Lunes', 'Dia 1 - Upper A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Martes', 'Dia 2 - Lower A', (NOW() - INTERVAL '6 days')::date::text, true),
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Jueves', 'Dia 3 - Upper B', (NOW() - INTERVAL '4 days')::date::text, true),
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Viernes', 'Dia 4 - Lower B', (NOW() - INTERVAL '3 days')::date::text, true);

-- Clear chat histories
DELETE FROM n8n_chat_histories WHERE session_id LIKE '%5700000001%' OR session_id LIKE '%e2e0001%';

-- ============================================
-- VERIFICATION
-- ============================================

SELECT u.full_name, up.week_schedule, up.mesocycle_number,
       COUNT(uws.*) FILTER (WHERE uws."Completed" = true) as completed_w4
FROM users u
JOIN users_plans up USING (user_id)
LEFT JOIN user_weekly_schedule uws ON u.user_id = uws.user_id AND uws.week = 4
WHERE u.full_phone_number::text LIKE '5700000001%'
GROUP BY u.full_name, up.week_schedule, up.mesocycle_number
ORDER BY u.full_name;
```

---

## Appendix: Test Execution Summary

### Quick Reference Commands

```bash
# Run all Go unit tests for renewal
cd /Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back
go test ./internal/domain/service/... -v -run "Preference|Rotation"
go test ./internal/application/usecase/... -v -run "Mesocycle|Renew"

# Run integration tests
chmod +x ./tests/*.sh
./tests/test_check_mesocycle_status.sh
./tests/test_renew_maintain.sh
./tests/test_renew_change_days.sh
./tests/test_renew_rotate_exercises.sh

# Setup test fixtures
psql $SUPABASE_URL -f /Users/camilodiazjaimes/Documents/GymBot/spec/mesocycle-renewal/test_fixtures.sql

# Cleanup after testing
psql $SUPABASE_URL -c "SELECT * FROM cleanup_renewal_tests();"
```

### Test User Reference

| Phone | User ID | Name | Purpose |
|-------|---------|------|---------|
| `570000000010` | `e2e00010-...-10` | Test_RenewalComplete | Auto-detect, maintain, invalid days |
| `570000000011` | `e2e00011-...-11` | Test_RenewalChangeDays | Change from 4 to 3 days |
| `570000000012` | `e2e00012-...-12` | Test_RenewalRotate | Exercise rotation with preferences |
| `570000000013` | `e2e00013-...-13` | Test_RenewalProfile | Profile modification flow |

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| v1.0 | 2026-02-01 | Initial test plan with Go unit tests, E2E cases, fixtures, integration scripts |
