package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// SessionCookieName is the session cookie set after a magic link is verified.
	SessionCookieName = "pp_session"
	// sessionCookiePath is fixed to "/" so the cookie is sent on every path of
	// the (single-origin) app: the Go API and the embedded frontend alike. It is
	// intentionally not configurable: the only reason to narrow it would be
	// mounting the app under a sub-path, which the architecture rules out, and a
	// misconfigured path would silently break auth (cookie never sent → 401).
	sessionCookiePath = "/"
)

// errInvalidSession is returned by decode for any malformed, tampered, or
// expired cookie value. The middleware maps it to 401 without leaking which.
var errInvalidSession = errors.New("invalid session")

// contextKey is unexported to keep auth's context values private to this package.
type contextKey int

const userIDKey contextKey = iota

// SessionManager issues and verifies stateless, HMAC-signed session cookies.
// The cookie carries the user id and an expiry, signed with secret; there is no
// server-side session store.
type SessionManager struct {
	secret []byte
	secure bool
	ttl    time.Duration
}

// NewSessionManager builds a SessionManager. secure controls the cookie's Secure
// flag (true in production, false over plain HTTP in development); ttl is how
// long an issued session stays valid (from config: auth.session_ttl).
func NewSessionManager(secret string, secure bool, ttl time.Duration) *SessionManager {
	return &SessionManager{secret: []byte(secret), secure: secure, ttl: ttl}
}

// SetCookie writes a signed session cookie for userID onto w, valid for the
// manager's ttl.
func (m *SessionManager) SetCookie(w http.ResponseWriter, userID uuid.UUID) {
	expiresAt := time.Now().Add(m.ttl)
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    m.encode(userID, expiresAt),
		Path:     sessionCookiePath,
		Expires:  expiresAt,
		MaxAge:   int(m.ttl.Seconds()),
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie writes an expired session cookie onto w, ending the session.
func (m *SessionManager) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// RequireSession is chi-compatible middleware that rejects requests without a
// valid session cookie with 401 and, on success, stores the user id in the
// request context (read with UserIDFromContext).
func (m *SessionManager) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		userID, err := m.decode(cookie.Value)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext returns the authenticated user id stored by RequireSession.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

// encode builds the cookie value "base64(payload).base64(hmac)" where payload is
// "<user_id>|<expiry_unix>".
func (m *SessionManager) encode(userID uuid.UUID, expiresAt time.Time) string {
	payload := userID.String() + "|" + strconv.FormatInt(expiresAt.Unix(), 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + m.sign(payload)
}

// decode verifies the signature and expiry of a cookie value and returns the
// user id, or errInvalidSession if it is malformed, tampered, or expired.
func (m *SessionManager) decode(value string) (uuid.UUID, error) {
	encodedPayload, sig, ok := strings.Cut(value, ".")
	if !ok {
		return uuid.Nil, errInvalidSession
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return uuid.Nil, errInvalidSession
	}
	payload := string(payloadBytes)

	if !hmac.Equal([]byte(sig), []byte(m.sign(payload))) {
		return uuid.Nil, errInvalidSession
	}

	rawID, rawExp, ok := strings.Cut(payload, "|")
	if !ok {
		return uuid.Nil, errInvalidSession
	}

	userID, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, errInvalidSession
	}

	exp, err := strconv.ParseInt(rawExp, 10, 64)
	if err != nil {
		return uuid.Nil, errInvalidSession
	}
	if time.Now().After(time.Unix(exp, 0)) {
		return uuid.Nil, errInvalidSession
	}

	return userID, nil
}

// sign returns the base64url HMAC-SHA256 of payload under the manager's secret.
func (m *SessionManager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// writeUnauthorized emits the project's standard error envelope with status 401.
func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	// Static payload: no user input is echoed back.
	_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"se requiere una sesión válida"}}`))
}
