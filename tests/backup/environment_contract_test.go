package backup

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// This is a conservative review inventory, not a Bash interpreter. Inspect
// embedded helper programs too; do not discard quoted/heredoc source text.
// New spellings, including harmless uses, may need an explicit review.
var backupEnvironmentConstructs = []*regexp.Regexp{
	regexp.MustCompile(`(?:^|[\s;&|()])["']?(?:source|eval|export)["']?(?:[\s;&|()]|$)`),
	regexp.MustCompile(`(?:^|[\s;&|()])["']?\.["']?(?:\s|$)`),
	regexp.MustCompile(`\$\{!`),
	regexp.MustCompile(`\bprintf\s+["']?-v`),
	regexp.MustCompile(`\bset\s+(?:-[a-zA-Z]+\s+)*(?:-[a-zA-Z]*a\b|-o\s+["']?allexport\b)`),
	regexp.MustCompile(`\b(?:declare|typeset|local)\s+(?:-[a-zA-Z]+\s+)*-[a-zA-Z]*[nx][a-zA-Z]*\b`),
	// env accepts both literal assignments and computed assignment words.
	regexp.MustCompile(`(?:^|[\s;&|()])["']?(?:/[^\s;&|()]*/)?env["']?\s`),
	// These builtins accept variable names or executable arithmetic as data.
	regexp.MustCompile(`(?:^|[\s;&|()])["']?(?:mapfile|readarray|getopts|let)["']?(?:\s|$)`),
	// wait -p writes a process identifier into its supplied variable name.
	regexp.MustCompile(`\bwait\s+(?:-[a-zA-Z]+\s+)*-p`),
	// Loadable builtins and readline command bindings introduce executable code.
	regexp.MustCompile(`\benable\s`),
	regexp.MustCompile(`\bbind\s+(?:-[a-zA-Z]+\s+)*-x`),
	// A computed trap or builtin name can execute an indirect assignment.
	regexp.MustCompile(`\btrap\s+[^;\n]*[$` + "`" + `]`),
	regexp.MustCompile(`\bbuiltin\s`),
	regexp.MustCompile(`\bcommand\s+(?:-[a-zA-Z]+\s+)*["']?[$` + "`" + `]`),
	// Expanded arithmetic can supply an assignment's destination or expression.
	regexp.MustCompile(`\(\([^\n)]*[$` + "`" + `]`),
}

