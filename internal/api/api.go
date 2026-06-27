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
)

// AuthService is the slice of the auth service the API depends on. *auth.Service
// satisfies it.
type AuthService interface {
	RequestMagicLink(ctx context.Context, email string) error
	VerifyMagicLink(ctx context.Context, token string) (domain.User, error)
}

// SessionWriter issues the session cookie after a successful verification.
// *auth.SessionManager satisfies it.
type SessionWriter interface {
	SetCookie(w http.ResponseWriter, userID uuid.UUID)
}

// Server holds the dependencies of the HTTP handlers and builds the router.
type Server struct {
	auth     AuthService
	sessions SessionWriter
}

// NewServer builds the API server over its service dependencies.
func NewServer(auth AuthService, sessions SessionWriter) *Server {
	return &Server{auth: auth, sessions: sessions}
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

	return r
}
