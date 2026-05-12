// Package port defines outbound interfaces (ports) that the application layer
// depends on. Concrete adapters (HTTP clients, message buses, etc.) live in
// internal/adapter/* and implement these interfaces so the application code
// stays free of transport-level concerns.
package port

import "context"

// FinalizeDraftResult holds the outcome of asking Kairos to materialize a
// draft into a real users_plans + workouts set.
type FinalizeDraftResult struct {
	// PlanID is the users_plans.plan_id created (or pre-existing) for this draft.
	PlanID string
	// WorkoutsCreated is the count of rows inserted into workouts. When Kairos
	// detects an active plan already exists for the user, it returns 0 and
	// AlreadyExisted is set to true.
	WorkoutsCreated int
	// AlreadyExisted reports whether Kairos short-circuited because the user
	// already had an active plan. Useful for idempotent retries.
	AlreadyExisted bool
}

// KairosClient is the outbound port used by the application to drive
// side-effecting operations on the Kairos agent service.
//
// Splitting this as a focused interface (Interface Segregation) means the
// application layer never sees an "everything client" — only the verbs it
// needs. New operations should add new methods here (or new sibling
// interfaces) rather than introducing a god-object.
type KairosClient interface {
	// FinalizeDraft asks Kairos to load draft_routines.draft_data by code and
	// produce the corresponding users_plans + workouts rows. The call must be
	// idempotent on the Kairos side: a retry for an already-finalized draft
	// returns the existing plan_id with WorkoutsCreated=0 and AlreadyExisted=true.
	FinalizeDraft(ctx context.Context, code string) (*FinalizeDraftResult, error)
}
