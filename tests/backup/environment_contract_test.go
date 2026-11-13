package backup

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Inventory code loading and environment/name writes, including helper text.
// Fail closed on writer arguments whose destinations cannot be classified.
// This is not a Bash parser; unusual spellings remain the reviewer's job.
var backupEnvironmentConstructs = []*regexp.Regexp{
	regexp.MustCompile(`(?:^|[\s;&|()])["']?(?:source|eval|export)["']?(?:[\s;&|()]|$)`),
	regexp.MustCompile(`(?:^|[\s;&|()])["']?\.["']?(?:\s|$)`),
	regexp.MustCompile(`\$\{!`),
	// Startup-file selectors load code even without a source/eval command.
	regexp.MustCompile(`\b(?:BASH_ENV|ENV)\b|--(?:rcfile|init-file)\b`),
	// env accepts both literal assignments and computed assignment words.
	regexp.MustCompile(`(?:^|[\s;&|()])["']?(?:/[^\s;&|()]*/)?env["']?\s`),
	regexp.MustCompile(`(?:^|[\s;&|()])["']?let["']?(?:\s|$)`),
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

var backupLiteralName = regexp.MustCompile(`(?s)^[a-zA-Z_][a-zA-Z0-9_]*(\+?=.*)?$`)

type backupShellWord struct {
	raw, text string
	complete  bool
	separator bool
}

// Preserve raw quoting for destination checks; text is only for recognizing
// builtin/option words. No expansion, execution, or command-position inference.
// continuation: 0 = no join, 1 = outside quotes/comments, 2 = uncertain quoting.
func backupShellWords(line string) (words []backupShellWord, continuation int) {
	for i := 0; i < len(line); {
		if strings.ContainsRune(" \t\r\n", rune(line[i])) {
			i++
			continue
		}
		if line[i] == '#' {
			break // A comment ends at this physical newline, even with a backslash.
		}
		if strings.ContainsRune(";&|()<>{}", rune(line[i])) {
			words = append(words, backupShellWord{raw: line[i : i+1], text: line[i : i+1], complete: true, separator: true})
			i++
			continue
		}
		start, quote := i, byte(0)
		var text strings.Builder
		for i < len(line) {
			c := line[i]
			if quote == 0 && (strings.ContainsRune(" \t\r\n;&|()<>", rune(c))) {
				break
			}
			if c == '\\' && quote != '\'' {
				if i+1 == len(line) {
					continuation = 1
					if quote != 0 {
						continuation = 2
					}
					i++
					break
				}
				text.WriteByte(line[i+1])
				i += 2
				continue
			}
			if c == '\'' || c == '"' || c == '`' {
				if quote == 0 {
					quote = c
					i++
					continue
				}
				if quote == c {
					quote = 0
					i++
					continue
				}
			}
			text.WriteByte(c)
			i++
		}
		if i == len(line) && quote != 0 && strings.HasSuffix(line, "\\") {
			continuation = 2
		}
		words = append(words, backupShellWord{raw: line[start:i], text: text.String(), complete: quote == 0})
	}
	return words, continuation
}

func backupLiteralDestination(word backupShellWord) bool {
	return word.complete && backupLiteralName.MatchString(word.raw)
}

// Redirections can occur between a writer and its destinations. Skip their
// operands, not the rest of the command; an argument after <file still writes.
func backupCommandArgs(words []backupShellWord) []backupShellWord {
	var args []backupShellWord
	for i := 0; i < len(words); i++ {
		word := words[i]
		if word.separator && (word.text == "<" || word.text == ">") {
			for i < len(words) && words[i].separator && strings.Contains("<>&", words[i].text) {
				i++
			}
			if i == len(words) || words[i].separator {
				return append(args, backupShellWord{}) // Unclassifiable redirection.
			}
			continue
		}
		if word.separator {
			break
		}
		args = append(args, word)
	}
	return args
}

func backupWriterNeedsReview(writer string, args []backupShellWord) bool {
	if writer == "printf" {
		for _, arg := range args {
			if strings.HasPrefix(arg.text, "-v") || strings.Contains(arg.text, "%n") {
				return true
			}
		}
		// A computed format could introduce %n even if its source word does
		// not contain it. Ordinary printf data arguments are not formats.
		if len(args) > 0 && args[0].text == "--" {
			args = args[1:]
		}
		if len(args) > 0 && (!args[0].complete || strings.ContainsAny(args[0].raw, "$`")) {
			return true
		}
		return false
	}
	if writer == "getopts" {
		// The option specification and trailing input arguments are data;
		// only the second argument is the destination.
		return len(args) < 2 || !args[0].complete || !backupLiteralDestination(args[1])
	}
	options := true
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !arg.complete {
			return true
		}
		if options && arg.text == "--" {
			options = false
			continue
		}
		if options && strings.HasPrefix(arg.text, "-") {
			flags := strings.TrimPrefix(arg.text, "-")
			if flags == "" {
				return true
			}
			for j := 0; j < len(flags); j++ {
				flag := flags[j]
				var simple, withValue string
				switch writer {
				case "read":
					simple, withValue = "ers", "adinNptu"
				case "mapfile", "readarray":
					simple, withValue = "t", "dnOsuc"
				case "declare", "typeset", "local":
					// -x and -n always need review, including combined flags.
					simple = "aAfFgilprtuI"
				case "readonly":
					simple = "aAfp"
				case "unset":
					simple = "fv"
				}
				if strings.ContainsRune(simple, rune(flag)) {
					continue
				}
				if !strings.ContainsRune(withValue, rune(flag)) {
					return true // Unknown option/attribute or executable callback.
				}
				// Option values are not destinations, except read -a's array.
				var value backupShellWord
				if j+1 < len(flags) {
					value = backupShellWord{raw: flags[j+1:], text: flags[j+1:], complete: arg.raw == arg.text}
				} else {
					i++
					if i >= len(args) {
						return true
					}
					value = args[i]
				}
				if !value.complete || (writer == "read" && flag == 'a' && !backupLiteralDestination(value)) {
					return true
				}
				break
			}
			continue
		}
		options = false
		if !backupLiteralDestination(arg) {
			return true
		}
	}
	return false
}

