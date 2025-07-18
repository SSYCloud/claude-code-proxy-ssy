package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"net"
	"net/http"
	"net/url"
)

// ServiceManager handles server lifecycle
type ServiceManager struct {
	configManager *ConfigManager
	pidFile       string
}

// NewServiceManager creates a new service manager
func NewServiceManager(cm *ConfigManager) *ServiceManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir, _ = os.Getwd()
	}

	configDir := filepath.Join(homeDir, ".claudeproxy")
	os.MkdirAll(configDir, 0755)

	return &ServiceManager{
		configManager: cm,
		pidFile:       filepath.Join(configDir, "server.pid"),
	}
}

// Start starts the server in background
func (sm *ServiceManager) Start() error {
	// Check if server is already running
	if sm.IsRunning() {
		return fmt.Errorf("服务已经在运行")
	}

	// Load configuration
	if err := sm.configManager.LoadConfig(); err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// Start server in background
	cmd := exec.Command(execPath, "server")
	cmd.Env = os.Environ() // Inherit environment variables

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}

	// Save PID
	if err := sm.savePID(cmd.Process.Pid); err != nil {
		return fmt.Errorf("保存PID失败: %v", err)
	}

	fmt.Printf("服务已启动，PID: %d\n", cmd.Process.Pid)

	host := sm.configManager.GetConfig("HOST")
	port := sm.configManager.GetConfig("PORT")

	fmt.Printf("服务地址: http://%s:%s\n", host, port)

	// 自动设置 ANTHROPIC 环境变量
	if err := sm.setAnthropicEnvVars(host, port); err != nil {
		fmt.Printf("⚠️  设置ANTHROPIC环境变量失败: %v\n", err)
		fmt.Println("💡 提示: 你可以手动设置以下环境变量:")
		fmt.Printf("   export ANTHROPIC_BASE_URL=http://%s:%s\n", host, port)
		fmt.Printf("   export ANTHROPIC_AUTH_TOKEN=claudeproxy\n")
	}

	return nil
}

// Stop stops the running server
func (sm *ServiceManager) Stop() error {
	pid, err := sm.readPID()
	if err != nil {
		return fmt.Errorf("服务未运行")
	}

	// Find and kill the process
	process, err := os.FindProcess(pid)
	if err != nil {
		sm.cleanupPID()
		return fmt.Errorf("找不到进程 %d", pid)
	}

	// Try graceful shutdown first
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// If graceful shutdown fails, force kill
		if err := process.Signal(syscall.SIGKILL); err != nil {
			sm.cleanupPID()
			return fmt.Errorf("停止服务失败: %v", err)
		}
	}

	// Wait a moment for process to stop
	time.Sleep(2 * time.Second)

	// Clean up PID file
	sm.cleanupPID()

	fmt.Printf("服务已停止 (PID: %d)\n", pid)
	return nil
}

// Restart restarts the running server
func (sm *ServiceManager) Restart() error {
	wasRunning := sm.IsRunning()

	if wasRunning {
		fmt.Println("🔄 正在重启服务...")
		if err := sm.Stop(); err != nil {
			return fmt.Errorf("停止服务失败: %v", err)
		}
		// Wait a moment for the service to fully stop
		time.Sleep(1 * time.Second)
	}

	return sm.Start()
}

// RestartIfRunning restarts the service only if it's currently running
func (sm *ServiceManager) RestartIfRunning() error {
	if sm.IsRunning() {
		fmt.Println("🔄 检测到服务正在运行，正在重启以使用新配置...")
		return sm.Restart()
	} else {
		fmt.Println("ℹ️  服务未运行，新配置将在下次启动时生效")
		return nil
	}
}

// Status shows the current status of the server
func (sm *ServiceManager) Status() error {
	if sm.IsRunning() {
		pid, _ := sm.readPID()
		fmt.Printf("服务正在运行 (PID: %d)\n", pid)
		fmt.Printf("服务地址: http://%s:%s\n",
			sm.configManager.GetConfig("HOST"),
			sm.configManager.GetConfig("PORT"))
	} else {
		fmt.Println("服务未运行")
	}
	return nil
}

// IsRunning checks if the server is currently running
func (sm *ServiceManager) IsRunning() bool {
	pid, err := sm.readPID()
	if err != nil {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		sm.cleanupPID()
		return false
	}

	// Send signal 0 to check if process is alive
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		sm.cleanupPID()
		return false
	}

	return true
}

func (sm *ServiceManager) IsProxyHealth() bool {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	host := sm.configManager.GetConfig("HOST")
	port := sm.configManager.GetConfig("PORT")
	u := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, port),
		Path:   "/health",
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close() 
	return resp.StatusCode == http.StatusOK
}

// savePID saves the process ID to file
func (sm *ServiceManager) savePID(pid int) error {
	return os.WriteFile(sm.pidFile, []byte(strconv.Itoa(pid)), 0644)
}

// readPID reads the process ID from file
func (sm *ServiceManager) readPID() (int, error) {
	data, err := os.ReadFile(sm.pidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, err
	}

	return pid, nil
}

// cleanupPID removes the PID file
func (sm *ServiceManager) cleanupPID() {
	os.Remove(sm.pidFile)
}

