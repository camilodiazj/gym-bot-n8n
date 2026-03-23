"""FastAPI server exposing all 6 LangGraph cases as HTTP endpoints.

Usage:
    source .venv/bin/activate
    uvicorn server:app --reload --port 8000

Endpoints:
    POST /case1              — Basic graph + Gemini summary
    POST /case2              — Conditional routing + Gemini exercise selection
    POST /case3              — Tools + Agent loop (mock data)
    POST /case3/live         — Tools + Agent loop (Supabase real)
    POST /case4/chat         — Multi-turn KYC (conversational)
    GET  /case4/history      — View conversation history for a thread
    POST /case4/live/chat    — Multi-turn KYC + Supabase workout generation
    GET  /case4/live/history — View conversation history (live version)
    POST /case5/kyc/chat     — Onboarding KYC (5-turn, 10 fields)
    GET  /case5/kyc/history  — View KYC conversation history
    GET  /case5/kyc/status   — Check KYC session status
    POST /case5/kyc/live/chat    — Onboarding KYC + Supabase persistence
    GET  /case5/kyc/live/history — View KYC conversation history (live)
    POST /case6/chat         — Unified Agent Kairos (context + tools)
    GET  /case6/history      — View Case 6 conversation history
"""

import asyncio
import logging
import os
import traceback
from contextlib import asynccontextmanager
from datetime import datetime, timezone

import httpx
from fastapi import FastAPI, HTTPException, Query, Request
from pydantic import BaseModel, Field
from langchain_core.messages import HumanMessage, ToolMessage

logger = logging.getLogger("kairos")

from cases.case1_basic_graph.graph import build_case1_graph
from cases.case2_conditional.graph import build_case2_graph
from cases.case3_tools_agent.graph import build_case3_graph
from cases.case3_tools_agent.graph_live import build_case3_live_graph
from cases.case4_memory.graph import build_case4_graph
from cases.case4_memory.graph_live import build_case4_live_graph
from cases.case5_onboarding_kyc.graph import build_case5_graph
from cases.case5_onboarding_kyc.graph_live import build_case5_live_graph
from cases.case6_unified_agent.graph import build_case6_graph
from cases.case6_unified_agent.graph_live import build_case6_live_workflow
from src.shared.checkpointer import postgres_checkpointer_context


# ═══════════════ KYC NUDGE TRACKER ═══════════════

# In-memory registry of active KYC sessions for nudge checking.
# Key: thread_id, Value: {"last_interaction_at": ISO str, "nudge_sent": bool, "phone": str}
_kyc_sessions: dict[str, dict] = {}

NUDGE_DELAY_SECONDS = 30 * 60  # 30 minutes
NUDGE_CHECK_INTERVAL = 5 * 60  # Check every 5 minutes

# ═══════════════ WEBHOOK DEDUP & LOCKING ═══════════════

_user_locks: dict[str, asyncio.Lock] = {}  # per-phone serialization

_background_tasks: set[asyncio.Task] = set()  # prevent GC of fire-and-forget tasks


async def _is_duplicate_message(message_id: str, phone_from: str) -> bool:
    """Return True if this message ID was already processed (Supabase-backed, multi-instance safe)."""
    from src.shared.supabase_client import SUPABASE_URL, SUPABASE_ANON_KEY
    if not SUPABASE_URL:
        return False  # no Supabase → skip dedup
    url = f"{SUPABASE_URL}/rest/v1/processed_webhook_messages"
    headers = {
        "apikey": SUPABASE_ANON_KEY,
        "Authorization": f"Bearer {SUPABASE_ANON_KEY}",
        "Content-Type": "application/json",
        "Prefer": "return=representation,resolution=ignore-duplicates",
    }
    try:
        async with httpx.AsyncClient() as client:
            resp = await client.post(
                url,
                headers=headers,
                json={"message_id": message_id, "phone_from": phone_from},
                params={"on_conflict": "message_id"},
                timeout=5,
            )
            if resp.status_code == 201 and resp.json():
                return False  # inserted → first time seeing this message
            return True  # conflict / empty → duplicate
    except Exception as e:
        logger.warning(f"[WA] Dedup check failed: {e} — processing anyway")
        return False  # on error, allow processing (better than dropping messages)


def _get_user_lock(phone_number: str) -> asyncio.Lock:
    """Get or create an asyncio.Lock for a specific user."""
    if phone_number not in _user_locks:
        _user_locks[phone_number] = asyncio.Lock()
    return _user_locks[phone_number]


async def _nudge_checker():
    """Background task: check for stale KYC sessions and log nudge."""
    while True:
        await asyncio.sleep(NUDGE_CHECK_INTERVAL)
        now = datetime.now(timezone.utc)
        for thread_id, session in list(_kyc_sessions.items()):
            if session.get("nudge_sent"):
                continue
            try:
                last_dt = datetime.fromisoformat(session["last_interaction_at"])
                gap = (now - last_dt).total_seconds()
                if gap > NUDGE_DELAY_SECONDS:
                    print(f"[NUDGE] Session {thread_id} inactive for {gap/60:.0f} min — sending nudge")
                    session["nudge_sent"] = True
            except (ValueError, KeyError):
                pass


