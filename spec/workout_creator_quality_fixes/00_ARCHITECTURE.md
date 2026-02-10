# WORKOUT_CREATOR Quality Fixes -- Architecture Specification

**Version:** 1.0
**Date:** 2026-02-09
**Author:** Lead Solutions Architect
**Status:** Draft

---

## Table of Contents

1. [Problem Statement](#1-problem-statement)
2. [High-Level Architecture Diagram](#2-high-level-architecture-diagram)
3. [Fix Inventory and Intervention Points](#3-fix-inventory-and-intervention-points)
4. [Database Schema Changes](#4-database-schema-changes)
5. [n8n Node Modifications](#5-n8n-node-modifications)
6. [Data Flow Diagrams -- Before vs After](#6-data-flow-diagrams----before-vs-after)
7. [AI Agent Prompt Changes](#7-ai-agent-prompt-changes)
8. [Risk Matrix](#8-risk-matrix)
9. [Testing Strategy](#9-testing-strategy)
10. [Implementation Order](#10-implementation-order)

---

## 1. Problem Statement

The WORKOUT_CREATOR workflow generates personalized 4-week workout plans. A systematic review identified 5 quality defects that affect ALL generated routines, regardless of user profile. These defects interact with each other -- for example, misclassified exercises feed into incorrect set_profiles lookups, which then distort the duration calculation used for trimming.

| # | Defect | Root Cause | Impact |
|---|--------|-----------|--------|
| QF-1 | W4 volume inflation | `ValidateWorkoutDuration` trims by time only; W4 has fewer sets per exercise, so total time is lower, so fewer exercises are trimmed -- resulting in W4 having MORE exercises than W1-W3 | Users see a "deload" week with more exercises than their heavy weeks |
| QF-2 | Duplicate exercises | AI Agent can select the same `exercise_id` multiple times for a single day; no dedup guard exists in `Code in JavaScript` (the parser) | Same exercise appears 2-3 times in a day's workout |
| QF-3 | Misclassified exercises | ~104 exercises have incorrect `pattern` vs `main_muscle` mapping (e.g., Abs exercise tagged `push_h`) | Wrong exercises appear in Push/Pull days; volume distribution is skewed |
| QF-4 | No cardio role | ~40 cardio/plyometric exercises (Burpee, Assault Bike, etc.) are `role = 'isolation'` | Cardio exercises receive hypertrophy set/rep parameters (3x12-15 @RIR 1-2) instead of appropriate conditioning parameters |
| QF-5 | Health status not enforced at SQL level | Health restrictions (D=spine, B=lower body) exist only as text warnings in the AI prompt; no SQL-level filter prevents restricted exercises from reaching the AI | AI occasionally ignores health warnings and selects contraindicated exercises |

---

## 2. High-Level Architecture Diagram

### 2.1 Current Pipeline with Fix Intervention Points

```
WORKOUT_CREATOR Pipeline
========================

  input (trigger)
    |
    v
  GetUserProfile ------> Supabase: users_gym_profile
    |
    v
  ProcessUserPreferences (Code)
    |                           +---> health flags (used by QF-5)
    |                           +---> volume_modifier
    |                           +---> priority_muscles_en
    v
  If_Is_Renewal -----> (renewal path, existing)
    |
    v (FALSE / rejoin)
  LoadProfile ---------> Supabase: set_profiles
    |                        |
    |                    [QF-4] Currently no 'cardio' role rows exist.
    |                         Exercises with role='isolation' get
    |                         hypertrophy params instead of conditioning.
    v
  Get_Day_Requirements --> Postgres JOIN: routine_templates +
    |                      template_days + day_requirements
    v
  GetUser / UserExists / CreateUser / CreatePlan / Merge
    |
    v
  Loop Over Items (SplitInBatches) -----> iterates day_requirement rows
    |
    +-------+
    |       |
    v       |
  GetExercisesByPattern ----------> Postgres: SELECT FROM exercises
    |                                  WHERE pattern = X
    |                                  [QF-3] Misclassified exercises
    |                                  returned for wrong patterns.
    |                                  [QF-5] No health-based WHERE
    |                                  clause filters contraindicated
    |                                  exercises.
    v
  AI Agent (Gemini) ----------------> Selects exercises from pool
    |                                  [QF-2] Can pick same exercise_id
    |                                  multiple times per day.
    v
  AI Transform (collect outputs)
    |
    +<------+ (loop back)
    |
    v
  Code in JavaScript (parse AI JSON)
    |     [QF-2] No deduplication guard exists here.
    v
  Get a row (set_profiles, all weeks)
    |
    v
  Code in JavaScript1 (expand W1 -> W1-W4)
    |     [QF-4] Exercises with role='isolation' get wrong
    |            set_profiles for cardio exercises.
    v
  ValidateWorkoutDuration (Code)
    |     [QF-1] Trims by time only. W4 exercises have
    |            fewer sets, so shorter total time, so
    |            fewer get trimmed. W4 ends up with MORE
    |            exercises than W1-W3.
    v
  Create a row ----------> Supabase: INSERT INTO workouts
    |
    v
  NotifyRoutineCreated --> WhatsApp message
    |
    v
  GetWeek1WithExercises -> email pipeline
```

### 2.2 Fix Intervention Map (Summary)

```
+------------------------------------------------------------+
| LAYER         | FIX   | WHERE                              |
|---------------|-------|------------------------------------|
| Database      | QF-3  | UPDATE exercises (pattern/muscle)   |
| Database      | QF-4  | INSERT exercise_role, set_profiles  |
| Database      | QF-4  | UPDATE exercises (role -> cardio)   |
| SQL Query     | QF-5  | GetExercisesByPattern WHERE clause  |
| Code Node     | QF-2  | Code in JavaScript (dedup)         |
| Code Node     | QF-1  | ValidateWorkoutDuration (W4 cap)   |
| AI Prompt     | QF-2  | AI Agent system prompt (reinforce)  |
| AI Prompt     | QF-5  | AI Agent system prompt (reinforce)  |
+------------------------------------------------------------+
```

---

## 3. Fix Inventory and Intervention Points

### 3.1 Fix Dependency Graph

Fixes must be applied in the correct order because some depend on data corrections made by others.

```
QF-3 (fix misclassified exercises)
  |
  +---> QF-4 (add cardio role)  <-- depends on QF-3 being clean first
  |       |
  |       +---> QF-1 (W4 volume cap)  <-- depends on roles being correct
  |
  +---> QF-5 (health SQL filter)  <-- depends on patterns being correct

QF-2 (dedup) is independent -- can be applied at any point
```

**Recommended order:** QF-3 --> QF-4 --> QF-5 --> QF-2 --> QF-1

---

## 4. Database Schema Changes

### 4.1 QF-4: New `cardio` Role

#### 4.1.1 Insert into `exercise_role`

```sql
-- Add new 'cardio' role to the exercise_role reference table
INSERT INTO exercise_role (role, detail)
VALUES (
  'cardio',
  'Cardiovascular and plyometric exercises. Includes HIIT movements, '
  'conditioning drills, and high-intensity bodyweight exercises. '
  'Programmed with time-based or low-rep explosive parameters.'
);
```

#### 4.1.2 New `set_profiles` Rows for `cardio`

The `cardio` role needs loading parameters for every `goal` x `level` x `week` combination that currently exists for `compound`, `core`, and `isolation`. Cardio exercises use time-based or low-rep conditioning parameters rather than hypertrophy parameters.

**Design rationale:** Cardio exercises in a strength program serve as finishers or conditioning blocks. They should use shorter rest, moderate duration (typically 30-60s work periods), and should deload in W4 by reducing rounds/sets, not by reducing intensity.

```sql
-- Cardio set_profiles for "Ganar masa muscular" / all levels
-- Pattern: 2-3 sets, 30-45s work (expressed as reps for the DB schema),
-- minimal RIR concept (replaced with "RPE 7-8" in notes), short rest

-- === PRINCIPIANTE ===
INSERT INTO set_profiles (goal, level, week, role, sets, reps, rir, rest_sec, tempo)
VALUES
  ('Ganar masa muscular', 'Principiante', 1, 'cardio', '2', '30-45s', 'N/A', 60, 'continuous'),
  ('Ganar masa muscular', 'Principiante', 2, 'cardio', '3', '30-45s', 'N/A', 60, 'continuous'),
  ('Ganar masa muscular', 'Principiante', 3, 'cardio', '3', '40-60s', 'N/A', 45, 'continuous'),
  ('Ganar masa muscular', 'Principiante', 4, 'cardio', '2', '20-30s', 'N/A', 90, 'continuous');

-- === INTERMEDIO ===
INSERT INTO set_profiles (goal, level, week, role, sets, reps, rir, rest_sec, tempo)
VALUES
  ('Ganar masa muscular', 'Intermedio', 1, 'cardio', '3', '30-45s', 'N/A', 45, 'continuous'),
  ('Ganar masa muscular', 'Intermedio', 2, 'cardio', '3', '40-60s', 'N/A', 45, 'continuous'),
  ('Ganar masa muscular', 'Intermedio', 3, 'cardio', '4', '40-60s', 'N/A', 30, 'continuous'),
  ('Ganar masa muscular', 'Intermedio', 4, 'cardio', '2', '20-30s', 'N/A', 60, 'continuous');

-- === AVANZADO ===
INSERT INTO set_profiles (goal, level, week, role, sets, reps, rir, rest_sec, tempo)
VALUES
  ('Ganar masa muscular', 'Avanzado', 1, 'cardio', '3', '40-60s', 'N/A', 30, 'continuous'),
  ('Ganar masa muscular', 'Avanzado', 2, 'cardio', '4', '40-60s', 'N/A', 30, 'continuous'),
  ('Ganar masa muscular', 'Avanzado', 3, 'cardio', '4', '45-60s', 'N/A', 20, 'continuous'),
  ('Ganar masa muscular', 'Avanzado', 4, 'cardio', '2', '30s', 'N/A', 60, 'continuous');

-- Repeat pattern for other goals:
-- "Bajar grasa"           -> higher sets, shorter rest (more conditioning focus)
-- "Mejorar fuerza"        -> lower sets, longer rest (less cardio emphasis)
-- "Mejorar resistencia"   -> higher sets, longer work periods
-- "Salud general / recomposicion corporal" -> moderate (similar to "Ganar masa muscular")
```

**Note:** The exact INSERT statements for the remaining 4 goals follow the same pattern. Each goal/level combination requires 4 rows (W1-W4). Total new rows: 5 goals x 3 levels x 4 weeks = **60 new set_profiles rows**.

#### 4.1.3 Update ~40 Cardio Exercises (role change)

```sql
-- Reclassify cardio/plyometric exercises from 'isolation' to 'cardio'
-- These exercises currently have role='isolation' but are conditioning movements

UPDATE exercises
SET role = 'cardio'
WHERE exercise_id IN (
  -- High-intensity cardio / plyometrics
  -- The full list of ~40 exercise_ids will be determined by a discovery query:
  --
  -- SELECT exercise_id, spanish_name, pattern, role, main_muscle
  -- FROM exercises
  -- WHERE spanish_name ILIKE ANY(ARRAY[
  --   '%burpee%', '%assault%bike%', '%salto%', '%box%jump%',
  --   '%mountain%climber%', '%jumping%jack%', '%sprint%',
  --   '%battle%rope%', '%remo%ergometro%', '%eliptica%',
  --   '%bicicleta%estacionaria%', '%cuerda%', '%ski%erg%',
  --   '%bear%crawl%', '%shuttle%run%', '%sled%'
  -- ])
  -- AND role = 'isolation'
  -- ORDER BY spanish_name;
  --
  -- Placeholder: actual IDs to be filled after running discovery query
  'EXERCISE_ID_1',
  'EXERCISE_ID_2'
  -- ... (approximately 40 exercises)
);
```

**Discovery query to identify candidates:**

```sql
-- Run this query to find all exercises that should be reclassified as cardio
SELECT exercise_id, spanish_name, pattern, role, main_muscle, equipment
FROM exercises
WHERE (
  -- Name-based detection (Spanish exercise names)
  spanish_name ILIKE ANY(ARRAY[
    '%burpee%', '%assault%', '%saltar%', '%salto%box%',
    '%mountain climber%', '%jumping jack%', '%sprint%',
    '%battle rope%', '%cuerdas de batalla%',
    '%remo ergometro%', '%ergometro%',
    '%bicicleta estacionaria%', '%eliptica%',
    '%saltar la cuerda%', '%cuerda para saltar%',
    '%ski erg%', '%bear crawl%', '%shuttle%',
    '%sled%push%', '%sled%pull%', '%trineo%',
    '%plyo%', '%pliometri%'
  ])
  OR
  -- Equipment-based detection
  equipment IN ('assault_bike', 'rower', 'ski_erg', 'battle_rope')
)
AND role = 'isolation'
ORDER BY pattern, spanish_name;
```

### 4.2 QF-3: Fix ~30 High-Impact Misclassified Exercises

These exercises have incorrect `pattern` vs `main_muscle` mapping. Priority is given to push/pull pattern exercises because they directly affect Push Day and Pull Day composition.

#### 4.2.1 Pattern Corrections (wrong pattern)

```sql
-- Fix exercises where the pattern does not match the movement
-- Example: An Abs exercise incorrectly tagged as push_h

-- Template for each correction:
-- UPDATE exercises
-- SET pattern = '<correct_pattern>'
-- WHERE exercise_id = '<exercise_id>';
-- -- Reason: <spanish_name> targets <main_muscle>, should be <correct_pattern>

-- Discovery query to find mismatches:
SELECT e.exercise_id, e.spanish_name, e.pattern, e.main_muscle,
       e.secondary_muscles, e.role, e.equipment
FROM exercises e
WHERE (
  -- Abs exercises tagged as push
  (e.main_muscle = 'Abs' AND e.pattern IN ('push_h', 'push_v'))
  OR
  -- Abs exercises tagged as pull
  (e.main_muscle = 'Abs' AND e.pattern IN ('pull_h', 'pull_v'))
  OR
  -- Back exercises tagged as push
  (e.main_muscle IN ('Back', 'Lats') AND e.pattern IN ('push_h', 'push_v'))
  OR
  -- Chest exercises tagged as pull
  (e.main_muscle = 'Chest' AND e.pattern IN ('pull_h', 'pull_v'))
  OR
  -- Shoulder exercises with wrong push direction
  (e.main_muscle = 'Shoulders' AND e.pattern = 'push_h'
   AND e.spanish_name ILIKE '%press%militar%')
  OR
  -- Biceps in push patterns
  (e.main_muscle = 'Biceps' AND e.pattern IN ('push_h', 'push_v'))
  OR
  -- Triceps in pull patterns
  (e.main_muscle = 'Triceps' AND e.pattern IN ('pull_h', 'pull_v'))
)
ORDER BY e.pattern, e.main_muscle, e.spanish_name;
```

**Expected corrections (~30 exercises):**

| Current Pattern | Main Muscle | Correct Pattern | Rationale |
|-----------------|-------------|-----------------|-----------|
| `push_h` | Abs | `core` | Ab exercises are core movements |
| `push_v` | Abs | `core` | Ab exercises are core movements |
| `pull_h` | Abs | `core` | Ab exercises are core movements |
| `push_h` | Back/Lats | `pull_h` or `pull_v` | Back exercises are pulling movements |
| `pull_h` | Chest | `push_h` | Chest exercises are pushing movements |
| `push_h` | Shoulders (overhead) | `push_v` | Overhead press is vertical push |
| `push_h` | Biceps | `arm` | Bicep curls are arm isolation |
| `pull_h` | Triceps | `arm` | Tricep extensions are arm isolation |

#### 4.2.2 Main Muscle Corrections (wrong muscle attribution)

```sql
-- Fix exercises where main_muscle is incorrect for the movement
-- Discovery query:
SELECT exercise_id, spanish_name, pattern, main_muscle, role
FROM exercises
WHERE (
  -- Exercises in 'core' pattern but main_muscle is not Abs/Core/Obliques
  (pattern = 'core' AND main_muscle NOT IN ('Abs', 'Core', 'Obliques', 'Lower back'))
  OR
  -- Exercises in 'arm' pattern but main_muscle is not Biceps/Triceps/Forearms
  (pattern = 'arm' AND main_muscle NOT IN ('Biceps', 'Triceps', 'Forearms'))
)
ORDER BY pattern, main_muscle;
```

### 4.3 Complete Migration Script Structure

All database changes should be applied as a single Supabase migration:

```sql
-- Migration: fix_exercise_quality_and_add_cardio_role
-- Description: QF-3 (misclassified exercises), QF-4 (cardio role + set_profiles)

BEGIN;

-- =============================================
-- SECTION 1: QF-3 - Fix misclassified exercises
-- =============================================

-- 1a. Pattern corrections
UPDATE exercises SET pattern = 'core'   WHERE exercise_id = '...' ; -- <spanish_name>: Abs, was push_h
UPDATE exercises SET pattern = 'pull_h' WHERE exercise_id = '...' ; -- <spanish_name>: Back, was push_h
-- ... (~30 rows)

-- 1b. Main muscle corrections (if any)
UPDATE exercises SET main_muscle = '...' WHERE exercise_id = '...' ;
-- ... (as discovered)

-- =============================================
-- SECTION 2: QF-4 - Add cardio role
-- =============================================

-- 2a. New role
INSERT INTO exercise_role (role, detail)
VALUES ('cardio', 'Cardiovascular and plyometric conditioning exercises');

-- 2b. New set_profiles (60 rows)
INSERT INTO set_profiles (goal, level, week, role, sets, reps, rir, rest_sec, tempo)
VALUES
  -- ... (all 60 rows as defined in 4.1.2)
;

-- 2c. Reclassify cardio exercises
UPDATE exercises SET role = 'cardio' WHERE exercise_id IN (...);

COMMIT;
```

---

## 5. n8n Node Modifications

All modifications target nodes in `n8n/running_flows/WORKOUT_CREATOR.json`.

### 5.1 QF-5: `GetExercisesByPattern` -- Health Status SQL Filter

**Node ID:** `6239f1ba-017e-4d4b-9c20-9dc6c093183f`
**Node type:** `n8n-nodes-base.postgres` (typeVersion 2.6)

#### Current SQL (inline expression)

```javascript
// Simplified view of the current dynamic SQL builder
let query = `
  SELECT exercise_id, spanish_name, pattern, role,
         main_muscle, secondary_muscles, level, link, equipment
  FROM exercises
  WHERE pattern = '${pattern}'
`;

if (isHome && equipmentSql) {
  query += ` AND equipment IN (${equipmentSql})`;
}

query += `
  ORDER BY
    CASE WHEN level = '${level}' THEN 0 ELSE 1 END,
    role,
    spanish_name
`;
```

#### Modified SQL (with health filter)

```javascript
// New variables from ProcessUserPreferences
const health = $items('ProcessUserPreferences')[0].json.processed.health;

let query = `
  SELECT exercise_id, spanish_name, pattern, role,
         main_muscle, secondary_muscles, level, link, equipment
  FROM exercises
  WHERE pattern = '${pattern}'
`;

// --- EXISTING: Home equipment filter ---
if (isHome && equipmentSql) {
  query += ` AND equipment IN (${equipmentSql})`;
}

// --- NEW: Health status exclusion filters (QF-5) ---
if (health.avoid_lower_body_impact) {
  // Health B: Exclude high-impact lower body exercises
  query += ` AND NOT (
    main_muscle IN ('Quads', 'Hamstrings', 'Glutes', 'Calfs')
    AND equipment IN ('barbell', 'smith_machine')
    AND (spanish_name ILIKE '%sentadilla%' OR spanish_name ILIKE '%peso muerto%'
         OR spanish_name ILIKE '%zancada%' OR spanish_name ILIKE '%salto%'
         OR spanish_name ILIKE '%prensa%pierna%')
  )`;
}

if (health.avoid_upper_body_overhead) {
  // Health C: Exclude overhead pressing movements
  query += ` AND NOT (
    pattern IN ('push_v')
    AND (spanish_name ILIKE '%press%militar%'
         OR spanish_name ILIKE '%press%hombro%'
         OR spanish_name ILIKE '%overhead%'
         OR spanish_name ILIKE '%tras nuca%'
         OR spanish_name ILIKE '%snatch%')
  )`;
}

if (health.avoid_spinal_loading) {
  // Health D: Exclude heavy axial loading exercises
  query += ` AND NOT (
    (spanish_name ILIKE '%peso muerto%'
     OR spanish_name ILIKE '%sentadilla%barra%'
     OR spanish_name ILIKE '%good morning%'
     OR spanish_name ILIKE '%rack pull%'
     OR spanish_name ILIKE '%clean%'
     OR spanish_name ILIKE '%snatch%')
    AND equipment IN ('barbell', 'smith_machine', 'trap_bar')
  )`;
}

if (health.special_condition) {
  // Health E: Only allow machine and bodyweight exercises
  query += ` AND equipment IN ('machine', 'bodyweight', 'cable')`;
}

query += `
  ORDER BY
    CASE WHEN level = '${level}' THEN 0 ELSE 1 END,
    role,
    spanish_name
`;
```

#### Interface Contract

**Input (from ProcessUserPreferences):**

```typescript
interface HealthFlags {
  has_restrictions: boolean;
  avoid_lower_body_impact: boolean;   // Health B
  avoid_upper_body_overhead: boolean; // Health C
  avoid_spinal_loading: boolean;      // Health D
  special_condition: boolean;         // Health E
}
```

**Output:** Same schema as current (array of exercise rows), but with health-contraindicated exercises excluded before they reach the AI Agent.

**Fallback behavior:** If all exercises for a pattern are excluded, the query returns an empty result set. The AI Agent will receive an empty `AVAILABLE_EXERCISES` array and should output an empty JSON array `[]` for that day_requirement. This is handled gracefully by existing downstream nodes.

### 5.2 QF-2: `Code in JavaScript` -- Exercise Deduplication

**Node ID:** `e4feabe9-660d-489d-ae4a-d489884e86cf`
**Node type:** `n8n-nodes-base.code` (typeVersion 2)

#### Current Logic

```javascript
// Current: Parse AI JSON outputs, collect exercises, no dedup
let allExercises = [];
for (const item of $input.all()) {
  const unified = item.json.unifiedArray;
  if (Array.isArray(unified)) {
    for (const entry of unified) {
      try {
        let outputStr = entry.output;
        outputStr = outputStr
          .replace(/^```json\s*/i, '')
          .replace(/^```\s*/i, '')
          .replace(/\s*```$/i, '')
          .trim();
        const exercises = JSON.parse(outputStr);
        allExercises.push(...exercises);
      } catch (error) {
        continue;
      }
    }
  }
}
return allExercises.map(exercise => ({ json: exercise }));
```

#### Modified Logic (with deduplication)

```javascript
let allExercises = [];
for (const item of $input.all()) {
  const unified = item.json.unifiedArray;
  if (Array.isArray(unified)) {
    for (const entry of unified) {
      try {
        let outputStr = entry.output;
        outputStr = outputStr
          .replace(/^```json\s*/i, '')
          .replace(/^```\s*/i, '')
          .replace(/\s*```$/i, '')
          .trim();
        const exercises = JSON.parse(outputStr);
        allExercises.push(...exercises);
      } catch (error) {
        continue;
      }
    }
  }
}

// === NEW: QF-2 Deduplication Guard ===
// Remove duplicate exercise_ids within the same day_name.
// Keep the FIRST occurrence (which the AI ranked higher).
const dedupedExercises = [];
const seenPerDay = {};  // { day_name: Set<exercise_id> }

for (const ex of allExercises) {
  const dayKey = ex.day_name;
  if (!seenPerDay[dayKey]) {
    seenPerDay[dayKey] = new Set();
  }
  if (seenPerDay[dayKey].has(ex.exercise_id)) {
    // Duplicate detected -- skip this occurrence
    console.log(`DEDUP: Removed duplicate exercise_id=${ex.exercise_id} on day=${dayKey}`);
    continue;
  }
  seenPerDay[dayKey].add(ex.exercise_id);
  dedupedExercises.push(ex);
}

console.log(`DEDUP SUMMARY: ${allExercises.length} input -> ${dedupedExercises.length} output (${allExercises.length - dedupedExercises.length} duplicates removed)`);

return dedupedExercises.map(exercise => ({ json: exercise }));
```

#### Interface Contract

**Input:** Same as current (array of AI JSON output items).

**Output:** Same schema as current (array of exercise objects), but guaranteed unique `exercise_id` per `day_name`.

**Dedup strategy:** Keep the first occurrence of each `exercise_id` per day. The rationale is that the AI Agent lists exercises in priority order, so the first mention represents the AI's primary selection.

### 5.3 QF-1: `ValidateWorkoutDuration` -- W4 Exercise Count Cap

**Node ID:** `db095e1f-1b73-425f-8841-fbeb28073419`
**Node type:** `n8n-nodes-base.code` (typeVersion 2)

#### Problem Analysis

The current `ValidateWorkoutDuration` node processes all 4 weeks of exercises (W1-W4). For each day in each week, it checks whether the total estimated duration exceeds the user's session target. If it does, it reduces sets (Phase 1) and then removes exercises (Phase 2).

The bug: W4 exercises have fewer sets per exercise (e.g., 2 sets instead of 4), which means their total estimated time is shorter. Because the trim threshold is purely time-based, W4 days rarely exceed the target -- so no exercises get trimmed from W4. But W1-W3 days (with 3-4 sets) often exceed the target, causing exercises to be trimmed. Net result: W4 has MORE exercises than W1-W3.

```
Example (Push Day, 70-minute target):
  W1: 7 exercises x 4 sets = estimated 82 min -> trimmed to 5 exercises
  W4: 7 exercises x 2 sets = estimated 48 min -> NO trimming -> stays at 7 exercises

User sees: W1 Push = 5 exercises, W4 Push = 7 exercises (WRONG)
```

#### Solution: W4 Exercise Count Cap

After all weeks are processed, enforce that W4 never has more exercises than the corresponding W1 day. This is an exercise-count cap, not a time-based trim.

#### Modified Logic (addition to existing code)

The following block is inserted **after** the main processing loop and **before** the final return statement in `ValidateWorkoutDuration`:

```javascript
// === NEW: QF-1 W4 Exercise Count Cap ===
// Ensure W4 (deload) never has more exercises per day than W1.
// This prevents the time-based trim from creating inverted volume.

// Count exercises per day for W1
const w1CountByDay = {};
const w4ExercisesByDay = {};

for (const w of processedWorkouts) {
  if (w.week_number === 1) {
    w1CountByDay[w.day_name] = (w1CountByDay[w.day_name] || 0) + 1;
  }
}

// Group W4 exercises by day for potential trimming
for (let i = 0; i < processedWorkouts.length; i++) {
  const w = processedWorkouts[i];
  if (w.week_number === 4) {
    if (!w4ExercisesByDay[w.day_name]) {
      w4ExercisesByDay[w.day_name] = [];
    }
    w4ExercisesByDay[w.day_name].push({ index: i, exercise: w });
  }
}

// For each W4 day, cap exercise count to W1 count
const indicesToRemove = new Set();

for (const [dayName, w4Exercises] of Object.entries(w4ExercisesByDay)) {
  const w1Count = w1CountByDay[dayName] || w4Exercises.length; // fallback: no cap

  if (w4Exercises.length > w1Count) {
    const excess = w4Exercises.length - w1Count;

    // Sort by priority score (ascending) to remove lowest-priority first
    const ranked = w4Exercises
      .map(item => ({
        ...item,
        priorityScore: getPriorityScore(item.exercise, priorityMuscles, exerciseLookup)
      }))
      .sort((a, b) => a.priorityScore - b.priorityScore);

    // Mark the lowest-priority exercises for removal
    for (let r = 0; r < excess; r++) {
      indicesToRemove.add(ranked[r].index);
      validationLog.push({
        week: 'W4',
        day: dayName,
        action: 'W4_CAP_REMOVE',
        exercise_id: ranked[r].exercise.exercise_id,
        priority_score: ranked[r].priorityScore,
        w1_count: w1Count,
        w4_count_before: w4Exercises.length,
        w4_count_after: w1Count
      });
    }

    console.log(`W4 CAP: ${dayName} trimmed from ${w4Exercises.length} to ${w1Count} exercises`);
  }
}

// Filter out removed exercises (iterate in reverse to preserve indices)
const finalWorkouts = processedWorkouts.filter((_, idx) => !indicesToRemove.has(idx));

// Replace processedWorkouts for return
// (modify the return statement to use finalWorkouts instead of processedWorkouts)
```

#### Interface Contract

**Input:** Same as current (all 4 weeks of workouts from `Code in JavaScript1`).

**Output:** Same schema as current, but with the guarantee:

```
For every day_name D:
  count(exercises WHERE week_number=4 AND day_name=D)
  <=
  count(exercises WHERE week_number=1 AND day_name=D)
```

**Edge cases:**
- If W1 has 0 exercises for a day (should not happen), W4 is not capped for that day.
- If W1 and W4 have the same count, no action is taken.
- The cap uses the same `getPriorityScore` function already defined in the node to decide which W4 exercises to remove.

---

## 6. Data Flow Diagrams -- Before vs After

### 6.1 QF-1: W4 Volume Inflation

```
BEFORE (time-only trimming):
+-------------------------------------------------------------------+
|                                                                   |
|  Code in JavaScript1 (expand W1 -> W1-W4)                       |
|  Output: 7 exercises x 4 weeks = 28 exercises per day            |
|                                                                   |
|      +------- all 28 exercises -------+                           |
|      |                                |                           |
|      v                                v                           |
|  ValidateWorkoutDuration              |                           |
|  Target: 70 min                       |                           |
|                                       |                           |
|  W1: 7 ex x 4 sets = 82 min          |                           |
|  OVER TARGET -> trim to 5 ex         |                           |
|                                       |                           |
|  W4: 7 ex x 2 sets = 48 min          |                           |
|  UNDER TARGET -> NO TRIM (7 ex)      |                           |
|                                       |                           |
|  Result: W1=5, W2=5, W3=5, W4=7     |  <-- BUG: W4 > W1        |
|                                       |                           |
+-------------------------------------------------------------------+

AFTER (time trim + W4 count cap):
+-------------------------------------------------------------------+
|                                                                   |
|  Code in JavaScript1 (expand W1 -> W1-W4)                       |
|  Output: 7 exercises x 4 weeks = 28 exercises per day            |
|                                                                   |
|      +------- all 28 exercises -------+                           |
|      |                                |                           |
|      v                                v                           |
|  ValidateWorkoutDuration              |                           |
|  Target: 70 min                       |                           |
|                                       |                           |
|  PHASE A (existing): Time-based trim  |                           |
|  W1: 7 -> 5 exercises                |                           |
|  W4: 7 -> 7 exercises (under target) |                           |
|                                       |                           |
|  PHASE B (NEW): W4 count cap          |                           |
|  W4 has 7, W1 has 5                   |                           |
|  Cap W4 to 5: remove 2 lowest-       |                           |
|  priority exercises                   |                           |
|                                       |                           |
|  Result: W1=5, W2=5, W3=5, W4=5     |  <-- FIXED               |
|                                       |                           |
+-------------------------------------------------------------------+
```

### 6.2 QF-2: Duplicate Exercise Prevention

```
BEFORE (no dedup):
+-------------------------------------------------------------------+
|                                                                   |
|  AI Agent output for Push Day:                                    |
|  [                                                                |
|    { exercise_id: "bench_press_bb", role: "compound", ... },      |
|    { exercise_id: "incline_db_press", role: "compound", ... },    |
|    { exercise_id: "bench_press_bb", role: "compound", ... },  <-- DUP
|    { exercise_id: "cable_fly",       role: "isolation", ... },    |
|  ]                                                                |
|                                                                   |
|  Code in JavaScript (parser):                                     |
|  - Parses JSON, no dedup check                                   |
|  - Output: 4 exercises (including duplicate)                      |
|                                                                   |
|  Code in JavaScript1 (expand):                                    |
|  - Expands to 4 weeks: 4 x 4 = 16 rows                          |
|  - bench_press_bb appears TWICE per week = 8 total rows           |
|                                                                   |
+-------------------------------------------------------------------+

AFTER (dedup in parser):
+-------------------------------------------------------------------+
|                                                                   |
|  AI Agent output for Push Day:                                    |
|  [same as above -- AI can still produce duplicates]               |
|                                                                   |
|  Code in JavaScript (parser + DEDUP):                             |
|  - Parses JSON                                                    |
|  - NEW: Track seen exercise_ids per day_name                      |
|  - bench_press_bb seen second time on "Push" -> SKIP              |
|  - Log: "DEDUP: Removed duplicate exercise_id=bench_press_bb"     |
|  - Output: 3 exercises (duplicate removed)                        |
|                                                                   |
|  Code in JavaScript1 (expand):                                    |
|  - Expands to 4 weeks: 3 x 4 = 12 rows                          |
|  - Each exercise appears ONCE per week                            |
|                                                                   |
+-------------------------------------------------------------------+
```

### 6.3 QF-3: Misclassified Exercise Pattern

```
BEFORE (wrong pattern):
+-------------------------------------------------------------------+
|                                                                   |
|  day_requirement: pattern = 'push_h', day = 'Push'               |
|                                                                   |
|  GetExercisesByPattern:                                           |
|  SELECT ... FROM exercises WHERE pattern = 'push_h'              |
|                                                                   |
|  Results include:                                                 |
|  - Bench Press (Chest)       <-- CORRECT                         |
|  - Dumbbell Fly (Chest)      <-- CORRECT                         |
|  - Ab Crunch Machine (Abs)   <-- WRONG: is core, not push_h      |
|  - Cable Crunch (Abs)        <-- WRONG: is core, not push_h      |
|                                                                   |
|  AI Agent receives Abs exercises in Push pool                     |
|  -> May select Ab exercises for Push day                          |
|                                                                   |
+-------------------------------------------------------------------+

AFTER (pattern fixed in DB):
+-------------------------------------------------------------------+
|                                                                   |
|  day_requirement: pattern = 'push_h', day = 'Push'               |
|                                                                   |
|  GetExercisesByPattern:                                           |
|  SELECT ... FROM exercises WHERE pattern = 'push_h'              |
|                                                                   |
|  Results ONLY include push_h exercises:                           |
|  - Bench Press (Chest)                                            |
|  - Dumbbell Fly (Chest)                                           |
|  - Machine Chest Press (Chest)                                    |
|                                                                   |
|  Ab exercises now correctly in pattern = 'core'                   |
|  -> Only appear when day_requirement.pattern = 'core'             |
|                                                                   |
+-------------------------------------------------------------------+
```

### 6.4 QF-4: Cardio Role Set/Rep Parameters

```
BEFORE (cardio as isolation):
+-------------------------------------------------------------------+
|                                                                   |
|  Exercise: Burpee                                                 |
|  role = 'isolation' (WRONG)                                       |
|                                                                   |
|  Code in JavaScript1 (expand W1 -> W4):                          |
|  Looks up set_profiles WHERE role = 'isolation'                   |
|                                                                   |
|  W1 profile match: { sets: 3, reps: "12-15", rir: "1-2",        |
|                       rest_sec: 75, tempo: "2-0-1-0" }           |
|                                                                   |
|  Result: Burpee programmed as 3x12-15 @RIR 1-2, 75s rest        |
|  -> Inappropriate for a conditioning movement                     |
|                                                                   |
+-------------------------------------------------------------------+

AFTER (cardio role with proper parameters):
+-------------------------------------------------------------------+
|                                                                   |
|  Exercise: Burpee                                                 |
|  role = 'cardio' (FIXED)                                          |
|                                                                   |
|  Code in JavaScript1 (expand W1 -> W4):                          |
|  Looks up set_profiles WHERE role = 'cardio'                      |
|                                                                   |
|  W1 profile match: { sets: 3, reps: "30-45s", rir: "N/A",       |
|                       rest_sec: 45, tempo: "continuous" }         |
|                                                                   |
|  Result: Burpee programmed as 3 sets x 30-45s, 45s rest          |
|  -> Appropriate conditioning parameters                           |
|                                                                   |
+-------------------------------------------------------------------+
```

### 6.5 QF-5: Health Status SQL Enforcement

```
BEFORE (prompt-only warning):
+-------------------------------------------------------------------+
|                                                                   |
|  User profile: health_status = 'D' (spine issues)                |
|                                                                   |
|  ProcessUserPreferences:                                          |
|  health.avoid_spinal_loading = true                               |
|                                                                   |
|  GetExercisesByPattern (pattern = 'hinge'):                       |
|  SELECT ... FROM exercises WHERE pattern = 'hinge'               |
|  -> Returns ALL hinge exercises including:                        |
|     - Peso Muerto con Barra (Barbell Deadlift)                   |
|     - Romanian Deadlift                                           |
|     - Good Morning                                                |
|                                                                   |
|  AI Agent system prompt includes:                                 |
|  "CUIDADO ESPALDA: Evitar carga axial pesada"                    |
|  -> AI MAY ignore this warning and select Barbell Deadlift        |
|                                                                   |
+-------------------------------------------------------------------+

AFTER (SQL filter + prompt warning):
+-------------------------------------------------------------------+
|                                                                   |
|  User profile: health_status = 'D' (spine issues)                |
|                                                                   |
|  ProcessUserPreferences:                                          |
|  health.avoid_spinal_loading = true                               |
|                                                                   |
|  GetExercisesByPattern (pattern = 'hinge'):                       |
|  SELECT ... FROM exercises WHERE pattern = 'hinge'               |
|  AND NOT (                                                        |
|    spanish_name ILIKE '%peso muerto%'                             |
|    AND equipment IN ('barbell', 'smith_machine', 'trap_bar')      |
|  )                                                                |
|  -> Returns ONLY safe hinge exercises:                            |
|     - Hip Thrust con Mancuerna                                    |
|     - Cable Pull-Through                                          |
|     - Hiperextension en Banco Romano                              |
|                                                                   |
|  AI Agent system prompt still includes warning (defense in depth) |
|  -> AI physically cannot select Barbell Deadlift (not in pool)    |
|                                                                   |
+-------------------------------------------------------------------+
```

---

## 7. AI Agent Prompt Changes

### 7.1 QF-2: Anti-Duplication Rule

Add to the `REGLAS DE ORO` section in the AI Agent system prompt:

```
## REGLAS DE ORO
- PROHIBIDO inventar ejercicios. Solo usar AVAILABLE_EXERCISES.
- PROHIBIDO ignorar exclusiones de salud o musculos no deseados.
- OBLIGATORIO personalizar segun el perfil completo.
- OBLIGATORIO compensar gaps si el ambiente es HOME (ver tabla de compensacion).
- PROHIBIDO seleccionar ejercicios que requieran equipamiento no disponible.
- **PROHIBIDO repetir exercise_id.** Cada exercise_id debe aparecer MAXIMO UNA VEZ en el JSON de salida. Si necesitas mas volumen para un patron, selecciona un ejercicio DIFERENTE con el mismo patron/musculo.
```

### 7.2 QF-5: Health Exclusion Reinforcement

The existing health warnings in the AI prompt are already present (emoji warnings). No additional prompt changes are needed for QF-5 because the SQL filter in `GetExercisesByPattern` provides a hard gate. The prompt warnings serve as documentation for the AI, not as the enforcement mechanism.

### 7.3 QF-4: Cardio Role Handling

Add to the `PASO 3: Aplicar Carga` section:

```
### PASO 3: Aplicar Carga
Usar datos de CARGA SEMANA cruzando `role` con el perfil de carga.
- Si `volume_modifier` < 1.0: Reducir 1 serie en ejercicios isolation
- Si `volume_modifier` > 1.0: Agregar 1 serie en ejercicios compound
- **Para ejercicios con `role` = "cardio":** Usar exactamente los valores de CARGA SEMANA para role=cardio. NO aplicar modificador de volumen. Los campos `reps` para cardio representan DURACION (ej: "30-45s"), no repeticiones.
```

---

## 8. Risk Matrix

| # | Risk | Probability | Impact | Mitigation |
|---|------|-------------|--------|------------|
| R1 | QF-3 UPDATE hits wrong exercises (false positive in discovery query) | Medium | High -- exercises disappear from expected patterns | Run discovery query first, review results manually. Apply UPDATEs one by one with spanish_name comments. Include RETURNING clause to verify. |
| R2 | QF-4 set_profiles for cardio have wrong time parameters | Low | Medium -- cardio exercises get unrealistic programming | Review with domain expert. Start with conservative parameters (2-3 sets, 30-45s) and adjust based on user feedback. |
| R3 | QF-5 health SQL filter too aggressive (excludes too many exercises, leaves empty pool) | Medium | High -- day_requirement has zero available exercises | Add `MIN_EXERCISES_THRESHOLD` check: if query returns < 3 exercises, fall back to unfiltered query with only the AI prompt warning. Log the fallback. |
| R4 | QF-2 dedup removes an exercise the AI intentionally repeated for volume reasons | Low | Low -- one fewer exercise per day, caught by existing MIN_EXERCISES guard | This is the correct behavior. If the AI needs more volume, it should pick a DIFFERENT exercise with the same pattern. Dedup log allows auditing. |
| R5 | QF-1 W4 cap removes exercises that are important for the deload protocol | Low | Medium -- W4 session may be too short | Cap only removes excess exercises. W4 will have EXACTLY the same exercise count as W1, which is the intended deload structure. |
| R6 | QF-4 `Code in JavaScript1` fails to find set_profile for `role='cardio'` | Medium | High -- exercises get `notes: "No match"` instead of parameters | Ensure set_profiles INSERT covers ALL goal/level combinations. Add defensive fallback in Code node: if no profile found for `cardio`, use `isolation` profile as fallback. |
| R7 | Multiple fixes interacting: a cardio exercise was also misclassified, both QF-3 and QF-4 update it | Low | Low -- last UPDATE wins | Apply QF-3 first (fix pattern), then QF-4 (fix role). Use explicit exercise_id lists, not pattern-based WHERE clauses. |

---

## 9. Testing Strategy

### 9.1 Database Validation Queries

Run these queries AFTER applying the migration to verify data integrity:

```sql
-- QF-3: Verify no remaining pattern/muscle mismatches
SELECT exercise_id, spanish_name, pattern, main_muscle
FROM exercises
WHERE (
  (main_muscle = 'Abs' AND pattern NOT IN ('core'))
  OR (main_muscle IN ('Back', 'Lats') AND pattern IN ('push_h', 'push_v'))
  OR (main_muscle = 'Chest' AND pattern IN ('pull_h', 'pull_v'))
  OR (main_muscle = 'Biceps' AND pattern IN ('push_h', 'push_v'))
  OR (main_muscle = 'Triceps' AND pattern IN ('pull_h', 'pull_v'))
)
ORDER BY pattern, main_muscle;
-- Expected: 0 rows (or only known exceptions with documented rationale)

-- QF-4: Verify cardio role exists and has set_profiles
SELECT role, detail FROM exercise_role WHERE role = 'cardio';
-- Expected: 1 row

SELECT goal, level, week, sets, reps, rir, rest_sec, tempo
FROM set_profiles
WHERE role = 'cardio'
ORDER BY goal, level, week;
-- Expected: 60 rows (5 goals x 3 levels x 4 weeks)

SELECT COUNT(*) as cardio_exercises
FROM exercises
WHERE role = 'cardio';
-- Expected: ~40

-- QF-4: Verify no cardio exercises remain as isolation
SELECT exercise_id, spanish_name, role
FROM exercises
WHERE role = 'isolation'
AND spanish_name ILIKE ANY(ARRAY[
  '%burpee%', '%assault%', '%salto%box%',
  '%mountain climber%', '%jumping jack%', '%battle rope%'
]);
-- Expected: 0 rows
```

### 9.2 Workflow-Level Tests

| Test ID | Fix | Description | Input | Expected Output |
|---------|-----|-------------|-------|-----------------|
| QT-001 | QF-1 | Generate routine for user with 60-min sessions, verify W4 exercise count | Standard user profile | For every day: W4 exercise count <= W1 exercise count |
| QT-002 | QF-2 | Force AI to return duplicate (mock test), verify dedup removes it | AI output with 2x same exercise_id | Output has exercise_id once; console log shows dedup |
| QT-003 | QF-3 | Generate Push day routine, verify no Abs/Core exercises appear | User with ppl_5 template | Push day exercises have main_muscle in (Chest, Shoulders, Triceps) |
| QT-004 | QF-4 | Generate routine with cardio exercises, verify set/rep params | User with Burpee in selected exercises | Burpee has sets=3, reps="30-45s", not "12-15" |
| QT-005 | QF-5 | Generate routine for Health D user, verify no heavy deadlifts | health_status = 'D' | No exercises with spanish_name LIKE '%peso muerto%' AND equipment = 'barbell' |
| QT-006 | QF-5 | Generate routine for Health E user, verify machines/bodyweight only | health_status = 'E' | All exercises have equipment IN ('machine', 'bodyweight', 'cable') |
| QT-007 | QF-5 | Health B user on Legs day, verify no barbell squats | health_status = 'B', ppl_5 template | No exercises with spanish_name LIKE '%sentadilla%barra%' |

### 9.3 E2E Test Integration

New test users should be added to `e2e/test_data_setup.sql`:

| Phone | User | Health | Purpose |
|-------|------|--------|---------|
| `570000000061` | Test_HealthB | B | QT-005/QT-007: Lower body restrictions |
| `570000000062` | Test_HealthD | D | QT-005: Spine restrictions |
| `570000000063` | Test_HealthE | E | QT-006: Special condition (machines only) |

---

## 10. Implementation Order

```
Phase 1: Data Layer (no workflow changes needed)
================================================
  Step 1.1: Run discovery queries for QF-3 misclassified exercises
  Step 1.2: Review and approve the specific UPDATE statements
  Step 1.3: Apply QF-3 migration (pattern/muscle corrections)
  Step 1.4: Apply QF-4 migration (cardio role + set_profiles + exercise reclassification)
  Step 1.5: Run validation queries (Section 9.1)

Phase 2: SQL Query Layer
========================
  Step 2.1: Modify GetExercisesByPattern SQL (QF-5 health filters)
  Step 2.2: Test with Health B/C/D/E profiles manually

Phase 3: Code Node Layer
=========================
  Step 3.1: Modify Code in JavaScript (QF-2 dedup)
  Step 3.2: Modify ValidateWorkoutDuration (QF-1 W4 cap)
  Step 3.3: Update AI Agent prompt (QF-2 anti-dup rule, QF-4 cardio note)

Phase 4: Integration Testing
=============================
  Step 4.1: Generate routine for standard user, verify all 5 fixes
  Step 4.2: Generate routine for Health D user, verify exercise exclusion
  Step 4.3: Compare W1 vs W4 exercise counts across all days
  Step 4.4: Add E2E test cases (QT-001 through QT-007)
```

**Estimated effort:** Phase 1 is the most labor-intensive (data discovery and manual review). Phases 2-3 are straightforward code changes. Phase 4 depends on having the n8n instance available for live testing.

---

*End of Architecture Specification*
