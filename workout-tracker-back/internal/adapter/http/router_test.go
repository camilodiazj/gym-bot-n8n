package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/internal/adapter/http/handler"
	"github.com/gymbot/workout-tracker-back/internal/application/usecase"
)

// MockWorkoutRepository for testing
type MockWorkoutRepository struct{}

func (m *MockWorkoutRepository) GetTodayByUserID(ctx context.Context, userID string) (*struct{}, error) {
	return nil, nil
}

func (m *MockWorkoutRepository) GetByID(ctx context.Context, workoutID string) (*struct{}, error) {
	return nil, nil
}

func (m *MockWorkoutRepository) MarkComplete(ctx context.Context, workoutID string) error {
	return nil
}

// MockSetRepository for testing
type MockSetRepository struct{}

func (m *MockSetRepository) GetByID(ctx context.Context, setID string) (*struct{}, error) {
	return nil, nil
}

func (m *MockSetRepository) Update(ctx context.Context, set interface{}) error {
	return nil
}

func (m *MockSetRepository) MarkComplete(ctx context.Context, setID string) error {
	return nil
}

// MockMagicLinkRepository for testing
type MockMagicLinkRepository struct{}

func (m *MockMagicLinkRepository) GetUserIDByCode(ctx context.Context, code string) (string, error) {
	if code == "VALID123" {
		return "user-123", nil
	}
	return "", errors.New("not found")
}

func (m *MockMagicLinkRepository) InvalidateByUser(ctx context.Context, userID string) error {
	return nil
}

func TestNewRouter(t *testing.T) {
	healthHandler := handler.NewHealthHandler()
	mockMagicLink := &MockMagicLinkRepository{}
	workoutHandler := handler.NewWorkoutHandler(
		usecase.NewGetTodayWorkoutUseCase(nil),
		usecase.NewCompleteWorkoutUseCase(nil, mockMagicLink),
	)
	setHandler := handler.NewSetHandler(
		usecase.NewMarkSetCompleteUseCase(nil),
		usecase.NewUpdateSetUseCase(nil),
	)

	codeResolver := func(code string) (string, error) {
		return "user-123", nil
	}

	corsOrigins := []string{"http://localhost:5173"}
	router := NewRouter(healthHandler, workoutHandler, setHandler, nil, codeResolver, corsOrigins)

	if router.healthHandler != healthHandler {
		t.Error("Expected healthHandler to be set")
	}
	if router.workoutHandler != workoutHandler {
		t.Error("Expected workoutHandler to be set")
	}
	if router.setHandler != setHandler {
		t.Error("Expected setHandler to be set")
	}
	if router.codeResolver == nil {
		t.Error("Expected codeResolver to be set")
	}
	if len(router.corsAllowedOrigins) == 0 {
		t.Error("Expected corsAllowedOrigins to be set")
	}
}

func TestRouter_Setup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	healthHandler := handler.NewHealthHandler()
	workoutHandler := handler.NewWorkoutHandler(nil, nil)
	setHandler := handler.NewSetHandler(nil, nil)

	codeResolver := func(code string) (string, error) {
		return "user-123", nil
	}

	router := NewRouter(healthHandler, workoutHandler, setHandler, nil, codeResolver, []string{"*"})
	engine := router.Setup(gin.TestMode)

	if engine == nil {
		t.Fatal("Expected engine to be created")
	}

	// Test health endpoint
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for health check, got %d", w.Code)
	}
}

func TestRouter_Setup_ProtectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	healthHandler := handler.NewHealthHandler()
	workoutHandler := handler.NewWorkoutHandler(nil, nil)
	setHandler := handler.NewSetHandler(nil, nil)

	codeResolver := func(code string) (string, error) {
		return "user-123", nil
	}

	router := NewRouter(healthHandler, workoutHandler, setHandler, nil, codeResolver, []string{"*"})
	engine := router.Setup(gin.TestMode)

	// Test protected route without auth
	req, _ := http.NewRequest("GET", "/api/v1/workouts/today", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for protected route without auth, got %d", w.Code)
	}

	// Test protected route with auth
	req, _ = http.NewRequest("GET", "/api/v1/workouts/today?c=ABC123", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// Should pass auth (may fail due to nil usecase, but shouldn't be 401)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("Expected auth to pass with valid code, got %d", w.Code)
	}
}

func TestRouter_Setup_CORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	healthHandler := handler.NewHealthHandler()
	workoutHandler := handler.NewWorkoutHandler(nil, nil)
	setHandler := handler.NewSetHandler(nil, nil)

	router := NewRouter(healthHandler, workoutHandler, setHandler, nil, nil, []string{"*"})
	engine := router.Setup(gin.TestMode)

	// Test OPTIONS request (CORS preflight)
	req, _ := http.NewRequest("OPTIONS", "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for OPTIONS request, got %d", w.Code)
	}

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin header")
	}
}
