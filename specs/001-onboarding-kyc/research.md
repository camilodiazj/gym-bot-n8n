# Phase 0 Research: Onboarding KYC

**Feature**: 001-onboarding-kyc | **Date**: 2026-03-15

## Summary

No NEEDS CLARIFICATION items in Technical Context. This document records
key decisions made during planning, with rationale and alternatives.

---

## R-01: Graph Orchestration Framework

**Decision**: LangGraph `StateGraph` with `TypedDict` state

**Rationale**: Cases 1-4 already establish the `StateGraph` pattern in the skeleton.
Constitution Principle I (Agent-First) mandates all features as StateGraphs. LangGraph
provides conditional edges (for health routing), checkpointers (for resumption), and
ToolNode (for Supabase queries) — all required by this feature.

**Alternatives considered**:
- Plain LangChain chains: No conditional routing or checkpointing built-in.
- Custom Python state machine: No LLM tool-calling integration; violates Principle I.

---

## R-02: KYC Agent LLM

**Decision**: Gemini 2.0 Flash via `langchain-google-genai`

**Rationale**: Already configured in `src/shared/llm.py` as project default. Supports
tool binding for Supabase queries. Fast response times (<2s typical) meet the <5s
per-message requirement (FR-014). Spanish language output confirmed in Cases 1-4.

**Alternatives considered**:
- OpenAI GPT-5.x: Higher cost per token, no advantage for conversational KYC.
- Local model: Latency and hosting complexity not justified for <100 users.

---

## R-03: Checkpointing Strategy

**Decision**: `InMemorySaver` for development; Supabase-backed persistence for production.

**Rationale**: Case 4 already uses `InMemorySaver` with `thread_id` isolation.
For FR-007 (partial KYC persistence) and FR-019 (7-day expiry), production needs
durable storage. Two approaches for production:

1. **PostgresCheckpointSaver** (LangGraph built-in): Requires `langgraph-checkpoint-postgres`.
   Direct Postgres connection. Supports TTL-based cleanup.
2. **Custom Supabase table**: Store serialized state in a `kyc_sessions` table via
   PostgREST. Aligns with existing `supabase_client.py` pattern.

**Chosen**: Option 1 for production (native LangGraph support, automatic state
serialization). Option 2 as fallback if PostgREST-only access is required.

For the mock graph (`graph.py`), `InMemorySaver` is sufficient.

---

## R-04: Health Classification Approach

**Decision**: Gemini-based classification within a dedicated `health_classifier` node.

