# 00_ARCHITECTURE.md - Alternative Exercises (Dynamic Lookup)

## 1. High-Level Architecture

```
                    exercises table (existing, unchanged)
                         |
                         | pattern match query
                         v
              GO BACKEND (workout-tracker-back)
              +---------------------------------+
              | GET /workouts/today             |
              |   1. Query today's workouts     |
              |   2. For each exercise:         |
              |      - Find 2 alternatives      |
              |        by same pattern          |
              |      - Apply set_profiles       |
              |      - Load weight history      |
              |   3. Return with alternatives   |
              +---------------------------------+
              | PATCH /sets/:setId              |
              |   - Parse {wId}:{exId}:{setN}   |
              |   - Upsert into set_values      |
              +---------------------------------+
                         |
                         v
              React Frontend (existing flip card UI)
```

### Design Principle

**No new tables. No n8n changes.** Alternatives are derived at read time from the existing `exercises` catalog using `pattern` matching. The only storage change is the `set_values` unique constraint.

---

## 2. Database Schema Changes

### 2.1 No New Tables

The `exercises` table already contains `pattern` (movement pattern). Finding alternatives is a query, not a storage problem:

```sql
-- For a primary exercise with pattern 'squat' and exercise_id 'ex-123':
SELECT exercise_id, spanish_name, role, link
FROM exercises
WHERE pattern = 'squat'
  AND exercise_id != 'ex-123'
ORDER BY RANDOM()
LIMIT 2;
```

### 2.2 Alter `set_values` Unique Constraint

```sql
-- Migration: alter_set_values_unique_constraint
--
-- Current constraint: (workout_id, set_number)
-- Problem: alternative exercises share the same workout_id but have
--          a different exercise_id. The old constraint would conflict.
--
-- New constraint: (workout_id, exercise_id, set_number)

-- Step 1: Find and drop existing unique constraint
-- (Run this query first to find the actual constraint name):
-- SELECT constraint_name FROM information_schema.table_constraints
-- WHERE table_name = 'set_values' AND constraint_type = 'UNIQUE';

ALTER TABLE set_values
    DROP CONSTRAINT IF EXISTS set_values_workout_id_set_number_key;

-- Step 2: Create new composite unique constraint
ALTER TABLE set_values
    ADD CONSTRAINT set_values_workout_exercise_set_unique
    UNIQUE (workout_id, exercise_id, set_number);
```

### 2.3 UPSERT Update

The `ON CONFLICT` clause in `SetRepository.Update()` changes from:

```sql
-- OLD
ON CONFLICT (workout_id, set_number) DO UPDATE SET ...

-- NEW
ON CONFLICT (workout_id, exercise_id, set_number) DO UPDATE SET ...
```

---

## 3. Set ID Format

### 3.1 Format Definition

| Set Type | Format | Example |
|----------|--------|---------|
| Primary | `{workoutId}-{setNumber}` | `550e8400-...-440001-2` |
| Alternative | `{workoutId}:{exerciseId}:{setNumber}` | `550e8400-...-440001:bicep_curl_db:2` |

**Key difference**: Primary uses `-` as separator (last dash). Alternative uses `:` as separator.

### 3.2 Parsing Logic

```go
func parseSetID(setID string) (workoutID, exerciseID string, setNumber int, isAlternative bool, err error) {
    if strings.Contains(setID, ":") {
        // Alternative format: {workoutId}:{exerciseId}:{setNumber}
        parts := strings.SplitN(setID, ":", 3)
        if len(parts) != 3 {
            return "", "", 0, false, errors.New("invalid alternative set_id format")
        }
        setNum, err := strconv.Atoi(parts[2])
        if err != nil {
            return "", "", 0, false, errors.New("invalid set number")
        }
        return parts[0], parts[1], setNum, true, nil
    }

    // Primary format: {workoutId}-{setNumber} (existing logic)
    lastDash := strings.LastIndex(setID, "-")
    if lastDash == -1 || lastDash == len(setID)-1 {
        return "", "", 0, false, errors.New("invalid set_id format")
    }
    workoutID = setID[:lastDash]
    setNum, err := strconv.Atoi(setID[lastDash+1:])
    if err != nil {
        return "", "", 0, false, errors.New("invalid set number")
    }
    return workoutID, "", setNum, false, nil
}
```

### 3.3 Write Path Resolution

