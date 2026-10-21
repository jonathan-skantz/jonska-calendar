// Package recurrence holds the rules of repeating events and expands them
// into occurrences. It supports the subset of RFC 5545 recurrence rules the
// app can edit: daily, weekly on chosen days, monthly on a date or a weekday
// of the month, and yearly, each every N, ending never, on a date or after N
// times.
package recurrence

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

type Frequency string

const (
	Daily   Frequency = "daily"
	Weekly  Frequency = "weekly"
	Monthly Frequency = "monthly"
	Yearly  Frequency = "yearly"
)

// MonthlyBy says how a monthly rule picks its day: the start's day of the
// month (the 15th), the start's weekday by its number in the month (the 2nd
// Tuesday), or the last such weekday of the month (the last Friday).
type MonthlyBy string

const (
	DayOfMonth  MonthlyBy = "day_of_month"
	Weekday     MonthlyBy = "weekday"
	LastWeekday MonthlyBy = "last_weekday"
)

const (
	MaxInterval = 99
	MaxCount    = 999
)

var (
	ErrInvalidRule = fmt.Errorf("repeat must be daily, weekly, monthly or yearly, every 1 to %d, ending never, on a date or after 1 to %d times", MaxInterval, MaxCount)
	ErrWeekdays    = errors.New("a weekly repeat must include the weekday the event starts on")
	ErrMonthlyBy   = errors.New("a monthly repeat must be on the start's day of the month, or on its weekday as the 1st to 4th or last of the month")
	ErrUntil       = errors.New("a repeat cannot end before the event starts")
	ErrUnsupported = errors.New("unsupported repeat rule")
)

// weekdayCodes are the RFC 5545 weekday names, Monday first.
var weekdayCodes = []string{"MO", "TU", "WE", "TH", "FR", "SA", "SU"}

// Rule is how an event repeats, as the API and the event form see it.
type Rule struct {
	Frequency Frequency `json:"frequency"`
	Interval  int       `json:"interval"`
	// Weekdays are RFC 5545 codes such as "MO", for weekly rules only.
	Weekdays  []string  `json:"weekdays,omitempty"`
	MonthlyBy MonthlyBy `json:"monthly_by,omitempty"`
	// Until is the last date, YYYY-MM-DD, that an occurrence may start on,
	// in the series' time zone. At most one of Until and Count is set.
	Until *string `json:"until,omitempty"`
	Count int     `json:"count,omitempty"`
}

const dateLayout = "2006-01-02"

// Validate checks the rule against the series' first start, given in the
// series' time zone, and returns it normalized: weekdays in order without
// repeats, and fields that do not apply to the frequency cleared.
func Validate(rule Rule, start time.Time) (Rule, error) {
	if rule.Interval < 1 || rule.Interval > MaxInterval || rule.Count < 0 || rule.Count > MaxCount ||
		(rule.Count > 0 && rule.Until != nil) {
		return rule, ErrInvalidRule
	}
	startDay := weekdayCode(start.Weekday())
	switch rule.Frequency {
	case Weekly:
		if len(rule.Weekdays) == 0 {
			rule.Weekdays = []string{startDay}
		}
		var days []string
		for _, code := range weekdayCodes {
			if slices.Contains(rule.Weekdays, code) {
				days = append(days, code)
			}
		}
		if len(days) != len(slices.Compact(slices.Sorted(slices.Values(rule.Weekdays)))) {
			return rule, ErrInvalidRule
		}
		if !slices.Contains(days, startDay) {
			return rule, ErrWeekdays
		}
		rule.Weekdays, rule.MonthlyBy = days, ""
	case Monthly:
		rule.Weekdays = nil
		switch rule.MonthlyBy {
		case "":
			rule.MonthlyBy = DayOfMonth
		case DayOfMonth:
		case Weekday:
			if weekOfMonth(start) > 4 {
				return rule, ErrMonthlyBy
			}
		case LastWeekday:
			if !isLastWeekdayOfMonth(start) {
				return rule, ErrMonthlyBy
			}
		default:
			return rule, ErrMonthlyBy
		}
	case Daily, Yearly:
		rule.Weekdays, rule.MonthlyBy = nil, ""
	default:
		return rule, ErrInvalidRule
	}
	if rule.Until != nil {
		until, err := time.Parse(dateLayout, *rule.Until)
		if err != nil {
			return rule, ErrInvalidRule
		}
		startDate := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
		if until.Before(startDate) {
			return rule, ErrUntil
		}
	}
	return rule, nil
}

