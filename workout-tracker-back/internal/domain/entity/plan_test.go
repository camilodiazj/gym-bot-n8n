package entity

import (
	"testing"
	"time"
)

func TestNewPlan(t *testing.T) {
	plan := NewPlan(
		"plan-123",
		"user-456",
		"template-789",
		"fb_3",
		"Ganar masa muscular",
		"Intermedio",
		1,
	)

	if plan.PlanID != "plan-123" {
		t.Errorf("expected PlanID to be 'plan-123', got '%s'", plan.PlanID)
	}
	if plan.UserID != "user-456" {
		t.Errorf("expected UserID to be 'user-456', got '%s'", plan.UserID)
	}
	if plan.Status != "active" {
		t.Errorf("expected Status to be 'active', got '%s'", plan.Status)
	}
	if plan.MesocycleNumber != 1 {
		t.Errorf("expected MesocycleNumber to be 1, got %d", plan.MesocycleNumber)
	}
}

func TestPlan_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"active plan", "active", true},
		{"inactive plan", "inactive", false},
		{"completed plan", "completed", false},
		{"paused plan", "paused", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &Plan{Status: tt.status}
			if got := plan.IsActive(); got != tt.expected {
				t.Errorf("IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlan_CanRenew(t *testing.T) {
	activePlan := &Plan{Status: "active"}
	inactivePlan := &Plan{Status: "inactive"}

	if !activePlan.CanRenew() {
		t.Error("expected active plan to be renewable")
	}
	if inactivePlan.CanRenew() {
		t.Error("expected inactive plan to not be renewable")
	}
}

func TestPlan_IncrementMesocycle(t *testing.T) {
	plan := &Plan{MesocycleNumber: 1}

	plan.IncrementMesocycle()

	if plan.MesocycleNumber != 2 {
		t.Errorf("expected MesocycleNumber to be 2, got %d", plan.MesocycleNumber)
	}
	if plan.LastRenewalDate == nil {
		t.Error("expected LastRenewalDate to be set")
	}
	// Check that the renewal date is recent (within last second)
	if time.Since(*plan.LastRenewalDate) > time.Second {
		t.Error("expected LastRenewalDate to be set to current time")
	}
}

func TestPlan_UpdateWeekSchedule(t *testing.T) {
	plan := &Plan{WeekSchedule: "fb_3"}

	plan.UpdateWeekSchedule("ul_4")

	if plan.WeekSchedule != "ul_4" {
		t.Errorf("expected WeekSchedule to be 'ul_4', got '%s'", plan.WeekSchedule)
	}
}

func TestNewMesocycleStatus(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mesocycle      int
		daysPerWeek    int
		week4Completed int
		weekSchedule   string
		wantComplete   bool
		wantRate       float64
	}{
		{
			name:           "completed mesocycle",
			userID:         "user-1",
			mesocycle:      1,
			daysPerWeek:    3,
			week4Completed: 3,
			weekSchedule:   "fb_3",
			wantComplete:   true,
			wantRate:       100.0,
		},
		{
			name:           "incomplete mesocycle",
			userID:         "user-2",
			mesocycle:      2,
			daysPerWeek:    3,
			week4Completed: 2,
			weekSchedule:   "fb_3",
			wantComplete:   false,
			wantRate:       66.66666666666666,
		},
		{
			name:           "no workouts completed",
			userID:         "user-3",
			mesocycle:      1,
			daysPerWeek:    4,
			week4Completed: 0,
			weekSchedule:   "ul_4",
			wantComplete:   false,
			wantRate:       0.0,
		},
		{
			name:           "exceeded workouts",
			userID:         "user-4",
			mesocycle:      3,
			daysPerWeek:    3,
			week4Completed: 4,
			weekSchedule:   "fb_3",
			wantComplete:   true,
			wantRate:       133.33333333333331,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := NewMesocycleStatus(
				tt.userID,
				tt.mesocycle,
				tt.daysPerWeek,
				tt.week4Completed,
				tt.weekSchedule,
			)

			if status.UserID != tt.userID {
				t.Errorf("UserID = %v, want %v", status.UserID, tt.userID)
			}
			if status.MesocycleNumber != tt.mesocycle {
				t.Errorf("MesocycleNumber = %v, want %v", status.MesocycleNumber, tt.mesocycle)
			}
			if status.DaysPerWeek != tt.daysPerWeek {
				t.Errorf("DaysPerWeek = %v, want %v", status.DaysPerWeek, tt.daysPerWeek)
			}
			if status.Week4Completed != tt.week4Completed {
				t.Errorf("Week4Completed = %v, want %v", status.Week4Completed, tt.week4Completed)
			}
			if status.WeekSchedule != tt.weekSchedule {
				t.Errorf("WeekSchedule = %v, want %v", status.WeekSchedule, tt.weekSchedule)
			}
			if status.IsComplete != tt.wantComplete {
				t.Errorf("IsComplete = %v, want %v", status.IsComplete, tt.wantComplete)
			}
			if status.CompletionRate != tt.wantRate {
				t.Errorf("CompletionRate = %v, want %v", status.CompletionRate, tt.wantRate)
			}
			if status.Week4Total != tt.daysPerWeek {
				t.Errorf("Week4Total = %v, want %v", status.Week4Total, tt.daysPerWeek)
			}
		})
	}
}

