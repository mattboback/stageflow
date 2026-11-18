// Package httputil provides standardized HTTP response helpers.
package httputil

import (
	"net/http"
	"strconv"
	"strings"
)

// ErrorCode represents a machine-readable error identifier.
type ErrorCode string

const (
	// Client errors (4xx).
	ErrCodeInvalidRequest    ErrorCode = "INVALID_REQUEST"
	ErrCodeValidationFailed  ErrorCode = "VALIDATION_FAILED"
	ErrCodeNotFound          ErrorCode = "NOT_FOUND"
	ErrCodeMethodNotAllowed  ErrorCode = "METHOD_NOT_ALLOWED"
	ErrCodeUnauthorized      ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden         ErrorCode = "FORBIDDEN"
	ErrCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodePayloadTooLarge   ErrorCode = "PAYLOAD_TOO_LARGE"
	ErrCodeUnsupportedModule ErrorCode = "UNSUPPORTED_MODULE"
	ErrCodeInvalidFormat     ErrorCode = "INVALID_FORMAT"
	ErrCodeMissingField      ErrorCode = "MISSING_FIELD"
	ErrCodeInvalidValue      ErrorCode = "INVALID_VALUE"
	ErrCodeJobNotFound       ErrorCode = "JOB_NOT_FOUND"

	// Server errors (5xx).
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeRequestTimeout     ErrorCode = "REQUEST_TIMEOUT"
	ErrCodeDatabaseError      ErrorCode = "DATABASE_ERROR"
	ErrCodeStorageError       ErrorCode = "STORAGE_ERROR"
	ErrCodeMessagingError     ErrorCode = "MESSAGING_ERROR"
)

// ErrorResponse represents a structured API error response.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains detailed error information.
type ErrorDetail struct {
	Code       ErrorCode         `json:"code"`                  // Machine-readable error code
	Message    string            `json:"message"`               // Human-readable error message
	Details    string            `json:"details,omitempty"`     // Additional context or explanation
	Suggestion string            `json:"suggestion,omitempty"`  // User-actionable guidance
	Field      string            `json:"field,omitempty"`       // Field name for validation errors
	RetryAfter int               `json:"retry_after,omitempty"` // Seconds to wait before retry (for rate limits)
	Meta       map[string]string `json:"meta,omitempty"`        // Additional metadata
}

// RespondStructuredError writes ErrorResponse and sets Retry-After when provided.
func RespondStructuredError(w http.ResponseWriter, statusCode int, detail ErrorDetail) {
	// Set retry-after header if present (value is in seconds)
	if detail.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(detail.RetryAfter))
	}

	RespondJSON(w, statusCode, ErrorResponse{Error: detail})
}

// NewValidationError standardizes field-level validation errors for API clients.
func NewValidationError(field, message, suggestion string) ErrorDetail {
	return ErrorDetail{
		Code:       ErrCodeValidationFailed,
		Message:    message,
		Field:      field,
		Suggestion: suggestion,
	}
}

// NewJobNotFoundError includes job ID context for client UX.
func NewJobNotFoundError(jobID string) ErrorDetail {
	return ErrorDetail{
		Code:       ErrCodeJobNotFound,
		Message:    "Job not found",
		Details:    "Job ID: " + jobID,
		Suggestion: "The job may have expired, been deleted, or the ID may be incorrect. Check your job history or submit a new scan.",
	}
}

// NewRateLimitError includes Retry-After metadata for client backoff.
func NewRateLimitError(retryAfterSeconds int, limit, window string) ErrorDetail {
	return ErrorDetail{
		Code:       ErrCodeRateLimitExceeded,
		Message:    "Rate limit exceeded",
		Details:    "You have exceeded the allowed number of requests.",
		Suggestion: "Wait for the Retry-After interval, then try again.",
		RetryAfter: retryAfterSeconds,
		Meta: map[string]string{
			"limit":  limit,
			"window": window,
		},
	}
}

// NewUnsupportedModuleError lists supported modules for self-serve correction.
func NewUnsupportedModuleError(module string, supported []string) ErrorDetail {
	supportedStr := strings.Join(supported, ", ")

	return ErrorDetail{
		Code:       ErrCodeUnsupportedModule,
		Message:    "Unsupported scanner module: " + module,
		Details:    "Supported modules: " + supportedStr,
		Suggestion: "Use one of the supported scanner modules. You can list available scanners at GET /api/v1/scanners",
		Field:      "modules",
	}
}

// NewDatabaseError returns a generic database failure payload.
func NewDatabaseError() ErrorDetail {
	return ErrorDetail{
		Code:       ErrCodeDatabaseError,
		Message:    "A database error occurred",
		Suggestion: "Please try again. If the problem persists, contact support.",
	}
}

// codeForStatus gives RespondError's plain messages a machine-readable code.
func codeForStatus(statusCode int) ErrorCode {
	switch statusCode {
	case http.StatusBadRequest:
		return ErrCodeInvalidRequest
	case http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case http.StatusForbidden:
		return ErrCodeForbidden
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusMethodNotAllowed:
		return ErrCodeMethodNotAllowed
	case http.StatusRequestEntityTooLarge:
		return ErrCodePayloadTooLarge
	case http.StatusTooManyRequests:
		return ErrCodeRateLimitExceeded
	case http.StatusServiceUnavailable:
		return ErrCodeServiceUnavailable
	}

	if statusCode >= http.StatusInternalServerError {
		return ErrCodeInternalError
	}

	return ErrCodeInvalidRequest
}
