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


def _normalize_enum(raw: str, valid: set[str], aliases: dict[str, str], default: str) -> str:
    """Normalize a raw LLM-extracted value to a valid Supabase enum.

    Tries: exact match → alias lookup → case-insensitive match → default.
    """
    if raw in valid:
        return raw
    if raw in aliases:
        return aliases[raw]
    # Case-insensitive + accent-insensitive match
    raw_lower = raw.lower().strip()
    for v in valid:
        if v.lower() == raw_lower:
            return v
    # Partial match (raw contained in a valid value)
    for v in valid:
        if raw_lower in v.lower():
            return v
    return default


# ── Enum definitions (must match Supabase exactly, including accents) ──

_VALID_GOALS = {
    "Ganar masa muscular", "Bajar grasa", "Mejorar fuerza",
    "Mejorar resistencia", "Salud general / recomposición corporal",
}
_GOAL_ALIASES = {
    "Mantener masa muscular": "Ganar masa muscular",
    "Mantener masa": "Ganar masa muscular",
    "Tonificar": "Salud general / recomposición corporal",
    "Recomposición corporal": "Salud general / recomposición corporal",
    "Recomposicion corporal": "Salud general / recomposición corporal",
    "Perder peso": "Bajar grasa",
    "Definir": "Bajar grasa",
    "Masa muscular": "Ganar masa muscular",
    "Fuerza": "Mejorar fuerza",
    "Resistencia": "Mejorar resistencia",
    "Salud general": "Salud general / recomposición corporal",
}

_VALID_EXPERIENCE = {
    "Nunca he entrenado", "Menos de 6 meses", "6 a 12 meses",
    "1 a 3 años", "Más de 3 años",
}
_EXPERIENCE_ALIASES = {
    "1 a 3 anos": "1 a 3 años",
    "Mas de 3 anos": "Más de 3 años",
    "Mas de 3 años": "Más de 3 años",
    "Más de 3 anos": "Más de 3 años",
    "2 años": "1 a 3 años",
    "3 años": "1 a 3 años",
    "4 años": "Más de 3 años",
    "5 años": "Más de 3 años",
    "1 año": "6 a 12 meses",
    "Sin experiencia": "Nunca he entrenado",
    "Principiante": "Menos de 6 meses",
}

_VALID_FREQUENCY = {
    "No entreno", "1-2 días por semana", "3-4 días por semana", "5-6 días por semana",
}
_FREQUENCY_ALIASES = {
    "1-2 dias por semana": "1-2 días por semana",
    "3-4 dias por semana": "3-4 días por semana",
    "5-6 dias por semana": "5-6 días por semana",
    "4 días": "3-4 días por semana",
    "4 dias": "3-4 días por semana",
    "3 días": "3-4 días por semana",
    "5 días": "5-6 días por semana",
    "6 días": "5-6 días por semana",
    "2 días": "1-2 días por semana",
    "1 día": "1-2 días por semana",
}

_VALID_SCHEDULE = {"Mañana", "Tarde", "Noche"}
_SCHEDULE_ALIASES = {
    "mañana": "Mañana", "Manana": "Mañana", "manana": "Mañana",
    "tarde": "Tarde", "noche": "Noche",
    "AM": "Mañana", "PM": "Tarde",
}

_VALID_STYLE = {"Pesas libres", "Máquinas", "Funcional", "Mixto"}
_STYLE_ALIASES = {
    "Maquinas": "Máquinas", "maquinas": "Máquinas",
    "Pesas": "Pesas libres", "pesas libres": "Pesas libres",
    "mixto": "Mixto", "funcional": "Funcional",
    "Pesas libres y máquinas": "Mixto",
    "Pesas libres y maquinas": "Mixto",
    "Pesas y máquinas": "Mixto",
}

_VALID_SEX = {"M", "F"}
_SEX_ALIASES = {
    "Masculino": "M", "masculino": "M", "Hombre": "M", "hombre": "M", "m": "M",
    "Femenino": "F", "femenino": "F", "Mujer": "F", "mujer": "F", "f": "F",
}

_VALID_CARDIO = {"No", "Caminata", "Bicicleta", "Running"}
_CARDIO_ALIASES = {
    "no": "No", "Ninguno": "No", "ninguno": "No", "No hago": "No",
    "caminata": "Caminata", "Caminar": "Caminata",
    "bicicleta": "Bicicleta", "Bici": "Bicicleta", "bici": "Bicicleta", "Ciclismo": "Bicicleta",
    "running": "Running", "Correr": "Running", "correr": "Running", "Trotar": "Running",
}

