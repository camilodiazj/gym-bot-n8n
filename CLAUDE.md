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

The n8n workflows are organized in the `/n8n/` directory:

```
n8n/
├── running_flows/          # Active production workflows
│   ├── GymRatFlow_Supabase_V2.json
│   ├── GymRatForm Supabase v2.json
│   ├── GymBotWorkoutCompletion.json
│   └── RoutineMorningReminder.json
├── tests/                  # E2E test runners
│   ├── GymRatFlow_E2E_TestRunner.json
│   └── GymBotWorkoutCompletion_E2E_TestRunner.json
├── deprecated/             # Old workflow versions (backup)
│   ├── GymRatFlow_Supabase.json
│   └── GymRatForm Supabase.json
└── system_prompts/         # AI agent system prompts
    └── RoutineCreation.txt
```

| Workflow | Purpose |
|----------|---------|
| `GymRatFlow_Supabase_V2.json` | Main orchestrator - handles WhatsApp messages, user validation, intention detection (CONFIRMAR_RUTINA, VER_RUTINA_DE_HOY, CHAT), and routine display |
| `GymRatForm Supabase v2.json` | **Advanced routine generation** - creates personalized 4-week workout plans using full user profile (22 fields) |
| `GymBotWorkoutCompletion.json` | Evening follow-up (8 PM) - tracks workout completion status, prevents duplicate pending_tasks |
| `RoutineMorningReminder.json` | Morning motivation (5 AM) - sends daily workout reminders |
| `GymRatFlow_E2E_TestRunner.json` | Automated E2E test suite - validates all user flows |
| `GymBotWorkoutCompletion_E2E_TestRunner.json` | E2E test suite for workout completion workflow (4 test cases) |

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

## Development Notes

- **No traditional build system**: Workflows are JSON files deployed directly to n8n
- **All workflows are active**: Check `"active": true` in each JSON
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

### 2026-01-28 (Priority-Based Duration Validation - KAN-51)
- **Algoritmo mejorado `ValidateWorkoutDuration` v2.0** en GymRatForm Supabase v2.1.json:
  - **Priorización muscular**: Protege ejercicios que trabajan `priority_muscles_en` del usuario
  - **Sistema de scoring**: Cada ejercicio recibe un puntaje basado en rol + prioridad muscular:
    | Rol | No Prioritario | Prioritario |
    |-----|----------------|-------------|
    | isolation | 0 | 10 |
    | core | 20 | 30 |
    | compound | 40 | 50 |
  - **Fase 1 - Reducción de series**: Reduce sets en ejercicios de menor puntaje primero
  - **Fase 2 - Eliminación**: Si aún excede tiempo, elimina ejercicios (nunca compound prioritarios)
  - **Mínimo dinámico**: 3 sets (semanas 1-3 hipertrofia), 2 sets (semana 4 descarga)
  - **Tiempo de transición**: Actualizado a 120 seg (2 min) para setup de máquinas
  - **Protección absoluta**: Ejercicios compound + músculo prioritario NUNCA se eliminan
  - **Mínimo ejercicios**: Nunca deja menos de 4 ejercicios por día
- **Lookup de ejercicios**: Obtiene `main_muscle` y `secondary_muscles` de `GetExercisesByPattern`
- **Logging mejorado**: Registra acciones de reducción/eliminación con puntajes de prioridad

### 2026-01-27 (Workout Time Validation - KAN-51)
- **Nuevo nodo `ValidateWorkoutDuration` en GymRatForm Supabase v2.json**: Sistema determinístico de validación de tiempo que garantiza que las rutinas diarias no excedan el tiempo disponible del usuario:
  - Cálculo matemático de duración: `tiempo_trabajo (sets × reps × tempo) + tiempo_descanso + warmup (10 min) + transiciones (30 seg/ejercicio)`
  - Parseo de tempo formato "X-Y-Z-W" (ej: "3-0-1-0" = 4 seg/rep)
  - Algoritmo de reducción determinística: Si rutina excede tiempo objetivo, reduce series gradualmente respetando prioridad (isolation > core > compound)
  - Respeta restricción dura: Nunca reduce por debajo de 2 sets por ejercicio
  - Mapeo de `session_duration_mins` a minutos objetivo:
    - "45-60 minutos" → 55 min
    - "60-75 minutos" → 70 min
    - "Más de 75 minutos" → 85 min
