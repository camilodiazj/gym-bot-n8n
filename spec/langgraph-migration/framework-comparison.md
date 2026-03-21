# Informe: Migración de n8n a Framework de Agentes

## Contexto

GymBot actualmente orquesta toda su lógica conversacional (WhatsApp → detección de intención → routing → ejecución) mediante workflows de n8n. El objetivo es migrar a un framework basado en código que replique las capacidades del `MAIN_FLOW.json` y workflows secundarios, ganando control, testabilidad y eliminando la dependencia de n8n.

Este informe compara **LangGraph**, **LangChain** y **Claude Agent SDK** para esta migración.

---

## 1. Resumen Ejecutivo

**LangGraph (TypeScript) es la opción recomendada.** Ofrece orquestación explícita basada en grafos que mapea 1:1 con la lógica del MAIN_FLOW de n8n, persistencia nativa en PostgreSQL (compatible con Supabase), y soporte multi-modelo (GPT, Gemini, Claude) — preservando la arquitectura actual de modelos especializados por tarea.

**Claude Agent SDK** es una alternativa viable si se acepta el lock-in a modelos Claude y se construye memoria/routing custom.

**LangChain standalone NO se recomienda** — sus propios creadores lo consideran legacy y recomiendan LangGraph.

---

## 2. Arquitectura Actual (MAIN_FLOW de n8n)

```
WhatsApp Message
       │
   Normalize_Message (Code)
       │
   GetUser (Supabase: users by phone)
       │
   ¿user_exists?
   ├─ No → KYC Agent (Gemini, 8 fases) → Tool_Create_User_Profile
   │                                      → Call WORKOUT_CREATOR
   └─ Sí ↓
      Check_Pending_Tasks (Supabase)
         │
      ¿Has_Pending_Task?
      ├─ Sí → CONFIRMATION_AGENT (GPT-4.1-mini)
      │        → Tool_Update_User_Weekly_Schedule1
      └─ No ↓
         GetWeeklySchedule → Filter_Today_Routine (grace period 1 día)
            │
         ¿has_planned_workouts?
         ├─ Sí → Intention_Agent (GPT-4.1-mini) → Switch:
         │       ├─ VER_RUTINA_DE_HOY → AI Agent (Gemini) + Tool_Get_Workout2
         │       ├─ CONFIRMAR_RUTINA  → CONFIRMATION_AGENT
         │       ├─ RENOVAR_MESOCICLO → Sub-workflow (MesocycleRenewal)
         │       └─ CHAT             → AI Agent (Gemini)
         └─ No → Check_Mesocycle_Complete
                 ├─ Completo → MesocycleRenewal (sub-workflow)
                 └─ No       → Scheduling Agent (GPT-5.2)
                               → Tool_Update_User_Weekly_Schedule
                               → Calendar Events (Google Calendar API)
```

### Inventario de Agentes

| Agente | Modelo | Propósito | Tools |
|--------|--------|-----------|-------|
| KYC Agent | Gemini 3-flash | Onboarding 8 fases, recolecta 22 campos | `Tool_Create_User_Profile`, `Tool_Session_Recommendation` |
| Intention Agent | GPT-4.1-mini | Clasifica intención del usuario | Ninguno |
| Confirmation Agent | GPT-4.1-mini | Confirma completación de entrenamientos | `Tool_Update_User_Weekly_Schedule1` |
| AI Agent (Rutina) | Gemini | Muestra rutina formateada para WhatsApp | `Tool_Get_Workout2` |
| Scheduling Agent | GPT-5.2 | Agenda sesiones de la semana | `Tool_Update_User_Weekly_Schedule` |

### Tools (10+)

| Tool | Tipo | Operación |
|------|------|-----------|
| `Tool_Get_Workout2` | Postgres JOIN | SELECT workouts + exercises por user/session/week |
| `Tool_Update_User_Weekly_Schedule` | Postgres UPSERT | INSERT/UPDATE schedule con planned_day |
| `Tool_Update_User_Weekly_Schedule1` | Postgres UPDATE | Marca pending_task y schedule como completed |
| `Tool_Create_User_Profile` | Supabase INSERT | Crea perfil en users_gym_profile |
| `Tool_Session_Recommendation` | Code (JS) | Cálculo determinístico de días recomendados |
| `GetCalendarData` | Postgres JOIN | Datos para crear eventos de Google Calendar |
| `CreateCalendarMagicLink` | Postgres INSERT | Genera magic link (48h expiry) |
| `PrepareCalendarEvents` | Code (JS) | Construye objetos de evento Calendar |
| `GoogleCalendar_CreateEvent` | Google API | Crea eventos en Google Calendar |
| `Check_Pending_Tasks` | Supabase SELECT | Busca tareas pendientes tipo CONFIRMAR_RUTINA |

