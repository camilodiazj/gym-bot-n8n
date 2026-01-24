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
--
-- TABLAS AFECTADAS:
-- users, users_plans, user_weekly_schedule, workouts,
-- users_gym_profile, n8n_chat_histories, pending_tasks
--
-- ============================================

-- ============================================
-- SECCION 1: TEARDOWN (Limpiar datos de prueba)
-- Ejecutar PRIMERO para estado limpio
-- IMPORTANTE: Incluye 570000000009 en TODAS las queries
-- ============================================

-- Eliminar pending_tasks de usuarios dummy
DELETE FROM pending_tasks
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000001', '570000000002', '570000000003', '570000000004', '570000000009')
);

-- Eliminar workouts de usuarios dummy
DELETE FROM workouts
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000001', '570000000002', '570000000003', '570000000004', '570000000009')
);

-- Eliminar schedules de usuarios dummy
DELETE FROM user_weekly_schedule
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000001', '570000000002', '570000000003', '570000000004', '570000000009')
);

-- Eliminar planes de usuarios dummy
DELETE FROM users_plans
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000001', '570000000002', '570000000003', '570000000004', '570000000009')
);

-- Eliminar usuarios dummy
DELETE FROM users
WHERE full_phone_number::text IN ('570000000001', '570000000002', '570000000003', '570000000004', '570000000009');

-- Eliminar perfiles gym de usuarios dummy (whatsapp_id es BIGINT, no string)
DELETE FROM users_gym_profile
WHERE whatsapp_id IN (570000000001, 570000000002, 570000000003, 570000000004, 570000000009);

-- Limpiar memoria de chat de usuarios dummy
DELETE FROM n8n_chat_histories
WHERE session_id LIKE '%570000000001%'
   OR session_id LIKE '%570000000002%'
   OR session_id LIKE '%570000000003%'
   OR session_id LIKE '%570000000004%'
   OR session_id LIKE '%570000000009%'
   OR session_id LIKE '%e2e00001-0000-0000-0000-000000000001%'
   OR session_id LIKE '%e2e00002-0000-0000-0000-000000000002%'
   OR session_id LIKE '%e2e00003-0000-0000-0000-000000000003%'
   OR session_id LIKE '%e2e00004-0000-0000-0000-000000000004%'
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

-- ============================================
-- SECCION 4: SETUP SCHEDULES
-- Nota: week_day es enum week_days con valores en espanol
-- ============================================

-- Usuario 2 (RestDay): Schedule para MANANA (no hoy)
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES (
    'e2e00002-0000-0000-0000-000000000002',
    1,
    (CASE EXTRACT(DOW FROM CURRENT_DATE + 1)
        WHEN 0 THEN 'Domingo'
        WHEN 1 THEN 'Lunes'
        WHEN 2 THEN 'Martes'
        WHEN 3 THEN 'Miercoles'
        WHEN 4 THEN 'Jueves'
        WHEN 5 THEN 'Viernes'
        WHEN 6 THEN 'Sabado'
    END)::week_days,
    'Dia 1 - Pecho y Triceps',
    (CURRENT_DATE + INTERVAL '1 day')::date::text,
    false
);

-- Usuario 3 (WithRoutine): Schedule para HOY
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES (
    'e2e00003-0000-0000-0000-000000000003',
    1,
    (CASE EXTRACT(DOW FROM CURRENT_DATE)
        WHEN 0 THEN 'Domingo'
        WHEN 1 THEN 'Lunes'
        WHEN 2 THEN 'Martes'
        WHEN 3 THEN 'Miercoles'
        WHEN 4 THEN 'Jueves'
        WHEN 5 THEN 'Viernes'
        WHEN 6 THEN 'Sabado'
    END)::week_days,
    'Dia 1 - Pecho y Triceps',
    CURRENT_DATE::text,
    false
);

-- Usuario 4 (WithPendingTask): Schedule para HOY [NUEVO v3.0]
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES (
    'e2e00004-0000-0000-0000-000000000004',
    1,
    (CASE EXTRACT(DOW FROM CURRENT_DATE)
        WHEN 0 THEN 'Domingo'
        WHEN 1 THEN 'Lunes'
        WHEN 2 THEN 'Martes'
        WHEN 3 THEN 'Miercoles'
        WHEN 4 THEN 'Jueves'
        WHEN 5 THEN 'Viernes'
        WHEN 6 THEN 'Sabado'
    END)::week_days,
    'Dia 1 - Pecho y Triceps',
    CURRENT_DATE::text,
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
AND planned_day = CURRENT_DATE::text
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
-- SECCION 6: VERIFICACION
-- Ejecutar para confirmar setup correcto
-- ============================================

-- Verificar usuarios creados
SELECT user_id, full_name, full_phone_number
FROM users
WHERE full_phone_number::text LIKE '57000000000%'
ORDER BY full_phone_number;

-- Verificar planes
SELECT u.full_name, up.goal, up.level, up.status
FROM users_plans up
JOIN users u USING (user_id)
WHERE u.full_phone_number::text LIKE '57000000000%';

-- Verificar schedules
SELECT u.full_name, uws.session_name, uws.planned_day, uws."Completed"
FROM user_weekly_schedule uws
JOIN users u USING (user_id)
WHERE u.full_phone_number::text LIKE '57000000000%'
ORDER BY uws.planned_day;

-- Verificar workouts
SELECT u.full_name, w.day_name, e.spanish_name, w.sets, w.reps
FROM workouts w
JOIN users u USING (user_id)
JOIN exercises e USING (exercise_id)
WHERE u.full_phone_number::text LIKE '57000000000%';

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
-- AND planned_day = CURRENT_DATE::text;

-- Resetear Completed y pending_task para Usuario 4 [NUEVO v3.0]
-- UPDATE user_weekly_schedule
-- SET "Completed" = false
-- WHERE user_id = 'e2e00004-0000-0000-0000-000000000004'
-- AND planned_day = CURRENT_DATE::text;

-- UPDATE pending_tasks
-- SET status = 'pending', resolved_at = NULL
-- WHERE user_id = 'e2e00004-0000-0000-0000-000000000004'
-- AND task_type = 'CONFIRMAR_RUTINA';

-- ============================================
-- RESUMEN DE ESTADOS POST-SETUP (v4.0)
-- ============================================
--
-- | Usuario             | Phone        | Plan | Schedule | Workouts | Pending Task | Para Tests          |
-- |---------------------|--------------|------|----------|----------|--------------|---------------------|
-- | Test_NoSchedule     | 570000000001 | SI   | NO       | NO       | NO           | TC003               |
-- | Test_RestDay        | 570000000002 | SI   | MANANA   | NO       | NO           | TC004               |
-- | Test_WithRoutine    | 570000000003 | SI   | HOY      | SI       | NO           | TC006, TC007        |
-- | Test_WithPendingTask| 570000000004 | SI   | HOY      | SI       | SI           | TC011, TC012        |
-- | (dinamico)          | 570000000009 | -    | -        | -        | -            | TC002, TC002_FULL_KYC|
--
-- TC001 no necesita usuario (es status update sin messages[])
-- ============================================
