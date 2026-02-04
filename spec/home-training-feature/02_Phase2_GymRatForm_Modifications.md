# Fase 2: Modificaciones al Generador de Rutinas (GymRatForm)

**Documento Tecnico de Implementacion**

| Campo | Valor |
|-------|-------|
| Version | 1.0 |
| Fecha | 2026-02-03 |
| Workflow | `n8n/running_flows/WORKOUT_CREATOR.json` |
| System Prompt | `n8n/system_prompts/RoutineCreation.txt` |
| Asignado a | n8n-agent |
| Revisores | code-reviewer, kiro-coach (validar reglas de entrenamiento) |

---

## 1. Objetivo

Modificar el workflow `WORKOUT_CREATOR` para:

1. **Filtrar ejercicios por equipamiento** cuando el usuario entrena en casa (HOME)
2. **Compensar gaps de ejercicios** en patrones con cobertura limitada para HOME
3. **Mantener compatibilidad total** con usuarios GYM (sin cambios en su flujo)

### 1.1 Contexto de Ejercicios

| Ambiente | Total Ejercicios | Equipamiento Incluido |
|----------|------------------|----------------------|
| GYM | 1,657 | Todos (machine, cable, barbell, dumbbell, kettlebell, bodyweight, smith) |
| HOME | ~818 | bodyweight (197), dumbbell (290), kettlebell (157), barbell (174) |

### 1.2 Gaps Identificados para HOME

| Patron | Ejercicios GYM | Ejercicios HOME | Gap % | Impacto |
|--------|----------------|-----------------|-------|---------|
| pull_v | 95+ | 19 | ~80% | **CRITICO** - Solo bodyweight (12) + dumbbell (4) + kettlebell (2) |
| accessory (Calfs) | 45+ | 19 | ~58% | ALTO - Limitado a calf raises basicos |
| accessory (Quads) | 60+ | ~8 | ~87% | ALTO - Sin leg extension machines |
| accessory (Hamstrings) | 50+ | ~11 | ~78% | ALTO - Sin leg curl machines |
| accessory (Back) | 70+ | ~8 | ~89% | ALTO - Sin cables/maquinas |

---

## 2. Nodos a Modificar

| ID Nodo | Nombre | Tipo | Modificacion |
|---------|--------|------|--------------|
| `6a5bf464-bd71-4132-bc0a-74c696032052` | ProcessUserPreferences | Code | Agregar `parseHomeEquipment()` y extender return |
| `496ccd04-7780-490a-80e4-afff928f43c3` | GetExercisesByPattern | Supabase | Cambiar a Postgres con query dinamico |
| `4024436d-c96c-46cb-9872-b3ec91e323eb` | AI Agent | Agent | Actualizar systemMessage con seccion HOME |

### 2.1 Dependencia de Datos

```
GetUserProfile
    |
    v
ProcessUserPreferences  <-- MODIFICAR: Agregar environment, home_equipment
    |
    v
GetExercisesByPattern   <-- MODIFICAR: Filtrar por equipment si HOME
    |
    v
AI Agent                <-- MODIFICAR: Reglas de compensacion HOME
```

---

## 3. Modificaciones a ProcessUserPreferences (Code Node)

### 3.1 Codigo JavaScript: parseHomeEquipment()

Agregar esta funcion al inicio del nodo `ProcessUserPreferences`:

