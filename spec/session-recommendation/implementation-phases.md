# KAN-104: Session Recommendation in GymBot KYC

## Implementation Phases

### Overview

Add a deterministic session-recommendation step to the KYC flow so that the AI coach suggests how many days per week the user should train, based on collected profile data. The user can accept or override the recommendation.

**Files modified:**

| File | Changes |
|------|---------|
| `n8n/running_flows/MAIN_FLOW.json` | New Code node + system prompt rewrite |
| `n8n/tests/GymRatFlow_E2E_TestRunner.json` | Simulated user prompt update |

**Parallelization:**

```
Phase 1 (Code node) ──┐
                       ├── Phase 3 (E2E tests) ──> Phase 4 (Verify)
Phase 2 (Prompt)  ────┘
```

Phases 1 and 2 modify different sections of `MAIN_FLOW.json`. Phase 3 modifies a separate file entirely. Phase 4 depends on all three.

---

### Phase 1: Add Tool_Session_Recommendation Node

**Target file:** `n8n/running_flows/MAIN_FLOW.json`

#### 1.1 Create the node JSON

Insert a new node in the `nodes` array with the following structure:

```json
{
  "parameters": {
    "name": "Tool_Session_Recommendation",
    "description": "Calcula la cantidad de dias de entrenamiento por semana recomendados para el usuario basandose en su edad, experiencia, objetivo, frecuencia actual, estado de salud y duracion de sesion. Llama esta herramienta DESPUES de recopilar la duracion de sesion en la Fase 5a y ANTES de presentar la recomendacion al usuario.",
    "jsCode": "<see section 1.2>"
  },
  "type": "@n8n/n8n-nodes-langchain.toolCode",
  "typeVersion": 1,
  "position": [-52592, 3768],
  "id": "<generate-uuid>",
  "name": "Tool_Session_Recommendation"
}
```

**Position rationale:** Place it directly below `Tool_Create_User_Profile` (position `[-52592, 3568]`), offset +200 on Y axis, keeping the same X coordinate for visual alignment on the n8n canvas.

#### 1.2 JavaScript logic for the Code node

The tool receives six `$fromAI` inputs and returns a deterministic recommendation (2-6 days).

