package postgres

import (
	"testing"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

func TestNewWorkoutRenewalRepository(t *testing.T) {
	repo := NewWorkoutRenewalRepository(nil)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestWorkoutExercise_PatternRoleKey(t *testing.T) {
	we := &entity.WorkoutExercise{
		ID:         "w1",
		ExerciseID: "ex1",
	}

	key := we.PatternRoleKey("horizontal_push", "compound")
	expected := "horizontal_push:compound"

	if key != expected {
		t.Errorf("PatternRoleKey = %s, want %s", key, expected)
	}
}

func TestExerciseRotation_Creation(t *testing.T) {
	rotation := entity.NewExerciseRotation(
		"workout-1",
		"old-ex-1",
		"new-ex-1",
		"horizontal_push",
		"compound",
	)

	if rotation.WorkoutID != "workout-1" {
		t.Errorf("WorkoutID = %s, want workout-1", rotation.WorkoutID)
	}
	if rotation.OldExerciseID != "old-ex-1" {
		t.Errorf("OldExerciseID = %s, want old-ex-1", rotation.OldExerciseID)
	}
	if rotation.NewExerciseID != "new-ex-1" {
		t.Errorf("NewExerciseID = %s, want new-ex-1", rotation.NewExerciseID)
	}
	if rotation.Pattern != "horizontal_push" {
		t.Errorf("Pattern = %s, want horizontal_push", rotation.Pattern)
	}
	if rotation.Role != "compound" {
		t.Errorf("Role = %s, want compound", rotation.Role)
	}
}
