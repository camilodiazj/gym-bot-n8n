# Estado de Migración: n8n → LangGraph (Kairos Agent)

**Fecha**: 2026-03-18
**Branch**: `001-kairos-unified-agent`
**Producción**: https://kairos-agent-148665080566.us-central1.run.app

---

## Resumen

GymBot tiene 7 workflows de n8n en producción. El agente Kairos (Case 6) reemplaza el **MAIN_FLOW** como punto de entrada para mensajes de WhatsApp, con un agente inteligente (Gemini + 11 tools) que decide libremente qué hacer. Los workflows automatizados (reportes, recordatorios) siguen en n8n por ahora.

---

## Workflows de n8n vs Kairos

### 1. MAIN_FLOW — Orquestador principal

| Capacidad | n8n | Kairos | Notas |
|-----------|-----|--------|-------|
| Recibir mensajes WhatsApp | Webhook trigger | `POST /webhook` directo | Sin n8n intermediario |
| Filtrar ruido (status, audio) | If node | `_extract_message()` en server.py | Funciona |
| Identificar usuario | Postgres query | `load_context` (asyncio.gather) | Funciona |
| Ver rutina del día | Switch → handler | `get_todays_routine` tool | Funciona |
| Confirmar entrenamiento | Switch → handler | `confirm_workout_completion` tool | Funciona (grace period) |
| Declinar entrenamiento | Switch → handler | `decline_workout` tool | Funciona |
| Chat general fitness | Switch → AI Agent | Respuesta directa del LLM | Funciona (personalizado) |
| Agendar sesiones | Switch → handler | `schedule_sessions` tool | Bug: 400 en insert |
| Detectar renovación mesociclo | Switch → subworkflow | `get_mesocycle_status` tool | Solo lectura, no ejecuta renovación |
| Google Calendar events | CreateCalendarEvent | No implementado | Pendiente |
| Detección de intenciones | Switch con 5+ outputs | Agente decide libremente | Mejor: no necesita intenciones fijas |
| Magic link tracker | No existía | `create_magic_link` tool | Nuevo en Kairos |

**Status: ~70% migrado** — Faltan: agendamiento (bug), renovación completa, calendario.

---

### 2. WORKOUT_CREATOR — Generador de rutinas

| Capacidad | n8n | Kairos | Notas |
|-----------|-----|--------|-------|
| Cargar perfil usuario | Postgres query | `context_loader` + gym_profile | Funciona |
| ProcessUserPreferences | Code node (400+ líneas) | No implementado | Mapeo músculos ES→EN, health filtering, volume modifier |
| Consultar day_requirements | Postgres JOIN | `get_day_requirements` tool | Funciona |
| Buscar ejercicios por patrón | Postgres query dinámico | `get_exercises_for_draft` tool | Funciona (sin health/equipment filtering) |
| Selección AI personalizada | LangChain Agent | El LLM selecciona de los candidatos | Funciona (menos sofisticado) |
| Deduplicación de ejercicios | Code node | No implementado | Riesgo de duplicados |
| Validación de duración | Code node | No implementado | Puede exceder tiempo objetivo |
| Expansión 4 semanas | Code node | `save_workout_plan` replica W1 × 4 | Funciona (sin progresión por semana) |
| Guardar plan + workouts | Postgres INSERT | `save_workout_plan` + resolución IDs | Funciona (60 workouts guardados) |
| Enviar email con rutina | Gmail node | No implementado | Pendiente |
| Borrador interactivo | No existía | Draft mode con feedback | Nuevo en Kairos — el usuario opina antes de guardar |

**Status: ~40% migrado** — El core funciona (buscar + seleccionar + guardar). Falta: ProcessUserPreferences, health filtering, dedup, validación duración, email.

---

### 3. MorningReminder — Recordatorio 5 AM

| Capacidad | n8n | Kairos | Notas |
|-----------|-----|--------|-------|
| Trigger diario 5 AM | Cron trigger | No implementado | Necesita scheduler externo |
| Query sesiones del día | Postgres query | Ya existe en `context_loader` | Reutilizable |
| Enviar WhatsApp template | WhatsApp Template node | No implementado | Templates ≠ mensajes de texto |

**Status: 0% migrado** — Sigue en n8n. No es urgente migrar (es un cron job simple).

---

### 4. GymBotMesocycleRenewal — Renovación de mesociclo

