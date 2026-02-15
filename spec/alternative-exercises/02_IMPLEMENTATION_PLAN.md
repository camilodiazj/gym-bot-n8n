# 02_IMPLEMENTATION_PLAN.md - Alternative Exercises (Dynamic Lookup)

## Overview

| Phase | Scope | Tasks |
|-------|-------|-------|
| Phase 1 | Database & Domain Layer | 3 tasks |
| Phase 2 | Backend Read Path (GET /workouts/today) | 3 tasks |
| Phase 3 | Backend Write Path (PATCH /sets/:setId) | 2 tasks |
| Phase 4 | Testing & Validation | 3 tasks |

**No Phase for n8n/WORKOUT_CREATOR** — zero workflow changes required.

**Dependency Chain**: Phase 1 → Phase 2 + Phase 3 (parallel) → Phase 4

---

## Phase 1: Database & Domain Layer

### T-101: Alter `set_values` Unique Constraint

- **Assignee**: [pixel-dev]
- **Input**: `00_ARCHITECTURE.md` Section 2.2
- **Technical Detail**:
  - Apply Supabase migration via `mcp__supabase__apply_migration`
  - Step 1: Query existing constraint name:
    ```sql
    SELECT constraint_name FROM information_schema.table_constraints
    WHERE table_name = 'set_values' AND constraint_type = 'UNIQUE';
    ```
  - Step 2: Drop it:
    ```sql
    ALTER TABLE set_values DROP CONSTRAINT IF EXISTS <actual_name>;
    ```
  - Step 3: Create new constraint:
    ```sql
    ALTER TABLE set_values
        ADD CONSTRAINT set_values_workout_exercise_set_unique
        UNIQUE (workout_id, exercise_id, set_number);
    ```
- **Validation [code-reviewer]**:
  - Verify old constraint is gone, new one exists
  - Test: INSERT two rows with same `(workout_id, set_number)` but different `exercise_id` → succeeds
  - Test: INSERT two rows with same `(workout_id, exercise_id, set_number)` → fails

---

### T-102: Add Domain Entity `AlternativeExercise`

- **Assignee**: [pixel-dev]
- **Input**: `00_ARCHITECTURE.md` Section 5
- **Technical Detail**:
  - File: `workout-tracker-back/internal/domain/entity/exercise.go`
  - Add struct:
    ```go
    type AlternativeExercise struct {
        ExerciseID  string
        Name        string
        RIR         string
        RestSeconds int
        Link        string
        Sets        []Set
    }
    ```
  - Add field to `Exercise`:
    ```go
    Alternatives []AlternativeExercise
    ```
  - Add methods:
    ```go
    func NewAlternativeExercise(exerciseID, name, rir string, restSeconds int, link string) *AlternativeExercise
    func (a *AlternativeExercise) AddSet(set Set)
    func (e *Exercise) AddAlternative(alt AlternativeExercise)
    ```
- **Validation [code-reviewer]**:
  - Struct fields match `00_ARCHITECTURE.md` Section 5.1
  - Constructor initializes `Sets` as empty slice (not nil)
  - No JSON tags on domain entity (JSON is DTO responsibility)

---

### T-103: Add DTO `AlternativeExerciseDTO`

- **Assignee**: [pixel-dev]
- **Input**: `00_ARCHITECTURE.md` Section 4.1 and 4.2
- **Technical Detail**:
  - File: `workout-tracker-back/internal/application/dto/workout_dto.go`
  - Add struct:
    ```go
    type AlternativeExerciseDTO struct {
        Name        string   `json:"name"`
        RIR         string   `json:"rir"`
        RestSeconds int      `json:"restSeconds,omitempty"`
        VideoLink   string   `json:"videoLink"`
        Sets        []SetDTO `json:"sets"`
    }
    ```
  - Add field to `ExerciseDTO`:
    ```go
    AlternativeExercises []AlternativeExerciseDTO `json:"alternativeExercises,omitempty"`
    ```
