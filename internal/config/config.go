package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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

// JSONConfig represents the configuration stored in JSON format
type JSONConfig struct {
	SSYAPIKey       string `json:"ssy_api_key"`
	BigModelName    string `json:"big_model_name"`
	SmallModelName  string `json:"small_model_name"`
	BaseURL         string `json:"base_url"`
	ReferrerURL     string `json:"referrer_url"`
	AppName         string `json:"app_name"`
	AppVersion      string `json:"app_version"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	Reload          string `json:"reload"`
	OpenClaudeCache string `json:"open_claude_cache"`
	LogLevel        string `json:"log_level"`
}

// Load loads configuration from JSON file with fallback to environment variables
func Load() *Config {
	// Try to load from JSON config first
	if jsonConfig := loadFromJSON(); jsonConfig != nil {
		cfg := &Config{
			AppName:         jsonConfig.AppName,
			AppVersion:      jsonConfig.AppVersion,
			ReferrerURL:     jsonConfig.ReferrerURL,
			Port:            jsonConfig.Port,
			Host:            jsonConfig.Host,
			OpenAIAPIKey:    jsonConfig.SSYAPIKey,
			OpenAIBaseURL:   jsonConfig.BaseURL,
			BigModelName:    jsonConfig.BigModelName,
			SmallModelName:  jsonConfig.SmallModelName,
			LogLevel:        jsonConfig.LogLevel,
			OpenClaudeCache: parseBool(jsonConfig.OpenClaudeCache, false),
			AllowOrigins:    []string{"*"},
			AllowHeaders:    []string{"Origin", "Content-Length", "Content-Type", "Authorization", "x-api-key", "anthropic-version", "Referer"},
			AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		}
		return cfg
	}

	// Fallback to environment variables (for backward compatibility)
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

// loadFromJSON attempts to load configuration from JSON file
func loadFromJSON() *JSONConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	configPath := filepath.Join(homeDir, ".claudeproxy", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var config JSONConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}

	return &config
}

// parseBool parses a string to boolean with default value
func parseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	if boolValue, err := strconv.ParseBool(value); err == nil {
		return boolValue
	}
	return defaultValue
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
