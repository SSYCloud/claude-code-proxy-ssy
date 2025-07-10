package models

// AnthropicRequest represents the request structure for Anthropic API
type AnthropicRequest struct {
	Model         string                 `json:"model" binding:"required"`
	MaxTokens     int                    `json:"max_tokens" binding:"required"`
	Messages      []AnthropicMessage     `json:"messages" binding:"required"`
	System        interface{}            `json:"system,omitempty"` // Can be string or array
	Temperature   *float64               `json:"temperature,omitempty"`
	TopP          *float64               `json:"top_p,omitempty"`
	TopK          *int                   `json:"top_k,omitempty"`
	StopSequences []string               `json:"stop_sequences,omitempty"`
	Stream        bool                   `json:"stream,omitempty"`
	Tools         []AnthropicTool        `json:"tools,omitempty"`
	ToolChoice    *AnthropicToolChoice   `json:"tool_choice,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AnthropicMessage represents a message in the conversation
type AnthropicMessage struct {
	Role    string      `json:"role" binding:"required"`
	Content interface{} `json:"content" binding:"required"`
}

// AnthropicTool represents a tool that can be used
type AnthropicTool struct {
	Name         string                 `json:"name" binding:"required"`
	Description  string                 `json:"description,omitempty"`
	InputSchema  map[string]interface{} `json:"input_schema" binding:"required"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicToolChoice represents tool choice configuration
type AnthropicToolChoice struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

// AnthropicResponse represents the response structure from Anthropic API
type AnthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []AnthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason"`
	StopSequence string             `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage     `json:"usage"`
}

// AnthropicContent represents content in the response
type AnthropicContent struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text,omitempty"`
	Source       *AnthropicImageSource  `json:"source,omitempty"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
	// For tool use
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
	// For tool results
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`
}

// GetContentBlocks safely converts content interface{} to content blocks
func (ac *AnthropicContent) GetContentBlocks(content interface{}) []AnthropicContentBlock {
	var blocks []AnthropicContentBlock

	switch v := content.(type) {
	case string:
		blocks = append(blocks, AnthropicContentBlock{
			Type: "text",
			Text: v,
		})
	case []interface{}:
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				block := AnthropicContentBlock{}
				if blockType, exists := itemMap["type"]; exists {
					if typeStr, ok := blockType.(string); ok {
						block.Type = typeStr
					}
				}
				if text, exists := itemMap["text"]; exists {
					if textStr, ok := text.(string); ok {
						block.Text = textStr
					}
				}
				// Handle cache_control
				if cacheControl, exists := itemMap["cache_control"]; exists {
					if ccMap, ok := cacheControl.(map[string]interface{}); ok {
						if ccType, exists := ccMap["type"]; exists {
							if ccTypeStr, ok := ccType.(string); ok {
								block.CacheControl = &AnthropicCacheControl{
									Type: ccTypeStr,
								}
							}
						}
					}
				}
				// Handle tool use
				if id, exists := itemMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						block.ID = idStr
					}
				}
				if name, exists := itemMap["name"]; exists {
					if nameStr, ok := name.(string); ok {
						block.Name = nameStr
					}
				}
				if input, exists := itemMap["input"]; exists {
					if inputMap, ok := input.(map[string]interface{}); ok {
						block.Input = inputMap
					}
				}
				// Handle tool results
				if toolUseID, exists := itemMap["tool_use_id"]; exists {
					if toolUseIDStr, ok := toolUseID.(string); ok {
						block.ToolUseID = toolUseIDStr
					}
				}
				if blockContent, exists := itemMap["content"]; exists {
					block.Content = blockContent
				}
				blocks = append(blocks, block)
			}
		}
	}

	return blocks
}

// AnthropicContentBlock represents a structured content block
type AnthropicContentBlock struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text,omitempty"`
	Source       *AnthropicImageSource  `json:"source,omitempty"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
	// For tool use
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
	// For tool results
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`
}

