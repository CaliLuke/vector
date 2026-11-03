//go:build integration

package projectmembershipreads

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var membershipDatabases = postgres.NewTemplate(func(context.Context, string) error { return nil })

var migratedSchema = postgres.NewTemplate(bootstrap.MigrateSchema)

func TestMain(m *testing.M) {
	status := m.Run()
	if err := errors.Join(migratedSchema.Close(), membershipDatabases.Close()); err != nil {
		fmt.Fprintln(os.Stderr, "close projectmembershipreads test template:", err)
		status = 1
	}
	os.Exit(status)
}
