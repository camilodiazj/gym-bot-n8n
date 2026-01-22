# Changelog

All notable changes to GymBot will be documented in this file.

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
