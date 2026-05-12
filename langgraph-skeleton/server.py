"""FastAPI server — Kairos AI fitness coach.

Usage:
    source .venv/bin/activate
    uvicorn server:app --reload --port 8000

Endpoints:
    POST /api/v1/chat                          — Send message to Kairos agent
    GET  /api/v1/chat/:thread_id/history       — View conversation history
    GET  /api/v1/chat/:thread_id/debug         — Debug raw state
    GET  /webhook                              — WhatsApp webhook verification
    POST /webhook                              — WhatsApp → Kairos → WhatsApp
"""

import asyncio
import logging
import os
import time as _time
import traceback
from contextlib import asynccontextmanager
from datetime import datetime, timezone

import httpx
from fastapi import FastAPI, HTTPException, Query, Request
from pydantic import BaseModel, Field
from langchain_core.messages import HumanMessage, ToolMessage

logger = logging.getLogger("kairos")

from cases.case6_unified_agent.graph import build_case6_graph
from cases.case6_unified_agent.graph_live import build_case6_live_workflow
from src.shared.checkpointer import postgres_checkpointer_context


# ═══════════════ KYC NUDGE TRACKER ═══════════════

# In-memory registry of active KYC sessions for nudge checking.
# Key: thread_id, Value: {"last_interaction_at": ISO str, "nudge_sent": bool, "phone": str}
_kyc_sessions: dict[str, dict] = {}

NUDGE_DELAY_SECONDS = 30 * 60  # 30 minutes
NUDGE_CHECK_INTERVAL = 5 * 60  # Check every 5 minutes

# ═══════════════ WEBHOOK DEDUP & LOCKING ═══════════════

_user_locks: dict[str, asyncio.Lock] = {}  # per-phone serialization

_background_tasks: set[asyncio.Task] = set()  # prevent GC of fire-and-forget tasks


async def _is_duplicate_message(message_id: str, phone_from: str) -> bool:
    """Return True if this message ID was already processed (Supabase-backed, multi-instance safe)."""
    from src.shared.supabase_client import SUPABASE_URL, SUPABASE_ANON_KEY
    if not SUPABASE_URL:
        return False  # no Supabase → skip dedup
    url = f"{SUPABASE_URL}/rest/v1/processed_webhook_messages"
    headers = {
        "apikey": SUPABASE_ANON_KEY,
        "Authorization": f"Bearer {SUPABASE_ANON_KEY}",
        "Content-Type": "application/json",
        "Prefer": "return=representation,resolution=ignore-duplicates",
    }
    try:
        async with httpx.AsyncClient() as client:
            resp = await client.post(
                url,
                headers=headers,
                json={"message_id": message_id, "phone_from": phone_from},
                params={"on_conflict": "message_id"},
                timeout=5,
            )
            if resp.status_code == 201 and resp.json():
                return False  # inserted → first time seeing this message
            return True  # conflict / empty → duplicate
    except Exception as e:
        logger.warning(f"[WA] Dedup check failed: {e} — processing anyway")
        return False  # on error, allow processing (better than dropping messages)


def _get_user_lock(phone_number: str) -> asyncio.Lock:
    """Get or create an asyncio.Lock for a specific user."""
    if phone_number not in _user_locks:
        _user_locks[phone_number] = asyncio.Lock()
    return _user_locks[phone_number]


_FALLBACK_RESPONSE = "Lo siento, tuve un problema procesando tu mensaje. Intenta de nuevo en un momento."


