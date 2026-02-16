# GymBot — Kairos Personal Trainer

An AI-powered fitness coaching platform that delivers personalized workout plans and daily accountability through WhatsApp. Built almost entirely with [Claude Code](https://claude.ai/code).

## What is GymBot?

GymBot (branded as **Kairos Personal Trainer**) is a conversational fitness coach that lives in WhatsApp. It onboards users through a natural conversation, generates personalized 4-week training programs (mesocycles), sends daily reminders, tracks workout completion, and adapts over time — all in Spanish, targeting a Colombian audience.

### Key capabilities

- **Conversational onboarding**: A KYC agent collects 22+ data points (goals, experience, equipment, health conditions, schedule) through natural WhatsApp conversation
- **Personalized routine generation**: AI creates 4-week mesocycles adapted to the user's goal, level, available equipment, health restrictions, and muscle priorities
- **Daily reminders**: 5 AM workout reminders with full exercise details (sets, reps, RIR, tempo, rest)
- **Completion tracking**: 8 PM follow-ups for accountability, with confirmation flow
- **Workout Tracker web app**: A companion web app where users log weights and reps per set, with rest timers between sets
- **Mesocycle renewal**: Automatic detection of program completion with options to maintain, rotate, or adjust the next cycle
- **GYM + HOME support**: Full programs for gym users and home-based training with minimal equipment

## Tech Stack

| Layer | Technology |
|-------|-----------|
| **Workflow orchestration** | [n8n](https://n8n.io) (self-hosted) |
| **AI models** | OpenAI GPT-5.x, Google Gemini 2.0-flash |
| **Database** | [Supabase](https://supabase.com) (PostgreSQL) |
| **Messaging** | WhatsApp Business API |
| **Frontend** | React 19 + TypeScript + Vite + Tailwind CSS |
| **Backend API** | Go + Gin (hexagonal architecture) |
| **Frontend hosting** | Firebase Hosting |
| **Backend hosting** | Google Cloud Run |
| **Project management** | Jira (Kanban) |

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
├── spec/                      # Feature specifications
└── docs/                      # Deployment guides and feature docs
```

### Multi-agent workflow system

The platform is orchestrated through n8n workflows, each with specialized AI agents:

| Workflow | Purpose |
|----------|---------|
| **MAIN_FLOW** | Main orchestrator — handles incoming WhatsApp messages, user validation, intention detection, scheduling, and routine display |
| **WORKOUT_CREATOR** | Generates personalized 4-week workout plans using full user profile (22 fields) with duration and volume validation |
| **MorningReminder-WorkoutTracker** | Daily workout reminders (5 AM) and completion tracking (8 PM) |
| **GymBotMesocycleRenewal** | Handles 4-week mesocycle renewal conversation |

### Data flow

```
User (WhatsApp) → MAIN_FLOW → Intention Detection
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
              KYC Onboarding   View Routine    Confirm Workout
                    │               │               │
                    ▼               │               ▼
            WORKOUT_CREATOR         │       Mark Completed
                    │               │
                    ▼               ▼
             4-week plan      Formatted routine
             saved to DB      sent via WhatsApp
```

## Built with Claude Code

This project has been developed almost entirely using **Claude Code** — Anthropic's CLI tool for AI-assisted software engineering. This includes:

- All n8n workflow JSON files (nodes, connections, system prompts, tool configurations)
- The full React + TypeScript frontend (workout tracker web app)
- The Go backend with hexagonal architecture
- Database schema design and migrations
- E2E test infrastructure (test runners, fixtures, multi-turn test execution)
- Feature specifications and documentation
- CI/CD configuration and deployment scripts

Claude Code served as the primary development tool for design, implementation, debugging, and iteration across the entire stack.

## Language

All user-facing content is in Spanish, targeting a Colombian audience (timezone: `America/Bogota`).

## Setup

1. Import the JSON workflows from `n8n/running_flows/` into your n8n instance
2. Configure credentials: OpenAI API, Google Gemini API, Supabase, WhatsApp Business API
3. Run the database setup scripts in Supabase
4. Deploy the frontend (`workout-tracker/`) to Firebase Hosting
5. Deploy the backend (`workout-tracker-back/`) to Google Cloud Run
6. Activate the workflows

See [docs/deployment-guide.md](docs/deployment-guide.md) for detailed instructions.

## License

Private project. All rights reserved.
