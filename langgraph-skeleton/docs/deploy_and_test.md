# Kairos Agent — Deploy & Test Guide

## Prerequisites

- `gcloud` CLI authenticated
- GCP Project: `gen-lang-client-0432163259`
- Python venv: `langgraph-skeleton/.venv/`

## Unit Tests

```bash
cd langgraph-skeleton
.venv/bin/python -m pytest tests/test_case6.py -v
```

## Deploy to Cloud Run

```bash
cd langgraph-skeleton
gcloud run deploy kairos-agent \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --project gen-lang-client-0432163259
```

- Uses `Dockerfile` in the directory (Python 3.11-slim + uvicorn)
- Env vars are already configured in the Cloud Run service — no need to pass `--set-env-vars` on subsequent deploys
- Build + deploy takes ~5 minutes
- Service URL: `https://kairos-agent-148665080566.us-central1.run.app`

## Verify Deployment

```bash
# Health check
curl -s https://kairos-agent-148665080566.us-central1.run.app/ | python3 -m json.tool

# Should return: {"status": "ok", ...}
```

## Test via `/api/v1/chat` Endpoint

Send messages as any test user. The endpoint simulates a WhatsApp conversation without going through WhatsApp.

```bash
curl -s -X POST https://kairos-agent-148665080566.us-central1.run.app/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "570000000001",
    "display_name": "Test_NoSchedule",
    "message": "Hola, que me toca hoy?"
  }' | python3 -m json.tool
```

### Test Users (from `e2e/test_data_setup.sql`)

| Phone | User | Has Plan | Has Schedule | Has Pending Task |
|-------|------|----------|--------------|------------------|
| `570000000001` | Test_NoSchedule | Yes | No | No |
| `570000000002` | Test_RestDay | Yes | Yes (not today) | No |
| `570000000003` | Test_WithRoutine | Yes | Yes (today) | No |
| `570000000004` | Test_WithPendingTask | Yes | Yes | Yes |
| `570000000009` | (dynamic) | No | No | No |

### Common Test Scenarios

**Schedule sessions** (tests `schedule_sessions` tool):
```bash
curl -s -X POST .../api/v1/chat -H "Content-Type: application/json" \
  -d '{"phone_number": "570000000001", "display_name": "Test", "message": "Agenda lunes 24/03, miercoles 26/03 y viernes 28/03"}'
```

**View routine** (tests `get_todays_routine` tool):
```bash
curl -s -X POST .../api/v1/chat -H "Content-Type: application/json" \
  -d '{"phone_number": "570000000003", "display_name": "Test", "message": "Que me toca hoy?"}'
```

**Confirm workout** (tests `confirm_workout_completion` tool):
```bash
curl -s -X POST .../api/v1/chat -H "Content-Type: application/json" \
  -d '{"phone_number": "570000000004", "display_name": "Test", "message": "Si, ya termine mi rutina"}'
```

### Verify in Supabase

After testing, check results directly in the DB:

```sql
-- Check schedule entries
SELECT week_day, session_name, planned_day, "Completed"
FROM user_weekly_schedule
WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = '570000000001');

-- Clean up test data
DELETE FROM user_weekly_schedule
WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = '570000000001');
```

## View Conversation History

```bash
curl -s "https://kairos-agent-148665080566.us-central1.run.app/api/v1/chat/570000000001/history" | python3 -m json.tool
```

## Swagger Docs

Interactive API docs at: `https://kairos-agent-148665080566.us-central1.run.app/docs`
