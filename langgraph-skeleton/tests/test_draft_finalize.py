"""Tests for POST /api/v1/drafts/{code}/finalize and save_workout_plan.

The /finalize endpoint is the deterministic, LLM-free counterpart to /chat:
when the user clicks "Aprobar rutina" the Go backend calls this so the exact
draft_data the user approved (including any swaps) gets persisted to workouts.

Two layers under test:
  1. HTTP handler in server.py — status codes + idempotency short-circuit +
     response shape. Mocks the tool itself (`save_workout_plan.ainvoke`).
  2. save_workout_plan in tools.py — the regression layer for BUG-1: the
     exercise_id the user chose must land in workouts. Plus the min-exercises
     validation and is_renewal flag. Mocks Supabase + the loading-params /
     role / equipment helpers so tests are hermetic.
"""

from __future__ import annotations

import json
from typing import Any
from unittest.mock import AsyncMock

import pytest
from fastapi.testclient import TestClient


# ─────────────────── Shared fixtures ───────────────────


def _valid_exercise(**overrides) -> dict:
    base = {
        "exercise_id": "ex_032",
        "spanish_name": "Sentadilla búlgara",
        "pattern": "squat",
        "role": "compound",
        "sets": 3,
        "reps": "8-10",
        "rir": "2",
        "rest_seconds": 120,
        "exercise_order": 1,
        "alternatives": [],
    }
    base.update(overrides)
    return base


def _valid_draft(num_days: int = 4, exercises_per_day: int = 5) -> dict:
    days = []
    for d in range(num_days):
        exercises = [
            _valid_exercise(
                exercise_id=f"ex_{d:02d}{e:02d}",
                spanish_name=f"Exercise D{d}E{e}",
                exercise_order=e + 1,
            )
            for e in range(exercises_per_day)
        ]
        days.append({
            "day_number": d + 1,
            "title": ["Upper A", "Lower A", "Upper B", "Lower B"][d % 4],
            "exercises": exercises,
        })
    return {
        "week_schedule": "ul_4",
        "goal": "Ganar masa muscular",
        "level": "Avanzado",
        "days": days,
    }


# ═══════════════════════════════════════════════════════════════════════════
# Endpoint tests — mock the tool, focus on HTTP behavior
# ═══════════════════════════════════════════════════════════════════════════


@pytest.fixture
def endpoint_client(monkeypatch):
    """TestClient with server.py imports patched.

    Mocks:
      - `_supabase_query` so we control which rows the route reads
      - `_save_workout_plan_tool.ainvoke` so we control the tool response

    Exposes:
      `client.state` — { draft_rows, plan_rows, tool_calls, tool_result, tool_exc }
    """
    import server as server_mod

    state: dict[str, Any] = {
        "draft_rows": [],            # what _supabase_query returns for draft_routines
        "plan_rows": [],             # what _supabase_query returns for users_plans
        "tool_calls": [],            # args captured each call to .ainvoke
        "tool_result": None,         # JSON string returned by .ainvoke (success path)
        "tool_exc": None,            # if set, .ainvoke raises this
    }

    async def fake_query(table, select="*", filters=None, limit=None):
        if table == "draft_routines":
            return state["draft_rows"]
        if table == "users_plans":
            return state["plan_rows"]
        return []

    async def fake_ainvoke(payload):
        state["tool_calls"].append(payload)
        if state["tool_exc"]:
            raise state["tool_exc"]
        return state["tool_result"]

    # The tool is a Pydantic StructuredTool (immutable attrs) — replace the
    # module-level binding wholesale rather than patching .ainvoke in place.
    class _FakeTool:
        ainvoke = staticmethod(fake_ainvoke)

    monkeypatch.setattr(server_mod, "_supabase_query", fake_query)
    monkeypatch.setattr(server_mod, "_save_workout_plan_tool", _FakeTool)

    # Short-circuit lifespan (compiles LangGraph + opens Postgres pool in real life)
    from contextlib import asynccontextmanager

    @asynccontextmanager
    async def noop_lifespan(app):
        yield

    server_mod.app.router.lifespan_context = noop_lifespan

    with TestClient(server_mod.app) as c:
        c.state = state
        yield c


