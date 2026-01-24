# Plan de Pruebas E2E - GymRatFlow_Supabase v4.0

## Guia Rapida de Ejecucion

### Paso 1: Setup Inicial (Solo Primera Vez)

1. Abrir Supabase SQL Editor
2. Ejecutar `e2e/test_data_setup.sql` completo (Secciones 1-5)
3. Verificar con Seccion 6 que los usuarios se crearon correctamente

```bash
# Archivos necesarios:
e2e/
├── test_data_setup.sql      # SQL para crear usuarios fixture
└── GymRatFlow_test_plan.md  # Este documento

n8n/
└── GymRatFlow_E2E_TestRunner.json  # Workflow de tests (importar en n8n)
```

### Paso 2: Importar Test Runner en n8n

1. Ir a n8n > Workflows > Import from File
2. Seleccionar `n8n/GymRatFlow_E2E_TestRunner.json`
3. Configurar credenciales:
   - **Postgres:** Apuntar a Supabase
   - **OpenAI API:** Para TC002_FULL_KYC (usuario simulado)
4. Verificar que el nodo "Execute GymRatFlow" apunte al workflow `GymRatFlow_Supabase`

### Paso 3: Ejecutar Tests

1. Abrir el workflow `GymRatFlow_E2E_TestRunner`
2. Click en **"Test Workflow"** (o ejecutar nodo "Run All Tests")
3. Esperar ~2-5 minutos (TC002_FULL_KYC toma mas tiempo)
4. Ver resultados en el nodo **"Generate Report"**

### Paso 4: Verificar Resultados

```
*📊 Test Report - GymRatFlow*
──────────────────────
*Status:* ✅ PASS
*Score:* 100% (9/9)

*📂 Categorías:*
✔ FILTRO_RUIDO: 1/1
✔ ONBOARDING: 1/1
✔ ONBOARDING_FULL: 1/1
✔ AGENDAR: 1/1
✔ DESCANSO: 1/1
✔ VER_RUTINA: 1/1
✔ CHAT: 1/1
✔ PENDING_TASK: 2/2
```

---

## Arquitectura de Tests

```
┌─────────────────────────────────────────────────────────────┐
│                    SETUP (Una vez)                          │
│  test_data_setup.sql → Crea usuarios 001-004 en Supabase   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              GymRatFlow_E2E_TestRunner.json                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │ Load Tests  │ -> │ Loop Tests  │ -> │  Generate   │     │
│  │  (9 cases)  │    │             │    │   Report    │     │
│  └─────────────┘    └──────┬──────┘    └─────────────┘     │
│                            │                                │
│         ┌──────────────────┼──────────────────┐            │
│         ▼                  ▼                  ▼            │
│   ┌──────────┐      ┌──────────┐      ┌──────────────┐     │
│   │  SINGLE  │      │  MULTI   │      │ MULTI_TURN   │     │
│   │  Tests   │      │  TURN    │      │     AI       │     │
│   │ TC001-12 │      │  (N/A)   │      │ TC002_FULL   │     │
│   └──────────┘      └──────────┘      └──────────────┘     │
│         │                                    │              │
│         ▼                                    ▼              │
│   Execute GymRatFlow              GPT-4o-mini simula       │
│   (sub-workflow)                  respuestas de usuario    │
└─────────────────────────────────────────────────────────────┘
```

---

## Usuarios Dummy

| Usuario | Phone | user_id | Estado | Tests |
|---------|-------|---------|--------|-------|
| `Test_NoSchedule` | `570000000001` | `e2e00001-...-000001` | Con plan, SIN schedule futuro | TC003 |
| `Test_RestDay` | `570000000002` | `e2e00002-...-000002` | Con schedule MAÑANA | TC004 |
| `Test_WithRoutine` | `570000000003` | `e2e00003-...-000003` | Con rutina HOY + workouts | TC006, TC007 |
| `Test_WithPendingTask` | `570000000004` | `e2e00004-...-000004` | Con pending_task CONFIRMAR_RUTINA | TC011, TC012 |
| `Test_NewUser` | `570000000009` | **CREADO POR TEST** | Usuario dinamico para KYC | TC002, TC002_FULL_KYC |

> **Importante:** Los usuarios 001-004 son **fixtures** creados por `test_data_setup.sql`. El usuario 009 es creado/eliminado dinamicamente por los tests de onboarding.

---

## Matriz de Casos de Prueba

