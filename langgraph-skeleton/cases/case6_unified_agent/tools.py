"""Tools for the Unified Agent Kairos (Case 6).

All tools use Supabase PostgREST via src/shared/supabase_client.
Tools are grouped by phase:
- US1: get_todays_routine, confirm_workout_completion, decline_workout, create_magic_link
- US3: get_day_requirements, get_exercises_for_draft, find_exercise_alternatives, save_workout_plan
- US4: get_schedule_info, schedule_sessions
- US6: get_mesocycle_status
"""

import json
import logging
import os
import uuid
from datetime import datetime, timedelta, timezone

logger = logging.getLogger(__name__)

from langchain_core.tools import tool

from src.shared.supabase_client import (
    supabase_query,
    supabase_insert,
    supabase_update,
    supabase_bulk_insert,
    supabase_delete,
)

from cases.case6_unified_agent.draft_schema import normalize_draft_data


BOGOTA_UTC_OFFSET = timedelta(hours=-5)

# Mapping from user-facing equipment terms (Spanish free text) to DB exercise.equipment values.
# The DB has mixed values: "dumbbell", "barbell", "machine", "cable", "bodyweight",
# "resistance_band", "kettlebell", "smith", "pull_bar", plus some Spanish ("Mancuerna", "Barra", etc.)
_EQUIPMENT_NORMALIZE: dict[str, set[str]] = {
    "mancuernas": {"dumbbell", "mancuerna"},
    "mancuerna": {"dumbbell", "mancuerna"},
    "pesas": {"dumbbell", "mancuerna", "barbell", "barra"},
    "barra": {"barbell", "barra"},
    "bandas": {"resistance_band", "bands"},
    "banda": {"resistance_band", "bands"},
    "bandas de resistencia": {"resistance_band", "bands"},
    "máquinas": {"machine", "máquina", "cable", "polea"},
    "maquinas": {"machine", "máquina", "cable", "polea"},
    "kettlebell": {"kettlebell"},
    "pesa rusa": {"kettlebell"},
    "peso corporal": {"bodyweight", "peso corporal"},
    "cuerpo": {"bodyweight", "peso corporal"},
    "smith": {"smith"},
    "barra guiada": {"smith"},
    "polea": {"cable", "polea"},
    "cables": {"cable", "polea"},
    # B-S02-001 part 3: pull-up bar unlocks pull_v/pull_h coverage for HOME users.
    # Only matches when explicitly mentioned (NOT included by default in "peso corporal"
    # since the typical bodyweight-only user doesn't have one).
    "barra de dominadas": {"bodyweight", "peso corporal", "pull_bar"},
    "barra para dominadas": {"bodyweight", "peso corporal", "pull_bar"},
    "dominadas": {"bodyweight", "peso corporal", "pull_bar"},
    "pull up": {"bodyweight", "peso corporal", "pull_bar"},
    "pull-up": {"bodyweight", "peso corporal", "pull_bar"},
}


def _normalize_equipment(raw_equipment: str) -> set[str]:
    """Convert user-facing equipment text to set of DB equipment values.

    Handles free text like "unas mancuernas, bandas" → {"dumbbell", "mancuerna", "resistance_band", "bands"}.
    Always includes bodyweight.
    """
    raw_lower = raw_equipment.lower()
    db_values: set[str] = {"bodyweight", "peso corporal"}  # always allowed for HOME

    for keyword, mapped in _EQUIPMENT_NORMALIZE.items():
        if keyword in raw_lower:
            db_values.update(mapped)

    return db_values


def _today_bogota() -> str:
    now_utc = datetime.now(timezone.utc)
    return (now_utc + BOGOTA_UTC_OFFSET).strftime("%Y-%m-%d")


def _yesterday_bogota() -> str:
    now_utc = datetime.now(timezone.utc)
    return (now_utc + BOGOTA_UTC_OFFSET - timedelta(days=1)).strftime("%Y-%m-%d")


# ═══════════════ Health Filtering (QF-5) ═══════════════

_HEALTH_B_KEYWORDS = [
    "salto", "jump", "sprint", "zancada", "pistol", "sissy",
    "sentadilla búlgara",
]

_HEALTH_C_KEYWORDS = [
    "press militar", "overhead", "arnold", "push press", "jerk",
    "snatch", "clean and press", "remo al mentón", "remo al menton",
    "elevacion lateral", "elevación lateral", "upright row",
    "por encima de la cabeza", "detrás del cuello", "behind",
]

_HEALTH_D_KEYWORDS = [
    "peso muerto", "deadlift", "good morning", "buenos días",
    "back squat", "sentadilla trasera", "front squat",
    "sentadilla frontal", "remo con barra", "barbell row",
    "snatch", "clean", "jerk", "hiperextensión", "hyperextension",
    "rack pull", "press militar", "overhead", "crunch", "crunches",
    "abdominal", "sit up", "sit-up",
]


def _name_matches_any(name: str, keywords: list[str]) -> bool:
    low = name.lower()
    return any(kw in low for kw in keywords)


def _apply_health_filter(
    exercises: list[dict], health_status: str | None,
) -> list[dict]:
    """Filter exercises based on user health restriction code (A-E).

    Returns list unchanged for None or 'A'. Pure function, no DB calls.
    """
    if not health_status or health_status.upper() == "A":
        return exercises

    hs = health_status.upper()
    if hs == "B":
        return [
            ex for ex in exercises
            if not (
                ex.get("pattern") == "lunge"
                or (ex.get("pattern") == "squat" and ex.get("equipment") == "barbell")
                or _name_matches_any(ex.get("spanish_name", ""), _HEALTH_B_KEYWORDS)
            )
        ]
    if hs == "C":
        return [
            ex for ex in exercises
            if not (
                ex.get("pattern") == "push_v"
                or _name_matches_any(ex.get("spanish_name", ""), _HEALTH_C_KEYWORDS)
                or "upright_row" in ex.get("exercise_id", "")
            )
        ]
    if hs == "D":
        return [
            ex for ex in exercises
            if not (
                ex.get("pattern") == "push_v"  # overhead = axial spinal load
                or (ex.get("pattern") == "hinge" and ex.get("equipment") == "barbell")
                or (ex.get("pattern") == "squat" and ex.get("equipment") == "barbell")
                or (ex.get("main_muscle") == "Lower back" and ex.get("role") == "compound")
                or _name_matches_any(ex.get("spanish_name", ""), _HEALTH_D_KEYWORDS)
            )
        ]
    if hs == "E":
        return [
            ex for ex in exercises
            if ex.get("level") != "Avanzado"
            and ex.get("role") != "cardio"
            and (
                ex.get("equipment") in ("machine", "cable")
                or (ex.get("equipment") == "bodyweight" and ex.get("pattern") in ("core", "accessory"))
            )
        ]
    return exercises


# ═══════════════ US1: Daily Operations ═══════════════


@tool
async def get_todays_routine(user_id: str, session_name: str, week: int) -> str:
    """Obtiene la rutina completa de una sesión específica con ejercicios, series, reps, RIR y descanso.

    Args:
        user_id: UUID del usuario
        session_name: Nombre de la sesión (ej: "Upper Body A", "Full Body A")
        week: Número de semana (1-4)

    Returns:
        Lista formateada de ejercicios con parámetros de carga
    """
    rows = await supabase_query(
        "workouts",
        select="exercise_id,sets,reps,rir,rest-seconds,tempo,exercise_order,notes",
        filters={
            "user_id": f"eq.{user_id}",
            "week": f"eq.{week}",
            "day_name": f"eq.{session_name}",
            "order": "exercise_order.asc",
        },
    )

    if not rows:
        return json.dumps({"error": f"No se encontraron ejercicios para {session_name} semana {week}"})

    # Defensive sort in Python (PostgREST order param should handle this, but just in case)
    rows = sorted(rows, key=lambda r: r.get("exercise_order", 99))

    # Get exercise details
    exercise_ids = [r["exercise_id"] for r in rows]
    exercises = await supabase_query(
        "exercises",
        select="exercise_id,spanish_name,main_muscle,link",
        filters={"exercise_id": f"in.({','.join(exercise_ids)})"},
    )
    ex_map = {e["exercise_id"]: e for e in exercises}

    result = []
    for r in rows:
        ex = ex_map.get(r["exercise_id"], {})
        result.append({
            "order": r.get("exercise_order", 0),
            "exercise": ex.get("spanish_name", r["exercise_id"]),
            "muscle": ex.get("main_muscle", ""),
            "sets": r.get("sets", ""),
            "reps": r.get("reps", ""),
            "rir": r.get("rir", ""),
            "rest_seconds": r.get("rest-seconds", 0),
            "tempo": r.get("tempo", ""),
            "video": ex.get("link", ""),
        })

    return json.dumps({
        "session_name": session_name,
        "week": week,
        "exercises": result,
    }, ensure_ascii=False)


def _bogota_date_to_utc_range(bogota_date: str) -> tuple[str, str]:
    """Convert a YYYY-MM-DD Bogota date to UTC start/end timestamps.

    Bogota is UTC-5, so midnight Bogota = 05:00 UTC.
    Returns (start_utc_iso, end_utc_iso) for the full day.
    """
    d = datetime.strptime(bogota_date, "%Y-%m-%d")
    utc_start = d - BOGOTA_UTC_OFFSET  # midnight Bogota → UTC
    utc_end = utc_start + timedelta(days=1)
    return utc_start.strftime("%Y-%m-%dT%H:%M:%S+00:00"), utc_end.strftime("%Y-%m-%dT%H:%M:%S+00:00")


