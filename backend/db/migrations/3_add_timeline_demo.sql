-- +goose Up
ALTER TABLE events
	ADD COLUMN kind TEXT NOT NULL DEFAULT 'informational',
	ADD COLUMN location TEXT,
	ADD COLUMN commute_before_seconds INTEGER NOT NULL DEFAULT 0,
	ADD COLUMN commute_after_seconds INTEGER NOT NULL DEFAULT 0,
	ADD CONSTRAINT events_kind_valid CHECK (kind IN ('attendable', 'informational')),
	ADD CONSTRAINT events_commute_before_valid CHECK (commute_before_seconds >= 0),
	ADD CONSTRAINT events_commute_after_valid CHECK (commute_after_seconds >= 0);

CREATE TABLE tasks (
	timeline_item_id UUID PRIMARY KEY REFERENCES timeline_items(id) ON DELETE CASCADE,
	deadline_at TIMESTAMPTZ NOT NULL,
	is_completed BOOLEAN NOT NULL DEFAULT FALSE
);

-- +goose Down
DROP TABLE tasks;
ALTER TABLE events
	DROP CONSTRAINT events_kind_valid,
	DROP CONSTRAINT events_commute_before_valid,
	DROP CONSTRAINT events_commute_after_valid,
	DROP COLUMN kind,
	DROP COLUMN location,
	DROP COLUMN commute_before_seconds,
	DROP COLUMN commute_after_seconds;