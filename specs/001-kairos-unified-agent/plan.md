# Implementation Plan: Agente Unificado Kairos

**Branch**: `001-kairos-unified-agent` | **Date**: 2026-03-17 | **Spec**: [spec.md](./spec.md)

---

## Summary

Migrar el MAIN_FLOW de n8n a LangGraph implementando un agente conversacional unificado (Kairos) que reemplaza el routing rígido por switch de intenciones con un ReAct agent que decide libremente qué hacer. La arquitectura combina un nodo determinístico de carga de contexto, un router liviano, el KYC subgraph existente (Case 5 reutilizado sin cambios), y un agente Gemini con 11 herramientas Supabase. Los usuarios reciben respuestas contextualizadas, pueden crear rutinas interactivamente con feedback, y el sistema maneja escenarios complejos (sesiones perdidas, tareas pendientes, renovación de mesociclo) que el n8n actual no soporta.

---

## Technical Context

**Language/Version**: Python 3.11+
**Primary Dependencies**: LangGraph >=0.4, langchain-google-genai >=2.1, langchain-core >=0.3, FastAPI >=0.115, httpx (Supabase PostgREST client)
**Storage**: Supabase (PostgreSQL) via PostgREST REST API — misma infraestructura que Case 5
**Testing**: pytest 8.0 + pytest-asyncio 0.24 — mismo setup que tests existentes
**Target Platform**: Linux server (mismo despliegue que el servicio FastAPI actual en puerto 8000)
**Project Type**: AI agent service (LangGraph StateGraph + FastAPI endpoints)
**Performance Goals**: Respuesta al usuario en <5 segundos incluyendo tool calls a Supabase
**Constraints**: Respuestas máximo 3-4 oraciones (canal WhatsApp). InMemorySaver (misma estrategia que Case 4/5 — sin Redis por ahora).
**Scale/Scope**: Mismos usuarios de GymBot (~decenas de usuarios activos). Un thread por número de teléfono.

---

## Constitution Check

La constitución del proyecto no está configurada (`constitution.md` contiene sólo el template). Se aplican los principios observados en el código existente (Cases 1-5):

| Principio | Status |
|-----------|--------|
| Reutilizar Case 5 sin modificaciones | ✅ KYC subgraph se importa como está |
| Separación `graph.py` (mock) / `graph_live.py` (Supabase) | ✅ Misma convención que todos los casos anteriores |
| `supabase_client.py` compartido en `src/shared/` | ✅ Se extiende con `supabase_update()` y `supabase_bulk_insert()` |
| Tests en `tests/test_case6.py` | ✅ Misma estructura que test_case5.py |
| FastAPI endpoints en `server.py` | ✅ Se agregan rutas `/case6/*` al servidor existente |

**No hay violaciones.** No se crea ningún nuevo proyecto ni dependencia externa.

---

## Project Structure

### Documentation (this feature)

```text
specs/001-kairos-unified-agent/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 — tool design decisions
├── data-model.md        # Phase 1 — state schema + entities
├── contracts/
│   └── api.md           # Phase 1 — POST /case6/chat endpoint contract
└── tasks.md             # Phase 2 — /speckit.tasks output
```

### Source Code (repository root)

```text
langgraph-skeleton/
├── cases/
│   ├── case5_onboarding_kyc/        # SIN CAMBIOS — reutilizado como subgraph
│   └── case6_unified_agent/         # NUEVO
│       ├── __init__.py
│       ├── state.py                 # UnifiedAgentState, UserContext, DraftRoutine
│       ├── nodes.py                 # load_context, router, kairos_agent, format_response
│       ├── prompts.py               # KAIROS_SYSTEM_PROMPT + format_user_context()
│       ├── tools.py                 # 11 herramientas Supabase (@tool decorators)
│       ├── context_loader.py        # Queries paralelas de Supabase para UserContext
│       ├── graph.py                 # Graph sin KYC + tools mock (desarrollo/tests)
│       └── graph_live.py            # Graph completo: Supabase + KYC subgraph
│
├── src/shared/
│   └── supabase_client.py           # MODIFICAR: agregar supabase_update() y supabase_bulk_insert()
│
├── tests/
│   └── test_case6.py                # NUEVO — tests unitarios e integración
│
└── server.py                        # MODIFICAR: agregar POST /case6/chat, GET /case6/history
```

