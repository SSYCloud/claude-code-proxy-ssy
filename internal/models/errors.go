package models

import (
	"fmt"
	"net/http"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "validation_error"
	ErrorTypeAuthentication ErrorType = "authentication_error"
	ErrorTypePermission    ErrorType = "permission_error"
	ErrorTypeNotFound      ErrorType = "not_found_error"
	ErrorTypeRateLimit     ErrorType = "rate_limit_error"
	ErrorTypeAPI           ErrorType = "api_error"
	ErrorTypeInternal      ErrorType = "internal_error"
	ErrorTypeInvalidRequest ErrorType = "invalid_request_error"
)

// APIError represents a structured API error
type APIError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Code    string    `json:"code,omitempty"`
	Param   string    `json:"param,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// HTTPStatus returns the appropriate HTTP status code for the error type
func (e *APIError) HTTPStatus() int {
	switch e.Type {
	case ErrorTypeValidation, ErrorTypeInvalidRequest:
		return http.StatusBadRequest
	case ErrorTypeAuthentication:
		return http.StatusUnauthorized
	case ErrorTypePermission:
		return http.StatusForbidden
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case ErrorTypeAPI:
		return http.StatusBadGateway
	case ErrorTypeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// ErrorResponse represents the structure of error responses
type ErrorResponse struct {
	Error *APIError `json:"error"`
}

// NewValidationError creates a new validation error
func NewValidationError(message string, param ...string) *APIError {
	err := &APIError{
		Type:    ErrorTypeValidation,
		Message: message,
	}
	if len(param) > 0 {
		err.Param = param[0]
	}
	return err
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message string) *APIError {
	return &APIError{
		Type:    ErrorTypeAuthentication,
		Message: message,
	}
}

// NewPermissionError creates a new permission error
func NewPermissionError(message string) *APIError {
	return &APIError{
		Type:    ErrorTypePermission,
		Message: message,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string) *APIError {
	return &APIError{
		Type:    ErrorTypeNotFound,
		Message: message,
	}
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(message string) *APIError {
	return &APIError{
		Type:    ErrorTypeRateLimit,
		Message: message,
	}
}

// NewAPIError creates a new API error
func NewAPIError(message string, code ...string) *APIError {
	err := &APIError{
		Type:    ErrorTypeAPI,
		Message: message,
	}
	if len(code) > 0 {
		err.Code = code[0]
	}
	return err
}

// NewInternalError creates a new internal error
func NewInternalError(message string) *APIError {
	return &APIError{
		Type:    ErrorTypeInternal,
		Message: message,
	}
}

// NewInvalidRequestError creates a new invalid request error
func NewInvalidRequestError(message string, param ...string) *APIError {
	err := &APIError{
		Type:    ErrorTypeInvalidRequest,
		Message: message,
	}
	if len(param) > 0 {
		err.Param = param[0]
	}
	return err
}

// WrapError wraps a generic error into an APIError
func WrapError(err error, errorType ErrorType) *APIError {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr
	}
	return &APIError{
		Type:    errorType,
		Message: err.Error(),
	}
}

// FormatValidationError formats validation errors from gin binding
func FormatValidationError(err error) *APIError {
	return NewValidationError(fmt.Sprintf("Invalid request format: %s", err.Error()))
}
