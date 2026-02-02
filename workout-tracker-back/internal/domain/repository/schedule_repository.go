package repository

import (
	"context"
)

// ScheduleReader defines read operations for user_weekly_schedule
type ScheduleReader interface {
	// GetCompletedCountForWeek returns the count of completed sessions for a week
	GetCompletedCountForWeek(ctx context.Context, userID string, week int) (int, error)
}

// ScheduleWriter defines write operations for user_weekly_schedule
type ScheduleWriter interface {
	// ClearSchedule deletes all schedule entries for a user
	ClearSchedule(ctx context.Context, userID string) error
	// ClearScheduleForWeeks deletes schedule entries for specific weeks
	ClearScheduleForWeeks(ctx context.Context, userID string, weeks []int) error
}

// ScheduleRepository combines all schedule repository operations
// Following Interface Segregation Principle (ISP)
type ScheduleRepository interface {
	ScheduleReader
	ScheduleWriter
}
