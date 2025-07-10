package services

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"claude-code-provider-proxy/internal/config"
	"claude-code-provider-proxy/internal/models"
)

// ConversionService handles conversion between Anthropic and OpenAI formats
type ConversionService struct {
	modelSelector *ModelSelectorService
	config        *config.Config
}

// NewConversionService creates a new conversion service
func NewConversionService(modelSelector *ModelSelectorService, cfg *config.Config) *ConversionService {
	return &ConversionService{
		modelSelector: modelSelector,
		config:        cfg,
	}
}

// isClaudeModel checks if cache control should be enabled for the target model
func (s *ConversionService) isClaudeModel(targetModelName string) bool {
	return s.config.OpenClaudeCache && strings.Contains(strings.ToLower(targetModelName), "claude")
}

// ConvertAnthropicToOpenAI converts an Anthropic request to OpenAI format
func (s *ConversionService) ConvertAnthropicToOpenAI(req *models.AnthropicRequest, fallbackModel string) (*models.OpenAIRequest, error) {
	// Use model selector to choose the appropriate model
	selectedModel := fallbackModel
	if s.modelSelector != nil {
		selectedModel = s.modelSelector.SelectModel(req.Model, req)
	}

	openAIReq := &models.OpenAIRequest{
		Model:       selectedModel,
		MaxTokens:   req.MaxTokens, // Use the max_tokens from the original request
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}

	// Convert stop sequences
	if len(req.StopSequences) > 0 {
		openAIReq.Stop = req.StopSequences
	}

	// Convert messages
	messages, err := s.convertMessages(req.Messages, req.System, selectedModel)
	if err != nil {
		return nil, err
	}
	openAIReq.Messages = messages

	// Convert tools
	if len(req.Tools) > 0 {
		tools, err := s.convertTools(req.Tools, selectedModel)
		if err != nil {
			return nil, err
		}
		openAIReq.Tools = tools

		// Convert tool choice
		if req.ToolChoice != nil {
			toolChoice, err := s.convertToolChoice(req.ToolChoice)
			if err != nil {
				return nil, err
			}
			openAIReq.ToolChoice = toolChoice
		}
	}

	return openAIReq, nil
}

// convertMessages converts Anthropic messages to OpenAI format
func (s *ConversionService) convertMessages(anthropicMessages []models.AnthropicMessage, system interface{}, targetModel string) ([]models.OpenAIMessage, error) {
	var messages []models.OpenAIMessage

	// Add system message if provided
	if system != nil {
		systemContent, err := s.convertSystemContent(system, targetModel)
		if err != nil {
			return nil, fmt.Errorf("failed to convert system content: %w", err)
		}
		// Check if we should add it as a system message
		if stringContent, ok := systemContent.(string); ok && stringContent != "" {
			messages = append(messages, models.OpenAIMessage{
				Role:    "system",
				Content: stringContent,
			})
		} else {
			// For structured content (e.g., with cache_control)
			messages = append(messages, models.OpenAIMessage{
				Role:    "system",
				Content: systemContent,
			})
		}
	}

	// Convert each message while preserving order
	for i, msg := range anthropicMessages {
		convertedMessages, err := s.convertSingleMessage(msg, i, targetModel)
		if err != nil {
			return nil, fmt.Errorf("failed to convert message at index %d: %w", i, err)
		}
		messages = append(messages, convertedMessages...)
	}

	return messages, nil
}

// convertSingleMessage converts a single Anthropic message to one or more OpenAI messages
func (s *ConversionService) convertSingleMessage(msg models.AnthropicMessage, index int, targetModel string) ([]models.OpenAIMessage, error) {
	var messages []models.OpenAIMessage

	// Handle different content types
	switch content := msg.Content.(type) {
	case string:
		// Simple string content
		messages = append(messages, models.OpenAIMessage{
			Role:    msg.Role,
			Content: content,
		})
	case []interface{}:
		// Complex content (text + images, tool calls, tool results, etc.)
		convertedMessages, err := s.convertComplexMessage(msg.Role, content, index, targetModel)
		if err != nil {
			return nil, err
		}
		messages = append(messages, convertedMessages...)
	default:
		// Try to convert to string
		if contentBytes, err := json.Marshal(content); err == nil {
			messages = append(messages, models.OpenAIMessage{
				Role:    msg.Role,
				Content: string(contentBytes),
			})
		} else {
			return nil, fmt.Errorf("unsupported content type: %T", content)
		}
	}

	return messages, nil
}