// setAnthropicEnvVars sets ANTHROPIC environment variables for Claude integration
func (sm *ServiceManager) setAnthropicEnvVars(host, port string) error {
	baseURL := fmt.Sprintf("http://%s:%s", host, port)

	envVars := map[string]string{
		"ANTHROPIC_BASE_URL":   baseURL,
		"ANTHROPIC_AUTH_TOKEN": "claudeproxy",
	}

	// Set environment variables silently
	for key, value := range envVars {
		if err := sm.configManager.UpdateGlobalEnvVarSilent(key, value); err != nil {
			return fmt.Errorf("设置 %s 失败: %v", key, err)
		}
	}

	// Print unified success message
	fmt.Printf("✅ 已设置环境变量 ANTHROPIC_AUTH_TOKEN、ANTHROPIC_BASE_URL\n")

	// Provide platform-specific export commands
	sm.printExportCommands(baseURL)

	return nil
}

// printExportCommands prints the appropriate export commands for the current platform
func (sm *ServiceManager) printExportCommands(baseURL string) {
	fmt.Printf("💡 提示: 请重启终端，或者在当前终端执行以下命令：\n")

	// Check if it's a Windows environment
	if runtime.GOOS == "windows" {
		// Windows uses different syntax for different shells
		fmt.Printf("set ANTHROPIC_BASE_URL=%s\n", baseURL)
		fmt.Printf("set ANTHROPIC_AUTH_TOKEN=claudeproxy\n")
	} else {
		// Unix-like systems (macOS, Linux)
		// Detect current shell
		shell := os.Getenv("SHELL")

		// Try to get shell info from parent process
		if shell == "" {
			shell = "/bin/bash" // default fallback
		}

		// For Unix systems, export syntax is standard
		fmt.Printf("export ANTHROPIC_BASE_URL=%s\n", baseURL)
		fmt.Printf("export ANTHROPIC_AUTH_TOKEN=claudeproxy\n")

		// Add shell-specific hint
		if strings.Contains(shell, "zsh") {
			fmt.Printf("# 或者添加到 ~/.zshrc 文件中\n")
		} else if strings.Contains(shell, "bash") {
			fmt.Printf("# 或者添加到 ~/.bashrc 或 ~/.bash_profile 文件中\n")
		} else if strings.Contains(shell, "fish") {
			fmt.Printf("# Fish shell 用户请使用: set -gx ANTHROPIC_BASE_URL %s\n", baseURL)
			fmt.Printf("# Fish shell 用户请使用: set -gx ANTHROPIC_AUTH_TOKEN claudeproxy\n")
		}
	}
}

// RunClaudeCode runs Claude Code with proxy environment variables unset
func (sm *ServiceManager) RunClaudeCode(args []string) error {
	// Make sure server is running
	// if !sm.IsRunning() {
	// 	return fmt.Errorf("服务未运行，请先运行 'claudeproxy start'")
	// }
	if !sm.IsProxyHealth() {
		return fmt.Errorf("服务未运行，请先运行 'claudeproxy start'")
	}
	// Ensure Claude Code is installed
	var claudePath string
	var err error

	// Windows may have .cmd or .exe extensions for the claude command
	if runtime.GOOS == "windows" {
		// 在Windows中，可能需要检查多个可能的可执行文件名
		possibleNames := []string{"claude.cmd", "claude.exe", "claude.bat", "claude"}
		for _, name := range possibleNames {
			claudePath, err = exec.LookPath(name)
			if err == nil {
				break // 找到了可执行文件
			}
		}
		if claudePath == "" {
			return fmt.Errorf("找不到 claude 命令，请先安装 Claude Code: npm install -g @anthropic-ai/claude-code")
		}
	} else {
		claudePath, err = exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("找不到 claude 命令，请先安装 Claude Code: npm install -g @anthropic-ai/claude-code")
		}
	}

	// Create a new command to run Claude Code
	cmd := exec.Command(claudePath, args...)

	// Copy the current environment variables
	env := os.Environ()

	// Filter out proxy environment variables
	filteredEnv := []string{}

	// Windows 环境变量名不区分大小写，需要处理各种大小写形式
	proxyVars := []string{"http_proxy", "https_proxy", "all_proxy"}
	if runtime.GOOS == "windows" {
		// Windows中添加更多可能的大小写变体
		proxyVars = append(proxyVars, "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY")
	}

	for _, e := range env {
		shouldKeep := true

		// 检查是否是代理环境变量
		for _, proxyVar := range proxyVars {
			if strings.HasPrefix(strings.ToLower(e), strings.ToLower(proxyVar)+"=") {
				shouldKeep = false
				break
			}
		}

		if shouldKeep {
			filteredEnv = append(filteredEnv, e)
		}
	}

	// Add NO_PROXY environment variables
	if runtime.GOOS == "windows" {
		filteredEnv = append(filteredEnv, "NO_PROXY=localhost,127.0.0.1,0.0.0.0,::1")
		// Windows 通常也需要大写形式
		filteredEnv = append(filteredEnv, "no_proxy=localhost,127.0.0.1,0.0.0.0,::1")
	} else {
		filteredEnv = append(filteredEnv, "NO_PROXY=localhost,127.0.0.1,0.0.0.0,::1")
		filteredEnv = append(filteredEnv, "no_proxy=localhost,127.0.0.1,0.0.0.0,::1")
	}

	// Set the environment variables for the command
	cmd.Env = filteredEnv

	// Connect the command's stdin, stdout, and stderr to the user's terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("🚀 正在启动 Claude Code (已禁用代理设置)...\n")

	// Run the command and wait for it to complete
	return cmd.Run()
}
