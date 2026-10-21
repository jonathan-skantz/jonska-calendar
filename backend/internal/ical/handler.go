package ical

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
	"github.com/jonathan-skantz/calendar/backend/internal/timeline"
)

// maxImportBodyBytes bounds an import request. The file travels as a JSON
// string, which escapes its line breaks, so this leaves room for a file of
// about 5 MB.
const maxImportBodyBytes = 6 << 20

// ReasonDuplicate is why an event whose UID the user already has is not
// imported.
const ReasonDuplicate = "already imported"

var errInvalidTimeZone = errors.New("time_zone must be an IANA time zone")

type Handler struct {
	Store       *db.Store
	CurrentUser auth.CurrentUser
	now         func() time.Time
}

func New(store *db.Store, currentUser auth.CurrentUser) *Handler {
	return &Handler{Store: store, CurrentUser: currentUser, now: time.Now}
}

type importRequest struct {
	ICS      string `json:"ics"`
	TimeZone string `json:"time_zone"`
	// TagID is the main tag of every imported event; omitted means the
	// default tag.
	TagID  uuid.UUID `json:"tag_id"`
	DryRun bool      `json:"dry_run"`
}

type importResponse struct {
	Imported int       `json:"imported"`
	Skipped  []Skipped `json:"skipped"`
	Warnings []string  `json:"warnings"`
}

// Import creates the events in an iCalendar file. With dry_run it creates
// nothing and answers what an import would do, so the user can review it.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	var request importRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxImportBodyBytes)).Decode(&request); err != nil {
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			http.Error(w, "the file is too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, timeline.ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	loc, err := parseTimeZone(request.TimeZone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	parsed, err := Parse(strings.NewReader(request.ICS), loc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tags, err := h.Store.ListTags(r.Context(), userID)
	if err != nil {
		writeServerError(w, "failed to load tags", err)
		return
	}
	tagIDs := make(map[string]uuid.UUID, len(tags))
	for _, tag := range tags {
		tagIDs[strings.ToLower(tag.Name)] = tag.ID
	}

	response := importResponse{Skipped: parsed.Skipped, Warnings: parsed.Warnings}
	var imports []db.ImportedEvent
	for _, event := range parsed.Events {
		validated, err := timeline.ValidateEvent(event.Event)
		var exceptions []db.OccurrenceException
		if err == nil && event.Recurrence != nil {
			// Exceptions that are not occurrences of the rule are dropped.
			validated, exceptions, err = timeline.WithSeries(validated, event.Recurrence, event.TimeZone, event.Exceptions)
		}
		if err != nil {
			response.Skipped = append(response.Skipped, Skipped{Title: event.Title, Reason: err.Error()})
			continue
		}
		imported := db.ImportedEvent{
			Event: validated, ICalUID: event.UID, ReminderOffsets: event.ReminderOffsets, Exceptions: exceptions,
		}
		// Tags are matched by name and never created.
		for _, name := range event.Categories {
			if tagID, ok := tagIDs[strings.ToLower(name)]; ok {
				imported.TagIDs = append(imported.TagIDs, tagID)
			}
		}
		imports = append(imports, imported)
	}

	created, err := h.Store.ImportEvents(r.Context(), userID, request.TagID, imports, request.DryRun)
	if errors.Is(err, db.ErrTagNotFound) {
		http.Error(w, "tag not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeServerError(w, "failed to import events", err)
		return
	}
	for index, ok := range created {
		if ok {
			response.Imported++
		} else {
			response.Skipped = append(response.Skipped, Skipped{Title: imports[index].Title, Reason: ReasonDuplicate})
		}
	}
	// Empty lists, not null, so clients can read them without checks.
	if response.Skipped == nil {
		response.Skipped = []Skipped{}
	}
	if response.Warnings == nil {
		response.Warnings = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// Export answers the user's own events as an iCalendar file download.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	loc, err := parseTimeZone(r.URL.Query().Get("time_zone"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	events, err := h.Store.ListEventsForExport(r.Context(), userID)
	if err != nil {
		writeServerError(w, "failed to export events", err)
		return
	}
	// Encoded in full first, so a failure can still answer 500.
	var file bytes.Buffer
	if err := Encode(&file, events, loc, h.now()); err != nil {
		writeServerError(w, "failed to export events", err)
		return
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="calendar.ics"`)
	_, _ = file.WriteTo(w)
}

// parseTimeZone reads the browser's IANA time zone. "Local" and "" are
// valid for time.LoadLocation, but mean the server's zone and UTC.
func parseTimeZone(name string) (*time.Location, error) {
	if name == "" || name == "Local" {
		return nil, errInvalidTimeZone
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, errInvalidTimeZone
	}
	return loc, nil
}

func writeServerError(w http.ResponseWriter, message string, err error) {
	log.Printf("%s: %v", message, err)
	http.Error(w, message, http.StatusInternalServerError)
}
