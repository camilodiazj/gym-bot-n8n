# Guia de Pruebas - Mesocycle Renewal Feature

## Pre-requisitos

### 1. Variables de Entorno

**Backend (Google Cloud Run):**
```bash
INTERNAL_API_KEY=<tu-api-key-minimo-32-caracteres>
SUPABASE_DB_URL=<tu-conexion-supabase>
GIN_MODE=release
PORT=8080
```

**n8n:**
```bash
WORKOUT_API_URL=https://workout-api-148665080566.us-central1.run.app
```

### 2. Desplegar Backend

```bash
cd workout-tracker-back

# Verificar tests
make test

# Build
make build

# Deploy a Cloud Run
gcloud run deploy workout-api \
  --source . \
  --project gen-lang-client-0432163259 \
  --region us-central1 \
  --set-env-vars "INTERNAL_API_KEY=zdbXRPM0TYlSwOMsHE5D2gk6y0TG/2nam/q+2Xn1pnk=,SUPABASE_DB_URL=postgresql://postgres.ixfdjvlrnxleilzlujxj:ECV8jah5kcr0cdk@xef@aws-0-us-west-2.pooler.supabase.com:6543/postgres"
```

### 3. Importar Workflows en n8n

1. **Backup del workflow actual:**
   - Exportar `GymRatFlow_Supabase_V2_Workout_Tracker` actual

2. **Importar workflows actualizados:**
   - `n8n/running_flows/GymRatFlow_Supabase_V2_Workout_Tracker.json`
   - `n8n/running_flows/GymBotMesocycleRenewal.json`

3. **Configurar credenciales:**
   - Actualizar IDs de credenciales en ambos workflows
   - Verificar `WORKOUT_API_URL` en variables de entorno

4. **Obtener Workflow ID:**
   - Despues de importar `GymBotMesocycleRenewal`, copiar su ID
   - Actualizar nodo `Execute_Mesocycle_Renewal` en GymRatFlow con el ID correcto

---

## Usuarios de Prueba

Ejecutar en Supabase SQL Editor:

```sql
-- Archivo: e2e/mesocycle_renewal_test_data.sql
-- Crea 4 usuarios de prueba con semana 4 completada
```

| Telefono | Usuario | Escenario |
|----------|---------|-----------|
| `570000000010` | Test_MesocycleComplete | Semana 4 completada (100%) |
| `570000000011` | Test_MesocyclePartial | Semana 4 parcial (50%) |
| `570000000012` | Test_MesocycleNotStarted | Sin completar semana 4 |
| `570000000013` | Test_MesocycleMultiple | Multiples mesociclos previos |

---

## Escenarios de Prueba

### Prueba 1: Verificar Deteccion Automatica

**Objetivo:** Verificar que el sistema detecta automaticamente cuando el usuario completa la semana 4.

**Pasos:**
1. Enviar mensaje de WhatsApp desde `570000000010`:
   ```
   Hola
   ```

2. **Resultado esperado:**
   - Sistema detecta mesociclo completado
   - Ejecuta sub-workflow de renovacion
   - Muestra opciones de renovacion:
     ```
     Has completado tu mesociclo de 4 semanas!
     Tienes las siguientes opciones:
     1. Mantener tu rutina actual (con progresion de carga)
     2. Cambiar los dias de entrenamiento
     3. Rotar ejercicios (mismos patrones, nuevos ejercicios)
     4. Modificar tu perfil (prioridades, restricciones)
     ```

---

### Prueba 2: Renovacion Manual

**Objetivo:** Verificar que el usuario puede solicitar renovacion manualmente.

**Pasos:**
1. Enviar mensaje desde `570000000010`:
   ```
   Quiero renovar mi rutina
   ```

2. **Resultado esperado:**
   - Intencion detectada: `RENOVAR_MESOCICLO`
   - Ejecuta sub-workflow de renovacion
   - Muestra opciones de renovacion

---

### Prueba 3: MANTENER_RUTINA

**Objetivo:** Verificar opcion de mantener rutina con progresion de carga.

**Pasos:**
1. Cuando aparezcan las opciones, responder:
   ```
   Quiero mantener mi rutina actual
   ```

