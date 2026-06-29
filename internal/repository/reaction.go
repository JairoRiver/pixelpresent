package repository

import (
	"context"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/repository/db/sqlc"
	"github.com/google/uuid"
)

// ReactionRepo is a domain.ReactionRepository backed by sqlc-generated queries.
type ReactionRepo struct {
	q *sqlc.Queries
}

var _ domain.ReactionRepository = (*ReactionRepo)(nil)

// NewReactionRepo builds a ReactionRepo over any sqlc.DBTX (a *pgxpool.Pool in
// production, a pgx.Tx in tests).
func NewReactionRepo(db sqlc.DBTX) *ReactionRepo {
	return &ReactionRepo{q: sqlc.New(db)}
}

func (r *ReactionRepo) Create(ctx context.Context, in domain.Reaction) (domain.Reaction, error) {
	created, err := r.q.CreateReaction(ctx, sqlc.CreateReactionParams{
		GiftID:          in.GiftID,
		Kind:            in.Kind,
		Emoji:           nullText(in.Emoji),
		Message:         nullText(in.Message),
		VoiceStorageKey: nullText(in.VoiceStorageKey),
	})
	if err != nil {
		return domain.Reaction{}, err
	}
	return toDomainReaction(created), nil
}

func (r *ReactionRepo) ListByGift(ctx context.Context, giftID uuid.UUID) ([]domain.Reaction, error) {
	rows, err := r.q.ListReactionsByGift(ctx, giftID)
	if err != nil {
		return nil, err
	}
	reactions := make([]domain.Reaction, len(rows))
	for i, row := range rows {
		reactions[i] = toDomainReaction(row)
	}
	return reactions, nil
}

func toDomainReaction(r sqlc.Reaction) domain.Reaction {
	return domain.Reaction{
		ID:              r.ID,
		GiftID:          r.GiftID,
		Kind:            r.Kind,
		Emoji:           textPtr(r.Emoji),
		Message:         textPtr(r.Message),
		VoiceStorageKey: textPtr(r.VoiceStorageKey),
		CreatedAt:       r.CreatedAt,
	}
}
