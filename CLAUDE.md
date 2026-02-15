# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GymBot is an AI-powered fitness coaching platform built on n8n workflows. It provides personalized workout plans and daily accountability through WhatsApp, targeting a Spanish-speaking (Colombian) audience.

**Tech Stack:**
- n8n (workflow automation platform)
- LLMs: OpenAI GPT-5.x, Google Gemini 2.0-flash
- Database: Supabase (PostgreSQL)
- Messaging: WhatsApp Business API

## Architecture

```
GymBot/
├── n8n/                       # n8n workflow automation
│   ├── running_flows/         # Active production workflows
│   ├── tests/                 # E2E test runners
│   └── archived/              # Archived workflow versions
├── workout-tracker/           # React/TypeScript frontend (Vite)
├── workout-tracker-back/      # Go/Gin backend (hexagonal architecture)
├── e2e/                       # E2E test fixtures and documentation
├── exercises/                 # Exercise data and utilities
├── spec/                      # Feature specifications (home-training, mesocycle-renewal, email)
├── docs/                      # Deployment guides and feature docs
└── ActionBody/                # Gym equipment photos (ActionBody inventory)
```

> **Note**: `n8n-mcp/` is gitignored — it's a local n8n MCP server dependency, not part of the GymBot codebase.

### n8n Workflows

**Production workflows** (`n8n/running_flows/`):

| Workflow | Purpose |
|----------|---------|
| `MAIN_FLOW.json` | Main orchestrator - handles WhatsApp messages, user validation, intention detection (CONFIRMAR_RUTINA, VER_RUTINA_DE_HOY, CHAT), and routine display |
| `WORKOUT_CREATOR.json` | **Advanced routine generation** - creates personalized 4-week workout plans using full user profile (22 fields) with duration validation |
| `MorningReminder-WorkoutTracker.json` | Daily workout reminders and completion tracking |
| `GymBotMesocycleRenewal.json` | Handles 4-week mesocycle renewal flow |

**Test workflows** (`n8n/tests/`):

| Workflow | Purpose |
|----------|---------|
| `GymRatFlow_E2E_TestRunner.json` | Automated E2E test suite - validates all user flows (parallel multi-turn execution) |
| `MesocycleRenewal_E2E_TestRunner.json` | E2E test suite for mesocycle renewal scenarios (3 test cases: auto-detect, MANTENER, manual intent) |
| `QualityFixes_E2E_TestRunner.json` | E2E test suite for WORKOUT_CREATOR quality fixes (QF-1 through QF-5) |

> `GymRatFlow_MultiTurnExecutor.json` is deployed directly in the n8n instance (not in the repo). It's the sub-workflow called by test runners for isolated multi-turn test execution.

### Workout Tracker (Web App)

**Frontend** (`workout-tracker/`): React 19 + TypeScript + Vite + Tailwind CSS
- Exercise tracking UI with set completion
- Deployed to Firebase Hosting

**Backend** (`workout-tracker-back/`): Go + Gin with hexagonal architecture
- REST API for workout data
- Connects to Supabase PostgreSQL
- Deployed to Google Cloud Run

```
workout-tracker-back/
├── cmd/api/              # Entry point
├── internal/
│   ├── domain/           # Core business logic (entities, repository interfaces, services)
│   ├── application/      # Use cases and DTOs
│   ├── adapter/          # HTTP handlers and PostgreSQL repository
│   └── config/
└── pkg/                  # Shared utilities (apperror, response)
```

### Data Flow Patterns

1. **User Onboarding**: WhatsApp → KYC Agent → Form submission → Profile creation → Routine generation
2. **Daily Routine**: User message → Intention detection → Routine retrieval → Formatted WhatsApp delivery
3. **Completion Tracking**: 8 PM trigger → Query uncompleted workouts → AI follow-up → Status update
4. **Mesocycle Renewal**: Week 4 completed → Detect completion → Offer options (maintain/change) → Update plan/workouts

### Multi-Agent Architecture

Each workflow uses specialized AI agents with Spanish system prompts:
- **KYC Agent**: Collects user profile information
- **Intention Agent**: Classifies user messages (including RENOVAR_MESOCICLO)
- **Confirmation Agent**: Handles schedule confirmations
- **Workout Display Agent**: Formats and presents routines
- **Renewal Agent**: Handles mesocycle renewal conversation (maintain routine, change days, rotate exercises)

