# Fase 4: Testing y QA - HOME Training Feature

**Documento:** Especificacion Tecnica de Pruebas
**Feature:** Soporte para entrenamiento en casa (HOME environment)
**Asignado a:** code-reviewer (QA)
**Apoyo de:** n8n-agent (ejecutar tests en n8n)
**Fecha:** Febrero 2026

---

## 1. Objetivo

Validar que la feature HOME funciona correctamente end-to-end, asegurando:

1. **KYC Flow**: El usuario puede seleccionar HOME como ambiente de entrenamiento
2. **Equipment Collection**: El sistema recopila correctamente el equipamiento disponible
3. **Routine Generation**: Solo se generan ejercicios compatibles con el equipamiento del usuario
4. **Pattern Compensation**: Los gaps de patrones (pull_v) son compensados correctamente
5. **No Regression**: Los usuarios GYM existentes no se ven afectados

### Criterios de Exito

| Criterio | Umbral |
|----------|--------|
| Test cases pasando | 100% |
| Ejercicios de maquina para usuarios HOME | 0 |
| Cobertura muscular | Todos los grupos principales (8+) |
| Tiempo de generacion de rutina | < 60 segundos |

---

## 2. Test Cases

### 2.1 TC_HOME_001: KYC Flow - Usuario elige HOME

**Categoria:** ONBOARDING_HOME
**Prioridad:** CRITICAL
**Tipo:** MULTI_TURN_AI
**Usuario:** `570000000091` (dinamico)

#### Precondiciones
- Usuario NO existe en la BD
- Phone `570000000091` no tiene registros previos

#### Pasos
1. Usuario envia mensaje inicial: "Hola, quiero empezar a entrenar"
2. KYC Agent inicia flujo de onboarding
3. Durante la conversacion, cuando se pregunte por ambiente:
   - Usuario responde: "En mi casa"
4. KYC Agent pregunta por equipamiento disponible
5. Usuario responde: "Tengo mancuernas y una barra de dominadas"
6. Completar resto del KYC normalmente

#### Datos de Entrada (Usuario Simulado)
```json
{
  "full_name": "Test Home User One",
  "email": "test_home_001@gymbot.test",
  "age": 28,
  "biological_sex": "M",
  "height_cm": 175,
  "weight_kg": 75,
  "primary_goal": "Ganar masa muscular",
  "training_experience": "1 a 3 anos",
  "days_available": 4,
  "session_duration_mins": "45-60 minutos",
  "training_environment": "HOME",
  "home_equipment": ["dumbbell", "pull_up_bar"]
}
```

#### Resultado Esperado
- `users_gym_profile.training_environment = 'HOME'`
- `users_gym_profile.home_equipment` contiene `dumbbell` y `bodyweight`
- Usuario recibe mensaje de confirmacion con rutina HOME

#### Query de Verificacion
```sql
-- Verificar perfil HOME creado
SELECT
    whatsapp_id,
    training_environment,
    home_equipment
FROM users_gym_profile
WHERE whatsapp_id = 570000000091;
-- DEBE retornar: training_environment = 'HOME', home_equipment incluye 'dumbbell'

-- Verificar usuario creado
SELECT COUNT(*) as user_exists
FROM users
WHERE full_phone_number = '570000000091';
-- DEBE retornar: 1

-- Verificar plan con ambiente HOME
SELECT
    up.goal,
    up.level,
    rt.environment
FROM users_plans up
JOIN routine_templates rt ON up.template_id = rt.template_id
WHERE up.user_id IN (
    SELECT user_id FROM users WHERE full_phone_number = '570000000091'
);
-- DEBE retornar: environment = 'HOME'
```

---

### 2.2 TC_HOME_002: KYC Flow - Usuario elige HOME sin especificar equipo

**Categoria:** ONBOARDING_HOME
**Prioridad:** HIGH
**Tipo:** MULTI_TURN_AI
**Usuario:** `570000000091` (reutilizado despues de cleanup)

#### Precondiciones
- Usuario NO existe en la BD (cleanup previo)

#### Pasos
1. Usuario inicia KYC normalmente
2. Cuando se pregunte por ambiente: "En casa"
3. Cuando se pregunte por equipamiento: "No tengo nada" o respuesta vacia
4. **Flujo de re-prompt esperado:**
   - KYC Agent DEBE re-preguntar con opciones especificas
   - Mensaje tipo: "Para entrenar en casa necesitamos saber que tienes. Selecciona las opciones: Mancuernas, Kettlebell, Barra de dominadas, Solo peso corporal"
