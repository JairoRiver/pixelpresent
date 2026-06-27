// Package api exposes the HTTP surface of pixelpresent: a chi router and its
// handlers, wired over the application services.
package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// MagicLinkRequester is the slice of the auth service the API needs to request a
// login link. *auth.Service satisfies it.
type MagicLinkRequester interface {
	RequestMagicLink(ctx context.Context, email string) error
}

// Server holds the dependencies of the HTTP handlers and builds the router.
type Server struct {
	auth MagicLinkRequester
}

// NewServer builds the API server over its service dependencies.
func NewServer(auth MagicLinkRequester) *Server {
	return &Server{auth: auth}
}

// Routes builds the chi router with every route mounted.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/magic-link", s.handleRequestMagicLink)
	})

	return r
}
