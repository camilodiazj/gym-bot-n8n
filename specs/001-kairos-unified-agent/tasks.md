# Tasks: Agente Unificado Kairos

**Input**: Design documents from `/specs/001-kairos-unified-agent/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/api.md

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create `case6_unified_agent/` module structure and extend shared Supabase client

- [x] T001 Create case6 module directory and `__init__.py` in `langgraph-skeleton/cases/case6_unified_agent/__init__.py`
- [x] T002 Implement `supabase_update()` (PATCH with filters) in `langgraph-skeleton/src/shared/supabase_client.py`
- [x] T003 Implement `supabase_bulk_insert()` (POST with list body) in `langgraph-skeleton/src/shared/supabase_client.py`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: State schema, context loader, system prompt, and graph shell — MUST be complete before any user story

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Define `UserContext`, `DraftExercise`, `DraftDay`, `DraftRoutine`, and `UnifiedAgentState` TypedDicts in `langgraph-skeleton/cases/case6_unified_agent/state.py` per data-model.md
- [x] T005 [P] Implement `load_user_context(phone_number)` with parallel Supabase queries (`asyncio.gather`) in `langgraph-skeleton/cases/case6_unified_agent/context_loader.py` — queries: users, users_gym_profile, users_plans, user_weekly_schedule (today + missed last 3 days + next future), pending_tasks; compute flags: is_new_user, kyc_complete, has_schedule, all_w4_completed
- [x] T006 [P] Create `KAIROS_SYSTEM_PROMPT` template and `format_user_context(ctx: UserContext) -> str` in `langgraph-skeleton/cases/case6_unified_agent/prompts.py` — prompt must include: user name, plan info, today's sessions, missed sessions, next session, pending tasks, tool descriptions, behavior rules (pending task first, brevity, never invent data, priority rule: if today's session exists show it first — only offer missed sessions when NO session today)
- [x] T007 Implement graph nodes: `load_context` (calls context_loader), `router` (new_user→KYC vs existing→agent), `kairos_agent` (Gemini with system prompt, no tools yet), `should_continue` routing function in `langgraph-skeleton/cases/case6_unified_agent/nodes.py`
- [x] T008 Build mock graph (no KYC, no tools — agent responds with context only) using `StateGraph` + `InMemorySaver` in `langgraph-skeleton/cases/case6_unified_agent/graph.py`
- [x] T009 Add `POST /case6/chat` and `GET /case6/history` endpoints to `langgraph-skeleton/server.py` — import `build_case6_graph`, use `thread_id = f"case6_{phone_number}"` (ensures FR-011 thread isolation), return `{response, thread_id, is_new_user, kyc_complete}` per contracts/api.md
- [x] T009b Add WhatsApp status message filtering in `POST /case6/chat` endpoint — return early with empty response if message is empty or indicates a status update (FR-013) in `langgraph-skeleton/server.py`

**Checkpoint**: Server starts, filters WhatsApp noise, agent responds to messages with contextualized responses (name, today's session, goal). No tools, no KYC routing.

---

## Phase 3: User Story 1 — Ver y Confirmar Rutina (Priority: P1) MVP

**Goal**: Existing user can view today's routine, confirm completion, get tracker link, and handle missed sessions — all via conversational agent with tools.

**Independent Test**: Send message as user 570000000003 (Test_WithRoutine fixture) asking "que me toca hoy?", verify routine displayed. Then send "ya terminé", verify session marked completed.

### Implementation for User Story 1

- [x] T010 [P] [US1] Implement `get_todays_routine(user_id, session_name, week)` tool — SELECT workouts JOIN exercises WHERE user_id AND week AND day_name ORDER BY exercise_order — return formatted exercise list with sets, reps, RIR, rest, video link in `langgraph-skeleton/cases/case6_unified_agent/tools.py`
- [x] T011 [P] [US1] Implement `confirm_workout_completion(user_id, session_date?)` tool — UPDATE user_weekly_schedule SET Completed=true (today or yesterday grace period) + UPDATE pending_tasks SET status='completed' WHERE CONFIRMAR_RUTINA in `langgraph-skeleton/cases/case6_unified_agent/tools.py`
- [x] T012 [P] [US1] Implement `decline_workout(user_id)` tool — UPDATE pending_tasks SET status='declined' WHERE status='pending' in `langgraph-skeleton/cases/case6_unified_agent/tools.py`
- [x] T013 [P] [US1] Implement `create_magic_link(user_id)` tool — INSERT magic_links with 48h expiry, generate short hex code, return full Workout Tracker URL in `langgraph-skeleton/cases/case6_unified_agent/tools.py`
- [x] T014 [US1] Update `kairos_agent` node in `langgraph-skeleton/cases/case6_unified_agent/nodes.py` to bind US1 tools (`get_todays_routine`, `confirm_workout_completion`, `decline_workout`, `create_magic_link`) via `llm.bind_tools()` and add `ToolNode` + `should_continue` ReAct loop
- [x] T015 [US1] Build `graph_live.py` with Supabase context loader + US1 tools + ToolNode loop (no KYC subgraph yet) in `langgraph-skeleton/cases/case6_unified_agent/graph_live.py`
- [x] T016 [US1] Update `server.py` to use `graph_live` when `SUPABASE_URL` env var is present (same pattern as case5 live/mock switching) in `langgraph-skeleton/server.py`

**Checkpoint**: Agent can view routine, confirm workout, create magic link, handle missed sessions, resolve pending tasks — all via natural conversation.

---

## Phase 4: User Story 2 — Onboarding KYC (Priority: P1)

**Goal**: New user is routed to KYC subgraph, completes 5-turn onboarding, profile saved to Supabase. Next message routes to agent mode.

**Independent Test**: Send message as unknown phone 579000000001 saying "hola", verify KYC starts. Complete all turns. Send another message — verify agent mode (no KYC repeat).

### Implementation for User Story 2

- [x] T017 [US2] Implement KYC state mapping functions: `unified_to_kyc_state(state: UnifiedAgentState) -> dict` and `kyc_to_unified_state(kyc_state: KYCState) -> dict` in `langgraph-skeleton/cases/case6_unified_agent/nodes.py`
- [x] T018 [US2] Integrate Case 5 KYC as subgraph in `langgraph-skeleton/cases/case6_unified_agent/graph_live.py` — import `build_case5_live_graph`, add as node with state mapping, wire router conditional edge: `is_new_user && !kyc_complete → kyc_subgraph`
- [x] T019 [US2] Handle KYC completion transition: after KYC subgraph ends, next message from same user should go through `load_context` (which now finds user in DB) → `router` → agent mode. Verify in `langgraph-skeleton/cases/case6_unified_agent/graph_live.py`

**Checkpoint**: Full new user journey: first message → KYC 5 turns → profile saved → next message → agent responds with context.

---

## Phase 5: User Story 3 — Creación de Rutina Borrador (Priority: P2)

**Goal**: User with profile but no plan can create a routine interactively — agent asks preference (all-at-once or day-by-day), generates draft, accepts modifications, saves approved version.

**Independent Test**: Start as user with KYC complete but no plan. Ask "quiero mi rutina". Verify agent asks preference, generates draft, allows swap, and saves workout plan to DB.

### Implementation for User Story 3

- [x] T020 [P] [US3] Implement `get_day_requirements(week_schedule)` tool — SELECT routine_templates JOIN template_days JOIN day_requirements, return `[{day_number, title, patterns: [{pattern, min_sets, priority}]}]` in `langgraph-skeleton/cases/case6_unified_agent/tools.py`
- [x] T021 [P] [US3] Implement `get_exercises_for_draft(pattern, level, equipment?, exclude_muscle?, limit=5)` tool — SELECT exercises with filters, return candidate list for LLM selection in `langgraph-skeleton/cases/case6_unified_agent/tools.py`
- [x] T022 [P] [US3] Implement `find_exercise_alternatives(pattern, level, exclude_name?, equipment?)` tool — SELECT exercises for swap scenarios, return alternative list in `langgraph-skeleton/cases/case6_unified_agent/tools.py`
- [x] T023 [US3] Implement `save_workout_plan(user_id, draft_json)` tool — parse DraftRoutine JSON, INSERT users_plans, bulk INSERT workouts (4 weeks × exercises per day) using `supabase_bulk_insert()` in `langgraph-skeleton/cases/case6_unified_agent/tools.py`
- [x] T024 [US3] Add draft mode tools to `kairos_agent` bind_tools list and update system prompt in `langgraph-skeleton/cases/case6_unified_agent/prompts.py` with instructions for draft creation flow: ask user preference (all-at-once vs day-by-day), build draft incrementally using tools, present for approval, accept modifications via `find_exercise_alternatives`
- [x] T025 [US3] Wire draft mode tools into `graph_live.py` ToolNode in `langgraph-skeleton/cases/case6_unified_agent/graph_live.py`

**Checkpoint**: User can create a routine interactively, swap exercises, approve and save — plan + 4 weeks of workouts persisted in Supabase.

---

## Phase 6: User Story 4 — Agendamiento de Sesiones (Priority: P2)

**Goal**: User with plan but no schedule can tell the agent which days they want to train. Agent creates the sessions.

**Independent Test**: Use a user with active plan but empty schedule. Send "quiero programar mis entrenamientos", provide days, verify user_weekly_schedule rows created in DB.

### Implementation for User Story 4

- [x] T026 [P] [US4] Implement `get_schedule_info(user_id)` tool — SELECT users_plans JOIN week_schedules JOIN template_days, return `{days_per_week, sessions: [{day_number, title}], current_week}` in `langgraph-skeleton/cases/case6_unified_agent/tools.py`
- [x] T027 [P] [US4] Implement `schedule_sessions(user_id, sessions_json)` tool — parse JSON `[{week_day, session_name, planned_day}]`, bulk INSERT into user_weekly_schedule using `supabase_bulk_insert()` in `langgraph-skeleton/cases/case6_unified_agent/tools.py`
- [x] T028 [US4] Add scheduling tools to `kairos_agent` bind_tools list and update system prompt with scheduling instructions in `langgraph-skeleton/cases/case6_unified_agent/prompts.py` — agent must validate day count matches plan, map sessions to days in order
- [x] T029 [US4] Wire scheduling tools into `graph_live.py` ToolNode in `langgraph-skeleton/cases/case6_unified_agent/graph_live.py`

**Checkpoint**: Agent can ask for preferred days, validate count, create schedule, confirm to user.

---

## Phase 7: User Story 5 — Tareas Pendientes y Chat Fitness (Priority: P3)

**Goal**: Agent resolves pending tasks before answering any other request. Agent answers fitness questions personalized to user profile without calling tools.

**Independent Test**: Use user 570000000004 (Test_WithPendingTask fixture). Send "hola", verify agent asks about pending task first. Then test fitness question with user that has a plan — verify personalized answer.

### Implementation for User Story 5

- [x] T030 [US5] Update system prompt with explicit pending task priority rule: "Si hay tarea pendiente CONFIRMAR_RUTINA, SIEMPRE pregunta primero si completó la sesión antes de responder cualquier otra cosa" in `langgraph-skeleton/cases/case6_unified_agent/prompts.py`
- [x] T031 [US5] Verify `format_user_context()` clearly surfaces pending tasks section when present — ensure the formatted context makes pending tasks unmissable for the LLM in `langgraph-skeleton/cases/case6_unified_agent/prompts.py`
- [x] T032 [US5] Verify chat fitness flow: ensure agent responds to general fitness questions using only the system prompt context (goal, level, weight from profile) without calling any tools — validate no false tool calls in `langgraph-skeleton/cases/case6_unified_agent/prompts.py`

**Checkpoint**: Pending tasks resolved before any other interaction. Fitness chat works without tool calls.

---

## Phase 8: User Story 6 — Renovación de Mesociclo (Priority: P3)

**Goal**: When user completes all W4 sessions, agent detects and offers renewal options (maintain with progression or change exercises).

**Independent Test**: Use fixture user with all_w4_completed=true. Send "hola" or "qué sigue", verify agent detects renewal and offers options.

### Implementation for User Story 6

- [x] T033 [US6] Implement `get_mesocycle_status(user_id)` tool — SELECT users_plans + COUNT user_weekly_schedule WHERE week=4 AND Completed=true, return `{week4_completed, week4_total, can_renew, mesocycle_number}` in `langgraph-skeleton/cases/case6_unified_agent/tools.py`
- [x] T034 [US6] Add mesocycle renewal instructions to system prompt: when `all_w4_completed=true` in context, proactively call `get_mesocycle_status` and offer renewal options (maintain vs change) in `langgraph-skeleton/cases/case6_unified_agent/prompts.py`
- [x] T035 [US6] Wire `get_mesocycle_status` into `graph_live.py` ToolNode in `langgraph-skeleton/cases/case6_unified_agent/graph_live.py`

**Checkpoint**: Agent detects W4 completion, offers renewal options, handles "mantener pero con más días" scenario.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Tests, validation, and edge case hardening

- [x] T036 [P] Create test file `langgraph-skeleton/tests/test_case6.py` with unit tests for: `load_user_context` with mock data (new user, existing user, user with missed sessions, user with pending tasks, user with W4 complete)
- [x] T037 [P] Add integration tests in `langgraph-skeleton/tests/test_case6.py` for graph invocation: mock graph (no Supabase) — send message as existing user, verify agent responds with context; send as new user, verify router returns appropriate response
- [x] T038 ~~Moved to T009b in Phase 2~~ (was: WhatsApp status filtering — now in Foundational phase)
- [x] T039 Validate `graph_live.py` has all 11 tools wired correctly — verify ToolNode includes: get_todays_routine, confirm_workout_completion, decline_workout, create_magic_link, get_schedule_info, schedule_sessions, get_mesocycle_status, get_day_requirements, get_exercises_for_draft, find_exercise_alternatives, save_workout_plan in `langgraph-skeleton/cases/case6_unified_agent/graph_live.py`
- [x] T040 Run full quickstart.md validation: start server, test all 4 curl commands (existing user, new user, confirm, fitness chat), verify responses in `langgraph-skeleton/`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (T001-T003) — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 — core daily flow
- **US2 (Phase 4)**: Depends on Phase 2 — can run in parallel with US1 (different tools)
- **US3 (Phase 5)**: Depends on Phase 2 + `supabase_bulk_insert` from Phase 1 — needs `graph_live.py` from US1 Phase 3
- **US4 (Phase 6)**: Depends on Phase 2 + `supabase_bulk_insert` — can start after US1
- **US5 (Phase 7)**: Depends on US1 (needs `confirm_workout_completion` and `decline_workout` tools)
- **US6 (Phase 8)**: Depends on Phase 2 — independent of other stories
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

```
Phase 1 (Setup)
    │
    ▼
