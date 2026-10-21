package db

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ListTimeline returns the user's events and the events shared with them
// that may happen in [from, to): plain events that overlap it, and series
// that may have an occurrence in it, as series. Tasks are all returned.
func (s *Store) ListTimeline(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]TimelineEntry, error) {
	return s.listTimeline(ctx, userID, &timeWindow{from: from, to: to})
}

type timeWindow struct {
	from, to time.Time
}

// bounds returns the window as query arguments; both are NULL without one.
func (window *timeWindow) bounds() (from, to *time.Time) {
	if window == nil {
		return nil, nil
	}
	return &window.from, &window.to
}

// inWindowSQL keeps the events that may happen in the window given as $2
// and $3, or every event when they are NULL. A series lasts until its last
// occurrence ends, or forever, and a moved occurrence may start before the
// series does.
const inWindowSQL = `($2::timestamptz IS NULL OR (
	LEAST(event.starts_at, (
		SELECT min(exception.starts_at) FROM event_occurrence_exceptions AS exception
		WHERE exception.event_id = event.timeline_item_id
	)) < $3
	AND CASE WHEN event.recurrence_rule IS NULL THEN event.ends_at
		ELSE COALESCE(event.recurrence_ends_at, 'infinity') END > $2
))`

func (s *Store) listTimeline(ctx context.Context, userID uuid.UUID, window *timeWindow) ([]TimelineEntry, error) {
	// Shared events are projected into the recipient's timeline; they are not
	// copied into timeline_items owned by the recipient.
	entries, err := s.listOwnedTimeline(ctx, userID, window)
	if err != nil {
		return nil, err
	}

	sharedEntries, err := s.listSharedTimeline(ctx, userID, window)
	if err != nil {
		return nil, err
	}
	entries = append(entries, sharedEntries...)
	if err := s.populateTimelineTags(ctx, userID, entries); err != nil {
		return nil, err
	}
	if err := s.populateExceptions(ctx, userID, entries); err != nil {
		return nil, err
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return timelineEntryTime(entries[i]).Before(timelineEntryTime(entries[j]))
	})
	return entries, nil
}

