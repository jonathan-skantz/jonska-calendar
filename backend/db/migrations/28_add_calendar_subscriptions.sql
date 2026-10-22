-- +goose Up
-- A subscription is a calendar the user follows by URL. The server fetches
-- it on a schedule; time_zone is the subscriber's browser zone, which floating
-- times and all-day dates in the file are read in.
CREATE TABLE calendar_subscriptions (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL CHECK (name <> ''),
	url TEXT NOT NULL,
	color TEXT NOT NULL CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
	time_zone TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	-- The last attempt, the last success, and the last attempt's error (NULL
	-- when it succeeded). A failed refresh keeps the events of the last
	-- success.
	last_fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	last_success_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	last_error TEXT,
	UNIQUE (user_id, url)
);

-- The events of a subscription as its last successful fetch had them. They
-- are owned through calendar_subscriptions.user_id, are read-only, and are
-- replaced as a whole on every refresh. The columns and checks mirror events
-- without the owner's own fields (tags, commute, skipping).
CREATE TABLE subscription_events (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	subscription_id UUID NOT NULL REFERENCES calendar_subscriptions(id) ON DELETE CASCADE,
	title TEXT NOT NULL,
	description TEXT,
	url TEXT,
	location TEXT,
	starts_at TIMESTAMPTZ NOT NULL,
	ends_at TIMESTAMPTZ NOT NULL,
	is_all_day BOOLEAN NOT NULL DEFAULT false,
	kind TEXT NOT NULL CHECK (kind IN ('attendable', 'informational')),
	start_is_approximate BOOLEAN NOT NULL DEFAULT false,
	end_is_approximate BOOLEAN NOT NULL DEFAULT false,
	gathering_seconds INT NOT NULL DEFAULT 0,
	gathering_is_optional BOOLEAN NOT NULL DEFAULT false,
	gathering_label TEXT,
	recurrence_rule TEXT,
	recurrence_time_zone TEXT,
	recurrence_ends_at TIMESTAMPTZ,
	CHECK (ends_at > starts_at),
	CHECK (NOT is_all_day OR (NOT start_is_approximate AND NOT end_is_approximate)),
	CHECK (gathering_seconds = 0 OR (kind = 'attendable' AND NOT is_all_day)),
	CHECK (gathering_seconds >= 0 AND gathering_seconds % 60 = 0 AND gathering_seconds <= 86400),
	CHECK (
		(gathering_seconds > 0 OR (NOT gathering_is_optional AND gathering_label IS NULL))
		AND (gathering_label IS NULL OR (gathering_label <> '' AND char_length(gathering_label) <= 40))
	),
	CHECK ((recurrence_rule IS NULL) = (recurrence_time_zone IS NULL)),
	CHECK (recurrence_rule IS NOT NULL OR recurrence_ends_at IS NULL)
);

CREATE INDEX subscription_events_subscription_id_idx ON subscription_events (subscription_id);

-- Like event_occurrence_exceptions, without the owner's skip decision.
CREATE TABLE subscription_event_exceptions (
	subscription_event_id UUID NOT NULL REFERENCES subscription_events(id) ON DELETE CASCADE,
	original_start TIMESTAMPTZ NOT NULL,
	is_cancelled BOOLEAN NOT NULL DEFAULT FALSE,
	starts_at TIMESTAMPTZ,
	ends_at TIMESTAMPTZ,
	PRIMARY KEY (subscription_event_id, original_start),
	CHECK ((starts_at IS NULL) = (ends_at IS NULL)),
	CHECK (ends_at IS NULL OR ends_at > starts_at)
);

-- +goose Down
DROP TABLE subscription_event_exceptions;
DROP TABLE subscription_events;
DROP TABLE calendar_subscriptions;
