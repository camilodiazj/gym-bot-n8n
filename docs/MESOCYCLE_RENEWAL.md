# Renovación de Mesociclo - GymBot

Este documento describe la funcionalidad de detección y renovación de mesociclo (ciclo de 4 semanas) implementada en GymBot.

## Resumen

Cuando un usuario completa todas las sesiones de entrenamiento de la semana 4, el sistema detecta automáticamente la finalización del mesociclo y ofrece opciones para continuar:

1. **Mantener rutina**: Reinicia el ciclo con los mismos ejercicios
2. **Cambiar rutina**: Permite modificar días por semana y/o rotar ejercicios

---

## Arquitectura

### Nuevo Subflow: `GymBotMesocycleRenewal.json`

Workflow independiente que maneja la conversación de renovación.

```
[Execute Workflow Trigger]
        │
        ▼
[Renewal_Agent] ◄── OpenAI + Postgres Memory
        │
        ▼
[Parse_Intention] ── Extrae intención del mensaje
        │
        ▼
[Switch_Intention]
   ├── MANTENER_RUTINA ──► Reset schedule + Incrementar mesociclo
   ├── CAMBIAR_DIAS ────► Regenerar rutina con GymRatForm
   ├── ROTAR_EJERCICIOS ► Seleccionar ejercicios alternativos
   └── PREGUNTAR_OPCIONES ► Mostrar opciones al usuario
```

### Integración con Flujo Principal

En `GymRatFlow_Supabase.json`:

```
[has_planned_workouts1]
        │ FALSE
        ▼
[Week_Schedule + User_Finished_Workouts + Template_Days]
        │
        ▼
[Merge]
        │
        ▼
[Check_Mesocycle_Complete] ◄── NUEVO
        │
        ▼
[If_Mesocycle_Complete]
   ├── TRUE ──► [Execute_Mesocycle_Renewal] (subflow)
   └── FALSE ─► [AI Agent1] (agendamiento normal)
```

---

## Cambios en Base de Datos

### Tabla: `users_plans`

Nuevas columnas agregadas:

| Columna | Tipo | Default | Descripción |
|---------|------|---------|-------------|
| `mesocycle_number` | INTEGER | 1 | Número de mesociclo actual |
| `last_renewal_date` | TIMESTAMP WITH TIME ZONE | NULL | Fecha de última renovación |

### Migración Aplicada

```sql
ALTER TABLE users_plans
ADD COLUMN mesocycle_number INTEGER DEFAULT 1,
ADD COLUMN last_renewal_date TIMESTAMP WITH TIME ZONE;
```

---

## Flujo de Detección

### Query de Detección

El nodo `Check_Mesocycle_Complete` verifica:

```javascript
const week4Completed = finishedWorkouts.filter(
  w => w.json.week === 4 && w.json.Completed === true
).length;

const mesocycleComplete = week4Completed >= daysPerWeek;
```

### Criterios para Mesociclo Completo

1. Usuario tiene sesiones en `user_weekly_schedule` para semana 4
2. Todas las sesiones de semana 4 tienen `Completed = true`
3. Cantidad de sesiones completadas >= `days_per_week` del plan

---

## Opciones de Renovación

### Opción 1: Mantener Rutina

- Limpia `user_weekly_schedule`
- Incrementa `mesocycle_number` en `users_plans`
- Mantiene los mismos ejercicios en `workouts`
- Usuario debe re-agendar semana 1

```sql
DELETE FROM user_weekly_schedule WHERE user_id = :user_id;

UPDATE users_plans
SET mesocycle_number = mesocycle_number + 1,
    last_renewal_date = NOW()
WHERE user_id = :user_id;
```

### Opción 2a: Cambiar Días por Semana

- Elimina workouts existentes
- Actualiza `week_schedule` en `users_plans`
- Llama a `GymRatForm Supabase` con `is_renewal = true`
- Genera rutina completamente nueva

### Opción 2b: Rotar Ejercicios

- Busca ejercicios alternativos por patrón de movimiento
- Mantiene la estructura de días
- Actualiza `workouts` con nuevos `exercise_id`

```sql
-- Buscar alternativas
SELECT exercise_id
FROM exercises
WHERE pattern = :current_pattern
  AND exercise_id != :current_exercise
ORDER BY RANDOM()
LIMIT 1;
```

---

## Intenciones del Agente

El `Renewal_Agent` detecta estas intenciones:

| Intención | Trigger | Acción |
|-----------|---------|--------|
| `MANTENER_RUTINA` | "mantener", "igual", "repetir" | Reset schedule |
| `CAMBIAR_DIAS` | "X días", "cambiar frecuencia" | Regenerar con GymRatForm |
| `ROTAR_EJERCICIOS` | "nuevos ejercicios", "variar" | Selección aleatoria |
| `PREGUNTAR_OPCIONES` | Preguntas, indecisión | Mostrar opciones |

