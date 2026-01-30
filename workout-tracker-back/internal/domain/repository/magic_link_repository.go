package repository

import (
	"context"
)

// MagicLinkInvalidator defines operations to invalidate magic links
type MagicLinkInvalidator interface {
	// InvalidateByUser marks all active magic links for a user as used
	InvalidateByUser(ctx context.Context, userID string) error
}