```javascript
// ============================================
// FUNCION: parseHomeEquipment
// PROPOSITO: Parsear equipamiento disponible en casa
// INPUT: String con equipamiento separado por comas (espanol)
// OUTPUT: Objeto con arrays y flags para queries
// ============================================

function parseHomeEquipment(equipmentStr, environment) {
  // Si es GYM, retornar configuracion que no filtra
  if (!environment || environment.toUpperCase() === 'GYM') {
    return {
      is_home: false,
      equipment_list: [], // Vacio = no filtrar
      home_equipment_sql: null,
      has_pull_bar: true, // Asumimos acceso a todo en gym
      has_barbell: true,
      has_dumbbells: true,
      has_kettlebells: true,
      equipment_tier: 'full'
    };
  }

  // Mapeo espanol -> canonico (valores en BD)
  const equipmentMap = {
    // Mancuernas
    'mancuernas': 'dumbbell',
    'mancuerna': 'dumbbell',
    'dumbbells': 'dumbbell',
    'dumbbell': 'dumbbell',
    'pesas': 'dumbbell',

    // Kettlebells
    'kettlebell': 'kettlebell',
    'kettlebells': 'kettlebell',
    'pesa rusa': 'kettlebell',
    'pesas rusas': 'kettlebell',

    // Barra/Barbell
    'barra': 'barbell',
    'barras': 'barbell',
    'barbell': 'barbell',
    'barra olimpica': 'barbell',
    'barra olímpica': 'barbell',

    // Barra de dominadas
    'barra de dominadas': 'pull_bar',
    'barra dominadas': 'pull_bar',
    'pull up bar': 'pull_bar',
    'pullup bar': 'pull_bar',
    'barra para dominadas': 'pull_bar',

    // Bandas
    'bandas': 'bands',
    'bandas elasticas': 'bands',
    'bandas elásticas': 'bands',
    'ligas': 'bands',
    'resistance bands': 'bands',

    // Banco
    'banco': 'bench',
    'banco plano': 'bench',
    'banco ajustable': 'bench'
  };

  // Parsear string de entrada
  const normalized = (equipmentStr || '')
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, ""); // Remover acentos

  const equipmentSet = new Set(['bodyweight']); // Siempre incluir bodyweight
  const flags = {
    has_pull_bar: false,
    has_barbell: false,
    has_dumbbells: false,
    has_kettlebells: false,
    has_bench: false,
    has_bands: false
  };

  // Buscar matches en el texto
  for (const [spanish, canonical] of Object.entries(equipmentMap)) {
    const normalizedSpanish = spanish
      .normalize("NFD")
      .replace(/[\u0300-\u036f]/g, "");

    if (normalized.includes(normalizedSpanish)) {
      if (canonical === 'pull_bar') {
        flags.has_pull_bar = true;
        // pull_bar habilita ejercicios bodyweight de pull_v
      } else if (canonical === 'bands') {
        flags.has_bands = true;
        // Las bandas se manejan como bodyweight con resistencia
      } else if (canonical === 'bench') {
        flags.has_bench = true;
      } else {
        equipmentSet.add(canonical);
        if (canonical === 'barbell') flags.has_barbell = true;
        if (canonical === 'dumbbell') flags.has_dumbbells = true;
        if (canonical === 'kettlebell') flags.has_kettlebells = true;
      }
    }
  }

  const equipmentList = Array.from(equipmentSet);

  // Generar SQL para filtrado
  // Formato: equipment IN ('bodyweight', 'dumbbell', 'kettlebell')
  const home_equipment_sql = equipmentList
    .map(e => `'${e}'`)
    .join(', ');

  // Determinar tier de equipamiento
  let equipment_tier = 'minimal'; // Solo bodyweight
  if (flags.has_dumbbells || flags.has_kettlebells) {
    equipment_tier = 'basic'; // Peso libre basico
  }
  if (flags.has_barbell) {
    equipment_tier = 'intermediate'; // Tiene barra
  }
  if (flags.has_barbell && flags.has_dumbbells && flags.has_pull_bar) {
    equipment_tier = 'advanced'; // Home gym completo
  }

  return {
    is_home: true,
    equipment_list: equipmentList,
    home_equipment_sql: home_equipment_sql,
    has_pull_bar: flags.has_pull_bar,
    has_barbell: flags.has_barbell,
    has_dumbbells: flags.has_dumbbells,
    has_kettlebells: flags.has_kettlebells,
    has_bench: flags.has_bench,
    has_bands: flags.has_bands,
    equipment_tier: equipment_tier
  };
}
```

### 3.2 Cambios al Return Statement

Modificar el return statement para incluir los nuevos campos:

```javascript
// Obtener environment del perfil (nuevo campo de users_gym_profile)
const environment = profile.training_environment || 'GYM';
const homeEquipment = parseHomeEquipment(profile.home_equipment, environment);

return [{
  json: {
    ...profile,
    processed: {
      // Campos existentes (NO MODIFICAR)
      priority_muscles_en: mapMuscles(profile.priority_muscles),
      disliked_muscles_en: mapMuscles(profile.disliked_exercises),
      experience_tier: getExperienceTier(profile.training_experience),
      volume_modifier: getVolumeModifier(profile.session_duration_mins),
      health: getHealthRestrictions(profile.health_status),

      // === NUEVOS CAMPOS PARA HOME ===
      environment: environment.toUpperCase(),
      home: homeEquipment
    }
  }
}];
```

