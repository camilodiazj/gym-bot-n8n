# FIX PLAN — S02 Equipment Filter (P0)

**Bug:** [B-S02-001](S02-home-bodyweight-only.md) — Usuarios HOME bodyweight reciben rutinas con máquinas, mancuernas y bandas (67% de ejercicios incompatibles).
**Severidad:** P0 — Bloquea el caso de uso HOME training.
**Reporte original:** [S02-home-bodyweight-only.md](S02-home-bodyweight-only.md)

---

## 1. Contexto

**¿Qué falla?** El usuario dice "solo peso corporal en casa, sin pesas ni nada". `users_gym_profile.home_equipment = "peso corporal"`, `training_environment = "HOME"`. Pero `workouts` queda con 10/15 ejercicios usando `equipment IN ('machine','Mancuerna','resistance_band')`.

**Causa raíz (3 capas):**

1. **Soft filter en código** ([tools.py:526-533](../../cases/case6_unified_agent/tools.py#L526-L533)): si el filtro por equipment devuelve `[]`, el código mantiene la lista sin filtrar (vuelve a incluir gym equipment). Mismo patrón en [tools.py:603-607](../../cases/case6_unified_agent/tools.py#L603-L607) para `find_exercise_alternatives`.

2. **DB carente de bodyweight** para algunos patterns:
   | Pattern | bodyweight count |
   |---|---|
   | accessory | 65 ✅ |
   | core | 35 ✅ |
   | squat | 19 ✅ |
   | push_h | 18 ✅ |
   | arm | 15 ✅ |
   | push_v | 6 ✅ |
   | hinge | **1** ⚠️ |
   | pull_h | **0** ❌ |
   | pull_v | **0** ❌ |

3. **`save_workout_plan` no re-valida equipment** ([tools.py:773-907](../../cases/case6_unified_agent/tools.py#L773-L907)): acepta cualquier exercise_id que el LLM proponga sin verificar compatibilidad con `home_equipment` del usuario.

**Outcome esperado tras el fix:** un usuario HOME bodyweight obtiene una rutina **100% ejecutable en casa sin equipo**, o el bot dice honestamente que un pattern no tiene opciones bodyweight y propone sustituirlo / pedir equipment mínimo.

---

## 2. Cambios al código

### 2.1 Fase 1 — Eliminar soft filter (30 min)

**Archivo:** `langgraph-skeleton/cases/case6_unified_agent/tools.py`

**Cambio A:** `get_exercises_for_draft` (líneas 526-533).
```python
# ANTES
if effective_equipment:
    allowed_equip = _normalize_equipment(effective_equipment)
    equip_filtered = [
        r for r in filtered
        if r.get("equipment", "").lower() in allowed_equip
    ]
    if equip_filtered:
        filtered = equip_filtered

# DESPUÉS
if effective_equipment:
    allowed_equip = _normalize_equipment(effective_equipment)
    filtered = [
        r for r in filtered
        if r.get("equipment", "").lower() in allowed_equip
    ]
```

**Cambio B:** `find_exercise_alternatives` (líneas 603-607). Mismo patrón:
```python
# ANTES
if effective_equipment:
    allowed_equip = _normalize_equipment(effective_equipment)
    equip_filtered = [r for r in filtered if r.get("equipment", "").lower() in allowed_equip]
    if equip_filtered:
        filtered = equip_filtered

# DESPUÉS
if effective_equipment:
    allowed_equip = _normalize_equipment(effective_equipment)
    filtered = [r for r in filtered if r.get("equipment", "").lower() in allowed_equip]
```

**Efecto:** ahora `get_exercises_for_draft(pattern='pull_h', user_id=<HOME_bodyweight>)` devuelve `[]` en lugar de devolver ejercicios con máquinas.

---

### 2.2 Fase 2 — Validar equipment en `save_workout_plan` (1-2 h)

**Archivo:** `langgraph-skeleton/cases/case6_unified_agent/tools.py`

**Nueva helper** (insertar después de `_resolve_exercise_ids`, antes de `save_workout_plan` — alrededor de línea 755):

```python
async def _validate_workout_equipment(
    resolved_ids: list[str],
    user_id: str,
) -> tuple[bool, list[dict]]:
    """Verify that all resolved exercises respect user's home_equipment.

    Returns (is_valid, violations). violations is a list of
    {exercise_id, spanish_name, equipment} for any exercise that doesn't fit.
    """
    profile = await _lookup_user_profile(user_id)
    if profile["training_environment"] != "HOME":
        return True, []  # GYM users: no equipment restriction

    home_eq = profile.get("home_equipment", "") or "peso corporal"
    allowed = _normalize_equipment(home_eq)

    if not resolved_ids:
        return True, []

    rows = await supabase_query(
        "exercises",
        select="exercise_id,spanish_name,equipment",
        filters={"exercise_id": f"in.({','.join(resolved_ids)})"},
    )

    violations = [
        r for r in rows
        if r.get("equipment", "").lower() not in allowed
    ]
    return (len(violations) == 0), violations
```

**Modificación a `save_workout_plan`** (después de `resolved, unresolved = await _resolve_exercise_ids(draft)` — línea 812):

```python
# Validate HOME equipment compliance before any insert
all_resolved_ids = list(set(resolved.values()))
is_valid, violations = await _validate_workout_equipment(all_resolved_ids, user_id)
if not is_valid:
    return json.dumps({
        "success": False,
        "error": (
            f"La rutina propuesta incluye {len(violations)} ejercicio(s) "
            f"que requieren equipo que el usuario no tiene en casa. "
            f"Por favor reemplázalos con alternativas bodyweight o pide al usuario "
            f"qué equipo mínimo tiene disponible (bandas, mancuernas, barra de dominadas)."
        ),
        "violations": [
            {"name": v["spanish_name"], "equipment": v["equipment"]}
            for v in violations
        ],
    }, ensure_ascii=False)
```

**Efecto:** si por cualquier razón el draft tiene exercises con equipment incompatible, el save falla con un error claro que el LLM puede leer y reintentar.

---

### 2.3 Fase 3 — System prompt guidance (15 min)

**Archivo:** `langgraph-skeleton/cases/case6_unified_agent/prompts.py`

Agregar en la sección "REGLAS DE COMPORTAMIENTO" (después de línea 52, antes de "## USUARIOS NUEVOS"):

```
- EQUIPMENT FILTER: Si get_exercises_for_draft o find_exercise_alternatives devuelven lista vacía para un pattern + user HOME, NO INVENTES ejercicios. Tienes 3 opciones, en orden de preferencia:
  1. Sustituye el pattern por uno relacionado con bodyweight disponible (push_v sin bodyweight → push_h con flexiones diamond, pull_h sin bodyweight → accessory con rows en suspensión)
  2. Pregunta al usuario si tiene algún equipment mínimo: "Para trabajar bien la espalda en casa idealmente necesitas una barra de dominadas o bandas elásticas. ¿Tienes alguno de estos?"
  3. Si insiste en bodyweight puro, omite ese pattern para esa sesión y compensa con más volumen en patterns que sí tienen opciones bodyweight (accessory, core).
- Si save_workout_plan devuelve success=false con violations, lee la lista y reintenta usando find_exercise_alternatives con equipment correcto. NO finjas que se guardó si falló.
```

---

### 2.4 Fase 4 — Enriquecer DB (separado, no bloqueante)

Crear migración SQL para agregar ejercicios bodyweight a patterns deficitarios. Esto **no bloquea el fix** pero mejora la calidad de la rutina HOME bodyweight.

**Archivo:** `migrations/add_bodyweight_exercises.sql`

```sql
-- pull_h bodyweight (actualmente 0)
INSERT INTO exercises (exercise_id, spanish_name, pattern, role, main_muscle, secondary_muscles, level, equipment, link)
VALUES
  ('ex_bw_aussie_row', 'Remo invertido (Australian row)', 'pull_h', 'compound', 'Back', ARRAY['Biceps','Lower back'], 'Principiante', 'bodyweight', NULL),
  ('ex_bw_trx_row', 'Remo en suspensión TRX', 'pull_h', 'compound', 'Back', ARRAY['Biceps','Core'], 'Intermedio', 'bodyweight', NULL),
  ('ex_bw_door_row', 'Remo con toalla en puerta', 'pull_h', 'compound', 'Back', ARRAY['Biceps'], 'Principiante', 'bodyweight', NULL);

-- pull_v bodyweight (actualmente 0) — requieren barra de dominadas implícita
INSERT INTO exercises (exercise_id, spanish_name, pattern, role, main_muscle, secondary_muscles, level, equipment, link)
VALUES
  ('ex_bw_pullup', 'Dominadas (pull-ups)', 'pull_v', 'compound', 'Back', ARRAY['Biceps'], 'Avanzado', 'bodyweight', NULL),
  ('ex_bw_chinup', 'Chin-ups (agarre supino)', 'pull_v', 'compound', 'Back', ARRAY['Biceps'], 'Avanzado', 'bodyweight', NULL),
  ('ex_bw_neg_pullup', 'Negativas de dominadas', 'pull_v', 'compound', 'Back', ARRAY['Biceps'], 'Intermedio', 'bodyweight', NULL);

-- hinge bodyweight (actualmente 1, agregar más)
INSERT INTO exercises (exercise_id, spanish_name, pattern, role, main_muscle, secondary_muscles, level, equipment, link)
VALUES
  ('ex_bw_sl_rdl', 'Peso muerto rumano a una pierna (bodyweight)', 'hinge', 'compound', 'Hamstrings', ARRAY['Glutes','Lower back'], 'Intermedio', 'bodyweight', NULL),
  ('ex_bw_glute_bridge', 'Puente de glúteos (glute bridge)', 'hinge', 'compound', 'Glutes', ARRAY['Hamstrings','Lower back'], 'Principiante', 'bodyweight', NULL),
  ('ex_bw_good_morning', 'Good morning con peso corporal', 'hinge', 'compound', 'Lower back', ARRAY['Hamstrings','Glutes'], 'Intermedio', 'bodyweight', NULL);
```

> **Nota sobre `pull_v` y dominadas**: Aunque requieren barra de dominadas, se clasifican como `bodyweight` porque el usuario solo levanta su propio peso. Si el usuario realmente no tiene ni barra, el bot debe pedirlo via Fase 3 system prompt (opción 2).

---

## 3. Plan de pruebas E2E via WhatsApp Web

**Objetivo:** validar end-to-end que un usuario HOME bodyweight genera una rutina con **0 ejercicios incompatibles** después del deploy.

**Setup compartido:**
- Tester: Camilo (`+57 350 626 7523`, `573506267523`)
- Driver: Claude vía Chrome MCP en tab `1415617601` (`Kai.Ros` chat)
- Supabase: project `ixfdjvlrnxleilzlujxj`
- Deploy: `gcloud run deploy kairos-agent --source . --region us-central1 --project gen-lang-client-0432163259` desde `langgraph-skeleton/`

### 3.1 Pre-condiciones

1. Fixes 1-3 deployed a Cloud Run (Fase 4 puede ir o no — el test cubre ambos casos).
2. Verificar revision desplegada:
   ```bash
   gcloud run services describe kairos-agent --region us-central1 \
     --format="value(status.latestReadyRevisionName)"
   ```
3. Smoke test de health:
   ```bash
   curl -s https://kairos-agent-148665080566.us-central1.run.app/ | python3 -m json.tool
   ```

### 3.2 Teardown (ejecutar antes de cada test case)

```sql
DELETE FROM checkpoint_blobs   WHERE thread_id IN ('case6_573506267523', 'kyc_573506267523');
DELETE FROM checkpoint_writes  WHERE thread_id IN ('case6_573506267523', 'kyc_573506267523');
DELETE FROM checkpoints        WHERE thread_id IN ('case6_573506267523', 'kyc_573506267523');
DELETE FROM draft_routines      WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM magic_links         WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM workouts             WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM user_weekly_schedule WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM users_plans       WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM users_gym_profile WHERE whatsapp_id = 573506267523;
DELETE FROM users WHERE full_phone_number = 573506267523;
```

### 3.3 Test Case TC-FIX-S02-001 — HOME Bodyweight estricto (replica S02)

**Objetivo:** confirmar que la regresión específica del reporte original ya no ocurre.

**Steps:**

| # | Acción | Mensaje vía WhatsApp Web | Esperado |
|---|--------|--------------------------|----------|
| 1 | Teardown | (SQL) | DB limpia |
| 2 | Enviar mensaje 1 | `Hola! Quiero entrenar pero solo tengo peso corporal en casa, sin pesas ni nada.` | Bot inicia KYC, pregunta objetivo/experiencia/lesiones (no muerde con equipment, ya lo tiene) |
| 3 | Enviar mensaje 2 | `Ganar músculo. Intermedio, llevo 1 año entrenando. Puedo 3 días a la semana. Sin lesiones.` | Bot genera draft y manda link `/draft?c=XXXXXX`. Latencia ~40-50s. |
| 4 | Enviar mensaje 3 | `Me gusta, déjala fija` | Bot ejecuta `save_workout_plan`. Debería responder éxito + ofrecer schedule. Latencia ~20-25s. |

**Verificación SQL:**

```sql
-- Query 1: distribución de equipment en workouts guardados
SELECT e.equipment, COUNT(*) AS uses
FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = (SELECT user_id FROM users WHERE full_phone_number=573506267523)
GROUP BY e.equipment
ORDER BY uses DESC;
```

**Criterio de éxito:**
- Si Fase 4 ejecutada: solo aparecen `bodyweight` y `Peso Corporal` (100% compatibles).
- Si Fase 4 NO ejecutada: aparece solo `bodyweight` pero workouts pueden ser <15 (pull_h, pull_v, hinge omitidos). Aceptable.
- **NUNCA** debe aparecer `machine`, `Mancuerna`, `Máquina`, `resistance_band`, `barbell`.

```sql
-- Query 2: contar violations explícitas
SELECT COUNT(*) AS violations
FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = (SELECT user_id FROM users WHERE full_phone_number=573506267523)
  AND LOWER(e.equipment) NOT IN ('bodyweight', 'peso corporal');
```

**Criterio de éxito:** `violations = 0`.

### 3.4 Test Case TC-FIX-S02-002 — Equipment "mancuernas + bandas" (Fase 1 valida que se respeta whitelist más amplia)

**Objetivo:** verificar que `_normalize_equipment` sigue funcionando para HOME con más equipment.

**Steps:**

| # | Mensaje | Esperado |
|---|---------|----------|
| 1 | Teardown | DB limpia |
| 2 | `Hola, entreno en casa con mancuernas y bandas elásticas. Quiero ganar fuerza.` | KYC inicia |
| 3 | `Intermedio, 1 año entrenando, 4 días disponibles, sin lesiones` | Genera draft |
| 4 | `dale, guárdala` | Save plan |

**Verificación SQL:**

```sql
SELECT e.equipment, COUNT(*)
FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = (SELECT user_id FROM users WHERE full_phone_number=573506267523)
GROUP BY e.equipment;
```

**Criterio de éxito:** equipment ⊆ `{bodyweight, Peso Corporal, dumbbell, Mancuerna, resistance_band, bands}`. **Nunca** `machine`, `cable`, `barbell`.

### 3.5 Test Case TC-FIX-S02-003 — Guardrail de violation en `save_workout_plan` (Fase 2)

**Objetivo:** verificar que si por alguna razón el LLM intenta guardar exercises incompatibles, el save falla con error claro.

**Approach:** difícil reproducir orgánicamente porque Fase 1 ya previene que el LLM vea esos exercises. Test alternativo via SQL directo: simular un draft malicioso vía `/api/v1/chat` o llamar el tool directamente.

**Bash:**

```bash
# Simular request directa al endpoint con un draft "envenenado"
# (Skip si Fase 2 no se quiere stress-testear ahora)
```

**Alternativa más práctica:** unit test (no E2E) — agregar a `tests/test_case6.py`:

```python
async def test_save_workout_plan_rejects_home_user_with_machine_exercises():
    # given a HOME bodyweight user
    # when save_workout_plan is called with a draft including 'ex_machine_squat'
    # then it should return success=False with violations list
```

### 3.6 Test Case TC-FIX-S02-004 — Catalog limitation handled gracefully (Fase 3)

**Objetivo:** validar que cuando un pattern no tiene bodyweight, el bot lo maneja con honestidad (no inventa, no sustituye silenciosamente con máquina).

**Steps:**

| # | Mensaje | Esperado |
|---|---------|----------|
| 1 | Teardown | — |
| 2 | `Hola, solo peso corporal en casa.` | KYC |
| 3 | `Ganar masa, intermedio 1 año, 3 días, sin lesiones` | Draft generado |

**Verificación:**

Si Fase 4 NO ejecutada, esperar uno de estos comportamientos del bot:
- (a) Generar rutina **sin** pull_h/pull_v y compensar con más push, accessory, core
- (b) Preguntar al usuario: "Para trabajar la espalda en casa necesitarías una barra de dominadas o bandas elásticas. ¿Tienes alguno?"

NO aceptable:
- (c) Generar rutina con `machine` exercises
- (d) Generar rutina pero mentir que "He adaptado para que sea bodyweight" (B-S02-003)

**Criterio de éxito:** observar la respuesta del bot. Si dice opción (a) o (b), pass. Si (c) o (d), fail.

### 3.7 Test Case TC-FIX-S02-005 — find_exercise_alternatives respeta equipment (Fase 1)

**Objetivo:** validar que el swap (S05-style) también respeta equipment para HOME users.

**Steps (CHAIN de TC-001 post save_workout_plan):**

| # | Mensaje | Esperado |
|---|---------|----------|
| 1 | `cambia las flexiones, prefiero algo diferente para pecho` | Bot debería sugerir alternativa **también bodyweight** (e.g. fondos en silla, push-ups diamond) |

**Criterio:** la nueva alternativa tiene `equipment IN ('bodyweight','Peso Corporal')`.

---

## 4. Workflow de ejecución (Claude vía Chrome MCP)

Para cada test case:

1. **Setup**
   - Run teardown SQL via `mcp__1262a0db..._execute_sql`
   - Verify: `SELECT COUNT(*) FROM users WHERE full_phone_number=573506267523` → 0

2. **Conversación**
   - Vía `mcp__Claude_in_Chrome__find` + `triple_click` + `cmd+a` + `type` + `Return`
   - Esperar 20-50s tras cada mensaje (usar `Bash run_in_background: true` con `sleep N && echo done`)
   - `screenshot` después de cada respuesta del bot

3. **Verificación**
   - Run queries SQL del test case
   - Pull Cloud Run logs:
     ```bash
     gcloud logging read 'resource.type=cloud_run_revision AND resource.labels.service_name=kairos-agent AND textPayload:"573506267523"' --limit 30 --freshness 10m
     ```
   - Confirmar que `get_exercises_for_draft` se invocó con `user_id` correcto (debe aparecer en `[TOOL]` logs)

4. **Reporte**
   - Crear archivo `langgraph-skeleton/testing-reports/fix-s02-validation-{YYYY-MM-DD}/{TC-FIX-S02-NNN}.md`
   - Llenar tabla turn-by-turn + queries + verdict (PASS/FAIL)

---

## 5. Rollback plan

Si los tests fallan o aparecen regresiones críticas:

1. **Revert Fase 1+2 (código):**
   ```bash
   git revert <commit-hash-fase-1-2>
   gcloud run deploy kairos-agent --source langgraph-skeleton/ --region us-central1
   ```

2. **Revert Fase 3 (prompt):** mismo procedimiento. El prompt vive en código.

3. **Revert Fase 4 (DB):** ejecutar:
   ```sql
   DELETE FROM exercises WHERE exercise_id IN (
     'ex_bw_aussie_row','ex_bw_trx_row','ex_bw_door_row',
     'ex_bw_pullup','ex_bw_chinup','ex_bw_neg_pullup',
     'ex_bw_sl_rdl','ex_bw_glute_bridge','ex_bw_good_morning'
   );
   ```

4. **Smoke test post-rollback:** repetir `curl /` health check + un mensaje WhatsApp simple ("hola") para verificar que Kairos sigue respondiendo.

---

## 6. Definición de "DONE"

El fix se considera completo cuando:

- ✅ TC-FIX-S02-001 retorna `violations = 0` en la query SQL
- ✅ TC-FIX-S02-002 retorna equipment ⊆ whitelist del usuario
- ✅ TC-FIX-S02-004 muestra comportamiento honesto (a) o (b), nunca (c)/(d)
- ✅ TC-FIX-S02-005 swap respeta equipment
- ✅ Existing E2E tests del repo (n8n test runners) siguen pasando — verificar [tests/test_case6.py](../../tests/test_case6.py) no rompió
- ✅ Smoke test post-deploy: usuario GYM normal (no HOME) sigue generando rutinas correctamente (regresión nula)

**Estimación total:**
- Fase 1+2+3 (código + prompt): **2-3 horas**
- Fase 4 (DB enrichment): **1-2 días** (separado, no bloqueante)
- E2E test suite (5 TCs): **45-60 minutos** vía Chrome MCP

---

## 7. Archivos referenciados

| Archivo | Líneas | Cambio |
|---|---|---|
| `langgraph-skeleton/cases/case6_unified_agent/tools.py` | 526-533, 603-607 | Fase 1: eliminar soft filter |
| `langgraph-skeleton/cases/case6_unified_agent/tools.py` | +nuevo helper antes de 773, +mod en 812 | Fase 2: `_validate_workout_equipment` |
| `langgraph-skeleton/cases/case6_unified_agent/prompts.py` | después de línea 52 | Fase 3: instrucción de equipment empty |
| `migrations/add_bodyweight_exercises.sql` | nuevo | Fase 4: INSERTs |
| `langgraph-skeleton/tests/test_case6.py` | nuevo test | Unit test para Fase 2 (opcional) |
| `langgraph-skeleton/testing-reports/fix-s02-validation-YYYY-MM-DD/` | nuevo dir | Reportes E2E |
