# KAN-109 Phase 2: MAIN_FLOW WhatsApp Grace Period

## Context

The Go backend already has a 1-day grace period: if a user opens the workout tracker web app the day after a missed workout, they see yesterday's uncompleted session. However, the WhatsApp flow (MAIN_FLOW.json) does NOT have this — when a user asks for their routine via WhatsApp and they missed yesterday, they get "no tienes sesion programada, descansa" instead of their pending workout. This creates an inconsistent experience across channels.

## Changes Required (4 nodes in MAIN_FLOW.json)

### 1. `Filter_Today_Routine` (Code node, line ~1011)

Replace the today-only filter with priority-based logic:

```javascript
// Rule 1: Today's uncompleted sessions (highest priority)
const todayRoutine = items.filter(item => {
  // ... existing date parsing ...
  return isToday && isNotCompleted;
});
if (todayRoutine.length > 0) return todayRoutine;

// Rule 2: Yesterday's uncompleted sessions (fallback only)
const yesterdayMidnightUTC = new Date(todayDate.getTime() - 24 * 60 * 60 * 1000);
const yesterdayRoutine = items.filter(item => {
  // ... same date parsing ...
  const isYesterday = plannedDate.getTime() === yesterdayMidnightUTC.getTime();
  return isYesterday && isNotCompleted;
});
return yesterdayRoutine.length > 0 ? yesterdayRoutine : [];
```

### 2. `AI Agent` prompt (line ~9)

Update the system prompt to:
- Rename `SESION_ENTRENAMIENTO_PARA_HOY` -> `SESION_PENDIENTE`
- Add a computed flag indicating if the session is from yesterday
- Instruct the AI: if session is from yesterday, acknowledge it ("Ayer no completaste [session]. Hagamosla hoy como recuperacion")
- Keep the same formatting rules for the workout display

### 3. `Tool_Update_User_Weekly_Schedule1` (Confirmation SQL, line ~447)

Change the WHERE clause from today-only to today OR yesterday:

```sql
UPDATE user_weekly_schedule
SET "Completed" = true
WHERE user_id = '...'
AND planned_day_utc IN (
  DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota',
  (DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota') - INTERVAL '1 day'
)
AND "Completed" = false;
```

This is simpler than passing `planned_day_utc` through the AI agent — the SQL itself handles the grace window.

### 4. `CONFIRMATION AGENT` prompt (line ~846)

Update from "rutina de hoy" to "rutina pendiente (de hoy o ayer)" so the AI knows the confirmation can target either day.

## Nodes NOT Changed (confirmed safe)

| Node | Why safe |
|------|----------|
| `Tool_Get_Workout2` | Queries by `session_name` + `week`, not by date |
| `If_Has_Routine_Today` | Checks if Filter output has data — date-agnostic |
| `Send message1` (rest day) | Only shown when Filter returns empty — still correct |
| `Create Magic Link` | Tied to `user_id`, not date |
| `Build URL` | Session-agnostic |
| `GetWeeklySchedule` | Already fetches ALL schedules (no date filter) |
| `pending_tasks UPDATE` (line 1153) | Filters by `user_id + task_type + status`, not by date |

## E2E Test Impact

### Existing tests: All PASS without changes

| Test | User | Schedule | Impact | Status |
|------|------|----------|--------|--------|
| TC004 (Rest Day) | 570000000002 | TOMORROW only | No yesterday data -> still rest day | Safe |
| TC006 (VER_RUTINA) | 570000000003 | TODAY | R1 priority returns today -> same | Safe |
| TC011/TC012 (CONFIRMAR) | 570000000004 | TODAY | Confirmation SQL matches today -> same | Safe |
| TC_MESO_* | 5700000005X | Past weeks (completed) | Grace ignores completed | Safe |

### New test cases: 3 GRACE_PERIOD scenarios

Phones `570000000081-083` (available in test range).

**TC_GRACE_001** — Yesterday uncompleted, no today (R2 -> expect workout)
- User: `570000000081` / `e2e00081-0000-0000-0000-000000000081`
- Schedule: yesterday `Full Body A`, `Completed = false`, NO today entry
- Message: `"Muestrame mi rutina de hoy"`
- Validation: response includes exercise names/sets (NOT rest day message)
- Type: SINGLE

**TC_GRACE_002** — Both today AND yesterday uncompleted (R1 priority -> expect today's)
- User: `570000000082` / `e2e00082-0000-0000-0000-000000000082`
- Schedule: yesterday `Full Body A` (uncompleted) + today `Full Body B` (uncompleted)
- Message: `"Muestrame mi rutina de hoy"`
- Validation: response includes `Full Body B` (today wins over yesterday)
- Type: SINGLE

**TC_GRACE_003** — Yesterday completed, no today (R3 -> rest day)
- User: `570000000083` / `e2e00083-0000-0000-0000-000000000083`
- Schedule: yesterday `Full Body A`, `Completed = true`, NO today entry
- Message: `"Que hay para hoy?"`
- Validation: response includes "descanso" or "no tienes sesion"
- Type: SINGLE

### Files to modify

1. **`e2e/test_data_setup.sql`** — Add teardown + fixtures for phones 081-083
2. **`n8n/tests/GymRatFlow_E2E_TestRunner.json`** — Add 3 test cases to "Load Tests" node
3. **`CLAUDE.md`** — Add phones 081-083 to test users reference

## Verification

1. Import updated MAIN_FLOW.json into n8n
2. Create test user with yesterday's uncompleted workout (same SQL as backend tests)
3. Test VER_RUTINA_DE_HOY -> should show yesterday's workout with makeup messaging
4. Test CONFIRMAR_RUTINA -> should mark yesterday's workout as completed
5. Verify today's workout still takes priority when both exist
6. Run full E2E suite (`GymRatFlow_E2E_TestRunner`) -> all existing tests still pass
