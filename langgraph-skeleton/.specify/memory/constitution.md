<!--
  Sync Impact Report
  ==================
  Version change: 0.0.0 (template) → 1.0.0
  Modified principles:
    - [PRINCIPLE_1_NAME] → I. Agent-First Architecture
    - [PRINCIPLE_2_NAME] → II. Conversational UX
    - [PRINCIPLE_3_NAME] → III. Data-Driven Personalization
    - [PRINCIPLE_4_NAME] → IV. Memory & Context Persistence
    - [PRINCIPLE_5_NAME] → V. Proactive Intelligence
  Added principles:
    - VI. Safety First
    - VII. Progressive Complexity
  Added sections:
    - Technology Stack (replaces [SECTION_2_NAME])
    - Development Workflow (replaces [SECTION_3_NAME])
  Removed sections: None (all template slots filled)
  Templates requiring updates:
    - .specify/templates/plan-template.md ✅ no update needed (generic gates)
    - .specify/templates/spec-template.md ✅ no update needed (generic structure)
    - .specify/templates/tasks-template.md ✅ no update needed (generic phases)
    - .specify/templates/checklist-template.md ✅ no update needed (generic structure)
    - .specify/templates/agent-file-template.md ✅ no update needed (generic structure)
  Follow-up TODOs: None
-->

# Kairos Personal Trainer Constitution

## Core Principles

### I. Agent-First Architecture

Every feature MUST be modeled as a LangGraph `StateGraph` with specialized nodes.
Each node has a single responsibility (KYC collection, intention detection,
exercise selection, message formatting). Nodes communicate exclusively through
typed state (`TypedDict` with `Annotated` reducers). Conditional edges route
between nodes based on state; no business logic lives outside the graph.

- All agent graphs MUST be compilable and invocable via `graph.ainvoke()`.
- Tools MUST use the `@tool` decorator and be bound via `model.bind_tools()`.
- Agent loops (tool-calling cycles) MUST use `ToolNode` from `langgraph.prebuilt`.
- Subgraphs MUST be composable and independently testable.

### II. Conversational UX

All user-facing interactions MUST feel like a natural conversation in Spanish
(Colombian dialect), never like a form or survey. Kairos addresses users by
name, uses motivational language, and adapts tone to context.

- System prompts MUST be written in Spanish and reflect Colombian conversational tone.
- The KYC flow MUST ask one question at a time, adapting to prior answers.
- If a user provides multiple data points in one message, the agent MUST register all of them.
- Kairos MUST NOT repeat questions for data already collected in the session.
- Progress indicators (e.g., "Pregunta 2 de 5") MUST be included during onboarding.
- WhatsApp message length MUST stay concise (2-3 sentences per response during KYC).

### III. Data-Driven Personalization

Every agent decision MUST be grounded in real user data — profile, training
history, preferences, and feedback. No generic or hardcoded routines.

- Workout generation MUST query the Supabase `exercises` table (1,657 real exercises).
- Set/rep parameters MUST come from the `set_profiles` table, not LLM invention.
- Exercise exclusion MUST respect `health_status` codes (A-E) from the user profile.
- Priority muscles and disliked exercises MUST influence exercise selection.
- The agent MUST use tools (`get_exercises_by_pattern`, `get_set_profile`,
  `search_exercises`) for all exercise data — never fabricate exercise names.

### IV. Memory & Context Persistence

Kairos MUST remember everything across sessions and conversations. No user
should ever need to repeat themselves.

- Multi-turn conversations MUST use LangGraph checkpointers (`InMemorySaver`
  for development, persistent store for production).
- Each conversation MUST be isolated via `thread_id` in the checkpointer config.
- User profile data MUST be persisted in Supabase (`users`, `users_gym_profile`,
  `users_plans`, `workouts` tables).
- Partial KYC state MUST be saved so users can resume onboarding from where
  they left off, even after 30+ minutes of inactivity.
- Exercise preference history (replacements, favorites, avoided) MUST be tracked
  and influence future routine generation.

### V. Proactive Intelligence

Kairos MUST NOT be purely reactive. The agent initiates contact at the right
moments — morning reminders, workout completion follow-ups, weekly scheduling,
milestone celebrations, and re-engagement after inactivity.

- Morning reminders MUST include the user's name and the day's muscle focus.
- Completion tracking MUST trigger follow-up if a scheduled session is not marked done.
- Weekly scheduling MUST categorize users (celebration / growth / re-engagement)
  based on session completion rates.
- Inactivity detection MUST trigger re-engagement after 72h of silence.
- Milestone celebrations MUST fire on first-week completion, streak achievements,
  and mesocycle completions.
- Proactive messages MUST vary weekly to avoid repetitiveness.

### VI. Safety First

When a user reports pain, injury, or any health concern, Kairos MUST prioritize
safety above all other objectives. The agent never pushes through pain.

