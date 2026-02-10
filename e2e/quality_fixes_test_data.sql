-- ============================================
-- SCRIPT DE DATOS DE PRUEBA - Quality Fixes
-- WORKOUT_CREATOR v6.1
-- ============================================
--
-- PROPOSITO: Crear fixture users para validar los 5 quality fixes (QF-1 a QF-5)
-- despues de importar el WORKOUT_CREATOR.json actualizado en n8n.
--
-- COMO USAR:
--   1. Ejecutar SECCION 1 + SECCION 2 (teardown + setup) en Supabase SQL Editor
--   2. Importar n8n/running_flows/WORKOUT_CREATOR.json en tu instancia n8n
--   3. Para cada test: poner el telefono en el nodo trigger y ejecutar el workflow
--   4. Despues de cada ejecucion: correr la query de validacion correspondiente
--   5. Para re-ejecutar: correr SECCION 1 (teardown) primero, luego SECCION 2
--
-- USUARIOS DE PRUEBA:
-- | Phone        | Nombre          | Health | Proposito                              |
-- |--------------|-----------------|--------|----------------------------------------|
-- | 570000000061 | Test_QF_HealthB | B      | T-401: Sin saltos/impacto tren inferior|
-- | 570000000062 | Test_QF_HealthC | C      | T-402: Sin press militar/overhead      |
-- | 570000000063 | Test_QF_HealthD | D      | T-401: Sin peso muerto/carga axial     |
-- | 570000000064 | Test_QF_HealthE | E      | T-405: Solo maquinas/cables            |
-- | 570000000065 | Test_QF_Standard| A      | T-403/T-404/T-405: Dedup, VolCap, Cardio|
-- | 570000000066 | Test_QF_5Day    | A      | T-406: Rutina estandar 5 dias, validacion completa|
--
-- ============================================

-- ============================================
-- SECCION 1: TEARDOWN (ejecutar antes de cada test run)
-- ============================================

-- Eliminar workouts
DELETE FROM workouts
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN (
        '570000000061', '570000000062', '570000000063', '570000000064', '570000000065', '570000000066'
    )
);

-- Eliminar schedules
DELETE FROM user_weekly_schedule
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN (
        '570000000061', '570000000062', '570000000063', '570000000064', '570000000065', '570000000066'
    )
);

-- Eliminar planes
DELETE FROM users_plans
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN (
        '570000000061', '570000000062', '570000000063', '570000000064', '570000000065', '570000000066'
    )
);

-- Eliminar pending_tasks
DELETE FROM pending_tasks
WHERE user_id IN (
    SELECT user_id FROM users
    WHERE full_phone_number::text IN (
        '570000000061', '570000000062', '570000000063', '570000000064', '570000000065', '570000000066'
    )
);

-- Eliminar usuarios
DELETE FROM users
WHERE full_phone_number::text IN (
    '570000000061', '570000000062', '570000000063', '570000000064', '570000000065', '570000000066'
);

-- Eliminar perfiles gym (whatsapp_id es BIGINT)
DELETE FROM users_gym_profile
WHERE whatsapp_id IN (
    570000000061, 570000000062, 570000000063, 570000000064, 570000000065, 570000000066
);

-- Eliminar chat histories
DELETE FROM n8n_chat_histories
WHERE session_id LIKE '%570000000061%'
   OR session_id LIKE '%570000000062%'
   OR session_id LIKE '%570000000063%'
   OR session_id LIKE '%570000000064%'
   OR session_id LIKE '%570000000065%'
   OR session_id LIKE '%570000000066%';

-- ============================================
-- SECCION 2: SETUP GYM PROFILES
-- Solo se necesita users_gym_profile; WORKOUT_CREATOR
-- crea automaticamente users, users_plans y workouts.
-- ============================================

-- Test_QF_HealthB: Restriccion tren inferior (rodillas/tobillos)
-- Espera: NO saltos, NO burpees, NO sentadilla bulgara
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment
) VALUES (
    NOW(), 570000000061, 'Test QF HealthB', 'test_qf_healthb@gymbot.test', 32, 'M',
    180, 85, 'Ganar masa muscular', 'Ninguna', '1 a 3 años',
    '3-4 días por semana', 'Intermedio', 'B', 3,
    '60-75 minutos', 'Mañana', 'Mixto', 'Pecho, Espalda',
    'Ninguno', 'No', '0',
    'GYM'
);