Agents use Postgres-based chat memory for conversation context persistence.

## Database Schema (Supabase/PostgreSQL)

### Core User Tables

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `users` | Core user identity | `user_id` (UUID PK), `full_name`, `email`, `cel_number`, `full_phone_number`, `timezone` |
| `users_gym_profile` | KYC profile data from onboarding | `whatsapp_id` (PK), fitness metrics, goals, preferences (24 columns) |
| `users_plans` | Active training plan per user | `plan_id` (UUID PK), `user_id` -> `users`, `template_id`, `goal`, `level`, `status`, `mesocycle_number`, `last_renewal_date` |

### users_gym_profile Enum Values

**IMPORTANT**: When inserting into `users_gym_profile`, use these exact enum values:

| Column | Enum Type | Valid Values |
|--------|-----------|--------------|
| `biological_sex` | `sex` | `M`, `F` |
| `primary_goal` | `goal` | `Ganar masa muscular`, `Bajar grasa`, `Mejorar fuerza`, `Mejorar resistencia`, `Salud general / recomposición corporal` |
| `training_experience` | `gym_experience` | `Nunca he entrenado`, `Menos de 6 meses`, `6 a 12 meses`, `1 a 3 años`, `Más de 3 años` |
| `current_frequency` | `current_gym_frecuency` | `No entreno`, `1-2 días por semana`, `3-4 días por semana`, `5-6 días por semana` |
| `preferred_schedule` | `usual_schedule` | `Mañana`, `Tarde`, `Noche` |
| `training_style` | `workout_preferences` | `Pesas libres`, `Máquinas`, `Funcional`, `Mixto` |
| `cardio_type` | `current_cardio` | `No`, `Caminata`, `Bicicleta`, `Running` |
| `cardio_frequency` | `cardio_frequency` | `0`, `1-2`, `3-4`, `5 o más` |

**Text columns** (NOT enums): `fitness_level`, `health_status`, `secondary_goal`, `priority_muscles`, `disliked_exercises`, `session_duration_mins`, `training_environment`, `home_equipment`

### Routine Template System

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `routine_templates` | Pre-built training programs | `template_id` (PK), `week_schedule`, `goal`, `level`, `days_per_week`, `environment` |
| `template_days` | Day structure per schedule type | `template_day_id` (PK), `week_schedule`, `day_number`, `title` |
| `day_requirements` | Exercise patterns required per day | `day_req_id` (PK), `template_day_id`, `pattern`, `min_sets`, `priority` |
| `week_schedules` | Schedule type definitions | `schedule_type` (PK), `days_per_week`, `detail` |

### Exercise Library

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `exercises` | Exercise catalog (1657 exercises) | `exercise_id` (PK), `spanish_name`, `pattern`, `role`, `main_muscle`, `secondary_muscles` (array), `level`, `link`, `equipment` |
| `exercise_patterns` | Movement patterns (e.g., hip_hinge, push) | `pattern` (PK), `detail` |
| `exercise_role` | Exercise classifications | `role` (PK): compound, isolation, core |
| `muscles` | Muscle groups | `main_muscle` (PK), `main_muscle_spanish` |

### Workout Programming

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `workouts` | User-assigned exercises | `id` (UUID PK), `user_id`, `week`, `day_name`, `exercise_id`, `sets`, `reps`, `rir`, `rest-seconds`, `tempo`, `exercise_order` |
| `set_profiles` | Loading parameters by goal/level/week | `profile_id` (PK), `goal`, `level`, `week`, `role`, `sets`, `reps`, `rir`, `rest_sec`, `tempo` |
| `user_weekly_schedule` | Scheduled workout sessions | `day_routine_id` (UUID PK), `user_id`, `week`, `week_day` (enum), `session_name`, `planned_day`, `Completed` |
| `set_values` | User-recorded weights/reps per set | `id` (UUID PK), `user_id`, `exercise_id`, `workout_id`, `set_number`, `actual_weight`, `actual_reps`, `recorded_at` |

### Pending Tasks (Confirmation Flow)

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `pending_tasks` | Tracks pending user confirmations | `task_id` (UUID PK), `user_id`, `task_type`, `related_id`, `session_name`, `week`, `status`, `created_at`, `resolved_at` |

Task types: `CONFIRMAR_RUTINA` (workout completion confirmation)

### Authentication (Workout Tracker)

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `magic_links` | Passwordless auth via WhatsApp deep links | `code` (VARCHAR PK), `user_id`, `created_at`, `expires_at` (24h default), `used_at` |