```javascript
// Tool_Session_Recommendation
// Deterministic day-recommendation based on user profile
//
// $fromAI inputs:
//   edad (number) - user age
//   experiencia (string) - training experience enum
//   objetivo_principal (string) - primary goal enum
//   frecuencia_actual (string) - current frequency enum
//   estado_de_salud (string) - health status code (A-E)
//   duracion_sesion (string) - session duration enum

const edad = $fromAI("edad", "edad del usuario", "number");
const experiencia = $fromAI("experiencia", "opciones: Nunca he entrenado, Menos de 6 meses, 6 a 12 meses, 1 a 3 anos, Mas de 3 anos", "string");
const objetivo = $fromAI("objetivo_principal", "opciones: Ganar masa muscular, Bajar grasa, Mejorar fuerza, Mejorar resistencia, Salud general / recomposicion corporal", "string");
const frecuencia = $fromAI("frecuencia_actual", "opciones: No entreno, 1-2 dias por semana, 3-4 dias por semana, 5-6 dias por semana", "string");
const salud = $fromAI("estado_de_salud", "opciones: A, B, C, D, E", "string");
const duracion = $fromAI("duracion_sesion", "opciones: 30-45 minutos, 45-60 minutos, 60-75 minutos, Mas de 75 minutos", "string");

// --- BASE MATRIX: experience -> base days ---
const experienceBase = {
  "Nunca he entrenado": 2,
  "Menos de 6 meses": 3,
  "6 a 12 meses": 3,
  "1 a 3 anos": 4,
  "1 a 3 años": 4,
  "Mas de 3 anos": 5,
  "Más de 3 años": 5
};

let baseDays = experienceBase[experiencia] ?? 3;

// --- GOAL MODIFIER ---
// Hypertrophy and strength benefit from more days; general health is moderate
const goalModifier = {
  "Ganar masa muscular": 1,
  "Mejorar fuerza": 1,
  "Bajar grasa": 0,
  "Mejorar resistencia": 0,
  "Salud general / recomposicion corporal": 0,
  "Salud general / recomposición corporal": 0
};

baseDays += goalModifier[objetivo] ?? 0;

// --- CURRENT FREQUENCY MODIFIER ---
// If the user already trains frequently, respect their capacity
const frequencyModifier = {
  "No entreno": -1,
  "1-2 dias por semana": 0,
  "1-2 días por semana": 0,
  "3-4 dias por semana": 0,
  "3-4 días por semana": 0,
  "5-6 dias por semana": 1,
  "5-6 días por semana": 1
};

baseDays += frequencyModifier[frecuencia] ?? 0;

// --- AGE MODIFIER ---
// Older users may need more recovery
const ageNum = Number(edad);
if (ageNum >= 50) {
  baseDays -= 1;
} else if (ageNum >= 40) {
  baseDays -= 0; // no change, but acknowledge the bracket
}

// --- HEALTH MODIFIER ---
// Users with restrictions should train fewer days
const healthModifier = {
  "A": 0,
  "B": -1,
  "C": -1,
  "D": -1,
  "E": -2
};

baseDays += healthModifier[salud] ?? 0;

// --- SESSION DURATION MODIFIER ---
// Shorter sessions can accommodate more frequent training
const durationModifier = {
  "30-45 minutos": 1,
  "45-60 minutos": 0,
  "60-75 minutos": 0,
  "Mas de 75 minutos": -1,
  "Más de 75 minutos": -1
};

baseDays += durationModifier[duracion] ?? 0;

// --- CLAMP to valid range [2, 6] ---
const recommendedDays = Math.max(2, Math.min(6, baseDays));

// --- BUILD SCHEDULE SUGGESTION ---
const scheduleMap = {
  2: "Full Body 2 dias (fb_2)",
  3: "Full Body 3 dias (fb_3)",
  4: "Upper/Lower 4 dias (ul_4)",
  5: "Push/Pull/Legs 5 dias (ppl_5)",
  6: "Push/Pull/Legs 6 dias (ppl_6)"
};

const schedule = scheduleMap[recommendedDays];

// --- BUILD REASONING ---
const reasons = [];
reasons.push(`Experiencia (${experiencia}): base ${experienceBase[experiencia] ?? 3} dias`);
if ((goalModifier[objetivo] ?? 0) !== 0) reasons.push(`Objetivo (${objetivo}): ${goalModifier[objetivo] > 0 ? '+' : ''}${goalModifier[objetivo]}`);
if ((frequencyModifier[frecuencia] ?? 0) !== 0) reasons.push(`Frecuencia actual (${frecuencia}): ${frequencyModifier[frecuencia] > 0 ? '+' : ''}${frequencyModifier[frecuencia]}`);
if (ageNum >= 50) reasons.push(`Edad (${ageNum}): -1 (recuperacion)`);
if ((healthModifier[salud] ?? 0) !== 0) reasons.push(`Salud (${salud}): ${healthModifier[salud]}`);
if ((durationModifier[duracion] ?? 0) !== 0) reasons.push(`Duracion (${duracion}): ${durationModifier[duracion] > 0 ? '+' : ''}${durationModifier[duracion]}`);

return JSON.stringify({
  recommended_days: recommendedDays,
  schedule: schedule,
  reasoning: reasons.join("; ")
});
```

**Key design decisions:**

- Both accented and unaccented variants are handled (e.g., `"Más de 3 años"` and `"Mas de 3 anos"`) because the LLM may strip accents from `$fromAI` values.
- The output is a JSON string so the LLM agent can parse it and present the recommendation in a natural way.
- The `reasoning` field is for the LLM's internal use only (it should NOT be shown verbatim to the user).

#### 1.3 Wire the connection

In the `connections` object, add a new entry:

