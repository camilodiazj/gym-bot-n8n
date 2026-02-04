-- ============================================
-- MIGRACIÓN: Corregir equipment de ejercicios con banda
-- Fecha: 2026-02-03
-- Autor: kiro-coach + pixel-dev (validación determinista)
-- ============================================
--
-- PROBLEMA: 83 ejercicios con "banda/band" en nombre/ID tenían
-- equipment = 'machine' cuando deberían ser 'resistance_band'
--
-- FRAMEWORK DE CLASIFICACIÓN (Test de Remoción):
-- Si remueves la banda y el ejercicio deja de existir → resistance_band
-- Si remueves la banda y el ejercicio sigue siendo válido → Otro equipment
--
-- REGLAS DETERMINISTAS:
-- R1: exercise_id LIKE 'ex_band_%' → resistance_band
-- R2: exercise_id LIKE '%_resisted%' → resistance_band
-- R3: exercise_id LIKE '%_band_%' → resistance_band
-- R4: exercise_id LIKE 'ex_barbell_banded%' → MANTENER barbell (híbrido)
-- R5: spanish_name ILIKE '%con banda%' → resistance_band (fallback)
--
-- IMPACTO: Usuarios HOME con bandas elásticas ahora reciben estos
-- ejercicios correctamente filtrados
--
-- ============================================

-- Pre-verificación: Estado actual (ejecutar primero para verificar)
-- SELECT equipment, COUNT(*) as total
-- FROM exercises
-- WHERE exercise_id LIKE '%band%' OR spanish_name ILIKE '%banda%'
-- GROUP BY equipment;

-- ============================================
-- ACTUALIZACIÓN PRINCIPAL
-- ============================================

-- PASO 1: Pure band exercises (ex_band_*)
-- Excluye híbridos barbell+band
UPDATE exercises
SET equipment = 'resistance_band'
WHERE exercise_id LIKE 'ex_band_%'
  AND exercise_id NOT LIKE 'ex_barbell_banded%';

-- PASO 2: Resisted exercises (*_resisted*)
UPDATE exercises
SET equipment = 'resistance_band'
WHERE exercise_id LIKE '%_resisted%';

-- PASO 3: Cardio band exercises (*_band_* no híbridos)
UPDATE exercises
SET equipment = 'resistance_band'
WHERE exercise_id LIKE '%_band_%'
  AND exercise_id NOT LIKE 'ex_barbell_banded%'
  AND equipment = 'machine';

-- PASO 4: Fallback por nombre español ("con banda")
UPDATE exercises
SET equipment = 'resistance_band'
WHERE spanish_name ILIKE '%con banda%'
  AND equipment = 'machine';

-- PASO 5: Corrección especial - ID incorrecto
-- ex_band_goblet_squat tiene nombre "Sentadilla con pesa tipo copa" (dumbbell)
UPDATE exercises
SET equipment = 'dumbbell'
WHERE exercise_id = 'ex_band_goblet_squat';

-- ============================================
-- HÍBRIDOS - NO MODIFICAR (ya correctos)
-- ============================================
-- Los siguientes ejercicios usan barbell + band (accommodating resistance)
-- El barbell es el equipo principal, la banda es accesorio
--
-- ex_barbell_banded_back_squat → barbell
-- ex_barbell_banded_deadlift → barbell
-- ex_barbell_banded_bench_press → barbell
-- ex_barbell_banded_hip_thrust → barbell
-- ex_barbell_banded_romanian_deadlift → barbell

-- ============================================
-- POST-VERIFICACIÓN
-- ============================================

-- Verificar distribución final
SELECT 'FINAL' as status, equipment, COUNT(*) as total
FROM exercises
WHERE exercise_id LIKE '%band%' OR spanish_name ILIKE '%banda%'
GROUP BY equipment
ORDER BY total DESC;

-- Resultado esperado:
-- resistance_band: 82
-- barbell: 5 (híbridos)
-- dumbbell: 1 (ex_band_goblet_squat corregido)

-- Verificar que no quedan bandas como machine incorrectamente
SELECT exercise_id, spanish_name, equipment
FROM exercises
WHERE (spanish_name ILIKE '%banda%' OR exercise_id LIKE '%band%')
  AND equipment = 'machine';
-- Esperado: 0 rows

-- Verificar total de resistance_band disponibles para HOME
SELECT COUNT(*) as total_band_exercises
FROM exercises
WHERE equipment = 'resistance_band';
-- Esperado: 82