### 3.3 Estructura Completa del Objeto `processed`

```javascript
{
  "processed": {
    // Campos existentes
    "priority_muscles_en": ["Glutes", "Hamstrings"],
    "disliked_muscles_en": ["Calfs"],
    "experience_tier": "intermediate",
    "volume_modifier": 1.0,
    "health": {
      "has_restrictions": false,
      "avoid_lower_body_impact": false,
      "avoid_upper_body_overhead": false,
      "avoid_spinal_loading": false,
      "special_condition": false
    },

    // Nuevos campos HOME
    "environment": "HOME",
    "home": {
      "is_home": true,
      "equipment_list": ["bodyweight", "dumbbell", "kettlebell"],
      "home_equipment_sql": "'bodyweight', 'dumbbell', 'kettlebell'",
      "has_pull_bar": true,
      "has_barbell": false,
      "has_dumbbells": true,
      "has_kettlebells": true,
      "has_bench": false,
      "has_bands": true,
      "equipment_tier": "basic"
    }
  }
}
```

---

## 4. Modificaciones a GetExercisesByPattern

### 4.1 Configuracion Actual (Supabase Node)

```json
{
  "operation": "get",
  "tableId": "exercises",
  "filters": {
    "conditions": [
      {
        "keyName": "pattern",
        "keyValue": "={{ $json.pattern }}"
      }
    ]
  }
}
```

**Problema**: No soporta filtrado dinamico por equipment.

### 4.2 Solucion: Cambiar a Postgres Node

Reemplazar el nodo Supabase por un nodo Postgres con query dinamico:

**Nuevo Nodo: GetExercisesByPattern (Postgres)**

| Parametro | Valor |
|-----------|-------|
| Operation | Execute Query |
| Query | Ver abajo |
| Credentials | Supabase Memory (postgres) |

### 4.3 Query SQL Dinamico

```sql
SELECT
  exercise_id,
  spanish_name,
  pattern,
  role,
  main_muscle,
  secondary_muscles,
  level,
  link,
  equipment
FROM exercises
WHERE pattern = '{{ $json.pattern }}'
{% if $items('ProcessUserPreferences')[0].json.processed.home.is_home %}
  AND equipment IN ({{ $items('ProcessUserPreferences')[0].json.processed.home.home_equipment_sql }})
{% endif %}
ORDER BY
  CASE
    WHEN level = '{{ $items('ProcessUserPreferences')[0].json.fitness_level }}' THEN 0
    ELSE 1
  END,
  role,
  spanish_name;
```

### 4.4 Query Alternativa (Sin Jinja - Expression Mode)

Si n8n no soporta Jinja en el nodo Postgres, usar Expression:

```sql
={{
  const isHome = $items('ProcessUserPreferences')[0].json.processed.home.is_home;
  const equipmentSql = $items('ProcessUserPreferences')[0].json.processed.home.home_equipment_sql;
  const pattern = $json.pattern;
  const level = $items('ProcessUserPreferences')[0].json.fitness_level;

  let query = `
    SELECT
      exercise_id,
      spanish_name,
      pattern,
      role,
      main_muscle,
      secondary_muscles,
      level,
      link,
      equipment
    FROM exercises
    WHERE pattern = '${pattern}'
  `;

  if (isHome && equipmentSql) {
    query += ` AND equipment IN (${equipmentSql})`;
  }

  query += `
    ORDER BY
      CASE WHEN level = '${level}' THEN 0 ELSE 1 END,
      role,
      spanish_name
  `;

  return query;
}}
```

### 4.5 Logica de Filtrado Resumida

| Ambiente | Filtro Equipment | Resultado |
|----------|------------------|-----------|
| GYM | Ninguno | Todos los ejercicios del patron |
| HOME | `equipment IN (home_equipment_sql)` | Solo ejercicios con equipo disponible |
| HOME sin equipo especificado | `equipment IN ('bodyweight')` | Solo ejercicios con peso corporal |

---

## 5. Modificaciones al System Prompt (RoutineCreation.txt)

### 5.1 Nueva Seccion: AMBIENTE DE ENTRENAMIENTO

