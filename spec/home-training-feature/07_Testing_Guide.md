# Guía de Pruebas - HOME Training Feature

Este documento explica cómo probar la funcionalidad de entrenamiento en casa (HOME) implementada en GymBot.

---

## 1. Usuarios de Prueba

Se crearon 5 usuarios de prueba con diferentes configuraciones HOME:

| Teléfono | Nombre | Equipamiento | Nivel | Health | Propósito |
|----------|--------|--------------|-------|--------|-----------|
| `570000000200` | Test_Home_Basic | mancuernas, bandas | Principiante | A | Equipamiento básico |
| `570000000201` | Test_Home_Full | mancuernas, barra, banco, kettlebells, barra dominadas, bandas | Intermedio | A | Equipamiento completo |
| `570000000202` | Test_Home_Advanced | mancuernas, barra, banco, rack, kettlebells, barra dominadas | Avanzado | A | Setup avanzado |
| `570000000203` | Test_Home_HealthC | mancuernas, bandas | Intermedio | C | Restricción upper body |
| `570000000204` | Test_Home_Bodyweight | peso corporal | Principiante | A | Solo bodyweight |

> **Nota**: Los teléfonos `57000000020X` están reservados para pruebas HOME. No usar para usuarios reales.

---

## 2. Pruebas de Flujo KYC (FASE 6.5)

### 2.1 Verificar Captura de Ambiente

**Escenario**: Nuevo usuario inicia onboarding

**Pasos**:
1. Enviar mensaje desde número nuevo (ej: `570000000299`)
2. Completar KYC hasta FASE 6 (health_status)
3. En FASE 6.5, el bot debe preguntar:
   > "¿Dónde vas a entrenar principalmente?"
4. Responder con una de las opciones:
   - "En casa" → `training_environment = HOME`
   - "En el gimnasio" → `training_environment = GYM`

**Validación**:
```sql
SELECT training_environment, home_equipment
FROM users_gym_profile
WHERE whatsapp_id = '570000000299';
```

### 2.2 Verificar Captura de Equipamiento (Solo HOME)

**Escenario**: Usuario selecciona HOME

**Pasos**:
1. Después de seleccionar "En casa", el bot pregunta:
   > "¿Qué equipamiento tienes disponible en casa?"
2. Responder con lista (ej: "Tengo mancuernas, una barra y bandas elásticas")

**Validación**:
```sql
-- Debe guardar equipamiento normalizado
SELECT home_equipment
FROM users_gym_profile
WHERE whatsapp_id = '570000000299';
-- Esperado: "mancuernas, barra, bandas"
```

### 2.3 Verificar Bypass para GYM

**Escenario**: Usuario selecciona GYM

**Pasos**:
1. Seleccionar "En el gimnasio"
2. El bot NO debe preguntar por equipamiento
3. Debe continuar a FASE 7 (días disponibles)

**Validación**:
```sql
SELECT training_environment, home_equipment
FROM users_gym_profile
WHERE whatsapp_id = '570000000XXX';
-- Esperado: training_environment='GYM', home_equipment=NULL
```

---

## 3. Pruebas de Generación de Rutina

### 3.1 Rutina HOME con Equipamiento Básico

**Usuario**: `570000000200` (Test_Home_Basic)

**Verificar**:
1. Template asignado debe ser `_home`:
```sql
SELECT rt.template_id, rt.environment
FROM users_plans up
JOIN routine_templates rt ON up.template_id = rt.template_id
WHERE up.user_id = (SELECT user_id FROM users WHERE full_phone_number = '570000000200');
-- Esperado: environment = 'HOME'
```

2. Ejercicios deben ser compatibles con equipamiento:
```sql
SELECT w.day_name, e.spanish_name, e.equipment
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = (SELECT user_id FROM users WHERE full_phone_number = '570000000200')
ORDER BY w.day_name, w.exercise_order;
-- Verificar: equipment IN ('dumbbell', 'resistance_band', 'bodyweight')
```

### 3.2 Rutina HOME Solo Bodyweight

**Usuario**: `570000000204` (Test_Home_Bodyweight)

**Verificar**:
```sql
SELECT DISTINCT e.equipment
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = (SELECT user_id FROM users WHERE full_phone_number = '570000000204');
-- Esperado: SOLO 'bodyweight'
```

### 3.3 Rutina HOME con Restricción de Salud

**Usuario**: `570000000203` (Test_Home_HealthC - upper body issues)

**Verificar**:
```sql
-- NO debe haber ejercicios overhead
SELECT e.spanish_name, e.pattern
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = (SELECT user_id FROM users WHERE full_phone_number = '570000000203')
  AND e.pattern IN ('push_v', 'pull_v');
-- Debe estar vacío o solo ejercicios seguros
```

### 3.4 Compensación de Gaps (Pull Vertical)

**Usuario**: `570000000200` (sin barra de dominadas)

