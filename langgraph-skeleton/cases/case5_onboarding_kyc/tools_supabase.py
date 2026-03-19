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

    # Normalize primary_goal to valid enum values (FK to user_goals table)
    valid_goals = {
        "Ganar masa muscular", "Bajar grasa", "Mejorar fuerza",
        "Mejorar resistencia", "Salud general / recomposición corporal",
    }
    goal_normalize = {
        "Mantener masa muscular": "Ganar masa muscular",
        "Mantener masa": "Ganar masa muscular",
        "Tonificar": "Salud general / recomposición corporal",
        "Recomposición corporal": "Salud general / recomposición corporal",
        "Perder peso": "Bajar grasa",
        "Definir": "Bajar grasa",
    }
    raw_goal = data.get("primary_goal", "Salud general / recomposición corporal")
    if raw_goal not in valid_goals:
        data["primary_goal"] = goal_normalize.get(raw_goal, "Salud general / recomposición corporal")

    # Map experience to frequency and fitness_level defaults
    exp = data.get("training_experience", "")
    freq_map = {
        "Nunca he entrenado": "No entreno",
        "Menos de 6 meses": "1-2 días por semana",
        "6 a 12 meses": "3-4 días por semana",
        "1 a 3 años": "3-4 días por semana",
        "Más de 3 años": "5-6 días por semana",
    }
    level_map = {
        "Nunca he entrenado": "Principiante",
        "Menos de 6 meses": "Principiante",
        "6 a 12 meses": "Intermedio",
        "1 a 3 años": "Intermedio",
        "Más de 3 años": "Avanzado",
    }
    style_map = {
        "GYM": "Mixto",
        "HOME": "Funcional",
    }

    profile = {
        "whatsapp_id": whatsapp_id,
        "full_name": display_name,
        "submission_date": datetime.now(timezone.utc).isoformat(),
        "email": f"{phone}@gymbot.local",
        # KYC collected fields
        "primary_goal": data.get("primary_goal", "Salud general / recomposición corporal"),
        "training_experience": data.get("training_experience", "Menos de 6 meses"),
        "days_available": int(data.get("days_available", 3)),
        "preferred_schedule": data.get("preferred_schedule", "Mañana"),
        "training_environment": data.get("training_environment", "GYM"),
        "home_equipment": data.get("home_equipment"),
        "biological_sex": data.get("biological_sex", "M"),
        "age": int(data.get("age", 25)),
        "height_cm": int(float(data.get("height_cm", 170))),
        "weight_kg": float(data.get("weight_kg", 70)),
        "health_status": health_code,
        # Defaults for NOT NULL fields not collected in KYC
        "secondary_goal": data.get("secondary_goal", ""),
        "current_frequency": data.get("current_frequency", freq_map.get(exp, "3-4 días por semana")),
        "fitness_level": data.get("fitness_level", level_map.get(exp, "Intermedio")),
        "session_duration_mins": data.get("session_duration_mins", "60-75 minutos"),
        "training_style": data.get("training_style", style_map.get(data.get("training_environment", "GYM"), "Mixto")),
        "priority_muscles": data.get("priority_muscles", ""),
        "disliked_exercises": data.get("disliked_exercises", ""),
        "cardio_type": data.get("cardio_type", "No"),
        "cardio_frequency": data.get("cardio_frequency", "0"),
    }

    result = await supabase_insert(table="users_gym_profile", data=profile, upsert=True)
    return json.dumps({"success": True, "profile": result}, ensure_ascii=False)