**Magic link URL format**: `https://workout-tracker-69b08.web.app/w?c={code}` (local: `http://localhost:5173/w?c={code}`). The `code` is a short hex string (e.g., `7fda02`). Do NOT use `/auth?code=` — that path does not exist.

### Reference/Lookup Tables

| Table | Purpose |
|-------|---------|
| `user_levels` | Fitness levels: Principiante, Intermedio, Avanzado |
| `user_goals` | Training goals (5 options) |
| `health_status` | Health condition codes (A-E) |
| `routine_environments` | Training location (GYM) |
| `n8n_chat_histories` | AI conversation memory storage |
| `e2e_test_run_results` | Temporary storage for parallel E2E test results (`run_id`, `test_id`, `result` JSONB) |

### Exercise Ordering

The `exercise_order` field in `workouts` ensures deterministic ordering:
- **compound** exercises: 1-4 (heavy lifts first)
- **core** exercises: 5-6 (after main lifts)
- **isolation** exercises: 7+ (accessories last)

This is set programmatically in `WORKOUT_CREATOR.json` and queried with `ORDER BY exercise_order`.

### Entity Relationships

```
users <------------------------------------------+
  |                                              |
  +-> users_plans -> routine_templates           |
  |        |              |                      |
  |        |              +-> template_days      |
  |        |                      |              |
  |        +-> week_schedules <---+              |
  |                                              |
  +-> user_weekly_schedule                       |
  |                                              |
  +-> workouts -> exercises                      |
                      |                          |
                      +-> exercise_patterns      |
                      +-> muscles                |
                      +-> user_levels            |
                                                 |
users_gym_profile ------------------------------>+
     (linked via whatsapp_id / phone matching)
```

## n8n Workflow <-> Supabase Mapping

### 1. MAIN_FLOW (Main Orchestrator)

| Action | Tables Used | Operation |
|--------|-------------|-----------|
| User lookup | `users` | SELECT by `full_phone_number` |
| Schedule check | `user_weekly_schedule` | SELECT by `user_id` |
| Workout retrieval | `workouts` + `exercises` | JOIN query |
| Mark completed | `user_weekly_schedule` | UPDATE `Completed = true` |
| Schedule creation | `user_weekly_schedule` | INSERT via tool |
| Plan info | `users_plans` + `week_schedules` + `template_days` | JOIN query |

### 2. WORKOUT_CREATOR (Advanced Routine Generator)

| Action | Tables Used | Operation |
|--------|-------------|-----------|
| Load user profile | `users_gym_profile` | SELECT by `whatsapp_id` |
| **Process preferences** | (in-memory) | `ProcessUserPreferences` node transforms profile |
| Get set profiles | `set_profiles` | SELECT by `goal`, `level` |
| Get day requirements | `routine_templates` + `template_days` + `day_requirements` | JOIN query |
| Find exercises | `exercises` | SELECT by `pattern` |
| Create user | `users` | INSERT |
| Create plan | `users_plans` | INSERT |
| Save workouts | `workouts` | INSERT (bulk) |

#### ProcessUserPreferences Node

New Code node that transforms user profile data for AI personalization:

| Input (Spanish) | Output (English) | Purpose |
|-----------------|------------------|---------|
| `priority_muscles` | `processed.priority_muscles_en` | Maps "Glúteo, pierna" → ["Glutes", "Quads", "Hamstrings"] |
| `disliked_exercises` | `processed.disliked_muscles_en` | Maps "Pantorrillas" → ["Calfs"] |
| `training_experience` | `processed.experience_tier` | "Más de 3 años" → "advanced" |
| `session_duration_mins` | `processed.volume_modifier` | "45-60 min" → 0.85 (reduce volume) |
| `health_status` | `processed.health.*` | "C" → `avoid_upper_body_overhead: true` |

#### Health Status Codes

| Code | Restriction | AI Behavior |
|------|-------------|-------------|
| A | None | Full exercise selection |
| B | Lower body issues | Avoid high-impact on knees/ankles |
| C | Upper body issues | **Avoid overhead pressing** |
| D | Spine issues | Avoid heavy axial loading |
| E | Special condition | Prioritize machines, low-risk exercises |

#### Personalization Rules (AI Agent)

