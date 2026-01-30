package entity

import (
	"testing"
)

func TestNewExercise(t *testing.T) {
	exercise := NewExercise("ex-1", "Squat", "#374151", "2-3", 90, "https://example.com/video")

	if exercise.ID != "ex-1" {
		t.Errorf("Expected ID ex-1, got %s", exercise.ID)
	}
	if exercise.Name != "Squat" {
		t.Errorf("Expected Name Squat, got %s", exercise.Name)
	}
	if exercise.BadgeColor != "#374151" {
		t.Errorf("Expected BadgeColor #374151, got %s", exercise.BadgeColor)
	}
	if exercise.RIR != "2-3" {
		t.Errorf("Expected RIR 2-3, got %s", exercise.RIR)
	}
	if exercise.RestSeconds != 90 {
		t.Errorf("Expected RestSeconds 90, got %d", exercise.RestSeconds)
	}
	if exercise.Link != "https://example.com/video" {
		t.Errorf("Expected Link, got %s", exercise.Link)
	}
	if len(exercise.Sets) != 0 {
		t.Errorf("Expected empty Sets, got %d", len(exercise.Sets))
	}
	if len(exercise.Tips) != 0 {
		t.Errorf("Expected empty Tips, got %d", len(exercise.Tips))
	}
	if len(exercise.Steps) != 0 {
		t.Errorf("Expected empty Steps, got %d", len(exercise.Steps))
	}
}

func TestExercise_AddSet(t *testing.T) {
	exercise := NewExercise("ex-1", "Squat", "#374151", "2-3", 90, "")
	set := *NewSet("s1", "ex-1", 1, 12, "50")

	exercise.AddSet(set)

	if len(exercise.Sets) != 1 {
		t.Errorf("Expected 1 set, got %d", len(exercise.Sets))
	}
	if exercise.Sets[0].ID != "s1" {
		t.Errorf("Expected set ID s1, got %s", exercise.Sets[0].ID)
	}
}

func TestExercise_AddTip(t *testing.T) {
	exercise := NewExercise("ex-1", "Squat", "#374151", "2-3", 90, "")
	tip := Tip{Text: "Keep your back straight"}

	exercise.AddTip(tip)

	if len(exercise.Tips) != 1 {
		t.Errorf("Expected 1 tip, got %d", len(exercise.Tips))
	}
	if exercise.Tips[0].Text != "Keep your back straight" {
		t.Errorf("Expected tip text, got %s", exercise.Tips[0].Text)
	}
}

func TestExercise_AddStep(t *testing.T) {
	exercise := NewExercise("ex-1", "Squat", "#374151", "2-3", 90, "")
	step := Step{Text: "Stand with feet shoulder-width apart"}

	exercise.AddStep(step)

	if len(exercise.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(exercise.Steps))
	}
	if exercise.Steps[0].Text != "Stand with feet shoulder-width apart" {
		t.Errorf("Expected step text, got %s", exercise.Steps[0].Text)
	}
}

func TestExercise_AllSetsCompleted(t *testing.T) {
	// Empty exercise
	exercise := NewExercise("ex-1", "Squat", "#374151", "2-3", 90, "")
	if exercise.AllSetsCompleted() {
		t.Error("Exercise with no sets should not be considered complete")
	}

	// Add incomplete set
	set := NewSet("s1", "ex-1", 1, 12, "50")
	exercise.AddSet(*set)

	if exercise.AllSetsCompleted() {
		t.Error("Exercise with incomplete set should not be complete")
	}

	// Complete the set
	exercise.Sets[0].MarkComplete()
	if !exercise.AllSetsCompleted() {
		t.Error("Exercise with all completed sets should be complete")
	}

	// Add another incomplete set
	set2 := NewSet("s2", "ex-1", 2, 12, "50")
	exercise.AddSet(*set2)

	if exercise.AllSetsCompleted() {
		t.Error("Exercise with one incomplete set should not be complete")
	}
}

func TestExercise_CompletedSetsCount(t *testing.T) {
	exercise := NewExercise("ex-1", "Squat", "#374151", "2-3", 90, "")

	if exercise.CompletedSetsCount() != 0 {
		t.Errorf("Expected 0 completed sets, got %d", exercise.CompletedSetsCount())
	}

	set1 := NewSet("s1", "ex-1", 1, 12, "50")
	set2 := NewSet("s2", "ex-1", 2, 12, "50")
	set3 := NewSet("s3", "ex-1", 3, 12, "50")

	exercise.AddSet(*set1)
	exercise.AddSet(*set2)
	exercise.AddSet(*set3)

	if exercise.CompletedSetsCount() != 0 {
		t.Errorf("Expected 0 completed sets, got %d", exercise.CompletedSetsCount())
	}

	exercise.Sets[0].MarkComplete()
	if exercise.CompletedSetsCount() != 1 {
		t.Errorf("Expected 1 completed set, got %d", exercise.CompletedSetsCount())
	}

	exercise.Sets[2].MarkComplete()
	if exercise.CompletedSetsCount() != 2 {
		t.Errorf("Expected 2 completed sets, got %d", exercise.CompletedSetsCount())
	}
}
