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

// UpdateInput carries the creator-editable fields of a gift. id, creator,
// view_token and the lifecycle timestamps (sent_at, opened_at, created_at) are
// not editable.
type UpdateInput struct {
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

// UpdateOwned full-replaces the editable fields of the gift with id, only if
// ownerID is its creator. Same error contract as GetOwned.
func (s *Service) UpdateOwned(ctx context.Context, id, ownerID uuid.UUID, in UpdateInput) (domain.Gift, error) {
	gift, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Gift{}, err
	}
	if gift.CreatorID != ownerID {
		return domain.Gift{}, domain.ErrGiftForbidden
	}

	gift.Title = in.Title
	gift.Message = in.Message
	gift.PixelArt = in.PixelArt
	gift.RevealType = in.RevealType
	gift.RevealConfig = in.RevealConfig
	gift.RecipientEmail = in.RecipientEmail
	gift.ScheduledOpenAt = in.ScheduledOpenAt
	gift.ScheduledSendAt = in.ScheduledSendAt
	gift.SingleOpen = in.SingleOpen
	gift.ExpiresAt = in.ExpiresAt

	return s.repo.Update(ctx, gift)
}

// Publish marks the gift with id as published, making it reachable at its public
// view token, only if ownerID is its creator. It is idempotent: a gift that is
// already published keeps its original published_at and is returned unchanged.
// Same error contract as GetOwned.
func (s *Service) Publish(ctx context.Context, id, ownerID uuid.UUID) (domain.Gift, error) {
	gift, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Gift{}, err
	}
	if gift.CreatorID != ownerID {
		return domain.Gift{}, domain.ErrGiftForbidden
	}
	if gift.PublishedAt != nil {
		return gift, nil
	}
	now := time.Now()
	gift.PublishedAt = &now
	return s.repo.Update(ctx, gift)
}

// ListByOwner returns every gift created by ownerID (newest first), for the
// creator's dashboard.
func (s *Service) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Gift, error) {
	return s.repo.ListByUser(ctx, ownerID)
}

// GetByViewToken returns the gift addressed by a public view token, or
// domain.ErrGiftNotFound. The caller applies the visibility gate (CheckVisibility).
func (s *Service) GetByViewToken(ctx context.Context, token string) (domain.Gift, error) {
	return s.repo.GetByViewToken(ctx, token)
}

// MarkOpened records that the gift addressed by the public view token has been
// opened — the reveal frontend calls it when the reveal animation completes. It
// sets opened_at the first time only; a second call is an idempotent no-op that
// leaves the original timestamp untouched (this is what flips a single_open gift
// to AlreadyOpened on the next view). Returns domain.ErrGiftNotFound for an
// unknown token.
func (s *Service) MarkOpened(ctx context.Context, token string) error {
	// Existence check first so an unknown token is a 404, distinct from an
	// already-opened gift (both of which make the atomic UPDATE match no row).
	if _, err := s.repo.GetByViewToken(ctx, token); err != nil {
		return err
	}
	_, err := s.repo.MarkOpened(ctx, token)
	return err
}

// DeleteOwned removes the gift with id, only if ownerID is its creator. Same
// error contract as GetOwned.
func (s *Service) DeleteOwned(ctx context.Context, id, ownerID uuid.UUID) error {
	gift, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if gift.CreatorID != ownerID {
		return domain.ErrGiftForbidden
	}
	return s.repo.Delete(ctx, id)
}
