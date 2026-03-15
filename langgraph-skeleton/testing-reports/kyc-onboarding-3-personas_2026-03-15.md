# KAIROS KYC ONBOARDING — 3-PERSONA E2E TEST REPORT

**Date:** 2026-03-15
**Endpoint:** `POST http://localhost:8000/case5/kyc/chat`
**Graph:** Case 5 — Onboarding KYC (mock, in-memory checkpointer)

---

## 1. Test Overview

Three concurrent sub-agents simulated real users chatting with Kairos through the KYC onboarding flow. Each persona has a distinct profile covering different training goals, environments, experience levels, and health conditions.

| Persona | Age | Goal | Environment | Equipment | Health Condition |
|---------|-----|------|-------------|-----------|-----------------|
| Sofia | 22, F | Bajar grasa | HOME | Peso corporal | Ninguna |
| Andres | 30, M | Ganar masa muscular | GYM | N/A | Dolor en hombro derecho |
| Marta | 52, F | Salud general / recomposicion | HOME | Mancuernas + bandas | Hernia discal L4-L5 |

---

## 2. Results Summary

| Persona | Final Status | Turns | Health Code | Profile Saved | Routed to Trainer |
|---------|-------------|-------|-------------|---------------|-------------------|
| Sofia | PROFILE_SAVED | 8 | A | Yes | No |
| Andres | PROFILE_SAVED | 7 | C | Yes | No |
| Marta | PROFILE_SAVED | 7 | D | Yes | No |

---

## 3. Conversation Transcripts

### 3.1 Sofia — Principiante total

**Phone:** `573100000001` | **Thread:** `test_persona_573100000001`

| Turn | KYC Turn | User Message | Kairos Response | Collected Fields |
|------|----------|-------------|-----------------|------------------|
| 1 | 1 | "Hola! Quiero empezar a entrenar, nunca he hecho ejercicio" | Welcome greeting, asks for training goal (Pregunta 1 de 5) | [] |
| 2 | 2 | "Quiero bajar de grasa, me siento pesada ultimamente" | Confirms goal, asks experience/days/schedule (Pregunta 2 de 5) | [primary_goal] |
| 3 | 3 | "Nunca he entrenado, tengo 3 dias disponibles, prefiero en la manana" | Confirms 3 fields, asks GYM or HOME (Pregunta 3 de 5) | [primary_goal, training_experience, days_available, preferred_schedule] |
| 4 | 4 | "En casa, solo tengo mi peso corporal" | Confirms home+bodyweight, asks bio data (Pregunta 4 de 5) | [+training_environment, home_equipment] |
| 5 | 5 | "Soy mujer, tengo 22 anos, mido 160 cm y peso 68 kg" | Asks about health/injuries (Pregunta 5 de 5) | [+biological_sex, age, height_cm, weight_kg] |
| 6 | 5 | "No tengo ninguna lesion ni problema de salud" | Shows profile summary, asks for confirmation | [+health_status] (11 total) |
| 7 | 5 | "Si, todo esta correcto" | **Empty response** — `awaiting_confirmation` still `true` | 11 fields |
| 8 | 5 | "Confirmo" | "Perfil guardado exitosamente, Sofia!" | 11 fields, health_code=A, profile_saved=true |

**Observations:**
- Turn 7 returned an **empty `kairos_response`** and confirmation was not processed. Required a second message ("Confirmo") to finalize.
- All 11 fields (including `home_equipment`) were correctly collected.

---

### 3.2 Andres — Intermedio gym bro

**Phone:** `573100000002` | **Thread:** `test_persona_573100000002`

| Turn | KYC Turn | User Message | Kairos Response | Collected Fields |
|------|----------|-------------|-----------------|------------------|
| 1 | 1 | "Que mas, quiero empezar con ustedes" | Welcome, asks training goal (Pregunta 1 de 5) | [] |
| 2 | 2 | "Mi meta es ganar masa muscular, quiero ponerme grande" | Confirms goal, asks experience/days/schedule (Pregunta 2 de 5) | [primary_goal] |
| 3 | 3 | "Llevo como 3 anos entrenando, voy 5 dias a la semana, por la tarde" | Confirms 3 fields, asks GYM or HOME (Pregunta 3 de 5) | [+training_experience, days_available, preferred_schedule] |
| 4 | 4 | "Entreno en gimnasio" | Confirms gym, asks bio data (Pregunta 4 de 5) | [+training_environment] |
| 5 | 5 | "Hombre, 30 anos, 178 cm, 82 kilos" | Asks about health/injuries (Pregunta 5 de 5) | [+biological_sex, age, height_cm, weight_kg] |
| 6 | 5 | "Tengo un dolor en el hombro derecho a veces cuando hago press" | Shows profile summary, asks for confirmation | [+health_status] (10 total) |
| 7 | 5 | "Si, confirmo" | "Perfil guardado exitosamente, Andres!" | health_code=C, profile_saved=true |

**Observations:**
- Completed in optimal 7 turns.
- Health code **C** (upper body issues) correctly assigned for shoulder pain.
- GYM environment = no `home_equipment` field collected (correct behavior).

---

### 3.3 Marta — Adulta con restricciones

