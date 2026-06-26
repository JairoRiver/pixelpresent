package repository

import (
	"context"
	"errors"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/repository/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MagicLinkRepo is a domain.MagicLinkRepository backed by sqlc-generated queries.
type MagicLinkRepo struct {
	q *sqlc.Queries
}

var _ domain.MagicLinkRepository = (*MagicLinkRepo)(nil)

// NewMagicLinkRepo builds a MagicLinkRepo over any sqlc.DBTX (a *pgxpool.Pool in
// production, a pgx.Tx in tests).
func NewMagicLinkRepo(db sqlc.DBTX) *MagicLinkRepo {
	return &MagicLinkRepo{q: sqlc.New(db)}
}

func (r *MagicLinkRepo) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (domain.MagicLink, error) {
	m, err := r.q.CreateMagicLink(ctx, sqlc.CreateMagicLinkParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return domain.MagicLink{}, err
	}
	return toDomainMagicLink(m), nil
}

func (r *MagicLinkRepo) GetByTokenHash(ctx context.Context, tokenHash string) (domain.MagicLink, error) {
	m, err := r.q.GetMagicLinkByTokenHash(ctx, tokenHash)
	return mapGetMagicLink(m, err)
}

func (r *MagicLinkRepo) MarkConsumed(ctx context.Context, id uuid.UUID) (domain.MagicLink, error) {
	m, err := r.q.MarkMagicLinkConsumed(ctx, id)
	return mapGetMagicLink(m, err)
}

func mapGetMagicLink(m sqlc.MagicLink, err error) (domain.MagicLink, error) {
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.MagicLink{}, domain.ErrMagicLinkNotFound
		}
		return domain.MagicLink{}, err
	}
	return toDomainMagicLink(m), nil
}

func toDomainMagicLink(m sqlc.MagicLink) domain.MagicLink {
	var consumedAt *time.Time
	if m.ConsumedAt.Valid {
		t := m.ConsumedAt.Time
		consumedAt = &t
	}
	return domain.MagicLink{
		ID:         m.ID,
		UserID:     m.UserID,
		TokenHash:  m.TokenHash,
		ExpiresAt:  m.ExpiresAt,
		ConsumedAt: consumedAt,
		CreatedAt:  m.CreatedAt,
	}
}
