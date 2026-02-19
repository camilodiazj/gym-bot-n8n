# Architecture: Calendar Event on Schedule (KAN-57)

**Feature:** Send Google Calendar invitations when user schedules training week
**Workflow:** `MAIN_FLOW.json`
**Status:** Implementation Guide
**Created:** 2026-02-15

---

## 1. Algorithm Audit Summary

| Original Requirement | Decision | Rationale |
|---------------------|----------|-----------|
| 2 separate Postgres queries (schedule + workouts) | **MERGED into 1** | Single query with JOINs reduces nodes and round-trips |
| HasEmailCal IF guard | **DELETED** | Users always have email (KYC collects it). Zero-value node. |
| .ics file generation + SMTP email | **REPLACED** with Google Calendar node | Google handles invitation delivery, sync, reminders. Eliminates SMTP setup, binary data, .ics RFC compliance. |
| SplitInBatches loop | **DELETED** | n8n auto-iterates items through nodes. Not needed. |
| Exercise translation maps (equipmentMap, muscleMap) | **SIMPLIFIED** | Calendar description uses plain text. Only `spanish_name` needed (already in DB). No translation maps required. |

**Result: 7 original nodes reduced to 5 core + 1 traceability = 6 total.**

---

## 2. Tech Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Workflow | n8n (`MAIN_FLOW.json`) | Orchestration |
| Calendar API | Google Calendar via `n8n-nodes-base.googleCalendar` | Event creation + invitation delivery |
| Database | Supabase (PostgreSQL) | Schedule data, magic links, event ID storage |
| Auth | Google OAuth 2.0 | Calendar API credential (n8n UI) |

**No changes to:** Go backend, React frontend, WhatsApp integration, existing MAIN_FLOW logic.

---

## 3. Data Flow

```
Send message2 ─────────────────────────────── Filtered Message4 (return)
      │                                        [EXISTING - UNCHANGED]
      │
      │ parallel branch
      ▼
IsScheduleComplete ──false──► (stop)
      │ true
      ▼
GetCalendarData ──────────────────────────── 1 Postgres query
      │                                      (schedule + workouts + user + profile)
      ▼
CreateCalendarMagicLink ─────────────────── 1 Postgres INSERT
      │                                      (magic link, 7-day expiry)
      ▼
PrepareCalendarEvents ───────────────────── 1 Code node
      │                                      (date parsing, description building)
      │                                      outputs N items (1 per training day)
      ▼
GoogleCalendar_CreateEvent ──────────────── N API calls (auto-iterated)
      │                                      (creates event, adds user as attendee)
      ▼
UpdateCalendarEventId ───────────────────── N Postgres UPDATEs
                                             (stores event_id for traceability)
```

**Total: 6 new nodes. Zero modifications to existing nodes (only 1 connection change).**

---

## 4. DB Schema Change

### Migration: `add_calendar_event_id`

```sql
ALTER TABLE user_weekly_schedule
  ADD COLUMN calendar_event_id TEXT DEFAULT NULL;

COMMENT ON COLUMN user_weekly_schedule.calendar_event_id IS
  'Google Calendar event ID. Set after calendar invitation is created. Enables event updates/deletion and E2E test verification.';
```

**Impact:** Column is nullable with default NULL. Zero effect on existing queries, inserts, or workflows.

---

## 5. Node Specifications

### Node 1: IsScheduleComplete (IF)

| Property | Value |
|----------|-------|
| Type | `n8n-nodes-base.if` |
| Condition | `{{ $('AI Agent1').first().json.output }}` contains `"agendada"` |
| True | → GetCalendarData |
| False | → (dead end) |

**Why this works:** AI Agent1's system prompt guarantees the word "agendada" only appears in the final scheduling confirmation message. Intermediate turns (collecting start date, confirming days) never use this word.

---

### Node 2: GetCalendarData (Postgres)

Single query that fetches everything needed — schedule, workouts, user, and profile:

```sql
SELECT
  uws.day_routine_id,
  uws.week,
  uws.week_day,
  uws.session_name,
  uws.planned_day,
  u.email,
  u.full_name,
  ugp.preferred_schedule,
  ugp.session_duration_mins,
  ugp.training_environment,
  w.day_name,
  w.exercise_order,
  w.sets,
  w.reps,
  w.rir,
  w."rest-seconds" AS rest_seconds,
  e.spanish_name,
  e.role
FROM user_weekly_schedule uws
JOIN users u ON uws.user_id = u.user_id
JOIN users_gym_profile ugp ON u.cel_number = ugp.whatsapp_id
LEFT JOIN workouts w ON w.user_id = uws.user_id
  AND w.week = uws.week
  AND w.day_name = uws.session_name
LEFT JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE uws.user_id = '{{ $items("GetUser")[0].json.user_id }}'
  AND uws."Completed" = false
  AND uws.week = (
    SELECT MAX(week) FROM user_weekly_schedule
    WHERE user_id = '{{ $items("GetUser")[0].json.user_id }}'
  )
ORDER BY uws.planned_day, w.exercise_order
```

| Config | Value |
|--------|-------|
| `executeOnce` | `true` |
| Credential | Supabase Memory (`vZLJtIWG5nYXMez4`) |

**Output:** Flat rows — multiple rows per day (one per exercise). The Code node groups them.

---

### Node 3: CreateCalendarMagicLink (Postgres)

```sql
INSERT INTO magic_links (code, user_id, expires_at)
VALUES (
  substr(md5(random()::text || '{{ $items("GetUser")[0].json.user_id }}' || now()::text), 1, 6),
  '{{ $items("GetUser")[0].json.user_id }}',
  NOW() + INTERVAL '7 days'
)
ON CONFLICT (code) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  created_at = NOW(),
  expires_at = NOW() + INTERVAL '7 days',
  used_at = NULL
RETURNING code
```

