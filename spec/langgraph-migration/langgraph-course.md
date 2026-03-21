# Curso Express — LangGraph para Devs Backend

> Para: Dev backend senior (5+ años, Java/Kotlin/Go/Python, MELI).
> Objetivo: Entender LangGraph lo suficiente para implementar la migración del MAIN_FLOW.
> Formato: 6 módulos, cada uno con concepto + analogía backend + código real.

---

## Módulo 1: Qué es LangGraph (Modelo Mental)

### La analogía que lo explica todo

Si vienes de backend, ya sabes lo que es un **state machine** (máquina de estados). LangGraph es exactamente eso, pero diseñado para orquestar LLMs.

```
┌──────────────────────────────────────────────────────┐
│                                                      │
│   n8n workflow  ≈  LangGraph StateGraph              │
│   n8n node      ≈  LangGraph node (function)         │
│   n8n connection≈  LangGraph edge                    │
│   n8n If/Switch ≈  LangGraph addConditionalEdges()   │
│   n8n trigger   ≈  Express route que llama invoke()  │
│   n8n sub-workflow ≈ LangGraph subgraph              │
│                                                      │
│   Spring Controller ≈  Express route handler          │
│   Service layer     ≈  Graph nodes (business logic)   │
│   Repository        ≈  Tools (DB queries)             │
│   @Transactional    ≈  Checkpointer (state persist)   │
│                                                      │
└──────────────────────────────────────────────────────┘
```

### Qué NO es LangGraph

- **No es un servidor**. Es una librería (como Gin en Go, Spring en Java). Tú la montas en Express/Fastify.
- **No es un LLM**. Orquesta LLMs (OpenAI, Gemini, Claude), no los reemplaza.
- **No es un ORM**. Tiene un checkpointer para persistir estado, pero tus queries Supabase las escribes tú.
- **No es LangChain**. Es del mismo equipo pero es un proyecto separado. LangChain = cadenas lineales (legacy). LangGraph = grafos con estado (actual).

### El flujo fundamental

```
                    invoke({ input })
                          │
                          ▼
              ┌──── StateGraph ────┐
              │                    │
   START ──→ [Node A] ──→ [Node B] ──→ END
              │    │               │
              │    └──→ [Node C] ──┘
              │                    │
              │   State = {datos   │
              │   compartidos      │
              │   entre todos      │
              │   los nodos}       │
              └────────────────────┘
                          │
                          ▼
                    return { result }
```

1. **Defines un State** (como un DTO/POJO compartido)
2. **Defines Nodes** (funciones que leen state y retornan updates parciales)
3. **Defines Edges** (quién va después de quién, con condiciones)
4. **Compilas** el grafo (como un `router.Build()` en Go)
5. **Invocas** con un input (como llamar un endpoint)

---

## Módulo 2: Setup y Estructura de Proyecto

### Instalación

```bash
mkdir gymbot-agent && cd gymbot-agent
npm init -y

# Core
npm install @langchain/langgraph @langchain/core

# LLM providers (los que ya usas en n8n)
npm install @langchain/openai          # GPT-4.1-mini, GPT-5.2
npm install @langchain/google-genai    # Gemini

# Persistencia (PostgreSQL = tu Supabase)
npm install @langchain/langgraph-checkpoint-postgres

# Schemas
npm install zod

# HTTP server
npm install express
npm install -D typescript @types/express @types/node tsx
```

### Estructura de proyecto sugerida