// convertComplexMessage handles complex message content and returns multiple OpenAI messages if needed
func (s *ConversionService) convertComplexMessage(role string, content []interface{}, messageIndex int, targetModel string) ([]models.OpenAIMessage, error) {
	var messages []models.OpenAIMessage

	if role == "user" {
		return s.convertUserMessage(content, messageIndex, targetModel)
	} else if role == "assistant" {
		return s.convertAssistantMessage(content, messageIndex, targetModel)
	}

	return messages, nil
}

// convertUserMessage converts user message content to OpenAI format
func (s *ConversionService) convertUserMessage(content []interface{}, messageIndex int, targetModel string) ([]models.OpenAIMessage, error) {
	var messages []models.OpenAIMessage
	var userContentParts []interface{}
	isClaudeModel := s.isClaudeModel(targetModel)

	for contentIndex, item := range content {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		contentType, ok := itemMap["type"].(string)
		if !ok {
			continue
		}

		// Check for cache_control in this content item
		if cacheControl, exists := itemMap["cache_control"]; exists {
			fmt.Printf("Cache control found in message %d, content %d: %+v\n", messageIndex, contentIndex, cacheControl)
		}

		switch contentType {
		case "text":
			if text, ok := itemMap["text"].(string); ok {
				textPart := map[string]interface{}{
					"type": "text",
					"text": text,
				}
				// Preserve cache_control for Claude models
				if isClaudeModel {
					if cacheControl, exists := itemMap["cache_control"]; exists && cacheControl != nil {
						textPart["cache_control"] = cacheControl
					}
				}
				userContentParts = append(userContentParts, textPart)
			}
		case "image":
			// Handle image content for user messages
			if source, ok := itemMap["source"].(map[string]interface{}); ok {
				if sourceType, ok := source["type"].(string); ok && sourceType == "base64" {
					if mediaType, ok := source["media_type"].(string); ok {
						if data, ok := source["data"].(string); ok {
							imagePart := map[string]interface{}{
								"type": "image_url",
								"image_url": map[string]string{
									"url": fmt.Sprintf("data:%s;base64,%s", mediaType, data),
								},
							}
							// Preserve cache_control for Claude models
							if isClaudeModel {
								if cacheControl, exists := itemMap["cache_control"]; exists && cacheControl != nil {
									imagePart["cache_control"] = cacheControl
								}
							}
							userContentParts = append(userContentParts, imagePart)
						}
					}
				}
			}
		case "tool_result":
			// Tool results should be converted to separate "tool" role messages
			toolResultMsg, err := s.convertToolResultToMessage(itemMap, isClaudeModel)
			if err != nil {
				return nil, fmt.Errorf("failed to convert tool result: %w", err)
			}
			messages = append(messages, toolResultMsg)
		default:
			// Handle unknown content types
			unknownBytes, err := json.Marshal(itemMap)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal unknown content type %s: %w", contentType, err)
			}
			textPart := map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("[UNKNOWN_CONTENT_TYPE:%s] %s", contentType, string(unknownBytes)),
			}
			userContentParts = append(userContentParts, textPart)
		}
	}

	// Add user message with collected content parts if any
	if len(userContentParts) > 0 {
		userMsg := models.OpenAIMessage{
			Role: "user",
		}

		// Check if we have multiple parts or need complex content structure
		if len(userContentParts) == 1 {
			if textPart, ok := userContentParts[0].(map[string]interface{}); ok {
				if textPart["type"] == "text" && !isClaudeModel {
					// Single text part can be simplified for non-Claude models
					userMsg.Content = textPart["text"].(string)
				} else {
					// Non-text content or Claude model needs array format
					userMsg.Content = userContentParts
				}
			}
		} else {
			// Multiple parts need array format
			userMsg.Content = userContentParts
		}

		messages = append(messages, userMsg)
	}

	return messages, nil
}

