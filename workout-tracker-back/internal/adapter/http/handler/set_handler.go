package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/application/usecase"
	"github.com/gymbot/workout-tracker-back/pkg/response"
)

// SetHandler handles set-related HTTP requests
type SetHandler struct {
	markSetComplete *usecase.MarkSetCompleteUseCase
	updateSet       *usecase.UpdateSetUseCase
}

// NewSetHandler creates a new SetHandler
func NewSetHandler(
	markSetComplete *usecase.MarkSetCompleteUseCase,
	updateSet *usecase.UpdateSetUseCase,
) *SetHandler {
	return &SetHandler{
		markSetComplete: markSetComplete,
		updateSet:       updateSet,
	}
}

// MarkComplete handles PATCH /api/v1/sets/:setId/complete
func (h *SetHandler) MarkComplete(c *gin.Context) {
	setID := c.Param("setId")
	if setID == "" {
		response.BadRequest(c, "setId path parameter is required")
		return
	}

	result, err := h.markSetComplete.Execute(c.Request.Context(), setID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// Update handles PATCH /api/v1/sets/:setId
func (h *SetHandler) Update(c *gin.Context) {
	setID := c.Param("setId")
	if setID == "" {
		response.BadRequest(c, "setId path parameter is required")
		return
	}

	var req dto.UpdateSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	result, err := h.updateSet.Execute(c.Request.Context(), setID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}
