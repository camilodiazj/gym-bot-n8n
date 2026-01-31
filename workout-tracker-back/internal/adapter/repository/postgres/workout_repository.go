package postgres

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// WorkoutRepository implements repository.WorkoutRepository using PostgreSQL
type WorkoutRepository struct {
	conn    *Connection
	setRepo repository.SetReader
}

// Ensure WorkoutRepository implements the interface
var _ repository.WorkoutRepository = (*WorkoutRepository)(nil)

// NewWorkoutRepository creates a new WorkoutRepository
func NewWorkoutRepository(conn *Connection, setRepo repository.SetReader) *WorkoutRepository {
	return &WorkoutRepository{conn: conn, setRepo: setRepo}
}

// parseReps parses reps from string, handling ranges like "10-12" (returns first number)
// Supports both hyphen (-) and en-dash (–)
func parseReps(repsStr string) int {
	// Handle range format "10-12" or "10–12" -> take first number
	for i, c := range repsStr {
		if (c == '-' || c == '–') && i > 0 {
			reps, _ := strconv.Atoi(repsStr[:i])
			return reps
		}
	}
	// Single number
	reps, _ := strconv.Atoi(repsStr)
	return reps
}

// parseRepsRange parses reps from string, returning min and max values
// For "10-12" returns (10, 12), for "10" returns (10, 10)
// Supports both hyphen (-) and en-dash (–)
func parseRepsRange(repsStr string) (min, max int) {
	for i, c := range repsStr {
		if (c == '-' || c == '–') && i > 0 {
			min, _ = strconv.Atoi(repsStr[:i])
			// Skip the separator character (can be multi-byte for en-dash)
			rest := repsStr[i:]
			if rest[0] == '-' {
				max, _ = strconv.Atoi(rest[1:])
			} else {
				// en-dash is 3 bytes in UTF-8
				max, _ = strconv.Atoi(rest[3:])
			}
			return min, max
		}
	}
	// Single number - min equals max
	val, _ := strconv.Atoi(repsStr)
	return val, val
}

