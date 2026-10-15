package timeline

import (
	"errors"
	"net/http"
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
