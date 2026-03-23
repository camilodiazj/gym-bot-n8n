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


def _parse_planned_date(row: dict) -> str:
    """Extract YYYY-MM-DD from planned_day_utc (preferred) or planned_day (fallback).

    planned_day_utc is a timestamptz populated by DB trigger — always reliable.
    planned_day is a text field in DD/MM/YYYY (Kairos) or YYYY-MM-DD (fixtures) format.
    """
    utc_val = row.get("planned_day_utc")
    if utc_val:
        return utc_val[:10]  # "2026-03-22T05:00:00+00:00" → "2026-03-22"

    # Fallback: parse planned_day text
    pd = row.get("planned_day", "")
    if "/" in pd:
        parts = pd.split("/")
        if len(parts) == 3:
            return f"{parts[2]}-{parts[1]}-{parts[0]}"  # DD/MM/YYYY → YYYY-MM-DD
    return pd  # Already YYYY-MM-DD or unknown format


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
            select="whatsapp_id,primary_goal,training_experience,days_available,preferred_schedule,training_environment,home_equipment,fitness_level,health_status,priority_muscles,disliked_exercises,biological_sex",
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
            gym_profile=kyc_result[0] if kyc_result else None,
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

    plan_result, schedule_result, pending_result, w4_result = await asyncio.gather(
        supabase_query(
            "users_plans",
            select="plan_id,goal,level,week_schedule,mesocycle_number,status",
            filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
            limit=1,
        ),
        supabase_query(
            "user_weekly_schedule",
            select='day_routine_id,session_name,week,planned_day,planned_day_utc,"Completed"',
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
        # Separate W4 query without date filter for mesocycle completion check
        supabase_query(
            "user_weekly_schedule",
            select='"Completed"',
            filters={"user_id": f"eq.{user_id}", "week": "eq.4"},
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
        planned_date = _parse_planned_date(row)
        completed = row.get("Completed", False)

        if planned_date == today:
            todays_sessions.append(row)
        elif planned_date < today and not completed:
            missed_sessions.append(row)
        elif planned_date > today:
            future_sessions.append(row)

    # Sort future sessions to find the next one
    future_sessions.sort(key=lambda r: _parse_planned_date(r))
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
    # Uses dedicated w4_result query (no date filter) to avoid missing old W4 sessions
    all_w4_completed = False
    if plan and w4_result:
        all_w4_completed = all(r.get("Completed", False) for r in w4_result)

    return UserContext(
        user_id=user_id,
        full_name=user.get("full_name", ""),
        phone_number=phone_number,
        plan=plan,
        todays_sessions=todays_sessions,
        missed_sessions=missed_sessions,
        next_scheduled_session=next_scheduled,
        pending_tasks=pending_result,
        gym_profile=kyc_result[0] if kyc_result else None,
        is_new_user=False,
        kyc_complete=bool(kyc_result),
        has_schedule=has_schedule,
        all_w4_completed=all_w4_completed,
    )
