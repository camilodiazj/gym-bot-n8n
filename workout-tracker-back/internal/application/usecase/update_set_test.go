package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
)

func TestUpdateSetUseCase_Execute_Success(t *testing.T) {
	mockRepo := &MockSetWriter{
		UpdateFunc: func(ctx context.Context, setID string, reps *int, weight *string) error {
			return nil
		},
	}

	uc := NewUpdateSetUseCase(mockRepo)
	reps := 10
	kg := "65"
	req := &dto.UpdateSetRequest{Reps: &reps, Kg: &kg}

	result, err := uc.Execute(context.Background(), "set-123", req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.SetID != "set-123" {
		t.Errorf("Expected SetID 'set-123', got '%s'", result.SetID)
	}
}

func TestUpdateSetUseCase_Execute_OnlyReps(t *testing.T) {
	mockRepo := &MockSetWriter{
		UpdateFunc: func(ctx context.Context, setID string, reps *int, weight *string) error {
			if reps == nil {
				t.Error("Expected reps to be provided")
			}
			return nil
		},
	}

	uc := NewUpdateSetUseCase(mockRepo)
	reps := 12
	req := &dto.UpdateSetRequest{Reps: &reps}

	result, err := uc.Execute(context.Background(), "set-123", req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestUpdateSetUseCase_Execute_OnlyWeight(t *testing.T) {
	mockRepo := &MockSetWriter{
		UpdateFunc: func(ctx context.Context, setID string, reps *int, weight *string) error {
			if weight == nil {
				t.Error("Expected weight to be provided")
			}
			return nil
		},
	}

	uc := NewUpdateSetUseCase(mockRepo)
	kg := "70"
	req := &dto.UpdateSetRequest{Kg: &kg}

	result, err := uc.Execute(context.Background(), "set-123", req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestUpdateSetUseCase_Execute_EmptySetID(t *testing.T) {
	mockRepo := &MockSetWriter{}
	uc := NewUpdateSetUseCase(mockRepo)
	reps := 10
	req := &dto.UpdateSetRequest{Reps: &reps}

	result, err := uc.Execute(context.Background(), "", req)

	if err == nil {
		t.Fatal("Expected error for empty set_id")
	}

	if result != nil {
		t.Error("Expected nil result for error case")
	}
}

func TestUpdateSetUseCase_Execute_NilRequest(t *testing.T) {
	mockRepo := &MockSetWriter{}
	uc := NewUpdateSetUseCase(mockRepo)

	result, err := uc.Execute(context.Background(), "set-123", nil)

	if err == nil {
		t.Fatal("Expected error for nil request")
	}

	if result != nil {
		t.Error("Expected nil result for error case")
	}
}

func TestUpdateSetUseCase_Execute_EmptyRequest(t *testing.T) {
	mockRepo := &MockSetWriter{}
	uc := NewUpdateSetUseCase(mockRepo)
	req := &dto.UpdateSetRequest{}

	result, err := uc.Execute(context.Background(), "set-123", req)

	if err == nil {
		t.Fatal("Expected error for empty request")
	}

	if result != nil {
		t.Error("Expected nil result for error case")
	}
}

func TestUpdateSetUseCase_Execute_NegativeReps(t *testing.T) {
	mockRepo := &MockSetWriter{}
	uc := NewUpdateSetUseCase(mockRepo)
	reps := -5
	req := &dto.UpdateSetRequest{Reps: &reps}

	result, err := uc.Execute(context.Background(), "set-123", req)

	if err == nil {
		t.Fatal("Expected error for negative reps")
	}

	if result != nil {
		t.Error("Expected nil result for error case")
	}
}

func TestUpdateSetUseCase_Execute_RepoError(t *testing.T) {
	mockRepo := &MockSetWriter{
		UpdateFunc: func(ctx context.Context, setID string, reps *int, weight *string) error {
			return errors.New("database error")
		},
	}

	uc := NewUpdateSetUseCase(mockRepo)
	reps := 10
	req := &dto.UpdateSetRequest{Reps: &reps}

	result, err := uc.Execute(context.Background(), "set-123", req)

	if err == nil {
		t.Fatal("Expected error from repository")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestNewUpdateSetUseCase(t *testing.T) {
	mockRepo := &MockSetWriter{}
	uc := NewUpdateSetUseCase(mockRepo)

	if uc == nil {
		t.Error("Expected non-nil use case")
	}
}