| Capacidad | n8n | Kairos | Notas |
|-----------|-----|--------|-------|
| Detectar W4 completada | Query + If | `all_w4_completed` flag en context | Funciona |
| Consultar estado | Postgres query | `get_mesocycle_status` tool | Funciona |
| Ofrecer opciones (mantener/cambiar/rotar) | AI Agent multi-turn | Prompt del agente | Funciona (conversacional) |
| Ejecutar MANTENER | Delete schedules + increment mesocycle | No implementado | Necesita tool de UPDATE |
| Ejecutar CAMBIAR_DIAS | Delete + regenerar con WORKOUT_CREATOR | No implementado | Necesita orquestación |
| Ejecutar ROTAR | Swap ejercicios por alternativas | `find_exercise_alternatives` tool | Parcial |

**Status: ~20% migrado** — Detecta y conversa, pero no ejecuta la renovación.

---

### 5. WeeklySchedulingPrompt — Outreach semanal

| Capacidad | n8n | Kairos |
|-----------|-----|--------|
| Trigger semanal 8 PM lunes | Cron | No implementado |
| Segmentación por completitud | SQL query | No implementado |
| Templates de celebración/growth/re-engagement | WhatsApp Templates | No implementado |

**Status: 0% migrado** — Sigue en n8n. Es un cron job independiente.

---

### 6. DailyReport — Reporte operacional 6 AM

| Capacidad | n8n | Kairos |
|-----------|-----|--------|
| 7 queries de métricas | Postgres nodes | No implementado |
| HTML email report | Code node + Gmail | No implementado |
| WhatsApp summary | WhatsApp send | No implementado |

**Status: 0% migrado** — Sigue en n8n. No tiene relación con el agente conversacional.

---

### 7. InteractionAnalysis — Análisis semanal

| Capacidad | n8n | Kairos |
|-----------|-----|--------|
| Métricas cuantitativas | SQL queries | No implementado |
| Muestras de conversación | SQL queries | No implementado |
| Análisis con Gemini | Google Gemini node | No implementado |
| HTML email report | Code + Gmail | No implementado |

**Status: 0% migrado** — Sigue en n8n. Es analytics interno.

---

## Matriz de Prioridades

### Prioridad Alta (bloquea flujos de usuario)
| Item | Impacto | Esfuerzo |
|------|---------|----------|
| Fix `schedule_sessions` (bug 400) | Usuarios no pueden agendar días | Bajo — probablemente campos faltantes |
| Renovación de mesociclo (ejecutar, no solo detectar) | Usuarios quedan sin rutina después de W4 | Medio — necesita tools de UPDATE/DELETE |
| Health filtering en `get_exercises_for_draft` | Usuarios con restricciones reciben ejercicios peligrosos | Medio — agregar filtros al query |

### Prioridad Media (mejora la experiencia)
| Item | Impacto | Esfuerzo |
|------|---------|----------|
| ProcessUserPreferences (músculos, volume modifier) | Rutinas menos personalizadas | Alto — 400+ líneas de lógica |
| Deduplicación de ejercicios en draft | Posibles duplicados en la rutina | Bajo — validación post-selección |
| Google Calendar events | Usuarios no reciben invitaciones | Medio — API de Google Calendar |
| Email con rutina semana 1 | Usuarios no tienen referencia escrita | Bajo — template HTML + Gmail API |

### Prioridad Baja (migrar vía Cloud Scheduler)
| Item | Impacto | Esfuerzo |
|------|---------|----------|
| MorningReminder (5 AM) | Elimina dependencia de n8n | Bajo — endpoint + Cloud Scheduler |
| 8PM Follow-up | Elimina dependencia de n8n | Bajo — endpoint + Cloud Scheduler |
| WeeklySchedulingPrompt | Elimina dependencia de n8n | Medio — segmentación + templates |
| DailyReport | Funciona bien en n8n | N/A — no migrar (analytics interno) |
| InteractionAnalysis | Funciona bien en n8n | N/A — no migrar (analytics interno) |

---

## Plan de Migración de Cron Jobs

**Revisado por**: n8n-agent (×3), pixel-dev (arquitectura), kiro-coach (fitness/coaching)

Los recordatorios y outreach proactivo migran a **Cloud Scheduler → Kairos HTTP endpoints**. Esto elimina n8n como dependencia para todo lo que toca WhatsApp.

### Arquitectura

```
Cloud Scheduler (GCP) + Header X-Cron-Secret
    │
    ├── Mañana/Tarde/Noche → GET /cron/morning-reminder (segmentado por preferred_schedule)
    ├── 8:00 PM Bogotá     → GET /cron/evening-followup
    └── 8:00 PM Lunes      → GET /cron/weekly-prompt
```

#### Estructura de código (pixel-dev approved)

