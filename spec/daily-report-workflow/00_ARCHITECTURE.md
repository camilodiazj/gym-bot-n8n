# 00_ARCHITECTURE.md — Daily Report Workflow (KAN-97)

## Overview

Single n8n workflow (`DailyReport.json`) that runs daily at 6 AM (America/Bogota). Queries 7 categories of operational data from existing Supabase tables, aggregates into HTML email + WhatsApp summary, and delivers to Kairos Soporte team.

**Zero new infrastructure.** No new DB tables, no backend changes, no frontend changes.

## Data Flow

```
Schedule Trigger (6 AM Bogota)
  → query_health_violations (Postgres)
  → query_plan_failures (Postgres)
  → query_new_users (Postgres)
  → query_engagement (Postgres)
  → query_churn_risk (Postgres)
  → query_never_completed (Postgres)
  → query_abandoned_kyc (Postgres)
  → aggregate_and_format (Code)
  → [parallel] send_email_report (Email) + send_whatsapp_summary (WhatsApp)
```

Sequential Postgres chain — each <100ms, total ~700ms. Code node references all upstream via `$('node_name').all()`. No Merge nodes needed.

## Node Inventory (11 nodes)

| # | Node | Type | Version | Config |
|---|------|------|---------|--------|
| 1 | `schedule_trigger_6am` | scheduleTrigger | 1.3 | `triggerAtHour: 6`, tz: America/Bogota |
| 2 | `query_health_violations` | postgres | 2.6 | Health C/D/E exercise violations |
| 3 | `query_plan_failures` | postgres | 2.6 | KYC→plan and plan→workouts gaps (7d) |
| 4 | `query_new_users` | postgres | 2.6 | Scalar counts: new users, KYC, plans (24h) + totals |
| 5 | `query_engagement` | postgres | 2.6 | 7-day completion rate, active users yesterday |
| 6 | `query_churn_risk` | postgres | 2.6 | 2+ consecutive misses (14d window) |
| 7 | `query_never_completed` | postgres | 2.6 | Active plan >3d, zero completions |
| 8 | `query_abandoned_kyc` | postgres | 2.6 | Chat history `*_kyc_v4` with no `users_gym_profile` |
| 9 | `aggregate_and_format` | code | 2 | Builds HTML + WhatsApp text |
| 10 | `send_email_report` | emailSend | 1 | `continueOnFail: true` |
| 11 | `send_whatsapp_summary` | whatsApp | 1.1 | `continueOnFail: true` |

All Postgres nodes: credential `vZLJtIWG5nYXMez4` ("Supabase Memory"), `executeOnce: true`, `alwaysOutputData: true`.

## SQL Queries

### Query 1: `query_health_violations`

Detects users with health restrictions whose generated workouts contain violating exercises. Reuses patterns from `spec/workout_creator_quality_fixes/`.

```sql
-- Health C: overhead pressing violations
SELECT
  u.full_name,
  u.full_phone_number,
  gp.health_status,
  e.spanish_name,
  e.pattern,
  w.day_name,
  w.week,
  'C: overhead/push_v violation' AS violation_type
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u ON w.user_id = u.user_id
JOIN users_gym_profile gp ON u.full_phone_number::text = gp.whatsapp_id::text
WHERE gp.health_status = 'C'
  AND (
    e.pattern = 'push_v'
    OR e.spanish_name ILIKE ANY(ARRAY[
      '%press militar%', '%overhead%',
      '%por encima de la cabeza%',
      '%detrás del cuello%', '%behind%neck%',
      '%push press%', '%jerk%', '%snatch%',
      '%remo al mentón%', '%upright row%'
    ])
  )

UNION ALL

-- Health D: heavy axial loading violations
SELECT
  u.full_name, u.full_phone_number, gp.health_status,
  e.spanish_name, e.pattern, w.day_name, w.week,
  'D: axial loading violation' AS violation_type
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u ON w.user_id = u.user_id
JOIN users_gym_profile gp ON u.full_phone_number::text = gp.whatsapp_id::text
WHERE gp.health_status = 'D'
  AND (
    (e.spanish_name ILIKE '%peso muerto%'
     OR e.spanish_name ILIKE '%sentadilla%barra%'
     OR e.spanish_name ILIKE '%good morning%'
     OR e.spanish_name ILIKE '%rack pull%'
     OR e.spanish_name ILIKE '%clean%'
     OR e.spanish_name ILIKE '%snatch%')
    AND e.equipment IN ('barbell', 'smith_machine', 'trap_bar')
  )

UNION ALL

-- Health E: non-machine exercise violations
SELECT
  u.full_name, u.full_phone_number, gp.health_status,
  e.spanish_name, e.pattern, w.day_name, w.week,
  'E: non-machine exercise violation' AS violation_type
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u ON w.user_id = u.user_id
JOIN users_gym_profile gp ON u.full_phone_number::text = gp.whatsapp_id::text
WHERE gp.health_status = 'E'
  AND e.equipment NOT IN ('machine', 'bodyweight', 'cable')

ORDER BY health_status, full_name, week, day_name;
```

