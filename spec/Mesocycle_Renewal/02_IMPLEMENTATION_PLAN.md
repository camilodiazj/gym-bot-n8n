# 02 - Mesocycle Renewal: Implementation Plan

Step-by-step implementation plan for migrating the Mesocycle Renewal feature into GymBot's production n8n workflows.

## Project Context

GymBot is an AI fitness coaching platform. Production workflows live in `n8n/running_flows/`. The mesocycle renewal subflow exists but is NOT integrated into production flows.

### Key Files

| File | Purpose |
|------|---------|
| `n8n/running_flows/MAIN_FLOW.json` | Main WhatsApp orchestrator |
| `n8n/running_flows/WORKOUT_CREATOR.json` | Routine generator |
| `n8n/wip/GymBotMesocycleRenewal.json` | V2 renewal subflow (ready to deploy) |
| `n8n/wip/GymRatFlow_Supabase_V3.json` | WIP main flow with integration (reference only) |
| `e2e/test_data_setup.sql` | Test fixture data |
| `n8n/tests/GymRatFlow_E2E_TestRunner.json` | E2E test runner with 13 existing test cases |

### Architecture Summary

- **Path A (Auto)**: MAIN_FLOW detects W4 completion on FALSE branch of has_planned_workouts -> calls renewal subflow
- **Path B (Manual)**: User says "renovar mesociclo" -> Intention_Agent detects RENOVAR_MESOCICLO -> fetches plan info -> calls renewal subflow
- **Subflow**: Renewal conversation with 4 options (MANTENER, CAMBIAR_DIAS, ROTAR, PREGUNTAR)
- **CAMBIAR_DIAS**: Subflow calls WORKOUT_CREATOR with is_renewal=true

### Role Definitions

- **[pixel-dev]**: Frontend/Backend code, database migrations, Go/TypeScript
- **[n8n-agent]**: n8n workflow JSON modifications, node configuration, connection wiring
- **[code-reviewer]**: QA validation, Definition of Done checks

---

## Phase 1: Subflow Deployment

### T-101: Deploy GymBotMesocycleRenewal subflow to production

- **Assignee**: [n8n-agent]
- **Input**: `n8n/wip/GymBotMesocycleRenewal.json` (V2, 842 lines)
- **Technical Detail**:
  - Copy `n8n/wip/GymBotMesocycleRenewal.json` to `n8n/running_flows/GymBotMesocycleRenewal.json`
  - Verify the Execute Workflow Trigger node accepts these inputs: `user_id` (string), `full_name` (string), `whatsapp_id` (string), `phone_number_id` (string), `user_message` (string), `days_per_week` (number)
  - Verify the `Call_GymRatForm` node (CAMBIAR_DIAS path) references the correct WORKOUT_CREATOR workflow ID
  - Verify Postgres credentials reference matches production credential ID
  - Verify OpenAI/LLM model reference is valid
  - Verify WhatsApp node credentials and phone_number_id references are correct
- **Validation [code-reviewer]**:
  - Subflow trigger accepts all 6 required input parameters
  - All 4 intention paths (MANTENER, CAMBIAR, ROTAR, PREGUNTAR) have valid terminal nodes
  - No dangling connections or orphan nodes
  - System prompt is in Spanish
  - Memory cleanup nodes exist for all 3 action paths
  - WhatsApp send nodes have correct credential references

---

## Phase 2: WORKOUT_CREATOR Enhancement

### T-201: Add renewal input parameters to WORKOUT_CREATOR trigger

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/WORKOUT_CREATOR.json`, node `input` (executeWorkflowTrigger)
- **Technical Detail**:
  - Add to `workflowInputs.values` array:
    ```json
    { "name": "is_renewal", "type": "string" },
    { "name": "override_days_available", "type": "number" }
    ```
  - These are optional parameters - existing calls with only `whatsapp_id` must continue to work
- **Validation [code-reviewer]**:
  - Trigger node has 3 input parameters: whatsapp_id (number), is_renewal (string), override_days_available (number)
  - Existing workflow executions that only pass `whatsapp_id` do NOT break (is_renewal defaults to undefined/empty)

### T-202: Add If_Is_Renewal conditional node

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/WORKOUT_CREATOR.json`
- **Technical Detail**:
  - Insert a new If node named `If_Is_Renewal` after `ProcessUserPreferences`
  - Condition: `{{ $items('input')[0].json.is_renewal }}` equals `"true"` (string comparison)
  - TRUE branch: goes to new `Clear_Old_Workouts` node
  - FALSE branch: goes to existing `GetUser` node (normal onboarding flow continues unchanged)
  - Rewire: ProcessUserPreferences currently connects to its downstream node. Insert If_Is_Renewal between ProcessUserPreferences and its current downstream node
