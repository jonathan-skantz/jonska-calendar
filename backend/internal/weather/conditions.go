package weather

import (
	"slices"
	"strings"
)

type Condition string

const (
	Clear        Condition = "clear"
	PartlyCloudy Condition = "partly-cloudy"
	Cloudy       Condition = "cloudy"
	Fog          Condition = "fog"
	Rain         Condition = "rain"
	Snow         Condition = "snow"
	Thunder      Condition = "thunder"
)

// conditionOrder is the display order and also the order conditions are
// returned in, so the same mix of weather always renders the same way.
var conditionOrder = []Condition{Clear, PartlyCloudy, Cloudy, Fog, Rain, Snow, Thunder}

const maxConditionsPerDay = 3

// Sunrise and Sunset are local "HH:MM" and are omitted when the sun does not
// rise or set that day.
type Day struct {
	Date       string      `json:"date"`
	Conditions []Condition `json:"conditions"`
	Sunrise    *string     `json:"sunrise,omitempty"`
	Sunset     *string     `json:"sunset,omitempty"`
}

// conditionForCode maps a WMO weather code to a display condition.
func conditionForCode(code int) (Condition, bool) {
	switch {
	case code <= 1:
		return Clear, true
	case code == 2:
		return PartlyCloudy, true
	case code == 3:
		return Cloudy, true
	case code == 45 || code == 48:
		return Fog, true
	case code >= 51 && code <= 67, code >= 80 && code <= 82:
		return Rain, true
	case code >= 71 && code <= 77, code == 85 || code == 86:
		return Snow, true
	case code >= 95 && code <= 99:
		return Thunder, true
	}
	return "", false
}

// minimumHours is how many daytime hours a condition must cover to be shown,
// so a single odd hour does not add an icon. Thunder is notable even briefly.
func minimumHours(condition Condition) int {
	if condition == Thunder {
		return 1
	}
	return 2
}

// ForecastDays combines each local day's conditions with its sun times. Days
// are returned in chronological order.
func ForecastDays(forecast Forecast) []Day {
	days := dailyConditions(forecast.Hourly)
	byDate := map[string]int{}
	for i, day := range days {
		byDate[day.Date] = i
	}
	daily := forecast.Daily
	for i, date := range daily.Time {
		sunrise, sunset := clockTime(date, daily.Sunrise[i]), clockTime(date, daily.Sunset[i])
		index, ok := byDate[date]
		if !ok {
			// Polar night has no daylight hours but still belongs in the list.
			index = len(days)
			byDate[date] = index
			days = append(days, Day{Date: date, Conditions: []Condition{}})
		}
		days[index].Sunrise, days[index].Sunset = sunrise, sunset
	}
	slices.SortFunc(days, func(a, b Day) int { return strings.Compare(a.Date, b.Date) })
	return days
}

// clockTime returns the "HH:MM" part of a local timestamp on date, or nil when
// it is missing or falls on another day (the sun did not rise or set).
func clockTime(date string, timestamp *string) *string {
	if timestamp == nil {
		return nil
	}
	day, clock, ok := strings.Cut(*timestamp, "T")
	if !ok || day != date || len(clock) < 5 {
		return nil
	}
	clock = clock[:5]
	return &clock
}

// dailyConditions reduces hourly codes to the distinct conditions seen during
// daylight on each local day. Days are returned in chronological order.
func dailyConditions(hourly Hourly) []Day {
	dates := []string{}
	hours := map[string]map[Condition]int{}
	for i, timestamp := range hourly.Time {
		date, _, ok := strings.Cut(timestamp, "T")
		if !ok {
			continue
		}
		if _, seen := hours[date]; !seen {
			dates = append(dates, date)
			hours[date] = map[Condition]int{}
		}
		if hourly.IsDay[i] == nil || *hourly.IsDay[i] != 1 || hourly.WeatherCode[i] == nil {
			continue
		}
		if condition, ok := conditionForCode(*hourly.WeatherCode[i]); ok {
			hours[date][condition]++
		}
	}

	days := make([]Day, 0, len(dates))
	for _, date := range dates {
		counts := hours[date]
		if len(counts) == 0 {
			continue
		}
		present := map[Condition]bool{}
		for condition, count := range counts {
			present[condition] = count >= minimumHours(condition)
		}
		// Overcast is implied by precipitation, but sun is not, so a day of
		// sun and showers keeps both icons.
		if present[Rain] || present[Snow] || present[Thunder] {
			present[Cloudy] = false
		}

		conditions := []Condition{}
		for _, condition := range conditionOrder {
			if present[condition] {
				conditions = append(conditions, condition)
			}
		}
		// conditionOrder ends with the most disruptive weather, so trimming
		// from the front keeps what matters most for planning.
		if len(conditions) > maxConditionsPerDay {
			conditions = conditions[len(conditions)-maxConditionsPerDay:]
		}
		days = append(days, Day{Date: date, Conditions: conditions})
	}
	return days
}