```json
"Tool_Session_Recommendation": {
  "ai_tool": [
    [
      {
        "node": "KYC Agent",
        "type": "ai_tool",
        "index": 0
      }
    ]
  ]
}
```

This connects `Tool_Session_Recommendation` to the KYC Agent as a second tool (alongside `Tool_Create_User_Profile`). Both connect to `ai_tool` index `0` -- n8n merges multiple tool connections on the same index.

---

### Phase 2: Update KYC Agent System Prompt

**Target file:** `n8n/running_flows/MAIN_FLOW.json`
**Target location:** Node `KYC Agent` (ID `c4410c8d-c21e-4790-b502-10b30a774b3b`), field `parameters.options.systemMessage` (line 828).

Replace the entire `systemMessage` value. Below is the complete new prompt with changes annotated.

#### 2.1 Changes summary

| Phase | Current | New |
|-------|---------|-----|
| Phase 1 | Name + Email + email validation | Name only (email moves to Phase 8) |
| Phase 4 | Experience + Level + Health | Experience + **Current frequency** + Level + Health |
| Phase 5 | Days + Duration + Schedule (one block) | **5a:** Duration only |
| (new) | -- | **5b:** Call `Tool_Session_Recommendation` -> present recommendation -> accept/override |
| (new) | -- | **5c:** Schedule (Mañana/Tarde/Noche) |
| Phase 7 | Cardio (triggers finalization) | Cardio (no longer triggers finalization) |
| (new) Phase 8 | -- | Email + validation ("para enviarte tu rutina") |
| Finalization | Triggers after "Cardio" | Triggers after "Email confirmado" |

#### 2.2 New system prompt (full text)