_VALID_CARDIO_FREQ = {"0", "1-2", "3-4", "5 o más"}
_CARDIO_FREQ_ALIASES = {
    "0 veces": "0", "ninguna": "0", "No": "0",
    "1": "1-2", "2": "1-2", "1-2 veces": "1-2", "2 veces": "1-2",
    "3": "3-4", "4": "3-4", "3-4 veces": "3-4",
    "5": "5 o más", "5 o mas": "5 o más", "6": "5 o más", "7": "5 o más",
}

_VALID_ENVIRONMENT = {"GYM", "HOME"}
_ENVIRONMENT_ALIASES = {
    "gym": "GYM", "Gym": "GYM", "Gimnasio": "GYM", "gimnasio": "GYM",
    "home": "HOME", "Home": "HOME", "Casa": "HOME", "casa": "HOME",
}


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

    # Normalize all enum fields
    goal = _normalize_enum(
        data.get("primary_goal", ""), _VALID_GOALS, _GOAL_ALIASES,
        "Salud general / recomposición corporal",
    )
    experience = _normalize_enum(
        data.get("training_experience", ""), _VALID_EXPERIENCE, _EXPERIENCE_ALIASES,
        "Menos de 6 meses",
    )
    schedule = _normalize_enum(
        data.get("preferred_schedule", ""), _VALID_SCHEDULE, _SCHEDULE_ALIASES,
        "Mañana",
    )
    style = _normalize_enum(
        data.get("training_style", ""), _VALID_STYLE, _STYLE_ALIASES,
        "Mixto",
    )
    sex = _normalize_enum(
        data.get("biological_sex", ""), _VALID_SEX, _SEX_ALIASES, "M",
    )
    cardio = _normalize_enum(
        data.get("cardio_type", ""), _VALID_CARDIO, _CARDIO_ALIASES, "No",
    )
    cardio_freq = _normalize_enum(
        str(data.get("cardio_frequency", "0")), _VALID_CARDIO_FREQ, _CARDIO_FREQ_ALIASES, "0",
    )
    environment = _normalize_enum(
        data.get("training_environment", ""), _VALID_ENVIRONMENT, _ENVIRONMENT_ALIASES,
        "GYM",
    )
    frequency = data.get("current_frequency", "")
    if not frequency or frequency not in _VALID_FREQUENCY:
        # Derive from days_available or experience
        days = int(data.get("days_available", 3))
        if days <= 0:
            frequency = "No entreno"
        elif days <= 2:
            frequency = "1-2 días por semana"
        elif days <= 4:
            frequency = "3-4 días por semana"
        else:
            frequency = "5-6 días por semana"

    # Map experience to fitness_level default
    level_map = {
        "Nunca he entrenado": "Principiante",
        "Menos de 6 meses": "Principiante",
        "6 a 12 meses": "Intermedio",
        "1 a 3 años": "Intermedio",
        "Más de 3 años": "Avanzado",
    }

    profile = {
        "whatsapp_id": whatsapp_id,
        "full_name": display_name,
        "submission_date": datetime.now(timezone.utc).isoformat(),
        "email": f"{phone}@gymbot.local",
        # KYC collected fields (all normalized)
        "primary_goal": goal,
        "training_experience": experience,
        "days_available": int(data.get("days_available", 3)),
        "preferred_schedule": schedule,
        "training_environment": environment,
        "home_equipment": data.get("home_equipment"),
        "biological_sex": sex,
        "age": int(data.get("age", 25)),
        "height_cm": int(float(data.get("height_cm", 170))),
        "weight_kg": float(data.get("weight_kg", 70)),
        "health_status": health_code,
        # Defaults for NOT NULL fields not collected in KYC
        "secondary_goal": data.get("secondary_goal", ""),
        "current_frequency": frequency,
        "fitness_level": data.get("fitness_level", level_map.get(experience, "Intermedio")),
        "session_duration_mins": data.get("session_duration_mins", "60-75 minutos"),
        "training_style": style,
        "priority_muscles": data.get("priority_muscles", ""),
        "disliked_exercises": data.get("disliked_exercises", ""),
        "cardio_type": cardio,
        "cardio_frequency": cardio_freq,
    }

    result = await supabase_insert(table="users_gym_profile", data=profile, upsert=True)
    return json.dumps({"success": True, "profile": result}, ensure_ascii=False)