- **Validation [code-reviewer]**:
  - If_Is_Renewal node exists and is wired after ProcessUserPreferences
  - FALSE branch connects to existing flow (no regression)
  - TRUE branch connects to Clear_Old_Workouts
  - When is_renewal is undefined (normal flow), it takes the FALSE branch

### T-203: Add Clear_Old_Workouts and Clear_Old_Schedule nodes

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/WORKOUT_CREATOR.json`
- **Technical Detail**:
  - **Clear_Old_Workouts** (Postgres node):
    ```sql
    DELETE FROM workouts
    WHERE user_id = (
      SELECT user_id FROM users
      WHERE full_phone_number = '{{ $items('input')[0].json.whatsapp_id }}'
    );
    ```
  - **Clear_Old_Schedule** (Postgres node, after Clear_Old_Workouts):
    ```sql
    DELETE FROM user_weekly_schedule
    WHERE user_id = (
      SELECT user_id FROM users
      WHERE full_phone_number = '{{ $items('input')[0].json.whatsapp_id }}'
    );
    ```
  - After Clear_Old_Schedule, connect to existing `GetUser` node to continue normal routine generation
  - Use production Postgres credential ID
- **Validation [code-reviewer]**:
  - Both DELETE queries use parameterized user lookup (no raw user_id injection)
  - Nodes are chained sequentially: Clear_Old_Workouts -> Clear_Old_Schedule -> GetUser
  - Postgres credential matches production

### T-204: Modify ProcessUserPreferences to handle override_days_available

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/WORKOUT_CREATOR.json`, node `ProcessUserPreferences` (Code node)
- **Technical Detail**:
  - Append to the end of the JavaScript code, before the final return statement:
    ```javascript
    // Override days_available if renewal with changed days
    const overrideDays = $items('input')[0].json.override_days_available;
    if (overrideDays && overrideDays > 0) {
      // Also update week_schedule mapping
      const scheduleMap = { 2: 'fb_2', 3: 'fb_3', 4: 'ua_4', 5: 'ppl_5', 6: 'ppl_6' };
      items[0].json.days_available = overrideDays;
      items[0].json.week_schedule = scheduleMap[overrideDays] || items[0].json.week_schedule;
    }
    ```
  - This ensures the downstream routine generation uses the new day count
- **Validation [code-reviewer]**:
  - When override_days_available is null/undefined/0, the original days_available is preserved
  - When override_days_available = 3, days_available becomes 3 and week_schedule becomes 'fb_3'
  - All 5 valid mappings work (2-6)

### T-205: Add UpdatePlan node for renewal path

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/WORKOUT_CREATOR.json`
- **Technical Detail**:
  - Add conditional logic after workouts are created: if is_renewal=true, run UpdatePlan instead of (or in addition to) CreatePlan
  - **UpdatePlan** (Postgres node):
    ```sql
    UPDATE users_plans
    SET mesocycle_number = COALESCE(mesocycle_number, 1) + 1,
        last_renewal_date = NOW(),
        week_schedule = '{{ $json.week_schedule }}',
        start_date = NOW()
    WHERE user_id = '{{ $json.user_id }}'
      AND status = 'active';
    ```
  - Alternative approach: Modify the CreatePlan node to handle both insert (new user) and update (renewal) with a conditional
  - The key requirement is: for renewals, DO NOT create a new plan row - UPDATE the existing one
- **Validation [code-reviewer]**:
  - For normal flow (is_renewal != 'true'): CreatePlan INSERT works as before
  - For renewal flow: UPDATE runs, mesocycle_number increments by 1, last_renewal_date is set
  - No duplicate plan rows created during renewal

---

## Phase 3: MAIN_FLOW Integration

### T-301: Add RENOVAR_MESOCICLO to Intention_Agent

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/MAIN_FLOW.json`, node `Intention_Agent`
- **Technical Detail**:
  - Modify the system prompt to add RENOVAR_MESOCICLO as a valid intention
  - Add to the INTENCIONES VALIDAS section:
    ```
    - RENOVAR_MESOCICLO: El usuario menciona explicitamente querer cambiar su rutina, rotar ejercicios, renovar su mesociclo, o empezar un nuevo ciclo de entrenamiento.
      Ejemplos: "Quiero cambiar mi rutina", "Nuevos ejercicios", "Renovar mesociclo", "Cambiar dias", "Quiero rotar ejercicios", "Nuevo ciclo"
    ```
  - Update the return instruction to include RENOVAR_MESOCICLO in the list of valid outputs
  - RENOVAR_MESOCICLO must be checked BEFORE CHAT to avoid false classification
