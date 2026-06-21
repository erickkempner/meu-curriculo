package middlewares

import (
	"net/http"

	"github.com/erick/curriculo/internal/services"
	"github.com/gin-gonic/gin"
)

// ContextKeyCSRFToken is the key used to store the CSRF token in the Gin context,
// making it available for templates to embed in forms.
const ContextKeyCSRFToken = "csrf_token"

// CSRFMiddleware validates CSRF tokens on state-changing requests (POST, PUT, DELETE)
// and embeds the CSRF token in the Gin context for safe methods (GET, HEAD, OPTIONS).
type CSRFMiddleware struct {
	sessionSvc services.SessionService
}

// NewCSRFMiddleware creates a new CSRFMiddleware with the given SessionService dependency.
func NewCSRFMiddleware(sessionSvc services.SessionService) *CSRFMiddleware {
	return &CSRFMiddleware{sessionSvc: sessionSvc}
}

// Protect returns a Gin handler that enforces CSRF protection.
//
// For safe methods (GET, HEAD, OPTIONS): retrieves the CSRF token for the session
// and stores it in the Gin context so templates can embed it in forms.
//
// For unsafe methods (POST, PUT, DELETE): validates the CSRF token from either
// the "_csrf" form field or the "X-CSRF-Token" header. If the token is missing
// or invalid, responds with 403 Forbidden.
func (m *CSRFMiddleware) Protect() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session token from cookie
		sessionToken, err := c.Cookie("session_token")
		if err != nil || sessionToken == "" {
			c.Next()
			return
		}

		// For safe methods, embed CSRF token in context
		if isSafeMethod(c.Request.Method) {
			csrfToken, err := m.sessionSvc.GetCSRFToken(c.Request.Context(), sessionToken)
			if err == nil {
				c.Set(ContextKeyCSRFToken, csrfToken)
			}
			c.Next()
			return
		}

		// For unsafe methods, validate CSRF token
		token := c.PostForm("_csrf")
		if token == "" {
			token = c.GetHeader("X-CSRF-Token")
		}

		if token == "" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if !m.sessionSvc.ValidateCSRFToken(c.Request.Context(), sessionToken, token) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}

// GetCSRFToken retrieves the CSRF token from the Gin context. Returns an empty
// string if no token is stored. This helper is intended for use in templates.
func GetCSRFToken(c *gin.Context) string {
	val, exists := c.Get(ContextKeyCSRFToken)
	if !exists {
		return ""
	}
	token, _ := val.(string)
	return token
}

// isSafeMethod returns true for HTTP methods that do not modify state.
func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}
