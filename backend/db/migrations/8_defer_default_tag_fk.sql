-- +goose Up
ALTER TABLE users
	DROP CONSTRAINT users_default_tag_fk,
	ADD CONSTRAINT users_default_tag_fk
	FOREIGN KEY (default_tag_id) REFERENCES tags(id)
	ON DELETE RESTRICT
	DEFERRABLE INITIALLY DEFERRED;

-- +goose Down
ALTER TABLE users
	DROP CONSTRAINT users_default_tag_fk,
	ADD CONSTRAINT users_default_tag_fk
	FOREIGN KEY (default_tag_id) REFERENCES tags(id)
	ON DELETE RESTRICT;
