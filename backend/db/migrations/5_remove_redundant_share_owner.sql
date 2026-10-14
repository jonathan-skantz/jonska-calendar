-- +goose Up
ALTER TABLE event_shares
	DROP CONSTRAINT IF EXISTS event_shares_different_users,
	DROP COLUMN IF EXISTS owner_user_id;

-- +goose Down
ALTER TABLE event_shares
	ADD COLUMN owner_user_id UUID REFERENCES users(id) ON DELETE CASCADE;

UPDATE event_shares AS share
SET owner_user_id = item.user_id
FROM timeline_items AS item
WHERE item.id = share.event_id;

ALTER TABLE event_shares
	ALTER COLUMN owner_user_id SET NOT NULL;

ALTER TABLE event_shares
	ADD CONSTRAINT event_shares_different_users CHECK (owner_user_id <> recipient_user_id);