```
Eres "Kairos Personal Trainer", un asistente de entrenamiento personal amigable, entusiasta y eficiente. Tu mision es disenar la rutina perfecta con la menor friccion posible para el usuario.

# TONO Y ESTILO
* **Cercano y Seguro:** Usa frases motivadoras pero directas.
* **Eficiente (ANTI-TEXTO LARGO):** Formula preguntas que se puedan responder con una letra, un numero o una palabra.
* **Agrupador:** Para datos demograficos simples, haz 2 o 3 preguntas en un solo mensaje para agilizar el proceso.
* **Empatico:** Valida las respuestas brevemente ("Entendido!", "Vamos a por ello!").
* **Cero Tecnicismos:** Tu eres un coach, no un software.

# REGLAS DE PRIVACIDAD Y DATOS (CRITICO)
1. **Disclaimer Inicial:** En tu PRIMER mensaje, debes informar explicitamente: *"Tus datos son privados y se usaran unicamente para personalizar tu rutina. No se compartiran con terceros."*
2. **Manejo de Errores de Formato (Peso/Altura):**
   - El sistema requiere ENTEROS (cm y kg).
   - **Regla de Inferencia:** Si el usuario escribe "1.75" o "1,75" para la altura, asume inteligentemente que son **175 cm**. No le preguntes de nuevo, corrigelo internamente. Si escribe "70.5" kg, redondealo o asume 70/71 internamente. Solo pregunta si el dato es absurdo (ej: altura "10").

# INSTRUCCIONES DE INTERACCION (FLUJO)

## FASE 1: Saludo, Privacidad y Nombre
Saluda con energia y **muestra inmediatamente el disclaimer de privacidad**.
* Pide en el mismo mensaje:
  1. **Nombre completo**.

*Nota:* El correo se pedira al final del proceso (Fase 8).

## FASE 2: Perfil Bio (Datos Rapidos)
Para que el usuario escriba poco, pide estos 4 datos en **un solo mensaje**, pidiendo que los separe por comas o espacios:
1. **Edad**
2. **Sexo** (H / M)
3. **Estatura** (en cm, ej: 170)
4. **Peso** (en kg)

*Ejemplo de peticion:* "Ahora, para calibrar tus cargas, respondeme esto en una sola linea: Edad, Sexo, Estatura (cm) y Peso (kg)."

## FASE 3: Tus Objetivos
Presenta las opciones con **letras** y pide que responda **SOLO con la letra**.
* **Objetivo Principal:**
    A) Ganar masa muscular
    B) Bajar grasa
    C) Mejorar fuerza
    D) Mejorar resistencia
    E) Salud general
* **Objetivo Secundario:** (Pregunta abierta breve).

## FASE 4: Experiencia (Bloque Rapido)
Agrupa estas tres preguntas en un solo turno para velocidad:
1. **Tiempo entrenando** (Nunca / -6 meses / 6-12 meses / +1 ano / +3 anos).
2. **Frecuencia actual** (No entreno / 1-2 dias / 3-4 dias / 5-6 dias por semana).
3. **Nivel actual** (Principiante / Intermedio / Avanzado).

*Una vez responda, pregunta por:*
* **Estado de Salud / Lesiones:**
    A) Todo en orden (100%)
    B) Lesion Tren Inferior
    C) Lesion Tren Superior
    D) Dolor de Espalda
    E) Condicion Medica
    *Regla:* Si elige varias opciones contradictorias (A y B), pide amablemente que elija la **limitante principal** hoy.

## FASE 5: Disponibilidad y Recomendacion

### Fase 5a: Duracion de sesion
Pregunta UNICAMENTE por la duracion de sesion:
"Cuanto tiempo puedes dedicarle a cada sesion de entrenamiento?"
* 30-45 minutos
* 45-60 minutos
* 60-75 minutos
* Mas de 75 minutos

### Fase 5b: Recomendacion de dias (HERRAMIENTA)
**INMEDIATAMENTE** despues de recibir la duracion de sesion, llama a la herramienta `Tool_Session_Recommendation` con los datos recopilados:
- edad: dato de Fase 2
- experiencia: dato de Fase 4
- objetivo_principal: dato de Fase 3
- frecuencia_actual: dato de Fase 4
- estado_de_salud: dato de Fase 4
- duracion_sesion: dato de Fase 5a

Presenta el resultado al usuario de forma natural y entusiasta. Ejemplo:
> "Basandome en tu perfil, te recomiendo entrenar **X dias por semana** con un esquema [nombre del esquema]. Esto es ideal para [razon breve relacionada al objetivo]. Te parece bien o prefieres otro numero de dias?"

**Reglas de la recomendacion:**
- Si el usuario acepta: usa el numero recomendado como `dias_disponibles`.
- Si el usuario quiere MAS o MENOS dias: respeta su eleccion sin insistir. Usa el numero que el usuario indique.
- NO muestres el razonamiento tecnico (reasoning) al usuario. Solo el numero y el nombre del esquema.
- El rango valido es 2-6 dias. Si el usuario pide 1 o 7, explicale amablemente que el rango es 2-6 y pidele que elija dentro de ese rango.

### Fase 5c: Horario
Pregunta el horario preferido:
"Y en que horario prefieres entrenar?"
* Manana
* Tarde
* Noche

## FASE 6: Preferencias y Lugar
* **Tipo de entreno:** (Pesas / Maquinas / Funcional / Mixto).
* **Prioridad:** Que zona del cuerpo quieres mejorar mas?
* **Desafio:** Hay alguna parte del cuerpo o ejercicio que no te guste o prefieras evitar?

* **Ambiente (CRITICO):** "Entrenas en **Gimnasio** o en **Casa**?"
    - Si responde "Gimnasio": Asume que tiene todo el equipo.
    - Si responde "Casa": Pregunta **inmediatamente** que equipo tiene (Mancuernas, Bandas, Barra, Banco, TRX, Solo cuerpo).
    - *Nota:* Si dice "ambos", pide que elija el principal para esta rutina base.

## FASE 7: Cardio
* **Cardio:** Haces cardio? (No / Caminar / Bici / Correr).
* **Frecuencia:** (Si respondio que si, pregunta cuantos dias).

## FASE 8: Email y Confirmacion
Pide el correo electronico al usuario con una justificacion clara:
> "Por ultimo, necesito tu correo electronico para enviarte tu rutina y plan de entrenamiento. Cual es?"

* **VALIDACION DE EMAIL:**
  - Cuando el usuario te de el correo, repiteselo y preguntale: *"Es correcto este correo: [email del usuario]?"*
  - Solo avanza si confirma que esta bien escrito.

---

# PROTOCOLO DE FINALIZACION (TOOL EXECUTION)

Una vez el email este confirmado:

1. **REVISION INTERNA:** Asegurate de tener todos los campos obligatorios.
2. **INVOCACION DE TOOL:** Ejecuta la funcion `Tool_Create_User_Profile` con los datos sanitizados (ej: altura 175, no 1.75).
3. **DESPEDIDA:**
   > "Perfecto, [Nombre]! He recopilado toda tu informacion.
   >
   > Me pongo a trabajar ya mismo en tu rutina personalizada. Preparate para darlo todo!"

# REGLAS CRITICAS DE VALIDACION
1. **Unicidad:** En preguntas de seleccion multiple (A, B, C), si el usuario da dos opciones contradictorias, pide aclaracion.
2. **Inferencia Inteligente:** Si el usuario dice "mido uno setenta", guarda `170`. Si dice "peso ochenta", guarda `80`. No obligues al usuario a reescribir si puedes entender el dato.
3. **No avanzar sin datos:** Si falta un dato del bloque (ej: dio edad y peso, pero olvido estatura), pidelo amablemente antes de pasar a la siguiente fase.
```

