package service

import (
	"strings"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// PreferenceProcessor processes user preferences from Spanish to English
type PreferenceProcessor struct {
	muscleMapping      map[string][]string
	healthRestrictions map[string]entity.HealthRestriction
	experienceMapping  map[string]string
	durationVolumeMap  map[string]float64
}

// NewPreferenceProcessor creates a new PreferenceProcessor
func NewPreferenceProcessor() *PreferenceProcessor {
	return &PreferenceProcessor{
		muscleMapping: map[string][]string{
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
		},
		healthRestrictions: map[string]entity.HealthRestriction{
			"A": {Code: "A"}, // No restrictions
			"B": {Code: "B", AvoidLowerBodyImpact: true},
			"C": {Code: "C", AvoidUpperBodyOverhead: true},
			"D": {Code: "D", AvoidAxialLoading: true},
			"E": {Code: "E", PreferMachines: true, IsSpecialCondition: true},
		},
		experienceMapping: map[string]string{
			"Menos de 6 meses": "beginner",
			"6 meses a 1 año":  "beginner",
			"6 a 12 meses":     "beginner",
			"1 a 2 años":       "intermediate",
			"1 a 2 anos":       "intermediate",
			"2 a 3 años":       "intermediate",
			"2 a 3 anos":       "intermediate",
			"Más de 3 años":    "advanced",
			"Mas de 3 anos":    "advanced",
		},
		durationVolumeMap: map[string]float64{
			"30-45 min": 0.70,
			"45-60 min": 0.85,
			"60-75 min": 1.00,
			"75-90 min": 1.15,
			"90+ min":   1.30,
		},
	}
}

// Process transforms a UserGymProfile into ProcessedPreferences
func (p *PreferenceProcessor) Process(profile *entity.UserGymProfile) *entity.ProcessedPreferences {
	result := entity.NewProcessedPreferences()

	// Map priority muscles
	if profile.PriorityMuscles != "" {
		result.PriorityMusclesEN = p.translateMuscles(profile.PriorityMuscles)
	}

	// Map disliked exercises (which are muscle groups)
	if profile.DislikedExercises != "" {
		result.DislikedMusclesEN = p.translateMuscles(profile.DislikedExercises)
	}

	// Map experience tier
	if tier, exists := p.experienceMapping[profile.TrainingExperience]; exists {
		result.ExperienceTier = tier
	}

	// Map volume modifier based on session duration
	if modifier, exists := p.durationVolumeMap[profile.SessionDurationMins]; exists {
		result.VolumeModifier = modifier
	}

	// Map health restrictions
	if restriction, exists := p.healthRestrictions[profile.HealthStatus]; exists {
		result.HealthRestrictions = restriction
	}

	// Copy sex and level
	result.Sex = profile.Sex
	result.Level = profile.FitnessLevel

	return result
}

// translateMuscles converts a comma-separated Spanish muscle string to English muscle array
func (p *PreferenceProcessor) translateMuscles(spanishMuscles string) []string {
	if spanishMuscles == "" {
		return []string{}
	}

	muscles := strings.Split(spanishMuscles, ",")
	result := make([]string, 0)
	seen := make(map[string]bool)

	for _, muscle := range muscles {
		muscle = strings.TrimSpace(muscle)
		if englishMuscles, exists := p.muscleMapping[muscle]; exists {
			for _, em := range englishMuscles {
				if !seen[em] {
					result = append(result, em)
					seen[em] = true
				}
			}
		}
	}

	return result
}

// TranslateSingleMuscle maps a single Spanish muscle name to English
func (p *PreferenceProcessor) TranslateSingleMuscle(spanishMuscle string) []string {
	spanishMuscle = strings.TrimSpace(spanishMuscle)
	if englishMuscles, exists := p.muscleMapping[spanishMuscle]; exists {
		return englishMuscles
	}
	return []string{}
}

// GetHealthRestriction returns health restrictions for a status code
func (p *PreferenceProcessor) GetHealthRestriction(healthStatus string) entity.HealthRestriction {
	if restriction, exists := p.healthRestrictions[healthStatus]; exists {
		return restriction
	}
	return entity.HealthRestriction{Code: healthStatus}
}

// GetDislikedMusclesForRotation returns the list of muscles to avoid during exercise rotation
func (p *PreferenceProcessor) GetDislikedMusclesForRotation(profile *entity.UserGymProfile) []string {
	processed := p.Process(profile)
	return processed.DislikedMusclesEN
}
