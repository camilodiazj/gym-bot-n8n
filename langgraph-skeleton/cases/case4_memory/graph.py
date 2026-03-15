"""Case 4: Checkpointing + Multi-turn Memory with Gemini KYC Agent.

Demonstrates:
- InMemorySaver checkpointer for conversation persistence
- thread_id for isolated conversations
- Gemini conducts multi-turn KYC collecting user data progressively
- Conditional routing: check_complete → kyc_agent or generate_plan
"""

import operator
from typing import Annotated
from typing_extensions import TypedDict

from langchain_core.messages import BaseMessage, HumanMessage, SystemMessage
from langgraph.checkpoint.memory import InMemorySaver
from langgraph.graph import StateGraph, START, END

from src.shared.llm import get_llm


# --- State ---

class Case4State(TypedDict):
    messages: Annotated[list[BaseMessage], operator.add]
    collected_data: dict
    is_complete: bool
    response: str
    workout_plan: str


# --- Nodes ---

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


async def kyc_agent(state: Case4State) -> dict:
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


async def generate_plan(state: Case4State) -> dict:
    """Generate a workout plan summary once all KYC data is collected."""
    llm = get_llm(temperature=0.5)

    prompt = (
        f"Basandote en la conversacion anterior, genera un resumen breve del plan "
        f"de entrenamiento recomendado para este usuario. "
        f"Incluye: objetivo, dias por semana, nivel, y 3-4 ejercicios sugeridos. "
        f"Responde en espanol, maximo 5 oraciones."
    )

    messages = [SystemMessage(content=KYC_SYSTEM_PROMPT)] + state["messages"]
    messages.append(HumanMessage(content=prompt))
    response = await llm.ainvoke(messages)

    return {
        "workout_plan": response.content,
        "response": response.content,
    }


# --- Routing ---

def check_complete(state: Case4State) -> str:
    """Route to generate_plan if KYC is complete, otherwise END (wait for next message)."""
    if state.get("is_complete", False):
        return "generate_plan"
    return END


# --- Graph ---

def build_case4_graph():
    """Build the Case 4 graph with InMemorySaver checkpointer."""
    workflow = StateGraph(Case4State)

    # Nodes
    workflow.add_node("kyc_agent", kyc_agent)
    workflow.add_node("generate_plan", generate_plan)

    # Edges
    workflow.add_edge(START, "kyc_agent")
    workflow.add_conditional_edges(
        "kyc_agent",
        check_complete,
        ["generate_plan", END],
    )
    workflow.add_edge("generate_plan", END)

    # Compile with checkpointer for multi-turn memory
    checkpointer = InMemorySaver()
    graph = workflow.compile(checkpointer=checkpointer)

    return graph
