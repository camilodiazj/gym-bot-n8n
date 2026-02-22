# KAN-108: Implementation Phases

## Phase 1: DailyReport.json (7 SQL nodes)

All changes are SQL-only edits within existing Postgres nodes.

### 1.1 `query_health_violations`
- Add `AND u.full_phone_number::bigint NOT BETWEEN 570000000000 AND 570000000799` to each of the 3 UNION ALL WHERE blocks (health C, D, E)

### 1.2 `query_plan_failures`
- Part 1 (kyc_no_plan): Add `AND gp.whatsapp_id::bigint NOT BETWEEN 570000000000 AND 570000000799`
- Part 2 (plan_no_workouts): Add `AND u.full_phone_number::bigint NOT BETWEEN 570000000000 AND 570000000799`

### 1.3 `query_new_users`
- Subquery 1 (new users count): Add `AND full_phone_number::bigint NOT BETWEEN 570000000000 AND 570000000799`
- Subquery 2 (KYC completions): Add `AND whatsapp_id::bigint NOT BETWEEN 570000000000 AND 570000000799`
- Subquery 3 (plans generated): Add JOIN to users + filter on `full_phone_number`
- Subquery 4 (total users): Add `WHERE full_phone_number::bigint NOT BETWEEN 570000000000 AND 570000000799`
- Subquery 5 (total active plans): Add JOIN to users + filter on `full_phone_number`

### 1.4 `query_engagement`
- Add `JOIN users u ON uws.user_id = u.user_id`
- Add `AND u.full_phone_number::bigint NOT BETWEEN 570000000000 AND 570000000799` to WHERE

### 1.5 `query_churn_risk`
- Add filter in `recent_sessions` CTE WHERE clause

### 1.6 `query_never_completed`
- Add filter in outer WHERE clause

### 1.7 `query_abandoned_kyc`
- Add `AND SUBSTRING(nch.session_id FROM '^[0-9]+')::bigint NOT BETWEEN 570000000000 AND 570000000799`

## Phase 2: InteractionAnalysis.json (2 SQL nodes)

### 2.1 `query_metrics`
- Add WHERE clause to `classified` CTE: `WHERE SUBSTRING(session_id FROM '^[0-9]+')::bigint NOT BETWEEN 570000000000 AND 570000000799`
- Add same filter to `tool_usage` CTE (queries `n8n_chat_histories` directly, not via `classified`)

### 2.2 `query_conversation_samples`
- Add filter to existing WHERE in `classified` CTE

## Verification

1. Run DailyReport manually → confirm no test phones in any of the 7 category outputs
2. Run InteractionAnalysis manually → confirm metrics exclude test sessions and conversation samples don't include test phones
3. Validate JSON syntax of both workflow files (valid n8n import)
