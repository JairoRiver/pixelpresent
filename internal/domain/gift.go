package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Gift is a pixel-art present created by a user and opened by a recipient who
// holds its view token. JSONB columns are kept as raw JSON; nullable columns are
// pointers (nil = SQL NULL).
type Gift struct {
	ID              uuid.UUID
	CreatorID       uuid.UUID
	Title           string
	Message         string
	PixelArt        json.RawMessage // { width, height, palette: [...], pixels: [...] }
	RevealType      string
	RevealConfig    json.RawMessage
	ViewToken       string
	RecipientEmail  *string
	ScheduledOpenAt *time.Time
	ScheduledSendAt *time.Time
	SentAt          *time.Time
	SingleOpen      bool
	OpenedAt        *time.Time
	ExpiresAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// GiftRepository persists and retrieves gifts. Create and Update return the
// stored row (with DB-generated id/timestamps), mirroring the other repositories.
type GiftRepository interface {
	Create(ctx context.Context, g Gift) (Gift, error)
	GetByID(ctx context.Context, id uuid.UUID) (Gift, error)
	GetByViewToken(ctx context.Context, token string) (Gift, error)
	Update(ctx context.Context, g Gift) (Gift, error)
	// Delete removes the gift with id (media and reactions cascade). It returns
	// ErrGiftNotFound if no gift had that id.
	Delete(ctx context.Context, id uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Gift, error)
	// ViewTokenExists reports whether a view token is already taken; it backs the
	// uniqueness check of the view-token generator (gifts.ViewTokenChecker).
	ViewTokenExists(ctx context.Context, token string) (bool, error)
}