-- Test_QF_HealthC: Restriccion tren superior (hombros)
-- Espera: NO press militar, NO overhead, NO snatch, NO push press
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment
) VALUES (
    NOW(), 570000000062, 'Test QF HealthC', 'test_qf_healthc@gymbot.test', 28, 'F',
    165, 62, 'Ganar masa muscular', 'Ninguna', '1 a 3 años',
    '3-4 días por semana', 'Intermedio', 'C', 3,
    '60-75 minutos', 'Tarde', 'Mixto', 'Glúteo, Pierna',
    'Ninguno', 'No', '0',
    'GYM'
);

-- Test_QF_HealthD: Restriccion columna (discos/vertebras)
-- Espera: NO peso muerto, NO good morning, NO back squat con barra
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment
) VALUES (
    NOW(), 570000000063, 'Test QF HealthD', 'test_qf_healthd@gymbot.test', 35, 'M',
    175, 82, 'Ganar masa muscular', 'Ninguna', '1 a 3 años',
    '3-4 días por semana', 'Intermedio', 'D', 3,
    '60-75 minutos', 'Mañana', 'Máquinas', 'Pecho, Espalda',
    'Ninguno', 'No', '0',
    'GYM'
);

-- Test_QF_HealthE: Condicion especial (solo maquinas/cables)
-- Espera: NO barbell, solo machine/cable/bodyweight(core)
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment
) VALUES (
    NOW(), 570000000064, 'Test QF HealthE', 'test_qf_healthe@gymbot.test', 45, 'M',
    170, 78, 'Salud general / recomposición corporal', 'Ninguna', '6 a 12 meses',
    '1-2 días por semana', 'Principiante', 'E', 3,
    '45-60 minutos', 'Mañana', 'Máquinas', 'Pecho, Espalda',
    'Ninguno', 'No', '0',
    'GYM'
);

-- Test_QF_Standard: Sin restricciones (Health A)
-- Para: T-403 (dedup), T-404 (volume cap W4), T-405 (cardio role)
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment
) VALUES (
    NOW(), 570000000065, 'Test QF Standard', 'test_qf_standard@gymbot.test', 27, 'M',
    178, 80, 'Ganar masa muscular', 'Ninguna', '1 a 3 años',
    '3-4 días por semana', 'Intermedio', 'A', 3,
    '60-75 minutos', 'Mañana', 'Mixto', 'Pecho, Espalda',
    'Pantorrillas', 'No', '0',
    'GYM'
);

-- Test_QF_5Day: Rutina estandar 5 dias (Health A, avanzado)
-- Para: Validacion integral de 5 quality fixes en split de 5 dias
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style,
    priority_muscles, disliked_exercises, cardio_type, cardio_frequency,
    training_environment
) VALUES (
    NOW(), 570000000066, 'Test QF 5Day', 'test_qf_5day@gymbot.test', 30, 'M',
    182, 88, 'Ganar masa muscular', 'Mejorar fuerza', 'Más de 3 años',
    '5-6 días por semana', 'Avanzado', 'A', 5,
    '60-75 minutos', 'Mañana', 'Pesas libres', 'Pecho, Espalda',
    'Pantorrillas', 'Running', '3-4',
    'GYM'
);

-- ============================================
-- SECCION 3: VERIFICACION SETUP
-- Ejecutar para confirmar que los perfiles se crearon
-- ============================================

SELECT whatsapp_id, full_name, health_status, primary_goal, fitness_level, training_environment
FROM users_gym_profile
WHERE whatsapp_id IN (570000000061, 570000000062, 570000000063, 570000000064, 570000000065, 570000000066)
ORDER BY whatsapp_id;


-- ############################################
-- ##                                        ##
-- ##   TESTS DE VALIDACION POST-EJECUCION   ##
-- ##                                        ##
-- ##   Ejecutar DESPUES de cada trigger      ##
-- ##   del WORKOUT_CREATOR en n8n            ##
-- ##                                        ##
-- ############################################


-- ============================================
-- TEST T-401: Health B — Sin impacto tren inferior
-- Phone: 570000000061
-- Ejecutar WORKOUT_CREATOR, luego correr esta query
-- ESPERADO: 0 filas (ninguna violacion)
-- ============================================

