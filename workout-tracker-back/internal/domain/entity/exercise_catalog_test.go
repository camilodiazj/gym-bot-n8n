package entity

import (
	"testing"
)

func TestNewExerciseCatalog(t *testing.T) {
	exercise := NewExerciseCatalog(
		"ex-123",
		"Press de banca",
		"horizontal_push",
		"compound",
		"Chest",
		"intermediate",
		"https://example.com/video",
		"barbell",
		[]string{"Triceps", "Front Delts"},
	)

	if exercise.ExerciseID != "ex-123" {
		t.Errorf("expected ExerciseID 'ex-123', got '%s'", exercise.ExerciseID)
	}
	if exercise.SpanishName != "Press de banca" {
		t.Errorf("expected SpanishName 'Press de banca', got '%s'", exercise.SpanishName)
	}
	if exercise.Pattern != "horizontal_push" {
		t.Errorf("expected Pattern 'horizontal_push', got '%s'", exercise.Pattern)
	}
	if exercise.Role != "compound" {
		t.Errorf("expected Role 'compound', got '%s'", exercise.Role)
	}
	if exercise.MainMuscle != "Chest" {
		t.Errorf("expected MainMuscle 'Chest', got '%s'", exercise.MainMuscle)
	}
	if len(exercise.SecondaryMuscles) != 2 {
		t.Errorf("expected 2 secondary muscles, got %d", len(exercise.SecondaryMuscles))
	}
}

func TestExerciseCatalog_MatchesMuscle(t *testing.T) {
	exercise := &ExerciseCatalog{
		MainMuscle:       "Chest",
		SecondaryMuscles: []string{"Triceps", "Front Delts"},
	}

	tests := []struct {
		name     string
		muscle   string
		expected bool
	}{
		{"main muscle", "Chest", true},
		{"secondary muscle 1", "Triceps", true},
		{"secondary muscle 2", "Front Delts", true},
		{"unrelated muscle", "Biceps", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exercise.MatchesMuscle(tt.muscle); got != tt.expected {
				t.Errorf("MatchesMuscle(%q) = %v, want %v", tt.muscle, got, tt.expected)
			}
		})
	}
}

func TestExerciseCatalog_MatchesPattern(t *testing.T) {
	exercise := &ExerciseCatalog{Pattern: "horizontal_push"}

	if !exercise.MatchesPattern("horizontal_push") {
		t.Error("expected exercise to match pattern 'horizontal_push'")
	}
	if exercise.MatchesPattern("vertical_push") {
		t.Error("expected exercise to not match pattern 'vertical_push'")
	}
}

func TestExerciseCatalog_MatchesRole(t *testing.T) {
	exercise := &ExerciseCatalog{Role: "compound"}

	if !exercise.MatchesRole("compound") {
		t.Error("expected exercise to match role 'compound'")
	}
	if exercise.MatchesRole("isolation") {
		t.Error("expected exercise to not match role 'isolation'")
	}
}