1. **Exclusion**: Remove exercises where `main_muscle` matches disliked muscles
2. **Prioritization**: Prefer exercises matching priority muscles (main or secondary)
3. **Sex adaptation**: F→Glutes/Hamstrings, M→Chest/Back/Shoulders
4. **Experience**: Beginner→machines, Advanced→barbell/compound
5. **Volume**: Apply `volume_modifier` to isolation exercises

### 3. GymBotWorkoutCompletion (8 PM Follow-up)

| Action | Tables Used | Operation |
|--------|-------------|-----------|
| Get today's uncompleted | `user_weekly_schedule` | SELECT WHERE `planned_day = today` AND `Completed = false` |
| Check existing pending_task | `pending_tasks` | SELECT by `user_id` + `related_id` (Merge with keepNonMatches filters duplicates) |
| Get user contact | `users` | SELECT by `user_id` |
| Create pending_task | `pending_tasks` | INSERT (only if no existing task for same workout) |
| Store conversation | `n8n_chat_histories` | INSERT (via Postgres Memory) |

### 4. RoutineMorningReminder (5 AM Reminder)

| Action | Tables Used | Operation |
|--------|-------------|-----------|
| Get scheduled routines | `user_weekly_schedule` | SELECT WHERE `planned_day = today` |
| Get full workout details | `users` + `user_weekly_schedule` + `workouts` + `exercises` | Complex JOIN query |

## Data Flow Summary

```
+---------------------------------------------------------------------+
|                         USER ONBOARDING                              |
|  WhatsApp -> KYC Agent -> users_gym_profile -> WORKOUT_CREATOR      |
|                              |                                       |
|            users + users_plans + workouts (4 weeks generated)        |
+---------------------------------------------------------------------+
                                    |
+---------------------------------------------------------------------+
|                         DAILY OPERATIONS                             |
|  5 AM: RoutineMorningReminder -> user_weekly_schedule + workouts    |
|        -> WhatsApp with full routine                                 |
|                                                                      |
|  User message: GymRatFlow -> Intention detection ->                  |
|        VER_RUTINA_DE_HOY: workouts + exercises -> WhatsApp           |
|        CONFIRMAR_RUTINA: user_weekly_schedule.Completed = true       |
|                                                                      |
|  8 PM: GymBotWorkoutCompletion -> uncompleted schedules ->          |
|        -> Follow-up WhatsApp                                         |
+---------------------------------------------------------------------+
```

## Development Commands

### Frontend (workout-tracker/)

```bash
cd workout-tracker
npm install              # Install dependencies
npm run dev              # Start dev server (http://localhost:5173)
npm run build            # Build for production
npm test                 # Run tests (Vitest)
npm run test:watch       # Watch mode
npm run test:coverage    # Tests with coverage report
```

### Backend (workout-tracker-back/)

```bash
cd workout-tracker-back
make deps                # Install Go dependencies
make run                 # Run server (http://localhost:8080)
make build               # Build binary
make test                # Run tests
make test-coverage       # Tests with coverage
make lint                # Run linter (requires golangci-lint)
make fmt                 # Format code
make dev                 # Hot reload (requires air)
```

### Deployment

- **Frontend**: Firebase Hosting (manual deploy via Firebase CLI)
- **Backend**: Google Cloud Run (deploy via `gcloud run deploy`)
- **Production API**: `https://workout-api-148665080566.us-central1.run.app/api/v1`

### n8n Workflows

Workflows are JSON files—import directly into n8n instance and configure credentials.

## Development Notes

- **Language**: All system prompts and user-facing content must be in Spanish
- **Timezone**: Configured for America/Bogota
- **Credentials**: OpenAI, Google Gemini, Supabase, WhatsApp APIs (managed in n8n)

### Backend Go Notes

- **Reps format**: Can be single number ("10") or range with hyphen ("10-12") or en-dash ("6–8"). The `parseReps` functions handle both.
- **API endpoints**:
  - `GET /api/v1/workouts/today?user_id=UUID` - Get today's workout
  - `POST /api/v1/auth/magic-link` - Validate magic link code
  - `PATCH /api/v1/sets/:id` - Update set (weight, reps, completed)

### Workflow Conventions

- Node names use snake_case
- Conditional nodes check user existence and scheduled routines before proceeding
- `alwaysOutputData: true` preserves data flow through false conditions
- `executeOnce: true` prevents duplicate processing on loops

### n8n Code Node Sandbox Restrictions

