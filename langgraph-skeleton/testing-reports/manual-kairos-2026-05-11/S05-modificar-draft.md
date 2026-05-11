# S05 — Modificar Draft (Swap Ejercicio) — ⚠️ PASS WITH ISSUES

**Fecha:** 2026-05-11
**Tester:** Camilo (`573506267523`)
**Resultado:** ⚠️ PASS — Swap se ejecutó pero la alternativa es minimal

## Pre-condiciones
- [x] Teardown ejecutado

## Conversación

| # | Tú enviaste | Bot respondió (resumen) | Latencia | Tools | Notas |
|---|-------------|--------------------------|----------|-------|-------|
| 1 | `Hola! Quiero ganar masa muscular, llevo 2 años entrenando, 3 días en gimnasio, sin lesiones.` | "Ya te registré, rutina de 3 días Full Body, hipertrofia. Link draft `020f67`" | ~50s | `register_new_user`, `save_draft_preview` | KYC respuesta única — bot pudo extraer goal, experience, days, gym, lesions en 1 turno |
| 2 | `No me gusta el press militar, prefiero algo diferente para hombros. Puedes cambiarlo?` | "He cambiado el press militar por **press con mancuernas**, suele ser más cómodo para los hombros. Nuevo link draft `020fbb`" | ~25s | `find_exercise_alternatives`, `save_draft_preview` (nuevo) | El "swap" fue minimal — mismo ejercicio con otro equipment |
| 3 | `Perfecto, guárdala así` | (procesando) | ~25s | `save_workout_plan`, `create_magic_link` | |

## Verificación DB

```sql
SELECT e.spanish_name, e.pattern FROM workouts w JOIN exercises e ON w.exercise_id=e.exercise_id
WHERE w.user_id IN (SELECT user_id FROM users WHERE full_phone_number=573506267523)
AND w.week=1 AND e.pattern='push_v';
```

→ `Full Body B: Press militar con mancuerna (push_v, Shoulders)`

**Comparación:**
- Antes: Press Militar (push_v, equipment=Barra)
- Después: Press militar **con mancuerna** (push_v, equipment=Mancuerna)

**Mismo patrón (push_v), mismo músculo (Shoulders), mismo movimiento. Solo cambió el equipment.**

## Bugs

### B-S05-001 — `find_exercise_alternatives` retorna alternativa "minimal" (mismo ejercicio, diferente equipment) **[P2]**

**Repro:**
- Usuario dice: "No me gusta X, prefiero **algo diferente**"
- Bot encuentra como "alternativa" el mismo ejercicio con otro equipment

**Esperado:** Alternativa con movimiento distinto (e.g. Arnold press, elevaciones laterales, face pulls, landmine press).

**Actual:** Solo cambió "Press militar (barra)" → "Press militar con mancuerna".

**Hipótesis:** `find_exercise_alternatives` filtra por `pattern` exacto y devuelve el primer match — termina siendo el mismo ejercicio con otra herramienta.

**Sugerencia:** En `find_exercise_alternatives`, excluir el `exercise_id` específico anterior y/o priorizar ejercicios con diferente `main_muscle` secundario o diferente nombre base.

### B-S05-002 — Draft se reemplaza con un draft nuevo (otro code) en lugar de actualizarse **[P3]**

El draft original tenía code `020f67`. Después del swap, ese row desapareció y aparece uno nuevo con code `020fbb`.

**Impacto bajo:** funcionalmente equivalente, pero rompe potenciales bookmarks del usuario al draft preview anterior y desorganiza el ID.

**Sugerencia:** Mantener el mismo `draft_id`/`code` y solo actualizar `draft_data` (JSONB) con los cambios.

## Por Mejorar

- [ ] **Mejor descripción de qué cambia:** "Lo cambié por X" sin más contexto. Mejor: "Reemplacé Press Militar (más demandante de hombro) por Arnold Press (más control)." Educar al usuario.
- [ ] **Mostrar 2-3 alternativas en lugar de imponer una:** "Tengo 3 opciones para ti: Arnold press, elevaciones laterales con polea, landmine press. ¿Cuál prefieres?" — empodera al usuario.
- [ ] **Bot dice "press con mancuernas" pero guarda "Press militar con mancuerna"** — micro-inconsistencia textual.

## Nice To Have

- [ ] **Swap múltiples a la vez:** "Cambia el press militar y el remo en T" — el bot debería poder manejar batch.
- [ ] **Recordar preferencias:** Si el usuario sustituyó el press militar en S05, en S15 (rotate ejercicios) no debería volver a aparecer.
- [ ] **Sugerir alternativas proactivamente:** Si el usuario tiene `disliked_exercises = "press militar"` en perfil (a llenar reactivamente), no incluirlo desde el inicio.
