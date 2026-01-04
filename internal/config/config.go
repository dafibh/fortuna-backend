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

	// S3 Storage
	S3 S3Config
}

// S3Config holds AWS S3 configuration
type S3Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // Optional: for MinIO/LocalStack local dev
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
		S3: S3Config{
			Region:          getEnv("S3_REGION", "us-east-1"),
			Bucket:          getEnv("S3_BUCKET", "fortuna-images"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			Endpoint:        getEnv("S3_ENDPOINT", ""), // Empty = use AWS, set for MinIO/LocalStack
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