// convertAssistantMessage converts assistant message content to OpenAI format
func (s *ConversionService) convertAssistantMessage(content []interface{}, messageIndex int, targetModel string) ([]models.OpenAIMessage, error) {
	var messages []models.OpenAIMessage
	var textParts []string
	var toolCalls []models.OpenAIToolCall
	isClaudeModel := s.isClaudeModel(targetModel)

	for contentIndex, item := range content {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		contentType, ok := itemMap["type"].(string)
		if !ok {
			continue
		}

		// Check for cache_control in this content item
		if cacheControl, exists := itemMap["cache_control"]; exists {
			fmt.Printf("Cache control found in message %d, content %d: %+v\n", messageIndex, contentIndex, cacheControl)
		}

		switch contentType {
		case "text":
			if text, ok := itemMap["text"].(string); ok {
				textParts = append(textParts, text)
			}
		case "tool_use":
			toolCall, err := s.convertToolUse(itemMap, isClaudeModel)
			if err != nil {
				return nil, err
			}
			toolCalls = append(toolCalls, toolCall)
		default:
			// Handle unknown content types by adding to text
			unknownBytes, err := json.Marshal(itemMap)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal unknown content type %s: %w", contentType, err)
			}
			textParts = append(textParts, fmt.Sprintf("[UNKNOWN_CONTENT_TYPE:%s] %s", contentType, string(unknownBytes)))
		}
	}

	// Create assistant messages based on what we found
	assistantText := strings.Join(textParts, "\n")

	if assistantText != "" && len(toolCalls) > 0 {
		// Both text and tool calls - create two separate messages
		messages = append(messages, models.OpenAIMessage{
			Role:    "assistant",
			Content: assistantText,
		})
		messages = append(messages, models.OpenAIMessage{
			Role:      "assistant",
			ToolCalls: toolCalls,
		})
	} else if assistantText != "" {
		// Only text
		messages = append(messages, models.OpenAIMessage{
			Role:    "assistant",
			Content: assistantText,
		})
	} else if len(toolCalls) > 0 {
		// Only tool calls
		messages = append(messages, models.OpenAIMessage{
			Role:      "assistant",
			ToolCalls: toolCalls,
		})
	}

	return messages, nil
}

// convertToolResultToMessage converts a tool result to an OpenAI "tool" role message
func (s *ConversionService) convertToolResultToMessage(toolResult map[string]interface{}, isClaudeModel bool) (models.OpenAIMessage, error) {
	var toolMsg models.OpenAIMessage
	toolMsg.Role = "tool"

	// Get tool_use_id
	if toolUseID, ok := toolResult["tool_use_id"].(string); ok {
		toolMsg.ToolCallID = toolUseID
	} else {
		return toolMsg, fmt.Errorf("tool_result missing tool_use_id")
	}

	// Handle content
	var contentParts []string

	if isError, ok := toolResult["is_error"].(bool); ok && isError {
		contentParts = append(contentParts, "[ERROR]")
	}

	if content, ok := toolResult["content"]; ok {
		switch c := content.(type) {
		case string:
			contentParts = append(contentParts, c)
		case []interface{}:
			// Handle complex tool result content
			for _, item := range c {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if itemType, ok := itemMap["type"].(string); ok && itemType == "text" {
						if text, ok := itemMap["text"].(string); ok {
							contentParts = append(contentParts, text)
						}
					}
				}
			}
		default:
			contentBytes, err := json.Marshal(content)
			if err != nil {
				return toolMsg, err
			}
			contentParts = append(contentParts, string(contentBytes))
		}
	}

	toolMsg.Content = strings.Join(contentParts, " ")

	// Preserve cache_control for Claude models
	if isClaudeModel {
		if cacheControl, exists := toolResult["cache_control"]; exists && cacheControl != nil {
			// Set cache_control directly on the message
			if cacheMap, ok := cacheControl.(map[string]interface{}); ok {
				toolMsg.CacheControl = &models.AnthropicCacheControl{}
				if cacheType, ok := cacheMap["type"].(string); ok {
					toolMsg.CacheControl.Type = cacheType
				}
			}
		}
	}

	return toolMsg, nil
}

