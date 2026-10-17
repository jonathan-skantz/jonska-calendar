-- +goose Up
-- The owner can skip an event they would normally attend (e.g. one lecture)
-- while keeping it in the calendar. It stays attendable and keeps its commute
-- and gathering; skipping only means it no longer takes up the owner's time.
ALTER TABLE events
	ADD COLUMN is_skipped BOOLEAN NOT NULL DEFAULT false,
	ADD CONSTRAINT events_skipped_attendable CHECK (NOT is_skipped OR kind = 'attendable');

-- +goose Down
ALTER TABLE events
	DROP CONSTRAINT events_skipped_attendable,
	DROP COLUMN is_skipped;
