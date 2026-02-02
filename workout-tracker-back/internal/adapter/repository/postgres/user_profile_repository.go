package postgres

import (
	"context"
	"database/sql"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// UserProfileRepository implements repository.ProfileRepository
type UserProfileRepository struct {
	conn *Connection
}

// Ensure UserProfileRepository implements the interface
var _ repository.ProfileRepository = (*UserProfileRepository)(nil)

// NewUserProfileRepository creates a new UserProfileRepository
func NewUserProfileRepository(conn *Connection) *UserProfileRepository {
	return &UserProfileRepository{conn: conn}
}

// GetByUserID retrieves the gym profile by user ID (via phone number matching)
func (r *UserProfileRepository) GetByUserID(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
	// Join users table to get the phone number, then match to users_gym_profile
	// The whatsapp_id format is typically: "573001234567@s.whatsapp.net"
	// The cel_number format is typically: "3001234567"
	query := `
		SELECT
			ugp.whatsapp_id,
			ugp.full_name,
			ugp.email,
			ugp.birthdate,
			ugp.sex,
			ugp.height,
			ugp.weight,
			ugp.training_goal,
			ugp.fitness_level,
			ugp.training_experience,
			ugp.days_per_week,
			ugp.session_duration_mins,
			ugp.health_status,
			ugp.priority_muscles,
			ugp.disliked_exercises,
			ugp.available_equipment,
			ugp.training_environment,
			ugp.preferred_training_time,
			ugp.created_at,
			ugp.updated_at
		FROM users_gym_profile ugp
		JOIN users u ON ugp.whatsapp_id LIKE '%' || u.cel_number || '%'
		WHERE u.user_id = $1
		LIMIT 1
	`

	return r.scanProfile(ctx, query, userID)
}

// GetByWhatsAppID retrieves the gym profile by WhatsApp ID
func (r *UserProfileRepository) GetByWhatsAppID(ctx context.Context, whatsappID string) (*entity.UserGymProfile, error) {
	query := `
		SELECT
			whatsapp_id,
			full_name,
			email,
			birthdate,
			sex,
			height,
			weight,
			training_goal,
			fitness_level,
			training_experience,
			days_per_week,
			session_duration_mins,
			health_status,
			priority_muscles,
			disliked_exercises,
			available_equipment,
			training_environment,
			preferred_training_time,
			created_at,
			updated_at
		FROM users_gym_profile
		WHERE whatsapp_id = $1
	`

	return r.scanProfile(ctx, query, whatsappID)
}

// scanProfile is a helper function to scan a profile from a query result
func (r *UserProfileRepository) scanProfile(ctx context.Context, query string, arg interface{}) (*entity.UserGymProfile, error) {
	var profile entity.UserGymProfile
	var fullName, email sql.NullString
	var birthdate, sex, trainingGoal, fitnessLevel, trainingExperience sql.NullString
	var sessionDuration, healthStatus, priorityMuscles, dislikedExercises sql.NullString
	var availableEquipment, trainingEnvironment, preferredTime sql.NullString
	var height sql.NullInt64
	var weight sql.NullFloat64
	var daysPerWeek sql.NullInt64
	var createdAt, updatedAt sql.NullTime

	err := r.conn.DB.QueryRowContext(ctx, query, arg).Scan(
		&profile.WhatsAppID,
		&fullName,
		&email,
		&birthdate,
		&sex,
		&height,
		&weight,
		&trainingGoal,
		&fitnessLevel,
		&trainingExperience,
		&daysPerWeek,
		&sessionDuration,
		&healthStatus,
		&priorityMuscles,
		&dislikedExercises,
		&availableEquipment,
		&trainingEnvironment,
		&preferredTime,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query user profile", err)
	}

	profile.FullName = fullName.String
	profile.Email = email.String
	profile.Birthdate = birthdate.String
	profile.Sex = sex.String
	profile.Height = int(height.Int64)
	profile.Weight = weight.Float64
	profile.TrainingGoal = trainingGoal.String
	profile.FitnessLevel = fitnessLevel.String
	profile.TrainingExperience = trainingExperience.String
	profile.DaysPerWeek = int(daysPerWeek.Int64)
	profile.SessionDurationMins = sessionDuration.String
	profile.HealthStatus = healthStatus.String
	profile.PriorityMuscles = priorityMuscles.String
	profile.DislikedExercises = dislikedExercises.String
	profile.AvailableEquipment = availableEquipment.String
	profile.TrainingEnvironment = trainingEnvironment.String
	profile.PreferredTrainingTime = preferredTime.String
	if createdAt.Valid {
		profile.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		profile.UpdatedAt = updatedAt.Time
	}

	return &profile, nil
}

// UpdatePreferences updates priority muscles, health status, and session duration
func (r *UserProfileRepository) UpdatePreferences(
	ctx context.Context,
	whatsappID string,
	priorityMuscles, healthStatus, sessionDuration string,
) error {
	query := `
		UPDATE users_gym_profile
		SET
			priority_muscles = COALESCE(NULLIF($2, ''), priority_muscles),
			health_status = COALESCE(NULLIF($3, ''), health_status),
			session_duration_mins = COALESCE(NULLIF($4, ''), session_duration_mins),
			updated_at = NOW()
		WHERE whatsapp_id = $1
	`

	result, err := r.conn.DB.ExecContext(ctx, query, whatsappID, priorityMuscles, healthStatus, sessionDuration)
	if err != nil {
		return apperror.NewInternalError("failed to update preferences", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return apperror.NewNotFoundError("user profile not found")
	}

	return nil
}
