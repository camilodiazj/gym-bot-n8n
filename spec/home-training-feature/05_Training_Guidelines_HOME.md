# Training Guidelines HOME - Reglas de Entrenamiento para Casa

**Documento:** 05_Training_Guidelines_HOME.md
**Version:** 1.0
**Status:** Draft
**Creado:** 2026-02-03
**Asignado a:** kiro-coach
**Revisado por:** code-reviewer

---

## 1. Objetivo

Este documento define las **reglas de entrenamiento cientificamente validadas** para rutinas en casa dentro de GymBot. El proposito es:

1. Establecer criterios de seleccion de ejercicios basados en equipamiento disponible
2. Definir estrategias de compensacion para patrones de movimiento con baja cobertura
3. Validar que las rutinas HOME cumplan estandares de efectividad comparables a GYM
4. Proveer guias de progresion para diferentes niveles de experiencia

**Fundamento Cientifico:**

Las investigaciones demuestran que el entrenamiento en casa puede producir ganancias de fuerza e hipertrofia comparables al gimnasio cuando se cumplen los principios fundamentales:

- Sobrecarga progresiva (Schoenfeld, 2010)
- Volumen semanal adecuado (Schoenfeld et al., 2017)
- Proximidad al fallo muscular (Refalo et al., 2022)
- Frecuencia de entrenamiento 2x/semana por grupo muscular (Schoenfeld et al., 2016)

---

## 2. Analisis de Equipamiento

### 2.1 Equipamiento Minimo Recomendado

| Equipamiento | Ejercicios Disponibles | Cobertura de Patrones |
|--------------|------------------------|----------------------|
| **Mancuernas ajustables** | 290 ejercicios | 9/10 patrones |
| **Sin equipamiento (bodyweight)** | 197 ejercicios | 8/10 patrones |

**Por que mancuernas es el minimo viable:**

1. **Versatilidad**: Cubren todos los patrones de movimiento excepto pull_v sin barra
2. **Progresion de carga**: Permiten incrementos graduales de peso
3. **Costo-beneficio**: Una inversion unica con alta durabilidad
4. **Espacio**: Requieren minimo espacio de almacenamiento

**Recomendacion GymBot:**
- Usuarios sin mancuernas: Rutinas bodyweight con progresiones
- Usuarios con mancuernas: Rutinas completas con sobrecarga progresiva

### 2.2 Equipamiento Optimo

| Equipamiento | Beneficio Principal | Patrones Mejorados |
|--------------|--------------------|--------------------|
| Mancuernas ajustables | Base de sobrecarga | Todos |
| Kettlebell | Trabajo unilateral, hinge, explosividad | hinge, lunge, squat |
| Barra de traccion (pull-up bar) | Pull vertical | pull_v |
| Bandas elasticas | Resistencia variable, pre-activacion | pull_v, arm, accessory |

**Ejercicios clave habilitados por cada pieza:**

**Kettlebell (157 ejercicios adicionales):**
- Kettlebell Swing (hinge explosivo)
- Goblet Squat (squat con core activation)
- Turkish Get-Up (full body)
- Single Arm RDL (hinge unilateral)

**Barra de traccion (12 ejercicios pull_v bodyweight):**
- Dominadas (pull_v compound principal)
- Chin-ups (biceps + espalda)
- Dominadas en L (core + pull)
- Dominadas Gironda (enfasis en espalda)

**Bandas elasticas:**
- Face pulls (rear delts, sin alternativa en casa)
- Lat pulldowns asistidos
- Curls con resistencia variable
- Pull-aparts (salud de hombros)

### 2.3 Solo Peso Corporal - Viabilidad y Limitaciones

**Es viable?** Si, con limitaciones importantes.

| Aspecto | Viabilidad | Limitacion |
|---------|------------|------------|
| Principiantes | Alta | Suficiente estimulo inicial |
| Intermedios | Media | Progresion de carga dificil |
| Avanzados | Baja | Insuficiente para hipertrofia optima |

**Principales limitaciones:**

1. **Progresion de carga**: Solo via leverage, tempo, o volumen (no ideal)
2. **Pull vertical**: Requiere barra de traccion (sin alternativa real)
3. **Hinge/Hamstrings**: Nordic curls requieren anclaje; pocas opciones
4. **Sobrecarga inferior**: Bodyweight squats insuficientes para piernas entrenadas

**Progresiones recomendadas (bodyweight only):**

