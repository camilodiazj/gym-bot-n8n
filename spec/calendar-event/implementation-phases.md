# Implementation Phases: Calendar Event on Schedule (KAN-57)

**Status:** Implementation Guide
**Created:** 2026-02-15
**Path Selection:** Logic-Driven (n8n workflow + DB, no UI changes)

---

## Phase 0: Prerequisites (Manual, ~20 min)

**Owner:** n8n-agent
**Blocking:** Phases 1 and 2 cannot proceed without this.

| Task | Action |
|------|--------|
| Google Cloud project | Create or reuse project. Enable **Google Calendar API**. |
| OAuth 2.0 credentials | Create OAuth client (Web Application type). Redirect URI: n8n instance callback URL. |
| n8n credential | In n8n UI: Credentials → Google Calendar OAuth2 → Connect with GymBot Google account. |
| Dedicated calendar | Create a Google Calendar named "Kairos Entrenamientos" in the GymBot account. Note the Calendar ID. |
| DB migration | Run on Supabase: `ALTER TABLE user_weekly_schedule ADD COLUMN calendar_event_id TEXT DEFAULT NULL;` |

**Exit criteria:** Google Calendar credential is connected and tested in n8n. Migration applied.

---

## Phase 1: Workflow Nodes (n8n-agent, ~2 hours)

**Can start immediately after Phase 0.**

### Task 1.1: Add IsScheduleComplete (IF node)

Add to MAIN_FLOW.json `nodes` array:

```json
{
  "parameters": {
    "conditions": {
      "options": { "caseSensitive": false, "leftValue": "", "typeValidation": "strict", "version": 3 },
      "conditions": [{
        "leftValue": "={{ $('AI Agent1').first().json.output }}",
        "rightValue": "agendada",
        "operator": { "type": "string", "operation": "contains" }
      }],
      "combinator": "and"
    },
    "options": {}
  },
  "type": "n8n-nodes-base.if",
  "typeVersion": 2.3,
  "position": [-4080, 2400],
  "id": "<auto>",
  "name": "IsScheduleComplete"
}
```

**Position note:** Place below Send message2 (`[-4304, 2144]`). Offset: x+224, y+256.

---

### Task 1.2: Add GetCalendarData (Postgres)

```json
{
  "parameters": {
    "operation": "executeQuery",
    "query": "SELECT uws.day_routine_id, uws.week, uws.week_day, uws.session_name, uws.planned_day, u.email, u.full_name, ugp.preferred_schedule, ugp.session_duration_mins, ugp.training_environment, w.day_name, w.exercise_order, w.sets, w.reps, w.rir, w.\"rest-seconds\" AS rest_seconds, e.spanish_name, e.role FROM user_weekly_schedule uws JOIN users u ON uws.user_id = u.user_id JOIN users_gym_profile ugp ON u.cel_number = ugp.whatsapp_id LEFT JOIN workouts w ON w.user_id = uws.user_id AND w.week = uws.week AND w.day_name = uws.session_name LEFT JOIN exercises e ON w.exercise_id = e.exercise_id WHERE uws.user_id = '{{ $items(\"GetUser\")[0].json.user_id }}' AND uws.\"Completed\" = false AND uws.week = (SELECT MAX(week) FROM user_weekly_schedule WHERE user_id = '{{ $items(\"GetUser\")[0].json.user_id }}') ORDER BY uws.planned_day, w.exercise_order",
    "options": {}
  },
  "type": "n8n-nodes-base.postgres",
  "typeVersion": 2.6,
  "position": [-3840, 2400],
  "id": "<auto>",
  "name": "GetCalendarData",
  "executeOnce": true,
  "credentials": {
    "postgres": { "id": "vZLJtIWG5nYXMez4", "name": "Supabase Memory" }
  }
}
```

---

### Task 1.3: Add CreateCalendarMagicLink (Postgres)

