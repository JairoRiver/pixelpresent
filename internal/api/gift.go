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

type createGiftBody struct {
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

	var body createGiftBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, codeInvalidRequest)
		return
	}

	body.Title = strings.TrimSpace(body.Title)
	if body.Title == "" {
		respondError(w, http.StatusBadRequest, codeInvalidGift)
		return
	}
	if !gifts.ValidRevealType(body.RevealType) {
		respondError(w, http.StatusBadRequest, codeInvalidRevealType)
		return
	}
	if !json.Valid(body.PixelArt) {
		respondError(w, http.StatusBadRequest, codeInvalidPixelArt)
		return
	}
	if body.RecipientEmail != nil {
		if _, err := mail.ParseAddress(*body.RecipientEmail); err != nil {
			respondError(w, http.StatusBadRequest, codeInvalidEmail)
			return
		}
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
