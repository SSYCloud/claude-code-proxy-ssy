package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
)

// ConfigManager handles configuration file operations
type ConfigManager struct {
	configPath string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		homeDir, _ = os.Getwd()
	}

	configDir := filepath.Join(homeDir, ".claudeproxy")
	os.MkdirAll(configDir, 0755)

	return &ConfigManager{
		configPath: filepath.Join(configDir, ".env"),
	}
}

// SetDefaults sets the default environment variables
func (cm *ConfigManager) SetDefaults() error {
	defaults := map[string]string{
		"BASE_URL":          "https://router.shengsuanyun.com/api/v1",
		"REFERRER_URL":      "https://www.shengsuanyun.com",
		"APP_NAME":          "ClaudeCodeProxy",
		"APP_VERSION":       "1.0.0",
		"HOST":              "127.0.0.1",
		"PORT":              "3180",
		"RELOAD":            "true",
		"OPEN_CLAUDE_CACHE": "true",
		"LOG_LEVEL":         "INFO",
	}

	return cm.updateConfig(defaults)
}

// SetAPIKey sets the OpenAI API key
func (cm *ConfigManager) SetAPIKey(apiKey string) error {
	// Only update local config, global env vars will be updated by caller if needed
	return cm.updateConfig(map[string]string{
		"SSY_API_KEY": apiKey,
	})
}

// SetModels sets the big and small model names
func (cm *ConfigManager) SetModels(bigModel, smallModel string) error {
	// Only update local config, global env vars will be updated by caller if needed
	return cm.updateConfig(map[string]string{
		"BIG_MODEL_NAME":   bigModel,
		"SMALL_MODEL_NAME": smallModel,
	})
}

// LoadConfig loads configuration from file and sets environment variables
func (cm *ConfigManager) LoadConfig() error {
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在: %s", cm.configPath)
	}

	return godotenv.Load(cm.configPath)
}

// GetConfig gets a configuration value
func (cm *ConfigManager) GetConfig(key string) string {
	return os.Getenv(key)
}

// updateConfig updates the configuration file with new values
func (cm *ConfigManager) updateConfig(updates map[string]string) error {
	// Read existing config
	existing := make(map[string]string)
	if _, err := os.Stat(cm.configPath); err == nil {
		envMap, err := godotenv.Read(cm.configPath)
		if err != nil {
			return fmt.Errorf("读取配置文件失败: %v", err)
		}
		existing = envMap
	}

	// Update with new values
	for key, value := range updates {
		existing[key] = value
	}

	// Write back to file
	file, err := os.Create(cm.configPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for key, value := range existing {
		fmt.Fprintf(writer, "%s=%s\n", key, value)
	}

	return writer.Flush()
}

// ConfigExists checks if config file exists
func (cm *ConfigManager) ConfigExists() bool {
	_, err := os.Stat(cm.configPath)
	return err == nil
}

// GetConfigPath returns the path to the config file
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}

// DeleteConfig removes the configuration file
func (cm *ConfigManager) DeleteConfig() error {
	return os.Remove(cm.configPath)
}

