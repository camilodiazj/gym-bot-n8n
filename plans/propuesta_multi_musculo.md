# Propuesta: Soporte Multi-Músculo para Ejercicios

## Objetivo
1. Modificar la tabla `exercises` para soportar múltiples músculos por ejercicio
2. Actualizar el script Python para extraer todos los músculos del JSON
3. Habilitar personalización de rutinas basada en `priority_muscles` del usuario

## Contexto
- `users_gym_profile` tiene `priority_muscles` y `disliked_exercises`
- El bot puede usar esta info para personalizar rutinas
- Necesitamos saber TODOS los músculos que trabaja cada ejercicio

---

## Parte 1: Migración de Base de Datos

### Nueva columna en `exercises`
```sql
ALTER TABLE exercises
ADD COLUMN secondary_muscles TEXT[] DEFAULT '{}';

COMMENT ON COLUMN exercises.secondary_muscles IS
'Array de músculos secundarios trabajados (en español). Permite queries como: WHERE ''Tríceps'' = ANY(secondary_muscles)';
```

**Ventajas de TEXT[]:**
- Fácil para el agente AI: `SELECT * FROM exercises WHERE 'Tríceps' = ANY(secondary_muscles) OR main_muscle = 'Triceps'`
- No requiere JOINs complejos
- Supabase soporta arrays nativamente

---

## Parte 2: Lógica de Extracción de Músculos

### Estructura del JSON de MuscleWiki

```json
"muscles": [
  {"name": "Tríceps", "name_en_us": "Triceps", "level": 0, "tree_id": 11},
  {"name": "Cabeza larga del tríceps", "level": 1, "tree_id": 11},
  {"name": "Hombros", "name_en_us": "Shoulders", "level": 0, "tree_id": 10}
]
```

### Reglas de Extracción

| Nivel | Descripción | Destino |
|-------|-------------|---------|
| `level: 0` (primero) | Músculo principal | `main_muscle` + `Músculo Principal` |
| `level: 1+` | Sub-músculos (cabeza larga, etc.) | Se ignoran |
| `level: 0` (otros) | Músculos secundarios | `secondary_muscles[]` |

**Resultado del ejemplo:**
- `main_muscle` = `"Triceps"`
- `Músculo Principal` = `"Tríceps"`
- `secondary_muscles` = `["Hombros"]`

---

## Parte 3: Integración con GymRatForm

### Flujo Actual (sin `secondary_muscles`)

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ GetUserProfile  │───▶│ Get_Day_Requirements │───▶│ Loop Over Items │
│                 │    │                      │    │                 │
│ priority_muscles│    │ pattern: "arm"       │    │ Por cada patrón │
│ = "Tríceps"     │    │ min_sets: 4          │    └────────┬────────┘
└─────────────────┘    └──────────────────────┘             │
                                                            ▼
                       ┌──────────────────────────────────────────────┐
                       │         GetExercisesByPattern                │
                       │  SELECT * FROM exercises WHERE pattern='arm' │
                       │                                              │
                       │  Resultado: Bíceps + Tríceps mezclados       │
                       │  (Sin saber cuál trabaja qué músculo)        │
                       └──────────────────────────────────────────────┘
                                           │
                                           ▼
                       ┌──────────────────────────────────────────────┐
                       │              AI Agent                        │
                       │  Recibe:                                     │
                       │  - priority_muscles: "Tríceps"               │
                       │  - AVAILABLE_EXERCISES: [...] (sin músculo)  │
                       │                                              │
                       │  ❌ No puede priorizar inteligentemente      │
                       └──────────────────────────────────────────────┘
```

### Flujo Mejorado (con `secondary_muscles`)

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ GetUserProfile  │───▶│ Get_Day_Requirements │───▶│ Loop Over Items │
│                 │    │                      │    │                 │
│ priority_muscles│    │ pattern: "arm"       │    │ Por cada patrón │
│ = "Tríceps"     │    │ min_sets: 4          │    └────────┬────────┘
└─────────────────┘    └──────────────────────┘             │
                                                            ▼
                       ┌──────────────────────────────────────────────┐
                       │         GetExercisesByPattern (mejorado)     │
                       │  SELECT *, main_muscle, secondary_muscles    │
                       │  FROM exercises WHERE pattern='arm'          │
                       │                                              │
                       │  Resultado incluye info de músculos:         │
                       │  - "Triceps Pushdown" → main: Triceps        │
                       │  - "Dips" → main: Triceps, sec: [Hombros]    │
                       └──────────────────────────────────────────────┘
                                           │
                                           ▼
                       ┌──────────────────────────────────────────────┐
                       │              AI Agent (mejorado)             │
                       │  Recibe:                                     │
                       │  - priority_muscles: "Tríceps"               │
                       │  - AVAILABLE_EXERCISES con main_muscle y     │
                       │    secondary_muscles                         │
                       │                                              │
                       │  ✅ Puede priorizar ejercicios de Tríceps    │
                       │  ✅ Evita ejercicios de músculos no deseados │
                       └──────────────────────────────────────────────┘
```

