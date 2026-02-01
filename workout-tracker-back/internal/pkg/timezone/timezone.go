package timezone

import (
	"fmt"
	"time"
)

// UserTimezone is the default timezone for GymBot users (Colombia)
const UserTimezone = "America/Bogota"

var bogotaLocation *time.Location

func init() {
	var err error
	bogotaLocation, err = time.LoadLocation(UserTimezone)
	if err != nil {
		panic(fmt.Sprintf("failed to load timezone %s: %v", UserTimezone, err))
	}
}

// TodayForUser returns today's date at midnight in the user's timezone
func TodayForUser() time.Time {
	now := time.Now().In(bogotaLocation)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, bogotaLocation)
}

// TodayMidnightUTC returns today's midnight in Bogota, expressed as UTC
// This is used for database queries where planned_day is stored as midnight Bogota in UTC
func TodayMidnightUTC() time.Time {
	return TodayForUser().UTC()
}

// ToUserTimezone converts a UTC time to the user's local timezone for display
func ToUserTimezone(t time.Time) time.Time {
	return t.In(bogotaLocation)
}

// ToUTC converts a time to UTC
func ToUTC(t time.Time) time.Time {
	return t.UTC()
}

// DateStringToUTC converts a date string (YYYY-MM-DD) to a UTC timestamp
// representing midnight of that date in Bogota timezone
func DateStringToUTC(dateStr string) (time.Time, error) {
	// Parse the date string as a date in Bogota timezone
	t, err := time.ParseInLocation("2006-01-02", dateStr, bogotaLocation)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: %w", err)
	}
	// Return as UTC (midnight Bogota = 05:00 UTC)
	return t.UTC(), nil
}

// ParsePlannedDay parses a planned_day value which could be a string or time.Time
// and returns the corresponding time.Time in UTC
func ParsePlannedDay(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v.UTC(), nil
	case string:
		// Try parsing as full timestamp first
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UTC(), nil
		}
		// Try parsing as date string
		return DateStringToUTC(v)
	default:
		return time.Time{}, fmt.Errorf("unsupported planned_day type: %T", value)
	}
}

// GetLocation returns the Bogota timezone location
func GetLocation() *time.Location {
	return bogotaLocation
}
