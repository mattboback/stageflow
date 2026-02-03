package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorCodes(t *testing.T) {
	// Verify error codes are non-empty strings
	codes := []ErrorCode{
		ErrCodeInvalidRequest,
		ErrCodeValidationFailed,
		ErrCodeNotFound,
		ErrCodeUnauthorized,
		ErrCodeForbidden,
		ErrCodeRateLimitExceeded,
		ErrCodePayloadTooLarge,
		ErrCodeUnsupportedModule,
		ErrCodeInvalidFormat,
		ErrCodeMissingField,
		ErrCodeInvalidValue,
		ErrCodeJobNotFound,
		ErrCodeInternalError,
		ErrCodeServiceUnavailable,
		ErrCodeDatabaseError,
		ErrCodeStorageError,
		ErrCodeMessagingError,
	}

	for _, code := range codes {
		if string(code) == "" {
			t.Errorf("Error code should not be empty")
		}
	}
}

func TestRespondStructuredError(t *testing.T) {
	t.Run("basic error response", func(t *testing.T) {
		w := httptest.NewRecorder()
		detail := ErrorDetail{
			Code:    ErrCodeNotFound,
			Message: "Resource not found",
		}

		RespondStructuredError(w, http.StatusNotFound, detail)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}

		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("expected JSON content type, got %s", w.Header().Get("Content-Type"))
		}
	})

	t.Run("sets Retry-After header when provided", func(t *testing.T) {
		w := httptest.NewRecorder()
		detail := ErrorDetail{
			Code:       ErrCodeRateLimitExceeded,
			Message:    "Rate limit exceeded",
			RetryAfter: 60,
		}

		RespondStructuredError(w, http.StatusTooManyRequests, detail)

		if w.Header().Get("Retry-After") != "60" {
			t.Errorf("expected Retry-After header to be 60, got %s", w.Header().Get("Retry-After"))
		}
	})

	t.Run("no Retry-After header when not provided", func(t *testing.T) {
		w := httptest.NewRecorder()
		detail := ErrorDetail{
			Code:    ErrCodeNotFound,
			Message: "Not found",
		}

		RespondStructuredError(w, http.StatusNotFound, detail)

		if w.Header().Get("Retry-After") != "" {
			t.Errorf("expected no Retry-After header, got %s", w.Header().Get("Retry-After"))
		}
	})
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("email", "Invalid email format", "Use format: user@example.com")

	if err.Code != ErrCodeValidationFailed {
		t.Errorf("expected code %s, got %s", ErrCodeValidationFailed, err.Code)
	}

	if err.Field != "email" {
		t.Errorf("expected field 'email', got '%s'", err.Field)
	}

	if err.Message != "Invalid email format" {
		t.Errorf("expected message 'Invalid email format', got '%s'", err.Message)
	}

	if err.Suggestion != "Use format: user@example.com" {
		t.Errorf("expected suggestion, got '%s'", err.Suggestion)
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("Job")

	if err.Code != ErrCodeNotFound {
		t.Errorf("expected code %s, got %s", ErrCodeNotFound, err.Code)
	}

	if err.Message != "Job not found" {
		t.Errorf("expected message 'Job not found', got '%s'", err.Message)
	}

	if err.Suggestion == "" {
		t.Error("expected suggestion to be set")
	}
}

func TestNewJobNotFoundError(t *testing.T) {
	err := NewJobNotFoundError("job-123")

	if err.Code != ErrCodeJobNotFound {
		t.Errorf("expected code %s, got %s", ErrCodeJobNotFound, err.Code)
	}

	if err.Details != "Job ID: job-123" {
		t.Errorf("expected details to contain job ID, got '%s'", err.Details)
	}

	if err.Suggestion == "" {
		t.Error("expected suggestion to be set")
	}
}