```json
{
  "parameters": {
    "operation": "executeQuery",
    "query": "INSERT INTO magic_links (code, user_id, expires_at) VALUES (substr(md5(random()::text || '{{ $items(\"GetUser\")[0].json.user_id }}' || now()::text), 1, 6), '{{ $items(\"GetUser\")[0].json.user_id }}', NOW() + INTERVAL '7 days') ON CONFLICT (code) DO UPDATE SET user_id = EXCLUDED.user_id, created_at = NOW(), expires_at = NOW() + INTERVAL '7 days', used_at = NULL RETURNING code",
    "options": {}
  },
  "type": "n8n-nodes-base.postgres",
  "typeVersion": 2.6,
  "position": [-3600, 2400],
  "id": "<auto>",
  "name": "CreateCalendarMagicLink",
  "executeOnce": true,
  "credentials": {
    "postgres": { "id": "vZLJtIWG5nYXMez4", "name": "Supabase Memory" }
  }
}
```

---

### Task 1.4: Add PrepareCalendarEvents (Code)

```json
{
  "parameters": {
    "jsCode": "<SEE SECTION BELOW FOR FULL CODE>"
  },
  "type": "n8n-nodes-base.code",
  "typeVersion": 2,
  "position": [-3360, 2400],
  "id": "<auto>",
  "name": "PrepareCalendarEvents",
  "executeOnce": true
}
```

#### Full JavaScript Code for PrepareCalendarEvents:

```javascript
// ---------- INPUT ----------
const allRows = $('GetCalendarData').all().map(i => i.json);
const magicCode = $('CreateCalendarMagicLink').first().json.code;
const magicUrl = `https://workout-tracker-69b08.web.app/w?c=${magicCode}`;

if (allRows.length === 0) return [];

// ---------- CONFIG ----------
const timeMap = { 'Mañana': '06:00', 'Tarde': '16:00', 'Noche': '19:00' };
const roleLabels = { compound: 'EJERCICIOS COMPUESTOS', core: 'CORE', isolation: 'AISLAMIENTO' };
const roleOrder = ['compound', 'core', 'isolation'];

function parseDuration(str) {
  if (!str) return 60;
  const match = str.match(/(\d+)\s*-\s*(\d+)/);
  return match ? parseInt(match[2]) : 60;
}

function parseDate(ddmm) {
  const [dd, mm] = ddmm.split('/').map(Number);
  const now = DateTime.now().setZone('America/Bogota');
  let year = now.year;
  if (mm < now.month) year += 1;
  return DateTime.fromObject({ year, month: mm, day: dd }, { zone: 'America/Bogota' });
}

function addMinutes(isoStr, mins) {
  return DateTime.fromISO(isoStr, { zone: 'America/Bogota' }).plus({ minutes: mins }).toFormat("yyyy-MM-dd'T'HH:mm:ss");
}

// ---------- GROUP BY DAY ----------
const dayMap = new Map();
for (const row of allRows) {
  const key = row.planned_day;
  if (!dayMap.has(key)) {
    dayMap.set(key, {
      day_routine_id: row.day_routine_id,
      session_name: row.session_name,
      planned_day: row.planned_day,
      week_day: row.week_day,
      email: row.email,
      full_name: row.full_name,
      preferred_schedule: row.preferred_schedule,
      session_duration_mins: row.session_duration_mins,
      training_environment: row.training_environment,
      exercises: []
    });
  }
  if (row.spanish_name) {
    dayMap.get(key).exercises.push({
      exercise_order: row.exercise_order,
      spanish_name: row.spanish_name,
      sets: row.sets,
      reps: row.reps,
      rir: row.rir,
      rest_seconds: row.rest_seconds,
      role: row.role || 'compound'
    });
  }
}

// ---------- BUILD EVENTS ----------
const events = [];
for (const [, day] of dayMap) {
  const startTime = timeMap[day.preferred_schedule] || '06:00';
  const durationMins = parseDuration(day.session_duration_mins);
  const date = parseDate(day.planned_day);
  const startDT = date.set({
    hour: parseInt(startTime.split(':')[0]),
    minute: parseInt(startTime.split(':')[1])
  }).toFormat("yyyy-MM-dd'T'HH:mm:ss");
  const endDT = addMinutes(startDT, durationMins);

  // Build description grouped by role
  let desc = 'Tu rutina del dia:\n\n';
  const byRole = {};
  for (const ex of day.exercises) {
    if (!byRole[ex.role]) byRole[ex.role] = [];
    byRole[ex.role].push(ex);
  }
  for (const role of roleOrder) {
    const exs = byRole[role];
    if (!exs || exs.length === 0) continue;
    const first = exs[0];
    const header = `${roleLabels[role] || role.toUpperCase()} (${first.sets} sets x ${first.reps} reps | RIR ${first.rir})`;
    desc += `${header}\n`;
    for (const ex of exs) {
      desc += `${ex.exercise_order}. ${ex.spanish_name}\n`;
    }
    desc += '\n';
  }
  desc += `Abre tu rutina: ${magicUrl}`;

  const location = (day.training_environment || 'GYM') === 'HOME' ? 'Casa' : 'Gimnasio';

  events.push({
    json: {
      day_routine_id: day.day_routine_id,
      summary: `${day.session_name} - Entrenamiento Kairos`,
      description: desc,
      startDateTime: startDT,
      endDateTime: endDT,
      attendeeEmail: day.email,
      location,
      timezone: 'America/Bogota'
    }
  });
}