Phase 2 (Foundational) ── BLOCKS ALL ──┐
    │                                    │
    ├──▶ US1 (P1: Ver/Confirmar) ◄──────┤
    │       │                            │
    ├──▶ US2 (P1: KYC) ◄───────────────┤   (US1 & US2 can run in parallel)
    │       │                            │
    ├──▶ US3 (P2: Draft Routine) ◄──────┤   (needs graph_live.py from US1)
    │       │                            │
    ├──▶ US4 (P2: Agendamiento) ◄───────┤   (can start after US1)
    │       │                            │
    ├──▶ US5 (P3: Pending + Chat) ◄─────┤   (needs US1 tools)
    │       │                            │
    └──▶ US6 (P3: Mesociclo) ◄──────────┘   (independent)
            │
            ▼
       Phase 9 (Polish)
```

### Within Each User Story

- Tools (marked [P]) can be implemented in parallel within the same story
- Node wiring depends on tools being complete
- Graph integration depends on nodes being complete
- System prompt updates can be done alongside tool implementation

### Parallel Opportunities

**Within Phase 1**:
- T002 and T003 modify the same file sequentially, but both are needed before Phase 3+

**Within Phase 2**:
- T005 (context_loader) and T006 (prompts) are independent — can run in parallel
- T004 (state) must come first — T005, T006, T007 depend on it

**Within US1 (Phase 3)**:
- T010, T011, T012, T013 are all independent tools — run in parallel
- T014 depends on T010-T013 (needs tools to bind)
- T015 depends on T014 (needs agent node)

**Within US3 (Phase 5)**:
- T020, T021, T022 are independent tools — run in parallel
- T023 depends on `supabase_bulk_insert` (T003)
- T024-T025 depend on T020-T023

**Cross-story parallelism**:
- US1 and US2 can be worked on simultaneously
- US4 and US6 can be worked on simultaneously after US1

---

## Parallel Example: User Story 1

```bash
# Launch all 4 tools in parallel (different functions in same file, independent):
Task T010: "Implement get_todays_routine tool"
Task T011: "Implement confirm_workout_completion tool"
Task T012: "Implement decline_workout tool"
Task T013: "Implement create_magic_link tool"