// options builds the rrule-go options for a validated rule whose series
// starts at start, in the series' time zone.
func (rule Rule) options(start time.Time) rrule.ROption {
	option := rrule.ROption{Dtstart: start, Interval: rule.Interval, Count: rule.Count}
	switch rule.Frequency {
	case Daily:
		option.Freq = rrule.DAILY
	case Weekly:
		option.Freq = rrule.WEEKLY
		for _, code := range rule.Weekdays {
			option.Byweekday = append(option.Byweekday, rruleWeekday(code))
		}
	case Monthly:
		option.Freq = rrule.MONTHLY
		day := rruleWeekday(weekdayCode(start.Weekday()))
		switch rule.MonthlyBy {
		case Weekday:
			option.Byweekday = []rrule.Weekday{day.Nth(weekOfMonth(start))}
		case LastWeekday:
			option.Byweekday = []rrule.Weekday{day.Nth(-1)}
		default:
			option.Bymonthday = []int{start.Day()}
		}
	case Yearly:
		option.Freq = rrule.YEARLY
	}
	if rule.Until != nil {
		// The last second of the until date, so an occurrence that starts
		// on it is included.
		until, _ := time.ParseInLocation(dateLayout, *rule.Until, start.Location())
		option.Until = until.AddDate(0, 0, 1).Add(-time.Second)
	}
	return option
}

// RRULE formats a validated rule as an RFC 5545 RRULE value. UNTIL is a UTC
// time for timed series, and a date for all-day ones, as the RFC requires.
func (rule Rule) RRULE(start time.Time, isAllDay bool) string {
	option := rule.options(start)
	until := option.Until
	option.Until = time.Time{}
	value := option.RRuleString()
	if !until.IsZero() {
		if isAllDay {
			value += ";UNTIL=" + until.Format("20060102")
		} else {
			value += ";UNTIL=" + until.UTC().Format("20060102T150405Z")
		}
	}
	return value
}