**Structure Decision**: Single project, misma estructura que todos los cases anteriores. `case6_unified_agent/` sigue exactamente el mismo patrón que `case5_onboarding_kyc/`.

---

## Phase 0: Research

Ver [research.md](./research.md) para decisiones completas.

### Decisión principal: herramientas granulares vs. monolíticas

**Decisión**: Herramientas granulares separadas. El borrador NO es retornado por una tool — vive en el estado de LangGraph como campo `draft_routine`.

**Rationale**:
- El agente llama herramientas incrementalmente (patrón ya establecido en Case 3: `get_exercises_by_pattern` por patrón, `get_set_profile` por rol)
- Mensajes de herramienta pequeños = mejor razonamiento del LLM
- `interrupt()` funciona limpiamente en boundary de nodos, no dentro de tools
- `swap_exercise` y `find_exercises` reutilizan la misma lógica de búsqueda

**Alternativa rechazada**: Una tool `draft_routine()` monolítica que devuelve el plan completo. Rechazada porque colapsa la orquestación, impide recovery granular, y complica el flujo de modificaciones.

### Decisión: `DraftRoutine` en estado vs. base de datos temporal

**Decisión**: El borrador vive en `state.draft_routine: DraftRoutine | None` (in-memory en el thread de LangGraph).

**Rationale**: El borrador no necesita persistencia entre sesiones — si el usuario abandona, se recrea. InMemorySaver ya persiste el estado del thread dentro de una sesión. Guardar en DB solo ocurre al confirmar (`save_workout_plan`).

### Decisión: Carga de contexto paralela vs. secuencial

**Decisión**: Queries paralelas usando `asyncio.gather()` en `load_context`.

**Rationale**: Las 5 queries de contexto (users, users_plans, user_weekly_schedule, pending_tasks, missed_sessions) son independientes entre sí. En paralelo reducen el tiempo de respuesta de ~500ms a ~150ms.

---

## Phase 1: Design

### State Schema

```python
# cases/case6_unified_agent/state.py

class UserContext(TypedDict):
    user_id: str | None
    full_name: str
    phone_number: str
    plan: dict | None                   # {plan_id, goal, level, week_schedule, mesocycle_number, status, current_week}
    todays_sessions: list[dict]         # [{session_id, session_name, week, Completed, planned_day}]
    missed_sessions: list[dict]         # Últimos 3 días, Completed=false
    next_scheduled_session: dict | None # {session_name, planned_day}
    pending_tasks: list[dict]           # [{task_id, task_type, session_name, status}]
    is_new_user: bool
    kyc_complete: bool
    has_schedule: bool
    all_w4_completed: bool

class DraftExercise(TypedDict):
    exercise_id: str
    spanish_name: str
    pattern: str
    role: str                           # compound | isolation | core
    sets: int
    reps: str                           # "8-10"
    rir: str                            # "1-2"
    rest_seconds: int
    exercise_order: int

class DraftDay(TypedDict):
    day_number: int
    title: str                          # "Full Body A", "Upper Body B"
    exercises: list[DraftExercise]

class DraftRoutine(TypedDict):
    week_schedule: str                  # "fb_3", "ul_4"
    goal: str
    level: str
    days: list[DraftDay]

class UnifiedAgentState(TypedDict):
    messages: Annotated[list[BaseMessage], operator.add]
    phone_number: str
    display_name: str
    user_context: UserContext           # Poblado por load_context, inmutable durante el turno
    draft_routine: DraftRoutine | None  # Borrador en construcción (no persiste aún)
    response: str                       # Output final al usuario
```

### Graph Architecture

