# RE-VERIFY tras fixes — Cloud Run revision 00062-8bg

**Fecha:** 2026-05-11 (post-fixes)
**Tester:** Camilo (`573506267523`) vía WhatsApp Web
**Branch:** `claude/reverify-b-s01-005-rev00062` ≡ `origin/main @ 43ab1b8` (0 diff lines)
**Cloud Run revision:** `kairos-agent-00062-8bg`

**Commits relevantes:**
- `5d5ed1f fix(kairos): close P0 B-S10 reschedule destroys plan + B-S01-005 volume + E2E test reports (#23)`
- `22cfee3 docs(testing): verify B-S01-005 fix — 4/4 PASS (#24)`
- `43ab1b8 docs(testing): re-verify B-S01-005 on Cloud Run revision 00062-8bg — 4/4 PASS (#25)`

---

## Resultados

| Bug | Severidad original | Estado tras fix | Verificación |
|---|---|---|---|
| **B-S10-001** Reschedule destruye plan | P0 | ✅ **FIXED** | "Necesito cambiar mis días, ahora puedo Mar/Jue/Sab" → workouts=60 intactos, mesocycle=1 sin cambio, week_schedule=fb_3 sin cambio |
| **B-S01-005** Volumen bajo para Avanzado | P1 | ✅ **FIXED** | Workouts ahora con sets variables por (week, role): W1 compound=4, W3=5, W4=2-3 (deload), RIR 2→0-1→4 |
| **B-S02-001** Equipment filter HOME | P0 | ❌ **STILL BROKEN** | Usuario "solo peso corporal" recibe 57% workouts con barbell/machine/mancuerna |

---

## RE-S10 — Reschedule no destruye plan ✅ PASS

### Conversación
1. `Hola! Quiero ganar masa muscular. Avanzado, llevo más de 3 años entrenando, 3 días en gimnasio bien equipado.` → KYC + draft `023559`
2. `Me gusta, guárdala. Y agéndame lunes, miércoles y viernes esta semana.` → save_workout_plan, schedule_sessions
3. **`Necesito cambiar mis días. Ahora puedo martes, jueves y sábado.`** → respuesta crítica

### Respuesta del bot (turn 3)
> ¡De una, Camilo! Ya actualicé tu horario para esta semana.
>
> Tus sesiones quedaron así:
> - **Martes 12/05:** Full Body A
> - **Jueves 14/05:** Full Body B
> - **Sábado 16/05:** Full Body C
>
> ¡Mañana empezamos con toda!

### DB state

| Métrica | Pre-reschedule | Post-reschedule | Esperado | ¿OK? |
|---|---|---|---|---|
| `users_plans.mesocycle_number` | 1 | **1** | 1 (sin cambio) | ✅ |
| `users_plans.week_schedule` | `fb_3` | **`fb_3`** | `fb_3` (sin cambio) | ✅ |
| `users_plans.last_renewal_date` | NULL | **NULL** | NULL (sin cambio) | ✅ |
| `workouts` count | 60 | **60** | 60 (NO se destruyen) | ✅ |
| `user_weekly_schedule` | 3 (L/Mi/V) | 6 (L/Ma/Mi/J/V/Sa) | 3 (Mar/Jue/Sab) | ⚠️ Bug menor nuevo |

### Bug nuevo menor — B-S10-FIXED-001

**Severidad:** P2

El bot dijo "Tus sesiones quedaron así: Martes/Jueves/Sábado" pero NO removió las viejas (L/Mi/V). En lugar de **reemplazar**, hizo **agregar**. La DB tiene 6 filas (3 viejas + 3 nuevas).

**Esperado:** UPDATE/DELETE old rows, INSERT new ones → schedule final con 3 filas (Mar/Jue/Sab).
**Actual:** INSERT only → 6 filas (L,Mi,V,Ma,J,Sa).

**Impacto:** Usuario abre el tracker y ve sesiones en días que ya no aplican. La conversación afirma lo correcto pero la DB tiene state inconsistente.

**Hipótesis:** `schedule_sessions` hace INSERT sin previo DELETE de la semana actual.

**Sugerencia:** Antes de INSERT new, DELETE old (sin tocar otras semanas):
```sql
DELETE FROM user_weekly_schedule
WHERE user_id = $1 AND week = $2 AND NOT "Completed";
```

(Mantener sesiones ya completadas para no perder historial.)

**Veredicto general de B-S10-001:** ✅ **FIXED** — el daño catastrófico (60 workouts destruidos + mesocycle++) está cerrado. Queda un bug residual menor de "agregar en lugar de reemplazar" que NO destruye datos, solo crea ambigüedad visual.

---

## RE-S01 — Volumen periodizado ✅ PASS

### Workouts guardados (15 ejercicios × 4 semanas = 60)

