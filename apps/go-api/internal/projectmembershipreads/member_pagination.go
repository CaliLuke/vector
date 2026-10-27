package projectmembershipreads

import (
	"errors"
	"strconv"
	"strings"
)

// MemberCursor retains the established members-lite cursor semantics.
// Value is checked only for previous-page requests; Page is a page index.
type MemberCursor struct {
	Value    float64
	Page     int
	Previous bool
}

func ParseMemberCursor(raw *string, size int) (MemberCursor, error) {
	if raw == nil {
		return MemberCursor{Value: float64(size)}, nil
	}
	parts := strings.Split(*raw, ":")
	if len(parts) != 3 {
		return MemberCursor{}, errors.New("invalid cursor")
	}
	var value float64
	var err error
	if strings.Contains(parts[0], ".") {
		value, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	} else {
		var integer int
		integer, err = strconv.Atoi(strings.TrimSpace(parts[0]))
		value = float64(integer)
	}
	if err != nil {
		return MemberCursor{}, err
	}
	page, err := strconv.Atoi(parts[1])
	if err != nil {
		return MemberCursor{}, err
	}
	previous, err := strconv.Atoi(parts[2])
	if err != nil {
		return MemberCursor{}, err
	}
	return MemberCursor{Value: value, Page: page, Previous: previous != 0}, nil
}

// MemberPage is the common cursor envelope projected by both transports.
type MemberPage[T any] struct {
	Count           int     `json:"count"`
	ExtraStats      *string `json:"extra_stats"`
	GroupedBy       *string `json:"grouped_by"`
	NextCursor      string  `json:"next_cursor"`
	NextPageResults bool    `json:"next_page_results"`
	PrevCursor      string  `json:"prev_cursor"`
	PrevPageResults bool    `json:"prev_page_results"`
	Results         []T     `json:"results"`
	SubGroupedBy    *string `json:"sub_grouped_by"`
	TotalCount      int     `json:"total_count"`
	TotalPages      int     `json:"total_pages"`
	TotalResults    int     `json:"total_results"`
}

func NewMemberPage[T any](items []T, total, size, page int) (MemberPage[T], error) {
	offset, err := memberPageOffset(size, page)
	if err != nil || total < 0 {
		return MemberPage[T]{}, ErrInvalidMemberQuery
	}
	count := min(len(items), size)
	results := make([]T, count)
	copy(results, items[:count])
	pages := total / size
	if total%size != 0 {
		pages++
	}
	prefix := strconv.Itoa(size) + ":"
	return MemberPage[T]{
		Count: count, NextCursor: prefix + strconv.Itoa(page+1) + ":0", PrevCursor: prefix + strconv.Itoa(page-1) + ":1",
		NextPageResults: len(items) > size || offset < total-count, PrevPageResults: page > 0,
		Results: results, TotalCount: total, TotalPages: pages, TotalResults: total,
	}, nil
}
