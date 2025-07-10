package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"claude-code-provider-proxy/internal/config"
	"claude-code-provider-proxy/internal/models"

	"github.com/sirupsen/logrus"
)

// OpenAIClient handles communication with OpenAI API
type OpenAIClient struct {
	config     *config.Config
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(cfg *config.Config, logger *logrus.Logger) *OpenAIClient {
	return &OpenAIClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 180 * time.Second, // 3 minutes timeout like Python version
		},
		logger: logger,
	}
}

// CreateChatCompletion sends a chat completion request to OpenAI
func (c *OpenAIClient) CreateChatCompletion(ctx context.Context, req *models.OpenAIRequest) (*models.OpenAIResponse, error) {
	// Prepare request body
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", c.config.OpenAIBaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	c.setHeaders(httpReq)

	// Log request
	c.logger.WithFields(logrus.Fields{
		"url":    url,
		"model":  req.Model,
		"stream": req.Stream,
	}).Debug("Making OpenAI API request")

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleAPIError(resp.StatusCode, respBody)
	}

	// Parse response
	var openAIResp models.OpenAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Log response
	c.logger.WithFields(logrus.Fields{
		"response_id":       openAIResp.ID,
		"model":             openAIResp.Model,
		"prompt_tokens":     openAIResp.Usage.PromptTokens,
		"completion_tokens": openAIResp.Usage.CompletionTokens,
		"total_tokens":      openAIResp.Usage.TotalTokens,
	}).Debug("Received OpenAI API response")

	return &openAIResp, nil
}

// CreateStreamingChatCompletion sends a streaming chat completion request to OpenAI
func (c *OpenAIClient) CreateStreamingChatCompletion(ctx context.Context, req *models.OpenAIRequest) (*http.Response, error) {
	// Ensure streaming is enabled
	req.Stream = true

	// Prepare request body
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", c.config.OpenAIBaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")

	// Log request
	c.logger.WithFields(logrus.Fields{
		"url":    url,
		"model":  req.Model,
		"stream": true,
	}).Debug("Making streaming OpenAI API request")

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, c.handleAPIError(resp.StatusCode, respBody)
	}

	return resp, nil
}

// setHeaders sets the required headers for OpenAI API requests
func (c *OpenAIClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.OpenAIAPIKey))
	req.Header.Set("User-Agent", "claude-code-provider-proxy/1.0")

	// Set custom headers as per Python version
	req.Header.Set("HTTP-Referer", c.config.ReferrerURL)
	req.Header.Set("X-Title", c.config.AppName)
}

// handleAPIError handles API errors from OpenAI
func (c *OpenAIClient) handleAPIError(statusCode int, body []byte) error {
	// Try to parse OpenAI error format
	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
		return &models.APIError{
			Type:    models.ErrorTypeAPI,
			Message: errorResp.Error.Message,
			Code:    errorResp.Error.Code,
		}
	}

	// Fallback to generic error
	switch statusCode {
	case http.StatusUnauthorized:
		return models.NewAuthenticationError("Invalid API key")
	case http.StatusForbidden:
		return models.NewPermissionError("Insufficient permissions")
	case http.StatusTooManyRequests:
		return models.NewRateLimitError("Rate limit exceeded")
	case http.StatusBadRequest:
		return models.NewValidationError(fmt.Sprintf("Bad request: %s", string(body)))
	default:
		return models.NewAPIError(fmt.Sprintf("OpenAI API error: %d - %s", statusCode, string(body)))
	}
}

// ValidateAPIKey validates the OpenAI API key
func (c *OpenAIClient) ValidateAPIKey(ctx context.Context) error {
	if c.config.OpenAIAPIKey == "" {
		return models.NewAuthenticationError("OpenAI API key is required")
	}

	// Make a simple request to validate the key
	req := &models.OpenAIRequest{
		Model:     c.config.SmallModelName, // Use a simple model for validation
		Messages:  []models.OpenAIMessage{{Role: "user", Content: "test"}},
		MaxTokens: 1,
	}

	_, err := c.CreateChatCompletion(ctx, req)
	if err != nil {
		if apiErr, ok := err.(*models.APIError); ok && apiErr.Type == models.ErrorTypeAuthentication {
			return models.NewAuthenticationError("Invalid OpenAI API key")
		}
		return err
	}

	return nil
}

// GetModels retrieves available models from OpenAI
func (c *OpenAIClient) GetModels(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/models", c.config.OpenAIBaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, c.handleAPIError(resp.StatusCode, body)
	}

	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	models := make([]string, len(modelsResp.Data))
	for i, model := range modelsResp.Data {
		models[i] = model.ID
	}

	return models, nil
}

// SetTimeout sets the HTTP client timeout
func (c *OpenAIClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// Close closes the HTTP client (cleanup)
func (c *OpenAIClient) Close() error {
	// HTTP client doesn't need explicit closing, but we can implement cleanup here if needed
	return nil
}
