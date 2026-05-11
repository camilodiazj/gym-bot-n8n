# Fix Plan — B-S10-001 (Reschedule destruye plan)

**Bug:** P0 bloqueante. Reschedule simple gatilla `renew_change_days` → borra workouts + schedule + marca `Completed=true` + incrementa `mesocycle_number`.
**Reporte detalle:** [testing-reports/manual-kairos-2026-05-11/S10-reschedule.md](../testing-reports/manual-kairos-2026-05-11/S10-reschedule.md)
**Estimación:** ~1.5 horas (código 35 min + deploy 5 min + E2E test 50 min)

---

## Contexto

Un usuario activo en mesociclo 1 (semana 1, 1 sesión completada) escribe en WhatsApp:

> *"Necesito cambiar mis días. Ahora puedo martes, jueves y sábado."*

El LLM interpreta "cambiar días" como opción 2 del menú de renovación (cambiar **cuántos** días). Llama `renew_change_days(user_id, 3)` — misma cantidad que el usuario ya tenía. El tool:

1. Borra los 60 workouts del plan (4 semanas × 3 días × 5 ejercicios)
2. Borra las 3 filas de `user_weekly_schedule` (incluyendo la marca `Completed=true` de hoy)
3. Incrementa `mesocycle_number` 1 → 2
4. Setea `last_renewal_date` a `now`
5. Crea un nuevo `draft_routines` pendiente de aprobación

El bot **miente** al usuario: dice *"Ya ajusté tu plan"* cuando en realidad lo destruyó y dejó el estado inconsistente.

---

## Causas raíz

