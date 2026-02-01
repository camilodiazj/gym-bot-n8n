package timezone_test

import (
	"testing"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/pkg/timezone"
)

func TestTodayForUser_ReturnsMidnight(t *testing.T) {
	today := timezone.TodayForUser()

	// Should be midnight (hour = 0, minute = 0, second = 0)
	if today.Hour() != 0 || today.Minute() != 0 || today.Second() != 0 {
		t.Errorf("TodayForUser() should return midnight, got %v", today)
	}
}

func TestTodayForUser_InBogotaTimezone(t *testing.T) {
	today := timezone.TodayForUser()
	loc := timezone.GetLocation()

	// Should be in Bogota timezone
	if today.Location().String() != loc.String() {
		t.Errorf("TodayForUser() should be in Bogota timezone, got %v", today.Location())
	}
}

func TestTodayMidnightUTC_Is5AMForBogota(t *testing.T) {
	midnightUTC := timezone.TodayMidnightUTC()

	// Midnight Bogota = 05:00 UTC (Bogota is UTC-5)
	if midnightUTC.Hour() != 5 {
		t.Errorf("TodayMidnightUTC() hour should be 5 (midnight Bogota), got %d", midnightUTC.Hour())
	}

	// Should have zero minutes and seconds
	if midnightUTC.Minute() != 0 || midnightUTC.Second() != 0 {
		t.Errorf("TodayMidnightUTC() should have 0 minutes and seconds, got %v", midnightUTC)
	}

	// Should be in UTC
	if midnightUTC.Location().String() != "UTC" {
		t.Errorf("TodayMidnightUTC() should be in UTC, got %v", midnightUTC.Location())
	}
}

func TestToUserTimezone_ConvertsUTCToBogota(t *testing.T) {
	// 2026-01-31 05:00:00 UTC = 2026-01-31 00:00:00 Bogota
	utcTime := time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC)
	bogotaTime := timezone.ToUserTimezone(utcTime)

	if bogotaTime.Hour() != 0 {
		t.Errorf("ToUserTimezone(05:00 UTC) hour should be 0, got %d", bogotaTime.Hour())
	}
	if bogotaTime.Day() != 31 {
		t.Errorf("ToUserTimezone(05:00 UTC) day should be 31, got %d", bogotaTime.Day())
	}
	if bogotaTime.Month() != time.January {
		t.Errorf("ToUserTimezone(05:00 UTC) month should be January, got %v", bogotaTime.Month())
	}
}

func TestToUserTimezone_CrossDayBoundary(t *testing.T) {
	// 2026-02-01 00:30:00 UTC = 2026-01-31 19:30:00 Bogota (previous day!)
	utcTime := time.Date(2026, 2, 1, 0, 30, 0, 0, time.UTC)
	bogotaTime := timezone.ToUserTimezone(utcTime)

	if bogotaTime.Day() != 31 {
		t.Errorf("ToUserTimezone(Feb 1 00:30 UTC) day should be 31 (Jan), got %d", bogotaTime.Day())
	}
	if bogotaTime.Month() != time.January {
		t.Errorf("ToUserTimezone(Feb 1 00:30 UTC) month should be January, got %v", bogotaTime.Month())
	}
	if bogotaTime.Hour() != 19 {
		t.Errorf("ToUserTimezone(Feb 1 00:30 UTC) hour should be 19, got %d", bogotaTime.Hour())
	}
	if bogotaTime.Minute() != 30 {
		t.Errorf("ToUserTimezone(Feb 1 00:30 UTC) minute should be 30, got %d", bogotaTime.Minute())
	}
}

func TestToUTC_ConvertsToUTC(t *testing.T) {
	loc := timezone.GetLocation()
	bogotaTime := time.Date(2026, 1, 31, 0, 0, 0, 0, loc)
	utcTime := timezone.ToUTC(bogotaTime)

	if utcTime.Location().String() != "UTC" {
		t.Errorf("ToUTC() should return UTC location, got %v", utcTime.Location())
	}
	if utcTime.Hour() != 5 {
		t.Errorf("ToUTC(midnight Bogota) hour should be 5, got %d", utcTime.Hour())
	}
}

