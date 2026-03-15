"""Case 5: Onboarding KYC StateGraph (mock/in-memory).

7-node graph:
  START → check_user → (existing→END, new→kyc_agent)
    → check_status → (continue→END, complete→confirm_profile, correction→kyc_agent)
    → confirm_profile → (awaiting→END, accepted→health_classifier, rejected→kyc_agent)
    → health_classifier → (safe A-D→save_profile, severe E→route_to_trainer)
    → save_profile → END
    → route_to_trainer → END

Uses InMemorySaver checkpointer for multi-turn persistence via thread_id.
"""

from langgraph.checkpoint.memory import InMemorySaver
from langgraph.graph import StateGraph, START, END

from cases.case5_onboarding_kyc.state import KYCState
from cases.case5_onboarding_kyc.nodes import (
    check_user,
    kyc_agent,
    check_status,
    confirm_profile,
    health_classifier,
    save_profile,
    route_to_trainer,
    route_after_check_user,
    route_after_check_status,
    route_after_confirm,
    route_after_health,
)


def build_case5_graph():
    """Build the Case 5 KYC graph with InMemorySaver checkpointer."""
    workflow = StateGraph(KYCState)

    # --- Nodes ---
    workflow.add_node("check_user", check_user)
    workflow.add_node("kyc_agent", kyc_agent)
    workflow.add_node("check_status", check_status)
    workflow.add_node("confirm_profile", confirm_profile)
    workflow.add_node("health_classifier", health_classifier)
    workflow.add_node("save_profile", save_profile)
    workflow.add_node("route_to_trainer", route_to_trainer)

    # --- Edges ---

    # START → check_user
    workflow.add_edge(START, "check_user")

    # check_user → kyc_agent (new) or END (existing)
    workflow.add_conditional_edges(
        "check_user",
        route_after_check_user,
        ["kyc_agent", END],
    )

    # kyc_agent → check_status
    workflow.add_edge("kyc_agent", "check_status")

    # check_status → END (wait), confirm_profile (complete), kyc_agent (correction),
    #                health_classifier (user confirmed profile)
    workflow.add_conditional_edges(
        "check_status",
        route_after_check_status,
        ["confirm_profile", "kyc_agent", "health_classifier", END],
    )

    # confirm_profile → health_classifier (accepted), kyc_agent (rejected), END (awaiting)
    workflow.add_conditional_edges(
        "confirm_profile",
        route_after_confirm,
        ["health_classifier", "kyc_agent", END],
    )

    # health_classifier → save_profile (safe) or route_to_trainer (severe)
    workflow.add_conditional_edges(
        "health_classifier",
        route_after_health,
        ["save_profile", "route_to_trainer"],
    )

    # Terminal nodes
    workflow.add_edge("save_profile", END)
    workflow.add_edge("route_to_trainer", END)

    # --- Compile with checkpointer ---
    checkpointer = InMemorySaver()
    graph = workflow.compile(checkpointer=checkpointer)

    return graph
