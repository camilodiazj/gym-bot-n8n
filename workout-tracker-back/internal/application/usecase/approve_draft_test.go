package usecase

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/application/port"
	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// ─────────────────────────── test doubles ───────────────────────────

// mockDraftRepo lets each test set the behavior of the methods the
// approve usecase exercises. Following the project convention: no mocking
// framework, just hand-rolled func fields.
type mockDraftRepo struct {
	GetByCodeFunc          func(ctx context.Context, code string) (*entity.DraftRoutine, error)
	GetByCodeAnyStatusFunc func(ctx context.Context, code string) (*entity.DraftRoutine, error)
	GetPhoneByUserIDFunc   func(ctx context.Context, userID string) (string, error)
	MarkApprovedFunc       func(ctx context.Context, code string) error
}

// Static interface assertion.
var _ repository.DraftRepository = (*mockDraftRepo)(nil)

func (m *mockDraftRepo) GetByCode(ctx context.Context, code string) (*entity.DraftRoutine, error) {
	return m.GetByCodeFunc(ctx, code)
}
func (m *mockDraftRepo) GetByCodeAnyStatus(ctx context.Context, code string) (*entity.DraftRoutine, error) {
	if m.GetByCodeAnyStatusFunc == nil {
		return nil, apperror.NewNotFoundError("draft not found")
	}
	return m.GetByCodeAnyStatusFunc(ctx, code)
}
func (m *mockDraftRepo) GetPhoneByUserID(ctx context.Context, userID string) (string, error) {
	return m.GetPhoneByUserIDFunc(ctx, userID)
}
func (m *mockDraftRepo) UpdateDraftData(ctx context.Context, code string, data entity.DraftData) error {
	return nil // unused here
}
func (m *mockDraftRepo) MarkApproved(ctx context.Context, code string) error {
	return m.MarkApprovedFunc(ctx, code)
}
func (m *mockDraftRepo) EnrichExercises(ctx context.Context, ids []string) (map[string]repository.ExerciseMeta, error) {
	return nil, nil // unused here
}

type mockKairosClient struct {
	FinalizeDraftFunc func(ctx context.Context, code string) (*port.FinalizeDraftResult, error)
	calls             int
}

var _ port.KairosClient = (*mockKairosClient)(nil)

func (m *mockKairosClient) FinalizeDraft(ctx context.Context, code string) (*port.FinalizeDraftResult, error) {
	m.calls++
	return m.FinalizeDraftFunc(ctx, code)
}

// fixtures
func pendingDraft() *entity.DraftRoutine {
	return &entity.DraftRoutine{
		DraftID: "draft-uuid",
		UserID:  "user-uuid",
		Code:    "abc123",
		Status:  "pending",
	}
}

// ─────────────────────────── tests ───────────────────────────

