package entity

// Tip represents a form/technique tip for an exercise
type Tip struct {
	Text string `json:"text"`
}

// Step represents an instruction step for an exercise
type Step struct {
	Text string `json:"text"`
}

// Exercise represents a single exercise in a workout
type Exercise struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	BadgeColor string `json:"badge_color"`
	RIR        string `json:"rir"`  // Reps In Reserve (e.g., "3-4")
	Link       string `json:"link"` // MuscleWiki video URL
	Sets       []Set  `json:"sets"`
	Tips       []Tip  `json:"tips"`
	Steps      []Step `json:"steps"`
}

// NewExercise creates a new Exercise entity
func NewExercise(id, name, badgeColor, rir, link string) *Exercise {
	return &Exercise{
		ID:         id,
		Name:       name,
		BadgeColor: badgeColor,
		RIR:        rir,
		Link:       link,
		Sets:       []Set{},
		Tips:       []Tip{},
		Steps:      []Step{},
	}
}

// AddSet adds a set to the exercise
func (e *Exercise) AddSet(set Set) {
	e.Sets = append(e.Sets, set)
}

// AddTip adds a tip to the exercise
func (e *Exercise) AddTip(tip Tip) {
	e.Tips = append(e.Tips, tip)
}

// AddStep adds a step to the exercise
func (e *Exercise) AddStep(step Step) {
	e.Steps = append(e.Steps, step)
}

// AllSetsCompleted returns true if all sets are completed
func (e *Exercise) AllSetsCompleted() bool {
	if len(e.Sets) == 0 {
		return false
	}
	for _, set := range e.Sets {
		if !set.Completed {
			return false
		}
	}
	return true
}

// CompletedSetsCount returns the number of completed sets
func (e *Exercise) CompletedSetsCount() int {
	count := 0
	for _, set := range e.Sets {
		if set.Completed {
			count++
		}
	}
	return count
}
