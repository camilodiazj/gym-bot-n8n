-- ============================================
-- SCRIPT DE DATOS DE PRUEBA E2E
-- GymRatFlow_Supabase v4.0
-- ============================================
--
-- USUARIOS DUMMY (Fixtures):
-- 570000000001 - Test_NoSchedule (TC003)
-- 570000000002 - Test_RestDay (TC004)
-- 570000000003 - Test_WithRoutine (TC006, TC007)
-- 570000000004 - Test_WithPendingTask (TC011, TC012)
-- 570000000009 - DINAMICO (TC002, TC002_FULL_KYC) - creado/eliminado por tests
-- 570000000051 - Test_MesoDetect (TC_MESO_001) - deteccion automatica
-- 570000000052 - Test_MesoMantener (TC_MESO_002) - flujo MANTENER
-- 570000000053 - Test_MesoManual (TC_MESO_003) - intencion manual
--
-- TABLAS AFECTADAS:
-- users, users_plans, user_weekly_schedule, workouts,
-- users_gym_profile, n8n_chat_histories, pending_tasks
--
-- ============================================

-- ============================================
-- SECCION 1: TEARDOWN (Limpiar datos de prueba)
-- Ejecutar PRIMERO para estado limpio
-- IMPORTANTE: Incluye TODOS los phones de prueba:
--   GYM: 570000000001-004, 570000000009
--   HOME: 570000000211, 570000000212, 570000000213
--   MESOCYCLE: 570000000051, 570000000052, 570000000053
-- ============================================

-- Eliminar pending_tasks de usuarios dummy (GYM + HOME + MESOCYCLE)
DELETE FROM pending_tasks
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN (
        '570000000001', '570000000002', '570000000003', '570000000004', '570000000009',
        '570000000211', '570000000212', '570000000213',
        '570000000051', '570000000052', '570000000053'
    )
);

-- Eliminar workouts de usuarios dummy (GYM + HOME + MESOCYCLE)
DELETE FROM workouts
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN (
        '570000000001', '570000000002', '570000000003', '570000000004', '570000000009',
        '570000000211', '570000000212', '570000000213',
        '570000000051', '570000000052', '570000000053'
    )
);

-- Eliminar schedules de usuarios dummy (GYM + HOME + MESOCYCLE)
DELETE FROM user_weekly_schedule
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN (
        '570000000001', '570000000002', '570000000003', '570000000004', '570000000009',
        '570000000211', '570000000212', '570000000213',
        '570000000051', '570000000052', '570000000053'
    )
);

-- Eliminar planes de usuarios dummy (GYM + HOME + MESOCYCLE)
DELETE FROM users_plans
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN (
        '570000000001', '570000000002', '570000000003', '570000000004', '570000000009',
        '570000000211', '570000000212', '570000000213',
        '570000000051', '570000000052', '570000000053'
    )
);

-- Eliminar usuarios dummy (GYM + HOME + MESOCYCLE)
DELETE FROM users
WHERE full_phone_number::text IN (
    '570000000001', '570000000002', '570000000003', '570000000004', '570000000009',
    '570000000211', '570000000212', '570000000213',
    '570000000051', '570000000052', '570000000053'
);

-- Eliminar perfiles gym de usuarios dummy (whatsapp_id es BIGINT, no string)
DELETE FROM users_gym_profile
WHERE whatsapp_id IN (
    570000000001, 570000000002, 570000000003, 570000000004, 570000000009,
    570000000211, 570000000212, 570000000213,
    570000000051, 570000000052, 570000000053
);

-- Limpiar memoria de chat de usuarios dummy (GYM + HOME + MESOCYCLE)
DELETE FROM n8n_chat_histories
WHERE session_id LIKE '%570000000001%'
   OR session_id LIKE '%570000000002%'
   OR session_id LIKE '%570000000003%'
   OR session_id LIKE '%570000000004%'
   OR session_id LIKE '%570000000009%'
   OR session_id LIKE '%570000000211%'
   OR session_id LIKE '%570000000212%'
   OR session_id LIKE '%570000000213%'
   OR session_id LIKE '%570000000051%'
   OR session_id LIKE '%570000000052%'
   OR session_id LIKE '%570000000053%'
   OR session_id LIKE '%e2e00001-0000-0000-0000-000000000001%'
   OR session_id LIKE '%e2e00002-0000-0000-0000-000000000002%'
   OR session_id LIKE '%e2e00003-0000-0000-0000-000000000003%'
   OR session_id LIKE '%e2e00004-0000-0000-0000-000000000004%'
   OR session_id LIKE '%e2e00051-0000-0000-0000-000000000051%'
   OR session_id LIKE '%e2e00052-0000-0000-0000-000000000052%'
   OR session_id LIKE '%e2e00053-0000-0000-0000-000000000053%'
   ;

