//go:build integration

package mcpactions_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

var milestoneSchema = postgresfixture.NewTemplate(bootstrap.MigrateSchema)
var identitySchemaTemplate = postgresfixture.NewTemplate(func(ctx context.Context, databaseURL string) error {
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()
	_, err = connection.Exec(ctx, identitySchema)
	return err
})

func TestMain(m *testing.M) {
	status := m.Run()
	if err := errors.Join(milestoneSchema.Close(), identitySchemaTemplate.Close()); err != nil {
		fmt.Fprintln(os.Stderr, "close MCP test templates:", err)
		status = 1
	}
	os.Exit(status)
}
