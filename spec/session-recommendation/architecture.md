# KAN-104: Session Recommendation in GymBot KYC

## Overview

Add a deterministic session-recommendation tool to the KYC Agent so it can suggest training days/week based on collected profile data. The recommendation happens mid-KYC (after duration is collected, before schedule preference), giving the user a data-driven suggestion they can accept or override.

## Data Flow

```
KYC Agent (LangChain AI Agent node in MAIN_FLOW.json)
│
├── Phase 1: Name (email removed from here)
├── Phase 2: Age, sex, height, weight
├── Phase 3: Goal, experience, current frequency
├── Phase 4: Health, training style, priority/disliked muscles
├── Phase 5a: Ask session duration
├── Phase 5b: Call Tool_Session_Recommendation($fromAI inputs)
│             ↓
│   ┌─────────────────────────────────────┐
│   │  Code Node (tool mode)              │
│   │  Inputs: edad, experiencia,         │
│   │    objetivo_principal,              │
│   │    frecuencia_actual,               │
│   │    estado_de_salud, duracion_sesion  │
│   │  Output: { recommended_days: N }    │
│   └─────────────────────────────────────┘
│             ↓
│   Agent presents recommendation naturally
│   User accepts or overrides (days_available set)
│
├── Phase 5c: Ask preferred schedule (Manana/Tarde/Noche)
├── Phase 6: Cardio type + frequency
├── Phase 7: Training environment + equipment (HOME only)
├── Phase 8: Ask email ("para enviarte tu rutina")
├── Finalization: Trigger on "Email confirmado"
│             ↓
└── Tool_Create_User_Profile (existing, ai_tool index 0, unchanged)
```

## New Node: Tool_Session_Recommendation

### Node Configuration

| Property | Value |
|----------|-------|
| Type | `n8n-nodes-base.code` |
| Mode | Tool (LangChain tool mode) |
| Connection | `ai_tool` to KYC Agent, index 1 |
| Language | JavaScript |

### $fromAI Inputs

| Parameter | Type | Values |
|-----------|------|--------|
| `edad` | number | User's age |
| `experiencia` | string | `"Nunca he entrenado"`, `"Menos de 6 meses"`, `"6 a 12 meses"`, `"1 a 3 anos"`, `"Mas de 3 anos"` |
| `objetivo_principal` | string | `"Ganar masa muscular"`, `"Bajar grasa"`, `"Mejorar fuerza"`, `"Mejorar resistencia"`, `"Salud general / recomposicion corporal"` |
| `frecuencia_actual` | string | `"No entreno"`, `"1-2 dias por semana"`, `"3-4 dias por semana"`, `"5-6 dias por semana"` |
| `estado_de_salud` | string | `A`, `B`, `C`, `D`, `E` |
| `duracion_sesion` | string | `"30-45 minutos"`, `"45-60 minutos"`, `"60-75 minutos"`, `"Mas de 75 minutos"` |

### Output

```json
{ "recommended_days": 3 }
```

Single integer. The agent is responsible for presenting this to the user in natural Spanish and handling accept/override.

### Deterministic Logic

```javascript
// ── Base: experience -> base days (simplified from matrix) ──
const experienceBase = {
  'Nunca he entrenado': 2,
  'Menos de 6 meses': 3,
  '6 a 12 meses': 3,
  '1 a 3 años': 4,
  'Más de 3 años': 5,
};
let days = experienceBase[experience] ?? 3;

// ── Goal modifier ──
if (goal === 'Ganar masa muscular' || goal === 'Mejorar fuerza') days += 1;

// ── Current frequency modifier (readiness signal) ──
if (frequency === 'No entreno') days -= 1;
if (frequency === '5-6 días por semana') days += 1;

// ── Age modifier ──
if (age >= 50) days -= 1;

// ── Health modifiers ──
if (health === 'B' || health === 'C' || health === 'D') days -= 1;
if (health === 'E') days = Math.min(days, 3); // hard cap

// ── Duration modifier ──
if (duration === '30-45 minutos') days += 1;  // shorter sessions = more frequent
if (duration === 'Más de 75 minutos') days -= 1; // longer sessions = fewer days needed

// ── Absolute clamp ──
days = Math.max(2, Math.min(6, days));

return { recommended_days: days };
```

### Logic Rationale