-- ============================================
-- SECCION 2: SETUP USUARIOS BASE
-- ============================================

-- Usuario 1: Test_NoSchedule (para TC003 - sin workouts planeados)
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00001-0000-0000-0000-000000000001',
    'Test NoSchedule',
    'test_noschedule@gymbot.test',
    '570000000001',
    0000000001,
    57,
    'America/Bogota',
    NOW()
);

-- Usuario 2: Test_RestDay (para TC004 - dia de descanso)
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00002-0000-0000-0000-000000000002',
    'Test RestDay',
    'test_restday@gymbot.test',
    '570000000002',
    0000000002,
    57,
    'America/Bogota',
    NOW()
);

-- Usuario 3: Test_WithRoutine (para TC006, TC007 - con rutina hoy)
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00003-0000-0000-0000-000000000003',
    'Test WithRoutine',
    'test_withroutine@gymbot.test',
    '570000000003',
    0000000003,
    57,
    'America/Bogota',
    NOW()
);

-- Usuario 4: Test_WithPendingTask (para TC011-TC012 - con pending_task) [NUEVO v3.0]
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00004-0000-0000-0000-000000000004',
    'Test WithPendingTask',
    'test_withpendingtask@gymbot.test',
    '570000000004',
    0000000004,
    57,
    'America/Bogota',
    NOW()
);

-- Usuario 51: Test_MesoDetect (para TC_MESO_001 - deteccion automatica W4) [NUEVO v5.0]
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00051-0000-0000-0000-000000000051',
    'Test MesoDetect',
    'test_mesodetect@gymbot.test',
    '570000000051',
    0000000051,
    57,
    'America/Bogota',
    NOW()
);

-- Usuario 52: Test_MesoMantener (para TC_MESO_002 - flujo MANTENER_RUTINA) [NUEVO v5.0]
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00052-0000-0000-0000-000000000052',
    'Test MesoMantener',
    'test_mesomantener@gymbot.test',
    '570000000052',
    0000000052,
    57,
    'America/Bogota',
    NOW()
);

-- Usuario 53: Test_MesoManual (para TC_MESO_003 - intencion manual RENOVAR_MESOCICLO) [NUEVO v5.0]
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00053-0000-0000-0000-000000000053',
    'Test MesoManual',
    'test_mesomanual@gymbot.test',
    '570000000053',
    0000000053,
    57,
    'America/Bogota',
    NOW()
);

-- ============================================
-- SECCION 2.5: SETUP GYM PROFILES (Mesocycle Users) [NUEVO v5.0]
-- ============================================

-- Perfil gym para Usuario 51 (MesoDetect)
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment
) VALUES (
    NOW(), 570000000051, 'Test MesoDetect', 'test_mesodetect@gymbot.test', 28, 'M',
    178, 80, 'Ganar masa muscular', 'Ninguna', '1 a 3 años',
    '3-4 días por semana', 'Intermedio', 'A', 3,
    '60-75 minutos', 'Mañana', 'Mixto', 'Pecho, Espalda',
    'Pantorrillas', 'No', '0',
    'GYM'
);

-- Perfil gym para Usuario 52 (MesoMantener)
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment
) VALUES (
    NOW(), 570000000052, 'Test MesoMantener', 'test_mesomantener@gymbot.test', 30, 'M',
    175, 78, 'Ganar masa muscular', 'Ninguna', '1 a 3 años',
    '3-4 días por semana', 'Intermedio', 'A', 3,
    '60-75 minutos', 'Tarde', 'Mixto', 'Pecho, Espalda',
    'Pantorrillas', 'No', '0',
    'GYM'
);

-- Perfil gym para Usuario 53 (MesoManual)
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment
) VALUES (
    NOW(), 570000000053, 'Test MesoManual', 'test_mesomanual@gymbot.test', 26, 'F',
    165, 62, 'Ganar masa muscular', 'Ninguna', '6 a 12 meses',
    '3-4 días por semana', 'Intermedio', 'A', 3,
    '60-75 minutos', 'Noche', 'Mixto', 'Glúteo, Pierna',
    'Pantorrillas', 'Caminata', '1-2',
    'GYM'
);

