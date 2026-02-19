# 01_IMPLEMENTATION_PHASES — Daily Report Workflow (KAN-97)

## Path: Logic-Driven (n8n-agent only)

No UI, no backend, no DB migrations. Single n8n workflow file.

## Phase 1: Build Workflow JSON

**Agent:** n8n-agent
**Output:** `n8n/running_flows/DailyReport.json`
**Estimated nodes:** 11

### Tasks (sequential):

1. **Scaffold workflow** — Schedule trigger + 7 Postgres nodes + Code node + 2 send nodes. All connections wired per connection map in `00_ARCHITECTURE.md`.

2. **Implement SQL queries** — Paste each of the 7 SQL queries from `00_ARCHITECTURE.md` into their respective Postgres nodes. Each node:
   - Credential: `vZLJtIWG5nYXMez4` ("Supabase Memory")
   - Operation: `executeQuery`
   - `executeOnce: true`
   - `alwaysOutputData: true`

3. **Implement Code node** (`aggregate_and_format`) — Single "Run Once for All Items" Code node that:
   - Reads all 7 query results via `$('query_name').all()` / `.first()`
   - Computes summary metrics with defensive defaults
   - Generates inline-CSS HTML email (follow pattern from `spec/email-routine-week1/02-HTML-TEMPLATE-SPEC.md`)
   - Generates WhatsApp text summary
   - Returns `{ html, whatsappMessage, subject, reportDate }`

4. **Configure send nodes**:
   - `send_email_report`: emailSend v1, `subject` = `{{ $json.subject }}`, `html` = `{{ $json.html }}`, `continueOnFail: true`
   - `send_whatsapp_summary`: whatsApp v1.1, `textBody` = `{{ $json.whatsappMessage }}`, credential `xIjy4zDHyjIvGQT4`, `continueOnFail: true`

### Definition of Done:
- [ ] Valid JSON importable into n8n
- [ ] All 11 nodes present with correct types and versions
- [ ] All connections match the connection map
- [ ] SQL queries match `00_ARCHITECTURE.md` exactly

## Phase 2: Validate & Test

**Agent:** n8n-agent (manual execution in n8n)

### Tasks:

1. **Import workflow** into n8n instance
2. **Manual trigger** — Execute workflow, verify each Postgres node returns expected columns
3. **Verify HTML output** — Copy `aggregate_and_format` output HTML, open in browser, check:
   - All sections render correctly
   - Tables are populated (or show "no issues" messages)
   - Color coding works (completion rate thresholds)
   - Alert section only appears when violations exist
4. **Verify WhatsApp output** — Check `whatsappMessage` text format matches spec
5. **Test empty state** — Ensure workflow completes without errors when all queries return zero rows
6. **Configure SMTP** — Set email credential in n8n UI, test email delivery
7. **Set recipient phone** — Configure Kairos Soporte WhatsApp number

### Definition of Done:
- [ ] Workflow executes end-to-end without errors
- [ ] Email received with correct HTML formatting
- [ ] WhatsApp message received with correct summary
- [ ] Empty state produces clean "no issues" report (not errors)

## Phase 3: Activate

1. Enable schedule trigger (6 AM America/Bogota)
2. Monitor first 3 days of automated execution
3. Verify report arrives daily at ~6:01 AM

## Reference Files

| File | Purpose |
|------|---------|
| `spec/daily-report-workflow/00_ARCHITECTURE.md` | SQL queries, node definitions, connection map |
| `spec/email-routine-week1/02-HTML-TEMPLATE-SPEC.md` | HTML inline CSS pattern |
| `spec/email-routine-week1/01-N8N-WORKFLOW-SPEC.md` | Postgres node JSON structure reference |
| `n8n/running_flows/WeeklySchedulingPrompt.json` | Schedule trigger + sequential Postgres chain pattern |
| `spec/workout_creator_quality_fixes/00_ARCHITECTURE.md` | Health restriction SQL patterns |
