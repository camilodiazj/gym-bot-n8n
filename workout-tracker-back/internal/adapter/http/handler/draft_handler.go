package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/application/usecase"
	"github.com/gymbot/workout-tracker-back/pkg/response"
)

// DraftHandler handles draft routine HTTP requests
type DraftHandler struct {
	getDraft     *usecase.GetDraftUseCase
	swapExercise *usecase.SwapExerciseUseCase
	approveDraft *usecase.ApproveDraftUseCase
}

// NewDraftHandler creates a new DraftHandler
func NewDraftHandler(
	getDraft *usecase.GetDraftUseCase,
	swapExercise *usecase.SwapExerciseUseCase,
	approveDraft *usecase.ApproveDraftUseCase,
) *DraftHandler {
	return &DraftHandler{
		getDraft:     getDraft,
		swapExercise: swapExercise,
		approveDraft: approveDraft,
	}
}

// GetDraft handles GET /api/v1/drafts/:code
func (h *DraftHandler) GetDraft(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.BadRequest(c, "code path parameter is required")
		return
	}

	result, err := h.getDraft.Execute(c.Request.Context(), code)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// SwapExercise handles PATCH /api/v1/drafts/:code/swap
func (h *DraftHandler) SwapExercise(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.BadRequest(c, "code path parameter is required")
		return
	}

	var req dto.SwapExerciseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: day_number, exercise_order, and new_exercise_id are required")
		return
	}

	result, err := h.swapExercise.Execute(c.Request.Context(), code, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// ApproveDraft handles POST /api/v1/drafts/:code/approve
func (h *DraftHandler) ApproveDraft(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.BadRequest(c, "code path parameter is required")
		return
	}

	result, err := h.approveDraft.Execute(c.Request.Context(), code)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}
