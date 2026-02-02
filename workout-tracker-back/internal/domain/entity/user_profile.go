package entity

import "time"

// UserGymProfile represents the user's fitness profile from KYC
type UserGymProfile struct {
	WhatsAppID            string    `json:"whatsapp_id"`
	FullName              string    `json:"full_name"`
	Email                 string    `json:"email,omitempty"`
	Birthdate             string    `json:"birthdate,omitempty"`
	Sex                   string    `json:"sex"`
	Height                int       `json:"height"`
	Weight                float64   `json:"weight"`
	TrainingGoal          string    `json:"training_goal"`
	FitnessLevel          string    `json:"fitness_level"`
	TrainingExperience    string    `json:"training_experience"`
	DaysPerWeek           int       `json:"days_per_week"`
	SessionDurationMins   string    `json:"session_duration_mins"`
	HealthStatus          string    `json:"health_status"`
	PriorityMuscles       string    `json:"priority_muscles"`
	DislikedExercises     string    `json:"disliked_exercises"`
	AvailableEquipment    string    `json:"available_equipment,omitempty"`
	TrainingEnvironment   string    `json:"training_environment"`
	PreferredTrainingTime string    `json:"preferred_training_time,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// NewUserGymProfile creates a new UserGymProfile entity
func NewUserGymProfile(whatsappID, fullName, email string) *UserGymProfile {
	now := time.Now()
	return &UserGymProfile{
		WhatsAppID: whatsappID,
		FullName:   fullName,
		Email:      email,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// HealthRestriction represents health-based exercise restrictions
type HealthRestriction struct {
	Code                   string `json:"code"`
	AvoidUpperBodyOverhead bool   `json:"avoid_upper_body_overhead"`
	AvoidLowerBodyImpact   bool   `json:"avoid_lower_body_impact"`
	AvoidAxialLoading      bool   `json:"avoid_axial_loading"`
	PreferMachines         bool   `json:"prefer_machines"`
	IsSpecialCondition     bool   `json:"is_special_condition"`
}

// ProcessedPreferences represents the processed user preferences in English
type ProcessedPreferences struct {
	PriorityMusclesEN  []string          `json:"priority_muscles_en"`
	DislikedMusclesEN  []string          `json:"disliked_muscles_en"`
	ExperienceTier     string            `json:"experience_tier"`
	VolumeModifier     float64           `json:"volume_modifier"`
	HealthRestrictions HealthRestriction `json:"health_restrictions"`
	Sex                string            `json:"sex"`
	Level              string            `json:"level"`
}

// NewProcessedPreferences creates empty processed preferences with defaults
func NewProcessedPreferences() *ProcessedPreferences {
	return &ProcessedPreferences{
		PriorityMusclesEN:  []string{},
		DislikedMusclesEN:  []string{},
		ExperienceTier:     "intermediate",
		VolumeModifier:     1.0,
		HealthRestrictions: HealthRestriction{},
	}
}

// HasDislikedMuscle checks if a muscle is in the disliked list
func (p *ProcessedPreferences) HasDislikedMuscle(muscle string) bool {
	for _, m := range p.DislikedMusclesEN {
		if m == muscle {
			return true
		}
	}
	return false
}

// HasPriorityMuscle checks if a muscle is in the priority list
func (p *ProcessedPreferences) HasPriorityMuscle(muscle string) bool {
	for _, m := range p.PriorityMusclesEN {
		if m == muscle {
			return true
		}
	}
	return false
}

// MuscleTranslation maps Spanish muscle names to English database values
var MuscleTranslation = map[string][]string{
	// Spanish muscle names -> English database values
	"Gluteo":         {"Glutes"},
	"Glúteo":         {"Glutes"},
	"Glúteos":        {"Glutes"},
	"Pierna":         {"Quads", "Hamstrings"},
	"Piernas":        {"Quads", "Hamstrings"},
	"Cuádriceps":     {"Quads"},
	"Cuadriceps":     {"Quads"},
	"Isquiotibiales": {"Hamstrings"},
	"Pantorrillas":   {"Calfs"},
	"Pantorrilla":    {"Calfs"},
	"Pecho":          {"Chest"},
	"Espalda":        {"Back", "Lats", "Traps"},
	"Hombros":        {"Shoulders", "Front Delts", "Rear Delts"},
	"Hombro":         {"Shoulders", "Front Delts", "Rear Delts"},
	"Bíceps":         {"Biceps"},
	"Biceps":         {"Biceps"},
	"Tríceps":        {"Triceps"},
	"Triceps":        {"Triceps"},
	"Brazos":         {"Biceps", "Triceps", "Forearms"},
	"Abdomen":        {"Abs", "Core"},
	"Abdominales":    {"Abs", "Core"},
	"Core":           {"Abs", "Core"},
}

// GetHealthRestrictions returns restrictions based on health status code
func GetHealthRestrictions(code string) HealthRestriction {
	restrictions := HealthRestriction{Code: code}

	switch code {
	case "A":
		// No restrictions
	case "B": // Lower body issues
		restrictions.AvoidLowerBodyImpact = true
	case "C": // Upper body issues
		restrictions.AvoidUpperBodyOverhead = true
	case "D": // Spine issues
		restrictions.AvoidAxialLoading = true
	case "E": // Special condition
		restrictions.PreferMachines = true
		restrictions.IsSpecialCondition = true
	}

	return restrictions
}

// GetVolumeModifier returns volume adjustment based on session duration string
func GetVolumeModifier(durationStr string) float64 {
	durationMap := map[string]float64{
		"30-45 min": 0.70,
		"45-60 min": 0.85,
		"60-75 min": 1.00,
		"75-90 min": 1.15,
		"90+ min":   1.30,
	}

	if modifier, exists := durationMap[durationStr]; exists {
		return modifier
	}
	return 1.0 // Default
}

// GetVolumeModifierByMins returns volume adjustment based on session duration in minutes
func GetVolumeModifierByMins(durationMins int) float64 {
	switch {
	case durationMins <= 30:
		return 0.60
	case durationMins <= 45:
		return 0.70
	case durationMins <= 60:
		return 0.85
	case durationMins <= 75:
		return 1.00
	case durationMins <= 90:
		return 1.15
	default:
		return 1.30
	}
}

// GetExperienceTier returns tier based on training experience (Spanish input)
func GetExperienceTier(experience string) string {
	experienceMapping := map[string]string{
		"Menos de 6 meses": "beginner",
		"6 meses a 1 año":  "beginner",
		"6 a 12 meses":     "beginner",
		"1 a 2 años":       "intermediate",
		"1 a 2 anos":       "intermediate",
		"2 a 3 años":       "intermediate",
		"2 a 3 anos":       "intermediate",
		"Más de 3 años":    "advanced",
		"Mas de 3 anos":    "advanced",
	}

	if tier, exists := experienceMapping[experience]; exists {
		return tier
	}
	return "beginner" // Default for unknown experience levels
}
