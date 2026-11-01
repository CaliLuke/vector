package browsersession

import "errors"

// IsReadFailure reports unavailable storage or malformed stored claims that
// prevent reliable authentication. These failures require an error response.
func IsReadFailure(err error) bool {
	return errors.Is(err, ErrStore) ||
		errors.Is(err, ErrSessionStore) ||
		errors.Is(err, ErrUserStore) ||
		errors.Is(err, ErrInvalidSessionShape) ||
		errors.Is(err, ErrInvalidUserID)
}
