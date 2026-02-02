package entity

import (
	"testing"
)

func TestNewUserGymProfile(t *testing.T) {
	profile := NewUserGymProfile("573001234567", "Juan Perez", "juan@example.com")

	if profile.WhatsAppID != "573001234567" {
		t.Errorf("expected WhatsAppID '573001234567', got '%s'", profile.WhatsAppID)
	}
	if profile.FullName != "Juan Perez" {
		t.Errorf("expected FullName 'Juan Perez', got '%s'", profile.FullName)
	}
	if profile.Email != "juan@example.com" {
		t.Errorf("expected Email 'juan@example.com', got '%s'", profile.Email)
	}
	if profile.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if profile.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestNewProcessedPreferences(t *testing.T) {
	prefs := NewProcessedPreferences()

	if prefs.ExperienceTier != "intermediate" {
		t.Errorf("expected default ExperienceTier 'intermediate', got '%s'", prefs.ExperienceTier)
	}
	if prefs.VolumeModifier != 1.0 {
		t.Errorf("expected default VolumeModifier 1.0, got %f", prefs.VolumeModifier)
	}
	if len(prefs.PriorityMusclesEN) != 0 {
		t.Errorf("expected empty PriorityMusclesEN, got %v", prefs.PriorityMusclesEN)
	}
	if len(prefs.DislikedMusclesEN) != 0 {
		t.Errorf("expected empty DislikedMusclesEN, got %v", prefs.DislikedMusclesEN)
	}
}

func TestProcessedPreferences_HasDislikedMuscle(t *testing.T) {
	prefs := &ProcessedPreferences{
		DislikedMusclesEN: []string{"Calfs", "Biceps"},
	}

	if !prefs.HasDislikedMuscle("Calfs") {
		t.Error("expected Calfs to be disliked")
	}
	if !prefs.HasDislikedMuscle("Biceps") {
		t.Error("expected Biceps to be disliked")
	}
	if prefs.HasDislikedMuscle("Chest") {
		t.Error("expected Chest to not be disliked")
	}
}

func TestProcessedPreferences_HasPriorityMuscle(t *testing.T) {
	prefs := &ProcessedPreferences{
		PriorityMusclesEN: []string{"Glutes", "Quads"},
	}

	if !prefs.HasPriorityMuscle("Glutes") {
		t.Error("expected Glutes to be priority")
	}
	if !prefs.HasPriorityMuscle("Quads") {
		t.Error("expected Quads to be priority")
	}
	if prefs.HasPriorityMuscle("Calfs") {
		t.Error("expected Calfs to not be priority")
	}
}

func TestGetHealthRestrictions(t *testing.T) {
	tests := []struct {
		name                     string
		code                     string
		wantAvoidUpperOverhead   bool
		wantAvoidLowerImpact     bool
		wantAvoidAxialLoading    bool
		wantPreferMachines       bool
		wantIsSpecialCondition   bool
	}{
		{
			name:                   "code A - no restrictions",
			code:                   "A",
			wantAvoidUpperOverhead: false,
			wantAvoidLowerImpact:   false,
			wantAvoidAxialLoading:  false,
			wantPreferMachines:     false,
		},
		{
			name:                 "code B - lower body issues",
			code:                 "B",
			wantAvoidLowerImpact: true,
		},
		{
			name:                   "code C - upper body issues",
			code:                   "C",
			wantAvoidUpperOverhead: true,
		},
		{
			name:                  "code D - spine issues",
			code:                  "D",
			wantAvoidAxialLoading: true,
		},
		{
			name:                   "code E - special condition",
			code:                   "E",
			wantPreferMachines:     true,
			wantIsSpecialCondition: true,
		},
		{
			name: "unknown code",
			code: "X",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restrictions := GetHealthRestrictions(tt.code)

			if restrictions.Code != tt.code {
				t.Errorf("Code = %v, want %v", restrictions.Code, tt.code)
			}
			if restrictions.AvoidUpperBodyOverhead != tt.wantAvoidUpperOverhead {
				t.Errorf("AvoidUpperBodyOverhead = %v, want %v", restrictions.AvoidUpperBodyOverhead, tt.wantAvoidUpperOverhead)
			}
			if restrictions.AvoidLowerBodyImpact != tt.wantAvoidLowerImpact {
				t.Errorf("AvoidLowerBodyImpact = %v, want %v", restrictions.AvoidLowerBodyImpact, tt.wantAvoidLowerImpact)
			}
			if restrictions.AvoidAxialLoading != tt.wantAvoidAxialLoading {
				t.Errorf("AvoidAxialLoading = %v, want %v", restrictions.AvoidAxialLoading, tt.wantAvoidAxialLoading)
			}
			if restrictions.PreferMachines != tt.wantPreferMachines {
				t.Errorf("PreferMachines = %v, want %v", restrictions.PreferMachines, tt.wantPreferMachines)
			}
			if restrictions.IsSpecialCondition != tt.wantIsSpecialCondition {
				t.Errorf("IsSpecialCondition = %v, want %v", restrictions.IsSpecialCondition, tt.wantIsSpecialCondition)
			}
		})
	}
}

