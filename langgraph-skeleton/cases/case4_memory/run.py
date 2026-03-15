"""Run Case 4: Checkpointing + Multi-turn Memory with Gemini KYC.

Usage: python -m cases.case4_memory.run
"""

import asyncio
from langchain_core.messages import HumanMessage
from cases.case4_memory.graph import build_case4_graph


async def main():
    graph = build_case4_graph()

    print("=" * 60)
    print("CASE 4: Multi-turn KYC with Memory (Gemini Real)")
    print("=" * 60)

    # Same thread_id across all turns → Gemini remembers context
    config = {"configurable": {"thread_id": "camilo-kyc-session"}}

    # --- Turn 1: User starts ---
    print("\n--- Turn 1 ---")
    print("  User: Hola, quiero crear mi rutina de gym")
    result1 = await graph.ainvoke(
        {"messages": [HumanMessage(content="Hola, quiero crear mi rutina de gym")]},
        config,
    )
    print(f"  Kairos: {result1['response']}")
    print(f"  Complete: {result1.get('is_complete', False)}")

    # --- Turn 2: User provides goal ---
    print("\n--- Turn 2 ---")
    print("  User: Quiero ganar masa muscular")
    result2 = await graph.ainvoke(
        {"messages": [HumanMessage(content="Quiero ganar masa muscular")]},
        config,
    )
    print(f"  Kairos: {result2['response']}")
    print(f"  Complete: {result2.get('is_complete', False)}")

    # --- Turn 3: User provides days + level (two at once) ---
    print("\n--- Turn 3 ---")
    print("  User: Puedo entrenar 3 dias, soy nivel intermedio")
    result3 = await graph.ainvoke(
        {"messages": [HumanMessage(content="Puedo entrenar 3 dias, soy nivel intermedio")]},
        config,
    )
    print(f"  Kairos: {result3['response']}")
    print(f"  Complete: {result3.get('is_complete', False)}")

    if result3.get("workout_plan"):
        print(f"\n  --- Generated Workout Plan ---")
        print(f"  {result3['workout_plan'][:500]}")

    # --- Verify thread isolation ---
    print("\n--- Thread Isolation Test ---")
    different_config = {"configurable": {"thread_id": "different-user-session"}}
    result_isolated = await graph.ainvoke(
        {"messages": [HumanMessage(content="Hola")]},
        different_config,
    )
    print(f"  Different thread response: {result_isolated['response'][:100]}...")
    print(f"  Is isolated (not complete): {not result_isolated.get('is_complete', False)}")

    # --- Show conversation history ---
    print("\n--- Conversation History (Thread: camilo-kyc-session) ---")
    state = await graph.aget_state(config)
    for i, msg in enumerate(state.values["messages"]):
        role = "User" if isinstance(msg, HumanMessage) else "Kairos"
        content = msg.content[:80] + "..." if len(msg.content) > 80 else msg.content
        print(f"  [{i}] {role}: {content}")

    print("\n" + "=" * 60)
    print("Case 4 completed!")


if __name__ == "__main__":
    asyncio.run(main())