---

## Parte 4: Cambios Requeridos en el AI Agent

### Nuevas Reglas para el System Prompt

Agregar al system prompt del nodo "AI Agent" en `GymRatForm Supabase.json`:

```
- REGLA DE MÚSCULOS PRIORITARIOS: Si el usuario tiene "priority_muscles" definidos,
  prioriza ejercicios donde ese músculo sea el "main_muscle" o esté en "secondary_muscles".

- REGLA DE MÚSCULOS EXCLUIDOS: Si hay "disliked_muscles" definidos, evita ejercicios
  donde ese músculo sea principal o secundario.
```

---

## Parte 5: Ejemplo Práctico

### Escenario

**Usuario:**
- `priority_muscles`: "Tríceps"
- `pattern` requerido: "arm" (4 sets)

**AVAILABLE_EXERCISES recibidos por el agente:**
```json
[
  {"exercise_id": "ex_009", "name": "Biceps Curl", "main_muscle": "Biceps", "secondary_muscles": []},
  {"exercise_id": "ex_015", "name": "Triceps Pushdown", "main_muscle": "Triceps", "secondary_muscles": []},
  {"exercise_id": "ex_042", "name": "Overhead Extension", "main_muscle": "Triceps", "secondary_muscles": ["Hombros"]},
  {"exercise_id": "ex_044", "name": "Dips", "main_muscle": "Triceps", "secondary_muscles": ["Pecho", "Hombros"]}
]
```

### Decisión del Agente

| Paso | Ejercicio Seleccionado | Razón |
|------|------------------------|-------|
| 1 | Triceps Pushdown | `main_muscle = Triceps` (prioridad del usuario) |
| 2 | Overhead Extension | Trabaja Triceps + Hombros (variedad) |
| ❌ | Biceps Curl | `main_muscle = Biceps` (no es prioridad) |

---

## Parte 6: Query de Ejemplo para el Agente

```sql
-- Buscar ejercicios que trabajen un músculo prioritario del usuario
SELECT e.spanish_name, e.main_muscle, e.secondary_muscles
FROM exercises e
WHERE e.main_muscle = 'Triceps'
   OR 'Tríceps' = ANY(e.secondary_muscles)
ORDER BY
  CASE WHEN e.main_muscle = 'Triceps' THEN 0 ELSE 1 END;  -- Priorizar donde es principal
```

---

## Resumen de Cambios

| Componente | Cambio |
|------------|--------|
| Tabla `exercises` | Agregar columna `secondary_muscles TEXT[]` |
| Script Python | Actualizar para extraer músculos secundarios |
| AI Agent (GymRatForm) | Agregar reglas de priorización por músculo |
| GetExercisesByPattern | Ya incluye todos los campos (no requiere cambios) |

---

## Parte 7: Clasificación Inteligente de Patterns

### Problema

Actualmente el script tiene `pattern` quemado como `"arm"`. Esto no escala cuando importemos ejercicios de otros grupos musculares (pecho, espalda, piernas, etc.).

### Estrategia Recomendada: Híbrida (Keywords + LLM)

#### Paso 1: Clasificación por Keywords (Rápido, 80-90% de casos)

```python
PATTERN_KEYWORDS = {
    'push_h': ['bench press', 'press banca', 'push-up', 'flexion', 'chest press'],
    'push_v': ['overhead press', 'shoulder press', 'military press', 'dip', 'fondo'],
    'pull_h': ['row', 'remo', 'cable row', 'seated row'],
    'pull_v': ['pulldown', 'pull-up', 'chin-up', 'dominada', 'lat pulldown'],
    'squat': ['squat', 'sentadilla', 'leg press', 'prensa', 'hack squat'],
    'hinge': ['deadlift', 'peso muerto', 'hip thrust', 'good morning', 'romanian'],
    'lunge': ['lunge', 'zancada', 'split squat', 'step-up'],
    'arm': ['curl', 'extension', 'pushdown', 'kickback', 'skull crusher'],
    'core': ['plank', 'crunch', 'abdominal', 'sit-up', 'leg raise'],
    'carry': ['carry', 'farmer', 'walk', 'caminata'],
    'rotation': ['rotation', 'twist', 'woodchop', 'pallof'],
}

def detect_pattern_by_keywords(exercise_name: str, url_slug: str) -> str | None:
    """Detecta el pattern basado en keywords. Retorna None si no hay match."""
    text = (exercise_name + ' ' + url_slug).lower()
    for pattern, keywords in PATTERN_KEYWORDS.items():
        if any(kw in text for kw in keywords):
            return pattern
    return None  # No match, necesita LLM
```

