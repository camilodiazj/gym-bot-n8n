"""Graph nodes for the Unified Agent Kairos (Case 6).

Nodes:
- load_context: Deterministic — queries Supabase for UserContext
- router: Deterministic — routes new users to KYC, existing to agent
- kairos_agent: Agentic — Gemini with tools (ReAct loop)
- should_continue: Routing function for ReAct tool loop
"""

from langchain_core.messages import AIMessage, SystemMessage

from cases.case6_unified_agent.state import UnifiedAgentState
from cases.case6_unified_agent.context_loader import load_user_context
from cases.case6_unified_agent.prompts import KAIROS_SYSTEM_PROMPT, format_user_context
from src.shared.llm import get_llm


async def load_context(state: UnifiedAgentState) -> dict:
    """Load user context from Supabase (deterministic, no LLM)."""
    phone = state.get("phone_number", "")
    ctx = await load_user_context(phone)

    # Use display_name from context if available, else keep input
    display_name = ctx.get("full_name") or state.get("display_name", "")

    return {
        "user_context": ctx,
        "display_name": display_name,
    }


def router(state: UnifiedAgentState) -> str:
    """Route new users to KYC, existing users to agent.

    Returns node name: 'kyc_subgraph' or 'kairos_agent'.
    """
    ctx = state.get("user_context", {})
    is_new = ctx.get("is_new_user", False)
    kyc_done = ctx.get("kyc_complete", False)

    if is_new and not kyc_done:
        return "kyc_subgraph"
    return "kairos_agent"


async def kairos_agent(state: UnifiedAgentState) -> dict:
    """Gemini agent node — decides what to do based on context + message.

    In Phase 2 (foundational), runs without tools.
    Tools are bound in graph_live.py via bind_tools().
    """
    ctx = state.get("user_context", {})
    formatted_ctx = format_user_context(ctx)

    system_prompt = KAIROS_SYSTEM_PROMPT.format(
        user_context_formatted=formatted_ctx,
    )

    # Build messages: system prompt + conversation history
    messages = [SystemMessage(content=system_prompt)] + list(state.get("messages", []))

    llm = get_llm(temperature=0.3)
    response = await llm.ainvoke(messages)

    return {
        "messages": [response],
        "response": response.content,
    }


def should_continue(state: UnifiedAgentState) -> str:
    """ReAct loop: continue if LLM made tool calls, else done."""
    messages = state.get("messages", [])
    if not messages:
        return "done"

    last_msg = messages[-1]
    if isinstance(last_msg, AIMessage) and last_msg.tool_calls:
        return "tools"
    return "done"


# ═══════════════ KYC STATE MAPPING ═══════════════

def unified_to_kyc_input(state: UnifiedAgentState) -> dict:
    """Map UnifiedAgentState → KYCState input for subgraph invocation."""
    return {
        "messages": state.get("messages", []),
        "phone_number": state.get("phone_number", ""),
        "display_name": state.get("display_name", ""),
        "is_new_user": True,
    }


def kyc_output_to_unified(kyc_result: dict) -> dict:
    """Map KYCState output → UnifiedAgentState updates."""
    return {
        "messages": kyc_result.get("messages", []),
        "response": kyc_result.get("response", ""),
    }
