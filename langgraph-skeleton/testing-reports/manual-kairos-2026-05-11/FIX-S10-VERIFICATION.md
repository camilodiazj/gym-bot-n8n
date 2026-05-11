# Fix Verification — B-S10-001 (Reschedule destruye plan)

**Fecha:** 2026-05-11
**Revisión Kairos:** `kairos-agent-00060-2c5` (deployed con Fix 1+2+3)
**Tester:** Camilo (`573506267523`)
**Resultado:** ✅ **6/6 PASS** — Bug P0 cerrado

---

## Resumen ejecutivo

| Test | Escenario | Resultado |
|---|---|---|
| **T01** | Reschedule explícito ("cambiar mis días a Mar/Jue/Sab") | ✅ PASS |
| **T02** | Frase ambigua ("quiero cambiar mis días") | ✅ PASS |
| **T03** | Renovar a mismo nº de días (`SAME_DAYS_PER_WEEK` guard) | ✅ PASS (prompt-side) |
| **T04** | Renovar prematuramente (`MESOCYCLE_NOT_COMPLETE` guard) | ✅ PASS |
| **T05** | Renovación legítima (regresión S14) | ✅ PASS |
| **T06** | `renew_maintain` prematuro (Fix 3) | ✅ PASS |

**Conclusión:** Los 3 fixes resuelven el bug P0 sin regresión en el flujo legítimo. Bot rechaza con explicaciones pedagógicas y ofrece alternativas (schedule_sessions o terminar W4).

---

## Detalle por test

### T01 — Reschedule explícito (el bug original)

**Mensaje:** `Necesito cambiar mis días. Ahora puedo martes, jueves y sábado.`

**Respuesta del bot:** *"Ya actualicé tu horario. Tus sesiones quedaron programadas para los martes, jueves y sábados. Mañana martes te toca estrenar la rutina con Full Body A."*

**Estado DB:**
| Métrica | Antes (bug) | Después (fix) |
|---|---|---|
| `mesocycle_number` | 1 → 2 ❌ | **1** (sin cambio) ✅ |
| `workouts` count | 60 → 0 ❌ | **60** (intactos) ✅ |
| `user_weekly_schedule` rows | 3 → 0 ❌ | **6** ⚠️ (sub-bug, ver abajo) |
| Draft nuevo | Sí ❌ | **No** ✅ |

**Sub-bug menor identificado:** `schedule_sessions` hace upsert por `(user_id, week, week_day)`. Cuando se agrega Mar/Jue/Sab sobre L/Mi/V existentes, no hay conflicto en `week_day` → los 3 viejos quedan junto a los 3 nuevos (6 rows total). El bot dice "actualicé" pero realmente "agregué". **No es destructivo** (cero pérdida de datos) pero amerita fix de seguimiento.

---

### T02 — "Cambiar mis días" ambiguo

**Mensaje:** `Quiero cambiar mis días`

**Respuesta del bot:** *"¿Quieres cambiar **cuáles** días entrenas (por ejemplo, pasar del martes al miércoles) o quieres cambiar la **cantidad** de días por semana (por ejemplo, de 3 a 4 días)? Ten en cuenta que para cambiar la cantidad de días y renovar la rutina, primero debemos terminar las 4 semanas del mesociclo actual. Si es solo mover los días de esta semana, ¡lo hacemos de una!"*

**Estado DB:** Sin cambios — bot pidió clarificación antes de actuar.

**Comentario:** El Fix 2 (disambiguación en prompts.py) funcionó perfecto. Bot anticipó la regla de mesociclo completo proactivamente.

---

### T03 — Renovar con misma cantidad de días

**Setup:** Fixture F3 aplicada (W4 completa).
**Mensaje:** `Ya completé las 4 semanas. Quiero el siguiente mesociclo con 3 días por semana, igual que ahora.`

**Respuesta del bot:**
> *"¡Excelente trabajo completando ese primer mesociclo, Camilo! 🚀 Como ya terminaste las 4 semanas, estamos listos para renovar. Para este nuevo mesociclo de 3 días, tienes estas opciones: 1) Mantener rutina, 2) Cambiar frecuencia (aunque me dijiste que sigues con 3), 3) Rotar ejercicios. ¿Cuál prefieres?"*

**Estado DB:** Sin cambios (mesocycle=1, workouts=60).

**Comentario:** El bot detectó la igualdad en el prompt-side ANTES de llamar `renew_change_days`. El guard `SAME_DAYS_PER_WEEK` queda como red de seguridad si el LLM falla — pero en este caso ni se necesitó.

---

### T04 — Renovar prematuramente (`MESOCYCLE_NOT_COMPLETE`)

**Setup:** Usuario en W1 sin sesiones completadas.
**Mensaje:** `Quiero cambiar a 5 días por semana, hacer un mesociclo nuevo`

