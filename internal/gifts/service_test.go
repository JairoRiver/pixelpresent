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

func (f *fakeGiftRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrGiftNotFound
	}
	delete(f.byID, id)
	return nil
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

func TestService_UpdateOwned_Own(t *testing.T) {
	repo := newFakeGiftRepo()
	svc := NewService(repo)
	creator := uuid.New()

	created, err := svc.Create(context.Background(), CreateInput{
		CreatorID:  creator,
		Title:      "Original",
		PixelArt:   json.RawMessage(`{}`),
		RevealType: "box",
	})
	require.NoError(t, err)

	updated, err := svc.UpdateOwned(context.Background(), created.ID, creator, UpdateInput{
		Title:      "Editado",
		RevealType: "scratch",
		PixelArt:   json.RawMessage(`{"v":1}`),
		SingleOpen: true,
	})
	require.NoError(t, err)
	require.Equal(t, "Editado", updated.Title)
	require.Equal(t, "scratch", updated.RevealType)
	require.True(t, updated.SingleOpen)
	// Immutable fields survive the edit.
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, creator, updated.CreatorID)
	require.Equal(t, created.ViewToken, updated.ViewToken)
}

func TestService_UpdateOwned_Foreign(t *testing.T) {
	repo := newFakeGiftRepo()
	svc := NewService(repo)

	created, err := svc.Create(context.Background(), CreateInput{
		CreatorID:  uuid.New(),
		Title:      "Ajeno",
		PixelArt:   json.RawMessage(`{}`),
		RevealType: "box",
	})
	require.NoError(t, err)

	_, err = svc.UpdateOwned(context.Background(), created.ID, uuid.New(), UpdateInput{
		Title:      "Hackeado",
		RevealType: "box",
		PixelArt:   json.RawMessage(`{}`),
	})
	require.ErrorIs(t, err, domain.ErrGiftForbidden)

	// The gift was not modified.
	unchanged, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "Ajeno", unchanged.Title)
}

func TestService_UpdateOwned_NotFound(t *testing.T) {
	svc := NewService(newFakeGiftRepo())

	_, err := svc.UpdateOwned(context.Background(), uuid.New(), uuid.New(), UpdateInput{
		Title:      "x",
		RevealType: "box",
		PixelArt:   json.RawMessage(`{}`),
	})
	require.ErrorIs(t, err, domain.ErrGiftNotFound)
}

func TestService_DeleteOwned_Own(t *testing.T) {
	repo := newFakeGiftRepo()
	svc := NewService(repo)
	creator := uuid.New()

	created, err := svc.Create(context.Background(), CreateInput{
		CreatorID:  creator,
		Title:      "Borrable",
		PixelArt:   json.RawMessage(`{}`),
		RevealType: "box",
	})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteOwned(context.Background(), created.ID, creator))

	_, err = repo.GetByID(context.Background(), created.ID)
	require.ErrorIs(t, err, domain.ErrGiftNotFound, "the gift is gone")
}

func TestService_DeleteOwned_Foreign(t *testing.T) {
	repo := newFakeGiftRepo()
	svc := NewService(repo)

	created, err := svc.Create(context.Background(), CreateInput{
		CreatorID:  uuid.New(),
		Title:      "Ajeno",
		PixelArt:   json.RawMessage(`{}`),
		RevealType: "box",
	})
	require.NoError(t, err)

	err = svc.DeleteOwned(context.Background(), created.ID, uuid.New())
	require.ErrorIs(t, err, domain.ErrGiftForbidden)

	_, err = repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err, "a foreign delete must not remove the gift")
}

func TestService_DeleteOwned_NotFound(t *testing.T) {
	svc := NewService(newFakeGiftRepo())

	err := svc.DeleteOwned(context.Background(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, domain.ErrGiftNotFound)
}

func TestService_GetByViewToken(t *testing.T) {
	repo := newFakeGiftRepo()
	svc := NewService(repo)

	created, err := svc.Create(context.Background(), CreateInput{
		CreatorID:  uuid.New(),
		Title:      "Público",
		PixelArt:   json.RawMessage(`{}`),
		RevealType: "box",
	})
	require.NoError(t, err)

	got, err := svc.GetByViewToken(context.Background(), created.ViewToken)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)

	_, err = svc.GetByViewToken(context.Background(), "no-such-token")
	require.ErrorIs(t, err, domain.ErrGiftNotFound)
}

func TestService_ListByOwner(t *testing.T) {
	repo := newFakeGiftRepo()
	svc := NewService(repo)
	owner := uuid.New()
	other := uuid.New()

	base := CreateInput{PixelArt: json.RawMessage(`{}`), RevealType: "box"}
	mine1 := base
	mine1.CreatorID = owner
	mine1.Title = "A"
	mine2 := base
	mine2.CreatorID = owner
	mine2.Title = "B"
	foreign := base
	foreign.CreatorID = other
	foreign.Title = "C"
	for _, in := range []CreateInput{mine1, mine2, foreign} {
		_, err := svc.Create(context.Background(), in)
		require.NoError(t, err)
	}

	list, err := svc.ListByOwner(context.Background(), owner)
	require.NoError(t, err)
	require.Len(t, list, 2, "only the owner's gifts")
	for _, g := range list {
		require.Equal(t, owner, g.CreatorID)
	}
}