// GetTodayWorkout retrieves today's workout for a user
func (r *WorkoutRepository) GetTodayWorkout(ctx context.Context, userID string) (*entity.Workout, error) {
	// First, get the scheduled workout session for today
	// Note: planned_day is stored as TEXT in format 'YYYY-MM-DD'
	scheduleQuery := `
		SELECT
			uws.day_routine_id,
			uws.week,
			uws.week_day,
			uws.session_name,
			uws.planned_day,
			uws."Completed"
		FROM user_weekly_schedule uws
		WHERE uws.user_id = $1
		AND uws.planned_day = to_char((NOW() AT TIME ZONE 'America/Bogota')::date, 'YYYY-MM-DD')
		LIMIT 1
	`

	var scheduleID, dayName, sessionName, plannedDayStr string
	var week int
	var completed bool

	err := r.conn.DB.QueryRowContext(ctx, scheduleQuery, userID).Scan(
		&scheduleID, &week, &dayName, &sessionName, &plannedDayStr, &completed,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No workout scheduled for today
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query schedule", err)
	}

	// Parse the planned_day string to time.Time
	plannedDay, _ := time.Parse("2006-01-02", plannedDayStr)

	workout := entity.NewWorkout(scheduleID, userID, week, dayName, sessionName, plannedDay)
	workout.Completed = completed

	// Get exercises for this workout
	// Note: workouts.day_name matches session_name (e.g., "Lower B"), not week_day (e.g., "Sabado")
	exerciseQuery := `
		SELECT
			w.id,
			w.exercise_id,
			e.spanish_name,
			w.sets,
			w.reps,
			w.rir,
			w."rest-seconds",
			e.link
		FROM workouts w
		JOIN exercises e ON w.exercise_id = e.exercise_id
		WHERE w.user_id = $1
		AND w.week = $2
		AND w.day_name = $3
		ORDER BY w.exercise_order
	`

	rows, err := r.conn.DB.QueryContext(ctx, exerciseQuery, userID, week, sessionName)
	if err != nil {
		return nil, apperror.NewInternalError("failed to query exercises", err)
	}
	defer rows.Close()

	// Badge color - neutral dark gray for all exercises
	badgeColor := "#374151"

	for rows.Next() {
		var workoutID, exerciseID, exerciseName string
		var setsStr, repsStr string
		var rir sql.NullString
		var restSeconds sql.NullInt64
		var link sql.NullString

		if err := rows.Scan(&workoutID, &exerciseID, &exerciseName, &setsStr, &repsStr, &rir, &restSeconds, &link); err != nil {
			return nil, apperror.NewInternalError("failed to scan exercise", err)
		}

		// Parse sets and reps from text to int
		// Note: both can be ranges like "3-4" or "10-12"
		sets := parseReps(setsStr)
		minReps, maxReps := parseRepsRange(repsStr)

		exercise := entity.NewExercise(
			workoutID,
			exerciseName,
			badgeColor,
			rir.String,
			int(restSeconds.Int64),
			link.String,
		)

		// Get historical weights for this exercise (from previous weeks)
		lastWeights := make(map[int]string)
		if r.setRepo != nil {
			lastWeights, _ = r.setRepo.GetLastWeightsForExercise(ctx, userID, exerciseID)
		}

		// Create sets for the exercise with pre-filled weights from history
		// Reps are distributed progressively: Set 1 = min, Set 2 = min+1, ... capped at max
		for i := 1; i <= sets; i++ {
			weight := "-" // Default weight
			if w, exists := lastWeights[i]; exists {
				weight = w // Use historical weight for this specific set
			} else if w, exists := lastWeights[1]; exists && i > 1 {
				// Fallback to Set 1's weight if specific set not found
				weight = w
			}

			// Progressive reps: start at min, increment by 1 per set, cap at max
			reps := minReps + (i - 1)
			if reps > maxReps {
				reps = maxReps
			}

			set := entity.NewSet(
				workoutID+"-"+strconv.Itoa(i), // Generate set ID
				workoutID,
				i,
				reps,
				weight,
			)
			exercise.AddSet(*set)
		}

		// TODO: Add tips and steps from exercise details
		exercise.AddTip(entity.Tip{Text: "Mantén la espalda recta durante todo el movimiento"})
		exercise.AddStep(entity.Step{Text: "Posición inicial: pies al ancho de hombros"})

		workout.AddExercise(*exercise)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.NewInternalError("error iterating exercises", err)
	}

	return workout, nil
}

// GetByID retrieves a workout by its ID
func (r *WorkoutRepository) GetByID(ctx context.Context, workoutID string) (*entity.Workout, error) {
	query := `
		SELECT
			uws.day_routine_id,
			uws.user_id,
			uws.week,
			uws.week_day,
			uws.session_name,
			uws.planned_day,
			uws."Completed"
		FROM user_weekly_schedule uws
		WHERE uws.day_routine_id = $1
	`

	var scheduleID, userID, dayName, sessionName, plannedDayStr string
	var week int
	var completed bool

	err := r.conn.DB.QueryRowContext(ctx, query, workoutID).Scan(
		&scheduleID, &userID, &week, &dayName, &sessionName, &plannedDayStr, &completed,
	)
	if err == sql.ErrNoRows {
		return nil, apperror.NewNotFoundError("workout not found")
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query workout", err)
	}

	// Parse the planned_day string to time.Time
	plannedDay, _ := time.Parse("2006-01-02", plannedDayStr)

	workout := entity.NewWorkout(scheduleID, userID, week, dayName, sessionName, plannedDay)
	workout.Completed = completed

	return workout, nil
}

// MarkComplete marks a workout as completed
func (r *WorkoutRepository) MarkComplete(ctx context.Context, workoutID string) error {
	query := `
		UPDATE user_weekly_schedule
		SET "Completed" = true
		WHERE day_routine_id = $1
	`

	result, err := r.conn.DB.ExecContext(ctx, query, workoutID)
	if err != nil {
		return apperror.NewInternalError("failed to mark workout complete", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperror.NewInternalError("failed to check rows affected", err)
	}

	if rowsAffected == 0 {
		return apperror.NewNotFoundError("workout not found")
	}

	return nil
}