| Ejercicio Base | Progresion 1 | Progresion 2 | Progresion 3 |
|----------------|--------------|--------------|--------------|
| Push-up regular | Push-up diamante | Archer push-up | One-arm push-up (asistido) |
| Squat | Split squat | Bulgarian (elevado) | Pistol squat (asistido) |
| Lunge | Reverse lunge | Walking lunge | Deficit lunge |
| Plank | Side plank | Plank con extension | Ab wheel (si disponible) |

**Recomendacion GymBot para bodyweight:**
- Enfasis en tempo lento (3-1-3-0) para aumentar tiempo bajo tension
- Mayor volumen total (mas sets) para compensar falta de carga
- Frecuencia 3x/semana por grupo muscular para principiantes

---

## 3. Compensacion de Gaps por Patron

### Resumen de Cobertura HOME

| Patron | Total GYM | Disponible HOME | % Cobertura | Criticidad |
|--------|-----------|-----------------|-------------|------------|
| lunge | 106 | 85 | 80.2% | Baja |
| push_v | 28 | 19 | 67.9% | Media |
| hinge | 69 | 46 | 66.7% | Media |
| squat | 151 | 95 | 62.9% | Media |
| push_h | 159 | 85 | 53.5% | Media |
| arm | 240 | 123 | 51.3% | Baja |
| core | 97 | 48 | 49.5% | Baja |
| pull_h | 104 | 48 | 46.2% | Alta |
| pull_v | 45 | 21 | 46.7% | **Critica** |
| accessory | 658 | 270 | 41.0% | Baja |

### 3.1 Pull Vertical (46.7% cobertura - CRITICO)

**Problema:**
- Solo 21 ejercicios disponibles para HOME
- 12 requieren barra de traccion (bodyweight pull-ups)
- Sin barra: solo 9 ejercicios (pullovers con mancuerna/kettlebell)
- Pullovers trabajan principalmente pecho/serratus, no dorsales como compound

**Ejercicios HOME disponibles:**

| Equipamiento | Ejercicios | Efectividad |
|--------------|------------|-------------|
| Pull-up bar | Dominadas, Chin-ups, L-sits, Gironda | Alta |
| Dumbbell | Pullover con mancuerna, Single arm pullover | Media |
| Kettlebell | Pullover con pesa rusa | Media |
| Barbell | Barbell pullover | Media |

**Solucion cientifica - Ratio de compensacion:**

Cuando **NO hay barra de traccion disponible**:

```
-4 sets pull_v → +2 sets pull_h + 2 sets arm (biceps)
```

**Justificacion:**
- El pull_h (remos) trabaja dorsales en plano horizontal
- El trabajo adicional de biceps compensa la falta de chin-ups
- Los pullovers mantienen ROM de hombro similar al pull_v

**Ejercicios sustitutos especificos:**

| Originalmente | Sustituir por | Razon |
|---------------|---------------|-------|
| Lat pulldown (4 sets) | Remo con mancuerna (2) + Curl biceps (2) | Similar activacion muscular combinada |
| Pull-ups | Remo invertido (bodyweight) + Pullover | Patron de tirion + extension de hombro |

**Regla GymBot:**
- SI `home_equipment` incluye `pull_up_bar` → Usar dominadas como ejercicio principal pull_v
- SI NO → Aplicar ratio de compensacion y priorizar remos

### 3.2 Quads (26.3% cobertura)

**Problema:**
- Solo 15 ejercicios de quads disponibles para HOME (de 57 totales)
- Sin leg press, hack squat, leg extension machine
- Dependencia de variantes de squat con carga limitada

**Ejercicios HOME disponibles para Quads:**

| Ejercicio | Equipamiento | Efectividad Quads |
|-----------|--------------|-------------------|
| Bulgarian Split Squat | Dumbbell/Bodyweight | **Alta** |
| Sissy Squat | Bodyweight | Alta |
| Goblet Squat | Dumbbell/Kettlebell | Media-Alta |
| Step-ups | Dumbbell/Bodyweight | Media |
| Wall Sit | Bodyweight | Media (isometrico) |
| Lunge variations | Dumbbell/Bodyweight | Media |

**Solucion - Volumen y variantes:**

1. **Aumentar frecuencia**: 3x/semana para quads (vs 2x en gym)
2. **Enfasis en unilateral**: Bulgarian split squat como ejercicio principal
3. **Sissy squat para aislamiento**: Reemplazo de leg extension
4. **Tempo lento**: 4-0-2-0 para aumentar tension mecanica

