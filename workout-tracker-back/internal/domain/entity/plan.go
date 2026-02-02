package entity

import "time"

// RenewalType represents the type of renewal operation
type RenewalType string

const (
	RenewalTypeMaintain        RenewalType = "MAINTAIN"
	RenewalTypeRotateExercises RenewalType = "ROTATE_EXERCISES"
	RenewalTypeChangeDays      RenewalType = "CHANGE_DAYS"
	RenewalTypeModifyProfile   RenewalType = "MODIFY_PROFILE"
)

// Plan represents a user's training plan
type Plan struct {
	PlanID          string     `json:"plan_id"`
	UserID          string     `json:"user_id"`
	TemplateID      string     `json:"template_id"`
	WeekSchedule    string     `json:"week_schedule"`
	Goal            string     `json:"goal"`
	Level           string     `json:"level"`
	Status          string     `json:"status"`
	MesocycleNumber int        `json:"mesocycle_number"`
	LastRenewalDate *time.Time `json:"last_renewal_date,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// NewPlan creates a new Plan entity
func NewPlan(planID, userID, templateID, weekSchedule, goal, level string, mesocycleNumber int) *Plan {
	return &Plan{
		PlanID:          planID,
		UserID:          userID,
		TemplateID:      templateID,
		WeekSchedule:    weekSchedule,
		Goal:            goal,
		Level:           level,
		Status:          "active",
		MesocycleNumber: mesocycleNumber,
		CreatedAt:       time.Now(),
	}
}

// IsActive returns true if the plan status is active
func (p *Plan) IsActive() bool {
	return p.Status == "active"
}

// CanRenew returns true if the plan is eligible for renewal
func (p *Plan) CanRenew() bool {
	return p.IsActive()
}

// IncrementMesocycle increments the mesocycle number and updates renewal date
func (p *Plan) IncrementMesocycle() {
	p.MesocycleNumber++
	now := time.Now()
	p.LastRenewalDate = &now
}

// UpdateWeekSchedule updates the weekly training schedule
func (p *Plan) UpdateWeekSchedule(weekSchedule string) {
	p.WeekSchedule = weekSchedule
}

// MesocycleStatus represents the completion status of a user's current mesocycle
type MesocycleStatus struct {
	UserID           string     `json:"user_id"`
	MesocycleNumber  int        `json:"mesocycle_number"`
	DaysPerWeek      int        `json:"days_per_week"`
	WeekSchedule     string     `json:"week_schedule"`
	Week4Completed   int        `json:"week4_completed"`
	Week4Total       int        `json:"week4_total"`
	IsComplete       bool       `json:"is_complete"`
	CompletionRate   float64    `json:"completion_rate"`
	LastRenewalDate  *time.Time `json:"last_renewal_date,omitempty"`
	Goal             string     `json:"goal,omitempty"`
	Level            string     `json:"level,omitempty"`
}

// NewMesocycleStatus creates a new MesocycleStatus with calculated fields
func NewMesocycleStatus(userID string, mesocycleNumber, daysPerWeek, week4Completed int, weekSchedule string) *MesocycleStatus {
	isComplete := week4Completed >= daysPerWeek
	var completionRate float64
	if daysPerWeek > 0 {
		completionRate = float64(week4Completed) / float64(daysPerWeek) * 100
	}
	return &MesocycleStatus{
		UserID:          userID,
		MesocycleNumber: mesocycleNumber,
		DaysPerWeek:     daysPerWeek,
		WeekSchedule:    weekSchedule,
		Week4Completed:  week4Completed,
		Week4Total:      daysPerWeek,
		IsComplete:      isComplete,
		CompletionRate:  completionRate,
	}
}

// RenewalResult represents the result of a mesocycle renewal operation
type RenewalResult struct {
	Success            bool        `json:"success"`
	NewMesocycleNumber int         `json:"new_mesocycle_number"`
	RenewalType        RenewalType `json:"renewal_type"`
	Message            string      `json:"message"`
	ScheduleCleared    bool        `json:"schedule_cleared"`
	WorkoutsUpdated    int         `json:"workouts_updated,omitempty"`
	ExercisesRotated   int         `json:"exercises_rotated,omitempty"`
	NewWeekSchedule    string      `json:"new_week_schedule,omitempty"`
}

// NewRenewalResult creates a new successful RenewalResult
func NewRenewalResult(mesocycleNumber int, renewalType RenewalType, message string) *RenewalResult {
	return &RenewalResult{
		Success:            true,
		NewMesocycleNumber: mesocycleNumber,
		RenewalType:        renewalType,
		Message:            message,
		ScheduleCleared:    true,
	}
}

// WithExercisesRotated sets the number of exercises rotated
func (r *RenewalResult) WithExercisesRotated(count int) *RenewalResult {
	r.ExercisesRotated = count
	return r
}

// WithNewWeekSchedule sets the new week schedule
func (r *RenewalResult) WithNewWeekSchedule(schedule string) *RenewalResult {
	r.NewWeekSchedule = schedule
	return r
}
