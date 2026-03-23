"""Tests for Case 6 — Unified Agent Kairos.

Unit tests for context_loader, prompts, state, and graph invocation.
Uses mock data (no Supabase).
"""

import pytest
from datetime import datetime, timedelta, timezone
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


# ═══════════════ SCHEDULE HELPERS TESTS ═══════════════


def test_compute_current_week_week1():
    from cases.case6_unified_agent.tools import _compute_current_week
    from datetime import datetime, timezone

    # Started 3 days ago → week 1
    start = (datetime.now(timezone.utc) - timedelta(days=3)).isoformat()
    assert _compute_current_week(start) == 1


def test_compute_current_week_week3():
    from cases.case6_unified_agent.tools import _compute_current_week

    # Started 16 days ago → week 3
    start = (datetime.now(timezone.utc) - timedelta(days=16)).isoformat()
    assert _compute_current_week(start) == 3


def test_compute_current_week_clamps_to_4():
    from cases.case6_unified_agent.tools import _compute_current_week

    # Started 60 days ago → should clamp to 4
    start = (datetime.now(timezone.utc) - timedelta(days=60)).isoformat()
    assert _compute_current_week(start) == 4


def test_compute_current_week_clamps_to_1():
    from cases.case6_unified_agent.tools import _compute_current_week

    # Started in the future (edge case) → should clamp to 1
    start = (datetime.now(timezone.utc) + timedelta(days=2)).isoformat()
    assert _compute_current_week(start) == 1


def test_normalize_week_day_accents():
    from cases.case6_unified_agent.tools import _normalize_week_day

    assert _normalize_week_day("Miércoles") == "Miercoles"
    assert _normalize_week_day("Sábado") == "Sabado"
    assert _normalize_week_day("lunes") == "Lunes"
    assert _normalize_week_day("VIERNES") == "Viernes"


def test_normalize_planned_day_short():
    from cases.case6_unified_agent.tools import _normalize_planned_day

    result = _normalize_planned_day("17/03")
    # Should append current year
    assert result.startswith("17/03/")
    assert len(result) == 10  # DD/MM/YYYY


def test_normalize_planned_day_full():
    from cases.case6_unified_agent.tools import _normalize_planned_day

    result = _normalize_planned_day("17/03/2026")
    assert result == "17/03/2026"


def test_normalize_planned_day_iso():
    from cases.case6_unified_agent.tools import _normalize_planned_day

    result = _normalize_planned_day("2026-03-17")
    assert result == "2026-03-17"


# ═══════════════ _parse_planned_date TESTS ═══════════════


def test_parse_planned_date_from_utc():
    from cases.case6_unified_agent.context_loader import _parse_planned_date

    row = {"planned_day_utc": "2026-03-22T05:00:00+00:00", "planned_day": "22/03/2026"}
    assert _parse_planned_date(row) == "2026-03-22"


def test_parse_planned_date_dd_mm_yyyy_fallback():
    from cases.case6_unified_agent.context_loader import _parse_planned_date

    row = {"planned_day": "22/03/2026"}
    assert _parse_planned_date(row) == "2026-03-22"


def test_parse_planned_date_iso_fallback():
    from cases.case6_unified_agent.context_loader import _parse_planned_date

    row = {"planned_day": "2026-03-22"}
    assert _parse_planned_date(row) == "2026-03-22"


def test_parse_planned_date_empty():
    from cases.case6_unified_agent.context_loader import _parse_planned_date

    row = {}
    assert _parse_planned_date(row) == ""


# ═══════════════ SCHEDULE_SESSIONS VALIDATION TESTS ═══════════════


@pytest.mark.asyncio
async def test_schedule_sessions_validates_week_day():
    from cases.case6_unified_agent.tools import schedule_sessions
    import json

    sessions = json.dumps([{"week_day": "Monday", "session_name": "Upper A", "planned_day": "17/03"}])
    result = await schedule_sessions.ainvoke({"user_id": "test-uuid", "sessions_json": sessions})
    parsed = json.loads(result)

    assert parsed["success"] is False
    assert any("week_day" in e for e in parsed["errors"])


@pytest.mark.asyncio
async def test_schedule_sessions_validates_empty_fields():
    from cases.case6_unified_agent.tools import schedule_sessions
    import json

    sessions = json.dumps([{"week_day": "Lunes", "session_name": "", "planned_day": ""}])
    result = await schedule_sessions.ainvoke({"user_id": "test-uuid", "sessions_json": sessions})
    parsed = json.loads(result)

    assert parsed["success"] is False
    assert len(parsed["errors"]) == 2


