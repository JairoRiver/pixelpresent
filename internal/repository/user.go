package repository

import (
	"context"
	"errors"

	"github.com/JairoRiver/pixelpresent/internal/domain"
	"github.com/JairoRiver/pixelpresent/internal/repository/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const uniqueViolationCode = "23505"

// UserRepo is a domain.UserRepository backed by sqlc-generated queries.
type UserRepo struct {
	q *sqlc.Queries
}

var _ domain.UserRepository = (*UserRepo)(nil)

// NewUserRepo builds a UserRepo over any sqlc.DBTX (a *pgxpool.Pool in
// production, a pgx.Tx in tests).
func NewUserRepo(db sqlc.DBTX) *UserRepo {
	return &UserRepo{q: sqlc.New(db)}
}

func (r *UserRepo) Create(ctx context.Context, email string) (domain.User, error) {
	u, err := r.q.CreateUser(ctx, email)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, domain.ErrDuplicateEmail
		}
		return domain.User{}, err
	}
	return toDomainUser(u), nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	u, err := r.q.GetUser(ctx, sqlc.GetUserParams{ID: uuid.NullUUID{UUID: id, Valid: true}})
	return mapGetUser(u, err)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := r.q.GetUser(ctx, sqlc.GetUserParams{Email: pgtype.Text{String: email, Valid: true}})
	return mapGetUser(u, err)
}

func mapGetUser(u sqlc.User, err error) (domain.User, error) {
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}
	return toDomainUser(u), nil
}

func toDomainUser(u sqlc.User) domain.User {
	return domain.User{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == uniqueViolationCode
	}
	return false
}
