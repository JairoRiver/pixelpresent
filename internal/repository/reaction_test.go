package repository

import (
	"context"
	"testing"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/repository/db/dbtest"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

// createTestReaction creates an emoji reaction for giftID on the same tx.
func createTestReaction(t *testing.T, tx pgx.Tx, giftID uuid.UUID) domain.Reaction {
	t.Helper()

	reaction, err := NewReactionRepo(tx).Create(context.Background(), domain.Reaction{
		GiftID: giftID,
		Kind:   "emoji",
		Emoji:  strPtr("🎉"),
	})
	require.NoError(t, err)
	return reaction
}

func TestReactionRepo_Create_Emoji(t *testing.T) {
	tx := dbtest.Tx(t)
	gift := createTestGift(t, tx)

	reaction, err := NewReactionRepo(tx).Create(context.Background(), domain.Reaction{
		GiftID: gift.ID,
		Kind:   "emoji",
		Emoji:  strPtr("❤️"),
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, reaction.ID)
	require.Equal(t, gift.ID, reaction.GiftID)
	require.Equal(t, "emoji", reaction.Kind)
	require.NotNil(t, reaction.Emoji)
	require.Equal(t, "❤️", *reaction.Emoji)
	// Unset optional fields stay nil.
	require.Nil(t, reaction.Message)
	require.Nil(t, reaction.VoiceStorageKey)
	require.False(t, reaction.CreatedAt.IsZero())
}

func TestReactionRepo_Create_Text(t *testing.T) {
	tx := dbtest.Tx(t)
	gift := createTestGift(t, tx)

	msg := util.RandomString(40)
	reaction, err := NewReactionRepo(tx).Create(context.Background(), domain.Reaction{
		GiftID:  gift.ID,
		Kind:    "text",
		Message: &msg,
	})
	require.NoError(t, err)
	require.Equal(t, "text", reaction.Kind)
	require.NotNil(t, reaction.Message)
	require.Equal(t, msg, *reaction.Message)
	require.Nil(t, reaction.Emoji)
}

func TestReactionRepo_ListByGift(t *testing.T) {
	tx := dbtest.Tx(t)
	gift := createTestGift(t, tx)
	otherGift := createTestGift(t, tx)

	r1 := createTestReaction(t, tx, gift.ID)
	r2 := createTestReaction(t, tx, gift.ID)
	otherReaction := createTestReaction(t, tx, otherGift.ID)

	reactions, err := NewReactionRepo(tx).ListByGift(context.Background(), gift.ID)
	require.NoError(t, err)
	require.Len(t, reactions, 2, "only this gift's reactions")

	ids := []uuid.UUID{reactions[0].ID, reactions[1].ID}
	require.Contains(t, ids, r1.ID)
	require.Contains(t, ids, r2.ID)
	require.NotContains(t, ids, otherReaction.ID)
}

func TestReactionRepo_ListByGift_Empty(t *testing.T) {
	tx := dbtest.Tx(t)
	gift := createTestGift(t, tx)

	reactions, err := NewReactionRepo(tx).ListByGift(context.Background(), gift.ID)
	require.NoError(t, err)
	require.Empty(t, reactions)
}
