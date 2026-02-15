package postgres

import (
	"testing"
)

func Test_parseSetID(t *testing.T) {
	tests := []struct {
		name           string
		setID          string
		wantWorkoutID  string
		wantExerciseID string
		wantSetNumber  int
		wantIsAlt      bool
		wantErr        bool
	}{
		{
			name:           "primary set ID",
			setID:          "550e8400-e29b-41d4-a716-446655440001-2",
			wantWorkoutID:  "550e8400-e29b-41d4-a716-446655440001",
			wantExerciseID: "",
			wantSetNumber:  2,
			wantIsAlt:      false,
		},
		{
			name:           "alternative set ID",
			setID:          "550e8400-e29b-41d4-a716-446655440001:bicep_curl_db:3",
			wantWorkoutID:  "550e8400-e29b-41d4-a716-446655440001",
			wantExerciseID: "bicep_curl_db",
			wantSetNumber:  3,
			wantIsAlt:      true,
		},
		{
			name:           "primary set number 1",
			setID:          "abc-def-123-1",
			wantWorkoutID:  "abc-def-123",
			wantExerciseID: "",
			wantSetNumber:  1,
			wantIsAlt:      false,
		},
		{
			name:    "invalid - empty string",
			setID:   "",
			wantErr: true,
		},
		{
			name:    "invalid - no separator",
			setID:   "noseparator",
			wantErr: true,
		},
		{
			name:    "invalid - colon with only 2 parts",
			setID:   "a:b",
			wantErr: true,
		},
		{
			name:    "invalid - colon with non-numeric set number",
			setID:   "a:b:notanum",
			wantErr: true,
		},
		{
			name:    "invalid - primary with non-numeric set number",
			setID:   "abc-def-xyz",
			wantErr: true,
		},
		{
			name:    "invalid - ends with dash",
			setID:   "abc-",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSetID(tt.setID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSetID(%q) expected error, got nil", tt.setID)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSetID(%q) unexpected error: %v", tt.setID, err)
			}
			if got.workoutID != tt.wantWorkoutID {
				t.Errorf("workoutID = %q, want %q", got.workoutID, tt.wantWorkoutID)
			}
			if got.exerciseID != tt.wantExerciseID {
				t.Errorf("exerciseID = %q, want %q", got.exerciseID, tt.wantExerciseID)
			}
			if got.setNumber != tt.wantSetNumber {
				t.Errorf("setNumber = %d, want %d", got.setNumber, tt.wantSetNumber)
			}
			if got.isAlternative != tt.wantIsAlt {
				t.Errorf("isAlternative = %v, want %v", got.isAlternative, tt.wantIsAlt)
			}
		})
	}
}
