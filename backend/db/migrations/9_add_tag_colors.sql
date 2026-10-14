-- +goose Up
ALTER TABLE tags
	ADD COLUMN color TEXT NOT NULL DEFAULT '#547968',
	ADD CONSTRAINT tags_color_hex CHECK (color ~ '^#[0-9A-Fa-f]{6}$');

-- +goose Down
ALTER TABLE tags
	DROP CONSTRAINT tags_color_hex,
	DROP COLUMN color;
