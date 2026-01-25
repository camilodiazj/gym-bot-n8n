# Importación de Ejercicios desde MuscleWiki

Este documento describe el proceso para importar ejercicios desde archivos JSON de MuscleWiki a Supabase.

> **Nota**: Usar el script Python con el cliente de Supabase directamente, NO el MCP de Supabase. El script es más eficiente para operaciones batch y consume menos tokens.

## Changelog

### v2.0 (2026-01-24) - Deduplicación Automática ✨

**Cambios principales:**
- ✅ **Deduplicación automática**: Elimina 1,855 ejercicios duplicados (53% reducción)
- ✅ **IDs consistentes**: Formato `ex_{slug}` en lugar de `ex_{muscle}_{index}`
- ✅ **Directorio `raw/`**: Archivos JSON organizados en subcarpeta
- ✅ **Test suite**: 34 tests unitarios con cobertura completa
- ✅ **Error handling**: Manejo robusto de archivos corruptos/vacíos

**Breaking changes:**
- Los IDs de ejercicios cambian de `ex_biceps_001` a `ex_barbell_curl`
- Requiere migración si ya tienes datos (ver sección "Migración")

### v1.0 (anterior) - Versión Original

- Importación por archivo individual
- IDs basados en nombre de archivo
- Sin deduplicación (duplicados permitidos)

---

## Uso del Script

### Comandos Básicos

```bash
cd /Users/camilodiazjaimes/Documents/GymBot/exercises

# Preview de todos los ejercicios con deduplicación (RECOMENDADO)
python3 transform_exercises.py --all --dry-run --no-llm

# Insertar todos los ejercicios a Supabase (con deduplicación automática)
python3 transform_exercises.py --all --insert

# Con LLM para clasificación avanzada (más lento, requiere OpenAI API key)
python3 transform_exercises.py --all --insert

# Generar CSV de todos los ejercicios únicos
python3 transform_exercises.py --all --output exercises_all.csv

# Procesar un solo archivo (legacy, sin deduplicación)
python3 transform_exercises.py raw/Biceps.json --dry-run
```

> **Importante**: El flag `--all` ahora procesa automáticamente todos los archivos en `raw/` con deduplicación. Esto elimina 1,855 ejercicios duplicados de un total de 3,471 ejercicios crudos, resultando en 1,616 ejercicios únicos.

### Opciones

| Flag | Descripción |
|------|-------------|
| `--dry-run` | Preview sin escribir archivos ni insertar a DB |
| `--insert` | Insertar directamente a Supabase (con upsert) |
| `--all` | Procesar todos los `*.json` en `raw/` con deduplicación |
| `--no-llm` | Deshabilitar clasificación con LLM (más rápido) |
| `--output FILE` | Especificar nombre del CSV de salida |
| `--openai-key KEY` | API key de OpenAI (alternativa a env var) |

### Estructura de Archivos

```
exercises/
├── raw/                              # 43 archivos JSON de MuscleWiki
│   ├── Abdominales.json              # 3,471 ejercicios totales
│   ├── Biceps.json                   # Con 1,855 duplicados
│   ├── Pecho.json
│   └── ...
├── transform_exercises.py            # Script principal v2.0
├── test_transform_exercises.py       # Tests (pytest)
├── test_transform_exercises_unittest.py  # Tests (unittest, sin deps)
├── README.md                         # Este documento
└── TESTING.md                        # Guía de testing

---

## Deduplicación Automática

El script **v2.0** ahora incluye deduplicación automática basada en URL slugs:

### ¿Por qué es necesario?

Los 43 archivos JSON organizados por músculo contienen **991 ejercicios duplicados**. El mismo ejercicio aparece en múltiples archivos porque trabaja varios músculos.

**Ejemplo**: `kettlebell-single-arm-curtsy-lunge` aparece en:
- `Cuadriceps.json`
- `Gluteo mayor.json`
- `Gluteo medio.json`
- `Gluteos.json`
- `Parte interna del cuadriceps.json`

### Cómo funciona

1. **Identificador único**: El slug de la URL (`barbell-curl`) se usa como identificador canónico
2. **Primera ocurrencia gana**: Si un ejercicio aparece en múltiples archivos, se guarda solo la primera ocurrencia
3. **IDs consistentes**: El mismo ejercicio siempre genera el mismo `exercise_id` (`ex_barbell_curl`)

### Estadísticas de Deduplicación

```
Files processed: 43
Total raw exercises: 3,471
Unique exercises: 1,616
Duplicates removed: 1,855
```

### Ejemplo de Salida

```bash
$ python3 transform_exercises.py --all --dry-run --no-llm

Collecting exercises from 44 files...
  Warning: Skipping raw/Ingle.json - Expecting value: line 1 column 1 (char 0)
  Found 1616 unique exercises (removed 1855 duplicates)

Transforming exercises...