async def _invoke_graph_safe(
    phone_number: str,
    message: str | list,
    display_name: str = "",
    *,
    source: str = "api",
    max_retries: int = 2,
) -> dict:
    """Thread-safe graph invocation with per-user locking, retries, and logging.

    Guarantees a non-empty 'response' field on failure (returns fallback).
    All entry points (API, webhook, future) MUST use this function.
    """
    lock = _get_user_lock(phone_number)
    t_start = _time.monotonic()
    tag = source.upper()

    async with lock:
        lock_ms = int((_time.monotonic() - t_start) * 1000)
        if lock_ms > 100:
            logger.info(f"[{tag}] {phone_number} lock_wait={lock_ms}ms")

        thread_id = f"case6_{phone_number}"
        config = {"configurable": {"thread_id": thread_id}}
        graph = case6_live_graph if os.getenv("SUPABASE_URL") else case6_graph

        last_error = None
        for attempt in range(max_retries):
            try:
                t_invoke = _time.monotonic()
                result = await graph.ainvoke(
                    {
                        "messages": [HumanMessage(content=message)],
                        "phone_number": phone_number,
                        "display_name": display_name,
                    },
                    config,
                )
                invoke_ms = int((_time.monotonic() - t_invoke) * 1000)
                logger.info(f"[{tag}] {phone_number} graph_invoke={invoke_ms}ms")

                # Guardrail: warn on empty response
                if not result.get("response"):
                    logger.warning(f"[{tag}] {phone_number} graph returned empty response")

                return result

            except Exception as e:
                last_error = e
                logger.error(
                    f"[{tag}] {phone_number} attempt {attempt + 1}/{max_retries}: "
                    f"{type(e).__name__}: {e}\n{traceback.format_exc()}"
                )
                if attempt < max_retries - 1:
                    await asyncio.sleep(1)

        # All retries exhausted — return fallback (never raise)
        logger.error(f"[{tag}] {phone_number} ALL RETRIES FAILED: {last_error}")
        return {"response": _FALLBACK_RESPONSE, "messages": [], "user_context": {}}


def _on_background_task_done(task: asyncio.Task):
    """Log unhandled exceptions in fire-and-forget background tasks."""
    _background_tasks.discard(task)
    if task.cancelled():
        return
    exc = task.exception()
    if exc:
        logger.error(
            f"[BG-TASK] Unhandled exception: {type(exc).__name__}: {exc}\n"
            f"{''.join(traceback.format_exception(type(exc), exc, exc.__traceback__))}"
        )


async def _nudge_checker():
    """Background task: check for stale KYC sessions and send WhatsApp nudge."""
    while True:
        await asyncio.sleep(NUDGE_CHECK_INTERVAL)
        now = datetime.now(timezone.utc)
        for thread_id, session in list(_kyc_sessions.items()):
            if session.get("nudge_sent"):
                continue
            try:
                last_dt = datetime.fromisoformat(session["last_interaction_at"])
                gap = (now - last_dt).total_seconds()
                if gap > NUDGE_DELAY_SECONDS:
                    phone = session.get("phone", "")
                    phone_number_id = session.get("phone_number_id", "")
                    if phone and phone_number_id:
                        nudge_text = (
                            "¡Hola! Vi que estábamos armando tu rutina. "
                            "¿Quieres que sigamos? Solo escríbeme y retomamos donde quedamos. 💪"
                        )
                        try:
                            await _send_whatsapp_message(phone_number_id, phone, nudge_text)
                            logger.info(f"[NUDGE] Sent nudge to {phone} (thread {thread_id})")
                        except Exception as e:
                            logger.error(f"[NUDGE] Failed to send to {phone}: {e}")
                    else:
                        logger.warning(f"[NUDGE] Session {thread_id} missing phone or phone_number_id")
                    session["nudge_sent"] = True
            except (ValueError, KeyError):
                pass


@asynccontextmanager
async def lifespan(app):
    """Initialize async resources on startup, cleanup on shutdown."""
    global case6_live_graph

    async with postgres_checkpointer_context() as checkpointer:
        case6_live_graph = case6_live_workflow.compile(checkpointer=checkpointer)
        cp_name = type(checkpointer).__name__
        print(f"[STARTUP] Case 6 live graph compiled (checkpointer: {cp_name})")
        task = asyncio.create_task(_nudge_checker())
        yield
        task.cancel()


# ═══════════════ APP ═══════════════

app = FastAPI(
    title="Kairos API",
    description="Kairos AI fitness coach — WhatsApp + REST API",
    version="1.0.0",
    lifespan=lifespan,
)

# Build graphs once at startup
case6_graph = build_case6_graph()
case6_live_workflow = build_case6_live_workflow()
case6_live_graph = None  # Compiled async in lifespan with checkpointer


