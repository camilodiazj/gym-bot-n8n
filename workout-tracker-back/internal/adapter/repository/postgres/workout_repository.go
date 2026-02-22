package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/internal/pkg/timezone"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// exerciseBuildData holds intermediate data during the two-pass exercise processing.
// First pass scans all exercises; second pass queries alternatives before adding to workout.
type exerciseBuildData struct {
	exercise   *entity.Exercise
	workoutID  string
	exerciseID string
	pattern    string
	role       string
	mainMuscle string
}

// userFilterContext holds pre-processed user profile data for filtering alternative exercises.
// Zero-values disable all filtering (graceful degradation when no profile exists).
type userFilterContext struct {
	hasProfile       bool
	trainingEnv      string   // "GYM" or "HOME"
	allowedEquipment []string // canonical equipment values for HOME users; nil for GYM
	healthStatus     string   // "A" through "E"
	fitnessLevel     string   // "Principiante", "Intermedio", "Avanzado"
	dislikedMuscles  []string // English main_muscle values to exclude
}

// equipmentMap maps Spanish home_equipment input tokens to canonical exercises.equipment DB values.
var equipmentMap = map[string]string{
	"mancuernas":       "dumbbell",
	"mancuerna":        "dumbbell",
	"dumbbells":        "dumbbell",
	"dumbbell":         "dumbbell",
	"pesas":            "dumbbell",
	"kettlebell":       "kettlebell",
	"kettlebells":      "kettlebell",
	"pesa rusa":        "kettlebell",
	"pesas rusas":      "kettlebell",
	"barra":            "barbell",
	"barras":           "barbell",
	"barbell":          "barbell",
	"barra olimpica":   "barbell",
	"bandas":           "resistance_band",
	"bandas elasticas": "resistance_band",
	"ligas":            "resistance_band",
	"peso corporal":    "bodyweight",
	"solo cuerpo":      "bodyweight",
	"bodyweight":       "bodyweight",
}

// spanishEquipmentToCanonical maps the ~20 exercises that have Spanish equipment values.
var spanishEquipmentToCanonical = map[string]string{
	"Mancuerna":     "dumbbell",
	"Máquina":       "machine",
	"Polea":         "cable",
	"Peso Corporal": "bodyweight",
	"Barra":         "barbell",
}

// dislikedMuscleMap maps Spanish disliked_exercises to English main_muscle values.
var dislikedMuscleMap = map[string][]string{
	"pantorrillas":     {"Calfs"},
	"las pantorrillas": {"Calfs"},
	"abdomen":          {"Abs", "Core"},
	"pecho":            {"Chest"},
	"piernas":          {"Quads", "Hamstrings", "Glutes", "Calfs"},
	"espalda":          {"Back", "Lower back", "Traps", "Traps (mid-back)"},
	"brazos":           {"Biceps", "Triceps", "Forearms"},
	"hombros":          {"Shoulders", "Front Shoulders"},
	"tren superior":    {"Chest", "Back", "Shoulders", "Front Shoulders", "Biceps", "Triceps", "Traps", "Upper Traps"},
}

func stripAccents(s string) string {
	r := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U",
		"ñ", "n", "Ñ", "N",
	)
	return r.Replace(s)
}

func levelHierarchy(userLevel string) []string {
	switch userLevel {
	case "Principiante":
		return []string{"Principiante"}
	case "Intermedio":
		return []string{"Principiante", "Intermedio"}
	case "Avanzado":
		return []string{"Principiante", "Intermedio", "Avanzado"}
	default:
		return nil
	}
}

func parseHomeEquipment(raw string) []string {
	equipSet := map[string]bool{"bodyweight": true, "Peso Corporal": true}

	normalized := strings.ToLower(stripAccents(raw))
	for token, canonical := range equipmentMap {
		if strings.Contains(normalized, token) {
			equipSet[canonical] = true
			for spanish, canon := range spanishEquipmentToCanonical {
				if canon == canonical {
					equipSet[spanish] = true
				}
			}
		}
	}

	result := make([]string, 0, len(equipSet))
	for eq := range equipSet {
		result = append(result, eq)
	}
	return result
}