**Ejercicios clave obligatorios para HOME:**

```sql
-- Quads HOME essentials
- Bulgarian split squat (dumbbell) -- OBLIGATORIO
- Sissy squat (bodyweight/assisted) -- OBLIGATORIO para aislamiento
- Step-ups (dumbbell)
- Goblet squat (kettlebell/dumbbell)
```

**Regla GymBot:**
- Minimo 2 ejercicios de quads por sesion de pierna
- Bulgarian split squat debe aparecer en al menos 1 sesion/semana

### 3.3 Hamstrings (25% cobertura)

**Problema:**
- Solo 12 ejercicios de hamstrings disponibles para HOME (de 48 totales)
- Sin leg curl machine (aislamiento principal)
- Dependencia de ejercicios de hinge (trabajan glutes tambien)

**Ejercicios HOME disponibles para Hamstrings:**

| Ejercicio | Equipamiento | Efectividad Hamstrings |
|-----------|--------------|------------------------|
| Nordic curl | Bodyweight | **Muy Alta** |
| RDL (Romanian Deadlift) | Dumbbell/Barbell | Alta |
| Slider hamstring curl | Bodyweight | Alta |
| Stiff-leg deadlift | Dumbbell | Media-Alta |
| Good mornings | Dumbbell | Media |
| Glute bridge single leg | Bodyweight | Media |

**Solucion - Progresiones para principiantes:**

**Nordic Curl Progression (Gold Standard para HOME):**

| Nivel | Variante | Descripcion |
|-------|----------|-------------|
| 1 | Eccentric only | Solo bajar controlado, usar brazos para subir |
| 2 | Assisted | Usar banda elastica para asistencia |
| 3 | Partial ROM | Rango parcial sin asistencia |
| 4 | Full ROM | Movimiento completo |

**RDL con mancuernas como base:**
- Single-leg RDL para deficit unilateral
- Staggered stance RDL para transicion
- B-stance RDL para enfasis unilateral

**Regla GymBot:**
- Incluir al menos 1 ejercicio de hip hinge + 1 de curl (o nordic) por sesion
- Para principiantes: Priorizar RDL bilateral antes de unilateral

### 3.4 Calfs (23.6% cobertura)

**Problema:**
- Solo 21 ejercicios de pantorrillas disponibles para HOME (de 89 totales)
- Sin maquina de pantorrillas sentado/de pie
- Carga limitada con mancuernas

**Ejercicios HOME disponibles para Calfs:**

| Ejercicio | Equipamiento | Notas |
|-----------|--------------|-------|
| Standing calf raise | Dumbbell/Kettlebell | Base con carga |
| Single-leg calf raise | Bodyweight/Dumbbell | Alta intensidad |
| Seated calf raise | Dumbbell | Soleo enfasis |
| Walking calf raise | Bodyweight | Volumen |
| Donkey calf raise | Bodyweight | Requiere setup |

**Solucion - Protocolo de alto volumen:**

Los calfs tienen alta resistencia a la fatiga y requieren:

| Parametro | Recomendacion HOME |
|-----------|-------------------|
| Sets/semana | 12-16 (vs 8-12 gym) |
| Reps | 15-25 por set |
| Tempo | 2-1-2-1 (pausa arriba y abajo) |
| Frecuencia | 3-4x/semana |

**Progresion de carga sin maquina:**
1. Usar escalon/step para ROM completo
2. Single-leg para duplicar carga relativa
3. Mancuerna pesada en una mano + pared para balance
4. Tempo muy lento (5 segundos excentrico)

**Regla GymBot:**
- Minimo 4 sets de calfs por sesion de pierna
- Priorizar single-leg versions para maximizar tension

### 3.5 Back General (25.9% cobertura)

**Problema:**
- Solo 14 ejercicios de espalda general disponibles para HOME (de 54 totales)
- Sin cables, lat pulldown, seated row machines
- Dependencia de remos con mancuerna

**Ejercicios HOME disponibles para Back:**

| Ejercicio | Equipamiento | Enfasis |
|-----------|--------------|---------|
| Dumbbell row (unilateral) | Dumbbell | Lats, rhomboids |
| Bent-over row | Barbell/Dumbbell | Espalda general |
| Chest-supported row | Dumbbell | Aislamiento sin lower back |
| Meadows row | Barbell (landmine) | Lats unilateral |
| Pullover | Dumbbell/Kettlebell | Lats, serratus |
| Renegade row | Dumbbell | Core + back |