| Rule | Why |
|------|-----|
| Experience base | More experience = higher capacity (simplified linear mapping) |
| Goal +1 for hypertrophy/strength | These goals benefit from higher training volume |
| Frequency "No entreno" -1 | Sedentary users need gradual ramp-up |
| Frequency "5-6" +1 | Already conditioned for high volume |
| Age 50+ -1 | Older adults need more recovery between sessions |
| Health B/C/D -1 | Injuries require more recovery between sessions |
| Health E cap at 3 | Special conditions limit to conservative frequency (hard cap) |
| Duration "30-45 min" +1 | Shorter sessions can accommodate more frequent training |
| Duration "75+ min" -1 | Longer sessions cover more volume per day, fewer days needed |
| Floor at 2, ceiling at 6 | Minimum effective stimulus / maximum supported schedule |

### Output Range

The algorithm produces values in `[2, 6]`. This maps directly to available `week_schedules`:

| Days | Schedule Type | Split |
|------|--------------|-------|
| 2 | `fb_2` | Full Body 2x |
| 3 | `fb_3` | Full Body 3x |
| 4 | `ul_4` | Upper/Lower 4x |
| 5 | `ppl_5` | Push/Pull/Legs 5x |
| 6 | `ppl_6` | Push/Pull/Legs 6x |

## Connection Structure

The KYC Agent node currently has one tool connection at `ai_tool` index 0 (`Tool_Create_User_Profile`). The new tool also connects at index 0 — n8n merges multiple tool connections on the same index.

```json
{
  "Tool_Session_Recommendation": {
    "ai_tool": [
      [
        {
          "node": "KYC Agent",
          "type": "ai_tool",
          "index": 0
        }
      ]
    ]
  }
}
```

Both tools connect to `ai_tool` index 0 — n8n merges multiple tool connections on the same index. `Tool_Create_User_Profile` uses `$fromAI` by parameter name (not positional), so adding a second tool does not affect it.

## KYC Agent System Prompt Changes

### Phase restructuring

| Phase | Before | After |
|-------|--------|-------|
| 1 | Name + email | Name only |
| 2-4 | Unchanged | Unchanged |
| 5 | Days available + schedule | Split into 5a/5b/5c (see below) |
| 6-7 | Unchanged | Unchanged |
| 8 (new) | N/A | Email with justification |
| Finalization | Trigger on "Cardio" | Trigger on "Email confirmado" |

### Phase 5 detail

**5a** - Ask session duration. Wait for answer.

**5b** - Call `Tool_Session_Recommendation` with all 6 collected parameters. Present result naturally in Spanish, e.g.:

> "Basado en tu perfil, te recomiendo entrenar **3 dias por semana**. Que te parece? Si prefieres mas o menos dias, dime."

User accepts (days_available = recommended) or overrides with their preference.

**5c** - Ask preferred schedule (Manana/Tarde/Noche).

### Phase 8 detail

Ask email last, with justification:

> "Por ultimo, necesito tu correo electronico para enviarte tu rutina de entrenamiento. Cual es?"

Trigger finalization on email confirmation instead of cardio completion.

## Scope Boundary: What Does NOT Change

| Component | Status |
|-----------|--------|
| `Tool_Create_User_Profile` | Unchanged (same $fromAI params, same Supabase insert) |
| `WORKOUT_CREATOR.json` | Unchanged (receives `days_available` from `users_gym_profile` as before) |
| Database schema | No new tables or columns |
| `users_gym_profile` columns | Same columns, same enums |
| Other workflows | No impact (MorningReminder, MesocycleRenewal, WeeklyScheduling, DailyReport) |
| E2E tests | Existing tests unaffected; new test cases may be added separately |

## Testing Strategy

### Manual validation

1. Verify edge cases in the deterministic logic:
   - Beginner (18yo, no experience, no training, health A, 30-45 min) -> 2 days
   - Advanced (25yo, 3+ years, 5-6 days, fat loss, health A, 60-75 min) -> 5 days (capped)
   - Senior (65yo, 1-3 years, health D, 45-60 min) -> 2 days (floor)
   - Health E override (any high value) -> capped at 3

2. Verify agent behavior:
   - Agent calls tool at correct phase (after duration, before schedule)
   - Agent presents recommendation in natural Spanish
   - User can accept or override
   - Override value flows correctly to `Tool_Create_User_Profile`

### E2E test cases (future)

| Test | Scenario | Expected |
|------|----------|----------|
| TC_REC_001 | Beginner accepts recommendation | days_available = recommended_days |
| TC_REC_002 | User overrides recommendation | days_available = user's choice |
| TC_REC_003 | Health E user | recommended_days <= 3 |
