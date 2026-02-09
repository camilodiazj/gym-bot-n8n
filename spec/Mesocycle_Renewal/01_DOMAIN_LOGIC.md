# 01 - Domain Logic: Mesocycle Renewal

> **Feature**: Mesocycle Renewal
> **Status**: Specification
> **Author**: Lead Solutions Architect + kiro-coach
> **Last Updated**: 2026-02-08

---

## Overview

A **mesocycle** is a 4-week training block that forms the fundamental periodization unit in GymBot. When a user finishes all scheduled workouts in week 4, the system must detect completion, offer renewal options, and execute the chosen path to start a new mesocycle.

This document defines the business rules, domain constraints, and coaching rationale that govern the Mesocycle Renewal feature.

### Database Tables Involved

| Table | Key Columns | Role in Renewal |
|-------|-------------|-----------------|
| `users_plans` | `user_id`, `template_id`, `week_schedule`, `goal`, `level`, `status`, `mesocycle_number`, `last_renewal_date` | Tracks current plan metadata and mesocycle count |
| `user_weekly_schedule` | `user_id`, `week` (1-4), `week_day`, `planned_day`, `Completed` (boolean) | Calendar of scheduled sessions and their completion status |
| `workouts` | `user_id`, `week` (1-4), `day_name`, `exercise_id`, `sets`, `reps`, `rir`, `rest-seconds`, `tempo`, `exercise_order` | The actual exercise prescriptions per day per week |
| `exercises` | `exercise_id`, `spanish_name`, `pattern`, `role`, `main_muscle`, `secondary_muscles`, `level`, `equipment` | Exercise catalog (1657 entries) used for rotation lookups |
| `set_profiles` | `goal`, `level`, `week`, `role`, `sets`, `reps`, `rir`, `rest_sec`, `tempo` | Loading parameters that encode progressive overload per week |
| `week_schedules` | `schedule_type` (`fb_2`, `fb_3`, `ua_4`, `ppl_5`, `ppl_6`), `days_per_week` | Maps frequency to training split |

### Health Status Codes

| Code | Restriction | Impact on Renewal |
|------|-------------|-------------------|
| A | No restrictions | Full exercise selection during rotation |
| B | Lower body issues | Avoid high-impact on knees/ankles during rotation |
| C | Upper body issues | Avoid overhead pressing during rotation |
| D | Spine issues | Avoid heavy axial loading during rotation |
| E | Special condition | Prioritize machines, low-risk exercises during rotation |

---

## Section 1: Mesocycle Completion Detection Rules

### Rule MC-001: Completion Criteria

A mesocycle is considered **complete** when ALL of the following conditions are met:

1. **All week 4 sessions are marked as completed**: Every row in `user_weekly_schedule` where `week = 4` for the given `user_id` must have `Completed = true`.
2. **Session count validation**: The count of completed W4 sessions must be **>=** `days_per_week` from the user's `week_schedule` in `users_plans`. This guards against partial data.
3. **Detection context**: Completion is detected on the **FALSE branch** of `has_planned_workouts` -- meaning the user has no future planned sessions remaining. This is the natural trigger point: the user has finished everything.

**SQL reference for detection:**

```sql
-- Count completed W4 sessions
SELECT COUNT(*) AS completed_w4
FROM user_weekly_schedule
WHERE user_id = :user_id
  AND week = 4
  AND "Completed" = true;

-- Compare against required days
SELECT ws.days_per_week
FROM users_plans up
JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
WHERE up.user_id = :user_id;

-- mesocycle_complete = (completed_w4 >= days_per_week)
```

### Rule MC-002: Edge Cases

| Scenario | Result | Routing |
|----------|--------|---------|
| User completed 3 of 4 sessions in W4 | NOT complete | Normal scheduling flow (schedule remaining W4 sessions) |
| User has no workouts at all (`finishedWorkouts` is empty) | NOT complete | Normal scheduling flow (likely new user or data issue) |
| User manually says "renovar mesociclo" but has not finished W4 | Allow via **Path B** (manual intent) | Route to Renewal Agent subflow; agent handles conversation and may advise finishing W4 first |
| User completed W4 but still has W3 sessions unfinished | Complete (W4 criterion met) | Renewal flow triggers; incomplete W3 sessions are historical data |
| User's `days_per_week` changed mid-mesocycle (admin edit) | Use current `days_per_week` for comparison | May cause false positive/negative; admin should also adjust schedule |

### Rule MC-003: Idempotency

