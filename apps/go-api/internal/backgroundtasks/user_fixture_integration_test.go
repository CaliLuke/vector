//go:build integration

package backgroundtasks

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func seedBackgroundTaskUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, username, firstName, lastName, email string, at time.Time) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO users
 (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,
 last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,
 is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,
 last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid,email)
 VALUES ($1,'!',$2,$3 || ' ' || $4,$3,$4,'',$5,$5,$5,'','',false,false,false,true,false,
 true,false,false,$2,'','','','',false,'UTC',true,NULLIF($6,''))`, id, username, firstName, lastName, at, email)
	require.NoError(t, err, "seed background task user")
}
