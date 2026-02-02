package repository

import (
	"context"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
)

// ProfileReader defines read operations for user gym profiles
type ProfileReader interface {
	// GetByUserID retrieves the gym profile by user ID (via phone number matching)
	GetByUserID(ctx context.Context, userID string) (*entity.UserGymProfile, error)
	// GetByWhatsAppID retrieves the gym profile by WhatsApp ID
	GetByWhatsAppID(ctx context.Context, whatsappID string) (*entity.UserGymProfile, error)
}

// ProfileWriter defines write operations for user gym profiles
type ProfileWriter interface {
	// UpdatePreferences updates priority muscles, health status, and session duration
	UpdatePreferences(ctx context.Context, whatsappID string, priorityMuscles, healthStatus, sessionDuration string) error
}

// ProfileRepository combines all user profile operations
// Following Interface Segregation Principle (ISP)
type ProfileRepository interface {
	ProfileReader
	ProfileWriter
}
