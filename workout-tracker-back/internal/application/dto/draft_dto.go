package dto

// DraftAlternativeDTO represents an alternative exercise in the draft response
type DraftAlternativeDTO struct {
	ExerciseID  string `json:"exercise_id"`
	SpanishName string `json:"spanish_name"`
	MainMuscle  string `json:"main_muscle,omitempty"`
	VideoLink   string `json:"video_link,omitempty"`
}

// DraftExerciseDTO represents an exercise in the draft response
type DraftExerciseDTO struct {
	ExerciseID    string                `json:"exercise_id"`
	SpanishName   string                `json:"spanish_name"`
	Pattern       string                `json:"pattern"`
	Role          string                `json:"role"`
	MainMuscle    string                `json:"main_muscle,omitempty"`
	VideoLink     string                `json:"video_link,omitempty"`
	Sets          int                   `json:"sets"`
	Reps          string                `json:"reps"`
	RIR           string                `json:"rir"`
	RestSeconds   int                   `json:"rest_seconds"`
	ExerciseOrder int                   `json:"exercise_order"`
	Alternatives  []DraftAlternativeDTO `json:"alternatives"`
}

// DraftDayDTO represents a training day in the draft response
type DraftDayDTO struct {
	DayNumber int                `json:"day_number"`
	Title     string             `json:"title"`
	Exercises []DraftExerciseDTO `json:"exercises"`
}

// GetDraftResponse is the response for GET /api/v1/drafts/:code
type GetDraftResponse struct {
	DraftID      string        `json:"draft_id"`
	Code         string        `json:"code"`
	WeekSchedule string        `json:"week_schedule"`
	Goal         string        `json:"goal"`
	Level        string        `json:"level"`
	Days         []DraftDayDTO `json:"days"`
}

// SwapExerciseRequest is the request for PATCH /api/v1/drafts/:code/swap
type SwapExerciseRequest struct {
	DayNumber     int    `json:"day_number" binding:"required"`
	ExerciseOrder int    `json:"exercise_order" binding:"required"`
	NewExerciseID string `json:"new_exercise_id" binding:"required"`
}

// SwapExerciseResponse is the response for PATCH /api/v1/drafts/:code/swap
type SwapExerciseResponse struct {
	Exercise DraftExerciseDTO `json:"exercise"`
}

// ApproveDraftResponse is the response for POST /api/v1/drafts/:code/approve
type ApproveDraftResponse struct {
	PhoneNumber string `json:"phone_number"`
}
