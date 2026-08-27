package response

import (
	"encoding/json"
	"net/http"
)

// RespondWithJSON writes a JSON payload with a given HTTP status code.
func RespondWithJSON(writer http.ResponseWriter, statusCode int, payload interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

// RespondWithError writes a structured JSON error response.
func RespondWithError(writer http.ResponseWriter, statusCode int, code string, message string) {
	RespondWithJSON(writer, statusCode, map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
