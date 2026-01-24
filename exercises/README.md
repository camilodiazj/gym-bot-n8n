# Importación de Ejercicios desde MuscleWiki

Este documento describe el proceso para importar ejercicios desde archivos JSON de MuscleWiki a Supabase.

> **Nota**: Usar el script Python con el cliente de Supabase directamente, NO el MCP de Supabase. El script es más eficiente para operaciones batch y consume menos tokens.

---

## Uso del Script

### Comandos Básicos

```bash
cd /Users/camilodiazjaimes/Documents/GymBot/exercises

# Preview sin escribir (dry-run)
python3 transform_exercises.py {muscle}.json --dry-run

# Generar CSV
python3 transform_exercises.py {muscle}.json

# Insertar directamente a Supabase
python3 transform_exercises.py {muscle}.json --insert

# Procesar todos los JSON en el directorio
python3 transform_exercises.py --all --insert

# Sin LLM (usa 'accessory' como fallback para patterns no detectados)
python3 transform_exercises.py {muscle}.json --no-llm --insert
```

> Reemplazar `{muscle}` con el nombre del archivo JSON (ej: `triceps`, `biceps`, `chest`)

### Opciones

| Flag | Descripción |
|------|-------------|
| `--dry-run` | Preview sin escribir archivos |
| `--insert` | Insertar directamente a Supabase |
| `--all` | Procesar todos los `*.json` del directorio |
| `--no-llm` | Deshabilitar clasificación con LLM |
| `--output FILE` | Especificar nombre del CSV de salida |
| `--openai-key KEY` | API key de OpenAI (alternativa a env var) |

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
| `exercise_id` | TEXT PK | Formato: `ex_{muscle}_{index:03d}` |
| `name` | TEXT | Nombre en inglés |
| `spanish_name` | TEXT | Nombre en español |
| `main_muscle` | TEXT FK | Músculo principal (inglés) |
| `Músculo Principal` | TEXT | Músculo principal (español) |
| `secondary_muscles` | TEXT[] | Array de músculos secundarios (español) |
| `pattern` | TEXT FK | Patrón de movimiento |
| `role` | TEXT | compound, isolation, core |
| `equipment` | TEXT | dumbbell, barbell, machine, cable, bodyweight |
| `level` | VARCHAR FK | Principiante, Intermedio, Avanzado |
| `unilateral` | TEXT | Yes/No |
| `link` | TEXT | URL de MuscleWiki |

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
2. Guardar en `exercises/{muscle}.json`
3. Ejecutar:
   ```bash
   python3 transform_exercises.py {muscle}.json --insert
   ```

El script automáticamente:
- Usa el nombre del archivo como prefijo de ID (`ex_{muscle}_001`)
- Extrae músculos principales y secundarios
- Clasifica patterns con keywords + LLM
- Inserta/actualiza en Supabase (upsert por `exercise_id`)
