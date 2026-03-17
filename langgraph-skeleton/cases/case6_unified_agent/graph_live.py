"""Case 6 LIVE: Unified Agent Kairos with Supabase context + tools + KYC subgraph.

Architecture:
  START → load_context → router ─→ kairos_agent ↔ tool_node → END
                          └─────→ kyc_subgraph → END

Tools bound: US1 tools. More tools added in later phases.
KYC subgraph: Case 5 live graph imported as-is.
"""

from langchain_core.messages import AIMessage, SystemMessage
from langgraph.checkpoint.memory import InMemorySaver
from langgraph.graph import StateGraph, START, END
from langgraph.prebuilt import ToolNode

from cases.case5_onboarding_kyc.graph_live import build_case5_live_graph
from cases.case6_unified_agent.state import UnifiedAgentState
from cases.case6_unified_agent.context_loader import load_user_context
from cases.case6_unified_agent.prompts import KAIROS_SYSTEM_PROMPT, format_user_context
from cases.case6_unified_agent.tools import (
    # US1: Daily operations
    get_todays_routine,
    confirm_workout_completion,
    decline_workout,
    create_magic_link,
    # US3: Draft routine creation
    get_day_requirements,
    get_exercises_for_draft,
    find_exercise_alternatives,
    save_workout_plan,
    # US4: Scheduling
    get_schedule_info,
    schedule_sessions,
    # US6: Mesocycle renewal
    get_mesocycle_status,
)
from src.shared.llm import get_llm


# ═══════════════ ALL 11 TOOLS ═══════════════

TOOLS = [
    # US1: Daily operations
    get_todays_routine,
    confirm_workout_completion,
    decline_workout,
    create_magic_link,
    # US3: Draft routine creation
    get_day_requirements,
    get_exercises_for_draft,
    find_exercise_alternatives,
    save_workout_plan,
    # US4: Scheduling
    get_schedule_info,
    schedule_sessions,
    # US6: Mesocycle renewal
    get_mesocycle_status,
]


# ═══════════════ LIVE NODES ═══════════════

async def load_context(state: UnifiedAgentState) -> dict:
    """Load user context from Supabase (deterministic, no LLM)."""
    phone = state.get("phone_number", "")
    ctx = await load_user_context(phone)
    display_name = ctx.get("full_name") or state.get("display_name", "")

    return {
        "user_context": ctx,
        "display_name": display_name,
    }


def router(state: UnifiedAgentState) -> str:
    """Route new users to KYC, existing users to agent."""
    ctx = state.get("user_context", {})
    is_new = ctx.get("is_new_user", False)
    kyc_done = ctx.get("kyc_complete", False)

    if is_new and not kyc_done:
        return "kyc_subgraph"
    return "kairos_agent"


async def kairos_agent(state: UnifiedAgentState) -> dict:
    """Gemini agent with tools — decides freely based on context + message."""
    ctx = state.get("user_context", {})
    formatted_ctx = format_user_context(ctx)

    system_prompt = KAIROS_SYSTEM_PROMPT.format(
        user_context_formatted=formatted_ctx,
    )

    messages = [SystemMessage(content=system_prompt)] + list(state.get("messages", []))

    llm = get_llm(temperature=0.3)
    model_with_tools = llm.bind_tools(TOOLS)
    response = await model_with_tools.ainvoke(messages)

    return {
        "messages": [response],
        "response": response.content if response.content else "",
    }


def should_continue(state: UnifiedAgentState) -> str:
    """ReAct loop: continue to tools if LLM made tool calls, else done."""
    messages = state.get("messages", [])
    if not messages:
        return "done"

    last_msg = messages[-1]
    if isinstance(last_msg, AIMessage) and last_msg.tool_calls:
        return "tools"
    return "done"


# ═══════════════ KYC SUBGRAPH WRAPPER ═══════════════

# Build KYC graph once (Case 5 live — Supabase persistence)
_kyc_graph = build_case5_live_graph()


async def kyc_subgraph_node(state: UnifiedAgentState) -> dict:
    """Wrapper: invokes Case 5 KYC subgraph with state mapping.

    Maps UnifiedAgentState → KYCState input, invokes, maps output back.
    The KYC subgraph uses its own internal checkpointer for multi-turn.
    """
    # Map input: only pass what KYC needs
    kyc_input = {
        "messages": state.get("messages", []),
        "phone_number": state.get("phone_number", ""),
        "display_name": state.get("display_name", ""),
        "is_new_user": True,
    }

    # Use a KYC-specific thread_id so KYC state is isolated
    kyc_thread_id = f"kyc_{state.get('phone_number', '')}"
    config = {"configurable": {"thread_id": kyc_thread_id}}

    result = await _kyc_graph.ainvoke(kyc_input, config)

    # Map output back to UnifiedAgentState
    return {
        "messages": result.get("messages", []),
        "response": result.get("response", ""),
    }


# ═══════════════ BUILD GRAPH ═══════════════

def build_case6_live_graph():
    """Build Case 6 live graph with Supabase context + tools + KYC subgraph.

    Graph:
      START → load_context → router ─→ kairos_agent ↔ tool_node → END
                               └─────→ kyc_subgraph → END
    """
    workflow = StateGraph(UnifiedAgentState)

    workflow.add_node("load_context", load_context)
    workflow.add_node("kairos_agent", kairos_agent)
    workflow.add_node("tool_node", ToolNode(TOOLS))
    workflow.add_node("kyc_subgraph", kyc_subgraph_node)

    # Edges
    workflow.add_edge(START, "load_context")

    # Router: new user → KYC, existing → agent
    workflow.add_conditional_edges(
        "load_context",
        router,
        {"kairos_agent": "kairos_agent", "kyc_subgraph": "kyc_subgraph"},
    )

    # KYC subgraph → END
    workflow.add_edge("kyc_subgraph", END)

    # ReAct loop for agent
    workflow.add_conditional_edges(
        "kairos_agent",
        should_continue,
        {"tools": "tool_node", "done": END},
    )
    workflow.add_edge("tool_node", "kairos_agent")

    checkpointer = InMemorySaver()
    return workflow.compile(checkpointer=checkpointer)
