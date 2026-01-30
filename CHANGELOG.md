# Changelog

All notable changes to GymBot will be documented in this file.

## [1.7.0] - 2026-01-28

### Added - Priority-Based Duration Validation (KAN-51)

- **Algoritmo mejorado `ValidateWorkoutDuration` v2.0** en GymRatForm Supabase v2.1.json:
  - **Priorización muscular**: Protege ejercicios que trabajan `priority_muscles_en` del usuario
  - **Sistema de scoring**: Cada ejercicio recibe un puntaje basado en rol + prioridad muscular (isolation: 0/10, core: 20/30, compound: 40/50)
  - **Fase 1 - Reducción de series**: Reduce sets en ejercicios de menor puntaje primero
  - **Fase 2 - Eliminación**: Si aún excede tiempo, elimina ejercicios (nunca compound prioritarios)
  - **Mínimo dinámico**: 3 sets (semanas 1-3 hipertrofia), 2 sets (semana 4 descarga)
  - **Tiempo de transición**: Actualizado a 120 seg (2 min) para setup de máquinas
  - **Protección absoluta**: Ejercicios compound + músculo prioritario NUNCA se eliminan
  - **Mínimo ejercicios**: Nunca deja menos de 4 ejercicios por día
- **Lookup de ejercicios**: Obtiene `main_muscle` y `secondary_muscles` de `GetExercisesByPattern`
- **Logging mejorado**: Registra acciones de reducción/eliminación con puntajes de prioridad

## [1.6.0] - 2026-01-27

### Added - Workout Time Validation (KAN-51)

- **Nuevo nodo `ValidateWorkoutDuration`** en GymRatForm Supabase v2.json: Sistema determinístico de validación de tiempo
  - Cálculo matemático de duración: `tiempo_trabajo (sets × reps × tempo) + tiempo_descanso + warmup (10 min) + transiciones (30 seg/ejercicio)`
  - Parseo de tempo formato "X-Y-Z-W" (ej: "3-0-1-0" = 4 seg/rep)
  - Algoritmo de reducción determinística: reduce series respetando prioridad (isolation > core > compound)
  - Nunca reduce por debajo de 2 sets por ejercicio
  - Mapeo de `session_duration_mins`: "45-60 min" → 55 min, "60-75 min" → 70 min, "75+ min" → 85 min
- **Flujo actualizado**: `Code in JavaScript1` → `ValidateWorkoutDuration` → `Create a row`
- **Usuario de prueba**: Creado `570000000020` (Test Short Session) con sesión de 45-60 min

## [1.5.0] - 2026-01-25

### Added - Personalization v2

- **Nueva versión GymRatForm Supabase v2.json**: Rutinas completamente personalizadas usando 22 campos de `users_gym_profile`
- **Nuevo nodo `ProcessUserPreferences`**: Transforma preferencias del usuario (español→inglés, mapeo de músculos)
  - Mapeo de músculos: "Glúteo, pierna" → ["Glutes", "Quads", "Hamstrings", "Calfs"]
  - Tier de experiencia basado en `training_experience`
  - Modificador de volumen según `session_duration_mins` (0.85x para sesiones cortas)
  - Restricciones de salud: Códigos A-E mapean a restricciones específicas
- **System prompt mejorado** (`RoutineCreation.txt`): Reglas de personalización para el AI Agent

### Fixed

- **Fix duplicate pending_tasks en GymBotWorkoutCompletion**: Agregado nodo `Merge` con `keepNonMatches` como LEFT ANTI JOIN
- **Fix timezone en E2E tests**: Cambiado `CURRENT_DATE` a `(NOW() AT TIME ZONE 'America/Bogota')::date`
- **Fix validación TC006**: Actualizada regla para soportar nuevo formato de rutina

### Added - E2E Testing

- **Nuevo E2E Test Runner para GymBotWorkoutCompletion**: 4 test cases (TC_WC_001-004)

### Changed