| # | Causa | Archivo / línea |
|---|---|---|
| **A** | Ambigüedad lingüística: el prompt no distingue "cambiar QUÉ días" (reschedule) de "cambiar CUÁNTOS días" (renew_change_days) | [prompts.py:170-174](../cases/case6_unified_agent/prompts.py#L170-L174) |
| **B** | `renew_change_days` no valida que `new_days_per_week ≠ current_days_per_week` (no-op destructivo) | [tools.py:1300-1318](../cases/case6_unified_agent/tools.py#L1300-L1318) |
| **C** | `renew_change_days` no valida `mesocycle_complete` antes de borrar (los datos del usuario activo pueden destruirse en W1) | [tools.py:1300-1318](../cases/case6_unified_agent/tools.py#L1300-L1318) |

---

## Fixes

### Fix 1 (CRITICAL) — Guardias en `renew_change_days`

**Archivo:** [`langgraph-skeleton/cases/case6_unified_agent/tools.py`](../cases/case6_unified_agent/tools.py) (función `renew_change_days`, línea 1300)

Insertar después del check de rango (línea 1318), antes de cualquier `DELETE`:

```python
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

# ─── GUARD 1: nueva cantidad debe ser distinta a la actual ───
ws_to_days = {"fb_2": 2, "fb_3": 3, "ul_4": 4, "ppl_5": 5, "ppl_6": 6}
current_days = ws_to_days.get(plan.get("week_schedule", ""), 0)
if new_days_per_week == current_days:
    return json.dumps({
        "success": False,
        "error": "SAME_DAYS_PER_WEEK",
        "message": (
            f"Ya entrenas {current_days} días por semana. "
            f"Si quieres cambiar QUÉ días específicos entrenas (no cuántos), "
            f"usa schedule_sessions. Si quieres mantener los mismos ejercicios "
            f"para el siguiente mesociclo, usa renew_maintain."
        ),
    }, ensure_ascii=False)

# ─── GUARD 2: mesociclo actual debe estar completo ───
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
            f"entrenas, usa schedule_sessions sin renovar."
        ),
    }, ensure_ascii=False)

new_meso = plan["mesocycle_number"] + 1
# ... resto del código original (DELETE workouts, UPDATE plan, etc.)
```

Eliminar la consulta de `plans` original (líneas 1323-1333) — ya está hecha arriba.

### Fix 2 (CRITICAL) — Disambiguación en el system prompt

**Archivo:** [`langgraph-skeleton/cases/case6_unified_agent/prompts.py`](../cases/case6_unified_agent/prompts.py)

Reemplazar el bloque "## RENOVACIÓN DE MESOCICLO" completo (líneas 163-192) por:

```markdown
## RESCHEDULE vs RENOVACIÓN DE MESOCICLO — DISTINGUIR

La frase "cambiar días" tiene DOS significados en español. Debes distinguir cuál antes de actuar:

### (A) Reschedule — cambiar QUÉ días de la semana entreno
- Trigger: "Quiero entrenar martes, jueves y sábado", "Cambia el lunes por martes"
- Acción: `schedule_sessions(user_id, sessions_json)` con los nuevos días
- NO renueva mesociclo, NO borra workouts, NO incrementa mesocycle_number

### (B) Renovación de frecuencia — cambiar CUÁNTOS días por semana
- Trigger: "Ahora quiero entrenar 5 días en vez de 3", "Subir de 3 a 4 días"
- Acción: `renew_change_days(user_id, N)` SOLO si N ≠ días actuales
- REQUIERE: mesociclo actual COMPLETO (W4 con todas las sesiones completadas)
- Destruye workouts del mesociclo anterior y genera plan nuevo

### Reglas de detección — sigue este árbol:
1. ¿El usuario menciona días específicos de la semana (lunes, martes, etc.)?
   - **Sí, y NO menciona nueva cantidad** → (A) Reschedule → `schedule_sessions`
2. ¿El usuario menciona explícitamente una cantidad nueva ("5 días", "3 días en vez de 4")?
   - **Sí, cantidad ≠ actual** → (B) Renovación → `renew_change_days`
   - **Sí, cantidad = actual** → no es renovación. Es reschedule o renew_maintain
3. ¿Es ambiguo (e.g. "quiero cambiar mis días" sin más detalle)?
   - **Sí** → PREGUNTA: "¿Quieres cambiar **cuáles** días entrenas (e.g. martes en vez de lunes) o cambiar **cuántos** días entrenas (e.g. de 3 a 5)?"
4. ¿El contexto NO muestra "Mesociclo COMPLETADO — listo para renovación"?
   - **No completado** → NUNCA invoques `renew_*`. Explica que primero debe terminar. Si solo quiere mover días, ofrece `schedule_sessions`.

## RENOVACIÓN DE MESOCICLO

### Cuándo activar
- Solo si el contexto dice "Mesociclo COMPLETADO — listo para renovación" Y el usuario lo pide.
- Si NO está completo, NO ofrezcas ni ejecutes renovación. Explica que falta terminar W4.

### Opciones (presentar EXACTAMENTE estas 3 cuando el mesociclo esté completo)
1. **Mantener rutina** — mismos ejercicios, nuevo mesociclo con progresión de carga
2. **Subir/bajar frecuencia** — cambiar cuántos días por semana (2-6), rutina nueva
3. **Rotar ejercicios** — misma estructura, ejercicios nuevos del mismo patrón

### Secuencia por opción

**Opción 1 — Mantener**: Llama `renew_maintain(user_id)`. Luego sugiere agendar semana 1.

**Opción 2 — Subir/bajar frecuencia**:
1. Pregunta cuántos días quiere entrenar (2-6). Si dice un número igual al actual, di "Ya entrenas N días, ¿quieres mantener la rutina con renew_maintain o cambiar QUÉ días específicos con schedule_sessions?"
2. Si es diferente, llama `renew_change_days(user_id, new_days)`.
3. LUEGO crea la nueva rutina con la secuencia de CREACIÓN DE RUTINA (Pasos 1-5).
4. En el Paso 5, agrega `"is_renewal": true` al JSON de save_workout_plan.
5. Sugiere agendar.

**Opción 3 — Rotar**: Llama `renew_rotate_exercises(user_id)`. Presenta resumen. Sugiere agendar.

### Reglas de renovación
- Si dice "mantener" o "1" → `renew_maintain` inmediato.
- Si dice "rotar" o "3" → `renew_rotate_exercises` inmediato.
- Si dice "cambiar días" o "2" → pregunta SOLO cuántos días.
- Si `renew_change_days` devuelve error `SAME_DAYS_PER_WEEK` → explica al usuario y ofrece `renew_maintain` o `schedule_sessions`.
- Si `renew_change_days` devuelve error `MESOCYCLE_NOT_COMPLETE` → explica cuántas sesiones faltan y ofrece `schedule_sessions` si solo quiere mover días.
- Después de CUALQUIER renovación, el usuario DEBE agendar sus sesiones de semana 1.
```

### Fix 3 — Guardia análoga en `renew_maintain` y `renew_rotate_exercises`

**Archivo:** [`tools.py`](../cases/case6_unified_agent/tools.py)

`renew_maintain` (línea 1126) y `renew_rotate_exercises` (línea 1174): aplicar la misma guardia "GUARD 2" (mesociclo completo). `renew_maintain` no destruye workouts pero sí borra schedule e incrementa mesocycle_number — debe respetar la pre-condición.

Extraer un helper para evitar duplicación:

```python
async def _validate_mesocycle_complete(user_id: str) -> tuple[bool, str | None]:
    """Returns (is_complete, error_message). Si is_complete=False, error_message tiene el JSON de error."""
    w4 = await supabase_query(
        "user_weekly_schedule",
        select='day_routine_id,"Completed"',
        filters={"user_id": f"eq.{user_id}", "week": "eq.4"},
    )
    w4_total = len(w4)
    w4_completed = sum(1 for s in w4 if s.get("Completed", False))
    if w4_total == 0 or w4_completed < w4_total:
        return False, json.dumps({
            "success": False,
            "error": "MESOCYCLE_NOT_COMPLETE",
            "week4_completed": w4_completed,
            "week4_total": w4_total,
            "message": f"Aún no terminas el mesociclo (W4: {w4_completed}/{w4_total}). Termina primero.",
        }, ensure_ascii=False)
    return True, None
```

Y al inicio de cada `renew_*`:

```python
is_complete, error = await _validate_mesocycle_complete(user_id)
if not is_complete:
    return error
```

---

## Deploy

```bash
cd langgraph-skeleton
gcloud run deploy kairos-agent \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --project gen-lang-client-0432163259
```

Tarda ~5 min. Verificar:

```bash
curl -s https://kairos-agent-148665080566.us-central1.run.app/ | python3 -m json.tool
# → status: ok
```

---

## E2E Test Plan vía WhatsApp Web

### Pre-requisitos

1. **WhatsApp Web abierto** en el chat `Kai.Ros` (tab 1415617601 en Browser 1)
2. **Supabase MCP** conectado (project `ixfdjvlrnxleilzlujxj`)
3. **Cloud Run** con la nueva revisión desplegada
4. **Phone:** `573506267523` (Camilo)
5. **Driver:** Claude conduce vía Chrome MCP + Supabase MCP, igual que en el test plan original

### Teardown común (ejecutar antes de cada caso)

```sql
DELETE FROM checkpoint_blobs   WHERE thread_id IN ('case6_573506267523', 'kyc_573506267523');
DELETE FROM checkpoint_writes  WHERE thread_id IN ('case6_573506267523', 'kyc_573506267523');
DELETE FROM checkpoints        WHERE thread_id IN ('case6_573506267523', 'kyc_573506267523');
DELETE FROM draft_routines      WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM set_values          WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM magic_links         WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM pending_tasks       WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM n8n_chat_histories  WHERE session_id LIKE '%573506267523%';
DELETE FROM workouts             WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM user_weekly_schedule WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM users_plans       WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM users_gym_profile WHERE whatsapp_id = 573506267523;
DELETE FROM users WHERE full_phone_number = 573506267523;
```

### Onboarding rápido común (para casos que requieren plan activo)

Send via WhatsApp:
```
Hola! Quiero ganar masa muscular, llevo 2 años entrenando, 3 días en gimnasio, sin lesiones.
```
Esperar respuesta del bot con draft link. Luego:
```
Aprueba la rutina
```
Esperar respuesta. Luego:
```
Agenda lunes hoy, miércoles y viernes
```

Verifica:
```sql
SELECT mesocycle_number, week_schedule FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: mesocycle=1, week_schedule=fb_3
SELECT COUNT(*) FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: ~60
SELECT COUNT(*) FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 3
```

### Fixture F3 (marcar W4 completo, para casos que necesitan can_renew=true)

```sql
DELETE FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
INSERT INTO user_weekly_schedule (day_routine_id, user_id, week, week_day, session_name, planned_day, "Completed")
SELECT gen_random_uuid(),
  (SELECT user_id FROM users WHERE full_phone_number=573506267523),
  w, wd::week_days, sn,
  TO_CHAR((NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '1 day' * (35 - ((w-1)*7 + (CASE wd WHEN 'Lunes' THEN 1 WHEN 'Miercoles' THEN 3 WHEN 'Viernes' THEN 5 END))), 'DD/MM/YYYY'),
  true
FROM (VALUES (1,'Lunes','Full Body A'),(1,'Miercoles','Full Body B'),(1,'Viernes','Full Body C'),
             (2,'Lunes','Full Body A'),(2,'Miercoles','Full Body B'),(2,'Viernes','Full Body C'),
             (3,'Lunes','Full Body A'),(3,'Miercoles','Full Body B'),(3,'Viernes','Full Body C'),
             (4,'Lunes','Full Body A'),(4,'Miercoles','Full Body B'),(4,'Viernes','Full Body C')) AS v(w,wd,sn);
```

---

### Caso T01 — Reschedule explícito (el original bug, debe ahora PASAR)

**Objetivo:** Confirmar que "cambiar mis días a Mar/Jue/Sab" NO destruye nada y solo cambia el schedule.

**Setup:**
1. Teardown común
2. Onboarding rápido común (deja usuario en W1 con 60 workouts, schedule L/Mi/V)

**Snapshot pre-test (guardar):**
```sql
SELECT mesocycle_number FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
SELECT COUNT(*) FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
SELECT week_day, planned_day FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523) ORDER BY planned_day;
```

**Acción WhatsApp:**
> `Necesito cambiar mis días. Ahora puedo martes, jueves y sábado.`

**Expected respuesta del bot:** Algo como *"Listo, agendé tus sesiones para martes, jueves y sábado. Tu plan sigue igual."* — SIN mencionar "nueva rutina" ni "rotar ejercicios" ni "draft".

**Verificación DB:**
```sql
SELECT mesocycle_number FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 1 (SIN cambio) ✅
SELECT COUNT(*) FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 60 (intactos) ✅
SELECT week_day, planned_day FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523) ORDER BY planned_day;
-- Expected: 3 filas con Martes/Jueves/Sábado, planned_day = mar/jue/sab esta semana ✅
SELECT COUNT(*) FROM draft_routines WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523) AND status='pending' AND created_at > NOW() - INTERVAL '5 minutes';
-- Expected: 0 (no draft nuevo) ✅
```

**Pass criteria:** mesocycle=1, workouts=60, schedule rows = 3 (con días nuevos), no draft nuevo, bot no menciona renovación.

---

### Caso T02 — "Cambiar días" ambiguo (sin contexto)

**Objetivo:** Confirmar que el bot pregunta antes de actuar cuando la frase es ambigua.

**Setup:** Igual que T01 (usuario en W1 con plan + schedule)

**Acción WhatsApp:**
> `Quiero cambiar mis días`

**Expected respuesta:** Pregunta clarificatoria. Algo como *"¿Quieres cambiar **cuáles** días entrenas (e.g. martes en vez de lunes) o cambiar **cuántos** días entrenas (e.g. de 3 a 5)?"*

**Verificación DB (nada debe haber cambiado):**
```sql
SELECT mesocycle_number FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 1 ✅
SELECT COUNT(*) FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 60 ✅
SELECT COUNT(*) FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 3 (sin cambios) ✅
```

**Pass criteria:** Bot pregunta clarificatoria, NADA en DB cambia.

---

### Caso T03 — Renovar con misma cantidad (no-op)

**Objetivo:** Confirmar que GUARD 1 (`SAME_DAYS_PER_WEEK`) se dispara cuando el usuario pide la misma cantidad que ya tiene.

**Setup:**
1. Teardown común
2. Onboarding rápido (fb_3, 3 días)
3. Fixture F3 (marcar W4 completa)

**Acción WhatsApp:**
> `Ya terminé las 4 semanas. Quiero el siguiente mesociclo con 3 días por semana, igual que ahora.`

**Expected respuesta:** Bot ofrece `renew_maintain` o pregunta si quiere cambiar QUÉ días. Algo como *"Ya entrenas 3 días por semana — si quieres mantener los mismos ejercicios para el nuevo mesociclo, dime 'mantener', y si quieres cambiar QUÉ días específicos, dime los nuevos."*

**Verificación DB:**
```sql
SELECT mesocycle_number, week_schedule FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: mesocycle=1, week_schedule=fb_3 (SIN cambios) ✅
SELECT COUNT(*) FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 60 (intactos) ✅
```

**Logs Cloud Run:** Buscar `SAME_DAYS_PER_WEEK` para confirmar que el guard se disparó.

```bash
gcloud logging read 'resource.type=cloud_run_revision AND resource.labels.service_name=kairos-agent AND textPayload:"SAME_DAYS_PER_WEEK"' --freshness 5m --limit 5
```

**Pass criteria:** No destruye nada, log muestra el error tool-side, bot ofrece alternativa.

---

### Caso T04 — Renovar prematuramente (W1, mesociclo NO completo)

**Objetivo:** Confirmar que GUARD 2 (`MESOCYCLE_NOT_COMPLETE`) se dispara.

**Setup:**
1. Teardown común
2. Onboarding rápido (usuario en W1, sin sesiones completadas)
3. NO aplicar F3

**Acción WhatsApp:**
> `Quiero cambiar a 5 días por semana, hacer un nuevo ciclo.`

**Expected respuesta:** Bot rechaza con explicación. Algo como *"Aún estás en semana 1 — te quedan 12 sesiones del mesociclo actual. Termínalo primero, o si solo quieres cambiar qué días específicos entrenas (no cuántos), dime los nuevos días."*

**Verificación DB:**
```sql
SELECT mesocycle_number, week_schedule FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: mesocycle=1, week_schedule=fb_3 ✅
SELECT COUNT(*) FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 60 ✅
SELECT COUNT(*) FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 3 (sin cambios) ✅
```

**Logs:**
```bash
gcloud logging read 'textPayload:"MESOCYCLE_NOT_COMPLETE"' --freshness 5m --limit 5
```

**Pass criteria:** No destruye nada, bot explica el rechazo.

---

### Caso T05 — Renovación legítima (regresión — debe seguir funcionando)

**Objetivo:** Confirmar que el flujo CORRECTO (mesociclo completo + cambio explícito de cantidad) sigue ejecutándose sin problemas.

**Setup:**
1. Teardown común
2. Onboarding rápido (fb_3)
3. Fixture F3 (W4 completa)

**Acción WhatsApp:**
> `Ya completé las 4 semanas. Quiero pasar a 5 días por semana en el siguiente mesociclo.`

**Expected respuesta:** Bot ejecuta `renew_change_days(5)` y genera draft de 5 días.

**Verificación DB:**
```sql
SELECT mesocycle_number, week_schedule FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: mesocycle=2, week_schedule=ppl_5 ✅
SELECT COUNT(*) FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 0 (workouts viejos borrados; nuevos solo después de aprobar el draft) ✅
SELECT COUNT(*) FROM draft_routines WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523) AND status='pending' AND created_at > NOW() - INTERVAL '5 minutes';
-- Expected: 1 (draft nuevo de 5 días) ✅
```

**Pass criteria:** Renovación se ejecuta exitosamente. Este caso es exactamente S14 del test plan original — no debe regresar.

---

### Caso T06 — Renew_maintain prematuro (Fix 3, opcional pero importante)

**Objetivo:** Confirmar que `renew_maintain` también respeta la pre-condición de mesociclo completo.

**Setup:**
1. Teardown común
2. Onboarding rápido (W1, sin completar)

**Acción WhatsApp:**
> `Quiero renovar mi rutina pero manteniendo los mismos ejercicios.`

**Expected respuesta:** Bot rechaza con explicación similar a T04.

**Verificación DB:**
```sql
SELECT mesocycle_number FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 1 (sin cambios) ✅
SELECT COUNT(*) FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523);
-- Expected: 3 (sin cambios) ✅
```

**Pass criteria:** No incrementa mesocycle, no borra schedule.

---

## Resumen de criterios de aceptación

| Caso | Tipo | DB cambia? | Bot menciona renovación? | Pass si |
|---|---|---|---|---|
| **T01** | Reschedule | Sólo schedule (días nuevos) | NO | mesocycle=1, workouts=60, schedule con Mar/Jue/Sab |
| **T02** | Ambiguo | NO | NO | Bot pregunta clarificatoria, DB intacta |
| **T03** | Same days | NO | Sí (rechaza con alternativas) | mesocycle=1, workouts=60, log con SAME_DAYS_PER_WEEK |
| **T04** | Renew premature | NO | Sí (rechaza, explica W4 falta) | mesocycle=1, workouts=60, log con MESOCYCLE_NOT_COMPLETE |
| **T05** | Renew legítimo (regresión) | Sí (renueva) | Sí (acepta) | mesocycle=2, week_schedule=ppl_5, nuevo draft |
| **T06** | Maintain premature (Fix 3) | NO | Sí (rechaza) | mesocycle=1, schedule intacto |

**Todos los 6 casos deben pasar.** T05 es crítico para confirmar que no introducimos regresión en S14.

---

## Rollback

Si el deploy rompe algo en producción:

```bash
# Listar revisiones
gcloud run revisions list --service kairos-agent --region us-central1 --limit 5

# Volver a la revisión anterior (kairos-agent-00059-stw es la pre-fix)
gcloud run services update-traffic kairos-agent --region us-central1 \
  --to-revisions=kairos-agent-00059-stw=100
```

---

## Archivos a modificar (resumen)

| Archivo | Cambio | Líneas afectadas |
|---|---|---|
| [`tools.py`](../cases/case6_unified_agent/tools.py) | Agregar 2 guardias en `renew_change_days` (Fix 1) | ~1300-1330 |
| [`tools.py`](../cases/case6_unified_agent/tools.py) | Agregar helper `_validate_mesocycle_complete` + llamarlo en `renew_maintain` y `renew_rotate_exercises` (Fix 3) | ~1126, ~1174, nuevo helper |
| [`prompts.py`](../cases/case6_unified_agent/prompts.py) | Reescribir sección RENOVACIÓN DE MESOCICLO con bloque "RESCHEDULE vs RENOVACIÓN — DISTINGUIR" (Fix 2) | ~163-192 |

---

## Próximos pasos sugeridos

1. Aplicar Fix 1, 2, 3 en una sola PR
2. Deploy a Cloud Run
3. Ejecutar T01-T06 vía Chrome MCP + Supabase MCP (yo conduzco si quieres, igual que el test plan original)
4. Si todos pasan → cerrar el ticket de B-S10-001
5. Pasar a fix B-S02-001 (Equipment filter), que es el otro P0
