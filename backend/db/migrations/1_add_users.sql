-- +goose Up
CREATE TABLE users (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	email TEXT NOT NULL UNIQUE,
        CHECK ((email = lower(email))),
	password_hash TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

CREATE TABLE timeline_items (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	
	title TEXT NOT NULL,
	description TEXT,
	url TEXT,

	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE events (
	timeline_item_id UUID PRIMARY KEY REFERENCES timeline_items(id) ON DELETE CASCADE,
	starts_at TIMESTAMPTZ NOT NULL,
	ends_at TIMESTAMPTZ NOT NULL,

	CONSTRAINT events_valid_duration CHECK (ends_at > starts_at)
);

-- +goose Down
DROP TABLE events;
DROP TABLE timeline_items;
DROP TABLE users;