package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	// Application configuration
	AppName     string
	AppVersion  string
	ReferrerURL string

	// Server configuration
	Port string
	Host string

	// OpenAI API configuration
	OpenAIAPIKey  string
	OpenAIBaseURL string

	// Model configuration
	BigModelName   string
	SmallModelName string

	// Logging configuration
	LogLevel string

	// Cache configuration
	OpenClaudeCache bool

	// CORS configuration
	AllowOrigins []string
	AllowHeaders []string
	AllowMethods []string
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		AppName:         getEnv("APP_NAME", "ClaudeCodeProxy"),
		AppVersion:      getEnv("APP_VERSION", "1.0.0"),
		ReferrerURL:     getEnv("REFERRER_URL", "https://www.shengsuanyun.com"),
		Port:            getEnv("PORT", "8000"),
		Host:            getEnv("HOST", "0.0.0.0"),
		OpenAIAPIKey:    getEnv("SSY_API_KEY", ""),
		OpenAIBaseURL:   getEnv("BASE_URL", "https://api.openai.com/v1"),
		BigModelName:    getEnv("BIG_MODEL_NAME", "anthropic/claude-3.7-sonnet"),
		SmallModelName:  getEnv("SMALL_MODEL_NAME", "deepseek/deepseek-v3"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		OpenClaudeCache: getEnvBool("OPEN_CLAUDE_CACHE", false),
		AllowOrigins:    []string{"*"},
		AllowHeaders:    []string{"Origin", "Content-Length", "Content-Type", "Authorization", "x-api-key", "anthropic-version", "Referer"},
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}

	return cfg
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool gets an environment variable as boolean with a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvInt gets an environment variable as integer with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
