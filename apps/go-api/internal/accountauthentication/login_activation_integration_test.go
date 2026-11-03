//go:build integration

package accountauthentication

import (
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestLoginActivationCommitsEmailIntentWithAccountMetadata(t *testing.T) {
	ctx, db, repository := openAccountRepository(t)
	user, err := repository.CreateUser(ctx, "activation@example.test", "activation", "!", false, false)
	require.NoError(t, err)
	_, err = db.Exec(ctx, `UPDATE users SET is_active=false,token='before' WHERE id=$1`, user.ID)
	require.NoError(t, err)
	_, err = db.Exec(ctx, `ALTER TABLE work_item_effects ADD CONSTRAINT reject_activation CHECK (false) NOT VALID`)
	require.NoError(t, err)
	input := LoginMetadata{UserID: user.ID, At: time.Now().UTC(), Medium: "github", ActivationOrigin: "https://vector.example.test"}
	_, err = repository.UpdateLoginMetadata(ctx, input)
	require.Error(t, err, "activation must fail when its durable email cannot be recorded")
	var active bool
	var token string
	require.NoError(t, db.QueryRow(ctx, `SELECT is_active,token FROM users WHERE id=$1`, user.ID).Scan(&active, &token))
	require.False(t, active, "failed admission must leave activation retryable")
	require.Equal(t, "before", token)
	_, err = db.Exec(ctx, `ALTER TABLE work_item_effects DROP CONSTRAINT reject_activation`)
	require.NoError(t, err)
	second, err := pgx.ConnectConfig(ctx, db.Config().Copy())
	require.NoError(t, err)
	t.Cleanup(func() { testsupport.CloseContext(t, second, ctx) })
	secondRepository, err := NewRepository(second, nil, false)
	require.NoError(t, err)
	type loginResult struct {
		activated bool
		err       error
	}
	start, results := make(chan struct{}), make(chan loginResult, 2)
	for _, accountRepository := range []*Repository{repository, secondRepository} {
		go func() {
			<-start
			activated, loginErr := accountRepository.UpdateLoginMetadata(ctx, input)
			results <- loginResult{activated, loginErr}
		}()
	}
	close(start)
	first, repeated := <-results, <-results
	require.NoError(t, first.err)
	require.NoError(t, repeated.err)
	require.NotEqual(t, first.activated, repeated.activated, "concurrent first logins must activate exactly once")
	var count int
	var payload []byte
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM work_item_effects`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, db.QueryRow(ctx, `SELECT payload FROM work_item_effects WHERE kind='email_task'`).Scan(&payload))
	require.JSONEq(t, `{"name":"plane.bgtasks.user_activation_email_task.user_activation_email","args":["https://vector.example.test","`+user.ID+`"]}`, string(payload))
}
