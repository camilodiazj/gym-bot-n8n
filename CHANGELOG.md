# Changelog

All notable changes to GymBot will be documented in this file.

## [0.2.0] - 2026-02-08

### Added - Mesocycle Renewal (v5.0)

- **Nuevo subflow `GymBotMesocycleRenewal.json`**: Maneja la conversación multi-turno de renovación de mesociclo (4 semanas)
  - **Path A (Auto-detección)**: Cuando el usuario completa todas las sesiones de W4, el sistema detecta automáticamente y ofrece opciones de renovación
  - **Path B (Manual)**: El usuario puede escribir "renovar" o similar → intención `RENOVAR_MESOCICLO` detectada por el Intention Agent
  - **Opciones de renovación**: Mantener rutina actual (MANTENER_RUTINA), cambiar días, rotar ejercicios
  - **Renewal Agent**: Nuevo AI agent especializado en la conversación de renovación (en español)

### Changed - MAIN_FLOW

- **Detección de mesociclo completo**: Nuevos nodos en el branch FALSE de `has_planned_workouts` para verificar si W4 está completa
  - Query `Week_Schedule` + `User_Finished_Workouts` + `Template_Days` → `Check_Mesocycle_Complete`
  - Si mesociclo completo → ejecuta `GymBotMesocycleRenewal` subflow
- **Nueva intención `RENOVAR_MESOCICLO`**: Agregada al Intention Agent para activación manual del flujo de renovación

### Changed - WORKOUT_CREATOR

- **Nodo `If_Skip_Create_For_Renewal`**: Previene creación duplicada de plan cuando se ejecuta desde renovación de mesociclo
- **Soporte para `mesocycle_number`**: Incremento automático del contador de mesociclo en `users_plans`

### Added - E2E Tests para Mesocycle Renewal

- **`MesocycleRenewal_E2E_TestRunner.json`**: 3 test cases de renovación
  - TC_MESO_001: Auto-detección de W4 completada activa renovación
  - TC_MESO_002: Flujo MANTENER_RUTINA completa exitosamente
  - TC_MESO_003: Intención manual RENOVAR_MESOCICLO detectada correctamente
- **Usuarios de prueba** (`5700000005X`): 3 fixtures pre-poblados en `test_data_setup.sql`

### Added - Especificaciones de Arquitectura

- **`spec/Mesocycle_Renewal/`**: Documentación completa del feature
  - `00_ARCHITECTURE.md`: Diagrama de sistema, contratos de interfaz, inventario de nodos
  - `01_DOMAIN_LOGIC.md`: Reglas de negocio, criterios de detección, lógica de renovación
  - `02_IMPLEMENTATION_PLAN.md`: Plan de implementación paso a paso

### Changed - Reorganización del Repositorio

- **Eliminado `n8n/wip/`**: Workflows WIP movidos o eliminados
- **Eliminado `n8n/system_prompts/`**: Prompts ahora embebidos directamente en los workflows
- **Nuevo `n8n/archived/`**: Versiones archivadas de workflows
- **Eliminado `GymBotWorkoutCompletion_E2E_TestRunner.json`**: Tests de completion consolidados
- **Actualizado CLAUDE.md**: Refleja estructura actual del proyecto

## [1.8.0] - 2026-02-03

### Fixed - Band Exercise Equipment Classification

- **Migración de 82 ejercicios de banda**: Corregido `equipment` de `machine` a `resistance_band`
  - Framework de clasificación basado en "Test de Remoción" (kiro-coach)
  - Reglas deterministas por patrón de `exercise_id`:
    - R1: `ex_band_*` → resistance_band (71 ejercicios)
    - R2: `*_resisted*` → resistance_band (8 ejercicios)
    - R3: `*_band_*` (cardio) → resistance_band (4 ejercicios)
  - Fallback por nombre español: `spanish_name ILIKE '%con banda%'`
  - **Híbridos preservados**: 5 ejercicios `ex_barbell_banded_*` mantienen `barbell` (accommodating resistance)
  - **Corrección especial**: `ex_band_goblet_squat` → `dumbbell` (ID incorrecto, nombre indica mancuerna)

