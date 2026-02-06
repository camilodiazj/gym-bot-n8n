# Content Review Checklist: Email Routine Week 1

**Feature**: HTML email containing the user's Week 1 training routine
**Role**: Kiro-Coach (Fitness Content Reviewer)
**Version**: 1.0
**Date**: 2026-02-06

---

## 1. Purpose

This document is for the **kiro-coach** role. The objective is to review the email template content for accuracy from a **fitness and coaching perspective**. This is NOT a technical or code review -- it is a content quality review.

The kiro-coach must verify that every piece of training advice, exercise grouping, loading parameter, and motivational text in the email is:

- **Accurate** according to exercise science principles
- **Safe** for all user levels (beginner through advanced)
- **Appropriate** for the target audience (Spanish-speaking, Colombian market)
- **Clear** and free of ambiguity that could lead to injury or confusion

---

## 2. Exercise Grouping Review

The HTML email template groups exercises by their `role` classification within each training day. The display order is:

1. **Compound exercises first** (role = `compound`)
2. **Core exercises second** (role = `core`)
3. **Isolation exercises last** (role = `isolation`)

### Why This Order Matters

- **Compound exercises** are multi-joint movements (squats, bench press, deadlifts, rows, overhead press, hip thrusts). They recruit the most muscle mass, require the highest neural drive, and benefit from being performed when the athlete is freshest. Placing them first maximizes force production and technique quality.
- **Core exercises** (anti-rotation, planks, dead bugs, pallof press) appear mid-workout. Training core before isolation work ensures trunk stability is addressed while the athlete still has focus, but after the heavy compound lifts that rely on core bracing.
- **Isolation exercises** are single-joint movements (bicep curls, tricep extensions, lateral raises, leg extensions, flyes). These are accessories that finish the session. They require less neural demand and can be performed effectively even when fatigued.

### Validation Questions

- [ ] Is the compound, then core, then isolation order correct for all standard training methodologies (hypertrophy, strength, general fitness)?
- [ ] Are there edge cases where isolation should come before compound? (Expected answer: No, except in pre-exhaustion techniques, which are advanced programming strategies not used in Week 1 of a standard mesocycle)
- [ ] Is the `role` classification in the `exercises` database table correct for the most common exercises? Specific checks:
  - [ ] Hip Thrust -- classified as `compound`? (Should be YES -- it is a multi-joint hip extension movement)
  - [ ] Plank variations -- classified as `core`? (Should be YES)
  - [ ] Cable Flyes -- classified as `isolation`? (Should be YES)
  - [ ] Romanian Deadlift -- classified as `compound`? (Should be YES -- hip hinge with knee and hip joint involvement)
  - [ ] Face Pulls -- classified as `isolation`? (Acceptable, though some classify as compound. Either is defensible)

---

## 3. Set/Rep/RIR Display Review

For each role group within a training day, the email displays a header row showing the loading parameters (sets, reps, RIR, rest) that apply to all exercises in that group. These values come from the `set_profiles` table.

### Expected Ranges by Role

| Parameter | Compound | Core | Isolation |
|---|---|---|---|
| **Sets** | 3-4 | 2-3 | 2-3 |
| **Reps** | 6-12 | 10-15 | 10-15 |
| **RIR** | 2-3 | 2-3 | 2-3 |
| **Rest (seconds)** | 90-180 | 60 | 60 |

### Validation Questions

- [ ] Do the displayed values for compound exercises fall within the 3-4 sets, 6-12 reps range? (For Week 1 of a mesocycle, moderate volume is expected)
- [ ] Do core exercises show appropriate volume (2-3 sets, 10-15 reps)?
- [ ] Do isolation exercises show appropriate volume (2-3 sets, 10-15 reps)?
- [ ] Is the RIR of 2-3 appropriate for Week 1? (Expected answer: Yes -- Week 1 should not be taken to failure. RIR 2-3 means "stop 2-3 reps before failure," which allows technique learning and gradual adaptation)
- [ ] Are rest periods appropriate? Compound lifts need 90-180 seconds for phosphocreatine recovery. Isolation and core can use shorter rest (60 seconds) because they create less systemic fatigue.
- [ ] Do the displayed values match what a certified personal trainer (ACE, NSCA-CSCS, or equivalent) would prescribe for a general population client?

---

## 4. Warmup Notes Review

The email includes the following warmup recommendation:

> "Movilidad articular general, 5 minutos de cardio ligero (caminar, saltar suave), 1 serie ligera del primer ejercicio"

Translation: General joint mobility, 5 minutes of light cardio (walking, light jumping), 1 light set of the first exercise.

