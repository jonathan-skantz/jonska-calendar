package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
	"golang.org/x/sync/singleflight"
)

// cacheTTL bounds upstream calls to a few per coordinate cell per day, shared
// by every user in that cell.
const cacheTTL = 3 * time.Hour

var (
	ErrInvalidCoordinates = errors.New("lat and lon must be valid coordinates")
	errUnavailable        = errors.New("weather is unavailable")
)

type Handler struct {
	Store       *db.Store
	CurrentUser auth.CurrentUser
	Client      *Client
	now         func() time.Time
	fetches     singleflight.Group
}

func New(store *db.Store, currentUser auth.CurrentUser, client *Client) *Handler {
	return &Handler{Store: store, CurrentUser: currentUser, Client: client, now: time.Now}
}

func (h *Handler) Forecast(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.RequireUser(w, r, h.CurrentUser); !ok {
		return
	}
	latitude, longitude, err := parseCoordinates(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Concurrent requests for the same cell share one lookup and upstream
	// call. The work is detached from this request so one client hanging up
	// does not fail the others waiting on it.
	key := fmt.Sprintf("%.1f,%.1f", latitude, longitude)
	result, err, _ := h.fetches.Do(key, func() (any, error) {
		return h.forecast(context.WithoutCancel(r.Context()), latitude, longitude)
	})
	if errors.Is(err, errUnavailable) {
		http.Error(w, "weather is unavailable", http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		log.Printf("weather: %v", err)
		http.Error(w, "failed to load weather", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ForecastDays(result.(Forecast)))
}

// forecast returns the cached forecast for the cell, refreshing it from
// upstream when it is older than cacheTTL or predates stored sun times. A
// stale forecast is still better than none, so it is served when the refresh
// fails.
func (h *Handler) forecast(ctx context.Context, latitude, longitude float64) (Forecast, error) {
	cached, err := h.Store.GetWeatherForecast(ctx, latitude, longitude)
	hasCached := err == nil
	if err != nil && !errors.Is(err, db.ErrWeatherNotCached) {
		return Forecast{}, err
	}
	if hasCached && cached.Daily != nil && h.now().Sub(cached.FetchedAt) < cacheTTL {
		return decodeForecast(cached)
	}

	fresh, err := h.Client.Forecast(ctx, latitude, longitude)
	if err != nil {
		log.Printf("weather: refreshing %.1f,%.1f: %v", latitude, longitude, err)
		if hasCached {
			return decodeForecast(cached)
		}
		return Forecast{}, errUnavailable
	}
	hourly, err := json.Marshal(fresh.Hourly)
	if err != nil {
		return Forecast{}, err
	}
	daily, err := json.Marshal(fresh.Daily)
	if err != nil {
		return Forecast{}, err
	}
	row := db.WeatherForecast{FetchedAt: h.now(), Hourly: hourly, Daily: daily}
	if err := h.Store.UpsertWeatherForecast(ctx, latitude, longitude, row); err != nil {
		return Forecast{}, err
	}
	return fresh, nil
}

func decodeForecast(row db.WeatherForecast) (Forecast, error) {
	var forecast Forecast
	if err := json.Unmarshal(row.Hourly, &forecast.Hourly); err != nil {
		return Forecast{}, err
	}
	if row.Daily != nil {
		if err := json.Unmarshal(row.Daily, &forecast.Daily); err != nil {
			return Forecast{}, err
		}
	}
	return forecast, nil
}

// parseCoordinates rounds to 0.1 degrees (about 10 km). That is the cache
// cell, and it also keeps exact positions out of the database.
func parseCoordinates(query url.Values) (float64, float64, error) {
	latitude, err := strconv.ParseFloat(query.Get("lat"), 64)
	if err != nil {
		return 0, 0, ErrInvalidCoordinates
	}
	longitude, err := strconv.ParseFloat(query.Get("lon"), 64)
	if err != nil {
		return 0, 0, ErrInvalidCoordinates
	}
	// NaN fails every comparison, so check range by inclusion.
	if !(latitude >= -90 && latitude <= 90) || !(longitude >= -180 && longitude <= 180) {
		return 0, 0, ErrInvalidCoordinates
	}
	return math.Round(latitude*10) / 10, math.Round(longitude*10) / 10, nil
}