| ID | Nombre | Usuario | Categoria | Prioridad | Tipo |
|----|--------|---------|-----------|-----------|------|
| TC001 | Bloqueo de Ruido - Status Update | - | FILTRO_RUIDO | CRITICAL | SINGLE |
| TC002 | Usuario Nuevo - Onboarding KYC (Greeting) | Test_NewUser | ONBOARDING | HIGH | SINGLE |
| **TC002_FULL_KYC** | **Onboarding Completo - KYC con IA** | Test_NewUser | ONBOARDING_FULL | CRITICAL | MULTI_TURN_AI |
| TC003 | Usuario sin Workouts - Flujo Agendar | Test_NoSchedule | AGENDAR | HIGH | SINGLE |
| TC004 | Dia de Descanso | Test_RestDay | DESCANSO | MEDIUM | SINGLE |
| TC006 | VER_RUTINA_DE_HOY | Test_WithRoutine | VER_RUTINA | HIGH | SINGLE |
| TC007 | CHAT - Pregunta General | Test_WithRoutine | CHAT | MEDIUM | SINGLE |
| TC011 | Confirmacion via Pending Task | Test_WithPendingTask | PENDING_TASK | CRITICAL | SINGLE |
| TC012 | Pending Task - No confirma | Test_WithPendingTask | PENDING_TASK | HIGH | SINGLE |

---

## Detalle de Casos de Prueba

### TC001: Bloqueo de Ruido - Status Update
**Prioridad:** CRITICAL | **Tipo:** SINGLE

**Objetivo:** Verificar que el flujo ignore mensajes de status (sent, delivered, read)

**Input:** Mensaje con `statuses` en lugar de `messages`

**Metrica:** `output === undefined || output === null || output === ''`

---

### TC002: Usuario Nuevo - Onboarding KYC (Greeting)
**Prioridad:** HIGH | **Tipo:** SINGLE | **Usuario:** `570000000009`

**Objetivo:** Usuario no registrado debe iniciar proceso de onboarding (primera respuesta)

**Cleanup Automatico:** Elimina todos los datos del usuario antes del test

**Input:** `"Hola, quiero empezar a entrenar"`

**Metrica:** `output.includes('conocerte') || output.includes('nombre') || output.includes('Coach')`

---

### TC002_FULL_KYC: Onboarding Completo con Usuario Simulado IA
**Prioridad:** CRITICAL | **Tipo:** MULTI_TURN_AI | **Usuario:** `570000000009`

**Objetivo:** Completar el flujo KYC completo usando GPT-4o-mini para simular las respuestas del usuario

**Usuario Simulado:**
- Nombre: Juan Perez Garcia
- Email: juan.perez.e2e@test.com
- Edad: 28, Sexo: M
- Estatura: 175cm, Peso: 75kg
- Objetivo: Ganar masa muscular
- Nivel: Intermedio
- Dias disponibles: 4

**Config:**
- Max turns: 25
- Completion indicators: "me pongo manos a la obra", "recibiras tu plan", "tu rutina 100%", etc.

**Verificacion DB (Ground Truth):**
```sql
SELECT COUNT(*) FROM users_gym_profile WHERE whatsapp_id = 570000000009; -- expected: 1
SELECT COUNT(*) FROM users WHERE full_phone_number = '570000000009'; -- expected: 1
SELECT COUNT(*) FROM users_plans WHERE user_id IN (...); -- expected: 1
SELECT COUNT(DISTINCT week) FROM workouts WHERE user_id IN (...); -- expected: 4
```

**Metrica:** `dbPassed === true` (todos los datos esperados existen en DB)

---

### TC003: Usuario sin Workouts - Flujo Agendar
**Prioridad:** HIGH | **Tipo:** SINGLE | **Usuario:** `570000000001`

**Objetivo:** Usuario sin rutinas futuras debe poder agendar su semana

**Cleanup Automatico:** Elimina scheduled workouts futuros del usuario

**Input:** `"Quiero agendar mi semana"`

**Metrica:** `output.includes('dias') || output.includes('días') || output.includes('semana')`

---

### TC004: Dia de Descanso
**Prioridad:** MEDIUM | **Tipo:** SINGLE | **Usuario:** `570000000002`

**Objetivo:** Usuario con workouts futuros pero ninguno hoy recibe mensaje de descanso

**Input:** `"Que hay para hoy?"`

**Metrica:** `output.includes('no tienes sesión') || output.includes('descansar') || output.includes('descanso')`

---

### TC006: VER_RUTINA_DE_HOY
**Prioridad:** HIGH | **Tipo:** SINGLE | **Usuario:** `570000000003`

**Objetivo:** Usuario solicita ver rutina, se muestra formateada con ejercicios

**Input:** `"Muestrame mi rutina de hoy"`

**Metrica:** `(output.includes('RUTINA') || output.includes('rutina')) && (output.includes('Series') || output.includes('series') || output.includes('ejercicio'))`