@pytest.mark.asyncio
async def test_schedule_sessions_upsert_params():
    from cases.case6_unified_agent.tools import schedule_sessions
    import json

    plan_data = [{"plan_id": "p1", "start_date": datetime.now(timezone.utc).isoformat()}]

    with patch(
        "cases.case6_unified_agent.tools.supabase_query",
        new_callable=AsyncMock,
        return_value=plan_data,
    ), patch(
        "cases.case6_unified_agent.tools.supabase_bulk_insert",
        new_callable=AsyncMock,
        return_value=[],
    ) as mock_bulk:
        sessions = json.dumps([
            {"week_day": "Lunes", "session_name": "Upper A", "planned_day": "17/03"},
        ])
        result = await schedule_sessions.ainvoke({"user_id": "test-uuid", "sessions_json": sessions})
        parsed = json.loads(result)

        assert parsed["success"] is True
        mock_bulk.assert_called_once()
        call_kwargs = mock_bulk.call_args
        assert call_kwargs.kwargs["upsert"] is True
        assert call_kwargs.kwargs["on_conflict"] == "user_id,week,week_day"


# ═══════════════ SUPABASE_DELETE TESTS ═══════════════


@pytest.mark.asyncio
async def test_supabase_delete_requires_filters():
    from src.shared.supabase_client import supabase_delete

    with pytest.raises(ValueError, match="requires at least one filter"):
        await supabase_delete("user_weekly_schedule", filters={})


# ═══════════════ MESOCYCLE RENEWAL TESTS ═══════════════


@pytest.mark.asyncio
async def test_renew_maintain_success():
    from cases.case6_unified_agent.tools import renew_maintain
    import json

    plan_data = [{"plan_id": "p1", "mesocycle_number": 2, "week_schedule": "ul_4"}]

    with patch("cases.case6_unified_agent.tools.supabase_query", new_callable=AsyncMock, return_value=plan_data), \
         patch("cases.case6_unified_agent.tools.supabase_delete", new_callable=AsyncMock, return_value=[]) as mock_del, \
         patch("cases.case6_unified_agent.tools.supabase_update", new_callable=AsyncMock, return_value=[]) as mock_upd:

        result = await renew_maintain.ainvoke({"user_id": "test-uuid"})
        parsed = json.loads(result)

        assert parsed["success"] is True
        assert parsed["new_mesocycle_number"] == 3
        assert parsed["action"] == "MANTENER_RUTINA"
        mock_del.assert_called_once()
        mock_upd.assert_called_once()
        assert mock_upd.call_args.kwargs["data"]["mesocycle_number"] == 3


@pytest.mark.asyncio
async def test_renew_maintain_no_plan():
    from cases.case6_unified_agent.tools import renew_maintain
    import json

    with patch("cases.case6_unified_agent.tools.supabase_query", new_callable=AsyncMock, return_value=[]):
        result = await renew_maintain.ainvoke({"user_id": "test-uuid"})
        parsed = json.loads(result)
        assert parsed["success"] is False
        assert "plan activo" in parsed["error"]


@pytest.mark.asyncio
async def test_renew_change_days_validates_range():
    from cases.case6_unified_agent.tools import renew_change_days
    import json

    result = await renew_change_days.ainvoke({"user_id": "test-uuid", "new_days_per_week": 7})
    parsed = json.loads(result)
    assert parsed["success"] is False
    assert "2 y 6" in parsed["error"]


@pytest.mark.asyncio
async def test_renew_change_days_success():
    from cases.case6_unified_agent.tools import renew_change_days
    import json

    plan_data = [{"plan_id": "p1", "mesocycle_number": 1, "goal": "Ganar masa muscular", "level": "Intermedio"}]

    with patch("cases.case6_unified_agent.tools.supabase_query", new_callable=AsyncMock, return_value=plan_data), \
         patch("cases.case6_unified_agent.tools.supabase_delete", new_callable=AsyncMock, return_value=[]) as mock_del, \
         patch("cases.case6_unified_agent.tools.supabase_update", new_callable=AsyncMock, return_value=[]):

        result = await renew_change_days.ainvoke({"user_id": "test-uuid", "new_days_per_week": 3})
        parsed = json.loads(result)

        assert parsed["success"] is True
        assert parsed["new_week_schedule"] == "fb_3"
        assert parsed["new_mesocycle_number"] == 2
        assert mock_del.call_count == 2  # workouts + schedule


