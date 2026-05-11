# S02 — New User HOME Bodyweight only — ❌ FAIL P0

**Fecha:** 2026-05-11
**Tester:** Camilo (`573506267523`)
**Resultado:** ❌ **FAIL — Equipment filter no funciona**

## Pre-condiciones
- [x] Teardown ejecutado (verify: users=0)

## Conversación

| # | Tú enviaste | Bot respondió (resumen) | Latencia | Tools fired | Notas |
|---|-------------|--------------------------|----------|-------------|-------|
| 1 | `Hola! Quiero entrenar pero solo tengo peso corporal en casa, sin pesas ni nada.` | "Entrenar en casa con el propio peso es una excelente forma. 3 preguntas: objetivo, experiencia+días, lesiones" | ~20s | (none) | KYC inició, solo 3 preguntas |
| 2 | `Ganar músculo. Intermedio, llevo 1 año entrenando. Puedo 3 días a la semana. Sin lesiones.` | "He adaptado los ejercicios para que puedas hacerlos en casa (como flexiones, fondos en silla y remos invertidos). Link draft: `/draft?c=020a21`" | ~50s | `register_new_user`, `save_draft_preview` | **Bot dice "flexiones, fondos, remos invertidos" pero los guarda con máquinas** |
| 3 | `Me gusta, déjala fija` | (a procesar) | ~20s | `save_workout_plan`, `create_magic_link` | |

## Verificación DB

### users_gym_profile guardado

```json
{
  "primary_goal": "Ganar masa muscular",                ← OK
  "training_environment": "HOME",                       ← OK
  "home_equipment": "peso corporal",                    ← OK
  "days_available": 3,                                  ← OK
  "health_status": "A",                                 ← OK

  "training_experience": "Menos de 6 meses",            ← ❌ usuario dijo "Intermedio, 1 año"
  "fitness_level": "Principiante",                       ← ❌ usuario dijo "Intermedio"
  
  "biological_sex": null, "age": null, "weight_kg": null, "height_cm": null  (NULL — esperado, ya conocido de S01)
}
```

### Ejercicios guardados (15 ejercicios, semana 1)

| Día | Ejercicio | Equipment | ¿OK con bodyweight? |
|---|---|---|---|
| FB A | Sentadilla con mancuerna al pecho | **Mancuerna** | ❌ |
| FB A | Máquina de pecho / Chest press | **Máquina** | ❌ |
| FB A | Remo en máquina | **machine** | ❌ |
| FB A | Abs (General) | bodyweight | ✅ |
| FB B | Peso muerto rumano con banda | **resistance_band** | ❌ (usuario no tiene bandas) |
| FB B | Dominadas en máquina | **machine** | ❌ |
| FB B | Fondos Asistidos en Caja (Con Peso Corporal) | bodyweight | ✅ |
| FB B | Plancha | Peso Corporal | ✅ |
| FB B | Isométrico de Curl de Isquiotibiales de Pie con Peso Corporal | bodyweight | ✅ |
| FB C | Prensa a una pierna | **machine** | ❌ |
| FB C | Jalón con el triángulo | **machine** | ❌ |
| FB C | Abs (General) | bodyweight | ✅ |
| FB C | Femoral acostado | **machine** | ❌ |
| FB C | Peck deck / Mariposa | **Máquina** | ❌ |

**Conteo:**
- ✅ Compatibles con bodyweight: 5/15 (33%)
- ❌ Incompatibles: 10/15 (**67%**)

## Bugs

### B-S02-001 — Equipment filter no funciona para usuarios HOME bodyweight **[P0 BLOQUEANTE]**

**Severidad:** P0 — Usuario sin gym recibe rutina inejecutable

**Repro:**
1. Wipe usuario
2. WhatsApp: `Hola! Quiero entrenar pero solo tengo peso corporal en casa, sin pesas ni nada`
3. Responder con objetivo/experiencia/días sin lesiones
4. Bot genera draft → aprobar → save_workout_plan
5. Inspeccionar `workouts` tabla

**Esperado:** Todos los `workouts.exercise_id` deben resolverse a ejercicios con `equipment IN ('bodyweight', 'Peso Corporal')`.

