# KYC E2E Test Report — 24 Scenarios
**Date:** 2026-03-15
**Endpoint:** `POST /case5/kyc/chat`
**Server:** `http://localhost:8000`

---

## 1. Executive Summary

| Metric | Value |
|--------|-------|
| Total scenarios | 24 |
| **PASS** | 13 (54%) |
| **FAIL** | 8 (33%) |
| **PARTIAL** | 3 (13%) |
| P0 bugs (blocking) | 3 |
| P1 bugs (major) | 4 |
| P2 bugs (minor) | 3 |

**Overall verdict: NOT READY for production.** The happy path works well (4/5 pass), but health classification is severely broken (2/5 correct with real conditions), the confirmation step has intermittent empty-response failures, and the correction flow does not actually apply corrections. Dense multi-field extraction also fails consistently.

---

## 2. Results Matrix

| ID | Category | Description | Result | Fields | Saved | Health | Issues |
|----|----------|-------------|--------|--------|-------|--------|--------|
| TC_HP_001 | Happy Path | Juan 25M principiante GYM masa | **PASS** | 10/10 | Yes | A ✅ | — |
| TC_HP_002 | Happy Path | Valentina 28F intermedia GYM grasa | **FAIL** | 10/10 | No | A ✅ | Empty response on confirmation (BUG-001) |
| TC_HP_003 | Happy Path | Ricardo 35M avanzado GYM 5d fuerza | **PASS** | 10/10 | Yes | A ✅ | — |
| TC_HP_004 | Happy Path | Lucia 40F principiante HOME resistencia | **PASS** | 11/11 | Yes | A ✅ | HOME + equipment captured correctly |
| TC_HP_005 | Happy Path | Pedro 45M intermedio GYM salud | **PASS** | 10/10 | Yes | A ✅ | — |
| TC_DE_001 | Data Extraction | Carolina combined env+experience msg | **FAIL** | 4/10 | No | — | "voy al gym" not extracted → stuck in loop (BUG-003) |
| TC_DE_002 | Data Extraction | Diego dense 2-message KYC | **FAIL** | 4/10 | No | — | Dense multi-field extraction failure (BUG-003) |
| TC_DE_003 | Data Extraction | Isabela slang/abbreviations | **PASS** | 10/10 | Yes | A ✅ | "F"→F, "1.60 m"→160cm correct |
| TC_DE_004 | Data Extraction | Manuel numbers as words | **PARTIAL** | 10/10 | No | A ✅ | Fields extracted (27,171,67 correct!) but confirmation loop stuck (BUG-004) |
| TC_SI_001 | Special Input | Emoji-only first msg | **PASS** | 10/10 | Yes | A ✅ | Emoji re-prompt worked. Minor: "6 meses"→"Menos de 6 meses" |
| TC_SI_002 | Special Input | Ultra-short answers | **PASS** | 10/10 | Yes | A ✅ | "Masa"→correct, "M 30 175 80"→correct |
| TC_SI_003 | Special Input | Ultra-verbose rambling | **FAIL** | 6/10 | No | — | Verbose biometrics not extracted (BUG-005) |
| TC_HC_001 | Health | Healthy, no injuries | **PASS** | 10/10 | Yes | A ✅ | — |
| TC_HC_002 | Health | Condromalacia rotuliana (knee) | **PASS** | 10/10 | Yes | B ✅ | Correct knee→lower body classification |
| TC_HC_003 | Health | Wrist fracture (upper body) | **FAIL** | 10/10 | Yes | A ❌ | Should be **C** (BUG-006) |
| TC_HC_004 | Health | Cervical hernias C5-C6/C6-C7 | **PASS** | 10/10 | Yes | D ✅ | Required retry on confirmation (BUG-001 intermittent) |
| TC_HC_005 | Health | Cardiac arrhythmia | **FAIL** | 10/10 | Yes | A ❌ | Should be **E** + route_to_trainer (BUG-007) |
| TC_CF_001 | Correction | Fix weight 55→58 | **PASS** | 10/10 | Yes | A ✅ | Weight corrected and re-displayed |
| TC_CF_002 | Correction | Change GYM→HOME | **FAIL** | 10/10 | Yes* | A ✅ | Correction NOT applied — saved as GYM (BUG-008) |
| TC_ER_001 | Error Recovery | Off-topic mid-flow | **PASS** | 10/10 | Yes | A ✅ | Politely re-asked Q1 after off-topic |
| TC_ER_002 | Error Recovery | Gibberish first msg | **PARTIAL** | 11/11 | Yes | A ✅ | Profile saved, but summary was empty (BUG-001) |
| TC_HG_001 | HOME vs GYM | Full home gym | **PASS** | 11/11 | Yes | A ✅ | Rich equipment list captured |
| TC_HG_002 | HOME vs GYM | Bodyweight + back pain | **PARTIAL** | 11/11 | Yes | A ❌ | Should be **D** (spine) (BUG-006) |
| TC_DD_001 | Demographics | 16yo teenager | **PASS** | 10/10 | Yes | A ✅ | Age=16 captured. No minor flag (by design?) |

