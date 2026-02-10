# 01_DOMAIN_LOGIC.md -- Workout Creator Quality Fixes

**Version:** 1.0
**Date:** 2026-02-09
**Author:** Lead Solutions Architect
**Status:** Draft -- Pending Review

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Business Rules Index](#2-business-rules-index)
3. [Issue 1: W4 Volume Inflation -- Deload Week Rules](#3-issue-1-w4-volume-inflation----deload-week-rules)
4. [Issue 2: Duplicate Exercises -- Deduplication Rules](#4-issue-2-duplicate-exercises----deduplication-rules)
5. [Issue 3: Pattern vs Muscle Mismatch -- Reclassification Rules](#5-issue-3-pattern-vs-muscle-mismatch----reclassification-rules)
6. [Issue 4: No Cardio Role -- Cardio Role Parameters](#6-issue-4-no-cardio-role----cardio-role-parameters)
7. [Issue 5: Health Status Not Enforced -- Exclusion Lists](#7-issue-5-health-status-not-enforced----exclusion-lists)
8. [Day Pattern Validity Matrix](#8-day-pattern-validity-matrix)
9. [Validation Invariants](#9-validation-invariants)
10. [References](#10-references)

---

## 1. Executive Summary

This document defines the domain logic, business rules, and validation invariants for five quality fixes to the GymBot Workout Creator. Each fix addresses a specific defect discovered in the AI-driven exercise selection pipeline. The rules defined here are **authoritative** -- all downstream implementation documents (database migrations, n8n workflow changes, AI prompt updates) MUST conform to these constraints.

### Scope

| Issue | Severity | Impact |
|-------|----------|--------|
| W4 Volume Inflation | Medium | Deload week has MORE exercises than training weeks |
| Duplicate Exercises | High | Same exercise assigned multiple times in one session |
| Pattern/Muscle Mismatch | High | 20+ exercises with wrong movement pattern classification |
| No Cardio Role | Medium | ~40 cardio/plyo exercises get inappropriate isolation parameters |
| Health Status Not Enforced | Critical | Users with medical restrictions receive contraindicated exercises |

---

## 2. Business Rules Index

| Rule ID | Title | Issue | Enforcement Level |
|---------|-------|-------|-------------------|
| BR-001 | W4 Exercise Count Cap | #1 | Post-generation validator |
| BR-002 | W4 Exercise Identity Preservation | #1 | AI prompt + validator |
| BR-003 | W4 Volume Reduction Target | #1 | set_profiles table |
| BR-004 | Exercise ID Uniqueness Per Session | #2 | SQL constraint + validator |
| BR-005 | Exercise Name Variant Deduplication | #2 | AI prompt constraint |
| BR-006 | Movement Pattern Muscle Alignment | #3 | Database migration |
| BR-007 | Cardio Role Classification | #4 | Database migration |
| BR-008 | Cardio Role Set Profiles | #4 | set_profiles table insert |
| BR-009 | Health Status B Exclusions | #5 | SQL WHERE clause |
| BR-010 | Health Status C Exclusions | #5 | SQL WHERE clause |
| BR-011 | Health Status D Exclusions | #5 | SQL WHERE clause |
| BR-012 | Health Status E Exclusions | #5 | SQL WHERE clause |
| BR-013 | Day Pattern Muscle Validity | #3/#5 | AI prompt + validator |
| BR-014 | Maximum Exercises Per Session | All | Post-generation validator |
| BR-015 | Minimum Exercises Per Session | All | Post-generation validator |

---

## 3. Issue 1: W4 Volume Inflation -- Deload Week Rules

### Sports Science Rationale

A deload week is a planned period of reduced training stress designed to mitigate physiological and psychological fatigue, promote recovery, and enhance preparedness for subsequent training (Pritchard et al., 2023). Evidence-based consensus recommends:

- **Maintain exercise selection** (same exercises as W1-W3)
- **Reduce volume** (fewer sets per exercise, typically 40-60% of peak volume)
- **Reduce intensity** (higher RIR, lighter loads)
- **Maintain training frequency** (same number of training days)

The key insight: deloading should NOT add more exercises. The current bug occurs because fewer sets per exercise means shorter session duration, causing the duration validator to add exercises to fill time. This violates the fundamental purpose of deloading.

### Business Rules

#### BR-001: W4 Exercise Count Cap

> **For every training day, the number of distinct exercises in Week 4 MUST be less than or equal to the number of distinct exercises in Week 1 for the same day.**

```
INVARIANT: COUNT(DISTINCT exercise_id WHERE week=4 AND day_name=D)
           <= COUNT(DISTINCT exercise_id WHERE week=1 AND day_name=D)
           FOR ALL days D in user_weekly_schedule
```

**Implementation guidance:**
- After the AI generates exercises for all 4 weeks, a post-generation Code node MUST validate this constraint
- If W4 has more exercises than W1 for any day, REMOVE the lowest-priority exercises (by `exercise_order` descending) from W4 until the count matches W1
- Removal priority: isolation > core > compound (never remove compound exercises from W4)

#### BR-002: W4 Exercise Identity Preservation

> **Week 4 MUST use the same exercise_ids as Week 1 for the same day. W4 is a lighter version of W1, not a different workout.**

```
INVARIANT: {exercise_id WHERE week=4 AND day_name=D}
           IS A SUBSET OF
           {exercise_id WHERE week=1 AND day_name=D}
           FOR ALL days D
```

**Rationale:** Research shows deloading should maintain the same movement patterns to preserve motor learning while reducing fatigue. Athletes maintain exercise selection during deloads, altering only volume and intensity variables (Pritchard et al., 2023; Grandou et al., 2020).

#### BR-003: W4 Volume Reduction Target

> **Week 4 total weekly volume (sets x reps summed across all exercises) MUST be 40-60% of Week 3 total weekly volume.**

The existing `set_profiles` table already correctly implements W4 deload parameters:

| Role | W3 Sets | W4 Sets | W3 RIR | W4 RIR | Reduction |
|------|---------|---------|--------|--------|-----------|
| compound | 4-5 | 2-3 | 0-1 | 3-4 | ~50% |
| isolation | 4-5 | 2 | 0-1 | 3-4 | ~50% |
| core | 3-4 | 2 | 0-1 | 3-4 | ~50% |
| cardio (new) | 3-4 | 2 | N/A | N/A | ~50% |

These set/rep reductions are correct. The fix is to prevent the duration validator from backfilling exercises.

**Duration validator modification:**
```
IF week == 4:
    SKIP duration-based exercise addition
    Apply BR-001 (cap at W1 count)
    Apply BR-002 (subset of W1 exercises)
```

---

## 4. Issue 2: Duplicate Exercises -- Deduplication Rules

### Problem Statement

The AI agent selects the same `exercise_id` multiple times within a single training session. Example: "Peso Muerto Rumano a Una Pierna" (single-leg Romanian deadlift) appeared 3 times in one session for a beginner HOME user. This is never appropriate -- performing the same exercise 3 times in one session provides no benefit over doing more sets of that exercise.

### Business Rules

#### BR-004: Exercise ID Uniqueness Per Session

> **Within a single training day (same `user_id`, same `week`, same `day_name`), each `exercise_id` MUST appear exactly once.**

```
CONSTRAINT: UNIQUE(user_id, week, day_name, exercise_id) ON workouts table
```

**Enforcement layers:**
1. **Database:** Add a UNIQUE constraint on `(user_id, week, day_name, exercise_id)` to the `workouts` table
2. **AI prompt:** Explicitly instruct: "Nunca repitas el mismo ejercicio dentro de un mismo dia de entrenamiento"
3. **Post-generation validator:** Check for duplicates before INSERT; if found, replace the duplicate with the next-best exercise from the same pattern/muscle group

#### BR-005: Exercise Name Variant Deduplication

> **Within a single training day, no two exercises may target the same `main_muscle` using the same `equipment` type unless they have different `pattern` classifications.**

```
SOFT CONSTRAINT: For any day D, if two exercises share
    (main_muscle, equipment), they MUST differ in pattern.
    Exception: "accessory" pattern exercises are exempt.
```

**Rationale:** This prevents the AI from selecting, for example, both "Curl con mancuerna" and "Curl martillo con mancuerna" (same muscle=Biceps, same equipment=dumbbell, same pattern=arm) in one session. The user benefits more from variety across equipment or movement angles.

**This is a soft constraint** -- it is enforced via AI prompt guidance and post-generation warnings but does not block workout creation.

---

## 5. Issue 3: Pattern vs Muscle Mismatch -- Reclassification Rules

### Problem Statement

20+ exercises have incorrect `pattern` classifications relative to their `main_muscle`. This causes them to appear in the wrong training days. For example, abdominal exercises classified as `push_h` will be selected on Push day instead of being treated as core work.

### Business Rule

#### BR-006: Movement Pattern Muscle Alignment

> **Every exercise's `pattern` classification MUST be biomechanically consistent with its `main_muscle`. The following reclassifications are REQUIRED.**

### Reclassification Table

#### Category A: Abs/Core exercises misclassified as push_h (10 exercises)

These exercises involve a plank or push-up position but their PRIMARY mover is the abdominals, not chest/triceps.

| exercise_id | spanish_name | Current | New Pattern | New Role | Rationale |
|-------------|-------------|---------|-------------|----------|-----------|
| `ex_alternating_bent_leg_raise` | Elevacion alternada de piernas flexionadas | push_h/compound | **core/core** | Primary mover: rectus abdominis |
| `ex_core_stability_4_crosslateral_limb_raise_push_up_position` | Estabilidad del Nucleo 4 Elevacion Cruzada | push_h/compound | **core/core** | Stability exercise, primary: abs |
| `ex_core_stability_5_crosslateral_limb_raise_into_knee_elbow_tuck_push_up_position` | Estabilidad del Nucleo 5 Levantamiento Cruzado | push_h/compound | **core/core** | Stability exercise, primary: abs |
| `ex_stability_ball_atomic_push_up` | Flexion atomica con balon de estabilidad | push_h/compound | **core/core** | Primary mover: abs (pike/tuck motion) |
| `ex_pushup_to_renegade_row` | Flexion con remo renegado | push_h/compound | **core/core** | Anti-rotation + plank dominant |
| `ex_bench_lift_off_active` | Levantamiento Activo de Banca | push_h/compound | **core/core** | Abs/hip flexor exercise |
| `ex_bench_lift_off_static` | Levantamiento Estatico en Banco | push_h/compound | **core/core** | Isometric abs exercise |
| `ex_barbell_hooklying_bench_press` | Press de banca con barra en posicion hook-lying | push_h/compound | **core/core** | Anti-extension core variant |
| `ex_barbell_larsen_bench_press` | Press de banca Larsen con barra | push_h/compound | **push_h/compound** | KEEP -- actually a bench press variant (Chest primary) |
| `ex_core_stability_regression_crosslateral_limb_raise_push_up_position` | Regresion de estabilidad del core | push_h/compound | **core/core** | Core stability exercise |

**Note on Larsen Press:** The Larsen bench press is a legitimate horizontal push exercise (main_muscle should be corrected to Chest, not Abs). This requires a `main_muscle` correction, not a pattern correction.

#### Category B: Quads exercises misclassified as push_h (3 exercises)

These are hip flexion exercises, not horizontal pressing movements.

| exercise_id | spanish_name | Current | New Pattern | New Role | Rationale |
|-------------|-------------|---------|-------------|----------|-----------|
| `ex_hip_flexions_standing_resisted` | Flexion de Cadera en Mini-Banda de Pie | push_h/compound | **accessory/isolation** | Hip flexor isolation |
| `ex_hip_flexions_straight_leg_standing_resisted` | Flexiones de Cadera con Pierna Recta | push_h/compound | **accessory/isolation** | Hip flexor isolation |
| `ex_hip_flexion_seated_bench_isometric` | Sosten de Flexion de Cadera Sentado | push_h/compound | **accessory/isolation** | Isometric hip flexion |

#### Category C: Abs exercises misclassified as pull_h (4 exercises)

These exercises involve a pull position (rowing stance) but the primary mover is the abdominals.

| exercise_id | spanish_name | Current | New Pattern | New Role | Rationale |
|-------------|-------------|---------|-------------|----------|-----------|
| `ex_stability_ball_stir_the_pot` | Balon de estabilidad: Remover la olla | pull_h/compound | **core/core** | Anti-extension core |
| `ex_cable_row_bar_kneeling_crunch` | Crunch Arrodillado con Barra de Remo | pull_h/compound | **core/core** | Kneeling crunch (abs primary) |
| `ex_rower_knee_tuck` | Encogimiento de Rodillas en Remo | pull_h/compound | **core/core** | Knee tuck (abs primary) |
| `ex_dumbbell_renegade_row` | Remo Renegado con Mancuernas | pull_h/compound | **pull_h/compound** | KEEP -- renegade row is a legitimate row |

**Note on Renegade Row:** While it involves significant core engagement, the primary movement pattern IS a horizontal pull. The `main_muscle` should be corrected from Abs to Back.

#### Category D: Abs exercises misclassified as push_v (3 exercises)

These are pike/tuck variations where the primary mover is the abdominals, not the shoulders.

| exercise_id | spanish_name | Current | New Pattern | New Role | Rationale |
|-------------|-------------|---------|-------------|----------|-----------|
| `ex_stability_ball_pike` | Elevacion de pelota de estabilidad | push_v/compound | **core/core** | Pike = abs primary |
| `ex_trx_pike` | Pike con TRX | push_v/compound | **core/core** | Pike = abs primary |
| `ex_rower_pike` | Remador Pica | push_v/compound | **core/core** | Pike = abs primary |

### Summary of Required Changes

| Change Type | Count |
|-------------|-------|
| Pattern: push_h -> core, Role: compound -> core | 8 |
| Pattern: push_h -> accessory, Role: compound -> isolation | 3 |
| Pattern: pull_h -> core, Role: compound -> core | 3 |
| Pattern: push_v -> core, Role: compound -> core | 3 |
| Main_muscle correction (Abs -> Chest): Larsen press | 1 |
| Main_muscle correction (Abs -> Back): Renegade row | 1 |
| **Total database updates** | **19** |

---

## 6. Issue 4: No Cardio Role -- Cardio Role Parameters

### Problem Statement

Approximately 40 exercises in the `exercises` table are cardio/plyometric movements (burpees, mountain climbers, box jumps, sprints, jumping jacks, assault bike, etc.) currently classified as `role = 'isolation'` with `pattern = 'accessory'`. They receive isolation parameters (3 sets x 12-15 reps, RIR 1-2, 75s rest), which is inappropriate:

- **Reps are wrong:** Cardio exercises should use time-based or high-rep prescriptions, not 12-15 reps
- **Rest periods are wrong:** Cardio circuits use 15-30s rest, not 75s
- **RIR is meaningless:** RIR (Reps In Reserve) does not apply to cardio movements
- **Tempo is wrong:** Cardio movements are performed explosively, not with controlled tempo

### Business Rules

#### BR-007: Cardio Role Classification

> **All exercises that are primarily cardiovascular/plyometric in nature MUST be classified as `role = 'cardio'`. The `exercise_role` table MUST include a `cardio` entry.**

**Identification criteria -- an exercise is `cardio` if ANY of the following are true:**
1. Exercise name contains: burpee, cardio, sprint, salto (jump), escalador (mountain climber), tijera (scissor/jack), shuttle, battle rope, assault bike, ski erg, box jump, quick feet, karaoke, skater, diamond hop
2. Equipment = 'machine' AND main_muscle = 'Calfs' AND pattern = 'accessory' (these are almost universally cardio drills)
3. The exercise is explosive/ballistic with no controlled eccentric phase

**Exercises that should NOT be reclassified as cardio:**
- Jump squats, jump lunges -> These remain `compound` under `squat`/`lunge` patterns (they are resistance exercises with a plyometric component)
- Calf raises with a jump -> Remain under their current pattern
- Battle rope variations that are primarily upper-body strength -> Keep as `isolation`

#### Exercise List for Cardio Reclassification

The following 38 exercises MUST have their role changed from `isolation` to `cardio`:

```
ex_burpee, ex_mountain_climber, ex_jumping_mountain_climber,
ex_slow_tempo_mountain_climber, ex_slalom_mountain_climber,
ex_switch_jump_mountain_climber, ex_trx_mountain_climber,
ex_cardio_assault_bike, ex_cardio_assault_bike_arms_only,
ex_cardio_box_quick_feet, ex_cardio_karaoke,
ex_cardio_criss_cross_jacks, ex_cardio_in_in_out_out_shuffle,
ex_cardio_lateral_shuffle, ex_cardio_ski_erg,
ex_cardio_lateral_quick_feet, ex_cardio_quick_feet,
ex_cardio_single_leg_forward_hop, ex_cardio_long_jump_shuffle_back,
ex_cardio_figure_eight_sprint, ex_cardio_three_step_heismans,
ex_cardio_in_and_outs, ex_cardio_single_leg_lateral_hop,
ex_cardio_jumping_jacks, ex_cardio_skater,
ex_cardio_shuttle_sprint, ex_treadmill_sprint,
ex_cardio_sprint_in_place, ex_cardio_forward_scissor,
ex_cardio_knee_taps, ex_cardio_seal_jacks,
ex_cardio_skater_to_single_leg_burpee, ex_cardio_long_jump,
ex_cardio_diamond_hop, ex_cardio_step_out_jacks,
ex_cardio_in_and_out_forward, ex_jump_rope,
ex_box_jump, ex_seated_box_jump, ex_depth_jump,
ex_jump_off_box_single_leg_landing, ex_scissor_kick,
ex_sideways_scissor_kick, ex_press_jack
```

**Also reclassify these band-based cardio exercises:**
```
ex_cardio_band_reverse_fly_jacks (currently push_h/compound -> accessory/cardio)
ex_cardio_band_seal_jacks (currently accessory/isolation -> accessory/cardio)
ex_cardio_band_hammer_curl_jacks (currently arm/isolation -> accessory/cardio)
ex_cardio_band_press_jacks (currently squat/compound -> accessory/cardio)
```

**Note:** `ex_single_leg_box_jump` and `ex_barbell_calf_jump` are advanced plyometric/strength hybrids and should remain as `isolation` (accessory pattern).

#### BR-008: Cardio Role Set Profiles

> **The `set_profiles` table MUST include entries for the `cardio` role across all 5 goals, 3 levels, and 4 weeks.**

### Cardio Set Profile Parameters

Cardio exercises are prescribed differently from resistance exercises. Key differences:
- **Reps represent seconds** (duration-based) rather than repetitions, except where explicit rep counts make sense (e.g., box jumps)
- **RIR is replaced by RPE guidance** in the notes column (since "reps in reserve" is meaningless for timed cardio)
- **Tempo is always "--"** (explosive/natural rhythm, no controlled eccentric)
- **Rest periods are short** (15-45s) to maintain elevated heart rate

#### Notation Convention

For cardio exercises, the `reps` column uses a seconds-based notation: `"20-30s"` means 20-30 seconds of work. The frontend and WhatsApp display logic MUST render this as time, not rep count.

For exercises where rep counting is natural (box jumps, burpees), standard rep notation applies: `"8-12"`.

#### Principiante (Beginner) -- All Goals

| Goal | Week | Sets | Reps | RIR | Rest (s) | Tempo | Notes |
|------|------|------|------|-----|----------|-------|-------|
| Ganar masa muscular | 1 | 2 | 20s | -- | 45 | -- | ritmo controlado |
| Ganar masa muscular | 2 | 2 | 25s | -- | 40 | -- | ritmo moderado |
| Ganar masa muscular | 3 | 3 | 25s | -- | 40 | -- | intensidad moderada |
| Ganar masa muscular | 4 | 2 | 20s | -- | 45 | -- | descarga |
| Bajar grasa | 1 | 3 | 25s | -- | 30 | -- | ritmo activo |
| Bajar grasa | 2 | 3 | 30s | -- | 25 | -- | densidad |
| Bajar grasa | 3 | 3-4 | 30s | -- | 20 | -- | pico densidad |
| Bajar grasa | 4 | 2 | 20s | -- | 40 | -- | descarga |
| Mejorar fuerza | 1 | 2 | 15s | -- | 45 | -- | calentamiento activo |
| Mejorar fuerza | 2 | 2 | 20s | -- | 40 | -- | activacion |
| Mejorar fuerza | 3 | 2 | 20s | -- | 40 | -- | activacion moderada |
| Mejorar fuerza | 4 | 2 | 15s | -- | 45 | -- | descarga |
| Mejorar resistencia | 1 | 3 | 30s | -- | 30 | -- | base aerobica |
| Mejorar resistencia | 2 | 3 | 35s | -- | 25 | -- | progresion |
| Mejorar resistencia | 3 | 4 | 40s | -- | 20 | -- | pico resistencia |
| Mejorar resistencia | 4 | 2 | 25s | -- | 40 | -- | descarga |
| Salud general / recomposicion corporal | 1 | 2 | 20s | -- | 40 | -- | base |
| Salud general / recomposicion corporal | 2 | 2 | 25s | -- | 35 | -- | progresion |
| Salud general / recomposicion corporal | 3 | 3 | 25s | -- | 30 | -- | moderado |
| Salud general / recomposicion corporal | 4 | 2 | 20s | -- | 40 | -- | descarga |

#### Intermedio (Intermediate) -- All Goals

| Goal | Week | Sets | Reps | RIR | Rest (s) | Tempo | Notes |
|------|------|------|------|-----|----------|-------|-------|
| Ganar masa muscular | 1 | 3 | 25s | -- | 40 | -- | complemento metabolico |
| Ganar masa muscular | 2 | 3 | 30s | -- | 35 | -- | sobrecarga metabolica |
| Ganar masa muscular | 3 | 3 | 30s | -- | 30 | -- | pico |
| Ganar masa muscular | 4 | 2 | 20s | -- | 45 | -- | descarga |
| Bajar grasa | 1 | 3 | 30s | -- | 25 | -- | base circuito |
| Bajar grasa | 2 | 3-4 | 35s | -- | 20 | -- | sobrecarga |
| Bajar grasa | 3 | 4 | 40s | -- | 15 | -- | pico HIIT |
| Bajar grasa | 4 | 2 | 25s | -- | 35 | -- | descarga |
| Mejorar fuerza | 1 | 2 | 20s | -- | 40 | -- | activacion |
| Mejorar fuerza | 2 | 2 | 25s | -- | 35 | -- | activacion moderada |
| Mejorar fuerza | 3 | 3 | 25s | -- | 30 | -- | densidad controlada |
| Mejorar fuerza | 4 | 2 | 15s | -- | 45 | -- | descarga |
| Mejorar resistencia | 1 | 3-4 | 35s | -- | 25 | -- | base |
| Mejorar resistencia | 2 | 4 | 40s | -- | 20 | -- | sobrecarga |
| Mejorar resistencia | 3 | 4-5 | 45s | -- | 15 | -- | pico densidad |
| Mejorar resistencia | 4 | 2-3 | 25s | -- | 35 | -- | descarga |
| Salud general / recomposicion corporal | 1 | 2-3 | 25s | -- | 35 | -- | base |
| Salud general / recomposicion corporal | 2 | 3 | 30s | -- | 30 | -- | sobrecarga |
| Salud general / recomposicion corporal | 3 | 3 | 30s | -- | 25 | -- | pico |
| Salud general / recomposicion corporal | 4 | 2 | 20s | -- | 40 | -- | descarga |

#### Avanzado (Advanced) -- All Goals

| Goal | Week | Sets | Reps | RIR | Rest (s) | Tempo | Notes |
|------|------|------|------|-----|----------|-------|-------|
| Ganar masa muscular | 1 | 3 | 30s | -- | 35 | -- | densidad metabolica |
| Ganar masa muscular | 2 | 3-4 | 35s | -- | 30 | -- | sobrecarga |
| Ganar masa muscular | 3 | 4 | 35s | -- | 25 | -- | pico metabolico |
| Ganar masa muscular | 4 | 2 | 25s | -- | 40 | -- | descarga |
| Bajar grasa | 1 | 4 | 35s | -- | 20 | -- | base alta |
| Bajar grasa | 2 | 4 | 40s | -- | 15 | -- | sobrecarga |
| Bajar grasa | 3 | 4-5 | 45s | -- | 10 | -- | pico HIIT extremo |
| Bajar grasa | 4 | 2-3 | 25s | -- | 35 | -- | descarga |
| Mejorar fuerza | 1 | 2-3 | 25s | -- | 35 | -- | activacion potente |
| Mejorar fuerza | 2 | 3 | 30s | -- | 30 | -- | potencia |
| Mejorar fuerza | 3 | 3 | 30s | -- | 25 | -- | pico potencia |
| Mejorar fuerza | 4 | 2 | 20s | -- | 40 | -- | descarga |
| Mejorar resistencia | 1 | 4 | 40s | -- | 20 | -- | base alta |
| Mejorar resistencia | 2 | 4-5 | 45s | -- | 15 | -- | sobrecarga |
| Mejorar resistencia | 3 | 5-6 | 50s | -- | 10 | -- | pico extremo |
| Mejorar resistencia | 4 | 2-3 | 30s | -- | 30 | -- | descarga |
| Salud general / recomposicion corporal | 1 | 3 | 30s | -- | 30 | -- | base fuerte |
| Salud general / recomposicion corporal | 2 | 3 | 35s | -- | 25 | -- | sobrecarga |
| Salud general / recomposicion corporal | 3 | 3-4 | 35s | -- | 20 | -- | pico |
| Salud general / recomposicion corporal | 4 | 2 | 25s | -- | 35 | -- | descarga |

### Cardio Exercise Ordering

> **Cardio role exercises MUST have `exercise_order` between 8-9 (after isolation, before cooldown).**

Current ordering:
- compound: 1-4
- core: 5-6
- isolation: 7+

New ordering:
- compound: 1-4
- core: 5-6
- isolation: 7-8
- **cardio: 9-10**

### Maximum Cardio Exercises Per Session

> **A single training session MUST NOT contain more than 2 cardio exercises.**

Rationale: Cardio exercises serve as metabolic finishers or active recovery, not as the primary training stimulus for a resistance training session.

---

## 7. Issue 5: Health Status Not Enforced -- Exclusion Lists

### Problem Statement

Health restriction codes (B, C, D, E) are currently passed as text instructions in the AI prompt, but the AI frequently ignores them. Example: A user with status D (spine issues) was assigned Romanian Deadlifts, which involve heavy axial loading.

**Solution:** Enforce health exclusions at the **SQL level** (WHERE clause when querying available exercises) AND in the AI prompt. Dual enforcement ensures no contraindicated exercise can ever reach the user.

### Business Rules

#### BR-009: Health Status B -- Lower Body Issues

> **Users with health_status = 'B' (lower body issues -- knee/ankle problems) MUST NOT be assigned exercises that impose high impact or heavy loading on knees and ankles.**

**Excluded patterns:**
- `lunge` (all lunge variations) -- high knee stress from single-leg loading
- `squat` WHERE equipment IN ('barbell') -- heavy axial loading on knees

**Excluded exercise name keywords** (SQL `ILIKE`):
```sql
'%salto%', '%jump%', '%box jump%', '%sprint%',
'%zancada%', '%lunge%', '%pistol%', '%sissy%',
'%sentadilla búlgara%'
```

**Excluded main_muscle values when combined with high-impact patterns:**
- Exercises with `main_muscle = 'Calfs'` AND `pattern = 'accessory'` (most are jumping/impact drills)

**Allowed alternatives:**
- Machine-based leg exercises (leg press, leg extension, leg curl)
- Seated/lying lower body exercises
- Low-impact squat variants (goblet squat, Smith machine squat)
- All upper body exercises (no restrictions)

**SQL WHERE clause addition:**
```sql
AND NOT (
    -- Exclude all lunges
    e.pattern = 'lunge'
    -- Exclude barbell squats (high axial load on knees)
    OR (e.pattern = 'squat' AND e.equipment = 'barbell')
    -- Exclude high-impact exercises
    OR e.spanish_name ILIKE ANY(ARRAY[
        '%salto%', '%jump%', '%sprint%',
        '%zancada%', '%pistol%', '%sissy%',
        '%sentadilla búlgara%'
    ])
    -- Exclude jumping cardio drills
    OR (e.main_muscle = 'Calfs' AND e.pattern = 'accessory'
        AND e.spanish_name ILIKE ANY(ARRAY['%cardio%', '%salto%', '%sprint%']))
)
```

#### BR-010: Health Status C -- Upper Body Issues

> **Users with health_status = 'C' (upper body issues -- shoulder problems) MUST NOT be assigned exercises that involve overhead pressing or positions that impinge the rotator cuff.**

**Excluded patterns:**
- `push_v` (all overhead pressing movements)

**Excluded exercise name keywords** (SQL `ILIKE`):
```sql
'%press militar%', '%overhead%', '%por encima de la cabeza%',
'%behind the neck%', '%detras del cuello%',
'%arnold%', '%push press%', '%jerk%',
'%snatch%', '%clean and press%',
'%upright row%', '%remo al menton%',
'%elevacion lateral%' -- only if above 90 degrees
```

**Excluded specific exercises:**
```sql
-- Behind-the-neck variations (high impingement risk)
WHERE e.spanish_name ILIKE '%detrás del cuello%'
OR e.spanish_name ILIKE '%behind%neck%'
-- Upright rows (internal rotation under load)
OR e.exercise_id LIKE '%upright_row%'
```

**Allowed alternatives:**
- All horizontal push (push_h) exercises
- All pull exercises (pull_h, pull_v) -- pulling is generally safe for shoulder issues
- Front raises below 90 degrees
- All lower body exercises (no restrictions)

**SQL WHERE clause addition:**
```sql
AND NOT (
    -- Exclude all overhead pressing
    e.pattern = 'push_v'
    -- Exclude specific high-risk exercises
    OR e.spanish_name ILIKE ANY(ARRAY[
        '%press militar%', '%overhead%',
        '%por encima de la cabeza%',
        '%detrás del cuello%', '%behind%neck%',
        '%push press%', '%jerk%', '%snatch%',
        '%remo al mentón%', '%upright row%'
    ])
    OR e.exercise_id LIKE '%upright_row%'
)
```

#### BR-011: Health Status D -- Spine Issues

> **Users with health_status = 'D' (spine issues -- disc, vertebral, or chronic back pain) MUST NOT be assigned exercises that impose heavy axial loading on the spine or require significant spinal flexion/extension under load.**

**Excluded patterns:**
- `hinge` WHERE equipment IN ('barbell') -- heavy deadlift variations
- `squat` WHERE equipment IN ('barbell') -- back squats, front squats

**Excluded exercise name keywords** (SQL `ILIKE`):
```sql
'%peso muerto%', '%deadlift%',
'%good morning%', '%buenos días%',
'%back squat%', '%sentadilla trasera%',
'%front squat%', '%sentadilla frontal%',
'%barbell row%', '%remo con barra%',
'%snatch%', '%clean%', '%jerk%',
'%hiperextensión%', '%hyperextension%',
'%rack pull%'
```

**Excluded main_muscle values in combination with heavy loading:**
- `main_muscle = 'Lower back'` AND `role = 'compound'` -- compound lower back exercises under load

**Allowed alternatives:**
- Machine-based exercises (leg press, Smith machine)
- Cable and dumbbell exercises (lower axial load)
- Bodyweight hinge variations (glute bridge, hip thrust)
- All upper body exercises that do not load the spine

**SQL WHERE clause addition:**
```sql
AND NOT (
    -- Exclude barbell hinges (deadlifts)
    (e.pattern = 'hinge' AND e.equipment = 'barbell')
    -- Exclude barbell squats
    OR (e.pattern = 'squat' AND e.equipment = 'barbell')
    -- Exclude compound lower back exercises
    OR (e.main_muscle = 'Lower back' AND e.role = 'compound')
    -- Exclude specific high-risk exercises
    OR e.spanish_name ILIKE ANY(ARRAY[
        '%peso muerto%', '%deadlift%',
        '%good morning%', '%buenos días%',
        '%back squat%', '%sentadilla trasera%',
        '%front squat%', '%sentadilla frontal%',
        '%remo con barra%', '%barbell row%',
        '%snatch%', '%clean%', '%jerk%',
        '%hiperextensión%', '%hyperextension%',
        '%rack pull%'
    ])
)
```

#### BR-012: Health Status E -- Special Condition

> **Users with health_status = 'E' (special medical condition) MUST be restricted to machine-based and low-risk exercises only. Free weights and explosive movements are excluded.**

**Allowed equipment only:**
- `machine`
- `cable`
- `bodyweight` (only for core and light accessory)

**Excluded equipment:**
- `barbell`
- `dumbbell` (except for exercises weighing < 5 kg, which cannot be filtered programmatically -- rely on AI prompt)
- `resistance_band` (unpredictable resistance curves)

**Excluded roles/patterns:**
- `role = 'cardio'` (all cardio/plyometric exercises)
- `pattern = 'hinge'` WHERE `equipment != 'machine'` AND `equipment != 'cable'`
- All exercises with `level = 'Avanzado'`

**Additional AI prompt restriction:**
- Prioritize seated and supported positions
- Avoid explosive concentric phases
- Use controlled tempos only (2-0-2 minimum)

**SQL WHERE clause addition:**
```sql
AND (
    -- Only allow safe equipment
    e.equipment IN ('machine', 'cable')
    -- Allow bodyweight ONLY for core and light accessory
    OR (e.equipment = 'bodyweight' AND e.pattern IN ('core', 'accessory')
        AND e.role != 'cardio')
)
AND e.level != 'Avanzado'
AND e.role != 'cardio'
```

### Health Status Enforcement Architecture

```
                    +---------------------+
                    | User Profile Query  |
                    | health_status = ?   |
                    +----------+----------+
                               |
                    +----------v----------+
                    | Exercise Query Node |
                    | (Supabase SELECT)   |
                    +----------+----------+
                               |
              +----------------+----------------+
              |                |                |
    +---------v----+  +--------v-----+  +-------v------+
    | Base WHERE   |  | Health       |  | Equipment    |
    | pattern,     |  | Exclusion    |  | Availability |
    | main_muscle  |  | WHERE clause |  | WHERE clause |
    +---------+----+  +--------+-----+  +-------+------+
              |                |                |
              +----------------+----------------+
                               |
                    +----------v----------+
                    | Filtered Exercise   |
                    | Pool (safe only)    |
                    +----------+----------+
                               |
                    +----------v----------+
                    | AI Agent Selection  |
                    | (from safe pool)    |
                    +---------------------+
```

**Dual enforcement guarantees:**
1. **SQL level:** Contraindicated exercises are never sent to the AI
2. **AI prompt level:** Reinforces restrictions as a defense-in-depth measure
3. **Post-generation validator:** Final check that no excluded exercise was inserted

---

## 8. Day Pattern Validity Matrix

### Business Rule

#### BR-013: Day Pattern Muscle Validity

> **For each day type in the training schedule, only exercises whose `main_muscle` falls within the VALID set for that day's patterns may be selected. This matrix defines which muscles are appropriate for each day.**

### Push Day (push_h + push_v + arm patterns)

| main_muscle | Valid? | Rationale |
|-------------|--------|-----------|
| Chest | YES | Primary push_h muscle |
| Triceps | YES | Synergist in all push movements |
| Shoulders | YES | Primary push_v muscle |
| Front Shoulders | YES | Active in overhead and incline press |
| Abs | NO | Move to core pattern |
| Back | NO | Antagonist (pull) |
| Biceps | NO | Antagonist (pull) |
| Quads | NO | Lower body |
| Glutes | NO | Lower body |
| Hamstrings | NO | Lower body |
| Calfs | NO | Lower body |
| Lower back | NO | Spinal stabilizer, not a push muscle |

### Pull Day (pull_h + pull_v + arm patterns)

| main_muscle | Valid? | Rationale |
|-------------|--------|-----------|
| Back | YES | Primary pull muscle |
| Biceps | YES | Synergist in all pull movements |
| Rear Shoulders | YES | Active in horizontal pulls |
| Traps | YES | Active in vertical pulls |
| Traps (mid-back) | YES | Rhomboids, active in rows |
| Upper Traps | YES | Active in shrugs and pulls |
| Forearms | YES | Grip involvement in pulls |
| Abs | NO | Move to core pattern |
| Chest | NO | Antagonist (push) |
| Triceps | NO | Antagonist (push) |
| Quads | NO | Lower body |
| Glutes | NO | Lower body |

### Legs Day (squat + hinge + lunge + accessory patterns)

| main_muscle | Valid? | Rationale |
|-------------|--------|-----------|
| Quads | YES | Primary squat muscle |
| Glutes | YES | Primary hinge/squat muscle |
| Hamstrings | YES | Primary hinge muscle |
| Calfs | YES | Lower leg accessory |
| Lower back | YES | Stabilizer in squats/hinges |
| Groin | YES | Adductor work on leg days |
| Abs | YES (core) | Core stability for heavy leg work |
| Chest | NO | Upper body push |
| Back | NO | Upper body pull |
| Biceps | NO | Upper body pull |
| Triceps | NO | Upper body push |
| Shoulders | NO | Upper body push |

### Upper Day (push_h + pull_h + push_v + pull_v + arm patterns)

| main_muscle | Valid? | Rationale |
|-------------|--------|-----------|
| Chest | YES | Push muscles |
| Back | YES | Pull muscles |
| Shoulders | YES | Push muscles |
| Front Shoulders | YES | Push muscles |
| Rear Shoulders | YES | Pull muscles |
| Biceps | YES | Pull/arm muscles |
| Triceps | YES | Push/arm muscles |
| Traps | YES | Pull muscles |
| Traps (mid-back) | YES | Pull muscles |
| Upper Traps | YES | Pull muscles |
| Forearms | YES | Arm muscles |
| Quads | NO | Lower body |
| Glutes | NO | Lower body |
| Hamstrings | NO | Lower body |
| Calfs | NO | Lower body |

### Lower Day (squat + hinge + lunge + accessory + core patterns)

| main_muscle | Valid? | Rationale |
|-------------|--------|-----------|
| Quads | YES | Squat dominant |
| Glutes | YES | Hinge/squat dominant |
| Hamstrings | YES | Hinge dominant |
| Calfs | YES | Accessory |
| Lower back | YES | Stabilizer |
| Groin | YES | Adductors |
| Abs | YES | Core pattern included |
| Core | YES | Core pattern included |
| Chest | NO | Upper body |
| Back | NO | Upper body |
| Biceps | NO | Upper body |
| Triceps | NO | Upper body |
| Shoulders | NO | Upper body |

### Full Body Day (all patterns)

All `main_muscle` values are valid for Full Body days, as these sessions include patterns from both upper and lower body.

### Upper Arms Day (arm + push_h + pull_h + core patterns)

This is a specialized PPL_5 day ("Upper Arms focus") with emphasis on arm isolation.

| main_muscle | Valid? | Rationale |
|-------------|--------|-----------|
| Biceps | YES | Primary arm focus |
| Triceps | YES | Primary arm focus |
| Forearms | YES | Arm accessory |
| Chest | YES | push_h pattern included |
| Back | YES | pull_h pattern included |
| Shoulders | YES | Involved in pressing |
| Abs | YES | Core pattern included |
| Quads | NO | Lower body |
| Glutes | NO | Lower body |
| Hamstrings | NO | Lower body |

### Lower Glutes/Posterior Day (hinge + accessory + squat + core patterns)

This is a specialized PPL_5 day with emphasis on glutes and posterior chain.

| main_muscle | Valid? | Rationale |
|-------------|--------|-----------|
| Glutes | YES | Primary focus |
| Hamstrings | YES | Posterior chain |
| Lower back | YES | Posterior chain |
| Quads | YES | Squat pattern included |
| Calfs | YES | Lower body accessory |
| Abs | YES | Core pattern included |
| Groin | YES | Adductors |
| Chest | NO | Upper body |
| Back | NO | Upper body |
| Biceps | NO | Upper body |
| Triceps | NO | Upper body |
| Shoulders | NO | Upper body |

---

## 9. Validation Invariants

These are post-generation checks that MUST pass before any workout plan is committed to the `workouts` table. If any invariant fails, the plan MUST be rejected and regenerated.

#### BR-014: Maximum Exercises Per Session

> **No single training day may contain more than 10 distinct exercises.**

```
INVARIANT: COUNT(DISTINCT exercise_id WHERE user_id=U AND week=W AND day_name=D) <= 10
           FOR ALL U, W, D
```

**Rationale:** More than 10 exercises per session exceeds reasonable session duration (60-90 minutes) and indicates the AI overfilled the workout.

#### BR-015: Minimum Exercises Per Session

> **Every training day MUST contain at least 4 distinct exercises.**

```
INVARIANT: COUNT(DISTINCT exercise_id WHERE user_id=U AND week=W AND day_name=D) >= 4
           FOR ALL U, W, D
```

**Rationale:** Fewer than 4 exercises indicates an insufficient training stimulus or a generation failure.

### Complete Invariant Checklist

| # | Invariant | Rule Ref | Check Level |
|---|-----------|----------|-------------|
| V-01 | W4 exercise count <= W1 exercise count per day | BR-001 | Post-gen |
| V-02 | W4 exercises are a subset of W1 exercises per day | BR-002 | Post-gen |
| V-03 | No duplicate exercise_id within same (user, week, day) | BR-004 | DB + Post-gen |
| V-04 | All exercises have valid pattern for their main_muscle | BR-006 | DB migration |
| V-05 | No cardio exercise has isolation parameters | BR-008 | DB migration |
| V-06 | Health B users have no excluded exercises | BR-009 | SQL WHERE |
| V-07 | Health C users have no push_v exercises | BR-010 | SQL WHERE |
| V-08 | Health D users have no barbell hinge/squat exercises | BR-011 | SQL WHERE |
| V-09 | Health E users only have machine/cable/bodyweight(core) | BR-012 | SQL WHERE |
| V-10 | Each exercise's main_muscle is valid for the day type | BR-013 | Post-gen |
| V-11 | Max 10 exercises per session | BR-014 | Post-gen |
| V-12 | Min 4 exercises per session | BR-015 | Post-gen |
| V-13 | Max 2 cardio exercises per session | BR-008 | Post-gen |
| V-14 | W4 total volume is 40-60% of W3 total volume | BR-003 | Post-gen |
| V-15 | exercise_order follows role hierarchy (compound < core < isolation < cardio) | -- | Post-gen |
| V-16 | Exercises from W1-W3 have consistent exercise_ids (same exercises across weeks) | BR-002 | Post-gen |

### Invariant Failure Handling

| Failure Type | Action |
|-------------|--------|
| V-01 fails (W4 too many exercises) | Remove lowest-priority exercises from W4 |
| V-02 fails (W4 has novel exercises) | Replace with matching exercises from W1 |
| V-03 fails (duplicates) | Remove duplicate, replace with next-best exercise from same pattern |
| V-06 to V-09 fails (health violation) | REJECT plan, re-query exercises with correct health filter |
| V-10 fails (wrong muscle for day) | Remove exercise, replace with valid-muscle exercise |
| V-11 fails (too many exercises) | Remove lowest-priority exercises |
| V-12 fails (too few exercises) | Add exercises from required patterns |
| V-13 fails (too many cardio) | Remove excess cardio, keep first 2 by exercise_order |

---

## 10. References

### Sports Science Literature

1. Pritchard, H.J., et al. (2023). "Integrating Deloading into Strength and Physique Sports Training Programmes: An International Delphi Consensus Approach." *Sports Medicine - Open*, 9, 87. [PMC10511399](https://pmc.ncbi.nlm.nih.gov/articles/PMC10511399/)

2. Grandou, C., et al. (2020). "Deloading Practices in Strength and Physique Sports: A Cross-sectional Survey." *Sports Medicine - Open*, 10, 33. [PMC10948666](https://pmc.ncbi.nlm.nih.gov/articles/PMC10948666/)

3. Sella, F.S., et al. (2024). "Gaining more from doing less? The effects of a one-week deload period during supervised resistance training on muscular adaptations." *PeerJ*, 12, e16777. [PMC10809978](https://pmc.ncbi.nlm.nih.gov/articles/PMC10809978/)

4. American College of Sports Medicine (2009). "Progression Models in Resistance Training for Healthy Adults." *Medicine & Science in Sports & Exercise*, 41(3), 687-708. [PubMed 19204579](https://pubmed.ncbi.nlm.nih.gov/19204579/)

5. Herring, C.H., et al. (2025). "Quantification of weekly strength-training volume per muscle group in competitive physique athletes." *Frontiers in Sports and Active Living*. [Frontiers](https://www.frontiersin.org/journals/sports-and-active-living/articles/10.3389/fspor.2025.1536360/full)

6. NSCA (National Strength and Conditioning Association). "Basics of Strength and Conditioning Manual." [NSCA](https://www.nsca.com/contentassets/de9aebfe7a7340b69217b99bb13862a7/basics_of_strength_and_conditioning_manual.pdf)

### Rehabilitation / Contraindication Sources

7. AAOS (American Academy of Orthopaedic Surgeons). "Knee Conditioning Program." [OrthoInfo](https://orthoinfo.aaos.org/en/recovery/knee-conditioning-program/)

8. Mattacola, C.G. & Dwyer, M.K. (2002). "Rehabilitation of the Ankle After Acute Sprain or Chronic Instability." *Journal of Athletic Training*, 37(4), 413-429. [PMC164373](https://pmc.ncbi.nlm.nih.gov/articles/PMC164373/)

9. AAOS. "Rotator Cuff and Shoulder Conditioning Program." [OrthoInfo](https://orthoinfo.aaos.org/globalassets/pdfs/2017-rehab_shoulder.pdf)

10. Wilk, K.E., et al. (2009). "Rehabilitation for Shoulder Instability." *Journal of Orthopaedic & Sports Physical Therapy*. [PMC5611703](https://pmc.ncbi.nlm.nih.gov/articles/PMC5611703/)

---

## Appendix A: Profile ID Convention for Cardio Set Profiles

The `profile_id` for new cardio entries MUST follow the existing naming convention:

```
{goal_prefix}_{level_prefix}_w{week}_cardio
```

Where:
- `goal_prefix`: `hyp` (Ganar masa muscular), `cut` (Bajar grasa), `str` (Mejorar fuerza), `end` (Mejorar resistencia), `rec` (Salud general)
- `level_prefix`: `beg` (Principiante), `int` (Intermedio), `adv` (Avanzado)
- `week`: 1-4

Example: `cut_int_w2_cardio` = Bajar grasa, Intermedio, Week 2, Cardio role

Total new rows: 5 goals x 3 levels x 4 weeks = **60 new set_profiles entries**.

## Appendix B: Implementation Priority

| Priority | Issue | Effort | Risk if Deferred |
|----------|-------|--------|------------------|
| P0 | Health Status Enforcement (#5) | Medium | **Critical** -- users with medical conditions get unsafe exercises |
| P1 | Duplicate Exercises (#2) | Low | High -- visible quality defect |
| P1 | Pattern/Muscle Mismatch (#3) | Low | High -- wrong exercises appear on wrong days |
| P2 | Cardio Role (#4) | Medium | Medium -- cardio exercises get suboptimal parameters |
| P2 | W4 Volume Inflation (#1) | Medium | Medium -- deload week is less effective |

## Appendix C: Exercises Excluded from Cardio Reclassification

The following exercises have plyometric elements but should retain their current classification because their primary training stimulus is resistance-based:

| exercise_id | spanish_name | Current Classification | Rationale for Keeping |
|-------------|-------------|----------------------|----------------------|
| `ex_jump_squats` | Sentadillas con salto | squat/compound | Primary: squat strength + power |
| `ex_bodyweight_pop_squat` | Sentadilla Con Salto | squat/compound | Primary: squat movement |
| `ex_in_and_out_jump_squat` | Sentadilla con salto in-and-out | squat/compound | Primary: squat movement |
| `ex_trx_jumping_squat` | Sentadilla con salto en TRX | squat/compound | Primary: squat with TRX support |
| `ex_bodyweight_alternating_jump_lunge` | Zancada Alterna con Salto | lunge/compound | Primary: lunge pattern |
| `ex_bodyweight_lateral_lunge_jump` | Zancada Lateral con Salto | lunge/isolation | Primary: lunge pattern |
| `ex_single_leg_box_jump` | Salto al Cajon con Una Pierna | accessory/isolation | Advanced plyometric (keep) |
| `ex_barbell_calf_jump` | Salto de gemelos con barra | accessory/isolation | Loaded calf exercise |
| `ex_cardio_row_erg_rower` | Cardio Remo Ergometro | pull_h/compound | Rowing is pull pattern |
| `ex_cardio_row_erg_rower_legs_only` | Remo cardio solo piernas | pull_h/compound | Rowing variant |
| `ex_cardio_row_erg_rower_four_stroke_sprint_start` | Salida de sprint en remo | pull_h/compound | Rowing variant |
