# Fase 1: Modificaciones al KYC/Encuesta - Ambiente de Entrenamiento

**Documento Tecnico de Especificacion**

| Campo | Valor |
|-------|-------|
| Fase | 1 de N |
| Asignado a | n8n-agent |
| Revisado por | code-reviewer |
| Estado | Pendiente |
| Fecha de creacion | 2026-02-03 |

---

## 1. Objetivo

Agregar una nueva pregunta al flujo KYC (Know Your Customer) para identificar el ambiente de entrenamiento preferido del usuario (Gimnasio o Casa), y en caso de seleccionar Casa, recopilar informacion sobre el equipamiento disponible.

### Resultado Esperado

- Nueva FASE 6.5 en el flujo de conversacion del KYC Agent
- Dos nuevos campos en la tabla `users_gym_profile`:
  - `training_environment` (TEXT, NOT NULL, valores: 'GYM' o 'HOME')
  - `home_equipment` (TEXT, nullable, solo aplica cuando `training_environment = 'HOME'`)
- Actualizacion del nodo `Tool_Create_User_Profile` para guardar los nuevos campos

---

## 2. Archivos a Modificar

| Archivo | Ruta Absoluta | Tipo de Cambio |
|---------|---------------|----------------|
| FormPrompt.txt | `/Users/camilodiazjaimes/Documents/GymBot/n8n/system_prompts/FormPrompt.txt` | Agregar FASE 6.5 |
| MAIN_FLOW.json | `/Users/camilodiazjaimes/Documents/GymBot/n8n/running_flows/MAIN_FLOW.json` | Modificar nodo `Tool_Create_User_Profile` |

### Archivos de Referencia (Solo Lectura)

| Archivo | Proposito |
|---------|-----------|
| `CLAUDE.md` | Documentacion de esquema de BD y flujos |

---

## 3. Cambios en FormPrompt.txt

### Ubicacion de Insercion

La nueva FASE 6.5 debe insertarse **inmediatamente despues de la FASE 6** (linea 94) y **antes de la FASE 7** (linea 95).

### Texto EXACTO a Insertar

Insertar el siguiente bloque despues de la linea 94 (`* **Desafios:** ...`):

```text

## FASE 6.5: Tu Espacio de Entrenamiento (El Lugar)
Ahora necesitamos saber donde vas a entrenar.

* **Ambiente de entrenamiento:**
    Pregunta: "Donde prefieres entrenar? Esto me ayudara a disenar ejercicios adecuados para ti."
    - Gimnasio (tienes acceso a maquinas, pesas, barras, etc.)
    - Casa (entrenas desde tu hogar)

* **VALIDACION AMBIENTE:**
    * Si el usuario responde "ambos", "los dos", "a veces gym a veces casa" o similar:
      *Respuesta del Agente:* "Entiendo que a veces alternas! Para disenar tu rutina base, necesito que elijas **uno principal**. Donde pasas la mayoria de tus entrenamientos: Gimnasio o Casa?"
    * Si el usuario responde algo ambiguo como "donde pueda":
      *Respuesta del Agente:* "Sin problema! Pero para personalizar bien tu plan, dime: si tuvieras que elegir uno, seria Gimnasio o Casa?"

* **Seguimiento para Casa (OBLIGATORIO si elige Casa):**
    Si el usuario selecciona "Casa", pregunta inmediatamente:
    "Genial, entrenar en casa es super conveniente! Cuentame, que equipamiento tienes disponible? Por ejemplo:"
    - Mancuernas o pesas
    - Bandas elasticas / ligas de resistencia
    - Barra de dominadas (pull-up bar)
    - Banco de pesas
    - TRX o suspension trainer
    - Kettlebells
    - Solo peso corporal (sin equipamiento)
    - Otro (especifica)

    *Nota para el agente:* Permite respuestas multiples. Si el usuario dice "solo mi cuerpo" o "nada", guarda "Peso corporal".

* **VALIDACION EQUIPAMIENTO:**
    * Si el usuario elige Casa pero no especifica equipamiento despues de 2 intentos:
      *Respuesta del Agente:* "No te preocupes! Asumire que entrenaras con peso corporal por ahora. Siempre podemos ajustar despues."
      Guardar: `home_equipment = "Peso corporal"`

* **REGLA FASE 6.5 (AMBIENTE):**
    - Si el usuario elige "Gimnasio": guarda `training_environment = "GYM"` y `home_equipment = null`
    - Si el usuario elige "Casa": guarda `training_environment = "HOME"` y pregunta por equipamiento
    - Despues de obtener el ambiente (y equipamiento si aplica), continua con FASE 7 (Cardio)

```

### Renumeracion de Fases

Despues de insertar FASE 6.5, la estructura de fases quedara:

