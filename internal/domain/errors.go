package domain

import "errors"

var (
	// ErrUserNotFound is returned when a user lookup matches no row.
	ErrUserNotFound = errors.New("user not found")
	// ErrDuplicateEmail is returned when creating a user whose email already exists.
	ErrDuplicateEmail = errors.New("email already exists")
)
