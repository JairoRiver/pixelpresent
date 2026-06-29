package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Reaction is a recipient's response to a gift: an emoji, a short text, or a
// voice note. Kind selects which payload field carries the content; the others
// are nil (SQL NULL), mirroring the gifts nullable-pointer convention.
//
// VoiceStorageKey is only an opaque pointer into ObjectStorage (S3-compatible in
// prod, local filesystem in dev): the audio bytes never live in Postgres, and
// the upload flow is post-MVP. This repository just persists/returns the key.
type Reaction struct {
	ID              uuid.UUID
	GiftID          uuid.UUID
	Kind            string
	Emoji           *string
	Message         *string
	VoiceStorageKey *string
	CreatedAt       time.Time
}

// ReactionRepository persists and retrieves reactions. Create returns the stored
// row (with DB-generated id/created_at), mirroring the other repositories.
type ReactionRepository interface {
	Create(ctx context.Context, r Reaction) (Reaction, error)
	// ListByGift returns a gift's reactions oldest-first.
	ListByGift(ctx context.Context, giftID uuid.UUID) ([]Reaction, error)
}