func parseDislikedMuscles(raw string) []string {
	muscleSet := map[string]bool{}
	normalized := strings.ToLower(stripAccents(raw))

	for spanish, muscles := range dislikedMuscleMap {
		if strings.Contains(normalized, spanish) {
			for _, m := range muscles {
				muscleSet[m] = true
			}
		}
	}

	result := make([]string, 0, len(muscleSet))
	for m := range muscleSet {
		result = append(result, m)
	}
	return result
}

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

// loadUserFilterContext fetches the user's gym profile and transforms it into
// a filter context used to narrow down alternative exercise candidates.
// Returns a zero-value context (all filters disabled) when no profile is found.
func (r *WorkoutRepository) loadUserFilterContext(ctx context.Context, userID string) userFilterContext {
	query := `
		SELECT ugp.training_environment, ugp.health_status, ugp.fitness_level,
		       ugp.home_equipment, ugp.disliked_exercises
		FROM users u
		LEFT JOIN users_gym_profile ugp ON u.cel_number = ugp.whatsapp_id
		WHERE u.user_id = $1
		LIMIT 1
	`
	var env, health, level, homeEquip, disliked sql.NullString

	err := r.conn.DB.QueryRowContext(ctx, query, userID).Scan(
		&env, &health, &level, &homeEquip, &disliked,
	)
	if err != nil {
		fmt.Printf("[WARN] failed to load user filter context for %s: %v\n", userID, err)
		return userFilterContext{}
	}

	if !env.Valid {
		return userFilterContext{}
	}

	fc := userFilterContext{
		hasProfile:   true,
		trainingEnv:  env.String,
		healthStatus: health.String,
		fitnessLevel: level.String,
	}

	if fc.trainingEnv == "HOME" && homeEquip.Valid && homeEquip.String != "" && homeEquip.String != "null" {
		fc.allowedEquipment = parseHomeEquipment(homeEquip.String)
	}

	if disliked.Valid && disliked.String != "" && disliked.String != "Ninguno" && disliked.String != "Ninguna" {
		fc.dislikedMuscles = parseDislikedMuscles(disliked.String)
	}

	return fc
}

// loadRecordedSets batch-loads all set_values rows for the given workout IDs.
// Returns a set of "workoutID:exerciseID:setNumber" keys that exist in set_values.
// A row in set_values means the user interacted with that set (weight/reps/complete).
func (r *WorkoutRepository) loadRecordedSets(ctx context.Context, workoutIDs []string) map[string]bool {
	if len(workoutIDs) == 0 {
		return nil
	}

	query := `
		SELECT workout_id, exercise_id, set_number
		FROM set_values
		WHERE workout_id = ANY($1::uuid[])
	`
	rows, err := r.conn.DB.QueryContext(ctx, query, workoutIDs)
	if err != nil {
		fmt.Printf("[WARN] failed to load recorded sets: %v\n", err)
		return nil
	}
	defer rows.Close()

	recorded := make(map[string]bool)
	for rows.Next() {
		var wID, eID string
		var setNum int
		if err := rows.Scan(&wID, &eID, &setNum); err != nil {
			continue
		}
		key := fmt.Sprintf("%s:%s:%d", wID, eID, setNum)
		recorded[key] = true
	}
	return recorded
}

