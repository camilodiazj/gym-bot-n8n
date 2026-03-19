"""Tools for the Unified Agent Kairos (Case 6).

All tools use Supabase PostgREST via src/shared/supabase_client.
Tools are grouped by phase:
- US1: get_todays_routine, confirm_workout_completion, decline_workout, create_magic_link
- US3: get_day_requirements, get_exercises_for_draft, find_exercise_alternatives, save_workout_plan
- US4: get_schedule_info, schedule_sessions
- US6: get_mesocycle_status
"""

import json
import os
import uuid
from datetime import datetime, timedelta, timezone

from langchain_core.tools import tool

from src.shared.supabase_client import (
    supabase_query,
    supabase_insert,
    supabase_update,
    supabase_bulk_insert,
)


BOGOTA_UTC_OFFSET = timedelta(hours=-5)


def _today_bogota() -> str:
    now_utc = datetime.now(timezone.utc)
    return (now_utc + BOGOTA_UTC_OFFSET).strftime("%Y-%m-%d")


def _yesterday_bogota() -> str:
    now_utc = datetime.now(timezone.utc)
    return (now_utc + BOGOTA_UTC_OFFSET - timedelta(days=1)).strftime("%Y-%m-%d")


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


@tool
async def confirm_workout_completion(user_id: str, session_date: str | None = None) -> str:
    """Marca una sesión como completada. Soporta grace period (hoy o ayer).

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
        # Find uncompleted session for this date
        sessions = await supabase_query(
            "user_weekly_schedule",
            select="day_routine_id,session_name,week",
            filters={
                "user_id": f"eq.{user_id}",
                "planned_day": f"eq.{date}",
                "Completed": "eq.false",
            },
            limit=1,
        )

        if sessions:
            session = sessions[0]
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


@tool
async def get_exercises_for_draft(
    pattern: str,
    level: str,
    equipment: str | None = None,
    exclude_muscle: str | None = None,
    limit: int = 5,
) -> str:
    """Busca ejercicios candidatos por patrón de movimiento para construir el borrador de rutina.

    Args:
        pattern: Patrón de movimiento (push_h, push_v, pull_h, pull_v, squat, hinge, lunge, core, arm, accessory)
        level: Nivel del usuario (Principiante, Intermedio, Avanzado)
        equipment: Equipamiento disponible (opcional, ej: "mancuernas, bandas")
        exclude_muscle: Músculo a excluir (opcional, ej: "Calfs" para ejercicios no deseados)
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

    # Filter by equipment if specified
    if equipment:
        equip_list = [e.strip().lower() for e in equipment.split(",")]
        equip_filtered = [
            r for r in filtered
            if r.get("equipment", "").lower() in equip_list or r.get("equipment", "").lower() == "peso corporal"
        ]
        if equip_filtered:
            filtered = equip_filtered

    return json.dumps(filtered[:limit], ensure_ascii=False)


@tool
async def find_exercise_alternatives(
    pattern: str,
    level: str,
    exclude_name: str | None = None,
    equipment: str | None = None,
) -> str:
    """Busca alternativas de ejercicios para swaps en el borrador de rutina.

    Args:
        pattern: Patrón de movimiento del ejercicio a reemplazar
        level: Nivel del usuario
        exclude_name: Nombre del ejercicio a excluir (el que se quiere cambiar)
        equipment: Equipamiento disponible (opcional)

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

    # Filter by equipment if specified
    if equipment:
        equip_list = [e.strip().lower() for e in equipment.split(",")]
        equip_filtered = [
            r for r in filtered
            if r.get("equipment", "").lower() in equip_list or r.get("equipment", "").lower() == "peso corporal"
        ]
        if equip_filtered:
            filtered = equip_filtered

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

    # Create plan
    await supabase_insert(
        "users_plans",
        data={
            "plan_id": plan_id, "user_id": user_id, "template_id": template_id,
            "start_date": now.isoformat(), "goal": goal,
            "level": draft.get("level", ""), "status": "active",
            "mesocycle_number": 1, "week_schedule": week_schedule,
        },
        upsert=True,
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

                workout_rows.append({
                    "id": str(uuid.uuid4()),
                    "user_id": user_id,
                    "week": week,
                    "day_name": day_title,
                    "exercise_id": ex_id,
                    "sets": str(ex.get("sets", 3)),
                    "reps": ex.get("reps", "8-12"),
                    "rir": ex.get("rir", "1-2"),
                    "rest-seconds": ex.get("rest_seconds", ex.get("rest", 120)),
                    "tempo": ex.get("tempo", "2-0-1"),
                    "created_at": now.isoformat(),
                    "notes": "",
                    "exercise_order": ex.get("exercise_order", ex.get("order", e_idx + 1)),
                })

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
        select="plan_id,week_schedule,goal,level,mesocycle_number",
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

    # Get current week from schedule
    schedules = await supabase_query(
        "user_weekly_schedule",
        select="week",
        filters={"user_id": f"eq.{user_id}"},
        limit=1,
    )
    current_week = schedules[0]["week"] if schedules else 1

    return json.dumps({
        "days_per_week": len(days_sorted),
        "sessions": [{"day_number": d["day_number"], "title": d["title"]} for d in days_sorted],
        "current_week": current_week,
        "week_schedule": ws,
    }, ensure_ascii=False)


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

    # Get current week
    plans = await supabase_query(
        "users_plans",
        select="plan_id",
        filters={"user_id": f"eq.{user_id}", "status": "eq.active"},
        limit=1,
    )

    schedules = await supabase_query(
        "user_weekly_schedule",
        select="week",
        filters={"user_id": f"eq.{user_id}"},
        limit=1,
    )
    current_week = schedules[0]["week"] if schedules else 1

    now = datetime.now(timezone.utc)
    rows = []
    for s in sessions:
        rows.append({
            "user_id": user_id,
            "week": current_week,
            "week_day": s.get("week_day", ""),
            "session_name": s.get("session_name", ""),
            "planned_day": s.get("planned_day", ""),
            "Completed": False,
        })

    if rows:
        await supabase_bulk_insert("user_weekly_schedule", rows)

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