@asynccontextmanager
async def lifespan(app):
    """Initialize async resources on startup, cleanup on shutdown."""
    global case6_live_graph

    async with postgres_checkpointer_context() as checkpointer:
        case6_live_graph = case6_live_workflow.compile(checkpointer=checkpointer)
        cp_name = type(checkpointer).__name__
        print(f"[STARTUP] Case 6 live graph compiled (checkpointer: {cp_name})")
        task = asyncio.create_task(_nudge_checker())
        yield
        task.cancel()


# ═══════════════ APP ═══════════════

app = FastAPI(
    title="LangGraph Skeleton API",
    description="6 casos progresivos de LangGraph con Gemini — GymBot workout scenarios",
    version="0.2.0",
    lifespan=lifespan,
)

# Build graphs once at startup
case1_graph = build_case1_graph()
case2_graph = build_case2_graph()
case3_graph = build_case3_graph()
case3_live_graph = build_case3_live_graph()
case4_graph = build_case4_graph()
case4_live_graph = build_case4_live_graph()
case5_graph = build_case5_graph()
case5_live_graph = build_case5_live_graph()
case6_graph = build_case6_graph()
case6_live_workflow = build_case6_live_workflow()
case6_live_graph = None  # Compiled async in lifespan with checkpointer


# ═══════════════ SCHEMAS ═══════════════

class UserRequest(BaseModel):
    user_id: str = Field(
        default="camilo-001",
        description="User ID. Options: camilo-001, ana-002, carlos-003",
    )


class ChatRequest(BaseModel):
    message: str = Field(
        description="User message for the KYC conversation",
    )
    thread_id: str = Field(
        default="postman-session-1",
        description="Thread ID for conversation memory. Same thread_id = same conversation",
    )


class ThreadRequest(BaseModel):
    thread_id: str = Field(default="postman-session-1")


class KYCChatRequest(BaseModel):
    message: str = Field(description="User message text")
    phone_number: str = Field(description="Full phone number (e.g., 573001234567)")
    display_name: str = Field(default="", description="WhatsApp display name")
    thread_id: str = Field(default="", description="Session ID (auto-generated from phone if empty)")


class Case6ChatRequest(BaseModel):
    message: str = Field(description="User message for Kairos agent")
    phone_number: str = Field(description="Full phone number (e.g., 573001234567)")
    display_name: str = Field(default="", description="WhatsApp display name (optional)")


# ═══════════════ CASE 1 ═══════════════

@app.post("/case1", tags=["Case 1 — Basic Graph"])
async def case1_basic_graph(req: UserRequest):
    """Grafo lineal: load_profile → validate → generate_summary (Gemini).

    Prueba con:
    - `camilo-001` (Health A, Intermedio)
    - `ana-002` (Health C, Principiante)
    - `unknown-999` (usuario inexistente)
    """
    result = await case1_graph.ainvoke({"user_id": req.user_id})
    return {
        "case": "1 — Basic Graph + Gemini",
        "user_id": req.user_id,
        "profile": result.get("profile"),
        "is_valid": result["is_valid"],
        "validation_errors": result["validation_errors"],
        "summary": result["summary"],
    }


# ═══════════════ CASE 2 ═══════════════

@app.post("/case2", tags=["Case 2 — Conditional Routing"])
async def case2_conditional(req: UserRequest):
    """Routing condicional por health_status + Gemini selecciona ejercicios.

    Rutas:
    - Health A → no_restrictions (pool completo: 16 ejercicios)
    - Health B → restrict_lower_body (sin barbell squat/deadlift)
    - Health C → restrict_upper_body (sin overhead press)
    """
    result = await case2_graph.ainvoke({"user_id": req.user_id})
    return {
        "case": "2 — Conditional Routing + Gemini",
        "user_id": req.user_id,
        "route_taken": result["route_taken"],
        "available_exercises_count": len(result["available_exercises"]),
        "selected_exercises": [
            {
                "exercise_id": e["exercise_id"],
                "name": e["spanish_name"],
                "pattern": e["pattern"],
                "muscle": e["main_muscle"],
            }
            for e in result["selected_exercises"]
        ],
    }


# ═══════════════ CASE 3 ═══════════════

