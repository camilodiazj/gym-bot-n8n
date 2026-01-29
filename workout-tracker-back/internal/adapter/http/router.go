package http

import (
	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/internal/adapter/http/handler"
	"github.com/gymbot/workout-tracker-back/internal/adapter/http/middleware"
)

// Router holds all HTTP handlers and configures routes
type Router struct {
	engine         *gin.Engine
	healthHandler  *handler.HealthHandler
	workoutHandler *handler.WorkoutHandler
	setHandler     *handler.SetHandler
}

// NewRouter creates a new Router with all dependencies
func NewRouter(
	healthHandler *handler.HealthHandler,
	workoutHandler *handler.WorkoutHandler,
	setHandler *handler.SetHandler,
) *Router {
	return &Router{
		healthHandler:  healthHandler,
		workoutHandler: workoutHandler,
		setHandler:     setHandler,
	}
}

// Setup configures the Gin engine with all routes and middleware
func (r *Router) Setup(ginMode string) *gin.Engine {
	gin.SetMode(ginMode)
	r.engine = gin.New()

	// Global middleware
	r.engine.Use(gin.Logger())
	r.engine.Use(middleware.ErrorHandler())
	r.engine.Use(middleware.CORS())

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", r.healthHandler.Check)

		// Workout routes
		workouts := v1.Group("/workouts")
		{
			workouts.GET("/today", r.workoutHandler.GetTodayWorkout)
			workouts.POST("/:workoutId/complete", r.workoutHandler.CompleteWorkout)
		}

		// Set routes
		sets := v1.Group("/sets")
		{
			sets.PATCH("/:setId", r.setHandler.Update)
			sets.PATCH("/:setId/complete", r.setHandler.MarkComplete)
		}
	}

	return r.engine
}

// Run starts the HTTP server
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}
