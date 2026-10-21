-- +goose Up
-- A reminder is delivered once per occurrence of its event: a plain event
-- has one occurrence, at its start. Deliveries are keyed by the start the
-- occurrence has, so moving an event or an occurrence re-arms its reminders.
CREATE TABLE reminder_deliveries (
	reminder_id UUID NOT NULL REFERENCES event_reminders(id) ON DELETE CASCADE,
	occurrence_start TIMESTAMPTZ NOT NULL,
	notified_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (reminder_id, occurrence_start)
);

INSERT INTO reminder_deliveries (reminder_id, occurrence_start, notified_at)
SELECT reminder.id, event.starts_at, reminder.notified_at
FROM event_reminders AS reminder
JOIN events AS event ON event.timeline_item_id = reminder.event_id
WHERE reminder.notified_at IS NOT NULL;

ALTER TABLE event_reminders DROP COLUMN notified_at;

-- +goose Down
ALTER TABLE event_reminders ADD COLUMN notified_at TIMESTAMPTZ;

UPDATE event_reminders AS reminder
SET notified_at = delivery.notified_at
FROM reminder_deliveries AS delivery, events AS event
WHERE delivery.reminder_id = reminder.id
  AND event.timeline_item_id = reminder.event_id
  AND delivery.occurrence_start = event.starts_at;

DROP TABLE reminder_deliveries;