**Solucion - Variedad de angulos:**

Para compensar falta de cables/maquinas:

1. **Remos inclinados** (chest-supported): Elimina momentum
2. **Remos unilaterales**: Mayor ROM y conexion mente-musculo
3. **Diferentes alturas de codo**: Codos pegados (lats) vs abiertos (rhomboids)
4. **Pullovers**: Unico ejercicio de extension de hombro disponible

**Ratio de compensacion para back:**

```
Rutina GYM tipica:
- Lat pulldown: 3 sets
- Cable row: 3 sets
- Face pull: 2 sets

Rutina HOME equivalente:
- Dumbbell row unilateral: 4 sets (2 por lado)
- Bent-over row bilateral: 3 sets
- Pullover: 2 sets
- Band pull-apart: 2 sets (si hay bandas)
```

**Regla GymBot:**
- Minimo 2 variantes de remo por sesion de espalda
- Incluir al menos 1 ejercicio unilateral

---

## 4. Volumen Semanal Recomendado por Grupo Muscular

### 4.1 Volumen GYM (Referencia)

Basado en meta-analisis de Schoenfeld et al. (2017):

| Grupo Muscular | Sets/Semana | Frecuencia | Rango Optimo |
|----------------|-------------|------------|--------------|
| Chest | 10-20 | 2x | 12-16 |
| Back | 10-20 | 2x | 14-18 |
| Shoulders | 8-16 | 2x | 10-14 |
| Biceps | 6-12 | 2x | 8-10 |
| Triceps | 6-12 | 2x | 8-10 |
| Quads | 10-20 | 2x | 12-16 |
| Hamstrings | 8-16 | 2x | 10-14 |
| Glutes | 8-16 | 2x | 10-14 |
| Calfs | 8-16 | 2-3x | 10-14 |
| Abs/Core | 6-12 | 2-3x | 8-10 |

### 4.2 Volumen HOME (Ajustado)

| Grupo Muscular | Sets/Semana | Frecuencia | Notas de Ajuste |
|----------------|-------------|------------|-----------------|
| Chest | 10-16 | 2x | Mas variantes de push-up; menor carga absoluta |
| Back | 12-18 | 2-3x | +2-4 sets para compensar falta de pull_v |
| Shoulders | 8-14 | 2x | Enfasis en push_v con mancuernas |
| Biceps | 8-12 | 2-3x | +2 sets para compensar pull_v |
| Triceps | 6-10 | 2x | Suficiente con push compounds |
| Quads | 12-18 | 2-3x | +frecuencia; Bulgarian obligatorio |
| Hamstrings | 10-16 | 2x | Nordics + RDL; compensar falta de leg curl |
| Glutes | 10-16 | 2x | Buena cobertura con hinge/lunge |
| Calfs | 12-20 | 3-4x | Alto volumen, single-leg prioritario |
| Abs/Core | 8-12 | 2-3x | Buena cobertura bodyweight |

**Justificacion de ajustes:**

1. **Back +20% volumen**: Compensa falta de ejercicios de pull_v
2. **Biceps +25% volumen**: Compensa falta de chin-ups
3. **Quads +25% volumen**: Compensa falta de leg press/extension
4. **Calfs +50% volumen**: Alto umbral de estimulo sin maquinas
5. **Triceps -15% volumen**: Push compounds ya estimulan suficiente

### 4.3 Distribucion Semanal Recomendada

**Para 3 dias/semana (HOME):**

| Dia | Enfoque | Ejemplo |
|-----|---------|---------|
| Dia 1 | Upper Push + Pull | Chest, Shoulders, Back, Arms |
| Dia 2 | Lower | Quads, Hamstrings, Glutes, Calfs |
| Dia 3 | Full Body | Compounds principales + accesorios |

**Para 4 dias/semana (HOME):**

| Dia | Enfoque | Grupos |
|-----|---------|--------|
| Dia 1 | Upper Push | Chest, Shoulders, Triceps |
| Dia 2 | Lower Push | Quads, Calfs, Glutes (quad-dominant) |
| Dia 3 | Upper Pull | Back, Biceps, Rear Delts |
| Dia 4 | Lower Pull | Hamstrings, Glutes (hip-dominant), Calfs |

---

## 5. Progresiones para Principiantes HOME

