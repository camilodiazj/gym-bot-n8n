package postgres

import (
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

func TestNewExerciseRepository(t *testing.T) {
	repo := NewExerciseRepository(nil)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestExerciseCatalog_MatchesMuscle(t *testing.T) {
	exercise := &entity.ExerciseCatalog{
		ExerciseID:       "ex-1",
		SpanishName:      "Press de Banca",
		Pattern:          "horizontal_push",
		Role:             "compound",
		MainMuscle:       "Chest",
		SecondaryMuscles: []string{"Triceps", "Front Delts"},
		Level:            "intermediate",
	}

	tests := []struct {
		name   string
		muscle string
		want   bool
	}{
		{"matches main muscle", "Chest", true},
		{"matches secondary muscle", "Triceps", true},
		{"matches another secondary", "Front Delts", true},
		{"does not match unrelated muscle", "Biceps", false},
		{"does not match empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exercise.MatchesMuscle(tt.muscle)
			if got != tt.want {
				t.Errorf("MatchesMuscle(%s) = %v, want %v", tt.muscle, got, tt.want)
			}
		})
	}
}

func TestExerciseCatalog_MatchesPattern(t *testing.T) {
	exercise := &entity.ExerciseCatalog{
		Pattern: "horizontal_push",
	}

	if !exercise.MatchesPattern("horizontal_push") {
		t.Error("Should match horizontal_push")
	}
	if exercise.MatchesPattern("vertical_push") {
		t.Error("Should not match vertical_push")
	}
}

func TestExerciseCatalog_MatchesRole(t *testing.T) {
	exercise := &entity.ExerciseCatalog{
		Role: "compound",
	}

	if !exercise.MatchesRole("compound") {
		t.Error("Should match compound")
	}
	if exercise.MatchesRole("isolation") {
		t.Error("Should not match isolation")
	}
}

func TestExerciseCatalog_CanRotateTo(t *testing.T) {
	current := &entity.ExerciseCatalog{
		ExerciseID: "ex-1",
		Pattern:    "horizontal_push",
		Role:       "compound",
		MainMuscle: "Chest",
	}

	tests := []struct {
		name            string
		candidate       *entity.ExerciseCatalog
		dislikedMuscles []string
		want            bool
	}{
		{
			name: "can rotate to different exercise same pattern/role",
			candidate: &entity.ExerciseCatalog{
				ExerciseID: "ex-2",
				Pattern:    "horizontal_push",
				Role:       "compound",
				MainMuscle: "Chest",
			},
			dislikedMuscles: []string{},
			want:            true,
		},
		{
			name: "cannot rotate to same exercise",
			candidate: &entity.ExerciseCatalog{
				ExerciseID: "ex-1",
				Pattern:    "horizontal_push",
				Role:       "compound",
				MainMuscle: "Chest",
			},
			dislikedMuscles: []string{},
			want:            false,
		},
		{
			name: "cannot rotate to different pattern",
			candidate: &entity.ExerciseCatalog{
				ExerciseID: "ex-2",
				Pattern:    "vertical_push",
				Role:       "compound",
				MainMuscle: "Shoulders",
			},
			dislikedMuscles: []string{},
			want:            false,
		},
		{
			name: "cannot rotate to different role",
			candidate: &entity.ExerciseCatalog{
				ExerciseID: "ex-2",
				Pattern:    "horizontal_push",
				Role:       "isolation",
				MainMuscle: "Chest",
			},
			dislikedMuscles: []string{},
			want:            false,
		},
		{
			name: "cannot rotate to exercise with disliked muscle",
			candidate: &entity.ExerciseCatalog{
				ExerciseID: "ex-2",
				Pattern:    "horizontal_push",
				Role:       "compound",
				MainMuscle: "Chest",
			},
			dislikedMuscles: []string{"Chest"},
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := current.CanRotateTo(tt.candidate, tt.dislikedMuscles)
			if got != tt.want {
				t.Errorf("CanRotateTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExerciseRotation_WithExerciseNames(t *testing.T) {
	rotation := entity.NewExerciseRotation("w1", "ex1", "ex2", "push", "compound")

	if rotation.OldExerciseName != "" {
		t.Error("Initial OldExerciseName should be empty")
	}
	if rotation.NewExerciseName != "" {
		t.Error("Initial NewExerciseName should be empty")
	}

	rotation.WithExerciseNames("Old Exercise", "New Exercise")

	if rotation.OldExerciseName != "Old Exercise" {
		t.Errorf("OldExerciseName = %s, want Old Exercise", rotation.OldExerciseName)
	}
	if rotation.NewExerciseName != "New Exercise" {
		t.Errorf("NewExerciseName = %s, want New Exercise", rotation.NewExerciseName)
	}
}

func TestNewWorkoutExercise(t *testing.T) {
	we := entity.NewWorkoutExercise(
		"w1", "user1", "Upper A", "ex1",
		"3", "10-12", "2", "3010",
		1, 90, 1,
	)

	if we.ID != "w1" {
		t.Errorf("ID = %s, want w1", we.ID)
	}
	if we.UserID != "user1" {
		t.Errorf("UserID = %s, want user1", we.UserID)
	}
	if we.DayName != "Upper A" {
		t.Errorf("DayName = %s, want Upper A", we.DayName)
	}
	if we.Week != 1 {
		t.Errorf("Week = %d, want 1", we.Week)
	}
	if we.ExerciseOrder != 1 {
		t.Errorf("ExerciseOrder = %d, want 1", we.ExerciseOrder)
	}
}