The renewal detection must be **idempotent** -- it must not trigger twice for the same mesocycle completion event.

**Idempotency mechanism:**

1. After renewal execution: `user_weekly_schedule` is cleared (all rows deleted for user).
2. On next message: `has_planned_workouts = FALSE` triggers `Check_Mesocycle_Complete`.
3. But W4 data has also been cleared, so `completed_w4 = 0`, which means `mesocycle_complete = FALSE`.
4. User is routed to normal scheduling flow (schedule W1 of new mesocycle).

This creates a natural guard: clearing the schedule data prevents re-triggering. No additional flags or locks are needed.

**Edge case**: If the schedule clear succeeds but the mesocycle_number increment fails (partial failure), the system would re-detect completion on the next message. This is acceptable because the renewal conversation would simply restart -- the user has not lost any data. See Section 6 (Data Integrity) for atomicity requirements.

---

## Section 2: Renewal Options (Business Rules)

### Rule RN-001: MANTENER_RUTINA (Maintain Routine)

**Intent**: User wants to keep the same exercises and training structure, relying on progressive overload through the `set_profiles` loading parameters.

**Operations (in order):**

1. `DELETE FROM user_weekly_schedule WHERE user_id = :user_id` -- Clear all scheduled sessions
2. `UPDATE users_plans SET mesocycle_number = mesocycle_number + 1, last_renewal_date = NOW() WHERE user_id = :user_id` -- Increment mesocycle counter
3. Keep ALL existing rows in `workouts` table unchanged

**Post-condition**: User must re-schedule week 1 via the normal scheduling flow on their next message. The scheduling agent will detect no planned workouts and offer to schedule.

**Progressive overload note**: Even though the exercises remain identical, the `set_profiles` table provides **different loading parameters per week** (sets, reps, RIR, rest, tempo). Week 1 of mesocycle N+1 uses the same Week 1 profile but the user should be stronger from the previous mesocycle. The coach may also update `set_profiles` over time to encode long-term progression.

### Rule RN-002: CAMBIAR_DIAS (Change Training Days)

**Intent**: User wants to change how many days per week they train (e.g., from 4 to 3 days), which requires a completely new routine with a different training split.

**Input validation:**

- `new_days_per_week` must be an integer between 2 and 6 (inclusive)
- Mapping: `2 -> fb_2`, `3 -> fb_3`, `4 -> ua_4`, `5 -> ppl_5`, `6 -> ppl_6`

**Operations (in order):**

1. `DELETE FROM workouts WHERE user_id = :user_id` -- Remove all current exercises
2. `DELETE FROM user_weekly_schedule WHERE user_id = :user_id` -- Clear schedule
3. `UPDATE users_plans SET week_schedule = :new_schedule, mesocycle_number = mesocycle_number + 1, last_renewal_date = NOW() WHERE user_id = :user_id` -- Update plan
4. Call `WORKOUT_CREATOR` with `is_renewal = true` and `override_days_available = :new_days_per_week` -- Generate completely new 4-week routine
5. New workouts are inserted by `WORKOUT_CREATOR`

**Rollback requirement**: If step 4 (WORKOUT_CREATOR) fails, steps 1-3 must be rolled back. See Rule DI-001.

### Rule RN-003: ROTAR_EJERCICIOS (Rotate Exercises)

**Intent**: User wants variety -- swap exercises for alternatives that target the same movement patterns, maintaining the existing training structure.

**Rotation algorithm:**

For each exercise in the user's current `workouts`:

1. Look up the exercise's `pattern` (e.g., `push`, `hip_hinge`, `pull`)
2. Look up the exercise's `role` (compound, isolation, core)
3. Query for an alternative:

```sql
SELECT exercise_id
FROM exercises
WHERE pattern = :current_pattern
  AND exercise_id != :current_exercise_id
  AND role = :current_role
  AND level IN (:appropriate_levels)
  AND equipment IN (:available_equipment)
  AND main_muscle NOT IN (:disliked_muscles)
ORDER BY RANDOM()
LIMIT 1;
```

4. If an alternative is found, update the workout row:

```sql
UPDATE workouts
SET exercise_id = :new_exercise_id
WHERE user_id = :user_id
  AND exercise_id = :current_exercise_id;
```

5. If NO alternative exists for a given pattern, **keep the current exercise** and continue to the next one.

**Rotation scope:**

- **Compound exercises**: Always rotate (these are the main lifts users want variety on)
- **Isolation exercises**: Always rotate
- **Core exercises**: Rotate only if alternatives exist; core exercise variety is typically limited

