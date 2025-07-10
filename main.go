package main

import (
	"fmt"
	"os"

	"claude-code-provider-proxy/internal/cli"
	"claude-code-provider-proxy/internal/config"
	"claude-code-provider-proxy/internal/server"

	"github.com/spf13/cobra"
)

var (
	configManager  *cli.ConfigManager
	serviceManager *cli.ServiceManager
)

func init() {
	configManager = cli.NewConfigManager()
	serviceManager = cli.NewServiceManager(configManager)
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "claudeproxy",
		Short: "Claude Code Proxy - 将Claude API转换为OpenAI格式的代理服务",
		Long: `Claude Code Proxy 是一个代理服务，可以将Claude API调用转换为OpenAI兼容的格式。
它允许您在支持OpenAI API的应用程序中使用Claude模型。`,
		Run: func(cmd *cobra.Command, args []string) {
			if !configManager.ConfigExists() {
				runInitialSetup()
			} else {
				cmd.Help()
			}
		},
	}

	// Setup command
	var setupCmd = &cobra.Command{
		Use:   "setup",
		Short: "初始化配置",
		Long:  "运行初始化向导来配置API密钥和模型选择",
		Run: func(cmd *cobra.Command, args []string) {
			runInitialSetup()
		},
	}

	// Start command
	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "启动服务",
		Long:  "在后台启动Claude代理服务",
		Run: func(cmd *cobra.Command, args []string) {
			if !configManager.ConfigExists() {
				fmt.Println("❌ 配置文件不存在，请先运行 'claudeproxy setup'")
				os.Exit(1)
			}

			if err := serviceManager.Start(); err != nil {
				cli.ShowError(err)
			}
		},
	}

	// Stop command
	var stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "停止服务",
		Long:  "停止正在运行的Claude代理服务",
		Run: func(cmd *cobra.Command, args []string) {
			if err := serviceManager.Stop(); err != nil {
				cli.ShowError(err)
			}
		},
	}

	// Status command
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "查看服务状态",
		Long:  "显示Claude代理服务的当前状态",
		Run: func(cmd *cobra.Command, args []string) {
			if err := serviceManager.Status(); err != nil {
				cli.ShowError(err)
			}
		},
	}

	// Set command
	var setCmd = &cobra.Command{
		Use:   "set",
		Short: "修改配置",
		Long:  "修改API密钥或模型配置",
		Run: func(cmd *cobra.Command, args []string) {
			runSetConfig()
		},
	}

	// Config command
	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "显示当前配置",
		Long:  "显示当前的配置信息",
		Run: func(cmd *cobra.Command, args []string) {
			if err := configManager.ListConfig(); err != nil {
				cli.ShowError(err)
			}
		},
	}

	// Server command (internal use for background service)
	var serverCmd = &cobra.Command{
		Use:    "server",
		Short:  "运行服务器 (内部使用)",
		Long:   "直接运行服务器，通常由start命令在后台调用",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			if err := configManager.LoadConfig(); err != nil {
				cli.ShowError(fmt.Errorf("加载配置失败: %v", err))
			}

			// Load config and start server
			cfg := config.Load()
			srv := server.New(cfg)

			fmt.Printf("🚀 启动服务器在 http://%s:%s\n", cfg.Host, cfg.Port)
			if err := srv.Start(); err != nil {
				cli.ShowError(fmt.Errorf("启动服务器失败: %v", err))
			}
		},
	}

	// Clean command
	var cleanCmd = &cobra.Command{
		Use:   "clean",
		Short: "清除所有环境变量",
		Long:  "清除所有与Claude Code Proxy相关的环境变量（包括当前终端和全局环境）",
		Run: func(cmd *cobra.Command, args []string) {
			if !cli.ConfirmAction("确认要清除所有项目相关的环境变量吗? 这将清除配置文件和全局环境变量") {
				fmt.Println("操作已取消")
				return
			}

			// Stop service if running
			if serviceManager.IsRunning() {
				fmt.Println("🛑 正在停止服务...")
				if err := serviceManager.Stop(); err != nil {
					fmt.Printf("⚠️  停止服务失败: %v\n", err)
				}
			}

			// Clear environment variables from current session
			projectEnvVars := []string{
				"SSY_API_KEY", "BIG_MODEL_NAME", "SMALL_MODEL_NAME",
				"BASE_URL", "REFERRER_URL", "APP_NAME", "APP_VERSION",
				"HOST", "PORT", "RELOAD", "OPEN_CLAUDE_CACHE", "LOG_LEVEL",
				"ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN", // 添加ANTHROPIC相关环境变量
			}

			fmt.Println("🧹 正在清除当前会话的环境变量...")
			clearedCount := 0
			for _, key := range projectEnvVars {
				if value := os.Getenv(key); value != "" {
					os.Unsetenv(key)
					fmt.Printf("✅ 已清除当前会话变量: %s\n", key)
					clearedCount++
				}
			}

			if clearedCount == 0 {
				fmt.Println("ℹ️  当前会话中没有发现项目相关的环境变量")
			}

			// Clear all environment variables from config files
			if err := configManager.ClearAllEnvVars(); err != nil {
				cli.ShowError(fmt.Errorf("清除环境变量失败: %v", err))
			}

			// Delete config file
			if configManager.ConfigExists() {
				if err := configManager.DeleteConfig(); err != nil {
					fmt.Printf("⚠️  删除配置文件失败: %v\n", err)
				} else {
					fmt.Println("✅ 配置文件已删除")
				}
			}

			fmt.Println("\n✅ 清理完成！")
			fmt.Println("💡 配置文件和shell配置文件中的环境变量已清除")
			fmt.Println("\n⚠️  注意: 当前终端会话的环境变量无法通过程序清除")
			fmt.Println("如需清除当前会话的环境变量，请手动执行以下命令:")
			for _, key := range projectEnvVars {
				fmt.Printf("   unset %s\n", key)
			}
			fmt.Println("\n💡 建议重启终端以确保所有环境变量完全清除")
		},
	}

	// Add commands to root
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(cleanCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		cli.ShowError(err)
	}
}