SELECT 'T-401 VIOLACIONES' as test,
       w.exercise_id, e.spanish_name, e.main_muscle, e.equipment, e.pattern
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000061'
AND (
    e.spanish_name ILIKE '%salto%'
    OR e.spanish_name ILIKE '%jump%'
    OR e.spanish_name ILIKE '%burpee%'
    OR e.spanish_name ILIKE '%box jump%'
    OR e.spanish_name ILIKE '%sentadilla búlgara%'
)
ORDER BY w.day_name, w.exercise_order;
-- Si aparecen filas -> FAIL: El filtro SQL de health B no funciono


-- ============================================
-- TEST T-402: Health C — Sin overhead/press militar
-- Phone: 570000000062
-- Ejecutar WORKOUT_CREATOR, luego correr esta query
-- ESPERADO: 0 filas (ninguna violacion)
-- ============================================

SELECT 'T-402 VIOLACIONES' as test,
       w.exercise_id, e.spanish_name, e.main_muscle, e.equipment, e.pattern
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000062'
AND (
    e.spanish_name ILIKE '%press militar%'
    OR e.spanish_name ILIKE '%overhead%'
    OR e.spanish_name ILIKE '%snatch%'
    OR e.spanish_name ILIKE '%press de hombro%'
    OR e.spanish_name ILIKE '%push press%'
    OR e.spanish_name ILIKE '%press arnold%'
)
ORDER BY w.day_name, w.exercise_order;
-- Si aparecen filas -> FAIL: El filtro SQL de health C no funciono


-- ============================================
-- TEST T-401b: Health D — Sin carga axial/deadlifts
-- Phone: 570000000063
-- Ejecutar WORKOUT_CREATOR, luego correr esta query
-- ESPERADO: 0 filas (ninguna violacion)
-- ============================================

SELECT 'T-401b VIOLACIONES' as test,
       w.exercise_id, e.spanish_name, e.main_muscle, e.equipment, e.pattern
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000063'
AND (
    e.main_muscle = 'Lower back'
    OR e.spanish_name ILIKE '%peso muerto%'
    OR e.spanish_name ILIKE '%deadlift%'
    OR e.spanish_name ILIKE '%buenos días%'
    OR e.spanish_name ILIKE '%good morning%'
)
ORDER BY w.day_name, w.exercise_order;
-- Si aparecen filas -> FAIL: El filtro SQL de health D no funciono


-- ============================================
-- TEST T-401c: Health E — Solo maquinas/cables/bodyweight(core)
-- Phone: 570000000064
-- Ejecutar WORKOUT_CREATOR, luego correr esta query
-- ESPERADO: 0 filas (ninguna violacion)
-- ============================================

SELECT 'T-401c VIOLACIONES' as test,
       w.exercise_id, e.spanish_name, e.equipment, e.role, e.pattern
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000064'
AND e.equipment = 'barbell'
ORDER BY w.day_name, w.exercise_order;
-- Si aparecen filas -> FAIL: El filtro SQL de health E no excluyo barbell


-- ============================================
-- TEST T-403: Dedup — Sin ejercicios duplicados por dia
-- Phone: 570000000065
-- Ejecutar WORKOUT_CREATOR, luego correr esta query
-- ESPERADO: 0 filas (sin duplicados)
-- NOTA: Ejecutar 3 veces (borrar workouts entre cada run)
-- ============================================

SELECT 'T-403 DUPLICADOS' as test,
       w.day_name, w.week, w.exercise_id, e.spanish_name, COUNT(*) as veces
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000065'
GROUP BY w.day_name, w.week, w.exercise_id, e.spanish_name
HAVING COUNT(*) > 1
ORDER BY w.day_name, w.week;
-- Si aparecen filas -> FAIL: El guard de dedup no elimino duplicados


-- ============================================
-- TEST T-404: Volume Cap — W2-W4 nunca mas ejercicios que W1
-- Phone: 570000000065
-- Ejecutar WORKOUT_CREATOR, luego correr esta query
-- ESPERADO: Todas las filas con w2/w3/w4 count <= w1_count
-- ============================================

