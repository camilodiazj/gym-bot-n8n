#!/usr/bin/env node
/**
 * Phase 4 Implementation Script
 * Modifies GymRatFlow_Supabase_V2_Workout_Tracker.json to add:
 * - HTTP_Check_Mesocycle_Status node
 * - If_Mesocycle_Complete conditional node
 * - Execute_Mesocycle_Renewal workflow node
 * - Updated Intention_Agent system prompt with RENOVAR_MESOCICLO
 * - RENOVAR_MESOCICLO branch in Switch node
 */

const fs = require('fs');
const path = require('path');

const WORKFLOW_PATH = path.join(__dirname, '../n8n/running_flows/GymRatFlow_Supabase_V2_Workout_Tracker.json');
const OUTPUT_PATH = WORKFLOW_PATH; // Overwrite the original

// Load the workflow
const workflow = JSON.parse(fs.readFileSync(WORKFLOW_PATH, 'utf8'));

console.log('Phase 4: n8n Main Flow Integration');
console.log('===================================');
console.log(`Loaded workflow: ${workflow.name}`);
console.log(`Current nodes count: ${workflow.nodes.length}`);

// ============================================================================
// Task 4.1: Add HTTP_Check_Mesocycle_Status Node
// ============================================================================
console.log('\nTask 4.1: Adding HTTP_Check_Mesocycle_Status node...');

const httpCheckMesocycleNode = {
  "parameters": {
    "method": "GET",
    "url": "={{ $env.WORKOUT_API_URL }}/api/v1/plans/{{ $items('GetUser')[0].json.user_id }}/mesocycle-status",
    "authentication": "predefinedCredentialType",
    "nodeCredentialType": "httpHeaderAuth",
    "sendHeaders": true,
    "headerParameters": {
      "parameters": [
        {
          "name": "Content-Type",
          "value": "application/json"
        }
      ]
    },
    "options": {
      "timeout": 10000
    }
  },
  "type": "n8n-nodes-base.httpRequest",
  "typeVersion": 4.2,
  "position": [
    -4400,
    25600
  ],
  "id": "http-check-mesocycle-status",
  "name": "HTTP_Check_Mesocycle_Status",
  "alwaysOutputData": true,
  "onError": "continueRegularOutput",
  "credentials": {
    "httpHeaderAuth": {
      "id": "INTERNAL_API_KEY_CREDENTIAL_ID",
      "name": "Workout API Internal"
    }
  }
};

workflow.nodes.push(httpCheckMesocycleNode);
console.log('  Added HTTP_Check_Mesocycle_Status node');

// ============================================================================
// Task 4.2: Add If_Mesocycle_Complete Conditional Node
// ============================================================================
console.log('\nTask 4.2: Adding If_Mesocycle_Complete conditional node...');

const ifMesocycleCompleteNode = {
  "parameters": {
    "conditions": {
      "options": {
        "caseSensitive": true,
        "leftValue": "",
        "typeValidation": "strict",
        "version": 3
      },
      "conditions": [
        {
          "id": "mesocycle-complete-check",
          "leftValue": "={{ $json.is_complete }}",
          "rightValue": "",
          "operator": {
            "type": "boolean",
            "operation": "true",
            "singleValue": true
          }
        }
      ],
      "combinator": "and"
    },
    "options": {}
  },
  "type": "n8n-nodes-base.if",
  "typeVersion": 2.3,
  "position": [
    -4200,
    25600
  ],
  "id": "if-mesocycle-complete",
  "name": "If_Mesocycle_Complete",
  "alwaysOutputData": true
};

workflow.nodes.push(ifMesocycleCompleteNode);
console.log('  Added If_Mesocycle_Complete node');

// ============================================================================
// Task 4.5: Add Execute_Mesocycle_Renewal Workflow Node
// ============================================================================
console.log('\nTask 4.5: Adding Execute_Mesocycle_Renewal workflow node...');

