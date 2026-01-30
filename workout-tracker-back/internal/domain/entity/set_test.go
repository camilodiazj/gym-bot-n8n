package entity

import (
	"testing"
)

func TestNewSet(t *testing.T) {
	set := NewSet("set-1", "ex-1", 1, 12, "50")

	if set.ID != "set-1" {
		t.Errorf("Expected ID set-1, got %s", set.ID)
	}
	if set.ExerciseID != "ex-1" {
		t.Errorf("Expected ExerciseID ex-1, got %s", set.ExerciseID)
	}
	if set.SetNumber != 1 {
		t.Errorf("Expected SetNumber 1, got %d", set.SetNumber)
	}
	if set.Reps != 12 {
		t.Errorf("Expected Reps 12, got %d", set.Reps)
	}
	if set.Weight != "50" {
		t.Errorf("Expected Weight 50, got %s", set.Weight)
	}
	if set.Completed {
		t.Error("Expected Completed to be false")
	}
	if set.CompletedAt != nil {
		t.Error("Expected CompletedAt to be nil")
	}
}

func TestSet_MarkComplete(t *testing.T) {
	set := NewSet("set-1", "ex-1", 1, 12, "50")

	if set.Completed {
		t.Error("Expected set to not be completed initially")
	}
	if set.CompletedAt != nil {
		t.Error("Expected CompletedAt to be nil initially")
	}

	set.MarkComplete()

	if !set.Completed {
		t.Error("Expected set to be completed")
	}
	if set.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestSet_UpdateReps(t *testing.T) {
	set := NewSet("set-1", "ex-1", 1, 12, "50")

	if set.Reps != 12 {
		t.Errorf("Expected initial Reps 12, got %d", set.Reps)
	}

	set.UpdateReps(15)

	if set.Reps != 15 {
		t.Errorf("Expected Reps 15, got %d", set.Reps)
	}
}

func TestSet_UpdateWeight(t *testing.T) {
	set := NewSet("set-1", "ex-1", 1, 12, "50")

	if set.Weight != "50" {
		t.Errorf("Expected initial Weight 50, got %s", set.Weight)
	}

	set.UpdateWeight("60")

	if set.Weight != "60" {
		t.Errorf("Expected Weight 60, got %s", set.Weight)
	}

	// Test bodyweight exercise
	set.UpdateWeight("-")
	if set.Weight != "-" {
		t.Errorf("Expected Weight -, got %s", set.Weight)
	}
}
