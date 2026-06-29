package api

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/JairoRiver/pixelpresent/internal/auth"
	"github.com/JairoRiver/pixelpresent/internal/gifts"
)

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
