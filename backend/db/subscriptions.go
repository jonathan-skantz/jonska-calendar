package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Subscription is a calendar the user follows by URL.
type Subscription struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Name     string
	URL      string
	Color    string
	TimeZone string
	// LastError is the last refresh's error, nil when it succeeded.
	LastFetchedAt time.Time
	LastSuccessAt time.Time
	LastError     *string
}

// SubscribedEvent is one event of a subscribed calendar, validated, with
// the occurrences of a repeating one that differ from it. Owner-only fields
// (tags, commute, skipping) are never set.
type SubscribedEvent struct {
	Event
	// UID is the feed's UID, nil when it gave none.
	UID        *string
	Exceptions []OccurrenceException
}

// SubscriptionRef marks a timeline entry as an event of a subscription.
type SubscriptionRef struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Color string    `json:"color"`
}

const subscriptionColumns = `id, user_id, name, url, color, time_zone, last_fetched_at, last_success_at, last_error`

func scanSubscription(row pgx.CollectableRow) (Subscription, error) {
	var subscription Subscription
	err := row.Scan(&subscription.ID, &subscription.UserID, &subscription.Name, &subscription.URL,
		&subscription.Color, &subscription.TimeZone, &subscription.LastFetchedAt,
		&subscription.LastSuccessAt, &subscription.LastError)
	return subscription, err
}

func collectSubscription(rows pgx.Rows, err error) (Subscription, error) {
	if err != nil {
		return Subscription{}, err
	}
	subscription, err := pgx.CollectExactlyOneRow(rows, scanSubscription)
	if isNoRows(err) {
		return Subscription{}, ErrSubscriptionNotFound
	}
	return subscription, err
}

// CreateSubscription saves a subscription with the events its first fetch
// found, in one transaction. A URL the user already follows is
// ErrSubscriptionExists.
func (s *Store) CreateSubscription(ctx context.Context, subscription Subscription, events []SubscribedEvent) (Subscription, error) {
	var created Subscription
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			INSERT INTO calendar_subscriptions (user_id, name, url, color, time_zone)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING `+subscriptionColumns,
			subscription.UserID, subscription.Name, subscription.URL, subscription.Color, subscription.TimeZone)
		created, err = collectSubscription(rows, err)
		if isUniqueViolation(err) {
			return ErrSubscriptionExists
		}
		if err != nil {
			return err
		}
		return insertSubscribedEvents(ctx, tx, created.ID, events)
	})
	return created, err
}

func (s *Store) ListSubscriptions(ctx context.Context, userID uuid.UUID) ([]Subscription, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+subscriptionColumns+` FROM calendar_subscriptions
		WHERE user_id = $1
		ORDER BY name, id
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanSubscription)
}

func (s *Store) GetSubscription(ctx context.Context, userID, subscriptionID uuid.UUID) (Subscription, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+subscriptionColumns+` FROM calendar_subscriptions
		WHERE id = $1 AND user_id = $2
	`, subscriptionID, userID)
	return collectSubscription(rows, err)
}

// UpdateSubscription renames and recolors a subscription.
func (s *Store) UpdateSubscription(ctx context.Context, userID, subscriptionID uuid.UUID, name, color string) (Subscription, error) {
	rows, err := s.db.Query(ctx, `
		UPDATE calendar_subscriptions SET name = $3, color = $4
		WHERE id = $1 AND user_id = $2
		RETURNING `+subscriptionColumns,
		subscriptionID, userID, name, color)
	return collectSubscription(rows, err)
}

// DeleteSubscription removes a subscription and its events.
func (s *Store) DeleteSubscription(ctx context.Context, userID, subscriptionID uuid.UUID) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM calendar_subscriptions WHERE id = $1 AND user_id = $2
	`, subscriptionID, userID)
	return requireRowsAffected(result, err, ErrSubscriptionNotFound)
}

// StaleSubscriptions returns every user's subscriptions last fetched before
// fetchedBefore, oldest first. It is for the background refresh, which acts
// for each subscription's owner.
func (s *Store) StaleSubscriptions(ctx context.Context, fetchedBefore time.Time) ([]Subscription, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+subscriptionColumns+` FROM calendar_subscriptions
		WHERE last_fetched_at < $1
		ORDER BY last_fetched_at
	`, fetchedBefore)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanSubscription)
}

// ReplaceSubscriptionEvents records a successful refresh: the subscription's
// events become the given ones, in one transaction.
func (s *Store) ReplaceSubscriptionEvents(ctx context.Context, userID, subscriptionID uuid.UUID, events []SubscribedEvent) (Subscription, error) {
	var updated Subscription
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		// Updating the subscription first locks it, so two refreshes of one
		// subscription take turns.
		rows, err := tx.Query(ctx, `
			UPDATE calendar_subscriptions
			SET last_fetched_at = now(), last_success_at = now(), last_error = NULL
			WHERE id = $1 AND user_id = $2
			RETURNING `+subscriptionColumns,
			subscriptionID, userID)
		if updated, err = collectSubscription(rows, err); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM subscription_events WHERE subscription_id = $1`, subscriptionID); err != nil {
			return err
		}
		return insertSubscribedEvents(ctx, tx, subscriptionID, events)
	})
	return updated, err
}