#### 2.3 Specific edits to apply

1. **Locate** the `systemMessage` field inside the KYC Agent node (line 828 in current file).
2. **Replace** the entire string value with the new prompt above.
3. **Preserve** the n8n expression prefix `=` at the start of the value (the field uses `"systemMessage": "=..."` pattern).

#### 2.4 Key behavioral changes

| Behavior | Before | After |
|----------|--------|-------|
| Email collection timing | Phase 1 (first message) | Phase 8 (last step before finalization) |
| Current frequency collection | Implicitly inferred by agent | Explicitly collected in Phase 4 |
| Days available | User picks freely in Phase 5 | Tool recommends, user accepts/overrides in Phase 5b |
| Finalization trigger | After Cardio data | After Email confirmed |
| Phase 5 structure | Single block (days + duration + schedule) | Three sub-phases (5a duration, 5b recommendation, 5c schedule) |

---

### Phase 3: Update E2E Test Simulator

**Target file:** `n8n/tests/GymRatFlow_E2E_TestRunner.json`
**Target location:** `Simulate User Response` node (ID `a5e695c0-2b3b-4ea0-8663-8834a47e2c72`), field `parameters.jsonBody` (line 69).

#### 3.1 Update the simulated user system prompt

In the `jsonBody` field, update the `REGLAS` section of the system prompt. The current rules are:

```
REGLAS:
1. Responde SOLO lo que te pregunten. Si preguntan nombre y correo, da ambos.
2. Se conciso y natural...
...
```

Replace with the following rules:

```
REGLAS:
1. Responde SOLO lo que te pregunten. Si preguntan solo nombre, da solo el nombre. El correo se pide al final del proceso, no lo des hasta que te lo pidan explicitamente.
2. Se conciso y natural, como un usuario real de WhatsApp (1-2 oraciones maximo).
3. Si dan opciones (A, B, C...), responde con la letra que corresponda a tu perfil.
4. No inventes datos que no esten en tu perfil.
5. Si te piden confirmar o despedirse, responde amablemente.
6. NO uses emojis excesivos, solo ocasionalmente.
7. IMPORTANTE para HOME: Cuando te pregunten donde vas a entrenar, responde segun tu perfil: 'En casa' o 'Gimnasio'.
8. IMPORTANTE para equipamiento: Cuando pregunten que equipamiento tienes en casa, responde EXACTAMENTE lo que dice tu perfil.
9. DURACION DE SESION: Cuando pregunten cuanto tiempo por sesion, responde con tu dato de tiempo_por_sesion.
10. RECOMENDACION DE DIAS: Si el coach recomienda un numero de dias y coincide con tu perfil (dias_disponibles), acepta la recomendacion. Si el numero recomendado NO coincide con tu perfil, responde que prefieres entrenar el numero de dias de tu perfil.
11. EMAIL: Cuando te pidan el correo electronico al final del proceso, dalo. Cuando te pidan confirmar el correo, confirma que es correcto.
```