@tool
async def confirm_workout_completion(user_id: str, session_date: str | None = None) -> str:
    """Marca una sesión como completada. Soporta grace period (hoy o ayer).

    Uses planned_day_utc (timestamptz) for reliable matching regardless of
    planned_day text format (DD/MM from Kairos vs YYYY-MM-DD from fixtures).

    Args:
        user_id: UUID del usuario
        session_date: Fecha opcional en formato YYYY-MM-DD. Si no se provee, intenta hoy y luego ayer.

    Returns:
        Confirmación de la sesión marcada
    """
    today = _today_bogota()
    yesterday = _yesterday_bogota()

    dates_to_try = [session_date] if session_date else [today, yesterday]

    for date in dates_to_try:
        utc_start, utc_end = _bogota_date_to_utc_range(date)

        # Fetch uncompleted sessions using planned_day_utc range
        sessions = await supabase_query(
            "user_weekly_schedule",
            select="day_routine_id,session_name,week,planned_day_utc",
            filters={
                "user_id": f"eq.{user_id}",
                "planned_day_utc": f"gte.{utc_start}",
                "Completed": "eq.false",
            },
            limit=10,
        )

        # Filter in Python: only sessions within the target day
        matched = [s for s in sessions if s.get("planned_day_utc", "") < utc_end]

        if matched:
            session = matched[0]
            # Mark as completed
            await supabase_update(
                "user_weekly_schedule",
                data={"Completed": True},
                filters={"day_routine_id": f"eq.{session['day_routine_id']}"},
            )

            # Also resolve any pending CONFIRMAR_RUTINA task
            await supabase_update(
                "pending_tasks",
                data={"status": "completed", "resolved_at": datetime.now(timezone.utc).isoformat()},
                filters={
                    "user_id": f"eq.{user_id}",
                    "task_type": "eq.CONFIRMAR_RUTINA",
                    "status": "eq.pending",
                },
            )

            return json.dumps({
                "success": True,
                "session_name": session["session_name"],
                "week": session["week"],
                "date": date,
            })

    return json.dumps({
        "success": False,
        "error": "No se encontró sesión sin completar para hoy ni ayer",
    })


@tool
async def decline_workout(user_id: str) -> str:
    """Marca la tarea pendiente de confirmación como declinada (usuario no pudo entrenar).

    Args:
        user_id: UUID del usuario

    Returns:
        Confirmación de la tarea declinada
    """
    result = await supabase_update(
        "pending_tasks",
        data={"status": "declined", "resolved_at": datetime.now(timezone.utc).isoformat()},
        filters={
            "user_id": f"eq.{user_id}",
            "status": "eq.pending",
        },
    )

    if result:
        return json.dumps({"success": True, "tasks_declined": len(result)})
    return json.dumps({"success": False, "error": "No se encontraron tareas pendientes"})


@tool
async def create_magic_link(user_id: str) -> str:
    """Genera un enlace de acceso al Workout Tracker web con expiración de 48 horas.

    Args:
        user_id: UUID del usuario

    Returns:
        URL completa del Workout Tracker
    """
    print(f"[DEBUG create_magic_link] user_id received: '{user_id}'")
    # Generate short hex code
    now = datetime.now(timezone.utc)
    code = f"{int(now.timestamp()) % 1000000:06x}"

    expires_at = (now + timedelta(hours=48)).isoformat()

    await supabase_insert(
        "magic_links",
        data={
            "code": code,
            "user_id": user_id,
            "created_at": now.isoformat(),
            "expires_at": expires_at,
        },
    )

    frontend_url = os.getenv("FRONTEND_URL", "https://kairos-tracker.web.app")
    url = f"{frontend_url}/w?c={code}"

    return json.dumps({
        "success": True,
        "url": url,
        "expires_at": expires_at,
        "code": code,
    })


# ═══════════════ US3: Draft Routine Creation ═══════════════


@tool
async def get_day_requirements(week_schedule: str) -> str:
    """Obtiene los días y patrones de movimiento requeridos para un schedule template.

    Args:
        week_schedule: Tipo de schedule (fb_2, fb_3, ul_4, ppl_5, ppl_6)

    Returns:
        Lista de días con sus patrones requeridos, min_sets y prioridad
    """
    # Get template
    templates = await supabase_query(
        "routine_templates",
        select="template_id",
        filters={"week_schedule": f"eq.{week_schedule}"},
        limit=1,
    )

    if not templates:
        return json.dumps({"error": f"No se encontró template para schedule {week_schedule}"})

    # Get template days
    days = await supabase_query(
        "template_days",
        select="template_day_id,day_number,title",
        filters={"week_schedule": f"eq.{week_schedule}"},
    )

    if not days:
        return json.dumps({"error": f"No se encontraron días para schedule {week_schedule}"})

    # Get requirements for each day
    result = []
    for day in sorted(days, key=lambda d: d["day_number"]):
        reqs = await supabase_query(
            "day_requirements",
            select="pattern,min_sets,priority",
            filters={"template_day_id": f"eq.{day['template_day_id']}"},
        )

        result.append({
            "day_number": day["day_number"],
            "title": day["title"],
            "patterns": sorted(reqs, key=lambda r: r.get("priority", 99)),
        })

    return json.dumps(result, ensure_ascii=False)


async def _lookup_user_profile(user_id: str | None) -> dict:
    """Look up health_status, training_environment, home_equipment from DB.

    Returns dict with keys: health_status, training_environment, home_equipment.
    whatsapp_id in users_gym_profile is bigint (phone number), so we must
    resolve user_id → full_phone_number first, then query by phone.
    """
    default = {"health_status": "A", "training_environment": "GYM", "home_equipment": ""}
    if not user_id:
        return default

    # Resolve user_id (UUID) → phone number first
    phone = None
    try:
        users = await supabase_query(
            "users", select="full_phone_number",
            filters={"user_id": f"eq.{user_id}"}, limit=1,
        )
        if users:
            phone = users[0].get("full_phone_number", "")
    except Exception:
        pass

    profiles = []
    if phone:
        try:
            profiles = await supabase_query(
                "users_gym_profile",
                select="health_status,training_environment,home_equipment",
                filters={"whatsapp_id": f"eq.{phone}"},
                limit=1,
            )
        except Exception:
            pass
    if profiles:
        p = profiles[0]
        return {
            "health_status": p.get("health_status", "A"),
            "training_environment": p.get("training_environment", "GYM"),
            "home_equipment": p.get("home_equipment", ""),
        }
    return default


async def _user_allowed_equipment(user_id: str | None) -> set[str] | None:
    """Return the set of allowed equipment values for a user, or None if no restriction.

    Returns None for GYM users (or when user_id is missing / profile lookup fails) —
    they can use any equipment. Returns a normalized set of allowed equipment values
    for HOME users based on their declared home_equipment.

    Single source of truth used by:
    - get_exercises_for_draft (filter exercise candidates)
    - find_exercise_alternatives (filter alternatives during swaps)
    - save_workout_plan (final validation before bulk insert)
    """
    if not user_id:
        return None
    profile = await _lookup_user_profile(user_id)
    if not profile or profile.get("training_environment") != "HOME":
        return None  # GYM users: no restriction
    raw = profile.get("home_equipment") or "peso corporal"
    return _normalize_equipment(raw)


@tool
async def get_exercises_for_draft(
    pattern: str,
    level: str,
    user_id: str | None = None,
    equipment: str | None = None,
    exclude_muscle: str | None = None,
    health_status: str | None = None,
    limit: int = 5,
) -> str:
    """Busca ejercicios candidatos por patrón de movimiento para construir el borrador de rutina.

    IMPORTANTE: Siempre pasa user_id para aplicar filtros de salud y equipamiento automáticamente.
    Sin user_id, usuarios con restricciones pueden recibir ejercicios peligrosos.

    Args:
        pattern: Patrón de movimiento (push_h, push_v, pull_h, pull_v, squat, hinge, lunge, core, arm, accessory)
        level: Nivel del usuario (Principiante, Intermedio, Avanzado)
        user_id: UUID del usuario — OBLIGATORIO para filtrar por salud y equipamiento
        equipment: Equipamiento disponible (opcional — se detecta automáticamente si se pasa user_id)
        exclude_muscle: Músculo a excluir (opcional, ej: "Calfs" para ejercicios no deseados)
        health_status: Código de salud (opcional — se detecta automáticamente si se pasa user_id)
        limit: Máximo de ejercicios a retornar (default 5)

    Returns:
        Lista de ejercicios candidatos con detalles
    """
    filters = {
        "pattern": f"eq.{pattern}",
    }

    if exclude_muscle:
        filters["main_muscle"] = f"neq.{exclude_muscle}"

    rows = await supabase_query(
        "exercises",
        select="exercise_id,spanish_name,pattern,role,main_muscle,secondary_muscles,level,equipment,link",
        filters=filters,
        limit=limit * 3,  # fetch more to filter by level
    )

    # Filter by level compatibility
    level_order = {"Principiante": 1, "Intermedio": 2, "Avanzado": 3}
    user_level = level_order.get(level, 2)

    filtered = [
        r for r in rows
        if level_order.get(r.get("level", "Intermedio"), 2) <= user_level
    ]

    # Auto-lookup profile if user_id provided (for equipment + health)
    profile = await _lookup_user_profile(user_id) if user_id else None

    # Equipment filter (B-S02-001 fix): hard filter — never fall back to unfiltered.
    # Prefer explicit equipment arg, then derive from HOME profile, then None (no restriction).
    if equipment:
        allowed_equip = _normalize_equipment(equipment)
    else:
        allowed_equip = await _user_allowed_equipment(user_id)

    if allowed_equip is not None:
        before = len(filtered)
        filtered = [
            r for r in filtered
            if r.get("equipment", "").lower() in allowed_equip
        ]
        if not filtered:
            logger.warning(
                f"[EQUIPMENT] No exercises for pattern={pattern} match equipment={sorted(allowed_equip)}. "
                f"({before} candidates dropped). Pattern will be omitted from draft."
            )

    # Apply health restrictions (auto-lookup from profile if not explicitly provided)
    if health_status and health_status != "A":
        hs = health_status
    elif profile:
        hs = profile.get("health_status", "A")
    else:
        hs = "A"
        if not user_id:
            logger.warning("[HEALTH] get_exercises_for_draft called without user_id — health filter skipped")
    filtered = _apply_health_filter(filtered, hs)

    return json.dumps(filtered[:limit], ensure_ascii=False)


