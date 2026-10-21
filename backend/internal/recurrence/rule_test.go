package recurrence

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

var stockholm = mustLoad("Europe/Stockholm")

func mustLoad(name string) *time.Location {
	zone, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return zone
}

// tuesday is Tuesday 13 October 2026 at 10:00 in Stockholm: the 2nd Tuesday
// of its month.
var tuesday = time.Date(2026, 10, 13, 10, 0, 0, 0, stockholm)

func TestValidateNormalizes(t *testing.T) {
	tests := []struct {
		name string
		rule Rule
		want Rule
	}{
		{
			name: "weekly defaults to the start's weekday",
			rule: Rule{Frequency: Weekly, Interval: 1},
			want: Rule{Frequency: Weekly, Interval: 1, Weekdays: []string{"TU"}},
		},
		{
			name: "weekdays in order without repeats",
			rule: Rule{Frequency: Weekly, Interval: 2, Weekdays: []string{"TH", "TU", "TH"}},
			want: Rule{Frequency: Weekly, Interval: 2, Weekdays: []string{"TU", "TH"}},
		},
		{
			name: "monthly defaults to the day of the month",
			rule: Rule{Frequency: Monthly, Interval: 1, Weekdays: []string{"TU"}},
			want: Rule{Frequency: Monthly, Interval: 1, MonthlyBy: DayOfMonth},
		},
		{
			name: "fields that do not apply are cleared",
			rule: Rule{Frequency: Daily, Interval: 1, Weekdays: []string{"TU"}, MonthlyBy: Weekday, Count: 5},
			want: Rule{Frequency: Daily, Interval: 1, Count: 5},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Validate(test.rule, tuesday)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("expected %+v, got %+v", test.want, got)
			}
		})
	}
}

func TestValidateRejects(t *testing.T) {
	lastFriday := time.Date(2026, 10, 30, 18, 0, 0, 0, stockholm)
	fifthThursday := time.Date(2026, 10, 29, 18, 0, 0, 0, stockholm)
	tests := []struct {
		name  string
		rule  Rule
		start time.Time
		want  error
	}{
		{"no frequency", Rule{Interval: 1}, tuesday, ErrInvalidRule},
		{"interval 0", Rule{Frequency: Daily}, tuesday, ErrInvalidRule},
		{"interval 100", Rule{Frequency: Daily, Interval: 100}, tuesday, ErrInvalidRule},
		{"count 1000", Rule{Frequency: Daily, Interval: 1, Count: 1000}, tuesday, ErrInvalidRule},
		{"count and until", Rule{Frequency: Daily, Interval: 1, Count: 2, Until: new("2026-12-01")}, tuesday, ErrInvalidRule},
		{"bad until", Rule{Frequency: Daily, Interval: 1, Until: new("1 Dec")}, tuesday, ErrInvalidRule},
		{"until before the start", Rule{Frequency: Daily, Interval: 1, Until: new("2026-10-12")}, tuesday, ErrUntil},
		{"unknown weekday", Rule{Frequency: Weekly, Interval: 1, Weekdays: []string{"TU", "XX"}}, tuesday, ErrInvalidRule},
		{"weekdays without the start's", Rule{Frequency: Weekly, Interval: 1, Weekdays: []string{"WE"}}, tuesday, ErrWeekdays},
		{"fifth weekday", Rule{Frequency: Monthly, Interval: 1, MonthlyBy: Weekday}, fifthThursday, ErrMonthlyBy},
		{"last weekday that is not last", Rule{Frequency: Monthly, Interval: 1, MonthlyBy: LastWeekday}, tuesday, ErrMonthlyBy},
		{"unknown monthly", Rule{Frequency: Monthly, Interval: 1, MonthlyBy: "sometimes"}, lastFriday, ErrMonthlyBy},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Validate(test.rule, test.start); !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
	if _, err := Validate(Rule{Frequency: Monthly, Interval: 1, MonthlyBy: LastWeekday}, lastFriday); err != nil {
		t.Fatalf("last Friday: %v", err)
	}
}

func TestRRULE(t *testing.T) {
	allDay := time.Date(2026, 10, 13, 0, 0, 0, 0, stockholm)
	tests := []struct {
		name     string
		rule     Rule
		start    time.Time
		isAllDay bool
		want     string
	}{
		{"weekly until", Rule{Frequency: Weekly, Interval: 2, Weekdays: []string{"TU", "TH"}, Until: new("2026-12-31")}, tuesday, false,
			"FREQ=WEEKLY;INTERVAL=2;BYDAY=TU,TH;UNTIL=20261231T225959Z"},
		{"all-day until is a date", Rule{Frequency: Daily, Interval: 1, Until: new("2026-12-31")}, allDay, true,
			"FREQ=DAILY;INTERVAL=1;UNTIL=20261231"},
		{"monthly on the 2nd Tuesday", Rule{Frequency: Monthly, Interval: 1, MonthlyBy: Weekday, Count: 6}, tuesday, false,
			"FREQ=MONTHLY;INTERVAL=1;COUNT=6;BYDAY=+2TU"},
		{"monthly on the day", Rule{Frequency: Monthly, Interval: 3, MonthlyBy: DayOfMonth}, tuesday, false,
			"FREQ=MONTHLY;INTERVAL=3;BYMONTHDAY=13"},
		{"yearly", Rule{Frequency: Yearly, Interval: 1}, allDay, true, "FREQ=YEARLY;INTERVAL=1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.rule.RRULE(test.start, test.isAllDay)
			if got != test.want {
				t.Fatalf("expected %s, got %s", test.want, got)
			}
			back, err := FromRRULE(got, test.start)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(back, test.rule) {
				t.Fatalf("read back %+v, expected %+v", back, test.rule)
			}
		})
	}
}