func (s *Store) populateTimelineTags(ctx context.Context, userID uuid.UUID, entries []TimelineEntry) error {
	// Tags are personal organization, so only tags and items owned by this
	// user may appear in the timeline response.
	rows, err := s.db.Query(ctx, `
		SELECT item_tag.timeline_item_id, tag.id, tag.name, tag.color
		FROM timeline_item_tags AS item_tag
		JOIN timeline_items AS item ON item.id = item_tag.timeline_item_id
		JOIN tags AS tag ON tag.id = item_tag.tag_id
		WHERE item.user_id = $1
	`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	tagsByItem := make(map[uuid.UUID][]Tag)
	for rows.Next() {
		var itemID uuid.UUID
		var tag Tag
		if err := rows.Scan(&itemID, &tag.ID, &tag.Name, &tag.Color); err != nil {
			return err
		}
		tagsByItem[itemID] = append(tagsByItem[itemID], tag)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for index := range entries {
		entries[index].Tags = tagsByItem[entries[index].ID]
	}
	return nil
}

func (s *Store) listOwnedTimeline(ctx context.Context, userID uuid.UUID, window *timeWindow) ([]TimelineEntry, error) {
	events, err := s.listOwnedEvents(ctx, userID, nil, window)
	if err != nil {
		return nil, err
	}
	tasks, err := s.listOwnedTasks(ctx, userID)
	if err != nil {
		return nil, err
	}
	return append(events, tasks...), nil
}

// listOwnedEvents lists the user's own events, only eventID when it is not
// nil, and only those in the window when there is one.
func (s *Store) listOwnedEvents(ctx context.Context, userID uuid.UUID, eventID *uuid.UUID, window *timeWindow) ([]TimelineEntry, error) {
	from, to := window.bounds()
	rows, err := s.db.Query(ctx, `
		SELECT item.id, item.title, item.description, item.url,
			event.starts_at, event.ends_at, event.is_all_day, event.kind, event.location,
			event.start_is_approximate, event.end_is_approximate, event.gathering_seconds,
			event.gathering_is_optional, event.gathering_label,
			event.commute_before_seconds, event.commute_after_seconds, event.main_tag_id,
			event.is_skipped,
			(SELECT count(*) FROM event_shares AS share WHERE share.event_id = item.id),
			event.recurrence_rule, event.recurrence_time_zone, event.recurrence_ends_at
		FROM timeline_items AS item
		JOIN events AS event ON event.timeline_item_id = item.id
		WHERE item.user_id = $1 AND ($4::uuid IS NULL OR item.id = $4) AND `+inWindowSQL+`
	`, userID, from, to, eventID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (TimelineEntry, error) {
		entry := TimelineEntry{Type: "event"}
		err := row.Scan(
			&entry.ID, &entry.Title, &entry.Description, &entry.URL,
			&entry.StartsAt, &entry.EndsAt, &entry.IsAllDay, &entry.Kind, &entry.Location,
			&entry.StartIsApproximate, &entry.EndIsApproximate, &entry.GatheringSeconds,
			&entry.GatheringIsOptional, &entry.GatheringLabel,
			&entry.CommuteBeforeSeconds, &entry.CommuteAfterSeconds,
			&entry.MainTagID, &entry.IsSkipped, &entry.ShareCount,
			&entry.RecurrenceRule, &entry.RecurrenceTimeZone, &entry.RecurrenceEndsAt,
		)
		return entry, err
	})
}

func (s *Store) listOwnedTasks(ctx context.Context, userID uuid.UUID) ([]TimelineEntry, error) {
	rows, err := s.db.Query(ctx, `
		SELECT item.id, item.title, item.description, item.url,
			task.deadline_at, task.is_completed
		FROM timeline_items AS item
		JOIN tasks AS task ON task.timeline_item_id = item.id
		WHERE item.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (TimelineEntry, error) {
		entry := TimelineEntry{Type: "task"}
		err := row.Scan(
			&entry.ID, &entry.Title, &entry.Description, &entry.URL,
			&entry.DeadlineAt, &entry.IsCompleted,
		)
		return entry, err
	})
}

func (s *Store) listSharedTimeline(ctx context.Context, userID uuid.UUID, window *timeWindow) ([]TimelineEntry, error) {
	from, to := window.bounds()
	rows, err := s.db.Query(ctx, `
		-- Recipients only ever see the effective values: the owner's
		-- per-recipient field when set, otherwise the event's own value. The
		-- owner's original fields, tags, commute, and skip decision are
		-- never projected.
		-- Gathering describes the event itself, so it is projected.
		SELECT
			share.event_id,
			COALESCE(share.title, item.title),
			COALESCE(share.description, item.description),
			COALESCE(share.url, item.url),
			event.starts_at, event.ends_at, event.is_all_day, event.kind,
			event.start_is_approximate, event.end_is_approximate, event.gathering_seconds,
			event.gathering_is_optional, event.gathering_label,
			COALESCE(share.location, event.location),
			user_display_name(owner.display_name, owner.email),
			event.recurrence_rule, event.recurrence_time_zone, event.recurrence_ends_at
		FROM event_shares AS share
		JOIN timeline_items AS item ON item.id = share.event_id
		JOIN events AS event ON event.timeline_item_id = share.event_id
		JOIN users AS owner ON owner.id = item.user_id
		WHERE share.recipient_user_id = $1 AND `+inWindowSQL+`
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (TimelineEntry, error) {
		entry := TimelineEntry{Type: "event", IsShared: true}
		err := row.Scan(
			&entry.ID, &entry.Title, &entry.Description, &entry.URL,
			&entry.StartsAt, &entry.EndsAt, &entry.IsAllDay, &entry.Kind,
			&entry.StartIsApproximate, &entry.EndIsApproximate, &entry.GatheringSeconds,
			&entry.GatheringIsOptional, &entry.GatheringLabel,
			&entry.Location,
			&entry.SharedBy,
			&entry.RecurrenceRule, &entry.RecurrenceTimeZone, &entry.RecurrenceEndsAt,
		)
		return entry, err
	})
}

// insertTimelineItem creates the parent row that owns an event or task. The
// caller inserts the subtype row in the same transaction.
func insertTimelineItem(ctx context.Context, tx pgx.Tx, userID uuid.UUID, title string, description, url *string) (uuid.UUID, error) {
	var itemID uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO timeline_items (user_id, title, description, url)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, userID, title, description, url).Scan(&itemID)
	return itemID, err
}

func tagOwnedBy(ctx context.Context, tx pgx.Tx, tagID, userID uuid.UUID) (bool, error) {
	var owned bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tags WHERE id = $1 AND user_id = $2)`, tagID, userID).Scan(&owned)
	return owned, err
}

func (s *Store) DeleteTimelineItem(ctx context.Context, userID, itemID uuid.UUID) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM timeline_items
		WHERE id = $1 AND user_id = $2
	`, itemID, userID)
	return requireRowsAffected(result, err, ErrTimelineItemNotFound)
}

func timelineEntryTime(entry TimelineEntry) time.Time {
	if entry.StartsAt != nil {
		return *entry.StartsAt
	}
	if entry.DeadlineAt != nil {
		return *entry.DeadlineAt
	}
	return time.Time{}
}
