package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ConfigManager handles configuration operations
type ConfigManager struct {
	jsonConfigManager *JSONConfigManager
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		jsonConfigManager: NewJSONConfigManager(),
	}
}

// SetDefaults sets the default configuration values
func (cm *ConfigManager) SetDefaults() error {
	return cm.jsonConfigManager.SetDefaults()
}

// SetAPIKey sets the API key
func (cm *ConfigManager) SetAPIKey(apiKey string) error {
	return cm.jsonConfigManager.UpdateConfig(map[string]string{
		"SSY_API_KEY": apiKey,
	})
}

// SetModels sets the big and small model names
func (cm *ConfigManager) SetModels(bigModel, smallModel string) error {
	return cm.jsonConfigManager.UpdateConfig(map[string]string{
		"BIG_MODEL_NAME":   bigModel,
		"SMALL_MODEL_NAME": smallModel,
	})
}

// LoadConfig loads configuration from JSON file
func (cm *ConfigManager) LoadConfig() error {
	// For backward compatibility, we just check if config exists
	if !cm.jsonConfigManager.ConfigExists() {
		return fmt.Errorf("配置文件不存在")
	}
	return nil
}

// GetConfig gets a configuration value
func (cm *ConfigManager) GetConfig(key string) string {
	return cm.jsonConfigManager.GetConfig(key)
}

// updateConfig updates the configuration with new values
func (cm *ConfigManager) updateConfig(updates map[string]string) error {
	return cm.jsonConfigManager.UpdateConfig(updates)
}

// ConfigExists checks if the configuration file exists
func (cm *ConfigManager) ConfigExists() bool {
	return cm.jsonConfigManager.ConfigExists()
}

// GetConfigPath returns the path to the configuration file
func (cm *ConfigManager) GetConfigPath() string {
	return cm.jsonConfigManager.GetConfigPath()
}

// DeleteConfig deletes the configuration file
func (cm *ConfigManager) DeleteConfig() error {
	return cm.jsonConfigManager.DeleteConfig()
}

// ListConfig displays the current configuration
func (cm *ConfigManager) ListConfig() error {
	return cm.jsonConfigManager.ListConfig()
}

// CheckExistingEnvVars checks for existing configuration values
func (cm *ConfigManager) CheckExistingEnvVars() map[string]string {
	return cm.jsonConfigManager.CheckExistingConfig()
}

// UpdateGlobalEnvVarSilent updates a global environment variable without printing messages
func (cm *ConfigManager) UpdateGlobalEnvVarSilent(key, value string) error {
	// Only handle ANTHROPIC environment variables
	if !strings.HasPrefix(key, "ANTHROPIC_") {
		return fmt.Errorf("只支持ANTHROPIC_开头的环境变量")
	}

	// Update global environment variable based on the OS
	switch runtime.GOOS {
	case "darwin", "linux":
		return cm.updateUnixEnvVarSilent(key, value)
	case "windows":
		return cm.updateWindowsEnvVarSilent(key, value)
	default:
		// For other systems, just update local config
		return nil
	}
}

// updateUnixEnvVarSilent updates environment variable on Unix-like systems without printing messages
func (cm *ConfigManager) updateUnixEnvVarSilent(key, value string) error {
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
			if err := cm.updateShellProfileSilent(profileFile, key, value); err != nil {
				continue // Silent failure
			} else {
				updated = true
			}
		}
	}

	// If no existing profile files found, create .profile
	if !updated {
		profileFile := filepath.Join(homeDir, ".profile")
		if err := cm.updateShellProfileSilent(profileFile, key, value); err != nil {
			return fmt.Errorf("创建 .profile 失败: %v", err)
		}
	}

	return nil
}

// updateWindowsEnvVarSilent updates environment variable on Windows without printing messages
func (cm *ConfigManager) updateWindowsEnvVarSilent(key, value string) error {
	// Use setx command to set user environment variable
	cmd := exec.Command("setx", key, value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("设置Windows环境变量失败: %v", err)
	}

	// Also set it for current session
	os.Setenv(key, value)

	return nil
}

// updateShellProfileSilent updates shell profile file without printing messages
func (cm *ConfigManager) updateShellProfileSilent(profileFile, key, value string) error {
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

	return nil
}

// ClearAllEnvVars clears all ANTHROPIC environment variables
func (cm *ConfigManager) ClearAllEnvVars() error {
	// Only clear ANTHROPIC environment variables
	anthropicEnvVars := []string{
		"ANTHROPIC_BASE_URL",
		"ANTHROPIC_AUTH_TOKEN",
	}

	fmt.Println("🧹 正在清除ANTHROPIC相关的环境变量...")

	// Clear from current session
	for _, key := range anthropicEnvVars {
		os.Unsetenv(key)
	}

	// Clear from global environment based on OS
	switch runtime.GOOS {
	case "darwin", "linux":
		return cm.clearUnixEnvVars(anthropicEnvVars)
	case "windows":
		return cm.clearWindowsEnvVars(anthropicEnvVars)
	default:
		fmt.Printf("⚠️  本系统不支持自动清除全局环境变量，请手动删除以下变量:\n")
		for _, key := range anthropicEnvVars {
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
