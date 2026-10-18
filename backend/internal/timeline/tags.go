package timeline

import (
	"net/http"

	"github.com/jonathan-skantz/calendar/backend/db"
)

var tagNotFound = mappedErrorResponse{
	cause:    db.ErrTagNotFound,
	response: httpErrorResponse{status: http.StatusNotFound, message: "tag not found"},
}

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var request tagRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	name, err := validateTagName(request.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tag, err := h.Store.CreateTag(r.Context(), userID, name)
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
	writeJSON(w, http.StatusCreated, tag)
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	tags, err := h.Store.ListTags(r.Context(), userID)
	if err != nil {
		writeServerError(w, "failed to list tags", err)
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

func (h *Handler) SetDefaultTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var request tagSelectionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := validateTagSelectionRequest(request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.SetDefaultTag(r.Context(), userID, request.TagID); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to set default tag"},
			tagNotFound,
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RenameTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	tagID, ok := pathID(w, r, "tagID")
	if !ok {
		return
	}
	var request tagUpdateRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request, err := validateTagUpdateRequest(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.RenameTag(r.Context(), userID, tagID, request.Name, request.Color); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to update tag"},
			tagNotFound,
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
	tagID, ok := pathID(w, r, "tagID")
	if !ok {
		return
	}
	if err := h.Store.DeleteTag(r.Context(), userID, tagID); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to delete tag"},
			tagNotFound,
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
	itemID, ok := pathID(w, r, "itemID")
	if !ok {
		return
	}
	var request tagSelectionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := validateTagSelectionRequest(request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.AddTagToItem(r.Context(), userID, itemID, request.TagID); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to add tag"},
			mappedErrorResponse{
				cause:    db.ErrTagNotFound,
				response: httpErrorResponse{status: http.StatusNotFound, message: "item or tag not found"},
			},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveTagFromItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	itemID, ok := pathID(w, r, "itemID")
	if !ok {
		return
	}
	tagID, ok := pathID(w, r, "tagID")
	if !ok {
		return
	}
	if err := h.Store.RemoveTagFromItem(r.Context(), userID, itemID, tagID); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to remove tag"},
			mappedErrorResponse{
				cause:    db.ErrTagNotFound,
				response: httpErrorResponse{status: http.StatusNotFound, message: "item tag not found"},
			},
			mappedErrorResponse{
				cause:    db.ErrLastTag,
				response: httpErrorResponse{status: http.StatusConflict, message: "events must have at least one tag"},
			},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
