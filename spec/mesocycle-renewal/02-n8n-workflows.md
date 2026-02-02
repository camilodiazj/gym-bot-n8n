# n8n Workflow Specification - Mesocycle Renewal

## Table of Contents

1. [Overview](#overview)
2. [Main Flow Modifications (GymRatFlow)](#main-flow-modifications-gymratflow)
3. [Renewal Subflow Updates (GymBotMesocycleRenewal)](#renewal-subflow-updates-gymbotmesocyclerenewal)
4. [GymRatForm Integration](#gymratform-integration)
5. [Memory Management](#memory-management)
6. [Error Handling](#error-handling)
7. [Node JSON Snippets](#node-json-snippets)

---

## Overview

This document specifies the n8n workflow modifications required for the mesocycle renewal feature. There are two workflows to modify:

1. **GymRatFlow_Supabase_V2_Workout_Tracker.json** - Main orchestrator (add detection + new intent)
2. **GymBotMesocycleRenewal.json** - Renewal subflow (replace SQL with HTTP calls)

### Architecture

```
                         WhatsApp Message
                               │
                               ▼
        ┌──────────────────────────────────────────────┐
        │           GymRatFlow (Main Flow)             │
        │                                              │
        │  [WhatsApp_Trigger1]                         │
        │         │                                    │
        │         ▼                                    │
        │  [If: valid message?]───FALSE───► [filter]   │
        │         │ TRUE                               │
        │         ▼                                    │
        │  [GetUser]                                   │
        │         │                                    │
        │         ▼                                    │
        │  [user_exists?]────FALSE────► [KYC Agent]    │
        │         │ TRUE                               │
        │         ▼                                    │
        │  [Has_Pending_Task?]───TRUE──► [CONFIRMATION]│
        │         │ FALSE                              │
        │         ▼                                    │
        │  [GetWeeklySchedule]                         │
        │         │                                    │
        │         ▼                                    │
        │  [has_planned_workouts?]                     │
        │         │                                    │
        │  TRUE   │    FALSE                           │
        │    │    │      │                             │
        │    ▼    │      ▼                             │
        │ [Filter │  ┌────────────────────────┐        │
        │  Today] │  │HTTP_Check_Mesocycle    │ ◄─NEW  │
        │    │    │  │Status (Go Backend)     │        │
        │    ▼    │  └──────────┬─────────────┘        │
        │[Has     │             │                      │
        │Routine? │   ┌─────────▼─────────┐            │
        │ Today?] │   │If_Mesocycle_      │ ◄─── NEW   │
        │    │    │   │Complete           │            │
        │    │    │   └───┬───────────┬───┘            │
        │    │    │       │TRUE       │FALSE           │
        │    │    │       ▼           ▼                │
        │    │    │ [Execute      [Scheduling         │
        │    │    │  Renewal]      Agent]              │
        │    ▼    │                                    │
        │[Intention_Agent]                             │
        │    │                                         │
        │    ▼                                         │
        │[Switch_Intention]                            │
        │  │ VER_RUTINA_DE_HOY ──► [AI Agent]         │
        │  │ CHAT ────────────────► [AI Agent]        │
        │  │ CONFIRMAR_RUTINA ───► [CONFIRMATION]     │
        │  │ RENOVAR_MESOCICLO ──► [Execute Renewal]  │◄─NEW
        │                                              │
        └──────────────────────────────────────────────┘
                               │
          ┌────────────────────┴────────────────────┐
          │                                         │
          ▼                                         ▼
┌───────────────────────────┐      ┌───────────────────────────┐
│ GymBotMesocycleRenewal    │      │ Scheduling/Routine Flow   │
│ (Sub-workflow)            │      │                           │
│                           │      │                           │
│ [Renewal_Agent]           │      │                           │
│       │                   │      │                           │
│ [Parse_Intention]         │      │                           │
│       │                   │      │                           │
│ [Switch_Intention]        │      │                           │
│  │ MANTENER ──► HTTP POST │      │                           │
│  │ CAMBIAR ───► HTTP POST │      │                           │
│  │ ROTAR ─────► HTTP POST │      │                           │
│  │ MODIFICAR ─► Agent+HTTP│      │                           │
│  │ PREGUNTAR ─► WhatsApp  │      │                           │
│       │                   │      │                           │
│ [WhatsApp Notify]         │      │                           │
│       │                   │      │                           │
│ [Cleanup_Memory]          │      │                           │
└───────────────────────────┘      └───────────────────────────┘
```

---

## Main Flow Modifications (GymRatFlow)

### 1. New Node: HTTP_Check_Mesocycle_Status

**Purpose:** Check if user has completed their current mesocycle (all week 4 workouts done).

**Position:** Insert after `has_planned_workouts1` FALSE branch, before scheduling flow.

**Connection Changes:**
- OLD: `has_planned_workouts1` FALSE --> `Week_Schedule`, `User_Finished_Workouts`, `Template_Days`
- NEW: `has_planned_workouts1` FALSE --> `HTTP_Check_Mesocycle_Status`

#### Node Configuration

```json
{
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
  "position": [-4400, 25600],
  "id": "http-check-mesocycle-status",
  "name": "HTTP_Check_Mesocycle_Status",
  "alwaysOutputData": true,
  "credentials": {
    "httpHeaderAuth": {
      "id": "INTERNAL_API_KEY_CREDENTIAL_ID",
      "name": "Workout API Internal"
    }
  }
}
```

**Expected Response:**
```json
{
  "user_id": "uuid",
  "mesocycle_number": 1,
  "days_per_week": 3,
  "is_complete": true,
  "completed_count": 3,
  "required_count": 3,
  "week_schedule": "fb_3"
}
```

---

### 2. New Node: If_Mesocycle_Complete

**Purpose:** Route user to renewal flow if mesocycle is complete, otherwise to scheduling.

**Position:** After `HTTP_Check_Mesocycle_Status`

#### Node Configuration

```json
{
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
  "position": [-4200, 25600],
  "id": "if-mesocycle-complete",
  "name": "If_Mesocycle_Complete",
  "alwaysOutputData": true
}
```

**Connection Logic:**
- TRUE branch --> `Execute_Mesocycle_Renewal`
- FALSE branch --> `Week_Schedule`, `User_Finished_Workouts`, `Template_Days` (existing scheduling flow)

---

### 3. New Node: Execute_Mesocycle_Renewal

**Purpose:** Call the renewal sub-workflow with user context.

**Position:** Connected from `If_Mesocycle_Complete` TRUE branch

#### Node Configuration

```json
{
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
        "current_mesocycle": "={{ $json.mesocycle_number }}",
        "current_days_per_week": "={{ $json.days_per_week }}",
        "user_name": "={{ $items('GetUser')[0].json.full_name }}",
        "user_message": "={{ $items('If')[0].json.messages[0].text.body }}"
      },
      "matchingColumns": [],
      "schema": [
        {
          "id": "user_id",
          "displayName": "user_id",
          "required": true,
          "type": "string"
        },
        {
          "id": "whatsapp_id",
          "displayName": "whatsapp_id",
          "required": true,
          "type": "string"
        },
        {
          "id": "phone_number_id",
          "displayName": "phone_number_id",
          "required": true,
          "type": "string"
        },
        {
          "id": "current_mesocycle",
          "displayName": "current_mesocycle",
          "required": true,
          "type": "number"
        },
        {
          "id": "current_days_per_week",
          "displayName": "current_days_per_week",
          "required": true,
          "type": "number"
        },
        {
          "id": "user_name",
          "displayName": "user_name",
          "required": true,
          "type": "string"
        },
        {
          "id": "user_message",
          "displayName": "user_message",
          "required": true,
          "type": "string"
        }
      ]
    },
    "options": {
      "waitForSubWorkflow": true
    }
  },
  "type": "n8n-nodes-base.executeWorkflow",
  "typeVersion": 1.3,
  "position": [-4000, 25500],
  "id": "execute-mesocycle-renewal",
  "name": "Execute_Mesocycle_Renewal",
  "alwaysOutputData": true
}
```

---

### 4. Update: Intention_Agent System Prompt

Add the `RENOVAR_MESOCICLO` intent to the existing Intention_Agent.

**Current Location:** Node ID `f409bee4-c5fe-4580-9890-d9c4b24b331a`

**Updated System Prompt:**

```
Eres un agente encargado de evaluar la intencion del usuario segun el mensaje que recibes.

INTENCIONES VALIDAS:
- VER_RUTINA_DE_HOY
- RENOVAR_MESOCICLO
- CHAT

VER_RUTINA_DE_HOY: El usuario quiere ver su rutina/entrenamiento del dia.
Ejemplos: "Muestrame mi rutina", "Que me toca hoy", "Mi entrenamiento", "Dame mi workout"

RENOVAR_MESOCICLO: El usuario quiere cambiar, renovar o modificar su rutina de entrenamiento.
Ejemplos: "Quiero cambiar mi rutina", "Nuevos ejercicios", "Renovar mesociclo", "Cambiar dias", "Quiero entrenar mas dias", "Quiero variar los ejercicios", "Cambiar mi plan", "Ya me aburri de los mismos ejercicios"

CHAT: Cualquier otra pregunta, comentario o conversacion general sobre fitness.
Ejemplos: "Que ejercicio es mejor para biceps", "Hola", "Gracias"

NOTA: Las confirmaciones de rutina completada se manejan por otro flujo (pending_tasks). Si el usuario dice que termino su rutina, responde CHAT.

Retorna SOLO la intencion (VER_RUTINA_DE_HOY, RENOVAR_MESOCICLO o CHAT), sin explicacion adicional.
```

---

### 5. Update: Switch Node (Switch_Intention)

Add `RENOVAR_MESOCICLO` branch to the existing Switch node.

**Current Location:** Node ID `8fc0ecb3-0f14-48ad-873a-11577cece84d`

**Add New Rule:**

```json
{
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
}
```

**Updated Switch Configuration:**

```json
{
  "parameters": {
    "rules": {
      "values": [
        {
          "conditions": {
            "options": { "caseSensitive": true, "typeValidation": "strict", "version": 3 },
            "conditions": [
              {
                "leftValue": "={{ $json.output.trim()}}",
                "rightValue": "CONFIRMAR_RUTINA",
                "operator": { "type": "string", "operation": "equals" },
                "id": "77fe3bdd-77b3-46ed-b707-a1f200c4dbd2"
              }
            ],
            "combinator": "and"
          },
          "renameOutput": true,
          "outputKey": "ROUTINE_CONFIRMATION"
        },
        {
          "conditions": {
            "options": { "caseSensitive": true, "typeValidation": "strict", "version": 3 },
            "conditions": [
              {
                "id": "cef2c8d2-71fd-4010-8777-9399c05906b3",
                "leftValue": "={{ $json.output.trim() }}",
                "rightValue": "CHAT",
                "operator": { "type": "string", "operation": "equals" }
              }
            ],
            "combinator": "and"
          },
          "renameOutput": true,
          "outputKey": "CHAT"
        },
        {
          "conditions": {
            "options": { "caseSensitive": true, "typeValidation": "strict", "version": 3 },
            "conditions": [
              {
                "id": "b44ecaad-ce21-47e4-9326-0ac5ed78c5a2",
                "leftValue": "={{ $json.output.trim() }}",
                "rightValue": "VER_RUTINA_DE_HOY",
                "operator": { "type": "string", "operation": "equals" }
              }
            ],
            "combinator": "and"
          },
          "renameOutput": true,
          "outputKey": "VER_RUTINA_DE_HOY"
        },
        {
          "conditions": {
            "options": { "caseSensitive": true, "typeValidation": "strict", "version": 3 },
            "conditions": [
              {
                "id": "renovar-mesociclo-condition",
                "leftValue": "={{ $json.output.trim() }}",
                "rightValue": "RENOVAR_MESOCICLO",
                "operator": { "type": "string", "operation": "equals" }
              }
            ],
            "combinator": "and"
          },
          "renameOutput": true,
          "outputKey": "RENOVAR_MESOCICLO"
        }
      ]
    },
    "options": {}
  },
  "type": "n8n-nodes-base.switch",
  "typeVersion": 3.4,
  "position": [-2848, 24384],
  "id": "8fc0ecb3-0f14-48ad-873a-11577cece84d",
  "name": "Switch"
}
```

---

### Connection Diagram: Main Flow Changes

```
BEFORE:
========
[has_planned_workouts1]
        │
 TRUE   │    FALSE
   │    │      │
   ▼    │      ├──► [Week_Schedule]
[Filter │      ├──► [User_Finished_Workouts]
 Today] │      └──► [Template_Days]
   │    │              │
   ▼    │              ▼
[user   │          [Merge] ──► [AI Agent1 - Scheduling]
HasRout │
ineFor  │
Today]  │
   │    │
   ▼    ▼
[Intention_Agent]
   │
   ▼
[Switch]
  │ ROUTINE_CONFIRMATION ──► [CONFIRMATION AGENT]
  │ CHAT ──────────────────► [AI Agent]
  │ VER_RUTINA_DE_HOY ─────► [AI Agent]


AFTER:
=======
[has_planned_workouts1]
        │
 TRUE   │    FALSE
   │    │      │
   ▼    │      ▼
[Filter │  [HTTP_Check_Mesocycle_Status] ◄── NEW
 Today] │      │
   │    │      ▼
   ▼    │  [If_Mesocycle_Complete] ◄─────── NEW
[user   │      │
HasRout │  TRUE│    FALSE
ineFor  │   │  │      │
Today]  │   │  │      ├──► [Week_Schedule]
   │    │   │  │      ├──► [User_Finished_Workouts]
   │    │   │  │      └──► [Template_Days]
   │    │   │  │              │
   │    │   │  │              ▼
   │    │   │  │          [Merge] ──► [AI Agent1 - Scheduling]
   │    │   │  │
   │    │   │  │
   │    │   ▼  │
   │    │[Execute_Mesocycle_Renewal] ◄───── NEW
   │    │   │
   ▼    ▼   │
[Intention_Agent]
   │
   ▼
[Switch]
  │ ROUTINE_CONFIRMATION ──► [CONFIRMATION AGENT]
  │ CHAT ──────────────────► [AI Agent]
  │ VER_RUTINA_DE_HOY ─────► [AI Agent]
  │ RENOVAR_MESOCICLO ─────► [Execute_Mesocycle_Renewal] ◄── NEW
```

---

## Renewal Subflow Updates (GymBotMesocycleRenewal)

The existing `GymBotMesocycleRenewal.json` needs significant updates to:
1. Replace SQL nodes with HTTP Request nodes calling the Go backend
2. Add the new `MODIFICAR_PERFIL` path
3. Fix the `ua_4` bug (should be `ul_4`)

### Updated Flow Structure

```
[Mesocycle_Renewal_Trigger]
        │
        ▼
[Renewal_Agent] ◄── [OpenAI_Renewal] + [Renewal_Memory]
        │
        ▼
[Parse_Intention]
        │
        ▼
[Switch_Intention]
        │
        ├──► MANTENER_RUTINA ──────────────────────────────────┐
        │         │                                             │
        │         ▼                                             │
        │    [HTTP_Renew_Maintain] ◄─── NEW                    │
        │         │                                             │
        │         ▼                                             │
        │    [Notify_Mantener_Success]                         │
        │         │                                             │
        │         ▼                                             │
        │    [Cleanup_Memory_Mantener]                         │
        │                                                       │
        ├──► CAMBIAR_DIAS ────────────────────────────────────┐│
        │         │                                            ││
        │         ▼                                            ││
        │    [HTTP_Renew_Change_Days] ◄─── NEW                ││
        │         │                                            ││
        │         ▼                                            ││
        │    [Call_GymRatForm_ChangeDays]                     ││
        │         │                                            ││
        │         ▼                                            ││
        │    [Notify_Days_Change_Success]                     ││
        │         │                                            ││
        │         ▼                                            ││
        │    [Cleanup_Memory_Days]                            ││
        │                                                      ││
        ├──► ROTAR_EJERCICIOS ───────────────────────────────┐││
        │         │                                           │││
        │         ▼                                           │││
        │    [HTTP_Renew_Rotate_Exercises] ◄─── NEW          │││
        │         │                                           │││
        │         ▼                                           │││
        │    [Notify_Rotation_Success]                       │││
        │         │                                           │││
        │         ▼                                           │││
        │    [Cleanup_Memory_Rotation]                       │││
        │                                                     │││
        ├──► MODIFICAR_PERFIL ◄─── NEW ─────────────────────┐││││
        │         │                                          ││││
        │         ▼                                          ││││
        │    [Profile_Modification_Agent] ◄─── NEW          ││││
        │         │                                          ││││
        │         ▼                                          ││││
        │    [HTTP_Update_Profile] ◄─── NEW                 ││││
        │         │                                          ││││
        │         ▼                                          ││││
        │    [Call_GymRatForm_ModifyProfile]                ││││
        │         │                                          ││││
        │         ▼                                          ││││
        │    [Notify_Profile_Change_Success]                ││││
        │         │                                          ││││
        │         ▼                                          ││││
        │    [Cleanup_Memory_Profile]                       ││││
        │                                                    ││││
        └──► PREGUNTAR_OPCIONES ──────────────────────────┐│││││
                  │                                        ││││││
                  ▼                                        ││││││
             [Send_Options_Message]                       ││││││
                                                          ││││││
                  ┌───────────────────────────────────────┘│││││
                  │   ┌───────────────────────────────────┘││││
                  │   │   ┌───────────────────────────────┘│││
                  │   │   │   ┌───────────────────────────┘││
                  │   │   │   │   ┌───────────────────────┘│
                  │   │   │   │   │   ┌───────────────────┘
                  ▼   ▼   ▼   ▼   ▼   ▼
              [Output: Return to GymRatFlow]
```

---

### Path 1: MANTENER_RUTINA

**Replace:** `Reset_For_Mantener` (SQL node)
**With:** `HTTP_Renew_Maintain` (HTTP Request node)

#### HTTP_Renew_Maintain Configuration

```json
{
  "parameters": {
    "method": "POST",
    "url": "={{ $env.WORKOUT_API_URL }}/api/v1/plans/{{ $json.user_id }}/renew/maintain",
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
    "sendBody": true,
    "bodyParameters": {
      "parameters": []
    },
    "options": {
      "timeout": 30000
    }
  },
  "type": "n8n-nodes-base.httpRequest",
  "typeVersion": 4.2,
  "position": [1120, 0],
  "id": "http-renew-maintain",
  "name": "HTTP_Renew_Maintain",
  "credentials": {
    "httpHeaderAuth": {
      "id": "INTERNAL_API_KEY_CREDENTIAL_ID",
      "name": "Workout API Internal"
    }
  }
}
```

**Expected Response:**
```json
{
  "success": true,
  "user_id": "uuid",
  "new_mesocycle_number": 2,
  "message": "Mesocycle renewed successfully"
}
```

#### Updated Notify_Mantener_Success

```json
{
  "parameters": {
    "operation": "send",
    "phoneNumberId": "={{ $('Mesocycle_Renewal_Trigger').first().json.phone_number_id }}",
    "recipientPhoneNumber": "={{ $('Mesocycle_Renewal_Trigger').first().json.whatsapp_id }}",
    "textBody": "=\uD83C\uDF89 !Excelente eleccion, {{ $('Mesocycle_Renewal_Trigger').first().json.user_name }}!\n\nTu rutina se ha renovado para el **Mesociclo {{ $json.new_mesocycle_number }}**.\n\nMantendras los mismos ejercicios con la progresion de carga optimizada.\n\n\uD83D\uDCC5 Escribeme cuando quieras agendar tu primera semana del nuevo ciclo.\n\n!Vamos con toda! \uD83D\uDCAA\uD83D\uDD25",
    "additionalFields": {}
  },
  "type": "n8n-nodes-base.whatsApp",
  "typeVersion": 1.1,
  "position": [1360, 0],
  "id": "notify-mantener",
  "name": "Notify_Mantener_Success",
  "credentials": {
    "whatsAppApi": {
      "id": "xIjy4zDHyjIvGQT4",
      "name": "WhatsApp account"
    }
  }
}
```

---

### Path 2: CAMBIAR_DIAS

**Replace:** `Prepare_Days_Change` (SQL node)
**With:** `HTTP_Renew_Change_Days` (HTTP Request node)

#### HTTP_Renew_Change_Days Configuration

```json
{
  "parameters": {
    "method": "POST",
    "url": "={{ $env.WORKOUT_API_URL }}/api/v1/plans/{{ $json.user_id }}/renew/change-days",
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
    "sendBody": true,
    "specifyBody": "json",
    "jsonBody": "={{ JSON.stringify({ \"new_days\": $json.newDays }) }}",
    "options": {
      "timeout": 30000
    }
  },
  "type": "n8n-nodes-base.httpRequest",
  "typeVersion": 4.2,
  "position": [1120, 160],
  "id": "http-renew-change-days",
  "name": "HTTP_Renew_Change_Days",
  "credentials": {
    "httpHeaderAuth": {
      "id": "INTERNAL_API_KEY_CREDENTIAL_ID",
      "name": "Workout API Internal"
    }
  }
}
```

**Request Body:**
```json
{
  "new_days": 4
}
```

**Expected Response:**
```json
{
  "success": true,
  "user_id": "uuid",
  "new_mesocycle_number": 2,
  "old_days_per_week": 3,
  "new_days_per_week": 4,
  "new_week_schedule": "ul_4",
  "requires_regeneration": true,
  "message": "Days changed, workouts cleared for regeneration"
}
```

#### Call_GymRatForm_ChangeDays Configuration

```json
{
  "parameters": {
    "workflowId": {
      "__rl": true,
      "value": "523zkE5vZ7aVXJk5k4GAG",
      "mode": "list",
      "cachedResultUrl": "/workflow/523zkE5vZ7aVXJk5k4GAG",
      "cachedResultName": "GymRatForm Supabase v2.1"
    },
    "workflowInputs": {
      "mappingMode": "defineBelow",
      "value": {
        "whatsapp_id": "={{ $('Mesocycle_Renewal_Trigger').first().json.whatsapp_id }}",
        "is_renewal": "true",
        "override_days_available": "={{ $('Switch_Intention').first().json.newDays }}"
      },
      "matchingColumns": [],
      "schema": [
        {
          "id": "whatsapp_id",
          "displayName": "whatsapp_id",
          "required": true,
          "type": "string"
        },
        {
          "id": "is_renewal",
          "displayName": "is_renewal",
          "required": false,
          "type": "string"
        },
        {
          "id": "override_days_available",
          "displayName": "override_days_available",
          "required": false,
          "type": "number"
        }
      ]
    },
    "options": {
      "waitForSubWorkflow": true
    }
  },
  "type": "n8n-nodes-base.executeWorkflow",
  "typeVersion": 1.3,
  "position": [1360, 160],
  "id": "call-gymratform-change-days",
  "name": "Call_GymRatForm_ChangeDays"
}
```

#### Notify_Days_Change_Success Configuration

```json
{
  "parameters": {
    "operation": "send",
    "phoneNumberId": "={{ $('Mesocycle_Renewal_Trigger').first().json.phone_number_id }}",
    "recipientPhoneNumber": "={{ $('Mesocycle_Renewal_Trigger').first().json.whatsapp_id }}",
    "textBody": "=\uD83C\uDF89 !Tu nueva rutina esta lista, {{ $('Mesocycle_Renewal_Trigger').first().json.user_name }}!\n\nHe creado un plan completamente nuevo con {{ $('Switch_Intention').first().json.newDays }} dias por semana.\n\n\uD83D\uDCC5 Escribeme cuando quieras agendar tu primera semana del nuevo mesociclo.\n\n!A romperla! \uD83D\uDCAA\uD83D\uDD25",
    "additionalFields": {}
  },
  "type": "n8n-nodes-base.whatsApp",
  "typeVersion": 1.1,
  "position": [1600, 160],
  "id": "notify-days-change",
  "name": "Notify_Days_Change_Success",
  "credentials": {
    "whatsAppApi": {
      "id": "xIjy4zDHyjIvGQT4",
      "name": "WhatsApp account"
    }
  }
}
```

---

### Path 3: ROTAR_EJERCICIOS

**Replace:** `Find_Alternative_Exercises` + `Process_Rotation` + `Apply_Rotation_Updates` (SQL nodes)
**With:** `HTTP_Renew_Rotate_Exercises` (single HTTP Request node)

#### HTTP_Renew_Rotate_Exercises Configuration

```json
{
  "parameters": {
    "method": "POST",
    "url": "={{ $env.WORKOUT_API_URL }}/api/v1/plans/{{ $json.user_id }}/renew/rotate-exercises",
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
    "sendBody": true,
    "bodyParameters": {
      "parameters": []
    },
    "options": {
      "timeout": 60000
    }
  },
  "type": "n8n-nodes-base.httpRequest",
  "typeVersion": 4.2,
  "position": [1120, 320],
  "id": "http-renew-rotate-exercises",
  "name": "HTTP_Renew_Rotate_Exercises",
  "credentials": {
    "httpHeaderAuth": {
      "id": "INTERNAL_API_KEY_CREDENTIAL_ID",
      "name": "Workout API Internal"
    }
  }
}
```

**Expected Response:**
```json
{
  "success": true,
  "user_id": "uuid",
  "new_mesocycle_number": 2,
  "exercises_rotated": 8,
  "rotation_details": [
    {
      "day_name": "Dia 1 - Pecho y Triceps",
      "old_exercise": "Press de banca con barra",
      "new_exercise": "Press de banca con mancuernas",
      "pattern": "push_h"
    }
  ],
  "message": "Exercises rotated successfully"
}
```

#### Updated Notify_Rotation_Success

```json
{
  "parameters": {
    "operation": "send",
    "phoneNumberId": "={{ $('Mesocycle_Renewal_Trigger').first().json.phone_number_id }}",
    "recipientPhoneNumber": "={{ $('Mesocycle_Renewal_Trigger').first().json.whatsapp_id }}",
    "textBody": "=\uD83D\uDD04 !Ejercicios rotados con exito, {{ $('Mesocycle_Renewal_Trigger').first().json.user_name }}!\n\nHe seleccionado {{ $json.exercises_rotated }} ejercicios nuevos para darte estimulos frescos.\n\nTu frecuencia de {{ $('Mesocycle_Renewal_Trigger').first().json.current_days_per_week }} dias por semana se mantiene.\n\n\uD83D\uDCC5 Escribeme cuando quieras agendar tu primera semana.\n\n!A por el nuevo ciclo! \uD83D\uDCAA\uD83D\uDE80",
    "additionalFields": {}
  },
  "type": "n8n-nodes-base.whatsApp",
  "typeVersion": 1.1,
  "position": [1360, 320],
  "id": "notify-rotation",
  "name": "Notify_Rotation_Success",
  "credentials": {
    "whatsAppApi": {
      "id": "xIjy4zDHyjIvGQT4",
      "name": "WhatsApp account"
    }
  }
}
```

---

### Path 4: MODIFICAR_PERFIL (NEW)

This is a new path that collects updated preferences from the user, then regenerates their routine.

#### Profile_Modification_Agent Configuration

```json
{
  "parameters": {
    "promptType": "define",
    "text": "={{ $json.user_message }}",
    "options": {
      "systemMessage": "=Eres \"FitBot\", el asistente de modificacion de perfil. Tu mision es recolectar los cambios que el usuario quiere hacer en su rutina.\n\n## CONTEXTO DEL USUARIO\n- user_id: {{ $json.user_id }}\n- Nombre: {{ $json.user_name }}\n- Mesociclo actual: {{ $json.current_mesocycle }}\n- Dias por semana: {{ $json.current_days_per_week }}\n\n## INFORMACION A RECOLECTAR\nPregunta al usuario que quiere cambiar. Puede modificar uno o varios de estos campos:\n\n1. **Musculos prioritarios**: Que partes del cuerpo quiere enfocarse\n   - Ejemplos: gluteos, piernas, espalda, pecho, brazos\n\n2. **Estado de salud/lesiones**: Si tiene alguna nueva lesion o molestia\n   - A: Sin limitaciones\n   - B: Cuidado tren inferior (rodillas, tobillos, cadera)\n   - C: Cuidado tren superior (hombros, codos, munecas)\n   - D: Cuidado espalda (lumbares, cervicales)\n   - E: Condicion medica especial\n\n3. **Duracion de sesion**: Cuanto tiempo tiene disponible\n   - 30-45 minutos\n   - 45-60 minutos\n   - 60-75 minutos\n   - Mas de 75 minutos\n\n4. **Ejercicios a evitar**: Musculos o ejercicios que no le gustan\n\n## FLUJO\n1. Pregunta que aspectos quiere modificar\n2. Recolecta la informacion necesaria\n3. Confirma los cambios antes de aplicarlos\n4. Cuando el usuario confirme, retorna los cambios\n\n## FORMATO DE SALIDA\nCuando tengas todos los cambios confirmados, tu ultima linea DEBE ser:\nCAMBIOS:priority_muscles=[valor]|health_status=[A-E]|session_duration=[valor]|disliked_exercises=[valor]\n\nSolo incluye los campos que el usuario quiere cambiar. Ejemplo:\nCAMBIOS:priority_muscles=gluteos,piernas|session_duration=45-60 minutos\n\n## REGLAS\n1. Se amigable y motivador\n2. Todo en espanol\n3. Si el usuario no quiere cambiar algo, no lo incluyas\n4. Confirma siempre antes de aplicar cambios"
    }
  },
  "type": "@n8n/n8n-nodes-langchain.agent",
  "typeVersion": 3.1,
  "position": [1120, 480],
  "id": "profile-modification-agent",
  "name": "Profile_Modification_Agent"
}
```

#### Parse_Profile_Changes Configuration

```json
{
  "parameters": {
    "jsCode": "// Extraer los cambios del output del agente\nconst output = $input.first().json.output || '';\nconst inputData = $('Switch_Intention').first().json;\n\n// Buscar el patron de cambios\nconst changesMatch = output.match(/CAMBIOS:(.+)/);\n\nlet changes = {};\nlet cleanOutput = output;\n\nif (changesMatch) {\n  const changesStr = changesMatch[1];\n  // Parsear los cambios: priority_muscles=valor|health_status=valor\n  const pairs = changesStr.split('|');\n  \n  for (const pair of pairs) {\n    const [key, value] = pair.split('=');\n    if (key && value) {\n      changes[key.trim()] = value.trim();\n    }\n  }\n  \n  cleanOutput = output.replace(/CAMBIOS:.+/g, '').trim();\n}\n\nreturn [{\n  json: {\n    ...inputData,\n    profile_changes: changes,\n    has_changes: Object.keys(changes).length > 0,\n    agentOutput: cleanOutput,\n    rawOutput: output\n  }\n}];"
  },
  "type": "n8n-nodes-base.code",
  "typeVersion": 2,
  "position": [1360, 480],
  "id": "parse-profile-changes",
  "name": "Parse_Profile_Changes"
}
```

#### HTTP_Update_Profile Configuration

```json
{
  "parameters": {
    "method": "POST",
    "url": "={{ $env.WORKOUT_API_URL }}/api/v1/plans/{{ $json.user_id }}/renew/update-profile",
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
    "sendBody": true,
    "specifyBody": "json",
    "jsonBody": "={{ JSON.stringify($json.profile_changes) }}",
    "options": {
      "timeout": 30000
    }
  },
  "type": "n8n-nodes-base.httpRequest",
  "typeVersion": 4.2,
  "position": [1600, 480],
  "id": "http-update-profile",
  "name": "HTTP_Update_Profile",
  "credentials": {
    "httpHeaderAuth": {
      "id": "INTERNAL_API_KEY_CREDENTIAL_ID",
      "name": "Workout API Internal"
    }
  }
}
```

**Request Body Example:**
```json
{
  "priority_muscles": "gluteos,piernas",
  "health_status": "B",
  "session_duration": "45-60 minutos",
  "disliked_exercises": "pantorrillas"
}
```

**Expected Response:**
```json
{
  "success": true,
  "user_id": "uuid",
  "new_mesocycle_number": 2,
  "updated_fields": ["priority_muscles", "session_duration"],
  "requires_regeneration": true,
  "message": "Profile updated, workouts cleared for regeneration"
}
```

#### Call_GymRatForm_ModifyProfile Configuration

```json
{
  "parameters": {
    "workflowId": {
      "__rl": true,
      "value": "523zkE5vZ7aVXJk5k4GAG",
      "mode": "list",
      "cachedResultUrl": "/workflow/523zkE5vZ7aVXJk5k4GAG",
      "cachedResultName": "GymRatForm Supabase v2.1"
    },
    "workflowInputs": {
      "mappingMode": "defineBelow",
      "value": {
        "whatsapp_id": "={{ $('Mesocycle_Renewal_Trigger').first().json.whatsapp_id }}",
        "is_renewal": "true"
      },
      "matchingColumns": [],
      "schema": [
        {
          "id": "whatsapp_id",
          "displayName": "whatsapp_id",
          "required": true,
          "type": "string"
        },
        {
          "id": "is_renewal",
          "displayName": "is_renewal",
          "required": false,
          "type": "string"
        }
      ]
    },
    "options": {
      "waitForSubWorkflow": true
    }
  },
  "type": "n8n-nodes-base.executeWorkflow",
  "typeVersion": 1.3,
  "position": [1840, 480],
  "id": "call-gymratform-modify-profile",
  "name": "Call_GymRatForm_ModifyProfile"
}
```

#### Notify_Profile_Change_Success Configuration

```json
{
  "parameters": {
    "operation": "send",
    "phoneNumberId": "={{ $('Mesocycle_Renewal_Trigger').first().json.phone_number_id }}",
    "recipientPhoneNumber": "={{ $('Mesocycle_Renewal_Trigger').first().json.whatsapp_id }}",
    "textBody": "=\u2699\uFE0F !Perfil actualizado, {{ $('Mesocycle_Renewal_Trigger').first().json.user_name }}!\n\nHe regenerado tu rutina con tus nuevas preferencias.\n\nTu plan ahora esta optimizado segun tus cambios.\n\n\uD83D\uDCC5 Escribeme cuando quieras agendar tu primera semana del nuevo mesociclo.\n\n!A por nuevos resultados! \uD83D\uDCAA\uD83D\uDE80",
    "additionalFields": {}
  },
  "type": "n8n-nodes-base.whatsApp",
  "typeVersion": 1.1,
  "position": [2080, 480],
  "id": "notify-profile-change",
  "name": "Notify_Profile_Change_Success",
  "credentials": {
    "whatsAppApi": {
      "id": "xIjy4zDHyjIvGQT4",
      "name": "WhatsApp account"
    }
  }
}
```

---

### Updated Switch_Intention for Renewal Subflow

Add the new `MODIFICAR_PERFIL` branch to the existing switch.

```json
{
  "parameters": {
    "rules": {
      "values": [
        {
          "conditions": {
            "options": { "caseSensitive": true, "typeValidation": "strict", "version": 3 },
            "conditions": [
              {
                "leftValue": "={{ $json.intention }}",
                "rightValue": "MANTENER_RUTINA",
                "operator": { "type": "string", "operation": "equals" },
                "id": "mantener-condition"
              }
            ],
            "combinator": "and"
          },
          "renameOutput": true,
          "outputKey": "MANTENER_RUTINA"
        },
        {
          "conditions": {
            "options": { "caseSensitive": true, "typeValidation": "strict", "version": 3 },
            "conditions": [
              {
                "leftValue": "={{ $json.intention }}",
                "rightValue": "CAMBIAR_DIAS",
                "operator": { "type": "string", "operation": "equals" },
                "id": "cambiar-dias-condition"
              }
            ],
            "combinator": "and"
          },
          "renameOutput": true,
          "outputKey": "CAMBIAR_DIAS"
        },
        {
          "conditions": {
            "options": { "caseSensitive": true, "typeValidation": "strict", "version": 3 },
            "conditions": [
              {
                "leftValue": "={{ $json.intention }}",
                "rightValue": "ROTAR_EJERCICIOS",
                "operator": { "type": "string", "operation": "equals" },
                "id": "rotar-condition"
              }
            ],
            "combinator": "and"
          },
          "renameOutput": true,
          "outputKey": "ROTAR_EJERCICIOS"
        },
        {
          "conditions": {
            "options": { "caseSensitive": true, "typeValidation": "strict", "version": 3 },
            "conditions": [
              {
                "leftValue": "={{ $json.intention }}",
                "rightValue": "MODIFICAR_PERFIL",
                "operator": { "type": "string", "operation": "equals" },
                "id": "modificar-perfil-condition"
              }
            ],
            "combinator": "and"
          },
          "renameOutput": true,
          "outputKey": "MODIFICAR_PERFIL"
        },
        {
          "conditions": {
            "options": { "caseSensitive": true, "typeValidation": "strict", "version": 3 },
            "conditions": [
              {
                "leftValue": "={{ $json.intention }}",
                "rightValue": "PREGUNTAR_OPCIONES",
                "operator": { "type": "string", "operation": "equals" },
                "id": "preguntar-condition"
              }
            ],
            "combinator": "and"
          },
          "renameOutput": true,
          "outputKey": "PREGUNTAR_OPCIONES"
        }
      ]
    },
    "options": {}
  },
  "type": "n8n-nodes-base.switch",
  "typeVersion": 3.4,
  "position": [800, 240],
  "id": "switch-intention",
  "name": "Switch_Intention"
}
```

---

## GymRatForm Integration

### Required Modifications to GymRatForm

The `GymRatForm Supabase v2.1` workflow needs minor modifications to support renewal scenarios.

#### 1. Update Input Trigger

Add new optional parameters to the workflow trigger.

```json
{
  "parameters": {
    "workflowInputs": {
      "values": [
        {
          "name": "whatsapp_id",
          "type": "number"
        },
        {
          "name": "is_renewal",
          "type": "string"
        },
        {
          "name": "override_days_available",
          "type": "number"
        }
      ]
    }
  },
  "type": "n8n-nodes-base.executeWorkflowTrigger",
  "typeVersion": 1.1,
  "position": [-2000, 144],
  "id": "91295f4c-8308-4a5b-851c-f668f8dce1b5",
  "name": "input",
  "executeOnce": true
}
```

#### 2. ProcessUserPreferences Modification

Add logic to use `override_days_available` when provided.

```javascript
// Add at the beginning of ProcessUserPreferences Code node
const profile = $('GetUserProfile').item.json;
const inputData = $('input').first().json;

// Override days_available if provided (for renewal scenarios)
const daysAvailable = inputData.override_days_available
  ? inputData.override_days_available
  : profile.days_available;

// Rest of the existing code...
// Replace profile.days_available with daysAvailable where used
```

#### 3. Skip User Creation for Renewals

Add conditional logic to skip user/plan creation during renewals.

```javascript
// In UserExists node or add new If node
const isRenewal = $('input').first().json.is_renewal === 'true';

// If is_renewal, user and plan already exist - skip creation
// Route directly to Loop Over Items
```

#### 4. Suppress WhatsApp Notification for Renewals

Modify the `NotifyRoutineCreated` node to be conditional.

```json
{
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
          "id": "not-renewal-check",
          "leftValue": "={{ $('input').first().json.is_renewal }}",
          "rightValue": "true",
          "operator": {
            "type": "string",
            "operation": "notEquals"
          }
        }
      ],
      "combinator": "and"
    },
    "options": {}
  },
  "type": "n8n-nodes-base.if",
  "typeVersion": 2.3,
  "position": [1500, 144],
  "id": "if-should-notify",
  "name": "If_Should_Notify"
}
```

---

## Memory Management

### Session Key Format

All renewal-related conversations use the following session key format:

```
{user_id}_mesocycle_renewal
```

Example: `5d3b501c-8ac5-4dfd-9c0d-4ef9a8a15f9c_mesocycle_renewal`

### Memory Cleanup

After any successful renewal operation, memory is cleaned up to prevent stale context.

#### Cleanup_Memory Node Configuration

```json
{
  "parameters": {
    "operation": "executeQuery",
    "query": "DELETE FROM n8n_chat_histories \nWHERE session_id LIKE '{{ $('Mesocycle_Renewal_Trigger').first().json.user_id }}_mesocycle_renewal%';",
    "options": {}
  },
  "type": "n8n-nodes-base.postgres",
  "typeVersion": 2.6,
  "position": [2080, 0],
  "id": "cleanup-memory",
  "name": "Cleanup_Memory",
  "credentials": {
    "postgres": {
      "id": "vZLJtIWG5nYXMez4",
      "name": "Supabase Memory"
    }
  }
}
```

### Memory for MODIFICAR_PERFIL Sub-conversation

The Profile Modification Agent uses a separate memory key to track the profile change conversation:

```
{user_id}_profile_modification
```

This allows multi-turn conversations within the profile modification flow.

---

## Error Handling

### HTTP Request Error Handling

All HTTP Request nodes should include error handling for API failures.

#### Add Error Workflow Configuration

```json
{
  "onError": "continueErrorOutput",
  "retryOnFail": true,
  "maxTries": 2,
  "waitBetweenTries": 1000
}
```

#### Error Handler Node

```json
{
  "parameters": {
    "operation": "send",
    "phoneNumberId": "={{ $('Mesocycle_Renewal_Trigger').first().json.phone_number_id }}",
    "recipientPhoneNumber": "={{ $('Mesocycle_Renewal_Trigger').first().json.whatsapp_id }}",
    "textBody": "Lo siento, {{ $('Mesocycle_Renewal_Trigger').first().json.user_name }}, hubo un problema procesando tu solicitud. Por favor intenta de nuevo en unos minutos. Si el problema persiste, escribe 'ayuda' para contactar soporte.",
    "additionalFields": {}
  },
  "type": "n8n-nodes-base.whatsApp",
  "typeVersion": 1.1,
  "position": [1600, 600],
  "id": "error-handler",
  "name": "Error_Handler",
  "credentials": {
    "whatsAppApi": {
      "id": "xIjy4zDHyjIvGQT4",
      "name": "WhatsApp account"
    }
  }
}
```

### Timeout Configuration

| Operation | Timeout |
|-----------|---------|
| Check Mesocycle Status | 10 seconds |
| Renew Maintain | 30 seconds |
| Renew Change Days | 30 seconds |
| Renew Rotate Exercises | 60 seconds |
| Update Profile | 30 seconds |
| GymRatForm Execution | 120 seconds |

---

## Node JSON Snippets

### Complete HTTP_Check_Mesocycle_Status

```json
{
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
      "timeout": 10000,
      "response": {
        "response": {
          "fullResponse": false
        }
      }
    }
  },
  "type": "n8n-nodes-base.httpRequest",
  "typeVersion": 4.2,
  "position": [-4400, 25600],
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
}
```

### Complete Execute_Mesocycle_Renewal (with error handling)

```json
{
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
  "position": [-4000, 25500],
  "id": "execute-mesocycle-renewal",
  "name": "Execute_Mesocycle_Renewal",
  "alwaysOutputData": true,
  "onError": "continueRegularOutput"
}
```

### Complete Updated Renewal_Agent System Prompt

```json
{
  "parameters": {
    "promptType": "define",
    "text": "={{ $json.user_message }}",
    "options": {
      "systemMessage": "=Eres \"FitBot\", el asistente de renovacion de mesociclo. Tu mision es ayudar al usuario a decidir como continuar despues de completar su ciclo de 4 semanas.\n\n## CONTEXTO DEL USUARIO\n- user_id: {{ $json.user_id }}\n- Nombre: {{ $json.user_name }}\n- Mesociclo completado: {{ $json.current_mesocycle }}\n- Dias por semana actuales: {{ $json.current_days_per_week }}\n\n## INTENCIONES A DETECTAR\nDebes identificar la intencion del usuario y retornar EXACTAMENTE una de estas opciones:\n\n1. **MANTENER_RUTINA**: El usuario quiere mantener los mismos ejercicios y frecuencia\n   - Palabras clave: \"mantener\", \"misma rutina\", \"igual\", \"seguir asi\", \"continuar\", \"repetir\"\n\n2. **CAMBIAR_DIAS**: El usuario quiere cambiar la cantidad de dias por semana\n   - Palabras clave: \"cambiar dias\", \"mas dias\", \"menos dias\", \"entrenar X dias\"\n   - IMPORTANTE: Debes obtener el nuevo numero de dias (2-6)\n\n3. **ROTAR_EJERCICIOS**: El usuario quiere diferentes ejercicios pero mantener la frecuencia\n   - Palabras clave: \"nuevos ejercicios\", \"cambiar ejercicios\", \"rotar\", \"variar\"\n\n4. **MODIFICAR_PERFIL**: El usuario quiere cambiar sus preferencias de entrenamiento\n   - Palabras clave: \"cambiar prioridades\", \"nueva lesion\", \"menos tiempo\", \"mas tiempo\", \"cambiar musculos\"\n\n5. **PREGUNTAR_OPCIONES**: El usuario necesita mas informacion o no ha decidido\n   - Palabras clave: \"que opciones\", \"no se\", \"ayuda\", preguntas generales\n\n## FLUJO DE CONVERSACION\n\n### Si es la primera interaccion o PREGUNTAR_OPCIONES:\nResponde con:\n\"!Felicitaciones {{ $json.user_name }}! Has completado tu mesociclo de 4 semanas.\n\nTienes estas opciones para continuar:\n\n1. Mantener rutina: Repites los mismos ejercicios con progresion de carga. Ideal si te sientes comodo y ves resultados.\n\n2. Cambiar dias: Puedes entrenar mas o menos dias por semana (actualmente {{ $json.current_days_per_week }} dias).\n\n3. Rotar ejercicios: Nuevos ejercicios para nuevos estimulos, manteniendo tu frecuencia.\n\n4. Modificar perfil: Actualiza tus prioridades, reporta lesiones, o cambia duracion de sesion.\n\nQue prefieres?\"\n\n### Si detectas MANTENER_RUTINA:\nResponde confirmando y retorna: INTENCION:MANTENER_RUTINA\n\n### Si detectas CAMBIAR_DIAS:\n- Si el usuario ya especifico el numero de dias, confirma y retorna: INTENCION:CAMBIAR_DIAS:X (donde X es el numero)\n- Si no especifico, pregunta: \"Cuantos dias por semana quieres entrenar? (2-6 dias)\"\n\n### Si detectas ROTAR_EJERCICIOS:\nResponde confirmando y retorna: INTENCION:ROTAR_EJERCICIOS\n\n### Si detectas MODIFICAR_PERFIL:\nResponde confirmando y retorna: INTENCION:MODIFICAR_PERFIL\n\n## REGLAS\n1. Siempre se motivador y positivo\n2. Los dias disponibles son SOLO: 2, 3, 4, 5, 6\n3. Si el usuario pide algo fuera de rango, sugiere el mas cercano\n4. La ultima linea de tu respuesta DEBE ser el codigo de intencion cuando la detectes\n5. Formato obligatorio cuando detectes intencion: INTENCION:NOMBRE_INTENCION o INTENCION:CAMBIAR_DIAS:X\n6. Todo en espanol"
    }
  },
  "type": "@n8n/n8n-nodes-langchain.agent",
  "typeVersion": 3.1,
  "position": [240, 240],
  "id": "renewal-agent",
  "name": "Renewal_Agent",
  "executeOnce": true
}
```

---

## Environment Variables Required

| Variable | Description | Example |
|----------|-------------|---------|
| `WORKOUT_API_URL` | Base URL for Go backend | `https://workout-api.run.app` |

## Credentials Required

| Credential | Type | Purpose |
|------------|------|---------|
| `Workout API Internal` | HTTP Header Auth | Internal API key for backend |
| `WhatsApp account` | WhatsApp Business API | Send messages |
| `Supabase Memory` | PostgreSQL | Chat memory storage |
| `OpenAi account` | OpenAI API | AI agent LLM |

---

## Implementation Checklist

### Main Flow (GymRatFlow)

- [ ] Add `HTTP_Check_Mesocycle_Status` node
- [ ] Add `If_Mesocycle_Complete` node
- [ ] Add `Execute_Mesocycle_Renewal` node
- [ ] Update `Intention_Agent` system prompt
- [ ] Update `Switch` node with `RENOVAR_MESOCICLO` branch
- [ ] Connect new nodes properly
- [ ] Test auto-detection path
- [ ] Test manual renewal request path

### Renewal Subflow (GymBotMesocycleRenewal)

- [ ] Replace `Reset_For_Mantener` with `HTTP_Renew_Maintain`
- [ ] Replace `Prepare_Days_Change` with `HTTP_Renew_Change_Days`
- [ ] Replace exercise rotation nodes with `HTTP_Renew_Rotate_Exercises`
- [ ] Add `MODIFICAR_PERFIL` path
- [ ] Update `Switch_Intention` node
- [ ] Update `Renewal_Agent` system prompt
- [ ] Add error handling nodes
- [ ] Test all 4 renewal paths

### GymRatForm Integration

- [ ] Add new input parameters
- [ ] Modify `ProcessUserPreferences` for overrides
- [ ] Add conditional user creation skip
- [ ] Add conditional notification skip
- [ ] Test renewal scenarios

### Testing

- [ ] Test MANTENER_RUTINA via HTTP
- [ ] Test CAMBIAR_DIAS via HTTP
- [ ] Test ROTAR_EJERCICIOS via HTTP
- [ ] Test MODIFICAR_PERFIL via HTTP + GymRatForm
- [ ] Test memory cleanup
- [ ] Test error handling
- [ ] Full E2E test from WhatsApp

---

## Authors

- Specification: Claude Code (Opus 4.5)
- Date: 2026-02-01