@pytest.mark.asyncio
async def test_renew_rotate_success():
    from cases.case6_unified_agent.tools import renew_rotate_exercises
    import json

    plan_data = [{"plan_id": "p1", "mesocycle_number": 1, "level": "Intermedio"}]
    w1_workouts = [{"id": "w1", "exercise_id": "ex_bench_press", "day_name": "Upper A"}]
    exercises_data = [{"exercise_id": "ex_bench_press", "spanish_name": "Press banca", "pattern": "push_h", "role": "compound", "level": "Intermedio"}]
    alternatives = [{"exercise_id": "ex_dumbbell_press", "spanish_name": "Press mancuernas", "pattern": "push_h", "role": "compound", "level": "Intermedio"}]

    call_count = {"n": 0}

    async def mock_query(table, select="*", filters=None, limit=None):
        call_count["n"] += 1
        if table == "users_plans":
            return plan_data
        if table == "workouts":
            return w1_workouts
        if table == "exercises":
            if filters and "in." in str(filters.get("exercise_id", "")):
                return exercises_data
            return alternatives
        return []

    with patch("cases.case6_unified_agent.tools.supabase_query", new_callable=AsyncMock, side_effect=mock_query), \
         patch("cases.case6_unified_agent.tools.supabase_update", new_callable=AsyncMock, return_value=[]) as mock_upd, \
         patch("cases.case6_unified_agent.tools.supabase_delete", new_callable=AsyncMock, return_value=[]):

        result = await renew_rotate_exercises.ainvoke({"user_id": "test-uuid"})
        parsed = json.loads(result)

        assert parsed["success"] is True
        assert parsed["new_mesocycle_number"] == 2
        assert parsed["action"] == "ROTAR_EJERCICIOS"
        assert len(parsed["rotated"]) == 1
        assert parsed["rotated"][0]["old"] == "Press banca"


@pytest.mark.asyncio
async def test_renew_rotate_no_alternatives():
    from cases.case6_unified_agent.tools import renew_rotate_exercises
    import json

    plan_data = [{"plan_id": "p1", "mesocycle_number": 1, "level": "Intermedio"}]
    w1_workouts = [{"id": "w1", "exercise_id": "ex_unique", "day_name": "Full Body A"}]
    exercises_data = [{"exercise_id": "ex_unique", "spanish_name": "Ejercicio unico", "pattern": "special", "role": "compound", "level": "Intermedio"}]

    async def mock_query(table, select="*", filters=None, limit=None):
        if table == "users_plans":
            return plan_data
        if table == "workouts":
            return w1_workouts
        if table == "exercises":
            if filters and "in." in str(filters.get("exercise_id", "")):
                return exercises_data
            return []  # No alternatives
        return []

    with patch("cases.case6_unified_agent.tools.supabase_query", new_callable=AsyncMock, side_effect=mock_query), \
         patch("cases.case6_unified_agent.tools.supabase_update", new_callable=AsyncMock, return_value=[]), \
         patch("cases.case6_unified_agent.tools.supabase_delete", new_callable=AsyncMock, return_value=[]):

        result = await renew_rotate_exercises.ainvoke({"user_id": "test-uuid"})
        parsed = json.loads(result)

        assert parsed["success"] is True
        assert len(parsed["rotated"]) == 0
        assert "Ejercicio unico" in parsed["kept"]


@pytest.mark.asyncio
async def test_save_workout_plan_is_renewal_skips_plan():
    from cases.case6_unified_agent.tools import save_workout_plan
    import json

    draft = {
        "is_renewal": True,
        "week_schedule": "fb_3",
        "goal": "Ganar masa muscular",
        "level": "Intermedio",
        "days": [{
            "day_number": 1,
            "title": "Full Body A",
            "exercises": [{
                "exercise_id": "ex_barbell_squat",
                "sets": 3, "reps": "8-10", "rir": "1-2",
                "rest_seconds": 150, "exercise_order": 1, "tempo": "2-0-1",
            }],
        }],
    }

    with patch("cases.case6_unified_agent.tools.supabase_insert", new_callable=AsyncMock, return_value=[]) as mock_ins, \
         patch("cases.case6_unified_agent.tools.supabase_bulk_insert", new_callable=AsyncMock, return_value=[]), \
         patch("cases.case6_unified_agent.tools.supabase_query", new_callable=AsyncMock, return_value=[]):

        result = await save_workout_plan.ainvoke({
            "user_id": "test-uuid",
            "draft_json": json.dumps(draft),
        })
        parsed = json.loads(result)

        assert parsed["success"] is True
        # supabase_insert should NOT be called (plan creation skipped)
        mock_ins.assert_not_called()


