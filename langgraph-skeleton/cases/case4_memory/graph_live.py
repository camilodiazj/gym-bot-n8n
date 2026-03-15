"""Case 4 LIVE: Multi-turn KYC + Supabase workout generation.

Combines Case 4's multi-turn KYC conversation (InMemorySaver + thread_id)
with Case 3 Live's Supabase tools. When KYC completes, Gemini uses real
exercise data (1,657 exercises) to generate a personalized workout plan.

Flow:
  Phase 1 (KYC) — Kairos collects: objetivo, dias, nivel
  Phase 2 (Workout) — Agent loop: exercise_selector ↔ ToolNode (Supabase)
"""

import operator
from typing import Annotated
from typing_extensions import TypedDict

from langchain_core.messages import BaseMessage, HumanMessage, SystemMessage, AIMessage
from langgraph.checkpoint.memory import InMemorySaver
from langgraph.graph import StateGraph, START, END
from langgraph.prebuilt import ToolNode

from src.shared.llm import get_llm
from cases.case3_tools_agent.tools_supabase import (
    get_exercises_by_pattern,
    get_set_profile,
    search_exercises,
)


# ═══════════════ STATE ═══════════════

class Case4LiveState(TypedDict):
    messages: Annotated[list[BaseMessage], operator.add]
    is_complete: bool
    response: str
    workout_plan: str


# ═══════════════ TOOLS ═══════════════

TOOLS = [get_exercises_by_pattern, get_set_profile, search_exercises]


# ═══════════════ SYSTEM PROMPTS ═══════════════

KYC_SYSTEM_PROMPT = (
    "Eres Kairos, un entrenador personal virtual amigable y motivador. "
    "Necesitas recolectar 3 datos del usuario para crear su rutina personalizada:\n"
    "1) Objetivo de entrenamiento (ganar masa muscular, bajar grasa, mejorar fuerza)\n"
    "2) Cuantos dias por semana puede entrenar (2-6)\n"
    "3) Nivel de experiencia (principiante, intermedio, avanzado)\n\n"
    "Reglas:\n"
    "- Pregunta UN dato a la vez, de forma conversacional y motivadora.\n"
    "- Si el usuario da varios datos en un mensaje, registralos todos.\n"
    "- Cuando tengas los 3 datos, responde con la palabra clave DATOS_COMPLETOS "
    "seguida de un resumen de los datos recolectados.\n"
    "- Responde siempre en espanol.\n"
    "- Se breve (2-3 oraciones maximo por respuesta)."
)

WORKOUT_SYSTEM_PROMPT = (
    "Eres un entrenador personal experto con acceso a una base de datos real "
    "de 1,657 ejercicios en Supabase.\n\n"
    "HERRAMIENTAS DISPONIBLES:\n"
    "1. `get_exercises_by_pattern(pattern, level)` — Busca ejercicios por patron "
    "de movimiento (push_h, push_v, pull_h, pull_v, squat, hinge, lunge, core, arm, accessory) "
    "y nivel del usuario.\n"
    "2. `get_set_profile(goal, role, week)` — Obtiene parametros de carga "
    "(sets, reps, RIR, descanso) segun objetivo y rol del ejercicio.\n"
    "3. `search_exercises(muscle, equipment)` — Busca ejercicios por musculo "
    "principal y equipamiento.\n\n"
    "INSTRUCCIONES:\n"
    "- Basandote en los datos del KYC (objetivo, dias, nivel), "
    "busca ejercicios apropiados usando las tools.\n"
    "- Obtiene los parametros de carga para compound y core.\n"
    "- Genera la rutina formateada con ejercicio, sets, reps, RIR y descanso.\n"
    "- Responde en espanol."
)


# ═══════════════ NODES ═══════════════

async def kyc_agent(state: Case4LiveState) -> dict:
    """Gemini conducts the KYC conversation, asking for data progressively."""
    llm = get_llm(temperature=0.7)

    messages = [SystemMessage(content=KYC_SYSTEM_PROMPT)] + state["messages"]
    response = await llm.ainvoke(messages)

    is_complete = "DATOS_COMPLETOS" in response.content.upper()

    return {
        "messages": [response],
        "is_complete": is_complete,
        "response": response.content,
    }


async def prepare_workout_context(state: Case4LiveState) -> dict:
    """Transition from KYC to workout generation.

    Injects the workout system prompt so the agent can now call tools.
    """
    # Build a user message summarizing what was collected
    transition_msg = HumanMessage(content=(
        "El usuario ya completo el KYC. Ahora genera su rutina de entrenamiento "
        "usando las herramientas disponibles para buscar ejercicios reales de la "
        "base de datos. Selecciona ejercicios de al menos 3 patrones de movimiento "
        "diferentes y obtiene los parametros de carga."
    ))

    return {
        "messages": [SystemMessage(content=WORKOUT_SYSTEM_PROMPT), transition_msg],
    }


async def exercise_selector(state: Case4LiveState) -> dict:
    """Gemini agent — decides whether to call Supabase tools or respond."""
    llm = get_llm(temperature=0.3)
    model_with_tools = llm.bind_tools(TOOLS)
    response = await model_with_tools.ainvoke(state["messages"])
    return {"messages": [response]}


def format_workout(state: Case4LiveState) -> dict:
    """Extract the final workout from the last AI message."""
    last_msg = state["messages"][-1]
    content = last_msg.content if hasattr(last_msg, "content") else str(last_msg)
    return {
        "workout_plan": content,
        "response": content,
    }


# ═══════════════ ROUTING ═══════════════

def check_complete(state: Case4LiveState) -> str:
    """After KYC: route to workout generation if complete, otherwise END."""
    if state.get("is_complete", False):
        return "prepare_workout_context"
    return END


def should_continue(state: Case4LiveState) -> str:
    """Agent loop: check if Gemini wants to call more tools."""
    last_msg = state["messages"][-1]
    if isinstance(last_msg, AIMessage) and last_msg.tool_calls:
        return "tool_node"
    return "format_workout"


# ═══════════════ BUILD GRAPH ═══════════════

def build_case4_live_graph():
    """Build Case 4 Live graph: KYC multi-turn + Supabase workout generation.

    Graph structure:
      START → kyc_agent → check_complete
        if not complete → END (wait for next message)
        if complete → prepare_workout_context → exercise_selector ↔ tool_node
          when done → format_workout → END
    """
    workflow = StateGraph(Case4LiveState)

    # Phase 1: KYC nodes
    workflow.add_node("kyc_agent", kyc_agent)

    # Phase 2: Workout generation nodes
    workflow.add_node("prepare_workout_context", prepare_workout_context)
    workflow.add_node("exercise_selector", exercise_selector)
    workflow.add_node("tool_node", ToolNode(TOOLS))
    workflow.add_node("format_workout", format_workout)

    # Phase 1 edges
    workflow.add_edge(START, "kyc_agent")
    workflow.add_conditional_edges(
        "kyc_agent",
        check_complete,
        ["prepare_workout_context", END],
    )

    # Phase 2 edges
    workflow.add_edge("prepare_workout_context", "exercise_selector")
    workflow.add_conditional_edges(
        "exercise_selector",
        should_continue,
        {"tool_node": "tool_node", "format_workout": "format_workout"},
    )
    workflow.add_edge("tool_node", "exercise_selector")
    workflow.add_edge("format_workout", END)

    # Compile with checkpointer for multi-turn memory
    checkpointer = InMemorySaver()
    return workflow.compile(checkpointer=checkpointer)
