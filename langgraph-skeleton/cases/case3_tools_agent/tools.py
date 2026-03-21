"""Tools for Case 3 — Gemini calls these to build workout routines.

Mirrors GymBot's n8n tools:
- get_exercises_by_pattern → GetExercisesByPattern (Postgres query)
- get_set_profile → GetSetProfiles (Postgres query)
"""

import json
from langchain_core.tools import tool
from src.mock_data.exercises import get_exercises_by_pattern_and_level
from src.mock_data.set_profiles import get_set_profile_data


@tool
def get_exercises_by_pattern(pattern: str, level: str) -> str:
    """Obtiene ejercicios disponibles por patrón de movimiento y nivel del usuario.

    Args:
        pattern: Patrón de movimiento (push, pull, squat, hip_hinge, core, isolation)
        level: Nivel del usuario (Principiante, Intermedio, Avanzado)

    Returns:
        JSON con la lista de ejercicios que coinciden.
    """
    exercises = get_exercises_by_pattern_and_level(pattern, level)
    if not exercises:
        # Try without level filter as fallback
        exercises = get_exercises_by_pattern_and_level(pattern)
    return json.dumps(exercises, ensure_ascii=False)


@tool
def get_set_profile(goal: str, role: str, week: int) -> str:
    """Obtiene los parámetros de carga (series, reps, RIR, descanso) para un ejercicio.

    Args:
        goal: Objetivo del usuario (Ganar masa muscular, Bajar grasa, Mejorar fuerza)
        role: Rol del ejercicio (compound, isolation, core)
        week: Número de semana (1-4)

    Returns:
        JSON con sets, reps, rir, rest_seconds.
    """
    profile = get_set_profile_data(goal, role, week)
    if profile:
        return json.dumps(profile, ensure_ascii=False)
    return json.dumps({"error": f"No set profile found for {goal}/{role}/week{week}"})
