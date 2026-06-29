package sqlc

import (
	"context"
	"testing"

	"github.com/JairoRiver/pixelpresent/internal/repository/db/dbtest"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// createRandomReaction creates an emoji reaction for giftID on the caller's
// transactional Queries, for reuse by tests that depend on reactions.
func createRandomReaction(t *testing.T, q *Queries, giftID uuid.UUID) Reaction {
	t.Helper()

	params := CreateReactionParams{
		GiftID: giftID,
		Kind:   "emoji",
		Emoji:  pgtype.Text{String: "🎉", Valid: true},
	}

	reaction, err := q.CreateReaction(context.Background(), params)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, reaction.ID)
	require.Equal(t, giftID, reaction.GiftID)
	require.Equal(t, "emoji", reaction.Kind)
	require.True(t, reaction.Emoji.Valid)
	require.Equal(t, "🎉", reaction.Emoji.String)
	// Unset optional fields default to NULL.
	require.False(t, reaction.Message.Valid)
	require.False(t, reaction.VoiceStorageKey.Valid)
	require.False(t, reaction.CreatedAt.IsZero())

	return reaction
}

func TestCreateReaction_Emoji(t *testing.T) {
	q := New(dbtest.Tx(t))

	user := createRandomUser(t, q)
	gift := createRandomGift(t, q, user.ID)
	_ = createRandomReaction(t, q, gift.ID)
}

func TestCreateReaction_Text(t *testing.T) {
	q := New(dbtest.Tx(t))

	user := createRandomUser(t, q)
	gift := createRandomGift(t, q, user.ID)

	msg := util.RandomString(40)
	reaction, err := q.CreateReaction(context.Background(), CreateReactionParams{
		GiftID:  gift.ID,
		Kind:    "text",
		Message: pgtype.Text{String: msg, Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, "text", reaction.Kind)
	require.True(t, reaction.Message.Valid)
	require.Equal(t, msg, reaction.Message.String)
	require.False(t, reaction.Emoji.Valid)
}

func TestListReactionsByGift(t *testing.T) {
	q := New(dbtest.Tx(t))

	user := createRandomUser(t, q)
	gift := createRandomGift(t, q, user.ID)
	otherGift := createRandomGift(t, q, user.ID)

	r1 := createRandomReaction(t, q, gift.ID)
	r2 := createRandomReaction(t, q, gift.ID)
	otherReaction := createRandomReaction(t, q, otherGift.ID)

	reactions, err := q.ListReactionsByGift(context.Background(), gift.ID)
	require.NoError(t, err)
	require.Len(t, reactions, 2, "only this gift's reactions")

	ids := []uuid.UUID{reactions[0].ID, reactions[1].ID}
	require.Contains(t, ids, r1.ID)
	require.Contains(t, ids, r2.ID)
	require.NotContains(t, ids, otherReaction.ID)
}

func TestListReactionsByGift_Empty(t *testing.T) {
	q := New(dbtest.Tx(t))

	user := createRandomUser(t, q)
	gift := createRandomGift(t, q, user.ID)

	reactions, err := q.ListReactionsByGift(context.Background(), gift.ID)
	require.NoError(t, err)
	require.Empty(t, reactions)
}
