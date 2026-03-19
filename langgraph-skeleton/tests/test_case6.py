"""Tests for Case 6 — Unified Agent Kairos.

Unit tests for context_loader, prompts, state, and graph invocation.
Uses mock data (no Supabase).
"""

import pytest
from unittest.mock import AsyncMock, patch

from langchain_core.messages import HumanMessage

from cases.case6_unified_agent.state import (
    UserContext,
    UnifiedAgentState,
    DraftRoutine,
    DraftDay,
    DraftExercise,
)
from cases.case6_unified_agent.prompts import format_user_context
from cases.case6_unified_agent.nodes import router, should_continue


# ═══════════════ FIXTURES ═══════════════


def _make_context(**overrides) -> UserContext:
    """Create a UserContext with defaults, overriding specified fields."""
    defaults = {
        "user_id": "test-user-001",
        "full_name": "Camilo Test",
        "phone_number": "573001234567",
        "plan": {
            "plan_id": "plan-001",
            "goal": "Ganar masa muscular",
            "level": "Intermedio",
            "week_schedule": "ul_4",
            "mesocycle_number": 1,
            "status": "active",
        },
        "todays_sessions": [],
        "missed_sessions": [],
        "next_scheduled_session": None,
        "pending_tasks": [],
        "gym_profile": None,
        "is_new_user": False,
        "kyc_complete": True,
        "has_schedule": True,
        "all_w4_completed": False,
    }
    defaults.update(overrides)
    return UserContext(**defaults)


# ═══════════════ STATE TESTS ═══════════════


def test_user_context_new_user():
    ctx = _make_context(user_id=None, is_new_user=True, kyc_complete=False, plan=None)
    assert ctx["is_new_user"] is True
    assert ctx["kyc_complete"] is False
    assert ctx["user_id"] is None


def test_user_context_existing_user():
    ctx = _make_context()
    assert ctx["is_new_user"] is False
    assert ctx["kyc_complete"] is True
    assert ctx["user_id"] == "test-user-001"


def test_user_context_with_missed_sessions():
    ctx = _make_context(
        todays_sessions=[],
        missed_sessions=[
            {"session_name": "Upper Body A", "week": 2, "planned_day": "2026-03-16"},
        ],
    )
    assert len(ctx["missed_sessions"]) == 1
    assert ctx["missed_sessions"][0]["session_name"] == "Upper Body A"


def test_user_context_with_pending_task():
    ctx = _make_context(
        pending_tasks=[
            {"task_id": "task-001", "task_type": "CONFIRMAR_RUTINA", "session_name": "Upper Body A"},
        ],
    )
    assert len(ctx["pending_tasks"]) == 1
    assert ctx["pending_tasks"][0]["task_type"] == "CONFIRMAR_RUTINA"


def test_user_context_w4_completed():
    ctx = _make_context(all_w4_completed=True)
    assert ctx["all_w4_completed"] is True


# ═══════════════ PROMPT FORMATTING TESTS ═══════════════


def test_format_context_existing_user():
    ctx = _make_context(
        todays_sessions=[{"session_name": "Upper Body A", "week": 2, "Completed": False}],
    )
    formatted = format_user_context(ctx)

    assert "Camilo Test" in formatted
    assert "Ganar masa muscular" in formatted
    assert "Upper Body A" in formatted
    assert "NO completada" in formatted


def test_format_context_new_user():
    ctx = _make_context(user_id=None, full_name="", is_new_user=True, kyc_complete=False, plan=None)
    formatted = format_user_context(ctx)

    assert "Usuario nuevo" in formatted
    assert "Ninguno" in formatted


def test_format_context_with_pending_task():
    ctx = _make_context(
        pending_tasks=[
            {"task_id": "t1", "task_type": "CONFIRMAR_RUTINA", "session_name": "Full Body B"},
        ],
    )
    formatted = format_user_context(ctx)

    assert "TAREAS PENDIENTES" in formatted
    assert "Full Body B" in formatted


def test_format_context_w4_completed():
    ctx = _make_context(all_w4_completed=True)
    formatted = format_user_context(ctx)

    assert "COMPLETADO" in formatted
    assert "renovación" in formatted


def test_format_context_missed_sessions():
    ctx = _make_context(
        missed_sessions=[
            {"session_name": "Lower Body A", "planned_day": "2026-03-16"},
        ],
    )
    formatted = format_user_context(ctx)

    assert "Lower Body A" in formatted
    assert "2026-03-16" in formatted


def test_format_context_no_schedule():
    ctx = _make_context(has_schedule=False)
    formatted = format_user_context(ctx)

    assert "NO tiene días agendados" in formatted


# ═══════════════ ROUTER TESTS ═══════════════


def test_router_new_user_no_kyc():
    state = {
        "user_context": _make_context(is_new_user=True, kyc_complete=False),
    }
    assert router(state) == "kyc_subgraph"


def test_router_existing_user():
    state = {
        "user_context": _make_context(is_new_user=False, kyc_complete=True),
    }
    assert router(state) == "kairos_agent"


def test_router_new_user_with_kyc():
    """User just completed KYC — should go to agent, not repeat KYC."""
    state = {
        "user_context": _make_context(is_new_user=True, kyc_complete=True),
    }
    # Even if is_new_user=True, if KYC is complete, go to agent
    # (the condition is is_new AND NOT kyc_complete)
    assert router(state) == "kairos_agent"


# ═══════════════ SHOULD_CONTINUE TESTS ═══════════════