- **Validation [code-reviewer]**:
  - System prompt includes RENOVAR_MESOCICLO with clear examples
  - The output instruction lists all valid intents including RENOVAR_MESOCICLO
  - Message "Quiero cambiar mi rutina" should be classified as RENOVAR_MESOCICLO, not CHAT

### T-302: Add RENOVAR_MESOCICLO case to Switch node

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/MAIN_FLOW.json`, node `Switch`
- **Technical Detail**:
  - Add a 4th output rule to the Switch node's rules array:
    ```json
    {
      "conditions": {
        "conditions": [{
          "leftValue": "={{ $json.output.trim() }}",
          "rightValue": "RENOVAR_MESOCICLO",
          "operator": { "type": "string", "operation": "equals" }
        }],
        "combinator": "and"
      },
      "renameOutput": true,
      "outputKey": "RENOVAR_MESOCICLO"
    }
    ```
  - This creates output index 3 for RENOVAR_MESOCICLO
- **Validation [code-reviewer]**:
  - Switch node has 4 outputs: CONFIRMAR_RUTINA(0), CHAT(1), VER_RUTINA_DE_HOY(2), RENOVAR_MESOCICLO(3)
  - Output key matches exactly "RENOVAR_MESOCICLO"
  - Comparison uses `$json.output.trim()` for whitespace safety

### T-303: Add Fetch_Plan_For_Renewal node (Path B)

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/MAIN_FLOW.json`
- **Technical Detail**:
  - Add a Postgres query node named `Fetch_Plan_For_Renewal`
  - Connect from Switch output 3 (RENOVAR_MESOCICLO) -> Fetch_Plan_For_Renewal
  - SQL query:
    ```sql
    SELECT up.user_id, ws.days_per_week, up.mesocycle_number, up.week_schedule
    FROM users_plans up
    JOIN week_schedules ws ON up.week_schedule = ws.schedule_type
    WHERE up.user_id = '{{ $items('GetUser')[0].json.user_id }}'
      AND up.status = 'active'
    ORDER BY up.start_date DESC
    LIMIT 1;
    ```
  - Use production Postgres credential
  - Purpose: Fetches days_per_week which is NOT available on Path B (the Week_Schedule node only runs on the FALSE branch of has_planned_workouts)
- **Validation [code-reviewer]**:
  - Query returns days_per_week, mesocycle_number, and user_id
  - JOIN with week_schedules is correct
  - Filters by active status and limits to 1 result
  - Connected from Switch output 3

