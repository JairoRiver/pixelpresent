package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/JairoRiver/pixelpresent/internal/auth"
)

func sessionServer() (*Server, *auth.SessionManager) {
	sessions := auth.NewSessionManager("test-secret", false, time.Hour)
	return NewServer(&fakeAuth{}, sessions, nil, nil), sessions
}

func TestMe_RequiresSession(t *testing.T) {
	srv, _ := sessionServer()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_ReturnsUserID(t *testing.T) {
	srv, sessions := sessionServer()
	userID := uuid.New()

	issue := httptest.NewRecorder()
	sessions.SetCookie(issue, userID)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(issue.Result().Cookies()[0])
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body meResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, userID, body.ID)
}

func TestLogout_ClearsCookie(t *testing.T) {
	srv, _ := sessionServer()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	c := sessionCookie(rec)
	require.NotNil(t, c, "logout sends a clearing cookie")
	require.Empty(t, c.Value)
	require.Equal(t, -1, c.MaxAge, "the cookie is expired immediately")
}
