"""KYCState — typed state for the Onboarding KYC StateGraph.

All 17 fields per data-model.md entity #1. Messages use an append
reducer so each node can return `{"messages": [new_msg]}`.
"""

import operator
from typing import Annotated
from typing_extensions import TypedDict

from langchain_core.messages import BaseMessage


# The KYC fields collected across 5 turns
KYC_FIELDS: dict[int, list[str]] = {
    1: ["primary_goal"],
    2: ["training_experience", "days_available", "preferred_schedule"],
    3: ["training_environment", "home_equipment", "training_style", "cardio_type"],
    4: ["biological_sex", "age", "height_cm", "weight_kg"],
    5: ["health_status", "priority_muscles", "disliked_exercises"],
}

ALL_KYC_FIELDS: set[str] = {f for fields in KYC_FIELDS.values() for f in fields}

# Fields that are always required (some are optional/conditional)
REQUIRED_FIELDS: set[str] = ALL_KYC_FIELDS - {
    "home_equipment",       # Only required for HOME users
    "training_style",       # Optional (defaults to Mixto)
    "cardio_type",          # Optional (defaults to No)
    "priority_muscles",     # Optional (defaults to empty)
    "disliked_exercises",   # Optional (defaults to empty)
}


class KYCState(TypedDict):
    """State flowing through the KYC graph nodes."""

    # Conversation history (reducer: append)
    messages: Annotated[list[BaseMessage], operator.add]

    # User identity
    phone_number: str
    display_name: str
    is_new_user: bool

    # KYC progress
    current_turn: int          # 1-5, 0 = not started
    collected_data: dict       # {field_name: value}
    is_complete: bool

    # Health classification
    health_code: str           # A, B, C, D, E
    affected_zones: list[str]  # e.g. ["rodilla"]

    # Flow control
    awaiting_confirmation: bool
    profile_confirmed: bool
    needs_correction: bool
    correction_field: str

    # Output
    response: str
    route_to_trainer: bool

    # Session tracking
    nudge_sent: bool
    last_interaction_at: str   # ISO timestamp
    is_resumption: bool        # True if gap > 30 min since last interaction


def compute_current_turn(collected_data: dict) -> int:
    """Determine which turn the user is on based on collected fields.

    Returns the NEXT turn to ask (1-5), or 6 if all turns complete.
    """
    for turn in range(1, 6):
        turn_fields = KYC_FIELDS[turn]
        required = [f for f in turn_fields if f != "home_equipment"]
        if not all(f in collected_data for f in required):
            return turn
    return 6  # All turns complete


def is_kyc_complete(collected_data: dict) -> bool:
    """Check if all required KYC fields are collected.

    home_equipment is only required if training_environment == 'HOME'.
    """
    for field in REQUIRED_FIELDS:
        if field not in collected_data:
            return False
    # home_equipment required only for HOME users
    if collected_data.get("training_environment") == "HOME":
        if "home_equipment" not in collected_data:
            return False
    return True