1. FASE 1: El Saludo y Contacto
2. FASE 2: Perfil Fisico
3. FASE 3: Tus Objetivos
4. FASE 4: Tu Experiencia
5. FASE 5: Logistica
6. FASE 6: Preferencias
7. **FASE 6.5: Tu Espacio de Entrenamiento (NUEVA)**
8. FASE 7: El Cardio

---

## 4. Cambios en Tool_Create_User_Profile

### Nodo a Modificar

- **Nombre del nodo:** `Tool_Create_User_Profile`
- **ID del nodo:** `ae1f74ff-a4e1-4047-b596-8f19fa2118a2`
- **Tipo:** `n8n-nodes-base.supabaseTool`
- **Tabla destino:** `users_gym_profile`

### Campos Actuales (Referencia)

El nodo actualmente tiene 18 campos en `fieldsUi.fieldValues`. Los nuevos campos deben agregarse **despues de `cardio_frequency`** (el ultimo campo actual).

### JSON de Campos Nuevos a Agregar

Agregar los siguientes dos objetos al array `fieldsUi.fieldValues`:

```json
{
  "fieldId": "training_environment",
  "fieldValue": "={{ $fromAI(\"ambiente_entrenamiento\", \"opciones: GYM, HOME\", \"string\") }}"
},
{
  "fieldId": "home_equipment",
  "fieldValue": "={{ $fromAI(\"equipamiento_disponible\", \"nullable, lista de equipos separados por coma o null si GYM\", \"string\") }}"
}
```

### Estructura Final del Nodo (Ultimos 4 Campos)

```json
{
  "fieldId": "cardio_type",
  "fieldValue": "={{ $fromAI(\"tipo_de_cardio_que_realiza\", \"opciones: No, Caminata, Bicicleta, Running\", \"string\") }}"
},
{
  "fieldId": "cardio_frequency",
  "fieldValue": "={{ $fromAI(\"frecuencia_de_cardio\", \"opciones: 0, 1-2, 3-4, 5 o mas\", \"string\") }}"
},
{
  "fieldId": "training_environment",
  "fieldValue": "={{ $fromAI(\"ambiente_entrenamiento\", \"opciones: GYM, HOME\", \"string\") }}"
},
{
  "fieldId": "home_equipment",
  "fieldValue": "={{ $fromAI(\"equipamiento_disponible\", \"nullable, lista de equipos separados por coma o null si GYM\", \"string\") }}"
}
```

---

## 5. Flujo de Conversacion

### Diagrama del Nuevo Flujo de 8 Fases

```
+------------------+
|   FASE 1         |
|  Saludo/Contacto |
|  - Nombre        |
|  - Email         |
+--------+---------+
         |
         v
+------------------+
|   FASE 2         |
|  Perfil Fisico   |
|  - Edad          |
|  - Sexo          |
|  - Estatura      |
|  - Peso          |
+--------+---------+
         |
         v
+------------------+
|   FASE 3         |
|  Objetivos       |
|  - Principal     |
|  - Secundario    |
+--------+---------+
         |
         v
+------------------+
|   FASE 4         |
|  Experiencia     |
|  - Tiempo        |
|  - Frecuencia    |
|  - Nivel         |
|  - Salud         |
+--------+---------+
         |
         v
+------------------+
|   FASE 5         |
|  Logistica       |
|  - Dias disp.    |
|  - Tiempo sesion |
|  - Horario       |
+--------+---------+
         |
         v
+------------------+
|   FASE 6         |
|  Preferencias    |
|  - Tipo entreno  |
|  - Prioridades   |
|  - Desafios      |
+--------+---------+
         |
         v
+------------------+     +--------------------+
|   FASE 6.5       |     |                    |
|  Ambiente (NUEVA)|---->| Si elige "Casa"    |
|  - GYM o HOME    |     | Preguntar equipo   |
+--------+---------+     +----------+---------+
         |                          |
         |<-------------------------+
         v
+------------------+
|   FASE 7         |
|  Cardio          |
|  - Tipo          |
|  - Frecuencia    |
+--------+---------+
         |
         v
+------------------+
| Tool_Create_     |
| User_Profile     |
| (20 campos)      |
+------------------+
```

### Flujo Condicional FASE 6.5

```
Usuario responde FASE 6 (Preferencias)
         |
         v
+------------------------+
| "Donde prefieres       |
|  entrenar?"            |
+------------------------+
         |
    +----+----+
    |         |
    v         v
+-------+  +-------+
|  GYM  |  | HOME  |
+---+---+  +---+---+
    |          |
    |          v
    |    +------------------+
    |    | "Que equipamiento|
    |    |  tienes?"        |
    |    +--------+---------+
    |             |
    |             v
    |    +------------------+
    |    | Guardar lista    |
    |    | de equipos       |
    |    +--------+---------+
    |             |
    +------+------+
           |
           v
    +------------------+
    | Continuar FASE 7 |
    +------------------+
```