n8n Code nodes run in a sandboxed environment. **These globals are NOT available**:
- `crypto` - Use `$execution.id` for unique IDs, or `Date.now().toString(36) + Math.random().toString(36).slice(2)` for random strings
- `process`, `require`, `Buffer` - No Node.js built-ins
- `fetch` - Use `this.helpers.httpRequest()` instead (only in "Run Once for All Items" mode)

**Available n8n globals**: `$input`, `$json`, `$execution`, `$node`, `$env`, `$now`, `$today`, `DateTime` (Luxon), `console.log`

### n8n Known Pitfalls

**Switch node with 6+ outputs**: Switch v3.2 unreliably routes to the last output when there are 6+ outputs. **Workaround**: Use If node + SplitInBatches loop instead of Switch for parallel routing.

**Task runner capacity**: Code nodes require a "task runner" slot. Running 5+ parallel sub-workflows with Code nodes can exhaust all runner slots. **Fix**: Replace Code-based logic with Postgres nodes + If nodes where possible (no runners needed).

**customData after Wait node**: `$execution.customData.get()` does NOT work after Wait node resume — in ANY node type (Postgres SQL, If conditions, and Code nodes all fail). **Best fix**: Avoid Wait nodes entirely. Use `this.helpers.httpRequest()` in a Code node to poll an external API (e.g., Supabase REST API) with an `await sleep()` loop, keeping all state in local JS variables.

### n8n Credentials Reference

- **Postgres (Supabase)**: Credential ID `vZLJtIWG5nYXMez4` — use this when configuring Postgres nodes in workflows

## E2E Testing

The `/e2e/` directory contains automated end-to-end tests:

```
e2e/
├── GymRatFlow_test_plan.md   # Test documentation and execution guide
└── test_data_setup.sql       # SQL to create fixture users (run once)
```

### Test Runner

`GymRatFlow_E2E_TestRunner.json` runs test cases using parallel execution for multi-turn tests: SINGLE tests run sequentially, while multi-turn tests (MULTI_TURN / MULTI_TURN_AI) each run in a parallel lane via `GymRatFlow_MultiTurnExecutor.json` sub-workflow:

| Test | Category | Description |
|------|----------|-------------|
| TC001 | FILTRO_RUIDO | Ignores WhatsApp status updates |
| TC002 | ONBOARDING | New user triggers KYC flow |
| TC002_FULL_KYC | ONBOARDING_FULL | AI-simulated user completes entire KYC |
| TC003 | AGENDAR | User without schedule gets scheduling flow |
| TC004 | DESCANSO | User with no workout today gets rest message |
| TC006 | VER_RUTINA | User sees formatted routine |
| TC007 | CHAT | General fitness questions answered |
| TC011 | PENDING_TASK | User confirms workout completion |
| TC012 | PENDING_TASK | User declines confirmation |
| TC013 | BUTTON_INPUT | Button input - VER_RUTINA_DE_HOY via interactive reply |
| TC_HOME_FULL_BASIC | HOME_BASIC | AI-simulated HOME user with basic equipment completes KYC |
| TC_HOME_FULL_BODYWEIGHT | HOME_BODYWEIGHT | AI-simulated HOME bodyweight-only user completes KYC |
| TC_HOME_FULL_HEALTH_C | HOME_HEALTH | AI-simulated HOME user with health restriction C completes KYC |
| TC_MESO_001 | MESOCYCLE_RENEWAL | Auto-detection of W4 completion triggers renewal |
| TC_MESO_002 | MESOCYCLE_RENEWAL | MANTENER_RUTINA flow completes successfully |
| TC_MESO_003 | MESOCYCLE_RENEWAL | Manual RENOVAR_MESOCICLO intent detected |

### Running Tests

1. **First time setup**: Run `e2e/test_data_setup.sql` in Supabase to create fixture users
2. **Import sub-workflow first**: Import `GymRatFlow_MultiTurnExecutor.json` into n8n
3. **Import test runner**: Import `GymRatFlow_E2E_TestRunner.json` (the 5 "Execute MT Lane" nodes already reference the sub-workflow ID)
4. **Configure credentials**: Postgres (Supabase), OpenAI API
5. **Execute**: Click "Test Workflow" - multi-turn tests run in parallel, results appear in "Generate Report" node

### Test Users (Reserved Phones)

**GYM Users** (`57000000000X`) - Pre-populated fixtures:

