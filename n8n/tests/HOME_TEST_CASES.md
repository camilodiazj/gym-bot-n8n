# HOME E2E Test Cases para GymRatFlow

Estos test cases deben agregarse al array `testCases` en el nodo **"Load Test Cases"** del workflow `GymRatFlow_E2E_TestRunner.json`.

## Instrucciones de Instalacion

1. Abrir n8n y cargar el workflow `GymRatFlow_E2E_TestRunner`
2. Editar el nodo **"Load Test Cases"** (Code node)
3. Agregar estos 3 test cases al final del array `testCases` (despues de TC013)
4. Editar el nodo **"Simulate User Response"** y **"Build AI Turn Input"** segun las instrucciones al final

---

## Test Case 1: TC_HOME_FULL_BASIC

```javascript
  {
    order: 11,
    id: "TC_HOME_FULL_BASIC",
    name: "Onboarding HOME Completo - Equipamiento Basico (Mancuernas + Bandas)",
    priority: "CRITICAL",
    category: "ONBOARDING_HOME",
    testType: "MULTI_TURN_AI",
    phone: "570000000211",
    simulatedUser: {
      nombre: "Maria Lopez Sanchez",
      email: "maria.lopez.home.e2e@test.com",
      edad: 32,
      sexo: "F",
      estatura_cm: 165,
      peso_kg: 62,
      objetivo_principal: "Salud general / recomposicion corporal",
      objetivo_secundario: "Tonificar",
      tiempo_entrenando: "Menos de 6 meses",
      frecuencia_actual: "1-2 dias por semana",
      nivel: "Principiante",
      estado_salud: "A",
      dias_disponibles: 3,
      tiempo_por_sesion: "45-60 minutos",
      horario: "Manana",
      tipo_entrenamiento: "Mixto",
      prioridades: "Gluteo y pierna",
      desafios: "Nada en particular",
      ambiente_entrenamiento: "En casa",
      equipamiento_casa: "Mancuernas y bandas elasticas",
      cardio_actual: "Caminata",
      frecuencia_cardio: "1-2"
    },
    config: {
      maxTurns: 30,
      completionIndicators: ["me pongo manos a la obra", "recibiras tu plan", "tu rutina 100%", "estoy emocionado por ver tu progreso", "en breve recibiras", "disenar tu rutina"],
      firstMessage: "Hola! Quiero empezar a entrenar en casa"
    },
    cleanup: [
      "DELETE FROM n8n_chat_histories WHERE session_id LIKE '570000000211%';",
      "DELETE FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000211');",
      "DELETE FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000211');",
      "DELETE FROM pending_tasks WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000211');",
      "DELETE FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000211');",
      "DELETE FROM users WHERE full_phone_number = '570000000211';",
      "DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000211;"
    ],
    verification: {
      queries: [
        { sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000211", expected: 1 },
        { sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000211 AND training_environment = 'HOME'", expected: 1 },
        { sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000211 AND home_equipment IS NOT NULL AND home_equipment != ''", expected: 1 },
        { sql: "SELECT COUNT(*) as cnt FROM users WHERE full_phone_number = '570000000211'", expected: 1 },
        { sql: "SELECT COUNT(*) as cnt FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000211')", expected: 1 },
        { sql: "SELECT COUNT(DISTINCT week) as cnt FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000211')", expected: 4 },
        { sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id WHERE w.user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000211') AND e.equipment IN ('machine', 'cable', 'smith')", expected: 0 }
      ]
    },
    metrics: {
      rule: "passed === true",
      description: "Debe completar KYC HOME con mancuernas+bandas y generar rutina sin ejercicios de maquina"
    }
  }
```

---

## Test Case 2: TC_HOME_FULL_BODYWEIGHT

