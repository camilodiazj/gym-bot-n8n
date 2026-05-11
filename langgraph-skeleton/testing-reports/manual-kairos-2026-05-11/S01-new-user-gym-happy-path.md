# S01 — New User GYM Happy Path

**Fecha:** 2026-05-11
**Tester:** Camilo
**Phone:** +57 350 626 7523 (`573506267523`)
**Revisión Kairos:** kairos-agent-00059-stw
**Resultado:** ⚠️ PASS WITH ISSUES

## Pre-condiciones
- [x] Teardown ejecutado (users=0, profile=0, plans=0, workouts=0, schedule=0, checkpoints=0)
- [x] Fixture aplicado: Ninguno

## Conversación (turn-by-turn)

| # | Tú enviaste | Bot respondió (resumen) | Latencia | Tools fired | Notas |
|---|-------------|--------------------------|----------|-------------|-------|
| 1 | `Hola, quiero empezar a entrenar` | "¡Qué nota, Camilo! ... 3 preguntas: objetivo, experiencia, lugar" | ~15s | (none) | Saludó por nombre desde profile.name de WhatsApp |
| 2 | `Quiero ganar masa muscular. Llevo más de 3 años entrenando. Voy a un gimnasio bien equipado.` | "¡Excelente, Camilo! 2 datos más: días disponibles + lesiones" | ~17s | (none) | |
| 3 | `Tengo 3 días disponibles, ninguna lesión, todo bien` | "¡Listo, Camilo! Ya te registré y armé tu rutina de 3 días. Link al draft: `/draft?c=02066f`" | ~38s | `register_new_user`, `get_day_requirements`, `get_exercises_for_draft` ×N, `save_draft_preview` | KYC ultra-corto (solo 5 preguntas). Muchos campos NULL. |
| 4 | `Me gusta, guárdala` | "Plan guardado. ¿Qué días esta semana? Link tracker: `/w?c=07e5ff`" | ~20s | `save_workout_plan`, `create_magic_link` (general) | Plan se guardó en `users_plans` + 60 workouts. Magic link general. |
| 5 | `Lunes, miércoles y viernes. Hoy mismo arranco.` | "Agendado L/Mi/V. Como hoy arrancas, aquí tu Full Body A. Link específico: `/w?c=07e650`" | ~18s | `schedule_sessions`, `get_todays_routine`, `create_magic_link` (Full Body A wk1) | 3 sessions creadas. Bot mostró rutina hoy proactivamente (cubre S06 implícito). 2do magic link específico. |

## Verificación DB

```sql
SELECT 'users' t, COUNT(*) FROM users WHERE full_phone_number=573506267523
UNION ALL SELECT 'profile', COUNT(*) FROM users_gym_profile WHERE whatsapp_id=573506267523
UNION ALL SELECT 'plan', COUNT(*) FROM users_plans WHERE user_id='6a98bdce-5a09-40ba-885d-fd17998768bc'
UNION ALL SELECT 'workouts', COUNT(*) FROM workouts WHERE user_id='6a98bdce-5a09-40ba-885d-fd17998768bc'
UNION ALL SELECT 'schedule', COUNT(*) FROM user_weekly_schedule WHERE user_id='6a98bdce-5a09-40ba-885d-fd17998768bc'
UNION ALL SELECT 'magic_links', COUNT(*) FROM magic_links WHERE user_id='6a98bdce-5a09-40ba-885d-fd17998768bc'
UNION ALL SELECT 'draft', COUNT(*) FROM draft_routines WHERE user_id='6a98bdce-5a09-40ba-885d-fd17998768bc';
```

| Métrica | Esperado | Actual | OK |
|---|---|---|---|
| users | 1 | 1 | ✅ |
| profile | 1 | 1 | ✅ |
| plan | 1 | 1 | ✅ |
| workouts | ≈60 (3 d × 5 ex × 4 sem) | 60 | ✅ |
| schedule | 3 | 3 | ✅ |
| magic_links | ≥1 | 2 | ⚠️ ver B-S01-004 |
| draft_routines | 1 (pending O approved) | 1 (pending) | ⚠️ ver B-S01-002 |

### Perfil guardado en `users_gym_profile`

