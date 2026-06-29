package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/JairoRiver/pixelpresent/internal/auth"
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

// listReactionsResponse wraps the reaction list so the shape can grow without
// breaking clients that expect a JSON object.
type listReactionsResponse struct {
	Reactions []reactionResponse `json:"reactions"`
}

// handleListReactions handles GET /gifts/{id}/reactions, returning a gift's
// reactions (oldest first) only to its creator. 200 on success, 403 foreign,
// 404 missing.
func (s *Server) handleListReactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, codeUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, codeInvalidID)
		return
	}

	list, err := s.reactions.ListForOwner(r.Context(), id, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrGiftNotFound):
			respondError(w, http.StatusNotFound, codeGiftNotFound)
		case errors.Is(err, domain.ErrGiftForbidden):
			respondError(w, http.StatusForbidden, codeForbidden)
		default:
			log.Error().Err(err).Msg("list reactions failed")
			respondError(w, http.StatusInternalServerError, codeInternalError)
		}
		return
	}

	resp := listReactionsResponse{Reactions: make([]reactionResponse, len(list))}
	for i, reaction := range list {
		resp.Reactions[i] = toReactionResponse(reaction)
	}
	respondJSON(w, http.StatusOK, resp)
}
