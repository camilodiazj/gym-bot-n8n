# S14 — Mesocycle Renewal: Change Days Explícito — ✅ PASS

**Fecha:** 2026-05-11
**Tester:** Camilo (`573506267523`)
**Resultado:** ✅ PASS

## Pre-condiciones
- [x] Usuario con plan + 60 workouts (de S05)
- [x] Fixture F3 aplicada: 12 sesiones marcadas Completed=true (4 semanas × 3 días)

## Conversación

| # | Tú enviaste | Bot respondió | Tools | Notas |
|---|-------------|---------------|-------|-------|
| 1 | `Ya completé las 4 semanas. Quiero hacer el siguiente mesociclo pero ahora puedo entrenar 5 días a la semana en vez de 3.` | "Subir a 5 días es excelente paso. He diseñado tu nueva rutina de 5 días **(Push/Pull/Legs + Upper/Lower)**. Link draft `0211a3`" | `get_mesocycle_status`, `renew_change_days(5)`, `save_draft_preview` | ✅ Flujo correcto |

## Verificación DB

| Métrica | Antes | Después | Status |
|---|---|---|---|
| `users_plans.mesocycle_number` | 1 | **2** | ✅ |
| `users_plans.week_schedule` | `fb_3` | **`ppl_5`** | ✅ Cambió a 5 días |
| `users_plans.last_renewal_date` | NULL | **now** | ✅ |
| Nuevo draft creado | — | code `0211a3`, pending | ✅ |

## Hallazgo clave: el tool funciona, el problema es tool selection

S14 confirma que `renew_change_days(5)` **se ejecuta correctamente** cuando:
1. El usuario menciona explícitamente la **cantidad** ("5 días en vez de 3")
2. Menciona explícitamente que terminó el mesociclo

vs. S10 donde el usuario dijo "cambiar mis días" (sin cantidad, sin mencionar fin del mesociclo) y aún así el LLM eligió `renew_change_days(3)` (mismos 3 días) — incorrecto.

**Root cause confirmado:** el problema NO está en la implementación de `renew_change_days`, está en:
1. **Tool routing del LLM** — elige `renew_change_days` ante cualquier mención de "cambiar días", aunque la cantidad no cambie
2. **Falta guardia "W4 completo"** — `renew_*` se ejecuta sin verificar precondiciones

## Por Mejorar

- [ ] **Pre-condición de renovación:** Agregar al inicio de `renew_change_days` / `renew_maintain` / `renew_rotate_exercises` una verificación: ¿Todas las semanas del mesociclo actual están completadas? Si no, abortar con error.
- [ ] **Validar que la nueva cantidad es DIFERENTE:** Si `new_days_per_week == current_days`, no llamar `renew_change_days` — debe ser `schedule_sessions` (reschedule).
- [ ] **Bot menciona "Push/Pull/Legs + Upper/Lower" para 5 días** — verificar que `set_profiles` para ppl_5 tenga sets configurados correctamente para esa estructura híbrida.

## Nice To Have

- [ ] **Mensaje de hito:** "¡Acabas de terminar tu primer mesociclo de 12 sesiones! Total volumen: X kg levantados. Promedio set RIR: Y."
- [ ] **Estimación de progreso:** Para usuario en mesociclo 2 con +2 días, "esperamos +1-2 kg de peso en compounds en este mesociclo".

---

**Conclusión:** El backend de mesocycle renewal está sólido. El bug P0 de S10 es **exclusivamente un problema de tool selection** del LLM, fácil de arreglar con un guardrail en el system prompt.