func TestExerciseCatalog_CanRotateTo(t *testing.T) {
	current := &ExerciseCatalog{
		ExerciseID: "ex-1",
		Pattern:    "horizontal_push",
		Role:       "compound",
		MainMuscle: "Chest",
	}

	tests := []struct {
		name           string
		candidate      *ExerciseCatalog
		dislikedMuscles []string
		expected       bool
	}{
		{
			name: "valid rotation - same pattern and role",
			candidate: &ExerciseCatalog{
				ExerciseID: "ex-2",
				Pattern:    "horizontal_push",
				Role:       "compound",
				MainMuscle: "Chest",
			},
			dislikedMuscles: []string{},
			expected:        true,
		},
		{
			name: "invalid - same exercise",
			candidate: &ExerciseCatalog{
				ExerciseID: "ex-1",
				Pattern:    "horizontal_push",
				Role:       "compound",
				MainMuscle: "Chest",
			},
			dislikedMuscles: []string{},
			expected:        false,
		},
		{
			name: "invalid - different pattern",
			candidate: &ExerciseCatalog{
				ExerciseID: "ex-2",
				Pattern:    "vertical_push",
				Role:       "compound",
				MainMuscle: "Shoulders",
			},
			dislikedMuscles: []string{},
			expected:        false,
		},
		{
			name: "invalid - different role",
			candidate: &ExerciseCatalog{
				ExerciseID: "ex-2",
				Pattern:    "horizontal_push",
				Role:       "isolation",
				MainMuscle: "Chest",
			},
			dislikedMuscles: []string{},
			expected:        false,
		},
		{
			name: "invalid - targets disliked muscle",
			candidate: &ExerciseCatalog{
				ExerciseID: "ex-2",
				Pattern:    "horizontal_push",
				Role:       "compound",
				MainMuscle: "Chest",
			},
			dislikedMuscles: []string{"Chest"},
			expected:        false,
		},
		{
			name: "valid - disliked muscle not targeted",
			candidate: &ExerciseCatalog{
				ExerciseID: "ex-2",
				Pattern:    "horizontal_push",
				Role:       "compound",
				MainMuscle: "Chest",
			},
			dislikedMuscles: []string{"Calfs", "Biceps"},
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := current.CanRotateTo(tt.candidate, tt.dislikedMuscles); got != tt.expected {
				t.Errorf("CanRotateTo() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewExerciseRotation(t *testing.T) {
	rotation := NewExerciseRotation(
		"workout-123",
		"old-ex-1",
		"new-ex-2",
		"horizontal_push",
		"compound",
	)

	if rotation.WorkoutID != "workout-123" {
		t.Errorf("expected WorkoutID 'workout-123', got '%s'", rotation.WorkoutID)
	}
	if rotation.OldExerciseID != "old-ex-1" {
		t.Errorf("expected OldExerciseID 'old-ex-1', got '%s'", rotation.OldExerciseID)
	}
	if rotation.NewExerciseID != "new-ex-2" {
		t.Errorf("expected NewExerciseID 'new-ex-2', got '%s'", rotation.NewExerciseID)
	}
	if rotation.Pattern != "horizontal_push" {
		t.Errorf("expected Pattern 'horizontal_push', got '%s'", rotation.Pattern)
	}
	if rotation.Role != "compound" {
		t.Errorf("expected Role 'compound', got '%s'", rotation.Role)
	}
}

func TestExerciseRotation_WithExerciseNames(t *testing.T) {
	rotation := NewExerciseRotation("w-1", "ex-1", "ex-2", "push", "compound").
		WithExerciseNames("Press de banca", "Press inclinado")

	if rotation.OldExerciseName != "Press de banca" {
		t.Errorf("expected OldExerciseName 'Press de banca', got '%s'", rotation.OldExerciseName)
	}
	if rotation.NewExerciseName != "Press inclinado" {
		t.Errorf("expected NewExerciseName 'Press inclinado', got '%s'", rotation.NewExerciseName)
	}
}

func TestNewWorkoutExercise(t *testing.T) {
	workout := NewWorkoutExercise(
		"w-123",
		"user-456",
		"Dia 1",
		"ex-789",
		"4",
		"8-10",
		"2-3",
		"2-0-1-0",
		1,
		90,
		1,
	)

	if workout.ID != "w-123" {
		t.Errorf("expected ID 'w-123', got '%s'", workout.ID)
	}
	if workout.UserID != "user-456" {
		t.Errorf("expected UserID 'user-456', got '%s'", workout.UserID)
	}
	if workout.DayName != "Dia 1" {
		t.Errorf("expected DayName 'Dia 1', got '%s'", workout.DayName)
	}
	if workout.Week != 1 {
		t.Errorf("expected Week 1, got %d", workout.Week)
	}
	if workout.ExerciseOrder != 1 {
		t.Errorf("expected ExerciseOrder 1, got %d", workout.ExerciseOrder)
	}
}

func TestWorkoutExercise_PatternRoleKey(t *testing.T) {
	workout := &WorkoutExercise{}
	key := workout.PatternRoleKey("horizontal_push", "compound")

	if key != "horizontal_push:compound" {
		t.Errorf("expected key 'horizontal_push:compound', got '%s'", key)
	}
}