@app.post("/case3", tags=["Case 3 — Tools + Agent Loop"])
async def case3_tools_agent(req: UserRequest):
    """Gemini decide qué tools llamar. Agent loop: exercise_selector ↔ ToolNode.

    Tools disponibles:
    - `get_exercises_by_pattern(pattern, level)`
    - `get_set_profile(goal, role, week)`
    """
    result = await case3_graph.ainvoke({"user_id": req.user_id})

    tool_calls = [
        {"index": i, "tool": m.name, "preview": m.content[:150]}
        for i, m in enumerate(result["messages"])
        if isinstance(m, ToolMessage)
    ]

    return {
        "case": "3 — Tools + Agent Loop",
        "user_id": req.user_id,
        "total_messages": len(result["messages"]),
        "tool_calls_count": len(tool_calls),
        "tool_calls": tool_calls,
        "workout_result": result["workout_result"],
    }


# ═══════════════ CASE 3 LIVE (Supabase) ═══════════════

@app.post("/case3/live", tags=["Case 3 — Tools + Agent Loop"])
async def case3_live(req: UserRequest):
    """Igual que /case3 pero las tools consultan Supabase REAL (1,657 ejercicios).

    Tools disponibles:
    - `get_exercises_by_pattern(pattern, level)` — Supabase exercises table
    - `get_set_profile(goal, role, week)` — Supabase set_profiles table
    - `search_exercises(muscle, equipment)` — Busca por músculo y equipo
    """
    result = await case3_live_graph.ainvoke({"user_id": req.user_id})

    tool_calls = [
        {"index": i, "tool": m.name, "preview": m.content[:200]}
        for i, m in enumerate(result["messages"])
        if isinstance(m, ToolMessage)
    ]

    return {
        "case": "3 LIVE — Tools + Agent Loop (Supabase)",
        "user_id": req.user_id,
        "data_source": "Supabase (1,657 real exercises)",
        "total_messages": len(result["messages"]),
        "tool_calls_count": len(tool_calls),
        "tool_calls": tool_calls,
        "workout_result": result["workout_result"],
    }


# ═══════════════ CASE 4 ═══════════════

@app.post("/case4/chat", tags=["Case 4 — Multi-turn KYC"])
async def case4_chat(req: ChatRequest):
    """Conversación multi-turn con Kairos (KYC Agent).

    Usa el mismo `thread_id` para mantener la conversación.
    Kairos pregunta: objetivo, días/semana, nivel.

    Ejemplo de flujo:
    1. "Hola, quiero mi rutina"
    2. "Ganar masa muscular"
    3. "3 días, soy intermedio"
    """
    config = {"configurable": {"thread_id": req.thread_id}}

    result = await case4_graph.ainvoke(
        {"messages": [HumanMessage(content=req.message)]},
        config,
    )

    response = {
        "case": "4 — Multi-turn KYC",
        "thread_id": req.thread_id,
        "kairos_response": result["response"],
        "is_complete": result.get("is_complete", False),
    }

    if result.get("workout_plan"):
        response["workout_plan"] = result["workout_plan"]

    return response


@app.get("/case4/history", tags=["Case 4 — Multi-turn KYC"])
async def case4_history(thread_id: str = "postman-session-1"):
    """Ver el historial de conversación de un thread."""
    config = {"configurable": {"thread_id": thread_id}}

    try:
        state = await case4_graph.aget_state(config)
        if not state.values:
            return {"thread_id": thread_id, "messages": [], "note": "Thread vacío o no existe"}

        messages = [
            {
                "index": i,
                "role": "user" if isinstance(m, HumanMessage) else "kairos",
                "content": m.content,
            }
            for i, m in enumerate(state.values.get("messages", []))
        ]
        return {
            "thread_id": thread_id,
            "total_messages": len(messages),
            "messages": messages,
        }
    except Exception:
        return {"thread_id": thread_id, "messages": [], "note": "Thread no encontrado"}


# ═══════════════ CASE 4 LIVE (KYC + Supabase) ═══════════════

@app.post("/case4/live/chat", tags=["Case 4 LIVE — KYC + Supabase"])
async def case4_live_chat(req: ChatRequest):
    """Conversación multi-turn con Kairos + generación de rutina con Supabase.

    Fase 1 (KYC): Kairos recolecta objetivo, días/semana, nivel.
    Fase 2 (Workout): Cuando el KYC se completa, el agente usa tools de Supabase
    para buscar ejercicios reales (1,657) y generar la rutina.

    Usa el mismo `thread_id` para mantener la conversación.

    Ejemplo de flujo:
    1. "Hola, quiero mi rutina"         → Kairos pregunta objetivo
    2. "Ganar masa muscular"            → Kairos pregunta días
    3. "3 días, soy intermedio"         → KYC completo → genera rutina con Supabase
    """
    config = {"configurable": {"thread_id": req.thread_id}}

    result = await case4_live_graph.ainvoke(
        {"messages": [HumanMessage(content=req.message)]},
        config,
    )

    # Count tool calls if any (Phase 2)
    tool_calls = [
        {"tool": m.name, "preview": m.content[:200]}
        for m in result.get("messages", [])
        if isinstance(m, ToolMessage)
    ]

    response = {
        "case": "4 LIVE — KYC + Supabase Workout",
        "thread_id": req.thread_id,
        "kairos_response": result["response"],
        "is_complete": result.get("is_complete", False),
    }

    if tool_calls:
        response["tool_calls_count"] = len(tool_calls)
        response["tool_calls"] = tool_calls
        response["data_source"] = "Supabase (1,657 real exercises)"

    if result.get("workout_plan"):
        response["workout_plan"] = result["workout_plan"]

    return response


