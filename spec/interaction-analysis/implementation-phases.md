# KAN-96: Implementation Phases

## Phase 1: Database Migration (n8n-agent)

**Task**: Add `created_at` to `n8n_chat_histories`

```sql
ALTER TABLE n8n_chat_histories
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_n8n_chat_histories_created_at
  ON n8n_chat_histories (created_at DESC);
```

**Verify**: `SELECT id, created_at FROM n8n_chat_histories LIMIT 3;`

> Existing rows backfill to migration time. Time-scoped queries become accurate for new data only.

---

## Phase 2: Build Workflow — `InteractionAnalysis.json` (n8n-agent)

### 2A: Triggers + Queries

Create the workflow JSON with:

1. **`schedule_trigger`** — scheduleTrigger v1.3, Monday 8 AM Bogota
2. **`manual_trigger`** — manualTrigger v1
3. **`query_metrics`** — Consolidated Postgres query:

```sql
WITH
classified AS (
  SELECT id, session_id, message, created_at,
    CASE
      WHEN session_id LIKE '%_kyc_v4' THEN 'kyc'
      WHEN session_id LIKE '%_chat' THEN 'chat'
      WHEN session_id LIKE '%_confirmation' THEN 'confirmation'
      WHEN session_id LIKE '%_renewal' THEN 'renewal'
      ELSE 'other'
    END AS session_type,
    (message->>'type') AS msg_type
  FROM n8n_chat_histories
),
session_stats AS (
  SELECT session_id, session_type,
    COUNT(*) AS msg_count,
    COUNT(*) FILTER (WHERE msg_type = 'human') AS human_msgs
  FROM classified GROUP BY session_id, session_type
),
recent AS (
  SELECT * FROM classified WHERE created_at >= NOW() - INTERVAL '7 days'
),
tool_usage AS (
  SELECT message->>'name' AS tool_name, COUNT(*) AS calls
  FROM n8n_chat_histories
  WHERE (message->>'type') = 'tool' AND message->>'name' IS NOT NULL
  GROUP BY message->>'name' ORDER BY calls DESC LIMIT 10
),
kyc_completion AS (
  SELECT
    CASE WHEN u.user_id IS NOT NULL THEN 'completado' ELSE 'abandonado' END AS outcome,
    COUNT(*) AS cnt
  FROM (
    SELECT DISTINCT SPLIT_PART(session_id, '_kyc_v4', 1) AS phone
    FROM classified WHERE session_type = 'kyc'
  ) k
  LEFT JOIN users u ON u.full_phone_number::text = k.phone
  GROUP BY outcome
)
SELECT json_build_object(
  'total_sessions', (SELECT COUNT(DISTINCT session_id) FROM classified),
  'total_messages', (SELECT COUNT(*) FROM classified),
  'sessions_last_7d', (SELECT COUNT(DISTINCT session_id) FROM recent),
  'messages_last_7d', (SELECT COUNT(*) FROM recent),
  'session_breakdown', (SELECT json_agg(json_build_object(
    'type', session_type, 'sessions', COUNT(*),
    'avg_msgs', ROUND(AVG(msg_count), 1), 'max_msgs', MAX(msg_count)
  )) FROM session_stats GROUP BY session_type),
  'tool_usage', (SELECT json_agg(json_build_object('name', tool_name, 'calls', calls)) FROM tool_usage),
  'kyc_completion', (SELECT json_agg(json_build_object('outcome', outcome, 'count', cnt)) FROM kyc_completion),
  'long_sessions', (SELECT json_agg(json_build_object(
    'session_id', session_id, 'type', session_type, 'msgs', msg_count
  )) FROM (SELECT * FROM session_stats ORDER BY msg_count DESC LIMIT 5) t)
) AS metrics;
```

4. **`query_conversation_samples`** — Stratified sampling:

```sql
WITH
classified AS (
  SELECT id, session_id, message,
    CASE
      WHEN session_id LIKE '%_kyc_v4' THEN 'kyc'
      WHEN session_id LIKE '%_chat' THEN 'chat'
      ELSE 'other'
    END AS session_type
  FROM n8n_chat_histories
  WHERE (message->>'type') IN ('human', 'ai')
),
sampled AS (
  (SELECT DISTINCT session_id FROM classified WHERE session_type = 'kyc'
   ORDER BY session_id DESC LIMIT 3)
  UNION
  (SELECT DISTINCT session_id FROM classified WHERE session_type = 'chat'
   ORDER BY session_id DESC LIMIT 3)
  UNION
  (SELECT session_id FROM (
    SELECT session_id, COUNT(*) AS c FROM classified GROUP BY session_id ORDER BY c DESC LIMIT 2
  ) t)
),
ranked AS (
  SELECT c.session_id, c.session_type,
    message->>'type' AS role,
    LEFT(message->>'content', 500) AS content,
    ROW_NUMBER() OVER (PARTITION BY c.session_id ORDER BY c.id) AS rn
  FROM classified c WHERE c.session_id IN (SELECT session_id FROM sampled)
)
SELECT session_id, session_type, role, content
FROM ranked WHERE rn <= 30
ORDER BY session_id, rn;
```

### 2B: Processing + LLM + Email

5. **`build_analysis_payload`** (Code v2, Run Once for All Items):
   - Read `$('query_metrics').first().json` and `$('query_conversation_samples').all()`
   - Group samples by session_id
   - Output: `{ metrics_summary, conversations_text, report_date }`

6. **`llm_analysis`** (Agent v3.1, no tools):
   - System prompt: Spanish product analyst, 7 sections
   - Input: metrics summary + conversation samples
   - Model: gpt-4.1-mini via `openai_model` sub-node

7. **`format_email`** (Code v2):
   - Convert LLM markdown to HTML
   - Wrap in email template (dark header, clean body)
   - Output: `{ html, subject, to }`

8. **`send_email`** (Gmail v2.1):
   - To: `camilodiazjaimes@gmail.com`
   - Credential: `H1GF1YdmcaZ0gfWB`

---

## Phase 3: Verify (manual)

1. Import workflow into n8n
2. Execute via manual trigger
3. Confirm email arrives with:
   - Correct metrics (cross-check with raw SQL)
   - LLM analysis covering all 7 sections
   - Clean HTML rendering
4. Activate workflow for weekly schedule

---

## Parallel Execution Map

```
Phase 1 (Migration)     ─── can start immediately
Phase 2A (Queries)       ─── depends on Phase 1
Phase 2B (Code+LLM)     ─── depends on 2A
Phase 3 (Verify)         ─── depends on 2B
```

All phases are sequential for this feature (single workflow, no UI).
