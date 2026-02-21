-- ============================================
-- MIGRACIÓN: Corregir equipment de ejercicios que requieren barra/anillas/paralelas
-- Fecha: 2026-02-19
-- Ticket: KAN-101
-- ============================================
--
-- PROBLEMA: 22 ejercicios marcados como equipment = 'bodyweight' realmente
-- necesitan barra de dominadas, anillas, o barras paralelas.
-- Esto causa que usuarios HOME con solo peso corporal reciban ejercicios
-- que no pueden realizar (ej: dominadas, remos invertidos, fondos en paralelas).
--
-- CASO REAL: Vanessa Aldana (bodyweight only) recibió Remo Invertido,
-- Remo en anillas, y Dominadas asistidas en su rutina generada.
--
-- FRAMEWORK DE CLASIFICACIÓN (Test de Remoción):
-- Si remueves la barra/anillas y el ejercicio NO se puede hacer → pull_bar
-- Si se puede hacer sin nada (ej: flexiones en el suelo) → bodyweight
--
-- NUEVO EQUIPMENT TYPE: 'pull_bar'
-- Ya soportado por ProcessUserPreferences en WORKOUT_CREATOR
-- (mapea "barra de dominadas/pull up bar" → pull_bar)
--
-- IMPACTO: Usuarios HOME bodyweight-only ya NO recibirán estos ejercicios.
-- Usuarios HOME con pull_bar SI los recibirán.
--
-- ============================================

-- Pre-verificación: Estado actual
-- SELECT exercise_id, spanish_name, equipment
-- FROM exercises
-- WHERE equipment = 'bodyweight'
--   AND (spanish_name ILIKE '%dominada%' OR spanish_name ILIKE '%anillas%'
--        OR spanish_name ILIKE '%remo invertido%' OR spanish_name ILIKE '%paralelas%'
--        OR spanish_name ILIKE '%colgad%' OR spanish_name ILIKE '%suspensión%');
-- Esperado: 22 rows con equipment = 'bodyweight'

-- ============================================
-- ACTUALIZACIÓN PRINCIPAL
-- ============================================

-- PASO 1: Dominadas y chin-ups (requieren barra de dominadas)
UPDATE exercises
SET equipment = 'pull_bar'
WHERE exercise_id IN (
  'ex_pull_ups',
  'ex_chin_ups',
  'ex_bodyweight_assisted_chin_up',
  'ex_bodyweight_assisted_pull_up',
  'ex_bodyweight_assisted_mixed_grip_pullup',
  'ex_bodyweight_assisted_gironda_chin_up',
  'ex_gironda_chin_up',
  'ex_l_sit_pull_up',
  'ex_l_sit_chin_up'
);

-- PASO 2: Ejercicios en anillas (requieren anillas colgadas de barra/soporte)
UPDATE exercises
SET equipment = 'pull_bar'
WHERE exercise_id IN (
  'ex_ring_pull_up',
  'ex_ring_chin_up',
  'ex_ring_twisting_chin_up',
  'ex_ring_row',
  'ex_ring_skullcrusher',
  'ex_ring_standing_chest_fly'
);

-- PASO 3: Remos invertidos (requieren barra horizontal o superficie elevada)
UPDATE exercises
SET equipment = 'pull_bar'
WHERE exercise_id IN (
  'ex_inverted_row',
  'ex_bodyweight_underhand_inverted_row',
  'ex_bodyweight_overhand_inverted_row',
  'ex_bodyweight_standing_inverted_row'
);

-- PASO 4: Paralelas y colgado (requieren barra o paralelas)
UPDATE exercises
SET equipment = 'pull_bar'
WHERE exercise_id IN (
  'ex_parralel_bar_dips',
  'ex_dead_hang',
  'ex_bodyweight_hanging_knee_tuck'
);

-- ============================================
-- POST-VERIFICACIÓN
-- ============================================

-- Verificar que los 22 ejercicios fueron actualizados
SELECT 'FINAL' as status, equipment, COUNT(*) as total
FROM exercises
WHERE exercise_id IN (
  'ex_pull_ups', 'ex_chin_ups', 'ex_bodyweight_assisted_chin_up',
  'ex_bodyweight_assisted_pull_up', 'ex_bodyweight_assisted_mixed_grip_pullup',
  'ex_bodyweight_assisted_gironda_chin_up', 'ex_gironda_chin_up',
  'ex_l_sit_pull_up', 'ex_l_sit_chin_up',
  'ex_ring_pull_up', 'ex_ring_chin_up', 'ex_ring_twisting_chin_up',
  'ex_ring_row', 'ex_ring_skullcrusher', 'ex_ring_standing_chest_fly',
  'ex_inverted_row', 'ex_bodyweight_underhand_inverted_row',
  'ex_bodyweight_overhand_inverted_row', 'ex_bodyweight_standing_inverted_row',
  'ex_parralel_bar_dips', 'ex_dead_hang', 'ex_bodyweight_hanging_knee_tuck'
)
GROUP BY equipment;
-- Esperado: pull_bar: 22

-- Verificar que NO quedan bodyweight mal clasificados
SELECT exercise_id, spanish_name
FROM exercises
WHERE equipment = 'bodyweight'
  AND (spanish_name ILIKE '%dominada%' OR spanish_name ILIKE '%anillas%'
       OR spanish_name ILIKE '%remo invertido%' OR spanish_name ILIKE '%paralelas%'
       OR spanish_name ILIKE '%colgad%' OR spanish_name ILIKE '%suspensión%'
       OR spanish_name ILIKE '%ring%');
-- Esperado: 0 rows

-- Verificar total de pull_bar disponibles
SELECT COUNT(*) as total_pullbar_exercises
FROM exercises
WHERE equipment = 'pull_bar';
-- Esperado: 22
