// Package auth implements the magic-link login flow: requesting a link by
// email and verifying it. This authenticates the creator of gifts; it is
// unrelated to the gift view links shared with recipients.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/domain"
)

const (
	// verifyPath is the path of the verification endpoint (see PP-19).
	verifyPath = "/auth/verify"
	// magicLinkSubject is the subject of the login email.
	magicLinkSubject = "Tu enlace de acceso a Pixel Present"
)

// Service orchestrates the magic-link flow over the user/link repositories and
// the email sender.
type Service struct {
	users   domain.UserRepository
	links   domain.MagicLinkRepository
	emails  domain.EmailSender
	baseURL string
	ttl     time.Duration
}

// NewService builds the auth service. ttl is how long a requested magic link
// stays valid (from config: auth.magic_link_ttl).
func NewService(users domain.UserRepository, links domain.MagicLinkRepository, emails domain.EmailSender, baseURL string, ttl time.Duration) *Service {
	return &Service{
		users:   users,
		links:   links,
		emails:  emails,
		baseURL: strings.TrimRight(baseURL, "/"),
		ttl:     ttl,
	}
}

// RequestMagicLink creates or retrieves the user for email, generates a secure
// single-use token, stores its hash with a short expiry, and emails the link.
func (s *Service) RequestMagicLink(ctx context.Context, email string) error {
	email = strings.TrimSpace(email)

	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, domain.ErrUserNotFound) {
		user, err = s.users.Create(ctx, email)
	}
	if err != nil {
		return err
	}

	token, err := generateToken()
	if err != nil {
		return err
	}

	if _, err := s.links.Create(ctx, user.ID, hashToken(token), time.Now().Add(s.ttl)); err != nil {
		return err
	}

	text, html, err := renderMagicLink(magicLinkData{
		Link:   s.verifyURL(token),
		Expiry: humanTTL(s.ttl),
	})
	if err != nil {
		return err
	}

	return s.emails.Send(ctx, domain.Email{
		To:       email,
		Subject:  magicLinkSubject,
		BodyText: text,
		BodyHTML: html,
	})
}

func (s *Service) verifyURL(token string) string {
	return s.baseURL + verifyPath + "?token=" + url.QueryEscape(token)
}

// humanTTL renders a duration for end users, e.g. "15 minutos" or "1 hora".
func humanTTL(d time.Duration) string {
	if d >= time.Hour && d%time.Hour == 0 {
		h := int(d / time.Hour)
		if h == 1 {
			return "1 hora"
		}
		return fmt.Sprintf("%d horas", h)
	}
	m := int(d.Round(time.Minute) / time.Minute)
	if m == 1 {
		return "1 minuto"
	}
	return fmt.Sprintf("%d minutos", m)
}
