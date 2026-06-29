package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/JairoRiver/pixelpresent/internal/auth"
	"github.com/JairoRiver/pixelpresent/internal/domain"
)

// fakeAuth is an in-memory AuthService recording its calls and returning
// scripted results.
type fakeAuth struct {
	requestCalls int
	gotEmail     string
	requestErr   error

	verifyToken string
	verifyUser  domain.User
	verifyErr   error
}

func (f *fakeAuth) RequestMagicLink(_ context.Context, email string) error {
	f.requestCalls++
	f.gotEmail = email
	return f.requestErr
}

func (f *fakeAuth) VerifyMagicLink(_ context.Context, token string) (domain.User, error) {
	f.verifyToken = token
	return f.verifyUser, f.verifyErr
}

// newTestServer builds a Server with a real session manager so cookie wiring is
// exercised end to end. The gift service is nil: auth tests do not hit /gifts.
func newTestServer(svc AuthService) *Server {
	sessions := auth.NewSessionManager("test-secret", false, 30*24*time.Hour)
	return NewServer(svc, sessions, nil, nil)
}

func decodeErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env errorEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	return env.Error.Code
}

// --- POST /auth/magic-link ---

// postMagicLink sends body to POST /auth/magic-link through the real chi router.
func postMagicLink(t *testing.T, svc AuthService, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newTestServer(svc).Routes().ServeHTTP(rec, req)
	return rec
}

func TestRequestMagicLink_Accepted(t *testing.T) {
	fake := &fakeAuth{}
	rec := postMagicLink(t, fake, `{"email":"  Alice@Example.com  "}`)

	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Equal(t, 1, fake.requestCalls)
	require.Equal(t, "Alice@Example.com", fake.gotEmail, "email should be trimmed before the service")
}

func TestRequestMagicLink_BadJSON(t *testing.T) {
	fake := &fakeAuth{}
	rec := postMagicLink(t, fake, `{not json`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidRequest, decodeErrorCode(t, rec))
	require.Zero(t, fake.requestCalls, "the service must not be called on a malformed body")
}

func TestRequestMagicLink_InvalidEmail(t *testing.T) {
	fake := &fakeAuth{}
	rec := postMagicLink(t, fake, `{"email":"not-an-email"}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, codeInvalidEmail, decodeErrorCode(t, rec))
	require.Zero(t, fake.requestCalls, "the service must not be called on an invalid email")
}

func TestRequestMagicLink_InternalError(t *testing.T) {
	fake := &fakeAuth{requestErr: errors.New("smtp down")}
	rec := postMagicLink(t, fake, `{"email":"bob@example.com"}`)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, codeInternalError, decodeErrorCode(t, rec))
}

// TestRequestMagicLink_DoesNotLeakExistence documents the anti-enumeration
// contract: any valid email yields the same 202, so a caller cannot tell a
// registered address from an unregistered one.
func TestRequestMagicLink_DoesNotLeakExistence(t *testing.T) {
	fake := &fakeAuth{}

	recNew := postMagicLink(t, fake, `{"email":"new@example.com"}`)
	recExisting := postMagicLink(t, fake, `{"email":"existing@example.com"}`)

	require.Equal(t, http.StatusAccepted, recNew.Code)
	require.Equal(t, http.StatusAccepted, recExisting.Code)
	require.Empty(t, recNew.Body.String())
	require.Empty(t, recExisting.Body.String())
}

// --- GET /auth/verify ---

// getVerify hits GET /auth/verify?token=<token> through the real chi router.
func getVerify(t *testing.T, svc AuthService, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/auth/verify?token="+token, nil)
	rec := httptest.NewRecorder()
	newTestServer(svc).Routes().ServeHTTP(rec, req)
	return rec
}

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			return c
		}
	}
	return nil
}

func TestVerify_Success(t *testing.T) {
	userID := uuid.New()
	fake := &fakeAuth{verifyUser: domain.User{ID: userID}}

	rec := getVerify(t, fake, "good-token")

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, dashboardPath, rec.Header().Get("Location"))
	require.Equal(t, "good-token", fake.verifyToken)

	c := sessionCookie(rec)
	require.NotNil(t, c, "a session cookie must be set on success")
	require.NotEmpty(t, c.Value)
}

func TestVerify_InvalidOrExpired(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"expired", domain.ErrMagicLinkExpired},
		{"consumed", domain.ErrMagicLinkConsumed},
		{"unknown", domain.ErrMagicLinkNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeAuth{verifyErr: tc.err}
			rec := getVerify(t, fake, "bad-token")

			require.Equal(t, http.StatusSeeOther, rec.Code)
			require.Equal(t, authErrorLocation, rec.Header().Get("Location"))
			require.Nil(t, sessionCookie(rec), "no session cookie on a failed verification")
		})
	}
}

func TestVerify_MissingToken(t *testing.T) {
	fake := &fakeAuth{}
	req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
	rec := httptest.NewRecorder()
	newTestServer(fake).Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, authErrorLocation, rec.Header().Get("Location"))
	require.Empty(t, fake.verifyToken, "the service must not be called without a token")
}