def test_endpoint_404_when_code_unknown(endpoint_client):
    endpoint_client.state["draft_rows"] = []
    r = endpoint_client.post("/api/v1/drafts/nope/finalize")
    assert r.status_code == 404
    assert r.json()["detail"]["error"] == "draft not found"


def test_endpoint_idempotent_when_active_plan_exists(endpoint_client):
    """If user already has an active plan, /finalize must short-circuit:
    return the existing plan_id with workouts_created=0 and NOT call the tool."""
    endpoint_client.state["draft_rows"] = [{
        "user_id": "user-uuid",
        "draft_data": _valid_draft(),
        "status": "approved",
    }]
    endpoint_client.state["plan_rows"] = [{"plan_id": "existing-plan"}]

    r = endpoint_client.post("/api/v1/drafts/code1/finalize")

    assert r.status_code == 200
    body = r.json()
    assert body["success"] is True
    assert body["plan_id"] == "existing-plan"
    assert body["workouts_created"] == 0
    assert body["user_id"] == "user-uuid"
    # Tool was NOT invoked
    assert endpoint_client.state["tool_calls"] == []


def test_endpoint_200_success_path(endpoint_client):
    endpoint_client.state["draft_rows"] = [{
        "user_id": "user-uuid",
        "draft_data": _valid_draft(),
        "status": "pending",
    }]
    endpoint_client.state["plan_rows"] = []      # no active plan
    endpoint_client.state["tool_result"] = json.dumps({
        "success": True,
        "plan_id": "new-plan-uuid",
        "workouts_created": 72,
    })

    r = endpoint_client.post("/api/v1/drafts/code1/finalize")

    assert r.status_code == 200
    body = r.json()
    assert body["success"] is True
    assert body["plan_id"] == "new-plan-uuid"
    assert body["workouts_created"] == 72
    assert body["user_id"] == "user-uuid"
    # Tool received the draft_data we loaded
    assert len(endpoint_client.state["tool_calls"]) == 1
    sent = endpoint_client.state["tool_calls"][0]
    assert sent["user_id"] == "user-uuid"
    # draft_json is a string; parse and check we forwarded the same content
    forwarded = json.loads(sent["draft_json"])
    assert forwarded["goal"] == "Ganar masa muscular"


def test_endpoint_422_when_tool_returns_validation_error(endpoint_client):
    """When save_workout_plan returns success=False (e.g. <3 exercises/day),
    the endpoint must surface it as 422 so Go doesn't blind-retry."""
    endpoint_client.state["draft_rows"] = [{
        "user_id": "user-uuid",
        "draft_data": _valid_draft(),
        "status": "pending",
    }]
    endpoint_client.state["tool_result"] = json.dumps({
        "success": False,
        "error": "Días con ejercicios insuficientes: ['Upper A']",
    })

    r = endpoint_client.post("/api/v1/drafts/code1/finalize")
    assert r.status_code == 422
    detail = r.json()["detail"]
    assert detail["success"] is False
    assert "insuficientes" in detail["error"]


def test_endpoint_500_when_tool_throws(endpoint_client):
    endpoint_client.state["draft_rows"] = [{
        "user_id": "user-uuid",
        "draft_data": _valid_draft(),
        "status": "pending",
    }]
    endpoint_client.state["tool_exc"] = RuntimeError("supabase boom")

    r = endpoint_client.post("/api/v1/drafts/code1/finalize")
    assert r.status_code == 500
    detail = r.json()["detail"]
    assert detail["success"] is False
    assert "supabase boom" in detail["error"]


