package timeline

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteMappedError(t *testing.T) {
	cause := errors.New("missing item")
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "mapped wrapped error",
			err:        fmt.Errorf("store operation: %w", cause),
			wantStatus: http.StatusNotFound,
			wantBody:   "item not found\n",
		},
		{
			name:       "fallback response",
			err:        errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "failed to list items\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			writeMappedError(response, test.err,
				httpErrorResponse{status: http.StatusInternalServerError, message: "failed to list items"},
				mappedErrorResponse{
					cause:    cause,
					response: httpErrorResponse{status: http.StatusNotFound, message: "item not found"},
				},
			)

			if response.Code != test.wantStatus {
				t.Fatalf("expected status %d, got %d", test.wantStatus, response.Code)
			}
			if response.Body.String() != test.wantBody {
				t.Fatalf("expected body %q, got %q", test.wantBody, response.Body.String())
			}
		})
	}
}
