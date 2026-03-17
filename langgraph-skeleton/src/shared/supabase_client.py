"""Supabase REST API client using httpx.

Uses the PostgREST API (Supabase's auto-generated REST layer)
to query tables directly via HTTP — no extra SDK needed.
"""

import os
import httpx
from dotenv import load_dotenv

load_dotenv()

SUPABASE_URL = os.getenv("SUPABASE_URL", "")
SUPABASE_ANON_KEY = os.getenv("SUPABASE_ANON_KEY", "")


def _get_headers() -> dict:
    return {
        "apikey": SUPABASE_ANON_KEY,
        "Authorization": f"Bearer {SUPABASE_ANON_KEY}",
        "Content-Type": "application/json",
    }


async def supabase_query(
    table: str,
    select: str = "*",
    filters: dict[str, str] | None = None,
    limit: int | None = None,
) -> list[dict]:
    """Query a Supabase table via PostgREST REST API.

    Args:
        table: Table name (e.g., "exercises", "set_profiles")
        select: Columns to select (e.g., "exercise_id,spanish_name,pattern")
        filters: PostgREST filters (e.g., {"pattern": "eq.push_h", "level": "eq.Intermedio"})
        limit: Max rows to return

    Returns:
        List of row dicts

    Example:
        exercises = await supabase_query(
            "exercises",
            select="exercise_id,spanish_name,pattern,role,main_muscle,level,equipment",
            filters={"pattern": "eq.push_h", "level": "eq.Intermedio"},
            limit=10,
        )
    """
    url = f"{SUPABASE_URL}/rest/v1/{table}"

    params = {"select": select}
    if filters:
        params.update(filters)
    if limit:
        params["limit"] = str(limit)

    async with httpx.AsyncClient() as client:
        response = await client.get(url, headers=_get_headers(), params=params)
        response.raise_for_status()
        return response.json()


async def supabase_insert(
    table: str,
    data: dict,
    upsert: bool = False,
) -> list[dict]:
    """Insert a row into a Supabase table via PostgREST REST API.

    Args:
        table: Table name (e.g., "users", "users_gym_profile")
        data: Row data as a dict
        upsert: If True, use merge-duplicates resolution for upserts

    Returns:
        List of inserted row dicts (with Prefer: return=representation)
    """
    url = f"{SUPABASE_URL}/rest/v1/{table}"
    headers = _get_headers()
    headers["Prefer"] = "return=representation"
    if upsert:
        headers["Prefer"] += ",resolution=merge-duplicates"

    async with httpx.AsyncClient() as client:
        response = await client.post(url, headers=headers, json=data)
        response.raise_for_status()
        return response.json()


async def supabase_update(
    table: str,
    data: dict,
    filters: dict[str, str],
) -> list[dict]:
    """Update rows in a Supabase table via PostgREST PATCH.

    Args:
        table: Table name (e.g., "user_weekly_schedule")
        data: Fields to update (e.g., {"Completed": True})
        filters: PostgREST filters for WHERE clause
                 (e.g., {"user_id": "eq.uuid-here", "planned_day": "eq.2026-03-17"})

    Returns:
        List of updated row dicts
    """
    url = f"{SUPABASE_URL}/rest/v1/{table}"
    headers = _get_headers()
    headers["Prefer"] = "return=representation"

    async with httpx.AsyncClient() as client:
        response = await client.patch(url, headers=headers, json=data, params=filters)
        response.raise_for_status()
        return response.json()


async def supabase_bulk_insert(
    table: str,
    rows: list[dict],
) -> list[dict]:
    """Bulk insert multiple rows into a Supabase table via PostgREST.

    Args:
        table: Table name (e.g., "workouts")
        rows: List of row dicts to insert

    Returns:
        List of inserted row dicts
    """
    url = f"{SUPABASE_URL}/rest/v1/{table}"
    headers = _get_headers()
    headers["Prefer"] = "return=representation"

    async with httpx.AsyncClient() as client:
        response = await client.post(url, headers=headers, json=rows)
        response.raise_for_status()
        return response.json()
