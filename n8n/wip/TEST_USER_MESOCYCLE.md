# Usuarios de Prueba - Renovación de Mesociclo

## ✅ 5 Usuarios Creados para Testing

| # | Nombre | WhatsApp | User ID | Mesociclo | Schedule | Propósito |
|---|--------|----------|---------|-----------|----------|-----------|
| 1 | Test_MesocycleRenewal | 570000000099 | 43109d04-ac72-42c6-8f5e-188047cef604 | 2 | Sin schedule | TC_MESO_001: Detección automática |
| 2 | Test_Mantener | 570000000100 | a298b12a-54e8-4a06-b20a-4a6d8b80aca1 | 1 | Semana 4 ✅ | TC_MESO_002: MANTENER_RUTINA |
| 3 | Test_CambiarDias | 570000000101 | a705cf9c-4745-47f4-aa15-f24697b0cec7 | 1 | Semana 4 ✅ | TC_MESO_003: CAMBIAR_DIAS |
| 4 | Test_RotarEjercicios | 570000000102 | ec0abf34-9ef8-43bf-83f0-9adb63886117 | 1 | Semana 4 ✅ | TC_MESO_004: ROTAR_EJERCICIOS |
| 5 | Test_IntencionManual | 570000000103 | 6b027107-4cf4-49f2-8a93-98bbf71dee36 | 1 | Semana 1 (activo) | TC_MESO_005: Intención manual |

---

### Usuario 1: Test_MesocycleRenewal (Base/TC_MESO_001)
- **Nombre**: Test_MesocycleRenewal
- **WhatsApp**: 570000000099
- **User ID**: 43109d04-ac72-42c6-8f5e-188047cef604
- **Email**: test_mesocycle@gymbot.test
- **Mesociclo actual**: 2 (ya renovado)
- **Días por semana**: 4 (ul_4 - Upper/Lower split)
- **Goal**: Ganar masa muscular
- **Nivel**: Intermedio
- **Template**: tpl_ul_4_hyp_int
- **Workouts**: 8 workouts (semanas 1 y 4)
- **Schedule**: Sin schedule activo
- **Propósito**: Usuario base para detección automática inicial

### Usuario 2: Test_Mantener (TC_MESO_002)
- **Nombre**: Test_Mantener
- **WhatsApp**: 570000000100
- **User ID**: a298b12a-54e8-4a06-b20a-4a6d8b80aca1
- **Email**: test_mantener@gymbot.test
- **Mesociclo actual**: 1
- **Días por semana**: 4 (ul_4)
- **Goal**: Ganar masa muscular
- **Nivel**: Intermedio
- **Workouts**: 16 workouts (semanas 1 y 4)
- **Schedule**: Semana 4 completada (4/4 sesiones ✅)
- **Propósito**: Probar opción MANTENER_RUTINA

### Usuario 3: Test_CambiarDias (TC_MESO_003)
- **Nombre**: Test_CambiarDias
- **WhatsApp**: 570000000101
- **User ID**: a705cf9c-4745-47f4-aa15-f24697b0cec7
- **Email**: test_cambiar@gymbot.test
- **Mesociclo actual**: 1
- **Días por semana**: 4 (ul_4)
- **Goal**: Ganar masa muscular
- **Nivel**: Intermedio
- **Workouts**: 16 workouts (semanas 1 y 4)
- **Schedule**: Semana 4 completada (4/4 sesiones ✅)
- **Propósito**: Probar opción CAMBIAR_DIAS (ej: cambiar a 3 días)

### Usuario 4: Test_RotarEjercicios (TC_MESO_004)
- **Nombre**: Test_RotarEjercicios
- **WhatsApp**: 570000000102
- **User ID**: ec0abf34-9ef8-43bf-83f0-9adb63886117
- **Email**: test_rotar@gymbot.test
- **Mesociclo actual**: 1
- **Días por semana**: 4 (ul_4)
- **Goal**: Ganar masa muscular
- **Nivel**: Intermedio
- **Workouts**: 16 workouts (semanas 1 y 4)
- **Schedule**: Semana 4 completada (4/4 sesiones ✅)
- **Propósito**: Probar opción ROTAR_EJERCICIOS

