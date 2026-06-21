package services

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/erick/curriculo/internal/config"
	"github.com/erick/curriculo/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// mockSessionRepository implements repositories.SessionRepository for testing.
type mockSessionRepository struct {
	sessions map[string]db.Session
	createFn func(ctx context.Context, params db.CreateSessionParams) (db.Session, error)
}

func newMockSessionRepository() *mockSessionRepository {
	return &mockSessionRepository{
		sessions: make(map[string]db.Session),
	}
}

func (m *mockSessionRepository) Create(ctx context.Context, params db.CreateSessionParams) (db.Session, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	session := db.Session{
		ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
		UserID:    params.UserID,
		Token:     params.Token,
		CsrfToken: params.CsrfToken,
		ExpiresAt: params.ExpiresAt,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	m.sessions[params.Token] = session
	return session, nil
}

func (m *mockSessionRepository) FindByToken(ctx context.Context, token string) (db.Session, error) {
	session, ok := m.sessions[token]
	if !ok {
		return db.Session{}, errors.New("no rows in result set")
	}
	return session, nil
}

func (m *mockSessionRepository) Delete(ctx context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

func (m *mockSessionRepository) DeleteExpired(ctx context.Context) error {
	for token, session := range m.sessions {
		if session.ExpiresAt.Time.Before(time.Now()) {
			delete(m.sessions, token)
		}
	}
	return nil
}

func TestGenerateToken(t *testing.T) {
	token, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() error: %v", err)
	}

	// Token should be 64 hex characters (32 bytes hex-encoded)
	if len(token) != 64 {
		t.Errorf("expected token length 64, got %d", len(token))
	}

	// Token should be valid hex
	_, err = hex.DecodeString(token)
	if err != nil {
		t.Errorf("token is not valid hex: %v", err)
	}

	// Two generated tokens should be different
	token2, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() second call error: %v", err)
	}
	if token == token2 {
		t.Error("two generated tokens should be different")
	}
}

func TestSessionService_Create(t *testing.T) {
	repo := newMockSessionRepository()
	cfg := config.SessionConfig{
		Secret:     "test-secret",
		MaxAge:     7 * 24 * time.Hour,
		CookieName: "session_token",
	}
	svc := NewSessionService(repo, cfg)

	userID := uuid.New()
	token, err := svc.Create(context.Background(), userID)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Token should be 64 hex characters
	if len(token) != 64 {
		t.Errorf("expected token length 64, got %d", len(token))
	}

	// Session should be stored in repository
	if len(repo.sessions) != 1 {
		t.Fatalf("expected 1 session in repo, got %d", len(repo.sessions))
	}

	stored := repo.sessions[token]
	if stored.UserID.Bytes != userID {
		t.Errorf("expected user ID %s, got %s", userID, stored.UserID.Bytes)
	}

	// CSRF token should also be 64 hex chars
	if len(stored.CsrfToken) != 64 {
		t.Errorf("expected CSRF token length 64, got %d", len(stored.CsrfToken))
	}

	// Tokens should be different from each other
	if stored.Token == stored.CsrfToken {
		t.Error("session token and CSRF token should be different")
	}

	// Expiration should be approximately 7 days from now
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	diff := stored.ExpiresAt.Time.Sub(expectedExpiry)
	if diff > time.Minute || diff < -time.Minute {
		t.Errorf("expiration time off by %v", diff)
	}
}

