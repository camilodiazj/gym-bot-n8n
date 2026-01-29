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
	conn *Connection
}

// Ensure WorkoutRepository implements the interface
var _ repository.WorkoutRepository = (*WorkoutRepository)(nil)

// NewWorkoutRepository creates a new WorkoutRepository
func NewWorkoutRepository(conn *Connection) *WorkoutRepository {
	return &WorkoutRepository{conn: conn}
}

// parseReps parses reps from string, handling ranges like "10-12" (returns first number)
func parseReps(repsStr string) int {
	// Handle range format "10-12" -> take first number
	for i, c := range repsStr {
		if c == '-' && i > 0 {
			reps, _ := strconv.Atoi(repsStr[:i])
			return reps
		}
	}
	// Single number
	reps, _ := strconv.Atoi(repsStr)
	return reps
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
			e.spanish_name,
			w.sets,
			w.reps,
			w.rir,
			e.link
		FROM workouts w
		JOIN exercises e ON w.exercise_id = e.exercise_id
		WHERE w.user_id = $1
		AND w.week = $2
		AND w.day_name = $3
		ORDER BY w.id
	`

	rows, err := r.conn.DB.QueryContext(ctx, exerciseQuery, userID, week, sessionName)
	if err != nil {
		return nil, apperror.NewInternalError("failed to query exercises", err)
	}
	defer rows.Close()

	// Badge colors for exercises
	badgeColors := []string{"#22C55E", "#3B82F6", "#A855F7", "#F59E0B", "#EF4444", "#06B6D4"}
	colorIdx := 0

	for rows.Next() {
		var workoutID, exerciseName string
		var setsStr, repsStr string
		var rir sql.NullString
		var link sql.NullString

		if err := rows.Scan(&workoutID, &exerciseName, &setsStr, &repsStr, &rir, &link); err != nil {
			return nil, apperror.NewInternalError("failed to scan exercise", err)
		}

		// Parse sets and reps from text to int
		// Note: both can be ranges like "3-4" or "10-12", we take the first number
		sets := parseReps(setsStr)
		reps := parseReps(repsStr)

		exercise := entity.NewExercise(
			workoutID,
			exerciseName,
			badgeColors[colorIdx%len(badgeColors)],
			rir.String,
			link.String,
		)
		colorIdx++

		// Create sets for the exercise
		for i := 1; i <= sets; i++ {
			set := entity.NewSet(
				workoutID+"-"+strconv.Itoa(i), // Generate set ID
				workoutID,
				i,
				reps,
				"-", // Default weight, will be updated by user
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
