#!/usr/bin/env node
/**
 * Script de Validación de Ejercicios con Banda - Lectura desde BD
 *
 * Lee TODOS los ejercicios de la base de datos y valida cuáles
 * deberían tener equipment = 'resistance_band'
 *
 * Uso:
 *   node validate_band_exercises_from_db.js [--dry-run] [--apply]
 *
 * Requiere:
 *   - SUPABASE_URL y SUPABASE_SERVICE_KEY en .env o variables de entorno
 */

require('dotenv').config({ path: '../../.env' });

const { createClient } = require('@supabase/supabase-js');

// ============================================
// CONFIGURACIÓN
// ============================================

const SUPABASE_URL = process.env.SUPABASE_URL;
const SUPABASE_KEY = process.env.SUPABASE_SERVICE_KEY || process.env.SUPABASE_ANON_KEY;

if (!SUPABASE_URL || !SUPABASE_KEY) {
  console.error('❌ Error: SUPABASE_URL y SUPABASE_SERVICE_KEY requeridos');
  console.error('   Configura las variables en .env o en el entorno');
  process.exit(1);
}

const supabase = createClient(SUPABASE_URL, SUPABASE_KEY);

// ============================================
// REGLAS DE CLASIFICACIÓN DETERMINISTAS
// ============================================

/**
 * Keywords que indican ejercicio con banda elástica
 */
const BAND_KEYWORDS = {
  spanish: [
    'con banda',
    'banda elástica',
    'banda de resistencia',
    'mini-banda',
    'bandas de cardio',
    'bandas elásticas',
  ],
  english: [
    'band ',
    ' band',
    'resistance band',
    'resisted',
  ],
  id_patterns: [
    /^ex_band_/,           // IDs que empiezan con ex_band_
    /_band_/,              // IDs que contienen _band_
    /_resisted/,           // IDs que contienen _resisted
  ]
};

/**
 * Exclusiones: ejercicios que mencionan "banda" pero NO son de banda elástica
 */
const EXCLUSIONS = [
  { pattern: /banda de dominadas/i, reason: 'Banda de dominadas es bodyweight assist' },
  { pattern: /assisted.*band/i, reason: 'Assisted band es bodyweight' },
  { pattern: /pull.?up.*band/i, reason: 'Pull-up band assist es bodyweight' },
];

/**
 * Casos especiales que requieren mapeo manual
 */
const MANUAL_OVERRIDES = {
  // ID -> { equipment, reason }
  // Ejemplo: 'ex_some_special_case': { equipment: 'bodyweight', reason: 'Manual override' }
};

/**
 * Clasifica un ejercicio basándose en reglas deterministas
 */
function classifyExercise(exercise) {
  const { exercise_id, spanish_name, equipment } = exercise;
  const nameLower = (spanish_name || '').toLowerCase();
  const idLower = (exercise_id || '').toLowerCase();

  // 1. Check manual overrides primero
  if (MANUAL_OVERRIDES[exercise_id]) {
    return {
      shouldBe: MANUAL_OVERRIDES[exercise_id].equipment,
      reason: MANUAL_OVERRIDES[exercise_id].reason,
      confidence: 'MANUAL',
      matchedRule: 'MANUAL_OVERRIDE'
    };
  }

  // 2. Check exclusiones
  for (const excl of EXCLUSIONS) {
    if (excl.pattern.test(nameLower) || excl.pattern.test(idLower)) {
      return {
        shouldBe: equipment, // Mantener el actual
        reason: excl.reason,
        confidence: 'EXCLUDED',
        matchedRule: 'EXCLUSION'
      };
    }
  }

  // 3. Check ID patterns (más confiable)
  for (const pattern of BAND_KEYWORDS.id_patterns) {
    if (pattern.test(idLower)) {
      return {
        shouldBe: 'resistance_band',
        reason: `ID matches pattern: ${pattern}`,
        confidence: 'HIGH',
        matchedRule: 'ID_PATTERN'
      };
    }
  }

  // 4. Check keywords en español
  for (const keyword of BAND_KEYWORDS.spanish) {
    if (nameLower.includes(keyword.toLowerCase())) {
      return {
        shouldBe: 'resistance_band',
        reason: `Name contains: "${keyword}"`,
        confidence: 'HIGH',
        matchedRule: 'SPANISH_KEYWORD'
      };
    }
  }

  // 5. Check keywords en inglés
  for (const keyword of BAND_KEYWORDS.english) {
    if (nameLower.includes(keyword.toLowerCase())) {
      return {
        shouldBe: 'resistance_band',
        reason: `Name contains: "${keyword}"`,
        confidence: 'MEDIUM',
        matchedRule: 'ENGLISH_KEYWORD'
      };
    }
  }

  // No match - mantener equipment actual
  return {
    shouldBe: equipment,
    reason: 'No band indicators found',
    confidence: 'NONE',
    matchedRule: 'NO_MATCH'
  };
}