---

## 6. Validaciones

### Antes de Guardar en BD

| Campo | Validacion | Accion si Falla |
|-------|------------|-----------------|
| `training_environment` | Debe ser exactamente `'GYM'` o `'HOME'` | Pedir aclaracion al usuario |
| `home_equipment` | Si `training_environment = 'GYM'`, debe ser `null` | Forzar a `null` |
| `home_equipment` | Si `training_environment = 'HOME'`, debe tener valor | Default a `'Peso corporal'` |

### Casos Edge

| Caso | Respuesta del Usuario | Manejo |
|------|----------------------|--------|
| Ambiguo | "ambos", "los dos", "depende" | Pedir que elija uno principal |
| Gimnasio de casa | "tengo un gym en casa" | Tratar como HOME + preguntar equipo |
| Sin equipo | "nada", "solo yo", "mi cuerpo" | Guardar `home_equipment = 'Peso corporal'` |
| Equipo parcial | "unas mancuernas viejas" | Guardar tal cual: "Mancuernas" |
| Multiples equipos | "mancuernas, bandas y barra" | Guardar como string: "Mancuernas, Bandas elasticas, Barra de dominadas" |
| Gimnasio pero sin maquinas | "voy al gym pero solo tiene pesas" | Guardar como GYM (el motor de rutinas manejara esto) |

### Normalizacion de Equipamiento

El agente debe normalizar las respuestas del usuario a terminos estandar:

| Respuesta Usuario | Valor Normalizado |
|-------------------|-------------------|
| "pesas", "dumbbells", "mancuernitas" | Mancuernas |
| "ligas", "bandas", "therabands" | Bandas elasticas |
| "barra de puerta", "barra para jalon" | Barra de dominadas |
| "banco", "banca" | Banco de pesas |
| "cuerdas", "TRX", "anillas" | TRX |
| "pesas rusas", "kettlebell" | Kettlebells |
| "nada", "solo yo", "cuerpo" | Peso corporal |

---

## 7. Tareas Accionables

### Para n8n-agent

- [ ] **TAREA 1:** Crear migracion SQL para agregar columnas a `users_gym_profile`
  ```sql
  ALTER TABLE users_gym_profile
  ADD COLUMN training_environment TEXT NOT NULL DEFAULT 'GYM',
  ADD COLUMN home_equipment TEXT;
  ```

- [ ] **TAREA 2:** Actualizar archivo `FormPrompt.txt`
  - Abrir `/Users/camilodiazjaimes/Documents/GymBot/n8n/system_prompts/FormPrompt.txt`
  - Insertar FASE 6.5 despues de linea 94 (despues de "Desafios")
  - Verificar que no se rompa el flujo existente

- [ ] **TAREA 3:** Actualizar nodo `Tool_Create_User_Profile` en workflow
  - Abrir `/Users/camilodiazjaimes/Documents/GymBot/n8n/running_flows/GymRatFlow_Supabase_V3_Workout_Tracker.json`
  - Localizar nodo con ID `ae1f74ff-a4e1-4047-b596-8f19fa2118a2`
  - Agregar los 2 nuevos campos al array `fieldsUi.fieldValues`

- [ ] **TAREA 4:** Actualizar reglas criticas en FormPrompt.txt
  - Agregar `ambiente_entrenamiento` a la lista de campos requeridos (linea 141-161)
  - Agregar `equipamiento_disponible` como campo condicional

- [ ] **TAREA 5:** Actualizar CLAUDE.md
  - Agregar las 2 nuevas columnas a la documentacion de `users_gym_profile`

- [ ] **TAREA 6:** Probar flujo completo
  - Ejecutar KYC con usuario que elige GYM
  - Ejecutar KYC con usuario que elige HOME + varios equipos
  - Ejecutar KYC con usuario que elige HOME + sin equipo
  - Verificar datos en tabla `users_gym_profile`

---

## 8. Criterios de Aceptacion

### Funcionales

| # | Criterio | Verificacion |
|---|----------|--------------|
| AC1 | El KYC Agent pregunta por ambiente de entrenamiento despues de FASE 6 | Manual - ejecutar flujo |
| AC2 | Si usuario elige "Casa", el agente pregunta por equipamiento | Manual - ejecutar flujo |
| AC3 | Si usuario elige "Gimnasio", el agente NO pregunta por equipamiento | Manual - ejecutar flujo |
| AC4 | Campo `training_environment` se guarda correctamente en BD | Query: `SELECT training_environment FROM users_gym_profile WHERE whatsapp_id = X` |
| AC5 | Campo `home_equipment` se guarda con lista de equipos separados por coma | Query: `SELECT home_equipment FROM users_gym_profile WHERE whatsapp_id = X` |
| AC6 | Campo `home_equipment` es NULL cuando `training_environment = 'GYM'` | Query SQL |
| AC7 | El flujo completo del KYC sigue funcionando sin errores | E2E test TC002 pasa |