```
gymbot-agent/
├── src/
│   ├── index.ts                    # Express server + webhook
│   ├── graph/
│   │   ├── state.ts                # GymBotState definition
│   │   ├── gymbot-graph.ts         # StateGraph construction
│   │   └── nodes/
│   │       ├── normalize.ts        # Normalize WhatsApp message
│   │       ├── get-user.ts         # Supabase: SELECT users
│   │       ├── check-pending.ts    # Supabase: SELECT pending_tasks
│   │       ├── filter-today.ts     # Grace period logic
│   │       ├── intention-agent.ts  # GPT-4.1-mini classifier
│   │       ├── kyc-agent.ts        # Gemini KYC (8 phases)
│   │       ├── confirmation-agent.ts
│   │       ├── routine-agent.ts    # Gemini routine display
│   │       ├── scheduling-agent.ts # GPT-5.2 scheduler
│   │       └── send-whatsapp.ts    # WhatsApp Business API
│   ├── tools/
│   │   ├── get-workout.ts          # Tool_Get_Workout2
│   │   ├── update-schedule.ts      # Tool_Update_User_Weekly_Schedule
│   │   ├── create-profile.ts       # Tool_Create_User_Profile
│   │   ├── session-recommendation.ts
│   │   └── calendar.ts             # Google Calendar tools
│   ├── db/
│   │   └── supabase.ts             # pg Pool singleton
│   └── config/
│       └── env.ts                  # Environment variables
├── Dockerfile
├── tsconfig.json
└── .env
```

Esto es como tu Go backend hexagonal pero más plano — no necesitas domain/application/adapter porque LangGraph ya te da la estructura (state = domain, nodes = application, tools = adapter).

### Mapa de imports clave

```typescript
// De LangGraph
import { StateGraph, Annotation, START, END } from "@langchain/langgraph";
import { ToolNode } from "@langchain/langgraph/prebuilt";
import { PostgresSaver } from "@langchain/langgraph-checkpoint-postgres";

// De LangChain Core (solo las interfaces)
import { tool } from "@langchain/core/tools";
import { BaseMessage, AIMessage, HumanMessage } from "@langchain/core/messages";

// LLM providers
import { ChatOpenAI } from "@langchain/openai";
import { ChatGoogleGenerativeAI } from "@langchain/google-genai";

// Schemas
import { z } from "zod";
```

---

## Módulo 3: State, Nodes y Edges (Los 3 Pilares)

### Pilar 1: State (El DTO compartido)

El State es un objeto tipado que **todos los nodos pueden leer y escribir**. Piénsalo como el `context` de tu request en un middleware chain, o el `$json` de n8n.

```typescript
// src/graph/state.ts
import { Annotation } from "@langchain/langgraph";
import { BaseMessage } from "@langchain/core/messages";

// Tipos de tu dominio (como entities en Go)
interface WhatsAppMessage {
  from: string;       // phone number
  body: string;       // message text
  type: "text" | "button" | "interactive";
}

interface User {
  user_id: string;
  full_name: string;
  full_phone_number: string;
  timezone: string;
}

interface PendingTask {
  task_id: string;
  task_type: string;
  session_name: string;
  week: number;
  related_id: string;
}

interface ScheduleEntry {
  day_routine_id: string;
  session_name: string;
  week: number;
  week_day: string;
  planned_day: string;
  completed: boolean;
}

// El State — equivale a todo lo que viaja entre nodos en n8n
export const GymBotState = Annotation.Root({
  // --- Input (viene del webhook) ---
  whatsappMessage: Annotation<WhatsAppMessage>,

  // --- Contexto del usuario (se llena progresivamente) ---
  user: Annotation<User | null>({
    default: () => null,
  }),
  pendingTask: Annotation<PendingTask | null>({
    default: () => null,
  }),
  weeklySchedule: Annotation<ScheduleEntry[]>({
    default: () => [],
  }),
  todayRoutine: Annotation<ScheduleEntry | null>({
    default: () => null,
  }),

  // --- Decisiones de routing ---
  intent: Annotation<string>({
    default: () => "",
  }),

  // --- Mensajes del agente (con reducer para acumular) ---
  messages: Annotation<BaseMessage[]>({
    reducer: (current, update) => [...current, ...update],
    default: () => [],
  }),

  // --- Output ---
  responseText: Annotation<string>({
    default: () => "",
  }),
});
```

**Concepto clave — Reducers:**

Sin reducer (default): **last-write-wins**. Si nodo A escribe `intent = "CHAT"` y nodo B escribe `intent = "VER_RUTINA"`, gana el último.

Con reducer: defines la lógica de merge. Para `messages` usamos append, así cada nodo agrega mensajes sin borrar los anteriores.

```
Sin reducer:     state.user = { nuevo }     → reemplaza
Con reducer:     state.messages = [nuevo]    → se SUMA a los existentes
```

### Pilar 2: Nodes (Las funciones de negocio)

