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


@tool
async def save_workout_plan(user_id: str, draft_json: str) -> str:
    """Guarda el plan de entrenamiento aprobado por el usuario.

    Crea users_plans + bulk insert de workouts para 4 semanas.

    Args:
        user_id: UUID del usuario
        draft_json: JSON string de DraftRoutine con la estructura:
            {week_schedule, goal, level, days: [{day_number, title, exercises: [{exercise_id, sets, reps, rir, rest_seconds, exercise_order, tempo}]}]}

    Returns:
        Confirmación con plan_id y cantidad de workouts creados
    """
    draft = json.loads(draft_json)

    # Generate plan_id using timestamp
    now = datetime.now(timezone.utc)
    plan_id = f"plan_{int(now.timestamp()):x}"

    # Determine template_id from week_schedule + goal + level
    goal_map = {
        "Ganar masa muscular": "hyp",
        "Mejorar fuerza": "str",
        "Bajar grasa": "cut",
        "Mejorar resistencia": "end",
        "Salud general / recomposición corporal": "rec",
    }
    level_map = {"Principiante": "beg", "Intermedio": "int", "Avanzado": "adv"}
    goal_code = goal_map.get(draft.get("goal", ""), "hyp")
    level_code = level_map.get(draft.get("level", ""), "int")
    template_id = f"tpl_{draft['week_schedule']}_{goal_code}_{level_code}"

    # Create plan
    await supabase_insert(
        "users_plans",
        data={
            "plan_id": plan_id,
            "user_id": user_id,
            "template_id": template_id,
            "start_date": now.isoformat(),
            "goal": draft.get("goal", ""),
            "level": draft.get("level", ""),
            "status": "active",
            "mesocycle_number": 1,
            "week_schedule": draft.get("week_schedule", ""),
        },
    )

    # Build workout rows for 4 weeks
    workout_rows = []
    for week in range(1, 5):
        for day in draft.get("days", []):
            for ex in day.get("exercises", []):
                workout_rows.append({
                    "user_id": user_id,
                    "week": week,
                    "day_name": day["title"],
                    "exercise_id": ex["exercise_id"],
                    "sets": str(ex.get("sets", 3)),
                    "reps": ex.get("reps", "8-10"),
                    "rir": ex.get("rir", "1-2"),
                    "rest-seconds": ex.get("rest_seconds", 120),
                    "tempo": ex.get("tempo", "2-0-1"),
                    "created_at": now.isoformat(),
                    "notes": "",
                    "exercise_order": ex.get("exercise_order", 1),
                })

    if workout_rows:
        await supabase_bulk_insert("workouts", workout_rows)

    return json.dumps({
        "success": True,
        "plan_id": plan_id,
        "workouts_created": len(workout_rows),
        "weeks": 4,
        "days_per_week": len(draft.get("days", [])),
    })


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
