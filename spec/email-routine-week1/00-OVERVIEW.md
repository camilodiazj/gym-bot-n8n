# Send Week 1 Routine via Email - Implementation Specification

**Feature:** Email delivery of Week 1 workout routine after plan creation
**Workflow:** `WORKOUT_CREATOR.json`
**Status:** Planned
**Created:** 2026-02-06
**Author:** Development Team

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Decision Record (ADR)](#2-architecture-decision-record-adr)
3. [System Architecture](#3-system-architecture)
4. [Data Flow](#4-data-flow)
5. [Team Roles & Responsibilities](#5-team-roles--responsibilities)
6. [Implementation Phases](#6-implementation-phases)
7. [Dependencies & Prerequisites](#7-dependencies--prerequisites)
8. [Risks & Mitigations](#8-risks--mitigations)
9. [Files Modified](#9-files-modified)
10. [Success Criteria](#10-success-criteria)
11. [Out of Scope (Future Enhancements)](#11-out-of-scope-future-enhancements)

---

## 1. Executive Summary

### Problem

When WORKOUT_CREATOR finishes generating a personalized 4-week workout plan, the user receives only a generic WhatsApp notification:

> "Tu rutina de entrenamiento de 4 semanas ha sido creada exitosamente! Escribeme cuando quieras ver tu rutina del dia!"

The user never receives the actual routine content. They must then send a separate WhatsApp message ("Ver rutina de hoy") for each individual day to see their exercises. There is no consolidated view of their Week 1 plan, no reference document they can save, and no way to review upcoming training days at a glance.

### Solution

Add an HTML email delivery step to the WORKOUT_CREATOR workflow. Immediately after the 4-week workout plan is saved to the database, the system queries Week 1 exercises with full details (exercise names, sets, reps, RIR, rest, tempo, equipment, video links) and sends a professionally formatted HTML email to the user's registered email address.

### Scope

- **Only Week 1** is included in the email. Weeks 2-4 use progressive overload parameters from `set_profiles` and are delivered separately (out of scope for this feature).
- The email serves as a reference document -- it does not replace the daily WhatsApp routine delivery or the Workout Tracker web app.

### Approach

- **n8n-only implementation** -- no changes to the Go backend, no Supabase Edge Functions, no new microservices.
- 4 new nodes added to the existing WORKOUT_CREATOR workflow as a parallel branch.
- Email failure is isolated and never blocks the WhatsApp notification or the workflow return value.

---

## 2. Architecture Decision Record (ADR)

### Decision

**Implement email delivery entirely within n8n using an HTML email (Option A).**

Four architectural options were evaluated:

| Option | Description | Pros | Cons | Verdict |
|--------|-------------|------|------|---------|
| **A: n8n-only HTML email** | Add a Code node to generate HTML + an Email Send node within the WORKOUT_CREATOR workflow | All workout data already available in the workflow context; no deployment pipeline changes; HTML renders natively on all devices and email clients; failure is isolated via parallel branch and `continueOnFail`; fastest time to delivery | Limited styling control (inline CSS only); SMTP credentials managed in n8n UI; no reusable API endpoint for other consumers | **SELECTED** |
| **B: Go Backend PDF + Email** | Add a new endpoint in `workout-tracker-back` that receives workout data, generates a PDF, and sends it via email | Better PDF rendering quality; reusable REST endpoint; unit-testable; type-safe Go implementation | Go backend has **zero email infrastructure** (no SMTP config, no SendGrid SDK, no email libraries in `go.mod`); requires adding PDF generation dependency (`go-pdf` or `wkhtmltopdf`); requires new deployment to Cloud Run; significant overhead for a single feature | **Rejected** |
| **C: Hybrid (n8n data -> backend PDF)** | n8n prepares the workout data payload and calls a new backend endpoint that generates the PDF and sends the email | Separation of concerns; backend handles rendering | Two failure points (n8n HTTP call + backend processing); more complex error handling; still requires all the backend infrastructure from Option B | **Rejected** |
| **D: Supabase Edge Function** | Create a Deno-based Edge Function deployed on Supabase that receives workout data, generates the email/PDF, and sends it | Serverless; scales automatically; co-located with database | Introduces new technology stack (Deno) unfamiliar to the team; separate deployment pipeline; Edge Function cold starts; no existing email sending infrastructure in the Supabase project | **Rejected** |

### Key Factors in the Decision

1. **No existing email infrastructure in Go backend.** The `workout-tracker-back` service has no SMTP configuration, no SendGrid/SES SDK, and no email-related packages in its dependency tree. Adding email capabilities would require significant foundational work unrelated to the feature goal.

2. **All data is already available.** The WORKOUT_CREATOR workflow already has the complete user profile (via `ProcessUserPreferences`), the `user_id` (via `GetUser`), and has just finished writing all workout rows to the `workouts` table. A single Postgres query retrieves the Week 1 exercises with exercise details.

3. **HTML email is more versatile than PDF on mobile.** Users interact with GymBot primarily via WhatsApp on their phones. An HTML email renders inline without requiring a download, opens instantly, and scrolls naturally. PDF attachments require a separate app to open and are harder to reference quickly at the gym.

4. **Parallel branch isolates failure.** By branching from the `Create a row` node output (rather than inserting into the serial chain), an SMTP failure or HTML generation error never blocks the existing WhatsApp notification (`NotifyRoutineCreated`) or the workflow return value (`Filtered Message2`). The main flow remains completely untouched.

---

## 3. System Architecture

### Current Workflow End (Existing)

The WORKOUT_CREATOR workflow currently ends with a linear chain after all workout rows are inserted:

```
ValidateWorkoutDuration --> Create a row --> NotifyRoutineCreated (WhatsApp) --> Filtered Message2 (return)
```

- **`Create a row`** (Supabase node, id: `4b83267d`): Inserts each workout row into the `workouts` table.
- **`NotifyRoutineCreated`** (WhatsApp node, id: `cfd10997`): Sends the generic "Tu rutina ha sido creada" message via WhatsApp Business API. Configured with `executeOnce: true`.
- **`Filtered Message2`** (Code node, id: `c379bae5`): Returns `{ output: 'routine created' }` to the calling workflow (MAIN_FLOW).

### New Workflow End (With Email Branch)

A parallel branch is added from `Create a row`'s output. The existing serial chain is unchanged:

```
                                                 +--> NotifyRoutineCreated (WhatsApp) --> Filtered Message2 (return)
                                                 |
Create a row --+---------------------------------+
               |
               +--> GetWeek1WithExercises (Postgres)
                        |
                        v
                    GenerateRoutineHTML (Code)
                        |
                        v
                    HasEmail (IF)
                        |
                   [true branch]
                        |
                        v
                    SendRoutineEmail (Email Send)
```

### Node Details

| Node Name | Node Type | Purpose | Key Configuration |
|-----------|-----------|---------|-------------------|
| `GetWeek1WithExercises` | `n8n-nodes-base.postgres` | Query Week 1 workouts joined with exercises table | `executeOnce: true`, uses existing Supabase Postgres credential (`vZLJtIWG5nYXMez4`) |
| `GenerateRoutineHTML` | `n8n-nodes-base.code` | JavaScript code that transforms query results into a complete HTML email string | `executeOnce: true`, outputs `{ html, email, subject, fullName }` |
| `HasEmail` | `n8n-nodes-base.if` | Guards against users with no email registered | Condition: `{{ $json.email }}` is not empty |
| `SendRoutineEmail` | `n8n-nodes-base.emailSend` | Sends the HTML email via SMTP | `continueOnFail: true`, uses new SMTP credential |

### Connection Changes in `WORKOUT_CREATOR.json`

The only modification to the existing connections is updating `Create a row`'s output to include a second connection entry. The current connection:

```json
"Create a row": {
  "main": [
    [
      { "node": "NotifyRoutineCreated", "type": "main", "index": 0 }
    ]
  ]
}
```

Becomes:

```json
"Create a row": {
  "main": [
    [
      { "node": "NotifyRoutineCreated", "type": "main", "index": 0 },
      { "node": "GetWeek1WithExercises", "type": "main", "index": 0 }
    ]
  ]
}
```

Both nodes receive the same output from `Create a row` and execute in parallel. The existing chain (`NotifyRoutineCreated --> Filtered Message2`) is completely untouched.

---

## 4. Data Flow

### Available Data at Each Stage

#### Stage 1: ProcessUserPreferences (already exists, accessible via `$('ProcessUserPreferences')`)

User profile fields available throughout the workflow:

| Field | Example Value | Usage in Email |
|-------|---------------|----------------|
| `full_name` | `"Xiomara Alejandra Diaz Ramirez"` | Email greeting, header |
| `email` | `"xiomara@example.com"` | Recipient address |
| `primary_goal` | `"Salud general / recomposicion corporal"` | Profile summary section |
| `fitness_level` | `"Principiante"` | Profile summary, progression notes |
| `days_available` | `5` | Weekly overview table |
| `priority_muscles` | `"Gluteo, pierna"` | Profile summary |
| `biological_sex` | `"F"` | Motivational copy adaptation |
| `age` | `28` | Profile summary |
| `height_cm` | `165` | Profile summary |
| `weight_kg` | `62` | Profile summary |
| `session_duration_mins` | `"45-60 minutos"` | Session time display |
| `training_experience` | `"Menos de 6 meses"` | Progression notes |
| `processed.environment` | `"HOME"` or `"GYM"` | Equipment section, exercise context |
| `processed.home.equipment_list` | `["bodyweight", "dumbbell", "resistance_band"]` | Equipment summary for HOME users |
| `processed.home.equipment_tier` | `"basic"` | Equipment tier display |
| `processed.health.avoid_upper_body_overhead` | `true` | Health notes in email footer |
| `processed.health.avoid_lower_body_impact` | `true` | Health notes in email footer |
| `processed.health.avoid_spinal_loading` | `true` | Health notes in email footer |
| `processed.health.special_condition` | `false` | Health notes in email footer |

#### Stage 2: GetUser (already exists, accessible via `$items('GetUser')`)

| Field | Example Value | Usage |
|-------|---------------|-------|
| `user_id` | `"a1b2c3d4-..."` (UUID) | Used in the SQL query to filter workouts |

#### Stage 3: GetWeek1WithExercises (NEW node -- Postgres query output)

This node executes a SQL query that joins `workouts` with `exercises` filtered to `week_number = 1` for the current user:

```sql
SELECT
  w.day_name,
  w.exercise_order,
  w.sets,
  w.reps,
  w.rir,
  w."rest-seconds" AS rest_seconds,
  w.tempo,
  e.spanish_name,
  e.main_muscle,
  e.secondary_muscles,
  e.equipment,
  e.link,
  e.role
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = '{{ $items("GetUser")[0].json.user_id }}'
  AND w.week = 1
ORDER BY
  CASE w.day_name
    WHEN 'Push' THEN 1
    WHEN 'Pull' THEN 2
    WHEN 'Legs' THEN 3
    WHEN 'Upper' THEN 4
    WHEN 'Upper (Arms)' THEN 4
    WHEN 'Lower' THEN 5
    WHEN 'Lower (Glutes)' THEN 5
    WHEN 'Full Body' THEN 6
  END,
  w.exercise_order ASC;
```

**Output shape** (one row per exercise):

| Column | Type | Example |
|--------|------|---------|
| `day_name` | string | `"Push"` |
| `exercise_order` | integer | `1` |
| `sets` | string | `"3"` |
| `reps` | string | `"10-12"` |
| `rir` | string | `"3"` |
| `rest_seconds` | integer | `90` |
| `tempo` | string | `"2-0-2-0"` |
| `role` | string | `"compound"` |
| `spanish_name` | string | `"Flexiones De Rodillas"` |
| `main_muscle` | string | `"Triceps"` |
| `secondary_muscles` | string[] | `["Chest"]` |
| `equipment` | string | `"bodyweight"` |
| `link` | string | `"https://musclewiki.com/..."` |

#### Stage 4: GenerateRoutineHTML (NEW node -- Code output)

The JavaScript code node processes all query rows and the user profile to produce:

| Output Field | Type | Description |
|--------------|------|-------------|
| `html` | string | Complete HTML email body with inline CSS, table-based layout, all exercise tables per day, profile summary, weekly overview, warmup/progression/nutrition notes |
| `email` | string | Recipient email address (copied from `ProcessUserPreferences.email`) |
| `subject` | string | Email subject line, e.g., `"Tu Plan de Entrenamiento - Semana 1 | Kairos Personal Trainer"` |
| `fullName` | string | User's full name for the email "To" field display |

#### Stage 5: HasEmail (IF node)

- **True branch** (email is not empty): Proceeds to `SendRoutineEmail`.
- **False branch** (no email registered): Execution stops gracefully. No error, no email sent.

#### Stage 6: SendRoutineEmail (Email Send node)

Consumes the output of `GenerateRoutineHTML`:

| Email Field | Source |
|-------------|--------|
| From | SMTP credential default sender (configured in n8n) |
| To | `{{ $json.email }}` |
| Subject | `{{ $json.subject }}` |
| HTML Body | `{{ $json.html }}` |

---

## 5. Team Roles & Responsibilities

| Role | Agent | Responsibilities | Spec Document |
|------|-------|-------------------|---------------|
| **n8n Expert** | n8n-agent | Implement all 4 workflow nodes (`GetWeek1WithExercises`, `GenerateRoutineHTML`, `HasEmail`, `SendRoutineEmail`); update the `Create a row` connection to add the parallel branch; configure SMTP credential in n8n; test end-to-end email delivery within the workflow | `01-N8N-WORKFLOW-SPEC.md` |
| **Developer** | pixel-dev | Write the JavaScript code for the `GenerateRoutineHTML` node; implement all helper functions (`translateEquipment`, `formatRest`, `formatSecondaryMuscles`, `getRoleLabel`, `getRoleHeaderParams`, `generateMotivationalQuote`); generate inline-CSS HTML that matches the format established in `e2e/rutina_xiomara_semana1.md`; ensure responsive table-based layout for email clients | `02-HTML-TEMPLATE-SPEC.md` |
| **QA Analyst** | code-reviewer | Validate email rendering across Gmail (web + Android + iOS), Outlook (desktop + web), and Apple Mail; test edge cases (no email, HOME user with band equipment, GYM user, health restriction codes B/C/D/E, 3-day vs 5-day vs 6-day plans); verify all existing E2E tests still pass without modification; verify SMTP failure does not break main flow | `03-QA-TEST-PLAN.md` |
| **Coach** | kiro-coach | Review routine content accuracy in the email; validate exercise grouping logic (compound first, then core, then isolation); confirm warmup/progression/nutrition note copy is appropriate for all fitness levels; approve motivational messaging tone; verify equipment translations are correct for HOME users | `04-CONTENT-REVIEW-CHECKLIST.md` |

### Collaboration Touchpoints

1. **n8n-agent** provides the node scaffolding and the SQL query; **pixel-dev** fills in the `GenerateRoutineHTML` JavaScript code.
2. **pixel-dev** delivers the HTML template; **code-reviewer** tests rendering and edge cases.
3. **kiro-coach** reviews the rendered email content and provides feedback to **pixel-dev** for copy adjustments.
4. All roles validate their work against the [Success Criteria](#10-success-criteria) before marking their phase complete.

---

## 6. Implementation Phases

### Phase 1: SMTP Setup (n8n-agent)

**Duration:** ~30 minutes

**Tasks:**
1. Select an SMTP provider:
   - **MVP:** Gmail App Password (sufficient for low-volume testing and initial rollout)
   - **Production recommendation:** Resend, Amazon SES, or SendGrid (higher deliverability, SPF/DKIM support)
2. Create SMTP credentials in the n8n instance:
   - Navigate to n8n Settings > Credentials > Add Credential > SMTP
   - Configure: host, port (587 for TLS), username, password, sender email, sender name ("Kairos Personal Trainer")
3. Test with a simple manual workflow:
   - Create a temporary test workflow with a single Email Send node
   - Send a basic "Hello World" email to a test address
   - Confirm delivery and check spam folder
4. Document the credential name (needed for Phase 2)

**Exit criteria:** A test email is successfully delivered to a Gmail inbox without landing in spam.

---

### Phase 2: Workflow Nodes (n8n-agent)

**Duration:** ~1 hour

**Tasks:**
1. **Add `GetWeek1WithExercises` node** (Postgres):
   - Type: `n8n-nodes-base.postgres`
   - Operation: `executeQuery`
   - Credential: `Supabase Memory` (id: `vZLJtIWG5nYXMez4`) -- same as existing Postgres nodes
   - Query: The SQL from [Stage 3](#stage-3-getweek1withexercises-new-node----postgres-query-output) above
   - Set `executeOnce: true`
   - Position: Below the main flow (e.g., `[1312, 400]` -- same x as `Create a row`, offset y)

2. **Add `GenerateRoutineHTML` node** (Code):
   - Type: `n8n-nodes-base.code`
   - TypeVersion: `2`
   - Set `executeOnce: true`
   - JavaScript code: Placeholder initially, replaced by pixel-dev in Phase 3
   - Position: `[1552, 400]`

3. **Add `HasEmail` node** (IF):
   - Type: `n8n-nodes-base.if`
   - Condition: `{{ $json.email }}` is not empty (string, isNotEmpty)
   - Position: `[1760, 400]`

4. **Add `SendRoutineEmail` node** (Email Send):
   - Type: `n8n-nodes-base.emailSend`
   - To: `{{ $json.email }}`
   - Subject: `{{ $json.subject }}`
   - HTML Body: `{{ $json.html }}`
   - Credential: SMTP credential created in Phase 1
   - Set `continueOnFail: true` (critical -- prevents SMTP errors from crashing the workflow)
   - Position: `[1952, 400]`

5. **Update connections:**
   - Modify `Create a row` output to include both `NotifyRoutineCreated` and `GetWeek1WithExercises`
   - Connect `GetWeek1WithExercises` --> `GenerateRoutineHTML`
   - Connect `GenerateRoutineHTML` --> `HasEmail`
   - Connect `HasEmail` (true branch) --> `SendRoutineEmail`

6. **Test with placeholder code:**
   - Set `GenerateRoutineHTML` to output a simple `<h1>Test</h1>` HTML
   - Run WORKOUT_CREATOR with a test user that has an email
   - Confirm the email arrives while WhatsApp notification still works

**Exit criteria:** Placeholder email is delivered; WhatsApp notification is unaffected; workflow returns `{ output: 'routine created' }` as before.

---

### Phase 3: HTML Template (pixel-dev)

**Duration:** ~2-3 hours

**Tasks:**
1. Write the `GenerateRoutineHTML` JavaScript code that:
   - Reads all exercise rows from `GetWeek1WithExercises` via `$('GetWeek1WithExercises').all()`
   - Reads user profile from `$('ProcessUserPreferences').first().json`
   - Groups exercises by `day_name`
   - Within each day, groups exercises by `role` (compound, core, isolation)
   - Generates the set/rep parameters header for each role group (e.g., "3 sets x 10-12 reps | RIR 3 | 90s descanso")
   - Creates HTML tables matching the format in `e2e/rutina_xiomara_semana1.md`
   - Includes all required sections:
     - Header with user name and plan overview
     - Quick reference guide (sets, reps, RIR, rest definitions)
     - Weekly overview table
     - Per-day exercise tables with video links
     - Warmup notes
     - Progression guidance
     - Hydration and nutrition tips
     - Motivational closing quote

2. Implement helper functions:
   - `translateEquipment(equipment)`: Maps database values (`dumbbell`, `barbell`, `bodyweight`, `resistance_band`, `machine`, `cable`, `kettlebell`) to Spanish display names (`Mancuerna`, `Barra`, `Peso corporal`, `Banda`, `Maquina`, `Cable`, `Kettlebell`)
   - `formatRest(seconds)`: Converts integer seconds to display format (e.g., `90` -> `90s`, `120` -> `2 min`)
   - `formatSecondaryMuscles(muscles)`: Joins the `secondary_muscles` array with the main muscle for display (e.g., `"Triceps, Pecho"`)
   - `getRoleLabel(role)`: Maps `compound` -> `"Ejercicios Compuestos"`, `core` -> `"Core"`, `isolation` -> `"Ejercicios de Aislamiento"`
   - `getRoleHeaderParams(exercises)`: Extracts the representative sets/reps/rir/rest for a role group header line

3. Ensure HTML email compatibility:
   - Use only inline CSS (no `<style>` blocks -- many email clients strip `<head>` styles)
   - Use table-based layout (not flexbox or grid -- Outlook does not support them)
   - Use web-safe fonts with fallbacks (`Arial, Helvetica, sans-serif`)
   - Include `alt` attributes on any images
   - Use absolute URLs for all links
   - Set explicit widths on table cells
   - Avoid CSS shorthand properties (Outlook compatibility)
   - Test with [Email on Acid](https://www.emailonacid.com/) or [Litmus](https://www.litmus.com/) if available

4. Handle HOME vs GYM differences:
   - HOME users: Show "Equipo" column header and equipment per exercise; add equipment summary at top
   - GYM users: Equipment column can show general equipment type

5. Handle health restrictions:
   - If any health restriction flag is `true`, include a health notice section in the email footer

**Exit criteria:** Running WORKOUT_CREATOR produces a complete, well-formatted HTML email that visually matches the structure of `e2e/rutina_xiomara_semana1.md`.

---

### Phase 4: Content Review (kiro-coach)

**Duration:** ~30 minutes

**Tasks:**
1. Receive a rendered email from a test WORKOUT_CREATOR run
2. Review exercise grouping:
   - Compound exercises appear first (exercise_order 1-4)
   - Core exercises appear after compounds (exercise_order 5-6)
   - Isolation exercises appear last (exercise_order 7+)
3. Validate the role group headers accurately reflect the set/rep parameters from `set_profiles`
4. Confirm warmup notes are appropriate for the user's fitness level
5. Confirm progression notes match the 4-week mesocycle structure (Weeks 1-2: technique, Week 3: deload, Week 4: intensification)
6. Review nutrition tips for general accuracy
7. Approve motivational quote tone and language
8. Sign off on the email content

**Exit criteria:** Coach approves the email content with no required changes, or provides specific feedback that pixel-dev implements.

---

### Phase 5: QA Validation (code-reviewer)

**Duration:** ~1-2 hours

**Tasks:**
1. **Functional testing:**
   - Run WORKOUT_CREATOR with a GYM user (5 days) -- verify email is received with correct content
   - Run WORKOUT_CREATOR with a HOME user (3 days) -- verify equipment column and HOME-specific content
   - Run WORKOUT_CREATOR with a user who has health restriction code C -- verify health notes appear
   - Run WORKOUT_CREATOR with a user who has NO email -- verify no error, workflow completes normally

2. **Email rendering testing:**
   - Open the received email in Gmail (web browser)
   - Open the received email in Gmail (Android app)
   - Open the received email in Gmail (iOS app)
   - Open the received email in Outlook (desktop)
   - Open the received email in Outlook (web)
   - Open the received email in Apple Mail (macOS)
   - Verify: tables are aligned, links are clickable, text is readable, no broken styles

3. **Failure isolation testing:**
   - Temporarily misconfigure the SMTP credential (wrong password)
   - Run WORKOUT_CREATOR
   - Verify: WhatsApp notification is still sent, workflow returns `{ output: 'routine created' }`, no uncaught errors

4. **Regression testing:**
   - Run the full E2E test suite (`GymRatFlow_E2E_TestRunner.json`)
   - Verify all 12 existing test cases pass without modification
   - Pay special attention to `TC002_FULL_KYC` which exercises the WORKOUT_CREATOR flow

5. **Edge case testing:**
   - User with 6-day plan (large email) -- verify scrolling and readability
   - User with 3-day plan (small email) -- verify layout is not broken with fewer sections
   - Exercise with very long `spanish_name` -- verify table does not break
   - Exercise with no `link` (null) -- verify no broken anchor tag

**Exit criteria:** All tests pass, email renders correctly in the top 3 email clients, failure isolation is confirmed, and all existing E2E tests pass.

---

## 7. Dependencies & Prerequisites

### Required Before Development Starts

| Dependency | Description | Owner | Status |
|------------|-------------|-------|--------|
| **SMTP service account** | Gmail App Password for MVP. For production: Resend API key, Amazon SES credentials, or SendGrid API key. Must support TLS on port 587. | n8n-agent | Pending |
| **n8n Email Send node** | The `n8n-nodes-base.emailSend` node. This is a built-in n8n node -- no installation or npm packages required. Available in all n8n versions >= 0.150. | -- | Available |
| **Test user with email** | At least one user in `users_gym_profile` must have a valid `email` field populated. The existing test user `573123623296` (pinned in WORKOUT_CREATOR) should have an email set. | QA | Pending |
| **SMTP sender domain** (production only) | If using a custom domain (e.g., `kairos@gymbot.com`), SPF, DKIM, and DMARC DNS records must be configured to prevent spam classification. Not required for MVP with Gmail. | DevOps | Not started |

### Technical Prerequisites

- n8n instance must be running and accessible
- Postgres credential `Supabase Memory` (id: `vZLJtIWG5nYXMez4`) must be configured (already exists)
- The `workouts` and `exercises` tables must contain data (populated by the WORKOUT_CREATOR flow itself)
- The `users_gym_profile` table must have the `email` column populated for target users

---

## 8. Risks & Mitigations

| # | Risk | Impact | Probability | Mitigation |
|---|------|--------|-------------|------------|
| R1 | **SMTP rate limiting** -- Gmail or SMTP provider blocks sends due to volume | Emails delayed or permanently blocked; user does not receive routine | Low (GymBot has low user volume, well under Gmail's 500/day free tier limit) | Use a dedicated transactional email service (Resend, SES, SendGrid) for production. Monitor SMTP response codes in n8n execution logs. |
| R2 | **Email classified as spam** -- User never sees the routine email | User does not receive the email, defeating the feature purpose | Medium (new sender domain, no sending reputation, HTML email with links) | Configure SPF, DKIM, and DMARC DNS records on the sender domain. Use a recognizable sender name ("Kairos Personal Trainer"). Avoid spam trigger words. Include unsubscribe link in footer. Ask users to add sender to contacts. |
| R3 | **HTML renders differently across email clients** -- Tables break in Outlook, fonts differ in Apple Mail | Poor user experience, unprofessional appearance | Medium (Outlook's rendering engine is notoriously limited) | Use only inline CSS, table-based layout, and web-safe fonts. Avoid flexbox, grid, CSS shorthand, and `<style>` blocks. Test in the top 3 clients before release. Use `mso-` conditional comments for Outlook-specific fixes if needed. |
| R4 | **User has no email registered** -- `email` field is null or empty in `users_gym_profile` | Feature does not activate for that user | Low (KYC flow collects email, but it may be optional) | The `HasEmail` IF node skips email sending gracefully when no email is present. No error is thrown. The WhatsApp notification still works. |
| R5 | **Large routine (6 days) makes email very long** -- User has to scroll extensively | Reduced readability, user may not engage with the full content | Low (HTML email scrolling is a natural UX on mobile; this is the expected interaction pattern) | No action needed. The table-based layout with per-day sections and clear headers provides natural visual anchors. Users can jump to relevant days. |
| R6 | **SMTP failure crashes the workflow** -- Misconfigured credential or SMTP server is down | WORKOUT_CREATOR fails, user does not get their plan | Medium (SMTP servers can have intermittent issues) | `SendRoutineEmail` node is configured with `continueOnFail: true`. Additionally, the email branch is parallel to the main flow, so even a catastrophic failure in the email branch does not affect `NotifyRoutineCreated` or `Filtered Message2`. |
| R7 | **SQL query returns no rows** -- Race condition between `Create a row` write and `GetWeek1WithExercises` read | Email is sent with empty content or the Code node throws an error | Very Low (n8n parallel branches execute after the triggering node completes; Supabase writes are synchronous) | Add a guard in `GenerateRoutineHTML`: if `$('GetWeek1WithExercises').all().length === 0`, output a fallback message or skip email generation entirely. |

---

## 9. Files Modified

| File | Change Description | Role | Impact |
|------|--------------------|------|--------|
| `n8n/running_flows/WORKOUT_CREATOR.json` | Add 4 new nodes (`GetWeek1WithExercises`, `GenerateRoutineHTML`, `HasEmail`, `SendRoutineEmail`); update `Create a row` connection to include parallel branch to `GetWeek1WithExercises`; add connections between the 4 new nodes | n8n-agent | Workflow file is the only code artifact modified |
| n8n Credentials (UI only) | Add a new SMTP credential via the n8n admin interface | n8n-agent | No file change -- stored in n8n's internal credential store |

### Files NOT Modified

| File/System | Reason |
|-------------|--------|
| `workout-tracker-back/` (Go backend) | No backend changes needed -- all logic is in n8n |
| `workout-tracker/` (React frontend) | No frontend changes needed |
| Supabase database schema | No new tables or columns -- query uses existing `workouts` and `exercises` tables |
| `n8n/running_flows/MAIN_FLOW.json` | No changes -- MAIN_FLOW calls WORKOUT_CREATOR as a sub-workflow and receives the same return value |
| `e2e/test_data_setup.sql` | No changes -- existing test data is sufficient |
| `GymRatFlow_E2E_TestRunner.json` | No changes -- existing tests should pass without modification |

---

## 10. Success Criteria

All of the following must be true for the feature to be considered complete:

| # | Criterion | Validation Method |
|---|-----------|-------------------|
| SC1 | After WORKOUT_CREATOR runs successfully, the user receives an HTML email containing their Week 1 routine **within 60 seconds** of workflow completion. | Run WORKOUT_CREATOR with a test user; check inbox timestamp vs. workflow execution timestamp. |
| SC2 | The email contains: user profile summary (name, goal, level, days, priority muscles), a weekly overview table listing all training days, per-day exercise tables with columns for exercise number, name, main muscle, equipment, and video link, organized by role (compound -> core -> isolation), and warmup/progression/nutrition notes. | Visual inspection of received email against the reference format in `e2e/rutina_xiomara_semana1.md`. |
| SC3 | The email renders correctly (tables aligned, links clickable, text readable, no broken styles) in Gmail (web + mobile), Outlook (desktop + web), and Apple Mail. | Manual testing in each client. Screenshot documentation. |
| SC4 | The WhatsApp notification (`NotifyRoutineCreated`) continues working exactly as before -- same message content, same delivery timing, same behavior. | Run WORKOUT_CREATOR; verify WhatsApp message is received unchanged. Compare with pre-feature WhatsApp message. |
| SC5 | If the user has **no email** registered (`email` is null or empty), the workflow completes without errors. No email is sent, and the WhatsApp notification is still delivered. | Run WORKOUT_CREATOR with a test user that has `email = NULL`; verify clean execution log with no errors. |
| SC6 | If the SMTP server fails (wrong credentials, server down), the workflow completes without errors. The WhatsApp notification is still delivered, and `Filtered Message2` still returns `{ output: 'routine created' }`. | Temporarily misconfigure SMTP credential; run WORKOUT_CREATOR; verify workflow completes successfully. |
| SC7 | All existing E2E tests pass without changes. Specifically, `TC002_FULL_KYC` (which exercises the full WORKOUT_CREATOR flow) must pass. | Run `GymRatFlow_E2E_TestRunner.json`; verify all 12 test cases pass. |

---

## 11. Out of Scope (Future Enhancements)

The following are explicitly **not** part of this implementation. They are listed here to document potential future work and to prevent scope creep:

| Enhancement | Description | Why Deferred |
|-------------|-------------|--------------|
| **PDF attachment** | Generate a downloadable PDF version of the routine and attach it to the email | Requires a PDF generation library or external service (e.g., Puppeteer, wkhtmltopdf, or a PDF API). Adds complexity and cost. HTML email covers the primary use case. |
| **Weekly progress emails** | Send automated emails at the end of each week summarizing workout completions and progress metrics | Requires new workflow, new data aggregation queries, and user preference management. Separate feature. |
| **Email opt-in/opt-out preferences** | Allow users to control whether they receive emails, and what type of emails | Requires new database column (`email_preferences`), KYC flow update, and unsubscribe link handling. Not needed at current scale. |
| **Weeks 2-4 email delivery** | Send the routine for subsequent weeks as the user progresses through the mesocycle | Week 2-4 have different set/rep parameters (progressive overload). Delivering them upfront may confuse users. Consider sending at the start of each week instead. |
| **Email open/click tracking** | Track whether the user opened the email and which exercise video links they clicked | Requires a tracking pixel service and link redirect infrastructure. Privacy considerations. |
| **Internationalization (i18n)** | Support languages other than Spanish | GymBot is currently Spanish-only (Colombian audience). No immediate need. |
| **Email template editor** | Allow non-developers to customize the email template via a UI | Over-engineering for current stage. The Code node is sufficient. |
| **Resend/retry on SMTP failure** | Automatically retry email delivery if the first attempt fails | n8n does not have built-in retry for individual nodes in a parallel branch. Would require a separate error-handling sub-workflow. Low priority given `continueOnFail`. |

---

*This document is the main overview for the "Send Week 1 Routine via Email" feature. For detailed implementation instructions, refer to the role-specific specification documents listed in [Section 5](#5-team-roles--responsibilities).*