# ═══════════════ SCHEMAS ═══════════════

class ChatRequest(BaseModel):
    message: str = Field(description="User message for Kairos agent")
    phone_number: str = Field(description="Full phone number (e.g., 573001234567)")
    display_name: str = Field(default="", description="WhatsApp display name (optional)")
    send_whatsapp: bool = Field(default=False, description="If true, send agent response via WhatsApp")


# ═══════════════ API v1 — Chat ═══════════════

@app.post("/api/v1/chat", tags=["Chat"])
async def chat(req: ChatRequest):
    """Send a message to the Kairos agent.

    Uses `phone_number` as thread identifier (one thread per user).
    Set `send_whatsapp=true` to also send the response via WhatsApp.
    """
    # Filter empty messages
    if not req.message or not req.message.strip():
        return {"response": "", "thread_id": f"case6_{req.phone_number}", "filtered": True}

    result = await _invoke_graph_safe(
        req.phone_number, req.message, req.display_name, source="api",
    )

    response_text = result.get("response", "")

    # Optionally send response via WhatsApp
    if req.send_whatsapp and response_text:
        phone_number_id = os.getenv("WA_PHONE_NUMBER_ID", WHATSAPP_PHONE_NUMBER_ID)
        await _send_whatsapp_message(phone_number_id, req.phone_number, response_text)

    ctx = result.get("user_context", {})
    return {
        "response": response_text,
        "thread_id": f"case6_{req.phone_number}",
        "is_new_user": ctx.get("is_new_user", False),
        "kyc_complete": ctx.get("kyc_complete", False),
    }


@app.get("/api/v1/chat/{phone_number}/history", tags=["Chat"])
async def chat_history(phone_number: str):
    """View conversation history for a thread."""
    thread_id = f"case6_{phone_number}"
    config = {"configurable": {"thread_id": thread_id}}

    graph = case6_live_graph if os.getenv("SUPABASE_URL") else case6_graph

    try:
        state = await graph.aget_state(config)
        if not state.values:
            raise HTTPException(status_code=404, detail=f"No conversation found for phone_number {phone_number}")

        messages = [
            {
                "role": "user" if isinstance(m, HumanMessage) else "assistant",
                "content": m.content,
            }
            for m in state.values.get("messages", [])
            if hasattr(m, "content") and not isinstance(m, ToolMessage)
        ]

        return {
            "thread_id": thread_id,
            "message_count": len(messages),
            "messages": messages,
        }
    except HTTPException:
        raise
    except Exception:
        raise HTTPException(status_code=404, detail=f"No conversation found for phone_number {phone_number}")


@app.get("/api/v1/chat/{phone_number}/debug", tags=["Chat"])
async def chat_debug(phone_number: str):
    """Debug: raw state including tool_calls and ToolMessages."""
    thread_id = f"case6_{phone_number}"
    config = {"configurable": {"thread_id": thread_id}}

    graph = case6_live_graph if os.getenv("SUPABASE_URL") else case6_graph

    try:
        state = await graph.aget_state(config)
        if not state.values:
            raise HTTPException(status_code=404, detail="No state found")

        raw_messages = []
        for m in state.values.get("messages", []):
            msg_type = type(m).__name__
            entry = {"type": msg_type, "content": str(m.content)[:500]}
            if hasattr(m, "tool_calls") and m.tool_calls:
                entry["tool_calls"] = [{"name": tc["name"], "args": tc["args"]} for tc in m.tool_calls]
            if hasattr(m, "name"):
                entry["tool_name"] = m.name
            raw_messages.append(entry)

        return {
            "thread_id": thread_id,
            "user_context": state.values.get("user_context", {}),
            "raw_messages": raw_messages,
        }
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))



# ═══════════════ WHATSAPP WEBHOOK (Direct) ═══════════════

WHATSAPP_API_URL = "https://graph.facebook.com/v21.0"
WHATSAPP_TOKEN = os.getenv("WHATSAPP_TOKEN", "")
WHATSAPP_VERIFY_TOKEN = os.getenv("WHATSAPP_VERIFY_TOKEN", "kairos-verify-token")
WHATSAPP_PHONE_NUMBER_ID = os.getenv("WHATSAPP_PHONE_NUMBER_ID", "")