**Preserved fields**: `day_name`, `week`, `sets`, `reps`, `rir`, `rest-seconds`, `tempo`, `exercise_order` -- ALL remain unchanged. Only `exercise_id` is swapped.

**Post-rotation operations:**

1. Apply all exercise swaps (batch update)
2. `DELETE FROM user_weekly_schedule WHERE user_id = :user_id` -- Clear schedule
3. `UPDATE users_plans SET mesocycle_number = mesocycle_number + 1, last_renewal_date = NOW() WHERE user_id = :user_id`

### Rule RN-004: PREGUNTAR_OPCIONES (Ask Options)

**Intent**: User's message was ambiguous, or the system is proactively offering options after detecting mesocycle completion.

**Behavior:**

- Present the 3 renewal options in a friendly, numbered Spanish message
- No database changes
- This is purely a conversational intent -- the Renewal Agent re-engages the user and waits for their choice

**Example agent output:**

```
Felicidades por completar tu mesociclo! Tienes 3 opciones:

1. Mantener tu rutina actual (seguir con los mismos ejercicios)
2. Cambiar dias de entrenamiento (entrenar mas o menos dias por semana)
3. Rotar ejercicios (misma estructura, ejercicios nuevos)

Cual prefieres?
```

---

## Section 3: Intention Detection Rules

### Rule ID-001: Automatic Detection (Path A)

**Trigger conditions** (both must be true):

- `has_planned_workouts = FALSE` (user has no future scheduled sessions)
- `mesocycle_complete = TRUE` (all W4 sessions completed per Rule MC-001)

**Behavior:**

- No specific message content is required from the user
- The system **proactively** routes to the Renewal Agent
- Any message from the user (even "hola") will trigger the renewal flow if conditions are met
- The agent opens with congratulations and presents options (Rule RN-004 behavior)

**Flow location**: FALSE branch of `has_planned_workouts` check in MAIN_FLOW, after `Check_Mesocycle_Complete` evaluates to TRUE.

### Rule ID-002: Manual Detection (Path B)

**Trigger**: The `Intention_Agent` classifies the user's message as `RENOVAR_MESOCICLO`.

**Detection keywords and phrases:**

| Spanish Phrase | English Equivalent |
|----------------|-------------------|
| "cambiar rutina" | "change routine" |
| "nuevos ejercicios" | "new exercises" |
| "renovar" | "renew" |
| "rotar" | "rotate" |
| "nuevo ciclo" | "new cycle" |
| "cambiar dias" | "change days" |
| "quiero otra rutina" | "I want another routine" |
| "nueva rutina" | "new routine" |
| "cambiar plan" | "change plan" |

**Behavior:**

- Can trigger **even if** user has active planned workouts (user proactively wants to change)
- Requires `Fetch_Plan_For_Renewal` to retrieve current `days_per_week` and plan metadata
- Routes to Renewal Agent subflow
- If user has NOT completed W4, the agent should:
  - Acknowledge the request
  - Inform user they have remaining sessions in the current mesocycle
  - Offer to proceed anyway OR suggest completing first
  - If user insists, execute the chosen renewal option

### Rule ID-003: Intent Priority

The `Intention_Agent` must follow this priority order when classifying messages:

```
1. CONFIRMAR_RUTINA       (highest - completion confirmation)
2. RENOVAR_MESOCICLO      (renewal-related requests)
3. VER_RUTINA_DE_HOY      (view today's routine)
4. AGENDAR                (schedule workouts)
5. CHAT                   (lowest - general conversation)
```

**Disambiguation rules:**

| User Message | Correct Intent | Reasoning |
|-------------|---------------|-----------|
| "quiero cambiar mi rutina" | `RENOVAR_MESOCICLO` | Action request to modify routine |
| "que es un mesociclo?" | `CHAT` | Information question, not an action request |
| "ya termine mi mesociclo" | `RENOVAR_MESOCICLO` | Implies desire to proceed to next cycle |
| "como van mis ejercicios?" | `CHAT` | Question about progress, not a renewal request |
| "quiero entrenar 3 dias" | `RENOVAR_MESOCICLO` | Implies changing training frequency |
| "me aburri de mis ejercicios" | `RENOVAR_MESOCICLO` | Implies desire to rotate exercises |

---

## Section 4: Progressive Overload Model (Coaching Perspective)

### Rule PO-001: 4-Week Mesocycle Structure

The 4-week mesocycle follows established sports science periodization principles:

