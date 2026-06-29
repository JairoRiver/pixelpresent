package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/repository/db/dbtest"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// newTestGift returns a domain.Gift with random data for creatorID. Optional
// fields are left nil so individual tests can populate what they need.
func newTestGift(creatorID uuid.UUID) domain.Gift {
	return domain.Gift{
		CreatorID:    creatorID,
		Title:        util.RandomString(10),
		Message:      util.RandomString(20),
		PixelArt:     json.RawMessage(`{"width":2,"height":2,"palette":["#000000","#ffffff"],"pixels":[0,1,1,0]}`),
		RevealType:   "box",
		RevealConfig: json.RawMessage(`{}`),
		ViewToken:    util.RandomString(43),
	}
}

// createTestGift creates a user and a gift for it on the same transaction.
func createTestGift(t *testing.T, tx pgx.Tx) domain.Gift {
	t.Helper()

	user, err := NewUserRepo(tx).Create(context.Background(), util.RandomEmail())
	require.NoError(t, err)

	gift, err := NewGiftRepo(tx).Create(context.Background(), newTestGift(user.ID))
	require.NoError(t, err)
	return gift
}

func TestGiftRepo_Create(t *testing.T) {
	tx := dbtest.Tx(t)
	user, err := NewUserRepo(tx).Create(context.Background(), util.RandomEmail())
	require.NoError(t, err)

	repo := NewGiftRepo(tx)
	in := newTestGift(user.ID)
	email := util.RandomEmail()
	openAt := time.Now().Add(24 * time.Hour)
	in.RecipientEmail = &email
	in.ScheduledOpenAt = &openAt
	in.SingleOpen = true

	gift, err := repo.Create(context.Background(), in)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, gift.ID)
	require.Equal(t, user.ID, gift.CreatorID)
	require.Equal(t, in.Title, gift.Title)
	require.JSONEq(t, string(in.PixelArt), string(gift.PixelArt))
	require.Equal(t, in.ViewToken, gift.ViewToken)
	require.True(t, gift.SingleOpen)
	require.NotNil(t, gift.RecipientEmail)
	require.Equal(t, email, *gift.RecipientEmail)
	require.NotNil(t, gift.ScheduledOpenAt)
	require.WithinDuration(t, openAt, *gift.ScheduledOpenAt, time.Second)
	// Unset optional fields stay nil.
	require.Nil(t, gift.SentAt)
	require.Nil(t, gift.OpenedAt)
	require.Nil(t, gift.ExpiresAt)
	require.False(t, gift.CreatedAt.IsZero())
}

func TestGiftRepo_Create_DefaultsRevealConfig(t *testing.T) {
	tx := dbtest.Tx(t)
	user, err := NewUserRepo(tx).Create(context.Background(), util.RandomEmail())
	require.NoError(t, err)

	in := newTestGift(user.ID)
	in.RevealConfig = nil // repo must default it to {} (NOT NULL column)

	gift, err := NewGiftRepo(tx).Create(context.Background(), in)
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(gift.RevealConfig))
}

func TestGiftRepo_GetByID(t *testing.T) {
	tx := dbtest.Tx(t)
	created := createTestGift(t, tx)

	got, err := NewGiftRepo(tx).GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.ViewToken, got.ViewToken)
}

func TestGiftRepo_GetByID_NotFound(t *testing.T) {
	tx := dbtest.Tx(t)

	_, err := NewGiftRepo(tx).GetByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, domain.ErrGiftNotFound)
}

func TestGiftRepo_GetByViewToken(t *testing.T) {
	tx := dbtest.Tx(t)
	created := createTestGift(t, tx)

	got, err := NewGiftRepo(tx).GetByViewToken(context.Background(), created.ViewToken)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
}

func TestGiftRepo_GetByViewToken_NotFound(t *testing.T) {
	tx := dbtest.Tx(t)

	_, err := NewGiftRepo(tx).GetByViewToken(context.Background(), util.RandomString(43))
	require.ErrorIs(t, err, domain.ErrGiftNotFound)
}

