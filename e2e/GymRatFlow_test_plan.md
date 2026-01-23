# Plan de Pruebas E2E - GymRatFlow_Supabase

## Usuarios Dummy

| Usuario | Phone | user_id | Estado | Tests |
|---------|-------|---------|--------|-------|
| `Test_NoSchedule` | `570000000001` | `e2e00001-...-000001` | Con plan, SIN schedule | TC003 |
| `Test_RestDay` | `570000000002` | `e2e00002-...-000002` | Con schedule MANANA | TC004 |
| `Test_WithRoutine` | `570000000003` | `e2e00003-...-000003` | Con rutina HOY + workouts | TC005-010 |
| `Test_NewUser` | `570000000009` | **NO EXISTE** | - | TC002 |

> **Setup:** Ejecutar `test_data_setup.sql` antes de correr los tests

---

## Resumen del Flujo

```
WhatsApp_Trigger1
       │
       ▼
    GetUser ──────────────────────────────────────────┐
       │                                              │
       ▼                                              │
  user_exists?                                        │
       │                                              │
   ┌───┴───┐                                          │
   │       │                                          │
FALSE    TRUE                                         │
   │       │                                          │
   ▼       ▼                                          │
KYC Agent  GetWeeklySchedule                          │
   │              │                                   │
   ▼              ▼                                   │
Send msg4   has_planned_workouts1?                    │
              │                                       │
          ┌───┴───┐                                   │
          │       │                                   │
        TRUE    FALSE                                 │
          │       │                                   │
          ▼       ▼                                   │
  Filter_Today   [Week_Schedule +                     │
  _Routine       User_Finished_Workouts +             │
          │       Template_Days]                      │
          ▼              │                            │
  userHasRoutine         ▼                            │
  ForToday?           Merge                           │
          │              │                            │
      ┌───┴───┐          ▼                            │
      │       │      AI Agent1 (Agendar)              │
    TRUE    FALSE        │                            │
      │       │          ▼                            │
      ▼       ▼      Send msg2                        │
Intention  Send msg1                                  │
_Agent     (Descanso)                                 │
      │                                               │
      ▼                                               │
   Switch ─────────────────────────────────────────────
      │
  ┌───┼───────────┐
  │   │           │
  ▼   ▼           ▼
CONFIRMAR  CHAT  VER_RUTINA
_RUTINA     │    _DE_HOY
  │         │         │
  ▼         ▼         ▼
CONFIRM.  AI Agent  AI Agent
AGENT        │         │
  │          ▼         ▼
  ▼      Send msg  Send msg
Send msg3
```

---

## Matriz de Casos de Prueba (v3.0)

| ID | Nombre | Usuario Dummy | Categoria | Prioridad | Estado |
|----|--------|---------------|-----------|-----------|--------|
| TC001 | Bloqueo de Ruido - Status Update | - | FILTRO_RUIDO | CRITICAL | [ ] |
| TC002 | Usuario Nuevo - Onboarding KYC | Test_NewUser | ONBOARDING | HIGH | [ ] |
| TC003 | Usuario sin Workouts Planeados | Test_NoSchedule | AGENDAR | HIGH | [ ] |
| TC004 | Dia de Descanso - Sin Rutina Hoy | Test_RestDay | DESCANSO | MEDIUM | [ ] |
| ~~TC005~~ | ~~Intencion CONFIRMAR_RUTINA~~ | - | DEPRECATED | SKIP | [x] |
| TC006 | Intencion VER_RUTINA_DE_HOY | Test_WithRoutine | VER_RUTINA | HIGH | [ ] |
| TC007 | Intencion CHAT - Pregunta General | Test_WithRoutine | CHAT | MEDIUM | [ ] |
| TC008 | Edge Case - Mensaje de Audio | Test_WithRoutine | EDGE_CASE | LOW | [ ] |
| ~~TC009~~ | ~~Variantes CONFIRMAR_RUTINA~~ | - | DEPRECATED | SKIP | [x] |
| TC010 | Variantes VER_RUTINA | Test_WithRoutine | VER_RUTINA | MEDIUM | [ ] |
| **TC011** | **Confirmacion via Pending Task** | Test_WithPendingTask | PENDING_TASK | **CRITICAL** | [ ] |
| TC012 | Pending Task - Usuario no confirma | Test_WithPendingTask | PENDING_TASK | HIGH | [ ] |

