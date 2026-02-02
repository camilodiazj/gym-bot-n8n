# Mesocycle Renewal - Database Operations

This document contains all SQL queries and database operations for the mesocycle renewal feature.

## Table of Contents

1. [Schema Reference](#1-schema-reference)
2. [Mesocycle Status Detection](#2-mesocycle-status-detection)
3. [MANTENER_RUTINA Operations](#3-mantener_rutina-operations)
4. [CAMBIAR_DIAS Operations](#4-cambiar_dias-operations)
5. [ROTAR_EJERCICIOS Operations](#5-rotar_ejercicios-operations)
6. [MODIFICAR_PERFIL Operations](#6-modificar_perfil-operations)
7. [Helper Queries](#7-helper-queries)
8. [Health Status Restriction Rules](#8-health-status-restriction-rules)
9. [Index Recommendations](#9-index-recommendations)
10. [Data Validation Queries](#10-data-validation-queries)

---

## 1. Schema Reference

### Core Tables Used

| Table | Primary Key | Key Columns |
|-------|-------------|-------------|
| `users_plans` | `plan_id` (UUID) | `user_id`, `week_schedule`, `mesocycle_number`, `last_renewal_date` |
| `user_weekly_schedule` | `day_routine_id` (UUID) | `user_id`, `week`, `week_day`, `session_name`, `Completed` |
| `workouts` | `id` (UUID) | `user_id`, `week`, `day_name`, `exercise_id`, `exercise_order` |
| `exercises` | `exercise_id` (TEXT) | `pattern`, `role`, `main_muscle`, `equipment` |
| `users_gym_profile` | `whatsapp_id` (BIGINT) | `health_status`, `priority_muscles`, `disliked_exercises` |
| `week_schedules` | `schedule_type` (TEXT) | `days_per_week` |

### Week Schedule Types (IMPORTANT: Bug Fix)

The database uses `ul_4` (NOT `ua_4`) for 4-day schedules:

| schedule_type | days_per_week | detail |
|---------------|---------------|--------|
| `fb_2` | 2 | Full Body 2x/week |
| `fb_3` | 3 | Full Body 3x/week |
| `ul_4` | 4 | Upper/Lower 2x each |
| `ppl_5` | 5 | Push/Pull/Legs + Upper/Lower |
| `ppl_6` | 6 | Push/Pull/Legs 2x/week |

**Note**: Any existing workflow code using `ua_4` must be corrected to `ul_4`.

### Exercise Patterns

| Pattern | Description |
|---------|-------------|
| `squat` | Squatting movements (quads dominant) |
| `hinge` | Hip hinge movements (posterior chain) |
| `lunge` | Single-leg movements |
| `push_h` | Horizontal push (chest, triceps) |
| `push_v` | Vertical push (shoulders, triceps) |
| `pull_h` | Horizontal pull (back, biceps) |
| `pull_v` | Vertical pull (lats, biceps) |
| `core` | Core/abdominal movements |
| `arm` | Isolation arm work |
| `accessory` | Accessory/isolation movements |

### Exercise Roles

| Role | exercise_order Range | Purpose |
|------|---------------------|---------|
| `compound` | 1-4 | Heavy multi-joint lifts first |
| `core` | 5-6 | Core work after main lifts |
| `isolation` | 7+ | Accessory exercises last |

---

## 2. Mesocycle Status Detection

### Primary Query: Check if Mesocycle is Complete

```sql
-- Check if user has completed week 4 of their mesocycle
-- Returns: mesocycle_number, days_per_week, completed_sessions, is_complete
WITH week4_sessions AS (
    SELECT
        COUNT(*) FILTER (WHERE "Completed" = true) as completed_count,
        COUNT(*) as total_sessions
    FROM user_weekly_schedule
    WHERE user_id = $1
      AND week = 4
),
plan_info AS (
    SELECT
        up.plan_id,
        up.mesocycle_number,
        up.last_renewal_date,
        up.week_schedule,
        ws.days_per_week
    FROM users_plans up
    JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
    WHERE up.user_id = $1
      AND up.status = 'active'
)
SELECT
    pi.plan_id,
    pi.mesocycle_number,
    pi.days_per_week,
    pi.week_schedule,
    pi.last_renewal_date,
    w4.completed_count,
    w4.total_sessions,
    (w4.completed_count >= pi.days_per_week) AS is_complete
FROM plan_info pi
CROSS JOIN week4_sessions w4;
```

### Alternative: Detailed Completion Status

```sql
-- Get detailed week-by-week completion status for a user
SELECT
    week,
    COUNT(*) as scheduled_sessions,
    COUNT(*) FILTER (WHERE "Completed" = true) as completed_sessions,
    COUNT(*) FILTER (WHERE "Completed" = false) as pending_sessions,
    ROUND(
        (COUNT(*) FILTER (WHERE "Completed" = true)::DECIMAL / COUNT(*)) * 100,
        1
    ) as completion_rate
FROM user_weekly_schedule
WHERE user_id = $1
GROUP BY week
ORDER BY week;
```

### Quick Check: Is Week 4 Complete?

```sql
-- Simple boolean check for backend use
SELECT EXISTS (
    SELECT 1
    FROM users_plans up
    JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
    WHERE up.user_id = $1
      AND up.status = 'active'
      AND (
          SELECT COUNT(*) FILTER (WHERE "Completed" = true)
          FROM user_weekly_schedule
          WHERE user_id = $1 AND week = 4
      ) >= ws.days_per_week
) AS mesocycle_complete;
```

---

## 3. MANTENER_RUTINA Operations

Keep the same exercises, clear schedule, increment mesocycle.

### Transaction Block

```sql
-- MANTENER_RUTINA: User keeps same routine with progressive overload
-- Parameters: $1 = user_id (UUID)

BEGIN;

-- Step 1: Clear existing schedule (keeps workouts/exercises intact)
DELETE FROM user_weekly_schedule
WHERE user_id = $1;

-- Step 2: Increment mesocycle counter and update renewal date
UPDATE users_plans
SET
    mesocycle_number = mesocycle_number + 1,
    last_renewal_date = NOW()
WHERE user_id = $1
  AND status = 'active';

COMMIT;
```

### Go Implementation Example

```go
// MaintainRoutine keeps current exercises, clears schedule, increments mesocycle
func (r *PlanRepository) MaintainRoutine(ctx context.Context, userID uuid.UUID) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Clear schedule
    _, err = tx.ExecContext(ctx, `
        DELETE FROM user_weekly_schedule WHERE user_id = $1
    `, userID)
    if err != nil {
        return fmt.Errorf("delete schedule: %w", err)
    }

    // Increment mesocycle
    result, err := tx.ExecContext(ctx, `
        UPDATE users_plans
        SET mesocycle_number = mesocycle_number + 1,
            last_renewal_date = NOW()
        WHERE user_id = $1 AND status = 'active'
    `, userID)
    if err != nil {
        return fmt.Errorf("update plan: %w", err)
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("no active plan found for user %s", userID)
    }

    return tx.Commit()
}
```

### Verification Query

```sql
-- Verify MANTENER_RUTINA succeeded
SELECT
    'Schedule cleared' as check_type,
    COUNT(*) = 0 as passed
FROM user_weekly_schedule
WHERE user_id = $1

UNION ALL

SELECT
    'Mesocycle incremented' as check_type,
    mesocycle_number > 1 AND last_renewal_date IS NOT NULL as passed
FROM users_plans
WHERE user_id = $1 AND status = 'active';
```

---

## 4. CAMBIAR_DIAS Operations

Change training frequency (delete workouts, update schedule type, regenerate via GymRatForm).

### Week Schedule Mapping

```sql
-- Reference: Days per week -> Schedule type mapping
-- IMPORTANT: 4 days = ul_4 (NOT ua_4)

SELECT * FROM (VALUES
    (2, 'fb_2'),
    (3, 'fb_3'),
    (4, 'ul_4'),  -- Bug fix: was incorrectly ua_4 in some workflows
    (5, 'ppl_5'),
    (6, 'ppl_6')
) AS mapping(days_per_week, schedule_type);
```

### Transaction Block

```sql
-- CAMBIAR_DIAS: User changes training frequency
-- Parameters: $1 = user_id (UUID), $2 = new_days (INT 2-6)

BEGIN;

-- Step 1: Delete all existing workouts for this user
DELETE FROM workouts
WHERE user_id = $1;

-- Step 2: Clear existing schedule
DELETE FROM user_weekly_schedule
WHERE user_id = $1;

-- Step 3: Update plan with new week_schedule
-- Map days to schedule_type: 2->fb_2, 3->fb_3, 4->ul_4, 5->ppl_5, 6->ppl_6
UPDATE users_plans
SET
    week_schedule = CASE $2::INT
        WHEN 2 THEN 'fb_2'
        WHEN 3 THEN 'fb_3'
        WHEN 4 THEN 'ul_4'  -- IMPORTANT: ul_4, not ua_4
        WHEN 5 THEN 'ppl_5'
        WHEN 6 THEN 'ppl_6'
    END,
    mesocycle_number = mesocycle_number + 1,
    last_renewal_date = NOW()
WHERE user_id = $1
  AND status = 'active';

-- Step 4: Update user profile days_available
UPDATE users_gym_profile
SET days_available = $2
WHERE whatsapp_id = (
    SELECT full_phone_number::BIGINT
    FROM users
    WHERE user_id = $1
);

COMMIT;

-- NOTE: After this transaction, call GymRatForm workflow to regenerate workouts
```

### Go Implementation Example

```go
// ChangeDays updates training frequency
func (r *PlanRepository) ChangeDays(ctx context.Context, userID uuid.UUID, newDays int) error {
    if newDays < 2 || newDays > 6 {
        return fmt.Errorf("invalid days: must be between 2 and 6")
    }

    scheduleMap := map[int]string{
        2: "fb_2",
        3: "fb_3",
        4: "ul_4", // NOT ua_4!
        5: "ppl_5",
        6: "ppl_6",
    }

    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Delete workouts
    _, err = tx.ExecContext(ctx, `DELETE FROM workouts WHERE user_id = $1`, userID)
    if err != nil {
        return fmt.Errorf("delete workouts: %w", err)
    }

    // Delete schedule
    _, err = tx.ExecContext(ctx, `DELETE FROM user_weekly_schedule WHERE user_id = $1`, userID)
    if err != nil {
        return fmt.Errorf("delete schedule: %w", err)
    }

    // Update plan
    _, err = tx.ExecContext(ctx, `
        UPDATE users_plans
        SET week_schedule = $2,
            mesocycle_number = mesocycle_number + 1,
            last_renewal_date = NOW()
        WHERE user_id = $1 AND status = 'active'
    `, userID, scheduleMap[newDays])
    if err != nil {
        return fmt.Errorf("update plan: %w", err)
    }

    return tx.Commit()
}
```

### Verification Query

```sql
-- Verify CAMBIAR_DIAS succeeded
SELECT
    up.user_id,
    up.week_schedule,
    ws.days_per_week,
    up.mesocycle_number,
    (SELECT COUNT(*) FROM workouts WHERE user_id = $1) as workout_count,
    (SELECT COUNT(*) FROM user_weekly_schedule WHERE user_id = $1) as schedule_count
FROM users_plans up
JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
WHERE up.user_id = $1;
-- Expected: workout_count = 0, schedule_count = 0, new week_schedule
```

---

## 5. ROTAR_EJERCICIOS Operations

Replace exercises with alternatives that match the same pattern and role.

### Step 1: Get Current Exercises with Patterns

```sql
-- CTE to get user's current exercises grouped by pattern + role
WITH current_exercises AS (
    SELECT DISTINCT
        w.exercise_id,
        e.pattern,
        e.role,
        e.main_muscle,
        e.equipment
    FROM workouts w
    JOIN exercises e ON w.exercise_id = e.exercise_id
    WHERE w.user_id = $1
)
SELECT * FROM current_exercises
ORDER BY pattern, role;
```

### Step 2: Find Alternatives by Pattern + Role

```sql
-- Find alternative exercises for a specific pattern + role
-- Parameters: $1 = user_id, $2 = pattern, $3 = role, $4 = current_exercise_id

WITH user_preferences AS (
    -- Get user's disliked muscles and health restrictions
    SELECT
        ugp.disliked_exercises,
        ugp.health_status,
        ugp.priority_muscles
    FROM users_gym_profile ugp
    JOIN users u ON u.full_phone_number::BIGINT = ugp.whatsapp_id
    WHERE u.user_id = $1
),
excluded_muscles AS (
    -- Map disliked exercises (Spanish) to English muscle names
    SELECT m.main_muscle
    FROM muscles m
    CROSS JOIN user_preferences up
    WHERE LOWER(m.main_muscle_spanish) = ANY(
        SELECT LOWER(TRIM(unnest(string_to_array(up.disliked_exercises, ','))))
    )
),
health_excluded_patterns AS (
    -- Patterns to exclude based on health_status
    SELECT pattern FROM (VALUES
        -- Health Status B: Lower body issues
        ('B', 'squat'),
        ('B', 'lunge'),
        -- Health Status C: Upper body issues
        ('C', 'push_v'),
        -- Health Status D: Spine issues
        ('D', 'hinge')
    ) AS restrictions(health_code, pattern)
    CROSS JOIN user_preferences up
    WHERE restrictions.health_code = up.health_status
)
SELECT
    e.exercise_id,
    e.spanish_name,
    e.pattern,
    e.role,
    e.main_muscle,
    e.equipment
FROM exercises e
WHERE e.pattern = $2
  AND e.role = $3
  AND e.exercise_id != $4  -- Exclude current exercise
  AND e.exercise_id NOT IN (
      -- Exclude exercises already in user's workouts
      SELECT exercise_id FROM workouts WHERE user_id = $1
  )
  AND e.main_muscle NOT IN (SELECT main_muscle FROM excluded_muscles)
  AND e.pattern NOT IN (SELECT pattern FROM health_excluded_patterns)
ORDER BY
    -- Prefer exercises matching priority muscles
    CASE WHEN e.main_muscle IN (
        SELECT m.main_muscle
        FROM muscles m
        CROSS JOIN user_preferences up
        WHERE LOWER(m.main_muscle_spanish) = ANY(
            SELECT LOWER(TRIM(unnest(string_to_array(up.priority_muscles, ','))))
        )
    ) THEN 0 ELSE 1 END,
    RANDOM()  -- Add randomness for variety
LIMIT 1;
```

### Step 3: Complete Exercise Rotation Transaction

```sql
-- ROTAR_EJERCICIOS: Replace exercises with alternatives
-- This is a complex operation best done in application code

BEGIN;

-- Step 1: Create temp table with rotation mapping
CREATE TEMP TABLE exercise_rotation AS
WITH current_exercises AS (
    SELECT DISTINCT
        w.exercise_id as old_exercise_id,
        e.pattern,
        e.role
    FROM workouts w
    JOIN exercises e ON w.exercise_id = e.exercise_id
    WHERE w.user_id = $1
),
alternatives AS (
    SELECT DISTINCT ON (ce.old_exercise_id)
        ce.old_exercise_id,
        alt.exercise_id as new_exercise_id
    FROM current_exercises ce
    JOIN exercises alt ON alt.pattern = ce.pattern
                       AND alt.role = ce.role
    WHERE alt.exercise_id != ce.old_exercise_id
      AND alt.exercise_id NOT IN (SELECT exercise_id FROM workouts WHERE user_id = $1)
      -- Add exclusion rules here (see health status section)
    ORDER BY ce.old_exercise_id, RANDOM()
)
SELECT * FROM alternatives;

-- Step 2: Update workouts with new exercises
UPDATE workouts w
SET exercise_id = er.new_exercise_id
FROM exercise_rotation er
WHERE w.user_id = $1
  AND w.exercise_id = er.old_exercise_id;

-- Step 3: Clear schedule
DELETE FROM user_weekly_schedule
WHERE user_id = $1;

-- Step 4: Increment mesocycle
UPDATE users_plans
SET
    mesocycle_number = mesocycle_number + 1,
    last_renewal_date = NOW()
WHERE user_id = $1
  AND status = 'active';

-- Step 5: Cleanup
DROP TABLE exercise_rotation;

COMMIT;
```

### Go Implementation: Deterministic Rotation

```go
// RotateExercises replaces exercises with alternatives of same pattern+role
type ExerciseRotation struct {
    OldExerciseID string
    NewExerciseID string
    Pattern       string
    Role          string
}

func (r *WorkoutRepository) RotateExercises(
    ctx context.Context,
    userID uuid.UUID,
    excludedMuscles []string,
    excludedPatterns []string,
) ([]ExerciseRotation, error) {

    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Get current exercises
    currentExercises := []struct {
        ExerciseID string
        Pattern    string
        Role       string
    }{}

    rows, err := tx.QueryContext(ctx, `
        SELECT DISTINCT w.exercise_id, e.pattern, e.role
        FROM workouts w
        JOIN exercises e ON w.exercise_id = e.exercise_id
        WHERE w.user_id = $1
    `, userID)
    // ... populate currentExercises

    rotations := []ExerciseRotation{}
    usedExercises := map[string]bool{}

    for _, curr := range currentExercises {
        // Find alternative
        var newExerciseID string
        err := tx.QueryRowContext(ctx, `
            SELECT e.exercise_id
            FROM exercises e
            WHERE e.pattern = $1
              AND e.role = $2
              AND e.exercise_id != $3
              AND e.exercise_id NOT IN (SELECT exercise_id FROM workouts WHERE user_id = $4)
              AND e.main_muscle NOT IN (SELECT unnest($5::text[]))
              AND e.pattern NOT IN (SELECT unnest($6::text[]))
            ORDER BY RANDOM()
            LIMIT 1
        `, curr.Pattern, curr.Role, curr.ExerciseID, userID,
           pq.Array(excludedMuscles), pq.Array(excludedPatterns),
        ).Scan(&newExerciseID)

        if err == nil && !usedExercises[newExerciseID] {
            usedExercises[newExerciseID] = true
            rotations = append(rotations, ExerciseRotation{
                OldExerciseID: curr.ExerciseID,
                NewExerciseID: newExerciseID,
                Pattern:       curr.Pattern,
                Role:          curr.Role,
            })
        }
    }

    // Apply rotations
    for _, rot := range rotations {
        _, err := tx.ExecContext(ctx, `
            UPDATE workouts
            SET exercise_id = $1
            WHERE user_id = $2 AND exercise_id = $3
        `, rot.NewExerciseID, userID, rot.OldExerciseID)
        if err != nil {
            return nil, fmt.Errorf("update exercise %s: %w", rot.OldExerciseID, err)
        }
    }

    // Clear schedule
    _, _ = tx.ExecContext(ctx, `DELETE FROM user_weekly_schedule WHERE user_id = $1`, userID)

    // Increment mesocycle
    _, _ = tx.ExecContext(ctx, `
        UPDATE users_plans
        SET mesocycle_number = mesocycle_number + 1, last_renewal_date = NOW()
        WHERE user_id = $1 AND status = 'active'
    `, userID)

    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit: %w", err)
    }

    return rotations, nil
}
```

---

## 6. MODIFICAR_PERFIL Operations

Update user preferences and regenerate workouts.

### Update Profile Query

```sql
-- MODIFICAR_PERFIL: Update user gym profile
-- Parameters: All fields are optional - only update provided values

UPDATE users_gym_profile
SET
    priority_muscles = COALESCE($2, priority_muscles),
    disliked_exercises = COALESCE($3, disliked_exercises),
    health_status = COALESCE($4, health_status),
    session_duration_mins = COALESCE($5, session_duration_mins),
    days_available = COALESCE($6, days_available),
    primary_goal = COALESCE($7, primary_goal)
WHERE whatsapp_id = $1
RETURNING *;
```

### Full MODIFICAR_PERFIL Transaction

```sql
-- MODIFICAR_PERFIL: Update profile, clear workouts, prepare for regeneration
-- Parameters: $1 = whatsapp_id, $2-$7 = profile fields (nullable)

BEGIN;

-- Step 1: Get user_id from whatsapp
SELECT user_id INTO TEMP user_lookup
FROM users
WHERE full_phone_number::BIGINT = $1;

-- Step 2: Update profile
UPDATE users_gym_profile
SET
    priority_muscles = COALESCE($2, priority_muscles),
    disliked_exercises = COALESCE($3, disliked_exercises),
    health_status = COALESCE($4, health_status),
    session_duration_mins = COALESCE($5, session_duration_mins),
    days_available = COALESCE($6, days_available)
WHERE whatsapp_id = $1;

-- Step 3: Delete workouts
DELETE FROM workouts
WHERE user_id = (SELECT user_id FROM user_lookup);

-- Step 4: Clear schedule
DELETE FROM user_weekly_schedule
WHERE user_id = (SELECT user_id FROM user_lookup);

-- Step 5: Update plan if days changed
UPDATE users_plans
SET
    week_schedule = CASE $6::INT
        WHEN 2 THEN 'fb_2'
        WHEN 3 THEN 'fb_3'
        WHEN 4 THEN 'ul_4'
        WHEN 5 THEN 'ppl_5'
        WHEN 6 THEN 'ppl_6'
        ELSE week_schedule
    END,
    mesocycle_number = mesocycle_number + 1,
    last_renewal_date = NOW()
WHERE user_id = (SELECT user_id FROM user_lookup)
  AND status = 'active';

DROP TABLE user_lookup;

COMMIT;

-- NOTE: After this transaction, call GymRatForm workflow to regenerate workouts
```

### Update Specific Fields

```sql
-- Update only priority muscles
UPDATE users_gym_profile
SET priority_muscles = $2
WHERE whatsapp_id = $1;

-- Update only health status
UPDATE users_gym_profile
SET health_status = $2
WHERE whatsapp_id = $1;

-- Update only session duration
UPDATE users_gym_profile
SET session_duration_mins = $2
WHERE whatsapp_id = $1;
```

---

## 7. Helper Queries

### Get User Profile with Preferences

```sql
-- Get complete user profile for renewal decisions
SELECT
    u.user_id,
    u.full_name,
    u.full_phone_number,
    ugp.whatsapp_id,
    ugp.primary_goal,
    ugp.health_status,
    hs.value as health_status_description,
    ugp.fitness_level,
    ugp.days_available,
    ugp.session_duration_mins,
    ugp.priority_muscles,
    ugp.disliked_exercises,
    ugp.training_style,
    ugp.biological_sex,
    up.plan_id,
    up.mesocycle_number,
    up.week_schedule,
    ws.days_per_week,
    up.last_renewal_date
FROM users u
JOIN users_gym_profile ugp ON u.full_phone_number::BIGINT = ugp.whatsapp_id
LEFT JOIN health_status hs ON ugp.health_status = hs.id
LEFT JOIN users_plans up ON u.user_id = up.user_id AND up.status = 'active'
LEFT JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
WHERE u.user_id = $1;
```

### Get Exercises by Pattern and Role

```sql
-- Get available exercises for a pattern/role combination
SELECT
    e.exercise_id,
    e.spanish_name,
    e.name as english_name,
    e.pattern,
    e.role,
    e.main_muscle,
    m.main_muscle_spanish,
    e.secondary_muscles,
    e.equipment,
    e.level,
    e.link
FROM exercises e
JOIN muscles m ON e.main_muscle = m.main_muscle
WHERE e.pattern = $1
  AND e.role = $2
  AND e.level IN ($3, 'Principiante')  -- Include beginner-friendly options
ORDER BY
    CASE e.level
        WHEN 'Principiante' THEN 1
        WHEN 'Intermedio' THEN 2
        WHEN 'Avanzado' THEN 3
    END,
    e.spanish_name;
```

### Spanish to English Muscle Mapping Reference

```sql
-- Complete muscle mapping for reference
SELECT
    main_muscle as english_name,
    main_muscle_spanish as spanish_name
FROM muscles
ORDER BY main_muscle_spanish;
```

| English | Spanish |
|---------|---------|
| Abs | Abdominales |
| Back | Espalda |
| Biceps | Biceps |
| Calfs | Pantorrillas |
| Chest | Pecho |
| Core | Abdomen |
| Forearms | Antebrazos |
| Front Shoulders | Deltoides frontales |
| Glutes | Gluteos |
| Hamstrings | Femorales |
| Lower back | Espalda baja |
| Neck | Cuello |
| Quads | Cuadriceps |
| Rear Shoulders | Deltoides posteriores |
| Shoulders | Hombros |
| Traps | Trapecio |
| Triceps | Triceps |

### Get User's Current Workout Summary

```sql
-- Summary of user's current workout structure
SELECT
    w.day_name,
    COUNT(*) as exercise_count,
    STRING_AGG(DISTINCT e.pattern, ', ' ORDER BY e.pattern) as patterns,
    STRING_AGG(DISTINCT e.role, ', ' ORDER BY e.role) as roles
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = $1
  AND w.week = 1  -- Week 1 is representative
GROUP BY w.day_name
ORDER BY w.day_name;
```

---

## 8. Health Status Restriction Rules

### Health Status Reference Table

```sql
-- Health status values from database
SELECT * FROM health_status;
```

| ID | Value (Spanish) | Restriction Level |
|----|-----------------|-------------------|
| A | Estoy al 100% (Sin dolor ni lesiones) | None |
| B | Cuidado en Tren Inferior (Rodillas, tobillos, cadera) | Lower Body |
| C | Cuidado en Tren Superior (Hombros, codos, munecas) | Upper Body |
| D | Cuidado en Espalda (Lumbares o cervicales) | Spine |
| E | Condicion Medica Especial | Full Caution |

### Exclusion Rules by Health Status

```sql
-- Create function to get excluded patterns/equipment by health status
CREATE OR REPLACE FUNCTION get_health_exclusions(health_code TEXT)
RETURNS TABLE (
    excluded_patterns TEXT[],
    excluded_equipment TEXT[],
    prefer_equipment TEXT[]
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        CASE health_code
            WHEN 'A' THEN ARRAY[]::TEXT[]
            WHEN 'B' THEN ARRAY['squat', 'lunge']  -- High-impact lower body
            WHEN 'C' THEN ARRAY['push_v']          -- Overhead pressing
            WHEN 'D' THEN ARRAY['hinge']           -- Heavy axial loading
            WHEN 'E' THEN ARRAY[]::TEXT[]          -- No pattern exclusions, equipment preference
        END as excluded_patterns,
        CASE health_code
            WHEN 'A' THEN ARRAY[]::TEXT[]
            WHEN 'B' THEN ARRAY[]::TEXT[]
            WHEN 'C' THEN ARRAY[]::TEXT[]
            WHEN 'D' THEN ARRAY['barbell']         -- Avoid heavy barbell work
            WHEN 'E' THEN ARRAY['barbell', 'dumbbell']  -- Prefer machines
        END as excluded_equipment,
        CASE health_code
            WHEN 'E' THEN ARRAY['machine', 'cable']  -- Prioritize controlled movements
            ELSE ARRAY[]::TEXT[]
        END as prefer_equipment;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
```

### Detailed Restriction Rules

#### Health Status A: No Restrictions
```sql
-- Full exercise selection available
SELECT * FROM exercises WHERE pattern = $1 AND role = $2;
```

#### Health Status B: Lower Body Issues
```sql
-- Exclude high-impact on knees/ankles
-- Avoid: squat jumps, box jumps, deep lunges
SELECT * FROM exercises
WHERE pattern = $1
  AND role = $2
  AND pattern NOT IN ('squat', 'lunge')  -- or use modified versions
  AND (
      spanish_name NOT ILIKE '%salto%'
      AND spanish_name NOT ILIKE '%jump%'
      AND spanish_name NOT ILIKE '%box%'
  );
```

#### Health Status C: Upper Body Issues
```sql
-- Avoid overhead movements (push_v pattern)
-- Redirects to horizontal pressing and lateral raises
SELECT * FROM exercises
WHERE pattern = $1
  AND role = $2
  AND pattern != 'push_v';

-- Alternative: If push_v was required, substitute with push_h or lateral work
-- Example: Replace military press with incline press
```

#### Health Status D: Spine Issues
```sql
-- Avoid heavy axial loading (deadlifts, heavy squats, good mornings)
-- Prefer supported/machine variations
SELECT * FROM exercises
WHERE pattern = $1
  AND role = $2
  AND pattern != 'hinge'  -- Avoid deadlift variations
  AND equipment != 'barbell'
  AND (
      spanish_name NOT ILIKE '%peso muerto%'
      AND spanish_name NOT ILIKE '%deadlift%'
      AND spanish_name NOT ILIKE '%good morning%'
  );
```

#### Health Status E: Special Medical Condition
```sql
-- Prioritize machines over free weights
-- Low-risk, controlled movements
SELECT * FROM exercises
WHERE pattern = $1
  AND role = $2
  AND equipment IN ('machine', 'cable', 'Maquina', 'Polea')
ORDER BY
    CASE equipment
        WHEN 'machine' THEN 1
        WHEN 'Maquina' THEN 1
        WHEN 'cable' THEN 2
        WHEN 'Polea' THEN 2
        ELSE 3
    END;
```

### Exercise Selection with Health Restrictions

```sql
-- Complete exercise selection query with health status consideration
WITH user_health AS (
    SELECT health_status
    FROM users_gym_profile ugp
    JOIN users u ON u.full_phone_number::BIGINT = ugp.whatsapp_id
    WHERE u.user_id = $1
),
health_rules AS (
    SELECT * FROM get_health_exclusions((SELECT health_status FROM user_health))
)
SELECT e.*
FROM exercises e
CROSS JOIN health_rules hr
WHERE e.pattern = $2
  AND e.role = $3
  AND e.pattern != ALL(hr.excluded_patterns)
  AND (
      CARDINALITY(hr.excluded_equipment) = 0
      OR e.equipment != ALL(hr.excluded_equipment)
  )
ORDER BY
    CASE
        WHEN CARDINALITY(hr.prefer_equipment) > 0
             AND e.equipment = ANY(hr.prefer_equipment)
        THEN 0
        ELSE 1
    END,
    RANDOM()
LIMIT 1;
```

---

## 9. Index Recommendations

### Existing Indexes to Verify

```sql
-- Check existing indexes
SELECT
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename IN (
    'users_plans',
    'user_weekly_schedule',
    'workouts',
    'exercises',
    'users_gym_profile'
)
ORDER BY tablename, indexname;
```

### Recommended Indexes for Mesocycle Renewal

```sql
-- Index for mesocycle status checks (user_id + week + Completed)
CREATE INDEX IF NOT EXISTS idx_user_weekly_schedule_completion
ON user_weekly_schedule (user_id, week, "Completed");

-- Index for workout lookups by user
CREATE INDEX IF NOT EXISTS idx_workouts_user_id
ON workouts (user_id);

-- Composite index for exercise rotation queries
CREATE INDEX IF NOT EXISTS idx_exercises_pattern_role
ON exercises (pattern, role);

-- Index for exercises by main_muscle (for exclusion filters)
CREATE INDEX IF NOT EXISTS idx_exercises_main_muscle
ON exercises (main_muscle);

-- Index for active plans lookup
CREATE INDEX IF NOT EXISTS idx_users_plans_user_status
ON users_plans (user_id, status)
WHERE status = 'active';

-- Index for profile lookup by whatsapp_id (should already be PK)
-- users_gym_profile.whatsapp_id is already PK

-- Index for health status filtering
CREATE INDEX IF NOT EXISTS idx_users_gym_profile_health
ON users_gym_profile (health_status);
```

### Query Performance Analysis

```sql
-- Analyze query plans for key queries
EXPLAIN ANALYZE
SELECT COUNT(*) FILTER (WHERE "Completed" = true)
FROM user_weekly_schedule
WHERE user_id = '00000000-0000-0000-0000-000000000000'
  AND week = 4;

EXPLAIN ANALYZE
SELECT e.exercise_id
FROM exercises e
WHERE e.pattern = 'push_h'
  AND e.role = 'compound'
  AND e.main_muscle NOT IN ('Calfs', 'Glutes');
```

---

## 10. Data Validation Queries

### Pre-Renewal Validation

```sql
-- Validate user is ready for renewal
SELECT
    u.user_id,
    u.full_name,
    CASE WHEN up.plan_id IS NULL THEN 'NO_ACTIVE_PLAN' ELSE 'OK' END as plan_status,
    CASE WHEN ugp.whatsapp_id IS NULL THEN 'NO_PROFILE' ELSE 'OK' END as profile_status,
    CASE WHEN COUNT(w.id) = 0 THEN 'NO_WORKOUTS' ELSE 'OK' END as workout_status,
    up.mesocycle_number,
    up.week_schedule,
    ws.days_per_week,
    (SELECT COUNT(*) FROM user_weekly_schedule
     WHERE user_id = u.user_id AND week = 4 AND "Completed" = true) as week4_completed
FROM users u
LEFT JOIN users_plans up ON u.user_id = up.user_id AND up.status = 'active'
LEFT JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
LEFT JOIN users_gym_profile ugp ON u.full_phone_number::BIGINT = ugp.whatsapp_id
LEFT JOIN workouts w ON u.user_id = w.user_id
WHERE u.user_id = $1
GROUP BY u.user_id, u.full_name, up.plan_id, ugp.whatsapp_id,
         up.mesocycle_number, up.week_schedule, ws.days_per_week;
```

### Post-MANTENER Validation

```sql
-- Verify MANTENER_RUTINA completed successfully
SELECT
    'Schedule cleared' as validation,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status,
    COUNT(*) as count
FROM user_weekly_schedule
WHERE user_id = $1

UNION ALL

SELECT
    'Mesocycle incremented',
    CASE WHEN mesocycle_number >= $2 THEN 'PASS' ELSE 'FAIL' END,
    mesocycle_number
FROM users_plans
WHERE user_id = $1 AND status = 'active'

UNION ALL

SELECT
    'Renewal date updated',
    CASE WHEN last_renewal_date >= NOW() - INTERVAL '1 minute' THEN 'PASS' ELSE 'FAIL' END,
    EXTRACT(EPOCH FROM last_renewal_date)::INT
FROM users_plans
WHERE user_id = $1 AND status = 'active'

UNION ALL

SELECT
    'Workouts preserved',
    CASE WHEN COUNT(*) > 0 THEN 'PASS' ELSE 'FAIL' END,
    COUNT(*)
FROM workouts
WHERE user_id = $1;
```

### Post-CAMBIAR_DIAS Validation

```sql
-- Verify CAMBIAR_DIAS completed successfully
SELECT
    'Workouts deleted' as validation,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM workouts
WHERE user_id = $1

UNION ALL

SELECT
    'Schedule cleared',
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END
FROM user_weekly_schedule
WHERE user_id = $1

UNION ALL

SELECT
    'Week schedule updated to ' || $2,
    CASE WHEN week_schedule = $2 THEN 'PASS' ELSE 'FAIL' END
FROM users_plans
WHERE user_id = $1 AND status = 'active';
```

### Post-ROTAR_EJERCICIOS Validation

```sql
-- Verify exercise rotation completed successfully
WITH rotation_stats AS (
    SELECT
        COUNT(DISTINCT exercise_id) as unique_exercises,
        COUNT(DISTINCT pattern) as patterns_covered,
        COUNT(DISTINCT role) as roles_covered
    FROM workouts w
    JOIN exercises e ON w.exercise_id = e.exercise_id
    WHERE w.user_id = $1
)
SELECT
    'Exercises rotated' as validation,
    CASE WHEN unique_exercises >= 5 THEN 'PASS' ELSE 'CHECK' END as status,
    unique_exercises as value,
    'Should have diverse exercises' as note
FROM rotation_stats

UNION ALL

SELECT
    'Patterns preserved',
    CASE WHEN patterns_covered >= 4 THEN 'PASS' ELSE 'FAIL' END,
    patterns_covered,
    'Should maintain pattern diversity'
FROM rotation_stats

UNION ALL

SELECT
    'Schedule cleared',
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END,
    COUNT(*),
    NULL
FROM user_weekly_schedule
WHERE user_id = $1;
```

### Data Integrity Check

```sql
-- Comprehensive data integrity validation
SELECT
    'Foreign key: workouts->exercises' as check_name,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status,
    COUNT(*) as orphaned_records
FROM workouts w
LEFT JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE e.exercise_id IS NULL

UNION ALL

SELECT
    'Foreign key: workouts->users',
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END,
    COUNT(*)
FROM workouts w
LEFT JOIN users u ON w.user_id = u.user_id
WHERE u.user_id IS NULL

UNION ALL

SELECT
    'Foreign key: users_plans->week_schedules',
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END,
    COUNT(*)
FROM users_plans up
LEFT JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
WHERE ws.schedule_type IS NULL AND up.week_schedule IS NOT NULL

UNION ALL

SELECT
    'Week schedule uses ul_4 (not ua_4)',
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL: Fix ua_4 references' END,
    COUNT(*)
FROM users_plans
WHERE week_schedule = 'ua_4';
```

### Detect ua_4 Bug in Existing Data

```sql
-- Check if any plans incorrectly use ua_4
SELECT
    plan_id,
    user_id,
    week_schedule,
    'Should be ul_4' as fix_note
FROM users_plans
WHERE week_schedule = 'ua_4';

-- Fix command (run only if needed)
-- UPDATE users_plans SET week_schedule = 'ul_4' WHERE week_schedule = 'ua_4';
```

---

## Appendix: Complete Transaction Templates

### Template: Safe Renewal Transaction

```sql
-- Generic safe renewal transaction with rollback on error
DO $$
DECLARE
    v_user_id UUID := $1;
    v_affected_rows INT;
BEGIN
    -- Validate user exists and has active plan
    IF NOT EXISTS (
        SELECT 1 FROM users_plans
        WHERE user_id = v_user_id AND status = 'active'
    ) THEN
        RAISE EXCEPTION 'No active plan found for user %', v_user_id;
    END IF;

    -- Your renewal operations here
    -- ...

    -- Verify operation succeeded
    GET DIAGNOSTICS v_affected_rows = ROW_COUNT;
    IF v_affected_rows = 0 THEN
        RAISE EXCEPTION 'Renewal operation affected 0 rows';
    END IF;

EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error during renewal: %', SQLERRM;
        RAISE;
END $$;
```

### Template: Audit Trail (Optional)

```sql
-- Optional: Create audit table for renewal tracking
CREATE TABLE IF NOT EXISTS mesocycle_renewal_log (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(user_id),
    renewal_type VARCHAR(50) NOT NULL, -- MANTENER, CAMBIAR_DIAS, ROTAR, MODIFICAR
    old_mesocycle INT,
    new_mesocycle INT,
    old_week_schedule TEXT,
    new_week_schedule TEXT,
    exercises_rotated INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Log renewal operation
INSERT INTO mesocycle_renewal_log (
    user_id, renewal_type, old_mesocycle, new_mesocycle,
    old_week_schedule, new_week_schedule, exercises_rotated
) VALUES (
    $1, 'ROTAR_EJERCICIOS', $2, $3, $4, $4, $5
);
```

---

## Summary

This document provides all SQL queries needed for the mesocycle renewal feature:

| Operation | Key Tables | Primary Action |
|-----------|------------|----------------|
| Status Detection | `users_plans`, `user_weekly_schedule`, `week_schedules` | Check week 4 completion |
| MANTENER_RUTINA | `user_weekly_schedule`, `users_plans` | Clear schedule, increment mesocycle |
| CAMBIAR_DIAS | `workouts`, `user_weekly_schedule`, `users_plans` | Delete all, update schedule type |
| ROTAR_EJERCICIOS | `workouts`, `exercises`, `users_plans` | Swap exercises by pattern+role |
| MODIFICAR_PERFIL | `users_gym_profile`, `workouts`, `user_weekly_schedule` | Update profile, delete workouts |

**Critical Note**: The week_schedule mapping uses `ul_4` for 4-day schedules (NOT `ua_4`). Any workflow code using `ua_4` contains a bug and should be corrected.
