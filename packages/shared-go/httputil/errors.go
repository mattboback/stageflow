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

// NewNotFoundError returns a consistent not-found payload for resources.
func NewNotFoundError(resource string) ErrorDetail {
	return ErrorDetail{
		Code:       ErrCodeNotFound,
		Message:    resource + " not found",
		Suggestion: "Please verify the ID is correct or check if the resource still exists.",
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
		Suggestion: "Please wait before submitting another request. Consider upgrading your plan for higher limits.",
		RetryAfter: retryAfterSeconds,
		Meta: map[string]string{
			"limit":  limit,
			"window": window,
		},
	}
}

// NewPayloadTooLargeError reports max size in metadata for debugging.
func NewPayloadTooLargeError(maxSize string) ErrorDetail {
	return ErrorDetail{
		Code:       ErrCodePayloadTooLarge,
		Message:    "Request payload is too large",
		Details:    "Maximum allowed size: " + maxSize,
		Suggestion: "Reduce the size of your upload or split it into smaller files.",
		Meta: map[string]string{
			"max_size": maxSize,
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

// NewInvalidFormatError reports expected formats for client correction.
func NewInvalidFormatError(field, expected string) ErrorDetail {
	return ErrorDetail{
		Code:       ErrCodeInvalidFormat,
		Message:    "Invalid format for " + field,
		Details:    "Expected format: " + expected,
		Field:      field,
		Suggestion: "Please check the API documentation for the correct format.",
	}
}

// NewInternalError is the fallback for unclassified failures.
func NewInternalError(details string) ErrorDetail {
	return ErrorDetail{
		Code:       ErrCodeInternalError,
		Message:    "An internal error occurred while processing your request",
		Details:    details,
		Suggestion: "Please try again later. If the problem persists, contact support.",
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

// NewStorageError returns a generic storage failure payload.
func NewStorageError(operation string) ErrorDetail {
	return ErrorDetail{
		Code:       ErrCodeStorageError,
		Message:    "A storage error occurred during " + operation,
		Suggestion: "Please try again. If the problem persists, contact support.",
	}
}