# ═══════════════ HEALTH FILTERING TESTS ═══════════════


def test_health_filter_none_passthrough():
    from cases.case6_unified_agent.tools import _apply_health_filter

    exercises = [
        {"exercise_id": "ex_lunge", "spanish_name": "Zancada", "pattern": "lunge", "equipment": "bodyweight", "role": "compound", "main_muscle": "Quads", "level": "Intermedio"},
    ]
    assert len(_apply_health_filter(exercises, None)) == 1
    assert len(_apply_health_filter(exercises, "A")) == 1


def test_health_filter_b_excludes_lunges():
    from cases.case6_unified_agent.tools import _apply_health_filter

    exercises = [
        {"exercise_id": "ex_lunge", "spanish_name": "Zancada", "pattern": "lunge", "equipment": "bodyweight", "role": "compound", "main_muscle": "Quads", "level": "Intermedio"},
        {"exercise_id": "ex_bench", "spanish_name": "Press banca", "pattern": "push_h", "equipment": "barbell", "role": "compound", "main_muscle": "Chest", "level": "Intermedio"},
    ]
    result = _apply_health_filter(exercises, "B")
    assert len(result) == 1
    assert result[0]["exercise_id"] == "ex_bench"


def test_health_filter_b_excludes_barbell_squats():
    from cases.case6_unified_agent.tools import _apply_health_filter

    exercises = [
        {"exercise_id": "ex_bb_squat", "spanish_name": "Sentadilla con barra", "pattern": "squat", "equipment": "barbell", "role": "compound", "main_muscle": "Quads", "level": "Intermedio"},
        {"exercise_id": "ex_goblet", "spanish_name": "Sentadilla goblet", "pattern": "squat", "equipment": "dumbbell", "role": "compound", "main_muscle": "Quads", "level": "Intermedio"},
    ]
    result = _apply_health_filter(exercises, "B")
    assert len(result) == 1
    assert result[0]["exercise_id"] == "ex_goblet"


def test_health_filter_b_excludes_jumps_by_name():
    from cases.case6_unified_agent.tools import _apply_health_filter

    exercises = [
        {"exercise_id": "ex_jump", "spanish_name": "Salto al cajón", "pattern": "accessory", "equipment": "bodyweight", "role": "compound", "main_muscle": "Quads", "level": "Intermedio"},
    ]
    assert len(_apply_health_filter(exercises, "B")) == 0


def test_health_filter_c_excludes_push_v():
    from cases.case6_unified_agent.tools import _apply_health_filter

    exercises = [
        {"exercise_id": "ex_ohp", "spanish_name": "Press militar", "pattern": "push_v", "equipment": "barbell", "role": "compound", "main_muscle": "Shoulders", "level": "Intermedio"},
        {"exercise_id": "ex_bench", "spanish_name": "Press banca", "pattern": "push_h", "equipment": "barbell", "role": "compound", "main_muscle": "Chest", "level": "Intermedio"},
    ]
    result = _apply_health_filter(exercises, "C")
    assert len(result) == 1
    assert result[0]["exercise_id"] == "ex_bench"


def test_health_filter_c_excludes_upright_row():
    from cases.case6_unified_agent.tools import _apply_health_filter

    exercises = [
        {"exercise_id": "ex_barbell_upright_row", "spanish_name": "Remo vertical", "pattern": "pull_v", "equipment": "barbell", "role": "compound", "main_muscle": "Shoulders", "level": "Intermedio"},
    ]
    assert len(_apply_health_filter(exercises, "C")) == 0


