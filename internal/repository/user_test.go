package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/repository/db/dbtest"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserRepo_Create(t *testing.T) {
	repo := NewUserRepo(dbtest.Tx(t))
	email := util.RandomEmail()

	u, err := repo.Create(context.Background(), email)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, u.ID)
	require.Equal(t, email, u.Email)
	require.False(t, u.CreatedAt.IsZero())
}

func TestUserRepo_CreateDuplicateEmail(t *testing.T) {
	repo := NewUserRepo(dbtest.Tx(t))
	email := util.RandomEmail()

	_, err := repo.Create(context.Background(), email)
	require.NoError(t, err)

	_, err = repo.Create(context.Background(), email)
	require.ErrorIs(t, err, domain.ErrDuplicateEmail)
}

func TestUserRepo_GetByID(t *testing.T) {
	repo := NewUserRepo(dbtest.Tx(t))
	created, err := repo.Create(context.Background(), util.RandomEmail())
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Email, got.Email)
	require.True(t, created.CreatedAt.Equal(got.CreatedAt))
}

func TestUserRepo_GetByEmailCaseInsensitive(t *testing.T) {
	repo := NewUserRepo(dbtest.Tx(t))
	created, err := repo.Create(context.Background(), util.RandomEmail())
	require.NoError(t, err)

	// email is citext, so the lookup ignores case.
	got, err := repo.GetByEmail(context.Background(), strings.ToUpper(created.Email))
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
}

func TestUserRepo_GetByIDNotFound(t *testing.T) {
	repo := NewUserRepo(dbtest.Tx(t))

	_, err := repo.GetByID(context.Background(), uuid.Nil)
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestUserRepo_GetByEmailNotFound(t *testing.T) {
	repo := NewUserRepo(dbtest.Tx(t))

	_, err := repo.GetByEmail(context.Background(), util.RandomEmail())
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}
