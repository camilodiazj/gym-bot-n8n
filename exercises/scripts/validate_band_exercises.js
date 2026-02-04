/**
 * Script de Validación de Ejercicios con Banda
 *
 * Objetivo: Identificar ejercicios mal clasificados como 'machine'
 * que en realidad usan bandas elásticas.
 *
 * Lógica determinista:
 * 1. Si exercise_id empieza con "ex_band_" → resistance_band
 * 2. Si exercise_id contiene "_band_" → resistance_band
 * 3. Si spanish_name contiene "banda" (sin "banda de dominadas") → resistance_band
 * 4. Si spanish_name empieza con "Band " → resistance_band
 *
 * Casos de exclusión:
 * - "banda de dominadas" → bodyweight (pull-up bar assist)
 * - Ejercicios que usan banda + otro equipo primario
 */

const exercises = [
  // Ejercicios identificados en la BD con equipment='machine' pero nombre con 'banda/band'
  { id: "ex_band_hip_abduction", name: "Abducción de cadera con banda" },
  { id: "ex_band_hip_adduction", name: "Aducción de Cadera con Banda" },
  { id: "ex_band_internally_rotated_chest_fly", name: "Apertura de Pecho con Banda y Rotación Interna" },
  { id: "ex_band_single_arm_pec_fly", name: "Aperturas con banda a un brazo" },
  { id: "ex_band_chest_fly", name: "Aperturas de Pecho con Banda" },
  { id: "ex_band_bench_fly", name: "Band Bench Fly" },
  { id: "ex_band_bench_press", name: "Band Bench Press" },
  { id: "ex_band_bent_over_row", name: "Band Bent Over Row" },
  { id: "ex_band_floor_press", name: "Band Floor Press" },
  { id: "ex_band_half_kneeling_single_arm_pulldown", name: "Band Half Kneeling Single Arm Pulldown" },
  { id: "ex_band_high_to_low_pallof_press_wood_chopper", name: "Band High To Low Pallof Press Wood Chopper" },
  { id: "ex_band_kneeling_pulldown", name: "Band Kneeling Pulldown" },
  { id: "ex_band_low_to_high_pallof_press_wood_chopper", name: "Band Low To High Pallof Press Wood Chopper" },
  { id: "ex_band_pallof_press_to_wood_chopper", name: "Band Pallof Press To Wood Chopper" },
  { id: "ex_band_press_around", name: "Band Press Around" },
  { id: "ex_band_pull_around", name: "Band Pull Around" },
  { id: "ex_band_seated_pulldown", name: "Band Seated Pulldown" },
  { id: "ex_band_seated_single_arm_pulldown", name: "Band Seated Single Arm Pulldown" },
  { id: "ex_band_side_bend", name: "Band Side Bend" },
  { id: "ex_band_single_arm_bent_over_row", name: "Band Single Arm Bent Over Row" },
  { id: "ex_band_single_arm_internally_rotated_chest_fly", name: "Band Single Arm Internally Rotated Chest Fly" },
  { id: "ex_band_single_arm_lateral_raise", name: "Band Single Arm Lateral Raise" },
  { id: "ex_band_standing_good_morning", name: "Band Standing Good Morning" },
  { id: "ex_band_single_arm_pulldown", name: "Banda de tracción de un solo brazo" },
  { id: "ex_cardio_band_reverse_fly_jacks", name: "Bandas de Cardio con Vuelos Inversos y Saltos Jack" },
  { id: "ex_cardio_band_seal_jacks", name: "Bandas elásticas de cardio para saltos Jack Seal" },
  { id: "ex_band_good_morning", name: "Buenos Días Con Banda" },
  { id: "ex_side_walks_resisted", name: "Caminatas Laterales con Banda" },
  { id: "ex_band_crunch", name: "Crunch con banda" },
  { id: "ex_band_curl", name: "Curl con banda" },
  { id: "ex_band_high_curl", name: "Curl de bíceps con banda (anclada en alto)" },
  { id: "ex_band_bayesian_curl", name: "Curl de bíceps con banda Bayesian" },
  { id: "ex_band_leg_curl", name: "Curl Femoral Con Banda Elástica" },
  { id: "ex_band_high_reverse_curl", name: "Curl Inverso con Banda Alta" },
  { id: "ex_band_reverse_curl", name: "Curl Invertido con Banda" },
  { id: "ex_band_bayesian_reverse_curl", name: "Curl Invertido con Banda Bayesian" },
  { id: "ex_band_high_hammer_curl", name: "Curl martillo con banda anclada en alto" },
  { id: "ex_band_bayesian_hammer_curl", name: "Curl martillo con banda Bayesian" },
  { id: "ex_band_calf_raise", name: "Elevación de gemelos con banda" },
  { id: "ex_band_shoulder_y_raise", name: "Elevación en Y con banda (hombros)" },
  { id: "ex_band_front_raise", name: "Elevación Frontal con Banda Elástica" },
  { id: "ex_band_low_lateral_raise", name: "Elevación Lateral Baja con Banda" },
  { id: "ex_band_mid_lateral_raise", name: "Elevación Lateral Media con Banda" },
  { id: "ex_band_lateral_raise", name: "Elevaciones Laterales Con Banda Elástica" },
  { id: "ex_hip_thrust_standing_resisted", name: "Empuje de Cadera con Banda" },
  { id: "ex_band_horizontal_shrug", name: "Encogimiento horizontal de hombros con banda" },
  { id: "ex_quad_stretch_floor_prone", name: "Estiramiento de Cuádriceps en el Suelo con Banda" },
  { id: "ex_hamstring_stretch_supine_dynamic", name: "Estiramiento dinámico de isquiotibiales en supino con banda" },
  { id: "ex_band_leg_extension", name: "Extensión de piernas con banda" },
  { id: "ex_knee_extension_closed_chain_standing_resisted", name: "Extensión de Rodilla con Banda" },
  { id: "ex_band_overhead_tricep_extension", name: "Extensión de tríceps por encima de la cabeza con banda elástica" },
  { id: "ex_hip_flexions_standing_resisted", name: "Flexión de Cadera en Mini-Banda de Pie" },
  { id: "ex_band_pushup", name: "Flexiones con banda de resistencia" },
  { id: "ex_hip_flexions_straight_leg_standing_resisted", name: "Flexiones de Cadera con Pierna Recta y Banda Elástica" },
  { id: "ex_band_seated_inversions", name: "Inversiones sentado con banda" },
  { id: "ex_band_high_face_pull", name: "Jalón Facial Alto con Banda" },
  { id: "ex_band_face_pull", name: "Jalón Facial Con Banda" },
  { id: "ex_band_kneeling_single_arm_pulldown", name: "Jalón unilateral con banda (arrodillado)" },
  { id: "ex_band_rear_delt_fly", name: "Pájaros con banda para deltoides posteriores" },
  { id: "ex_band_glute_kickback", name: "Patada de glúteos con banda elástica" },
  { id: "ex_band_romanian_deadlift", name: "Peso muerto rumano con banda" },
  { id: "ex_band_standing_chest_press", name: "Prensa de pecho de pie con banda" },
  { id: "ex_band_chest_press", name: "Press de pecho con banda elástica" },
  { id: "ex_band_single_arm_chest_press", name: "Press de pecho unilateral con banda" },
  { id: "ex_band_leg_press", name: "Press de piernas con banda" },
  { id: "ex_band_overhead_press", name: "Press militar con banda" },
  { id: "ex_band_single_arm_overhead_press", name: "Press Militar con Banda Elástica a Un Brazo" },
  { id: "ex_band_pallof_press", name: "Press Pallof con banda" },
  { id: "ex_band_pullthrough", name: "Pull-through con banda elástica" },
  { id: "ex_band_pullover", name: "Pullover con banda" },
  { id: "ex_band_single_arm_row", name: "Remo con banda con un brazo" },
  { id: "ex_band_row", name: "Remo con banda elástica" },
  { id: "ex_band_upright_row", name: "Remo vertical con banda" },
  { id: "ex_band_wood_chopper", name: "Rotación de tronco con banda de resistencia" },
  { id: "ex_cardio_band_press_jacks", name: "Saltos de Prensa con Banda de Cardio" },
  { id: "ex_cardio_band_hammer_curl_jacks", name: "Saltos de tijera con banda de resistencia y curl martillo" },
  { id: "ex_band_squat", name: "Sentadilla con banda" },
  { id: "ex_band_squat_hold_single_arm_row", name: "Sentadilla con banda mantenida y remo con un solo brazo" },
  { id: "ex_band_squat_hold_row", name: "Sentadilla con Banda y Remo en Isometría" },
  { id: "ex_band_spanish_squat", name: "Sentadilla española con banda" },
  { id: "ex_band_pull_apart", name: "Separación con banda elástica" },
  { id: "ex_hip_extension_standing_resisted_isometric", name: "Sostén de Patada Glútea con Banda" },
];

