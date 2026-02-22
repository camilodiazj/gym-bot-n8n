# KAN-109: Magic Link Expiry — Architecture

## Algorithm Audit

| Original Requirement | Verdict | Reason |
|---------------------|---------|--------|
| SQL query fallback yesterday | **KEEP** | Core fix — without this nothing works |
| Magic link 24h->48h (n8n WhatsApp) | **KEEP** | Aligns with 1-day grace window |
| Magic link 24h->48h (Go `Create()`) | **DELETE** | Dead code — `Create()` is never called from any handler |
| Rename GetTodayWorkout interface | **DELETE** | Zero value. Grace period is internal repo detail |
| Dual-path SQL (planned_day_utc + planned_day) | **DELETE** | Supabase trigger `trg_convert_planned_day` auto-populates `planned_day_utc` |
| Frontend changes | **DELETE** | Frontend already handles backend response correctly |
| Update CLAUDE.md | **KEEP** | Incorrect docs about NULL planned_day_utc cause future waste |

**Result: 3 changes survive (from 7 originally proposed)**

## Path: Logic-Driven (No UI changes)

## Data Flow: Current vs Fixed

### Current (broken for missed days)
```
User opens link
  -> Auth middleware validates magic link (OK if <24h)
  -> GetTodayWorkout: WHERE planned_day_utc = TODAY
  -> No match if planned day was yesterday
  -> 404 "no workout scheduled for today"
```

### Fixed (1-day grace)
```
User opens link
  -> Auth middleware validates magic link (OK if <48h)
  -> GetTodayWorkout:
      Query 1: WHERE planned_day_utc = TODAY AND Completed = false
      -> Found? Return it.
      -> Not found?
      Query 2: WHERE planned_day_utc = YESTERDAY AND Completed = false
      -> Found? Return it.
      -> Not found? return nil (rest day)
```

## Critical Infrastructure: Supabase Trigger

```sql
-- Trigger: trg_convert_planned_day
-- Fires: BEFORE INSERT OR UPDATE on user_weekly_schedule
-- Function: convert_planned_day_to_utc()
-- Behavior: Parses planned_day (YYYY-MM-DD or D/M/YYYY) -> planned_day_utc (TIMESTAMPTZ)
-- Guarantee: planned_day_utc is ALWAYS populated in production
```

This trigger means we only need to query `planned_day_utc`. No dual-path SQL required.

## Grace Period Rules (validated by kiro-coach, 18 scenarios)

| Priority | Condition | Result |
|----------|-----------|--------|
| 1 | Today has uncompleted session | Show today's workout |
| 2 | Yesterday has uncompleted session (today has none) | Show yesterday's workout |
| 3 | Neither | Rest day (404) |

**Why 1 day, not 7:** Beyond 24h, muscle groups are no longer in the correct adaptation window. Showing Monday's Full Body A on Wednesday when Full Body B is planned disrupts recovery windows and periodization.

## Files Modified

| File | Change | Role |
|------|--------|------|
| `workout-tracker-back/internal/adapter/repository/postgres/workout_repository.go` | SQL: today + yesterday fallback | pixel-dev |
| `n8n/running_flows/MAIN_FLOW.json` | Magic link INSERT: 24h -> 48h | n8n-agent |
| `CLAUDE.md` | Fix planned_day_utc documentation | pixel-dev |
