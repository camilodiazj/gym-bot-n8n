"""Context loader — parallel Supabase queries to build UserContext.

Runs 5 queries via asyncio.gather to minimize latency (~150ms vs ~500ms sequential).
"""

import asyncio
from datetime import datetime, timedelta, timezone

from cases.case6_unified_agent.state import UserContext
from src.shared.supabase_client import supabase_query

MISSED_SESSIONS_WINDOW_DAYS = 3
BOGOTA_UTC_OFFSET = timedelta(hours=-5)


def _today_bogota() -> str:
    """Return today's date in Bogota timezone as YYYY-MM-DD string."""
    now_utc = datetime.now(timezone.utc)
    now_bogota = now_utc + BOGOTA_UTC_OFFSET
    return now_bogota.strftime("%Y-%m-%d")


def _days_ago_bogota(days: int) -> str:
    """Return date N days ago in Bogota timezone as YYYY-MM-DD string."""
    now_utc = datetime.now(timezone.utc)
    now_bogota = now_utc + BOGOTA_UTC_OFFSET
    target = now_bogota - timedelta(days=days)
    return target.strftime("%Y-%m-%d")


async def load_user_context(phone_number: str) -> UserContext:
    """Load full user context from Supabase in parallel queries.

    Returns a UserContext snapshot with all flags computed.
    """
    # Phase 1: Check if user exists + KYC profile
    user_result, kyc_result = await asyncio.gather(
        supabase_query(
            "users",
            select="user_id,full_name,email,timezone",
            filters={"full_phone_number": f"eq.{phone_number}"},
        ),
        supabase_query(
            "users_gym_profile",
            select="whatsapp_id",
            filters={"whatsapp_id": f"eq.{phone_number}"},
            limit=1,
        ),
    )

    if not user_result:
        return UserContext(
            user_id=None,
            full_name="",
            phone_number=phone_number,
            plan=None,
            todays_sessions=[],
            missed_sessions=[],
            next_scheduled_session=None,
            pending_tasks=[],
            is_new_user=True,
            kyc_complete=bool(kyc_result),
            has_schedule=False,
            all_w4_completed=False,
        )

    user = user_result[0]
    user_id = user["user_id"]

    # Phase 2: Load plan, schedule, and pending tasks in parallel
    today = _today_bogota()
    window_start = _days_ago_bogota(MISSED_SESSIONS_WINDOW_DAYS)

    plan_result, schedule_result, pending_result = await asyncio.gather(
        supabase_query(
            "users_plans",
            select="plan_id,goal,level,week_schedule,mesocycle_number,status",
            filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
            limit=1,
        ),
        supabase_query(
            "user_weekly_schedule",
            select='day_routine_id,session_name,week,planned_day,"Completed"',
            filters={
                "user_id": f"eq.{user_id}",
                "planned_day_utc": f"gte.{window_start}",
            },
        ),
        supabase_query(
            "pending_tasks",
            select="task_id,task_type,session_name,status",
            filters={"user_id": f"eq.{user_id}", "status": "eq.pending"},
        ),
    )

    # Process plan
    plan = None
    if plan_result:
        p = plan_result[0]
        plan = {
            "plan_id": p["plan_id"],
            "goal": p["goal"],
            "level": p["level"],
            "week_schedule": p["week_schedule"],
            "mesocycle_number": p["mesocycle_number"],
            "status": p["status"],
        }

    # Process schedule into: today's sessions, missed sessions, next session
    todays_sessions = []
    missed_sessions = []
    future_sessions = []

    for row in schedule_result:
        planned = row.get("planned_day", "")
        completed = row.get("Completed", False)

        if planned == today:
            todays_sessions.append(row)
        elif planned < today and not completed:
            missed_sessions.append(row)
        elif planned > today:
            future_sessions.append(row)

    # Sort future sessions to find the next one
    future_sessions.sort(key=lambda r: r.get("planned_day", ""))
    next_scheduled = None
    if future_sessions:
        ns = future_sessions[0]
        next_scheduled = {
            "session_name": ns["session_name"],
            "planned_day": ns["planned_day"],
        }

    # Compute flags
    has_schedule = bool(todays_sessions or future_sessions)

    # all_w4_completed: check if ALL week-4 sessions are completed
    all_w4_completed = False
    if plan:
        w4_sessions = [
            r for r in schedule_result if r.get("week") == 4
        ]
        if w4_sessions:
            all_w4_completed = all(r.get("Completed", False) for r in w4_sessions)

    return UserContext(
        user_id=user_id,
        full_name=user.get("full_name", ""),
        phone_number=phone_number,
        plan=plan,
        todays_sessions=todays_sessions,
        missed_sessions=missed_sessions,
        next_scheduled_session=next_scheduled,
        pending_tasks=pending_result,
        is_new_user=False,
        kyc_complete=bool(kyc_result),
        has_schedule=has_schedule,
        all_w4_completed=all_w4_completed,
    )
