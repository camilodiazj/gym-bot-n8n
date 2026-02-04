# Phase 3: Database Changes & Templates for HOME Training

**Document ID:** 03_Phase3_Database_Templates
**Version:** 1.0
**Status:** Ready for Implementation
**Created:** 2026-02-03
**Assigned to:** pixel-dev
**Reviewed by:** code-reviewer

---

## Table of Contents

1. [Objetivo](#1-objetivo)
2. [Schema Actual vs Propuesto](#2-schema-actual-vs-propuesto)
3. [Nuevos Templates HOME](#3-nuevos-templates-home)
4. [Day Requirements para HOME](#4-day-requirements-para-home)
5. [Verificacion de Datos](#5-verificacion-de-datos)
6. [Tareas Accionables](#6-tareas-accionables)
7. [Rollback Plan](#7-rollback-plan)
8. [Criterios de Aceptacion](#8-criterios-de-aceptacion)

---

## 1. Objetivo

Preparar la base de datos PostgreSQL (Supabase) para soportar rutinas de entrenamiento en ambiente HOME, incluyendo:

1. **Registro del valor 'HOME' en `routine_environments`** - Agregar HOME como ambiente valido
2. **Creacion de 75 nuevos templates HOME** - Matriz completa: 5 objetivos x 3 niveles x 5 schedules
3. **Day requirements optimizados para HOME** - Ajustar volumenes por patron segun disponibilidad de ejercicios
4. **Garantizar integridad referencial** - Todas las FKs y constraints validos

### Contexto de Ejercicios Disponibles

Analisis de ejercicios HOME-compatibles (equipment: bodyweight, dumbbell, kettlebell):

| Patron | Ejercicios HOME | Ejercicios GYM-only | Total | Cobertura HOME |
|--------|-----------------|---------------------|-------|----------------|
| `accessory` | 203 | 453 | 656 | 30.9% |
| `arm` | 105 | 131 | 236 | 44.5% |
| `core` | 43 | 52 | 95 | 45.3% |
| `hinge` | 18 | 50 | 68 | 26.5% |
| `lunge` | 66 | 40 | 106 | 62.3% |
| `pull_h` | 37 | 66 | 103 | 35.9% |
| `pull_v` | 18 | 25 | 43 | **41.9%** (critico) |
| `push_h` | 72 | 83 | 155 | 46.5% |
| `push_v` | 18 | 9 | 27 | 66.7% |
| `squat` | 64 | 84 | 148 | 43.2% |

**Patrones criticos identificados:**
- `pull_v` (18 ejercicios) - Requiere barra de traccion o bandas
- `hinge` (18 ejercicios) - Limitado sin barbell pesado

---

## 2. Schema Actual vs Propuesto

### 2.1 users_gym_profile (YA MODIFICADA - Completado)

Las siguientes columnas fueron agregadas en una migracion anterior:

```sql
-- Columnas ya aplicadas en users_gym_profile:
-- training_environment TEXT DEFAULT 'GYM' CHECK (training_environment IN ('GYM', 'HOME'))
-- home_equipment TEXT DEFAULT NULL

-- Verificacion del estado actual:
SELECT column_name, data_type, column_default, is_nullable
FROM information_schema.columns
WHERE table_name = 'users_gym_profile'
  AND column_name IN ('training_environment', 'home_equipment');

-- Resultado esperado:
-- training_environment | text | 'GYM'::text | YES
-- home_equipment       | text | null        | YES
```

**Valores validos para `home_equipment`** (separados por coma):
- Mancuernas
- Bandas elasticas
- Kettlebell
- Barra
- Barra de traccion
- Banco
- TRX
- Balon medicinal

### 2.2 routine_environments (PENDIENTE)

Actualmente solo existe el valor 'GYM'. Se debe agregar 'HOME':

```sql
-- Migration: add_home_environment
-- Description: Agregar valor HOME a routine_environments

INSERT INTO routine_environments (environment)
VALUES ('HOME')
ON CONFLICT (environment) DO NOTHING;
```

### 2.3 routine_templates (ESTRUCTURA EXISTENTE)

La tabla ya tiene la columna `environment` con FK a `routine_environments`:

```sql
-- Estructura actual de routine_templates:
-- template_id    TEXT PRIMARY KEY
-- week_schedule  TEXT REFERENCES week_schedules(schedule_type)
-- name           TEXT NOT NULL
-- days_per_week  BIGINT NOT NULL
-- goal           TEXT REFERENCES user_goals(goal)
-- level          TEXT REFERENCES user_levels(level)
-- environment    VARCHAR DEFAULT 'GYM' REFERENCES routine_environments(environment)
```

**Templates GYM existentes:** 75 (5 schedules x 5 goals x 3 levels)

---

## 3. Nuevos Templates HOME

### 3.1 Matriz de Templates a Crear

Se crean 75 nuevos templates para ambiente HOME (misma estructura que GYM):

| schedule | days | goal | levels | templates |
|----------|------|------|--------|-----------|
| fb_2 | 2 | 5 goals | 3 | 15 |
| fb_3 | 3 | 5 goals | 3 | 15 |
| ul_4 | 4 | 5 goals | 3 | 15 |
| ppl_5 | 5 | 5 goals | 3 | 15 |
| ppl_6 | 6 | 5 goals | 3 | 15 |
| **Total** | | | | **75** |

### 3.2 Convencion de Nomenclatura

```
tpl_{schedule}_{goal_abbrev}_{level_abbrev}_home

Donde:
- schedule: fb_2, fb_3, ul_4, ppl_5, ppl_6
- goal_abbrev: hyp (Ganar masa), cut (Bajar grasa), str (Fuerza), end (Resistencia), rec (Recomposicion)
- level_abbrev: beg (Principiante), int (Intermedio), adv (Avanzado)
```

**Ejemplos:**
- `tpl_fb_2_hyp_beg_home` - Full Body 2D, Hipertrofia, Principiante, HOME
- `tpl_ppl_5_cut_adv_home` - PPL 5D, Cutting, Avanzado, HOME

### 3.3 SQL de Insercion - Templates HOME

```sql
-- Migration: create_home_routine_templates
-- Description: Crear 75 templates de rutinas para ambiente HOME
-- Prerequisite: routine_environments debe tener valor 'HOME'

-- Ganar masa muscular (hyp) - HOME
INSERT INTO routine_templates (template_id, week_schedule, name, days_per_week, goal, level, environment) VALUES
-- Principiante
('tpl_fb_2_hyp_beg_home', 'fb_2', 'Full Body 2D Casa', 2, 'Ganar masa muscular', 'Principiante', 'HOME'),
('tpl_fb_3_hyp_beg_home', 'fb_3', 'Full Body 3D Casa', 3, 'Ganar masa muscular', 'Principiante', 'HOME'),
('tpl_ul_4_hyp_beg_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Ganar masa muscular', 'Principiante', 'HOME'),
('tpl_ppl_5_hyp_beg_home', 'ppl_5', 'PPL 5D Casa', 5, 'Ganar masa muscular', 'Principiante', 'HOME'),
('tpl_ppl_6_hyp_beg_home', 'ppl_6', 'PPL 6D Casa', 6, 'Ganar masa muscular', 'Principiante', 'HOME'),
-- Intermedio
('tpl_fb_2_hyp_int_home', 'fb_2', 'Full Body 2D Casa', 2, 'Ganar masa muscular', 'Intermedio', 'HOME'),
('tpl_fb_3_hyp_int_home', 'fb_3', 'Full Body 3D Casa', 3, 'Ganar masa muscular', 'Intermedio', 'HOME'),
('tpl_ul_4_hyp_int_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Ganar masa muscular', 'Intermedio', 'HOME'),
('tpl_ppl_5_hyp_int_home', 'ppl_5', 'PPL 5D Casa', 5, 'Ganar masa muscular', 'Intermedio', 'HOME'),
('tpl_ppl_6_hyp_int_home', 'ppl_6', 'PPL 6D Casa', 6, 'Ganar masa muscular', 'Intermedio', 'HOME'),
-- Avanzado
('tpl_fb_2_hyp_adv_home', 'fb_2', 'Full Body 2D Casa', 2, 'Ganar masa muscular', 'Avanzado', 'HOME'),
('tpl_fb_3_hyp_adv_home', 'fb_3', 'Full Body 3D Casa', 3, 'Ganar masa muscular', 'Avanzado', 'HOME'),
('tpl_ul_4_hyp_adv_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Ganar masa muscular', 'Avanzado', 'HOME'),
('tpl_ppl_5_hyp_adv_home', 'ppl_5', 'PPL 5D Casa', 5, 'Ganar masa muscular', 'Avanzado', 'HOME'),
('tpl_ppl_6_hyp_adv_home', 'ppl_6', 'PPL 6D Casa', 6, 'Ganar masa muscular', 'Avanzado', 'HOME'),

-- Bajar grasa (cut) - HOME
-- Principiante
('tpl_fb_2_cut_beg_home', 'fb_2', 'Full Body 2D Casa', 2, 'Bajar grasa', 'Principiante', 'HOME'),
('tpl_fb_3_cut_beg_home', 'fb_3', 'Full Body 3D Casa', 3, 'Bajar grasa', 'Principiante', 'HOME'),
('tpl_ul_4_cut_beg_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Bajar grasa', 'Principiante', 'HOME'),
('tpl_ppl_5_cut_beg_home', 'ppl_5', 'PPL 5D Casa', 5, 'Bajar grasa', 'Principiante', 'HOME'),
('tpl_ppl_6_cut_beg_home', 'ppl_6', 'PPL 6D Casa', 6, 'Bajar grasa', 'Principiante', 'HOME'),
-- Intermedio
('tpl_fb_2_cut_int_home', 'fb_2', 'Full Body 2D Casa', 2, 'Bajar grasa', 'Intermedio', 'HOME'),
('tpl_fb_3_cut_int_home', 'fb_3', 'Full Body 3D Casa', 3, 'Bajar grasa', 'Intermedio', 'HOME'),
('tpl_ul_4_cut_int_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Bajar grasa', 'Intermedio', 'HOME'),
('tpl_ppl_5_cut_int_home', 'ppl_5', 'PPL 5D Casa', 5, 'Bajar grasa', 'Intermedio', 'HOME'),
('tpl_ppl_6_cut_int_home', 'ppl_6', 'PPL 6D Casa', 6, 'Bajar grasa', 'Intermedio', 'HOME'),
-- Avanzado
('tpl_fb_2_cut_adv_home', 'fb_2', 'Full Body 2D Casa', 2, 'Bajar grasa', 'Avanzado', 'HOME'),
('tpl_fb_3_cut_adv_home', 'fb_3', 'Full Body 3D Casa', 3, 'Bajar grasa', 'Avanzado', 'HOME'),
('tpl_ul_4_cut_adv_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Bajar grasa', 'Avanzado', 'HOME'),
('tpl_ppl_5_cut_adv_home', 'ppl_5', 'PPL 5D Casa', 5, 'Bajar grasa', 'Avanzado', 'HOME'),
('tpl_ppl_6_cut_adv_home', 'ppl_6', 'PPL 6D Casa', 6, 'Bajar grasa', 'Avanzado', 'HOME'),

-- Mejorar fuerza (str) - HOME
-- Principiante
('tpl_fb_2_str_beg_home', 'fb_2', 'Full Body 2D Casa', 2, 'Mejorar fuerza', 'Principiante', 'HOME'),
('tpl_fb_3_str_beg_home', 'fb_3', 'Full Body 3D Casa', 3, 'Mejorar fuerza', 'Principiante', 'HOME'),
('tpl_ul_4_str_beg_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Mejorar fuerza', 'Principiante', 'HOME'),
('tpl_ppl_5_str_beg_home', 'ppl_5', 'PPL 5D Casa', 5, 'Mejorar fuerza', 'Principiante', 'HOME'),
('tpl_ppl_6_str_beg_home', 'ppl_6', 'PPL 6D Casa', 6, 'Mejorar fuerza', 'Principiante', 'HOME'),
-- Intermedio
('tpl_fb_2_str_int_home', 'fb_2', 'Full Body 2D Casa', 2, 'Mejorar fuerza', 'Intermedio', 'HOME'),
('tpl_fb_3_str_int_home', 'fb_3', 'Full Body 3D Casa', 3, 'Mejorar fuerza', 'Intermedio', 'HOME'),
('tpl_ul_4_str_int_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Mejorar fuerza', 'Intermedio', 'HOME'),
('tpl_ppl_5_str_int_home', 'ppl_5', 'PPL 5D Casa', 5, 'Mejorar fuerza', 'Intermedio', 'HOME'),
('tpl_ppl_6_str_int_home', 'ppl_6', 'PPL 6D Casa', 6, 'Mejorar fuerza', 'Intermedio', 'HOME'),
-- Avanzado
('tpl_fb_2_str_adv_home', 'fb_2', 'Full Body 2D Casa', 2, 'Mejorar fuerza', 'Avanzado', 'HOME'),
('tpl_fb_3_str_adv_home', 'fb_3', 'Full Body 3D Casa', 3, 'Mejorar fuerza', 'Avanzado', 'HOME'),
('tpl_ul_4_str_adv_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Mejorar fuerza', 'Avanzado', 'HOME'),
('tpl_ppl_5_str_adv_home', 'ppl_5', 'PPL 5D Casa', 5, 'Mejorar fuerza', 'Avanzado', 'HOME'),
('tpl_ppl_6_str_adv_home', 'ppl_6', 'PPL 6D Casa', 6, 'Mejorar fuerza', 'Avanzado', 'HOME'),

-- Mejorar resistencia (end) - HOME
-- Principiante
('tpl_fb_2_end_beg_home', 'fb_2', 'Full Body 2D Casa', 2, 'Mejorar resistencia', 'Principiante', 'HOME'),
('tpl_fb_3_end_beg_home', 'fb_3', 'Full Body 3D Casa', 3, 'Mejorar resistencia', 'Principiante', 'HOME'),
('tpl_ul_4_end_beg_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Mejorar resistencia', 'Principiante', 'HOME'),
('tpl_ppl_5_end_beg_home', 'ppl_5', 'PPL 5D Casa', 5, 'Mejorar resistencia', 'Principiante', 'HOME'),
('tpl_ppl_6_end_beg_home', 'ppl_6', 'PPL 6D Casa', 6, 'Mejorar resistencia', 'Principiante', 'HOME'),
-- Intermedio
('tpl_fb_2_end_int_home', 'fb_2', 'Full Body 2D Casa', 2, 'Mejorar resistencia', 'Intermedio', 'HOME'),
('tpl_fb_3_end_int_home', 'fb_3', 'Full Body 3D Casa', 3, 'Mejorar resistencia', 'Intermedio', 'HOME'),
('tpl_ul_4_end_int_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Mejorar resistencia', 'Intermedio', 'HOME'),
('tpl_ppl_5_end_int_home', 'ppl_5', 'PPL 5D Casa', 5, 'Mejorar resistencia', 'Intermedio', 'HOME'),
('tpl_ppl_6_end_int_home', 'ppl_6', 'PPL 6D Casa', 6, 'Mejorar resistencia', 'Intermedio', 'HOME'),
-- Avanzado
('tpl_fb_2_end_adv_home', 'fb_2', 'Full Body 2D Casa', 2, 'Mejorar resistencia', 'Avanzado', 'HOME'),
('tpl_fb_3_end_adv_home', 'fb_3', 'Full Body 3D Casa', 3, 'Mejorar resistencia', 'Avanzado', 'HOME'),
('tpl_ul_4_end_adv_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Mejorar resistencia', 'Avanzado', 'HOME'),
('tpl_ppl_5_end_adv_home', 'ppl_5', 'PPL 5D Casa', 5, 'Mejorar resistencia', 'Avanzado', 'HOME'),
('tpl_ppl_6_end_adv_home', 'ppl_6', 'PPL 6D Casa', 6, 'Mejorar resistencia', 'Avanzado', 'HOME'),

-- Salud general / recomposicion corporal (rec) - HOME
-- Principiante
('tpl_fb_2_rec_beg_home', 'fb_2', 'Full Body 2D Casa', 2, 'Salud general / recomposición corporal', 'Principiante', 'HOME'),
('tpl_fb_3_rec_beg_home', 'fb_3', 'Full Body 3D Casa', 3, 'Salud general / recomposición corporal', 'Principiante', 'HOME'),
('tpl_ul_4_rec_beg_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Salud general / recomposición corporal', 'Principiante', 'HOME'),
('tpl_ppl_5_rec_beg_home', 'ppl_5', 'PPL 5D Casa', 5, 'Salud general / recomposición corporal', 'Principiante', 'HOME'),
('tpl_ppl_6_rec_beg_home', 'ppl_6', 'PPL 6D Casa', 6, 'Salud general / recomposición corporal', 'Principiante', 'HOME'),
-- Intermedio
('tpl_fb_2_rec_int_home', 'fb_2', 'Full Body 2D Casa', 2, 'Salud general / recomposición corporal', 'Intermedio', 'HOME'),
('tpl_fb_3_rec_int_home', 'fb_3', 'Full Body 3D Casa', 3, 'Salud general / recomposición corporal', 'Intermedio', 'HOME'),
('tpl_ul_4_rec_int_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Salud general / recomposición corporal', 'Intermedio', 'HOME'),
('tpl_ppl_5_rec_int_home', 'ppl_5', 'PPL 5D Casa', 5, 'Salud general / recomposición corporal', 'Intermedio', 'HOME'),
('tpl_ppl_6_rec_int_home', 'ppl_6', 'PPL 6D Casa', 6, 'Salud general / recomposición corporal', 'Intermedio', 'HOME'),
-- Avanzado
('tpl_fb_2_rec_adv_home', 'fb_2', 'Full Body 2D Casa', 2, 'Salud general / recomposición corporal', 'Avanzado', 'HOME'),
('tpl_fb_3_rec_adv_home', 'fb_3', 'Full Body 3D Casa', 3, 'Salud general / recomposición corporal', 'Avanzado', 'HOME'),
('tpl_ul_4_rec_adv_home', 'ul_4', 'Upper/Lower 4D Casa', 4, 'Salud general / recomposición corporal', 'Avanzado', 'HOME'),
('tpl_ppl_5_rec_adv_home', 'ppl_5', 'PPL 5D Casa', 5, 'Salud general / recomposición corporal', 'Avanzado', 'HOME'),
('tpl_ppl_6_rec_adv_home', 'ppl_6', 'PPL 6D Casa', 6, 'Salud general / recomposición corporal', 'Avanzado', 'HOME');
```

---

## 4. Day Requirements para HOME

### 4.1 Estrategia de Ajuste de Volumen

Los templates HOME reutilizan los mismos `template_days` que GYM (la estructura de dias es identica).
La diferencia esta en como el workflow `GymRatForm` procesa los `day_requirements`:

**Reglas de ajuste para HOME (implementadas en n8n workflow, NO en BD):**

| Patron | min_sets GYM | Factor HOME | min_sets HOME | Razon |
|--------|--------------|-------------|---------------|-------|
| `pull_v` | 4-8 | 0.5x | 2-4 | Solo 18 ejercicios disponibles |
| `pull_h` | 6-8 | 1.25x | 8-10 | Compensar reduccion de pull_v |
| `hinge` | 4-8 | 0.75x | 3-6 | Solo 18 ejercicios disponibles |
| `lunge` | 0 | +4 | 4 | Agregar lunges para compensar hinge |
| `arm` | 4-10 | 1.0x | 4-10 | 105 ejercicios - buena cobertura |
| `core` | 4 | 1.0x | 4 | 43 ejercicios - suficiente |
| `squat` | 4-8 | 1.0x | 4-8 | 64 ejercicios - buena cobertura |
| `push_h` | 6-8 | 1.0x | 6-8 | 72 ejercicios - buena cobertura |
| `push_v` | 4-6 | 1.0x | 4-6 | 18 ejercicios pero variados |
| `accessory` | 6-8 | 1.0x | 6-8 | 203 ejercicios - excelente |

### 4.2 Nota Importante: No se Crean Nuevos day_requirements

Los `day_requirements` existentes estan ligados a `template_days`, que a su vez estan ligados a `week_schedules`.
Los templates HOME usan los MISMOS `week_schedules` que GYM, por lo tanto usan los MISMOS `template_days` y `day_requirements`.

**El ajuste de volumen para HOME se implementa en el workflow GymRatForm mediante:**

1. Un nodo Code que detecta `training_environment = 'HOME'`
2. Aplica los factores de ajuste listados arriba a los `min_sets`
3. Agrega `lunge` como patron complementario cuando `hinge` tiene volumen reducido

**Pseudocodigo del ajuste (para n8n-agent):**

```javascript
// En GymRatForm Supabase v3.json - ProcessHomeAdjustments node
if (userProfile.training_environment === 'HOME') {
  dayRequirements = dayRequirements.map(req => {
    switch(req.pattern) {
      case 'pull_v':
        return {...req, min_sets: Math.ceil(req.min_sets * 0.5)};
      case 'pull_h':
        return {...req, min_sets: Math.ceil(req.min_sets * 1.25)};
      case 'hinge':
        return {...req, min_sets: Math.ceil(req.min_sets * 0.75)};
      default:
        return req;
    }
  });

  // Agregar lunge si hay dias con hinge reducido
  if (dayRequirements.some(r => r.pattern === 'hinge')) {
    dayRequirements.push({
      pattern: 'lunge',
      min_sets: 4,
      priority: 'Medium'
    });
  }
}
```

### 4.3 Tabla de Referencia: Day Requirements por Schedule (GYM baseline)

#### Full Body 2D (fb_2)

| Day | Patron | min_sets | Priority |
|-----|--------|----------|----------|
| Full Body A | squat | 6 | High |
| Full Body A | push_h | 6 | High |
| Full Body A | pull_h | 6 | High |
| Full Body A | hinge | 4 | Medium |
| Full Body A | core | 4 | Medium |
| Full Body B | hinge | 6 | High |
| Full Body B | pull_v | 6 | High |
| Full Body B | push_v | 4 | Medium |
| Full Body B | squat | 4 | Medium |
| Full Body B | arm | 4 | Medium |
| Full Body B | core | 4 | Medium |

#### Full Body 3D (fb_3)

| Day | Patron | min_sets | Priority |
|-----|--------|----------|----------|
| Full Body A | squat | 6 | High |
| Full Body A | push_h | 6 | High |
| Full Body A | pull_h | 6 | High |
| Full Body A | core | 4 | Medium |
| Full Body B | hinge | 6 | High |
| Full Body B | pull_v | 6 | High |
| Full Body B | push_v | 4 | Medium |
| Full Body B | arm | 4 | Medium |
| Full Body B | core | 4 | Medium |
| Full Body C | squat | 4 | Medium |
| Full Body C | hinge | 4 | Medium |
| Full Body C | push_h | 4 | Medium |
| Full Body C | pull_v | 4 | Medium |
| Full Body C | arm | 6 | Medium |
| Full Body C | core | 4 | Medium |

#### Upper/Lower 4D (ul_4)

| Day | Patron | min_sets | Priority |
|-----|--------|----------|----------|
| Upper A | push_h | 8 | High |
| Upper A | pull_h | 8 | High |
| Upper A | push_v | 4 | Medium |
| Upper A | pull_v | 4 | Medium |
| Upper A | arm | 6 | Medium |
| Lower A | squat | 8 | High |
| Lower A | hinge | 6 | High |
| Lower A | accessory | 6 | Medium |
| Lower A | core | 4 | Medium |
| Upper B | push_h | 6 | High |
| Upper B | pull_h | 6 | High |
| Upper B | push_v | 6 | Medium |
| Upper B | pull_v | 6 | Medium |
| Upper B | arm | 6 | Medium |
| Lower B | hinge | 8 | High |
| Lower B | squat | 6 | High |
| Lower B | accessory | 6 | Medium |
| Lower B | core | 4 | Medium |

#### PPL 5D (ppl_5)

| Day | Patron | min_sets | Priority |
|-----|--------|----------|----------|
| Push | push_h | 8 | High |
| Push | push_v | 6 | High |
| Push | arm | 6 | Medium |
| Pull | pull_v | 8 | High |
| Pull | pull_h | 8 | High |
| Pull | arm | 6 | Medium |
| Legs | squat | 8 | High |
| Legs | hinge | 8 | High |
| Legs | accessory | 6 | Medium |
| Upper (Arms focus) | push_h | 6 | Medium |
| Upper (Arms focus) | pull_h | 6 | Medium |
| Upper (Arms focus) | arm | 10 | High |
| Upper (Arms focus) | core | 4 | Medium |
| Lower (Glutes/Posterior) | hinge | 6 | High |
| Lower (Glutes/Posterior) | squat | 6 | Medium |
| Lower (Glutes/Posterior) | accessory | 8 | High |
| Lower (Glutes/Posterior) | core | 4 | Medium |

#### PPL 6D (ppl_6)

| Day | Patron | min_sets | Priority |
|-----|--------|----------|----------|
| Push 1 | push_h | 8 | High |
| Push 1 | push_v | 6 | High |
| Push 1 | arm | 6 | Medium |
| Pull 1 | pull_v | 8 | High |
| Pull 1 | pull_h | 8 | High |
| Pull 1 | arm | 6 | Medium |
| Legs 1 | squat | 8 | High |
| Legs 1 | hinge | 8 | High |
| Legs 1 | accessory | 6 | Medium |
| Push 2 | push_h | 6 | High |
| Push 2 | push_v | 8 | High |
| Push 2 | arm | 4 | Medium |
| Pull 2 | pull_v | 6 | High |
| Pull 2 | pull_h | 8 | High |
| Pull 2 | arm | 4 | Medium |
| Legs 2 | hinge | 8 | High |
| Legs 2 | squat | 6 | High |
| Legs 2 | accessory | 6 | Medium |
| Legs 2 | core | 4 | Medium |

---

## 5. Verificacion de Datos

### 5.1 Queries de Validacion Post-Migration

```sql
-- 1. Verificar que HOME existe en routine_environments
SELECT * FROM routine_environments WHERE environment = 'HOME';
-- Expected: 1 row

-- 2. Contar templates por ambiente
SELECT environment, COUNT(*) as total_templates
FROM routine_templates
GROUP BY environment
ORDER BY environment;
-- Expected: GYM: 75, HOME: 75

-- 3. Verificar distribucion de templates HOME
SELECT
    environment,
    goal,
    level,
    COUNT(*) as count
FROM routine_templates
WHERE environment = 'HOME'
GROUP BY environment, goal, level
ORDER BY goal, level;
-- Expected: 5 rows por goal x level (5 schedules cada uno)

-- 4. Verificar integridad referencial
SELECT rt.template_id, rt.environment, re.environment as env_exists
FROM routine_templates rt
LEFT JOIN routine_environments re ON rt.environment = re.environment
WHERE re.environment IS NULL;
-- Expected: 0 rows (todos los templates tienen ambiente valido)

-- 5. Verificar ejercicios HOME disponibles por patron
SELECT
    pattern,
    COUNT(*) as home_exercises
FROM exercises
WHERE equipment IN ('bodyweight', 'dumbbell', 'kettlebell')
GROUP BY pattern
ORDER BY home_exercises ASC;
-- Expected: pull_v y hinge con menor cantidad

-- 6. Verificar cobertura minima por patron para HOME
WITH home_exercises AS (
    SELECT pattern, COUNT(*) as count
    FROM exercises
    WHERE equipment IN ('bodyweight', 'dumbbell', 'kettlebell')
    GROUP BY pattern
)
SELECT
    dr.pattern,
    COALESCE(he.count, 0) as available_exercises,
    MAX(dr.min_sets) as max_required_sets,
    CASE
        WHEN COALESCE(he.count, 0) >= MAX(dr.min_sets) THEN 'OK'
        ELSE 'WARNING'
    END as status
FROM day_requirements dr
LEFT JOIN home_exercises he ON dr.pattern = he.pattern
GROUP BY dr.pattern, he.count
ORDER BY status DESC, available_exercises ASC;
-- Expected: Todos OK o WARNING solo en pull_v, hinge

-- 7. Verificar que users_gym_profile tiene las columnas HOME
SELECT
    column_name,
    data_type,
    column_default
FROM information_schema.columns
WHERE table_name = 'users_gym_profile'
  AND column_name IN ('training_environment', 'home_equipment');
-- Expected: 2 rows
```

### 5.2 Query para Listar Ejercicios HOME por Patron (Debug)

```sql
-- Listado completo de ejercicios HOME para verificacion manual
SELECT
    pattern,
    equipment,
    exercise_id,
    spanish_name,
    main_muscle,
    role,
    level
FROM exercises
WHERE equipment IN ('bodyweight', 'dumbbell', 'kettlebell')
ORDER BY pattern, equipment, role, spanish_name;
```

---

## 6. Tareas Accionables

### Para pixel-dev

- [ ] **6.1** Ejecutar migration para agregar 'HOME' a `routine_environments`
- [ ] **6.2** Ejecutar migration para crear 75 templates HOME en `routine_templates`
- [ ] **6.3** Ejecutar queries de verificacion (Seccion 5.1)
- [ ] **6.4** Documentar resultados en comentario de PR
- [ ] **6.5** Crear backup de tablas antes de migration (opcional pero recomendado)

### Para code-reviewer

- [ ] **6.6** Revisar SQL de migrations antes de ejecucion
- [ ] **6.7** Validar queries de verificacion post-migration
- [ ] **6.8** Ejecutar regression tests para flujo GYM existente
- [ ] **6.9** Aprobar PR de migrations

### Para n8n-agent (Informativo - Fase 4)

- [ ] **6.10** Implementar logica de ajuste de volumen HOME en GymRatForm (ver Seccion 4.2)
- [ ] **6.11** Agregar filtro de ejercicios por equipment en GymRatForm
- [ ] **6.12** Actualizar queries de seleccion de template para incluir environment

---

## 7. Rollback Plan

En caso de problemas, ejecutar en orden inverso:

```sql
-- ROLLBACK Step 1: Eliminar templates HOME
DELETE FROM routine_templates WHERE environment = 'HOME';

-- Verificar eliminacion
SELECT COUNT(*) FROM routine_templates WHERE environment = 'HOME';
-- Expected: 0

-- ROLLBACK Step 2: Eliminar valor HOME de routine_environments
DELETE FROM routine_environments WHERE environment = 'HOME';

-- Verificar eliminacion
SELECT * FROM routine_environments;
-- Expected: Solo 'GYM'

-- ROLLBACK Step 3 (si se modificaron columnas de users_gym_profile):
-- NOTA: Las columnas training_environment y home_equipment ya existian
-- No es necesario eliminarlas a menos que se requiera rollback completo

-- Para rollback completo de columnas (DESTRUCTIVO - usar con cuidado):
-- ALTER TABLE users_gym_profile DROP COLUMN IF EXISTS training_environment;
-- ALTER TABLE users_gym_profile DROP COLUMN IF EXISTS home_equipment;
```

### Backup Pre-Migration (Recomendado)

```sql
-- Crear backup de routine_templates antes de modificar
CREATE TABLE routine_templates_backup_20260203 AS
SELECT * FROM routine_templates;

-- Crear backup de routine_environments
CREATE TABLE routine_environments_backup_20260203 AS
SELECT * FROM routine_environments;

-- Verificar backups
SELECT COUNT(*) FROM routine_templates_backup_20260203;
SELECT COUNT(*) FROM routine_environments_backup_20260203;
```

---

## 8. Criterios de Aceptacion

### Mandatory (Must Have)

| ID | Criterio | Verificacion |
|----|----------|--------------|
| AC-01 | Valor 'HOME' existe en `routine_environments` | Query SELECT devuelve 1 row |
| AC-02 | 75 templates HOME creados en `routine_templates` | COUNT = 75 WHERE environment = 'HOME' |
| AC-03 | Todos los templates HOME tienen FK validas | Query de integridad devuelve 0 rows |
| AC-04 | Nomenclatura de template_id sigue convencion | Todos terminan en `_home` |
| AC-05 | Templates cubren todas las combinaciones | 5 goals x 3 levels x 5 schedules = 75 |
| AC-06 | Flujo GYM existente no afectado | E2E tests GYM pasan |

### Optional (Nice to Have)

| ID | Criterio | Verificacion |
|----|----------|--------------|
| AC-07 | Backups creados antes de migration | Tablas `*_backup_*` existen |
| AC-08 | Queries de verificacion documentados | Este documento actualizado |
| AC-09 | Tiempo de migration < 30 segundos | Log de ejecucion |

### Acceptance Checklist

```
[ ] AC-01: HOME en routine_environments
[ ] AC-02: 75 templates HOME creados
[ ] AC-03: Integridad referencial OK
[ ] AC-04: Nomenclatura correcta
[ ] AC-05: Cobertura completa
[ ] AC-06: Regression tests GYM OK
[ ] AC-07: Backups creados (opcional)
[ ] AC-08: Documentacion actualizada (opcional)
[ ] AC-09: Performance OK (opcional)
```

---

## Appendix A: SQL Completo para Ejecucion

### Migration Script Completo

```sql
-- =====================================================
-- MIGRATION: Home Training Templates
-- Date: 2026-02-03
-- Author: pixel-dev
-- Description: Agregar soporte para rutinas HOME
-- =====================================================

-- Step 0: Create backups (recommended)
CREATE TABLE IF NOT EXISTS routine_templates_backup_20260203 AS
SELECT * FROM routine_templates;

CREATE TABLE IF NOT EXISTS routine_environments_backup_20260203 AS
SELECT * FROM routine_environments;

-- Step 1: Add HOME to routine_environments
INSERT INTO routine_environments (environment)
VALUES ('HOME')
ON CONFLICT (environment) DO NOTHING;

-- Step 2: Create HOME templates (75 total)
-- [Insert completo de Seccion 3.3]

-- Step 3: Verify
SELECT environment, COUNT(*) FROM routine_templates GROUP BY environment;
```

---

## Appendix B: Entity Relationship Diagram (Relevant Tables)

```
routine_environments
+---------------+
| environment   |  <-- PK ('GYM', 'HOME')
+---------------+
       |
       | FK
       v
routine_templates
+---------------+
| template_id   |  <-- PK
| week_schedule |  --> FK to week_schedules
| name          |
| days_per_week |
| goal          |  --> FK to user_goals
| level         |  --> FK to user_levels
| environment   |  --> FK to routine_environments
+---------------+
       |
       | uses same
       v
week_schedules
+---------------+
| schedule_type |  <-- PK (fb_2, fb_3, ul_4, ppl_5, ppl_6)
| detail        |
| days_per_week |
+---------------+
       |
       | FK
       v
template_days
+------------------+
| template_day_id  |  <-- PK
| week_schedule    |  --> FK to week_schedules
| day_number       |
| title            |
+------------------+
       |
       | FK
       v
day_requirements
+------------------+
| day_req_id       |  <-- PK
| template_day_id  |  --> FK to template_days
| pattern          |  --> FK to exercise_patterns
| min_sets         |
| priority         |
| day_name         |
+------------------+
```

---

**Document Control**

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-02-03 | Claude Code (pixel-dev) | Initial creation |

---

*Este documento es parte del proyecto GymBot Home Training Feature.*
*Para preguntas, contactar al Product Owner o al rol asignado.*
