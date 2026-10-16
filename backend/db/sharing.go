package db

import (
	"context"

	"github.com/google/uuid"
)

func (s *Store) ShareEvent(ctx context.Context, ownerID, eventID, recipientID uuid.UUID) (uuid.UUID, error) {
	var shareID uuid.UUID
	err := s.db.QueryRow(ctx, `
		INSERT INTO event_shares (event_id, recipient_user_id)
		SELECT e.timeline_item_id, $3
		FROM events AS e
		JOIN timeline_items AS item ON item.id = e.timeline_item_id
		WHERE e.timeline_item_id = $2
		  AND item.user_id = $1
		  AND item.user_id <> $3
		RETURNING id
	`, ownerID, eventID, recipientID).Scan(&shareID)
	return shareID, err
}

func (s *Store) FindSharedEvent(ctx context.Context, recipientID, shareID uuid.UUID) (SharedEvent, error) {
	var event SharedEvent
	err := s.db.QueryRow(ctx, `
		-- Source fields come from the owner's event. Override fields below are
		-- recipient-local and may be absent until the recipient changes them.
		SELECT
			share.id, share.event_id, share.recipient_user_id,
			item.title, item.description, item.url, event.location,
			event.starts_at, event.ends_at, event.is_all_day, event.kind,
			override.title, override.description, override.url,
			override.location, override.custom_info
		FROM event_shares AS share
		JOIN timeline_items AS item ON item.id = share.event_id
		JOIN events AS event ON event.timeline_item_id = share.event_id
		LEFT JOIN event_share_overrides AS override ON override.event_share_id = share.id
		WHERE share.id = $1 AND share.recipient_user_id = $2
	`, shareID, recipientID).Scan(
		&event.ShareID, &event.EventID, &event.RecipientUserID,
		&event.OriginalTitle, &event.OriginalDescription, &event.OriginalURL,
		&event.OriginalLocation, &event.StartsAt, &event.EndsAt, &event.IsAllDay, &event.Kind,
		&event.LocalTitle, &event.LocalDescription, &event.LocalURL,
		&event.LocalLocation, &event.PrivateCustomInfo,
	)
	return event, err
}

func (s *Store) UpdateSharedEventOverride(ctx context.Context, recipientID, shareID uuid.UUID, event SharedEvent) error {
	result, err := s.db.Exec(ctx, `
		INSERT INTO event_share_overrides (
			event_share_id, title, description, url, location, custom_info
		)
		SELECT share.id, $3, $4, $5, $6, $7
		FROM event_shares AS share
		WHERE share.id = $1 AND share.recipient_user_id = $2
		ON CONFLICT (event_share_id) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			url = EXCLUDED.url,
			location = EXCLUDED.location,
			custom_info = EXCLUDED.custom_info,
			updated_at = now()
	`, shareID, recipientID, event.LocalTitle, event.LocalDescription,
		event.LocalURL, event.LocalLocation, event.PrivateCustomInfo)
	if err == nil && result.RowsAffected() == 0 {
		return ErrSharedEventNotFound
	}
	return err
}
