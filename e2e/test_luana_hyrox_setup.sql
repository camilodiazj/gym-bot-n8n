-- ═══════════════════════════════════════════════════════════════════
-- Test Luana Hyrox — Fixture Setup
-- Replicates Luana Cordero's exact profile for Hyrox scenario testing
-- Phone: 570000000091 (reserved test range)
-- ═══════════════════════════════════════════════════════════════════

-- ══════════════ SECTION 1: TEARDOWN ══════════════

DELETE FROM draft_routines WHERE user_id = 'e2e00091-0000-0000-0000-000000000091';
DELETE FROM magic_links WHERE user_id = 'e2e00091-0000-0000-0000-000000000091';
DELETE FROM pending_tasks WHERE user_id = 'e2e00091-0000-0000-0000-000000000091';
DELETE FROM set_values WHERE user_id = 'e2e00091-0000-0000-0000-000000000091';
DELETE FROM workouts WHERE user_id = 'e2e00091-0000-0000-0000-000000000091';
DELETE FROM user_weekly_schedule WHERE user_id = 'e2e00091-0000-0000-0000-000000000091';
DELETE FROM users_plans WHERE user_id = 'e2e00091-0000-0000-0000-000000000091';
DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000091;
DELETE FROM users WHERE user_id = 'e2e00091-0000-0000-0000-000000000091';

-- Checkpointer cleanup
DELETE FROM checkpoint_blobs WHERE thread_id IN ('case6_570000000091', 'kyc_570000000091');
DELETE FROM checkpoint_writes WHERE thread_id IN ('case6_570000000091', 'kyc_570000000091');
DELETE FROM checkpoints WHERE thread_id IN ('case6_570000000091', 'kyc_570000000091');

-- ══════════════ SECTION 2: USER ══════════════

INSERT INTO users (user_id, full_name, cel_number, full_phone_number, email, timezone, created_at, country_indicative)
VALUES ('e2e00091-0000-0000-0000-000000000091', 'Test Luana Hyrox', 570000000091, '570000000091', 'test.luana.hyrox@test.com', 'America/Bogota', NOW(), 57);

-- ══════════════ SECTION 3: GYM PROFILE (exact copy of Luana Cordero) ══════════════

INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment, home_equipment
) VALUES (
    NOW(), 570000000091, 'Test Luana Hyrox', 'test.luana.hyrox@test.com',
    18, 'F', 165, 60,
    'Bajar grasa', 'Mejorar resistencia',
    '1 a 3 anos', '3-4 dias por semana',
    'Avanzado', 'A', 4,
    '45-60 minutos', 'Manana', 'Maquinas',
    'tren inferior', 'tren superior',
    'Running', '3-4', 'GYM', NULL
);

