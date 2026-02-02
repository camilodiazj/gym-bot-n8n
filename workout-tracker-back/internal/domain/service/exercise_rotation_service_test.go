package service

import (
	"context"
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// mockExerciseReader is a mock implementation of repository.ExerciseReader
type mockExerciseReader struct {
	exercises    map[string]*entity.ExerciseCatalog
	alternatives []*entity.ExerciseCatalog
	findError    error
}

func newMockExerciseReader() *mockExerciseReader {
	return &mockExerciseReader{
		exercises:    make(map[string]*entity.ExerciseCatalog),
		alternatives: make([]*entity.ExerciseCatalog, 0),
	}
}

func (m *mockExerciseReader) GetByID(ctx context.Context, exerciseID string) (*entity.ExerciseCatalog, error) {
	if m.findError != nil {
		return nil, m.findError
	}
	return m.exercises[exerciseID], nil
}

func (m *mockExerciseReader) FindAlternatives(ctx context.Context, pattern, role string, excludeIDs, excludeMuscles []string, limit int) ([]*entity.ExerciseCatalog, error) {
	if m.findError != nil {
		return nil, m.findError
	}

	// Filter by pattern and role
	result := make([]*entity.ExerciseCatalog, 0)
	for _, e := range m.alternatives {
		if e.Pattern == pattern && e.Role == role {
			// Check if excluded
			excluded := false
			for _, id := range excludeIDs {
				if e.ExerciseID == id {
					excluded = true
					break
				}
			}
			if !excluded {
				// Check if muscle is excluded
				for _, muscle := range excludeMuscles {
					if e.MainMuscle == muscle {
						excluded = true
						break
					}
				}
			}
			if !excluded {
				result = append(result, e)
			}
		}
	}

	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (m *mockExerciseReader) FindByPatternAndRole(ctx context.Context, pattern, role string) ([]*entity.ExerciseCatalog, error) {
	if m.findError != nil {
		return nil, m.findError
	}

	result := make([]*entity.ExerciseCatalog, 0)
	for _, e := range m.alternatives {
		if e.Pattern == pattern && e.Role == role {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockExerciseReader) addExercise(e *entity.ExerciseCatalog) {
	m.exercises[e.ExerciseID] = e
	m.alternatives = append(m.alternatives, e)
}

func TestNewExerciseRotationService(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()

	service := NewExerciseRotationService(mockRepo, pp)

	if service == nil {
		t.Fatal("expected service to be created")
	}
	if service.exerciseRepo == nil {
		t.Error("expected exerciseRepo to be set")
	}
	if service.prefProcessor == nil {
		t.Error("expected prefProcessor to be set")
	}
	if service.rng == nil {
		t.Error("expected rng to be set")
	}
}

func TestNewExerciseRotationServiceWithSeed(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()

	service1 := NewExerciseRotationServiceWithSeed(mockRepo, pp, 12345)
	service2 := NewExerciseRotationServiceWithSeed(mockRepo, pp, 12345)

	// Both services with same seed should produce same random sequence
	r1 := service1.rng.Intn(100)
	r2 := service2.rng.Intn(100)

	if r1 != r2 {
		t.Errorf("expected same random values with same seed, got %d and %d", r1, r2)
	}
}

func TestExerciseRotationService_WithSeed(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()

	service := NewExerciseRotationService(mockRepo, pp).WithSeed(54321)

	// Should be able to chain WithSeed
	if service == nil {
		t.Error("expected service to be returned from WithSeed")
	}
}

func TestExerciseRotationService_ShouldRotate(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()
	service := NewExerciseRotationService(mockRepo, pp)

	tests := []struct {
		name       string
		role       string
		mesocycle  int
		wantRotate bool
	}{
		// Compound exercises rotate every 3-4 mesocycles
		{"compound mesocycle 1", "compound", 1, false},
		{"compound mesocycle 2", "compound", 2, false},
		{"compound mesocycle 3", "compound", 3, true},
		{"compound mesocycle 4", "compound", 4, true},
		{"compound mesocycle 5", "compound", 5, false},
		{"compound mesocycle 6", "compound", 6, true},

		// Isolation exercises rotate every 2-3 mesocycles
		{"isolation mesocycle 1", "isolation", 1, false},
		{"isolation mesocycle 2", "isolation", 2, true},
		{"isolation mesocycle 3", "isolation", 3, true},
		{"isolation mesocycle 4", "isolation", 4, true},
		{"isolation mesocycle 5", "isolation", 5, false},

		// Core exercises rotate every 3 mesocycles
		{"core mesocycle 1", "core", 1, false},
		{"core mesocycle 2", "core", 2, false},
		{"core mesocycle 3", "core", 3, true},
		{"core mesocycle 4", "core", 4, false},
		{"core mesocycle 5", "core", 5, false},
		{"core mesocycle 6", "core", 6, true},

		// Unknown role
		{"unknown role", "unknown", 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.ShouldRotate(tt.role, tt.mesocycle); got != tt.wantRotate {
				t.Errorf("ShouldRotate(%q, %d) = %v, want %v", tt.role, tt.mesocycle, got, tt.wantRotate)
			}
		})
	}
}

func TestExerciseRotationService_FindAlternative(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()

	// Add some exercises to the mock
	mockRepo.addExercise(&entity.ExerciseCatalog{
		ExerciseID:  "ex-1",
		SpanishName: "Press de banca",
		Pattern:     "horizontal_push",
		Role:        "compound",
		MainMuscle:  "Chest",
	})
	mockRepo.addExercise(&entity.ExerciseCatalog{
		ExerciseID:  "ex-2",
		SpanishName: "Press inclinado",
		Pattern:     "horizontal_push",
		Role:        "compound",
		MainMuscle:  "Chest",
	})
	mockRepo.addExercise(&entity.ExerciseCatalog{
		ExerciseID:  "ex-3",
		SpanishName: "Press declinado",
		Pattern:     "horizontal_push",
		Role:        "compound",
		MainMuscle:  "Chest",
	})

	service := NewExerciseRotationServiceWithSeed(mockRepo, pp, 12345)

	opts := RotationOptions{
		ExcludeExerciseIDs: []string{"ex-1"},
		DislikedMuscles:    []string{},
		PriorityMuscles:    []string{},
		MaxCandidates:      5,
	}

	alternative, err := service.FindAlternative(context.Background(), "horizontal_push", "compound", opts)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alternative == nil {
		t.Fatal("expected an alternative to be found")
	}
	if alternative.ExerciseID == "ex-1" {
		t.Error("expected alternative to not be the excluded exercise")
	}
}

func TestExerciseRotationService_FindAlternative_NoAlternatives(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()
	service := NewExerciseRotationService(mockRepo, pp)

	opts := RotationOptions{
		ExcludeExerciseIDs: []string{},
		MaxCandidates:      5,
	}

	alternative, err := service.FindAlternative(context.Background(), "nonexistent_pattern", "compound", opts)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alternative != nil {
		t.Error("expected nil when no alternatives found")
	}
}

func TestExerciseRotationService_FindAlternative_WithDislikedMuscles(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()

	mockRepo.addExercise(&entity.ExerciseCatalog{
		ExerciseID:  "ex-1",
		SpanishName: "Elevacion de talones",
		Pattern:     "calf_raise",
		Role:        "isolation",
		MainMuscle:  "Calfs",
	})
	mockRepo.addExercise(&entity.ExerciseCatalog{
		ExerciseID:  "ex-2",
		SpanishName: "Curl de biceps",
		Pattern:     "curl",
		Role:        "isolation",
		MainMuscle:  "Biceps",
	})

	service := NewExerciseRotationService(mockRepo, pp)

	opts := RotationOptions{
		DislikedMuscles: []string{"Calfs"},
		MaxCandidates:   5,
	}

	// Should not return calf exercise
	alternative, err := service.FindAlternative(context.Background(), "calf_raise", "isolation", opts)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alternative != nil && alternative.MainMuscle == "Calfs" {
		t.Error("expected alternative to not target disliked muscle")
	}
}

func TestExerciseRotationService_HelperFunctions(t *testing.T) {
	// Test makePatternRoleKey
	key := makePatternRoleKey("horizontal_push", "compound")
	if key != "horizontal_push:compound" {
		t.Errorf("makePatternRoleKey = %v, want horizontal_push:compound", key)
	}

	// Test splitPatternRoleKey
	parts := splitPatternRoleKey("horizontal_push:compound")
	if len(parts) != 2 || parts[0] != "horizontal_push" || parts[1] != "compound" {
		t.Errorf("splitPatternRoleKey = %v, want [horizontal_push, compound]", parts)
	}

	// Test mapKeysToSlice
	m := map[string]bool{"a": true, "b": true, "c": true}
	keys := mapKeysToSlice(m)
	if len(keys) != 3 {
		t.Errorf("mapKeysToSlice returned %d keys, want 3", len(keys))
	}
}

func TestRotationOptions_Defaults(t *testing.T) {
	opts := RotationOptions{}

	if opts.MaxCandidates != 0 {
		t.Errorf("expected default MaxCandidates to be 0, got %d", opts.MaxCandidates)
	}
	if opts.ExcludeExerciseIDs != nil {
		t.Error("expected ExcludeExerciseIDs to be nil by default")
	}
}

func TestExerciseRotationService_applyHealthRestrictions_NoRestrictions(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()
	service := NewExerciseRotationService(mockRepo, pp)

	exercises := []*entity.ExerciseCatalog{
		{ExerciseID: "ex-1", Pattern: "vertical_push"},
		{ExerciseID: "ex-2", Pattern: "squat"},
	}

	restrictions := entity.HealthRestriction{} // No restrictions

	result := service.applyHealthRestrictions(exercises, restrictions)

	if len(result) != 2 {
		t.Errorf("expected all exercises to pass with no restrictions, got %d", len(result))
	}
}

func TestExerciseRotationService_applyHealthRestrictions_AvoidOverhead(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()
	service := NewExerciseRotationService(mockRepo, pp)

	exercises := []*entity.ExerciseCatalog{
		{ExerciseID: "ex-1", Pattern: "vertical_push"},
		{ExerciseID: "ex-2", Pattern: "horizontal_push"},
		{ExerciseID: "ex-3", Pattern: "overhead_press"},
	}

	restrictions := entity.HealthRestriction{AvoidUpperBodyOverhead: true}

	result := service.applyHealthRestrictions(exercises, restrictions)

	// Should exclude vertical_push and overhead_press
	if len(result) != 1 {
		t.Errorf("expected 1 exercise after filtering overhead, got %d", len(result))
	}
	if result[0].Pattern != "horizontal_push" {
		t.Errorf("expected horizontal_push to remain, got %s", result[0].Pattern)
	}
}

func TestExerciseRotationService_applyHealthRestrictions_AvoidImpact(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()
	service := NewExerciseRotationService(mockRepo, pp)

	exercises := []*entity.ExerciseCatalog{
		{ExerciseID: "ex-1", Pattern: "jump"},
		{ExerciseID: "ex-2", Pattern: "squat"},
		{ExerciseID: "ex-3", Pattern: "plyometric"},
	}

	restrictions := entity.HealthRestriction{AvoidLowerBodyImpact: true}

	result := service.applyHealthRestrictions(exercises, restrictions)

	// Should exclude jump and plyometric
	if len(result) != 1 {
		t.Errorf("expected 1 exercise after filtering impact, got %d", len(result))
	}
	if result[0].Pattern != "squat" {
		t.Errorf("expected squat to remain, got %s", result[0].Pattern)
	}
}

func TestExerciseRotationService_scoreExercises(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()
	service := NewExerciseRotationService(mockRepo, pp)

	exercises := []*entity.ExerciseCatalog{
		{ExerciseID: "ex-1", MainMuscle: "Chest", SecondaryMuscles: []string{"Triceps"}},
		{ExerciseID: "ex-2", MainMuscle: "Shoulders", SecondaryMuscles: []string{"Chest"}},
		{ExerciseID: "ex-3", MainMuscle: "Biceps", SecondaryMuscles: []string{}},
	}

	priorityMuscles := []string{"Chest", "Triceps"}

	scored := service.scoreExercises(exercises, priorityMuscles)

	if len(scored) != 3 {
		t.Errorf("expected 3 scored exercises, got %d", len(scored))
	}

	// ex-1 should have highest score (Chest main + Triceps secondary = 10 + 5 = 15)
	// ex-2 should have score 5 (Chest secondary)
	// ex-3 should have score 0

	if scored[0].score != 15 {
		t.Errorf("expected score 15 for ex-1, got %d", scored[0].score)
	}
	if scored[1].score != 5 {
		t.Errorf("expected score 5 for ex-2, got %d", scored[1].score)
	}
	if scored[2].score != 0 {
		t.Errorf("expected score 0 for ex-3, got %d", scored[2].score)
	}
}

func TestExerciseRotationService_selectFromTopCandidates(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()
	service := NewExerciseRotationServiceWithSeed(mockRepo, pp, 12345)

	exercises := []*entity.ExerciseCatalog{
		{ExerciseID: "ex-1"},
		{ExerciseID: "ex-2"},
		{ExerciseID: "ex-3"},
		{ExerciseID: "ex-4"},
		{ExerciseID: "ex-5"},
	}

	scored := make([]*scoredExercise, len(exercises))
	for i, e := range exercises {
		scored[i] = &scoredExercise{exercise: e, score: 10 - i}
	}

	// With fixed seed, should return consistent result
	result := service.selectFromTopCandidates(scored, 3)

	if result == nil {
		t.Error("expected a result to be selected")
	}
}

func TestExerciseRotationService_selectFromTopCandidates_EmptyList(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()
	service := NewExerciseRotationService(mockRepo, pp)

	result := service.selectFromTopCandidates([]*scoredExercise{}, 5)

	if result != nil {
		t.Error("expected nil for empty list")
	}
}

func TestExerciseRotationService_RotateExercises(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()

	// Add exercises to the mock
	ex1 := &entity.ExerciseCatalog{
		ExerciseID:  "ex-1",
		SpanishName: "Press de banca",
		Pattern:     "horizontal_push",
		Role:        "compound",
		MainMuscle:  "Chest",
	}
	ex2 := &entity.ExerciseCatalog{
		ExerciseID:  "ex-2",
		SpanishName: "Press inclinado",
		Pattern:     "horizontal_push",
		Role:        "compound",
		MainMuscle:  "Chest",
	}
	ex3 := &entity.ExerciseCatalog{
		ExerciseID:  "ex-3",
		SpanishName: "Curl biceps",
		Pattern:     "curl",
		Role:        "isolation",
		MainMuscle:  "Biceps",
	}
	ex4 := &entity.ExerciseCatalog{
		ExerciseID:  "ex-4",
		SpanishName: "Curl martillo",
		Pattern:     "curl",
		Role:        "isolation",
		MainMuscle:  "Biceps",
	}

	mockRepo.addExercise(ex1)
	mockRepo.addExercise(ex2)
	mockRepo.addExercise(ex3)
	mockRepo.addExercise(ex4)

	service := NewExerciseRotationServiceWithSeed(mockRepo, pp, 12345)

	workouts := []*entity.WorkoutExercise{
		{ID: "w-1", UserID: "user-1", ExerciseID: "ex-1", DayName: "Dia 1"},
		{ID: "w-2", UserID: "user-1", ExerciseID: "ex-3", DayName: "Dia 1"},
	}

	profile := &entity.UserGymProfile{
		DislikedExercises: "",
		HealthStatus:      "A",
	}

	rotations, err := service.RotateExercises(context.Background(), workouts, profile)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have generated rotations
	if rotations == nil {
		t.Fatal("expected rotations to be generated")
	}

	// At least some rotations should exist (depends on exercise alternatives)
	t.Logf("Generated %d rotations", len(rotations))
}

func TestExerciseRotationService_RotateExercises_EmptyWorkouts(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()
	service := NewExerciseRotationService(mockRepo, pp)

	profile := &entity.UserGymProfile{}

	rotations, err := service.RotateExercises(context.Background(), []*entity.WorkoutExercise{}, profile)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rotations) != 0 {
		t.Errorf("expected 0 rotations for empty workouts, got %d", len(rotations))
	}
}

func TestExerciseRotationService_RotateExercises_ExerciseNotFound(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()
	service := NewExerciseRotationService(mockRepo, pp)

	// Workout references an exercise that doesn't exist in the mock
	workouts := []*entity.WorkoutExercise{
		{ID: "w-1", UserID: "user-1", ExerciseID: "non-existent", DayName: "Dia 1"},
	}

	profile := &entity.UserGymProfile{}

	rotations, err := service.RotateExercises(context.Background(), workouts, profile)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should handle missing exercise gracefully
	if len(rotations) != 0 {
		t.Errorf("expected 0 rotations for non-existent exercise, got %d", len(rotations))
	}
}

func TestExerciseRotationService_RotateExercises_WithDislikedMuscles(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()

	ex1 := &entity.ExerciseCatalog{
		ExerciseID:  "ex-1",
		SpanishName: "Elevacion talones",
		Pattern:     "calf_raise",
		Role:        "isolation",
		MainMuscle:  "Calfs",
	}
	ex2 := &entity.ExerciseCatalog{
		ExerciseID:  "ex-2",
		SpanishName: "Elevacion sentado",
		Pattern:     "calf_raise",
		Role:        "isolation",
		MainMuscle:  "Calfs",
	}

	mockRepo.addExercise(ex1)
	mockRepo.addExercise(ex2)

	service := NewExerciseRotationService(mockRepo, pp)

	workouts := []*entity.WorkoutExercise{
		{ID: "w-1", UserID: "user-1", ExerciseID: "ex-1", DayName: "Dia 1"},
	}

	profile := &entity.UserGymProfile{
		DislikedExercises: "Pantorrillas", // Dislike calfs
	}

	rotations, err := service.RotateExercises(context.Background(), workouts, profile)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not rotate to another calf exercise since user dislikes calfs
	for _, r := range rotations {
		if r.NewExercise != nil && r.NewExercise.MainMuscle == "Calfs" {
			t.Error("should not rotate to exercise targeting disliked muscle")
		}
	}
}

func TestExerciseRotationService_applyHealthRestrictions_AvoidAxialLoading(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()
	service := NewExerciseRotationService(mockRepo, pp)

	exercises := []*entity.ExerciseCatalog{
		{ExerciseID: "ex-1", Pattern: "squat", Equipment: "barbell"},
		{ExerciseID: "ex-2", Pattern: "deadlift", Equipment: "barbell"},
		{ExerciseID: "ex-3", Pattern: "hip_hinge", Equipment: "machine"},
		{ExerciseID: "ex-4", Pattern: "leg_press", Equipment: "machine"},
	}

	restrictions := entity.HealthRestriction{
		AvoidAxialLoading: true,
		PreferMachines:    true,
	}

	result := service.applyHealthRestrictions(exercises, restrictions)

	// Should filter out squat, deadlift, and potentially hip_hinge based on equipment
	// leg_press should remain as it's machine-based and doesn't have axial loading patterns
	if len(result) == 0 {
		t.Error("expected some exercises to remain after filtering")
	}
}

func TestExerciseRotationService_FindAlternative_WithError(t *testing.T) {
	mockRepo := newMockExerciseReader()
	mockRepo.findError = context.Canceled // Simulate an error
	pp := NewPreferenceProcessor()

	service := NewExerciseRotationService(mockRepo, pp)

	opts := RotationOptions{
		MaxCandidates: 5,
	}

	_, err := service.FindAlternative(context.Background(), "push", "compound", opts)

	if err == nil {
		t.Error("expected error to be returned")
	}
}

func TestExerciseRotationService_RotateExercises_WithError(t *testing.T) {
	mockRepo := newMockExerciseReader()
	pp := NewPreferenceProcessor()

	// Add an exercise first
	mockRepo.addExercise(&entity.ExerciseCatalog{
		ExerciseID:  "ex-1",
		SpanishName: "Press",
		Pattern:     "push",
		Role:        "compound",
		MainMuscle:  "Chest",
	})

	// Then set error for subsequent calls
	mockRepo.findError = context.Canceled

	service := NewExerciseRotationService(mockRepo, pp)

	workouts := []*entity.WorkoutExercise{
		{ID: "w-1", UserID: "user-1", ExerciseID: "ex-1", DayName: "Dia 1"},
	}

	profile := &entity.UserGymProfile{}

	_, err := service.RotateExercises(context.Background(), workouts, profile)

	if err == nil {
		t.Error("expected error to be returned")
	}
}