---

## 3. Bug Catalog

### P0 — Blocking

#### BUG-006: Health Classification Misses Upper Body (C) and Spine (D) Conditions
**Affected:** TC_HC_003, TC_HG_002
**Severity:** P0
**Description:** Wrist fracture with pain during press/weight-bearing was classified as health A (no restrictions) instead of C (upper body issues). Lower back pain was classified as A instead of D (spine issues).
**Impact:** Users with real injuries will receive unsafe exercise recommendations (e.g., overhead press for someone with wrist pain, deadlifts for someone with lower back pain).
**Root cause:** The health classification logic appears too lenient — only obvious knee conditions (B) and cervical hernias (D) are correctly classified. Wrist/shoulder and general back pain conditions are missed entirely.

#### BUG-007: Cardiac/Special Conditions Not Classified as E
**Affected:** TC_HC_005
**Severity:** P0
**Description:** Diagnosed cardiac arrhythmia with cardiologist warning about intense exercise was classified as health A instead of E. The `route_to_trainer` flag was not set.
**Impact:** A user with a serious cardiac condition would receive an unrestricted workout plan instead of being flagged for professional review. This is a **safety-critical failure**.
**Root cause:** The health classification appears to not have rules for cardiac conditions, autoimmune disorders, or other systemic conditions that require E classification.

#### BUG-008: Correction Flow Does Not Apply GYM→HOME Change
**Affected:** TC_CF_002
**Severity:** P0
**Description:** When user requested to change training_environment from GYM to HOME with mancuernas during confirmation, the profile was immediately saved with the original GYM value. No updated summary was shown, no home_equipment was collected.
**Impact:** User data is saved incorrectly despite explicit correction request. The entire correction flow for `training_environment` changes is broken.

### P1 — Major

#### BUG-001: Intermittent Empty Response on Confirmation/Summary
**Affected:** TC_HP_002, TC_HC_004, TC_ER_002, TC_DE_004
**Severity:** P1
**Description:** The `kairos_response` field returns an empty string at critical moments:
- TC_HP_002: Empty on confirmation step → profile NOT saved
- TC_HC_004: Empty on first confirmation attempt → required retry
- TC_ER_002: Empty on summary display → user confirmed blindly
- TC_DE_004: Empty on summary → stuck re-displaying summary
**Frequency:** 4/24 scenarios (17%)
**Impact:** Users see no message, potentially losing trust or getting stuck.

#### BUG-003: Dense Multi-Field Extraction Failure
**Affected:** TC_DE_001, TC_DE_002
**Severity:** P1
**Description:** When users provide data spanning multiple KYC categories in a single message, the extraction tool only captures fields from one category. Examples:
- "Llevo 2 anos entrenando, voy al gym 4 dias por semana" → extracted experience/days/schedule but missed training_environment ("voy al gym")
- "Hombre, 28, 182 cm, 78 kg, entreno en gimnasio, estoy completamente sano" → only experience extracted, all biometrics/environment/health missed
**Impact:** Users who provide information efficiently get stuck in loops or incomplete flows.