async def _send_whatsapp_message(phone_number_id: str, to: str, text: str):
    """Send a text message via WhatsApp Business API."""
    url = f"{WHATSAPP_API_URL}/{phone_number_id}/messages"
    headers = {
        "Authorization": f"Bearer {WHATSAPP_TOKEN}",
        "Content-Type": "application/json",
    }
    body = {
        "messaging_product": "whatsapp",
        "to": to,
        "type": "text",
        "text": {"body": text},
    }
    async with httpx.AsyncClient() as client:
        resp = await client.post(url, headers=headers, json=body, timeout=30)
        if resp.status_code != 200:
            logger.error(f"WhatsApp send failed: {resp.status_code} {resp.text}")


async def _download_whatsapp_media(media_id: str) -> tuple[str, str] | None:
    """Download media from WhatsApp API, return (base64_data, mime_type) or None.

    Two-step process:
    1. GET media metadata from Graph API → obtain temporary download URL
    2. GET binary data from that URL
    """
    import base64

    headers = {"Authorization": f"Bearer {WHATSAPP_TOKEN}"}
    try:
        async with httpx.AsyncClient(timeout=30) as client:
            # Step 1: Get download URL
            meta_resp = await client.get(
                f"{WHATSAPP_API_URL}/{media_id}", headers=headers,
            )
            if meta_resp.status_code != 200:
                logger.error(f"[WA] Media meta failed: {meta_resp.status_code}")
                return None
            meta_json = meta_resp.json()
            download_url = meta_json.get("url")
            if not download_url:
                return None

            # Step 2: Download binary
            data_resp = await client.get(download_url, headers=headers)
            if data_resp.status_code != 200:
                logger.error(f"[WA] Media download failed: {data_resp.status_code}")
                return None

            b64 = base64.b64encode(data_resp.content).decode("utf-8")
            mime = meta_json.get("mime_type", "image/jpeg")
            return b64, mime
    except httpx.TimeoutException:
        logger.error(f"[WA] Media download timeout for {media_id}")
        return None
    except Exception as e:
        logger.error(f"[WA] Media download error: {e}")
        return None


async def _extract_message(data: dict) -> tuple[str | list, str, str, str, str] | None:
    """Extract (message_body, phone_from, display_name, phone_number_id, msg_id) from webhook payload.

    Returns None if the payload is not a valid user message (status update, audio, etc).
    message_body is str for text messages, list[dict] for image messages (langchain content blocks).
    """
    entry = data.get("entry", [])
    if not entry:
        return None

    changes = entry[0].get("changes", [])
    if not changes:
        return None

    value = changes[0].get("value", {})
    messages = value.get("messages", [])
    if not messages:
        return None

    msg = messages[0]
    msg_type = msg.get("type", "")
    msg_id = msg.get("id", "")  # WhatsApp message ID (wamid.xxx)

    # Filter audio and unsupported types
    if msg_type == "audio":
        return None

    phone_from = msg.get("from", "")
    if not phone_from:
        return None

    # Extract content based on message type
    body: str | list = ""
    if msg_type == "text":
        body = msg.get("text", {}).get("body", "")
    elif msg_type == "button":
        body = msg.get("button", {}).get("payload", "") or msg.get("button", {}).get("text", "")
    elif msg_type == "interactive":
        interactive = msg.get("interactive", {})
        body = (interactive.get("button_reply", {}).get("title", "")
                or interactive.get("list_reply", {}).get("title", ""))
    elif msg_type == "image":
        image_info = msg.get("image", {})
        media_id = image_info.get("id")
        if not media_id:
            return None
        media_result = await _download_whatsapp_media(media_id)
        if media_result is None:
            body = "[IMAGE_DOWNLOAD_FAILED]"
        else:
            b64_data, mime_type = media_result
            caption = image_info.get("caption", "").strip()
            content_parts: list[dict] = []
            if caption:
                content_parts.append({"type": "text", "text": caption})
            content_parts.append({
                "type": "image_url",
                "image_url": {"url": f"data:{mime_type};base64,{b64_data}"},
            })
            if not caption:
                content_parts.append({"type": "text", "text": "El usuario envió esta imagen."})
            body = content_parts
    else:
        body = msg.get("text", {}).get("body", "")

    if not body or (isinstance(body, str) and not body.strip()):
        return None

    # Get display name and phone_number_id
    contacts = value.get("contacts", [])
    display_name = contacts[0].get("profile", {}).get("name", "") if contacts else ""
    phone_number_id = value.get("metadata", {}).get("phone_number_id", WHATSAPP_PHONE_NUMBER_ID)

    return (body.strip() if isinstance(body, str) else body), phone_from, display_name, phone_number_id, msg_id


