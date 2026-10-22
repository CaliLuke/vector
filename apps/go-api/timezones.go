package systemapi

import (
	"context"
	"fmt"
	"sort"
	"time"
	_ "time/tzdata"

	timezoneapi "github.com/CaliLuke/go-plane/gen/timezones"
)

type timezoneLocation struct {
	label string
	value string
}

type timezoneResult struct {
	location timezoneLocation
	offset   int
}

type timezoneService struct {
	now func() time.Time
}

// NewTimezones returns the supported timezone list service.
func NewTimezones() timezoneapi.Service {
	return newTimezones(time.Now)
}

func newTimezones(now func() time.Time) timezoneapi.Service {
	return &timezoneService{now: now}
}

func (service *timezoneService) List(context.Context) (*timezoneapi.TimezoneList, error) {
	now := service.now()
	results := make([]timezoneResult, 0, len(timezoneLocations))
	for _, item := range timezoneLocations {
		location, err := time.LoadLocation(item.value)
		if err != nil {
			continue
		}
		_, offset := now.In(location).Zone()
		results = append(results, timezoneResult{location: item, offset: offset})
	}
	sort.Slice(results, func(left, right int) bool {
		if results[left].offset == results[right].offset {
			return results[left].location.label < results[right].location.label
		}
		return results[left].offset < results[right].offset
	})

	timezones := make([]*timezoneapi.Timezone, 0, len(results))
	for _, result := range results {
		offset := formatTimezoneOffset(result.offset)
		timezones = append(timezones, &timezoneapi.Timezone{
			GmtOffset: "GMT" + offset,
			Label:     result.location.label,
			UtcOffset: "UTC" + offset,
			Value:     result.location.value,
		})
	}
	return &timezoneapi.TimezoneList{Timezones: timezones}, nil
}

func formatTimezoneOffset(offsetSeconds int) string {
	hours := offsetSeconds / 3600
	remainder := offsetSeconds % 3600
	if remainder < 0 {
		hours--
		remainder += 3600
	}
	sign := "+"
	if hours < 0 {
		sign = "-"
		hours = -hours
	}
	return fmt.Sprintf("%s%02d:%02d", sign, hours, remainder/60)
}

