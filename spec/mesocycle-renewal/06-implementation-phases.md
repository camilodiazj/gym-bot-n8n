# 06 - Implementation Phases: Mesocycle Renewal Feature

**Estimated Total Duration:** 16-23 days (3-5 weeks)
**Team:** 1 Full-stack Developer
**Tech Stack:** Go/Gin (backend), n8n (workflows), PostgreSQL (Supabase), WhatsApp API

---

## Overview

This document provides detailed, actionable implementation phases for the mesocycle renewal feature. Each task includes:
- Specific file paths (absolute)
- Code sections to add/modify
- Verification steps
- Effort estimates (S = <4 hours, M = 4-8 hours, L = 8-16 hours)

---

## Dependency Graph

```
                        Phase 1 (Domain Layer)
                               │
    ┌──────────────────────────┼──────────────────────────┐
    │                          │                          │
Task 1.1               Task 1.2                   Task 1.3
plan.go          exercise_catalog.go          user_profile.go
    │                          │                          │
    └──────────────────────────┼──────────────────────────┘
                               │
                         Task 1.4
                   repository interfaces
                               │
              ┌────────────────┼────────────────┐
              │                │                │
        Task 1.5          Task 1.6         (parallel)
  PreferenceProcessor  ExerciseRotation
              │                │
              └────────────────┼────────────────┘
                               │
                        Phase 2 (Application Layer)
                               │
                         Task 2.1
                       renewal_dto.go
                               │
    ┌──────────────────────────┼──────────────────────────┐
    │              │           │           │              │
Task 2.2      Task 2.3    Task 2.4    Task 2.5      (parallel)
CheckStatus  RenewMaintain RenewRotate RenewChangeDays
    │              │           │           │
    └──────────────┴───────────┼───────────┴──────────────┘
                               │
                        Phase 3 (Adapter Layer)
                               │
    ┌──────────────────────────┼──────────────────────────┐
    │              │           │           │              │
Task 3.1      Task 3.2    Task 3.3    Task 3.4      (parallel)
plan_repo   exercise_repo  renewal_repo  (sequential)
    │              │           │           │
    └──────────────┴───────────┼───────────┘
                               │
                         Task 3.5
                   internal_auth.go
                               │
                         Task 3.6
                        router.go
                               │
                        Phase 4 (n8n Main Flow)
                               │
    ┌──────────────────────────┼──────────────────────────┐
Task 4.1 ─────► Task 4.2 ─────► Task 4.3 ─────► Task 4.4 ─────► Task 4.5
HTTP Check    If Complete    Intent Update   Switch Branch    Test
                               │
                        Phase 5 (n8n Renewal Subflow)
                               │
    ┌──────────────────────────┼──────────────────────────┐
Task 5.1      Task 5.2    Task 5.3    Task 5.4      (parallel)
MANTENER   CAMBIAR_DIAS    ROTAR    MODIFICAR_PERFIL
    │              │           │           │
    └──────────────┴───────────┼───────────┴──────────────┘
                               │
                         Task 5.5
                      System Prompts
                               │
                         Task 5.6
                      Test All Paths
                               │
                        Phase 6 (Testing)
                               │
    ┌──────────────────────────┼──────────────────────────┐
Task 6.1      Task 6.2    Task 6.3    Task 6.4      (parallel)
Fixtures    E2E Cases   Test Suite    Fix Issues
                               │
                        Phase 7 (Deployment)
                               │
    ┌──────────────────────────┼──────────────────────────┐
Task 7.1      Task 7.2    Task 7.3    Task 7.4
Cloud Run   n8n Workflows  Prod Test    Alerts
```

---

## Phase 1: Backend Domain Layer

**Duration:** 3-4 days
**Dependencies:** None
**Deliverables:** Domain layer compiles, all entities and interfaces defined

---

### Task 1.1: Create plan.go Entity

**Effort:** Medium (M) - 4-6 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/entity/plan.go`

**Code to Add:**

```go
package entity

import "time"

// Plan represents a user's training plan
type Plan struct {
    PlanID          string     `json:"plan_id"`
    UserID          string     `json:"user_id"`
    TemplateID      *string    `json:"template_id,omitempty"`
    WeekSchedule    string     `json:"week_schedule"`
    Goal            string     `json:"goal"`
    Level           string     `json:"level"`
    Status          string     `json:"status"`
    MesocycleNumber int        `json:"mesocycle_number"`
    LastRenewalDate *time.Time `json:"last_renewal_date,omitempty"`
    CreatedAt       time.Time  `json:"created_at"`
}

// MesocycleStatus represents the current state of a user's mesocycle
type MesocycleStatus struct {
    UserID           string  `json:"user_id"`
    MesocycleNumber  int     `json:"mesocycle_number"`
    DaysPerWeek      int     `json:"days_per_week"`
    WeekSchedule     string  `json:"week_schedule"`
    Week4Completed   int     `json:"week4_completed"`
    IsComplete       bool    `json:"is_complete"`
    CompletionRate   float64 `json:"completion_rate"`
}

// NewMesocycleStatus creates a new MesocycleStatus with calculated fields
func NewMesocycleStatus(userID string, mesocycle, daysPerWeek, week4Completed int, weekSchedule string) *MesocycleStatus {
    isComplete := week4Completed >= daysPerWeek
    var completionRate float64
    if daysPerWeek > 0 {
        completionRate = float64(week4Completed) / float64(daysPerWeek) * 100
    }
    return &MesocycleStatus{
        UserID:          userID,
        MesocycleNumber: mesocycle,
        DaysPerWeek:     daysPerWeek,
        WeekSchedule:    weekSchedule,
        Week4Completed:  week4Completed,
        IsComplete:      isComplete,
        CompletionRate:  completionRate,
    }
}

// RenewalResult represents the outcome of a renewal operation
type RenewalResult struct {
    Success            bool   `json:"success"`
    NewMesocycleNumber int    `json:"new_mesocycle_number"`
    NewWeekSchedule    string `json:"new_week_schedule,omitempty"`
    ExercisesRotated   int    `json:"exercises_rotated,omitempty"`
    Message            string `json:"message"`
}

// NewRenewalResult creates a successful renewal result
func NewRenewalResult(newMesocycle int, message string) *RenewalResult {
    return &RenewalResult{
        Success:            true,
        NewMesocycleNumber: newMesocycle,
        Message:            message,
    }
}

// RenewalType represents the type of renewal operation
type RenewalType string

const (
    RenewalTypeMaintain        RenewalType = "MAINTAIN"
    RenewalTypeRotateExercises RenewalType = "ROTATE_EXERCISES"
    RenewalTypeChangeDays      RenewalType = "CHANGE_DAYS"
    RenewalTypeModifyProfile   RenewalType = "MODIFY_PROFILE"
)

// Plan methods
func (p *Plan) IsActive() bool {
    return p.Status == "active"
}

func (p *Plan) CanRenew() bool {
    return p.IsActive()
}
```

**Verification Steps:**
1. `cd /Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back && go build ./internal/domain/entity/`
2. Create test file `plan_test.go` with tests for `NewMesocycleStatus` and business methods
3. Run `go test ./internal/domain/entity/ -v`

---

### Task 1.2: Create exercise_catalog.go Entity

**Effort:** Medium (M) - 3-4 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/entity/exercise_catalog.go`

**Code to Add:**

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
    Link             string   `json:"link,omitempty"`
    Equipment        string   `json:"equipment,omitempty"`
}

// ExerciseRotation represents a mapping from old to new exercise
type ExerciseRotation struct {
    OldExerciseID   string `json:"old_exercise_id"`
    OldExerciseName string `json:"old_exercise_name"`
    NewExerciseID   string `json:"new_exercise_id"`
    NewExerciseName string `json:"new_exercise_name"`
    Pattern         string `json:"pattern"`
    Role            string `json:"role"`
    DayName         string `json:"day_name"`
}

