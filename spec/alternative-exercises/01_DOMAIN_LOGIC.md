# 01_DOMAIN_LOGIC.md - Alternative Exercises (Dynamic Lookup)

## 1. Business Rules

### BR-1: Pattern Matching

An alternative exercise MUST share the same `pattern` as the primary exercise.

| Primary Pattern | Valid Alternatives | Invalid Alternatives |
|----------------|-------------------|---------------------|
| `squat` | Goblet Squat, Hack Squat, Front Squat | Bench Press (`push_h`) |
| `push_h` | Dumbbell Press, Machine Chest Press | Pull-up (`pull_v`) |
| `hinge` | Romanian Deadlift, Good Morning | Leg Press (`squat`) |

**Available patterns** (10): `squat`, `hinge`, `push_h`, `push_v`, `pull_h`, `pull_v`, `lunge`, `core`, `arm`, `accessory`.

Query: `SELECT FROM exercises WHERE pattern = <same> AND exercise_id != <primary>`.

---

### BR-2: Alternative Count

Each primary exercise gets **up to 2** alternatives. If fewer than 2 valid alternatives exist for a pattern, return whatever is available (0 or 1).

The frontend supports N alternatives with cyclic navigation.

---

### BR-3: Deterministic vs Random Selection

Alternatives are selected with `ORDER BY RANDOM() LIMIT 2`. This means alternatives may differ between page refreshes.

**Acceptable because**:
- Alternatives are suggestions, not prescriptions
- The user commits to one by completing a set (BR-5)
- Once a set is recorded in `set_values`, the exercise choice is persisted

**Future improvement**: Seed the random with user_id + date for daily consistency (not required for v1).

---

### BR-4: Set Profile Application

Alternative exercises receive set/rep/RIR/rest parameters from `set_profiles` based on the **alternative's own `role`**.

| Scenario | Primary Role | Alternative Role | Set Profile Used |
|----------|-------------|-----------------|-----------------|
| Same | compound | compound | Same parameters |
| Different | compound | isolation | Different parameters |

Lookup: `set_profiles WHERE goal = X AND level = Y AND week = Z AND role = <alternative_role>`

This means an alternative may have different sets, reps, RIR, and rest than the primary.

---

### BR-5: Set Commitment Rule

When a user completes **any set** of an exercise (primary or alternative), they are committed to that exercise for the session.

- **Frontend**: Hides the "Ver alternativa" button once any set is completed (already implemented)
- **Backend**: Stateless. Does not enforce commitment. The `set_values` rows record which `exercise_id` was actually used.

---

### BR-6: Historical Weight Pre-loading

Alternative exercise sets pre-load the user's last recorded weight for that specific `exercise_id`.

Logic (identical to primary, uses existing `GetLastWeightsForExercise`):
1. Query `set_values` for `(user_id, exercise_id)` ordered by `recorded_at DESC`
2. `DISTINCT ON (set_number)` gets most recent weight per set
3. If weight found for set N → use it
4. If no weight for set N but weight for set 1 exists → use set 1's weight
5. If no weight history → `kg = "-"`

---

### BR-7: No Duplicate Alternatives

Within a single exercise's alternatives list, no duplicate `exercise_id` should appear. Enforced by `DISTINCT` or `GROUP BY` in the query.

Additionally, a primary exercise should NOT appear as an alternative for another primary in the same session. This requires collecting all primary `exercise_id`s first and excluding them from the alternatives query.

---

### BR-8: Exclusion of Primary Exercise

The alternatives query MUST exclude the primary `exercise_id`:

```sql
WHERE exercise_id != '<primary_exercise_id>'
```

---

## 2. Data Integrity

### DI-1: set_values for Alternatives

`set_values` rows for alternative exercises use:
- `workout_id` = the original `workouts.id` (the workout slot)
- `exercise_id` = the alternative's `exercise_id`
- `set_number` = sequential within the alternative's sets

Unique constraint: `(workout_id, exercise_id, set_number)` prevents duplicates.

### DI-2: Determining Which Exercise Was Performed

The source of truth is `set_values`:
- If rows exist with the primary `exercise_id` → user did the primary
- If rows exist with an alternative `exercise_id` → user did the alternative
- If no rows → user didn't perform that slot

### DI-3: No Orphaned Data

Since alternatives are derived dynamically (not stored), there are no orphan rows to clean up during mesocycle renewal or workout deletion.

---

## 3. Training Science Rationale

### Why Pattern-Based Alternatives?

Movement pattern matching ensures alternatives target the same primary movers:
- `squat` pattern → always hits quads and glutes
- `hinge` pattern → always hits posterior chain
- `push_h` pattern → always hits chest and triceps

This maintains the mesocycle's volume distribution and progressive overload intent regardless of which specific exercise the user performs.

### Why Different Set Profiles by Role?

A `squat` pattern can include both compound (Barbell Squat, 4x6-8) and isolation (Leg Extension, 3x10-12) exercises. Using the alternative's own role for set_profiles ensures appropriate loading:
- Compounds → heavier, fewer reps, longer rest
- Isolation → lighter, more reps, shorter rest

---

## 4. Edge Cases

### EC-1: Zero Alternatives for a Pattern

**Scenario**: Pattern `accessory` might have very few exercises after filtering.

**Behavior**: `alternativeExercises` field is omitted from JSON (via `omitempty`). Frontend shows no alternative button.

### EC-2: Alternative Has No Weight History

**Scenario**: User has never done the alternative exercise.

**Behavior**: All sets show `kg: "-"`. User enters weight manually. Next session pre-loads this weight.

### EC-3: User Does Alternative in Week 1, Primary in Week 2

**Behavior**: Independent `set_values` histories. Week 2 primary loads primary's history; it doesn't mix with the alternative's.

### EC-4: set_profiles Missing for Alternative's Role

**Scenario**: Alternative has role `cardio` but no `set_profiles` row exists for `(goal, level, week, cardio)`.

**Behavior**: Fall back to the primary exercise's set_profiles parameters. Log a warning.

### EC-5: Same Exercise Appears as Primary AND Alternative

**Scenario**: Exercise A is primary in slot 1. Exercise A matches the pattern for slot 2.

**Behavior**: BR-7 prevents this by excluding all primary exercise_ids from alternatives queries.
