# Mesocycle Renewal -- Architecture Specification

**Version:** 1.0
**Date:** 2026-02-08
**Author:** Lead Solutions Architect
**Status:** Draft

---

## Table of Contents

1. [System Architecture Diagram](#1-system-architecture-diagram)
2. [Database Schema](#2-database-schema)
3. [n8n Workflow Interface Contracts](#3-n8n-workflow-interface-contracts)
4. [Node Inventory](#4-node-inventory)
5. [Connection Rewiring Map](#5-connection-rewiring-map)
6. [Existing Node Modifications](#6-existing-node-modifications)

---

## 1. System Architecture Diagram

The mesocycle renewal feature integrates into GymBot through two activation paths and a shared subflow. Both paths converge on the same `GymBotMesocycleRenewal` subflow, which handles the multi-turn renewal conversation with the user.

### 1.1 High-Level Flow

```
                            USER MESSAGE (WhatsApp)
                                    |
                                    v
                         +--------------------+
                         |  MAIN_FLOW         |
                         |  WhatsApp_Trigger1 |
                         +--------+-----------+
                                  |
                                  v
                         +--------+-----------+
                         |  If (noise filter)  |
                         +--------+-----------+
                                  |
                                  v
                         +--------+-----------+
                         |  Normalize_Message  |
                         +--------+-----------+
                                  |
                                  v
                         +--------+-----------+
                         |  GetUser            |
                         +--------+-----------+
                                  |
                                  v
                         +--------+-----------+
                         |  user_exists        |
                         +----+--------+------+
                         TRUE |        | FALSE
                              v        v
                    +---------+--+   KYC Agent (onboarding)
                    | GetWeekly  |
                    | Schedule   |
                    +-----+------+
                          |
                          v
               +----------+-----------+
               | has_planned_workouts1 |
               +----+----------+------+
               TRUE |          | FALSE
                    |          |
          +---------+          +------------------+
          |                                       |
          v                                       v
   +-----------+              +-------------------------------------------+
   | PATH B    |              | PATH A: Automatic Detection               |
   | (Manual)  |              |                                           |
   |           |              | Week_Schedule ----+                       |
   v           |              | User_Finished  ---+--> Merge              |
   Filter_     |              | Workouts          |       |               |
   Today_      |              | Template_Days ----+       v               |
   Routine     |              |               Check_Mesocycle_Complete    |
   |           |              |                          |                |
   v           |              |                          v                |
   userHas     |              |               If_Mesocycle_Complete       |
   Routine     |              |                  /           \            |
   ForToday    |              |              TRUE             FALSE      |
   |           |              |                |                 |        |
   v           |              |                v                 v        |
  Intention_   |              |   Execute_Mesocycle_     AI Agent1        |
  Agent        |              |   Renewal (subflow)    (normal sched.)   |
   |           |              +-------------------------------------------+
   v           |
  Switch       |
   |           |
   +-----+----+----+----+
   |     |    |         |
   v     v    v         v
  CONF  CHAT VER    RENOVAR_MESOCICLO
  IRMAR      RUTINA      |
                          v
                   Fetch_Plan_For_Renewal
                          |
                          v
                   Execute_Mesocycle_Renewal_Manual
                          (subflow)
```

### 1.2 Path A -- Automatic Detection (No Scheduled Workouts Remaining)

When a returning user has no future planned workouts (`has_planned_workouts1` = FALSE), the system checks whether this is because they completed their 4-week mesocycle or simply haven't scheduled yet.

```
has_planned_workouts1 (FALSE output)
        |
        +---> Week_Schedule --------+
        |                           |
        +---> User_Finished  -------+--> Merge (3 inputs, chooseBranch)
        |     Workouts              |         |
        |                           |         v
        +---> Template_Days --------+  Check_Mesocycle_Complete (Code)
                                              |
                                              v
                                       If_Mesocycle_Complete (If)
                                        /                \
                                    TRUE                  FALSE
                                      |                     |
                                      v                     v
                          Execute_Mesocycle_          AI Agent1
                          Renewal (subflow)       (normal scheduling)
```

**Detection Logic** (`Check_Mesocycle_Complete` Code node):

1. Count completed workouts in week 4 from `User_Finished_Workouts`
2. Get total sessions per week from `Week_Schedule.days_per_week`
3. Compare: if `week4_completed >= days_per_week`, the mesocycle is complete

### 1.3 Path B -- Manual Trigger (User Requests Renewal)

When a user with active planned workouts sends a message that the `Intention_Agent` classifies as `RENOVAR_MESOCICLO`, the system fetches their plan details and invokes the renewal subflow.

```
has_planned_workouts1 (TRUE output)
        |
        v
  Filter_Today_Routine --> userHasRoutineForToday
        |
        v
  Intention_Agent --> Switch
        |                |
        v                +--> Output 3 (RENOVAR_MESOCICLO)
                                      |
                                      v
                              Fetch_Plan_For_Renewal (Postgres)
                                      |
                                      v
                              Execute_Mesocycle_Renewal_Manual
                                    (subflow)
```

### 1.4 Subflow Internals -- GymBotMesocycleRenewal

The subflow is an existing workflow (`n8n/wip/GymBotMesocycleRenewal.json`) with 20 nodes. It handles the multi-turn conversation for renewal.

```
Execute Workflow Trigger
    (user_id, full_name, whatsapp_id, phone_number_id, user_message, days_per_week)
        |
        v
  Renewal_Agent (AI - GPT-5.2)
  [Postgres Chat Memory: {user_id}_mesocycle_renewal]
        |
        v
  Parse_Intention (AI - GPT-4.1-mini)
        |
        v
  Switch_Intention
    |         |            |              |
    v         v            v              v
  MANTENER  CAMBIAR     ROTAR          PREGUNTAR
  _RUTINA   _DIAS       _EJERCICIOS    _OPCIONES
    |         |            |              |
    v         v            v              v
  Reset_    Extract_     Get_Current_   Send_Options
  Schedule  New_Days     Workouts       (WhatsApp)
    |         |            |
    v         v            v
  Increment Delete_Old   Loop_Rotate_Exercises
  Mesocycle Workouts       |
    |         |            v
    v         v          Get_Alternative_Exercises
  Send_     Execute_       |
  Confirm.  GymRatForm     v
  Mantener  (WORKOUT_    Select_Alternative
            CREATOR)       |
                           v
                         Update_Exercise --> (loop)
                           |
                           v (done)
                         Send_Confirm._Rotar
                           |
                           v
                         Reset_Schedule_Rotar
                           |
                           v
                         Increment_Mesocycle_Rotar
```

### 1.5 CAMBIAR_DIAS Chain (Subflow to WORKOUT_CREATOR)

When the user chooses to change their training frequency, the subflow invokes `WORKOUT_CREATOR` with renewal parameters.

```
Switch_Intention (CAMBIAR_DIAS)
        |
        v
  Extract_New_Days (Code)
  [Parses "3 dias" / "quiero 5" from user_message]
        |
        v
  Delete_Old_Workouts (Postgres)
  [DELETE FROM workouts WHERE user_id = ...]
        |
        v
  Execute_GymRatForm (Execute Workflow)
  [Calls WORKOUT_CREATOR with: whatsapp_id, is_renewal="true", override_days_available]
        |
        v
  WORKOUT_CREATOR (modified)
        |
        v
  If_Is_Renewal (If)
    /          \
  TRUE         FALSE
    |            |
    v            v
  Clear_Old_   (normal flow: CreateUser/CreatePlan)
  Workouts
    |
    v
  Clear_Old_
  Schedule
    |
    v
  UpdatePlan
  [UPDATE users_plans SET mesocycle_number = mesocycle_number + 1]
    |
    v
  (rejoin normal flow at Merge --> Loop Over Items)
```

---

## 2. Database Schema

### 2.1 Tables Involved in Mesocycle Renewal

No new tables or migrations are required. All necessary columns already exist.

| Table | Role in Renewal | Key Columns Used |
|-------|-----------------|------------------|
| `users` | User identity and contact info | `user_id` (UUID PK), `full_name`, `full_phone_number` |
| `users_plans` | Active plan tracking with mesocycle state | `plan_id`, `user_id`, `template_id`, `week_schedule`, `goal`, `level`, `status`, **`mesocycle_number`**, **`last_renewal_date`** |
| `user_weekly_schedule` | Scheduled workout sessions and completion status | `day_routine_id`, `user_id`, `week`, `session_name`, `planned_day`, `Completed` |
| `workouts` | User-assigned exercises per week/day | `id`, `user_id`, `week`, `day_name`, `exercise_id`, `sets`, `reps`, `rir`, `rest-seconds`, `tempo`, `exercise_order` |
| `exercises` | Exercise catalog for rotation lookups | `exercise_id`, `spanish_name`, `pattern`, `role`, `main_muscle`, `level`, `equipment` |
| `week_schedules` | Maps schedule_type to days_per_week | `schedule_type` (PK), `days_per_week` |
| `set_profiles` | Loading parameters by goal/level/week | `profile_id`, `goal`, `level`, `week`, `role`, `sets`, `reps`, `rir`, `rest_sec`, `tempo` |
| `routine_templates` | Template lookup for new plan creation | `template_id`, `week_schedule`, `goal`, `level`, `days_per_week`, `environment` |
| `template_days` | Day structure per schedule | `template_day_id`, `week_schedule`, `day_number`, `title` |

### 2.2 Mesocycle-Specific Columns on `users_plans`

These columns already exist in production. No migration needed.

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `mesocycle_number` | `INTEGER` | `1` | Tracks which mesocycle the user is on. Incremented on each renewal. |
| `last_renewal_date` | `TIMESTAMP WITH TIME ZONE` | `NULL` | Records when the last renewal occurred. Set to `NOW()` on renewal. |

### 2.3 Week Schedule Mapping

The `week_schedules` table maps `schedule_type` to training frequency. This mapping is used when the user changes days during renewal.

| `schedule_type` | `days_per_week` | Description |
|-----------------|-----------------|-------------|
| `fb_2` | 2 | Full Body, 2 days/week |
| `fb_3` | 3 | Full Body, 3 days/week |
| `ua_4` | 4 | Upper/Lower, 4 days/week |
| `ppl_5` | 5 | Push/Pull/Legs, 5 days/week |
| `ppl_6` | 6 | Push/Pull/Legs, 6 days/week |

### 2.4 SQL Operations by Renewal Intent

**MANTENER_RUTINA:**
```sql
-- 1. Clear schedule (keep workouts)
DELETE FROM user_weekly_schedule WHERE user_id = '{user_id}';

-- 2. Increment mesocycle
UPDATE users_plans
SET mesocycle_number = mesocycle_number + 1,
    last_renewal_date = NOW()
WHERE user_id = '{user_id}'
RETURNING mesocycle_number;
```

**CAMBIAR_DIAS:**
```sql
-- 1. Delete old workouts (subflow does this)
DELETE FROM workouts WHERE user_id = '{user_id}';

-- 2. WORKOUT_CREATOR regenerates everything with new days_per_week
-- 3. UpdatePlan in WORKOUT_CREATOR (renewal path):
UPDATE users_plans
SET mesocycle_number = mesocycle_number + 1,
    last_renewal_date = NOW(),
    week_schedule = '{new_week_schedule}',
    template_id = '{new_template_id}'
WHERE user_id = '{user_id}';
```

**ROTAR_EJERCICIOS:**
```sql
-- 1. For each exercise in week 1, swap to alternative with same pattern
UPDATE workouts
SET exercise_id = '{new_exercise_id}'
WHERE id = '{workout_id}';

-- 2. Clear schedule
DELETE FROM user_weekly_schedule WHERE user_id = '{user_id}';

-- 3. Increment mesocycle
UPDATE users_plans
SET mesocycle_number = mesocycle_number + 1,
    last_renewal_date = NOW()
WHERE user_id = '{user_id}';
```

---

## 3. n8n Workflow Interface Contracts

### 3.1 MAIN_FLOW --> GymBotMesocycleRenewal (Execute Workflow)

Both `Execute_Mesocycle_Renewal` (Path A) and `Execute_Mesocycle_Renewal_Manual` (Path B) call the same subflow with this contract.

**Input Schema (Execute Workflow Trigger inputs):**

```json
{
  "user_id": {
    "type": "string",
    "format": "UUID",
    "description": "Primary key from users table",
    "source": "$items('GetUser')[0].json.user_id",
    "required": true
  },
  "full_name": {
    "type": "string",
    "description": "User's display name for WhatsApp messages",
    "source": "$items('GetUser')[0].json.full_name",
    "required": true
  },
  "whatsapp_id": {
    "type": "string",
    "description": "User's WhatsApp phone number (e.g., '573001234567')",
    "source": "$items('If')[0].json.contacts[0].wa_id",
    "required": true
  },
  "phone_number_id": {
    "type": "string",
    "description": "WhatsApp Business API phone number ID for sending replies",
    "source": "$items('If')[0].json.metadata.phone_number_id",
    "required": true
  },
  "user_message": {
    "type": "string",
    "description": "The raw message text from the user",
    "source": "$items('Normalize_Message')[0].json.message_body",
    "required": true
  },
  "days_per_week": {
    "type": "number",
    "minimum": 2,
    "maximum": 6,
    "description": "Current training frequency from user's active plan",
    "source": "Fetched from users_plans JOIN week_schedules",
    "required": true
  }
}
```

**Example payload:**
```json
{
  "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "full_name": "Juan Perez",
  "whatsapp_id": "573001234567",
  "phone_number_id": "914510145083991",
  "user_message": "Ya termine las 4 semanas!",
  "days_per_week": 4
}
```

**Output Schema (returned to MAIN_FLOW):**

The subflow sends WhatsApp messages directly and does not return structured data to the caller. The Execute Workflow node should have `alwaysOutputData: true` set to prevent flow interruption.

### 3.2 GymBotMesocycleRenewal --> WORKOUT_CREATOR (for CAMBIAR_DIAS)

The renewal subflow calls `WORKOUT_CREATOR` via `Execute_GymRatForm` when the user wants to change their training frequency.

**Input Schema (Execute Workflow inputs):**

```json
{
  "whatsapp_id": {
    "type": "number",
    "description": "User's WhatsApp phone number (numeric, matching users_gym_profile.whatsapp_id)",
    "source": "$json.whatsapp_id (from Extract_New_Days)",
    "required": true
  },
  "is_renewal": {
    "type": "string",
    "enum": ["true", "false"],
    "description": "Flag indicating this is a mesocycle renewal, not a new user onboarding. String type because n8n workflow inputs don't support boolean.",
    "source": "Hardcoded as 'true'",
    "required": true
  },
  "override_days_available": {
    "type": "number",
    "minimum": 2,
    "maximum": 6,
    "description": "The new training frequency chosen by the user. Overrides the days_available value from users_gym_profile.",
    "source": "$json.new_days (from Extract_New_Days)",
    "required": true
  }
}
```

**Example payload:**
```json
{
  "whatsapp_id": 573001234567,
  "is_renewal": "true",
  "override_days_available": 3
}
```

### 3.3 Check_Mesocycle_Complete Output Schema

The `Check_Mesocycle_Complete` Code node analyzes data from the three Merge inputs and outputs a determination.

**Input data sources (available in the Code node via $items):**

| Source Node | Data | Access Pattern |
|-------------|------|----------------|
| `Week_Schedule` | `{ user_id, days_per_week }` | `$items('Week_Schedule')[0].json` |
| `User_Finished_Workouts` | Array of completed schedule entries | `$('User_Finished_Workouts').all()` |
| `Template_Days` | Array of `{ title }` for each day in the template | `$('Template_Days').all()` |

**Output schema:**

```json
{
  "mesocycle_complete": {
    "type": "boolean",
    "description": "True if the user has completed all sessions in week 4"
  },
  "week4_completed": {
    "type": "number",
    "description": "Count of week 4 sessions marked as Completed=true"
  },
  "days_per_week": {
    "type": "number",
    "description": "Total sessions expected per week (from week_schedules)"
  },
  "user_id": {
    "type": "string",
    "format": "UUID",
    "description": "Passed through from Week_Schedule for downstream use"
  }
}
```

**Example output:**
```json
{
  "mesocycle_complete": true,
  "week4_completed": 4,
  "days_per_week": 4,
  "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

**Detection algorithm (pseudocode):**
```
finished = User_Finished_Workouts.filter(w => w.week == 4 && w.Completed == true)
week4_completed = finished.length
days_per_week = Week_Schedule[0].days_per_week
mesocycle_complete = (week4_completed >= days_per_week)
```

### 3.4 Fetch_Plan_For_Renewal Output Schema (Path B)

This Postgres node fetches the user's current plan details for the manual renewal trigger.

**SQL Query:**
```sql
SELECT up.user_id, up.plan_id, up.mesocycle_number,
       ws.days_per_week, ws.schedule_type
FROM users_plans up
JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
WHERE up.user_id = '{user_id}'
AND up.status = 'active';
```

**Output schema:**
```json
{
  "user_id": "UUID",
  "plan_id": "UUID",
  "mesocycle_number": "number",
  "days_per_week": "number",
  "schedule_type": "string"
}
```

---

## 4. Node Inventory

### 4.1 MAIN_FLOW.json -- New Nodes

| # | Node Name | n8n Type | typeVersion | Position (after) | Purpose | executeOnce |
|---|-----------|----------|-------------|------------------|---------|-------------|
| 1 | `Check_Mesocycle_Complete` | `n8n-nodes-base.code` | 2 | After `Merge` (replaces direct connection to `AI Agent1`) | JavaScript Code node. Reads data from `Week_Schedule`, `User_Finished_Workouts`, and `Template_Days` via the Merge. Computes whether week 4 is fully completed. Outputs `{ mesocycle_complete, week4_completed, days_per_week, user_id }`. | true |
| 2 | `If_Mesocycle_Complete` | `n8n-nodes-base.if` | 2.3 | After `Check_Mesocycle_Complete` | Condition: `{{ $json.mesocycle_complete }}` equals `true`. TRUE branch goes to renewal subflow. FALSE branch goes to `AI Agent1` (normal scheduling). | true |
| 3 | `Execute_Mesocycle_Renewal` | `n8n-nodes-base.executeWorkflow` | 1.3 | After `If_Mesocycle_Complete` TRUE output (Path A) | Calls `GymBotMesocycleRenewal` subflow with 6 parameters. `alwaysOutputData: true`. | -- |
| 4 | `Fetch_Plan_For_Renewal` | `n8n-nodes-base.postgres` | 2.6 | After `Switch` output 3 (RENOVAR_MESOCICLO) | Postgres query to fetch `days_per_week` and plan details for Path B. Uses Supabase Memory credentials. | true |
| 5 | `Execute_Mesocycle_Renewal_Manual` | `n8n-nodes-base.executeWorkflow` | 1.3 | After `Fetch_Plan_For_Renewal` (Path B) | Calls same `GymBotMesocycleRenewal` subflow with 6 parameters. `alwaysOutputData: true`. | -- |

### 4.2 WORKOUT_CREATOR.json -- New Nodes

| # | Node Name | n8n Type | typeVersion | Position (after) | Purpose | executeOnce |
|---|-----------|----------|-------------|------------------|---------|-------------|
| 1 | `If_Is_Renewal` | `n8n-nodes-base.if` | 2.3 | After `ProcessUserPreferences`, before `LoadProfile` | Condition: `{{ $items('input')[0].json.is_renewal }}` equals `"true"`. TRUE branch goes to cleanup nodes. FALSE branch continues normal flow (`LoadProfile`). | true |
| 2 | `Clear_Old_Workouts` | `n8n-nodes-base.postgres` | 2.6 | After `If_Is_Renewal` TRUE output | `DELETE FROM workouts WHERE user_id = '{user_id}'`. Uses Supabase Memory credentials. | true |
| 3 | `Clear_Old_Schedule` | `n8n-nodes-base.postgres` | 2.6 | After `Clear_Old_Workouts` | `DELETE FROM user_weekly_schedule WHERE user_id = '{user_id}'`. Uses Supabase Memory credentials. | true |
| 4 | `UpdatePlan` | `n8n-nodes-base.postgres` | 2.6 | After `Clear_Old_Schedule` | `UPDATE users_plans SET mesocycle_number = mesocycle_number + 1, last_renewal_date = NOW(), week_schedule = '{new_schedule}', template_id = '{new_template}' WHERE user_id = '{user_id}'`. Uses Supabase Memory credentials. | true |

### 4.3 GymBotMesocycleRenewal.json -- Existing Nodes (No Changes)

The subflow is already built with 20 nodes. It is moved from `n8n/wip/` to `n8n/running_flows/` as-is, with only the `Execute_GymRatForm` node needing its `workflowId` updated to reference the production WORKOUT_CREATOR workflow ID.

| Node Name | Type | Purpose |
|-----------|------|---------|
| `Execute Workflow Trigger` | executeWorkflowTrigger | Entry point, accepts 6 parameters |
| `Renewal_Agent` | agent (GPT-5.2) | Multi-turn conversation with user about renewal options |
| `Parse_Intention` | agent (GPT-4.1-mini) | Classifies renewal response into 4 intents |
| `Switch_Intention` | switch | Routes to 4 branches |
| `Reset_Schedule` | postgres | Clears schedule for MANTENER |
| `Increment_Mesocycle` | postgres | Bumps mesocycle_number for MANTENER |
| `Send_Confirmation_Mantener` | whatsApp | Confirms MANTENER to user |
| `Extract_New_Days` | code | Parses desired day count from message |
| `Delete_Old_Workouts` | postgres | Removes workouts for CAMBIAR |
| `Execute_GymRatForm` | executeWorkflow | Calls WORKOUT_CREATOR with renewal params |
| `Get_Current_Workouts` | postgres | Fetches week 1 exercises for ROTAR |
| `Loop_Rotate_Exercises` | splitInBatches | Iterates over exercises |
| `Get_Alternative_Exercises` | supabase | Finds alternatives by pattern |
| `Select_Alternative` | code | Randomly picks a different exercise |
| `Update_Exercise` | postgres | Swaps exercise_id in workouts |
| `Send_Confirmation_Rotar` | whatsApp | Confirms ROTAR to user |
| `Reset_Schedule_Rotar` | postgres | Clears schedule for ROTAR |
| `Increment_Mesocycle_Rotar` | postgres | Bumps mesocycle_number for ROTAR |
| `Send_Options` | whatsApp | Sends options menu for PREGUNTAR |
| `Postgres Chat Memory` | memoryPostgresChat | Session: `{user_id}_mesocycle_renewal` |
| `OpenAI Chat Model` | lmChatOpenAi | GPT-5.2 for Renewal_Agent |
| `OpenAI Chat Model Mini` | lmChatOpenAi | GPT-4.1-mini for Parse_Intention |

---

## 5. Connection Rewiring Map

### 5.1 MAIN_FLOW.json -- Connection Changes

#### 5.1.1 Path A: Automatic Detection (Rewire Merge output)

**BEFORE:**
```
Merge --> AI Agent1
```

**AFTER:**
```
Merge --> Check_Mesocycle_Complete --> If_Mesocycle_Complete
                                          |
                                    TRUE: Execute_Mesocycle_Renewal (subflow)
                                    FALSE: AI Agent1
```

| # | From Node | From Output | To Node | To Input | Action |
|---|-----------|-------------|---------|----------|--------|
| 1 | `Merge` | main[0] | `AI Agent1` | main[0] | **REMOVE** this connection |
| 2 | `Merge` | main[0] | `Check_Mesocycle_Complete` | main[0] | **ADD** new connection |
| 3 | `Check_Mesocycle_Complete` | main[0] | `If_Mesocycle_Complete` | main[0] | **ADD** new connection |
| 4 | `If_Mesocycle_Complete` | main[0] (TRUE) | `Execute_Mesocycle_Renewal` | main[0] | **ADD** new connection |
| 5 | `If_Mesocycle_Complete` | main[1] (FALSE) | `AI Agent1` | main[0] | **ADD** new connection |

#### 5.1.2 Path B: Manual Trigger (Add new Switch output)

**BEFORE (Switch has 3 outputs):**
```
Switch output 0 (CONFIRMAR_RUTINA) --> CONFIRMATION AGENT
Switch output 1 (CHAT)             --> AI Agent
Switch output 2 (VER_RUTINA_DE_HOY) --> AI Agent
```

**AFTER (Switch has 4 outputs):**
```
Switch output 0 (CONFIRMAR_RUTINA)  --> CONFIRMATION AGENT  (unchanged)
Switch output 1 (CHAT)              --> AI Agent             (unchanged)
Switch output 2 (VER_RUTINA_DE_HOY) --> AI Agent             (unchanged)
Switch output 3 (RENOVAR_MESOCICLO) --> Fetch_Plan_For_Renewal --> Execute_Mesocycle_Renewal_Manual
```

| # | From Node | From Output | To Node | To Input | Action |
|---|-----------|-------------|---------|----------|--------|
| 6 | `Switch` | main[3] (RENOVAR_MESOCICLO) | `Fetch_Plan_For_Renewal` | main[0] | **ADD** new connection |
| 7 | `Fetch_Plan_For_Renewal` | main[0] | `Execute_Mesocycle_Renewal_Manual` | main[0] | **ADD** new connection |

#### 5.1.3 Upstream Connections (Unchanged)

These existing connections remain as-is:

| From | To | Status |
|------|----|--------|
| `has_planned_workouts1` TRUE | `Filter_Today_Routine` | KEEP |
| `has_planned_workouts1` FALSE | `Week_Schedule`, `User_Finished_Workouts`, `Template_Days` | KEEP |
| `Week_Schedule` | `Merge` input 0 | KEEP |
| `User_Finished_Workouts` | `Merge` input 1 | KEEP |
| `Template_Days` | `Merge` input 2 | KEEP |
| `Filter_Today_Routine` | `userHasRoutineForToday` | KEEP |
| `userHasRoutineForToday` TRUE | `Intention_Agent` | KEEP |
| `Intention_Agent` | `Switch` | KEEP |

### 5.2 WORKOUT_CREATOR.json -- Connection Changes

**BEFORE:**
```
input --> GetUserProfile --> ProcessUserPreferences --> LoadProfile --> Get_Day_Requirements --> GetUser --> UserExists
                                                                                                  |
                                                                                              TRUE: CreatePlan --> Merge
                                                                                             FALSE: CreateUser --> GetUser
```

**AFTER:**
```
input --> GetUserProfile --> ProcessUserPreferences --> If_Is_Renewal
                                                         |
                                                   TRUE: Clear_Old_Workouts --> Clear_Old_Schedule --> UpdatePlan --> LoadProfile
                                                   FALSE: LoadProfile
                                                         |
                                            (both paths rejoin at LoadProfile --> Get_Day_Requirements --> ...)
```

| # | From Node | From Output | To Node | To Input | Action |
|---|-----------|-------------|---------|----------|--------|
| 1 | `ProcessUserPreferences` | main[0] | `LoadProfile` | main[0] | **REMOVE** this connection |
| 2 | `ProcessUserPreferences` | main[0] | `If_Is_Renewal` | main[0] | **ADD** new connection |
| 3 | `If_Is_Renewal` | main[0] (TRUE) | `Clear_Old_Workouts` | main[0] | **ADD** new connection |
| 4 | `If_Is_Renewal` | main[1] (FALSE) | `LoadProfile` | main[0] | **ADD** new connection |
| 5 | `Clear_Old_Workouts` | main[0] | `Clear_Old_Schedule` | main[0] | **ADD** new connection |
| 6 | `Clear_Old_Schedule` | main[0] | `UpdatePlan` | main[0] | **ADD** new connection |
| 7 | `UpdatePlan` | main[0] | `LoadProfile` | main[0] | **ADD** new connection |

**Note:** On the renewal path, the flow skips `GetUser`, `UserExists`, `CreateUser`, and `CreatePlan` because the user and plan already exist. The `UpdatePlan` node updates the existing plan record instead of creating a new one. After `UpdatePlan`, the flow joins back at `LoadProfile` and continues through the normal exercise generation pipeline.

### 5.3 GymBotMesocycleRenewal.json -- Internal Connections (Existing, No Changes)

All internal connections are already wired in the subflow. The only configuration change needed is updating the `workflowId` in `Execute_GymRatForm` to point to the production `WORKOUT_CREATOR` workflow ID.

```
Execute Workflow Trigger --> Renewal_Agent --> Parse_Intention --> Switch_Intention
  |-- MANTENER_RUTINA  --> Reset_Schedule --> Increment_Mesocycle --> Send_Confirmation_Mantener
  |-- CAMBIAR_DIAS     --> Extract_New_Days --> Delete_Old_Workouts --> Execute_GymRatForm
  |-- ROTAR_EJERCICIOS --> Get_Current_Workouts --> Loop_Rotate_Exercises
  |                            |-- (each) --> Get_Alternative --> Select_Alternative --> Update_Exercise --> (loop)
  |                            |-- (done) --> Reset_Schedule_Rotar --> Increment_Mesocycle_Rotar
  |                                                                        |
  |                                                                        v
  |                                                              Send_Confirmation_Rotar
  |-- PREGUNTAR_OPCIONES --> Send_Options
```

---

## 6. Existing Node Modifications

### 6.1 MAIN_FLOW: Intention_Agent

**Current system prompt (relevant excerpt):**
```
INTENCIONES VALIDAS:
- VER_RUTINA_DE_HOY
- CHAT
```

**Modified system prompt:**
```
INTENCIONES VALIDAS:
- VER_RUTINA_DE_HOY
- CHAT
- RENOVAR_MESOCICLO

VER_RUTINA_DE_HOY: El usuario quiere ver su rutina/entrenamiento del dia.
Ejemplos: "Muestrame mi rutina", "Que me toca hoy", "Mi entrenamiento", "Dame mi workout"

RENOVAR_MESOCICLO: El usuario quiere renovar, cambiar o reiniciar su plan de entrenamiento
de 4 semanas (mesociclo). Quiere empezar un nuevo ciclo, cambiar su rutina, o rotar ejercicios.
Ejemplos: "Quiero cambiar mi rutina", "Ya termine las 4 semanas", "Quiero renovar mi plan",
"Quiero rotar ejercicios", "Quiero cambiar de dias", "Necesito una rutina nueva",
"Quiero empezar otro ciclo", "Ya acabe el mesociclo", "Quiero mantener mi rutina pero
empezar de nuevo"

CHAT: Cualquier otra pregunta, comentario o conversacion general sobre fitness.
Ejemplos: "Que ejercicio es mejor para biceps", "Hola", "Gracias"

NOTA: Las confirmaciones de rutina completada se manejan por otro flujo (pending_tasks).
Si el usuario dice que termino su rutina, responde CHAT.

Retorna SOLO la intencion (VER_RUTINA_DE_HOY, CHAT o RENOVAR_MESOCICLO), sin explicacion adicional.
```

**Changes summary:**
- Add `RENOVAR_MESOCICLO` to valid intents list
- Add description and examples for the new intent
- Update closing instruction to include the new intent in the valid return values

### 6.2 MAIN_FLOW: Switch Node

**Current configuration (3 outputs):**

| Output | Condition | Output Key |
|--------|-----------|------------|
| 0 | `$json.output.trim()` == `CONFIRMAR_RUTINA` | `ROUTINE_CONFIRMATION` |
| 1 | `$json.output.trim()` == `CHAT` | `CHAT` |
| 2 | `$json.output.trim()` == `VER_RUTINA_DE_HOY` | `VER_RUTINA_DE_HOY` |

**Modified configuration (4 outputs):**

| Output | Condition | Output Key | Action |
|--------|-----------|------------|--------|
| 0 | `$json.output.trim()` == `CONFIRMAR_RUTINA` | `ROUTINE_CONFIRMATION` | KEEP |
| 1 | `$json.output.trim()` == `CHAT` | `CHAT` | KEEP |
| 2 | `$json.output.trim()` == `VER_RUTINA_DE_HOY` | `VER_RUTINA_DE_HOY` | KEEP |
| 3 | `$json.output.trim()` == `RENOVAR_MESOCICLO` | `RENOVAR_MESOCICLO` | **ADD** |

**New output 3 JSON configuration:**
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
        "id": "new-renewal-condition-id",
        "leftValue": "={{ $json.output.trim() }}",
        "rightValue": "RENOVAR_MESOCICLO",
        "operator": {
          "type": "string",
          "operation": "equals"
        }
      }
    ],
    "combinator": "and"
  },
  "renameOutput": true,
  "outputKey": "RENOVAR_MESOCICLO"
}
```

### 6.3 WORKOUT_CREATOR: `input` Trigger Node

**Current input parameters:**
```json
{
  "workflowInputs": {
    "values": [
      { "name": "whatsapp_id", "type": "number" }
    ]
  }
}
```

**Modified input parameters:**
```json
{
  "workflowInputs": {
    "values": [
      { "name": "whatsapp_id", "type": "number" },
      { "name": "is_renewal", "type": "string" },
      { "name": "override_days_available", "type": "number" }
    ]
  }
}
```

**Notes:**
- `is_renewal` is a string (`"true"` / `"false"`) because n8n workflow trigger inputs do not support boolean type. For normal (non-renewal) calls, this field will be absent or empty, which evaluates to falsy.
- `override_days_available` is only meaningful when `is_renewal` is `"true"`. For normal calls, this field will be absent or `0`.

### 6.4 WORKOUT_CREATOR: `ProcessUserPreferences` Code Node

**Current behavior:**
- Reads `profile.days_available` from `GetUserProfile` result
- Passes it through as `days_available` in the output

**Modification -- add override logic at the end of the Code node:**

```javascript
// === OVERRIDE FOR RENEWAL ===
// If this is a renewal with changed days, override the days_available
const inputData = $items('input')[0].json;
if (inputData.is_renewal === 'true' && inputData.override_days_available) {
  profile.days_available = inputData.override_days_available;
}
```

This must be inserted **before** the `return` statement in the existing `ProcessUserPreferences` Code node. It overrides the `days_available` field from the user profile with the new value chosen during renewal, ensuring the downstream `Get_Day_Requirements` and `LoadProfile` queries use the correct frequency.

### 6.5 GymBotMesocycleRenewal: `Execute_GymRatForm` Node

**Current configuration:**
```json
{
  "workflowId": {
    "__rl": true,
    "value": "GYMRATFORM_V3_ID",
    "mode": "id"
  }
}
```

**Required change:**
- Update `"value"` from `"GYMRATFORM_V3_ID"` to the actual production workflow ID of `WORKOUT_CREATOR.json` after import.
- This is a deployment-time configuration step, not a code change.

### 6.6 GymBotMesocycleRenewal: Renewal_Agent System Prompt

**Current behavior:**
- References "FitBot" as the agent name

**Required change:**
- Replace `"FitBot"` with `"Kairos Personal Trainer"` to maintain brand consistency with all other agents in the system.
- This is a minor text change in the system prompt string of the `Renewal_Agent` node.

---

## Appendix A: Risk Matrix

| Risk | Impact | Mitigation |
|------|--------|------------|
| `Check_Mesocycle_Complete` false positive (user just hasn't scheduled yet, not at week 4) | User shown renewal flow prematurely | Algorithm checks specifically for week 4 completed sessions, not just absence of schedule |
| `ROTAR_EJERCICIOS` selects same exercise (no alternatives for that pattern) | No change in routine | `Select_Alternative` code falls back to keeping the current exercise if no alternatives exist |
| `CAMBIAR_DIAS` with invalid day count (e.g., 7 or 1) | SQL errors or no matching template | `Extract_New_Days` code only accepts values 2-6; if none parsed, `new_days` is null and flow should handle gracefully |
| Multi-turn conversation state lost | User has to restart renewal | Postgres Chat Memory with session key `{user_id}_mesocycle_renewal` persists conversation across messages |
| WORKOUT_CREATOR called in renewal mode but user/plan doesn't exist | Null pointer errors | `If_Is_Renewal` gate ensures renewal path only executes when `is_renewal` is explicitly `"true"` |

## Appendix B: Testing Considerations

New E2E test cases should be added to validate:

| Test ID | Category | Description | Fixture User |
|---------|----------|-------------|--------------|
| TC_RENEW_001 | MESOCYCLE_AUTO | User with all W4 completed triggers automatic renewal | New fixture |
| TC_RENEW_002 | MESOCYCLE_MANUAL | User sends "Quiero renovar mi plan" triggers manual renewal | Existing user with active plan |
| TC_RENEW_003 | MANTENER | User chooses option 1 (maintain) during renewal | Reuse TC_RENEW_001 user |
| TC_RENEW_004 | CAMBIAR | User chooses option 2 (change to 3 days) during renewal | Reuse TC_RENEW_001 user |
| TC_RENEW_005 | ROTAR | User chooses option 3 (rotate exercises) during renewal | Reuse TC_RENEW_001 user |
| TC_RENEW_006 | NO_RENEWAL | User without W4 completed sees normal scheduling (not renewal) | Existing fixture |

---

*End of Architecture Specification*