### 5.1 Semana 1-2: Adaptacion Anatomica

**Objetivo:** Establecer patrones de movimiento correctos, adaptar tejido conectivo.

| Parametro | Valor |
|-----------|-------|
| RPE/RIR | 6-7 / 4-5 RIR |
| Sets por ejercicio | 2-3 |
| Reps | 12-15 |
| Tempo | 3-0-2-0 (controlado) |
| Descanso | 90-120 seg |

**Ejercicios prioritarios (patron de movimiento):**

| Patron | Ejercicio Semana 1-2 |
|--------|---------------------|
| push_h | Push-up (rodillas si necesario) |
| push_v | Dumbbell shoulder press (sentado) |
| pull_h | Dumbbell row (apoyado en banco/silla) |
| pull_v | Pullover con mancuerna |
| squat | Goblet squat |
| hinge | RDL con mancuernas (ligero) |
| lunge | Split squat estatico |
| core | Plank, dead bug |

**Checklist Semana 1-2:**
- [ ] Usuario puede completar movimiento con ROM completo
- [ ] Sin dolor articular
- [ ] Control excentrico demostrado
- [ ] Respiracion correcta (no Valsalva excesivo)

### 5.2 Semana 3-4: Acumulacion de Volumen

**Objetivo:** Incrementar volumen gradualmente, introducir proximidad al fallo.

| Parametro | Valor |
|-----------|-------|
| RPE/RIR | 7-8 / 2-3 RIR |
| Sets por ejercicio | 3-4 |
| Reps | 10-12 |
| Tempo | 2-0-2-0 |
| Descanso | 60-90 seg |

**Progresiones de ejercicios:**

| Patron | Progresion Semana 3-4 |
|--------|----------------------|
| push_h | Push-up completo, incline push-up |
| push_v | Standing dumbbell press |
| pull_h | Bent-over row (bilateral) |
| pull_v | Dominadas asistidas (si hay barra) |
| squat | Bulgarian split squat (bodyweight) |
| hinge | Single-leg RDL (asistido) |
| lunge | Reverse lunge dinamico |
| core | Plank lateral, mountain climbers |

**Incrementos de carga:**
- Mancuernas: +1-2 kg por semana si se completan todas las reps
- Bodyweight: Progresar a variante mas dificil

### 5.3 Semana 5-8: Intensificacion

**Objetivo:** Aumentar carga relativa, trabajar mas cerca del fallo.

| Parametro | Semana 5-6 | Semana 7-8 |
|-----------|------------|------------|
| RPE/RIR | 8 / 2 RIR | 8-9 / 1-2 RIR |
| Sets | 3-4 | 4 |
| Reps | 8-10 | 6-10 |
| Tempo | 2-0-1-0 | Controlado |

**Introduccion de tecnicas de intensidad (Semana 7-8):**

| Tecnica | Aplicacion HOME |
|---------|-----------------|
| Drop sets | Reducir peso de mancuerna inmediatamente |
| Rest-pause | 10-15 seg descanso, continuar al fallo |
| Tempo lento | 4 seg excentrico en ultimo set |
| Myo-reps | Set activacion + mini-sets |

---

## 6. Ejercicios Prohibidos/Permitidos

### 6.1 NUNCA Seleccionar para HOME

Ejercicios que requieren equipamiento de gimnasio y NO tienen sustituto efectivo:

| Ejercicio | Equipamiento | Razon de Exclusion |
|-----------|--------------|-------------------|
| Lat pulldown | Cable machine | Sin barra de traccion, usar pull_h |
| Leg press | Machine | Usar squat/lunge variations |
| Leg extension | Machine | Usar sissy squat |
| Leg curl | Machine | Usar nordic curl, slider curl |
| Cable crossover | Cable | Usar dumbbell fly |
| Pec deck | Machine | Usar dumbbell fly |
| Seated cable row | Cable | Usar dumbbell rows |
| Face pull | Cable | Usar band pull-apart (si hay bandas) |
| Tricep pushdown | Cable | Usar overhead extension, dips |
| Smith machine squats | Smith | Usar dumbbell/barbell squat |

**Query SQL para filtrar:**

```sql
-- Ejercicios a EXCLUIR para ambiente HOME
SELECT exercise_id, spanish_name
FROM exercises
WHERE equipment IN (
    'cable',
    'cable_machine',
    'machine',
    'smith_machine',
    'leg_press_machine',
    'hack_squat_machine',
    'lat_pulldown_machine'
)
```