### Usuario 5: Test_IntencionManual (TC_MESO_005)
- **Nombre**: Test_IntencionManual
- **WhatsApp**: 570000000103
- **User ID**: 6b027107-4cf4-49f2-8a93-98bbf71dee36
- **Email**: test_manual@gymbot.test
- **Mesociclo actual**: 1
- **Días por semana**: 4 (ul_4)
- **Goal**: Ganar masa muscular
- **Nivel**: Intermedio
- **Workouts**: 16 workouts (semanas 1 y 4)
- **Schedule**: Semana 1 activa (4 sesiones pendientes)
- **Propósito**: Probar intención manual RENOVAR_MESOCICLO

---

## 🧪 Cómo Probar el Feature

### Escenario 1: TC_MESO_001 - Detección Automática (Test_MesocycleRenewal)

**Usuario**: 570000000099 (Test_MesocycleRenewal)
**Estado**: Mesociclo 2, sin schedule activo

#### Paso 1: Enviar mensaje al bot
```
Usuario: 570000000099
Mensaje: "Hola, quiero agendar mi próxima semana"
```

#### Resultado Esperado:
El flujo **GymRatFlow_Supabase_V3** debería:
1. Detectar que no hay scheduled workouts
2. Ejecutar `Check_Mesocycle_Complete`
3. Verificar: `week4_completed >= days_per_week`
4. `If_Mesocycle_Complete` → TRUE o FALSE según historial
5. Si TRUE: Ejecutar subflow `GymBotMesocycleRenewal`

---

### Escenario 2: TC_MESO_002 - Opción MANTENER_RUTINA (Test_Mantener)

**Usuario**: 570000000100 (Test_Mantener)
**Estado**: Semana 4 completada (4/4 sesiones ✅)

#### Paso 1: Enviar mensaje inicial
```
Usuario: 570000000100
Mensaje: "Hola"
```

#### Resultado Esperado:
- Detección automática activa
- Mensaje con opciones de renovación

#### Paso 2: Seleccionar MANTENER
```
Usuario: 570000000100
Mensaje: "1" o "Mantener la rutina"
```

#### Resultado Esperado:
- `Reset_Schedule`: DELETE FROM user_weekly_schedule
- `Increment_Mesocycle`: mesocycle_number = 2
- Mensaje de confirmación: "¡Mesociclo 2 iniciado!"

---

### Escenario 3: TC_MESO_003 - Opción CAMBIAR_DIAS (Test_CambiarDias)

**Usuario**: 570000000101 (Test_CambiarDias)
**Estado**: Semana 4 completada (4/4 sesiones ✅)

#### Paso 1: Enviar mensaje inicial
```
Usuario: 570000000101
Mensaje: "Hola"
```

#### Paso 2: Seleccionar CAMBIAR_DIAS
```
Usuario: 570000000101
Mensaje: "2" o "Quiero cambiar a 3 días"
```

#### Resultado Esperado:
- `Extract_New_Days`: Detecta "3 días"
- `Delete_Old_Workouts`: DELETE workouts
- `Execute_GymRatForm` con is_renewal=true, override_days_available=3
- Nueva rutina generada con fb_3 schedule
- mesocycle_number = 2

---

### Escenario 4: TC_MESO_004 - Opción ROTAR_EJERCICIOS (Test_RotarEjercicios)

**Usuario**: 570000000102 (Test_RotarEjercicios)
**Estado**: Semana 4 completada (4/4 sesiones ✅)

#### Paso 1: Enviar mensaje inicial
```
Usuario: 570000000102
Mensaje: "Hola"
```

#### Paso 2: Seleccionar ROTAR_EJERCICIOS
```
Usuario: 570000000102
Mensaje: "3" o "Rotar ejercicios"
```

#### Resultado Esperado:
- `Get_Current_Workouts`: Lee workouts semana 1
- `Loop_Rotate_Exercises`: Para cada ejercicio
  - Busca alternativas por pattern
  - UPDATE workouts con nuevos exercise_id
- `Reset_Schedule`: DELETE user_weekly_schedule
- `Increment_Mesocycle`: mesocycle_number = 2
- Mensaje de confirmación con ejercicios rotados

---

### Escenario 5: TC_MESO_005 - Intención Manual (Test_IntencionManual)

