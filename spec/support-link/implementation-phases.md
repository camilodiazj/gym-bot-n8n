# KAN-92: Support Link Implementation Plan

## Pre-requisite: n8n Variable Setup

In n8n UI: **Settings > Variables > Add Variable**

| Name | Value |
|------|-------|
| `SUPPORT_LINK` | `https://wa.me/573001234567` *(replace with actual support WhatsApp number)* |

This follows the existing `FRONTEND_URL` variable pattern already used in production.

---

## Phase 1: WhatsApp Messages (6 nodes)

All nodes get the same footer appended:

```
\n\n---\n¿Necesitas ayuda? Escríbenos: {{ $vars.SUPPORT_LINK }}
```

### 1.1 WORKOUT_CREATOR.json -- NotifyRoutineCreated

- **Node ID**: `6f1f864d-d4c0-442e-aad3-c190b6278cf4`
- **Field**: `parameters.textBody`

**Before (last line):**
```
Cuando quieras, escríbeme y agendamos juntos tu semana de entrenamiento 📅
```

**After (last line):**
```
Cuando quieras, escríbeme y agendamos juntos tu semana de entrenamiento 📅\n\n---\n¿Necesitas ayuda? Escríbenos: {{ $vars.SUPPORT_LINK }}
```

Full textBody value:
```
=🎉 ¡Hola {{ $items('ProcessUserPreferences')[0].json.full_name }}!

¡Tu rutina de entrenamiento de 4 semanas ha sido creada exitosamente! 💪

📋 Tu plan está personalizado para:
• Objetivo: {{ $items('ProcessUserPreferences')[0].json.primary_goal }}
• Nivel: {{ $items('ProcessUserPreferences')[0].json.fitness_level }}
• Días por semana: {{ $items('ProcessUserPreferences')[0].json.days_available }}
• Músculos prioritarios: {{ $items('ProcessUserPreferences')[0].json.priority_muscles || 'Balanceado' }}

📧 Ya te enviamos tu rutina de la Semana 1 a *{{ $items('ProcessUserPreferences')[0].json.email }}*. Revísala con calma para familiarizarte con cada ejercicio y su técnica.

⚠️ Si no lo ves en tu bandeja de entrada, revisa la carpeta de SPAM o correo no deseado.

Cuando quieras, escríbeme y agendamos juntos tu semana de entrenamiento 📅

---
¿Necesitas ayuda? Escríbenos: {{ $vars.SUPPORT_LINK }}
```

---

### 1.2 GymBotMesocycleRenewal.json -- Send_Confirmation_Mantener

- **Node ID**: `1ddd8907-fa48-4561-9f75-04b75ec4d2bc`
- **Field**: `parameters.textBody`

**Before:**
```
=¡Perfecto {{ $items('Execute Workflow Trigger')[0].json.full_name }}! 🎉\n\nTu mesociclo {{ $json.mesocycle_number }} ha iniciado con la misma rutina. Ahora puedes agendar tu Semana 1.\n\n¡Escríbeme cuando quieras agendar! 💪
```

**After:**
```
=¡Perfecto {{ $items('Execute Workflow Trigger')[0].json.full_name }}! 🎉\n\nTu mesociclo {{ $json.mesocycle_number }} ha iniciado con la misma rutina. Ahora puedes agendar tu Semana 1.\n\n¡Escríbeme cuando quieras agendar! 💪\n\n---\n¿Necesitas ayuda? Escríbenos: {{ $vars.SUPPORT_LINK }}
```

---

### 1.3 GymBotMesocycleRenewal.json -- Send_Confirmation_Rotar

- **Node ID**: `81557265-c7fe-45ee-b6f9-e5acb1aaa43f`
- **Field**: `parameters.textBody`

**Before:**
```
=¡Excelente {{ $items('Execute Workflow Trigger')[0].json.full_name }}! 🔄\n\nHe rotado los ejercicios de tu rutina manteniendo la misma estructura de {{ $items('Execute Workflow Trigger')[0].json.days_per_week }} días por semana.\n\nAhora puedes agendar tu nueva Semana 1 con ejercicios frescos. ¡Escríbeme cuando estés listo! 💪
```

**After:**
```
=¡Excelente {{ $items('Execute Workflow Trigger')[0].json.full_name }}! 🔄\n\nHe rotado los ejercicios de tu rutina manteniendo la misma estructura de {{ $items('Execute Workflow Trigger')[0].json.days_per_week }} días por semana.\n\nAhora puedes agendar tu nueva Semana 1 con ejercicios frescos. ¡Escríbeme cuando estés listo! 💪\n\n---\n¿Necesitas ayuda? Escríbenos: {{ $vars.SUPPORT_LINK }}
```

---

### 1.4 WeeklySchedulingPrompt.json -- set_celebration_msg

- **Node ID**: `a1b2c3d4-0007-4000-8000-000000000007`
- **Field**: `parameters.assignments.assignments[0].value` (the `message` assignment)

**Before:**
```
=Felicidades {{ $json.full_name }}! Completaste todas tus {{ $json.total_sessions }} sesiones de la Semana {{ $json.current_week }}.\n\nTu constancia es admirable. Listo para seguir con la Semana {{ $json.current_week + 1 }}? Escribeme "agendar" y organizamos tus dias.
```