Un nodo es una **función async** que recibe el state completo y retorna un **Partial** del state (solo los campos que quiere actualizar).

```typescript
// src/graph/nodes/get-user.ts
import { pool } from "../../db/supabase";
import { GymBotState } from "../state";

// Un nodo = una función. Así de simple.
export async function getUser(state: typeof GymBotState.State) {
  const phone = state.whatsappMessage.from;

  const result = await pool.query(
    `SELECT user_id, full_name, full_phone_number, timezone
     FROM users WHERE full_phone_number = $1`,
    [phone]
  );

  if (result.rows.length === 0) {
    return { user: null }; // retorna SOLO lo que cambia
  }

  return { user: result.rows[0] };
}
```

```typescript
// src/graph/nodes/normalize.ts
export async function normalizeMessage(
  state: typeof GymBotState.State
) {
  const msg = state.whatsappMessage;
  let body = "";

  switch (msg.type) {
    case "text":
      body = msg.body;
      break;
    case "button":
      body = msg.body; // button payload
      break;
    case "interactive":
      body = msg.body; // interactive reply
      break;
  }

  // Actualiza el body normalizado directamente en el message
  return {
    whatsappMessage: { ...msg, body: body.trim() },
  };
}
```

```typescript
// src/graph/nodes/check-pending.ts
export async function checkPendingTasks(
  state: typeof GymBotState.State
) {
  if (!state.user) return { pendingTask: null };

  const result = await pool.query(
    `SELECT task_id, task_type, session_name, week, related_id
     FROM pending_tasks
     WHERE user_id = $1 AND status = 'pending'
     ORDER BY created_at DESC LIMIT 1`,
    [state.user.user_id]
  );

  return {
    pendingTask: result.rows.length > 0 ? result.rows[0] : null,
  };
}
```

**Analogía Go/Java:**
```
// Go (tu backend actual)
func (s *WorkoutService) GetTodayWorkout(ctx context.Context, userID string) (*Workout, error)

// LangGraph node (misma idea, diferente shape)
async function getWorkoutNode(state: GymBotState): Promise<Partial<GymBotState>>
```

### Pilar 3: Edges (El routing)

Los edges son las **conexiones entre nodos**. Hay 3 tipos:

```typescript
// src/graph/gymbot-graph.ts
import { StateGraph, START, END } from "@langchain/langgraph";
import { GymBotState } from "./state";

const workflow = new StateGraph(GymBotState);

// --- 1. Edge fijo: A siempre va a B ---
workflow.addEdge(START, "normalize");
workflow.addEdge("normalize", "getUser");

// --- 2. Edge condicional: If/Switch de n8n ---
workflow.addConditionalEdges(
  "getUser",                          // desde este nodo...
  (state) => {                        // evalúa esta función...
    if (!state.user) return "newUser";
    return "existingUser";
  },
  {                                   // y mapea el resultado a nodos:
    newUser: "kycAgent",
    existingUser: "checkPending",
  }
);

// --- 3. Edge condicional con 4+ salidas (tu Switch de intenciones) ---
workflow.addConditionalEdges(
  "intentionAgent",
  (state) => state.intent,            // el nodo ya dejó el intent en el state
  {
    VER_RUTINA_DE_HOY: "routineAgent",
    CONFIRMAR_RUTINA: "confirmationAgent",
    RENOVAR_MESOCICLO: "renewalSubflow",
    CHAT: "routineAgent",
  }
);

// --- Edges terminales ---
workflow.addEdge("routineAgent", "sendWhatsApp");
workflow.addEdge("confirmationAgent", "sendWhatsApp");
workflow.addEdge("sendWhatsApp", END);
```

**Diagrama del grafo resultante:**

```
START
  │
  ▼
[normalize] ──→ [getUser]
                    │
          ┌────────┼────────┐
          ▼                 ▼
     [kycAgent]       [checkPending]
          │                 │
          │           ┌─────┼─────┐
          │           ▼           ▼
          │    [confirmationAgent] [getSchedule]
          │           │            │
          │           │      [filterToday]
          │           │            │
          │           │     [intentionAgent]
          │           │       │  │  │  │
          │           │       ▼  ▼  ▼  ▼
          │           │    VER CONF REN CHAT
          │           │       │  │  │   │
          │           │       ▼  │  ▼   │
          │           │  [routineAgent]  │
          │           │       │         │
          │           └───┬───┘         │
          │               ▼             │
          │        [sendWhatsApp]◄──────┘
          │               │
          └───────────────┘
                  │
                  ▼
                 END
```