const executeMesocycleRenewalNode = {
  "parameters": {
    "workflowId": {
      "__rl": true,
      "value": "MESOCYCLE_RENEWAL_WORKFLOW_ID",
      "mode": "list",
      "cachedResultUrl": "/workflow/MESOCYCLE_RENEWAL_WORKFLOW_ID",
      "cachedResultName": "GymBotMesocycleRenewal"
    },
    "workflowInputs": {
      "mappingMode": "defineBelow",
      "value": {
        "user_id": "={{ $items('GetUser')[0].json.user_id }}",
        "whatsapp_id": "={{ $items('If')[0].json.contacts[0].wa_id }}",
        "phone_number_id": "={{ $items('If')[0].json.metadata.phone_number_id }}",
        "current_mesocycle": "={{ $json.mesocycle_number || 1 }}",
        "current_days_per_week": "={{ $json.days_per_week || 3 }}",
        "user_name": "={{ $items('GetUser')[0].json.full_name }}",
        "user_message": "={{ $items('If')[0].json.messages[0].text.body }}"
      },
      "matchingColumns": [],
      "schema": [
        {"id": "user_id", "displayName": "user_id", "required": true, "type": "string"},
        {"id": "whatsapp_id", "displayName": "whatsapp_id", "required": true, "type": "string"},
        {"id": "phone_number_id", "displayName": "phone_number_id", "required": true, "type": "string"},
        {"id": "current_mesocycle", "displayName": "current_mesocycle", "required": true, "type": "number"},
        {"id": "current_days_per_week", "displayName": "current_days_per_week", "required": true, "type": "number"},
        {"id": "user_name", "displayName": "user_name", "required": true, "type": "string"},
        {"id": "user_message", "displayName": "user_message", "required": true, "type": "string"}
      ]
    },
    "options": {
      "waitForSubWorkflow": true
    }
  },
  "type": "n8n-nodes-base.executeWorkflow",
  "typeVersion": 1.3,
  "position": [
    -4000,
    25500
  ],
  "id": "execute-mesocycle-renewal",
  "name": "Execute_Mesocycle_Renewal",
  "alwaysOutputData": true,
  "onError": "continueRegularOutput"
};

workflow.nodes.push(executeMesocycleRenewalNode);
console.log('  Added Execute_Mesocycle_Renewal node');

// Add a placeholder output node for the renewal workflow result
const renewalOutputNode = {
  "parameters": {
    "jsCode": "return [\n  {\n    json: {\n      output: $('Execute_Mesocycle_Renewal').first().json.output || 'Renewal completed'\n    }\n  }\n];"
  },
  "type": "n8n-nodes-base.code",
  "typeVersion": 2,
  "position": [
    -3800,
    25500
  ],
  "id": "filtered-message-renewal",
  "name": "Filtered Message Renewal"
};

workflow.nodes.push(renewalOutputNode);
console.log('  Added Filtered Message Renewal output node');

// ============================================================================
// Task 4.3: Update Intention_Agent System Prompt
// ============================================================================
console.log('\nTask 4.3: Updating Intention_Agent system prompt...');

