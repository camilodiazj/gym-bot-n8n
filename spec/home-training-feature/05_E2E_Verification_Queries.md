# E2E Verification Queries for HOME Training Tests

**Document ID:** 05_E2E_Verification_Queries
**Version:** 1.0
**Status:** Ready for Implementation
**Created:** 2026-02-03
**Purpose:** Define SQL verification queries for HOME training E2E tests

---

## Table of Contents

1. [Overview](#1-overview)
2. [Equipment Mapping Reference](#2-equipment-mapping-reference)
3. [Verification Query Categories](#3-verification-query-categories)
4. [Complete Test Case Definition](#4-complete-test-case-definition)
5. [Query Templates by Scenario](#5-query-templates-by-scenario)

---

## 1. Overview

This document defines the SQL verification queries used by the E2E TestRunner to validate that HOME training routines are correctly generated. The queries follow the same pattern as `TC002_FULL_KYC` in `GymRatFlow_E2E_TestRunner.json`.

### Verification Strategy

| Category | What We Verify | Expected Result |
|----------|----------------|-----------------|
| Profile | `training_environment = 'HOME'` | Exactly 1 row |
| Template | Template assigned has `environment = 'HOME'` | Exactly 1 row |
| Equipment Compliance | All exercises use allowed equipment | 0 violations |
| No GYM-only Equipment | No machine/cable/smith exercises | 0 violations |
| Health Restrictions | Exercises respect health_status constraints | 0 violations |

---

## 2. Equipment Mapping Reference

### User Input (Spanish) to Database Values

| User Input | DB Value | HOME Compatible |
|------------|----------|-----------------|
| mancuernas | `dumbbell` | YES |
| kettlebell / pesa rusa | `kettlebell` | YES |
| barra | `barbell` | YES |
| peso corporal | `bodyweight` | ALWAYS |
| bandas / bandas elasticas | (treated as `bodyweight`) | YES |
| barra de dominadas | (enables `bodyweight` pull exercises) | YES |
| banco | (modifier, not equipment filter) | YES |
| maquinas | `machine` | NO |
| poleas / cables | `cable` | NO |
| smith | `smith` | NO |

### HOME-Compatible Equipment Values

```sql
-- These are the ONLY equipment values allowed for HOME users
('bodyweight', 'dumbbell', 'kettlebell', 'barbell')
```

### GYM-Only Equipment Values

```sql
-- These should NEVER appear in HOME user workouts
('machine', 'cable', 'smith', 'Máquina', 'Polea')
```

---

## 3. Verification Query Categories

### 3.1 Profile Verification

**Purpose:** Confirm user profile was created with correct HOME settings.

```sql
-- V1: Profile exists with HOME environment
SELECT COUNT(*) as cnt
FROM users_gym_profile
WHERE whatsapp_id = {PHONE_NUMBER}
  AND training_environment = 'HOME';
-- Expected: 1

-- V2: Home equipment was captured
SELECT COUNT(*) as cnt
FROM users_gym_profile
WHERE whatsapp_id = {PHONE_NUMBER}
  AND training_environment = 'HOME'
  AND home_equipment IS NOT NULL
  AND home_equipment != '';
-- Expected: 1
```

### 3.2 Template Verification

**Purpose:** Confirm the assigned template is a HOME template.

```sql
-- V3: User plan uses HOME template
SELECT COUNT(*) as cnt
FROM users_plans up
JOIN routine_templates rt ON up.template_id = rt.template_id
JOIN users u ON up.user_id = u.user_id
WHERE u.full_phone_number = '{PHONE_NUMBER}'
  AND rt.environment = 'HOME';
-- Expected: 1

-- V4: Template ID follows HOME naming convention
SELECT COUNT(*) as cnt
FROM users_plans up
JOIN users u ON up.user_id = u.user_id
WHERE u.full_phone_number = '{PHONE_NUMBER}'
  AND up.template_id LIKE '%_home';
-- Expected: 1
```

### 3.3 Equipment Compliance Verification

**Purpose:** Ensure ALL exercises assigned use only the user's available equipment.

```sql
-- V5: No exercises with GYM-only equipment (machine, cable, smith)
SELECT COUNT(*) as cnt
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
JOIN users u ON w.user_id = u.user_id
WHERE u.full_phone_number = '{PHONE_NUMBER}'
  AND e.equipment IN ('machine', 'cable', 'smith', 'Máquina', 'Polea');
-- Expected: 0 (CRITICAL - any value > 0 is a failure)

-- V6: All exercises are HOME-compatible (general check)
SELECT COUNT(*) as cnt
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
JOIN users u ON w.user_id = u.user_id
WHERE u.full_phone_number = '{PHONE_NUMBER}'
  AND e.equipment NOT IN ('bodyweight', 'dumbbell', 'kettlebell', 'barbell');
-- Expected: 0
```

### 3.4 Equipment-Specific Compliance (Dynamic)

For users with limited equipment (e.g., only bodyweight + dumbbells):

```sql
-- V7: Exercises match user's specific equipment
-- This query uses a CTE to build the allowed equipment list dynamically

WITH user_equipment AS (
  SELECT
    CASE
      WHEN home_equipment ILIKE '%mancuerna%' OR home_equipment ILIKE '%dumbbell%' THEN true
      ELSE false
    END as has_dumbbell,
    CASE
      WHEN home_equipment ILIKE '%kettlebell%' OR home_equipment ILIKE '%pesa rusa%' THEN true
      ELSE false
    END as has_kettlebell,
    CASE
      WHEN home_equipment ILIKE '%barra%' AND home_equipment NOT ILIKE '%dominadas%' THEN true
      ELSE false
    END as has_barbell
  FROM users_gym_profile
  WHERE whatsapp_id = {PHONE_NUMBER}
),
violations AS (
  SELECT w.id, e.exercise_id, e.equipment
  FROM workouts w
  JOIN exercises e ON w.exercise_id = e.exercise_id
  JOIN users u ON w.user_id = u.user_id
  CROSS JOIN user_equipment ue
  WHERE u.full_phone_number = '{PHONE_NUMBER}'
    AND (
      (e.equipment = 'dumbbell' AND NOT ue.has_dumbbell)
      OR (e.equipment = 'kettlebell' AND NOT ue.has_kettlebell)
      OR (e.equipment = 'barbell' AND NOT ue.has_barbell)
    )
)
SELECT COUNT(*) as cnt FROM violations;
-- Expected: 0
```

### 3.5 Health Status Restrictions

**Purpose:** Verify exercises respect health restrictions.

```sql
-- V8: Health Status C (upper body issues) - No overhead pressing
-- Exercises with push_v pattern should not include overhead movements
SELECT COUNT(*) as cnt
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
JOIN users u ON w.user_id = u.user_id
JOIN users_gym_profile ugp ON u.full_phone_number = ugp.whatsapp_id::text
WHERE u.full_phone_number = '{PHONE_NUMBER}'
  AND ugp.health_status = 'C'
  AND e.pattern = 'push_v'
  AND (
    e.spanish_name ILIKE '%press%militar%'
    OR e.spanish_name ILIKE '%press%hombro%'
    OR e.spanish_name ILIKE '%overhead%'
    OR e.spanish_name ILIKE '%sobre%cabeza%'
  );
-- Expected: 0

-- V9: Health Status D (spine issues) - No heavy axial loading
SELECT COUNT(*) as cnt
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
JOIN users u ON w.user_id = u.user_id
JOIN users_gym_profile ugp ON u.full_phone_number = ugp.whatsapp_id::text
WHERE u.full_phone_number = '{PHONE_NUMBER}'
  AND ugp.health_status = 'D'
  AND (
    e.spanish_name ILIKE '%peso muerto%'
    OR e.spanish_name ILIKE '%deadlift%'
    OR e.spanish_name ILIKE '%sentadilla%barra%'
    OR e.spanish_name ILIKE '%back squat%'
  );
-- Expected: 0
```

### 3.6 Workout Structure Verification

**Purpose:** Verify workout structure is complete.

```sql
-- V10: Workouts created for 4 weeks
SELECT COUNT(DISTINCT week) as cnt
FROM workouts w
JOIN users u ON w.user_id = u.user_id
WHERE u.full_phone_number = '{PHONE_NUMBER}';
-- Expected: 4

-- V11: Exercises have proper ordering
SELECT COUNT(*) as cnt
FROM workouts w
JOIN users u ON w.user_id = u.user_id
WHERE u.full_phone_number = '{PHONE_NUMBER}'
  AND w.exercise_order IS NOT NULL
  AND w.exercise_order > 0;
-- Expected: > 0 (at least some exercises have ordering)
```

---

## 4. Complete Test Case Definition

### TC_HOME_001: Full KYC with HOME Environment (Bodyweight + Dumbbells)

```javascript
{
  order: 11,
  id: "TC_HOME_001",
  name: "Onboarding HOME - Usuario con Mancuernas",
  priority: "CRITICAL",
  category: "ONBOARDING_HOME",
  testType: "MULTI_TURN_AI",
  simulatedUser: {
    nombre: "Maria Garcia Lopez",
    email: "maria.home.e2e@test.com",
    edad: 32,
    sexo: "F",
    estatura_cm: 165,
    peso_kg: 62,
    objetivo_principal: "Bajar grasa",
    objetivo_secundario: "Mejorar resistencia",
    tiempo_entrenando: "6 a 12 meses",
    frecuencia_actual: "2-3 días por semana",
    nivel: "Principiante",
    estado_salud: "A",
    dias_disponibles: 3,
    tiempo_por_sesion: "30-45 minutos",
    horario: "Mañana",
    tipo_entrenamiento: "En casa",
    // NEW HOME-SPECIFIC FIELDS
    ambiente_entrenamiento: "HOME",
    equipamiento_casa: "mancuernas, bandas elasticas",
    prioridades: "Glúteos y piernas",
    desafios: "Ninguno",
    cardio_actual: "Sí",
    frecuencia_cardio: "2"
  },
  config: {
    maxTurns: 25,
    completionIndicators: ["me pongo manos a la obra", "recibirás tu plan", "tu rutina 100%", "estoy emocionado", "en breve recibirás"],
    firstMessage: "Hola, quiero empezar a entrenar en casa"
  },
  cleanup: [
    "DELETE FROM n8n_chat_histories WHERE session_id LIKE '570000000010%';",
    "DELETE FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000010');",
    "DELETE FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000010');",
    "DELETE FROM pending_tasks WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000010');",
    "DELETE FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000010');",
    "DELETE FROM users WHERE full_phone_number = '570000000010';",
    "DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000010;"
  ],
  verification: {
    queries: [
      // Profile verification
      {
        sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000010 AND training_environment = 'HOME'",
        expected: 1,
        description: "Profile created with HOME environment"
      },
      // User and plan created
      {
        sql: "SELECT COUNT(*) as cnt FROM users WHERE full_phone_number = '570000000010'",
        expected: 1,
        description: "User record created"
      },
      {
        sql: "SELECT COUNT(*) as cnt FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000010')",
        expected: 1,
        description: "User plan created"
      },
      // Template is HOME
      {
        sql: "SELECT COUNT(*) as cnt FROM users_plans up JOIN routine_templates rt ON up.template_id = rt.template_id JOIN users u ON up.user_id = u.user_id WHERE u.full_phone_number = '570000000010' AND rt.environment = 'HOME'",
        expected: 1,
        description: "Assigned template is HOME environment"
      },
      // 4 weeks of workouts
      {
        sql: "SELECT COUNT(DISTINCT week) as cnt FROM workouts w JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000010'",
        expected: 4,
        description: "4 weeks of workouts created"
      },
      // NO GYM-ONLY EQUIPMENT (CRITICAL)
      {
        sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000010' AND e.equipment IN ('machine', 'cable', 'smith', 'Máquina', 'Polea')",
        expected: 0,
        description: "CRITICAL: No gym-only equipment in workouts"
      },
      // Only allowed equipment (bodyweight, dumbbell - user has mancuernas)
      {
        sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000010' AND e.equipment NOT IN ('bodyweight', 'dumbbell')",
        expected: 0,
        description: "Only bodyweight and dumbbell exercises (user equipment)"
      }
    ]
  },
  metrics: {
    rule: "passed === true",
    description: "Debe completar KYC HOME y crear rutina solo con equipamiento disponible"
  }
}
```

### TC_HOME_002: HOME with Full Equipment (Barbell + Dumbbell + Kettlebell)

```javascript
{
  order: 12,
  id: "TC_HOME_002",
  name: "Onboarding HOME - Home Gym Completo",
  priority: "HIGH",
  category: "ONBOARDING_HOME",
  testType: "MULTI_TURN_AI",
  simulatedUser: {
    nombre: "Carlos Rodriguez Martinez",
    email: "carlos.homegym.e2e@test.com",
    edad: 35,
    sexo: "M",
    estatura_cm: 180,
    peso_kg: 85,
    objetivo_principal: "Ganar masa muscular",
    objetivo_secundario: "Mejorar fuerza",
    tiempo_entrenando: "1 a 3 años",
    frecuencia_actual: "4-5 días por semana",
    nivel: "Intermedio",
    estado_salud: "A",
    dias_disponibles: 4,
    tiempo_por_sesion: "60-75 minutos",
    horario: "Tarde",
    tipo_entrenamiento: "En casa",
    ambiente_entrenamiento: "HOME",
    equipamiento_casa: "mancuernas, barra, kettlebell, barra de dominadas, banco",
    prioridades: "Pecho y espalda",
    desafios: "Ninguno",
    cardio_actual: "No",
    frecuencia_cardio: "0"
  },
  config: {
    maxTurns: 25,
    completionIndicators: ["me pongo manos a la obra", "recibirás tu plan", "tu rutina 100%", "estoy emocionado", "en breve recibirás"],
    firstMessage: "Hola, quiero entrenar en mi casa, tengo un home gym"
  },
  cleanup: [
    "DELETE FROM n8n_chat_histories WHERE session_id LIKE '570000000011%';",
    "DELETE FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000011');",
    "DELETE FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000011');",
    "DELETE FROM pending_tasks WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000011');",
    "DELETE FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000011');",
    "DELETE FROM users WHERE full_phone_number = '570000000011';",
    "DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000011;"
  ],
  verification: {
    queries: [
      // Profile with HOME
      {
        sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000011 AND training_environment = 'HOME'",
        expected: 1,
        description: "Profile created with HOME environment"
      },
      // User created
      {
        sql: "SELECT COUNT(*) as cnt FROM users WHERE full_phone_number = '570000000011'",
        expected: 1,
        description: "User record created"
      },
      // Plan created
      {
        sql: "SELECT COUNT(*) as cnt FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000011')",
        expected: 1,
        description: "User plan created"
      },
      // HOME template assigned
      {
        sql: "SELECT COUNT(*) as cnt FROM users_plans up JOIN routine_templates rt ON up.template_id = rt.template_id JOIN users u ON up.user_id = u.user_id WHERE u.full_phone_number = '570000000011' AND rt.environment = 'HOME'",
        expected: 1,
        description: "Assigned template is HOME environment"
      },
      // 4 weeks
      {
        sql: "SELECT COUNT(DISTINCT week) as cnt FROM workouts w JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000011'",
        expected: 4,
        description: "4 weeks of workouts created"
      },
      // NO GYM-ONLY EQUIPMENT
      {
        sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000011' AND e.equipment IN ('machine', 'cable', 'smith', 'Máquina', 'Polea')",
        expected: 0,
        description: "CRITICAL: No gym-only equipment in workouts"
      },
      // All equipment allowed (full home gym)
      {
        sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000011' AND e.equipment NOT IN ('bodyweight', 'dumbbell', 'kettlebell', 'barbell')",
        expected: 0,
        description: "Only HOME-compatible equipment (bodyweight, dumbbell, kettlebell, barbell)"
      }
    ]
  },
  metrics: {
    rule: "passed === true",
    description: "Debe completar KYC HOME con home gym completo"
  }
}
```

### TC_HOME_003: HOME with Bodyweight Only

```javascript
{
  order: 13,
  id: "TC_HOME_003",
  name: "Onboarding HOME - Solo Peso Corporal",
  priority: "HIGH",
  category: "ONBOARDING_HOME",
  testType: "MULTI_TURN_AI",
  simulatedUser: {
    nombre: "Ana Martinez Silva",
    email: "ana.bodyweight.e2e@test.com",
    edad: 25,
    sexo: "F",
    estatura_cm: 160,
    peso_kg: 55,
    objetivo_principal: "Mejorar resistencia",
    objetivo_secundario: "Salud general / recomposición corporal",
    tiempo_entrenando: "Menos de 6 meses",
    frecuencia_actual: "1-2 días por semana",
    nivel: "Principiante",
    estado_salud: "A",
    dias_disponibles: 2,
    tiempo_por_sesion: "30-45 minutos",
    horario: "Noche",
    tipo_entrenamiento: "En casa",
    ambiente_entrenamiento: "HOME",
    equipamiento_casa: "peso corporal",
    prioridades: "Core y piernas",
    desafios: "Ninguno",
    cardio_actual: "Sí",
    frecuencia_cardio: "3"
  },
  config: {
    maxTurns: 25,
    completionIndicators: ["me pongo manos a la obra", "recibirás tu plan", "tu rutina 100%", "estoy emocionado", "en breve recibirás"],
    firstMessage: "Hola, quiero entrenar en casa pero no tengo equipo"
  },
  cleanup: [
    "DELETE FROM n8n_chat_histories WHERE session_id LIKE '570000000012%';",
    "DELETE FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000012');",
    "DELETE FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000012');",
    "DELETE FROM pending_tasks WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000012');",
    "DELETE FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000012');",
    "DELETE FROM users WHERE full_phone_number = '570000000012';",
    "DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000012;"
  ],
  verification: {
    queries: [
      // Profile with HOME
      {
        sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000012 AND training_environment = 'HOME'",
        expected: 1,
        description: "Profile created with HOME environment"
      },
      // User created
      {
        sql: "SELECT COUNT(*) as cnt FROM users WHERE full_phone_number = '570000000012'",
        expected: 1,
        description: "User record created"
      },
      // Plan created
      {
        sql: "SELECT COUNT(*) as cnt FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000012')",
        expected: 1,
        description: "User plan created"
      },
      // HOME template
      {
        sql: "SELECT COUNT(*) as cnt FROM users_plans up JOIN routine_templates rt ON up.template_id = rt.template_id JOIN users u ON up.user_id = u.user_id WHERE u.full_phone_number = '570000000012' AND rt.environment = 'HOME'",
        expected: 1,
        description: "Assigned template is HOME environment"
      },
      // 4 weeks
      {
        sql: "SELECT COUNT(DISTINCT week) as cnt FROM workouts w JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000012'",
        expected: 4,
        description: "4 weeks of workouts created"
      },
      // ONLY BODYWEIGHT (most restrictive)
      {
        sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000012' AND e.equipment != 'bodyweight'",
        expected: 0,
        description: "CRITICAL: Only bodyweight exercises (no equipment)"
      }
    ]
  },
  metrics: {
    rule: "passed === true",
    description: "Debe completar KYC HOME con solo peso corporal"
  }
}
```

### TC_HOME_004: HOME with Health Restriction (Status C - Upper Body)

```javascript
{
  order: 14,
  id: "TC_HOME_004",
  name: "Onboarding HOME - Con Restriccion Tren Superior",
  priority: "HIGH",
  category: "ONBOARDING_HOME_HEALTH",
  testType: "MULTI_TURN_AI",
  simulatedUser: {
    nombre: "Pedro Sanchez Ruiz",
    email: "pedro.healthc.e2e@test.com",
    edad: 45,
    sexo: "M",
    estatura_cm: 175,
    peso_kg: 80,
    objetivo_principal: "Salud general / recomposición corporal",
    objetivo_secundario: "Bajar grasa",
    tiempo_entrenando: "1 a 3 años",
    frecuencia_actual: "2-3 días por semana",
    nivel: "Intermedio",
    estado_salud: "C", // Upper body issues - avoid overhead
    dias_disponibles: 3,
    tiempo_por_sesion: "45-60 minutos",
    horario: "Mañana",
    tipo_entrenamiento: "En casa",
    ambiente_entrenamiento: "HOME",
    equipamiento_casa: "mancuernas, bandas elasticas",
    prioridades: "Piernas y core",
    desafios: "Ejercicios de hombro",
    cardio_actual: "Sí",
    frecuencia_cardio: "2"
  },
  config: {
    maxTurns: 25,
    completionIndicators: ["me pongo manos a la obra", "recibirás tu plan", "tu rutina 100%", "estoy emocionado", "en breve recibirás"],
    firstMessage: "Hola, quiero entrenar en casa pero tengo problemas de hombro"
  },
  cleanup: [
    "DELETE FROM n8n_chat_histories WHERE session_id LIKE '570000000013%';",
    "DELETE FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000013');",
    "DELETE FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000013');",
    "DELETE FROM pending_tasks WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000013');",
    "DELETE FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000013');",
    "DELETE FROM users WHERE full_phone_number = '570000000013';",
    "DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000013;"
  ],
  verification: {
    queries: [
      // Profile with HOME and health status C
      {
        sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000013 AND training_environment = 'HOME' AND health_status = 'C'",
        expected: 1,
        description: "Profile created with HOME environment and health_status C"
      },
      // User created
      {
        sql: "SELECT COUNT(*) as cnt FROM users WHERE full_phone_number = '570000000013'",
        expected: 1,
        description: "User record created"
      },
      // Plan created
      {
        sql: "SELECT COUNT(*) as cnt FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000013')",
        expected: 1,
        description: "User plan created"
      },
      // 4 weeks
      {
        sql: "SELECT COUNT(DISTINCT week) as cnt FROM workouts w JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000013'",
        expected: 4,
        description: "4 weeks of workouts created"
      },
      // NO GYM-ONLY EQUIPMENT
      {
        sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000013' AND e.equipment IN ('machine', 'cable', 'smith', 'Máquina', 'Polea')",
        expected: 0,
        description: "CRITICAL: No gym-only equipment in workouts"
      },
      // Only allowed equipment
      {
        sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000013' AND e.equipment NOT IN ('bodyweight', 'dumbbell')",
        expected: 0,
        description: "Only bodyweight and dumbbell exercises"
      },
      // Health C: No overhead pressing (push_v with overhead keywords)
      {
        sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '570000000013' AND e.pattern = 'push_v' AND (e.spanish_name ILIKE '%press%militar%' OR e.spanish_name ILIKE '%press%hombro%' OR e.spanish_name ILIKE '%overhead%' OR e.spanish_name ILIKE '%sobre%cabeza%' OR e.spanish_name ILIKE '%arnold%')",
        expected: 0,
        description: "HEALTH: No overhead pressing exercises (health_status C)"
      }
    ]
  },
  metrics: {
    rule: "passed === true",
    description: "Debe completar KYC HOME respetando restriccion de tren superior"
  }
}
```

---

## 5. Query Templates by Scenario

### Generic Verification Query Template

Use these parameterized queries in your test cases:

```javascript
// Template: Replace {PHONE} with actual phone number

const verificationQueries = {
  // Profile Checks
  profileExists: (phone) => ({
    sql: `SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = ${phone}`,
    expected: 1
  }),

  profileIsHome: (phone) => ({
    sql: `SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = ${phone} AND training_environment = 'HOME'`,
    expected: 1
  }),

  // User & Plan Checks
  userExists: (phone) => ({
    sql: `SELECT COUNT(*) as cnt FROM users WHERE full_phone_number = '${phone}'`,
    expected: 1
  }),

  planExists: (phone) => ({
    sql: `SELECT COUNT(*) as cnt FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '${phone}')`,
    expected: 1
  }),

  templateIsHome: (phone) => ({
    sql: `SELECT COUNT(*) as cnt FROM users_plans up JOIN routine_templates rt ON up.template_id = rt.template_id JOIN users u ON up.user_id = u.user_id WHERE u.full_phone_number = '${phone}' AND rt.environment = 'HOME'`,
    expected: 1
  }),

  // Workout Checks
  fourWeeksCreated: (phone) => ({
    sql: `SELECT COUNT(DISTINCT week) as cnt FROM workouts w JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '${phone}'`,
    expected: 4
  }),

  // Equipment Compliance (CRITICAL)
  noGymOnlyEquipment: (phone) => ({
    sql: `SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '${phone}' AND e.equipment IN ('machine', 'cable', 'smith', 'Máquina', 'Polea')`,
    expected: 0
  }),

  onlyBodyweight: (phone) => ({
    sql: `SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '${phone}' AND e.equipment != 'bodyweight'`,
    expected: 0
  }),

  onlyBodyweightAndDumbbell: (phone) => ({
    sql: `SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '${phone}' AND e.equipment NOT IN ('bodyweight', 'dumbbell')`,
    expected: 0
  }),

  allHomeEquipment: (phone) => ({
    sql: `SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '${phone}' AND e.equipment NOT IN ('bodyweight', 'dumbbell', 'kettlebell', 'barbell')`,
    expected: 0
  }),

  // Health Restrictions
  noOverheadForHealthC: (phone) => ({
    sql: `SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id JOIN users u ON w.user_id = u.user_id WHERE u.full_phone_number = '${phone}' AND e.pattern = 'push_v' AND (e.spanish_name ILIKE '%press%militar%' OR e.spanish_name ILIKE '%press%hombro%' OR e.spanish_name ILIKE '%overhead%' OR e.spanish_name ILIKE '%sobre%cabeza%' OR e.spanish_name ILIKE '%arnold%')`,
    expected: 0
  })
};
```

---

## Appendix: Reserved Phone Numbers for HOME Tests

| Phone Number | Test Case | User Type |
|--------------|-----------|-----------|
| `570000000010` | TC_HOME_001 | HOME - Mancuernas + Bandas |
| `570000000011` | TC_HOME_002 | HOME - Full Home Gym |
| `570000000012` | TC_HOME_003 | HOME - Bodyweight Only |
| `570000000013` | TC_HOME_004 | HOME - Health Status C |
| `570000000014` | (Reserved) | Future HOME tests |
| `570000000015` | (Reserved) | Future HOME tests |

> **Important**: Phone numbers `57000000001X` (10-19) are reserved for HOME training E2E tests.

---

**Document Control**

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-02-03 | Claude Code (pixel-dev) | Initial creation |

---

*This document is part of the GymBot Home Training Feature.*