```
src/shared/
    whatsapp.py            # NUEVO — extraer de server.py: _send_whatsapp_message() + _send_whatsapp_template()
    supabase_client.py     # MODIFICAR — agregar supabase_rpc()
src/cron/
    __init__.py
    router.py              # FastAPI APIRouter con los 3 endpoints + seguridad (X-Cron-Secret)
    morning_reminder.py    # Lógica de negocio
    evening_followup.py    # Lógica de negocio
    weekly_prompt.py       # Lógica de negocio
```

En `server.py`: `app.include_router(cron_router, prefix="/cron", tags=["Cron Jobs"])`

#### Seguridad (pixel-dev)
- Header `X-Cron-Secret` validado por FastAPI dependency en el router de cron
- Cloud Scheduler envía el header automáticamente
- Secret almacenado en env var `CRON_SECRET`

#### Idempotencia (pixel-dev)
- Tabla `cron_executions(execution_id PK, started_at, completed_at, users_sent)` en Supabase
- execution_id = `{tipo}-{fecha}` (ej: `morning-reminder-2026-03-18`)
- Si ya existe registro `completed` para hoy → retorna 200 con `already_executed: true`

---

### Endpoint 1: `GET /cron/morning-reminder`

**Query**: PostgREST resource embedding (pixel-dev: no necesita RPC)
```
user_weekly_schedule?select=session_name,week,users(full_name,full_phone_number)
&planned_day_utc=gte.{today_start}&planned_day_utc=lt.{tomorrow_start}&Completed=eq.false
```

**Mensaje**: WhatsApp template `prueba_2|es_CO` — parámetros: [nombre]

**Mejora kiro-coach**: Incluir nombre de sesión + 3 ejercicios principales en el reminder. Query adicional a `workouts JOIN exercises` para los ejercicios compound (exercise_order 1-3). Esto duplica la adherencia según la investigación sobre implementation intentions.

**Mejora kiro-coach**: Segmentar por `preferred_schedule`:
| Horario usuario | Cron schedule | Justificación |
|-----------------|---------------|---------------|
| Mañana | 5:30 AM | Llega antes de ir al gym |
| Tarde | 12:30 PM | Recordatorio al mediodía |
| Noche | 5:30 PM | Antes de salir del trabajo |

**Filtros**: Excluir test phones (`full_phone_number NOT LIKE '5700000%'`)

**Complejidad**: BAJA (~2-3h)

---

### Endpoint 2: `GET /cron/evening-followup`

**Query**: 2 queries PostgREST paralelas (pixel-dev: no necesita RPC)
1. `user_weekly_schedule` WHERE `planned_day_utc=hoy` AND `Completed=false`
2. `pending_tasks` WHERE `status=pending` (filtro duplicados)

**Acción**: Crear `pending_task` tipo `CONFIRMAR_RUTINA` + enviar WhatsApp

**Mensaje**: WhatsApp **template** (pixel-dev: NO texto libre — evita el gap de 24h de WhatsApp). Crear template `confirmacion_entrenamiento|es_CO` con parámetro [session_name].

**Mejora kiro-coach — Cooldown**: NO enviar si:
1. Ya respondió a un follow-up en las últimas 24h
2. Ya confirmó la rutina por cualquier canal hoy
3. Lleva 3+ días consecutivos sin responder → cambiar a estrategia de re-engagement

**Mejora kiro-coach — Grace period**: Incluir sesiones de AYER sin completar (no solo hoy). Si el usuario no tiene sesión hoy pero tiene una de ayer pendiente, ofrecer como recuperación.

**Mejora kiro-coach — Excluir horario noche**: Usuarios con `preferred_schedule='Noche'` podrían estar entrenando a las 8 PM. Desplazar su follow-up a las 10 PM o excluirlos del 8 PM.

**Interacción con Kairos**: No requiere cambios — `context_loader`, `prompts.py`, `confirm_workout_completion` y `decline_workout` ya manejan pending_tasks.

**Complejidad**: MEDIA (~3h)

---

### Endpoint 3: `GET /cron/weekly-prompt`

**Query**: Supabase RPC function (pixel-dev: PostgREST no puede expresar la CTE)
- Crear `CREATE FUNCTION get_users_needing_prompt(phone_filter TEXT)` en Supabase
- Agregar `supabase_rpc()` a `supabase_client.py`
- CTE: segmentación por completitud semanal, ventana de 3 días, excluir semana 4

**Segmentación kiro-coach (5 segmentos en vez de 3)**:

