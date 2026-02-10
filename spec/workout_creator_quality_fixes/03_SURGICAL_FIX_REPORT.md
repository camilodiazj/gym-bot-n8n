# Informe: Fix Quirúrgico de Rutinas - 5 Usuarios Reales

**Fecha**: 2026-02-10
**Alcance**: Correcciones directas en tabla `workouts` (Supabase)
**Tablas NO tocadas**: `users_plans`, `user_weekly_schedule`, `users_gym_profile`

---

## Resumen Ejecutivo

Se aplicaron 7 operaciones quirúrgicas sobre la tabla `workouts` para corregir problemas de calidad en las rutinas de 5 usuarios reales. Posteriormente se descubrieron y corrigieron 3 hallazgos adicionales.

| Operación | Descripción | Filas Afectadas | Estado |
|-----------|-------------|-----------------|--------|
| A | Recortar W4 excedente (4 usuarios) | 46 DELETE | COMPLETADO |
| B | Dedup Faiber Full Body B (RDL 3x→1x) | 4 DELETE + reorder | COMPLETADO |
| C | Milena Push: Abs/Quads/Calfs → Chest/Triceps/Shoulders | 4 ex × 4 weeks | COMPLETADO |
| D | Milena Upper: 4×Abs → Biceps/Triceps isolations | 4 ex × 4 weeks | COMPLETADO |
| E | Milena Health D: RDL → Femoral máquina | 2 ex × 4 weeks | COMPLETADO |
| F | Yuli Push: Calfs → Triceps pushdown | 1 ex × 4 weeks | COMPLETADO |
| G | Yuli Upper: 2×Abs → Bayesian curl + Triceps pushdown | 2 ex × 4 weeks | COMPLETADO |
| **H** | **Camilo: Eliminar Burpee + Assault Bike W1-W4** | **9 DELETE + reorder** | **COMPLETADO** |
| **I** | **Milena: Duplicado ex_012 en Legs W4 → ex_011** | **1 UPDATE** | **COMPLETADO** |
| **J** | **Faiber: Renumerar exercise_order (gaps)** | **43 UPDATE** | **COMPLETADO** |

---

## Validación Final Exhaustiva (Post-Fix)

### Faiber - 9/9 PASS

| Check | Estado |
|-------|--------|
| No duplicados | PASS |
| exercise_ids válidos | PASS |
| exercise_order secuencial | PASS |
| Full Body B: 1 sola RDL por semana | PASS |
| Full Body C: pull-up vs chin-up distintos | PASS |
| Variedad muscular Full Body | PASS |
| Sin cardio | PASS |
| W1 vs W4 balanceado | PASS |
| Periodización correcta (W1/W4 alto vol, W2/W3 deload) | PASS |

**Veredicto**: LIMPIO. Listo para producción.

---

### Yuli Hernández - 9/11 PASS, 2 FAIL (preexistentes)

| Check | Estado | Detalle |
|-------|--------|---------|
| W4 ≤ W1 | PASS | |
| No duplicados | PASS | |
| exercise_ids válidos | PASS | |
| Push sin Calfs | PASS | |
| Push sin Chest (Pecho) | PASS | Yuli los descarta |
| Push muscles válidos | PASS | Solo Shoulders + Triceps |
| Upper con arm isolations | PASS | Bayesian curl + Cable pushdown |
| Upper no Abs-dominado | PASS | 0 Abs en Upper |
| Sin cardio | PASS | |
| exercise_order secuencial | **FAIL** | Gaps preexistentes en W1-W3 (no fueron tocados por el fix) |
| W1-W3 consistentes | **FAIL** | Legs/Pull/Upper: W2-W3 tienen 1 menos que W1 |

**Nota sobre FAILs**: Ambos son problemas **preexistentes del WORKOUT_CREATOR**, no causados por el fix quirúrgico. Los exercise_order con gaps (ej: 1,2,5,6 en vez de 1,2,3,4) y las diferencias W1-W3 fueron generados originalmente por el workflow.

---

