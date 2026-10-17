package weather

import (
	"fmt"
	"slices"
	"testing"
)

// hourlyDay builds one local day from (hour, code, isDay) entries.
func hourlyDay(date string, entries ...[3]int) Hourly {
	hourly := Hourly{}
	for _, entry := range entries {
		code, isDay := entry[1], entry[2]
		hourly.Time = append(hourly.Time, fmt.Sprintf("%sT%02d:00", date, entry[0]))
		hourly.WeatherCode = append(hourly.WeatherCode, &code)
		hourly.IsDay = append(hourly.IsDay, &isDay)
	}
	return hourly
}

func TestDailyConditions(t *testing.T) {
	tests := []struct {
		name    string
		entries [][3]int
		want    []Condition
	}{
		{name: "sun only", entries: [][3]int{{10, 0, 1}, {11, 1, 1}, {12, 0, 1}}, want: []Condition{Clear}},
		{name: "rain only", entries: [][3]int{{10, 61, 1}, {11, 63, 1}}, want: []Condition{Rain}},
		{name: "sun and rain", entries: [][3]int{{10, 0, 1}, {11, 0, 1}, {14, 80, 1}, {15, 81, 1}}, want: []Condition{Clear, Rain}},
		{name: "cloudy implied by rain", entries: [][3]int{{10, 3, 1}, {11, 3, 1}, {12, 61, 1}, {13, 61, 1}}, want: []Condition{Rain}},
		{name: "cloudy alone", entries: [][3]int{{10, 3, 1}, {11, 3, 1}}, want: []Condition{Cloudy}},
		{name: "night ignored", entries: [][3]int{{1, 0, 0}, {2, 0, 0}, {12, 3, 1}, {13, 3, 1}}, want: []Condition{Cloudy}},
		{name: "single hour is noise", entries: [][3]int{{10, 0, 1}, {11, 0, 1}, {12, 61, 1}}, want: []Condition{Clear}},
		{name: "single thunder hour counts", entries: [][3]int{{10, 0, 1}, {11, 0, 1}, {12, 95, 1}}, want: []Condition{Clear, Thunder}},
		{
			name:    "capped to the most disruptive three",
			entries: [][3]int{{8, 0, 1}, {9, 0, 1}, {10, 45, 1}, {11, 45, 1}, {12, 61, 1}, {13, 61, 1}, {14, 71, 1}, {15, 71, 1}},
			want:    []Condition{Fog, Rain, Snow},
		},
		{name: "only noise gives no icons", entries: [][3]int{{10, 0, 1}, {11, 61, 1}}, want: []Condition{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			days := DailyConditions(hourlyDay("2026-10-17", test.entries...))
			if len(days) != 1 || days[0].Date != "2026-10-17" {
				t.Fatalf("expected one day 2026-10-17, got %+v", days)
			}
			if !slices.Equal(days[0].Conditions, test.want) {
				t.Fatalf("expected %v, got %v", test.want, days[0].Conditions)
			}
		})
	}
}

func TestDailyConditionsSplitsDaysAndSkipsMissingCodes(t *testing.T) {
	hourly := hourlyDay("2026-10-17", [3]int{10, 0, 1}, [3]int{11, 0, 1})
	next := hourlyDay("2026-10-18", [3]int{10, 61, 1}, [3]int{11, 61, 1}, [3]int{12, 0, 1})
	next.WeatherCode[2] = nil
	hourly.Time = append(hourly.Time, next.Time...)
	hourly.WeatherCode = append(hourly.WeatherCode, next.WeatherCode...)
	hourly.IsDay = append(hourly.IsDay, next.IsDay...)

	days := DailyConditions(hourly)
	want := []Day{{Date: "2026-10-17", Conditions: []Condition{Clear}}, {Date: "2026-10-18", Conditions: []Condition{Rain}}}
	if len(days) != len(want) {
		t.Fatalf("expected %v, got %v", want, days)
	}
	for i := range want {
		if days[i].Date != want[i].Date || !slices.Equal(days[i].Conditions, want[i].Conditions) {
			t.Fatalf("expected %v, got %v", want, days)
		}
	}
}
