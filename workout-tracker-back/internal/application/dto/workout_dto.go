package dto

// TipDTO represents a tip in the response
type TipDTO struct {
	Text string `json:"text"`
}

// StepDTO represents a step in the response
type StepDTO struct {
	Text string `json:"text"`
}

// SetDTO represents a set in the response
type SetDTO struct {
	ID        string `json:"id"`
	SetNumber int    `json:"setNumber"`
	Reps      int    `json:"reps"`
	Kg        string `json:"kg"`
	Completed bool   `json:"completed"`
}

// ExerciseDTO represents an exercise in the response
type ExerciseDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	BadgeColor  string    `json:"badgeColor"`
	RIR         string    `json:"rir"`
	RestSeconds int       `json:"restSeconds"`
	VideoLink   string    `json:"videoLink,omitempty"`
	Sets        []SetDTO  `json:"sets"`
	Tips        []TipDTO  `json:"tips"`
	Steps       []StepDTO `json:"steps"`
}

// TodayWorkoutResponse is the response for GET /api/v1/workouts/today
type TodayWorkoutResponse struct {
	SessionID   string        `json:"session_id"`
	SessionName string        `json:"session_name"`
	Week        int           `json:"week"`
	DayName     string        `json:"day_name"`
	Exercises   []ExerciseDTO `json:"exercises"`
}

// CompleteWorkoutRequest is the request for POST /api/v1/workouts/:workoutId/complete
type CompleteWorkoutRequest struct {
	Notes string `json:"notes,omitempty"`
}

// CompleteWorkoutResponse is the response for POST /api/v1/workouts/:workoutId/complete
type CompleteWorkoutResponse struct {
	Success     bool   `json:"success"`
	WorkoutID   string `json:"workout_id"`
	CompletedAt string `json:"completed_at"`
}