---

## Módulo 4: Tools (Las capacidades del Agente)

### Qué es un Tool

Un Tool es una **función que el LLM puede decidir llamar**. Es el equivalente a los "Tools" que conectas a un AI Agent en n8n.

El flujo es:
```
  User message
       │
       ▼
  LLM piensa → "necesito datos de la DB"
       │
       ▼
  LLM genera tool_call: { name: "get_workout", args: { user_id: "abc", ... } }
       │
       ▼
  LangGraph ejecuta la función automáticamente (ToolNode)
       │
       ▼
  Resultado vuelve al LLM como ToolMessage
       │
       ▼
  LLM genera respuesta final con los datos
```

### Definir un Tool

```typescript
// src/tools/get-workout.ts
import { tool } from "@langchain/core/tools";
import { z } from "zod";
import { pool } from "../db/supabase";

export const getWorkoutTool = tool(
  // 1. La función que ejecuta (tu lógica de negocio)
  async ({ user_id, session_name, week }) => {
    const result = await pool.query(
      `SELECT e.spanish_name, e.main_muscle, w.reps, w.sets,
              w.rir, w."rest-seconds", e.link
       FROM workouts w
       JOIN exercises e USING(exercise_id)
       WHERE w.user_id = $1 AND w.day_name = $2 AND w.week = $3
       ORDER BY w.exercise_order`,
      [user_id, session_name, week]
    );
    return JSON.stringify(result.rows);
  },
  {
    // 2. Metadata que el LLM lee para decidir cuándo usar el tool
    name: "get_workout",
    description:
      "Obtiene los ejercicios de una sesión de entrenamiento del usuario. " +
      "Devuelve nombre del ejercicio, músculo, series, reps, RIR, descanso y link de video.",
    // 3. Schema Zod = validación + genera el JSON Schema para el LLM
    schema: z.object({
      user_id: z.string().describe("UUID del usuario"),
      session_name: z.string().describe("Nombre de la sesión, ej: 'Full Body A'"),
      week: z.number().describe("Número de semana (1-4)"),
    }),
  }
);
```

```typescript
// src/tools/update-schedule.ts
export const updateScheduleTool = tool(
  async ({ user_id, week, week_day, session_name, planned_day }) => {
    await pool.query(
      `INSERT INTO user_weekly_schedule
         (day_routine_id, user_id, week, week_day, session_name, planned_day, "Completed")
       VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, false)
       ON CONFLICT (user_id, week, session_name)
       DO UPDATE SET planned_day = $5, week_day = $3`,
      [user_id, week, week_day, session_name, planned_day]
    );
    return "Schedule updated successfully";
  },
  {
    name: "update_user_weekly_schedule",
    description:
      "Crea o actualiza el horario semanal de entrenamiento del usuario. " +
      "Asigna una sesión a un día específico de la semana.",
    schema: z.object({
      user_id: z.string().describe("UUID del usuario"),
      week: z.number().describe("Número de semana (1-4)"),
      week_day: z.enum([
        "Lunes", "Martes", "Miercoles", "Jueves", "Viernes", "Sabado", "Domingo"
      ]).describe("Día de la semana en español"),
      session_name: z.string().describe("Nombre de la sesión, ej: 'Upper Body A'"),
      planned_day: z.string().describe("Fecha en formato DD/MM"),
    }),
  }
);
```

### Tool determinístico (no necesita LLM)

Algunos "tools" de n8n no son para el LLM, son lógica determinística. En LangGraph estos son simplemente **nodos regulares**, no tools:

