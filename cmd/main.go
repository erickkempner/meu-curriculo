package main

import (
	"context"
	"log"
	"os"

	"github.com/erick/curriculo/internal/config"
	"github.com/erick/curriculo/internal/controllers"
	internaldb "github.com/erick/curriculo/internal/db"
	"github.com/erick/curriculo/internal/middlewares"
	"github.com/erick/curriculo/internal/repositories"
	"github.com/erick/curriculo/internal/routes"
	"github.com/erick/curriculo/internal/services"
	"github.com/erick/curriculo/migrations"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	pool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Run migrations
	if err := internaldb.RunMigrations(ctx, pool, migrations.FS); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Initialize layers
	queries := internaldb.New(pool)
	userRepo := repositories.NewUserRepository(queries)
	sessionRepo := repositories.NewSessionRepository(queries)
	resumeRepo := repositories.NewResumeRepository(queries)

	authSvc := services.NewAuthService(userRepo)
	sessionSvc := services.NewSessionService(sessionRepo, cfg.Session)
	resumeSvc := services.NewResumeService(resumeRepo)

	authMW := middlewares.NewAuthMiddleware(sessionSvc)
	csrfMW := middlewares.NewCSRFMiddleware(sessionSvc)

	authCtrl := controllers.NewAuthController(authSvc, sessionSvc)
	resumeCtrl := controllers.NewResumeController(resumeSvc, nil) // PDFService nil for now

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Ensure uploads directory exists
	if err := os.MkdirAll("uploads/photos", 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}

	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.Static("/assets", "./assets")
	r.Static("/uploads", "./uploads")

	routes.Setup(r, &routes.Dependencies{
		AuthCtrl:       authCtrl,
		ResumeCtrl:     resumeCtrl,
		AuthMiddleware: authMW,
		CSRFMiddleware: csrfMW,
	})

	log.Printf("Server starting on :%s", cfg.App.Port)
	if err := r.Run(":" + cfg.App.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
