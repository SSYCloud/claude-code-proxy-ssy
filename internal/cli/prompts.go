package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
)

// PromptForAPIKey prompts user for API key
func PromptForAPIKey() (string, error) {
	prompt := promptui.Prompt{
		Label:    "请输入您的胜算云API密钥",
		Mask:     '*',
		Validate: validateAPIKey,
	}

	result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("输入API密钥失败: %v", err)
	}

	return strings.TrimSpace(result), nil
}

// PromptForModel prompts user to select a model with fuzzy search
func PromptForModel(models []Model, modelType string) (string, error) {
	if len(models) == 0 {
		return "", fmt.Errorf("没有可用的模型")
	}

	fmt.Printf("\n💡 共找到 %d 个模型，您可以输入关键词进行搜索筛选\n", len(models))
	fmt.Println("💡 搜索支持: 模型名称、API名称、公司名称")
	fmt.Println("💡 留空直接回车可查看所有模型")

	// Add specific guidance based on model type
	if modelType == "大" {
		fmt.Println("🎯 大模型用于处理复杂任务，推荐使用高性能模型")
		fmt.Println("💡 建议选择: Claude-Sonnet、Gemini Pro等高性能模型")
	} else if modelType == "小" {
		fmt.Println("🎯 小模型用于处理简单任务，推荐使用高性价比模型")
		fmt.Println("💡 建议选择: DeepSeek、Doubao、Qwen等高性价比模型")
	}

	for {
		// Prompt for search keyword
		searchPrompt := promptui.Prompt{
			Label: fmt.Sprintf("请输入搜索关键词 (为%s模型)", modelType),
			Validate: func(input string) error {
				// Allow empty input
				return nil
			},
		}

		searchKeyword, err := searchPrompt.Run()
		if err != nil {
			return "", fmt.Errorf("输入搜索关键词失败: %v", err)
		}

		// Filter models based on search keyword
		var filteredModels []Model
		var filteredItems []string

		searchKeyword = strings.ToLower(strings.TrimSpace(searchKeyword))

		for _, model := range models {
			if searchKeyword == "" ||
				strings.Contains(strings.ToLower(model.Name), searchKeyword) ||
				strings.Contains(strings.ToLower(model.APIName), searchKeyword) ||
				strings.Contains(strings.ToLower(model.Company), searchKeyword) {
				filteredModels = append(filteredModels, model)
				filteredItems = append(filteredItems, fmt.Sprintf("%s (%s) - %s", model.Name, model.APIName, model.Company))
			}
		}

		if len(filteredModels) == 0 {
			fmt.Printf("❌ 没有找到匹配关键词 '%s' 的模型，请重新搜索\n", searchKeyword)
			continue // Retry in the same loop
		}

		fmt.Printf("\n✅ 找到 %d 个匹配的模型\n", len(filteredModels))

		// Add option to search again
		filteredItems = append([]string{"🔍 重新搜索 (输入新的关键词)"}, filteredItems...)

		// Show selection prompt
		selectPrompt := promptui.Select{
			Label: fmt.Sprintf("请选择%s模型", modelType),
			Items: filteredItems,
			Size:  15,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ . }}:",
				Active:   "▶ {{ . | cyan }}",
				Inactive: "  {{ . }}",
				Selected: "✓ {{ . | green }}",
			},
		}

		index, _, err := selectPrompt.Run()
		if err != nil {
			return "", fmt.Errorf("选择模型失败: %v", err)
		}

		// Check if user wants to search again
		if index == 0 {
			continue // Retry search in the same loop
		}

		// Adjust index for actual model selection (subtract 1 for the search option)
		return filteredModels[index-1].APIName, nil
	}
}

// ConfirmAction prompts user for confirmation
func ConfirmAction(message string) bool {
	prompt := promptui.Prompt{
		Label:     message,
		IsConfirm: true,
	}

	result, err := prompt.Run()
	if err != nil {
		return false
	}

	return strings.ToLower(result) == "y"
}

// PromptForChoice prompts user to select from a list of choices
func PromptForChoice(label string, choices []string) (string, error) {
	prompt := promptui.Select{
		Label: label,
		Items: choices,
	}

	_, result, err := prompt.Run()
	return result, err
}

