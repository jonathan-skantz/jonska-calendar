-- +goose Up
ALTER TABLE events
	ADD COLUMN is_all_day BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE events DROP COLUMN is_all_day;