### Query 2: `query_plan_failures`

Users stuck in onboarding funnel (last 7 days).

```sql
SELECT
  'kyc_no_plan' AS failure_type,
  gp.full_name,
  gp.whatsapp_id,
  gp.submission_date,
  NULL::text AS plan_status
FROM users_gym_profile gp
LEFT JOIN users u ON u.full_phone_number::text = gp.whatsapp_id::text
LEFT JOIN users_plans up ON u.user_id = up.user_id
WHERE up.plan_id IS NULL
  AND gp.submission_date >= NOW() - INTERVAL '7 days'

UNION ALL

SELECT
  'plan_no_workouts' AS failure_type,
  u.full_name,
  u.full_phone_number::text AS whatsapp_id,
  up.start_date AS submission_date,
  up.status AS plan_status
FROM users_plans up
JOIN users u ON up.user_id = u.user_id
LEFT JOIN workouts w ON up.user_id = w.user_id
WHERE w.id IS NULL
  AND up.start_date >= NOW() - INTERVAL '7 days'

ORDER BY submission_date DESC;
```

### Query 3: `query_new_users`

Scalar counts for the last 24 hours + platform totals.

```sql
SELECT
  (SELECT COUNT(*) FROM users
   WHERE created_at >= (NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '1 day'
     AND created_at < (NOW() AT TIME ZONE 'America/Bogota')::date
  ) AS new_users_count,

  (SELECT COUNT(*) FROM users_gym_profile
   WHERE submission_date >= (NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '1 day'
     AND submission_date < (NOW() AT TIME ZONE 'America/Bogota')::date
  ) AS kyc_completions_count,

  (SELECT COUNT(*) FROM users_plans
   WHERE start_date >= (NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '1 day'
     AND start_date < (NOW() AT TIME ZONE 'America/Bogota')::date
  ) AS plans_generated_count,

  (SELECT COUNT(*) FROM users) AS total_users,
  (SELECT COUNT(*) FROM users_plans WHERE status = 'active') AS total_active_plans;
```

### Query 4: `query_engagement`

7-day rolling window completion metrics.

```sql
SELECT
  COUNT(*) FILTER (WHERE uws."Completed" = true) AS completed_sessions,
  COUNT(*) AS total_scheduled_sessions,
  ROUND(
    100.0 * COUNT(*) FILTER (WHERE uws."Completed" = true) / NULLIF(COUNT(*), 0), 1
  ) AS completion_rate_pct,

  COUNT(DISTINCT uws.user_id) FILTER (
    WHERE uws.planned_day = TO_CHAR((NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '1 day', 'YYYY-MM-DD')
  ) AS active_users_yesterday,

  COUNT(DISTINCT uws.user_id) FILTER (
    WHERE uws.planned_day = TO_CHAR((NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '1 day', 'YYYY-MM-DD')
      AND uws."Completed" = true
  ) AS completed_users_yesterday

FROM user_weekly_schedule uws
WHERE uws.planned_day >= TO_CHAR((NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '7 days', 'YYYY-MM-DD')
  AND uws.planned_day < TO_CHAR((NOW() AT TIME ZONE 'America/Bogota')::date, 'YYYY-MM-DD');
```

### Query 5: `query_churn_risk`

Users with 2+ missed sessions in the last 14 days.

```sql
WITH recent_sessions AS (
  SELECT
    uws.user_id, u.full_name, u.full_phone_number,
    uws.planned_day, uws."Completed", uws.week,
    ROW_NUMBER() OVER (PARTITION BY uws.user_id ORDER BY uws.planned_day DESC) AS rn
  FROM user_weekly_schedule uws
  JOIN users u ON uws.user_id = u.user_id
  WHERE uws.planned_day <= TO_CHAR((NOW() AT TIME ZONE 'America/Bogota')::date, 'YYYY-MM-DD')
    AND uws.planned_day >= TO_CHAR((NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '14 days', 'YYYY-MM-DD')
),
streak AS (
  SELECT
    user_id, full_name, full_phone_number,
    COUNT(*) FILTER (WHERE "Completed" = false AND rn <= 5) AS recent_misses,
    MAX(CASE WHEN "Completed" = true THEN planned_day END) AS last_completed_day,
    MAX(week) AS current_week
  FROM recent_sessions
  GROUP BY user_id, full_name, full_phone_number
)
SELECT
  full_name, full_phone_number, recent_misses,
  last_completed_day, current_week,
  CASE
    WHEN last_completed_day IS NULL THEN 'Nunca completo'
    ELSE (NOW()::date - last_completed_day::date)::text || ' dias inactivo'
  END AS inactivity_status
FROM streak
WHERE recent_misses >= 2
ORDER BY recent_misses DESC, full_name;
```

### Query 6: `query_never_completed`

Users with active plans (>3 days old) and zero completed sessions.