5. Usuario responde: "Solo peso corporal"
6. Completar KYC

#### Resultado Esperado
- Sistema hace re-prompt cuando equipamiento no es claro
- `home_equipment = ['bodyweight']` (default minimo)
- Rutina generada SOLO con ejercicios bodyweight

#### Query de Verificacion
```sql
-- Verificar solo bodyweight
SELECT home_equipment
FROM users_gym_profile
WHERE whatsapp_id = 570000000091;
-- DEBE retornar: ['bodyweight'] o similar

-- Verificar ejercicios generados son bodyweight
SELECT COUNT(*) as non_bodyweight_exercises
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id IN (
    SELECT user_id FROM users WHERE full_phone_number = '570000000091'
)
AND e.equipment NOT IN ('bodyweight');
-- DEBE retornar: 0
```

---

### 2.3 TC_HOME_003: Generacion de rutina HOME con mancuernas

**Categoria:** ROUTINE_GENERATION
**Prioridad:** CRITICAL
**Tipo:** SINGLE (post-KYC)
**Usuario:** `570000000092` (fixture)

#### Precondiciones
- Usuario existe con `training_environment = 'HOME'`
- `home_equipment = ['dumbbell', 'bodyweight', 'kettlebell']`
- Plan activo con template HOME

#### Pasos
1. Ejecutar workflow `GymRatForm Supabase v3` con usuario fixture
2. Verificar ejercicios generados

#### Datos del Usuario Fixture
```sql
-- Ver seccion 3.1 para INSERT completo
-- Usuario con mancuernas + kettlebell + bodyweight
```

#### Resultado Esperado
- Todos los ejercicios tienen `equipment IN ('dumbbell', 'bodyweight', 'kettlebell')`
- **0 ejercicios** con `equipment IN ('machine', 'cable', 'smith')`
- 4 semanas de workouts generadas
- Cobertura de todos los patrones requeridos

#### Query de Verificacion
```sql
-- Verificar solo ejercicios HOME-viable
SELECT
    w.week,
    w.day_name,
    e.spanish_name,
    e.equipment,
    e.pattern,
    e.main_muscle
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = 'e2e00092-0000-0000-0000-000000000092'
ORDER BY w.week, w.day_name, w.exercise_order;

-- Verificar NO hay ejercicios de maquina
SELECT COUNT(*) as machine_exercises
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = 'e2e00092-0000-0000-0000-000000000092'
AND e.equipment IN ('machine', 'cable', 'smith');
-- DEBE retornar: 0
```

---

### 2.4 TC_HOME_004: Generacion de rutina HOME solo peso corporal

**Categoria:** ROUTINE_GENERATION
**Prioridad:** HIGH
**Tipo:** SINGLE (post-KYC)
**Usuario:** `570000000093` (fixture)

#### Precondiciones
- Usuario existe con `training_environment = 'HOME'`
- `home_equipment = ['bodyweight']` (solo peso corporal)
- Plan activo con template HOME

#### Pasos
1. Ejecutar generacion de rutina para usuario
2. Verificar cobertura muscular adecuada

#### Resultado Esperado
- Todos los ejercicios tienen `equipment = 'bodyweight'`
- Cobertura muscular minima garantizada:
  - Pecho (push-ups variaciones)
  - Espalda (bodyweight rows si hay barra)
  - Piernas (squats, lunges)
  - Core (planks, crunches)
  - Gluteos (hip thrusts, bridges)
- Minimo 8 grupos musculares cubiertos

#### Query de Verificacion
```sql
-- Verificar solo bodyweight
SELECT COUNT(*) as non_bodyweight
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = 'e2e00093-0000-0000-0000-000000000093'
AND e.equipment != 'bodyweight';
-- DEBE retornar: 0

-- Verificar cobertura muscular
SELECT
    e.main_muscle,
    COUNT(DISTINCT w.id) as exercise_count
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = 'e2e00093-0000-0000-0000-000000000093'
AND w.week = 1
GROUP BY e.main_muscle
ORDER BY exercise_count DESC;
-- DEBE retornar: minimo 8 grupos musculares diferentes
```

---

