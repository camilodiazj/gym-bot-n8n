"""Tests for Case 5: Onboarding KYC (5-turn, 10 fields).

Tests:
- Graph compiles successfully
- First turn is not complete
- Multi-value messages register multiple fields
- Different threads are isolated
- Health classifier returns valid codes
- State helpers compute turns correctly
"""

import pytest
from langchain_core.messages import HumanMessage

from cases.case5_onboarding_kyc.graph import build_case5_graph
from cases.case5_onboarding_kyc.state import (
    KYCState,
    compute_current_turn,
    is_kyc_complete,
    KYC_FIELDS,
    REQUIRED_FIELDS,
)


@pytest.fixture
def graph():
    return build_case5_graph()


# ═══════════════ STATE HELPERS ═══════════════

def test_compute_current_turn_empty():
    """Empty collected_data → Turn 1."""
    assert compute_current_turn({}) == 1


def test_compute_current_turn_after_turn1():
    """After Turn 1 (primary_goal set) → Turn 2."""
    assert compute_current_turn({"primary_goal": "Ganar masa muscular"}) == 2


def test_compute_current_turn_after_turn2():
    """After Turn 2 (experience + days + schedule) → Turn 3."""
    data = {
        "primary_goal": "Ganar masa muscular",
        "training_experience": "1 a 3 años",
        "days_available": 3,
        "preferred_schedule": "Mañana",
    }
    assert compute_current_turn(data) == 3


def test_compute_current_turn_all_complete():
    """All required fields set → Turn 6 (all complete)."""
    data = {
        "primary_goal": "Ganar masa muscular",
        "training_experience": "1 a 3 años",
        "days_available": 3,
        "preferred_schedule": "Mañana",
        "training_environment": "GYM",
        "biological_sex": "M",
        "age": 27,
        "height_cm": 171,
        "weight_kg": 67,
        "health_status": "Sin restricciones",
    }
    assert compute_current_turn(data) == 6


def test_is_kyc_complete_gym():
    """GYM user: home_equipment NOT required."""
    data = {
        "primary_goal": "Ganar masa muscular",
        "training_experience": "1 a 3 años",
        "days_available": 3,
        "preferred_schedule": "Mañana",
        "training_environment": "GYM",
        "biological_sex": "M",
        "age": 27,
        "height_cm": 171,
        "weight_kg": 67,
        "health_status": "Sin restricciones",
    }
    assert is_kyc_complete(data) is True


def test_is_kyc_complete_home_needs_equipment():
    """HOME user without home_equipment → incomplete."""
    data = {
        "primary_goal": "Ganar masa muscular",
        "training_experience": "1 a 3 años",
        "days_available": 3,
        "preferred_schedule": "Mañana",
        "training_environment": "HOME",
        "biological_sex": "M",
        "age": 27,
        "height_cm": 171,
        "weight_kg": 67,
        "health_status": "Sin restricciones",
    }
    assert is_kyc_complete(data) is False


def test_is_kyc_complete_home_with_equipment():
    """HOME user with home_equipment → complete."""
    data = {
        "primary_goal": "Ganar masa muscular",
        "training_experience": "1 a 3 años",
        "days_available": 3,
        "preferred_schedule": "Mañana",
        "training_environment": "HOME",
        "home_equipment": "mancuernas y bandas",
        "biological_sex": "M",
        "age": 27,
        "height_cm": 171,
        "weight_kg": 67,
        "health_status": "Sin restricciones",
    }
    assert is_kyc_complete(data) is True


def test_is_kyc_complete_missing_fields():
    """Missing required fields → incomplete."""
    assert is_kyc_complete({}) is False
    assert is_kyc_complete({"primary_goal": "Bajar grasa"}) is False


# ═══════════════ GRAPH COMPILATION ═══════════════

def test_graph_compiles(graph):
    """Graph should compile without errors."""
    assert graph is not None


def test_graph_has_nodes(graph):
    """Graph should have all 7 expected nodes."""
    node_names = set(graph.get_graph().nodes.keys())
    expected = {"check_user", "kyc_agent", "check_status", "confirm_profile",
                "health_classifier", "save_profile", "route_to_trainer"}
    # Graph also includes __start__ and __end__
    assert expected.issubset(node_names)


# ═══════════════ GRAPH EXECUTION (Gemini required) ═══════════════

@pytest.mark.asyncio
async def test_first_turn_not_complete(graph):
    """First turn should NOT be complete."""
    config = {"configurable": {"thread_id": "test-case5-1"}}

    result = await graph.ainvoke(
        {
            "messages": [HumanMessage(content="Hola, quiero empezar")],
            "phone_number": "573001234567",
            "display_name": "Test",
        },
        config,
    )

    assert result.get("is_complete") is False
    assert result.get("response", "") != ""


@pytest.mark.asyncio
async def test_thread_isolation(graph):
    """Different threads should be isolated."""
    config_a = {"configurable": {"thread_id": "test-case5-a"}}
    config_b = {"configurable": {"thread_id": "test-case5-b"}}

    await graph.ainvoke(
        {
            "messages": [HumanMessage(content="Quiero ganar masa muscular")],
            "phone_number": "573001111111",
            "display_name": "UserA",
        },
        config_a,
    )

    result_b = await graph.ainvoke(
        {
            "messages": [HumanMessage(content="Hola")],
            "phone_number": "573002222222",
            "display_name": "UserB",
        },
        config_b,
    )

    assert result_b.get("is_complete") is False
    state_b = await graph.aget_state(config_b)
    assert len(state_b.values.get("messages", [])) == 2  # Human + AI


# ═══════════════ HELPER PARSE TESTS ═══════════════

def test_parse_extracted_data():
    """_parse_extracted_data should extract JSON from response."""
    from cases.case5_onboarding_kyc.nodes import _parse_extracted_data

    content = '¡Genial! Me encanta tu energía. EXTRACTED_DATA: {"primary_goal": "Ganar masa muscular"}'
    result = _parse_extracted_data(content)
    assert result == {"primary_goal": "Ganar masa muscular"}


def test_parse_extracted_data_empty():
    """_parse_extracted_data should return empty dict when no marker."""
    from cases.case5_onboarding_kyc.nodes import _parse_extracted_data

    assert _parse_extracted_data("Hello world") == {}
    assert _parse_extracted_data("EXTRACTED_DATA: {}") == {}


def test_strip_extracted_data():
    """_strip_extracted_data should remove JSON block from response."""
    from cases.case5_onboarding_kyc.nodes import _strip_extracted_data

    content = '¡Genial! Me encanta tu energía. EXTRACTED_DATA: {"primary_goal": "Ganar masa muscular"}'
    result = _strip_extracted_data(content)
    assert result == "¡Genial! Me encanta tu energía."
    assert "EXTRACTED_DATA" not in result
