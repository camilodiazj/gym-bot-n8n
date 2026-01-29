package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/application/usecase"
	"github.com/gymbot/workout-tracker-back/pkg/response"
)

// WorkoutHandler handles workout-related HTTP requests
type WorkoutHandler struct {
	getTodayWorkout *usecase.GetTodayWorkoutUseCase
	completeWorkout *usecase.CompleteWorkoutUseCase
}

// NewWorkoutHandler creates a new WorkoutHandler
func NewWorkoutHandler(
	getTodayWorkout *usecase.GetTodayWorkoutUseCase,
	completeWorkout *usecase.CompleteWorkoutUseCase,
) *WorkoutHandler {
	return &WorkoutHandler{
		getTodayWorkout: getTodayWorkout,
		completeWorkout: completeWorkout,
	}
}

// GetTodayWorkout handles GET /api/v1/workouts/today
func (h *WorkoutHandler) GetTodayWorkout(c *gin.Context) {
	// TODO: Get user ID from auth context/token
	// For now, accept it as a query parameter for testing
	userID := c.Query("user_id")
	if userID == "" {
		response.BadRequest(c, "user_id query parameter is required")
		return
	}

	result, err := h.getTodayWorkout.Execute(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// CompleteWorkout handles POST /api/v1/workouts/:workoutId/complete
func (h *WorkoutHandler) CompleteWorkout(c *gin.Context) {
	workoutID := c.Param("workoutId")
	if workoutID == "" {
		response.BadRequest(c, "workoutId path parameter is required")
		return
	}

	var req dto.CompleteWorkoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Request body is optional, so we ignore binding errors
		req = dto.CompleteWorkoutRequest{}
	}

	result, err := h.completeWorkout.Execute(c.Request.Context(), workoutID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}
