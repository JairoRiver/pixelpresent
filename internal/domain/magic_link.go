package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MagicLink is a single-use, time-limited authentication token. Only the hash
// of the token is stored, never the token itself.
type MagicLink struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

// MagicLinkRepository persists and retrieves magic links.
type MagicLinkRepository interface {
	Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (MagicLink, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (MagicLink, error)
	MarkConsumed(ctx context.Context, id uuid.UUID) (MagicLink, error)
}
