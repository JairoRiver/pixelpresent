// Package reactions orchestrates recipient reactions on a gift: it validates the
// payload and stores it only against a currently-visible gift addressed by its
// public view token.
package reactions

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/gifts"
)

// Supported reaction kinds for the public endpoint. Voice notes are post-MVP and
// are not accepted here (they need the upload flow over ObjectStorage).
const (
	KindEmoji = "emoji"
	KindText  = "text"
)

// Payload limits. This is a public, unauthenticated write endpoint, so a length
// cap is cheap abuse defense; rate-limiting is out of scope for now.
const (
	maxEmojiBytes   = 32  // room for ZWJ emoji sequences
	maxMessageRunes = 500 // a short note, not an essay
)

// CreateInput carries a recipient's reaction. ViewToken addresses the gift; Kind
// selects which content field is meaningful.
type CreateInput struct {
	ViewToken string
	Kind      string
	Emoji     string
	Message   string
}

// Service orchestrates reactions over the gift and reaction repositories.
type Service struct {
	gifts     domain.GiftRepository
	reactions domain.ReactionRepository
}

// NewService builds the reaction service.
func NewService(giftRepo domain.GiftRepository, reactionRepo domain.ReactionRepository) *Service {
	return &Service{gifts: giftRepo, reactions: reactionRepo}
}

// Create stores a reaction against the visible gift addressed by in.ViewToken.
// It returns domain.ErrGiftNotFound (unknown token), domain.ErrGiftNotVisible
// (the gift exists but is gated by the visibility rules), or
// domain.ErrReactionInvalid (malformed payload).
func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Reaction, error) {
	gift, err := s.gifts.GetByViewToken(ctx, in.ViewToken)
	if err != nil {
		return domain.Reaction{}, err
	}
	if gifts.CheckVisibility(gift, time.Now()) != gifts.Visible {
		return domain.Reaction{}, domain.ErrGiftNotVisible
	}

	reaction, err := buildReaction(gift.ID, in)
	if err != nil {
		return domain.Reaction{}, err
	}
	return s.reactions.Create(ctx, reaction)
}

// buildReaction validates the payload for its kind and returns the domain
// reaction to persist, or domain.ErrReactionInvalid.
func buildReaction(giftID uuid.UUID, in CreateInput) (domain.Reaction, error) {
	switch in.Kind {
	case KindEmoji:
		emoji := strings.TrimSpace(in.Emoji)
		if emoji == "" || len(emoji) > maxEmojiBytes {
			return domain.Reaction{}, domain.ErrReactionInvalid
		}
		return domain.Reaction{GiftID: giftID, Kind: KindEmoji, Emoji: &emoji}, nil
	case KindText:
		message := strings.TrimSpace(in.Message)
		if message == "" || utf8.RuneCountInString(message) > maxMessageRunes {
			return domain.Reaction{}, domain.ErrReactionInvalid
		}
		return domain.Reaction{GiftID: giftID, Kind: KindText, Message: &message}, nil
	default:
		return domain.Reaction{}, domain.ErrReactionInvalid
	}
}
