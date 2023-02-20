// Package dates provides some human friendly relative date functions.
package dates

import (
	"fmt"
	"strings"
	"time"
)

func weekday(day time.Weekday) func(now time.Time) time.Time {
	return func(now time.Time) time.Time {
		now = now.AddDate(0, 0, 1)
		for now.Weekday() != day {
			now = now.AddDate(0, 0, 1)
		}
		return now
	}
}

var (
	Monday    = weekday(time.Monday)
	Tuesday   = weekday(time.Tuesday)
	Wednesday = weekday(time.Wednesday)
	Thursday  = weekday(time.Thursday)
	Friday    = weekday(time.Friday)
	Saturday  = weekday(time.Saturday)
	Sunday    = weekday(time.Sunday)
)

func cutPrefix(str, prefix string) (string, bool) {
	if len(str) < len(prefix) {
		return "", false
	}
	if str[:len(prefix)] != prefix {
		return "", false
	}
	return str[len(prefix):], true
}

func ParseRelative(now time.Time, date string) (time.Time, error) {
	if date, found := CutPrefix(date, "next"); found {
		return ParseRelative(now.AddDate(0, 0, 7), date)
	}
	switch strings.ToLower(date) {
	case "tomorrow", "tom":
		return now.AddDate(0, 0, 1), nil
	case "monday", "mon":
		return Monday(now), nil
	case "tuesday", "tue":
		return Tuesday(now), nil
	case "wednesday", "wed":
		return Wednesday(now), nil
	case "thursday", "thu":
		return Thursday(now), nil
	case "friday", "fri":
		return Friday(now), nil
	case "saturday", "sat":
		return Saturday(now), nil
	case "sunday", "sun":
		return Sunday(now), nil
	case "today", "tod":
		return now, nil
	default:
		return time.Time{}, fmt.Errorf("unknown datestring: %q", date)
	}
}
