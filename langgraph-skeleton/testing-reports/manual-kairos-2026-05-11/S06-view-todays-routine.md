# S06 — View Today's Routine (CHAIN S01)

**Fecha:** 2026-05-11
**Tester:** Camilo (`573506267523`)
**Resultado:** ✅ PASS

## Pre-condiciones
- [x] CHAIN al estado S01 (3 sesiones agendadas L/Mi/V, hoy es Lunes = Full Body A)
- [x] Teardown: No

## Conversación

| # | Tú enviaste | Bot respondió | Latencia | Tools fired | Notas |
|---|-------------|---------------|----------|-------------|-------|
| 1 | `qué me toca hoy?` | "Para hoy lunes te corresponde Full Body A: 1. Sentadilla Hack 3x8-10... 2. Press inclinado con mancuerna... 3. Remo en T... 4. Plancha. Link tracker: /w?c=07e739" | ~4s | `get_todays_routine`, `create_magic_link` | Respuesta súper rápida. Listó los 4 ejercicios. |

## Verificación DB

```sql
SELECT session_name, "Completed" FROM user_weekly_schedule
WHERE user_id='6a98bdce-5a09-40ba-885d-fd17998768bc'
  AND planned_day = '11/05/2026';
```
→ Full Body A, Completed=false (sin cambio — view es read-only ✓)

## Bugs

Ninguno.

## Por Mejorar

- [ ] **Magic link 4 creado** (`07e739`) — tercer/cuarto link en esta sesión. Ya hay 3 magic_links acumulados. Sugerir reusar el más reciente válido en lugar de crear uno nuevo cada vez que el usuario pregunta "qué me toca hoy".
- [ ] **Latencia muy buena (4s)** — vs otros turns que tardaron 15-38s. Confirma que view-only es rápido cuando no se llama LLM intensivamente.

## Nice To Have

- [ ] Mostrar el "número de sesión completadas en la semana" en la respuesta. Ej: "Esta es tu sesión 1 de 3 de la semana".
- [ ] Mostrar streak: "Llevas X días sin saltarte un entrenamiento".
