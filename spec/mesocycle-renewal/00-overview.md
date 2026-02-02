# Mesocycle Renewal Feature - Technical Specification

## Executive Summary

This specification documents the implementation of a mesocycle renewal system for GymBot. The feature enables users to renew their 4-week training plan after completion, with options to maintain, modify, or completely regenerate their routine.

## Business Requirements

1. **Sub-workflow** to help users create their next month of training
2. **Flexibility** for changes: days available, session duration, injuries, exercises, priorities
3. **User feedback** integration + option to maintain current structure
4. **Deterministic logic in backend** (Go) for consistency and testability
5. **AI agents only for user interaction** (n8n)

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         WhatsApp Message                             │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│              GymRatFlow (Main Orchestrator - n8n)                    │
│  ┌─────────────────┐    ┌──────────────────┐    ┌────────────────┐ │
│  │ Intention Agent │───►│ RENOVAR_MESOCICLO│───►│ Execute Renewal│ │
│  │   (AI - GPT)    │    │     Intent       │    │   Sub-workflow │ │
│  └─────────────────┘    └──────────────────┘    └────────────────┘ │
│                                                          │          │
│  ┌─────────────────────────────────────────────────────┐│          │
│  │ Auto-Detection: Week 4 Complete? ──────────────────►││          │
│  │   HTTP GET /api/v1/plans/:userId/mesocycle-status   ││          │
│  └─────────────────────────────────────────────────────┘│          │
└─────────────────────────────────────────────────────────│──────────┘
                                                          │
                                                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│            GymBotMesocycleRenewal (Sub-workflow - n8n)               │
│  ┌─────────────────┐                                                │
│  │ Renewal Agent   │◄─── User Conversation (AI)                     │
│  │  (AI - GPT)     │                                                │
│  └────────┬────────┘                                                │
│           │                                                          │
│  ┌────────▼────────────────────────────────────────────────────┐   │
│  │                    Switch on Intention                       │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐│   │
│  │  │MANTENER  │ │CAMBIAR   │ │ROTAR     │ │MODIFICAR_PERFIL  ││   │
│  │  │_RUTINA   │ │_DIAS     │ │_EJERC.   │ │                  ││   │
│  │  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────────┬─────────┘│   │
│  └───────│────────────│────────────│────────────────│──────────┘   │
│          │            │            │                │               │
└──────────│────────────│────────────│────────────────│───────────────┘
           │            │            │                │
           ▼            ▼            ▼                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Go Backend (Deterministic Logic)                  │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ POST /api/v1/plans/:userId/renew/maintain                       ││
│  │ POST /api/v1/plans/:userId/renew/change-days                    ││
│  │ POST /api/v1/plans/:userId/renew/rotate-exercises               ││
│  │ POST /api/v1/plans/:userId/renew/update-profile                 ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                   │                                  │
│                                   ▼                                  │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                     PostgreSQL (Supabase)                       ││
│  │  users_plans | user_weekly_schedule | workouts | exercises      ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
```

## Renewal Options

| Option | Description | Backend Operation | n8n Operation |
|--------|-------------|-------------------|---------------|
| **MANTENER_RUTINA** | Keep same exercises with load progression | Clear schedule, increment mesocycle | Notify user |
| **CAMBIAR_DIAS** | Change training frequency (2-6 days) | Delete workouts, update week_schedule | Call GymRatForm |
| **ROTAR_EJERCICIOS** | New exercises, same movement patterns | Rotate exercises deterministically | Notify user |
| **MODIFICAR_PERFIL** | Update priorities, injuries, duration | Update profile, delete workouts | Collect data, call GymRatForm |

## Specification Documents

| Document | Description |
|----------|-------------|
| [01-backend-api.md](./01-backend-api.md) | Go backend implementation (entities, repositories, services, handlers) |
| [02-n8n-workflows.md](./02-n8n-workflows.md) | n8n workflow modifications (main flow, renewal subflow) |
| [03-system-prompts.md](./03-system-prompts.md) | AI agent system prompts (Spanish) |
| [04-database.md](./04-database.md) | SQL queries and database operations |
| [05-testing.md](./05-testing.md) | Test plan, fixtures, and E2E scenarios |
| [06-implementation-phases.md](./06-implementation-phases.md) | Phase-by-phase implementation tasks |

## Key Technical Decisions

### 1. Backend vs n8n Responsibility Split

**Backend (Go) handles:**
- Mesocycle completion detection (SQL query)
- Exercise rotation algorithm (deterministic with seeded random)
- All CRUD operations on plans, workouts, schedules
- Preference processing (Spanish → English muscle mapping)
- Health restriction filtering

**n8n handles:**
- User conversation (AI agents)
- Intention detection (NLP)
- WhatsApp messaging
- Workflow orchestration
- Routine regeneration (calls GymRatForm)

### 2. Exercise Rotation Algorithm

Deterministic selection based on:
1. Same `pattern` (e.g., push_h, pull_v)
2. Same `role` (compound, isolation, core)
3. Exclude exercises where `main_muscle` is in user's `disliked_exercises`
4. Apply health restrictions (e.g., no overhead for health_status=C)
5. Prioritize exercises matching `priority_muscles`
6. Random selection from top candidates (seeded for reproducibility)

### 3. Mesocycle Progression

Based on sports science principles:
- **MANTENER_RUTINA**: 2.5-5% load increase on compounds
- **Periodization rotation**: Change set/rep schemes every 2-3 mesocycles
- **Plateau detection**: If <1% progress after 4 mesocycles, suggest ROTAR_EJERCICIOS

### 4. Database Operations

All operations use transactions with proper rollback:
- Mesocycle increment is atomic with schedule clear
- Exercise rotation is a single batch UPDATE
- Profile updates cascade to workout deletion

## Dependencies

| Component | Dependency | Purpose |
|-----------|------------|---------|
| Backend API | Internal API Key | Authentication for n8n calls |
| n8n → Backend | HTTP Request nodes | API calls for all operations |
| n8n → GymRatForm | Execute Workflow | Routine regeneration |
| AI Agents | OpenAI GPT-4 | Conversation handling |
| Memory | Postgres Chat Memory | Conversation persistence |

## Success Criteria

1. [ ] User completing week 4 automatically sees renewal options
2. [ ] User can manually request renewal with "quiero renovar"
3. [ ] All 4 renewal paths complete successfully
4. [ ] mesocycle_number increments correctly
5. [ ] user_weekly_schedule clears on renewal
6. [ ] Exercise rotation respects user preferences
7. [ ] WhatsApp notifications sent for all outcomes
8. [ ] E2E tests pass for all scenarios

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| No alternative exercises found | Medium | Keep current exercise, log warning |
| GymRatForm timeout | High | Increase timeout to 120s, add retry logic |
| User abandons mid-conversation | Low | Memory cleanup on next interaction |
| Backend API unavailable | High | n8n error handling with user-friendly message |

## Timeline Estimate

| Phase | Duration | Description |
|-------|----------|-------------|
| Phase 1 | 3-4 days | Backend domain layer |
| Phase 2 | 2-3 days | Backend application layer |
| Phase 3 | 3-4 days | Backend adapter layer |
| Phase 4 | 2-3 days | n8n main flow integration |
| Phase 5 | 3-4 days | n8n renewal subflow |
| Phase 6 | 2-3 days | Testing & QA |
| Phase 7 | 1-2 days | Deployment & monitoring |
| **Total** | **16-23 days** | ~3-4 weeks |

## Authors

- Planning agents: Claude Code (Opus 4.5)
- Training science: kiro-coach agent
- Date: 2026-02-01
