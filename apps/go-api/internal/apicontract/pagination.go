// Package apicontract supplies shared native API response and query contracts.
package apicontract

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

type Query struct {
	Params  map[string]string
	Present map[string]bool
}
type HTTPError struct {
	Status int
	Body   map[string]any
}

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d", e.Status) }

func PageSize(request Query) (int, error) {
	if !request.Present["per_page"] {
		return 1000, nil
	}
	value, err := strconv.Atoi(request.Params["per_page"])
	if err != nil || value < 1 || value > 1000 {
		return 0, invalidPageSizeError()
	}
	return value, nil
}

func invalidPageSizeError() error {
	return &HTTPError{Status: 400, Body: map[string]any{"detail": "per_page must be an integer from 1 to 1000."}}
}

func CursorPage(request Query, limit int) (int, error) {
	if limit < 1 || limit > 1000 {
		return 0, invalidCursorError()
	}
	if !request.Present["cursor"] {
		return 0, nil
	}
	parts := strings.Split(request.Params["cursor"], ":")
	if len(parts) != 3 {
		return 0, invalidCursorError()
	}
	previousSize, sizeErr := strconv.Atoi(parts[0])
	page, pageErr := strconv.Atoi(parts[1])
	if sizeErr != nil || previousSize < 1 || previousSize > 1000 || pageErr != nil || page < 0 || page > (math.MaxInt-limit)/limit || (parts[2] != "0" && parts[2] != "1") {
		return 0, invalidCursorError()
	}
	return page, nil
}

func invalidCursorError() error {
	return &HTTPError{Status: 400, Body: map[string]any{"detail": "Invalid cursor parameter."}}
}

func Envelope(results []any, total, limit, page int) map[string]any {
	pages := 0
	if total > 0 {
		pages = int(math.Ceil(float64(total) / float64(limit)))
	}
	nextPage := new(big.Int).Add(big.NewInt(int64(page)), big.NewInt(1))
	previousPage := new(big.Int).Sub(big.NewInt(int64(page)), big.NewInt(1))
	nextOffset := new(big.Int).Mul(new(big.Int).Set(nextPage), big.NewInt(int64(limit)))
	return map[string]any{"grouped_by": nil, "sub_grouped_by": nil, "total_count": total, "next_cursor": fmt.Sprintf("%d:%s:0", limit, nextPage), "prev_cursor": fmt.Sprintf("%d:%s:1", limit, previousPage), "next_page_results": nextOffset.Cmp(big.NewInt(int64(total))) < 0, "prev_page_results": page > 0, "count": len(results), "total_pages": pages, "total_results": total, "extra_stats": nil, "results": results}
}

func PageWindow(request Query) (limit, page, offset int, err error) {
	limit, err = PageSize(request)
	if err != nil {
		return 0, 0, 0, err
	}
	page, err = CursorPage(request, limit)
	if err != nil {
		return 0, 0, 0, err
	}
	return limit, page, page * limit, nil
}

// CursorWindow reads the page size embedded in a self-contained cursor.
func CursorWindow(cursor *string) (limit, page, offset int, err error) {
	query := Query{Params: map[string]string{}, Present: map[string]bool{}}
	if cursor != nil {
		size, _, found := strings.Cut(*cursor, ":")
		if !found {
			return 0, 0, 0, invalidCursorError()
		}
		query.Params["per_page"], query.Present["per_page"] = size, true
		query.Params["cursor"], query.Present["cursor"] = *cursor, true
	}
	limit, page, offset, err = PageWindow(query)
	if err != nil {
		return 0, 0, 0, invalidCursorError()
	}
	return limit, page, offset, nil
}
