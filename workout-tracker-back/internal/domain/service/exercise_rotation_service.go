package service

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
)

// ExerciseRotationService handles exercise rotation logic
type ExerciseRotationService struct {
	exerciseRepo  repository.ExerciseReader
	prefProcessor *PreferenceProcessor
	rng           *rand.Rand
}

// NewExerciseRotationService creates a new ExerciseRotationService
func NewExerciseRotationService(exerciseRepo repository.ExerciseReader, prefProcessor *PreferenceProcessor) *ExerciseRotationService {
	// Seed with current time for production randomness
	return &ExerciseRotationService{
		exerciseRepo:  exerciseRepo,
		prefProcessor: prefProcessor,
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewExerciseRotationServiceWithSeed creates a service with a fixed seed for testing
func NewExerciseRotationServiceWithSeed(exerciseRepo repository.ExerciseReader, prefProcessor *PreferenceProcessor, seed int64) *ExerciseRotationService {
	return &ExerciseRotationService{
		exerciseRepo:  exerciseRepo,
		prefProcessor: prefProcessor,
		rng:           rand.New(rand.NewSource(seed)),
	}
}

// WithSeed sets a specific seed for deterministic testing
func (s *ExerciseRotationService) WithSeed(seed int64) *ExerciseRotationService {
	s.rng = rand.New(rand.NewSource(seed))
	return s
}

// RotationOptions contains options for exercise rotation
type RotationOptions struct {
	ExcludeExerciseIDs []string
	DislikedMuscles    []string
	PriorityMuscles    []string
	HealthRestrictions entity.HealthRestriction
	MaxCandidates      int
}

// FindAlternative finds an alternative exercise for a given pattern and role
func (s *ExerciseRotationService) FindAlternative(
	ctx context.Context,
	pattern, role string,
	opts RotationOptions,
) (*entity.ExerciseCatalog, error) {
	if opts.MaxCandidates == 0 {
		opts.MaxCandidates = 10
	}

	// Find candidate exercises
	candidates, err := s.exerciseRepo.FindAlternatives(
		ctx,
		pattern,
		role,
		opts.ExcludeExerciseIDs,
		opts.DislikedMuscles,
		opts.MaxCandidates*2, // Fetch extra to allow filtering
	)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return nil, nil // No alternatives found
	}

	// Apply health restrictions
	filtered := s.applyHealthRestrictions(candidates, opts.HealthRestrictions)
	if len(filtered) == 0 {
		filtered = candidates // Fallback to unfiltered if too restrictive
	}

	// Score and rank candidates by priority muscles
	scored := s.scoreExercises(filtered, opts.PriorityMuscles)

	// Select from top candidates with weighted randomness
	return s.selectFromTopCandidates(scored, opts.MaxCandidates), nil
}

// RotateExercises generates exercise rotations for a user based on their current workouts
func (s *ExerciseRotationService) RotateExercises(
	ctx context.Context,
	workouts []*entity.WorkoutExercise,
	profile *entity.UserGymProfile,
) ([]*entity.ExerciseRotation, error) {
	rotations := make([]*entity.ExerciseRotation, 0)
	usedExerciseIDs := make(map[string]bool)

	// Get processed preferences
	preferences := s.prefProcessor.Process(profile)

	// Collect all current exercise IDs to exclude
	for _, w := range workouts {
		usedExerciseIDs[w.ExerciseID] = true
	}

	// We need to get exercise details to know pattern/role
	// Group workouts by exercise ID first, then by pattern+role
	exerciseDetails := make(map[string]*entity.ExerciseCatalog)
	patternRoleMap := make(map[string][]*entity.WorkoutExercise)

	for _, w := range workouts {
		// Get exercise details if not already fetched
		if _, exists := exerciseDetails[w.ExerciseID]; !exists {
			exercise, err := s.exerciseRepo.GetByID(ctx, w.ExerciseID)
			if err != nil {
				return nil, err
			}
			if exercise == nil {
				continue // Skip if exercise not found
			}
			exerciseDetails[w.ExerciseID] = exercise
		}

		exercise := exerciseDetails[w.ExerciseID]
		key := makePatternRoleKey(exercise.Pattern, exercise.Role)
		patternRoleMap[key] = append(patternRoleMap[key], w)
	}

	// For each pattern+role group, find one alternative
	for key, workoutGroup := range patternRoleMap {
		parts := splitPatternRoleKey(key)
		if len(parts) != 2 {
			continue
		}
		pattern, role := parts[0], parts[1]

		// Find one alternative for this pattern+role
		opts := RotationOptions{
			ExcludeExerciseIDs: mapKeysToSlice(usedExerciseIDs),
			DislikedMuscles:    preferences.DislikedMusclesEN,
			PriorityMuscles:    preferences.PriorityMusclesEN,
			HealthRestrictions: preferences.HealthRestrictions,
			MaxCandidates:      5,
		}

		alternative, err := s.FindAlternative(ctx, pattern, role, opts)
		if err != nil {
			return nil, err
		}

		if alternative == nil {
			continue // Keep original if no alternative found
		}

		// Apply same alternative to all workouts with this pattern+role
		for _, w := range workoutGroup {
			oldExercise := exerciseDetails[w.ExerciseID]
			rotation := entity.NewExerciseRotation(
				w.ID,
				w.ExerciseID,
				alternative.ExerciseID,
				pattern,
				role,
			)
			rotation.OldExercise = oldExercise
			rotation.OldExerciseName = oldExercise.SpanishName
			rotation.NewExercise = alternative
			rotation.NewExerciseName = alternative.SpanishName
			rotation.DayName = w.DayName
			rotations = append(rotations, rotation)
		}

		// Mark new exercise as used
		usedExerciseIDs[alternative.ExerciseID] = true
	}

	return rotations, nil
}

// ShouldRotate determines if an exercise should be rotated based on mesocycle count and role
func (s *ExerciseRotationService) ShouldRotate(role string, mesocycleNumber int) bool {
	// Rotation frequency depends on exercise role
	// Compound exercises rotate less frequently to maintain strength progression
	// Isolation exercises rotate more frequently for variety
	switch role {
	case "compound":
		// Rotate compound exercises every 3-4 mesocycles
		return mesocycleNumber%4 == 0 || mesocycleNumber%3 == 0
	case "isolation":
		// Rotate isolation exercises every 2-3 mesocycles
		return mesocycleNumber%3 == 0 || mesocycleNumber%2 == 0
	case "core":
		// Rotate core exercises every 3 mesocycles
		return mesocycleNumber%3 == 0
	default:
		return false
	}
}

// applyHealthRestrictions filters exercises based on health restrictions
func (s *ExerciseRotationService) applyHealthRestrictions(
	exercises []*entity.ExerciseCatalog,
	restrictions entity.HealthRestriction,
) []*entity.ExerciseCatalog {
	// If no restrictions, return all exercises
	if !restrictions.AvoidUpperBodyOverhead &&
		!restrictions.AvoidLowerBodyImpact &&
		!restrictions.AvoidAxialLoading &&
		!restrictions.PreferMachines {
		return exercises
	}

	filtered := make([]*entity.ExerciseCatalog, 0)
	for _, e := range exercises {
		if s.exercisePassesHealthRestrictions(e, restrictions) {
			filtered = append(filtered, e)
		}
	}

	return filtered
}

// exercisePassesHealthRestrictions checks if an exercise passes health restrictions
func (s *ExerciseRotationService) exercisePassesHealthRestrictions(
	exercise *entity.ExerciseCatalog,
	restrictions entity.HealthRestriction,
) bool {
	// Check pattern-based restrictions
	pattern := strings.ToLower(exercise.Pattern)

	if restrictions.AvoidUpperBodyOverhead {
		// Avoid overhead pressing movements
		if pattern == "vertical_push" || pattern == "overhead_press" {
			return false
		}
	}

	if restrictions.AvoidLowerBodyImpact {
		// Avoid high-impact lower body movements
		if pattern == "jump" || pattern == "plyometric" {
			return false
		}
	}

	if restrictions.AvoidAxialLoading {
		// Avoid exercises with heavy spinal loading
		if pattern == "squat" || pattern == "deadlift" || pattern == "hip_hinge" {
			// Machine versions might be acceptable
			if restrictions.PreferMachines && exercise.Equipment != "machine" {
				return false
			}
		}
	}

	if restrictions.PreferMachines {
		// Prefer machine exercises for special conditions
		// But don't completely exclude free weights
		// This is a preference, not a hard filter
	}

	return true
}

// scoreExercises scores exercises based on priority muscles
func (s *ExerciseRotationService) scoreExercises(
	exercises []*entity.ExerciseCatalog,
	priorityMuscles []string,
) []*scoredExercise {
	scored := make([]*scoredExercise, len(exercises))

	for i, e := range exercises {
		score := 0

		// Higher score for exercises targeting priority muscles
		for _, muscle := range priorityMuscles {
			if e.MainMuscle == muscle {
				score += 10 // Main muscle match
			}
			for _, secondary := range e.SecondaryMuscles {
				if secondary == muscle {
					score += 5 // Secondary muscle match
				}
			}
		}

		scored[i] = &scoredExercise{
			exercise: e,
			score:    score,
		}
	}

	return scored
}

// selectFromTopCandidates selects an exercise from the top candidates with weighted randomness
func (s *ExerciseRotationService) selectFromTopCandidates(
	scored []*scoredExercise,
	maxCandidates int,
) *entity.ExerciseCatalog {
	if len(scored) == 0 {
		return nil
	}

	// Sort by score descending (simple bubble sort for small arrays)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Limit to top candidates
	limit := maxCandidates
	if limit > len(scored) {
		limit = len(scored)
	}
	topCandidates := scored[:limit]

	// Select randomly from top candidates (weighted towards higher scores)
	// Simple approach: pick randomly from top candidates
	idx := s.rng.Intn(len(topCandidates))
	return topCandidates[idx].exercise
}

// scoredExercise holds an exercise with its score
type scoredExercise struct {
	exercise *entity.ExerciseCatalog
	score    int
}

// makePatternRoleKey creates a composite key for grouping exercises
func makePatternRoleKey(pattern, role string) string {
	return pattern + ":" + role
}

// splitPatternRoleKey splits a composite key into pattern and role
func splitPatternRoleKey(key string) []string {
	return strings.Split(key, ":")
}

// mapKeysToSlice converts map keys to a slice
func mapKeysToSlice(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
