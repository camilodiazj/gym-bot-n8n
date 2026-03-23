---
name: deploy
description: "Deploy all Kairos services to production. Use this skill whenever the user says 'deploy', 'push to prod', 'desplegar', 'subir a produccion', 'deploy backend', 'deploy frontend', 'deploy kairos', or any variation of deploying one or more of the three services (Go backend, React frontend, Kairos agent). Also use when the user mentions Cloud Run deploy, Firebase deploy, or wants to update production after code changes."
---

# Deploy Kairos Services

Deploys the three Kairos services to production. Each can be deployed independently or all together.

## Services Overview

| Service | Platform | URL | Source |
|---------|----------|-----|--------|
| **Backend** (Go/Gin) | Google Cloud Run | `https://workout-api-148665080566.us-central1.run.app` | `workout-tracker-back/` |
| **Frontend** (React/Vite) | Firebase Hosting | `https://workout-tracker-69b08.web.app` | `workout-tracker/` |
| **Kairos Agent** (Python/LangGraph) | Google Cloud Run | `https://kairos-agent-148665080566.us-central1.run.app` | `langgraph-skeleton/` |

**GCP Project**: `gen-lang-client-0432163259`
**Firebase Project**: `workout-tracker-69b08`

## Deployment Steps

Ask the user which services to deploy, or deploy all three if they say "deploy everything" / "deploy all".

### 1. Go Backend (Cloud Run)

The backend needs ALL env vars passed with `--set-env-vars` because Cloud Run replaces them entirely (not merges).

**CRITICAL**: Do NOT include `PORT` in env vars — Cloud Run reserves it and the deploy will fail.

```bash
cd workout-tracker-back

# Get Supabase DB URL from langgraph .env
SUPABASE_DB_URL=$(grep SUPABASE_DB_URL ../langgraph-skeleton/.env | cut -d'=' -f2-)

gcloud run deploy workout-api \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --project gen-lang-client-0432163259 \
  --set-env-vars "SUPABASE_DB_URL=${SUPABASE_DB_URL},KAIROS_API_URL=https://kairos-agent-148665080566.us-central1.run.app,CORS_ALLOWED_ORIGINS=https://workout-tracker-69b08.web.app,GIN_MODE=release"
```

**Verification:**
```bash
curl -s https://workout-api-148665080566.us-central1.run.app/api/v1/health
# Expected: {"status":"healthy",...}
```

Build + deploy takes ~3-5 minutes.

### 2. React Frontend (Firebase Hosting)

Firebase CLI requires interactive authentication. If `firebase login` hasn't been run in this terminal, the user needs to do it manually.

```bash
cd workout-tracker

# Build with production API URL
VITE_API_BASE_URL=https://workout-api-148665080566.us-central1.run.app/api/v1 npm run build

# Deploy (requires firebase auth)
npx firebase-tools use workout-tracker-69b08
npx firebase-tools deploy --only hosting
```

If Firebase CLI is not authenticated, tell the user to run:
```bash
npx firebase-tools login
```

**Verification:**
```bash
curl -s https://workout-tracker-69b08.web.app | head -3
# Expected: <!DOCTYPE html>
```

### 3. Kairos Agent (Cloud Run)

Use `--update-env-vars` (not `--set-env-vars`) to ADD new vars without deleting existing ones (GOOGLE_API_KEY, SUPABASE_URL, WHATSAPP_TOKEN, etc. are already configured).

```bash
cd langgraph-skeleton

gcloud run deploy kairos-agent \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --project gen-lang-client-0432163259 \
  --update-env-vars "FRONTEND_URL=https://workout-tracker-69b08.web.app"
```

Only use `--set-env-vars` if you need to replace ALL env vars. The current env vars are:
- `GOOGLE_API_KEY`, `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_DB_URL`
- `WHATSAPP_TOKEN`, `WHATSAPP_VERIFY_TOKEN`, `WHATSAPP_PHONE_NUMBER_ID`
- `GMAIL_APP_PASSWORD`, `GMAIL_USER`
- `FRONTEND_URL`

**Verification:**
```bash
curl -s https://kairos-agent-148665080566.us-central1.run.app/ | python3 -m json.tool | head -5
# Expected: {"status": "ok", "project": "LangGraph Skeleton..."}
```

Build + deploy takes ~5-7 minutes.

## Parallel Deployment

When deploying all three, deploy backend and Kairos in parallel (both use `gcloud run deploy`), then frontend last (depends on backend URL being live for the build).

Run backend and Kairos as background tasks:
```bash
# In parallel
gcloud run deploy workout-api --source workout-tracker-back/ ... &
gcloud run deploy kairos-agent --source langgraph-skeleton/ ... &

# Then frontend after both complete
cd workout-tracker && VITE_API_BASE_URL=... npm run build && firebase deploy --only hosting
```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| `PORT env var reserved` | Included `PORT=8080` in `--set-env-vars` | Remove PORT — Cloud Run sets it automatically |
| `container failed to start` | Missing `SUPABASE_DB_URL` | `--set-env-vars` replaces ALL vars — include SUPABASE_DB_URL |
| `firebase login` fails in non-interactive | Claude Code terminal is non-interactive | User must run `npx firebase-tools login` manually in their terminal |
| CORS errors after deploy | `CORS_ALLOWED_ORIGINS` doesn't include Firebase URL | Set to `https://workout-tracker-69b08.web.app` |
| Frontend shows `localhost` API calls | Built without `VITE_API_BASE_URL` | Rebuild with `VITE_API_BASE_URL=https://workout-api-148665080566.us-central1.run.app/api/v1` |

## Viewing Logs

```bash
# Backend logs
gcloud run logs read workout-api --region us-central1 --limit 50

# Kairos logs
gcloud run logs read kairos-agent --region us-central1 --limit 50

# Current env vars (to verify)
gcloud run services describe workout-api --region us-central1 --format="yaml(spec.template.spec.containers[0].env)"
gcloud run services describe kairos-agent --region us-central1 --format="yaml(spec.template.spec.containers[0].env)"
```
