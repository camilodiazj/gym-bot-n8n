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
│   ├── wip/                   # Work-in-progress workflows
│   ├── deprecated/            # Old workflow versions (backup)
│   └── system_prompts/        # AI agent system prompts
├── workout-tracker/           # React/TypeScript frontend (Vite)
├── workout-tracker-back/      # Go/Gin backend (hexagonal architecture)
├── e2e/                       # E2E test fixtures and documentation
└── exercises/                 # Exercise data and utilities
```

### n8n Workflows

| Workflow | Purpose |
|----------|---------|
| `GymRatFlow_Supabase_V2.json` | Main orchestrator - handles WhatsApp messages, user validation, intention detection (CONFIRMAR_RUTINA, VER_RUTINA_DE_HOY, CHAT), and routine display |
| `GymRatForm Supabase v2.1.json` | **Advanced routine generation** - creates personalized 4-week workout plans using full user profile (22 fields) with duration validation |
| `MorningReminder-WorkoutTracker.json` | Daily workout reminders and completion tracking |
| `GymBotMesocycleRenewal.json` | Handles 4-week mesocycle renewal flow |
| `GymRatFlow_E2E_TestRunner.json` | Automated E2E test suite - validates all user flows |
| `GymBotWorkoutCompletion_E2E_TestRunner.json` | E2E test suite for workout completion workflow (4 test cases) |

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
| `users_gym_profile` | KYC profile data from onboarding | `whatsapp_id` (PK), fitness metrics, goals, preferences (22 columns) |
| `users_plans` | Active training plan per user | `plan_id` (UUID PK), `user_id` -> `users`, `template_id`, `goal`, `level`, `status`, `mesocycle_number`, `last_renewal_date` |

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
| `exercises` | Exercise catalog | `exercise_id` (PK), `spanish_name`, `pattern`, `role`, `main_muscle`, `level`, `link` |
| `exercise_patterns` | Movement patterns (e.g., hip_hinge, push) | `pattern` (PK), `detail` |
| `exercise_role` | Exercise classifications | `role` (PK): compound, isolation, core |
| `muscles` | Muscle groups | `main_muscle` (PK), `main_muscle_spanish` |

### Workout Programming

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `workouts` | User-assigned exercises | `id` (UUID PK), `user_id`, `week`, `day_name`, `exercise_id`, `sets`, `reps`, `rir`, `rest-seconds`, `tempo` |
| `set_profiles` | Loading parameters by goal/level/week | `profile_id` (PK), `goal`, `level`, `week`, `role`, `sets`, `reps`, `rir`, `rest_sec`, `tempo` |
| `user_weekly_schedule` | Scheduled workout sessions | `day_routine_id` (UUID PK), `user_id`, `week`, `week_day` (enum), `session_name`, `planned_day`, `Completed` |

### Pending Tasks (Confirmation Flow)

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `pending_tasks` | Tracks pending user confirmations | `task_id` (UUID PK), `user_id`, `task_type`, `related_id`, `session_name`, `week`, `status`, `created_at`, `resolved_at` |

Task types: `CONFIRMAR_RUTINA` (workout completion confirmation)

### Reference/Lookup Tables

| Table | Purpose |
|-------|---------|
| `user_levels` | Fitness levels: Principiante, Intermedio, Avanzado |
| `user_goals` | Training goals (5 options) |
| `health_status` | Health condition codes (A-E) |
| `routine_environments` | Training location (GYM) |
| `n8n_chat_histories` | AI conversation memory storage |

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

### 1. GymRatFlow_Supabase (Main Orchestrator)

| Action | Tables Used | Operation |
|--------|-------------|-----------|
| User lookup | `users` | SELECT by `full_phone_number` |
| Schedule check | `user_weekly_schedule` | SELECT by `user_id` |
| Workout retrieval | `workouts` + `exercises` | JOIN query |
| Mark completed | `user_weekly_schedule` | UPDATE `Completed = true` |
| Schedule creation | `user_weekly_schedule` | INSERT via tool |
| Plan info | `users_plans` + `week_schedules` + `template_days` | JOIN query |

### 2. GymRatForm Supabase v2 (Advanced Routine Generator)

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
|  WhatsApp -> KYC Agent -> users_gym_profile -> GymRatForm workflow  |
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

```bash
./deploy.sh              # Build frontend + deploy both to Cloud Run
```

### n8n Workflows

Workflows are JSON files—import directly into n8n instance and configure credentials.

## Development Notes

- **Language**: All system prompts and user-facing content must be in Spanish
- **Timezone**: Configured for America/Bogota
- **Credentials**: OpenAI, Google Gemini, Supabase, WhatsApp APIs (managed in n8n)

### Workflow Conventions

- Node names use snake_case
- Conditional nodes check user existence and scheduled routines before proceeding
- `alwaysOutputData: true` preserves data flow through false conditions
- `executeOnce: true` prevents duplicate processing on loops

## E2E Testing

The `/e2e/` directory contains automated end-to-end tests:

```
e2e/
├── GymRatFlow_test_plan.md   # Test documentation and execution guide
└── test_data_setup.sql       # SQL to create fixture users (run once)
```

### Test Runner

`GymRatFlow_E2E_TestRunner.json` is an n8n workflow that runs 9 test cases:

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

### Running Tests

1. **First time setup**: Run `e2e/test_data_setup.sql` in Supabase to create fixture users
2. **Import workflow**: Import `GymRatFlow_E2E_TestRunner.json` into n8n
3. **Configure credentials**: Postgres (Supabase), OpenAI API
4. **Execute**: Click "Test Workflow" - results appear in "Generate Report" node

### Test Users (Reserved Phones)

| Phone | User | Purpose |
|-------|------|---------|
| `570000000001` | Test_NoSchedule | TC003 |
| `570000000002` | Test_RestDay | TC004 |
| `570000000003` | Test_WithRoutine | TC006, TC007 |
| `570000000004` | Test_WithPendingTask | TC011, TC012 |
| `570000000009` | Dynamic (created/deleted) | TC002, TC002_FULL_KYC |

> **Important**: Phone numbers `57000000000X` are reserved for testing. Do not use for real users.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.
