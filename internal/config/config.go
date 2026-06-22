package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all application configuration.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Session  SessionConfig
	Uploads  UploadsConfig
}

// UploadsConfig holds file upload storage configuration.
type UploadsConfig struct {
	Dir string // Base directory for uploads (default: "./uploads")
}

// AppConfig holds HTTP server configuration.
type AppConfig struct {
	Port string
	Env  string // "development" | "production"
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	Extra    string // Additional connection params (e.g. channel_binding=require)
}

// SessionConfig holds session management parameters.
type SessionConfig struct {
	Secret     string
	MaxAge     time.Duration // Default: 7 days
	CookieName string        // Default: "session_token"
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
	if d.Extra != "" {
		dsn += " " + d.Extra
	}
	return dsn
}

// Load reads configuration from environment variables.
// It returns a descriptive error mentioning the missing variable name
// if any required variable is unset.
func Load() (*Config, error) {
	required := []struct {
		name  string
		value *string
	}{
		{"DB_HOST", nil},
		{"DB_PORT", nil},
		{"DB_USER", nil},
		{"DB_PASSWORD", nil},
		{"DB_NAME", nil},
		{"SESSION_SECRET", nil},
	}

	values := make(map[string]string)
	for _, r := range required {
		val, ok := os.LookupEnv(r.name)
		if !ok || val == "" {
			return nil, fmt.Errorf("required environment variable %s is not set", r.name)
		}
		values[r.name] = val
	}

	appPort := getEnvOrDefault("APP_PORT", getEnvOrDefault("PORT", "8080"))
	appEnv := getEnvOrDefault("APP_ENV", "development")
	dbSSLMode := getEnvOrDefault("DB_SSLMODE", "disable")
	dbExtra := getEnvOrDefault("DB_EXTRA", "")

	uploadsDir := getEnvOrDefault("UPLOADS_DIR", "./uploads")

	cfg := &Config{
		App: AppConfig{
			Port: appPort,
			Env:  appEnv,
		},
		Database: DatabaseConfig{
			Host:     values["DB_HOST"],
			Port:     values["DB_PORT"],
			User:     values["DB_USER"],
			Password: values["DB_PASSWORD"],
			Name:     values["DB_NAME"],
			SSLMode:  dbSSLMode,
			Extra:    dbExtra,
		},
		Session: SessionConfig{
			Secret:     values["SESSION_SECRET"],
			MaxAge:     7 * 24 * time.Hour,
			CookieName: "session_token",
		},
		Uploads: UploadsConfig{
			Dir: uploadsDir,
		},
	}

	return cfg, nil
}

// getEnvOrDefault returns the value of the environment variable named by key,
// or defaultValue if the variable is not set or is empty.
func getEnvOrDefault(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultValue
}
