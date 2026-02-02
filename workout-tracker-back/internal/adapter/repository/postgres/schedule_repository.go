package postgres

import (
	"context"
	"database/sql"

	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
	"github.com/lib/pq"
)

// ScheduleRepository implements repository.ScheduleRepository using PostgreSQL
type ScheduleRepository struct {
	conn *Connection
}

// Ensure ScheduleRepository implements the interface
var _ repository.ScheduleRepository = (*ScheduleRepository)(nil)

// NewScheduleRepository creates a new ScheduleRepository
func NewScheduleRepository(conn *Connection) *ScheduleRepository {
	return &ScheduleRepository{conn: conn}
}

// ClearSchedule deletes all schedule entries for a user
func (r *ScheduleRepository) ClearSchedule(ctx context.Context, userID string) error {
	query := `
		DELETE FROM user_weekly_schedule
		WHERE user_id = $1
	`

	_, err := r.conn.DB.ExecContext(ctx, query, userID)
	if err != nil {
		return apperror.NewInternalError("failed to clear schedule", err)
	}

	return nil
}

// ClearScheduleForWeeks deletes schedule entries for specific weeks
func (r *ScheduleRepository) ClearScheduleForWeeks(ctx context.Context, userID string, weeks []int) error {
	if len(weeks) == 0 {
		return nil
	}

	// Build query with parameterized week list using pq.Array
	query := `
		DELETE FROM user_weekly_schedule
		WHERE user_id = $1 AND week = ANY($2::int[])
	`

	_, err := r.conn.DB.ExecContext(ctx, query, userID, pq.Array(weeks))
	if err != nil {
		return apperror.NewInternalError("failed to clear schedule for weeks", err)
	}

	return nil
}

// GetCompletedCountForWeek returns the count of completed sessions for a week
func (r *ScheduleRepository) GetCompletedCountForWeek(ctx context.Context, userID string, week int) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM user_weekly_schedule
		WHERE user_id = $1 AND week = $2 AND "Completed" = true
	`

	var count int
	err := r.conn.DB.QueryRowContext(ctx, query, userID, week).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, apperror.NewInternalError("failed to get completed count", err)
	}

	return count, nil
}