Preview (first 10):

  ID              Name                                Pattern      Role       Secondary
  ------------------------------------------------------------------------------------------
  ex_barbell_curl Barbell Curl                        arm          isolation  {}
  ex_barbell_squat Barbell Squat                      squat        compound   {"Cuádriceps"}
  ex_dumbbell_lateral_raise Dumbbell Lateral Raise    push_v       isolation  {}
  ...

  Deduplication Summary:
    Files processed: 43
    Total raw exercises: 3,471
    Unique exercises: 1,616
    Duplicates removed: 1,855

============================================================
Total unique exercises: 1616
```

---

## Estructura de Archivos JSON (MuscleWiki)

```json
{
  "name": "Extensión de tríceps con mancuerna",
  "muscles": [
    {"name": "Tríceps", "name_en_us": "Triceps", "level": 0},
    {"name": "Cabeza larga", "level": 1}
  ],
  "category": {"name_en_us": "Dumbbells"},
  "difficulty": {"name": "Intermedio"},
  "target_url": {"male": "dumbbells/male/triceps/..."}
}
```

### Reglas de Extracción

| Campo JSON | Campo DB | Regla |
|------------|----------|-------|
| `muscles[0]` donde `level=0` | `main_muscle` | Primer músculo nivel 0 |
| `muscles[1+]` donde `level=0` | `secondary_muscles` | Resto de músculos nivel 0 |
| `muscles` donde `level=1+` | (ignorados) | Sub-músculos no se importan |

---

## Clasificación de Patterns

### Por Keywords (automático)

| Pattern | Keywords |
|---------|----------|
| `arm` | curl, extension, pushdown, kickback, skull, tricep, bicep |
| `push_h` | bench, press banca, push-up, chest press, fly |
| `push_v` | overhead, shoulder press, military, dip, lateral raise |
| `pull_h` | row, remo, face pull, t-bar |
| `pull_v` | pulldown, pull-up, chin-up, dominada, lat pulldown |
| `squat` | squat, sentadilla, leg press, hack |
| `hinge` | deadlift, peso muerto, hip thrust, romanian |
| `lunge` | lunge, zancada, split squat, step-up |
| `core` | plank, crunch, abdominal, leg raise |

### Por LLM (casos no detectados)

Ejercicios que no matchean keywords se clasifican con OpenAI GPT-4o-mini.

---

## Esquema de Base de Datos

### Tabla `exercises`

```sql
-- Columna agregada para multi-músculo
ALTER TABLE exercises
ADD COLUMN secondary_muscles TEXT[] DEFAULT '{}';
```

### Campos

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `exercise_id` | TEXT PK | Formato: `ex_{slug}` (ej: `ex_barbell_curl`, `ex_dumbbell_lateral_raise`) |
| `name` | TEXT | Nombre en inglés (generado del slug) |
| `spanish_name` | TEXT | Nombre en español (original del JSON) |
| `main_muscle` | TEXT FK | Músculo principal (inglés) |
| `Músculo Principal` | TEXT | Músculo principal (español) |
| `secondary_muscles` | TEXT[] | Array de músculos secundarios (español) |
| `pattern` | TEXT FK | Patrón de movimiento |
| `role` | TEXT | compound, isolation, core |
| `equipment` | TEXT | dumbbell, barbell, machine, cable, bodyweight |
| `level` | VARCHAR FK | Principiante, Intermedio, Avanzado |
| `unilateral` | TEXT | Yes/No |
| `link` | TEXT | URL de MuscleWiki |

**Cambio importante (v2.0)**: Los IDs ahora se generan del slug de la URL en lugar del nombre del archivo. Esto garantiza que el mismo ejercicio siempre tenga el mismo ID sin importar en qué archivo JSON aparezca.

---

## Queries de Ejemplo

```sql
-- Buscar ejercicios que trabajen un músculo como secundario
SELECT spanish_name, main_muscle, secondary_muscles
FROM exercises
WHERE 'Pecho' = ANY(secondary_muscles);

-- Buscar ejercicios que trabajen Tríceps (principal o secundario)
SELECT spanish_name, main_muscle, secondary_muscles
FROM exercises
WHERE main_muscle = 'Triceps'
   OR 'Tríceps' = ANY(secondary_muscles);

-- Contar ejercicios por músculo secundario
SELECT unnest(secondary_muscles) as muscle, COUNT(*)
FROM exercises
WHERE secondary_muscles != '{}'
GROUP BY muscle
ORDER BY COUNT(*) DESC;
```

---

## Migración desde v1.0 a v2.0

Si ya tienes ejercicios en la base de datos con el formato antiguo de IDs (`ex_biceps_001`), necesitas migrar:

### Opción 1: Borrar y Re-importar (Recomendado para desarrollo)

```sql
-- En Supabase SQL Editor
TRUNCATE exercises CASCADE;
```

Luego ejecutar:
```bash
python3 transform_exercises.py --all --insert
```

### Opción 2: Migración con Script (Para producción con datos relacionados)

Si tienes datos en `workouts` que referencian los IDs antiguos:

```sql
-- 1. Crear tabla de mapeo (old_id -> new_id basado en link/slug)
CREATE TEMP TABLE exercise_id_mapping AS
SELECT
  exercise_id as old_id,
  'ex_' || regexp_replace(split_part(link, '/', -1), '-', '_', 'g') as new_id