// MarkSubscriptionFailed records a failed refresh. The events of the last
// successful one stay.
func (s *Store) MarkSubscriptionFailed(ctx context.Context, userID, subscriptionID uuid.UUID, message string) (Subscription, error) {
	rows, err := s.db.Query(ctx, `
		UPDATE calendar_subscriptions SET last_fetched_at = now(), last_error = $3
		WHERE id = $1 AND user_id = $2
		RETURNING `+subscriptionColumns,
		subscriptionID, userID, message)
	return collectSubscription(rows, err)
}

// insertSubscribedEvents copies the events and their exceptions in. A file
// holds up to thousands of events, so they are copied rather than inserted
// one by one; their IDs are made here to link the exceptions.
func insertSubscribedEvents(ctx context.Context, tx pgx.Tx, subscriptionID uuid.UUID, events []SubscribedEvent) error {
	eventRows := make([][]any, 0, len(events))
	var exceptionRows [][]any
	for _, event := range events {
		eventID := uuid.New()
		eventRows = append(eventRows, []any{
			eventID, subscriptionID, event.Title, event.Description, event.URL, event.Location,
			event.StartsAt, event.EndsAt, event.IsAllDay, event.Kind,
			event.StartIsApproximate, event.EndIsApproximate,
			event.GatheringSeconds, event.GatheringIsOptional, event.GatheringLabel,
			event.RecurrenceRule, event.RecurrenceTimeZone, event.RecurrenceEndsAt, event.UID,
		})
		for _, exception := range event.Exceptions {
			exceptionRows = append(exceptionRows, []any{
				eventID, exception.OriginalStart, exception.IsCancelled, exception.StartsAt, exception.EndsAt,
			})
		}
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"subscription_events"}, []string{
		"id", "subscription_id", "title", "description", "url", "location",
		"starts_at", "ends_at", "is_all_day", "kind",
		"start_is_approximate", "end_is_approximate",
		"gathering_seconds", "gathering_is_optional", "gathering_label",
		"recurrence_rule", "recurrence_time_zone", "recurrence_ends_at", "uid",
	}, pgx.CopyFromRows(eventRows)); err != nil {
		return err
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"subscription_event_exceptions"}, []string{
		"subscription_event_id", "original_start", "is_cancelled", "starts_at", "ends_at",
	}, pgx.CopyFromRows(exceptionRows))
	return err
}

// listSubscribedTimeline returns the events of the user's subscriptions that
// may happen in the window, with their exceptions. They are read-only and
// carry the subscription instead of tags.
func (s *Store) listSubscribedTimeline(ctx context.Context, userID uuid.UUID, window *timeWindow) ([]TimelineEntry, error) {
	from, to := window.bounds()
	rows, err := s.db.Query(ctx, `
		SELECT event.id, event.title, event.description, event.url,
			event.starts_at, event.ends_at, event.is_all_day, event.kind, event.location,
			event.start_is_approximate, event.end_is_approximate, event.gathering_seconds,
			event.gathering_is_optional, event.gathering_label,
			subscription.id, subscription.name, subscription.color,
			event.recurrence_rule, event.recurrence_time_zone, event.recurrence_ends_at,
			`+customizationSQL+`
		FROM subscription_events AS event
		JOIN calendar_subscriptions AS subscription ON subscription.id = event.subscription_id
		LEFT JOIN subscription_customizations AS custom
			ON custom.subscription_id = event.subscription_id AND custom.event_uid = event.uid
		WHERE subscription.user_id = $1 AND `+subscribedInWindowSQL+`
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	entries, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (TimelineEntry, error) {
		entry := TimelineEntry{Type: "event", Subscription: &SubscriptionRef{}}
		var customization scannedCustomization
		err := row.Scan(append([]any{
			&entry.ID, &entry.Title, &entry.Description, &entry.URL,
			&entry.StartsAt, &entry.EndsAt, &entry.IsAllDay, &entry.Kind, &entry.Location,
			&entry.StartIsApproximate, &entry.EndIsApproximate, &entry.GatheringSeconds,
			&entry.GatheringIsOptional, &entry.GatheringLabel,
			&entry.Subscription.ID, &entry.Subscription.Name, &entry.Subscription.Color,
			&entry.RecurrenceRule, &entry.RecurrenceTimeZone, &entry.RecurrenceEndsAt,
		}, customization.targets()...)...)
		entry.Customization = customization.value()
		return entry, err
	})
	if err != nil {
		return nil, err
	}
	return entries, s.populateSubscribedExceptions(ctx, userID, entries)
}

func (s *Store) populateSubscribedExceptions(ctx context.Context, userID uuid.UUID, entries []TimelineEntry) error {
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
		SELECT exception.subscription_event_id, exception.original_start, exception.is_cancelled,
			exception.starts_at, exception.ends_at
		FROM subscription_event_exceptions AS exception
		JOIN subscription_events AS event ON event.id = exception.subscription_event_id
		JOIN calendar_subscriptions AS subscription ON subscription.id = event.subscription_id
		WHERE exception.subscription_event_id = ANY($2) AND subscription.user_id = $1
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
			&exception.StartsAt, &exception.EndsAt); err != nil {
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