-- ============================================
-- SECCION 3: SETUP PLANES DE USUARIOS
-- Nota: plan_id debe ser UUID valido (sin letras no-hex)
-- ============================================

-- Plan para Usuario 1 (NoSchedule)
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date)
SELECT
    'e2e00001-0000-0000-0001-000000000001',
    'e2e00001-0000-0000-0000-000000000001',
    template_id,
    week_schedule,
    goal,
    level,
    'active',
    NOW()
FROM routine_templates
WHERE days_per_week = 3 AND level = 'Principiante'
LIMIT 1;

-- Plan para Usuario 2 (RestDay)
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date)
SELECT
    'e2e00002-0000-0000-0001-000000000002',
    'e2e00002-0000-0000-0000-000000000002',
    template_id,
    week_schedule,
    goal,
    level,
    'active',
    NOW()
FROM routine_templates
WHERE days_per_week = 3 AND level = 'Principiante'
LIMIT 1;

-- Plan para Usuario 3 (WithRoutine)
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date)
SELECT
    'e2e00003-0000-0000-0001-000000000003',
    'e2e00003-0000-0000-0000-000000000003',
    template_id,
    week_schedule,
    goal,
    level,
    'active',
    NOW()
FROM routine_templates
WHERE days_per_week = 3 AND level = 'Principiante'
LIMIT 1;

-- Plan para Usuario 4 (WithPendingTask) [NUEVO v3.0]
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date)
SELECT
    'e2e00004-0000-0000-0001-000000000004',
    'e2e00004-0000-0000-0000-000000000004',
    template_id,
    week_schedule,
    goal,
    level,
    'active',
    NOW()
FROM routine_templates
WHERE days_per_week = 3 AND level = 'Principiante'
LIMIT 1;

-- Plan para Usuario 51 (MesoDetect) [NUEVO v5.0]
-- fb_3 = Full Body 3 dias/semana, Intermedio, Ganar masa muscular
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, mesocycle_number, last_renewal_date, start_date)
SELECT
    'e2e00051-0000-0000-0001-000000000051',
    'e2e00051-0000-0000-0000-000000000051',
    template_id,
    'fb_3',
    'Ganar masa muscular',
    'Intermedio',
    'active',
    1,
    NULL,
    NOW() - INTERVAL '28 days'
FROM routine_templates
WHERE week_schedule = 'fb_3' AND level = 'Intermedio' AND goal = 'Ganar masa muscular'
LIMIT 1;

-- Plan para Usuario 52 (MesoMantener) [NUEVO v5.0]
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, mesocycle_number, last_renewal_date, start_date)
SELECT
    'e2e00052-0000-0000-0001-000000000052',
    'e2e00052-0000-0000-0000-000000000052',
    template_id,
    'fb_3',
    'Ganar masa muscular',
    'Intermedio',
    'active',
    1,
    NULL,
    NOW() - INTERVAL '28 days'
FROM routine_templates
WHERE week_schedule = 'fb_3' AND level = 'Intermedio' AND goal = 'Ganar masa muscular'
LIMIT 1;

-- Plan para Usuario 53 (MesoManual) [NUEVO v5.0]
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, mesocycle_number, last_renewal_date, start_date)
SELECT
    'e2e00053-0000-0000-0001-000000000053',
    'e2e00053-0000-0000-0000-000000000053',
    template_id,
    'fb_3',
    'Ganar masa muscular',
    'Intermedio',
    'active',
    1,
    NULL,
    NOW() - INTERVAL '7 days'
FROM routine_templates
WHERE week_schedule = 'fb_3' AND level = 'Intermedio' AND goal = 'Ganar masa muscular'
LIMIT 1;

-- ============================================
-- SECCION 4: SETUP SCHEDULES
-- Nota: week_day es enum week_days con valores en espanol
-- ============================================

-- Usuario 2 (RestDay): Schedule para MANANA (no hoy)
-- Usando timezone de Bogota para evitar desfase con UTC
-- planned_day_utc = midnight Bogota expresado en UTC (05:00:00+00)
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00002-0000-0000-0000-000000000002',
    1,
    (CASE EXTRACT(DOW FROM (NOW() AT TIME ZONE 'America/Bogota')::date + 1)
        WHEN 0 THEN 'Domingo'
        WHEN 1 THEN 'Lunes'
        WHEN 2 THEN 'Martes'
        WHEN 3 THEN 'Miercoles'
        WHEN 4 THEN 'Jueves'
        WHEN 5 THEN 'Viernes'
        WHEN 6 THEN 'Sabado'
    END)::week_days,
    'Dia 1 - Pecho y Triceps',
    ((NOW() AT TIME ZONE 'America/Bogota')::date + INTERVAL '1 day')::date::text,
    (DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota' + INTERVAL '1 day'),
    false
);