- **Reorganización de directorio n8n/**: `running_flows/`, `tests/`, `deprecated/`, `system_prompts/`

## [1.4.0] - 2026-01-24

### Added - E2E Test Suite v4.0

- **TC002_FULL_KYC test**: AI-simulated user completes entire KYC flow using GPT-4o-mini
- **DB verification as ground truth**: TC002_FULL_KYC validates user creation via database queries instead of turn tracking
- **Automatic cleanup queries**: TC002, TC002_FULL_KYC, and TC003 now auto-clean data before each run

### Fixed

- **TC002 cleanup**: Added missing cleanup queries to delete user 570000000009 data before test
- **TC003 cleanup**: Added cleanup to delete future scheduled workouts preventing false positives
- **whatsapp_id data type**: Fixed cleanup queries using string instead of numeric for BIGINT column
- **test_data_setup.sql FK error**: Added 570000000009 to all DELETE subqueries to prevent foreign key constraint violations

### Changed

- **Test cases embedded in workflow**: Removed external `GymRatFlow_test_cases.json`, tests now defined in `GymRatFlow_E2E_TestRunner.json`
- **Deprecated tests removed**: Removed TC005, TC008, TC009, TC010 (obsolete confirmation flow tests)
- **Documentation consolidated**: Removed `how_to_make.md`, updated `GymRatFlow_test_plan.md` with complete execution guide

### Documentation

- Updated `GymRatFlow_test_plan.md` to v4.0 with step-by-step execution guide, architecture diagram, and troubleshooting section

## [1.3.2] - 2026-01-22
https://gym-rat.atlassian.net/browse/KAN-49
Automated tests with "TestRunner"


## [1.3.1] - 2026-01-22
https://gym-rat.atlassian.net/browse/KAN-49
Added e2e tests for gym rat flow

## [1.3.0] - 2026-01-21

### Fixed - Workout Confirmation Flow

- **Fixed routine completion update**: Changed `Tool_Update_User_Weekly_Schedule1` query to use `user_id` + `planned_day` instead of `$fromAI("day_routine_id")` which caused the LLM to invent random IDs
- **Simplified CONFIRMATION AGENT prompt**: Removed dependency on remembering `day_routine_id` from memory

### Changed

- **Memory session key**: Changed from `year` to `weekNumber` for confirmation memory in both `GymRatFlow_Supabase` and `GymBotWorkoutCompletion` workflows
- **Memory context window**: Reduced `contextWindowLength` from 50 to 10 for confirmation memory in both workflows

## [1.2.0] - 2026-01-21

### Added - GymRatForm Supabase Workflow

- **WhatsApp notification on routine creation**: Added `NotifyRoutineCreated` node that sends a confirmation message to the user via WhatsApp after their 4-week workout plan has been generated
  - Displays personalized summary with user's name, goal, fitness level, and days per week
  - Connected after `Create a row` node to trigger once all workouts are saved
  - Uses `String()` conversion for `whatsapp_id` to ensure compatibility with WhatsApp API

## [1.1.0] - 2026-01-21

### Fixed - GymRatForm Supabase Workflow

- **Connected GetUserProfile to workflow**: Added missing connection from `GetUserProfile` node to `LoadProfile` node
- **Updated data references**: Replaced all `$('FORM')` references with `$('GetUserProfile')` to use KYC profile data instead of form input
  - LoadProfile filters (goal, level)
  - Get_Day_Requirements query (days_available, primary_goal, fitness_level)
  - AI Agent prompt (full_name, primary_goal, fitness_level, days_available, priority_muscles)
  - CreateUser fields (full_name, email, whatsapp_id)
  - GetUser filter (email)
- **Fixed UserExists condition**: Changed from array `notEmpty` check to checking if `user_id` exists, preventing false positives when Supabase returns empty results
- **Added full_phone_number to CreateUser**: Ensures new users can be found by phone number in other workflows (GymRatFlow)
- **Added alwaysOutputData flag to GetUser**: Ensures the node outputs data even when no user is found, allowing the flow to continue to CreateUser branch

### Changed

- Workflow now triggers via `whatsapp_id` input instead of form submission
- User data sourced from `users_gym_profile` table (populated by KYC agent)
