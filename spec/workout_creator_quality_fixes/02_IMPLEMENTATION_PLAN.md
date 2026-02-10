# 02 - Technical Implementation Plan: WORKOUT_CREATOR Quality Fixes

**Document**: Implementation Specification
**Author**: Lead Solutions Architect
**Date**: 2026-02-09
**Status**: Ready for Execution
**Target Workflow**: `n8n/running_flows/WORKOUT_CREATOR.json`

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Overview](#2-architecture-overview)
3. [Phase 1: Database Schema & Data Fixes (Fix 4 + Fix 5)](#3-phase-1-database-schema--data-fixes)
4. [Phase 2: n8n Workflow Logic (Fix 1 + Fix 2 + Fix 3)](#4-phase-2-n8n-workflow-logic)
5. [Phase 3: AI Prompt Engineering (Fix 1 + Fix 3)](#5-phase-3-ai-prompt-engineering)
6. [Phase 4: Integration Testing](#6-phase-4-integration-testing)
7. [Dependency Graph](#7-dependency-graph)
8. [Rollback Plan](#8-rollback-plan)

---

## 1. Executive Summary

Five systemic quality issues have been identified in the WORKOUT_CREATOR workflow. This document provides the step-by-step implementation plan with exact code, SQL, and validation instructions for each task.

| Fix | Priority | Effort | Impact | Phase |
|-----|----------|--------|--------|-------|
| Fix 1 - Exercise Deduplication | P0 | Low | High | Phase 2 + 3 |
| Fix 2 - W4 Volume Cap | P1 | Medium | High | Phase 2 |
| Fix 3 - Health Status SQL Enforcement | P0 | Medium | Critical | Phase 2 + 3 |
| Fix 4 - New Cardio Role | P2 | Low-Medium | Medium | Phase 1 |
| Fix 5 - Exercise Data Cleanup | P3 | Low | Medium | Phase 1 |

**Execution order**: Phase 1 (database) must complete before Phase 2 (workflow logic). Phase 3 (prompts) can run in parallel with Phase 2. Phase 4 (testing) follows all others.

---

## 2. Architecture Overview

### Current Workflow Pipeline

```
input (whatsapp_id)
  -> GetUserProfile
  -> ProcessUserPreferences        [Code node: transforms profile]
  -> If_Is_Renewal
  -> LoadProfile                   [Supabase: set_profiles W1]
  -> Get_Day_Requirements          [Postgres: routine_templates JOIN]
  -> If_Skip_Create_For_Renewal
  -> GetUser / CreateUser / CreatePlan
  -> Merge
  -> Loop Over Items               [SplitInBatches: iterates day_requirements]
     -> GetExercisesByPattern       [Postgres: exercises WHERE pattern]  <-- Fix 3 SQL here
     -> AI Agent                    [Gemini: selects exercises]          <-- Fix 1 + Fix 3 prompt here
  -> AI Transform                  [Unifies loop outputs]
  -> Code in JavaScript            [Parses AI JSON output]              <-- Fix 1 dedup here
  -> Get a row                     [Supabase: set_profiles all weeks]
  -> Code in JavaScript1           [Expands W1 -> W1-W4]               <-- Fix 4 cardio handling here
  -> ValidateWorkoutDuration       [Code node: time validation]         <-- Fix 2 volume cap here
  -> Create a row                  [Supabase: INSERT workouts]
```

### Key Node IDs (from WORKOUT_CREATOR.json)

| Node Name | Node ID | Type |
|-----------|---------|------|
| `AI Agent` | `cdf4b2d2-d446-46d9-8464-25a26844dc0a` | langchain.agent |
| `ProcessUserPreferences` | `0d254b45-f4c8-40ee-95c6-27e96af394e3` | code v2 |
| `Code in JavaScript` | `e4feabe9-660d-489d-ae4a-d489884e86cf` | code v2 |
| `Code in JavaScript1` | `6c5260a3-5b87-4235-911b-1b181adfb61f` | code v2 |
| `ValidateWorkoutDuration` | `db095e1f-1b73-425f-8841-fbeb28073419` | code v2 |
| `GetExercisesByPattern` | `6239f1ba-017e-4d4b-9c20-9dc6c093183f` | postgres v2.6 |
| `LoadProfile` | `37f52e9d-464d-46de-ad20-0d3a88ee1bd3` | supabase v1 |
| `Get a row` | `9f4d92f8-1109-4977-b72f-dc77254156f3` | supabase v1 |

---

## 3. Phase 1: Database Schema & Data Fixes

### T-101: Insert `cardio` into `exercise_role` table

**Assignee**: [pixel-dev]
**Depends on**: None
**Input**: Supabase `exercise_role` table (current values: `compound`, `isolation`, `core`)

**Technical Detail**:

Create a Supabase migration with the following SQL:

```sql
-- Migration: add_cardio_role
-- Description: Add 'cardio' to exercise_role lookup table

INSERT INTO exercise_role (role)
VALUES ('cardio')
ON CONFLICT (role) DO NOTHING;
```

**Validation (QA)** [code-reviewer]:
1. Run: `SELECT * FROM exercise_role ORDER BY role;`
2. Confirm output includes exactly 4 rows: `cardio`, `compound`, `core`, `isolation`
3. Confirm no existing data was modified

---

### T-102: Update ~40 exercises to `role = 'cardio'`

**Assignee**: [pixel-dev]
**Depends on**: T-101
**Input**: `exercises` table, keyword-based identification of cardio/plyometric exercises

**Technical Detail**:

Step 1 - Discovery query to identify candidates:

```sql
-- Identify cardio/plyo exercises currently misclassified
SELECT exercise_id, spanish_name, pattern, role, main_muscle, equipment
FROM exercises
WHERE (
    spanish_name ILIKE '%salto%'
    OR spanish_name ILIKE '%jump%'
    OR spanish_name ILIKE '%burpee%'
    OR spanish_name ILIKE '%mountain climber%'
    OR spanish_name ILIKE '%escalador%'
    OR spanish_name ILIKE '%skipping%'
    OR spanish_name ILIKE '%sprint%'
    OR spanish_name ILIKE '%box jump%'
    OR spanish_name ILIKE '%jumping jack%'
    OR spanish_name ILIKE '%tuck jump%'
    OR spanish_name ILIKE '%high knees%'
    OR spanish_name ILIKE '%rodillas altas%'
    OR spanish_name ILIKE '%cardio%'
    OR spanish_name ILIKE '%plyo%'
    OR spanish_name ILIKE '%battle rope%'
    OR spanish_name ILIKE '%cuerda de batalla%'
    OR spanish_name ILIKE '%remo ergometro%'
    OR spanish_name ILIKE '%bicicleta%'
    OR spanish_name ILIKE '%assault bike%'
    OR spanish_name ILIKE '%sled%'
    OR spanish_name ILIKE '%trineo%'
)
AND role != 'cardio'
ORDER BY spanish_name;
```

Step 2 - Migration (after reviewing discovery results):

```sql
-- Migration: update_exercises_cardio_role
-- Description: Reclassify cardio/plyometric exercises from compound/isolation to cardio

UPDATE exercises
SET role = 'cardio'
WHERE exercise_id IN (
    -- List the specific exercise_ids from the discovery query above.
    -- DO NOT use pattern matching in the UPDATE itself; only update
    -- exercises that have been manually verified as cardio movements.
    -- Example (replace with actual IDs after review):
    -- 'ex_jump_squat_001', 'ex_burpee_001', ...
);

-- Log count for audit
-- Expected: ~30-50 rows affected
```

**IMPORTANT**: The [pixel-dev] must run the discovery query first, manually review each result to confirm it is genuinely a cardio exercise (not a compound movement like "Sentadilla con salto" which might serve double duty), and only then build the final UPDATE with verified `exercise_id` values.

**Validation (QA)** [code-reviewer]:
1. Run: `SELECT role, COUNT(*) FROM exercises GROUP BY role ORDER BY role;`
2. Confirm `cardio` count is between 25 and 50
3. Run: `SELECT exercise_id, spanish_name, role FROM exercises WHERE role = 'cardio' ORDER BY spanish_name;`
4. Spot-check 5 exercises: each should be clearly a cardio/plyometric movement
5. Confirm no compound exercises (squat, bench press, deadlift, row) were reclassified

---

### T-103: Insert `cardio` rows into `set_profiles`

**Assignee**: [pixel-dev]
**Depends on**: T-101
**Input**: `set_profiles` table schema, existing rows for `compound`/`isolation`/`core`

**Technical Detail**:

Cardio exercises use time-based parameters rather than traditional rep schemes. The `reps` field will store seconds (e.g., `'30'` means 30 seconds of work).

```sql
-- Migration: add_cardio_set_profiles
-- Description: Add set_profiles rows for cardio role across all goal/level/week combos

-- Discover existing goal/level combinations:
-- SELECT DISTINCT goal, level FROM set_profiles ORDER BY goal, level;

-- Insert cardio profiles for every existing (goal, level, week) combination.
-- Pattern:
--   W1: 3 sets, 30s work, RIR N/A, rest 60s, tempo N/A
--   W2: 3 sets, 35s work, RIR N/A, rest 60s, tempo N/A
--   W3: 4 sets, 40s work, RIR N/A, rest 45s, tempo N/A (peak)
--   W4: 2 sets, 25s work, RIR N/A, rest 60s, tempo N/A (deload)

INSERT INTO set_profiles (goal, level, week, role, sets, reps, rir, rest_sec, tempo, notes)
SELECT
    sp.goal,
    sp.level,
    w.week_num,
    'cardio' AS role,
    CASE w.week_num
        WHEN 1 THEN 3
        WHEN 2 THEN 3
        WHEN 3 THEN 4
        WHEN 4 THEN 2
    END AS sets,
    CASE w.week_num
        WHEN 1 THEN '30'
        WHEN 2 THEN '35'
        WHEN 3 THEN '40'
        WHEN 4 THEN '25'
    END AS reps,
    'N/A' AS rir,
    CASE w.week_num
        WHEN 1 THEN 60
        WHEN 2 THEN 60
        WHEN 3 THEN 45
        WHEN 4 THEN 60
    END AS rest_sec,
    'N/A' AS tempo,
    'Time-based: reps value = seconds of work' AS notes
FROM (SELECT DISTINCT goal, level FROM set_profiles) sp
CROSS JOIN (VALUES (1), (2), (3), (4)) AS w(week_num)
WHERE NOT EXISTS (
    SELECT 1 FROM set_profiles existing
    WHERE existing.goal = sp.goal
      AND existing.level = sp.level
      AND existing.week = w.week_num
      AND existing.role = 'cardio'
);
```

**Validation (QA)** [code-reviewer]:
1. Run: `SELECT COUNT(*) FROM set_profiles WHERE role = 'cardio';`
2. Expected count = (number of distinct goal/level combos) x 4 weeks. Currently 5 goals x 3 levels = 15 combos x 4 = **60 rows**.
3. Run: `SELECT goal, level, week, sets, reps, rest_sec FROM set_profiles WHERE role = 'cardio' ORDER BY goal, level, week;`
4. Verify W4 has `sets = 2` (deload) and W3 has `sets = 4` (peak)
5. Verify no duplicate rows exist: `SELECT goal, level, week, role, COUNT(*) FROM set_profiles WHERE role = 'cardio' GROUP BY goal, level, week, role HAVING COUNT(*) > 1;` (should return 0 rows)

---

### T-104: Fix misclassified push/pull exercises (~30 exercises)

**Assignee**: [pixel-dev]
**Depends on**: None (independent of T-101 through T-103)
**Input**: `exercises` table, `exercise_patterns` table

**Technical Detail**:

Step 1 - Discovery query to find misclassified exercises:

```sql
-- Find exercises where pattern is push_h/push_v/pull_h/pull_v
-- but main_muscle does not match expected muscles for that pattern

-- Expected muscle mappings:
-- push_h (horizontal push): Chest, Triceps, Shoulders(front)
-- push_v (vertical push): Shoulders, Triceps
-- pull_h (horizontal pull): Back, Lats, Biceps, Traps
-- pull_v (vertical pull): Lats, Back, Biceps

-- PUSH patterns with non-push muscles
SELECT exercise_id, spanish_name, pattern, main_muscle, secondary_muscles, role, equipment
FROM exercises
WHERE pattern IN ('push_h', 'push_v')
  AND main_muscle NOT IN ('Chest', 'Shoulders', 'Triceps')
ORDER BY pattern, main_muscle, spanish_name;

-- PULL patterns with non-pull muscles
SELECT exercise_id, spanish_name, pattern, main_muscle, secondary_muscles, role, equipment
FROM exercises
WHERE pattern IN ('pull_h', 'pull_v')
  AND main_muscle NOT IN ('Back', 'Lats', 'Biceps', 'Traps', 'Lower back')
ORDER BY pattern, main_muscle, spanish_name;
```

Step 2 - For each misclassified exercise, determine the correct fix:

| Scenario | Action |
|----------|--------|
| Pattern is wrong but main_muscle is correct | UPDATE pattern to match the muscle |
| Main_muscle is wrong but pattern is correct | UPDATE main_muscle to match the pattern |
| Both seem wrong | Research the exercise and fix both |

Step 3 - Migration (after manual review):

```sql
-- Migration: fix_push_pull_exercise_classification
-- Description: Correct ~30 exercises with mismatched pattern/main_muscle

-- Example fixes (replace with actual values after discovery):
-- UPDATE exercises SET pattern = 'hip_hinge' WHERE exercise_id = 'XXX';
-- UPDATE exercises SET main_muscle = 'Chest' WHERE exercise_id = 'YYY';

-- Each UPDATE must be individually justified with a comment:
-- exercise_id 'XXX': "Peso muerto rumano" is a hip_hinge, not pull_h
-- exercise_id 'YYY': "Press de banca" targets Chest, was incorrectly listed as Shoulders
```

**Validation (QA)** [code-reviewer]:
1. Re-run both discovery queries from Step 1 after the migration
2. Push exercises should only have main_muscle in: `Chest`, `Shoulders`, `Triceps`
3. Pull exercises should only have main_muscle in: `Back`, `Lats`, `Biceps`, `Traps`, `Lower back`
4. Total exercises changed should be documented (expected: ~30)
5. Run: `SELECT COUNT(*) FROM exercises;` before and after - total count must not change (no deletes or inserts)

---

## 4. Phase 2: n8n Workflow Logic

### T-201: Add exercise deduplication guard to `Code in JavaScript` node

**Assignee**: [n8n-agent]
**Depends on**: None (can start immediately)
**Input**: `n8n/running_flows/WORKOUT_CREATOR.json`, node `Code in JavaScript` (ID: `e4feabe9-660d-489d-ae4a-d489884e86cf`)

**Technical Detail**:

The current `Code in JavaScript` node parses the AI Agent's JSON output. After parsing, add a deduplication pass that removes duplicate `exercise_id` values within the same `day_name`.

Current code (relevant section):

```javascript
// Agregamos los ejercicios al arreglo maestro
allExercises.push(...exercises);
```

Replace with:

```javascript
// Agregamos los ejercicios al arreglo maestro
allExercises.push(...exercises);
```

Then, AFTER the `for` loop completes (before the `return` statement), insert the deduplication block:

```javascript
// === DEDUPLICATION GUARD ===
// Remove duplicate exercise_ids within the same day_name.
// The AI may select the same exercise twice; keep only the first occurrence.
const deduplicatedExercises = [];
const seenByDay = {};  // { day_name: Set<exercise_id> }

for (const ex of allExercises) {
  const dayKey = ex.day_name || 'unknown';
  if (!seenByDay[dayKey]) {
    seenByDay[dayKey] = new Set();
  }
  if (seenByDay[dayKey].has(ex.exercise_id)) {
    console.log(`DEDUP: Removed duplicate exercise_id=${ex.exercise_id} from day=${dayKey}`);
    continue;
  }
  seenByDay[dayKey].add(ex.exercise_id);
  deduplicatedExercises.push(ex);
}

const dupCount = allExercises.length - deduplicatedExercises.length;
if (dupCount > 0) {
  console.log(`DEDUP: Removed ${dupCount} duplicate exercises total`);
}

// Replace allExercises with deduplicated version
allExercises = deduplicatedExercises;
```

The full updated `jsCode` for the node becomes:

```javascript
// Arreglo para acumular todos los ejercicios finales
let allExercises = [];

// Recorremos los items de entrada (usualmente es 1 que contiene el unifiedArray)
for (const item of $input.all()) {
  const unified = item.json.unifiedArray;
  if (Array.isArray(unified)) {
    for (const entry of unified) {
      try {
        let outputStr = entry.output;

        // Limpiar los bloques de codigo markdown (```json ... ```)
        outputStr = outputStr
          .replace(/^```json\s*/i, '')  // Quitar ```json del inicio
          .replace(/^```\s*/i, '')       // O solo ``` del inicio
          .replace(/\s*```$/i, '')       // Quitar ``` del final
          .trim();

        // Parseamos el string limpio
        const exercises = JSON.parse(outputStr);

        // Agregamos los ejercicios al arreglo maestro
        allExercises.push(...exercises);
      } catch (error) {
        // Ignoramos errores de parseo si algun output viene mal
        continue;
      }
    }
  }
}

// === DEDUPLICATION GUARD ===
// Remove duplicate exercise_ids within the same day_name.
const deduplicatedExercises = [];
const seenByDay = {};

for (const ex of allExercises) {
  const dayKey = ex.day_name || 'unknown';
  if (!seenByDay[dayKey]) {
    seenByDay[dayKey] = new Set();
  }
  if (seenByDay[dayKey].has(ex.exercise_id)) {
    console.log(`DEDUP: Removed duplicate exercise_id=${ex.exercise_id} from day=${dayKey}`);
    continue;
  }
  seenByDay[dayKey].add(ex.exercise_id);
  deduplicatedExercises.push(ex);
}

const dupCount = allExercises.length - deduplicatedExercises.length;
if (dupCount > 0) {
  console.log(`DEDUP: Removed ${dupCount} duplicate exercises total`);
}

allExercises = deduplicatedExercises;

// Retornamos cada ejercicio como un item independiente para n8n
return allExercises.map(exercise => ({ json: exercise }));
```

**Validation (QA)** [code-reviewer]:
1. Deploy updated workflow to n8n
2. Run WORKOUT_CREATOR with pinned test user (`whatsapp_id: 573123623296`)
3. Check `Code in JavaScript` node output: no two items should share the same `exercise_id + day_name` combination
4. Check console logs for `DEDUP:` messages - if present, the guard is working
5. Verify downstream nodes (`Code in JavaScript1`, `ValidateWorkoutDuration`, `Create a row`) still receive valid data

---

### T-202: Add W4 Volume Cap to `ValidateWorkoutDuration` node

**Assignee**: [n8n-agent]
**Depends on**: T-201 (dedup must run first so W1 exercise counts are accurate)
**Input**: `n8n/running_flows/WORKOUT_CREATOR.json`, node `ValidateWorkoutDuration` (ID: `db095e1f-1b73-425f-8841-fbeb28073419`)

**Technical Detail**:

After the existing time-based validation loop, add a **volume cap pass** that ensures no week has more exercises per day than W1. This prevents the AI from adding extra exercises in weeks 2-4.

Insert the following block into the `ValidateWorkoutDuration` node's `jsCode`, AFTER the existing `for (const [key, dayWorkouts] ...` loop and BEFORE the final `console.log` statements. The new code operates on the already-time-validated `processedWorkouts` array.

```javascript
// === VOLUME CAP: Ensure W2-W4 never exceed W1 exercise count per day ===

// Step 1: Count W1 exercises per day_name
const w1CountByDay = {};
for (const w of processedWorkouts) {
  if (w.week_number === 1) {
    if (!w1CountByDay[w.day_name]) w1CountByDay[w.day_name] = 0;
    w1CountByDay[w.day_name]++;
  }
}

// Step 2: For each day in W2-W4, trim excess exercises
const volumeCapLog = [];
const volumeCappedWorkouts = [];

// Group processed workouts by week+day for volume cap check
const groupedForCap = {};
for (const w of processedWorkouts) {
  const key = `W${w.week_number}-${w.day_name}`;
  if (!groupedForCap[key]) groupedForCap[key] = [];
  groupedForCap[key].push(w);
}

for (const [key, dayWorkouts] of Object.entries(groupedForCap)) {
  const [weekStr, dayName] = key.split('-');
  const weekNumber = parseInt(weekStr.replace('W', ''), 10);
  const maxExercises = w1CountByDay[dayName];

  // W1 or no W1 reference: keep as-is
  if (weekNumber === 1 || !maxExercises || dayWorkouts.length <= maxExercises) {
    volumeCappedWorkouts.push(...dayWorkouts);
    continue;
  }

  // W2-W4 exceeds W1 count: remove lowest-priority exercises
  const sorted = dayWorkouts
    .map(ex => ({
      ...ex,
      _priorityScore: getPriorityScore(ex, priorityMuscles, exerciseLookup)
    }))
    .sort((a, b) => b._priorityScore - a._priorityScore); // highest first = keep

  const kept = sorted.slice(0, maxExercises);
  const removed = sorted.slice(maxExercises);

  volumeCapLog.push({
    week: weekStr,
    day: dayName,
    w1_count: maxExercises,
    before_count: dayWorkouts.length,
    after_count: kept.length,
    removed_ids: removed.map(r => r.exercise_id)
  });

  // Clean up temp field and push kept exercises
  kept.forEach(ex => { delete ex._priorityScore; });
  volumeCappedWorkouts.push(...kept);
}

if (volumeCapLog.length > 0) {
  console.log('=== VOLUME CAP APPLIED ===');
  console.log(JSON.stringify(volumeCapLog, null, 2));
}
```

Then replace the final return statement. Change:

```javascript
return processedWorkouts.map(w => ({ json: w }));
```

To:

```javascript
return volumeCappedWorkouts.map(w => ({ json: w }));
```

**Validation (QA)** [code-reviewer]:
1. Deploy and run WORKOUT_CREATOR with test user
2. Query output: for each `day_name`, count exercises per week:
   ```sql
   SELECT day_name, week, COUNT(*) as exercise_count
   FROM workouts
   WHERE user_id = '<test_user_id>'
   GROUP BY day_name, week
   ORDER BY day_name, week;
   ```
3. For every day_name, W2/W3/W4 exercise_count must be <= W1 exercise_count
4. Check console logs for `VOLUME CAP APPLIED` - if present, verify the removed exercises were low-priority
5. W4 should have same exercise count as W1 but with reduced sets (MIN_SETS = 2)

---

### T-203: Add health status SQL filters to `GetExercisesByPattern` node

**Assignee**: [n8n-agent]
**Depends on**: T-204 (ProcessUserPreferences must pass health_status first)
**Input**: `n8n/running_flows/WORKOUT_CREATOR.json`, node `GetExercisesByPattern` (ID: `6239f1ba-017e-4d4b-9c20-9dc6c093183f`)

**Technical Detail**:

The current `GetExercisesByPattern` node builds a dynamic SQL query in an n8n expression. The health status is already available via `ProcessUserPreferences` but is NOT used in the SQL. Add health-based WHERE clauses.

Current query expression (abbreviated):

```javascript
let query = `
  SELECT exercise_id, spanish_name, pattern, role, main_muscle,
         secondary_muscles, level, link, equipment
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

Updated query expression -- add health status variable extraction and WHERE clauses between the equipment filter and the ORDER BY:

```javascript
={{ (() => {
  const isHome = $items('ProcessUserPreferences')[0].json.processed.home.is_home;
  const equipmentSql = $items('ProcessUserPreferences')[0].json.processed.home.home_equipment_sql;
  const pattern = $json.pattern;
  const level = $items('ProcessUserPreferences')[0].json.fitness_level;
  const healthStatus = $items('ProcessUserPreferences')[0].json.health_status || 'A';

  let query = `
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
    WHERE pattern = '${pattern}'
  `;

  if (isHome && equipmentSql) {
    query += ` AND equipment IN (${equipmentSql})`;
  }

  // === HEALTH STATUS SQL ENFORCEMENT ===

  // Health B (lower body issues): exclude high-impact jumps and plyometrics
  if (healthStatus === 'B') {
    query += `
      AND spanish_name NOT ILIKE '%salto%'
      AND spanish_name NOT ILIKE '%jump%'
      AND spanish_name NOT ILIKE '%burpee%'
      AND spanish_name NOT ILIKE '%box jump%'
      AND spanish_name NOT ILIKE '%sentadilla bulgara%'
    `;
  }

  // Health C (upper body issues): exclude overhead pressing and shoulder stress
  if (healthStatus === 'C') {
    query += `
      AND spanish_name NOT ILIKE '%press militar%'
      AND spanish_name NOT ILIKE '%overhead%'
      AND spanish_name NOT ILIKE '%snatch%'
      AND spanish_name NOT ILIKE '%press de hombro%'
      AND spanish_name NOT ILIKE '%push press%'
      AND spanish_name NOT ILIKE '%press arnold%'
    `;
  }

  // Health D (spine issues): exclude lower back exercises and heavy axial loading
  if (healthStatus === 'D') {
    query += `
      AND main_muscle NOT IN ('Lower back')
      AND spanish_name NOT ILIKE '%peso muerto%'
      AND spanish_name NOT ILIKE '%deadlift%'
      AND spanish_name NOT ILIKE '%buenos dias%'
      AND spanish_name NOT ILIKE '%good morning%'
    `;
  }

  // Health E (special condition): exclude free barbell exercises
  if (healthStatus === 'E') {
    query += `
      AND equipment NOT IN ('barbell')
    `;
  }

  query += `
    ORDER BY
      CASE WHEN level = '${level}' THEN 0 ELSE 1 END,
      role,
      spanish_name
  `;

  return query;
})() }}
```

**Validation (QA)** [code-reviewer]:
1. Test with health status B user: run `GetExercisesByPattern` for a leg day pattern
   - Verify NO exercises with "salto", "jump", "burpee" in spanish_name appear
2. Test with health status C user: run `GetExercisesByPattern` for a push_v pattern
   - Verify NO exercises with "press militar", "overhead", "snatch" appear
3. Test with health status D user: run `GetExercisesByPattern` for a hip_hinge pattern
   - Verify NO exercises with main_muscle = "Lower back" or "peso muerto" in name appear
4. Test with health status E user: run for any pattern
   - Verify NO exercises with equipment = "barbell" appear
5. Test with health status A user: verify NO filters are applied (full exercise list returned)
6. Edge case: verify the query does not fail when 0 exercises match (empty result set is valid)

---

### T-204: Pass `health_status` through ProcessUserPreferences to SQL context

**Assignee**: [n8n-agent]
**Depends on**: None
**Input**: `n8n/running_flows/WORKOUT_CREATOR.json`, node `ProcessUserPreferences` (ID: `0d254b45-f4c8-40ee-95c6-27e96af394e3`)

**Technical Detail**:

The `health_status` field is already included in the `ProcessUserPreferences` output (it passes through as `...profile`), so it is already available at `$items('ProcessUserPreferences')[0].json.health_status`. No code change is required in ProcessUserPreferences itself.

However, verify the field exists. The current code does:

```javascript
return [{
  json: {
    ...profile,  // <-- health_status comes through here
    processed: {
      // ...
      health: getHealthRestrictions(profile.health_status),
      // ...
    }
  }
}];
```

The `health_status` raw value (e.g., `'A'`, `'B'`, `'C'`, `'D'`, `'E'`) is available at `$items('ProcessUserPreferences')[0].json.health_status` because of the `...profile` spread.

**Action**: No code change needed. This ticket exists to document and verify the data flow.

**Validation (QA)** [code-reviewer]:
1. Run WORKOUT_CREATOR with a user who has `health_status = 'C'` in `users_gym_profile`
2. Check `ProcessUserPreferences` node output: confirm `json.health_status` = `'C'`
3. Check `GetExercisesByPattern` node: confirm the expression `$items('ProcessUserPreferences')[0].json.health_status` resolves to `'C'`

---

### T-205: Update `Code in JavaScript1` to handle cardio role in week expansion

**Assignee**: [n8n-agent]
**Depends on**: T-101, T-103 (cardio role and set_profiles must exist)
**Input**: `n8n/running_flows/WORKOUT_CREATOR.json`, node `Code in JavaScript1` (ID: `6c5260a3-5b87-4235-911b-1b181adfb61f`)

**Technical Detail**:

The `Code in JavaScript1` node expands W1 exercises into W1-W4 by looking up `set_profiles` for each `(goal, level, role, week)` combination. Since T-103 adds `cardio` rows to `set_profiles`, this node will automatically find matches for `role = 'cardio'` without code changes.

However, the `rolePriority` sorting needs to be updated to place cardio exercises LAST (as finishers):

Current code:

```javascript
const rolePriority = { compound: 1, core: 2, isolation: 3 };
```

Updated code:

```javascript
const rolePriority = { compound: 1, core: 2, isolation: 3, cardio: 4 };
```

This ensures cardio exercises appear at the end of each day's exercise list (exercise_order 7+), functioning as finishers.

**Validation (QA)** [code-reviewer]:
1. Run WORKOUT_CREATOR with a test user whose routine includes a cardio exercise
2. Check `Code in JavaScript1` output: cardio exercises should have the highest `exercise_order` values
3. Verify cardio exercises have W1-W4 variants with correct set_profiles data:
   - W1: 3 sets, 30s
   - W4: 2 sets, 25s
4. Verify no `"No match para:"` notes appear in the output for cardio role exercises

---

### T-206: Update `ValidateWorkoutDuration` to handle cardio time calculation

**Assignee**: [n8n-agent]
**Depends on**: T-205
**Input**: `n8n/running_flows/WORKOUT_CREATOR.json`, node `ValidateWorkoutDuration` (ID: `db095e1f-1b73-425f-8841-fbeb28073419`)

**Technical Detail**:

The current `calculateExerciseTime` function assumes all exercises use a rep-based calculation: `sets * reps * tempoPerRep`. For cardio exercises where `reps` represents seconds of work and `tempo` is `'N/A'`, this formula produces incorrect results.

Update the `calculateExerciseTime` function:

Current:

```javascript
function calculateExerciseTime(exercise) {
  const sets = parseInt(exercise.sets, 10) || 3;
  const reps = parseInt(exercise.reps, 10) || 10;
  const restSeconds = parseInt(exercise['rest-seconds'] || exercise.rest_seconds, 10) || 60;
  const tempoPerRep = parseTempo(exercise.tempo);

  const workTime = sets * reps * tempoPerRep;
  const restTime = (sets - 1) * restSeconds;

  return workTime + restTime;
}
```

Updated:

```javascript
function calculateExerciseTime(exercise) {
  const sets = parseInt(exercise.sets, 10) || 3;
  const reps = parseInt(exercise.reps, 10) || 10;
  const restSeconds = parseInt(exercise['rest-seconds'] || exercise.rest_seconds, 10) || 60;
  const role = (exercise.role || '').toLowerCase();

  let workTime;
  if (role === 'cardio') {
    // Cardio: reps = seconds of work per set, no tempo multiplier
    workTime = sets * reps;  // e.g., 3 sets * 30 seconds = 90 seconds
  } else {
    // Standard: sets * reps * tempo
    const tempoPerRep = parseTempo(exercise.tempo);
    workTime = sets * reps * tempoPerRep;
  }

  const restTime = (sets - 1) * restSeconds;
  return workTime + restTime;
}
```

**Validation (QA)** [code-reviewer]:
1. Manual calculation check:
   - Cardio W1: 3 sets x 30s work + 2 x 60s rest = 90 + 120 = 210 seconds (3.5 min) -- correct
   - Standard: 3 sets x 10 reps x 4s tempo + 2 x 60s rest = 120 + 120 = 240 seconds (4 min) -- unchanged
2. Run WORKOUT_CREATOR with a routine that includes cardio exercises
3. Check `ValidateWorkoutDuration` console output: day durations should be reasonable (not inflated by bad cardio math)
4. Verify cardio exercises are NOT being removed by the time validator (they should be short ~3-4 min each)

---

## 5. Phase 3: AI Prompt Engineering

### T-301: Add deduplication rules to AI Agent system prompt

**Assignee**: [n8n-agent]
**Depends on**: None (can start in parallel with Phase 2)
**Input**: `n8n/running_flows/WORKOUT_CREATOR.json`, node `AI Agent` (ID: `cdf4b2d2-d446-46d9-8464-25a26844dc0a`), `systemMessage` field

**Technical Detail**:

Locate the `## REGLAS DE ORO` section in the AI Agent's system prompt. Currently:

```
## REGLAS DE ORO
- PROHIBIDO inventar ejercicios. Solo usar AVAILABLE_EXERCISES.
- PROHIBIDO ignorar exclusiones de salud o musculos no deseados.
- OBLIGATORIO personalizar segun el perfil completo.
- OBLIGATORIO compensar gaps si el ambiente es HOME (ver tabla de compensacion).
- PROHIBIDO seleccionar ejercicios que requieran equipamiento no disponible.
```

Updated (add 2 new rules after the existing ones):

```
## REGLAS DE ORO
- PROHIBIDO inventar ejercicios. Solo usar AVAILABLE_EXERCISES.
- PROHIBIDO ignorar exclusiones de salud o musculos no deseados.
- OBLIGATORIO personalizar segun el perfil completo.
- OBLIGATORIO compensar gaps si el ambiente es HOME (ver tabla de compensacion).
- PROHIBIDO seleccionar ejercicios que requieran equipamiento no disponible.
- PROHIBIDO repetir el mismo exercise_id en el mismo dia. Cada ejercicio debe aparecer UNA SOLA VEZ por dia.
- PROHIBIDO seleccionar variantes casi identicas del mismo movimiento en el mismo dia (ejemplo: NO poner 3 tipos de curl de biceps, NO poner remo con barra + remo con barra T + remo invertido en el mismo dia). Buscar VARIEDAD de angulos, equipos y musculos secundarios.
```

**Validation (QA)** [code-reviewer]:
1. Deploy and run WORKOUT_CREATOR 3 times with different test users
2. For each run, check the AI Agent's output JSON: within each day, no `exercise_id` should appear more than once
3. Within each day, visually check that exercises provide variety (different main_muscles or different equipment)
4. If duplicates still appear despite the prompt, the Code guard (T-201) will catch them as a safety net

---

### T-302: Replace health "CUIDADO" warnings with explicit "EXCLUSIONES OBLIGATORIAS" in AI prompt

**Assignee**: [n8n-agent]
**Depends on**: None
**Input**: `n8n/running_flows/WORKOUT_CREATOR.json`, node `AI Agent` (ID: `cdf4b2d2-d446-46d9-8464-25a26844dc0a`), `systemMessage` field

**Technical Detail**:

Locate the health status section in the system prompt. Currently it uses conditional emoji warnings:

```
**Salud (codigos confirmados de tabla health_status):**
- Estado: {{ $('ProcessUserPreferences').item.json.health_status }}
{{ $('ProcessUserPreferences').item.json.processed.health.avoid_lower_body_impact ? '⚠️ CUIDADO TREN INFERIOR: Evitar alto impacto en rodillas, tobillos, cadera' : '' }}
{{ $('ProcessUserPreferences').item.json.processed.health.avoid_upper_body_overhead ? '⚠️ CUIDADO TREN SUPERIOR: Evitar ejercicios overhead y estres en hombros, codos, munecas' : '' }}
{{ $('ProcessUserPreferences').item.json.processed.health.avoid_spinal_loading ? '⚠️ CUIDADO ESPALDA: Evitar carga axial pesada, proteger lumbares y cervicales' : '' }}
{{ $('ProcessUserPreferences').item.json.processed.health.special_condition ? '⚠️ CONDICION MEDICA ESPECIAL: Priorizar ejercicios de bajo riesgo y maquinas guiadas' : '' }}
```

Replace with explicit exclusion lists:

```
**Salud (codigos confirmados de tabla health_status):**
- Estado: {{ $('ProcessUserPreferences').item.json.health_status }}
{{ $('ProcessUserPreferences').item.json.processed.health.avoid_lower_body_impact ? `
### EXCLUSIONES OBLIGATORIAS - SALUD B (Tren Inferior)
Los siguientes ejercicios ya fueron ELIMINADOS de AVAILABLE_EXERCISES por el filtro SQL.
Si alguno aparece en la lista, NO SELECCIONARLO:
- Ejercicios con saltos (jump squats, box jumps, burpees)
- Ejercicios de alto impacto en rodillas, tobillos o cadera
- PREFERIR: maquinas guiadas para pierna, ejercicios sentados, movimientos controlados
` : '' }}
{{ $('ProcessUserPreferences').item.json.processed.health.avoid_upper_body_overhead ? `
### EXCLUSIONES OBLIGATORIAS - SALUD C (Tren Superior)
Los siguientes ejercicios ya fueron ELIMINADOS de AVAILABLE_EXERCISES por el filtro SQL.
Si alguno aparece en la lista, NO SELECCIONARLO:
- Press militar, press de hombro, overhead press (cualquier variante)
- Snatch, push press, jerk, press Arnold
- Cualquier movimiento que lleve las manos por encima de la cabeza con carga
- PREFERIR: press inclinado con mancuerna (angulo <60), elevaciones laterales, face pulls
` : '' }}
{{ $('ProcessUserPreferences').item.json.processed.health.avoid_spinal_loading ? `
### EXCLUSIONES OBLIGATORIAS - SALUD D (Columna)
Los siguientes ejercicios ya fueron ELIMINADOS de AVAILABLE_EXERCISES por el filtro SQL.
Si alguno aparece en la lista, NO SELECCIONARLO:
- Peso muerto convencional, peso muerto rumano, peso muerto sumo
- Good mornings / Buenos dias
- Cualquier ejercicio con main_muscle = Lower back
- Sentadilla con barra en espalda (preferir sentadilla goblet o en maquina)
- PREFERIR: ejercicios con soporte lumbar, maquinas, cables
` : '' }}
{{ $('ProcessUserPreferences').item.json.processed.health.special_condition ? `
### EXCLUSIONES OBLIGATORIAS - SALUD E (Condicion Especial)
Los siguientes ejercicios ya fueron ELIMINADOS de AVAILABLE_EXERCISES por el filtro SQL.
Si alguno aparece en la lista, NO SELECCIONARLO:
- Cualquier ejercicio con equipment = barbell
- Ejercicios con alta demanda tecnica o riesgo de lesion
- PREFERIR: maquinas guiadas, cables, peso corporal con soporte
- PRIORIZAR: movimientos controlados, rangos de movimiento seguros
` : '' }}
```

**Validation (QA)** [code-reviewer]:
1. Run with health_status = 'C' user: verify the system prompt contains "EXCLUSIONES OBLIGATORIAS - SALUD C" section
2. Run with health_status = 'A' user: verify NO exclusion sections appear (all conditionals should resolve to empty strings)
3. Check AI Agent output: for health C user, confirm no overhead press variants are selected
4. Cross-reference with T-203 SQL filters: the AI should never even see the excluded exercises in AVAILABLE_EXERCISES

---

### T-303: Add cardio role recognition to AI Agent system prompt

**Assignee**: [n8n-agent]
**Depends on**: T-101 (cardio role must exist in DB)
**Input**: `n8n/running_flows/WORKOUT_CREATOR.json`, node `AI Agent` (ID: `cdf4b2d2-d446-46d9-8464-25a26844dc0a`), `systemMessage` field

**Technical Detail**:

Add a new section after `### PASO 3: Aplicar Carga` in the system prompt:

```
### PASO 3.5: Manejo de Ejercicios Cardio

Si AVAILABLE_EXERCISES contiene ejercicios con `role = 'cardio'`:
- Usarlos como **finisher** al final de la sesion (despues de isolation)
- Maximo 1-2 ejercicios cardio por dia
- La carga para cardio viene de CARGA SEMANA con role = 'cardio'
- El campo `reps` en cardio representa SEGUNDOS de trabajo (no repeticiones)
- El campo `tempo` para cardio es 'N/A' (no aplica)
- Si el day_requirement no incluye un patron cardio, NO agregar cardio por tu cuenta
```

Also update the output format section to include cardio as a valid role:

Current:
```
  "role": "compound/isolation/core",
```

Updated:
```
  "role": "compound/isolation/core/cardio",
```

**Validation (QA)** [code-reviewer]:
1. Run with a test user whose routine includes a pattern that has cardio exercises in AVAILABLE_EXERCISES
2. Verify AI Agent outputs cardio exercises with `role: "cardio"` at the end of the day
3. Verify cardio exercises use set_profiles values (not invented values)
4. Verify no more than 2 cardio exercises per day

---

## 6. Phase 4: Integration Testing

### T-401: End-to-end test - Health Status B user

**Assignee**: [n8n-agent]
**Depends on**: T-201, T-203, T-302 (all health-related fixes)
**Input**: Test user with `health_status = 'B'` in `users_gym_profile`

**Technical Detail**:

1. Create or update a test user in `users_gym_profile`:
   ```sql
   UPDATE users_gym_profile
   SET health_status = 'B'
   WHERE whatsapp_id = '570000000003';  -- Test_WithRoutine
   ```
   (Or use a dedicated test user phone number.)

2. Trigger WORKOUT_CREATOR for this user.

3. Query generated workouts:
   ```sql
   SELECT w.exercise_id, e.spanish_name, e.main_muscle, e.equipment
   FROM workouts w
   JOIN exercises e USING(exercise_id)
   WHERE w.user_id = '<user_id>'
   ORDER BY w.day_name, w.exercise_order;
   ```

4. Verify NONE of the following appear:
   - `spanish_name ILIKE '%salto%'`
   - `spanish_name ILIKE '%jump%'`
   - `spanish_name ILIKE '%burpee%'`

**Validation (QA)** [code-reviewer]:
1. Run the verification query
2. Document results: total exercises generated, any violations found
3. Reset test user health_status back to 'A' after testing

---

### T-402: End-to-end test - Health Status C user

**Assignee**: [n8n-agent]
**Depends on**: T-201, T-203, T-302
**Input**: Test user with `health_status = 'C'` (already exists: TC_HOME_FULL_HEALTH_C, phone `570000000213`)

**Technical Detail**:

1. Trigger WORKOUT_CREATOR for the health C test user.

2. Query generated workouts:
   ```sql
   SELECT w.exercise_id, e.spanish_name, e.main_muscle, e.equipment
   FROM workouts w
   JOIN exercises e USING(exercise_id)
   WHERE w.user_id = '<user_id>'
   ORDER BY w.day_name, w.exercise_order;
   ```

3. Verify NONE of the following appear:
   - `spanish_name ILIKE '%press militar%'`
   - `spanish_name ILIKE '%overhead%'`
   - `spanish_name ILIKE '%snatch%'`
   - `spanish_name ILIKE '%press de hombro%'`

**Validation (QA)** [code-reviewer]:
1. Run the verification query
2. Confirm zero overhead exercises in the generated plan
3. Confirm push_v pattern exercises are non-overhead alternatives (e.g., lateral raises, front raises)

---

### T-403: End-to-end test - Exercise deduplication

**Assignee**: [n8n-agent]
**Depends on**: T-201, T-301
**Input**: Any test user (use pinned data)

**Technical Detail**:

1. Run WORKOUT_CREATOR 3 times with the same test user (delete workouts between runs).

2. After each run, check for duplicates:
   ```sql
   SELECT user_id, day_name, week, exercise_id, COUNT(*) as cnt
   FROM workouts
   WHERE user_id = '<user_id>'
   GROUP BY user_id, day_name, week, exercise_id
   HAVING COUNT(*) > 1;
   ```

3. Expected: 0 rows returned (no duplicates).

**Validation (QA)** [code-reviewer]:
1. All 3 runs must return 0 duplicate rows
2. Check console logs: if DEDUP messages appear, note how many were caught (indicates the AI still tried to duplicate, but the guard caught it)
3. Document: was deduplication needed in 0/3, 1/3, 2/3, or 3/3 runs?

---

### T-404: End-to-end test - W4 Volume Cap

**Assignee**: [n8n-agent]
**Depends on**: T-202
**Input**: Any test user

**Technical Detail**:

1. Run WORKOUT_CREATOR for a test user.

2. Verify volume cap across all weeks:
   ```sql
   SELECT day_name, week, COUNT(*) as exercise_count
   FROM workouts
   WHERE user_id = '<user_id>'
   GROUP BY day_name, week
   ORDER BY day_name, week;
   ```

3. For each `day_name`, verify:
   - W2 exercise_count <= W1 exercise_count
   - W3 exercise_count <= W1 exercise_count
   - W4 exercise_count <= W1 exercise_count

4. Additionally verify W4 deload sets:
   ```sql
   SELECT day_name, exercise_id, sets
   FROM workouts
   WHERE user_id = '<user_id>' AND week = 4
   ORDER BY day_name, exercise_order;
   ```
   All sets should be <= 2 (deload week).

**Validation (QA)** [code-reviewer]:
1. Construct a comparison table from the query results
2. Confirm no week exceeds W1 count for any day
3. Confirm W4 sets are consistently 2 (MIN_SETS for deload)

---

### T-405: End-to-end test - Cardio role integration

**Assignee**: [n8n-agent]
**Depends on**: T-101, T-102, T-103, T-205, T-206, T-303
**Input**: Test user with a routine template that includes a pattern matching cardio exercises

**Technical Detail**:

1. Verify cardio exercises exist in the `exercises` table:
   ```sql
   SELECT COUNT(*) FROM exercises WHERE role = 'cardio';
   ```

2. Run WORKOUT_CREATOR for a test user whose template includes a day with cardio-eligible patterns.

3. Check if cardio exercises were included:
   ```sql
   SELECT w.day_name, w.week, w.exercise_order, w.sets, w.reps, w.tempo,
          e.spanish_name, e.role
   FROM workouts w
   JOIN exercises e USING(exercise_id)
   WHERE w.user_id = '<user_id>' AND e.role = 'cardio'
   ORDER BY w.day_name, w.week, w.exercise_order;
   ```

4. If cardio exercises are present, verify:
   - `exercise_order` is highest in the day (finisher position)
   - W1: sets = 3, reps = 30
   - W4: sets = 2, reps = 25
   - tempo = 'N/A'

**Validation (QA)** [code-reviewer]:
1. If no cardio exercises appear, verify whether the template's patterns actually matched any cardio exercises in the filtered set
2. If they do appear, confirm the set_profiles values are correct for all 4 weeks
3. Verify `ValidateWorkoutDuration` did not incorrectly calculate cardio time

---

### T-406: Regression test - Existing E2E test suite

**Assignee**: [n8n-agent]
**Depends on**: All prior tasks
**Input**: `n8n/tests/GymRatFlow_E2E_TestRunner.json`, `n8n/tests/MesocycleRenewal_E2E_TestRunner.json`

**Technical Detail**:

1. Run the full E2E test suite (`GymRatFlow_E2E_TestRunner.json`)
2. Run the mesocycle renewal test suite (`MesocycleRenewal_E2E_TestRunner.json`)
3. All existing test cases must pass:
   - TC002_FULL_KYC (full onboarding)
   - TC_HOME_FULL_BASIC, TC_HOME_FULL_BODYWEIGHT, TC_HOME_FULL_HEALTH_C
   - TC_MESO_001, TC_MESO_002, TC_MESO_003

**Validation (QA)** [code-reviewer]:
1. Screenshot or export the "Generate Report" node output
2. All tests must show PASS status
3. If any test fails, identify whether the failure is caused by the quality fixes or a pre-existing issue
4. Document any flaky tests separately

---

## 7. Dependency Graph

```
Phase 1 (Database)                    Phase 3 (Prompts)
===================                   =================
T-101 (cardio role)                   T-301 (dedup rules)      -- no deps
  |                                   T-302 (health prompts)   -- no deps
  +-> T-102 (update exercises)        T-303 (cardio prompt)    -- depends T-101
  +-> T-103 (set_profiles)

T-104 (push/pull cleanup)

          |
          v
Phase 2 (Workflow Logic)
========================
T-204 (verify health passthrough)     -- no deps
T-201 (dedup guard)                   -- no deps
  |
  +-> T-202 (volume cap)              -- depends T-201
T-203 (health SQL)                    -- depends T-204
T-205 (cardio in Code JS1)           -- depends T-101, T-103
  |
  +-> T-206 (cardio in validator)     -- depends T-205

          |
          v
Phase 4 (Testing)
=================
T-401 (health B test)                 -- depends T-201, T-203, T-302
T-402 (health C test)                 -- depends T-201, T-203, T-302
T-403 (dedup test)                    -- depends T-201, T-301
T-404 (volume cap test)              -- depends T-202
T-405 (cardio test)                  -- depends T-101-103, T-205-206, T-303
T-406 (regression)                   -- depends all
```

### Critical Path

```
T-101 -> T-103 -> T-205 -> T-206 -> T-405 -> T-406
```

Estimated duration: ~3 working days for all phases.

---

## 8. Rollback Plan

### Database Rollbacks (Phase 1)

Each migration should have a corresponding rollback script:

```sql
-- Rollback T-101: Remove cardio role
DELETE FROM exercise_role WHERE role = 'cardio';

-- Rollback T-102: Revert exercises to original roles
-- (requires storing original values before migration - use a temp table)
-- CREATE TABLE _backup_exercise_roles AS
--   SELECT exercise_id, role FROM exercises WHERE exercise_id IN (...);

-- Rollback T-103: Remove cardio set_profiles
DELETE FROM set_profiles WHERE role = 'cardio';

-- Rollback T-104: Revert push/pull fixes
-- (requires backup table created before migration)
```

**IMPORTANT**: Before running any Phase 1 migration, [pixel-dev] must create a backup:

```sql
CREATE TABLE _backup_pre_quality_fixes AS
SELECT exercise_id, role, pattern, main_muscle
FROM exercises;
```

### Workflow Rollbacks (Phase 2 + 3)

The current `WORKOUT_CREATOR.json` file is version-controlled in git. To rollback:

```bash
git checkout HEAD -- n8n/running_flows/WORKOUT_CREATOR.json
```

Then re-import the JSON into the n8n instance.

### Partial Rollback Strategy

Each fix is independent enough to be rolled back individually:
- **Fix 1 rollback**: Remove the dedup block from `Code in JavaScript` and the two new rules from the prompt
- **Fix 2 rollback**: Remove the volume cap block from `ValidateWorkoutDuration`
- **Fix 3 rollback**: Remove the health WHERE clauses from `GetExercisesByPattern` and revert prompt to CUIDADO warnings
- **Fix 4 rollback**: Run T-101/T-102/T-103 rollback SQL; revert `Code in JavaScript1` and `ValidateWorkoutDuration` cardio handling; remove cardio prompt section
- **Fix 5 rollback**: Restore from `_backup_pre_quality_fixes` table

---

## Appendix A: File Reference

| File | Path | Modified By |
|------|------|-------------|
| WORKOUT_CREATOR workflow | `n8n/running_flows/WORKOUT_CREATOR.json` | T-201, T-202, T-203, T-205, T-206, T-301, T-302, T-303 |
| Exercise role table | Supabase `exercise_role` | T-101 |
| Exercises table | Supabase `exercises` | T-102, T-104 |
| Set profiles table | Supabase `set_profiles` | T-103 |
| E2E test runner | `n8n/tests/GymRatFlow_E2E_TestRunner.json` | T-406 (run only, no modification) |
| Mesocycle test runner | `n8n/tests/MesocycleRenewal_E2E_TestRunner.json` | T-406 (run only, no modification) |

## Appendix B: Task Summary Table

| Ticket | Fix | Phase | Assignee | Depends On | Effort |
|--------|-----|-------|----------|------------|--------|
| T-101 | Fix 4 | 1 | [pixel-dev] | None | 15 min |
| T-102 | Fix 4 | 1 | [pixel-dev] | T-101 | 1 hour |
| T-103 | Fix 4 | 1 | [pixel-dev] | T-101 | 30 min |
| T-104 | Fix 5 | 1 | [pixel-dev] | None | 1-2 hours |
| T-201 | Fix 1 | 2 | [n8n-agent] | None | 30 min |
| T-202 | Fix 2 | 2 | [n8n-agent] | T-201 | 1 hour |
| T-203 | Fix 3 | 2 | [n8n-agent] | T-204 | 45 min |
| T-204 | Fix 3 | 2 | [n8n-agent] | None | 15 min |
| T-205 | Fix 4 | 2 | [n8n-agent] | T-101, T-103 | 30 min |
| T-206 | Fix 4 | 2 | [n8n-agent] | T-205 | 30 min |
| T-301 | Fix 1 | 3 | [n8n-agent] | None | 15 min |
| T-302 | Fix 3 | 3 | [n8n-agent] | None | 30 min |
| T-303 | Fix 4 | 3 | [n8n-agent] | T-101 | 20 min |
| T-401 | Test | 4 | [n8n-agent] | T-201, T-203, T-302 | 30 min |
| T-402 | Test | 4 | [n8n-agent] | T-201, T-203, T-302 | 30 min |
| T-403 | Test | 4 | [n8n-agent] | T-201, T-301 | 30 min |
| T-404 | Test | 4 | [n8n-agent] | T-202 | 20 min |
| T-405 | Test | 4 | [n8n-agent] | T-101-103, T-205-206, T-303 | 30 min |
| T-406 | Test | 4 | [n8n-agent] | All | 45 min |

**Total estimated effort**: ~10-12 hours across all phases.