### 2.5 TC_HOME_005: Compensacion de gaps en patrones

**Categoria:** PATTERN_COMPENSATION
**Prioridad:** HIGH
**Tipo:** SINGLE
**Usuario:** `570000000094` (fixture)

#### Precondiciones
- Usuario HOME sin barra de dominadas
- `home_equipment = ['dumbbell', 'bodyweight']` (sin pull_up_bar)

#### Contexto
El patron `pull_v` (dominadas, pulldowns) tiene solo 19 ejercicios disponibles para casa, la mayoria requieren barra de traccion. El sistema debe compensar este gap aumentando otros patrones.

#### Pasos
1. Generar rutina para usuario sin pull_up_bar
2. Verificar que pull_v reducido y pull_h/arm aumentado

#### Resultado Esperado
- Sets de `pull_v` reducidos (o sustituidos)
- Sets de `pull_h` y `arm` aumentados para compensar
- **Cobertura de espalda mantenida** via pull_h

#### Query de Verificacion
```sql
-- Analizar distribucion de patrones para usuario HOME sin pull-up bar
SELECT
    e.pattern,
    SUM(w.sets::integer) as total_sets,
    COUNT(*) as exercise_count
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = 'e2e00094-0000-0000-0000-000000000094'
AND w.week = 1
GROUP BY e.pattern
ORDER BY total_sets DESC;

-- Verificar que pull_h tiene mas sets que pull_v
-- ESPERADO: pull_h_sets >= pull_v_sets * 1.5

-- Verificar que no hay ejercicios pull_v que requieran pull-up bar
SELECT COUNT(*) as invalid_pullv
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = 'e2e00094-0000-0000-0000-000000000094'
AND e.pattern = 'pull_v'
AND e.equipment NOT IN ('dumbbell', 'bodyweight', 'kettlebell');
-- DEBE retornar: 0
```

---

### 2.6 TC_HOME_006: Usuario GYM existente (sin regresion)

**Categoria:** REGRESSION
**Prioridad:** CRITICAL
**Tipo:** SINGLE
**Usuario:** `570000000003` (fixture existente: Test_WithRoutine)

#### Precondiciones
- Usuario existente con `training_environment = 'GYM'` (o NULL/default)
- Plan activo con template GYM
- Rutina existente con ejercicios de maquina

#### Pasos
1. Verificar que usuario GYM existente mantiene su rutina
2. Simular VER_RUTINA_DE_HOY
3. Verificar respuesta incluye ejercicios de maquina

#### Resultado Esperado
- Usuario GYM NO se ve afectado por cambios HOME
- Ejercicios de maquina siguen apareciendo en rutina GYM
- Flujo VER_RUTINA funciona igual que antes

#### Query de Verificacion
```sql
-- Verificar que usuario GYM tiene ejercicios de maquina
SELECT
    COUNT(*) as machine_exercises
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = 'e2e00003-0000-0000-0000-000000000003'
AND e.equipment IN ('machine', 'cable');
-- DEBE retornar: > 0 (tiene ejercicios de maquina)

-- Verificar que training_environment es GYM o NULL
SELECT training_environment
FROM users_gym_profile
WHERE whatsapp_id = 570000000003;
-- DEBE retornar: 'GYM' o NULL
```

---

## 3. Datos de Prueba

### 3.1 Usuarios de prueba a crear