// ListConfig displays all current configuration
func (cm *ConfigManager) ListConfig() error {
	if !cm.ConfigExists() {
		fmt.Println("配置文件不存在")
		return nil
	}

	envMap, err := godotenv.Read(cm.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	fmt.Printf("\n当前配置 (%s):\n", cm.configPath)
	fmt.Println(strings.Repeat("-", 50))

	for key, value := range envMap {
		// Hide sensitive information
		if key == "SSY_API_KEY" && value != "" {
			maskedValue := value[:min(8, len(value))] + strings.Repeat("*", max(0, len(value)-8))
			fmt.Printf("%-20s: %s\n", key, maskedValue)
		} else {
			fmt.Printf("%-20s: %s\n", key, value)
		}
	}
	fmt.Println()

	return nil
}

// CheckExistingEnvVars checks if important environment variables already exist
func (cm *ConfigManager) CheckExistingEnvVars() map[string]string {
	existing := make(map[string]string)

	// Check for existing environment variables
	if apiKey := os.Getenv("SSY_API_KEY"); apiKey != "" {
		existing["SSY_API_KEY"] = apiKey
	}

	if bigModel := os.Getenv("BIG_MODEL_NAME"); bigModel != "" {
		existing["BIG_MODEL_NAME"] = bigModel
	}

	if smallModel := os.Getenv("SMALL_MODEL_NAME"); smallModel != "" {
		existing["SMALL_MODEL_NAME"] = smallModel
	}

	return existing
}

// UpdateGlobalEnvVar updates a global environment variable
func (cm *ConfigManager) UpdateGlobalEnvVar(key, value string) error {
	// Update local config first
	if err := cm.updateConfig(map[string]string{key: value}); err != nil {
		return err
	}

	// Update global environment variable based on the OS
	switch runtime.GOOS {
	case "darwin", "linux":
		if err := cm.updateUnixEnvVar(key, value); err != nil {
			return err
		}
		fmt.Printf("💡 提示: 请重启终端或执行 'source ~/.zshrc' 和 'source ~/.bash_profile' 来使环境变量在所有shell中生效\n")
		return nil
	case "windows":
		return cm.updateWindowsEnvVar(key, value)
	default:
		// For other systems, just update local config
		fmt.Printf("⚠️  本系统不支持自动更新全局环境变量，请手动设置 %s=%s\n", key, value)
		return nil
	}
}

// updateUnixEnvVar updates environment variable on Unix-like systems
func (cm *ConfigManager) updateUnixEnvVar(key, value string) error {
	homeDir, _ := os.UserHomeDir()

	// List of possible shell configuration files
	var profileFiles []string

	if runtime.GOOS == "darwin" {
		// macOS: Update multiple shell config files
		profileFiles = []string{
			filepath.Join(homeDir, ".zshrc"),
			filepath.Join(homeDir, ".bash_profile"),
			filepath.Join(homeDir, ".bashrc"),
			filepath.Join(homeDir, ".profile"),
		}
	} else {
		// Linux: Update common shell config files
		profileFiles = []string{
			filepath.Join(homeDir, ".bashrc"),
			filepath.Join(homeDir, ".zshrc"),
			filepath.Join(homeDir, ".profile"),
		}
	}

	// Update all existing shell configuration files
	updated := false
	for _, profileFile := range profileFiles {
		if _, err := os.Stat(profileFile); err == nil {
			if err := cm.updateShellProfile(profileFile, key, value); err != nil {
				fmt.Printf("⚠️  更新 %s 失败: %v\n", profileFile, err)
			} else {
				updated = true
			}
		}
	}

	// If no existing profile files found, create .profile
	if !updated {
		profileFile := filepath.Join(homeDir, ".profile")
		if err := cm.updateShellProfile(profileFile, key, value); err != nil {
			return fmt.Errorf("创建 .profile 失败: %v", err)
		}
	}

	return nil
}

// updateWindowsEnvVar updates environment variable on Windows
func (cm *ConfigManager) updateWindowsEnvVar(key, value string) error {
	// Use setx command to set user environment variable
	cmd := exec.Command("setx", key, value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("设置Windows环境变量失败: %v", err)
	}

	// Also set it for current session
	os.Setenv(key, value)

	fmt.Printf("✅ 已更新Windows环境变量 %s\n", key)
	return nil
}

// updateShellProfile updates shell profile file
func (cm *ConfigManager) updateShellProfile(profileFile, key, value string) error {
	exportLine := fmt.Sprintf("export %s=\"%s\"", key, value)

	// Read existing content
	var lines []string
	if content, err := os.ReadFile(profileFile); err == nil {
		lines = strings.Split(string(content), "\n")
	}

	// Check if variable already exists and update it
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), fmt.Sprintf("export %s=", key)) {
			lines[i] = exportLine
			found = true
			break
		}
	}

	// If not found, append it
	if !found {
		lines = append(lines, exportLine)
	}

	// Write back to file
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(profileFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("更新profile文件失败: %v", err)
	}

	// Set for current session
	os.Setenv(key, value)

	fmt.Printf("✅ 已更新环境变量 %s 到 %s\n", key, filepath.Base(profileFile))
	return nil
}

// HasConfigChanged checks if the new config values differ from current ones
func (cm *ConfigManager) HasConfigChanged(newValues map[string]string) bool {
	for key, newValue := range newValues {
		if currentValue := cm.GetConfig(key); currentValue != newValue {
			return true
		}
	}
	return false
}

