"""Run Case 5: Onboarding KYC — interactive 5-turn conversation.

Usage: python -m cases.case5_onboarding_kyc.run
"""

import asyncio
from langchain_core.messages import HumanMessage
from cases.case5_onboarding_kyc.graph import build_case5_graph


async def main():
    graph = build_case5_graph()

    print("=" * 60)
    print("CASE 5: Onboarding KYC (5-turn, 10 fields)")
    print("=" * 60)
    print()
    print("Simulating a new user completing the full KYC flow.")
    print("Phone: 573001234567 | Name: Camilo")
    print()

    config = {"configurable": {"thread_id": "kyc_573001234567"}}
    base_input = {
        "phone_number": "573001234567",
        "display_name": "Camilo",
    }

    turns = [
        ("Hola, quiero empezar mi rutina", "Turn 1: Initial greeting + goal"),
        (
            "Quiero ganar masa muscular, tengo 3 años de experiencia",
            "Turn 2: Goal + experience",
        ),
        (
            "Entreno en gym, 3 días a la semana, por las mañanas",
            "Turn 3: Environment + schedule",
        ),
        ("Soy hombre, 27 años, 171 cm, 67 kg", "Turn 4: Body metrics"),
        (
            "No tengo ninguna lesión ni condición",
            "Turn 5: Health status",
        ),
    ]

    for user_msg, description in turns:
        print(f"--- {description} ---")
        print(f"  User: {user_msg}")

        result = await graph.ainvoke(
            {**base_input, "messages": [HumanMessage(content=user_msg)]},
            config,
        )

        print(f"  Kairos: {result.get('response', '')[:200]}")
        print(f"  Turn: {result.get('current_turn', '?')} | Complete: {result.get('is_complete', False)}")

        collected = result.get("collected_data", {})
        if collected:
            print(f"  Collected: {list(collected.keys())}")
        print()

    # After all 5 turns, check if confirmation is pending
    if result.get("awaiting_confirmation"):
        print("--- Confirmation ---")
        confirm_msg = "Sí, todo está correcto"
        print(f"  User: {confirm_msg}")

        result = await graph.ainvoke(
            {**base_input, "messages": [HumanMessage(content=confirm_msg)]},
            config,
        )
        print(f"  Kairos: {result.get('response', '')[:200]}")
        print(f"  Profile saved: {result.get('profile_confirmed', False)}")
        print(f"  Health code: {result.get('health_code', 'N/A')}")
        print()

    # Show final state
    print("--- Final State ---")
    state = await graph.aget_state(config)
    vals = state.values
    print(f"  Total messages: {len(vals.get('messages', []))}")
    print(f"  Collected data: {vals.get('collected_data', {})}")
    print(f"  Health code: {vals.get('health_code', 'N/A')}")
    print(f"  Route to trainer: {vals.get('route_to_trainer', False)}")
    print(f"  Profile confirmed: {vals.get('profile_confirmed', False)}")

    print()
    print("=" * 60)
    print("Case 5 completed!")


if __name__ == "__main__":
    asyncio.run(main())