def test_endpoint_500_when_tool_returns_unparseable_json(endpoint_client):
    """Defensive: a malformed tool response should not crash the server."""
    endpoint_client.state["draft_rows"] = [{
        "user_id": "user-uuid",
        "draft_data": _valid_draft(),
        "status": "pending",
    }]
    endpoint_client.state["tool_result"] = "not json at all"

    r = endpoint_client.post("/api/v1/drafts/code1/finalize")
    assert r.status_code == 500
    assert "invalid tool response" in r.json()["detail"]["error"]


# ═══════════════════════════════════════════════════════════════════════════
# save_workout_plan tests — mock supabase + helpers, exercise the real tool
# ═══════════════════════════════════════════════════════════════════════════


@pytest.fixture
def tool_supabase(monkeypatch):
    """Mocks Supabase helpers and the loading/role/equipment helpers used by
    save_workout_plan, so we can assert what gets inserted into workouts.
    """
    import cases.case6_unified_agent.tools as tools_mod

    state: dict[str, Any] = {
        "insert_calls": [],
        "bulk_insert_calls": [],
        "query_results": {},
        "allowed_equipment": None,    # None → equipment validation is skipped (anon user)
    }

    async def fake_query(table, select="*", filters=None, limit=None):
        return state["query_results"].get(table, [])

    async def fake_insert(table, data=None, upsert=False, on_conflict=None):
        state["insert_calls"].append({"table": table, "data": data})
        return [data]

    async def fake_bulk_insert(table, rows):
        state["bulk_insert_calls"].append({"table": table, "rows": rows})
        return rows

    async def fake_delete(table, filters=None):
        return []

    async def fake_resolve(draft):
        """Resolve every exercise to its provided exercise_id (no DB lookup)."""
        resolved = {}
        for d_idx, day in enumerate(draft.get("days", [])):
            for e_idx, ex in enumerate(day.get("exercises", [])):
                resolved[(d_idx, e_idx)] = ex["exercise_id"]
        return resolved, []

    async def fake_user_allowed_equipment(user_id):
        return state["allowed_equipment"]

    async def fake_fetch_loading_params(goal, level):
        # Return None for everything → tool falls back to DEFAULT_LOADING_PARAMS.
        return {}

    async def fake_fetch_exercise_roles(exercise_ids):
        return {ex_id: "compound" for ex_id in exercise_ids}

    monkeypatch.setattr(tools_mod, "supabase_query", fake_query)
    monkeypatch.setattr(tools_mod, "supabase_insert", fake_insert)
    monkeypatch.setattr(tools_mod, "supabase_bulk_insert", fake_bulk_insert)
    monkeypatch.setattr(tools_mod, "supabase_delete", fake_delete)
    monkeypatch.setattr(tools_mod, "_resolve_exercise_ids", fake_resolve)
    monkeypatch.setattr(tools_mod, "_user_allowed_equipment", fake_user_allowed_equipment)
    monkeypatch.setattr(tools_mod, "_fetch_loading_params", fake_fetch_loading_params)
    monkeypatch.setattr(tools_mod, "_fetch_exercise_roles", fake_fetch_exercise_roles)
    return state


@pytest.mark.asyncio
async def test_tool_writes_workouts_with_exact_swapped_exercise_id(tool_supabase):
    """Regression for BUG-1: the swapped exercise_id from the draft must
    survive into the workouts bulk insert. This is the contract the user
    cares about when they click "Elegir esta" in the preview UI.
    """
    from cases.case6_unified_agent.tools import save_workout_plan

    draft = _valid_draft(num_days=2, exercises_per_day=3)
    # User swapped Upper A first exercise to ex_swap_target_xyz
    draft["days"][0]["exercises"][0]["exercise_id"] = "ex_swap_target_xyz"
    draft["days"][0]["exercises"][0]["spanish_name"] = "Swapped Exercise"

    raw = await save_workout_plan.ainvoke({
        "user_id": "user-uuid",
        "draft_json": json.dumps(draft),
    })
    result = json.loads(raw)
    assert result["success"] is True
    assert result["workouts_created"] > 0

    rows = tool_supabase["bulk_insert_calls"][0]["rows"]
    first_day_first_ex = next(
        r for r in rows
        if r["week"] == 1 and r["day_name"] == draft["days"][0]["title"]
        and r["exercise_order"] == 1
    )
    assert first_day_first_ex["exercise_id"] == "ex_swap_target_xyz"


