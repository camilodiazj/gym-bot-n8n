package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// newTestServer spins up a tiny HTTP server that mirrors what Kairos's
// /finalize endpoint will look like. Each test plugs in its own handler.
func newTestServer(handler http.HandlerFunc) (*httptest.Server, *HTTPKairosClient) {
	srv := httptest.NewServer(handler)
	client := NewKairosClient(srv.URL, 5*time.Second)
	return srv, client
}

func TestHTTPKairosClient_Success(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/api/v1/drafts/abc123/finalize") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"plan_id":"plan-uuid","workouts_created":28,"user_id":"user-uuid"}`))
	})
	defer srv.Close()

	res, err := client.FinalizeDraft(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PlanID != "plan-uuid" {
		t.Errorf("PlanID: want plan-uuid, got %q", res.PlanID)
	}
	if res.WorkoutsCreated != 28 {
		t.Errorf("WorkoutsCreated: want 28, got %d", res.WorkoutsCreated)
	}
	if res.AlreadyExisted {
		t.Error("AlreadyExisted should be false when workouts_created>0")
	}
}

func TestHTTPKairosClient_IdempotentReturn(t *testing.T) {
	// workouts_created=0 indicates Kairos detected an existing plan.
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"plan_id":"plan-uuid","workouts_created":0,"user_id":"user-uuid"}`))
	})
	defer srv.Close()

	res, err := client.FinalizeDraft(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.AlreadyExisted {
		t.Error("AlreadyExisted should be true when workouts_created==0")
	}
}

func TestHTTPKairosClient_404_MapsToNotFound(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"error":"draft not found"}`))
	})
	defer srv.Close()

	_, err := client.FinalizeDraft(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error")
	}
	if !apperror.IsNotFound(err) {
		t.Errorf("expected NotFound, got %v", err)
	}
}

func TestHTTPKairosClient_422_MapsToValidation(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"success":false,"error":"validation: missing pattern"}`))
	})
	defer srv.Close()

	_, err := client.FinalizeDraft(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !apperror.IsValidation(err) {
		t.Errorf("expected Validation, got %v", err)
	}
}

func TestHTTPKairosClient_500_MapsToInternalError(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"success":false,"error":"db timeout"}`))
	})
	defer srv.Close()

	_, err := client.FinalizeDraft(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected internal error")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *AppError, got %T", err)
	}
	if appErr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", appErr.Code)
	}
}

func TestHTTPKairosClient_SuccessFalseInPayload_TreatedAsInternalError(t *testing.T) {
	// HTTP 200 but the JSON envelope says success=false.
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":false,"error":"unexpected condition"}`))
	})
	defer srv.Close()

	_, err := client.FinalizeDraft(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unexpected condition") {
		t.Errorf("error message should include upstream detail, got %v", err)
	}
}

func TestHTTPKairosClient_NetworkFailure_MapsToInternalError(t *testing.T) {
	// Point client at a dead address.
	client := NewKairosClient("http://127.0.0.1:1", 100*time.Millisecond)
	_, err := client.FinalizeDraft(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected network error")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *AppError, got %T", err)
	}
	if appErr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for network failure, got %d", appErr.Code)
	}
}