func TestApproveDraft_Success_FreshApproval(t *testing.T) {
	repo := &mockDraftRepo{
		GetByCodeFunc:        func(ctx context.Context, code string) (*entity.DraftRoutine, error) { return pendingDraft(), nil },
		GetPhoneByUserIDFunc: func(ctx context.Context, uid string) (string, error) { return "573500000000", nil },
		MarkApprovedFunc:     func(ctx context.Context, code string) error { return nil },
	}
	kairos := &mockKairosClient{
		FinalizeDraftFunc: func(ctx context.Context, code string) (*port.FinalizeDraftResult, error) {
			return &port.FinalizeDraftResult{PlanID: "plan-uuid", WorkoutsCreated: 28}, nil
		},
	}

	uc := NewApproveDraftUseCase(repo, kairos)
	resp, err := uc.Execute(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PlanID != "plan-uuid" {
		t.Errorf("PlanID: want plan-uuid, got %q", resp.PlanID)
	}
	if resp.WorkoutsCreated != 28 {
		t.Errorf("WorkoutsCreated: want 28, got %d", resp.WorkoutsCreated)
	}
	if resp.AlreadyApproved {
		t.Errorf("AlreadyApproved: want false, got true")
	}
	if resp.PhoneNumber != "573500000000" {
		t.Errorf("PhoneNumber: want 573500000000, got %q", resp.PhoneNumber)
	}
}

func TestApproveDraft_Success_AlreadyApproved_FromIdempotentRetry(t *testing.T) {
	// Kairos says workouts_created=0 → AlreadyExisted=true, usecase surfaces it.
	repo := &mockDraftRepo{
		GetByCodeFunc:        func(ctx context.Context, code string) (*entity.DraftRoutine, error) { return pendingDraft(), nil },
		GetPhoneByUserIDFunc: func(ctx context.Context, uid string) (string, error) { return "573500000000", nil },
		MarkApprovedFunc:     func(ctx context.Context, code string) error { return nil },
	}
	kairos := &mockKairosClient{
		FinalizeDraftFunc: func(ctx context.Context, code string) (*port.FinalizeDraftResult, error) {
			return &port.FinalizeDraftResult{PlanID: "plan-uuid", WorkoutsCreated: 0, AlreadyExisted: true}, nil
		},
	}
	uc := NewApproveDraftUseCase(repo, kairos)
	resp, err := uc.Execute(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.AlreadyApproved {
		t.Errorf("AlreadyApproved: want true, got false")
	}
}

func TestApproveDraft_Conflict_WhenDraftAlreadyApproved(t *testing.T) {
	// GetByCode returns 404 because status filter excludes 'approved' rows.
	// The usecase should look it up via AnyStatus and surface 409.
	repo := &mockDraftRepo{
		GetByCodeFunc: func(ctx context.Context, code string) (*entity.DraftRoutine, error) {
			return nil, apperror.NewNotFoundError("draft not found or expired")
		},
		GetByCodeAnyStatusFunc: func(ctx context.Context, code string) (*entity.DraftRoutine, error) {
			d := pendingDraft()
			d.Status = "approved"
			return d, nil
		},
	}
	kairos := &mockKairosClient{
		FinalizeDraftFunc: func(ctx context.Context, code string) (*port.FinalizeDraftResult, error) {
			t.Fatal("Kairos should not be called when draft is already approved")
			return nil, nil
		},
	}
	uc := NewApproveDraftUseCase(repo, kairos)
	_, err := uc.Execute(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *AppError, got %T", err)
	}
	if appErr.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", appErr.Code)
	}
	if kairos.calls != 0 {
		t.Errorf("Kairos should not be called, calls=%d", kairos.calls)
	}
}

func TestApproveDraft_NotFound_WhenDraftDoesNotExist(t *testing.T) {
	repo := &mockDraftRepo{
		GetByCodeFunc: func(ctx context.Context, code string) (*entity.DraftRoutine, error) {
			return nil, apperror.NewNotFoundError("draft not found or expired")
		},
		GetByCodeAnyStatusFunc: func(ctx context.Context, code string) (*entity.DraftRoutine, error) {
			return nil, apperror.NewNotFoundError("draft not found")
		},
	}
	uc := NewApproveDraftUseCase(repo, &mockKairosClient{})
	_, err := uc.Execute(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error")
	}
	if !apperror.IsNotFound(err) {
		t.Errorf("expected 404, got %v", err)
	}
}

func TestApproveDraft_KairosTransientFailure_DoesNotMarkApproved(t *testing.T) {
	markApprovedCalled := false
	repo := &mockDraftRepo{
		GetByCodeFunc:        func(ctx context.Context, code string) (*entity.DraftRoutine, error) { return pendingDraft(), nil },
		GetPhoneByUserIDFunc: func(ctx context.Context, uid string) (string, error) { return "573500000000", nil },
		MarkApprovedFunc: func(ctx context.Context, code string) error {
			markApprovedCalled = true
			return nil
		},
	}
	kairos := &mockKairosClient{
		FinalizeDraftFunc: func(ctx context.Context, code string) (*port.FinalizeDraftResult, error) {
			return nil, apperror.NewInternalError("Kairos upstream error (status 503)", nil)
		},
	}
	uc := NewApproveDraftUseCase(repo, kairos)
	_, err := uc.Execute(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected error from Kairos failure")
	}
	if markApprovedCalled {
		t.Error("MarkApproved should NOT be called when Kairos fails — draft must stay pending for retry")
	}
}

func TestApproveDraft_KairosValidationFailure_DoesNotMarkApproved(t *testing.T) {
	markApprovedCalled := false
	repo := &mockDraftRepo{
		GetByCodeFunc:        func(ctx context.Context, code string) (*entity.DraftRoutine, error) { return pendingDraft(), nil },
		GetPhoneByUserIDFunc: func(ctx context.Context, uid string) (string, error) { return "573500000000", nil },
		MarkApprovedFunc: func(ctx context.Context, code string) error {
			markApprovedCalled = true
			return nil
		},
	}
	kairos := &mockKairosClient{
		FinalizeDraftFunc: func(ctx context.Context, code string) (*port.FinalizeDraftResult, error) {
			return nil, apperror.NewValidationError("draft missing required field 'pattern'")
		},
	}
	uc := NewApproveDraftUseCase(repo, kairos)
	_, err := uc.Execute(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !apperror.IsValidation(err) {
		t.Errorf("expected 400 validation, got %v", err)
	}
	if markApprovedCalled {
		t.Error("MarkApproved should NOT be called on validation failure")
	}
}

func TestApproveDraft_KairosSucceedsButMarkApprovedFails_StillReturns200(t *testing.T) {
	// Workouts are already created on Kairos side — we should NOT bubble the
	// MarkApproved error up as a 5xx. The user-visible result is success.
	repo := &mockDraftRepo{
		GetByCodeFunc:        func(ctx context.Context, code string) (*entity.DraftRoutine, error) { return pendingDraft(), nil },
		GetPhoneByUserIDFunc: func(ctx context.Context, uid string) (string, error) { return "573500000000", nil },
		MarkApprovedFunc: func(ctx context.Context, code string) error {
			return errors.New("network blip writing to Postgres")
		},
	}
	kairos := &mockKairosClient{
		FinalizeDraftFunc: func(ctx context.Context, code string) (*port.FinalizeDraftResult, error) {
			return &port.FinalizeDraftResult{PlanID: "plan-uuid", WorkoutsCreated: 28}, nil
		},
	}
	uc := NewApproveDraftUseCase(repo, kairos)
	resp, err := uc.Execute(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.WorkoutsCreated != 28 {
		t.Errorf("expected to surface workouts_created=28 even when MarkApproved failed, got %d", resp.WorkoutsCreated)
	}
}

func TestApproveDraft_EmptyCode_ReturnsValidationError(t *testing.T) {
	uc := NewApproveDraftUseCase(&mockDraftRepo{}, &mockKairosClient{})
	_, err := uc.Execute(context.Background(), "")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !apperror.IsValidation(err) {
		t.Errorf("expected validation error, got %v", err)
	}
}