// FromRRULE reads an RRULE value into a rule, for a series that starts at
// start in its time zone. Rules the app cannot show or edit, such as
// several days of the month, return ErrUnsupported.
func FromRRULE(value string, start time.Time) (Rule, error) {
	option, err := rrule.StrToROptionInLocation(strings.TrimPrefix(strings.TrimSpace(value), "RRULE:"), start.Location())
	if err != nil {
		return Rule{}, fmt.Errorf("%w: %v", ErrUnsupported, err)
	}
	if len(option.Byyearday) > 0 || len(option.Byweekno) > 0 || len(option.Byhour) > 0 ||
		len(option.Byminute) > 0 || len(option.Bysecond) > 0 || len(option.Byeaster) > 0 {
		return Rule{}, ErrUnsupported
	}
	rule := Rule{Interval: max(option.Interval, 1), Count: option.Count}
	if !option.Until.IsZero() {
		// UNTIL is an instant. An occurrence on its date that starts after
		// it is not included, so the last date is the day before.
		untilAt := option.Until.In(start.Location())
		lastDay := time.Date(untilAt.Year(), untilAt.Month(), untilAt.Day(),
			start.Hour(), start.Minute(), start.Second(), 0, start.Location())
		if lastDay.After(untilAt) {
			lastDay = lastDay.AddDate(0, 0, -1)
		}
		until := lastDay.Format(dateLayout)
		rule.Until = &until
	}
	startDay := rruleWeekday(weekdayCode(start.Weekday()))
	onlyStartDay := func(days []rrule.Weekday) bool {
		return len(days) == 1 && days[0].Day() == startDay.Day()
	}
	switch option.Freq {
	case rrule.DAILY:
		rule.Frequency = Daily
		if len(option.Bymonthday) > 0 || len(option.Bymonth) > 0 || len(option.Bysetpos) > 0 {
			return Rule{}, ErrUnsupported
		}
		// Every weekday, as calendars often write it, is weekly on those days.
		if len(option.Byweekday) > 0 {
			if rule.Interval > 1 {
				return Rule{}, ErrUnsupported
			}
			rule.Frequency = Weekly
			for _, day := range option.Byweekday {
				if day.N() != 0 {
					return Rule{}, ErrUnsupported
				}
				rule.Weekdays = append(rule.Weekdays, weekdayCodes[day.Day()])
			}
		}
	case rrule.WEEKLY:
		rule.Frequency = Weekly
		if len(option.Bymonthday) > 0 || len(option.Bymonth) > 0 || len(option.Bysetpos) > 0 {
			return Rule{}, ErrUnsupported
		}
		for _, day := range option.Byweekday {
			if day.N() != 0 {
				return Rule{}, ErrUnsupported
			}
			rule.Weekdays = append(rule.Weekdays, weekdayCodes[day.Day()])
		}
		// The week's first day changes which weeks count when a rule skips
		// weeks over several days, which the app does not model.
		if option.Wkst.Day() != rrule.MO.Day() && rule.Interval > 1 && len(rule.Weekdays) > 1 {
			return Rule{}, ErrUnsupported
		}
	case rrule.MONTHLY:
		rule.Frequency = Monthly
		nth := 0
		switch {
		case len(option.Byweekday) == 0 && len(option.Bysetpos) == 0 &&
			(len(option.Bymonthday) == 0 || slices.Equal(option.Bymonthday, []int{start.Day()})):
			rule.MonthlyBy = DayOfMonth
		case len(option.Bymonthday) == 0 && len(option.Bysetpos) == 0 && onlyStartDay(option.Byweekday):
			nth = option.Byweekday[0].N()
		case len(option.Bymonthday) == 0 && len(option.Bysetpos) == 1 && onlyStartDay(option.Byweekday) &&
			option.Byweekday[0].N() == 0:
			nth = option.Bysetpos[0]
		default:
			return Rule{}, ErrUnsupported
		}
		switch {
		case rule.MonthlyBy == DayOfMonth:
		case nth == -1:
			rule.MonthlyBy = LastWeekday
		case nth == weekOfMonth(start):
			rule.MonthlyBy = Weekday
		default:
			return Rule{}, ErrUnsupported
		}
		if len(option.Bymonth) > 0 {
			return Rule{}, ErrUnsupported
		}
	case rrule.YEARLY:
		rule.Frequency = Yearly
		if len(option.Byweekday) > 0 || len(option.Bysetpos) > 0 ||
			(len(option.Bymonth) > 0 && !slices.Equal(option.Bymonth, []int{int(start.Month())})) ||
			(len(option.Bymonthday) > 0 && !slices.Equal(option.Bymonthday, []int{start.Day()})) {
			return Rule{}, ErrUnsupported
		}
	default:
		return Rule{}, ErrUnsupported
	}
	validated, err := Validate(rule, start)
	if err != nil {
		return Rule{}, fmt.Errorf("%w: %v", ErrUnsupported, err)
	}
	return validated, nil
}

func weekdayCode(day time.Weekday) string {
	// time.Weekday starts on Sunday; the codes start on Monday.
	return weekdayCodes[(int(day)+6)%7]
}

func rruleWeekday(code string) rrule.Weekday {
	return []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR, rrule.SA, rrule.SU}[slices.Index(weekdayCodes, code)]
}

// weekOfMonth is which of its weekday in the month a day is: 1 for the
// first Tuesday, 5 for a fifth.
func weekOfMonth(day time.Time) int {
	return (day.Day()-1)/7 + 1
}

func isLastWeekdayOfMonth(day time.Time) bool {
	return day.AddDate(0, 0, 7).Month() != day.Month()
}
