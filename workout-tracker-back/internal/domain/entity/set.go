package entity

import "time"

// Set represents a single set within an exercise
type Set struct {
	ID          string    `json:"id"`
	ExerciseID  string    `json:"exercise_id"`
	SetNumber   int       `json:"set_number"`
	Reps        int       `json:"reps"`
	Weight      string    `json:"kg"` // "-" for bodyweight exercises
	Completed   bool      `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// NewSet creates a new Set entity
func NewSet(id, exerciseID string, setNumber, reps int, weight string) *Set {
	return &Set{
		ID:         id,
		ExerciseID: exerciseID,
		SetNumber:  setNumber,
		Reps:       reps,
		Weight:     weight,
		Completed:  false,
	}
}

// MarkComplete marks the set as completed
func (s *Set) MarkComplete() {
	now := time.Now()
	s.Completed = true
	s.CompletedAt = &now
}

// UpdateReps updates the reps for the set
func (s *Set) UpdateReps(reps int) {
	s.Reps = reps
}

// UpdateWeight updates the weight for the set
func (s *Set) UpdateWeight(weight string) {
	s.Weight = weight
}
