package service

import (
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

func TestNewPreferenceProcessor(t *testing.T) {
	pp := NewPreferenceProcessor()

	if pp == nil {
		t.Fatal("expected PreferenceProcessor to be created")
	}
	if pp.muscleMapping == nil {
		t.Error("expected muscleMapping to be initialized")
	}
	if pp.healthRestrictions == nil {
		t.Error("expected healthRestrictions to be initialized")
	}
	if pp.experienceMapping == nil {
		t.Error("expected experienceMapping to be initialized")
	}
	if pp.durationVolumeMap == nil {
		t.Error("expected durationVolumeMap to be initialized")
	}
}

func TestPreferenceProcessor_TranslateMuscles(t *testing.T) {
	pp := NewPreferenceProcessor()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single muscle",
			input:    "Pecho",
			expected: []string{"Chest"},
		},
		{
			name:     "multiple muscles",
			input:    "Pecho, Espalda",
			expected: []string{"Chest", "Back", "Lats", "Traps"},
		},
		{
			name:     "pierna expands to multiple",
			input:    "Pierna",
			expected: []string{"Quads", "Hamstrings"},
		},
		{
			name:     "brazos expands to multiple",
			input:    "Brazos",
			expected: []string{"Biceps", "Triceps", "Forearms"},
		},
		{
			name:     "with extra spaces",
			input:    "  Pecho  ,  Hombros  ",
			expected: []string{"Chest", "Shoulders", "Front Delts", "Rear Delts"},
		},
		{
			name:     "unknown muscle returns empty",
			input:    "UnknownMuscle",
			expected: []string{},
		},
		{
			name:     "mixed known and unknown",
			input:    "Pecho, UnknownMuscle, Biceps",
			expected: []string{"Chest", "Biceps"},
		},
		{
			name:     "no duplicates",
			input:    "Biceps, Bíceps",
			expected: []string{"Biceps"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pp.translateMuscles(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("translateMuscles(%q) returned %d muscles, want %d: got %v, want %v",
					tt.input, len(result), len(tt.expected), result, tt.expected)
				return
			}

			for i, muscle := range tt.expected {
				if result[i] != muscle {
					t.Errorf("translateMuscles(%q)[%d] = %v, want %v",
						tt.input, i, result[i], muscle)
				}
			}
		})
	}
}

func TestPreferenceProcessor_TranslateSingleMuscle(t *testing.T) {
	pp := NewPreferenceProcessor()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"chest", "Pecho", []string{"Chest"}},
		{"chest with space", " Pecho ", []string{"Chest"}},
		{"glutes with accent", "Glúteo", []string{"Glutes"}},
		{"glutes no accent", "Gluteo", []string{"Glutes"}},
		{"unknown", "Unknown", []string{}},
		{"empty", "", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pp.TranslateSingleMuscle(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("TranslateSingleMuscle(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}

			for i, muscle := range tt.expected {
				if result[i] != muscle {
					t.Errorf("TranslateSingleMuscle(%q)[%d] = %v, want %v", tt.input, i, result[i], muscle)
				}
			}
		})
	}
}

func TestPreferenceProcessor_GetHealthRestriction(t *testing.T) {
	pp := NewPreferenceProcessor()

	tests := []struct {
		name               string
		healthStatus       string
		wantAvoidOverhead  bool
		wantAvoidImpact    bool
		wantAvoidAxial     bool
		wantPreferMachines bool
		wantSpecial        bool
	}{
		{"code A", "A", false, false, false, false, false},
		{"code B", "B", false, true, false, false, false},
		{"code C", "C", true, false, false, false, false},
		{"code D", "D", false, false, true, false, false},
		{"code E", "E", false, false, false, true, true},
		{"unknown", "X", false, false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restriction := pp.GetHealthRestriction(tt.healthStatus)

			if restriction.AvoidUpperBodyOverhead != tt.wantAvoidOverhead {
				t.Errorf("AvoidUpperBodyOverhead = %v, want %v", restriction.AvoidUpperBodyOverhead, tt.wantAvoidOverhead)
			}
			if restriction.AvoidLowerBodyImpact != tt.wantAvoidImpact {
				t.Errorf("AvoidLowerBodyImpact = %v, want %v", restriction.AvoidLowerBodyImpact, tt.wantAvoidImpact)
			}
			if restriction.AvoidAxialLoading != tt.wantAvoidAxial {
				t.Errorf("AvoidAxialLoading = %v, want %v", restriction.AvoidAxialLoading, tt.wantAvoidAxial)
			}
			if restriction.PreferMachines != tt.wantPreferMachines {
				t.Errorf("PreferMachines = %v, want %v", restriction.PreferMachines, tt.wantPreferMachines)
			}
			if restriction.IsSpecialCondition != tt.wantSpecial {
				t.Errorf("IsSpecialCondition = %v, want %v", restriction.IsSpecialCondition, tt.wantSpecial)
			}
		})
	}
}