func backupSetNeedsReview(args []backupShellWord) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg.text == "--" {
			return false
		}
		if !arg.complete || strings.ContainsAny(arg.text, "$`\\") {
			return true // Computed options cannot be shown safe.
		}
		if strings.HasPrefix(arg.text, "-") && strings.Contains(arg.text, "a") {
			return true
		}
		if arg.text == "o" || (strings.HasPrefix(arg.text, "-") || strings.HasPrefix(arg.text, "+")) && strings.Contains(arg.text, "o") {
			i++
			if i >= len(args) || !args[i].complete || strings.ContainsAny(args[i].text, "$`\\") || args[i].text == "allexport" {
				return true
			}
		}
	}
	return false
}

func backupNeedsEnvironmentReview(line string) bool {
	return backupReviewShellText(line, 0)
}

func backupReviewShellText(line string, depth int) bool {
	if depth > 8 {
		return true
	}
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		return false
	}
	for _, pattern := range backupEnvironmentConstructs {
		if pattern.MatchString(line) {
			return true
		}
	}
	words, _ := backupShellWords(line)
	for i, word := range words {
		args := backupCommandArgs(words[i+1:])
		switch word.text {
		case "declare", "typeset", "local", "readonly", "read", "unset", "mapfile", "readarray", "printf", "getopts":
			if backupWriterNeedsReview(word.text, args) {
				return true
			}
		case "set":
			if backupSetNeedsReview(args) {
				return true
			}
		}
		// Quoted shell programs are code, unlike printf's format-string data.
		switch filepath.Base(word.text) {
		case "bash", "sh", "dash", "ksh", "zsh":
			for j, arg := range args {
				if strings.HasPrefix(arg.text, "-") && strings.Contains(arg.text, "c") && j+1 < len(args) {
					if backupReviewShellText(args[j+1].text, depth+1) {
						return true
					}
					break
				}
			}
		}
	}
	return false
}

type backupEnvironmentUse struct {
	line int
	text string
}

