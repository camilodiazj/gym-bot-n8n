# QA Test Plan: Email Routine Week 1

**Feature**: Send Week 1 routine via HTML email after WORKOUT_CREATOR generates workouts
**Role**: Code Reviewer / QA Analyst
**Version**: 1.0
**Date**: 2026-02-06

---

## 1. Test Scope

### In Scope

- **Feature under test**: After WORKOUT_CREATOR creates the 4-week workout plan, a parallel branch sends an HTML email containing the Week 1 routine to the user (if they have an email on file).
- **Components tested**:
  - `GetWeek1WithExercises` (Postgres node) -- Queries workouts for week 1 joined with exercise details
  - `GenerateRoutineHTML` (Code node) -- Builds the full HTML email body from workout data and user profile
  - `HasEmail` (IF node) -- Routes based on whether the user has an email address
  - `SendRoutineEmail` (Email Send node) -- Delivers the HTML email via SMTP
- **Integration points**: Parallel branch execution alongside the existing WhatsApp notification branch

### Out of Scope

- Existing WhatsApp flow -- must remain completely unchanged
- Workout generation logic (AI agent, exercise selection, set_profiles)
- Database schema changes (none expected)
- Frontend workout-tracker application
- Morning reminder and workout completion workflows

---

## 2. Prerequisites

| Prerequisite | Details |
|---|---|
| SMTP Credential | `GymBot Email SMTP` configured in n8n with valid SMTP host, port, username, password |
| Test email account | Accessible inbox to verify received emails (e.g., a Gmail account the QA tester controls) |
| WORKOUT_CREATOR deployed | Updated workflow with the new email branch imported into n8n |
| Test users available | Existing test fixtures from `e2e/test_data_setup.sql` executed against the Supabase database |
| n8n instance running | With Postgres (Supabase), OpenAI, and SMTP credentials configured |
| Email clients for rendering tests | Access to Gmail (web + mobile), Outlook (desktop), and Apple Mail |

---

## 3. Test Cases

### TC_EMAIL_001: Happy Path - GYM User

**Priority**: Critical
**Category**: Functional

| Field | Value |
|---|---|
| **Input** | Run WORKOUT_CREATOR with the `whatsapp_id` of a GYM user who has an email address in `users_gym_profile` |
| **Precondition** | User exists in `users_gym_profile` with a valid `email` field, `training_environment = 'GYM'` |

**Expected Results**:

1. WhatsApp notification is sent (existing behavior unchanged -- verify message content is identical to pre-feature behavior)
2. HTML email is received in the user's inbox within 60 seconds of workflow completion
3. Email subject line follows the format: `Tu Rutina Semana 1 - {full_name} | Kairos`
4. Email body contains:
   - Profile summary section (Objetivo, Nivel, Dias/semana, Ambiente)
   - Weekly overview table listing all training days and their session names
   - Per-day exercise tables grouped by role (compound, core, isolation)
   - All video links (`musclewiki.com`) are clickable and open correctly
5. Exercise ordering within each day is correct:
   - Compound exercises: `exercise_order` 1-4
   - Core exercises: `exercise_order` 5-6
   - Isolation exercises: `exercise_order` 7+

**Verification**: Open the received email in Gmail web, Gmail mobile (Android or iOS), and Outlook desktop. Confirm rendering is acceptable in all three.

---

### TC_EMAIL_002: Happy Path - HOME User

**Priority**: Critical
**Category**: Functional

| Field | Value |
|---|---|
| **Input** | Run WORKOUT_CREATOR with a HOME user (e.g., phone `570000000211` - Maria Lopez, equipment: mancuernas + bandas) |
| **Precondition** | User exists with `training_environment = 'HOME'`, `home_equipment` populated |

**Expected Results**:

1. Profile section displays `Ambiente: HOME`
2. Equipment list is shown in Spanish (e.g., "Mancuernas, Bandas elasticas")
3. Equipment column in exercise tables shows Spanish translations for each exercise's equipment
4. Only exercises compatible with the user's available equipment appear in the routine
5. All other email sections render correctly (same structure as GYM user)

---

### TC_EMAIL_003: User Without Email

**Priority**: Critical
**Category**: Negative / Boundary

| Field | Value |
|---|---|
| **Input** | Run WORKOUT_CREATOR with a user whose `email` field is `NULL` or an empty string in `users_gym_profile` |
| **Precondition** | User exists but has no email address |

**Expected Results**:

1. `HasEmail` node evaluates to the **False** branch
2. No email is sent (no SMTP connection attempt)
3. WhatsApp notification is still sent normally
4. Workflow completes and returns `"routine created"` output
5. No errors appear in the n8n execution log