```
Parse setId
    |
    +-- Contains ":"? → Alternative
    |       workoutId = parts[0]
    |       exerciseId = parts[1]    (already known!)
    |       setNumber = parts[2]
    |       → Validate workoutId exists in workouts table
    |       → Upsert set_values(workoutId, exerciseId, setNumber, ...)
    |
    +-- No ":"? → Primary (existing logic)
            workoutId = setId[:lastDash]
            setNumber = setId[lastDash+1:]
            → Lookup exerciseId from workouts WHERE id = workoutId
            → Upsert set_values(workoutId, exerciseId, setNumber, ...)
```

---

## 4. API Interface Definitions

### 4.1 New DTO: `AlternativeExerciseDTO`

```go
// File: internal/application/dto/workout_dto.go

type AlternativeExerciseDTO struct {
    Name        string   `json:"name"`
    RIR         string   `json:"rir"`
    RestSeconds int      `json:"restSeconds,omitempty"`
    VideoLink   string   `json:"videoLink"`
    Sets        []SetDTO `json:"sets"`
}
```

### 4.2 Modified DTO: `ExerciseDTO`

```go
type ExerciseDTO struct {
    ID                   string                    `json:"id"`
    Name                 string                    `json:"name"`
    BadgeColor           string                    `json:"badgeColor"`
    RIR                  string                    `json:"rir"`
    RestSeconds          int                       `json:"restSeconds"`
    VideoLink            string                    `json:"videoLink,omitempty"`
    Sets                 []SetDTO                  `json:"sets"`
    Tips                 []TipDTO                  `json:"tips"`
    Steps                []StepDTO                 `json:"steps"`
    AlternativeExercises []AlternativeExerciseDTO  `json:"alternativeExercises,omitempty"` // NEW
}
```

### 4.3 JSON Response (unchanged from backend_requirements.md)

```json
{
  "success": true,
  "data": {
    "exercises": [
      {
        "id": "workout-uuid-1",
        "name": "Sentadilla con barra",
        "rir": "3",
        "restSeconds": 120,
        "sets": [
          { "id": "workout-uuid-1-1", "setNumber": 1, "reps": 8, "kg": "80", "completed": false }
        ],
        "alternativeExercises": [
          {
            "name": "Goblet Squat",
            "rir": "3",
            "restSeconds": 120,
            "videoLink": "https://...",
            "sets": [
              { "id": "workout-uuid-1:goblet_squat_db:1", "setNumber": 1, "reps": 10, "kg": "-", "completed": false }
            ]
          }
        ]
      }
    ]
  }
}
```

### 4.4 Endpoints - Compatibility Matrix

| Endpoint | Change | Details |
|----------|--------|---------|
| `GET /api/v1/workouts/today` | **Modified** | Queries alternatives dynamically, returns `alternativeExercises` |
| `PATCH /api/v1/sets/:setId` | **Modified** | Parses both primary (`-`) and alternative (`:`) formats |
| `PATCH /api/v1/sets/:setId/complete` | **Modified** | Same parsing as above |
| `POST /api/v1/workouts/:workoutId/complete` | Unchanged | |
| `POST /api/v1/auth/magic-link` | Unchanged | |

---

## 5. Domain Entity Updates

### 5.1 New Entity: `AlternativeExercise`

```go
// File: internal/domain/entity/exercise.go

type AlternativeExercise struct {
    ExerciseID  string // exercises.exercise_id
    Name        string // exercises.spanish_name
    RIR         string
    RestSeconds int
    Link        string // exercises.link
    Sets        []Set
}
```

### 5.2 Modified Entity: `Exercise`

Add one field:

```go
type Exercise struct {
    // ... existing fields ...
    Alternatives []AlternativeExercise // NEW
}
```

---

## 6. Component Summary

| Component | Change Type | Scope |
|-----------|-------------|-------|
| `set_values` constraint | Migration | 1 ALTER statement |
| `entity/exercise.go` | New struct + field | ~20 lines |
| `dto/workout_dto.go` | New struct + field | ~15 lines |
| `postgres/workout_repository.go` | New query + mapping | ~80 lines |
| `usecase/get_today_workout.go` | DTO mapping | ~25 lines |
| `postgres/set_repository.go` | Parse logic + validation | ~40 lines |
| **WORKOUT_CREATOR** | **None** | **Zero changes** |
| **Frontend** | **None** | **Already implemented** |
