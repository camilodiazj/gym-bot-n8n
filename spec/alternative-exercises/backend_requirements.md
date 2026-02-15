# Ejercicios Alternativos - Requerimientos del Backend

## Estado Actual

El frontend (`workout-tracker/`) ya tiene la funcionalidad completa de ejercicios alternativos con flip card funcionando con datos demo/mock. Este documento define **que** debe entregar el backend para soportar esta funcionalidad con datos reales.

---

## 1. Contrato de API - Respuesta de `GET /api/v1/workouts/today`

### Respuesta Actual (sin alternativas)

```json
{
  "success": true,
  "data": {
    "session_id": "uuid",
    "session_name": "Upper A",
    "week": 1,
    "day_name": "Upper A",
    "exercises": [
      {
        "id": "workout-uuid",
        "name": "Sentadilla con barra",
        "badgeColor": "#374151",
        "rir": "3",
        "restSeconds": 120,
        "videoLink": "https://...",
        "sets": [
          { "id": "workout-uuid-1", "setNumber": 1, "reps": 8, "kg": "-", "completed": false },
          { "id": "workout-uuid-2", "setNumber": 2, "reps": 9, "kg": "-", "completed": false },
          { "id": "workout-uuid-3", "setNumber": 3, "reps": 10, "kg": "-", "completed": false }
        ],
        "tips": [],
        "steps": []
      }
    ]
  }
}
```

### Respuesta Requerida (con alternativas)

Cada objeto `exercise` debe incluir un campo opcional `alternativeExercises` cuando existan alternativas asignadas. El frontend ya consume esta estructura exacta:

```json
{
  "success": true,
  "data": {
    "session_id": "uuid",
    "session_name": "Upper A",
    "week": 1,
    "day_name": "Upper A",
    "exercises": [
      {
        "id": "workout-uuid",
        "name": "Sentadilla con barra",
        "badgeColor": "#374151",
        "rir": "3",
        "restSeconds": 120,
        "videoLink": "https://...",
        "sets": [
          { "id": "workout-uuid-1", "setNumber": 1, "reps": 8, "kg": "40", "completed": false },
          { "id": "workout-uuid-2", "setNumber": 2, "reps": 9, "kg": "40", "completed": false },
          { "id": "workout-uuid-3", "setNumber": 3, "reps": 10, "kg": "40", "completed": false }
        ],
        "tips": [],
        "steps": [],
        "alternativeExercises": [
          {
            "name": "Goblet Squat",
            "rir": "3",
            "restSeconds": 120,
            "videoLink": "https://...",
            "sets": [
              { "id": "alt-uuid-1", "setNumber": 1, "reps": 10, "kg": "-", "completed": false },
              { "id": "alt-uuid-2", "setNumber": 2, "reps": 10, "kg": "-", "completed": false },
              { "id": "alt-uuid-3", "setNumber": 3, "reps": 10, "kg": "-", "completed": false }
            ]
          },
          {
            "name": "Hack Squat",
            "rir": "2-3",
            "restSeconds": 120,
            "videoLink": "https://...",
            "sets": [
              { "id": "alt-uuid-4", "setNumber": 1, "reps": 10, "kg": "-", "completed": false },
              { "id": "alt-uuid-5", "setNumber": 2, "reps": 10, "kg": "-", "completed": false },
              { "id": "alt-uuid-6", "setNumber": 3, "reps": 10, "kg": "-", "completed": false }
            ]
          }
        ]
      },
      {
        "id": "workout-uuid-2",
        "name": "Plancha",
        "badgeColor": "#374151",
        "rir": "1-2",
        "sets": [
          { "id": "workout-uuid-2-1", "setNumber": 1, "reps": 30, "kg": "-", "completed": false }
        ],
        "tips": [],
        "steps": []
      }
    ]
  }
}
```

### Contrato del campo `alternativeExercises`

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `name` | `string` | Si | Nombre en espanol del ejercicio alternativo (campo `spanish_name` de la tabla `exercises`) |
| `rir` | `string` | Si | RIR del ejercicio alternativo (puede diferir del original) |
| `restSeconds` | `number` | No | Segundos de descanso entre series (puede diferir del original) |
| `videoLink` | `string` | Si | URL del video (campo `link` de la tabla `exercises`) |
| `sets` | `SetData[]` | Si | Array de sets con la misma estructura que los sets del ejercicio original |

### Contrato de cada `set` dentro de alternativas

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `id` | `string` | Si | Identificador unico del set. Debe permitir operaciones PATCH individuales |
| `setNumber` | `number` | Si | Numero del set (1, 2, 3...) |
| `reps` | `number` | Si | Repeticiones prescritas para este set |
| `kg` | `string` | Si | Peso pre-cargado (`"-"` si no hay historial) |
| `completed` | `boolean` | Si | Estado de completitud del set |

