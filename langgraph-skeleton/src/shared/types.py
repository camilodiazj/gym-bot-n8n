"""TypedDicts that mirror GymBot's Supabase schema (simplified)."""

from typing import TypedDict


class UserProfile(TypedDict):
    user_id: str
    full_name: str
    primary_goal: str
    health_status: str  # A, B, C, D, E
    level: str  # Principiante, Intermedio, Avanzado
    days_available: int


class Exercise(TypedDict):
    exercise_id: str
    spanish_name: str
    pattern: str
    role: str  # compound, isolation, core
    main_muscle: str
    level: str
    equipment: str


class DayTemplate(TypedDict):
    template_name: str
    patterns: list[str]  # e.g. ["push", "squat", "pull", "core"]


class SetProfile(TypedDict):
    goal: str
    role: str  # compound, isolation, core
    week: int
    sets: int
    reps: str  # e.g. "8-10"
    rir: str  # e.g. "1-2"
    rest_seconds: int
