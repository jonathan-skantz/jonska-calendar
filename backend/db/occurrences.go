package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Occurrences of a repeating event are not stored; only the series and the
// occurrences that differ from it are. The caller works out the rule and
// which exceptions still match it; the store keeps them consistent in one
// transaction and checks ownership.

// GetOwnEvent returns one of the user's own events with its tags and, for a
// series, its exceptions.
func (s *Store) GetOwnEvent(ctx context.Context, userID, eventID uuid.UUID) (TimelineEntry, error) {
	entries, err := s.listOwnedEvents(ctx, userID, &eventID)
	if err != nil {
		return TimelineEntry{}, err
	}
	if len(entries) == 0 {
		return TimelineEntry{}, ErrEventNotFound
	}
	if err := s.populateTimelineTags(ctx, userID, entries); err != nil {
		return TimelineEntry{}, err
	}
	if err := s.populateExceptions(ctx, userID, entries); err != nil {
		return TimelineEntry{}, err
	}
	return entries[0], nil
}

// UpdateSeries updates an event that is or becomes a series, or stops being
// one, and replaces its exceptions with the given ones.
func (s *Store) UpdateSeries(ctx context.Context, userID, eventID uuid.UUID, event Event, exceptions []OccurrenceException) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		if err := updateEventTx(ctx, tx, userID, eventID, event); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM event_occurrence_exceptions WHERE event_id = $1`, eventID); err != nil {
			return err
		}
		return insertExceptions(ctx, tx, eventID, exceptions)
	})
}

// SetOccurrenceException records how one occurrence of the user's series
// differs from it, replacing an earlier exception for the occurrence.
// recurrenceEndsAt is the series' end with the exception applied.
func (s *Store) SetOccurrenceException(ctx context.Context, userID, eventID uuid.UUID, exception OccurrenceException, recurrenceEndsAt *time.Time) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, `
			UPDATE events AS e
			SET recurrence_ends_at = $3
			FROM timeline_items AS item
			WHERE e.timeline_item_id = $1 AND item.id = e.timeline_item_id
			  AND item.user_id = $2 AND e.recurrence_rule IS NOT NULL
		`, eventID, userID, recurrenceEndsAt)
		if err := requireRowsAffected(result, err, ErrEventNotFound); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM event_occurrence_exceptions WHERE event_id = $1 AND original_start = $2
		`, eventID, exception.OriginalStart); err != nil {
			return err
		}
		return insertExceptions(ctx, tx, eventID, []OccurrenceException{exception})
	})
}

// EndSeries ends the user's series before the occurrence that starts at
// from, with rule as its new RRULE value, and drops the exceptions from it on.
func (s *Store) EndSeries(ctx context.Context, userID, eventID uuid.UUID, from time.Time, rule string, recurrenceEndsAt *time.Time) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		return endSeriesTx(ctx, tx, userID, eventID, from, rule, recurrenceEndsAt)
	})
}

