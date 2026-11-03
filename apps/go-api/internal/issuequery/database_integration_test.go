//go:build integration

package issuequery

import (
	"context"

	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

// QueryTestDatabases is shared with external-package integration tests. Every
// caller gets an independent database; TestMain owns the container lifetime.
var QueryTestDatabases = postgres.NewTemplate(func(context.Context, string) error { return nil })
