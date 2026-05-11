# SUMMARY — Kairos Manual Test Plan Execution

**Fecha:** 2026-05-11
**Tester:** Camilo (`+57 350 626 7523`, `573506267523`)
**Revisión Kairos:** `kairos-agent-00059-stw` (Cloud Run `kairos-agent`)
**Entorno:** WhatsApp Business Development mode → Webhook directo a Kairos (reconectado en esta misma sesión, antes apuntaba a n8n cloud)

---

## 1. Métricas

| Métrica | Valor |
|---|---|
| Escenarios planeados | 20 |
| Escenarios ejecutados | **12** (7 antes del check-in + 5 priorizados después) |
| Escenarios saltados | 8 (priorizados de menor a mayor según consenso con usuario) |
| ✅ PASS | 6 (S04, S06, S07, S08, S13, S14, S16, S17 — 8 PASS, contando split) |
| ⚠️ PASS WITH ISSUES | 2 (S01, S05) |
| ❌ FAIL | 2 P0 (S02, S10) |
| Bugs únicos catalogados | **18** (2 P0 + 6 P1 + 7 P2 + 3 P3) |

---

## 2. Matriz de resultados

| ID | Escenario | Resultado | Issues clave |
|----|-----------|-----------|--------------|
| **S01** | New User GYM Happy Path | ⚠️ PASS w/ issues | KYC ultra-corto (13/19 campos NULL), volumen bajo para Avanzado, draft.status no se actualiza |
| **S02** | New User HOME Bodyweight only | ❌ **FAIL P0** | **Equipment filter ROTO — 67% de ejercicios usan máquinas pese a "solo peso corporal"** |
| ~~S03~~ | HOME Mancuernas+Bandas | — | Saltado (variación de S02, probablemente mismo P0) |
| **S04** | New User Health Status C | ✅ PASS | Health filter funciona (push_v=0); arrastra B-S02-002 (experiencia mal mapeada) |
| **S05** | Modificar Draft (swap) | ⚠️ PASS w/ issues | Alternativa minimal (Press Militar barra → Press militar mancuerna, mismo movimiento) |
| **S06** | View Today's Routine | ✅ PASS | Magic link redundante (#3 en sesión) |
| **S07** | Confirmar Workout | ✅ PASS | — |
| **S08** | Decline Workout | ✅ PASS | Bot ofreció reschedule proactivo; no creó pending_task |
| ~~S09~~ | Grace Period | — | Saltado |
| **S10** | Reschedule (cambiar días) | ❌ **FAIL P0** | **"cambiar mis días" gatilla renew_change_days → destruye 60 workouts + schedule + Completed=true de hoy** |
| ~~S11~~ | View Schedule | — | Skipped (cadena corrupta por S10) |
| ~~S12~~ | Magic Link | — | Skipped |
| **S13** | Mesocycle Maintain | ✅ PASS | Mesocycle++, workouts mantenidos, schedule limpiado |
| **S14** | Mesocycle Change Days (3→5) | ✅ PASS | week_schedule fb_3→ppl_5, draft nuevo generado |
| ~~S15~~ | Mesocycle Rotate | — | Saltado |
| **S16** | Mesocycle Early Attempt | ✅ PASS | Bot rechazó con explicación pedagógica |
| **S17** | Chat General Fitness | ✅ PASS | Respuesta científica + cálculo personalizado + no creó usuario |
| ~~S18~~ | Chitchat | — | Saltado |
| ~~S19~~ | Reactive Email | — | Saltado |
| ~~S20~~ | User sin plan | — | Saltado |

---

## 3. Bug Catalog Consolidado

### 3.1 P0 — Bloqueantes para producción (2)

| ID | Título | Reproducción | Causa raíz |
|---|---|---|---|
| **B-S02-001** | Equipment filter no aplica para HOME bodyweight users | S02 turn 1-3 | `get_exercises_for_draft` o `save_workout_plan` ignora `home_equipment='peso corporal'`. 10/15 ejercicios guardados requieren máquinas/mancuernas. |
| **B-S10-001** | Reschedule simple destruye datos del usuario | S10 turn 1 | Tool selection del LLM: "cambiar mis días" gatilla `renew_change_days` cuando debería ser `schedule_sessions`. Borra workouts, schedule, marca Completed. Adicionalmente, `renew_*` no valida que mesociclo esté completo. |

### 3.2 P1 — Mayores (6)