#### Paso 2: Clasificación por LLM (Casos ambiguos, 10-20%)

Para ejercicios que no matchean keywords, usar un LLM:

```python
def classify_pattern_with_llm(exercises_without_pattern: list[dict]) -> list[dict]:
    """Clasifica ejercicios usando LLM en batch."""

    prompt = """Clasifica cada ejercicio en UNO de estos patterns de movimiento:

PATTERNS DISPONIBLES:
- squat: Dominante de rodilla (sentadillas, prensa, extensiones de cuádriceps)
- hinge: Dominante de cadera (peso muerto, hip thrust, curl femoral)
- lunge: Zancadas y movimientos unilaterales de pierna
- push_h: Empuje horizontal (press banca, flexiones, aperturas)
- push_v: Empuje vertical (press militar, elevaciones laterales, fondos)
- pull_h: Tracción horizontal (remos, face pulls)
- pull_v: Tracción vertical (dominadas, jalones, pullovers)
- arm: Aislamiento de brazos (curls, extensiones de tríceps, kickbacks)
- core: Estabilidad abdominal (planchas, crunches, leg raises)
- accessory: Ejercicios complementarios que no encajan en otros patterns

EJERCICIOS A CLASIFICAR:
{exercises_json}

Responde en formato JSON:
[
  {{"exercise_name": "...", "pattern": "..."}},
  ...
]
"""

    # Llamar al LLM (OpenAI, Gemini, etc.)
    response = llm.complete(prompt.format(exercises_json=json.dumps(exercises_without_pattern)))
    return json.loads(response)
```

#### Paso 3: Pipeline Completo

```python
def assign_patterns(exercises: list[dict]) -> list[dict]:
    """Asigna patterns a todos los ejercicios usando estrategia híbrida."""

    classified = []
    needs_llm = []

    # Paso 1: Intentar clasificar por keywords
    for ex in exercises:
        pattern = detect_pattern_by_keywords(ex['name'], ex['url_slug'])
        if pattern:
            ex['pattern'] = pattern
            classified.append(ex)
        else:
            needs_llm.append(ex)

    print(f"Clasificados por keywords: {len(classified)}")
    print(f"Requieren LLM: {len(needs_llm)}")

    # Paso 2: Clasificar el resto con LLM (en batch para eficiencia)
    if needs_llm:
        llm_results = classify_pattern_with_llm(needs_llm)
        for ex, result in zip(needs_llm, llm_results):
            ex['pattern'] = result['pattern']
            classified.append(ex)

    return classified
```

### Ventajas de la Estrategia Híbrida

| Aspecto | Keywords | LLM | Híbrido |
|---------|----------|-----|---------|
| Velocidad | ✅ Instantáneo | ❌ Latencia API | ✅ Rápido (90% sin API) |
| Costo | ✅ Gratis | ❌ $$ por token | ✅ Bajo (solo 10% usa API) |
| Precisión | ⚠️ 80-90% | ✅ 95%+ | ✅ 95%+ |
| Mantenimiento | ⚠️ Actualizar keywords | ✅ Auto-adapta | ✅ Mínimo |

### Flujo de Importación Propuesto

```
┌─────────────────┐
│  triceps.json   │
│  (197 ejercicios)│
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────┐
│  Paso 1: Extraer datos del JSON    │
│  - name, muscles, equipment, level │
└────────┬────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│  Paso 2: Clasificar por Keywords   │
│  ~170 ejercicios clasificados      │
└────────┬────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│  Paso 3: LLM para casos ambiguos   │
│  ~27 ejercicios restantes          │
└────────┬────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│  Paso 4: Generar CSV               │
│  197 ejercicios con pattern        │
└────────┬────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│  Paso 5: Importar a Supabase       │
└─────────────────────────────────────┘
```

### Implementación en n8n (Alternativa)

Si prefieres no usar Python para el LLM, puedes crear un workflow en n8n:

1. **Nodo Code**: Lee el JSON y clasifica por keywords
2. **Nodo IF**: Separa ejercicios clasificados vs pendientes
3. **Nodo AI Agent**: Clasifica los pendientes con OpenAI/Gemini
4. **Nodo Merge**: Combina resultados
5. **Nodo Supabase**: Inserta en la tabla

---

## Parte 8: Clasificación de Role (compound/isolation/core)

### Roles Existentes en Supabase