-- ══════════════ SECTION 4: PLAN (ul_4 like Luana's original) ══════════════

INSERT INTO users_plans (plan_id, user_id, template_id, start_date, goal, level, status, mesocycle_number, week_schedule)
VALUES ('e2e01000-0000-0000-0000-000000000091', 'e2e00091-0000-0000-0000-000000000091',
        'tpl_ul_4_cut_adv', NOW(), 'Bajar grasa', 'Avanzado', 'active', 1, 'ul_4');

-- ══════════════ SECTION 5: TODAY's SCHEDULE (Lunes = Upper A) ══════════════

INSERT INTO user_weekly_schedule (day_routine_id, user_id, week, week_day, session_name, planned_day, planned_day_utc, "Completed")
VALUES (gen_random_uuid(), 'e2e00091-0000-0000-0000-000000000091', 1, 'Lunes', 'Upper A',
        TO_CHAR(NOW() AT TIME ZONE 'America/Bogota', 'YYYY-MM-DD'),
        (DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Bogota') AT TIME ZONE 'America/Bogota'),
        false);

-- ══════════════ SECTION 6: WORKOUTS (4 weeks x 4 days) ══════════════
-- Upper A: 5 exercises, Lower A: 6 exercises, Upper B: 5 exercises, Lower B: 6 exercises

DO $$
DECLARE
    uid UUID := 'e2e00091-0000-0000-0000-000000000091';
    w INT;
BEGIN
    FOR w IN 1..4 LOOP
        -- Upper A (5 exercises)
        INSERT INTO workouts (id, user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, created_at, notes, exercise_order) VALUES
            (gen_random_uuid(), uid, w, 'Upper A', 'ex_barbell_supinated_row', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 1),
            (gen_random_uuid(), uid, w, 'Upper A', 'ex_barbell_pronated_pendlay_row', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 2),
            (gen_random_uuid(), uid, w, 'Upper A', 'ex_dumbbell_alternating_arnold_press', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 3),
            (gen_random_uuid(), uid, w, 'Upper A', 'ex_plate_weighted_dip', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 4),
            (gen_random_uuid(), uid, w, 'Upper A', 'ex_weighted_pull_ups', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 5);

        -- Lower A (6 exercises)
        INSERT INTO workouts (id, user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, created_at, notes, exercise_order) VALUES
            (gen_random_uuid(), uid, w, 'Lower A', 'ex_barbell_heels_up_back_squat', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 1),
            (gen_random_uuid(), uid, w, 'Lower A', 'ex_barbell_bulgarian_split_squat', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 2),
            (gen_random_uuid(), uid, w, 'Lower A', 'ex_barbell_figure_four_heels_elevated_hip_thrust', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 3),
            (gen_random_uuid(), uid, w, 'Lower A', 'ex_plate_staggered_deadlift', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 4),
            (gen_random_uuid(), uid, w, 'Lower A', 'ex_smith_machine_oblique_crunch', '3-4', '10-12', '1', 60, '2-0-1', NOW(), '', 5),
            (gen_random_uuid(), uid, w, 'Lower A', 'ex_cable_standing_crunch', '3-4', '10-12', '1', 60, '2-0-1', NOW(), '', 6);

        -- Upper B (5 exercises)
        INSERT INTO workouts (id, user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, created_at, notes, exercise_order) VALUES
            (gen_random_uuid(), uid, w, 'Upper B', 'ex_cardio_row_erg_rower_legs_only', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 1),
            (gen_random_uuid(), uid, w, 'Upper B', 'ex_swing_to_upright_row', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 2),
            (gen_random_uuid(), uid, w, 'Upper B', 'ex_dumbbell_alternating_arnold_press', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 3),
            (gen_random_uuid(), uid, w, 'Upper B', 'ex_plate_weighted_dip', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 4),
            (gen_random_uuid(), uid, w, 'Upper B', 'ex_weighted_pull_ups', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 5);

        -- Lower B (6 exercises)
        INSERT INTO workouts (id, user_id, week, day_name, exercise_id, sets, reps, rir, "rest-seconds", tempo, created_at, notes, exercise_order) VALUES
            (gen_random_uuid(), uid, w, 'Lower B', 'ex_barbell_figure_four_heels_elevated_hip_thrust', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 1),
            (gen_random_uuid(), uid, w, 'Lower B', 'ex_barbell_staggered_deadlift', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 2),
            (gen_random_uuid(), uid, w, 'Lower B', 'ex_machine_hack_squat', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 3),
            (gen_random_uuid(), uid, w, 'Lower B', 'ex_smith_machine_leg_press', '3', '6-8', '1-2', 180, '3-0-1', NOW(), '', 4),
            (gen_random_uuid(), uid, w, 'Lower B', 'ex_cable_standing_crunch', '3-4', '10-12', '1', 60, '2-0-1', NOW(), '', 5),
            (gen_random_uuid(), uid, w, 'Lower B', 'ex_smith_machine_oblique_crunch', '3-4', '10-12', '1', 60, '2-0-1', NOW(), '', 6);
    END LOOP;
END $$;

-- ══════════════ VERIFICATION ══════════════

SELECT 'Exercises per day (week 1):' AS info;
SELECT day_name, COUNT(*) AS exercises
FROM workouts
WHERE user_id = 'e2e00091-0000-0000-0000-000000000091' AND week = 1
GROUP BY day_name ORDER BY day_name;

SELECT 'Schedule:' AS info;
SELECT week_day, session_name, planned_day, "Completed"
FROM user_weekly_schedule
WHERE user_id = 'e2e00091-0000-0000-0000-000000000091';
