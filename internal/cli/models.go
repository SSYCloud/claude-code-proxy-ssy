package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Model represents a model from the API
type Model struct {
	Company     string `json:"company"`
	Name        string `json:"name"`
	APIName     string `json:"api_name"`
	Description string `json:"description"`
	ID          string `json:"id"`
}

// ModelsResponse represents the API response for models
type ModelsResponse struct {
	Data []Model `json:"data"`
}

// FetchModels fetches the list of available models from the API
func FetchModels(apiKey string) ([]Model, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://router.shengsuanyun.com/api/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// Add API key if provided
	if apiKey != "" {
		req.Header.Add("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var modelsResponse ModelsResponse
	if err := json.Unmarshal(body, &modelsResponse); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	return modelsResponse.Data, nil
}
