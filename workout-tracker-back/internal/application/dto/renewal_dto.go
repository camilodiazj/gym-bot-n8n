package dto

import (
	"errors"
	"time"
)

// ==================== REQUEST DTOs ====================

// CheckMesocycleStatusRequest is the request for checking mesocycle status
// User ID comes from path parameter, so this is empty
type CheckMesocycleStatusRequest struct{}

// RenewMaintainRequest is the request for maintaining the same routine
type RenewMaintainRequest struct {
	Notes string `json:"notes,omitempty"`
}

// RenewRotateExercisesRequest is the request for rotating exercises
type RenewRotateExercisesRequest struct {
	RotateCompounds bool `json:"rotate_compounds"` // Default true
	RotateIsolation bool `json:"rotate_isolation"` // Default true
	RotateCore      bool `json:"rotate_core"`      // Default true
}

// RenewChangeDaysRequest is the request for changing training days
type RenewChangeDaysRequest struct {
	NewDaysPerWeek int    `json:"new_days_per_week" binding:"required,min=2,max=6"`
	Notes          string `json:"notes,omitempty"`
}

// Validate validates the RenewChangeDaysRequest
func (r *RenewChangeDaysRequest) Validate() error {
	if r.NewDaysPerWeek < 2 || r.NewDaysPerWeek > 6 {
		return ErrInvalidDaysPerWeek
	}
	return nil
}

// UpdateProfileRequest is the request for updating user preferences
type UpdateProfileRequest struct {
	PriorityMuscles *string `json:"priority_muscles,omitempty"`
	HealthStatus    *string `json:"health_status,omitempty"`
	SessionDuration *string `json:"session_duration,omitempty"`
	Notes           string  `json:"notes,omitempty"`
}

// ==================== RESPONSE DTOs ====================

// MesocycleStatusResponse is the response for GET /plans/:userId/mesocycle-status
type MesocycleStatusResponse struct {
	UserID          string     `json:"user_id"`
	MesocycleNumber int        `json:"mesocycle_number"`
	DaysPerWeek     int        `json:"days_per_week"`
	WeekSchedule    string     `json:"week_schedule"`
	Week4Completed  int        `json:"week4_completed"`
	Week4Total      int        `json:"week4_total"`
	IsComplete      bool       `json:"is_complete"`
	CompletionRate  float64    `json:"completion_rate"`
	LastRenewalDate *time.Time `json:"last_renewal_date,omitempty"`
	Goal            string     `json:"goal"`
	Level           string     `json:"level"`
	CanRenew        bool       `json:"can_renew"`
	Message         string     `json:"message"`
}

// RenewalResponse is the common response for all renewal operations
type RenewalResponse struct {
	Success            bool   `json:"success"`
	RenewalType        string `json:"renewal_type"`
	NewMesocycleNumber int    `json:"new_mesocycle_number"`
	Message            string `json:"message"`
	ScheduleCleared    bool   `json:"schedule_cleared"`
	WorkoutsUpdated    int    `json:"workouts_updated,omitempty"`
	ExercisesRotated   int    `json:"exercises_rotated,omitempty"`
	NewWeekSchedule    string `json:"new_week_schedule,omitempty"`
}

// ExerciseRotationDTO represents a single exercise rotation in the response
type ExerciseRotationDTO struct {
	WorkoutID       string `json:"workout_id"`
	OldExerciseID   string `json:"old_exercise_id"`
	OldExerciseName string `json:"old_exercise_name"`
	NewExerciseID   string `json:"new_exercise_id"`
	NewExerciseName string `json:"new_exercise_name"`
	Pattern         string `json:"pattern"`
	Role            string `json:"role"`
}

// RotateExercisesResponse extends RenewalResponse with rotation details
type RotateExercisesResponse struct {
	RenewalResponse
	Rotations []ExerciseRotationDTO `json:"rotations,omitempty"`
}

// ChangeDaysResponse extends RenewalResponse with schedule change details
type ChangeDaysResponse struct {
	RenewalResponse
	OldDaysPerWeek       int  `json:"old_days_per_week"`
	NewDaysPerWeek       int  `json:"new_days_per_week"`
	RequiresRegeneration bool `json:"requires_regeneration"`
}

// UpdateProfileResponse is the response for POST /plans/:userId/renew/update-profile
type UpdateProfileResponse struct {
	Success              bool   `json:"success"`
	Message              string `json:"message"`
	NewMesocycleNumber   int    `json:"new_mesocycle_number"`
	RequiresRegeneration bool   `json:"requires_regeneration"`
	ProfileUpdated       bool   `json:"profile_updated"`
	WorkoutsDeleted      bool   `json:"workouts_deleted"`
	ScheduleCleared      bool   `json:"schedule_cleared"`
}

// ==================== ERROR TYPES ====================

// Validation errors
var (
	ErrInvalidDaysPerWeek = errors.New("new_days_per_week must be between 2 and 6")
)

// RenewalErrorResponse represents an error response for renewal operations
type RenewalErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code"`
}
