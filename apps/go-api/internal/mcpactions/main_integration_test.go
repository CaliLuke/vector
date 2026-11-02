//go:build integration

package mcpactions_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

var milestoneSchema = postgresfixture.NewTemplate(func(ctx context.Context, databaseURL string) error {
	if err := bootstrap.MigrateSchema(ctx, databaseURL); err != nil {
		return err
	}
	return seedMCPDatabase(ctx, databaseURL, testsupport.MilestoneWriteSeed+`
UPDATE workspaces SET slug='workspace-one';
INSERT INTO api_tokens(id,created_at,updated_at,token,label,user_type,user_id,description,is_active,is_service,allowed_rate_limit)
VALUES(gen_random_uuid(),now(),now(),'test-one','MCP test',0,'00000000-0000-4000-8000-000000000003','',true,false,'100/minute');`)
})
var identitySchemaTemplate = postgresfixture.NewTemplate(func(ctx context.Context, databaseURL string) error {
	return seedMCPDatabase(ctx, databaseURL, identitySchema)
})

func seedMCPDatabase(ctx context.Context, databaseURL, seed string) (err error) {
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		err = errors.Join(err, connection.Close(closeCtx))
	}()
	_, err = connection.Exec(ctx, seed)
	return err
}

func TestMain(m *testing.M) {
	status := m.Run()
	if err := errors.Join(milestoneSchema.Close(), identitySchemaTemplate.Close()); err != nil {
		fmt.Fprintln(os.Stderr, "close MCP test templates:", err)
		status = 1
	}
	os.Exit(status)
}
