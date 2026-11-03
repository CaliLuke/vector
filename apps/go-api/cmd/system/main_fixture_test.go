//go:build integration

package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var migratedDatabases = postgres.NewTemplate(bootstrap.MigrateSchema)

func TestMain(m *testing.M) {
	status := m.Run()
	for _, template := range []*postgres.Template{migratedDatabases} {
		if err := template.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close main test template:", err)
			status = 1
		}
	}
	os.Exit(status)
}
