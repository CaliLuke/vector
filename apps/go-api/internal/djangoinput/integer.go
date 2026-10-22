package djangoinput

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

var integerDecimalSuffix = regexp.MustCompile(`\.0*\s*$`)

// IntegerField follows DRF IntegerField's decimal-zero suffix normalization.
// ScalarString supplies Python's numeric spelling before Python int parsing.
func IntegerField(value any) (*big.Int, error) {
	text := ScalarString(value)
	if utf8.RuneCountInString(text) > 1000 {
		return nil, fmt.Errorf("integer input exceeds DRF's string limit")
	}
	return pythoncompat.DecimalInteger(integerDecimalSuffix.ReplaceAllString(strings.TrimSpace(text), ""))
}
