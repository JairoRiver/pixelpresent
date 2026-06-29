// Package api exposes the HTTP surface of pixelpresent: a chi router and its
// handlers, wired over the application services.
package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/gifts"
)

// AuthService is the slice of the auth service the API depends on. *auth.Service
// satisfies it.
type AuthService interface {
	RequestMagicLink(ctx context.Context, email string) error
	VerifyMagicLink(ctx context.Context, token string) (domain.User, error)
}

// Sessions issues the session cookie and guards protected routes.
// *auth.SessionManager satisfies it.
type Sessions interface {
	SetCookie(w http.ResponseWriter, userID uuid.UUID)
	RequireSession(next http.Handler) http.Handler
}

// GiftService is the slice of the gift service the API depends on. *gifts.Service
// satisfies it.
type GiftService interface {
	Create(ctx context.Context, in gifts.CreateInput) (domain.Gift, error)
	GetOwned(ctx context.Context, id, ownerID uuid.UUID) (domain.Gift, error)
}

// Server holds the dependencies of the HTTP handlers and builds the router.
type Server struct {
	auth     AuthService
	sessions Sessions
	gifts    GiftService
}

// NewServer builds the API server over its service dependencies.
func NewServer(auth AuthService, sessions Sessions, gifts GiftService) *Server {
	return &Server{auth: auth, sessions: sessions, gifts: gifts}
}

// Routes builds the chi router with every route mounted.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/magic-link", s.handleRequestMagicLink)
		r.Get("/verify", s.handleVerify)
	})

	// Routes requiring an authenticated creator session.
	r.Group(func(r chi.Router) {
		r.Use(s.sessions.RequireSession)
		r.Post("/gifts", s.handleCreateGift)
		r.Get("/gifts/{id}", s.handleGetGift)
	})

	return r
}
