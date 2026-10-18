package timeline

import (
	"net/http"

	"github.com/jonathan-skantz/calendar/backend/db"
)

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	var request taskRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := validateTaskRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.Store.CreateTask(r.Context(), userID, db.Task{
		Title:       request.Title,
		Description: request.Description,
		URL:         request.URL,
		DeadlineAt:  request.DeadlineAt,
	})
	if err != nil {
		http.Error(w, "failed to create task", http.StatusInternalServerError)
		return
	}
	writeCreated(w, id)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "taskID")
	if !ok {
		return
	}
	var request taskRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := validateTaskRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateTask(r.Context(), userID, taskID, db.Task{
		Title: request.Title, Description: request.Description, URL: request.URL,
		DeadlineAt: request.DeadlineAt,
	}); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to update task"},
			mappedErrorResponse{
				cause:    db.ErrTaskNotFound,
				response: httpErrorResponse{status: http.StatusNotFound, message: "task not found"},
			},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetTaskCompleted(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "taskID")
	if !ok {
		return
	}
	var request taskCompletionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := h.Store.SetTaskCompleted(r.Context(), userID, taskID, request.IsCompleted); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to update task"},
			mappedErrorResponse{
				cause:    db.ErrTaskNotFound,
				response: httpErrorResponse{status: http.StatusNotFound, message: "task not found"},
			},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
