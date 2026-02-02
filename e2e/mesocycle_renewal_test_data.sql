-- ============================================
-- MESOCYCLE RENEWAL TEST DATA
-- GymBot E2E Tests - Phase 6.1
-- ============================================
--
-- Reserved phones: 570000000010-570000000019
--
-- USUARIOS DUMMY (Fixtures):
-- 570000000010 - Test_RenewalComplete (TC_REN_001, TC_REN_002, TC_REN_003, TC_REN_007)
-- 570000000011 - Test_RenewalChangeDays (TC_REN_004)
-- 570000000012 - Test_RenewalRotate (TC_REN_005)
-- 570000000013 - Test_RenewalProfile (TC_REN_006)
--
-- TABLAS AFECTADAS:
-- users, users_plans, user_weekly_schedule, workouts,
-- users_gym_profile, n8n_chat_histories, pending_tasks
--
-- ============================================

-- ============================================
-- SECTION 1: TEARDOWN (Clean existing test data)
-- Execute FIRST for clean state
-- ============================================

-- Delete pending_tasks for renewal test users
DELETE FROM pending_tasks
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000010', '570000000011', '570000000012', '570000000013')
);

-- Delete workouts for renewal test users
DELETE FROM workouts
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000010', '570000000011', '570000000012', '570000000013')
);

-- Delete schedules for renewal test users
DELETE FROM user_weekly_schedule
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000010', '570000000011', '570000000012', '570000000013')
);

-- Delete plans for renewal test users
DELETE FROM users_plans
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN ('570000000010', '570000000011', '570000000012', '570000000013')
);

-- Delete users
DELETE FROM users
WHERE full_phone_number::text IN ('570000000010', '570000000011', '570000000012', '570000000013');

-- Delete gym profiles (whatsapp_id is BIGINT)
DELETE FROM users_gym_profile
WHERE whatsapp_id IN (570000000010, 570000000011, 570000000012, 570000000013);

-- Clean chat histories
DELETE FROM n8n_chat_histories
WHERE session_id LIKE '%570000000010%'
   OR session_id LIKE '%570000000011%'
   OR session_id LIKE '%570000000012%'
   OR session_id LIKE '%570000000013%'
   OR session_id LIKE '%e2e00010%'
   OR session_id LIKE '%e2e00011%'
   OR session_id LIKE '%e2e00012%'
   OR session_id LIKE '%e2e00013%';

-- ============================================
-- SECTION 2: CREATE USERS
-- ============================================

-- User 10: Test_RenewalComplete (week 4 all completed, triggers auto-renewal)
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00010-0000-0000-0000-000000000010',
    'Test RenewalComplete',
    'test_renewal_complete@gymbot.test',
    '570000000010',
    0000000010,
    57,
    'America/Bogota',
    NOW() - INTERVAL '5 weeks'
);

-- User 11: Test_RenewalChangeDays (for changing frequency 4->3 days)
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00011-0000-0000-0000-000000000011',
    'Test RenewalChangeDays',
    'test_renewal_changedays@gymbot.test',
    '570000000011',
    0000000011,
    57,
    'America/Bogota',
    NOW() - INTERVAL '5 weeks'
);

-- User 12: Test_RenewalRotate (for exercise rotation with preferences)
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00012-0000-0000-0000-000000000012',
    'Test RenewalRotate',
    'test_renewal_rotate@gymbot.test',
    '570000000012',
    0000000012,
    57,
    'America/Bogota',
    NOW() - INTERVAL '12 weeks'
);

-- User 13: Test_RenewalProfile (for profile modification flow)
INSERT INTO users (user_id, full_name, email, full_phone_number, cel_number, country_indicative, timezone, created_at)
VALUES (
    'e2e00013-0000-0000-0000-000000000013',
    'Test RenewalProfile',
    'test_renewal_profile@gymbot.test',
    '570000000013',
    0000000013,
    57,
    'America/Bogota',
    NOW() - INTERVAL '5 weeks'
);

