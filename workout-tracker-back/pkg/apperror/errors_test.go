package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	// Without underlying error
	err := &AppError{Code: 404, Message: "not found"}
	if err.Error() != "not found" {
		t.Errorf("Expected 'not found', got '%s'", err.Error())
	}

	// With underlying error
	underlying := errors.New("database error")
	err2 := &AppError{Code: 500, Message: "internal error", Err: underlying}
	expected := "internal error: database error"
	if err2.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err2.Error())
	}
}

func TestAppError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &AppError{Code: 500, Message: "wrapper", Err: underlying}

	if err.Unwrap() != underlying {
		t.Error("Unwrap should return the underlying error")
	}

	// Without underlying error
	err2 := &AppError{Code: 404, Message: "not found"}
	if err2.Unwrap() != nil {
		t.Error("Unwrap should return nil when no underlying error")
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("resource not found")

	if err.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, err.Code)
	}

	if err.Message != "resource not found" {
		t.Errorf("Expected message 'resource not found', got '%s'", err.Message)
	}

	if err.Err != nil {
		t.Error("Expected nil underlying error")
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("invalid input")

	if err.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, err.Code)
	}

	if err.Message != "invalid input" {
		t.Errorf("Expected message 'invalid input', got '%s'", err.Message)
	}
}

func TestNewInternalError(t *testing.T) {
	underlying := errors.New("db connection failed")
	err := NewInternalError("internal error", underlying)

	if err.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, err.Code)
	}

	if err.Message != "internal error" {
		t.Errorf("Expected message 'internal error', got '%s'", err.Message)
	}

	if err.Err != underlying {
		t.Error("Expected underlying error to be set")
	}
}

func TestNewUnauthorizedError(t *testing.T) {
	err := NewUnauthorizedError("unauthorized access")

	if err.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, err.Code)
	}

	if err.Message != "unauthorized access" {
		t.Errorf("Expected message 'unauthorized access', got '%s'", err.Message)
	}
}

func TestNewConflictError(t *testing.T) {
	err := NewConflictError("resource conflict")

	if err.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, err.Code)
	}

	if err.Message != "resource conflict" {
		t.Errorf("Expected message 'resource conflict', got '%s'", err.Message)
	}
}

func TestIsNotFound(t *testing.T) {
	notFoundErr := NewNotFoundError("not found")
	if !IsNotFound(notFoundErr) {
		t.Error("Expected IsNotFound to return true for NotFoundError")
	}

	validationErr := NewValidationError("invalid")
	if IsNotFound(validationErr) {
		t.Error("Expected IsNotFound to return false for ValidationError")
	}

	regularErr := errors.New("regular error")
	if IsNotFound(regularErr) {
		t.Error("Expected IsNotFound to return false for regular error")
	}
}

func TestIsValidation(t *testing.T) {
	validationErr := NewValidationError("invalid")
	if !IsValidation(validationErr) {
		t.Error("Expected IsValidation to return true for ValidationError")
	}

	notFoundErr := NewNotFoundError("not found")
	if IsValidation(notFoundErr) {
		t.Error("Expected IsValidation to return false for NotFoundError")
	}

	regularErr := errors.New("regular error")
	if IsValidation(regularErr) {
		t.Error("Expected IsValidation to return false for regular error")
	}
}
