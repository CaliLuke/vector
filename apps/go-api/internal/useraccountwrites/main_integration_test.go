//go:build integration

package useraccountwrites

import (
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var migratedSchema = postgres.NewTemplate(bootstrap.MigrateSchema)

func TestMain(m *testing.M) {
	status := m.Run()
	if err := migratedSchema.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "close useraccountwrites test template:", err)
		status = 1
	}
	os.Exit(status)
}