WITH exercise_counts AS (
    SELECT w.day_name, w.week, COUNT(DISTINCT w.exercise_id) as exercise_count
    FROM workouts w
    JOIN users u USING(user_id)
    WHERE u.full_phone_number::text = '570000000065'
    GROUP BY w.day_name, w.week
),
w1_counts AS (
    SELECT day_name, exercise_count as w1_count
    FROM exercise_counts
    WHERE week = 1
)
SELECT
    ec.day_name,
    ec.week,
    ec.exercise_count,
    w1.w1_count,
    CASE
        WHEN ec.exercise_count > w1.w1_count THEN 'FAIL: W' || ec.week || ' > W1'
        ELSE 'PASS'
    END as resultado
FROM exercise_counts ec
JOIN w1_counts w1 USING(day_name)
ORDER BY ec.day_name, ec.week;
-- Si alguna fila dice FAIL -> El volume cap no funciono


-- ============================================
-- TEST T-405: Cardio — Parametros correctos para role=cardio
-- Phone: 570000000065
-- Ejecutar WORKOUT_CREATOR, luego correr esta query
-- ESPERADO: Si hay cardio, reps debe ser en segundos (ej: 20s, 30s),
--           tempo debe ser '--', exercise_order debe ser el mas alto del dia
-- ============================================

-- 5a: Verificar que ejercicios cardio tienen parametros correctos
SELECT 'T-405a CARDIO PARAMS' as test,
       w.day_name, w.week, w.exercise_order,
       e.spanish_name, e.role,
       w.sets, w.reps, w.tempo, w."rest-seconds"
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000065'
AND e.role = 'cardio'
ORDER BY w.day_name, w.week, w.exercise_order;
-- Verificar: reps contiene 's' (segundos), tempo es '--' o 'N/A'

-- 5b: Verificar que cardio tiene exercise_order mas alto que isolation
SELECT 'T-405b CARDIO ORDER' as test,
       w.day_name, w.week,
       e.role, w.exercise_order, e.spanish_name
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000065'
AND e.role IN ('isolation', 'cardio')
ORDER BY w.day_name, w.week, w.exercise_order;
-- Verificar: cardio siempre aparece DESPUES de isolation

-- 5c: Verificar que NO hay ejercicios cardio con parametros de isolation
SELECT 'T-405c CARDIO NO ISOLATION PARAMS' as test,
       w.exercise_id, e.spanish_name, e.role,
       w.sets, w.reps, w.tempo
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000065'
AND e.role = 'cardio'
AND w.tempo NOT IN ('--', 'N/A', 'continuous')
ORDER BY w.day_name;
-- ESPERADO: 0 filas. Si aparecen -> cardio recibio params de isolation (FAIL)


-- ============================================
-- TEST T-404b: Deload check — W4 sets deben ser menores que W1
-- Phone: 570000000065
-- Ejecutar despues de WORKOUT_CREATOR
-- ============================================

SELECT 'T-404b DELOAD SETS' as test,
       w.day_name,
       w.exercise_id,
       e.spanish_name,
       w.sets as w4_sets
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000065'
AND w.week = 4
ORDER BY w.day_name, w.exercise_order;
-- Verificar: todos los sets son 2 o 2-3 (deload)


-- ============================================
-- TEST T-QF3: Misclassified — Verificar que push day no tiene Abs
-- Phone: 570000000065
-- ============================================

SELECT 'T-QF3 PATTERN MISMATCH' as test,
       w.day_name, e.spanish_name, e.pattern, e.main_muscle, e.role
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000065'
AND (
    -- Abs en un dia Push (deberian estar en core)
    (w.day_name ILIKE '%push%' AND e.main_muscle = 'Abs')
    -- Back en un dia Push
    OR (w.day_name ILIKE '%push%' AND e.main_muscle IN ('Back', 'Lats'))
    -- Chest en un dia Pull
    OR (w.day_name ILIKE '%pull%' AND e.main_muscle = 'Chest')
)
ORDER BY w.day_name, w.exercise_order;
-- ESPERADO: 0 filas


-- ############################################
-- ##                                        ##
-- ##   TESTS USUARIO 5 DIAS (570000000066)  ##
-- ##                                        ##
-- ############################################


-- ============================================
-- TEST T-406a: 5-Day — Dedup por dia
-- ESPERADO: 0 filas
-- ============================================

SELECT 'T-406a DEDUP 5DAY' as test,
       w.day_name, w.week, w.exercise_id, e.spanish_name, COUNT(*) as veces
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000066'
GROUP BY w.day_name, w.week, w.exercise_id, e.spanish_name
HAVING COUNT(*) > 1
ORDER BY w.day_name, w.week;