> **Nota v3.0:** TC005 y TC009 fueron deprecados. Las confirmaciones de rutina ahora requieren un `pending_task` creado por el reminder de las 8PM (GymBotWorkoutCompletion). Ver TC011 como flujo principal.

---

## Detalle de Casos de Prueba

### TC001: Bloqueo de Ruido - Status Update
**Prioridad:** CRITICAL | **Usuario:** Ninguno

**Objetivo:** Verificar que el flujo ignore mensajes de status (sent, delivered, read)

**Input:**
```json
{
  "messaging_product": "whatsapp",
  "metadata": {
    "display_phone_number": "573213413664",
    "phone_number_id": "914510145083991"
  },
  "statuses": [
    {
      "id": "wamid.xxx",
      "status": "sent",
      "timestamp": "1769084739",
      "recipient_id": "573208780020"
    }
  ],
  "field": "messages"
}
```

**Path Esperado:** `WhatsApp_Trigger1` -> `GetUser` (falla por no tener `messages[0].from`)

**Metrica:** `output === undefined || output === null`

---

### TC002: Usuario Nuevo - Onboarding KYC
**Prioridad:** HIGH | **Usuario:** Test_NewUser (`570000000009`)

**Objetivo:** Usuario no registrado debe iniciar proceso de onboarding

**Precondiciones:** Usuario con phone `570000000009` NO existe en DB

**Input:**
```json
{
  "contacts": [{ "profile": { "name": "Test NewUser" }, "wa_id": "570000000009" }],
  "messages": [{
    "from": "570000000009",
    "id": "wamid.E2E-TC002-NewUser",
    "text": { "body": "Hola, quiero empezar a entrenar" },
    "type": "text"
  }]
}
```

**Path Esperado:** `user_exists[FALSE]` -> `KYC Agent` -> `Send message4`

**Metrica:** `output.includes('conocerte') || output.includes('nombre')`

---

### TC003: Usuario sin Workouts Planeados - Flujo Agendar
**Prioridad:** HIGH | **Usuario:** Test_NoSchedule (`570000000001`)

**Objetivo:** Usuario sin rutinas futuras debe poder agendar su semana

**Precondiciones:** Usuario existe, `user_weekly_schedule` vacio

**Input:**
```json
{
  "contacts": [{ "profile": { "name": "Test NoSchedule" }, "wa_id": "570000000001" }],
  "messages": [{
    "from": "570000000001",
    "id": "wamid.E2E-TC003-NoSchedule",
    "text": { "body": "Quiero agendar mi semana" },
    "type": "text"
  }]
}
```

**Path Esperado:** `has_planned_workouts1[FALSE]` -> `Merge` -> `AI Agent1` -> `Send message2`

**Metrica:** `output.includes('dias') || output.includes('días') || output.includes('semana')`

---

### TC004: Dia de Descanso - Sin Rutina para Hoy
**Prioridad:** MEDIUM | **Usuario:** Test_RestDay (`570000000002`)

**Objetivo:** Usuario con workouts futuros pero ninguno hoy recibe mensaje de descanso

**Precondiciones:** Usuario tiene schedule para MANANA, no para HOY

**Input:**
```json
{
  "contacts": [{ "profile": { "name": "Test RestDay" }, "wa_id": "570000000002" }],
  "messages": [{
    "from": "570000000002",
    "id": "wamid.E2E-TC004-RestDay",
    "text": { "body": "Que hay para hoy?" },
    "type": "text"
  }]
}
```

