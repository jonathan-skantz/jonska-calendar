-- +goose Up
CREATE TABLE tags (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

	CONSTRAINT tags_name_not_blank CHECK (length(trim(name)) > 0),
	CONSTRAINT tags_user_name_unique UNIQUE (user_id, name)
);

CREATE TABLE timeline_item_tags (
	timeline_item_id UUID NOT NULL REFERENCES timeline_items(id) ON DELETE CASCADE,
	tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
	PRIMARY KEY (timeline_item_id, tag_id)
);

-- +goose Down
DROP TABLE timeline_item_tags;
DROP TABLE tags;