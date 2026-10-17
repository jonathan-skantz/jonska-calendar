-- +goose Up
-- Reminders are private to the user who set them: an owner or a recipient of
-- a shared event. They are an offset before the event starts, so they move
-- with the event. notified_at records delivery so other tabs and devices do
-- not repeat a reminder.
CREATE TABLE event_reminders (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	event_id UUID NOT NULL REFERENCES events(timeline_item_id) ON DELETE CASCADE,
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	offset_seconds INT NOT NULL CHECK (
		offset_seconds >= 0 AND offset_seconds <= 30 * 86400 AND offset_seconds % 60 = 0
	),
	notified_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (event_id, user_id, offset_seconds)
);

CREATE INDEX event_reminders_user_id ON event_reminders(user_id);

-- +goose Down
DROP TABLE event_reminders;