2. **Resultado esperado:**
   - Llamada API: `POST /api/v1/plans/{userId}/renew/maintain`
   - Mensaje de confirmacion:
     ```
     Tu rutina se ha renovado para el mesociclo 2.
     Se ha aplicado una progresion de carga del 2.5-5%
     en tus ejercicios compuestos.
     ```

3. **Verificar en base de datos:**
   ```sql
   SELECT mesocycle_number, last_renewal_date
   FROM users_plans
   WHERE user_id = '<user_id>';
   -- mesocycle_number debe ser 2

   SELECT COUNT(*) FROM user_weekly_schedule
   WHERE user_id = '<user_id>';
   -- Debe ser 0 (schedule limpiado)
   ```

---

### Prueba 4: CAMBIAR_DIAS

**Objetivo:** Verificar cambio de frecuencia de entrenamiento.

**Pasos:**
1. Iniciar renovacion y responder:
   ```
   Quiero cambiar a 4 dias por semana
   ```

2. **Resultado esperado:**
   - Llamada API: `POST /api/v1/plans/{userId}/renew/change-days`
   - Ejecuta GymRatForm para regenerar rutina
   - Mensaje de confirmacion:
     ```
     Tu frecuencia de entrenamiento ha sido actualizada a 4 dias.
     Se ha generado una nueva rutina personalizada.
     ```

3. **Verificar en base de datos:**
   ```sql
   SELECT week_schedule FROM users_plans
   WHERE user_id = '<user_id>';
   -- Debe ser 'ul_4'

   SELECT DISTINCT day_name FROM workouts
   WHERE user_id = '<user_id>';
   -- Debe mostrar 4 dias diferentes
   ```

---

### Prueba 5: ROTAR_EJERCICIOS

**Objetivo:** Verificar rotacion de ejercicios manteniendo patrones.

**Pasos:**
1. Antes de la prueba, guardar ejercicios actuales:
   ```sql
   SELECT exercise_id, day_name FROM workouts
   WHERE user_id = '<user_id>'
   ORDER BY day_name, exercise_order;
   ```

2. Iniciar renovacion y responder:
   ```
   Quiero rotar los ejercicios
   ```

3. **Resultado esperado:**
   - Llamada API: `POST /api/v1/plans/{userId}/renew/rotate-exercises`
   - Mensaje con lista de nuevos ejercicios:
     ```
     Tus ejercicios han sido actualizados:
     - Press de banca -> Press inclinado con mancuernas
     - Sentadilla -> Prensa de piernas
     ...
     ```

4. **Verificar en base de datos:**
   ```sql
   SELECT w.exercise_id, e.spanish_name, e.pattern
   FROM workouts w
   JOIN exercises e ON w.exercise_id = e.exercise_id
   WHERE w.user_id = '<user_id>'
   ORDER BY w.day_name, w.exercise_order;
   -- Ejercicios diferentes pero mismos patterns
   ```

---

### Prueba 6: MODIFICAR_PERFIL

**Objetivo:** Verificar actualizacion de perfil y regeneracion de rutina.

**Pasos:**
1. Iniciar renovacion y responder:
   ```
   Quiero modificar mi perfil
   ```

2. El agente preguntara sobre cambios. Responder:
   ```
   Ahora quiero enfocarme en espalda y biceps,
   y tengo una lesion en el hombro
   ```

3. **Resultado esperado:**
   - Agente de modificacion de perfil recopila cambios
   - Llamada API: `POST /api/v1/plans/{userId}/renew/update-profile`
   - Ejecuta GymRatForm para regenerar rutina
   - Mensaje de confirmacion:
     ```
     Tu perfil ha sido actualizado y se ha generado
     una nueva rutina personalizada.
     ```

4. **Verificar en base de datos:**
   ```sql
   SELECT priority_muscles, health_status
   FROM users_gym_profile
   WHERE whatsapp_id = '570000000010';
   -- Debe reflejar nuevas prioridades

   SELECT COUNT(*) FROM workouts
   WHERE user_id = '<user_id>';
   -- Nueva rutina generada
   ```

---

### Prueba 7: Mesociclo Incompleto

