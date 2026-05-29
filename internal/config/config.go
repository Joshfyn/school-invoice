package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port string
	Env  string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// JWT
	JWTSecret      string
	JWTExpiryHours int

	// Paystack
	PaystackSecretKey string
	PaystackPublicKey string

	// Termii
	TermiiAPIKey  string
	TermiiSenderID string

	// App URLs
	AppURL string
	APIURL string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	jwtExpiry, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))

	return &Config{
		Port:              getEnv("PORT", "8080"),
		Env:               getEnv("ENV", "development"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/school_invoice?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:         getEnv("JWT_SECRET", "default-secret-change-me"),
		JWTExpiryHours:    jwtExpiry,
		PaystackSecretKey: getEnv("PAYSTACK_SECRET_KEY", ""),
		PaystackPublicKey: getEnv("PAYSTACK_PUBLIC_KEY", ""),
		TermiiAPIKey:      getEnv("TERMII_API_KEY", ""),
		TermiiSenderID:    getEnv("TERMII_SENDER_ID", "SchoolInv"),
		AppURL:            getEnv("APP_URL", "http://localhost:3000"),
		APIURL:            getEnv("API_URL", "http://localhost:8080"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func (c *Config) IsStaging() bool {
	return c.Env == "staging"
}

// IsDeployed returns true for any non-local environment (staging, production).
func (c *Config) IsDeployed() bool {
	return c.IsProduction() || c.IsStaging()
}
