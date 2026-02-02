package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestValidateInternalAPIKey_ValidKey(t *testing.T) {
	apiKey := "test-api-key-123"

	router := gin.New()
	router.Use(ValidateInternalAPIKey(apiKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", apiKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestValidateInternalAPIKey_ValidKeyWithBearer(t *testing.T) {
	apiKey := "test-api-key-123"

	router := gin.New()
	router.Use(ValidateInternalAPIKey(apiKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestValidateInternalAPIKey_InvalidKey(t *testing.T) {
	apiKey := "test-api-key-123"

	router := gin.New()
	router.Use(ValidateInternalAPIKey(apiKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestValidateInternalAPIKey_NoKey(t *testing.T) {
	apiKey := "test-api-key-123"

	router := gin.New()
	router.Use(ValidateInternalAPIKey(apiKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestValidateInternalAPIKey_EmptyKey(t *testing.T) {
	apiKey := "test-api-key-123"

	router := gin.New()
	router.Use(ValidateInternalAPIKey(apiKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestValidateInternalAPIKey_AuthorizationWithoutBearer(t *testing.T) {
	apiKey := "test-api-key-123"

	router := gin.New()
	router.Use(ValidateInternalAPIKey(apiKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	// Authorization header without Bearer prefix - should use the value directly
	req.Header.Set("Authorization", apiKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestValidateInternalAPIKey_ShortBearer(t *testing.T) {
	apiKey := "abc"

	router := gin.New()
	router.Use(ValidateInternalAPIKey(apiKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	// Authorization header shorter than "Bearer " - should use the value directly
	req.Header.Set("Authorization", "abc")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestValidateInternalAPIKey_XAPIKeyPriority(t *testing.T) {
	apiKey := "test-api-key-123"

	router := gin.New()
	router.Use(ValidateInternalAPIKey(apiKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	// Both headers set, but X-API-Key takes priority
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 (X-API-Key takes priority), got %d", w.Code)
	}
}
