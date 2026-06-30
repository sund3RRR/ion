package migrations

import "embed"

// FS contains embedded SQLite migrations for the ION store database.
//
//go:embed *.sql
var FS embed.FS
