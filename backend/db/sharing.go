package db

import (
	"context"

	"github.com/google/uuid"
)

// Every query here is scoped to the event owner: the share's event must belong
// to a timeline item owned by ownerID. The owner is derived through
// event_shares.event_id -> timeline_items.user_id, never stored on the share.

func (s *Store) ShareEvent(ctx context.Context, ownerID, eventID, recipientID uuid.UUID, fields ShareFields) (uuid.UUID, error) {
	var shareID uuid.UUID
	err := s.db.QueryRow(ctx, `
		INSERT INTO event_shares (event_id, recipient_user_id, title, description, url, location)
		SELECT e.timeline_item_id, $3, $4, $5, $6, $7
		FROM events AS e
		JOIN timeline_items AS item ON item.id = e.timeline_item_id
		WHERE e.timeline_item_id = $2
		  AND item.user_id = $1
		  AND item.user_id <> $3
		RETURNING id
	`, ownerID, eventID, recipientID,
		fields.Title, fields.Description, fields.URL, fields.Location,
	).Scan(&shareID)
	if isUniqueViolation(err) {
		return uuid.Nil, ErrAlreadyShared
	}
	if isNoRows(err) {
		return uuid.Nil, ErrEventNotFound
	}
	return shareID, err
}

// ListEventShares returns the recipients of an event. It returns
// ErrEventNotFound when the event does not exist or is not the caller's.
func (s *Store) ListEventShares(ctx context.Context, ownerID, eventID uuid.UUID) ([]EventShare, error) {
	var owned bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM events AS e
			JOIN timeline_items AS item ON item.id = e.timeline_item_id
			WHERE e.timeline_item_id = $2 AND item.user_id = $1
		)
	`, ownerID, eventID).Scan(&owned); err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrEventNotFound
	}

	rows, err := s.db.Query(ctx, `
		SELECT share.id, recipient.id, recipient.email,
			user_display_name(recipient.display_name, recipient.email),
			share.title, share.description, share.url, share.location
		FROM event_shares AS share
		JOIN users AS recipient ON recipient.id = share.recipient_user_id
		WHERE share.event_id = $1
		ORDER BY share.created_at, share.id
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shares := []EventShare{}
	for rows.Next() {
		var share EventShare
		if err := rows.Scan(
			&share.ID, &share.RecipientUserID, &share.RecipientEmail, &share.RecipientDisplayName,
			&share.Title, &share.Description, &share.URL, &share.Location,
		); err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}
	return shares, rows.Err()
}

func (s *Store) UpdateEventShare(ctx context.Context, ownerID, eventID, shareID uuid.UUID, fields ShareFields) error {
	result, err := s.db.Exec(ctx, `
		UPDATE event_shares AS share
		SET title = $4, description = $5, url = $6, location = $7, updated_at = now()
		FROM timeline_items AS item
		WHERE share.id = $3
		  AND share.event_id = $2
		  AND item.id = share.event_id
		  AND item.user_id = $1
	`, ownerID, eventID, shareID,
		fields.Title, fields.Description, fields.URL, fields.Location)
	if err == nil && result.RowsAffected() == 0 {
		return ErrSharedEventNotFound
	}
	return err
}

// DeleteEventShare also deletes the recipient's own reminders for the event,
// since they can no longer see it.
func (s *Store) DeleteEventShare(ctx context.Context, ownerID, eventID, shareID uuid.UUID) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var recipientID uuid.UUID
	err = tx.QueryRow(ctx, `
		DELETE FROM event_shares AS share
		USING timeline_items AS item
		WHERE share.id = $3
		  AND share.event_id = $2
		  AND item.id = share.event_id
		  AND item.user_id = $1
		RETURNING share.recipient_user_id
	`, ownerID, eventID, shareID).Scan(&recipientID)
	if isNoRows(err) {
		return ErrSharedEventNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM event_reminders WHERE event_id = $1 AND user_id = $2
	`, eventID, recipientID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