@app.get("/webhook", tags=["WhatsApp Webhook"])
async def whatsapp_verify(
    hub_mode: str = Query(None, alias="hub.mode"),
    hub_challenge: str = Query(None, alias="hub.challenge"),
    hub_verify_token: str = Query(None, alias="hub.verify_token"),
):
    """WhatsApp webhook verification (challenge-response)."""
    if hub_mode == "subscribe" and hub_verify_token == WHATSAPP_VERIFY_TOKEN:
        return int(hub_challenge)
    raise HTTPException(status_code=403, detail="Verification failed")


async def _process_message_background(
    message_body: str | list,
    phone_from: str,
    display_name: str,
    phone_number_id: str,
    msg_id: str,
):
    """Process a WhatsApp message in the background (after 200 already returned)."""
    t_start = _time.monotonic()

    result = await _invoke_graph_safe(
        phone_from, message_body, display_name, source="webhook",
    )

    # Track KYC sessions for nudge checker
    ctx = result.get("user_context", {})
    is_new = ctx.get("is_new_user", False)
    kyc_done = ctx.get("kyc_complete", False)
    has_plan = bool(ctx.get("plan"))
    thread_id = f"case6_{phone_from}"
    if is_new or (kyc_done and not has_plan):
        _kyc_sessions[thread_id] = {
            "last_interaction_at": datetime.now(timezone.utc).isoformat(),
            "nudge_sent": False,
            "phone": phone_from,
            "phone_number_id": phone_number_id,
        }
    elif thread_id in _kyc_sessions and has_plan:
        # User completed KYC + got plan — remove from nudge tracking
        _kyc_sessions.pop(thread_id, None)
    elif thread_id in _kyc_sessions:
        # Update last interaction time
        _kyc_sessions[thread_id]["last_interaction_at"] = datetime.now(timezone.utc).isoformat()

    response_text = result.get("response", "")
    if response_text:
        t_send = _time.monotonic()
        await _send_whatsapp_message(phone_number_id, phone_from, response_text)
        send_ms = int((_time.monotonic() - t_send) * 1000)
        logger.info(f"[WEBHOOK] {phone_from} wa_send={send_ms}ms")

    total_ms = int((_time.monotonic() - t_start) * 1000)
    logger.info(f"[WEBHOOK] {phone_from} TOTAL={total_ms}ms msg_id={msg_id}")


@app.post("/webhook", tags=["WhatsApp Webhook"])
async def whatsapp_webhook(request: Request):
    """Receive WhatsApp messages — return 200 immediately, process in background."""
    data = await request.json()

    extracted = await _extract_message(data)
    if not extracted:
        return {"status": "ignored"}

    message_body, phone_from, display_name, phone_number_id, msg_id = extracted

    # Dedup: skip if we've already seen this message ID (Supabase-backed)
    if msg_id and await _is_duplicate_message(msg_id, phone_from):
        logger.info(f"[WA] Duplicate {msg_id} from {phone_from} — skip")
        return {"status": "duplicate"}

    # Handle failed image downloads — short-circuit (cheap, no background needed)
    if message_body == "[IMAGE_DOWNLOAD_FAILED]":
        await _send_whatsapp_message(
            phone_number_id, phone_from,
            "No pude descargar tu imagen. ¿Puedes enviarla de nuevo?",
        )
        return {"status": "ok"}

    log_preview = message_body[:50] if isinstance(message_body, str) else "[image]"
    logger.info(f"[WA] {phone_from} ({display_name}): {log_preview} [msg_id={msg_id}]")

    # Fire-and-forget: process in background, return 200 immediately
    task = asyncio.create_task(
        _process_message_background(
            message_body, phone_from, display_name, phone_number_id, msg_id,
        )
    )
    _background_tasks.add(task)
    task.add_done_callback(_on_background_task_done)

    return {"status": "ok"}