```
START
  │
  ▼
load_context          ← Determinístico: 5 queries Supabase en paralelo (asyncio.gather)
  │                     Popula user_context: user, plan, todays_sessions, missed_sessions,
  │                     next_scheduled_session, pending_tasks, flags booleanos
  ▼
router                ← Determinístico: is_new_user && !kyc_complete → KYC | else → agent
  │             │
  ▼             ▼
KYC subgraph   kairos_agent  ← Gemini + 11 tools bound via bind_tools()
(Case 5)        │
  │             ▼
  │         tool_node         ← ToolNode(TOOLS) ejecuta tool calls del LLM
  │             │
  │             └─ loop back to kairos_agent until no tool_calls
  │
  ▼             ▼
             END
```

### 11 Tools del Agente

```python
# cases/case6_unified_agent/tools.py

# ── Operación del día ──────────────────────────────────────
@tool  get_todays_routine(user_id: str, session_name: str, week: int) -> str
  → SELECT workouts JOIN exercises WHERE user_id AND week AND day_name
  → ORDER BY exercise_order
  → Returns: rutina formateada con ejercicios, sets, reps, RIR, descanso, link video

@tool  confirm_workout_completion(user_id: str, session_date: str | None = None) -> str
  → UPDATE user_weekly_schedule SET "Completed"=true WHERE user_id AND planned_day
  → UPDATE pending_tasks SET status='completed' WHERE user_id AND task_type='CONFIRMAR_RUTINA'
  → Grace period: acepta hoy O ayer (últimas 24h + 1 día)

@tool  decline_workout(user_id: str) -> str
  → UPDATE pending_tasks SET status='declined' WHERE user_id AND status='pending'

@tool  create_magic_link(user_id: str) -> str
  → INSERT INTO magic_links (code, user_id, expires_at = NOW()+48h)
  → Returns: URL completa del Workout Tracker

# ── Agendamiento ───────────────────────────────────────────
@tool  get_schedule_info(user_id: str) -> str
  → SELECT users_plans JOIN week_schedules JOIN template_days WHERE user_id
  → Returns: {days_per_week, sessions: [{day_number, title}], current_week}

@tool  schedule_sessions(user_id: str, sessions_json: str) -> str
  → INSERT INTO user_weekly_schedule (bulk, una fila por sesión)
  → sessions_json: [{week_day, session_name, planned_day: "DD/MM"}]

# ── Estado del mesociclo ───────────────────────────────────
@tool  get_mesocycle_status(user_id: str) -> str
  → SELECT users_plans + COUNT completadas semana 4
  → Returns: {week4_completed, week4_total, can_renew, mesocycle_number}

# ── Construcción de rutina (Draft Mode) ────────────────────
@tool  get_day_requirements(week_schedule: str) -> str
  → SELECT routine_templates JOIN template_days JOIN day_requirements
  → Returns: [{template_day_id, day_number, title, patterns: [{pattern, min_sets, priority}]}]

@tool  get_exercises_for_draft(
         pattern: str,
         level: str,
         equipment: str | None = None,
         exclude_muscle: str | None = None,
         limit: int = 5
       ) -> str
  → SELECT exercises WHERE pattern AND level AND equipment (optional)
  → Returns: list of candidates para que el LLM seleccione

@tool  find_exercise_alternatives(
         pattern: str,
         level: str,
         exclude_name: str | None = None,
         equipment: str | None = None
       ) -> str
  → SELECT exercises WHERE pattern AND level (para swaps y alternativas)
  → Returns: list de alternativas

@tool  save_workout_plan(user_id: str, draft_json: str) -> str
  → INSERT users_plans (plan_id, user_id, template_id, goal, level, week_schedule, start_date)
  → INSERT workouts (bulk: 4 semanas × ejercicios por día)
  → draft_json: DraftRoutine serializado
  → Returns: {plan_id, workouts_created: N}
```