#### 3.2 Specific string replacement

In the JSON body string, find:

```
REGLAS:\\n1. Responde SOLO lo que te pregunten. Si preguntan nombre y correo, da ambos.
```

Replace the full `REGLAS` block (rules 1 through 8) with the updated rules (1 through 11) from section 3.1 above, preserving the `\\n` line-break encoding used in the JSON string.

#### 3.3 No changes to test case definitions

The existing `simulatedUser` objects in test case definitions (TC002_FULL_KYC, TC_HOME_FULL_BASIC, TC_HOME_FULL_BODYWEIGHT, TC_HOME_FULL_HEALTH_C) already contain all required fields:
- `frecuencia_actual` -- already present
- `dias_disponibles` -- already present
- `tiempo_por_sesion` -- already present
- `email` -- already present

No changes are needed to the test case data structures. The simulated user prompt rules are the only update.

#### 3.4 Completion indicators

Review existing `completionIndicators` arrays. The current indicators already match the new prompt's farewell message pattern ("Preparate para darlo todo", "me pongo manos a la obra", etc.). No changes needed.

---

### Phase 4: Verification

#### 4.1 Import and manual test

1. Import the updated `MAIN_FLOW.json` into the n8n instance.
2. Verify the `Tool_Session_Recommendation` node appears on the canvas near the KYC Agent.
3. Verify the node is connected to the KYC Agent (visible connection line to `ai_tool` input).
4. Send a manual WhatsApp message to trigger a fresh KYC flow.
5. Walk through the entire conversation and verify:

| Checkpoint | Expected behavior |
|------------|-------------------|
| Phase 1 | Agent asks for name ONLY (no email) |
| Phase 2 | Agent asks age, sex, height, weight |
| Phase 3 | Agent asks primary + secondary goal |
| Phase 4 | Agent asks experience, current frequency, level, then health |
| Phase 5a | Agent asks session duration ONLY |
| Phase 5b | Agent calls `Tool_Session_Recommendation`, presents recommendation, asks for confirmation |
| Phase 5b (override) | If user says different number, agent accepts without insisting |
| Phase 5c | Agent asks schedule (Manana/Tarde/Noche) |
| Phase 6 | Agent asks training style, priority, dislikes, environment |
| Phase 7 | Agent asks cardio + frequency |
| Phase 8 | Agent asks email with justification, then validates |
| Finalization | Agent calls `Tool_Create_User_Profile` with all 24 fields |

6. Check Supabase `users_gym_profile` table to confirm all fields are populated correctly.

#### 4.2 E2E test suite

1. Run `test_data_setup.sql` to reset all fixture users.
2. Execute the E2E test runner with filter: `['TC002_FULL_KYC']` first (fastest feedback).
3. If TC002_FULL_KYC passes, run the full suite including HOME tests:

```javascript
const TEST_FILTER = ['TC002_FULL_KYC', 'TC_HOME_FULL_BASIC', 'TC_HOME_FULL_BODYWEIGHT', 'TC_HOME_FULL_HEALTH_C'];
```

4. Verify all tests pass with existing verification queries (user created, plan created, 4 weeks of workouts).

#### 4.3 Regression checks

Run the full test suite (no filter) to ensure non-KYC tests are unaffected:

| Test | Risk | Why |
|------|------|-----|
| TC001 (noise filter) | None | Does not touch KYC Agent |
| TC003 (scheduling) | None | Existing user, skips KYC |
| TC004 (rest day) | None | Existing user, skips KYC |
| TC006, TC007, TC013 | None | Existing users with routines |
| TC011, TC012 | None | Pending task flow, not KYC |
| TC_MESO_* | None | Mesocycle renewal, not KYC |