// runInitialSetup runs the initial setup wizard
func runInitialSetup() {
	cli.ShowWelcome()

	// Check for existing environment variables
	existingVars := configManager.CheckExistingEnvVars()

	// Set default configuration
	if err := configManager.SetDefaults(); err != nil {
		cli.ShowError(fmt.Errorf("设置默认配置失败: %v", err))
	}

	// Handle API key
	var apiKey string
	var isNewAPIKey bool
	if existing, hasExisting := existingVars["SSY_API_KEY"]; hasExisting {
		var err error
		apiKey, isNewAPIKey, err = cli.PromptForAPIKeyWithExisting(existing)
		if err != nil {
			cli.ShowError(err)
		}
	} else {
		var err error
		apiKey, err = cli.PromptForAPIKey()
		if err != nil {
			cli.ShowError(err)
		}
		isNewAPIKey = true
	}

	if err := configManager.SetAPIKey(apiKey); err != nil {
		cli.ShowError(fmt.Errorf("保存API密钥失败: %v", err))
	}

	// Update global environment variable if it's a new API key
	if isNewAPIKey {
		if err := configManager.UpdateGlobalEnvVar("SSY_API_KEY", apiKey); err != nil {
			fmt.Printf("⚠️  更新全局环境变量失败: %v\n", err)
		}
	}

	// Fetch models
	fmt.Println("\n🔄 获取可用模型列表...")
	models, err := cli.FetchModels(apiKey)
	if err != nil {
		cli.ShowError(fmt.Errorf("获取模型列表失败: %v", err))
	}

	if len(models) == 0 {
		cli.ShowError(fmt.Errorf("没有可用的模型"))
	}

	fmt.Printf("✅ 找到 %d 个可用模型\n\n", len(models))

	// Handle big model selection
	var bigModel string
	var isNewBigModel bool
	if existing, hasExisting := existingVars["BIG_MODEL_NAME"]; hasExisting {
		bigModel, isNewBigModel, err = cli.PromptForModelWithExisting(models, "大", existing)
		if err != nil {
			cli.ShowError(err)
		}
	} else {
		bigModel, err = cli.PromptForModel(models, "大")
		if err != nil {
			cli.ShowError(err)
		}
		isNewBigModel = true
	}

	// Handle small model selection
	var smallModel string
	var isNewSmallModel bool
	if existing, hasExisting := existingVars["SMALL_MODEL_NAME"]; hasExisting {
		smallModel, isNewSmallModel, err = cli.PromptForModelWithExisting(models, "小", existing)
		if err != nil {
			cli.ShowError(err)
		}
	} else {
		smallModel, err = cli.PromptForModel(models, "小")
		if err != nil {
			cli.ShowError(err)
		}
		isNewSmallModel = true
	}

	// Save model configuration
	if err := configManager.SetModels(bigModel, smallModel); err != nil {
		cli.ShowError(fmt.Errorf("保存模型配置失败: %v", err))
	}

	// Update global environment variables for models if they are new
	if isNewBigModel {
		if err := configManager.UpdateGlobalEnvVar("BIG_MODEL_NAME", bigModel); err != nil {
			fmt.Printf("⚠️  更新BIG_MODEL_NAME环境变量失败: %v\n", err)
		}
	}

	if isNewSmallModel {
		if err := configManager.UpdateGlobalEnvVar("SMALL_MODEL_NAME", smallModel); err != nil {
			fmt.Printf("⚠️  更新SMALL_MODEL_NAME环境变量失败: %v\n", err)
		}
	}

	// Restart service if running and any configuration changed
	if isNewAPIKey || isNewBigModel || isNewSmallModel {
		if err := serviceManager.RestartIfRunning(); err != nil {
			fmt.Printf("⚠️  重启服务失败: %v\n", err)
			fmt.Println("请手动重启服务: claudeproxy stop && claudeproxy start")
		}
	}

	cli.ShowSetupComplete()
}

