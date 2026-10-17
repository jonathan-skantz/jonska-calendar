package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateEvent(ctx context.Context, userID uuid.UUID, event Event) (uuid.UUID, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	if event.MainTagID != uuid.Nil {
		var validMainTag bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tags WHERE id = $1 AND user_id = $2)`, event.MainTagID, userID).Scan(&validMainTag); err != nil {
			return uuid.Nil, err
		}
		if !validMainTag {
			return uuid.Nil, ErrTagNotFound
		}
	}

	var itemID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO timeline_items (user_id, title, description, url)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, userID, event.Title, event.Description, event.URL).Scan(&itemID)
	if err != nil {
		return uuid.Nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO events (
			timeline_item_id, starts_at, ends_at, is_all_day, kind, location,
			commute_before_seconds, commute_after_seconds, main_tag_id,
			start_is_approximate, end_is_approximate, gathering_seconds,
			gathering_is_optional, gathering_label
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE(NULLIF($9, '00000000-0000-0000-0000-000000000000'::uuid), (SELECT default_tag_id FROM users WHERE id = $10)), $11, $12, $13, $14, $15)
	`, itemID, event.StartsAt, event.EndsAt, event.IsAllDay, event.Kind, event.Location,
		event.CommuteBeforeSeconds, event.CommuteAfterSeconds, event.MainTagID, userID,
		event.StartIsApproximate, event.EndIsApproximate, event.GatheringSeconds,
		event.GatheringIsOptional, event.GatheringLabel)
	if err != nil {
		return uuid.Nil, err
	}

	result, err := tx.Exec(ctx, `
		INSERT INTO timeline_item_tags (timeline_item_id, tag_id)
		SELECT $1, COALESCE(NULLIF($2, '00000000-0000-0000-0000-000000000000'::uuid), default_tag_id)
		FROM users WHERE id = $3
	`, itemID, event.MainTagID, userID)
	if err != nil {
		return uuid.Nil, err
	}
	if result.RowsAffected() != 1 {
		return uuid.Nil, ErrDefaultTagNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return itemID, nil
}

func (s *Store) UpdateEvent(ctx context.Context, userID, eventID uuid.UUID, event Event) error {
	// An event spans the generic timeline row and the event subtype row, so
	// both updates must commit together.
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if event.MainTagID == uuid.Nil {
		if err := tx.QueryRow(ctx, `SELECT main_tag_id FROM events WHERE timeline_item_id = $1`, eventID).Scan(&event.MainTagID); err != nil {
			return err
		}
	}
	var validMainTag bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tags WHERE id = $1 AND user_id = $2)`, event.MainTagID, userID).Scan(&validMainTag); err != nil {
		return err
	}
	if !validMainTag {
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
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	result, err = tx.Exec(ctx, `
		UPDATE events
		SET starts_at = $2, ends_at = $3, is_all_day = $4, kind = $5, location = $6,
			commute_before_seconds = $7, commute_after_seconds = $8, main_tag_id = $9,
			start_is_approximate = $10, end_is_approximate = $11, gathering_seconds = $12,
			gathering_is_optional = $13, gathering_label = $14
		WHERE timeline_item_id = $1
	`, eventID, event.StartsAt, event.EndsAt, event.IsAllDay, event.Kind, event.Location,
		event.CommuteBeforeSeconds, event.CommuteAfterSeconds, event.MainTagID,
		event.StartIsApproximate, event.EndIsApproximate, event.GatheringSeconds,
		event.GatheringIsOptional, event.GatheringLabel)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return tx.Commit(ctx)
}