### Workflows Secundarios a Migrar

| Workflow | Trigger | Complejidad |
|----------|---------|-------------|
| WORKOUT_CREATOR | Sub-workflow (desde KYC) | Alta — AI genera plan 4 semanas |
| GymBotMesocycleRenewal | Sub-workflow (desde MAIN_FLOW) | Media — conversación 3 opciones |
| MorningReminder | Cron 5 AM | Baja — query + template WhatsApp |
| WeeklySchedulingPrompt | Cron 8 PM | Baja — query + template por tasa |
| DailyReport | Cron 6 AM | Baja — 7 queries SQL + email HTML |
| InteractionAnalysis | Cron Lunes 8 AM | Media — queries + análisis LLM |

---

## 3. Matriz de Comparación

| Criterio | LangGraph | Claude Agent SDK | LangChain (standalone) |
|----------|-----------|-----------------|----------------------|
| **Estado** | Activo, recomendado | Production-ready | **DEPRECADO** |
| **Modelo de orquestación** | Grafo explícito (StateGraph con nodos + edges condicionales) | Loop implícito de tool-use (el agente decide) | Cadena lineal/secuencial |
| **Gestión de estado** | Typed state con reducers; persiste entre nodos | Solo mensajes de conversación | Sin estado built-in |
| **PostgreSQL nativo** | Sí — `PostgresSaver` | No — implementación custom necesaria | No |
| **Soporte multi-modelo** | GPT, Claude, Gemini, Mistral, etc. | **Solo Claude** | Multi-modelo |
| **Lenguaje** | Python (primario), TypeScript (oficial) | TypeScript + Python (ambos first-class) | Python primario |
| **Routing condicional** | `addConditionalEdges()` — determinístico en código | Implícito — el agente decide vía razonamiento | No soportado |
| **Ejecución paralela** | Built-in (fan-out/fan-in) | No built-in | No |
| **Sub-agentes** | Grafos anidados o patrón supervisor | Handoff via tool calls | No |
| **Streaming** | Completo (tokens + state updates) | Completo (SSE) | Básico |
| **Debugging** | Time-travel, checkpoints, LangSmith | Logging básico | Logging básico |
| **Costo plataforma** | Self-hosted: gratis. Cloud: $39/user/mes | Gratis (solo API calls) | N/A |
| **Curva de aprendizaje** | Media | Baja | N/A |
| **Recovery de errores** | Retry + rollback via checkpoints | Retry manual | Mínimo |

---

## 4. Cómo Mapea Cada Framework al MAIN_FLOW

### 4.1 LangGraph (TypeScript)

El MAIN_FLOW de n8n **ES** un grafo dirigido. LangGraph lo representa directamente en código:

**Definición de estado:**
```typescript
const GymBotState = Annotation.Root({
  whatsappMessage: Annotation<WhatsAppMessage>,
  normalizedBody: Annotation<string>,
  user: Annotation<User | null>,
  pendingTask: Annotation<PendingTask | null>,
  weeklySchedule: Annotation<WeeklySchedule[]>,
  intent: Annotation<"VER_RUTINA" | "CONFIRMAR" | "RENOVAR" | "CHAT">,
  hasTodayRoutine: Annotation<boolean>,
  responseText: Annotation<string>,
  messages: Annotation<BaseMessage[]>({ reducer: messagesReducer }),
});
```

**Grafo (mapeo directo de nodos n8n → nodos LangGraph):**
```typescript
const graph = new StateGraph(GymBotState)
  .addNode("normalize", normalizeMessage)
  .addNode("getUser", getUserFromSupabase)
  .addNode("checkPending", checkPendingTasks)
  .addNode("intentionAgent", runIntentionAgent)       // GPT-4.1-mini
  .addNode("kycAgent", runKYCAgent)                    // Gemini
  .addNode("confirmationAgent", runConfirmationAgent)  // GPT-4.1-mini
  .addNode("routineAgent", runRoutineDisplayAgent)     // Gemini
  .addNode("schedulingAgent", runSchedulingAgent)      // GPT-5.2
  .addNode("sendWhatsApp", sendWhatsAppMessage)
  // Edges condicionales = If/Switch nodes de n8n
  .addConditionalEdges("getUser", routeByUserExists, {
    newUser: "kycAgent",
    existingUser: "checkPending",
  })
  .addConditionalEdges("checkPending", routeByPendingTask, {
    hasPending: "confirmationAgent",
    noPending: "intentionAgent",
  })
  .addConditionalEdges("intentionAgent", routeByIntent, {
    VER_RUTINA: "routineAgent",
    CONFIRMAR: "confirmationAgent",
    RENOVAR: "renewalSubflow",
    CHAT: "routineAgent",
  });
```

