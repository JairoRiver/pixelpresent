package auth

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/email"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/stretchr/testify/require"
)

const testTTL = 15 * time.Minute

// extractToken pulls the token query param out of the first URL found in body.
func extractToken(t *testing.T, body string) string {
	t.Helper()

	idx := strings.Index(body, "http")
	require.GreaterOrEqual(t, idx, 0, "no URL found in body")

	raw := body[idx:]
	if end := strings.IndexAny(raw, " \n\r\t"); end >= 0 {
		raw = raw[:end]
	}

	u, err := url.Parse(raw)
	require.NoError(t, err)
	token := u.Query().Get("token")
	require.NotEmpty(t, token)
	return token
}

func TestRequestMagicLink_NewUser(t *testing.T) {
	users := newFakeUserRepo()
	links := newFakeMagicLinkRepo()
	emails := email.NewFake()
	svc := NewService(users, links, emails, "https://pixel.example/", testTTL)

	err := svc.RequestMagicLink(context.Background(), "  Alice@Example.com  ")
	require.NoError(t, err)

	// A user was created.
	require.Len(t, users.byEmail, 1)

	// Exactly one magic link was stored.
	stored := links.all()
	require.Len(t, stored, 1)
	link := stored[0]

	// One email was sent to the trimmed address.
	sent, ok := emails.Last()
	require.True(t, ok)
	require.Equal(t, "Alice@Example.com", sent.To)
	require.Equal(t, magicLinkSubject, sent.Subject)

	// The token in the email hashes to the stored hash (token never stored raw).
	token := extractToken(t, sent.BodyText)
	require.Equal(t, hashToken(token), link.TokenHash)
	require.NotEqual(t, token, link.TokenHash)

	// The link belongs to the created user and expires within the TTL window.
	user, err := users.GetByEmail(context.Background(), "alice@example.com")
	require.NoError(t, err)
	require.Equal(t, user.ID, link.UserID)
	require.True(t, link.ExpiresAt.After(time.Now()))
	require.WithinDuration(t, time.Now().Add(testTTL), link.ExpiresAt, time.Minute)

	// The URL has no duplicated slash and the copy reflects the TTL.
	require.Contains(t, sent.BodyText, "https://pixel.example/api/auth/verify?token=")
	require.Contains(t, sent.BodyText, "15 minutos")
	require.Contains(t, sent.BodyHTML, "https://pixel.example/api/auth/verify?token=")
}

func TestRequestMagicLink_ExistingUserIsReused(t *testing.T) {
	users := newFakeUserRepo()
	existing, err := users.Create(context.Background(), "bob@example.com")
	require.NoError(t, err)

	links := newFakeMagicLinkRepo()
	emails := email.NewFake()
	svc := NewService(users, links, emails, "https://pixel.example", testTTL)

	// Different casing must resolve to the same user (citext semantics).
	err = svc.RequestMagicLink(context.Background(), "BOB@Example.com")
	require.NoError(t, err)

	require.Len(t, users.byEmail, 1, "no new user should be created")

	stored := links.all()
	require.Len(t, stored, 1)
	require.Equal(t, existing.ID, stored[0].UserID)
}

// seedLink creates a user and a magic link for it in the fakes, returning the
// raw token (only the hash is stored, as in production) and the stored link.
func seedLink(t *testing.T, users *fakeUserRepo, links *fakeMagicLinkRepo, expiresAt time.Time) (string, domain.MagicLink) {
	t.Helper()

	user, err := users.Create(context.Background(), util.RandomEmail())
	require.NoError(t, err)

	token, err := generateToken()
	require.NoError(t, err)

	link, err := links.Create(context.Background(), user.ID, hashToken(token), expiresAt)
	require.NoError(t, err)

	return token, link
}

func TestVerifyMagicLink_Valid(t *testing.T) {
	users := newFakeUserRepo()
	links := newFakeMagicLinkRepo()
	svc := NewService(users, links, email.NewFake(), "https://pixel.example", testTTL)

	token, link := seedLink(t, users, links, time.Now().Add(testTTL))

	user, err := svc.VerifyMagicLink(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, link.UserID, user.ID)

	// The link is now marked consumed.
	consumed, err := links.GetByTokenHash(context.Background(), hashToken(token))
	require.NoError(t, err)
	require.NotNil(t, consumed.ConsumedAt)
}

func TestVerifyMagicLink_SingleUse(t *testing.T) {
	users := newFakeUserRepo()
	links := newFakeMagicLinkRepo()
	svc := NewService(users, links, email.NewFake(), "https://pixel.example", testTTL)

	token, _ := seedLink(t, users, links, time.Now().Add(testTTL))

	_, err := svc.VerifyMagicLink(context.Background(), token)
	require.NoError(t, err)

	// A second verification of the same token must fail as already consumed.
	_, err = svc.VerifyMagicLink(context.Background(), token)
	require.ErrorIs(t, err, domain.ErrMagicLinkConsumed)
}

func TestVerifyMagicLink_Expired(t *testing.T) {
	users := newFakeUserRepo()
	links := newFakeMagicLinkRepo()
	svc := NewService(users, links, email.NewFake(), "https://pixel.example", testTTL)

	token, _ := seedLink(t, users, links, time.Now().Add(-time.Minute))

	_, err := svc.VerifyMagicLink(context.Background(), token)
	require.ErrorIs(t, err, domain.ErrMagicLinkExpired)
}

func TestVerifyMagicLink_AlreadyConsumed(t *testing.T) {
	users := newFakeUserRepo()
	links := newFakeMagicLinkRepo()
	svc := NewService(users, links, email.NewFake(), "https://pixel.example", testTTL)

	token, link := seedLink(t, users, links, time.Now().Add(testTTL))

	// Consume it out of band, then verify.
	_, err := links.MarkConsumed(context.Background(), link.ID)
	require.NoError(t, err)

	_, err = svc.VerifyMagicLink(context.Background(), token)
	require.ErrorIs(t, err, domain.ErrMagicLinkConsumed)
}

func TestVerifyMagicLink_NotFound(t *testing.T) {
	users := newFakeUserRepo()
	links := newFakeMagicLinkRepo()
	svc := NewService(users, links, email.NewFake(), "https://pixel.example", testTTL)

	_, err := svc.VerifyMagicLink(context.Background(), "a-token-that-was-never-issued")
	require.ErrorIs(t, err, domain.ErrMagicLinkNotFound)
}