func TestNewMesocycleStatus_ZeroDaysPerWeek(t *testing.T) {
	status := NewMesocycleStatus("user-1", 1, 0, 0, "fb_0")

	if status.CompletionRate != 0.0 {
		t.Errorf("expected CompletionRate to be 0.0 for zero days, got %v", status.CompletionRate)
	}
}

func TestNewRenewalResult(t *testing.T) {
	result := NewRenewalResult(2, RenewalTypeMaintain, "Plan renovado exitosamente")

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.NewMesocycleNumber != 2 {
		t.Errorf("expected NewMesocycleNumber to be 2, got %d", result.NewMesocycleNumber)
	}
	if result.RenewalType != RenewalTypeMaintain {
		t.Errorf("expected RenewalType to be MAINTAIN, got %v", result.RenewalType)
	}
	if result.Message != "Plan renovado exitosamente" {
		t.Errorf("unexpected message: %s", result.Message)
	}
	if !result.ScheduleCleared {
		t.Error("expected ScheduleCleared to be true")
	}
}

func TestRenewalResult_WithExercisesRotated(t *testing.T) {
	result := NewRenewalResult(2, RenewalTypeRotateExercises, "Ejercicios rotados").
		WithExercisesRotated(5)

	if result.ExercisesRotated != 5 {
		t.Errorf("expected ExercisesRotated to be 5, got %d", result.ExercisesRotated)
	}
}

func TestRenewalResult_WithNewWeekSchedule(t *testing.T) {
	result := NewRenewalResult(2, RenewalTypeChangeDays, "Dias cambiados").
		WithNewWeekSchedule("ul_4")

	if result.NewWeekSchedule != "ul_4" {
		t.Errorf("expected NewWeekSchedule to be 'ul_4', got '%s'", result.NewWeekSchedule)
	}
}

func TestRenewalType_Constants(t *testing.T) {
	if RenewalTypeMaintain != "MAINTAIN" {
		t.Errorf("expected MAINTAIN, got %v", RenewalTypeMaintain)
	}
	if RenewalTypeRotateExercises != "ROTATE_EXERCISES" {
		t.Errorf("expected ROTATE_EXERCISES, got %v", RenewalTypeRotateExercises)
	}
	if RenewalTypeChangeDays != "CHANGE_DAYS" {
		t.Errorf("expected CHANGE_DAYS, got %v", RenewalTypeChangeDays)
	}
	if RenewalTypeModifyProfile != "MODIFY_PROFILE" {
		t.Errorf("expected MODIFY_PROFILE, got %v", RenewalTypeModifyProfile)
	}
}