**Tool (ejemplo Tool_Get_Workout2):**
```typescript
const getWorkoutTool = tool(
  async ({ user_id, session_name, week }) => {
    const result = await supabase.query(`
      SELECT spanish_name, main_muscle, reps, sets, rir, "rest-seconds"
      FROM workouts JOIN exercises USING(exercise_id)
      WHERE user_id = $1 AND day_name = $2 AND week = $3
      ORDER BY exercise_order`, [user_id, session_name, week]);
    return JSON.stringify(result.rows);
  },
  {
    name: "get_workout",
    description: "Obtiene ejercicios de una sesión del usuario",
    schema: z.object({
      user_id: z.string(),
      session_name: z.string(),
      week: z.number(),
    }),
  }
);
```

**Memoria (out of the box):**
```typescript
import { PostgresSaver } from "@langchain/langgraph-checkpoint-postgres";
const checkpointer = PostgresSaver.fromConnString(SUPABASE_DB_URL);
const app = graph.compile({ checkpointer });
// Cada conversación WhatsApp = un thread_id
await app.invoke(state, { configurable: { thread_id: `${userId}_${week}_chat` } });
```

### 4.2 Claude Agent SDK

El routing es **implícito** — el agente decide qué hacer basado en su system prompt:

```typescript
const gymBotAgent = new Agent({
  model: "claude-sonnet-4-20250514",  // Locked to Claude
  systemPrompt: `Eres Kairos Personal Trainer...
    Basado en el contexto, decide:
    1. Si es nuevo → recolecta perfil KYC
    2. Si tiene tarea pendiente → maneja confirmación
    3. Detecta intención y actúa`,
  tools: [getWorkoutTool, updateScheduleTool, ...],
});
```

**Diferencia clave**: No hay `addConditionalEdges()`. El Switch de 4 salidas (CONFIRMAR/CHAT/VER_RUTINA/RENOVAR) depende de que el LLM siga instrucciones correctamente. No es determinístico.

**Memoria**: No hay `PostgresSaver`. Necesitas construir tu propia capa de persistencia.

---

## 5. Pros y Contras para GymBot

### LangGraph

| Pros | Contras |
|------|---------|
| Mapeo 1:1 con routing de n8n (If/Switch → conditional edges) | TypeScript SDK va 1-2 meses detrás del Python SDK |
| `PostgresSaver` se conecta directo a Supabase existente | Agrega dependencia del ecosistema LangChain (muchos packages) |
| Multi-modelo: preserva GPT-4.1-mini (barato, clasificación) + GPT-5.2 (scheduling) + Gemini (display) | WhatsApp no es built-in (custom, igual que Claude SDK) |
| Time-travel debugging para troubleshoot conversaciones multi-turn | LangGraph Cloud cuesta $39/user/mes (self-hosted es gratis) |
| Routing determinístico — testeable con assertions exactas | Curva de aprendizaje de conceptos de grafos y reducers |
| Checkpoint recovery si el webhook crashea mid-ejecución | |

### Claude Agent SDK

| Pros | Contras |
|------|---------|
| Patrón de tools más simple (JSON Schema directo) | **Lock-in a Claude** — no puede usar GPT-4.1-mini ni Gemini |
| Sin costo de plataforma (solo API calls) | Routing implícito — no hay garantía de que siga el Switch de 4 vías |
| TypeScript first-class (alineado con frontend actual) | Sin `PostgresSaver` — memoria custom a construir |
| Claude excel en system prompts complejos en español | Sin visualización de grafo para debugging |
| Deploy más simple — un servicio Node.js | Sub-workflows (WORKOUT_CREATOR, Renewal) requieren orchestración custom |
| | Sin checkpoint/resume — si crashea, pierde estado |
| | API de Claude más cara que GPT-4.1-mini para clasificación simple |

---

## 6. Estimación de Esfuerzo

| Componente | LangGraph (TS) | Claude Agent SDK |
|------------|----------------|-----------------|
| Webhook WhatsApp + normalize | Medio | Medio |
| Clasificación de intención | Bajo | Bajo |
| **Routing (Switch/If nodes)** | **Bajo** (conditional edges) | **Alto** (system prompt) |
| KYC Agent (8 fases) | Medio | Bajo |
| Scheduling Agent | Medio | Medio |
| **Memoria PostgreSQL** | **Bajo** (PostgresSaver) | **Alto** (custom) |
| Definición de 10+ tools | Medio | Medio |
| **Sub-workflows** | **Medio** (grafos anidados) | **Alto** (custom orchestration) |
| Google Calendar integration | Medio | Medio |
| Cron workflows (5) | Medio | Medio |
| **Testing E2E** | **Medio** (determinístico) | **Alto** (no-determinístico) |
| **Debugging/Monitoring** | **Bajo** (LangSmith + checkpoints) | **Alto** (custom logging) |

