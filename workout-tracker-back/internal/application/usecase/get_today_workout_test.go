package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// MockWorkoutReader is a mock implementation of WorkoutReader
type MockWorkoutReader struct {
	GetTodayWorkoutFunc func(ctx context.Context, userID string) (*entity.Workout, error)
	GetByIDFunc         func(ctx context.Context, workoutID string) (*entity.Workout, error)
}

func (m *MockWorkoutReader) GetTodayWorkout(ctx context.Context, userID string) (*entity.Workout, error) {
	if m.GetTodayWorkoutFunc != nil {
		return m.GetTodayWorkoutFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockWorkoutReader) GetByID(ctx context.Context, workoutID string) (*entity.Workout, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, workoutID)
	}
	return nil, nil
}

func TestGetTodayWorkoutUseCase_Execute_Success(t *testing.T) {
	mockRepo := &MockWorkoutReader{
		GetTodayWorkoutFunc: func(ctx context.Context, userID string) (*entity.Workout, error) {
			return &entity.Workout{
				ID:          "workout-123",
				SessionName: "Upper A",
				Week:        1,
				DayName:     "Monday",
				Exercises: []entity.Exercise{
					{
						ID:          "exercise-1",
						Name:        "Bench Press",
						BadgeColor:  "#374151",
						RIR:         "2",
						RestSeconds: 180,
						Link:        "https://example.com/bench",
						Sets: []entity.Set{
							{ID: "set-1", SetNumber: 1, Reps: 8, Weight: "60", Completed: false},
							{ID: "set-2", SetNumber: 2, Reps: 8, Weight: "60", Completed: false},
						},
						Tips:  []entity.Tip{{Text: "Keep elbows tucked"}},
						Steps: []entity.Step{{Text: "Lie on bench"}},
					},
				},
			}, nil
		},
	}

	uc := NewGetTodayWorkoutUseCase(mockRepo)
	result, err := uc.Execute(context.Background(), "user-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.SessionID != "workout-123" {
		t.Errorf("Expected SessionID 'workout-123', got '%s'", result.SessionID)
	}

	if result.SessionName != "Upper A" {
		t.Errorf("Expected SessionName 'Upper A', got '%s'", result.SessionName)
	}

	if len(result.Exercises) != 1 {
		t.Errorf("Expected 1 exercise, got %d", len(result.Exercises))
	}

	if len(result.Exercises[0].Sets) != 2 {
		t.Errorf("Expected 2 sets, got %d", len(result.Exercises[0].Sets))
	}
}

func TestGetTodayWorkoutUseCase_Execute_EmptyUserID(t *testing.T) {
	mockRepo := &MockWorkoutReader{}
	uc := NewGetTodayWorkoutUseCase(mockRepo)

	result, err := uc.Execute(context.Background(), "")

	if err == nil {
		t.Fatal("Expected error for empty user_id")
	}

	if result != nil {
		t.Error("Expected nil result for error case")
	}
}

func TestGetTodayWorkoutUseCase_Execute_NoWorkout(t *testing.T) {
	mockRepo := &MockWorkoutReader{
		GetTodayWorkoutFunc: func(ctx context.Context, userID string) (*entity.Workout, error) {
			return nil, nil
		},
	}

	uc := NewGetTodayWorkoutUseCase(mockRepo)
	result, err := uc.Execute(context.Background(), "user-123")

	if err == nil {
		t.Fatal("Expected error when no workout found")
	}

	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestGetTodayWorkoutUseCase_Execute_RepoError(t *testing.T) {
	mockRepo := &MockWorkoutReader{
		GetTodayWorkoutFunc: func(ctx context.Context, userID string) (*entity.Workout, error) {
			return nil, errors.New("database error")
		},
	}

	uc := NewGetTodayWorkoutUseCase(mockRepo)
	result, err := uc.Execute(context.Background(), "user-123")

	if err == nil {
		t.Fatal("Expected error from repository")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestGetTodayWorkoutUseCase_Execute_EmptyExercises(t *testing.T) {
	mockRepo := &MockWorkoutReader{
		GetTodayWorkoutFunc: func(ctx context.Context, userID string) (*entity.Workout, error) {
			return &entity.Workout{
				ID:          "workout-123",
				SessionName: "Rest Day",
				Week:        1,
				DayName:     "Sunday",
				Exercises:   []entity.Exercise{},
			}, nil
		},
	}

	uc := NewGetTodayWorkoutUseCase(mockRepo)
	result, err := uc.Execute(context.Background(), "user-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result.Exercises) != 0 {
		t.Errorf("Expected 0 exercises, got %d", len(result.Exercises))
	}
}

func TestNewGetTodayWorkoutUseCase(t *testing.T) {
	mockRepo := &MockWorkoutReader{}
	uc := NewGetTodayWorkoutUseCase(mockRepo)

	if uc == nil {
		t.Error("Expected non-nil use case")
	}
}
