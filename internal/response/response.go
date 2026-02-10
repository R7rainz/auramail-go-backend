package response

import (
	"encoding/json"
	"net/http"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	ErrCodeUnauthorized   ErrorCode = "unauthorized"
	ErrCodeForbidden      ErrorCode = "forbidden"
	ErrCodeNotFound       ErrorCode = "not_found"
	ErrCodeBadRequest     ErrorCode = "bad_request"
	ErrCodeValidation     ErrorCode = "validation_error"
	ErrCodeInternal       ErrorCode = "internal_error"
	ErrCodeRateLimited    ErrorCode = "rate_limited"
	ErrCodeServiceUnavail ErrorCode = "service_unavailable"
	ErrCodeConflict       ErrorCode = "conflict"
	ErrCodeGatewayTimeout ErrorCode = "gateway_timeout"
)

// APIResponse is the standard response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// APIError represents a structured error
type APIError struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Meta contains optional metadata for responses
type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// JSON writes a JSON response with the given status code
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Success writes a successful JSON response
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// Created writes a 201 Created response
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
	})
}

// NoContent writes a 204 No Content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error writes an error response
func Error(w http.ResponseWriter, status int, code ErrorCode, message string, details interface{}) {
	JSON(w, status, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// BadRequest writes a 400 Bad Request error
func BadRequest(w http.ResponseWriter, message string, details interface{}) {
	Error(w, http.StatusBadRequest, ErrCodeBadRequest, message, details)
}

// Unauthorized writes a 401 Unauthorized error
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Authentication required"
	}
	Error(w, http.StatusUnauthorized, ErrCodeUnauthorized, message, nil)
}

// Forbidden writes a 403 Forbidden error
func Forbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Access denied"
	}
	Error(w, http.StatusForbidden, ErrCodeForbidden, message, nil)
}

// NotFound writes a 404 Not Found error
func NotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Resource not found"
	}
	Error(w, http.StatusNotFound, ErrCodeNotFound, message, nil)
}

// Conflict writes a 409 Conflict error
func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, ErrCodeConflict, message, nil)
}

// ValidationError writes a 422 Unprocessable Entity error
func ValidationError(w http.ResponseWriter, details interface{}) {
	Error(w, http.StatusUnprocessableEntity, ErrCodeValidation, "Validation failed", details)
}

// InternalError writes a 500 Internal Server Error
func InternalError(w http.ResponseWriter, message string) {
	if message == "" {
		message = "An internal error occurred"
	}
	Error(w, http.StatusInternalServerError, ErrCodeInternal, message, nil)
}

// ServiceUnavailable writes a 503 Service Unavailable error
func ServiceUnavailable(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	Error(w, http.StatusServiceUnavailable, ErrCodeServiceUnavail, message, nil)
}

// RateLimited writes a 429 Too Many Requests error
func RateLimited(w http.ResponseWriter, retryAfter string) {
	w.Header().Set("Retry-After", retryAfter)
	Error(w, http.StatusTooManyRequests, ErrCodeRateLimited, "Rate limit exceeded", nil)
}

// GatewayTimeout writes a 504 Gateway Timeout error
func GatewayTimeout(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Request timed out"
	}
	Error(w, http.StatusGatewayTimeout, ErrCodeGatewayTimeout, message, nil)
}

// SuccessWithMeta writes a successful response with pagination metadata
func SuccessWithMeta(w http.ResponseWriter, data interface{}, meta *Meta) {
	JSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}