```typescript
// Esto NO es un tool — es un nodo normal (lógica fija, no la decide el LLM)
export async function filterTodayRoutine(
  state: typeof GymBotState.State
) {
  const now = new Date();
  const bogota = new Date(now.toLocaleString("en-US", { timeZone: "America/Bogota" }));
  const today = bogota.toISOString().split("T")[0];
  const yesterday = new Date(bogota);
  yesterday.setDate(yesterday.getDate() - 1);
  const yesterdayStr = yesterday.toISOString().split("T")[0];

  // Grace period: si no tiene rutina hoy, busca la de ayer sin completar
  const todayEntry = state.weeklySchedule.find(
    (s) => s.planned_day === today && !s.completed
  );
  const yesterdayEntry = !todayEntry
    ? state.weeklySchedule.find(
        (s) => s.planned_day === yesterdayStr && !s.completed
      )
    : null;

  return {
    todayRoutine: todayEntry || yesterdayEntry || null,
  };
}
```

### ToolNode: El ejecutor automático

`ToolNode` es un nodo prebuilt de LangGraph que **ejecuta automáticamente** el tool que el LLM pidió:

```typescript
import { ToolNode } from "@langchain/langgraph/prebuilt";

// Registra todos tus tools
const tools = [getWorkoutTool, updateScheduleTool, markCompletedTool];
const toolNode = new ToolNode(tools);

// En el grafo:
workflow.addNode("tools", toolNode);
```

### El loop Agent → Tools → Agent

Este es el patrón más importante. Es cómo un LLM "usa" herramientas:

```typescript
// src/graph/nodes/routine-agent.ts
import { ChatGoogleGenerativeAI } from "@langchain/google-genai";
import { AIMessage } from "@langchain/core/messages";

const gemini = new ChatGoogleGenerativeAI({
  model: "gemini-2.0-flash",
  temperature: 0.7,
});

const tools = [getWorkoutTool];
const modelWithTools = gemini.bindTools(tools);

export async function routineAgentNode(
  state: typeof GymBotState.State
) {
  const systemPrompt = `Eres Kairos, un entrenador personal virtual...
    Cuando el usuario pida ver su rutina, usa la herramienta get_workout
    para obtener los ejercicios y luego formatea la respuesta para WhatsApp.`;

  const response = await modelWithTools.invoke([
    { role: "system", content: systemPrompt },
    ...state.messages,
  ]);

  return { messages: [response] };
}

// Router: ¿el LLM quiere usar un tool o ya terminó?
export function shouldContinueRoutineAgent(
  state: typeof GymBotState.State
): string {
  const lastMsg = state.messages[state.messages.length - 1] as AIMessage;
  if (lastMsg.tool_calls && lastMsg.tool_calls.length > 0) {
    return "tools";  // → va al ToolNode
  }
  return "done";     // → va al siguiente nodo (sendWhatsApp)
}
```

```
┌─────────────┐     tool_calls?     ┌───────────┐
│             │──── Sí ────────────→│           │
│ routineAgent│                     │  ToolNode │
│  (Gemini)   │◄────────────────────│           │
│             │   tool results      └───────────┘
│             │──── No (respuesta final) ────→ [sendWhatsApp]
└─────────────┘
```

---

## Módulo 5: Memory y Persistence (Checkpointing)

### El problema que resuelve

Tu bot de WhatsApp es **stateless por naturaleza** — cada webhook es un HTTP request independiente. Pero una conversación necesita contexto:

```
Request 1: "Hola"           → KYC Agent: "¡Hola! ¿Cómo te llamas?"
Request 2: "Camilo"         → KYC Agent necesita RECORDAR que preguntó el nombre
Request 3: "27 años"        → KYC Agent necesita RECORDAR nombre + que preguntó edad
```

En n8n esto lo hace el nodo **Postgres Chat Memory**. En LangGraph lo hace el **Checkpointer**.

### Cómo funciona PostgresSaver

```typescript
// src/db/checkpointer.ts
import { PostgresSaver } from "@langchain/langgraph-checkpoint-postgres";

// Usa tu misma conexión de Supabase
const checkpointer = PostgresSaver.fromConnString(
  process.env.SUPABASE_DB_URL!
);

// IMPORTANTE: llamar setup() una vez (crea tablas de checkpoints)
await checkpointer.setup();

export { checkpointer };
```