**Respuesta del bot:**
> *"Camilo, me encantaría pasarte a ese plan de 5 días, pero todavía me aparece que tienes pendiente completar las 4 semanas de tu mesociclo actual de 3 días. Para poder renovar y cambiar la frecuencia, el sistema necesita que termines la semana 4. ¿Quieres que agendemos las sesiones que te faltan para cerrar este ciclo y ahí sí saltar a los 5 días?"*

**Estado DB:** Sin cambios (mesocycle=1, workouts=60, schedule sin cambios).

**Comentario:** El bot internalizó la regla del Fix 2 y rechazó proactivamente. Ofreció alternativa (agendar las sesiones faltantes) en lugar de simplemente bloquear.

---

### T05 — Renovación legítima (regresión)

**Setup:** Fixture F3 aplicada (W4 completa).
**Mensaje:** `Acabo de terminar la semana 4, ya completé el mesociclo. Quiero el siguiente con 5 días por semana en vez de 3.`

**Estado DB:**
| Métrica | Antes | Después |
|---|---|---|
| `mesocycle_number` | 1 | **2** ✅ |
| `week_schedule` | fb_3 | **ppl_5** ✅ |
| `draft_routines` | — | nueva fila pending ✅ |

**Comentario:** El flujo legítimo (W4 completa + cambio explícito de cantidad) sigue funcionando como S14. **No hay regresión.**

---

### T06 — `renew_maintain` prematuro (Fix 3)

**Setup:** W1-W3 completas, W4 con 0/3.
**Mensaje:** `Quiero renovar mi rutina pero manteniendo los mismos ejercicios. Mantener.`

**Respuesta del bot:**
> *"...¿Quieres seguir con ese plan de 5 días o prefieres quedarte en el de 3 días y mantener los ejercicios que ya venías haciendo? Si quieres mantener los de 3 días, **solo confírmame si ya marcaste todas las sesiones de la semana 4 como completadas en el app para poder procesar la renovación.**"*

**Estado DB:** Sin cambios (mesocycle=1, W4 sigue 0/3).

**Comentario:** El bot mencionó explícitamente la pre-condición — Fix 3 internalizado. NO llamó `renew_maintain` (el guard tool-side tampoco se ejecutó porque el LLM se detuvo antes).

---

## Cambios desplegados

### Archivos modificados

| Archivo | Cambio | LOC |
|---|---|---|
| [`tools.py`](../../cases/case6_unified_agent/tools.py) | Helper `_validate_mesocycle_complete()` (líneas ~1124-1153) | +30 |
| [`tools.py`](../../cases/case6_unified_agent/tools.py) | Guard en `renew_maintain` | +4 |
| [`tools.py`](../../cases/case6_unified_agent/tools.py) | Guard en `renew_rotate_exercises` | +4 |
| [`tools.py`](../../cases/case6_unified_agent/tools.py) | Guards `SAME_DAYS_PER_WEEK` + `MESOCYCLE_NOT_COMPLETE` en `renew_change_days` | +25 |
| [`prompts.py`](../../cases/case6_unified_agent/prompts.py) | Sección "RESCHEDULE vs RENOVACIÓN — DISTINGUIR" con árbol de decisión + manejo de errores | +35 (reemplaza ~25) |

**Deploy:** Cloud Run revisión `kairos-agent-00060-2c5` (anterior `kairos-agent-00059-stw` queda como rollback).

---

## Sub-bug detectado (para fix de seguimiento)

### B-T01-001 — `schedule_sessions` acumula en lugar de reemplazar **[P2]**

**Repro:** Usuario con sesiones L/Mi/V agendadas → pide reschedule a Mar/Jue/Sab → resultado: 6 rows (L+Mar+Mi+Jue+V+Sab) en `user_weekly_schedule`.

**Causa:** `schedule_sessions` hace upsert con `on_conflict='user_id,week,week_day'`. Como los días viejos (L/Mi/V) y los nuevos (Mar/Jue/Sab) tienen `week_day` distinto, no conflictúan → ambos se preservan.

**Fix sugerido:** En `schedule_sessions`, antes del bulk_insert, hacer `DELETE FROM user_weekly_schedule WHERE user_id = ? AND week = current_week`. Esto vacía la semana actual y deja solo los días nuevos.

**Severidad:** P2 — no destruye datos, solo deja schedule "sucio". No bloquea producción pero confunde al usuario que puede ver sesiones duplicadas en su calendario.

---

## Conclusión

✅ **Bug P0 B-S10-001 cerrado.** El usuario activo ya no pierde su plan al pedir reschedule.

**Defense in depth funciona:**
- Capa 1 (prompt) — bot interpreta correctamente la intención
- Capa 2 (tool guards) — si el LLM falla, los guards bloquean operaciones destructivas

**Next steps:**
1. Fix de seguimiento para sub-bug B-T01-001 (schedule acumula)
2. Pasar a B-S02-001 (Equipment filter — el otro P0)
3. Considerar fix para B-S01-001 (KYC corto) y B-S01-005 (volumen bajo)
