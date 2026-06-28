package sqlc

import (
	"context"
	"testing"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/repository/db/dbtest"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// createRandomGift creates a gift for creatorID on the caller's transactional
// Queries, for reuse by tests that depend on gifts.
func createRandomGift(t *testing.T, q *Queries, creatorID uuid.UUID) Gift {
	t.Helper()

	params := CreateGiftParams{
		CreatorID:    creatorID,
		Title:        util.RandomString(10),
		Message:      util.RandomString(20),
		PixelArt:     []byte(`{"width":2,"height":2,"palette":["#000000","#ffffff"],"pixels":[0,1,1,0]}`),
		RevealType:   "box",
		RevealConfig: []byte(`{}`),
		ViewToken:    util.RandomString(43),
		SingleOpen:   false,
	}

	gift, err := q.CreateGift(context.Background(), params)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, gift.ID)
	require.Equal(t, creatorID, gift.CreatorID)
	require.Equal(t, params.Title, gift.Title)
	require.Equal(t, params.Message, gift.Message)
	require.Equal(t, params.RevealType, gift.RevealType)
	require.Equal(t, params.ViewToken, gift.ViewToken)
	require.JSONEq(t, string(params.PixelArt), string(gift.PixelArt))
	require.JSONEq(t, string(params.RevealConfig), string(gift.RevealConfig))
	// Unset optional fields default to NULL / false / zero.
	require.False(t, gift.RecipientEmail.Valid)
	require.False(t, gift.ScheduledOpenAt.Valid)
	require.False(t, gift.SentAt.Valid)
	require.False(t, gift.OpenedAt.Valid)
	require.False(t, gift.SingleOpen)
	require.False(t, gift.CreatedAt.IsZero())
	require.False(t, gift.UpdatedAt.IsZero())

	return gift
}

func TestCreateGift(t *testing.T) {
	q := New(dbtest.Tx(t))

	user := createRandomUser(t, q)
	_ = createRandomGift(t, q, user.ID)
}

func TestGetGiftByID(t *testing.T) {
	q := New(dbtest.Tx(t))

	user := createRandomUser(t, q)
	created := createRandomGift(t, q, user.ID)

	got, err := q.GetGiftByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.ViewToken, got.ViewToken)
}

func TestGetGiftByID_NotFound(t *testing.T) {
	q := New(dbtest.Tx(t))

	_, err := q.GetGiftByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestGetGiftByViewToken(t *testing.T) {
	q := New(dbtest.Tx(t))

	user := createRandomUser(t, q)
	created := createRandomGift(t, q, user.ID)

	got, err := q.GetGiftByViewToken(context.Background(), created.ViewToken)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
}

func TestGetGiftByViewToken_NotFound(t *testing.T) {
	q := New(dbtest.Tx(t))

	_, err := q.GetGiftByViewToken(context.Background(), util.RandomString(43))
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestGiftViewTokenExists(t *testing.T) {
	q := New(dbtest.Tx(t))

	user := createRandomUser(t, q)
	created := createRandomGift(t, q, user.ID)

	exists, err := q.GiftViewTokenExists(context.Background(), created.ViewToken)
	require.NoError(t, err)
	require.True(t, exists)

	free, err := q.GiftViewTokenExists(context.Background(), util.RandomString(43))
	require.NoError(t, err)
	require.False(t, free)
}

func TestUpdateGift_ReplacesFieldsAndClearsNullables(t *testing.T) {
	q := New(dbtest.Tx(t))

	user := createRandomUser(t, q)
	created := createRandomGift(t, q, user.ID)

	// First update: set new content and populate the nullable fields.
	openAt := time.Now().Add(24 * time.Hour)
	updated, err := q.UpdateGift(context.Background(), UpdateGiftParams{
		ID:              created.ID,
		Title:           "Nuevo título",
		Message:         "Nuevo mensaje",
		PixelArt:        []byte(`{"width":1,"height":1,"palette":["#ff0000"],"pixels":[0]}`),
		RevealType:      "envelope",
		RevealConfig:    []byte(`{"speed":2}`),
		RecipientEmail:  pgtype.Text{String: util.RandomEmail(), Valid: true},
		ScheduledOpenAt: pgtype.Timestamptz{Time: openAt, Valid: true},
		SingleOpen:      true,
	})
	require.NoError(t, err)
	require.Equal(t, "Nuevo título", updated.Title)
	require.Equal(t, "envelope", updated.RevealType)
	require.JSONEq(t, `{"speed":2}`, string(updated.RevealConfig))
	require.True(t, updated.RecipientEmail.Valid)
	require.True(t, updated.ScheduledOpenAt.Valid)
	require.WithinDuration(t, openAt, updated.ScheduledOpenAt.Time, time.Second)
	require.True(t, updated.SingleOpen)
	// view_token / creator_id / created_at are immutable.
	require.Equal(t, created.ViewToken, updated.ViewToken)
	require.Equal(t, created.CreatorID, updated.CreatorID)
	require.Equal(t, created.CreatedAt, updated.CreatedAt)

	// Second update: omit the nullable fields → full-replace clears them to NULL.
	cleared, err := q.UpdateGift(context.Background(), UpdateGiftParams{
		ID:           created.ID,
		Title:        updated.Title,
		Message:      updated.Message,
		PixelArt:     updated.PixelArt,
		RevealType:   updated.RevealType,
		RevealConfig: updated.RevealConfig,
		SingleOpen:   updated.SingleOpen,
		// RecipientEmail / ScheduledOpenAt left as zero value (Valid:false).
	})
	require.NoError(t, err)
	require.False(t, cleared.RecipientEmail.Valid, "full-replace must be able to clear a nullable")
	require.False(t, cleared.ScheduledOpenAt.Valid)
}

func TestUpdateGift_NotFound(t *testing.T) {
	q := New(dbtest.Tx(t))

	_, err := q.UpdateGift(context.Background(), UpdateGiftParams{
		ID:           uuid.New(),
		Title:        "x",
		Message:      "y",
		PixelArt:     []byte(`{}`),
		RevealType:   "box",
		RevealConfig: []byte(`{}`),
	})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestListGiftsByUser(t *testing.T) {
	q := New(dbtest.Tx(t))

	user := createRandomUser(t, q)
	other := createRandomUser(t, q)

	g1 := createRandomGift(t, q, user.ID)
	g2 := createRandomGift(t, q, user.ID)
	otherGift := createRandomGift(t, q, other.ID)

	gifts, err := q.ListGiftsByUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, gifts, 2, "only this user's gifts")

	ids := []uuid.UUID{gifts[0].ID, gifts[1].ID}
	require.Contains(t, ids, g1.ID)
	require.Contains(t, ids, g2.ID)
	require.NotContains(t, ids, otherGift.ID)
}

func TestListGiftsByUser_Empty(t *testing.T) {
	q := New(dbtest.Tx(t))

	user := createRandomUser(t, q)

	gifts, err := q.ListGiftsByUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Empty(t, gifts)
}