| Role | Cantidad | Descripción | Uso en `set_profiles` |
|------|----------|-------------|----------------------|
| `compound` | 21 | Multiarticulares (2+ articulaciones) | Más series, menos reps, más descanso |
| `isolation` | 17 | Aislamiento (1 articulación) | Menos series, más reps, menos descanso |
| `core` | 3 | Abdomen/estabilidad | Parámetros específicos |

### Importancia del Role

El `role` determina los parámetros de carga en `set_profiles`:

```sql
SELECT role, sets, reps, rir, rest_sec
FROM set_profiles
WHERE goal = 'Hipertrofia' AND level = 'Intermedio' AND week = 1;
```

| Role | Sets | Reps | RIR | Rest |
|------|------|------|-----|------|
| compound | 3-4 | 6-10 | 2-3 | 120s |
| isolation | 2-3 | 10-15 | 1-2 | 60s |
| core | 2-3 | 12-20 | 1 | 45s |

### Estrategia de Clasificación: Pattern → Role

La forma más simple es derivar el `role` del `pattern`:

```python
PATTERN_TO_ROLE = {
    # Compound (multiarticulares)
    'squat': 'compound',
    'hinge': 'compound',
    'lunge': 'compound',
    'push_h': 'compound',
    'push_v': 'compound',
    'pull_h': 'compound',
    'pull_v': 'compound',

    # Isolation (aislamiento)
    'arm': 'isolation',
    'accessory': 'isolation',

    # Core
    'core': 'core',

    # Casos especiales
    'carry': 'compound',
    'rotation': 'core',
}

def get_role_from_pattern(pattern: str) -> str:
    """Deriva el role basado en el pattern."""
    return PATTERN_TO_ROLE.get(pattern, 'isolation')
```

### Excepciones a Considerar

Algunos ejercicios pueden tener un pattern pero diferente role:

| Ejercicio | Pattern | Role Esperado | Razón |
|-----------|---------|---------------|-------|
| Leg Extension | `squat` | `isolation` | Solo mueve rodilla |
| Leg Curl | `hinge` | `isolation` | Solo mueve rodilla |
| Lateral Raise | `push_v` | `isolation` | Solo mueve hombro |
| Face Pull | `pull_h` | `isolation` | Ejercicio correctivo |

### Solución: Keywords para Excepciones

```python
ISOLATION_OVERRIDE_KEYWORDS = [
    'extension', 'curl', 'raise', 'lateral', 'fly', 'apertura',
    'kickback', 'pushdown', 'face pull', 'reverse fly'
]

def get_role(pattern: str, exercise_name: str) -> str:
    """Determina el role considerando excepciones."""
    name_lower = exercise_name.lower()

    # Core siempre es core
    if pattern == 'core':
        return 'core'

    # Verificar si es una excepción de aislamiento
    if any(kw in name_lower for kw in ISOLATION_OVERRIDE_KEYWORDS):
        return 'isolation'

    # Usar mapeo por defecto
    return PATTERN_TO_ROLE.get(pattern, 'isolation')
```

### Pipeline Actualizado (Pattern + Role)

```python
def classify_exercise(exercise: dict) -> dict:
    """Clasifica pattern y role de un ejercicio."""
    name = exercise['name']
    url_slug = exercise.get('url_slug', '')

    # 1. Clasificar pattern
    pattern = detect_pattern_by_keywords(name, url_slug)
    if not pattern:
        pattern = 'accessory'  # O usar LLM

    # 2. Derivar role del pattern (con excepciones)
    role = get_role(pattern, name)

    exercise['pattern'] = pattern
    exercise['role'] = role

    return exercise
```

### Incluir Role en Prompt del LLM

Si usas LLM para casos ambiguos, pedir ambos valores:

```python
prompt = """Clasifica cada ejercicio:

PATTERNS: squat, hinge, lunge, push_h, push_v, pull_h, pull_v, arm, core, accessory
ROLES: compound (multiarticular), isolation (1 articulación), core (abdomen)

EJERCICIOS:
{exercises_json}

Responde en JSON:
[
  {{"exercise_name": "...", "pattern": "...", "role": "..."}},
  ...
]
"""
```

---

## Notas Técnicas

- `main_muscle` usa el nombre en **inglés** (consistencia con datos existentes)
- `Músculo Principal` usa el nombre en **español** (mostrar al usuario)
- `secondary_muscles` es un array PostgreSQL en **español** (para queries del agente)
- Solo se incluyen músculos nivel 0 (grupos principales), no sub-músculos como "Cabeza larga"
- La clasificación de patterns y roles se hace **una sola vez** al importar, no en runtime
- El `role` es crítico porque determina los parámetros de carga (sets, reps, rir, rest)
