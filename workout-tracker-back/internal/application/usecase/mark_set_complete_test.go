package usecase

import (
	"context"
	"errors"
	"testing"
)

// MockSetWriter for testing
type MockSetWriter struct {
	MarkCompleteFunc func(ctx context.Context, setID string) error
	UpdateFunc       func(ctx context.Context, setID string, reps *int, weight *string) error
}

func (m *MockSetWriter) MarkComplete(ctx context.Context, setID string) error {
	if m.MarkCompleteFunc != nil {
		return m.MarkCompleteFunc(ctx, setID)
	}
	return nil
}

func (m *MockSetWriter) Update(ctx context.Context, setID string, reps *int, weight *string) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, setID, reps, weight)
	}
	return nil
}

func TestMarkSetCompleteUseCase_Execute_Success(t *testing.T) {
	mockRepo := &MockSetWriter{
		MarkCompleteFunc: func(ctx context.Context, setID string) error {
			return nil
		},
	}

	uc := NewMarkSetCompleteUseCase(mockRepo)
	result, err := uc.Execute(context.Background(), "set-123")

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

	if result.CompletedAt == "" {
		t.Error("Expected non-empty CompletedAt")
	}
}

func TestMarkSetCompleteUseCase_Execute_EmptySetID(t *testing.T) {
	mockRepo := &MockSetWriter{}
	uc := NewMarkSetCompleteUseCase(mockRepo)

	result, err := uc.Execute(context.Background(), "")

	if err == nil {
		t.Fatal("Expected error for empty set_id")
	}

	if result != nil {
		t.Error("Expected nil result for error case")
	}
}

func TestMarkSetCompleteUseCase_Execute_RepoError(t *testing.T) {
	mockRepo := &MockSetWriter{
		MarkCompleteFunc: func(ctx context.Context, setID string) error {
			return errors.New("database error")
		},
	}

	uc := NewMarkSetCompleteUseCase(mockRepo)
	result, err := uc.Execute(context.Background(), "set-123")

	if err == nil {
		t.Fatal("Expected error from repository")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestNewMarkSetCompleteUseCase(t *testing.T) {
	mockRepo := &MockSetWriter{}
	uc := NewMarkSetCompleteUseCase(mockRepo)

	if uc == nil {
		t.Error("Expected non-nil use case")
	}
}
