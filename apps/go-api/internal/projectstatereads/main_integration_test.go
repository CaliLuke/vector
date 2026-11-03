//go:build integration

package projectstatereads

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var stateDatabases = postgres.NewTemplate(func(context.Context, string) error { return nil })
var stateSchema = postgres.NewTemplate(bootstrap.MigrateSchema)
var testTemplates = []*postgres.Template{stateDatabases, stateSchema}

func TestMain(m *testing.M) {
	status := m.Run()
	for _, template := range testTemplates {
		if err := template.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close projectstatereads test template:", err)
			status = 1
		}
	}
	os.Exit(status)
}