| ID | Título | Repro |
|---|---|---|
| **B-S01-001** | KYC ultra-corto deja 13/19 campos NULL (email, sex, age, weight, height, training_style, etc.) | S01 turn 3 |
| **B-S01-005** | Volumen insuficiente: 3 sets fijos para usuario Avanzado/Hipertrofia (debería 4-5) | S01 DB inspection |
| **B-S02-002** / **B-S04-002** | Enum mapping: "Intermedio, 1 año" → guarda "Menos de 6 meses / Principiante" (reproducido en S02 y S04) | S02 + S04 |
| **B-S02-003** | El bot describe ejercicios distintos a los que guarda (texto dice "flexiones, fondos" pero workouts tiene "chest press, prensa") | S02 turn 2 |
| **B-S10-002** | `renew_*` no verifica precondición "mesociclo completo" — puede ejecutarse en W1 | S10 |
| **B-S10-003** | Inconsistencia conversación↔DB: bot dijo "ya ajusté tu plan" pero schedule quedó en 0 | S10 |

### 3.3 P2 — Menores (7)

| ID | Título |
|---|---|
| **B-S01-002** | `draft_routines.status` no pasa a "approved" tras save_workout_plan (queda "pending") |
| **B-S01-008** | Asimetría de ejercicios entre días: FB A:4, FB B:5, FB C:6 |
| **B-S04-001** | Cuando push_v se excluye por health C, el ratio push_h:pull_h queda 3:1 (debería ser ~1:1) |
| **B-S04-003** | Reproducción de B-S01-001 (campos NULL) — confirma sistémico |
| **B-S05-001** | `find_exercise_alternatives` retorna alternativa minimal (mismo ejercicio, otro equipment) en lugar de movimiento distinto |
| **B-S05-002** | Modificar draft crea NUEVO draft (nuevo code) en lugar de actualizar el existente |

### 3.4 P3 — Polish (3)

| ID | Título |
|---|---|
| **B-S01-004** | Magic link duplicado en mismo flow (general + específico de sesión) |
| **B-S06-001** | Se crea nuevo magic link cada vez que se pregunta "qué me toca hoy" (acumula sin reusar) |
| Conversational | El bot a veces "imagina" detalles que no están en DB (e.g. describe "flexiones" pero guarda "chest press") |

---

## 4. Top 10 Por Mejorar (priorizado)

1. **Filtro estricto de equipment** en `get_exercises_for_draft` — bloquea fix P0 de S02
2. **Distinguir "cambiar días" vs "cambiar cuántos días"** en system prompt — bloquea fix P0 de S10
3. **Precondición de mesociclo completo** en todos los `renew_*` tools
4. **Expandir KYC mínimo** a: nombre completo, email, edad, sexo, peso, altura, training_style (en mínimo 1-2 turns extra)
5. **Fix enum mapping de training_experience** — "1 año" debe ir a "6 a 12 meses" o "1 a 3 años", no "Menos de 6 meses"
6. **Volumen para Avanzado/Hipertrofia** — `set_profiles` debería tener 4-5 sets para compounds avanzados
7. **Validación post-save** — leer back los nombres reales guardados y reportar esos al usuario (evitar B-S02-003)
8. **Balancear push_h ↔ pull_h** automáticamente cuando push_v se filtra por health
9. **Email proactivo** tras save_workout_plan — el bot no ofreció enviar email aunque está en el flow esperado
10. **Reusar magic_link reciente válido** en lugar de crear uno por cada interacción

---

## 5. Backlog Nice To Have (sin priorizar)

- Botones interactivos en WhatsApp para "Me gusta / Cambiar / Rechazar"
- Capturar PRs y feedback de sesión post-confirm ("¿qué tal estuvo?")
- Streak counter en respuestas de confirm
- Detección de patrón de declines ("Llevas 2 sábados sin entrenar, ¿quieres mover el día?")
- Stats de progreso al cerrar mesociclo (volumen total, mejoras)
- Comparativa entre mesociclos
- Soporte para enviar la rutina como PDF
- Foto del gym → detectar equipment con vision
- Whitelist de ejercicios bodyweight curada
- Detección de meseta y sugerencia automática de deload
- Override consciente para renovación prematura ("Escribe CONFIRMAR")
- Mostrar diff entre plan viejo y nuevo en draft preview

---

## 6. Hallazgos arquitectónicos

### Lo que funciona bien
- **Webhook reconnected** end-to-end (Meta → Kairos → Graph API → user)
- **PostgresSaver** mantiene estado entre turns correctamente
- **`get_mesocycle_status`** detecta W4 completa de forma robusta
- **`_apply_health_filter()`** funciona como modelo correcto para B-E codes
- **`renew_maintain` y `renew_change_days`** ejecutan correctamente cuando el LLM los selecciona bien
- **Filtro de noise** (mensaje "status updates", emojis, etc.) — implícitamente probado
- **Detección de tools por intención** funciona para chat general (no gatilla register_new_user con preguntas de nutrición)
- **Tono empático y profesional** consistentemente — buen producto a nivel UX conversacional

### Lo que NO funciona
- **Equipment filter** (P0)
- **Tool routing ambiguo** — el LLM elige `renew_*` ante cualquier mención de "cambiar días", aunque sea reschedule simple (P0)
- **Enum normalization para training_experience** (P1, reproducido 2 veces)
- **KYC superficial** — el bot termina con sólo 5 datos de los ~17 críticos (P1)
- **Volumen de entrenamiento** subóptimo para Avanzado (P1)

