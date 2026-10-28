//go:build integration

package workitemtypes_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var typeSchema = postgresfixture.NewTemplate(bootstrap.MigrateSchema)

func TestMain(m *testing.M) {
	status := m.Run()
	if err := typeSchema.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "close work-item type test template:", err)
		status = 1
	}
	os.Exit(status)
}