def test_should_continue_no_tool_calls():
    from langchain_core.messages import AIMessage
    state = {"messages": [AIMessage(content="Hola Camilo!")]}
    assert should_continue(state) == "done"


def test_should_continue_with_tool_calls():
    from langchain_core.messages import AIMessage
    msg = AIMessage(content="", tool_calls=[{"name": "get_todays_routine", "args": {}, "id": "1"}])
    state = {"messages": [msg]}
    assert should_continue(state) == "tools"


def test_should_continue_empty_messages():
    state = {"messages": []}
    assert should_continue(state) == "done"


# ═══════════════ GRAPH INVOCATION TEST ═══════════════


@pytest.mark.asyncio
async def test_mock_graph_invocation():
    """Test that the mock graph can be invoked and returns a response."""
    from cases.case6_unified_agent.graph import build_case6_graph

    graph = build_case6_graph()

    result = await graph.ainvoke(
        {
            "messages": [HumanMessage(content="Hola, qué me toca hoy?")],
            "phone_number": "573001234567",
            "display_name": "Test User",
        },
        {"configurable": {"thread_id": "test-thread-001"}},
    )

    assert result.get("response")
    assert len(result["response"]) > 0
    assert result.get("user_context", {}).get("full_name") == "Test User"


# ═══════════════ DRAFT ROUTINE STATE TESTS ═══════════════


def test_draft_routine_structure():
    draft = DraftRoutine(
        week_schedule="ul_4",
        goal="Ganar masa muscular",
        level="Intermedio",
        days=[
            DraftDay(
                day_number=1,
                title="Upper Body A",
                exercises=[
                    DraftExercise(
                        exercise_id="ex_bench_press",
                        spanish_name="Press banca",
                        pattern="push_h",
                        role="compound",
                        sets=4,
                        reps="8-10",
                        rir="1-2",
                        rest_seconds=150,
                        exercise_order=1,
                    ),
                ],
            ),
        ],
    )

    assert draft["week_schedule"] == "ul_4"
    assert len(draft["days"]) == 1
    assert draft["days"][0]["title"] == "Upper Body A"
    assert draft["days"][0]["exercises"][0]["spanish_name"] == "Press banca"


# ═══════════════ EXERCISE ID RESOLUTION TESTS ═══════════════


def test_extract_exercise_identifiers_valid_id():
    from cases.case6_unified_agent.tools import _extract_exercise_identifiers
    cid, cname = _extract_exercise_identifiers({"exercise_id": "ex_barbell_bench_press"})
    assert cid == "ex_barbell_bench_press"
    assert cname is None


def test_extract_exercise_identifiers_name_in_id_field():
    from cases.case6_unified_agent.tools import _extract_exercise_identifiers
    cid, cname = _extract_exercise_identifiers({"exercise_id": "Press de banca con barra"})
    assert cid is None
    assert cname == "Press de banca con barra"


def test_extract_exercise_identifiers_name_field():
    from cases.case6_unified_agent.tools import _extract_exercise_identifiers
    cid, cname = _extract_exercise_identifiers({"name": "Sentadilla con barra"})
    assert cid is None
    assert cname == "Sentadilla con barra"


def test_extract_exercise_identifiers_empty():
    from cases.case6_unified_agent.tools import _extract_exercise_identifiers
    cid, cname = _extract_exercise_identifiers({})
    assert cid is None
    assert cname is None


def test_match_names_exact():
    from cases.case6_unified_agent.tools import _match_names_to_exercises
    candidates = [
        {"exercise_id": "ex_barbell_bench_press", "spanish_name": "Press de banca con barra"},
        {"exercise_id": "ex_barbell_squat", "spanish_name": "Sentadilla con barra"},
    ]
    result = _match_names_to_exercises(["Press de banca con barra"], candidates)
    assert result["Press de banca con barra"] == "ex_barbell_bench_press"


def test_match_names_partial():
    from cases.case6_unified_agent.tools import _match_names_to_exercises
    candidates = [
        {"exercise_id": "ex_barbell_bench_press", "spanish_name": "Press de banca con barra"},
        {"exercise_id": "ex_dumbbell_bench_press", "spanish_name": "Press de banca con mancuernas"},
    ]
    result = _match_names_to_exercises(["Press banca barra"], candidates)
    assert result.get("Press banca barra") == "ex_barbell_bench_press"


def test_match_names_single_word():
    from cases.case6_unified_agent.tools import _match_names_to_exercises
    candidates = [
        {"exercise_id": "ex_pull_up", "spanish_name": "Dominadas"},
    ]
    result = _match_names_to_exercises(["Dominadas"], candidates)
    assert result["Dominadas"] == "ex_pull_up"


def test_match_names_no_match():
    from cases.case6_unified_agent.tools import _match_names_to_exercises
    candidates = [
        {"exercise_id": "ex_barbell_squat", "spanish_name": "Sentadilla con barra"},
    ]
    result = _match_names_to_exercises(["Ejercicio inventado xyz"], candidates)
    assert "Ejercicio inventado xyz" not in result


def test_format_context_with_gym_profile():
    ctx = _make_context(
        plan=None,
        gym_profile={
            "primary_goal": "Ganar masa muscular",
            "training_experience": "Más de 3 años",
            "days_available": 4,
            "training_environment": "GYM",
            "fitness_level": "Avanzado",
        },
    )
    formatted = format_user_context(ctx)
    assert "Perfil KYC" in formatted
    assert "Ganar masa muscular" in formatted
    assert "Avanzado" in formatted
