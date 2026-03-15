"""Case 3 LIVE: Tools + Agent Loop with real Supabase data.

Same architecture as graph.py but tools query the real GymBot database
(1,657 exercises in Supabase) instead of mock data.
"""

import operator
from typing import Annotated, TypedDict

from langchain_core.messages import BaseMessage, HumanMessage, SystemMessage, AIMessage
from langgraph.graph import StateGraph, START, END
from langgraph.prebuilt import ToolNode

from src.shared.types import UserProfile, DayTemplate
from src.shared.llm import get_llm
from src.mock_data.users import get_user
from src.mock_data.templates import get_template
from cases.case3_tools_agent.tools_supabase import (
    get_exercises_by_pattern,
    get_set_profile,
    search_exercises,
)


# ═══════════════ STATE ═══════════════

class Case3LiveState(TypedDict):
    user_id: str
    profile: UserProfile | None
    template: DayTemplate | None
    messages: Annotated[list[BaseMessage], operator.add]
    workout_result: str


# ═══════════════ TOOLS ═══════════════

TOOLS = [get_exercises_by_pattern, get_set_profile, search_exercises]


# ═══════════════ NODES ═══════════════

def load_profile(state: Case3LiveState) -> dict:
    """Load user profile and template."""
    profile = get_user(state["user_id"])
    template = get_template("Full Body A")
    return {"profile": profile, "template": template}


def prepare_context(state: Case3LiveState) -> dict:
    """Build initial messages with system prompt and user request."""
    profile = state["profile"]
    template = state["template"]

    system_msg = SystemMessage(content=(
        "Eres un entrenador personal experto con acceso a una base de datos real "
        "de 1,657 ejercicios en Supabase.\n\n"
        "HERRAMIENTAS DISPONIBLES:\n"
        "1. `get_exercises_by_pattern(pattern, level)` — Busca ejercicios por patrón "
        "de movimiento (push_h, push_v, pull_h, pull_v, squat, hinge, lunge, core, arm, accessory) "
        "y nivel del usuario.\n"
        "2. `get_set_profile(goal, role, week)` — Obtiene parámetros de carga "
        "(sets, reps, RIR, descanso) según objetivo y rol del ejercicio.\n"
        "3. `search_exercises(muscle, equipment)` — Busca ejercicios por músculo "
        "principal y equipamiento.\n\n"
        "INSTRUCCIONES:\n"
        "- Busca ejercicios para CADA patrón del template.\n"
        "- Obtén los parámetros de carga para compound y core.\n"
        "- Genera la rutina formateada con ejercicio, sets, reps, RIR y descanso.\n"
        "- Responde en español."
    ))

    user_msg = HumanMessage(content=(
        f"Crea una rutina para {profile['full_name']}:\n"
        f"- Nivel: {profile['level']}\n"
        f"- Objetivo: {profile['primary_goal']}\n"
        f"- Template: {template['template_name']}\n"
        f"- Patrones requeridos: {', '.join(template['patterns'])}\n\n"
        f"Busca ejercicios reales de la base de datos para cada patrón."
    ))

    return {"messages": [system_msg, user_msg]}


async def exercise_selector(state: Case3LiveState) -> dict:
    """Gemini agent — decides whether to call tools or respond."""
    llm = get_llm(temperature=0.3)
    model_with_tools = llm.bind_tools(TOOLS)
    response = await model_with_tools.ainvoke(state["messages"])
    return {"messages": [response]}


def format_workout(state: Case3LiveState) -> dict:
    """Extract the final workout from the last AI message."""
    last_msg = state["messages"][-1]
    content = last_msg.content if hasattr(last_msg, "content") else str(last_msg)
    return {"workout_result": content}


# ═══════════════ ROUTING ═══════════════

def should_continue(state: Case3LiveState) -> str:
    last_msg = state["messages"][-1]
    if isinstance(last_msg, AIMessage) and last_msg.tool_calls:
        return "tools"
    return "done"


# ═══════════════ BUILD GRAPH ═══════════════

def build_case3_live_graph():
    """Build Case 3 graph with real Supabase tools."""
    workflow = StateGraph(Case3LiveState)

    workflow.add_node("load_profile", load_profile)
    workflow.add_node("prepare_context", prepare_context)
    workflow.add_node("exercise_selector", exercise_selector)
    workflow.add_node("tool_node", ToolNode(TOOLS))
    workflow.add_node("format_workout", format_workout)

    workflow.add_edge(START, "load_profile")
    workflow.add_edge("load_profile", "prepare_context")
    workflow.add_edge("prepare_context", "exercise_selector")

    workflow.add_conditional_edges(
        "exercise_selector",
        should_continue,
        {"tools": "tool_node", "done": "format_workout"},
    )
    workflow.add_edge("tool_node", "exercise_selector")
    workflow.add_edge("format_workout", END)

    return workflow.compile()