// validateAPIKey validates the API key format
func validateAPIKey(input string) error {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return fmt.Errorf("API密钥不能为空")
	}
	if len(input) < 10 {
		return fmt.Errorf("API密钥长度太短")
	}
	return nil
}

// ShowWelcome displays welcome message
func ShowWelcome() {
	fmt.Println()
	fmt.Println("🤖 欢迎使用 Claude Code Proxy")
	fmt.Println("========================================")
	fmt.Println("这是一个将Claude API转换为OpenAI格式的代理服务")
	fmt.Println()
}

// ShowSetupComplete displays setup completion message
func ShowSetupComplete() {
	fmt.Println()
	fmt.Println("✅ 配置完成!")
	fmt.Println("========================================")
	fmt.Println("您现在可以使用以下命令:")
	fmt.Println("  claudeproxy start   - 启动服务")
	fmt.Println("  claudeproxy stop    - 停止服务")
	fmt.Println("  claudeproxy status  - 查看状态")
	fmt.Println("  claudeproxy config  - 查看配置")
	fmt.Println("  claudeproxy set     - 修改配置")
	fmt.Println("  claudeproxy clean   - 清除所有配置")
	fmt.Println()
}

// ShowError displays error message
func ShowError(err error) {
	fmt.Printf("❌ 错误: %v\n", err)
	os.Exit(1)
}

// PromptForExistingOrNew prompts user to use existing value or enter new one
func PromptForExistingOrNew(varName, existingValue, description string) (string, bool, error) {
	// Mask sensitive values
	displayValue := existingValue
	if varName == "SSY_API_KEY" && len(existingValue) > 8 {
		displayValue = existingValue[:8] + strings.Repeat("*", len(existingValue)-8)
	}

	choices := []string{
		fmt.Sprintf("使用现有的%s: %s", description, displayValue),
		fmt.Sprintf("输入新的%s", description),
	}

	prompt := promptui.Select{
		Label: fmt.Sprintf("检测到系统中已有%s", description),
		Items: choices,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}:",
			Active:   "▶ {{ . | cyan }}",
			Inactive: "  {{ . }}",
			Selected: "✓ {{ . | green }}",
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		return "", false, fmt.Errorf("选择失败: %v", err)
	}

	if index == 0 {
		// Use existing value
		return existingValue, false, nil
	} else {
		// Prompt for new value
		if varName == "SSY_API_KEY" {
			newValue, err := PromptForAPIKey()
			return newValue, true, err
		}
		// For other values, we'll handle them separately
		return "", true, nil
	}
}

// PromptForAPIKeyWithExisting prompts for API key, considering existing value
func PromptForAPIKeyWithExisting(existingValue string) (string, bool, error) {
	if existingValue != "" {
		return PromptForExistingOrNew("SSY_API_KEY", existingValue, "API密钥")
	}

	apiKey, err := PromptForAPIKey()
	return apiKey, true, err
}

// PromptForModelWithExisting prompts for model selection, considering existing value
func PromptForModelWithExisting(models []Model, modelType, existingValue string) (string, bool, error) {
	if existingValue != "" {
		// Find the model name for display
		var modelName string
		for _, model := range models {
			if model.APIName == existingValue {
				modelName = fmt.Sprintf("%s (%s)", model.Name, model.Company)
				break
			}
		}
		if modelName == "" {
			modelName = existingValue // Fallback to API name
		}

		choices := []string{
			fmt.Sprintf("使用现有的%s模型: %s", modelType, modelName),
			fmt.Sprintf("重新选择%s模型", modelType),
		}

		prompt := promptui.Select{
			Label: fmt.Sprintf("检测到系统中已有%s模型配置", modelType),
			Items: choices,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ . }}:",
				Active:   "▶ {{ . | cyan }}",
				Inactive: "  {{ . }}",
				Selected: "✓ {{ . | green }}",
			},
		}

		index, _, err := prompt.Run()
		if err != nil {
			return "", false, fmt.Errorf("选择失败: %v", err)
		}

		if index == 0 {
			// Use existing value
			return existingValue, false, nil
		}
	}

	// Prompt for new model selection
	newModel, err := PromptForModel(models, modelType)
	return newModel, true, err
}
