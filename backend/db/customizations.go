package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// A recipient customizes only events they receive. A share is the
// recipient's when its recipient_user_id is theirs; a subscribed event when
// its subscription's user_id is. The owner of a shared event never reads
// these tables.

const customizationColumns = `title, description, url, location, note, start_shift_seconds, end_shift_seconds`

// customizationSQL selects a LEFT JOINed customization aliased as custom,
// with whether there is one first.
const customizationSQL = `custom.updated_at IS NOT NULL,
	custom.title, custom.description, custom.url, custom.location, custom.note,
	COALESCE(custom.start_shift_seconds, 0), COALESCE(custom.end_shift_seconds, 0)`

// scannedCustomization receives customizationSQL.
type scannedCustomization struct {
	exists bool
	Customization
}

func (scanned *scannedCustomization) targets() []any {
	return []any{
		&scanned.exists, &scanned.Title, &scanned.Description, &scanned.URL, &scanned.Location,
		&scanned.Note, &scanned.StartShiftSeconds, &scanned.EndShiftSeconds,
	}
}

func (scanned *scannedCustomization) value() *Customization {
	if !scanned.exists {
		return nil
	}
	customization := scanned.Customization
	return &customization
}

func customizationArgs(customization Customization) []any {
	return []any{
		customization.Title, customization.Description, customization.URL, customization.Location,
		customization.Note, customization.StartShiftSeconds, customization.EndShiftSeconds,
	}
}

// CustomizeSharedEvent sets the recipient's customization of an event shared
// with them, replacing any earlier one.
func (s *Store) CustomizeSharedEvent(ctx context.Context, recipientID, eventID uuid.UUID, customization Customization) error {
	result, err := s.db.Exec(ctx, `
		INSERT INTO share_customizations (event_share_id, `+customizationColumns+`)
		SELECT share.id, $3, $4, $5, $6, $7, $8, $9
		FROM event_shares AS share
		WHERE share.event_id = $2 AND share.recipient_user_id = $1
		ON CONFLICT (event_share_id) DO UPDATE SET
			title = EXCLUDED.title, description = EXCLUDED.description, url = EXCLUDED.url,
			location = EXCLUDED.location, note = EXCLUDED.note,
			start_shift_seconds = EXCLUDED.start_shift_seconds, end_shift_seconds = EXCLUDED.end_shift_seconds,
			updated_at = now()
	`, append([]any{recipientID, eventID}, customizationArgs(customization)...)...)
	return requireRowsAffected(result, err, ErrSharedEventNotFound)
}

// UncustomizeSharedEvent removes the recipient's customization, if any, of
// an event shared with them.
func (s *Store) UncustomizeSharedEvent(ctx context.Context, recipientID, eventID uuid.UUID) error {
	var shared bool
	err := s.db.QueryRow(ctx, `
		WITH share AS (
			SELECT id FROM event_shares WHERE event_id = $2 AND recipient_user_id = $1
		), removed AS (
			DELETE FROM share_customizations WHERE event_share_id IN (SELECT id FROM share)
		)
		SELECT EXISTS (SELECT 1 FROM share)
	`, recipientID, eventID).Scan(&shared)
	if err == nil && !shared {
		return ErrSharedEventNotFound
	}
	return err
}

// subscribedEventUID returns the subscription and UID of one of the user's
// subscribed events. Its row is replaced on every refresh, so the
// customization is kept by UID.
func subscribedEventUID(ctx context.Context, tx pgx.Tx, userID, eventID uuid.UUID) (uuid.UUID, string, error) {
	var subscriptionID uuid.UUID
	var uid *string
	err := tx.QueryRow(ctx, `
		SELECT event.subscription_id, event.uid
		FROM subscription_events AS event
		JOIN calendar_subscriptions AS subscription ON subscription.id = event.subscription_id
		WHERE event.id = $2 AND subscription.user_id = $1
	`, userID, eventID).Scan(&subscriptionID, &uid)
	if isNoRows(err) {
		return uuid.Nil, "", ErrSubscribedEventNotFound
	}
	if err != nil {
		return uuid.Nil, "", err
	}
	if uid == nil {
		return uuid.Nil, "", ErrEventWithoutUID
	}
	return subscriptionID, *uid, nil
}

// CustomizeSubscribedEvent sets the user's customization of an event of one
// of their subscriptions, replacing any earlier one. It applies to every
// event the feed gives the same UID.
func (s *Store) CustomizeSubscribedEvent(ctx context.Context, userID, eventID uuid.UUID, customization Customization) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		subscriptionID, uid, err := subscribedEventUID(ctx, tx, userID, eventID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO subscription_customizations (subscription_id, event_uid, `+customizationColumns+`)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (subscription_id, event_uid) DO UPDATE SET
				title = EXCLUDED.title, description = EXCLUDED.description, url = EXCLUDED.url,
				location = EXCLUDED.location, note = EXCLUDED.note,
				start_shift_seconds = EXCLUDED.start_shift_seconds, end_shift_seconds = EXCLUDED.end_shift_seconds,
				updated_at = now()
		`, append([]any{subscriptionID, uid}, customizationArgs(customization)...)...)
		return err
	})
}

// UncustomizeSubscribedEvent removes the user's customization, if any, of
// an event of one of their subscriptions.
func (s *Store) UncustomizeSubscribedEvent(ctx context.Context, userID, eventID uuid.UUID) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		subscriptionID, uid, err := subscribedEventUID(ctx, tx, userID, eventID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			DELETE FROM subscription_customizations WHERE subscription_id = $1 AND event_uid = $2
		`, subscriptionID, uid)
		return err
	})
}