Agregar despues de la seccion **Modificador de volumen** y antes de **PROTOCOLO DE SELECCION**:

```text
---

## AMBIENTE DE ENTRENAMIENTO

**Ambiente:** {{ $('ProcessUserPreferences').item.json.processed.environment }}
{% if $('ProcessUserPreferences').item.json.processed.home.is_home %}

### CONFIGURACION HOME

**Equipamiento disponible:** {{ $('ProcessUserPreferences').item.json.processed.home.equipment_list.join(', ') }}
**Tier de equipamiento:** {{ $('ProcessUserPreferences').item.json.processed.home.equipment_tier }}

**Flags de equipamiento:**
- Barra de dominadas: {{ $('ProcessUserPreferences').item.json.processed.home.has_pull_bar ? 'SI' : 'NO' }}
- Barbell: {{ $('ProcessUserPreferences').item.json.processed.home.has_barbell ? 'SI' : 'NO' }}
- Mancuernas: {{ $('ProcessUserPreferences').item.json.processed.home.has_dumbbells ? 'SI' : 'NO' }}
- Kettlebells: {{ $('ProcessUserPreferences').item.json.processed.home.has_kettlebells ? 'SI' : 'NO' }}
- Banco: {{ $('ProcessUserPreferences').item.json.processed.home.has_bench ? 'SI' : 'NO' }}

### REGLAS ESPECIALES HOME

1. **Solo usar ejercicios de AVAILABLE_EXERCISES** - Ya estan pre-filtrados por equipamiento
2. **Compensacion de gaps obligatoria** - Ver tabla de compensacion abajo
3. **Priorizar variaciones** - Usar diferentes angulos/posiciones del mismo equipo
4. **Creatividad con bodyweight** - Explotar ejercicios isometricos, excentricos, unilaterales

### TABLA DE COMPENSACION DE GAPS

| Gap Detectado | Patron Original | Compensacion |
|---------------|-----------------|--------------|
| pull_v limitado (sin barra dominadas) | pull_v | +2 series pull_h, +1 ejercicio arm (back) |
| pull_v limitado (con barra dominadas) | pull_v | Priorizar chin-ups, pull-ups, variaciones |
| Quads limitado (sin leg extension) | accessory (Quads) | +1 ejercicio squat unilateral, +1 lunge |
| Hamstrings limitado (sin leg curl) | accessory (Hamstrings) | +2 series hinge, +1 ejercicio lunge posterior |
| Calfs limitado | accessory (Calfs) | Incluir calf raises en cada dia de pierna |
| Back limitado (sin cables) | accessory (Back) | +1 ejercicio pull_h, priorizar rows con dumbbell |

### REGLAS DE VOLUMEN HOME

- **Si equipment_tier = 'minimal'**: Reducir 1 serie por ejercicio, aumentar reps
- **Si equipment_tier = 'basic'**: Mantener volumen, distribuir en mas ejercicios
- **Si equipment_tier = 'intermediate/advanced'**: Volumen normal

{% else %}

### CONFIGURACION GYM

Acceso completo a todo el equipamiento de gimnasio.
No aplican restricciones de equipamiento.

{% endif %}
```

### 5.2 Modificar Seccion ADAPTACION POR EXPERIENCIA

Actualizar la regla existente para considerar HOME:

```text
4. **ADAPTACION POR EXPERIENCIA**:
   - `beginner`:
     {% if $('ProcessUserPreferences').item.json.processed.home.is_home %}
     - HOME: Priorizar bodyweight y mancuernas ligeras
     - Enfoque en forma y control antes que peso
     {% else %}
     - GYM: Preferir `equipment` = "machine" o "bodyweight", evitar tecnicas complejas
     {% endif %}
   - `intermediate`: Balance entre ejercicios
   - `advanced`:
     {% if $('ProcessUserPreferences').item.json.processed.home.is_home %}
     - HOME: Maximizar intensidad con equipo disponible
     - Usar tecnicas avanzadas: tempo lento, isometricos, drop sets manuales
     {% else %}
     - GYM: Priorizar compuestos con barbell/dumbbell
     {% endif %}
```

### 5.3 Agregar Regla de Oro para HOME

En la seccion **REGLAS DE ORO**, agregar:

```text
- OBLIGATORIO compensar gaps si el ambiente es HOME (ver tabla de compensacion).
- PROHIBIDO seleccionar ejercicios que requieran equipamiento no disponible.
```

