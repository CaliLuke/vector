package mcpoauth

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidFlow  = errors.New("invalid or expired account linking flow")
	ErrLinkConflict = errors.New("account link cannot be confirmed")
)

type linkDatabase interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

// Links keeps account linking on the primary database. Issuer is immutable and
// comes from configuration, never a tool parameter or browser payload.
type Links struct {
	primary linkDatabase
	issuer  string
}

func NewLinks(primary linkDatabase, issuer string) (*Links, error) {
	if primary == nil || !validEndpoint(issuer) {
		return nil, ErrInvalidConfiguration
	}
	return &Links{primary: primary, issuer: issuer}, nil
}

type LinkedUser struct {
	ID       uuid.UUID
	Timezone string
}

func (s *Links) Resolve(ctx context.Context, subject string) (LinkedUser, error) {
	var result LinkedUser
	err := s.primary.QueryRow(ctx, `SELECT u.id,u.user_timezone
 FROM mcp_oauth_links l JOIN users u ON u.id=l.user_id
 WHERE l.issuer=$1 AND l.subject=$2 AND u.is_active AND NOT u.is_bot`, s.issuer, subject).Scan(&result.ID, &result.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return LinkedUser{}, ErrInvalidCredential
	}
	return result, err
}

type PendingLink struct {
	ID        uuid.UUID
	Subject   string
	ExpiresAt time.Time
}

type LinkStatus struct {
	Issuer    string
	Connected bool
	Subject   string
	Pending   *PendingLink
}

// Status does not reveal a pending issuer identity without both the initiating
// session and proof received only by the successful callback browser.
func (s *Links) Status(ctx context.Context, user uuid.UUID, session, proof [32]byte) (LinkStatus, error) {
	result := LinkStatus{Issuer: s.issuer}
	var subject *string
	var pendingID *uuid.UUID
	var pendingSubject *string
	var pendingExpiry *time.Time
	err := s.primary.QueryRow(ctx, `SELECT l.subject,f.id,f.subject,f.expires_at
 FROM users u
 LEFT JOIN mcp_oauth_links l ON l.user_id=u.id AND l.issuer=$2
 LEFT JOIN mcp_oauth_link_flows f ON f.user_id=u.id AND f.issuer=$2
  AND f.phase='verified' AND f.session_hash=$3 AND f.proof_hash=$4 AND f.expires_at>clock_timestamp()
 WHERE u.id=$1 AND u.is_active AND NOT u.is_bot`, user, s.issuer, session[:], proof[:]).Scan(&subject, &pendingID, &pendingSubject, &pendingExpiry)
	if errors.Is(err, pgx.ErrNoRows) {
		return LinkStatus{}, ErrInvalidCredential
	}
	if err != nil {
		return LinkStatus{}, err
	}
	if subject != nil {
		result.Connected = true
		result.Subject = *subject
	}
	if pendingID != nil && pendingSubject != nil && pendingExpiry != nil {
		result.Pending = &PendingLink{ID: *pendingID, Subject: *pendingSubject, ExpiresAt: *pendingExpiry}
	}
	return result, nil
}

type StartLink struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	SessionHash        [32]byte
	StateHash          [32]byte
	NonceHash          [32]byte
	VerifierCiphertext string
}

func (s *Links) Start(ctx context.Context, input StartLink) error {
	if input.ID == uuid.Nil || input.UserID == uuid.Nil || input.VerifierCiphertext == "" {
		return ErrInvalidFlow
	}
	return pgx.BeginFunc(ctx, s.primary, func(tx pgx.Tx) error {
		if err := lockLinkUser(ctx, tx, input.UserID); err != nil {
			return err
		}
		var connected bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM mcp_oauth_links WHERE issuer=$1 AND user_id=$2)`, s.issuer, input.UserID).Scan(&connected); err != nil {
			return err
		}
		if connected {
			return ErrLinkConflict
		}
		if _, err := tx.Exec(ctx, `DELETE FROM mcp_oauth_link_flows WHERE issuer=$1 AND user_id=$2`, s.issuer, input.UserID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO mcp_oauth_link_flows
 (id,issuer,user_id,session_hash,state_hash,nonce_hash,verifier_ciphertext,phase,created_at,expires_at)
 VALUES($1,$2,$3,$4,$5,$6,$7,'pending',statement_timestamp(),statement_timestamp()+INTERVAL '10 minutes')`,
			input.ID, s.issuer, input.UserID, input.SessionHash[:], input.StateHash[:], input.NonceHash[:], input.VerifierCiphertext)
		return err
	})
}

type LinkExchange struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	NonceHash          [32]byte
	VerifierCiphertext string
}