# ═══════════════ API v1 — Drafts ═══════════════
#
# Direct draft finalization: bypasses the LLM so what the user approved in the
# preview UI (after any swaps) is exactly what gets persisted to workouts.
# Idempotent: if the user already has an active plan, returns the existing
# plan_id with workouts_created=0.

import json as _json
import uuid as _uuid

from cases.case6_unified_agent.tools import save_workout_plan as _save_workout_plan_tool
from src.shared.supabase_client import supabase_query as _supabase_query


@app.post("/api/v1/drafts/{code}/finalize", tags=["Drafts"])
async def finalize_draft(code: str):
    """Materialize a draft into users_plans + workouts.

    Contract:
      - 200: success or idempotent no-op (workouts_created=0)
      - 404: draft not found / expired
      - 422: draft content rejected by save_workout_plan validation
      - 5xx: upstream error
    """
    # 1. Load draft by code (any status — approve flow may flip it before we run)
    rows = await _supabase_query(
        "draft_routines",
        select="user_id,draft_data,status",
        filters={"code": f"eq.{code}"},
        limit=1,
    )
    if not rows:
        raise HTTPException(status_code=404, detail={"success": False, "error": "draft not found"})

    user_id = rows[0]["user_id"]
    draft_data = rows[0]["draft_data"]

    # 2. Idempotency: if this user already has an active plan, short-circuit.
    existing_plans = await _supabase_query(
        "users_plans",
        select="plan_id",
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
        limit=1,
    )
    if existing_plans:
        return {
            "success": True,
            "plan_id": existing_plans[0]["plan_id"],
            "workouts_created": 0,
            "user_id": user_id,
        }

    # 3. Materialize via the existing tool. We use .ainvoke so the tool's
    #    @tool wrapper doesn't trip on missing config; the tool returns a JSON
    #    string with success/plan_id/created_workouts/etc.
    try:
        raw_result = await _save_workout_plan_tool.ainvoke({
            "user_id": user_id,
            "draft_json": _json.dumps(draft_data),
        })
    except Exception as e:
        logger.exception(f"[finalize] save_workout_plan failed for code={code}")
        raise HTTPException(status_code=500, detail={"success": False, "error": str(e)})

    try:
        parsed = _json.loads(raw_result)
    except Exception:
        raise HTTPException(status_code=500, detail={"success": False, "error": "invalid tool response"})

    if not parsed.get("success", False):
        # Validation failure surfaced by save_workout_plan (e.g., insufficient
        # exercises per day). Treat as 422 so the Go backend doesn't retry blindly.
        raise HTTPException(status_code=422, detail={"success": False, "error": parsed.get("error", "validation failed")})

    return {
        "success": True,
        "plan_id": parsed.get("plan_id"),
        "workouts_created": parsed.get("created_workouts", parsed.get("workouts_created", 0)),
        "user_id": user_id,
    }


# ═══════════════ HEALTH CHECK ═══════════════

@app.get("/", tags=["Health"])
async def root():
    """Health check + lista de endpoints disponibles."""
    return {
        "status": "ok",
        "project": "LangGraph Skeleton — GymBot",
        "endpoints": {
            "POST /api/v1/chat": "Send message to Kairos agent",
            "GET /api/v1/chat/{phone_number}/history": "View conversation history",
            "GET /api/v1/chat/{phone_number}/debug": "Debug raw state",
            "GET /webhook": "WhatsApp webhook verification",
            "POST /webhook": "WhatsApp → Kairos → WhatsApp",
            "GET /docs": "Swagger UI",
        },
    }
