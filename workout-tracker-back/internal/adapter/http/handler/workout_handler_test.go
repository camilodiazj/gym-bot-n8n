package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/application/usecase"
	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// MockWorkoutReader for testing
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

// MockWorkoutRepository for complete workout testing
type MockWorkoutRepository struct {
	MockWorkoutReader
	MarkCompleteFunc func(ctx context.Context, workoutID string) error
}

func (m *MockWorkoutRepository) MarkComplete(ctx context.Context, workoutID string) error {
	if m.MarkCompleteFunc != nil {
		return m.MarkCompleteFunc(ctx, workoutID)
	}
	return nil
}

func TestWorkoutHandler_GetTodayWorkout_Success(t *testing.T) {
	mockRepo := &MockWorkoutReader{
		GetTodayWorkoutFunc: func(ctx context.Context, userID string) (*entity.Workout, error) {
			return &entity.Workout{
				ID:          "workout-123",
				SessionName: "Upper A",
				Week:        1,
				DayName:     "Monday",
				Exercises:   []entity.Exercise{},
			}, nil
		},
	}

	getTodayWorkoutUC := usecase.NewGetTodayWorkoutUseCase(mockRepo)
	handler := NewWorkoutHandler(getTodayWorkoutUC, nil)

	router := gin.New()
	router.GET("/workouts/today", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		handler.GetTodayWorkout(c)
	})

	req, _ := http.NewRequest("GET", "/workouts/today", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["success"] != true {
		t.Error("Expected success to be true")
	}
}

func TestWorkoutHandler_GetTodayWorkout_NoUserID(t *testing.T) {
	handler := NewWorkoutHandler(nil, nil)

	router := gin.New()
	router.GET("/workouts/today", handler.GetTodayWorkout)

	req, _ := http.NewRequest("GET", "/workouts/today", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestWorkoutHandler_GetTodayWorkout_NotFound(t *testing.T) {
	mockRepo := &MockWorkoutReader{
		GetTodayWorkoutFunc: func(ctx context.Context, userID string) (*entity.Workout, error) {
			return nil, nil
		},
	}

	getTodayWorkoutUC := usecase.NewGetTodayWorkoutUseCase(mockRepo)
	handler := NewWorkoutHandler(getTodayWorkoutUC, nil)

	router := gin.New()
	router.GET("/workouts/today", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		handler.GetTodayWorkout(c)
	})

	req, _ := http.NewRequest("GET", "/workouts/today", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestWorkoutHandler_CompleteWorkout_Success(t *testing.T) {
	mockRepo := &MockWorkoutRepository{
		MockWorkoutReader: MockWorkoutReader{
			GetByIDFunc: func(ctx context.Context, workoutID string) (*entity.Workout, error) {
				return &entity.Workout{ID: workoutID}, nil
			},
		},
		MarkCompleteFunc: func(ctx context.Context, workoutID string) error {
			return nil
		},
	}

	completeWorkoutUC := usecase.NewCompleteWorkoutUseCase(mockRepo)
	handler := NewWorkoutHandler(nil, completeWorkoutUC)

	router := gin.New()
	router.POST("/workouts/:workoutId/complete", handler.CompleteWorkout)

	req, _ := http.NewRequest("POST", "/workouts/workout-123/complete", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestWorkoutHandler_CompleteWorkout_NotFound(t *testing.T) {
	mockRepo := &MockWorkoutRepository{
		MarkCompleteFunc: func(ctx context.Context, workoutID string) error {
			return apperror.NewNotFoundError("workout not found")
		},
	}

	completeWorkoutUC := usecase.NewCompleteWorkoutUseCase(mockRepo)
	handler := NewWorkoutHandler(nil, completeWorkoutUC)

	router := gin.New()
	router.POST("/workouts/:workoutId/complete", handler.CompleteWorkout)

	req, _ := http.NewRequest("POST", "/workouts/invalid-id/complete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestNewWorkoutHandler(t *testing.T) {
	handler := NewWorkoutHandler(nil, nil)
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestWorkoutHandler_CompleteWorkout_WithRequestBody(t *testing.T) {
	mockRepo := &MockWorkoutRepository{
		MockWorkoutReader: MockWorkoutReader{
			GetByIDFunc: func(ctx context.Context, workoutID string) (*entity.Workout, error) {
				return &entity.Workout{ID: workoutID}, nil
			},
		},
		MarkCompleteFunc: func(ctx context.Context, workoutID string) error {
			return nil
		},
	}

	completeWorkoutUC := usecase.NewCompleteWorkoutUseCase(mockRepo)
	handler := NewWorkoutHandler(nil, completeWorkoutUC)

	router := gin.New()
	router.POST("/workouts/:workoutId/complete", handler.CompleteWorkout)

	body := `{"notes": "Great workout!"}`
	req, _ := http.NewRequest("POST", "/workouts/workout-123/complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestWorkoutHandler_GetTodayWorkout_WithExercises(t *testing.T) {
	mockRepo := &MockWorkoutReader{
		GetTodayWorkoutFunc: func(ctx context.Context, userID string) (*entity.Workout, error) {
			return &entity.Workout{
				ID:          "workout-123",
				SessionName: "Upper A",
				Week:        1,
				DayName:     "Monday",
				Exercises: []entity.Exercise{
					{
						ID:          "ex-1",
						Name:        "Bench Press",
						BadgeColor:  "#374151",
						RIR:         "2",
						RestSeconds: 180,
						Link:        "https://example.com",
						Sets: []entity.Set{
							{ID: "s1", SetNumber: 1, Reps: 8, Weight: "60", Completed: false},
						},
						Tips:  []entity.Tip{{Text: "Tip 1"}},
						Steps: []entity.Step{{Text: "Step 1"}},
					},
				},
			}, nil
		},
	}

	getTodayWorkoutUC := usecase.NewGetTodayWorkoutUseCase(mockRepo)
	handler := NewWorkoutHandler(getTodayWorkoutUC, nil)

	router := gin.New()
	router.GET("/workouts/today", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		handler.GetTodayWorkout(c)
	})

	req, _ := http.NewRequest("GET", "/workouts/today", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Success bool `json:"success"`
		Data    dto.TodayWorkoutResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)

	if len(response.Data.Exercises) != 1 {
		t.Errorf("Expected 1 exercise, got %d", len(response.Data.Exercises))
	}
}
