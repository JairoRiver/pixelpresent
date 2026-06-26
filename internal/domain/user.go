package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User is a registered account, identified by a unique (case-insensitive) email.
type User struct {
	ID        uuid.UUID
	Email     string
	CreatedAt time.Time
}

// UserRepository persists and retrieves users.
type UserRepository interface {
	Create(ctx context.Context, email string) (User, error)
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
}