### Validation Questions

- [ ] Is this warmup protocol appropriate for ALL fitness levels (beginner, intermediate, advanced)?
  - Beginners: Need more guidance -- is "movilidad articular general" specific enough? Consider whether joint circles, arm swings, and leg swings should be mentioned.
  - Advanced: May need more specific warmup (e.g., activation drills, band work). However, for a general recommendation, this is acceptable.
- [ ] Should warmup recommendations differ by training day? For example:
  - Leg day: Additional hip and ankle mobility
  - Upper body push day: Shoulder circles, band pull-aparts
  - Upper body pull day: Thoracic spine mobility
  - (Note: Day-specific warmups would add complexity. The current general recommendation is a reasonable baseline.)
- [ ] Any critical warmup elements missing?
  - [ ] Should dynamic stretching be explicitly mentioned? (The "movilidad articular" covers this implicitly)
  - [ ] Should foam rolling be mentioned? (Optional, not necessary for a general recommendation)
  - [ ] Is the "1 serie ligera del primer ejercicio" sufficient? (Some trainers recommend 2-3 progressive warmup sets for heavy compound lifts. For Week 1, 1 light set is acceptable since loads are moderate.)

---

## 5. Progression Notes Review

The email includes the following progression guidance:

> "Semana 1-2: Enfocate en aprender la tecnica correcta. Semana 3: Semana de descarga. Semana 4: Intensificacion"

Translation: Week 1-2: Focus on learning correct technique. Week 3: Deload week. Week 4: Intensification.

### Validation Questions

- [ ] Is this mesocycle structure accurate?
  - Traditional mesocycle progression is: accumulation (weeks 1-2), intensification (week 3), deload (week 4). The email shows deload at week 3 and intensification at week 4, which is an **inverted order** compared to the most common periodization models.
  - **Action required**: Verify against the `set_profiles` table. The actual loading progression in the database is what the user will follow -- these notes are educational context only. If `set_profiles` shows volume increasing through week 3 and dropping in week 4, then the note should be corrected to match.
  - Alternative valid structure: Some coaches place the deload mid-cycle (week 3) to allow supercompensation in week 4. If this is intentional, it should be explicitly justified.
- [ ] Should the text adapt based on user level?
  - **Beginner**: "Focus on technique" is excellent for weeks 1-2. Beginners genuinely need this emphasis.
  - **Intermediate**: May not need the technique reminder as much; could instead focus on "establishing your working weights."
  - **Advanced**: Should focus on "building volume base" rather than technique.
  - (Note: Adapting this text per level adds complexity. For v1, a universal message focused on technique is safe for all levels.)
- [ ] Is the week 3 "descarga" (deload) note accurate? Or should it say week 4?
  - [ ] Cross-reference with `set_profiles` table data to confirm which week has reduced volume/intensity.
  - [ ] If there is a mismatch, flag for correction.

---

## 6. Nutrition Notes Review

The email includes the following nutrition guidance:

> "Bebe agua antes, durante y despues del entrenamiento. Come algo ligero 1-2 horas antes. Proteina despues del entrenamiento."

Translation: Drink water before, during, and after training. Eat something light 1-2 hours before. Protein after training.

### Validation Questions

- [ ] Are these recommendations safe and universally applicable?
  - Hydration: Yes, universally safe. No contraindications.
  - Pre-workout meal: "Something light 1-2 hours before" is safe and general. Does not specify quantities or macros, which is appropriate for a general recommendation.
  - Post-workout protein: This is a well-established recommendation supported by current sports nutrition science. Safe for all populations.
- [ ] Anything too specific or potentially harmful?
  - [ ] The current text does NOT mention specific macro quantities, which is good. Avoid prescribing grams of protein or caloric targets in a general email.
  - [ ] No mention of supplements, which is appropriate. GymBot should not recommend supplements without context.
  - [ ] No mention of fasting protocols, which is appropriate. Avoid controversy.
- [ ] Should recommendations differ by goal?
  - **Ganar masa muscular** (muscle gain): Could emphasize caloric surplus and protein intake timing, but this level of specificity may be inappropriate for a general email.
  - **Bajar grasa** (fat loss): Could mention caloric deficit, but dietary advice should be handled carefully and ideally by a nutritionist.
  - (Recommendation: Keep the current general advice for v1. Goal-specific nutrition guidance could be a future enhancement with proper disclaimers.)
- [ ] Should there be a disclaimer that this is general guidance and not a substitute for professional nutritional advice?

---

## 7. Motivational Quote Review

Reference quote (from a generated plan):

