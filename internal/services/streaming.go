package services

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"claude-code-provider-proxy/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// StreamingService handles streaming responses
type StreamingService struct {
	conversionService *ConversionService
	logger            *logrus.Logger
	// Track streaming state
	currentContentBlockIndex int
	toolCallStates           map[int]*ToolCallState
	messageID                string
	outputTokens             int
	hasStartedTextBlock      bool
	hasStartedToolBlocks     map[int]bool
}

// ToolCallState tracks the state of a tool call during streaming
type ToolCallState struct {
	ID              string
	Name            string
	ArgumentsBuffer string
	AnthropicIndex  int
	OpenAIIndex     int
	HasSentStart    bool
}

// NewStreamingService creates a new streaming service
func NewStreamingService(conversionService *ConversionService, logger *logrus.Logger) *StreamingService {
	return &StreamingService{
		conversionService:    conversionService,
		logger:               logger,
		toolCallStates:       make(map[int]*ToolCallState),
		hasStartedToolBlocks: make(map[int]bool),
	}
}

// StreamResponse handles streaming response from OpenAI and converts to Anthropic format
func (s *StreamingService) StreamResponse(c *gin.Context, resp *http.Response, originalModel string) error {
	// Initialize streaming state
	s.resetStreamingState()
	s.messageID = s.generateMessageID()

	// Set headers for Server-Sent Events
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Create a scanner to read the response line by line
	scanner := bufio.NewScanner(resp.Body)
	defer resp.Body.Close()

	// Send initial message_start event
	if err := s.sendMessageStart(c, originalModel); err != nil {
		return err
	}

	// Send initial ping
	if err := s.writeStreamEvent(c, "ping", map[string]interface{}{
		"type": "ping",
	}); err != nil {
		return err
	}

	// Process each line from the stream
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse Server-Sent Events format
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Check for stream end
			if data == "[DONE]" {
				s.logger.Debug("Stream ended with [DONE]")
				break
			}

			// Debug log: raw streaming data
			s.logger.WithFields(logrus.Fields{
				"data": data,
			}).Debug("Streaming data received")

			// Parse the JSON data
			var openAIResp models.OpenAIStreamResponse
			if err := json.Unmarshal([]byte(data), &openAIResp); err != nil {
				s.logger.WithFields(logrus.Fields{
					"error": err.Error(),
					"data":  data,
				}).Warn("Failed to parse streaming response")
				continue
			}

			// Process the chunk
			if err := s.processStreamChunk(c, &openAIResp, originalModel); err != nil {
				s.logger.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("Failed to process stream chunk")
				return err
			}

			// Flush the response
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		s.logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Scanner error during streaming")
		return err
	}

	s.logger.Debug("Stream processing completed successfully")

	// Send final events
	if err := s.sendStreamEnd(c); err != nil {
		return err
	}

	return scanner.Err()
}

// writeStreamEvent writes a Server-Sent Event to the response
func (s *StreamingService) writeStreamEvent(c *gin.Context, eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Write in Server-Sent Events format
	if _, err := fmt.Fprintf(c.Writer, "event: %s\n", eventType); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonData)); err != nil {
		return err
	}

	return nil
}

// StreamOpenAIRequest makes a streaming request to OpenAI API
func (s *StreamingService) StreamOpenAIRequest(ctx context.Context, client *http.Client, url string, headers map[string]string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set streaming specific headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// HandleStreamingError handles errors during streaming
func (s *StreamingService) HandleStreamingError(c *gin.Context, err error) {
	s.logger.WithError(err).Error("Streaming error")

	// Send error event
	errorEvent := map[string]interface{}{
		"type": "error",
		"error": map[string]interface{}{
			"type":    "api_error",
			"message": err.Error(),
		},
	}

	if streamErr := s.writeStreamEvent(c, "error", errorEvent); streamErr != nil {
		s.logger.WithError(streamErr).Error("Failed to write error event")
	}

	// Flush the response
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

// resetStreamingState resets the streaming state for a new request
func (s *StreamingService) resetStreamingState() {
	s.currentContentBlockIndex = 0
	s.toolCallStates = make(map[int]*ToolCallState)
	s.outputTokens = 0
	s.hasStartedTextBlock = false
	s.hasStartedToolBlocks = make(map[int]bool)
}

// generateMessageID generates a unique message ID
func (s *StreamingService) generateMessageID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("msg_%s", hex.EncodeToString(bytes))
}

// sendMessageStart sends the initial message_start event
func (s *StreamingService) sendMessageStart(c *gin.Context, originalModel string) error {
	return s.writeStreamEvent(c, "message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":            s.messageID,
			"type":          "message",
			"role":          "assistant",
			"content":       []interface{}{},
			"model":         originalModel,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]int{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	})
}

