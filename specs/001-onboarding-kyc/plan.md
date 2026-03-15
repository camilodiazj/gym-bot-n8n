# Implementation Plan: Onboarding KYC

**Branch**: `001-onboarding-kyc` | **Date**: 2026-03-15 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-onboarding-kyc/spec.md`

## Summary

Implement the F-01 Onboarding KYC as a LangGraph StateGraph with 5 conversational
turns that collect 10 user data points. The graph uses Gemini as the KYC agent,
persists partial state via checkpointers (7-day expiry), classifies health conditions
into codes A-E, and supports mid-flow correction and profile summary confirmation.
Exposed via FastAPI endpoint for Postman testing; later integrated with WhatsApp.

## Technical Context

**Language/Version**: Python 3.11+
**Primary Dependencies**: LangGraph >=0.4, langchain-google-genai, FastAPI, httpx, pydantic
**Storage**: Supabase (PostgreSQL) via PostgREST API (httpx)
**Testing**: pytest + pytest-asyncio (asyncio_mode = "auto")
**Target Platform**: Linux/macOS server (local dev), Google Cloud Run (production)
**Project Type**: AI agent service (LangGraph graphs exposed via FastAPI)
**Performance Goals**: KYC session completes in <8 minutes; individual message response <5 seconds
**Constraints**: WhatsApp message limit ~4096 chars; Gemini 2.0-flash rate limits
**Scale/Scope**: ~100 concurrent users initially; 10 KYC fields across 5 turns

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status |
|-----------|------|--------|
| I. Agent-First | KYC MUST be a StateGraph with specialized nodes | PASS — graph design below |
| II. Conversational UX | Spanish, one question at a time, progress indicators | PASS — 5-turn design in FR-004 |
| III. Data-Driven | Profile persisted to Supabase, health codes enforced | PASS — Supabase tools planned |
| IV. Memory & Context | Checkpointer for multi-turn, partial state saved | PASS — InMemorySaver + Supabase persistence |
| V. Proactive Intelligence | Inactivity nudge after 30 min | PASS — nudge node in graph |
| VI. Safety First | Health codes A-E classified, code E blocks routine gen | PASS — health_classifier node |
| VII. Progressive Complexity | Start with isolated case, mock → live | PASS — case5_onboarding_kyc structure |

All gates pass. No violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/001-onboarding-kyc/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (FastAPI endpoints)
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
langgraph-skeleton/
├── cases/
│   └── case5_onboarding_kyc/
│       ├── __init__.py
│       ├── graph.py              # KYC graph (mock/in-memory)
│       ├── graph_live.py         # KYC graph (Supabase-connected)
│       ├── state.py              # KYCState TypedDict
│       ├── nodes.py              # Node functions (kyc_agent, health_classifier, etc.)
│       ├── prompts.py            # Spanish system prompts
│       ├── tools_supabase.py     # Supabase tools (user lookup, profile save)
│       └── run.py                # Standalone runner for testing
├── src/
│   └── shared/
│       ├── supabase_client.py    # Existing — reuse for queries
│       └── llm.py                # Existing — Gemini singleton
├── server.py                     # Add /case5/kyc/chat + /case5/kyc/history
└── tests/
    └── test_case5.py             # pytest tests for KYC graph
```

**Structure Decision**: Single project extending the existing `cases/` pattern
(cases 1-4 already established). New case5 follows the same convention with
added modularity (separate state, nodes, prompts files) due to graph complexity.

## Graph Architecture

### KYC StateGraph Design

```text
                    ┌─────────────┐
                    │    START    │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │ check_user  │ ← Supabase: user exists?
                    └──────┬──────┘
                     ┌─────┴─────┐
                  existing    new_user
                     │           │
               ┌─────▼────┐ ┌───▼──────────┐
               │  END      │ │  kyc_agent   │ ← Gemini: ask next question
               │(redirect) │ └───┬──────────┘
               └──────────┘     │
                          ┌─────▼──────┐
                          │check_status│ ← Is KYC complete?
                          └─────┬──────┘
                     ┌──────────┼──────────┐
                  continue   complete   correction
                     │           │           │
                     ▼       ┌───▼────────┐  │
                    END      │confirm_     │  │
                  (wait)     │profile      │  │
                             └───┬────────┘  │
                          ┌──────┴─────┐     │
                       accepted   rejected   │
                          │           │      │
                    ┌─────▼────┐      └──────┘
                    │ health_  │      (back to kyc_agent)
                    │classifier│
                    └─────┬────┘
                    ┌─────┴──────┐
                 safe(A-D)    severe(E)
                    │            │
              ┌─────▼────┐ ┌────▼──────┐
              │save_      │ │route_to_  │
              │profile    │ │trainer    │
              └─────┬────┘ └────┬──────┘
                    │           │
                   END         END
```

### State Schema

```python
class KYCState(TypedDict):
    messages: Annotated[list[BaseMessage], operator.add]
    # User identity
    phone_number: str
    display_name: str
    is_new_user: bool
    # KYC progress
    current_turn: int                    # 1-5
    collected_data: dict                 # {field_name: value}
    is_complete: bool
    # Health classification
    health_code: str                     # A, B, C, D, E
    affected_zones: list[str]
    # Flow control
    awaiting_confirmation: bool
    profile_confirmed: bool
    needs_correction: bool
    correction_field: str
    # Output
    response: str
    route_to_trainer: bool
```

### Node Responsibilities

| Node | Responsibility | Constitution Principle |
|------|---------------|----------------------|
| `check_user` | Query Supabase for existing user by phone | III. Data-Driven |
| `kyc_agent` | Gemini conducts conversational KYC, asks one turn at a time | I. Agent-First, II. Conversational UX |
| `check_status` | Determine if KYC is complete, correction requested, or continue | I. Agent-First |
| `confirm_profile` | Present summary, handle accept/reject | II. Conversational UX |
| `health_classifier` | Map free-text health description to code A-E | VI. Safety First |
| `save_profile` | Persist completed profile to Supabase | III. Data-Driven, IV. Memory |
| `route_to_trainer` | Health code E → recommend human trainer | VI. Safety First |

### Turn-to-Fields Mapping

| Turn | Fields | System Prompt Focus |
|------|--------|-------------------|
| 1 | `primary_goal` | "¿Cuál es tu objetivo principal?" |
| 2 | `training_experience`, `days_available`, `preferred_schedule` | "¿Cuánta experiencia tienes? ¿Cuántos días puedes entrenar?" |
| 3 | `training_environment`, `home_equipment` (if HOME) | "¿Dónde entrenas? ¿Qué equipo tienes?" |
| 4 | `biological_sex`, `age`, `height_cm`, `weight_kg` | "Datos básicos para personalizar tu plan" |
| 5 | `health_status` (free-text → classified) | "¿Tienes alguna lesión o condición de salud?" |

## Complexity Tracking

> No violations to justify. All gates pass.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
