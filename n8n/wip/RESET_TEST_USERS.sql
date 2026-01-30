-- ================================================
-- RESET TEST USERS - Renovación de Mesociclo
-- ================================================
--
-- Ejecutar este script para resetear los usuarios de prueba
-- a su estado inicial y poder volver a ejecutar los tests.
--
-- Usuarios afectados:
-- - Test_MesocycleRenewal (570000000099)
-- - Test_Mantener (570000000100)
-- - Test_CambiarDias (570000000101)
-- - Test_RotarEjercicios (570000000102)
-- - Test_IntencionManual (570000000103)
--
-- ================================================

-- 1. Limpiar memoria del agente de renovación
DELETE FROM n8n_chat_histories
WHERE session_id IN (
  '43109d04-ac72-42c6-8f5e-188047cef604_mesocycle_renewal',
  'a298b12a-54e8-4a06-b20a-4a6d8b80aca1_mesocycle_renewal',
  'a705cf9c-4745-47f4-aa15-f24697b0cec7_mesocycle_renewal',
  'ec0abf34-9ef8-43bf-83f0-9adb63886117_mesocycle_renewal',
  '6b027107-4cf4-49f2-8a93-98bbf71dee36_mesocycle_renewal'
);

-- 2. Restaurar mesocycle_number a 1 para todos
UPDATE users_plans
SET mesocycle_number = 1,
    last_renewal_date = NOW() - INTERVAL '4 weeks'
WHERE user_id IN (
  '43109d04-ac72-42c6-8f5e-188047cef604',
  'a298b12a-54e8-4a06-b20a-4a6d8b80aca1',
  'a705cf9c-4745-47f4-aa15-f24697b0cec7',
  'ec0abf34-9ef8-43bf-83f0-9adb63886117',
  '6b027107-4cf4-49f2-8a93-98bbf71dee36'
);

-- 3. Limpiar schedules actuales
DELETE FROM user_weekly_schedule
WHERE user_id IN (
  '43109d04-ac72-42c6-8f5e-188047cef604',
  'a298b12a-54e8-4a06-b20a-4a6d8b80aca1',
  'a705cf9c-4745-47f4-aa15-f24697b0cec7',
  'ec0abf34-9ef8-43bf-83f0-9adb63886117',
  '6b027107-4cf4-49f2-8a93-98bbf71dee36'
);

-- 4. Recrear schedules semana 4 completada (Test_Mantener, Test_CambiarDias, Test_RotarEjercicios)
INSERT INTO user_weekly_schedule (day_routine_id, user_id, week, week_day, session_name, planned_day, "Completed")
SELECT
  gen_random_uuid(),
  u.user_id,
  s.week,
  s.week_day_enum,
  s.session_name,
  s.planned_day,
  s.completed
FROM users u
CROSS JOIN LATERAL (
  VALUES
    (4, 1, 'Lunes'::week_days, 'Upper A', ((NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '7 days')::timestamp, true),
    (4, 3, 'Miercoles'::week_days, 'Lower A', ((NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '5 days')::timestamp, true),
    (4, 4, 'Jueves'::week_days, 'Upper B', ((NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '4 days')::timestamp, true),
    (4, 6, 'Sabado'::week_days, 'Lower B', ((NOW() AT TIME ZONE 'America/Bogota')::date - INTERVAL '2 days')::timestamp, true)
) AS s(week, week_day, week_day_enum, session_name, planned_day, completed)
WHERE u.user_id IN ('a298b12a-54e8-4a06-b20a-4a6d8b80aca1', 'a705cf9c-4745-47f4-aa15-f24697b0cec7', 'ec0abf34-9ef8-43bf-83f0-9adb63886117');

-- 5. Recrear schedule semana 1 activo (Test_IntencionManual)
INSERT INTO user_weekly_schedule (day_routine_id, user_id, week, week_day, session_name, planned_day, "Completed")
SELECT
  gen_random_uuid(),
  u.user_id,
  s.week,
  s.week_day_enum,
  s.session_name,
  s.planned_day,
  s.completed
FROM users u
CROSS JOIN LATERAL (
  VALUES
    (1, 1, 'Lunes'::week_days, 'Upper A', ((NOW() AT TIME ZONE 'America/Bogota')::date + INTERVAL '0 days')::timestamp, false),
    (1, 3, 'Miercoles'::week_days, 'Lower A', ((NOW() AT TIME ZONE 'America/Bogota')::date + INTERVAL '2 days')::timestamp, false),
    (1, 4, 'Jueves'::week_days, 'Upper B', ((NOW() AT TIME ZONE 'America/Bogota')::date + INTERVAL '3 days')::timestamp, false),
    (1, 6, 'Sabado'::week_days, 'Lower B', ((NOW() AT TIME ZONE 'America/Bogota')::date + INTERVAL '5 days')::timestamp, false)
) AS s(week, week_day, week_day_enum, session_name, planned_day, completed)
WHERE u.user_id = '6b027107-4cf4-49f2-8a93-98bbf71dee36';

-- 6. Verificar estado final
SELECT
  u.full_name,
  up.mesocycle_number,
  COUNT(DISTINCT uws.day_routine_id) as schedule_count,
  COUNT(DISTINCT CASE WHEN uws."Completed" = true THEN uws.day_routine_id END) as completed_count,
  MAX(uws.week) as max_week
FROM users u
JOIN users_plans up ON u.user_id = up.user_id
LEFT JOIN user_weekly_schedule uws ON u.user_id = uws.user_id
WHERE u.user_id IN (
  '43109d04-ac72-42c6-8f5e-188047cef604',
  'a298b12a-54e8-4a06-b20a-4a6d8b80aca1',
  'a705cf9c-4745-47f4-aa15-f24697b0cec7',
  'ec0abf34-9ef8-43bf-83f0-9adb63886117',
  '6b027107-4cf4-49f2-8a93-98bbf71dee36'
)
GROUP BY u.full_name, up.mesocycle_number
ORDER BY u.full_name;

-- ================================================
-- RESULTADO ESPERADO:
-- ================================================
-- full_name               | mesocycle_number | schedule_count | completed_count | max_week
-- ----------------------- | ---------------- | -------------- | --------------- | --------
-- Test_MesocycleRenewal   | 1                | 0              | 0               | null
-- Test_Mantener           | 1                | 4              | 4               | 4
-- Test_CambiarDias        | 1                | 4              | 4               | 4
-- Test_RotarEjercicios    | 1                | 4              | 4               | 4
-- Test_IntencionManual    | 1                | 4              | 0               | 1
