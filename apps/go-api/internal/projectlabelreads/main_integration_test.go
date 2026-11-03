//go:build integration

package projectlabelreads

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var labelDatabases = postgres.NewTemplate(func(context.Context, string) error { return nil })
var testTemplates = []*postgres.Template{labelDatabases}

func TestMain(m *testing.M) {
	status := m.Run()
	for _, template := range testTemplates {
		if err := template.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close projectlabelreads test template:", err)
			status = 1
		}
	}
	os.Exit(status)
}