// runSetConfig runs the configuration modification wizard
func runSetConfig() {
	if !configManager.ConfigExists() {
		fmt.Println("❌ 配置文件不存在，请先运行 'claudeproxy setup'")
		os.Exit(1)
	}

	choices := []string{
		"修改API密钥",
		"修改模型配置",
		"查看当前配置",
		"重新初始化配置",
	}

	choice, err := cli.PromptForChoice("请选择要修改的配置", choices)
	if err != nil {
		cli.ShowError(err)
	}

	var needRestart bool
	var configChanges map[string]string

	switch choice {
	case "修改API密钥":
		// Load current configuration to check for changes
		if err := configManager.LoadConfig(); err != nil {
			cli.ShowError(fmt.Errorf("加载配置失败: %v", err))
		}

		currentAPIKey := configManager.GetConfig("SSY_API_KEY")

		apiKey, err := cli.PromptForAPIKey()
		if err != nil {
			cli.ShowError(err)
		}

		if err := configManager.SetAPIKey(apiKey); err != nil {
			cli.ShowError(fmt.Errorf("保存API密钥失败: %v", err))
		} // Check if API key changed
		if currentAPIKey != apiKey {
			configChanges = map[string]string{"SSY_API_KEY": apiKey}
			needRestart = true

			// Update global environment variable
			if err := configManager.UpdateGlobalEnvVar("SSY_API_KEY", apiKey); err != nil {
				fmt.Printf("⚠️  更新全局环境变量失败: %v\n", err)
			}
		}

		fmt.Println("✅ API密钥已更新")

	case "修改模型配置":
		// Load current API key and models
		if err := configManager.LoadConfig(); err != nil {
			cli.ShowError(fmt.Errorf("加载配置失败: %v", err))
		}

		apiKey := configManager.GetConfig("SSY_API_KEY")
		if apiKey == "" {
			cli.ShowError(fmt.Errorf("API密钥未配置"))
		}

		currentBigModel := configManager.GetConfig("BIG_MODEL_NAME")
		currentSmallModel := configManager.GetConfig("SMALL_MODEL_NAME")

		// Fetch models
		fmt.Println("\n🔄 获取可用模型列表...")
		models, err := cli.FetchModels(apiKey)
		if err != nil {
			cli.ShowError(fmt.Errorf("获取模型列表失败: %v", err))
		}

		// Select models
		bigModel, err := cli.PromptForModel(models, "大")
		if err != nil {
			cli.ShowError(err)
		}

		smallModel, err := cli.PromptForModel(models, "小")
		if err != nil {
			cli.ShowError(err)
		}

		if err := configManager.SetModels(bigModel, smallModel); err != nil {
			cli.ShowError(fmt.Errorf("保存模型配置失败: %v", err))
		}

		// Check if models changed
		configChanges = make(map[string]string)
		if currentBigModel != bigModel {
			configChanges["BIG_MODEL_NAME"] = bigModel
			needRestart = true

			// Update global environment variable
			if err := configManager.UpdateGlobalEnvVar("BIG_MODEL_NAME", bigModel); err != nil {
				fmt.Printf("⚠️  更新BIG_MODEL_NAME环境变量失败: %v\n", err)
			}
		}

		if currentSmallModel != smallModel {
			configChanges["SMALL_MODEL_NAME"] = smallModel
			needRestart = true

			// Update global environment variable
			if err := configManager.UpdateGlobalEnvVar("SMALL_MODEL_NAME", smallModel); err != nil {
				fmt.Printf("⚠️  更新SMALL_MODEL_NAME环境变量失败: %v\n", err)
			}
		}

		fmt.Println("✅ 模型配置已更新")

	case "查看当前配置":
		if err := configManager.ListConfig(); err != nil {
			cli.ShowError(err)
		}

	case "重新初始化配置":
		if cli.ConfirmAction("确认要重新初始化配置吗? 这将删除现有配置") {
			// Stop service if running
			if serviceManager.IsRunning() {
				fmt.Println("正在停止服务...")
				if err := serviceManager.Stop(); err != nil {
					fmt.Printf("⚠️  停止服务失败: %v\n", err)
				}
			}

			if err := configManager.DeleteConfig(); err != nil {
				cli.ShowError(fmt.Errorf("删除配置失败: %v", err))
			}
			runInitialSetup()
			return
		}
	}

	// Restart service if configuration changed and service is running
	if needRestart && serviceManager.IsRunning() {
		fmt.Printf("\n检测到配置变更，需要重启服务以使配置生效。\n")
		if cli.ConfirmAction("是否现在重启服务?") {
			if err := serviceManager.Restart(); err != nil {
				cli.ShowError(fmt.Errorf("重启服务失败: %v", err))
			} else {
				fmt.Println("✅ 服务已重启，新配置已生效")
			}
		} else {
			fmt.Println("⚠️  配置已保存，但需要手动重启服务以使配置生效")
			fmt.Println("   使用 'claudeproxy stop' 然后 'claudeproxy start' 重启服务")
		}
	}
}
