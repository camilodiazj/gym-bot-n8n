package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gymbot/workout-tracker-back/internal/adapter/http"
	"github.com/gymbot/workout-tracker-back/internal/adapter/http/handler"
	"github.com/gymbot/workout-tracker-back/internal/adapter/repository/postgres"
	"github.com/gymbot/workout-tracker-back/internal/application/usecase"
	"github.com/gymbot/workout-tracker-back/internal/config"
	"github.com/gymbot/workout-tracker-back/internal/domain/service"
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

	// ==================== REPOSITORIES ====================

	// Existing repositories
	setRepo := postgres.NewSetRepository(dbConn)
	workoutRepo := postgres.NewWorkoutRepository(dbConn, setRepo)
	magicLinkRepo := postgres.NewMagicLinkRepository(dbConn)

	// Renewal repositories
	planRepo := postgres.NewPlanRepository(dbConn)
	scheduleRepo := postgres.NewScheduleRepository(dbConn)
	exerciseRepo := postgres.NewExerciseRepository(dbConn)
	workoutRenewalRepo := postgres.NewWorkoutRenewalRepository(dbConn)
	userProfileRepo := postgres.NewUserProfileRepository(dbConn)

	// ==================== SERVICES ====================

	// Domain services
	preferenceProcessor := service.NewPreferenceProcessor()
	exerciseRotationService := service.NewExerciseRotationService(exerciseRepo, preferenceProcessor)

	// Code resolver for magic links
	codeResolver := func(code string) (string, error) {
		return magicLinkRepo.GetUserID(context.Background(), code)
	}

	// ==================== USE CASES ====================

	// Existing use cases
	getTodayWorkoutUC := usecase.NewGetTodayWorkoutUseCase(workoutRepo)
	completeWorkoutUC := usecase.NewCompleteWorkoutUseCase(workoutRepo, magicLinkRepo)
	markSetCompleteUC := usecase.NewMarkSetCompleteUseCase(setRepo)
	updateSetUC := usecase.NewUpdateSetUseCase(setRepo)

	// Renewal use cases
	checkMesocycleStatusUC := usecase.NewCheckMesocycleStatusUseCase(planRepo)
	renewMaintainUC := usecase.NewRenewMaintainUseCase(planRepo, scheduleRepo, workoutRenewalRepo)
	renewRotateExercisesUC := usecase.NewRenewRotateExercisesUseCase(
		planRepo,
		scheduleRepo,
		workoutRenewalRepo,
		userProfileRepo,
		exerciseRotationService,
	)
	renewChangeDaysUC := usecase.NewRenewChangeDaysUseCase(planRepo, scheduleRepo, workoutRenewalRepo)
	renewUpdateProfileUC := usecase.NewRenewUpdateProfileUseCase(planRepo, scheduleRepo, workoutRenewalRepo, userProfileRepo)

	// ==================== HANDLERS ====================

	// Existing handlers
	healthHandler := handler.NewHealthHandler()
	workoutHandler := handler.NewWorkoutHandler(getTodayWorkoutUC, completeWorkoutUC)
	setHandler := handler.NewSetHandler(markSetCompleteUC, updateSetUC)

	// Plan handler
	planHandler := handler.NewPlanHandler(
		checkMesocycleStatusUC,
		renewMaintainUC,
		renewRotateExercisesUC,
		renewChangeDaysUC,
		renewUpdateProfileUC,
	)

	// ==================== ROUTER ====================

	router := http.NewRouter(
		healthHandler,
		workoutHandler,
		setHandler,
		planHandler,
		codeResolver,
		cfg.Server.InternalAPIKey,
		cfg.Server.CORSAllowedOrigins,
	)
	engine := router.Setup(cfg.Server.GinMode)

	// ==================== START SERVER ====================

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
