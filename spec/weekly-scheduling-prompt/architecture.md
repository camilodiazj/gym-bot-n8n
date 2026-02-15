# Weekly Scheduling Prompt — Architecture (KAN-61)

## Summary

One new n8n workflow (`WeeklySchedulingPrompt.json`) triggers daily at 8 PM Bogota time. It queries users whose current training week (1-3) has ended but who haven't scheduled week+1 yet. It sends a WhatsApp message differentiated by completion rate. When the user replies, MAIN_FLOW's existing Intention Agent classifies it as AGENDAR and routes to the scheduling agent. Zero backend changes. Zero new tables.

---

## Detection Logic

A user qualifies for a scheduling prompt when ALL conditions hold:

1. `users_plans.status = 'active'`
2. Current scheduled week is 1, 2, or 3 (week 4 triggers mesocycle renewal instead)
3. `MAX(user_weekly_schedule.planned_day)` for that week is strictly before today (Bogota time)
4. No `user_weekly_schedule` rows exist for `week + 1`
5. `last_planned_day` is within the last 3 days (dedup: stops prompting after 3 days of no response)

All data comes from existing tables: `users`, `users_plans`, `week_schedules`, `user_weekly_schedule`.

---

## Core SQL Query

```sql
WITH current_week_stats AS (
    SELECT
        u.user_id,
        u.full_phone_number,
        u.full_name,
        uws.week AS current_week,
        ws.days_per_week AS total_sessions,
        COUNT(*) FILTER (WHERE uws."Completed" = true) AS completed_count,
        MAX(uws.planned_day) AS last_planned_day
    FROM users u
    JOIN users_plans up ON u.user_id = up.user_id AND up.status = 'active'
    JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
    JOIN user_weekly_schedule uws ON u.user_id = uws.user_id
    WHERE uws.week BETWEEN 1 AND 3
    GROUP BY u.user_id, u.full_phone_number, u.full_name, uws.week, ws.days_per_week
)
SELECT
    user_id,
    full_phone_number,
    full_name,
    current_week,
    total_sessions,
    completed_count,
    last_planned_day
FROM current_week_stats cws
WHERE cws.last_planned_day < TO_CHAR(NOW() AT TIME ZONE 'America/Bogota', 'YYYY-MM-DD')
  AND cws.last_planned_day >= TO_CHAR(
      (NOW() AT TIME ZONE 'America/Bogota') - INTERVAL '3 days', 'YYYY-MM-DD'
  )
  AND NOT EXISTS (
      SELECT 1 FROM user_weekly_schedule uws2
      WHERE uws2.user_id = cws.user_id
        AND uws2.week = cws.current_week + 1
  );
```

**Query behavior:**
- Groups by `(user_id, week)` to get per-week stats.
- `FILTER (WHERE "Completed" = true)` counts completed sessions without a subquery.
- The 3-day window (`last_planned_day >= today - 3 days`) prevents indefinite daily messages. After 3 days with no scheduling action, the user drops out silently.
- `NOT EXISTS` on `week + 1` ensures users who already scheduled next week are excluded. This is the primary dedup mechanism — once scheduled, they never appear again.

---

## Workflow Node Structure

```
schedule_trigger_8pm
    |
    v
query_users_needing_prompt          [Postgres node, credential vZLJtIWG5nYXMez4]
    |
    v
if_has_results                      [If node: {{$json.length}} > 0]
    |
    v (TRUE)
split_in_batches                    [SplitInBatches: batchSize=1]
    |
    v
if_full_completion                  [If: {{$json.completed_count}} == {{$json.total_sessions}}]
    |                       |
    v (TRUE)                v (FALSE)
set_celebration_msg     if_zero_completion      [If: {{$json.completed_count}} == 0]
    |                       |               |
    |                   v (TRUE)        v (FALSE)
    |               set_reengagement   set_growth_msg
    |                   |               |
    v                   v               v
merge_messages          <───────────────┘
    |
    v
send_whatsapp_message               [WhatsApp Cloud API: Send Message]
    |
    v
(loop back to split_in_batches)
```

### Node Configuration Details

**`schedule_trigger_8pm`**
- Type: `n8n-nodes-base.scheduleTrigger`
- Rule: Every day at 20:00, timezone `America/Bogota`

**`query_users_needing_prompt`**
- Type: `n8n-nodes-base.postgres`
- Credential: `vZLJtIWG5nYXMez4`
- Operation: `executeQuery`
- Query: Core SQL above

**`if_has_results`**
- Type: `n8n-nodes-base.if` (typeVersion 2.2)
- Condition: `{{ $input.all().length > 0 }}`
- `alwaysOutputData: true` on FALSE branch (no-op, workflow ends cleanly)

**`split_in_batches`**
- Type: `n8n-nodes-base.splitInBatches`
- Batch size: 1
- Processes each user sequentially to avoid WhatsApp rate limits

**`if_full_completion`**
- Condition: `{{ $json.completed_count == $json.total_sessions }}`

**`if_zero_completion`**
- Condition: `{{ $json.completed_count == 0 }}`

**`set_celebration_msg`** / **`set_growth_msg`** / **`set_reengagement_msg`**
- Type: `n8n-nodes-base.set`
- Each sets a `message` field with the appropriate template (see Messages below)
- Each computes `next_week` as `{{ $json.current_week + 1 }}`

**`merge_messages`**
- Type: `n8n-nodes-base.merge`
- Mode: `chooseBranch` (passthrough whichever branch has data)