@tool
async def find_exercise_alternatives(
    pattern: str,
    level: str,
    user_id: str | None = None,
    exclude_name: str | None = None,
    equipment: str | None = None,
    health_status: str | None = None,
) -> str:
    """Busca alternativas de ejercicios para swaps en el borrador de rutina.

    Args:
        pattern: Patrón de movimiento del ejercicio a reemplazar
        level: Nivel del usuario
        user_id: UUID del usuario (para buscar restricciones de salud automáticamente)
        exclude_name: Nombre del ejercicio a excluir (el que se quiere cambiar)
        equipment: Equipamiento disponible (opcional)
        health_status: Código de salud del usuario (A, B, C, D, E). Si no se pasa, se busca automáticamente

    Returns:
        Lista de ejercicios alternativos
    """
    filters = {"pattern": f"eq.{pattern}"}

    rows = await supabase_query(
        "exercises",
        select="exercise_id,spanish_name,pattern,role,main_muscle,level,equipment,link",
        filters=filters,
        limit=20,
    )

    # Filter by level
    level_order = {"Principiante": 1, "Intermedio": 2, "Avanzado": 3}
    user_level = level_order.get(level, 2)

    filtered = [
        r for r in rows
        if level_order.get(r.get("level", "Intermedio"), 2) <= user_level
    ]

    # Exclude the current exercise
    if exclude_name:
        exclude_lower = exclude_name.lower()
        filtered = [r for r in filtered if r.get("spanish_name", "").lower() != exclude_lower]

    # Auto-lookup profile if user_id provided
    profile = await _lookup_user_profile(user_id) if user_id else None

    # Equipment filter (B-S02-001 fix): hard filter, no fallback to unfiltered.
    if equipment:
        allowed_equip = _normalize_equipment(equipment)
    else:
        allowed_equip = await _user_allowed_equipment(user_id)

    if allowed_equip is not None:
        before = len(filtered)
        filtered = [r for r in filtered if r.get("equipment", "").lower() in allowed_equip]
        if not filtered:
            logger.warning(
                f"[EQUIPMENT] No alternatives for pattern={pattern} match equipment={sorted(allowed_equip)}. "
                f"({before} candidates dropped)."
            )

    # Apply health restrictions (auto-lookup from profile if not explicitly provided)
    if health_status and health_status != "A":
        hs = health_status
    elif profile:
        hs = profile.get("health_status", "A")
    else:
        hs = "A"
    filtered = _apply_health_filter(filtered, hs)

    return json.dumps(filtered[:5], ensure_ascii=False)


# ═══════════════ Exercise ID Resolution Helpers ═══════════════


def _extract_exercise_identifiers(ex: dict) -> tuple[str | None, str | None]:
    """Extract (candidate_id, candidate_name) from an exercise dict.

    Returns a real exercise_id if it starts with 'ex_', otherwise
    treats it as a name that needs resolution.
    """
    raw_id = ex.get("exercise_id") or ex.get("id")
    if raw_id and raw_id.startswith("ex_"):
        return (raw_id, None)

    candidate_name = (
        raw_id
        or ex.get("name")
        or ex.get("exercise")
        or ex.get("spanish_name")
        or ex.get("nombre")
    )
    return (None, candidate_name)


def _match_names_to_exercises(
    names: list[str],
    candidates: list[dict],
) -> dict[str, str]:
    """Match each name to the best candidate exercise_id.

    Priority: exact match > case-insensitive exact > word overlap.
    """
    result = {}
    for name in names:
        name_lower = name.lower().strip()
        best_id = None
        best_score = 0

        exact_match = False
        for c in candidates:
            c_name = c.get("spanish_name", "").lower().strip()

            if c_name == name_lower:
                best_id = c["exercise_id"]
                exact_match = True
                break

            name_words = set(name_lower.split())
            c_words = set(c_name.split())
            overlap = len(name_words & c_words)

            if c_name.startswith(name_lower):
                overlap += 2

            if overlap > best_score:
                best_score = overlap
                best_id = c["exercise_id"]

        if exact_match:
            result[name] = best_id
        else:
            min_threshold = 1 if len(name_lower.split()) == 1 else 2
            if best_id and best_score >= min_threshold:
                result[name] = best_id

    return result


async def _resolve_exercise_ids(draft: dict) -> tuple[dict[tuple[int, int], str], list[dict]]:
    """Resolve all exercises in the draft to valid exercise_ids.

    Returns:
        resolved: {(day_idx, ex_idx): "ex_real_id", ...}
        unresolved: [{"day_idx": 0, "ex_idx": 2, "name": "..."}, ...]
    """
    candidate_ids: set[str] = set()
    needs_name_resolution: list[tuple[int, int, str]] = []

    for d_idx, day in enumerate(draft.get("days", [])):
        for e_idx, ex in enumerate(day.get("exercises", [])):
            cid, cname = _extract_exercise_identifiers(ex)
            if cid:
                candidate_ids.add(cid)
            elif cname:
                needs_name_resolution.append((d_idx, e_idx, cname))

    resolved: dict[tuple[int, int], str] = {}

    # Batch-validate candidate IDs
    valid_ids: set[str] = set()
    if candidate_ids:
        rows = await supabase_query(
            "exercises",
            select="exercise_id",
            filters={"exercise_id": f"in.({','.join(candidate_ids)})"},
        )
        valid_ids = {r["exercise_id"] for r in rows}

    # Map valid IDs, move invalid to name resolution
    for d_idx, day in enumerate(draft.get("days", [])):
        for e_idx, ex in enumerate(day.get("exercises", [])):
            cid, cname = _extract_exercise_identifiers(ex)
            if cid and cid in valid_ids:
                resolved[(d_idx, e_idx)] = cid
            elif cid and cid not in valid_ids:
                needs_name_resolution.append((d_idx, e_idx, cid))

    # Batch-resolve names via ILIKE
    unresolved: list[dict] = []
    if needs_name_resolution:
        unique_names = list({name for _, _, name in needs_name_resolution})
        or_parts = []
        for name in unique_names:
            safe = name.replace("(", "").replace(")", "").replace(",", "").replace("'", "")
            or_parts.append(f"spanish_name.ilike.*{safe}*")

        all_candidates = await supabase_query(
            "exercises",
            select="exercise_id,spanish_name",
            filters={"or": f"({','.join(or_parts)})"},
        )

        name_to_id = _match_names_to_exercises(unique_names, all_candidates)

        for d_idx, e_idx, name in needs_name_resolution:
            if (d_idx, e_idx) in resolved:
                continue
            matched_id = name_to_id.get(name)
            if matched_id:
                resolved[(d_idx, e_idx)] = matched_id
            else:
                unresolved.append({"day_idx": d_idx, "ex_idx": e_idx, "name": name})

    return resolved, unresolved


# ═══════════════ Loading Parameters (set_profiles) ═══════════════


async def _fetch_loading_params(
    goal: str, level: str,
) -> dict[tuple[int, str], dict]:
    """Fetch sets/reps/rir/rest_sec/tempo from set_profiles indexed by (week, role).

    Centralizes loading prescription so save_workout_plan applies periodized
    parameters from the curated table instead of trusting the LLM.

    Returns:
        Dict keyed by (week:int, role:str) → {sets, reps, rir, rest_sec, tempo}.
        Empty dict if the (goal, level) combination has no rows.
    """
    rows = await supabase_query(
        "set_profiles",
        select="week,role,sets,reps,rir,rest_sec,tempo",
        filters={"goal": f"eq.{goal}", "level": f"eq.{level}"},
    )
    return {(r["week"], r["role"]): r for r in rows}


# Conservative fallback if set_profiles has no row for (goal, level, week, role).
# Coverage check (2026-05-11): 5 goals × 3 levels × 4 weeks × 4 roles = 240 rows
# all populated, so this should rarely fire. Logged when used.
DEFAULT_LOADING_PARAMS = {
    "sets": "3",
    "reps": "8-12",
    "rir": "1-2",
    "rest_sec": 120,
    "tempo": "2-0-1",
}


async def _fetch_exercise_roles(exercise_ids: list[str]) -> dict[str, str]:
    """Fetch role for each exercise_id (compound, isolation, core, cardio).

    Used by save_workout_plan to know which loading profile to apply per
    exercise. Batches a single Postgres query.

    Args:
        exercise_ids: List of resolved exercise IDs (e.g. ["ex_squat", "ex_bench"]).

    Returns:
        Dict {exercise_id: role}. Missing IDs are simply absent from the dict;
        callers should fall back to "compound" or DEFAULT_LOADING_PARAMS.
    """
    if not exercise_ids:
        return {}
    unique_ids = list(set(exercise_ids))
    rows = await supabase_query(
        "exercises",
        select="exercise_id,role",
        filters={"exercise_id": f"in.({','.join(unique_ids)})"},
    )
    return {r["exercise_id"]: r["role"] for r in rows}


# ═══════════════ Save Workout Plan ═══════════════