| Phone | User | Purpose |
|-------|------|---------|
| `570000000001` | Test_NoSchedule | TC003 |
| `570000000002` | Test_RestDay | TC004 |
| `570000000003` | Test_WithRoutine | TC006, TC007 |
| `570000000004` | Test_WithPendingTask | TC011, TC012 |
| `570000000009` | Dynamic (created/deleted) | TC002, TC002_FULL_KYC |

**HOME Users** (`5700000002XX`) - Created dynamically by MULTI_TURN_AI tests:

| Phone | User | Equipment | Health | Purpose |
|-------|------|-----------|--------|---------|
| `570000000211` | Maria Lopez | mancuernas, bandas | A | TC_HOME_FULL_BASIC |
| `570000000212` | Carlos Rodriguez | peso corporal | A | TC_HOME_FULL_BODYWEIGHT |
| `570000000213` | Ana Martinez | mancuernas, bandas | C | TC_HOME_FULL_HEALTH_C |

**MESOCYCLE Users** (`5700000005XX`) - Pre-populated fixtures:

| Phone | User | Purpose |
|-------|------|---------|
| `570000000051` | Test_MesoDetect | TC_MESO_001 |
| `570000000052` | Test_MesoMantener | TC_MESO_002 |
| `570000000053` | Test_MesoManual | TC_MESO_003 |

> **Important**: Phone numbers `57000000000X`, `5700000002XX`, and `5700000005XX` are reserved for testing. Do not use for real users.

### Teardown (Critical)

The `test_data_setup.sql` script includes a **teardown section** that deletes ALL test users (GYM + HOME) before recreating fixtures. This ensures:

1. **Clean state** before each test run
2. **Re-runnable tests** - HOME users are deleted so MULTI_TURN_AI can recreate them
3. **No stale data** - Old workouts/schedules don't pollute new test runs

**When adding new test users**: Always add their phone numbers to BOTH:
- The teardown DELETE statements (Section 1)
- The summary table at the end of the script

```sql
-- Phones included in teardown:
-- GYM: 570000000001, 570000000002, 570000000003, 570000000004, 570000000009
-- HOME: 570000000211, 570000000212, 570000000213
-- MESOCYCLE: 570000000051, 570000000052, 570000000053
```

### Creating Test Users Manually (SQL Template)

