package pythoncompat

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Django first uses fromisoformat, then accepts its unpadded datetime grammar.
var djangoDateTimePattern = regexp.MustCompile(`^([0-9]{4})-([0-9]{1,2})-([0-9]{1,2})[T ]([0-9]{1,2}):([0-9]{1,2})(?::([0-9]{1,2})(?:[.,]([0-9]{1,12}))?)?[\p{Z}\t\n\r\f\v\x{0085}\x{001c}-\x{001f}]*(Z|[+-][0-9]{2}(?::?[0-9]{2})?)?\n?$`)

// DjangoDateTime matches django.utils.dateparse.parse_datetime, interpreting
// naive inputs in the request timezone as DRF DateTimeField does.
func DjangoDateTime(value string, naiveLocation *time.Location) (time.Time, error) {
	if parsed, err := ISODateTime(value, naiveLocation); err == nil {
		return parsed, nil
	}
	// Django's fallback uses Unicode decimal digits; fromisoformat uses ASCII.
	value = strings.Map(func(r rune) rune {
		if digit, ok := unicodeDecimalDigit(r); ok {
			return rune('0' + digit)
		}
		return r
	}, value)
	parts := djangoDateTimePattern.FindStringSubmatch(value)
	if parts == nil {
		return time.Time{}, fmt.Errorf("invalid Django datetime")
	}
	numbers := make([]int, 6)
	for i := range numbers {
		numbers[i], _ = strconv.Atoi(parts[i+1])
	}
	fraction := parts[7]
	if len(fraction) > 6 {
		fraction = fraction[:6]
	}
	if fraction != "" {
		fraction = "." + fraction
	}
	canonical := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d%s%s", numbers[0], numbers[1], numbers[2], numbers[3], numbers[4], numbers[5], fraction, parts[8])
	return ISODateTime(canonical, naiveLocation)
}
