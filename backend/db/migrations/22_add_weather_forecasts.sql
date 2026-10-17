-- +goose Up
-- A shared cache of upstream forecasts, not user data: it has no user_id and
-- every user whose rounded position falls in the same cell reuses the row.
-- Coordinates are rounded to 0.1 degrees (about 10 km) before they are stored.
CREATE TABLE weather_forecasts (
	latitude NUMERIC(4, 1) NOT NULL,
	longitude NUMERIC(4, 1) NOT NULL,
	fetched_at TIMESTAMPTZ NOT NULL,
	hourly JSONB NOT NULL,
	PRIMARY KEY (latitude, longitude),
	CONSTRAINT weather_forecasts_latitude_valid CHECK (latitude BETWEEN -90 AND 90),
	CONSTRAINT weather_forecasts_longitude_valid CHECK (longitude BETWEEN -180 AND 180)
);

-- +goose Down
DROP TABLE weather_forecasts;