```typescript
// src/graph/gymbot-graph.ts
import { checkpointer } from "../db/checkpointer";

const workflow = new StateGraph(GymBotState)
  .addNode(/* ... */)
  .addEdge(/* ... */);

// Compilar CON checkpointer = tiene memoria
export const app = workflow.compile({ checkpointer });
```

### thread_id: La clave de la memoria

Cada conversación se identifica por un `thread_id`. Es como tu `session_id` en `n8n_chat_histories`:

```typescript
// src/index.ts (webhook handler)
app.post("/webhook/whatsapp", async (req, res) => {
  const message = parseWhatsAppWebhook(req.body);
  const userId = message.from;   // phone number

  // El thread_id determina QUÉ conversación se carga
  const threadId = `${userId}_main`;

  const result = await gymBotGraph.invoke(
    { whatsappMessage: message },
    { configurable: { thread_id: threadId } }
  );

  res.status(200).send("OK");
});
```

**Qué pasa internamente:**

```
invoke({ whatsappMessage }, { thread_id: "573001234567_main" })
  │
  ├─ 1. PostgresSaver busca el último checkpoint para "573001234567_main"
  │     → Si existe: carga el state completo (user, messages, schedule, etc.)
  │     → Si no existe: usa los defaults del Annotation
  │
  ├─ 2. Ejecuta los nodos del grafo
  │     → Después de CADA nodo, guarda un checkpoint en PostgreSQL
  │
  └─ 3. Retorna el state final
        → El próximo invoke con el mismo thread_id empezará donde quedó
```

### Diferentes thread_ids por contexto

En n8n tienes diferentes `sessionKey` por flujo. Mismo concepto:

```typescript
// Conversación principal del usuario
const mainThread = `${userId}_main`;

// KYC tiene su propio hilo (como en n8n: {wa_id}_kyc_v4)
const kycThread = `${userId}_kyc`;

// Chat por semana (como en n8n: {user_id}_{weekNum}_chat_v2)
const weeklyThread = `${userId}_week${weekNum}_chat`;
```

### Diferencia con n8n Postgres Chat Memory

| n8n Postgres Chat Memory | LangGraph PostgresSaver |
|--------------------------|------------------------|
| Guarda solo `messages` (role + content) | Guarda **todo el State** (messages + user + schedule + intent + etc.) |
| Un registro por mensaje | Un checkpoint por nodo ejecutado |
| Sin rollback | Puedes volver a cualquier checkpoint anterior |
| sessionKey = string | thread_id = string (misma idea) |
| windowSize = N mensajes | Puedes implementar window en el reducer |

---

## Módulo 6: Integrando Todo — El MAIN_FLOW en LangGraph

### El grafo completo

