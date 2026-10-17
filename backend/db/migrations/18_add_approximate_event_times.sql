-- +goose Up
-- Approximate edges only change how a timed event is presented; the stored
-- times stay exact. An all-day event has no meaningful approximate midnight.
ALTER TABLE events
	ADD COLUMN start_is_approximate BOOLEAN NOT NULL DEFAULT false,
	ADD COLUMN end_is_approximate BOOLEAN NOT NULL DEFAULT false,
	ADD CONSTRAINT events_approximate_timed CHECK (
		NOT is_all_day OR (NOT start_is_approximate AND NOT end_is_approximate)
	);

-- +goose Down
ALTER TABLE events
	DROP CONSTRAINT events_approximate_timed,
	DROP COLUMN start_is_approximate,
	DROP COLUMN end_is_approximate;