@app.get("/case4/live/history", tags=["Case 4 LIVE — KYC + Supabase"])
async def case4_live_history(thread_id: str = "postman-session-1"):
    """Ver el historial de conversación de un thread (versión live)."""
    config = {"configurable": {"thread_id": thread_id}}

    try:
        state = await case4_live_graph.aget_state(config)
        if not state.values:
            return {"thread_id": thread_id, "messages": [], "note": "Thread vacío o no existe"}

        messages = []
        for i, m in enumerate(state.values.get("messages", [])):
            role = "user" if isinstance(m, HumanMessage) else (
                "tool" if isinstance(m, ToolMessage) else "kairos"
            )
            messages.append({
                "index": i,
                "role": role,
                "content": m.content[:300] if isinstance(m, ToolMessage) else m.content,
            })

        return {
            "thread_id": thread_id,
            "total_messages": len(messages),
            "messages": messages,
        }
    except Exception:
        return {"thread_id": thread_id, "messages": [], "note": "Thread no encontrado"}


# ═══════════════ CASE 5 — Onboarding KYC ═══════════════

@app.post("/case5/kyc/chat", tags=["Case 5 — Onboarding KYC"])
async def case5_kyc_chat(req: KYCChatRequest):
    """Conversación multi-turn para onboarding KYC de 5 turnos, 10 campos.

    Usa `phone_number` como identificador. El `thread_id` se genera
    automáticamente como `kyc_{phone_number}` si no se proporciona.

    Ejemplo de flujo:
    1. "Hola, quiero empezar" → Pregunta objetivo
    2. "Ganar masa muscular, 3 años de exp" → Pregunta entreno
    3. "Gym, 3 días, por la mañana" → Pregunta métricas
    4. "Hombre, 27 años, 171 cm, 67 kg" → Pregunta salud
    5. "No tengo lesiones" → Resumen de perfil
    """
    thread_id = req.thread_id or f"kyc_{req.phone_number}"
    config = {"configurable": {"thread_id": thread_id}}

    result = await case5_graph.ainvoke(
        {
            "messages": [HumanMessage(content=req.message)],
            "phone_number": req.phone_number,
            "display_name": req.display_name,
        },
        config,
    )

    collected = result.get("collected_data", {})

    response = {
        "case": "5 — Onboarding KYC",
        "thread_id": thread_id,
        "kairos_response": result.get("response", ""),
        "current_turn": result.get("current_turn", 0),
        "is_complete": result.get("is_complete", False),
        "awaiting_confirmation": result.get("awaiting_confirmation", False),
        "collected_fields": list(collected.keys()),
        "route_to_trainer": result.get("route_to_trainer", False),
    }

    if result.get("health_code"):
        response["health_code"] = result["health_code"]
    if result.get("profile_confirmed"):
        response["profile_saved"] = True

    # Track session for nudge checker (US2)
    _kyc_sessions[thread_id] = {
        "last_interaction_at": result.get("last_interaction_at", datetime.now(timezone.utc).isoformat()),
        "nudge_sent": False,
        "phone": req.phone_number,
    }

    return response


@app.get("/case5/kyc/history", tags=["Case 5 — Onboarding KYC"])
async def case5_kyc_history(thread_id: str = "kyc_test"):
    """Ver historial de conversación y datos recolectados de un thread KYC."""
    config = {"configurable": {"thread_id": thread_id}}

    try:
        state = await case5_graph.aget_state(config)
        if not state.values:
            return {"thread_id": thread_id, "messages": [], "note": "Thread vacío o no existe"}

        collected = state.values.get("collected_data", {})
        messages = [
            {
                "index": i,
                "role": "user" if isinstance(m, HumanMessage) else "kairos",
                "content": m.content,
            }
            for i, m in enumerate(state.values.get("messages", []))
        ]

        return {
            "thread_id": thread_id,
            "total_messages": len(messages),
            "current_turn": state.values.get("current_turn", 0),
            "collected_data": collected,
            "messages": messages,
        }
    except Exception:
        return {"thread_id": thread_id, "messages": [], "note": "Thread no encontrado"}


