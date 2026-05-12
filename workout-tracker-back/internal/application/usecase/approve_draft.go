package usecase

import (
	"context"
	"log"
	"net/http"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/application/port"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// ApproveDraftUseCase finalizes a draft routine: it asks Kairos to load the
// persisted draft_data JSONB and materialize it into users_plans + workouts,
// then marks the draft as approved.
//
// The key invariant: the workouts created reflect exactly what the user saw
// in the preview (including any swaps), because Kairos reads from
// draft_routines.draft_data — not from an LLM regeneration.
type ApproveDraftUseCase struct {
	draftRepo    repository.DraftRepository
	kairosClient port.KairosClient
}

// NewApproveDraftUseCase wires the usecase against its dependencies.
// The kairosClient is a port interface so tests can substitute a fake.
func NewApproveDraftUseCase(draftRepo repository.DraftRepository, kairosClient port.KairosClient) *ApproveDraftUseCase {
	return &ApproveDraftUseCase{
		draftRepo:    draftRepo,
		kairosClient: kairosClient,
	}
}

// Execute drives the approve flow.
//
// Ordering: Kairos first, MarkApproved after. This relies on Kairos's
// FinalizeDraft being idempotent (see port.KairosClient docs). Rationale:
//
//   - If Kairos fails, the draft stays pending and the user can retry without
//     leaving an "approved but unmaterialized" zombie state.
//   - If Kairos succeeds and MarkApproved fails, the workouts exist, so we
//     respond 200 and log a WARN — manual recovery is to set status='approved'.
//
// Double-click / true retry semantics:
//
//   - If the draft is already 'approved', GetByCode (which filters status='pending')
//     returns 404. We then look it up via GetByCodeAnyStatus; if present and
//     status='approved', we respond 409 Conflict so the caller can treat it as
//     "already done" rather than as a hard failure.
func (uc *ApproveDraftUseCase) Execute(ctx context.Context, code string) (*dto.ApproveDraftResponse, error) {
	if code == "" {
		return nil, apperror.NewValidationError("code is required")
	}

	draft, err := uc.draftRepo.GetByCode(ctx, code)
	if err != nil {
		if apperror.IsNotFound(err) {
			// Either the draft never existed, expired, or is already approved.
			// Disambiguate so the client gets a useful status code.
			return nil, uc.classifyMissingDraft(ctx, code)
		}
		return nil, err
	}

	phone, err := uc.draftRepo.GetPhoneByUserID(ctx, draft.UserID)
	if err != nil {
		return nil, err
	}

	result, err := uc.kairosClient.FinalizeDraft(ctx, code)
	if err != nil {
		// Typed error from the adapter (404 / 422 / 500) bubbles up untouched.
		// Draft stays in 'pending' so the next retry hits a clean state.
		return nil, err
	}

	if markErr := uc.draftRepo.MarkApproved(ctx, code); markErr != nil {
		// Workouts already exist in DB — losing the status flip is annoying
		// but not data-corrupting. Log loudly and proceed with a 200.
		log.Printf("WARN approve_draft: kairos finalize succeeded but MarkApproved failed: code=%s user_id=%s plan_id=%s err=%v",
			code, draft.UserID, result.PlanID, markErr)
	}

	return &dto.ApproveDraftResponse{
		PhoneNumber:     phone,
		PlanID:          result.PlanID,
		WorkoutsCreated: result.WorkoutsCreated,
		AlreadyApproved: result.AlreadyExisted,
		MagicLinkCode:   result.MagicLinkCode,
	}, nil
}

// classifyMissingDraft converts a GetByCode 404 into the most accurate status
// code: 404 if the draft truly doesn't exist, 409 if it's already approved,
// and the original error for any other state.
func (uc *ApproveDraftUseCase) classifyMissingDraft(ctx context.Context, code string) error {
	draft, lookupErr := uc.draftRepo.GetByCodeAnyStatus(ctx, code)
	if lookupErr != nil {
		// 404 from this method = draft truly doesn't exist. Propagate.
		return lookupErr
	}
	if draft.Status == "approved" {
		return &apperror.AppError{
			Code:    http.StatusConflict,
			Message: "draft already approved",
		}
	}
	// Some other status (expired, etc.). Treat as gone.
	return apperror.NewNotFoundError("draft not found or expired")
}