### Reglas del campo

- `alternativeExercises` es **opcional** (`omitempty` en Go). Si un ejercicio no tiene alternativas, el campo no aparece en el JSON.
- El array puede tener 1 o mas alternativas (el frontend soporta N alternativas con navegacion ciclica).
- Cada alternativa **no** necesita `tips` ni `steps` (el frontend no los muestra para alternativas).
- Cada alternativa **no** necesita `badgeColor` (el frontend usa un color fijo `#6366F1` para alternativas).

---

## 2. Datos Necesarios para Alternativas

### Por cada ejercicio alternativo se requiere

| Dato | Fuente Actual | Descripcion |
|------|---------------|-------------|
| Nombre en espanol | `exercises.spanish_name` | Ya existe en el catalogo |
| Video link | `exercises.link` | Ya existe en el catalogo |
| RIR | Debe calcularse/asignarse | Puede venir del `set_profiles` o ser heredado del ejercicio original |
| Rest seconds | Debe calcularse/asignarse | Puede venir del `set_profiles` o ser heredado del ejercicio original |
| Numero de sets | Debe calcularse/asignarse | Puede venir del `set_profiles` o ser heredado del ejercicio original |
| Reps por set | Debe calcularse/asignarse | Puede venir del `set_profiles` o ser heredado del ejercicio original |
| Peso historico | `set_values` | Pre-cargar si el usuario ya hizo este ejercicio antes |

### Relacion con el catalogo de ejercicios

Las alternativas deben ser ejercicios validos de la tabla `exercises` que compartan el mismo `pattern` (patron de movimiento) que el ejercicio original. Ejemplo:

- Ejercicio original: "Sentadilla con barra" (pattern: `squat`)
- Alternativa valida: "Goblet Squat" (pattern: `squat`)
- Alternativa invalida: "Press banca" (pattern: `horizontal_push`)

---

## 3. Operaciones de Sets para Alternativas

### Endpoint existente: `PATCH /api/v1/sets/:setId`

El frontend ya usa este endpoint para guardar peso (`kg`) y reps en los sets. Para alternativas, necesita funcionar **exactamente igual**:

- El `setId` de un set alternativo debe seguir el mismo formato que funciona hoy.
- El body sigue siendo: `{ "kg": "25" }` o `{ "reps": 10 }` o ambos.
- La escritura va a la tabla `set_values` igual que hoy.

### Pregunta clave: formato del `setId` para alternativas

El formato actual es `{workoutId}-{setNumber}` donde `workoutId` es el `id` de la tabla `workouts`. Para alternativas, el backend debe definir un formato de `setId` que permita:

1. Identificar que se trata de un ejercicio alternativo (no el original).
2. Saber cual `exercise_id` alternativo se esta usando.
3. Persistir el `set_values` asociado al `exercise_id` correcto del alternativo.

### Endpoint existente: `PATCH /api/v1/sets/:setId/complete`

Mismo requerimiento: debe funcionar para sets de alternativas. Cuando se completa un set de una alternativa, se marca ese set como completado.

### No se requieren endpoints nuevos

Todos los endpoints existentes deben soportar operaciones en sets de ejercicios alternativos. No se necesita crear rutas nuevas.

---

## 4. Datos a Persistir

### 4.1 Asignacion de alternativas por ejercicio

Se necesita saber **que ejercicios alternativos tiene cada workout asignado**. Datos necesarios:

| Dato | Descripcion |
|------|-------------|
| Referencia al workout original | Cual fila de `workouts` es el ejercicio principal |
| Exercise ID alternativo | Cual `exercise_id` de la tabla `exercises` es la alternativa |
| Orden de la alternativa | Posicion en la lista de alternativas (1, 2, ...) |
| Sets prescritos | Numero de sets para esta alternativa |
| Reps prescritos | Repeticiones (o rango) para esta alternativa |
| RIR | Reps en reserva |
| Rest seconds | Descanso entre series |

### 4.2 Registro de valores por set (ya existe)

La tabla `set_values` ya almacena:

| Columna | Uso |
|---------|-----|
| `user_id` | Usuario |
| `exercise_id` | Ejercicio (original O alternativo) |
| `workout_id` | Referencia al workout |
| `set_number` | Numero de set |
| `actual_weight` | Peso registrado |
| `actual_reps` | Reps realizadas |
| `recorded_at` | Timestamp |

Esta tabla **ya soporta** guardar valores para cualquier `exercise_id`, sea original o alternativo. Lo unico necesario es que el `exercise_id` del alternativo se pase correctamente.

