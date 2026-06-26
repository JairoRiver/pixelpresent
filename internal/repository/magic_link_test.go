package repository

import (
	"context"
	"testing"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/repository/db/dbtest"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// createTestMagicLink creates a user and a magic link for it on the same
// transaction, returning the link.
func createTestMagicLink(t *testing.T, tx pgx.Tx) domain.MagicLink {
	t.Helper()

	user, err := NewUserRepo(tx).Create(context.Background(), util.RandomEmail())
	require.NoError(t, err)

	link, err := NewMagicLinkRepo(tx).Create(
		context.Background(), user.ID, util.RandomString(64), time.Now().Add(15*time.Minute),
	)
	require.NoError(t, err)

	return link
}

func TestMagicLinkRepo_Create(t *testing.T) {
	tx := dbtest.Tx(t)
	user, err := NewUserRepo(tx).Create(context.Background(), util.RandomEmail())
	require.NoError(t, err)

	repo := NewMagicLinkRepo(tx)
	tokenHash := util.RandomString(64)
	expiresAt := time.Now().Add(15 * time.Minute)

	link, err := repo.Create(context.Background(), user.ID, tokenHash, expiresAt)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, link.ID)
	require.Equal(t, user.ID, link.UserID)
	require.Equal(t, tokenHash, link.TokenHash)
	require.WithinDuration(t, expiresAt, link.ExpiresAt, time.Second)
	require.Nil(t, link.ConsumedAt)
	require.False(t, link.CreatedAt.IsZero())
}

func TestMagicLinkRepo_GetByTokenHash(t *testing.T) {
	tx := dbtest.Tx(t)
	created := createTestMagicLink(t, tx)

	repo := NewMagicLinkRepo(tx)
	got, err := repo.GetByTokenHash(context.Background(), created.TokenHash)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.UserID, got.UserID)
}

func TestMagicLinkRepo_GetByTokenHashNotFound(t *testing.T) {
	repo := NewMagicLinkRepo(dbtest.Tx(t))

	_, err := repo.GetByTokenHash(context.Background(), util.RandomString(64))
	require.ErrorIs(t, err, domain.ErrMagicLinkNotFound)
}

func TestMagicLinkRepo_MarkConsumed(t *testing.T) {
	tx := dbtest.Tx(t)
	created := createTestMagicLink(t, tx)

	repo := NewMagicLinkRepo(tx)
	consumed, err := repo.MarkConsumed(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, consumed.ID)
	require.NotNil(t, consumed.ConsumedAt)
}

func TestMagicLinkRepo_MarkConsumedAlreadyConsumed(t *testing.T) {
	tx := dbtest.Tx(t)
	created := createTestMagicLink(t, tx)

	repo := NewMagicLinkRepo(tx)
	_, err := repo.MarkConsumed(context.Background(), created.ID)
	require.NoError(t, err)

	// Second consume finds no unconsumed row → not found (single-use guard).
	_, err = repo.MarkConsumed(context.Background(), created.ID)
	require.ErrorIs(t, err, domain.ErrMagicLinkNotFound)
}