func endSeriesTx(ctx context.Context, tx pgx.Tx, userID, eventID uuid.UUID, from time.Time, rule string, recurrenceEndsAt *time.Time) error {
	result, err := tx.Exec(ctx, `
		UPDATE events AS e
		SET recurrence_rule = $3, recurrence_ends_at = $4
		FROM timeline_items AS item
		WHERE e.timeline_item_id = $1 AND item.id = e.timeline_item_id
		  AND item.user_id = $2 AND e.recurrence_rule IS NOT NULL
	`, eventID, userID, rule, recurrenceEndsAt)
	if err := requireRowsAffected(result, err, ErrEventNotFound); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		DELETE FROM event_occurrence_exceptions WHERE event_id = $1 AND original_start >= $2
	`, eventID, from)
	return err
}

// SplitSeries changes a series from the occurrence that starts at from on.
// In one transaction the old series ends before it (with oldRule as its new
// RRULE value) and a new series starts with the given event. The new series
// keeps the old one's tags, shares and everyone's reminders, so recipients
// and reminders carry on; exceptions are the old ones the caller re-keyed
// for the new series. It returns the new series' id.
func (s *Store) SplitSeries(ctx context.Context, userID, eventID uuid.UUID, from time.Time,
	oldRule string, oldEndsAt *time.Time, event Event, exceptions []OccurrenceException) (uuid.UUID, error) {
	var newID uuid.UUID
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		if err := endSeriesTx(ctx, tx, userID, eventID, from, oldRule, oldEndsAt); err != nil {
			return err
		}
		// The new series keeps the old main tag unless another of the old
		// series' tags was chosen.
		if err := tx.QueryRow(ctx, `
			SELECT CASE WHEN EXISTS (
				SELECT 1 FROM timeline_item_tags WHERE timeline_item_id = $1 AND tag_id = $2
			) THEN $2 ELSE main_tag_id END
			FROM events WHERE timeline_item_id = $1
		`, eventID, event.MainTagID).Scan(&event.MainTagID); err != nil {
			return err
		}
		var err error
		newID, err = insertTimelineItem(ctx, tx, userID, event.Title, event.Description, event.URL)
		if err != nil {
			return err
		}
		if err := insertEventRows(ctx, tx, userID, newID, event); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO timeline_item_tags (timeline_item_id, tag_id)
			SELECT $2, tag_id FROM timeline_item_tags WHERE timeline_item_id = $1
			ON CONFLICT DO NOTHING
		`, eventID, newID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_shares (event_id, recipient_user_id, title, description, url, location)
			SELECT $2, recipient_user_id, title, description, url, location
			FROM event_shares WHERE event_id = $1
		`, eventID, newID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_reminders (event_id, user_id, offset_seconds)
			SELECT $2, user_id, offset_seconds FROM event_reminders WHERE event_id = $1
		`, eventID, newID); err != nil {
			return err
		}
		return insertExceptions(ctx, tx, newID, exceptions)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return newID, nil
}

func insertExceptions(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, exceptions []OccurrenceException) error {
	for _, exception := range exceptions {
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_occurrence_exceptions
				(event_id, original_start, is_cancelled, starts_at, ends_at, is_skipped)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, eventID, exception.OriginalStart, exception.IsCancelled,
			exception.StartsAt, exception.EndsAt, exception.IsSkipped); err != nil {
			return err
		}
	}
	return nil
}

// populateExceptions attaches their exceptions to the series among entries:
// the user's own, or ones shared with them. The owner's skip decisions are
// only attached to the owner's own series.
func (s *Store) populateExceptions(ctx context.Context, userID uuid.UUID, entries []TimelineEntry) error {
	var seriesIDs []uuid.UUID
	for _, entry := range entries {
		if entry.RecurrenceRule != nil {
			seriesIDs = append(seriesIDs, entry.ID)
		}
	}
	if len(seriesIDs) == 0 {
		return nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT exception.event_id, exception.original_start, exception.is_cancelled,
			exception.starts_at, exception.ends_at,
			CASE WHEN item.user_id = $1 THEN exception.is_skipped END
		FROM event_occurrence_exceptions AS exception
		JOIN timeline_items AS item ON item.id = exception.event_id
		WHERE exception.event_id = ANY($2)
		  AND `+canSeeEventSQL+`
		ORDER BY exception.original_start
	`, userID, seriesIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	byEvent := make(map[uuid.UUID][]OccurrenceException)
	for rows.Next() {
		var eventID uuid.UUID
		var exception OccurrenceException
		if err := rows.Scan(&eventID, &exception.OriginalStart, &exception.IsCancelled,
			&exception.StartsAt, &exception.EndsAt, &exception.IsSkipped); err != nil {
			return err
		}
		byEvent[eventID] = append(byEvent[eventID], exception)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for index := range entries {
		entries[index].Exceptions = byEvent[entries[index].ID]
	}
	return nil
}
