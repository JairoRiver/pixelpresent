package repository

import (
	"context"
	"errors"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/repository/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// emptyJSON is the value stored for a JSONB column when no config is provided;
// it honors the gifts.reveal_config DEFAULT '{}' even though sqlc always names
// the column in the INSERT (so the SQL default never fires).
var emptyJSON = []byte("{}")

// GiftRepo is a domain.GiftRepository backed by sqlc-generated queries.
type GiftRepo struct {
	q *sqlc.Queries
}

var _ domain.GiftRepository = (*GiftRepo)(nil)

// NewGiftRepo builds a GiftRepo over any sqlc.DBTX (a *pgxpool.Pool in
// production, a pgx.Tx in tests).
func NewGiftRepo(db sqlc.DBTX) *GiftRepo {
	return &GiftRepo{q: sqlc.New(db)}
}

func (r *GiftRepo) Create(ctx context.Context, g domain.Gift) (domain.Gift, error) {
	created, err := r.q.CreateGift(ctx, sqlc.CreateGiftParams{
		CreatorID:       g.CreatorID,
		Title:           g.Title,
		Message:         g.Message,
		PixelArt:        g.PixelArt,
		RevealType:      g.RevealType,
		RevealConfig:    orEmptyJSON(g.RevealConfig),
		ViewToken:       g.ViewToken,
		RecipientEmail:  nullText(g.RecipientEmail),
		ScheduledOpenAt: nullTimestamptz(g.ScheduledOpenAt),
		ScheduledSendAt: nullTimestamptz(g.ScheduledSendAt),
		SingleOpen:      g.SingleOpen,
		ExpiresAt:       nullTimestamptz(g.ExpiresAt),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Gift{}, domain.ErrDuplicateViewToken
		}
		return domain.Gift{}, err
	}
	return toDomainGift(created), nil
}

func (r *GiftRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Gift, error) {
	g, err := r.q.GetGiftByID(ctx, id)
	return mapGetGift(g, err)
}

func (r *GiftRepo) GetByViewToken(ctx context.Context, token string) (domain.Gift, error) {
	g, err := r.q.GetGiftByViewToken(ctx, token)
	return mapGetGift(g, err)
}

func (r *GiftRepo) Update(ctx context.Context, g domain.Gift) (domain.Gift, error) {
	updated, err := r.q.UpdateGift(ctx, sqlc.UpdateGiftParams{
		ID:              g.ID,
		Title:           g.Title,
		Message:         g.Message,
		PixelArt:        g.PixelArt,
		RevealType:      g.RevealType,
		RevealConfig:    orEmptyJSON(g.RevealConfig),
		RecipientEmail:  nullText(g.RecipientEmail),
		ScheduledOpenAt: nullTimestamptz(g.ScheduledOpenAt),
		ScheduledSendAt: nullTimestamptz(g.ScheduledSendAt),
		SingleOpen:      g.SingleOpen,
		ExpiresAt:       nullTimestamptz(g.ExpiresAt),
	})
	return mapGetGift(updated, err)
}

func (r *GiftRepo) Delete(ctx context.Context, id uuid.UUID) error {
	rows, err := r.q.DeleteGift(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrGiftNotFound
	}
	return nil
}

func (r *GiftRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Gift, error) {
	rows, err := r.q.ListGiftsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	gifts := make([]domain.Gift, len(rows))
	for i, row := range rows {
		gifts[i] = toDomainGift(row)
	}
	return gifts, nil
}

func (r *GiftRepo) ViewTokenExists(ctx context.Context, token string) (bool, error) {
	return r.q.GiftViewTokenExists(ctx, token)
}

func mapGetGift(g sqlc.Gift, err error) (domain.Gift, error) {
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Gift{}, domain.ErrGiftNotFound
		}
		return domain.Gift{}, err
	}
	return toDomainGift(g), nil
}

func toDomainGift(g sqlc.Gift) domain.Gift {
	return domain.Gift{
		ID:              g.ID,
		CreatorID:       g.CreatorID,
		Title:           g.Title,
		Message:         g.Message,
		PixelArt:        g.PixelArt,
		RevealType:      g.RevealType,
		RevealConfig:    g.RevealConfig,
		ViewToken:       g.ViewToken,
		RecipientEmail:  textPtr(g.RecipientEmail),
		ScheduledOpenAt: timestamptzPtr(g.ScheduledOpenAt),
		ScheduledSendAt: timestamptzPtr(g.ScheduledSendAt),
		SentAt:          timestamptzPtr(g.SentAt),
		SingleOpen:      g.SingleOpen,
		OpenedAt:        timestamptzPtr(g.OpenedAt),
		ExpiresAt:       timestamptzPtr(g.ExpiresAt),
		CreatedAt:       g.CreatedAt,
		UpdatedAt:       g.UpdatedAt,
	}
}

func orEmptyJSON(b []byte) []byte {
	if len(b) == 0 {
		return emptyJSON
	}
	return b
}

// --- nullable conversions between domain pointers and pgtype values ---

func nullText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func nullTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
