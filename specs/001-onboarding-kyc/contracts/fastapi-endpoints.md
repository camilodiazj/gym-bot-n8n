# API Contracts: Onboarding KYC

**Feature**: 001-onboarding-kyc | **Date**: 2026-03-15
**Server**: `langgraph-skeleton/server.py` (FastAPI + Uvicorn)

## Endpoints

### POST /case5/kyc/chat

Send a user message to the KYC conversation. Uses `thread_id` for session persistence.

**Request**:
```json
{
  "message": "Hola, quiero empezar mi rutina",
  "phone_number": "573001234567",
  "display_name": "Camilo",
  "thread_id": "kyc_573001234567"
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `message` | string | Yes | — | User's WhatsApp message text |
| `phone_number` | string | Yes | — | User's full phone number |
| `display_name` | string | No | `""` | WhatsApp display name |
| `thread_id` | string | No | `"kyc_{phone_number}"` | Session ID for checkpointer |

**Response** (200):
```json
{
  "case": "5 — Onboarding KYC",
  "thread_id": "kyc_573001234567",
  "kairos_response": "¡Hola Camilo! Soy Kairos, tu entrenador personal. Pregunta 1 de 5: ¿Cuál es tu objetivo principal?",
  "current_turn": 1,
  "is_complete": false,
  "awaiting_confirmation": false,
  "collected_fields": ["primary_goal"],
  "route_to_trainer": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `case` | string | Endpoint identifier |
| `thread_id` | string | Session ID used |
| `kairos_response` | string | Kairos' response message |
| `current_turn` | int | Current turn (1-5), 0 if not started |
| `is_complete` | bool | All 10 fields collected |
| `awaiting_confirmation` | bool | Profile summary shown, waiting for response |
| `collected_fields` | list[str] | Fields collected so far |
| `route_to_trainer` | bool | Health code E detected |
| `health_code` | string | (only when Turn 5 done) A-E classification |
| `profile_saved` | bool | (only after confirmation) Profile written to Supabase |

**Error responses**:
- `400`: Missing `message` or `phone_number`
- `500`: LLM or Supabase failure

---

### GET /case5/kyc/history

View the full conversation history for a KYC session.

**Query parameters**:

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `thread_id` | string | `"kyc_test"` | Session ID to retrieve |

**Response** (200):
```json
{
  "thread_id": "kyc_573001234567",
  "total_messages": 6,
  "current_turn": 3,
  "collected_data": {
    "primary_goal": "Ganar masa muscular",
    "training_experience": "1 a 3 años",
    "days_available": 3,
    "preferred_schedule": "Mañana"
  },
  "messages": [
    { "index": 0, "role": "user", "content": "Hola, quiero empezar mi rutina" },
    { "index": 1, "role": "kairos", "content": "¡Hola Camilo! Soy Kairos..." },
    { "index": 2, "role": "user", "content": "Quiero ganar masa muscular" },
    { "index": 3, "role": "kairos", "content": "Pregunta 2 de 5: ¿Cuánta experiencia..." }
  ]
}
```

**Empty thread**:
```json
{
  "thread_id": "nonexistent",
  "messages": [],
  "note": "Thread vacío o no existe"
}
```

---

### GET /case5/kyc/status

Check the status of a KYC session without sending a message.

**Query parameters**:

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `phone_number` | string | — | User's phone number |

**Response** (200):
```json
{
  "phone_number": "573001234567",
  "status": "in_progress",
  "current_turn": 3,
  "collected_fields": ["primary_goal", "training_experience", "days_available", "preferred_schedule"],
  "remaining_fields": ["training_environment", "home_equipment", "biological_sex", "age", "height_cm", "weight_kg", "health_status"],
  "last_interaction_at": "2026-03-15T14:30:00Z",
  "nudge_sent": false
}
```

Status values: `not_started`, `in_progress`, `awaiting_confirmation`, `completed`, `expired`

---

## Pydantic Models

```python
class KYCChatRequest(BaseModel):
    message: str = Field(description="User message text")
    phone_number: str = Field(description="Full phone number (e.g., 573001234567)")
    display_name: str = Field(default="", description="WhatsApp display name")
    thread_id: str = Field(default="", description="Session ID (auto-generated if empty)")

class KYCChatResponse(BaseModel):
    case: str = "5 — Onboarding KYC"
    thread_id: str
    kairos_response: str
    current_turn: int
    is_complete: bool
    awaiting_confirmation: bool
    collected_fields: list[str]
    route_to_trainer: bool
    health_code: str = ""
    profile_saved: bool = False

class KYCHistoryResponse(BaseModel):
    thread_id: str
    total_messages: int
    current_turn: int
    collected_data: dict
    messages: list[dict]
```

## Integration Notes

- Endpoints follow the existing pattern: Cases 1-4 use `/case{N}/...`.
- The `phone_number` is used both as the Supabase lookup key and to generate
  the default `thread_id` (`kyc_{phone_number}`).
- For WhatsApp integration (future), the WhatsApp webhook handler will call
  `POST /case5/kyc/chat` with the incoming message payload.
- The `display_name` parameter is optional — if empty, Kairos uses a generic greeting.