**Actual:** 67% de los ejercicios guardados requieren máquinas (chest press, prensa, jalón, peck deck, femoral) o mancuernas o bandas elásticas. **Usuario no puede ejecutar la rutina en casa con solo peso corporal.**

**Evidencia:** Ver tabla de equipment arriba.

**Hipótesis de causa:**
1. `get_exercises_for_draft` en `langgraph-skeleton/cases/case6_unified_agent/tools.py` posiblemente no filtra por `equipment` de forma estricta.
2. La función `_normalize_equipment(raw_equipment)` (líneas 56-71 según grep anterior) puede no estar siendo invocada antes del query.
3. El system prompt no instruye al LLM a forzar el filtro de equipment al pedir ejercicios.
4. El draft_data fue generado con todos los ejercicios pero al hacer save no se respetó el filtro.

**Inconsistencia adicional:** El bot CONVERSACIONALMENTE dice "flexiones, fondos en silla, remos invertidos" — esto NO coincide con lo que guarda. Sugiere que el LLM "hablar bonito" pero el tool execution lo ignora.

**Impacto:** Usuario abandona inmediatamente. La promesa "te armo rutina en casa" no se cumple.

### B-S02-002 — KYC ignora explícito "Intermedio, 1 año" → mapea a "Menos de 6 meses / Principiante" **[P1]**

**Repro:**
- Usuario dice: `Intermedio, llevo 1 año entrenando`
- DB guarda: `training_experience: "Menos de 6 meses"`, `fitness_level: "Principiante"`

**Esperado:**
- `training_experience: "6 a 12 meses"` o `"1 a 3 años"` (1 año está en frontera; "6 a 12 meses" es lo más cercano)
- `fitness_level: "Intermedio"` (textual del usuario)

**Hipótesis:**
- `_normalize_enum()` en `tools.py` puede tener fallback default a "Menos de 6 meses" si no encuentra match exacto.
- O el LLM ignoró "Intermedio" y solo parseó la palabra "año" como bandera.

**Impacto:** El usuario que se identifica como Intermedio recibe carga de Principiante → progreso subóptimo. Inversamente, para `set_profiles` se usa `level` así que afecta sets/reps/rir asignados.

### B-S02-003 — El bot describe ejercicios diferentes de los que guarda **[P1]**

El bot dice: *"He adaptado los ejercicios para que puedas hacerlos en casa (como flexiones, fondos en silla y remos invertidos)"*

Pero los guardados son: chest press, prensa a una pierna, jalón con triángulo. NO flexiones, fondos en silla ni remos invertidos.

**Hipótesis:** El LLM genera texto descriptivo independiente del tool execution. Hay desconexión entre lo que el LLM "imagina" y lo que `save_workout_plan` realmente guarda.

## Por Mejorar

- [ ] **Filtro estricto en `get_exercises_for_draft`:** Forzar `WHERE equipment IN (...)` basado en `home_equipment` antes de devolver candidatos al LLM.
- [ ] **Validación post-save:** Después de `save_workout_plan`, validar que todos los `exercise_id` resueltos respetan el equipment del usuario. Si no, abortar y reintentar.
- [ ] **Verificación de coherencia LLM ↔ tool:** Tras guardar, el bot debería leer de vuelta los nombres reales guardados y reportar esos (no inventar nombres).
- [ ] **Whitelist de ejercicios bodyweight:** Mantener una lista curada de ~30 ejercicios garantizados bodyweight (flexiones, fondos diamond, pistol squats, australian rows, mountain climbers, etc.) para fallback.

## Nice To Have

- [ ] **Detector de equipment en lenguaje natural:** "Tengo mancuernas, bandas y barra de dominadas en la puerta" → parsear a `home_equipment = "mancuernas, bandas, barra_dominadas"`.
- [ ] **Foto del lugar:** Si el usuario sube foto del lugar donde entrena, detectar equipment automáticamente con vision.
- [ ] **Sugerir compras:** "Con una banda elástica más podrías hacer 5 ejercicios más. ¿Te interesa?"

---

**Estado tras S02:** **Bloqueante para HOME users**. Más grave aún que S10 porque afecta directamente el caso de uso prometido (rutina en casa).
