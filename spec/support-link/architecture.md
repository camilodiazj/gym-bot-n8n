# KAN-92: Support Link - Architecture

## Problem

Users have no way to request help when they encounter issues (email not delivered, wrong routine, confusion about next steps). Zero support escape hatch in any message.

## Solution

Append a one-line support footer to terminal/static messages where the user is "left alone" after receiving the message.

## Data Flow

```
n8n Variable ($vars.SUPPORT_LINK)
        |
        v
+-------+-------------------+
|  WhatsApp Messages (6)    |
|  Append: plain text footer|
+---------------------------+
        |
+-------+-------------------+
|  Email Footer (1)         |
|  Append: HTML <a> link    |
+---------------------------+
```

## Tech Decision: `$vars.SUPPORT_LINK`

| Option | Pros | Cons | Decision |
|--------|------|------|----------|
| Hardcoded URL in each node | Zero setup | Change = edit 7 nodes | Rejected |
| `$vars.SUPPORT_LINK` | Single source of truth, matches `$vars.FRONTEND_URL` pattern | Manual n8n setup | **Selected** |
| Environment variable | Same benefits | Requires n8n restart | Rejected |

## Placement Criteria

A message gets the support link if ALL of these are true:
1. It's a **terminal message** (conversation ends after it)
2. It's **static or semi-static** (not mid-conversation AI output)
3. The user may need help with **next steps**

## Scope: 7 Nodes Across 3 Workflows

| Workflow | Node | Why included |
|----------|------|-------------|
| WORKOUT_CREATOR | NotifyRoutineCreated | Post-onboarding, user may have email issues |
| WORKOUT_CREATOR | GenerateRoutineHTML (email) | Email footer, non-intrusive |
| GymBotMesocycleRenewal | Send_Confirmation_Mantener | Terminal, user navigates alone |
| GymBotMesocycleRenewal | Send_Confirmation_Rotar | Terminal, user navigates alone |
| WeeklySchedulingPrompt | set_celebration_msg | Proactive outbound with CTA |
| WeeklySchedulingPrompt | set_growth_msg | Partial completion, moderate risk |
| WeeklySchedulingPrompt | set_reengagement_msg | Zero sessions, highest churn risk |

## Excluded

| Node | Reason |
|------|--------|
| AI agent responses (KYC, scheduling, confirmation, chat, renewal) | Disrupts conversational UX |
| MorningReminder template | Requires Meta re-approval |
| VER_RUTINA_DE_HOY display | Info-dense, user actively engaged |
| Rest day message | Low-value, user isn't stuck |
| Calendar event nodes (KAN-57) | No user-facing messages added |

## Format

**WhatsApp:**
```
{existing message}

¿Necesitas ayuda? Escríbenos: {{ $vars.SUPPORT_LINK }}
```

**Email HTML:**
```html
<p style="...footer...">
  Generado por Kairos Personal Trainer<br>
  <a href="{{ $vars.SUPPORT_LINK }}">¿Necesitas ayuda? Escríbenos aquí</a>
</p>
```
