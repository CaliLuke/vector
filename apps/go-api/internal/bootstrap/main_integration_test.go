//go:build integration

package bootstrap_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

// Schema behavior tests get independent clones. Fresh installation tests still
// start with empty databases and run the migrations themselves.
var migratedSchema = postgres.NewTemplate(bootstrap.MigrateSchema)

func TestMain(m *testing.M) {
	status := m.Run()
	if err := migratedSchema.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "close bootstrap test template:", err)
		status = 1
	}
	os.Exit(status)
}
