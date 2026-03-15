"""Tests for Case 4: Multi-turn Memory with Gemini KYC.

Tests:
- First turn is not complete (asks for data)
- Multi-turn accumulates context
- Different threads are isolated
- KYC eventually completes
"""

import pytest
from langchain_core.messages import HumanMessage
from cases.case4_memory.graph import build_case4_graph


@pytest.fixture
def graph():
    return build_case4_graph()


@pytest.mark.asyncio
async def test_first_turn_not_complete(graph):
    """First turn should NOT be complete — Kairos should ask for data."""
    config = {"configurable": {"thread_id": "test-kyc-1"}}

    result = await graph.ainvoke(
        {"messages": [HumanMessage(content="Hola, quiero crear mi rutina")]},
        config,
    )

    assert result["is_complete"] is False
    assert result["response"] != ""
    assert len(result["response"]) > 5


@pytest.mark.asyncio
async def test_multi_turn_accumulates_context(graph):
    """Multiple turns should accumulate messages via checkpointer."""
    config = {"configurable": {"thread_id": "test-kyc-2"}}

    # Turn 1
    r1 = await graph.ainvoke(
        {"messages": [HumanMessage(content="Quiero crear mi rutina")]},
        config,
    )
    assert r1["is_complete"] is False

    # Turn 2 — provide goal
    r2 = await graph.ainvoke(
        {"messages": [HumanMessage(content="Ganar masa muscular")]},
        config,
    )

    # Should have accumulated messages (turn 1 + turn 2)
    state = await graph.aget_state(config)
    assert len(state.values["messages"]) >= 4  # At least: H1 + AI1 + H2 + AI2


@pytest.mark.asyncio
async def test_different_threads_isolated(graph):
    """Different thread_ids should have isolated conversations."""
    config_a = {"configurable": {"thread_id": "test-kyc-thread-a"}}
    config_b = {"configurable": {"thread_id": "test-kyc-thread-b"}}

    # Thread A: provide data
    await graph.ainvoke(
        {"messages": [HumanMessage(content="Quiero ganar masa muscular")]},
        config_a,
    )

    # Thread B: fresh start
    r_b = await graph.ainvoke(
        {"messages": [HumanMessage(content="Hola")]},
        config_b,
    )

    # Thread B should not have Thread A's context
    assert r_b["is_complete"] is False

    # Thread B should have only 2 messages (Human + AI)
    state_b = await graph.aget_state(config_b)
    assert len(state_b.values["messages"]) == 2


@pytest.mark.asyncio
async def test_kyc_completes_with_all_data(graph):
    """Providing all data should eventually complete the KYC."""
    config = {"configurable": {"thread_id": "test-kyc-complete"}}

    # Turn 1 — greeting
    await graph.ainvoke(
        {"messages": [HumanMessage(content="Hola quiero mi rutina")]},
        config,
    )

    # Turn 2 — give ALL three data points at once
    result = await graph.ainvoke(
        {"messages": [HumanMessage(
            content="Quiero ganar masa muscular, puedo 3 dias a la semana, soy intermedio"
        )]},
        config,
    )

    assert result["is_complete"] is True
    # Should have generated a workout plan
    assert result.get("workout_plan", "") != ""