> **Magic link behavior:** The link authenticates the user. The frontend shows the workout for **the day it's opened** (`GET /api/v1/workouts/today`). Since each calendar event IS a training day, clicking the link from that event shows the correct routine. On rest days, no workout displays — expected behavior.

---

### Node 4: PrepareCalendarEvents (Code)

**Input:** All rows from GetCalendarData + magic link code from CreateCalendarMagicLink.

**Logic:**

1. **Group rows by `planned_day`** (each group = one calendar event)
2. **Parse date:** `planned_day` is DD/MM format → convert to `YYYY-MM-DD` using current year
3. **Map time from `preferred_schedule`:**

| preferred_schedule | Start |
|-------------------|-------|
| Manana | 06:00 |
| Tarde | 16:00 |
| Noche | 19:00 |

4. **Parse duration from `session_duration_mins`:**

| Value | Minutes |
|-------|---------|
| "45-60 minutos" | 60 |
| "60-75 minutos" | 75 |
| "75-90 minutos" | 90 |
| default | 60 |

5. **Build description** per day (plain text):

```
Tu rutina del dia:

EJERCICIOS COMPUESTOS (3 sets x 10-12 reps | RIR 3)
1. Press de Banca con Barra
2. Press Militar con Barra

CORE (2 sets x 12-15 reps | RIR 3)
3. Dead Bug

AISLAMIENTO (2 sets x 12-15 reps | RIR 3)
4. Aperturas con Mancuerna

Abre tu rutina: https://workout-tracker-69b08.web.app/w?c={code}
```

6. **Output N items** (one per training day):

```javascript
{
  json: {
    day_routine_id: "uuid-here",           // for UpdateCalendarEventId
    summary: "Full Body A - Entrenamiento Kairos",
    description: "...",
    startDateTime: "2026-02-16T06:00:00",
    endDateTime: "2026-02-16T07:00:00",
    attendeeEmail: "user@email.com",
    location: "Gimnasio",                  // or "Casa" if HOME
    timezone: "America/Bogota"
  }
}
```

---

### Node 5: GoogleCalendar_CreateEvent

| Property | Value |
|----------|-------|
| Type | `n8n-nodes-base.googleCalendar` |
| Operation | Create Event |
| Calendar | GymBot training calendar |
| Summary | `{{ $json.summary }}` |
| Description | `{{ $json.description }}` |
| Start | `{{ $json.startDateTime }}` |
| End | `{{ $json.endDateTime }}` |
| Timezone | `America/Bogota` |
| Attendees | `{{ $json.attendeeEmail }}` |
| Send Updates | `all` |
| Reminders | 15 min (popup) |
| Location | `{{ $json.location }}` |
| `continueOnFail` | `true` |
| Credential | Google Calendar OAuth (setup in n8n UI) |

**Auto-iteration:** n8n processes all N items from PrepareCalendarEvents sequentially. No loop node needed.

---

### Node 6: UpdateCalendarEventId (Postgres)

```sql
UPDATE user_weekly_schedule
SET calendar_event_id = '{{ $json.id }}'
WHERE day_routine_id = '{{ $json.day_routine_id }}'
```

| Config | Value |
|--------|-------|
| `continueOnFail` | `true` |
| Credential | Supabase Memory (`vZLJtIWG5nYXMez4`) |

**Note:** `$json.id` comes from Google Calendar API response. `$json.day_routine_id` must be passed through from PrepareCalendarEvents — the Google Calendar node preserves `$json` fields that aren't consumed. If not, we use `$('PrepareCalendarEvents').item` reference.

---

## 6. Connection Changes

### Only modification to existing workflow:

**Current** `Send message2` connections:
```json
"Send message2": {
  "main": [[
    { "node": "Filtered Message4", "type": "main", "index": 0 }
  ]]
}
```

**New:**
```json
"Send message2": {
  "main": [[
    { "node": "Filtered Message4", "type": "main", "index": 0 },
    { "node": "IsScheduleComplete", "type": "main", "index": 0 }
  ]]
}
```

### New connection chain:
```
IsScheduleComplete(true) → GetCalendarData → CreateCalendarMagicLink
  → PrepareCalendarEvents → GoogleCalendar_CreateEvent → UpdateCalendarEventId
```

---

## 7. Prerequisites

| Prerequisite | Action | Owner |
|-------------|--------|-------|
| Google Cloud project | Create project, enable Calendar API | n8n-agent |
| OAuth credentials | Create OAuth 2.0 client, configure in n8n | n8n-agent |
| GymBot calendar | Create dedicated Google Calendar for training events | n8n-agent |
| DB migration | Run `ALTER TABLE` on Supabase | n8n-agent |

---

## 8. Error Handling

| Failure Point | Behavior | User Impact |
|--------------|----------|-------------|
| IsScheduleComplete false | Branch stops silently | None (intermediate turn) |
| GetCalendarData fails | Branch dies | No calendar events. WhatsApp unaffected. |
| CreateCalendarMagicLink fails | Branch dies | No calendar events. WhatsApp unaffected. |
| PrepareCalendarEvents JS error | Branch dies | No calendar events. WhatsApp unaffected. |
| Google Calendar API fails | `continueOnFail: true` | No events created. WhatsApp unaffected. |
| UpdateCalendarEventId fails | `continueOnFail: true` | Events created but IDs not stored. Traceability gap only. |

**Parallel branch isolation ensures the existing WhatsApp flow is NEVER affected.**