**Rationale**: FR-011 requires classifying free-text health descriptions into codes A-E.
Keyword matching alone is brittle ("me duele la rodilla" vs "tengo problemas en las
rodillas" vs "lesión de menisco"). Gemini can understand Spanish medical context and
map to the correct code with high accuracy (SC-005: 95% target).

**Implementation**: The `health_classifier` node receives the free-text `health_status`
from Turn 5, invokes Gemini with a structured prompt listing code definitions:
- A: Sin restricciones
- B: Problemas en tren inferior (rodilla, tobillo, cadera)
- C: Problemas en tren superior (hombro, codo, muñeca)
- D: Problemas de columna (espalda baja, cervical, hernia)
- E: Condición severa (cardíaca, respiratoria, neurológica, múltiples zonas)

Gemini responds with structured JSON: `{"code": "B", "zones": ["rodilla"]}`.

**Alternatives considered**:
- Rule-based keyword matching: Too brittle for free-text Spanish input.
- Separate classification model: Over-engineered for this volume.

---

## R-05: Supabase Integration Pattern

**Decision**: PostgREST API via `httpx` (reuse `supabase_client.py`)

**Rationale**: The existing `supabase_query()` and `supabase_insert()` functions in
`src/shared/supabase_client.py` provide async HTTP access to all Supabase tables.
Case 3 Live and Case 4 Live already use this pattern for tools. No additional SDK needed.

New tools for Case 5:
- `lookup_user_by_phone(phone)`: Query `users` table by `full_phone_number`.
- `save_gym_profile(profile_data)`: Insert into `users_gym_profile`.
- `save_user(user_data)`: Insert into `users` table.

These follow the same `@tool` + `supabase_query/insert` pattern as Case 3.

**Alternatives considered**:
- `supabase-py` SDK: Extra dependency, no advantage over direct PostgREST.
- Direct Postgres connection: Requires connection pooling; PostgREST handles this.

---

## R-06: Inactivity Nudge (FR-008)

**Decision**: External scheduler, NOT in-graph timer.

**Rationale**: LangGraph graphs are invoked per-message. There is no built-in way to
trigger a node 30 minutes after the last message within the graph itself. The nudge
requires an external mechanism:

1. **Option A**: Background task in FastAPI (`asyncio.create_task`) that checks
   session timestamps periodically.
2. **Option B**: Supabase scheduled function / cron job that queries stale sessions.
3. **Option C**: Separate n8n workflow that monitors `kyc_sessions` table.

**Chosen**: Option A for MVP (simplest, self-contained). The graph stores
`last_interaction_at` in state. A FastAPI background task runs every 5 minutes,
finds sessions where `now - last_interaction_at > 30min` and `nudge_sent = false`,
then sends the nudge message.

For production (WhatsApp integration), Option C is preferred (n8n already handles
scheduled messaging).

---

## R-07: Profile Summary and Correction Flow

**Decision**: `confirm_profile` node with structured state flags.

**Rationale**: FR-016 requires presenting a profile summary for confirmation.
FR-018 requires targeted correction (ask which field, update only that field,
re-present summary). This maps to state flags:

- `awaiting_confirmation: bool` — summary has been shown, waiting for response
- `profile_confirmed: bool` — user accepted the summary
- `needs_correction: bool` — user rejected and wants to fix a field
- `correction_field: str` — which field to correct

The `check_status` conditional edge routes:
- If `awaiting_confirmation` and user says "sí" → `health_classifier`
- If `awaiting_confirmation` and user rejects → set `needs_correction`, route to `kyc_agent`
- `kyc_agent` detects `needs_correction` flag and asks which field to fix

This avoids a separate "correction" node — `kyc_agent` handles both normal
collection and corrections based on state.

---

## R-08: Turn Progress Tracking

**Decision**: `current_turn: int` in state, updated by `check_status` node.

**Rationale**: FR-005 requires progress indicators ("Pregunta 2 de 5"). The
`check_status` node inspects `collected_data` to determine which turn the user
is on. The `kyc_agent` system prompt includes `{current_turn}/5` in its response.

Turn detection logic:
- Turn 1 complete when: `primary_goal` is set
- Turn 2 complete when: `training_experience`, `days_available`, `preferred_schedule` are set
- Turn 3 complete when: `training_environment` is set (+ `home_equipment` if HOME)
- Turn 4 complete when: `biological_sex`, `age`, `height_cm`, `weight_kg` are set
- Turn 5 complete when: `health_status` is set

---

## Dependencies Best Practices

### LangGraph >=0.4
- Use `StateGraph` (not `MessageGraph`) for typed state.
- `Annotated[list[BaseMessage], operator.add]` for message accumulation.
- `InMemorySaver` for dev checkpointing; `PostgresSaver` for production.
- Conditional edges via `add_conditional_edges(source, router_fn, destinations)`.

### Gemini 2.0 Flash
- Temperature 0.3-0.5 for classification tasks (health codes).
- Temperature 0.7 for conversational KYC (natural tone).
- Max ~30 requests/minute on free tier; sufficient for <100 concurrent users.
- Structured output via prompt engineering (JSON response format).

### Supabase PostgREST
- Filter syntax: `{"column": "eq.value"}` for equality.
- Insert: POST to `/rest/v1/{table}` with JSON body + `Prefer: return=representation`.
- Upsert: Add `Prefer: resolution=merge-duplicates` header.
- The `supabase_client.py` already handles headers and auth.
