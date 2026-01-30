package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_RegularRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS([]string{"*"}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected Access-Control-Allow-Origin header")
	}

	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Expected Access-Control-Allow-Credentials header")
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}

	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("Expected Access-Control-Allow-Headers header")
	}
}

func TestCORS_OptionsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS([]string{"*"}))
	router.OPTIONS("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for OPTIONS, got %d", w.Code)
	}

	// Check CORS headers are set
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected Access-Control-Allow-Origin header on OPTIONS")
	}
}

func TestCORS_POSTRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS([]string{"*"}))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"success": true})
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Verify CORS headers are present
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Expected CORS headers on POST request")
	}
}