var backupNameWriter = regexp.MustCompile(`(?:^|[;&|]|\$\(|\bthen\b|\bdo\b|\bif\b|\bwhile\b|\buntil\b)\s*(?:command\s+)?["']?(declare|typeset|local|readonly|read|unset)["']?(?:\s+([^;&|<>]*))?$`)
var backupLiteralName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\+?=.*)?$`)

func backupNeedsEnvironmentReview(line string) bool {
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		return false
	}
	for _, pattern := range backupEnvironmentConstructs {
		if pattern.MatchString(line) {
			return true
		}
	}
	// A literal local/read/unset destination cannot select a locale variable
	// indirectly. Everything else (including -x/-n attributes, quoted names,
	// computed options and unfamiliar syntax) needs review. Stop at redirections
	// and command separators so read's input is not confused with its targets.
	for _, segment := range strings.FieldsFunc(line, func(r rune) bool { return strings.ContainsRune(";&|<>", r) }) {
		match := backupNameWriter.FindStringSubmatch(segment)
		if match == nil {
			continue
		}
		for _, word := range strings.Fields(match[2]) {
			if match[1] == "read" && word == "-r" {
				continue
			}
			if !backupLiteralName.MatchString(word) {
				return true
			}
		}
	}
	return false
}

func TestAT30_BackupEnvironmentWrites(t *testing.T) {
	// Exactly the six reviewed uses at c02eb001, by full line content and path.
	// Do not generate this inventory from the scripts under test.
	allowed := map[string]map[string]string{
		"deploy/backup/backup.sh": {},
		"deploy/backup/restore-pitr.sh": {
			"\texport PGBACKREST_REPO1_CIPHER_PASS":                                                  "Exports the resolved local-repository cipher passphrase by a literal name.",
			"\texport PGBACKREST_REPO1_CIPHER_TYPE=\"${PGBACKREST_REPO1_CIPHER_TYPE:-aes-256-cbc}\"": "Literal cipher type with the reviewed encryption default.",
			"\t\tsource /usr/local/bin/fvpn-postgres-entrypoint.sh":                                  "Loads the image's fixed helper implementation to rewrite recovery settings.",
		},
		"deploy/backup/verify.sh": {
			"export TMPDIR=\"$WORK_TMP\" GOTMPDIR=\"$WORK_TMP\"": "Places verifier and Go scratch in its disposable disk directory.",
		},
		"deploy/backup/postgres/fvpn-postgres-entrypoint.sh": {
			"\t\texport PGBACKREST_REPO1_CIPHER_PASS":                                                  "Exports the resolved local-repository cipher passphrase by a literal name.",
			"\t\texport PGBACKREST_REPO1_CIPHER_TYPE=\"${PGBACKREST_REPO1_CIPHER_TYPE:-aes-256-cbc}\"": "Literal cipher type with the reviewed encryption default.",
		},
	}
	count := 0
	for path, entries := range allowed {
		content, err := os.ReadFile(filepath.Join("../..", path))
		if err != nil {
			t.Fatal(err)
		}
		seen := make(map[string]int)
		lines := strings.Split(string(content), "\n")
		for i := 0; i < len(lines); i++ {
			start, line := i, lines[i]
			// Inspect continued commands as one unit; retain the physical source
			// for exact matching and the first line number for diagnostics.
			for strings.HasSuffix(line, "\\") && i+1 < len(lines) {
				i++
				line = strings.TrimSuffix(line, "\\") + lines[i]
			}
			if !backupNeedsEnvironmentReview(line) {
				continue
			}
			original := strings.Join(lines[start:i+1], "\n")
			count++
			if _, ok := entries[original]; !ok {
				t.Errorf("%s:%d: unapproved code loading or environment/name write: %s", path, start+1, original)
			} else {
				seen[original]++
				if seen[original] > 1 {
					t.Errorf("%s:%d: duplicated environment allowlist entry: %s", path, start+1, original)
				}
				t.Logf("%s:%d: reviewed environment use: %s", path, start+1, original)
			}
		}
		for line, reason := range entries {
			if seen[line] == 0 {
				t.Errorf("%s: missing fixed environment allowlist entry %q (%s)", path, line, reason)
			}
		}
	}
	if count != 6 {
		t.Errorf("environment inventory: expected 6 reviewed uses, found %d", count)
	}
}

func TestAT30_BackupEnvironmentConstructs(t *testing.T) {
	for _, line := range []string{
		`part=LC; export "${part}_ALL=C"`, `source "${FVPN_BACKUP_SETTINGS:-/dev/null}"`,
		`. "${FVPN_BACKUP_SETTINGS}"`, `eval "$FVPN_BACKUP_EXTRA"`,
		`declare -x NAME=C`, `typeset -rn target="$name"`, `local -n ref="$name"`,
		`if declare -x NAME=C; then`, `command local -xn ref="$name"`,
		`declare "$name=C"`, `local "$name"`, `readonly "$name=C"`,
		`printf -v "${part}_ALL" C`, `set -a`, `set -ea`, `set -eu -a`, `set -o allexport`,
		`env NAME=C docker ps`, `env "$assignment" docker ps`, `value=${!name}`,
		`/usr/bin/env "$assignment" docker ps`,
		`read -r "$name"`, `read -a "$name"`, `mapfile -t "$name"`, `readarray "$name"`,
		`getopts ab "$name"`, `let "$name=1"`, `(( $name = 1 ))`, `unset "$name"`,
		`wait -n -p "$name"`, `enable -f "$library" writer`, `bind -x "$binding"`,
		`trap "$code" EXIT`, `builtin "$writer" "$name=C"`, `command "$writer" "$name=C"`,
	} {
		t.Run(line, func(t *testing.T) {
			if !backupNeedsEnvironmentReview(line) {
				t.Fatalf("unreviewed construct escaped inventory: %s", line)
			}
		})
	}
	for _, line := range []string{
		`local value="$1" other`, `read -r first second <<<"$input"`, `unset PGBACKREST_REPO1_CIPHER_PASS`,
		`set -euo pipefail`, `set -- "$@"`, `(( value >= minimum )) || value="$minimum"`,
	} {
		if backupNeedsEnvironmentReview(line) {
			t.Errorf("literal operation unexpectedly needs review: %s", line)
		}
	}
}