**Objetivo:** Verificar que no se permite renovar sin completar semana 4.

**Pasos:**
1. Enviar mensaje desde `570000000012` (usuario sin completar):
   ```
   Quiero renovar mi rutina
   ```

2. **Resultado esperado:**
   - Sistema detecta mesociclo incompleto
   - Mensaje:
     ```
     Todavia no has completado tu mesociclo actual.
     Termina la semana 4 antes de renovar tu rutina.
     ```

---

### Prueba 8: API Directa (Backend)

**Objetivo:** Verificar endpoints de API directamente.

```bash
# Variables
API_URL="https://workout-api-148665080566.us-central1.run.app"
API_KEY="<tu-internal-api-key>"
USER_ID="<uuid-del-usuario>"

# 1. Check Mesocycle Status
curl -X GET "$API_URL/api/v1/plans/$USER_ID/mesocycle-status" \
  -H "X-API-Key: $API_KEY" | jq

# Respuesta esperada:
# {
#   "is_complete": true,
#   "current_mesocycle": 1,
#   "week4_completed": 4,
#   "week4_total": 4,
#   "completion_rate": 100,
#   "message": "Mesociclo completado..."
# }

# 2. Renew Maintain
curl -X POST "$API_URL/api/v1/plans/$USER_ID/renew/maintain" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" | jq

# 3. Renew Rotate Exercises
curl -X POST "$API_URL/api/v1/plans/$USER_ID/renew/rotate-exercises" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"rotate_compounds": true, "rotate_isolation": true}' | jq

# 4. Renew Change Days
curl -X POST "$API_URL/api/v1/plans/$USER_ID/renew/change-days" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"new_days_per_week": 4}' | jq

# 5. Renew Update Profile
curl -X POST "$API_URL/api/v1/plans/$USER_ID/renew/update-profile" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"priority_muscles": "Espalda, biceps", "health_status": "C"}' | jq
```

---

## Verificacion de Errores

### Error: API Key Invalida
```bash
curl -X GET "$API_URL/api/v1/plans/$USER_ID/mesocycle-status" \
  -H "X-API-Key: wrong-key"

# Respuesta: 403 Forbidden
```

### Error: Usuario No Encontrado
```bash
curl -X GET "$API_URL/api/v1/plans/00000000-0000-0000-0000-000000000000/mesocycle-status" \
  -H "X-API-Key: $API_KEY"

# Respuesta: 404 Not Found
```

### Error: Dias Invalidos
```bash
curl -X POST "$API_URL/api/v1/plans/$USER_ID/renew/change-days" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"new_days_per_week": 7}'

# Respuesta: 400 Bad Request - "days must be between 2 and 6"
```

---

## Checklist Final

- [ ] Backend desplegado y health check OK
- [ ] n8n workflows importados y activos
- [ ] Credenciales configuradas en n8n
- [ ] WORKOUT_API_URL configurado
- [ ] Workflow ID actualizado en Execute_Mesocycle_Renewal
- [ ] Usuarios de prueba creados
- [ ] Prueba 1: Deteccion automatica OK
- [ ] Prueba 2: Renovacion manual OK
- [ ] Prueba 3: MANTENER_RUTINA OK
- [ ] Prueba 4: CAMBIAR_DIAS OK
- [ ] Prueba 5: ROTAR_EJERCICIOS OK
- [ ] Prueba 6: MODIFICAR_PERFIL OK
- [ ] Prueba 7: Mesociclo incompleto OK
- [ ] Prueba 8: API directa OK

---

## Rollback

Si algo sale mal:

1. **Restaurar workflow anterior:**
   - Importar backup: `GymRatFlow_Supabase_V2_Workout_Tracker_backup_20260201_100405.json`

2. **Desactivar workflow de renovacion:**
   - Desactivar `GymBotMesocycleRenewal` en n8n

3. **Los endpoints de API no afectan funcionalidad existente** - solo agregan nuevas rutas

---

## Contacto

Si encuentras problemas durante las pruebas, revisar:
1. Logs de n8n (Executions)
2. Logs de Cloud Run: `gcloud run logs read workout-api`
3. Tabla `n8n_chat_histories` para contexto de conversacion
