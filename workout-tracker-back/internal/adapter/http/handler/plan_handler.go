package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/application/usecase"
	"github.com/gymbot/workout-tracker-back/pkg/response"
)

// PlanHandler handles plan-related HTTP requests
type PlanHandler struct {
	checkStatus        *usecase.CheckMesocycleStatusUseCase
	renewMaintain      *usecase.RenewMaintainUseCase
	renewRotate        *usecase.RenewRotateExercisesUseCase
	renewChangeDays    *usecase.RenewChangeDaysUseCase
	renewUpdateProfile *usecase.RenewUpdateProfileUseCase
}

// NewPlanHandler creates a new PlanHandler
func NewPlanHandler(
	checkStatus *usecase.CheckMesocycleStatusUseCase,
	renewMaintain *usecase.RenewMaintainUseCase,
	renewRotate *usecase.RenewRotateExercisesUseCase,
	renewChangeDays *usecase.RenewChangeDaysUseCase,
	renewUpdateProfile *usecase.RenewUpdateProfileUseCase,
) *PlanHandler {
	return &PlanHandler{
		checkStatus:        checkStatus,
		renewMaintain:      renewMaintain,
		renewRotate:        renewRotate,
		renewChangeDays:    renewChangeDays,
		renewUpdateProfile: renewUpdateProfile,
	}
}

// GetMesocycleStatus handles GET /api/v1/plans/:userId/mesocycle-status
func (h *PlanHandler) GetMesocycleStatus(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		response.BadRequest(c, "userId path parameter is required")
		return
	}

	result, err := h.checkStatus.Execute(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// RenewMaintain handles POST /api/v1/plans/:userId/renew/maintain
func (h *PlanHandler) RenewMaintain(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		response.BadRequest(c, "userId path parameter is required")
		return
	}

	var req dto.RenewMaintainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Request body is optional
		req = dto.RenewMaintainRequest{}
	}

	result, err := h.renewMaintain.Execute(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// RenewRotateExercises handles POST /api/v1/plans/:userId/renew/rotate-exercises
func (h *PlanHandler) RenewRotateExercises(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		response.BadRequest(c, "userId path parameter is required")
		return
	}

	var req dto.RenewRotateExercisesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Default: rotate all exercise types
		req = dto.RenewRotateExercisesRequest{
			RotateCompounds: true,
			RotateIsolation: true,
			RotateCore:      true,
		}
	}

	result, err := h.renewRotate.Execute(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// RenewChangeDays handles POST /api/v1/plans/:userId/renew/change-days
func (h *PlanHandler) RenewChangeDays(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		response.BadRequest(c, "userId path parameter is required")
		return
	}

	var req dto.RenewChangeDaysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	result, err := h.renewChangeDays.Execute(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// RenewUpdateProfile handles POST /api/v1/plans/:userId/renew/update-profile
func (h *PlanHandler) RenewUpdateProfile(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		response.BadRequest(c, "userId path parameter is required")
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Request body is optional - profile updates are optional
		req = dto.UpdateProfileRequest{}
	}

	result, err := h.renewUpdateProfile.Execute(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}