### Fixed - WORKOUT_CREATOR Band Equipment Filter

- **Bug crítico corregido en `ProcessUserPreferences`**: Bandas elásticas no se incluían en el filtro de equipment
  - **Antes**: `"bandas"` solo seteaba `flags.has_bands = true` pero NO agregaba al `equipmentSet`
  - **Después**: Ahora también ejecuta `equipmentSet.add('resistance_band')`
  - **Resultado**: HOME users con bandas ahora reciben ejercicios con `equipment = 'resistance_band'`

### Added - E2E Test Infrastructure for HOME Users

- **Nuevos test cases HOME** (TC_HOME_001, TC_HOME_002, TC_HOME_003) con usuarios MULTI_TURN_AI
- **Teardown actualizado** en `test_data_setup.sql`: Incluye phones HOME (570000000211-213) para re-ejecución limpia
- **Documentación HOME_TEST_CASES.md**: Guía de validación para rutinas HOME

### Impact

- **HOME users con bandas elásticas** ahora reciben ejercicios de banda correctamente filtrados
- Query `WHERE equipment = 'resistance_band'` ahora devuelve 82 ejercicios
- Ejemplo María (570000000211): 48 resistance_band + 26 dumbbell + 16 bodyweight exercises

## [0.1.0] - 2026-01-31

### Added - Workout Tracker Integration

- **Workout Tracker Web App**: Nueva aplicación web para que los usuarios vean y registren sus rutinas diarias
  - Frontend: React 19 + TypeScript + Vite + Tailwind CSS (Firebase Hosting)
  - Backend: Go + Gin con arquitectura hexagonal (Google Cloud Run)
  - Magic links para autenticación sin contraseña
  - Seguimiento de sets completados con peso y repeticiones

### Added - Exercise Ordering System

- **Nuevo campo `exercise_order`** en tabla `workouts`: Garantiza orden determinístico de ejercicios
  - Orden: compound (1-4) → core (5-6) → isolation (7+)
  - Backend Go: `ORDER BY exercise_order` en consultas
  - Workflow n8n: Ordenamiento programático por `role` antes de insertar
- **Migración automática**: Ejercicios existentes actualizados con orden correcto basado en `role`

### Fixed

- **Soporte para en-dash en reps**: Funciones `parseReps` y `parseRepsRange` ahora soportan tanto guión corto (-) como en-dash (–)
- **Tests unitarios**: Agregados tests para funciones de parsing de reps

### Changed

- **Workflows renombrados**: `GymRatFlow_Supabase_V2.json` → `GymRatFlow_Supabase_V2_Workout_Tracker.json`
- **Limpieza de archivos deprecados**: Eliminados workflows obsoletos de `n8n/deprecated/`

## [1.7.0] - 2026-01-28

### Added - Priority-Based Duration Validation (KAN-51)

- **Algoritmo mejorado `ValidateWorkoutDuration` v2.0** en WORKOUT_CREATOR.json:
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

- **Nuevo nodo `ValidateWorkoutDuration`** en WORKOUT_CREATOR.json: Sistema determinístico de validación de tiempo
  - Cálculo matemático de duración: `tiempo_trabajo (sets × reps × tempo) + tiempo_descanso + warmup (10 min) + transiciones (30 seg/ejercicio)`
  - Parseo de tempo formato "X-Y-Z-W" (ej: "3-0-1-0" = 4 seg/rep)
  - Algoritmo de reducción determinística: reduce series respetando prioridad (isolation > core > compound)
  - Nunca reduce por debajo de 2 sets por ejercicio
  - Mapeo de `session_duration_mins`: "45-60 min" → 55 min, "60-75 min" → 70 min, "75+ min" → 85 min
- **Flujo actualizado**: `Code in JavaScript1` → `ValidateWorkoutDuration` → `Create a row`
- **Usuario de prueba**: Creado `570000000020` (Test Short Session) con sesión de 45-60 min

## [1.5.0] - 2026-01-25

### Added - Personalization v2

- **Nueva versión WORKOUT_CREATOR.json**: Rutinas completamente personalizadas usando 22 campos de `users_gym_profile`
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