VALID_GOALS = {
    "Ganar masa muscular", "Bajar grasa", "Mejorar fuerza",
    "Mejorar resistencia", "Salud general / recomposición corporal",
}
GOAL_NORMALIZE = {
    "Mantener masa muscular": "Ganar masa muscular",
    "Mantener masa": "Ganar masa muscular",
    "Tonificar": "Salud general / recomposición corporal",
    "Recomposición corporal": "Salud general / recomposición corporal",
    "Perder peso": "Bajar grasa",
    "Definir": "Bajar grasa",
}


@tool
async def save_workout_plan(user_id: str, draft_json: str) -> str:
    """Guarda el plan de entrenamiento aprobado por el usuario.

    Crea users_plans + bulk insert de workouts para 4 semanas.
    Resuelve automáticamente nombres de ejercicios a exercise_ids válidos.

    Args:
        user_id: UUID del usuario
        draft_json: JSON string con la estructura:
            {week_schedule, goal, level, days: [{day_number, title, exercises: [{exercise_id, sets, reps, rir, rest_seconds, exercise_order, tempo}]}]}

    Returns:
        Confirmación con plan_id y cantidad de workouts creados
    """
    draft = json.loads(draft_json)
    is_renewal = draft.pop("is_renewal", False)

    now = datetime.now(timezone.utc)
    plan_id = str(uuid.uuid4())

    # Determine week_schedule
    days_count = len(draft.get("days", []))
    ws_map = {2: "fb_2", 3: "fb_3", 4: "ul_4", 5: "ppl_5", 6: "ppl_6"}
    week_schedule = draft.get("week_schedule", ws_map.get(days_count, "fb_3"))

    # Normalize goal
    raw_goal = draft.get("goal", "Salud general / recomposición corporal")
    goal = raw_goal if raw_goal in VALID_GOALS else GOAL_NORMALIZE.get(raw_goal, "Salud general / recomposición corporal")

    goal_code_map = {
        "Ganar masa muscular": "hyp", "Mejorar fuerza": "str",
        "Bajar grasa": "cut", "Mejorar resistencia": "end",
        "Salud general / recomposición corporal": "rec",
    }
    level_map = {"Principiante": "beg", "Intermedio": "int", "Avanzado": "adv"}
    template_id = f"tpl_{week_schedule}_{goal_code_map.get(goal, 'hyp')}_{level_map.get(draft.get('level', ''), 'int')}"

    # Resolve exercise IDs (handles names → real IDs)
    resolved, unresolved = await _resolve_exercise_ids(draft)

    # Equipment validation (B-S02-001 fix): fail-fast if any resolved exercise
    # uses equipment the user doesn't have. The LLM may "hallucinate" exercises
    # outside the curated list (e.g. invent "Press de banca con barra" which
    # _resolve_exercise_ids will ILIKE-match to a real barbell exercise even
    # for a bodyweight user). This is the final safety net.
    allowed_equip = await _user_allowed_equipment(user_id)
    if allowed_equip and resolved:
        resolved_ids = sorted(set(resolved.values()))
        eq_rows = await supabase_query(
            "exercises",
            select="exercise_id,equipment,spanish_name",
            filters={"exercise_id": f"in.({','.join(resolved_ids)})"},
        )
        equip_by_id = {r["exercise_id"]: r for r in eq_rows}

        violations = []
        for ex_id in resolved_ids:
            row = equip_by_id.get(ex_id)
            if not row:
                continue
            if row.get("equipment", "").lower() not in allowed_equip:
                violations.append({
                    "exercise_id": ex_id,
                    "name": row.get("spanish_name", ex_id),
                    "equipment": row.get("equipment", ""),
                })

        if violations:
            logger.warning(
                f"[EQUIPMENT] save_workout_plan rejected for user_id={user_id}: "
                f"{len(violations)} exercise(s) violate allowed_equip={sorted(allowed_equip)}"
            )
            return json.dumps({
                "success": False,
                "error": "EQUIPMENT_MISMATCH",
                "violations": violations,
                "user_allowed_equipment": sorted(allowed_equip),
                "hint": (
                    "El usuario solo tiene este equipamiento. Vuelve al Paso 2 y llama "
                    "get_exercises_for_draft con user_id correcto para que el filtro de "
                    "equipamiento se aplique. Elige solo ejercicios devueltos por el tool. "
                    "NO reintentes guardar con los mismos ejercicios."
                ),
            }, ensure_ascii=False)

    # Create plan (skip if renewal — plan already updated by renew_change_days)
    if not is_renewal:
        await supabase_insert(
            "users_plans",
            data={
                "plan_id": plan_id, "user_id": user_id, "template_id": template_id,
                "start_date": now.isoformat(), "goal": goal,
                "level": draft.get("level", ""), "status": "active",
                "mesocycle_number": 1, "week_schedule": week_schedule,
            },
            upsert=True,
            on_conflict="user_id",
        )

    # Prefetch loading parameters (set_profiles) and exercise roles in 2 queries.
    # This is the source of truth for sets/reps/rir/rest/tempo — the LLM no longer
    # decides these. Values are periodized per week (W1 accumulation → W3 intensification
    # → W4 deload) and differ by role (compound vs isolation vs core vs cardio).
    user_level = draft.get("level", "Intermedio")
    profile_index = await _fetch_loading_params(goal, user_level)
    exercise_roles = await _fetch_exercise_roles(list(resolved.values()))

    if not profile_index:
        logger.warning(
            f"[LOADING] No set_profiles rows for goal={goal} level={user_level}. "
            f"All exercises will use DEFAULT_LOADING_PARAMS."
        )

    # Build workout rows for 4 weeks (only resolved exercises)
    workout_rows = []
    for week in range(1, 5):
        for d_idx, day in enumerate(draft.get("days", [])):
            day_title = day.get("title", day.get("name", f"Day {day.get('day_number', 1)}"))
            for e_idx, ex in enumerate(day.get("exercises", [])):
                ex_id = resolved.get((d_idx, e_idx))
                if not ex_id:
                    continue  # Skip unresolved

                # Lookup loading params by (week, role). Role defaults to "compound"
                # if the exercise isn't in exercise_roles (shouldn't happen, but defensive).
                role = exercise_roles.get(ex_id, "compound")
                params = profile_index.get((week, role))
                if params is None:
                    logger.warning(
                        f"[LOADING] No set_profile for goal={goal} level={user_level} "
                        f"week={week} role={role} ex_id={ex_id} — using DEFAULT_LOADING_PARAMS."
                    )
                    params = DEFAULT_LOADING_PARAMS

                workout_rows.append({
                    "id": str(uuid.uuid4()),
                    "user_id": user_id,
                    "week": week,
                    "day_name": day_title,
                    "exercise_id": ex_id,
                    "sets": str(params["sets"]),
                    "reps": params["reps"],
                    "rir": params["rir"],
                    "rest-seconds": params["rest_sec"],
                    "tempo": params["tempo"],
                    "created_at": now.isoformat(),
                    "notes": "",
                    "exercise_order": ex.get("exercise_order", ex.get("order", e_idx + 1)),
                })

    # Deduplicate: keep first occurrence of each exercise_id per day
    seen_per_day = {}
    deduped_rows = []
    for row in workout_rows:
        key = (row["week"], row["day_name"], row["exercise_id"])
        if key in seen_per_day:
            continue
        seen_per_day[key] = True
        deduped_rows.append(row)
    workout_rows = deduped_rows

    # Validate minimum exercises per day (week 1 is representative)
    MIN_EXERCISES_PER_DAY = 3
    exercises_per_day: dict[str, int] = {}
    for row in workout_rows:
        if row["week"] == 1:
            exercises_per_day[row["day_name"]] = exercises_per_day.get(row["day_name"], 0) + 1

    incomplete_days = [d for d, count in exercises_per_day.items() if count < MIN_EXERCISES_PER_DAY]
    if incomplete_days:
        return json.dumps({
            "success": False,
            "error": f"Días con ejercicios insuficientes: {incomplete_days}. "
                     f"Cada día debe tener al menos {MIN_EXERCISES_PER_DAY} ejercicios. "
                     f"Revisa el draft y agrega los ejercicios faltantes.",
            "exercises_per_day": exercises_per_day,
        }, ensure_ascii=False)

    if workout_rows:
        try:
            await supabase_bulk_insert("workouts", workout_rows)
        except Exception:
            for row in workout_rows:
                try:
                    await supabase_insert("workouts", row)
                except Exception:
                    pass

    response = {
        "success": True,
        "plan_id": plan_id,
        "workouts_created": len(workout_rows),
        "weeks": 4,
        "days_per_week": days_count,
    }

    if unresolved:
        response["warning"] = f"{len(unresolved)} ejercicio(s) no se pudieron resolver y fueron omitidos."
        response["unresolved_exercises"] = [
            {"day": draft["days"][u["day_idx"]]["title"] if u["day_idx"] < days_count else "?", "name": u["name"]}
            for u in unresolved
        ]

    return json.dumps(response, ensure_ascii=False)


# ═══════════════ US4: Scheduling ═══════════════