func scanBackupEnvironment(content string) []backupEnvironmentUse {
	lines := strings.Split(content, "\n")
	var uses []backupEnvironmentUse
	for i := 0; i < len(lines); i++ {
		start, joined := i, lines[i]
		_, continuation := backupShellWords(joined)
		for continuation != 0 && i+1 < len(lines) {
			i++
			joined = strings.TrimSuffix(joined, "\\") + lines[i]
			_, continuation = backupShellWords(joined)
		}
		// Always inspect physical lines too: uncertain quoted/helper contexts
		// must never let joining hide a later executable line (especially p21).
		found := false
		for n := start; n <= i; n++ {
			physical := lines[n]
			if _, kind := backupShellWords(physical); kind == 1 {
				physical = strings.TrimSuffix(physical, "\\")
			}
			if backupNeedsEnvironmentReview(physical) {
				uses = append(uses, backupEnvironmentUse{n + 1, lines[n]})
				found = true
			}
		}
		if !found && backupNeedsEnvironmentReview(joined) {
			uses = append(uses, backupEnvironmentUse{start + 1, strings.Join(lines[start:i+1], "\n")})
		}
	}
	return uses
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
		for _, use := range scanBackupEnvironment(string(content)) {
			original := use.text
			count++
			if _, ok := entries[original]; !ok {
				t.Errorf("%s:%d: unapproved code loading or environment/name write: %s", path, use.line, original)
			} else {
				seen[original]++
				if seen[original] > 1 {
					t.Errorf("%s:%d: duplicated environment allowlist entry: %s", path, use.line, original)
				}
				t.Logf("%s:%d: reviewed environment use: %s", path, use.line, original)
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
		`part=LC; IFS= read -r "${part}_ALL" <<<C`,
		`part=LC; { read -r "${part}_ALL" <<<C; }`,
		`part=R5_VALUE; printf '%n' "$part"`,
		`BASH_ENV="${FVPN_BACKUP_SETTINGS:-/dev/null}" bash -c true`,
		`ENV="${FVPN_BACKUP_SETTINGS:-/dev/null}" sh -c true`,
		`bash --rcfile "$settings"`, `bash --init-file "$settings"`,
		`set -aeu`, `set -o nounset -o allexport`, `set '-o' 'allexport'`,
		`bash -c 'IFS= read -r "${part}_ALL" <<<C'`,
		`( read -r "$name" )`, `! read -r "$name"`, `if read -r "$name"; then :; fi`,
		`true && read -r "$name"`, `false || read -r "$name"`,
		`while true; do read -r "$name"; done`, `if true; then read -r "$name"; fi`,
		`read -d '' -a "$name"`, `read -Z destination`, `local "destination"=value`,
		`read -r <<<C "$name"`, `IFS= read <"$input" "$name"`,
		`printf "$format" "$name"`,
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
		`local x="two words"`, `local x=1`, `read -r line <<<C`,
		`IFS= read -r first second <<<"$input"`,
		`read -r -d '' -a parts`, `read -p "$prompt" -u 3 line`,
		`declare -g value=1`, `typeset value+=more`, `readonly value="two words"`,
		`mapfile -t -d '' lines`, `readarray -n 3 lines`, `getopts ':ab' option "$@"`,
		`read -r <"$input" line`,
	} {
		if backupNeedsEnvironmentReview(line) {
			t.Errorf("literal operation unexpectedly needs review: %s", line)
		}
	}
}

func TestAT30_BackupEnvironmentPhysicalLines(t *testing.T) {
	for _, comment := range []string{"# Optional settings \\", "true # Optional settings \\"} {
		source := `source "${FVPN_BACKUP_SETTINGS:-/dev/null}"`
		uses := scanBackupEnvironment(comment + "\n" + source + "\n")
		if len(uses) != 1 || uses[0].line != 2 || uses[0].text != source {
			t.Fatalf("comment continuation hid executable line 2: %#v", uses)
		}
	}
	for _, program := range []string{
		"read -r \\\n\"$name\"",
		"bash <<'PROGRAM'\nIFS= read -r \"$name\"\nPROGRAM",
		"'uncertain quoted text \\\nsource \"$settings\"",
	} {
		if uses := scanBackupEnvironment(program); len(uses) == 0 {
			t.Fatalf("continued/heredoc/uncertain program escaped review: %q", program)
		}
	}
}
