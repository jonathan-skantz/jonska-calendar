-- +goose Up
-- Every timeline read filters timeline_items by owner and event_shares by
-- recipient; neither had an index that serves those lookups. Deleting a tag
-- cascades to timeline_item_tags by tag_id, which leads no index either.
CREATE INDEX timeline_items_user_id_idx ON timeline_items (user_id);
CREATE INDEX event_shares_recipient_user_id_idx ON event_shares (recipient_user_id);
CREATE INDEX timeline_item_tags_tag_id_idx ON timeline_item_tags (tag_id);

-- +goose Down
DROP INDEX timeline_item_tags_tag_id_idx;
DROP INDEX event_shares_recipient_user_id_idx;
DROP INDEX timeline_items_user_id_idx;
