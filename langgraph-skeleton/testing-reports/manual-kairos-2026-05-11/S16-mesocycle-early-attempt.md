# S16 — Mesocycle Renewal: Intento Prematuro — ✅ PASS

**Fecha:** 2026-05-11
**Tester:** Camilo (`573506267523`)
**Resultado:** ✅ PASS — Bot rechaza correctamente

## Pre-condiciones
- [x] Usuario con plan ppl_5 aprobado (mesociclo 2, 68 workouts)
- [x] 0 sesiones completadas en mesociclo actual

## Conversación

| # | Tú enviaste | Bot respondió | Tools | Status |
|---|-------------|---------------|-------|--------|
| 1 | `aprueba la rutina` (del draft de S14) | "Tu nueva rutina de 5 días para el Mesociclo #2 ya quedó guardada y activada. ¿Qué días esta semana vas a entrenar?" | `save_workout_plan` | OK |
| 2 | `Quiero renovar mi rutina otra vez, hacer otro mesociclo nuevo` | "Camilo, **veo que apenas estamos empezando el Mesociclo #2 y todavía no has completado las sesiones de la semana 4.** Normalmente renovamos el plan al terminar las 4 semanas... ¿Hay alguna razón especial? (te aburriste, no te gusta algún ejercicio, cambiaste de objetivo)" | `get_mesocycle_status` | ✅ Rechazó |

## Verificación DB

| Métrica | Esperado | Actual |
|---|---|---|
| `users_plans.mesocycle_number` | 2 (sin cambio) | 2 ✅ |
| `users_plans.week_schedule` | ppl_5 (sin cambio) | ppl_5 ✅ |
| `workouts` count | 68 (sin tocar) | 68 ✅ |

## Observaciones positivas

- ✅ Bot consulta `get_mesocycle_status` antes de invocar cualquier `renew_*`
- ✅ Bot **rechaza** con explicación pedagógica (no solo "no se puede")
- ✅ Bot explora alternativas en lugar de bloquear: "¿te aburriste, no te gusta ejercicio, cambiaste objetivo?"
- ✅ Datos íntegros — nada se borra ni modifica

## Bugs

Ninguno.

## Por Mejorar

- [ ] **Sugerencia proactiva:** "Si no te gusta un ejercicio, puedo cambiártelo (find_exercise_alternatives) sin renovar. ¿Cuál no te gusta?" — invita a acción concreta.
- [ ] **Métricas de progreso visibles:** "Llevas 0 de 12 sesiones del mesociclo. Te quedan ~4 semanas. ¿Necesitas ayuda con algún día?"
- [ ] **Mostrar diff con el mesociclo anterior:** "En este mesociclo subimos los sets de 3 a 4 en compounds. Aún no lo has probado". Útil para que el usuario entienda la progresión que va a perder si renueva ya.

## Nice To Have

- [ ] **Permitir override consciente:** Si el usuario insiste ("sí, igual quiero renovar"), permitirle pero pedir confirmación explícita: "Estás seguro? Esto borrará la rutina actual y perderás la progresión que llevas. Escribe 'CONFIRMAR' para proceder."
- [ ] **Detectar "no me gusta" → ofrecer S05** (modificar draft) en lugar de renovar full.

---

**Conclusión:** Combinada con S14 (cambio explícito con cantidad), confirma que **la lógica de renewal funciona bien cuando hay precondiciones claras** (mencion de "completé 4 semanas" o cambio de cantidad). El bug P0 de S10 es **exclusivamente de tool selection ante input ambiguo**.