```typescript
// src/graph/gymbot-graph.ts
import { StateGraph, START, END } from "@langchain/langgraph";
import { ToolNode } from "@langchain/langgraph/prebuilt";
import { GymBotState } from "./state";
import { checkpointer } from "../db/checkpointer";

// Nodes
import { normalizeMessage } from "./nodes/normalize";
import { getUser } from "./nodes/get-user";
import { checkPendingTasks } from "./nodes/check-pending";
import { getWeeklySchedule } from "./nodes/get-schedule";
import { filterTodayRoutine } from "./nodes/filter-today";
import { intentionAgentNode } from "./nodes/intention-agent";
import { kycAgentNode, shouldContinueKyc } from "./nodes/kyc-agent";
import { confirmationAgentNode } from "./nodes/confirmation-agent";
import { routineAgentNode, shouldContinueRoutine } from "./nodes/routine-agent";
import { schedulingAgentNode, shouldContinueScheduling } from "./nodes/scheduling-agent";
import { sendWhatsApp } from "./nodes/send-whatsapp";

// Tools
import { getWorkoutTool } from "../tools/get-workout";
import { updateScheduleTool } from "../tools/update-schedule";
import { markCompletedTool } from "../tools/mark-completed";
import { createProfileTool } from "../tools/create-profile";

// Tool executors (uno por grupo de agente)
const routineTools = new ToolNode([getWorkoutTool]);
const schedulingTools = new ToolNode([updateScheduleTool]);
const confirmationTools = new ToolNode([markCompletedTool]);
const kycTools = new ToolNode([createProfileTool]);

// ═══════════════ BUILD GRAPH ═══════════════

const workflow = new StateGraph(GymBotState)

  // ── Nodos determinísticos (no usan LLM) ──
  .addNode("normalize", normalizeMessage)
  .addNode("getUser", getUser)
  .addNode("checkPending", checkPendingTasks)
  .addNode("getSchedule", getWeeklySchedule)
  .addNode("filterToday", filterTodayRoutine)
  .addNode("sendWhatsApp", sendWhatsApp)

  // ── Nodos de agente (usan LLM) ──
  .addNode("intentionAgent", intentionAgentNode)
  .addNode("kycAgent", kycAgentNode)
  .addNode("kycTools", kycTools)
  .addNode("confirmationAgent", confirmationAgentNode)
  .addNode("confirmationTools", confirmationTools)
  .addNode("routineAgent", routineAgentNode)
  .addNode("routineTools", routineTools)
  .addNode("schedulingAgent", schedulingAgentNode)
  .addNode("schedulingTools", schedulingTools)

  // ═══════════════ EDGES ═══════════════

  // Flujo principal (determinístico)
  .addEdge(START, "normalize")
  .addEdge("normalize", "getUser")

  // ¿Existe el usuario?
  .addConditionalEdges("getUser", (state) => {
    return state.user ? "existingUser" : "newUser";
  }, {
    newUser: "kycAgent",
    existingUser: "checkPending",
  })

  // ¿Tiene tarea pendiente?
  .addConditionalEdges("checkPending", (state) => {
    return state.pendingTask ? "hasPending" : "noPending";
  }, {
    hasPending: "confirmationAgent",
    noPending: "getSchedule",
  })

  // Obtener schedule y filtrar hoy
  .addEdge("getSchedule", "filterToday")

  // ¿Tiene rutina para hoy/ayer?
  .addConditionalEdges("filterToday", (state) => {
    return state.todayRoutine ? "hasRoutine" : "noRoutine";
  }, {
    hasRoutine: "intentionAgent",
    noRoutine: "schedulingAgent",  // necesita agendar
  })

  // Switch de intenciones (el corazón del routing)
  .addConditionalEdges("intentionAgent", (state) => {
    return state.intent;
  }, {
    VER_RUTINA_DE_HOY: "routineAgent",
    CONFIRMAR_RUTINA: "confirmationAgent",
    RENOVAR_MESOCICLO: "sendWhatsApp", // TODO: subgraph de renewal
    CHAT: "routineAgent",
  })

  // ── Agent loops (LLM ↔ Tools) ──

  // KYC Agent loop
  .addConditionalEdges("kycAgent", shouldContinueKyc, {
    tools: "kycTools",
    done: "sendWhatsApp",
  })
  .addEdge("kycTools", "kycAgent")

  // Routine Agent loop
  .addConditionalEdges("routineAgent", shouldContinueRoutine, {
    tools: "routineTools",
    done: "sendWhatsApp",
  })
  .addEdge("routineTools", "routineAgent")

  // Confirmation Agent loop
  .addConditionalEdges("confirmationAgent", (state) => {
    const last = state.messages[state.messages.length - 1];
    if ("tool_calls" in last && (last as any).tool_calls?.length > 0) {
      return "tools";
    }
    return "done";
  }, {
    tools: "confirmationTools",
    done: "sendWhatsApp",
  })
  .addEdge("confirmationTools", "confirmationAgent")

  // Scheduling Agent loop
  .addConditionalEdges("schedulingAgent", shouldContinueScheduling, {
    tools: "schedulingTools",
    done: "sendWhatsApp",
  })
  .addEdge("schedulingTools", "schedulingAgent")

  // Terminal
  .addEdge("sendWhatsApp", END);

// ═══════════════ COMPILE ═══════════════

export const gymBotGraph = workflow.compile({ checkpointer });
```

### El servidor Express

