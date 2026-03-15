"""Mock exercises mirroring GymBot's exercises table."""

from src.shared.types import Exercise

MOCK_EXERCISES: list[Exercise] = [
    # --- PUSH ---
    {
        "exercise_id": "ex_bench_press",
        "spanish_name": "Press de banca con barra",
        "pattern": "push",
        "role": "compound",
        "main_muscle": "Chest",
        "level": "Intermedio",
        "equipment": "barbell",
    },
    {
        "exercise_id": "ex_overhead_press",
        "spanish_name": "Press militar con barra",
        "pattern": "push",
        "role": "compound",
        "main_muscle": "Shoulders",
        "level": "Intermedio",
        "equipment": "barbell",
    },
    {
        "exercise_id": "ex_incline_db_press",
        "spanish_name": "Press inclinado con mancuernas",
        "pattern": "push",
        "role": "compound",
        "main_muscle": "Chest",
        "level": "Principiante",
        "equipment": "dumbbell",
    },
    # --- PULL ---
    {
        "exercise_id": "ex_barbell_row",
        "spanish_name": "Remo con barra",
        "pattern": "pull",
        "role": "compound",
        "main_muscle": "Back",
        "level": "Intermedio",
        "equipment": "barbell",
    },
    {
        "exercise_id": "ex_lat_pulldown",
        "spanish_name": "Jalón al pecho",
        "pattern": "pull",
        "role": "compound",
        "main_muscle": "Back",
        "level": "Principiante",
        "equipment": "machine",
    },
    {
        "exercise_id": "ex_pullup",
        "spanish_name": "Dominadas",
        "pattern": "pull",
        "role": "compound",
        "main_muscle": "Back",
        "level": "Avanzado",
        "equipment": "bodyweight",
    },
    # --- SQUAT ---
    {
        "exercise_id": "ex_barbell_squat",
        "spanish_name": "Sentadilla con barra",
        "pattern": "squat",
        "role": "compound",
        "main_muscle": "Quads",
        "level": "Intermedio",
        "equipment": "barbell",
    },
    {
        "exercise_id": "ex_leg_press",
        "spanish_name": "Prensa de piernas",
        "pattern": "squat",
        "role": "compound",
        "main_muscle": "Quads",
        "level": "Principiante",
        "equipment": "machine",
    },
    {
        "exercise_id": "ex_goblet_squat",
        "spanish_name": "Sentadilla goblet",
        "pattern": "squat",
        "role": "compound",
        "main_muscle": "Quads",
        "level": "Principiante",
        "equipment": "dumbbell",
    },
    # --- HIP HINGE ---
    {
        "exercise_id": "ex_deadlift",
        "spanish_name": "Peso muerto convencional",
        "pattern": "hip_hinge",
        "role": "compound",
        "main_muscle": "Hamstrings",
        "level": "Avanzado",
        "equipment": "barbell",
    },
    {
        "exercise_id": "ex_rdl",
        "spanish_name": "Peso muerto rumano",
        "pattern": "hip_hinge",
        "role": "compound",
        "main_muscle": "Hamstrings",
        "level": "Intermedio",
        "equipment": "barbell",
    },
    # --- CORE ---
    {
        "exercise_id": "ex_plank",
        "spanish_name": "Plancha frontal",
        "pattern": "core",
        "role": "core",
        "main_muscle": "Abs",
        "level": "Principiante",
        "equipment": "bodyweight",
    },
    {
        "exercise_id": "ex_cable_crunch",
        "spanish_name": "Crunch en polea",
        "pattern": "core",
        "role": "core",
        "main_muscle": "Abs",
        "level": "Intermedio",
        "equipment": "cable",
    },
    # --- ISOLATION ---
    {
        "exercise_id": "ex_bicep_curl",
        "spanish_name": "Curl de bíceps con mancuernas",
        "pattern": "isolation",
        "role": "isolation",
        "main_muscle": "Biceps",
        "level": "Principiante",
        "equipment": "dumbbell",
    },
    {
        "exercise_id": "ex_lateral_raise",
        "spanish_name": "Elevaciones laterales",
        "pattern": "isolation",
        "role": "isolation",
        "main_muscle": "Shoulders",
        "level": "Principiante",
        "equipment": "dumbbell",
    },
    {
        "exercise_id": "ex_tricep_pushdown",
        "spanish_name": "Extensión de tríceps en polea",
        "pattern": "isolation",
        "role": "isolation",
        "main_muscle": "Triceps",
        "level": "Principiante",
        "equipment": "cable",
    },
]


def get_exercises_by_pattern_and_level(
    pattern: str, level: str | None = None
) -> list[Exercise]:
    """Filter exercises by movement pattern and optionally by level."""
    result = [e for e in MOCK_EXERCISES if e["pattern"] == pattern]
    if level:
        result = [e for e in result if e["level"] == level]
    return result


def get_all_exercises() -> list[Exercise]:
    """Return all mock exercises."""
    return MOCK_EXERCISES.copy()