// convertSystemContent converts system content which can be string or array
func (s *ConversionService) convertSystemContent(system interface{}, targetModel string) (interface{}, error) {
	isClaudeModel := s.isClaudeModel(targetModel)

	switch sys := system.(type) {
	case string:
		// Simple string content
		return sys, nil
	case []interface{}:
		if isClaudeModel {
			// For Claude models, convert to OpenAI content parts format with cache_control preserved
			var contentParts []models.OpenAIContentPart
			for _, item := range sys {
				if itemMap, ok := item.(map[string]interface{}); ok {
					contentType, _ := itemMap["type"].(string)
					if contentType == "text" {
						part := models.OpenAIContentPart{
							Type: "text",
						}
						if text, ok := itemMap["text"].(string); ok {
							part.Text = text
						}
						// Preserve cache_control if present
						if cacheControl, exists := itemMap["cache_control"]; exists && cacheControl != nil {
							if cacheMap, ok := cacheControl.(map[string]interface{}); ok {
								part.CacheControl = &models.AnthropicCacheControl{}
								if cacheType, ok := cacheMap["type"].(string); ok {
									part.CacheControl.Type = cacheType
								}
							}
						}
						contentParts = append(contentParts, part)
					}
				} else if textStr, ok := item.(string); ok {
					// Plain string item
					contentParts = append(contentParts, models.OpenAIContentPart{
						Type: "text",
						Text: textStr,
					})
				}
			}
			return contentParts, nil
		} else {
			// For non-Claude models, concatenate text parts
			var textParts []string
			for _, item := range sys {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if contentType, ok := itemMap["type"].(string); ok && contentType == "text" {
						if text, ok := itemMap["text"].(string); ok {
							textParts = append(textParts, text)
						}
					}
				} else if textStr, ok := item.(string); ok {
					textParts = append(textParts, textStr)
				}
			}
			return strings.Join(textParts, "\n"), nil
		}
	default:
		// Try to convert to JSON string as fallback
		if systemBytes, err := json.Marshal(system); err == nil {
			return string(systemBytes), nil
		}
		return "", fmt.Errorf("unsupported system content type: %T", system)
	}
}

// convertToolUse converts Anthropic tool use to OpenAI tool call
func (s *ConversionService) convertToolUse(toolUse map[string]interface{}, isClaudeModel bool) (models.OpenAIToolCall, error) {
	toolCall := models.OpenAIToolCall{
		Type: "function",
	}

	// Use provided ID or generate one if missing
	if id, ok := toolUse["id"].(string); ok && id != "" {
		toolCall.ID = id
	} else {
		toolCall.ID = s.generateToolUseID()
	}

	if name, ok := toolUse["name"].(string); ok {
		toolCall.Function.Name = name
	}

	// Handle input parameters more robustly
	if input, ok := toolUse["input"]; ok {
		switch inp := input.(type) {
		case map[string]interface{}:
			inputBytes, err := json.Marshal(inp)
			if err != nil {
				return toolCall, fmt.Errorf("failed to marshal tool input: %w", err)
			}
			toolCall.Function.Arguments = string(inputBytes)
		case string:
			// Already a JSON string
			toolCall.Function.Arguments = inp
		default:
			// Convert any other type to JSON
			inputBytes, err := json.Marshal(inp)
			if err != nil {
				return toolCall, fmt.Errorf("failed to marshal tool input of type %T: %w", inp, err)
			}
			toolCall.Function.Arguments = string(inputBytes)
		}
	} else {
		// Default empty object
		toolCall.Function.Arguments = "{}"
	}

	// For Claude models, preserve cache_control directly on the tool call
	if isClaudeModel {
		if cacheControl, exists := toolUse["cache_control"]; exists && cacheControl != nil {
			if cacheMap, ok := cacheControl.(map[string]interface{}); ok {
				toolCall.CacheControl = &models.AnthropicCacheControl{}
				if cacheType, ok := cacheMap["type"].(string); ok {
					toolCall.CacheControl.Type = cacheType
				}
			}
		}
	}

	return toolCall, nil
}

// convertTools converts Anthropic tools to OpenAI format
func (s *ConversionService) convertTools(anthropicTools []models.AnthropicTool, targetModel string) ([]models.OpenAITool, error) {
	var tools []models.OpenAITool
	isClaudeModel := s.isClaudeModel(targetModel)

	for _, tool := range anthropicTools {
		openAITool := models.OpenAITool{
			Type: "function",
			Function: models.OpenAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		}

		// For Claude models, preserve cache_control directly on the tool
		if isClaudeModel && tool.CacheControl != nil {
			openAITool.CacheControl = tool.CacheControl
		}

		tools = append(tools, openAITool)
	}

	return tools, nil
}

