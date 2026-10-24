package db

import (
	"context"
	"encoding/json"
	"time"
)

// WeatherForecast is a cached upstream forecast for one rounded coordinate
// cell. It is shared by all users and is not owned by anyone.
// Daily is nil for rows cached before sun times were stored.
type WeatherForecast struct {
	FetchedAt time.Time
	Hourly    json.RawMessage
	Daily     json.RawMessage
}

func (s *Store) GetWeatherForecast(ctx context.Context, latitude, longitude float64) (WeatherForecast, error) {
	var forecast WeatherForecast
	err := s.db.QueryRow(ctx, `
		SELECT fetched_at, hourly, daily
		FROM weather_forecasts
		WHERE latitude = $1 AND longitude = $2
	`, latitude, longitude).Scan(&forecast.FetchedAt, &forecast.Hourly, &forecast.Daily)
	if isNoRows(err) {
		return WeatherForecast{}, ErrWeatherNotCached
	}
	return forecast, err
}

func (s *Store) UpsertWeatherForecast(ctx context.Context, latitude, longitude float64, forecast WeatherForecast) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO weather_forecasts (latitude, longitude, fetched_at, hourly, daily)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (latitude, longitude) DO UPDATE
		SET fetched_at = EXCLUDED.fetched_at, hourly = EXCLUDED.hourly, daily = EXCLUDED.daily
	`, latitude, longitude, forecast.FetchedAt, forecast.Hourly, forecast.Daily)
	return err
}

// DeleteWeatherFetchedBefore removes forecasts fetched before the given time,
// which are too old to show, and returns how many it removed.
func (s *Store) DeleteWeatherFetchedBefore(ctx context.Context, before time.Time) (int64, error) {
	tag, err := s.db.Exec(ctx, `DELETE FROM weather_forecasts WHERE fetched_at < $1`, before)
	return tag.RowsAffected(), err
}
