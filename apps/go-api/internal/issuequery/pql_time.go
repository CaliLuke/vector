package issuequery

import (
	"strconv"
	"strings"
	"time"
)

func (c *pqlCompiler) dateValue(v pqlValue, dateOnly bool) (time.Time, error) {
	invalid := pqlError(v.pos, "invalid date or date function")
	if v.kind == "call" {
		value, err := c.dateFunction(v)
		if err == nil && (value.Year() < 1 || value.Year() > 9999) {
			return time.Time{}, invalid
		}
		return value, err
	}
	if v.kind != "string" {
		return time.Time{}, invalid
	}
	date, err := time.Parse(time.DateOnly, v.text)
	if err == nil {
		if date.Year() == 0 {
			return time.Time{}, invalid
		}
		return c.civilTime(date, v.pos)
	}
	if !dateOnly {
		if timestamp, parseErr := time.Parse(time.RFC3339Nano, v.text); parseErr == nil && timestamp.Year() > 0 {
			return timestamp, nil
		}
	}
	return time.Time{}, invalid
}

func (c *pqlCompiler) dateFunction(v pqlValue) (time.Time, error) {
	invalid := pqlError(v.pos, "invalid date function or arguments")
	if c.request.Now.IsZero() {
		return time.Time{}, pqlError(v.pos, "request time is required")
	}
	now := c.request.Now.In(c.location).Truncate(time.Microsecond)
	if shift, ok := pqlShifts[v.text]; ok {
		if len(v.arguments) != 1 || v.arguments[0].kind != "number" {
			return time.Time{}, invalid
		}
		amount, err := strconv.Atoi(v.arguments[0].text)
		if err != nil || amount < 0 || amount > 10000 {
			return time.Time{}, invalid
		}
		wall := pqlWall(now)
		if shift.months != 0 {
			month := time.Date(wall.Year(), wall.Month()+time.Month(amount*shift.months), 1, 0, 0, 0, 0, time.UTC)
			last := month.AddDate(0, 1, -1).Day()
			wall = time.Date(month.Year(), month.Month(), min(wall.Day(), last), wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), time.UTC)
		} else {
			wall = wall.AddDate(0, 0, amount*shift.days)
		}
		return c.civilTime(wall, v.pos)
	}
	if len(v.arguments) != 0 {
		return time.Time{}, invalid
	}
	if v.text == "now" {
		return now, nil
	}
	start, end, ok := pqlPeriod(now, v.text)
	if !ok {
		return time.Time{}, invalid
	}
	value, err := c.civilTime(start, v.pos)
	if err == nil && end {
		value = value.Add(-time.Microsecond)
	}
	return value, err
}

var pqlShifts = map[string]struct{ days, months int }{
	"daysAgo": {-1, 0}, "daysFromNow": {1, 0},
	"weeksAgo": {-7, 0}, "weeksFromNow": {7, 0},
	"monthsAgo": {0, -1}, "monthsFromNow": {0, 1},
}

func pqlPeriod(now time.Time, name string) (time.Time, bool, bool) {
	wall := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := strings.HasPrefix(name, "endOf")
	switch name {
	case "today", "startOfDay", "endOfDay":
		if end {
			wall = wall.AddDate(0, 0, 1)
		}
	case "startOfWeek", "endOfWeek":
		wall = wall.AddDate(0, 0, -(int(now.Weekday())+6)%7)
		if end {
			wall = wall.AddDate(0, 0, 7)
		}
	case "startOfMonth", "endOfMonth":
		wall = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if end {
			wall = wall.AddDate(0, 1, 0)
		}
	case "startOfYear", "endOfYear":
		wall = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		if end {
			wall = wall.AddDate(1, 0, 0)
		}
	default:
		return time.Time{}, false, false
	}
	return wall, end, true
}

func pqlWall(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}

// Resolve the civil time explicitly. time.Date alone makes an unspecified
// choice in folds and may normalize a gap into a different date. The enclosing
// zone intervals supply both candidate offsets even for whole-day transitions.
func (c *pqlCompiler) civilTime(wall time.Time, pos int) (time.Time, error) {
	guess := time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), c.location)
	start, end := guess.ZoneBounds()
	probes := []time.Time{guess}
	if !start.IsZero() {
		probes = append(probes, start.Add(-time.Nanosecond))
	}
	if !end.IsZero() {
		probes = append(probes, end)
	}
	var earliest time.Time
	found := false
	for _, probe := range probes {
		_, offset := probe.Zone()
		candidate := wall.Add(-time.Duration(offset) * time.Second).In(c.location)
		if pqlWall(candidate).Equal(wall) && (!found || candidate.Before(earliest)) {
			earliest = candidate
			found = true
		}
	}
	if !found {
		return time.Time{}, pqlError(pos, "local time does not exist in timezone")
	}
	return earliest, nil
}