- If a user reports acute pain, Kairos MUST immediately stop exercise
  recommendations and suggest consulting a medical professional.
- If a user reports mild discomfort, Kairos MUST offer alternative exercises
  that do not load the affected zone.
- Health status codes MUST be enforced at workout generation time:
  B → avoid high-impact lower body; C → avoid overhead pressing;
  D → avoid heavy axial loading; E → prioritize machines and low-risk exercises.
- The affected zone MUST be recorded in the user profile and excluded from
  future routine generation until explicitly cleared.
- Users flagged with serious conditions SHOULD be routed to a human trainer
  (F-07: Kairos como Aliado de Entrenadores Profesionales).

### VII. Progressive Complexity

Implementation MUST follow an incremental path: skeleton → MVP → production
features. Each increment MUST be independently deployable and testable.

- New features MUST start as isolated LangGraph cases before integration.
- Mock data MUST be used first, then replaced with Supabase live data.
- Each feature MUST have at least one pytest test that validates the graph compiles
  and produces expected output structure.
- FastAPI endpoints MUST be added for every graph to enable Postman/integration testing.
- The implementation order follows feature priority: F-01 (Onboarding) → F-02
  (Routine Generation) → F-03 (Flexibility) → F-06 (Proactive) → F-04
  (Emotional Intelligence) → F-05 (Progression).

## Technology Stack

| Layer | Technology | Constraint |
|-------|-----------|------------|
| Agent orchestration | LangGraph (Python) | All graphs use `StateGraph` + `TypedDict` state |
| LLMs | Google Gemini 2.0-flash, OpenAI GPT-5.x | Via `langchain-google-genai` / `langchain-openai` |
| Database | Supabase (PostgreSQL) | Accessed via PostgREST API (`httpx`) or direct connection |
| Messaging | WhatsApp Business API | All user-facing content in Spanish |
| API layer | FastAPI + Uvicorn | Async endpoints, Pydantic models for validation |
| Frontend | React 19 + TypeScript + Vite | Workout Tracker web app (Firebase Hosting) |
| Backend API | Go + Gin (hexagonal architecture) | Google Cloud Run deployment |
| Testing | pytest + pytest-asyncio | `asyncio_mode = "auto"` in pyproject.toml |
| Package management | uv + pyproject.toml | Virtual environment via `.venv/` |

- All Python code MUST target Python 3.11+.
- Dependencies MUST be declared in `pyproject.toml` with optional `[dev]` extras.
- Environment variables MUST be loaded via `python-dotenv` from `.env` files.
- Secrets (API keys, Supabase credentials) MUST NOT be committed to version control.

## Development Workflow

### Feature Implementation Process

1. **Specify**: Define the feature spec using Gherkin scenarios from `Kairos_Gherkin_v2.md`.
2. **Plan**: Create implementation plan with graph design, node responsibilities, and state schema.
3. **Build incrementally**: Start with isolated `cases/` graph → add mock tools → replace with Supabase tools → add FastAPI endpoint.
4. **Test**: Write pytest cases that validate graph compilation, node routing, and output structure.
5. **Integrate**: Connect to existing flows and verify end-to-end via Postman or E2E test runner.

### Code Conventions

- Graph files: `graph.py` (mock), `graph_live.py` (Supabase-connected).
- Tool files: `tools.py` (mock), `tools_supabase.py` (real).
- State classes: `TypedDict` with `Annotated[list, operator.add]` for message reducers.
- System prompts: Always Spanish, defined as module-level constants.
- Node functions: `async def` with explicit return dict matching state keys.
- Naming: `snake_case` for functions and files, `PascalCase` for state classes.

### Quality Gates

- Every graph MUST compile without errors (`workflow.compile()`).
- Every graph MUST be invocable via `graph.ainvoke()` with a valid initial state.
- Every FastAPI endpoint MUST return structured JSON with case identifier.
- LLM responses MUST be in Spanish.
- Tool calls MUST reference real Supabase tables when in "live" mode.

## Governance

This constitution is the authoritative source for architectural decisions and
development standards in the Kairos LangGraph migration. All code, specs, plans,
and tasks MUST comply with these principles.

- **Amendments** require updating this file, incrementing the version, and
  verifying consistency with all templates in `.specify/templates/`.
- **Principle violations** MUST be documented in a Complexity Tracking table
  (see plan template) with explicit justification.
- **New features** MUST reference the Gherkin spec (`Kairos_Gherkin_v2.md`)
  for acceptance scenarios.
- **Constitution version** follows semantic versioning: MAJOR for principle
  removals/redefinitions, MINOR for additions, PATCH for clarifications.

**Version**: 1.0.0 | **Ratified**: 2026-03-15 | **Last Amended**: 2026-03-15
