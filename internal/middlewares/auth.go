package middlewares

import (
	"net/http"
	"net/url"

	"github.com/erick/curriculo/internal/models"
	"github.com/erick/curriculo/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// ContextKeySession is the Gin context key for the authenticated session.
	ContextKeySession = "session"
	// ContextKeyUserID is the Gin context key for the authenticated user's UUID.
	ContextKeyUserID = "user_id"
	// ContextKeyUserName is the Gin context key for the authenticated user's name.
	ContextKeyUserName = "user_name"
)

// AuthMiddleware provides authentication middleware handlers for Gin routes.
type AuthMiddleware struct {
	sessionSvc services.SessionService
}

// NewAuthMiddleware creates a new AuthMiddleware with the given SessionService dependency.
func NewAuthMiddleware(sessionSvc services.SessionService) *AuthMiddleware {
	return &AuthMiddleware{sessionSvc: sessionSvc}
}

// RequireAuth returns a Gin middleware that validates the session cookie.
// On failure, it redirects to /login with the original path preserved for post-login redirect.
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session_token")
		if err != nil || token == "" {
			redirectToLogin(c)
			return
		}

		session, err := m.sessionSvc.Validate(c.Request.Context(), token)
		if err != nil {
			redirectToLogin(c)
			return
		}

		c.Set(ContextKeySession, session)
		c.Set(ContextKeyUserID, session.UserID)
		c.Next()
	}
}

// OptionalAuth returns a Gin middleware that attempts to validate the session cookie
// but does not block the request on failure. If a valid session exists, the user
// context is attached; otherwise the request proceeds without authentication.
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session_token")
		if err != nil || token == "" {
			c.Next()
			return
		}

		session, err := m.sessionSvc.Validate(c.Request.Context(), token)
		if err != nil {
			c.Next()
			return
		}

		c.Set(ContextKeySession, session)
		c.Set(ContextKeyUserID, session.UserID)
		c.Next()
	}
}

// GetUserIDFromContext extracts the authenticated user's UUID from the Gin context.
// Returns ErrUnauthorized if the user ID is not present or has an invalid type.
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get(ContextKeyUserID)
	if !exists {
		return uuid.Nil, models.ErrUnauthorized
	}
	userID, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, models.ErrUnauthorized
	}
	return userID, nil
}

// redirectToLogin redirects the user to the login page, preserving the original
// request path and query string in a "redirect" query parameter.
func redirectToLogin(c *gin.Context) {
	redirect := c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		redirect += "?" + c.Request.URL.RawQuery
	}
	c.Redirect(http.StatusFound, "/login?redirect="+url.QueryEscape(redirect))
	c.Abort()
}