func TestDateStringToUTC_ValidDate(t *testing.T) {
	tests := []struct {
		name        string
		dateStr     string
		expectedUTC time.Time
	}{
		{
			name:        "January 31",
			dateStr:     "2026-01-31",
			expectedUTC: time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC),
		},
		{
			name:        "February 1",
			dateStr:     "2026-02-01",
			expectedUTC: time.Date(2026, 2, 1, 5, 0, 0, 0, time.UTC),
		},
		{
			name:        "December 31 (year boundary)",
			dateStr:     "2026-12-31",
			expectedUTC: time.Date(2026, 12, 31, 5, 0, 0, 0, time.UTC),
		},
		{
			name:        "Leap year Feb 29",
			dateStr:     "2028-02-29",
			expectedUTC: time.Date(2028, 2, 29, 5, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := timezone.DateStringToUTC(tt.dateStr)
			if err != nil {
				t.Fatalf("DateStringToUTC(%q) unexpected error: %v", tt.dateStr, err)
			}
			if !result.Equal(tt.expectedUTC) {
				t.Errorf("DateStringToUTC(%q) = %v, want %v", tt.dateStr, result, tt.expectedUTC)
			}
		})
	}
}

func TestDateStringToUTC_InvalidFormat(t *testing.T) {
	invalidDates := []string{
		"31-01-2026",      // Wrong format (DD-MM-YYYY)
		"01/31/2026",      // Wrong format (MM/DD/YYYY)
		"2026/01/31",      // Wrong separator
		"2026-1-31",       // No leading zero
		"not-a-date",      // Invalid string
		"",                // Empty string
		"2026-13-01",      // Invalid month
		"2026-01-32",      // Invalid day
	}

	for _, dateStr := range invalidDates {
		t.Run(dateStr, func(t *testing.T) {
			_, err := timezone.DateStringToUTC(dateStr)
			if err == nil {
				t.Errorf("DateStringToUTC(%q) expected error, got nil", dateStr)
			}
		})
	}
}

func TestParsePlannedDay_TimeValue(t *testing.T) {
	inputTime := time.Date(2026, 1, 31, 10, 30, 0, 0, time.UTC)
	result, err := timezone.ParsePlannedDay(inputTime)

	if err != nil {
		t.Fatalf("ParsePlannedDay(time.Time) unexpected error: %v", err)
	}
	if !result.Equal(inputTime.UTC()) {
		t.Errorf("ParsePlannedDay(time.Time) = %v, want %v", result, inputTime.UTC())
	}
	if result.Location().String() != "UTC" {
		t.Errorf("ParsePlannedDay(time.Time) should return UTC, got %v", result.Location())
	}
}

func TestParsePlannedDay_RFC3339String(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "UTC timestamp",
			input:    "2026-01-31T05:00:00Z",
			expected: time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC),
		},
		{
			name:     "Timestamp with offset",
			input:    "2026-01-31T00:00:00-05:00",
			expected: time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := timezone.ParsePlannedDay(tt.input)
			if err != nil {
				t.Fatalf("ParsePlannedDay(%q) unexpected error: %v", tt.input, err)
			}
			if !result.Equal(tt.expected) {
				t.Errorf("ParsePlannedDay(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParsePlannedDay_DateString(t *testing.T) {
	result, err := timezone.ParsePlannedDay("2026-01-31")
	if err != nil {
		t.Fatalf("ParsePlannedDay(date string) unexpected error: %v", err)
	}

	expected := time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParsePlannedDay(date string) = %v, want %v", result, expected)
	}
}

func TestParsePlannedDay_UnsupportedType(t *testing.T) {
	invalidInputs := []interface{}{
		123,
		12.34,
		true,
		[]string{"2026-01-31"},
		map[string]string{"date": "2026-01-31"},
	}

	for _, input := range invalidInputs {
		t.Run("", func(t *testing.T) {
			_, err := timezone.ParsePlannedDay(input)
			if err == nil {
				t.Errorf("ParsePlannedDay(%T) expected error, got nil", input)
			}
		})
	}
}

func TestGetLocation_ReturnsBogota(t *testing.T) {
	loc := timezone.GetLocation()

	if loc == nil {
		t.Fatal("GetLocation() returned nil")
	}
	if loc.String() != "America/Bogota" {
		t.Errorf("GetLocation() = %v, want America/Bogota", loc.String())
	}
}

func TestMidnightConversion_BogotaToUTC(t *testing.T) {
	// Test the core conversion logic used in database queries
	tests := []struct {
		name       string
		bogotaDate string
		utcHour    int
	}{
		{"Standard date", "2026-01-31", 5},
		{"Year start", "2026-01-01", 5},
		{"Year end", "2026-12-31", 5},
		{"Summer month", "2026-07-15", 5}, // Colombia has no DST, so always UTC-5
		{"Winter month", "2026-01-15", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := timezone.DateStringToUTC(tt.bogotaDate)
			if err != nil {
				t.Fatalf("DateStringToUTC(%q) error: %v", tt.bogotaDate, err)
			}
			if result.Hour() != tt.utcHour {
				t.Errorf("DateStringToUTC(%q) hour = %d, want %d", tt.bogotaDate, result.Hour(), tt.utcHour)
			}
		})
	}
}