| Week | Phase | Purpose | Loading Characteristics |
|------|-------|---------|------------------------|
| 1 | **Adaptation** | Learn/re-learn movement patterns; establish baseline | Moderate load, higher RIR (2-3), standard tempo |
| 2 | **Loading** | Increase training stress | Increased intensity, moderate RIR (1-2) |
| 3 | **Overreach** | Peak volume and/or intensity | Highest volume, lowest RIR (0-1), potentially slower tempos |
| 4 | **Deload / Performance** | Recovery or testing | Reduced volume, moderate intensity, higher RIR |

The `set_profiles` table encodes this periodization by providing **different values for `sets`, `reps`, `rir`, `rest_sec`, and `tempo`** for each combination of `(goal, level, week, role)`. This means the same exercise prescription automatically adjusts difficulty across the 4-week block.

**Example progression for a compound exercise (Ganar masa muscular, Intermedio):**

| Week | Sets | Reps | RIR | Rest (sec) | Tempo |
|------|------|------|-----|------------|-------|
| 1 | 3 | 8-10 | 3 | 120 | 2-0-1-0 |
| 2 | 4 | 8-10 | 2 | 120 | 2-0-1-0 |
| 3 | 4 | 8-10 | 1 | 120 | 3-0-1-0 |
| 4 | 3 | 6-8 | 3 | 150 | 2-0-1-0 |

### Rule PO-002: When to Maintain vs Change

The Renewal Agent should **suggest** the most appropriate option based on context, but **always let the user decide**. The following guidelines inform the agent's recommendation:

**MANTENER_RUTINA is recommended when:**

- User is in their first 1-2 mesocycles (still learning exercises and building neural adaptations)
- User reports satisfaction with current exercises
- User is progressing well (increasing weight or reps)
- User's schedule has not changed
- Rationale: Repeating the same exercises allows neuromuscular adaptation and strength gains before introducing novelty

**CAMBIAR_DIAS is recommended when:**

- User explicitly mentions a schedule change ("ahora solo puedo 3 dias")
- Current frequency is causing burnout or fatigue
- User wants to increase training frequency as they advance
- User's life circumstances changed (work, school, travel)
- Rationale: Training frequency is a primary driver of results; matching it to the user's real availability improves adherence

**ROTAR_EJERCICIOS is recommended when:**

- User has completed 2-3 mesocycles with the same exercises
- User expresses boredom ("me aburri", "quiero algo diferente")
- User reports joint discomfort on specific exercises (rotation can find alternatives)
- User wants to address muscle imbalances
- Rationale: Exercise variety prevents staleness, reduces overuse injury risk, and can address weak points by targeting muscles from different angles

### Rule PO-003: Exercise Rotation Principles

When executing ROTAR_EJERCICIOS, the following sports science principles must be respected:

1. **Rotate by movement pattern**: Swap exercises within the same pattern category. A bench press (push) can be replaced with a dumbbell press (push), NOT with a row (pull). The `pattern` field in the `exercises` table enforces this.

2. **Maintain muscle balance**: The rotation must not shift the overall muscle emphasis of the program. Since exercises are swapped 1-for-1 within the same pattern, the push/pull/legs balance is preserved by design.

3. **Respect training environment**:
   - GYM users: Can access barbells, machines, cables, dumbbells
   - HOME users: Limited to their declared `home_equipment` (e.g., mancuernas, bandas, peso corporal)
   - The rotation query must filter by `equipment` compatible with the user's environment

4. **Respect health status restrictions**:
   - Code B: Exclude exercises flagged for knee/ankle impact
   - Code C: Exclude overhead pressing movements
   - Code D: Exclude heavy axial loading (e.g., back squat, deadlift)
   - Code E: Prefer machine-based alternatives
   - These restrictions apply during rotation just as they do during initial routine generation

5. **Respect disliked exercises**: Cross-reference the user's `disliked_exercises` field from `users_gym_profile`. Never rotate INTO a disliked exercise. The `disliked_muscles_en` processed field should be used to filter `main_muscle`.

6. **Level appropriateness**: Principiante users should prefer machine and guided exercises. Avanzado users can access the full catalog. The `level` field on exercises provides this filtering.

---

## Section 5: Conversation Flow (Agent Behavior)

### Rule CF-001: Renewal Agent System Prompt Requirements

The Renewal Agent is a specialized AI agent within the Mesocycle Renewal subflow. Its behavior is governed by these requirements:

**Language and tone:**