| Week | Role | Sets | Reps | RIR | Rest | Periodización |
|---|---|---|---|---|---|---|
| 1 | compound | 4 | 6-10 | 2 | 180 | Acumulación |
| 1 | isolation | 4 | 10-15 | 1-2 | 90 | Acumulación |
| 1 | core | 3-4 | 8-15 | 2 | 60 | Acumulación |
| 2 | compound | 4-5 | 5-8 | 1-2 | 210 | Intensificación |
| 2 | isolation | 4-5 | 10-12 | 0-1 | 90 | Intensificación |
| 2 | core | 3-4 | 8-15 | 1-2 | 60 | Intensificación |
| 3 | compound | **5** | 4-6 | **0-1** | 240 | Pico |
| 3 | isolation | 5 | 8-12 | 0-1 | 90 | Pico |
| 3 | core | 3-4 | 8-15 | 1 | 60 | Pico |
| 4 | compound | **2-3** | 6-10 | **4** | 180 | **DELOAD** |
| 4 | isolation | 2-3 | 10-15 | 4 | 75 | DELOAD |
| 4 | core | 2-3 | 8-12 | 4 | 60 | DELOAD |

### Comparativa antes/después

| Aspecto | Antes (S01 original) | Después (Re-S01) |
|---|---|---|
| Sets por compound | 3 fijos | 4 → 4-5 → 5 → 2-3 |
| Reps | 8-10 fijos | 6-10 / 5-8 / 4-6 / 6-10 |
| RIR | 1-2 fijos | 2 → 1-2 → 0-1 → 4 |
| Rest compound | 150s fijos | 180 → 210 → 240 → 180 |
| Distinción role | No (todo igual) | Sí (compound/isolation/core) |
| Deload semana 4 | No | Sí (RIR 4, sets reducidos) |

✅ Match exacto con `set_profiles` (Avanzado/Ganar masa muscular).

---

## RE-S02 — Equipment filter HOME bodyweight ❌ STILL FAIL P0

### Conversación
1. `Hola! Quiero entrenar pero solo tengo peso corporal en casa, sin pesas ni nada. Ganar músculo, intermedio 1 año entrenando, 3 días.` → KYC + draft `0236d6`
2. `Me gusta, guárdala` → save_workout_plan

### Distribución de equipment en workouts guardados

```sql
SELECT DISTINCT e.equipment, COUNT(*) AS uses
FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523)
GROUP BY e.equipment ORDER BY uses DESC;
```

| Equipment | Uses | ¿OK bodyweight? |
|---|---|---|
| bodyweight | 16 | ✅ |
| Peso Corporal | 8 | ✅ |
| **barbell** | 8 | ❌ |
| **machine** | 8 | ❌ |
| **Mancuerna** | 8 | ❌ |
| **Máquina** | 8 | ❌ |

**Totales:**
- ✅ Compatibles con bodyweight: 24/56 (43%)
- ❌ Incompatibles: 32/56 (**57%**)

### Estado

**Sin cambios respecto al test original.** El usuario sigue recibiendo rutina con barbell + máquinas + mancuernas pese a decir explícitamente "solo peso corporal, sin pesas ni nada".

El fix `5d5ed1f` no incluyó este bug — los commits abordaron:
- B-S10 (reschedule destroys) ✅
- B-S01-005 (volumen) ✅
- E2E test reports (docs)

Pero **NO** B-S02-001 (equipment filter).

### Hipótesis pendiente

El filtro de equipment requiere modificaciones a `get_exercises_for_draft` ([langgraph-skeleton/cases/case6_unified_agent/tools.py:467](../../langgraph-skeleton/cases/case6_unified_agent/tools.py)) para:
1. Aplicar filtro WHERE estricto basado en `home_equipment` cuando `training_environment='HOME'`
2. Usar la función `_normalize_equipment()` ya existente ([tools.py:56-71](../../langgraph-skeleton/cases/case6_unified_agent/tools.py))
3. Validación post-save: leer workouts guardados + verificar que todos respetan equipment del usuario; si no, abortar

---

## Resumen ejecutivo

**De los 3 bugs críticos del test original, 2 cerrados, 1 pendiente:**

| Bug | Estado |
|---|---|
| B-S10-001 (P0) — Reschedule destruye plan | ✅ **FIXED** |
| B-S01-005 (P1) — Volumen bajo Avanzado | ✅ **FIXED** |
| B-S02-001 (P0) — Equipment filter HOME | ❌ **PENDIENTE** |

**Bugs residuales detectados en re-verify:**

| ID | Sev | Título |
|---|---|---|
| B-S10-FIXED-001 | P2 | Reschedule agrega filas en lugar de reemplazar (state inconsistente, no destructivo) |

**Recomendación:** Atacar **B-S02-001** como próximo P0. Fix esperado en `get_exercises_for_draft` + función `_normalize_equipment` ya existente. ETA estimada del fix: 2-4 horas.

---

## Anexo — Logs Cloud Run

```bash
gcloud logging read 'resource.type=cloud_run_revision AND resource.labels.service_name=kairos-agent AND textPayload:"573506267523"' \
  --freshness 1h --limit 50
```

Ninguna warning `[LOADING]` durante la corrida — `set_profiles` tiene cobertura completa para `(Ganar masa muscular, Avanzado, *, *)`.
