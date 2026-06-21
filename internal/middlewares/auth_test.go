package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erick/curriculo/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequireAuth_NoCookie_RedirectsToLogin(t *testing.T) {
	mock := &mockSessionService{}
	mw := NewAuthMiddleware(mock)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(mw.RequireAuth())
	r.GET("/resumes", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	c.Request = httptest.NewRequest(http.MethodGet, "/resumes", nil)
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, w.Code)
	}

	location := w.Header().Get("Location")
	expected := "/login?redirect=%2Fresumes"
	if location != expected {
		t.Errorf("expected redirect to %q, got %q", expected, location)
	}
}

func TestRequireAuth_InvalidToken_RedirectsToLogin(t *testing.T) {
	mock := &mockSessionService{
		validateFn: func(_ context.Context, _ string) (*models.Session, error) {
			return nil, errors.New("session not found")
		},
	}
	mw := NewAuthMiddleware(mock)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(mw.RequireAuth())
	r.GET("/resumes", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/resumes", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "invalid-token"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, w.Code)
	}

	location := w.Header().Get("Location")
	expected := "/login?redirect=%2Fresumes"
	if location != expected {
		t.Errorf("expected redirect to %q, got %q", expected, location)
	}
}

func TestRequireAuth_ValidToken_SetsContextAndProceeds(t *testing.T) {
	userID := uuid.New()
	session := &models.Session{
		ID:     uuid.New(),
		UserID: userID,
		Token:  "valid-token",
	}

	mock := &mockSessionService{
		validateFn: func(_ context.Context, token string) (*models.Session, error) {
			if token == "valid-token" {
				return session, nil
			}
			return nil, errors.New("not found")
		},
	}
	mw := NewAuthMiddleware(mock)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var capturedUserID uuid.UUID
	var capturedSession *models.Session

	r.Use(mw.RequireAuth())
	r.GET("/resumes", func(c *gin.Context) {
		val, _ := c.Get(ContextKeyUserID)
		capturedUserID = val.(uuid.UUID)
		sessVal, _ := c.Get(ContextKeySession)
		capturedSession = sessVal.(*models.Session)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/resumes", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-token"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if capturedUserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, capturedUserID)
	}

	if capturedSession != session {
		t.Errorf("expected session to be set in context")
	}
}

func TestRequireAuth_PreservesQueryString(t *testing.T) {
	mock := &mockSessionService{}
	mw := NewAuthMiddleware(mock)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(mw.RequireAuth())
	r.GET("/resumes/:id/edit", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/resumes/123/edit?tab=experience", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, w.Code)
	}

	location := w.Header().Get("Location")
	expected := "/login?redirect=%2Fresumes%2F123%2Fedit%3Ftab%3Dexperience"
	if location != expected {
		t.Errorf("expected redirect to %q, got %q", expected, location)
	}
}

func TestOptionalAuth_NoCookie_ProceedsWithoutContext(t *testing.T) {
	mock := &mockSessionService{}
	mw := NewAuthMiddleware(mock)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var hasUserID bool

	r.Use(mw.OptionalAuth())
	r.GET("/r/some-token", func(c *gin.Context) {
		_, hasUserID = c.Get(ContextKeyUserID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/r/some-token", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if hasUserID {
		t.Error("expected no user ID in context when no cookie is provided")
	}
}

func TestOptionalAuth_InvalidToken_ProceedsWithoutContext(t *testing.T) {
	mock := &mockSessionService{
		validateFn: func(_ context.Context, _ string) (*models.Session, error) {
			return nil, errors.New("expired")
		},
	}
	mw := NewAuthMiddleware(mock)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var hasUserID bool

	r.Use(mw.OptionalAuth())
	r.GET("/r/some-token", func(c *gin.Context) {
		_, hasUserID = c.Get(ContextKeyUserID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/r/some-token", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "bad-token"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if hasUserID {
		t.Error("expected no user ID in context when token is invalid")
	}
}

func TestOptionalAuth_ValidToken_SetsContext(t *testing.T) {
	userID := uuid.New()
	session := &models.Session{
		ID:     uuid.New(),
		UserID: userID,
		Token:  "valid-token",
	}

	mock := &mockSessionService{
		validateFn: func(_ context.Context, token string) (*models.Session, error) {
			if token == "valid-token" {
				return session, nil
			}
			return nil, errors.New("not found")
		},
	}
	mw := NewAuthMiddleware(mock)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var capturedUserID uuid.UUID

	r.Use(mw.OptionalAuth())
	r.GET("/r/some-token", func(c *gin.Context) {
		val, _ := c.Get(ContextKeyUserID)
		capturedUserID = val.(uuid.UUID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/r/some-token", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-token"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if capturedUserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, capturedUserID)
	}
}

func TestGetUserIDFromContext_NoValue_ReturnsError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	id, err := GetUserIDFromContext(c)

	if !errors.Is(err, models.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}

	if id != uuid.Nil {
		t.Errorf("expected uuid.Nil, got %s", id)
	}
}

func TestGetUserIDFromContext_WrongType_ReturnsError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ContextKeyUserID, "not-a-uuid")

	id, err := GetUserIDFromContext(c)

	if !errors.Is(err, models.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}

	if id != uuid.Nil {
		t.Errorf("expected uuid.Nil, got %s", id)
	}
}

func TestGetUserIDFromContext_ValidUUID_ReturnsID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	expected := uuid.New()
	c.Set(ContextKeyUserID, expected)

	id, err := GetUserIDFromContext(c)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if id != expected {
		t.Errorf("expected %s, got %s", expected, id)
	}
}
