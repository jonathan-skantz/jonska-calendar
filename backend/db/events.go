package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateEvent(ctx context.Context, userID uuid.UUID, event Event) (uuid.UUID, error) {
	var itemID uuid.UUID
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		if event.MainTagID != uuid.Nil {
			owned, err := tagOwnedBy(ctx, tx, event.MainTagID, userID)
			if err != nil {
				return err
			}
			if !owned {
				return ErrTagNotFound
			}
		}

		var err error
		itemID, err = insertTimelineItem(ctx, tx, userID, event.Title, event.Description, event.URL)
		if err != nil {
			return err
		}
		return insertEventRows(ctx, tx, userID, itemID, event)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return itemID, nil
}

// insertEventRows writes the event subtype row and attaches its main tag, for
// a timeline item the caller just created in tx. The caller checks that the
// main tag, when chosen, belongs to the user.
func insertEventRows(ctx context.Context, tx pgx.Tx, userID, itemID uuid.UUID, event Event) error {
	// Without a chosen main tag, the user's default tag is used.
	if _, err := tx.Exec(ctx, `
		INSERT INTO events (
			timeline_item_id, starts_at, ends_at, is_all_day, kind, location,
			commute_before_seconds, commute_after_seconds, main_tag_id,
			start_is_approximate, end_is_approximate, gathering_seconds,
			gathering_is_optional, gathering_label, is_skipped
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE(NULLIF($9, '00000000-0000-0000-0000-000000000000'::uuid), (SELECT default_tag_id FROM users WHERE id = $10)), $11, $12, $13, $14, $15, $16)
	`, itemID, event.StartsAt, event.EndsAt, event.IsAllDay, event.Kind, event.Location,
		event.CommuteBeforeSeconds, event.CommuteAfterSeconds, event.MainTagID, userID,
		event.StartIsApproximate, event.EndIsApproximate, event.GatheringSeconds,
		event.GatheringIsOptional, event.GatheringLabel, event.IsSkipped); err != nil {
		return err
	}

	// The main tag is also the event's first attached tag.
	result, err := tx.Exec(ctx, `
		INSERT INTO timeline_item_tags (timeline_item_id, tag_id)
		SELECT $1, COALESCE(NULLIF($2, '00000000-0000-0000-0000-000000000000'::uuid), default_tag_id)
		FROM users WHERE id = $3
	`, itemID, event.MainTagID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrDefaultTagNotFound
	}
	return nil
}

func (s *Store) UpdateEvent(ctx context.Context, userID, eventID uuid.UUID, event Event) error {
	// An event spans the generic timeline row and the event subtype row, so
	// both updates must commit together.
	return s.withTx(ctx, func(tx pgx.Tx) error {
		if event.MainTagID == uuid.Nil {
			if err := tx.QueryRow(ctx, `
				SELECT e.main_tag_id
				FROM events AS e
				JOIN timeline_items AS item ON item.id = e.timeline_item_id
				WHERE e.timeline_item_id = $1 AND item.user_id = $2
			`, eventID, userID).Scan(&event.MainTagID); err != nil {
				if isNoRows(err) {
					return ErrEventNotFound
				}
				return err
			}
		}
		owned, err := tagOwnedBy(ctx, tx, event.MainTagID, userID)
		if err != nil {
			return err
		}
		if !owned {
			return ErrTagNotFound
		}
		var attachedMainTag bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM timeline_item_tags AS item_tag
				JOIN timeline_items AS item ON item.id = item_tag.timeline_item_id
				WHERE item_tag.timeline_item_id = $1
				  AND item_tag.tag_id = $2
				  AND item.user_id = $3
			)
		`, eventID, event.MainTagID, userID).Scan(&attachedMainTag); err != nil {
			return err
		}
		if !attachedMainTag {
			return ErrTagNotFound
		}

		result, err := tx.Exec(ctx, `
			UPDATE timeline_items
			SET title = $3, description = $4, url = $5, updated_at = now()
			WHERE id = $1 AND user_id = $2
		`, eventID, userID, event.Title, event.Description, event.URL)
		if err := requireRowsAffected(result, err, ErrEventNotFound); err != nil {
			return err
		}

		// A moved event's reminders fire again at their new time, for every user.
		if _, err := tx.Exec(ctx, `
			UPDATE event_reminders AS reminder
			SET notified_at = NULL
			FROM events AS e
			WHERE reminder.event_id = $1
			  AND e.timeline_item_id = $1
			  AND e.starts_at <> $2
		`, eventID, event.StartsAt); err != nil {
			return err
		}

		result, err = tx.Exec(ctx, `
			UPDATE events
			SET starts_at = $2, ends_at = $3, is_all_day = $4, kind = $5, location = $6,
				commute_before_seconds = $7, commute_after_seconds = $8, main_tag_id = $9,
				start_is_approximate = $10, end_is_approximate = $11, gathering_seconds = $12,
				gathering_is_optional = $13, gathering_label = $14, is_skipped = $15
			WHERE timeline_item_id = $1
		`, eventID, event.StartsAt, event.EndsAt, event.IsAllDay, event.Kind, event.Location,
			event.CommuteBeforeSeconds, event.CommuteAfterSeconds, event.MainTagID,
			event.StartIsApproximate, event.EndIsApproximate, event.GatheringSeconds,
			event.GatheringIsOptional, event.GatheringLabel, event.IsSkipped)
		return requireRowsAffected(result, err, ErrEventNotFound)
	})
}
