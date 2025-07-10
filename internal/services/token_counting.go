package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"claude-code-provider-proxy/internal/models"
)

// TokenCountingService handles token counting for different models
type TokenCountingService struct{}

// NewTokenCountingService creates a new token counting service
func NewTokenCountingService() *TokenCountingService {
	return &TokenCountingService{}
}

// CountTokens estimates the number of tokens in a request
func (s *TokenCountingService) CountTokens(req *models.TokenCountRequest) (*models.TokenCountResponse, error) {
	totalTokens := 0

	// Count system message tokens
	if req.System != nil {
		systemTokens, err := s.countSystemTokens(req.System)
		if err != nil {
			return nil, fmt.Errorf("failed to count system tokens: %w", err)
		}
		totalTokens += systemTokens
	}

	// Count message tokens
	for _, msg := range req.Messages {
		tokens, err := s.countMessageTokens(msg)
		if err != nil {
			return nil, err
		}
		totalTokens += tokens
	}

	// Count tool tokens
	for _, tool := range req.Tools {
		tokens, err := s.countToolTokens(tool)
		if err != nil {
			return nil, err
		}
		totalTokens += tokens
	}

	// Add overhead for message formatting (approximate)
	totalTokens += len(req.Messages) * 4 // Rough estimate for message formatting overhead

	return &models.TokenCountResponse{
		InputTokens: totalTokens,
	}, nil
}

// countSystemTokens counts tokens in system content
func (s *TokenCountingService) countSystemTokens(system interface{}) (int, error) {
	switch sys := system.(type) {
	case string:
		return s.estimateTokens(sys), nil
	case []interface{}:
		totalTokens := 0
		for _, item := range sys {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if contentType, ok := itemMap["type"].(string); ok && contentType == "text" {
					if text, ok := itemMap["text"].(string); ok {
						totalTokens += s.estimateTokens(text)
					}
				}
			} else if textStr, ok := item.(string); ok {
				totalTokens += s.estimateTokens(textStr)
			}
		}
		return totalTokens, nil
	default:
		// Convert to JSON and count
		systemBytes, err := json.Marshal(system)
		if err != nil {
			return 0, err
		}
		return s.estimateTokens(string(systemBytes)), nil
	}
}

// countMessageTokens counts tokens in a single message
func (s *TokenCountingService) countMessageTokens(msg models.AnthropicMessage) (int, error) {
	tokens := 0

	// Count role tokens (approximate)
	tokens += s.estimateTokens(msg.Role)

	// Count content tokens
	switch content := msg.Content.(type) {
	case string:
		tokens += s.estimateTokens(content)
	case []interface{}:
		for _, item := range content {
			itemTokens, err := s.countContentItem(item)
			if err != nil {
				return 0, err
			}
			tokens += itemTokens
		}
	default:
		// Convert to JSON and count
		contentBytes, err := json.Marshal(content)
		if err != nil {
			return 0, err
		}
		tokens += s.estimateTokens(string(contentBytes))
	}

	return tokens, nil
}

// countContentItem counts tokens in a content item
func (s *TokenCountingService) countContentItem(item interface{}) (int, error) {
	itemMap, ok := item.(map[string]interface{})
	if !ok {
		// Convert to string and count
		itemBytes, err := json.Marshal(item)
		if err != nil {
			return 0, err
		}
		return s.estimateTokens(string(itemBytes)), nil
	}

	tokens := 0
	contentType, ok := itemMap["type"].(string)
	if !ok {
		return 0, nil
	}

	switch contentType {
	case "text":
		if text, ok := itemMap["text"].(string); ok {
			tokens += s.estimateTokens(text)
		}
	case "image":
		// Images typically cost more tokens
		// This is a rough estimate - actual cost depends on image size and model
		tokens += 85 // Base cost for image processing
		if detail, ok := itemMap["detail"].(string); ok && detail == "high" {
			tokens += 170 // Additional cost for high detail
		}
	case "tool_use":
		if name, ok := itemMap["name"].(string); ok {
			tokens += s.estimateTokens(name)
		}
		if input, ok := itemMap["input"]; ok {
			inputBytes, err := json.Marshal(input)
			if err != nil {
				return 0, err
			}
			tokens += s.estimateTokens(string(inputBytes))
		}
	case "tool_result":
		if content, ok := itemMap["content"].(string); ok {
			tokens += s.estimateTokens(content)
		}
	}

	return tokens, nil
}

// countToolTokens counts tokens in a tool definition
func (s *TokenCountingService) countToolTokens(tool models.AnthropicTool) (int, error) {
	tokens := 0

	// Count name and description
	tokens += s.estimateTokens(tool.Name)
	tokens += s.estimateTokens(tool.Description)

	// Count input schema
	schemaBytes, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return 0, err
	}
	tokens += s.estimateTokens(string(schemaBytes))

	return tokens, nil
}

// estimateTokens provides a rough estimate of token count for text
// This is a simplified approximation - actual tokenization depends on the specific model
func (s *TokenCountingService) estimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// Remove extra whitespace
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	
	// Collapse multiple spaces
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	// Count characters and words
	charCount := utf8.RuneCountInString(text)
	wordCount := len(strings.Fields(text))

	// Rough estimation: 
	// - English text: ~4 characters per token
	// - But also consider word boundaries
	// - Add some overhead for special characters and formatting
	
	charBasedEstimate := charCount / 4
	wordBasedEstimate := int(float64(wordCount) * 1.3) // Words + some overhead for subword tokens
	
	// Use the higher estimate to be conservative
	estimate := charBasedEstimate
	if wordBasedEstimate > estimate {
		estimate = wordBasedEstimate
	}

	// Minimum of 1 token for non-empty text
	if estimate == 0 && text != "" {
		estimate = 1
	}

	return estimate
}

// GetModelTokenLimits returns the token limits for different models
func (s *TokenCountingService) GetModelTokenLimits(model string) (inputLimit, outputLimit int) {
	switch {
	case strings.Contains(model, "claude-3-opus"):
		return 200000, 4096
	case strings.Contains(model, "claude-3-sonnet"):
		return 200000, 4096
	case strings.Contains(model, "claude-3-haiku"):
		return 200000, 4096
	case strings.Contains(model, "claude-2.1"):
		return 200000, 4096
	case strings.Contains(model, "claude-2.0"):
		return 100000, 4096
	case strings.Contains(model, "claude-instant"):
		return 100000, 4096
	default:
		// Default limits
		return 100000, 4096
	}
}

// ValidateTokenLimits checks if the request exceeds model token limits
func (s *TokenCountingService) ValidateTokenLimits(req *models.TokenCountRequest) error {
	inputLimit, _ := s.GetModelTokenLimits(req.Model)
	
	// Count input tokens
	tokenResp, err := s.CountTokens(req)
	if err != nil {
		return err
	}

	if tokenResp.InputTokens > inputLimit {
		return fmt.Errorf("input tokens (%d) exceed model limit (%d)", tokenResp.InputTokens, inputLimit)
	}

	return nil
}

// EstimateResponseTokens estimates how many tokens a response might use
func (s *TokenCountingService) EstimateResponseTokens(maxTokens int, model string) int {
	_, outputLimit := s.GetModelTokenLimits(model)
	
	if maxTokens > 0 && maxTokens < outputLimit {
		return maxTokens
	}
	
	return outputLimit
}
