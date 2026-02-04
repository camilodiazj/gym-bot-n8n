# GymBot

AI-powered fitness coaching bot that delivers personalized workout plans and daily accountability through WhatsApp.

## Tech Stack

- **Automation**: n8n workflows
- **AI Models**: OpenAI GPT-5.x, Google Gemini 2.0-flash
- **Database**: Supabase (PostgreSQL)
- **Messaging**: WhatsApp Business API

## Workflows

| File | Description |
|------|-------------|
| `MAIN_FLOW.json` | Main orchestrator - handles incoming WhatsApp messages, user validation, and routine display |
| `WORKOUT_CREATOR.json` | Generates personalized 4-week workout plans based on user profiles |
| `GymBotWorkoutCompletion.json` | Evening follow-up (8 PM) - tracks workout completion |
| `RoutineMorningReminder (2).json` | Morning motivation (5 AM) - sends daily workout reminders |

## Features

- Conversational onboarding via WhatsApp
- Personalized routine generation based on fitness level and goals
- Daily workout reminders with exercise details
- Completion tracking and accountability follow-ups
- Multi-agent architecture with specialized AI agents for different tasks

## Setup

1. Import the JSON workflows into your n8n instance
2. Configure credentials:
   - OpenAI API
   - Google Gemini API
   - Supabase (database connection)
   - WhatsApp Business API
3. Set up the required database tables in Supabase
4. Activate the workflows

## Language

All user-facing content is in Spanish, targeting a Colombian audience (timezone: America/Bogota).
