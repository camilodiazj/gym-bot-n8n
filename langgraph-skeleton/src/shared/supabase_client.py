"""Supabase REST API client using httpx.

Uses the PostgREST API (Supabase's auto-generated REST layer)
to query tables directly via HTTP — no extra SDK needed.

Uses a module-level AsyncClient for connection pooling (reuses TCP/TLS
connections across requests instead of creating a new one per query).
"""

import os
import httpx
from dotenv import load_dotenv

load_dotenv()

SUPABASE_URL = os.getenv("SUPABASE_URL", "")
SUPABASE_ANON_KEY = os.getenv("SUPABASE_ANON_KEY", "")

# Global HTTP client with connection pooling — reuses TCP/TLS connections.
# httpx.AsyncClient is safe to share across async tasks.
_http_client: httpx.AsyncClient | None = None


def _get_client() -> httpx.AsyncClient:
    """Get or create the shared HTTP client."""
    global _http_client
    if _http_client is None or _http_client.is_closed:
        _http_client = httpx.AsyncClient(
            timeout=httpx.Timeout(30.0, connect=10.0),
            limits=httpx.Limits(max_connections=20, max_keepalive_connections=10),
        )
    return _http_client


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

    client = _get_client()
    response = await client.get(url, headers=_get_headers(), params=params)
    response.raise_for_status()
    return response.json()


async def supabase_insert(
    table: str,
    data: dict,
    upsert: bool = False,
    on_conflict: str | None = None,
) -> list[dict]:
    """Insert a row into a Supabase table via PostgREST REST API.

    Args:
        table: Table name (e.g., "users", "users_gym_profile")
        data: Row data as a dict
        upsert: If True, use merge-duplicates resolution for upserts
        on_conflict: Columns for conflict resolution (e.g., "user_id,week,week_day")

    Returns:
        List of inserted row dicts (with Prefer: return=representation)
    """
    url = f"{SUPABASE_URL}/rest/v1/{table}"
    headers = _get_headers()
    headers["Prefer"] = "return=representation"
    if upsert:
        headers["Prefer"] += ",resolution=merge-duplicates"

    params = {}
    if on_conflict:
        params["on_conflict"] = on_conflict

    client = _get_client()
    response = await client.post(url, headers=headers, json=data, params=params)
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

    client = _get_client()
    response = await client.patch(url, headers=headers, json=data, params=filters)
    response.raise_for_status()
    return response.json()


async def supabase_bulk_insert(
    table: str,
    rows: list[dict],
    upsert: bool = False,
    on_conflict: str | None = None,
) -> list[dict]:
    """Bulk insert multiple rows into a Supabase table via PostgREST.

    Args:
        table: Table name (e.g., "workouts")
        rows: List of row dicts to insert
        upsert: If True, use merge-duplicates resolution for upserts
        on_conflict: Columns for conflict resolution (e.g., "user_id,week,week_day")

    Returns:
        List of inserted row dicts
    """
    url = f"{SUPABASE_URL}/rest/v1/{table}"
    headers = _get_headers()
    headers["Prefer"] = "return=representation"
    if upsert:
        headers["Prefer"] += ",resolution=merge-duplicates"

    params = {}
    if on_conflict:
        params["on_conflict"] = on_conflict

    client = _get_client()
    response = await client.post(url, headers=headers, json=rows, params=params)
    response.raise_for_status()
    return response.json()


async def supabase_delete(
    table: str,
    filters: dict[str, str],
) -> list[dict]:
    """Delete rows from a Supabase table via PostgREST DELETE.

    Args:
        table: Table name (e.g., "user_weekly_schedule")
        filters: PostgREST filters for WHERE clause (REQUIRED)
                 (e.g., {"user_id": "eq.uuid-here"})

    Returns:
        List of deleted row dicts

    Raises:
        ValueError: If filters dict is empty (safety guard)
    """
    if not filters:
        raise ValueError("supabase_delete requires at least one filter (safety guard)")

    url = f"{SUPABASE_URL}/rest/v1/{table}"
    headers = _get_headers()
    headers["Prefer"] = "return=representation"

    client = _get_client()
    response = await client.delete(url, headers=headers, params=filters)
    response.raise_for_status()
    return response.json()
