package ical

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var errInvalidDuration = errors.New("invalid duration")

// duration is an RFC 5545 duration, such as -PT15M, P1D or P1W. Weeks and
// days are nominal: adding a day keeps the wall-clock time across a
// daylight saving change.
type duration struct {
	negative bool
	days     int
	clock    time.Duration
}

func parseDuration(value string) (duration, error) {
	var length duration
	value = strings.TrimSpace(value)
	if rest, ok := strings.CutPrefix(value, "-"); ok {
		length.negative, value = true, rest
	} else {
		value = strings.TrimPrefix(value, "+")
	}
	value, ok := strings.CutPrefix(value, "P")
	if !ok || value == "" {
		return duration{}, errInvalidDuration
	}
	datePart, timePart, hasTime := strings.Cut(value, "T")
	if hasTime && timePart == "" {
		return duration{}, errInvalidDuration
	}
	if err := eachComponent(datePart, func(amount int, unit byte) error {
		switch unit {
		case 'W':
			length.days += amount * 7
		case 'D':
			length.days += amount
		default:
			return errInvalidDuration
		}
		return nil
	}); err != nil {
		return duration{}, err
	}
	if err := eachComponent(timePart, func(amount int, unit byte) error {
		switch unit {
		case 'H':
			length.clock += time.Duration(amount) * time.Hour
		case 'M':
			length.clock += time.Duration(amount) * time.Minute
		case 'S':
			length.clock += time.Duration(amount) * time.Second
		default:
			return errInvalidDuration
		}
		return nil
	}); err != nil {
		return duration{}, err
	}
	return length, nil
}

// eachComponent calls fn for each number and unit letter in a part such
// as "1H30M".
func eachComponent(part string, fn func(amount int, unit byte) error) error {
	for part != "" {
		end := strings.IndexFunc(part, func(r rune) bool { return r < '0' || r > '9' })
		if end <= 0 {
			return errInvalidDuration
		}
		amount, err := strconv.Atoi(part[:end])
		if err != nil {
			return fmt.Errorf("%w: %v", errInvalidDuration, err)
		}
		if err := fn(amount, part[end]); err != nil {
			return err
		}
		part = part[end+1:]
	}
	return nil
}

func (length duration) addTo(at time.Time) time.Time {
	if length.negative {
		return at.AddDate(0, 0, -length.days).Add(-length.clock)
	}
	return at.AddDate(0, 0, length.days).Add(length.clock)
}

// nominal is the length with each day counted as 24 hours.
func (length duration) nominal() time.Duration {
	total := time.Duration(length.days)*24*time.Hour + length.clock
	if length.negative {
		return -total
	}
	return total
}