```javascript
  {
    order: 12,
    id: "TC_HOME_FULL_BODYWEIGHT",
    name: "Onboarding HOME Completo - Solo Peso Corporal",
    priority: "HIGH",
    category: "ONBOARDING_HOME",
    testType: "MULTI_TURN_AI",
    phone: "570000000212",
    simulatedUser: {
      nombre: "Carlos Ramirez Diaz",
      email: "carlos.ramirez.home.e2e@test.com",
      edad: 25,
      sexo: "M",
      estatura_cm: 178,
      peso_kg: 70,
      objetivo_principal: "Mejorar resistencia",
      objetivo_secundario: "Mantenerme activo",
      tiempo_entrenando: "Nunca he entrenado",
      frecuencia_actual: "No entreno",
      nivel: "Principiante",
      estado_salud: "A",
      dias_disponibles: 4,
      tiempo_por_sesion: "30-45 minutos",
      horario: "Noche",
      tipo_entrenamiento: "Funcional",
      prioridades: "Core y resistencia general",
      desafios: "Ninguno",
      ambiente_entrenamiento: "En casa",
      equipamiento_casa: "Solo tengo mi cuerpo, no tengo equipamiento",
      cardio_actual: "No",
      frecuencia_cardio: "0"
    },
    config: {
      maxTurns: 30,
      completionIndicators: ["me pongo manos a la obra", "recibiras tu plan", "tu rutina 100%", "estoy emocionado por ver tu progreso", "en breve recibiras", "disenar tu rutina"],
      firstMessage: "Hola, quiero empezar a entrenar pero solo tengo mi cuerpo, no tengo equipamiento"
    },
    cleanup: [
      "DELETE FROM n8n_chat_histories WHERE session_id LIKE '570000000212%';",
      "DELETE FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000212');",
      "DELETE FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000212');",
      "DELETE FROM pending_tasks WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000212');",
      "DELETE FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000212');",
      "DELETE FROM users WHERE full_phone_number = '570000000212';",
      "DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000212;"
    ],
    verification: {
      queries: [
        { sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000212", expected: 1 },
        { sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000212 AND training_environment = 'HOME'", expected: 1 },
        { sql: "SELECT COUNT(*) as cnt FROM users WHERE full_phone_number = '570000000212'", expected: 1 },
        { sql: "SELECT COUNT(*) as cnt FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000212')", expected: 1 },
        { sql: "SELECT COUNT(DISTINCT week) as cnt FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000212')", expected: 4 },
        { sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id WHERE w.user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000212') AND e.equipment NOT IN ('bodyweight')", expected: 0 }
      ]
    },
    metrics: {
      rule: "passed === true",
      description: "Debe completar KYC HOME bodyweight-only y generar rutina SOLO con ejercicios bodyweight"
    }
  }
```

---

## Test Case 3: TC_HOME_FULL_HEALTH_C

```javascript
  {
    order: 13,
    id: "TC_HOME_FULL_HEALTH_C",
    name: "Onboarding HOME Completo - Restriccion Upper Body (Health C)",
    priority: "HIGH",
    category: "ONBOARDING_HOME_HEALTH",
    testType: "MULTI_TURN_AI",
    phone: "570000000213",
    simulatedUser: {
      nombre: "Ana Garcia Torres",
      email: "ana.garcia.home.e2e@test.com",
      edad: 40,
      sexo: "F",
      estatura_cm: 160,
      peso_kg: 65,
      objetivo_principal: "Bajar grasa",
      objetivo_secundario: "Fortalecer piernas",
      tiempo_entrenando: "1 a 3 anos",
      frecuencia_actual: "3-4 dias por semana",
      nivel: "Intermedio",
      estado_salud: "C",
      dias_disponibles: 4,
      tiempo_por_sesion: "45-60 minutos",
      horario: "Tarde",
      tipo_entrenamiento: "Mixto",
      prioridades: "Pierna y gluteo",
      desafios: "Hombros, me lastimo facilmente",
      ambiente_entrenamiento: "En casa",
      equipamiento_casa: "Mancuernas y bandas elasticas",
      cardio_actual: "Bicicleta",
      frecuencia_cardio: "3-4"
    },
    config: {
      maxTurns: 30,
      completionIndicators: ["me pongo manos a la obra", "recibiras tu plan", "tu rutina 100%", "estoy emocionado por ver tu progreso", "en breve recibiras", "disenar tu rutina"],
      firstMessage: "Hola! Quiero entrenar en casa pero tengo una molestia en los hombros"
    },
    cleanup: [
      "DELETE FROM n8n_chat_histories WHERE session_id LIKE '570000000213%';",
      "DELETE FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000213');",
      "DELETE FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000213');",
      "DELETE FROM pending_tasks WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000213');",
      "DELETE FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000213');",
      "DELETE FROM users WHERE full_phone_number = '570000000213';",
      "DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000213;"
    ],
    verification: {
      queries: [
        { sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000213", expected: 1 },
        { sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000213 AND training_environment = 'HOME'", expected: 1 },
        { sql: "SELECT COUNT(*) as cnt FROM users_gym_profile WHERE whatsapp_id = 570000000213 AND health_status = 'C'", expected: 1 },
        { sql: "SELECT COUNT(*) as cnt FROM users WHERE full_phone_number = '570000000213'", expected: 1 },
        { sql: "SELECT COUNT(*) as cnt FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000213')", expected: 1 },
        { sql: "SELECT COUNT(DISTINCT week) as cnt FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000213')", expected: 4 },
        { sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id WHERE w.user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000213') AND e.equipment IN ('machine', 'cable', 'smith')", expected: 0 },
        { sql: "SELECT COUNT(*) as cnt FROM workouts w JOIN exercises e ON w.exercise_id = e.exercise_id WHERE w.user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000213') AND e.pattern = 'push_v' AND e.spanish_name ILIKE '%press%hombro%'", expected: 0 }
      ]
    },
    metrics: {
      rule: "passed === true",
      description: "Debe completar KYC HOME con health_status=C y generar rutina sin ejercicios overhead/maquina"
    }
  }
```