// processStreamChunk processes a single streaming chunk
func (s *StreamingService) processStreamChunk(c *gin.Context, openAIResp *models.OpenAIStreamResponse, originalModel string) error {
	if len(openAIResp.Choices) == 0 {
		return nil
	}

	choice := openAIResp.Choices[0]

	// Handle text content
	if choice.Delta != nil && choice.Delta.Content != nil {
		if textContent, ok := choice.Delta.Content.(string); ok && textContent != "" {
			return s.handleTextDelta(c, textContent)
		}
	}

	// Handle tool calls
	if choice.Delta != nil && len(choice.Delta.ToolCalls) > 0 {
		return s.handleToolCallDeltas(c, choice.Delta.ToolCalls)
	}

	// Handle finish reason
	if choice.FinishReason != "" {
		return s.handleFinishReason(c, choice.FinishReason)
	}

	return nil
}

// handleTextDelta handles text content streaming
func (s *StreamingService) handleTextDelta(c *gin.Context, textContent string) error {
	// Start text block if not started
	if !s.hasStartedTextBlock {
		if err := s.writeStreamEvent(c, "content_block_start", map[string]interface{}{
			"type":  "content_block_start",
			"index": s.currentContentBlockIndex,
			"content_block": map[string]interface{}{
				"type": "text",
				"text": "",
			},
		}); err != nil {
			return err
		}
		s.hasStartedTextBlock = true
	}

	// Send text delta
	return s.writeStreamEvent(c, "content_block_delta", map[string]interface{}{
		"type":  "content_block_delta",
		"index": s.currentContentBlockIndex,
		"delta": map[string]interface{}{
			"type": "text_delta",
			"text": textContent,
		},
	})
}

// handleToolCallDeltas handles tool call streaming
func (s *StreamingService) handleToolCallDeltas(c *gin.Context, toolCalls []models.OpenAIToolCall) error {
	for _, toolCall := range toolCalls {
		openAIIndex := toolCall.Index

		// Get or create tool call state
		state, exists := s.toolCallStates[openAIIndex]
		if !exists {
			s.currentContentBlockIndex++
			state = &ToolCallState{
				AnthropicIndex: s.currentContentBlockIndex,
				OpenAIIndex:    openAIIndex,
			}
			s.toolCallStates[openAIIndex] = state
		}

		// Update state
		if toolCall.ID != "" {
			state.ID = toolCall.ID
		}
		if toolCall.Function.Name != "" {
			state.Name = toolCall.Function.Name
		}
		if toolCall.Function.Arguments != "" {
			state.ArgumentsBuffer += toolCall.Function.Arguments
		}

		// Send content_block_start if needed
		if !state.HasSentStart && state.ID != "" && state.Name != "" {
			if err := s.writeStreamEvent(c, "content_block_start", map[string]interface{}{
				"type":  "content_block_start",
				"index": state.AnthropicIndex,
				"content_block": map[string]interface{}{
					"type":  "tool_use",
					"id":    state.ID,
					"name":  state.Name,
					"input": map[string]interface{}{},
				},
			}); err != nil {
				return err
			}
			state.HasSentStart = true
		}

		// Send arguments delta if we have started and there are new arguments
		if state.HasSentStart && toolCall.Function.Arguments != "" {
			if err := s.writeStreamEvent(c, "content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": state.AnthropicIndex,
				"delta": map[string]interface{}{
					"type":         "input_json_delta",
					"partial_json": toolCall.Function.Arguments,
				},
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// handleFinishReason handles the finish reason and sends final events
func (s *StreamingService) handleFinishReason(c *gin.Context, finishReason string) error {
	// Send content_block_stop for text if it was started
	if s.hasStartedTextBlock {
		if err := s.writeStreamEvent(c, "content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": 0,
		}); err != nil {
			return err
		}
	}

	// Send content_block_stop for each tool call
	for _, state := range s.toolCallStates {
		if state.HasSentStart {
			if err := s.writeStreamEvent(c, "content_block_stop", map[string]interface{}{
				"type":  "content_block_stop",
				"index": state.AnthropicIndex,
			}); err != nil {
				return err
			}
		}
	}

	// Convert finish reason
	anthropicStopReason := s.convertFinishReason(finishReason)

	// Send message_delta with stop reason and usage
	return s.writeStreamEvent(c, "message_delta", map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason":   anthropicStopReason,
			"stop_sequence": nil,
		},
		"usage": map[string]int{
			"output_tokens": s.outputTokens,
		},
	})
}

// sendStreamEnd sends the final message_stop event
func (s *StreamingService) sendStreamEnd(c *gin.Context) error {
	return s.writeStreamEvent(c, "message_stop", map[string]interface{}{
		"type": "message_stop",
	})
}

// convertFinishReason converts OpenAI finish reason to Anthropic format
func (s *StreamingService) convertFinishReason(reason string) string {
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
