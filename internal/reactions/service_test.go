package reactions

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/JairoRiver/pixelpresent/internal/domain"
)

// fakeGiftRepo is an in-memory domain.GiftRepository; only GetByViewToken and
// GetByID are exercised by the reaction service, the rest satisfy the interface.
type fakeGiftRepo struct {
	byToken map[string]domain.Gift
	byID    map[uuid.UUID]domain.Gift
}

var _ domain.GiftRepository = (*fakeGiftRepo)(nil)

func (f *fakeGiftRepo) GetByViewToken(_ context.Context, token string) (domain.Gift, error) {
	g, ok := f.byToken[token]
	if !ok {
		return domain.Gift{}, domain.ErrGiftNotFound
	}
	return g, nil
}

func (f *fakeGiftRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Gift, error) {
	g, ok := f.byID[id]
	if !ok {
		return domain.Gift{}, domain.ErrGiftNotFound
	}
	return g, nil
}

func (f *fakeGiftRepo) Create(context.Context, domain.Gift) (domain.Gift, error) {
	return domain.Gift{}, nil
}
func (f *fakeGiftRepo) Update(context.Context, domain.Gift) (domain.Gift, error) {
	return domain.Gift{}, nil
}
func (f *fakeGiftRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (f *fakeGiftRepo) ListByUser(context.Context, uuid.UUID) ([]domain.Gift, error) {
	return nil, nil
}
func (f *fakeGiftRepo) ViewTokenExists(context.Context, string) (bool, error) { return false, nil }

// fakeReactionRepo is an in-memory domain.ReactionRepository.
type fakeReactionRepo struct {
	created []domain.Reaction
}

var _ domain.ReactionRepository = (*fakeReactionRepo)(nil)

func (f *fakeReactionRepo) Create(_ context.Context, r domain.Reaction) (domain.Reaction, error) {
	r.ID = uuid.New()
	r.CreatedAt = time.Now()
	f.created = append(f.created, r)
	return r, nil
}

func (f *fakeReactionRepo) ListByGift(_ context.Context, giftID uuid.UUID) ([]domain.Reaction, error) {
	out := []domain.Reaction{}
	for _, r := range f.created {
		if r.GiftID == giftID {
			out = append(out, r)
		}
	}
	return out, nil
}

// newServiceWith builds a service whose gift repo holds a single gift under
// "tok" (also retrievable by its id), letting each test shape that gift's fields.
func newServiceWith(gift domain.Gift) (*Service, *fakeReactionRepo) {
	gift.ViewToken = "tok"
	gifts := &fakeGiftRepo{
		byToken: map[string]domain.Gift{"tok": gift},
		byID:    map[uuid.UUID]domain.Gift{gift.ID: gift},
	}
	reactions := &fakeReactionRepo{}
	return NewService(gifts, reactions), reactions
}

func visibleGift() domain.Gift {
	published := time.Now().Add(-time.Hour)
	return domain.Gift{ID: uuid.New(), Title: "Para ti", PublishedAt: &published}
}

func TestService_Create_Emoji(t *testing.T) {
	svc, repo := newServiceWith(visibleGift())

	reaction, err := svc.Create(context.Background(), CreateInput{
		ViewToken: "tok",
		Kind:      KindEmoji,
		Emoji:     "🎉",
	})
	require.NoError(t, err)
	require.Equal(t, KindEmoji, reaction.Kind)
	require.NotNil(t, reaction.Emoji)
	require.Equal(t, "🎉", *reaction.Emoji)
	require.Nil(t, reaction.Message)
	require.Len(t, repo.created, 1, "the reaction was persisted")
}

func TestService_Create_Text(t *testing.T) {
	svc, _ := newServiceWith(visibleGift())

	reaction, err := svc.Create(context.Background(), CreateInput{
		ViewToken: "tok",
		Kind:      KindText,
		Message:   "  ¡gracias!  ",
	})
	require.NoError(t, err)
	require.Equal(t, KindText, reaction.Kind)
	require.NotNil(t, reaction.Message)
	require.Equal(t, "¡gracias!", *reaction.Message, "content is trimmed")
	require.Nil(t, reaction.Emoji)
}

func TestService_Create_GiftNotFound(t *testing.T) {
	svc, _ := newServiceWith(visibleGift())

	_, err := svc.Create(context.Background(), CreateInput{
		ViewToken: "unknown",
		Kind:      KindEmoji,
		Emoji:     "🎉",
	})
	require.ErrorIs(t, err, domain.ErrGiftNotFound)
}

func TestService_Create_NotYetOpen(t *testing.T) {
	gift := visibleGift()
	openAt := time.Now().Add(24 * time.Hour)
	gift.ScheduledOpenAt = &openAt
	svc, repo := newServiceWith(gift)

	_, err := svc.Create(context.Background(), CreateInput{ViewToken: "tok", Kind: KindEmoji, Emoji: "🎉"})
	require.ErrorIs(t, err, domain.ErrGiftNotVisible)
	require.Empty(t, repo.created, "nothing persisted for a gated gift")
}

func TestService_Create_Expired(t *testing.T) {
	gift := visibleGift()
	past := time.Now().Add(-time.Hour)
	gift.ExpiresAt = &past
	svc, _ := newServiceWith(gift)

	_, err := svc.Create(context.Background(), CreateInput{ViewToken: "tok", Kind: KindEmoji, Emoji: "🎉"})
	require.ErrorIs(t, err, domain.ErrGiftNotVisible)
}

func TestService_Create_AlreadyOpened(t *testing.T) {
	gift := visibleGift()
	opened := time.Now().Add(-time.Minute)
	gift.SingleOpen = true
	gift.OpenedAt = &opened
	svc, _ := newServiceWith(gift)

	_, err := svc.Create(context.Background(), CreateInput{ViewToken: "tok", Kind: KindEmoji, Emoji: "🎉"})
	require.ErrorIs(t, err, domain.ErrGiftNotVisible)
}

func TestService_Create_InvalidKind(t *testing.T) {
	svc, _ := newServiceWith(visibleGift())

	_, err := svc.Create(context.Background(), CreateInput{ViewToken: "tok", Kind: "voice", Message: "hola"})
	require.ErrorIs(t, err, domain.ErrReactionInvalid)
}

func TestService_Create_EmptyEmoji(t *testing.T) {
	svc, _ := newServiceWith(visibleGift())

	_, err := svc.Create(context.Background(), CreateInput{ViewToken: "tok", Kind: KindEmoji, Emoji: "   "})
	require.ErrorIs(t, err, domain.ErrReactionInvalid)
}

func TestService_Create_OversizeMessage(t *testing.T) {
	svc, _ := newServiceWith(visibleGift())

	_, err := svc.Create(context.Background(), CreateInput{
		ViewToken: "tok",
		Kind:      KindText,
		Message:   strings.Repeat("a", maxMessageRunes+1),
	})
	require.ErrorIs(t, err, domain.ErrReactionInvalid)
}

func TestService_ListForOwner_Own(t *testing.T) {
	owner := uuid.New()
	gift := visibleGift()
	gift.CreatorID = owner
	svc, repo := newServiceWith(gift)
	repo.created = []domain.Reaction{
		{ID: uuid.New(), GiftID: gift.ID, Kind: KindEmoji},
		{ID: uuid.New(), GiftID: gift.ID, Kind: KindText},
		{ID: uuid.New(), GiftID: uuid.New(), Kind: KindEmoji}, // another gift's
	}

	list, err := svc.ListForOwner(context.Background(), gift.ID, owner)
	require.NoError(t, err)
	require.Len(t, list, 2, "only this gift's reactions")
}

func TestService_ListForOwner_Foreign(t *testing.T) {
	gift := visibleGift()
	gift.CreatorID = uuid.New()
	svc, _ := newServiceWith(gift)

	_, err := svc.ListForOwner(context.Background(), gift.ID, uuid.New())
	require.ErrorIs(t, err, domain.ErrGiftForbidden)
}

func TestService_ListForOwner_NotFound(t *testing.T) {
	svc, _ := newServiceWith(visibleGift())

	_, err := svc.ListForOwner(context.Background(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, domain.ErrGiftNotFound)
}