-- ============================================
-- SECTION 3: CREATE GYM PROFILES
-- Required for ALL renewal tests (GymRatForm needs profile data)
-- All columns are NOT NULL - must provide all values
-- ============================================

-- Profile for User 10 (renewal complete - needs profile for CAMBIAR_DIAS test)
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex, height_cm, weight_kg,
    primary_goal, secondary_goal, training_experience, current_frequency, fitness_level,
    health_status, days_available, session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency
)
VALUES (
    NOW() - INTERVAL '5 weeks',
    570000000010,
    'Test RenewalComplete',
    'test_renewal_complete@gymbot.test',
    25,
    'M',
    180,
    80,
    'Ganar masa muscular',
    'Ninguna',
    '1 a 3 años',
    '3-4 días por semana',
    'Principiante',
    'A',
    3,
    '45-60 minutos',
    'Mañana',
    'Mixto',
    'Pecho, Espalda',
    'Ninguno',
    'No',
    '0'
);

-- Profile for User 11 (change days test - needs profile for routine regeneration)
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex, height_cm, weight_kg,
    primary_goal, secondary_goal, training_experience, current_frequency, fitness_level,
    health_status, days_available, session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency
)
VALUES (
    NOW() - INTERVAL '5 weeks',
    570000000011,
    'Test RenewalChangeDays',
    'test_renewal_changedays@gymbot.test',
    28,
    'M',
    175,
    75,
    'Ganar masa muscular',
    'Ninguna',
    '1 a 3 años',
    '3-4 días por semana',
    'Intermedio',
    'A',
    4,
    '60-75 minutos',
    'Mañana',
    'Mixto',
    'Pecho, Brazos',
    'Ninguno',
    'No',
    '0'
);

-- Profile for User 12 (rotation test - needs disliked muscles and priority muscles)
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex, height_cm, weight_kg,
    primary_goal, secondary_goal, training_experience, current_frequency, fitness_level,
    health_status, days_available, session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency
)
VALUES (
    NOW() - INTERVAL '12 weeks',
    570000000012,
    'Test RenewalRotate',
    'test_renewal_rotate@gymbot.test',
    30,
    'M',
    175,
    75,
    'Ganar masa muscular',
    'Ninguna',
    'Más de 3 años',
    '3-4 días por semana',
    'Intermedio',
    'A',
    3,
    '60-75 minutos',
    'Mañana',
    'Mixto',
    'Pecho, Espalda',
    'Pantorrillas',
    'No',
    '0'
);

-- Profile for User 13 (profile modification test)
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex, height_cm, weight_kg,
    primary_goal, secondary_goal, training_experience, current_frequency, fitness_level,
    health_status, days_available, session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency
)
VALUES (
    NOW() - INTERVAL '5 weeks',
    570000000013,
    'Test RenewalProfile',
    'test_renewal_profile@gymbot.test',
    28,
    'F',
    165,
    60,
    'Salud general / recomposición corporal',
    'Ninguna',
    '1 a 3 años',
    '3-4 días por semana',
    'Intermedio',
    'A',
    4,
    '45-60 minutos',
    'Mañana',
    'Mixto',
    'Gluteo, Pierna',
    'Ninguno',
    'No',
    '0'
);

-- ============================================
-- SECTION 4: CREATE USER PLANS
-- ============================================

-- Plan for User 10: fb_3 (3 days/week full body), mesocycle 1
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date, mesocycle_number, last_renewal_date)
SELECT
    'e2e00010-0000-0000-0001-000000000010',
    'e2e00010-0000-0000-0000-000000000010',
    template_id,
    'fb_3',
    goal,
    level,
    'active',
    NOW() - INTERVAL '4 weeks',
    1,
    NULL
FROM routine_templates
WHERE week_schedule = 'fb_3' AND level = 'Principiante'
LIMIT 1;