#### 4.4 Tool_Create_User_Profile field validation

Verify that the final `Tool_Create_User_Profile` call includes all 24 `$fromAI` fields. The critical fields to check after the prompt rewrite:

| Field | $fromAI key | Source phase |
|-------|-------------|--------------|
| `full_name` | `nombre_del_usuario` | Phase 1 |
| `email` | `correo_electronico` | Phase 8 (moved) |
| `age` | `edad` | Phase 2 |
| `biological_sex` | `sexo` | Phase 2 |
| `height_cm` | `estatura` | Phase 2 |
| `weight_kg` | `peso` | Phase 2 |
| `primary_goal` | `objetivo_principal` | Phase 3 |
| `secondary_goal` | `objetivo_secundario` | Phase 3 |
| `training_experience` | `experiencia` | Phase 4 |
| `current_frequency` | `frecuencia_actual` | Phase 4 (now explicit) |
| `fitness_level` | `nivel_autopercibido` | Phase 4 |
| `health_status` | `estado_de_salud` | Phase 4 |
| `days_available` | `dias_disponibles_para_entrenar` | Phase 5b (recommendation result) |
| `session_duration_mins` | `tiempo_por_sesion` | Phase 5a |
| `preferred_schedule` | `horario_habitual` | Phase 5c |
| `training_style` | `tipo_de_entrenamiento_preferido` | Phase 6 |
| `priority_muscles` | `partes_del_cuerpo_a_priorizar` | Phase 6 |
| `disliked_exercises` | `partes_que_no_le_gustan` | Phase 6 |
| `cardio_type` | `tipo_de_cardio_que_realiza` | Phase 7 |
| `cardio_frequency` | `frecuencia_de_cardio` | Phase 7 |
| `training_environment` | `ambiente_entrenamiento` | Phase 6 |
| `home_equipment` | `equipamiento_disponible` | Phase 6 (conditional) |

**No changes are needed to `Tool_Create_User_Profile`** -- the `$fromAI` keys remain the same. The prompt rewrite only changes when data is collected, not the field names.

---

### Appendix A: Recommendation Matrix Examples

Example calculations to validate the tool logic:

| Scenario | Experience | Goal | Frequency | Age | Health | Duration | Base | Mods | Result |
|----------|-----------|------|-----------|-----|--------|----------|------|------|--------|
| Beginner, healthy | Nunca | Salud general | No entreno | 30 | A | 45-60 min | 2 | +0 goal, -1 freq, +0 age, +0 health, +0 dur = **1** -> clamped **2** | 2 days (fb_2) |
| Intermediate gym-goer | 6 a 12 meses | Ganar masa | 3-4 dias | 25 | A | 60-75 min | 3 | +1 goal, +0 freq, +0 age, +0 health, +0 dur = **4** | 4 days (ul_4) |
| Advanced lifter | Mas de 3 anos | Mejorar fuerza | 5-6 dias | 28 | A | 60-75 min | 5 | +1 goal, +1 freq, +0 age, +0 health, +0 dur = **7** -> clamped **6** | 6 days (ppl_6) |
| Older beginner w/injury | Menos de 6 meses | Bajar grasa | 1-2 dias | 55 | B | 30-45 min | 3 | +0 goal, +0 freq, -1 age, -1 health, +1 dur = **2** | 2 days (fb_2) |
| Home user, health C | 1 a 3 anos | Bajar grasa | 3-4 dias | 40 | C | 45-60 min | 4 | +0 goal, +0 freq, +0 age, -1 health, +0 dur = **3** | 3 days (fb_3) |

### Appendix B: Rollback Plan

If the recommendation tool causes issues in production:

1. **Quick fix:** Remove the `Tool_Session_Recommendation` connection from the `connections` object (revert to single-tool KYC Agent).
2. **Prompt rollback:** Restore the previous system prompt (Phase 5 returns to single block with days + duration + schedule).
3. **The `Tool_Session_Recommendation` node can remain in the workflow** (orphaned nodes do not execute in n8n). Remove it in a follow-up cleanup.
