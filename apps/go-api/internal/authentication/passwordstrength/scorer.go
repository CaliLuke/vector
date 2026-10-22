// Package passwordstrength preserves Plane's pinned Django zxcvbn score cutoff.
package passwordstrength

import (
	"errors"

	"github.com/timbutler/zxcvbn"
)

var ErrEmptyPassword = errors.New("password is empty")

type Scorer struct{}

// Score mirrors Plane's source calls to zxcvbn(password) without user inputs.
// Email is accepted to satisfy the shared auth workflow interface; source does
// not feed it into the strength estimator.
func (Scorer) Score(password, _ string) (int, error) {
	if password == "" {
		return 0, ErrEmptyPassword
	}
	return zxcvbn.PasswordStrength(password, nil).Score, nil
}