When creating ad-hoc test users (e.g., to replicate a real user's workout for local testing), use the following template. Many columns are NOT NULL without defaults — omitting them causes hard-to-debug insert failures.

**Required NOT NULL columns per table:**

| Table | Column | Type | Gotcha |
|-------|--------|------|--------|
| `users` | `created_at` | timestamptz | No default — must supply `NOW()` |
| `users` | `country_indicative` | bigint | No default — use `57` for Colombia |
| `users_plans` | `start_date` | timestamptz | No default — use `NOW()` |
| `users_plans` | `week_schedule` | text | FK to `week_schedules.schedule_type` — valid values: `fb_2`, `fb_3`, `ul_4`, `ppl_5`, `ppl_6` |
| `user_weekly_schedule` | `planned_day` | text | NOT NULL — use `TO_CHAR(NOW() AT TIME ZONE 'America/Bogota', 'YYYY-MM-DD')` |
| `user_weekly_schedule` | `planned_day_utc` | timestamptz | Separate from `planned_day` — use `DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota'` |

> **CRITICAL — `planned_day` format inconsistency**: Fixture SQL writes `planned_day` in ISO format (`2026-02-15`) and populates `planned_day_utc`. But the MAIN_FLOW scheduling tool (`Tool_Update_User_Weekly_Schedule`) writes `planned_day` in `DD/MM` format (e.g., `"30/10"`) and leaves `planned_day_utc` as **NULL**. When querying records created by the scheduling tool, do NOT filter on `planned_day_utc` (it's NULL) or compare `planned_day` with `CURRENT_DATE::text` (format mismatch). Instead, filter by `user_id` + `week` or just `user_id`.
| `workouts` | `created_at` | timestamptz | No default — must supply `NOW()` |
| `workouts` | `notes` | text | NOT NULL — use `''` (empty string) |
| `magic_links` | `code` | varchar(8) | Max 8 chars |

**Full SQL template:**

```sql
-- Variables (change these)
-- user_id:  e2e00010-0000-0000-0000-000000000010
-- phone:    570000000010
-- plan_id:  e2e01000-0000-0000-0000-000000000010
-- sched_id: e2e01010-0000-0000-0000-000000000010

-- 1. User
INSERT INTO users (user_id, full_name, cel_number, full_phone_number, email, timezone, created_at, country_indicative)
VALUES ('e2e00010-0000-0000-0000-000000000010', 'Test User Name', 570000000010, '570000000010', 'test@test.com', 'America/Bogota', NOW(), 57);

-- 2. Gym profile (copy from source user, change whatsapp_id)
INSERT INTO users_gym_profile (submission_date, whatsapp_id, full_name, email, age, biological_sex, height_cm, weight_kg, primary_goal, secondary_goal, training_experience, current_frequency, fitness_level, health_status, days_available, session_duration_mins, preferred_schedule, training_style, priority_muscles, disliked_exercises, cardio_type, cardio_frequency, training_environment, home_equipment)
VALUES (NOW(), 570000000010, 'Test User Name', 'test@test.com', 27, 'M', 171, 67, 'Ganar masa muscular', 'Mejorar fuerza', 'Más de 3 años', '1-2 días por semana', 'Intermedio', 'A', 3, '60-75 minutos', 'Mañana', 'Pesas libres', 'Los brazos', 'Las pantorrillas', 'No', '0', 'GYM', NULL);

-- 3. Plan (week_schedule must be valid FK: fb_2, fb_3, ul_4, ppl_5, ppl_6)
INSERT INTO users_plans (plan_id, user_id, template_id, start_date, goal, level, status, mesocycle_number, week_schedule)
VALUES ('e2e01000-0000-0000-0000-000000000010', 'e2e00010-0000-0000-0000-000000000010', 'tpl_fb_3_hyp_int', NOW(), 'Ganar masa muscular', 'Intermedio', 'active', 1, 'fb_3');

-- 4. Today's schedule (needs BOTH planned_day text AND planned_day_utc timestamptz)
INSERT INTO user_weekly_schedule (day_routine_id, user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES ('e2e01010-0000-0000-0000-000000000010', 'e2e00010-0000-0000-0000-000000000010', 1, 'Martes', 'Full Body A',
  TO_CHAR(NOW() AT TIME ZONE 'America/Bogota', 'YYYY-MM-DD'),
  (DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota'),
  false);

-- 5. Workouts (needs created_at + notes, exercise_order determines display order)
INSERT INTO workouts (id, user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, created_at, notes, exercise_order)
VALUES (gen_random_uuid(), 'e2e00010-0000-0000-0000-000000000010', 1, 'Full Body A', 'ex_barbell_squat', '3', '8–10', '1–2', 150, '2-0-1', NOW(), '', 1);

-- 6. Magic link for frontend access (code max 8 chars, expires in 24h)
INSERT INTO magic_links (code, user_id, created_at, expires_at)
VALUES ('testcode', 'e2e00010-0000-0000-0000-000000000010', NOW(), NOW() + INTERVAL '24 hours');
-- Frontend URL: http://localhost:5173/?c=testcode

-- 7. Cleanup (run when done)
DELETE FROM workouts WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
DELETE FROM user_weekly_schedule WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
DELETE FROM users_plans WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000010;
DELETE FROM magic_links WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
DELETE FROM users WHERE user_id = 'e2e00010-0000-0000-0000-000000000010';
```

> **UUID format**: All IDs must be valid hex UUIDs. `e2e00010-plan-0000-...` is **invalid** (`plan` is not hex). Use patterns like `e2e01000-...` instead.

> **Enum values reminder**: `week_day` is an enum — valid values: `Lunes`, `Martes`, `Miercoles`, `Jueves`, `Viernes`, `Sabado`, `Domingo`.

## Feature Specifications

The `spec/` directory contains detailed implementation specs for major features:

| Feature | Directory | Contents |
|---------|-----------|----------|
| HOME Training | `spec/home-training-feature/` | KYC mods, DB templates, testing guide, training guidelines |
| Mesocycle Renewal | `spec/Mesocycle_Renewal/` | Architecture, domain logic, implementation plan |
| Email Routine | `spec/email-routine-week1/` | Workflow spec, HTML template, QA test plan |
| Quality Fixes | `spec/workout_creator_quality_fixes/` | WORKOUT_CREATOR defect fixes (QF-1 through QF-5): volume inflation, dedup, misclassified exercises, cardio role, health enforcement |

The `docs/` directory has operational docs: deployment guide, mesocycle renewal design, and WhatsApp deep link plan.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.
