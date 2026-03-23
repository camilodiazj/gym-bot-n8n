package entity

import "time"

// DraftAlternative represents an alternative exercise option in a draft routine
type DraftAlternative struct {
	ExerciseID  string `json:"exercise_id"`
	SpanishName string `json:"spanish_name"`
	MainMuscle  string `json:"main_muscle,omitempty"`
	VideoLink   string `json:"video_link,omitempty"`
}

// DraftExercise represents a single exercise in a draft routine day
type DraftExercise struct {
	ExerciseID    string             `json:"exercise_id"`
	SpanishName   string             `json:"spanish_name"`
	Pattern       string             `json:"pattern"`
	Role          string             `json:"role"`
	MainMuscle    string             `json:"main_muscle,omitempty"`
	VideoLink     string             `json:"video_link,omitempty"`
	Sets          int                `json:"sets"`
	Reps          string             `json:"reps"`
	RIR           string             `json:"rir"`
	RestSeconds   int                `json:"rest_seconds"`
	ExerciseOrder int                `json:"exercise_order"`
	Alternatives  []DraftAlternative `json:"alternatives"`
}

// DraftDay represents a single training day in a draft routine
type DraftDay struct {
	DayNumber int             `json:"day_number"`
	Title     string          `json:"title"`
	Exercises []DraftExercise `json:"exercises"`
}

// DraftData holds the JSONB content of a draft routine
type DraftData struct {
	WeekSchedule string     `json:"week_schedule"`
	Goal         string     `json:"goal"`
	Level        string     `json:"level"`
	Days         []DraftDay `json:"days"`
}

// DraftRoutine represents a draft routine pending user approval
type DraftRoutine struct {
	DraftID   string    `json:"draft_id"`
	UserID    string    `json:"user_id"`
	Code      string    `json:"code"`
	DraftData DraftData `json:"draft_data"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// FindExerciseAt locates an exercise in the draft by day number and exercise order.
// Returns pointers to the day and exercise, or nil if not found.
func (d *DraftData) FindExerciseAt(dayNumber, exerciseOrder int) (*DraftDay, *DraftExercise) {
	for i := range d.Days {
		if d.Days[i].DayNumber == dayNumber {
			for j := range d.Days[i].Exercises {
				if d.Days[i].Exercises[j].ExerciseOrder == exerciseOrder {
					return &d.Days[i], &d.Days[i].Exercises[j]
				}
			}
			return &d.Days[i], nil
		}
	}
	return nil, nil
}

// SwapExercise swaps the primary exercise with one of its alternatives.
// The current primary moves into the alternatives list, the chosen alternative becomes primary.
// Returns false if the new exercise is not found in the alternatives list.
func (ex *DraftExercise) SwapExercise(newExerciseID string) bool {
	targetIdx := -1
	for i, alt := range ex.Alternatives {
		if alt.ExerciseID == newExerciseID {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		return false
	}

	chosen := ex.Alternatives[targetIdx]

	// Build new alternatives: remove chosen, add current primary
	newAlternatives := make([]DraftAlternative, 0, len(ex.Alternatives))
	newAlternatives = append(newAlternatives, DraftAlternative{
		ExerciseID:  ex.ExerciseID,
		SpanishName: ex.SpanishName,
		MainMuscle:  ex.MainMuscle,
		VideoLink:   ex.VideoLink,
	})
	for i, alt := range ex.Alternatives {
		if i != targetIdx {
			newAlternatives = append(newAlternatives, alt)
		}
	}

	// Promote chosen to primary
	ex.ExerciseID = chosen.ExerciseID
	ex.SpanishName = chosen.SpanishName
	ex.MainMuscle = chosen.MainMuscle
	ex.VideoLink = chosen.VideoLink
	ex.Alternatives = newAlternatives

	return true
}
