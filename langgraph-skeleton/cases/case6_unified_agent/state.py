"""State definitions for the Unified Agent Kairos (Case 6).

Defines UserContext (pre-loaded from Supabase), DraftRoutine (in-memory
workout draft), and UnifiedAgentState (the main graph state).
"""

import operator
from typing import Annotated
from typing_extensions import TypedDict

from langchain_core.messages import BaseMessage


class UserContext(TypedDict):
    """Snapshot of user state loaded from Supabase before each turn."""

    user_id: str | None
    full_name: str
    phone_number: str
    plan: dict | None  # {plan_id, goal, level, week_schedule, mesocycle_number, status, current_week}
    todays_sessions: list[dict]  # [{session_id, session_name, week, Completed, planned_day}]
    missed_sessions: list[dict]  # Last 3 days, Completed=false
    next_scheduled_session: dict | None  # {session_name, planned_day}
    pending_tasks: list[dict]  # [{task_id, task_type, session_name, status}]
    gym_profile: dict | None  # KYC data: {primary_goal, training_experience, days_available, training_environment, ...}
    is_new_user: bool
    kyc_complete: bool
    has_schedule: bool
    all_w4_completed: bool


class DraftExercise(TypedDict):
    """A single exercise in a draft routine."""

    exercise_id: str
    spanish_name: str
    pattern: str
    role: str  # compound | isolation | core
    sets: int
    reps: str  # "8-10"
    rir: str  # "1-2"
    rest_seconds: int
    exercise_order: int


class DraftDay(TypedDict):
    """A single day in a draft routine."""

    day_number: int
    title: str  # "Full Body A", "Upper Body B"
    exercises: list[DraftExercise]


class DraftRoutine(TypedDict):
    """In-memory workout draft built interactively before saving."""

    week_schedule: str  # "fb_3", "ul_4"
    goal: str
    level: str
    days: list[DraftDay]


class UnifiedAgentState(TypedDict):
    """Main state for the Case 6 unified agent graph."""

    messages: Annotated[list[BaseMessage], operator.add]
    phone_number: str
    display_name: str
    user_context: UserContext  # Populated by load_context each turn
    draft_routine: DraftRoutine | None  # In-memory draft (not persisted until approved)
    response: str  # Final output to user