-- Plan for User 11: ul_4 (4 days/week upper/lower), mesocycle 1
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date, mesocycle_number, last_renewal_date)
SELECT
    'e2e00011-0000-0000-0001-000000000011',
    'e2e00011-0000-0000-0000-000000000011',
    template_id,
    'ul_4',
    goal,
    level,
    'active',
    NOW() - INTERVAL '4 weeks',
    1,
    NULL
FROM routine_templates
WHERE week_schedule = 'ul_4' AND level = 'Intermedio'
LIMIT 1;

-- Plan for User 12: fb_3 (3 days/week), mesocycle 3 (advanced user, eligible for rotation)
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date, mesocycle_number, last_renewal_date)
SELECT
    'e2e00012-0000-0000-0001-000000000012',
    'e2e00012-0000-0000-0000-000000000012',
    template_id,
    'fb_3',
    goal,
    level,
    'active',
    NOW() - INTERVAL '4 weeks',
    3,
    NOW() - INTERVAL '4 weeks'
FROM routine_templates
WHERE week_schedule = 'fb_3' AND level = 'Intermedio'
LIMIT 1;

-- Plan for User 13: ul_4 (4 days/week), mesocycle 1
INSERT INTO users_plans (plan_id, user_id, template_id, week_schedule, goal, level, status, start_date, mesocycle_number, last_renewal_date)
SELECT
    'e2e00013-0000-0000-0001-000000000013',
    'e2e00013-0000-0000-0000-000000000013',
    template_id,
    'ul_4',
    goal,
    level,
    'active',
    NOW() - INTERVAL '4 weeks',
    1,
    NULL
FROM routine_templates
WHERE week_schedule = 'ul_4' AND level = 'Intermedio'
LIMIT 1;

-- ============================================
-- SECTION 5: CREATE WEEK 4 SCHEDULES (COMPLETED)
-- All users have completed week 4 to trigger renewal
-- ============================================

-- User 10: 3 sessions in week 4, ALL COMPLETED
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES
    ('e2e00010-0000-0000-0000-000000000010', 4, 'Lunes', 'Full Body A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00010-0000-0000-0000-000000000010', 4, 'Miercoles', 'Full Body B', (NOW() - INTERVAL '5 days')::date::text, true),
    ('e2e00010-0000-0000-0000-000000000010', 4, 'Viernes', 'Full Body C', (NOW() - INTERVAL '3 days')::date::text, true);

-- User 11: 4 sessions in week 4, ALL COMPLETED (ul_4 schedule)
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Lunes', 'Upper A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Martes', 'Lower A', (NOW() - INTERVAL '6 days')::date::text, true),
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Jueves', 'Upper B', (NOW() - INTERVAL '4 days')::date::text, true),
    ('e2e00011-0000-0000-0000-000000000011', 4, 'Viernes', 'Lower B', (NOW() - INTERVAL '3 days')::date::text, true);

-- User 12: 3 sessions in week 4, ALL COMPLETED (rotation candidate)
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES
    ('e2e00012-0000-0000-0000-000000000012', 4, 'Lunes', 'Full Body A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00012-0000-0000-0000-000000000012', 4, 'Miercoles', 'Full Body B', (NOW() - INTERVAL '5 days')::date::text, true),
    ('e2e00012-0000-0000-0000-000000000012', 4, 'Viernes', 'Full Body C', (NOW() - INTERVAL '3 days')::date::text, true);

-- User 13: 4 sessions in week 4, ALL COMPLETED
INSERT INTO user_weekly_schedule (user_id, week, week_day, session_name, planned_day, "Completed")
VALUES
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Lunes', 'Upper A', (NOW() - INTERVAL '7 days')::date::text, true),
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Martes', 'Lower A', (NOW() - INTERVAL '6 days')::date::text, true),
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Jueves', 'Upper B', (NOW() - INTERVAL '4 days')::date::text, true),
    ('e2e00013-0000-0000-0000-000000000013', 4, 'Viernes', 'Lower B', (NOW() - INTERVAL '3 days')::date::text, true);