**`send_whatsapp_message`**
- Type: `n8n-nodes-base.httpRequest` (WhatsApp Cloud API)
- URL: `https://graph.facebook.com/v21.0/{{PHONE_NUMBER_ID}}/messages`
- Body: `{ "messaging_product": "whatsapp", "to": "{{$json.full_phone_number}}", "type": "text", "text": { "body": "{{$json.message}}" } }`

---

## WhatsApp Messages (Spanish)

### Celebration (completed_count == total_sessions)

```
Felicidades {{full_name}}! Completaste todas tus {{total_sessions}} sesiones de la Semana {{current_week}}.

Tu constancia es admirable. Listo para seguir con la Semana {{next_week}}? Escribeme "agendar" y organizamos tus dias.
```

### Growth Mindset (0 < completed_count < total_sessions)

```
Hola {{full_name}}! Tu Semana {{current_week}} ya paso. Completaste {{completed_count}} de {{total_sessions}} sesiones.

Cada entrenamiento suma. Quieres programar tu Semana {{next_week}}? Escribeme "agendar" y arrancamos.
```

### Re-engagement (completed_count == 0)

```
Hola {{full_name}}! Vi que la Semana {{current_week}} fue dificil.

No pasa nada, lo importante es volver. Quieres intentar con tu Semana {{next_week}}? Escribeme "agendar" y planeamos juntos.
```

> Emojis omitted intentionally — add per brand guidelines during implementation if desired.

---

## Dedup Strategy

No new table. Two mechanisms:

1. **SQL-level exclusion**: `NOT EXISTS (... week = current_week + 1)` means the moment a user schedules next week, they vanish from the result set permanently for that transition.
2. **3-day window**: `last_planned_day >= today - 3 days` caps repeated prompts at 3 consecutive evenings. After that, silence. This handles users who ghost — no infinite nagging.

**Edge case**: A user could receive up to 3 identical-category messages (same completion rate, same week). This is acceptable behavior — it's a gentle nudge, not a notification storm.

---

## Data Flow Diagram

```
                     8 PM Daily (America/Bogota)
                              |
                              v
                    +--------------------+
                    | schedule_trigger_   |
                    | 8pm                 |
                    +--------+-----------+
                             |
                             v
                    +--------------------+
                    | query_users_       |     Reads from:
                    | needing_prompt     |---> user_weekly_schedule
                    | (Postgres)         |     users_plans
                    +--------+-----------+     week_schedules
                             |                 users
                             v
                    +--------------------+
                    | if_has_results     |
                    +----+----------+----+
                         |          |
                    TRUE |     FALSE| (end)
                         v
                    +--------------------+
                    | split_in_batches   |<-----------+
                    +--------+-----------+            |
                             |                        |
                             v                        |
                    +--------------------+            |
                    | if_full_completion |            |
                    +----+----------+----+            |
                    TRUE |     FALSE|                  |
                         v          v                  |
               +-----------+  +------------------+    |
               |celebration|  |if_zero_completion|    |
               |_msg       |  +---+----------+---+    |
               +-----+-----+TRUE |     FALSE |        |
                     |       v          v             |
                     | +----------+ +----------+      |
                     | |reengage_ | |growth_   |      |
                     | |msg       | |msg       |      |
                     | +----+-----+ +----+-----+      |
                     |      |            |             |
                     v      v            v             |
                    +--------------------+             |
                    | merge_messages     |             |
                    +--------+-----------+             |
                             |                        |
                             v                        |
                    +--------------------+             |
                    | send_whatsapp_     |             |
                    | message            |             |
                    +--------+-----------+             |
                             |                        |
                             +------------------------+
                             (loop)


    --- User responds "agendar" (async, separate flow) ---

                    +--------------------+
                    | MAIN_FLOW          |
                    | WhatsApp webhook   |
                    +--------+-----------+
                             |
                             v
                    +--------------------+
                    | Intention Agent    |
                    | detects: AGENDAR   |
                    +--------+-----------+
                             |
                             v
                    +--------------------+
                    | AI Agent1          |
                    | (scheduling agent) |
                    +--------------------+
```

---

## Integration Points

| System | Interaction | Direction |
|--------|-------------|-----------|
| `user_weekly_schedule` | Read: planned_day, Completed, week | Query |
| `users_plans` | Read: status, week_schedule | Query |
| `week_schedules` | Read: days_per_week | Query |
| `users` | Read: full_phone_number, full_name | Query |
| WhatsApp Cloud API | Send text message | Outbound |
| MAIN_FLOW | User reply triggers AGENDAR intent | Indirect (no coupling) |

**Zero modifications to**: MAIN_FLOW, WORKOUT_CREATOR, MorningReminder, MesocycleRenewal, Go backend, React frontend.

---

## Week 4 Boundary

This workflow explicitly excludes week 4 via `WHERE uws.week BETWEEN 1 AND 3`. Week 4 completion is handled by `GymBotMesocycleRenewal.json` which detects W4 completion and triggers the renewal flow. There is no overlap — a user finishing week 3 gets a scheduling prompt for week 4; a user finishing week 4 gets a mesocycle renewal prompt from a different workflow.

---

## Error Handling

- **Empty result set**: `if_has_results` FALSE branch ends workflow. No error, no log noise.
- **WhatsApp API failure**: n8n's built-in retry (configure 2 retries with 5s backoff on `send_whatsapp_message` node).
- **Postgres timeout**: Unlikely given query simplicity (<100 users). No special handling needed at current scale.
- **Duplicate sends within same day**: Not possible — Schedule Trigger fires exactly once per day. If workflow errors mid-batch and re-executes, some users may get duplicate messages that day. Acceptable at current scale.
