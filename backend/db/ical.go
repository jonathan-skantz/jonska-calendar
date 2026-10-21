package db

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ExportUIDSuffix makes the iCalendar UID of an event created in the app:
// its ID followed by this suffix. Importing such a UID into the account that
// owns the event is a duplicate.
const ExportUIDSuffix = "@jonska-calendar"

// ImportedEvent is one event read from an iCalendar file, with the rows it
// needs besides the event itself.
type ImportedEvent struct {
	Event
	// ICalUID is empty when the file gave none; such events are never
	// recognized as duplicates.
	ICalUID         string
	TagIDs          []uuid.UUID
	ReminderOffsets []int
	// The occurrences of a repeating event that differ from it.
	Exceptions []OccurrenceException
}

// ExportEvent is one of the user's own events with what an iCalendar file
// carries besides the event fields. TagNames starts with the main tag.
type ExportEvent struct {
	Event
	UID             string
	TagNames        []string
	ReminderOffsets []int
	Exceptions      []OccurrenceException
}

var errDryRun = errors.New("dry run")

// ImportEvents creates the events in one transaction, all with mainTagID (or
// the default tag when it is uuid.Nil). It reports for each event whether it
// was created; events whose UID the user already has are not. With dryRun the
// transaction is rolled back, so the result shows what an import would do.
func (s *Store) ImportEvents(ctx context.Context, userID, mainTagID uuid.UUID, events []ImportedEvent, dryRun bool) ([]bool, error) {
	created := make([]bool, len(events))
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		if mainTagID != uuid.Nil {
			owned, err := tagOwnedBy(ctx, tx, mainTagID, userID)
			if err != nil {
				return err
			}
			if !owned {
				return ErrTagNotFound
			}
		}
		for index, imported := range events {
			itemID, ok, err := insertImportedItem(ctx, tx, userID, imported)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			event := imported.Event
			event.MainTagID = mainTagID
			if err := insertEventRows(ctx, tx, userID, itemID, event); err != nil {
				return err
			}
			// Extra tags must be the user's own; others are left out.
			if _, err := tx.Exec(ctx, `
				INSERT INTO timeline_item_tags (timeline_item_id, tag_id)
				SELECT $1, id FROM tags WHERE id = ANY($2) AND user_id = $3
				ON CONFLICT DO NOTHING
			`, itemID, imported.TagIDs, userID); err != nil {
				return err
			}
			// A reminder whose time has passed counts as delivered, so it is
			// not shown as missed on the next open.
			if _, err := tx.Exec(ctx, `
				WITH reminder AS (
					INSERT INTO event_reminders (event_id, user_id, offset_seconds)
					SELECT $1, $2, reminder_offset FROM unnest($4::int[]) AS reminder_offset
					ON CONFLICT DO NOTHING
					RETURNING id, offset_seconds
				)
				INSERT INTO reminder_deliveries (reminder_id, occurrence_start, notified_at)
				SELECT id, $3, now() FROM reminder
				WHERE $3::timestamptz - make_interval(secs => offset_seconds) <= now()
			`, itemID, userID, event.StartsAt, imported.ReminderOffsets); err != nil {
				return err
			}
			if err := insertExceptions(ctx, tx, itemID, imported.Exceptions); err != nil {
				return err
			}
			created[index] = true
		}
		if dryRun {
			return errDryRun
		}
		return nil
	})
	if err != nil && !errors.Is(err, errDryRun) {
		return nil, err
	}
	return created, nil
}

