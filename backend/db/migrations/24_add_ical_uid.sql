-- +goose Up
-- The iCalendar UID an item was imported with, so importing the same file
-- again skips the items it already created. NULL for items made in the app.
-- It lives on timeline_items because the uniqueness is per owner.
ALTER TABLE timeline_items ADD COLUMN ical_uid TEXT;

CREATE UNIQUE INDEX timeline_items_user_ical_uid
	ON timeline_items (user_id, ical_uid)
	WHERE ical_uid IS NOT NULL;

-- +goose Down
DROP INDEX timeline_items_user_ical_uid;
ALTER TABLE timeline_items DROP COLUMN ical_uid;
