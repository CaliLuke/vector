package useraccountwrites

import (
	"context"
	"encoding/json"
	"errors"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/password"
	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/useridentity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrUnauthenticated = errors.New("account session is no longer valid")
	ErrEmailUnchanged  = errors.New("new email matches current email")
	ErrCodeInvalid     = errors.New("invalid email verification code")
	ErrCodeExpired     = errors.New("email verification code expired or exhausted")
)

// QueueEmailCode replaces the account's challenge and queues its delivery together.
func (repository *Repository) QueueEmailCode(ctx context.Context, userID uuid.UUID, sessionID, email, code string) (resultErr error) {
	if !validChallengeCode(code) {
		return ErrInvalid
	}
	hash, err := password.Default().Encode(code)
	if err != nil {
		return err
	}
	tx, err := repository.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	current, err := repository.lockEmailAccount(ctx, tx, userID, sessionID, email)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO account_email_challenges(user_id,current_email,email,code_hash,expires_at)
 VALUES($1,$2,$3,$4,clock_timestamp()+interval '10 minutes')
 ON CONFLICT(user_id) DO UPDATE SET current_email=EXCLUDED.current_email,email=EXCLUDED.email,code_hash=EXCLUDED.code_hash,expires_at=EXCLUDED.expires_at,attempts=0`, userID, current, email, hash)
	if err != nil {
		return err
	}
	if queueErr := queueAccountEmail(ctx, tx, "plane.bgtasks.user_email_update_task.send_email_update_magic_code", email, code); queueErr != nil {
		return queueErr
	}
	return tx.Commit(ctx)
}

// ChangeEmail consumes the challenge, revokes sessions and records both notices
// in one transaction. A failed effect or response projection rolls back the write.
func (repository *Repository) ChangeEmail(ctx context.Context, userID uuid.UUID, sessionID, email, code string) (_ json.RawMessage, resultErr error) {
	tx, err := repository.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	current, err := repository.lockEmailAccount(ctx, tx, userID, sessionID, email)
	if err != nil {
		return nil, err
	}
	var hash, original, target string
	var attempts int
	var active bool
	err = tx.QueryRow(ctx, `SELECT code_hash,current_email,email,attempts,expires_at>clock_timestamp() FROM account_email_challenges WHERE user_id=$1 FOR UPDATE`, userID).Scan(&hash, &original, &target, &attempts, &active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCodeExpired
	}
	if err != nil {
		return nil, err
	}
	if !active || attempts >= 5 || original != current || target != email {
		return nil, ErrCodeExpired
	}
	matches := false
	if validChallengeCode(code) {
		matches, err = password.Default().Verify(hash, code)
		if err != nil {
			return nil, err
		}
	}
	if !matches {
		if _, writeErr := tx.Exec(ctx, `UPDATE account_email_challenges SET attempts=attempts+1 WHERE user_id=$1`, userID); writeErr != nil {
			return nil, writeErr
		}
		if writeErr := tx.Commit(ctx); writeErr != nil {
			return nil, writeErr
		}
		if attempts == 4 {
			return nil, ErrCodeExpired
		}
		return nil, ErrCodeInvalid
	}
	// Hash verification can cross an expiry boundary; check both leases again.
	if writeErr := tx.QueryRow(ctx, `SELECT c.expires_at>clock_timestamp() AND s.expire_date>clock_timestamp() FROM account_email_challenges c JOIN sessions s ON s.user_id=c.user_id::text WHERE c.user_id=$1 AND s.session_key=$2`, userID, sessionID).Scan(&active); writeErr != nil {
		return nil, writeErr
	}
	if !active {
		return nil, ErrCodeExpired
	}
	if _, writeErr := tx.Exec(ctx, `UPDATE users SET email=$2,is_email_verified=true,is_email_valid=true,updated_at=clock_timestamp() WHERE id=$1`, userID, email); writeErr != nil {
		var duplicate *pgconn.PgError
		if errors.As(writeErr, &duplicate) && duplicate.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, writeErr
	}
	if _, writeErr := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, userID.String()); writeErr != nil {
		return nil, writeErr
	}
	if _, writeErr := tx.Exec(ctx, `DELETE FROM account_email_challenges WHERE user_id=$1`, userID); writeErr != nil {
		return nil, writeErr
	}
	for _, recipient := range []string{current, email} {
		if writeErr := queueAccountEmail(ctx, tx, "plane.bgtasks.user_email_update_task.send_email_update_confirmation", recipient); writeErr != nil {
			return nil, writeErr
		}
	}
	identity, err := useridentity.NewRepository(tx)
	if err != nil {
		return nil, err
	}
	result, err := identity.SessionUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return result, tx.Commit(ctx)
}

func (repository *Repository) lockEmailAccount(ctx context.Context, tx pgx.Tx, userID uuid.UUID, sessionID, email string) (string, error) {
	var current string
	if err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id=$1 AND is_active FOR UPDATE`, userID).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrUnauthenticated
		}
		return "", err
	}
	var locked string
	if lockErr := tx.QueryRow(ctx, `SELECT session_key FROM sessions WHERE user_id=$1 AND session_key=$2 FOR UPDATE`, userID.String(), sessionID).Scan(&locked); lockErr != nil {
		if errors.Is(lockErr, pgx.ErrNoRows) {
			return "", ErrUnauthenticated
		}
		return "", lockErr
	}
	principal, authErr := repository.sessions.ValidateInDatabase(ctx, tx, sessionID)
	if authErr != nil || principal.UserID != userID.String() {
		return "", ErrUnauthenticated
	}
	var active bool
	if leaseErr := tx.QueryRow(ctx, `SELECT expire_date>clock_timestamp() FROM sessions WHERE session_key=$1`, sessionID).Scan(&active); leaseErr != nil {
		return "", leaseErr
	}
	if !active {
		return "", ErrUnauthenticated
	}

	if current == email {
		return "", ErrEmailUnchanged
	}
	var taken bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email=$1 AND id<>$2)`, email, userID).Scan(&taken); err != nil {
		return "", err
	}
	if taken {
		return "", ErrEmailTaken
	}
	return current, nil
}

func queueAccountEmail(ctx context.Context, tx pgx.Tx, name string, args ...string) error {
	values := make([]any, len(args))
	for i, value := range args {
		values[i] = value
	}
	return (projectmutations.Effects{Tx: tx}).Email(ctx, name, values)
}

func validChallengeCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, digit := range code {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}