@app.get("/case5/kyc/status", tags=["Case 5 — Onboarding KYC"])
async def case5_kyc_status(phone_number: str = "573001234567"):
    """Check KYC session status without sending a message."""
    thread_id = f"kyc_{phone_number}"
    config = {"configurable": {"thread_id": thread_id}}

    try:
        state = await case5_graph.aget_state(config)
        if not state.values:
            return {
                "phone_number": phone_number,
                "status": "not_started",
                "current_turn": 0,
                "collected_fields": [],
                "remaining_fields": [],
            }

        vals = state.values
        collected = vals.get("collected_data", {})
        from cases.case5_onboarding_kyc.state import ALL_KYC_FIELDS
        collected_fields = list(collected.keys())
        remaining_fields = [f for f in ALL_KYC_FIELDS if f not in collected]

        # Determine status
        if vals.get("profile_confirmed"):
            status = "completed"
        elif vals.get("awaiting_confirmation"):
            status = "awaiting_confirmation"
        elif collected:
            status = "in_progress"
        else:
            status = "not_started"

        # Get nudge info from in-memory tracker
        session_info = _kyc_sessions.get(thread_id, {})

        return {
            "phone_number": phone_number,
            "status": status,
            "current_turn": vals.get("current_turn", 0),
            "collected_fields": collected_fields,
            "remaining_fields": remaining_fields,
            "last_interaction_at": vals.get("last_interaction_at", ""),
            "nudge_sent": session_info.get("nudge_sent", False),
        }
    except Exception:
        return {"phone_number": phone_number, "status": "error", "note": "No se pudo obtener estado"}


# ═══════════════ CASE 5 LIVE — KYC + Supabase ═══════════════

@app.post("/case5/kyc/live/chat", tags=["Case 5 LIVE — KYC + Supabase"])
async def case5_live_chat(req: KYCChatRequest):
    """KYC conversation with Supabase persistence.

    Same as /case5/kyc/chat but:
    - check_user queries real Supabase users table
    - save_profile writes to real Supabase users + users_gym_profile tables
    """
    thread_id = req.thread_id or f"kyc_{req.phone_number}"
    config = {"configurable": {"thread_id": thread_id}}

    result = await case5_live_graph.ainvoke(
        {
            "messages": [HumanMessage(content=req.message)],
            "phone_number": req.phone_number,
            "display_name": req.display_name,
        },
        config,
    )

    collected = result.get("collected_data", {})

    response = {
        "case": "5 LIVE — Onboarding KYC (Supabase)",
        "thread_id": thread_id,
        "data_source": "Supabase (real users + users_gym_profile)",
        "kairos_response": result.get("response", ""),
        "current_turn": result.get("current_turn", 0),
        "is_complete": result.get("is_complete", False),
        "awaiting_confirmation": result.get("awaiting_confirmation", False),
        "collected_fields": list(collected.keys()),
        "route_to_trainer": result.get("route_to_trainer", False),
    }

    if result.get("health_code"):
        response["health_code"] = result["health_code"]
    if result.get("profile_confirmed"):
        response["profile_saved"] = True

    _kyc_sessions[thread_id] = {
        "last_interaction_at": result.get("last_interaction_at", datetime.now(timezone.utc).isoformat()),
        "nudge_sent": False,
        "phone": req.phone_number,
    }

    return response


@app.get("/case5/kyc/live/history", tags=["Case 5 LIVE — KYC + Supabase"])
async def case5_live_history(thread_id: str = "kyc_test"):
    """Ver historial de conversación KYC (versión live con Supabase)."""
    config = {"configurable": {"thread_id": thread_id}}

    try:
        state = await case5_live_graph.aget_state(config)
        if not state.values:
            return {"thread_id": thread_id, "messages": [], "note": "Thread vacío o no existe"}

        collected = state.values.get("collected_data", {})
        messages = [
            {
                "index": i,
                "role": "user" if isinstance(m, HumanMessage) else "kairos",
                "content": m.content,
            }
            for i, m in enumerate(state.values.get("messages", []))
        ]

        return {
            "thread_id": thread_id,
            "total_messages": len(messages),
            "current_turn": state.values.get("current_turn", 0),
            "collected_data": collected,
            "messages": messages,
        }
    except Exception:
        return {"thread_id": thread_id, "messages": [], "note": "Thread no encontrado"}


# ═══════════════ CASE 6 — Unified Agent Kairos ═══════════════