> "No se trata de ser perfecta, se trata de ser constante. Cada repeticion cuenta, cada dia que te presentas es una victoria."

Translation: It's not about being perfect, it's about being consistent. Every rep counts, every day you show up is a victory.

### Validation Questions

- [ ] Should the quote be gender-adapted?
  - The word "perfecta" is feminine. For male users, this should be "perfecto."
  - **Check**: Does the `GenerateRoutineHTML` code node or the AI agent adapt the gender based on `biological_sex`? If not, this should be flagged for implementation.
- [ ] Should different quotes be used for different goals?
  - Muscle gain: Quotes about strength and growth
  - Fat loss: Quotes about discipline and transformation
  - General health: Quotes about consistency and well-being
  - (For v1, a single universal quote about consistency is acceptable. Goal-specific quotes could be a future enhancement.)
- [ ] Is the tone appropriate for the Colombian audience?
  - [ ] Does the language feel natural in Colombian Spanish? (Avoid Castilian Spanish or Mexican-specific slang)
  - [ ] Is the tone motivating without being condescending or excessively "coach-bro"?
  - [ ] Does the use of "tu" (informal you) match the brand's communication style? (GymBot/Kairos uses "tu" throughout WhatsApp interactions, so this is consistent)

---

## 8. HOME-Specific Content

For users with `training_environment = 'HOME'`, the email displays their available equipment and adapts the routine accordingly.

### Equipment Terminology

- [ ] Is the equipment terminology correct in Spanish?
  - "Mancuernas" for dumbbells -- Correct
  - "Banda elastica" / "Bandas elasticas" for resistance bands -- Correct
  - "Peso corporal" for bodyweight -- Correct
  - "Barra" for barbell -- Correct (if applicable for HOME users)
  - "Kettlebell" / "Pesa rusa" -- either is acceptable in Colombian Spanish
- [ ] Are there any equipment terms that might confuse a HOME user?

### Additional HOME User Considerations

- [ ] Should HOME users get additional notes about exercise form without mirrors or spotters?
  - Many HOME users do not have mirrors to check form. A note like "Si no tienes espejo, grabate con el celular para revisar tu tecnica" (If you don't have a mirror, record yourself with your phone to check technique) could be valuable.
  - Without a spotter, heavy compound lifts should be approached cautiously. A note about not training to failure on exercises like squats or overhead press when alone could improve safety.
- [ ] Should there be a note about creating a safe workout space at home?
  - Ensure enough clearance for overhead movements
  - Use a non-slip surface or yoga mat
  - Keep the area free of obstacles
  - (This could be a brief one-line note rather than a full section)
- [ ] Are the exercises shown appropriate for a home environment?
  - No exercises requiring cable machines, Smith machines, or gym-specific equipment should appear for HOME users
  - Exercises should be feasible with the user's declared equipment

---

## 9. Sign-off

After reviewing all items in sections 2 through 8, the kiro-coach should complete the following:

### 9.1 Summary of Findings

| Section | Status | Notes |
|---|---|---|
| Exercise Grouping (Section 2) | Approved / Flagged | |
| Set/Rep/RIR Display (Section 3) | Approved / Flagged | |
| Warmup Notes (Section 4) | Approved / Flagged | |
| Progression Notes (Section 5) | Approved / Flagged | |
| Nutrition Notes (Section 6) | Approved / Flagged | |
| Motivational Quote (Section 7) | Approved / Flagged | |
| HOME-Specific Content (Section 8) | Approved / Flagged | |

### 9.2 Required Text Changes

List any text modifications needed for the warmup, progression, or nutrition notes:

1. _[Text change 1]_
2. _[Text change 2]_
3. _[Text change 3]_

### 9.3 Motivational Quote Decision

- [ ] Approve current quote as-is
- [ ] Approve with gender adaptation (perfecto/perfecta based on `biological_sex`)
- [ ] Replace with alternative quote: _[insert quote]_

### 9.4 Exercise Grouping Logic Confirmation

- [ ] I confirm that the compound, then core, then isolation ordering is correct for all standard training methodologies presented in this email
- [ ] I have reviewed the `role` classifications for common exercises and they are accurate

### 9.5 Final Approval

- [ ] **APPROVED**: Content is accurate, safe, and appropriate for the target audience
- [ ] **APPROVED WITH CHANGES**: Content is acceptable pending the text changes listed in section 9.2
- [ ] **NOT APPROVED**: Content has issues that must be resolved before launch (detail in notes above)

**Reviewer Name**: ___________________________
**Date**: ___________________________
**Signature**: ___________________________
