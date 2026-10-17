-- +goose Up
-- Daily sunrise/sunset times. Rows cached before this column existed have
-- NULL and are refetched on their next request.
ALTER TABLE weather_forecasts ADD COLUMN daily JSONB;

-- +goose Down
ALTER TABLE weather_forecasts DROP COLUMN daily;
