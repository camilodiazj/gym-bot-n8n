# Feature Specification: Onboarding Inteligente y Perfilamiento Adaptativo

**Feature Branch**: `001-onboarding-kyc`
**Created**: 2026-03-15
**Status**: Draft
**Input**: User description: "F-01: Onboarding Inteligente y Perfilamiento Adaptativo — KYC conversacional via WhatsApp con memoria, abandono/retoma, y filtro de salud"

## Clarifications

### Session 2026-03-15

- Q: KYC question grouping (10 fields into how many turns?) → A: 5 turns — (1) goal, (2) experience + days + schedule, (3) environment + equipment, (4) sex + age + height + weight, (5) health status
- Q: Are additional profile fields (secondary_goal, priority_muscles, disliked_exercises, cardio) part of KYC? → A: Core only — collect 10 fields during KYC; preference fields collected post-routine via feedback
- Q: What happens if user rejects the profile summary? → A: Targeted correction — Kairos asks "¿Qué dato quieres corregir?", updates only that field, re-shows summary
- Q: How long does partial KYC state persist before expiring? → A: 7-day expiration — resume within 7 days; restart KYC if longer

## User Scenarios & Testing *(mandatory)*

### User Story 1 - First Contact and KYC Completion (Priority: P1)

A new user messages Kairos on WhatsApp for the first time. Kairos detects there is no existing profile, greets the user by their WhatsApp display name, and begins the KYC questionnaire conversationally — asking one question at a time. The user answers each question naturally (not selecting from a menu). When all required data points are collected, Kairos confirms the profile summary and transitions to routine generation.

**Why this priority**: This is the foundational user journey. No other feature works without a completed user profile. Every user's first interaction with Kairos starts here.

**Independent Test**: Can be fully tested by sending a series of WhatsApp messages from an unregistered number and verifying that a complete user profile is created in the database with all required fields populated.

**Acceptance Scenarios**:

1. **Given** a user with phone number not in the database, **When** they send "Kairos, quiero mi rutina" via WhatsApp, **Then** Kairos responds with a personalized welcome using their WhatsApp display name and asks the first KYC question.

2. **Given** a user is in the onboarding flow and has answered their training goal, **When** Kairos asks the next question (experience level), **Then** the question adapts to the previously stated goal (e.g., if goal is "bajar grasa", follow-up language reflects fat loss context).

3. **Given** a user provides two data points in a single message (e.g., "3 dias, soy intermedio"), **When** Kairos processes the message, **Then** both values are registered and Kairos skips to the next unanswered question.

4. **Given** a user has answered all required KYC fields, **When** Kairos detects all data is collected, **Then** Kairos presents a profile summary for confirmation and triggers routine generation.

5. **Given** a user is in the onboarding flow, **When** they see a Kairos response, **Then** the message includes a progress indicator (e.g., "Pregunta 2 de 5").

6. **Given** Kairos presents the profile summary and the user says "No, mi objetivo está mal", **When** Kairos processes the rejection, **Then** Kairos asks which field to correct, updates only that field, and re-presents the updated summary.

---

### User Story 2 - KYC Abandonment and Resumption (Priority: P2)

A user starts the KYC but stops responding mid-flow (e.g., after question 3 of 5). After 30 minutes of inactivity, Kairos sends a friendly nudge. When the user returns — whether minutes, hours, or days later — the conversation resumes from where they left off without repeating previously answered questions.

**Why this priority**: Drop-off during onboarding is the #1 churn risk. Saving partial state and enabling seamless resumption directly impacts conversion rates.

**Independent Test**: Can be tested by starting a KYC conversation, answering 3 questions, waiting 30+ minutes, and verifying that (a) a nudge message is sent, and (b) when the user responds, Kairos continues from question 4.

**Acceptance Scenarios**:

1. **Given** a user has completed 3 of 5 KYC questions and stops responding, **When** 30 minutes of inactivity elapse, **Then** Kairos sends a friendly push message (e.g., "Hey, quedamos a mitad. Seguimos cuando quieras?") without being pushy or guilt-inducing.

