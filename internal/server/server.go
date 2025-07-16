package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"claude-code-provider-proxy/internal/config"
	"claude-code-provider-proxy/internal/handlers"
	"claude-code-provider-proxy/internal/middleware"
	"claude-code-provider-proxy/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Server represents the HTTP server
type Server struct {
	config     *config.Config
	logger     *logrus.Logger
	httpServer *http.Server
	handler    *handlers.Handler
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	// Setup logger
	logger := logrus.New()
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Setup log file
	if err := setupLogFile(logger); err != nil {
		logger.WithError(err).Warn("Failed to setup log file, using stdout")
	}

	// Log application startup
	logger.WithFields(logrus.Fields{
		"app_name":     cfg.AppName,
		"app_version":  cfg.AppVersion,
		"referrer_url": cfg.ReferrerURL,
		"big_model":    cfg.BigModelName,
		"small_model":  cfg.SmallModelName,
	}).Info("Starting application")

	// Create services
	openAIClient := services.NewOpenAIClient(cfg, logger)
	modelSelector := services.NewModelSelectorService(cfg, logger)
	conversionService := services.NewConversionService(modelSelector, cfg, logger)
	tokenService := services.NewTokenCountingService()
	streamingService := services.NewStreamingService(conversionService, logger)

	// Create handler
	handler := handlers.NewHandler(
		cfg,
		logger,
		openAIClient,
		conversionService,
		tokenService,
		streamingService,
		modelSelector,
	)

	return &Server{
		config:  cfg,
		logger:  logger,
		handler: handler,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Set Gin mode based on log level
	if s.config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	router := s.setupRouter()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", s.config.Host, s.config.Port),
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		s.logger.WithFields(logrus.Fields{
			"host": s.config.Host,
			"port": s.config.Port,
		}).Info("Starting HTTP server")

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	s.waitForShutdown()

	return nil
}

// setupRouter configures the Gin router with all routes and middleware
func (s *Server) setupRouter() *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(middleware.ErrorHandlingMiddleware(s.logger))
	router.Use(middleware.LoggingMiddleware(s.logger))
	router.Use(middleware.CORSMiddleware(s.config))
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.ReferrerMiddleware(s.config))

	// Health check endpoint (no auth required)
	router.GET("/", s.handler.HealthCheck)
	router.GET("/health", s.handler.HealthCheck)
	router.GET("/status", s.handler.GetStatus)

	// API routes with authentication
	v1 := router.Group("/v1")
	v1.Use(middleware.AuthMiddleware(s.config))
	v1.Use(middleware.ContentTypeMiddleware())
	v1.Use(middleware.AnthropicVersionMiddleware())
	{
		// Anthropic-compatible endpoints
		v1.POST("/messages", s.handler.CreateMessage)
		v1.POST("/messages/count_tokens", s.handler.CountTokens)

		// Additional utility endpoints
		v1.GET("/models", s.handler.GetModels)
		v1.POST("/validate", s.handler.ValidateAPIKey)
	}

	// Add custom 404 handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"type":    "not_found_error",
				"message": "The requested endpoint was not found",
			},
		})
	})

	// Add custom 405 handler
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": gin.H{
				"type":    "method_not_allowed_error",
				"message": "The requested method is not allowed for this endpoint",
			},
		})
	})

	return router
}

// waitForShutdown waits for interrupt signal and gracefully shuts down the server
func (s *Server) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.WithError(err).Error("Server forced to shutdown")
	} else {
		s.logger.Info("Server shutdown complete")
	}
}

// Stop stops the server gracefully
func (s *Server) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// setupLogFile configures the logger to write to a file
func setupLogFile(logger *logrus.Logger) error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Create log directory
	logDir := filepath.Join(homeDir, ".claudeproxy", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Create log file
	logFile := filepath.Join(logDir, "service.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// Set output to both file and stdout
	logger.SetOutput(io.MultiWriter(os.Stdout, file))

	return nil
}