**Usuario**: 570000000103 (Test_IntencionManual)
**Estado**: Semana 1 activa (schedule pendiente)

#### Paso 1: Mensaje con intención de renovar
```
Usuario: 570000000103
Mensaje: "Quiero cambiar mi rutina"
```

#### Resultado Esperado:
- `Intention_Agent` detecta: RENOVAR_MESOCICLO
- `Switch` redirige a rama RENOVAR_MESOCICLO
- Ejecuta subflow `GymBotMesocycleRenewal`
- Usuario recibe opciones de renovación (a pesar de tener schedule activo)

---

## 🔬 Test Cases

### TC_MESO_001: Detección Automática
**Usuario**: 570000000099 (Test_MesocycleRenewal)
**Input**: Usuario sin schedule, puede tener o no semana 4 completa
**Expected**: Detección automática se ejecuta
**Verify**:
```sql
-- Verificar que Check_Mesocycle_Complete se ejecutó
SELECT
  COUNT(DISTINCT uws.day_routine_id) FILTER (WHERE uws.week = 4 AND uws."Completed" = true) as week4_completed,
  ws.days_per_week,
  CASE WHEN COUNT(DISTINCT uws.day_routine_id) FILTER (WHERE uws.week = 4 AND uws."Completed" = true) >= ws.days_per_week
    THEN true ELSE false END as mesocycle_complete
FROM users u
JOIN users_plans up ON u.user_id = up.user_id
JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
LEFT JOIN user_weekly_schedule uws ON u.user_id = uws.user_id
WHERE u.user_id = '43109d04-ac72-42c6-8f5e-188047cef604'
GROUP BY ws.days_per_week;
```

### TC_MESO_002: Opción MANTENER
**Usuario**: 570000000100 (Test_Mantener)
**Input**: Usuario responde "1" o "mantener"
**Action**:
- DELETE FROM user_weekly_schedule
- UPDATE users_plans SET mesocycle_number = 2
**Verify**:
```sql
SELECT mesocycle_number, last_renewal_date
FROM users_plans
WHERE user_id = 'a298b12a-54e8-4a06-b20a-4a6d8b80aca1';
-- mesocycle_number debe ser 2
-- last_renewal_date debe ser NOW()

SELECT COUNT(*) as schedule_count
FROM user_weekly_schedule
WHERE user_id = 'a298b12a-54e8-4a06-b20a-4a6d8b80aca1';
-- Debe ser 0 (schedule limpiado)
```

### TC_MESO_003: Opción CAMBIAR DÍAS
**Usuario**: 570000000101 (Test_CambiarDias)
**Input**: Usuario responde "2" o "cambiar a 3 días"
**Action**:
- DELETE FROM workouts
- Ejecutar GymRatForm v3 con is_renewal=true, override_days_available=3
**Verify**:
```sql
SELECT week_schedule, mesocycle_number,
       (SELECT COUNT(*) FROM workouts WHERE user_id = up.user_id) as workout_count
FROM users_plans up
WHERE user_id = 'a705cf9c-4745-47f4-aa15-f24697b0cec7';
-- week_schedule debe ser 'fb_3' (3 días)
-- mesocycle_number debe ser 2
-- workout_count debe ser > 0 (nueva rutina generada)
```

### TC_MESO_004: Opción ROTAR EJERCICIOS
**Usuario**: 570000000102 (Test_RotarEjercicios)
**Input**: Usuario responde "3" o "rotar"
**Action**:
- Loop sobre workouts semana 1
- Buscar alternativas por pattern
- UPDATE workouts con nuevos exercise_id
- DELETE FROM user_weekly_schedule
- UPDATE users_plans SET mesocycle_number = 2
**Verify**:
```sql
-- Verificar que ejercicios cambiaron
WITH original_exercises AS (
  SELECT exercise_id FROM workouts
  WHERE user_id = 'ec0abf34-9ef8-43bf-83f0-9adb63886117' AND week = 4
),
new_exercises AS (
  SELECT exercise_id FROM workouts
  WHERE user_id = 'ec0abf34-9ef8-43bf-83f0-9adb63886117' AND week = 1
)
SELECT COUNT(*) as changed_exercises
FROM new_exercises ne
WHERE ne.exercise_id NOT IN (SELECT exercise_id FROM original_exercises);
-- Debe haber al menos 1 ejercicio diferente

SELECT mesocycle_number FROM users_plans
WHERE user_id = 'ec0abf34-9ef8-43bf-83f0-9adb63886117';
-- Debe ser 2
```

