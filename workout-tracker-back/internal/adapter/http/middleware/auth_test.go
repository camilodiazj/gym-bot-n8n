package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestValidateAuth_WithValidCode(t *testing.T) {
	// Setup
	mockResolver := func(code string) (string, error) {
		if code == "VALID1" {
			return "user-123", nil
		}
		return "", errors.New("invalid code")
	}

	router := gin.New()
	router.Use(ValidateAuth(mockResolver))
	router.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not set"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	// Test valid code
	req, _ := http.NewRequest("GET", "/test?c=VALID1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestValidateAuth_WithInvalidCode(t *testing.T) {
	mockResolver := func(code string) (string, error) {
		return "", errors.New("invalid code")
	}

	router := gin.New()
	router.Use(ValidateAuth(mockResolver))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test?c=INVALID", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestValidateAuth_WithUserIDFallback(t *testing.T) {
	mockResolver := func(code string) (string, error) {
		return "", errors.New("not found")
	}

	router := gin.New()
	router.Use(ValidateAuth(mockResolver))
	router.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not set"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	req, _ := http.NewRequest("GET", "/test?user_id=dev-user-456", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestValidateAuth_NoAuth(t *testing.T) {
	mockResolver := func(code string) (string, error) {
		return "", errors.New("not found")
	}

	router := gin.New()
	router.Use(ValidateAuth(mockResolver))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestValidateAuth_NilResolver(t *testing.T) {
	router := gin.New()
	router.Use(ValidateAuth(nil))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	// With code but nil resolver, should fall through to user_id check
	req, _ := http.NewRequest("GET", "/test?c=CODE123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 401 because no user_id either
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestValidateAuth_CodePriorityOverUserID(t *testing.T) {
	mockResolver := func(code string) (string, error) {
		if code == "CODE1" {
			return "code-user", nil
		}
		return "", errors.New("invalid")
	}

	router := gin.New()
	router.Use(ValidateAuth(mockResolver))
	router.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	// Both code and user_id provided - code should take priority
	req, _ := http.NewRequest("GET", "/test?c=CODE1&user_id=other-user", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestValidateAuth_EmptyCode(t *testing.T) {
	mockResolver := func(code string) (string, error) {
		return "user", nil
	}

	router := gin.New()
	router.Use(ValidateAuth(mockResolver))
	router.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	// Empty code should fall through to user_id
	req, _ := http.NewRequest("GET", "/test?c=&user_id=fallback-user", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