### 4.3 Registro de cual ejercicio se realizo

Se necesita saber **cual ejercicio eligio finalmente el usuario** (el original o una alternativa). Datos necesarios:

| Dato | Descripcion |
|------|-------------|
| Referencia al workout | Cual fila de `workouts` |
| Exercise ID seleccionado | El `exercise_id` que el usuario realmente hizo |
| Timestamp de seleccion | Cuando se confirmo la eleccion |

Esta informacion es implicita si el usuario tiene `set_values` registrados para un `exercise_id` alternativo, pero podria ser beneficioso tener un registro explicito.

---

## 5. Reglas de Negocio

### RN-1: Compromiso al completar un set

Cuando un usuario completa (marca como completado) **cualquier set** de un ejercicio (original o alternativa), queda comprometido con ese ejercicio. No puede cambiar a otra alternativa.

- **Que aplica**: El frontend ya implementa esta regla visualmente (oculta el boton "Ver alternativa" cuando algun set esta completado).
- **Que debe hacer el backend**: El backend no necesita bloquear el swap a nivel de API. Pero si un usuario tiene `set_values` registrados para un `exercise_id` alternativo, los valores del ejercicio original deben ignorarse para esa sesion (y viceversa).

### RN-2: Pre-carga de pesos historicos

Los sets de ejercicios alternativos deben pre-cargar el ultimo peso registrado del usuario para ese `exercise_id` alternativo, usando la misma logica que ya existe para el ejercicio original (`GetLastWeightsForExercise`).

- Si el usuario nunca hizo el ejercicio alternativo, `kg` = `"-"`.
- Si el usuario lo hizo en una semana anterior, `kg` = ultimo peso registrado.

### RN-3: Calculo de sets y reps para alternativas

Los sets y reps de cada alternativa deben calcularse usando la misma logica del ejercicio original:

- Usar `set_profiles` segun goal/level/week/role para determinar sets, reps, RIR y rest.
- El `role` del ejercicio alternativo viene de la tabla `exercises` (compound, isolation, core).
- Si el role del alternativo difiere del original, los parametros de sets/reps pueden diferir.

### RN-4: Un ejercicio puede no tener alternativas

No todos los ejercicios tendran alternativas asignadas. El campo `alternativeExercises` simplemente no aparece en el JSON para esos ejercicios.

### RN-5: Determinacion del ejercicio realizado

Para reportes y progresion, el backend debe poder determinar cual ejercicio hizo realmente el usuario en cada slot de la rutina. La fuente de verdad es la existencia de registros en `set_values`:

- Si hay registros con el `exercise_id` original: hizo el original.
- Si hay registros con un `exercise_id` alternativo: hizo la alternativa.
- Si no hay registros: no hizo ninguno.

---

## 6. Origen de las Alternativas

### Quien genera las alternativas y cuando

Las alternativas deben asignarse durante la **generacion de la rutina** en el flujo `WORKOUT_CREATOR.json`. Para cada ejercicio seleccionado por la IA, se deben identificar 1-2 ejercicios alternativos del mismo `pattern` que cumplan:

- Mismo patron de movimiento (`exercises.pattern`)
- Compatible con el nivel del usuario (`exercises.level`)
- Compatible con el entorno de entrenamiento (gym vs home, `exercises.equipment`)
- No estar en la lista de ejercicios rechazados del usuario (`disliked_exercises`)

### Este documento no cubre

- Como el `WORKOUT_CREATOR` selecciona las alternativas (eso es logica del workflow).
- Como se genera la estructura de datos en n8n (eso es implementacion del workflow).
- Solo define que el backend necesita **leer** esas alternativas ya asignadas y **devolverlas** en el contrato de API.

---

## 7. Resumen de Cambios Requeridos

| Componente | Cambio |
|------------|--------|
| **DTO** `ExerciseDTO` | Agregar campo `AlternativeExercises []AlternativeExerciseDTO` |
| **DTO** nuevo `AlternativeExerciseDTO` | Crear con campos: Name, RIR, RestSeconds, VideoLink, Sets |
| **Entidad** `Exercise` | Agregar campo para alternativas |
| **Repositorio** `GetTodayWorkout` | Consultar alternativas asociadas a cada ejercicio |
| **Use case** `GetTodayWorkoutUseCase` | Mapear alternativas de entidad a DTO |
| **Repositorio** `SetRepository.Update` | Soportar `setId` de alternativas (resolver el `exercise_id` correcto) |
| **Base de datos** | Almacenar asignacion de alternativas por workout |
| **WORKOUT_CREATOR (n8n)** | Generar y guardar alternativas al crear la rutina |
