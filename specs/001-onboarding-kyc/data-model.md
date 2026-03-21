# Data Model: Onboarding KYC

**Feature**: 001-onboarding-kyc | **Date**: 2026-03-15

## Entities

### 1. KYCState (LangGraph Graph State)

In-memory state managed by the LangGraph checkpointer. Persisted across
invocations via `thread_id`. This is NOT a database table — it's the
`TypedDict` that flows through the graph nodes.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `messages` | `Annotated[list[BaseMessage], operator.add]` | `[]` | Full conversation history (reducer: append) |
| `phone_number` | `str` | `""` | User's WhatsApp phone (e.g., `"573001234567"`) |
| `display_name` | `str` | `""` | WhatsApp profile display name |
| `is_new_user` | `bool` | `False` | Result of `check_user` Supabase lookup |
| `current_turn` | `int` | `0` | Current KYC turn (1-5), 0 = not started |
| `collected_data` | `dict` | `{}` | Accumulated KYC fields: `{field_name: value}` |
| `is_complete` | `bool` | `False` | All 10 fields collected |
| `health_code` | `str` | `""` | Classified health code: A, B, C, D, or E |
| `affected_zones` | `list[str]` | `[]` | Body zones affected (e.g., `["rodilla"]`) |
| `awaiting_confirmation` | `bool` | `False` | Profile summary shown, waiting for accept/reject |
| `profile_confirmed` | `bool` | `False` | User accepted the profile summary |
| `needs_correction` | `bool` | `False` | User wants to correct a field |
| `correction_field` | `str` | `""` | Which field the user wants to fix |
| `response` | `str` | `""` | Latest Kairos response text |
| `route_to_trainer` | `bool` | `False` | Health code E → recommend human trainer |
| `nudge_sent` | `bool` | `False` | Inactivity nudge already sent for this session |
| `last_interaction_at` | `str` | `""` | ISO timestamp of last user message |

**Validation rules**:
- `current_turn` must be 0-5
- `health_code` must be one of: `""`, `A`, `B`, `C`, `D`, `E`
- `collected_data` keys must be from the 10 KYC fields set

### 2. User Profile (Supabase: `users`)

Existing table. The KYC flow creates a new row when a first-time user
completes onboarding.

| Field | Type | Constraint | KYC Source |
|-------|------|------------|------------|
| `user_id` | UUID | PK, auto-generated | Generated on save |
| `full_name` | text | NOT NULL | `display_name` from WhatsApp |
| `cel_number` | bigint | NOT NULL | Extracted from `phone_number` |
| `full_phone_number` | text | NOT NULL, UNIQUE | `phone_number` as-is |
| `email` | text | | Not collected in KYC |
| `timezone` | text | NOT NULL | Default: `America/Bogota` |
| `created_at` | timestamptz | NOT NULL | `NOW()` on insert |
| `country_indicative` | bigint | NOT NULL | Default: `57` |

**State transition**: Row created by `save_profile` node after profile confirmation.

### 3. Gym Profile (Supabase: `users_gym_profile`)

Existing table. Stores the 10 KYC data points plus derived fields.

| Field | Type | Enum? | KYC Turn | `collected_data` Key |
|-------|------|-------|----------|---------------------|
| `whatsapp_id` | bigint | — | — | `phone_number` (PK) |
| `full_name` | text | — | — | `display_name` |
| `submission_date` | timestamptz | — | — | `NOW()` |
| `primary_goal` | `goal` enum | Yes | 1 | `primary_goal` |
| `training_experience` | `gym_experience` enum | Yes | 2 | `training_experience` |
| `days_available` | integer | — | 2 | `days_available` |
| `preferred_schedule` | `usual_schedule` enum | Yes | 2 | `preferred_schedule` |
| `training_environment` | text | — | 3 | `training_environment` |
| `home_equipment` | text | — | 3 | `home_equipment` |
| `biological_sex` | `sex` enum | Yes | 4 | `biological_sex` |
| `age` | integer | — | 4 | `age` |
| `height_cm` | numeric | — | 4 | `height_cm` |
| `weight_kg` | numeric | — | 4 | `weight_kg` |
| `health_status` | text | — | 5 | `health_code` (classified) |

