//go:build integration

package issuequery_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/issuequery"
)

func TestMain(m *testing.M) {
	status := m.Run()
	if err := issuequery.QueryTestDatabases.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "close issuequery test template:", err)
		status = 1
	}
	os.Exit(status)
}