**Path Esperado:** `Filter_Today_Routine` (retorna `[]`) -> `userHasRoutineForToday[FALSE]` -> `Send message1`

**Metrica:** `output.includes('no tienes sesión') || output.includes('descansar')`

---

### ~~TC005: Intencion CONFIRMAR_RUTINA~~ [DEPRECATED v3.0]
> **SKIP** - Este caso fue deprecado. Las confirmaciones ahora usan el flujo de `pending_tasks` (ver TC011).
>
> **Razón:** El Intention_Agent requería contexto en memoria del reminder de 8PM para detectar CONFIRMAR_RUTINA. Sin ese contexto, clasificaba como CHAT.

---

### TC011: Confirmacion via Pending Task (NUEVO v3.0)
**Prioridad:** CRITICAL | **Usuario:** Test_WithPendingTask (`570000000004`)

**Objetivo:** Usuario con pending_task responde confirmando su rutina. Flujo principal de confirmación.

**Precondiciones:**
- Usuario tiene rutina HOY con `Completed = false`
- `pending_tasks` tiene entrada con `task_type='CONFIRMAR_RUTINA'`, `status='pending'`

**Input:**
```json
{
  "contacts": [{ "profile": { "name": "Test WithPendingTask" }, "wa_id": "570000000004" }],
  "messages": [{
    "from": "570000000004",
    "id": "wamid.E2E-TC011-PendingConfirm",
    "text": { "body": "Si, ya terminé mi rutina" },
    "type": "text"
  }]
}
```

**Path Esperado:** `Check_Pending_Tasks` -> `Has_Pending_Task[TRUE]` -> `CONFIRMATION AGENT` -> `Update_Pending_Task_Completed` -> `Send message3`

**Verificacion DB:**
- `user_weekly_schedule.Completed = true`
- `pending_tasks.status = 'completed'`

**Cleanup:**
```sql
UPDATE user_weekly_schedule SET "Completed" = false
WHERE user_id = 'e2e00004-0000-0000-0000-000000000004' AND planned_day = CURRENT_DATE::text;

UPDATE pending_tasks SET status = 'pending', resolved_at = NULL
WHERE user_id = 'e2e00004-0000-0000-0000-000000000004' AND task_type = 'CONFIRMAR_RUTINA';
```

**Metrica:** `output.includes('felicit') || output.includes('excelente') || output.includes('genial')`

---

### TC012: Pending Task - Usuario no confirma
**Prioridad:** HIGH | **Usuario:** Test_WithPendingTask (`570000000004`)

**Objetivo:** Usuario con pending_task responde pero no confirma (ej: "No pude hoy")

**Input:**
```json
{
  "contacts": [{ "profile": { "name": "Test WithPendingTask" }, "wa_id": "570000000004" }],
  "messages": [{
    "from": "570000000004",
    "id": "wamid.E2E-TC012-PendingDecline",
    "text": { "body": "No pude hoy, me senti mal" },
    "type": "text"
  }]
}
```

**Path Esperado:** `Check_Pending_Tasks` -> `Has_Pending_Task[TRUE]` -> `CONFIRMATION AGENT` -> `Update_Pending_Task_Completed` -> `Send message3`

**Metrica:** `output !== undefined && output.length > 30`

---

### TC006: Intencion VER_RUTINA_DE_HOY
**Prioridad:** HIGH | **Usuario:** Test_WithRoutine (`570000000003`)

**Objetivo:** Usuario solicita ver rutina, se muestra formateada

**Precondiciones:** Usuario tiene rutina HOY + workouts asignados

**Input:**
```json
{
  "contacts": [{ "profile": { "name": "Test WithRoutine" }, "wa_id": "570000000003" }],
  "messages": [{
    "from": "570000000003",
    "id": "wamid.E2E-TC006-ViewRoutine",
    "text": { "body": "Muestrame mi rutina de hoy" },
    "type": "text"
  }]
}
```

