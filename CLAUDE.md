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

The system consists of 4 n8n workflows in the `/n8n/` directory:

| Workflow | Purpose |
|----------|---------|
| `GymRatFlow_Supabase.json` | Main orchestrator - handles WhatsApp messages, user validation, intention detection (CONFIRMAR_RUTINA, VER_RUTINA_DE_HOY, CHAT), and routine display |
| `GymRatForm Supabase.json` | Routine generation engine - creates personalized 4-week workout plans based on user profiles |
| `GymBotWorkoutCompletion.json` | Evening follow-up (8 PM) - tracks workout completion status |
| `RoutineMorningReminder (2).json` | Morning motivation (5 AM) - sends daily workout reminders |

### Data Flow Patterns

1. **User Onboarding**: WhatsApp → KYC Agent → Form submission → Profile creation → Routine generation
2. **Daily Routine**: User message → Intention detection → Routine retrieval → Formatted WhatsApp delivery
3. **Completion Tracking**: 8 PM trigger → Query uncompleted workouts → AI follow-up → Status update

### Multi-Agent Architecture

Each workflow uses specialized AI agents with Spanish system prompts:
- **KYC Agent**: Collects user profile information
- **Intention Agent**: Classifies user messages
- **Confirmation Agent**: Handles schedule confirmations
- **Workout Display Agent**: Formats and presents routines

Agents use Postgres-based chat memory for conversation context persistence.

## Database Schema

Key tables (Supabase/PostgreSQL):
- `users` - User identity (user_id, full_name, email, full_phone_number, timezone)
- `users_gym_profile` - Profile data from KYC form
- `user_weekly_schedule` - Planned workouts (planned_day, session_name, week, Completed, day_routine_id)
- `workouts` - Assigned exercises (user_id, week, day_name, exercise_id, sets, reps, rir, rest_seconds, tempo)
- `exercises` - Exercise library (exercise_id, spanish_name, main_muscle, level, pattern, role, link)
- `routine_templates` / `template_days` - Templates by level/goal
- `set_profiles` - Loading parameters by goal/level/role/week

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
