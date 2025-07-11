package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// LogManager handles log viewing operations
type LogManager struct {
	logFile string
}

// NewLogManager creates a new log manager
func NewLogManager() *LogManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir, _ = os.Getwd()
	}

	logFile := filepath.Join(homeDir, ".claudeproxy", "logs", "service.log")
	return &LogManager{
		logFile: logFile,
	}
}

// GetLogFile returns the log file path
func (lm *LogManager) GetLogFile() string {
	return lm.logFile
}

// LogExists checks if the log file exists
func (lm *LogManager) LogExists() bool {
	_, err := os.Stat(lm.logFile)
	return err == nil
}

// ViewLogs shows the last n lines of the log file
func (lm *LogManager) ViewLogs(lines int) error {
	if !lm.LogExists() {
		fmt.Println("📝 日志文件不存在")
		fmt.Printf("日志文件位置: %s\n", lm.logFile)
		fmt.Println("💡 请先启动服务以生成日志")
		return nil
	}

	// Use tail command for better performance on large files
	if runtime.GOOS == "windows" {
		return lm.viewLogsWindows(lines)
	} else {
		return lm.viewLogsUnix(lines)
	}
}

// FollowLogs shows real-time logs (like tail -f)
func (lm *LogManager) FollowLogs() error {
	if !lm.LogExists() {
		fmt.Println("📝 日志文件不存在")
		fmt.Printf("日志文件位置: %s\n", lm.logFile)
		fmt.Println("💡 请先启动服务以生成日志")
		return nil
	}

	fmt.Printf("🔍 正在监控日志文件: %s\n", lm.logFile)
	fmt.Println("💡 按 Ctrl+C 退出监控")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if runtime.GOOS == "windows" {
		return lm.followLogsWindows()
	} else {
		return lm.followLogsUnix()
	}
}

// viewLogsUnix shows logs on Unix-like systems
func (lm *LogManager) viewLogsUnix(lines int) error {
	cmd := exec.Command("tail", "-n", strconv.Itoa(lines), lm.logFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// viewLogsWindows shows logs on Windows
func (lm *LogManager) viewLogsWindows(lines int) error {
	file, err := os.Open(lm.logFile)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}
	defer file.Close()

	// For Windows, we'll read the file and show the last n lines
	var logLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		logLines = append(logLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取日志文件失败: %v", err)
	}

	// Show the last n lines
	start := len(logLines) - lines
	if start < 0 {
		start = 0
	}

	for i := start; i < len(logLines); i++ {
		fmt.Println(logLines[i])
	}

	return nil
}

// followLogsUnix follows logs on Unix-like systems
func (lm *LogManager) followLogsUnix() error {
	cmd := exec.Command("tail", "-f", lm.logFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// followLogsWindows follows logs on Windows
func (lm *LogManager) followLogsWindows() error {
	file, err := os.Open(lm.logFile)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}
	defer file.Close()

	// Seek to the end of the file
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("定位日志文件失败: %v", err)
	}

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Wait a moment and try again
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("读取日志文件失败: %v", err)
		}

		// Remove trailing newline and print
		line = strings.TrimSuffix(line, "\n")
		fmt.Println(line)
	}
}

// ClearLogs clears the log file
func (lm *LogManager) ClearLogs() error {
	if !lm.LogExists() {
		fmt.Println("📝 日志文件不存在，无需清理")
		return nil
	}

	if !ConfirmAction("确认要清除所有日志吗？此操作无法撤销") {
		fmt.Println("操作已取消")
		return nil
	}

	if err := os.Truncate(lm.logFile, 0); err != nil {
		return fmt.Errorf("清除日志文件失败: %v", err)
	}

	fmt.Println("✅ 日志文件已清除")
	return nil
}

// GetLogSize returns the size of the log file
func (lm *LogManager) GetLogSize() (int64, error) {
	if !lm.LogExists() {
		return 0, nil
	}

	info, err := os.Stat(lm.logFile)
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

// formatSize formats file size to human readable format
func (lm *LogManager) formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// ShowLogInfo displays log file information
func (lm *LogManager) ShowLogInfo() error {
	fmt.Printf("📝 日志文件信息\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📂 文件路径: %s\n", lm.logFile)

	if !lm.LogExists() {
		fmt.Printf("📊 文件状态: 不存在\n")
		fmt.Printf("💡 提示: 请先启动服务以生成日志\n")
		return nil
	}

	info, err := os.Stat(lm.logFile)
	if err != nil {
		return fmt.Errorf("获取日志文件信息失败: %v", err)
	}

	fmt.Printf("📊 文件大小: %s\n", lm.formatSize(info.Size()))
	fmt.Printf("📅 修改时间: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	fmt.Printf("📊 文件状态: 存在\n")

	return nil
}