// convertToolChoice converts Anthropic tool choice to OpenAI format
func (s *ConversionService) convertToolChoice(choice *models.AnthropicToolChoice) (interface{}, error) {
	switch choice.Type {
	case "auto":
		return "auto", nil
	case "any":
		return "required", nil
	case "tool":
		if choice.Name != "" {
			return map[string]interface{}{
				"type": "function",
				"function": map[string]string{
					"name": choice.Name,
				},
			}, nil
		}
		return "required", nil
	default:
		return "auto", nil
	}
}

// ConvertOpenAIToAnthropic converts OpenAI response to Anthropic format
func (s *ConversionService) ConvertOpenAIToAnthropic(resp *models.OpenAIResponse, originalModel string) (*models.AnthropicResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	choice := resp.Choices[0]

	anthropicResp := &models.AnthropicResponse{
		ID:         resp.ID,
		Type:       "message",
		Role:       "assistant",
		Model:      originalModel,
		StopReason: s.convertFinishReason(choice.FinishReason),
		Usage: models.AnthropicUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	// Convert content
	content, err := s.convertOpenAIMessageContent(choice.Message)
	if err != nil {
		return nil, err
	}
	anthropicResp.Content = content

	return anthropicResp, nil
}

// convertOpenAIMessageContent converts OpenAI message content to Anthropic format
func (s *ConversionService) convertOpenAIMessageContent(msg models.OpenAIMessage) ([]models.AnthropicContent, error) {
	var content []models.AnthropicContent

	// Handle text content
	if msg.Content != nil {
		switch msgContent := msg.Content.(type) {
		case string:
			if msgContent != "" {
				// Simple string content
				content = append(content, models.AnthropicContent{
					Type: "text",
					Text: msgContent,
				})
			}
		case []interface{}:
			// Complex content with multiple parts (potentially with cache_control)
			for _, part := range msgContent {
				if partMap, ok := part.(map[string]interface{}); ok {
					if partType, ok := partMap["type"].(string); ok {
						switch partType {
						case "text":
							textContent := models.AnthropicContent{
								Type: "text",
							}
							if text, ok := partMap["text"].(string); ok {
								textContent.Text = text
							}
							// Restore cache_control if present
							if cacheControl, exists := partMap["cache_control"]; exists && cacheControl != nil {
								if cacheMap, ok := cacheControl.(map[string]interface{}); ok {
									textContent.CacheControl = &models.AnthropicCacheControl{}
									if cacheType, ok := cacheMap["type"].(string); ok {
										textContent.CacheControl.Type = cacheType
									}
								}
							}
							content = append(content, textContent)
						case "image_url":
							// Image content (not typically in responses, but handle it)
							textContent := models.AnthropicContent{
								Type: "text",
								Text: "[IMAGE_CONTENT]", // Placeholder
							}
							content = append(content, textContent)
						}
					}
				}
			}
		}
	}

	// Handle tool calls - convert to tool_use blocks
	for _, toolCall := range msg.ToolCalls {
		var input map[string]interface{}
		if toolCall.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input); err != nil {
				// If JSON parsing fails, store the raw arguments with error info
				input = map[string]interface{}{
					"error_parsing_arguments": toolCall.Function.Arguments,
				}
			}
		} else {
			input = map[string]interface{}{}
		}

		toolUseContent := models.AnthropicContent{
			Type:  "tool_use",
			ID:    toolCall.ID,
			Name:  toolCall.Function.Name,
			Input: input,
		}

		// Restore cache_control if present
		if toolCall.CacheControl != nil {
			toolUseContent.CacheControl = toolCall.CacheControl
		}

		content = append(content, toolUseContent)
	}

	// If no content, add empty text (same as Python project)
	if len(content) == 0 {
		content = append(content, models.AnthropicContent{
			Type: "text",
			Text: "",
		})
	}

	return content, nil
}

// convertFinishReason converts OpenAI finish reason to Anthropic format
func (s *ConversionService) convertFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	case "content_filter":
		return "stop_sequence"
	default:
		return "end_turn"
	}
}