func TestSessionService_Create_DefaultMaxAge(t *testing.T) {
	repo := newMockSessionRepository()
	cfg := config.SessionConfig{
		Secret:     "test-secret",
		MaxAge:     0, // should default to 7 days
		CookieName: "session_token",
	}
	svc := NewSessionService(repo, cfg)

	token, err := svc.Create(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	stored := repo.sessions[token]
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	diff := stored.ExpiresAt.Time.Sub(expectedExpiry)
	if diff > time.Minute || diff < -time.Minute {
		t.Errorf("default expiration time off by %v", diff)
	}
}

func TestSessionService_Create_RepoError(t *testing.T) {
	repo := newMockSessionRepository()
	repo.createFn = func(ctx context.Context, params db.CreateSessionParams) (db.Session, error) {
		return db.Session{}, errors.New("db connection error")
	}
	cfg := config.SessionConfig{
		Secret:     "test-secret",
		MaxAge:     7 * 24 * time.Hour,
		CookieName: "session_token",
	}
	svc := NewSessionService(repo, cfg)

	_, err := svc.Create(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error from Create when repo fails")
	}
}

func TestSessionService_Validate(t *testing.T) {
	repo := newMockSessionRepository()
	cfg := config.SessionConfig{
		Secret:     "test-secret",
		MaxAge:     7 * 24 * time.Hour,
		CookieName: "session_token",
	}
	svc := NewSessionService(repo, cfg)

	userID := uuid.New()
	token, err := svc.Create(context.Background(), userID)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	session, err := svc.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if session.UserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, session.UserID)
	}
	if session.Token != token {
		t.Errorf("expected token %s, got %s", token, session.Token)
	}
}

func TestSessionService_Validate_InvalidToken(t *testing.T) {
	repo := newMockSessionRepository()
	cfg := config.SessionConfig{
		Secret:     "test-secret",
		MaxAge:     7 * 24 * time.Hour,
		CookieName: "session_token",
	}
	svc := NewSessionService(repo, cfg)

	_, err := svc.Validate(context.Background(), "nonexistent-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestSessionService_Delete(t *testing.T) {
	repo := newMockSessionRepository()
	cfg := config.SessionConfig{
		Secret:     "test-secret",
		MaxAge:     7 * 24 * time.Hour,
		CookieName: "session_token",
	}
	svc := NewSessionService(repo, cfg)

	userID := uuid.New()
	token, err := svc.Create(context.Background(), userID)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	err = svc.Delete(context.Background(), token)
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Session should no longer be findable
	_, err = svc.Validate(context.Background(), token)
	if err == nil {
		t.Fatal("expected error after deleting session")
	}
}

func TestSessionService_GetCSRFToken(t *testing.T) {
	repo := newMockSessionRepository()
	cfg := config.SessionConfig{
		Secret:     "test-secret",
		MaxAge:     7 * 24 * time.Hour,
		CookieName: "session_token",
	}
	svc := NewSessionService(repo, cfg)

	userID := uuid.New()
	token, err := svc.Create(context.Background(), userID)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	csrfToken, err := svc.GetCSRFToken(context.Background(), token)
	if err != nil {
		t.Fatalf("GetCSRFToken() error: %v", err)
	}

	if len(csrfToken) != 64 {
		t.Errorf("expected CSRF token length 64, got %d", len(csrfToken))
	}
}

func TestSessionService_GetCSRFToken_InvalidSession(t *testing.T) {
	repo := newMockSessionRepository()
	cfg := config.SessionConfig{
		Secret:     "test-secret",
		MaxAge:     7 * 24 * time.Hour,
		CookieName: "session_token",
	}
	svc := NewSessionService(repo, cfg)

	_, err := svc.GetCSRFToken(context.Background(), "nonexistent-token")
	if err == nil {
		t.Fatal("expected error for invalid session token")
	}
}

func TestSessionService_ValidateCSRFToken(t *testing.T) {
	repo := newMockSessionRepository()
	cfg := config.SessionConfig{
		Secret:     "test-secret",
		MaxAge:     7 * 24 * time.Hour,
		CookieName: "session_token",
	}
	svc := NewSessionService(repo, cfg)

	userID := uuid.New()
	token, err := svc.Create(context.Background(), userID)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	csrfToken, err := svc.GetCSRFToken(context.Background(), token)
	if err != nil {
		t.Fatalf("GetCSRFToken() error: %v", err)
	}

	// Valid CSRF token should pass
	if !svc.ValidateCSRFToken(context.Background(), token, csrfToken) {
		t.Error("ValidateCSRFToken should return true for valid token")
	}

	// Invalid CSRF token should fail
	if svc.ValidateCSRFToken(context.Background(), token, "invalid-csrf-token") {
		t.Error("ValidateCSRFToken should return false for invalid token")
	}

	// Invalid session token should fail
	if svc.ValidateCSRFToken(context.Background(), "invalid-session", csrfToken) {
		t.Error("ValidateCSRFToken should return false for invalid session")
	}
}