**After:**
```
=Felicidades {{ $json.full_name }}! Completaste todas tus {{ $json.total_sessions }} sesiones de la Semana {{ $json.current_week }}.\n\nTu constancia es admirable. Listo para seguir con la Semana {{ $json.current_week + 1 }}? Escribeme "agendar" y organizamos tus dias.\n\n---\n¿Necesitas ayuda? Escribenos: {{ $vars.SUPPORT_LINK }}
```

---

### 1.5 WeeklySchedulingPrompt.json -- set_growth_msg

- **Node ID**: `a1b2c3d4-0008-4000-8000-000000000008`
- **Field**: `parameters.assignments.assignments[0].value` (the `message` assignment)

**Before:**
```
=Hola {{ $json.full_name }}! Completaste {{ $json.completed_count }} de {{ $json.total_sessions }} sesiones en la Semana {{ $json.current_week }}.\n\nCada entrenamiento suma. Quieres programar tu Semana {{ $json.current_week + 1 }}? Escribeme "agendar" y arrancamos.
```

**After:**
```
=Hola {{ $json.full_name }}! Completaste {{ $json.completed_count }} de {{ $json.total_sessions }} sesiones en la Semana {{ $json.current_week }}.\n\nCada entrenamiento suma. Quieres programar tu Semana {{ $json.current_week + 1 }}? Escribeme "agendar" y arrancamos.\n\n---\n¿Necesitas ayuda? Escribenos: {{ $vars.SUPPORT_LINK }}
```

---

### 1.6 WeeklySchedulingPrompt.json -- set_reengagement_msg

- **Node ID**: `a1b2c3d4-0009-4000-8000-000000000009`
- **Field**: `parameters.assignments.assignments[0].value` (the `message` assignment)

**Before:**
```
=Hola {{ $json.full_name }}! Veo que la Semana {{ $json.current_week }} quedo sin entrenamientos.\n\nNo pasa nada, lo importante es volver. Quieres intentar con tu Semana {{ $json.current_week + 1 }}? Escribeme "agendar" y planeamos juntos.
```

**After:**
```
=Hola {{ $json.full_name }}! Veo que la Semana {{ $json.current_week }} quedo sin entrenamientos.\n\nNo pasa nada, lo importante es volver. Quieres intentar con tu Semana {{ $json.current_week + 1 }}? Escribeme "agendar" y planeamos juntos.\n\n---\n¿Necesitas ayuda? Escribenos: {{ $vars.SUPPORT_LINK }}
```

---

## Phase 2: Email HTML Footer (1 node)

### 2.1 WORKOUT_CREATOR.json -- GenerateRoutineHTML

- **Node ID**: `5d6a14ef-6d53-4204-801e-2c0ca4c47d35`
- **Field**: `parameters.jsCode` (Code node)
- **Lines**: 433-436 of the JS code

**Before (lines 433-436):**
```javascript
      <!-- Footer -->
      <p style="${CSS.footer}">
        Generado por Kairos Personal Trainer
      </p>
```

**After:**
```javascript
      <!-- Footer -->
      <p style="${CSS.footer}">
        Generado por Kairos Personal Trainer<br>
        <a href="${supportLink}" style="color: #e63946; text-decoration: underline;">¿Necesitas ayuda? Escríbenos</a>
      </p>
```

Additionally, add the `supportLink` constant near the top of the code where other variables are declared (after the `subject` line ~128):

```javascript
const supportLink = $vars.SUPPORT_LINK;
```

This keeps the template literal clean and the variable source explicit.

---

## Phase 3: Verification Checklist

### Pre-deployment

- [ ] `SUPPORT_LINK` variable exists in n8n Settings > Variables
- [ ] Variable value is a valid URL (e.g., `https://wa.me/57XXXXXXXXXX`)

### Per-node verification (test each workflow)

| # | Workflow | Node | Verify |
|---|----------|------|--------|
| 1 | WORKOUT_CREATOR | NotifyRoutineCreated | Trigger test execution; WhatsApp message ends with support line |
| 2 | GymBotMesocycleRenewal | Send_Confirmation_Mantener | Mock mesocycle mantener flow; message ends with support line |
| 3 | GymBotMesocycleRenewal | Send_Confirmation_Rotar | Mock mesocycle rotar flow; message ends with support line |
| 4 | WeeklySchedulingPrompt | set_celebration_msg | Run with full-completion test user; message ends with support line |
| 5 | WeeklySchedulingPrompt | set_growth_msg | Run with partial-completion test user; message ends with support line |
| 6 | WeeklySchedulingPrompt | set_reengagement_msg | Run with zero-session test user; message ends with support line |
| 7 | WORKOUT_CREATOR | GenerateRoutineHTML | Trigger full onboarding; email footer shows clickable support link below "Generado por Kairos" |

### Post-deployment

- [ ] `$vars.SUPPORT_LINK` resolves correctly at runtime (not literal text)
- [ ] Email HTML renders the `<a>` tag as a clickable link (test in Gmail + Outlook)
- [ ] WhatsApp `---` renders as a visible separator (not markdown-parsed)
- [ ] Support link is tappable on mobile WhatsApp

---

## File Summary

| File | Nodes Modified | Phase |
|------|---------------|-------|
| `n8n/running_flows/WORKOUT_CREATOR.json` | `NotifyRoutineCreated`, `GenerateRoutineHTML` | 1, 2 |
| `n8n/running_flows/GymBotMesocycleRenewal.json` | `Send_Confirmation_Mantener`, `Send_Confirmation_Rotar` | 1 |
| `n8n/running_flows/WeeklySchedulingPrompt.json` | `set_celebration_msg`, `set_growth_msg`, `set_reengagement_msg` | 1 |