-- ============================================
-- TEST T-406b: 5-Day — Volume cap W2-W4 <= W1
-- ESPERADO: Todas las filas PASS
-- ============================================

WITH exercise_counts AS (
    SELECT w.day_name, w.week, COUNT(DISTINCT w.exercise_id) as exercise_count
    FROM workouts w
    JOIN users u USING(user_id)
    WHERE u.full_phone_number::text = '570000000066'
    GROUP BY w.day_name, w.week
),
w1_counts AS (
    SELECT day_name, exercise_count as w1_count
    FROM exercise_counts
    WHERE week = 1
)
SELECT
    ec.day_name,
    ec.week,
    ec.exercise_count,
    w1.w1_count,
    CASE
        WHEN ec.exercise_count > w1.w1_count THEN 'FAIL: W' || ec.week || ' > W1'
        ELSE 'PASS'
    END as resultado
FROM exercise_counts ec
JOIN w1_counts w1 USING(day_name)
ORDER BY ec.day_name, ec.week;


-- ============================================
-- TEST T-406c: 5-Day — Deload W4 sets
-- ESPERADO: Todos los sets = 2
-- ============================================

SELECT 'T-406c DELOAD 5DAY' as test,
       w.day_name, w.exercise_order,
       e.spanish_name, e.role,
       w.sets as w4_sets
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000066'
AND w.week = 4
ORDER BY w.day_name, w.exercise_order;


-- ============================================
-- TEST T-406d: 5-Day — Cardio params correctos
-- ESPERADO: Si hay cardio, reps en segundos y tempo '--'
-- ============================================

SELECT 'T-406d CARDIO 5DAY' as test,
       w.day_name, w.week, w.exercise_order,
       e.spanish_name, e.role,
       w.sets, w.reps, w.tempo, w."rest-seconds"
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000066'
AND e.role = 'cardio'
ORDER BY w.day_name, w.week, w.exercise_order;


-- ============================================
-- TEST T-406e: 5-Day — Cardio no tiene params de isolation
-- ESPERADO: 0 filas
-- ============================================

SELECT 'T-406e CARDIO NO ISO' as test,
       w.exercise_id, e.spanish_name, e.role,
       w.sets, w.reps, w.tempo
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000066'
AND e.role = 'cardio'
AND w.tempo NOT IN ('--', 'N/A', 'continuous')
ORDER BY w.day_name;


-- ============================================
-- TEST T-406f: 5-Day — Tiene exactamente 5 dias distintos
-- ESPERADO: 5 filas
-- ============================================

SELECT 'T-406f DIAS' as test,
       w.day_name, COUNT(DISTINCT w.exercise_id) as ejercicios_w1
FROM workouts w
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000066'
AND w.week = 1
GROUP BY w.day_name
ORDER BY w.day_name;


-- ============================================
-- TEST T-406g: 5-Day — Roles correctos (compound primero, isolation despues)
-- ESPERADO: compound order < core order < isolation order < cardio order
-- ============================================

SELECT 'T-406g ROLE ORDER' as test,
       w.day_name, w.week,
       e.role, w.exercise_order, e.spanish_name
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000066'
AND w.week = 1
ORDER BY w.day_name, w.exercise_order;


-- ============================================
-- TEST T-406h: 5-Day — Resumen general por semana
-- ESPERADO: Vista completa de la rutina generada
-- ============================================

SELECT
    w.day_name,
    w.week,
    COUNT(DISTINCT w.exercise_id) as total_exercises,
    COUNT(DISTINCT CASE WHEN e.role = 'compound' THEN w.exercise_id END) as compounds,
    COUNT(DISTINCT CASE WHEN e.role = 'core' THEN w.exercise_id END) as cores,
    COUNT(DISTINCT CASE WHEN e.role = 'isolation' THEN w.exercise_id END) as isolations,
    COUNT(DISTINCT CASE WHEN e.role = 'cardio' THEN w.exercise_id END) as cardios
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000066'
GROUP BY w.day_name, w.week
ORDER BY w.day_name, w.week;


-- ============================================
-- RESUMEN RAPIDO: Query de resumen post-ejecucion
-- Correr despues de ejecutar WORKOUT_CREATOR para
-- cualquier usuario QF para ver el panorama general
-- ============================================

