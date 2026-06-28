// Package gifts implements gift creation and retrieval logic, including the
// generation of the shareable view token.
package gifts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

const (
	// viewTokenBytes is the entropy (before base64url encoding) of a view token.
	// 32 bytes = 256 bits from a CSPRNG: unguessable and collision-free in
	// practice. The token is a bearer capability (whoever holds the link can open
	// the gift), so unpredictability is what matters; we deliberately mix in no
	// timestamp, which would be guessable and would leak the creation time.
	viewTokenBytes = 32
	// maxViewTokenAttempts bounds the retry loop on the (astronomically unlikely)
	// collision so a misbehaving checker cannot spin forever.
	maxViewTokenAttempts = 5
)

// ErrViewTokenExhausted is returned when a unique token could not be found
// within maxViewTokenAttempts. With 32 random bytes a real collision is
// negligible, so in practice this signals a broken uniqueness checker.
var ErrViewTokenExhausted = errors.New("could not generate a unique view token")

// ViewTokenChecker reports whether a candidate view token is already taken.
// The gifts repository satisfies it.
type ViewTokenChecker interface {
	ViewTokenExists(ctx context.Context, token string) (bool, error)
}

// GenerateViewToken returns a fresh, cryptographically random view token that is
// unique against checker, retrying on the (vanishingly rare) collision.
func GenerateViewToken(ctx context.Context, checker ViewTokenChecker) (string, error) {
	for attempt := 0; attempt < maxViewTokenAttempts; attempt++ {
		token, err := randomViewToken()
		if err != nil {
			return "", err
		}

		exists, err := checker.ViewTokenExists(ctx, token)
		if err != nil {
			return "", err
		}
		if !exists {
			return token, nil
		}
	}
	return "", ErrViewTokenExhausted
}

// randomViewToken returns a base64url-encoded token of viewTokenBytes of entropy.
func randomViewToken() (string, error) {
	b := make([]byte, viewTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