### TC_MESO_005: Intención Manual
**Usuario**: 570000000103 (Test_IntencionManual)
**Input**: Usuario con schedule activo dice "quiero cambiar mi rutina"
**Expected**: Intention_Agent detecta RENOVAR_MESOCICLO
**Action**: Ejecuta mismo subflow de renovación
**Verify**:
```sql
-- Verificar que tiene schedule activo
SELECT COUNT(*) as active_schedule
FROM user_weekly_schedule
WHERE user_id = '6b027107-4cf4-49f2-8a93-98bbf71dee36'
AND "Completed" = false;
-- Debe ser > 0

-- Después de ejecutar renovación:
SELECT mesocycle_number FROM users_plans
WHERE user_id = '6b027107-4cf4-49f2-8a93-98bbf71dee36';
-- Debe actualizarse según la opción elegida
```

---

## 🔍 Queries de Verificación

### Ver estado actual del usuario
```sql
SELECT
  u.full_name,
  up.mesocycle_number,
  up.last_renewal_date,
  ws.days_per_week,
  ws.schedule_type,
  COUNT(DISTINCT uws.day_routine_id) FILTER (WHERE uws.week = 4 AND uws."Completed" = true) as week4_completed
FROM users u
JOIN users_plans up ON u.user_id = up.user_id
JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
LEFT JOIN user_weekly_schedule uws ON u.user_id = uws.user_id
WHERE u.full_phone_number = '570000000099'
GROUP BY u.user_id, u.full_name, up.mesocycle_number, up.last_renewal_date, ws.days_per_week, ws.schedule_type;
```

### Ver workouts por semana
```sql
SELECT
  week,
  day_name,
  COUNT(*) as exercises,
  string_agg(DISTINCT e.spanish_name, ', ' ORDER BY e.spanish_name) as exercise_names
FROM workouts w
JOIN exercises e USING (exercise_id)
WHERE user_id = '43109d04-ac72-42c6-8f5e-188047cef604'
GROUP BY week, day_name
ORDER BY week, day_name;
```

### Ver schedule completo
```sql
SELECT
  week,
  week_day,
  session_name,
  planned_day,
  "Completed"
FROM user_weekly_schedule
WHERE user_id = '43109d04-ac72-42c6-8f5e-188047cef604'
ORDER BY week, planned_day;
```

---

## 🔄 Resetear Usuarios de Prueba

Para restaurar los usuarios a su estado inicial y poder volver a ejecutar las pruebas:

**Archivo**: [RESET_TEST_USERS.sql](RESET_TEST_USERS.sql)

Este script:
1. ✅ Limpia memoria del agente de renovación (n8n_chat_histories)
2. ✅ Restaura `mesocycle_number = 1` para todos
3. ✅ Limpia todos los schedules
4. ✅ Recrea semana 4 completa para usuarios 100, 101, 102
5. ✅ Recrea semana 1 activa para usuario 103
6. ✅ Verifica el estado final

**Resultado esperado:**
- Test_MesocycleRenewal: mesocycle=1, 0 schedules
- Test_Mantener: mesocycle=1, 4 schedules (4 completados)
- Test_CambiarDias: mesocycle=1, 4 schedules (4 completados)
- Test_RotarEjercicios: mesocycle=1, 4 schedules (4 completados)
- Test_IntencionManual: mesocycle=1, 4 schedules (0 completados)

---

## 🧹 Limpiar Usuarios de Prueba

Cuando termines las pruebas, eliminar todos los usuarios:

