package middlewares

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupCSRFRouter(mock *mockSessionService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := NewCSRFMiddleware(mock)
	r.Use(mw.Protect())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.PUT("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.DELETE("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

func TestProtect_SafeMethod_EmbedTokenInContext(t *testing.T) {
	mock := &mockSessionService{csrfToken: "test-csrf-token-123"}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := NewCSRFMiddleware(mock)
	r.Use(mw.Protect())

	var gotToken string
	r.GET("/test", func(c *gin.Context) {
		gotToken = GetCSRFToken(c)
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-session"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if gotToken != "test-csrf-token-123" {
		t.Errorf("expected csrf token 'test-csrf-token-123', got '%s'", gotToken)
	}
}

func TestProtect_SafeMethod_NoSessionCookie_Proceeds(t *testing.T) {
	mock := &mockSessionService{}
	router := setupCSRFRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestProtect_UnsafeMethod_MissingToken_Returns403(t *testing.T) {
	mock := &mockSessionService{}
	router := setupCSRFRouter(mock)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-session"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestProtect_UnsafeMethod_InvalidToken_Returns403(t *testing.T) {
	mock := &mockSessionService{validateResult: false}
	router := setupCSRFRouter(mock)

	form := url.Values{"_csrf": {"wrong-token"}}
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-session"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestProtect_UnsafeMethod_ValidFormToken_Proceeds(t *testing.T) {
	mock := &mockSessionService{validateResult: true}
	router := setupCSRFRouter(mock)

	form := url.Values{"_csrf": {"valid-csrf-token"}}
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-session"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestProtect_UnsafeMethod_ValidHeaderToken_Proceeds(t *testing.T) {
	mock := &mockSessionService{validateResult: true}
	router := setupCSRFRouter(mock)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("X-CSRF-Token", "valid-csrf-token")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-session"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestProtect_PUT_MissingToken_Returns403(t *testing.T) {
	mock := &mockSessionService{}
	router := setupCSRFRouter(mock)

	req := httptest.NewRequest(http.MethodPut, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-session"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestProtect_DELETE_MissingToken_Returns403(t *testing.T) {
	mock := &mockSessionService{}
	router := setupCSRFRouter(mock)

	req := httptest.NewRequest(http.MethodDelete, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-session"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestProtect_NoSessionCookie_UnsafeMethod_Proceeds(t *testing.T) {
	mock := &mockSessionService{}
	router := setupCSRFRouter(mock)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 (no session, skip CSRF), got %d", w.Code)
	}
}

func TestProtect_HEAD_EmbedTokenInContext(t *testing.T) {
	mock := &mockSessionService{csrfToken: "head-csrf-token"}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := NewCSRFMiddleware(mock)
	r.Use(mw.Protect())

	var gotToken string
	r.HEAD("/test", func(c *gin.Context) {
		gotToken = GetCSRFToken(c)
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodHead, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-session"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if gotToken != "head-csrf-token" {
		t.Errorf("expected csrf token 'head-csrf-token', got '%s'", gotToken)
	}
}

func TestGetCSRFToken_NoTokenInContext_ReturnsEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	token := GetCSRFToken(c)
	if token != "" {
		t.Errorf("expected empty token, got '%s'", token)
	}
}

func TestIsSafeMethod(t *testing.T) {
	tests := []struct {
		method string
		safe   bool
	}{
		{http.MethodGet, true},
		{http.MethodHead, true},
		{http.MethodOptions, true},
		{http.MethodPost, false},
		{http.MethodPut, false},
		{http.MethodDelete, false},
		{http.MethodPatch, false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := isSafeMethod(tt.method)
			if result != tt.safe {
				t.Errorf("isSafeMethod(%s) = %v, want %v", tt.method, result, tt.safe)
			}
		})
	}
}