func TestGetVolumeModifier(t *testing.T) {
	tests := []struct {
		name         string
		durationStr  string
		wantModifier float64
	}{
		{"30-45 min", "30-45 min", 0.70},
		{"45-60 min", "45-60 min", 0.85},
		{"60-75 min", "60-75 min", 1.00},
		{"75-90 min", "75-90 min", 1.15},
		{"90+ min", "90+ min", 1.30},
		{"unknown", "unknown", 1.0},
		{"empty", "", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetVolumeModifier(tt.durationStr); got != tt.wantModifier {
				t.Errorf("GetVolumeModifier(%q) = %v, want %v", tt.durationStr, got, tt.wantModifier)
			}
		})
	}
}

func TestGetVolumeModifierByMins(t *testing.T) {
	tests := []struct {
		name         string
		durationMins int
		wantModifier float64
	}{
		{"30 min or less", 30, 0.60},
		{"31-45 min", 45, 0.70},
		{"46-60 min", 60, 0.85},
		{"61-75 min", 75, 1.00},
		{"76-90 min", 90, 1.15},
		{"over 90 min", 120, 1.30},
		{"exactly 15 min", 15, 0.60},
		{"exactly 40 min", 40, 0.70},
		{"exactly 55 min", 55, 0.85},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetVolumeModifierByMins(tt.durationMins); got != tt.wantModifier {
				t.Errorf("GetVolumeModifierByMins(%d) = %v, want %v", tt.durationMins, got, tt.wantModifier)
			}
		})
	}
}

func TestGetExperienceTier(t *testing.T) {
	tests := []struct {
		name       string
		experience string
		wantTier   string
	}{
		{"less than 6 months", "Menos de 6 meses", "beginner"},
		{"6 months to 1 year", "6 meses a 1 año", "beginner"},
		{"6 to 12 months variant", "6 a 12 meses", "beginner"},
		{"1 to 2 years", "1 a 2 años", "intermediate"},
		{"1 to 2 years no accent", "1 a 2 anos", "intermediate"},
		{"2 to 3 years", "2 a 3 años", "intermediate"},
		{"2 to 3 years no accent", "2 a 3 anos", "intermediate"},
		{"more than 3 years", "Más de 3 años", "advanced"},
		{"more than 3 years no accent", "Mas de 3 anos", "advanced"},
		{"unknown", "some unknown value", "beginner"},
		{"empty", "", "beginner"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetExperienceTier(tt.experience); got != tt.wantTier {
				t.Errorf("GetExperienceTier(%q) = %v, want %v", tt.experience, got, tt.wantTier)
			}
		})
	}
}

func TestMuscleTranslation(t *testing.T) {
	// Test that the muscle translation map has expected entries
	tests := []struct {
		spanish  string
		expected []string
	}{
		{"Pecho", []string{"Chest"}},
		{"Espalda", []string{"Back", "Lats", "Traps"}},
		{"Pierna", []string{"Quads", "Hamstrings"}},
		{"Gluteo", []string{"Glutes"}},
		{"Glúteo", []string{"Glutes"}},
		{"Pantorrillas", []string{"Calfs"}},
		{"Hombros", []string{"Shoulders", "Front Delts", "Rear Delts"}},
		{"Biceps", []string{"Biceps"}},
		{"Bíceps", []string{"Biceps"}},
		{"Triceps", []string{"Triceps"}},
		{"Tríceps", []string{"Triceps"}},
		{"Abdominales", []string{"Abs", "Core"}},
	}

	for _, tt := range tests {
		t.Run(tt.spanish, func(t *testing.T) {
			got, exists := MuscleTranslation[tt.spanish]
			if !exists {
				t.Errorf("expected '%s' to exist in MuscleTranslation", tt.spanish)
				return
			}
			if len(got) != len(tt.expected) {
				t.Errorf("MuscleTranslation[%q] = %v, want %v", tt.spanish, got, tt.expected)
				return
			}
			for i, muscle := range tt.expected {
				if got[i] != muscle {
					t.Errorf("MuscleTranslation[%q][%d] = %v, want %v", tt.spanish, i, got[i], muscle)
				}
			}
		})
	}
}
