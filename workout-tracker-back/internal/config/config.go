package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port               int
	GinMode            string
	CORSAllowedOrigins []string
	InternalAPIKey     string // API key for n8n -> Backend communication
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	URL string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{}

	// Server config
	port, err := getEnvAsInt("PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}
	cfg.Server.Port = port
	cfg.Server.GinMode = getEnv("GIN_MODE", "debug")
	cfg.Server.CORSAllowedOrigins = getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"*"})

	// Internal API key (required for production)
	cfg.Server.InternalAPIKey = getEnv("INTERNAL_API_KEY", "")
	if cfg.Server.InternalAPIKey == "" && cfg.Server.GinMode == "release" {
		return nil, fmt.Errorf("INTERNAL_API_KEY is required in production")
	}
	// Set default for development
	if cfg.Server.InternalAPIKey == "" {
		cfg.Server.InternalAPIKey = "dev-api-key-not-for-production"
	}

	// Database config
	cfg.Database.URL = getEnv("SUPABASE_DB_URL", "")
	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("SUPABASE_DB_URL is required")
	}

	return cfg, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as an integer with a default value
func getEnvAsInt(key string, defaultValue int) (int, error) {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, err
	}
	return value, nil
}

// getEnvAsBool gets an environment variable as a boolean with a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsSlice gets an environment variable as a comma-separated slice
func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	values := strings.Split(valueStr, ",")
	for i, v := range values {
		values[i] = strings.TrimSpace(v)
	}
	return values
}

// ServerAddr returns the server address string
func (c *Config) ServerAddr() string {
	return fmt.Sprintf(":%d", c.Server.Port)
}
