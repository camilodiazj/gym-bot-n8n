// Package client contains outbound HTTP adapters that implement the ports
// declared in internal/application/port.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/application/port"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// HTTPKairosClient is the HTTP implementation of port.KairosClient. It targets
// the Kairos agent's REST API. No retry/backoff logic lives here — the caller
// decides whether and how to retry based on the typed error returned.
type HTTPKairosClient struct {
	baseURL    string
	httpClient *http.Client
}

// Compile-time assertion: HTTPKairosClient satisfies the port.
var _ port.KairosClient = (*HTTPKairosClient)(nil)

// NewKairosClient builds an HTTP client targeting baseURL with the given
// per-request timeout.
func NewKairosClient(baseURL string, timeout time.Duration) *HTTPKairosClient {
	return &HTTPKairosClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// kairosFinalizeResponse mirrors the JSON envelope returned by
// POST /api/v1/drafts/{code}/finalize.
type kairosFinalizeResponse struct {
	Success         bool   `json:"success"`
	PlanID          string `json:"plan_id"`
	WorkoutsCreated int    `json:"workouts_created"`
	UserID          string `json:"user_id"`
	Error           string `json:"error,omitempty"`
}

// FinalizeDraft POSTs to Kairos's /finalize endpoint and translates HTTP
// outcomes into typed application errors.
func (c *HTTPKairosClient) FinalizeDraft(ctx context.Context, code string) (*port.FinalizeDraftResult, error) {
	url := fmt.Sprintf("%s/api/v1/drafts/%s/finalize", c.baseURL, code)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return nil, apperror.NewInternalError("failed to build Kairos finalize request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network-level failure: timeout, DNS, connection refused. Always retry-able.
		return nil, apperror.NewInternalError("failed to call Kairos finalize endpoint", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode == http.StatusOK:
		var parsed kairosFinalizeResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, apperror.NewInternalError("failed to decode Kairos finalize response", err)
		}
		if !parsed.Success {
			return nil, apperror.NewInternalError(
				fmt.Sprintf("Kairos returned success=false: %s", parsed.Error), nil,
			)
		}
		return &port.FinalizeDraftResult{
			PlanID:          parsed.PlanID,
			WorkoutsCreated: parsed.WorkoutsCreated,
			AlreadyExisted:  parsed.WorkoutsCreated == 0,
		}, nil

	case resp.StatusCode == http.StatusNotFound:
		return nil, apperror.NewNotFoundError(extractError(body, "draft not found in Kairos"))

	case resp.StatusCode == http.StatusUnprocessableEntity:
		return nil, apperror.NewValidationError(extractError(body, "Kairos rejected draft content"))

	case resp.StatusCode >= 500:
		return nil, apperror.NewInternalError(
			fmt.Sprintf("Kairos upstream error (status %d): %s", resp.StatusCode, extractError(body, "no detail")),
			nil,
		)

	default:
		return nil, apperror.NewInternalError(
			fmt.Sprintf("unexpected Kairos status %d: %s", resp.StatusCode, string(body)),
			nil,
		)
	}
}

// extractError pulls a human-readable error string out of a Kairos error body
// or falls back to the provided default.
func extractError(body []byte, fallback string) string {
	var parsed kairosFinalizeResponse
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error != "" {
		return parsed.Error
	}
	return fallback
}
