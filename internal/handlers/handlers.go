package handlers

import (
	"context"
	"net/http"
	"time"

	"claude-code-provider-proxy/internal/config"
	"claude-code-provider-proxy/internal/models"
	"claude-code-provider-proxy/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler contains all the dependencies for handling HTTP requests
type Handler struct {
	config            *config.Config
	logger            *logrus.Logger
	openAIClient      *services.OpenAIClient
	conversionService *services.ConversionService
	tokenService      *services.TokenCountingService
	streamingService  *services.StreamingService
	modelSelector     *services.ModelSelectorService
}

// NewHandler creates a new handler instance
func NewHandler(
	cfg *config.Config,
	logger *logrus.Logger,
	openAIClient *services.OpenAIClient,
	conversionService *services.ConversionService,
	tokenService *services.TokenCountingService,
	streamingService *services.StreamingService,
	modelSelector *services.ModelSelectorService,
) *Handler {
	return &Handler{
		config:            cfg,
		logger:            logger,
		openAIClient:      openAIClient,
		conversionService: conversionService,
		tokenService:      tokenService,
		streamingService:  streamingService,
		modelSelector:     modelSelector,
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"app_name":  h.config.AppName,
		"version":   h.config.AppVersion,
		"referrer":  h.config.ReferrerURL,
	})
}

// CreateMessage handles Anthropic-compatible message creation
func (h *Handler) CreateMessage(c *gin.Context) {
	var req models.AnthropicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request format")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.FormatValidationError(err),
		})
		return
	}

	// Validate the requested model
	if !h.modelSelector.ValidateModel(req.Model) {
		h.logger.WithField("model", req.Model).Warn("Unsupported model requested")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.NewValidationError("Unsupported model: " + req.Model),
		})
		return
	}

	// Log the request with cache control info
	h.logger.WithFields(logrus.Fields{
		"model":       req.Model,
		"max_tokens":  req.MaxTokens,
		"stream":      req.Stream,
		"messages":    len(req.Messages),
		"has_system":  req.System != nil,
		"has_tools":   len(req.Tools) > 0,
		"app_name":    h.config.AppName,
		"app_version": h.config.AppVersion,
		"referrer":    c.GetString("referrer"),
	}).Info("Processing message request")

	// Validate token limits
	tokenReq := &models.TokenCountRequest{
		Model:    req.Model,
		Messages: req.Messages,
		System:   req.System,
		Tools:    req.Tools,
	}

	if err := h.tokenService.ValidateTokenLimits(tokenReq); err != nil {
		h.logger.WithError(err).Warn("Token limit exceeded")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.NewValidationError(err.Error()),
		})
		return
	}

	// Convert to OpenAI format
	openAIReq, err := h.conversionService.ConvertAnthropicToOpenAI(&req, "gpt-4") // Simple fallback
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert request")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.NewInternalError("Failed to process request"),
		})
		return
	}

	// Log the selected model
	h.logger.WithFields(logrus.Fields{
		"original_model": req.Model,
		"selected_model": openAIReq.Model,
		"big_model":      h.config.BigModelName,
		"small_model":    h.config.SmallModelName,
	}).Info("Model selection completed")

	// Handle streaming vs non-streaming
	if req.Stream {
		h.handleStreamingRequest(c, openAIReq, req.Model)
	} else {
		h.handleNonStreamingRequest(c, openAIReq, req.Model)
	}
}

// handleStreamingRequest handles streaming message requests
func (h *Handler) handleStreamingRequest(c *gin.Context, openAIReq *models.OpenAIRequest, originalModel string) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// Make streaming request to OpenAI
	resp, err := h.openAIClient.CreateStreamingChatCompletion(ctx, openAIReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create streaming completion")
		h.streamingService.HandleStreamingError(c, err)
		return
	}

	// Stream the response
	if err := h.streamingService.StreamResponse(c, resp, originalModel); err != nil {
		h.logger.WithError(err).Error("Failed to stream response")
		h.streamingService.HandleStreamingError(c, err)
		return
	}
}

// handleNonStreamingRequest handles non-streaming message requests
func (h *Handler) handleNonStreamingRequest(c *gin.Context, openAIReq *models.OpenAIRequest, originalModel string) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// Make request to OpenAI
	openAIResp, err := h.openAIClient.CreateChatCompletion(ctx, openAIReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create completion")
		if apiErr, ok := err.(*models.APIError); ok {
			c.JSON(apiErr.HTTPStatus(), models.ErrorResponse{Error: apiErr})
		} else {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: models.NewInternalError("Failed to process request"),
			})
		}
		return
	}

	// Convert response to Anthropic format
	anthropicResp, err := h.conversionService.ConvertOpenAIToAnthropic(openAIResp, originalModel)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert response")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.NewInternalError("Failed to process response"),
		})
		return
	}

	// Log the response
	h.logger.WithFields(logrus.Fields{
		"response_id":   anthropicResp.ID,
		"model":         anthropicResp.Model,
		"stop_reason":   anthropicResp.StopReason,
		"input_tokens":  anthropicResp.Usage.InputTokens,
		"output_tokens": anthropicResp.Usage.OutputTokens,
	}).Info("Sending response")

	c.JSON(http.StatusOK, anthropicResp)
}

// CountTokens handles token counting requests
func (h *Handler) CountTokens(c *gin.Context) {
	var req models.TokenCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid token count request format")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.FormatValidationError(err),
		})
		return
	}

	// Count tokens
	resp, err := h.tokenService.CountTokens(&req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to count tokens")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.NewInternalError("Failed to count tokens"),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"model":        req.Model,
		"messages":     len(req.Messages),
		"input_tokens": resp.InputTokens,
	}).Info("Token count completed")

	c.JSON(http.StatusOK, resp)
}

// GetModels handles model listing requests
func (h *Handler) GetModels(c *gin.Context) {
	// Return both OpenAI models and our configured models
	availableModels := h.modelSelector.GetAvailableModels()

	// Also try to get OpenAI models if possible
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	openAIModels, err := h.openAIClient.GetModels(ctx)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get OpenAI models, returning configured models only")
	}

	response := gin.H{
		"configured_models": availableModels,
		"app_name":          h.config.AppName,
		"app_version":       h.config.AppVersion,
	}

	if openAIModels != nil {
		response["openai_models"] = openAIModels
	}

	c.JSON(http.StatusOK, response)
}

// ValidateAPIKey handles API key validation
func (h *Handler) ValidateAPIKey(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.openAIClient.ValidateAPIKey(ctx); err != nil {
		h.logger.WithError(err).Warn("API key validation failed")
		if apiErr, ok := err.(*models.APIError); ok {
			c.JSON(apiErr.HTTPStatus(), models.ErrorResponse{Error: apiErr})
		} else {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: models.NewAuthenticationError("Invalid API key"),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
	})
}

// GetStatus provides detailed service status
func (h *Handler) GetStatus(c *gin.Context) {
	status := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"app_name":  h.config.AppName,
		"version":   h.config.AppVersion,
		"referrer":  h.config.ReferrerURL,
		"config": gin.H{
			"base_url":    h.config.OpenAIBaseURL,
			"big_model":   h.config.BigModelName,
			"small_model": h.config.SmallModelName,
		},
		"models": h.modelSelector.GetAvailableModels(),
	}

	// Check OpenAI API connectivity
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.openAIClient.ValidateAPIKey(ctx); err != nil {
		status["openai_status"] = "error"
		status["openai_error"] = err.Error()
	} else {
		status["openai_status"] = "connected"
	}

	c.JSON(http.StatusOK, status)
}
