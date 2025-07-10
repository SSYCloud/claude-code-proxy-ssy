package middleware

import (
	"net/http"
	"strings"
	"time"

	"claude-code-provider-proxy/internal/config"
	"claude-code-provider-proxy/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware handles API key authentication
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for health check
		if c.Request.URL.Path == "/" || c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Get API key from header
		apiKey := c.GetHeader("x-api-key")
		if apiKey == "" {
			apiKey = c.GetHeader("Authorization")
			if strings.HasPrefix(apiKey, "Bearer ") {
				apiKey = strings.TrimPrefix(apiKey, "Bearer ")
			}
		}

		// For now, we just check if an API key is provided
		// In a production environment, you would validate against a database or service
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: models.NewAuthenticationError("API key is required"),
			})
			c.Abort()
			return
		}

		// Store API key in context for later use
		c.Set("api_key", apiKey)
		c.Next()
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set CORS headers
		c.Header("Access-Control-Allow-Origin", "*") // In production, be more specific
		c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware provides structured logging
func LoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.WithFields(logrus.Fields{
			"status_code":  param.StatusCode,
			"latency":      param.Latency,
			"client_ip":    param.ClientIP,
			"method":       param.Method,
			"path":         param.Path,
			"user_agent":   param.Request.UserAgent(),
			"error":        param.ErrorMessage,
		}).Info("HTTP Request")
		return ""
	})
}

// ErrorHandlingMiddleware handles panics and errors
func ErrorHandlingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.WithFields(logrus.Fields{
			"panic": recovered,
			"path":  c.Request.URL.Path,
			"method": c.Request.Method,
		}).Error("Panic recovered")

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.NewInternalError("Internal server error"),
		})
	})
}

// ReferrerMiddleware checks and sets referrer information
func ReferrerMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set application headers
		c.Header("X-App-Name", cfg.AppName)
		c.Header("X-App-Version", cfg.AppVersion)
		
		// Check referrer if configured
		if cfg.ReferrerURL != "" {
			referrer := c.GetHeader("Referer")
			if referrer == "" {
				referrer = c.GetHeader("Referrer")
			}
			
			// Log referrer information
			if referrer != "" {
				// In production, you might want to validate the referrer
				// For now, we just log it
				c.Set("referrer", referrer)
			}
			
			// Set the expected referrer in response header for client reference
			c.Header("X-Expected-Referrer", cfg.ReferrerURL)
		}
		
		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// ContentTypeMiddleware ensures proper content type for API endpoints
func ContentTypeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for GET requests and health checks
		if c.Request.Method == "GET" || c.Request.URL.Path == "/" || c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		contentType := c.GetHeader("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error: models.NewValidationError("Content-Type must be application/json"),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// AnthropicVersionMiddleware handles Anthropic API version header
func AnthropicVersionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set default Anthropic version if not provided
		if c.GetHeader("anthropic-version") == "" {
			c.Header("anthropic-version", "2023-06-01")
		}
		c.Next()
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