@tool
async def get_schedule_info(user_id: str) -> str:
    """Obtiene información del plan activo para agendar sesiones.

    Args:
        user_id: UUID del usuario

    Returns:
        Días por semana, nombres de sesiones, semana actual
    """
    plans = await supabase_query(
        "users_plans",
        select="plan_id,week_schedule,goal,level,mesocycle_number,start_date",
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
        limit=1,
    )

    if not plans:
        return json.dumps({"error": "No se encontró plan activo"})

    plan = plans[0]
    ws = plan["week_schedule"]

    # Get template days
    days = await supabase_query(
        "template_days",
        select="day_number,title",
        filters={"week_schedule": f"eq.{ws}"},
    )

    days_sorted = sorted(days, key=lambda d: d["day_number"])

    # Get current week from plan start_date
    if plan.get("start_date"):
        current_week = _compute_current_week(plan["start_date"])
    else:
        schedules = await supabase_query(
            "user_weekly_schedule",
            select="week",
            filters={"user_id": f"eq.{user_id}", "order": "week.desc"},
            limit=1,
        )
        current_week = schedules[0]["week"] if schedules else 1

    return json.dumps({
        "days_per_week": len(days_sorted),
        "sessions": [{"day_number": d["day_number"], "title": d["title"]} for d in days_sorted],
        "current_week": current_week,
        "week_schedule": ws,
    }, ensure_ascii=False)


VALID_WEEK_DAYS = {
    "Lunes", "Martes", "Miercoles", "Jueves", "Viernes", "Sabado", "Domingo",
}

# LLM often sends accented day names — map to enum values (no accents)
_ACCENT_MAP = str.maketrans("áéíóúÁÉÍÓÚ", "aeiouAEIOU")


def _normalize_week_day(raw: str) -> str:
    """Normalize week_day: strip accents and title-case."""
    return raw.strip().translate(_ACCENT_MAP).capitalize()


