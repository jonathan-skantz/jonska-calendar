-- +goose Up
-- A demo user is a throwaway account for visitors who want to try the app.
-- It is deleted, with everything it owns, once demo_expires_at has passed.
ALTER TABLE users ADD COLUMN demo_expires_at TIMESTAMPTZ;

CREATE INDEX users_demo_expires_at_idx ON users (demo_expires_at) WHERE demo_expires_at IS NOT NULL;

-- +goose Down
DROP INDEX users_demo_expires_at_idx;
ALTER TABLE users DROP COLUMN demo_expires_at;
