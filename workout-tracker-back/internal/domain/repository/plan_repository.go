package repository

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// PlanReader defines read operations for plans
type PlanReader interface {
	// GetMesocycleStatus retrieves the current mesocycle status for a user
	GetMesocycleStatus(ctx context.Context, userID string) (*entity.MesocycleStatus, error)
	// GetByUserID retrieves the active plan for a user
	GetByUserID(ctx context.Context, userID string) (*entity.Plan, error)
}

// PlanWriter defines write operations for plans
type PlanWriter interface {
	// IncrementMesocycle increments the mesocycle number and updates renewal date
	IncrementMesocycle(ctx context.Context, userID string) error
	// UpdateWeekSchedule updates the week_schedule for a user's plan
	UpdateWeekSchedule(ctx context.Context, userID, weekSchedule string) error
}

// PlanRepository combines all plan repository operations
// Following Interface Segregation Principle (ISP)
type PlanRepository interface {
	PlanReader
	PlanWriter
}
