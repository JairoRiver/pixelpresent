package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const testSessionSecret = "test-secret"

// protectedHandler records whether it ran and the user id it saw in context.
func protectedHandler(seen *uuid.UUID, ran *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*ran = true
		if id, ok := UserIDFromContext(r.Context()); ok {
			*seen = id
		}
		w.WriteHeader(http.StatusOK)
	})
}

func TestSetCookie(t *testing.T) {
	m := NewSessionManager(testSessionSecret, true, 30*24*time.Hour)
	userID := uuid.New()

	rec := httptest.NewRecorder()
	m.SetCookie(rec, userID)

	res := rec.Result()
	cookies := res.Cookies()
	require.Len(t, cookies, 1)
	c := cookies[0]

	require.Equal(t, SessionCookieName, c.Name)
	require.Equal(t, "/", c.Path)
	require.True(t, c.HttpOnly)
	require.True(t, c.Secure)
	require.Equal(t, http.SameSiteLaxMode, c.SameSite)

	// The value round-trips back to the same user id.
	gotID, err := m.decode(c.Value)
	require.NoError(t, err)
	require.Equal(t, userID, gotID)
}

func TestRequireSession_ValidCookie(t *testing.T) {
	m := NewSessionManager(testSessionSecret, false, 30*24*time.Hour)
	userID := uuid.New()

	var seen uuid.UUID
	var ran bool
	handler := m.RequireSession(protectedHandler(&seen, &ran))

	// Issue a cookie, then replay it on a fresh request.
	issue := httptest.NewRecorder()
	m.SetCookie(issue, userID)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(issue.Result().Cookies()[0])

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, ran, "the protected handler should run")
	require.Equal(t, userID, seen, "the user id should be exposed in context")
}

func TestRequireSession_NoCookie(t *testing.T) {
	m := NewSessionManager(testSessionSecret, false, 30*24*time.Hour)

	var seen uuid.UUID
	var ran bool
	handler := m.RequireSession(protectedHandler(&seen, &ran))

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, ran, "the protected handler must not run without a session")
	require.Contains(t, rec.Body.String(), "unauthorized")
}

func TestRequireSession_TamperedCookie(t *testing.T) {
	m := NewSessionManager(testSessionSecret, false, 30*24*time.Hour)
	userID := uuid.New()

	var ran bool
	handler := m.RequireSession(protectedHandler(new(uuid.UUID), &ran))

	issue := httptest.NewRecorder()
	m.SetCookie(issue, userID)
	cookie := issue.Result().Cookies()[0]
	cookie.Value += "x" // break the signature

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, ran)
}

func TestRequireSession_WrongSecret(t *testing.T) {
	issuer := NewSessionManager("secret-A", false, 30*24*time.Hour)
	verifier := NewSessionManager("secret-B", false, 30*24*time.Hour)

	issue := httptest.NewRecorder()
	issuer.SetCookie(issue, uuid.New())

	var ran bool
	handler := verifier.RequireSession(protectedHandler(new(uuid.UUID), &ran))

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(issue.Result().Cookies()[0])
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, ran, "a cookie signed with another secret must be rejected")
}

func TestRequireSession_ExpiredCookie(t *testing.T) {
	// Negative ttl makes SetCookie mint an already-expired payload.
	m := NewSessionManager(testSessionSecret, false, -time.Minute)

	issue := httptest.NewRecorder()
	m.SetCookie(issue, uuid.New())

	var ran bool
	handler := m.RequireSession(protectedHandler(new(uuid.UUID), &ran))

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(issue.Result().Cookies()[0])
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, ran, "an expired session must be rejected")
}