@app.post("/case6/chat", tags=["Case 6 — Unified Agent Kairos"])
async def case6_chat(req: Case6ChatRequest):
    """Agente unificado Kairos — reemplaza MAIN_FLOW de n8n.

    El agente carga contexto del usuario (plan, sesiones, tareas pendientes),
    decide qué hacer, y responde usando tools si es necesario.

    Usa `phone_number` como identificador de thread (un thread por usuario).
    """
    # FR-013: Filter WhatsApp status messages
    if not req.message or not req.message.strip():
        return {"response": "", "thread_id": f"case6_{req.phone_number}", "filtered": True}

    thread_id = f"case6_{req.phone_number}"
    config = {"configurable": {"thread_id": thread_id}}

    # Use live graph (with Supabase tools) when SUPABASE_URL is configured
    graph = case6_live_graph if os.getenv("SUPABASE_URL") else case6_graph

    result = None
    for attempt in range(2):
        try:
            result = await graph.ainvoke(
                {
                    "messages": [HumanMessage(content=req.message)],
                    "phone_number": req.phone_number,
                    "display_name": req.display_name,
                },
                config,
            )
            break
        except Exception as e:
            logger.error(
                f"[API] Agent error for {req.phone_number} (attempt {attempt + 1}/2): "
                f"{type(e).__name__}: {e}\n{traceback.format_exc()}"
            )
            if attempt == 0:
                await asyncio.sleep(1)
            else:
                raise HTTPException(status_code=502, detail="Agent invocation failed after retries")

    ctx = result.get("user_context", {})

    return {
        "response": result.get("response", ""),
        "thread_id": thread_id,
        "is_new_user": ctx.get("is_new_user", False),
        "kyc_complete": ctx.get("kyc_complete", False),
    }


@app.get("/case6/history", tags=["Case 6 — Unified Agent Kairos"])
async def case6_history(phone_number: str = "573001234567"):
    """View conversation history for a Case 6 thread."""
    thread_id = f"case6_{phone_number}"
    config = {"configurable": {"thread_id": thread_id}}

    graph = case6_live_graph if os.getenv("SUPABASE_URL") else case6_graph

    try:
        state = await graph.aget_state(config)
        if not state.values:
            raise HTTPException(status_code=404, detail=f"No conversation found for phone_number {phone_number}")

        messages = [
            {
                "role": "user" if isinstance(m, HumanMessage) else "assistant",
                "content": m.content,
            }
            for m in state.values.get("messages", [])
            if hasattr(m, "content") and not isinstance(m, ToolMessage)
        ]

        return {
            "thread_id": thread_id,
            "message_count": len(messages),
            "messages": messages,
        }
    except HTTPException:
        raise
    except Exception:
        raise HTTPException(status_code=404, detail=f"No conversation found for phone_number {phone_number}")


@app.get("/case6/debug", tags=["Case 6 — Unified Agent Kairos"])
async def case6_debug(phone_number: str = "570000000004"):
    """Debug: muestra el state raw incluyendo tool_calls y ToolMessages."""
    thread_id = f"case6_{phone_number}"
    config = {"configurable": {"thread_id": thread_id}}

    graph = case6_live_graph if os.getenv("SUPABASE_URL") else case6_graph

    try:
        state = await graph.aget_state(config)
        if not state.values:
            raise HTTPException(status_code=404, detail="No state found")

        raw_messages = []
        for m in state.values.get("messages", []):
            msg_type = type(m).__name__
            entry = {"type": msg_type, "content": str(m.content)[:500]}
            if hasattr(m, "tool_calls") and m.tool_calls:
                entry["tool_calls"] = [{"name": tc["name"], "args": tc["args"]} for tc in m.tool_calls]
            if hasattr(m, "name"):
                entry["tool_name"] = m.name
            raw_messages.append(entry)

        return {
            "thread_id": thread_id,
            "user_context": state.values.get("user_context", {}),
            "raw_messages": raw_messages,
        }
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


# ═══════════════ WHATSAPP WEBHOOK (Direct) ═══════════════

WHATSAPP_API_URL = "https://graph.facebook.com/v21.0"
WHATSAPP_TOKEN = os.getenv("WHATSAPP_TOKEN", "")
WHATSAPP_VERIFY_TOKEN = os.getenv("WHATSAPP_VERIFY_TOKEN", "kairos-verify-token")
WHATSAPP_PHONE_NUMBER_ID = os.getenv("WHATSAPP_PHONE_NUMBER_ID", "")


async def _send_whatsapp_message(phone_number_id: str, to: str, text: str):
    """Send a text message via WhatsApp Business API."""
    url = f"{WHATSAPP_API_URL}/{phone_number_id}/messages"
    headers = {
        "Authorization": f"Bearer {WHATSAPP_TOKEN}",
        "Content-Type": "application/json",
    }
    body = {
        "messaging_product": "whatsapp",
        "to": to,
        "type": "text",
        "text": {"body": text},
    }
    async with httpx.AsyncClient() as client:
        resp = await client.post(url, headers=headers, json=body, timeout=30)
        if resp.status_code != 200:
            logger.error(f"WhatsApp send failed: {resp.status_code} {resp.text}")


