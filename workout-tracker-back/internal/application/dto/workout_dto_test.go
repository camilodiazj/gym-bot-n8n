package dto

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExerciseDTO_OmitsAlternativesWhenNil(t *testing.T) {
	exerciseDTO := ExerciseDTO{
		ID:         "test-id",
		Name:       "Test Exercise",
		BadgeColor: "#374151",
		RIR:        "3",
		Sets:       []SetDTO{},
		Tips:       []TipDTO{},
		Steps:      []StepDTO{},
		// AlternativeExercises intentionally left nil
	}

	data, err := json.Marshal(exerciseDTO)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "alternativeExercises") {
		t.Errorf("JSON should NOT contain alternativeExercises when nil, got: %s", jsonStr)
	}
}

func TestExerciseDTO_IncludesAlternativesWhenPresent(t *testing.T) {
	exerciseDTO := ExerciseDTO{
		ID:         "test-id",
		Name:       "Test Exercise",
		BadgeColor: "#374151",
		RIR:        "3",
		Sets:       []SetDTO{},
		Tips:       []TipDTO{},
		Steps:      []StepDTO{},
		AlternativeExercises: []AlternativeExerciseDTO{
			{
				Name:      "Alt Exercise",
				RIR:       "2",
				VideoLink: "https://example.com",
				Sets: []SetDTO{
					{ID: "alt-1", SetNumber: 1, Reps: 10, Kg: "-", Completed: false},
				},
			},
		},
	}

	data, err := json.Marshal(exerciseDTO)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, "alternativeExercises") {
		t.Errorf("JSON should contain alternativeExercises when present, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "Alt Exercise") {
		t.Errorf("JSON should contain alternative exercise name, got: %s", jsonStr)
	}
}

func TestAlternativeExerciseDTO_JSONContract(t *testing.T) {
	alt := AlternativeExerciseDTO{
		Name:        "Goblet Squat",
		RIR:         "3",
		RestSeconds: 120,
		VideoLink:   "https://example.com/video",
		Sets: []SetDTO{
			{ID: "wid:goblet_squat:1", SetNumber: 1, Reps: 10, Kg: "-", Completed: false},
			{ID: "wid:goblet_squat:2", SetNumber: 2, Reps: 10, Kg: "20", Completed: true},
		},
	}

	data, err := json.Marshal(alt)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Verify JSON keys match the expected API contract
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	expectedKeys := []string{"name", "rir", "restSeconds", "videoLink", "sets"}
	for _, key := range expectedKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("JSON missing expected key %q", key)
		}
	}
}

func TestAlternativeExerciseDTO_OmitsRestSecondsWhenZero(t *testing.T) {
	alt := AlternativeExerciseDTO{
		Name:      "Test",
		RIR:       "2",
		VideoLink: "https://example.com",
		Sets:      []SetDTO{},
		// RestSeconds = 0 (zero value)
	}

	data, err := json.Marshal(alt)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	if strings.Contains(string(data), "restSeconds") {
		t.Errorf("JSON should NOT contain restSeconds when 0, got: %s", string(data))
	}
}