| Segmento | Condición | Template | Tono |
|----------|-----------|----------|------|
| CELEBRATION | 100% completado | `semana_completa\|es_CO` | Felicitación + momentum |
| IMPROVEMENT | Mejoró vs semana anterior | Nuevo template | Reconocimiento del progreso |
| GROWTH | Parcial, tendencia estable | `semana_incompleta\|es_CO` | Motivacional neutro |
| REBOOT | Semana 1 del mesociclo | Nuevo template | Celebrar el inicio |
| RE_ENGAGEMENT | 0%, semana ≥ 2 | `kairos_rutina_sin_abrir_dias` | Sin culpa, reencuadre |

**Mejora kiro-coach — RE_ENGAGEMENT**: NUNCA mencionar sesiones perdidas. Solo futuro. "La semana que viene arrancamos. ¿Quieres retomar con 2 días en vez de 3?"

**Mejora kiro-coach — Timing**: RE_ENGAGEMENT el miércoles en vez del lunes (dar 2 días de gracia).

**Complejidad**: MEDIA-ALTA (~4-5h)

---

### Mejoras de coaching adicionales (kiro-coach — backlog)

| Mejora | Impacto | Esfuerzo | Fase |
|--------|---------|----------|------|
| **Milestone recognition** (1er semana completa, 1er meso, 10 sesiones) | Alto en retención | Medio | Futuro |
| **Feedback post-entrenamiento** ("¿Cómo te fue?") cada 1-2 semanas | Alto en personalización | Bajo | Futuro |
| **Ajuste de carga semana 3** (aviso de semana dura del meso) | Medio | Bajo | Futuro |
| **Mensajes en días de descanso** (por qué el descanso es parte del plan) | Medio en retención | Bajo | Futuro |

---

### Decisiones técnicas (pixel-dev approved)

| Decisión | Elección | Justificación |
|----------|----------|---------------|
| Endpoints en server.py vs módulo | **Módulo `src/cron/`** | SRP — server.py ya tiene 900+ líneas |
| PostgREST vs RPC vs psycopg | **PostgREST** (morning, evening) + **RPC** (weekly) | Cada query usa el approach adecuado a su complejidad |
| WhatsApp funciones | **`src/shared/whatsapp.py`** | Compartido entre webhook + cron; extraer de server.py |
| Texto libre vs templates | **Templates siempre** | Funcionan fuera de ventana de 24h |
| httpx client | **Reutilizar dentro del endpoint** | Un `AsyncClient()` por ejecución del cron, no por mensaje |
| Agregar httpx a pyproject.toml | **Sí** | Dependencia implícita actual, hacerla explícita |

---

### Qué NO migrar (se queda en n8n)
- **DailyReport**: 7 queries SQL + HTML email — analytics interno, no toca WhatsApp
- **InteractionAnalysis**: Gemini analysis + HTML email — analytics interno

### Orden de implementación

1. **Extraer `src/shared/whatsapp.py`** — desbloquea los 3 endpoints
2. **Morning Reminder** — complejidad baja, valor alto, prueba la infra
3. **Evening Follow-up** — complejidad media, cierra el loop de accountability
4. **Supabase RPC + `supabase_rpc()`** — prerequisito del weekly
5. **Weekly Prompt** — complejidad media-alta, mayor personalización
6. **Cloud Scheduler** — 3 jobs (gratis en GCP)
7. **Desactivar workflows en n8n** — solo después de validar en producción

---

## Arquitectura Objetivo

```
                    ┌─────────────────────────────────┐
                    │         WhatsApp User            │
                    └──────────┬──────────────────────┘
                               │
                    ┌──────────▼──────────────────────┐
                    │   Cloud Run: kairos-agent        │
                    │                                  │
                    │   POST /webhook (chat directo)   │
                    │   GET /cron/* (Cloud Scheduler)  │
                    │                                  │
                    │   load_context → router →        │
                    │   kairos_agent ↔ tools            │
                    │   (Gemini + 11 tools)            │
                    │                                  │
                    │   PostgresSaver (Supabase)       │
                    └──────────┬──────────────────────┘
                               │
          ┌────────────────────┼────────────────────┐
          │                    │                    │
┌─────────▼──────┐  ┌─────────▼──────┐  ┌─────────▼──────┐
│ Cloud Scheduler│  │ Workout Tracker │  │  n8n (solo     │
│                │  │  (React/Go)     │  │  analytics)    │
│ • 5AM Reminder │  │  Firebase +     │  │                │
│ • 8PM Follow-up│  │  Cloud Run      │  │ • Daily Report │
│ • Weekly Prompt│  │                 │  │ • Interaction  │
│                │  │                 │  │   Analysis     │
└────────────────┘  └─────────────────┘  └────────────────┘
```

**Meta final**: n8n solo queda para analytics interno (DailyReport + InteractionAnalysis). Todo lo que toca WhatsApp vive en Kairos.