-- Usuario 3 (WithRoutine): Schedule para HOY
-- Usando timezone de Bogota para evitar desfase con UTC
-- planned_day_utc = midnight Bogota expresado en UTC (05:00:00+00)
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00003-0000-0000-0000-000000000003',
    1,
    (CASE EXTRACT(DOW FROM (NOW() AT TIME ZONE 'America/Bogota')::date)
        WHEN 0 THEN 'Domingo'
        WHEN 1 THEN 'Lunes'
        WHEN 2 THEN 'Martes'
        WHEN 3 THEN 'Miercoles'
        WHEN 4 THEN 'Jueves'
        WHEN 5 THEN 'Viernes'
        WHEN 6 THEN 'Sabado'
    END)::week_days,
    'Dia 1 - Pecho y Triceps',
    (NOW() AT TIME ZONE 'America/Bogota')::date::text,
    (DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota'),
    false
);

-- Usuario 4 (WithPendingTask): Schedule para HOY [NUEVO v3.0]
-- Usando timezone de Bogota para evitar desfase con UTC
-- planned_day_utc = midnight Bogota expresado en UTC (05:00:00+00)
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00004-0000-0000-0000-000000000004',
    1,
    (CASE EXTRACT(DOW FROM (NOW() AT TIME ZONE 'America/Bogota')::date)
        WHEN 0 THEN 'Domingo'
        WHEN 1 THEN 'Lunes'
        WHEN 2 THEN 'Martes'
        WHEN 3 THEN 'Miercoles'
        WHEN 4 THEN 'Jueves'
        WHEN 5 THEN 'Viernes'
        WHEN 6 THEN 'Sabado'
    END)::week_days,
    'Dia 1 - Pecho y Triceps',
    (NOW() AT TIME ZONE 'America/Bogota')::date::text,
    (DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota'),
    false
);

-- ============================================
-- SECCION 4.1: SETUP SCHEDULES MESOCYCLE (Usuarios 051, 052, 053) [NUEVO v5.0]
-- Usuarios 051 y 052: 4 semanas x 3 sesiones = 12 entradas, TODAS completadas en el pasado
-- Usuario 053: Semana 1 con 3 sesiones en el FUTURO, NO completadas
-- ============================================

