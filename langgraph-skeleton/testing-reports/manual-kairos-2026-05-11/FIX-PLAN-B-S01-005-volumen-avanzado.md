# Fix Plan: B-S01-005 — Volumen bajo (3 sets) para Avanzado

**Severidad:** P1
**Estimación:** 1 día (implementación + tests)
**Owner:** TBD

---

## 1. Contexto

Detectado en [S01](S01-new-user-gym-happy-path.md). Usuario `Avanzado` + `Ganar masa muscular` recibe `sets='3', reps='8-10', rir='1-2', rest=150s` para TODOS los ejercicios y las 4 semanas. Sin periodización, sin distinción compound/isolation/core. Equivale al programa de un **Principiante**.

### Root cause (tres capas)

1. **`set_profiles` huérfana** — La tabla en Supabase tiene los valores correctos (4-5 sets para compounds avanzados, periodización W1→W4 con deload) pero NUNCA se consulta desde [tools.py](langgraph-skeleton/cases/case6_unified_agent/tools.py). `grep set_profiles` retorna 0 referencias.

2. **El LLM copia el ejemplo del prompt** — [prompts.py:124](langgraph-skeleton/cases/case6_unified_agent/prompts.py#L124) contiene `"sets":3,"reps":"8-10","rir":"1-2","rest_seconds":150`. Como el LLM no tiene otra fuente, copia esos valores literalmente al construir el draft. Los workouts guardados en S01 tienen EXACTAMENTE esos valores.

3. **`save_workout_plan` no varía por semana** — [tools.py:830-852](langgraph-skeleton/cases/case6_unified_agent/tools.py#L830) hace `for week in range(1, 5)` con los MISMOS sets/reps/rir/rest en cada iteración. Cero periodización.

---

## 2. Solución (Opción C — recomendada)

Mover la responsabilidad de los parámetros de carga de "lo que el LLM inventa" a "lo que la tabla prescribe". El LLM solo decide qué ejercicios y en qué orden; los sets/reps/rir/rest/tempo los pone `save_workout_plan` leyendo `set_profiles`.

### Cambios

| Archivo | Línea aprox | Cambio | LOC |
|---|---|---|---|
| `tools.py` | nuevo | Helper `_fetch_loading_params(goal, level)` → dict `{(week, role): {sets, reps, rir, rest_sec, tempo}}` | ~25 |
| `tools.py` | nuevo | Helper `_fetch_exercise_roles(exercise_ids)` → dict `{exercise_id: role}` | ~15 |
| `tools.py` | 828-852 | Reemplazar `ex.get("sets", 3)` etc. por lookup en los helpers según `(week, role)` | ~30 |
| `prompts.py` | 124 | Eliminar `sets, reps, rir, rest_seconds, tempo` del ejemplo del draft (el LLM no debe inventarlos) | ~5 |

**Total ~75 LOC.**

### Pseudo-código del cambio en `save_workout_plan`

```python
# DESPUÉS de _resolve_exercise_ids, antes del loop for week in range(1, 5):

profile_index = await _fetch_loading_params(goal, draft.get("level", "Intermedio"))
exercise_roles = await _fetch_exercise_roles(list(set(resolved.values())))

DEFAULT_PARAMS = {"sets": "3", "reps": "8-12", "rir": "1-2", "rest_sec": 120, "tempo": "2-0-1"}

for week in range(1, 5):
    for d_idx, day in enumerate(draft.get("days", [])):
        day_title = day.get("title", ...)
        for e_idx, ex in enumerate(day.get("exercises", [])):
            ex_id = resolved.get((d_idx, e_idx))
            if not ex_id:
                continue

            role = exercise_roles.get(ex_id, "compound")
            params = profile_index.get((week, role), DEFAULT_PARAMS)

            workout_rows.append({
                "id": str(uuid.uuid4()),
                "user_id": user_id,
                "week": week,
                "day_name": day_title,
                "exercise_id": ex_id,
                "sets": params["sets"],
                "reps": params["reps"],
                "rir": params["rir"],
                "rest-seconds": params["rest_sec"],
                "tempo": params["tempo"],
                "created_at": now.isoformat(),
                "notes": "",
                "exercise_order": ex.get("exercise_order", ex.get("order", e_idx + 1)),
            })
```

### Fallback strategy

Si `set_profiles` no tiene la combinación (goal, level, week, role) — usar `DEFAULT_PARAMS` y emitir warning a Cloud Run logs:
```python
if (week, role) not in profile_index:
    logger.warning(f"[LOADING] No set_profile for goal={goal} level={level} week={week} role={role} — using defaults")
```

---

## 3. Pre-requisitos antes de implementar

- [ ] **Verificar enum normalization (B-S02-002)** — si `training_experience` se guarda mal ("1 año" → "Principiante"), `set_profiles` lookup va a devolver valores incorrectos. Idealmente arreglar primero o coordinar.
- [ ] **Verificar cobertura de `set_profiles`** — confirmar que para cada combinación válida de (goal, level, week, role) hay al menos un row. Query:
  ```sql
  SELECT goal, level, COUNT(DISTINCT week) AS weeks, COUNT(DISTINCT role) AS roles
  FROM set_profiles GROUP BY goal, level ORDER BY goal, level;
  ```
  Esperado: cada combinación con ≥4 weeks × ≥4 roles (compound, isolation, core, cardio).

---

## 4. Implementación (4 commits)

### Commit 1 — Add `_fetch_loading_params` helper
- Función pura, async.
- Tests unitarios mockeando `supabase_query`.

### Commit 2 — Add `_fetch_exercise_roles` helper
- Función pura, async.
- Tests unitarios.

### Commit 3 — Wire helpers into `save_workout_plan`
- Refactor del loop.
- Test unitario integrado (mockeando ambos helpers + supabase_insert).

### Commit 4 — Update prompt template
- [prompts.py:124](langgraph-skeleton/cases/case6_unified_agent/prompts.py#L124) — simplificar el ejemplo:
  ```json
  {"exercise_id":"VALOR_DEL_TOOL","exercise_order":1}
  ```
- Agregar nota explícita en el prompt: "Los sets, reps, RIR, rest y tempo se calculan automáticamente. No los incluyas en el JSON."

---

## 5. Test E2E vía WhatsApp Web

Verificación end-to-end con usuario real (Camilo `573506267523`) usando WhatsApp Web.

### Pre-condiciones

1. Deploy a Cloud Run con los 4 commits aplicados. Verificar:
   ```bash
   curl -s https://kairos-agent-148665080566.us-central1.run.app/ | python3 -m json.tool
   ```
2. WhatsApp Web abierto en chat Kai.Ros (tab 1415617601).
3. Supabase MCP listo.

### Setup — Teardown completo de Camilo

```sql
DELETE FROM checkpoint_blobs WHERE thread_id IN ('case6_573506267523','kyc_573506267523');
DELETE FROM checkpoint_writes WHERE thread_id IN ('case6_573506267523','kyc_573506267523');
DELETE FROM checkpoints WHERE thread_id IN ('case6_573506267523','kyc_573506267523');
DELETE FROM draft_routines WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM magic_links WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM workouts WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM user_weekly_schedule WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM users_plans WHERE user_id = (SELECT user_id FROM users WHERE full_phone_number = 573506267523);
DELETE FROM users_gym_profile WHERE whatsapp_id = 573506267523;
DELETE FROM users WHERE full_phone_number = 573506267523;
```

Verificar wipe: `SELECT COUNT(*) FROM users WHERE full_phone_number=573506267523;` → `0`

### Test Case 1: Avanzado + Ganar masa muscular (caso del bug original)

#### Pasos en WhatsApp Web

| # | Mensaje al bot | Esperar |
|---|----------------|---------|
| 1 | `Hola, quiero empezar a entrenar` | ~15s. Bot inicia KYC. |
| 2 | `Quiero ganar masa muscular. Llevo más de 3 años entrenando. Voy a un gimnasio bien equipado.` | ~17s. Bot pide días+lesiones. |
| 3 | `Tengo 3 días disponibles, ninguna lesión, todo bien` | ~40s. Bot registra usuario + genera draft. |
| 4 | `Me gusta, guárdala` | ~20s. Bot llama `save_workout_plan`. |

#### Verificación crítica — DB después del save

```sql
-- Q1: Confirmar nivel guardado
SELECT fitness_level, training_experience FROM users_gym_profile
WHERE whatsapp_id = 573506267523;
```

**Esperado:** `fitness_level='Avanzado'`, `training_experience='Más de 3 años'`.

(Si NO es 'Avanzado', el bug B-S02-002 contamina el test → arreglar enum primero.)

```sql
-- Q2: Verificar periodización por semana × role (la prueba CLAVE del fix)
SELECT
  w.week,
  e.role,
  w.sets,
  w.reps,
  w.rir,
  w."rest-seconds" AS rest_sec,
  w.tempo,
  COUNT(*) AS n_exercises
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = (SELECT user_id FROM users WHERE full_phone_number=573506267523)
GROUP BY w.week, e.role, w.sets, w.reps, w.rir, w."rest-seconds", w.tempo
ORDER BY w.week, e.role;
```

**Esperado** (de `set_profiles` para Avanzado + Ganar masa muscular):

| week | role | sets | reps | rir | rest_sec | tempo |
|------|------|------|------|-----|----------|-------|
| 1 | compound | **4** | 6–10 | 2 | 180 | 2-0-1 |
| 1 | isolation | **4** | 10–15 | 1–2 | 90 | 2-0-2 |
| 1 | core | 3–4 | 8–15 | 2 | 60 | — |
| 2 | compound | **4–5** | 5–8 | 1–2 | 210 | 2-0-1 |
| 2 | isolation | **4–5** | 10–12 | 0–1 | 90 | 2-0-2 |
| 3 | compound | **5** | 4–6 | 0–1 | 240 | 2-0-1 |
| 3 | isolation | **5** | 8–12 | 0–1 | 90 | 2-0-2 |
| 4 | compound | **2–3** | 6–10 | 4 | 180 | 2-0-2 |
| 4 | isolation | **2–3** | 10–15 | 4 | 75 | 2-0-2 |

**Criterios de éxito (PASS):**
- ✅ `sets` para week=1 compound = `"4"` (NO `"3"` como en el bug)
- ✅ `sets` para week=3 compound = `"5"` (ramp-up confirmado)
- ✅ `sets` para week=4 compound contiene `"2"` o `"3"` (deload confirmado)
- ✅ `rir` varía por semana: W1=`2`, W3=`0–1`, W4=`4`
- ✅ `rest-seconds` aumenta de W1 (180) a W3 (240)
- ✅ Para cada (week, role) la combinación de valores coincide con `set_profiles`

#### Verificación en Cloud Run logs

```bash
gcloud logging read 'resource.type=cloud_run_revision
  AND resource.labels.service_name=kairos-agent
  AND textPayload:"LOADING"' --freshness 5m --limit 20
```

**Esperado:** **0 warnings** de `[LOADING] No set_profile for ...`. Si aparecen → falta cobertura en `set_profiles`.

### Test Case 2: Principiante (verificar que carga sea conservadora)

#### Setup
- Teardown completo (mismo SQL que arriba).

#### Pasos en WhatsApp Web

| # | Mensaje | Esperar |
|---|---------|---------|
| 1 | `Hola, quiero empezar a entrenar pero soy totalmente nuevo` | — |
| 2 | `Quiero ganar músculo. Nunca he entrenado. 3 días en gimnasio. Sin lesiones.` | — |
| 3 | `Me gusta, guárdala` | — |

#### Verificación

```sql
SELECT DISTINCT e.role, w.week, w.sets, w.reps, w.rir, w."rest-seconds"
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = (SELECT user_id FROM users WHERE full_phone_number=573506267523)
  AND e.role = 'compound'
ORDER BY w.week;
```

**Esperado** (de `set_profiles` para Principiante + Ganar masa muscular — confirmar query antes):
- Sets debería ser **menor** que Avanzado (probable 2-3 en lugar de 4-5)
- Reps mayor (10-15) — más volumen, menos intensidad
- RIR mayor (3+) — más lejos del fallo

**Criterio:** los valores de Principiante deben ser **distintos** de los de Avanzado del Test Case 1.

### Test Case 3: Goal switch (Bajar grasa)

#### Setup
- Teardown completo.

#### Pasos
| # | Mensaje |
|---|---------|
| 1 | `Hola, quiero bajar grasa y tonificar` |
| 2 | `Intermedio, 1 año entrenando, 3 días en gimnasio, sin lesiones` |
| 3 | `Apruebala` |

#### Verificación

```sql
SELECT primary_goal FROM users_gym_profile WHERE whatsapp_id = 573506267523;
-- Esperado: 'Bajar grasa'

SELECT DISTINCT w.week, e.role, w.sets, w.reps, w."rest-seconds"
FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523)
ORDER BY w.week, e.role;
```

**Esperado** (cargas "Bajar grasa" típicamente difieren de "Ganar masa muscular"):
- Reps más altos (12-20)
- Rest más corto (45-90s)
- Más circuit-style
- Comparar contra `SELECT * FROM set_profiles WHERE goal='Bajar grasa' AND level='Intermedio'`

**Criterio:** los valores deben ser distintos de los de Test Case 1 (`Ganar masa muscular`).

### Test Case 4: Regresión — health filter sigue funcionando

#### Setup
- Teardown completo.

#### Pasos
| # | Mensaje |
|---|---------|
| 1 | `Hola, quiero ganar músculo pero me lesioné el hombro y no puedo hacer ejercicios por encima de la cabeza` |
| 2 | `Más de 3 años, 3 días en gimnasio` |
| 3 | `Listo` |

#### Verificación

```sql
-- No debe haber push_v (regresión de B-S04-001)
SELECT COUNT(*) FROM workouts w JOIN exercises e ON w.exercise_id=e.exercise_id
WHERE w.user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523)
  AND e.pattern = 'push_v';
-- Esperado: 0

-- Pero los sets deben seguir siendo de Avanzado (no afectado por health)
SELECT DISTINCT w.sets, w.week FROM workouts w JOIN exercises e ON w.exercise_id=e.exercise_id
WHERE w.user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523)
  AND e.role='compound'
ORDER BY w.week;
-- Esperado: W1=4, W2=4-5, W3=5, W4=2-3
```

---

## 6. Reporte del E2E test

Crear archivo `langgraph-skeleton/testing-reports/FIX-VERIFY-B-S01-005-YYYY-MM-DD.md` con el resultado:

```markdown
# Verificación Fix B-S01-005

**Fecha:** YYYY-MM-DD
**Cloud Run revision:** kairos-agent-NNNNN-xxx
**Commits aplicados:** SHA1, SHA2, SHA3, SHA4

## Resultados

| Test Case | Resultado | Detalle |
|---|---|---|
| 1 - Avanzado/Hipertrofia | ✅/❌ | W1 compound sets=X (esperado 4) |
| 2 - Principiante | ✅/❌ | ... |
| 3 - Bajar grasa | ✅/❌ | ... |
| 4 - Health C + Avanzado | ✅/❌ | ... |

## Snapshot DB (Test Case 1, query Q2)

[Pegar resultado de Q2 acá]

## Logs Cloud Run (warnings de loading)

[Pegar output de gcloud logging acá]

## Conclusión

[PASS / FAIL / PASS con notas]
```

---

## 7. Rollback plan

Si el e2e test falla:

1. **Revertir los 4 commits** (`git revert HEAD~3..HEAD`).
2. Re-deploy a Cloud Run: `gcloud run deploy kairos-agent --source langgraph-skeleton --region us-central1`.
3. Verificar con smoke test (S01 turn 1 en WhatsApp).
4. Documentar el fallo en `FIX-VERIFY-B-S01-005-YYYY-MM-DD.md` y crear ticket de seguimiento.

---

## 8. Definition of Done

- [ ] Los 4 commits mergeados en `main`
- [ ] Deploy a Cloud Run exitoso (revision READY)
- [ ] Test Case 1 PASS (Avanzado recibe 4-5 sets con periodización)
- [ ] Test Case 2 PASS (Principiante recibe carga conservadora)
- [ ] Test Case 3 PASS (Bajar grasa cambia los valores)
- [ ] Test Case 4 PASS (health filter intacto)
- [ ] 0 warnings `[LOADING]` en Cloud Run logs durante e2e
- [ ] Reporte `FIX-VERIFY-B-S01-005-YYYY-MM-DD.md` creado y commit
- [ ] CHANGELOG.md actualizado con la entrada del fix
