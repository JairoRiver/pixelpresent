// Package api exposes the HTTP surface of pixelpresent: a chi router and its
// handlers, wired over the application services.
package api

import (
	"context"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/gifts"
	"github.com/JairoRiver/pixelpresent/internal/reactions"
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
	ClearCookie(w http.ResponseWriter)
	RequireSession(next http.Handler) http.Handler
}

// GiftService is the slice of the gift service the API depends on. *gifts.Service
// satisfies it.
type GiftService interface {
	Create(ctx context.Context, in gifts.CreateInput) (domain.Gift, error)
	GetOwned(ctx context.Context, id, ownerID uuid.UUID) (domain.Gift, error)
	UpdateOwned(ctx context.Context, id, ownerID uuid.UUID, in gifts.UpdateInput) (domain.Gift, error)
	Publish(ctx context.Context, id, ownerID uuid.UUID) (domain.Gift, error)
	DeleteOwned(ctx context.Context, id, ownerID uuid.UUID) error
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Gift, error)
	GetByViewToken(ctx context.Context, token string) (domain.Gift, error)
	MarkOpened(ctx context.Context, token string) error
}

// ReactionService is the slice of the reaction service the API depends on.
// *reactions.Service satisfies it.
type ReactionService interface {
	Create(ctx context.Context, in reactions.CreateInput) (domain.Reaction, error)
	ListForOwner(ctx context.Context, giftID, ownerID uuid.UUID) ([]domain.Reaction, error)
}

// Server holds the dependencies of the HTTP handlers and builds the router.
type Server struct {
	auth      AuthService
	sessions  Sessions
	gifts     GiftService
	reactions ReactionService

	// docsEnabled mounts the OpenAPI docs routes; set via EnableDocs (dev only).
	docsEnabled bool
	// static serves the embedded frontend as a fallback; set via ServeStatic.
	static fs.FS
}

// NewServer builds the API server over its service dependencies.
func NewServer(auth AuthService, sessions Sessions, gifts GiftService, reactions ReactionService) *Server {
	return &Server{auth: auth, sessions: sessions, gifts: gifts, reactions: reactions}
}

// Routes builds the chi router with every route mounted.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// The JSON API lives under /api so the static frontend can own the clean URLs
	// (/, /dashboard, /g/{token}) at the same origin in production.
	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/magic-link", s.handleRequestMagicLink)
			r.Get("/verify", s.handleVerify)
			// Logout just clears the cookie; it works whether or not the caller
			// still has a valid session.
			r.Post("/logout", s.handleLogout)
		})

		// Public recipient-facing routes addressed by the shareable token (no session).
		r.Get("/g/{view_token}", s.handleViewGift)
		r.Post("/g/{view_token}/opened", s.handleMarkGiftOpened)
		r.Post("/g/{view_token}/reactions", s.handleCreateReaction)

		// Routes requiring an authenticated creator session.
		r.Group(func(r chi.Router) {
			r.Use(s.sessions.RequireSession)
			r.Get("/auth/me", s.handleMe)
			r.Post("/gifts", s.handleCreateGift)
			r.Get("/gifts", s.handleListGifts)
			r.Get("/gifts/{id}", s.handleGetGift)
			r.Put("/gifts/{id}", s.handleUpdateGift)
			r.Post("/gifts/{id}/publish", s.handlePublishGift)
			r.Delete("/gifts/{id}", s.handleDeleteGift)
			r.Get("/gifts/{id}/reactions", s.handleListReactions)
		})

		// Development-only API docs, mounted only when explicitly enabled.
		if s.docsEnabled {
			r.Get("/docs", s.handleDocsUI)
			r.Get("/docs/openapi.yaml", s.handleOpenAPISpec)
		}
	})

	// Serve the embedded frontend for any path the API does not handle (root and
	// every non-/api route): /, /_astro/*, ...
	if s.static != nil {
		// The public reveal URL /g/{view_token} carries a token, not a file, so the
		// file server below would 404 on it. Serve the reveal document explicitly;
		// the page reads the token from the URL and calls GET /api/g/{view_token}.
		r.Get("/g/{view_token}", s.handleRevealPage)

		fileServer := http.FileServerFS(s.static)
		r.NotFound(fileServer.ServeHTTP)
	}

	return r
}

// handleRevealPage serves the public reveal document (Astro's g/index.html) for
// any tokenized /g/{view_token} browser URL. See the route comment above.
func (s *Server) handleRevealPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, s.static, "g/index.html")
}

// ServeStatic mounts fsys (the embedded frontend) as the fallback for any route
// the API does not handle. Like EnableDocs, it is a wiring-time switch kept off
// NewServer so adding it does not churn every call site.
func (s *Server) ServeStatic(fsys fs.FS) {
	s.static = fsys
}