- **Validation [code-reviewer]**:
  - JSON tags match `backend_requirements.md` contract exactly
  - `omitempty` on `AlternativeExercises` — nil slice produces no JSON key
  - DTO does NOT include `tips`, `steps`, or `badgeColor`

---

## Phase 2: Backend Read Path (GET /workouts/today)

### T-201: Add Alternatives Query to `WorkoutRepository.GetTodayWorkout()`

- **Assignee**: [pixel-dev]
- **Input**: File `workout-tracker-back/internal/adapter/repository/postgres/workout_repository.go` (lines ~130-195)
- **Technical Detail**:
  - After the existing exercise query loop, collect all `(workoutID, exerciseID, pattern)` tuples
  - Also collect all primary `exercise_id`s into a set for dedup (BR-7)
  - Add new query for alternatives — **one query per exercise** (batching across exercises is complex due to different patterns):
    ```sql
    SELECT exercise_id, spanish_name, role, link
    FROM exercises
    WHERE pattern = $1
      AND exercise_id != $2
      AND exercise_id != ALL($3)
    ORDER BY RANDOM()
    LIMIT 2
    ```
    - `$1`: the primary exercise's pattern
    - `$2`: the primary exercise's exercise_id
    - `$3`: array of all primary exercise_ids in the session (dedup)
  - **To get the pattern**: modify the existing exercise query to also SELECT `e.pattern`:
    ```sql
    SELECT w.id, w.exercise_id, e.spanish_name, w.sets, w.reps,
           w.rir, w."rest-seconds", e.link, e.pattern, e.role
    FROM workouts w
    JOIN exercises e ON w.exercise_id = e.exercise_id
    WHERE w.user_id = $1 AND w.week = $2 AND w.day_name = $3
    ORDER BY w.exercise_order
    ```
  - For each alternative:
    1. Look up `set_profiles` for the alternative's `role` → get sets, reps, rir, rest_sec
    2. Parse sets/reps with existing `parseReps()`/`parseRepsRange()`
    3. Call `r.setRepo.GetLastWeightsForExercise(ctx, userID, altExerciseID)` for weight history
    4. Build `entity.Set` objects with IDs: `{workoutID}:{altExerciseID}:{setNumber}`
    5. Create `entity.AlternativeExercise` and attach to the `entity.Exercise`
  - **set_profiles query** (new, needed for alternatives):
    ```sql
    SELECT sets, reps, rir, rest_sec
    FROM set_profiles
    WHERE goal = $1 AND level = $2 AND week = $3 AND role = $4
    LIMIT 1
    ```
  - **Data needed from users_plans**: `goal` and `level` — these must be passed into the repository or queried. Currently `GetTodayWorkout` doesn't have access to plan data. Options:
    - **Option A**: Add a JOIN to `users_plans` in the schedule query (recommended)
    - **Option B**: Pass goal/level as additional parameters
  - Recommended: extend the schedule query:
    ```sql
    SELECT uws.day_routine_id, uws.week, uws.week_day, uws.session_name,
           uws.planned_day_utc, uws."Completed",
           up.goal, up.level
    FROM user_weekly_schedule uws
    JOIN users_plans up ON up.user_id = uws.user_id AND up.status = 'active'
    WHERE uws.user_id = $1
    AND uws.planned_day_utc = (DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota')
                                AT TIME ZONE 'America/Bogota')
    LIMIT 1
    ```
- **Validation [code-reviewer]**:
  - Verify the alternatives query excludes ALL primary exercise_ids (not just the current one)
  - Verify `set_profiles` lookup uses the alternative's `role`
  - Verify `GetLastWeightsForExercise` is called per alternative exercise
  - Verify set IDs use format `{workoutId}:{exerciseId}:{setNumber}`
  - Verify exercises with 0 alternatives get `nil` Alternatives slice
  - Verify fallback if `set_profiles` row not found for alternative's role (use primary's params)
  - Performance: N+1 queries per exercise for alternatives. Acceptable for 5-8 exercises per session. Log total query time.

---

### T-202: Update `GetTodayWorkoutUseCase.Execute()` to Map Alternatives

