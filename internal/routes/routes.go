package routes

import (
	"github.com/erick/curriculo/internal/controllers"
	"github.com/erick/curriculo/internal/middlewares"
	"github.com/gin-gonic/gin"
)

// Dependencies holds all injected controllers and middleware for route setup.
type Dependencies struct {
	AuthCtrl       *controllers.AuthController
	ResumeCtrl     *controllers.ResumeController
	AuthMiddleware *middlewares.AuthMiddleware
	CSRFMiddleware *middlewares.CSRFMiddleware
}

// Setup registers all application routes on the Gin engine.
func Setup(r *gin.Engine, deps *Dependencies) {
	// Public routes
	public := r.Group("/")
	{
		public.GET("/login", deps.AuthCtrl.LoginPage)
		public.POST("/login", deps.AuthCtrl.Login)
		public.GET("/register", deps.AuthCtrl.RegisterPage)
		public.POST("/register", deps.AuthCtrl.Register)
		public.GET("/r/:token", deps.ResumeCtrl.PublicView)
	}

	// Protected routes (auth only, no CSRF - for logout, duplicate, share actions)
	authOnly := r.Group("/")
	authOnly.Use(deps.AuthMiddleware.RequireAuth())
	{
		authOnly.POST("/logout", deps.AuthCtrl.Logout)
		authOnly.POST("/resumes/:id/duplicate", deps.ResumeCtrl.Duplicate)
		authOnly.POST("/resumes/:id/share", deps.ResumeCtrl.Share)
		authOnly.POST("/resumes/:id/photo", deps.ResumeCtrl.UploadPhoto)
		authOnly.POST("/resumes/:id/thumbnail", deps.ResumeCtrl.UploadThumbnail)
		authOnly.GET("/resumes/:id/thumbnail.jpg", deps.ResumeCtrl.ServeThumbnail)
		authOnly.DELETE("/resumes/:id/share", deps.ResumeCtrl.RevokeShare)
		authOnly.POST("/resumes/:id/share/regenerate", deps.ResumeCtrl.RegenerateShare)
		authOnly.DELETE("/resumes/:id", deps.ResumeCtrl.Delete)
	}

	// Protected routes (auth + CSRF)
	protected := r.Group("/")
	protected.Use(deps.AuthMiddleware.RequireAuth())
	protected.Use(deps.CSRFMiddleware.Protect())
	{
		protected.GET("/", deps.ResumeCtrl.List)
		protected.GET("/resumes", deps.ResumeCtrl.List)
		protected.GET("/resumes/new", deps.ResumeCtrl.CreatePage)
		protected.POST("/resumes", deps.ResumeCtrl.Create)
		protected.GET("/resumes/:id/edit", deps.ResumeCtrl.EditPage)
		protected.GET("/resumes/:id/success", deps.ResumeCtrl.Success)
		protected.PUT("/resumes/:id", deps.ResumeCtrl.Update)
		protected.GET("/resumes/:id/pdf", deps.ResumeCtrl.ExportPDF)
	}
}
