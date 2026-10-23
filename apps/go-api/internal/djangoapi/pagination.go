// Package djangoapi implements source-proven shared Django API contracts.
package djangoapi

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
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
	value, err := parsePageSizeValue(request)
	if err != nil {
		return 0, err
	}
	return narrowPageSize(value)
}

// PageSizeWithMinimum applies an endpoint-specific lower bound before converting
// Django's arbitrary-precision per_page value to the platform's machine int.
// PageSize intentionally keeps its historical behavior for existing callers.
func PageSizeWithMinimum(request Query, minimum int) (int, error) {
	if !request.Present["per_page"] {
		if minimum > 1000 {
			return 0, &HTTPError{Status: 400, Body: map[string]any{"detail": "Error in parsing"}}
		}
		return 1000, nil
	}
	value, err := parsePageSizeValue(request)
	if err != nil {
		return 0, err
	}
	if value.Cmp(big.NewInt(int64(minimum))) < 0 {
		return 0, &HTTPError{Status: 400, Body: map[string]any{"detail": "Error in parsing"}}
	}
	return narrowPageSize(value)
}

func parsePageSizeValue(request Query) (*big.Int, error) {
	value, err := pythoncompat.DecimalInteger(request.Params["per_page"])
	if err != nil {
		return nil, &HTTPError{Status: 400, Body: map[string]any{"detail": "Invalid per_page parameter."}}
	}
	if value.Cmp(big.NewInt(1000)) > 0 {
		return nil, &HTTPError{Status: 400, Body: map[string]any{"detail": "Invalid per_page value. Cannot exceed 1000."}}
	}
	return value, nil
}

func narrowPageSize(value *big.Int) (int, error) {
	if !value.IsInt64() {
		return 0, &HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	}
	limit := value.Int64()
	if int64(int(limit)) != limit {
		return 0, &HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	}
	return int(limit), nil
}

func CursorPage(request Query, limit int) (int, error) {
	if !request.Present["cursor"] {
		return 0, nil
	}
	parts := strings.Split(request.Params["cursor"], ":")
	if len(parts) != 3 || !pythonCursorValue(parts[0]) {
		return 0, invalidCursorError()
	}
	pageValue, err := pythoncompat.DecimalInteger(parts[1])
	if err != nil {
		return 0, invalidCursorError()
	}
	_, err = pythoncompat.DecimalInteger(parts[2])
	if err != nil {
		return 0, invalidCursorError()
	}
	offsetValue := new(big.Int).Mul(pageValue, big.NewInt(int64(limit)))
	if offsetValue.Sign() < 0 {
		return 0, &HTTPError{Status: 400, Body: map[string]any{"detail": "Error in parsing"}}
	}
	if !pageValue.IsInt64() {
		return 0, &HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	}
	page := pageValue.Int64()
	if int64(int(page)) != page {
		return 0, &HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	}
	return int(page), nil
}

func pythonCursorValue(raw string) bool {
	if !strings.Contains(raw, ".") {
		_, err := pythoncompat.DecimalInteger(raw)
		return err == nil
	}
	_, err := pythoncompat.DecimalFloat64(raw)
	return err == nil
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
	if limit <= 0 {
		return 0, 0, 0, &HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	}
	if page > math.MaxInt/limit {
		return 0, 0, 0, &HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	}
	return limit, page, page * limit, nil
}