- **Assignee**: [pixel-dev]
- **Input**: File `workout-tracker-back/internal/application/usecase/get_today_workout.go` (lines ~48-79)
- **Technical Detail**:
  - In the exercise mapping loop, after tips/steps mapping, add:
    ```go
    if len(exercise.Alternatives) > 0 {
        altDTOs := make([]dto.AlternativeExerciseDTO, 0, len(exercise.Alternatives))
        for _, alt := range exercise.Alternatives {
            altDTO := dto.AlternativeExerciseDTO{
                Name:        alt.Name,
                RIR:         alt.RIR,
                RestSeconds: alt.RestSeconds,
                VideoLink:   alt.Link,
                Sets:        make([]dto.SetDTO, 0, len(alt.Sets)),
            }
            for _, set := range alt.Sets {
                altDTO.Sets = append(altDTO.Sets, dto.SetDTO{
                    ID:        set.ID,
                    SetNumber: set.SetNumber,
                    Reps:      set.Reps,
                    Kg:        set.Weight,
                    Completed: set.Completed,
                })
            }
            altDTOs = append(altDTOs, altDTO)
        }
        exerciseDTO.AlternativeExercises = altDTOs
    }
    ```
  - **Critical**: When no alternatives, `exerciseDTO.AlternativeExercises` stays `nil` (zero value). The `omitempty` tag omits nil slices in JSON. Do NOT initialize to empty slice.
- **Validation [code-reviewer]**:
  - Field mapping: `alt.Link` → `altDTO.VideoLink`, `set.Weight` → `setDTO.Kg`
  - `AlternativeExercises` is nil (not `[]`) when no alternatives → verify with `json.Marshal` test
  - curl test: JSON has no `alternativeExercises` key for exercises without alternatives

---

### T-203: Add `parseSetID` Utility and Integrate into Repository

- **Assignee**: [pixel-dev]
- **Input**: `00_ARCHITECTURE.md` Section 3.2
- **Technical Detail**:
  - File: `workout-tracker-back/internal/adapter/repository/postgres/set_repository.go`
  - Add function (can be package-private):
    ```go
    type parsedSetID struct {
        workoutID     string
        exerciseID    string // empty for primary sets
        setNumber     int
        isAlternative bool
    }

    func parseSetID(setID string) (parsedSetID, error) {
        if strings.Contains(setID, ":") {
            parts := strings.SplitN(setID, ":", 3)
            if len(parts) != 3 {
                return parsedSetID{}, fmt.Errorf("invalid alternative set_id format")
            }
            setNum, err := strconv.Atoi(parts[2])
            if err != nil {
                return parsedSetID{}, fmt.Errorf("invalid set number in alternative set_id")
            }
            return parsedSetID{
                workoutID:     parts[0],
                exerciseID:    parts[1],
                setNumber:     setNum,
                isAlternative: true,
            }, nil
        }

        // Existing primary logic
        lastDash := strings.LastIndex(setID, "-")
        if lastDash == -1 || lastDash == len(setID)-1 {
            return parsedSetID{}, fmt.Errorf("invalid set_id format")
        }
        setNum, err := strconv.Atoi(setID[lastDash+1:])
        if err != nil {
            return parsedSetID{}, fmt.Errorf("invalid set number")
        }
        return parsedSetID{
            workoutID:     setID[:lastDash],
            exerciseID:    "",
            setNumber:     setNum,
            isAlternative: false,
        }, nil
    }
    ```
  - This utility is used by both `Update()` and `MarkComplete()` in Phase 3
- **Validation [code-reviewer]**:
  - Unit test `parseSetID` with:
    - Primary: `"550e8400-e29b-41d4-a716-446655440001-2"` → workoutID, setNumber=2, isAlt=false
    - Alternative: `"550e8400-e29b-41d4-a716-446655440001:bicep_curl_db:3"` → workoutID, exerciseID, setNumber=3, isAlt=true
    - Invalid: `"bad-format"` → error
    - Invalid alt: `"a:b"` → error (missing setNumber)

---

