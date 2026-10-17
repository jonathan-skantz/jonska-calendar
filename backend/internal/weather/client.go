package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Hourly is the subset of the Open-Meteo hourly forecast the app uses. Times
// are local to the forecast location (timezone=auto). Codes can be null for
// hours the model does not cover.
type Hourly struct {
	Time        []string `json:"time"`
	WeatherCode []*int   `json:"weather_code"`
	IsDay       []*int   `json:"is_day"`
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func NewClient() *Client {
	return &Client{
		BaseURL: "https://api.open-meteo.com",
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Hourly(ctx context.Context, latitude, longitude float64) (Hourly, error) {
	query := url.Values{
		"latitude":      {strconv.FormatFloat(latitude, 'f', -1, 64)},
		"longitude":     {strconv.FormatFloat(longitude, 'f', -1, 64)},
		"hourly":        {"weather_code,is_day"},
		"timezone":      {"auto"},
		"past_days":     {"7"},
		"forecast_days": {"16"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/forecast?"+query.Encode(), nil)
	if err != nil {
		return Hourly{}, err
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return Hourly{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Hourly{}, fmt.Errorf("open-meteo returned status %d", response.StatusCode)
	}

	var body struct {
		Hourly Hourly `json:"hourly"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return Hourly{}, err
	}
	hourly := body.Hourly
	if len(hourly.WeatherCode) != len(hourly.Time) || len(hourly.IsDay) != len(hourly.Time) {
		return Hourly{}, fmt.Errorf("open-meteo returned mismatched hourly arrays")
	}
	return hourly, nil
}
