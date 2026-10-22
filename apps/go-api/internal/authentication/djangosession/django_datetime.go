package djangosession

import (
	"time"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

// Session expiry rejects naive timestamps because Django subtracts timezone.now().
func djangoDateTimeFromISOFormat(value string) (time.Time, error) {
	return pythoncompat.ISODateTime(value, nil)
}