- **Flujo actualizado**: `Code in JavaScript1` → `ValidateWorkoutDuration` → `Create a row`
- **Logging detallado**: Cada validación registra duración inicial, final, ajustes realizados y cumplimiento de objetivo
- **Usuario de prueba**: Creado `570000000020` (Test Short Session) con sesión de 45-60 min para testing
- **Beneficios**:
  - Solución 100% determinística (mismo input → mismo output)
  - No depende de AI Agent para cumplir restricciones de tiempo
  - Mejora adherencia al plan (workouts que caben en tiempo disponible del usuario)

### 2026-01-25 (Personalization v2)
- **Nueva versión GymRatForm Supabase v2.json**: Rutinas completamente personalizadas usando los 22 campos de `users_gym_profile`:
  - Nuevo nodo `ProcessUserPreferences`: Transforma preferencias del usuario (español→inglés, mapeo de músculos)
  - Mapeo de músculos: "Glúteo, pierna" → ["Glutes", "Quads", "Hamstrings", "Calfs"]
  - Tier de experiencia: Principiante/Intermedio/Avanzado basado en `training_experience`
  - Modificador de volumen: Ajusta series según `session_duration_mins` (0.85x para sesiones cortas)
  - Restricciones de salud: Códigos A-E mapean a restricciones específicas (ej: C = evitar overhead)
- **System prompt mejorado** (`RoutineCreation.txt`): Reglas de personalización para el AI Agent:
  - Exclusión obligatoria de músculos no deseados
  - Priorización por músculos favoritos (main_muscle o secondary_muscles)
  - Adaptación por sexo biológico (F→glúteos, M→pecho/espalda)
  - Adaptación por experiencia (beginner→máquinas, advanced→barbell)
- **Reorganización de directorio n8n/**:
  - `running_flows/`: Workflows activos en producción
  - `tests/`: E2E test runners
  - `deprecated/`: Versiones anteriores (backup)
  - `system_prompts/`: Prompts de AI agents

### 2026-01-25 (Earlier)
- **Fix duplicate pending_tasks en GymBotWorkoutCompletion**: Reestructurado workflow para prevenir creación de pending_tasks duplicados:
  - Agregado nodo `PendingTasks` (Supabase GET) para consultar pending_tasks existentes
  - Agregado nodo `Merge` con `joinMode: "keepNonMatches"` que actúa como LEFT ANTI JOIN
  - Solo procesa workouts que NO tienen pending_task existente (evita duplicados y mensajes WhatsApp repetidos)
  - `Create_Pending_Task` ahora usa `$('Merge').item.json.*` para datos del workout
- **Nuevo E2E Test Runner para GymBotWorkoutCompletion**: Creado `GymBotWorkoutCompletion_E2E_TestRunner.json` con 4 test cases:
  - TC_WC_001: No crea pending_task duplicado (DUPLICATE_PREVENTION)
  - TC_WC_002: Crea pending_task cuando no existe (TASK_CREATION)
  - TC_WC_003: No procesa usuarios sin workout hoy (FILTER)
  - TC_WC_004: No procesa workouts ya completados (FILTER)
- **Fix timezone en E2E tests**: Cambiado `CURRENT_DATE` a `(NOW() AT TIME ZONE 'America/Bogota')::date` en:
  - `e2e/test_data_setup.sql` - Todas las queries de schedule y pending_tasks
  - `n8n/GymRatFlow_E2E_TestRunner.json` - Queries de cleanup
  - Esto evita el desfase de 5 horas entre UTC (Supabase) y hora local de Colombia
- **Fix validación TC006**: Actualizada regla de validación para soportar nuevo formato de rutina:
  - Antes: Buscaba "RUTINA" o "rutina"
  - Ahora: Busca "plan para hoy", "rutina", o "RUTINA" + "Series", "series", o "Repeticiones"
