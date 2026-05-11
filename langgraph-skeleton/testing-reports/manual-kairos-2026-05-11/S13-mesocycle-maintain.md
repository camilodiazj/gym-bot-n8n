# S13 — Mesocycle Renewal: Maintain — ✅ PASS

**Fecha:** 2026-05-11
**Tester:** Camilo (`573506267523`)
**Resultado:** ✅ PASS

## Pre-condiciones
- [x] Estado base: S04 (Health C usuario con plan + 60 workouts, schedule vacío)
- [x] **Fixture F3 aplicada:** Inserté 12 schedule rows (4 semanas × 3 días) todas con Completed=true

## Conversación

| # | Tú enviaste | Bot respondió | Latencia | Tools fired | Notas |
|---|-------------|---------------|----------|-------------|-------|
| 1 | `Ya terminé las 4 semanas del plan. Qué sigue ahora?` | "Completar las 4 semanas es un excelente trabajo. **3 opciones:** 1) Mantener rutina, 2) Cambiar días, 3) Rotar ejercicios" | ~25s | `get_mesocycle_status` | ✅ Bot detectó W4 completa, listó 3 opciones bien |
| 2 | `Mantener rutina` | "Vamos a sacarle más provecho a esos mismos movimientos en este nuevo mesociclo #2. Ya reinicié tu plan. Qué días esta semana vas a entrenar?" | ~18s | `renew_maintain` | ✅ Mesociclo incrementado, workouts mantenidos |

## Verificación DB

| Métrica | Antes (post-F3) | Después (post-S13) | Status |
|---|---|---|---|
| `users_plans.mesocycle_number` | 1 | **2** | ✅ incrementado |
| `users_plans.last_renewal_date` | NULL | **2026-05-11 17:05:31** | ✅ set |
| `users_plans.status` | active | active | ✅ |
| `workouts` count | 60 | **60** | ✅ mantenidos |
| `user_weekly_schedule` count | 12 (Completed=true) | **0** | ⚠️ borrado (esperado para re-agendar) |

## Observaciones importantes

### Patrón confirmado en bot: descripción de opción 2 en respuesta turn 1

El bot describió la opción "Cambiar días" como:
> *"Si quieres entrenar **más o menos días** por semana, **armamos una rutina nueva desde cero**."*

**Esto explica el bug P0 de S10.** El system prompt instruye al bot a interpretar "cambiar días" como cambiar la **cantidad** de días (week_schedule fb_3 → ppl_5), no como reschedule de los mismos 3 días.

Cuando en S10 dije "cambiar mis días" para reschedule, el bot ejecutó `renew_change_days` siguiendo esta interpretación. Mismo nombre, intenciones distintas.

**Sugerencia de fix:** Renombrar `renew_change_days` a algo como `renew_change_frequency` o `renew_resize` para distinguir conceptualmente del simple reschedule. O agregar al system prompt una pregunta de clarificación cuando el usuario diga "cambiar días": *"¿Quieres cambiar **cuáles** días entrenas (e.g. Mar/Jue/Sab en lugar de L/Mi/V) o cambiar **cuántos** días entrenas (e.g. 3 → 5 días)?"*

### renew_maintain funciona correctamente

A diferencia de S10, aquí el usuario pidió explícitamente "Mantener rutina" → tool routing correcto → `renew_maintain` se ejecutó como esperado:
- mesocycle_number++
- workouts mantenidos (no se borran)
- user_weekly_schedule limpiado para que el usuario re-agende

## Bugs

Ninguno crítico en S13 mismo. Confirma que la lógica de `renew_*` funciona — el problema es **tool selection** (qué tool elige el LLM para qué query del usuario), no la implementación de los tools.

## Por Mejorar

- [ ] **Distinguir lingüísticamente "cambiar días" vs "cambiar cuántos días":** Como discutido arriba.
- [ ] **Bot dijo "nuevo mesociclo #2"** — buena UX, conserva el conteo de mesociclos para el usuario.
- [ ] **Bot no menciona progresión de carga concreta:** "Mantener" dice "reiniciar el ciclo para buscar nuevas progresiones de carga", pero el bot no actualiza `set_profiles` ni ajusta sets/reps/RIR. Verificar si workouts actualizó las cargas o son las mismas exactas.
- [ ] **Sin resumen de progreso del mesociclo anterior:** "Llevas 12 sesiones completadas, este mesociclo mejoraste X" — útil emocionalmente.

## Nice To Have

- [ ] Comparativa entre mesociclo 1 y 2: "En este mesociclo bajamos el descanso de 150s → 120s para más densidad".
- [ ] Detección de meseta: Si los pesos no subieron en 2 mesociclos, sugerir cambio (rotar) o deload.
- [ ] Notificación push al detectar W4 completa automáticamente: el bot debería ser proactivo, no esperar que el usuario pregunte.

---

**Conclusión:** El flujo de mesocycle renewal **mantener** funciona correctamente. Confirma que los tools `renew_*` están implementados — el problema es de **routing/tool selection** del LLM, no del backend.
