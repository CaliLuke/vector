//go:build integration

package useraccountwrites

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
	"github.com/CaliLuke/go-plane/internal/useridentity"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type emailFixture struct {
	ctx                                    context.Context
	pool                                   *pgxpool.Pool
	repo                                   *Repository
	user, foreign                          uuid.UUID
	session, secondSession, foreignSession string
}

func newEmailFixture(t *testing.T) emailFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	database, err := migratedSchema.CloneDatabase(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		require.NoError(t, database.Close(cleanup))
	})
	pool, err := pgxpool.New(ctx, database.URL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	user, foreign := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO users (password,id,username,email,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ('!', $1, 'account-proof', 'original@example.test', 'Account proof', '', '', '', NOW(),NOW(),NOW(), '', '', false, false, false, true, false, true, false, false, '', '', '', '', '', false, 'UTC', true)`, user)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users SELECT (jsonb_populate_record(NULL::users,to_jsonb(source)||jsonb_build_object('id',$2::uuid,'username','foreign','email','foreign@example.test'))).* FROM users source WHERE id=$1`, user, foreign)
	require.NoError(t, err)

	identity, err := useridentity.NewRepository(pool)
	require.NoError(t, err)
	validator, err := browsersession.New(browsersession.NewPostgreSQLReader(pool), browsersession.Config{SecretKey: "account-regression-secret"})
	require.NoError(t, err)
	session, err := validator.Login(ctx, nil, browsersession.AuthenticatedSession{UserID: user.String(), PasswordHash: "!"}, 3600, time.Now())
	require.NoError(t, err)
	second, err := validator.Login(ctx, nil, browsersession.AuthenticatedSession{UserID: user.String(), PasswordHash: "!"}, 3600, time.Now())
	require.NoError(t, err)
	other, err := validator.Login(ctx, nil, browsersession.AuthenticatedSession{UserID: foreign.String(), PasswordHash: "!"}, 3600, time.Now())
	require.NoError(t, err)
	repo, err := NewRepository(pool, identity, validator)
	require.NoError(t, err)
	return emailFixture{ctx: ctx, pool: pool, repo: repo, user: user, foreign: foreign, session: session.SessionKey, secondSession: second.SessionKey, foreignSession: other.SessionKey}
}

type emailState struct {
	Email                                   string
	Sessions, Challenges, Effects, Attempts int
}

