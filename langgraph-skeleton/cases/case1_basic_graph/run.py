"""Run Case 1: Basic Graph with Gemini summary.

Usage: python -m cases.case1_basic_graph.run
"""

import asyncio
from cases.case1_basic_graph.graph import build_case1_graph


async def main():
    graph = build_case1_graph()

    print("=" * 60)
    print("CASE 1: Basic Graph — State + Nodes + Edges + Gemini")
    print("=" * 60)

    # Test 1: Valid user
    print("\n--- Test 1: Valid user (Camilo) ---")
    result = await graph.ainvoke({"user_id": "camilo-001"})
    print(f"  Profile: {result['profile']['full_name']}")
    print(f"  Valid: {result['is_valid']}")
    print(f"  Summary: {result['summary']}")

    # Test 2: Another valid user
    print("\n--- Test 2: Valid user (Ana) ---")
    result = await graph.ainvoke({"user_id": "ana-002"})
    print(f"  Profile: {result['profile']['full_name']}")
    print(f"  Valid: {result['is_valid']}")
    print(f"  Summary: {result['summary']}")

    # Test 3: Unknown user
    print("\n--- Test 3: Unknown user ---")
    result = await graph.ainvoke({"user_id": "unknown-999"})
    print(f"  Profile: {result['profile']}")
    print(f"  Valid: {result['is_valid']}")
    print(f"  Errors: {result['validation_errors']}")
    print(f"  Summary: {result['summary']}")

    print("\n" + "=" * 60)
    print("Case 1 completed!")


if __name__ == "__main__":
    asyncio.run(main())