def test_health_filter_d_excludes_barbell_hinge():
    from cases.case6_unified_agent.tools import _apply_health_filter

    exercises = [
        {"exercise_id": "ex_bb_dl", "spanish_name": "Peso muerto con barra", "pattern": "hinge", "equipment": "barbell", "role": "compound", "main_muscle": "Hamstrings", "level": "Intermedio"},
        {"exercise_id": "ex_db_rdl", "spanish_name": "RDL con mancuernas", "pattern": "hinge", "equipment": "dumbbell", "role": "compound", "main_muscle": "Hamstrings", "level": "Intermedio"},
    ]
    result = _apply_health_filter(exercises, "D")
    assert len(result) == 1
    assert result[0]["exercise_id"] == "ex_db_rdl"


def test_health_filter_d_excludes_compound_lower_back():
    from cases.case6_unified_agent.tools import _apply_health_filter

    exercises = [
        {"exercise_id": "ex_gm", "spanish_name": "Good morning", "pattern": "hinge", "equipment": "barbell", "role": "compound", "main_muscle": "Lower back", "level": "Intermedio"},
    ]
    assert len(_apply_health_filter(exercises, "D")) == 0


def test_health_filter_e_only_machine_cable_bw_core():
    from cases.case6_unified_agent.tools import _apply_health_filter

    exercises = [
        {"exercise_id": "ex_leg_press", "spanish_name": "Prensa", "pattern": "squat", "equipment": "machine", "role": "compound", "main_muscle": "Quads", "level": "Intermedio"},
        {"exercise_id": "ex_cable_row", "spanish_name": "Remo cable", "pattern": "pull_h", "equipment": "cable", "role": "compound", "main_muscle": "Back", "level": "Intermedio"},
        {"exercise_id": "ex_plank", "spanish_name": "Plancha", "pattern": "core", "equipment": "bodyweight", "role": "core", "main_muscle": "Abs", "level": "Principiante"},
        {"exercise_id": "ex_bb_squat", "spanish_name": "Sentadilla barra", "pattern": "squat", "equipment": "barbell", "role": "compound", "main_muscle": "Quads", "level": "Intermedio"},
        {"exercise_id": "ex_db_press", "spanish_name": "Press mancuernas", "pattern": "push_h", "equipment": "dumbbell", "role": "compound", "main_muscle": "Chest", "level": "Intermedio"},
    ]
    result = _apply_health_filter(exercises, "E")
    ids = [e["exercise_id"] for e in result]
    assert "ex_leg_press" in ids
    assert "ex_cable_row" in ids
    assert "ex_plank" in ids
    assert "ex_bb_squat" not in ids
    assert "ex_db_press" not in ids


def test_health_filter_e_excludes_advanced():
    from cases.case6_unified_agent.tools import _apply_health_filter

    exercises = [
        {"exercise_id": "ex_adv", "spanish_name": "Algo avanzado", "pattern": "squat", "equipment": "machine", "role": "compound", "main_muscle": "Quads", "level": "Avanzado"},
    ]
    assert len(_apply_health_filter(exercises, "E")) == 0


def test_health_filter_e_excludes_cardio():
    from cases.case6_unified_agent.tools import _apply_health_filter

    exercises = [
        {"exercise_id": "ex_burpee", "spanish_name": "Burpee", "pattern": "accessory", "equipment": "bodyweight", "role": "cardio", "main_muscle": "Quads", "level": "Intermedio"},
    ]
    assert len(_apply_health_filter(exercises, "E")) == 0


# ═══════════════ DEDUPLICATION TESTS ═══════════════