```sql
-- ============================================
-- SCRIPT DE DATOS DE PRUEBA - HOME TRAINING FEATURE
-- Ejecutar en Supabase SQL Editor
-- ============================================

-- ============================================
-- SECCION 1: TEARDOWN (Limpiar datos previos)
-- ============================================

-- Eliminar datos de usuarios HOME de prueba
DELETE FROM workouts WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000091', '570000000092', '570000000093', '570000000094')
);

DELETE FROM user_weekly_schedule WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000091', '570000000092', '570000000093', '570000000094')
);

DELETE FROM users_plans WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000091', '570000000092', '570000000093', '570000000094')
);

DELETE FROM users
WHERE full_phone_number::text IN ('570000000091', '570000000092', '570000000093', '570000000094');

DELETE FROM users_gym_profile
WHERE whatsapp_id IN (570000000091, 570000000092, 570000000093, 570000000094);

DELETE FROM n8n_chat_histories
WHERE session_id LIKE '%570000000091%'
   OR session_id LIKE '%570000000092%'
   OR session_id LIKE '%570000000093%'
   OR session_id LIKE '%570000000094%';

-- ============================================
-- SECCION 2: Usuario HOME con mancuernas (TC_HOME_003)
-- ============================================

-- Crear perfil gym HOME con mancuernas + kettlebell
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment, home_equipment
) VALUES (
    NOW(),
    570000000092,
    'Test Home Dumbbell',
    'test_home_dumbbell@gymbot.test',
    30,
    'M',
    180,
    80,
    'Ganar masa muscular',
    'Ninguna',
    '1 a 3 anos',
    '3-4 dias por semana',
    'Intermedio',
    'A',
    4,
    '45-60 minutos',
    'Manana',
    'Hipertrofia',
    'Pecho, Espalda',
    'Ninguno',
    'No',
    '0',
    'HOME',
    ARRAY['dumbbell', 'bodyweight', 'kettlebell']
);

-- Crear usuario
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00092-0000-0000-0000-000000000092',
    'Test Home Dumbbell',
    'test_home_dumbbell@gymbot.test',
    '570000000092',
    0000000092,
    57,
    'America/Bogota',
    NOW()
);

-- Crear plan HOME
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date)
SELECT
    'e2e00092-0000-0000-0001-000000000092',
    'e2e00092-0000-0000-0000-000000000092',
    template_id,
    week_schedule,
    goal,
    level,
    'active',
    NOW()
FROM routine_templates
WHERE days_per_week = 4
AND level = 'Intermedio'
AND environment = 'HOME'  -- Requiere template HOME creado
LIMIT 1;

-- ============================================
-- SECCION 3: Usuario HOME solo bodyweight (TC_HOME_004)
-- ============================================

INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment, home_equipment
) VALUES (
    NOW(),
    570000000093,
    'Test Home Bodyweight',
    'test_home_bodyweight@gymbot.test',
    25,
    'F',
    165,
    60,
    'Salud general / recomposicion corporal',
    'Ninguna',
    'Nunca he entrenado',
    '1-2 dias por semana',
    'Principiante',
    'A',
    3,
    '30-45 minutos',
    'Tarde',
    'Mixto',
    'Gluteo, Pierna',
    'Ninguno',
    'Caminata',
    '1-2',
    'HOME',
    ARRAY['bodyweight']
);

INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00093-0000-0000-0000-000000000093',
    'Test Home Bodyweight',
    'test_home_bodyweight@gymbot.test',
    '570000000093',
    0000000093,
    57,
    'America/Bogota',
    NOW()
);

INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date)
SELECT
    'e2e00093-0000-0000-0001-000000000093',
    'e2e00093-0000-0000-0000-000000000093',
    template_id,
    week_schedule,
    goal,
    level,
    'active',
    NOW()
FROM routine_templates
WHERE days_per_week = 3
AND level = 'Principiante'
AND environment = 'HOME'
LIMIT 1;

-- ============================================
-- SECCION 4: Usuario HOME sin pull-up bar (TC_HOME_005)
-- ============================================

INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment, home_equipment
) VALUES (
    NOW(),
    570000000094,
    'Test Home NoPullupBar',
    'test_home_nopullupbar@gymbot.test',
    35,
    'M',
    178,
    85,
    'Ganar masa muscular',
    'Ninguna',
    'Mas de 3 anos',
    '5+ dias por semana',
    'Avanzado',
    'A',
    5,
    '60-75 minutos',
    'Noche',
    'Fuerza',
    'Espalda, Hombros',
    'Ninguno',
    'No',
    '0',
    'HOME',
    ARRAY['dumbbell', 'bodyweight']  -- Sin pull_up_bar ni kettlebell
);

INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00094-0000-0000-0000-000000000094',
    'Test Home NoPullupBar',
    'test_home_nopullupbar@gymbot.test',
    '570000000094',
    0000000094,
    57,
    'America/Bogota',
    NOW()
);

INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date)
SELECT
    'e2e00094-0000-0000-0001-000000000094',
    'e2e00094-0000-0000-0000-000000000094',
    template_id,
    week_schedule,
    goal,
    level,
    'active',
    NOW()
FROM routine_templates
WHERE days_per_week = 5
AND level = 'Avanzado'
AND environment = 'HOME'
LIMIT 1;

-- ============================================
-- SECCION 5: VERIFICACION
-- ============================================

-- Verificar usuarios HOME creados
SELECT
    ugp.whatsapp_id,
    ugp.full_name,
    ugp.training_environment,
    ugp.home_equipment,
    u.user_id
FROM users_gym_profile ugp
LEFT JOIN users u ON u.full_phone_number::text = ugp.whatsapp_id::text
WHERE ugp.whatsapp_id IN (570000000092, 570000000093, 570000000094);

-- Verificar planes creados
SELECT
    u.full_name,
    up.goal,
    up.level,
    rt.environment,
    rt.days_per_week
FROM users_plans up
JOIN users u ON up.user_id = u.user_id
JOIN routine_templates rt ON up.template_id = rt.template_id
WHERE u.full_phone_number::text LIKE '5700000009%';
```