// ============================================
// FUNCIONES PRINCIPALES
// ============================================

/**
 * Obtiene todos los ejercicios de la BD
 */
async function fetchAllExercises() {
  console.log('📥 Cargando ejercicios de la base de datos...');

  const { data, error } = await supabase
    .from('exercises')
    .select('exercise_id, spanish_name, main_muscle, pattern, equipment, level')
    .order('exercise_id');

  if (error) {
    console.error('❌ Error al cargar ejercicios:', error.message);
    process.exit(1);
  }

  console.log(`   ✓ ${data.length} ejercicios cargados\n`);
  return data;
}

/**
 * Valida todos los ejercicios
 */
function validateExercises(exercises) {
  const results = {
    correct: [],           // Ya tienen el equipment correcto
    needsUpdate: [],       // Necesitan actualización
    excluded: [],          // Excluidos por reglas
    ambiguous: [],         // Requieren revisión manual
  };

  for (const exercise of exercises) {
    const classification = classifyExercise(exercise);

    const result = {
      ...exercise,
      classification
    };

    if (classification.confidence === 'EXCLUDED') {
      results.excluded.push(result);
    } else if (classification.shouldBe !== exercise.equipment) {
      if (classification.confidence === 'HIGH' || classification.confidence === 'MANUAL') {
        results.needsUpdate.push(result);
      } else if (classification.confidence === 'MEDIUM') {
        // Medium confidence también se actualiza pero se reporta
        results.needsUpdate.push(result);
      } else {
        results.ambiguous.push(result);
      }
    } else {
      results.correct.push(result);
    }
  }

  return results;
}

/**
 * Genera el SQL de migración
 */
function generateMigrationSQL(needsUpdate) {
  if (needsUpdate.length === 0) {
    return '-- No hay ejercicios que actualizar';
  }

  // Agrupar por equipment destino
  const byEquipment = {};
  for (const ex of needsUpdate) {
    const target = ex.classification.shouldBe;
    if (!byEquipment[target]) {
      byEquipment[target] = [];
    }
    byEquipment[target].push(ex);
  }

  let sql = `-- ============================================
-- MIGRACIÓN: Corregir equipment de ejercicios
-- Generado: ${new Date().toISOString()}
-- Total ejercicios a actualizar: ${needsUpdate.length}
-- ============================================

`;

  for (const [equipment, exercises] of Object.entries(byEquipment)) {
    const ids = exercises.map(ex => `'${ex.exercise_id}'`);

    sql += `-- Actualizar ${exercises.length} ejercicios a ${equipment}
UPDATE exercises
SET equipment = '${equipment}'
WHERE exercise_id IN (
`;

    // Dividir en chunks de 5 para legibilidad
    for (let i = 0; i < ids.length; i += 5) {
      const chunk = ids.slice(i, i + 5);
      const isLast = i + 5 >= ids.length;
      sql += `  ${chunk.join(', ')}${isLast ? '' : ','}\n`;
    }

    sql += `);\n\n`;
  }

  sql += `-- Verificación
SELECT equipment, COUNT(*) as total
FROM exercises
WHERE exercise_id IN (${needsUpdate.map(ex => `'${ex.exercise_id}'`).join(', ')})
GROUP BY equipment;
`;

  return sql;
}

/**
 * Aplica los cambios a la BD
 */
