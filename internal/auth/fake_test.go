package auth

import (
	"context"
	"strings"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/google/uuid"
)

// fakeUserRepo is an in-memory domain.UserRepository for unit tests. Emails are
// matched case-insensitively to mimic the citext column.
type fakeUserRepo struct {
	byID    map[uuid.UUID]domain.User
	byEmail map[string]domain.User
}

var _ domain.UserRepository = (*fakeUserRepo)(nil)

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:    make(map[uuid.UUID]domain.User),
		byEmail: make(map[string]domain.User),
	}
}

func emailKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (f *fakeUserRepo) Create(_ context.Context, email string) (domain.User, error) {
	key := emailKey(email)
	if _, ok := f.byEmail[key]; ok {
		return domain.User{}, domain.ErrDuplicateEmail
	}
	u := domain.User{ID: uuid.New(), Email: email, CreatedAt: time.Now()}
	f.byID[u.ID] = u
	f.byEmail[key] = u
	return u, nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (domain.User, error) {
	u, ok := f.byEmail[emailKey(email)]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

// fakeMagicLinkRepo is an in-memory domain.MagicLinkRepository for unit tests.
type fakeMagicLinkRepo struct {
	byID   map[uuid.UUID]domain.MagicLink
	byHash map[string]domain.MagicLink
}

var _ domain.MagicLinkRepository = (*fakeMagicLinkRepo)(nil)

func newFakeMagicLinkRepo() *fakeMagicLinkRepo {
	return &fakeMagicLinkRepo{
		byID:   make(map[uuid.UUID]domain.MagicLink),
		byHash: make(map[string]domain.MagicLink),
	}
}

func (f *fakeMagicLinkRepo) Create(_ context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (domain.MagicLink, error) {
	ml := domain.MagicLink{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	f.byID[ml.ID] = ml
	f.byHash[tokenHash] = ml
	return ml, nil
}

func (f *fakeMagicLinkRepo) GetByTokenHash(_ context.Context, tokenHash string) (domain.MagicLink, error) {
	ml, ok := f.byHash[tokenHash]
	if !ok {
		return domain.MagicLink{}, domain.ErrMagicLinkNotFound
	}
	return ml, nil
}

func (f *fakeMagicLinkRepo) MarkConsumed(_ context.Context, id uuid.UUID) (domain.MagicLink, error) {
	ml, ok := f.byID[id]
	if !ok || ml.ConsumedAt != nil {
		// Missing or already consumed: single-use guard, like the SQL query.
		return domain.MagicLink{}, domain.ErrMagicLinkNotFound
	}
	now := time.Now()
	ml.ConsumedAt = &now
	f.byID[id] = ml
	f.byHash[ml.TokenHash] = ml
	return ml, nil
}

func (f *fakeMagicLinkRepo) all() []domain.MagicLink {
	links := make([]domain.MagicLink, 0, len(f.byID))
	for _, ml := range f.byID {
		links = append(links, ml)
	}
	return links
}
