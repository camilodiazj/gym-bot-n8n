package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	startTime time.Time
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Uptime    string `json:"uptime"`
}

// Check handles GET /api/v1/health
func (h *HealthHandler) Check(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    time.Since(h.startTime).String(),
	})
}
