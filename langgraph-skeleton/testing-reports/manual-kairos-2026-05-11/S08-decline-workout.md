# S08 — Decline Workout — ✅ PASS

**Fecha:** 2026-05-11
**Tester:** Camilo (`573506267523`)
**Resultado:** ✅ PASS

## Pre-condiciones
- [x] CHAIN de S05: usuario con plan + 3 sesiones agendadas (Lunes hoy = Full Body A, Miércoles, Viernes)

## Conversación

| # | Tú enviaste | Bot respondió | Latencia | Tools | Notas |
|---|-------------|---------------|----------|-------|-------|
| 1 | `Agenda lunes hoy, miercoles y viernes esta semana` | "Ya quedaron agendadas tus sesiones, como hoy es lunes, empezamos. Full Body A: Sentadilla Hack, Press inclinado con mancuerna, Remo en T, Levantamiento Activo de Banca. Link tracker: /w?c=07ef95" | ~22s | `schedule_sessions`, `get_todays_routine`, `create_magic_link` | Mostró rutina + 1 magic link |
| 2 | `No puedo entrenar hoy, prefiero descansar` | "El descanso también es fundamental. Ya quedó anotado. **¿Quieres que movamos la sesión de hoy para mañana martes, o prefieres simplemente retomar el miércoles?**" | ~18s | `decline_workout` | ✅ Bot ofreció reschedule proactivo |

## Verificación DB

```sql
SELECT week_day, "Completed" FROM user_weekly_schedule WHERE user_id='...' AND planned_day='11/05/2026';
SELECT COUNT(*) FROM pending_tasks WHERE user_id='...';
```

| Métrica | Esperado | Actual |
|---|---|---|
| `user_weekly_schedule.Completed` (hoy) | `false` (sin tocar) | `false` ✅ |
| `pending_tasks` count | 0 | 0 ✅ |
| `user_weekly_schedule` (rows) | 3 (sin cambios) | 3 ✅ |

## Observaciones positivas

- ✅ Bot **NO creó pending_task** (correcto — usuario declinó proactivamente, no es un olvido)
- ✅ Bot **NO marca Completed=true** (correcto)
- ✅ Bot **NO modifica `user_weekly_schedule.planned_day`** automáticamente (espera respuesta del usuario)
- ✅ Tono empático: "El descanso también es fundamental para ganar masa muscular"
- ✅ Ofrece 2 opciones claras: mover sesión a mañana o saltar y seguir miércoles

## Bugs

Ninguno.

## Por Mejorar

- [ ] **Si el usuario contesta "muévela a mañana", verificar que `schedule_sessions` actualice correctamente** sin gatillar el bug P0 de S10. (No probado en este escenario; vale ver en S14.)
- [ ] **Registrar la razón del decline:** El usuario dijo "prefiero descansar" — podría guardarse en una tabla `decline_reasons` o como nota en `user_weekly_schedule` para análisis posterior ("X% de los users declinan los lunes").
- [ ] **Bot podría preguntar la razón:** "¿Todo bien? ¿Fatiga, cosas en casa, dolor?" — útil para detectar overtraining/lesiones tempranamente.

## Nice To Have

- [ ] **Detección de patrón:** Si el usuario declina más de 2 sesiones por mesociclo, sugerir reducir frecuencia.
- [ ] **Notificación: "Estás declinando seguido los lunes, ¿quieres moverlo?"**
- [ ] **Stickers de empatía:** En lugar de solo texto, el bot podría usar emojis más expresivos (☕️ 🛌 💆).
