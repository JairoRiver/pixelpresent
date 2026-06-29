package gifts

import (
	"context"
	"encoding/json"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/google/uuid"
)

// revealTypes is the set of accepted reveal mechanics. It is the single source
// of truth for validating reveal_type (the column is free text in the schema).
var revealTypes = map[string]struct{}{
	"box":      {},
	"envelope": {},
	"scratch":  {},
	"puzzle":   {},
	"confetti": {},
	"cake":     {},
	"arcade":   {},
}

// ValidRevealType reports whether t is one of the supported reveal mechanics.
func ValidRevealType(t string) bool {
	_, ok := revealTypes[t]
	return ok
}

// CreateInput carries the creator-provided fields of a new gift. The view token
// and DB-generated fields (id, timestamps) are not part of it.
type CreateInput struct {
	CreatorID       uuid.UUID
	Title           string
	Message         string
	PixelArt        json.RawMessage
	RevealType      string
	RevealConfig    json.RawMessage
	RecipientEmail  *string
	ScheduledOpenAt *time.Time
	ScheduledSendAt *time.Time
	SingleOpen      bool
	ExpiresAt       *time.Time
}

// Service orchestrates gift operations over the gift repository.
type Service struct {
	repo domain.GiftRepository
}

// NewService builds the gift service.
func NewService(repo domain.GiftRepository) *Service {
	return &Service{repo: repo}
}

// Create generates a unique view token and persists a new gift owned by
// in.CreatorID, returning the stored row.
func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Gift, error) {
	token, err := GenerateViewToken(ctx, s.repo)
	if err != nil {
		return domain.Gift{}, err
	}

	return s.repo.Create(ctx, domain.Gift{
		CreatorID:       in.CreatorID,
		Title:           in.Title,
		Message:         in.Message,
		PixelArt:        in.PixelArt,
		RevealType:      in.RevealType,
		RevealConfig:    in.RevealConfig,
		ViewToken:       token,
		RecipientEmail:  in.RecipientEmail,
		ScheduledOpenAt: in.ScheduledOpenAt,
		ScheduledSendAt: in.ScheduledSendAt,
		SingleOpen:      in.SingleOpen,
		ExpiresAt:       in.ExpiresAt,
	})
}

// GetOwned returns the gift with id only if ownerID is its creator. It returns
// domain.ErrGiftNotFound if the gift does not exist and domain.ErrGiftForbidden
// if it exists but belongs to someone else.
func (s *Service) GetOwned(ctx context.Context, id, ownerID uuid.UUID) (domain.Gift, error) {
	gift, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Gift{}, err
	}
	if gift.CreatorID != ownerID {
		return domain.Gift{}, domain.ErrGiftForbidden
	}
	return gift, nil
}