func (f emailFixture) state(t *testing.T) emailState {
	t.Helper()
	var s emailState
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT email,(SELECT count(*) FROM sessions WHERE user_id=$1::text),(SELECT count(*) FROM account_email_challenges WHERE user_id=$1::uuid),(SELECT count(*) FROM work_item_effects),COALESCE((SELECT attempts FROM account_email_challenges WHERE user_id=$1::uuid),0) FROM users WHERE id=$1::uuid`, f.user.String()).Scan(&s.Email, &s.Sessions, &s.Challenges, &s.Effects, &s.Attempts))
	return s
}
func (f emailFixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	_, err := f.pool.Exec(f.ctx, sql, args...)
	require.NoError(t, err)
}

func TestEmailChangeRollsBackEveryEffectAndCanRetry(t *testing.T) {
	f := newEmailFixture(t)
	f.exec(t, `UPDATE users SET is_email_valid=false WHERE id=$1`, f.user)
	require.NoError(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "new@example.test", "123456"))
	before := f.state(t)
	require.Equal(t, emailState{"original@example.test", 2, 1, 1, 0}, before)
	f.exec(t, `CREATE FUNCTION reject_confirmation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.payload->>'name'='plane.bgtasks.user_email_update_task.send_email_update_confirmation' THEN RAISE EXCEPTION 'notification storage unavailable'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_confirmation BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_confirmation()`)
	_, err := f.repo.ChangeEmail(f.ctx, f.user, f.session, "new@example.test", "123456")
	require.Error(t, err)
	require.Equal(t, before, f.state(t), "the email, proof, sessions and effects must all roll back")
	f.exec(t, `DROP TRIGGER reject_confirmation ON work_item_effects`)
	f.exec(t, `UPDATE users SET user_timezone='invalid/account-zone' WHERE id=$1`, f.user)
	_, err = f.repo.ChangeEmail(f.ctx, f.user, f.session, "new@example.test", "123456")
	require.Error(t, err)
	require.Equal(t, before, f.state(t), "response projection must not commit an incomplete change")
	f.exec(t, `UPDATE users SET user_timezone='UTC' WHERE id=$1`, f.user)

	body, err := f.repo.ChangeEmail(f.ctx, f.user, f.session, "new@example.test", "123456")
	require.NoError(t, err)
	var result struct {
		Email    string
		Verified bool `json:"is_email_verified"`
	}
	require.NoError(t, json.Unmarshal(body, &result))
	require.Equal(t, "new@example.test", result.Email)
	require.True(t, result.Verified)
	var deliverable bool
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT is_email_valid FROM users WHERE id=$1`, f.user).Scan(&deliverable))
	require.True(t, deliverable, "a verified replacement address must not inherit the old mailbox's delivery failure")

	require.Equal(t, emailState{"new@example.test", 0, 0, 3, 0}, f.state(t))
	var foreignSessions int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM sessions WHERE user_id=$1`, f.foreign.String()).Scan(&foreignSessions))
	require.Equal(t, 1, foreignSessions)
	_, err = f.repo.ChangeEmail(f.ctx, f.user, f.session, "new@example.test", "123456")
	require.ErrorIs(t, err, ErrUnauthenticated)
	require.Equal(t, 3, f.state(t).Effects)
}

func TestEmailChallengeAdmissionIsAtomicAndRequiresCurrentSession(t *testing.T) {
	f := newEmailFixture(t)
	require.ErrorIs(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "original@example.test", "123456"), ErrEmailUnchanged)
	require.ErrorIs(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "foreign@example.test", "123456"), ErrEmailTaken)
	require.ErrorIs(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "new@example.test", "bad"), ErrInvalid)

	f.exec(t, `CREATE FUNCTION reject_email() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'email outbox unavailable'; END $$; CREATE TRIGGER reject_email BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_email()`)
	require.Error(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "new@example.test", "123456"))
	require.Equal(t, emailState{"original@example.test", 2, 0, 0, 0}, f.state(t))
	f.exec(t, `DROP TRIGGER reject_email ON work_item_effects`)
	require.ErrorIs(t, f.repo.QueueEmailCode(f.ctx, f.user, f.foreignSession, "new@example.test", "123456"), ErrUnauthenticated)
	require.NoError(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "new@example.test", "123456"))
	_, err := f.repo.ChangeEmail(f.ctx, f.foreign, f.foreignSession, "new@example.test", "123456")
	require.ErrorIs(t, err, ErrCodeExpired)
	f.exec(t, `UPDATE users SET is_active=false WHERE id=$1`, f.user)
	_, err = f.repo.ChangeEmail(f.ctx, f.user, f.session, "new@example.test", "123456")
	require.ErrorIs(t, err, ErrUnauthenticated)
	require.ErrorIs(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "other@example.test", "654321"), ErrUnauthenticated)
	require.Equal(t, emailState{"original@example.test", 2, 1, 1, 0}, f.state(t))
	f.exec(t, `UPDATE users SET is_active=true WHERE id=$1`, f.user)
	f.exec(t, `UPDATE account_email_challenges SET expires_at=clock_timestamp()-interval '1 second'`)
	_, err = f.repo.ChangeEmail(f.ctx, f.user, f.session, "new@example.test", "123456")
	require.ErrorIs(t, err, ErrCodeExpired)
	f.exec(t, `UPDATE sessions SET expire_date=clock_timestamp()-interval '1 second' WHERE user_id=$1`, f.user.String())
	require.ErrorIs(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "new@example.test", "654321"), ErrUnauthenticated)
	require.Equal(t, emailState{"original@example.test", 2, 1, 1, 0}, f.state(t))
}

func TestEmailChallengeGuessBudgetAndReplacementAreAtomic(t *testing.T) {
	f := newEmailFixture(t)
	require.NoError(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "new@example.test", "123456"))
	start := make(chan struct{})
	results := make(chan error, 10)
	for range 10 {
		go func() {
			<-start
			_, err := f.repo.ChangeEmail(f.ctx, f.user, f.session, "new@example.test", "999999")
			results <- err
		}()
	}
	close(start)
	invalid, expired := 0, 0
	for range 10 {
		err := <-results
		switch {
		case errors.Is(err, ErrCodeInvalid):
			invalid++
		case errors.Is(err, ErrCodeExpired):
			expired++
		default:
			require.NoError(t, err)
		}
	}
	require.Equal(t, 4, invalid)
	require.Equal(t, 6, expired)
	require.Equal(t, 5, f.state(t).Attempts)
	_, err := f.repo.ChangeEmail(f.ctx, f.user, f.session, "new@example.test", "123456")
	require.ErrorIs(t, err, ErrCodeExpired)
	require.NoError(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "other@example.test", "654321"))
	_, err = f.repo.ChangeEmail(f.ctx, f.user, f.session, "new@example.test", "123456")
	require.ErrorIs(t, err, ErrCodeExpired)
	require.Zero(t, f.state(t).Attempts)
	_, err = f.repo.ChangeEmail(f.ctx, f.user, f.session, "other@example.test", "654321")
	require.NoError(t, err)
	require.Equal(t, "other@example.test", f.state(t).Email)
}

func TestConcurrentEmailChangeConsumesCodeOnce(t *testing.T) {
	f := newEmailFixture(t)
	require.NoError(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "new@example.test", "123456"))
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, session := range []string{f.session, f.secondSession} {
		go func() {
			<-start
			_, err := f.repo.ChangeEmail(f.ctx, f.user, session, "new@example.test", "123456")
			results <- err
		}()
	}
	close(start)
	first, second := <-results, <-results
	require.True(t, first == nil && errors.Is(second, ErrUnauthenticated) || second == nil && errors.Is(first, ErrUnauthenticated), "one complete change must win: %v / %v", first, second)
	require.Equal(t, emailState{"new@example.test", 0, 0, 3, 0}, f.state(t))
}

func TestConcurrentAccountsCannotClaimTheSameEmail(t *testing.T) {
	f := newEmailFixture(t)
	require.NoError(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "claimed@example.test", "123456"))
	require.NoError(t, f.repo.QueueEmailCode(f.ctx, f.foreign, f.foreignSession, "claimed@example.test", "654321"))
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, err := f.repo.ChangeEmail(f.ctx, f.user, f.session, "claimed@example.test", "123456")
		results <- err
	}()
	go func() {
		<-start
		_, err := f.repo.ChangeEmail(f.ctx, f.foreign, f.foreignSession, "claimed@example.test", "654321")
		results <- err
	}()
	close(start)
	first, second := <-results, <-results
	require.True(t, first == nil && errors.Is(second, ErrEmailTaken) || second == nil && errors.Is(first, ErrEmailTaken), "one owner must win the unique address: %v / %v", first, second)
	var changed, proofs, effects int
	var retainedIdentity, sessionsRetained bool
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM users WHERE email='claimed@example.test'),(SELECT count(*) FROM account_email_challenges),(SELECT count(*) FROM work_item_effects),(SELECT bool_and(c.current_email=u.email) FROM account_email_challenges c JOIN users u ON u.id=c.user_id),(SELECT bool_and(u.email<>'claimed@example.test') FROM sessions s JOIN users u ON u.id::text=s.user_id)`).Scan(&changed, &proofs, &effects, &retainedIdentity, &sessionsRetained))
	require.Equal(t, 1, changed)
	require.Equal(t, 1, proofs)
	require.Equal(t, 4, effects)
	require.True(t, retainedIdentity, "the rejected account must retain its original identity and proof")
	require.True(t, sessionsRetained, "only the winning account's sessions are revoked")
}

func TestPasswordRotationRejectsPreviouslyAuthenticatedEmailSession(t *testing.T) {
	f := newEmailFixture(t)
	require.NoError(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "new@example.test", "123456"))
	before := f.state(t)
	f.exec(t, `UPDATE users SET password='rotated-password' WHERE id=$1`, f.user)
	_, err := f.repo.ChangeEmail(f.ctx, f.user, f.session, "new@example.test", "123456")
	require.ErrorIs(t, err, ErrUnauthenticated)
	require.ErrorIs(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "other@example.test", "654321"), ErrUnauthenticated)
	require.Equal(t, before, f.state(t))
}
