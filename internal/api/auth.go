package api

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"

	"github.com/rs/zerolog/log"
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
