# Changelog

All notable changes to GymBot will be documented in this file.

## [1.4.0] - 2026-01-22

### Added - Mesocycle Renewal Feature

- **New subflow `GymBotMesocycleRenewal.json`**: Handles end-of-mesocycle (4-week cycle) options
  - `Renewal_Agent`: AI agent that guides users through renewal options
  - Option 1 - **Mantener rutina**: Keeps same exercises, resets schedule for new cycle
  - Option 2a - **Cambiar días**: Regenerates routine with different days per week
  - Option 2b - **Rotar ejercicios**: Selects alternative exercises by movement pattern
  - Postgres-based conversation memory with automatic cleanup

- **Mesocycle detection in `GymRatFlow_Supabase.json`**:
  - `Check_Mesocycle_Complete` node: Detects when user has completed all week 4 sessions
  - `If_Mesocycle_Complete` node: Routes to renewal subflow when mesocycle is done
  - New intention `RENOVAR_MESOCICLO` added to `Intention_Agent`
  - New Switch output for manual renewal requests

- **Renewal support in `GymRatForm Supabase.json`**:
  - New input parameters: `is_renewal`, `override_days_available`
  - `If_Is_Renewal` node: Skips user/plan creation for renewals
  - `Clear_Old_Workouts` node: Cleans existing workouts before regenerating

- **Database migration**: Added columns to `users_plans` table
  - `mesocycle_number` (INTEGER, default 1): Tracks current mesocycle count
  - `last_renewal_date` (TIMESTAMP): Records last renewal date

- **Documentation**: Created `docs/MESOCYCLE_RENEWAL.md` with flow diagrams, SQL queries, and testing scenarios

### Changed

- Updated `CLAUDE.md` with new workflow documentation and database schema changes

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