**Verificar** que se aplique compensación:
```sql
-- Debe tener más ejercicios pull_h para compensar falta de pull_v
SELECT e.pattern, COUNT(*) as count
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = (SELECT user_id FROM users WHERE full_phone_number = '570000000200')
  AND e.pattern LIKE 'pull%'
GROUP BY e.pattern;
```

---

## 4. Pruebas de Regresión (Usuarios GYM)

### 4.1 Verificar Usuario GYM Existente

**Usuario**: `570000000003` (Test_WithRoutine - usuario GYM existente)

**Verificar** que no se vea afectado:
```sql
-- Template debe seguir siendo GYM
SELECT rt.environment
FROM users_plans up
JOIN routine_templates rt ON up.template_id = rt.template_id
WHERE up.user_id = (SELECT user_id FROM users WHERE full_phone_number = '570000000003');
-- Esperado: environment = 'GYM' (o NULL para templates legacy)
```

### 4.2 Nuevo Usuario GYM

**Pasos**:
1. Crear nuevo usuario de prueba vía KYC
2. Seleccionar "En el gimnasio"
3. Completar onboarding

**Verificar**:
- Template asignado es GYM
- Ejercicios incluyen equipamiento de gimnasio (cable, machine, barbell, etc.)

---

## 5. Pruebas E2E con Workflow

### 5.1 Ejecutar Test Runner

El workflow `GymRatFlow_E2E_TestRunner.json` incluye casos para usuarios existentes. Para probar HOME:

1. Importar workflow en n8n
2. Agregar nuevos test cases para usuarios HOME:

```javascript
// Agregar en node "Define Test Cases"
{
  "test_id": "TC_HOME_001",
  "category": "HOME_BASIC",
  "phone": "570000000200",
  "message": "Hola, quiero ver mi rutina",
  "expected_contains": ["ejercicio", "series", "repeticiones"],
  "description": "Usuario HOME básico ve su rutina"
}
```

### 5.2 Verificar Flujo Completo

**Simular mensaje de usuario HOME**:

```
POST /webhook
{
  "from": "570000000200",
  "body": "Quiero ver mi rutina de hoy"
}
```

**Esperado**:
- Respuesta con rutina formateada
- Ejercicios compatibles con equipamiento del usuario
- Sin ejercicios que requieran equipamiento no disponible

---

## 6. Queries de Validación

### 6.1 Verificar Templates HOME Creados

```sql
SELECT
  COUNT(*) as total,
  environment,
  goal,
  level
FROM routine_templates
WHERE environment = 'HOME'
GROUP BY environment, goal, level
ORDER BY goal, level;
-- Esperado: 75 templates (5 goals × 3 levels × 5 schedules)
```

### 6.2 Verificar Ejercicios Disponibles HOME

```sql
SELECT
  equipment,
  COUNT(*) as exercises_count
FROM exercises
WHERE home_friendly = true
GROUP BY equipment
ORDER BY exercises_count DESC;
-- Esperado: ~825 ejercicios HOME-friendly
```

### 6.3 Verificar parseHomeEquipment Output

Para un usuario específico, verificar el procesamiento:

```sql
SELECT
  home_equipment as raw_input,
  -- El procesamiento se hace en n8n, verificar en logs
  training_environment
FROM users_gym_profile
WHERE whatsapp_id = '570000000201';
```

---

## 7. Escenarios de Error

### 7.1 Equipamiento No Reconocido

**Input**: "tengo unas pesas raras y un aparato extraño"

**Esperado**:
- Sistema debe hacer fallback a bodyweight
- O solicitar clarificación al usuario

### 7.2 Usuario HOME sin Ejercicios Disponibles

**Escenario**: Combinación de equipamiento + health_status que resulta en 0 ejercicios para un patrón

**Verificar**:
- El sistema debe manejar gracefully
- Aplicar compensación de gaps
- No generar rutina vacía

---

## 8. Checklist de Pruebas

| # | Prueba | Estado | Notas |
|---|--------|--------|-------|
| 1 | KYC captura training_environment | ⬜ | |
| 2 | KYC captura home_equipment (solo HOME) | ⬜ | |
| 3 | GYM bypass equipamiento | ⬜ | |
| 4 | Template HOME asignado correctamente | ⬜ | |
| 5 | Ejercicios filtrados por equipamiento | ⬜ | |
| 6 | Bodyweight-only funciona | ⬜ | |
| 7 | Health restrictions aplicadas | ⬜ | |
| 8 | Gap compensation funciona | ⬜ | |
| 9 | GYM users no afectados | ⬜ | |
| 10 | E2E flow completo | ⬜ | |

---

## 9. Contacto

Para problemas con las pruebas, revisar:
- Logs de n8n en los workflows
- Tabla `n8n_chat_histories` para contexto de conversación
- Supabase logs para queries fallidas
