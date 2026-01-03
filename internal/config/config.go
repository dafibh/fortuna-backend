package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Database
	DatabaseURL string

	// Auth0
	Auth0Domain   string
	Auth0Audience string
	Auth0ClientID string

	// Server
	Port        string
	CORSOrigins []string
	Env         string

	// MinIO
	MinIO MinIOConfig
}

// MinIOConfig holds MinIO/S3 configuration
type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	UseSSL          bool
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		Auth0Domain:   getEnv("AUTH0_DOMAIN", ""),
		Auth0Audience: getEnv("AUTH0_AUDIENCE", ""),
		Auth0ClientID: getEnv("AUTH0_CLIENT_ID", ""),
		Port:          getEnv("PORT", "8080"),
		CORSOrigins:   strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000"), ","),
		Env:           getEnv("ENV", "development"),
		MinIO: MinIOConfig{
			Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKeyID:     getEnv("MINIO_ACCESS_KEY", ""),
			SecretAccessKey: getEnv("MINIO_SECRET_KEY", ""),
			BucketName:      getEnv("MINIO_BUCKET", "fortuna-images"),
			UseSSL:          getEnv("MINIO_USE_SSL", "false") == "true",
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.Auth0Domain == "" {
		return fmt.Errorf("AUTH0_DOMAIN is required")
	}
	if c.Auth0Audience == "" {
		return fmt.Errorf("AUTH0_AUDIENCE is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