```sql
-- Eliminar en orden (respetando foreign keys)
-- User 1: Test_MesocycleRenewal
DELETE FROM user_weekly_schedule WHERE user_id = '43109d04-ac72-42c6-8f5e-188047cef604';
DELETE FROM workouts WHERE user_id = '43109d04-ac72-42c6-8f5e-188047cef604';
DELETE FROM users_plans WHERE user_id = '43109d04-ac72-42c6-8f5e-188047cef604';
DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000099;
DELETE FROM users WHERE user_id = '43109d04-ac72-42c6-8f5e-188047cef604';

-- User 2: Test_Mantener
DELETE FROM user_weekly_schedule WHERE user_id = 'a298b12a-54e8-4a06-b20a-4a6d8b80aca1';
DELETE FROM workouts WHERE user_id = 'a298b12a-54e8-4a06-b20a-4a6d8b80aca1';
DELETE FROM users_plans WHERE user_id = 'a298b12a-54e8-4a06-b20a-4a6d8b80aca1';
DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000100;
DELETE FROM users WHERE user_id = 'a298b12a-54e8-4a06-b20a-4a6d8b80aca1';

-- User 3: Test_CambiarDias
DELETE FROM user_weekly_schedule WHERE user_id = 'a705cf9c-4745-47f4-aa15-f24697b0cec7';
DELETE FROM workouts WHERE user_id = 'a705cf9c-4745-47f4-aa15-f24697b0cec7';
DELETE FROM users_plans WHERE user_id = 'a705cf9c-4745-47f4-aa15-f24697b0cec7';
DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000101;
DELETE FROM users WHERE user_id = 'a705cf9c-4745-47f4-aa15-f24697b0cec7';

-- User 4: Test_RotarEjercicios
DELETE FROM user_weekly_schedule WHERE user_id = 'ec0abf34-9ef8-43bf-83f0-9adb63886117';
DELETE FROM workouts WHERE user_id = 'ec0abf34-9ef8-43bf-83f0-9adb63886117';
DELETE FROM users_plans WHERE user_id = 'ec0abf34-9ef8-43bf-83f0-9adb63886117';
DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000102;
DELETE FROM users WHERE user_id = 'ec0abf34-9ef8-43bf-83f0-9adb63886117';

-- User 5: Test_IntencionManual
DELETE FROM user_weekly_schedule WHERE user_id = '6b027107-4cf4-49f2-8a93-98bbf71dee36';
DELETE FROM workouts WHERE user_id = '6b027107-4cf4-49f2-8a93-98bbf71dee36';
DELETE FROM users_plans WHERE user_id = '6b027107-4cf4-49f2-8a93-98bbf71dee36';
DELETE FROM users_gym_profile WHERE whatsapp_id = 570000000103;
DELETE FROM users WHERE user_id = '6b027107-4cf4-49f2-8a93-98bbf71dee36';
```

### Cleanup rápido (todos los usuarios)

```sql
-- Eliminar todos los usuarios de prueba de una vez
DELETE FROM user_weekly_schedule
WHERE user_id IN (
  '43109d04-ac72-42c6-8f5e-188047cef604',
  'a298b12a-54e8-4a06-b20a-4a6d8b80aca1',
  'a705cf9c-4745-47f4-aa15-f24697b0cec7',
  'ec0abf34-9ef8-43bf-83f0-9adb63886117',
  '6b027107-4cf4-49f2-8a93-98bbf71dee36'
);

DELETE FROM workouts
WHERE user_id IN (
  '43109d04-ac72-42c6-8f5e-188047cef604',
  'a298b12a-54e8-4a06-b20a-4a6d8b80aca1',
  'a705cf9c-4745-47f4-aa15-f24697b0cec7',
  'ec0abf34-9ef8-43bf-83f0-9adb63886117',
  '6b027107-4cf4-49f2-8a93-98bbf71dee36'
);

DELETE FROM users_plans
WHERE user_id IN (
  '43109d04-ac72-42c6-8f5e-188047cef604',
  'a298b12a-54e8-4a06-b20a-4a6d8b80aca1',
  'a705cf9c-4745-47f4-aa15-f24697b0cec7',
  'ec0abf34-9ef8-43bf-83f0-9adb63886117',
  '6b027107-4cf4-49f2-8a93-98bbf71dee36'
);

DELETE FROM users_gym_profile
WHERE whatsapp_id IN (570000000099, 570000000100, 570000000101, 570000000102, 570000000103);

DELETE FROM users
WHERE user_id IN (
  '43109d04-ac72-42c6-8f5e-188047cef604',
  'a298b12a-54e8-4a06-b20a-4a6d8b80aca1',
  'a705cf9c-4745-47f4-aa15-f24697b0cec7',
  'ec0abf34-9ef8-43bf-83f0-9adb63886117',
  '6b027107-4cf4-49f2-8a93-98bbf71dee36'
);
```