### T-304: Add Execute_Mesocycle_Renewal_Manual node (Path B)

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/MAIN_FLOW.json`
- **Technical Detail**:
  - Add an Execute Workflow node named `Execute_Mesocycle_Renewal_Manual`
  - Connect from Fetch_Plan_For_Renewal -> Execute_Mesocycle_Renewal_Manual
  - Workflow ID: reference to the deployed GymBotMesocycleRenewal workflow (from T-101)
  - Parameter mapping:
    ```
    user_id: {{ $items('GetUser')[0].json.user_id }}
    full_name: {{ $items('GetUser')[0].json.full_name }}
    whatsapp_id: {{ $items('If')[0].json.contacts[0].wa_id }}
    phone_number_id: {{ $items('If')[0].json.metadata.phone_number_id }}
    user_message: {{ $items('Normalize_Message')[0].json.message_body }}
    days_per_week: {{ $json.days_per_week }}
    ```
  - Note: `days_per_week` comes from Fetch_Plan_For_Renewal ($json), not from Week_Schedule
  - Set `alwaysOutputData: true`
- **Validation [code-reviewer]**:
  - All 6 parameters are mapped correctly
  - days_per_week uses $json (from Fetch_Plan_For_Renewal), NOT $items('Week_Schedule')
  - user_message uses Normalize_Message (not raw webhook body) for button/interactive message support
  - Workflow ID references the correct deployed subflow

### T-305: Add Check_Mesocycle_Complete node (Path A)

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/MAIN_FLOW.json`
- **Technical Detail**:
  - Add a Code node named `Check_Mesocycle_Complete`
  - Insert AFTER the `Merge` node (which merges Week_Schedule + User_Finished_Workouts + Template_Days)
  - **REWIRE**: Change Merge output from AI Agent1 to Check_Mesocycle_Complete
  - JavaScript code:
    ```javascript
    // Verificar si el mesociclo esta completo (semana 4)
    const daysPerWeek = $items('Week_Schedule')[0].json.days_per_week;
    const finishedWorkouts = $('User_Finished_Workouts').all();

    const week4Completed = finishedWorkouts.filter(
      w => w.json.week === 4 && w.json.Completed === true
    ).length;

    const mesocycleComplete = week4Completed >= daysPerWeek;

    return [{
      json: {
        mesocycle_complete: mesocycleComplete,
        week4_completed: week4Completed,
        days_per_week: daysPerWeek
      }
    }];
    ```
  - Set `executeOnce: true` to prevent duplicate processing
- **Validation [code-reviewer]**:
  - Code correctly references $items('Week_Schedule') and $('User_Finished_Workouts')
  - Comparison is >= (not ==) to handle edge cases
  - Returns a clean JSON object with mesocycle_complete boolean
  - Merge connection is rewired from AI Agent1 to this node

