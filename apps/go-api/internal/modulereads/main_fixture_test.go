//go:build integration

package modulereads

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var suiteDatabases = postgres.NewTemplate(func(context.Context, string) error { return nil })

func TestMain(m *testing.M) {
	status := m.Run()
	for _, template := range []*postgres.Template{suiteDatabases} {
		if err := template.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close modulereads test template:", err)
			status = 1
		}
	}
	os.Exit(status)
}