// GetTodayWorkout retrieves today's workout for a user, with yesterday fallback
func (r *WorkoutRepository) GetTodayWorkout(ctx context.Context, userID string) (*entity.Workout, error) {
	// First, get the scheduled uncompleted workout session for today (fallback to yesterday)
	// Note: planned_day_utc is stored as TIMESTAMPTZ (midnight Bogota in UTC = 05:00:00+00)
	todayQuery := `
		SELECT
			uws.day_routine_id,
			uws.week,
			uws.week_day,
			uws.session_name,
			uws.planned_day_utc,
			uws."Completed",
			up.goal,
			up.level
		FROM user_weekly_schedule uws
		JOIN users_plans up ON up.user_id = uws.user_id AND up.status = 'active'
		WHERE uws.user_id = $1
		AND uws."Completed" = false
		AND uws.planned_day_utc = (
			DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota')
			AT TIME ZONE 'America/Bogota'
		)
		ORDER BY uws.day_routine_id
		LIMIT 1
	`

	yesterdayQuery := `
		SELECT
			uws.day_routine_id,
			uws.week,
			uws.week_day,
			uws.session_name,
			uws.planned_day_utc,
			uws."Completed",
			up.goal,
			up.level
		FROM user_weekly_schedule uws
		JOIN users_plans up ON up.user_id = uws.user_id AND up.status = 'active'
		WHERE uws.user_id = $1
		AND uws."Completed" = false
		AND uws.planned_day_utc = (
			DATE_TRUNC('day', (NOW() AT TIME ZONE 'America/Bogota') - INTERVAL '1 day')
			AT TIME ZONE 'America/Bogota'
		)
		ORDER BY uws.day_routine_id
		LIMIT 1
	`

	var scheduleID, dayName, sessionName, goal, level string
	var plannedDay time.Time
	var week int
	var completed bool

	err := r.conn.DB.QueryRowContext(ctx, todayQuery, userID).Scan(
		&scheduleID, &week, &dayName, &sessionName, &plannedDay, &completed, &goal, &level,
	)
	if err == sql.ErrNoRows {
		// Fallback: yesterday's uncompleted workout
		err = r.conn.DB.QueryRowContext(ctx, yesterdayQuery, userID).Scan(
			&scheduleID, &week, &dayName, &sessionName, &plannedDay, &completed, &goal, &level,
		)
		if err == sql.ErrNoRows {
			return nil, nil // No pending workout (rest day)
		}
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query schedule", err)
	}

	// plannedDay is already a time.Time from TIMESTAMPTZ, ensure it's in UTC
	plannedDay = timezone.ToUTC(plannedDay)

	filterCtx := r.loadUserFilterContext(ctx, userID)

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
			e.link,
			e.pattern,
			e.role,
			e.main_muscle
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

	var exerciseDataList []exerciseBuildData
	var primaryExerciseIDs []string

	// First pass: scan all exercises and build entities (without alternatives)
	for rows.Next() {
		var workoutID, exerciseID, exerciseName string
		var setsStr, repsStr string
		var rir sql.NullString
		var restSeconds sql.NullInt64
		var link sql.NullString
		var pattern, role string
		var mainMuscle string

		if err := rows.Scan(&workoutID, &exerciseID, &exerciseName, &setsStr, &repsStr, &rir, &restSeconds, &link, &pattern, &role, &mainMuscle); err != nil {
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

		exerciseDataList = append(exerciseDataList, exerciseBuildData{
			exercise:   exercise,
			workoutID:  workoutID,
			exerciseID: exerciseID,
			pattern:    pattern,
			role:       role,
			mainMuscle: mainMuscle,
		})
		primaryExerciseIDs = append(primaryExerciseIDs, exerciseID)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.NewInternalError("error iterating exercises", err)
	}

	// Batch-load set completion state for all workout IDs in this session
	workoutIDs := make([]string, 0, len(exerciseDataList))
	for _, ed := range exerciseDataList {
		workoutIDs = append(workoutIDs, ed.workoutID)
	}
	recordedSets := r.loadRecordedSets(ctx, workoutIDs)

	// Second pass: apply completion state, query alternatives, then add to workout
	for i := range exerciseDataList {
		ed := &exerciseDataList[i]

		// Mark primary sets as completed if recorded in set_values
		for j := range ed.exercise.Sets {
			s := &ed.exercise.Sets[j]
			key := fmt.Sprintf("%s:%s:%d", ed.workoutID, ed.exerciseID, s.SetNumber)
			if recordedSets[key] {
				s.Completed = true
			}
		}

		alternatives := r.queryAlternatives(ctx, ed.workoutID, ed.exerciseID, ed.pattern, ed.role, ed.mainMuscle, primaryExerciseIDs, userID, goal, level, week, filterCtx, recordedSets)
		ed.exercise.Alternatives = alternatives
		workout.AddExercise(*ed.exercise)
	}

	return workout, nil
}

// queryAlternatives finds up to 2 alternative exercises matching the same movement pattern,
// excluding the primary exercise and all other primary exercises in the session (BR-7 dedup).
// Errors are logged but do not fail the request (graceful degradation).
func (r *WorkoutRepository) queryAlternatives(
	ctx context.Context,
	workoutID, exerciseID, pattern, primaryRole, primaryMainMuscle string,
	primaryIDs []string,
	userID, goal, level string,
	week int,
	fc userFilterContext,
	recordedSets map[string]bool,
) []entity.AlternativeExercise {
	// Build dynamic query with parameterized conditions
	args := []interface{}{pattern, primaryIDs}
	paramIdx := 2

	altQuery := `
		SELECT exercise_id, spanish_name, role, link
		FROM exercises
		WHERE pattern = $1
		  AND exercise_id != ALL($2::text[])`

	// --- Tier 1: Equipment availability (HOME users only) ---
	if fc.hasProfile && fc.trainingEnv == "HOME" && len(fc.allowedEquipment) > 0 {
		paramIdx++
		altQuery += fmt.Sprintf("\n\t\t  AND equipment = ANY($%d::text[])", paramIdx)
		args = append(args, fc.allowedEquipment)
	}

	// --- Tier 1: Health status exclusions ---
	if fc.hasProfile {
		switch fc.healthStatus {
		case "B":
			altQuery += `
		  AND NOT (
			main_muscle IN ('Quads', 'Hamstrings', 'Glutes', 'Calfs')
			AND equipment IN ('barbell', 'smith', 'Barra')
			AND (spanish_name ILIKE '%sentadilla%' OR spanish_name ILIKE '%peso muerto%'
				 OR spanish_name ILIKE '%zancada%' OR spanish_name ILIKE '%salto%'
				 OR spanish_name ILIKE '%prensa%pierna%')
		  )`
		case "C":
			altQuery += `
		  AND NOT (
			pattern IN ('push_v')
			AND (spanish_name ILIKE '%press%militar%'
				 OR spanish_name ILIKE '%press%hombro%'
				 OR spanish_name ILIKE '%overhead%'
				 OR spanish_name ILIKE '%tras nuca%'
				 OR spanish_name ILIKE '%snatch%')
		  )`
		case "D":
			altQuery += `
		  AND NOT (
			(spanish_name ILIKE '%peso muerto%'
			 OR spanish_name ILIKE '%sentadilla%barra%'
			 OR spanish_name ILIKE '%good morning%'
			 OR spanish_name ILIKE '%rack pull%'
			 OR spanish_name ILIKE '%clean%'
			 OR spanish_name ILIKE '%snatch%')
			AND equipment IN ('barbell', 'smith', 'Barra')
		  )`
		case "E":
			altQuery += `
		  AND equipment IN ('machine', 'bodyweight', 'cable', 'Máquina', 'Peso Corporal', 'Polea')`
		}
	}

	// --- Tier 2: Role preservation ---
	if primaryRole != "" {
		paramIdx++
		altQuery += fmt.Sprintf("\n\t\t  AND role = $%d", paramIdx)
		args = append(args, primaryRole)
	}

	// --- Tier 2: Level filtering ---
	if fc.hasProfile && fc.fitnessLevel != "" {
		allowedLevels := levelHierarchy(fc.fitnessLevel)
		if len(allowedLevels) > 0 {
			paramIdx++
			altQuery += fmt.Sprintf("\n\t\t  AND level = ANY($%d::text[])", paramIdx)
			args = append(args, allowedLevels)
		}
	}

	// --- Tier 3: Disliked muscles exclusion ---
	if len(fc.dislikedMuscles) > 0 {
		paramIdx++
		altQuery += fmt.Sprintf("\n\t\t  AND main_muscle != ALL($%d::text[])", paramIdx)
		args = append(args, fc.dislikedMuscles)
	}

	// --- Tier 3: Prefer same main muscle, deterministic ordering ---
	// Use md5(exercise_id || workoutID) instead of RANDOM() so alternatives
	// are stable across page reloads for the same workout.
	paramIdx++
	seedParam := paramIdx
	args = append(args, workoutID)

	if primaryMainMuscle != "" {
		paramIdx++
		altQuery += fmt.Sprintf(`
		ORDER BY
			CASE WHEN main_muscle = $%d THEN 0 ELSE 1 END,
			md5(exercise_id || $%d)
		LIMIT 2`, paramIdx, seedParam)
		args = append(args, primaryMainMuscle)
	} else {
		altQuery += fmt.Sprintf(`
		ORDER BY md5(exercise_id || $%d)
		LIMIT 2`, seedParam)
	}

	altRows, err := r.conn.DB.QueryContext(ctx, altQuery, args...)
	if err != nil {
		fmt.Printf("[WARN] failed to query alternatives for exercise %s: %v\n", exerciseID, err)
		return nil
	}
	defer altRows.Close()

	alternatives := r.buildAlternativesFromRows(ctx, altRows, workoutID, exerciseID, userID, goal, level, week, recordedSets)

	// Fallback: if filtered query returned no alternatives and we had filters active,
	// retry with only pattern + dedup (no role/level/disliked/muscle filters)
	if len(alternatives) == 0 && fc.hasProfile {
		alternatives = r.queryAlternativesFallback(ctx, workoutID, exerciseID, pattern,
			primaryIDs, userID, goal, level, week, fc, recordedSets)
	}

	return alternatives
}

// queryAlternativesFallback retries with only Tier 1 filters (equipment + health).
// Called when the full-filtered query returns zero results.
func (r *WorkoutRepository) queryAlternativesFallback(
	ctx context.Context,
	workoutID, exerciseID, pattern string,
	primaryIDs []string,
	userID, goal, level string,
	week int,
	fc userFilterContext,
	recordedSets map[string]bool,
) []entity.AlternativeExercise {
	args := []interface{}{pattern, primaryIDs}
	paramIdx := 2

	fallbackQuery := `
		SELECT exercise_id, spanish_name, role, link
		FROM exercises
		WHERE pattern = $1
		  AND exercise_id != ALL($2::text[])`

	// Only keep Tier 1: equipment + health
	if fc.trainingEnv == "HOME" && len(fc.allowedEquipment) > 0 {
		paramIdx++
		fallbackQuery += fmt.Sprintf("\n\t\t  AND equipment = ANY($%d::text[])", paramIdx)
		args = append(args, fc.allowedEquipment)
	}

	if fc.healthStatus == "E" {
		fallbackQuery += `
		  AND equipment IN ('machine', 'bodyweight', 'cable', 'Máquina', 'Peso Corporal', 'Polea')`
	}

	paramIdx++
	fallbackQuery += fmt.Sprintf(`
		ORDER BY md5(exercise_id || $%d)
		LIMIT 2`, paramIdx)
	args = append(args, workoutID)

	altRows, err := r.conn.DB.QueryContext(ctx, fallbackQuery, args...)
	if err != nil {
		fmt.Printf("[WARN] fallback query failed for exercise %s: %v\n", exerciseID, err)
		return nil
	}
	defer altRows.Close()

	return r.buildAlternativesFromRows(ctx, altRows, workoutID, exerciseID, userID, goal, level, week, recordedSets)
}

// buildAlternativesFromRows scans exercise rows and builds AlternativeExercise entities
// with set profiles and weight history. Shared by queryAlternatives and its fallback.
func (r *WorkoutRepository) buildAlternativesFromRows(
	ctx context.Context,
	altRows *sql.Rows,
	workoutID, exerciseID, userID, goal, level string,
	week int,
	recordedSets map[string]bool,
) []entity.AlternativeExercise {
	var alternatives []entity.AlternativeExercise

	for altRows.Next() {
		var altExerciseID, altName, altRole string
		var altLink sql.NullString

		if err := altRows.Scan(&altExerciseID, &altName, &altRole, &altLink); err != nil {
			fmt.Printf("[WARN] failed to scan alternative exercise: %v\n", err)
			continue
		}

		// Query set_profiles for the alternative's role to get proper loading parameters
		var altSetsStr, altRepsStr, altRIR string
		var altRestSec int
		profileFound := false

		profileQuery := `
			SELECT sets, reps, rir, rest_sec
			FROM set_profiles
			WHERE goal = $1 AND level = $2 AND week = $3 AND role = $4
			LIMIT 1
		`
		err := r.conn.DB.QueryRowContext(ctx, profileQuery, goal, level, week, altRole).Scan(
			&altSetsStr, &altRepsStr, &altRIR, &altRestSec,
		)
		if err == nil {
			profileFound = true
		} else if err != sql.ErrNoRows {
			fmt.Printf("[WARN] failed to query set_profiles for alternative %s (role=%s): %v\n", altExerciseID, altRole, err)
		}

		// If no profile found for the alternative's role, try compound as fallback
		if !profileFound && altRole != "compound" {
			err = r.conn.DB.QueryRowContext(ctx, profileQuery, goal, level, week, "compound").Scan(
				&altSetsStr, &altRepsStr, &altRIR, &altRestSec,
			)
			if err == nil {
				profileFound = true
			} else if err != sql.ErrNoRows {
				fmt.Printf("[WARN] failed to query set_profiles fallback for alternative %s: %v\n", altExerciseID, err)
			}
		}

		if !profileFound {
			fmt.Printf("[WARN] no set_profiles found for alternative %s (role=%s), skipping\n", altExerciseID, altRole)
			continue
		}

		altSetsCount := parseReps(altSetsStr)
		altMinReps, altMaxReps := parseRepsRange(altRepsStr)

		// Get historical weights for the alternative exercise
		altLastWeights := make(map[int]string)
		if r.setRepo != nil {
			altLastWeights, _ = r.setRepo.GetLastWeightsForExercise(ctx, userID, altExerciseID)
		}

		// Build sets for the alternative exercise
		altSets := make([]entity.Set, 0, altSetsCount)
		for i := 1; i <= altSetsCount; i++ {
			weight := "-"
			if w, exists := altLastWeights[i]; exists {
				weight = w
			} else if w, exists := altLastWeights[1]; exists && i > 1 {
				weight = w
			}

			reps := altMinReps + (i - 1)
			if reps > altMaxReps {
				reps = altMaxReps
			}

			setID := fmt.Sprintf("%s:%s:%d", workoutID, altExerciseID, i)
			set := entity.NewSet(setID, workoutID, i, reps, weight)
			// Mark as completed if recorded in set_values
			recordKey := fmt.Sprintf("%s:%s:%d", workoutID, altExerciseID, i)
			if recordedSets[recordKey] {
				set.Completed = true
			}
			altSets = append(altSets, *set)
		}

		alt := entity.AlternativeExercise{
			ExerciseID:  altExerciseID,
			Name:        altName,
			RIR:         altRIR,
			RestSeconds: altRestSec,
			Link:        altLink.String,
			Sets:        altSets,
		}
		alternatives = append(alternatives, alt)
	}

	if err := altRows.Err(); err != nil {
		fmt.Printf("[WARN] error iterating alternative exercises for %s: %v\n", exerciseID, err)
	}

	return alternatives
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
			uws.planned_day_utc,
			uws."Completed"
		FROM user_weekly_schedule uws
		WHERE uws.day_routine_id = $1
	`

	var scheduleID, userID, dayName, sessionName string
	var plannedDay time.Time
	var week int
	var completed bool

	err := r.conn.DB.QueryRowContext(ctx, query, workoutID).Scan(
		&scheduleID, &userID, &week, &dayName, &sessionName, &plannedDay, &completed,
	)
	if err == sql.ErrNoRows {
		return nil, apperror.NewNotFoundError("workout not found")
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query workout", err)
	}

	// plannedDay is already a time.Time from TIMESTAMPTZ, ensure it's in UTC
	plannedDay = timezone.ToUTC(plannedDay)

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
