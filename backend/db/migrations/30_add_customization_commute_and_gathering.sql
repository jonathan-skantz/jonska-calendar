-- +goose Up
-- A recipient may add commute to an event they receive, which is theirs
-- alone, and replace its gathering. A NULL gathering_seconds follows the
-- received gathering; 0 removes it. The checks mirror the events table.
ALTER TABLE share_customizations
	ADD COLUMN commute_before_seconds INT NOT NULL DEFAULT 0
		CHECK (commute_before_seconds BETWEEN 0 AND 86400 AND commute_before_seconds % 60 = 0),
	ADD COLUMN commute_after_seconds INT NOT NULL DEFAULT 0
		CHECK (commute_after_seconds BETWEEN 0 AND 86400 AND commute_after_seconds % 60 = 0),
	ADD COLUMN gathering_seconds INT
		CHECK (gathering_seconds BETWEEN 0 AND 86400 AND gathering_seconds % 60 = 0),
	ADD COLUMN gathering_is_optional BOOLEAN NOT NULL DEFAULT false,
	ADD COLUMN gathering_label TEXT
		CHECK (gathering_label <> '' AND char_length(gathering_label) <= 40),
	ADD CHECK (COALESCE(gathering_seconds, 0) > 0 OR (NOT gathering_is_optional AND gathering_label IS NULL));

ALTER TABLE subscription_customizations
	ADD COLUMN commute_before_seconds INT NOT NULL DEFAULT 0
		CHECK (commute_before_seconds BETWEEN 0 AND 86400 AND commute_before_seconds % 60 = 0),
	ADD COLUMN commute_after_seconds INT NOT NULL DEFAULT 0
		CHECK (commute_after_seconds BETWEEN 0 AND 86400 AND commute_after_seconds % 60 = 0),
	ADD COLUMN gathering_seconds INT
		CHECK (gathering_seconds BETWEEN 0 AND 86400 AND gathering_seconds % 60 = 0),
	ADD COLUMN gathering_is_optional BOOLEAN NOT NULL DEFAULT false,
	ADD COLUMN gathering_label TEXT
		CHECK (gathering_label <> '' AND char_length(gathering_label) <= 40),
	ADD CHECK (COALESCE(gathering_seconds, 0) > 0 OR (NOT gathering_is_optional AND gathering_label IS NULL));

-- +goose Down
ALTER TABLE subscription_customizations
	DROP COLUMN commute_before_seconds, DROP COLUMN commute_after_seconds,
	DROP COLUMN gathering_seconds, DROP COLUMN gathering_is_optional, DROP COLUMN gathering_label;
ALTER TABLE share_customizations
	DROP COLUMN commute_before_seconds, DROP COLUMN commute_after_seconds,
	DROP COLUMN gathering_seconds, DROP COLUMN gathering_is_optional, DROP COLUMN gathering_label;