---

## 6. Mapeo de Equipamiento

### 6.1 Tabla Completa: Termino Usuario -> Valor BD

| Termino Usuario (Espanol) | Valor Canonico BD | Notas |
|---------------------------|-------------------|-------|
| mancuernas | dumbbell | Plural e singular |
| mancuerna | dumbbell | |
| pesas | dumbbell | Generico, asumimos mancuernas |
| dumbbells | dumbbell | Ingles |
| kettlebell | kettlebell | |
| kettlebells | kettlebell | Plural |
| pesa rusa | kettlebell | |
| pesas rusas | kettlebell | Plural |
| barra | barbell | |
| barras | barbell | |
| barbell | barbell | Ingles |
| barra olimpica | barbell | |
| barra de dominadas | pull_bar | Flag especial, no filtra BD |
| pull up bar | pull_bar | Ingles |
| bandas | bands | Flag especial |
| bandas elasticas | bands | |
| ligas | bands | Coloquial |
| banco | bench | Flag especial |
| banco plano | bench | |
| banco ajustable | bench | |

### 6.2 Valores de Equipment en BD

| Valor BD | Ejercicios | Disponible HOME |
|----------|------------|-----------------|
| machine | 656 | NO |
| dumbbell | 290 | SI (si usuario tiene) |
| bodyweight | 197 | SIEMPRE |
| barbell | 174 | SI (si usuario tiene) |
| cable | 161 | NO |
| kettlebell | 157 | SI (si usuario tiene) |
| smith | 2 | NO |

---

## 7. Tareas Accionables

### 7.1 Checklist para n8n-agent

- [ ] **TASK-2.1**: Actualizar nodo `ProcessUserPreferences` (ID: `6a5bf464...`)
  - [ ] Agregar funcion `parseHomeEquipment()` al inicio del codigo
  - [ ] Modificar return statement para incluir campos `environment` y `home`
  - [ ] Probar con usuario HOME mock

- [ ] **TASK-2.2**: Reemplazar nodo `GetExercisesByPattern` (ID: `496ccd04...`)
  - [ ] Crear nuevo nodo Postgres
  - [ ] Implementar query dinamico con filtro condicional
  - [ ] Conectar a las mismas entradas/salidas
  - [ ] Probar con patron `pull_v` para HOME

- [ ] **TASK-2.3**: Actualizar system prompt del AI Agent
  - [ ] Agregar seccion AMBIENTE DE ENTRENAMIENTO
  - [ ] Agregar tabla de compensacion de gaps
  - [ ] Agregar reglas de volumen HOME
  - [ ] Actualizar REGLAS DE ORO

- [ ] **TASK-2.4**: Actualizar archivo `RoutineCreation.txt`
  - [ ] Sincronizar cambios del system prompt
  - [ ] Documentar variables Jinja usadas

- [ ] **TASK-2.5**: Testing
  - [ ] Test E2E: Usuario HOME con mancuernas + kettlebell
  - [ ] Test E2E: Usuario HOME solo bodyweight
  - [ ] Test E2E: Usuario GYM (regresion - sin cambios)
  - [ ] Validar compensacion de gaps en rutina generada

---

## 8. Criterios de Aceptacion

### 8.1 Funcionales

| # | Criterio | Metodo de Validacion |
|---|----------|---------------------|
| AC-2.1 | Usuario HOME recibe solo ejercicios con su equipamiento | Query BD post-generacion |
| AC-2.2 | Usuario GYM recibe todos los ejercicios (sin filtro) | Query BD post-generacion |
| AC-2.3 | Gaps de pull_v compensados con pull_h cuando no hay barra dominadas | Revisar rutina generada |
| AC-2.4 | ProcessUserPreferences genera `home_equipment_sql` correcto | Log del nodo |
| AC-2.5 | No hay ejercicios con `equipment='machine'` o `equipment='cable'` para HOME | Query BD |

### 8.2 No Funcionales

| # | Criterio | Umbral |
|---|----------|--------|
| AC-2.6 | Tiempo de generacion de rutina | < 60 segundos |
| AC-2.7 | Sin errores de parseo en equipment | 0 errores en 10 ejecuciones |
| AC-2.8 | Compatibilidad con usuarios existentes (GYM) | 100% funcional |

