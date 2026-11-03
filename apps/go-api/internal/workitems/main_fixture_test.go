//go:build integration

package workitems

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var suiteDatabases = postgres.NewTemplate(func(context.Context, string) error { return nil })
var migratedDatabases = postgres.NewTemplate(bootstrap.MigrateSchema)

func TestMain(m *testing.M) {
	status := m.Run()
	for _, template := range []*postgres.Template{suiteDatabases, migratedDatabases} {
		if err := template.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close workitems test template:", err)
			status = 1
		}
	}
	os.Exit(status)
}