/**
 * Reglas de clasificación determinista
 */
function classifyEquipment(exercise) {
  const { id, name } = exercise;
  const nameLower = name.toLowerCase();

  // Exclusiones: ejercicios que usan banda como asistencia pero equipo principal es otro
  const exclusions = [
    'banda de dominadas',  // → bodyweight
    'assisted band',       // → bodyweight
  ];

  for (const excl of exclusions) {
    if (nameLower.includes(excl)) {
      return { equipment: 'bodyweight', reason: `Exclusión: "${excl}" - banda como asistencia` };
    }
  }

  // Regla 1: ID empieza con "ex_band_"
  if (id.startsWith('ex_band_')) {
    return { equipment: 'resistance_band', reason: 'ID starts with ex_band_' };
  }

  // Regla 2: ID contiene "_band_" (ej: ex_cardio_band_...)
  if (id.includes('_band_')) {
    return { equipment: 'resistance_band', reason: 'ID contains _band_' };
  }

  // Regla 3: ID contiene "_resisted" (ej: ex_hip_thrust_standing_resisted)
  if (id.includes('_resisted')) {
    return { equipment: 'resistance_band', reason: 'ID contains _resisted (band resistance)' };
  }

  // Regla 4: Nombre contiene "banda" (español)
  if (nameLower.includes('banda')) {
    return { equipment: 'resistance_band', reason: 'Name contains "banda"' };
  }

  // Regla 5: Nombre empieza con "Band " (inglés)
  if (nameLower.startsWith('band ')) {
    return { equipment: 'resistance_band', reason: 'Name starts with "Band "' };
  }

  // No match - requiere revisión manual
  return { equipment: 'REVIEW', reason: 'No deterministic rule matched' };
}

