"""Supabase tools for Case 5 KYC — user lookup and profile persistence.

Tools:
- lookup_user_by_phone: Query users table by full_phone_number
- save_user: Insert into users table
- save_gym_profile: Insert into users_gym_profile with enum mapping
"""

import json
import uuid
from datetime import datetime, timezone

from langchain_core.tools import tool

from src.shared.supabase_client import supabase_query, supabase_insert


@tool
async def lookup_user_by_phone(phone: str) -> str:
    """Look up a user in Supabase by their full phone number.

    Args:
        phone: Full phone number (e.g., "573001234567")

    Returns:
        JSON with user data if found, or {"found": false}
    """
    rows = await supabase_query(
        table="users",
        select="user_id,full_name,full_phone_number,email",
        filters={"full_phone_number": f"eq.{phone}"},
        limit=1,
    )
    if rows:
        return json.dumps({"found": True, "user": rows[0]}, ensure_ascii=False)
    return json.dumps({"found": False})


@tool
async def save_user(full_name: str, phone: str, email: str = "") -> str:
    """Create a new user in the Supabase users table.

    Args:
        full_name: User's display name
        phone: Full phone number (e.g., "573001234567")
        email: Optional email address

    Returns:
        JSON with created user data
    """
    user_id = str(uuid.uuid4())
    cel_number = int(phone) if phone.isdigit() else 0

    data = {
        "user_id": user_id,
        "full_name": full_name,
        "cel_number": cel_number,
        "full_phone_number": phone,
        "email": email or f"{phone}@gymbot.local",
        "timezone": "America/Bogota",
        "created_at": datetime.now(timezone.utc).isoformat(),
        "country_indicative": 57,
    }

    result = await supabase_insert(table="users", data=data)
    return json.dumps({"success": True, "user_id": user_id, "user": result}, ensure_ascii=False)


@tool
async def save_gym_profile(
    phone: str,
    display_name: str,
    collected_data: str,
    health_code: str = "A",
) -> str:
    """Save the KYC gym profile to Supabase users_gym_profile table.

    Maps collected_data fields to the correct Supabase columns with enum values.

    Args:
        phone: Full phone number (used as whatsapp_id)
        display_name: User's display name
        collected_data: JSON string of collected KYC data
        health_code: Health classification code (A-E)

    Returns:
        JSON with saved profile data
    """
    data = json.loads(collected_data) if isinstance(collected_data, str) else collected_data
    whatsapp_id = int(phone) if phone.isdigit() else 0

    profile = {
        "whatsapp_id": whatsapp_id,
        "full_name": display_name,
        "submission_date": datetime.now(timezone.utc).isoformat(),
        "primary_goal": data.get("primary_goal", ""),
        "training_experience": data.get("training_experience", ""),
        "days_available": int(data.get("days_available", 3)),
        "preferred_schedule": data.get("preferred_schedule", "Mañana"),
        "training_environment": data.get("training_environment", "GYM"),
        "home_equipment": data.get("home_equipment"),
        "biological_sex": data.get("biological_sex", "M"),
        "age": int(data.get("age", 25)),
        "height_cm": float(data.get("height_cm", 170)),
        "weight_kg": float(data.get("weight_kg", 70)),
        "health_status": health_code,
        "email": f"{phone}@gymbot.local",
    }

    result = await supabase_insert(table="users_gym_profile", data=profile, upsert=True)
    return json.dumps({"success": True, "profile": result}, ensure_ascii=False)