func TestPreferenceProcessor_Process(t *testing.T) {
	pp := NewPreferenceProcessor()

	profile := &entity.UserGymProfile{
		PriorityMuscles:     "Pecho, Espalda",
		DislikedExercises:   "Pantorrillas",
		TrainingExperience:  "1 a 2 años",
		SessionDurationMins: "60-75 min",
		HealthStatus:        "C",
		Sex:                 "M",
		FitnessLevel:        "Intermedio",
	}

	result := pp.Process(profile)

	// Check priority muscles
	expectedPriority := []string{"Chest", "Back", "Lats", "Traps"}
	if len(result.PriorityMusclesEN) != len(expectedPriority) {
		t.Errorf("PriorityMusclesEN = %v, want %v", result.PriorityMusclesEN, expectedPriority)
	}

	// Check disliked muscles
	if len(result.DislikedMusclesEN) != 1 || result.DislikedMusclesEN[0] != "Calfs" {
		t.Errorf("DislikedMusclesEN = %v, want [Calfs]", result.DislikedMusclesEN)
	}

	// Check experience tier
	if result.ExperienceTier != "intermediate" {
		t.Errorf("ExperienceTier = %v, want intermediate", result.ExperienceTier)
	}

	// Check volume modifier
	if result.VolumeModifier != 1.00 {
		t.Errorf("VolumeModifier = %v, want 1.00", result.VolumeModifier)
	}

	// Check health restrictions
	if !result.HealthRestrictions.AvoidUpperBodyOverhead {
		t.Error("expected AvoidUpperBodyOverhead to be true for code C")
	}

	// Check sex and level
	if result.Sex != "M" {
		t.Errorf("Sex = %v, want M", result.Sex)
	}
	if result.Level != "Intermedio" {
		t.Errorf("Level = %v, want Intermedio", result.Level)
	}
}

func TestPreferenceProcessor_Process_EmptyProfile(t *testing.T) {
	pp := NewPreferenceProcessor()

	profile := &entity.UserGymProfile{}

	result := pp.Process(profile)

	// Should return defaults
	if len(result.PriorityMusclesEN) != 0 {
		t.Errorf("expected empty PriorityMusclesEN, got %v", result.PriorityMusclesEN)
	}
	if len(result.DislikedMusclesEN) != 0 {
		t.Errorf("expected empty DislikedMusclesEN, got %v", result.DislikedMusclesEN)
	}
	if result.ExperienceTier != "intermediate" {
		t.Errorf("ExperienceTier = %v, want intermediate (default)", result.ExperienceTier)
	}
	if result.VolumeModifier != 1.0 {
		t.Errorf("VolumeModifier = %v, want 1.0 (default)", result.VolumeModifier)
	}
}

func TestPreferenceProcessor_GetDislikedMusclesForRotation(t *testing.T) {
	pp := NewPreferenceProcessor()

	profile := &entity.UserGymProfile{
		DislikedExercises: "Pantorrillas, Abdominales",
	}

	disliked := pp.GetDislikedMusclesForRotation(profile)

	expected := []string{"Calfs", "Abs", "Core"}
	if len(disliked) != len(expected) {
		t.Errorf("GetDislikedMusclesForRotation returned %v, want %v", disliked, expected)
		return
	}

	for i, muscle := range expected {
		if disliked[i] != muscle {
			t.Errorf("disliked[%d] = %v, want %v", i, disliked[i], muscle)
		}
	}
}

func TestPreferenceProcessor_ExperienceMapping(t *testing.T) {
	pp := NewPreferenceProcessor()

	tests := []struct {
		experience string
		wantTier   string
	}{
		{"Menos de 6 meses", "beginner"},
		{"6 meses a 1 año", "beginner"},
		{"1 a 2 años", "intermediate"},
		{"2 a 3 años", "intermediate"},
		{"Más de 3 años", "advanced"},
	}

	for _, tt := range tests {
		t.Run(tt.experience, func(t *testing.T) {
			profile := &entity.UserGymProfile{TrainingExperience: tt.experience}
			result := pp.Process(profile)

			if result.ExperienceTier != tt.wantTier {
				t.Errorf("ExperienceTier for %q = %v, want %v", tt.experience, result.ExperienceTier, tt.wantTier)
			}
		})
	}
}

func TestPreferenceProcessor_DurationVolumeMapping(t *testing.T) {
	pp := NewPreferenceProcessor()

	tests := []struct {
		duration   string
		wantVolume float64
	}{
		{"30-45 min", 0.70},
		{"45-60 min", 0.85},
		{"60-75 min", 1.00},
		{"75-90 min", 1.15},
		{"90+ min", 1.30},
	}

	for _, tt := range tests {
		t.Run(tt.duration, func(t *testing.T) {
			profile := &entity.UserGymProfile{SessionDurationMins: tt.duration}
			result := pp.Process(profile)

			if result.VolumeModifier != tt.wantVolume {
				t.Errorf("VolumeModifier for %q = %v, want %v", tt.duration, result.VolumeModifier, tt.wantVolume)
			}
		})
	}
}
