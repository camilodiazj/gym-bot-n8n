package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// PlanRepository implements repository.PlanRepository using PostgreSQL
type PlanRepository struct {
	conn *Connection
}

// Ensure PlanRepository implements the interface
var _ repository.PlanRepository = (*PlanRepository)(nil)

// NewPlanRepository creates a new PlanRepository
func NewPlanRepository(conn *Connection) *PlanRepository {
	return &PlanRepository{conn: conn}
}

// GetMesocycleStatus retrieves the current mesocycle status for a user
func (r *PlanRepository) GetMesocycleStatus(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
	query := `
		WITH week4_sessions AS (
			SELECT
				COUNT(*) FILTER (WHERE "Completed" = true) as completed,
				COUNT(*) as total
			FROM user_weekly_schedule
			WHERE user_id = $1 AND week = 4
		),
		plan_info AS (
			SELECT
				up.plan_id,
				up.user_id,
				up.mesocycle_number,
				up.last_renewal_date,
				up.goal,
				up.level,
				up.week_schedule,
				ws.days_per_week
			FROM users_plans up
			JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
			WHERE up.user_id = $1 AND up.status = 'active'
			LIMIT 1
		)
		SELECT
			pi.user_id,
			pi.mesocycle_number,
			pi.days_per_week,
			pi.week_schedule,
			COALESCE(w4.completed, 0) as week4_completed,
			pi.days_per_week as week4_total,
			COALESCE(w4.completed, 0) >= pi.days_per_week as is_complete,
			pi.last_renewal_date,
			pi.goal,
			pi.level
		FROM plan_info pi
		LEFT JOIN week4_sessions w4 ON true
	`

	var status entity.MesocycleStatus
	var lastRenewalDate sql.NullTime
	var goal, level sql.NullString

	err := r.conn.DB.QueryRowContext(ctx, query, userID).Scan(
		&status.UserID,
		&status.MesocycleNumber,
		&status.DaysPerWeek,
		&status.WeekSchedule,
		&status.Week4Completed,
		&status.Week4Total,
		&status.IsComplete,
		&lastRenewalDate,
		&goal,
		&level,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query mesocycle status", err)
	}

	if lastRenewalDate.Valid {
		status.LastRenewalDate = &lastRenewalDate.Time
	}
	status.Goal = goal.String
	status.Level = level.String

	// Calculate completion rate
	if status.Week4Total > 0 {
		status.CompletionRate = float64(status.Week4Completed) / float64(status.Week4Total) * 100
	}

	return &status, nil
}

// GetByUserID retrieves the active plan for a user
func (r *PlanRepository) GetByUserID(ctx context.Context, userID string) (*entity.Plan, error) {
	query := `
		SELECT
			plan_id,
			user_id,
			template_id,
			week_schedule,
			goal,
			level,
			status,
			mesocycle_number,
			last_renewal_date,
			created_at
		FROM users_plans
		WHERE user_id = $1 AND status = 'active'
		LIMIT 1
	`

	var plan entity.Plan
	var templateID, goal, level sql.NullString
	var lastRenewalDate sql.NullTime
	var createdAt sql.NullTime

	err := r.conn.DB.QueryRowContext(ctx, query, userID).Scan(
		&plan.PlanID,
		&plan.UserID,
		&templateID,
		&plan.WeekSchedule,
		&goal,
		&level,
		&plan.Status,
		&plan.MesocycleNumber,
		&lastRenewalDate,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query plan", err)
	}

	plan.TemplateID = templateID.String
	plan.Goal = goal.String
	plan.Level = level.String
	if lastRenewalDate.Valid {
		plan.LastRenewalDate = &lastRenewalDate.Time
	}
	if createdAt.Valid {
		plan.CreatedAt = createdAt.Time
	}

	return &plan, nil
}

// IncrementMesocycle increments the mesocycle number and updates renewal date
func (r *PlanRepository) IncrementMesocycle(ctx context.Context, userID string) error {
	query := `
		UPDATE users_plans
		SET
			mesocycle_number = mesocycle_number + 1,
			last_renewal_date = $2
		WHERE user_id = $1 AND status = 'active'
	`

	result, err := r.conn.DB.ExecContext(ctx, query, userID, time.Now())
	if err != nil {
		return apperror.NewInternalError("failed to increment mesocycle", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return apperror.NewNotFoundError("no active plan found for user")
	}

	return nil
}

// UpdateWeekSchedule updates the week_schedule for a user's plan
func (r *PlanRepository) UpdateWeekSchedule(ctx context.Context, userID, weekSchedule string) error {
	query := `
		UPDATE users_plans
		SET week_schedule = $2
		WHERE user_id = $1 AND status = 'active'
	`

	result, err := r.conn.DB.ExecContext(ctx, query, userID, weekSchedule)
	if err != nil {
		return apperror.NewInternalError("failed to update week schedule", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return apperror.NewNotFoundError("no active plan found for user")
	}

	return nil
}