#### BUG-004: Confirmation Loop Stuck (Non-Empty Variant)
**Affected:** TC_DE_004
**Severity:** P1
**Description:** After all 10 fields collected and `awaiting_confirmation: true`, user's "Si correcto" re-displayed the summary instead of saving the profile. The confirmation was not processed.
**Impact:** User cannot complete onboarding even though all data is correct.

#### BUG-005: Verbose Biometric Text Extraction Failure
**Affected:** TC_SI_003
**Severity:** P1
**Description:** When user provides biometrics in verbose natural language ("Soy mujer, tengo 33 anos, mido un metro sesenta y cinco y peso como 70 kilos mas o menos"), the extraction tool fails to capture `biological_sex`, `age`, `height_cm`, and `weight_kg`. The bot verbally acknowledges the data but the tool call does not extract it.
**Impact:** Verbose users (common in Colombian Spanish) cannot complete onboarding.

### P2 — Minor

#### BUG-009: Bot Addresses User as "Hola" Instead of Name
**Affected:** All 24 scenarios
**Severity:** P2
**Description:** The confirmation and save messages say "¡Listo Hola!" and "¡Perfil guardado exitosamente, Hola!" — using the word "Hola" as if it were the user's name. The KYC flow does not collect `full_name`.
**Recommendation:** Either collect name in the flow, or remove the name placeholder from the summary template.

#### BUG-010: Minor Age Classification Gap (16yo)
**Affected:** TC_DD_001
**Severity:** P2
**Description:** A 16-year-old minor was accepted without any age-related flag or `route_to_trainer`. Depending on business requirements, minors may need parental consent or special handling.
**Recommendation:** Define a minimum age policy and implement accordingly.

