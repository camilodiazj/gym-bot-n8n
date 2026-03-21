"""Run Case 2: Conditional Routing + Gemini selection.

Usage: python -m cases.case2_conditional.run
"""

import asyncio
from cases.case2_conditional.graph import build_case2_graph


async def main():
    graph = build_case2_graph()

    print("=" * 60)
    print("CASE 2: Conditional Routing + Gemini Exercise Selection")
    print("=" * 60)

    # Test 1: Health A — no restrictions
    print("\n--- Test 1: Camilo (Health A — full pool) ---")
    result = await graph.ainvoke({"user_id": "camilo-001"})
    print(f"  Route: {result['route_taken']}")
    print(f"  Available pool: {len(result['available_exercises'])} exercises")
    print(f"  Gemini selected: {len(result['selected_exercises'])} exercises")
    for ex in result["selected_exercises"]:
        print(f"    - {ex['spanish_name']} ({ex['pattern']}, {ex['role']})")

    # Test 2: Health C — upper body restrictions
    print("\n--- Test 2: Ana (Health C — no overhead pressing) ---")
    result = await graph.ainvoke({"user_id": "ana-002"})
    print(f"  Route: {result['route_taken']}")
    print(f"  Available pool: {len(result['available_exercises'])} exercises")
    print(f"  Gemini selected: {len(result['selected_exercises'])} exercises")
    for ex in result["selected_exercises"]:
        print(f"    - {ex['spanish_name']} ({ex['pattern']}, {ex['role']})")
    # Verify no overhead press
    overhead = [e for e in result["available_exercises"] if "militar" in e["spanish_name"].lower()]
    print(f"  Overhead press in pool? {'YES (BUG!)' if overhead else 'NO (correct)'}")

    # Test 3: Health B — lower body restrictions
    print("\n--- Test 3: Carlos (Health B — no barbell squat/deadlift) ---")
    result = await graph.ainvoke({"user_id": "carlos-003"})
    print(f"  Route: {result['route_taken']}")
    print(f"  Available pool: {len(result['available_exercises'])} exercises")
    print(f"  Gemini selected: {len(result['selected_exercises'])} exercises")
    for ex in result["selected_exercises"]:
        print(f"    - {ex['spanish_name']} ({ex['pattern']}, {ex['role']})")

    print("\n" + "=" * 60)
    print("Case 2 completed!")


if __name__ == "__main__":
    asyncio.run(main())
