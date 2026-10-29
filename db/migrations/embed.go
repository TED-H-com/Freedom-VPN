// Package migrations contains forward-only PostgreSQL migrations (§15.5).
package migrations

import "embed"

// FS is the immutable migration source shared by runtime and integration tests.
//
//go:embed *.sql
var FS embed.FS
