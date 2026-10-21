-- +goose Up
-- A repeating event is an ordinary event with a rule: starts_at and ends_at
-- stay its first occurrence. recurrence_rule is an RFC 5545 RRULE value, and
-- the wall-clock times repeat in recurrence_time_zone. recurrence_ends_at is
-- when the last occurrence ends, NULL when the series never ends, so a range
-- query can find the series that overlap it without expanding them.
ALTER TABLE events
	ADD COLUMN recurrence_rule TEXT,
	ADD COLUMN recurrence_time_zone TEXT,
	ADD COLUMN recurrence_ends_at TIMESTAMPTZ,
	ADD CONSTRAINT events_recurrence_complete
		CHECK ((recurrence_rule IS NULL) = (recurrence_time_zone IS NULL)),
	ADD CONSTRAINT events_recurrence_ends_only_on_series
		CHECK (recurrence_rule IS NOT NULL OR recurrence_ends_at IS NULL);

-- An occurrence that differs from its series, keyed by the start the rule
-- gives it (an iCalendar RECURRENCE-ID). It is owned through
-- events -> timeline_items.user_id, like the event. Only the time, the
-- owner's skip decision, and whether it happens at all can differ.
CREATE TABLE event_occurrence_exceptions (
	event_id UUID NOT NULL REFERENCES events(timeline_item_id) ON DELETE CASCADE,
	original_start TIMESTAMPTZ NOT NULL,
	is_cancelled BOOLEAN NOT NULL DEFAULT FALSE,
	starts_at TIMESTAMPTZ,
	ends_at TIMESTAMPTZ,
	is_skipped BOOLEAN,
	PRIMARY KEY (event_id, original_start),
	CONSTRAINT event_occurrence_exceptions_times_together
		CHECK ((starts_at IS NULL) = (ends_at IS NULL)),
	CONSTRAINT event_occurrence_exceptions_end_after_start
		CHECK (ends_at IS NULL OR ends_at > starts_at)
);

-- +goose Down
DROP TABLE event_occurrence_exceptions;
ALTER TABLE events
	DROP CONSTRAINT events_recurrence_ends_only_on_series,
	DROP CONSTRAINT events_recurrence_complete,
	DROP COLUMN recurrence_ends_at,
	DROP COLUMN recurrence_time_zone,
	DROP COLUMN recurrence_rule;
