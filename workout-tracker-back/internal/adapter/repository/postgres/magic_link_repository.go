package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"time"

	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// MagicLinkRepository handles magic link operations
type MagicLinkRepository struct {
	conn *Connection
}

// NewMagicLinkRepository creates a new MagicLinkRepository
func NewMagicLinkRepository(conn *Connection) *MagicLinkRepository {
	return &MagicLinkRepository{conn: conn}
}

// generateCode creates a random 6-character code
func generateCode() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use URL-safe base64, take first 6 chars
	code := base64.URLEncoding.EncodeToString(bytes)[:6]
	return code, nil
}

// Create generates a new magic link code for a user
func (r *MagicLinkRepository) Create(ctx context.Context, userID string) (string, error) {
	code, err := generateCode()
	if err != nil {
		return "", apperror.NewInternalError("failed to generate code", err)
	}

	query := `
		INSERT INTO magic_links (code, user_id)
		VALUES ($1, $2)
		ON CONFLICT (code) DO UPDATE SET
			user_id = $2,
			created_at = NOW(),
			expires_at = NOW() + INTERVAL '24 hours',
			used_at = NULL
		RETURNING code
	`

	var resultCode string
	err = r.conn.DB.QueryRowContext(ctx, query, code, userID).Scan(&resultCode)
	if err != nil {
		return "", apperror.NewInternalError("failed to create magic link", err)
	}

	return resultCode, nil
}

// GetUserID retrieves the user ID for a valid (non-expired) code
func (r *MagicLinkRepository) GetUserID(ctx context.Context, code string) (string, error) {
	query := `
		SELECT user_id
		FROM magic_links
		WHERE code = $1
		  AND expires_at > NOW()
	`

	var userID string
	err := r.conn.DB.QueryRowContext(ctx, query, code).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", apperror.NewUnauthorizedError("invalid or expired code")
	}
	if err != nil {
		return "", apperror.NewInternalError("failed to lookup code", err)
	}

	// Mark as used (optional, for analytics)
	go func() {
		updateQuery := `UPDATE magic_links SET used_at = NOW() WHERE code = $1 AND used_at IS NULL`
		r.conn.DB.ExecContext(context.Background(), updateQuery, code)
	}()

	return userID, nil
}

// Cleanup removes expired magic links (call periodically)
func (r *MagicLinkRepository) Cleanup(ctx context.Context) (int64, error) {
	query := `DELETE FROM magic_links WHERE expires_at < NOW()`
	result, err := r.conn.DB.ExecContext(ctx, query)
	if err != nil {
		return 0, apperror.NewInternalError("failed to cleanup magic links", err)
	}
	return result.RowsAffected()
}

// MagicLink represents a magic link record
type MagicLink struct {
	Code      string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}
