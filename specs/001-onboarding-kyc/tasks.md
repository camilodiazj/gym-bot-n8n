# Tasks: Onboarding KYC (Case 5)

**Input**: Design documents from `/specs/001-onboarding-kyc/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Not explicitly requested in spec. Included in Polish phase for graph validation.

**Organization**: Tasks grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- All paths relative to `langgraph-skeleton/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create case5 directory structure and shared state definition

- [x] T001 Create case5 directory structure: `cases/case5_onboarding_kyc/__init__.py`
- [x] T002 [P] Define KYCState TypedDict with all 17 fields in `cases/case5_onboarding_kyc/state.py` per data-model.md entity #1

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared components needed by ALL user stories — system prompts, user detection, and turn tracking

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T003 Create 5 turn-specific Spanish system prompts + KYC master prompt in `cases/case5_onboarding_kyc/prompts.py` per plan.md Turn-to-Fields Mapping
- [x] T004 [P] Implement `check_user` node (mock: always returns `is_new_user=True`) in `cases/case5_onboarding_kyc/nodes.py` — FR-001
- [x] T005 [P] Implement `check_status` node with turn detection logic in `cases/case5_onboarding_kyc/nodes.py` — maps `collected_data` keys to `current_turn` (1-5) per research.md R-08

**Checkpoint**: Foundation ready — user story implementation can begin

---

## Phase 3: User Story 1 — First Contact and KYC Completion (Priority: P1) MVP

**Goal**: New user completes full 5-turn KYC via conversational agent, gets profile summary confirmed, profile saved (mock).

**Independent Test**: Send 5 sequential messages via `POST /case5/kyc/chat` with same `thread_id` → verify `is_complete=true` and all 10 `collected_fields` present in final response.

**Acceptance**: US1-1 (new user greeting), US1-2 (adaptive questions), US1-3 (multi-value parsing), US1-4 (completion + summary), US1-5 (progress indicators), US1-6 (summary rejection → targeted correction)

### Implementation for User Story 1

- [x] T006 [US1] Implement `kyc_agent` node in `cases/case5_onboarding_kyc/nodes.py` — Gemini conducts conversational KYC using turn-specific prompts from prompts.py, includes progress indicator "Pregunta X de 5" (FR-003, FR-004, FR-005, FR-006, FR-015)
- [x] T007 [US1] Implement `confirm_profile` node in `cases/case5_onboarding_kyc/nodes.py` — formats collected_data as Spanish profile summary, sets `awaiting_confirmation=True` (FR-016, FR-018)
- [x] T008 [P] [US1] Implement `save_profile` node (mock version) in `cases/case5_onboarding_kyc/nodes.py` — sets `profile_confirmed=True`, logs saved data (no Supabase yet)
- [x] T009 [US1] Build complete KYC StateGraph in `cases/case5_onboarding_kyc/graph.py` — wire: START → check_user → (existing→END, new→kyc_agent) → check_status → (continue→END, complete→confirm_profile) → (accepted→save_profile→END, rejected→kyc_agent). Use `InMemorySaver` checkpointer
- [x] T010 [US1] Add Pydantic models (`KYCChatRequest`, `KYCChatResponse`) and `POST /case5/kyc/chat` endpoint in `server.py` per contracts/fastapi-endpoints.md
- [x] T011 [P] [US1] Add `GET /case5/kyc/history` endpoint in `server.py` per contracts/fastapi-endpoints.md — returns messages, collected_data, current_turn
- [x] T012 [US1] Create standalone interactive runner in `cases/case5_onboarding_kyc/run.py` — CLI loop that simulates multi-turn KYC conversation

**Checkpoint**: Full KYC flow works end-to-end via Postman/curl. New user can complete 5 turns, see profile summary, confirm, and get success response.

---

## Phase 4: User Story 2 — KYC Abandonment and Resumption (Priority: P2)

**Goal**: Users who stop mid-KYC get a nudge after 30 min, resume seamlessly from where they left off, and sessions expire after 7 days.

**Independent Test**: Send 3 messages (partial KYC), wait, send 4th message with same `thread_id` → verify conversation resumes from Turn 4 (not Turn 1). Verify via `/case5/kyc/status` that session state shows partial data.

**Acceptance**: US2-1 (30 min nudge), US2-2 (seamless resumption), US2-3 (single nudge limit), US2-4 (7-day expiration restart)

### Implementation for User Story 2