### 8.3 Ejemplo de Validacion SQL

```sql
-- Verificar que usuario HOME no tiene ejercicios de maquina
SELECT w.*, e.equipment
FROM workouts w
JOIN exercises e USING(exercise_id)
WHERE w.user_id = '<USER_ID_HOME>'
AND e.equipment IN ('machine', 'cable', 'smith');

-- Resultado esperado: 0 filas
```

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigacion |
|--------|--------------|---------|------------|
| Query dinamico falla por sintaxis | Media | Alto | Probar en Supabase SQL Editor primero |
| Jinja no soportado en Postgres node | Alta | Medio | Usar Expression mode con JS |
| Compensacion de gaps genera rutina muy larga | Media | Medio | ValidateWorkoutDuration ya existe |
| Usuario no especifica equipamiento | Baja | Bajo | Default a bodyweight only |

---

## 10. Dependencias

### 10.1 Prerequisitos (Fase 1)

- [ ] Campo `training_environment` agregado a `users_gym_profile`
- [ ] Campo `home_equipment` agregado a `users_gym_profile`
- [ ] KYC actualizado para capturar estos campos

### 10.2 Datos de Prueba

Crear usuario de prueba en `users_gym_profile`:

```sql
INSERT INTO users_gym_profile (
    submission_date, whatsapp_id, full_name, email, age, biological_sex,
    height_cm, weight_kg, primary_goal, secondary_goal, training_experience,
    current_frequency, fitness_level, health_status, days_available,
    session_duration_mins, preferred_schedule, training_style, priority_muscles,
    disliked_exercises, cardio_type, cardio_frequency,
    training_environment, home_equipment  -- Nuevos campos
) VALUES (
    NOW(), 570000000010, 'Test_HomeUser', 'home@test.com', 28, 'M',
    178, 80, 'Ganar masa muscular', 'Ninguna', '1 a 3 anos',
    '3-4 dias por semana', 'Intermedio', 'A', 4,
    '60-75 minutos', 'Manana', 'Mixto', 'Pecho, Espalda',
    'Pantorrillas', 'No', '0',
    'HOME', 'mancuernas, kettlebell, barra de dominadas'
);
```

---

## Anexo A: Diagrama de Flujo Modificado

```
                                    START
                                      |
                                      v
                              +---------------+
                              | GetUserProfile |
                              +---------------+
                                      |
                                      v
                    +----------------------------------+
                    | ProcessUserPreferences           |
                    | - mapMuscles()                   |
                    | - getHealthRestrictions()        |
                    | + parseHomeEquipment() [NUEVO]   |
                    +----------------------------------+
                                      |
                    environment = HOME?
                           /          \
                         YES          NO
                          |            |
                          v            v
              +------------------+  +------------------+
              | equipment IN     |  | Sin filtro       |
              | (home_equipment) |  | equipment        |
              +------------------+  +------------------+
                          \          /
                           \        /
                            v      v
                    +-------------------+
                    | GetExercisesByPattern |
                    | (Postgres Node)       |
                    +-------------------+
                              |
                              v
                    +-------------------+
                    | AI Agent           |
                    | + Reglas HOME      |
                    | + Compensacion gaps|
                    +-------------------+
                              |
                              v
                         [Continua...]
```

---

## Anexo B: Cobertura de Ejercicios por Patron (HOME)

Datos actuales de la BD para referencia:

| Pattern | bodyweight | dumbbell | kettlebell | barbell | TOTAL HOME |
|---------|------------|----------|------------|---------|------------|
| accessory | 67 | 77 | 59 | 65 | 268 |
| arm | 16 | 73 | 16 | 14 | 119 |
| core | 29 | 12 | 2 | 3 | 46 |
| hinge | 1 | 8 | 9 | 27 | 45 |
| lunge | 18 | 19 | 29 | 19 | 85 |
| pull_h | 6 | 19 | 12 | 10 | 47 |
| **pull_v** | **12** | **4** | **2** | **1** | **19** |
| push_h | 21 | 36 | 15 | 9 | 81 |
| push_v | 8 | 10 | 0 | 0 | 18 |
| squat | 19 | 32 | 13 | 26 | 90 |

**Patrones criticos para HOME**: `pull_v` (19 ejercicios totales, muy dependiente de barra de dominadas)