```json
{
  "primary_goal": "Ganar masa muscular",        ← OK
  "training_experience": "Más de 3 años",        ← OK
  "fitness_level": "Avanzado",                   ← OK (inferido)
  "health_status": "A",                          ← OK
  "days_available": 3,                           ← OK
  "training_environment": "GYM",                 ← OK

  "age": null,                                   ← ❌ NO preguntó
  "biological_sex": null,                        ← ❌ NO preguntó (crítico)
  "height_cm": null,                             ← ❌ NO preguntó
  "weight_kg": null,                             ← ❌ NO preguntó
  "email": null,                                 ← ❌ NO preguntó (rompe send_routine_email)
  "secondary_goal": null,
  "current_frequency": null,
  "session_duration_mins": null,
  "preferred_schedule": null,
  "training_style": null,                        ← ⚠️ NO preguntó (pesas libres vs máquinas)
  "priority_muscles": null,
  "disliked_exercises": null,
  "cardio_type": null,
  "cardio_frequency": null
}
```

### Distribución de workouts (semana 1)

| Día | Ejercicios | Detalle |
|---|---|---|
| Full Body A | **4** | Sentadilla Hack, Press inclinado mancuerna, Remo en T, Plancha |
| Full Body B | **5** | Peso muerto RDL, Jalón abierto, Press Militar, Abs general, Curl banca inclinada |
| Full Body C | **6** | Sentadilla búlgara, Hip Thrust, Máquina pecho, Jalón triángulo, Levantamiento activo banca, Máquina bíceps |

Total wk1: 15 ejercicios. Reps 8-10 compound / 12-15 isolation+core. **3 sets fijos**. RIR 1-2.

### Logs Cloud Run relevantes

```
16:38:08 POST /webhook 200 (turn 1)
16:38:33 POST /webhook 200 (status)
16:40:05 POST /webhook 200 (turn 4 - "Me gusta")
16:40:36 [DEBUG create_magic_link] session_name='' week=0    ← magic link general
16:41:47 POST /webhook 200 (turn 5 - "Lunes mi vi")
16:41:56 [DEBUG create_magic_link] session_name='Full Body A' week=1  ← magic link específico
```

---

## Bugs

### B-S01-001 — KYC ultra-corto: campos críticos quedan NULL **[P1]**
- **Repro:** Turn 1-3
- **Esperado:** Bot pregunta al menos: nombre real, edad, sexo biológico, peso, altura, email, training_style. Estos son necesarios para personalización (sex adaptation per CLAUDE.md), `send_routine_email` y métricas de progreso.
- **Actual:** Bot terminó el KYC con sólo 5 preguntas (goal, experience, environment, days, lesiones). Dejó 13 campos NULL en `users_gym_profile`.
- **Evidencia:** Query del perfil mostró 13/19 columnas NULL.
- **Hipótesis de causa:** El system prompt en `prompts.py` permite que el LLM decida "ya tengo suficiente para crear la rutina". Sugerir agregar lista de campos mínimos obligatorios.
- **Impacto:** Bloquea `send_routine_email` (S19), reduce calidad de personalización, sin training_style el algoritmo de selección de ejercicios pierde precisión.

### B-S01-002 — `draft_routines.status` no se actualiza a "approved" tras `save_workout_plan` **[P2]**
- **Repro:** Turn 4 ("Me gusta, guárdala")
- **Esperado:** Después de save_workout_plan, el draft asociado al user_id debería pasar de `pending` → `approved`.
- **Actual:** `draft_routines.status` sigue siendo `pending`. Expira en 24h y se queda como "fantasma" en la tabla.
- **Evidencia:** Query `SELECT status FROM draft_routines WHERE user_id='...'` retornó `pending` después del save.
- **Hipótesis de causa:** `save_workout_plan` en `tools.py` no marca el draft. Buscar `UPDATE draft_routines SET status='approved'`. Probablemente falta esa línea.

### B-S01-005 — Volumen insuficiente para usuario AVANZADO + masa muscular **[P1]**
- **Repro:** Verificación de workouts
- **Esperado:** Para usuario Avanzado/Hipertrofia: 10-15 sets por grupo muscular por semana. Con 3 días Full Body × ~5 ejercicios × 3 sets = ~45 sets totales, ~9 sets/grupo. Apenas en el umbral mínimo.
- **Actual:** Solo 3 sets fijos por ejercicio para todos los ejercicios. Avanzados típicamente trabajan 4-5 sets en compounds.
- **Evidencia:** `SELECT sets FROM workouts` → todos `'3'`.
- **Hipótesis de causa:** `set_profiles` table puede tener configurado 3 sets para Avanzado/hipertrofia. Revisar `SELECT * FROM set_profiles WHERE goal='Ganar masa muscular' AND level='Avanzado'`.

