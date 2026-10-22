package pythoncompat

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ISODateTime parses Python 3.12 datetime.fromisoformat input. A nil
// naiveLocation rejects timezone-less values, as session expiry arithmetic does.
func ISODateTime(value string, naiveLocation *time.Location) (time.Time, error) {
	dateLength, err := djangoISODateLength(value)
	if err != nil {
		return time.Time{}, err
	}
	date, err := parseDjangoISODate(value[:dateLength])
	if err != nil {
		return time.Time{}, err
	}
	rest := value[dateLength:]
	if rest == "" {
		if naiveLocation == nil {
			return time.Time{}, errors.New("naive Django ISO session expiry")
		}
		return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, naiveLocation), nil
	}
	_, separatorSize := utf8.DecodeRuneInString(rest)
	if separatorSize == 0 || (separatorSize == 1 && rest[0] >= utf8.RuneSelf) {
		return time.Time{}, errors.New("invalid Django ISO datetime separator")
	}
	rest = rest[separatorSize:]
	timeText, offsetText, aware := splitDjangoISOTimezone(rest)
	if !aware {
		if naiveLocation == nil {
			return time.Time{}, errors.New("naive Django ISO session expiry")
		}
		timeText = rest
	}
	hour, minute, second, microsecond, err := parseDjangoISOClock(timeText, true)
	if err != nil || hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, errors.New("invalid Django ISO session expiry clock")
	}
	if !aware {
		return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, second, microsecond*1_000, naiveLocation), nil
	}
	offset, err := parseDjangoISOOffset(offsetText)
	if err != nil {
		return time.Time{}, err
	}
	local := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, second, microsecond*1_000, time.UTC)
	return local.Add(-offset), nil
}

// PythonDateTimeString formats a UTC aware datetime like Python's str(datetime).
// Python datetimes retain microsecond precision, so sub-microsecond values are
// truncated before formatting.
func PythonDateTimeString(value time.Time) string {
	value = value.UTC().Truncate(time.Microsecond)
	formatted := value.Format("2006-01-02 15:04:05")
	if value.Nanosecond() != 0 {
		formatted += value.Format(".000000")
	}
	return formatted + "+00:00"
}

// DRFDateTimeString formats an aware datetime using DRF's ISO representation
// in the request timezone. Python datetime precision is microseconds; a
// nonzero fraction is always rendered with six digits.
func DRFDateTimeString(value time.Time, location *time.Location) string {
	if location == nil {
		location = time.UTC
	}
	value = value.In(location).Truncate(time.Microsecond)
	if value.Nanosecond() == 0 {
		return value.Format("2006-01-02T15:04:05Z07:00")
	}
	return value.Format("2006-01-02T15:04:05.000000Z07:00")
}

func djangoISODateLength(value string) (int, error) {
	if len(value) >= 7 && len(value) > 4 && value[4] == 'W' {
		if len(value) >= 8 && asciiDigit(value[7]) {
			return 8, nil
		}
		return 7, nil
	}
	if len(value) >= 8 && strings.HasPrefix(value[4:], "-W") {
		if len(value) >= 10 && value[8] == '-' && asciiDigit(value[9]) {
			return 10, nil
		}
		return 8, nil
	}
	if len(value) >= 10 && value[4] == '-' && value[7] == '-' {
		return 10, nil
	}
	if len(value) >= 8 {
		return 8, nil
	}
	return 0, errors.New("invalid Django ISO session expiry date")
}

func parseDjangoISODate(value string) (time.Time, error) {
	var year, month, day int
	var err error
	switch {
	case len(value) == 10 && value[4] == '-' && value[7] == '-':
		year, err = asciiInteger(value[0:4])
		if err == nil {
			month, err = asciiInteger(value[5:7])
		}
		if err == nil {
			day, err = asciiInteger(value[8:10])
		}
	case len(value) == 8 && value[4] != 'W' && value[4] != '-':
		year, err = asciiInteger(value[0:4])
		if err == nil {
			month, err = asciiInteger(value[4:6])
		}
		if err == nil {
			day, err = asciiInteger(value[6:8])
		}
	case (len(value) == 8 || len(value) == 10) && strings.HasPrefix(value[4:], "-W"):
		year, err = asciiInteger(value[0:4])
		week := 0
		if err == nil {
			week, err = asciiInteger(value[6:8])
		}
		weekday := 1
		if err == nil && len(value) == 10 {
			weekday, err = asciiInteger(value[9:10])
		}
		if err == nil {
			return djangoISOWeekDate(year, week, weekday)
		}
	case (len(value) == 7 || len(value) == 8) && value[4] == 'W':
		year, err = asciiInteger(value[0:4])
		week := 0
		if err == nil {
			week, err = asciiInteger(value[5:7])
		}
		weekday := 1
		if err == nil && len(value) == 8 {
			weekday, err = asciiInteger(value[7:8])
		}
		if err == nil {
			return djangoISOWeekDate(year, week, weekday)
		}
	default:
		err = errors.New("unsupported date form")
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("parse Django ISO session expiry date: %w", err)
	}
	if year < 1 || year > 9999 {
		return time.Time{}, errors.New("invalid Django ISO session expiry year")
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return time.Time{}, errors.New("invalid Django ISO session expiry date")
	}
	return date, nil
}