FROM exercises;

-- 2. Actualizar referencias en workouts
UPDATE workouts w
SET exercise_id = m.new_id
FROM exercise_id_mapping m
WHERE w.exercise_id = m.old_id;

-- 3. Borrar ejercicios antiguos
TRUNCATE exercises CASCADE;

-- 4. Re-importar con nuevo script
-- (ejecutar transform_exercises.py --all --insert)
```

---

## Troubleshooting

### Error: "permission denied for table exercises"

Ejecutar en Supabase:
```sql
GRANT ALL ON exercises TO service_role;
GRANT ALL ON muscles TO service_role;
```

### Error: "duplicate key value violates unique constraint"

El campo `coloquial_name` tiene constraint UNIQUE. El script usa `NULL` para evitar conflictos.

### Error: "OpenAI not available" o "Supabase client not available"

```bash
pip3 install openai supabase
```

### Error: Muchos ejercicios duplicados en la base de datos

Si importaste archivos individuales antes de la v2.0, tendrás duplicados. Solución:

```sql
-- Ver duplicados (mismo link, diferentes IDs)
SELECT link, COUNT(*) as count
FROM exercises
GROUP BY link
HAVING COUNT(*) > 1;

-- Solución: Borrar todo y re-importar con --all
TRUNCATE exercises CASCADE;
```

Luego:
```bash
python3 transform_exercises.py --all --insert
```

### Warning: "Skipping {file}.json - Expecting value: line 1 column 1"

Archivo JSON vacío o corrupto. Esto es normal - el script lo salta automáticamente. Ejemplo: `Ingle.json` está vacío en el dataset de MuscleWiki.

---

## Mapeo de Músculos (Español → Inglés)

| Español | Inglés (DB) |
|---------|-------------|
| Tríceps | Triceps |
| Bíceps | Biceps |
| Hombros | Shoulders |
| Pecho | Chest |
| Espalda | Back |
| Dorsales | Back |
| Cuádriceps | Quads |
| Femorales | Hamstrings |
| Glúteos | Glutes |
| Abdominales | Abs |
| Pantorrillas | Calfs |
| Trapecio | Traps |

---

## Agregar Nuevos Archivos JSON

1. Descargar JSON de MuscleWiki para el músculo deseado
2. Guardar en `exercises/raw/{muscle}.json`
3. Ejecutar:
   ```bash
   python3 transform_exercises.py --all --dry-run --no-llm  # Preview primero
   python3 transform_exercises.py --all --insert            # Insertar después
   ```

> **Importante**: Siempre usar `--all` para procesar todos los archivos juntos. Esto activa la deduplicación automática.

El script automáticamente:
- **Deduplica ejercicios** por URL slug (no más duplicados)
- Genera IDs consistentes (`ex_{slug}` en lugar de `ex_{muscle}_{index}`)
- Extrae músculos principales y secundarios
- Clasifica patterns con keywords (o LLM si se omite `--no-llm`)
- Inserta/actualiza en Supabase (upsert por `exercise_id`)

---

## Testing

El script incluye un test suite completo. Ver [TESTING.md](TESTING.md) para detalles.

```bash
# Ejecutar tests (sin dependencias)
python3 test_transform_exercises_unittest.py

# 34 tests cubriendo:
# - Deduplicación (correcta eliminación de 1,855 duplicados)
# - Generación de IDs consistentes
# - Clasificación de patterns
# - Extracción de músculos (nivel 0 vs nivel 1)
# - Formateo PostgreSQL
```

---

## Comparación v1.0 vs v2.0

| Aspecto | v1.0 (Antigua) | v2.0 (Actual) |
|---------|----------------|---------------|
| **Comando** | `python3 transform_exercises.py biceps.json --insert` | `python3 transform_exercises.py --all --insert` |
| **Duplicados** | ❌ Permitidos (3,471 ejercicios) | ✅ Eliminados (1,616 únicos) |
| **exercise_id** | `ex_biceps_001` (basado en archivo) | `ex_barbell_curl` (basado en slug) |
| **Consistencia** | ❌ Mismo ejercicio = diferentes IDs en diferentes archivos | ✅ Mismo ejercicio = mismo ID siempre |
| **Directorio** | Archivos en raíz de `exercises/` | Archivos en `exercises/raw/` |
| **Tests** | ❌ Sin tests | ✅ 34 tests unitarios |
| **Stats** | ❌ Sin reporte | ✅ Deduplication Summary automático |

**Recomendación**: Siempre usar v2.0 con `--all` para evitar duplicados.
