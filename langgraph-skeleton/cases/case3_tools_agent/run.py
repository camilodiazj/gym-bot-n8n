"""Run Case 3: Tools + Agent Loop with Gemini.

Usage: python -m cases.case3_tools_agent.run
"""

import asyncio
from langchain_core.messages import ToolMessage
from cases.case3_tools_agent.graph import build_case3_graph


async def main():
    graph = build_case3_graph()

    print("=" * 60)
    print("CASE 3: Tools + Agent Loop (Gemini Real)")
    print("=" * 60)

    print("\n--- Creating workout for Camilo (Intermedio, Masa muscular) ---")
    result = await graph.ainvoke({"user_id": "camilo-001"})

    # Count message types
    tool_calls_count = sum(
        1 for m in result["messages"]
        if isinstance(m, ToolMessage)
    )

    print(f"\n  Total messages in conversation: {len(result['messages'])}")
    print(f"  Tool calls executed: {tool_calls_count}")
    print(f"\n  --- Generated Workout ---")
    print(f"  {result['workout_result'][:500]}...")

    # Show the tool calls that happened
    print(f"\n  --- Tool Call Details ---")
    for i, msg in enumerate(result["messages"]):
        if isinstance(msg, ToolMessage):
            print(f"  [{i}] Tool: {msg.name} → {msg.content[:100]}...")

    print("\n" + "=" * 60)
    print("Case 3 completed!")


if __name__ == "__main__":
    asyncio.run(main())