```typescript
// src/index.ts
import express from "express";
import { gymBotGraph } from "./graph/gymbot-graph";

const app = express();
app.use(express.json());

// WhatsApp webhook verification (GET)
app.get("/webhook", (req, res) => {
  const mode = req.query["hub.mode"];
  const token = req.query["hub.verify_token"];
  const challenge = req.query["hub.challenge"];

  if (mode === "subscribe" && token === process.env.WA_VERIFY_TOKEN) {
    res.status(200).send(challenge);
  } else {
    res.sendStatus(403);
  }
});

// WhatsApp webhook (POST) — reemplaza al Trigger de n8n
app.post("/webhook", async (req, res) => {
  // Responde 200 inmediatamente (WhatsApp requiere <5s)
  res.status(200).send("OK");

  try {
    const entry = req.body?.entry?.[0];
    const change = entry?.changes?.[0];
    const message = change?.value?.messages?.[0];

    if (!message) return; // No message (status update, etc.)

    // Construir input del grafo
    const whatsappMessage = {
      from: message.from,
      body: message.text?.body || message.button?.text || message.interactive?.button_reply?.title || "",
      type: message.type as "text" | "button" | "interactive",
    };

    // Invocar el grafo con thread_id basado en el teléfono
    const threadId = `${message.from}_main`;

    await gymBotGraph.invoke(
      { whatsappMessage },
      { configurable: { thread_id: threadId } }
    );
  } catch (error) {
    console.error("Error processing webhook:", error);
  }
});

app.listen(3000, () => {
  console.log("GymBot Agent running on :3000");
});
```

### Mapeo n8n → LangGraph (referencia rápida)

| Concepto n8n | LangGraph equivalente |
|---|---|
| `WhatsApp Trigger1` | `app.post("/webhook")` |
| `If` node | `addConditionalEdges()` con función `(state) => ...` |
| `Switch` node (4 outputs) | `addConditionalEdges()` retornando el intent |
| `Postgres` node (query) | Nodo normal con `pool.query()` |
| `Code` node | Nodo normal con lógica JS/TS |
| `AI Agent` + Tools | Agent node + `ToolNode` + conditional loop |
| `Postgres Chat Memory` | `PostgresSaver` + `thread_id` |
| `Send Message` (WhatsApp) | Nodo `sendWhatsApp` con `fetch()` a la API |
| `Execute Workflow` | Subgraph compilado como nodo |
| `$json.propertyName` | `state.propertyName` |
| `{{ $node["X"].json.y }}` | El nodo anterior ya lo dejó en `state.y` |
| Credential (postgres) | `process.env.SUPABASE_DB_URL` |
| Credential (OpenAI) | `process.env.OPENAI_API_KEY` (auto-detected) |

---

## Resumen: Mapa Mental Completo

```
┌─────────────────────────────────────────────────────────────┐
│                    LangGraph = 5 cosas                      │
│                                                             │
│  1. STATE      → El DTO compartido entre nodos              │
│                  (Annotation.Root con tipos + reducers)      │
│                                                             │
│  2. NODES      → Funciones async (state) => Partial<state>  │
│                  Pueden ser:                                │
│                  • Determinísticas (query, code logic)       │
│                  • De agente (LLM + tools en loop)           │
│                                                             │
│  3. EDGES      → Conexiones entre nodos                     │
│                  • Fijas: addEdge("a", "b")                 │
│                  • Condicionales: addConditionalEdges()      │
│                                                             │
│  4. TOOLS      → Funciones que el LLM puede llamar          │
│                  Definidas con tool() + schema Zod           │
│                  Ejecutadas por ToolNode automáticamente     │
│                                                             │
│  5. CHECKPOINTER → Persistencia de estado en PostgreSQL     │
│                    Identificado por thread_id                │
│                    Guarda snapshot COMPLETO del state        │
│                    Permite resume, rollback, debugging       │
└─────────────────────────────────────────────────────────────┘

Flujo de ejecución:

  HTTP Request ──→ invoke(input, { thread_id }) ──→ [cargar checkpoint]
       │                                                    │
       │              ┌──────── StateGraph ────────┐        │
       │              │                            │        │
       │    START ──→ [Node1] ──→ [Node2] ──→ END  │        │
       │              │   ↕ checkpoint   ↕          │        │
       │              │   [PostgreSQL]              │        │
       │              └────────────────────────────┘        │
       │                                                    │
       └◄───────────── return state ◄───────────────────────┘
```
