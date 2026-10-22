// Package support contains the existing Go support schema migrations.
package support

import "embed"

// Files embeds the same migrations used by support-migrate.
//
//go:embed *.sql
var Files embed.FS
