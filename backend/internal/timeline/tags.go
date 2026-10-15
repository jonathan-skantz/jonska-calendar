package timeline

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
)

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var request tagRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateTagName(request.Name); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	tag, err := h.Store.CreateTag(r.Context(), userID, strings.TrimSpace(request.Name))
	if err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to create tag"},
			mappedErrorResponse{
				cause:    db.ErrTagAlreadyExists,
				response: httpErrorResponse{status: http.StatusConflict, message: "tag already exists"},
			},
		)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tag)
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	tags, err := h.Store.ListTags(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to list tags", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tags)
}

func (h *Handler) SetDefaultTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var request tagSelectionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateTagSelectionRequest(request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.SetDefaultTag(r.Context(), userID, request.TagID); err != nil {
		http.Error(w, "tag not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RenameTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	tagID, err := uuid.Parse(r.PathValue("tagID"))
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}
	var request tagUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateTagUpdateRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.RenameTag(r.Context(), userID, tagID, strings.TrimSpace(request.Name), request.Color); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusNotFound, message: "tag not found"},
			mappedErrorResponse{
				cause:    db.ErrTagAlreadyExists,
				response: httpErrorResponse{status: http.StatusConflict, message: "tag already exists"},
			},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	tagID, err := uuid.Parse(r.PathValue("tagID"))
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}
	if err := h.Store.DeleteTag(r.Context(), userID, tagID); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusNotFound, message: "tag not found"},
			mappedErrorResponse{
				cause:    db.ErrLastTag,
				response: httpErrorResponse{status: http.StatusConflict, message: "at least one tag is required"},
			},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddTagToItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemID"))
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}
	var request tagSelectionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateTagSelectionRequest(request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.AddTagToItem(r.Context(), userID, itemID, request.TagID); err != nil {
		http.Error(w, "item or tag not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveTagFromItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemID"))
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}
	tagID, err := uuid.Parse(r.PathValue("tagID"))
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}
	if err := h.Store.RemoveTagFromItem(r.Context(), userID, itemID, tagID); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusNotFound, message: "item tag not found"},
			mappedErrorResponse{
				cause:    db.ErrLastTag,
				response: httpErrorResponse{status: http.StatusConflict, message: "events must have at least one tag"},
			},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