-- Usuario 51 (MesoDetect): 4 semanas completas en el pasado (Lunes, Miercoles, Viernes)
-- Semana 1
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 1, 'Lunes', 'Full Body A',
    (CURRENT_DATE - INTERVAL '27 days')::text,
    (CURRENT_DATE - INTERVAL '27 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 1, 'Miercoles', 'Full Body B',
    (CURRENT_DATE - INTERVAL '25 days')::text,
    (CURRENT_DATE - INTERVAL '25 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 1, 'Viernes', 'Full Body C',
    (CURRENT_DATE - INTERVAL '23 days')::text,
    (CURRENT_DATE - INTERVAL '23 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
-- Semana 2
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 2, 'Lunes', 'Full Body A',
    (CURRENT_DATE - INTERVAL '20 days')::text,
    (CURRENT_DATE - INTERVAL '20 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 2, 'Miercoles', 'Full Body B',
    (CURRENT_DATE - INTERVAL '18 days')::text,
    (CURRENT_DATE - INTERVAL '18 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 2, 'Viernes', 'Full Body C',
    (CURRENT_DATE - INTERVAL '16 days')::text,
    (CURRENT_DATE - INTERVAL '16 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
-- Semana 3
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 3, 'Lunes', 'Full Body A',
    (CURRENT_DATE - INTERVAL '13 days')::text,
    (CURRENT_DATE - INTERVAL '13 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 3, 'Miercoles', 'Full Body B',
    (CURRENT_DATE - INTERVAL '11 days')::text,
    (CURRENT_DATE - INTERVAL '11 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 3, 'Viernes', 'Full Body C',
    (CURRENT_DATE - INTERVAL '9 days')::text,
    (CURRENT_DATE - INTERVAL '9 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
-- Semana 4
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 4, 'Lunes', 'Full Body A',
    (CURRENT_DATE - INTERVAL '6 days')::text,
    (CURRENT_DATE - INTERVAL '6 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 4, 'Miercoles', 'Full Body B',
    (CURRENT_DATE - INTERVAL '4 days')::text,
    (CURRENT_DATE - INTERVAL '4 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00051-0000-0000-0000-000000000051', 4, 'Viernes', 'Full Body C',
    (CURRENT_DATE - INTERVAL '2 days')::text,
    (CURRENT_DATE - INTERVAL '2 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);

-- Usuario 52 (MesoMantener): 4 semanas completas en el pasado (Lunes, Miercoles, Viernes)
-- Semana 1
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 1, 'Lunes', 'Full Body A',
    (CURRENT_DATE - INTERVAL '27 days')::text,
    (CURRENT_DATE - INTERVAL '27 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 1, 'Miercoles', 'Full Body B',
    (CURRENT_DATE - INTERVAL '25 days')::text,
    (CURRENT_DATE - INTERVAL '25 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 1, 'Viernes', 'Full Body C',
    (CURRENT_DATE - INTERVAL '23 days')::text,
    (CURRENT_DATE - INTERVAL '23 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
-- Semana 2
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 2, 'Lunes', 'Full Body A',
    (CURRENT_DATE - INTERVAL '20 days')::text,
    (CURRENT_DATE - INTERVAL '20 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 2, 'Miercoles', 'Full Body B',
    (CURRENT_DATE - INTERVAL '18 days')::text,
    (CURRENT_DATE - INTERVAL '18 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 2, 'Viernes', 'Full Body C',
    (CURRENT_DATE - INTERVAL '16 days')::text,
    (CURRENT_DATE - INTERVAL '16 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
-- Semana 3
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 3, 'Lunes', 'Full Body A',
    (CURRENT_DATE - INTERVAL '13 days')::text,
    (CURRENT_DATE - INTERVAL '13 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 3, 'Miercoles', 'Full Body B',
    (CURRENT_DATE - INTERVAL '11 days')::text,
    (CURRENT_DATE - INTERVAL '11 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 3, 'Viernes', 'Full Body C',
    (CURRENT_DATE - INTERVAL '9 days')::text,
    (CURRENT_DATE - INTERVAL '9 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
-- Semana 4
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 4, 'Lunes', 'Full Body A',
    (CURRENT_DATE - INTERVAL '6 days')::text,
    (CURRENT_DATE - INTERVAL '6 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 4, 'Miercoles', 'Full Body B',
    (CURRENT_DATE - INTERVAL '4 days')::text,
    (CURRENT_DATE - INTERVAL '4 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00052-0000-0000-0000-000000000052', 4, 'Viernes', 'Full Body C',
    (CURRENT_DATE - INTERVAL '2 days')::text,
    (CURRENT_DATE - INTERVAL '2 days')::timestamp AT TIME ZONE 'America/Bogota',
    true
);

-- Usuario 53 (MesoManual): Semana 1 con sesiones en el FUTURO, NO completadas
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00053-0000-0000-0000-000000000053', 1, 'Lunes', 'Full Body A',
    (CURRENT_DATE + INTERVAL '1 day')::text,
    (CURRENT_DATE + INTERVAL '1 day')::timestamp AT TIME ZONE 'America/Bogota',
    false
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00053-0000-0000-0000-000000000053', 1, 'Miercoles', 'Full Body B',
    (CURRENT_DATE + INTERVAL '3 days')::text,
    (CURRENT_DATE + INTERVAL '3 days')::timestamp AT TIME ZONE 'America/Bogota',
    false
);
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (
    'e2e00053-0000-0000-0000-000000000053', 1, 'Viernes', 'Full Body C',
    (CURRENT_DATE + INTERVAL '5 days')::text,
    (CURRENT_DATE + INTERVAL '5 days')::timestamp AT TIME ZONE 'America/Bogota',
    false
);

-- ============================================
-- SECCION 4.5: SETUP PENDING_TASKS [NUEVO v3.0]
-- Simula que el GymBotWorkoutCompletion ya envio
-- el reminder de las 8 PM al usuario
-- ============================================

-- Pending task para Usuario 4 (WithPendingTask): CONFIRMAR_RUTINA
INSERT INTO pending_tasks (task_id, user_id, task_type, related_id, session_name, week, status, created_at)
SELECT
    'e2e00004-0000-0000-0002-000000000004',
    'e2e00004-0000-0000-0000-000000000004',
    'CONFIRMAR_RUTINA',
    day_routine_id,
    'Dia 1 - Pecho y Triceps',
    1,
    'pending',
    NOW() - INTERVAL '2 hours'  -- Simula que se creo hace 2 horas (8 PM reminder)
FROM user_weekly_schedule
WHERE user_id = 'e2e00004-0000-0000-0000-000000000004'
AND planned_day_utc = (DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota')
LIMIT 1;

-- ============================================
-- SECCION 5: SETUP WORKOUTS (Ejercicios)
-- Para Usuario 3 que necesita ver su rutina
-- Nota: sets, reps, rir son TEXT; notes es NOT NULL
-- ============================================

-- Insertar ejercicios de muestra para Usuario 3
INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, created_at, notes)
SELECT
    'e2e00003-0000-0000-0000-000000000003' as user_id,
    1 as week,
    'Dia 1 - Pecho y Triceps' as day_name,
    exercise_id,
    '3' as sets,
    '10' as reps,
    '2' as rir,
    90 as "rest-seconds",
    '2-0-2-0' as tempo,
    NOW() as created_at,
    '' as notes
FROM exercises
WHERE main_muscle IN ('Pecho', 'Triceps')
AND level = 'Principiante'
LIMIT 4;

-- Insertar ejercicios de muestra para Usuario 4 [NUEVO v3.0]
INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, created_at, notes)
SELECT
    'e2e00004-0000-0000-0000-000000000004' as user_id,
    1 as week,
    'Dia 1 - Pecho y Triceps' as day_name,
    exercise_id,
    '3' as sets,
    '10' as reps,
    '2' as rir,
    90 as "rest-seconds",
    '2-0-2-0' as tempo,
    NOW() as created_at,
    '' as notes
FROM exercises
WHERE main_muscle IN ('Pecho', 'Triceps')
AND level = 'Principiante'
LIMIT 4;

-- ============================================
-- SECCION 5.5: SETUP WORKOUTS MESOCYCLE (Usuarios 051, 052, 053) [NUEVO v5.0]
-- Usuarios 051 y 052: 4 semanas x 3 dias = 12 day_names, con ejercicios reales
-- Usuario 053: Solo semana 1
-- Nota: exercise_order: compound (1-3), core (4), isolation (5-6)
-- ============================================

-- Funcion helper: Generar workouts para un usuario mesocycle en una semana/dia especifico
-- Usamos subqueries para obtener exercise_ids reales

-- === Usuario 51 (MesoDetect): 4 semanas de workouts ===

-- Semana 1 - Full Body A (squat + push_h + pull_h + core)
INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body A', exercise_id,
    '4', '8-10', '2', 90, '2-0-2-0', 1, NOW(), ''
FROM exercises WHERE pattern = 'squat' AND role = 'compound' AND level IN ('Intermedio', 'Principiante') LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body A', exercise_id,
    '3', '8-10', '2', 90, '2-0-2-0', 2, NOW(), ''
FROM exercises WHERE pattern = 'push_h' AND role = 'compound' AND level IN ('Intermedio', 'Principiante') LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body A', exercise_id,
    '3', '8-10', '2', 90, '2-0-2-0', 3, NOW(), ''
FROM exercises WHERE pattern = 'pull_h' AND role = 'compound' AND level IN ('Intermedio', 'Principiante') LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body A', exercise_id,
    '3', '12-15', '1', 60, '2-0-2-0', 4, NOW(), ''
FROM exercises WHERE pattern = 'core' AND role = 'core' LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body A', exercise_id,
    '3', '10-12', '2', 60, '2-0-2-0', 5, NOW(), ''
FROM exercises WHERE pattern = 'push_h' AND role = 'isolation' AND level IN ('Intermedio', 'Principiante') LIMIT 1;

-- Semana 1 - Full Body B (hinge + pull_v + push_v + arm + core)
INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body B', exercise_id,
    '4', '6-8', '2', 120, '2-0-2-0', 1, NOW(), ''
FROM exercises WHERE pattern = 'hip_hinge' AND role = 'compound' AND level IN ('Intermedio', 'Principiante') LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body B', exercise_id,
    '3', '8-10', '2', 90, '2-0-2-0', 2, NOW(), ''
FROM exercises WHERE pattern = 'pull_v' AND role = 'compound' AND level IN ('Intermedio', 'Principiante') LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body B', exercise_id,
    '3', '8-10', '2', 90, '2-0-2-0', 3, NOW(), ''
FROM exercises WHERE pattern = 'push_v' AND role = 'compound' AND level IN ('Intermedio', 'Principiante') LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body B', exercise_id,
    '3', '12-15', '1', 60, '2-0-2-0', 4, NOW(), ''
FROM exercises WHERE pattern = 'core' AND role = 'core' LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body B', exercise_id,
    '3', '10-12', '2', 60, '2-0-2-0', 5, NOW(), ''
FROM exercises WHERE pattern = 'arm' AND role = 'isolation' AND level IN ('Intermedio', 'Principiante') LIMIT 1;

-- Semana 1 - Full Body C (squat + hinge + push_h + pull_h + core)
INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body C', exercise_id,
    '3', '10-12', '2', 90, '2-0-2-0', 1, NOW(), ''
FROM exercises WHERE pattern = 'squat' AND role = 'compound' AND level IN ('Intermedio', 'Principiante') OFFSET 1 LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body C', exercise_id,
    '3', '10-12', '2', 90, '2-0-2-0', 2, NOW(), ''
FROM exercises WHERE pattern = 'hip_hinge' AND role = 'compound' AND level IN ('Intermedio', 'Principiante') OFFSET 1 LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body C', exercise_id,
    '3', '10-12', '2', 90, '2-0-2-0', 3, NOW(), ''
FROM exercises WHERE pattern = 'push_h' AND role = 'compound' AND level IN ('Intermedio', 'Principiante') OFFSET 1 LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body C', exercise_id,
    '3', '12-15', '1', 60, '2-0-2-0', 4, NOW(), ''
FROM exercises WHERE pattern = 'core' AND role = 'core' OFFSET 1 LIMIT 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00051-0000-0000-0000-000000000051', 1, 'Full Body C', exercise_id,
    '3', '10-12', '2', 60, '2-0-2-0', 5, NOW(), ''
FROM exercises WHERE pattern = 'pull_h' AND role = 'isolation' AND level IN ('Intermedio', 'Principiante') LIMIT 1;

-- Semanas 2-4 para Usuario 51: Copiar estructura de semana 1 con week cambiado
INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT user_id, 2, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, NOW(), ''
FROM workouts WHERE user_id = 'e2e00051-0000-0000-0000-000000000051' AND week = 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT user_id, 3, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, NOW(), ''
FROM workouts WHERE user_id = 'e2e00051-0000-0000-0000-000000000051' AND week = 1;

INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT user_id, 4, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, NOW(), ''
FROM workouts WHERE user_id = 'e2e00051-0000-0000-0000-000000000051' AND week = 1;

-- === Usuario 52 (MesoMantener): Copiar misma estructura de workouts de Usuario 51 para las 4 semanas ===
INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00052-0000-0000-0000-000000000052', week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, NOW(), ''
FROM workouts WHERE user_id = 'e2e00051-0000-0000-0000-000000000051';

-- === Usuario 53 (MesoManual): Solo semana 1, mismos ejercicios ===
INSERT INTO workouts (user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, created_at, notes)
SELECT 'e2e00053-0000-0000-0000-000000000053', 1, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, exercise_order, NOW(), ''
FROM workouts WHERE user_id = 'e2e00051-0000-0000-0000-000000000051' AND week = 1;

-- ============================================
-- SECCION 6: VERIFICACION
-- Ejecutar para confirmar setup correcto
-- ============================================

-- Verificar usuarios creados (GYM + MESOCYCLE)
SELECT user_id, full_name, full_phone_number
FROM users
WHERE full_phone_number::text LIKE '57000000000%'
ORDER BY full_phone_number;

-- Verificar planes (GYM + MESOCYCLE)
SELECT u.full_name, up.goal, up.level, up.status, up.mesocycle_number
FROM users_plans up
JOIN users u USING (user_id)
WHERE u.full_phone_number::text LIKE '57000000000%';

-- Verificar schedules (mostrando ambas columnas para verificar conversion)
SELECT u.full_name, uws.session_name, uws.week, uws.planned_day, uws.planned_day_utc, uws."Completed"
FROM user_weekly_schedule uws
JOIN users u USING (user_id)
WHERE u.full_phone_number::text LIKE '57000000000%'
ORDER BY u.full_name, uws.planned_day_utc;

-- Verificar workouts (GYM + MESOCYCLE)
SELECT u.full_name, w.week, w.day_name, e.spanish_name, w.sets, w.reps, w.exercise_order
FROM workouts w
JOIN users u USING (user_id)
JOIN exercises e USING (exercise_id)
WHERE u.full_phone_number::text LIKE '57000000000%'
ORDER BY u.full_name, w.week, w.day_name, w.exercise_order;

-- Verificar que Usuario nuevo NO existe
SELECT COUNT(*) as should_be_zero
FROM users
WHERE full_phone_number::text = '570000000009';

-- Verificar pending_tasks [NUEVO v3.0]
SELECT u.full_name, pt.task_type, pt.session_name, pt.status, pt.created_at
FROM pending_tasks pt
JOIN users u USING (user_id)
WHERE u.full_phone_number::text LIKE '57000000000%';

-- ============================================
-- SECCION 7: CLEANUP POST-TEST
-- Ejecutar despues de los tests para resetear estado
-- ============================================

-- Resetear Completed a false para Usuario 3
-- UPDATE user_weekly_schedule
-- SET "Completed" = false
-- WHERE user_id = 'e2e00003-0000-0000-0000-000000000003'
-- AND planned_day_utc = (DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota');

-- Resetear Completed y pending_task para Usuario 4 [NUEVO v3.0]
-- UPDATE user_weekly_schedule
-- SET "Completed" = false
-- WHERE user_id = 'e2e00004-0000-0000-0000-000000000004'
-- AND planned_day_utc = (DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota');

-- UPDATE pending_tasks
-- SET status = 'pending', resolved_at = NULL
-- WHERE user_id = 'e2e00004-0000-0000-0000-000000000004'
-- AND task_type = 'CONFIRMAR_RUTINA';

-- ============================================
-- RESUMEN DE ESTADOS POST-SETUP (v6.0)
-- ============================================
--
-- GYM USERS (fixtures pre-poblados):
-- | Usuario             | Phone        | Plan | Schedule | Workouts | Pending Task | Para Tests          |
-- |---------------------|--------------|------|----------|----------|--------------|---------------------|
-- | Test_NoSchedule     | 570000000001 | SI   | NO       | NO       | NO           | TC003               |
-- | Test_RestDay        | 570000000002 | SI   | MANANA   | NO       | NO           | TC004               |
-- | Test_WithRoutine    | 570000000003 | SI   | HOY      | SI       | NO           | TC006, TC007, TC013 |
-- | Test_WithPendingTask| 570000000004 | SI   | HOY      | SI       | SI           | TC011, TC012        |
-- | (dinamico)          | 570000000009 | -    | -        | -        | -            | TC002, TC002_FULL_KYC|
--
-- HOME USERS (creados dinamicamente por MULTI_TURN_AI, limpiados por teardown):
-- | Usuario             | Phone        | Equipment              | Health | Para Tests      |
-- |---------------------|--------------|------------------------|--------|-----------------|
-- | Maria Lopez         | 570000000211 | mancuernas, bandas     | A      | TC_HOME_001     |
-- | Carlos Rodriguez    | 570000000212 | peso corporal          | A      | TC_HOME_002     |
-- | Ana Martinez        | 570000000213 | mancuernas, bandas     | C      | TC_HOME_003     |
--
-- MESOCYCLE USERS (fixtures pre-poblados, fb_3 / Intermedio / Ganar masa muscular):
-- | Usuario             | Phone        | Schedule       | W4 Complete | Meso# | Para Tests      |
-- |---------------------|--------------|----------------|-------------|-------|-----------------|
-- | Test_MesoDetect     | 570000000051 | 4 sem PASADO   | SI (12/12)  | 1     | TC_MESO_001     |
-- | Test_MesoMantener   | 570000000052 | 4 sem PASADO   | SI (12/12)  | 1     | TC_MESO_002     |
-- | Test_MesoManual     | 570000000053 | 1 sem FUTURO   | NO (0/3)    | 1     | TC_MESO_003     |
--
-- TC001 no necesita usuario (es status update sin messages[])
--
-- NOTA: Los tests HOME usan MULTI_TURN_AI y crean sus datos dinamicamente durante el KYC.
-- El teardown limpia estos usuarios para permitir re-ejecucion de tests.
--
-- Phones incluidos en teardown:
--   GYM: 570000000001, 570000000002, 570000000003, 570000000004, 570000000009
--   HOME: 570000000211, 570000000212, 570000000213
--   MESOCYCLE: 570000000051, 570000000052, 570000000053
--
-- NOTA: Quality Fix test users (570000000061-065) tienen su propio script:
--   e2e/quality_fixes_test_data.sql
-- ============================================