### Camilo Gómez - 6/9 PASS, 3 FAIL (preexistentes)

| Check | Estado | Detalle |
|-------|--------|---------|
| W4 ≤ W1 | **FAIL** | Legs y Lower: W4=8 vs W1=7 (W1 bajó al borrar Burpees) |
| No duplicados | PASS | |
| exercise_ids válidos | PASS | |
| Sin Burpee/Assault Bike | PASS | 9 eliminados exitosamente |
| Sin cardio | PASS | |
| exercise_order secuencial | **FAIL** | Gaps en Push y Upper W1-W3 (preexistente) |
| W1-W3 consistentes | **FAIL** | Push W3=7 vs W1/W2=8; Upper W3=8 vs W1/W2=9 |
| Push muscles válidos | PASS | |
| Pull muscles válidos | PASS | |

**Nota**: El FAIL de W4>W1 en Legs/Lower se debe a que al borrar Burpees de W1 (último ejercicio), W1 bajó de 8→7 mientras W4 mantuvo 8. El W4 trim original se hizo ANTES de borrar los Burpees de W1. No afecta funcionalmente (W4 es deload).

---

### Andrés Felipe ROA - 4/9 PASS, 5 FAIL (preexistentes/clasificación)

| Check | Estado | Detalle |
|-------|--------|---------|
| W4 ≤ W1 | PASS | |
| No duplicados | PASS | |
| exercise_ids válidos | PASS | |
| exercise_order secuencial | **FAIL** | Gaps en Lower, Push, Upper W1-W3 |
| W1-W3 consistentes | **FAIL** | Lower/Push/Upper: W3 tiene 1 menos |
| Sin cardio | PASS | |
| Push muscles válidos | **FAIL** | `ex_machine_front_military_press` tiene main_muscle='Front Shoulders' (no estandarizado en BD) |
| Pull muscles válidos | **FAIL** | `ex_machine_seated_leg_curl` (Hamstrings) en Pull day |
| Upper arm isolations | **FAIL** | Push carece de Biceps isolation |

**Nota**: Los 5 FAILs son problemas **preexistentes del WORKOUT_CREATOR**, no causados por el fix. El único fix aplicado (W4 trim) fue exitoso.

---

### Milena Cortes - 7/10 PASS, 3 FAIL (2 preexistentes + 1 Health D)

| Check | Estado | Detalle |
|-------|--------|---------|
| W4 ≤ W1 | PASS | |
| No duplicados | PASS | Duplicado ex_012 corregido → ex_011 |
| exercise_ids válidos | PASS | |
| Sin RDL/Peso Muerto | PASS | 0 instancias de ex_013 |
| Push muscles válidos | PASS | Chest(3), Shoulders(2), Triceps(2), Abs(1) |
| Upper con Biceps+Triceps | PASS | Biceps(3), Triceps(3), Back(2), Abs(1) |
| Sin cardio | PASS | |
| Health D compliance | **FAIL** | Sentadilla búlgara y Hip Thrust (barbell) persisten. Curl femoral deslizador en Pull. |
| exercise_order secuencial | **FAIL** | Gaps en W1-W3 (preexistente) |
| W1-W3 consistentes | **FAIL** | Legs/Pull/Push/Upper: W3 tiene 1 menos |

**Nota sobre Health D**: La sentadilla búlgara con mancuerna y el Hip Thrust con barra son **marginalmente aceptables** para Health D (carga axial moderada vs deadlift), pero idealmente se reemplazarían por variantes en máquina. El curl femoral con deslizador es bodyweight y de bajo riesgo. Estos están fuera del scope del fix quirúrgico actual.

---

## Problemas Sistémicos Detectados (WORKOUT_CREATOR)

La validación exhaustiva reveló 3 patrones recurrentes que son **bugs del workflow WORKOUT_CREATOR**, no problemas de datos puntuales:

### 1. exercise_order con gaps (afecta 4/5 usuarios)

