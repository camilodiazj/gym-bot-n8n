package postgres

import (
	"testing"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

func TestNewPlanRepository(t *testing.T) {
	repo := NewPlanRepository(nil)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestMesocycleStatus_IsComplete(t *testing.T) {
	tests := []struct {
		name           string
		week4Completed int
		daysPerWeek    int
		wantComplete   bool
	}{
		{
			name:           "complete when all sessions done",
			week4Completed: 4,
			daysPerWeek:    4,
			wantComplete:   true,
		},
		{
			name:           "complete when more than required",
			week4Completed: 5,
			daysPerWeek:    4,
			wantComplete:   true,
		},
		{
			name:           "incomplete when missing sessions",
			week4Completed: 2,
			daysPerWeek:    4,
			wantComplete:   false,
		},
		{
			name:           "incomplete when no sessions done",
			week4Completed: 0,
			daysPerWeek:    4,
			wantComplete:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := entity.NewMesocycleStatus(
				"user-123",
				1,
				tt.daysPerWeek,
				tt.week4Completed,
				"ul_4",
			)
			if status.IsComplete != tt.wantComplete {
				t.Errorf("IsComplete = %v, want %v", status.IsComplete, tt.wantComplete)
			}
		})
	}
}

func TestMesocycleStatus_CompletionRate(t *testing.T) {
	tests := []struct {
		name             string
		week4Completed   int
		daysPerWeek      int
		wantRate         float64
	}{
		{
			name:           "50% completion",
			week4Completed: 2,
			daysPerWeek:    4,
			wantRate:       50.0,
		},
		{
			name:           "100% completion",
			week4Completed: 4,
			daysPerWeek:    4,
			wantRate:       100.0,
		},
		{
			name:           "0% completion",
			week4Completed: 0,
			daysPerWeek:    4,
			wantRate:       0.0,
		},
		{
			name:           "75% completion",
			week4Completed: 3,
			daysPerWeek:    4,
			wantRate:       75.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := entity.NewMesocycleStatus(
				"user-123",
				1,
				tt.daysPerWeek,
				tt.week4Completed,
				"ul_4",
			)
			if status.CompletionRate != tt.wantRate {
				t.Errorf("CompletionRate = %v, want %v", status.CompletionRate, tt.wantRate)
			}
		})
	}
}

func TestPlan_IncrementMesocycle(t *testing.T) {
	plan := entity.NewPlan("plan-1", "user-1", "template-1", "ul_4", "hipertrofia", "intermedio", 1)

	if plan.MesocycleNumber != 1 {
		t.Errorf("Initial MesocycleNumber = %d, want 1", plan.MesocycleNumber)
	}
	if plan.LastRenewalDate != nil {
		t.Error("Initial LastRenewalDate should be nil")
	}

	plan.IncrementMesocycle()

	if plan.MesocycleNumber != 2 {
		t.Errorf("After increment MesocycleNumber = %d, want 2", plan.MesocycleNumber)
	}
	if plan.LastRenewalDate == nil {
		t.Error("LastRenewalDate should be set after increment")
	}
	if time.Since(*plan.LastRenewalDate) > time.Second {
		t.Error("LastRenewalDate should be set to now")
	}
}

func TestPlan_UpdateWeekSchedule(t *testing.T) {
	plan := entity.NewPlan("plan-1", "user-1", "template-1", "ul_4", "hipertrofia", "intermedio", 1)

	if plan.WeekSchedule != "ul_4" {
		t.Errorf("Initial WeekSchedule = %s, want ul_4", plan.WeekSchedule)
	}

	plan.UpdateWeekSchedule("ppl_5")

	if plan.WeekSchedule != "ppl_5" {
		t.Errorf("After update WeekSchedule = %s, want ppl_5", plan.WeekSchedule)
	}
}

func TestPlan_IsActive(t *testing.T) {
	plan := entity.NewPlan("plan-1", "user-1", "template-1", "ul_4", "hipertrofia", "intermedio", 1)

	if !plan.IsActive() {
		t.Error("New plan should be active")
	}

	plan.Status = "inactive"
	if plan.IsActive() {
		t.Error("Inactive plan should not be active")
	}
}

func TestPlan_CanRenew(t *testing.T) {
	plan := entity.NewPlan("plan-1", "user-1", "template-1", "ul_4", "hipertrofia", "intermedio", 1)

	if !plan.CanRenew() {
		t.Error("Active plan should be able to renew")
	}

	plan.Status = "inactive"
	if plan.CanRenew() {
		t.Error("Inactive plan should not be able to renew")
	}
}
