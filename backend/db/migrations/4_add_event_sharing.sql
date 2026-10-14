-- +goose Up
CREATE TABLE event_shares (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	event_id UUID NOT NULL REFERENCES events(timeline_item_id) ON DELETE CASCADE,
	recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

	CONSTRAINT event_shares_unique_recipient UNIQUE (event_id, recipient_user_id)
);

CREATE TABLE event_share_overrides (
	event_share_id UUID PRIMARY KEY REFERENCES event_shares(id) ON DELETE CASCADE,
	title TEXT,
	description TEXT,
	url TEXT,
	location TEXT,
	custom_info TEXT,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE event_share_overrides;
DROP TABLE event_shares;