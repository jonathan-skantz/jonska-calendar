package db

import (
	"context"
	"encoding/json"
	"time"
)

// WeatherForecast is a cached upstream forecast for one rounded coordinate
// cell. It is shared by all users and is not owned by anyone.
type WeatherForecast struct {
	FetchedAt time.Time
	Hourly    json.RawMessage
}

func (s *Store) GetWeatherForecast(ctx context.Context, latitude, longitude float64) (WeatherForecast, error) {
	var forecast WeatherForecast
	err := s.db.QueryRow(ctx, `
		SELECT fetched_at, hourly
		FROM weather_forecasts
		WHERE latitude = $1 AND longitude = $2
	`, latitude, longitude).Scan(&forecast.FetchedAt, &forecast.Hourly)
	if isNoRows(err) {
		return WeatherForecast{}, ErrWeatherNotCached
	}
	return forecast, err
}

func (s *Store) UpsertWeatherForecast(ctx context.Context, latitude, longitude float64, fetchedAt time.Time, hourly json.RawMessage) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO weather_forecasts (latitude, longitude, fetched_at, hourly)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (latitude, longitude) DO UPDATE
		SET fetched_at = EXCLUDED.fetched_at, hourly = EXCLUDED.hourly
	`, latitude, longitude, fetchedAt, hourly)
	return err
}
