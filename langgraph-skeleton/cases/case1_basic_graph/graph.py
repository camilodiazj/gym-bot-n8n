"""Case 1: Basic Graph — State + Nodes + Edges + Gemini Summary.

Demonstrates:
- StateGraph with TypedDict state
- 3 nodes in linear sequence
- addEdge for fixed connections
- Gemini generates a personalized profile summary

Diagram:
    START → load_profile → validate_profile → generate_summary → END
"""

from typing import TypedDict

from langgraph.graph import StateGraph, START, END

from src.shared.types import UserProfile
from src.shared.llm import get_llm
from src.mock_data.users import get_user


# ═══════════════ STATE ═══════════════

class Case1State(TypedDict):
    user_id: str  # Input
    profile: UserProfile | None  # Filled by load_profile
    is_valid: bool  # Filled by validate_profile
    validation_errors: list[str]  # Errors found
    summary: str  # Output — Gemini-generated summary


# ═══════════════ NODES ═══════════════

def load_profile(state: Case1State) -> dict:
    """Load user profile from mock data. Equivalent to n8n's GetUser node."""
    profile = get_user(state["user_id"])
    return {"profile": profile}


def validate_profile(state: Case1State) -> dict:
    """Validate profile fields. Equivalent to n8n's implicit KYC validations."""
    errors: list[str] = []
    profile = state.get("profile")

    if not profile:
        return {"is_valid": False, "validation_errors": ["User not found"]}

    if profile["health_status"] not in ("A", "B", "C", "D", "E"):
        errors.append(f"Invalid health_status: {profile['health_status']}")

    if profile["days_available"] <= 0:
        errors.append("days_available must be > 0")

    if profile["level"] not in ("Principiante", "Intermedio", "Avanzado"):
        errors.append(f"Invalid level: {profile['level']}")

    return {
        "is_valid": len(errors) == 0,
        "validation_errors": errors,
    }


async def generate_summary(state: Case1State) -> dict:
    """Generate a personalized summary using Gemini. First LLM usage!"""
    if not state.get("is_valid") or not state.get("profile"):
        return {"summary": "No se puede generar resumen: perfil inválido."}

    llm = get_llm(temperature=0.3)
    profile = state["profile"]

    response = await llm.ainvoke(
        f"Genera un resumen breve (2-3 oraciones) del perfil de este usuario de gym. "
        f"Nombre: {profile['full_name']}, Nivel: {profile['level']}, "
        f"Objetivo: {profile['primary_goal']}, Días disponibles: {profile['days_available']}, "
        f"Estado de salud: {profile['health_status']}. "
        f"Responde en español."
    )
    return {"summary": response.content}


# ═══════════════ BUILD GRAPH ═══════════════

def build_case1_graph():
    """Build and compile the Case 1 graph."""
    workflow = StateGraph(Case1State)

    # Add nodes
    workflow.add_node("load_profile", load_profile)
    workflow.add_node("validate_profile", validate_profile)
    workflow.add_node("generate_summary", generate_summary)

    # Add edges (linear sequence)
    workflow.add_edge(START, "load_profile")
    workflow.add_edge("load_profile", "validate_profile")
    workflow.add_edge("validate_profile", "generate_summary")
    workflow.add_edge("generate_summary", END)

    return workflow.compile()
