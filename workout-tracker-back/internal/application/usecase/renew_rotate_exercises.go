package usecase

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/internal/domain/service"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// RenewRotateExercisesUseCase handles renewing with new exercises
type RenewRotateExercisesUseCase struct {
	planRepo           repository.PlanRepository
	scheduleRepo       repository.ScheduleWriter
	workoutRenewalRepo repository.WorkoutRenewalRepository
	profileRepo        repository.ProfileReader
	exerciseRotation   *service.ExerciseRotationService
}

// NewRenewRotateExercisesUseCase creates a new RenewRotateExercisesUseCase
func NewRenewRotateExercisesUseCase(
	planRepo repository.PlanRepository,
	scheduleRepo repository.ScheduleWriter,
	workoutRenewalRepo repository.WorkoutRenewalRepository,
	profileRepo repository.ProfileReader,
	exerciseRotation *service.ExerciseRotationService,
) *RenewRotateExercisesUseCase {
	return &RenewRotateExercisesUseCase{
		planRepo:           planRepo,
		scheduleRepo:       scheduleRepo,
		workoutRenewalRepo: workoutRenewalRepo,
		profileRepo:        profileRepo,
		exerciseRotation:   exerciseRotation,
	}
}

// Execute renews the mesocycle with rotated exercises
func (uc *RenewRotateExercisesUseCase) Execute(
	ctx context.Context,
	userID string,
	req *dto.RenewRotateExercisesRequest,
) (*dto.RotateExercisesResponse, error) {
	if userID == "" {
		return nil, apperror.NewValidationError("user_id is required")
	}

	// Default: rotate all exercise types
	if req == nil {
		req = &dto.RenewRotateExercisesRequest{
			RotateCompounds: true,
			RotateIsolation: true,
			RotateCore:      true,
		}
	}

	// 1. Verify mesocycle is complete
	status, err := uc.planRepo.GetMesocycleStatus(ctx, userID)
	if err != nil {
		return nil, err
	}

	if status == nil {
		return nil, apperror.NewNotFoundError("no active plan found for user")
	}

	if !status.IsComplete {
		return nil, apperror.NewValidationError("mesocycle is not yet complete")
	}

	// 2. Get user profile for preferences
	profile, err := uc.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.NewInternalError("failed to get user profile", err)
	}

	// Use empty profile if not found (allows rotation without profile)
	if profile == nil {
		profile = &entity.UserGymProfile{}
	}

	// 3. Get current workouts
	workouts, err := uc.workoutRenewalRepo.GetUserWorkoutExercises(ctx, userID)
	if err != nil {
		return nil, apperror.NewInternalError("failed to get workouts", err)
	}

	// 4. Filter workouts by rotation options
	workoutsToRotate := filterWorkoutsByRole(workouts, req)

	// 5. Rotate exercises
	rotations, err := uc.exerciseRotation.RotateExercises(ctx, workoutsToRotate, profile)
	if err != nil {
		return nil, apperror.NewInternalError("failed to rotate exercises", err)
	}

	// 6. Apply rotations to database
	if len(rotations) > 0 {
		if err := uc.workoutRenewalRepo.BatchUpdateExercises(ctx, rotations); err != nil {
			return nil, apperror.NewInternalError("failed to apply exercise rotations", err)
		}
	}

	// 7. Clear schedule
	if err := uc.scheduleRepo.ClearSchedule(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to clear schedule", err)
	}

	// 8. Reset workout weeks to 1
	if err := uc.workoutRenewalRepo.ResetWeeksToOne(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to reset workout weeks", err)
	}

	// 9. Increment mesocycle
	if err := uc.planRepo.IncrementMesocycle(ctx, userID); err != nil {
		return nil, apperror.NewInternalError("failed to increment mesocycle", err)
	}

	// Build response
	rotationDTOs := make([]dto.ExerciseRotationDTO, len(rotations))
	for i, r := range rotations {
		rotationDTOs[i] = dto.ExerciseRotationDTO{
			WorkoutID:       r.WorkoutID,
			OldExerciseID:   r.OldExerciseID,
			OldExerciseName: r.OldExerciseName,
			NewExerciseID:   r.NewExerciseID,
			NewExerciseName: r.NewExerciseName,
			Pattern:         r.Pattern,
			Role:            r.Role,
		}
	}

	return &dto.RotateExercisesResponse{
		RenewalResponse: dto.RenewalResponse{
			Success:            true,
			RenewalType:        "ROTAR_EJERCICIOS",
			NewMesocycleNumber: status.MesocycleNumber + 1,
			Message:            "Tus ejercicios han sido actualizados para el nuevo mesociclo.",
			ScheduleCleared:    true,
			ExercisesRotated:   len(rotations),
		},
		Rotations: rotationDTOs,
	}, nil
}

// filterWorkoutsByRole filters workouts based on rotation request options
func filterWorkoutsByRole(
	workouts []*entity.WorkoutExercise,
	req *dto.RenewRotateExercisesRequest,
) []*entity.WorkoutExercise {
	// If all options are true, return all workouts
	if req.RotateCompounds && req.RotateIsolation && req.RotateCore {
		return workouts
	}

	filtered := make([]*entity.WorkoutExercise, 0)
	for _, w := range workouts {
		switch w.Role {
		case "compound":
			if req.RotateCompounds {
				filtered = append(filtered, w)
			}
		case "isolation":
			if req.RotateIsolation {
				filtered = append(filtered, w)
			}
		case "core":
			if req.RotateCore {
				filtered = append(filtered, w)
			}
		}
	}
	return filtered
}
