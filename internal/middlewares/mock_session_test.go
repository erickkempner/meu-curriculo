package middlewares

import (
	"context"

	"github.com/erick/curriculo/internal/models"
	"github.com/google/uuid"
)

// mockSessionService is a shared test double for services.SessionService
// used by both auth and CSRF middleware tests.
type mockSessionService struct {
	// validateFn allows per-test customization of Validate behavior.
	validateFn func(ctx context.Context, token string) (*models.Session, error)
	// csrfToken is the token returned by GetCSRFToken.
	csrfToken string
	// csrfErr is the error returned by GetCSRFToken.
	csrfErr error
	// validateResult is the boolean returned by ValidateCSRFToken.
	validateResult bool
}

func (m *mockSessionService) Create(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}

func (m *mockSessionService) Validate(ctx context.Context, token string) (*models.Session, error) {
	if m.validateFn != nil {
		return m.validateFn(ctx, token)
	}
	return nil, nil
}

func (m *mockSessionService) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *mockSessionService) GetCSRFToken(_ context.Context, _ string) (string, error) {
	return m.csrfToken, m.csrfErr
}

func (m *mockSessionService) ValidateCSRFToken(_ context.Context, _, _ string) bool {
	return m.validateResult
}
