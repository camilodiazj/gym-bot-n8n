package dto

// UpdateSetRequest is the request for PATCH /api/v1/sets/:setId
type UpdateSetRequest struct {
	Reps *int    `json:"reps,omitempty"`
	Kg   *string `json:"kg,omitempty"`
}

// UpdateSetResponse is the response for PATCH /api/v1/sets/:setId
type UpdateSetResponse struct {
	Success   bool   `json:"success"`
	SetID     string `json:"set_id"`
	UpdatedAt string `json:"updated_at"`
}

// MarkSetCompleteResponse is the response for PATCH /api/v1/sets/:setId/complete
type MarkSetCompleteResponse struct {
	Success     bool   `json:"success"`
	SetID       string `json:"set_id"`
	CompletedAt string `json:"completed_at"`
}
