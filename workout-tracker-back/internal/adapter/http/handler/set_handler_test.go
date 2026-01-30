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
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// MockSetWriter for testing
type MockSetWriter struct {
	MarkCompleteFunc func(ctx context.Context, setID string) error
	UpdateFunc       func(ctx context.Context, setID string, reps *int, weight *string) error
}

func (m *MockSetWriter) MarkComplete(ctx context.Context, setID string) error {
	if m.MarkCompleteFunc != nil {
		return m.MarkCompleteFunc(ctx, setID)
	}
	return nil
}

func (m *MockSetWriter) Update(ctx context.Context, setID string, reps *int, weight *string) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, setID, reps, weight)
	}
	return nil
}

func TestSetHandler_MarkComplete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockSetWriter{
		MarkCompleteFunc: func(ctx context.Context, setID string) error {
			return nil
		},
	}

	markSetCompleteUC := usecase.NewMarkSetCompleteUseCase(mockRepo)
	handler := NewSetHandler(markSetCompleteUC, nil)

	router := gin.New()
	router.PATCH("/sets/:setId/complete", handler.MarkComplete)

	req, _ := http.NewRequest("PATCH", "/sets/set-123/complete", nil)
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

func TestSetHandler_MarkComplete_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockSetWriter{
		MarkCompleteFunc: func(ctx context.Context, setID string) error {
			return apperror.NewNotFoundError("set not found")
		},
	}

	markSetCompleteUC := usecase.NewMarkSetCompleteUseCase(mockRepo)
	handler := NewSetHandler(markSetCompleteUC, nil)

	router := gin.New()
	router.PATCH("/sets/:setId/complete", handler.MarkComplete)

	req, _ := http.NewRequest("PATCH", "/sets/invalid-id/complete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestSetHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockSetWriter{
		UpdateFunc: func(ctx context.Context, setID string, reps *int, weight *string) error {
			return nil
		},
	}

	updateSetUC := usecase.NewUpdateSetUseCase(mockRepo)
	handler := NewSetHandler(nil, updateSetUC)

	router := gin.New()
	router.PATCH("/sets/:setId", handler.Update)

	body := `{"reps": 10, "kg": "65"}`
	req, _ := http.NewRequest("PATCH", "/sets/set-123", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestSetHandler_Update_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSetHandler(nil, nil)

	router := gin.New()
	router.PATCH("/sets/:setId", handler.Update)

	body := `invalid json`
	req, _ := http.NewRequest("PATCH", "/sets/set-123", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestSetHandler_Update_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockSetWriter{
		UpdateFunc: func(ctx context.Context, setID string, reps *int, weight *string) error {
			return apperror.NewNotFoundError("set not found")
		},
	}

	updateSetUC := usecase.NewUpdateSetUseCase(mockRepo)
	handler := NewSetHandler(nil, updateSetUC)

	router := gin.New()
	router.PATCH("/sets/:setId", handler.Update)

	body := `{"reps": 10}`
	req, _ := http.NewRequest("PATCH", "/sets/invalid-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestNewSetHandler(t *testing.T) {
	handler := NewSetHandler(nil, nil)
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestSetHandler_Update_PartialUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockSetWriter{
		UpdateFunc: func(ctx context.Context, setID string, reps *int, weight *string) error {
			return nil
		},
	}

	updateSetUC := usecase.NewUpdateSetUseCase(mockRepo)
	handler := NewSetHandler(nil, updateSetUC)

	router := gin.New()
	router.PATCH("/sets/:setId", handler.Update)

	// Only update reps
	body := `{"reps": 12}`
	req, _ := http.NewRequest("PATCH", "/sets/set-123", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Success bool                   `json:"success"`
		Data    dto.UpdateSetResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Success {
		t.Error("Expected success to be true")
	}
}