### 6.2 SIEMPRE Disponibles (Bodyweight Core)

Ejercicios que pueden incluirse sin importar equipamiento declarado:

| Ejercicio | Patron | Musculo Principal |
|-----------|--------|-------------------|
| Push-up (y variantes) | push_h | Chest |
| Pike push-up | push_v | Shoulders |
| Dips (entre sillas) | push_h/push_v | Chest/Triceps |
| Plank (y variantes) | core | Abs |
| Mountain climbers | core | Abs |
| Squat (bodyweight) | squat | Quads |
| Lunge (y variantes) | lunge | Quads/Glutes |
| Glute bridge | hinge | Glutes |
| Superman | hinge | Lower back |
| Inverted row (mesa) | pull_h | Back |
| Nordic curl | arm | Hamstrings |

**Query SQL para bodyweight:**

```sql
-- Ejercicios bodyweight SIEMPRE disponibles
SELECT exercise_id, spanish_name, pattern, main_muscle
FROM exercises
WHERE equipment = 'bodyweight'
ORDER BY pattern, main_muscle
```

### 6.3 Matriz de Ejercicios por Equipamiento Declarado

| home_equipment | Ejercicios Adicionales Habilitados |
|----------------|-----------------------------------|
| `dumbbell` | +290 ejercicios (todos los patrones) |
| `kettlebell` | +157 ejercicios (hinge, lunge, squat especialmente) |
| `barbell` | +174 ejercicios (hinge, squat, compounds) |
| `pull_up_bar` | +12 ejercicios pull_v (dominadas) |
| `resistance_band` | Face pulls, lat pulldown asistido, curls |
| `bench` | Chest press, incline work, chest-supported rows |

---

## 7. Validacion de Rutina HOME

### 7.1 Checklist de Calidad

Toda rutina HOME generada por GymBot debe pasar estas validaciones:

**Cobertura Muscular:**
- [ ] Todos los grupos musculares principales cubiertos en la semana
- [ ] Ningn grupo muscular con 0 sets directos
- [ ] Quads, Hamstrings, Glutes tienen ejercicios dedicados (no solo compound)

**Balance de Patrones:**
- [ ] Ratio Push:Pull entre 1:1 y 1:1.2
- [ ] Si no hay pull_v disponible, pull_h aumentado proporcionalmente
- [ ] Hinge y Squat patterns ambos presentes en dias de pierna

**Progresion de Carga Viable:**
- [ ] Todos los ejercicios tienen variante de progresion (mas peso, mas dificil, mas reps)
- [ ] No hay ejercicios que requieran saltos de carga >5kg
- [ ] Ejercicios bodyweight tienen progresion documentada

**Tiempo de Sesion Realista:**
- [ ] Sesion completa estimada: 45-75 minutos
- [ ] Descansos incluidos en calculo
- [ ] No mas de 8 ejercicios por sesion (HOME tiene menos equipamiento = mas setup)

**Equipamiento Coherente:**
- [ ] Ningun ejercicio requiere equipamiento NO declarado por usuario
- [ ] Si `home_equipment = []`, solo bodyweight exercises
- [ ] Si `pull_up_bar = false`, no hay dominadas en rutina

### 7.2 Validaciones Automaticas (GymRatForm)

```javascript
// Pseudo-codigo para validacion en workflow
function validateHomeRoutine(routine, userProfile) {
    const errors = [];

    // 1. Verificar equipamiento
    routine.exercises.forEach(ex => {
        if (!isEquipmentAvailable(ex.equipment, userProfile.home_equipment)) {
            errors.push(`Ejercicio ${ex.name} requiere ${ex.equipment} no disponible`);
        }
    });

    // 2. Verificar cobertura muscular
    const musclesCovered = getMusclesCovered(routine);
    const requiredMuscles = ['Chest', 'Back', 'Shoulders', 'Quads', 'Hamstrings', 'Glutes'];
    requiredMuscles.forEach(muscle => {
        if (!musclesCovered.includes(muscle)) {
            errors.push(`Musculo ${muscle} no tiene ejercicios directos`);
        }
    });

    // 3. Verificar balance push/pull
    const pushSets = countSets(routine, ['push_h', 'push_v']);
    const pullSets = countSets(routine, ['pull_h', 'pull_v']);
    const ratio = pushSets / pullSets;
    if (ratio < 0.8 || ratio > 1.3) {
        errors.push(`Ratio push:pull desbalanceado (${ratio.toFixed(2)})`);
    }

    // 4. Verificar duracion
    const estimatedDuration = calculateDuration(routine);
    if (estimatedDuration > 90) {
        errors.push(`Sesion muy larga: ${estimatedDuration} minutos estimados`);
    }

    return errors;
}
```

