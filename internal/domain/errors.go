package domain

import "errors"

var (
	// ErrUserNotFound is returned when a user lookup matches no row.
	ErrUserNotFound = errors.New("user not found")
	// ErrDuplicateEmail is returned when creating a user whose email already exists.
	ErrDuplicateEmail = errors.New("email already exists")
	// ErrMagicLinkNotFound is returned when a magic link lookup matches no row,
	// including when marking consumed a link that is missing or already consumed.
	ErrMagicLinkNotFound = errors.New("magic link not found")
	// ErrMagicLinkExpired is returned when verifying a magic link past its expiry.
	ErrMagicLinkExpired = errors.New("magic link expired")
	// ErrMagicLinkConsumed is returned when verifying a magic link that was already used.
	ErrMagicLinkConsumed = errors.New("magic link already consumed")
	// ErrGiftNotFound is returned when a gift lookup matches no row.
	ErrGiftNotFound = errors.New("gift not found")
	// ErrDuplicateViewToken is returned when creating a gift whose view token
	// already exists (the unique constraint backstops the generator).
	ErrDuplicateViewToken = errors.New("view token already exists")
)