## Phase 3: Backend Write Path (PATCH /sets/:setId)

### T-301: Update `SetRepository.Update()` to Handle Alternative Set IDs

- **Assignee**: [pixel-dev]
- **Input**: File `workout-tracker-back/internal/adapter/repository/postgres/set_repository.go` (lines 63-114)
- **Technical Detail**:
  - Replace the manual setID parsing (lines 64-74) with `parseSetID()` from T-203
  - Replace the lookup logic:
    ```go
    parsed, err := parseSetID(setID)
    if err != nil {
        return apperror.NewValidationError(err.Error())
    }

    var exerciseID, userID string
    if parsed.isAlternative {
        // Alternative: exerciseID already known from setID
        exerciseID = parsed.exerciseID
        // Still need userID from workouts table
        err = r.conn.DB.QueryRowContext(ctx,
            `SELECT user_id FROM workouts WHERE id = $1`,
            parsed.workoutID).Scan(&userID)
    } else {
        // Primary: lookup both from workouts table (existing logic)
        err = r.conn.DB.QueryRowContext(ctx,
            `SELECT exercise_id, user_id FROM workouts WHERE id = $1`,
            parsed.workoutID).Scan(&exerciseID, &userID)
    }
    if err == sql.ErrNoRows {
        return apperror.NewNotFoundError("workout not found")
    }
    if err != nil {
        return apperror.NewInternalError("failed to lookup workout", err)
    }
    ```
  - Update the UPSERT `ON CONFLICT` to match new constraint:
    ```sql
    ON CONFLICT (workout_id, exercise_id, set_number) DO UPDATE SET ...
    ```
  - Use `parsed.workoutID` (the original workout UUID) as `workout_id` in `set_values`
- **Validation [code-reviewer]**:
  - Test primary PATCH: `PATCH /sets/{workouts.id}-1` → upserts with primary exercise_id
  - Test alternative PATCH: `PATCH /sets/{workouts.id}:alt_exercise_id:1` → upserts with alt exercise_id
  - Test invalid: `PATCH /sets/garbage` → 400 validation error
  - Test nonexistent workout: `PATCH /sets/{fake-uuid}-1` → 404
  - Verify `set_values.workout_id` always stores `workouts.id` (never an alternative reference)
  - Verify `set_values.exercise_id` stores the correct exercise (primary or alternative)

---

### T-302: Update `SetRepository.MarkComplete()` to Handle Alternative Set IDs

- **Assignee**: [pixel-dev]
- **Input**: Same file as T-301, `MarkComplete` method
- **Technical Detail**:
  - Read the current `MarkComplete` implementation first
  - Apply same `parseSetID()` pattern as T-301
  - If `MarkComplete` writes to `set_values`: use same UNION resolution
  - If `MarkComplete` uses a different mechanism: adapt accordingly
  - Ensure the completion record has the correct `exercise_id`
- **Validation [code-reviewer]**:
  - Test: `PATCH /sets/{workouts.id}:alt_exercise_id:1/complete` → 200 success
  - Test: `PATCH /sets/{workouts.id}-1/complete` → 200 success (no regression)
  - Test: `PATCH /sets/invalid/complete` → 400 or 404

---

## Phase 4: Testing & Validation

### T-401: Unit Tests for New Code

- **Assignee**: [pixel-dev]
- **Input**: All Phase 1-3 code
- **Technical Detail**:
  - **`parseSetID` tests** (file: `set_repository_test.go`):
    - Primary format: valid UUID-int parsing
    - Alternative format: valid UUID:string:int parsing
    - Edge cases: empty string, missing parts, non-numeric setNumber
  - **Repository tests** (mock DB or integration):
    - `TestGetTodayWorkout_WithAlternatives` — exercise returns 2 alternatives
    - `TestGetTodayWorkout_NoAlternatives` — exercise has no matching pattern peers
    - `TestGetTodayWorkout_AlternativeWeightHistory` — alt has recorded weights
    - `TestUpdate_AlternativeSetId` — PATCH with `:` format
    - `TestMarkComplete_AlternativeSetId` — completion with `:` format
  - **Use case tests**:
    - `TestDTO_OmitsEmptyAlternatives` — `json.Marshal` produces no `alternativeExercises` key
    - `TestDTO_IncludesAlternatives` — `json.Marshal` produces correct structure
