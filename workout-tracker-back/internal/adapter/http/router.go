package http

import (
	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/internal/adapter/http/handler"
	"github.com/gymbot/workout-tracker-back/internal/adapter/http/middleware"
)

// Router holds all HTTP handlers and configures routes
type Router struct {
	engine             *gin.Engine
	healthHandler      *handler.HealthHandler
	workoutHandler     *handler.WorkoutHandler
	setHandler         *handler.SetHandler
	draftHandler       *handler.DraftHandler
	codeResolver       middleware.CodeResolver
	corsAllowedOrigins []string
}

// NewRouter creates a new Router with all dependencies
func NewRouter(
	healthHandler *handler.HealthHandler,
	workoutHandler *handler.WorkoutHandler,
	setHandler *handler.SetHandler,
	draftHandler *handler.DraftHandler,
	codeResolver middleware.CodeResolver,
	corsAllowedOrigins []string,
) *Router {
	return &Router{
		healthHandler:      healthHandler,
		workoutHandler:     workoutHandler,
		setHandler:         setHandler,
		draftHandler:       draftHandler,
		codeResolver:       codeResolver,
		corsAllowedOrigins: corsAllowedOrigins,
	}
}

// Setup configures the Gin engine with all routes and middleware
func (r *Router) Setup(ginMode string) *gin.Engine {
	gin.SetMode(ginMode)
	r.engine = gin.New()

	// Global middleware
	r.engine.Use(gin.Logger())
	r.engine.Use(middleware.ErrorHandler())
	r.engine.Use(middleware.CORS(r.corsAllowedOrigins))

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Health check (public)
		v1.GET("/health", r.healthHandler.Check)

		// Draft routes (public — the code in the URL IS the authentication)
		drafts := v1.Group("/drafts")
		{
			drafts.GET("/:code", r.draftHandler.GetDraft)
			drafts.PATCH("/:code/swap", r.draftHandler.SwapExercise)
			drafts.POST("/:code/approve", r.draftHandler.ApproveDraft)
		}

		// Auth middleware (supports ?c= and ?user_id= for development)
		authMiddleware := middleware.ValidateAuth(r.codeResolver)

		// Workout routes (protected)
		workouts := v1.Group("/workouts")
		workouts.Use(authMiddleware)
		{
			workouts.GET("/today", r.workoutHandler.GetTodayWorkout)
			workouts.POST("/:workoutId/complete", r.workoutHandler.CompleteWorkout)
		}

		// Set routes (protected)
		sets := v1.Group("/sets")
		sets.Use(authMiddleware)
		{
			sets.PATCH("/:setId", r.setHandler.Update)
			sets.PATCH("/:setId/complete", r.setHandler.MarkComplete)
		}
	}

	// Serve static frontend files (SPA)
	r.engine.Static("/assets", "./static/assets")
	r.engine.StaticFile("/vite.svg", "./static/vite.svg")

	// SPA fallback: serve index.html for all non-API routes
	r.engine.NoRoute(func(c *gin.Context) {
		c.File("./static/index.html")
	})

	return r.engine
}

// Run starts the HTTP server
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}
