package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Reminders belong to the user who set them and are never shared. A user may
// only remind themselves of events they can see: their own, or one shared
// with them.

const canSeeEventSQL = `(
	item.user_id = $1
	OR EXISTS (
		SELECT 1 FROM event_shares AS share
		WHERE share.event_id = item.id AND share.recipient_user_id = $1
	)
)`

func (s *Store) CreateReminder(ctx context.Context, userID, eventID uuid.UUID, offsetSeconds int) (uuid.UUID, error) {
	var reminderID uuid.UUID
	err := s.db.QueryRow(ctx, `
		INSERT INTO event_reminders (event_id, user_id, offset_seconds)
		SELECT e.timeline_item_id, $1, $3
		FROM events AS e
		JOIN timeline_items AS item ON item.id = e.timeline_item_id
		WHERE e.timeline_item_id = $2 AND `+canSeeEventSQL+`
		RETURNING id
	`, userID, eventID, offsetSeconds).Scan(&reminderID)
	if isUniqueViolation(err) {
		return uuid.Nil, ErrReminderExists
	}
	if isNoRows(err) {
		return uuid.Nil, ErrEventNotFound
	}
	return reminderID, err
}

// ListReminders returns the user's reminders for events that have not ended,
// and for series that may still have occurrences. Recipients see their
// per-recipient title and location, and never the owner's skip decision.
// A recipient's customization may shift a shared event by up to
// MaxShiftSeconds, so shared events that ended that recently are listed
// too, for the caller to shift and filter.
func (s *Store) ListReminders(ctx context.Context, userID uuid.UUID) ([]Reminder, error) {
	rows, err := s.db.Query(ctx, `
		SELECT reminder.id, reminder.event_id, reminder.offset_seconds,
			e.starts_at - make_interval(secs => reminder.offset_seconds),
			e.starts_at, delivery.notified_at,
			COALESCE(share.title, item.title), COALESCE(share.location, e.location),
			e.starts_at, e.ends_at, e.is_all_day,
			item.user_id = $1 AND e.is_skipped,
			share.id IS NOT NULL,
			e.recurrence_rule, e.recurrence_time_zone,
			`+customizationSQL+`
		FROM event_reminders AS reminder
		JOIN events AS e ON e.timeline_item_id = reminder.event_id
		JOIN timeline_items AS item ON item.id = reminder.event_id
		LEFT JOIN event_shares AS share
			ON share.event_id = reminder.event_id AND share.recipient_user_id = $1
		LEFT JOIN share_customizations AS custom ON custom.event_share_id = share.id
		LEFT JOIN reminder_deliveries AS delivery
			ON delivery.reminder_id = reminder.id AND delivery.occurrence_start = e.starts_at
		WHERE reminder.user_id = $1
		  AND CASE WHEN e.recurrence_rule IS NULL THEN e.ends_at
			ELSE COALESCE(e.recurrence_ends_at, 'infinity') END
			> now() - CASE WHEN share.id IS NULL THEN interval '0' ELSE make_interval(secs => $2) END
		  AND (item.user_id = $1 OR share.id IS NOT NULL)
		ORDER BY 4, reminder.id
	`, userID, MaxShiftSeconds)
	if err != nil {
		return nil, err
	}
	reminders, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Reminder, error) {
		var reminder Reminder
		var customization scannedCustomization
		err := row.Scan(append([]any{
			&reminder.ID, &reminder.EventID, &reminder.OffsetSeconds,
			&reminder.RemindAt, &reminder.OccurrenceStart, &reminder.NotifiedAt,
			&reminder.EventTitle, &reminder.EventLocation,
			&reminder.EventStartsAt, &reminder.EventEndsAt, &reminder.EventIsAllDay,
			&reminder.EventIsSkipped, &reminder.IsShared,
			&reminder.RecurrenceRule, &reminder.RecurrenceTimeZone,
		}, customization.targets()...)...)
		reminder.Customization = customization.value()
		return reminder, err
	})
	if err != nil {
		return nil, err
	}
	return reminders, s.populateReminderDeliveries(ctx, userID, reminders)
}

// populateReminderDeliveries attaches the exceptions of the reminders of
// repeating events, and the deliveries of those and of customized ones,
// whose occurrences the caller works out.
func (s *Store) populateReminderDeliveries(ctx context.Context, userID uuid.UUID, reminders []Reminder) error {
	var eventIDs, reminderIDs []uuid.UUID
	for _, reminder := range reminders {
		if reminder.RecurrenceRule != nil {
			eventIDs = append(eventIDs, reminder.EventID)
		}
		if reminder.RecurrenceRule != nil || reminder.Customization != nil {
			reminderIDs = append(reminderIDs, reminder.ID)
		}
	}
	if len(reminderIDs) == 0 {
		return nil
	}
	exceptions, err := s.exceptionsByEvent(ctx, userID, eventIDs)
	if err != nil {
		return err
	}
	rows, err := s.db.Query(ctx, `
		SELECT reminder_id, occurrence_start, notified_at
		FROM reminder_deliveries WHERE reminder_id = ANY($1)
	`, reminderIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	deliveries := make(map[uuid.UUID]map[time.Time]time.Time)
	for rows.Next() {
		var reminderID uuid.UUID
		var occurrenceStart, notifiedAt time.Time
		if err := rows.Scan(&reminderID, &occurrenceStart, &notifiedAt); err != nil {
			return err
		}
		if deliveries[reminderID] == nil {
			deliveries[reminderID] = make(map[time.Time]time.Time)
		}
		deliveries[reminderID][occurrenceStart.UTC()] = notifiedAt
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for index, reminder := range reminders {
		reminders[index].Exceptions = exceptions[reminder.EventID]
		reminders[index].Deliveries = deliveries[reminder.ID]
	}
	return nil
}

func (s *Store) DeleteReminder(ctx context.Context, userID, reminderID uuid.UUID) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM event_reminders WHERE id = $2 AND user_id = $1
	`, userID, reminderID)
	return requireRowsAffected(result, err, ErrReminderNotFound)
}

// MarkReminderNotified records that the reminder was delivered for the
// occurrence that starts at occurrenceStart, or for the event's own start
// when it is nil. It is idempotent: the first delivery time is kept.
func (s *Store) MarkReminderNotified(ctx context.Context, userID, reminderID uuid.UUID, occurrenceStart *time.Time, at time.Time) error {
	var eventStart time.Time
	if err := s.db.QueryRow(ctx, `
		SELECT e.starts_at
		FROM event_reminders AS reminder
		JOIN events AS e ON e.timeline_item_id = reminder.event_id
		WHERE reminder.id = $2 AND reminder.user_id = $1
	`, userID, reminderID).Scan(&eventStart); err != nil {
		if isNoRows(err) {
			return ErrReminderNotFound
		}
		return err
	}
	if occurrenceStart == nil {
		occurrenceStart = &eventStart
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO reminder_deliveries (reminder_id, occurrence_start, notified_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, reminderID, occurrenceStart, at)
	return err
}
