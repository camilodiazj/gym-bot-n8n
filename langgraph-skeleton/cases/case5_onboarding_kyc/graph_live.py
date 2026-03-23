"""Case 5 LIVE: Onboarding KYC with Supabase persistence.

Same 7-node graph as graph.py, but:
- check_user queries Supabase users table via lookup_user_by_phone
- save_profile writes to Supabase users + users_gym_profile tables

Checkpointer is injected externally (PostgreSQL in production).
"""

import json

from langchain_core.messages import AIMessage
from langgraph.graph import StateGraph, START, END

from cases.case5_onboarding_kyc.state import KYCState
from cases.case5_onboarding_kyc.nodes import (
    kyc_agent,
    check_status,
    confirm_profile,
    health_classifier,
    route_to_trainer,
    route_after_check_user,
    route_after_check_status,
    route_after_confirm,
    route_after_health,
)
from cases.case5_onboarding_kyc.tools_supabase import (
    lookup_user_by_phone,
    save_user,
    save_gym_profile,
)


# ═══════════════ LIVE NODES (override mock versions) ═══════════════

async def check_user_live(state: KYCState) -> dict:
    """Query Supabase for existing user by phone number."""
    from datetime import datetime, timezone

    phone = state.get("phone_number", "")
    display_name = state.get("display_name", "")

    result_str = await lookup_user_by_phone.ainvoke({"phone": phone})
    result = json.loads(result_str)

    if result.get("found"):
        return {
            "is_new_user": False,
            "display_name": result["user"].get("full_name", display_name),
            "response": f"¡Hola {result['user'].get('full_name', '')}! Ya tienes un perfil registrado. 🎉",
            "last_interaction_at": datetime.now(timezone.utc).isoformat(),
        }

    return {
        "is_new_user": True,
        "display_name": display_name if display_name else "Hola",
        "last_interaction_at": datetime.now(timezone.utc).isoformat(),
    }


async def save_profile_live(state: KYCState) -> dict:
    """Save completed KYC profile to Supabase users + users_gym_profile."""
    collected = state.get("collected_data", {})
    # Prefer full_name from KYC collected data over WhatsApp display_name
    display_name = collected.get("full_name") or state.get("display_name", "") or "Usuario"
    phone = state.get("phone_number", "")
    health_code = state.get("health_code", "A")

    # 1. Create user in users table
    user_result_str = await save_user.ainvoke({
        "full_name": display_name,
        "phone": phone,
        "email": collected.get("email", ""),
    })
    user_result = json.loads(user_result_str)

    # 2. Save gym profile
    profile_result_str = await save_gym_profile.ainvoke({
        "phone": phone,
        "display_name": display_name,
        "collected_data": json.dumps(collected),
        "health_code": health_code,
    })

    success_msg = (
        f"✅ ¡Perfil guardado exitosamente en Supabase, {display_name}! "
        "Ahora voy a crear tu rutina personalizada. 💪"
    )

    return {
        "profile_confirmed": True,
        "awaiting_confirmation": False,
        "messages": [AIMessage(content=success_msg)],
        "response": success_msg,
    }


# ═══════════════ BUILD GRAPH ═══════════════

def build_case5_live_graph(checkpointer=None):
    """Build Case 5 LIVE graph with Supabase-connected check_user and save_profile.

    Args:
        checkpointer: LangGraph checkpointer (e.g. AsyncPostgresSaver). If None, compiles without one.
    """
    workflow = StateGraph(KYCState)

    # --- Nodes (live versions for check_user and save_profile) ---
    workflow.add_node("check_user", check_user_live)
    workflow.add_node("kyc_agent", kyc_agent)
    workflow.add_node("check_status", check_status)
    workflow.add_node("confirm_profile", confirm_profile)
    workflow.add_node("health_classifier", health_classifier)
    workflow.add_node("save_profile", save_profile_live)
    workflow.add_node("route_to_trainer", route_to_trainer)

    # --- Edges (same topology as mock graph) ---
    workflow.add_edge(START, "check_user")

    workflow.add_conditional_edges(
        "check_user",
        route_after_check_user,
        ["kyc_agent", END],
    )

    workflow.add_edge("kyc_agent", "check_status")

    workflow.add_conditional_edges(
        "check_status",
        route_after_check_status,
        ["confirm_profile", "kyc_agent", "health_classifier", END],
    )

    workflow.add_conditional_edges(
        "confirm_profile",
        route_after_confirm,
        ["health_classifier", "kyc_agent", END],
    )

    workflow.add_conditional_edges(
        "health_classifier",
        route_after_health,
        ["save_profile", "route_to_trainer"],
    )

    workflow.add_edge("save_profile", END)
    workflow.add_edge("route_to_trainer", END)

    # --- Compile (checkpointer injected externally for production) ---
    return workflow.compile(checkpointer=checkpointer)