### Tecnicos

| # | Criterio | Verificacion |
|---|----------|--------------|
| TC1 | La migracion SQL se ejecuta sin errores | Supabase logs |
| TC2 | El workflow se importa sin errores en n8n | n8n UI |
| TC3 | Las expresiones `$fromAI()` son validas | n8n executions sin error |
| TC4 | No hay regresiones en campos existentes | Comparar perfil completo antes/despues |

### Datos de Prueba

**Usuario de prueba GYM:**
```
Ambiente: Gimnasio
Equipamiento: NULL
```

**Usuario de prueba HOME completo:**
```
Ambiente: Casa
Equipamiento: "Mancuernas, Bandas elasticas, Barra de dominadas, Kettlebells"
```

**Usuario de prueba HOME minimo:**
```
Ambiente: Casa
Equipamiento: "Peso corporal"
```

---

## Anexos

### A. Campos Completos de Tool_Create_User_Profile (Despues de Modificacion)

| # | fieldId | Expresion $fromAI |
|---|---------|-------------------|
| 1 | submission_date | `$now` |
| 2 | whatsapp_id | `$items("If")[0].json.contacts[0].wa_id` |
| 3 | full_name | `$fromAI("nombre_del_usuario")` |
| 4 | email | `$fromAI('correo_electronico', '', 'string')` |
| 5 | age | `$fromAI("edad", "", "number")` |
| 6 | biological_sex | `$fromAI("sexo", "M/F")` |
| 7 | height_cm | `$fromAI("estatura", "", "number")` |
| 8 | weight_kg | `$fromAI("peso", "", "number")` |
| 9 | primary_goal | `$fromAI("objetivo_principal", "opciones: ...")` |
| 10 | secondary_goal | `$fromAI("objetivo_secundario", "", "string")` |
| 11 | training_experience | `$fromAI("experiencia", "opciones: ...")` |
| 12 | current_frequency | `$fromAI("frecuencia_actual", "opciones: ...")` |
| 13 | fitness_level | `$fromAI("nivel_autopercibido", "opciones: ...")` |
| 14 | health_status | `$fromAI("estado_de_salud", "opciones: A,B,C,D,E")` |
| 15 | days_available | `$fromAI("dias_disponibles_para_entrenar", "opciones: 2,3,4,5,6", "number")` |
| 16 | session_duration_mins | `$fromAI("tiempo_por_sesion", "opciones: ...")` |
| 17 | preferred_schedule | `$fromAI("horario_habitual", "opciones: ...")` |
| 18 | training_style | `$fromAI("tipo_de_entrenamiento_preferido", "opciones: ...")` |
| 19 | priority_muscles | `$fromAI("partes_del_cuerpo_a_priorizar", "", "string")` |
| 20 | disliked_exercises | `$fromAI("partes_que_no_le_gustan", "", "string")` |
| 21 | cardio_type | `$fromAI("tipo_de_cardio_que_realiza", "opciones: ...")` |
| 22 | cardio_frequency | `$fromAI("frecuencia_de_cardio", "opciones: ...")` |
| **23** | **training_environment** | **`$fromAI("ambiente_entrenamiento", "opciones: GYM, HOME", "string")`** |
| **24** | **home_equipment** | **`$fromAI("equipamiento_disponible", "nullable, lista separada por coma o null si GYM", "string")`** |

### B. Ejemplo de Conversacion KYC con FASE 6.5

```
AGENTE: [Completa FASE 6]
        Genial! Ahora cuentame, que partes del cuerpo te cuesta mas
        entrenar o simplemente no te gustan?

USUARIO: No me gusta entrenar pantorrillas

AGENTE: Entendido, las dejaremos en segundo plano entonces.

        Ahora una pregunta importante: Donde prefieres entrenar?
        - Gimnasio (tienes acceso a maquinas, pesas, barras, etc.)
        - Casa (entrenas desde tu hogar)

USUARIO: En mi casa

AGENTE: Genial, entrenar en casa es super conveniente!
        Cuentame, que equipamiento tienes disponible? Por ejemplo:
        - Mancuernas o pesas
        - Bandas elasticas
        - Barra de dominadas
        - Banco de pesas
        - Kettlebells
        - Solo peso corporal

USUARIO: Tengo unas mancuernas y unas bandas

AGENTE: Perfecto! Con mancuernas y bandas elasticas podemos armar
        una rutina muy completa.

        [Continua FASE 7 - Cardio]
        Y para terminar, realizas cardio actualmente?
```

---

**Fin del Documento**
