package httputil

import (
	"encoding/json"
	"log"
	"net/http"
)

// RespondJSON writes a JSON response and logs encoding failures instead of returning them.
func RespondJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

// RespondError is a minimal {"error": "..."} helper for endpoints that don't use structured errors.
func RespondError(w http.ResponseWriter, statusCode int, message string) {
	RespondJSON(w, statusCode, map[string]string{
		"error": message,
	})
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