**Nota**: El borrador se construye incrementalmente en el estado (`draft_routine`) llamando `get_day_requirements` → `get_exercises_for_draft` × N → `get_set_profile` (de `src/shared`). El LLM ensambla el `DraftRoutine` en el estado. Solo `save_workout_plan` escribe en la base de datos.

### Context Loader

```python
# cases/case6_unified_agent/context_loader.py

async def load_user_context(phone_number: str) -> UserContext:
    """Carga el contexto completo del usuario en paralelo (5 queries)."""

    user_result, kyc_result = await asyncio.gather(
        supabase_query("users", filters={"full_phone_number": f"eq.{phone_number}"}),
        supabase_query("users_gym_profile", filters={"whatsapp_id": f"eq.{phone_number}"}),
    )

    if not user_result:
        return UserContext(is_new_user=True, kyc_complete=False, ...)

    user = user_result[0]
    user_id = user["user_id"]

    plan_result, schedule_result, tasks_result = await asyncio.gather(
        supabase_query("users_plans", filters={"user_id": f"eq.{user_id}", "status": "eq.active"}),
        supabase_query("user_weekly_schedule", filters={"user_id": f"eq.{user_id}", ...}),
        supabase_query("pending_tasks", filters={"user_id": f"eq.{user_id}", "status": "eq.pending"}),
    )

    # Calcular missed_sessions: schedule rows con Completed=false y planned_day en últimos 3 días
    # Calcular next_scheduled_session: primera sesión futura
    # Calcular all_w4_completed: todas las sesiones de semana 4 completadas

    return UserContext(...)
```

### Supabase Client Extensions

```python
# src/shared/supabase_client.py — NUEVAS FUNCIONES

async def supabase_update(
    table: str,
    data: dict,
    filters: dict[str, str],
) -> list[dict]:
    """PATCH a Supabase table via PostgREST (UPDATE with WHERE)."""
    # PATCH /rest/v1/{table}?col=eq.val
    # Prefer: return=representation

async def supabase_bulk_insert(
    table: str,
    rows: list[dict],
) -> list[dict]:
    """Bulk INSERT — envía lista de filas en un solo POST."""
    # POST /rest/v1/{table}
    # Body: list of dicts (PostgREST acepta arrays directamente)
```

---

## Implementation Phases

### Phase 1: Foundation — Context Loader + Agent Shell (MVP)

**Goal**: Agente responde mensajes de usuarios existentes con contexto real.

**Files**:
- `cases/case6_unified_agent/__init__.py`
- `cases/case6_unified_agent/state.py` — `UnifiedAgentState`, `UserContext`, `DraftRoutine`
- `cases/case6_unified_agent/context_loader.py` — queries paralelas Supabase
- `cases/case6_unified_agent/prompts.py` — `KAIROS_SYSTEM_PROMPT`, `format_user_context()`
- `cases/case6_unified_agent/nodes.py` — `load_context`, `router`, `kairos_agent` (stub sin tools)
- `cases/case6_unified_agent/graph.py` — graph básico sin KYC ni tools
- `server.py` — agregar `POST /case6/chat`, `GET /case6/history`

**Test criteria**:
- Usuario existente recibe respuesta contextualizada (nombre, sesión de hoy, objetivo)
- Usuario nuevo recibe mensaje genérico de bienvenida (router aún no conectado a KYC)
- `/case6/history` retorna historial del thread por `phone_number`

---

### Phase 2: Tools para usuarios existentes

**Goal**: El agente puede ver rutinas, confirmar, agendar, generar magic links.

**Files**:
- `cases/case6_unified_agent/tools.py` — 7 tools operacionales (excluye draft mode)
- `src/shared/supabase_client.py` — agregar `supabase_update()` y `supabase_bulk_insert()`
- `cases/case6_unified_agent/nodes.py` — `kairos_agent` con `bind_tools()` + `ToolNode` loop
- `cases/case6_unified_agent/graph_live.py` — graph con tools reales (sin KYC aún)