### 7.3 Metricas de Calidad Post-Generacion

| Metrica | Valor Aceptable | Accion si Falla |
|---------|-----------------|-----------------|
| Ejercicios por sesion | 5-8 | Reducir/aumentar segun duracion |
| Sets totales/semana | 40-70 | Ajustar volumen por grupo |
| Ratio compuestos:aislamiento | 60:40 a 70:30 | Mas compuestos para HOME |
| Ejercicios unilaterales | Min 20% | Agregar variantes single-leg/arm |

---

## 8. Referencias Cientificas

### 8.1 Entrenamiento en Casa - Evidencia

| Estudio | Hallazgo Principal | Aplicacion GymBot |
|---------|-------------------|-------------------|
| Gentil et al. (2021) | Bodyweight training produce hipertrofia en principiantes | Rutinas bodyweight viables para nivel inicial |
| Kikuchi & Nakazato (2017) | Push-ups con carga equivalente a bench press 40-50% 1RM | Progresiones de push-up efectivas |
| Lacio et al. (2021) | Band resistance training comparable a pesas libres | Bandas elasticas como complemento valido |
| Schoenfeld (2010) | Sobrecarga progresiva es principio fundamental | Progresiones de carga deben ser claras |

### 8.2 Comparacion GYM vs HOME

| Parametro | GYM | HOME | Diferencia |
|-----------|-----|------|------------|
| Variedad de ejercicios | 1657 | 818 (49.4%) | HOME tiene ~50% menos opciones |
| Carga maxima posible | Alta | Media-Baja | HOME limitado por peso de mancuernas |
| Precision de carga | Alta (maquinas) | Media | Incrementos mayores en HOME |
| Pull vertical | Excelente | Limitado* | *Requiere barra de traccion |
| Leg isolation | Excelente | Limitado | Depende de Nordic curls, sissy squats |

### 8.3 Volumenes Optimos

Basado en Schoenfeld et al. (2017) "Dose-response relationship between weekly resistance training volume and increases in muscle mass":

- **Minimo efectivo:** 6 sets/semana por grupo muscular
- **Optimo para hipertrofia:** 10-20 sets/semana
- **Punto de retornos decrecientes:** >20 sets/semana

**Ajuste HOME:** Debido a menor intensidad absoluta, aumentar volumen 10-25% para compensar.

### 8.4 Frecuencia de Entrenamiento

Basado en Schoenfeld et al. (2016) "Effects of Resistance Training Frequency on Measures of Muscle Hypertrophy":

- Entrenar cada grupo muscular **2x/semana minimo** es superior a 1x/semana
- 3x/semana puede ser beneficioso para grupos rezagados
- HOME: Considerar 3x/semana para grupos con gaps (quads, hamstrings, calfs)

---

## Apendice A: Resumen de Reglas para IA

**Reglas de seleccion de ejercicios HOME:**

1. Filtrar ejercicios por `equipment IN (user.home_equipment + 'bodyweight')`
2. Si `pull_up_bar = false`: Compensar pull_v con pull_h (+50% sets)
3. Si `dumbbell = false`: Solo bodyweight, aplicar progresiones de tempo
4. Minimo 2 ejercicios por grupo muscular grande por semana
5. Incluir al menos 1 ejercicio compuesto por patron de movimiento
6. Priorizar ejercicios unilaterales para maximizar carga relativa

**Volumenes HOME por defecto:**

| Grupo | Sets/Semana HOME |
|-------|------------------|
| Chest | 12 |
| Back | 16 (compensado) |
| Shoulders | 10 |
| Arms | 12 (compensado) |
| Quads | 14 |
| Hamstrings | 12 |
| Glutes | 12 |
| Calfs | 16 |
| Core | 8 |

---

**Document Control**

| Version | Fecha | Autor | Cambios |
|---------|-------|-------|---------|
| 1.0 | 2026-02-03 | kiro-coach (Claude Code) | Creacion inicial |

---

*Este documento es parte del proyecto GymBot Home Training Feature. Para revisiones o sugerencias, contactar a code-reviewer.*