/**
 * Ejecutar validación
 */
function validateAll() {
  console.log('='.repeat(80));
  console.log('VALIDACIÓN DE EJERCICIOS CON BANDA');
  console.log('='.repeat(80));
  console.log('');

  const results = {
    resistance_band: [],
    bodyweight: [],
    review: []
  };

  for (const exercise of exercises) {
    const { equipment, reason } = classifyEquipment(exercise);
    const result = { ...exercise, newEquipment: equipment, reason };

    if (equipment === 'resistance_band') {
      results.resistance_band.push(result);
    } else if (equipment === 'bodyweight') {
      results.bodyweight.push(result);
    } else {
      results.review.push(result);
    }
  }

  // Reporte
  console.log(`✅ RESISTANCE_BAND: ${results.resistance_band.length} ejercicios`);
  console.log(`⚠️  BODYWEIGHT: ${results.bodyweight.length} ejercicios`);
  console.log(`❓ REQUIEREN REVISIÓN: ${results.review.length} ejercicios`);
  console.log('');

  // Mostrar los que requieren revisión
  if (results.review.length > 0) {
    console.log('--- EJERCICIOS QUE REQUIEREN REVISIÓN MANUAL ---');
    for (const ex of results.review) {
      console.log(`  - ${ex.id}: "${ex.name}"`);
    }
    console.log('');
  }

  // Generar SQL
  console.log('='.repeat(80));
  console.log('SQL DE MIGRACIÓN GENERADO');
  console.log('='.repeat(80));
  console.log('');

  const bandIds = results.resistance_band.map(ex => `'${ex.id}'`);

  console.log(`-- Actualizar ${bandIds.length} ejercicios a resistance_band`);
  console.log(`UPDATE exercises`);
  console.log(`SET equipment = 'resistance_band'`);
  console.log(`WHERE exercise_id IN (`);

  // Dividir en chunks de 5 para legibilidad
  for (let i = 0; i < bandIds.length; i += 5) {
    const chunk = bandIds.slice(i, i + 5);
    const isLast = i + 5 >= bandIds.length;
    console.log(`  ${chunk.join(', ')}${isLast ? '' : ','}`);
  }
  console.log(`);`);
  console.log('');

  // Verificación
  console.log('-- Verificación post-update');
  console.log(`SELECT equipment, COUNT(*) as total`);
  console.log(`FROM exercises`);
  console.log(`WHERE spanish_name ILIKE '%banda%' OR spanish_name ILIKE '%band %'`);
  console.log(`GROUP BY equipment;`);

  return results;
}

// Ejecutar
const results = validateAll();

// Exportar para uso externo
module.exports = { validateAll, classifyEquipment, exercises };