2. **Given** a user abandoned KYC 2 hours ago with partial data saved, **When** they send a new message, **Then** Kairos acknowledges their return and continues from the next unanswered question (not from the beginning).

3. **Given** a user abandoned KYC and Kairos already sent the inactivity nudge, **When** the user does not respond for another 72 hours, **Then** Kairos does NOT send additional nudge messages for the same onboarding attempt.

4. **Given** a user abandoned KYC 10 days ago with partial data, **When** they send a new message, **Then** Kairos greets them fresh and restarts the KYC from Turn 1 (partial state expired after 7 days).

---

### User Story 3 - Data Correction During KYC (Priority: P3)

A user realizes they gave a wrong answer to a previous question and wants to correct it mid-flow. Kairos understands the correction intent, updates the specific field, and continues the flow without requiring the user to restart the entire KYC.

**Why this priority**: Users make mistakes. Forcing a restart creates frustration and increases abandonment. This builds trust that Kairos truly "listens."

**Independent Test**: Can be tested by completing 4 questions, then sending "Espera, mi objetivo no es ganar masa, es bajar grasa", and verifying the goal field is updated while other answers remain intact.

**Acceptance Scenarios**:

1. **Given** a user has answered 4 of 5 questions with goal = "Ganar masa muscular", **When** the user says "Cambio de idea, quiero bajar grasa", **Then** Kairos updates only the goal field and confirms the change without asking the user to re-answer other questions.

2. **Given** a user corrects a previously answered field, **When** the correction is processed, **Then** subsequent questions that depend on the corrected field are re-evaluated (e.g., if goal changes, follow-up questions adapt to new goal context).

---

### User Story 4 - Health Condition Filter (Priority: P3)

During KYC, when a user reports a health condition (injury, chronic pain, physical limitation), Kairos flags the user's health status accordingly. For severe conditions, Kairos recommends consulting a professional trainer rather than generating an automated routine.

**Why this priority**: Safety is a core principle. Identifying health risks early prevents injury and establishes trust. Users with conditions like "tengo una lesion en la rodilla" need appropriate handling before any routine is generated.

**Independent Test**: Can be tested by answering the health question with "Tengo una lesion en la rodilla derecha" and verifying the health status is set to the appropriate code (B for lower body) and exercise restrictions are recorded.

**Acceptance Scenarios**:

1. **Given** a user is answering the health status question, **When** they report a lower body issue (e.g., knee pain), **Then** Kairos sets health status to code B and records "rodilla" as an affected zone.

2. **Given** a user is answering the health status question, **When** they report a condition that classifies as severe (health code E), **Then** Kairos recommends connecting with a human trainer and does NOT proceed to automated routine generation.

3. **Given** a user reports no health issues, **When** health status is recorded, **Then** the profile is set to health code A (no restrictions) and routine generation proceeds normally.

---

### Edge Cases

