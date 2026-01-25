# Changelog

All notable changes to GymBot will be documented in this file.

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
