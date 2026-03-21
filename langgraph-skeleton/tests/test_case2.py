"""Tests for Case 2: Conditional Routing + Gemini Selection.

Tests:
- Health A routes to no_restrictions (full pool)
- Health C routes to restrict_upper_body (no overhead pressing)
- Health B routes to restrict_lower_body (no heavy barbell squats/deadlifts)
- Gemini selects exercises from filtered pool
"""

import pytest
from cases.case2_conditional.graph import build_case2_graph


@pytest.fixture
def graph():
    return build_case2_graph()


@pytest.mark.asyncio
async def test_health_a_no_restrictions(graph):
    """Camilo (Health A) should get full exercise pool and no_restrictions route."""
    result = await graph.ainvoke({"user_id": "camilo-001"})

    assert result["route_taken"] == "no_restrictions"
    # Full pool should have all 16 exercises
    assert len(result["available_exercises"]) == 16
    # Gemini should select some exercises
    assert len(result["selected_exercises"]) >= 1


@pytest.mark.asyncio
async def test_health_c_filters_overhead(graph):
    """Ana (Health C) should not have overhead press in available exercises."""
    result = await graph.ainvoke({"user_id": "ana-002"})

    assert result["route_taken"] == "restrict_upper_body"
    # Should have fewer exercises than full pool
    assert len(result["available_exercises"]) < 16
    # No overhead pressing exercise (press militar) in the pool
    for ex in result["available_exercises"]:
        assert not (
            ex["pattern"] == "push" and ex["main_muscle"] == "Shoulders"
        ), f"Overhead press should be filtered: {ex['spanish_name']}"


@pytest.mark.asyncio
async def test_health_b_filters_lower_body(graph):
    """Carlos (Health B) should not have barbell squats/deadlifts."""
    result = await graph.ainvoke({"user_id": "carlos-003"})

    assert result["route_taken"] == "restrict_lower_body"
    # No barbell squat or deadlift
    for ex in result["available_exercises"]:
        assert not (
            ex["pattern"] in ("squat", "hip_hinge") and ex["equipment"] == "barbell"
        ), f"Heavy barbell lower body should be filtered: {ex['spanish_name']}"


@pytest.mark.asyncio
async def test_gemini_selects_from_filtered_pool(graph):
    """Selected exercises should all come from the available pool."""
    result = await graph.ainvoke({"user_id": "camilo-001"})

    available_ids = {e["exercise_id"] for e in result["available_exercises"]}
    for ex in result["selected_exercises"]:
        assert ex["exercise_id"] in available_ids, (
            f"Selected exercise {ex['exercise_id']} not in available pool"
        )