- [x] T013 [US2] Extend `check_status` node in `cases/case5_onboarding_kyc/nodes.py` — update `last_interaction_at` on each invocation, set resumption-aware response context (FR-007)
- [x] T014 [US2] Update `kyc_agent` prompts in `cases/case5_onboarding_kyc/prompts.py` — add resumption greeting template: "Hola de nuevo, [name]! Quedamos en la pregunta X de 5..." (FR-007)
- [x] T015 [US2] Implement inactivity nudge background task in `server.py` — FastAPI `on_startup` task that checks sessions every 5 min, sends nudge when `now - last_interaction_at > 30min` and `nudge_sent=False`, marks `nudge_sent=True` (FR-008, FR-009, research R-06)
- [x] T016 [US2] Add `GET /case5/kyc/status` endpoint in `server.py` per contracts/fastapi-endpoints.md — returns session status, collected fields, remaining fields, nudge status (FR-019)

**Checkpoint**: Partial KYC sessions persist and resume correctly. Nudge fires after 30 min. Status endpoint shows session state.

---

## Phase 5: User Story 3 — Data Correction During KYC (Priority: P3)

**Goal**: Users can correct a previously answered field mid-flow without restarting. Kairos detects correction intent, updates the specific field, and continues.

**Independent Test**: Complete 4 turns, then send "mi objetivo no es ganar masa, es bajar grasa" → verify `collected_data.primary_goal` changes to "Bajar grasa" while other fields remain intact. Verify via `/case5/kyc/history`.

**Acceptance**: US3-1 (single field correction), US3-2 (dependent fields re-evaluated)

### Implementation for User Story 3

- [x] T017 [US3] Add correction detection to `check_status` in `cases/case5_onboarding_kyc/nodes.py` — detect correction intent from `kyc_agent` response (e.g., user says "cambiar mi objetivo"), set `needs_correction=True` and `correction_field` (FR-010)
- [x] T018 [US3] Add correction-mode system prompt in `cases/case5_onboarding_kyc/prompts.py` — instruct Gemini to ask "¿Qué dato quieres corregir?" when `needs_correction=True`, update only that field in `collected_data` (FR-010, FR-018)
- [x] T019 [US3] Wire correction routing edge in `cases/case5_onboarding_kyc/graph.py` — add `correction` path from `check_status` back to `kyc_agent` with `needs_correction=True` context

**Checkpoint**: Correction flow works — user can change a single field mid-KYC without losing other data.

---

## Phase 6: User Story 4 — Health Condition Filter (Priority: P3)

**Goal**: Health status from Turn 5 is classified into codes A-E via Gemini. Severe conditions (E) route to human trainer recommendation instead of automated routine.

**Independent Test**: Complete 5 turns with health answer "Tengo dolor en la rodilla derecha" → verify `health_code=B` and `affected_zones=["rodilla"]`. Separately test with "Tengo problemas cardíacos" → verify `route_to_trainer=true`.

**Acceptance**: US4-1 (lower body → code B), US4-2 (severe → code E → trainer), US4-3 (no issues → code A)

### Implementation for User Story 4

- [x] T020 [US4] Create health classification prompt in `cases/case5_onboarding_kyc/prompts.py` — structured prompt with A-E code definitions, instructs Gemini to respond with JSON `{"code": "B", "zones": ["rodilla"]}` (research R-04)
- [x] T021 [P] [US4] Implement `health_classifier` node in `cases/case5_onboarding_kyc/nodes.py` — invokes Gemini with health prompt + user's Turn 5 answer, parses JSON response, sets `health_code` and `affected_zones` (FR-011, FR-012)
- [x] T022 [P] [US4] Implement `route_to_trainer` node in `cases/case5_onboarding_kyc/nodes.py` — sets `route_to_trainer=True`, generates Spanish message recommending human trainer consultation (FR-013)
- [x] T023 [US4] Add health routing edges in `cases/case5_onboarding_kyc/graph.py` — after `confirm_profile` accepted: → `health_classifier` → (safe A-D → `save_profile` → END, severe E → `route_to_trainer` → END) (SC-007)

**Checkpoint**: Health classification works end-to-end. Code A users proceed normally. Code E users get trainer recommendation and never reach automated routine generation.

---

## Phase 7: Supabase Live Integration

**Purpose**: Connect mock graph to real Supabase database for production-ready KYC

- [x] T024 Add `supabase_insert()` function to `src/shared/supabase_client.py` — POST to PostgREST with `Prefer: return=representation` header, supports upsert via `Prefer: resolution=merge-duplicates`
- [x] T025 [P] Create Supabase tools in `cases/case5_onboarding_kyc/tools_supabase.py` — `lookup_user_by_phone(phone)` queries `users` by `full_phone_number`; `save_user(data)` inserts into `users`; `save_gym_profile(data)` inserts into `users_gym_profile` with enum mapping per data-model.md Field Mapping table (research R-05)
- [x] T026 Build Supabase-connected graph in `cases/case5_onboarding_kyc/graph_live.py` — same topology as graph.py but `check_user` uses `lookup_user_by_phone` tool, `save_profile` uses `save_user` + `save_gym_profile` tools
- [x] T027 Add live endpoints (`POST /case5/kyc/live/chat`, `GET /case5/kyc/live/history`) in `server.py` — follows existing pattern of `/case3/live` and `/case4/live/chat`