// ClearAllEnvVars clears all project-related environment variables
func (cm *ConfigManager) ClearAllEnvVars() error {
	projectEnvVars := []string{
		"SSY_API_KEY",
		"BIG_MODEL_NAME",
		"SMALL_MODEL_NAME",
		"BASE_URL",
		"REFERRER_URL",
		"APP_NAME",
		"APP_VERSION",
		"HOST",
		"PORT",
		"RELOAD",
		"OPEN_CLAUDE_CACHE",
		"LOG_LEVEL",
		"ANTHROPIC_BASE_URL",   // 添加ANTHROPIC相关环境变量
		"ANTHROPIC_AUTH_TOKEN", // 添加ANTHROPIC相关环境变量
	}

	fmt.Println("🧹 正在清除项目相关的环境变量...")

	// Clear from current session
	for _, key := range projectEnvVars {
		os.Unsetenv(key)
	}

	// Clear from global environment based on OS
	switch runtime.GOOS {
	case "darwin", "linux":
		return cm.clearUnixEnvVars(projectEnvVars)
	case "windows":
		return cm.clearWindowsEnvVars(projectEnvVars)
	default:
		fmt.Printf("⚠️  本系统不支持自动清除全局环境变量，请手动删除以下变量:\n")
		for _, key := range projectEnvVars {
			fmt.Printf("   %s\n", key)
		}
		return nil
	}
}

// clearUnixEnvVars clears environment variables from Unix shell profiles
func (cm *ConfigManager) clearUnixEnvVars(keys []string) error {
	homeDir, _ := os.UserHomeDir()

	var profileFiles []string
	if runtime.GOOS == "darwin" {
		profileFiles = []string{
			filepath.Join(homeDir, ".zshrc"),
			filepath.Join(homeDir, ".bash_profile"),
			filepath.Join(homeDir, ".bashrc"),
			filepath.Join(homeDir, ".profile"),
		}
	} else {
		profileFiles = []string{
			filepath.Join(homeDir, ".bashrc"),
			filepath.Join(homeDir, ".zshrc"),
			filepath.Join(homeDir, ".profile"),
		}
	}

	for _, profileFile := range profileFiles {
		if _, err := os.Stat(profileFile); err == nil {
			if err := cm.removeEnvVarsFromProfile(profileFile, keys); err != nil {
				fmt.Printf("⚠️  清理 %s 失败: %v\n", filepath.Base(profileFile), err)
			}
		}
	}

	fmt.Printf("💡 提示: 请重启终端或执行以下命令来使环境变量清除在所有shell中生效:\n")
	fmt.Printf("   source ~/.zshrc && source ~/.bash_profile\n")
	return nil
}

// clearWindowsEnvVars clears environment variables on Windows
func (cm *ConfigManager) clearWindowsEnvVars(keys []string) error {
	for _, key := range keys {
		// Use reg command to delete user environment variable
		cmd := exec.Command("reg", "delete", "HKEY_CURRENT_USER\\Environment", "/v", key, "/f")
		if err := cmd.Run(); err != nil {
			// Ignore errors for non-existent keys
			continue
		}
		fmt.Printf("✅ 已清除Windows环境变量 %s\n", key)
	}

	fmt.Printf("💡 提示: 请重启命令行或注销重新登录以使环境变量清除生效\n")
	return nil
}

// removeEnvVarsFromProfile removes environment variables from shell profile
func (cm *ConfigManager) removeEnvVarsFromProfile(profileFile string, keys []string) error {
	// Read existing content
	content, err := os.ReadFile(profileFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var filteredLines []string
	removedCount := 0

	for _, line := range lines {
		shouldRemove := false
		trimmedLine := strings.TrimSpace(line)

		// Check if this line exports any of our project variables
		for _, key := range keys {
			if strings.HasPrefix(trimmedLine, fmt.Sprintf("export %s=", key)) {
				shouldRemove = true
				removedCount++
				break
			}
		}

		if !shouldRemove {
			filteredLines = append(filteredLines, line)
		}
	}

	// Only write if we removed something
	if removedCount > 0 {
		newContent := strings.Join(filteredLines, "\n")
		if err := os.WriteFile(profileFile, []byte(newContent), 0644); err != nil {
			return err
		}
		fmt.Printf("✅ 从 %s 中清除了 %d 个环境变量\n", filepath.Base(profileFile), removedCount)
	}

	return nil
}