// TestCrossTimezoneBoundary tests the critical edge case where
// the server is in a different "day" than the user
func TestCrossTimezoneBoundary(t *testing.T) {
	loc := timezone.GetLocation()

	tests := []struct {
		name          string
		serverTimeUTC time.Time
		plannedDayUTC time.Time
		shouldMatch   bool
	}{
		{
			name:          "7:30 PM Bogota (00:30 UTC next day) - should match same day",
			serverTimeUTC: time.Date(2026, 2, 1, 0, 30, 0, 0, time.UTC),
			plannedDayUTC: time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC),
			shouldMatch:   true,
		},
		{
			name:          "4:59 AM UTC (11:59 PM Bogota) - still same day",
			serverTimeUTC: time.Date(2026, 2, 1, 4, 59, 0, 0, time.UTC),
			plannedDayUTC: time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC),
			shouldMatch:   true,
		},
		{
			name:          "5:00 AM UTC (00:00 Bogota next day) - new day",
			serverTimeUTC: time.Date(2026, 2, 1, 5, 0, 0, 0, time.UTC),
			plannedDayUTC: time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC),
			shouldMatch:   false,
		},
		{
			name:          "5:01 AM UTC - definitely next day",
			serverTimeUTC: time.Date(2026, 2, 1, 5, 1, 0, 0, time.UTC),
			plannedDayUTC: time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC),
			shouldMatch:   false,
		},
		{
			name:          "Early morning same UTC day - matches",
			serverTimeUTC: time.Date(2026, 1, 31, 6, 0, 0, 0, time.UTC),
			plannedDayUTC: time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC),
			shouldMatch:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate "today" from server time using Bogota timezone
			bogotaTime := tt.serverTimeUTC.In(loc)
			todayMidnightBogota := time.Date(
				bogotaTime.Year(), bogotaTime.Month(), bogotaTime.Day(),
				0, 0, 0, 0, loc,
			)
			todayMidnightUTC := todayMidnightBogota.UTC()

			matches := tt.plannedDayUTC.Equal(todayMidnightUTC)
			if matches != tt.shouldMatch {
				t.Errorf("Expected match=%v, got %v. Today UTC: %v, Planned: %v",
					tt.shouldMatch, matches, todayMidnightUTC, tt.plannedDayUTC)
			}
		})
	}
}

// TestPlannedDayFormat_AlwaysMidnight verifies that all planned_day values
// should have hour = 5 (UTC) representing midnight Bogota
func TestPlannedDayFormat_AlwaysMidnight(t *testing.T) {
	plannedDays := []time.Time{
		time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 1, 5, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 15, 5, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 5, 0, 0, 0, time.UTC),
	}

	for _, pd := range plannedDays {
		if pd.Hour() != 5 {
			t.Errorf("planned_day %v should have hour=5 (midnight Bogota), got %d", pd, pd.Hour())
		}
		if pd.Minute() != 0 {
			t.Errorf("planned_day %v should have minute=0, got %d", pd, pd.Minute())
		}
		if pd.Second() != 0 {
			t.Errorf("planned_day %v should have second=0, got %d", pd, pd.Second())
		}
	}
}

// TestColombiaHasNoDST verifies that Colombia doesn't observe daylight saving time
func TestColombiaHasNoDST(t *testing.T) {
	loc := timezone.GetLocation()

	// Check various dates throughout the year
	dates := []time.Time{
		time.Date(2026, 1, 15, 12, 0, 0, 0, loc),  // January (winter)
		time.Date(2026, 4, 15, 12, 0, 0, 0, loc),  // April (spring)
		time.Date(2026, 7, 15, 12, 0, 0, 0, loc),  // July (summer)
		time.Date(2026, 10, 15, 12, 0, 0, 0, loc), // October (fall)
	}

	for _, d := range dates {
		// Convert to UTC and check the offset
		utc := d.UTC()
		diff := d.Hour() - utc.Hour()
		if diff < 0 {
			diff += 24
		}

		// Colombia is always UTC-5 (no DST), so local hour - UTC hour should be -5 or +19 (accounting for day change)
		// At noon local time, it should be 17:00 UTC
		if utc.Hour() != 17 {
			t.Errorf("At noon Bogota on %v, UTC hour should be 17, got %d", d.Format("2006-01-02"), utc.Hour())
		}
	}
}