- Language: Spanish (Colombian dialect)
- Tone: Encouraging, celebratory -- the user just accomplished something significant
- Must congratulate the user on completing the mesocycle
- Use informal "tu" form, not "usted"
- Avoid technical jargon unless the user initiates it

**Presentation:**

- Present options as a clear, numbered list
- Be DIRECT: do not ask unnecessary follow-up questions about load, volume, or progression details
- If user says "mantener" -> execute immediately, do NOT ask "estas seguro?"
- If user says "rotar" -> execute immediately, do NOT ask which exercises to swap
- If user says a number (1, 2, 3) -> map to the corresponding option and execute
- If user says "cambiar dias" -> ask ONLY how many days per week (2-6), nothing else

**Memory configuration:**

- Uses Postgres chat memory (`n8n_chat_histories` table)
- Session key format: `{user_id}_mesocycle_renewal`
- Memory window: Last 10 messages (sufficient for the renewal conversation)
- Memory cleanup: `DELETE FROM n8n_chat_histories WHERE session_id = :session_key` after renewal is completed successfully

**Session lifecycle:**

```
1. User enters renewal flow (Path A or Path B)
2. Agent greets and presents options (or responds to user's explicit request)
3. User chooses option
4. Agent confirms choice with brief summary
5. System executes database operations
6. Agent confirms completion: "Listo! Tu nuevo mesociclo esta preparado."
7. Memory is cleaned up
8. User returns to MAIN_FLOW on next message
```

### Rule CF-002: Error Handling

| Error Scenario | Agent Behavior | System Behavior |
|---------------|----------------|-----------------|
| No alternative exercises found during ROTAR | Keep current exercise for that slot; inform user: "Algunos ejercicios no tienen alternativa disponible, asi que los mantuve." | Log which patterns had no alternatives |
| WORKOUT_CREATOR fails during CAMBIAR_DIAS | Inform user: "Hubo un problema generando tu nueva rutina. Tu rutina anterior sigue activa. Intenta de nuevo mas tarde." | Rollback: restore previous workouts, schedule, and plan state |
| User sends unrelated message during renewal | Agent redirects: "Entiendo, pero primero terminemos con tu renovacion. Cual opcion prefieres?" | No database changes; stay in renewal subflow |
| Database connection error | Agent: "Hay un problema temporal. Intenta de nuevo en unos minutos." | Retry logic at n8n workflow level; no partial commits |
| Invalid days_per_week (e.g., user says "10 dias") | Agent: "Puedes elegir entre 2 y 6 dias por semana. Cuantos dias quieres entrenar?" | No database changes; re-prompt |

---

## Section 6: Data Integrity Constraints

### Rule DI-001: Atomicity

Each renewal option involves multiple database operations that must succeed or fail as a unit:

**MANTENER_RUTINA (2 operations):**

```
BEGIN;
  DELETE FROM user_weekly_schedule WHERE user_id = :user_id;
  UPDATE users_plans
    SET mesocycle_number = mesocycle_number + 1,
        last_renewal_date = NOW()
    WHERE user_id = :user_id;
COMMIT;
```

**CAMBIAR_DIAS (4+ operations):**

```
-- Phase 1: Cleanup (transactional)
BEGIN;
  DELETE FROM workouts WHERE user_id = :user_id;
  DELETE FROM user_weekly_schedule WHERE user_id = :user_id;
  UPDATE users_plans
    SET week_schedule = :new_schedule,
        mesocycle_number = mesocycle_number + 1,
        last_renewal_date = NOW()
    WHERE user_id = :user_id;
COMMIT;

-- Phase 2: Generation (separate call)
-- Call WORKOUT_CREATOR with is_renewal=true
-- If this fails, Phase 1 must be rolled back
```

**Note on n8n limitations**: n8n does not natively support distributed transactions across multiple nodes. The rollback for CAMBIAR_DIAS Phase 2 failure must be implemented as a compensating transaction (re-insert old workouts from a backup query run before Phase 1).

**ROTAR_EJERCICIOS (batch update):**

```
BEGIN;
  -- All exercise swaps in a single UPDATE with CASE
  UPDATE workouts
  SET exercise_id = CASE
    WHEN exercise_id = :old_1 THEN :new_1
    WHEN exercise_id = :old_2 THEN :new_2
    ...
    ELSE exercise_id
  END
  WHERE user_id = :user_id;

  DELETE FROM user_weekly_schedule WHERE user_id = :user_id;
  UPDATE users_plans
    SET mesocycle_number = mesocycle_number + 1,
        last_renewal_date = NOW()
    WHERE user_id = :user_id;
COMMIT;
```