**Enum constraints** (from CLAUDE.md):
- `primary_goal`: `Ganar masa muscular`, `Bajar grasa`, `Mejorar fuerza`, `Mejorar resistencia`, `Salud general / recomposición corporal`
- `training_experience`: `Nunca he entrenado`, `Menos de 6 meses`, `6 a 12 meses`, `1 a 3 años`, `Más de 3 años`
- `preferred_schedule`: `Mañana`, `Tarde`, `Noche`
- `biological_sex`: `M`, `F`

**Fields NOT collected in KYC** (FR-020 — out of scope):
`secondary_goal`, `priority_muscles`, `disliked_exercises`, `cardio_type`,
`cardio_frequency`, `training_style`, `current_frequency`, `fitness_level`,
`session_duration_mins`. These default to NULL or empty on insert.

### 4. Health Condition Record (Derived, NOT a separate table)

The health classification is stored as fields in `users_gym_profile` and
`KYCState`. No separate table is needed.

| Storage | Field | Value |
|---------|-------|-------|
| `users_gym_profile.health_status` | text | `A`, `B`, `C`, `D`, or `E` |
| `KYCState.health_code` | str | Same as above |
| `KYCState.affected_zones` | list[str] | `["rodilla"]`, `["hombro", "codo"]`, etc. |

**Health code definitions** (from constitution):

| Code | Description | Restriction |
|------|-------------|-------------|
| A | Sin restricciones | Full exercise pool |
| B | Problemas tren inferior | Avoid high-impact lower body |
| C | Problemas tren superior | Avoid overhead pressing |
| D | Problemas columna | Avoid heavy axial loading |
| E | Condición severa | Route to human trainer |

### 5. KYC Session (Checkpointer State)

The KYC session is implicitly managed by the LangGraph checkpointer.
Each session is identified by `thread_id` in the format:
`kyc_{phone_number}` (e.g., `kyc_573001234567`).

| Aspect | Value |
|--------|-------|
| Storage | `InMemorySaver` (dev) / `PostgresSaver` (prod) |
| Key | `thread_id` = `kyc_{phone_number}` |
| Expiry | 7 days (FR-019) — cleanup via background task |
| State | Full `KYCState` TypedDict serialized by checkpointer |

**State transitions**:

```
NOT_STARTED → IN_PROGRESS → AWAITING_CONFIRMATION → CONFIRMED → SAVED
                  ↑                    ↓
                  └── CORRECTION ──────┘
```

- `NOT_STARTED`: No thread exists for this phone
- `IN_PROGRESS`: `current_turn` between 1-5, `is_complete = False`
- `AWAITING_CONFIRMATION`: `awaiting_confirmation = True`
- `CORRECTION`: `needs_correction = True` (loops back to IN_PROGRESS)
- `CONFIRMED`: `profile_confirmed = True`
- `SAVED`: Profile persisted to Supabase, session can be cleaned up

## Relationships

```
KYCState (graph state)
    │
    │ phone_number ─────────────► users.full_phone_number (lookup)
    │ collected_data ───────────► users_gym_profile fields (on save)
    │ health_code ──────────────► users_gym_profile.health_status
    │ display_name ─────────────► users.full_name
    │
    ▼
users ◄──────────────────────── users_gym_profile
  (user_id)                    (whatsapp_id ≈ cel_number)
```

## Field Mapping: collected_data → Supabase

When `save_profile` executes, it maps `KYCState.collected_data` to Supabase columns:

| `collected_data` Key | Target Column | Transform |
|---------------------|---------------|-----------|
| `primary_goal` | `users_gym_profile.primary_goal` | Map to enum value |
| `training_experience` | `users_gym_profile.training_experience` | Map to enum value |
| `days_available` | `users_gym_profile.days_available` | `int()` |
| `preferred_schedule` | `users_gym_profile.preferred_schedule` | Map to enum value |
| `training_environment` | `users_gym_profile.training_environment` | `"GYM"` or `"HOME"` |
| `home_equipment` | `users_gym_profile.home_equipment` | Free text or NULL |
| `biological_sex` | `users_gym_profile.biological_sex` | `"M"` or `"F"` |
| `age` | `users_gym_profile.age` | `int()` |
| `height_cm` | `users_gym_profile.height_cm` | `float()` |
| `weight_kg` | `users_gym_profile.weight_kg` | `float()` |

The `health_code` is stored separately (from `KYCState.health_code`, not from
`collected_data`) after classification.
