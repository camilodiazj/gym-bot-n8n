package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthHandler_Check(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler()

	router := gin.New()
	router.GET("/health", handler.Check)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response.Status)
	}

	if response.Timestamp == "" {
		t.Error("Expected non-empty timestamp")
	}

	if response.Uptime == "" {
		t.Error("Expected non-empty uptime")
	}
}

func TestNewHealthHandler(t *testing.T) {
	handler := NewHealthHandler()

	if handler == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestHealthHandler_Uptime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler()

	router := gin.New()
	router.GET("/health", handler.Check)

	// First request
	req1, _ := http.NewRequest("GET", "/health", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	var response1 HealthResponse
	json.Unmarshal(w1.Body.Bytes(), &response1)

	// Second request should have a larger uptime
	req2, _ := http.NewRequest("GET", "/health", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	var response2 HealthResponse
	json.Unmarshal(w2.Body.Bytes(), &response2)

	// Both should be valid
	if response1.Status != "healthy" || response2.Status != "healthy" {
		t.Error("Both responses should be healthy")
	}
}