**Test criteria**:
- `get_todays_routine`: retorna rutina correcta con ejercicios reales de Supabase
- `confirm_workout_completion`: marca sesión como completada + resuelve pending task
- `create_magic_link`: inserta en `magic_links`, retorna URL válida
- `schedule_sessions`: crea filas en `user_weekly_schedule`
- Pending task: agente pregunta por tarea antes de responder mensaje principal
- Sesión perdida: agente ofrece sesión de días anteriores cuando no hay sesión hoy

---

### Phase 3: KYC Subgraph + Creación de Rutina

**Goal**: Usuario nuevo completa KYC y puede crear su rutina en modo borrador.

**Files**:
- `cases/case6_unified_agent/graph_live.py` — integrar KYC subgraph (Case 5)
- `cases/case6_unified_agent/nodes.py` — state mapping `UnifiedAgentState ↔ KYCState`
- `cases/case6_unified_agent/tools.py` — agregar 4 tools de draft mode: `get_day_requirements`, `get_exercises_for_draft`, `find_exercise_alternatives`, `save_workout_plan`

**Test criteria**:
- Usuario nuevo: router envía a KYC subgraph, completa 5 turnos, perfil guardado en Supabase
- Siguiente mensaje post-KYC: router envía al agent mode (no repite KYC)
- Draft mode: agente pregunta preferencia (todo junto vs. día a día)
- Swap de ejercicio: agente encuentra alternativa y actualiza borrador
- `save_workout_plan`: crea `users_plans` + bulk insert `workouts` (4 semanas)

---

### Phase 4: Renovación de mesociclo + Polish

**Goal**: Detección automática de W4 completada, renovación, tests E2E completos.

**Files**:
- `cases/case6_unified_agent/tools.py` — `get_mesocycle_status` ya existe, validar lógica de renovación
- `tests/test_case6.py` — suite completa de tests
- `server.py` — agregar `GET /case6/kyc/status` si aplica

**Test criteria**:
- `all_w4_completed=True` en contexto → agente detecta y ofrece renovación
- Renovación con cambio de días: agente informa cambio de `week_schedule`
- Todos los tests de fases 1-3 siguen pasando (no regresiones)
- Tests E2E: flujo completo usuario nuevo → KYC → borrador → agendar → confirmar

---

## Quickstart para el siguiente desarrollador

```bash
# Clonar y setup
cd langgraph-skeleton
python -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"

# Variables de entorno necesarias
cp .env.example .env
# Completar: GOOGLE_API_KEY, SUPABASE_URL, SUPABASE_ANON_KEY

# Levantar servidor
uvicorn server:app --reload --port 8000

# Test básico — usuario existente (usar phone de fixture e2e)
curl -X POST http://localhost:8000/case6/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "que me toca hoy?", "phone_number": "570000000003"}'

# Test usuario nuevo → KYC
curl -X POST http://localhost:8000/case6/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "hola quiero empezar", "phone_number": "579000000001"}'

# Ver historial de conversación
curl http://localhost:8000/case6/history?phone_number=570000000003

# Correr tests
pytest tests/test_case6.py -v
```

---

## API Contract

Ver [contracts/api.md](./contracts/api.md) para especificación completa.

**Endpoints nuevos en `server.py`**:

```
POST /case6/chat
  Body: { "message": str, "phone_number": str, "display_name": str? }
  Response: { "response": str, "thread_id": str, "user_context": UserContext }

GET  /case6/history?phone_number={phone}
  Response: { "thread_id": str, "messages": [{"role": str, "content": str}] }
```

El `thread_id` es `f"case6_{phone_number}"` — mismo patrón que Case 5.

---

## Complexity Tracking

No hay violaciones a los principios del proyecto. Todo se construye sobre infraestructura existente.

| Adición | Justificación |
|---------|---------------|
| `context_loader.py` separado de `nodes.py` | Las 5 queries paralelas son suficientemente complejas para merecer su propio módulo; facilita testing independiente |
| `DraftRoutine` en estado (no en DB temporal) | Evita crear una tabla de borradores; el InMemorySaver ya persiste el estado del thread dentro de la sesión |
