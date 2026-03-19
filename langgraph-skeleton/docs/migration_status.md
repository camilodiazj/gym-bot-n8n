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

Los recordatorios y outreach proactivo pueden migrarse a Kairos usando **Cloud Scheduler → endpoint HTTP** en el mismo servicio. Esto elimina n8n como dependencia para flujos que tocan WhatsApp.

### Estrategia: Cloud Scheduler → Kairos Cron Endpoints

```
Cloud Scheduler (GCP)
    │
    ├── 5:00 AM Bogotá  → GET /cron/morning-reminder
    ├── 8:00 PM Bogotá  → GET /cron/evening-followup
    └── 8:00 PM Lunes   → GET /cron/weekly-prompt
```

Cada endpoint:
1. Query Supabase para obtener usuarios relevantes
2. Genera mensaje personalizado (el agente ya tiene el contexto)
3. Envía WhatsApp via `_send_whatsapp_message()` (ya existe en server.py)
4. Retorna resumen de envíos

### Endpoint: `GET /cron/morning-reminder`
- Query: `user_weekly_schedule` WHERE `planned_day = hoy` AND `Completed = false`
- JOIN: `users` para obtener `full_phone_number`
- Para cada usuario: envía "Hoy tienes [session_name]. ¡Dale con toda! 💪"
- Sin LLM — mensaje template directo

### Endpoint: `GET /cron/evening-followup`
- Query: `user_weekly_schedule` WHERE `planned_day = hoy` AND `Completed = false`
- Para cada usuario sin completar: invoca el agente Kairos con mensaje interno
  "Pregúntale al usuario si completó su sesión de hoy"
- El agente usa el contexto y responde naturalmente
- Crea `pending_task` de tipo `CONFIRMAR_RUTINA`

### Endpoint: `GET /cron/weekly-prompt`
- Query: completitud de la semana por usuario
- Segmentar: CELEBRATION (100%), GROWTH (parcial), RE_ENGAGEMENT (0%)
- Enviar WhatsApp template personalizado por segmento

### Qué NO migrar (se queda en n8n)
- **DailyReport**: 7 queries SQL complejas + HTML email — no toca WhatsApp, puro analytics
- **InteractionAnalysis**: Gemini analysis + HTML email — analytics interno

### Ventajas de la migración
- **Elimina n8n como punto de falla** para recordatorios (si n8n cae, usuarios no reciben reminder)
- **Un solo servicio** maneja todo el flujo de WhatsApp
- **Reutiliza** `_send_whatsapp_message()` y `context_loader` que ya existen
- **Cloud Scheduler** es gratis (3 jobs gratis en GCP) y más confiable que n8n cron

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