### Resumen

| Framework | Esfuerzo Total | Timeline Estimado | Riesgo |
|-----------|---------------|-------------------|--------|
| **LangGraph (TypeScript)** | **Medio** | 6-8 semanas | Bajo |
| **Claude Agent SDK** | **Alto** | 8-12 semanas | Medio |
| **LangChain standalone** | N/A | N/A | **No usar** |

---

## 7. Recomendación Final

### LangGraph (TypeScript)

**Razones principales:**

1. **Match arquitectónico**: El MAIN_FLOW de n8n ES un grafo dirigido. LangGraph es literalmente la representación en código de lo que n8n dibuja visualmente. Cada nodo `If` → `addConditionalEdges()`, cada agente → nodo del grafo.

2. **Preserva multi-modelo**: GymBot usa 3 modelos optimizados por tarea (GPT-4.1-mini para clasificación barata, GPT-5.2 para scheduling complejo, Gemini para display). LangGraph los soporta todos. Forzar todo a Claude costaría más y potencialmente degradaría la clasificación.

3. **Memoria PostgreSQL gratis**: `PostgresSaver` se conecta directamente a Supabase. No hay que construir capa custom ni migrar `n8n_chat_histories`.

4. **Routing determinístico**: El Switch de 4 vías (CONFIRMAR/CHAT/VER_RUTINA/RENOVAR) queda codificado explícitamente, no dependiendo del razonamiento del LLM. Esto hace los tests confiables.

5. **TypeScript**: El equipo ya tiene TypeScript (workout-tracker/) y Go (backend). No necesita aprender Python.

### Path de Migración Sugerido

| Fase | Semanas | Entregable |
|------|---------|------------|
| **1. Fundación** | 1-2 | Proyecto TS + LangGraph + PostgresSaver + webhook WhatsApp + normalize + getUser |
| **2. Routing Core** | 3-4 | Intention Agent + Switch routing + Confirmation Agent + Routine Display Agent |
| **3. Flujos Complejos** | 5-6 | KYC Agent + Scheduling Agent + Calendar + sub-workflows (WORKOUT_CREATOR, Renewal) |
| **4. Cron + Testing** | 7-8 | MorningReminder, DailyReport, WeeklySchedulingPrompt, InteractionAnalysis + E2E tests |

### Arquitectura de Deploy (Self-Hosted — Sin Costo de Plataforma)

LangGraph tiene 2 modos de deploy:

| Modo | Costo Plataforma | Descripción |
|------|-----------------|-------------|
| **Self-hosted (recomendado)** | **$0** | Despliegas como cualquier servicio HTTP (igual que el Go backend). LangGraph es una librería npm (`@langchain/langgraph`), no una plataforma. |
| LangGraph Cloud (managed) | $39/user/mes | Plataforma managed de LangChain. **No la necesitas.** Es para enterprise con dashboard integrado. |

**Deploy self-hosted = exactamente como tu Go backend en Cloud Run:**

```
Cloud Run (existente)          Cloud Run (nuevo, self-hosted)
┌──────────────────┐          ┌──────────────────────────┐
│  Go Backend      │          │  LangGraph Service       │
│  Dockerfile      │          │  Dockerfile              │
│  Port 8080       │          │  Port 3000               │
│                  │          │  npm: @langchain/langgraph│
│  REST API para   │          │  WhatsApp webhook        │
│  el frontend     │          │  Agent orchestration     │
│                  │          │  Cron jobs (scheduler)    │
│                  │          │  Google Calendar API      │
└────────┬─────────┘          └────────┬─────────────────┘
         │                             │
         └──────────┬──────────────────┘
                    │
              ┌──────▼──────┐
              │  Supabase    │
              │  PostgreSQL  │
              │  (shared DB) │
              └──────────────┘
```

**Costos reales del servicio LangGraph self-hosted:**
- Cloud Run: ~$5-15/mes (misma escala que tu Go backend)
- API calls OpenAI/Gemini: variable según uso (igual que ahora con n8n)
- Supabase: ya lo pagas, misma instancia
- LangGraph librería: **$0** (open source, MIT license)

El Go backend sigue sirviendo el frontend React. El nuevo servicio LangGraph reemplaza TODOS los workflows de n8n.

---

## Archivos Clave para la Implementación

- `n8n/running_flows/MAIN_FLOW.json` — Fuente de verdad para routing, prompts, tools
- `n8n/running_flows/WORKOUT_CREATOR.json` — Lógica de generación de planes
- `n8n/running_flows/GymBotMesocycleRenewal.json` — Flujo de renovación
- `workout-tracker-back/internal/adapter/repository/postgres/` — Queries SQL de referencia
- `e2e/test_data_setup.sql` — Fixtures para validar migración
- `spec/` — Especificaciones de features que deben preservarse
