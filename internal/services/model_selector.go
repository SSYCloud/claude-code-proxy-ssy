package services

import (
	"strings"

	"claude-code-provider-proxy/internal/config"
	"claude-code-provider-proxy/internal/models"

	"github.com/sirupsen/logrus"
)

// ModelSelectorService handles model selection logic
type ModelSelectorService struct {
	config *config.Config
	logger *logrus.Logger
}

// NewModelSelectorService creates a new model selector service
func NewModelSelectorService(cfg *config.Config, logger *logrus.Logger) *ModelSelectorService {
	return &ModelSelectorService{
		config: cfg,
		logger: logger,
	}
}

// SelectModel selects the appropriate OpenAI model based on the Anthropic model request
func (s *ModelSelectorService) SelectModel(anthropicModel string, req *models.AnthropicRequest) string {
	// Log the model selection process
	s.logger.WithFields(logrus.Fields{
		"anthropic_model": anthropicModel,
		"big_model":       s.config.BigModelName,
		"small_model":     s.config.SmallModelName,
		"max_tokens":      req.MaxTokens,
		"has_tools":       len(req.Tools) > 0,
		"message_count":   len(req.Messages),
	}).Debug("Selecting model")

	// Follow Python project logic for model selection
	clientModelLower := strings.ToLower(anthropicModel)
	var targetModel string

	if strings.Contains(clientModelLower, "opus") || strings.Contains(clientModelLower, "sonnet") {
		targetModel = s.config.BigModelName
		s.logger.WithFields(logrus.Fields{
			"client_model": anthropicModel,
			"target_model": targetModel,
			"reason":       "opus/sonnet detected",
		}).Debug("Selected big model")
	} else if strings.Contains(clientModelLower, "haiku") {
		targetModel = s.config.SmallModelName
		s.logger.WithFields(logrus.Fields{
			"client_model": anthropicModel,
			"target_model": targetModel,
			"reason":       "haiku detected",
		}).Debug("Selected small model")
	} else {
		// Default to small model for unknown models
		targetModel = s.config.SmallModelName
		s.logger.WithFields(logrus.Fields{
			"client_model": anthropicModel,
			"target_model": targetModel,
			"reason":       "unknown model, defaulting to small",
		}).Warn("Unknown client model, defaulting to small model")
	}

	s.logger.WithFields(logrus.Fields{
		"client_model": anthropicModel,
		"target_model": targetModel,
	}).Info("Model selection completed")

	return targetModel
}

// GetModelInfo returns information about the selected model
func (s *ModelSelectorService) GetModelInfo(modelName string) map[string]interface{} {
	info := map[string]interface{}{
		"name": modelName,
	}

	if modelName == s.config.BigModelName {
		info["type"] = "big"
		info["description"] = "High-capability model for complex tasks"
	} else if modelName == s.config.SmallModelName {
		info["type"] = "small"
		info["description"] = "Efficient model for simple tasks"
	} else {
		info["type"] = "default"
		info["description"] = "Default model"
	}

	return info
}

// GetAvailableModels returns a list of available models
func (s *ModelSelectorService) GetAvailableModels() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":          s.config.BigModelName,
			"type":        "big",
			"description": "High-capability model for complex tasks",
		},
		{
			"id":          s.config.SmallModelName,
			"type":        "small",
			"description": "Efficient model for simple tasks",
		},
	}
}

// ValidateModel checks if the requested model is supported
func (s *ModelSelectorService) ValidateModel(modelName string) bool {
	// Use a more flexible validation strategy that matches the Python project
	// Accept any model that starts with "claude" and contains known model types
	modelNameLower := strings.ToLower(modelName)

	// Check if it's a Claude model
	if !strings.HasPrefix(modelNameLower, "claude") {
		return false
	}

	// Known Claude model patterns that we support
	supportedPatterns := []string{
		"opus",
		"sonnet",
		"haiku",
		"claude-2",
		"claude-3",
		"claude-instant",
	}

	// Check if the model name contains any supported pattern
	for _, pattern := range supportedPatterns {
		if strings.Contains(modelNameLower, pattern) {
			return true
		}
	}

	// If it starts with "claude" but doesn't match known patterns,
	// still accept it but log a warning (same as Python project behavior)
	s.logger.WithFields(logrus.Fields{
		"model_name": modelName,
		"reason":     "unknown_claude_model_accepting_anyway",
	}).Warn("Unknown Claude model variant, accepting anyway")

	return true
}