- **Validation [code-reviewer]**:
  - All tests pass with `make test`
  - No regression in existing tests
  - Coverage for new code > 80%

---

### T-402: E2E Test Data Setup

- **Assignee**: [pixel-dev]
- **Input**: `e2e/test_data_setup.sql`
- **Technical Detail**:
  - For test user `570000000003` (Test_WithRoutine), verify their workouts have exercises from patterns with multiple catalog entries (so alternatives can be found)
  - Add a `set_values` row for an alternative exercise to test weight pre-loading:
    ```sql
    -- Assume exercise 'goblet_squat_db' is an alternative for a squat-pattern exercise
    INSERT INTO set_values (user_id, exercise_id, workout_id, set_number, actual_weight, actual_reps, recorded_at)
    VALUES ('<test_user_uuid>', 'goblet_squat_db', '<test_workout_uuid>', 1, '20', 10, NOW() - INTERVAL '7 days');
    ```
  - Update teardown section if needed (set_values cleanup already covers by user_id)
- **Validation [code-reviewer]**:
  - Script runs cleanly end-to-end
  - Test user has exercises in patterns with 3+ exercises in catalog

---

### T-403: Manual Integration Test

- **Assignee**: [pixel-dev]
- **Input**: All phases completed, backend running locally
- **Technical Detail**:
  - **Test 1: GET response**
    ```bash
    curl "http://localhost:8080/api/v1/workouts/today?user_id=<uuid>" | jq '.data.exercises[] | {name, alt: .alternativeExercises}'
    ```
    Verify: some exercises have alternatives, others don't
  - **Test 2: Alternative set update**
    ```bash
    ALT_SET_ID="<workoutId>:<altExerciseId>:1"
    curl -X PATCH "http://localhost:8080/api/v1/sets/$ALT_SET_ID?user_id=<uuid>" \
         -H "Content-Type: application/json" -d '{"kg": "25"}'
    ```
    Verify: 200 response, `set_values` row has correct exercise_id
  - **Test 3: Alternative set complete**
    ```bash
    curl -X PATCH "http://localhost:8080/api/v1/sets/$ALT_SET_ID/complete?user_id=<uuid>"
    ```
    Verify: 200 response
  - **Test 4: Weight pre-load on next request**
    - Call GET again → verify the alternative now shows `kg: "25"` instead of `"-"`
  - **Test 5: Frontend smoke test**
    - Open workout-tracker, verify flip card shows alternatives
    - Record a weight on an alternative, refresh, verify persistence
- **Validation [code-reviewer]**:
  - All 5 tests pass
  - `set_values` data integrity verified via direct DB query
  - No console errors in frontend

---

## Dependency Graph

```
T-101 (DB constraint) ─────────────────────────┐
                                                 │
T-102 (domain entity) ──┐                       │
T-103 (DTO) ────────────┤                       │
                         │                       │
                         v                       v
               T-201 (repo: read) ◄──── T-101 done
               T-202 (use case map) ◄── T-201
               T-203 (parseSetID) ──────────────┐
                                                 │
                         T-301 (set update) ◄────┤ T-101 + T-203
                         T-302 (set complete) ◄──┘
                                    │
                                    v
                         T-401 (unit tests) ◄── Phase 2+3
                         T-402 (test data) ◄─── T-101
                         T-403 (integration) ◄─ All
```

## Parallel Execution

| Can Run in Parallel | Tasks |
|---------------------|-------|
| Group 1 | T-101 (migration) + T-102 + T-103 (Go code) |
| Group 2 | T-201 (read path) + T-203 (parseSetID utility) |
| Group 3 | T-301 + T-302 (both use parseSetID, independent of each other) |
| Group 4 | T-401 + T-402 (tests + test data, independent) |

## Total: 11 tasks (down from 15)