### B-S01-008 — Asimetría de ejercicios entre días (4/5/6) **[P2]**
- **Repro:** Verificación de workouts wk1
- **Esperado:** Distribución balanceada entre días Full Body (idealmente 5-6 ejercicios cada uno).
- **Actual:** FB A: 4 ejercicios, FB B: 5, FB C: 6.
- **Hipótesis:** La lógica de generación en `get_exercises_for_draft` o `save_workout_plan` falla en algunos patrones para FB A, o el filtro de patterns difiere por día.

---

## Por Mejorar

- [ ] **Email proactivo:** El bot no ofreció enviar la rutina por correo después de save_workout_plan, aunque está en el flow esperado per `onboarding_conversation_example.md`. Sugerir que tras guardar el plan, ofrezca email automáticamente.
- [ ] **Two magic links creados en un mismo flow:** Uno general (post save) + uno específico para "Full Body A wk1" (post schedule + view today). UX puede confundir. Considerar emitir solo el específico cuando ya hay sesión hoy, o consolidar a uno solo.
- [ ] **Solo `display_name` de WhatsApp como nombre:** El bot usó "Camilo" como `full_name`. No preguntó apellido. Para emails formales, magic link cards, etc., conviene nombre completo.
- [ ] **No menciona contraste compound/isolation/core:** El usuario verá la rutina pero no entiende el ordenamiento. Sugerir agrupar visualmente: "Compound (pesado)", "Core", "Aislados (accesorios)".
- [ ] **Reps en formato `8-10` con guion ASCII** — verificar que el frontend los renderice bien.
- [ ] **No preguntó duración de sesión** (`session_duration_mins`). Es un input crítico que afecta el volumen total (per CLAUDE.md, ProcessUserPreferences usa `volume_modifier` derivado de esto).
- [ ] **No preguntó música prioritaria/disliked:** Si el usuario odia hacer cardio en máquinas, el bot no lo sabrá.

## Nice To Have

- [ ] **Preview del Workout Tracker desde el draft preview:** Que el usuario pueda revisar todos los ejercicios con video antes de aprobar, no solo después.
- [ ] **Soporte para "rutina como PDF":** Algunos usuarios quieren guardar la rutina como PDF para llevarla impresa al gym.
- [ ] **Detección automática de día de la semana actual:** En lugar de "Lunes, miércoles y viernes" textual, podría aceptar `hoy` + 2 inferidos para los próximos.
- [ ] **Calendario push:** Per migration_status.md, falta `Google Calendar events` en `schedule_sessions`. Cuando esto se implemente, el bot debería ofrecer adjuntar el calendar event al confirmar el schedule.
- [ ] **Botones interactivos** para "Me gusta / Cambiar / Rechazar" tras el draft, en lugar de texto libre. Reduce ambigüedad.
- [ ] **Stats de progreso mensual:** Al cerrar el mesociclo (S13-S15), mostrar resumen "Completaste X de 12 sesiones, +Y kg promedio en compounds".

---

## Estado final de la DB tras S01

| Tabla | Filas Camilo |
|---|---|
| users | 1 |
| users_gym_profile | 1 (13 campos NULL) |
| users_plans | 1 (mesocycle=1, status=active, week_schedule=fb_3) |
| workouts | 60 (3 días × 4 semanas × 5±1 ejercicios) |
| user_weekly_schedule | 3 (L=FB_A, Mi=FB_B, V=FB_C; ninguna completed) |
| magic_links | 2 (general + Full Body A wk1) |
| draft_routines | 1 (status=pending) |
| checkpoints | 86+ (varias entries de la conversación) |

**Conclusión:** Happy path **funcional end-to-end**, pero la calidad del onboarding y de la rutina generada está significativamente por debajo de lo que un usuario avanzado esperaría. Bloqueante para producción: B-S01-001 (perfil incompleto). Prioritarios: B-S01-005 (volumen bajo), B-S01-008 (asimetría).
