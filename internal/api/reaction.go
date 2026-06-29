package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/reactions"
)

// reactionResponse is the JSON representation of a reaction. Emoji/message are
// omitted when empty so the shape reflects the reaction kind.
type reactionResponse struct {
	ID        uuid.UUID `json:"id"`
	Kind      string    `json:"kind"`
	Emoji     *string   `json:"emoji,omitempty"`
	Message   *string   `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func toReactionResponse(r domain.Reaction) reactionResponse {
	return reactionResponse{
		ID:        r.ID,
		Kind:      r.Kind,
		Emoji:     r.Emoji,
		Message:   r.Message,
		CreatedAt: r.CreatedAt,
	}
}

// reactionWriteBody is the request payload of POST /g/{view_token}/reactions.
type reactionWriteBody struct {
	Kind    string `json:"kind"`
	Emoji   string `json:"emoji"`
	Message string `json:"message"`
}

// handleCreateReaction handles the public POST /g/{view_token}/reactions. No
// session: anyone holding the token may react, but only while the gift is
// visible. 201 on success, 400 malformed, 404 unknown token, 409 gift gated.
func (s *Server) handleCreateReaction(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "view_token")

	var body reactionWriteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, codeInvalidRequest)
		return
	}

	reaction, err := s.reactions.Create(r.Context(), reactions.CreateInput{
		ViewToken: token,
		Kind:      body.Kind,
		Emoji:     body.Emoji,
		Message:   body.Message,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrGiftNotFound):
			respondError(w, http.StatusNotFound, codeGiftNotFound)
		case errors.Is(err, domain.ErrGiftNotVisible):
			respondError(w, http.StatusConflict, codeGiftNotVisible)
		case errors.Is(err, domain.ErrReactionInvalid):
			respondError(w, http.StatusBadRequest, codeInvalidReaction)
		default:
			log.Error().Err(err).Msg("create reaction failed")
			respondError(w, http.StatusInternalServerError, codeInternalError)
		}
		return
	}

	respondJSON(w, http.StatusCreated, toReactionResponse(reaction))
}
