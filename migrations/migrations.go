// Package migrations embeds the SQL migration files for use with the migration runner.
package migrations

import "embed"

// FS contains all .sql migration files embedded in the binary.
//
//go:embed *.sql
var FS embed.FS
