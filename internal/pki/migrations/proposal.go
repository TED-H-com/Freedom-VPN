// Package migrations holds the reviewable reinstall migration for the database
// owner. It is deliberately never executed by a runtime service.
package migrations

import _ "embed"

//go:embed reinstall.sql
var Reinstall string
