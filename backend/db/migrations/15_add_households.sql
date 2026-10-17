-- +goose Up
-- A household is a family or group that shares transit cards. A user belongs
-- to at most one household; transit-card declarations stay keyed by user, and
-- the household is derived through membership.
CREATE TABLE households (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name TEXT NOT NULL,
	transit_card_count INTEGER NOT NULL DEFAULT 1,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	CONSTRAINT households_name_valid CHECK (name <> ''),
	CONSTRAINT households_transit_card_count_valid CHECK (transit_card_count BETWEEN 1 AND 20)
);

CREATE TABLE household_members (
	household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
	user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
	joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (household_id, user_id)
);

-- +goose Down
DROP TABLE household_members;
DROP TABLE households;