---

## Integración con GymRatForm

El workflow `GymRatForm Supabase.json` ahora soporta:

### Nuevos Parámetros de Entrada

| Parámetro | Tipo | Descripción |
|-----------|------|-------------|
| `is_renewal` | string | "true" si es renovación |
| `override_days_available` | number | Nuevo número de días (si cambió) |

### Flujo Condicional

```
[GetUser]
    │
    ▼
[If_Is_Renewal]
   ├── TRUE ──► [Clear_Old_Workouts] → [Merge] → Generar nueva rutina
   └── FALSE ─► [UserExists] → Flujo normal de creación
```

---

## Archivos Modificados

| Archivo | Cambios |
|---------|---------|
| `n8n/GymBotMesocycleRenewal.json` | **NUEVO** - Subflow de renovación |
| `n8n/GymRatFlow_Supabase.json` | + Detección de mesociclo completo |
| `n8n/GymRatForm Supabase.json` | + Soporte para `is_renewal` |
| `users_plans` (tabla) | + `mesocycle_number`, `last_renewal_date` |

---

## Diagrama de Flujo Completo

```
┌─────────────────────────────────────────────────────────────────┐
│                    MENSAJE DE USUARIO                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   GetUser       │
                    └────────┬────────┘
                              │
                              ▼
              ┌───────────────────────────────┐
              │   GetWeeklySchedule           │
              └───────────────┬───────────────┘
                              │
                              ▼
              ┌───────────────────────────────┐
              │   has_planned_workouts?       │
              └───────────────┬───────────────┘
                    ┌─────────┴─────────┐
                    │ TRUE              │ FALSE
                    ▼                   ▼
          ┌──────────────┐    ┌─────────────────────┐
          │ Rutina de    │    │ Check_Mesocycle     │
          │ hoy          │    │ Complete            │
          └──────────────┘    └──────────┬──────────┘
                                   ┌─────┴─────┐
                                   │ TRUE      │ FALSE
                                   ▼           ▼
                    ┌──────────────────┐  ┌───────────┐
                    │ Mesocycle        │  │ AI Agent1 │
                    │ Renewal Subflow  │  │ (Agendar) │
                    └────────┬─────────┘  └───────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
              ▼               ▼               ▼
    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
    │ MANTENER    │  │ CAMBIAR     │  │ ROTAR       │
    │ RUTINA      │  │ DÍAS        │  │ EJERCICIOS  │
    └──────┬──────┘  └──────┬──────┘  └──────┬──────┘
           │                │                │
           ▼                ▼                ▼
    Reset schedule   Llamar GymRatForm   Update workouts
           │                │                │
           └────────────────┴────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │ Notificar       │
                    │ al usuario      │
                    └─────────────────┘
```

---

## Testing

### Escenarios de Prueba

1. **Usuario completa semana 4**
   - Completar todas las sesiones de week=4
   - Enviar mensaje → Debe aparecer opciones de renovación

2. **Opción Mantener**
   - Seleccionar "mantener"
   - Verificar: `mesocycle_number` incrementado
   - Verificar: `user_weekly_schedule` vacío
   - Verificar: `workouts` sin cambios

3. **Opción Cambiar Días**
   - Seleccionar "cambiar" → "3 días"
   - Verificar: Nueva rutina generada
   - Verificar: `week_schedule` actualizado

4. **Opción Rotar Ejercicios**
   - Seleccionar "rotar"
   - Verificar: `exercise_id` diferentes en `workouts`

### Queries de Verificación

```sql
-- Ver estado actual del mesociclo
SELECT
  u.full_name,
  up.mesocycle_number,
  up.last_renewal_date,
  ws.days_per_week
FROM users u
JOIN users_plans up ON u.user_id = up.user_id
JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
WHERE u.user_id = :user_id;

-- Contar sesiones completadas por semana
SELECT
  week,
  COUNT(*) as total,
  COUNT(CASE WHEN "Completed" = true THEN 1 END) as completed
FROM user_weekly_schedule
WHERE user_id = :user_id
GROUP BY week
ORDER BY week;
```

---

## Notas de Implementación

- El subflow usa memoria Postgres con session key `{user_id}_mesocycle_renewal`
- La memoria se limpia después de completar la renovación
- El flujo principal detecta la intención `RENOVAR_MESOCICLO` si el usuario lo menciona manualmente
- El mapeo de `week_schedule` para cambio de días:
  - 2 días → `fb_2`
  - 3 días → `fb_3`
  - 4 días → `ua_4`
  - 5 días → `ppl_5`
  - 6 días → `ppl_6`
