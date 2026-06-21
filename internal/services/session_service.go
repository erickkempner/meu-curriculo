package services

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/erick/curriculo/internal/config"
	"github.com/erick/curriculo/internal/db"
	"github.com/erick/curriculo/internal/models"
	"github.com/erick/curriculo/internal/repositories"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// SessionService defines the contract for session management operations.
type SessionService interface {
	Create(ctx context.Context, userID uuid.UUID) (token string, err error)
	Validate(ctx context.Context, token string) (*models.Session, error)
	Delete(ctx context.Context, token string) error
	GetCSRFToken(ctx context.Context, sessionToken string) (string, error)
	ValidateCSRFToken(ctx context.Context, sessionToken, csrfToken string) bool
}

// sessionService implements SessionService using a SessionRepository and SessionConfig.
type sessionService struct {
	repo repositories.SessionRepository
	cfg  config.SessionConfig
}

// NewSessionService creates a new SessionService backed by the given repository and config.
func NewSessionService(repo repositories.SessionRepository, cfg config.SessionConfig) SessionService {
	return &sessionService{repo: repo, cfg: cfg}
}

// Create generates a new session for the given user with cryptographically random tokens.
func (s *sessionService) Create(ctx context.Context, userID uuid.UUID) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("session create: %w", err)
	}

	csrfToken, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("session create csrf: %w", err)
	}

	maxAge := s.cfg.MaxAge
	if maxAge == 0 {
		maxAge = 7 * 24 * time.Hour
	}

	expiresAt := time.Now().Add(maxAge)

	params := db.CreateSessionParams{
		UserID: pgtype.UUID{
			Bytes: userID,
			Valid: true,
		},
		Token:     token,
		CsrfToken: csrfToken,
		ExpiresAt: pgtype.Timestamptz{
			Time:  expiresAt,
			Valid: true,
		},
	}

	_, err = s.repo.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("session create: %w", err)
	}

	return token, nil
}

// Validate finds a session by token and returns it. Returns an error if the session
// is not found or expired (expiration is enforced by the SQL query WHERE expires_at > NOW()).
func (s *sessionService) Validate(ctx context.Context, token string) (*models.Session, error) {
	dbSession, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("session validate: %w", err)
	}

	session := mapDBSessionToModel(dbSession)
	return session, nil
}

// Delete removes a session from the database by its token.
func (s *sessionService) Delete(ctx context.Context, token string) error {
	if err := s.repo.Delete(ctx, token); err != nil {
		return fmt.Errorf("session delete: %w", err)
	}
	return nil
}

// GetCSRFToken retrieves the CSRF token associated with the given session token.
func (s *sessionService) GetCSRFToken(ctx context.Context, sessionToken string) (string, error) {
	dbSession, err := s.repo.FindByToken(ctx, sessionToken)
	if err != nil {
		return "", fmt.Errorf("session get csrf token: %w", err)
	}

	return dbSession.CsrfToken, nil
}

// ValidateCSRFToken compares the provided CSRF token against the stored one using
// constant-time comparison to prevent timing attacks.
func (s *sessionService) ValidateCSRFToken(ctx context.Context, sessionToken, csrfToken string) bool {
	dbSession, err := s.repo.FindByToken(ctx, sessionToken)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(dbSession.CsrfToken), []byte(csrfToken)) == 1
}

// generateToken creates a 64-character hex string from 32 cryptographically random bytes.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// mapDBSessionToModel converts a SQLC-generated db.Session to a domain models.Session.
func mapDBSessionToModel(dbSession db.Session) *models.Session {
	return &models.Session{
		ID:        dbSession.ID.Bytes,
		UserID:    dbSession.UserID.Bytes,
		Token:     dbSession.Token,
		CSRFToken: dbSession.CsrfToken,
		ExpiresAt: dbSession.ExpiresAt.Time,
		CreatedAt: dbSession.CreatedAt.Time,
	}
}
