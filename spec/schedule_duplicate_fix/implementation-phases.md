# Implementation Phases: Schedule Duplicate Fix

## Phase 1: Database Guard (Blocking — do first)

**Owner**: n8n-agent
**Estimated effort**: Small

### Task 1.1: Add unique constraint

```sql
ALTER TABLE user_weekly_schedule
ADD CONSTRAINT uq_user_week_day UNIQUE (user_id, week, week_day);
```

**Risk**: If duplicate rows already exist for other users, the constraint will fail. Run this check first:

```sql
SELECT user_id, week, week_day, COUNT(*)
FROM user_weekly_schedule
GROUP BY user_id, week, week_day
HAVING COUNT(*) > 1;
```

If duplicates exist, clean them up before applying the constraint (keep the row with `planned_day_utc` populated, delete the other).

---

## Phase 2: Replace Tool with Upsert (Core fix)

**Owner**: n8n-agent
**Estimated effort**: Medium
**Depends on**: Phase 1

### Task 2.1: Replace `Tool_Update_User_Weekly_Schedule`

**Current**: `n8n-nodes-base.supabaseTool` (INSERT only)
**Target**: `n8n-nodes-base.postgresTool` (UPSERT via SQL)

Replace the Supabase tool node with a Postgres tool node using this SQL:

```sql
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES (
  '{{ $items('GetUser')[0].json.user_id }}',
  {{ $fromAI('week', 'Número de semana', 'number') }},
  '{{ $fromAI('week_day', 'Día de la semana: Lunes, Martes, etc.', 'string') }}',
  '{{ $fromAI('session_name', 'Nombre de la sesión', 'string') }}',
  '{{ $fromAI('planned_day', 'Fecha en formato DD/MM/YYYY', 'string') }}',
  false
)
ON CONFLICT (user_id, week, week_day)
DO UPDATE SET
  session_name = EXCLUDED.session_name,
  planned_day = EXCLUDED.planned_day,
  "Completed" = false
RETURNING *;
```

**Key points**:
- Uses `ON CONFLICT` on the new unique constraint
- On duplicate: updates `session_name`, `planned_day`, resets `Completed` to false
- Returns the upserted row so the AI agent gets confirmation
- Credential: `vZLJtIWG5nYXMez4` (Postgres Supabase Memory)

### Task 2.2: Verify node connections

Ensure the new Postgres tool node is connected to AI Agent1 as a tool (same as the old Supabase tool node).

---

## Phase 3: System Prompt Fix (Prevent premature calls)

**Owner**: n8n-agent
**Estimated effort**: Small
**Can run in parallel with Phase 2**

### Task 3.1: Add prohibition rule to AI Agent1 system prompt

Add this block after the existing "REGLAS TECNICAS Y DE CONTROL" section:

```
6. **PROHIBICIÓN ABSOLUTA:** NUNCA llames a `Tool_Update_User_Weekly_Schedule` antes del PASO 4.
   En la Fase 0 y Pasos 1-3, tu única tarea es RECOLECTAR información y CONFIRMAR con el usuario.
   La herramienta SOLO se ejecuta después de que el usuario confirme explícitamente la agenda completa.
```

### Task 3.2: Enforce date format in prompt

In PASO 4 instructions, change:

```
Envía: user_id, week (calculada), week_day, session_name y planned_day.
```

To:

```
Envía: user_id, week (calculada), week_day, session_name y planned_day (SIEMPRE en formato DD/MM/YYYY, ejemplo: 16/02/2026).
```

---

## Phase 4: Filter Hardening (Defense in depth)

**Owner**: n8n-agent
**Estimated effort**: Small
**Can run in parallel with Phase 2 & 3**

### Task 4.1: Update `Filter_Today_Routine` date parsing

Replace the current fallback logic:

```javascript
const plannedUTC = item.json.planned_day_utc || item.json.planned_day;
const plannedDate = new Date(plannedUTC);
```

With robust parsing:

```javascript
let plannedDate;
if (item.json.planned_day_utc) {
  plannedDate = new Date(item.json.planned_day_utc);
} else {
  // Fallback: parse DD/MM or DD/MM/YYYY from planned_day
  const parts = item.json.planned_day.split('/');
  const day = parseInt(parts[0], 10);
  const month = parseInt(parts[1], 10) - 1;
  const year = parts[2] ? parseInt(parts[2], 10) : todayMidnightUTC.year;
  plannedDate = new Date(Date.UTC(year, month, day, 5, 0, 0)); // 05:00 UTC = midnight Bogota
}
```

---

## Phase 5: Verification

**Owner**: Manual QA (or E2E test)
**Depends on**: Phases 1-4

### Task 5.1: Test with a new user

1. Create a test user through the full KYC flow
2. Enter the scheduling flow
3. Verify only ONE row per day in `user_weekly_schedule`
4. Verify `planned_day` format is `DD/MM/YYYY`
5. Verify `Filter_Today_Routine` returns the correct session
6. Verify the user can see their routine via "VER_RUTINA_DE_HOY"

### Task 5.2: Test idempotency

1. Trigger the scheduling flow twice for the same week
2. Verify the UPSERT updates the existing row instead of creating a duplicate

### Task 5.3: Verify existing users are unaffected

Query `user_weekly_schedule` for active users and confirm no data was lost or corrupted.

---

## Execution Order

```
Phase 1 (DB constraint)
    │
    ├── Phase 2 (Upsert tool) ──┐
    ├── Phase 3 (Prompt fix) ───┤── Phase 5 (Verification)
    └── Phase 4 (Filter fix) ───┘
```

Phases 2, 3, 4 can execute in **parallel** after Phase 1 completes.

---

## Files Modified

| File | Change |
|------|--------|
| `n8n/running_flows/MAIN_FLOW.json` | Replace Tool_Update_User_Weekly_Schedule node, update AI Agent1 system prompt, update Filter_Today_Routine code |
| Supabase (migration) | `ALTER TABLE user_weekly_schedule ADD CONSTRAINT uq_user_week_day UNIQUE (user_id, week, week_day)` |

**Re-import in n8n**: `n8n/running_flows/MAIN_FLOW.json`
