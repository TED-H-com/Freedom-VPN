package db_test

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/pressly/goose/v3"
)

// Keep in sync with internal/nodes.TestSECOPS19_D077_NodeDetailUsageOutboxProjectionAndMigration.
// That test fixes the oldest version reached by the migration round trip.
const migrationDownFloor int64 = 188

func Test15_5_MigrationDownLint(t *testing.T) {
	names, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		version, err := goose.NumericComponent(name)
		if err != nil {
			t.Errorf("%s: invalid migration number", name)
			continue
		}
		if version <= migrationDownFloor {
			continue
		}
		source, err := migrations.FS.ReadFile(name)
		if err != nil {
			t.Fatalf("%s: cannot read migration: %v", name, err)
		}
		if reason := migrationDownReason(string(source)); reason != "" {
			t.Errorf("%s: %s", name, reason)
		}
	}
}

// This is a presence check, not a SQL validator. Ignore comments and quoted
// content; the integration round trip proves that Down and the next Up execute.
func migrationDownReason(source string) string {
	dollarQuote := regexp.MustCompile(`^\$(?:[A-Za-z_][A-Za-z_0-9]*)?\$`)
	down, sql := false, false
	for i := 0; i < len(source); {
		switch {
		case strings.HasPrefix(source[i:], "--"):
			end := strings.IndexByte(source[i:], '\n')
			if end < 0 {
				end = len(source) - i
			}
			lineStart := strings.LastIndexByte(source[:i], '\n') + 1
			if strings.TrimSpace(source[lineStart:i]) == "" && strings.TrimSpace(source[i:i+end]) == "-- +goose Down" {
				down, sql = true, false
			}
			i += end
		case strings.HasPrefix(source[i:], "/*"):
			i += 2
			for depth := 1; i < len(source) && depth > 0; {
				switch {
				case strings.HasPrefix(source[i:], "/*"):
					depth++
					i += 2
				case strings.HasPrefix(source[i:], "*/"):
					depth--
					i += 2
				default:
					i++
				}
			}
		case source[i] == '\'' || source[i] == '"':
			quote := source[i]
			i++
			for i < len(source) {
				c := source[i]
				i++
				if c == quote {
					if i < len(source) && source[i] == quote {
						i++
						continue
					}
					break
				}
			}
		case source[i] == '$':
			delimiter := dollarQuote.FindString(source[i:])
			if delimiter == "" {
				i++
				continue
			}
			i += len(delimiter)
			end := strings.Index(source[i:], delimiter)
			if end < 0 {
				i = len(source)
			} else {
				i += end + len(delimiter)
			}
		default:
			if down && sql && source[i] == ';' {
				return ""
			}
			if down && ((source[i] >= 'A' && source[i] <= 'Z') || (source[i] >= 'a' && source[i] <= 'z')) {
				sql = true
			}
			i++
		}
	}
	if !down {
		return "missing -- +goose Down"
	}
	return "-- +goose Down has no SQL statement ending in a semicolon"
}

func Test15_5_MigrationDownLintCases(t *testing.T) {
	for _, tt := range []struct {
		name, source, want string
	}{
		{"missing", "-- +goose Up\nSELECT 1;\n", "missing -- +goose Down"},
		{"empty", "-- +goose Down\n", "-- +goose Down has no SQL statement ending in a semicolon"},
		{"comments", "-- +goose Down\n-- DROP TABLE example;\n/* DROP TABLE example; /* nested */ */\n;", "-- +goose Down has no SQL statement ending in a semicolon"},
		{"commented marker", "/*\n-- +goose Down\n*/\nDROP TABLE example;", "missing -- +goose Down"},
		{"unfinished", "-- +goose Down\nDROP TABLE example -- ;\n", "-- +goose Down has no SQL statement ending in a semicolon"},
		{"quoted semicolon", "-- +goose Down\nSELECT ';'", "-- +goose Down has no SQL statement ending in a semicolon"},
		{"dollar quoted marker", "-- +goose Up\nSELECT $body$\n-- +goose Down\nDROP TABLE example;\n$body$;", "missing -- +goose Down"},
		{"multiline", "-- +goose Down\n-- reverse Up\nDROP TABLE\nexample; -- done\n", ""},
		{"quoted comment", "-- +goose Down\nSELECT '--', 'it''s /* quoted */';", ""},
		{"statement block", "-- +goose Down\n-- +goose StatementBegin\nDO $$ BEGIN PERFORM 1; END $$;\n-- +goose StatementEnd\n", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := migrationDownReason(tt.source); got != tt.want {
				t.Fatalf("reason = %q, want %q", got, tt.want)
			}
		})
	}
}
