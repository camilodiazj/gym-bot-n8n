"""Case 3: Tools + Agent Loop with Gemini.

Demonstrates:
- tool() decorator with Zod-like schemas
- ToolNode for automatic tool execution
- Agent ↔ ToolNode loop with should_continue
- model.bind_tools() to enable tool calling

Diagram:
    START → load_profile → prepare_context
                              ↓
                     exercise_selector (Gemini)
                        ↓           ↑
                   tool_calls?    tool_node
                        ↓ No         (executes tools)
                   format_workout → END
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
from cases.case3_tools_agent.tools import get_exercises_by_pattern, get_set_profile


# ═══════════════ STATE ═══════════════

class Case3State(TypedDict):
    user_id: str
    profile: UserProfile | None
    template: DayTemplate | None
    messages: Annotated[list[BaseMessage], operator.add]  # Reducer: append
    workout_result: str  # Final formatted output


# ═══════════════ TOOLS ═══════════════

TOOLS = [get_exercises_by_pattern, get_set_profile]


# ═══════════════ NODES ═══════════════

def load_profile(state: Case3State) -> dict:
    """Load user profile and template."""
    profile = get_user(state["user_id"])
    template = get_template("Full Body A")
    return {"profile": profile, "template": template}


def prepare_context(state: Case3State) -> dict:
    """Build the initial messages with system prompt and user request."""
    profile = state["profile"]
    template = state["template"]

    system_msg = SystemMessage(content=(
        "Eres un entrenador personal experto. Tu tarea es crear una rutina de ejercicios.\n\n"
        "INSTRUCCIONES:\n"
        "1. Usa la herramienta 'get_exercises_by_pattern' para buscar ejercicios por patrón y nivel.\n"
        "2. Usa la herramienta 'get_set_profile' para obtener los parámetros de carga.\n"
        "3. Una vez tengas los datos, genera la rutina formateada.\n\n"
        "IMPORTANTE: Llama las herramientas ANTES de generar la rutina final."
    ))

    user_msg = HumanMessage(content=(
        f"Crea una rutina para {profile['full_name']}:\n"
        f"- Nivel: {profile['level']}\n"
        f"- Objetivo: {profile['primary_goal']}\n"
        f"- Template: {template['template_name']}\n"
        f"- Patrones requeridos: {', '.join(template['patterns'])}\n\n"
        f"Busca ejercicios para cada patrón y obtén los parámetros de carga."
    ))

    return {"messages": [system_msg, user_msg]}


async def exercise_selector(state: Case3State) -> dict:
    """Gemini agent node — decides whether to call tools or respond."""
    llm = get_llm(temperature=0.3)
    model_with_tools = llm.bind_tools(TOOLS)

    response = await model_with_tools.ainvoke(state["messages"])
    return {"messages": [response]}


def format_workout(state: Case3State) -> dict:
    """Extract the final workout from the last AI message."""
    last_msg = state["messages"][-1]
    content = last_msg.content if hasattr(last_msg, "content") else str(last_msg)
    return {"workout_result": content}


# ═══════════════ ROUTING ═══════════════

def should_continue(state: Case3State) -> str:
    """Check if Gemini wants to call more tools or is done."""
    last_msg = state["messages"][-1]
    if isinstance(last_msg, AIMessage) and last_msg.tool_calls:
        return "tools"
    return "done"


# ═══════════════ BUILD GRAPH ═══════════════

def build_case3_graph():
    """Build and compile the Case 3 graph with agent loop."""
    workflow = StateGraph(Case3State)

    # Nodes
    workflow.add_node("load_profile", load_profile)
    workflow.add_node("prepare_context", prepare_context)
    workflow.add_node("exercise_selector", exercise_selector)
    workflow.add_node("tool_node", ToolNode(TOOLS))
    workflow.add_node("format_workout", format_workout)

    # Edges
    workflow.add_edge(START, "load_profile")
    workflow.add_edge("load_profile", "prepare_context")
    workflow.add_edge("prepare_context", "exercise_selector")

    # Agent loop: exercise_selector ↔ tool_node
    workflow.add_conditional_edges(
        "exercise_selector",
        should_continue,
        {
            "tools": "tool_node",
            "done": "format_workout",
        },
    )
    workflow.add_edge("tool_node", "exercise_selector")  # Back to agent after tool

    # Terminal
    workflow.add_edge("format_workout", END)

    return workflow.compile()