- What happens when a user sends only emojis or stickers during KYC? Kairos acknowledges the input and re-asks the current question in a friendly way.
- What happens when a user sends a voice note instead of text? Kairos informs the user it currently only processes text messages and asks them to type their response.
- What happens when a user sends a WhatsApp status update (not a direct message)? The system ignores it entirely (noise filter).
- What happens when two messages arrive simultaneously from the same user? The system processes them sequentially to avoid race conditions in profile creation.
- What happens when a user who already completed KYC sends "quiero empezar de nuevo"? Kairos offers to update specific fields rather than restarting the entire questionnaire.
- What happens when the user's WhatsApp display name is empty or contains only special characters? Kairos uses a generic greeting ("Hola!") instead of the name.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST detect first-time users by checking whether the incoming WhatsApp phone number exists in the users database.
- **FR-002**: System MUST extract the user's display name from the WhatsApp profile for personalized greetings.
- **FR-003**: System MUST conduct the KYC as a natural conversation, asking one question at a time, NOT presenting all questions simultaneously.
- **FR-004**: System MUST collect 10 data points during KYC, grouped into 5 conversational turns: Turn 1 — training goal; Turn 2 — experience level + days per week + preferred schedule; Turn 3 — training environment (gym/home) + equipment available; Turn 4 — biological sex + age + height + weight; Turn 5 — health status.
- **FR-005**: System MUST display progress indicators with each question (e.g., "Pregunta 3 de 5") to set user expectations.
- **FR-006**: System MUST detect and register multiple data points provided in a single user message.
- **FR-007**: System MUST persist partial KYC state so users can resume from where they left off after any interruption.
- **FR-008**: System MUST send an inactivity nudge after 30 minutes of silence during an active KYC session.
- **FR-009**: System MUST NOT send more than one nudge message per abandoned KYC attempt.
- **FR-010**: System MUST allow users to correct previously provided answers without restarting the KYC.
- **FR-011**: System MUST classify user health conditions into status codes (A through E) based on reported issues.
- **FR-012**: System MUST record specific affected body zones when a health condition is reported.
- **FR-013**: System MUST route users with severe health conditions (code E) to a human trainer recommendation instead of automated routine generation.
- **FR-014**: System MUST complete the entire onboarding session within 8 minutes for a cooperative user.
- **FR-015**: All KYC messages from Kairos MUST be in Spanish (Colombian dialect) with a motivational, friendly tone.
- **FR-016**: System MUST present a profile summary for user confirmation before triggering routine generation.
- **FR-017**: System MUST ignore WhatsApp status updates and non-direct-message events (noise filter).
- **FR-018**: When a user rejects the profile summary, system MUST ask which specific field is incorrect, update only that field, and re-present the summary for confirmation.
- **FR-019**: Partial KYC state MUST expire after 7 days of inactivity. If a user returns after 7+ days, the KYC restarts from Turn 1.
- **FR-020**: Preference fields (secondary goal, priority muscles, disliked exercises, cardio type, cardio frequency) are explicitly OUT OF SCOPE for KYC. These MUST be collected post-routine via feedback interactions (F-03).

### Key Entities

- **User Profile**: The core identity record. Contains phone number, display name, email, timezone. Created upon first interaction. Linked to all other entities via user ID.
- **Gym Profile (KYC Data)**: The detailed fitness profile collected during onboarding. Contains training goal, experience level, available days, health status, biological sex, body metrics, equipment preferences. Linked to User Profile via phone number.
- **KYC Session State**: The in-progress onboarding conversation state. Tracks which questions have been answered, partial responses collected, and the timestamp of last interaction. Used for resumption after abandonment.
- **Health Condition Record**: Classification of user-reported health issues. Maps free-text descriptions to health codes (A-E) and records specific affected body zones for exercise filtering.

### Assumptions

- Users interact exclusively via WhatsApp text messages (voice notes, images, and documents are acknowledged but not processed for KYC data extraction).
- The WhatsApp Business API provides the sender's display name in the message metadata.
- The 30-minute inactivity timer is measured from the last user message, not the last Kairos message.
- KYC question grouping: The 10 data points are grouped into exactly 5 conversational turns: (1) goal, (2) experience + days + schedule, (3) environment + equipment, (4) sex + age + height + weight, (5) health status.
- Health condition classification uses keyword matching + LLM understanding to map free-text descriptions to health codes.
- The system uses UTC internally but displays times in the user's configured timezone (America/Bogota by default).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 80% of users who start the KYC complete it in a single session without abandonment.
- **SC-002**: Users who do abandon and later return resume from their last answered question 100% of the time (no re-asks).
- **SC-003**: The complete KYC conversation takes no more than 8 minutes for a cooperative user who answers promptly.
- **SC-004**: 100% of completed profiles contain all required fields with valid values (no nulls in mandatory fields).
- **SC-005**: Users who report health conditions are classified into the correct health code with 95% accuracy.
- **SC-006**: The inactivity nudge is sent within 1 minute of the 30-minute threshold being reached.
- **SC-007**: Zero users with health code E receive an automatically generated routine (safety gate has 100% enforcement).
- **SC-008**: Multi-value messages (user provides 2+ data points at once) are correctly parsed and registered in 90%+ of cases.
