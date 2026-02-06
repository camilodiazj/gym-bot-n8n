# Email Routine Week 1 - Implementation Spec

**Feature:** Send the user's Week 1 workout routine via HTML email after WORKOUT_CREATOR generates the plan.

**Workflow modified:** `n8n/running_flows/WORKOUT_CREATOR.json`

**Reference format:** `e2e/rutina_xiomara_semana1.md`

---

## Documents

| # | Document | Role | Description |
|---|----------|------|-------------|
| 0 | [00-OVERVIEW.md](00-OVERVIEW.md) | All | Executive summary, ADR, architecture, data flow, phases, risks, success criteria |
| 1 | [01-N8N-WORKFLOW-SPEC.md](01-N8N-WORKFLOW-SPEC.md) | n8n-agent | Full node JSON, connections, SMTP setup (Gmail/Resend/SES), error handling matrix, validation checklist, rollback plan |
| 2 | [02-HTML-TEMPLATE-SPEC.md](02-HTML-TEMPLATE-SPEC.md) | pixel-dev | Complete JS code for GenerateRoutineHTML, input/output contracts, translation maps, CSS palette, edge cases, testing guide |
| 3 | [03-QA-TEST-PLAN.md](03-QA-TEST-PLAN.md) | code-reviewer | 9 test cases (TC_EMAIL_001-009), email content checklist, regression checklist, performance expectations |
| 4 | [04-CONTENT-REVIEW-CHECKLIST.md](04-CONTENT-REVIEW-CHECKLIST.md) | kiro-coach | Exercise grouping, set/rep/RIR validation, warmup/progression/nutrition notes, motivational quote, HOME-specific content, sign-off form |

## Implementation Phases

| Phase | Owner | Summary |
|-------|-------|---------|
| 1. SMTP Setup | n8n-agent | Create SMTP credential in n8n, test delivery |
| 2. Workflow Nodes | n8n-agent | Add 4 nodes + connections to WORKOUT_CREATOR.json |
| 3. HTML Template | pixel-dev | Write GenerateRoutineHTML JS code |
| 4. Content Review | kiro-coach | Validate fitness content accuracy |
| 5. QA Validation | code-reviewer | Functional + rendering + regression tests |

## Architecture

```
Create a row ----+---> NotifyRoutineCreated ---> Filtered Message2 (return)   [unchanged]
                 |
                 +---> GetWeek1WithExercises ---> GenerateRoutineHTML ---> HasEmail ---> SendRoutineEmail
                       (Postgres)                 (Code - JS)             (IF)          (Email Send)
```

- Parallel branch: email failure never blocks WhatsApp notification or workflow return
- `continueOnFail: true` on SendRoutineEmail absorbs SMTP errors
- `HasEmail` guard skips gracefully when user has no email

## Key Decisions

- **n8n-only** (no Go backend changes, no Edge Functions) -- all data already in workflow context
- **HTML email** (not PDF) -- renders inline on mobile, no download required
- **Week 1 only** -- subsequent weeks delivered separately as user progresses

## Open Items for Review

| Item | Flagged In | Description |
|------|-----------|-------------|
| Progression order | 04-CONTENT-REVIEW-CHECKLIST.md, Section 5 | Week 3 deload vs Week 4 intensification may be inverted. Cross-reference `set_profiles` table. |
| Gender adaptation | 04-CONTENT-REVIEW-CHECKLIST.md, Section 7 | Motivational quote uses "perfecto" (masculine). Should adapt to "perfecta" for F users. |
| Spanish accents | 02-HTML-TEMPLATE-SPEC.md, Section 11 | Code omits tildes/accents for email client safety. UTF-8 is declared, so accents should work. Consider adding them. |
