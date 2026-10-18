package timeline

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type httpErrorResponse struct {
	status  int
	message string
}

type mappedErrorResponse struct {
	cause    error
	response httpErrorResponse
}

func writeMappedError(w http.ResponseWriter, err error, fallback httpErrorResponse, mappings ...mappedErrorResponse) {
	for _, mapping := range mappings {
		if errors.Is(err, mapping.cause) {
			writeHTTPError(w, mapping.response)
			return
		}
	}
	if fallback.status >= http.StatusInternalServerError {
		log.Printf("%s: %v", fallback.message, err)
	}
	writeHTTPError(w, fallback)
}

// writeServerError answers 500 with a generic message and logs the cause,
// which the client never sees.
func writeServerError(w http.ResponseWriter, message string, err error) {
	log.Printf("%s: %v", message, err)
	http.Error(w, message, http.StatusInternalServerError)
}

func writeHTTPError(w http.ResponseWriter, response httpErrorResponse) {
	http.Error(w, response.message, response.status)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The status is already sent, so a failed write can only be logged.
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

func writeCreated(w http.ResponseWriter, id uuid.UUID) {
	writeJSON(w, http.StatusCreated, createResponse{ID: id})
}
