"""Tests for Case 3: Tools + Agent Loop with Gemini.

Tests:
- Agent calls tools and produces a workout
- Tool messages appear in the conversation
- Agent loop terminates (no infinite loop)
"""

import pytest
from langchain_core.messages import ToolMessage, AIMessage
from cases.case3_tools_agent.graph import build_case3_graph


@pytest.fixture
def graph():
    return build_case3_graph()


@pytest.mark.asyncio
async def test_agent_produces_workout(graph):
    """Agent should produce a non-empty workout result."""
    result = await graph.ainvoke({"user_id": "camilo-001"})

    assert result["workout_result"] != ""
    assert len(result["workout_result"]) > 20


@pytest.mark.asyncio
async def test_agent_calls_tools(graph):
    """Agent should call at least one tool (ToolMessages in messages)."""
    result = await graph.ainvoke({"user_id": "camilo-001"})

    tool_msgs = [m for m in result["messages"] if isinstance(m, ToolMessage)]
    assert len(tool_msgs) >= 1, "Agent should call at least one tool"


@pytest.mark.asyncio
async def test_agent_loop_terminates(graph):
    """Agent loop should terminate with reasonable number of messages."""
    result = await graph.ainvoke({"user_id": "camilo-001"})

    # Reasonable limit: system + user + multiple (AI + Tool) rounds + AI final
    assert len(result["messages"]) <= 20, (
        f"Too many messages ({len(result['messages'])}), possible infinite loop"
    )


@pytest.mark.asyncio
async def test_profile_and_template_loaded(graph):
    """Profile and template should be loaded correctly."""
    result = await graph.ainvoke({"user_id": "camilo-001"})

    assert result["profile"]["full_name"] == "Camilo Diaz"
    assert result["template"]["template_name"] == "Full Body A"
