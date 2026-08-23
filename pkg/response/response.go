package response

import (
	"encoding/json"
	"net/http"
)

// RespondWithJSON writes a JSON payload with a given HTTP status code.
func RespondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

// RespondWithError writes a structured JSON error response.
func RespondWithError(w http.ResponseWriter, statusCode int, code string, message string) {
	RespondWithJSON(w, statusCode, map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
