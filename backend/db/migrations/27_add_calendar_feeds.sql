-- +goose Up
-- A feed publishes the user's own events at a secret URL that other calendar
-- apps subscribe to. The token is the only credential, so it is long and
-- random; it is stored as is because the URL must be shown again, and the
-- events it serves are in the same database. A feed limited to a tag goes
-- with the tag, so it never widens to every event.
CREATE TABLE calendar_feeds (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL CHECK (name <> ''),
	token TEXT NOT NULL UNIQUE,
	tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
	include_alarms BOOLEAN NOT NULL DEFAULT false,
	time_zone TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX calendar_feeds_user_id_idx ON calendar_feeds (user_id);

-- +goose Down
DROP TABLE calendar_feeds;
