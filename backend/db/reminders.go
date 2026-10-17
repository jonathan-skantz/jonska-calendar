package db

import (
	"context"
	"time"

	"github.com/google/uuid"
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

// ListReminders returns the user's reminders for events that have not ended.
// Recipients see their per-recipient title and location, and never the
// owner's skip decision.
func (s *Store) ListReminders(ctx context.Context, userID uuid.UUID) ([]Reminder, error) {
	rows, err := s.db.Query(ctx, `
		SELECT reminder.id, reminder.event_id, reminder.offset_seconds,
			e.starts_at - make_interval(secs => reminder.offset_seconds),
			reminder.notified_at,
			COALESCE(share.title, item.title), COALESCE(share.location, e.location),
			e.starts_at, e.ends_at, e.is_all_day,
			item.user_id = $1 AND e.is_skipped,
			share.id IS NOT NULL
		FROM event_reminders AS reminder
		JOIN events AS e ON e.timeline_item_id = reminder.event_id
		JOIN timeline_items AS item ON item.id = reminder.event_id
		LEFT JOIN event_shares AS share
			ON share.event_id = reminder.event_id AND share.recipient_user_id = $1
		WHERE reminder.user_id = $1
		  AND e.ends_at > now()
		  AND (item.user_id = $1 OR share.id IS NOT NULL)
		ORDER BY 4, reminder.id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reminders := []Reminder{}
	for rows.Next() {
		var reminder Reminder
		if err := rows.Scan(
			&reminder.ID, &reminder.EventID, &reminder.OffsetSeconds,
			&reminder.RemindAt, &reminder.NotifiedAt,
			&reminder.EventTitle, &reminder.EventLocation,
			&reminder.EventStartsAt, &reminder.EventEndsAt, &reminder.EventIsAllDay,
			&reminder.EventIsSkipped, &reminder.IsShared,
		); err != nil {
			return nil, err
		}
		reminders = append(reminders, reminder)
	}
	return reminders, rows.Err()
}

func (s *Store) DeleteReminder(ctx context.Context, userID, reminderID uuid.UUID) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM event_reminders WHERE id = $2 AND user_id = $1
	`, userID, reminderID)
	if err == nil && result.RowsAffected() == 0 {
		return ErrReminderNotFound
	}
	return err
}

// MarkReminderNotified is idempotent: the first delivery time is kept.
func (s *Store) MarkReminderNotified(ctx context.Context, userID, reminderID uuid.UUID, at time.Time) error {
	result, err := s.db.Exec(ctx, `
		UPDATE event_reminders
		SET notified_at = COALESCE(notified_at, $3)
		WHERE id = $2 AND user_id = $1
	`, userID, reminderID, at)
	if err == nil && result.RowsAffected() == 0 {
		return ErrReminderNotFound
	}
	return err
}
