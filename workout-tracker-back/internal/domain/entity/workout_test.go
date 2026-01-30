package entity

import (
	"testing"
	"time"
)

func TestNewWorkout(t *testing.T) {
	id := "workout-123"
	userID := "user-456"
	week := 2
	dayName := "Monday"
	sessionName := "Leg Day"
	plannedDay := time.Now()

	workout := NewWorkout(id, userID, week, dayName, sessionName, plannedDay)

	if workout.ID != id {
		t.Errorf("Expected ID %s, got %s", id, workout.ID)
	}
	if workout.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, workout.UserID)
	}
	if workout.Week != week {
		t.Errorf("Expected Week %d, got %d", week, workout.Week)
	}
	if workout.DayName != dayName {
		t.Errorf("Expected DayName %s, got %s", dayName, workout.DayName)
	}
	if workout.SessionName != sessionName {
		t.Errorf("Expected SessionName %s, got %s", sessionName, workout.SessionName)
	}
	if workout.Completed {
		t.Error("Expected Completed to be false")
	}
	if len(workout.Exercises) != 0 {
		t.Errorf("Expected empty Exercises, got %d", len(workout.Exercises))
	}
}

func TestWorkout_AddExercise(t *testing.T) {
	workout := NewWorkout("w1", "u1", 1, "Mon", "Session", time.Now())
	exercise := *NewExercise("e1", "Squat", "#374151", "2-3", 90, "")

	workout.AddExercise(exercise)

	if len(workout.Exercises) != 1 {
		t.Errorf("Expected 1 exercise, got %d", len(workout.Exercises))
	}
	if workout.Exercises[0].ID != "e1" {
		t.Errorf("Expected exercise ID e1, got %s", workout.Exercises[0].ID)
	}
}

func TestWorkout_MarkComplete(t *testing.T) {
	workout := NewWorkout("w1", "u1", 1, "Mon", "Session", time.Now())

	if workout.Completed {
		t.Error("Expected workout to not be completed initially")
	}
	if workout.CompletedAt != nil {
		t.Error("Expected CompletedAt to be nil initially")
	}

	workout.MarkComplete()

	if !workout.Completed {
		t.Error("Expected workout to be completed")
	}
	if workout.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestWorkout_AllExercisesCompleted(t *testing.T) {
	// Empty workout
	workout := NewWorkout("w1", "u1", 1, "Mon", "Session", time.Now())
	if workout.AllExercisesCompleted() {
		t.Error("Empty workout should not be considered complete")
	}

	// Workout with incomplete exercise
	exercise := *NewExercise("e1", "Squat", "#374151", "2-3", 90, "")
	set := *NewSet("s1", "e1", 1, 12, "50")
	exercise.AddSet(set)
	workout.AddExercise(exercise)

	if workout.AllExercisesCompleted() {
		t.Error("Workout with incomplete sets should not be complete")
	}

	// Complete all sets
	workout.Exercises[0].Sets[0].MarkComplete()
	if !workout.AllExercisesCompleted() {
		t.Error("Workout with all completed sets should be complete")
	}
}

func TestWorkout_TotalSets(t *testing.T) {
	workout := NewWorkout("w1", "u1", 1, "Mon", "Session", time.Now())

	if workout.TotalSets() != 0 {
		t.Errorf("Expected 0 total sets, got %d", workout.TotalSets())
	}

	exercise := *NewExercise("e1", "Squat", "#374151", "2-3", 90, "")
	exercise.AddSet(*NewSet("s1", "e1", 1, 12, "50"))
	exercise.AddSet(*NewSet("s2", "e1", 2, 12, "50"))
	workout.AddExercise(exercise)

	if workout.TotalSets() != 2 {
		t.Errorf("Expected 2 total sets, got %d", workout.TotalSets())
	}

	exercise2 := *NewExercise("e2", "Press", "#374151", "3-4", 120, "")
	exercise2.AddSet(*NewSet("s3", "e2", 1, 10, "30"))
	workout.AddExercise(exercise2)

	if workout.TotalSets() != 3 {
		t.Errorf("Expected 3 total sets, got %d", workout.TotalSets())
	}
}

func TestWorkout_CompletedSets(t *testing.T) {
	workout := NewWorkout("w1", "u1", 1, "Mon", "Session", time.Now())

	if workout.CompletedSets() != 0 {
		t.Errorf("Expected 0 completed sets, got %d", workout.CompletedSets())
	}

	exercise := *NewExercise("e1", "Squat", "#374151", "2-3", 90, "")
	set1 := NewSet("s1", "e1", 1, 12, "50")
	set2 := NewSet("s2", "e1", 2, 12, "50")
	exercise.AddSet(*set1)
	exercise.AddSet(*set2)
	workout.AddExercise(exercise)

	if workout.CompletedSets() != 0 {
		t.Errorf("Expected 0 completed sets, got %d", workout.CompletedSets())
	}

	workout.Exercises[0].Sets[0].MarkComplete()
	if workout.CompletedSets() != 1 {
		t.Errorf("Expected 1 completed set, got %d", workout.CompletedSets())
	}

	workout.Exercises[0].Sets[1].MarkComplete()
	if workout.CompletedSets() != 2 {
		t.Errorf("Expected 2 completed sets, got %d", workout.CompletedSets())
	}
}
