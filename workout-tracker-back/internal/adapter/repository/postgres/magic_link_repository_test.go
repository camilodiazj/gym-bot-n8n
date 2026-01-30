package postgres

import (
	"testing"
)

func TestGenerateCode(t *testing.T) {
	// Test that generateCode returns a 6-character string
	code, err := generateCode()
	if err != nil {
		t.Fatalf("generateCode failed: %v", err)
	}

	if len(code) != 6 {
		t.Errorf("Expected code length 6, got %d", len(code))
	}

	// Test uniqueness - generate multiple codes
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		c, err := generateCode()
		if err != nil {
			t.Fatalf("generateCode failed on iteration %d: %v", i, err)
		}
		if codes[c] {
			// Collision is possible but very unlikely in 100 iterations
			t.Logf("Warning: code collision detected: %s", c)
		}
		codes[c] = true
	}
}

func TestGenerateCode_ValidCharacters(t *testing.T) {
	// Test that generated codes contain only URL-safe characters
	for i := 0; i < 50; i++ {
		code, err := generateCode()
		if err != nil {
			t.Fatalf("generateCode failed: %v", err)
		}

		for _, char := range code {
			// URL-safe base64 characters: A-Z, a-z, 0-9, -, _
			isValid := (char >= 'A' && char <= 'Z') ||
				(char >= 'a' && char <= 'z') ||
				(char >= '0' && char <= '9') ||
				char == '-' || char == '_'
			if !isValid {
				t.Errorf("Invalid character in code: %c", char)
			}
		}
	}
}

func TestNewMagicLinkRepository(t *testing.T) {
	// Test that NewMagicLinkRepository returns a non-nil repository
	// Note: We can't test with a real connection here, just the constructor
	repo := NewMagicLinkRepository(nil)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}