def _compute_current_week(start_date_iso: str) -> int:
    """Compute current training week (1-4) from plan start date."""
    start = datetime.fromisoformat(start_date_iso.replace("Z", "+00:00"))
    days_elapsed = (datetime.now(timezone.utc) - start).days
    return max(1, min((days_elapsed // 7) + 1, 4))


def _normalize_planned_day(raw: str) -> str:
    """Ensure planned_day is in DD/MM/YYYY format for the DB trigger."""
    raw = raw.strip()
    if len(raw) >= 8:
        return raw
    now_bogota = datetime.now(timezone.utc) + BOGOTA_UTC_OFFSET
    return f"{raw}/{now_bogota.year}"


@tool
async def schedule_sessions(user_id: str, sessions_json: str) -> str:
    """Agenda sesiones de entrenamiento en los días indicados por el usuario.

    Args:
        user_id: UUID del usuario
        sessions_json: JSON string con lista de sesiones:
            [{week_day: "Lunes", session_name: "Upper A", planned_day: "17/03"}]

    Returns:
        Confirmación de sesiones agendadas
    """
    sessions = json.loads(sessions_json)

    # Normalize and validate inputs
    errors = []
    for i, s in enumerate(sessions):
        wd = _normalize_week_day(s.get("week_day", ""))
        s["week_day"] = wd  # write back normalized value
        if wd not in VALID_WEEK_DAYS:
            errors.append(
                f"Sesión {i+1}: week_day '{wd}' inválido. "
                f"Usar: {', '.join(sorted(VALID_WEEK_DAYS))}"
            )
        if not s.get("session_name"):
            errors.append(f"Sesión {i+1}: session_name vacío")
        if not s.get("planned_day"):
            errors.append(f"Sesión {i+1}: planned_day vacío")

    if errors:
        return json.dumps({"success": False, "errors": errors}, ensure_ascii=False)

    # Get current week from plan start_date
    plans = await supabase_query(
        "users_plans",
        select="plan_id,start_date",
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
        limit=1,
    )

    if not plans:
        return json.dumps({"success": False, "error": "No se encontró plan activo para el usuario"})

    plan = plans[0]
    if plan.get("start_date"):
        current_week = _compute_current_week(plan["start_date"])
    else:
        schedules = await supabase_query(
            "user_weekly_schedule",
            select="week",
            filters={"user_id": f"eq.{user_id}", "order": "week.desc"},
            limit=1,
        )
        current_week = schedules[0]["week"] if schedules else 1

    rows = []
    for s in sessions:
        rows.append({
            "user_id": user_id,
            "week": current_week,
            "week_day": s["week_day"],
            "session_name": s["session_name"],
            "planned_day": _normalize_planned_day(s["planned_day"]),
            "Completed": False,
        })

    try:
        await supabase_bulk_insert(
            "user_weekly_schedule",
            rows,
            upsert=True,
            on_conflict="user_id,week,week_day",
        )
    except Exception as e:
        detail = str(e)[:200]
        return json.dumps({"success": False, "error": f"Error al guardar sesiones: {detail}"})

    return json.dumps({
        "success": True,
        "sessions_created": len(rows),
        "week": current_week,
    })


# ═══════════════ US6: Mesocycle Renewal ═══════════════


@tool
async def get_mesocycle_status(user_id: str) -> str:
    """Consulta el estado del mesociclo actual — si está listo para renovación.

    Args:
        user_id: UUID del usuario

    Returns:
        Estado del mesociclo: semana actual, sesiones completadas en W4, si puede renovar
    """
    plans = await supabase_query(
        "users_plans",
        select="plan_id,mesocycle_number,week_schedule,goal,level",
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
        limit=1,
    )

    if not plans:
        return json.dumps({"error": "No se encontró plan activo"})

    plan = plans[0]

    # Count W4 sessions
    w4_all = await supabase_query(
        "user_weekly_schedule",
        select='day_routine_id,"Completed"',
        filters={"user_id": f"eq.{user_id}", "week": "eq.4"},
    )

    w4_completed = sum(1 for s in w4_all if s.get("Completed", False))
    w4_total = len(w4_all)
    can_renew = w4_total > 0 and w4_completed == w4_total

    return json.dumps({
        "mesocycle_number": plan["mesocycle_number"],
        "week4_completed": w4_completed,
        "week4_total": w4_total,
        "can_renew": can_renew,
        "week_schedule": plan["week_schedule"],
        "goal": plan["goal"],
        "level": plan["level"],
    }, ensure_ascii=False)


async def _validate_mesocycle_complete(user_id: str) -> str | None:
    """Guard for renew_* tools: ensure W4 is fully completed before destructive renewal.

    Returns:
        None if mesocycle is complete (caller may proceed).
        JSON error string (MESOCYCLE_NOT_COMPLETE) if not — caller should return it directly.
    """
    w4 = await supabase_query(
        "user_weekly_schedule",
        select='day_routine_id,"Completed"',
        filters={"user_id": f"eq.{user_id}", "week": "eq.4"},
    )
    w4_total = len(w4)
    w4_completed = sum(1 for s in w4 if s.get("Completed", False))
    if w4_total == 0 or w4_completed < w4_total:
        return json.dumps({
            "success": False,
            "error": "MESOCYCLE_NOT_COMPLETE",
            "week4_completed": w4_completed,
            "week4_total": w4_total,
            "message": (
                f"Aún no terminas el mesociclo actual "
                f"(semana 4: {w4_completed}/{w4_total} sesiones completadas). "
                f"Termínalo primero, o si solo quieres cambiar QUÉ días específicos "
                f"entrenas (no cuántos), usa schedule_sessions sin renovar."
            ),
        }, ensure_ascii=False)
    return None


@tool
async def renew_maintain(user_id: str) -> str:
    """Renueva el mesociclo MANTENIENDO la rutina actual.

    Conserva todos los ejercicios. Limpia el schedule e incrementa el mesociclo.
    Después de usar esta herramienta, el usuario debe agendar semana 1.

    REQUIERE: mesociclo actual COMPLETO (W4 con todas las sesiones marcadas Completed=true).

    Args:
        user_id: UUID del usuario

    Returns:
        Confirmación con nuevo número de mesociclo, o error MESOCYCLE_NOT_COMPLETE
    """
    # GUARD: mesociclo debe estar completo antes de renovar
    guard_err = await _validate_mesocycle_complete(user_id)
    if guard_err is not None:
        return guard_err

    plans = await supabase_query(
        "users_plans",
        select="plan_id,mesocycle_number,week_schedule",
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
        limit=1,
    )
    if not plans:
        return json.dumps({"success": False, "error": "No se encontró plan activo"})

    new_meso = plans[0]["mesocycle_number"] + 1

    await supabase_delete(
        "user_weekly_schedule",
        filters={"user_id": f"eq.{user_id}"},
    )

    now = datetime.now(timezone.utc)
    await supabase_update(
        "users_plans",
        data={
            "mesocycle_number": new_meso,
            "last_renewal_date": now.isoformat(),
            "start_date": now.isoformat(),
        },
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
    )

    return json.dumps({
        "success": True,
        "new_mesocycle_number": new_meso,
        "week_schedule": plans[0]["week_schedule"],
        "action": "MANTENER_RUTINA",
    }, ensure_ascii=False)


@tool
async def renew_rotate_exercises(user_id: str, health_status: str = "A") -> str:
    """Renueva el mesociclo ROTANDO ejercicios por alternativas del mismo patrón.

    Mantiene la estructura (días, sets, reps) pero cambia ejercicios por alternativas.
    Después de usar esta herramienta, el usuario debe agendar semana 1.

    REQUIERE: mesociclo actual COMPLETO (W4 con todas las sesiones marcadas Completed=true).

    Args:
        user_id: UUID del usuario
        health_status: Código de salud del usuario (A, B, C, D, E). Default A

    Returns:
        Resumen de ejercicios rotados y conservados, o error MESOCYCLE_NOT_COMPLETE
    """
    import random

    # GUARD: mesociclo debe estar completo antes de renovar
    guard_err = await _validate_mesocycle_complete(user_id)
    if guard_err is not None:
        return guard_err

    plans = await supabase_query(
        "users_plans",
        select="plan_id,mesocycle_number,level",
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
        limit=1,
    )
    if not plans:
        return json.dumps({"success": False, "error": "No se encontró plan activo"})

    level = plans[0]["level"]
    new_meso = plans[0]["mesocycle_number"] + 1

    # Get W1 exercises
    w1_workouts = await supabase_query(
        "workouts",
        select="id,exercise_id,day_name",
        filters={"user_id": f"eq.{user_id}", "week": "eq.1"},
    )
    if not w1_workouts:
        return json.dumps({"success": False, "error": "No se encontraron ejercicios en semana 1"})

    # Get exercise details
    ex_ids = list({w["exercise_id"] for w in w1_workouts})
    exercises_data = await supabase_query(
        "exercises",
        select="exercise_id,spanish_name,pattern,role,level",
        filters={"exercise_id": f"in.({','.join(ex_ids)})"},
    )
    ex_map = {e["exercise_id"]: e for e in exercises_data}

    # Lookup user profile for equipment filtering (HOME users)
    profile = await _lookup_user_profile(user_id)
    allowed_equip = None
    if profile and profile["training_environment"] == "HOME":
        home_eq = profile.get("home_equipment", "") or "peso corporal"
        allowed_equip = _normalize_equipment(home_eq)

    # Find alternatives for each unique exercise
    rotated = []
    kept = []
    swap_map = {}

    for ex_id in ex_ids:
        ex = ex_map.get(ex_id)
        if not ex:
            kept.append(ex_id)
            continue

        candidates = await supabase_query(
            "exercises",
            select="exercise_id,spanish_name,pattern,role,level,equipment,main_muscle",
            filters={
                "pattern": f"eq.{ex['pattern']}",
                "role": f"eq.{ex['role']}",
                "exercise_id": f"neq.{ex_id}",
            },
            limit=10,
        )

        # Apply equipment filter (HOME users only get exercises matching their equipment)
        if allowed_equip:
            equip_filtered = [c for c in candidates if c.get("equipment", "").lower() in allowed_equip]
            if equip_filtered:
                candidates = equip_filtered

        # Apply health restrictions
        candidates = _apply_health_filter(candidates, health_status)

        if candidates:
            chosen = random.choice(candidates)
            swap_map[ex_id] = chosen["exercise_id"]
            rotated.append({"old": ex.get("spanish_name", ex_id), "new": chosen["spanish_name"]})
        else:
            kept.append(ex.get("spanish_name", ex_id))

    # Apply swaps across all 4 weeks
    for old_id, new_id in swap_map.items():
        await supabase_update(
            "workouts",
            data={"exercise_id": new_id},
            filters={"user_id": f"eq.{user_id}", "exercise_id": f"eq.{old_id}"},
        )

    # Clear schedule
    await supabase_delete(
        "user_weekly_schedule",
        filters={"user_id": f"eq.{user_id}"},
    )

    # Increment mesocycle
    now = datetime.now(timezone.utc)
    await supabase_update(
        "users_plans",
        data={
            "mesocycle_number": new_meso,
            "last_renewal_date": now.isoformat(),
            "start_date": now.isoformat(),
        },
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
    )

    return json.dumps({
        "success": True,
        "new_mesocycle_number": new_meso,
        "rotated": rotated,
        "kept": kept,
        "action": "ROTAR_EJERCICIOS",
    }, ensure_ascii=False)


@tool
async def renew_change_days(user_id: str, new_days_per_week: int) -> str:
    """Renueva el mesociclo CAMBIANDO la frecuencia (cantidad de días por semana).

    Solo para cuando el usuario quiere entrenar MÁS o MENOS días que en el mesociclo actual.
    NO usar cuando el usuario solo quiere cambiar QUÉ días específicos entrena (usa schedule_sessions).

    Elimina ejercicios y schedule actuales, actualiza el plan con nuevo schedule.
    Después de usar esta herramienta, DEBES crear una nueva rutina usando:
    get_day_requirements → get_exercises_for_draft → save_workout_plan (con "is_renewal": true en el JSON)

    REQUIERE:
    - Mesociclo actual COMPLETO (W4 con todas las sesiones marcadas Completed=true)
    - new_days_per_week debe ser DIFERENTE al actual

    Args:
        user_id: UUID del usuario
        new_days_per_week: Nuevo número de días por semana (2-6). DEBE ser distinto al actual.

    Returns:
        Confirmación y nuevo week_schedule, o error SAME_DAYS_PER_WEEK / MESOCYCLE_NOT_COMPLETE
    """
    if new_days_per_week < 2 or new_days_per_week > 6:
        return json.dumps({
            "success": False,
            "error": f"Días debe ser entre 2 y 6, recibido: {new_days_per_week}",
        })

    ws_map = {2: "fb_2", 3: "fb_3", 4: "ul_4", 5: "ppl_5", 6: "ppl_6"}
    new_ws = ws_map[new_days_per_week]

    plans = await supabase_query(
        "users_plans",
        select="plan_id,mesocycle_number,goal,level,week_schedule",
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
        limit=1,
    )
    if not plans:
        return json.dumps({"success": False, "error": "No se encontró plan activo"})

    plan = plans[0]

    # GUARD 1: nueva cantidad debe ser distinta a la actual
    ws_to_days = {"fb_2": 2, "fb_3": 3, "ul_4": 4, "ppl_5": 5, "ppl_6": 6}
    current_days = ws_to_days.get(plan.get("week_schedule", ""), 0)
    if new_days_per_week == current_days:
        return json.dumps({
            "success": False,
            "error": "SAME_DAYS_PER_WEEK",
            "current_days_per_week": current_days,
            "message": (
                f"Ya entrenas {current_days} días por semana. "
                f"Si quieres cambiar QUÉ días específicos entrenas (no cuántos), "
                f"usa schedule_sessions con los días nuevos. "
                f"Si quieres mantener los mismos ejercicios para el siguiente mesociclo, "
                f"usa renew_maintain."
            ),
        }, ensure_ascii=False)

    # GUARD 2: mesociclo debe estar completo
    guard_err = await _validate_mesocycle_complete(user_id)
    if guard_err is not None:
        return guard_err

    new_meso = plan["mesocycle_number"] + 1

    # Delete old workouts and schedule
    await supabase_delete("workouts", filters={"user_id": f"eq.{user_id}"})
    await supabase_delete("user_weekly_schedule", filters={"user_id": f"eq.{user_id}"})

    # Update plan
    goal_code_map = {
        "Ganar masa muscular": "hyp", "Mejorar fuerza": "str",
        "Bajar grasa": "cut", "Mejorar resistencia": "end",
        "Salud general / recomposición corporal": "rec",
    }
    level_map = {"Principiante": "beg", "Intermedio": "int", "Avanzado": "adv"}
    template_id = f"tpl_{new_ws}_{goal_code_map.get(plan['goal'], 'hyp')}_{level_map.get(plan['level'], 'int')}"

    now = datetime.now(timezone.utc)
    await supabase_update(
        "users_plans",
        data={
            "mesocycle_number": new_meso,
            "week_schedule": new_ws,
            "template_id": template_id,
            "last_renewal_date": now.isoformat(),
            "start_date": now.isoformat(),
        },
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
    )

    return json.dumps({
        "success": True,
        "new_mesocycle_number": new_meso,
        "new_week_schedule": new_ws,
        "new_days_per_week": new_days_per_week,
        "goal": plan["goal"],
        "level": plan["level"],
        "action": "CAMBIAR_DIAS",
        "next_step": "Crear nueva rutina con get_day_requirements + get_exercises_for_draft + save_workout_plan (is_renewal: true)",
    }, ensure_ascii=False)


# ═══════════════ Email ═══════════════


async def _send_email(to: str, subject: str, html_body: str) -> None:
    """Send HTML email via Gmail SMTP."""
    import aiosmtplib
    from email.mime.text import MIMEText
    from email.mime.multipart import MIMEMultipart

    gmail_user = os.getenv("GMAIL_USER", "")
    gmail_password = os.getenv("GMAIL_APP_PASSWORD", "")

    if not gmail_user or not gmail_password:
        raise ValueError("GMAIL_USER and GMAIL_APP_PASSWORD env vars required")

    msg = MIMEMultipart("alternative")
    msg["From"] = f"Kairos Personal Trainer <{gmail_user}>"
    msg["To"] = to
    msg["Subject"] = subject
    msg.attach(MIMEText(html_body, "html"))

    await aiosmtplib.send(
        msg,
        hostname="smtp.gmail.com",
        port=587,
        start_tls=True,
        username=gmail_user,
        password=gmail_password,
    )


def _build_routine_html(user_name: str, plan: dict, exercises_by_day: dict[str, list[dict]]) -> str:
    """Build HTML email with routine tables grouped by day."""
    goal = plan.get("goal", "")
    level = plan.get("level", "")
    ws = plan.get("week_schedule", "")

    day_sections = ""
    for day_name in sorted(exercises_by_day.keys()):
        exercises = exercises_by_day[day_name]
        rows_html = ""
        for ex in exercises:
            video = f'<a href="{ex["link"]}" style="color:#4A90D9;">Video</a>' if ex.get("link") else ""
            rows_html += f"""<tr>
                <td style="padding:8px;border-bottom:1px solid #eee;">{ex.get("exercise_order", "")}</td>
                <td style="padding:8px;border-bottom:1px solid #eee;">{ex.get("spanish_name", "")}</td>
                <td style="padding:8px;border-bottom:1px solid #eee;">{ex.get("main_muscle", "")}</td>
                <td style="padding:8px;border-bottom:1px solid #eee;">{ex.get("sets", "")}x{ex.get("reps", "")}</td>
                <td style="padding:8px;border-bottom:1px solid #eee;">RIR {ex.get("rir", "")}</td>
                <td style="padding:8px;border-bottom:1px solid #eee;">{ex.get("rest_seconds", "")}s</td>
                <td style="padding:8px;border-bottom:1px solid #eee;">{video}</td>
            </tr>"""

        day_sections += f"""
        <h2 style="color:#2C3E50;margin-top:24px;">{day_name}</h2>
        <table style="width:100%;border-collapse:collapse;font-size:14px;">
            <thead>
                <tr style="background:#f8f9fa;">
                    <th style="padding:8px;text-align:left;">#</th>
                    <th style="padding:8px;text-align:left;">Ejercicio</th>
                    <th style="padding:8px;text-align:left;">Músculo</th>
                    <th style="padding:8px;text-align:left;">Series x Reps</th>
                    <th style="padding:8px;text-align:left;">RIR</th>
                    <th style="padding:8px;text-align:left;">Descanso</th>
                    <th style="padding:8px;text-align:left;">Video</th>
                </tr>
            </thead>
            <tbody>{rows_html}</tbody>
        </table>"""

    return f"""<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family:Arial,sans-serif;max-width:700px;margin:0 auto;padding:20px;color:#333;">
    <div style="background:#2C3E50;color:white;padding:20px;border-radius:8px 8px 0 0;">
        <h1 style="margin:0;">Tu Rutina de Entrenamiento</h1>
        <p style="margin:8px 0 0;">{user_name} | {goal} | {level} | {ws}</p>
    </div>
    <div style="padding:20px;background:#fff;border:1px solid #e0e0e0;border-radius:0 0 8px 8px;">
        {day_sections}
        <hr style="margin:24px 0;border:none;border-top:1px solid #eee;">
        <p style="color:#888;font-size:12px;text-align:center;">
            Kairos Personal Trainer — Tu entrenador virtual
        </p>
    </div>
</body>
</html>"""


@tool
async def send_routine_email(user_id: str) -> str:
    """Envía la rutina de entrenamiento completa al email del usuario.

    Consulta la base de datos directamente para obtener email, plan y ejercicios.
    No necesita parámetros adicionales — todo se obtiene del user_id.

    Args:
        user_id: UUID del usuario

    Returns:
        Confirmación de envío con dirección de email
    """
    # 1. Get user email
    users = await supabase_query(
        "users",
        select="full_name,email",
        filters={"user_id": f"eq.{user_id}"},
        limit=1,
    )
    if not users:
        return json.dumps({"success": False, "error": "Usuario no encontrado"})

    user = users[0]
    email = user.get("email")
    if not email:
        return json.dumps({
            "success": False,
            "error": "El usuario no tiene email registrado. Pregúntale su email y usa update_user_profile para guardarlo, luego reintenta send_routine_email.",
        })

    # 2. Get plan
    plans = await supabase_query(
        "users_plans",
        select="plan_id,goal,level,week_schedule",
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
        limit=1,
    )
    if not plans:
        return json.dumps({"success": False, "error": "No se encontró plan activo"})

    plan = plans[0]

    # 3. Get W1 exercises
    workouts = await supabase_query(
        "workouts",
        select="day_name,exercise_id,sets,reps,rir,rest-seconds,tempo,exercise_order",
        filters={"user_id": f"eq.{user_id}", "week": "eq.1"},
    )
    if not workouts:
        return json.dumps({"success": False, "error": "No se encontraron ejercicios"})

    # 4. Get exercise details
    ex_ids = list({w["exercise_id"] for w in workouts})
    exercises = await supabase_query(
        "exercises",
        select="exercise_id,spanish_name,main_muscle,link",
        filters={"exercise_id": f"in.({','.join(ex_ids)})"},
    )
    ex_map = {e["exercise_id"]: e for e in exercises}

    # 5. Build exercises by day
    exercises_by_day: dict[str, list[dict]] = {}
    for w in workouts:
        day = w["day_name"]
        if day not in exercises_by_day:
            exercises_by_day[day] = []
        ex = ex_map.get(w["exercise_id"], {})
        exercises_by_day[day].append({
            "exercise_order": w.get("exercise_order", 0),
            "spanish_name": ex.get("spanish_name", w["exercise_id"]),
            "main_muscle": ex.get("main_muscle", ""),
            "sets": w.get("sets", ""),
            "reps": w.get("reps", ""),
            "rir": w.get("rir", ""),
            "rest_seconds": w.get("rest-seconds", ""),
            "link": ex.get("link", ""),
        })

    # Sort exercises within each day
    for day in exercises_by_day:
        exercises_by_day[day].sort(key=lambda x: x.get("exercise_order", 0))

    # 6. Build HTML and send
    html = _build_routine_html(user.get("full_name", ""), plan, exercises_by_day)

    try:
        print(f"[EMAIL] Sending routine to {email} for user {user_id} ({len(exercises_by_day)} days)", flush=True)
        await _send_email(
            to=email,
            subject=f"Tu rutina Kairos — {plan.get('goal', '')}",
            html_body=html,
        )
        print(f"[EMAIL] Sent successfully to {email}", flush=True)
    except Exception as e:
        print(f"[EMAIL] FAILED to send to {email}: {e}", flush=True)
        return json.dumps({"success": False, "error": f"Error al enviar email: {str(e)[:200]}"})

    return json.dumps({
        "success": True,
        "email": email,
        "days_sent": len(exercises_by_day),
        "exercises_total": sum(len(v) for v in exercises_by_day.values()),
    })


# ═══════════════ ONBOARDING & PROFILE ═══════════════

from cases.case5_onboarding_kyc.tools_supabase import (
    _normalize_enum,
    _VALID_GOALS, _GOAL_ALIASES,
    _VALID_EXPERIENCE, _EXPERIENCE_ALIASES,
    _VALID_ENVIRONMENT, _ENVIRONMENT_ALIASES,
    _VALID_SCHEDULE, _SCHEDULE_ALIASES,
    _VALID_STYLE, _STYLE_ALIASES,
    _VALID_SEX, _SEX_ALIASES,
    _VALID_CARDIO, _CARDIO_ALIASES,
    _VALID_CARDIO_FREQ, _CARDIO_FREQ_ALIASES,
    _VALID_FREQUENCY, _FREQUENCY_ALIASES,
)
from cases.case5_onboarding_kyc.prompts import HEALTH_CLASSIFIER_PROMPT


# Experience → fitness level mapping
_EXPERIENCE_TO_LEVEL = {
    "Nunca he entrenado": "Principiante",
    "Menos de 6 meses": "Principiante",
    "6 a 12 meses": "Intermedio",
    "1 a 3 años": "Intermedio",
    "Más de 3 años": "Avanzado",
}


async def _classify_health(health_text: str) -> dict:
    """Classify health text into code A-E using LLM.

    Returns: {"code": "A", "zones": []}
    """
    from src.shared.llm import get_llm

    # Short-circuit for obvious "no issues" phrases
    lower = health_text.lower().strip()
    no_issues = [
        "sin restricciones", "no", "ninguna", "estoy bien", "nada",
        "no tengo", "sano", "sana", "sin problemas", "todo bien",
        "no, estoy bien", "nop", "nel", "nope",
    ]
    if lower in no_issues or lower.startswith("no,") or lower.startswith("no "):
        return {"code": "A", "zones": []}

    prompt = HEALTH_CLASSIFIER_PROMPT.format(health_text=health_text)
    llm = get_llm(temperature=0)
    response = await llm.ainvoke(prompt)

    content = response.content or ""
    if isinstance(content, list):
        content = "".join(
            p.get("text", "") if isinstance(p, dict) else str(p) for p in content
        )

    try:
        result = json.loads(content.strip())
        return {"code": result.get("code", "A"), "zones": result.get("zones", [])}
    except (json.JSONDecodeError, AttributeError):
        logger.warning(f"[HEALTH] Failed to parse classification: {content}")
        return {"code": "A", "zones": []}


@tool
async def register_new_user(
    phone: str,
    full_name: str,
    primary_goal: str,
    training_experience: str,
    days_available: int,
    training_environment: str,
    health_text: str,
    home_equipment: str = "",
) -> str:
    """Registra un usuario nuevo con sus datos esenciales de perfil.

    Llama esta herramienta cuando hayas recolectado los datos necesarios del usuario
    para crear su rutina: objetivo, experiencia, días disponibles, ambiente de
    entrenamiento, y estado de salud.

    Args:
        phone: Número completo del usuario (ej: "573001234567")
        full_name: Nombre del usuario
        primary_goal: Objetivo de entrenamiento (ej: "Ganar masa muscular", "Bajar grasa")
        training_experience: Experiencia (ej: "Más de 3 años", "Nunca he entrenado")
        days_available: Días por semana disponibles (2-6)
        training_environment: "GYM" o "HOME"
        health_text: Descripción de salud del usuario (ej: "Sin restricciones", "Me duele la rodilla")
        home_equipment: Equipo en casa, solo si training_environment=HOME (ej: "mancuernas, bandas")

    Returns:
        JSON con user_id, health_code, y confirmación del registro
    """
    # 1. Classify health
    health_result = await _classify_health(health_text)
    health_code = health_result["code"]

    if health_code == "E":
        return json.dumps({
            "success": False,
            "health_code": "E",
            "route_to_trainer": True,
            "affected_zones": health_result["zones"],
            "message": "Condición de salud severa detectada. Recomendar consulta con profesional.",
        }, ensure_ascii=False)

    # 2. Create user in users table
    user_id = str(uuid.uuid4())
    cel_number = int(phone) if phone.isdigit() else 0

    user_data = {
        "user_id": user_id,
        "full_name": full_name,
        "cel_number": cel_number,
        "full_phone_number": phone,
        "timezone": "America/Bogota",
        "created_at": datetime.now(timezone.utc).isoformat(),
        "country_indicative": 57,
    }

    await supabase_insert(table="users", data=user_data)

    # 3. Normalize enums
    goal = _normalize_enum(primary_goal, _VALID_GOALS, _GOAL_ALIASES, "Salud general / recomposición corporal")
    experience = _normalize_enum(training_experience, _VALID_EXPERIENCE, _EXPERIENCE_ALIASES, "Menos de 6 meses")
    environment = _normalize_enum(training_environment, _VALID_ENVIRONMENT, _ENVIRONMENT_ALIASES, "GYM")
    fitness_level = _EXPERIENCE_TO_LEVEL.get(experience, "Intermedio")

    # 4. Save gym profile with essential fields only (rest stays NULL)
    profile_data = {
        "whatsapp_id": cel_number,
        "full_name": full_name,
        "primary_goal": goal,
        "training_experience": experience,
        "days_available": days_available,
        "training_environment": environment,
        "health_status": health_code,
        "fitness_level": fitness_level,
    }
    if home_equipment and environment == "HOME":
        profile_data["home_equipment"] = home_equipment

    await supabase_insert(table="users_gym_profile", data=profile_data, upsert=True)

    logger.info(f"[REGISTER] New user {full_name} ({phone}) registered: {goal}, {experience}, {days_available}d, {environment}, health={health_code}")

    return json.dumps({
        "success": True,
        "user_id": user_id,
        "health_code": health_code,
        "fitness_level": fitness_level,
        "goal": goal,
        "days_available": days_available,
        "environment": environment,
    }, ensure_ascii=False)


# Allowed fields for update_user_profile
_USERS_FIELDS = {"email"}
_PROFILE_FIELDS = {
    "age", "biological_sex", "height_cm", "weight_kg",
    "preferred_schedule", "training_style", "priority_muscles",
    "disliked_exercises", "cardio_type", "cardio_frequency",
    "primary_goal", "training_experience", "days_available",
    "training_environment", "home_equipment", "health_status",
    "fitness_level",
}

# Enum normalization map for profile fields
_ENUM_NORMALIZERS = {
    "primary_goal": (_VALID_GOALS, _GOAL_ALIASES, "Salud general / recomposición corporal"),
    "training_experience": (_VALID_EXPERIENCE, _EXPERIENCE_ALIASES, "Menos de 6 meses"),
    "preferred_schedule": (_VALID_SCHEDULE, _SCHEDULE_ALIASES, "Mañana"),
    "training_style": (_VALID_STYLE, _STYLE_ALIASES, "Mixto"),
    "biological_sex": (_VALID_SEX, _SEX_ALIASES, "M"),
    "cardio_type": (_VALID_CARDIO, _CARDIO_ALIASES, "No"),
    "cardio_frequency": (_VALID_CARDIO_FREQ, _CARDIO_FREQ_ALIASES, "0"),
    "training_environment": (_VALID_ENVIRONMENT, _ENVIRONMENT_ALIASES, "GYM"),
}


@tool
async def update_user_profile(user_id: str, field: str, value: str) -> str:
    """Actualiza un campo del perfil del usuario.

    Usa esta herramienta cuando necesites guardar un dato que el usuario
    acaba de proporcionar (email, edad, peso, etc).

    Args:
        user_id: UUID del usuario
        field: Campo a actualizar. Valores válidos:
            - Para tabla users: "email"
            - Para tabla users_gym_profile: "age", "biological_sex", "height_cm",
              "weight_kg", "preferred_schedule", "training_style", "priority_muscles",
              "disliked_exercises", "cardio_type", "cardio_frequency", "primary_goal",
              "training_experience", "days_available", "training_environment",
              "home_equipment", "health_status", "fitness_level"
        value: Nuevo valor del campo

    Returns:
        JSON confirmando la actualización
    """
    if field in _USERS_FIELDS:
        result = await supabase_update(
            "users", {field: value}, {"user_id": f"eq.{user_id}"}
        )
        return json.dumps({"success": True, "table": "users", "field": field, "updated": len(result)})

    if field in _PROFILE_FIELDS:
        # Normalize enum if applicable
        final_value = value
        if field in _ENUM_NORMALIZERS:
            valid, aliases, default = _ENUM_NORMALIZERS[field]
            final_value = _normalize_enum(value, valid, aliases, default)

        # Convert numeric fields
        if field in ("age", "height_cm", "days_available"):
            final_value = int(float(final_value))
        elif field == "weight_kg":
            final_value = float(final_value)

        # Get phone from user_id to update gym profile
        users = await supabase_query(
            "users", select="full_phone_number",
            filters={"user_id": f"eq.{user_id}"}, limit=1,
        )
        if not users:
            return json.dumps({"success": False, "error": "Usuario no encontrado"})

        phone = str(users[0]["full_phone_number"])
        result = await supabase_update(
            "users_gym_profile", {field: final_value},
            {"whatsapp_id": f"eq.{phone}"},
        )
        return json.dumps({"success": True, "table": "users_gym_profile", "field": field, "updated": len(result)})

    return json.dumps({
        "success": False,
        "error": f"Campo '{field}' no es válido. Campos permitidos: {sorted(_USERS_FIELDS | _PROFILE_FIELDS)}",
    })


# ═══════════════ DRAFT ROUTINE PREVIEW ═══════════════

async def _fetch_alternatives_for_exercise(
    pattern: str,
    level: str,
    exclude_ids: list[str],
    profile: dict | None,
    limit: int = 4,
) -> list[dict]:
    """Fetch alternative exercises for a given pattern, excluding already-used IDs."""
    rows = await supabase_query(
        "exercises",
        select="exercise_id,spanish_name,pattern,role,main_muscle,link,equipment,level",
        filters={"pattern": f"eq.{pattern}"},
        limit=limit * 4,
    )

    # Filter by level compatibility
    level_order = {"Principiante": 1, "Intermedio": 2, "Avanzado": 3}
    user_level = level_order.get(level, 2)
    filtered = [
        r for r in rows
        if level_order.get(r.get("level", "Intermedio"), 2) <= user_level
        and r.get("exercise_id") not in exclude_ids
    ]

    # Equipment filter (HOME users)
    if profile and profile.get("training_environment") == "HOME":
        home_eq = profile.get("home_equipment", "peso corporal")
        allowed = _normalize_equipment(home_eq)
        equip_filtered = [r for r in filtered if r.get("equipment", "").lower() in allowed]
        if equip_filtered:
            filtered = equip_filtered

    # Health filter
    hs = profile.get("health_status", "A") if profile else "A"
    filtered = _apply_health_filter(filtered, hs)

    return [
        {
            "exercise_id": r["exercise_id"],
            "spanish_name": r["spanish_name"],
            "main_muscle": r.get("main_muscle", ""),
            "link": r.get("link", ""),
        }
        for r in filtered[:limit]
    ]


@tool
async def save_draft_preview(user_id: str, draft_json: str) -> str:
    """Guarda el borrador de rutina y genera un enlace de preview para el usuario.

    Enriquece cada ejercicio con 3-5 alternativas del mismo patrón/nivel.
    El usuario puede ver la rutina visualmente y hacer swaps antes de aprobar.

    Args:
        user_id: UUID del usuario
        draft_json: JSON string con la estructura de DraftRoutine:
            {week_schedule, goal, level, days: [{day_number, title, exercises: [{exercise_id, spanish_name, pattern, role, sets, reps, rir, rest_seconds, exercise_order}]}]}

    Returns:
        URL de preview y código de acceso
    """
    draft = json.loads(draft_json)
    level = draft.get("level", "Intermedio")

    # Lookup user profile for health/equipment filtering
    profile = await _lookup_user_profile(user_id)

    # Collect all exercise_ids in the draft to exclude from alternatives
    all_exercise_ids: set[str] = set()
    for day in draft.get("days", []):
        for ex in day.get("exercises", []):
            eid = ex.get("exercise_id", "")
            if eid:
                all_exercise_ids.add(eid)

    # Enrich each exercise with alternatives
    for day in draft.get("days", []):
        for ex in day.get("exercises", []):
            pattern = ex.get("pattern", "")
            eid = ex.get("exercise_id", "")
            if not pattern:
                ex["alternatives"] = []
                continue

            # Exclude all exercises already in the draft + this exercise
            exclude = list(all_exercise_ids)
            alts = await _fetch_alternatives_for_exercise(
                pattern=pattern,
                level=level,
                exclude_ids=exclude,
                profile=profile,
                limit=4,
            )
            ex["alternatives"] = alts

    # Generate short code
    now = datetime.now(timezone.utc)
    code = f"{int(now.timestamp()) % 0xFFFFFF:06x}"

    # Delete any previous pending drafts for this user
    try:
        await supabase_delete(
            "draft_routines",
            filters={"user_id": f"eq.{user_id}", "status": "eq.pending"},
        )
    except Exception:
        pass  # OK if no previous drafts

    # Validate and normalize against canonical contract before persisting.
    # Coerces rir/reps to str, renames alternative.link -> video_link, drops extras.
    normalized = normalize_draft_data(draft)

    # Insert draft
    await supabase_insert("draft_routines", {
        "user_id": user_id,
        "code": code,
        "draft_data": normalized,
        "status": "pending",
        "created_at": now.isoformat(),
        "expires_at": (now + timedelta(hours=24)).isoformat(),
    })

    frontend_url = os.getenv("FRONTEND_URL", "https://kairos-tracker.web.app")
    preview_url = f"{frontend_url}/draft?c={code}"

    return json.dumps({
        "success": True,
        "url": preview_url,
        "code": code,
        "exercises_enriched": sum(
            len(ex.get("alternatives", []))
            for day in normalized.get("days", [])
            for ex in day.get("exercises", [])
        ),
    }, ensure_ascii=False)