// ConvertStreamResponse converts OpenAI streaming response to Anthropic format
func (s *ConversionService) ConvertStreamResponse(openAIResp *models.OpenAIStreamResponse, originalModel string) (*models.AnthropicStreamResponse, error) {
	if len(openAIResp.Choices) == 0 {
		return &models.AnthropicStreamResponse{
			Type: "ping",
		}, nil
	}

	choice := openAIResp.Choices[0]

	// Handle different streaming event types while preserving order
	if choice.Delta != nil {
		// Handle text content delta
		if choice.Delta.Content != nil {
			if textContent, ok := choice.Delta.Content.(string); ok && textContent != "" {
				// Check for cache control markers
				deltaType := "text_delta"
				textContent = strings.TrimPrefix(textContent, "[CACHE_CONTROL] ")

				return &models.AnthropicStreamResponse{
					Type:  "content_block_delta",
					Index: choice.Index,
					Delta: &models.AnthropicDelta{
						Type: deltaType,
						Text: textContent,
					},
				}, nil
			}
		}

		// Handle tool calls in streaming
		if len(choice.Delta.ToolCalls) > 0 {
			for _, toolCall := range choice.Delta.ToolCalls {
				// Handle tool call start
				if toolCall.Function.Name != "" {
					return &models.AnthropicStreamResponse{
						Type:  "content_block_start",
						Index: choice.Index + 1, // Tool blocks come after text blocks
						Delta: &models.AnthropicDelta{
							Type: "tool_use",
						},
					}, nil
				}

				// Handle tool arguments delta
				if toolCall.Function.Arguments != "" {
					return &models.AnthropicStreamResponse{
						Type:  "content_block_delta",
						Index: choice.Index + 1,
						Delta: &models.AnthropicDelta{
							Type:        "input_json_delta",
							PartialJSON: toolCall.Function.Arguments,
						},
					}, nil
				}
			}
		}
	}

	// Handle finish reason
	if choice.FinishReason != "" {
		stopReason := s.convertFinishReason(choice.FinishReason)
		return &models.AnthropicStreamResponse{
			Type: "message_delta",
			Delta: &models.AnthropicDelta{
				StopReason: stopReason,
			},
			// Add usage if available
			Usage: &models.AnthropicUsage{
				InputTokens:  0, // Would be calculated
				OutputTokens: 0, // Would be calculated
			},
		}, nil
	}

	// Default return for ping
	return &models.AnthropicStreamResponse{
		Type: "ping",
	}, nil
}

// PreserveCacheControlInfo extracts and preserves cache control information
func (s *ConversionService) PreserveCacheControlInfo(content interface{}) (interface{}, map[string]interface{}) {
	cacheInfo := make(map[string]interface{})

	switch c := content.(type) {
	case []interface{}:
		for i, item := range c {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if cacheControl, exists := itemMap["cache_control"]; exists {
					cacheInfo[fmt.Sprintf("item_%d", i)] = cacheControl
				}
			}
		}
	case map[string]interface{}:
		if cacheControl, exists := c["cache_control"]; exists {
			cacheInfo["root"] = cacheControl
		}
	}

	return content, cacheInfo
}

// generateToolUseID generates a unique tool use ID in Anthropic format
func (s *ConversionService) generateToolUseID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("toolu_%s", hex.EncodeToString(bytes))
}

// RestoreCacheControlInfo restores cache control information to converted content
func (s *ConversionService) RestoreCacheControlInfo(content []models.AnthropicContent, cacheInfo map[string]interface{}) []models.AnthropicContent {
	for i := range content {
		// Check for cache control info for this item
		if cacheData, exists := cacheInfo[fmt.Sprintf("item_%d", i)]; exists {
			if cacheMap, ok := cacheData.(map[string]interface{}); ok {
				if cacheType, ok := cacheMap["type"].(string); ok {
					content[i].CacheControl = &models.AnthropicCacheControl{
						Type: cacheType,
					}
				}
			}
		}

		// Check for root level cache control
		if cacheData, exists := cacheInfo["root"]; exists {
			if cacheMap, ok := cacheData.(map[string]interface{}); ok {
				if cacheType, ok := cacheMap["type"].(string); ok {
					content[i].CacheControl = &models.AnthropicCacheControl{
						Type: cacheType,
					}
				}
			}
		}
	}

	return content
}