func TestFromRRULE(t *testing.T) {
	lastFriday := time.Date(2026, 10, 30, 18, 0, 0, 0, stockholm)
	tests := []struct {
		name  string
		value string
		start time.Time
		want  Rule
	}{
		{"Google weekly", "RRULE:FREQ=WEEKLY;BYDAY=TU,TH", tuesday,
			Rule{Frequency: Weekly, Interval: 1, Weekdays: []string{"TU", "TH"}}},
		{"every weekday", "FREQ=DAILY;BYDAY=MO,TU,WE,TH,FR", tuesday,
			Rule{Frequency: Weekly, Interval: 1, Weekdays: []string{"MO", "TU", "WE", "TH", "FR"}}},
		{"last Friday", "FREQ=MONTHLY;BYDAY=-1FR", lastFriday,
			Rule{Frequency: Monthly, Interval: 1, MonthlyBy: LastWeekday}},
		{"Outlook 2nd Tuesday", "FREQ=MONTHLY;BYDAY=TU;BYSETPOS=2", tuesday,
			Rule{Frequency: Monthly, Interval: 1, MonthlyBy: Weekday}},
		{"yearly with its own date", "FREQ=YEARLY;BYMONTH=10;BYMONTHDAY=13", tuesday,
			Rule{Frequency: Yearly, Interval: 1}},
		{"week start that does not matter", "FREQ=WEEKLY;WKST=SU;BYDAY=TU", tuesday,
			Rule{Frequency: Weekly, Interval: 1, Weekdays: []string{"TU"}}},
		{"UNTIL before the start time ends the day before", "FREQ=DAILY;UNTIL=20261020T070000Z", tuesday,
			Rule{Frequency: Daily, Interval: 1, Until: new("2026-10-19")}},
		{"UNTIL after the start time includes the day", "FREQ=DAILY;UNTIL=20261020T080000Z", tuesday,
			Rule{Frequency: Daily, Interval: 1, Until: new("2026-10-20")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := FromRRULE(test.value, test.start)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("expected %+v, got %+v", test.want, got)
			}
		})
	}

	for _, value := range []string{
		"FREQ=HOURLY",
		"FREQ=MONTHLY;BYMONTHDAY=1,15",
		"FREQ=MONTHLY;BYDAY=3TU",
		"FREQ=WEEKLY;BYDAY=WE",
		"FREQ=WEEKLY;WKST=SU;INTERVAL=2;BYDAY=TU,TH",
		"FREQ=YEARLY;BYMONTH=6",
		"FREQ=YEARLY;BYWEEKNO=20",
		"FREQ=DAILY;INTERVAL=2;BYDAY=MO,TU",
		"INTERVAL=2",
		"not a rule",
	} {
		if _, err := FromRRULE(value, tuesday); !errors.Is(err, ErrUnsupported) {
			t.Errorf("%s: expected ErrUnsupported, got %v", value, err)
		}
	}
}