---

### TC_EMAIL_004: SMTP Failure

**Priority**: High
**Category**: Error Handling / Resilience

| Field | Value |
|---|---|
| **Input** | Temporarily modify the `GymBot Email SMTP` credential with an incorrect password, then run WORKOUT_CREATOR with a user who has an email |
| **Precondition** | SMTP credential is intentionally broken |

**Expected Results**:

1. `SendRoutineEmail` node fails with an authentication error
2. The `continueOnFail` setting on the node handles the error gracefully
3. WhatsApp notification is still sent (parallel branch independence)
4. Workflow completes without a top-level error -- returns `"routine created"`
5. The SMTP error is visible in the n8n execution log (click on the `SendRoutineEmail` node to see the error details)
6. No user-facing error message is generated

**Teardown**: Restore the correct SMTP password after this test.

---

### TC_EMAIL_005: Health Restriction User

**Priority**: Medium
**Category**: Functional / Content Validation

| Field | Value |
|---|---|
| **Input** | Run WORKOUT_CREATOR with a user whose `health_status = 'C'` (upper body issues) |
| **Precondition** | User exists with health restriction C |

**Expected Results**:

1. Exercises in the generated routine already avoid overhead pressing (filtered by the AI agent during workout creation)
2. The HTML email displays correctly regardless of the exercise count per day
3. No broken HTML from days that have fewer exercises than typical
4. Day sections with fewer exercises do not show empty rows or placeholder text
5. All table structures remain intact

---

### TC_EMAIL_006: Large Routine (6 Days)

**Priority**: Medium
**Category**: Boundary / Rendering

| Field | Value |
|---|---|
| **Input** | Run WORKOUT_CREATOR with a user whose `days_available = 6` |
| **Precondition** | User has a 6-day training schedule configured |

**Expected Results**:

1. Email contains 6 distinct day sections, each with its own exercise table(s)
2. HTML renders without horizontal overflow on desktop email clients
3. On mobile email clients, scrolling is vertical and natural -- no pinch-to-zoom required to read tables
4. The weekly overview table correctly lists all 6 days
5. Total email size remains reasonable (under 100KB for the HTML body)

---

### TC_EMAIL_007: Existing E2E Tests Unaffected

**Priority**: Critical
**Category**: Regression

| Field | Value |
|---|---|
| **Input** | Run `GymRatFlow_E2E_TestRunner.json` (the full E2E test suite) |
| **Precondition** | All test fixtures from `e2e/test_data_setup.sql` are in place |

**Expected Results**:

1. All 12 existing test cases pass:
   - TC001 (FILTRO_RUIDO)
   - TC002 (ONBOARDING)
   - TC002_FULL_KYC (ONBOARDING_FULL)
   - TC003 (AGENDAR)
   - TC004 (DESCANSO)
   - TC006 (VER_RUTINA)
   - TC007 (CHAT)
   - TC011 (PENDING_TASK confirm)
   - TC012 (PENDING_TASK decline)
   - TC_HOME_001 (HOME basic)
   - TC_HOME_002 (HOME bodyweight)
   - TC_HOME_003 (HOME health restriction)
2. Specifically: TC002_FULL_KYC (full onboarding that triggers WORKOUT_CREATOR) must pass end-to-end
3. No test case shows increased execution time beyond normal variance

---

### TC_EMAIL_008: Email Rendering Across Clients

**Priority**: High
**Category**: Visual / Compatibility

| Field | Value |
|---|---|
| **Input** | Send a test email to at least 3 different email providers/clients |
| **Precondition** | A successful TC_EMAIL_001 execution has generated an email |

**Expected Rendering Checks**:

| Client | Checks |
|---|---|
| **Gmail (web)** | Tables are aligned, background colors render correctly, links show in blue/red as styled, no clipping |
| **Gmail (Android/iOS)** | Email is responsive and readable without horizontal scroll, text is legible without zoom |
| **Outlook (desktop)** | Tables render correctly (Outlook uses the Word rendering engine -- the most restrictive client). Background colors on table cells may vary; verify they degrade gracefully |
| **Apple Mail** | Clean rendering, proper font sizes, no layout breaks |

**Additional Checks**:

- No broken images (there should be no images -- text-only email)
- No missing sections or content gaps
- Spanish characters display correctly in all clients: a, e, i, o, u (with accents), n (with tilde), inverted question marks, inverted exclamation marks
- Email does not land in spam/junk folder (check SPF/DKIM if configured)

---

### TC_EMAIL_009: Parallel Branch Independence

**Priority**: High
**Category**: Architecture / Error Isolation

