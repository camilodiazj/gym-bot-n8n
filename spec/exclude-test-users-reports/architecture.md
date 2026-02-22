# KAN-108: Exclude Test Users from Reports — Architecture

## Problem

DailyReport and InteractionAnalysis workflows include test user data (phones 570000000000-570000000799) in production metrics, polluting operational reports.

## Solution

Add SQL WHERE filters to exclude test phone ranges at query level. No new tables, endpoints, or infrastructure.

## Scope

| Workflow | File | Nodes Affected |
|----------|------|----------------|
| DailyReport | `n8n/running_flows/DailyReport.json` | 7 Postgres query nodes |
| InteractionAnalysis | `n8n/running_flows/InteractionAnalysis.json` | 2 Postgres query nodes |

## Filter Strategy

Single exclusion clause adapted per column type:

| Column Source | Filter Clause |
|--------------|---------------|
| `users.full_phone_number` | `AND u.full_phone_number::bigint NOT BETWEEN 570000000000 AND 570000000799` |
| `users_gym_profile.whatsapp_id` | `AND gp.whatsapp_id::bigint NOT BETWEEN 570000000000 AND 570000000799` |
| `n8n_chat_histories.session_id` | `AND SUBSTRING(session_id FROM '^[0-9]+')::bigint NOT BETWEEN 570000000000 AND 570000000799` |

## Test Phone Ranges Covered

| Group | Range | Current Phones |
|-------|-------|----------------|
| GYM | 570000000001-570000000009 | 5 phones |
| MESOCYCLE | 570000000051-570000000053 | 3 phones |
| WSP | 570000000071-570000000073 | 3 phones |
| HOME | 570000000211-570000000213 | 3 phones |

Range 570000000000-570000000799 covers all current + future test phones.

## Data Flow (unchanged)

```
Cron trigger → Postgres queries (+ exclusion filter) → Aggregate → Format HTML → Send Email/WhatsApp
```

No changes to aggregation, formatting, or delivery nodes.