```sql
SELECT
  u.full_name, u.full_phone_number,
  up.start_date, up.week_schedule,
  COUNT(uws.day_routine_id) AS total_scheduled
FROM users u
JOIN users_plans up ON u.user_id = up.user_id AND up.status = 'active'
JOIN user_weekly_schedule uws ON u.user_id = uws.user_id
WHERE NOT EXISTS (
  SELECT 1 FROM user_weekly_schedule uws2
  WHERE uws2.user_id = u.user_id AND uws2."Completed" = true
)
AND up.start_date <= NOW() - INTERVAL '3 days'
GROUP BY u.full_name, u.full_phone_number, up.start_date, up.week_schedule
ORDER BY up.start_date;
```

### Query 7: `query_abandoned_kyc`

Users who started KYC conversation but never completed the survey. Detects via `n8n_chat_histories` sessions ending in `_kyc_v4` with no corresponding `users_gym_profile` entry.

```sql
SELECT
  SUBSTRING(nch.session_id FROM '^[0-9]+') AS whatsapp_id,
  MAX(nch.id) AS last_message_id,
  COUNT(*) AS message_count
FROM n8n_chat_histories nch
WHERE nch.session_id LIKE '%_kyc_v4'
  AND NOT EXISTS (
    SELECT 1 FROM users_gym_profile ugp
    WHERE ugp.whatsapp_id::text = SUBSTRING(nch.session_id FROM '^[0-9]+')
  )
GROUP BY 1
ORDER BY last_message_id DESC;
```

## Code Node: `aggregate_and_format`

Single Code node (mode: "Run Once for All Items", `executeOnce: true`).

**Inputs**: Reads all 7 upstream query results by node name.
**Outputs**: `{ html, whatsappMessage, subject, reportDate }`

### WhatsApp Summary Format

```
Reporte Diario GymBot — DD/MM/YYYY

Nuevos: X usuarios | X planes generados
KYC abandonados: X empezaron pero no terminaron
Alertas: X violaciones salud | X fallos generacion
Completamiento 7d: XX%
Riesgo abandono: X usuarios (2+ sesiones perdidas)

Reporte completo enviado por email
```

### HTML Email Structure

Inline CSS (same pattern as `spec/email-routine-week1/02-HTML-TEMPLATE-SPEC.md`):
- **Header**: Dark navy `#1a1a2e`, white text, report date
- **Section 1 — ACCION INMEDIATA** (red alert boxes): Health violations table + plan failures table. Only rendered if alerts exist.
- **Section 2 — NUEVOS USUARIOS & ONBOARDING**: Metrics table (new users, KYC, plans, totals, funnel drop-off) + abandoned KYC table
- **Section 3 — ENGAGEMENT**: Completion rate (color-coded), sessions, active users
- **Section 4 — RIESGO DE ABANDONO**: Churn risk table + never-completed table
- **Footer**: Auto-generated timestamp

Color palette: `#e63946` (section borders), `#fee2e2` (alert bg), `#fff3cd` (warning bg), `#d1fae5` (success bg), `#f1f3f5` (table headers).

## Connection Map

```json
{
  "schedule_trigger_6am": { "main": [[{ "node": "query_health_violations", "type": "main", "index": 0 }]] },
  "query_health_violations": { "main": [[{ "node": "query_plan_failures", "type": "main", "index": 0 }]] },
  "query_plan_failures": { "main": [[{ "node": "query_new_users", "type": "main", "index": 0 }]] },
  "query_new_users": { "main": [[{ "node": "query_engagement", "type": "main", "index": 0 }]] },
  "query_engagement": { "main": [[{ "node": "query_churn_risk", "type": "main", "index": 0 }]] },
  "query_churn_risk": { "main": [[{ "node": "query_never_completed", "type": "main", "index": 0 }]] },
  "query_never_completed": { "main": [[{ "node": "query_abandoned_kyc", "type": "main", "index": 0 }]] },
  "query_abandoned_kyc": { "main": [[{ "node": "aggregate_and_format", "type": "main", "index": 0 }]] },
  "aggregate_and_format": { "main": [[
    { "node": "send_email_report", "type": "main", "index": 0 },
    { "node": "send_whatsapp_summary", "type": "main", "index": 0 }
  ]] }
}
```

## Error Handling

| Node | Strategy |
|------|----------|
| Postgres nodes | `alwaysOutputData: true` — empty results still flow |
| `aggregate_and_format` | Defensive `|| 0` / `|| []` for all metrics |
| `send_email_report` | `continueOnFail: true` |
| `send_whatsapp_summary` | `continueOnFail: true` |

## Credentials

| Credential | ID | Used By |
|------------|-----|---------|
| Postgres (Supabase) | `vZLJtIWG5nYXMez4` | All 7 query nodes |
| WhatsApp Business | `xIjy4zDHyjIvGQT4` | `send_whatsapp_summary` |
| SMTP | `<CONFIGURE_IN_N8N_UI>` | `send_email_report` |

## Tables Queried (no new tables)

`users`, `users_gym_profile`, `users_plans`, `workouts`, `exercises`, `user_weekly_schedule`, `pending_tasks`, `n8n_chat_histories`