return events;
```

---

### Task 1.5: Add GoogleCalendar_CreateEvent

```json
{
  "parameters": {
    "calendarId": "<KAIROS_CALENDAR_ID>",
    "additionalFields": {
      "summary": "={{ $json.summary }}",
      "description": "={{ $json.description }}",
      "location": "={{ $json.location }}",
      "start": "={{ $json.startDateTime }}",
      "end": "={{ $json.endDateTime }}",
      "timeZone": "America/Bogota",
      "attendees": "={{ $json.attendeeEmail }}",
      "sendUpdates": "all",
      "reminders": {
        "useDefault": false,
        "overrides": [{ "method": "popup", "minutes": 15 }]
      }
    }
  },
  "type": "n8n-nodes-base.googleCalendar",
  "typeVersion": 1.3,
  "position": [-3120, 2400],
  "id": "<auto>",
  "name": "GoogleCalendar_CreateEvent",
  "continueOnFail": true,
  "credentials": {
    "googleCalendarOAuth2Api": { "id": "<CONFIGURE_IN_N8N>", "name": "GymBot Google Calendar" }
  }
}
```

**Note:** The exact parameter structure may vary by n8n version. Verify against the node's input panel after adding it. The `calendarId` should be the email-format ID from the Kairos calendar (e.g., `abc123@group.calendar.google.com`).

---

### Task 1.6: Add UpdateCalendarEventId (Postgres)

```json
{
  "parameters": {
    "operation": "executeQuery",
    "query": "UPDATE user_weekly_schedule SET calendar_event_id = '{{ $json.id }}' WHERE day_routine_id = '{{ $('PrepareCalendarEvents').item.json.day_routine_id }}'",
    "options": {}
  },
  "type": "n8n-nodes-base.postgres",
  "typeVersion": 2.6,
  "position": [-2880, 2400],
  "id": "<auto>",
  "name": "UpdateCalendarEventId",
  "continueOnFail": true,
  "credentials": {
    "postgres": { "id": "vZLJtIWG5nYXMez4", "name": "Supabase Memory" }
  }
}
```

**Note:** `$json.id` comes from Google Calendar API response (the created event ID). `$('PrepareCalendarEvents').item.json.day_routine_id` references the input that generated this event. If the Google Calendar node doesn't pass through the original `day_routine_id`, use `$('PrepareCalendarEvents').item` index matching instead.

---

### Task 1.7: Update Connections

Modify `Send message2` in the `connections` object:

```json
"Send message2": {
  "main": [[
    { "node": "Filtered Message4", "type": "main", "index": 0 },
    { "node": "IsScheduleComplete", "type": "main", "index": 0 }
  ]]
}
```

Add new connection entries:

```json
"IsScheduleComplete": {
  "main": [
    [{ "node": "GetCalendarData", "type": "main", "index": 0 }],
    []
  ]
},
"GetCalendarData": {
  "main": [[{ "node": "CreateCalendarMagicLink", "type": "main", "index": 0 }]]
},
"CreateCalendarMagicLink": {
  "main": [[{ "node": "PrepareCalendarEvents", "type": "main", "index": 0 }]]
},
"PrepareCalendarEvents": {
  "main": [[{ "node": "GoogleCalendar_CreateEvent", "type": "main", "index": 0 }]]
},
"GoogleCalendar_CreateEvent": {
  "main": [[{ "node": "UpdateCalendarEventId", "type": "main", "index": 0 }]]
}
```

---

## Phase 2: E2E Test Updates (n8n-agent, ~30 min)

**Can run in parallel with Phase 1 Tasks 1.5-1.7 (test modifications don't depend on node implementation).**

### Task 2.1: Modify TC003 in GymRatFlow_E2E_TestRunner.json

**File:** `n8n/tests/GymRatFlow_E2E_TestRunner.json`
**Node:** `Load Test Cases` (Code node)

Find TC003 test case object. Add a second verification query:

```javascript
// EXISTING query (keep as-is):
{
  sql: "SELECT COUNT(*) as cnt FROM user_weekly_schedule WHERE user_id = 'e2e00001-0000-0000-0000-000000000001'",
  expected: 3
}

