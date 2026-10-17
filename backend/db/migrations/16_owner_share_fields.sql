-- +goose Up
-- The owner may present a shared event differently to each recipient. NULL
-- means "use the event's own value", so later edits to the event still reach
-- that recipient. Recipients can no longer override anything themselves.
ALTER TABLE event_shares
	ADD COLUMN title TEXT,
	ADD COLUMN description TEXT,
	ADD COLUMN url TEXT,
	ADD COLUMN location TEXT,
	ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	ADD CONSTRAINT event_shares_title_not_empty CHECK (title IS NULL OR title <> '');

DROP TABLE event_share_overrides;

-- +goose Down
CREATE TABLE event_share_overrides (
	event_share_id UUID PRIMARY KEY REFERENCES event_shares(id) ON DELETE CASCADE,
	title TEXT,
	description TEXT,
	url TEXT,
	location TEXT,
	custom_info TEXT,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE event_shares
	DROP CONSTRAINT event_shares_title_not_empty,
	DROP COLUMN title,
	DROP COLUMN description,
	DROP COLUMN url,
	DROP COLUMN location,
	DROP COLUMN updated_at;