async def _download_whatsapp_media(media_id: str) -> tuple[str, str] | None:
    """Download media from WhatsApp API, return (base64_data, mime_type) or None.

    Two-step process:
    1. GET media metadata from Graph API → obtain temporary download URL
    2. GET binary data from that URL
    """
    import base64

    headers = {"Authorization": f"Bearer {WHATSAPP_TOKEN}"}
    try:
        async with httpx.AsyncClient(timeout=30) as client:
            # Step 1: Get download URL
            meta_resp = await client.get(
                f"{WHATSAPP_API_URL}/{media_id}", headers=headers,
            )
            if meta_resp.status_code != 200:
                logger.error(f"[WA] Media meta failed: {meta_resp.status_code}")
                return None
            meta_json = meta_resp.json()
            download_url = meta_json.get("url")
            if not download_url:
                return None

            # Step 2: Download binary
            data_resp = await client.get(download_url, headers=headers)
            if data_resp.status_code != 200:
                logger.error(f"[WA] Media download failed: {data_resp.status_code}")
                return None

            b64 = base64.b64encode(data_resp.content).decode("utf-8")
            mime = meta_json.get("mime_type", "image/jpeg")
            return b64, mime
    except httpx.TimeoutException:
        logger.error(f"[WA] Media download timeout for {media_id}")
        return None
    except Exception as e:
        logger.error(f"[WA] Media download error: {e}")
        return None


async def _extract_message(data: dict) -> tuple[str | list, str, str, str, str] | None:
    """Extract (message_body, phone_from, display_name, phone_number_id, msg_id) from webhook payload.

    Returns None if the payload is not a valid user message (status update, audio, etc).
    message_body is str for text messages, list[dict] for image messages (langchain content blocks).
    """
    entry = data.get("entry", [])
    if not entry:
        return None

    changes = entry[0].get("changes", [])
    if not changes:
        return None

    value = changes[0].get("value", {})
    messages = value.get("messages", [])
    if not messages:
        return None

    msg = messages[0]
    msg_type = msg.get("type", "")
    msg_id = msg.get("id", "")  # WhatsApp message ID (wamid.xxx)

    # Filter audio and unsupported types
    if msg_type == "audio":
        return None

    phone_from = msg.get("from", "")
    if not phone_from:
        return None

    # Extract content based on message type
    body: str | list = ""
    if msg_type == "text":
        body = msg.get("text", {}).get("body", "")
    elif msg_type == "button":
        body = msg.get("button", {}).get("payload", "") or msg.get("button", {}).get("text", "")
    elif msg_type == "interactive":
        interactive = msg.get("interactive", {})
        body = (interactive.get("button_reply", {}).get("title", "")
                or interactive.get("list_reply", {}).get("title", ""))
    elif msg_type == "image":
        image_info = msg.get("image", {})
        media_id = image_info.get("id")
        if not media_id:
            return None
        media_result = await _download_whatsapp_media(media_id)
        if media_result is None:
            body = "[IMAGE_DOWNLOAD_FAILED]"
        else:
            b64_data, mime_type = media_result
            caption = image_info.get("caption", "").strip()
            content_parts: list[dict] = []
            if caption:
                content_parts.append({"type": "text", "text": caption})
            content_parts.append({
                "type": "image_url",
                "image_url": {"url": f"data:{mime_type};base64,{b64_data}"},
            })
            if not caption:
                content_parts.append({"type": "text", "text": "El usuario envió esta imagen."})
            body = content_parts
    else:
        body = msg.get("text", {}).get("body", "")

    if not body or (isinstance(body, str) and not body.strip()):
        return None

    # Get display name and phone_number_id
    contacts = value.get("contacts", [])
    display_name = contacts[0].get("profile", {}).get("name", "") if contacts else ""
    phone_number_id = value.get("metadata", {}).get("phone_number_id", WHATSAPP_PHONE_NUMBER_ID)

    return (body.strip() if isinstance(body, str) else body), phone_from, display_name, phone_number_id, msg_id


@app.get("/webhook", tags=["WhatsApp Webhook"])
async def whatsapp_verify(
    hub_mode: str = Query(None, alias="hub.mode"),
    hub_challenge: str = Query(None, alias="hub.challenge"),
    hub_verify_token: str = Query(None, alias="hub.verify_token"),
):
    """WhatsApp webhook verification (challenge-response)."""
    if hub_mode == "subscribe" and hub_verify_token == WHATSAPP_VERIFY_TOKEN:
        return int(hub_challenge)
    raise HTTPException(status_code=403, detail="Verification failed")


