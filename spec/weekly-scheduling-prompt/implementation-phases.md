# Weekly Scheduling Prompt — Implementation Phases (KAN-61)

## Scope

**1 new n8n workflow**. Zero backend changes. Zero DB migrations. Zero frontend changes.

---

## Phase 1: Build Workflow (n8n-agent)

### Task 1.1: Create `WeeklySchedulingPrompt.json`

**Nodes to build** (10 total):

| # | Node Name | Type | Config |
|---|-----------|------|--------|
| 1 | `schedule_trigger_8pm` | `scheduleTrigger` | Daily 20:00, tz `America/Bogota` |
| 2 | `query_users_needing_prompt` | `postgres` | Credential `vZLJtIWG5nYXMez4`, core SQL below |
| 3 | `if_has_results` | `if` (v2.2) | `$input.all().length > 0` |
| 4 | `split_in_batches` | `splitInBatches` | batchSize: 1 |
| 5 | `if_full_completion` | `if` (v2.2) | `$json.completed_count == $json.total_sessions` |
| 6 | `if_zero_completion` | `if` (v2.2) | `$json.completed_count == 0` |
| 7 | `set_celebration_msg` | `set` | Celebration template + next_week calc |
| 8 | `set_growth_msg` | `set` | Growth mindset template |
| 9 | `set_reengagement_msg` | `set` | Re-engagement template |
| 10 | `send_whatsapp` | `whatsApp` (v1.1) | Credential `xIjy4zDHyjIvGQT4`, phoneNumberId `914510145083991` |

**Core SQL** (node 2):
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
SELECT * FROM current_week_stats cws
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

**Node connections**:
```
1 → 2 → 3
         ├─ TRUE → 4 → 5
         │              ├─ TRUE → 7 ──────→ 10 → (back to 4)
         │              └─ FALSE → 6
         │                         ├─ TRUE → 9 → 10
         │                         └─ FALSE → 8 → 10
         └─ FALSE → (end)
```

**WhatsApp messages** (Spanish, set nodes 7/8/9):

Celebration (node 7):
```
Felicidades {{ $json.full_name }}! Completaste todas tus {{ $json.total_sessions }} sesiones de la Semana {{ $json.current_week }}.

Tu constancia es admirable. Listo para seguir con la Semana {{ $json.current_week + 1 }}? Escribeme "agendar" y organizamos tus dias.
```

Growth mindset (node 8):
```
Hola {{ $json.full_name }}! Tu Semana {{ $json.current_week }} ya paso. Completaste {{ $json.completed_count }} de {{ $json.total_sessions }} sesiones.

Cada entrenamiento suma. Quieres programar tu Semana {{ $json.current_week + 1 }}? Escribeme "agendar" y arrancamos.
```

Re-engagement (node 9):
```
Hola {{ $json.full_name }}! Vi que la Semana {{ $json.current_week }} fue dificil.

No pasa nada, lo importante es volver. Quieres intentar con tu Semana {{ $json.current_week + 1 }}? Escribeme "agendar" y planeamos juntos.
```

### Task 1.2: Validate SQL against production data

```sql
-- DRY RUN: Check which users would be prompted (run manually in Supabase)
WITH current_week_stats AS (
    -- same CTE as above
)
SELECT user_id, full_name, current_week, completed_count, total_sessions, last_planned_day
FROM current_week_stats cws
WHERE cws.last_planned_day < TO_CHAR(NOW() AT TIME ZONE 'America/Bogota', 'YYYY-MM-DD')
  AND cws.last_planned_day >= TO_CHAR(
      (NOW() AT TIME ZONE 'America/Bogota') - INTERVAL '3 days', 'YYYY-MM-DD'
  )
  AND NOT EXISTS (
      SELECT 1 FROM user_weekly_schedule uws2
      WHERE uws2.user_id = cws.user_id AND uws2.week = cws.current_week + 1
  );
```

**Acceptance**: Query returns expected users. No test users (`5700000000%`). No week 4 users.

---

## Phase 2: Integration Test

### Task 2.1: Import workflow (inactive)

1. Import `WeeklySchedulingPrompt.json` into n8n
2. Configure Postgres credential: `vZLJtIWG5nYXMez4`
3. Configure WhatsApp credential: `xIjy4zDHyjIvGQT4`
4. **Keep workflow INACTIVE**

### Task 2.2: Manual trigger test

1. Create temporary test user via Supabase SQL:
```sql
-- Create user with completed week 1 (3/3 sessions done, all planned_days in the past)
-- Use existing test user pattern from e2e/test_data_setup.sql
-- Phone: 570000000061, UUID: e2e00061-0000-0000-0000-000000000061
```

2. Manually execute the `schedule_trigger_8pm` node in n8n
3. **Verify**:
   - Query returns the test user
   - Correct message variant selected (celebration for 3/3)
   - WhatsApp message delivered to test number

### Task 2.3: AGENDAR flow test

1. After receiving WhatsApp prompt, reply "agendar" to the bot
2. **Verify**: MAIN_FLOW routes to AI Agent1 (scheduling agent)
3. Complete scheduling flow → verify `user_weekly_schedule` has week 2 entries
4. Re-run workflow → **verify test user no longer appears** (dedup via NOT EXISTS)

### Task 2.4: Edge cases

| Case | Setup | Expected |
|------|-------|----------|
| Week 4 user | User with `uws.week = 4` completed | NOT in query results |
| Already scheduled | User has week 2 in `user_weekly_schedule` | NOT in query results |
| 4+ days since last_planned_day | `last_planned_day` = 5 days ago | NOT in query results (3-day window) |
| Partial completion | 2/3 sessions done | Growth mindset message |
| Zero completion | 0/3 sessions done | Re-engagement message |

---

## Phase 3: Deploy & Monitor

### Task 3.1: Activate

1. Activate workflow
2. Confirm Schedule Trigger is set to 20:00 America/Bogota

### Task 3.2: Monitor (first 3 days)

Check after each 8 PM execution:
- [ ] Workflow executed successfully (n8n execution log)
- [ ] Correct number of users prompted (cross-check with SQL dry run)
- [ ] No duplicate messages to same user on consecutive days (unless they haven't scheduled)
- [ ] Users who scheduled next week stopped receiving prompts

### Task 3.3: Clean up test data

```sql
DELETE FROM user_weekly_schedule WHERE user_id = 'e2e00061-0000-0000-0000-000000000061';
DELETE FROM users_plans WHERE user_id = 'e2e00061-0000-0000-0000-000000000061';
DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000061;
DELETE FROM users WHERE user_id = 'e2e00061-0000-0000-0000-000000000061';
```

---

## Role Assignment

| Role | Tasks | Parallel? |
|------|-------|-----------|
| **n8n-agent** | 1.1 (build workflow) | Start immediately |
| **code-reviewer** | 1.2 (validate SQL) | After 1.1 |
| **kiro-coach** | Review WhatsApp message copy | Parallel with 1.1 |
| **pixel-dev** | NOT NEEDED | - |
| **claude-designer** | NOT NEEDED | - |

---

## Files Created/Modified

| File | Action |
|------|--------|
| `n8n/running_flows/WeeklySchedulingPrompt.json` | **CREATE** |

That's it. One file.