---

### TC007: CHAT - Pregunta General
**Prioridad:** MEDIUM | **Tipo:** SINGLE | **Usuario:** `570000000003`

**Objetivo:** Preguntas generales son respondidas por AI Agent

**Input:** `"Que ejercicio es mejor para biceps?"`

**Metrica:** `output !== undefined && output.length > 50`

---

### TC011: Confirmacion via Pending Task
**Prioridad:** CRITICAL | **Tipo:** SINGLE | **Usuario:** `570000000004`

**Objetivo:** Usuario con pending_task responde confirmando su rutina

**Precondiciones:**
- Usuario tiene rutina HOY con `Completed = false`
- `pending_tasks` tiene entrada con `task_type='CONFIRMAR_RUTINA'`, `status='pending'`

**Cleanup Automatico:**
```sql
UPDATE user_weekly_schedule SET "Completed" = false WHERE user_id = '...' AND planned_day = CURRENT_DATE::text;
UPDATE pending_tasks SET status = 'pending', resolved_at = NULL WHERE user_id = '...' AND task_type = 'CONFIRMAR_RUTINA';
```

**Input:** `"Si, ya terminé mi rutina"`

**Metrica:** `output.includes('Felicidades') || output.includes('excelente') || output.includes('increíble') || output.includes('genial') || output.includes('Completada')`

---

### TC012: Pending Task - No confirma
**Prioridad:** HIGH | **Tipo:** SINGLE | **Usuario:** `570000000004`

**Objetivo:** Usuario con pending_task responde pero no confirma (ej: "No pude hoy")

**Cleanup Automatico:** Reset pending_task status

**Input:** `"No pude hoy, me senti mal"`

**Metrica:** `output !== undefined && output.length > 30`

---

## Troubleshooting

### Test falla: "User already exists"
**Causa:** El cleanup no se ejecuto correctamente
**Solucion:** Ejecutar manualmente en Supabase:
```sql
DELETE FROM n8n_chat_histories WHERE session_id LIKE '570000000009%';
DELETE FROM workouts WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000009');
DELETE FROM user_weekly_schedule WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000009');
DELETE FROM pending_tasks WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000009');
DELETE FROM users_plans WHERE user_id IN (SELECT user_id FROM users WHERE full_phone_number = '570000000009');
DELETE FROM users WHERE full_phone_number = '570000000009';
DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000009;
```

### Test falla: "User not found" (TC003, TC004, TC006, TC007, TC011, TC012)
**Causa:** Los usuarios fixture no existen
**Solucion:** Ejecutar `e2e/test_data_setup.sql` en Supabase

### TC002_FULL_KYC falla pero DB tiene datos correctos
**Causa:** El test usa DB verification como ground truth
**Verificar:** Revisar el nodo "Aggregate AI Results" para ver detalles de DB checks

### TC003 falla con "has_planned_workouts = true"
**Causa:** El usuario tiene schedules futuros de ejecuciones previas
**Solucion:** El cleanup automatico deberia limpiar esto, pero si persiste:
```sql
DELETE FROM user_weekly_schedule
WHERE user_id = 'e2e00001-0000-0000-0000-000000000001'
AND planned_day >= CURRENT_DATE::text;
```

---

## Notas Importantes

1. **Phones Reservados:** `57000000000X` estan reservados para testing. NO usar para usuarios reales.

2. **Cleanup Automatico:** Los tests TC002, TC002_FULL_KYC y TC003 tienen cleanup queries que se ejecutan automaticamente antes de cada test.

3. **TC002_FULL_KYC:** Este test usa GPT-4o-mini para simular un usuario real completando el KYC. Es el test mas importante para validar el flujo de onboarding completo.

4. **DB Verification:** TC002_FULL_KYC valida que se creen correctamente: `users_gym_profile`, `users`, `users_plans`, y 4 semanas de `workouts`.

5. **Re-ejecucion:** Los tests son idempotentes - pueden ejecutarse multiples veces sin setup adicional (excepto si los fixtures se eliminan).

---

## Historial de Versiones

| Version | Fecha | Cambios |
|---------|-------|---------|
| v4.0 | 2026-01-24 | Tests embebidos en workflow, cleanup automatico, TC002_FULL_KYC con DB verification como ground truth |
| v3.0 | 2026-01-23 | Agregado pending_tasks flow (TC011, TC012), deprecado TC005/TC009 |
| v2.1 | 2026-01-22 | Inputs actualizados a formato array para match con webhook real |
| v1.0 | 2026-01-20 | Version inicial con TC001-TC010 |
