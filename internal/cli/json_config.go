package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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

// JSONConfigManager handles JSON configuration file operations
type JSONConfigManager struct {
	configPath string
}

// NewJSONConfigManager creates a new JSON configuration manager
func NewJSONConfigManager() *JSONConfigManager {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		homeDir, _ = os.Getwd()
	}

	configDir := filepath.Join(homeDir, ".claudeproxy")
	os.MkdirAll(configDir, 0755)

	return &JSONConfigManager{
		configPath: filepath.Join(configDir, "config.json"),
	}
}

// LoadConfig loads configuration from JSON file
func (jcm *JSONConfigManager) LoadConfig() (*JSONConfig, error) {
	if _, err := os.Stat(jcm.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", jcm.configPath)
	}

	data, err := os.ReadFile(jcm.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config JSONConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to JSON file
func (jcm *JSONConfigManager) SaveConfig(config *JSONConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(jcm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// ConfigExists checks if the configuration file exists
func (jcm *JSONConfigManager) ConfigExists() bool {
	_, err := os.Stat(jcm.configPath)
	return err == nil
}

// DeleteConfig deletes the configuration file
func (jcm *JSONConfigManager) DeleteConfig() error {
	if err := os.Remove(jcm.configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除配置文件失败: %v", err)
	}
	return nil
}

// GetConfigPath returns the path to the configuration file
func (jcm *JSONConfigManager) GetConfigPath() string {
	return jcm.configPath
}

// SetDefaults creates a default configuration
func (jcm *JSONConfigManager) SetDefaults() error {
	config := &JSONConfig{
		BaseURL:         "https://router.shengsuanyun.com/api/v1",
		ReferrerURL:     "https://www.shengsuanyun.com",
		AppName:         "ClaudeCodeProxy",
		AppVersion:      "0.1.3",
		Host:            "0.0.0.0",
		Port:            "3180",
		Reload:          "true",
		OpenClaudeCache: "true",
		LogLevel:        "INFO",
	}

	return jcm.SaveConfig(config)
}

// UpdateConfig updates specific fields in the configuration
func (jcm *JSONConfigManager) UpdateConfig(updates map[string]string) error {
	var config *JSONConfig
	var err error

	// Load existing config or create new one
	if jcm.ConfigExists() {
		config, err = jcm.LoadConfig()
		if err != nil {
			return err
		}
	} else {
		config = &JSONConfig{}
	}

	// Update fields
	for key, value := range updates {
		switch key {
		case "SSY_API_KEY":
			config.SSYAPIKey = value
		case "BIG_MODEL_NAME":
			config.BigModelName = value
		case "SMALL_MODEL_NAME":
			config.SmallModelName = value
		case "BASE_URL":
			config.BaseURL = value
		case "REFERRER_URL":
			config.ReferrerURL = value
		case "APP_NAME":
			config.AppName = value
		case "APP_VERSION":
			config.AppVersion = value
		case "HOST":
			config.Host = value
		case "PORT":
			config.Port = value
		case "RELOAD":
			config.Reload = value
		case "OPEN_CLAUDE_CACHE":
			config.OpenClaudeCache = value
		case "LOG_LEVEL":
			config.LogLevel = value
		}
	}

	return jcm.SaveConfig(config)
}

// GetConfig gets a configuration value
func (jcm *JSONConfigManager) GetConfig(key string) string {
	config, err := jcm.LoadConfig()
	if err != nil {
		return ""
	}

	switch key {
	case "SSY_API_KEY":
		return config.SSYAPIKey
	case "BIG_MODEL_NAME":
		return config.BigModelName
	case "SMALL_MODEL_NAME":
		return config.SmallModelName
	case "BASE_URL":
		return config.BaseURL
	case "REFERRER_URL":
		return config.ReferrerURL
	case "APP_NAME":
		return config.AppName
	case "APP_VERSION":
		return config.AppVersion
	case "HOST":
		return config.Host
	case "PORT":
		return config.Port
	case "RELOAD":
		return config.Reload
	case "OPEN_CLAUDE_CACHE":
		return config.OpenClaudeCache
	case "LOG_LEVEL":
		return config.LogLevel
	default:
		return ""
	}
}

// ListConfig displays the current configuration
func (jcm *JSONConfigManager) ListConfig() error {
	config, err := jcm.LoadConfig()
	if err != nil {
		return err
	}

	fmt.Println("📋 当前配置:")
	fmt.Printf("├── SSY API Key: %s\n", maskAPIKey(config.SSYAPIKey))
	fmt.Printf("├── 大模型: %s\n", config.BigModelName)
	fmt.Printf("├── 小模型: %s\n", config.SmallModelName)
	fmt.Printf("├── 基础URL: %s\n", config.BaseURL)
	fmt.Printf("├── 引用URL: %s\n", config.ReferrerURL)
	fmt.Printf("├── 应用名称: %s\n", config.AppName)
	fmt.Printf("├── 应用版本: %s\n", config.AppVersion)
	fmt.Printf("├── 主机: %s\n", config.Host)
	fmt.Printf("├── 端口: %s\n", config.Port)
	fmt.Printf("├── 重载: %s\n", config.Reload)
	fmt.Printf("├── Claude缓存: %s\n", config.OpenClaudeCache)
	fmt.Printf("└── 日志级别: %s\n", config.LogLevel)
	fmt.Printf("\n配置文件路径: %s\n", jcm.configPath)

	return nil
}

// CheckExistingConfig checks for existing configuration values
func (jcm *JSONConfigManager) CheckExistingConfig() map[string]string {
	existing := make(map[string]string)

	if !jcm.ConfigExists() {
		return existing
	}

	config, err := jcm.LoadConfig()
	if err != nil {
		return existing
	}

	if config.SSYAPIKey != "" {
		existing["SSY_API_KEY"] = config.SSYAPIKey
	}
	if config.BigModelName != "" {
		existing["BIG_MODEL_NAME"] = config.BigModelName
	}
	if config.SmallModelName != "" {
		existing["SMALL_MODEL_NAME"] = config.SmallModelName
	}

	return existing
}

// maskAPIKey masks the API key for display
func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "未设置"
	}
	if len(apiKey) < 8 {
		return "***"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}
