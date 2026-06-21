package controllers

import (
	"errors"
	"net/http"

	"github.com/a-h/templ"
	"github.com/erick/curriculo/internal/middlewares"
	"github.com/erick/curriculo/internal/models"
	"github.com/erick/curriculo/internal/services"
	"github.com/erick/curriculo/internal/views/pages"
	"github.com/gin-gonic/gin"
)

// AuthController handles authentication-related HTTP requests.
type AuthController struct {
	authService    services.AuthService
	sessionService services.SessionService
}

// NewAuthController creates a new AuthController with the given service dependencies.
func NewAuthController(authSvc services.AuthService, sessionSvc services.SessionService) *AuthController {
	return &AuthController{
		authService:    authSvc,
		sessionService: sessionSvc,
	}
}

// LoginPage renders the login form.
func (ctrl *AuthController) LoginPage(c *gin.Context) {
	render(c, http.StatusOK, pages.LoginWithError(""))
}

// Login handles login form submission.
func (ctrl *AuthController) Login(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	if email == "" || password == "" {
		render(c, http.StatusUnprocessableEntity, pages.LoginWithError("Preencha todos os campos."))
		return
	}

	user, err := ctrl.authService.Login(c.Request.Context(), email, password)
	if err != nil {
		render(c, http.StatusUnprocessableEntity, pages.LoginWithError("E-mail ou senha incorretos."))
		return
	}

	token, err := ctrl.sessionService.Create(c.Request.Context(), user.ID)
	if err != nil {
		render(c, http.StatusInternalServerError, pages.LoginWithError("Erro interno. Tente novamente."))
		return
	}

	ctrl.setSessionCookie(c, token, user.Name)

	if isHTMXRequest(c) {
		c.Header("HX-Redirect", "/resumes")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/resumes")
}

// RegisterPage renders the registration form.
func (ctrl *AuthController) RegisterPage(c *gin.Context) {
	render(c, http.StatusOK, pages.RegisterWithError(""))
}

// Register handles registration form submission.
func (ctrl *AuthController) Register(c *gin.Context) {
	name := c.PostForm("name")
	email := c.PostForm("email")
	password := c.PostForm("password")

	input := services.RegisterInput{
		Name:     name,
		Email:    email,
		Password: password,
	}

	user, err := ctrl.authService.Register(c.Request.Context(), input)
	if err != nil {
		errorMsg := "Erro ao criar conta. Tente novamente."

		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			// Get first field error message
			for _, msg := range validationErr.Fields {
				errorMsg = msg
				break
			}
		} else if errors.Is(err, models.ErrDuplicateEmail) {
			errorMsg = "Este e-mail já está cadastrado."
		}

		render(c, http.StatusUnprocessableEntity, pages.RegisterWithError(errorMsg))
		return
	}

	token, err := ctrl.sessionService.Create(c.Request.Context(), user.ID)
	if err != nil {
		render(c, http.StatusInternalServerError, pages.RegisterWithError("Erro interno. Tente novamente."))
		return
	}

	ctrl.setSessionCookie(c, token, user.Name)

	if isHTMXRequest(c) {
		c.Header("HX-Redirect", "/resumes")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/resumes")
}

// Logout destroys the user's session and clears the session cookie.
func (ctrl *AuthController) Logout(c *gin.Context) {
	token, err := c.Cookie("session_token")
	if err == nil && token != "" {
		_ = ctrl.sessionService.Delete(c.Request.Context(), token)
	}

	c.SetCookie("session_token", "", -1, "/", "", false, true)
	c.SetCookie("user_name", "", -1, "/", "", false, false)

	if isHTMXRequest(c) {
		c.Header("HX-Redirect", "/login")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/login")
}

// GetCSRFToken extracts the CSRF token from the Gin context.
func GetCSRFToken(c *gin.Context) string {
	return middlewares.GetCSRFToken(c)
}

func (ctrl *AuthController) setSessionCookie(c *gin.Context, token string, userName string) {
	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session_token", token, 7*24*60*60, "/", "", secure, true)
	// Store user name in a non-httponly cookie for display purposes
	c.SetCookie("user_name", userName, 7*24*60*60, "/", "", secure, false)
}

func isHTMXRequest(c *gin.Context) bool {
	return c.GetHeader("HX-Request") == "true"
}

func render(c *gin.Context, status int, component templ.Component) {
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.String(http.StatusInternalServerError, "render error")
	}
}