// ADD this new query:
{
  sql: "SELECT COUNT(*) as cnt FROM user_weekly_schedule WHERE user_id = 'e2e00001-0000-0000-0000-000000000001' AND calendar_event_id IS NOT NULL",
  expected: 3
}
```

### Task 2.2: Verify TC003 Fixture User Has Email

Confirm that user `e2e00001-0000-0000-0000-000000000001` (phone `570000000001`) has a valid email in the `users` table. If not, update the fixture:

```sql
-- Check:
SELECT email FROM users WHERE user_id = 'e2e00001-0000-0000-0000-000000000001';

-- Fix if needed:
UPDATE users SET email = 'test_noschedule@test.com'
WHERE user_id = 'e2e00001-0000-0000-0000-000000000001';
```

Also add this to `e2e/test_data_setup.sql` if the fixture INSERT doesn't include an email.

---

## Phase 3: Validation (code-reviewer, ~1 hour)

**After Phases 1 and 2 are complete.**

### Task 3.1: Manual Smoke Test

1. Open MAIN_FLOW in n8n
2. Send WhatsApp message as test user: "Quiero agendar mi semana"
3. Complete the scheduling conversation (select days, confirm)
4. Verify:
   - [ ] WhatsApp confirmation received (unchanged)
   - [ ] Google Calendar events appear in the Kairos calendar
   - [ ] User receives calendar invitation emails
   - [ ] Event times match `preferred_schedule`
   - [ ] Event descriptions contain exercise list + magic link
   - [ ] `calendar_event_id` populated in `user_weekly_schedule`

### Task 3.2: Failure Isolation Test

1. Temporarily disconnect Google Calendar credential in n8n
2. Run scheduling flow
3. Verify:
   - [ ] WhatsApp confirmation still works
   - [ ] No errors in workflow execution log
   - [ ] `calendar_event_id` remains NULL (expected)

### Task 3.3: E2E Test Suite

1. Run `GymRatFlow_E2E_TestRunner.json`
2. Verify:
   - [ ] TC003 passes with both verification queries
   - [ ] All other tests unaffected
   - [ ] No test regressions

### Task 3.4: Magic Link Verification

1. Open magic link from calendar event ON a training day
2. Verify workout tracker shows that day's routine
3. Open magic link on a REST day
4. Verify no workout is displayed (expected behavior)

---

## Parallel Execution Map

```
Phase 0 (Prerequisites) ─────── BLOCKING ──────────┐
                                                    │
                                          ┌─────────▼──────────┐
                                          │                    │
                                    Phase 1.1-1.4        Phase 2.1-2.2
                                    (Nodes: IF,          (E2E Test:
                                     Postgres, Code)      modify TC003)
                                          │                    │
                                          ▼                    │
                                    Phase 1.5-1.7              │
                                    (Google Cal node,          │
                                     connections)              │
                                          │                    │
                                          ├────────────────────┘
                                          ▼
                                       Phase 3
                                    (Validation)
```

---

## Files Modified Summary

| File | Owner | Change |
|------|-------|--------|
| `n8n/running_flows/MAIN_FLOW.json` | n8n-agent | Add 6 nodes, update Send message2 connection |
| `n8n/tests/GymRatFlow_E2E_TestRunner.json` | n8n-agent | Add calendar_event_id verification to TC003 |
| `e2e/test_data_setup.sql` | n8n-agent | Ensure TC003 fixture user has email |
| Supabase DB | n8n-agent | `ALTER TABLE user_weekly_schedule ADD COLUMN calendar_event_id TEXT` |

**Files NOT modified:** Go backend, React frontend, WORKOUT_CREATOR.json, other test runners.
