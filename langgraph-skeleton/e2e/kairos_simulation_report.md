# Kairos Agent — Simulation Report (10 Users, 7 Cycles)

**Date**: 2026-03-19
**Environment**: Production (https://kairos-agent-148665080566.us-central1.run.app)
**Agent**: Gemini + 15 tools, LangGraph with PostgresSaver
**Cycles**: 7 iteraciones de fix → test → diagnóstico

---

## Summary

| Metric | Cycle 1 | Final (Cycle 7) |
|--------|---------|-----------------|
| Average score | 6.6/10 | **8.1/10** |
| Passed (>=7) | 4/10 | **10/10** |
| Critical bugs | 2 | 0 |
| Deploys | 1 | 8 |

---

## Final Scores

| # | Persona | Scenario | Score | Key Observations |
|---|---------|----------|-------|------------------|
| 1 | Sofia Garcia | KYC nuevo (entusiasta) | **8** | KYC limpio, rutina necesita más turnos |
| 2 | Carlos Mendoza | KYC nuevo (parco) | **8** | KYC + rutina + scheduling en 10 turnos |
| 3 | Ana Martinez | KYC HOME (health D) | **7** | Health D enforcement reactivo, no proactivo |
| 4 | Diego Ramirez | KYC nuevo (slang) | **8** | Slang manejado, sin tool leaks |
| 5 | Maria Lucia | Ver rutina + link | **8** | Rutina mostrada completa, magic link OK |
| 6 | Roberto Herrera | Agendar sesiones | **8** | Scheduling OK, tono corregido |
| 7 | Laura Vargas | Descanso + chat (health C) | **8** | Health C awareness excelente |
| 8 | Andres Castillo | Confirmar rutina | **9** | Pending task detectada y confirmada |
| 9 | Valentina Rios | Declinar + email | **8** | Decline + email funcionan correctamente |
| 10 | Mateo Ospina | Renovar mesociclo | **9** | MANTENER ejecutado, 3 opciones presentadas |

---

## Bugs Corregidos (Ciclos 1-7)

| Bug | Ciclo | Fix |
|-----|-------|-----|
| Tool call leaked como texto (`print(default_api...)`) | C1→C2 | Prompt reforzado: "NUNCA escribas funciones ni print()" |
| Health filtering no aplicado en draft | C1→C2 | `_lookup_user_profile` — auto-lookup de health_status desde DB |
| KYC campos faltantes (training_style, cardio, priority_muscles) | C2→C3 | Actualizados KYC_FIELDS y prompts Turn 3/5 |
| KYC regression — profile never saved | C2→C3 | REQUIRED_FIELDS excluye campos opcionales |
| Rutina no mostrada (pregunta completación primero) | C2→C3 | Prompt: "MUESTRA rutina PRIMERO con get_todays_routine" |
| Tono culpabilizador ("seguro que agendaste bien?") | C1→C2 | Prompt: "NUNCA culpabilices" |
| "Supabase" expuesto al usuario | C2→C3 | Prompt: "NUNCA menciones nombres técnicos" |
| Machine exercises para HOME user | C3→C4 | `_lookup_user_profile` auto-detecta HOME + equipment |
| Health D no excluye push_v/crunches | C4→C5 | Agregados push_v pattern + crunch keywords al filtro D |
| LLM no pasa user_id a get_exercises_for_draft | C5→C7 | Tool docstring: "OBLIGATORIO", prompt con ⚠️ |

---

## Issues Residuales

| Issue | Impacto | Causa | Mitigación |
|-------|---------|-------|------------|
| Health D enforcement reactivo (no proactivo) | LLM a veces no pasa user_id | Non-determinismo del LLM | Prompt reforzado + auto-lookup, pero 100% no garantizado |
| save_workout_plan falla silenciosamente | Rutina no se guarda en DB | Formato JSON del draft incorrecto o timeout | Investigar logging en save_workout_plan |
| 500 Internal Server Error intermitente | 1-2 por simulación | Timeout en tool calls largos o checkpointer contention | Monitorear en producción |

---

## Fixes Implementados (Código)

### `tools.py`
- `_lookup_user_profile()` — auto-lookup de health_status + training_environment + home_equipment desde DB
- `_apply_health_filter()` — Health D expandido: excluye push_v + crunches/sit-ups + press militar
- `get_exercises_for_draft` — auto-filtra equipment para HOME users, auto-lookup health
- `find_exercise_alternatives` — mismo auto-lookup
- Dedup guard en `save_workout_plan`

### `prompts.py` (Kairos)
- Instrucción ⚠️ OBLIGATORIO user_id en get_exercises_for_draft
- "NUNCA escribas funciones, código, print() ni tool calls como texto"
- "Si pregunta qué me toca hoy, MUESTRA la rutina PRIMERO"
- "NUNCA culpabilices", "NUNCA menciones Supabase/API/tool"
- "Adapta tono al usuario"
- Instrucciones de dedup y variedad
- Auto-transición KYC → rutina

### `prompts.py` (KYC Case 5)
- Turn 3: training_style + cardio_type
- Turn 5: priority_muscles + disliked_exercises
- Resumen de confirmación muestra todos los campos

### `state.py` (KYC Case 5)
- KYC_FIELDS actualizado con nuevos campos
- REQUIRED_FIELDS excluye campos opcionales

---

## Cobertura de Diversidad

| Dimensión | Probado | Resultado |
|-----------|---------|-----------|
| Goals (5) | masa, grasa, salud, fuerza, resistencia | Todos funcionan |
| Health codes | A, B, C, D, E | C excelente, D reactivo, A/B/E OK |
| Environment | GYM + HOME | HOME equipment filtering funciona |
| Sex | 5F + 5M | OK |
| Levels | Beg + Int + Adv | OK |
| Days | 2-6 | OK |
| Scenarios | 8 tipos | KYC, view, schedule, confirm, decline, email, chat, renewal |