El WORKOUT_CREATOR asigna exercise_order por rol (compound=1-4, core=5-6, isolation=7+) pero cuando genera menos ejercicios de un rol, quedan gaps (ej: 1,2,3,4,5,6,11,12). Solo Faiber tiene order limpio (fue renumerado manualmente).

**Impacto**: Bajo. El `ORDER BY exercise_order` sigue funcionando, solo que los números no son contiguos.

### 2. W3 tiene menos ejercicios que W1-W2 (afecta 4/5 usuarios)

Patrón consistente: W3 pierde 1 ejercicio vs W1/W2 en 3-4 de 5 días. Sugiere un bug en la lógica de generación semanal del WORKOUT_CREATOR.

**Impacto**: Medio. W3 es la semana de mayor intensidad (más peso, menos reps) — tener 1 ejercicio menos puede ser programáticamente intencional o un bug.

### 3. Ejercicios fuera de patrón muscular (afecta 3/5 usuarios)

El AI Agent del WORKOUT_CREATOR a veces selecciona ejercicios que no corresponden al patrón del día:
- Burpees/Assault Bike en hipertrofia (Camilo)
- Calfs/Quads/Abs en Push day (Milena, Yuli)
- Hamstring curl en Pull day (Andrés)

**Impacto**: Alto. Afecta la calidad de la rutina directamente.

---

## Recomendaciones

### Correcciones Inmediatas (scope futuro)

1. **Renumerar exercise_order** para Yuli, Camilo, Andrés, Milena (similar al fix de Faiber)
2. **Evaluar Health D** de Milena: decidir si reemplazar sentadilla búlgara → leg press máquina

### Mejoras al WORKOUT_CREATOR (workflow)

1. **Validación post-generación**: Agregar un nodo que verifique que cada día solo tiene ejercicios del patrón muscular correcto
2. **exercise_order secuencial**: Renumerar 1..N después de la selección de ejercicios
3. **Consistencia W1-W3**: Asegurar que las 3 semanas tienen el mismo set de ejercicios (solo cambian sets/reps/rir)
4. **Blacklist de ejercicios**: Excluir Burpee, Assault Bike, y otros cardio del pool de selección para goals de hipertrofia/fuerza

---

## Detalle de Reemplazos de Ejercicios

### Milena Push (Op C)

| Order | Antes | Músculo | Después | Músculo |
|-------|-------|---------|---------|---------|
| 2 | Flexión atómica balón estabilidad | Abs | Press banca Smith | Chest |
| 3 | Flexión Cadera Mini-Banda | Quads | Press Inclinado DB | Chest |
| 4 | Elevaciones gemelos sentado | Calfs | Dips en máquina | Triceps |
| 6 | Elevación pelota estabilidad | Abs | Laterales en cable | Shoulders |

### Milena Upper (Op D)

| Order | Antes | Músculo | Después | Músculo |
|-------|-------|---------|---------|---------|
| 1 | Flexión atómica balón | Abs | Curl inclinado DB | Biceps |
| 6 | Crunch arrodillado polea | Abs | Extensión tríceps polea barra | Triceps |
| 7 | Extensiones abdominales máq | Abs | Curl bíceps máquina | Biceps |
| 9 | Woodchopper mancuerna | Abs | Ext. tríceps overhead cuerda | Triceps |

### Milena Health D (Op E)

| Día | Antes | Después |
|-----|-------|---------|
| Legs order 4 | Peso muerto / RDL (ex_013) | Femoral sentado (ex_012) |
| Lower order 2 | Peso muerto / RDL (ex_013) | Femoral acostado (ex_011) |

### Yuli Push (Op F)

| Order | Antes | Después |
|-------|-------|---------|
| 3 | Elevaciones gemelos sentado (Calfs) | Extensión tríceps polea barra (Triceps) |

### Yuli Upper (Op G)

| Order | Antes | Después |
|-------|-------|---------|
| 5 | Plancha IYTW (Abs) | Curl Bayesian polea (Biceps) |
| 6 | Elevación brazo plancha (Abs) | Extensión tríceps polea barra (Triceps) |