### T-306: Add If_Mesocycle_Complete node (Path A)

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/MAIN_FLOW.json`
- **Technical Detail**:
  - Add an If node named `If_Mesocycle_Complete`
  - Connect from Check_Mesocycle_Complete -> If_Mesocycle_Complete
  - Condition: `{{ $json.mesocycle_complete }}` is TRUE (boolean check)
  - TRUE output -> Execute_Mesocycle_Renewal (new node, T-307)
  - FALSE output -> AI Agent1 (existing node, normal scheduling)
  - Set `executeOnce: true`
- **Validation [code-reviewer]**:
  - TRUE branch routes to renewal subflow call
  - FALSE branch routes to AI Agent1 (preserving existing scheduling behavior)
  - When mesocycle_complete is false, the normal scheduling flow works exactly as before

### T-307: Add Execute_Mesocycle_Renewal node (Path A)

- **Assignee**: [n8n-agent]
- **Input**: `n8n/running_flows/MAIN_FLOW.json`
- **Technical Detail**:
  - Add an Execute Workflow node named `Execute_Mesocycle_Renewal`
  - Connect from If_Mesocycle_Complete TRUE -> Execute_Mesocycle_Renewal
  - Workflow ID: same as T-304 (deployed GymBotMesocycleRenewal)
  - Parameter mapping:
    ```
    user_id: {{ $items('GetUser')[0].json.user_id }}
    full_name: {{ $items('GetUser')[0].json.full_name }}
    whatsapp_id: {{ $items('If')[0].json.contacts[0].wa_id }}
    phone_number_id: {{ $items('If')[0].json.metadata.phone_number_id }}
    user_message: {{ $items('Normalize_Message')[0].json.message_body }}
    days_per_week: {{ $items('Week_Schedule')[0].json.days_per_week }}
    ```
  - Note: `days_per_week` comes from Week_Schedule (available on Path A, the FALSE branch)
  - Set `alwaysOutputData: true`
- **Validation [code-reviewer]**:
  - All 6 parameters mapped correctly
  - days_per_week uses $items('Week_Schedule') (available on Path A)
  - This is a DIFFERENT node from T-304 because the data sources differ between Path A and Path B

---

## Phase 4: E2E Test Infrastructure

### T-401: Add mesocycle test users to test_data_setup.sql

- **Assignee**: [pixel-dev]
- **Input**: `e2e/test_data_setup.sql`
- **Technical Detail**:
  - **TEARDOWN (Section 1)**: Add phones to DELETE cascade:
    ```sql
    '570000000051', '570000000052', '570000000053'
    ```
  - **USER FIXTURES (Section 2)**: Create 3 users:

    | Phone | UUID pattern | Name | Purpose |
    |-------|-------------|------|---------|
    | 570000000051 | e2e00051-0000-0000-0000-000000000051 | Test_MesoDetect | TC_MESO_001: Auto detection |
    | 570000000052 | e2e00052-0000-0000-0000-000000000052 | Test_MesoMantener | TC_MESO_002: MANTENER flow |
    | 570000000053 | e2e00053-0000-0000-0000-000000000053 | Test_MesoManual | TC_MESO_003: Manual intent |

  - **PLANS (Section 3)**: Create active plans for each user:
    - week_schedule: 'fb_3' (3 days/week) for users 051/052
    - week_schedule: 'fb_3' for user 053
    - mesocycle_number: 1
    - goal: 'Ganar masa muscular', level: 'Intermedio'
  - **WORKOUTS (Section 4)**: Create workout entries for users 051/052:
    - 4 weeks x 3 days = 12 workout day groups
    - Use real exercise_ids from the exercises table
    - Set exercise_order correctly (compound first, then isolation)
  - **SCHEDULES (Section 5)**: For users 051/052:
    - All weeks 1-4 sessions marked Completed = true
    - planned_day in the past
    - NO future planned sessions (has_planned_workouts must be FALSE)
  - **SCHEDULES for user 053**: Week 1 active schedule with future planned_day
  - **GYM PROFILES (Section 6)**: Create users_gym_profile entries for all 3 users
- **Validation [code-reviewer]**:
  - Teardown DELETE includes all 3 new phones
  - Users 051/052 have ALL week 4 sessions Completed = true
  - Users 051/052 have NO future planned_day entries
  - User 053 has active future schedule (has_planned_workouts = TRUE)
  - All FK references are valid (template_id, exercise_id, etc.)
  - SQL is idempotent (can run multiple times safely)

### T-402: Add TC_MESO_001 test case (Auto Detection)

- **Assignee**: [n8n-agent]
- **Input**: `n8n/tests/GymRatFlow_E2E_TestRunner.json`, node "Load Test Cases"
- **Technical Detail**:
  - Add to the test cases array:
    ```javascript
    {
      order: 20,
      id: "TC_MESO_001",
      name: "Deteccion automatica mesociclo completo",
      priority: "CRITICAL",
      category: "MESOCYCLE_RENEWAL",
      testType: "SINGLE",
      metrics: {
        rule: "output.includes('mesociclo') || output.includes('opciones') || output.includes('Felicidades') || output.includes('completaste')",
        description: "Bot detects W4 completion and offers renewal options"
      },
      input: [{
        messaging_product: "whatsapp",
        metadata: { display_phone_number: "573213413664", phone_number_id: "914510145083991" },
        contacts: [{ profile: { name: "Test MesoDetect" }, wa_id: "570000000051" }],
        messages: [{
          from: "570000000051",
          id: "wamid.E2E-TC_MESO_001",
          timestamp: String(Math.floor(Date.now() / 1000)),
          text: { body: "Hola, quiero agendar mi semana" },
          type: "text"
        }],
        field: "messages"
      }],
      cleanup: [
        "DELETE FROM n8n_chat_histories WHERE session_id LIKE '%e2e00051%';"
      ]
    }
    ```
  - The user sends a scheduling message, but since W4 is complete, the system should detect mesocycle completion and offer renewal options instead
- **Validation [code-reviewer]**:
  - Test case has correct phone (570000000051) matching the fixture user
  - Metric rule checks for multiple possible renewal-related keywords
  - Cleanup clears chat histories to prevent state leakage between tests
  - testType is SINGLE (one message, one response)

### T-403: Add TC_MESO_002 test case (MANTENER flow)

- **Assignee**: [n8n-agent]
- **Input**: `n8n/tests/GymRatFlow_E2E_TestRunner.json`, node "Load Test Cases"
- **Technical Detail**:
  - This test verifies the MANTENER_RUTINA path
  - Uses MULTI_TURN type with 2 turns:
    1. Turn 1: "Hola" -> expects renewal options
    2. Turn 2: "Quiero mantener mi rutina igual" -> expects confirmation
  - DB verification: Check mesocycle_number incremented
    ```javascript
    {
      order: 21,
      id: "TC_MESO_002",
      name: "MANTENER_RUTINA - mesociclo incrementado",
      priority: "HIGH",
      category: "MESOCYCLE_RENEWAL",
      testType: "MULTI_TURN",
      turns: [
        {
          message: "Hola, quiero organizar mi semana",
          expectContains: ["mesociclo", "opciones", "mantener"]
        },
        {
          message: "Quiero mantener mi rutina igual",
          expectContains: ["mesociclo", "nuevo", "agendar"]
        }
      ],
      input: [{
        messaging_product: "whatsapp",
        metadata: { display_phone_number: "573213413664", phone_number_id: "914510145083991" },
        contacts: [{ profile: { name: "Test MesoMantener" }, wa_id: "570000000052" }],
        messages: [{ from: "570000000052", id: "wamid.E2E-TC_MESO_002", timestamp: String(Math.floor(Date.now() / 1000)), text: { body: "" }, type: "text" }],
        field: "messages"
      }],
      cleanup: [
        "DELETE FROM n8n_chat_histories WHERE session_id LIKE '%e2e00052%';"
      ]
    }
    ```
- **Validation [code-reviewer]**:
  - Multi-turn structure with 2 turns
  - Turn 1 triggers renewal detection, turn 2 executes MANTENER
  - Phone matches fixture user 052

### T-404: Add TC_MESO_003 test case (Manual Intent)

- **Assignee**: [n8n-agent]
- **Input**: `n8n/tests/GymRatFlow_E2E_TestRunner.json`, node "Load Test Cases"
- **Technical Detail**:
  - This test verifies Path B (manual RENOVAR_MESOCICLO intent)
  - User 053 has ACTIVE planned workouts but explicitly requests renewal
    ```javascript
    {
      order: 22,
      id: "TC_MESO_003",
      name: "Intencion manual RENOVAR_MESOCICLO",
      priority: "HIGH",
      category: "MESOCYCLE_RENEWAL",
      testType: "SINGLE",
      metrics: {
        rule: "output.includes('mesociclo') || output.includes('opciones') || output.includes('rutina')",
        description: "Bot detects RENOVAR_MESOCICLO intent and offers renewal options"
      },
      input: [{
        messaging_product: "whatsapp",
        metadata: { display_phone_number: "573213413664", phone_number_id: "914510145083991" },
        contacts: [{ profile: { name: "Test MesoManual" }, wa_id: "570000000053" }],
        messages: [{
          from: "570000000053",
          id: "wamid.E2E-TC_MESO_003",
          timestamp: String(Math.floor(Date.now() / 1000)),
          text: { body: "Quiero cambiar mi rutina y hacer nuevos ejercicios" },
          type: "text"
        }],
        field: "messages"
      }],
      cleanup: [
        "DELETE FROM n8n_chat_histories WHERE session_id LIKE '%e2e00053%';"
      ]
    }
    ```
- **Validation [code-reviewer]**:
  - Phone 053 has active planned_workouts (TRUE branch)
  - Message clearly requests routine change (should trigger RENOVAR_MESOCICLO, not CHAT)
  - Metric rule checks for renewal-related keywords

---

## Phase 5: Documentation & Cleanup

### T-501: Update CLAUDE.md with mesocycle test users

- **Assignee**: [pixel-dev]
- **Input**: `CLAUDE.md`
- **Technical Detail**:
  - Add mesocycle test users to the "Test Users (Reserved Phones)" section:
    ```markdown
    **MESOCYCLE Users** (`5700000005XX`) - Pre-populated fixtures:
    | Phone | User | Purpose |
    |-------|------|---------|
    | `570000000051` | Test_MesoDetect | TC_MESO_001 |
    | `570000000052` | Test_MesoMantener | TC_MESO_002 |
    | `570000000053` | Test_MesoManual | TC_MESO_003 |
    ```
  - Add test cases to the "Test Runner" table
  - Add phones to the teardown comment block
  - Add GymBotMesocycleRenewal.json to the workflow table in running_flows
- **Validation [code-reviewer]**:
  - All 3 phones listed in reserved phones documentation
  - Test case table includes TC_MESO_001-003
  - Workflow table includes GymBotMesocycleRenewal.json

### T-502: Move WIP files or mark as archived

- **Assignee**: [pixel-dev]
- **Input**: `n8n/wip/` directory
- **Technical Detail**:
  - After production deployment, the WIP reference files should be archived:
    - `n8n/wip/GymBotMesocycleRenewal.json` -> mark with `_ARCHIVED` suffix or move to `n8n/archived/`
    - `n8n/wip/GymRatFlow_Supabase_V3.json` -> keep as reference (contains other WIP features)
    - `n8n/wip/TEST_USER_MESOCYCLE.md` -> keep as reference documentation
  - Do NOT delete WIP files until production is verified working
- **Validation [code-reviewer]**:
  - Production `running_flows/` contains GymBotMesocycleRenewal.json
  - WIP files are clearly marked as archived or moved
  - No duplicate active workflows

---

## Phase 6: Integration Testing & Verification

### T-601: Run full E2E test suite

- **Assignee**: [n8n-agent]
- **Input**: All modified workflows deployed to n8n, test fixtures loaded
- **Technical Detail**:
  - **Pre-requisite**: Run `e2e/test_data_setup.sql` to create/reset all test fixtures
  - **Step 1**: Execute GymRatFlow_E2E_TestRunner workflow in n8n
  - **Step 2**: Verify ALL existing tests still pass (TC001-TC012, TC_HOME_001-003)
  - **Step 3**: Verify new tests pass (TC_MESO_001-003)
  - **Step 4**: Check Generate Report node for pass rates
  - Expected: 16/16 tests pass (13 existing + 3 new)
- **Validation [code-reviewer]**:
  - Zero regressions on existing tests
  - TC_MESO_001 passes: auto detection works
  - TC_MESO_002 passes: MANTENER flow works end-to-end
  - TC_MESO_003 passes: manual intent detection works
  - Report shows 100% pass rate

### T-602: Manual DB verification after test run

- **Assignee**: [pixel-dev]
- **Input**: Supabase database access
- **Technical Detail**:
  - After TC_MESO_002 runs, verify:
    ```sql
    -- Mesocycle number should be incremented
    SELECT mesocycle_number, last_renewal_date
    FROM users_plans
    WHERE user_id = 'e2e00052-0000-0000-0000-000000000052';
    -- Expect: mesocycle_number = 2, last_renewal_date IS NOT NULL

    -- Schedule should be cleared
    SELECT COUNT(*) FROM user_weekly_schedule
    WHERE user_id = 'e2e00052-0000-0000-0000-000000000052';
    -- Expect: 0

    -- Workouts should still exist (MANTENER keeps them)
    SELECT COUNT(*) FROM workouts
    WHERE user_id = 'e2e00052-0000-0000-0000-000000000052';
    -- Expect: > 0
    ```
- **Validation [code-reviewer]**:
  - mesocycle_number incremented from 1 to 2
  - last_renewal_date is set and recent
  - user_weekly_schedule is empty (cleared for re-scheduling)
  - workouts table still has exercises (MANTENER preserves them)

---

## Dependency Graph

```
T-101 (Deploy Subflow)
  |
  v