**Phone:** `573100000003` | **Thread:** `test_persona_573100000003`

| Turn | KYC Turn | User Message | Kairos Response | Collected Fields |
|------|----------|-------------|-----------------|------------------|
| 1 | 1 | "Buenas tardes, una amiga me recomendo esto" | Welcome, asks training goal (Pregunta 1 de 5) | [] |
| 2 | 2 | "Quiero mejorar mi salud en general, recomposicion corporal" | Confirms goal, asks experience/days/schedule (Pregunta 2 de 5) | [primary_goal] |
| 3 | 3 | "Entrenee hace unos anos como 6 meses, puedo 3 dias, en la noche mejor" | Confirms 3 fields, asks environment (Pregunta 3 de 5) | [+training_experience, days_available, preferred_schedule] |
| 4 | 4 | "Desde casa, tengo unas mancuernas y bandas elasticas" | Confirms home+equipment, asks bio data (Pregunta 4 de 5) | [+training_environment, home_equipment] |
| 5 | 5 | "Mujer, 52 anos, 162 cm, 71 kg" | Asks about health/injuries (Pregunta 5 de 5) | [+biological_sex, age, height_cm, weight_kg] |
| 6 | 5 | "Tengo hernia discal L4-L5 y a veces me duele mucho la espalda baja" | Shows profile summary, asks for confirmation | [+health_status] (11 total) |
| 7 | 5 | "Si, todo bien, confirmo" | "Perfil guardado exitosamente, Marta!" | health_code=D, profile_saved=true |

**Observations:**
- Completed in optimal 7 turns.
- Health code **D** (spine issues) correctly assigned for hernia discal L4-L5.
- HOME environment with equipment correctly captured.
- `route_to_trainer` remained `false` — hernia was classified as manageable (D, not E).

---

## 4. Quality Analysis

### 4.1 Passed Checks

| Check | Sofia | Andres | Marta |
|-------|-------|--------|-------|
| Profile saved | PASS | PASS | PASS |
| Health code accuracy | PASS (A) | PASS (C) | PASS (D) |
| All KYC fields collected | PASS (11) | PASS (10) | PASS (11) |
| Spanish language responses | PASS | PASS | PASS |
| Progress indicator shown | PASS | PASS | PASS |
| Turn efficiency (<=7) | **FAIL (8)** | PASS (7) | PASS (7) |

### 4.2 Issues Found

| ID | Severity | Persona | Description |
|----|----------|---------|-------------|
| BUG-001 | Medium | Sofia | Confirmation phrase "Si, todo esta correcto" was not recognized. The response was **empty** and `awaiting_confirmation` remained `true`. Required a second message "Confirmo" to proceed. Andres's "Si, confirmo" and Marta's "Si, todo bien, confirmo" both worked on the first try. |
| BUG-002 | Low | Sofia | When confirmation fails to parse, Kairos returns an **empty string** instead of re-prompting. Should respond with something like "No te entendi bien, confirmas que tu perfil esta correcto?" |

### 4.3 Health Classification Accuracy

| Persona | Reported Condition | Expected Code | Actual Code | Correct? |
|---------|-------------------|---------------|-------------|----------|
| Sofia | No injuries | A | A | Yes |
| Andres | Shoulder pain during press | C | C | Yes |
| Marta | Hernia discal L4-L5, lower back pain | D | D | Yes |

**Result: 3/3 correct (100%)**

---

## 5. Improvement Recommendations

### P0 — Fix Immediately

1. **Confirmation parsing robustness (BUG-001)**
   - The `check_status` node should recognize more natural confirmation phrases: "todo esta correcto", "esta bien", "si, correcto", "dale", "perfecto"
   - Suggested fix: Expand the keyword matching in `check_status` or use the LLM to classify the intent as confirmation/correction/other

2. **Empty response fallback (BUG-002)**
   - When the confirmation handler cannot determine the user's intent, return a re-prompt instead of an empty string
   - Suggested fix: Add a fallback in the `confirm_profile` or `check_status` node

### P1 — Improve Soon

3. **Turn 1 efficiency**
   - When a user states their goal in the first message (e.g., Sofia: "quiero empezar a entrenar"), Kairos still asks for the goal separately. Consider extracting the goal if it's already implied.

4. **Colombian Spanish consistency**
   - Responses are in Spanish but could lean more into Colombian expressions for warmth (e.g., "chevere", "parcero/a", "que nota")

### P2 — Future Tests

5. **Edge case personas to add:**
   - User with health code E (severe condition) to test `route_to_trainer` path
   - User who gives vague/incomplete answers to test recovery ("mido como 1 con 60")
   - User who requests a correction ("mi peso esta mal, son 72")
   - User who goes silent for 30+ min to test resumption/nudge flow
   - User who sends only emojis or audio notes to test special case handling

6. **Concurrency stress test**
   - Run 10+ simultaneous agents to verify thread isolation under load

---

## 6. Conclusion

The KYC onboarding flow is **functional and accurate**. All 3 personas completed the flow successfully with correct health classification. The main issue is a confirmation parsing gap that caused one persona to need an extra turn. Once BUG-001 and BUG-002 are fixed, the flow should reliably complete in 7 turns for all user types.
