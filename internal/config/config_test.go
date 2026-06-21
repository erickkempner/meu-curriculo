package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// setRequiredEnvVars sets all required environment variables for testing.
func setRequiredEnvVars(t *testing.T) {
	t.Helper()
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("SESSION_SECRET", "supersecret")
}

func TestLoad_AllRequiredVarsSet(t *testing.T) {
	setRequiredEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("expected DB_HOST=localhost, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != "5432" {
		t.Errorf("expected DB_PORT=5432, got %s", cfg.Database.Port)
	}
	if cfg.Database.User != "testuser" {
		t.Errorf("expected DB_USER=testuser, got %s", cfg.Database.User)
	}
	if cfg.Database.Password != "testpass" {
		t.Errorf("expected DB_PASSWORD=testpass, got %s", cfg.Database.Password)
	}
	if cfg.Database.Name != "testdb" {
		t.Errorf("expected DB_NAME=testdb, got %s", cfg.Database.Name)
	}
	if cfg.Session.Secret != "supersecret" {
		t.Errorf("expected SESSION_SECRET=supersecret, got %s", cfg.Session.Secret)
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	setRequiredEnvVars(t)
	// Ensure optional vars are NOT set so defaults kick in
	os.Unsetenv("APP_PORT")
	os.Unsetenv("APP_ENV")
	os.Unsetenv("DB_SSLMODE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.App.Port != "8080" {
		t.Errorf("expected default APP_PORT=8080, got %s", cfg.App.Port)
	}
	if cfg.App.Env != "development" {
		t.Errorf("expected default APP_ENV=development, got %s", cfg.App.Env)
	}
	if cfg.Database.SSLMode != "disable" {
		t.Errorf("expected default DB_SSLMODE=disable, got %s", cfg.Database.SSLMode)
	}
}

func TestLoad_OptionalVarsOverrideDefaults(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("APP_PORT", "9090")
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_SSLMODE", "require")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.App.Port != "9090" {
		t.Errorf("expected APP_PORT=9090, got %s", cfg.App.Port)
	}
	if cfg.App.Env != "production" {
		t.Errorf("expected APP_ENV=production, got %s", cfg.App.Env)
	}
	if cfg.Database.SSLMode != "require" {
		t.Errorf("expected DB_SSLMODE=require, got %s", cfg.Database.SSLMode)
	}
}

func TestLoad_MissingRequiredVar(t *testing.T) {
	requiredVars := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "SESSION_SECRET",
	}

	for _, varName := range requiredVars {
		t.Run("missing_"+varName, func(t *testing.T) {
			setRequiredEnvVars(t)
			os.Unsetenv(varName)

			_, err := Load()
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", varName)
			}
			if !strings.Contains(err.Error(), varName) {
				t.Errorf("expected error to mention %s, got: %v", varName, err)
			}
		})
	}
}

func TestLoad_EmptyRequiredVar(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("DB_HOST", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for empty DB_HOST, got nil")
	}
	if !strings.Contains(err.Error(), "DB_HOST") {
		t.Errorf("expected error to mention DB_HOST, got: %v", err)
	}
}

func TestLoad_SessionDefaults(t *testing.T) {
	setRequiredEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expectedMaxAge := 7 * 24 * time.Hour
	if cfg.Session.MaxAge != expectedMaxAge {
		t.Errorf("expected MaxAge=%v, got %v", expectedMaxAge, cfg.Session.MaxAge)
	}
	if cfg.Session.CookieName != "session_token" {
		t.Errorf("expected CookieName=session_token, got %s", cfg.Session.CookieName)
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	db := DatabaseConfig{
		Host:     "myhost",
		Port:     "5433",
		User:     "admin",
		Password: "secret",
		Name:     "mydb",
		SSLMode:  "require",
	}

	expected := "host=myhost port=5433 user=admin password=secret dbname=mydb sslmode=require"
	got := db.DSN()
	if got != expected {
		t.Errorf("expected DSN=%q, got %q", expected, got)
	}
}