@pytest.mark.asyncio
async def test_save_workout_plan_dedup_same_day():
    """Duplicate exercise_id on same day should be deduplicated."""
    from cases.case6_unified_agent.tools import save_workout_plan
    import json

    draft = {
        "week_schedule": "fb_3",
        "goal": "Ganar masa muscular",
        "level": "Intermedio",
        "days": [{
            "day_number": 1,
            "title": "Full Body A",
            "exercises": [
                {"exercise_id": "ex_barbell_squat", "sets": 3, "reps": "8-10", "rir": "1-2", "rest_seconds": 150, "exercise_order": 1, "tempo": "2-0-1"},
                {"exercise_id": "ex_barbell_squat", "sets": 3, "reps": "12-15", "rir": "1-2", "rest_seconds": 90, "exercise_order": 2, "tempo": "2-0-1"},
                {"exercise_id": "ex_bench_press", "sets": 3, "reps": "8-10", "rir": "1-2", "rest_seconds": 150, "exercise_order": 3, "tempo": "2-0-1"},
            ],
        }],
    }

    # Mock supabase_query to validate exercise IDs
    valid_ids = [{"exercise_id": "ex_barbell_squat"}, {"exercise_id": "ex_bench_press"}]

    async def capture_bulk_insert(table, rows, **kwargs):
        return rows

    with patch("cases.case6_unified_agent.tools.supabase_insert", new_callable=AsyncMock, return_value=[]), \
         patch("cases.case6_unified_agent.tools.supabase_bulk_insert", new_callable=AsyncMock, side_effect=capture_bulk_insert) as mock_bulk, \
         patch("cases.case6_unified_agent.tools.supabase_query", new_callable=AsyncMock, return_value=valid_ids):

        result = await save_workout_plan.ainvoke({"user_id": "test-uuid", "draft_json": json.dumps(draft)})
        parsed = json.loads(result)

        assert parsed["success"] is True
        # 2 unique exercises × 4 weeks = 8 (not 3 × 4 = 12)
        assert parsed["workouts_created"] == 8


@pytest.mark.asyncio
async def test_save_workout_plan_same_exercise_different_days():
    """Same exercise on different days should NOT be deduplicated."""
    from cases.case6_unified_agent.tools import save_workout_plan
    import json

    draft = {
        "week_schedule": "fb_3",
        "goal": "Ganar masa muscular",
        "level": "Intermedio",
        "days": [
            {"day_number": 1, "title": "Full Body A", "exercises": [
                {"exercise_id": "ex_barbell_squat", "sets": 3, "reps": "8-10", "rir": "1-2", "rest_seconds": 150, "exercise_order": 1, "tempo": "2-0-1"},
            ]},
            {"day_number": 2, "title": "Full Body B", "exercises": [
                {"exercise_id": "ex_barbell_squat", "sets": 3, "reps": "8-10", "rir": "1-2", "rest_seconds": 150, "exercise_order": 1, "tempo": "2-0-1"},
            ]},
        ],
    }

    valid_ids = [{"exercise_id": "ex_barbell_squat"}]

    with patch("cases.case6_unified_agent.tools.supabase_insert", new_callable=AsyncMock, return_value=[]), \
         patch("cases.case6_unified_agent.tools.supabase_bulk_insert", new_callable=AsyncMock, return_value=[]) as mock_bulk, \
         patch("cases.case6_unified_agent.tools.supabase_query", new_callable=AsyncMock, return_value=valid_ids):

        result = await save_workout_plan.ainvoke({"user_id": "test-uuid", "draft_json": json.dumps(draft)})
        parsed = json.loads(result)

        assert parsed["success"] is True
        # 1 exercise × 2 days × 4 weeks = 8 (both kept — different days)
        assert parsed["workouts_created"] == 8


# ═══════════════ EMAIL TESTS ═══════════════


@pytest.mark.asyncio
async def test_send_routine_email_no_email():
    from cases.case6_unified_agent.tools import send_routine_email
    import json

    user_data = [{"full_name": "Test User", "email": None}]

    with patch("cases.case6_unified_agent.tools.supabase_query", new_callable=AsyncMock, return_value=user_data):
        result = await send_routine_email.ainvoke({"user_id": "test-uuid"})
        parsed = json.loads(result)
        assert parsed["success"] is False
        assert "email" in parsed["error"].lower()


@pytest.mark.asyncio
async def test_send_routine_email_no_workouts():
    from cases.case6_unified_agent.tools import send_routine_email
    import json

    call_count = {"n": 0}

    async def mock_query(table, select="*", filters=None, limit=None):
        call_count["n"] += 1
        if table == "users":
            return [{"full_name": "Test User", "email": "test@test.com"}]
        if table == "users_plans":
            return [{"plan_id": "p1", "goal": "Ganar masa muscular", "level": "Intermedio", "week_schedule": "fb_3"}]
        if table == "workouts":
            return []  # No workouts
        return []

    with patch("cases.case6_unified_agent.tools.supabase_query", new_callable=AsyncMock, side_effect=mock_query):
        result = await send_routine_email.ainvoke({"user_id": "test-uuid"})
        parsed = json.loads(result)
        assert parsed["success"] is False
        assert "ejercicios" in parsed["error"].lower()