### 3.2 Phones reservados para HOME tests

| Phone | User ID | Proposito | Equipment |
|-------|---------|-----------|-----------|
| `570000000091` | (dinamico) | TC_HOME_001, TC_HOME_002 | Varia segun test |
| `570000000092` | `e2e00092-...` | TC_HOME_003 | dumbbell, bodyweight, kettlebell |
| `570000000093` | `e2e00093-...` | TC_HOME_004 | bodyweight only |
| `570000000094` | `e2e00094-...` | TC_HOME_005 | dumbbell, bodyweight (sin pull_up_bar) |

### 3.3 Phones existentes (NO modificar)

| Phone | Proposito Original | Nota |
|-------|-------------------|------|
| `570000000001` | TC003 - NoSchedule | Mantener GYM |
| `570000000002` | TC004 - RestDay | Mantener GYM |
| `570000000003` | TC006, TC007 - WithRoutine | **Usar para TC_HOME_006 (regression)** |
| `570000000004` | TC011, TC012 - PendingTask | Mantener GYM |
| `570000000009` | TC002 - Onboarding dinamico | NO usar para HOME tests |

---

## 4. Queries de Validacion

### 4.1 Verificar ejercicios generados son HOME-viable

```sql
-- Query principal: Detectar ejercicios NO compatibles con HOME
-- DEBE retornar 0 filas para usuarios HOME
SELECT
    w.id as workout_id,
    u.full_name,
    w.week,
    w.day_name,
    e.spanish_name,
    e.equipment,
    e.pattern
FROM workouts w
JOIN users u ON w.user_id = u.user_id
JOIN exercises e ON w.exercise_id = e.exercise_id
JOIN users_gym_profile ugp ON u.full_phone_number::text = ugp.whatsapp_id::text
WHERE ugp.training_environment = 'HOME'
AND e.equipment NOT IN ('dumbbell', 'bodyweight', 'kettlebell', 'barbell')
ORDER BY u.full_name, w.week, w.day_name;

-- CRITERIO: DEBE retornar 0 filas
```

### 4.2 Verificar cobertura muscular

```sql
-- Analizar distribucion de musculos por usuario HOME
WITH user_muscle_coverage AS (
    SELECT
        u.full_name,
        u.user_id,
        e.main_muscle,
        COUNT(DISTINCT w.id) as exercise_count,
        SUM(w.sets::integer) as total_sets
    FROM workouts w
    JOIN users u ON w.user_id = u.user_id
    JOIN exercises e ON w.exercise_id = e.exercise_id
    JOIN users_gym_profile ugp ON u.full_phone_number::text = ugp.whatsapp_id::text
    WHERE ugp.training_environment = 'HOME'
    AND w.week = 1  -- Primera semana como muestra
    GROUP BY u.full_name, u.user_id, e.main_muscle
)
SELECT
    full_name,
    COUNT(DISTINCT main_muscle) as muscle_groups_covered,
    STRING_AGG(main_muscle || '(' || exercise_count || ')', ', ') as muscle_distribution
FROM user_muscle_coverage
GROUP BY full_name, user_id;

-- CRITERIO: muscle_groups_covered >= 8 para cada usuario
```

### 4.3 Verificar equipamiento respetado

