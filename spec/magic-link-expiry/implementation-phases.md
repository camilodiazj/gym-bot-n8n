# KAN-109: Magic Link Expiry — Implementation Phases

## Phase 1: Backend SQL Fix (pixel-dev) — BLOCKING

**File:** `workout-tracker-back/internal/adapter/repository/postgres/workout_repository.go`
**Lines:** 280-313 (`GetTodayWorkout` method)

### Change

Replace single today-only query with two sequential queries:

```go
// Query 1: Today's uncompleted workout (Rule 1 — highest priority)
todayQuery := `
    SELECT uws.day_routine_id, uws.week, uws.week_day, uws.session_name,
           uws.planned_day_utc, uws."Completed", up.goal, up.level
    FROM user_weekly_schedule uws
    JOIN users_plans up ON up.user_id = uws.user_id AND up.status = 'active'
    WHERE uws.user_id = $1
      AND uws."Completed" = false
      AND uws.planned_day_utc = (
        DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota')
        AT TIME ZONE 'America/Bogota'
      )
    ORDER BY uws.day_routine_id
    LIMIT 1
`

// Query 2: Yesterday's uncompleted workout (Rule 2 — fallback only)
yesterdayQuery := `
    SELECT uws.day_routine_id, uws.week, uws.week_day, uws.session_name,
           uws.planned_day_utc, uws."Completed", up.goal, up.level
    FROM user_weekly_schedule uws
    JOIN users_plans up ON up.user_id = uws.user_id AND up.status = 'active'
    WHERE uws.user_id = $1
      AND uws."Completed" = false
      AND uws.planned_day_utc = (
        DATE_TRUNC('day', (NOW() AT TIME ZONE 'America/Bogota') - INTERVAL '1 day')
        AT TIME ZONE 'America/Bogota'
      )
    ORDER BY uws.day_routine_id
    LIMIT 1
`
```

**Go logic:**
```go
err := r.conn.DB.QueryRowContext(ctx, todayQuery, userID).Scan(
    &scheduleID, &week, &dayName, &sessionName, &plannedDay, &completed, &goal, &level,
)
if err == sql.ErrNoRows {
    // Fallback: yesterday's uncompleted workout
    err = r.conn.DB.QueryRowContext(ctx, yesterdayQuery, userID).Scan(
        &scheduleID, &week, &dayName, &sessionName, &plannedDay, &completed, &goal, &level,
    )
    if err == sql.ErrNoRows {
        return nil, nil // No pending workout (rest day)
    }
}
if err != nil {
    return nil, apperror.NewInternalError("failed to query schedule", err)
}
```

### Key differences from current code
1. Added `AND uws."Completed" = false` (current query doesn't filter completed)
2. Added `ORDER BY uws.day_routine_id` for determinism
3. Added yesterday fallback query

### Tests
Existing unit tests pass without changes (mock interface unchanged). Run: `cd workout-tracker-back && make test`

---

## Phase 2: n8n Magic Link Expiry (n8n-agent) — PARALLEL with Phase 1

**File:** `n8n/running_flows/MAIN_FLOW.json`
**Line:** ~1373

### Change

In the WhatsApp magic link INSERT SQL, change:
```sql
expires_at = NOW() + INTERVAL '24 hours'
```
To:
```sql
expires_at = NOW() + INTERVAL '48 hours'
```

**Note:** The Calendar magic link at line ~1741 already uses `INTERVAL '7 days'` — no change needed.

### Deployment
Import updated MAIN_FLOW.json into n8n instance after change.

---

## Phase 3: Documentation Fix (pixel-dev) — PARALLEL with Phase 1

**File:** `CLAUDE.md`

### Change

In the "CRITICAL — planned_day format inconsistency" note, update to reflect the Supabase trigger:

> **Note:** A Supabase trigger `trg_convert_planned_day` (function `convert_planned_day_to_utc()`) fires on INSERT/UPDATE of `user_weekly_schedule`. It auto-populates `planned_day_utc` from `planned_day` by parsing both ISO (`YYYY-MM-DD`) and slash (`D/M/YYYY`) formats. Therefore `planned_day_utc` is always available in production despite not being explicitly set by the n8n scheduling tool.

---

## Verification Checklist

1. `cd workout-tracker-back && make test` — all pass
2. Manual test via Supabase MCP:
   - Query a real user's `user_weekly_schedule` to confirm `planned_day_utc` is populated
   - Verify grace period logic with test data (workout from yesterday, Completed=false)
3. Deploy backend to Cloud Run
4. Import MAIN_FLOW.json to n8n
5. End-to-end: Open magic link day after planned workout, verify workout appears

## Scenario Matrix (18 validated)

| # | Scenario | Rule | Result |
|---|----------|------|--------|
| 1 | Tuesday after missing Monday | R2 | Yesterday's workout |
| 2 | Wednesday after missing Mon+Tue | R1 | Today's workout |
| 3 | Re-open after completing today | - | Rest day (Completed=true filtered) |
| 4 | Monday new week after Friday miss | R1 | Today's workout |
| 5 | Sunday after missing Saturday | R2 | Yesterday's workout |
| 6 | Tuesday (Lower A) + missed Monday (Upper A) | R1 | Today's workout |
| 7 | Wednesday after Tuesday makeup | R1 | Today's workout |
| 8 | Saturday, end of mesocycle, missed Friday | R2 | Yesterday's workout |
| 9 | n8n-created session | R1/R2 | Works (trigger populates UTC) |
| 10 | Two sessions same day | R1 | Deterministic (ORDER BY) |
| 11 | All sessions completed, Thursday | R3 | Rest day |
| 12 | Missed entire week, Saturday | R2 | Yesterday's workout |
| 13 | End-of-week gap, no next week scheduled | R3 | Rest day |
| 14 | Non-standard week (Wed/Fri/Sun) | R1/R2 | Works (date-agnostic) |
| 15 | Mid-week reschedule (Wed->Thu) | R3 | Rest day on Wed |
| 16 | Stale calendar link (Mon link on Fri) | R1 | Today's workout |
| 17 | Post-onboarding, no schedule | R3 | No workout |
| 18 | Weekend schedule (Sat/Sun/Tue) | R1/R2 | Works |
