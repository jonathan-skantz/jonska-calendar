-- +goose Up
-- Commute only makes sense for timed attendable events, in whole minutes.
-- Clean up rows written before the API enforced this, then let the database
-- enforce it too.
UPDATE events
SET commute_before_seconds = 0, commute_after_seconds = 0
WHERE kind <> 'attendable' OR is_all_day;

UPDATE events
SET
	commute_before_seconds = LEAST(ROUND(commute_before_seconds / 60.0) * 60, 86400),
	commute_after_seconds = LEAST(ROUND(commute_after_seconds / 60.0) * 60, 86400)
WHERE commute_before_seconds % 60 <> 0 OR commute_after_seconds % 60 <> 0
	OR commute_before_seconds > 86400 OR commute_after_seconds > 86400;

ALTER TABLE events
	ADD CONSTRAINT events_commute_timed_attendable CHECK (
		(commute_before_seconds = 0 AND commute_after_seconds = 0)
		OR (kind = 'attendable' AND NOT is_all_day)
	),
	ADD CONSTRAINT events_commute_whole_minutes CHECK (
		commute_before_seconds % 60 = 0 AND commute_after_seconds % 60 = 0
		AND commute_before_seconds <= 86400 AND commute_after_seconds <= 86400
	);

-- +goose Down
ALTER TABLE events
	DROP CONSTRAINT events_commute_timed_attendable,
	DROP CONSTRAINT events_commute_whole_minutes;
