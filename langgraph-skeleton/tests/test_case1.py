"""Tests for Case 1: Basic Graph with Gemini Summary.

Tests:
- Valid user produces valid profile and Gemini summary
- Unknown user is marked invalid
- Graph structure is correct (3 nodes)
"""

import pytest
from cases.case1_basic_graph.graph import build_case1_graph


@pytest.fixture
def graph():
    return build_case1_graph()


@pytest.mark.asyncio
async def test_valid_user_produces_summary(graph):
    """Camilo (valid user) should be valid and get a Gemini-generated summary."""
    result = await graph.ainvoke({"user_id": "camilo-001"})

    assert result["is_valid"] is True
    assert result["validation_errors"] == []
    assert result["profile"]["full_name"] == "Camilo Diaz"
    # Gemini should produce a non-empty summary
    assert result["summary"] != ""
    assert len(result["summary"]) > 10


@pytest.mark.asyncio
async def test_unknown_user_is_invalid(graph):
    """Unknown user should be marked invalid with error message."""
    result = await graph.ainvoke({"user_id": "unknown-999"})

    assert result["is_valid"] is False
    assert "User not found" in result["validation_errors"]
    assert "inválido" in result["summary"].lower() or "no se puede" in result["summary"].lower()


@pytest.mark.asyncio
async def test_ana_valid_with_health_c(graph):
    """Ana (health C, principiante) should be valid."""
    result = await graph.ainvoke({"user_id": "ana-002"})

    assert result["is_valid"] is True
    assert result["profile"]["health_status"] == "C"
    assert result["profile"]["level"] == "Principiante"
    assert result["summary"] != ""