func TestNewRateLimitError(t *testing.T) {
	err := NewRateLimitError(60, "100", "1 minute")

	if err.Code != ErrCodeRateLimitExceeded {
		t.Errorf("expected code %s, got %s", ErrCodeRateLimitExceeded, err.Code)
	}

	if err.RetryAfter != 60 {
		t.Errorf("expected RetryAfter 60, got %d", err.RetryAfter)
	}

	if err.Meta == nil {
		t.Fatal("expected Meta to be set")
	}

	if err.Meta["limit"] != "100" {
		t.Errorf("expected Meta['limit'] = '100', got '%s'", err.Meta["limit"])
	}

	if err.Meta["window"] != "1 minute" {
		t.Errorf("expected Meta['window'] = '1 minute', got '%s'", err.Meta["window"])
	}
}

func TestNewPayloadTooLargeError(t *testing.T) {
	err := NewPayloadTooLargeError("100MB")

	if err.Code != ErrCodePayloadTooLarge {
		t.Errorf("expected code %s, got %s", ErrCodePayloadTooLarge, err.Code)
	}

	if err.Details != "Maximum allowed size: 100MB" {
		t.Errorf("expected details to contain max size, got '%s'", err.Details)
	}

	if err.Meta == nil || err.Meta["max_size"] != "100MB" {
		t.Error("expected Meta to contain max_size")
	}
}

func TestNewUnsupportedModuleError(t *testing.T) {
	err := NewUnsupportedModuleError("unknown-scanner", []string{"axe", "lighthouse", "seo"})

	if err.Code != ErrCodeUnsupportedModule {
		t.Errorf("expected code %s, got %s", ErrCodeUnsupportedModule, err.Code)
	}

	if err.Message != "Unsupported scanner module: unknown-scanner" {
		t.Errorf("expected message to contain module name, got '%s'", err.Message)
	}

	if err.Details != "Supported modules: axe, lighthouse, seo" {
		t.Errorf("expected details to list supported modules, got '%s'", err.Details)
	}

	if err.Field != "modules" {
		t.Errorf("expected field 'modules', got '%s'", err.Field)
	}
}

func TestNewInvalidFormatError(t *testing.T) {
	err := NewInvalidFormatError("date", "YYYY-MM-DD")

	if err.Code != ErrCodeInvalidFormat {
		t.Errorf("expected code %s, got %s", ErrCodeInvalidFormat, err.Code)
	}

	if err.Message != "Invalid format for date" {
		t.Errorf("expected message, got '%s'", err.Message)
	}

	if err.Details != "Expected format: YYYY-MM-DD" {
		t.Errorf("expected details to contain format, got '%s'", err.Details)
	}

	if err.Field != "date" {
		t.Errorf("expected field 'date', got '%s'", err.Field)
	}
}

func TestNewInternalError(t *testing.T) {
	err := NewInternalError("Database connection failed")

	if err.Code != ErrCodeInternalError {
		t.Errorf("expected code %s, got %s", ErrCodeInternalError, err.Code)
	}

	if err.Details != "Database connection failed" {
		t.Errorf("expected details, got '%s'", err.Details)
	}

	if err.Suggestion == "" {
		t.Error("expected suggestion to be set")
	}
}

func TestNewDatabaseError(t *testing.T) {
	err := NewDatabaseError()

	if err.Code != ErrCodeDatabaseError {
		t.Errorf("expected code %s, got %s", ErrCodeDatabaseError, err.Code)
	}

	if err.Message == "" {
		t.Error("expected message to be set")
	}

	if err.Suggestion == "" {
		t.Error("expected suggestion to be set")
	}
}

func TestNewStorageError(t *testing.T) {
	err := NewStorageError("upload")

	if err.Code != ErrCodeStorageError {
		t.Errorf("expected code %s, got %s", ErrCodeStorageError, err.Code)
	}

	if err.Message != "A storage error occurred during upload" {
		t.Errorf("expected message to contain operation, got '%s'", err.Message)
	}

	if err.Suggestion == "" {
		t.Error("expected suggestion to be set")
	}
}
