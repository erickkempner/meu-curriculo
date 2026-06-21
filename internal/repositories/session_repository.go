package repositories

import (
	"context"

	"github.com/erick/curriculo/internal/db"
)

// SessionRepository defines the contract for session persistence operations.
type SessionRepository interface {
	Create(ctx context.Context, params db.CreateSessionParams) (db.Session, error)
	FindByToken(ctx context.Context, token string) (db.Session, error)
	Delete(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
}

// sessionRepository implements SessionRepository using SQLC-generated queries.
type sessionRepository struct {
	queries *db.Queries
}

// NewSessionRepository creates a new SessionRepository backed by the given SQLC queries.
func NewSessionRepository(queries *db.Queries) SessionRepository {
	return &sessionRepository{queries: queries}
}

func (r *sessionRepository) Create(ctx context.Context, params db.CreateSessionParams) (db.Session, error) {
	return r.queries.CreateSession(ctx, params)
}

func (r *sessionRepository) FindByToken(ctx context.Context, token string) (db.Session, error) {
	return r.queries.FindSessionByToken(ctx, token)
}

func (r *sessionRepository) Delete(ctx context.Context, token string) error {
	return r.queries.DeleteSession(ctx, token)
}

func (r *sessionRepository) DeleteExpired(ctx context.Context) error {
	return r.queries.DeleteExpiredSessions(ctx)
}
