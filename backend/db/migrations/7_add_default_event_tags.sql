-- +goose Up
ALTER TABLE users
	ADD COLUMN default_tag_id UUID;

INSERT INTO tags (user_id, name)
SELECT id, 'general'
FROM users
ON CONFLICT (user_id, name) DO NOTHING;

INSERT INTO timeline_item_tags (timeline_item_id, tag_id)
SELECT item.id, tag.id
FROM timeline_items AS item
JOIN tags AS tag ON tag.user_id = item.user_id AND tag.name = 'general'
JOIN events AS event ON event.timeline_item_id = item.id
ON CONFLICT DO NOTHING;

UPDATE users
SET default_tag_id = tag.id
FROM tags AS tag
WHERE tag.user_id = users.id AND tag.name = 'general';

ALTER TABLE users
	ADD CONSTRAINT users_default_tag_fk
	FOREIGN KEY (default_tag_id) REFERENCES tags(id) ON DELETE RESTRICT;

ALTER TABLE users
	ALTER COLUMN default_tag_id SET NOT NULL;

-- +goose Down
ALTER TABLE users
	ALTER COLUMN default_tag_id DROP NOT NULL;

ALTER TABLE users
	DROP CONSTRAINT users_default_tag_fk;

ALTER TABLE users
	DROP COLUMN default_tag_id;