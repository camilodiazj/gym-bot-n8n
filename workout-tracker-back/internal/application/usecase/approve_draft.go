package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// ApproveDraftUseCase handles the business logic for approving a draft routine
type ApproveDraftUseCase struct {
	draftRepo    repository.DraftRepository
	kairosAPIURL string
	httpClient   *http.Client
}

// NewApproveDraftUseCase creates a new ApproveDraftUseCase
func NewApproveDraftUseCase(draftRepo repository.DraftRepository, kairosAPIURL string) *ApproveDraftUseCase {
	return &ApproveDraftUseCase{
		draftRepo:    draftRepo,
		kairosAPIURL: kairosAPIURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// kairosRequest is the payload sent to the Kairos agent API
type kairosRequest struct {
	PhoneNumber  string `json:"phone_number"`
	DisplayName  string `json:"display_name"`
	Message      string `json:"message"`
	SendWhatsApp bool   `json:"send_whatsapp"`
}

// Execute validates the draft, calls Kairos to finalize, and marks as approved
func (uc *ApproveDraftUseCase) Execute(ctx context.Context, code string) (*dto.ApproveDraftResponse, error) {
	if code == "" {
		return nil, apperror.NewValidationError("code is required")
	}

	// Validate draft exists and is pending
	draft, err := uc.draftRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Get user's phone number
	phone, err := uc.draftRepo.GetPhoneByUserID(ctx, draft.UserID)
	if err != nil {
		return nil, err
	}

	// Call Kairos API to trigger routine approval
	if err := uc.callKairosApproval(ctx, phone); err != nil {
		return nil, err
	}

	// Mark draft as approved
	if err := uc.draftRepo.MarkApproved(ctx, code); err != nil {
		return nil, err
	}

	return &dto.ApproveDraftResponse{
		PhoneNumber: phone,
	}, nil
}

// callKairosApproval sends the approval message to the Kairos agent
func (uc *ApproveDraftUseCase) callKairosApproval(ctx context.Context, phoneNumber string) error {
	payload := kairosRequest{
		PhoneNumber:  phoneNumber,
		DisplayName:  "User",
		Message:      "Apruebo la rutina",
		SendWhatsApp: true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return apperror.NewInternalError("failed to marshal Kairos request", err)
	}

	url := fmt.Sprintf("%s/api/v1/chat", uc.kairosAPIURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return apperror.NewInternalError("failed to create Kairos request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return apperror.NewInternalError("failed to call Kairos API", err)
	}
	defer resp.Body.Close()

	// Drain the response body to allow connection reuse
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apperror.NewInternalError(
			fmt.Sprintf("Kairos API returned status %d", resp.StatusCode), nil,
		)
	}

	return nil
}
