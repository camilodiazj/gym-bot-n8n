"""Case 6 mock graph: Context Loader + Agent (no KYC, no tools).

For development and testing without Supabase.
The agent responds based on a mock UserContext.
"""

from langchain_core.messages import AIMessage, SystemMessage

from langgraph.checkpoint.memory import InMemorySaver
from langgraph.graph import StateGraph, START, END

from cases.case6_unified_agent.state import UnifiedAgentState, UserContext
from cases.case6_unified_agent.prompts import KAIROS_SYSTEM_PROMPT, format_user_context
from src.shared.llm import get_llm


# ═══════════════ MOCK NODES ═══════════════

async def load_context_mock(state: UnifiedAgentState) -> dict:
    """Load a mock UserContext (no Supabase)."""
    phone = state.get("phone_number", "")
    display_name = state.get("display_name", "Usuario")

    ctx = UserContext(
        user_id="mock-user-id",
        full_name=display_name,
        phone_number=phone,
        plan={
            "plan_id": "mock-plan",
            "goal": "Ganar masa muscular",
            "level": "Intermedio",
            "week_schedule": "ul_4",
            "mesocycle_number": 1,
            "status": "active",
        },
        todays_sessions=[{
            "session_name": "Upper Body A",
            "week": 2,
            "Completed": False,
            "planned_day": "2026-03-17",
        }],
        missed_sessions=[],
        next_scheduled_session={"session_name": "Lower Body A", "planned_day": "2026-03-19"},
        pending_tasks=[],
        is_new_user=False,
        kyc_complete=True,
        has_schedule=True,
        all_w4_completed=False,
    )

    return {"user_context": ctx, "display_name": display_name}


async def kairos_agent_mock(state: UnifiedAgentState) -> dict:
    """Gemini agent without tools (mock graph)."""
    ctx = state.get("user_context", {})
    formatted_ctx = format_user_context(ctx)

    system_prompt = KAIROS_SYSTEM_PROMPT.format(
        user_context_formatted=formatted_ctx,
    )

    messages = [SystemMessage(content=system_prompt)] + list(state.get("messages", []))

    llm = get_llm(temperature=0.3)
    response = await llm.ainvoke(messages)

    return {
        "messages": [response],
        "response": response.content,
    }


# ═══════════════ BUILD GRAPH ═══════════════

def build_case6_graph():
    """Build Case 6 mock graph (no Supabase, no KYC, no tools)."""
    workflow = StateGraph(UnifiedAgentState)

    workflow.add_node("load_context", load_context_mock)
    workflow.add_node("kairos_agent", kairos_agent_mock)

    workflow.add_edge(START, "load_context")
    workflow.add_edge("load_context", "kairos_agent")
    workflow.add_edge("kairos_agent", END)

    checkpointer = InMemorySaver()
    return workflow.compile(checkpointer=checkpointer)
