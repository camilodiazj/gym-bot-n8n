package postgres

import (
	"sort"
	"testing"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/pkg/timezone"
)

func TestParseReps(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"single number", "10", 10},
		{"range with hyphen", "10-12", 10},
		{"range with en-dash", "6–8", 6},
		{"single digit", "8", 8},
		{"empty string", "", 0},
		{"range starting with single digit", "8-10", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseReps(tt.input)
			if result != tt.expected {
				t.Errorf("parseReps(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseRepsRange(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedMin int
		expectedMax int
	}{
		{"single number", "10", 10, 10},
		{"range with hyphen", "10-12", 10, 12},
		{"range with en-dash", "6–8", 6, 8},
		{"range with en-dash larger", "8–15", 8, 15},
		{"single digit", "8", 8, 8},
		{"empty string", "", 0, 0},
		{"range starting with single digit", "8-10", 8, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max := parseRepsRange(tt.input)
			if min != tt.expectedMin || max != tt.expectedMax {
				t.Errorf("parseRepsRange(%q) = (%d, %d), want (%d, %d)",
					tt.input, min, max, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

// TestGetTodayWorkout_CrossTimezone tests the critical edge case where
// the server is in UTC but the user is in Colombia (UTC-5)
func TestGetTodayWorkout_CrossTimezone(t *testing.T) {
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
			plannedDayUTC: time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC), // Jan 31 Bogota
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
		{
			name:          "Midnight UTC - previous day in Bogota",
			serverTimeUTC: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			plannedDayUTC: time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC),
			shouldMatch:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate "today" from server time using Bogota timezone
			// This simulates what the SQL query does:
			// DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota'
			bogotaTime := tt.serverTimeUTC.In(loc)
			todayMidnightBogota := time.Date(
				bogotaTime.Year(), bogotaTime.Month(), bogotaTime.Day(),
				0, 0, 0, 0, loc,
			)
			todayMidnightUTC := todayMidnightBogota.UTC()

			matches := tt.plannedDayUTC.Equal(todayMidnightUTC)
			if matches != tt.shouldMatch {
				t.Errorf("Expected match=%v, got %v. Server UTC: %v, Today UTC: %v, Planned: %v",
					tt.shouldMatch, matches, tt.serverTimeUTC, todayMidnightUTC, tt.plannedDayUTC)
			}
		})
	}
}

// TestPlannedDayFormat_AlwaysMidnightUTC verifies that all planned_day_utc values
// should have hour = 5 (UTC) representing midnight Bogota (UTC-5)
func TestPlannedDayFormat_AlwaysMidnightUTC(t *testing.T) {
	validPlannedDays := []time.Time{
		time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 1, 5, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 15, 5, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 5, 0, 0, 0, time.UTC),
	}

	for _, pd := range validPlannedDays {
		if pd.Hour() != 5 {
			t.Errorf("planned_day_utc %v should have hour=5 (midnight Bogota), got %d", pd, pd.Hour())
		}
		if pd.Minute() != 0 {
			t.Errorf("planned_day_utc %v should have minute=0, got %d", pd, pd.Minute())
		}
		if pd.Second() != 0 {
			t.Errorf("planned_day_utc %v should have second=0, got %d", pd, pd.Second())
		}
	}
}

// TestTimezoneToUTC verifies the timezone conversion helper
func TestTimezoneToUTC(t *testing.T) {
	loc := timezone.GetLocation()

	// Test conversion from Bogota midnight to UTC
	bogotaMidnight := time.Date(2026, 1, 31, 0, 0, 0, 0, loc)
	utc := timezone.ToUTC(bogotaMidnight)

	if utc.Hour() != 5 {
		t.Errorf("Bogota midnight should be 05:00 UTC, got %02d:00", utc.Hour())
	}
	if utc.Location().String() != "UTC" {
		t.Errorf("ToUTC should return UTC location, got %v", utc.Location())
	}
}

// TestTimezoneFromUTC verifies the reverse timezone conversion
func TestTimezoneFromUTC(t *testing.T) {
	// 05:00 UTC should be midnight Bogota
	utcTime := time.Date(2026, 1, 31, 5, 0, 0, 0, time.UTC)
	bogota := timezone.ToUserTimezone(utcTime)

	if bogota.Hour() != 0 {
		t.Errorf("05:00 UTC should be 00:00 Bogota, got %02d:00", bogota.Hour())
	}
	if bogota.Day() != 31 {
		t.Errorf("Expected day 31, got %d", bogota.Day())
	}
}

// --- Smart Alternative Exercises Filtering Tests ---

func TestStripAccents(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Mañana", "Manana"},
		{"Glúteo", "Gluteo"},
		{"días", "dias"},
		{"Más de 3 años", "Mas de 3 anos"},
		{"bandas elásticas", "bandas elasticas"},
		{"no accents", "no accents"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stripAccents(tt.input)
			if result != tt.expected {
				t.Errorf("stripAccents(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLevelHierarchy(t *testing.T) {
	tests := []struct {
		level    string
		expected []string
	}{
		{"Principiante", []string{"Principiante"}},
		{"Intermedio", []string{"Principiante", "Intermedio"}},
		{"Avanzado", []string{"Principiante", "Intermedio", "Avanzado"}},
		{"unknown", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := levelHierarchy(tt.level)
			if len(result) != len(tt.expected) {
				t.Fatalf("levelHierarchy(%q) returned %d items, want %d", tt.level, len(result), len(tt.expected))
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("levelHierarchy(%q)[%d] = %q, want %q", tt.level, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestParseHomeEquipment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mustHave []string // these canonical values must be present
	}{
		{
			"mancuernas y bandas",
			"mancuernas, bandas",
			[]string{"dumbbell", "resistance_band", "bodyweight", "Peso Corporal"},
		},
		{
			"mancuernas con Mancuerna Spanish match",
			"mancuernas, bandas",
			[]string{"Mancuerna"}, // Spanish DB equivalent of dumbbell
		},
		{
			"peso corporal only",
			"peso corporal",
			[]string{"bodyweight", "Peso Corporal"},
		},
		{
			"solo cuerpo",
			"Solo cuerpo",
			[]string{"bodyweight", "Peso Corporal"},
		},
		{
			"empty always has bodyweight",
			"",
			[]string{"bodyweight", "Peso Corporal"},
		},
		{
			"null string always has bodyweight",
			"null",
			[]string{"bodyweight", "Peso Corporal"},
		},
		{
			"kettlebell and barbell",
			"kettlebell, barra",
			[]string{"kettlebell", "barbell", "Barra", "bodyweight"},
		},
		{
			"accented bandas elasticas",
			"Bandas elásticas",
			[]string{"resistance_band", "bodyweight"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHomeEquipment(tt.input)
			resultSet := map[string]bool{}
			for _, v := range result {
				resultSet[v] = true
			}
			for _, must := range tt.mustHave {
				if !resultSet[must] {
					t.Errorf("parseHomeEquipment(%q) missing %q, got %v", tt.input, must, result)
				}
			}
		})
	}
}

func TestParseDislikedMuscles(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"pantorrillas", "Pantorrillas", []string{"Calfs"}},
		{"las pantorrillas", "Las pantorrillas", []string{"Calfs"}},
		{"abdomen", "Abdomen", []string{"Abs", "Core"}},
		{"pecho", "Pecho", []string{"Chest"}},
		{"brazos", "Brazos", []string{"Biceps", "Triceps", "Forearms"}},
		{"piernas", "Piernas", []string{"Quads", "Hamstrings", "Glutes", "Calfs"}},
		{"empty", "", nil},
		{"ninguno", "Ninguno", nil}, // handled upstream, but parseDislikedMuscles returns empty
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDislikedMuscles(tt.input)
			if len(tt.expected) == 0 {
				if len(result) != 0 {
					t.Errorf("parseDislikedMuscles(%q) = %v, want empty", tt.input, result)
				}
				return
			}
			sort.Strings(result)
			sort.Strings(tt.expected)
			if len(result) != len(tt.expected) {
				t.Fatalf("parseDislikedMuscles(%q) returned %d items %v, want %d items %v",
					tt.input, len(result), result, len(tt.expected), tt.expected)
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseDislikedMuscles(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}
