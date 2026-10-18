package timeline

import (
	"net/http"

	"github.com/jonathan-skantz/calendar/backend/db"
)

var (
	householdNotFound = mappedErrorResponse{
		cause:    db.ErrHouseholdNotFound,
		response: httpErrorResponse{status: http.StatusNotFound, message: "you are not in a household"},
	}
	alreadyInHousehold = mappedErrorResponse{
		cause:    db.ErrAlreadyInHousehold,
		response: httpErrorResponse{status: http.StatusConflict, message: db.ErrAlreadyInHousehold.Error()},
	}
)

// GetHousehold responds with null when the user has no household.
func (h *Handler) GetHousehold(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	household, err := h.Store.GetHousehold(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to load household", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, household)
}

func (h *Handler) decodeHouseholdRequest(w http.ResponseWriter, r *http.Request) (householdRequest, bool) {
	var request householdRequest
	if !decodeJSON(w, r, &request) {
		return request, false
	}
	if err := validateHouseholdRequest(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return request, false
	}
	return request, true
}

func (h *Handler) CreateHousehold(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	request, ok := h.decodeHouseholdRequest(w, r)
	if !ok {
		return
	}
	householdID, err := h.Store.CreateHousehold(r.Context(), userID, request.Name, request.TransitCardCount)
	if err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to create household"},
			alreadyInHousehold,
		)
		return
	}
	writeCreated(w, householdID)
}

func (h *Handler) UpdateHousehold(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	request, ok := h.decodeHouseholdRequest(w, r)
	if !ok {
		return
	}
	if err := h.Store.UpdateHousehold(r.Context(), userID, request.Name, request.TransitCardCount); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to update household"},
			householdNotFound,
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddHouseholdMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var request householdMemberRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	member, ok := h.findUserByEmail(w, r, request.Email, "user not found")
	if !ok {
		return
	}
	if err := h.Store.AddHouseholdMember(r.Context(), userID, member.ID); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to add household member"},
			householdNotFound,
			alreadyInHousehold,
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) LeaveHousehold(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if err := h.Store.LeaveHousehold(r.Context(), userID); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to leave household"},
			householdNotFound,
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
