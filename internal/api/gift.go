package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/JairoRiver/pixelpresent/internal/auth"
	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/gifts"
)

// giftResponse is the JSON representation of a gift returned to its creator.
type giftResponse struct {
	ID              uuid.UUID       `json:"id"`
	CreatorID       uuid.UUID       `json:"creator_id"`
	Title           string          `json:"title"`
	Message         string          `json:"message"`
	PixelArt        json.RawMessage `json:"pixel_art"`
	RevealType      string          `json:"reveal_type"`
	RevealConfig    json.RawMessage `json:"reveal_config"`
	ViewToken       string          `json:"view_token"`
	RecipientEmail  *string         `json:"recipient_email,omitempty"`
	ScheduledOpenAt *time.Time      `json:"scheduled_open_at,omitempty"`
	ScheduledSendAt *time.Time      `json:"scheduled_send_at,omitempty"`
	SentAt          *time.Time      `json:"sent_at,omitempty"`
	SingleOpen      bool            `json:"single_open"`
	OpenedAt        *time.Time      `json:"opened_at,omitempty"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func toGiftResponse(g domain.Gift) giftResponse {
	return giftResponse{
		ID:              g.ID,
		CreatorID:       g.CreatorID,
		Title:           g.Title,
		Message:         g.Message,
		PixelArt:        g.PixelArt,
		RevealType:      g.RevealType,
		RevealConfig:    g.RevealConfig,
		ViewToken:       g.ViewToken,
		RecipientEmail:  g.RecipientEmail,
		ScheduledOpenAt: g.ScheduledOpenAt,
		ScheduledSendAt: g.ScheduledSendAt,
		SentAt:          g.SentAt,
		SingleOpen:      g.SingleOpen,
		OpenedAt:        g.OpenedAt,
		ExpiresAt:       g.ExpiresAt,
		CreatedAt:       g.CreatedAt,
		UpdatedAt:       g.UpdatedAt,
	}
}

// giftWriteBody is the shared request payload of POST and PUT /gifts.
type giftWriteBody struct {
	Title           string          `json:"title"`
	Message         string          `json:"message"`
	PixelArt        json.RawMessage `json:"pixel_art"`
	RevealType      string          `json:"reveal_type"`
	RevealConfig    json.RawMessage `json:"reveal_config"`
	RecipientEmail  *string         `json:"recipient_email"`
	ScheduledOpenAt *time.Time      `json:"scheduled_open_at"`
	ScheduledSendAt *time.Time      `json:"scheduled_send_at"`
	SingleOpen      bool            `json:"single_open"`
	ExpiresAt       *time.Time      `json:"expires_at"`
}

// decodeGiftWriteBody decodes and validates the gift payload shared by create
// and update. On any problem it writes the error response and returns ok=false.
func decodeGiftWriteBody(w http.ResponseWriter, r *http.Request) (giftWriteBody, bool) {
	var body giftWriteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, codeInvalidRequest)
		return body, false
	}

	body.Title = strings.TrimSpace(body.Title)
	if body.Title == "" {
		respondError(w, http.StatusBadRequest, codeInvalidGift)
		return body, false
	}
	if !gifts.ValidRevealType(body.RevealType) {
		respondError(w, http.StatusBadRequest, codeInvalidRevealType)
		return body, false
	}
	if !json.Valid(body.PixelArt) {
		respondError(w, http.StatusBadRequest, codeInvalidPixelArt)
		return body, false
	}
	if body.RecipientEmail != nil {
		if _, err := mail.ParseAddress(*body.RecipientEmail); err != nil {
			respondError(w, http.StatusBadRequest, codeInvalidEmail)
			return body, false
		}
	}
	return body, true
}

type createGiftResponse struct {
	ID        uuid.UUID `json:"id"`
	ViewToken string    `json:"view_token"`
}

// handleCreateGift handles POST /gifts for the authenticated creator. The
// session middleware guarantees a user id in the context.
func (s *Server) handleCreateGift(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		// Should never happen behind RequireSession; treat defensively.
		respondError(w, http.StatusUnauthorized, codeUnauthorized)
		return
	}

	body, ok := decodeGiftWriteBody(w, r)
	if !ok {
		return
	}

	gift, err := s.gifts.Create(r.Context(), gifts.CreateInput{
		CreatorID:       userID,
		Title:           body.Title,
		Message:         body.Message,
		PixelArt:        body.PixelArt,
		RevealType:      body.RevealType,
		RevealConfig:    body.RevealConfig,
		RecipientEmail:  body.RecipientEmail,
		ScheduledOpenAt: body.ScheduledOpenAt,
		ScheduledSendAt: body.ScheduledSendAt,
		SingleOpen:      body.SingleOpen,
		ExpiresAt:       body.ExpiresAt,
	})
	if err != nil {
		log.Error().Err(err).Msg("create gift failed")
		respondError(w, http.StatusInternalServerError, codeInternalError)
		return
	}

	respondJSON(w, http.StatusCreated, createGiftResponse{ID: gift.ID, ViewToken: gift.ViewToken})
}

// handleUpdateGift handles PUT /gifts/{id}: full-replace of the editable fields,
// only for the gift's creator (403 foreign, 404 missing).
func (s *Server) handleUpdateGift(w http.ResponseWriter, r *http.Request) {
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

	body, ok := decodeGiftWriteBody(w, r)
	if !ok {
		return
	}

	gift, err := s.gifts.UpdateOwned(r.Context(), id, userID, gifts.UpdateInput{
		Title:           body.Title,
		Message:         body.Message,
		PixelArt:        body.PixelArt,
		RevealType:      body.RevealType,
		RevealConfig:    body.RevealConfig,
		RecipientEmail:  body.RecipientEmail,
		ScheduledOpenAt: body.ScheduledOpenAt,
		ScheduledSendAt: body.ScheduledSendAt,
		SingleOpen:      body.SingleOpen,
		ExpiresAt:       body.ExpiresAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrGiftNotFound):
			respondError(w, http.StatusNotFound, codeGiftNotFound)
		case errors.Is(err, domain.ErrGiftForbidden):
			respondError(w, http.StatusForbidden, codeForbidden)
		default:
			log.Error().Err(err).Msg("update gift failed")
			respondError(w, http.StatusInternalServerError, codeInternalError)
		}
		return
	}

	respondJSON(w, http.StatusOK, toGiftResponse(gift))
}

// handleGetGift handles GET /gifts/{id}, returning the gift only to its creator:
// 404 if it does not exist, 403 if it belongs to another user.
func (s *Server) handleGetGift(w http.ResponseWriter, r *http.Request) {
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

	gift, err := s.gifts.GetOwned(r.Context(), id, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrGiftNotFound):
			respondError(w, http.StatusNotFound, codeGiftNotFound)
		case errors.Is(err, domain.ErrGiftForbidden):
			respondError(w, http.StatusForbidden, codeForbidden)
		default:
			log.Error().Err(err).Msg("get gift failed")
			respondError(w, http.StatusInternalServerError, codeInternalError)
		}
		return
	}

	respondJSON(w, http.StatusOK, toGiftResponse(gift))
}

// handleDeleteGift handles DELETE /gifts/{id}: hard-deletes the gift (media and
// reactions cascade) for its creator only. 204 on success, 403 foreign, 404 missing.
func (s *Server) handleDeleteGift(w http.ResponseWriter, r *http.Request) {
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

	if err := s.gifts.DeleteOwned(r.Context(), id, userID); err != nil {
		switch {
		case errors.Is(err, domain.ErrGiftNotFound):
			respondError(w, http.StatusNotFound, codeGiftNotFound)
		case errors.Is(err, domain.ErrGiftForbidden):
			respondError(w, http.StatusForbidden, codeForbidden)
		default:
			log.Error().Err(err).Msg("delete gift failed")
			respondError(w, http.StatusInternalServerError, codeInternalError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