-- ============================================
-- SECTION 6: CREATE WORKOUTS FOR USER 12
-- User 12 needs actual workouts for rotation testing
-- Exercise order: compound (1-4), core (5-6), isolation (7+)
-- ============================================

-- Full Body A - Day 1 workouts for User 12
INSERT INTO workouts (id, user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, created_at, notes, exercise_order)
VALUES
    -- Compound exercises (order 1-4)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body A', 'ex_039', '4', '8-10', '2', 120, '2-0-2-0', NOW(), '', 1),  -- Prensa de piernas (squat)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body A', 'ex_022', '4', '8-10', '2', 90, '2-0-2-0', NOW(), '', 2),   -- Chest press (push_h)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body A', 'ex_006', '4', '8-10', '2', 90, '2-0-2-0', NOW(), '', 3),   -- Remo en maquina (pull_h)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body A', 'ex_barbell_hip_thrust', '3', '10-12', '2', 90, '2-0-2-0', NOW(), '', 4),  -- Hip thrust (hinge)
    -- Core exercises (order 5-6)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body A', 'ex_040', '3', '30-45s', '0', 60, 'hold', NOW(), '', 5),   -- Plancha (core)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body A', 'ex_bicycle_crunch', '3', '15-20', '1', 60, '1-0-1-0', NOW(), '', 6);  -- Bicycle crunch (core)

-- Full Body B - Day 2 workouts for User 12
INSERT INTO workouts (id, user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, created_at, notes, exercise_order)
VALUES
    -- Compound exercises (order 1-4)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body B', 'ex_033', '4', '8-10', '2', 120, '2-0-2-0', NOW(), '', 1),  -- Sentadilla goblet (squat)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body B', 'ex_008', '4', '8-10', '2', 90, '2-0-2-0', NOW(), '', 2),   -- Jalon triangulo (pull_v)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body B', 'ex_017', '3', '10-12', '2', 90, '2-0-2-0', NOW(), '', 3),  -- Press militar (push_v)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body B', 'ex_dumbbell_romanian_deadlift', '3', '10-12', '2', 90, '2-0-2-0', NOW(), '', 4),  -- RDL mancuerna (hinge)
    -- Core exercises (order 5-6)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body B', 'ex_band_pallof_press', '3', '12-15', '1', 60, '2-1-2-0', NOW(), '', 5);  -- Pallof press (core)

-- Full Body C - Day 3 workouts for User 12
INSERT INTO workouts (id, user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, created_at, notes, exercise_order)
VALUES
    -- Compound exercises (order 1-4)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body C', 'ex_machine_hack_squat', '4', '8-10', '2', 120, '2-0-2-0', NOW(), '', 1),  -- Hack squat (squat)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body C', 'ex_023', '4', '8-10', '2', 90, '2-0-2-0', NOW(), '', 2),   -- Press inclinado mancuerna (push_h)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body C', 'ex_025', '4', '8-10', '2', 90, '2-0-2-0', NOW(), '', 3),   -- Remo sentado (pull_h)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body C', 'ex_dumbbell_hip_thrust', '3', '10-12', '2', 90, '2-0-2-0', NOW(), '', 4),  -- Hip thrust mancuerna (hinge)
    -- Core exercises (order 5-6)
    (gen_random_uuid(), 'e2e00012-0000-0000-0000-000000000012', 4, 'Full Body C', 'ex_041', '3', '10-15', '1', 60, '2-0-2-0', NOW(), '', 5);  -- Abdominales colgado (core)

-- ============================================
-- SECTION 7: VERIFICATION QUERIES
-- Execute to confirm correct setup
-- ============================================

-- Verify users created
SELECT user_id, full_name, full_phone_number
FROM users
WHERE full_phone_number::text LIKE '5700000001%'
ORDER BY full_phone_number;

