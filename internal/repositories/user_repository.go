package repositories

import (
	"context"

	"github.com/erick/curriculo/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserRepository defines the persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, params db.CreateUserParams) (db.User, error)
	FindByEmail(ctx context.Context, email string) (db.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (db.User, error)
}

// userRepository implements UserRepository using SQLC-generated queries.
type userRepository struct {
	queries *db.Queries
}

// NewUserRepository creates a new UserRepository backed by SQLC queries.
func NewUserRepository(queries *db.Queries) UserRepository {
	return &userRepository{queries: queries}
}

func (r *userRepository) Create(ctx context.Context, params db.CreateUserParams) (db.User, error) {
	return r.queries.CreateUser(ctx, params)
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (db.User, error) {
	return r.queries.FindUserByEmail(ctx, email)
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	pgID := pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
	return r.queries.FindUserByID(ctx, pgID)
}