**Checkpoint**: KYC flow works against real Supabase — profile data persisted in `users` + `users_gym_profile` tables with correct enum values.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Edge cases, tests, and end-to-end validation

- [x] T028 [P] Add edge case handling to `kyc_agent` prompts in `cases/case5_onboarding_kyc/prompts.py` — emoji-only input re-asks question, voice note rejection message, empty display name fallback to "Hola!" (spec Edge Cases)
- [x] T029 [P] Write pytest tests in `tests/test_case5.py` — test graph compilation, 5-turn happy path, multi-value message parsing, existing user redirect, health code E blocks save_profile, correction flow preserves other fields
- [x] T030 Update health check endpoint in `server.py` — add Case 5 endpoints to the root `/` endpoint list
- [x] T031 Validate quickstart.md end-to-end — run all curl examples from quickstart.md against running server, verify responses match contract

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1: Setup ──────────► Phase 2: Foundational ──────► Phase 3: US1 (MVP)
                                                              │
                                                              ├──► Phase 4: US2
                                                              ├──► Phase 5: US3
                                                              └──► Phase 6: US4
                                                                       │
                                                              All ◄────┘
                                                               │
                                                          Phase 7: Live
                                                               │
                                                          Phase 8: Polish
```

### User Story Dependencies

- **US1 (P1)**: Depends on Phase 2 only — **can start immediately after Foundational**
- **US2 (P2)**: Depends on US1 — needs working KYC graph + checkpointer to test resumption
- **US3 (P3)**: Depends on US1 — needs working KYC graph to test mid-flow correction
- **US4 (P3)**: Depends on US1 — needs working graph to insert health routing edges
- **US3 and US4 are independent of each other** — can run in parallel after US1

### Within Each User Story

- Prompts/state changes before node implementations
- Node implementations before graph wiring
- Graph wiring before endpoint creation
- Endpoints before runner/validation

### Parallel Opportunities

**Phase 2**: T004 and T005 can run in parallel (different functions in nodes.py, but same file — [P] marked for logical independence)

**Phase 3 (US1)**: T008 (save_profile mock) and T011 (history endpoint) are parallelizable

**Phase 6 (US4)**: T021 (health_classifier) and T022 (route_to_trainer) are parallelizable — independent nodes in same file

**Phase 7**: T025 (Supabase tools) is parallelizable with T024 (supabase_insert) since they're different files

**Cross-story parallelism**: After US1 completes, US2, US3, and US4 can all start concurrently (US3 ∥ US4 specifically)

---

## Parallel Example: User Story 1

```text
# Sequential: foundation must complete first
T003 → T004, T005 (parallel) → T006

# After kyc_agent (T006):
T007 (confirm_profile) → T008 (save_profile, parallel with T011)
                       → T009 (graph.py, depends on all nodes)
                       → T010, T011 (parallel: chat endpoint, history endpoint)
                       → T012 (run.py, after endpoints exist)
```

## Parallel Example: After US1

```text
# All three can start concurrently:
US2: T013 → T014 → T015 → T016
US3: T017 → T018 → T019
US4: T020 → T021, T022 (parallel) → T023
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T002)
2. Complete Phase 2: Foundational (T003-T005)
3. Complete Phase 3: User Story 1 (T006-T012)
4. **STOP and VALIDATE**: Test 5-turn KYC via Postman, verify profile summary + confirmation
5. Demo-ready: working conversational KYC with mock persistence

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. **US1** → Test independently → Demo (MVP: 5-turn KYC works)
3. **US4** → Test independently → Demo (health safety gate enforced)
4. **US2** → Test independently → Demo (resumption + nudge)
5. **US3** → Test independently → Demo (correction without restart)
6. **Phase 7** → Test independently → Demo (Supabase live data)
7. **Phase 8** → Polish → Production ready

> **Note**: US4 (Health Filter) is recommended before US2/US3 because it's a safety-critical feature (Constitution Principle VI) and completes the core graph topology.

### Single Developer Strategy

Phase 1 → Phase 2 → Phase 3 → Phase 6 → Phase 4 → Phase 5 → Phase 7 → Phase 8

---

## Notes

- [P] tasks = different files or logically independent functions, no blocking dependencies
- [Story] label maps task to specific user story for traceability
- All node functions are `async def` returning `dict` matching KYCState keys
- System prompts MUST be in Spanish (Colombian dialect) — Constitution Principle II
- Enum values for Supabase inserts MUST match exactly (with accents) — see CLAUDE.md
- Mock graph uses `InMemorySaver`; live graph also uses `InMemorySaver` (production will use `PostgresSaver`)
- `supabase_client.py` currently only has `supabase_query()` — `supabase_insert()` added in Phase 7
