-- +goose Up
-- A missing row means the user has not reported that day. need = 'no' is an
-- explicit "not needed" answer.
CREATE TABLE transit_card_days (
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	day DATE NOT NULL,
	need TEXT NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (user_id, day),
	CONSTRAINT transit_card_days_need_valid CHECK (need IN ('yes', 'maybe', 'no'))
);

-- Supports future conflict lookups across users.
CREATE INDEX transit_card_days_wanted_day ON transit_card_days (day) WHERE need <> 'no';

-- +goose Down
DROP TABLE transit_card_days;
