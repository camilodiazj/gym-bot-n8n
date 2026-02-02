package entity

// ExerciseCatalog represents an exercise from the exercises table
type ExerciseCatalog struct {
	ExerciseID       string   `json:"exercise_id"`
	SpanishName      string   `json:"spanish_name"`
	Pattern          string   `json:"pattern"`
	Role             string   `json:"role"`
	MainMuscle       string   `json:"main_muscle"`
	SecondaryMuscles []string `json:"secondary_muscles"`
	Level            string   `json:"level"`
	Link             string   `json:"link,omitempty"`
	Equipment        string   `json:"equipment,omitempty"`
}

// NewExerciseCatalog creates a new ExerciseCatalog entity
func NewExerciseCatalog(exerciseID, spanishName, pattern, role, mainMuscle, level, link, equipment string, secondaryMuscles []string) *ExerciseCatalog {
	return &ExerciseCatalog{
		ExerciseID:       exerciseID,
		SpanishName:      spanishName,
		Pattern:          pattern,
		Role:             role,
		MainMuscle:       mainMuscle,
		SecondaryMuscles: secondaryMuscles,
		Level:            level,
		Link:             link,
		Equipment:        equipment,
	}
}

// MatchesMuscle checks if the exercise targets a specific muscle (main or secondary)
func (e *ExerciseCatalog) MatchesMuscle(muscle string) bool {
	if e.MainMuscle == muscle {
		return true
	}
	for _, m := range e.SecondaryMuscles {
		if m == muscle {
			return true
		}
	}
	return false
}

// MatchesPattern checks if exercise matches the given pattern
func (e *ExerciseCatalog) MatchesPattern(pattern string) bool {
	return e.Pattern == pattern
}

// MatchesRole checks if exercise matches the given role
func (e *ExerciseCatalog) MatchesRole(role string) bool {
	return e.Role == role
}

// CanRotateTo checks if the current exercise can be replaced by the candidate
func (e *ExerciseCatalog) CanRotateTo(candidate *ExerciseCatalog, dislikedMuscles []string) bool {
	// Must match pattern and role
	if e.Pattern != candidate.Pattern || e.Role != candidate.Role {
		return false
	}

	// Must not be the same exercise
	if e.ExerciseID == candidate.ExerciseID {
		return false
	}

	// Must not target disliked muscles
	for _, muscle := range dislikedMuscles {
		if candidate.MainMuscle == muscle {
			return false
		}
	}

	return true
}

// ExerciseRotation represents a rotation from one exercise to another
type ExerciseRotation struct {
	WorkoutID       string           `json:"workout_id"`
	OldExerciseID   string           `json:"old_exercise_id"`
	OldExerciseName string           `json:"old_exercise_name,omitempty"`
	OldExercise     *ExerciseCatalog `json:"old_exercise,omitempty"`
	NewExerciseID   string           `json:"new_exercise_id"`
	NewExerciseName string           `json:"new_exercise_name,omitempty"`
	NewExercise     *ExerciseCatalog `json:"new_exercise,omitempty"`
	Pattern         string           `json:"pattern"`
	Role            string           `json:"role"`
	DayName         string           `json:"day_name,omitempty"`
}

// NewExerciseRotation creates a new ExerciseRotation
func NewExerciseRotation(workoutID, oldExerciseID, newExerciseID, pattern, role string) *ExerciseRotation {
	return &ExerciseRotation{
		WorkoutID:     workoutID,
		OldExerciseID: oldExerciseID,
		NewExerciseID: newExerciseID,
		Pattern:       pattern,
		Role:          role,
	}
}

// WithExerciseNames sets the exercise names for display
func (r *ExerciseRotation) WithExerciseNames(oldName, newName string) *ExerciseRotation {
	r.OldExerciseName = oldName
	r.NewExerciseName = newName
	return r
}

// WorkoutExercise represents an exercise assignment in the workouts table
type WorkoutExercise struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	Week          int    `json:"week"`
	DayName       string `json:"day_name"`
	ExerciseID    string `json:"exercise_id"`
	Sets          string `json:"sets"`
	Reps          string `json:"reps"`
	RIR           string `json:"rir,omitempty"`
	RestSeconds   int    `json:"rest_seconds"`
	Tempo         string `json:"tempo,omitempty"`
	ExerciseOrder int    `json:"exercise_order"`
	// Role and Pattern are fetched from exercises table via JOIN
	Role    string `json:"role,omitempty"`
	Pattern string `json:"pattern,omitempty"`
}

// NewWorkoutExercise creates a new WorkoutExercise entity
func NewWorkoutExercise(id, userID, dayName, exerciseID, sets, reps, rir, tempo string, week, restSeconds, exerciseOrder int) *WorkoutExercise {
	return &WorkoutExercise{
		ID:            id,
		UserID:        userID,
		Week:          week,
		DayName:       dayName,
		ExerciseID:    exerciseID,
		Sets:          sets,
		Reps:          reps,
		RIR:           rir,
		RestSeconds:   restSeconds,
		Tempo:         tempo,
		ExerciseOrder: exerciseOrder,
	}
}

// PatternRoleKey returns a composite key for grouping exercises by pattern and role
func (w *WorkoutExercise) PatternRoleKey(pattern, role string) string {
	return pattern + ":" + role
}
