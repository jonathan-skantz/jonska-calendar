package timeline

import (
	"encoding/json"
	"errors"
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
	writeHTTPError(w, fallback)
}

func writeHTTPError(w http.ResponseWriter, response httpErrorResponse) {
	http.Error(w, response.message, response.status)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeCreated(w http.ResponseWriter, id uuid.UUID) {
	writeJSON(w, http.StatusCreated, createResponse{ID: id})
}
