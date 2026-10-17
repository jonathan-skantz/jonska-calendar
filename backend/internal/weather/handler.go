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

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"golang.org/x/sync/singleflight"
)

// cacheTTL bounds upstream calls to a few per coordinate cell per day, shared
// by every user in that cell.
const cacheTTL = 3 * time.Hour

var (
	ErrInvalidCoordinates = errors.New("lat and lon must be valid coordinates")
	errUnavailable        = errors.New("weather is unavailable")
)

type CurrentUser func(*http.Request) (uuid.UUID, error)

type Handler struct {
	Store       *db.Store
	CurrentUser CurrentUser
	Client      *Client
	now         func() time.Time
	fetches     singleflight.Group
}

func New(store *db.Store, currentUser CurrentUser, client *Client) *Handler {
	return &Handler{Store: store, CurrentUser: currentUser, Client: client, now: time.Now}
}

func (h *Handler) Forecast(w http.ResponseWriter, r *http.Request) {
	if _, err := h.CurrentUser(r); err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
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
		return h.hourly(context.WithoutCancel(r.Context()), latitude, longitude)
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
	_ = json.NewEncoder(w).Encode(DailyConditions(result.(Hourly)))
}

// hourly returns the cached forecast for the cell, refreshing it from
// upstream when it is older than cacheTTL. A stale forecast is still better
// than none, so it is served when the refresh fails.
func (h *Handler) hourly(ctx context.Context, latitude, longitude float64) (Hourly, error) {
	cached, err := h.Store.GetWeatherForecast(ctx, latitude, longitude)
	hasCached := err == nil
	if err != nil && !errors.Is(err, db.ErrWeatherNotCached) {
		return Hourly{}, err
	}
	if hasCached && h.now().Sub(cached.FetchedAt) < cacheTTL {
		return decodeHourly(cached.Hourly)
	}

	fresh, err := h.Client.Hourly(ctx, latitude, longitude)
	if err != nil {
		log.Printf("weather: refreshing %.1f,%.1f: %v", latitude, longitude, err)
		if hasCached {
			return decodeHourly(cached.Hourly)
		}
		return Hourly{}, errUnavailable
	}
	encoded, err := json.Marshal(fresh)
	if err != nil {
		return Hourly{}, err
	}
	if err := h.Store.UpsertWeatherForecast(ctx, latitude, longitude, h.now(), encoded); err != nil {
		return Hourly{}, err
	}
	return fresh, nil
}

func decodeHourly(raw json.RawMessage) (Hourly, error) {
	var hourly Hourly
	err := json.Unmarshal(raw, &hourly)
	return hourly, err
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