---

## 📱 Testing en n8n

### Opción 1: Pinned Data
Crear pinned data en el nodo trigger de GymRatFlow_V3:
```json
{
  "messaging_product": "whatsapp",
  "metadata": {
    "display_phone_number": "573213413664",
    "phone_number_id": "914510145083991"
  },
  "contacts": [{
    "profile": {"name": "Test_MesocycleRenewal"},
    "wa_id": "570000000099"
  }],
  "messages": [{
    "from": "570000000099",
    "id": "wamid.TEST-MESOCYCLE-001",
    "timestamp": "1769085000",
    "text": {"body": "Hola, quiero agendar"},
    "type": "text"
  }]
}
```

### Opción 2: Execute Workflow
Desde otro workflow, llamar a GymRatFlow_V3 con:
```json
{
  "contacts": [{"wa_id": "570000000099"}],
  "messages": [{"from": "570000000099", "text": {"body": "test"}}],
  "metadata": {"phone_number_id": "914510145083991"}
}
```

---

## ✅ Checklist de Validación

### Usuarios de Prueba
- [x] Usuario 1: Test_MesocycleRenewal (570000000099) - TC_MESO_001
- [x] Usuario 2: Test_Mantener (570000000100) - TC_MESO_002
- [x] Usuario 3: Test_CambiarDias (570000000101) - TC_MESO_003
- [x] Usuario 4: Test_RotarEjercicios (570000000102) - TC_MESO_004
- [x] Usuario 5: Test_IntencionManual (570000000103) - TC_MESO_005

### Workflows
- [ ] GymRatFlow_Supabase_V3.json importado en n8n
- [ ] GymRatForm Supabase v3.json importado en n8n
- [ ] GymBotMesocycleRenewal.json importado en n8n
- [ ] Todos los workflows tienen "active": false inicialmente

### Funcionalidad Core
- [ ] Detección automática: Check_Mesocycle_Complete funciona
- [ ] If_Mesocycle_Complete redirige correctamente
- [ ] Subflow GymBotMesocycleRenewal se ejecuta
- [ ] Mensaje de renovación con 3 opciones llega por WhatsApp

### TC_MESO_002: MANTENER_RUTINA
- [ ] Parse_Intention detecta "mantener" o "1"
- [ ] Reset_Schedule elimina user_weekly_schedule
- [ ] Increment_Mesocycle actualiza mesocycle_number
- [ ] last_renewal_date se actualiza a NOW()
- [ ] Mensaje de confirmación enviado

### TC_MESO_003: CAMBIAR_DIAS
- [ ] Parse_Intention detecta "cambiar" o "2"
- [ ] Extract_New_Days extrae número correcto (ej: 3)
- [ ] Delete_Old_Workouts elimina workouts anteriores
- [ ] Execute_GymRatForm con is_renewal=true funciona
- [ ] Nueva rutina generada con week_schedule correcto
- [ ] mesocycle_number incrementado

### TC_MESO_004: ROTAR_EJERCICIOS
- [ ] Parse_Intention detecta "rotar" o "3"
- [ ] Get_Current_Workouts obtiene workouts semana 1
- [ ] Loop_Rotate_Exercises itera sobre ejercicios
- [ ] Select_Alternative encuentra alternativas por pattern
- [ ] Update_Exercise actualiza exercise_id
- [ ] Al menos 1 ejercicio cambiado
- [ ] Schedule limpiado y mesocycle_number incrementado

### TC_MESO_005: Intención Manual
- [ ] Intention_Agent detecta RENOVAR_MESOCICLO
- [ ] Switch redirige a rama RENOVAR_MESOCICLO
- [ ] Execute_Mesocycle_Renewal se ejecuta
- [ ] Funciona incluso con schedule activo

### Integración
- [ ] Postgres Chat Memory persiste conversación
- [ ] WhatsApp envía mensajes correctamente
- [ ] n8n_chat_histories guarda sesiones
- [ ] No hay errores en logs de n8n
