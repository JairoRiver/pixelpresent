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
)