// WorkoutEntry represents an exercise assignment in a workout
type WorkoutEntry struct {
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

// MatchesPattern checks if exercise matches the given pattern
func (e *ExerciseCatalog) MatchesPattern(pattern string) bool {
    return e.Pattern == pattern
}

// MatchesRole checks if exercise matches the given role
func (e *ExerciseCatalog) MatchesRole(role string) bool {
    return e.Role == role
}
```

**Verification Steps:**
1. `go build ./internal/domain/entity/`
2. Create `exercise_catalog_test.go` with tests for `CanRotateTo`, `MatchesPattern`, `MatchesRole`
3. Run tests: `go test ./internal/domain/entity/ -v -run Exercise`

---

### Task 1.3: Create user_profile.go Entity

**Effort:** Medium (M) - 3-4 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/entity/user_profile.go`

**Code to Add:**

```go
package entity

// UserGymProfile represents the user's gym profile from KYC
type UserGymProfile struct {
    WhatsAppID         string  `json:"whatsapp_id"`
    FullName           string  `json:"full_name"`
    Email              string  `json:"email,omitempty"`
    Age                int     `json:"age"`
    Sex                string  `json:"sex"`
    HeightCM           float64 `json:"height_cm"`
    WeightKG           float64 `json:"weight_kg"`
    Goal               string  `json:"goal"`
    Level              string  `json:"level"`
    TrainingExperience string  `json:"training_experience"`
    DaysAvailable      int     `json:"days_available"`
    SessionDurationMin int     `json:"session_duration_mins"`
    PriorityMuscles    string  `json:"priority_muscles"`
    DislikedExercises  string  `json:"disliked_exercises"`
    HealthStatus       string  `json:"health_status"`
    Environment        string  `json:"environment"`
}

// ProcessedPreferences represents transformed user preferences for AI/algorithm use
type ProcessedPreferences struct {
    PriorityMusclesEN  []string          `json:"priority_muscles_en"`
    DislikedMusclesEN  []string          `json:"disliked_muscles_en"`
    ExperienceTier     string            `json:"experience_tier"`
    VolumeModifier     float64           `json:"volume_modifier"`
    HealthRestrictions HealthRestriction `json:"health"`
}

// HealthRestriction represents movement restrictions based on health status
type HealthRestriction struct {
    Code                   string `json:"code"`
    AvoidUpperBodyOverhead bool   `json:"avoid_upper_body_overhead"`
    AvoidHighImpactLegs    bool   `json:"avoid_high_impact_legs"`
    AvoidHeavyAxialLoading bool   `json:"avoid_heavy_axial_loading"`
    PreferMachines         bool   `json:"prefer_machines"`
}

// MuscleTranslation maps Spanish muscle names to English
var MuscleTranslation = map[string][]string{
    "Pecho":          {"Chest"},
    "Espalda":        {"Back", "Lats"},
    "Hombros":        {"Shoulders", "Delts"},
    "Biceps":         {"Biceps"},
    "Triceps":        {"Triceps"},
    "Gluteo":         {"Glutes"},
    "Cuadriceps":     {"Quads"},
    "Isquiotibiales": {"Hamstrings"},
    "Pantorrillas":   {"Calfs"},
    "Abdominales":    {"Abs", "Core"},
    "Pierna":         {"Quads", "Hamstrings", "Glutes"},
}

// GetHealthRestrictions returns restrictions based on health status code
func GetHealthRestrictions(code string) HealthRestriction {
    restrictions := HealthRestriction{Code: code}

    switch code {
    case "B": // Lower body issues
        restrictions.AvoidHighImpactLegs = true
    case "C": // Upper body issues
        restrictions.AvoidUpperBodyOverhead = true
    case "D": // Spine issues
        restrictions.AvoidHeavyAxialLoading = true
    case "E": // Special condition
        restrictions.PreferMachines = true
        restrictions.AvoidHeavyAxialLoading = true
    }

    return restrictions
}

// GetVolumeModifier returns volume adjustment based on session duration
func GetVolumeModifier(durationMins int) float64 {
    switch {
    case durationMins <= 30:
        return 0.6
    case durationMins <= 45:
        return 0.75
    case durationMins <= 60:
        return 0.85
    case durationMins <= 75:
        return 0.95
    default:
        return 1.0
    }
}

// GetExperienceTier returns tier based on training experience (Spanish input)
func GetExperienceTier(experience string) string {
    switch experience {
    case "Menos de 6 meses", "6 a 12 meses":
        return "beginner"
    case "1 a 2 anos", "2 a 3 anos":
        return "intermediate"
    case "Mas de 3 anos":
        return "advanced"
    default:
        return "beginner"
    }
}
```

**Verification Steps:**
1. `go build ./internal/domain/entity/`
2. Create `user_profile_test.go` with table-driven tests for:
   - `GetHealthRestrictions` (test codes A-E)
   - `GetVolumeModifier` (test boundary values)
   - `GetExperienceTier` (test all Spanish values)
3. Run tests: `go test ./internal/domain/entity/ -v -run Profile`

---

### Task 1.4: Create Repository Interfaces

**Effort:** Small (S) - 2-3 hours

**File 1:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/repository/plan_repository.go`

```go
package repository

import (
    "context"

    "github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// PlanReader defines read operations for plans
type PlanReader interface {
    GetByUserID(ctx context.Context, userID string) (*entity.Plan, error)
    GetMesocycleStatus(ctx context.Context, userID string) (*entity.MesocycleStatus, error)
}

// PlanWriter defines write operations for plans
type PlanWriter interface {
    UpdateMesocycle(ctx context.Context, userID string) (*entity.RenewalResult, error)
    UpdateWeekSchedule(ctx context.Context, userID string, newSchedule string) error
}

// PlanRepository combines all plan repository operations
type PlanRepository interface {
    PlanReader
    PlanWriter
}
```

**File 2:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/repository/exercise_repository.go`

```go
package repository

import (
    "context"

    "github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// ExerciseReader defines read operations for exercises
type ExerciseReader interface {
    GetByID(ctx context.Context, exerciseID string) (*entity.ExerciseCatalog, error)
    GetByPattern(ctx context.Context, pattern, role string) ([]entity.ExerciseCatalog, error)
    GetAlternatives(ctx context.Context, currentExerciseID string, dislikedMuscles []string) ([]entity.ExerciseCatalog, error)
}

// ExerciseRepository combines all exercise repository operations
type ExerciseRepository interface {
    ExerciseReader
}
```

**File 3:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/repository/workout_renewal_repository.go`

```go
package repository

import (
    "context"

    "github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// WorkoutRenewalReader defines read operations for workout renewal
type WorkoutRenewalReader interface {
    GetUserWorkouts(ctx context.Context, userID string, week int) ([]entity.WorkoutEntry, error)
    GetCurrentExercises(ctx context.Context, userID string) ([]entity.ExerciseCatalog, error)
}

// WorkoutRenewalWriter defines write operations for workout renewal
type WorkoutRenewalWriter interface {
    ClearSchedule(ctx context.Context, userID string) error
    DeleteWorkouts(ctx context.Context, userID string) error
    RotateExercise(ctx context.Context, userID, oldExerciseID, newExerciseID string) error
    BulkRotateExercises(ctx context.Context, userID string, rotations []entity.ExerciseRotation) error
}

// WorkoutRenewalRepository combines all workout renewal operations
type WorkoutRenewalRepository interface {
    WorkoutRenewalReader
    WorkoutRenewalWriter
}
```

**File 4:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/repository/profile_repository.go`

```go
package repository

import (
    "context"

    "github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// ProfileReader defines read operations for user profiles
type ProfileReader interface {
    GetByUserID(ctx context.Context, userID string) (*entity.UserGymProfile, error)
    GetByWhatsAppID(ctx context.Context, whatsappID string) (*entity.UserGymProfile, error)
}

// ProfileRepository combines all profile repository operations
type ProfileRepository interface {
    ProfileReader
}
```

**Verification Steps:**
1. `go build ./internal/domain/repository/`
2. Verify no import errors
3. Check interfaces follow ISP pattern (separate Reader/Writer)

---

### Task 1.5: Create PreferenceProcessor Service

**Effort:** Medium (M) - 3-4 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/service/preference_processor.go`

**First create the service directory if it doesn't exist:**
```bash
mkdir -p /Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/service
```

```go
package service

import (
    "strings"

    "github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// PreferenceProcessor transforms raw user preferences into processed format
type PreferenceProcessor struct{}

// NewPreferenceProcessor creates a new PreferenceProcessor
func NewPreferenceProcessor() *PreferenceProcessor {
    return &PreferenceProcessor{}
}

// Process transforms a UserGymProfile into ProcessedPreferences
func (p *PreferenceProcessor) Process(profile *entity.UserGymProfile) *entity.ProcessedPreferences {
    return &entity.ProcessedPreferences{
        PriorityMusclesEN:  p.translateMuscles(profile.PriorityMuscles),
        DislikedMusclesEN:  p.translateMuscles(profile.DislikedExercises),
        ExperienceTier:     entity.GetExperienceTier(profile.TrainingExperience),
        VolumeModifier:     entity.GetVolumeModifier(profile.SessionDurationMin),
        HealthRestrictions: entity.GetHealthRestrictions(profile.HealthStatus),
    }
}

// translateMuscles converts Spanish muscle names to English
func (p *PreferenceProcessor) translateMuscles(spanishMuscles string) []string {
    if spanishMuscles == "" {
        return []string{}
    }

    result := []string{}
    parts := strings.Split(spanishMuscles, ",")

    for _, part := range parts {
        muscle := strings.TrimSpace(part)
        if englishNames, ok := entity.MuscleTranslation[muscle]; ok {
            result = append(result, englishNames...)
        }
    }

    return result
}

// GetDislikedMusclesForRotation returns the list of muscles to avoid during exercise rotation
func (p *PreferenceProcessor) GetDislikedMusclesForRotation(profile *entity.UserGymProfile) []string {
    processed := p.Process(profile)
    return processed.DislikedMusclesEN
}
```

**Verification Steps:**
1. `go build ./internal/domain/service/`
2. Create `preference_processor_test.go`:

```go
package service

import (
    "testing"

    "github.com/gymbot/workout-tracker-back/internal/domain/entity"
    "github.com/stretchr/testify/assert"
)

func TestPreferenceProcessor_TranslateMuscles(t *testing.T) {
    pp := NewPreferenceProcessor()

    tests := []struct {
        name     string
        input    string
        expected []string
    }{
        {"empty", "", []string{}},
        {"single", "Pecho", []string{"Chest"}},
        {"multiple", "Pecho, Espalda", []string{"Chest", "Back", "Lats"}},
        {"pierna expands", "Pierna", []string{"Quads", "Hamstrings", "Glutes"}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := pp.translateMuscles(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

3. Run `go test ./internal/domain/service/ -v`

---

### Task 1.6: Create ExerciseRotationService

**Effort:** Large (L) - 6-8 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/service/exercise_rotation_service.go`

```go
package service

import (
    "context"
    "math/rand"
    "time"

    "github.com/gymbot/workout-tracker-back/internal/domain/entity"
    "github.com/gymbot/workout-tracker-back/internal/domain/repository"
)

// ExerciseRotationService handles deterministic exercise rotation
type ExerciseRotationService struct {
    exerciseRepo  repository.ExerciseReader
    workoutRepo   repository.WorkoutRenewalReader
    prefProcessor *PreferenceProcessor
    rng           *rand.Rand
}

// NewExerciseRotationService creates a new ExerciseRotationService
func NewExerciseRotationService(
    exerciseRepo repository.ExerciseReader,
    workoutRepo repository.WorkoutRenewalReader,
    prefProcessor *PreferenceProcessor,
) *ExerciseRotationService {
    return &ExerciseRotationService{
        exerciseRepo:  exerciseRepo,
        workoutRepo:   workoutRepo,
        prefProcessor: prefProcessor,
        rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
    }
}

// WithSeed sets a specific seed for deterministic testing
func (s *ExerciseRotationService) WithSeed(seed int64) *ExerciseRotationService {
    s.rng = rand.New(rand.NewSource(seed))
    return s
}

// RotateExercises generates exercise rotations for a user
func (s *ExerciseRotationService) RotateExercises(
    ctx context.Context,
    userID string,
    profile *entity.UserGymProfile,
) ([]entity.ExerciseRotation, error) {
    // Get current exercises from user's workouts
    currentExercises, err := s.workoutRepo.GetCurrentExercises(ctx, userID)
    if err != nil {
        return nil, err
    }

    // Get disliked muscles for filtering
    dislikedMuscles := s.prefProcessor.GetDislikedMusclesForRotation(profile)

    rotations := []entity.ExerciseRotation{}

    for _, current := range currentExercises {
        // Get alternatives for this exercise
        alternatives, err := s.exerciseRepo.GetAlternatives(ctx, current.ExerciseID, dislikedMuscles)
        if err != nil || len(alternatives) == 0 {
            continue // Skip if no alternatives found
        }

        // Filter valid candidates
        validCandidates := []entity.ExerciseCatalog{}
        for _, alt := range alternatives {
            if current.CanRotateTo(&alt, dislikedMuscles) {
                validCandidates = append(validCandidates, alt)
            }
        }

        if len(validCandidates) == 0 {
            continue
        }

        // Select a random alternative (deterministic with seed)
        selected := validCandidates[s.rng.Intn(len(validCandidates))]

        rotations = append(rotations, entity.ExerciseRotation{
            OldExerciseID:   current.ExerciseID,
            OldExerciseName: current.SpanishName,
            NewExerciseID:   selected.ExerciseID,
            NewExerciseName: selected.SpanishName,
            Pattern:         current.Pattern,
            Role:            current.Role,
        })
    }

    return rotations, nil
}

// ShouldRotate determines if an exercise should be rotated based on mesocycle count
func (s *ExerciseRotationService) ShouldRotate(exercise *entity.ExerciseCatalog, mesocycleNumber int) bool {
    switch exercise.Role {
    case "compound":
        return mesocycleNumber%4 == 0 || mesocycleNumber%3 == 0
    case "isolation":
        return mesocycleNumber%3 == 0 || mesocycleNumber%2 == 0
    case "core":
        return mesocycleNumber%3 == 0
    default:
        return false
    }
}
```

**Verification Steps:**
1. `go build ./internal/domain/service/`
2. Create `exercise_rotation_service_test.go` with mock repositories
3. Test `ShouldRotate` with various mesocycle numbers
4. Test `RotateExercises` with seeded randomness for determinism
5. Run `go test ./internal/domain/service/ -v -cover`

---

## Phase 1 Deliverables Checklist

| Task | File | Status |
|------|------|--------|
| 1.1 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/entity/plan.go` | [ ] |
| 1.2 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/entity/exercise_catalog.go` | [ ] |
| 1.3 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/entity/user_profile.go` | [ ] |
| 1.4a | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/repository/plan_repository.go` | [ ] |
| 1.4b | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/repository/exercise_repository.go` | [ ] |
| 1.4c | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/repository/workout_renewal_repository.go` | [ ] |
| 1.4d | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/repository/profile_repository.go` | [ ] |
| 1.5 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/service/preference_processor.go` | [ ] |
| 1.6 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/domain/service/exercise_rotation_service.go` | [ ] |

**Final Verification:** `cd /Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back && go build ./... && go test ./internal/domain/... -v`

---

## Phase 2: Backend Application Layer

**Duration:** 2-3 days
**Dependencies:** Phase 1 complete
**Deliverables:** Use cases with unit tests (>80% coverage)

---

### Task 2.1: Create renewal_dto.go

**Effort:** Small (S) - 2-3 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/application/dto/renewal_dto.go`

```go
package dto

import "errors"

// MesocycleStatusResponse represents the API response for mesocycle status
type MesocycleStatusResponse struct {
    UserID          string  `json:"user_id"`
    MesocycleNumber int     `json:"mesocycle_number"`
    DaysPerWeek     int     `json:"days_per_week"`
    WeekSchedule    string  `json:"week_schedule"`
    Week4Completed  int     `json:"week4_completed"`
    IsComplete      bool    `json:"is_complete"`
    CompletionRate  float64 `json:"completion_rate"`
}

// RenewalRequest represents a renewal operation request
type RenewalRequest struct {
    NewDaysPerWeek int `json:"new_days_per_week,omitempty"`
}

// RenewalResponse represents the API response for renewal operations
type RenewalResponse struct {
    Success            bool   `json:"success"`
    NewMesocycleNumber int    `json:"new_mesocycle_number"`
    NewWeekSchedule    string `json:"new_week_schedule,omitempty"`
    ExercisesRotated   int    `json:"exercises_rotated,omitempty"`
    Message            string `json:"message"`
}

// ExerciseRotationDTO represents a single exercise rotation for API response
type ExerciseRotationDTO struct {
    OldExerciseID   string `json:"old_exercise_id"`
    OldExerciseName string `json:"old_exercise_name"`
    NewExerciseID   string `json:"new_exercise_id"`
    NewExerciseName string `json:"new_exercise_name"`
    Pattern         string `json:"pattern"`
}

// RenewalWithRotationsResponse includes rotation details
type RenewalWithRotationsResponse struct {
    RenewalResponse
    Rotations []ExerciseRotationDTO `json:"rotations,omitempty"`
}

// Validation errors
var (
    ErrInvalidDaysPerWeek = errors.New("days_per_week must be between 2 and 6")
)

// ValidateChangeDaysRequest validates the new_days_per_week value
func (r *RenewalRequest) ValidateChangeDaysRequest() error {
    if r.NewDaysPerWeek < 2 || r.NewDaysPerWeek > 6 {
        return ErrInvalidDaysPerWeek
    }
    return nil
}
```

**Verification Steps:**
1. `go build ./internal/application/dto/`

---

### Task 2.2: Implement CheckMesocycleStatusUseCase

**Effort:** Medium (M) - 3-4 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/application/usecase/check_mesocycle_status.go`

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

// Execute retrieves the mesocycle status for a user
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

    return &dto.MesocycleStatusResponse{
        UserID:          status.UserID,
        MesocycleNumber: status.MesocycleNumber,
        DaysPerWeek:     status.DaysPerWeek,
        WeekSchedule:    status.WeekSchedule,
        Week4Completed:  status.Week4Completed,
        IsComplete:      status.IsComplete,
        CompletionRate:  status.CompletionRate,
    }, nil
}
```

**Test File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/application/usecase/check_mesocycle_status_test.go`

```go
package usecase

import (
    "context"
    "testing"

    "github.com/gymbot/workout-tracker-back/internal/domain/entity"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// MockPlanReader is a mock implementation of repository.PlanReader
type MockPlanReader struct {
    mock.Mock
}

func (m *MockPlanReader) GetByUserID(ctx context.Context, userID string) (*entity.Plan, error) {
    args := m.Called(ctx, userID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*entity.Plan), args.Error(1)
}

func (m *MockPlanReader) GetMesocycleStatus(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
    args := m.Called(ctx, userID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*entity.MesocycleStatus), args.Error(1)
}

func TestCheckMesocycleStatus_Complete(t *testing.T) {
    mockRepo := new(MockPlanReader)
    uc := NewCheckMesocycleStatusUseCase(mockRepo)

    expectedStatus := entity.NewMesocycleStatus("user-123", 1, 3, 3, "fb_3")

    mockRepo.On("GetMesocycleStatus", mock.Anything, "user-123").Return(expectedStatus, nil)

    result, err := uc.Execute(context.Background(), "user-123")

    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.True(t, result.IsComplete)
    assert.Equal(t, 100.0, result.CompletionRate)
    mockRepo.AssertExpectations(t)
}

func TestCheckMesocycleStatus_EmptyUserID(t *testing.T) {
    mockRepo := new(MockPlanReader)
    uc := NewCheckMesocycleStatusUseCase(mockRepo)

    result, err := uc.Execute(context.Background(), "")

    assert.Error(t, err)
    assert.Nil(t, result)
    assert.Contains(t, err.Error(), "user_id is required")
}
```

**Verification Steps:**
1. `go test ./internal/application/usecase/check_mesocycle_status_test.go -v`

---

### Task 2.3: Implement RenewMaintainUseCase

**Effort:** Medium (M) - 3-4 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/application/usecase/renew_maintain.go`

```go
package usecase

import (
    "context"
    "fmt"

    "github.com/gymbot/workout-tracker-back/internal/application/dto"
    "github.com/gymbot/workout-tracker-back/internal/domain/repository"
    "github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// RenewMaintainUseCase handles maintaining the same routine for a new mesocycle
type RenewMaintainUseCase struct {
    planRepo    repository.PlanRepository
    renewalRepo repository.WorkoutRenewalWriter
}

// NewRenewMaintainUseCase creates a new RenewMaintainUseCase
func NewRenewMaintainUseCase(
    planRepo repository.PlanRepository,
    renewalRepo repository.WorkoutRenewalWriter,
) *RenewMaintainUseCase {
    return &RenewMaintainUseCase{
        planRepo:    planRepo,
        renewalRepo: renewalRepo,
    }
}

// Execute maintains the current routine and starts a new mesocycle
func (uc *RenewMaintainUseCase) Execute(ctx context.Context, userID string) (*dto.RenewalResponse, error) {
    if userID == "" {
        return nil, apperror.NewValidationError("user_id is required")
    }

    // Clear the weekly schedule (user will need to re-schedule)
    if err := uc.renewalRepo.ClearSchedule(ctx, userID); err != nil {
        return nil, apperror.NewInternalError("failed to clear schedule", err)
    }

    // Increment mesocycle number
    result, err := uc.planRepo.UpdateMesocycle(ctx, userID)
    if err != nil {
        return nil, err
    }

    return &dto.RenewalResponse{
        Success:            result.Success,
        NewMesocycleNumber: result.NewMesocycleNumber,
        Message:            fmt.Sprintf("Rutina renovada con progresion de carga. Mesociclo %d", result.NewMesocycleNumber),
    }, nil
}
```

**Verification Steps:**
1. Create `renew_maintain_test.go` with mocked repositories
2. Test success path
3. Test error handling for ClearSchedule failure
4. `go test ./internal/application/usecase/renew_maintain_test.go -v`

---

### Task 2.4: Implement RenewRotateExercisesUseCase

**Effort:** Large (L) - 4-6 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/application/usecase/renew_rotate_exercises.go`

```go
package usecase

import (
    "context"
    "fmt"

    "github.com/gymbot/workout-tracker-back/internal/application/dto"
    "github.com/gymbot/workout-tracker-back/internal/domain/entity"
    "github.com/gymbot/workout-tracker-back/internal/domain/repository"
    "github.com/gymbot/workout-tracker-back/internal/domain/service"
    "github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// RenewRotateExercisesUseCase handles rotating exercises while keeping same frequency
type RenewRotateExercisesUseCase struct {
    planRepo        repository.PlanRepository
    renewalRepo     repository.WorkoutRenewalRepository
    rotationService *service.ExerciseRotationService
    profileRepo     repository.ProfileReader
}

// NewRenewRotateExercisesUseCase creates a new RenewRotateExercisesUseCase
func NewRenewRotateExercisesUseCase(
    planRepo repository.PlanRepository,
    renewalRepo repository.WorkoutRenewalRepository,
    rotationService *service.ExerciseRotationService,
    profileRepo repository.ProfileReader,
) *RenewRotateExercisesUseCase {
    return &RenewRotateExercisesUseCase{
        planRepo:        planRepo,
        renewalRepo:     renewalRepo,
        rotationService: rotationService,
        profileRepo:     profileRepo,
    }
}

// Execute rotates exercises and starts a new mesocycle
func (uc *RenewRotateExercisesUseCase) Execute(ctx context.Context, userID string) (*dto.RenewalWithRotationsResponse, error) {
    if userID == "" {
        return nil, apperror.NewValidationError("user_id is required")
    }

    // Get user profile for preferences
    profile, err := uc.profileRepo.GetByUserID(ctx, userID)
    if err != nil {
        return nil, apperror.NewInternalError("failed to get user profile", err)
    }

    if profile == nil {
        // If no profile found, create empty preferences
        profile = &entity.UserGymProfile{}
    }

    // Generate rotations
    rotations, err := uc.rotationService.RotateExercises(ctx, userID, profile)
    if err != nil {
        return nil, apperror.NewInternalError("failed to generate exercise rotations", err)
    }

    // Apply rotations in a transaction
    if len(rotations) > 0 {
        if err := uc.renewalRepo.BulkRotateExercises(ctx, userID, rotations); err != nil {
            return nil, apperror.NewInternalError("failed to apply exercise rotations", err)
        }
    }

    // Clear schedule
    if err := uc.renewalRepo.ClearSchedule(ctx, userID); err != nil {
        return nil, apperror.NewInternalError("failed to clear schedule", err)
    }

    // Increment mesocycle
    result, err := uc.planRepo.UpdateMesocycle(ctx, userID)
    if err != nil {
        return nil, err
    }

    // Convert rotations to DTOs
    rotationDTOs := make([]dto.ExerciseRotationDTO, len(rotations))
    for i, r := range rotations {
        rotationDTOs[i] = dto.ExerciseRotationDTO{
            OldExerciseID:   r.OldExerciseID,
            OldExerciseName: r.OldExerciseName,
            NewExerciseID:   r.NewExerciseID,
            NewExerciseName: r.NewExerciseName,
            Pattern:         r.Pattern,
        }
    }

    return &dto.RenewalWithRotationsResponse{
        RenewalResponse: dto.RenewalResponse{
            Success:            true,
            NewMesocycleNumber: result.NewMesocycleNumber,
            ExercisesRotated:   len(rotations),
            Message:            fmt.Sprintf("%d ejercicios rotados para nuevos estimulos", len(rotations)),
        },
        Rotations: rotationDTOs,
    }, nil
}
```

**Verification Steps:**
1. Create `renew_rotate_exercises_test.go`
2. Test with mocked rotation service
3. Test when no rotations are generated
4. `go test ./internal/application/usecase/renew_rotate_exercises_test.go -v`

---

### Task 2.5: Implement RenewChangeDaysUseCase

**Effort:** Medium (M) - 3-4 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/application/usecase/renew_change_days.go`

```go
package usecase

import (
    "context"
    "fmt"

    "github.com/gymbot/workout-tracker-back/internal/application/dto"
    "github.com/gymbot/workout-tracker-back/internal/domain/repository"
    "github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// RenewChangeDaysUseCase handles changing training frequency
type RenewChangeDaysUseCase struct {
    planRepo    repository.PlanRepository
    renewalRepo repository.WorkoutRenewalWriter
}

// NewRenewChangeDaysUseCase creates a new RenewChangeDaysUseCase
func NewRenewChangeDaysUseCase(
    planRepo repository.PlanRepository,
    renewalRepo repository.WorkoutRenewalWriter,
) *RenewChangeDaysUseCase {
    return &RenewChangeDaysUseCase{
        planRepo:    planRepo,
        renewalRepo: renewalRepo,
    }
}

// weekScheduleMapping maps days per week to schedule type
// Note: Using ul_4 (not ua_4) - fixed from bug in existing workflow
var weekScheduleMapping = map[int]string{
    2: "fb_2",
    3: "fb_3",
    4: "ul_4",
    5: "ppl_5",
    6: "ppl_6",
}

// Execute changes the training frequency and clears workouts for regeneration
func (uc *RenewChangeDaysUseCase) Execute(ctx context.Context, userID string, req *dto.RenewalRequest) (*dto.RenewalResponse, error) {
    if userID == "" {
        return nil, apperror.NewValidationError("user_id is required")
    }

    if err := req.ValidateChangeDaysRequest(); err != nil {
        return nil, apperror.NewValidationError(err.Error())
    }

    newSchedule, ok := weekScheduleMapping[req.NewDaysPerWeek]
    if !ok {
        return nil, apperror.NewValidationError("invalid days_per_week value")
    }

    // Delete existing workouts (they will be regenerated by GymRatForm)
    if err := uc.renewalRepo.DeleteWorkouts(ctx, userID); err != nil {
        return nil, apperror.NewInternalError("failed to delete workouts", err)
    }

    // Clear schedule
    if err := uc.renewalRepo.ClearSchedule(ctx, userID); err != nil {
        return nil, apperror.NewInternalError("failed to clear schedule", err)
    }

    // Update week schedule in plan
    if err := uc.planRepo.UpdateWeekSchedule(ctx, userID, newSchedule); err != nil {
        return nil, apperror.NewInternalError("failed to update week schedule", err)
    }

    // Increment mesocycle
    result, err := uc.planRepo.UpdateMesocycle(ctx, userID)
    if err != nil {
        return nil, err
    }

    return &dto.RenewalResponse{
        Success:            true,
        NewMesocycleNumber: result.NewMesocycleNumber,
        NewWeekSchedule:    newSchedule,
        Message:            fmt.Sprintf("Plan actualizado a %d dias por semana", req.NewDaysPerWeek),
    }, nil
}
```

**Verification Steps:**
1. Create `renew_change_days_test.go`
2. Test valid days (2-6)
3. Test invalid days (1, 7, 0)
4. Verify correct schedule mapping
5. `go test ./internal/application/usecase/renew_change_days_test.go -v`

---

## Phase 2 Deliverables Checklist

| Task | File | Status |
|------|------|--------|
| 2.1 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/application/dto/renewal_dto.go` | [ ] |
| 2.2 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/application/usecase/check_mesocycle_status.go` | [ ] |
| 2.3 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/application/usecase/renew_maintain.go` | [ ] |
| 2.4 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/application/usecase/renew_rotate_exercises.go` | [ ] |
| 2.5 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/application/usecase/renew_change_days.go` | [ ] |

**Final Verification:** `go test ./internal/application/... -v -cover` (target: >80% coverage)

---

## Phase 3: Backend Adapter Layer

**Duration:** 3-4 days
**Dependencies:** Phase 2 complete
**Deliverables:** API endpoints functional, database integration working

---

### Task 3.1: Implement PostgreSQL plan_repository

**Effort:** Large (L) - 4-6 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/repository/postgres/plan_repository.go`

See `01-backend-api.md` for complete implementation.

**Key SQL Query (GetMesocycleStatus):**
```sql
WITH week4_sessions AS (
    SELECT COUNT(*) FILTER (WHERE "Completed" = true) as completed
    FROM user_weekly_schedule
    WHERE user_id = $1 AND week = 4
),
plan_info AS (
    SELECT up.mesocycle_number, ws.days_per_week, up.week_schedule
    FROM users_plans up
    JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
    WHERE up.user_id = $1 AND up.status = 'active'
)
SELECT pi.mesocycle_number, pi.days_per_week, pi.week_schedule, COALESCE(w4.completed, 0)
FROM plan_info pi LEFT JOIN week4_sessions w4 ON true;
```

**Verification Steps:**
1. `go build ./internal/adapter/repository/postgres/`
2. Create integration test with test database
3. Test against Supabase with test user

---

### Task 3.2: Implement PostgreSQL exercise_repository

**Effort:** Medium (M) - 3-4 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/repository/postgres/exercise_repository.go`

See `01-backend-api.md` for complete implementation.

**Verification Steps:**
1. `go build ./internal/adapter/repository/postgres/`
2. Test `GetAlternatives` with real exercise data

---

### Task 3.3: Implement PostgreSQL workout_renewal_repository

**Effort:** Large (L) - 4-6 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/repository/postgres/workout_renewal_repository.go`

See `01-backend-api.md` for complete implementation.

**Key Feature:** `BulkRotateExercises` uses transaction and dynamic CASE statement.

**Verification Steps:**
1. `go build ./internal/adapter/repository/postgres/`
2. Test bulk rotation with multiple exercises
3. Verify transaction rollback on error

---

### Task 3.4: Create plan_handler with All Endpoints

**Effort:** Large (L) - 4-6 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/http/handler/plan_handler.go`

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
    checkMesocycleStatus   *usecase.CheckMesocycleStatusUseCase
    renewMaintain          *usecase.RenewMaintainUseCase
    renewRotateExercises   *usecase.RenewRotateExercisesUseCase
    renewChangeDays        *usecase.RenewChangeDaysUseCase
}

// NewPlanHandler creates a new PlanHandler
func NewPlanHandler(
    checkMesocycleStatus *usecase.CheckMesocycleStatusUseCase,
    renewMaintain *usecase.RenewMaintainUseCase,
    renewRotateExercises *usecase.RenewRotateExercisesUseCase,
    renewChangeDays *usecase.RenewChangeDaysUseCase,
) *PlanHandler {
    return &PlanHandler{
        checkMesocycleStatus:   checkMesocycleStatus,
        renewMaintain:          renewMaintain,
        renewRotateExercises:   renewRotateExercises,
        renewChangeDays:        renewChangeDays,
    }
}

// GetMesocycleStatus handles GET /api/v1/plans/:userId/mesocycle-status
func (h *PlanHandler) GetMesocycleStatus(c *gin.Context) {
    userID := c.Param("userId")
    if userID == "" {
        response.BadRequest(c, "userId path parameter is required")
        return
    }

    result, err := h.checkMesocycleStatus.Execute(c.Request.Context(), userID)
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

    result, err := h.renewMaintain.Execute(c.Request.Context(), userID)
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

    result, err := h.renewRotateExercises.Execute(c.Request.Context(), userID)
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

    var req dto.RenewalRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "invalid request body: new_days_per_week is required")
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

**Verification Steps:**
1. `go build ./internal/adapter/http/handler/`
2. Create `plan_handler_test.go` with mocked use cases
3. Test all HTTP response codes

---

### Task 3.5: Add Internal API Key Middleware

**Effort:** Small (S) - 2-3 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/http/middleware/internal_auth.go`

```go
package middleware

import (
    "os"

    "github.com/gin-gonic/gin"
)

// ValidateInternalAPIKey validates the X-Internal-API-Key header
func ValidateInternalAPIKey() gin.HandlerFunc {
    expectedKey := os.Getenv("INTERNAL_API_KEY")

    return func(c *gin.Context) {
        if expectedKey == "" {
            // Development mode - allow all requests
            c.Next()
            return
        }

        apiKey := c.GetHeader("X-Internal-API-Key")
        if apiKey == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "internal API key required"})
            return
        }

        if apiKey != expectedKey {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid internal API key"})
            return
        }

        c.Next()
    }
}
```

**Verification Steps:**
1. `go build ./internal/adapter/http/middleware/`
2. Create `internal_auth_test.go`
3. Test with/without key, valid/invalid key

---

### Task 3.6: Update router.go with New Routes

**Effort:** Medium (M) - 3-4 hours
**File to Modify:** `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/http/router.go`

**Add to Router struct:**
```go
planHandler *handler.PlanHandler
```

**Add to NewRouter parameters:**
```go
planHandler *handler.PlanHandler,
```

**Add in Setup function (inside v1 group):**
```go
// Plan routes (internal API - for n8n)
plans := v1.Group("/plans")
plans.Use(middleware.ValidateInternalAPIKey())
{
    plans.GET("/:userId/mesocycle-status", r.planHandler.GetMesocycleStatus)
    plans.POST("/:userId/renew/maintain", r.planHandler.RenewMaintain)
    plans.POST("/:userId/renew/rotate-exercises", r.planHandler.RenewRotateExercises)
    plans.POST("/:userId/renew/change-days", r.planHandler.RenewChangeDays)
}
```

**Also update main.go** with dependency wiring (see Phase 1 plan for details).

**Verification Steps:**
1. `go build ./cmd/api/`
2. `go run ./cmd/api/` - verify server starts
3. Test endpoints with curl:

```bash
# Test health (should work)
curl http://localhost:8080/api/v1/health

# Test without API key (should fail with 401)
curl http://localhost:8080/api/v1/plans/test-user/mesocycle-status

# Test with API key (set INTERNAL_API_KEY env var first)
curl -H "X-Internal-API-Key: $INTERNAL_API_KEY" \
  http://localhost:8080/api/v1/plans/test-user/mesocycle-status
```

---

## Phase 3 Deliverables Checklist

| Task | File | Status |
|------|------|--------|
| 3.1 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/repository/postgres/plan_repository.go` | [ ] |
| 3.2 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/repository/postgres/exercise_repository.go` | [ ] |
| 3.3 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/repository/postgres/workout_renewal_repository.go` | [ ] |
| 3.4 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/http/handler/plan_handler.go` | [ ] |
| 3.5 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/http/middleware/internal_auth.go` | [ ] |
| 3.6 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/internal/adapter/http/router.go` (modified) | [ ] |
| 3.6 | `/Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back/cmd/api/main.go` (modified) | [ ] |

**Final Verification:** `make test && make build && make run` - all endpoints respond correctly

---

## Phase 4: n8n Main Flow Integration

**Duration:** 2-3 days
**Dependencies:** Phase 3 complete (backend deployed)
**Deliverables:** Renewal subflow triggers correctly from main flow

---

### Task 4.1: Add HTTP_Check_Mesocycle_Status Node

**Effort:** Small (S) - 2 hours
**File to Modify:** `/Users/camilodiazjaimes/Documents/GymBot/n8n/running_flows/GymRatFlow_Supabase_V2_Workout_Tracker.json`

**Location:** After `has_planned_workouts1` FALSE branch

**Node JSON:**
```json
{
  "parameters": {
    "method": "GET",
    "url": "={{ $env.WORKOUT_API_URL }}/api/v1/plans/{{ $json.user_id }}/mesocycle-status",
    "authentication": "genericCredentialType",
    "genericAuthType": "httpHeaderAuth",
    "options": {}
  },
  "type": "n8n-nodes-base.httpRequest",
  "typeVersion": 4.2,
  "position": [1200, 400],
  "id": "http-check-mesocycle",
  "name": "HTTP_Check_Mesocycle_Status",
  "credentials": {
    "httpHeaderAuth": {
      "id": "YOUR_CREDENTIAL_ID",
      "name": "Internal API Key"
    }
  },
  "continueOnFail": true
}
```

**Verification Steps:**
1. Import modified workflow
2. Test with user ID
3. Verify response contains `is_complete` field

---

### Task 4.2: Add If_Mesocycle_Complete Conditional

**Effort:** Small (S) - 1 hour
**File to Modify:** Same workflow

**Node JSON:**
```json
{
  "parameters": {
    "conditions": {
      "options": {
        "caseSensitive": true,
        "leftValue": "",
        "typeValidation": "strict",
        "version": 2
      },
      "conditions": [
        {
          "id": "mesocycle-complete",
          "leftValue": "={{ $json.is_complete }}",
          "rightValue": true,
          "operator": {
            "type": "boolean",
            "operation": "equals"
          }
        }
      ],
      "combinator": "and"
    },
    "options": {}
  },
  "type": "n8n-nodes-base.if",
  "typeVersion": 2.2,
  "position": [1400, 400],
  "id": "if-mesocycle-complete",
  "name": "If_Mesocycle_Complete"
}
```

**Verification Steps:**
1. TRUE output should route to renewal subflow
2. FALSE output should continue to scheduling agent

---

### Task 4.3: Add RENOVAR_MESOCICLO to Intention_Agent Prompt

**Effort:** Small (S) - 1 hour
**File to Modify:** Same workflow - find the Intention_Agent node

**Add to system prompt (after CONFIRMAR_RUTINA):**
```
RENOVAR_MESOCICLO: El usuario quiere cambiar/renovar su rutina o mesociclo.
Ejemplos: "Quiero cambiar mi rutina", "Nuevos ejercicios", "Renovar mi plan", "Rotar ejercicios", "Cambiar dias de entrenamiento"
```

**Verification Steps:**
1. Test with message "Quiero renovar mi rutina"
2. Verify intention detected as RENOVAR_MESOCICLO

---

### Task 4.4: Add RENOVAR_MESOCICLO Branch to Switch Node

**Effort:** Small (S) - 1 hour
**File to Modify:** Same workflow - find the Switch_Intention node

**Add new condition:**
```json
{
  "conditions": {
    "options": {
      "caseSensitive": true,
      "leftValue": "",
      "typeValidation": "strict",
      "version": 3
    },
    "conditions": [
      {
        "leftValue": "={{ $json.intention }}",
        "rightValue": "RENOVAR_MESOCICLO",
        "operator": {
          "type": "string",
          "operation": "equals"
        },
        "id": "renovar-condition"
      }
    ],
    "combinator": "and"
  },
  "renameOutput": true,
  "outputKey": "RENOVAR_MESOCICLO"
}
```

**Verification Steps:**
1. Test routing with RENOVAR_MESOCICLO intention
2. Verify Execute Workflow node is called

---

### Task 4.5: Test Manual Renewal Trigger

**Effort:** Small (S) - 2 hours

**Test Cases:**

| Scenario | User | Message | Expected |
|----------|------|---------|----------|
| Auto-detect completion | User with week 4 complete | "Hola" | Renewal options shown |
| Manual request | Any user | "Quiero renovar" | Routes to renewal |
| Normal message | Any user | "Muestrame mi rutina" | Normal flow continues |

**Verification Steps:**
1. Execute each test case in n8n
2. Document results

---

## Phase 4 Deliverables Checklist

| Task | Description | Status |
|------|-------------|--------|
| 4.1 | HTTP_Check_Mesocycle_Status node added | [ ] |
| 4.2 | If_Mesocycle_Complete conditional added | [ ] |
| 4.3 | RENOVAR_MESOCICLO added to Intention_Agent | [ ] |
| 4.4 | RENOVAR_MESOCICLO branch in Switch node | [ ] |
| 4.5 | Manual testing completed | [ ] |

---

## Phase 5: n8n Renewal Subflow

**Duration:** 3-4 days
**Dependencies:** Phase 4 complete
**Deliverables:** All 4 renewal paths work via WhatsApp

---

### Task 5.1: Replace MANTENER SQL with HTTP POST

**Effort:** Medium (M) - 3 hours
**File to Modify:** `/Users/camilodiazjaimes/Documents/GymBot/n8n/GymBotMesocycleRenewal.json`

**Replace node `Reset_For_Mantener` with:**
```json
{
  "parameters": {
    "method": "POST",
    "url": "={{ $env.WORKOUT_API_URL }}/api/v1/plans/{{ $json.user_id }}/renew/maintain",
    "authentication": "genericCredentialType",
    "genericAuthType": "httpHeaderAuth",
    "options": {}
  },
  "type": "n8n-nodes-base.httpRequest",
  "typeVersion": 4.2,
  "position": [1120, 0],
  "id": "http-renew-maintain",
  "name": "HTTP_Renew_Maintain"
}
```

**Update Notify_Mantener_Success text:**
```
Tu rutina se ha renovado para el **Mesociclo {{ $json.new_mesocycle_number }}**.
```

**Verification Steps:**
1. Test MANTENER_RUTINA flow
2. Verify mesocycle incremented
3. Verify WhatsApp message received

---

### Task 5.2: Replace CAMBIAR_DIAS SQL with HTTP POST

**Effort:** Large (L) - 4 hours
**File to Modify:** Same workflow

**Replace `Prepare_Days_Change` with HTTP Request:**
```json
{
  "parameters": {
    "method": "POST",
    "url": "={{ $env.WORKOUT_API_URL }}/api/v1/plans/{{ $json.user_id }}/renew/change-days",
    "authentication": "genericCredentialType",
    "genericAuthType": "httpHeaderAuth",
    "sendBody": true,
    "bodyParameters": {
      "parameters": [
        {
          "name": "new_days_per_week",
          "value": "={{ $json.newDays }}"
        }
      ]
    }
  },
  "type": "n8n-nodes-base.httpRequest",
  "typeVersion": 4.2,
  "position": [1120, 160],
  "id": "http-renew-change-days",
  "name": "HTTP_Renew_Change_Days"
}
```

**Keep GymRatForm call** - it regenerates workouts after backend clears them.

**Verification Steps:**
1. Test with newDays = 4
2. Verify week_schedule updated to "ul_4"
3. Verify workouts regenerated

---

### Task 5.3: Replace ROTAR_EJERCICIOS SQL with HTTP POST

**Effort:** Medium (M) - 3 hours
**File to Modify:** Same workflow

**Remove nodes:** `Find_Alternative_Exercises`, `Process_Rotation`, `Apply_Rotation_Updates`

**Replace with single HTTP Request:**
```json
{
  "parameters": {
    "method": "POST",
    "url": "={{ $env.WORKOUT_API_URL }}/api/v1/plans/{{ $json.user_id }}/renew/rotate-exercises",
    "authentication": "genericCredentialType",
    "genericAuthType": "httpHeaderAuth"
  },
  "type": "n8n-nodes-base.httpRequest",
  "typeVersion": 4.2,
  "position": [1120, 320],
  "id": "http-renew-rotate",
  "name": "HTTP_Renew_Rotate_Exercises"
}
```

**Update notification:**
```
He seleccionado {{ $json.exercises_rotated }} ejercicios nuevos para darte estimulos frescos.
```

**Verification Steps:**
1. Test ROTAR_EJERCICIOS flow
2. Verify exercises rotated in database
3. Verify count in WhatsApp message

---

### Task 5.4: Add MODIFICAR_PERFIL Path (Optional for MVP)

**Effort:** Large (L) - 6 hours
**File to Modify:** Same workflow

This task adds a new path for users to modify their fitness profile. It requires:
1. New AI agent to collect profile changes
2. Profile update endpoint (future backend work)
3. GymRatForm regeneration

**Can be deferred to post-MVP** if time constrained.

---

### Task 5.5: Update System Prompts

**Effort:** Medium (M) - 2 hours
**File to Modify:** Same workflow - Renewal_Agent system prompt

See `03-system-prompts.md` for the complete updated prompt.

**Key changes:**
1. Add MODIFICAR_PERFIL option
2. Update first interaction message
3. Clarify output format requirements

**Verification Steps:**
1. Test all intention detection
2. Verify output format is consistent

---

### Task 5.6: Test All 4 Renewal Paths

**Effort:** Medium (M) - 3 hours

| Path | Input | Expected HTTP | Expected Outcome |
|------|-------|---------------|------------------|
| MANTENER_RUTINA | "Mantener" | POST /renew/maintain | Mesocycle +1 |
| CAMBIAR_DIAS | "4 dias" | POST /renew/change-days | ul_4 schedule |
| ROTAR_EJERCICIOS | "Nuevos ejercicios" | POST /renew/rotate-exercises | X exercises rotated |
| MODIFICAR_PERFIL | "Cambiar prioridades" | (Profile flow) | Profile updated |

**Verification Steps:**
1. Test each path with WhatsApp
2. Verify database state
3. Verify WhatsApp messages

---

## Phase 5 Deliverables Checklist

| Task | Description | Status |
|------|-------------|--------|
| 5.1 | MANTENER uses HTTP POST | [ ] |
| 5.2 | CAMBIAR_DIAS uses HTTP POST | [ ] |
| 5.3 | ROTAR_EJERCICIOS uses HTTP POST | [ ] |
| 5.4 | MODIFICAR_PERFIL path (optional) | [ ] |
| 5.5 | System prompts updated | [ ] |
| 5.6 | All paths tested | [ ] |

---

## Phase 6: Testing & QA

**Duration:** 2-3 days
**Dependencies:** Phase 5 complete
**Deliverables:** All tests pass

---

### Task 6.1: Create Test Fixture Users

**Effort:** Small (S) - 2 hours
**File:** `/Users/camilodiazjaimes/Documents/GymBot/e2e/mesocycle_renewal_test_data.sql`

```sql
-- Test User for Mesocycle Renewal
-- Phone: 570000000010

-- Cleanup
DELETE FROM n8n_chat_histories WHERE session_id LIKE '570000000010%';
DELETE FROM user_weekly_schedule WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
DELETE FROM workouts WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
DELETE FROM pending_tasks WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
DELETE FROM users_plans WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
DELETE FROM users WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
DELETE FROM users_gym_profile WHERE whatsapp_id = '570000000010';

-- Create profile
INSERT INTO users_gym_profile (
    whatsapp_id, full_name, email, age, sex, height_cm, weight_kg,
    goal, level, training_experience, days_available, session_duration_mins,
    priority_muscles, disliked_exercises, health_status, environment
) VALUES (
    '570000000010', 'Test Renewal User', 'renewal@test.com', 30, 'M', 175, 80,
    'Ganar masa muscular', 'Intermedio', '1 a 2 anos', 3, 60,
    'Pecho, Espalda', 'Pantorrillas', 'A', 'GYM'
);

-- Create user
INSERT INTO users (user_id, full_name, email, cel_number, full_phone_number, timezone)
VALUES ('e2e00010-0000-0000-0000-000000000010', 'Test Renewal User', 'renewal@test.com',
        '0000000010', '570000000010', 'America/Bogota');

-- Create plan
INSERT INTO users_plans (plan_id, user_id, week_schedule, goal, level, status, mesocycle_number)
VALUES ('plan00010-0000-0000-0000-000000000010', 'e2e00010-0000-0000-0000-000000000010',
        'fb_3', 'Ganar masa muscular', 'Intermedio', 'active', 1);

-- Create week 4 completed schedule
INSERT INTO user_weekly_schedule (day_routine_id, user_id, week, week_day, session_name, planned_day, "Completed")
VALUES
    (gen_random_uuid(), 'e2e00010-0000-0000-0000-000000000010', 4, 'Lunes', 'Dia 1', '2026-01-27', true),
    (gen_random_uuid(), 'e2e00010-0000-0000-0000-000000000010', 4, 'Miercoles', 'Dia 2', '2026-01-29', true),
    (gen_random_uuid(), 'e2e00010-0000-0000-0000-000000000010', 4, 'Viernes', 'Dia 3', '2026-01-31', true);
```

**Verification Steps:**
1. Run SQL in Supabase
2. Verify test user exists

---

### Task 6.2: Add E2E Test Cases to Test Runner

**Effort:** Large (L) - 4 hours
**File to Modify:** `/Users/camilodiazjaimes/Documents/GymBot/n8n/tests/GymRatFlow_E2E_TestRunner.json`

Add 5 new test cases (TC_REN_001 through TC_REN_005) - see `05-testing.md` for complete definitions.

**Verification Steps:**
1. Import updated test runner
2. Run individual tests
3. All pass

---

### Task 6.3: Run Full Test Suite

**Effort:** Medium (M) - 2 hours

**Steps:**
1. Run Go tests: `make test`
2. Run E2E tests in n8n
3. Generate test report

**Verification Steps:**
1. Go coverage >80%
2. E2E 100% pass rate

---

### Task 6.4: Fix Any Issues Found

**Effort:** Variable

Document and fix any bugs discovered during testing.

---

## Phase 6 Deliverables Checklist

| Task | Description | Status |
|------|-------------|--------|
| 6.1 | Test fixture users created | [ ] |
| 6.2 | 5 E2E test cases added | [ ] |
| 6.3 | Full test suite passes | [ ] |
| 6.4 | Issues fixed | [ ] |

---

## Phase 7: Deployment & Monitoring

**Duration:** 1-2 days
**Dependencies:** Phase 6 complete
**Deliverables:** Feature live in production

---

### Task 7.1: Deploy Backend to Cloud Run

**Effort:** Medium (M) - 3 hours

```bash
cd /Users/camilodiazjaimes/Documents/GymBot/workout-tracker-back

# Set environment variables
gcloud run services update workout-api \
  --set-env-vars="INTERNAL_API_KEY=${INTERNAL_API_KEY}"

# Deploy
gcloud run deploy workout-api \
  --source . \
  --region us-central1

# Verify
curl https://workout-api-xxx.run.app/api/v1/health
```

**Verification Steps:**
1. Health check returns 200
2. Internal API endpoints work

---

### Task 7.2: Update n8n Production Workflows

**Effort:** Small (S) - 2 hours

1. Set `WORKOUT_API_URL` environment variable
2. Set `INTERNAL_API_KEY` credential
3. Import updated workflows
4. Activate workflows

**Verification Steps:**
1. Workflows active
2. No execution errors

---

### Task 7.3: Test in Production with Test User

**Effort:** Small (S) - 2 hours

Test all 4 renewal paths with phone number `570000000010`.

**Verification Steps:**
1. All paths complete successfully
2. WhatsApp messages received

---

### Task 7.4: Set Up Error Alerts

**Effort:** Small (S) - 2 hours

1. Cloud Run alerting for 5xx errors
2. n8n error workflow for notifications

**Verification Steps:**
1. Trigger test error
2. Alert received

---

## Phase 7 Deliverables Checklist

| Task | Description | Status |
|------|-------------|--------|
| 7.1 | Backend deployed | [ ] |
| 7.2 | n8n workflows updated | [ ] |
| 7.3 | Production tested | [ ] |
| 7.4 | Error alerts configured | [ ] |

---

## Summary: Complete Task List

| Phase | Task | Effort | Dependencies |
|-------|------|--------|--------------|
| 1 | 1.1 plan.go | M | None |
| 1 | 1.2 exercise_catalog.go | M | None |
| 1 | 1.3 user_profile.go | M | None |
| 1 | 1.4 repository interfaces | S | 1.1-1.3 |
| 1 | 1.5 PreferenceProcessor | M | 1.3, 1.4 |
| 1 | 1.6 ExerciseRotationService | L | 1.2, 1.4, 1.5 |
| 2 | 2.1 renewal_dto.go | S | Phase 1 |
| 2 | 2.2 CheckMesocycleStatus | M | 2.1 |
| 2 | 2.3 RenewMaintain | M | 2.1 |
| 2 | 2.4 RenewRotateExercises | L | 2.1, 1.6 |
| 2 | 2.5 RenewChangeDays | M | 2.1 |
| 3 | 3.1 plan_repository | L | Phase 2 |
| 3 | 3.2 exercise_repository | M | Phase 2 |
| 3 | 3.3 workout_renewal_repository | L | Phase 2 |
| 3 | 3.4 plan_handler | L | 3.1-3.3 |
| 3 | 3.5 internal_auth middleware | S | None |
| 3 | 3.6 router.go | M | 3.4, 3.5 |
| 4 | 4.1 HTTP_Check_Mesocycle node | S | Phase 3 |
| 4 | 4.2 If_Mesocycle_Complete | S | 4.1 |
| 4 | 4.3 Intention_Agent update | S | None |
| 4 | 4.4 Switch node update | S | 4.3 |
| 4 | 4.5 Test manual trigger | S | 4.1-4.4 |
| 5 | 5.1 MANTENER HTTP | M | Phase 4 |
| 5 | 5.2 CAMBIAR_DIAS HTTP | L | Phase 4 |
| 5 | 5.3 ROTAR_EJERCICIOS HTTP | M | Phase 4 |
| 5 | 5.4 MODIFICAR_PERFIL (optional) | L | Phase 4 |
| 5 | 5.5 System prompts | M | None |
| 5 | 5.6 Test all paths | M | 5.1-5.5 |
| 6 | 6.1 Test fixtures | S | None |
| 6 | 6.2 E2E test cases | L | 6.1 |
| 6 | 6.3 Run test suite | M | 6.2 |
| 6 | 6.4 Fix issues | Variable | 6.3 |
| 7 | 7.1 Deploy backend | M | Phase 6 |
| 7 | 7.2 Deploy workflows | S | 7.1 |
| 7 | 7.3 Production test | S | 7.2 |
| 7 | 7.4 Error alerts | S | 7.3 |

**Total: 33 tasks**
- Small (S): 12 tasks
- Medium (M): 15 tasks
- Large (L): 6 tasks

**Estimated Total Duration: 16-23 days**
