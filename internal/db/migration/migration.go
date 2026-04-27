package migration

import "embed"

//go:embed *.sql
var MigrationFS embed.FS
