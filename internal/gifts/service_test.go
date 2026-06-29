package gifts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// fakeGiftRepo is an in-memory domain.GiftRepository for unit tests.
type fakeGiftRepo struct {
	byID   map[uuid.UUID]domain.Gift
	tokens map[string]bool
}

var _ domain.GiftRepository = (*fakeGiftRepo)(nil)

func newFakeGiftRepo() *fakeGiftRepo {
	return &fakeGiftRepo{
		byID:   make(map[uuid.UUID]domain.Gift),
		tokens: make(map[string]bool),
	}
}

func (f *fakeGiftRepo) Create(_ context.Context, g domain.Gift) (domain.Gift, error) {
	g.ID = uuid.New()
	f.byID[g.ID] = g
	f.tokens[g.ViewToken] = true
	return g, nil
}

func (f *fakeGiftRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Gift, error) {
	g, ok := f.byID[id]
	if !ok {
		return domain.Gift{}, domain.ErrGiftNotFound
	}
	return g, nil
}

func (f *fakeGiftRepo) GetByViewToken(_ context.Context, token string) (domain.Gift, error) {
	for _, g := range f.byID {
		if g.ViewToken == token {
			return g, nil
		}
	}
	return domain.Gift{}, domain.ErrGiftNotFound
}

func (f *fakeGiftRepo) Update(_ context.Context, g domain.Gift) (domain.Gift, error) {
	if _, ok := f.byID[g.ID]; !ok {
		return domain.Gift{}, domain.ErrGiftNotFound
	}
	f.byID[g.ID] = g
	return g, nil
}

func (f *fakeGiftRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]domain.Gift, error) {
	out := []domain.Gift{}
	for _, g := range f.byID {
		if g.CreatorID == userID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (f *fakeGiftRepo) ViewTokenExists(_ context.Context, token string) (bool, error) {
	return f.tokens[token], nil
}

func TestService_Create(t *testing.T) {
	repo := newFakeGiftRepo()
	svc := NewService(repo)
	creator := uuid.New()

	gift, err := svc.Create(context.Background(), CreateInput{
		CreatorID:  creator,
		Title:      "Hola",
		PixelArt:   json.RawMessage(`{"width":1,"height":1,"palette":["#000"],"pixels":[0]}`),
		RevealType: "box",
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, gift.ID)
	require.Equal(t, creator, gift.CreatorID)
	require.Len(t, gift.ViewToken, 43, "a view token was generated")

	// It was persisted and is retrievable.
	stored, err := repo.GetByID(context.Background(), gift.ID)
	require.NoError(t, err)
	require.Equal(t, gift.ViewToken, stored.ViewToken)
}

func TestService_GetOwned_Own(t *testing.T) {
	repo := newFakeGiftRepo()
	svc := NewService(repo)
	creator := uuid.New()

	created, err := svc.Create(context.Background(), CreateInput{
		CreatorID:  creator,
		Title:      "Mío",
		PixelArt:   json.RawMessage(`{}`),
		RevealType: "box",
	})
	require.NoError(t, err)

	got, err := svc.GetOwned(context.Background(), created.ID, creator)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
}

func TestService_GetOwned_Foreign(t *testing.T) {
	repo := newFakeGiftRepo()
	svc := NewService(repo)

	created, err := svc.Create(context.Background(), CreateInput{
		CreatorID:  uuid.New(),
		Title:      "Ajeno",
		PixelArt:   json.RawMessage(`{}`),
		RevealType: "box",
	})
	require.NoError(t, err)

	_, err = svc.GetOwned(context.Background(), created.ID, uuid.New())
	require.ErrorIs(t, err, domain.ErrGiftForbidden)
}

func TestService_GetOwned_NotFound(t *testing.T) {
	svc := NewService(newFakeGiftRepo())

	_, err := svc.GetOwned(context.Background(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, domain.ErrGiftNotFound)
}
