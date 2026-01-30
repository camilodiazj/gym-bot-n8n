package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSuccess(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Success(c, map[string]string{"key": "value"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Success {
		t.Error("Expected success to be true")
	}

	if response.Error != nil {
		t.Error("Expected no error in success response")
	}
}

func TestCreated(t *testing.T) {
	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		Created(c, map[string]string{"id": "123"})
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Success {
		t.Error("Expected success to be true")
	}
}

func TestNoContent(t *testing.T) {
	router := gin.New()
	router.DELETE("/test", func(c *gin.Context) {
		NoContent(c)
	})

	req, _ := http.NewRequest("DELETE", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

func TestError_AppError(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Error(c, apperror.NewNotFoundError("resource not found"))
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error == nil {
		t.Fatal("Expected error in response")
	}

	if response.Error.Message != "resource not found" {
		t.Errorf("Expected message 'resource not found', got '%s'", response.Error.Message)
	}
}

func TestError_RegularError(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Error(c, errors.New("something went wrong"))
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Success {
		t.Error("Expected success to be false")
	}
}

func TestBadRequest(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		BadRequest(c, "invalid input")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error.Message != "invalid input" {
		t.Errorf("Expected message 'invalid input', got '%s'", response.Error.Message)
	}
}

func TestNotFound(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		NotFound(c, "resource not found")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Unauthorized(c, "not authorized")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestInternalServerError(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		InternalServerError(c, "something went wrong")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}