T-201 --> T-202 --> T-203 --> T-204 --> T-205 (WORKOUT_CREATOR changes, sequential)
  |
  v
T-301 --> T-302 (Intention + Switch, can be parallel)
T-303 --> T-304 (Path B nodes, sequential)
T-305 --> T-306 --> T-307 (Path A nodes, sequential)
  |
  v
T-401 (Test data, can start after T-101)
T-402 --> T-403 --> T-404 (Test cases, can start after T-401)
  |
  v
T-501 --> T-502 (Documentation, parallel with tests)
  |
  v
T-601 --> T-602 (Integration testing, LAST)
```

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Workflow ID mismatch | HIGH | Blocks subflow calls | Document the deployed workflow ID in T-101 output |
| Intention_Agent misclassifies RENOVAR_MESOCICLO as CHAT | MEDIUM | Path B doesn't trigger | Add clear examples, test with edge cases |
| WORKOUT_CREATOR breaks for normal onboarding | HIGH | Critical regression | T-202 must default to FALSE branch when is_renewal is undefined |
| Week_Schedule not available on Path B | HIGH | Runtime error | T-303 Fetch_Plan_For_Renewal solves this |
| Subflow memory not cleaned | LOW | Stale conversation on next renewal | Verify cleanup nodes exist in subflow |