async function applyChanges(needsUpdate) {
  if (needsUpdate.length === 0) {
    console.log('✓ No hay cambios que aplicar');
    return;
  }

  console.log(`\n🔄 Aplicando ${needsUpdate.length} cambios a la base de datos...`);

  // Agrupar por equipment destino para hacer updates eficientes
  const byEquipment = {};
  for (const ex of needsUpdate) {
    const target = ex.classification.shouldBe;
    if (!byEquipment[target]) {
      byEquipment[target] = [];
    }
    byEquipment[target].push(ex.exercise_id);
  }

  for (const [equipment, ids] of Object.entries(byEquipment)) {
    const { error } = await supabase
      .from('exercises')
      .update({ equipment })
      .in('exercise_id', ids);

    if (error) {
      console.error(`❌ Error actualizando a ${equipment}:`, error.message);
    } else {
      console.log(`   ✓ ${ids.length} ejercicios actualizados a ${equipment}`);
    }
  }

  console.log('\n✅ Migración completada');
}

/**
 * Imprime el reporte de validación
 */
function printReport(results) {
  console.log('='.repeat(80));
  console.log('REPORTE DE VALIDACIÓN DE EJERCICIOS');
  console.log('='.repeat(80));
  console.log('');

  console.log(`📊 RESUMEN:`);
  console.log(`   ✅ Correctos:        ${results.correct.length}`);
  console.log(`   🔄 Necesitan update: ${results.needsUpdate.length}`);
  console.log(`   ⏭️  Excluidos:        ${results.excluded.length}`);
  console.log(`   ❓ Ambiguos:         ${results.ambiguous.length}`);
  console.log('');

  // Detalles de los que necesitan update
  if (results.needsUpdate.length > 0) {
    console.log('─'.repeat(80));
    console.log('🔄 EJERCICIOS QUE NECESITAN ACTUALIZACIÓN:');
    console.log('─'.repeat(80));

    for (const ex of results.needsUpdate) {
      console.log(`\n  📝 ${ex.exercise_id}`);
      console.log(`     Nombre: ${ex.spanish_name}`);
      console.log(`     Actual: ${ex.equipment} → Debería ser: ${ex.classification.shouldBe}`);
      console.log(`     Razón: ${ex.classification.reason}`);
      console.log(`     Confianza: ${ex.classification.confidence}`);
    }
  }

  // Ejercicios ambiguos (requieren revisión)
  if (results.ambiguous.length > 0) {
    console.log('');
    console.log('─'.repeat(80));
    console.log('❓ EJERCICIOS QUE REQUIEREN REVISIÓN MANUAL:');
    console.log('─'.repeat(80));

    for (const ex of results.ambiguous) {
      console.log(`\n  ⚠️  ${ex.exercise_id}`);
      console.log(`     Nombre: ${ex.spanish_name}`);
      console.log(`     Equipment actual: ${ex.equipment}`);
    }
  }

  console.log('');
}

// ============================================
// MAIN
// ============================================

async function main() {
  const args = process.argv.slice(2);
  const dryRun = args.includes('--dry-run') || !args.includes('--apply');
  const showSQL = args.includes('--sql') || dryRun;

  console.log('');
  console.log('🏋️  VALIDADOR DE EJERCICIOS CON BANDA');
  console.log('─'.repeat(40));
  console.log(`   Modo: ${dryRun ? 'DRY RUN (solo reporte)' : 'APLICAR CAMBIOS'}`);
  console.log('');

  // 1. Cargar ejercicios
  const exercises = await fetchAllExercises();

  // 2. Validar
  const results = validateExercises(exercises);

  // 3. Imprimir reporte
  printReport(results);

  // 4. Mostrar SQL si aplica
  if (showSQL && results.needsUpdate.length > 0) {
    console.log('─'.repeat(80));
    console.log('📄 SQL DE MIGRACIÓN:');
    console.log('─'.repeat(80));
    console.log('');
    console.log(generateMigrationSQL(results.needsUpdate));
  }

  // 5. Aplicar cambios si no es dry-run
  if (!dryRun) {
    await applyChanges(results.needsUpdate);
  } else if (results.needsUpdate.length > 0) {
    console.log('');
    console.log('💡 Para aplicar los cambios, ejecuta:');
    console.log('   node validate_band_exercises_from_db.js --apply');
  }

  return results;
}

// Ejecutar
main().catch(console.error);
