-- +goose Up
-- A refresh replaces a subscription's events with new rows, so the feed's
-- UID is what recognizes an event from one refresh to the next. It is NULL
-- when the feed gave none. It is not unique: a feed may repeat a UID.
ALTER TABLE subscription_events ADD COLUMN uid TEXT CHECK (uid <> '');

-- A recipient's own changes to an event they receive: shared with them, or
-- from a calendar they subscribe to. They apply on top of what they receive
-- and are never shown to the owner or the publisher. A NULL field and a zero
-- shift follow the received value, so later changes at the source still
-- show. The recipient is derived through the share or the subscription.
CREATE TABLE share_customizations (
	event_share_id UUID PRIMARY KEY REFERENCES event_shares(id) ON DELETE CASCADE,
	title TEXT CHECK (title <> ''),
	description TEXT CHECK (description <> ''),
	url TEXT CHECK (url <> ''),
	location TEXT CHECK (location <> ''),
	note TEXT CHECK (note <> ''),
	start_shift_seconds INT NOT NULL DEFAULT 0
		CHECK (start_shift_seconds BETWEEN -86400 AND 86400 AND start_shift_seconds % 60 = 0),
	end_shift_seconds INT NOT NULL DEFAULT 0
		CHECK (end_shift_seconds BETWEEN -86400 AND 86400 AND end_shift_seconds % 60 = 0),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Keyed by the feed's UID, so the changes survive refreshes. An event of a
-- feed without UIDs cannot be customized.
CREATE TABLE subscription_customizations (
	subscription_id UUID NOT NULL REFERENCES calendar_subscriptions(id) ON DELETE CASCADE,
	event_uid TEXT NOT NULL,
	title TEXT CHECK (title <> ''),
	description TEXT CHECK (description <> ''),
	url TEXT CHECK (url <> ''),
	location TEXT CHECK (location <> ''),
	note TEXT CHECK (note <> ''),
	start_shift_seconds INT NOT NULL DEFAULT 0
		CHECK (start_shift_seconds BETWEEN -86400 AND 86400 AND start_shift_seconds % 60 = 0),
	end_shift_seconds INT NOT NULL DEFAULT 0
		CHECK (end_shift_seconds BETWEEN -86400 AND 86400 AND end_shift_seconds % 60 = 0),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (subscription_id, event_uid)
);

-- +goose Down
DROP TABLE subscription_customizations;
DROP TABLE share_customizations;
ALTER TABLE subscription_events DROP COLUMN uid;