func TestGiftRepo_Update(t *testing.T) {
	tx := dbtest.Tx(t)
	repo := NewGiftRepo(tx)
	created := createTestGift(t, tx)

	// Populate nullable fields.
	email := util.RandomEmail()
	openAt := time.Now().Add(48 * time.Hour)
	created.Title = "Actualizado"
	created.RevealType = "scratch"
	created.RecipientEmail = &email
	created.ScheduledOpenAt = &openAt
	created.SingleOpen = true

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.Equal(t, "Actualizado", updated.Title)
	require.Equal(t, "scratch", updated.RevealType)
	require.NotNil(t, updated.RecipientEmail)
	require.Equal(t, email, *updated.RecipientEmail)
	require.True(t, updated.SingleOpen)
	require.Equal(t, created.ViewToken, updated.ViewToken, "view_token immutable")

	// Clear nullables via full-replace.
	updated.RecipientEmail = nil
	updated.ScheduledOpenAt = nil
	cleared, err := repo.Update(context.Background(), updated)
	require.NoError(t, err)
	require.Nil(t, cleared.RecipientEmail)
	require.Nil(t, cleared.ScheduledOpenAt)
}

func TestGiftRepo_Update_NotFound(t *testing.T) {
	tx := dbtest.Tx(t)

	ghost := newTestGift(uuid.New())
	ghost.ID = uuid.New()
	_, err := NewGiftRepo(tx).Update(context.Background(), ghost)
	require.ErrorIs(t, err, domain.ErrGiftNotFound)
}

func TestGiftRepo_ListByUser(t *testing.T) {
	tx := dbtest.Tx(t)
	repo := NewGiftRepo(tx)

	user, err := NewUserRepo(tx).Create(context.Background(), util.RandomEmail())
	require.NoError(t, err)
	other, err := NewUserRepo(tx).Create(context.Background(), util.RandomEmail())
	require.NoError(t, err)

	g1, err := repo.Create(context.Background(), newTestGift(user.ID))
	require.NoError(t, err)
	g2, err := repo.Create(context.Background(), newTestGift(user.ID))
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), newTestGift(other.ID))
	require.NoError(t, err)

	gifts, err := repo.ListByUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, gifts, 2)

	ids := []uuid.UUID{gifts[0].ID, gifts[1].ID}
	require.Contains(t, ids, g1.ID)
	require.Contains(t, ids, g2.ID)
}

func TestGiftRepo_ListByUser_Empty(t *testing.T) {
	tx := dbtest.Tx(t)
	user, err := NewUserRepo(tx).Create(context.Background(), util.RandomEmail())
	require.NoError(t, err)

	gifts, err := NewGiftRepo(tx).ListByUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Empty(t, gifts)
}

func TestGiftRepo_Delete(t *testing.T) {
	tx := dbtest.Tx(t)
	repo := NewGiftRepo(tx)
	created := createTestGift(t, tx)

	require.NoError(t, repo.Delete(context.Background(), created.ID))

	_, err := repo.GetByID(context.Background(), created.ID)
	require.ErrorIs(t, err, domain.ErrGiftNotFound)
}

func TestGiftRepo_Delete_NotFound(t *testing.T) {
	tx := dbtest.Tx(t)

	err := NewGiftRepo(tx).Delete(context.Background(), uuid.New())
	require.ErrorIs(t, err, domain.ErrGiftNotFound)
}

func TestGiftRepo_ViewTokenExists(t *testing.T) {
	tx := dbtest.Tx(t)
	created := createTestGift(t, tx)

	repo := NewGiftRepo(tx)
	exists, err := repo.ViewTokenExists(context.Background(), created.ViewToken)
	require.NoError(t, err)
	require.True(t, exists)

	free, err := repo.ViewTokenExists(context.Background(), util.RandomString(43))
	require.NoError(t, err)
	require.False(t, free)
}