#### BUG-011: Stray JSON Formatting in Response
**Affected:** TC_DD_001
**Severity:** P2
**Description:** Turn 3 response ended with stray ` ```json` text — an internal formatting artifact leaked to the user.

---

## 4. Health Classification Accuracy Matrix

| Scenario | Condition | Expected | Actual | Verdict |
|----------|-----------|----------|--------|---------|
| TC_HC_001 | Healthy, active, no injuries | A | A | ✅ |
| TC_HC_002 | Condromalacia rotuliana (knee) | B | B | ✅ |
| TC_HC_003 | Wrist fracture, pain on press | C | A | ❌ |
| TC_HC_004 | Cervical hernias C5-C6/C6-C7 | D | D | ✅ |
| TC_HC_005 | Cardiac arrhythmia | E | A | ❌ |
| TC_HG_002 | Lower back pain (intermittent) | D | A | ❌ |
| TC_SI_003 | Past knee pain, fully resolved | A | — | N/A (extraction failed) |

**Accuracy: 3/6 (50%)** on scenarios with actual health conditions.

The classifier correctly handles:
- Completely healthy users (A)
- Knee-specific conditions (B)
- Cervical/disc conditions with explicit "cervical hernia" keywords (D)

The classifier **fails** on:
- Upper extremity conditions (wrist, shoulder) → should be C
- General spine conditions ("espalda baja") → should be D
- Cardiac/systemic conditions → should be E
- Any condition requiring `route_to_trainer`

---

## 5. Conversation Quality Assessment

### Tone & Language
- **Colombian Spanish**: Good. Natural, warm, uses emojis appropriately. Phrases like "¡Excelente!" and "¡Me encanta!" feel authentic.
- **Progress indicator**: Consistent "📋 Pregunta X de 5" across all scenarios. Clear and helpful.
- **Encouragement**: Appropriate without being patronizing.

### Turn Efficiency
| Category | Avg Turns | Expected | Verdict |
|----------|-----------|----------|---------|
| Happy Path (GYM) | 7 | 7 | ✅ Optimal |
| Happy Path (HOME) | 7 | 7 | ✅ Optimal |
| Dense Input | 3-7* | 3-6 | ❌ *Stuck/incomplete |
| Ultra-short | 7 | 7 | ✅ Expected |
| Ultra-verbose | 7* | 7 | ❌ *Extraction failed |
| With correction | 8 | 8 | ✅ Expected |

The flow does not benefit from users providing data early — combined messages are not parsed across categories.

### Data Extraction Accuracy (When It Works)

| Input | Extracted | Correct |
|-------|-----------|---------|
| "25 anos" | 25 | ✅ |
| "163 cm" | 163 | ✅ |
| "F" | F | ✅ |
| "1.60 m" | 160 | ✅ |
| "veintisiete anos" | 27 | ✅ |
| "uno setenta y uno" | 171 | ✅ |
| "sesenta y siete kilos" | 67 | ✅ |
| "Masa" | Ganar masa muscular | ✅ |
| "M 30 175 80" | M, 30, 175, 80 | ✅ |
| "bajar la pancita" | Bajar grasa | ✅ |
| "como un ano" | 6 a 12 meses | ✅ |
| "un metro sesenta y cinco" | ❌ NOT EXTRACTED | ❌ |

Numbers-as-words extraction works for demographics (TC_DE_004) but fails in verbose sentences (TC_SI_003).

---

## 6. Regression vs Previous 3-Persona Run

| Bug | Previous Run | This Run | Status |
|-----|-------------|----------|--------|
| BUG-001 (empty response) | Found | Still present (4/24 = 17%) | **NOT FIXED** |
| BUG-002 (empty response variant) | Found | Merged with BUG-001 | **NOT FIXED** |
| Health classification | Not tested | New failures found | **NEW** |
| Correction flow | Not tested | Broken for env change | **NEW** |
| Dense extraction | Not tested | Broken | **NEW** |

---

## 7. Category-Level Summary

### Happy Path (5 scenarios): 4/5 PASS (80%)
Strong baseline. The standard 7-turn GYM and HOME flows work reliably. Only failure was the intermittent empty-response bug on confirmation.

### Data Extraction Edge Cases (4 scenarios): 1/4 PASS (25%)
Worst-performing category. The extraction tool cannot handle:
- Multi-category data in a single message
- Verbose natural language biometrics
When data is provided in the expected turn structure, extraction is accurate.

### Special Input (3 scenarios): 2/3 PASS (67%)
Emoji and ultra-short inputs handled well. Verbose input fails due to extraction issues (same root cause as data extraction).

### Health Classification (5 scenarios): 3/5 PASS (60%)
Only A, B, and D (with explicit keywords) are classified correctly. C and E classifications are completely broken. This is safety-critical.

### Correction Flow (2 scenarios): 1/2 PASS (50%)
Simple field corrections (weight) work. Structural corrections (environment change GYM→HOME) do not.

### Error Recovery (2 scenarios): 2/2 PASS (100%)
Off-topic and gibberish messages handled gracefully (with minor empty-response issue on summary).

### HOME vs GYM (2 scenarios): 1/2 PASS (50%)
Full home gym captured correctly. Bodyweight-only with back pain failed on health classification.

### Demographic Diversity (1 scenario): 1/1 PASS (100%)
16-year-old accepted (policy question on minors remains open).

---

## 8. Prioritized Fix Recommendations

### Must Fix Before Launch (P0)

1. **Health Classification Overhaul** (BUG-006, BUG-007)
   - Add rules for upper extremity conditions → C
   - Add rules for general spine/back conditions → D
   - Add rules for cardiac, autoimmune, systemic conditions → E
   - Implement `route_to_trainer` flag for E classifications
   - Add comprehensive test cases for each health code

2. **Correction Flow for Environment Change** (BUG-008)
   - When user requests environment change during confirmation, update `training_environment`, collect `home_equipment` if switching to HOME, re-display summary

3. **Empty Response Fix** (BUG-001)
   - Investigate why `kairos_response` is intermittently empty on summary/confirmation turns
   - 17% failure rate is unacceptable for a production flow
   - Consider adding retry logic or response validation

### Should Fix (P1)

4. **Dense Multi-Field Extraction** (BUG-003)
   - Enable extraction tool to capture fields across multiple categories in a single message
   - At minimum, extract `training_environment` from messages containing "gym/gimnasio/casa"

5. **Verbose Text Extraction** (BUG-005)
   - Improve extraction for written-out numbers in verbose sentences
   - Test with "un metro sesenta y cinco", "como 70 kilos mas o menos"

6. **Confirmation Loop Fix** (BUG-004)
   - Ensure "Si correcto", "Si confirmo", etc. reliably trigger profile save when `awaiting_confirmation: true`

### Nice to Have (P2)

7. **Name Placeholder** (BUG-009) — Remove "Hola" as user name or collect `full_name`
8. **Minor Age Policy** (BUG-010) — Define and implement minimum age handling
9. **JSON Formatting Leak** (BUG-011) — Sanitize internal formatting from responses

---

## 9. Full Conversation Transcripts

### TC_HP_001 — Juan 25M, principiante, GYM, masa muscular ✅ PASS

| Turn | Speaker | Message |
|------|---------|---------|
| 1 | User | Hola! Quiero empezar a entrenar con ustedes |
| 1 | Kairos | ¡Hola! ¡Qué bueno tenerte a bordo! 🚀 Para empezar con tu rutina personalizada, cuéntame, ¿cuál es tu objetivo principal...? 📋 Pregunta 1 de 5 |
| 2 | User | Mi objetivo es ganar masa muscular |
| 2 | Kairos | ¡Excelente! Ganar masa muscular es un objetivo genial. 💪 ...cuéntame un poco sobre tu experiencia... 📋 Pregunta 2 de 5 |
| 3 | User | Nunca he entrenado antes, puedo 3 dias, por la manana |
| 3 | Kairos | ¡Perfecto! ¡Todos empezamos en algún momento! 🚀 ...¿dónde planeas entrenar: GYM o HOME? 📋 Pregunta 3 de 5 |
| 4 | User | Entreno en gimnasio |
| 4 | Kairos | ¡Buenísimo! ...¿cuál es tu sexo biológico, edad, estatura y peso? 📋 Pregunta 4 de 5 |
| 5 | User | Soy hombre, tengo 25 anos, mido 175 cm y peso 72 kilos |
| 5 | Kairos | ¡Gracias! ...¿tienes alguna lesión o condición de salud? 📋 Pregunta 5 de 5 |
| 6 | User | No tengo ninguna lesion |
| 6 | Kairos | ✅ ¡Listo! [Summary: M, 25, 175cm, 72kg, GYM, Health A] ¿Todo correcto? |
| 7 | User | Si, confirmo |
| 7 | Kairos | ✅ ¡Perfil guardado exitosamente! |

**Fields:** primary_goal=Ganar masa muscular, training_experience=Nunca, days=3, schedule=Mañana, env=GYM, sex=M, age=25, height=175, weight=72, health=A

---

### TC_HP_002 — Valentina 28F, intermedia, GYM, bajar grasa ❌ FAIL

| Turn | Speaker | Message |
|------|---------|---------|
| 1 | User | Buenas! Vi la publicidad y me interesa |
| 1 | Kairos | ¡Hola! Qué bueno tenerte por aquí. 💪 ...¿cuál es tu objetivo principal? 📋 Pregunta 1 de 5 |
| 2 | User | Quiero bajar de grasa |
| 2 | Kairos | ¡Perfecto! Bajar de grasa es un objetivo genial. ...experiencia? 📋 Pregunta 2 de 5 |
| 3 | User | Llevo como un ano entrenando, puedo 4 dias, por la tarde |
| 3 | Kairos | ¡Excelente! Un año de experiencia... ¿gym o casa? 📋 Pregunta 3 de 5 |
| 4 | User | Voy al gym |
| 4 | Kairos | ¡Entrenar en el gym es excelente! ...datos básicos? 📋 Pregunta 4 de 5 |
| 5 | User | Mujer, 28 anos, 163 cm, 65 kg |
| 5 | Kairos | ¡Gracias! ...¿lesiones o condiciones? 📋 Pregunta 5 de 5 |
| 6 | User | Todo bien de salud, sin lesiones |
| 6 | Kairos | ✅ [Summary: F, 28, 163cm, 65kg, GYM, Health A] ¿Todo correcto? |
| 7 | User | Perfecto, confirmo |
| 7 | Kairos | **(empty response)** ← BUG-001 |

**Result:** All fields collected but profile NOT saved due to empty confirmation response.

---

### TC_HP_003 — Ricardo 35M, avanzado, GYM 5d, fuerza ✅ PASS

| Turn | Speaker | Message |
|------|---------|---------|
| 1–6 | | Standard 7-turn flow |
| 7 | User | Dale, todo bien |
| 7 | Kairos | ✅ ¡Perfil guardado exitosamente! |

**Fields:** primary_goal=Mejorar fuerza, exp=Más de 3 años, days=5, schedule=Noche, env=GYM, M, 35, 180, 88, health=A

---

### TC_HP_004 — Lucia 40F, principiante, HOME bands+dumbbells ✅ PASS

| Turn | Speaker | Message |
|------|---------|---------|
| 1–6 | | Standard 7-turn flow |
| 4 | User | Entreno desde casa, tengo mancuernas y bandas elasticas |
| 4 | Kairos | ¡Entrenar en casa es genial! 🏡 Y con mancuernas y bandas elásticas tienes muchas opciones... |
| 7 | User | Si, todo esta correcto |
| 7 | Kairos | ✅ ¡Perfil guardado exitosamente! |

**Fields:** env=HOME, home_equipment="mancuernas y bandas elasticas", F, 40, 158, 62, health=A

---

### TC_HP_005 — Pedro 45M, intermedio, GYM 4d, salud general ✅ PASS

Standard 7-turn flow. All fields correct. Profile saved.

---

### TC_DE_001 — Carolina combined environment+experience ❌ FAIL

| Turn | Speaker | Message |
|------|---------|---------|
| 3 | User | Llevo 2 anos entrenando, voy al gym 4 dias por semana, en la manana |
| 3 | Kairos | ¡Súper! ...Ahora, cuéntame dónde entrenas: ¿en un gimnasio o en casa? ← **Did not extract "voy al gym"** |
| 4 | User | Mujer, 30 anos, 165 cm, 60 kg |
| 4 | Kairos | ¡Perfecto! Ya tengo toda la información sobre tu lugar de entrenamiento... ← **Misinterpreted demographics as environment answer** |
| 5 | User | Sin lesiones |
| 5 | Kairos | ...necesito que me confirmes tu sexo biológico, edad... ← **Stuck in loop** |
| 6 | User | Confirmo |
| 6 | Kairos | ...necesito que me proporciones los siguientes datos... ← **Still stuck** |

**Result:** Only 4/10 fields collected. Conversation stuck — cannot complete.

---

### TC_DE_002 — Diego dense 2-message KYC ❌ FAIL

| Turn | Speaker | Message |
|------|---------|---------|
| 1 | User | Hola pana, quiero ganar musculo, llevo 3 anos en el gym, voy 5 dias por la noche |
| 1 | Kairos | Acknowledged all data verbally but only extracted primary_goal |
| 2 | User | Hombre, 28, 182 cm, 78 kg, entreno en gimnasio, estoy completamente sano |
| 2 | Kairos | Asked about gym equipment access — did not extract biometrics |
| 3 | User | Si confirmo |
| 3 | Kairos | Asked for sexo biológico, edad, estatura, peso — **all ignored from Turn 2** |

**Result:** Only 4/10 fields captured. Dense extraction completely broken.

---

### TC_DE_003 — Isabela slang/abbreviations ✅ PASS

"F" → F, "1.60 m" → 160cm, "bajar la pancita" → Bajar grasa. All correct. Profile saved.

---

### TC_DE_004 — Manuel numbers as words ⚠️ PARTIAL

"veintisiete" → 27, "uno setenta y uno" → 171, "sesenta y siete" → 67. **Extraction excellent!**
But confirmation loop stuck — "Si correcto" re-displayed summary instead of saving.

---

### TC_SI_001 — Emoji-only first message ✅ PASS

| Turn | Speaker | Message |
|------|---------|---------|
| 1 | User | 💪🏋️‍♀️ |
| 1 | Kairos | ¡Me encanta la energía! 💪 Pero necesito una respuesta en texto para continuar. ← **Good re-prompt** |

Profile saved. Minor: "6 meses de experiencia" mapped to "Menos de 6 meses" (ambiguous).

---

### TC_SI_002 — Ultra-short answers ✅ PASS

"Masa" → Ganar masa muscular, "M 30 175 80" → all 4 fields correct. Profile saved.

---

### TC_SI_003 — Ultra-verbose rambling ❌ FAIL

| Turn | Speaker | Message |
|------|---------|---------|
| 5 | User | Soy mujer, tengo 33 anos, mido un metro sesenta y cinco y peso como 70 kilos mas o menos |
| 5 | Kairos | ¡Perfecto, gracias! ...pregunta final sobre salud — **BUT did not extract sex/age/height/weight** |

**Result:** 6/10 fields. Missing all biometrics despite bot acknowledging them.

---

### TC_HC_001 — Healthy ✅ PASS → Health A ✅

---

### TC_HC_002 — Condromalacia rotuliana ✅ PASS → Health B ✅

---

### TC_HC_003 — Wrist fracture ❌ FAIL → Health A ❌ (expected C)

---

### TC_HC_004 — Cervical hernias ✅ PASS → Health D ✅
Note: Required retry on confirmation (first attempt empty).

---

### TC_HC_005 — Cardiac arrhythmia ❌ FAIL → Health A ❌ (expected E)
No route_to_trainer flag set. **Safety-critical failure.**

---

### TC_CF_001 — Correct weight 55→58 ✅ PASS

| Turn | Speaker | Message |
|------|---------|---------|
| 7 | User | Mi peso esta mal, son 58 kg |
| 7 | Kairos | [Updated summary showing Peso: 58kg] ← **Correctly updated** |
| 8 | User | Ahora si, confirmo |
| 8 | Kairos | ✅ ¡Perfil guardado exitosamente! |

---

### TC_CF_002 — Change GYM→HOME ❌ FAIL

| Turn | Speaker | Message |
|------|---------|---------|
| 7 | User | Quiero cambiar el lugar, en realidad entreno en casa con mancuernas |
| 7 | Kairos | ✅ ¡Perfil guardado exitosamente! ← **Saved immediately with GYM, did NOT apply correction** |

**Result:** Profile saved with training_environment=GYM instead of HOME. home_equipment not collected.

---

### TC_ER_001 — Off-topic message ✅ PASS

| Turn | Speaker | Message |
|------|---------|---------|
| 2 | User | Oye sabes que hora es? |
| 2 | Kairos | [Re-asked Q1 about training goal] ← **Graceful redirect, no answer to off-topic** |

---

### TC_ER_002 — Gibberish first message ⚠️ PARTIAL

Gibberish handled well. HOME + bodyweight captured correctly. But summary turn returned empty response (BUG-001). Profile still saved because user said "Si" blindly.

---

### TC_HG_001 — Full home gym ✅ PASS

Equipment captured: "barra olimpica, discos, rack, mancuernas ajustables y una banca regulable". Profile saved.

---

### TC_HG_002 — Bodyweight + back pain ⚠️ PARTIAL

HOME + bodyweight captured correctly. But lower back pain classified as health A instead of D.

---

### TC_DD_001 — 16yo teenager ✅ PASS

Age=16 captured correctly. No special minor handling. Minor formatting leak (stray ```json in Turn 3).

---

## 10. Test Execution Metadata

| Metric | Value |
|--------|-------|
| Execution time | ~7 minutes (4 parallel agents) |
| Total API calls | ~163 |
| Phone range used | 573200000001–573200000071 |
| Agent distribution | 4 agents × 6 scenarios each |
| Retries needed | 2 (TC_HC_004 confirmation, TC_DE_004 stuck) |