---

## 7. Recomendaciones de próximos pasos

### Cerrar antes de pasar a Live Mode en Meta

1. **🔴 P0 — Fixes obligatorios (1-2 días):**
   - Equipment filter strict en `get_exercises_for_draft` ([tools.py:467](langgraph-skeleton/cases/case6_unified_agent/tools.py:467) y `_normalize_equipment` línea 56)
   - Reschedule vs Renew distinction en `prompts.py` + guard en `renew_change_days` ([tools.py:1630](langgraph-skeleton/cases/case6_unified_agent/tools.py:1630))
   - Validación de mesociclo-completo antes de cualquier `renew_*`

2. **🟡 P1 — Calidad mínima (3-5 días):**
   - Extender KYC obligatorio (email, age, sex, weight, height, training_style) — 2 turnos más
   - Fix enum mapping para training_experience en `_normalize_enum` ([tools.py](langgraph-skeleton/cases/case6_unified_agent/tools.py))
   - Ajustar `set_profiles` para Avanzado/hipertrofia (4-5 sets compounds)
   - Email proactivo tras save_workout_plan
   - Validación post-save de equipment + nombres reales

3. **🟢 P2-P3 — Polish (después de fixes críticos):**
   - Reusar magic links válidos
   - Update `draft_routines.status='approved'` tras save_workout_plan
   - Balance push/pull para health C
   - Mejorar `find_exercise_alternatives` para sugerir movimientos distintos

### Para continuar testing (escenarios saltados)
- **S03** (HOME Mancuernas+Bandas) — repetir test de S02 con otro equipment para confirmar generalidad del fix
- **S09** (Grace Period) — único bug área no testeada que podría tener edge cases
- **S15** (Rotate Exercises) — equivalente a S13/S14 pero rota ejercicios, no estructura
- **S18** (Chitchat) — validar respuesta a off-topic
- **S19** (Reactive Email) — validar que `send_routine_email` falla bien y pide email reactivamente
- **S20** (User sin plan) — validar offer-to-generate

---

## 8. Archivos del testing run

| Archivo | Descripción |
|---|---|
| [SUMMARY.md](SUMMARY.md) | Este reporte |
| [S01-new-user-gym-happy-path.md](S01-new-user-gym-happy-path.md) | Onboarding GYM full happy path |
| [S02-home-bodyweight-only.md](S02-home-bodyweight-only.md) | ❌ P0: Equipment filter roto |
| [S04-health-status-c.md](S04-health-status-c.md) | Health restriction funciona |
| [S05-modificar-draft.md](S05-modificar-draft.md) | Swap minimal en alternativas |
| [S06-view-todays-routine.md](S06-view-todays-routine.md) | View routine OK |
| [S07-confirm-workout.md](S07-confirm-workout.md) | Confirm OK |
| [S08-decline-workout.md](S08-decline-workout.md) | Decline OK + reschedule proactivo |
| [S10-reschedule.md](S10-reschedule.md) | ❌ P0: Reschedule destruye plan |
| [S13-mesocycle-maintain.md](S13-mesocycle-maintain.md) | Renew_maintain OK |
| [S14-mesocycle-change-days.md](S14-mesocycle-change-days.md) | Renew_change_days OK (explícito) |
| [S16-mesocycle-early-attempt.md](S16-mesocycle-early-attempt.md) | Rechazo prematuro OK |
| [S17-chat-general-fitness.md](S17-chat-general-fitness.md) | Chat general OK |

---

## 9. Mensaje al equipo

**TL;DR:** Kairos está **funcionalmente vivo** end-to-end después de reconectar el webhook hoy. El happy path GYM funciona. PERO hay 2 bugs P0 que **destruyen datos del usuario** (S10) o **producen output inejecutable** (S02) — ambos son bloqueantes para Live Mode en Meta. Los fixes son acotados (1-2 días) y no requieren refactor mayor: el filtro de equipment y un guardrail en el system prompt para distinguir reschedule vs renewal. El resto de los hallazgos son de calidad (KYC más completo, volumen de Avanzado correcto, polish) que pueden trabajarse en paralelo.

El **modelo Gemini 3 Flash + 18 tools + PostgresSaver responde bien**. La arquitectura de Case 6 es sólida. Los problemas están en:
- Lógica de filtros específicos (equipment)
- System prompt instructions para tool selection
- Validaciones de precondición en tools peligrosos

**Acción inmediata sugerida:** Crear 2 tickets en el backlog:
1. `[P0] Equipment filter strict for HOME bodyweight users` — bloquea uso de HOME training
2. `[P0] Distinguish reschedule from mesocycle renewal in system prompt` — bloquea reschedule diario

Y antes de seguir construyendo features nuevos (Google Calendar events, ProcessUserPreferences completo), **cerrar estos 2 P0**.
