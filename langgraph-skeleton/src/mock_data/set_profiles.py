"""Mock set profiles mirroring GymBot's set_profiles table."""

from src.shared.types import SetProfile

MOCK_SET_PROFILES: list[SetProfile] = [
    # --- Ganar masa muscular ---
    {"goal": "Ganar masa muscular", "role": "compound", "week": 1, "sets": 3, "reps": "8-10", "rir": "1-2", "rest_seconds": 150},
    {"goal": "Ganar masa muscular", "role": "compound", "week": 2, "sets": 3, "reps": "8-10", "rir": "1-2", "rest_seconds": 150},
    {"goal": "Ganar masa muscular", "role": "compound", "week": 3, "sets": 4, "reps": "6-8", "rir": "0-1", "rest_seconds": 180},
    {"goal": "Ganar masa muscular", "role": "compound", "week": 4, "sets": 2, "reps": "10-12", "rir": "3-4", "rest_seconds": 120},
    {"goal": "Ganar masa muscular", "role": "isolation", "week": 1, "sets": 3, "reps": "12-15", "rir": "1-2", "rest_seconds": 90},
    {"goal": "Ganar masa muscular", "role": "core", "week": 1, "sets": 3, "reps": "12-15", "rir": "1-2", "rest_seconds": 60},
    # --- Bajar grasa ---
    {"goal": "Bajar grasa", "role": "compound", "week": 1, "sets": 3, "reps": "12-15", "rir": "1-2", "rest_seconds": 90},
    {"goal": "Bajar grasa", "role": "isolation", "week": 1, "sets": 2, "reps": "15-20", "rir": "1-2", "rest_seconds": 60},
    {"goal": "Bajar grasa", "role": "core", "week": 1, "sets": 3, "reps": "15-20", "rir": "1-2", "rest_seconds": 45},
    # --- Mejorar fuerza ---
    {"goal": "Mejorar fuerza", "role": "compound", "week": 1, "sets": 5, "reps": "3-5", "rir": "1-2", "rest_seconds": 240},
    {"goal": "Mejorar fuerza", "role": "isolation", "week": 1, "sets": 3, "reps": "8-10", "rir": "2-3", "rest_seconds": 90},
    {"goal": "Mejorar fuerza", "role": "core", "week": 1, "sets": 3, "reps": "8-12", "rir": "1-2", "rest_seconds": 60},
]


def get_set_profile_data(goal: str, role: str, week: int = 1) -> SetProfile | None:
    """Find set profile matching goal, role, and week."""
    for sp in MOCK_SET_PROFILES:
        if sp["goal"] == goal and sp["role"] == role and sp["week"] == week:
            return sp
    return None