async def _process_message_background(
    message_body: str | list,
    phone_from: str,
    display_name: str,
    phone_number_id: str,
    msg_id: str,
):
    """Process a WhatsApp message in the background (after 200 already returned)."""
    import time as _time

    t_start = _time.monotonic()
    lock = _get_user_lock(phone_from)

    t_lock_wait = _time.monotonic()
    async with lock:
        t_lock_acquired = _time.monotonic()
        lock_wait_ms = int((t_lock_acquired - t_lock_wait) * 1000)
        if lock_wait_ms > 100:
            print(f"[TIMING] {phone_from} lock_wait={lock_wait_ms}ms", flush=True)

        thread_id = f"case6_{phone_from}"
        config = {"configurable": {"thread_id": thread_id}}
        graph = case6_live_graph if os.getenv("SUPABASE_URL") else case6_graph

        response_text = ""
        last_error = None
        for attempt in range(2):
            try:
                t_invoke = _time.monotonic()
                result = await graph.ainvoke(
                    {
                        "messages": [HumanMessage(content=message_body)],
                        "phone_number": phone_from,
                        "display_name": display_name,
                    },
                    config,
                )
                t_invoke_done = _time.monotonic()
                invoke_ms = int((t_invoke_done - t_invoke) * 1000)
                print(f"[TIMING] {phone_from} graph_invoke={invoke_ms}ms", flush=True)

                response_text = result.get("response", "")
                last_error = None
                break
            except Exception as e:
                last_error = e
                logger.error(
                    f"[WA-BG] Agent error for {phone_from} (attempt {attempt + 1}/2): "
                    f"{type(e).__name__}: {e}\n{traceback.format_exc()}"
                )
                if attempt == 0:
                    await asyncio.sleep(1)

        if last_error is not None:
            response_text = "Lo siento, tuve un problema procesando tu mensaje. Intenta de nuevo en un momento."

        if response_text:
            t_send = _time.monotonic()
            await _send_whatsapp_message(phone_number_id, phone_from, response_text)
            send_ms = int((_time.monotonic() - t_send) * 1000)
            print(f"[TIMING] {phone_from} wa_send={send_ms}ms", flush=True)

        total_ms = int((_time.monotonic() - t_start) * 1000)
        print(f"[TIMING] {phone_from} TOTAL={total_ms}ms msg_id={msg_id}", flush=True)


@app.post("/webhook", tags=["WhatsApp Webhook"])
async def whatsapp_webhook(request: Request):
    """Receive WhatsApp messages — return 200 immediately, process in background."""
    data = await request.json()

    extracted = await _extract_message(data)
    if not extracted:
        return {"status": "ignored"}

    message_body, phone_from, display_name, phone_number_id, msg_id = extracted

    # Dedup: skip if we've already seen this message ID (Supabase-backed)
    if msg_id and await _is_duplicate_message(msg_id, phone_from):
        logger.info(f"[WA] Duplicate {msg_id} from {phone_from} — skip")
        return {"status": "duplicate"}

    # Handle failed image downloads — short-circuit (cheap, no background needed)
    if message_body == "[IMAGE_DOWNLOAD_FAILED]":
        await _send_whatsapp_message(
            phone_number_id, phone_from,
            "No pude descargar tu imagen. ¿Puedes enviarla de nuevo?",
        )
        return {"status": "ok"}

    log_preview = message_body[:50] if isinstance(message_body, str) else "[image]"
    logger.info(f"[WA] {phone_from} ({display_name}): {log_preview} [msg_id={msg_id}]")

    # Fire-and-forget: process in background, return 200 immediately
    task = asyncio.create_task(
        _process_message_background(
            message_body, phone_from, display_name, phone_number_id, msg_id,
        )
    )
    _background_tasks.add(task)
    task.add_done_callback(_background_tasks.discard)

    return {"status": "ok"}


# ═══════════════ HEALTH CHECK ═══════════════

@app.get("/", tags=["Health"])
async def root():
    """Health check + lista de endpoints disponibles."""
    return {
        "status": "ok",
        "project": "LangGraph Skeleton — GymBot",
        "available_users": ["camilo-001", "ana-002", "carlos-003"],
        "endpoints": {
            "POST /case1": "Basic Graph + Gemini summary",
            "POST /case2": "Conditional routing + Gemini exercise selection",
            "POST /case3": "Tools + Agent loop (mock data)",
            "POST /case3/live": "Tools + Agent loop (SUPABASE REAL — 1,657 exercises)",
            "POST /case4/chat": "Multi-turn KYC conversation",
            "GET /case4/history?thread_id=X": "View conversation history",
            "POST /case4/live/chat": "KYC multi-turn + Supabase workout generation",
            "GET /case4/live/history?thread_id=X": "View conversation history (live)",
            "POST /case5/kyc/chat": "Onboarding KYC (5-turn, 10 fields)",
            "GET /case5/kyc/history?thread_id=X": "View KYC conversation history",
            "GET /case5/kyc/status?phone_number=X": "Check KYC session status",
            "POST /case5/kyc/live/chat": "Onboarding KYC + Supabase persistence",
            "GET /case5/kyc/live/history?thread_id=X": "View KYC conversation history (live)",
            "POST /case6/chat": "Unified Agent Kairos (context + tools)",
            "GET /case6/history?phone_number=X": "View Case 6 conversation history",
            "GET /webhook": "WhatsApp webhook verification",
            "POST /webhook": "WhatsApp → Kairos → WhatsApp (direct, no n8n)",
            "GET /docs": "Swagger UI (interactive docs)",
        },
    }
