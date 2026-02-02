package postgres

import (
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

func TestNewUserProfileRepository(t *testing.T) {
	repo := NewUserProfileRepository(nil)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestUserGymProfile_Creation(t *testing.T) {
	profile := entity.NewUserGymProfile("573001234567@s.whatsapp.net", "John Doe", "john@example.com")

	if profile.WhatsAppID != "573001234567@s.whatsapp.net" {
		t.Errorf("WhatsAppID = %s, want 573001234567@s.whatsapp.net", profile.WhatsAppID)
	}
	if profile.FullName != "John Doe" {
		t.Errorf("FullName = %s, want John Doe", profile.FullName)
	}
	if profile.Email != "john@example.com" {
		t.Errorf("Email = %s, want john@example.com", profile.Email)
	}
	if profile.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if profile.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestProcessedPreferences_HasDislikedMuscle(t *testing.T) {
	prefs := entity.NewProcessedPreferences()
	prefs.DislikedMusclesEN = []string{"Calfs", "Forearms"}

	tests := []struct {
		name   string
		muscle string
		want   bool
	}{
		{"disliked muscle found", "Calfs", true},
		{"another disliked muscle found", "Forearms", true},
		{"not disliked muscle", "Chest", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prefs.HasDislikedMuscle(tt.muscle)
			if got != tt.want {
				t.Errorf("HasDislikedMuscle(%s) = %v, want %v", tt.muscle, got, tt.want)
			}
		})
	}
}

func TestProcessedPreferences_HasPriorityMuscle(t *testing.T) {
	prefs := entity.NewProcessedPreferences()
	prefs.PriorityMusclesEN = []string{"Glutes", "Quads", "Hamstrings"}

	tests := []struct {
		name   string
		muscle string
		want   bool
	}{
		{"priority muscle found", "Glutes", true},
		{"another priority muscle found", "Quads", true},
		{"third priority muscle found", "Hamstrings", true},
		{"not priority muscle", "Biceps", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prefs.HasPriorityMuscle(tt.muscle)
			if got != tt.want {
				t.Errorf("HasPriorityMuscle(%s) = %v, want %v", tt.muscle, got, tt.want)
			}
		})
	}
}

func TestGetHealthRestrictions(t *testing.T) {
	tests := []struct {
		name                   string
		code                   string
		wantOverhead           bool
		wantLowerBodyImpact    bool
		wantAxialLoading       bool
		wantPreferMachines     bool
		wantSpecialCondition   bool
	}{
		{
			name:                 "code A - no restrictions",
			code:                 "A",
			wantOverhead:         false,
			wantLowerBodyImpact:  false,
			wantAxialLoading:     false,
			wantPreferMachines:   false,
			wantSpecialCondition: false,
		},
		{
			name:                 "code B - lower body issues",
			code:                 "B",
			wantOverhead:         false,
			wantLowerBodyImpact:  true,
			wantAxialLoading:     false,
			wantPreferMachines:   false,
			wantSpecialCondition: false,
		},
		{
			name:                 "code C - upper body issues",
			code:                 "C",
			wantOverhead:         true,
			wantLowerBodyImpact:  false,
			wantAxialLoading:     false,
			wantPreferMachines:   false,
			wantSpecialCondition: false,
		},
		{
			name:                 "code D - spine issues",
			code:                 "D",
			wantOverhead:         false,
			wantLowerBodyImpact:  false,
			wantAxialLoading:     true,
			wantPreferMachines:   false,
			wantSpecialCondition: false,
		},
		{
			name:                 "code E - special condition",
			code:                 "E",
			wantOverhead:         false,
			wantLowerBodyImpact:  false,
			wantAxialLoading:     false,
			wantPreferMachines:   true,
			wantSpecialCondition: true,
		},
		{
			name:                 "unknown code - no restrictions",
			code:                 "X",
			wantOverhead:         false,
			wantLowerBodyImpact:  false,
			wantAxialLoading:     false,
			wantPreferMachines:   false,
			wantSpecialCondition: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := entity.GetHealthRestrictions(tt.code)

			if r.AvoidUpperBodyOverhead != tt.wantOverhead {
				t.Errorf("AvoidUpperBodyOverhead = %v, want %v", r.AvoidUpperBodyOverhead, tt.wantOverhead)
			}
			if r.AvoidLowerBodyImpact != tt.wantLowerBodyImpact {
				t.Errorf("AvoidLowerBodyImpact = %v, want %v", r.AvoidLowerBodyImpact, tt.wantLowerBodyImpact)
			}
			if r.AvoidAxialLoading != tt.wantAxialLoading {
				t.Errorf("AvoidAxialLoading = %v, want %v", r.AvoidAxialLoading, tt.wantAxialLoading)
			}
			if r.PreferMachines != tt.wantPreferMachines {
				t.Errorf("PreferMachines = %v, want %v", r.PreferMachines, tt.wantPreferMachines)
			}
			if r.IsSpecialCondition != tt.wantSpecialCondition {
				t.Errorf("IsSpecialCondition = %v, want %v", r.IsSpecialCondition, tt.wantSpecialCondition)
			}
		})
	}
}

func TestGetVolumeModifier(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		want     float64
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
			got := entity.GetVolumeModifier(tt.duration)
			if got != tt.want {
				t.Errorf("GetVolumeModifier(%s) = %v, want %v", tt.duration, got, tt.want)
			}
		})
	}
}

func TestGetExperienceTier(t *testing.T) {
	tests := []struct {
		name       string
		experience string
		want       string
	}{
		{"less than 6 months", "Menos de 6 meses", "beginner"},
		{"6 months to 1 year", "6 meses a 1 año", "beginner"},
		{"1 to 2 years", "1 a 2 años", "intermediate"},
		{"2 to 3 years", "2 a 3 años", "intermediate"},
		{"more than 3 years", "Más de 3 años", "advanced"},
		{"unknown", "unknown", "beginner"},
		{"empty", "", "beginner"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := entity.GetExperienceTier(tt.experience)
			if got != tt.want {
				t.Errorf("GetExperienceTier(%s) = %s, want %s", tt.experience, got, tt.want)
			}
		})
	}
}

func TestMuscleTranslation(t *testing.T) {
	tests := []struct {
		name     string
		spanish  string
		expected []string
	}{
		{"Gluteo", "Glúteo", []string{"Glutes"}},
		{"Pierna maps to quads and hamstrings", "Pierna", []string{"Quads", "Hamstrings"}},
		{"Pantorrillas", "Pantorrillas", []string{"Calfs"}},
		{"Pecho", "Pecho", []string{"Chest"}},
		{"Espalda maps to multiple", "Espalda", []string{"Back", "Lats", "Traps"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, exists := entity.MuscleTranslation[tt.spanish]
			if !exists {
				t.Errorf("MuscleTranslation[%s] not found", tt.spanish)
				return
			}
			if len(got) != len(tt.expected) {
				t.Errorf("MuscleTranslation[%s] = %v, want %v", tt.spanish, got, tt.expected)
				return
			}
			for i, muscle := range got {
				if muscle != tt.expected[i] {
					t.Errorf("MuscleTranslation[%s][%d] = %s, want %s", tt.spanish, i, muscle, tt.expected[i])
				}
			}
		})
	}
}