```sql
-- Query para validar que los ejercicios coinciden con el equipamiento del usuario
WITH user_equipment AS (
    SELECT
        whatsapp_id,
        full_name,
        UNNEST(home_equipment) as allowed_equipment
    FROM users_gym_profile
    WHERE training_environment = 'HOME'
),
exercise_equipment AS (
    SELECT
        u.full_phone_number::bigint as whatsapp_id,
        e.equipment as exercise_equipment,
        COUNT(*) as count
    FROM workouts w
    JOIN users u ON w.user_id = u.user_id
    JOIN exercises e ON w.exercise_id = e.exercise_id
    GROUP BY u.full_phone_number, e.equipment
)
SELECT
    ee.whatsapp_id,
    ee.exercise_equipment,
    ee.count,
    CASE
        WHEN EXISTS (
            SELECT 1 FROM user_equipment ue
            WHERE ue.whatsapp_id = ee.whatsapp_id
            AND ue.allowed_equipment = ee.exercise_equipment
        ) THEN 'ALLOWED'
        ELSE 'NOT ALLOWED'
    END as status
FROM exercise_equipment ee
WHERE ee.whatsapp_id IN (570000000092, 570000000093, 570000000094)
ORDER BY ee.whatsapp_id, status;

-- CRITERIO: Todos los ejercicios deben tener status = 'ALLOWED'
```

### 4.4 Verificar compensacion pull_v

```sql
-- Comparar distribucion de patrones entre usuario HOME (sin pull-up bar)
-- y usuario GYM equivalente
WITH pattern_distribution AS (
    SELECT
        u.full_name,
        ugp.training_environment,
        e.pattern,
        SUM(w.sets::integer) as total_sets,
        COUNT(*) as exercise_count
    FROM workouts w
    JOIN users u ON w.user_id = u.user_id
    JOIN exercises e ON w.exercise_id = e.exercise_id
    JOIN users_gym_profile ugp ON u.full_phone_number::text = ugp.whatsapp_id::text
    WHERE w.week = 1
    GROUP BY u.full_name, ugp.training_environment, e.pattern
)
SELECT
    full_name,
    training_environment,
    pattern,
    total_sets,
    exercise_count
FROM pattern_distribution
WHERE pattern IN ('pull_v', 'pull_h', 'arm')
ORDER BY training_environment, pattern;

-- CRITERIO para HOME sin pull-up bar:
-- pull_h_sets >= pull_v_sets
-- pull_v exercises deben ser compatibles con dumbbell/bodyweight
```

---

## 5. Checklist de Regresion

### 5.1 Funcionalidad GYM (No debe romperse)

- [ ] **TC006**: Usuario GYM puede ver rutina con ejercicios de maquina
- [ ] **TC007**: Usuario GYM puede hacer preguntas de fitness
- [ ] **TC003**: Usuario GYM sin schedule puede agendar semana
- [ ] **TC002**: Usuario nuevo puede completar KYC (default GYM si no especifica)
- [ ] **TC011/TC012**: Pending tasks funcionan para usuarios GYM

### 5.2 Workflows afectados

- [ ] `GymRatFlow_Supabase_V2_Workout_Tracker.json`: Sin cambios criticos
- [ ] `GymRatForm Supabase v3.json`: Filtrado por environment funciona
- [ ] `MorningReminder-WorkoutTracker.json`: Funciona para usuarios HOME
- [ ] `GymBotWorkoutCompletion.json`: Funciona para usuarios HOME

### 5.3 Base de datos

- [ ] Nueva columna `training_environment` en `users_gym_profile` tiene default 'GYM'
- [ ] Nueva columna `home_equipment` acepta arrays y NULL
- [ ] Templates HOME existen en `routine_templates`
- [ ] Constraint en `workouts` no rompe inserciones HOME

### 5.4 Integraciones

- [ ] WhatsApp: Mensajes formateados correctamente para rutinas HOME
- [ ] Workout Tracker web: Puede mostrar ejercicios HOME (sin links rotos)
- [ ] Morning Reminder: Incluye usuarios HOME en envios

---

## 6. Criterios de Aceptacion

### 6.1 Funcionales

| Criterio | Medicion | Umbral |
|----------|----------|--------|
| Test cases HOME pasan | TC_HOME_001 a TC_HOME_006 | 100% |
| Ejercicios maquina en rutinas HOME | Query 4.1 | 0 filas |
| Cobertura muscular | Query 4.2 | >= 8 grupos |
| Equipamiento respetado | Query 4.3 | 100% ALLOWED |
| Tests regresion GYM | TC001-TC012 originales | 100% |

