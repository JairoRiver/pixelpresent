package sqlc

import (
	"context"
	"strings"
	"testing"

	"github.com/JairoRiver/pixelpresent/internal/repository/db/dbtest"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// createRandomUser creates a user with random data for test purposes. It runs
// on the caller's transactional Queries so it can be reused by tests of other
// tables that reference users, sharing the same rolled-back transaction.
func createRandomUser(t *testing.T, q *Queries) User {
	t.Helper()

	email := util.RandomEmail()

	user, err := q.CreateUser(context.Background(), email)
	require.NoError(t, err)
	require.Equal(t, email, user.Email)
	require.True(t, user.ID.Valid)
	require.True(t, user.CreatedAt.Valid)

	return user
}

func TestCreateUser(t *testing.T) {
	q := New(dbtest.Tx(t))

	_ = createRandomUser(t, q)
}

func TestGetUser_ByID(t *testing.T) {
	q := New(dbtest.Tx(t))

	created := createRandomUser(t, q)

	got, err := q.GetUser(context.Background(), GetUserParams{ID: created.ID})
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Email, got.Email)
}

func TestGetUser_ByEmail(t *testing.T) {
	q := New(dbtest.Tx(t))

	created := createRandomUser(t, q)

	got, err := q.GetUser(context.Background(), GetUserParams{
		Email: pgtype.Text{String: created.Email, Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
}

func TestGetUser_ByEmailIsCaseInsensitive(t *testing.T) {
	q := New(dbtest.Tx(t))

	created := createRandomUser(t, q)

	// email is citext, so the match must ignore case.
	got, err := q.GetUser(context.Background(), GetUserParams{
		Email: pgtype.Text{String: strings.ToUpper(created.Email), Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
}

func TestGetUser_NoFilterReturnsNoRows(t *testing.T) {
	q := New(dbtest.Tx(t))

	_ = createRandomUser(t, q)

	// With neither id nor email set, the guard prevents matching any row.
	_, err := q.GetUser(context.Background(), GetUserParams{})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