**Path Esperado:** `Intention_Agent` -> `Switch[VER_RUTINA_DE_HOY]` -> `AI Agent`

**Metrica:** `output.includes('RUTINA') && output.includes('Series')`

---

### TC007: Intencion CHAT - Pregunta General
**Prioridad:** MEDIUM | **Usuario:** Test_WithRoutine (`570000000003`)

**Objetivo:** Preguntas generales son respondidas por AI Agent

**Input:**
```json
{
  "contacts": [{ "profile": { "name": "Test WithRoutine" }, "wa_id": "570000000003" }],
  "messages": [{
    "from": "570000000003",
    "id": "wamid.E2E-TC007-Chat",
    "text": { "body": "Que ejercicio es mejor para biceps?" },
    "type": "text"
  }]
}
```

**Path Esperado:** `Intention_Agent` -> `Switch[CHAT]` -> `AI Agent`

**Metrica:** `output !== undefined && output.length > 50`

---

### TC008: Edge Case - Mensaje de Audio
**Prioridad:** LOW | **Usuario:** Test_WithRoutine (`570000000003`)

**Objetivo:** El flujo no crashea con mensajes no-texto

**Input:**
```json
{
  "contacts": [{ "profile": { "name": "Test WithRoutine" }, "wa_id": "570000000003" }],
  "messages": [{
    "from": "570000000003",
    "id": "wamid.E2E-TC008-Audio",
    "audio": { "mime_type": "audio/ogg", "id": "audio123" },
    "type": "audio"
  }]
}
```

**Metrica:** `!crashed`

---

## Checklist de Ejecucion

### Setup Inicial (Una vez)
- [ ] Ejecutar `test_data_setup.sql` SECCION 1 (Teardown)
- [ ] Ejecutar `test_data_setup.sql` SECCIONES 2-5 (Setup)
- [ ] Ejecutar `test_data_setup.sql` SECCION 6 (Verificacion)

### Pre-Flight Check (Antes de Deploy)
- [ ] **TC001** - Bloqueo de Ruido
- [ ] **TC002** - Usuario nuevo inicia KYC
- [ ] **TC005** - Confirmacion de rutina actualiza DB
- [ ] **TC006** - Rutina se muestra con formato correcto

### Regression Check (Despues de Cambios en Prompts)
- [ ] **TC005** - Intencion CONFIRMAR_RUTINA detectada
- [ ] **TC006** - Intencion VER_RUTINA_DE_HOY detectada
- [ ] **TC007** - Intencion CHAT no interfiere
- [ ] **TC009** - Sinonimos de confirmacion
- [ ] **TC010** - Sinonimos de ver rutina

### Full Suite
- [ ] TC001 - TC010 todos pasando

---

## Como Ejecutar Pruebas

### Metodo Manual (Rapido)

1. Abrir workflow `GymRatFlow_Supabase` en n8n
2. Click en nodo `WhatsApp_Trigger1`
3. En panel derecho, usar "Pin Data"
4. Pegar JSON del caso de prueba (ver `GymRatFlow_test_cases.json`)
5. Click "Test Workflow"
6. Verificar path ejecutado y output

### Metodo Automatizado (n8n Evaluation)

Ver archivo `GymRatFlow_test_cases.json` para dataset completo con metricas.

---

## Notas Importantes

1. **Usuarios Dummy:** Los phones `57000000000X` estan reservados para testing. NO usar para usuarios reales.

2. **Cleanup TC005:** Despues de ejecutar, el registro quedara con `Completed = true`. Ejecutar cleanup SQL antes de repetir.

3. **Schedule Dinamico:** TC004 tiene schedule para "MANANA", que se calcula con `CURRENT_DATE + 1 day`. Si corres tests en dias consecutivos, el schedule de ayer sera "hoy".

4. **Memoria del Agente:** Para pruebas limpias del Intention_Agent, limpiar `n8n_chat_histories` con:
```sql
DELETE FROM n8n_chat_histories WHERE session_id LIKE '%57000000000%';
```