-- Verify plans with mesocycle info
SELECT u.full_name, up.week_schedule, up.goal, up.level, up.mesocycle_number, up.status
FROM users_plans up
JOIN users u USING (user_id)
WHERE u.full_phone_number::text LIKE '5700000001%';

-- Verify week 4 completion status
SELECT u.full_name,
       COUNT(*) as total_sessions,
       COUNT(*) FILTER (WHERE "Completed" = true) as completed_sessions
FROM user_weekly_schedule uws
JOIN users u USING (user_id)
WHERE u.full_phone_number::text LIKE '5700000001%'
AND uws.week = 4
GROUP BY u.full_name;

-- Verify gym profiles for rotation test
SELECT whatsapp_id, priority_muscles, disliked_exercises, health_status
FROM users_gym_profile
WHERE whatsapp_id IN (570000000012, 570000000013);

-- Verify workouts for rotation test user (User 12)
SELECT u.full_name, w.day_name, e.spanish_name, e.pattern, e.role, w.exercise_order
FROM workouts w
JOIN users u USING (user_id)
JOIN exercises e USING (exercise_id)
WHERE u.full_phone_number = '570000000012'
ORDER BY w.day_name, w.exercise_order;

-- ============================================
-- SECTION 8: CLEANUP / RESET HELPERS
-- ============================================

-- OPTION 1: Full teardown (run Section 1)

-- OPTION 2: Soft reset - restore initial state for re-running tests

-- Reset mesocycle numbers to initial values
-- UPDATE users_plans SET mesocycle_number = 1, last_renewal_date = NULL
-- WHERE user_id IN ('e2e00010-0000-0000-0000-000000000010', 'e2e00011-0000-0000-0000-000000000011', 'e2e00013-0000-0000-0000-000000000013');

-- UPDATE users_plans SET mesocycle_number = 3, last_renewal_date = NOW() - INTERVAL '4 weeks'
-- WHERE user_id = 'e2e00012-0000-0000-0000-000000000012';

-- Reset week 4 schedules to completed
-- UPDATE user_weekly_schedule SET "Completed" = true
-- WHERE user_id IN ('e2e00010-0000-0000-0000-000000000010', 'e2e00011-0000-0000-0000-000000000011', 'e2e00012-0000-0000-0000-000000000012', 'e2e00013-0000-0000-0000-000000000013')
-- AND week = 4;

-- Reset week_schedule for User 11 (change days test)
-- UPDATE users_plans SET week_schedule = 'ul_4'
-- WHERE user_id = 'e2e00011-0000-0000-0000-000000000011';

-- Clear any newly created schedules (week < 4)
-- DELETE FROM user_weekly_schedule
-- WHERE user_id IN ('e2e00010-0000-0000-0000-000000000010', 'e2e00011-0000-0000-0000-000000000011', 'e2e00012-0000-0000-0000-000000000012', 'e2e00013-0000-0000-0000-000000000013')
-- AND week < 4;

-- Clear chat histories for fresh conversation state
-- DELETE FROM n8n_chat_histories
-- WHERE session_id LIKE '%5700000001%';

-- ============================================
-- SUMMARY OF STATES POST-SETUP
-- ============================================
--
-- | Usuario               | Phone        | Plan  | Week4   | Meso# | Workouts | Purpose                    |
-- |-----------------------|--------------|-------|---------|-------|----------|----------------------------|
-- | Test_RenewalComplete  | 570000000010 | fb_3  | DONE    | 1     | NO       | TC_REN_001-003, 007        |
-- | Test_RenewalChangeDays| 570000000011 | ul_4  | DONE    | 1     | NO       | TC_REN_004 (change to 3d)  |
-- | Test_RenewalRotate    | 570000000012 | fb_3  | DONE    | 3     | YES      | TC_REN_005 (rotate exs)    |
-- | Test_RenewalProfile   | 570000000013 | ul_4  | DONE    | 1     | NO       | TC_REN_006 (modify profile)|
--
-- ============================================