const intentionAgentIndex = workflow.nodes.findIndex(n => n.name === 'Intention_Agent');
if (intentionAgentIndex !== -1) {
  // Read the new system prompt from the file we created
  const newIntentionPrompt = `=Eres un agente encargado de evaluar la intencion del usuario segun el mensaje que recibes.

INTENCIONES VALIDAS:
- VER_RUTINA_DE_HOY
- CHAT
- CONFIRMAR_RUTINA
- RENOVAR_MESOCICLO

VER_RUTINA_DE_HOY: El usuario quiere ver su rutina/entrenamiento del dia.
Ejemplos: "Muestrame mi rutina", "Que me toca hoy", "Mi entrenamiento", "Dame mi workout", "Cual es mi rutina", "Que ejercicios tengo"

CHAT: Cualquier otra pregunta, comentario o conversacion general sobre fitness que NO sea sobre ver rutina, renovar o confirmar.
Ejemplos: "Que ejercicio es mejor para biceps", "Hola", "Gracias", "Como hago un peso muerto", "Cuantas calorias debo comer"

CONFIRMAR_RUTINA: El usuario esta respondiendo a una pregunta sobre si completo su rutina de hoy.
NOTA: Esta intencion SOLO aplica cuando el contexto indica que hay una pending_task activa.
Ejemplos: "Si, termine", "Ya complete mi rutina", "No pude entrenar hoy", "La hice"

RENOVAR_MESOCICLO: El usuario quiere cambiar, renovar o modificar su plan de entrenamiento para el siguiente ciclo.
Palabras clave: "renovar", "cambiar plan", "cambiar rutina", "nuevo ciclo", "nuevos ejercicios", "cambiar ejercicios", "rotar ejercicios", "cambiar dias", "entrenar mas dias", "entrenar menos dias", "modificar mi plan", "actualizar rutina"
Ejemplos: "Quiero renovar mi rutina", "Puedo cambiar mis ejercicios?", "Ya termine las 4 semanas, que sigue?", "Quiero entrenar 5 dias en vez de 3", "Estoy aburrido de los mismos ejercicios", "Quiero empezar un nuevo ciclo"

REGLAS DE DIFERENCIACION:
- Si el usuario PREGUNTA sobre como cambiar/renovar -> RENOVAR_MESOCICLO
- Si el usuario quiere VER la rutina actual -> VER_RUTINA_DE_HOY
- Si el usuario quiere CAMBIAR/MODIFICAR la rutina -> RENOVAR_MESOCICLO
- Si menciona ver/mostrar sin cambiar -> VER_RUTINA_DE_HOY
- Si menciona cambiar/nuevo/renovar/rotar -> RENOVAR_MESOCICLO

Retorna SOLO la intencion (VER_RUTINA_DE_HOY, CHAT, CONFIRMAR_RUTINA o RENOVAR_MESOCICLO), sin explicacion adicional.`;

  workflow.nodes[intentionAgentIndex].parameters.options.systemMessage = newIntentionPrompt;
  console.log('  Updated Intention_Agent system prompt with RENOVAR_MESOCICLO intent');
} else {
  console.log('  WARNING: Intention_Agent node not found!');
}

// ============================================================================
// Task 4.4: Add RENOVAR_MESOCICLO Branch to Switch Node
// ============================================================================
console.log('\nTask 4.4: Adding RENOVAR_MESOCICLO branch to Switch node...');

const switchNodeIndex = workflow.nodes.findIndex(n => n.name === 'Switch');
if (switchNodeIndex !== -1) {
  const switchNode = workflow.nodes[switchNodeIndex];

  // Add the new RENOVAR_MESOCICLO rule
  const renovarMesocicloRule = {
    "conditions": {
      "options": {
        "caseSensitive": true,
        "leftValue": "",
        "typeValidation": "strict",
        "version": 3
      },
      "conditions": [
        {
          "id": "renovar-mesociclo-condition",
          "leftValue": "={{ $json.output.trim() }}",
          "rightValue": "RENOVAR_MESOCICLO",
          "operator": {
            "type": "string",
            "operation": "equals"
          }
        }
      ],
      "combinator": "and"
    },
    "renameOutput": true,
    "outputKey": "RENOVAR_MESOCICLO"
  };

  switchNode.parameters.rules.values.push(renovarMesocicloRule);
  console.log('  Added RENOVAR_MESOCICLO branch to Switch node');
} else {
  console.log('  WARNING: Switch node not found!');
}

// ============================================================================
// Update Connections
// ============================================================================
console.log('\nUpdating connections...');

// 1. Modify has_planned_workouts1 FALSE branch to go to HTTP_Check_Mesocycle_Status
// Current: has_planned_workouts1 FALSE -> Week_Schedule, User_Finished_Workouts, Template_Days
// New: has_planned_workouts1 FALSE -> HTTP_Check_Mesocycle_Status