| Field | Value |
|---|---|
| **Input** | Temporarily add `throw new Error('TEST FAILURE')` at the top of the `GenerateRoutineHTML` Code node, then run WORKOUT_CREATOR |
| **Precondition** | Intentional error injected into the email branch |

**Expected Results**:

1. The email branch fails at the `GenerateRoutineHTML` node
2. The WhatsApp branch completes normally and sends the notification
3. The workflow returns `"routine created"` as its final output
4. In the n8n execution view:
   - Main branch nodes show green (success)
   - Email branch nodes show red (failure) at `GenerateRoutineHTML`
5. No partial email is sent (the failure occurs before `SendRoutineEmail`)

**Teardown**: Remove the injected error from `GenerateRoutineHTML` after this test.

---

## 4. Email Content Validation Checklist

For each email received during testing, verify every item below:

### Header and Profile

- [ ] Header shows "Plan de Entrenamiento - Semana 1"
- [ ] User's full name is displayed correctly (no truncation, proper capitalization)
- [ ] Profile section shows: Objetivo, Nivel, Dias/semana, Ambiente
- [ ] If HOME user: equipment list is shown in Spanish

### Quick Reference Guide

- [ ] Quick Reference Guide table is present with columns for Sets, Reps, RIR, Descanso
- [ ] RIR tip blockquote is present and text is readable

### Weekly Overview

- [ ] Weekly overview table has the correct number of days matching the user's schedule
- [ ] Day names and session titles match what is stored in `user_weekly_schedule`

### Day Sections

- [ ] Each day section has a title with an emoji icon
- [ ] Each day has exercise tables grouped by role: compound first, then core, then isolation
- [ ] Table headers for each role group show the correct sets/reps/RIR/rest values for that role

### Exercise Rows

- [ ] Each exercise row displays: number (#), Spanish name, muscle (in Spanish), equipment (in Spanish), video link
- [ ] Video links point to `musclewiki.com` URLs and are clickable
- [ ] Exercise numbering within each role group is sequential

### Notes Section

- [ ] Calentamiento (warmup) note is present
- [ ] Progresion (progression) note is present
- [ ] Hidratacion y Nutricion note is present

### Footer

- [ ] Motivational quote is present and displays correctly
- [ ] Footer shows "Generado por Kairos Personal Trainer"

### Technical Quality

- [ ] No broken HTML (unclosed tags, missing styles, raw code visible)
- [ ] Spanish characters render correctly: a, e, i, o, u (accented), n (tilde), inverted punctuation
- [ ] No empty sections or placeholder text visible
- [ ] Email total size is reasonable (under 100KB)

---

## 5. Regression Test Checklist

After deploying the email feature, verify that existing functionality is unaffected:

- [ ] WORKOUT_CREATOR workflow still creates workouts for all 4 weeks in the `workouts` table
- [ ] WhatsApp notification message content is unchanged (compare with a pre-feature screenshot/log)
- [ ] The `Filtered Message2` node still returns `"routine created"` as the workflow output
- [ ] No increase in total workflow execution time greater than 10 seconds compared to baseline
- [ ] `users` table -- no new columns, no modified data
- [ ] `users_plans` table -- no new columns, no modified data
- [ ] `workouts` table -- data is identical to what would be generated without the email feature
- [ ] `user_weekly_schedule` table -- schedule creation is unchanged
- [ ] Morning reminder workflow (`MorningReminder-WorkoutTracker.json`) still functions correctly
- [ ] Workout completion workflow (`GymBotMesocycleRenewal.json`) still functions correctly

---

## 6. Performance Expectations

| Component | Expected Duration | Notes |
|---|---|---|
| `GetWeek1WithExercises` | < 500ms | Simple indexed query joining `workouts` and `exercises` for `week = 1` |
| `GenerateRoutineHTML` | < 100ms | Pure string concatenation and template operations, no external calls |
| `HasEmail` (IF node) | < 10ms | Simple null/empty check |
| `SendRoutineEmail` | < 5s | SMTP delivery (depends on mail server response time) |
| **Total email branch overhead** | **< 6 seconds** | Sum of all email branch nodes |

**Key constraint**: Because the email branch runs in parallel with the WhatsApp branch, the email overhead should NOT noticeably delay the workflow's final completion time. The workflow completes when the main branch finishes -- the email branch runs independently.

If the email branch takes longer than 6 seconds, investigate:
1. SMTP server latency (try a different provider or check network)
2. HTML generation performance (check for accidental loops or large string allocations)
3. Database query performance (verify indexes on `workouts.user_id` and `workouts.week`)
