package timeline

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jonathan-skantz/calendar/backend/db"
)

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	var request taskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
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
	taskID, err := uuid.Parse(r.PathValue("taskID"))
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}
	var request taskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
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
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update task", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetTaskCompleted(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	taskID, err := uuid.Parse(r.PathValue("taskID"))
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}
	var request taskCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.SetTaskCompleted(r.Context(), userID, taskID, request.IsCompleted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update task", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