---

## Modificacion Requerida: Nodo "Build AI Turn Input"

Actualizar el codigo para usar el phone del test case en lugar de hardcoded:

```javascript
// Build AI Turn Input - construye el mensaje WhatsApp para el turno actual
const testCase = $('Pre-Test Log').first().json;
const turnData = $input.first().json;
const turnNumber = turnData.turnNumber || 1;
const message = turnData.currentMessage;

// Usar phone del test case o default a 570000000009
const phone = testCase.phone || "570000000009";

console.log(`   📤 AI Turn ${turnNumber}: "${message.substring(0, 60)}..."`);

return [{
  json: {
    messaging_product: "whatsapp",
    metadata: {
      display_phone_number: "573213413664",
      phone_number_id: "914510145083991"
    },
    contacts: [{
      profile: { name: testCase.simulatedUser?.nombre || "Test KYC User" },
      wa_id: phone
    }],
    messages: [{
      from: phone,
      id: `wamid.E2E-${testCase.id}-TURN${turnNumber}`,
      timestamp: String(Math.floor(Date.now() / 1000)),
      text: { body: message },
      type: "text"
    }],
    field: "messages"
  }
}];
```

---

## Modificacion Requerida: Nodo "Simulate User Response"

Agregar estos campos al system prompt del usuario simulado (en el jsonBody del HTTP Request):

```
- Ambiente de entrenamiento: {{ $('Pre-Test Log').first().json.simulatedUser.ambiente_entrenamiento || 'Gimnasio' }}
- Equipamiento en casa: {{ $('Pre-Test Log').first().json.simulatedUser.equipamiento_casa || 'N/A' }}
```

Y agregar estas reglas:

```
7. IMPORTANTE para HOME: Cuando te pregunten donde vas a entrenar, responde segun tu perfil: 'En casa' o 'Gimnasio'.
8. IMPORTANTE para equipamiento: Cuando pregunten que equipamiento tienes en casa, responde EXACTAMENTE lo que dice tu perfil.
```

---

## Phones Reservados para Tests HOME

| Phone | Test ID | Descripcion |
|-------|---------|-------------|
| `570000000211` | TC_HOME_FULL_BASIC | HOME + mancuernas + bandas |
| `570000000212` | TC_HOME_FULL_BODYWEIGHT | HOME + solo peso corporal |
| `570000000213` | TC_HOME_FULL_HEALTH_C | HOME + health_status=C |

---

## Verificaciones de BD

Cada test verifica:

1. **Profile creado** con `training_environment = 'HOME'`
2. **home_equipment** capturado correctamente
3. **4 semanas de workouts** generados
4. **0 ejercicios de maquina** (machine, cable, smith)
5. **Bodyweight-only test**: 0 ejercicios con equipment != bodyweight
6. **Health C test**: 0 ejercicios de press de hombros overhead
