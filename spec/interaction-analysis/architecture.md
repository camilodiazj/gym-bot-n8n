# KAN-96: Interaction Analysis — Architecture

## Problem
GymBot accumulates conversations in `n8n_chat_histories` (2,290+ msgs, 167+ sessions) with zero visibility into usage patterns, friction points, or user needs.

## Solution
A single n8n workflow that runs weekly (+ manual), queries metrics from Supabase, samples conversations for LLM analysis, and emails a structured report.

## Data Source

### `n8n_chat_histories` (existing table)
| Column | Type | Notes |
|--------|------|-------|
| `id` | integer (PK, auto-increment) | Chronological ordering |
| `session_id` | varchar | Format: `{phone}_kyc_v4` or `{user_uuid}_{n}_{type}` |
| `message` | jsonb | `{ type: "human"|"ai"|"tool", content: "...", name?: "..." }` |
| `created_at` | timestamptz | **NEW** — `DEFAULT NOW()`, enables time-window queries |

### Session Types (extracted from `session_id` suffix)
| Suffix | Type | Volume |
|--------|------|--------|
| `_kyc_v4` | Onboarding | 46 sessions, 924 msgs |
| `_chat` | Fitness Q&A | 53 sessions, 814 msgs |
| `_confirmation` | Workout confirmation | 35 sessions, 268 msgs |
| `_renewal` | Mesocycle renewal | 2 sessions, 8 msgs |
| other | Misc | 31 sessions, 276 msgs |

### Message JSONB Schema
- **Content path**: `message->>'content'` (direct, NOT nested under `data`)
- **Type path**: `message->>'type'` → `human` | `ai` | `tool`
- **Tool name**: `message->>'name'` (only for `type = 'tool'`)

## Workflow Architecture (10 nodes)

```
schedule_trigger (Mon 8AM) ──┐
manual_trigger ──────────────┤
                             ├─► query_metrics (Postgres)
                             ├─► query_conversation_samples (Postgres)
                             │
                             ├─► build_analysis_payload (Code)
                             ├─► llm_analysis (Agent + gpt-4.1-mini)
                             ├─► format_and_send_email (Code + Gmail)
```

### Node Inventory

| # | Node | Type | Purpose |
|---|------|------|---------|
| 1 | `schedule_trigger` | scheduleTrigger v1.3 | Monday 8 AM Bogota |
| 2 | `manual_trigger` | manualTrigger v1 | On-demand execution |
| 3 | `query_metrics` | postgres v2.6 | Single consolidated query: all KPIs |
| 4 | `query_conversation_samples` | postgres v2.6 | Stratified sample for LLM |
| 5 | `build_analysis_payload` | code v2 | Assemble metrics + conversations |
| 6 | `llm_analysis` | agent v3.1 | Qualitative analysis in Spanish |
| 7 | `openai_model` | lmChatOpenAi v1 | gpt-4.1-mini |
| 8 | `format_email` | code v2 | Markdown → HTML email |
| 9 | `send_email` | gmail v2.1 | Deliver report |

### Credentials
| Service | Credential ID | Name |
|---------|--------------|------|
| Postgres (Supabase) | `vZLJtIWG5nYXMez4` | Supabase |
| OpenAI | `mwhZD1w0asX5zwKw` | OpenAI (gpt-4.1-mini) |
| Gmail | `H1GF1YdmcaZ0gfWB` | Gmail Kai.Ros |

### Data Flow
1. Trigger fires → 2 Postgres queries execute in parallel
2. `build_analysis_payload` reads both via `$('node_name').all()` and assembles:
   - Quantitative metrics JSON
   - Grouped conversation samples (max 8 sessions, 30 msgs each, human+ai only)
3. `llm_analysis` receives payload, generates structured Spanish report (7 sections)
4. `format_email` wraps LLM markdown output in HTML email template
5. `send_email` delivers to `camilodiazjaimes@gmail.com`

### LLM Token Budget
| Source | Estimate |
|--------|----------|
| Conversation samples (8 sessions x 30 msgs x ~100 tokens) | ~24K tokens |
| Metrics payload | ~2K tokens |
| System prompt | ~800 tokens |
| **Total input** | **~27K tokens** (within 128K limit) |

## SQL Queries

### Query 1: `query_metrics` (consolidated)

All quantitative metrics in a single query using CTEs:
- Total sessions/messages (all-time + last 7 days)
- Session breakdown by type (count, avg msgs, max msgs)
- Tool usage frequency
- KYC completion rate (cross-join with `users` table)
- Top 5 longest sessions

### Query 2: `query_conversation_samples` (stratified)

Sampling strategy:
- 3 most recent KYC sessions
- 3 most recent chat sessions
- 2 longest sessions (any type) — friction signals
- Filter: `human` + `ai` only (skip `tool` — verbose JSON, low signal)
- Cap: 30 messages per session, content truncated via `LEFT(content, 500)`

## LLM Prompt Structure (Spanish)

7 analysis sections:
1. Resumen Ejecutivo (3-4 sentences)
2. Patrones de Uso
3. Temas Frecuentes en Chat (top 3-5 specific topics)
4. Puntos de Friccion (long sessions, confusion patterns, KYC sticking points)
5. Analisis de KYC (completion rate, drop-off moment)
6. Uso de Herramientas
7. Recomendaciones de Mejora (Top 3 actionable)

## Output: Email Report

- **To**: camilodiazjaimes@gmail.com
- **Subject**: "GymBot Analisis Semanal - {DD/MM/YYYY}"
- **Body**: HTML with dark header, metrics summary table, LLM analysis rendered from markdown
- **Schedule**: Every Monday 8 AM America/Bogota