### Rule DI-002: Validation

| Constraint | Rule | Enforcement Point |
|-----------|------|-------------------|
| `days_per_week` range | Must be integer between 2 and 6 (inclusive) | Renewal Agent (conversational) + Code node (programmatic) |
| `week_schedule` validity | Must be one of: `fb_2`, `fb_3`, `ua_4`, `ppl_5`, `ppl_6` | Derived from `days_per_week` mapping; validated against `week_schedules` table |
| Exercise existence | Alternative `exercise_id` must exist in `exercises` table | JOIN constraint in rotation query |
| `mesocycle_number` monotonicity | Must always increment, never decrease | `SET mesocycle_number = mesocycle_number + 1` (never SET to absolute value) |
| User existence | `user_id` must exist in `users` table | Foreign key constraint; checked before renewal flow entry |
| Plan existence | `users_plans` row must exist for `user_id` | Checked in `Fetch_Plan_For_Renewal` node; if missing, route to onboarding |

### Rule DI-003: State Transitions

```
ACTIVE_MESOCYCLE
  |  (user has workouts in `workouts` table)
  |  (user has/had sessions in `user_weekly_schedule`)
  |
  |--- [W4 all completed (Path A)] ---+
  |--- [Manual "renovar" (Path B)] ---+
  |                                    |
  v                                    v
RENEWAL_CONVERSATION
  |  (Renewal Agent is talking to user)
  |  (no DB changes yet)
  |  (session_key: {user_id}_mesocycle_renewal)
  |
  |--- [User chooses MANTENER / CAMBIAR_DIAS / ROTAR]
  |
  v
RENEWAL_EXECUTING
  |  (DB operations in progress)
  |  (user should not send messages during this)
  |  (if they do, messages queue in WhatsApp)
  |
  |--- [All operations succeed] ----> NEW_MESOCYCLE_READY
  |--- [Operations fail] ----------> ACTIVE_MESOCYCLE (rollback)
  |
  v
NEW_MESOCYCLE_READY
  |  (mesocycle_number incremented)
  |  (user_weekly_schedule cleared)
  |  (chat memory cleaned up)
  |  (workouts: kept / replaced / rotated depending on option)
  |
  |--- [User sends next message]
  |--- [has_planned_workouts = FALSE]
  |--- [mesocycle_complete = FALSE (schedule cleared)]
  |--- [Normal scheduling flow activates]
  |
  v
ACTIVE_MESOCYCLE
  |  (user schedules W1 of new mesocycle)
  |  (cycle repeats)
```

**Important state invariants:**

- A user can only be in ONE state at a time
- `RENEWAL_CONVERSATION` has no timeout -- if the user abandons the conversation and comes back days later, the Renewal Agent resumes (chat memory persists until cleanup)
- `RENEWAL_EXECUTING` is transient -- it lasts only as long as the database operations take (typically < 5 seconds)
- There is no explicit state column in the database; state is inferred from the combination of `user_weekly_schedule` contents, `mesocycle_number`, and `n8n_chat_histories` session existence

---

## Appendix: Days-to-Schedule Mapping

| Days Per Week | Schedule Type | Split Name | Description |
|--------------|---------------|------------|-------------|
| 2 | `fb_2` | Full Body | 2x full body sessions per week |
| 3 | `fb_3` | Full Body | 3x full body sessions per week |
| 4 | `ua_4` | Upper/Lower (Arriba) | 2x upper + 2x lower per week |
| 5 | `ppl_5` | Push/Pull/Legs | Push, Pull, Legs, Upper, Lower |
| 6 | `ppl_6` | Push/Pull/Legs | 2x Push, 2x Pull, 2x Legs |

## Appendix: Glossary

| Term | Definition |
|------|------------|
| **Mesocycle** | A 4-week training block with structured progressive overload |
| **RIR** | Reps In Reserve -- how many reps the user could still perform before failure |
| **Tempo** | Eccentric-Pause-Concentric-Pause timing (e.g., 2-0-1-0 = 2s down, 0 pause, 1s up, 0 pause) |
| **Pattern** | Movement category (push, pull, hip_hinge, squat, etc.) |
| **Role** | Exercise classification: compound (multi-joint), isolation (single-joint), core |
| **Deload** | A planned reduction in training volume/intensity for recovery |
| **Progressive Overload** | Systematically increasing training demands over time to drive adaptation |