// insertImportedItem creates the parent row of an imported event. It reports
// false, writing nothing, when the user already has an item with the UID: one
// imported before, or one of their own events exported from the app.
func insertImportedItem(ctx context.Context, tx pgx.Tx, userID uuid.UUID, imported ImportedEvent) (uuid.UUID, bool, error) {
	var icalUID *string
	if imported.ICalUID != "" {
		icalUID = &imported.ICalUID
		if ownID, ok := exportedItemID(imported.ICalUID); ok {
			var exists bool
			if err := tx.QueryRow(ctx, `
				SELECT EXISTS (SELECT 1 FROM timeline_items WHERE id = $1 AND user_id = $2)
			`, ownID, userID).Scan(&exists); err != nil {
				return uuid.Nil, false, err
			}
			if exists {
				return uuid.Nil, false, nil
			}
		}
	}
	var itemID uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO timeline_items (user_id, title, description, url, ical_uid)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, ical_uid) WHERE ical_uid IS NOT NULL DO NOTHING
		RETURNING id
	`, userID, imported.Title, imported.Description, imported.URL, icalUID).Scan(&itemID)
	if isNoRows(err) {
		return uuid.Nil, false, nil
	}
	return itemID, err == nil, err
}

// exportedItemID returns the item ID in a UID the app exported.
func exportedItemID(uid string) (uuid.UUID, bool) {
	idText, ok := strings.CutSuffix(uid, ExportUIDSuffix)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idText)
	return id, err == nil
}

// ListEventsForExport returns the user's own events, never events shared with
// them, with the user's own reminders.
func (s *Store) ListEventsForExport(ctx context.Context, userID uuid.UUID) ([]ExportEvent, error) {
	rows, err := s.db.Query(ctx, `
		SELECT item.id, COALESCE(item.ical_uid, item.id::text || $2),
			item.title, item.description, item.url,
			event.starts_at, event.ends_at, event.is_all_day, event.kind, event.location,
			event.start_is_approximate, event.end_is_approximate, event.gathering_seconds,
			event.gathering_is_optional, event.gathering_label,
			event.commute_before_seconds, event.commute_after_seconds, event.main_tag_id,
			event.is_skipped, event.recurrence_rule, event.recurrence_time_zone, event.recurrence_ends_at,
			ARRAY(
				SELECT tag.name
				FROM timeline_item_tags AS item_tag
				JOIN tags AS tag ON tag.id = item_tag.tag_id
				WHERE item_tag.timeline_item_id = item.id
				ORDER BY tag.id <> event.main_tag_id, tag.name
			),
			ARRAY(
				SELECT reminder.offset_seconds
				FROM event_reminders AS reminder
				WHERE reminder.event_id = item.id AND reminder.user_id = $1
				ORDER BY reminder.offset_seconds
			)
		FROM timeline_items AS item
		JOIN events AS event ON event.timeline_item_id = item.id
		WHERE item.user_id = $1
		ORDER BY event.starts_at, item.id
	`, userID, ExportUIDSuffix)
	if err != nil {
		return nil, err
	}
	events, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ExportEvent, error) {
		var event ExportEvent
		err := row.Scan(
			&event.ID, &event.UID, &event.Title, &event.Description, &event.URL,
			&event.StartsAt, &event.EndsAt, &event.IsAllDay, &event.Kind, &event.Location,
			&event.StartIsApproximate, &event.EndIsApproximate, &event.GatheringSeconds,
			&event.GatheringIsOptional, &event.GatheringLabel,
			&event.CommuteBeforeSeconds, &event.CommuteAfterSeconds, &event.MainTagID,
			&event.IsSkipped, &event.RecurrenceRule, &event.RecurrenceTimeZone, &event.RecurrenceEndsAt,
			&event.TagNames, &event.ReminderOffsets,
		)
		return event, err
	})
	if err != nil {
		return nil, err
	}
	var seriesIDs []uuid.UUID
	for _, event := range events {
		if event.RecurrenceRule != nil {
			seriesIDs = append(seriesIDs, event.ID)
		}
	}
	if len(seriesIDs) == 0 {
		return events, nil
	}
	exceptions, err := s.exceptionsByEvent(ctx, userID, seriesIDs)
	if err != nil {
		return nil, err
	}
	for index := range events {
		events[index].Exceptions = exceptions[events[index].ID]
	}
	return events, nil
}
