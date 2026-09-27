package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// RespondJSON writes a JSON response and logs encoding failures instead of returning them.
func RespondJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Warn("Failed to encode JSON response", "error", err)
	}
}

// RespondError writes the structured error shape with a code derived from the
// status, for errors that need no field, details or suggestion.
func RespondError(w http.ResponseWriter, statusCode int, message string) {
	RespondStructuredError(w, statusCode, ErrorDetail{Code: codeForStatus(statusCode), Message: message})
}

// RespondOK is a convenience wrapper for 200 JSON responses.
func RespondOK(w http.ResponseWriter, data any) {
	RespondJSON(w, http.StatusOK, data)
}

// RespondCreated is a convenience wrapper for 201 JSON responses.
func RespondCreated(w http.ResponseWriter, data any) {
	RespondJSON(w, http.StatusCreated, data)
}

// RespondNotFound emits a standard "<resource> not found" payload for simple handlers.
func RespondNotFound(w http.ResponseWriter, resource string) {
	RespondError(w, http.StatusNotFound, resource+" not found")
}
