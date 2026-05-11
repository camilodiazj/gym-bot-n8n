# S04 — New User Health Status C (sin overhead pressing) — ✅ PASS

**Fecha:** 2026-05-11
**Tester:** Camilo (`573506267523`)
**Resultado:** ⚠️ PASS WITH ISSUES (filtro funcionó, pero arrastra bugs de S01/S02)

## Pre-condiciones
- [x] Teardown ejecutado

## Conversación

| # | Tú enviaste | Bot respondió | Latencia | Tools fired | Notas |
|---|-------------|---------------|----------|-------------|-------|
| 1 | `Hola! Quiero entrenar pero tengo un problema en el hombro izquierdo, mi médico me dijo que evite ejercicios por encima de la cabeza.` | "Entendido perfectamente, **la salud es lo primero y vamos a cuidar ese hombro evitando cualquier empuje vertical**. 3 preguntas: objetivo, experiencia, días+lugar" | ~17s | (none) | ✅ Bot reconoció restricción explícitamente |
| 2 | `Ganar músculo. Intermedio, 1 año entrenando. 3 días en gimnasio bien equipado.` | (procesa, registra, genera draft) | ~50s | `register_new_user` (health_status=C), `save_draft_preview` | |
| 3 | `perfecto, guárdala` | "Plan guardado..." | ~25s | `save_workout_plan` | |

## Verificación DB

### Profile
```json
{
  "health_status": "C",                  ← ✅ guardado correctamente
  "training_environment": "GYM",         ← ✅
  "primary_goal": "Ganar masa muscular", ← ✅
  "training_experience": "Menos de 6 meses",  ← ❌ usuario dijo "Intermedio, 1 año" (B-S02-002 confirmado)
  "fitness_level": "Principiante"        ← ❌
}
```

### Distribución de patterns (4 semanas)

| Pattern | Uses | Notas |
|---|---|---|
| core | 12 | OK (1 por día × 12 sesiones) |
| **push_h** | 12 | Push horizontal (bench, fly) |
| arm | 8 | Isolation brazos |
| hinge | 8 | Peso muerto / RDL |
| pull_v | 8 | Jalón / dominadas (pull vertical OK, va PARA ABAJO no para arriba) |
| squat | 8 | Sentadillas |
| pull_h | 4 | Remos (poco) |
| **push_v** | **0** | ✅ **CORRECTO — Filtro de health C funciona** |

**Query de verificación:**
```sql
SELECT e.spanish_name, e.pattern FROM workouts w JOIN exercises e ON w.exercise_id=e.exercise_id
WHERE w.user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523)
  AND (e.pattern = 'push_v' OR LOWER(e.spanish_name) ~ 'militar|overhead|hombro|vertical');
```
→ **0 filas** ✅

## Bugs

### B-S04-001 — Desbalance push_h vs pull_h: 12 vs 4 (3:1) **[P2]**

Para hipertrofia el ratio ideal push:pull es 1:1 o levemente push:pull = 1:1.2. Tener 3x más push horizontal que pull horizontal arriesga desbalance muscular en hombros y postura.

**Hipótesis:** Al excluir push_v, el algoritmo "compensa" agregando más push_h, pero no balancea con más pull_h.

### B-S04-002 — Mismo bug que B-S02-002: KYC mapeo experiencia incorrecto

"Intermedio, 1 año" → `training_experience: "Menos de 6 meses"`, `fitness_level: "Principiante"`.
Confirmado que es bug **reproducible** (apareció también en S02).

### B-S04-003 — Misma observación que S01: campos NULL en KYC

biological_sex, age, weight_kg, height_cm, email = NULL. Confirma B-S01-001.

## Por Mejorar

- [ ] **Balancear push_h ↔ pull_h:** Cuando push_v se excluye por health C, agregar 1-2 ejercicios de pull_h extra para no desbalancear (filas o face-pulls).
- [ ] **Bot mencionar la adaptación:** "Como cuidamos tu hombro, agregué [face pull] para mejorar la salud postural posterior" — educar al usuario sobre la lógica.
- [ ] **Validar coherencia post-save:** Re-correr `_apply_health_filter` después del save y abortar si hay falso negativo.

## Nice To Have

- [ ] Visualización en el draft preview de "ejercicios omitidos por tu condición": "No incluí press militar porque dijiste que evitas overhead. Si quieres reactivarlo, dime."
- [ ] Programa de rehabilitación de hombro paralelo: "Mientras tanto, aquí 3 ejercicios de movilidad/rehab para hombro" (separado del workout principal).

---

**Conclusión:** **El filtro de health funciona** — buen indicador de que el bug de equipment (S02) NO es un problema generalizado de todos los filtros, sino específico de equipment. Volver a `_apply_health_filter()` en `tools.py:111-166` como modelo correcto.
