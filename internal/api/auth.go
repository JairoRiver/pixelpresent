package api

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/JairoRiver/pixelpresent/internal/auth"
)

const (
	// dashboardPath is the frontend route a logged-in creator lands on after a
	// successful verification (a Fase 4 Astro route, same origin).
	dashboardPath = "/dashboard"
	// authErrorLocation is where verification failures redirect: the app root
	// with an error marker the frontend surfaces. We do not distinguish invalid,
	// expired, consumed, or unknown tokens.
	authErrorLocation = "/?auth_error=invalid_or_expired_link"
)

type requestMagicLinkBody struct {
	Email string `json:"email"`
}

// handleRequestMagicLink handles POST /auth/magic-link. It always responds 202
// for a syntactically valid email, whether or not that email already has an
// account, so the endpoint cannot be used to enumerate registered users.
func (s *Server) handleRequestMagicLink(w http.ResponseWriter, r *http.Request) {
	var body requestMagicLinkBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, codeInvalidRequest)
		return
	}

	email := strings.TrimSpace(body.Email)
	if _, err := mail.ParseAddress(email); err != nil {
		respondError(w, http.StatusBadRequest, codeInvalidEmail)
		return
	}

	if err := s.auth.RequestMagicLink(r.Context(), email); err != nil {
		// The create-or-get path is identical for new and existing emails, so a
		// failure here is a genuine internal error, not an existence signal.
		// Log it server-side; never echo it back.
		log.Error().Err(err).Msg("request magic link failed")
		respondError(w, http.StatusInternalServerError, codeInternalError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// handleVerify handles GET /auth/verify?token=. On a valid token it consumes the
// link, sets the session cookie, and redirects to the dashboard; on any failure
// it redirects to the auth-error page without revealing the cause.
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, authErrorLocation, http.StatusSeeOther)
		return
	}

	user, err := s.auth.VerifyMagicLink(r.Context(), token)
	if err != nil {
		// Invalid, expired, already consumed, or unknown: all look the same to
		// the visitor. Logged at info because it is an expected user-facing case.
		log.Info().Err(err).Msg("magic link verification rejected")
		http.Redirect(w, r, authErrorLocation, http.StatusSeeOther)
		return
	}

	s.sessions.SetCookie(w, user.ID)
	http.Redirect(w, r, dashboardPath, http.StatusSeeOther)
}

type meResponse struct {
	ID uuid.UUID `json:"id"`
}

// handleMe returns the authenticated user's id (from the session cookie), or 401.
// It lets the static frontend ask "am I logged in?" since the session cookie is
// HttpOnly and cannot be read from JS.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		// RequireSession should guarantee this; treat defensively.
		respondError(w, http.StatusUnauthorized, codeUnauthorized)
		return
	}
	respondJSON(w, http.StatusOK, meResponse{ID: userID})
}

// handleLogout clears the session cookie. It is intentionally public and
// idempotent: clearing the cookie is harmless whether or not the caller still
// has a valid session.
func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	s.sessions.ClearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