// Consume serializes code exchange admission. A callback replay can never make
// a second exchange, including after an issuer timeout or lost response.
func (s *Links) Consume(ctx context.Context, state [32]byte) (LinkExchange, error) {
	var result LinkExchange
	var nonce []byte
	err := s.primary.QueryRow(ctx, `UPDATE mcp_oauth_link_flows f SET phase='exchanging'
 WHERE issuer=$1 AND state_hash=$2 AND phase='pending' AND expires_at>clock_timestamp()
 AND EXISTS(SELECT 1 FROM users u WHERE u.id=f.user_id AND u.is_active AND NOT u.is_bot)
 RETURNING id,user_id,nonce_hash,verifier_ciphertext`, s.issuer, state[:]).Scan(&result.ID, &result.UserID, &nonce, &result.VerifierCiphertext)
	if errors.Is(err, pgx.ErrNoRows) {
		return LinkExchange{}, ErrInvalidFlow
	}
	if err != nil {
		return LinkExchange{}, err
	}
	if len(nonce) != 32 {
		return LinkExchange{}, ErrInvalidFlow
	}
	copy(result.NonceHash[:], nonce)
	return result, nil
}

func (s *Links) Verified(ctx context.Context, id uuid.UUID, subject string, proof [32]byte) (time.Time, error) {
	if subject == "" {
		return time.Time{}, ErrInvalidFlow
	}
	var expires time.Time
	err := s.primary.QueryRow(ctx, `UPDATE mcp_oauth_link_flows f
 SET phase='verified',subject=$3,proof_hash=$4,verifier_ciphertext=NULL
 WHERE issuer=$1 AND id=$2 AND phase='exchanging' AND expires_at>clock_timestamp()
 AND EXISTS(SELECT 1 FROM users u WHERE u.id=f.user_id AND u.is_active AND NOT u.is_bot)
 RETURNING expires_at`, s.issuer, id, subject, proof[:]).Scan(&expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrInvalidFlow
	}
	return expires, err
}

func (s *Links) Fail(ctx context.Context, id uuid.UUID) error {
	_, err := s.primary.Exec(ctx, `DELETE FROM mcp_oauth_link_flows WHERE issuer=$1 AND id=$2 AND phase='exchanging'`, s.issuer, id)
	return err
}

func (s *Links) Confirm(ctx context.Context, user, id uuid.UUID, session, proof [32]byte) error {
	err := pgx.BeginFunc(ctx, s.primary, func(tx pgx.Tx) error {
		if err := lockLinkUser(ctx, tx, user); err != nil {
			return err
		}
		var subject string
		var storedSession, storedProof []byte
		err := tx.QueryRow(ctx, `SELECT subject,session_hash,proof_hash FROM mcp_oauth_link_flows
 WHERE issuer=$1 AND user_id=$2 AND id=$3 AND phase='verified' AND expires_at>clock_timestamp()
 FOR UPDATE`, s.issuer, user, id).Scan(&subject, &storedSession, &storedProof)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLinkConflict
		}
		if err != nil {
			return err
		}
		if subtle.ConstantTimeCompare(storedSession, session[:]) != 1 || subtle.ConstantTimeCompare(storedProof, proof[:]) != 1 {
			return ErrLinkConflict
		}
		if _, err = tx.Exec(ctx, `INSERT INTO mcp_oauth_links(issuer,subject,user_id) VALUES($1,$2,$3)`, s.issuer, subject, user); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `DELETE FROM mcp_oauth_link_flows WHERE id=$1`, id)
		return err
	})
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return ErrLinkConflict
	}
	return err
}

func (s *Links) Disconnect(ctx context.Context, user uuid.UUID) error {
	return pgx.BeginFunc(ctx, s.primary, func(tx pgx.Tx) error {
		if err := lockLinkUser(ctx, tx, user); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM mcp_oauth_links WHERE issuer=$1 AND user_id=$2`, s.issuer, user); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM mcp_oauth_link_flows WHERE issuer=$1 AND user_id=$2`, s.issuer, user)
		return err
	})
}

func (s *Links) PruneExpired(ctx context.Context) error {
	_, err := s.primary.Exec(ctx, `DELETE FROM mcp_oauth_link_flows WHERE id IN (
 SELECT id FROM mcp_oauth_link_flows WHERE expires_at<=clock_timestamp()
 ORDER BY expires_at,id LIMIT 100 FOR UPDATE SKIP LOCKED)`)
	return err
}

func lockLinkUser(ctx context.Context, tx pgx.Tx, user uuid.UUID) error {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT true FROM users WHERE id=$1 AND is_active AND NOT is_bot FOR UPDATE`, user).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidCredential
	}
	return err
}
