# Mesocycle Renewal - Backend API Specification

This document provides complete, production-ready Go code for the mesocycle renewal feature, following the existing hexagonal architecture patterns in `workout-tracker-back/`.

## Table of Contents

1. [Domain Entities](#1-domain-entities)
2. [Repository Interfaces](#2-repository-interfaces)
3. [Domain Services](#3-domain-services)
4. [Application Layer (DTOs & Use Cases)](#4-application-layer)
5. [Adapter Layer (Handlers & Repositories)](#5-adapter-layer)
6. [Router Configuration](#6-router-configuration)
7. [Main.go Wiring](#7-maingo-wiring)
8. [Environment Configuration](#8-environment-configuration)

---

## 1. Domain Entities

### 1.1 Plan Entity

**File:** `internal/domain/entity/plan.go`

```go
package entity

import "time"

// MesocycleStatus represents the completion status of a user's current mesocycle
type MesocycleStatus struct {
	UserID           string    `json:"user_id"`
	MesocycleNumber  int       `json:"mesocycle_number"`
	DaysPerWeek      int       `json:"days_per_week"`
	WeekSchedule     string    `json:"week_schedule"`
	Week4Completed   int       `json:"week4_completed"`
	Week4Total       int       `json:"week4_total"`
	IsComplete       bool      `json:"is_complete"`
	LastRenewalDate  *time.Time `json:"last_renewal_date,omitempty"`
	Goal             string    `json:"goal"`
	Level            string    `json:"level"`
}

// NewMesocycleStatus creates a new MesocycleStatus
func NewMesocycleStatus(userID string, mesocycleNumber, daysPerWeek, week4Completed int, weekSchedule string) *MesocycleStatus {
	return &MesocycleStatus{
		UserID:          userID,
		MesocycleNumber: mesocycleNumber,
		DaysPerWeek:     daysPerWeek,
		WeekSchedule:    weekSchedule,
		Week4Completed:  week4Completed,
		Week4Total:      daysPerWeek,
		IsComplete:      week4Completed >= daysPerWeek,
	}
}

// Plan represents a user's training plan
type Plan struct {
	PlanID          string     `json:"plan_id"`
	UserID          string     `json:"user_id"`
	TemplateID      string     `json:"template_id"`
	WeekSchedule    string     `json:"week_schedule"`
	Goal            string     `json:"goal"`
	Level           string     `json:"level"`
	Status          string     `json:"status"`
	MesocycleNumber int        `json:"mesocycle_number"`
	LastRenewalDate *time.Time `json:"last_renewal_date,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// NewPlan creates a new Plan entity
func NewPlan(planID, userID, templateID, weekSchedule, goal, level string, mesocycleNumber int) *Plan {
	return &Plan{
		PlanID:          planID,
		UserID:          userID,
		TemplateID:      templateID,
		WeekSchedule:    weekSchedule,
		Goal:            goal,
		Level:           level,
		Status:          "active",
		MesocycleNumber: mesocycleNumber,
		CreatedAt:       time.Now(),
	}
}

// IncrementMesocycle increments the mesocycle number and updates renewal date
func (p *Plan) IncrementMesocycle() {
	p.MesocycleNumber++
	now := time.Now()
	p.LastRenewalDate = &now
}

// UpdateWeekSchedule updates the weekly training schedule
func (p *Plan) UpdateWeekSchedule(weekSchedule string) {
	p.WeekSchedule = weekSchedule
}

// RenewalResult represents the result of a mesocycle renewal operation
type RenewalResult struct {
	Success            bool   `json:"success"`
	NewMesocycleNumber int    `json:"new_mesocycle_number"`
	RenewalType        string `json:"renewal_type"`
	Message            string `json:"message"`
	ScheduleCleared    bool   `json:"schedule_cleared"`
	WorkoutsUpdated    int    `json:"workouts_updated,omitempty"`
	ExercisesRotated   int    `json:"exercises_rotated,omitempty"`
}

// NewRenewalResult creates a new RenewalResult
func NewRenewalResult(mesocycleNumber int, renewalType, message string) *RenewalResult {
	return &RenewalResult{
		Success:            true,
		NewMesocycleNumber: mesocycleNumber,
		RenewalType:        renewalType,
		Message:            message,
		ScheduleCleared:    true,
	}
}
```

### 1.2 Exercise Catalog Entity

**File:** `internal/domain/entity/exercise_catalog.go`

```go
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
	Link             string   `json:"link"`
	Equipment        string   `json:"equipment"`
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

// ExerciseRotation represents a rotation from one exercise to another
type ExerciseRotation struct {
	WorkoutID      string           `json:"workout_id"`
	OldExerciseID  string           `json:"old_exercise_id"`
	OldExercise    *ExerciseCatalog `json:"old_exercise,omitempty"`
	NewExerciseID  string           `json:"new_exercise_id"`
	NewExercise    *ExerciseCatalog `json:"new_exercise,omitempty"`
	Pattern        string           `json:"pattern"`
	Role           string           `json:"role"`
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

// WorkoutExercise represents an exercise assignment in the workouts table
type WorkoutExercise struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	Week          int    `json:"week"`
	DayName       string `json:"day_name"`
	ExerciseID    string `json:"exercise_id"`
	Sets          string `json:"sets"`
	Reps          string `json:"reps"`
	RIR           string `json:"rir"`
	RestSeconds   int    `json:"rest_seconds"`
	Tempo         string `json:"tempo"`
	ExerciseOrder int    `json:"exercise_order"`
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
```

### 1.3 User Gym Profile Entity

**File:** `internal/domain/entity/user_profile.go`

```go
package entity

import "time"

// UserGymProfile represents the user's fitness profile from KYC
type UserGymProfile struct {
	WhatsAppID           string    `json:"whatsapp_id"`
	FullName             string    `json:"full_name"`
	Email                string    `json:"email"`
	Birthdate            string    `json:"birthdate"`
	Sex                  string    `json:"sex"`
	Height               int       `json:"height"`
	Weight               float64   `json:"weight"`
	TrainingGoal         string    `json:"training_goal"`
	FitnessLevel         string    `json:"fitness_level"`
	TrainingExperience   string    `json:"training_experience"`
	DaysPerWeek          int       `json:"days_per_week"`
	SessionDurationMins  string    `json:"session_duration_mins"`
	HealthStatus         string    `json:"health_status"`
	PriorityMuscles      string    `json:"priority_muscles"`
	DislikedExercises    string    `json:"disliked_exercises"`
	AvailableEquipment   string    `json:"available_equipment"`
	TrainingEnvironment  string    `json:"training_environment"`
	PreferredTrainingTime string   `json:"preferred_training_time"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// NewUserGymProfile creates a new UserGymProfile entity
func NewUserGymProfile(whatsappID, fullName, email string) *UserGymProfile {
	return &UserGymProfile{
		WhatsAppID: whatsappID,
		FullName:   fullName,
		Email:      email,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// HealthRestriction represents health-based exercise restrictions
type HealthRestriction struct {
	AvoidUpperBodyOverhead bool `json:"avoid_upper_body_overhead"`
	AvoidLowerBodyImpact   bool `json:"avoid_lower_body_impact"`
	AvoidAxialLoading      bool `json:"avoid_axial_loading"`
	PreferMachines         bool `json:"prefer_machines"`
	IsSpecialCondition     bool `json:"is_special_condition"`
}

// ProcessedPreferences represents the processed user preferences in English
type ProcessedPreferences struct {
	PriorityMusclesEN   []string           `json:"priority_muscles_en"`
	DislikedMusclesEN   []string           `json:"disliked_muscles_en"`
	ExperienceTier      string             `json:"experience_tier"`
	VolumeModifier      float64            `json:"volume_modifier"`
	HealthRestrictions  HealthRestriction  `json:"health_restrictions"`
	Sex                 string             `json:"sex"`
	Level               string             `json:"level"`
}

// NewProcessedPreferences creates empty processed preferences
func NewProcessedPreferences() *ProcessedPreferences {
	return &ProcessedPreferences{
		PriorityMusclesEN:  []string{},
		DislikedMusclesEN:  []string{},
		ExperienceTier:     "intermediate",
		VolumeModifier:     1.0,
		HealthRestrictions: HealthRestriction{},
	}
}

// HasDislikedMuscle checks if a muscle is in the disliked list
func (p *ProcessedPreferences) HasDislikedMuscle(muscle string) bool {
	for _, m := range p.DislikedMusclesEN {
		if m == muscle {
			return true
		}
	}
	return false
}

// HasPriorityMuscle checks if a muscle is in the priority list
func (p *ProcessedPreferences) HasPriorityMuscle(muscle string) bool {
	for _, m := range p.PriorityMusclesEN {
		if m == muscle {
			return true
		}
	}
	return false
}
```

---

## 2. Repository Interfaces

### 2.1 Plan Repository

**File:** `internal/domain/repository/plan_repository.go`

```go
package repository

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// PlanReader defines read operations for plans
type PlanReader interface {
	// GetMesocycleStatus retrieves the current mesocycle status for a user
	GetMesocycleStatus(ctx context.Context, userID string) (*entity.MesocycleStatus, error)
	// GetByUserID retrieves the active plan for a user
	GetByUserID(ctx context.Context, userID string) (*entity.Plan, error)
}

// PlanWriter defines write operations for plans
type PlanWriter interface {
	// IncrementMesocycle increments the mesocycle number and updates renewal date
	IncrementMesocycle(ctx context.Context, userID string) error
	// UpdateWeekSchedule updates the week_schedule for a user's plan
	UpdateWeekSchedule(ctx context.Context, userID, weekSchedule string) error
}

// PlanRepository combines all plan repository operations
type PlanRepository interface {
	PlanReader
	PlanWriter
}
```

### 2.2 Schedule Repository

**File:** `internal/domain/repository/schedule_repository.go`

```go
package repository

import (
	"context"
)

// ScheduleWriter defines write operations for user_weekly_schedule
type ScheduleWriter interface {
	// ClearSchedule deletes all schedule entries for a user
	ClearSchedule(ctx context.Context, userID string) error
	// ClearScheduleForWeeks deletes schedule entries for specific weeks
	ClearScheduleForWeeks(ctx context.Context, userID string, weeks []int) error
}

// ScheduleReader defines read operations for user_weekly_schedule
type ScheduleReader interface {
	// GetCompletedCountForWeek returns the count of completed sessions for a week
	GetCompletedCountForWeek(ctx context.Context, userID string, week int) (int, error)
}

// ScheduleRepository combines all schedule repository operations
type ScheduleRepository interface {
	ScheduleReader
	ScheduleWriter
}
```

### 2.3 Exercise Repository

**File:** `internal/domain/repository/exercise_repository.go`

```go
package repository

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// ExerciseReader defines read operations for the exercise catalog
type ExerciseReader interface {
	// GetByID retrieves an exercise by its ID
	GetByID(ctx context.Context, exerciseID string) (*entity.ExerciseCatalog, error)
	// FindAlternatives finds alternative exercises matching criteria
	FindAlternatives(ctx context.Context, pattern, role string, excludeIDs, excludeMuscles []string, limit int) ([]*entity.ExerciseCatalog, error)
	// FindByPatternAndRole finds exercises by pattern and role
	FindByPatternAndRole(ctx context.Context, pattern, role string) ([]*entity.ExerciseCatalog, error)
}

// ExerciseRepository combines all exercise catalog operations
type ExerciseRepository interface {
	ExerciseReader
}
```

### 2.4 Workout Renewal Repository

**File:** `internal/domain/repository/workout_renewal_repository.go`

```go
package repository

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// WorkoutRenewalReader defines read operations for workout renewal
type WorkoutRenewalReader interface {
	// GetUserWorkoutExercises retrieves all workout exercises for a user
	GetUserWorkoutExercises(ctx context.Context, userID string) ([]*entity.WorkoutExercise, error)
	// GetDistinctExerciseIDs returns distinct exercise IDs for a user's workouts
	GetDistinctExerciseIDs(ctx context.Context, userID string) ([]string, error)
}

// WorkoutRenewalWriter defines write operations for workout renewal
type WorkoutRenewalWriter interface {
	// DeleteUserWorkouts deletes all workouts for a user
	DeleteUserWorkouts(ctx context.Context, userID string) error
	// UpdateExerciseID updates the exercise_id for a specific workout
	UpdateExerciseID(ctx context.Context, workoutID, newExerciseID string) error
	// BatchUpdateExercises applies multiple exercise rotations
	BatchUpdateExercises(ctx context.Context, rotations []*entity.ExerciseRotation) error
	// ResetWeeksToOne updates all workouts to week=1 (for MANTENER flow)
	ResetWeeksToOne(ctx context.Context, userID string) error
}

// WorkoutRenewalRepository combines all workout renewal operations
type WorkoutRenewalRepository interface {
	WorkoutRenewalReader
	WorkoutRenewalWriter
}
```

### 2.5 User Profile Repository

**File:** `internal/domain/repository/user_profile_repository.go`

```go
package repository

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// UserProfileReader defines read operations for user gym profiles
type UserProfileReader interface {
	// GetByUserID retrieves the gym profile by user ID (via phone number matching)
	GetByUserID(ctx context.Context, userID string) (*entity.UserGymProfile, error)
	// GetByWhatsAppID retrieves the gym profile by WhatsApp ID
	GetByWhatsAppID(ctx context.Context, whatsappID string) (*entity.UserGymProfile, error)
}

// UserProfileWriter defines write operations for user gym profiles
type UserProfileWriter interface {
	// UpdatePreferences updates priority muscles, health status, and session duration
	UpdatePreferences(ctx context.Context, whatsappID string, priorityMuscles, healthStatus, sessionDuration string) error
}

// UserProfileRepository combines all user profile operations
type UserProfileRepository interface {
	UserProfileReader
	UserProfileWriter
}
```

---

## 3. Domain Services

### 3.1 Preference Processor Service

**File:** `internal/domain/service/preference_processor.go`

```go
package service

import (
	"strings"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// PreferenceProcessor processes user preferences from Spanish to English
type PreferenceProcessor struct {
	muscleMapping       map[string][]string
	healthRestrictions  map[string]entity.HealthRestriction
	experienceMapping   map[string]string
	durationVolumeMap   map[string]float64
}

// NewPreferenceProcessor creates a new PreferenceProcessor
func NewPreferenceProcessor() *PreferenceProcessor {
	return &PreferenceProcessor{
		muscleMapping: map[string][]string{
			// Spanish muscle names -> English database values
			"Glúteo":       {"Glutes"},
			"Glúteos":      {"Glutes"},
			"Pierna":       {"Quads", "Hamstrings"},
			"Piernas":      {"Quads", "Hamstrings"},
			"Cuádriceps":   {"Quads"},
			"Isquiotibiales": {"Hamstrings"},
			"Pantorrillas": {"Calfs"},
			"Pantorrilla":  {"Calfs"},
			"Pecho":        {"Chest"},
			"Espalda":      {"Back", "Lats", "Traps"},
			"Hombros":      {"Shoulders", "Front Delts", "Rear Delts"},
			"Hombro":       {"Shoulders", "Front Delts", "Rear Delts"},
			"Bíceps":       {"Biceps"},
			"Tríceps":      {"Triceps"},
			"Brazos":       {"Biceps", "Triceps", "Forearms"},
			"Abdomen":      {"Abs", "Core"},
			"Abdominales":  {"Abs", "Core"},
			"Core":         {"Abs", "Core"},
		},
		healthRestrictions: map[string]entity.HealthRestriction{
			"A": {}, // No restrictions
			"B": {AvoidLowerBodyImpact: true},
			"C": {AvoidUpperBodyOverhead: true},
			"D": {AvoidAxialLoading: true},
			"E": {PreferMachines: true, IsSpecialCondition: true},
		},
		experienceMapping: map[string]string{
			"Menos de 6 meses":  "beginner",
			"6 meses a 1 año":   "beginner",
			"1 a 2 años":        "intermediate",
			"2 a 3 años":        "intermediate",
			"Más de 3 años":     "advanced",
		},
		durationVolumeMap: map[string]float64{
			"30-45 min": 0.70,
			"45-60 min": 0.85,
			"60-75 min": 1.00,
			"75-90 min": 1.15,
			"90+ min":   1.30,
		},
	}
}

// ProcessProfile processes a user's gym profile into processed preferences
func (p *PreferenceProcessor) ProcessProfile(profile *entity.UserGymProfile) *entity.ProcessedPreferences {
	result := entity.NewProcessedPreferences()

	// Map priority muscles
	if profile.PriorityMuscles != "" {
		result.PriorityMusclesEN = p.mapMuscles(profile.PriorityMuscles)
	}

	// Map disliked exercises (which are muscle groups)
	if profile.DislikedExercises != "" {
		result.DislikedMusclesEN = p.mapMuscles(profile.DislikedExercises)
	}

	// Map experience tier
	if tier, exists := p.experienceMapping[profile.TrainingExperience]; exists {
		result.ExperienceTier = tier
	}

	// Map volume modifier based on session duration
	if modifier, exists := p.durationVolumeMap[profile.SessionDurationMins]; exists {
		result.VolumeModifier = modifier
	}

	// Map health restrictions
	if restriction, exists := p.healthRestrictions[profile.HealthStatus]; exists {
		result.HealthRestrictions = restriction
	}

	// Copy sex and level
	result.Sex = profile.Sex
	result.Level = profile.FitnessLevel

	return result
}

// mapMuscles converts a comma-separated Spanish muscle string to English muscle array
func (p *PreferenceProcessor) mapMuscles(spanishMuscles string) []string {
	muscles := strings.Split(spanishMuscles, ",")
	result := make([]string, 0)
	seen := make(map[string]bool)

	for _, muscle := range muscles {
		muscle = strings.TrimSpace(muscle)
		if englishMuscles, exists := p.muscleMapping[muscle]; exists {
			for _, em := range englishMuscles {
				if !seen[em] {
					result = append(result, em)
					seen[em] = true
				}
			}
		}
	}

	return result
}

// MapSingleMuscle maps a single Spanish muscle name to English
func (p *PreferenceProcessor) MapSingleMuscle(spanishMuscle string) []string {
	spanishMuscle = strings.TrimSpace(spanishMuscle)
	if englishMuscles, exists := p.muscleMapping[spanishMuscle]; exists {
		return englishMuscles
	}
	return []string{}
}

// GetHealthRestriction returns health restrictions for a status code
func (p *PreferenceProcessor) GetHealthRestriction(healthStatus string) entity.HealthRestriction {
	if restriction, exists := p.healthRestrictions[healthStatus]; exists {
		return restriction
	}
	return entity.HealthRestriction{}
}
```

### 3.2 Exercise Rotation Service

**File:** `internal/domain/service/exercise_rotation_service.go`

```go
package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
)

// ExerciseRotationService handles exercise rotation logic
type ExerciseRotationService struct {
	exerciseRepo repository.ExerciseReader
	rng          *rand.Rand
}

// NewExerciseRotationService creates a new ExerciseRotationService
func NewExerciseRotationService(exerciseRepo repository.ExerciseReader) *ExerciseRotationService {
	// Seed with current time for production randomness
	// For deterministic tests, use NewExerciseRotationServiceWithSeed
	return &ExerciseRotationService{
		exerciseRepo: exerciseRepo,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewExerciseRotationServiceWithSeed creates a service with a fixed seed for testing
func NewExerciseRotationServiceWithSeed(exerciseRepo repository.ExerciseReader, seed int64) *ExerciseRotationService {
	return &ExerciseRotationService{
		exerciseRepo: exerciseRepo,
		rng:          rand.New(rand.NewSource(seed)),
	}
}

// RotationOptions contains options for exercise rotation
type RotationOptions struct {
	ExcludeExerciseIDs []string
	DislikedMuscles    []string
	PriorityMuscles    []string
	HealthRestrictions entity.HealthRestriction
	MaxCandidates      int
}

// FindAlternative finds an alternative exercise for a given pattern and role
func (s *ExerciseRotationService) FindAlternative(
	ctx context.Context,
	pattern, role string,
	opts RotationOptions,
) (*entity.ExerciseCatalog, error) {
	if opts.MaxCandidates == 0 {
		opts.MaxCandidates = 10
	}

	// Find candidate exercises
	candidates, err := s.exerciseRepo.FindAlternatives(
		ctx,
		pattern,
		role,
		opts.ExcludeExerciseIDs,
		opts.DislikedMuscles,
		opts.MaxCandidates*2, // Fetch extra to allow filtering
	)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return nil, nil // No alternatives found
	}

	// Apply health restrictions
	filtered := s.applyHealthRestrictions(candidates, opts.HealthRestrictions)
	if len(filtered) == 0 {
		filtered = candidates // Fallback to unfiltered if too restrictive
	}

	// Score and rank candidates
	scored := s.scoreExercises(filtered, opts.PriorityMuscles)

	// Select from top candidates with weighted randomness
	return s.selectFromTopCandidates(scored, opts.MaxCandidates), nil
}

// RotateAllExercises rotates all exercises for a user
func (s *ExerciseRotationService) RotateAllExercises(
	ctx context.Context,
	workouts []*entity.WorkoutExercise,
	preferences *entity.ProcessedPreferences,
) ([]*entity.ExerciseRotation, error) {
	rotations := make([]*entity.ExerciseRotation, 0)
	usedExerciseIDs := make(map[string]bool)

	// Collect all current exercise IDs to exclude
	for _, w := range workouts {
		usedExerciseIDs[w.ExerciseID] = true
	}

	// Group workouts by pattern+role for consistent rotation
	patternRoleMap := s.groupByPatternRole(workouts)

	for key, workoutGroup := range patternRoleMap {
		parts := splitPatternRoleKey(key)
		pattern, role := parts[0], parts[1]

		// Find one alternative for this pattern+role
		opts := RotationOptions{
			ExcludeExerciseIDs: mapKeysToSlice(usedExerciseIDs),
			DislikedMuscles:    preferences.DislikedMusclesEN,
			PriorityMuscles:    preferences.PriorityMusclesEN,
			HealthRestrictions: preferences.HealthRestrictions,
			MaxCandidates:      5,
		}

		alternative, err := s.FindAlternative(ctx, pattern, role, opts)
		if err != nil {
			return nil, err
		}

		if alternative == nil {
			continue // Keep original if no alternative found
		}

		// Apply same alternative to all workouts with this pattern+role
		for _, w := range workoutGroup {
			rotation := entity.NewExerciseRotation(
				w.ID,
				w.ExerciseID,
				alternative.ExerciseID,
				pattern,
				role,
			)
			rotation.NewExercise = alternative
			rotations = append(rotations, rotation)
		}

		// Mark new exercise as used
		usedExerciseIDs[alternative.ExerciseID] = true
	}

	return rotations, nil
}

// applyHealthRestrictions filters exercises based on health restrictions
func (s *ExerciseRotationService) applyHealthRestrictions(
	exercises []*entity.ExerciseCatalog,
	restrictions entity.HealthRestriction,
) []*entity.ExerciseCatalog {
	if !restrictions.AvoidUpperBodyOverhead &&
	   !restrictions.AvoidLowerBodyImpact &&
	   !restrictions.AvoidAxialLoading &&
	   !restrictions.PreferMachines {
		return exercises // No filtering needed
	}

	filtered := make([]*entity.ExerciseCatalog, 0)
	for _, e := range exercises {
		// Skip overhead exercises for upper body issues
		if restrictions.AvoidUpperBodyOverhead && isOverheadExercise(e) {
			continue
		}
		// Skip high-impact lower body exercises
		if restrictions.AvoidLowerBodyImpact && isHighImpactLowerBody(e) {
			continue
		}
		// Skip heavy axial loading exercises
		if restrictions.AvoidAxialLoading && isAxialLoading(e) {
			continue
		}
		// Prefer machines for special conditions
		if restrictions.PreferMachines && e.Equipment != "Machine" && len(filtered) > 0 {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

// scoredExercise holds an exercise with its score
type scoredExercise struct {
	exercise *entity.ExerciseCatalog
	score    int
}

// scoreExercises scores exercises based on priority muscle matching
func (s *ExerciseRotationService) scoreExercises(
	exercises []*entity.ExerciseCatalog,
	priorityMuscles []string,
) []scoredExercise {
	scored := make([]scoredExercise, len(exercises))

	for i, e := range exercises {
		score := 0
		// +2 for main muscle match
		for _, pm := range priorityMuscles {
			if e.MainMuscle == pm {
				score += 2
			}
			// +1 for secondary muscle match
			for _, sm := range e.SecondaryMuscles {
				if sm == pm {
					score += 1
				}
			}
		}
		scored[i] = scoredExercise{exercise: e, score: score}
	}

	return scored
}

// selectFromTopCandidates selects an exercise from top candidates with weighted random
func (s *ExerciseRotationService) selectFromTopCandidates(
	scored []scoredExercise,
	maxCandidates int,
) *entity.ExerciseCatalog {
	if len(scored) == 0 {
		return nil
	}

	// Sort by score descending (simple bubble sort for small arrays)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Take top candidates
	if len(scored) > maxCandidates {
		scored = scored[:maxCandidates]
	}

	// Weighted random selection (higher score = higher probability)
	totalWeight := 0
	for _, se := range scored {
		totalWeight += se.score + 1 // +1 to give non-matching exercises a chance
	}

	target := s.rng.Intn(totalWeight)
	cumulative := 0
	for _, se := range scored {
		cumulative += se.score + 1
		if cumulative > target {
			return se.exercise
		}
	}

	return scored[0].exercise // Fallback to top scorer
}

// groupByPatternRole groups workouts by pattern+role
func (s *ExerciseRotationService) groupByPatternRole(
	workouts []*entity.WorkoutExercise,
) map[string][]*entity.WorkoutExercise {
	result := make(map[string][]*entity.WorkoutExercise)
	// This requires exercise catalog data - simplified version uses workout ID grouping
	// In practice, you'd join with exercises table to get pattern/role
	return result
}

// Helper functions

func isOverheadExercise(e *entity.ExerciseCatalog) bool {
	overheadPatterns := []string{"push_v", "shoulder_press"}
	for _, p := range overheadPatterns {
		if e.Pattern == p {
			return true
		}
	}
	return false
}

func isHighImpactLowerBody(e *entity.ExerciseCatalog) bool {
	highImpactMuscles := []string{"Quads", "Glutes", "Hamstrings"}
	for _, m := range highImpactMuscles {
		if e.MainMuscle == m && e.Equipment == "Barbell" {
			return true
		}
	}
	return false
}

func isAxialLoading(e *entity.ExerciseCatalog) bool {
	// Exercises that load the spine vertically
	axialPatterns := []string{"squat", "deadlift", "overhead_press"}
	for _, p := range axialPatterns {
		if e.Pattern == p {
			return true
		}
	}
	return false
}

func splitPatternRoleKey(key string) []string {
	// Key format: "pattern:role"
	for i, c := range key {
		if c == ':' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key, ""}
}

func mapKeysToSlice(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
```

---

## 4. Application Layer

### 4.1 Renewal DTOs

**File:** `internal/application/dto/renewal_dto.go`

```go
package dto

import "time"

// ==================== REQUEST DTOs ====================

// CheckMesocycleStatusRequest is the request for checking mesocycle status
// User ID comes from path parameter, so this is empty
type CheckMesocycleStatusRequest struct{}

// RenewMaintainRequest is the request for maintaining the same routine
type RenewMaintainRequest struct {
	Notes string `json:"notes,omitempty"`
}

// RenewRotateExercisesRequest is the request for rotating exercises
type RenewRotateExercisesRequest struct {
	RotateCompounds bool `json:"rotate_compounds"` // Default true
	RotateIsolation bool `json:"rotate_isolation"` // Default true
	RotateCore      bool `json:"rotate_core"`      // Default true
}

// RenewChangeDaysRequest is the request for changing training days
type RenewChangeDaysRequest struct {
	NewDaysPerWeek int    `json:"new_days_per_week" binding:"required,min=2,max=6"`
	Notes          string `json:"notes,omitempty"`
}

// UpdateProfileRequest is the request for updating user preferences
type UpdateProfileRequest struct {
	PriorityMuscles   *string `json:"priority_muscles,omitempty"`
	HealthStatus      *string `json:"health_status,omitempty"`
	SessionDuration   *string `json:"session_duration,omitempty"`
	Notes             string  `json:"notes,omitempty"`
}

// ==================== RESPONSE DTOs ====================

// MesocycleStatusResponse is the response for GET /plans/:userId/mesocycle-status
type MesocycleStatusResponse struct {
	UserID           string     `json:"user_id"`
	MesocycleNumber  int        `json:"mesocycle_number"`
	DaysPerWeek      int        `json:"days_per_week"`
	WeekSchedule     string     `json:"week_schedule"`
	Week4Completed   int        `json:"week4_completed"`
	Week4Total       int        `json:"week4_total"`
	IsComplete       bool       `json:"is_complete"`
	LastRenewalDate  *time.Time `json:"last_renewal_date,omitempty"`
	Goal             string     `json:"goal"`
	Level            string     `json:"level"`
	CanRenew         bool       `json:"can_renew"`
	Message          string     `json:"message"`
}

// RenewalResponse is the common response for all renewal operations
type RenewalResponse struct {
	Success            bool   `json:"success"`
	RenewalType        string `json:"renewal_type"`
	NewMesocycleNumber int    `json:"new_mesocycle_number"`
	Message            string `json:"message"`
	ScheduleCleared    bool   `json:"schedule_cleared"`
	WorkoutsUpdated    int    `json:"workouts_updated,omitempty"`
	ExercisesRotated   int    `json:"exercises_rotated,omitempty"`
	NewWeekSchedule    string `json:"new_week_schedule,omitempty"`
}

// ExerciseRotationDTO represents a single exercise rotation in the response
type ExerciseRotationDTO struct {
	WorkoutID       string `json:"workout_id"`
	OldExerciseID   string `json:"old_exercise_id"`
	OldExerciseName string `json:"old_exercise_name"`
	NewExerciseID   string `json:"new_exercise_id"`
	NewExerciseName string `json:"new_exercise_name"`
	Pattern         string `json:"pattern"`
	Role            string `json:"role"`
}

// RotateExercisesResponse extends RenewalResponse with rotation details
type RotateExercisesResponse struct {
	RenewalResponse
	Rotations []ExerciseRotationDTO `json:"rotations,omitempty"`
}

// ChangeDaysResponse extends RenewalResponse with schedule change details
type ChangeDaysResponse struct {
	RenewalResponse
	OldDaysPerWeek int    `json:"old_days_per_week"`
	NewDaysPerWeek int    `json:"new_days_per_week"`
	RequiresRegeneration bool `json:"requires_regeneration"`
}

// ==================== ERROR DTOs ====================

// RenewalErrorResponse represents an error response for renewal operations
type RenewalErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code"`
}
```

### 4.2 Check Mesocycle Status Use Case

**File:** `internal/application/usecase/check_mesocycle_status.go`

```go
package usecase

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// CheckMesocycleStatusUseCase handles checking if a user's mesocycle is complete
type CheckMesocycleStatusUseCase struct {
	planRepo repository.PlanReader
}

// NewCheckMesocycleStatusUseCase creates a new CheckMesocycleStatusUseCase
func NewCheckMesocycleStatusUseCase(planRepo repository.PlanReader) *CheckMesocycleStatusUseCase {
	return &CheckMesocycleStatusUseCase{
		planRepo: planRepo,
	}
}

// Execute checks the mesocycle status for a user
func (uc *CheckMesocycleStatusUseCase) Execute(ctx context.Context, userID string) (*dto.MesocycleStatusResponse, error) {
	if userID == "" {
		return nil, apperror.NewValidationError("user_id is required")
	}

	status, err := uc.planRepo.GetMesocycleStatus(ctx, userID)
	if err != nil {
		return nil, err
	}

	if status == nil {
		return nil, apperror.NewNotFoundError("no active plan found for user")
	}

	// Determine message based on completion status
	var message string
	if status.IsComplete {
		message = "Mesociclo completado. Puedes renovar tu rutina."
	} else {
		remaining := status.Week4Total - status.Week4Completed
		message = buildIncompleteMessage(remaining)
	}

	return &dto.MesocycleStatusResponse{
		UserID:          status.UserID,
		MesocycleNumber: status.MesocycleNumber,
		DaysPerWeek:     status.DaysPerWeek,
		WeekSchedule:    status.WeekSchedule,
		Week4Completed:  status.Week4Completed,
		Week4Total:      status.Week4Total,
		IsComplete:      status.IsComplete,
		LastRenewalDate: status.LastRenewalDate,
		Goal:            status.Goal,
		Level:           status.Level,
		CanRenew:        status.IsComplete,
		Message:         message,
	}, nil
}

func buildIncompleteMessage(remaining int) string {
	if remaining == 1 {
		return "Te falta 1 sesion para completar el mesociclo."
	}
	return "Te faltan " + itoa(remaining) + " sesiones para completar el mesociclo."
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
```

### 4.3 Renew Maintain Use Case

**File:** `internal/application/usecase/renew_maintain.go`

```go
package usecase

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// RenewMaintainUseCase handles renewing a mesocycle while keeping the same routine
type RenewMaintainUseCase struct {
	planRepo           repository.PlanRepository
	scheduleRepo       repository.ScheduleWriter
	workoutRenewalRepo repository.WorkoutRenewalWriter
}

// NewRenewMaintainUseCase creates a new RenewMaintainUseCase
func NewRenewMaintainUseCase(
	planRepo repository.PlanRepository,
	scheduleRepo repository.ScheduleWriter,
	workoutRenewalRepo repository.WorkoutRenewalWriter,
) *RenewMaintainUseCase {
	return &RenewMaintainUseCase{
		planRepo:           planRepo,
		scheduleRepo:       scheduleRepo,
		workoutRenewalRepo: workoutRenewalRepo,
	}
}

// Execute renews the mesocycle while maintaining the same exercises
func (uc *RenewMaintainUseCase) Execute(ctx context.Context, userID string, req *dto.RenewMaintainRequest) (*dto.RenewalResponse, error) {
	if userID == "" {
		return nil, apperror.NewValidationError("user_id is required")
	}

	// 1. Verify mesocycle is complete
	status, err := uc.planRepo.GetMesocycleStatus(ctx, userID)
	if err != nil {
		return nil, err
	}

	if status == nil {
		return nil, apperror.NewNotFoundError("no active plan found for user")
	}

	if !status.IsComplete {
		return nil, apperror.NewValidationError("mesocycle is not yet complete")
	}

	// 2. Clear the user's schedule
	if err := uc.scheduleRepo.ClearSchedule(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to clear schedule", err)
	}

	// 3. Reset workout weeks to 1 (keeps same exercises, restarts progression)
	if err := uc.workoutRenewalRepo.ResetWeeksToOne(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to reset workout weeks", err)
	}

	// 4. Increment mesocycle number
	if err := uc.planRepo.IncrementMesocycle(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to increment mesocycle", err)
	}

	return &dto.RenewalResponse{
		Success:            true,
		RenewalType:        "MANTENER_RUTINA",
		NewMesocycleNumber: status.MesocycleNumber + 1,
		Message:            "Tu rutina se ha renovado. Continuas con los mismos ejercicios con progresion de carga.",
		ScheduleCleared:    true,
	}, nil
}
```

### 4.4 Renew Rotate Exercises Use Case

**File:** `internal/application/usecase/renew_rotate_exercises.go`

```go
package usecase

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/internal/domain/service"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// RenewRotateExercisesUseCase handles renewing with new exercises
type RenewRotateExercisesUseCase struct {
	planRepo           repository.PlanRepository
	scheduleRepo       repository.ScheduleWriter
	workoutRenewalRepo repository.WorkoutRenewalRepository
	profileRepo        repository.UserProfileReader
	exerciseRotation   *service.ExerciseRotationService
	preferenceProc     *service.PreferenceProcessor
}

// NewRenewRotateExercisesUseCase creates a new RenewRotateExercisesUseCase
func NewRenewRotateExercisesUseCase(
	planRepo repository.PlanRepository,
	scheduleRepo repository.ScheduleWriter,
	workoutRenewalRepo repository.WorkoutRenewalRepository,
	profileRepo repository.UserProfileReader,
	exerciseRotation *service.ExerciseRotationService,
	preferenceProc *service.PreferenceProcessor,
) *RenewRotateExercisesUseCase {
	return &RenewRotateExercisesUseCase{
		planRepo:           planRepo,
		scheduleRepo:       scheduleRepo,
		workoutRenewalRepo: workoutRenewalRepo,
		profileRepo:        profileRepo,
		exerciseRotation:   exerciseRotation,
		preferenceProc:     preferenceProc,
	}
}

// Execute renews the mesocycle with rotated exercises
func (uc *RenewRotateExercisesUseCase) Execute(
	ctx context.Context,
	userID string,
	req *dto.RenewRotateExercisesRequest,
) (*dto.RotateExercisesResponse, error) {
	if userID == "" {
		return nil, apperror.NewValidationError("user_id is required")
	}

	// Default: rotate all exercise types
	if req == nil {
		req = &dto.RenewRotateExercisesRequest{
			RotateCompounds: true,
			RotateIsolation: true,
			RotateCore:      true,
		}
	}

	// 1. Verify mesocycle is complete
	status, err := uc.planRepo.GetMesocycleStatus(ctx, userID)
	if err != nil {
		return nil, err
	}

	if status == nil {
		return nil, apperror.NewNotFoundError("no active plan found for user")
	}

	if !status.IsComplete {
		return nil, apperror.NewValidationError("mesocycle is not yet complete")
	}

	// 2. Get user profile for preferences
	profile, err := uc.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.NewInternalError("failed to get user profile", err)
	}

	var preferences *entity.ProcessedPreferences
	if profile != nil {
		preferences = uc.preferenceProc.ProcessProfile(profile)
	} else {
		preferences = entity.NewProcessedPreferences()
	}

	// 3. Get current workouts
	workouts, err := uc.workoutRenewalRepo.GetUserWorkoutExercises(ctx, userID)
	if err != nil {
		return nil, apperror.NewInternalError("failed to get workouts", err)
	}

	// 4. Filter workouts by rotation options
	workoutsToRotate := filterWorkoutsByRole(workouts, req)

	// 5. Rotate exercises
	rotations, err := uc.exerciseRotation.RotateAllExercises(ctx, workoutsToRotate, preferences)
	if err != nil {
		return nil, apperror.NewInternalError("failed to rotate exercises", err)
	}

	// 6. Apply rotations to database
	if len(rotations) > 0 {
		if err := uc.workoutRenewalRepo.BatchUpdateExercises(ctx, rotations); err != nil {
			return nil, apperror.NewInternalError("failed to apply exercise rotations", err)
		}
	}

	// 7. Clear schedule
	if err := uc.scheduleRepo.ClearSchedule(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to clear schedule", err)
	}

	// 8. Reset workout weeks to 1
	if err := uc.workoutRenewalRepo.ResetWeeksToOne(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to reset workout weeks", err)
	}

	// 9. Increment mesocycle
	if err := uc.planRepo.IncrementMesocycle(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to increment mesocycle", err)
	}

	// Build response
	rotationDTOs := make([]dto.ExerciseRotationDTO, len(rotations))
	for i, r := range rotations {
		rotationDTOs[i] = dto.ExerciseRotationDTO{
			WorkoutID:       r.WorkoutID,
			OldExerciseID:   r.OldExerciseID,
			NewExerciseID:   r.NewExerciseID,
			Pattern:         r.Pattern,
			Role:            r.Role,
		}
		if r.OldExercise != nil {
			rotationDTOs[i].OldExerciseName = r.OldExercise.SpanishName
		}
		if r.NewExercise != nil {
			rotationDTOs[i].NewExerciseName = r.NewExercise.SpanishName
		}
	}

	return &dto.RotateExercisesResponse{
		RenewalResponse: dto.RenewalResponse{
			Success:            true,
			RenewalType:        "ROTAR_EJERCICIOS",
			NewMesocycleNumber: status.MesocycleNumber + 1,
			Message:            "Tus ejercicios han sido actualizados para el nuevo mesociclo.",
			ScheduleCleared:    true,
			ExercisesRotated:   len(rotations),
		},
		Rotations: rotationDTOs,
	}, nil
}

// Need to import entity for ProcessedPreferences
import "github.com/gymbot/workout-tracker-back/internal/domain/entity"

func filterWorkoutsByRole(
	workouts []*entity.WorkoutExercise,
	req *dto.RenewRotateExercisesRequest,
) []*entity.WorkoutExercise {
	// In practice, this requires joining with exercises table to get role
	// For now, return all workouts and let rotation service handle filtering
	return workouts
}
```

### 4.5 Renew Change Days Use Case

**File:** `internal/application/usecase/renew_change_days.go`

```go
package usecase

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// WeekScheduleMapping maps days per week to schedule type
var WeekScheduleMapping = map[int]string{
	2: "fb_2",
	3: "fb_3",
	4: "ul_4", // Note: database has ul_4, not ua_4
	5: "ppl_5",
	6: "ppl_6",
}

// RenewChangeDaysUseCase handles changing training frequency
type RenewChangeDaysUseCase struct {
	planRepo           repository.PlanRepository
	scheduleRepo       repository.ScheduleWriter
	workoutRenewalRepo repository.WorkoutRenewalWriter
}

// NewRenewChangeDaysUseCase creates a new RenewChangeDaysUseCase
func NewRenewChangeDaysUseCase(
	planRepo repository.PlanRepository,
	scheduleRepo repository.ScheduleWriter,
	workoutRenewalRepo repository.WorkoutRenewalWriter,
) *RenewChangeDaysUseCase {
	return &RenewChangeDaysUseCase{
		planRepo:           planRepo,
		scheduleRepo:       scheduleRepo,
		workoutRenewalRepo: workoutRenewalRepo,
	}
}

// Execute changes the training frequency for the next mesocycle
func (uc *RenewChangeDaysUseCase) Execute(
	ctx context.Context,
	userID string,
	req *dto.RenewChangeDaysRequest,
) (*dto.ChangeDaysResponse, error) {
	if userID == "" {
		return nil, apperror.NewValidationError("user_id is required")
	}

	if req == nil || req.NewDaysPerWeek < 2 || req.NewDaysPerWeek > 6 {
		return nil, apperror.NewValidationError("new_days_per_week must be between 2 and 6")
	}

	// 1. Verify mesocycle is complete
	status, err := uc.planRepo.GetMesocycleStatus(ctx, userID)
	if err != nil {
		return nil, err
	}

	if status == nil {
		return nil, apperror.NewNotFoundError("no active plan found for user")
	}

	if !status.IsComplete {
		return nil, apperror.NewValidationError("mesocycle is not yet complete")
	}

	// 2. Get new week schedule type
	newWeekSchedule, exists := WeekScheduleMapping[req.NewDaysPerWeek]
	if !exists {
		return nil, apperror.NewValidationError("invalid days per week")
	}

	oldDaysPerWeek := status.DaysPerWeek
	requiresRegeneration := oldDaysPerWeek != req.NewDaysPerWeek

	// 3. Clear schedule
	if err := uc.scheduleRepo.ClearSchedule(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to clear schedule", err)
	}

	// 4. Delete workouts (will be regenerated by GymRatForm)
	if requiresRegeneration {
		if err := uc.workoutRenewalRepo.DeleteUserWorkouts(ctx, userID); err != nil {
			return nil, apperror.NewInternalError("failed to delete workouts", err)
		}
	}

	// 5. Update week schedule in plan
	if err := uc.planRepo.UpdateWeekSchedule(ctx, userID, newWeekSchedule); err != nil {
		return nil, apperror.NewInternalError("failed to update week schedule", err)
	}

	// 6. Increment mesocycle
	if err := uc.planRepo.IncrementMesocycle(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to increment mesocycle", err)
	}

	message := "Tu frecuencia de entrenamiento ha sido actualizada."
	if requiresRegeneration {
		message += " Se generara una nueva rutina con " + itoa(req.NewDaysPerWeek) + " dias."
	}

	return &dto.ChangeDaysResponse{
		RenewalResponse: dto.RenewalResponse{
			Success:            true,
			RenewalType:        "CAMBIAR_DIAS",
			NewMesocycleNumber: status.MesocycleNumber + 1,
			Message:            message,
			ScheduleCleared:    true,
			NewWeekSchedule:    newWeekSchedule,
		},
		OldDaysPerWeek:       oldDaysPerWeek,
		NewDaysPerWeek:       req.NewDaysPerWeek,
		RequiresRegeneration: requiresRegeneration,
	}, nil
}
```

---

## 5. Adapter Layer

### 5.1 Plan Handler

**File:** `internal/adapter/http/handler/plan_handler.go`

```go
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/application/usecase"
	"github.com/gymbot/workout-tracker-back/pkg/response"
)

// PlanHandler handles plan-related HTTP requests
type PlanHandler struct {
	checkStatus      *usecase.CheckMesocycleStatusUseCase
	renewMaintain    *usecase.RenewMaintainUseCase
	renewRotate      *usecase.RenewRotateExercisesUseCase
	renewChangeDays  *usecase.RenewChangeDaysUseCase
}

// NewPlanHandler creates a new PlanHandler
func NewPlanHandler(
	checkStatus *usecase.CheckMesocycleStatusUseCase,
	renewMaintain *usecase.RenewMaintainUseCase,
	renewRotate *usecase.RenewRotateExercisesUseCase,
	renewChangeDays *usecase.RenewChangeDaysUseCase,
) *PlanHandler {
	return &PlanHandler{
		checkStatus:     checkStatus,
		renewMaintain:   renewMaintain,
		renewRotate:     renewRotate,
		renewChangeDays: renewChangeDays,
	}
}

// GetMesocycleStatus handles GET /api/v1/plans/:userId/mesocycle-status
func (h *PlanHandler) GetMesocycleStatus(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		response.BadRequest(c, "userId path parameter is required")
		return
	}

	result, err := h.checkStatus.Execute(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// RenewMaintain handles POST /api/v1/plans/:userId/renew/maintain
func (h *PlanHandler) RenewMaintain(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		response.BadRequest(c, "userId path parameter is required")
		return
	}

	var req dto.RenewMaintainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Request body is optional
		req = dto.RenewMaintainRequest{}
	}

	result, err := h.renewMaintain.Execute(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// RenewRotateExercises handles POST /api/v1/plans/:userId/renew/rotate-exercises
func (h *PlanHandler) RenewRotateExercises(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		response.BadRequest(c, "userId path parameter is required")
		return
	}

	var req dto.RenewRotateExercisesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Default: rotate all
		req = dto.RenewRotateExercisesRequest{
			RotateCompounds: true,
			RotateIsolation: true,
			RotateCore:      true,
		}
	}

	result, err := h.renewRotate.Execute(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// RenewChangeDays handles POST /api/v1/plans/:userId/renew/change-days
func (h *PlanHandler) RenewChangeDays(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		response.BadRequest(c, "userId path parameter is required")
		return
	}

	var req dto.RenewChangeDaysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	result, err := h.renewChangeDays.Execute(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}
```

### 5.2 Internal API Key Middleware

**File:** `internal/adapter/http/middleware/internal_api_key.go`

```go
package middleware

import (
	"github.com/gin-gonic/gin"
)

// ValidateInternalAPIKey returns a middleware that validates the internal API key
// This is used for n8n -> Backend communication
func ValidateInternalAPIKey(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check X-API-Key header
		providedKey := c.GetHeader("X-API-Key")
		if providedKey == "" {
			// Fallback to Authorization header
			providedKey = c.GetHeader("Authorization")
			// Strip "Bearer " prefix if present
			if len(providedKey) > 7 && providedKey[:7] == "Bearer " {
				providedKey = providedKey[7:]
			}
		}

		if providedKey == "" {
			c.AbortWithStatusJSON(401, gin.H{
				"success": false,
				"error": gin.H{
					"code":    401,
					"message": "API key required",
				},
			})
			return
		}

		if providedKey != apiKey {
			c.AbortWithStatusJSON(403, gin.H{
				"success": false,
				"error": gin.H{
					"code":    403,
					"message": "invalid API key",
				},
			})
			return
		}

		c.Next()
	}
}
```

### 5.3 Plan Repository (PostgreSQL)

**File:** `internal/adapter/repository/postgres/plan_repository.go`

```go
package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// PlanRepository implements repository.PlanRepository using PostgreSQL
type PlanRepository struct {
	conn *Connection
}

// Ensure PlanRepository implements the interface
var _ repository.PlanRepository = (*PlanRepository)(nil)

// NewPlanRepository creates a new PlanRepository
func NewPlanRepository(conn *Connection) *PlanRepository {
	return &PlanRepository{conn: conn}
}

// GetMesocycleStatus retrieves the current mesocycle status for a user
func (r *PlanRepository) GetMesocycleStatus(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
	query := `
		WITH week4_sessions AS (
			SELECT
				COUNT(*) FILTER (WHERE "Completed" = true) as completed,
				COUNT(*) as total
			FROM user_weekly_schedule
			WHERE user_id = $1 AND week = 4
		),
		plan_info AS (
			SELECT
				up.plan_id,
				up.user_id,
				up.mesocycle_number,
				up.last_renewal_date,
				up.goal,
				up.level,
				up.week_schedule,
				ws.days_per_week
			FROM users_plans up
			JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
			WHERE up.user_id = $1 AND up.status = 'active'
			LIMIT 1
		)
		SELECT
			pi.user_id,
			pi.mesocycle_number,
			pi.days_per_week,
			pi.week_schedule,
			COALESCE(w4.completed, 0) as week4_completed,
			pi.days_per_week as week4_total,
			COALESCE(w4.completed, 0) >= pi.days_per_week as is_complete,
			pi.last_renewal_date,
			pi.goal,
			pi.level
		FROM plan_info pi
		LEFT JOIN week4_sessions w4 ON true
	`

	var status entity.MesocycleStatus
	var lastRenewalDate sql.NullTime
	var goal, level sql.NullString

	err := r.conn.DB.QueryRowContext(ctx, query, userID).Scan(
		&status.UserID,
		&status.MesocycleNumber,
		&status.DaysPerWeek,
		&status.WeekSchedule,
		&status.Week4Completed,
		&status.Week4Total,
		&status.IsComplete,
		&lastRenewalDate,
		&goal,
		&level,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query mesocycle status", err)
	}

	if lastRenewalDate.Valid {
		status.LastRenewalDate = &lastRenewalDate.Time
	}
	status.Goal = goal.String
	status.Level = level.String

	return &status, nil
}

// GetByUserID retrieves the active plan for a user
func (r *PlanRepository) GetByUserID(ctx context.Context, userID string) (*entity.Plan, error) {
	query := `
		SELECT
			plan_id,
			user_id,
			template_id,
			week_schedule,
			goal,
			level,
			status,
			mesocycle_number,
			last_renewal_date,
			created_at
		FROM users_plans
		WHERE user_id = $1 AND status = 'active'
		LIMIT 1
	`

	var plan entity.Plan
	var templateID, goal, level sql.NullString
	var lastRenewalDate sql.NullTime
	var createdAt sql.NullTime

	err := r.conn.DB.QueryRowContext(ctx, query, userID).Scan(
		&plan.PlanID,
		&plan.UserID,
		&templateID,
		&plan.WeekSchedule,
		&goal,
		&level,
		&plan.Status,
		&plan.MesocycleNumber,
		&lastRenewalDate,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query plan", err)
	}

	plan.TemplateID = templateID.String
	plan.Goal = goal.String
	plan.Level = level.String
	if lastRenewalDate.Valid {
		plan.LastRenewalDate = &lastRenewalDate.Time
	}
	if createdAt.Valid {
		plan.CreatedAt = createdAt.Time
	}

	return &plan, nil
}

// IncrementMesocycle increments the mesocycle number and updates renewal date
func (r *PlanRepository) IncrementMesocycle(ctx context.Context, userID string) error {
	query := `
		UPDATE users_plans
		SET
			mesocycle_number = mesocycle_number + 1,
			last_renewal_date = $2
		WHERE user_id = $1 AND status = 'active'
	`

	result, err := r.conn.DB.ExecContext(ctx, query, userID, time.Now())
	if err != nil {
		return apperror.NewInternalError("failed to increment mesocycle", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return apperror.NewNotFoundError("no active plan found for user")
	}

	return nil
}

// UpdateWeekSchedule updates the week_schedule for a user's plan
func (r *PlanRepository) UpdateWeekSchedule(ctx context.Context, userID, weekSchedule string) error {
	query := `
		UPDATE users_plans
		SET week_schedule = $2
		WHERE user_id = $1 AND status = 'active'
	`

	result, err := r.conn.DB.ExecContext(ctx, query, userID, weekSchedule)
	if err != nil {
		return apperror.NewInternalError("failed to update week schedule", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return apperror.NewNotFoundError("no active plan found for user")
	}

	return nil
}
```

### 5.4 Schedule Repository (PostgreSQL)

**File:** `internal/adapter/repository/postgres/schedule_repository.go`

```go
package postgres

import (
	"context"
	"database/sql"

	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// ScheduleRepository implements repository.ScheduleRepository using PostgreSQL
type ScheduleRepository struct {
	conn *Connection
}

// Ensure ScheduleRepository implements the interface
var _ repository.ScheduleRepository = (*ScheduleRepository)(nil)

// NewScheduleRepository creates a new ScheduleRepository
func NewScheduleRepository(conn *Connection) *ScheduleRepository {
	return &ScheduleRepository{conn: conn}
}

// ClearSchedule deletes all schedule entries for a user
func (r *ScheduleRepository) ClearSchedule(ctx context.Context, userID string) error {
	query := `
		DELETE FROM user_weekly_schedule
		WHERE user_id = $1
	`

	_, err := r.conn.DB.ExecContext(ctx, query, userID)
	if err != nil {
		return apperror.NewInternalError("failed to clear schedule", err)
	}

	return nil
}

// ClearScheduleForWeeks deletes schedule entries for specific weeks
func (r *ScheduleRepository) ClearScheduleForWeeks(ctx context.Context, userID string, weeks []int) error {
	if len(weeks) == 0 {
		return nil
	}

	// Build query with parameterized week list
	query := `
		DELETE FROM user_weekly_schedule
		WHERE user_id = $1 AND week = ANY($2)
	`

	_, err := r.conn.DB.ExecContext(ctx, query, userID, weeks)
	if err != nil {
		return apperror.NewInternalError("failed to clear schedule for weeks", err)
	}

	return nil
}

// GetCompletedCountForWeek returns the count of completed sessions for a week
func (r *ScheduleRepository) GetCompletedCountForWeek(ctx context.Context, userID string, week int) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM user_weekly_schedule
		WHERE user_id = $1 AND week = $2 AND "Completed" = true
	`

	var count int
	err := r.conn.DB.QueryRowContext(ctx, query, userID, week).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, apperror.NewInternalError("failed to get completed count", err)
	}

	return count, nil
}
```

### 5.5 Exercise Repository (PostgreSQL)

**File:** `internal/adapter/repository/postgres/exercise_repository.go`

```go
package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
	"github.com/lib/pq"
)

// ExerciseRepository implements repository.ExerciseRepository using PostgreSQL
type ExerciseRepository struct {
	conn *Connection
}

// Ensure ExerciseRepository implements the interface
var _ repository.ExerciseRepository = (*ExerciseRepository)(nil)

// NewExerciseRepository creates a new ExerciseRepository
func NewExerciseRepository(conn *Connection) *ExerciseRepository {
	return &ExerciseRepository{conn: conn}
}

// GetByID retrieves an exercise by its ID
func (r *ExerciseRepository) GetByID(ctx context.Context, exerciseID string) (*entity.ExerciseCatalog, error) {
	query := `
		SELECT
			exercise_id,
			spanish_name,
			pattern,
			role,
			main_muscle,
			secondary_muscles,
			level,
			link,
			equipment
		FROM exercises
		WHERE exercise_id = $1
	`

	var exercise entity.ExerciseCatalog
	var secondaryMuscles pq.StringArray
	var link, equipment sql.NullString

	err := r.conn.DB.QueryRowContext(ctx, query, exerciseID).Scan(
		&exercise.ExerciseID,
		&exercise.SpanishName,
		&exercise.Pattern,
		&exercise.Role,
		&exercise.MainMuscle,
		&secondaryMuscles,
		&exercise.Level,
		&link,
		&equipment,
	)

	if err == sql.ErrNoRows {
		return nil, apperror.NewNotFoundError("exercise not found")
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query exercise", err)
	}

	exercise.SecondaryMuscles = []string(secondaryMuscles)
	exercise.Link = link.String
	exercise.Equipment = equipment.String

	return &exercise, nil
}

// FindAlternatives finds alternative exercises matching criteria
func (r *ExerciseRepository) FindAlternatives(
	ctx context.Context,
	pattern, role string,
	excludeIDs, excludeMuscles []string,
	limit int,
) ([]*entity.ExerciseCatalog, error) {
	// Build dynamic query
	query := `
		SELECT
			exercise_id,
			spanish_name,
			pattern,
			role,
			main_muscle,
			secondary_muscles,
			level,
			link,
			equipment
		FROM exercises
		WHERE pattern = $1 AND role = $2
	`
	args := []interface{}{pattern, role}
	argIndex := 3

	// Exclude specific exercise IDs
	if len(excludeIDs) > 0 {
		query += " AND exercise_id != ALL($" + itoa(argIndex) + ")"
		args = append(args, pq.StringArray(excludeIDs))
		argIndex++
	}

	// Exclude disliked muscles
	if len(excludeMuscles) > 0 {
		query += " AND main_muscle != ALL($" + itoa(argIndex) + ")"
		args = append(args, pq.StringArray(excludeMuscles))
		argIndex++
	}

	query += " ORDER BY RANDOM() LIMIT $" + itoa(argIndex)
	args = append(args, limit)

	rows, err := r.conn.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperror.NewInternalError("failed to find alternatives", err)
	}
	defer rows.Close()

	exercises := make([]*entity.ExerciseCatalog, 0)
	for rows.Next() {
		var exercise entity.ExerciseCatalog
		var secondaryMuscles pq.StringArray
		var link, equipment sql.NullString

		if err := rows.Scan(
			&exercise.ExerciseID,
			&exercise.SpanishName,
			&exercise.Pattern,
			&exercise.Role,
			&exercise.MainMuscle,
			&secondaryMuscles,
			&exercise.Level,
			&link,
			&equipment,
		); err != nil {
			return nil, apperror.NewInternalError("failed to scan exercise", err)
		}

		exercise.SecondaryMuscles = []string(secondaryMuscles)
		exercise.Link = link.String
		exercise.Equipment = equipment.String
		exercises = append(exercises, &exercise)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.NewInternalError("error iterating exercises", err)
	}

	return exercises, nil
}

// FindByPatternAndRole finds exercises by pattern and role
func (r *ExerciseRepository) FindByPatternAndRole(ctx context.Context, pattern, role string) ([]*entity.ExerciseCatalog, error) {
	query := `
		SELECT
			exercise_id,
			spanish_name,
			pattern,
			role,
			main_muscle,
			secondary_muscles,
			level,
			link,
			equipment
		FROM exercises
		WHERE pattern = $1 AND role = $2
		ORDER BY spanish_name
	`

	rows, err := r.conn.DB.QueryContext(ctx, query, pattern, role)
	if err != nil {
		return nil, apperror.NewInternalError("failed to find exercises", err)
	}
	defer rows.Close()

	exercises := make([]*entity.ExerciseCatalog, 0)
	for rows.Next() {
		var exercise entity.ExerciseCatalog
		var secondaryMuscles pq.StringArray
		var link, equipment sql.NullString

		if err := rows.Scan(
			&exercise.ExerciseID,
			&exercise.SpanishName,
			&exercise.Pattern,
			&exercise.Role,
			&exercise.MainMuscle,
			&secondaryMuscles,
			&exercise.Level,
			&link,
			&equipment,
		); err != nil {
			return nil, apperror.NewInternalError("failed to scan exercise", err)
		}

		exercise.SecondaryMuscles = []string(secondaryMuscles)
		exercise.Link = link.String
		exercise.Equipment = equipment.String
		exercises = append(exercises, &exercise)
	}

	return exercises, nil
}

func itoa(n int) string {
	return strings.TrimLeft(strings.Replace(string(rune('0'+n)), "\x00", "", -1), "0")
}
```

### 5.6 Workout Renewal Repository (PostgreSQL)

**File:** `internal/adapter/repository/postgres/workout_renewal_repository.go`

```go
package postgres

import (
	"context"
	"database/sql"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
	"github.com/lib/pq"
)

// WorkoutRenewalRepository implements repository.WorkoutRenewalRepository
type WorkoutRenewalRepository struct {
	conn *Connection
}

// Ensure WorkoutRenewalRepository implements the interface
var _ repository.WorkoutRenewalRepository = (*WorkoutRenewalRepository)(nil)

// NewWorkoutRenewalRepository creates a new WorkoutRenewalRepository
func NewWorkoutRenewalRepository(conn *Connection) *WorkoutRenewalRepository {
	return &WorkoutRenewalRepository{conn: conn}
}

// GetUserWorkoutExercises retrieves all workout exercises for a user
func (r *WorkoutRenewalRepository) GetUserWorkoutExercises(ctx context.Context, userID string) ([]*entity.WorkoutExercise, error) {
	query := `
		SELECT
			w.id,
			w.user_id,
			w.week,
			w.day_name,
			w.exercise_id,
			w.sets,
			w.reps,
			w.rir,
			w."rest-seconds",
			w.tempo,
			w.exercise_order,
			e.pattern,
			e.role
		FROM workouts w
		JOIN exercises e ON w.exercise_id = e.exercise_id
		WHERE w.user_id = $1
		ORDER BY w.week, w.exercise_order
	`

	rows, err := r.conn.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, apperror.NewInternalError("failed to query workouts", err)
	}
	defer rows.Close()

	workouts := make([]*entity.WorkoutExercise, 0)
	for rows.Next() {
		var w entity.WorkoutExercise
		var rir sql.NullString
		var restSeconds sql.NullInt64
		var tempo sql.NullString
		var pattern, role string

		if err := rows.Scan(
			&w.ID,
			&w.UserID,
			&w.Week,
			&w.DayName,
			&w.ExerciseID,
			&w.Sets,
			&w.Reps,
			&rir,
			&restSeconds,
			&tempo,
			&w.ExerciseOrder,
			&pattern,
			&role,
		); err != nil {
			return nil, apperror.NewInternalError("failed to scan workout", err)
		}

		w.RIR = rir.String
		w.RestSeconds = int(restSeconds.Int64)
		w.Tempo = tempo.String
		workouts = append(workouts, &w)
	}

	return workouts, nil
}

// GetDistinctExerciseIDs returns distinct exercise IDs for a user's workouts
func (r *WorkoutRenewalRepository) GetDistinctExerciseIDs(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT DISTINCT exercise_id
		FROM workouts
		WHERE user_id = $1
	`

	rows, err := r.conn.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, apperror.NewInternalError("failed to query distinct exercises", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, apperror.NewInternalError("failed to scan exercise id", err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// DeleteUserWorkouts deletes all workouts for a user
func (r *WorkoutRenewalRepository) DeleteUserWorkouts(ctx context.Context, userID string) error {
	query := `
		DELETE FROM workouts
		WHERE user_id = $1
	`

	_, err := r.conn.DB.ExecContext(ctx, query, userID)
	if err != nil {
		return apperror.NewInternalError("failed to delete workouts", err)
	}

	return nil
}

// UpdateExerciseID updates the exercise_id for a specific workout
func (r *WorkoutRenewalRepository) UpdateExerciseID(ctx context.Context, workoutID, newExerciseID string) error {
	query := `
		UPDATE workouts
		SET exercise_id = $2
		WHERE id = $1
	`

	result, err := r.conn.DB.ExecContext(ctx, query, workoutID, newExerciseID)
	if err != nil {
		return apperror.NewInternalError("failed to update exercise", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return apperror.NewNotFoundError("workout not found")
	}

	return nil
}

// BatchUpdateExercises applies multiple exercise rotations in a transaction
func (r *WorkoutRenewalRepository) BatchUpdateExercises(ctx context.Context, rotations []*entity.ExerciseRotation) error {
	if len(rotations) == 0 {
		return nil
	}

	tx, err := r.conn.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.NewInternalError("failed to begin transaction", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE workouts
		SET exercise_id = $2
		WHERE id = $1
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return apperror.NewInternalError("failed to prepare statement", err)
	}
	defer stmt.Close()

	for _, rotation := range rotations {
		_, err := stmt.ExecContext(ctx, rotation.WorkoutID, rotation.NewExerciseID)
		if err != nil {
			return apperror.NewInternalError("failed to update exercise rotation", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return apperror.NewInternalError("failed to commit transaction", err)
	}

	return nil
}

// ResetWeeksToOne updates all workouts to week=1 for MANTENER flow
func (r *WorkoutRenewalRepository) ResetWeeksToOne(ctx context.Context, userID string) error {
	query := `
		UPDATE workouts
		SET week = 1
		WHERE user_id = $1
	`

	_, err := r.conn.DB.ExecContext(ctx, query, userID)
	if err != nil {
		return apperror.NewInternalError("failed to reset weeks", err)
	}

	return nil
}
```

### 5.7 User Profile Repository (PostgreSQL)

**File:** `internal/adapter/repository/postgres/user_profile_repository.go`

```go
package postgres

import (
	"context"
	"database/sql"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// UserProfileRepository implements repository.UserProfileRepository
type UserProfileRepository struct {
	conn *Connection
}

// Ensure UserProfileRepository implements the interface
var _ repository.UserProfileRepository = (*UserProfileRepository)(nil)

// NewUserProfileRepository creates a new UserProfileRepository
func NewUserProfileRepository(conn *Connection) *UserProfileRepository {
	return &UserProfileRepository{conn: conn}
}

// GetByUserID retrieves the gym profile by user ID (via phone number matching)
func (r *UserProfileRepository) GetByUserID(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
	// Join users table to get the phone number, then match to users_gym_profile
	query := `
		SELECT
			ugp.whatsapp_id,
			ugp.full_name,
			ugp.email,
			ugp.birthdate,
			ugp.sex,
			ugp.height,
			ugp.weight,
			ugp.training_goal,
			ugp.fitness_level,
			ugp.training_experience,
			ugp.days_per_week,
			ugp.session_duration_mins,
			ugp.health_status,
			ugp.priority_muscles,
			ugp.disliked_exercises,
			ugp.available_equipment,
			ugp.training_environment,
			ugp.preferred_training_time,
			ugp.created_at,
			ugp.updated_at
		FROM users_gym_profile ugp
		JOIN users u ON ugp.whatsapp_id LIKE '%' || u.cel_number || '%'
		WHERE u.user_id = $1
		LIMIT 1
	`

	return r.scanProfile(ctx, query, userID)
}

// GetByWhatsAppID retrieves the gym profile by WhatsApp ID
func (r *UserProfileRepository) GetByWhatsAppID(ctx context.Context, whatsappID string) (*entity.UserGymProfile, error) {
	query := `
		SELECT
			whatsapp_id,
			full_name,
			email,
			birthdate,
			sex,
			height,
			weight,
			training_goal,
			fitness_level,
			training_experience,
			days_per_week,
			session_duration_mins,
			health_status,
			priority_muscles,
			disliked_exercises,
			available_equipment,
			training_environment,
			preferred_training_time,
			created_at,
			updated_at
		FROM users_gym_profile
		WHERE whatsapp_id = $1
	`

	return r.scanProfile(ctx, query, whatsappID)
}

func (r *UserProfileRepository) scanProfile(ctx context.Context, query string, arg interface{}) (*entity.UserGymProfile, error) {
	var profile entity.UserGymProfile
	var birthdate, sex, trainingGoal, fitnessLevel, trainingExperience sql.NullString
	var sessionDuration, healthStatus, priorityMuscles, dislikedExercises sql.NullString
	var availableEquipment, trainingEnvironment, preferredTime sql.NullString
	var height sql.NullInt64
	var weight sql.NullFloat64
	var daysPerWeek sql.NullInt64
	var createdAt, updatedAt sql.NullTime

	err := r.conn.DB.QueryRowContext(ctx, query, arg).Scan(
		&profile.WhatsAppID,
		&profile.FullName,
		&profile.Email,
		&birthdate,
		&sex,
		&height,
		&weight,
		&trainingGoal,
		&fitnessLevel,
		&trainingExperience,
		&daysPerWeek,
		&sessionDuration,
		&healthStatus,
		&priorityMuscles,
		&dislikedExercises,
		&availableEquipment,
		&trainingEnvironment,
		&preferredTime,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query user profile", err)
	}

	profile.Birthdate = birthdate.String
	profile.Sex = sex.String
	profile.Height = int(height.Int64)
	profile.Weight = weight.Float64
	profile.TrainingGoal = trainingGoal.String
	profile.FitnessLevel = fitnessLevel.String
	profile.TrainingExperience = trainingExperience.String
	profile.DaysPerWeek = int(daysPerWeek.Int64)
	profile.SessionDurationMins = sessionDuration.String
	profile.HealthStatus = healthStatus.String
	profile.PriorityMuscles = priorityMuscles.String
	profile.DislikedExercises = dislikedExercises.String
	profile.AvailableEquipment = availableEquipment.String
	profile.TrainingEnvironment = trainingEnvironment.String
	profile.PreferredTrainingTime = preferredTime.String
	if createdAt.Valid {
		profile.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		profile.UpdatedAt = updatedAt.Time
	}

	return &profile, nil
}

// UpdatePreferences updates priority muscles, health status, and session duration
func (r *UserProfileRepository) UpdatePreferences(
	ctx context.Context,
	whatsappID string,
	priorityMuscles, healthStatus, sessionDuration string,
) error {
	query := `
		UPDATE users_gym_profile
		SET
			priority_muscles = COALESCE(NULLIF($2, ''), priority_muscles),
			health_status = COALESCE(NULLIF($3, ''), health_status),
			session_duration_mins = COALESCE(NULLIF($4, ''), session_duration_mins),
			updated_at = NOW()
		WHERE whatsapp_id = $1
	`

	result, err := r.conn.DB.ExecContext(ctx, query, whatsappID, priorityMuscles, healthStatus, sessionDuration)
	if err != nil {
		return apperror.NewInternalError("failed to update preferences", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return apperror.NewNotFoundError("user profile not found")
	}

	return nil
}
```

---

## 6. Router Configuration

**File:** `internal/adapter/http/router.go` (additions to existing file)

```go
package http

import (
	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/internal/adapter/http/handler"
	"github.com/gymbot/workout-tracker-back/internal/adapter/http/middleware"
)

// Router holds all HTTP handlers and configures routes
type Router struct {
	engine             *gin.Engine
	healthHandler      *handler.HealthHandler
	workoutHandler     *handler.WorkoutHandler
	setHandler         *handler.SetHandler
	planHandler        *handler.PlanHandler        // NEW
	codeResolver       middleware.CodeResolver
	internalAPIKey     string                       // NEW
	corsAllowedOrigins []string
}

// NewRouter creates a new Router with all dependencies
func NewRouter(
	healthHandler *handler.HealthHandler,
	workoutHandler *handler.WorkoutHandler,
	setHandler *handler.SetHandler,
	planHandler *handler.PlanHandler,                // NEW
	codeResolver middleware.CodeResolver,
	internalAPIKey string,                           // NEW
	corsAllowedOrigins []string,
) *Router {
	return &Router{
		healthHandler:      healthHandler,
		workoutHandler:     workoutHandler,
		setHandler:         setHandler,
		planHandler:        planHandler,              // NEW
		codeResolver:       codeResolver,
		internalAPIKey:     internalAPIKey,           // NEW
		corsAllowedOrigins: corsAllowedOrigins,
	}
}

// Setup configures the Gin engine with all routes and middleware
func (r *Router) Setup(ginMode string) *gin.Engine {
	gin.SetMode(ginMode)
	r.engine = gin.New()

	// Global middleware
	r.engine.Use(gin.Logger())
	r.engine.Use(middleware.ErrorHandler())
	r.engine.Use(middleware.CORS(r.corsAllowedOrigins))

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Health check (public)
		v1.GET("/health", r.healthHandler.Check)

		// Auth middleware (supports ?c= and ?user_id= for development)
		authMiddleware := middleware.ValidateAuth(r.codeResolver)

		// Internal API key middleware (for n8n)
		internalAuthMiddleware := middleware.ValidateInternalAPIKey(r.internalAPIKey)

		// Workout routes (protected by user auth)
		workouts := v1.Group("/workouts")
		workouts.Use(authMiddleware)
		{
			workouts.GET("/today", r.workoutHandler.GetTodayWorkout)
			workouts.POST("/:workoutId/complete", r.workoutHandler.CompleteWorkout)
		}

		// Set routes (protected by user auth)
		sets := v1.Group("/sets")
		sets.Use(authMiddleware)
		{
			sets.PATCH("/:setId", r.setHandler.Update)
			sets.PATCH("/:setId/complete", r.setHandler.MarkComplete)
		}

		// Plan routes (protected by internal API key - for n8n)
		plans := v1.Group("/plans")
		plans.Use(internalAuthMiddleware)
		{
			plans.GET("/:userId/mesocycle-status", r.planHandler.GetMesocycleStatus)
			plans.POST("/:userId/renew/maintain", r.planHandler.RenewMaintain)
			plans.POST("/:userId/renew/rotate-exercises", r.planHandler.RenewRotateExercises)
			plans.POST("/:userId/renew/change-days", r.planHandler.RenewChangeDays)
		}
	}

	// Serve static frontend files (SPA)
	r.engine.Static("/assets", "./static/assets")
	r.engine.StaticFile("/vite.svg", "./static/vite.svg")

	// SPA fallback: serve index.html for all non-API routes
	r.engine.NoRoute(func(c *gin.Context) {
		c.File("./static/index.html")
	})

	return r.engine
}

// Run starts the HTTP server
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}
```

---

## 7. Main.go Wiring

**File:** `cmd/api/main.go` (updated version)

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gymbot/workout-tracker-back/internal/adapter/http"
	"github.com/gymbot/workout-tracker-back/internal/adapter/http/handler"
	"github.com/gymbot/workout-tracker-back/internal/adapter/repository/postgres"
	"github.com/gymbot/workout-tracker-back/internal/application/usecase"
	"github.com/gymbot/workout-tracker-back/internal/config"
	"github.com/gymbot/workout-tracker-back/internal/domain/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	dbConn, err := postgres.NewConnectionFromURL(cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	log.Println("Connected to database successfully")

	// ==================== REPOSITORIES ====================

	// Existing repositories
	setRepo := postgres.NewSetRepository(dbConn)
	workoutRepo := postgres.NewWorkoutRepository(dbConn, setRepo)
	magicLinkRepo := postgres.NewMagicLinkRepository(dbConn)

	// NEW: Renewal repositories
	planRepo := postgres.NewPlanRepository(dbConn)
	scheduleRepo := postgres.NewScheduleRepository(dbConn)
	exerciseRepo := postgres.NewExerciseRepository(dbConn)
	workoutRenewalRepo := postgres.NewWorkoutRenewalRepository(dbConn)
	userProfileRepo := postgres.NewUserProfileRepository(dbConn)

	// ==================== SERVICES ====================

	// NEW: Domain services
	preferenceProcessor := service.NewPreferenceProcessor()
	exerciseRotationService := service.NewExerciseRotationService(exerciseRepo)

	// Code resolver for magic links
	codeResolver := func(code string) (string, error) {
		return magicLinkRepo.GetUserID(context.Background(), code)
	}

	// ==================== USE CASES ====================

	// Existing use cases
	getTodayWorkoutUC := usecase.NewGetTodayWorkoutUseCase(workoutRepo)
	completeWorkoutUC := usecase.NewCompleteWorkoutUseCase(workoutRepo, magicLinkRepo)
	markSetCompleteUC := usecase.NewMarkSetCompleteUseCase(setRepo)
	updateSetUC := usecase.NewUpdateSetUseCase(setRepo)

	// NEW: Renewal use cases
	checkMesocycleStatusUC := usecase.NewCheckMesocycleStatusUseCase(planRepo)
	renewMaintainUC := usecase.NewRenewMaintainUseCase(planRepo, scheduleRepo, workoutRenewalRepo)
	renewRotateExercisesUC := usecase.NewRenewRotateExercisesUseCase(
		planRepo,
		scheduleRepo,
		workoutRenewalRepo,
		userProfileRepo,
		exerciseRotationService,
		preferenceProcessor,
	)
	renewChangeDaysUC := usecase.NewRenewChangeDaysUseCase(planRepo, scheduleRepo, workoutRenewalRepo)

	// ==================== HANDLERS ====================

	// Existing handlers
	healthHandler := handler.NewHealthHandler()
	workoutHandler := handler.NewWorkoutHandler(getTodayWorkoutUC, completeWorkoutUC)
	setHandler := handler.NewSetHandler(markSetCompleteUC, updateSetUC)

	// NEW: Plan handler
	planHandler := handler.NewPlanHandler(
		checkMesocycleStatusUC,
		renewMaintainUC,
		renewRotateExercisesUC,
		renewChangeDaysUC,
	)

	// ==================== ROUTER ====================

	router := http.NewRouter(
		healthHandler,
		workoutHandler,
		setHandler,
		planHandler,                        // NEW
		codeResolver,
		cfg.Server.InternalAPIKey,          // NEW
		cfg.Server.CORSAllowedOrigins,
	)
	engine := router.Setup(cfg.Server.GinMode)

	// ==================== START SERVER ====================

	go func() {
		log.Printf("Starting server on %s", cfg.ServerAddr())
		if err := engine.Run(cfg.ServerAddr()); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
```

---

## 8. Environment Configuration

**File:** `internal/config/config.go` (additions)

```go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port               int
	GinMode            string
	CORSAllowedOrigins []string
	InternalAPIKey     string    // NEW: API key for n8n authentication
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	URL string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{}

	// Server config
	port, err := getEnvAsInt("PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}
	cfg.Server.Port = port
	cfg.Server.GinMode = getEnv("GIN_MODE", "debug")
	cfg.Server.CORSAllowedOrigins = getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"*"})

	// NEW: Internal API key (required for production)
	cfg.Server.InternalAPIKey = getEnv("INTERNAL_API_KEY", "")
	if cfg.Server.InternalAPIKey == "" && cfg.Server.GinMode == "release" {
		return nil, fmt.Errorf("INTERNAL_API_KEY is required in production")
	}
	// Set default for development
	if cfg.Server.InternalAPIKey == "" {
		cfg.Server.InternalAPIKey = "dev-api-key-not-for-production"
	}

	// Database config
	cfg.Database.URL = getEnv("SUPABASE_DB_URL", "")
	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("SUPABASE_DB_URL is required")
	}

	return cfg, nil
}

// ... rest of existing helper functions ...
```

**Example `.env` additions:**

```bash
# Existing
SUPABASE_DB_URL=postgresql://user:pass@host:5432/db
PORT=8080
GIN_MODE=debug
CORS_ALLOWED_ORIGINS=http://localhost:5173,https://gymbot.app

# NEW: Internal API key for n8n -> Backend authentication
INTERNAL_API_KEY=your-secure-api-key-here
```

---

## API Endpoint Summary

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/v1/plans/:userId/mesocycle-status` | GET | API Key | Check if mesocycle is complete |
| `/api/v1/plans/:userId/renew/maintain` | POST | API Key | Keep same routine, increment mesocycle |
| `/api/v1/plans/:userId/renew/rotate-exercises` | POST | API Key | New exercises, same frequency |
| `/api/v1/plans/:userId/renew/change-days` | POST | API Key | Change training frequency |

### Example Requests

**Check Mesocycle Status:**
```bash
curl -X GET \
  'https://api.gymbot.app/api/v1/plans/123e4567-e89b-12d3-a456-426614174000/mesocycle-status' \
  -H 'X-API-Key: your-api-key'
```

**Renew Maintain:**
```bash
curl -X POST \
  'https://api.gymbot.app/api/v1/plans/123e4567-e89b-12d3-a456-426614174000/renew/maintain' \
  -H 'X-API-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{}'
```

**Renew Rotate Exercises:**
```bash
curl -X POST \
  'https://api.gymbot.app/api/v1/plans/123e4567-e89b-12d3-a456-426614174000/renew/rotate-exercises' \
  -H 'X-API-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{"rotate_compounds": true, "rotate_isolation": true, "rotate_core": false}'
```

**Renew Change Days:**
```bash
curl -X POST \
  'https://api.gymbot.app/api/v1/plans/123e4567-e89b-12d3-a456-426614174000/renew/change-days' \
  -H 'X-API-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{"new_days_per_week": 4}'
```

---

## File Structure Summary

```
workout-tracker-back/
├── cmd/api/
│   └── main.go                              # Updated with renewal wiring
├── internal/
│   ├── adapter/
│   │   ├── http/
│   │   │   ├── handler/
│   │   │   │   ├── plan_handler.go          # NEW
│   │   │   │   └── ... (existing)
│   │   │   ├── middleware/
│   │   │   │   ├── internal_api_key.go      # NEW
│   │   │   │   └── ... (existing)
│   │   │   └── router.go                    # Updated
│   │   └── repository/postgres/
│   │       ├── plan_repository.go           # NEW
│   │       ├── schedule_repository.go       # NEW
│   │       ├── exercise_repository.go       # NEW
│   │       ├── workout_renewal_repository.go # NEW
│   │       ├── user_profile_repository.go   # NEW
│   │       └── ... (existing)
│   ├── application/
│   │   ├── dto/
│   │   │   ├── renewal_dto.go               # NEW
│   │   │   └── ... (existing)
│   │   └── usecase/
│   │       ├── check_mesocycle_status.go    # NEW
│   │       ├── renew_maintain.go            # NEW
│   │       ├── renew_rotate_exercises.go    # NEW
│   │       ├── renew_change_days.go         # NEW
│   │       └── ... (existing)
│   ├── config/
│   │   └── config.go                        # Updated
│   └── domain/
│       ├── entity/
│       │   ├── plan.go                      # NEW
│       │   ├── exercise_catalog.go          # NEW
│       │   ├── user_profile.go              # NEW
│       │   └── ... (existing)
│       ├── repository/
│       │   ├── plan_repository.go           # NEW
│       │   ├── schedule_repository.go       # NEW
│       │   ├── exercise_repository.go       # NEW
│       │   ├── workout_renewal_repository.go # NEW
│       │   ├── user_profile_repository.go   # NEW
│       │   └── ... (existing)
│       └── service/
│           ├── preference_processor.go      # NEW
│           └── exercise_rotation_service.go # NEW
└── pkg/
    ├── apperror/
    │   └── errors.go                        # Existing
    └── response/
        └── response.go                      # Existing
```

---

## Implementation Notes

1. **Transaction Safety**: The `BatchUpdateExercises` method uses a database transaction to ensure all exercise rotations are applied atomically.

2. **Health Restrictions**: The `PreferenceProcessor` maps health status codes (A-E) to specific exercise restrictions, which are then applied during exercise rotation.

3. **Deterministic Rotation**: The `ExerciseRotationService` can be seeded for reproducible results in tests using `NewExerciseRotationServiceWithSeed`.

4. **API Key Security**: The internal API key middleware should use a strong, randomly generated key in production. Never commit the actual key to source control.

5. **Import Fix**: The `renew_rotate_exercises.go` file has an import statement that needs to be moved to the top of the file in the actual implementation.

6. **Week Schedule Bug**: The mapping uses `ul_4` (Upper/Lower) for 4-day schedules, fixing the existing bug that used `ua_4`.