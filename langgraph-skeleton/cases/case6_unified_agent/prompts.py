"""System prompt and context formatting for Kairos unified agent."""

from cases.case6_unified_agent.state import UserContext


KAIROS_SYSTEM_PROMPT = """Eres Kairos, entrenador personal virtual de Kairos Personal Trainer.
Amigable, motivador, experto en fitness. Respondes en español colombiano.

## CONTEXTO DEL USUARIO
{user_context_formatted}

## TUS CAPACIDADES (herramientas disponibles)
1. Ver rutina del día → get_todays_routine
2. Confirmar entrenamiento completado → confirm_workout_completion
3. Declinar entrenamiento → decline_workout
4. Enlace al Workout Tracker web → create_magic_link
5. Info del plan para agendar → get_schedule_info + schedule_sessions
6. Estado del mesociclo → get_mesocycle_status
7. Buscar ejercicios para rutina → get_day_requirements, get_exercises_for_draft
8. Buscar alternativas de ejercicios → find_exercise_alternatives
9. Guardar plan de entrenamiento → save_workout_plan
10. Chat general de fitness → responde directo (sin tool)

## REGLAS DE COMPORTAMIENTO
- Si hay TAREA PENDIENTE (CONFIRMAR_RUTINA), SIEMPRE pregunta primero si completó su rutina antes de responder cualquier otra cosa.
- Si tiene sesión HOY sin completar, priorízala sobre sesiones perdidas. Solo ofrece sesiones perdidas cuando NO hay sesión hoy.
- Si NO tiene horario programado, sugiere programar sesiones.
- Si el mesociclo está completado (all_w4_completed), ofrece renovación proactivamente.
- Sé breve: máximo 3-4 oraciones (es WhatsApp).
- NUNCA inventes datos de rutinas — siempre consulta herramientas.
- SIEMPRE usa el user_id (UUID) del contexto al llamar herramientas. NUNCA uses el nombre del usuario como user_id.
- Cuando necesites ejecutar una acción (confirmar rutina, crear link, etc.), LLAMA la herramienta directamente. NUNCA muestres el nombre de la herramienta ni código al usuario. El usuario no debe ver nombres de funciones.
- Si no estás seguro de qué quiere el usuario, pregunta.
- Usa el nombre del usuario cuando lo tengas.
"""


def format_user_context(ctx: UserContext) -> str:
    """Format UserContext into a readable string for the system prompt."""
    lines = []

    # Identity
    name = ctx.get("full_name") or "Usuario nuevo"
    user_id = ctx.get("user_id") or "DESCONOCIDO"
    lines.append(f"Nombre: {name} | user_id: {user_id}")

    # Plan info
    plan = ctx.get("plan")
    if plan:
        week_schedule = plan.get("week_schedule", "")
        goal = plan.get("goal", "")
        level = plan.get("level", "")
        meso = plan.get("mesocycle_number", 1)
        lines.append(f"Plan: {goal} | {level} | Schedule: {week_schedule} | Mesociclo #{meso}")
    else:
        lines.append("Plan: Ninguno (sin rutina generada)")

    # Today's sessions
    todays = ctx.get("todays_sessions", [])
    if todays:
        session_strs = []
        for s in todays:
            status = "COMPLETADA" if s.get("Completed") else "NO completada"
            session_strs.append(f"{s.get('session_name', '?')} (Semana {s.get('week', '?')}, {status})")
        lines.append(f"Horario hoy: {', '.join(session_strs)}")
    else:
        lines.append("Horario hoy: Ninguno (día de descanso)")

    # Missed sessions
    missed = ctx.get("missed_sessions", [])
    if missed:
        missed_strs = []
        for m in missed:
            missed_strs.append(f"{m.get('session_name', '?')} ({m.get('planned_day', '?')})")
        lines.append(f"Sesiones pendientes sin completar: {', '.join(missed_strs)}")
    else:
        lines.append("Sesiones pendientes: Ninguna")

    # Next scheduled
    next_s = ctx.get("next_scheduled_session")
    if next_s:
        lines.append(f"Próxima sesión: {next_s.get('session_name', '?')} ({next_s.get('planned_day', '?')})")

    # Pending tasks
    tasks = ctx.get("pending_tasks", [])
    if tasks:
        task_strs = []
        for t in tasks:
            task_strs.append(f"Confirmar {t.get('session_name', '?')} ({t.get('task_type', '?')})")
        lines.append(f"⚠️ TAREAS PENDIENTES: {', '.join(task_strs)}")
    else:
        lines.append("Tareas pendientes: Ninguna")

    # W4 completion
    if ctx.get("all_w4_completed"):
        lines.append("→ Mesociclo COMPLETADO — listo para renovación")

    # Flags
    if not ctx.get("has_schedule") and plan:
        lines.append("→ Tiene plan pero NO tiene días agendados")

    if ctx.get("kyc_complete") and not plan:
        lines.append("→ Perfil completo pero SIN plan de entrenamiento")

    return "\n".join(lines)
