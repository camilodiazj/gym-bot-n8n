# Architecture: Schedule Duplicate Fix

## Problem Statement

When a new user completes onboarding and enters the scheduling flow, duplicate `user_weekly_schedule` rows are created for the first training day. This causes:

1. **Duplicate rows**: AI Agent1 calls `Tool_Update_User_Weekly_Schedule` twice for the same day — once prematurely before user confirmation, once after.
2. **Filter failure**: `Filter_Today_Routine` uses `planned_day_utc` comparison, but rows created with `DD/MM` format have `planned_day_utc = NULL`, causing "no routine found" responses.
3. **Phantom completion**: One of the duplicate rows ends up with `Completed = true` despite the user never confirming — likely caused by a concurrent MAIN_FLOW execution matching the wrong row.

## Root Cause Analysis

### Bug 1: Premature Tool Call

**Node**: `AI Agent1` (scheduling agent)
**Evidence** (chat history session `_8_chat`):

```
id:9321 human: "Ok"
id:9322 ai:    Calls Tool_Update_User_Weekly_Schedule → planned_day: "16/2"  ← PREMATURE
id:9324 ai:    "¿Qué días prefieres?"
...
id:9329 human: "Si" (confirms)
id:9330 ai:    Calls Tool_Update_User_Weekly_Schedule → planned_day: "16/2/2026"  ← DUPLICATE
```

The system prompt says "PASO 4: Condición: El usuario confirma → Llama a Tool_Update_User_Weekly_Schedule", but the AI ignores this and calls the tool in PASO 1 (during Fase 0 data collection). The prompt lacks an explicit **prohibition** against early tool calls.

### Bug 2: Format Inconsistency

`Tool_Update_User_Weekly_Schedule` uses `$fromAI('planned_day')` — the AI decides the format. First call used `DD/MM`, second used `DD/MM/YYYY`. The DB column `planned_day_utc` is populated by a Supabase trigger that only parses `DD/MM/YYYY` format, leaving it `NULL` for `DD/MM`.

`Filter_Today_Routine` compares against `planned_day_utc`:
```javascript
const plannedUTC = item.json.planned_day_utc || item.json.planned_day;
const plannedDate = new Date(plannedUTC);
```

When `planned_day_utc` is NULL, it falls back to `planned_day` (e.g., `"16/2"`), which `new Date("16/2")` parses incorrectly → filter fails → user gets "no routine for today".

### Bug 3: No Duplicate Guard

`Tool_Update_User_Weekly_Schedule` is a Supabase INSERT tool with no `ON CONFLICT` clause. Every call creates a new row regardless of whether one already exists for the same `user_id + week + week_day`.

## Affected Components

```
MAIN_FLOW.json
├── AI Agent1 (system prompt)           ← Fix 1: Prohibit premature tool calls
├── Tool_Update_User_Weekly_Schedule    ← Fix 2: Replace with Postgres upsert
├── Filter_Today_Routine                ← Fix 3: Normalize date comparison
└── (No other workflows affected)
```

## Data Flow (Current vs Fixed)

### Current (Broken)
```
User "Ok" → AI Agent1 → Tool call (INSERT, DD/MM, no UTC) → duplicate row
User confirms → AI Agent1 → Tool call (INSERT, DD/MM/YYYY, with UTC) → another row
User asks for routine → Filter_Today_Routine → fails on DD/MM row → "no routine"
```

### Fixed
```
User "Ok" → AI Agent1 → NO tool call (prompt prohibits it)
User confirms → AI Agent1 → Tool call (UPSERT, DD/MM/YYYY) → single row with UTC
User asks for routine → Filter_Today_Routine → matches planned_day_utc → shows routine
```

## Fix Strategy

| Fix | Node | Type | Description |
|-----|------|------|-------------|
| F1 | AI Agent1 system prompt | Prompt edit | Add explicit rule: "NUNCA llames a Tool_Update_User_Weekly_Schedule antes del PASO 4" |
| F2 | Tool_Update_User_Weekly_Schedule | Node replacement | Replace Supabase INSERT tool with Postgres UPSERT tool using `ON CONFLICT (user_id, week, week_day)` |
| F3 | AI Agent1 system prompt | Prompt edit | Enforce `DD/MM/YYYY` format in planned_day instructions |
| F4 | Filter_Today_Routine | Code edit | Add fallback parsing for `DD/MM` format (defense in depth) |

### DB Prerequisite

Add unique constraint to prevent duplicates at DB level:
```sql
ALTER TABLE user_weekly_schedule
ADD CONSTRAINT uq_user_week_day UNIQUE (user_id, week, week_day);
```

## Out of Scope

- "Juan Perez Garcia" wrong recipient issue (separate concurrent execution bug)
- MorningReminder / WorkoutCompletion flows (not affected by this fix)
- Mesocycle renewal scheduling (uses different path)