@pytest.mark.asyncio
async def test_tool_writes_4_weeks_for_each_resolved_exercise(tool_supabase):
    from cases.case6_unified_agent.tools import save_workout_plan

    draft = _valid_draft(num_days=2, exercises_per_day=3)

    raw = await save_workout_plan.ainvoke({
        "user_id": "user-uuid",
        "draft_json": json.dumps(draft),
    })
    result = json.loads(raw)

    assert result["success"] is True
    assert result["workouts_created"] == 4 * 2 * 3   # weeks * days * exercises
    assert result["weeks"] == 4
    assert result["days_per_week"] == 2


@pytest.mark.asyncio
async def test_tool_rejects_insufficient_exercises_per_day(tool_supabase):
    """A day with <3 exercises must produce success=False with the
    exercises_per_day map so the caller can diagnose the bad day."""
    from cases.case6_unified_agent.tools import save_workout_plan

    draft = _valid_draft(num_days=2, exercises_per_day=3)
    # Truncate day 0 to a single exercise
    draft["days"][0]["exercises"] = draft["days"][0]["exercises"][:1]

    raw = await save_workout_plan.ainvoke({
        "user_id": "user-uuid",
        "draft_json": json.dumps(draft),
    })
    result = json.loads(raw)

    assert result["success"] is False
    assert "insuficientes" in result["error"].lower()
    assert "exercises_per_day" in result
    # And no destructive write happened
    assert tool_supabase["bulk_insert_calls"] == []


@pytest.mark.asyncio
async def test_tool_skips_users_plans_on_renewal(tool_supabase):
    """is_renewal=True must skip the users_plans insert — renewal updates the
    plan elsewhere; here we only want the new workouts."""
    from cases.case6_unified_agent.tools import save_workout_plan

    draft = _valid_draft(num_days=2, exercises_per_day=3)
    draft["is_renewal"] = True

    raw = await save_workout_plan.ainvoke({
        "user_id": "user-uuid",
        "draft_json": json.dumps(draft),
    })
    result = json.loads(raw)

    assert result["success"] is True
    inserted_tables = [c["table"] for c in tool_supabase["insert_calls"]]
    assert "users_plans" not in inserted_tables
    # But workouts WERE bulk-inserted
    assert len(tool_supabase["bulk_insert_calls"]) == 1


@pytest.mark.asyncio
async def test_tool_rejects_equipment_mismatch(tool_supabase):
    """If the user only has bodyweight but the draft includes a barbell
    exercise, save_workout_plan must fail with EQUIPMENT_MISMATCH and NOT
    write any workouts."""
    from cases.case6_unified_agent.tools import save_workout_plan

    draft = _valid_draft(num_days=2, exercises_per_day=3)
    # User can only do bodyweight
    tool_supabase["allowed_equipment"] = {"bodyweight"}
    # Make the exercises query return barbell equipment for everything
    resolved_ids = sorted({
        ex["exercise_id"]
        for day in draft["days"]
        for ex in day["exercises"]
    })
    tool_supabase["query_results"]["exercises"] = [
        {"exercise_id": eid, "equipment": "barbell", "spanish_name": "Barbell move"}
        for eid in resolved_ids
    ]

    raw = await save_workout_plan.ainvoke({
        "user_id": "user-uuid",
        "draft_json": json.dumps(draft),
    })
    result = json.loads(raw)

    assert result["success"] is False
    assert result["error"] == "EQUIPMENT_MISMATCH"
    assert "violations" in result
    assert len(result["violations"]) == len(resolved_ids)
    # No bulk insert happened
    assert tool_supabase["bulk_insert_calls"] == []
