-- +goose Up
-- The name shown for a user everywhere: their display name, or the email
-- local-part when it is not set. Queries call this instead of repeating it.
CREATE FUNCTION user_display_name(display_name TEXT, email TEXT) RETURNS TEXT
	LANGUAGE sql IMMUTABLE
	RETURN COALESCE(NULLIF(display_name, ''), split_part(email, '@', 1));

-- +goose Down
DROP FUNCTION user_display_name(TEXT, TEXT);
