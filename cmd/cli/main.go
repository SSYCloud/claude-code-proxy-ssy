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

	// Add commands to root
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(serverCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		cli.ShowError(err)
	}
}

// runInitialSetup runs the initial setup wizard
func runInitialSetup() {
	cli.ShowWelcome()

	// Set default configuration
	if err := configManager.SetDefaults(); err != nil {
		cli.ShowError(fmt.Errorf("设置默认配置失败: %v", err))
	}

	// Get API key
	apiKey, err := cli.PromptForAPIKey()
	if err != nil {
		cli.ShowError(err)
	}

	if err := configManager.SetAPIKey(apiKey); err != nil {
		cli.ShowError(fmt.Errorf("保存API密钥失败: %v", err))
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

	// Select big model
	bigModel, err := cli.PromptForModel(models, "大")
	if err != nil {
		cli.ShowError(err)
	}

	// Select small model
	smallModel, err := cli.PromptForModel(models, "小")
	if err != nil {
		cli.ShowError(err)
	}

	// Save model configuration
	if err := configManager.SetModels(bigModel, smallModel); err != nil {
		cli.ShowError(fmt.Errorf("保存模型配置失败: %v", err))
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

	switch choice {
	case "修改API密钥":
		apiKey, err := cli.PromptForAPIKey()
		if err != nil {
			cli.ShowError(err)
		}

		if err := configManager.SetAPIKey(apiKey); err != nil {
			cli.ShowError(fmt.Errorf("保存API密钥失败: %v", err))
		}

		fmt.Println("✅ API密钥已更新")

	case "修改模型配置":
		// Load current API key
		if err := configManager.LoadConfig(); err != nil {
			cli.ShowError(fmt.Errorf("加载配置失败: %v", err))
		}

		apiKey := configManager.GetConfig("SSY_API_KEY")
		if apiKey == "" {
			cli.ShowError(fmt.Errorf("API密钥未配置"))
		}

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

		fmt.Println("✅ 模型配置已更新")

	case "查看当前配置":
		if err := configManager.ListConfig(); err != nil {
			cli.ShowError(err)
		}

	case "重新初始化配置":
		if cli.ConfirmAction("确认要重新初始化配置吗? 这将删除现有配置") {
			if err := configManager.DeleteConfig(); err != nil {
				cli.ShowError(fmt.Errorf("删除配置失败: %v", err))
			}
			runInitialSetup()
		}
	}
}
