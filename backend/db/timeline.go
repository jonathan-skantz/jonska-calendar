package db

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListTimeline(ctx context.Context, userID uuid.UUID) ([]TimelineEntry, error) {
	// Shared events are projected into the recipient's timeline; they are not
	// copied into timeline_items owned by the recipient.
	entries, err := s.listOwnedTimeline(ctx, userID)
	if err != nil {
		return nil, err
	}

	sharedEntries, err := s.listSharedTimeline(ctx, userID)
	if err != nil {
		return nil, err
	}
	entries = append(entries, sharedEntries...)
	if err := s.populateTimelineTags(ctx, userID, entries); err != nil {
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

func (s *Store) listOwnedTimeline(ctx context.Context, userID uuid.UUID) ([]TimelineEntry, error) {
	entries := make([]TimelineEntry, 0)
	eventRows, err := s.db.Query(ctx, `
		SELECT item.id, item.title, item.description, item.url,
			event.starts_at, event.ends_at, event.is_all_day, event.kind, event.location,
			event.start_is_approximate, event.end_is_approximate, event.gathering_seconds,
			event.gathering_is_optional, event.gathering_label,
			event.commute_before_seconds, event.commute_after_seconds, event.main_tag_id,
			event.is_skipped,
			(SELECT count(*) FROM event_shares AS share WHERE share.event_id = item.id)
		FROM timeline_items AS item
		JOIN events AS event ON event.timeline_item_id = item.id
		WHERE item.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	for eventRows.Next() {
		var entry TimelineEntry
		entry.Type = "event"
		if err := eventRows.Scan(
			&entry.ID, &entry.Title, &entry.Description, &entry.URL,
			&entry.StartsAt, &entry.EndsAt, &entry.IsAllDay, &entry.Kind, &entry.Location,
			&entry.StartIsApproximate, &entry.EndIsApproximate, &entry.GatheringSeconds,
			&entry.GatheringIsOptional, &entry.GatheringLabel,
			&entry.CommuteBeforeSeconds, &entry.CommuteAfterSeconds,
			&entry.MainTagID, &entry.IsSkipped, &entry.ShareCount,
		); err != nil {
			eventRows.Close()
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := eventRows.Err(); err != nil {
		eventRows.Close()
		return nil, err
	}
	eventRows.Close()

	taskRows, err := s.db.Query(ctx, `
		SELECT item.id, item.title, item.description, item.url,
			task.deadline_at, task.is_completed
		FROM timeline_items AS item
		JOIN tasks AS task ON task.timeline_item_id = item.id
		WHERE item.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	for taskRows.Next() {
		var entry TimelineEntry
		entry.Type = "task"
		if err := taskRows.Scan(
			&entry.ID, &entry.Title, &entry.Description, &entry.URL,
			&entry.DeadlineAt, &entry.IsCompleted,
		); err != nil {
			taskRows.Close()
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := taskRows.Err(); err != nil {
		taskRows.Close()
		return nil, err
	}
	taskRows.Close()
	return entries, nil
}

func (s *Store) listSharedTimeline(ctx context.Context, userID uuid.UUID) ([]TimelineEntry, error) {
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
			user_display_name(owner.display_name, owner.email)
		FROM event_shares AS share
		JOIN timeline_items AS item ON item.id = share.event_id
		JOIN events AS event ON event.timeline_item_id = share.event_id
		JOIN users AS owner ON owner.id = item.user_id
		WHERE share.recipient_user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]TimelineEntry, 0)
	for rows.Next() {
		var entry TimelineEntry
		entry.Type = "event"
		entry.IsShared = true
		if err := rows.Scan(
			&entry.ID, &entry.Title, &entry.Description, &entry.URL,
			&entry.StartsAt, &entry.EndsAt, &entry.IsAllDay, &entry.Kind,
			&entry.StartIsApproximate, &entry.EndIsApproximate, &entry.GatheringSeconds,
			&entry.GatheringIsOptional, &entry.GatheringLabel,
			&entry.Location,
			&entry.SharedBy,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Store) DeleteTimelineItem(ctx context.Context, userID, itemID uuid.UUID) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM timeline_items
		WHERE id = $1 AND user_id = $2
	`, itemID, userID)
	if err == nil && result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
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