func djangoISOWeekDate(year int, week int, weekday int) (time.Time, error) {
	if year < 1 || year > 9999 || week < 1 || week > 53 || weekday < 1 || weekday > 7 {
		return time.Time{}, errors.New("invalid Django ISO week date")
	}
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	mondayOffset := (int(jan4.Weekday()) + 6) % 7
	date := jan4.AddDate(0, 0, -mondayOffset+(week-1)*7+(weekday-1))
	isoYear, isoWeek := date.ISOWeek()
	if isoYear != year || isoWeek != week {
		return time.Time{}, errors.New("invalid Django ISO week date")
	}
	return date, nil
}

func splitDjangoISOTimezone(value string) (string, string, bool) {
	if strings.HasSuffix(value, "Z") {
		return value[:len(value)-1], "Z", true
	}
	for index := 1; index < len(value); index++ {
		if value[index] == '+' || value[index] == '-' {
			return value[:index], value[index:], true
		}
	}
	return "", "", false
}

func parseDjangoISOOffset(value string) (time.Duration, error) {
	if value == "Z" {
		return 0, nil
	}
	if len(value) < 2 || (value[0] != '+' && value[0] != '-') {
		return 0, errors.New("invalid Django ISO session expiry offset")
	}
	hour, minute, second, microsecond, err := parseDjangoISOClock(value[1:], false)
	if err != nil {
		return 0, errors.New("invalid Django ISO session expiry offset")
	}
	// CPython's fromisoformat() treats an offset with zero integral
	// components as UTC, even when it carries fractional microseconds.
	if hour == 0 && minute == 0 && second == 0 {
		return 0, nil
	}
	offset := time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute + time.Duration(second)*time.Second + time.Duration(microsecond)*time.Microsecond
	if offset >= 24*time.Hour {
		return 0, errors.New("invalid Django ISO session expiry offset")
	}
	if value[0] == '-' {
		offset = -offset
	}
	return offset, nil
}

func parseDjangoISOClock(value string, allowEmptyFraction bool) (int, int, int, int, error) {
	fraction := ""
	if dot := strings.IndexAny(value, ".,"); dot >= 0 {
		fraction = value[dot+1:]
		value = value[:dot]
		if fraction == "" && (!allowEmptyFraction || strings.HasSuffix(value, ":")) {
			return 0, 0, 0, 0, errors.New("empty Django ISO fractional second")
		}
	}
	parts := strings.Split(value, ":")
	if fraction == "" && len(parts) > 1 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 1 && !strings.Contains(value, ":") {
		switch len(value) {
		case 2:
			parts = []string{value}
		case 4:
			parts = []string{value[:2], value[2:4]}
		case 6:
			parts = []string{value[:2], value[2:4], value[4:6]}
		default:
			return 0, 0, 0, 0, errors.New("invalid Django ISO clock")
		}
	}
	if len(parts) < 1 || len(parts) > 3 {
		return 0, 0, 0, 0, errors.New("invalid Django ISO clock")
	}
	components := [3]int{}
	for index, part := range parts {
		if len(part) != 2 {
			return 0, 0, 0, 0, errors.New("invalid Django ISO clock component")
		}
		parsed, err := asciiInteger(part)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		components[index] = parsed
	}
	microsecond := 0
	if fraction != "" {
		for index := range fraction {
			if !asciiDigit(fraction[index]) {
				return 0, 0, 0, 0, errors.New("invalid Django ISO fractional second")
			}
		}
		if len(fraction) > 6 {
			fraction = fraction[:6]
		}
		fraction += strings.Repeat("0", 6-len(fraction))
		microsecond, _ = strconv.Atoi(fraction)
	}
	return components[0], components[1], components[2], microsecond, nil
}

func asciiInteger(value string) (int, error) {
	if value == "" {
		return 0, errors.New("empty decimal component")
	}
	for index := range value {
		if !asciiDigit(value[index]) {
			return 0, errors.New("non-decimal component")
		}
	}
	return strconv.Atoi(value)
}

func asciiDigit(value byte) bool {
	return value >= '0' && value <= '9'
}