// AnthropicImageSource represents image source information
type AnthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// AnthropicCacheControl represents cache control settings
type AnthropicCacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// AnthropicUsage represents token usage information
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicStreamResponse represents a streaming response chunk
type AnthropicStreamResponse struct {
	Type    string             `json:"type"`
	Message *AnthropicResponse `json:"message,omitempty"`
	Index   int                `json:"index,omitempty"`
	Delta   *AnthropicDelta    `json:"delta,omitempty"`
	Usage   *AnthropicUsage    `json:"usage,omitempty"`
}

// AnthropicDelta represents incremental content in streaming
type AnthropicDelta struct {
	Type         string `json:"type,omitempty"`
	Text         string `json:"text,omitempty"`
	PartialJSON  string `json:"partial_json,omitempty"` // For tool argument streaming
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

// TokenCountRequest represents a request to count tokens
type TokenCountRequest struct {
	Model    string             `json:"model" binding:"required"`
	Messages []AnthropicMessage `json:"messages" binding:"required"`
	System   interface{}        `json:"system,omitempty"` // Can be string or array
	Tools    []AnthropicTool    `json:"tools,omitempty"`
}

// TokenCountResponse represents the response for token counting
type TokenCountResponse struct {
	InputTokens int `json:"input_tokens"`
}

// OpenAIRequest represents the converted request for OpenAI API
type OpenAIRequest struct {
	Model            string          `json:"model"`
	Messages         []OpenAIMessage `json:"messages"`
	MaxTokens        int             `json:"max_tokens,omitempty"`
	Temperature      *float64        `json:"temperature,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	Stop             []string        `json:"stop,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
	Tools            []OpenAITool    `json:"tools,omitempty"`
	ToolChoice       interface{}     `json:"tool_choice,omitempty"`
	User             string          `json:"user,omitempty"`
	FrequencyPenalty *float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64        `json:"presence_penalty,omitempty"`
}

// OpenAIMessage represents a message in OpenAI format
type OpenAIMessage struct {
	Role         string                 `json:"role"`
	Content      interface{}            `json:"content,omitempty"` // Can be string or []OpenAIContentPart
	Name         string                 `json:"name,omitempty"`
	ToolCalls    []OpenAIToolCall       `json:"tool_calls,omitempty"`
	ToolCallID   string                 `json:"tool_call_id,omitempty"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"` // For tool messages
}

// OpenAIContentPart represents a content part in OpenAI format (for multimodal content)
type OpenAIContentPart struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text,omitempty"`
	ImageURL     *OpenAIImageURL        `json:"image_url,omitempty"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// OpenAIImageURL represents an image URL in OpenAI format
type OpenAIImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// OpenAITool represents a tool in OpenAI format
type OpenAITool struct {
	Type         string                 `json:"type"`
	Function     OpenAIFunction         `json:"function"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// OpenAIFunction represents a function definition
type OpenAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// OpenAIToolCall represents a tool call
type OpenAIToolCall struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Index        int                    `json:"index,omitempty"` // For streaming
	Function     OpenAIFunctionCall     `json:"function"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// OpenAIFunctionCall represents a function call
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAIResponse represents the response from OpenAI API
type OpenAIResponse struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	Choices           []OpenAIChoice `json:"choices"`
	Usage             OpenAIUsage    `json:"usage"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
}

// OpenAIChoice represents a choice in the response
type OpenAIChoice struct {
	Index        int            `json:"index"`
	Message      OpenAIMessage  `json:"message,omitempty"`
	Delta        *OpenAIMessage `json:"delta,omitempty"`
	FinishReason string         `json:"finish_reason"`
	Logprobs     interface{}    `json:"logprobs,omitempty"`
}

// OpenAIUsage represents usage information
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIStreamResponse represents a streaming response chunk from OpenAI
type OpenAIStreamResponse struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	Choices           []OpenAIChoice `json:"choices"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
}
