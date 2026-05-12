package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// buildTestRouter wires ValidateAuth in front of a probe endpoint that
// surfaces what the middleware put on the context, so assertions can be made
// on user_id + auth_method without re-implementing reflection in every test.
func buildTestRouter(resolver CodeResolver, allowDev bool) *gin.Engine {
	r := gin.New()
	r.Use(ValidateAuth(resolver, allowDev))
	r.GET("/probe", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		authMethod, _ := c.Get("auth_method")
		c.JSON(http.StatusOK, gin.H{
			"user_id":     userID,
			"auth_method": authMethod,
		})
	})
	return r
}

func TestShortCode_PassesThrough(t *testing.T) {
	resolver := func(code string) (string, error) {
		if code == "VALID1" {
			return "user-123", nil
		}
		return "", errors.New("invalid code")
	}

	r := buildTestRouter(resolver, false)
	req, _ := http.NewRequest("GET", "/probe?c=VALID1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"user_id":"user-123"`) {
		t.Errorf("body missing user_id: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"auth_method":"short_code"`) {
		t.Errorf("body missing auth_method=short_code: %s", w.Body.String())
	}
}

func TestUserIDFallback_DeniedByDefault(t *testing.T) {
	// allowDev=false (production default). ?user_id= must NOT authenticate.
	r := buildTestRouter(nil, false)
	req, _ := http.NewRequest("GET", "/probe?user_id=any-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401 — ?user_id= must be closed by default. body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "authentication required") {
		t.Errorf("expected 'authentication required' in body, got: %s", w.Body.String())
	}
}

func TestUserIDFallback_AllowedWhenFlagSet(t *testing.T) {
	r := buildTestRouter(nil, true)
	req, _ := http.NewRequest("GET", "/probe?user_id=dev-user-456", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"user_id":"dev-user-456"`) {
		t.Errorf("body missing user_id: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"auth_method":"dev_user_id_fallback"`) {
		t.Errorf("body missing auth_method=dev_user_id_fallback: %s", w.Body.String())
	}
}

func TestNoAuth_Returns401(t *testing.T) {
	r := buildTestRouter(nil, true) // even with flag on, missing both creds = 401
	req, _ := http.NewRequest("GET", "/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", w.Code)
	}
}

func TestShortCodePriority(t *testing.T) {
	// With both ?c= and ?user_id= present (and allowDev=true), short code wins.
	resolver := func(code string) (string, error) {
		if code == "CODE1" {
			return "code-user", nil
		}
		return "", errors.New("invalid")
	}

	r := buildTestRouter(resolver, true)
	req, _ := http.NewRequest("GET", "/probe?c=CODE1&user_id=other-user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"user_id":"code-user"`) {
		t.Errorf("short_code should win over user_id; got body: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "dev_user_id_fallback") {
		t.Errorf("expected auth_method=short_code, but fallback path was taken: %s", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Regression coverage for edge cases that used to live in the old test file.
// ---------------------------------------------------------------------------

func TestShortCode_InvalidCodeReturns401(t *testing.T) {
	resolver := func(code string) (string, error) { return "", errors.New("expired") }
	r := buildTestRouter(resolver, true) // flag on shouldn't matter — code path errored
	req, _ := http.NewRequest("GET", "/probe?c=BADCODE", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid or expired code") {
		t.Errorf("expected 'invalid or expired code', got: %s", w.Body.String())
	}
}

func TestEmptyCode_DoesNotFallThroughWhenDevDisabled(t *testing.T) {
	// ?c=&user_id=x with allowDev=false → 401. (Previously this would have
	// fallen through to the ?user_id= branch and returned 200, the very
	// behavior that motivated this hardening.)
	resolver := func(code string) (string, error) { return "u", nil }
	r := buildTestRouter(resolver, false)
	req, _ := http.NewRequest("GET", "/probe?c=&user_id=fallback-user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401 — empty ?c= must not fall through to ?user_id= in prod", w.Code)
	}
}
