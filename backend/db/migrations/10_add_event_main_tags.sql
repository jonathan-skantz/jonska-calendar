-- +goose Up
ALTER TABLE events ADD COLUMN main_tag_id UUID;

UPDATE events AS event
SET main_tag_id = item_tag.tag_id
FROM timeline_item_tags AS item_tag
WHERE item_tag.timeline_item_id = event.timeline_item_id;

UPDATE events AS event
SET main_tag_id = users.default_tag_id
FROM timeline_items AS item, users
WHERE event.timeline_item_id = item.id
  AND item.user_id = users.id
  AND event.main_tag_id IS NULL;

ALTER TABLE events
	ALTER COLUMN main_tag_id SET NOT NULL,
	ADD CONSTRAINT events_main_tag_fk FOREIGN KEY (main_tag_id) REFERENCES tags(id);

-- +goose Down
ALTER TABLE events
	DROP CONSTRAINT events_main_tag_fk,
	DROP COLUMN main_tag_id;