-- Cambiar el phone por el usuario que acabas de ejecutar
-- Ejemplo: '570000000065'

/*
SELECT
    w.day_name,
    w.week,
    w.exercise_order,
    e.spanish_name,
    e.role,
    e.pattern,
    e.main_muscle,
    e.equipment,
    w.sets,
    w.reps,
    w.rir,
    w."rest-seconds",
    w.tempo
FROM workouts w
JOIN exercises e USING(exercise_id)
JOIN users u USING(user_id)
WHERE u.full_phone_number::text = '570000000065'
ORDER BY w.day_name, w.week, w.exercise_order;
*/


-- ============================================
-- CLEANUP RAPIDO: Borrar workouts de un usuario especifico
-- para re-ejecutar el test sin correr todo el teardown
-- ============================================

/*
-- Cambiar phone por el usuario a limpiar
DELETE FROM workouts WHERE user_id IN (
    SELECT user_id FROM users WHERE full_phone_number::text = '570000000065'
);
DELETE FROM user_weekly_schedule WHERE user_id IN (
    SELECT user_id FROM users WHERE full_phone_number::text = '570000000065'
);
DELETE FROM users_plans WHERE user_id IN (
    SELECT user_id FROM users WHERE full_phone_number::text = '570000000065'
);
DELETE FROM users WHERE full_phone_number::text = '570000000065';
-- No borrar users_gym_profile (se necesita para re-ejecutar WORKOUT_CREATOR)
*/


-- ============================================
-- RESUMEN DE TESTS (v6.1)
-- ============================================
--
-- | Test   | Phone        | Fix  | Que valida                                | Query resultado |
-- |--------|--------------|------|-------------------------------------------|-----------------|
-- | T-401  | 570000000061 | QF-5 | Health B: no saltos/burpees               | 0 filas = PASS  |
-- | T-402  | 570000000062 | QF-5 | Health C: no overhead/press militar       | 0 filas = PASS  |
-- | T-401b | 570000000063 | QF-5 | Health D: no peso muerto/carga axial      | 0 filas = PASS  |
-- | T-401c | 570000000064 | QF-5 | Health E: no barbell                      | 0 filas = PASS  |
-- | T-403  | 570000000065 | QF-2 | Dedup: no exercise_id duplicado por dia   | 0 filas = PASS  |
-- | T-404  | 570000000065 | QF-1 | Volume cap: W2-W4 <= W1 count            | Todo PASS       |
-- | T-404b | 570000000065 | QF-1 | Deload: W4 sets bajos                    | Sets = 2 o 2-3  |
-- | T-405a | 570000000065 | QF-4 | Cardio: reps en segundos, tempo '--'      | Visual check    |
-- | T-405b | 570000000065 | QF-4 | Cardio: exercise_order mas alto           | Visual check    |
-- | T-405c | 570000000065 | QF-4 | Cardio: no params de isolation            | 0 filas = PASS  |
-- | T-QF3  | 570000000065 | QF-3 | Pattern: no Abs en Push, no Back en Push  | 0 filas = PASS  |
-- | T-406a | 570000000066 | QF-2 | 5-Day: no duplicados por dia               | 0 filas = PASS  |
-- | T-406b | 570000000066 | QF-1 | 5-Day: volume cap W2-W4 <= W1              | Todo PASS       |
-- | T-406c | 570000000066 | QF-1 | 5-Day: deload W4 sets = 2                  | Sets = 2        |
-- | T-406d | 570000000066 | QF-4 | 5-Day: cardio params (reps seg, tempo --)  | Visual check    |
-- | T-406e | 570000000066 | QF-4 | 5-Day: cardio no tiene params isolation    | 0 filas = PASS  |
-- | T-406f | 570000000066 | ALL  | 5-Day: tiene 5 dias distintos              | 5 filas = PASS  |
-- | T-406g | 570000000066 | ALL  | 5-Day: role order (compound>core>iso>card) | Visual check    |
-- | T-406h | 570000000066 | ALL  | 5-Day: resumen general por semana          | Visual check    |
--
-- Phones incluidos en teardown:
--   QF: 570000000061, 570000000062, 570000000063, 570000000064, 570000000065, 570000000066
-- ============================================
