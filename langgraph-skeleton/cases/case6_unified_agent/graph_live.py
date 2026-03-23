"""Case 6 LIVE: Unified Agent Kairos with Supabase context + tools.

Architecture:
  START → load_context → kairos_agent ↔ tool_node → END

All users (new and existing) go to kairos_agent. New user onboarding
is handled conversationally by the agent using register_new_user tool.
"""

import logging
import re

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langgraph.graph import StateGraph, START, END
from langgraph.prebuilt import ToolNode

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
    renew_maintain,
    renew_change_days,
    renew_rotate_exercises,
    # Email
    send_routine_email,
    # Onboarding & profile
    register_new_user,
    update_user_profile,
    # Draft preview
    save_draft_preview,
)
from src.shared.llm import get_llm


# ═══════════════ ALL TOOLS ═══════════════

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
    renew_maintain,
    renew_change_days,
    renew_rotate_exercises,
    # Email
    send_routine_email,
    # Onboarding & profile
    register_new_user,
    update_user_profile,
    # Draft preview
    save_draft_preview,
]


# ═══════════════ TOOL-LEAK GUARDRAIL ═══════════════

logger = logging.getLogger("kairos")

_TOOL_NAMES = [t.name for t in TOOLS]
_names_re = "|".join(re.escape(n) for n in _TOOL_NAMES)
_TOOL_LEAK_RE = re.compile(
    rf"(?:print\s*\(|default_api\.|(?:{_names_re})\s*\()",
    re.IGNORECASE,
)

_TOOL_LEAK_CORRECTION = (
    "[SISTEMA] Tu respuesta anterior contenía código Python visible. "
    "NUNCA escribas funciones, print() ni código. "
    "Usa tool_call para ejecutar herramientas. "
    "Responde al usuario en texto natural."
)


def _has_tool_leak(text: str) -> bool:
    """Detect if LLM wrote tool calls as text instead of using tool_call."""
    return bool(_TOOL_LEAK_RE.search(text)) if text else False


def _sanitize_response(text: str) -> str:
    """Strip leaked tool-call patterns from response text (last defense)."""
    if not text:
        return text
    cleaned = re.sub(r"print\s*\((?:[^()]*|\([^()]*\))*\)", "", text)
    cleaned = re.sub(r"default_api\.\w+\((?:[^()]*|\([^()]*\))*\)", "", cleaned)
    cleaned = re.sub(
        rf"(?:{_names_re})\s*\([^)]*\)", "", cleaned, flags=re.IGNORECASE
    )
    cleaned = re.sub(r"\n{3,}", "\n\n", cleaned).strip()
    return cleaned or "Dame un momento, estoy procesando tu solicitud."


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


async def kairos_agent(state: UnifiedAgentState) -> dict:
    """Gemini agent with tools — decides freely based on context + message."""
    ctx = state.get("user_context", {})
    display_name = state.get("display_name", "")
    formatted_ctx = format_user_context(ctx, display_name=display_name)

    system_prompt = KAIROS_SYSTEM_PROMPT.format(
        user_context_formatted=formatted_ctx,
    )

    messages = [SystemMessage(content=system_prompt)] + list(state.get("messages", []))

    llm = get_llm(temperature=0.3)
    model_with_tools = llm.bind_tools(TOOLS)
    response = await model_with_tools.ainvoke(messages)

    # Normalize content (Gemini 3 may return list instead of str)
    content = response.content or ""
    if isinstance(content, list):
        content = "".join(
            p.get("text", "") if isinstance(p, dict) else str(p) for p in content
        )

    # Capa 1: Si el LLM escribió tool calls como texto sin tool_calls reales, retry una vez
    if content and not response.tool_calls and _has_tool_leak(content):
        logger.warning("[TOOL-LEAK] Detected text-based tool call, retrying")
        retry_messages = messages + [response, HumanMessage(content=_TOOL_LEAK_CORRECTION)]
        response = await model_with_tools.ainvoke(retry_messages)
        content = response.content or ""
        if isinstance(content, list):
            content = "".join(
                p.get("text", "") if isinstance(p, dict) else str(p) for p in content
            )

    # Capa 2: Sanitizar cualquier patrón residual en el texto visible al usuario
    if _has_tool_leak(content):
        content = _sanitize_response(content)
        response = AIMessage(content=content, tool_calls=response.tool_calls)

    result = {"messages": [response]}
    if content:  # Only update response if there's visible text for the user.
        # This prevents intermediate tool-call iterations (with empty content)
        # from overwriting a previous non-empty response.
        result["response"] = content
    return result


def should_continue(state: UnifiedAgentState) -> str:
    """ReAct loop: continue to tools if LLM made tool calls, else done."""
    messages = state.get("messages", [])
    if not messages:
        return "done"

    last_msg = messages[-1]
    if isinstance(last_msg, AIMessage) and last_msg.tool_calls:
        return "tools"
    return "done"


# ═══════════════ BUILD GRAPH ═══════════════

def build_case6_live_workflow():
    """Build Case 6 live workflow (uncompiled — checkpointer added by server.py).

    Graph:
      START → load_context → kairos_agent ↔ tool_node → END
    """
    workflow = StateGraph(UnifiedAgentState)

    workflow.add_node("load_context", load_context)
    workflow.add_node("kairos_agent", kairos_agent)
    workflow.add_node("tool_node", ToolNode(TOOLS))

    workflow.add_edge(START, "load_context")
    workflow.add_edge("load_context", "kairos_agent")
    workflow.add_conditional_edges(
        "kairos_agent",
        should_continue,
        {"tools": "tool_node", "done": END},
    )
    workflow.add_edge("tool_node", "kairos_agent")

    return workflow