# Then sequentially:
Task T014: "Update kairos_agent to bind tools + ReAct loop"
Task T015: "Build graph_live.py with tools"
Task T016: "Update server.py for live switching"
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T009)
3. Complete Phase 3: US1 — Ver y Confirmar Rutina (T010-T016)
4. **STOP and VALIDATE**: Test with fixture user 570000000003 via curl
5. Agente puede: ver rutina, confirmar, magic link, sesiones perdidas, pending tasks

### Incremental Delivery

1. Setup + Foundational → Agent shell responding with context
2. + US1 → Daily routine flow working (MVP!)
3. + US2 → New users can onboard via KYC
4. + US3 → Interactive routine creation with draft mode
5. + US4 → Users can schedule their training days
6. + US5 → Pending task priority + fitness chat
7. + US6 → Mesocycle renewal detection
8. Each story adds value without breaking previous stories

---

## Notes

- All tools go in the same file (`tools.py`) — [P] markers indicate the functions within are independent
- `graph_live.py` is progressively built: US1 adds tool loop, US2 adds KYC subgraph, US3-US6 add more tools to the ToolNode
- System prompt (`prompts.py`) grows with each story — new behavior rules are added incrementally
- Use fixture phones from CLAUDE.md for testing: 570000000003 (with routine), 570000000004 (with pending task), 579000000001 (new user)
- Total: 40 tasks across 9 phases