### 6.2 No Funcionales

| Criterio | Medicion | Umbral |
|----------|----------|--------|
| Tiempo generacion rutina HOME | Medicion en workflow | < 60s |
| Mensajes WhatsApp legibles | Revision manual | Sin caracteres rotos |
| Re-prompt de equipamiento | TC_HOME_002 | Mensaje claro |

### 6.3 Definition of Done

- [ ] Todos los test cases TC_HOME_001 a TC_HOME_006 pasan
- [ ] Todos los test cases existentes (TC001-TC012) siguen pasando
- [ ] Queries de validacion retornan resultados esperados
- [ ] Checklist de regresion completado sin fallas
- [ ] Documentacion de tests actualizada en `e2e/GymRatFlow_test_plan.md`

---

## 7. Tareas para code-reviewer (QA)

### Pre-Testing Setup

- [ ] Ejecutar SQL de teardown para limpiar datos previos
- [ ] Verificar que templates HOME existen en BD (Fase 2)
- [ ] Verificar columnas `training_environment` y `home_equipment` existen
- [ ] Importar test data setup SQL en Supabase

### Test Execution

- [ ] Ejecutar TC_HOME_001 (KYC HOME con equipo)
- [ ] Ejecutar TC_HOME_002 (KYC HOME sin equipo - re-prompt)
- [ ] Ejecutar TC_HOME_003 (Rutina HOME mancuernas)
- [ ] Ejecutar TC_HOME_004 (Rutina HOME solo bodyweight)
- [ ] Ejecutar TC_HOME_005 (Compensacion pull_v)
- [ ] Ejecutar TC_HOME_006 (Regresion GYM)
- [ ] Ejecutar suite completa de tests existentes (TC001-TC012)

### Validation Queries

- [ ] Ejecutar Query 4.1 - Verificar ejercicios HOME-viable
- [ ] Ejecutar Query 4.2 - Verificar cobertura muscular
- [ ] Ejecutar Query 4.3 - Verificar equipamiento respetado
- [ ] Ejecutar Query 4.4 - Verificar compensacion pull_v

### Documentation

- [ ] Documentar resultados de cada test case
- [ ] Capturar screenshots de respuestas WhatsApp (si aplica)
- [ ] Reportar bugs encontrados en formato estandar
- [ ] Actualizar `e2e/GymRatFlow_test_plan.md` con nuevos tests HOME

### Sign-off

- [ ] Confirmar todos los criterios de aceptacion cumplidos
- [ ] Aprobar merge de feature a main (o reportar blockers)

---

## 8. Dependencias

### Dependencias de otras Fases

| Fase | Dependencia | Estado |
|------|-------------|--------|
| Fase 1 | Columnas `training_environment`, `home_equipment` en BD | Pendiente |
| Fase 2 | Templates HOME en `routine_templates` | Pendiente |
| Fase 3 | KYC Agent actualizado con preguntas HOME | Pendiente |
| Fase 3 | GymRatForm filtrado por equipment | Pendiente |

### Prerequisitos para ejecutar tests

1. **Migraciones aplicadas**: Fase 1 completada
2. **Templates creados**: Fase 2 completada
3. **Workflows actualizados**: Fase 3 completada
4. **Test data insertada**: SQL de seccion 3.1 ejecutado

---

## 9. Notas Adicionales

### Manejo de Errores Esperados

1. **Template HOME no existe**: Si los tests de fixture fallan porque no hay templates HOME, reportar como blocker de Fase 2

2. **Columna no existe**: Si las queries fallan por columnas faltantes, reportar como blocker de Fase 1

3. **Re-prompt no funciona**: Si TC_HOME_002 no muestra re-prompt, reportar como bug de KYC Agent (Fase 3)

### Ambiente de Testing

- **Supabase**: Usar ambiente de staging/desarrollo
- **n8n**: Instancia de desarrollo con workflows actualizados
- **WhatsApp**: Sandbox o numeros de prueba

### Rollback Plan

Si los tests revelan problemas criticos:
1. Revertir cambios en workflows (n8n tiene versionamiento)
2. Mantener columnas BD (no afectan usuarios existentes)
3. Templates HOME pueden desactivarse sin afectar GYM

---

*Documento creado: Febrero 2026*
*Ultima actualizacion: [Fecha de actualizacion]*
*Version: 1.0*
