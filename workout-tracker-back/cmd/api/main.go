package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gymbot/workout-tracker-back/internal/adapter/http"
	"github.com/gymbot/workout-tracker-back/internal/adapter/http/handler"
	"github.com/gymbot/workout-tracker-back/internal/adapter/repository/postgres"
	"github.com/gymbot/workout-tracker-back/internal/application/usecase"
	"github.com/gymbot/workout-tracker-back/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	dbConn, err := postgres.NewConnectionFromURL(cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	log.Println("Connected to database successfully")

	// Initialize repositories (Secondary Adapters)
	workoutRepo := postgres.NewWorkoutRepository(dbConn)
	setRepo := postgres.NewSetRepository(dbConn)

	// Initialize use cases (Application Layer)
	getTodayWorkoutUC := usecase.NewGetTodayWorkoutUseCase(workoutRepo)
	completeWorkoutUC := usecase.NewCompleteWorkoutUseCase(workoutRepo)
	markSetCompleteUC := usecase.NewMarkSetCompleteUseCase(setRepo)
	updateSetUC := usecase.NewUpdateSetUseCase(setRepo)

	// Initialize HTTP handlers (Primary Adapters)
	healthHandler := handler.NewHealthHandler()
	workoutHandler := handler.NewWorkoutHandler(getTodayWorkoutUC, completeWorkoutUC)
	setHandler := handler.NewSetHandler(markSetCompleteUC, updateSetUC)

	// Initialize router and setup routes
	router := http.NewRouter(healthHandler, workoutHandler, setHandler)
	engine := router.Setup(cfg.Server.GinMode)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", cfg.ServerAddr())
		if err := engine.Run(cfg.ServerAddr()); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
