//go:build integration

package workspaceprojectreads_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var projectDatabases = postgres.NewTemplate(func(context.Context, string) error { return nil })
var testTemplates = []*postgres.Template{projectDatabases}

func TestMain(m *testing.M) {
	status := m.Run()
	for _, template := range testTemplates {
		if err := template.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close workspaceprojectreads_test test template:", err)
			status = 1
		}
	}
	os.Exit(status)
}
