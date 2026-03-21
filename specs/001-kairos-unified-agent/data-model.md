# Data Model: Agente Unificado Kairos

**Feature**: `001-kairos-unified-agent`
**Date**: 2026-03-17

---

## Entidades del Estado de LangGraph

Estas entidades viven en el estado del graph — no son tablas nuevas en la base de datos.

### `UserContext`

Snapshot inmutable del estado del usuario cargado al inicio de cada turno.

| Campo | Tipo | Fuente (tabla Supabase) | Notas |
|-------|------|-------------------------|-------|
| `user_id` | `str \| None` | `users.user_id` | None si es usuario nuevo |
| `full_name` | `str` | `users.full_name` | "" si es nuevo |
| `phone_number` | `str` | Input del endpoint | Siempre presente |
| `plan` | `dict \| None` | `users_plans` (WHERE status='active') | `{plan_id, goal, level, week_schedule, mesocycle_number, current_week}` |
| `todays_sessions` | `list[dict]` | `user_weekly_schedule` (WHERE planned_day = hoy) | `[{session_id, session_name, week, Completed}]` |
| `missed_sessions` | `list[dict]` | `user_weekly_schedule` (WHERE planned_day >= hoy-3 AND Completed=false) | Excluye hoy |
| `next_scheduled_session` | `dict \| None` | `user_weekly_schedule` (WHERE planned_day > hoy ORDER BY planned_day LIMIT 1) | `{session_name, planned_day}` |
| `pending_tasks` | `list[dict]` | `pending_tasks` (WHERE status='pending') | `[{task_id, task_type, session_name}]` |
| `is_new_user` | `bool` | Calculado: `user_id is None` | |
| `kyc_complete` | `bool` | `users_gym_profile` (EXISTS WHERE whatsapp_id = phone) | |
| `has_schedule` | `bool` | Calculado: `len(todays_sessions + next_scheduled_session) > 0` | |
| `all_w4_completed` | `bool` | Calculado: todas las sesiones de `week=4` tienen `Completed=true` | Solo True si mesocycle_number >= 1 |

**Reglas de validación**:
- `missed_sessions` no incluye sesiones de hoy (esas van en `todays_sessions`)
- `missed_sessions` ventana: últimos 3 días (`MISSED_SESSIONS_WINDOW_DAYS = 3`)
- `all_w4_completed` es `False` si el plan está en semana < 4

---

### `DraftRoutine`

Borrador en construcción durante la creación interactiva de rutina. Vive en `UnifiedAgentState.draft_routine`. **No se persiste en Supabase hasta que el usuario confirma.**

| Campo | Tipo | Notas |
|-------|------|-------|
| `week_schedule` | `str` | Ej: "fb_3", "ul_4", "ppl_5" |
| `goal` | `str` | Del perfil KYC: "Ganar masa muscular" |
| `level` | `str` | Del perfil KYC: "Intermedio" |
| `days` | `list[DraftDay]` | 2-6 días dependiendo del schedule |

#### `DraftDay`

| Campo | Tipo | Notas |
|-------|------|-------|
| `day_number` | `int` | 1-6 (según template) |
| `title` | `str` | Ej: "Full Body A", "Upper Body B" |
| `exercises` | `list[DraftExercise]` | Ordenados por `exercise_order` |

#### `DraftExercise`

| Campo | Tipo | Fuente | Notas |
|-------|------|--------|-------|
| `exercise_id` | `str` | `exercises.exercise_id` | Ej: "ex_barbell_squat" |
| `spanish_name` | `str` | `exercises.spanish_name` | |
| `pattern` | `str` | `exercises.pattern` | Ej: "squat", "push_h" |
| `role` | `str` | `exercises.role` | compound \| isolation \| core |
| `sets` | `int` | `set_profiles.sets` | |
| `reps` | `str` | `set_profiles.reps` | Ej: "8-10", "12-15" |
| `rir` | `str` | `set_profiles.rir` | Ej: "1-2" |
| `rest_seconds` | `int` | `set_profiles.rest_sec` | |
| `exercise_order` | `int` | compound: 1-4, core: 5-6, isolation: 7+ | |

**Transiciones de estado del borrador**:
```
None
  │ (agente llama get_day_requirements + get_exercises_for_draft)
  ▼
DraftRoutine (incompleto — días siendo construidos)
  │ (agente llama find_exercise_alternatives + swap)
  ▼
DraftRoutine (modificado por feedback del usuario)
  │ (usuario confirma: "sí, guarda")
  ▼
save_workout_plan() → persiste en users_plans + workouts
  │
  ▼
None (draft_routine limpiado del estado)
```

---

### `UnifiedAgentState`

Estado completo del graph para un turno de conversación.

| Campo | Tipo | Reducer | Notas |
|-------|------|---------|-------|
| `messages` | `list[BaseMessage]` | `operator.add` (append) | Historial completo del thread |
| `phone_number` | `str` | last-write | Inmutable durante el thread |
| `display_name` | `str` | last-write | Del endpoint o del perfil |
| `user_context` | `UserContext` | last-write | Recargado en cada turno |
| `draft_routine` | `DraftRoutine \| None` | last-write | Persiste entre turnos del mismo thread |
| `response` | `str` | last-write | Último mensaje del agente al usuario |

---

## Tablas Supabase Existentes Utilizadas

No se crean tablas nuevas. Se utilizan las tablas existentes con las operaciones indicadas:

| Tabla | Operaciones | Tools / Nodos |
|-------|-------------|---------------|
| `users` | SELECT by `full_phone_number` | `load_context` |
| `users_gym_profile` | SELECT by `whatsapp_id` | `load_context` |
| `users_plans` | SELECT by `user_id` + INSERT | `load_context`, `save_workout_plan` |
| `user_weekly_schedule` | SELECT (hoy, últimos 3d, próxima) + INSERT + UPDATE | `load_context`, `schedule_sessions`, `confirm_workout_completion` |
| `pending_tasks` | SELECT by `user_id` + UPDATE | `load_context`, `confirm_workout_completion`, `decline_workout` |
| `workouts` | SELECT by `user_id`+`week`+`day_name` + INSERT bulk | `get_todays_routine`, `save_workout_plan` |
| `exercises` | SELECT by pattern/level/equipment | `get_exercises_for_draft`, `find_exercise_alternatives` |
| `set_profiles` | SELECT by goal/level/role/week | `get_exercises_for_draft` (via get_set_profile) |
| `routine_templates` + `template_days` + `day_requirements` | SELECT (JOIN) | `get_day_requirements` |
| `magic_links` | INSERT | `create_magic_link` |

---

## Mapeo de Estado KYC ↔ Unified Agent

Al integrar el KYC subgraph, se requiere mapping entre los dos estados:

**Entrada (UnifiedAgentState → KYCState)**:
```python
kyc_input = {
    "messages": state["messages"],
    "phone_number": state["phone_number"],
    "display_name": state["display_name"],
    "is_new_user": state["user_context"]["is_new_user"],
}
```

**Salida (KYCState → UnifiedAgentState)**:
```python
# Al finalizar el KYC subgraph:
return {
    "messages": kyc_state["messages"],
    "response": kyc_state["response"],
    # user_context se recargará en el siguiente turno via load_context
}
```
