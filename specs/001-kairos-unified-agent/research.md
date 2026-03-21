# Research: Agente Unificado Kairos

**Feature**: `001-kairos-unified-agent`
**Date**: 2026-03-17

---

## Decision 1: Herramientas granulares vs. monolítica para creación de rutina

**Decision**: Herramientas granulares separadas. El borrador vive en el estado de LangGraph, no es retornado por una tool.

**Rationale**:
- El patrón ya establecido en Case 3 llama `get_exercises_by_pattern` por patrón y `get_set_profile` por rol — herramientas pequeñas con razonamiento incremental
- ToolMessages pequeños = mejor razonamiento del LLM (un JSON de 4 días entero es innecesariamente grande)
- `interrupt()` funciona limpiamente en boundary de nodos (después de que el agent construye el borrador, antes de `save_workout_plan`)
- `find_exercise_alternatives` reutiliza la misma lógica de búsqueda que la construcción inicial — no hay duplicación

**Alternatives considered**:
- Tool monolítica `draft_routine()` que devuelve el plan completo: rechazada porque colapsa la orquestación, impide recovery granular, y dificulta el flujo de modificaciones

---

## Decision 2: Persistencia del borrador — estado vs. tabla temporal

**Decision**: `draft_routine: DraftRoutine | None` en el estado de LangGraph (InMemorySaver).

**Rationale**:
- El borrador no necesita sobrevivir entre sesiones de conversación
- Si el usuario abandona, se recrea en el siguiente turno
- InMemorySaver persiste el estado dentro del thread (misma sesión)
- Evita crear una tabla de borradores y su lógica de limpieza

**Alternatives considered**:
- Tabla `draft_plans` en Supabase: rechazada por overhead innecesario (limpieza de borradores huérfanos, migraciones adicionales)

---

## Decision 3: Carga de contexto — paralela vs. secuencial

**Decision**: `asyncio.gather()` para las 5 queries Supabase en `load_context`.

**Rationale**:
- Las queries son independientes entre sí (users, plans, schedule, pending_tasks, gym_profile)
- En paralelo: ~150ms. Secuencial: ~500ms. El contexto se carga en cada turno.
- httpx ya es async — el patrón `await asyncio.gather(...)` es idiomático en la codebase

**Alternatives considered**:
- Secuencial: rechazado por latencia acumulada inaceptable (suma de tiempos vs. máximo)

---

## Decision 4: Manejo de sesiones perdidas — cuántos días hacia atrás

**Decision**: 3 días hacia atrás (configurable como constante `MISSED_SESSIONS_WINDOW_DAYS = 3`).

**Rationale**:
- El sistema de grace period actual (KAN-109) ya maneja ayer (1 día)
- 3 días cubre fines de semana y días libres inesperados sin ser excesivo
- Más de 3 días resultaría en sesiones "perdidas" que el usuario probablemente no quiere recuperar

**Alternatives considered**:
- 1 día (solo ayer): demasiado restrictivo para el caso de uso descrito
- 7 días: demasiado amplio — sesiones de la semana pasada pertenecen conceptualmente al mesociclo anterior

---

## Decision 5: Thread ID — por teléfono vs. por sesión

**Decision**: `thread_id = f"case6_{phone_number}"` — un thread persistente por usuario.

**Rationale**:
- Mismo patrón que Case 4 y Case 5 (`f"case4_{thread_id}"`, `f"case5_{phone_number}"`)
- El contexto del agente (historial de conversación) debe persistir entre mensajes del mismo día
- InMemorySaver reinicia al reiniciar el servidor — aceptable para el estado actual del proyecto

**Alternatives considered**:
- Thread por sesión del día: rechazado porque pierde continuidad intra-día ("dame la rutina" + "ya terminé" deben estar en el mismo thread)
- PostgresSaver: considerado pero no implementado en ningún case previo — fuera de scope

---

## Decision 6: Integración KYC — subgraph vs. redirect

**Decision**: KYC como subgraph directo dentro de `graph_live.py`, con state mapping entre `UnifiedAgentState` y `KYCState`.

**Rationale**:
- LangGraph soporta compilar un graph como subgraph e invocarlo con `add_node("kyc", kyc_graph.compile())`
- Case 5 permanece intacto — zero modificaciones al código existente
- El state mapping es mínimo: `phone_number` y `display_name` se pasan; el `response` se extrae al finalizar

**Alternatives considered**:
- Llamar al endpoint `/case5/kyc/live/chat` via HTTP desde el agente: rechazado por complejidad innecesaria y latencia adicional
- Copiar código de Case 5 en Case 6: rechazado por duplicación

---

## Decision 7: `supabase_update()` — endpoint PATCH de PostgREST

**Decision**: Nuevo método en `src/shared/supabase_client.py` usando `PATCH /rest/v1/{table}?filters`.

**Rationale**:
- PostgREST soporta PATCH con filtros en query params (mismo estilo que GET con filters)
- Consistente con `supabase_query()` y `supabase_insert()` existentes
- Necesario para `confirm_workout_completion` (UPDATE user_weekly_schedule) y `decline_workout` (UPDATE pending_tasks)

**Pattern**:
```python
async def supabase_update(table: str, data: dict, filters: dict[str, str]) -> list[dict]:
    url = f"{SUPABASE_URL}/rest/v1/{table}"
    headers = _get_headers()
    headers["Prefer"] = "return=representation"
    async with httpx.AsyncClient() as client:
        response = await client.patch(url, headers=headers, json=data, params=filters)
        response.raise_for_status()
        return response.json()
```
