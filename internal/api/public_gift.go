package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/gifts"
)

// Public view states returned to a recipient holding a view token.
const (
	stateVisible       = "visible"
	stateNotYetOpen    = "not_yet_open"
	stateExpired       = "expired"
	stateAlreadyOpened = "already_opened"
)

// publicGift is the recipient-facing payload: only what is revealed, never the
// view token, creator or recipient address.
type publicGift struct {
	Title        string          `json:"title"`
	Message      string          `json:"message"`
	PixelArt     json.RawMessage `json:"pixel_art"`
	RevealType   string          `json:"reveal_type"`
	RevealConfig json.RawMessage `json:"reveal_config"`
}

// publicGiftResponse is a discriminated union: state is always present, and the
// extra fields depend on it (gift when visible, scheduled_open_at when pending).
type publicGiftResponse struct {
	State           string      `json:"state"`
	Gift            *publicGift `json:"gift,omitempty"`
	ScheduledOpenAt *time.Time  `json:"scheduled_open_at,omitempty"`
}

func toPublicGift(g domain.Gift) *publicGift {
	return &publicGift{
		Title:        g.Title,
		Message:      g.Message,
		PixelArt:     g.PixelArt,
		RevealType:   g.RevealType,
		RevealConfig: g.RevealConfig,
	}
}

// handleViewGift handles the public GET /g/{view_token}. It applies the
// visibility gate and returns 200 with a state discriminator; only an unknown
// token yields 404. It never marks opened_at (that happens when the reveal
// completes, signalled separately by the reveal frontend).
func (s *Server) handleViewGift(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "view_token")

	gift, err := s.gifts.GetByViewToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, domain.ErrGiftNotFound) {
			respondError(w, http.StatusNotFound, codeGiftNotFound)
			return
		}
		log.Error().Err(err).Msg("view gift failed")
		respondError(w, http.StatusInternalServerError, codeInternalError)
		return
	}

	switch gifts.CheckVisibility(gift, time.Now()) {
	case gifts.Visible:
		respondJSON(w, http.StatusOK, publicGiftResponse{State: stateVisible, Gift: toPublicGift(gift)})
	case gifts.NotYetOpen:
		respondJSON(w, http.StatusOK, publicGiftResponse{State: stateNotYetOpen, ScheduledOpenAt: gift.ScheduledOpenAt})
	case gifts.Expired:
		respondJSON(w, http.StatusOK, publicGiftResponse{State: stateExpired})
	case gifts.AlreadyOpened:
		respondJSON(w, http.StatusOK, publicGiftResponse{State: stateAlreadyOpened})
	}
}
