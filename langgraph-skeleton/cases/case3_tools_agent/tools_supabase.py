"""Supabase-backed tools for Case 3 — query real GymBot database.

These tools replace the mock versions, querying Supabase's
1,657 real exercises and set_profiles tables.
"""

import json
from langchain_core.tools import tool

from src.shared.supabase_client import supabase_query


@tool
async def get_exercises_by_pattern(pattern: str, level: str) -> str:
    """Obtiene ejercicios reales de Supabase por patrón de movimiento y nivel.

    Args:
        pattern: Movement pattern. Options: push_h, push_v, pull_h, pull_v,
                 squat, hinge, lunge, core, arm, accessory
        level: User level. Options: Principiante, Intermedio, Avanzado
    """
    rows = await supabase_query(
        table="exercises",
        select="exercise_id,spanish_name,pattern,role,main_muscle,level,equipment",
        filters={"pattern": f"eq.{pattern}", "level": f"eq.{level}"},
        limit=15,
    )
    if not rows:
        return json.dumps({"error": f"No exercises found for pattern={pattern}, level={level}"})
    return json.dumps(rows, ensure_ascii=False)


@tool
async def get_set_profile(goal: str, role: str, week: int = 1) -> str:
    """Obtiene parámetros de carga reales de Supabase (sets, reps, RIR, descanso).

    Args:
        goal: Training goal. Options: Ganar masa muscular, Bajar grasa, Mejorar fuerza
        role: Exercise role. Options: compound, isolation, core
        week: Mesocycle week (1-4). Default: 1
    """
    rows = await supabase_query(
        table="set_profiles",
        select="goal,level,week,role,sets,reps,rir,rest_sec,tempo",
        filters={
            "goal": f"eq.{goal}",
            "role": f"eq.{role}",
            "week": f"eq.{week}",
        },
    )
    if not rows:
        return json.dumps({"error": f"No set profile for goal={goal}, role={role}, week={week}"})
    return json.dumps(rows, ensure_ascii=False)


@tool
async def search_exercises(muscle: str, equipment: str = "") -> str:
    """Busca ejercicios por músculo principal y opcionalmente por equipamiento.

    Args:
        muscle: Main muscle group. Examples: Chest, Back, Quads, Hamstrings,
                Shoulders, Biceps, Triceps, Glutes, Abs, Calfs
        equipment: Optional equipment filter. Examples: barbell, dumbbell,
                   machine, cable, bodyweight, band
    """
    filters = {"main_muscle": f"eq.{muscle}"}
    if equipment:
        filters["equipment"] = f"eq.{equipment}"

    rows = await supabase_query(
        table="exercises",
        select="exercise_id,spanish_name,pattern,role,main_muscle,level,equipment",
        filters=filters,
        limit=15,
    )
    if not rows:
        return json.dumps({"error": f"No exercises found for muscle={muscle}"})
    return json.dumps(rows, ensure_ascii=False)
