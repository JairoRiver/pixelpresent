package auth

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/email"
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
	require.Contains(t, sent.BodyText, "https://pixel.example/auth/verify?token=")
	require.Contains(t, sent.BodyText, "15 minutos")
	require.Contains(t, sent.BodyHTML, "https://pixel.example/auth/verify?token=")
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
