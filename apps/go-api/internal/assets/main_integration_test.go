//go:build integration

package assets

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var assetDatabases = postgres.NewTemplate(func(context.Context, string) error { return nil })
var testTemplates = []*postgres.Template{assetDatabases}

func TestMain(m *testing.M) {
	status := m.Run()
	for _, template := range testTemplates {
		if err := template.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close assets test template:", err)
			status = 1
		}
	}
	os.Exit(status)
}