var timezoneLocations = []timezoneLocation{
	{label: "American Samoa", value: "Pacific/Pago_Pago"},
	{label: "Midway Island", value: "Pacific/Midway"},
	{label: "Aleutian Islands", value: "America/Adak"},
	{label: "Hawaii", value: "Pacific/Honolulu"},
	{label: "Tahiti", value: "Pacific/Tahiti"},
	{label: "Marquesas Islands", value: "Pacific/Marquesas"},
	{label: "Alaska", value: "America/Anchorage"},
	{label: "Gambier Islands", value: "Pacific/Gambier"},
	{label: "Baja California", value: "America/Tijuana"},
	{label: "Pacific Time (US and Canada)", value: "America/Los_Angeles"},
	{label: "Arizona", value: "America/Phoenix"},
	{label: "Mountain Time (US and Canada)", value: "America/Denver"},
	{label: "Central Time (US and Canada)", value: "America/Chicago"},
	{label: "Chihuahua, Mazatlan", value: "America/Chihuahua"},
	{label: "Costa Rica", value: "America/Costa_Rica"},
	{label: "Guadalajara, Mexico City, Monterrey", value: "America/Mexico_City"},
	{label: "Saskatchewan", value: "America/Regina"},
	{label: "Tegucigalpa, Honduras", value: "America/Tegucigalpa"},
	{label: "Bogota", value: "America/Bogota"},
	{label: "Chetumal", value: "America/Cancun"},
	{label: "Eastern Time (US and Canada)", value: "America/New_York"},
	{label: "Lima", value: "America/Lima"},
	{label: "Quito", value: "America/Guayaquil"},
	{label: "Atlantic Time (Canada)", value: "America/Halifax"},
	{label: "Bermuda", value: "Atlantic/Bermuda"},
	{label: "Caracas", value: "America/Caracas"},
	{label: "Caracas (Old Venezuela Time)", value: "America/Caracas"},
	{label: "Georgetown", value: "America/Guyana"},
	{label: "La Paz", value: "America/La_Paz"},
	{label: "Manaus", value: "America/Manaus"},
	{label: "Newfoundland Time (Canada)", value: "America/St_Johns"},
	{label: "Brasilia", value: "America/Sao_Paulo"},
	{label: "Buenos Aires", value: "America/Argentina/Buenos_Aires"},
	{label: "Falkland Islands", value: "Atlantic/Stanley"},
	{label: "Montevideo", value: "America/Montevideo"},
	{label: "Santiago", value: "America/Santiago"},
	{label: "Greenland", value: "America/Godthab"},
	{label: "South Georgia and the South Sandwich Islands", value: "Atlantic/South_Georgia"},
	{label: "Azores", value: "Atlantic/Azores"},
	{label: "Cape Verde Islands", value: "Atlantic/Cape_Verde"},
	{label: "Dublin", value: "Europe/Dublin"},
	{label: "Lisbon", value: "Europe/Lisbon"},
	{label: "Monrovia", value: "Africa/Monrovia"},
	{label: "Reykjavik", value: "Atlantic/Reykjavik"},
	{label: "Algiers", value: "Africa/Algiers"},
	{label: "Casablanca", value: "Africa/Casablanca"},
	{label: "Central European Time (Berlin, Rome, Paris)", value: "Europe/Paris"},
	{label: "Lagos", value: "Africa/Lagos"},
	{label: "Tunis", value: "Africa/Tunis"},
	{label: "West Central Africa", value: "Africa/Lagos"},
	{label: "Athens", value: "Europe/Athens"},
	{label: "Eastern European Time (Cairo, Helsinki, Kyiv)", value: "Europe/Kyiv"},
	{label: "Harare, Pretoria", value: "Africa/Harare"},
	{label: "Jerusalem", value: "Asia/Jerusalem"},
	{label: "Johannesburg", value: "Africa/Johannesburg"},
	{label: "Baghdad", value: "Asia/Baghdad"},
	{label: "Kuwait, Riyadh", value: "Asia/Riyadh"},
	{label: "Moscow Time", value: "Europe/Moscow"},
	{label: "Nairobi", value: "Africa/Nairobi"},
	{label: "Tehran", value: "Asia/Tehran"},
	{label: "Abu Dhabi", value: "Asia/Dubai"},
	{label: "Astrakhan", value: "Europe/Astrakhan"},
	{label: "Baku", value: "Asia/Baku"},
	{label: "Mauritius", value: "Indian/Mauritius"},
	{label: "Tbilisi", value: "Asia/Tbilisi"},
	{label: "Yerevan", value: "Asia/Yerevan"},
	{label: "Kabul", value: "Asia/Kabul"},
	{label: "Almaty", value: "Asia/Almaty"},
	{label: "Islamabad", value: "Asia/Karachi"},
	{label: "Karachi", value: "Asia/Karachi"},
	{label: "Maldives", value: "Indian/Maldives"},
	{label: "Tashkent", value: "Asia/Tashkent"},
	{label: "Yekaterinburg", value: "Asia/Yekaterinburg"},
	{label: "Chennai", value: "Asia/Kolkata"},
	{label: "Kolkata", value: "Asia/Kolkata"},
	{label: "Mumbai", value: "Asia/Kolkata"},
	{label: "New Delhi", value: "Asia/Kolkata"},
	{label: "Sri Jayawardenepura", value: "Asia/Colombo"},
	{label: "Kathmandu", value: "Asia/Kathmandu"},
	{label: "Bishkek", value: "Asia/Bishkek"},
	{label: "Chagos", value: "Indian/Chagos"},
	{label: "Dhaka", value: "Asia/Dhaka"},
	{label: "Thimphu", value: "Asia/Thimphu"},
	{label: "Cocos Islands", value: "Indian/Cocos"},
	{label: "Yangon (Rangoon)", value: "Asia/Yangon"},
	{label: "Bangkok", value: "Asia/Bangkok"},
	{label: "Hanoi", value: "Asia/Ho_Chi_Minh"},
	{label: "Jakarta", value: "Asia/Jakarta"},
	{label: "Krasnoyarsk", value: "Asia/Krasnoyarsk"},
	{label: "Novosibirsk", value: "Asia/Novosibirsk"},
	{label: "Beijing", value: "Asia/Shanghai"},
	{label: "Hong Kong", value: "Asia/Hong_Kong"},
	{label: "Perth", value: "Australia/Perth"},
	{label: "Singapore", value: "Asia/Singapore"},
	{label: "Ulaanbaatar", value: "Asia/Ulaanbaatar"},
	{label: "Eucla", value: "Australia/Eucla"},
	{label: "Palau", value: "Pacific/Palau"},
	{label: "Seoul", value: "Asia/Seoul"},
	{label: "Tokyo", value: "Asia/Tokyo"},
	{label: "Yakutsk", value: "Asia/Yakutsk"},
	{label: "Darwin", value: "Australia/Darwin"},
	{label: "Brisbane", value: "Australia/Brisbane"},
	{label: "Guam", value: "Pacific/Guam"},
	{label: "Vladivostok", value: "Asia/Vladivostok"},
	{label: "Adelaide", value: "Australia/Adelaide"},
	{label: "Bougainville Island", value: "Pacific/Bougainville"},
	{label: "Chokurdakh", value: "Asia/Srednekolymsk"},
	{label: "Lord Howe Island", value: "Australia/Lord_Howe"},
	{label: "Magadan", value: "Asia/Magadan"},
	{label: "Solomon Islands", value: "Pacific/Guadalcanal"},
	{label: "Sydney", value: "Australia/Sydney"},
	{label: "Anadyr", value: "Asia/Anadyr"},
	{label: "Fiji Islands", value: "Pacific/Fiji"},
	{label: "Norfolk Island", value: "Pacific/Norfolk"},
	{label: "Auckland", value: "Pacific/Auckland"},
	{label: "Nuku'alofa", value: "Pacific/Tongatapu"},
	{label: "Samoa", value: "Pacific/Apia"},
	{label: "Wellington", value: "Pacific/Auckland"},
	{label: "Chatham Islands", value: "Pacific/Chatham"},
	{label: "Kiritimati Island", value: "Pacific/Kiritimati"},
}
