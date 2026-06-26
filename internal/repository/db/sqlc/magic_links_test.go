package sqlc

import (
	"context"
	"testing"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/repository/db/dbtest"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// createRandomMagicLink creates a user and a magic link for it, on the caller's
// transactional Queries, for reuse by tests that depend on magic links.
func createRandomMagicLink(t *testing.T, q *Queries) MagicLink {
	t.Helper()

	user := createRandomUser(t, q)

	params := CreateMagicLinkParams{
		UserID:    user.ID,
		TokenHash: util.RandomString(64),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	link, err := q.CreateMagicLink(context.Background(), params)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, link.ID)
	require.Equal(t, params.UserID, link.UserID)
	require.Equal(t, params.TokenHash, link.TokenHash)
	require.WithinDuration(t, params.ExpiresAt, link.ExpiresAt, time.Second)
	require.False(t, link.ConsumedAt.Valid)
	require.False(t, link.CreatedAt.IsZero())

	return link
}

func TestCreateMagicLink(t *testing.T) {
	q := New(dbtest.Tx(t))

	_ = createRandomMagicLink(t, q)
}

func TestGetMagicLinkByTokenHash(t *testing.T) {
	q := New(dbtest.Tx(t))

	created := createRandomMagicLink(t, q)

	got, err := q.GetMagicLinkByTokenHash(context.Background(), created.TokenHash)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.UserID, got.UserID)
}

func TestGetMagicLinkByTokenHash_NotFound(t *testing.T) {
	q := New(dbtest.Tx(t))

	_, err := q.GetMagicLinkByTokenHash(context.Background(), util.RandomString(64))
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestMarkMagicLinkConsumed(t *testing.T) {
	q := New(dbtest.Tx(t))

	created := createRandomMagicLink(t, q)

	consumed, err := q.MarkMagicLinkConsumed(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, consumed.ID)
	require.True(t, consumed.ConsumedAt.Valid)
}

func TestMarkMagicLinkConsumed_AlreadyConsumed(t *testing.T) {
	q := New(dbtest.Tx(t))

	created := createRandomMagicLink(t, q)

	_, err := q.MarkMagicLinkConsumed(context.Background(), created.ID)
	require.NoError(t, err)

	// The guard (consumed_at IS NULL) makes a second consume a no-op: no row.
	_, err = q.MarkMagicLinkConsumed(context.Background(), created.ID)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
