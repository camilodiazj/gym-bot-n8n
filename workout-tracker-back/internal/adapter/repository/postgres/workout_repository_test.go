package postgres

import "testing"

func TestParseReps(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"single number", "10", 10},
		{"range with hyphen", "10-12", 10},
		{"range with en-dash", "6–8", 6},
		{"single digit", "8", 8},
		{"empty string", "", 0},
		{"range starting with single digit", "8-10", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseReps(tt.input)
			if result != tt.expected {
				t.Errorf("parseReps(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseRepsRange(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedMin int
		expectedMax int
	}{
		{"single number", "10", 10, 10},
		{"range with hyphen", "10-12", 10, 12},
		{"range with en-dash", "6–8", 6, 8},
		{"range with en-dash larger", "8–15", 8, 15},
		{"single digit", "8", 8, 8},
		{"empty string", "", 0, 0},
		{"range starting with single digit", "8-10", 8, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max := parseRepsRange(tt.input)
			if min != tt.expectedMin || max != tt.expectedMax {
				t.Errorf("parseRepsRange(%q) = (%d, %d), want (%d, %d)",
					tt.input, min, max, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}
