"""Case 2: Conditional Routing + Gemini exercise selection.

Demonstrates:
- add_conditional_edges for routing by health_status
- Fan-in convergence (3 restriction nodes → 1 selection node)
- Gemini selects best exercises from filtered pool

Diagram:
                            ┌─ restrict_upper_body ─┐
    START → load_profile →  ├─ restrict_lower_body ─├→ gemini_select → END
                            └─ no_restrictions ─────┘
"""

import json
from typing import TypedDict

from langgraph.graph import StateGraph, START, END

from src.shared.types import UserProfile, Exercise
from src.shared.llm import get_llm
from src.mock_data.users import get_user
from src.mock_data.exercises import get_all_exercises


# ═══════════════ STATE ═══════════════

class Case2State(TypedDict):
    user_id: str
    profile: UserProfile | None
    available_exercises: list[Exercise]  # Pool after health filtering
    selected_exercises: list[Exercise]  # Gemini's selection
    route_taken: str  # For debugging


# ═══════════════ NODES ═══════════════

def load_profile(state: Case2State) -> dict:
    """Load user profile from mock data."""
    profile = get_user(state["user_id"])
    return {"profile": profile}


def no_restrictions(state: Case2State) -> dict:
    """Health A: full exercise pool, no restrictions."""
    return {
        "available_exercises": get_all_exercises(),
        "route_taken": "no_restrictions",
    }


def restrict_upper_body(state: Case2State) -> dict:
    """Health C: remove overhead pressing exercises."""
    all_exercises = get_all_exercises()
    filtered = [
        e for e in all_exercises
        if not (
            e["pattern"] == "push"
            and e["main_muscle"] == "Shoulders"
        )
    ]
    return {
        "available_exercises": filtered,
        "route_taken": "restrict_upper_body",
    }


def restrict_lower_body(state: Case2State) -> dict:
    """Health B: remove high-impact lower body exercises."""
    all_exercises = get_all_exercises()
    filtered = [
        e for e in all_exercises
        if not (
            e["pattern"] in ("squat", "hip_hinge")
            and e["equipment"] == "barbell"
        )
    ]
    return {
        "available_exercises": filtered,
        "route_taken": "restrict_lower_body",
    }


async def gemini_select_exercises(state: Case2State) -> dict:
    """Gemini selects the best 4-6 exercises from the filtered pool."""
    llm = get_llm(temperature=0.3)
    profile = state["profile"]
    exercises = state["available_exercises"]

    exercises_text = "\n".join(
        f"- {e['exercise_id']}: {e['spanish_name']} ({e['pattern']}, {e['role']}, {e['main_muscle']}, nivel: {e['level']})"
        for e in exercises
    )

    response = await llm.ainvoke(
        f"Eres un entrenador personal. Selecciona los 4-6 mejores ejercicios para:\n"
        f"- Usuario: {profile['full_name']}\n"
        f"- Nivel: {profile['level']}\n"
        f"- Objetivo: {profile['primary_goal']}\n"
        f"- Días disponibles: {profile['days_available']}\n\n"
        f"Ejercicios disponibles:\n{exercises_text}\n\n"
        f"Responde SOLO con los exercise_id seleccionados, uno por línea, sin explicación."
    )

    selected_ids = [
        line.strip().lstrip("- ")
        for line in response.content.strip().split("\n")
        if line.strip()
    ]
    selected = [
        e for e in exercises
        if e["exercise_id"] in selected_ids
    ]

    # Fallback: if Gemini didn't match IDs exactly, take first 5
    if not selected:
        selected = exercises[:5]

    return {"selected_exercises": selected}


# ═══════════════ ROUTING ═══════════════

def route_by_health(state: Case2State) -> str:
    """Route based on health_status. Equivalent to n8n's If/Switch nodes."""
    health = state["profile"]["health_status"] if state.get("profile") else "A"
    return {
        "A": "no_restrictions",
        "B": "restrict_lower_body",
        "C": "restrict_upper_body",
        "D": "restrict_lower_body",  # Simplified: spine → treat like lower body
        "E": "restrict_upper_body",  # Simplified: special → treat like upper body
    }.get(health, "no_restrictions")


# ═══════════════ BUILD GRAPH ═══════════════

def build_case2_graph():
    """Build and compile the Case 2 graph."""
    workflow = StateGraph(Case2State)

    # Nodes
    workflow.add_node("load_profile", load_profile)
    workflow.add_node("no_restrictions", no_restrictions)
    workflow.add_node("restrict_upper_body", restrict_upper_body)
    workflow.add_node("restrict_lower_body", restrict_lower_body)
    workflow.add_node("gemini_select", gemini_select_exercises)

    # Entry
    workflow.add_edge(START, "load_profile")

    # Conditional routing by health_status
    workflow.add_conditional_edges(
        "load_profile",
        route_by_health,
        ["no_restrictions", "restrict_upper_body", "restrict_lower_body"],
    )

    # Fan-in: all restriction nodes converge to gemini_select
    workflow.add_edge("no_restrictions", "gemini_select")
    workflow.add_edge("restrict_upper_body", "gemini_select")
    workflow.add_edge("restrict_lower_body", "gemini_select")

    # Terminal
    workflow.add_edge("gemini_select", END)

    return workflow.compile()
