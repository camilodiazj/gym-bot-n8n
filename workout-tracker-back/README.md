# Workout Tracker Backend

Go + Gin backend API with hexagonal architecture for the GymBot workout tracking system.

## Architecture

This project follows **Hexagonal Architecture** (Ports and Adapters) with SOLID principles:

```
workout-tracker-back/
├── cmd/api/                    # Application entry point
├── internal/
│   ├── domain/                 # Core business logic (no external deps)
│   │   ├── entity/             # Domain entities
│   │   ├── repository/         # Port interfaces (abstractions)
│   │   └── service/            # Domain services
│   ├── application/            # Application layer
│   │   ├── dto/                # Data Transfer Objects
│   │   └── usecase/            # Use case implementations
│   ├── adapter/                # External adapters
│   │   ├── http/               # Primary adapter (Gin handlers)
│   │   └── repository/         # Secondary adapter (PostgreSQL)
│   └── config/                 # Configuration
└── pkg/                        # Shared utilities
    ├── apperror/               # Custom error types
    └── response/               # Standardized API responses
```

### SOLID Principles

| Principle | Implementation |
|-----------|----------------|
| **S**ingle Responsibility | Each handler, use case, and repository has one job |
| **O**pen/Closed | Interfaces allow extension without modification |
| **L**iskov Substitution | Repository implementations can be swapped |
| **I**nterface Segregation | Small, focused interfaces (Reader, Writer) |
| **D**ependency Inversion | Domain depends on abstractions, not concrete implementations |

## Prerequisites

- Go 1.22+
- Supabase account (PostgreSQL database)

## Setup

1. **Clone and navigate to the directory:**
   ```bash
   cd workout-tracker-back
   ```

2. **Install dependencies:**
   ```bash
   make deps
   # or
   go mod download && go mod tidy
   ```

3. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your Supabase credentials
   ```

4. **Run the server:**
   ```bash
   make run
   # or
   go run ./cmd/api
   ```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/workouts/today?user_id=xxx` | Get today's workout |
| POST | `/api/v1/workouts/:workoutId/complete` | Complete a workout |
| PATCH | `/api/v1/sets/:setId` | Update set reps/weight |
| PATCH | `/api/v1/sets/:setId/complete` | Mark set as completed |

### Example Responses

**GET /api/v1/workouts/today**
```json
{
  "success": true,
  "data": {
    "session_id": "uuid",
    "session_name": "Piernas",
    "week": 1,
    "day_name": "Monday",
    "exercises": [
      {
        "id": "uuid",
        "name": "DB Front Squat",
        "badgeColor": "#22C55E",
        "rir": "3-4",
        "sets": [
          {"id": "uuid-1", "setNumber": 1, "reps": 12, "kg": "-", "completed": false}
        ],
        "tips": [{"text": "Mantén la espalda recta"}],
        "steps": [{"text": "Posición inicial: pies al ancho de hombros"}]
      }
    ]
  }
}
```

**Health Check**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-29T10:00:00Z",
  "uptime": "1h30m45s"
}
```

## Development

```bash
# Build binary
make build

# Run tests
make test

# Run with hot reload (requires air)
make dev

# Format code
make fmt

# Run linter (requires golangci-lint)
make lint
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `GIN_MODE` | Gin mode (debug/release/test) | `debug` |
| `SUPABASE_DB_URL` | PostgreSQL connection URL | Required |

## Database Tables

This API interacts with the following Supabase tables:

- `users` - User identity
- `workouts` - Exercise assignments
- `exercises` - Exercise catalog
- `user_weekly_schedule` - Session tracking
- `users_plans` - Active training plans

## License

Private - GymBot Project
