package entity

import "time"

// Workout represents a user's workout session for a specific day
type Workout struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Week        int        `json:"week"`
	DayName     string     `json:"day_name"`
	SessionName string     `json:"session_name"`
	PlannedDay  time.Time  `json:"planned_day"`
	Exercises   []Exercise `json:"exercises"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// NewWorkout creates a new Workout entity
func NewWorkout(id, userID string, week int, dayName, sessionName string, plannedDay time.Time) *Workout {
	return &Workout{
		ID:          id,
		UserID:      userID,
		Week:        week,
		DayName:     dayName,
		SessionName: sessionName,
		PlannedDay:  plannedDay,
		Exercises:   []Exercise{},
		Completed:   false,
	}
}

// AddExercise adds an exercise to the workout
func (w *Workout) AddExercise(exercise Exercise) {
	w.Exercises = append(w.Exercises, exercise)
}

// MarkComplete marks the workout as completed
func (w *Workout) MarkComplete() {
	now := time.Now()
	w.Completed = true
	w.CompletedAt = &now
}

// AllExercisesCompleted returns true if all exercises have all sets completed
func (w *Workout) AllExercisesCompleted() bool {
	if len(w.Exercises) == 0 {
		return false
	}
	for _, exercise := range w.Exercises {
		if !exercise.AllSetsCompleted() {
			return false
		}
	}
	return true
}

// TotalSets returns the total number of sets in the workout
func (w *Workout) TotalSets() int {
	total := 0
	for _, exercise := range w.Exercises {
		total += len(exercise.Sets)
	}
	return total
}

// CompletedSets returns the total number of completed sets
func (w *Workout) CompletedSets() int {
	completed := 0
	for _, exercise := range w.Exercises {
		completed += exercise.CompletedSetsCount()
	}
	return completed
}
