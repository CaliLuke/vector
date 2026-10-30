package djangosession

import "errors"

// IsReadFailure reports storage failures and malformed signed session data that
// prevent authentication from completing. These failures must not be treated as
// an ordinary anonymous session.
func IsReadFailure(err error) bool {
	return errors.Is(err, ErrStore) ||
		errors.Is(err, ErrSessionStore) ||
		errors.Is(err, ErrUserStore) ||
		errors.Is(err, ErrInvalidSessionShape) ||
		errors.Is(err, ErrInvalidUserID)
}