// Find and update the has_planned_workouts1 connections
if (workflow.connections['has_planned_workouts1']) {
  // Keep TRUE branch (index 0), modify FALSE branch (index 1)
  workflow.connections['has_planned_workouts1'].main[1] = [
    {
      "node": "HTTP_Check_Mesocycle_Status",
      "type": "main",
      "index": 0
    }
  ];
  console.log('  Updated has_planned_workouts1 FALSE -> HTTP_Check_Mesocycle_Status');
}

// 2. Add HTTP_Check_Mesocycle_Status -> If_Mesocycle_Complete
workflow.connections['HTTP_Check_Mesocycle_Status'] = {
  "main": [
    [
      {
        "node": "If_Mesocycle_Complete",
        "type": "main",
        "index": 0
      }
    ]
  ]
};
console.log('  Added HTTP_Check_Mesocycle_Status -> If_Mesocycle_Complete');

// 3. Add If_Mesocycle_Complete connections
// TRUE -> Execute_Mesocycle_Renewal
// FALSE -> Week_Schedule, User_Finished_Workouts, Template_Days (original scheduling flow)
workflow.connections['If_Mesocycle_Complete'] = {
  "main": [
    [
      {
        "node": "Execute_Mesocycle_Renewal",
        "type": "main",
        "index": 0
      }
    ],
    [
      {
        "node": "Week_Schedule",
        "type": "main",
        "index": 0
      },
      {
        "node": "User_Finished_Workouts",
        "type": "main",
        "index": 0
      },
      {
        "node": "Template_Days",
        "type": "main",
        "index": 0
      }
    ]
  ]
};
console.log('  Added If_Mesocycle_Complete TRUE -> Execute_Mesocycle_Renewal');
console.log('  Added If_Mesocycle_Complete FALSE -> Week_Schedule, User_Finished_Workouts, Template_Days');

// 4. Add Execute_Mesocycle_Renewal -> Filtered Message Renewal
workflow.connections['Execute_Mesocycle_Renewal'] = {
  "main": [
    [
      {
        "node": "Filtered Message Renewal",
        "type": "main",
        "index": 0
      }
    ]
  ]
};
console.log('  Added Execute_Mesocycle_Renewal -> Filtered Message Renewal');

// 5. Add Switch RENOVAR_MESOCICLO output -> Execute_Mesocycle_Renewal
// The Switch node has 3 existing outputs (ROUTINE_CONFIRMATION, CHAT, VER_RUTINA_DE_HOY)
// We need to add a 4th output for RENOVAR_MESOCICLO
if (workflow.connections['Switch']) {
  workflow.connections['Switch'].main.push([
    {
      "node": "Execute_Mesocycle_Renewal",
      "type": "main",
      "index": 0
    }
  ]);
  console.log('  Added Switch RENOVAR_MESOCICLO -> Execute_Mesocycle_Renewal');
}

// ============================================================================
// Save the modified workflow
// ============================================================================
console.log('\nSaving modified workflow...');
fs.writeFileSync(OUTPUT_PATH, JSON.stringify(workflow, null, 2));
console.log(`  Saved to: ${OUTPUT_PATH}`);

console.log('\n===================================');
console.log('Phase 4 Implementation Complete!');
console.log(`Final nodes count: ${workflow.nodes.length}`);
console.log('\nNOTE: You will need to update the following placeholder values in n8n:');
console.log('  1. INTERNAL_API_KEY_CREDENTIAL_ID - Set to your actual httpHeaderAuth credential ID');
console.log('  2. MESOCYCLE_RENEWAL_WORKFLOW_ID - Set to the actual GymBotMesocycleRenewal workflow ID');
console.log('  3. WORKOUT_API_URL environment variable must be set in n8n');
console.log('\nVerification steps:');
console.log('  1. Import the updated workflow into n8n');
console.log('  2. Configure credentials for HTTP_Check_Mesocycle_Status');
console.log('  3. Link Execute_Mesocycle_Renewal to the actual GymBotMesocycleRenewal workflow');
console.log('  4. Test with message "Quiero renovar mi rutina" - should detect RENOVAR_MESOCICLO');
console.log('  5. Test with a user who has completed week 4 - should auto-trigger renewal');
