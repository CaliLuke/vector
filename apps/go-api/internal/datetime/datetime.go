// Package datetime owns API calendar values and timestamps.
package datetime

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var timestampPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]{1,9})?(?:Z|[+-](?:[01][0-9]|2[0-3]):[0-5][0-9])$`)
var localPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}(?:[T ][0-9]{2}:[0-9]{2}(?::[0-9]{2}(?:\.[0-9]{1,9})?)?)?$`)

// Parse accepts RFC 3339 timestamps and local calendar values. Local dates
// select midnight. Local times require the caller's explicit location.
// Zoned input is normalized to UTC.
func Parse(value string, location *time.Location) (time.Time, error) {
	var parsed time.Time
	var err error
	switch {
	case len(value) <= 35 && timestampPattern.MatchString(value):
		parsed, err = time.Parse(time.RFC3339Nano, value)
		parsed = parsed.UTC()
	case location != nil && len(value) <= 29 && localPattern.MatchString(value):
		layout := time.DateOnly
		if len(value) > 10 {
			layout = "2006-01-02T15:04"
			if len(value) > 16 {
				layout += ":05.999999999"
			}
		}
		parsed, err = time.ParseInLocation(layout, strings.Replace(value, " ", "T", 1), location)
	default:
		return time.Time{}, errors.New("timestamp must be RFC 3339 or a local calendar value")
	}
	if err != nil || parsed.Year() < 1 || parsed.Year() > 9999 {
		return time.Time{}, errors.New("invalid calendar timestamp")
	}
	return parsed.Truncate(time.Microsecond), nil
}

// Format uses database microsecond precision and the requested timezone.
func Format(value time.Time, location *time.Location) string {
	if location == nil {
		location = time.UTC
	}
	return value.In(location).Truncate(time.Microsecond).Format(time.RFC3339Nano)
}
