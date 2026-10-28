//go:build integration

package mcpactions_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var milestoneSchema = postgresfixture.NewTemplate(bootstrap.MigrateSchema)

func TestMain(m *testing.M) {
	status := m.Run()
	if err := milestoneSchema.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "close MCP test template:", err)
		status = 1
	}
	os.Exit(status)
}