@pytest.mark.asyncio
async def test_send_routine_email_success():
    from cases.case6_unified_agent.tools import send_routine_email
    import json

    async def mock_query(table, select="*", filters=None, limit=None):
        if table == "users":
            return [{"full_name": "Test User", "email": "test@test.com"}]
        if table == "users_plans":
            return [{"plan_id": "p1", "goal": "Ganar masa muscular", "level": "Intermedio", "week_schedule": "fb_3"}]
        if table == "workouts":
            return [{"day_name": "Full Body A", "exercise_id": "ex_squat", "sets": "3", "reps": "8-10", "rir": "1-2", "rest-seconds": 150, "tempo": "2-0-1", "exercise_order": 1}]
        if table == "exercises":
            return [{"exercise_id": "ex_squat", "spanish_name": "Sentadilla", "main_muscle": "Quads", "link": "https://example.com"}]
        return []

    with patch("cases.case6_unified_agent.tools.supabase_query", new_callable=AsyncMock, side_effect=mock_query), \
         patch("cases.case6_unified_agent.tools._send_email", new_callable=AsyncMock) as mock_send:

        result = await send_routine_email.ainvoke({"user_id": "test-uuid"})
        parsed = json.loads(result)

        assert parsed["success"] is True
        assert parsed["email"] == "test@test.com"
        assert parsed["days_sent"] == 1
        mock_send.assert_called_once()
        call_args = mock_send.call_args
        assert call_args.kwargs["to"] == "test@test.com"
        assert "Sentadilla" in call_args.kwargs["html_body"]


# ═══════════════ TOOL-LEAK GUARDRAIL TESTS ═══════════════


class TestHasToolLeak:
    """Tests for _has_tool_leak() — detects tool calls written as text."""

    def setup_method(self):
        from cases.case6_unified_agent.graph_live import _has_tool_leak
        self._has_tool_leak = _has_tool_leak

    def test_detects_print_default_api(self):
        text = "Voy a buscar tu rutina. print(default_api.get_day_requirements(week_schedule='ul_4'))"
        assert self._has_tool_leak(text) is True

    def test_detects_default_api_without_print(self):
        text = "default_api.get_exercises_for_draft(pattern='push', level='Intermedio')"
        assert self._has_tool_leak(text) is True

    def test_detects_bare_tool_call(self):
        text = "get_exercises_for_draft(pattern='push', level='Intermedio')"
        assert self._has_tool_leak(text) is True

    def test_detects_print_with_tool_name(self):
        text = "print(get_todays_routine(user_id='abc'))"
        assert self._has_tool_leak(text) is True

    def test_false_for_normal_text(self):
        text = "¡Hola Santiago! Aquí tienes tu rutina de hoy. 💪"
        assert self._has_tool_leak(text) is False

    def test_false_for_empty(self):
        assert self._has_tool_leak("") is False
        assert self._has_tool_leak(None) is False

    def test_false_for_natural_mention(self):
        text = "Voy a buscar tus ejercicios y crear tu rutina personalizada."
        assert self._has_tool_leak(text) is False


class TestSanitizeResponse:
    """Tests for _sanitize_response() — strips leaked tool patterns."""

    def setup_method(self):
        from cases.case6_unified_agent.graph_live import _sanitize_response
        self._sanitize_response = _sanitize_response

    def test_removes_print_block(self):
        text = "Dame un momento. print(default_api.get_day_requirements('ul_4'))"
        result = self._sanitize_response(text)
        assert "print" not in result
        assert "default_api" not in result
        assert "Dame un momento" in result

    def test_removes_default_api(self):
        text = "Buscando ejercicios. default_api.get_exercises_for_draft(pattern='push')"
        result = self._sanitize_response(text)
        assert "default_api" not in result
        assert "Buscando ejercicios" in result

    def test_removes_bare_tool_call(self):
        text = "Listo! get_todays_routine(user_id='abc') Aquí va."
        result = self._sanitize_response(text)
        assert "get_todays_routine" not in result
        assert "Aquí va" in result

    def test_fallback_when_all_stripped(self):
        text = "print(default_api.get_day_requirements('fb_3'))"
        result = self._sanitize_response(text)
        assert result == "Dame un momento, estoy procesando tu solicitud."

    def test_preserves_normal_text(self):
        text = "¡Excelente, Santiago! Tu rutina de 4 días está lista. 💪"
        result = self._sanitize_response(text)
        assert result == text
