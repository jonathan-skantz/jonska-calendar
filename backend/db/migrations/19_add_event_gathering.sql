-- +goose Up
-- Gathering is time spent at the event before it starts (e.g. "warm up 30 min
-- before kickoff"). Like commute it is whole minutes on timed attendable
-- events, but it describes the event itself, so recipients see it too. An
-- optional gathering ("mingle from 17:00") may be skipped; the label is a
-- short name such as "Warm-up".
ALTER TABLE events
	ADD COLUMN gathering_seconds INT NOT NULL DEFAULT 0,
	ADD COLUMN gathering_is_optional BOOLEAN NOT NULL DEFAULT false,
	ADD COLUMN gathering_label TEXT,
	ADD CONSTRAINT events_gathering_timed_attendable CHECK (
		gathering_seconds = 0 OR (kind = 'attendable' AND NOT is_all_day)
	),
	ADD CONSTRAINT events_gathering_whole_minutes CHECK (
		gathering_seconds >= 0 AND gathering_seconds % 60 = 0 AND gathering_seconds <= 86400
	),
	ADD CONSTRAINT events_gathering_details CHECK (
		(gathering_seconds > 0 OR (NOT gathering_is_optional AND gathering_label IS NULL))
		AND (gathering_label IS NULL OR (gathering_label <> '' AND char_length(gathering_label) <= 40))
	);

-- +goose Down
-- IF EXISTS so this also undoes an earlier draft of this migration that had
-- only gathering_seconds.
ALTER TABLE events
	DROP CONSTRAINT IF EXISTS events_gathering_timed_attendable,
	DROP CONSTRAINT IF EXISTS events_gathering_whole_minutes,
	DROP CONSTRAINT IF EXISTS events_gathering_details,
	DROP COLUMN IF EXISTS gathering_seconds,
	DROP COLUMN IF EXISTS gathering_is_optional,
	DROP COLUMN IF EXISTS gathering_label;
