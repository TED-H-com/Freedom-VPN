// Package handover_test verifies that the owned handover and operations
// Markdown documents stay synchronized with the repository layout, Make
// targets, built fvpn-* binaries, migration inventory, and PostgreSQL schema.
package handover_test

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// ownedDocs lists the handover, ops, and root Markdown files owned by
// w4-handover. The follow-up 13 brief includes the restore drill and indexes.
var ownedDocs = []string{
	"README.md",
	"docs/handover/README.md",
	"docs/handover/00-index.md",
	"docs/handover/11-database-and-retention.md",
	"docs/handover/restore-drill.md",
	"docs/handover/deploy.md",
	"docs/handover/calibration.md",
	"docs/handover/operator-guide.md",
	"docs/handover/member-guide.md",
	"docs/handover/follow-up-17.md",
	"docs/handover/follow-up-18.md",
	"docs/handover/follow-up-19.md",
	"docs/handover/follow-up-20.md",
	"docs/handover/follow-up-21.md",
	"docs/handover/follow-up-23.md",
	"docs/handover/follow-up-24.md",
	"docs/handover/follow-up-25.md",
	"docs/handover/follow-up-26.md",
	"docs/handover/follow-up-27.md",
	"docs/handover/follow-up-28.md",
	"docs/handover/follow-up-29.md",
	"docs/ops/README.md",
	"docs/reviews/README.md",
	"docs/reviews/r3-d055-review.md",
	"docs/reviews/r4-dnsfeeds-review.md",
	"docs/reviews/r5-backups-review.md",
	"docs/reviews/r6-d053-review.md",
	"docs/reviews/r7-d052-review.md",
	"docs/reviews/r8-d057-review.md",
	"docs/reviews/r9-d062-review.md",
	"docs/reviews/r10-d063-review.md",
	"docs/reviews/r11-d064-review.md",
	"docs/reviews/r12-enforce-limits-review.md",
	"docs/reviews/r13-r12-fixes-review.md",
	"docs/reviews/r14-d067-d070-review.md",
	"docs/reviews/r15-d072-d075-review.md",
}

var repoPathPrefixes = []string{
	"deploy/",
	"mk/",
	"docs/",
	"contracts/",
	"db/migrations/",
	"apps/",
	"node/",
	"internal/",
	"pkg/",
	"packaging/",
	"tests/",
}

func repoRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve caller path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("cannot locate repo root from %s: %v", root, err)
	}
	return root
}

func readOwnedDoc(t testing.TB, root, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

func stripFencedCodeBlocks(md string) string {
	re := regexp.MustCompile("(?s)```.*?```")
	return re.ReplaceAllString(md, "")
}

func TestHandover_OpenAPIClaims(t *testing.T) {
	root := repoRoot(t)
	var contract struct {
		Info  struct{ Version string }        `yaml:"info"`
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal([]byte(readOwnedDoc(t, root, "contracts/openapi.yaml")), &contract); err != nil {
		t.Fatal(err)
	}
	if contract.Info.Version == "" {
		t.Fatal("OpenAPI info.version is empty")
	}
	operations := 0
	for _, path := range contract.Paths {
		for method := range path {
			if slices.Contains([]string{"get", "put", "post", "delete", "patch", "options", "head", "trace"}, method) {
				operations++
			}
		}
	}
	if operations == 0 {
		t.Fatal("OpenAPI contains no operations")
	}
	member := readOwnedDoc(t, root, "docs/handover/member-guide.md")
	if !strings.Contains(member, "OpenAPI `"+contract.Info.Version+"`") {
		t.Errorf("member guide must state OpenAPI info.version %q", contract.Info.Version)
	}
	versionRE := regexp.MustCompile("OpenAPI `([0-9]+\\.[0-9]+\\.[0-9]+)`")
	countRE := regexp.MustCompile("([0-9]+)(?:-operation OpenAPI| HTTP operations|` Operations|` operations)")
	for _, doc := range ownedDocs {
		content := readOwnedDoc(t, root, doc)
		for _, match := range versionRE.FindAllStringSubmatch(content, -1) {
			if match[1] != contract.Info.Version {
				t.Errorf("%s: OpenAPI version %s, want %s", doc, match[1], contract.Info.Version)
			}
		}
		for _, match := range countRE.FindAllStringSubmatch(content, -1) {
			if match[1] != strconv.Itoa(operations) {
				t.Errorf("%s: operation count %s, want %d", doc, match[1], operations)
			}
		}
	}
}

func TestHandover_MarkdownLinks(t *testing.T) {
	root := repoRoot(t)
	linkRE := regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)

	checked := 0
	for _, doc := range ownedDocs {
		content := stripFencedCodeBlocks(readOwnedDoc(t, root, doc))
		docDir := filepath.Dir(filepath.Join(root, doc))
		for _, m := range linkRE.FindAllStringSubmatch(content, -1) {
			target := strings.TrimSpace(m[2])
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
				t.Logf("[non-mechanical: external-url] %s -> %s", doc, target)
				continue
			}
			pathPart, _, _ := strings.Cut(target, "#")
			if pathPart == "" {
				continue
			}
			resolved := filepath.Clean(filepath.Join(docDir, pathPart))
			if _, err := os.Stat(resolved); err != nil {
				t.Errorf("%s: broken Markdown link %q (resolved to %s): %v", doc, target, resolved, err)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("expected to check at least one Markdown link")
	}
}

func TestHandover_RepoPathsInBackticks(t *testing.T) {
	root := repoRoot(t)
	backtickRE := regexp.MustCompile("`([^`\\n]+)`")
	cmdBuildPathRE := regexp.MustCompile("(?:^|[[:space:]`])\\./((?:apps|node|internal|packaging|tests)/[A-Za-z0-9_./-]+)")

	checked := 0
	for _, doc := range ownedDocs {
		full := readOwnedDoc(t, root, doc)
		inline := stripFencedCodeBlocks(full)

		for _, m := range backtickRE.FindAllStringSubmatch(inline, -1) {
			span := strings.TrimSpace(m[1])
			// Also inspect whitespace-separated tokens inside a command span
			// (e.g. `go test -v ./tests/handover/...`).
			candidates := []string{span}
			if strings.ContainsAny(span, " \t") {
				for _, f := range strings.Fields(span) {
					f = strings.TrimPrefix(f, "./")
					f = strings.TrimSuffix(f, "/...")
					candidates = append(candidates, f)
				}
			}
			for _, cand := range candidates {
				if !hasRepoPrefix(cand) {
					continue
				}
				verifyRepoPathClaim(t, root, doc, cand)
				checked++
			}
		}

		for _, m := range cmdBuildPathRE.FindAllStringSubmatch(full, -1) {
			cand := strings.TrimSuffix(m[1], "/...")
			verifyRepoPathClaim(t, root, doc, cand)
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("expected to check at least one repo path in backticks")
	}
}

func hasRepoPrefix(s string) bool {
	for _, p := range repoPathPrefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func verifyRepoPathClaim(t *testing.T, root, doc, claim string) {
	t.Helper()
	// Strip section suffix like "docs/handover/deploy.md §8" if present.
	if idx := strings.Index(claim, " "); idx > 0 {
		claim = claim[:idx]
	}
	// Strip ":field" config/schema suffix if present.
	if idx := strings.Index(claim, ":"); idx > 0 {
		claim = claim[:idx]
	}

	// Template patterns using ellipsis or <placeholder> (e.g. `deploy/…`, `mk/<file>.mk`).
	if strings.Contains(claim, "…") || (strings.Contains(claim, "<") && strings.Contains(claim, ">")) {
		prefix := claim
		if idx := strings.Index(prefix, "…"); idx >= 0 {
			prefix = prefix[:idx]
		}
		if idx := strings.Index(prefix, "<"); idx >= 0 {
			prefix = prefix[:idx]
		}
		if !strings.HasSuffix(prefix, "/") {
			prefix = filepath.Dir(prefix)
		}
		prefix = filepath.Clean(filepath.Join(root, prefix))
		if _, err := os.Stat(prefix); err != nil {
			t.Errorf("%s: template path %q base directory does not exist: %v", doc, claim, err)
		}
		t.Logf("[non-mechanical: template-path] %s -> %s (base verified)", doc, claim)
		return
	}

	// Glob patterns (e.g. `db/migrations/*.sql`, `apps/*/cmd/*`).
	if strings.Contains(claim, "*") {
		matches, err := filepath.Glob(filepath.Join(root, claim))
		if err != nil || len(matches) == 0 {
			t.Errorf("%s: glob path %q matched 0 files in repo", doc, claim)
		}
		return
	}

	fullPath := filepath.Join(root, claim)
	if _, err := os.Stat(fullPath); err == nil {
		return
	}

	// Check if claim is a Go package + exported symbol reference (e.g. `node/dns.Applier`
	// or `internal/netbird.Client.RemoveDefaultPolicy`).
	if dotIdx := strings.Index(claim, "."); dotIdx > 0 {
		pkgRel := claim[:dotIdx]
		sym := claim[dotIdx+1:]
		if idx := strings.Index(sym, "."); idx > 0 {
			sym = sym[:idx]
		}
		pkgDir := filepath.Join(root, pkgRel)
		if info, err := os.Stat(pkgDir); err == nil && info.IsDir() && goSymbolExists(pkgDir, sym) {
			return
		}
	}

	t.Errorf("%s: repo path in backticks does not exist: %q", doc, claim)
}

func goSymbolExists(dir, sym string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		if strings.Contains(string(raw), sym) {
			return true
		}
	}
	return false
}

func TestHandover_MakeTargets(t *testing.T) {
	root := repoRoot(t)
	makeRE := regexp.MustCompile(`make\s+(?:--no-print-directory\s+)?-f\s+(mk/[^\s` + "`" + `]+)\s+([A-Za-z0-9_<>/.-]+)`)

	checked := 0
	for _, doc := range ownedDocs {
		content := readOwnedDoc(t, root, doc)
		for _, m := range makeRE.FindAllStringSubmatch(content, -1) {
			mkFile := m[1]
			target := m[2]
			if strings.Contains(mkFile, "<") || strings.Contains(target, "<") {
				t.Logf("[non-mechanical: template-make-target] %s -> make -f %s %s", doc, mkFile, target)
				continue
			}
			mkPath := filepath.Join(root, mkFile)
			raw, err := os.ReadFile(mkPath)
			if err != nil {
				t.Errorf("%s: referenced Makefile %q does not exist: %v", doc, mkFile, err)
				continue
			}
			targetRE := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(target) + `\s*:`)
			if !targetRE.Match(raw) {
				t.Errorf("%s: target %q not found in %s", doc, target, mkFile)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("expected to check at least one make -f mk/<file>.mk <target> invocation")
	}
}

func TestHandover_FVPnBinaries(t *testing.T) {
	root := repoRoot(t)
	cmdGlobs := []string{
		"apps/*/cmd/fvpn-*",
		"node/*/cmd/fvpn-*",
		"node/cmd/fvpn-*",
		"internal/*/cmd/fvpn-*",
		"packaging/*/cmd/fvpn-*",
		"tests/*/cmd/fvpn-*",
	}
	binaries := map[string]string{}
	for _, g := range cmdGlobs {
		matches, err := filepath.Glob(filepath.Join(root, g))
		if err != nil {
			t.Fatalf("glob %s: %v", g, err)
		}
		for _, m := range matches {
			if _, err := os.Stat(filepath.Join(m, "main.go")); err == nil {
				binaries[filepath.Base(m)] = m
			}
		}
	}
	if len(binaries) == 0 {
		t.Fatal("discovered 0 fvpn-* binaries in repo")
	}

	// Non-binary fvpn-* identifiers used in the specification and configuration,
	// classified explicitly so nothing is skipped silently.
	nonBinaryClassification := map[string]string{
		"fvpn-netbird":                "packaged-NetBird-runtime-directory",
		"fvpn-w4-secops":              "security-ops-harness-Docker-prefix-D103",
		"fvpn-paid":                   "netbird-paid-group-prefix",
		"fvpn-logical":                "logical-backup-file-prefix",
		"fvpn-w4-ops-backups-gha":     "historical-backup-CI-container-prefix",
		"fvpn-pg":                     "historical-shared-test-postgres-name",
		"fvpn-admin":                  "admin-session-cookie (__Host-fvpn-admin)",
		"fvpn-member":                 "member-session-cookie (__Host-fvpn-member)",
		"fvpn-enrollment-v1":          "protocol-domain-separation-label",
		"fvpn-release-trust-v1":       "protocol-domain-separation-label",
		"fvpn-portal-prod":            "example-oidc-client-id",
		"fvpn-portal-test":            "example-oidc-client-id",
		"fvpn-enroll-peer":            "netbird-correlation-group-prefix (fvpn-enroll-peer-<hash>)",
		"fvpn-exit":                   "netbird-exit-network-prefix (fvpn-exit-<node>)",
		"fvpn-unpublished-nodes":      "netbird-shared-staging-group-name",
		"fvpn-node":                   "netbird-setup-key-prefix (fvpn-node-<node_id>-<incarnation_id>)",
		"fvpn-updater-recover":        "systemd-oneshot-unit (fvpn-updater-recover.service)",
		"fvpn-w4-ops-backups":         "backup-drill-docker-prefix",
		"fvpn-w4-ops-backups-primary": "backup-drill-container-name",
		"fvpn-backup":                 "backup-systemd-unit-prefix",
		"fvpn-postgres-entrypoint":    "backup-postgres-entrypoint-script",
		"fvpn-oidc":                   "devstack-oidc-container-name",
		"fvpn-nb-login":               "devstack-netbird-login-temp-log-prefix (/var/run/fvpn-nb-login-*.log)",
	}

	tokenRE := regexp.MustCompile(`\b(fvpn-[a-z0-9-]+)(\.service)?\b`)
	checked := 0
	for _, doc := range ownedDocs {
		content := readOwnedDoc(t, root, doc)
		for _, m := range tokenRE.FindAllStringSubmatch(content, -1) {
			name := m[1]
			suffix := m[2]
			if suffix == ".service" {
				// Systemd service units live in node/agent/systemd/ or deploy/backup/systemd/
				// (central units fvpn-control.service and fvpn-worker.service are
				// inline unit files in deploy.md corresponding to built binaries).
				if _, ok := binaries[name]; ok {
					checked++
					continue
				}
				nodeUnitPath := filepath.Join(root, "node/agent/systemd", name+".service")
				backupUnitPath := filepath.Join(root, "deploy/backup/systemd", name+".service")
				if _, err := os.Stat(nodeUnitPath); err != nil {
					if _, err2 := os.Stat(backupUnitPath); err2 != nil {
						t.Errorf("%s: systemd unit %s.service neither matches a binary nor exists under node/agent/systemd or deploy/backup/systemd", doc, name)
					}
				}
				checked++
				continue
			}
			if kind, ok := nonBinaryClassification[name]; ok {
				t.Logf("[non-mechanical: %s] %s -> %s", kind, doc, name)
				continue
			}
			if _, ok := binaries[name]; !ok {
				t.Errorf("%s: %q does not match any built binary under apps/*/cmd/*, node/*/cmd/*, internal/*/cmd/*, packaging/*/cmd/*, or tests/e2e/cmd/*", doc, name)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("expected to verify at least one fvpn-* binary reference")
	}
}

func repoMigrationIDs(t testing.TB, root string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(root, "db/migrations/*.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	slices.Sort(files)
	if len(files) == 0 {
		t.Fatal("no SQL files found in db/migrations")
	}

	var wantIDs []string
	for _, f := range files {
		base := filepath.Base(f)
		id, _, ok := strings.Cut(base, "_")
		if !ok {
			t.Fatalf("unexpected migration filename %q", base)
		}
		wantIDs = append(wantIDs, id)
	}
	return wantIDs
}

func auditMigrationInventoryClaims(doc, content string, wantIDs []string) (literalClaims []string, mismatchErrors []string) {
	wantCount := len(wantIDs)
	wantFirst := wantIDs[0]
	wantLast := wantIDs[len(wantIDs)-1]

	countRE := regexp.MustCompile("(?i)(?:`?([0-9]+)`?\\s+(?:forward\\s+)?migrations(?:\\s+applied)?|migration\\s+count\\s+`?([0-9]+)`?)")
	rangeRE := regexp.MustCompile("`(00[0-9]{3})`\\s*[–-]\\s*`(00[0-9]{3})`")
	lastIDRE := regexp.MustCompile("(?i)(?:last(?:-migration)?\\s+ID\\s+`(00[0-9]{3})`|through\\s+`(00[0-9]{3})(?:_[a-z0-9_]+\\.sql)?`)")

	for _, m := range countRE.FindAllStringSubmatch(content, -1) {
		numStr := m[1]
		if numStr == "" {
			numStr = m[2]
		}
		literalClaims = append(literalClaims, fmt.Sprintf("%s: literal migration count %q", doc, m[0]))
		gotCount, err := strconv.Atoi(numStr)
		if err != nil {
			mismatchErrors = append(mismatchErrors, fmt.Sprintf("%s: parse migration count %q: %v", doc, numStr, err))
			continue
		}
		if gotCount != wantCount {
			mismatchErrors = append(mismatchErrors, fmt.Sprintf("%s: migration count claim is %d, want %d (from db/migrations/)", doc, gotCount, wantCount))
		}
	}
	for _, m := range rangeRE.FindAllStringSubmatch(content, -1) {
		literalClaims = append(literalClaims, fmt.Sprintf("%s: literal migration range %q", doc, m[0]))
		if m[1] != wantFirst || m[2] != wantLast {
			mismatchErrors = append(mismatchErrors, fmt.Sprintf("%s: migration range claim is %s..%s, want %s..%s", doc, m[1], m[2], wantFirst, wantLast))
		}
	}
	for _, m := range lastIDRE.FindAllStringSubmatch(content, -1) {
		gotLast := m[1]
		if gotLast == "" {
			gotLast = m[2]
		}
		literalClaims = append(literalClaims, fmt.Sprintf("%s: literal migration last-ID %q", doc, m[0]))
		if gotLast != wantLast {
			mismatchErrors = append(mismatchErrors, fmt.Sprintf("%s: migration last-ID claim is %s, want %s (from db/migrations/)", doc, gotLast, wantLast))
		}
	}

	statusBlockRE := regexp.MustCompile("(?s)```text\\ncurrent_database=\"fvpn\"\\n(.*?)```")
	if m := statusBlockRE.FindStringSubmatch(content); m != nil {
		lines := strings.Split(strings.TrimSpace(m[1]), "\n")
		abbreviated := []string{wantFirst + " applied", "...", "<last file> applied"}
		if !slices.Equal(lines, abbreviated) {
			literalClaims = append(literalClaims, fmt.Sprintf("%s: non-abbreviated fvpn-migrate status block (%d lines)", doc, len(lines)))
			var gotIDs []string
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) != 2 || fields[1] != "applied" {
					mismatchErrors = append(mismatchErrors, fmt.Sprintf("%s: unexpected status line %q", doc, line))
					continue
				}
				gotIDs = append(gotIDs, fields[0])
			}
			if !slices.Equal(gotIDs, wantIDs) {
				mismatchErrors = append(mismatchErrors, fmt.Sprintf("%s: fvpn-migrate status IDs = %v, want %v", doc, gotIDs, wantIDs))
			}
		}
	}
	return literalClaims, mismatchErrors
}

func TestHandover_MigrationInventory(t *testing.T) {
	root := repoRoot(t)
	wantIDs := repoMigrationIDs(t, root)
	wantFirst := wantIDs[0]

	var allLiteralClaims []string
	for _, doc := range ownedDocs {
		content := readOwnedDoc(t, root, doc)
		claims, errs := auditMigrationInventoryClaims(doc, content, wantIDs)
		allLiteralClaims = append(allLiteralClaims, claims...)
		for _, e := range errs {
			t.Error(e)
		}
	}

	// Verify docs/handover/deploy.md states db/migrations/*.sql and the
	// abbreviated fvpn-migrate status output block.
	deployMD := readOwnedDoc(t, root, "docs/handover/deploy.md")
	if !strings.Contains(deployMD, "`db/migrations/*.sql`") {
		t.Error("docs/handover/deploy.md: expected reference to `db/migrations/*.sql` as the migration inventory")
	}
	statusBlockRE := regexp.MustCompile("(?s)```text\\ncurrent_database=\"fvpn\"\\n(.*?)```")
	m := statusBlockRE.FindStringSubmatch(deployMD)
	if m == nil {
		t.Fatal("docs/handover/deploy.md: missing fvpn-migrate status output block")
	}
	gotLines := strings.Split(strings.TrimSpace(m[1]), "\n")
	wantLines := []string{wantFirst + " applied", "...", "<last file> applied"}
	if !slices.Equal(gotLines, wantLines) {
		t.Errorf("docs/handover/deploy.md: fvpn-migrate status lines = %q, want abbreviated %q", gotLines, wantLines)
	}

	t.Run("OwnedDocsMakeNoLiteralCountOrLastIDClaims", func(t *testing.T) {
		if len(allLiteralClaims) > 0 {
			t.Errorf("owned docs must not hardcode literal migration counts, ranges, or last IDs; found:\n%s", strings.Join(allLiteralClaims, "\n"))
		}
		syntheticClaims, syntheticErrs := auditMigrationInventoryClaims(
			"synthetic.md",
			"(`00001`–`00999`, 99 migrations applied through `00999_future.sql`, last ID `00999`)\n```text\ncurrent_database=\"fvpn\"\n00001 applied\n00999 applied\n```",
			wantIDs,
		)
		if len(syntheticClaims) < 4 || len(syntheticErrs) < 4 {
			t.Fatalf("expected strict detector to catch synthetic literal claims and mismatches; got claims=%v errs=%v", syntheticClaims, syntheticErrs)
		}
	})
}

type sqlBlock struct {
	Doc                    string
	Index                  int
	ClusterRoleDDL         bool
	MutatingMaintenanceSQL bool
	Raw                    string
}

func sqlCheckedDocs(t testing.TB, root string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, "docs", "handover", "*.md"))
	if err != nil {
		t.Fatalf("glob docs/handover/*.md: %v", err)
	}
	var docs []string
	for _, m := range matches {
		rel, err := filepath.Rel(root, m)
		if err != nil {
			t.Fatalf("rel path for %s: %v", m, err)
		}
		docs = append(docs, filepath.ToSlash(rel))
	}
	docs = append(docs, "docs/ops/README.md")
	slices.Sort(docs)
	return docs
}

// extractSQLBlocks extracts every ```sql fence from docs/handover/*.md (including
// restore-drill.md) and docs/ops/README.md using the exact line-based handoverSQL
// rule from tests/backup/drill_integration_test.go.
func extractSQLBlocks(t testing.TB, root string) []sqlBlock {
	t.Helper()
	var out []sqlBlock
	for _, doc := range sqlCheckedDocs(t, root) {
		content := readOwnedDoc(t, root, doc)
		var inFence bool
		var cur []string
		idx := 0
		for _, line := range strings.Split(content, "\n") {
			trimmed := strings.TrimSpace(line)
			if !inFence {
				if trimmed == "```sql" {
					inFence = true
					cur = nil
				}
				continue
			}
			if trimmed == "```" {
				inFence = false
				body := strings.TrimSpace(strings.Join(cur, "\n"))
				if body != "" {
					idx++
					clusterDDL := strings.HasPrefix(body, "-- [mechanical-check: cluster-role-ddl]")
					mutatingMaint := strings.HasPrefix(body, "-- [mechanical-check: mutating-maintenance-sql]")
					out = append(out, sqlBlock{
						Doc:                    doc,
						Index:                  idx,
						ClusterRoleDDL:         clusterDDL,
						MutatingMaintenanceSQL: mutatingMaint,
						Raw:                    body,
					})
				}
				continue
			}
			cur = append(cur, line)
		}
	}
	return out
}

func TestHandover_SQLBlocksClassified(t *testing.T) {
	root := repoRoot(t)
	blocks := extractSQLBlocks(t, root)
	if len(blocks) == 0 {
		t.Fatal("expected to find SQL blocks in owned handover docs")
	}
	restoreDrillBlocks := 0
	for _, b := range blocks {
		if b.Doc == "docs/handover/restore-drill.md" {
			restoreDrillBlocks++
			if b.ClusterRoleDDL || b.MutatingMaintenanceSQL || strings.Contains(b.Raw, "[mechanical-check:") {
				t.Errorf("%s SQL block #%d must not carry any [mechanical-check: ...] exemption because TestAT30_BackupPITRDrill executes every fence read-only", b.Doc, b.Index)
			}
		}
		if b.ClusterRoleDDL {
			t.Logf("[non-mechanical: cluster-role-ddl] %s SQL block #%d explicitly marked as superuser/cluster role DDL", b.Doc, b.Index)
			continue
		}
		if b.MutatingMaintenanceSQL {
			t.Logf("[mechanical-check: mutating-maintenance-sql] %s SQL block #%d explicitly marked as mutating maintenance SQL (executed in rolled-back read-write tx)", b.Doc, b.Index)
			continue
		}
		if strings.Contains(b.Raw, "CREATE DATABASE") || strings.Contains(b.Raw, "CREATE ROLE") {
			t.Errorf("%s SQL block #%d contains cluster DDL without -- [mechanical-check: cluster-role-ddl] marker", b.Doc, b.Index)
		}
	}
	if restoreDrillBlocks < 5 {
		t.Errorf("expected at least 5 SQL blocks in docs/handover/restore-drill.md, got %d", restoreDrillBlocks)
	}
}

func TestHandover_ReviewsArchive(t *testing.T) {
	root := repoRoot(t)
	reviewsDir := filepath.Join(root, "docs", "reviews")
	entries, err := os.ReadDir(reviewsDir)
	if err != nil {
		t.Fatalf("read docs/reviews: %v", err)
	}

	wantFiles := []string{
		"README.md",
		"r3-d055-review.md",
		"r4-dnsfeeds-review.md",
		"r5-backups-review.md",
		"r6-d053-review.md",
		"r7-d052-review.md",
		"r8-d057-review.md",
		"r9-d062-review.md",
		"r10-d063-review.md",
		"r11-d064-review.md",
		"r12-enforce-limits-review.md",
		"r13-r12-fixes-review.md",
		"r14-d067-d070-review.md",
		"r15-d072-d075-review.md",
		"r16-rev13-d089-d099-review.md",
		"r17-rev14-d104-d117-review.md",
		"r18-generation-change-review.md",
		"r19-intersection-design-review.md",
	}

	var gotFiles []string
	for _, e := range entries {
		if e.IsDir() {
			t.Errorf("unexpected subdirectory in docs/reviews: %s", e.Name())
			continue
		}
		gotFiles = append(gotFiles, e.Name())
	}
	for _, want := range wantFiles {
		if !slices.Contains(gotFiles, want) {
			t.Errorf("docs/reviews missing expected file %q", want)
		}
	}

	dateRE := regexp.MustCompile(`(?m)^-\s+\*\*Date:\*\*\s+(.+)$`)
	modelRE := regexp.MustCompile(`(?m)^-\s+\*\*Reviewing model:\*\*\s+(.+)$`)
	headerLine := "| Severity | Finding | Status | Closing PR |"

	for _, name := range gotFiles {
		rel := filepath.Join("docs", "reviews", name)
		content := readOwnedDoc(t, root, rel)
		if name == "r16-rev13-d089-d099-review.md" || name == "r17-rev14-d104-d117-review.md" || name == "r18-generation-change-review.md" || name == "r19-intersection-design-review.md" {
			// D-098 forbids reformatting the raw report to our older table schema.
			// Followup15Claims / Followup18Claims / Followup25Claims / Followup27Claims verify SHA-256 and review metadata.
			continue
		}

		if m := dateRE.FindStringSubmatch(content); m == nil || strings.TrimSpace(m[1]) == "" {
			t.Errorf("%s: missing '- **Date:** ...' line", rel)
		}
		if m := modelRE.FindStringSubmatch(content); m == nil || strings.TrimSpace(m[1]) == "" {
			t.Errorf("%s: missing '- **Reviewing model:** ...' line", rel)
		}

		idx := strings.Index(content, headerLine)
		if idx < 0 {
			t.Errorf("%s: missing findings table header %q", rel, headerLine)
			continue
		}

		lines := strings.Split(content[idx:], "\n")
		if len(lines) < 3 {
			t.Errorf("%s: findings table after header has fewer than 3 lines", rel)
			continue
		}
		sepCells := splitMarkdownRow(lines[1])
		if len(sepCells) != 4 {
			t.Errorf("%s: findings table separator row has %d columns, want 4: %q", rel, len(sepCells), lines[1])
			continue
		}

		dataRows := 0
		for _, line := range lines[2:] {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "|") {
				break
			}
			cells := splitMarkdownRow(line)
			if len(cells) != 4 {
				t.Errorf("%s: findings table row has %d columns, want 4: %q", rel, len(cells), line)
				continue
			}
			sev, finding, status, closingPR := cells[0], cells[1], cells[2], cells[3]
			if !slices.Contains([]string{"High", "Medium", "Low"}, sev) {
				t.Errorf("%s: unexpected Severity %q in row %q", rel, sev, line)
			}
			if finding == "" {
				t.Errorf("%s: empty Finding cell in row %q", rel, line)
			}
			if !slices.Contains([]string{"closed", "open"}, status) {
				t.Errorf("%s: unexpected Status %q in row %q (want closed or open)", rel, status, line)
			}
			if closingPR == "" {
				t.Errorf("%s: empty Closing PR cell in row %q", rel, line)
			}
			dataRows++
		}
		if dataRows == 0 {
			t.Errorf("%s: findings table contains 0 data rows", rel)
		}
	}
}

func splitMarkdownRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

func TestHandover_DOC15AndFollowup6Claims(t *testing.T) {
	root := repoRoot(t)

	memberGuide := readOwnedDoc(t, root, "docs/handover/member-guide.md")
	var zhStatusLine, enStatusLine string
	for _, line := range strings.Split(memberGuide, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- **實測狀態（`DOC15`") {
			zhStatusLine = trimmed
		}
		if strings.HasPrefix(trimmed, "- **Measurement status (`DOC15`") {
			enStatusLine = trimmed
		}
	}
	if zhStatusLine == "" {
		t.Fatal("docs/handover/member-guide.md: missing zh-TW DOC15 measurement status paragraph")
	}
	if enStatusLine == "" {
		t.Fatal("docs/handover/member-guide.md: missing en DOC15 measurement status paragraph")
	}

	requiredMemberLinks := []string{
		"../e2e/at/secops/history/follow-up-5/counted-flow.json",
		"../e2e/at/secops/history/follow-up-5/post-revoke.json",
		"../e2e/at/secops/history/follow-up-5/deadlines.json",
		"../e2e/at/secops/history/follow-up-5/capacity.json",
		"ab96acac-8713-485a-9c29-88f20110e2e5",
		"../e2e/runs/2026-10-30/followup10-run3/device-revoke.md",
		"../e2e/requests.md",
		"StartSecondExit",
		"FVPN_E2E_GRACE_SECONDS",
		"clock_settime",
	}
	for _, link := range requiredMemberLinks {
		if !strings.Contains(zhStatusLine, link) {
			t.Errorf("docs/handover/member-guide.md zh-TW DOC15 paragraph missing %q", link)
		}
		if !strings.Contains(enStatusLine, link) {
			t.Errorf("docs/handover/member-guide.md en DOC15 paragraph missing %q", link)
		}
	}
	if !strings.Contains(zhStatusLine, "歷史快照") {
		t.Error("docs/handover/member-guide.md zh-TW DOC15 paragraph must attribute earlier blocked records as historical")
	}
	if !strings.Contains(enStatusLine, "historical records") || !strings.Contains(enStatusLine, "do not represent current `main` status") {
		t.Error("docs/handover/member-guide.md en DOC15 paragraph must attribute earlier blocked records as historical")
	}

	operatorGuide := readOwnedDoc(t, root, "docs/handover/operator-guide.md")
	for _, token := range []string{
		"contract revision 8 (#164)",
		"#164", "#165", "#166", "#167", "#168", "#169", "#170", "#171", "#172", "#174",
		"ISOLATION_POLICY_INVALID",
		"GENERATION_MISMATCH",
		"HOOK_PRIORITY_MISMATCH",
		"COUNTER_ORDER_INVALID",
		"FLOWTABLE_BYPASS",
		"DEVICE_CLASS_MISSING",
		"QOS_CLASS_MISSING",
		"MARK_SEMANTICS_VIOLATED",
		"PROBE_UNBOUNDED",
		"conntrack delete pending; serving ruleset verified",
		"pending_revocations",
		"effectiveJob",
		"probe_omitted",
		"DisallowUnknownFields",
		"checkLargeJSONDocument",
		"trusted_proxies",
		"ParseTrustedProxies",
		"StartSecondExit",
	} {
		if !strings.Contains(operatorGuide, token) {
			t.Errorf("docs/handover/operator-guide.md missing expected follow-up 6 claim %q", token)
		}
	}

	deployGuide := readOwnedDoc(t, root, "docs/handover/deploy.md")
	for _, token := range []string{
		"r.RemoteAddr",
		"trusted_proxies",
		"ParseTrustedProxies",
		"trusting the world lets any peer choose its bucket",
		"#174",
		"conntrack delete pending; serving ruleset verified",
		"pending_revocations",
		"ExtraDeniedV4",
		"probe_omitted",
		"DisallowUnknownFields",
	} {
		if !strings.Contains(deployGuide, token) {
			t.Errorf("docs/handover/deploy.md missing expected follow-up 6 claim %q", token)
		}
	}

	calibrationGuide := readOwnedDoc(t, root, "docs/handover/calibration.md")
	for _, token := range []string{
		"probe_omitted",
		"DisallowUnknownFields",
		"checkLargeJSONDocument",
		"anonClass",
		"trusted_proxies",
	} {
		if !strings.Contains(calibrationGuide, token) {
			t.Errorf("docs/handover/calibration.md missing expected follow-up 6 claim %q", token)
		}
	}

	r12 := readOwnedDoc(t, root, "docs/reviews/r12-enforce-limits-review.md")
	for _, token := range []string{
		"844b59a32124ca6ada6939ef81dd406aebce2b51",
		"d5a2bed55ef0cc81c9d6f48b76ddf501ba5e6f47",
		"9c67e4b9eafffc312ead24e9bf023702285ee4f36a95423e2f8b5a488ad4a7b6",
		"agy/gemini-4-argon-high",
		"#167",
		"#168",
		"#169",
		"#171",
		"#172",
		"#174",
	} {
		if !strings.Contains(r12, token) {
			t.Errorf("docs/reviews/r12-enforce-limits-review.md missing expected provenance/closing token %q", token)
		}
	}

}

func TestHandover_Followup7Claims(t *testing.T) {
	root := repoRoot(t)

	// 1. Parse contracts/openapi.yaml for the live operation count and FleetConfigUpdate max_offline_lease_seconds bounds (60..86400).
	// Revision 14 advances the count to 50 (revokeMemberGrant). This test still pins follow-up 7's fleet-config decisions.
	rawSpec, err := os.ReadFile(filepath.Join(root, "contracts", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read contracts/openapi.yaml: %v", err)
	}
	var spec struct {
		Paths      map[string]map[string]any `yaml:"paths"`
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Minimum int `yaml:"minimum"`
					Maximum int `yaml:"maximum"`
				} `yaml:"properties"`
			} `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(rawSpec, &spec); err != nil {
		t.Fatalf("parse contracts/openapi.yaml: %v", err)
	}
	opCount := 0
	for _, path := range spec.Paths {
		for method := range path {
			if slices.Contains([]string{"get", "put", "post", "delete", "patch", "options", "head", "trace"}, method) {
				opCount++
			}
		}
	}
	if opCount != 50 {
		t.Fatalf("contracts/openapi.yaml operation count = %d, want 50", opCount)
	}
	if _, ok := spec.Paths["/admin/fleet-config"]; !ok {
		t.Fatal("contracts/openapi.yaml missing /admin/fleet-config path")
	}
	fcProp := spec.Components.Schemas["FleetConfigUpdate"].Properties["max_offline_lease_seconds"]
	if fcProp.Minimum != 60 || fcProp.Maximum != 86400 {
		t.Fatalf("FleetConfigUpdate.max_offline_lease_seconds bounds = %d..%d, want 60..86400", fcProp.Minimum, fcProp.Maximum)
	}

	// 2. Verify operator-guide.md fleet-config section, the wired-operation claim, and PR #173..#182 catch-up claims.
	requireHandoverClaimTokens(t, root, "tests/handover/testdata/history/follow-up-15/operator-guide.md", "The existing 49 are wired in `apps/control` on `main`")
	operatorGuide := readOwnedDoc(t, root, "docs/handover/operator-guide.md")
	for _, token := range []string{
		"#173", "#174", "#175", "#176", "#177", "#178", "#179", "#181", "#182",
		"D-065",
		"/api/v1/admin/fleet-config",
		"max_offline_lease_seconds",
		"default_max_offline_lease_seconds",
		"`60`–`86400`",
		"min(120 s, window/2)",
		"900 s",
		"428 PRECONDITION_REQUIRED",
		"412 RESOURCE_VERSION_CONFLICT",
		"409 IDEMPOTENCY_CONFLICT",
		"422 VALIDATION_FAILED",
		"fleet_config.update",
		"fleet_config:00000000-0000-0000-0000-000000000001",
		"TestD065_FleetConfigHTTPReadWrite",
		"TestD065_AT29_FleetConfigOnlyChangesNewLeases",
		"TestD065_FleetConfigReleaseWriterInterleaving",
		"TestD065_FleetConfigMissingRowUsesIssuerDefault",
		"w4-enroll",
		"12000",
		"66000",
		"F5-01",
		"F6-02",
		"docs/updates/central-lifecycle.md",
		"shared development host, not candidate hardware",
		"904.360 s",
		"300.386 s",
		"ScaleNativeMembers",
		"D-066",
		"gateway-principals",
		"-grace-seconds",
		"PauseNativeControl",
		"StartNodeTelemetry",
		"TestMTR03_RunSkipsBusySampleWithoutLosingCounters",
	} {
		if !strings.Contains(operatorGuide, token) {
			t.Errorf("docs/handover/operator-guide.md missing expected follow-up 7 token %q", token)
		}
	}

	// 3. Verify deploy.md and calibration.md fleet-config and devstack follow-up 11 claims.
	deployGuide := readOwnedDoc(t, root, "docs/handover/deploy.md")
	for _, token := range []string{
		"D-065",
		"#176",
		"#179",
		"#182",
		"/api/v1/admin/fleet-config",
		"max_offline_lease_seconds",
		"default_max_offline_lease_seconds",
		"`60`–`86400`",
		"min(120 s, window/2)",
		"fleet_config.update",
		"428 PRECONDITION_REQUIRED",
		"412 RESOURCE_VERSION_CONFLICT",
		"409 IDEMPOTENCY_CONFLICT",
		"422 VALIDATION_FAILED",
		"TestD065_AT29_FleetConfigOnlyChangesNewLeases",
		"gateway-principals",
		"-grace-seconds",
		"FVPN_E2E_GRACE_SECONDS",
	} {
		if !strings.Contains(deployGuide, token) {
			t.Errorf("docs/handover/deploy.md missing expected follow-up 7 token %q", token)
		}
	}

	calibrationGuide := readOwnedDoc(t, root, "docs/handover/calibration.md")
	for _, token := range []string{
		"D-065",
		"#176",
		"#179",
		"#178",
		"#182",
		"/api/v1/admin/fleet-config",
		"max_offline_lease_seconds",
		"`60`–`86400` seconds",
		"min(120 s, window/2)",
		"fleet_config.update",
		"TestD065_AT29_FleetConfigOnlyChangesNewLeases",
		"shared development host, not candidate hardware",
		"StartNodeTelemetry",
		"-grace-seconds",
	} {
		if !strings.Contains(calibrationGuide, token) {
			t.Errorf("docs/handover/calibration.md missing expected follow-up 7 token %q", token)
		}
	}

	// 4. Verify r13-r12-fixes-review.md archive provenance and README.md row.
	r13 := readOwnedDoc(t, root, "docs/reviews/r13-r12-fixes-review.md")
	for _, token := range []string{
		"12496fd0d057632919d8f1153b92a9a660fdc520",
		"e66393db6a16b7ce94472564e526a7254720114a",
		"13bc71172ff626baded35dee6993448ca6d0df842e08c1bd766f8085e3f13134",
		"agy/gemini-4-argon-high",
		"H-1", "H-2", "H-3", "H-4",
		"M-1", "M-2", "M-3", "M-4", "M-5",
		"L-1", "L-2", "L-3",
	} {
		if !strings.Contains(r13, token) {
			t.Errorf("docs/reviews/r13-r12-fixes-review.md missing expected provenance/finding token %q", token)
		}
	}
	reviewsREADME := readOwnedDoc(t, root, "docs/reviews/README.md")
	if !strings.Contains(reviewsREADME, "r13-r12-fixes-review.md") || !strings.Contains(reviewsREADME, "12496fd") {
		t.Error("docs/reviews/README.md missing r13-r12-fixes-review archive entry")
	}
}

func TestHandover_Followup8Claims(t *testing.T) {
	root := repoRoot(t)

	// 1. Verify OpenAPI 1.7.0 revision 11 carries listNodeIncarnations, Enrollment.token_sha256, and awaiting_member_grants.
	rawSpec, err := os.ReadFile(filepath.Join(root, "contracts", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read contracts/openapi.yaml: %v", err)
	}
	var spec struct {
		Paths      map[string]map[string]any `yaml:"paths"`
		Components struct {
			Schemas map[string]struct {
				Properties map[string]any `yaml:"properties"`
				Enum       []string       `yaml:"enum"`
			} `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(rawSpec, &spec); err != nil {
		t.Fatalf("parse contracts/openapi.yaml: %v", err)
	}
	if _, ok := spec.Paths["/admin/nodes/{node_id}/incarnations"]; !ok {
		t.Fatal("contracts/openapi.yaml missing /admin/nodes/{node_id}/incarnations path")
	}
	if _, ok := spec.Components.Schemas["Enrollment"].Properties["token_sha256"]; !ok {
		t.Fatal("contracts/openapi.yaml Enrollment schema missing token_sha256")
	}
	if !slices.Contains(spec.Components.Schemas["ForwardingBlockReason"].Enum, "awaiting_member_grants") {
		t.Fatal("contracts/openapi.yaml ForwardingBlockReason enum missing awaiting_member_grants")
	}

	// 2. Verify all 47 revision 11 operations are wired on main (no "Forty-six" or "46 of 47" wording remains).
	requireHandoverClaimTokens(t, root, "tests/handover/testdata/history/follow-up-15/operator-guide.md", "The existing 49 are wired in `apps/control` on `main`")
	operatorGuide := readOwnedDoc(t, root, "docs/handover/operator-guide.md")
	handoverREADME := readOwnedDoc(t, root, "docs/handover/README.md")
	opsREADME := readOwnedDoc(t, root, "docs/ops/README.md")
	for name, doc := range map[string]string{
		"docs/handover/operator-guide.md": operatorGuide,
		"docs/handover/README.md":         handoverREADME,
		"docs/ops/README.md":              opsREADME,
	} {
		for _, stale := range []string{"Forty-six", "forty-six", "46 of 47", "not implemented in this revision"} {
			if strings.Contains(doc, stale) {
				t.Errorf("%s still contains stale unwired operation wording %q", name, stale)
			}
		}
	}

	// 3. Verify operator-guide.md pins for control follow-up 19/20 (#190/#215), revision 11 (#191/#195/#200/#213),
	// D-067 (#193/#198/#201/#202/#206/#216), D-068 (#189), updater follow-up 12 (#192), and secops follow-up 7 (#203).
	for _, token := range []string{
		"#186", "#187", "#189", "#190", "#191", "#192", "#193", "#194", "#195", "#196", "#197", "#198", "#199", "#200", "#201", "#202", "#203",
		// Control follow-up 19/20 (#190, #215): parse rule, certificate-key, LRU tail walk, /48 reserve-of-one budget, single-walk body identity.
		"bracketed input",
		"an unbracketed multi-colon string is parsed whole before a trailing port is removed",
		"eight hextets written with a zero-width `::` (`2001:db8:1::2:3:4:5:6`) are `2001:db8:1:2:3:4:5:6`, `/64` `2001:db8:1:2::/64`",
		"one `/48` may hold at most `min(max_rate_principals/2, 64)`",
		"no verified chain is `401 UNAUTHENTICATED` before the limiter",
		"a verified leaf keys ingress by its `urn:fvpn:node:<uuid>` URI SAN (`node:<uuid>`)",
		"and `sha256(leaf.Raw)` is the key only for a verified leaf without that SAN",
		"spent the token",
		"eviction walks from the tail to the first bucket idle at full tokens or idle for more than an hour, bounded by `max_rate_principals`",
		"single streaming walk (no `json.Unmarshal` of the body)",
		"sets `ExcludeRequestBody`",
		"returns `501 NOT_IMPLEMENTED` without a third decode",
		"order for a large body is `400 MALFORMED_REQUEST` syntax, `422 VALIDATION_FAILED` schema, `403 INCARNATION_MISMATCH` identity",
		// Contract revision 11 (#191, #195, #200, #213): incarnations, token_sha256, awaiting_member_grants.
		"GET /api/v1/admin/nodes/{node_id}/incarnations",
		"listNodeIncarnations",
		"retire_reason",
		"\"reinstall\"",
		"\"revocation\"",
		"\"supersession\"",
		"token_sha256",
		"enrollment.created",
		"changes.after.token_sha256",
		"awaiting_member_grants",
		// Agent follow-up 15-18 (#193, #198, #202, #216 — D-067) & conntrack retry bound.
		"D-067",
		"**provisional** and renews after **one heartbeat**, then **doubling**, capped at **`60 s`**",
		"Serving lease has no member grants yet.",
		"**twelve consecutive failed `conntrack -D` attempts**",
		"member_forwarding_enabled=false",
		"safety_status=failed",
		"ConntrackDeleteError{Pending, Failed}",
		"(addr, device_id, member_id)",
		// Fleet-update follow-up 9 (#189 — D-068).
		"D-068",
		"advances **only** through an accepted `UpdateResult`",
		"**only when `nodes.current_release_id` is `NULL`**",
		"superseded attempt's `job_id`",
		"do not move the batch read model",
		// Updater follow-up 12 (#192).
		"blankHealth",
		"probe_omission/<txn_id>/<nonce>",
		"cleanTxnNotes",
		// Secops follow-up 7 (#203) verdicts & numbers.
		"ab96acac-8713-485a-9c29-88f20110e2e5",
		"subset passed, case blocked on",
		"5,960/5,960",
		"60/60",
	} {
		if !strings.Contains(operatorGuide, token) {
			t.Errorf("docs/handover/operator-guide.md missing expected follow-up 8 token %q", token)
		}
	}

	// 4. Verify deploy.md and calibration.md pins for #190, #192, #193, #200, #203, #215.
	deployGuide := readOwnedDoc(t, root, "docs/handover/deploy.md")
	for _, token := range []string{
		"#190", "#192", "#193", "#200", "#202", "#215",
		"bracketed input is never split, an unbracketed multi-colon string is parsed whole before a trailing port is removed, and eight hextets written with a zero-width `::` (`2001:db8:1::2:3:4:5:6`) are `2001:db8:1:2:3:4:5:6`, `/64` `2001:db8:1:2::/64`",
		"the agent-listener ingress key is the verified leaf certificate's `urn:fvpn:node:<uuid>` URI SAN",
		"so an address-rewriting proxy does not choose the bucket",
		"twelve consecutive failed `conntrack -D` attempts",
		"blankHealth",
		"token_sha256",
	} {
		if !strings.Contains(deployGuide, token) {
			t.Errorf("docs/handover/deploy.md missing expected follow-up 8 token %q", token)
		}
	}

	calibrationGuide := readOwnedDoc(t, root, "docs/handover/calibration.md")
	for _, token := range []string{
		"#190", "#192", "#193", "#198", "#200", "#203", "#215",
		"D-067",
		"awaiting_member_grants",
		"Serving lease has no member grants yet.",
		"renews after one heartbeat, then doubling, capped at `60 s`",
		"bracketed input is never split, an unbracketed multi-colon string is parsed whole before a trailing port is removed, and eight hextets written with a zero-width `::` `2001:db8:1::2:3:4:5:6` are `2001:db8:1:2:3:4:5:6`, `/64` `2001:db8:1:2::/64`",
		"one `/48` may hold at most `min(max_rate_principals/2, 64)`",
		"no verified chain is `401 UNAUTHENTICATED` before the limiter",
		"a verified leaf keys ingress by its `urn:fvpn:node:<uuid>` URI SAN (`node:<uuid>`), so every still-usable serial of a node shares one burst during renewal overlap, and `sha256(leaf.Raw)` is the key only for a verified leaf without that SAN",
		"eviction walks from the tail to the first bucket idle at full tokens or idle for more than an hour, bounded by `max_rate_principals`",
		"identity is read during the single streaming walk (no `json.Unmarshal` of the body)",
		"sets `ExcludeRequestBody`",
		"returns `501 NOT_IMPLEMENTED` without a third decode; order for a large body is `400` syntax, `422` schema, `403` identity",
		"blankHealth",
		"ab96acac-8713-485a-9c29-88f20110e2e5",
	} {
		if !strings.Contains(calibrationGuide, token) {
			t.Errorf("docs/handover/calibration.md missing expected follow-up 8 token %q", token)
		}
	}

	// 5. Verify DOC16 in member-guide.md (no mutable root counted-flow/post-revoke links) and live sha256 pins in scope-review.json follow_up_7.files.
	memberGuide := readOwnedDoc(t, root, "docs/handover/member-guide.md")
	for _, forbidden := range []string{
		"(../e2e/at/secops/counted-flow.json)",
		"(../e2e/at/secops/post-revoke.json)",
	} {
		if strings.Contains(memberGuide, forbidden) {
			t.Errorf("docs/handover/member-guide.md still links follow-up 5 evidence to mutable root path %s", forbidden)
		}
	}
	if !strings.Contains(memberGuide, "DOC16") || !strings.Contains(memberGuide, "ab96acac-8713-485a-9c29-88f20110e2e5") {
		t.Error("docs/handover/member-guide.md missing DOC16 or follow-up 7 run ID ab96acac-8713-485a-9c29-88f20110e2e5")
	}

}

func TestHandover_Followup9Claims(t *testing.T) {
	root := repoRoot(t)

	// 1. Verify revision 11 schema elements still present after revision 14 advanced info.version to 1.10.0:
	// ErrorCode.AWAITING_MEMBER_GRANTS (D-071) and NodeDetail.identity_conflict / NodeIdentityConflict (D-072).
	rawSpec, err := os.ReadFile(filepath.Join(root, "contracts", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read contracts/openapi.yaml: %v", err)
	}
	var spec struct {
		Info struct {
			Version string `yaml:"version"`
		} `yaml:"info"`
		Components struct {
			Schemas map[string]struct {
				Properties map[string]any `yaml:"properties"`
				Enum       []string       `yaml:"enum"`
				AllOf      []struct {
					Properties map[string]any `yaml:"properties"`
				} `yaml:"allOf"`
			} `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(rawSpec, &spec); err != nil {
		t.Fatalf("parse contracts/openapi.yaml: %v", err)
	}
	if spec.Info.Version != "1.10.0" {
		t.Fatalf("contracts/openapi.yaml info.version = %q, want 1.10.0", spec.Info.Version)
	}
	if !slices.Contains(spec.Components.Schemas["ErrorCode"].Enum, "AWAITING_MEMBER_GRANTS") {
		t.Fatal("contracts/openapi.yaml ErrorCode enum missing AWAITING_MEMBER_GRANTS")
	}
	hasIdentityConflict := false
	for _, part := range spec.Components.Schemas["NodeDetail"].AllOf {
		if _, ok := part.Properties["identity_conflict"]; ok {
			hasIdentityConflict = true
		}
	}
	if !hasIdentityConflict {
		t.Fatal("contracts/openapi.yaml NodeDetail schema missing identity_conflict")
	}
	for _, field := range []string{"mapping_frozen", "conflict_count", "last_conflict_at"} {
		if _, ok := spec.Components.Schemas["NodeIdentityConflict"].Properties[field]; !ok {
			t.Fatalf("contracts/openapi.yaml NodeIdentityConflict schema missing %s", field)
		}
	}

	// 2. Current runbooks must not advertise superseded contracts. Archives retain historical scope.
	for _, rel := range ownedDocs {
		if strings.HasPrefix(rel, "docs/reviews/") {
			continue
		}
		doc := readOwnedDoc(t, root, rel)
		for _, stale := range []string{"revision 10", "Revision 10", "1.6.0"} {
			if strings.Contains(doc, stale) {
				t.Errorf("%s still contains stale contract version/revision string %q", rel, stale)
			}
		}
	}

	// 3. Verify operator-guide.md pins for follow-up 9: revision 11 (#213), reading the provisional window
	// (D-067/D-071/#206/#216), cloned-identity runbook (D-072/#211/#213), revoke nodes step outcomes (D-070/#206),
	// worker contention (#205/#208), control follow-up 20 (#215), and acceptance cross-references (#204/#207/#210/#212/#214).
	operatorGuide := readOwnedDoc(t, root, "docs/handover/operator-guide.md")
	for _, token := range []string{
		"OpenAPI `1.10.0`, revision 14",
		"#204", "#205", "#206", "#207", "#208", "#210", "#211", "#212", "#213", "#214", "#215", "#216",
		"D-070", "D-071", "D-072",
		// Provisional window (D-067, D-071, #206, #213, #216).
		"ErrorCode.AWAITING_MEMBER_GRANTS",
		"the `lease`, `qos`, and `source_binding` checks as `failed` with `AWAITING_MEMBER_GRANTS`",
		"one `errors[]` entry at `stage=lease`",
		"ProvisionalGrantlessLeaseSeconds = 60",
		"`D-067` backstop, `#206`",
		"expires such a grant-less lease after **`60 s`**",
		"This is not an incident; `LEASE_EXPIRED` / `lease_expired` is.",
		"persists beyond a few minutes, check desired bindings",
		"member's entitlement",
		// Cloned or duplicated node identity (D-072, #211, #213) runbook.
		"Cloned or Duplicated Node Identity Runbook (`D-072`",
		"ObserveEndpointTx",
		"IPv4 `/24`",
		"IPv6 `/48`",
		"DefaultIdentityActiveWindow = 120 * time.Second",
		"Roaming rule (`X → Y`, not a conflict)",
		"Interleaving rule (conflict)",
		"node_identity_conflicts",
		"mapping_frozen = true",
		"pki_mapping_frozen",
		"provisioning_state = 'failed'",
		"safety_status = 'failed'",
		"member_forwarding_observed = false",
		"node.identity.conflict",
		"NodeDetail.identity_conflict",
		"conflict_count",
		"last_conflict_at",
		"D-055",
		"revokeNode",
		"starts unfrozen (`mapping_frozen = false`)",
		"There is no unfreeze operation.",
		"Two clones inside one `/24` or `/48` are not distinguishable; a multi-homed exit presenting two addresses is the operator’s declaration, not a clone",
		// Revoke nodes step outcomes (D-070, #206, #216).
		"conntrack_delete_pending",
		"conntrack_delete_succeeded",
		"node_failed_closed",
		"A second registered node of the exit group still reporting a pending `{code: \"ISOLATION_POLICY_INVALID\", stage: \"apply\"}` error while forwarding holds the `\"nodes\"` step",
		"migrated with nil `DeviceID`/`MemberID`",
		// Worker contention is not failure (#205, #208).
		"Worker Contention Is Not Failure (`#205`, `#208`",
		"Operations are **never** marked `failed` because two worker goroutines handled the same member concurrently.",
		"pg_try_advisory_lock(hashtextextended(device_id, 120))",
		"discovery_decision",
		"attributed to `fleet_config:00000000-0000-0000-0000-000000000001` (`fleet_config`), never to an invented member",
		"operations are never observed `failed` in the live suite",
		// Control follow-up 20 (#215) exact sentences.
		"one `/48` may hold at most `min(max_rate_principals/2, 64)`",
		"idle replacement is allowed only within the same group",
		"no verified chain is `401 UNAUTHENTICATED` before the limiter",
		"sha256(leaf.Raw)",
		"eviction walks from the tail to the first bucket idle at full tokens or idle for more than an hour",
		// Acceptance cross-references (#204, #207, #210, #212, #214).
		"c359fa37-61e0-4469-9f73-5a1e60949c3b",
		"AT10 `7 pass / 1 blocked / 0 fail`",
		"**subset passed, case blocked** for every full AT",
		"`AT13` `60 s` whole-node expiry",
		"finite-member expiry under control loss pass",
		"105/105",
		"d50da75fd14fc72c",
		"2393df75-6261-42bc-99b4-df3c57c4b741",
	} {
		if !strings.Contains(operatorGuide, token) {
			t.Errorf("docs/handover/operator-guide.md missing expected follow-up 9 token %q", token)
		}
	}

	// 4. Verify calibration.md and ops/README.md pins for follow-up 9.
	calibrationGuide := readOwnedDoc(t, root, "docs/handover/calibration.md")
	for _, token := range []string{
		"#206", "#213", "#215", "#216",
		"D-071",
		"AWAITING_MEMBER_GRANTS",
		"ProvisionalGrantlessLeaseSeconds = 60",
		"expires such a grant-less lease after `60 s`",
		"one `/48` may hold at most `min(max_rate_principals/2, 64)`",
		"a verified leaf keys ingress by its `urn:fvpn:node:<uuid>` URI SAN (`node:<uuid>`), so every still-usable serial of a node shares one burst during renewal overlap, and `sha256(leaf.Raw)` is the key only for a verified leaf without that SAN",
		"eviction walks from the tail to the first bucket idle at full tokens or idle for more than an hour, bounded by `max_rate_principals`",
	} {
		if !strings.Contains(calibrationGuide, token) {
			t.Errorf("docs/handover/calibration.md missing expected follow-up 9 token %q", token)
		}
	}

	opsREADME := readOwnedDoc(t, root, "docs/ops/README.md")
	for _, token := range []string{
		"#204", "#205", "#206", "#207", "#208", "#210", "#211", "#212", "#213", "#214", "#215", "#216",
		"c359fa37-61e0-4469-9f73-5a1e60949c3b",
		"AT10 `7 pass / 1 blocked / 0 fail`",
		"**subset passed, case blocked** for every full AT",
		"finite-member expiry, and whole-node `60 s` expiry passed",
		"105/105",
		"d50da75fd14fc72c",
		"2393df75-6261-42bc-99b4-df3c57c4b741",
	} {
		if !strings.Contains(opsREADME, token) {
			t.Errorf("docs/ops/README.md missing expected follow-up 9 token %q", token)
		}
	}
}

func TestHandover_Followup10Claims(t *testing.T) {
	root := repoRoot(t)

	// 1. Verify operator-guide.md pins for follow-up 10:
	// - Cloned-identity runbook addition (#217, #223) + docs/portal/README.md §6 cross-reference
	// - Revoke completion fence (exit-group + 90 s freshness) and one-shot stale-generation grace in node.observation_applied audit (#217, D-075 note)
	// - Whole-ruleset nftables read-back (#220)
	// - Fair-Use enforcement reach and fair_use.applying convergence (#222)
	// - D-074 gated node.publish waiting state (#224)
	// - Acceptance cross-references for devstack follow-up 13 (#219) and secops follow-up 8 (#221)
	operatorGuide := readOwnedDoc(t, root, "docs/handover/operator-guide.md")
	for _, token := range []string{
		"#217", "#219", "#220", "#221", "#222", "#223", "#224",
		"D-074",
		// Cloned-identity projection & portal cross-reference (#217, #223).
		"identity_conflict.mapping_frozen: true",
		"participates in the node strong `ETag`",
		"304 Not Modified",
		"mapping_frozen: false, conflict_count: 0",
		"the new incarnation resets the projection",
		"../portal/README.md#6-operator-node-detail-notes-provisional-lease-waiting-copy--identity-conflict-d-067-d-071-d-072",
		// Revoke completion fence & stale-generation grace (#217).
		"ObserveRevokedDeviceCutoffTx",
		"DefaultStaleTimeout = 90 s",
		"stale or ungrouped/unrelated nodes never hold a revoke",
		"awaiting_serving_exit_observation",
		"staleGenerationGraceAvailableTx",
		"once per generation change",
		"node.observation_applied",
		"stale_generation_grace: true",
		"D-075",
		// Whole-ruleset nftables read-back (#220).
		"node/enforce/verify_foreign.go",
		"nft -j list ruleset",
		"table <t> family <f> hook <h> priority <p> chain <c> handle <n> statement <s>",
		"Production exit nodes must not run Docker or any other NAT for non-loopback traffic",
		// Fair-Use enforcement reach (#222).
		"Fair-Use Enforcement Reach and `fair_use.applying` Convergence",
		"fup.member_generation_changed",
		"Compiler.RecompileBatch",
		"computeMemberFUPApplying",
		"\"look at the node\", not \"look at the policy\"",
		"`plan`-only rate changes (`policy.activated` with `domain = \"plan\"`) wait for the next compile trigger",
		// D-074 gated node.publish waiting state (#224).
		"When a `node.publish` operation stays `running` with `agent_policy` or `data_path_probe` in `status: \"pending\"`, `error_code: \"NODE_NOT_SERVICEABLE\"`, and `message: \"member_forwarding_unconfirmed (lease_unconfirmed)\"`, central is waiting for the node to apply the latest compiled configuration generation and report passing serving checks (`exits.publication_observed` remains unchanged, or `\"withdrawn\"` if an empty-exception route was withdrawn first); the step clears automatically on the next qualifying observation and worker pass, and transitions to `status: \"failed\"` (`publication_observed: \"error\"`) only if the worker retry budget (`worker_attempts` = 10 attempts, ~4.3–8.5 minutes of jittered exponential backoff from 1s to 300s) is exhausted, at which point you should verify node health and re-issue `PUT /api/v1/admin/nodes/{node_id}/publication`.",
		// Acceptance cross-references (#219, #221).
		"InjectMemberIngress",
		"ObserveMemberIngress",
		"ctunku",
		"4c97ebef",
		"26 pass / 1 blocked / 0 fail",
		"0f5fb60d-f6a7-45c6-8ebe-175e0a480bd8",
		"105.94 s",
		"111.51 s",
		"116.651 s",
		"120.12 s",
		"SECOPS-18",
		"D-076",
	} {
		if !strings.Contains(operatorGuide, token) {
			t.Errorf("docs/handover/operator-guide.md missing expected follow-up 10 token %q", token)
		}
	}

	// 2. Verify calibration.md pins for #220 (whole-ruleset read-back), #222 (Fair-Use recompile & fair_use.applying), and #224 (D-074 retry budget).
	calibrationGuide := readOwnedDoc(t, root, "docs/handover/calibration.md")
	for _, token := range []string{
		"#220", "#222", "#224",
		"D-074",
		"Compiler.RecompileBatch",
		"fup.member_generation_changed",
		"computeMemberFUPApplying",
		"\"look at the node\", not \"look at the policy\"",
		"`plan`-only rate changes",
		"255.5 s..511 s",
		"~4.3–8.5 minutes",
		"node/enforce/verify_foreign.go",
		"nft -j list ruleset",
		"table <t> family <f> hook <h> priority <p> chain <c> handle <n> statement <s>",
		"Production exit nodes must not run Docker or any other NAT for non-loopback traffic",
	} {
		if !strings.Contains(calibrationGuide, token) {
			t.Errorf("docs/handover/calibration.md missing expected follow-up 10 token %q", token)
		}
	}

	// 3. Verify restore-drill.md and deploy.md pins for #221 (measured AT30 5-step protocol, RTO/RPO numbers, and dated D-076 pointer) and #220.
	restoreDrill := readOwnedDoc(t, root, "docs/handover/restore-drill.md")
	for _, token := range []string{
		"#221",
		"0f5fb60d-f6a7-45c6-8ebe-175e0a480bd8",
		"3.64 s",
		"120.12 s",
		"ErrAckRegress",
		"105.94 s",
		"111.51 s",
		"116.651 s",
		"0 s",
		"SECOPS-18",
		"D-076",
		"review devices fenced as authorization-unconfirmed and re-authorize explicitly",
	} {
		if !strings.Contains(restoreDrill, token) {
			t.Errorf("docs/handover/restore-drill.md missing expected follow-up 10 token %q", token)
		}
	}

	deployGuide := readOwnedDoc(t, root, "docs/handover/deploy.md")
	for _, token := range []string{
		"#220", "#221",
		"node/enforce/verify_foreign.go",
		"nft -j list ruleset",
		"120.12 s",
		"SECOPS-18",
		"D-076",
		"review devices fenced as authorization-unconfirmed and re-authorize explicitly",
	} {
		if !strings.Contains(deployGuide, token) {
			t.Errorf("docs/handover/deploy.md missing expected follow-up 10 token %q", token)
		}
	}
}

func TestHandover_Followup11Claims(t *testing.T) {
	root := repoRoot(t)

	// 1. Verify operator-guide.md pins for Follow-up 11:
	// - D-076 (#227): device_authorization_fences (migration 00188), device.authorization.external_withdrawal, fvpn_device_restore_fence_decisions_total, exact quoted paragraph from docs/members/README.md, explicit AuthorizeDevice / member.device_authorize, dated pointer to revision 12 D-077.
	// - D-073 & D-075 (#228): node_identity_endpoints candidate_hash / candidate_first_seen_at / candidate_last_seen_at under SELECT ... FOR UPDATE, node_incarnations.stale_generation_grace_at (migration 00187), decisions never read audit_events.
	// - fvpn-agent usage snapshot (#225): -config /etc/vpn-node/agent-run.json [-json], fields, exit codes 0/1/2/3/4/5/64, ApplyACK non-torn rule, installer re-run preservation.
	// - SECOPS-19 (#226): ObservationReport.usage carries live UsageOutboxStatus in node_observations.report_payload, dated pointer to revision 12 NodeDetail.usage_outbox.
	// - D-078 & acceptance cross-references (#229, #231): Retry-After 429 rule, control follow-up 21 dated note, enrollment follow-up 3 (run 60a7102e, ENR-08 closed, AT06 254 pass / 1 blocked, AT07 15 pass / 7 blocked), members follow-up 6 (run 80db103b, AT02 42/0 strict fvmb0 ingress injection, AT09 36.976 s / 41.695 s).
	operatorGuide := readOwnedDoc(t, root, "docs/handover/operator-guide.md")
	for _, token := range []string{
		"#225", "#226", "#227", "#228", "#229", "#231",
		// D-076 (#227) & exact owner quote.
		"D-076",
		"db/migrations/00188_device_authorization_fences.sql",
		"device_authorization_fences",
		"device.authorization.external_withdrawal",
		"fvpn_device_restore_fence_decisions_total",
		"device authorization unconfirmed during ledger restore",
		"GET /api/v1/admin/audit-events?resource_type=member_device&resource_id=<device_id>",
		"While `usage_ledger_state.mode` is `frozen` or `reconciling`, each device that `SyncDevice` newly fences is logged once as `device authorization unconfirmed during ledger restore` with `device_id`, `member_id`, `peer_id`, and `ledger_mode`, and counted in `fvpn_device_restore_fence_decisions_total` on the worker registry. Those device ids need an explicit re-authorization decision after `complete-recovery` if the member is still entitled.",
		"Entitlement restoration does not clear a fence; only an explicit re-authorization (`AuthorizeDevice`, operation kind `member.device_authorize`, actor `system` or `admin`, new idempotency key) or a revoke does",
		"D-077",
		"MemberDevice.authorization_unconfirmed",
		// D-073 & D-075 (#228).
		"D-073",
		"D-075",
		"db/migrations/00187_identity_roaming_and_readiness_grace.sql",
		"candidate_hash",
		"candidate_first_seen_at",
		"candidate_last_seen_at",
		"SELECT ... FOR UPDATE",
		"node_incarnations.stale_generation_grace_at",
		"readiness and grace decisions never read `audit_events`",
		// fvpn-agent usage snapshot (#225).
		"fvpn-agent usage snapshot -config /etc/vpn-node/agent-run.json",
		"node/agent/cmd/fvpn-agent/usage_snapshot.go",
		"docs/node/usage-store.md",
		"acked_through",
		"unacked_count",
		"unacked_payload_bytes",
		"unacked_events",
		"payload_sha256",
		// SECOPS-19 (#226).
		"SECOPS-19",
		"ObservationReport.usage",
		"UsageOutboxStatus",
		"unacked_event_count",
		"oldest_unacked_at",
		"node/agent/usage_status.go",
		"node_observations.report_payload",
		"NodeDetail.usage_outbox",
		// D-078 & acceptance cross-references (#229, #231).
		"D-078",
		"a `429` that carries `Retry-After` and whose retry succeeds is the limiter working as specified; harnesses report the counts as evidence, and calibration stays with `control`",
		"TestFollowup21_BrowserClassCalibration",
		"60a7102e-b824-4927-a576-62033f458157",
		"254 pass / 1 blocked / 0 fail",
		"rate_limit_review",
		"15 pass / 7 blocked / 0 fail",
		"80db103b-622c-42b1-946d-4628daaa213d",
		"3 pass / 7 blocked / 0 fail",
		"36.976 s",
		"41.695 s",
		"31 -> 32",
	} {
		if !strings.Contains(operatorGuide, token) {
			t.Errorf("docs/handover/operator-guide.md missing expected follow-up 11 token %q", token)
		}
	}

	// 2. Verify deploy.md and restore-drill.md pins for #227 (D-076 post-complete-recovery step) and #228 (D-073 durable roaming candidate across two replicas).
	deployGuide := readOwnedDoc(t, root, "docs/handover/deploy.md")
	for _, token := range []string{
		"#227", "#228",
		"D-073",
		"D-076",
		"D-077",
		"db/migrations/00187_identity_roaming_and_readiness_grace.sql",
		"db/migrations/00188_device_authorization_fences.sql",
		"candidate_hash",
		"candidate_first_seen_at",
		"candidate_last_seen_at",
		"SELECT ... FOR UPDATE",
		"device_authorization_fences",
		"device.authorization.external_withdrawal",
		"device authorization unconfirmed during ledger restore",
		"fvpn_device_restore_fence_decisions_total",
	} {
		if !strings.Contains(deployGuide, token) {
			t.Errorf("docs/handover/deploy.md missing expected follow-up 11 token %q", token)
		}
	}

	restoreDrill := readOwnedDoc(t, root, "docs/handover/restore-drill.md")
	for _, token := range []string{
		"#227",
		"D-076",
		"D-077",
		"db/migrations/00188_device_authorization_fences.sql",
		"device_authorization_fences",
		"device.authorization.external_withdrawal",
		"device authorization unconfirmed during ledger restore",
		"fvpn_device_restore_fence_decisions_total",
		"6.3 Review Devices Fenced as Authorization-Unconfirmed",
	} {
		if !strings.Contains(restoreDrill, token) {
			t.Errorf("docs/handover/restore-drill.md missing expected follow-up 11 token %q", token)
		}
	}

	// 3. Verify member-guide.md (zh-TW and en) pins for D-076 (#227) and members follow-up 6 (#231).
	memberGuide := readOwnedDoc(t, root, "docs/handover/member-guide.md")
	for _, token := range []string{
		"#227", "#231",
		"D-076",
		"在服務備份還原後，若某台裝置的授權狀態尚待營運人員確認，該裝置亦可能暫時顯示為 `pending`，直到營運人員完成確認為止，會員端的任何操作均不會改變此狀態",
		"After a service restore, a device may show as `pending` until an operator confirms it, and nothing you do on the member side changes that",
		"80db103b-622c-42b1-946d-4628daaa213d",
		"36.976",
		"41.695",
	} {
		if !strings.Contains(memberGuide, token) {
			t.Errorf("docs/handover/member-guide.md missing expected follow-up 11 token %q", token)
		}
	}

	// 4. Verify calibration.md pins for D-078 (#229) and BrowserRate 10/s burst 30 / AgentRate 5/s burst 10.
	calibrationGuide := readOwnedDoc(t, root, "docs/handover/calibration.md")
	for _, token := range []string{
		"#229",
		"D-078",
		"`browser_rate` (`10/s`, burst `30`",
		"`agent_rate` (`5/s`, burst `10`)",
		"a `429` that carries `Retry-After` and whose retry succeeds is the limiter working as specified; harnesses report the counts as evidence, and calibration stays with `control`",
		"TestFollowup21_BrowserClassCalibration",
	} {
		if !strings.Contains(calibrationGuide, token) {
			t.Errorf("docs/handover/calibration.md missing expected follow-up 11 token %q", token)
		}
	}

	// 5. Verify ops/README.md and handover/README.md index pins.
	for _, rel := range []string{"docs/ops/README.md", "docs/handover/README.md"} {
		doc := readOwnedDoc(t, root, rel)
		for _, token := range []string{
			"#225", "#226", "#227", "#228", "#229", "#231",
			"D-073", "D-075", "D-076", "D-078",
		} {
			if !strings.Contains(doc, token) {
				t.Errorf("%s missing expected follow-up 11 token %q", rel, token)
			}
		}
	}
}

func TestHandover_Followup12Claims(t *testing.T) {
	root := repoRoot(t)

	restoreDrill := readOwnedDoc(t, root, "docs/handover/restore-drill.md")
	wantFenceQuery := strings.Join([]string{
		"SELECT f.device_id, d.member_id, d.peer_id, d.status, f.fenced_at, f.observed_group_ids",
		"     FROM device_authorization_fences f",
		"     JOIN member_devices d ON d.id = f.device_id",
		"     ORDER BY f.fenced_at DESC;",
	}, "\n")
	if !strings.Contains(restoreDrill, wantFenceQuery) {
		t.Errorf("docs/handover/restore-drill.md missing corrected D-076 fence review SQL query:\n%s", wantFenceQuery)
	}

	operatorGuide := readOwnedDoc(t, root, "docs/handover/operator-guide.md")
	for _, docName := range sqlCheckedDocs(t, root) {
		content := readOwnedDoc(t, root, docName)
		for _, forbidden := range []string{
			"SELECT device_id, member_id, peer_id, reason, fenced_at",
			"columns `device_id`, `member_id`, `fenced_at`, `reason`",
			"`device_authorization_fences` (`reason =",
		} {
			if strings.Contains(content, forbidden) {
				t.Errorf("%s still contains stale D-076 fence schema text %q", docName, forbidden)
			}
		}
	}

	for _, rel := range []string{"docs/handover/restore-drill.md", "docs/handover/operator-guide.md"} {
		content := readOwnedDoc(t, root, rel)
		for _, token := range []string{
			"`device_id`, `fenced_at`, `observed_group_ids`",
			"member_devices",
			"device.authorization.external_withdrawal",
			"device authorization unconfirmed during ledger restore",
		} {
			if !strings.Contains(content, token) {
				t.Errorf("%s missing expected follow-up 12 token %q", rel, token)
			}
		}
	}
	if !strings.Contains(operatorGuide, "-- [mechanical-check: mutating-maintenance-sql]") {
		t.Error("docs/handover/operator-guide.md missing -- [mechanical-check: mutating-maintenance-sql] marker on purge_retained SQL fence")
	}
}

// TestHandover_Followup13Claims ties the operator statements to contract,
// configuration and owner evidence. OpenAPI version/count remain derived by
// TestHandover_OpenAPIClaims, rather than a second hardcoded contract inventory.
func TestHandover_Followup13Claims(t *testing.T) {
	root := repoRoot(t)
	require := func(path string, tokens ...string) {
		t.Helper()
		requireHandoverClaimTokens(t, root, path, tokens...)
	}
	t.Run("restore_authorization_contract_and_implementation", func(t *testing.T) {
		for _, path := range []string{"docs/handover/operator-guide.md", "docs/handover/deploy.md", "docs/handover/restore-drill.md"} {
			require(path, "GET /api/v1/admin/devices?authorization_unconfirmed=true", "POST /api/v1/admin/members/{member_id}/devices/{device_id}/authorize", "authorization_unconfirmed_since", "Idempotency-Key", "CSRF", "member.device_authorize", "409 STATE_CONFLICT", "409 DEVICE_LIMIT_REACHED", "authorization_unconfirmed: false")
		}
		require("docs/handover/operator-guide.md", "existing `member.device_authorize` operation", "after the worker sync", "device.authorization.external_withdrawal", "device.authorize", "only path that re-grants", "always", "optional in the contract", "no request body", "#247")
		require("docs/handover/member-guide.md", "需營運人員重新授權", "needs operator re-authorization", "authorization_unconfirmed: true", "always emits", "absent means the server does not project it yet")
		require("internal/members/httpadapter/admin_devices_integration_test.go", "TestD077_AdminDeviceListAndReauthorize")
		if got := apierr.DeviceLimitReached.HTTPStatus(); got != 409 {
			t.Fatalf("D-081 HTTP status = %d, want 409", got)
		}
		var contract struct {
			Paths map[string]map[string]struct {
				OperationID string               `yaml:"operationId"`
				Responses   map[string]yaml.Node `yaml:"responses"`
			} `yaml:"paths"`
		}
		if err := yaml.Unmarshal([]byte(readOwnedDoc(t, root, "contracts/openapi.yaml")), &contract); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct{ path, method, id, status string }{
			{"/admin/devices", "get", "listAdminDevices", "200"},
			{"/admin/members/{member_id}/devices/{device_id}/authorize", "post", "authorizeMemberDevice", "202"},
		} {
			op := contract.Paths[tc.path][tc.method]
			if op.OperationID != tc.id {
				t.Errorf("%s operation = %q, want %q", tc.path, op.OperationID, tc.id)
			}
			if _, ok := op.Responses[tc.status]; !ok {
				t.Errorf("%s missing response %s", tc.path, tc.status)
			}
		}
	})
	t.Run("usage_projection_and_migration", func(t *testing.T) {
		const migration = "00189_node_incarnation_usage_outbox.sql"
		require("docs/handover/11-database-and-retention.md", "`node_incarnations.usage_outbox` JSONB", "`usage_outbox_observed_at` timestamp")
		for _, path := range []string{"docs/handover/00-index.md", "docs/handover/11-database-and-retention.md", "docs/handover/operator-guide.md"} {
			require(path, migration)
		}
		require("docs/handover/operator-guide.md", "usage_outbox: null", "never absent", "ETag", "informational", "meter.Health", "EvaluateGate", "pure reads", "until a verified recovery checkpoint", "ApplyCheckpoint", "central_acked_through")
		require("internal/nodes/httpwire/handlers_integration_test.go", "func TestSECOPS19_D077_HTTPNodeDetailUsageOutboxETagAndDomainColumn")
		require("internal/nodes/observation_readiness_integration_test.go", "func TestSECOPS19_D077_NodeDetailUsageOutboxProjectionAndMigration")
		require("node/internal/usagestore/status_test.go", "func TestOutboxStatusIsPure")
		require("docs/handover/operator-guide.md", "ledger_epoch", "until a verified recovery checkpoint", "meter.Health")
		// Guard against the old regex interpreting fence/audit counts as migrations.
		claims, errs := auditMigrationInventoryClaims("restore-evidence", "fence count `1`; audit count `1`", repoMigrationIDs(t, root))
		if len(claims)+len(errs) != 0 {
			t.Fatalf("non-migration counters mistaken for inventory: %v %v", claims, errs)
		}
	})
	t.Run("browser_calibration", func(t *testing.T) {
		c := config.Defaults()
		if c.BrowserRate != (config.Bucket{PerSecond: 10, Burst: 30}) || c.AgentRate != (config.Bucket{PerSecond: 5, Burst: 10}) || c.IngressRate != (config.Bucket{PerSecond: 20, Burst: 60}) {
			t.Fatalf("calibration defaults changed: browser=%+v agent=%+v ingress=%+v", c.BrowserRate, c.AgentRate, c.IngressRate)
		}
		require("docs/handover/calibration.md", "#233", "#browser-class-calibration", "TestFollowup21_BrowserClassCalibration", "internal/httpapi/middleware_browser_rate_test.go", "half the `20/s` burst `60`", "`3` GETs", "`5` GETs")
		require("internal/httpapi/middleware_browser_rate_test.go", "func TestFollowup21_BrowserClassCalibration")
	})
	t.Run("AT30_measured_JSON_and_generation_floor", func(t *testing.T) {
		var evidence struct {
			Status     string          `json:"status"`
			RunID      string          `json:"run_id"`
			Evaluation map[string]bool `json:"evaluation"`
			Comparison map[string]struct {
				RPO         float64 `json:"pre_replay_db_rpo_seconds"`
				ServiceRPO  float64 `json:"post_replay_service_rpo_seconds"`
				ProtocolRTO float64 `json:"protocol_rto_seconds"`
				TotalRTO    float64 `json:"total_outage_rto_seconds"`
			} `json:"rpo_rto_comparison"`
		}
		// D-098 — the current file now carries follow-up 10's run, which handover follow-up 14 adopts.
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/history/follow-up-9/restore-reconcile.json")), &evidence); err != nil {
			t.Fatal(err)
		}
		if evidence.Status != "passed" {
			t.Fatalf("AT30 status %q", evidence.Status)
		}
		for _, name := range []string{"t0_t1_device_revocation_fenced_d076", "regrant_authorize_device_clears_fence", "negative_central_revoke_not_fenced", "negative_post_restore_fresh_enroll_not_fenced", "secops19_visibility_and_zero_convergence", "retained_node_usage_replay", "provider_reconciliation"} {
			if !evidence.Evaluation[name] {
				t.Errorf("AT30 evidence %s is not passed", name)
			}
		}
		m := evidence.Comparison["followup_9_current"]
		if m.ServiceRPO != 0 {
			t.Fatalf("service RPO = %v", m.ServiceRPO)
		}
		for _, path := range []string{"docs/handover/restore-drill.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md"} {
			require(path, evidence.RunID, fmt.Sprintf("%.2f s", m.ProtocolRTO), fmt.Sprintf("%.2f s", m.TotalRTO), fmt.Sprintf("%.3f s", m.RPO), "0 B", "SECOPS-18", "SECOPS-19", "SECOPS-21", "#245", "ErrStaleGeneration", "config_generation.desired", "nodes.applied_config_generation")
		}
		require("docs/handover/restore-drill.md", "7.17 s", "3/3", "unacked_count = 3", "unacked_event_count = 3", "2352", "ErrAckRegress` log count `0`", "service-level `AuthorizeDevice`", "peerM2D2", "peerM2D3", "peerM2D1", "6.2a Verify post-restore generation")
		require("internal/desiredstate/compiler_test.go", "func TestNextConfigGenerationIncludesApplied")
		require("internal/desiredstate/compiler_integration_test.go", "func TestCompiler_RestoreAppliedGenerationAdvancesPastNode")
	})
	t.Run("installer_and_foreign_rules", func(t *testing.T) {
		require("docs/handover/operator-guide.md", "docs/node/install.md", "already_current", "check --json", "update-staging/pending-release.json", "does not move `current`", "reenroll --token-file", "identity.retired/<utc>/", "keeps `usage.db`", "image-prepare", "firstboot", "`--token` on argv", "exit `64`", "`latest` is refused", "journal-first", "`0` success, `1` failed, `2` drifted, `3` busy, `4` unsupported platform, `5` dependency, `64` usage", "hardware clone boot", "ENR-02", "D-080", "fvpn-updater apply --pending")
		require("packaging/installer/install_test.go", "TestInstallIdempotentAndUpgradeKeepsIdentity", "TestUpgradeDoesNotMoveCurrent", "TestReenrollKeepsUsage")
		for _, path := range []string{"docs/handover/calibration.md", "docs/handover/deploy.md"} {
			require(path, "D-079", "loopback_scoped", "127.0.0.0/8", "::1", "`xt` matches", "names prove nothing", "handle <n>", "Production exit nodes must not run Docker or any other NAT for non-loopback traffic")
		}
		require("docs/handover/calibration.md", "not a production topology")
		require("node/enforce/followup11_loopback_test.go", "func TestFollowup11_DockerDNSLoopbackScoped", "func TestFollowup11_NonLoopbackMasquerade", "func TestFollowup11_XTMatchDoesNotProve")
	})
	t.Run("acceptance_counts_keep_case_blockers", func(t *testing.T) {
		for _, id := range []string{"AT10", "AT11", "AT12", "AT13"} {
			var record struct {
				RunID  string `json:"run_id"`
				Status string `json:"status"`
				Checks []struct {
					Status string `json:"status"`
				} `json:"checks"`
			}
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/isolation/history/follow-up-5/"+id+".json")), &record); err != nil {
				t.Fatal(err)
			}
			counts := map[string]int{}
			for _, check := range record.Checks {
				counts[check.Status]++
			}
			if record.Status != "blocked" {
				t.Errorf("%s status changed to %s; refresh dated handover", id, record.Status)
			}
			require("docs/ops/README.md", record.RunID, fmt.Sprintf("%s `%d pass / %d blocked / %d fail`", id, counts["pass"], counts["blocked"], counts["fail"]))
		}
		require("docs/ops/README.md", "#237", "#246", "**subset passed, case blocked**", "superseded by usage follow-up 4 (#257)", "29 pass / 0 blocked / 0 fail", "16 pass / 0 blocked / 0 fail", "121.88 s")
		require("tests/handover/testdata/4e3e9487-fe0f-4d88-a396-d5d5fa4a8e1d-run.md", "pass=29 blocked=0 fail=0", "pass=16 blocked=0 fail=0")
	})
	t.Run("review_provenance_and_finding_identity", func(t *testing.T) {
		require("docs/reviews/r14-d067-d070-review.md", "70efbde6c94fb8d87f2cb88334703cfff4afe004", "96a379e53b8744f71b51322d36ec4f098a96feb2e6e2d9dd29c5e03484b82b60", "**M-1** (agent): A v2", "**M-4** (control): One node with two", "**L-1** (control): Eight depleted", "**L-2** (control): One anonymous", "#215", "#216", "#217", "#220")
		for _, path := range []string{"internal/httpapi/followup20_review_test.go", "node/agent/review_r14_m1_test.go", "node/agent/followup15_test.go", "internal/nodes/observation_readiness_integration_test.go", "node/enforce/followup10_review_test.go", "tests/netlab/scenario_foreign_test.go"} {
			require("docs/reviews/r14-d067-d070-review.md", path)
			if _, err := os.Stat(filepath.Join(root, path)); err != nil {
				t.Fatal(err)
			}
		}
		require("docs/reviews/r15-d072-d075-review.md", "dfafae0f985a5209ccf380afb9c615987f2bd775", "3929d4ca5ec00703e1d2dd9ae6306f9acf5aa3b445f288be31d4ee50acffbbbb", "grok/grok-4.7", "Owner fixes adopted — 2026-10-31 EDT", "**H-1**", "**M-6**", "**L-1**", "Adopted proof files on main")
		require("docs/reviews/README.md", "r14-d067-d070-review.md", "r15-d072-d075-review.md", "8 (1 High, 6 Medium, 1 Low)")
	})
	t.Run("no_obsolete_followup13_claims", func(t *testing.T) {
		for _, path := range []string{"docs/handover/README.md", "docs/ops/README.md", "docs/handover/operator-guide.md", "docs/handover/deploy.md", "docs/handover/restore-drill.md"} {
			text := readOwnedDoc(t, root, path)
			for _, stale := range []string{"until wire-members follow-up 10", "all 47 wired", "422 DEVICE_LIMIT_REACHED", "contract revision 12 `D-077`),", "update-journal/pending-release.json", "install/pending-release.json", "follow-up 5 in flight"} {
				if strings.Contains(text, stale) {
					t.Errorf("%s has stale follow-up 13 text %q", path, stale)
				}
			}
		}
	})
}

// Live document-integrity checks are deliberately separate from dated evidence claims.
func TestHandover_ScopeReviewPins(t *testing.T) {
	root := repoRoot(t)
	t.Run("manifest_set_0", func(t *testing.T) {
		var scopeReview struct {
			FollowUp6         map[string]any `json:"follow_up_6"`
			HandoverFollowUp6 struct {
				ReviewedDocuments []struct {
					Path   string `json:"path"`
					SHA256 string `json:"sha256"`
				} `json:"reviewed_documents"`
			} `json:"handover_follow_up_6"`
		}
		rawScope := readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")
		if err := json.Unmarshal([]byte(rawScope), &scopeReview); err != nil {
			t.Fatalf("unmarshal docs/e2e/at/secops/scope-review.json: %v", err)
		}
		if len(scopeReview.FollowUp6) == 0 {
			t.Error("docs/e2e/at/secops/scope-review.json missing follow_up_6 entry")
		}
		if len(scopeReview.HandoverFollowUp6.ReviewedDocuments) != 7 {
			t.Fatalf("docs/e2e/at/secops/scope-review.json handover_follow_up_6.reviewed_documents count = %d, want 7", len(scopeReview.HandoverFollowUp6.ReviewedDocuments))
		}
		for _, entry := range scopeReview.HandoverFollowUp6.ReviewedDocuments {
			raw, err := os.ReadFile(filepath.Join(root, entry.Path))
			if err != nil {
				t.Errorf("read %s: %v", entry.Path, err)
				continue
			}
			sum := sha256.Sum256(raw)
			got := hex.EncodeToString(sum[:])
			if got != entry.SHA256 {
				t.Errorf("docs/e2e/at/secops/scope-review.json handover_follow_up_6 sha256 for %s = %s, want %s", entry.Path, entry.SHA256, got)
			}
		}
	})
	t.Run("manifest_set_1", func(t *testing.T) {
		var scopeReview struct {
			FollowUp7 struct {
				Files []struct {
					Path   string `json:"path"`
					SHA256 string `json:"sha256"`
				} `json:"files"`
				NewRequests []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"new_requests"`
			} `json:"follow_up_7"`
		}
		rawScope := readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")
		if err := json.Unmarshal([]byte(rawScope), &scopeReview); err != nil {
			t.Fatalf("unmarshal docs/e2e/at/secops/scope-review.json: %v", err)
		}
		if len(scopeReview.FollowUp7.Files) != 7 {
			t.Fatalf("docs/e2e/at/secops/scope-review.json follow_up_7.files count = %d, want 7", len(scopeReview.FollowUp7.Files))
		}
		for _, entry := range scopeReview.FollowUp7.Files {
			raw, err := os.ReadFile(filepath.Join(root, entry.Path))
			if err != nil {
				t.Errorf("read %s: %v", entry.Path, err)
				continue
			}
			sum := sha256.Sum256(raw)
			got := hex.EncodeToString(sum[:])
			if got != entry.SHA256 {
				t.Errorf("docs/e2e/at/secops/scope-review.json follow_up_7 sha256 for %s = %s, want %s", entry.Path, entry.SHA256, got)
			}
		}
		if len(scopeReview.FollowUp7.NewRequests) == 0 || scopeReview.FollowUp7.NewRequests[0].ID != "DOC16" || !strings.HasPrefix(scopeReview.FollowUp7.NewRequests[0].Status, "resolved") {
			t.Errorf("docs/e2e/at/secops/scope-review.json follow_up_7.new_requests DOC16 status = %+v, want resolved", scopeReview.FollowUp7.NewRequests)
		}
	})
	t.Run("manifest_set_2", func(t *testing.T) {
		var review struct {
			Followup struct {
				Documents []struct {
					Path   string `json:"path"`
					SHA256 string `json:"sha256"`
				} `json:"reviewed_documents"`
			} `json:"handover_follow_up_13"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &review); err != nil {
			t.Fatal(err)
		}
		if len(review.Followup.Documents) < 13 {
			t.Fatal("follow-up 13 document pins incomplete")
		}
		for _, doc := range review.Followup.Documents {
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, doc.Path)))
			if hex.EncodeToString(sum[:]) != doc.SHA256 {
				t.Errorf("%s SHA-256 pin is stale", doc.Path)
			}
		}
	})
	t.Run("followup15_live_document_hashes", func(t *testing.T) {
		var manifest struct {
			Followup struct {
				Documents []struct {
					Path   string `json:"path"`
					SHA256 string `json:"sha256"`
				} `json:"reviewed_documents"`
			} `json:"handover_follow_up_15"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
			t.Fatal(err)
		}
		if len(manifest.Followup.Documents) != 10 {
			t.Fatalf("follow-up 15 document count = %d, want 10", len(manifest.Followup.Documents))
		}
		for _, doc := range manifest.Followup.Documents {
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, doc.Path)))
			if hex.EncodeToString(sum[:]) != doc.SHA256 {
				t.Errorf("live document hash stale: %s", doc.Path)
			}
		}
	})
	t.Run("followup17_live_document_hashes", func(t *testing.T) {
		var manifest struct {
			Followup struct {
				Documents []struct {
					Path   string `json:"path"`
					SHA256 string `json:"sha256"`
				} `json:"reviewed_documents"`
			} `json:"handover_follow_up_17"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
			t.Fatal(err)
		}
		if len(manifest.Followup.Documents) != 13 {
			t.Fatalf("follow-up 17 live inventory = %d, want 13", len(manifest.Followup.Documents))
		}
		for _, doc := range manifest.Followup.Documents {
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, doc.Path)))
			if hex.EncodeToString(sum[:]) != doc.SHA256 {
				t.Errorf("live document hash stale: %s", doc.Path)
			}
		}
	})
	t.Run("followup19_live_document_hashes", func(t *testing.T) {
		var manifest struct {
			Followup struct {
				Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
			} `json:"handover_follow_up_19"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
			t.Fatal(err)
		}
		if len(manifest.Followup.Documents) != 12 {
			t.Fatalf("follow-up 19 live inventory %d", len(manifest.Followup.Documents))
		}
		for _, pin := range manifest.Followup.Documents {
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, pin.Path)))
			if hex.EncodeToString(sum[:]) != pin.SHA256 {
				t.Errorf("live hash stale: %s", pin.Path)
			}
		}
	})
	t.Run("followup20_live_document_hashes", func(t *testing.T) {
		var manifest struct {
			Followup struct {
				Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
			} `json:"handover_follow_up_20"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
			t.Fatal(err)
		}
		if len(manifest.Followup.Documents) != 13 {
			t.Fatalf("follow-up 20 live inventory %d", len(manifest.Followup.Documents))
		}
		for _, pin := range manifest.Followup.Documents {
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, pin.Path)))
			if hex.EncodeToString(sum[:]) != pin.SHA256 {
				t.Errorf("live hash stale: %s", pin.Path)
			}
		}
	})

	t.Run("followup21_live_document_hashes", func(t *testing.T) {
		var manifest struct {
			Followup struct {
				Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
			} `json:"handover_follow_up_21"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
			t.Fatal(err)
		}
		if len(manifest.Followup.Documents) != 13 {
			t.Fatalf("follow-up 21 live inventory %d", len(manifest.Followup.Documents))
		}
		for _, pin := range manifest.Followup.Documents {
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, pin.Path)))
			if hex.EncodeToString(sum[:]) != pin.SHA256 {
				t.Errorf("live hash stale: %s", pin.Path)
			}
		}
	})

}

// TestHandover_Followup14Claims checks the dated main boundary, implemented
// contracts and the owner measurements. Numeric evidence is always a history
// snapshot or run-ID-named fixture (D-098), never a lane's current record.
func TestHandover_Followup14Claims(t *testing.T) {
	root := repoRoot(t)
	require := func(path string, tokens ...string) {
		t.Helper()
		requireHandoverClaimTokens(t, root, path, tokens...)
	}
	const op = "docs/handover/operator-guide.md"
	const ops = "docs/ops/README.md"
	const fixtures = "tests/handover/testdata/"
	t.Run("merged_roadmap_boundary", func(t *testing.T) {
		var evidence struct {
			Commit string   `json:"commit"`
			Rows   []string `json:"rows"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fixtures+"followup14-main.json")), &evidence); err != nil {
			t.Fatal(err)
		}
		if evidence.Commit != "c6e3eca92d3813aeec4a4c212b204fd32648ca5a" {
			t.Fatal("wrong main snapshot")
		}
		rows := strings.Join(evidence.Rows, "\n")
		for _, pr := range []int{247, 250, 260, 264, 252, 253, 254, 259, 256, 262, 257, 258, 261, 263, 268, 266, 265, 255, 267, 269, 270, 271, 272} {
			if !strings.Contains(rows, fmt.Sprintf("#%d)", pr)) {
				t.Errorf("missing merged PR #%d", pr)
			}
		}
		for _, coming := range []string{"| w2-fleet-update follow-up 10 |", "| w4-at-enrollment follow-up 5 |", "| w3-wire-nodes follow-up 29 |"} {
			if strings.Contains(rows, coming) {
				t.Errorf("coming boundary changed: %s", coming)
			}
		}
	})

	t.Run("revision13_and_freeze_producer_boundary", func(t *testing.T) {
		var contract struct {
			Components struct {
				Schemas map[string]struct {
					Type       string   `yaml:"type"`
					Enum       []string `yaml:"enum"`
					Required   []string `yaml:"required"`
					Default    string   `yaml:"default"`
					Properties map[string]struct {
						Type string `yaml:"type"`
					} `yaml:"properties"`
				} `yaml:"schemas"`
			} `yaml:"components"`
		}
		if err := yaml.Unmarshal([]byte(readOwnedDoc(t, root, "contracts/openapi.yaml")), &contract); err != nil {
			t.Fatal(err)
		}
		accepted := contract.Components.Schemas["ObservationAccepted"]
		if accepted.Properties["identity_frozen"].Type != "boolean" || slices.Contains(accepted.Required, "identity_frozen") {
			t.Fatal("identity_frozen must remain optional boolean")
		}
		source := contract.Components.Schemas["UpdateResultSource"]
		if !slices.Equal(source.Enum, []string{"central", "installer"}) || source.Default != "central" {
			t.Fatalf("source enum/default: %+v", source)
		}
		code, ok := apierr.Lookup(apierr.TCForeignFilter)
		if !ok || code.HTTPStatus != 500 || code.Retryable || !code.NodeLocal {
			t.Fatalf("TC_FOREIGN_FILTER classification: %+v", code)
		}
		if apierr.DeviceLimitReached.HTTPStatus() != 409 {
			t.Fatal("D-081 must be 409")
		}
		require(op, "optional boolean", "emitted since #267", "report-ID replays and HTTP idempotency-key replays", "immutable receipt", "durable incarnation latch", "Frozen observations and renewals omit generation ACKs", "restart cannot reopen", "500-class, non-retryable, node-local", "D-093")
		require("internal/nodes/httpwire/followup28_integration_test.go", "TestD088_AT07_AT29_HTTPObservationFreezeLiveReceipt", "TestD093_ISO02_HTTPTCForeignFilterStoredEvidence")
		require("docs/handover/member-guide.md", "identity_frozen` has no member-visible surface", "non-serving", "Fair-Use summary")
	})
	t.Run("D084_clamp_and_migration", func(t *testing.T) {
		require(op, "The freeze transaction atomically marks the incarnation frozen / not ready / safety failed and clamps every matching stored lease beyond transaction time + 60 s to that deadline (migration 00190 permits only shortening `expires_at`).", "Signed envelopes and same-ID replay are unchanged", "A server clamp alone is not a 60-second packet-stop guarantee.", "Two clones inside one `/24` or `/48` are not distinguishable; a multi-homed exit presenting two addresses is the operator’s declaration, not a clone")
		for _, path := range []string{"docs/handover/00-index.md", "docs/handover/11-database-and-retention.md"} {
			require(path, "00190_serving_leases_clamp.sql")
		}
		require("internal/nodes/httpwire/review_r15_clamp_integration_test.go", "TestD088_ServingLeaseClampMigrationRoundTrip")
		require("internal/nodes/httpwire/review_r15_clamp_integration_test.go", "TestReviewR15_StoredLeaseClampGuardProof")
	})
	t.Run("D085_live_gate_D090_agent_containment_D087_schedule", func(t *testing.T) {
		require(op, "before reconciliation, after Apply before the ACK, and before the heartbeat", "min(1 s, min(heartbeat, poll)/2)", "52.6 ms per serving step", "never acknowledges the offered generation", "Member links are administratively down until a clean verified Apply", "re-containment on the next step is proven, uninterrupted containment is not", "D-100", "quarantine_<interface>", "never deletes foreign tc objects", "D-096")
		require("node/agent/followup21_test.go", "TestD085_ISO01_BetweenApplyInjection", "TestD085_InjectionAfterApplyBeforeAck", "TestD085_ReadTimeoutAndCanceledStep")
		require("node/agent/followup21_test.go", "TestD085_ISO01_BetweenApplyInjection", "TestD085_InjectionAfterApplyBeforeAck", "TestD085_ReadTimeoutAndCanceledStep")
		require("node/agent/followup21_netlab_test.go", "TestD085_LiveReadNetlab")
		require("node/enforce/quarantine_test.go", "TestD090_ISO01_QuarantineAndVerifiedRecovery", "TestD090_QuarantinePartialFailures")
		for _, path := range []string{op, "docs/handover/calibration.md", "docs/handover/member-guide.md"} {
			require(path, "D-087", "every accepted grant-less lease", "Empty bindings do not stop renewal", "A populated lease additionally needs", "A scheduled request slot is not completed I/O")
		}
		require("node/agent/followup20_test.go", "TestD067_ReadyRenewsWithin60Seconds", "TestD067_GrantlessReplayKeepsDeadline")
	})
	t.Run("D083_D078_limiter_calibration", func(t *testing.T) {
		c := config.Defaults()
		if c.BrowserRate != (config.Bucket{PerSecond: 10, Burst: 30}) || c.ReadRate != (config.Bucket{PerSecond: 30, Burst: 60}) || c.IngressRate != (config.Bucket{PerSecond: 20, Burst: 60}) {
			t.Fatalf("limiter defaults changed: %+v", c)
		}
		for _, path := range []string{op, "docs/handover/calibration.md"} {
			require(path, "min(max_rate_principals/2, 64)", "per authenticated session UUID", "read_rate` `30/s`, burst `60`", "ingress `20/s`, burst `60` per class and address", "GET /admin/operations` is not a collection", "#233", "#252")
		}
		require("internal/httpapi/middleware_browser_rate_test.go", "TestFollowup21_BrowserClassCalibration")
		require("internal/httpapi/middleware_browser_rate_integration_test.go", "TestAT06_BrowserBucketsUseAuthenticatedSessionRows")
		require("internal/httpapi/review_r15_q6_ingress_test.go", "TestReviewR15_AnonymousReserveLeavesOneSlot")
	})
	t.Run("D080_D089_seed_and_pending_central_acceptance", func(t *testing.T) {
		require(op, "next `run` or `fvpn-updater apply --pending`", "manifest_path` plus `request_id`", "/proc/sys/kernel/random/boot_id", "fresh node exit `0`, no forwarding block", "install → verified seeded baseline → staged upgrade → journaled apply → local result → central acceptance", "fvpn-updater seed --config --manifest --tree", "atomically and idempotently", "not seeded (updater without seed)", "never a batch result", "central answers `404`", "`409` is terminal", "not end-to-end", "systemd container lifecycle gate did not run")
		require("node/updater/followup13_test.go", "TestUPD_F13_OneshotAppliesInstallerRelease", "TestUPD_F13_MissingDesiredStateIsNoWork")
		require("node/updater/seed_test.go", "TestD089_SeedThenInstallerApply", "TestD089_SeedRefusesUnverifiedTree")
		require("packaging/installer/handoff_test.go", "TestD089_PendingEnvelopeAndInvocationIdentity", "TestD089_RealUpdaterConsumesWriterRecord")
		require("packaging/installer/handoff_test.go", "TestAT26_InstallerUsesSharedKernelBootLock")
		require("docs/handover/calibration.md", "fs.inotify.max_user_instances", "`128` was exhausted")
		require("docs/handover/deploy.md", "ENR-06", "/var/run/netbird.sock", "installer follow-up 3", "21 pass / 1 fail")
	})
	t.Run("portal16_and_D091_restore_review", func(t *testing.T) {
		for _, path := range []string{op, "docs/handover/deploy.md", "docs/handover/restore-drill.md"} {
			require(path, "Portal follow-up 16 (#255)", "/admin/devices", "neutral receipt", "paginated read-back", "usage-outbox")
		}
		require(op, "D-091 (#263)", "no published exit carries this member", "node_identity_endpoints.mapping_frozen", "registered-node fallback reading `checks.applied_policy_refs` is deleted", "owner's Step 6")
		require("apps/worker/review_r15_q9_fup_integration_test.go", "TestReviewR15_UnpublishDropsTheCarryingExitAndApplyingSticks", "TestReviewR15_FrozenCarryingExitKeepsFairUseApplying")
		require(op, "owner's Step 6", "409 STATE_CONFLICT", "409 DEVICE_LIMIT_REACHED")
		require(op, "neutral receipt", "authorization_unconfirmed", "paginated read-back")
		require(op, "usage-outbox block")
	})
	t.Run("usage_followup4_measured_numbers", func(t *testing.T) {
		type record struct {
			RunID        string                     `json:"run_id"`
			Status       string                     `json:"status"`
			Observations map[string]json.RawMessage `json:"observations"`
		}
		var records = map[int]record{}
		for id := 14; id <= 19; id++ {
			var r record
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fmt.Sprintf(fixtures+"351d59aa-AT%d.json", id))), &r); err != nil {
				t.Fatal(err)
			}
			if r.RunID != "351d59aa" || r.Status != "pass" {
				t.Fatalf("AT%d: %+v", id, r)
			}
			records[id] = r
		}
		for _, tc := range []struct {
			name                       string
			issuance, serving, counted int
		}{{"bring_up", 5021, 5848, 6837}, {"agent_restart", 5022, 5819, 6135}} {
			var cycle struct {
				Issued  int  `json:"ready_to_populated_lease_issued_ms"`
				Serving int  `json:"ready_to_populated_lease_ms"`
				Counted int  `json:"ready_to_first_counted_byte_ms"`
				Expired bool `json:"lease_expired_reported_during_window"`
			}
			if err := json.Unmarshal(records[16].Observations["d067_provisional_lease_"+tc.name], &cycle); err != nil {
				t.Fatal(err)
			}
			if cycle.Issued != tc.issuance || cycle.Serving != tc.serving || cycle.Counted != tc.counted || cycle.Expired {
				t.Fatalf("%s: %+v", tc.name, cycle)
			}
		}
		for _, tc := range []struct {
			key string
			ms  int
		}{{"fup_transition_1_policy_activated", 7032}, {"fup_transition_2_member_generation_changed", 29864}} {
			var v struct {
				MS      int  `json:"wall_clock_ms_to_applying_false"`
				Cleared bool `json:"applying_cleared_to_false"`
				Both    bool `json:"both_exits_installed_tier2_tc"`
			}
			if err := json.Unmarshal(records[19].Observations[tc.key], &v); err != nil {
				t.Fatal(err)
			}
			if v.MS != tc.ms || !v.Cleared || !v.Both {
				t.Fatalf("%s: %+v", tc.key, v)
			}
		}
		var enr map[string]json.RawMessage
		if err := json.Unmarshal(records[19].Observations["enr03_per_incarnation_retained_usage"], &enr); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"old_incarnation_usage_fixed", "member_total_unchanged_on_reenroll", "new_traffic_accumulated_same_month"} {
			if string(enr[key]) != "true" {
				t.Errorf("ENR03 %s=%s", key, enr[key])
			}
		}
		require(ops, "351d59aa", "e8a5e04", "5021/5022 ms", "5848/5819 ms", "6837/6135 ms", "7032 ms", "29864 ms", "synthetic ACK/traffic workarounds removed", "no `LEASE_EXPIRED`")
	})
	t.Run("members_followup7_HTTP_and_counts", func(t *testing.T) {
		const run = "297faa31-2f90-4aab-8cf0-6818629f46c8"
		counts := map[string]int{}
		for _, id := range []int{1, 2, 3, 4, 5, 8, 9, 20, 21, 22} {
			var r struct {
				RunID  string `json:"run_id"`
				Status string `json:"status"`
			}
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fmt.Sprintf("docs/e2e/at/members/history/follow-up-7/AT%02d.json", id))), &r); err != nil {
				t.Fatal(err)
			}
			if r.RunID != run {
				t.Fatalf("wrong member run %s", r.RunID)
			}
			counts[r.Status]++
		}
		if counts["pass"] != 3 || counts["blocked"] != 7 || counts["fail"] != 0 {
			t.Fatalf("member counts %v", counts)
		}
		require("docs/e2e/at/members/history/follow-up-7/summary.md", "471 pass / 0 fail", "36.997 s", "41.729 s")
		require("docs/e2e/at/members/history/follow-up-7/D077.md", "STATE_CONFLICT", "DEVICE_LIMIT_REACHED", "fence_after=0", "fence_count=1")
		require(ops, run, "3 pass / 7 externally blocked / 0 fail", "36.997 s", "41.729 s", "fence `1→0` only on explicit `202` grant", "Portal rendering of D-077 was not exercised by members")
	})
	t.Run("devstack15_counts_keep_ENR06_failure", func(t *testing.T) {
		for _, tc := range []struct {
			run                 string
			pass, blocked, fail int
		}{{"2347f2a4-02aa-4f56-94f2-c3bde88b5a6d", 32, 1, 0}, {"12d10041-9d33-4fbb-a4b0-9d2a21bce785", 21, 0, 1}} {
			var evidence struct {
				Records []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
					RunID  string `json:"run_id"`
					Hash   string `json:"source_sha256"`
				} `json:"records"`
			}
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fixtures+tc.run+"-verdicts.json")), &evidence); err != nil {
				t.Fatal(err)
			}
			counts := map[string]int{}
			for _, r := range evidence.Records {
				counts[r.Status]++
				if r.RunID != tc.run || len(r.Hash) != 64 {
					t.Fatal("invalid source pin")
				}
				if r.Status == "fail" && r.ID != "installer-wrappers" {
					t.Fatalf("unexpected failed scenario %s", r.ID)
				}
			}
			if counts["pass"] != tc.pass || counts["blocked"] != tc.blocked || counts["fail"] != tc.fail {
				t.Fatalf("devstack counts %v", counts)
			}
			require(ops, tc.run)
		}
		require(ops, "32 pass / 1 blocked / 0 fail", "21 pass / 1 fail", "IPv4-only native fixture", "ENR-06")
	})
	t.Run("AT30_followup10_and_D095_D099", func(t *testing.T) {
		const run = "0e8ff9a4-4fa1-4154-ae35-847a5be4d985"
		var e struct {
			Status     string                    `json:"status"`
			RunID      string                    `json:"run_id"`
			Comparison map[string]map[string]any `json:"measurement_comparison"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/history/follow-up-10/restore-reconcile.json")), &e); err != nil {
			t.Fatal(err)
		}
		if e.Status != "passed" || e.RunID != run {
			t.Fatalf("AT30: %+v", e)
		}
		for key, want := range map[string]float64{"Pre-replay database RPO (s)": 95.507504, "Post-replay service usage RPO (s)": 0, "Protocol RTO (s)": 93.532554563, "Total outage RTO (s)": 106.868537734} {
			if e.Comparison["follow_up_10"][key] != want {
				t.Errorf("AT30 %s=%v want %v", key, e.Comparison["follow_up_10"][key], want)
			}
		}
		for _, id := range []string{"SECOPS-16", "SECOPS-17", "SECOPS-18", "SECOPS-19", "SECOPS-20", "SECOPS-21", "D-077"} {
			var result struct {
				Status string `json:"status"`
			}
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/history/follow-up-10/"+id+".json")), &result); err != nil {
				t.Fatal(err)
			}
			want := "passed"
			if id == "SECOPS-21" {
				want = "blocked"
			}
			if result.Status != want {
				t.Errorf("%s=%s want %s", id, result.Status, want)
			}
		}
		require("docs/e2e/at/secops/history/follow-up-10/AT30.md", "95.507504", "93.532554563", "106.868537734")

		for _, path := range []string{op, "docs/handover/deploy.md", "docs/handover/restore-drill.md"} {
			require(path, run, "shared development host", "95.507504 s", "93.532554563 s", "106.868537734 s", "#242", "#221", "SECOPS-21 remains open", "pending fresh observation after restore", "before starting either service", "D-099", "first new offer `15`", "applied ACK")
		}
		require("internal/nodes/httpwire/restore_cache_probe_integration_test.go", "TestAT30_D095_HTTPRestoreCacheProbe")
		require("docs/handover/README.md", "Do not close SECOPS-21 from the compiler tests alone")
		require("internal/desiredstate/restore_integration_test.go", "TestAT30_D095_FirstOfferWaitsForFreshCurrentIncarnation", "TestAT30_D095_CompleteRecoveryDoesNotReleaseUnreportedNode")
	})
	t.Run("r15_adopted_proofs_and_coming_decisions", func(t *testing.T) {
		review := "docs/reviews/r15-d072-d075-review.md"
		require(review, "Owner fixes adopted — 2026-10-31 EDT", "#254", "#259", "#263", "#256", "#262", "#252", "#233", "r16 delivered on base `5575470`")
		for _, path := range []string{"internal/nodes/httpwire/review_r15_identity_integration_test.go", "internal/nodes/httpwire/review_r15_clamp_integration_test.go", "internal/nodes/httpwire/review_r15_q2_lock_integration_test.go", "internal/nodes/review_r15_q3_fence_integration_test.go", "internal/nodes/httpwire/review_r15_q4_publish_integration_test.go", "internal/nodes/review_r15_observation_integration_test.go", "apps/worker/review_r15_q9_fup_integration_test.go", "node/enforce/review_r15_q8_verify_test.go", "tests/netlab/review_r15_q8_bypass_test.go", "node/enforce/quarantine_test.go", "internal/httpapi/review_r15_q6_ingress_test.go"} {
			require(review, path)
			if _, err := os.Stat(filepath.Join(root, path)); err != nil {
				t.Fatal(err)
			}
		}
		require("docs/handover/README.md", "D-093", "D-097 **coming**", "D-098", "D-099", "D-100", "batch_id: null", "central job's projection wins", "Isolation follow-up 6", "Enrollment follow-up 5")
	})
	t.Run("D088_D090_D096_agent22_and_isolation6", func(t *testing.T) {
		require("node/agent/followup22_test.go", "TestD088_IdentityFrozenTLSAndRestart", "TestD088_AbsentAndFalseDoNotChangeIdentity", "TestD090_QuarantineRetainedModelEveryStepAndD096Recovery", "TestD090_ApplyQuarantineEvidenceTLS", "TestD090_AlreadyBlockedAndRestartRetainContainmentModel")
		require("node/agent/followup22_path_test.go", "TestD096_ReadOnlyMemberTopology")
		require(op, "consumed since #272", "persists the latch", "renews at once", "containment evidence only", "sanitized evidence", "applied peers’ member route", "no out-of-band override")
		const run = "bf671f3a-c6ac-4824-9107-425c4c5a5f02"
		var window struct {
			RunID   string `json:"run_id"`
			Cadence int    `json:"sample_cadence_ms"`
			Result  struct {
				Times  []string `json:"sample_times"`
				MS     int      `json:"ready_to_populated_ms"`
				Detail string   `json:"detail"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/isolation/history/follow-up-6/provisional-recovery.json")), &window); err != nil {
			t.Fatal(err)
		}
		if window.RunID != run || window.Cadence != 250 || len(window.Result.Times) != 19 || window.Result.MS != 4767 || !strings.Contains(window.Result.Detail, "5021 / 5025 ms") {
			t.Fatalf("provisional window: %+v", window)
		}
		for id := 10; id <= 13; id++ {
			var r struct {
				RunID  string `json:"run_id"`
				Status string `json:"status"`
				Checks []struct {
					Status string `json:"status"`
				} `json:"checks"`
			}
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fmt.Sprintf("docs/e2e/at/isolation/history/follow-up-6/AT%d.json", id))), &r); err != nil {
				t.Fatal(err)
			}
			want := "blocked"
			if id == 13 {
				want = "fail"
			}
			if r.RunID != run || r.Status != want {
				t.Fatalf("AT%d run/status: %+v", id, r)
			}
			counts := map[string]int{}
			for _, c := range r.Checks {
				counts[c.Status]++
			}
			require(ops, fmt.Sprintf("AT%d `%d pass / %d blocked / %d fail`", id, counts["pass"], counts["blocked"], counts["fail"]))
		}
		require("docs/e2e/at/isolation/history/follow-up-6/recovery-action.json", "latched TC_FOREIGN_FILTER", "docker restart")
		require(ops, run, "`19` samples at `250 ms`", "4,767 ms", "assisted recovery", "not an unattended clean pass", "AT13 **fail**", "post-deadline drop passed")
	})

	t.Run("D098_snapshot_integrity", func(t *testing.T) {
		var pins []struct {
			Path   string `json:"path"`
			Hash   string `json:"sha256"`
			Source string `json:"source"`
			Commit string `json:"source_commit"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fixtures+"followup14-snapshots.json")), &pins); err != nil {
			t.Fatal(err)
		}
		runName := regexp.MustCompile(`^(?:[0-9a-f]{8}|[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12})-`)
		for _, pin := range pins {
			if (!(strings.HasPrefix(pin.Path, fixtures) && runName.MatchString(filepath.Base(pin.Path))) && !strings.HasPrefix(pin.Path, "docs/e2e/at/members/history/follow-up-7/")) || len(pin.Commit) != 40 {
				t.Fatalf("unpinned evidence %+v", pin)
			}
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, pin.Path)))
			if hex.EncodeToString(sum[:]) != pin.Hash {
				t.Errorf("snapshot changed: %s", pin.Path)
			}
		}
	})
	t.Run("no_superseded_current_operator_claims", func(t *testing.T) {
		for _, path := range []string{op, "docs/handover/calibration.md", "docs/handover/deploy.md", "docs/handover/restore-drill.md", "docs/handover/member-guide.md", "docs/handover/README.md", "docs/ops/README.md"} {
			content := readOwnedDoc(t, root, path)
			for _, stale := range []string{"422 DEVICE_LIMIT_REACHED", "SECOPS-21 is fixed", "`SECOPS-21` fixed", "reserve of one", "bindings become empty", "updater does not consume that pending file", "HEALTH_CHECK_FAILED` with message `TC_FOREIGN_FILTER"} {
				if strings.Contains(content, stale) {
					t.Errorf("%s stale claim: %s", path, stale)
				}
			}
		}
	})
}

// claimTokenAllowed enforces D-109 for dated claim token assertions. Structural
// contract/schema checks and live document-integrity checks are separate tests.
func claimTokenAllowed(path, token string) bool {
	if path == "README.md" || path == "docs/ROADMAP.md" {
		return true
	}
	for _, prefix := range []string{"docs/handover/", "docs/ops/", "docs/reviews/", "tests/handover/testdata/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	if path == "docs/node/followup35-diagnostic-and-restart-2026-11-01.md" || strings.HasPrefix(path, "docs/e2e/runs/2026-11-01/followup27/") || strings.HasPrefix(path, "docs/poc/G2-followup25-evidence/") {
		return true
	}
	if strings.Contains(path, "/history/") {
		return true
	}
	return strings.HasSuffix(path, "_test.go") && regexp.MustCompile(`^(?:func )?Test[A-Za-z0-9_]+$`).MatchString(token)
}

func requireHandoverClaimTokens(t *testing.T, root, path string, tokens ...string) {
	t.Helper()
	content := readOwnedDoc(t, root, path)
	for _, token := range tokens {
		if !claimTokenAllowed(path, token) {
			t.Errorf("D-109 disallows token pin in %s: %q", path, token)
			continue
		}
		if strings.HasSuffix(path, "_test.go") {
			name := strings.TrimPrefix(token, "func ")
			if !regexp.MustCompile(`(?m)^func ` + regexp.QuoteMeta(name) + `\(`).MatchString(content) {
				t.Errorf("%s missing Go test declaration %s", path, name)
			}
		} else if !strings.Contains(content, token) {
			t.Errorf("%s missing handover claim %q", path, token)
		}
	}
}

func TestHandover_Followup15Claims(t *testing.T) {
	root := repoRoot(t)
	require := func(path string, tokens ...string) {
		t.Helper()
		requireHandoverClaimTokens(t, root, path, tokens...)
	}
	const fixture = "tests/handover/testdata/"
	const op = "docs/handover/operator-guide.md"
	const ops = "docs/ops/README.md"
	const register = fixture + "history/follow-up-15/register.md"
	const mainPin = fixture + "followup15-main.json"
	t.Run("merged_boundary_and_decisions", func(t *testing.T) {
		var pin struct {
			Commit string   `json:"commit"`
			Rows   []string `json:"rows"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, mainPin)), &pin); err != nil {
			t.Fatal(err)
		}
		if pin.Commit != "b79ee236821bbf8d2c47c43c1146b9be9006e26f" || len(pin.Rows) != 17 {
			t.Fatalf("wrong follow-up 15 cutoff: %+v", pin)
		}
		for _, pr := range []int{273, 274, 275, 276, 277, 279, 280, 281, 282, 283, 284, 285, 286, 287, 288, 289, 290} {
			require(mainPin, fmt.Sprintf("done (#%d)", pr))
			require(ops, fmt.Sprintf("#%d", pr))
		}
		for _, coming := range []string{"w4-at-security-ops follow-up 12", "w1-netlab follow-up 16", "w2-agent-core follow-up 25", "w4-at-isolation follow-up 8", "w3-wire-members follow-up 11", "w2-agent-core follow-up 24", "w4-at-members follow-up 8", "w3-devstack-e2e follow-up 17", "w2-fleet-update follow-up 11", "w3-wire-nodes follow-up 31"} {
			if strings.Contains(strings.Join(pin.Rows, "\n"), "| "+coming+" |") {
				t.Errorf("coming work included in delivered claims: %s", coming)
			}
		}
		text := readOwnedDoc(t, root, register)
		previous := -1
		for n := 98; n <= 111; n++ {
			pos := strings.Index(text, fmt.Sprintf("| D-%03d", n))
			if pos <= previous {
				t.Errorf("decision D-%03d missing or out of order", n)
			}
			previous = pos
		}
		require(register, "Follow-up 14 stopped at #272", "Current contract is revision 14", "Security-ops follow-up 12", "Netlab follow-up 15", "Enrollment follow-up 6", "Agent-core follow-up 24", "Members follow-up 8", "Devstack follow-up 17", "Wire-members follow-up 11", "Fleet-update follow-up 11; updater follow-up 15")
		require(register, "current incarnation may deliver the un-ACKed tail", "enrolled after retirement", "within one sampling interval", "retired certificate is never accepted", "Contract merged in revision 14 (#289)", "Clean serving evidence belongs on the observation", "tc_evidence", "job_id_mismatch", "uuid.NewSHA1", "one row per (incarnation, release)", "replay rules stay unchanged", "nodes.resource_version", "current_release_id = last_good_release_id = that release", "without promoting the candidate")
	})
	t.Run("D099_HTTP_regression_and_live_boundary", func(t *testing.T) {
		require("internal/nodes/httpwire/state_cache_integration_test.go", "TestAT30_D099_HTTPDesiredStateRestoreCache", "TestAT30_D099_HTTPDesiredStateGenerationFences")
		require("internal/nodes/httpwire/restore_cache_probe_integration_test.go", "TestAT30_D095_HTTPRestoreCacheProbe")
		require("internal/nodes/httpwire/followup29_receipts_integration_test.go", "TestD089_HTTPInstallerResultKeyedReplay")
		require(mainPin, "ordinary one-step proposal", "first offer 15", "304 / generation 11", "ACKs simulated")
		for _, path := range []string{op, "docs/handover/deploy.md", "docs/handover/restore-drill.md"} {
			require(path, "max(desired, applied, stored)+1", "no higher applied/stored generation", "no restore observation is outstanding", "Do not close SECOPS-21 from the compiler tests alone", "matching node-local accepted envelope/ETag", "separate real applied ACK", "SECOPS-21 remains open", "304 / generation 11", "first new offer `15`", "before starting either service")
		}
	})
	t.Run("portal_publication_and_request_ownership", func(t *testing.T) {
		require(mainPin, "Portal CI fix (red on main since #259)", "lease-renewal endpoint", "member_forwarding_unconfirmed (lease_unconfirmed)", "D-074 hold unchanged")
		require("internal/operations/failure_cause_test.go", "TestD107_PublicationGateMessageRejectsUnboundedInputs")
		require("internal/nodes/httpwire/publication_empty_integration_test.go", "TestAT08_PublicationWorkerEmptyBindingEvidence")
		require(op, "member_forwarding_unconfirmed (lease_unconfirmed)", "held publication stays `running`", "a polling timeout is not a failure", "2026-10-31-publication-gate-cause.md", "fixtures never fabricate lease evidence", "POST /api/v1/agent/lease-renewals")
		require(register, "written only by its owner", "docs/e2e/RESPONSES.md", "fvpn-w4-secops-*", "prepared but did not run live AT29 / AT31 / AT30")
	})
	t.Run("fleet_acceptance_and_database_Down", func(t *testing.T) {
		require("internal/fleetupdate/installer_results_integration_test.go", "TestD089_InstallerAcceptReplayAndIsolation", "TestD089_InstallerRejections", "TestD097_InstallerProjectionCentralPrecedenceAndIncarnation", "TestD089_InstallerDoesNotCompleteQueuedBatch")
		require("internal/db/migration_lint_test.go", "Test15_5_MigrationDownLint", "Test15_5_MigrationDownLintCases")
		require("internal/db/migration_roundtrip_integration_test.go", "Test15_5_MigrationsRoundTrip")
		for _, path := range []string{"docs/handover/00-index.md", "docs/handover/11-database-and-retention.md"} {
			require(path, "00191_installer_update_results.sql", "Down", "before the new control binary")
		}
		require("docs/handover/11-database-and-retention.md", "every migration above 00188", "db-lint", "Up → DownTo(188) → Up", "test-time")
		require(op, "409 ACK_CONFLICT", "without completing a central job or batch", "source: installer", "batch_id: null", "current central job wins", "eligible", "central answers `404`")
	})
	t.Run("netbird_path_closure_and_deadlines", func(t *testing.T) {
		require(mainPin, "10 clean recoveries", "median 54.4 ms / worst 78.6 ms", "10/10 deliberate reconnects", "zero redirected frames", "no D-100 override", "installer")
		require("node/agent/followup12_test.go", "TestD096_RealNetBird", "TestISO_F6_RetainedFailureBlocksNewGeneration")
		require("node/agent/followup23_path_test.go", "TestD100_TopologyRefreshRequiresCleanNewerPublishedApply", "TestD100_LegacyTopologyGenerationPinsBeforeBaselineACK")
		require("node/agent/followup23_freeze_integration_test.go", "TestD088_RealHTTPWireFreezeConsumer")
		require(op, "54.4 ms", "78.6 ms", "1.8–2.0 s", "zero\nredirected frames", "one-second harness hold", "Installer-owned transit and IPv6", "no agent route", "no D-100 override", "recovery latch after `TC_FOREIGN_FILTER`", "whole-second floor", "diagnostic lab exit 0 is not acceptance", "**AT13 stays failed**", "agent-core follow-ups 24/25 and isolation")
	})
	t.Run("D104_renderer_delivery_keeps_assembled_AT13_open", func(t *testing.T) {
		require(mainPin, "done (#290)", "1 ms set-element quantum", "100 ms render→installed-by budget", "D−100.707 ms", "D−0.837 ms", "assembled AT13 stays open")
		require("node/enforce/deadline_test.go", "TestD104_AT13_MillisecondDeadline", "TestD104_AllowedPrecisionRead", "TestD104_AT13_InstallOverrunNeverOpensGuard")
		require("tests/netlab/scenario_deadline_test.go", "TestD104_AT13_DeadlineContinuity")
		for _, path := range []string{op, ops} {
			require(path, "1 ms quantum", "100 ms guarded-install bound", "text read-back", "D−100.7 ms", "D−0.8 ms", "follow-ups 24/25", "follow-up 8", "physical arm64", "suspend")
		}
		require(op, "**AT13 stays failed**", "A stalled install keeps", "G2-deadline-contract-2026-10-31.md")
		require(ops, "**Assembled AT13 stays open**")
	})
	t.Run("revision14_contracts_and_pending_consumers", func(t *testing.T) {
		require(mainPin, "done (#289)", "API 1.10.0, 50 operations", "ACK per original stream", "late events rejected + audited")
		require("contracts/revision14_test.go", "TestAT29_ManualGrantWithdrawalContract", "TestMTR03_D105RetiredStreamVectors", "TestISO01_D106TCReadbackBoundsAndDigest")
		require("internal/httpapi/gen/revision14_test.go", "TestAT29_RevokeMemberGrantCompositionStub")
		require(register, "Current contract is revision 14", "OpenAPI `1.10.0`, `50` operations", "POST /admin/members/{member_id}/grants/{grant_id}/revoke", "admin session + CSRF + required `Idempotency-Key`", "existing `Operation`", "member.grant_revoke", "501 NOT_IMPLEMENTED", "wire-members follow-up 11", "optional, non-null `UsageBatchRequest.origin_incarnation_id`", "UsageBatchResult.node_incarnation_id", "late events are rejected and audited without advancing that ACK", "optional nullable `ObservationReport.tc_evidence`", "Usage/agent consumers and live ENR-05 acceptance remain pending", "D-110/D-111 remain coming")
		for _, vector := range []string{"member-grant-revoke", "usage-retired-handoff", "tc-readback-evidence"} {
			require(register, "contracts/vectors/"+vector+".json")
		}
	})
	t.Run("isolation7_is_a_new_dated_claim_set", func(t *testing.T) {
		const run = "041e9e48-79a6-45dd-a19d-8b5a63290655"
		for _, tc := range []struct {
			id, status          string
			pass, blocked, fail int
		}{{"AT10", "blocked", 7, 2, 0}, {"AT11", "blocked", 2, 1, 0}, {"AT12", "pass", 10, 0, 0}, {"AT13", "fail", 5, 1, 1}} {
			checkHandoverRecord(t, root, fixture+run+"-"+tc.id+".json", run, tc.status, tc.pass, tc.blocked, tc.fail)
			require(ops, fmt.Sprintf("%s `%d pass / %d blocked / %d fail`", tc.id, tc.pass, tc.blocked, tc.fail))
		}
		require(ops, run, "Assisted, not an unattended clean pass", "does not replace follow-up 5 or 6", "explicit pre-quarantine packet", "AT13 missed the one-second baseline tick")
		require(fixture+run+"-recovery-action.json", "assisted")
	})
	t.Run("enrollment5_and_installer3_keep_unresolved_acceptance", func(t *testing.T) {
		const run = "4011dd32-73ef-4f49-b78b-6cf1ab8496c9"
		base := "docs/e2e/at/enrollment/history/follow-up-5/"
		checkHandoverRecord(t, root, base+"AT06.json", run, "pass", 344, 0, 0)
		checkHandoverRecord(t, root, base+"AT07.json", run, "blocked", 19, 6, 0)
		require(ops, run, "AT06 344 pass / 0 blocked / 0 fail", "AT07 19 pass / 6 blocked / 0 fail", "ENR-05 same-slot exports", "enrollment follow-up 6 verifies the fix", "9 pass / 1 required fail")
		require("tests/installer/handoff_test.go", "TestD089_RealSeedThenJournaledUpgrade", "TestD089_MergedUpdaterSeedProbe")
		require("packaging/installer/reconcile_test.go", "TestD089_CheckAfterJournaledSwitch")
		require(op, "/run/fvpn-netbird/netbird.sock", "shared runtime survives consumer restarts", "offline seed", "firstboot", "required", "**red**", "state/meter.checkpoint", "CHECKPOINT_FAILED", "final_sample", "updater exits 0", "Agent-core follow-up 24")
	})
	t.Run("enrollment6_startup_fixed_five_AT07_rows_blocked", func(t *testing.T) {
		const run = "1af27741-cbd9-4fdb-aa1c-7d02790e7a46"
		const base = "docs/e2e/at/enrollment/history/follow-up-6/"
		checkHandoverRecord(t, root, base+"AT06.json", run, "pass", 376, 0, 0)
		checkHandoverRecord(t, root, base+"AT07.json", run, "blocked", 26, 5, 0)
		var record struct {
			Checks []struct{ Name, Status string } `json:"checks"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, base+"AT07.json")), &record); err != nil {
			t.Fatal(err)
		}
		var blocked []string
		for _, check := range record.Checks {
			if check.Status == "blocked" {
				blocked = append(blocked, check.Name)
			}
		}
		slices.Sort(blocked)
		if !slices.Equal(blocked, []string{"controlled_reinstall", "counted_unacked_usage", "historical_usage_retained", "identity_ledger_preserved", "installer_rerun"}) {
			t.Fatalf("AT07 remaining blockers changed: %v", blocked)
		}
		require(mainPin, "done (#288)", "installer-startup half of ENR-06 verified fixed on #279", "five rows blocked")
		require("tests/e2e/at/enrollment/installer_evidence_test.go", "TestAT07_InstallerEvidenceUsesDigestLabels")
		require(ops, run, "AT06 376 pass / 0 blocked / 0 fail", "AT07 26 pass / 5 blocked / 0 fail", "ENR-06 installer startup verified fixed on #279", "Five AT07 rows remain blocked", "same Node, fresh incarnation", "old certificate denied on both routes", "post-wrapper service restoration", "No full journaled-upgrade pass is claimed", "eight original files byte-identical")
	})
	t.Run("updates9_regular_pass_and_blocked_extension", func(t *testing.T) {
		const run = "89fc6769-0a09-403d-ae88-84f736ee1dce"
		for id := 23; id <= 27; id++ {
			checkHandoverRecord(t, root, fmt.Sprintf(fixture+run+"-AT%d.json", id), run, "pass", -1, 0, 0)
		}
		checkHandoverRecord(t, root, fixture+run+"-installer-handoff.json", run, "blocked", -1, 0, 0)
		var evidence struct {
			Observations []struct{ Name, Detail string } `json:"observations"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fixture+run+"-installer-handoff.json")), &evidence); err != nil {
			t.Fatal(err)
		}
		values := map[string]string{}
		for _, o := range evidence.Observations {
			values[o.Name] = o.Detail
		}
		for _, key := range []string{"check_seed", "check_fresh_no_work", "check_fresh_guard_clear", "check_boot_kernel", "check_legacy_boot_config"} {
			if values[key] != "true" {
				t.Errorf("D-080 %s=%s", key, values[key])
			}
		}
		for _, key := range []string{"check_apply", "check_central", "check_downgrade"} {
			if values[key] != "false" {
				t.Errorf("installer limitation %s=%s", key, values[key])
			}
		}
		require(fixture+run+"-requests.md", "F9-01", "F9-02", "F9-03", "F9-04", "F9-05", "5575470587a4e3014bd515e624ba2ad0aacbd140")
		require(ops, run, "AT23–AT27 all pass", "systemd lab node", "D-080 fresh-node", "kernel boot-identity", "Installer extension is **blocked**", "F9-01", "F9-02", "F9-03", "F9-04", "F9-05")
	})
	t.Run("devstack16_fixture_counts_and_limits", func(t *testing.T) {
		const run = "a81785fa-77eb-428a-b846-fb33196a2dcf"
		base := "docs/e2e/runs/2026-10-31/followup16/" + run + "/"
		files, err := filepath.Glob(filepath.Join(root, base, "*.json"))
		if err != nil || len(files) != 16 {
			t.Fatalf("devstack fixture set: %d, %v", len(files), err)
		}
		for _, file := range files {
			checkHandoverRecord(t, root, base+filepath.Base(file), run, "pass", -1, 0, 0)
		}
		require(ops, run, "16/16 acceptance fixtures pass", "finite-grant transition", "MTU 1500/1280", "RR/cache/rebinding", "retained pre-enrollment third node", "QUIC and non-Linux selectors explicitly report unavailable", "ENR-06 was not rerun")
	})
	t.Run("D098_archive_integrity_and_r16_dispositions", func(t *testing.T) {
		var pins []struct {
			Path         string `json:"path"`
			Source       string `json:"source"`
			SourceCommit string `json:"source_commit"`
			SHA256       string `json:"sha256"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fixture+"followup15-snapshots.json")), &pins); err != nil {
			t.Fatal(err)
		}
		counts := map[string]int{}
		for _, p := range pins {
			if len(p.SourceCommit) != 40 || p.Source == "" || len(p.SHA256) != 64 {
				t.Fatalf("invalid provenance %+v", p)
			}
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, p.Path)))
			if hex.EncodeToString(sum[:]) != p.SHA256 {
				t.Errorf("D-098 archived bytes changed: %s", p.Path)
			}
			for _, lane := range []string{"isolation", "secops", "enrollment"} {
				if strings.HasPrefix(p.Path, "docs/e2e/at/"+lane+"/history/follow-up-") {
					counts[lane]++
				}
			}
		}
		if counts["isolation"] != 44 || counts["secops"] != 104 || counts["enrollment"] != 12 || len(pins) != 197 {
			t.Fatalf("archive inventory changed: %v total=%d", counts, len(pins))
		}
		require("docs/reviews/r16-rev13-d089-d099-review.md", "main` at `5575470", "Reviewing model: `grok/grok-4.7`", "H-1", "M-1", "M-2", "L-1", "Known, reconfirmed, not filed")
		require(fixture+"history/follow-up-15/reviews.md", "e672d5d", "aa1e3fd95905d85975a43ecf304673d993d5fbbfc9817110635fc183c5f081fb", "wire-nodes follow-up 31 with the desired-state owner", "agent-core follow-up 24", "fleet-update follow-up 11", "D-110", "state_cache.go", "replay.go", "Fixed after the review base by #277", "Owners adopt")
		require("docs/reviews/r15-d072-d075-review.md", "r16 delivered on base `5575470`")
	})
	t.Run("D109_rejects_cross_lane_prose_pins", func(t *testing.T) {
		for _, tc := range []struct{ path, token string }{
			{"docs/policies/2026-10-31-secops-21-handover.md", "cachedStateSigner.Sign"},
			{"docs/poc/G2-deadline-contract-2026-10-31.md", "1 ms"},
			{"docs/usage/restore-runbook.md", "Step 6"},
			{"docs/node/install.md", "already_current"},
			{"internal/nodes/httpwire/state_cache.go", "cachedStateSigner"},
			{"internal/nodes/httpwire/state_cache_test.go", "some assertion message"},
			{"docs/e2e/at/isolation/AT10.json", "blocked"},
		} {
			if claimTokenAllowed(tc.path, tc.token) {
				t.Errorf("cross-lane prose pin accepted: %+v", tc)
			}
		}
	})
}

func checkHandoverRecord(t *testing.T, root, path, run, status string, pass, blocked, fail int) {
	t.Helper()
	var record struct {
		RunID  string `json:"run_id"`
		Status string `json:"status"`
		Checks []struct {
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), &record); err != nil {
		t.Fatal(err)
	}
	if record.RunID != run || record.Status != status {
		t.Fatalf("%s: run/status %s/%s, want %s/%s", path, record.RunID, record.Status, run, status)
	}
	if pass < 0 {
		return
	}
	counts := map[string]int{}
	for _, c := range record.Checks {
		counts[c.Status]++
	}
	if counts["pass"] != pass || counts["blocked"] != blocked || counts["fail"] != fail || len(record.Checks) != pass+blocked+fail {
		t.Fatalf("%s: check counts %v, want %d/%d/%d", path, counts, pass, blocked, fail)
	}
}

// TestHandover_Followup16Claims pins the #291–#303 cut. Owner measurements
// come only from final archives or run-ID files; implementation pins name
// existing Go proofs, not another lane's mutable prose (D-098 / D-109).
func TestHandover_Followup16Claims(t *testing.T) {
	root := repoRoot(t)
	const fixture = "tests/handover/testdata/"
	const mainPin = fixture + "followup16-main.json"
	const op = fixture + "history/follow-up-16/operator-guide.md"
	const ops = "docs/ops/README.md"
	const register = fixture + "history/follow-up-16/register.md"
	require := func(path string, tokens ...string) {
		t.Helper()
		requireHandoverClaimTokens(t, root, path, tokens...)
	}
	t.Run("merged_boundary_and_ordered_decisions", func(t *testing.T) {
		var pin struct {
			Commit string   `json:"commit"`
			Rows   []string `json:"rows"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, mainPin)), &pin); err != nil {
			t.Fatal(err)
		}
		if pin.Commit != "982c209e3f1fb6603138e0bc2a6ee4a5191b4e89" || len(pin.Rows) != 13 {
			t.Fatalf("wrong follow-up 16 cutoff: %+v", pin)
		}
		for pr := 291; pr <= 303; pr++ {
			require(mainPin, fmt.Sprintf("done (#%d)", pr))
			require(ops, fmt.Sprintf("#%d", pr))
		}
		for _, pending := range []string{"| w2-agent-core follow-up 24 |", "| w2-updater follow-up 15 |", "| w1-netlab follow-up 17 |"} {
			if strings.Contains(strings.Join(pin.Rows, "\n"), pending) {
				t.Errorf("pending delivery promoted: %s", pending)
			}
		}
		text := readOwnedDoc(t, root, register)
		previous := -1
		for n := 98; n <= 119; n++ {
			pos := strings.Index(text, fmt.Sprintf("| D-%03d", n))
			if pos <= previous {
				t.Errorf("decision D-%03d missing or out of order", n)
			}
			previous = pos
		}
		require(register, "OpenAPI `1.10.0`, `50` operations", "Follow-up 15 (#292)", "D-114 and D-119", "orchestrator's decisions", "Pending at the follow-up 16 cutoff")
		require(op, "hold applies only to those suites", "`git diff --check` needs no wait", "two or more other suites", "TMPDIR/GOTMPDIR", "without shell cleanup traps")
	})
	t.Run("r16_fence_and_installer_dispositions", func(t *testing.T) {
		require("internal/desiredstate/review_r16_fence_grant_integration_test.go", "TestReviewR16_FencedDeviceReceivesNoGrant")
		require("apps/worker/jobs_nodes_fences_integration_test.go", "TestD077_FenceRecompilesBoundExitsBeforeJobReceipt", "TestD077_FenceSerializesWithFirstBindingCompile")
		require("internal/fleetupdate/review_r16_installer_integration_test.go", "TestReviewR16_InstallerRollbackMovesRecordedVersion", "TestReviewR16_InstallerResultCardinalityIsBounded", "TestReviewR16_InstallerCannotSelfPromote")
		require("internal/fleetupdate/installer_followup11_integration_test.go", "TestD111_InstallerNullBaseline", "TestD111_InstallerNullBaselineRejectsUnprovenRelease", "TestD110_InstallerProjectionETagAndGenerationFence", "TestD110_InstallerMigrationDown190Up", "TestD110_InstallerLegacyReceiptStillReplays")
		require("internal/installerjob/id_test.go", "TestD110_StableManifestIdentity", "TestD110_MainUpdaterClaimNamespaceCompatibility")
		require(op, "an `authorization_unconfirmed` device is\nneither bound nor leased", "worker recompiles current signed envelopes", "r16 M-1 failed freeze", "pending agent-core follow-up 24", "Main's updater still mints the legacy whole-record ID", "job_id_mismatch", "D-111 seeds a null baseline", "Migration 00191's Down/Up round trip")
		require(fixture+"history/follow-up-16/reviews.md", "Fixed by wire-nodes follow-up 31", "(#291)", "Open → agent-core follow-up 24", "Fixed by fleet-update follow-up 11 (#300)")
		require(register, "one-file test-fixture ownership extension", "second r16 replay fixture", "production replay unchanged", "legacy unaccepted outbox receipts must be re-derived once or retired on 409", "Pending updater follow-up 15")
	})
	// This is the frozen #303 register, before #312 activation.
	// Followup18Claims checks the current active register and consuming proof.
	t.Run("historical_D105_before_D118_activation", func(t *testing.T) {
		require("internal/usage/handoff_integration_test.go", "TestMTR03_D105ContractVectorsAndExamples", "TestMTR03_D105IneligibleAndMixedStreams", "TestMTR03_D105HTTPSameKeyStoredReplayAndStreamACK", "TestENR05_D105RetiredCertificateRefused", "TestMTR03_D112HTTPConfiguredSamplingBound", "TestMTR03_D105PartialBatchAndIndependentACKs")
		require("internal/httpapi/usage_handoff_test.go", "TestMTR03_D115UsageGateDelegatesOnlyExplicitOrigin", "TestMTR03_D115UsageGateKeepsAuthenticationAndValidation")
		require("internal/httpapi/usage_handoff_integration_test.go", "TestMTR03_D115UsageForeignIdentityRejectedAndAudited")
		require("node/internal/usageupload/handoff_test.go", "TestD105_HandoffAcceptedThenCurrent", "TestD105_WrongStreamACKRejected", "TestD105_FinalOriginRejectionAndTransientRetry")
		require("node/internal/meter/production_test.go", "TestD112_ProductionSamplingInterval")
		require(register, "presenter enrolled strictly after retirement", "one origin per batch", "ACK per original stream", "retired certificate is refused at mTLS", "sampling_interval_bound_seconds", "default 10 s", "node sampling ≤ that bound", "only for `submitUsageBatch` with a schema-valid origin", "#299 is merged inactive", "activation hook in `node/agent/usage.go`", "usage AT follow-up 5")
		require(op, "A mixed batch is\nrejected before ledger/checkpoint mutation", "retired-tail ACK cannot advance the current cursor", "inactive pending\nagent-core follow-up 25 (D-118)")
	})
	t.Run("grant_registration_and_finite_deadline", func(t *testing.T) {
		require("internal/members/httpadapter/grant_revoke_integration_test.go", "TestAT29_D117_RevokeFarFutureFiniteGrant", "TestAT29_RevokeMemberGrantContractVectors", "TestAT29_RevokeFutureGrantAndHTTPGuards")
		require("apps/worker/jobs_members_grant_revoke_integration_test.go", "TestAT29_D077_GrantWithdrawalReconcilesFenceBeforeReceipt")
		require(register, "registers `revokeMemberGrant`", "`member.grant_revoked` joins #291's", "there are no open-ended grants", "NOT NULL with `isfinite` since migration 00001", "No scheduled plan-change contract exists", "a finite grant's deadline means expiry", "billing-issued replacement grant")
		require(op, "All 50 are wired", "fallback is retired", "Wire-members follow-up 11 (#297) implements this route")
		for _, path := range []string{register, op} {
			for _, stale := range []string{"49 are wired", "D-110/D-111 remain coming", "existing open-ended grants", "The generated handler currently returns `501"} {
				if strings.Contains(readOwnedDoc(t, root, path), stale) {
					t.Errorf("stale current claim in %s: %s", path, stale)
				}
			}
		}
	})
	t.Run("portal_audit_and_gate_cause", func(t *testing.T) {
		require(mainPin, "done (#293)", "D093-06", "D093-10", "13 entries open", "D093-12")
		require(op, "Portal follow-up 18 (#293)", "publication-gate cause beside", "held publication stays `running`", "15 entries, 13 open", "D093-06 / D093-10 are retired", "D093-12 remains open", "release_verifications.state=verified", "fixture transport decision is pending")
	})
	t.Run("installer_marker_and_agent_latch", func(t *testing.T) {
		const run = "a7c41436-82b3-4665-b75c-e2887f2e1009"
		base := "docs/e2e/runs/2026-10-31/followup17/" + run + "/"
		files, err := filepath.Glob(filepath.Join(root, base, "*.json"))
		if err != nil || len(files) != 12 {
			t.Fatalf("devstack scenario set: %d, %v", len(files), err)
		}
		counts := map[string]int{}
		for _, file := range files {
			var wire struct {
				RunID  string `json:"run_id"`
				Status string `json:"status"`
			}
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, base+filepath.Base(file))), &wire); err != nil {
				t.Fatal(err)
			}
			if wire.RunID != run {
				t.Fatal("wrong devstack run")
			}
			counts[wire.Status]++
		}
		if counts["pass"] != 8 || counts["blocked"] != 1 || counts["fail"] != 3 {
			t.Fatalf("devstack counts: %v", counts)
		}
		require("tests/installer/marker_test.go", "TestAT07_D116InstallerMarkerAndAgentRules")
		require("packaging/installer/marker_test.go", "TestAT07_D116MigrateOnRerunAndUpgrade", "TestPKG01_D116MarkerRefusesUnknownObjects")
		require("node/enforce/evidence_contract_test.go", "TestISO_F6_01_CleanNewGenerationRereadsTC", "TestD106_ContractVectorsAndObservationRoundTrip", "TestD106_ReadTimeCompleteOutcomeAndImmutableSnapshot")
		require("tests/netlab/scenario_tc_recovery_test.go", "TestISO_F6_01_ForeignRemovalNewGeneration")
		require("node/agent/followup12_test.go", "TestISO_F6_RetainedFailureBlocksNewGeneration")
		require(op, "hookless `inet fvpn_install`", "four rendered chains", "F9-01 is confirmed", "**10 pass / 1 fail**", "full journaled upgrade stays **red**", "Baseline-verifier proof", "does not claim serving", "netlab follow-up 17 scope tests", "devstack follow-up 18's serving fixture", "latch in `node/agent`, not the\nverifier", "`noteApply` retains `applyError`", "`needRenew`, `ackApplied`", "`renewAppliedLease`", "`leaseGenerationStale`", "`verifyCurrentServing`", "**AT13 stays failed**")
	})
	t.Run("enrollment7_freeze_and_five_blockers", func(t *testing.T) {
		const run = "59ef4bf8-7d4a-490f-a74d-6b54394375e0"
		const base = "docs/e2e/at/enrollment/history/follow-up-7/"
		checkHandoverRecord(t, root, base+"AT06.json", run, "pass", 424, 0, 0)
		checkHandoverRecord(t, root, base+"AT07.json", run, "blocked", 36, 5, 0)
		require(ops, "AT06 424 pass / 0 blocked / 0 fail", "AT07 36 pass / 5 blocked / 0 fail", "Five rows blocked on D-116 / D-105", "within **227 ms**", "survived a process restart", "zero un-ACKed bytes")
		require(base+"AT07.json", "identity_frozen", "227", "installer_rerun", "controlled_reinstall", "counted_unacked_usage", "identity_ledger_preserved", "historical_usage_retained")
	})
	t.Run("secops12_measured_restore_and_open_first_offer", func(t *testing.T) {
		const restore = fixture + "dd845a7f-2660-431c-a8e6-c513108cc07d-"
		var record struct {
			Status       string                                `json:"status"`
			Measurements map[string]map[string]json.RawMessage `json:"measurement_comparison"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, restore+"restore-reconcile.json")), &record); err != nil {
			t.Fatal(err)
		}
		if record.Status != "passed" {
			t.Fatal("AT30 restore no longer passed")
		}
		for key, want := range map[string]string{
			"Pre-replay database RPO (s)":                      "106.450816",
			"Post-replay service usage RPO (s)":                "0",
			"Protocol RTO (s)":                                 "66.903926483",
			"Total outage RTO (s)":                             "74.531204784",
			"Replay events / hashes / lost / duplicate / gaps": `"5 / 5 / 0 / 0 / 0"`,
		} {
			if got := string(record.Measurements["follow_up_12"][key]); got != want {
				t.Errorf("%s = %s, want %s", key, got, want)
			}
		}
		var first struct {
			Status   string `json:"status"`
			Request  string `json:"request_status"`
			Analysis struct {
				Guards map[string]bool `json:"closure_guards"`
			} `json:"first_offer_analysis"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, restore+"SECOPS-21.json")), &first); err != nil {
			t.Fatal(err)
		}
		if first.Status != "blocked" || first.Request != "open" {
			t.Fatal("SECOPS-21 was promoted")
		}
		for key, want := range map[string]bool{"d099_on_measured_base": true, "d095_hold_proved": true, "first_stored_row_above_retained": true, "first_delivered_http_response_proved": false, "first_offer_local_store_accepted": false, "first_stored_row_applied_ack": false} {
			got, ok := first.Analysis.Guards[key]
			if !ok || got != want {
				t.Errorf("closure guard %s = %v present=%v", key, got, ok)
			}
		}
		const transport = fixture + "cf81f6f8-4ccf-43d2-aae2-10121f6b169d-"
		for _, id := range []string{"AT29", "AT31"} {
			var r struct {
				Status string `json:"status"`
				Actual map[string]struct {
					Status string `json:"status"`
				} `json:"actual"`
			}
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, transport+id+".json")), &r); err != nil {
				t.Fatal(err)
			}
			if r.Status != "blocked" {
				t.Fatalf("full %s promoted", id)
			}
			want := map[string]string{"forced_relay": "passed"}
			if id == "AT29" {
				want = map[string]string{"grant_expiry": "passed", "lease_expiry": "passed", "device_revoke": "passed", "manual_grant_withdrawal": "blocked", "guest_clock_rollback": "blocked"}
			}
			for key, status := range want {
				if r.Actual[key].Status != status {
					t.Errorf("%s %s = %s", id, key, r.Actual[key].Status)
				}
			}
		}
		require(ops, "**AT30 passed**", "**106.45 s**", "**74.53 s**", "**5 / 5 / 0 / 0 / 0**", "**SECOPS-21 stays open**", "first-offer delivery / local acceptance / ACK unproven", "devstack 18 / wire-nodes 32", "Full AT29 / AT31 blocked", "candidate hardware")
	})
	t.Run("members8_passes_and_AT09_failure", func(t *testing.T) {
		const run = "69f92f49-73dc-4779-8101-dfb6a802a96b"
		for _, tc := range []struct {
			id                  int
			status              string
			pass, blocked, fail int
		}{
			{1, "blocked", 18, 0, 0}, {2, "pass", 122, 0, 0}, {3, "pass", 92, 0, 0}, {4, "blocked", 99, 0, 0}, {5, "blocked", 30, 0, 0}, {8, "pass", 31, 0, 0}, {9, "fail", 34, 5, 4}, {20, "blocked", 81, 0, 0}, {21, "blocked", 12, 0, 0}, {22, "blocked", 39, 0, 0},
		} {
			var r struct {
				RunID  string `json:"run_id"`
				Status string `json:"status"`
				Steps  []struct {
					Status string `json:"status"`
				} `json:"steps"`
			}
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fmt.Sprintf(fixture+run+"-AT%02d.json", tc.id))), &r); err != nil {
				t.Fatal(err)
			}
			counts := map[string]int{}
			for _, s := range r.Steps {
				counts[s.Status]++
			}
			if r.RunID != run || r.Status != tc.status || counts["pass"] != tc.pass || counts["blocked"] != tc.blocked || counts["fail"] != tc.fail {
				t.Errorf("AT%02d status=%s counts=%v", tc.id, r.Status, counts)
			}
		}
		require(fixture+run+"-summary.md", "3 complete cases passed; 6 blocked; 1 failed", "558 pass / 5 blocked / 4 fail", "Genuine deadline expiry")
		require(ops, "`ReplaceFiniteGrant`", "**3/0 → 0/3 → 3/0**", "**AT09 fails**", "F8-01 → agent-core 25", "F8-02 → devstack 18 / netlab 18", "11,011 of 64,064 bytes", "final 31-file archive; local run-ID pins retired")
	})
	t.Run("D098_archives_and_D109_prose_boundaries", func(t *testing.T) {
		var pins []struct {
			Path   string `json:"path"`
			Source string `json:"source"`
			Commit string `json:"source_commit"`
			SHA256 string `json:"sha256"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fixture+"followup16-snapshots.json")), &pins); err != nil {
			t.Fatal(err)
		}
		counts := map[string]int{}
		for _, p := range pins {
			if p.Source == "" || len(p.Commit) != 40 || len(p.SHA256) != 64 {
				t.Fatalf("invalid provenance: %+v", p)
			}
			if !strings.Contains(p.Path, "/history/") && !regexp.MustCompile(`[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}`).MatchString(p.Path) {
				t.Fatalf("D-098 mutable evidence path %s", p.Path)
			}
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, p.Path)))
			if hex.EncodeToString(sum[:]) != p.SHA256 {
				t.Errorf("archived bytes changed: %s", p.Path)
			}
			for lane, n := range map[string]int{"members": 7, "enrollment": 6, "secops": 11} {
				if strings.HasPrefix(p.Path, fmt.Sprintf("docs/e2e/at/%s/history/follow-up-%d/", lane, n)) {
					counts[lane]++
				}
			}
		}
		if counts["members"] != 31 || counts["enrollment"] != 8 || counts["secops"] != 105 || len(pins) != 185 {
			t.Fatalf("archive inventory %v total=%d", counts, len(pins))
		}
		for _, p := range []string{"docs/portal/DEV.md", "docs/usage/2026-10-31-d105-retired-stream-handoff.md", "docs/members/README.md", "docs/poc/G2-followup14.md", "docs/e2e/at/members/AT09.json", "docs/e2e/at/secops/AT30.json"} {
			if claimTokenAllowed(p, "some prose") {
				t.Errorf("D-109 allows living prose %s", p)
			}
		}
	})
}

// TestHandover_Followup17Claims fixes the #304–#313 evidence cut. D-098
// measurements come from frozen history or run-ID paths. D-109 implementation
// pins name Go tests; no owner's living prose is a claim-token dependency.
func TestHandover_Followup17Claims(t *testing.T) {
	root := repoRoot(t)
	const fixture = "tests/handover/testdata/"
	const archive = fixture + "history/follow-up-17/"
	const mainPin = fixture + "followup17-main.json"
	const cut = "7e044c71ecf1d07f63e89721bd213db6611a6a47"
	const doc = "docs/handover/follow-up-17.md"
	const register = fixture + "history/follow-up-18/register17.md"
	const runs = "docs/e2e/runs/2026-10-31/followup18/"
	require := func(path string, tokens ...string) {
		t.Helper()
		requireHandoverClaimTokens(t, root, path, tokens...)
	}
	t.Run("merged_cut_and_process_decisions", func(t *testing.T) {
		var pin struct {
			Commit string   `json:"commit"`
			Rows   []string `json:"rows"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, mainPin)), &pin); err != nil {
			t.Fatal(err)
		}
		if pin.Commit != cut || len(pin.Rows) != 10 {
			t.Fatalf("wrong follow-up 17 cut: %+v", pin)
		}
		for pr := 304; pr <= 313; pr++ {
			require(mainPin, fmt.Sprintf("done (#%d)", pr))
			require("docs/ops/README.md", fmt.Sprintf("#%d", pr))
		}
		require(mainPin, "w4-handover follow-up 16", "through #303", "D-112–D-119", "#304 (netlab 17, D-120)")
		require(doc, cut, "Handover follow-up 16 (#306) stopped at #303", "Handover ran no new live acceptance")
		text := readOwnedDoc(t, root, register)
		previous := -1
		for n := 98; n <= 123; n++ {
			pos := strings.Index(text, fmt.Sprintf("| D-%03d", n))
			if pos <= previous {
				t.Errorf("decision D-%03d missing or out of order", n)
			}
			previous = pos
		}
		for _, n := range []int{121, 122} {
			start := strings.Index(text, fmt.Sprintf("| D-%d |", n))
			if start < 0 {
				t.Fatalf("missing D-%d", n)
			}
			line := strings.SplitN(text[start:], "\n", 2)[0]
			if !strings.Contains(line, "Orchestration process only") || !strings.Contains(strings.ToLower(line), "no operator guarantee") {
				t.Errorf("D-%d is not process-only: %s", n, line)
			}
			if strings.Contains(readOwnedDoc(t, root, doc), fmt.Sprintf("D-%d", n)) {
				t.Errorf("process decision D-%d escaped into operator evidence", n)
			}
		}
		require(register, "one-follow-up ownership extension", "packaging/installer/handoff_test.go", "TestD089_RealUpdaterConsumesWriterRecord", "node/enforce/review_r16_quarantine_test.go", "test-only ownership exception")
		require("packaging/installer/handoff_test.go", "TestD089_RealUpdaterConsumesWriterRecord")
		require("node/enforce/review_r16_quarantine_test.go", "TestReviewR16_QuarantineNeverLowersUplink")
	})
	t.Run("post_cutoff_secops_notice", func(t *testing.T) {
		for _, path := range []string{doc, register} {
			require(path, "2026-10-31 08:17 EDT resume", "security-ops follow-up 13 (#316) landed", "after this reconciliation through #313", "SECOPS-21 closure and AT30", "reserved for reconciliation in handover follow-up 18", "original #313 cutoff")
		}
		require("docs/ROADMAP.md", "| w4-at-security-ops follow-up 13 |", "done (#316)")
	})
	t.Run("D116_and_pre_D120_failure", func(t *testing.T) {
		require("node/enforce/verify_installer_test.go", "TestD116_InstallerMarkerScope")
		require("tests/netlab/agent/scenario_d116_test.go", "TestD116_AT10_InstallerMarkerEnrollment")
		require(archive+"installer-marker.md", "identical error text and code", "ISOLATION_POLICY_INVALID", "HOOK_PRIORITY_MISMATCH", "before enrollment", "nft 1.0.9", "Use text read-back", "Generation 1, forwarding enabled")
		require(archive+"reload-stream.md", "failed under the old rule", "0 / 2 / 0 / 0", "0 / 1 / 0 / 0", "54.861", "18.174", "29.520", "Three bracket overlaps, zero outside-bracket timeouts", "zero packets proven to arrive in the narrow definitely-guarded interval")
		require(doc, "broken-owned-ruleset controls", "hooked **drop** chain", "HOOK_PRIORITY_MISMATCH", "text read-back", "three UDP timeouts over two 100-reload runs", "18–55 ms", "29.520 ms before rounding", "AT10 failed under the pre-D-120 rule")
	})
	t.Run("agent24_recovery_and_installer_boundary", func(t *testing.T) {
		require(mainPin, "897 ms", "494 ms", "3.161 s", "applied and lease generations 3", "durable FileMeter checkpoint", "D-087 provisional transition", "r16 M-1 durable freeze", "D-104 continuity deferred")
		require("node/agent/followup24_test.go", "TestISO_F6_ForeignStillPresentWithholdsNewACK", "TestISO_F6_ObservationEvidenceRevision13", "TestISO_F6_ApplyRejectionTimestampPrecedesRollback", "TestMTR03_UPD05_DurableFileMeterBarrier", "TestUPD05_CandidateIdentityAndHealth", "TestD087_ObservationRetryAfterCoalescesAndRetainsTransition", "TestD088_FailedFreezeAcceptanceStaysReplayable", "TestD088_LostFreezeResponseBlocksRestartUntilReplay")
		require("node/agent/review_r16_freeze_test.go", "TestReviewR16_FailedFreezePersistIsLostOnRestart")
		require(archive+"agent24-updater.md", "atomic rename, directory fsync", "invalidates stale evidence", "actual running", "HTTP 404", "meter_checkpoint_present=true", "health both fail")
		require(doc, "retained tc", "failure blocked ACK / renewal", "deny-all candidate", "without reusing old grants", "same- and newer-generation retries", "**897 ms**", "**494 ms**", "**3.161 s**", "bounded", "`qos` ReportDetail", "durable FileMeter checkpoint", "running executable/current pointer/journal", "D-087", "r16 M-1", "w5-control fixture returned 404", "D-104 continuity was deferred", "13/13")
		require("docs/reviews/README.md", "Fixed by agent-core follow-up 24 (#305)", "lost-response replay guard", "updater follow-up 15 (#307)")
	})
	t.Run("updater15_and_subsequent_limits", func(t *testing.T) {
		require(mainPin, "D-110", "central catalog", "job_id_mismatch", "terminal diagnostic row", "directories 0755", "zero → 0644", "exit 2 only", "updater-integration", "F9-05", "done (#310)", "uncorrelated", "done (#311)", "13/13")
		require("node/updater/installer_result_test.go", "TestD110_PendingIntegrityIsSeparateFromJobIdentity", "TestD114_RebindBeforeUploadAndReplayExactSeal", "TestD114_AcceptedLegacyHistoryNeverReminted", "TestD114_UnrecreatableSealMismatchIsTerminal", "TestF903_ApplySealedCheckpointFailure")
		require("node/updater/cmd/fvpn-updater/followup15_test.go", "TestF902_ExtractAndSeedAcrossUmasks", "TestF903_ApplyCLIReportsSealedCheckpointFailure")
		require("node/updater/installer_interop_integration_test.go", "TestD110_RealInstallerReceiptInteroperability")
		require("node/updater/meter_barrier_test.go", "TestAT25_FileMeterWaitFailsClosedWithoutCorrelation", "TestAT25_FileMeterWaitRestartRechecksCurrentFile")
		require("packaging/installer/upgrade_test.go", "TestF905_DowngradeRefusedBeforeStaging")
		require("tests/installer/handoff_test.go", "TestD089_RealSeedThenJournaledUpgrade")
		require(doc, "`internal/installerjob` ID", "`job_id_mismatch` 409", "Accepted legacy receipts retain stored replay semantics", "unknown seal", "directories 0755", "zero mode → 0644", "exit **1**", "exit **2**", "isolated-database proofs", "no new live updates acceptance", "physical", "arm64", "`packaging/installer/upgrade.go`", "bounded FileGate → FileMeter wait", "barrier_id", "no service switch")
	})
	t.Run("D120_verdict_records_and_fail_closed_inputs", func(t *testing.T) {
		for _, tc := range []struct {
			name            string
			datagrams, loss int
		}{{"lane", 48, 1}, {"focused", 52, 0}} {
			var r struct {
				Summary map[string]json.RawMessage `json:"summary"`
				D120    struct {
					Passed  bool     `json:"passed"`
					Verdict string   `json:"verdict"`
					Reloads int      `json:"reloads"`
					Count   int      `json:"udp_losses"`
					Errors  []string `json:"errors"`
					Losses  []struct {
						Sequence       int      `json:"sequence"`
						Generation     string   `json:"generation"`
						Upper          int64    `json:"upper_ns"`
						Installed      int64    `json:"render_to_installed_ns"`
						Classification string   `json:"classification"`
						Next           string   `json:"next_datagram"`
						Failures       []string `json:"failures"`
					} `json:"losses"`
				} `json:"d120"`
			}
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, archive+"at10-"+tc.name+".json")), &r); err != nil {
				t.Fatal(err)
			}
			if !r.D120.Passed || r.D120.Verdict != "AT10 PASS under D-120" || r.D120.Reloads != 100 || r.D120.Count != tc.loss || len(r.D120.Losses) != tc.loss || len(r.D120.Errors) != 0 {
				t.Fatalf("%s: wrong D-120 verdict %+v", tc.name, r.D120)
			}
			for k, want := range map[string]int{"apply_count": 101, "verified_applies": 101, "datagrams": tc.datagrams, "udp_timeouts": tc.loss, "tcp_timeouts": 0, "guard_upper_over_100ms": 0, "install_over_100ms": 0, "unobserved_node_ingress": 0, "incomplete_udp_reads": 0, "mismatched_echoes": 0} {
				if string(r.Summary[k]) != strconv.Itoa(want) {
					t.Errorf("%s %s = %s, want %d", tc.name, k, r.Summary[k], want)
				}
			}
			for _, loss := range r.D120.Losses {
				if loss.Sequence != 19 || loss.Generation != "1039" || loss.Upper != 25528070 || loss.Installed != 13781637 || loss.Classification != "boundary" || loss.Next != "ok" || len(loss.Failures) != 0 {
					t.Fatalf("loss evidence changed: %+v", loss)
				}
			}
		}
		require(archive+"reload-stream.md", "exactly one verified Apply", "2 µs timestamp margin", "next datagram succeeds", "zero TCP timeouts", "at most 5 UDP losses", "zero kernel capture drops", "two matching", "terminal loss", "not a product guarantee", "D-104's 100 ms installation")
		require(doc, "test-acceptance definition, not a product guarantee", "exactly one verified", "≤ 100 ms", "≤ 5 UDP losses per 100 reloads", "Missing sidecar / trace / Apply", "capture drops", "successful", "AT10 PASS under D-120", "48 datagrams", "25.528070 ms", "zero losses", "D-104's 100 ms installation bound is\nunchanged")
	})
	t.Run("D123_overlay_vs_native_underlay", func(t *testing.T) {
		require(archive+"mtu1280.md", "64,064/64,064", "fragmentation-needed/1280", "Quoted IPv4 error toward member", "Quoted IPv4 error toward origin", "ctv6", "accepts only `IPv6Block`", "11,011/64,064", "Do not remove these rules")
		require(doc, "IPv4 transfers", "at MTU 1280", "in both directions", "`node/enforce/render_nft.go`", "`node/enforce/validate.go`", "D-123: the v1.0 overlay is IPv6Block by design", "no IPv6\nforwarding through the node", "no end-to-end IPv6 MTU / PMTU acceptance row", "never deletion of", "11,011-byte", "underlay")
		require(register, "Native IPv6 underlay failures remain applicable")
	})
	t.Run("dns_response_shapes_and_TTL", func(t *testing.T) {
		require(archive+"dns-backend.md", "### 2.1.1", "### 2.1.2", "### 2.1.3", "### 2.1.4", "432 raw UDP/TCP probe exchanges", "F8-A", "F8-AAAA", "F8-HTTPS", "F8-CNAME", "F8-SVCB", "F8-TXT", "F8-MX", "F8-NS", "0 NOERROR (NODATA)", "present / 60", "SOA TTL from 60 to **5**", "fail-closed for answers", "not a promise of uniform NXDOMAIN")
		require("tests/netlab/dns/response_shape_test.go", "TestAT20_ResponseShape")
		require(mainPin, "CI conformance probe", "done (#309)")
		require(doc, "A / AAAA / HTTPS → NXDOMAIN", "CNAME / SVCB / TXT / MX / NS →", "NODATA", "RCODE 0, zero answers, SOA in authority", "blocked_response_ttl", "**60 s**", "**5 s**", "production choice remains pending", "fail-closed for answers, not uniform NXDOMAIN", "432 exchanges")
		require("docs/handover/member-guide.md", "A/AAAA/HTTPS", "CNAME/SVCB/TXT/MX/NS", "RCODE 0, zero answers, SOA authority", "不保證一律 NXDOMAIN", "D-123")
	})
	t.Run("agent25_activation_and_measurement_limits", func(t *testing.T) {
		require("node/agent/followup25_usage_test.go", "TestD118_ENR05_VerifiedReenrollmentHandsOffFinalSample", "TestD118_UnchangedIncarnationDoesNotRetire")
		require("node/agent/followup25_evidence_test.go", "TestD106_ContractExamplesRoundTrip", "TestD106_ProducerRetainsAuthoritativeSnapshot", "TestISO_F6_RetainedErrorRenewsDesiredWithoutACK")
		require(archive+"agent25.md", "MemberLease.DeadlineMono", "Model.NowMono", "1.419 s", "74,796 / 76,000", "98.4158%", "75 guarded Applies", "D − 100.651423 ms", "D − 0.719753 ms", "RetirePreviousAt: e.clock.Wall()", "origin_incarnation_id", "not replace central D-112", "physical arm64")
		require(mainPin, "ObserveSelectedNativeExit", "4,096-byte", "D-118 active", "F8-01", "no generation bump or fabricated ACK")
		require(doc, "D-118 is active", "RetirePreviousAt: e.clock.Wall()", "**4,096-byte**", "independent ACKs", "sequence 1", "1.419 s", "74,796/76,000 probes (98.4158%)", "TEST ONLY 1 s Apply cadence", "D-104 assembled continuity remains open", "null before any retained read", "historical rejection rows")
	})
	t.Run("devstack18_preserves_three_failures_and_scope", func(t *testing.T) {
		for _, tc := range []struct {
			run         string
			pass, fail  int
			failedCases string
		}{
			{"09cb1198-435f-4609-9d0b-d53e6c1ed441", 13, 1, "installer-journaled-upgrade"},
			{"8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179", 13, 1, "installer-reenroll"},
			{"daf790fc-81f7-49a2-8a65-f3ec8faec656", 9, 2, "first-desired-offer,native-mtu-comparison"},
			{"5f74d7c3-6404-461d-9f47-296bd69958c2", 2, 0, ""},
		} {
			files, err := filepath.Glob(filepath.Join(root, runs, tc.run, "*.json"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(tc.run, "daf790fc") {
				files = append(files, filepath.Join(root, runs, tc.run, "node2/publish.json"))
			}
			counts := map[string]int{}
			var failed []string
			for _, file := range files {
				var r struct {
					RunID  string `json:"run_id"`
					ID     string `json:"id"`
					Status string `json:"status"`
				}
				b, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(b, &r); err != nil {
					t.Fatal(err)
				}
				if r.RunID != tc.run {
					t.Fatalf("run mismatch in %s", file)
				}
				counts[r.Status]++
				if r.Status == "fail" {
					failed = append(failed, r.ID)
				}
			}
			if counts["pass"] != tc.pass || counts["fail"] != tc.fail || len(files) != tc.pass+tc.fail || strings.Join(failed, ",") != tc.failedCases {
				t.Fatalf("%s counts=%v failures=%v", tc.run, counts, failed)
			}
		}
		// Assert the actual sealed failure, not merely a FAIL label or process exit.
		var record struct {
			Observations []struct{ Name, Detail string } `json:"observations"`
		}
		readObservations := func(path string) map[string]string {
			t.Helper()
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), &record); err != nil {
				t.Fatal(err)
			}
			out := map[string]string{}
			for _, o := range record.Observations {
				out[o.Name] = o.Detail
			}
			return out
		}
		upgrade := readObservations(runs + "09cb1198-435f-4609-9d0b-d53e6c1ed441/installer-journaled-upgrade.json")
		var seal struct {
			Stage   string `json:"stage"`
			State   string `json:"state"`
			Journal string `json:"journal_stage"`
		}
		if err := json.Unmarshal([]byte(upgrade["durable_result"]), &seal); err != nil {
			t.Fatal(err)
		}
		if upgrade["apply_process_success"] != "true" || upgrade["genuine_meter_checkpoint_present"] != "false" || upgrade["current_before"] == "" || upgrade["current_before"] != upgrade["current_after"] || seal.Stage != "final_sample" || seal.State != "failed" || seal.Journal != "rolled_back" {
			t.Fatalf("serving failure was lost: %+v", seal)
		}
		first := readObservations(runs + "5f74d7c3-6404-461d-9f47-296bd69958c2/first-desired-offer.json")
		var capture struct {
			First struct {
				Generation string `json:"generation"`
				Digest     string `json:"body_sha256"`
				Status     int    `json:"http_status"`
				Complete   bool   `json:"write_complete"`
			} `json:"first"`
			Local struct {
				Accepted string `json:"accepted_generation"`
				Applied  string `json:"applied_generation"`
				Digest   string `json:"envelope_sha256"`
			} `json:"local"`
			ACK struct {
				Report  string `json:"report_id"`
				Applied string `json:"applied_generation"`
				Status  int    `json:"http_status"`
			} `json:"ack"`
		}
		if err := json.Unmarshal([]byte(first["capture"]), &capture); err != nil {
			t.Fatal(err)
		}
		var receipt struct {
			Report  string `json:"report_id"`
			Applied string `json:"applied_generation"`
		}
		if err := json.Unmarshal([]byte(first["central_receipt"]), &receipt); err != nil {
			t.Fatal(err)
		}
		if first["retained_generation"] != "3" || capture.First.Generation != "4" || capture.First.Status != 200 || !capture.First.Complete || len(capture.First.Digest) != 64 || capture.First.Digest != capture.Local.Digest || capture.Local.Accepted != "4" || capture.Local.Applied != "4" || capture.ACK.Applied != "4" || capture.ACK.Status != 202 || capture.ACK.Report == "" || capture.ACK.Report != receipt.Report || receipt.Applied != "4" {
			t.Fatalf("first-delivery/local-acceptance/ACK proof changed: %+v", capture)
		}
		require(archive+"devstack18.md", "ENR-05 passes on the real installer node", "1,376 upload / 1,001 download", "sequence/ACK 3", "does not establish D-105", "**`HASH_MISMATCH", "Both families have", "not explained by the matrix's reconnect step alone", "does not", "close the full SECOPS-21", "5.96 seconds", "generation 30")
		require(archive+"responses.md", "HoldInstallerFirstPoll", "retired_streams", "new stream's ACK is not the retired ACK", "activation hook", "compiled TEST ONLY fixture fetcher", "This is a view, not an implemented product", "ArmFirstDesiredOffer", "ReleaseDesiredOffer", "FirstDesiredOffer", "RestartNodeNamespace90")
		require(doc, "ENR-05 passed in this harness", "final_sample / failed / rolled_back", "baseline_failed", "HASH_MISMATCH", "**2/0/0 passed**", "4 > retained 3", "**202** receipt", "SECOPS-21 remains open", "secops 13", "installer 6 / updater 17", "netlab 19 / wire-nodes 32", "both selected underlay families", "not yet proved", "generation-30/full-history", "journal-copy", "current stream's CLI ACK is not the retired ACK", "not implemented", "13 open / 2 retired")
	})
	t.Run("unpromoted_acceptance_and_snapshot_integrity", func(t *testing.T) {
		checkHandoverRecord(t, root, "docs/e2e/at/enrollment/history/follow-up-7/AT07.json", "59ef4bf8-7d4a-490f-a74d-6b54394375e0", "blocked", 36, 5, 0)
		for _, path := range []string{fixture + "cf81f6f8-4ccf-43d2-aae2-10121f6b169d-AT29.json", fixture + "cf81f6f8-4ccf-43d2-aae2-10121f6b169d-AT31.json", fixture + "69f92f49-73dc-4779-8101-dfb6a802a96b-AT09.json"} {
			var r struct{ Status string }
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), &r); err != nil {
				t.Fatal(err)
			}
			if r.Status != "blocked" && r.Status != "fail" {
				t.Errorf("acceptance promoted: %s = %s", path, r.Status)
			}
		}
		require(doc, "five AT07 rows", "AT09 F8-01 / F8-02", "D-104 assembled continuity / AT13", "AT31 candidate hardware", "AT29 guest-clock", "real Windows, macOS, iOS and Android clients", "physical arm64")
		var pins []struct {
			Path   string `json:"path"`
			Source string `json:"source"`
			Commit string `json:"source_commit"`
			SHA256 string `json:"sha256"`
		}
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fixture+"followup17-snapshots.json")), &pins); err != nil {
			t.Fatal(err)
		}
		if len(pins) != 59 {
			t.Fatalf("snapshot inventory = %d, want 59", len(pins))
		}
		seen := map[string]bool{}
		for _, p := range pins {
			if p.Commit != cut || p.Source == "" || len(p.SHA256) != 64 || seen[p.Path] {
				t.Fatalf("bad provenance or duplicate: %+v", p)
			}
			seen[p.Path] = true
			if !strings.Contains(p.Path, "/history/") && !regexp.MustCompile(`[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}`).MatchString(p.Path) {
				t.Fatalf("D-098 mutable evidence: %s", p.Path)
			}
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, p.Path)))
			if hex.EncodeToString(sum[:]) != p.SHA256 {
				t.Errorf("pinned bytes changed: %s", p.Path)
			}
		}
		for _, path := range []string{"docs/node/2026-10-31-d104-d106-agent.md", "docs/e2e/RESPONSES.md", "docs/dns/BACKEND.md", "docs/e2e/at/enrollment/AT07.json"} {
			if claimTokenAllowed(path, "some prose") {
				t.Errorf("D-109 allows living prose: %s", path)
			}
		}
	})
}

// TestHandover_Followup18Claims fixes the starting main cutoff, including #323.
// D-098 evidence is frozen; D-109 runtime pins are Go test declarations only.
func TestHandover_Followup18Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "6d0022d83345ae78d26aaaf31c64f026d38dea3b"
	const fixture = "tests/handover/testdata/"
	const archive = fixture + "history/follow-up-18/"
	const mainPin = fixture + "followup18-main.json"
	const doc = "docs/handover/follow-up-18.md"
	// The original register is frozen; Followup19Claims checks current dispositions.
	const register = fixture + "history/follow-up-19/register18.md"
	const restore = fixture + "5218b7d5-1415-4e1b-8618-48ea2d3cf03e-"
	require := func(path string, tokens ...string) {
		t.Helper()
		requireHandoverClaimTokens(t, root, path, tokens...)
	}
	readJSON := func(path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	// Navigate objects without turning decimal strings or integers into float64.
	value := func(path string, keys ...string) json.RawMessage {
		t.Helper()
		raw := json.RawMessage(readOwnedDoc(t, root, path))
		for _, key := range keys {
			var object map[string]json.RawMessage
			if err := json.Unmarshal(raw, &object); err != nil || object[key] == nil {
				t.Fatalf("%s: missing object key %s (%v)", path, key, err)
			}
			raw = object[key]
		}
		return raw
	}
	wantValue := func(path, want string, keys ...string) {
		t.Helper()
		if got := string(value(path, keys...)); got != want {
			t.Errorf("%s %v = %s, want %s", path, keys, got, want)
		}
	}
	t.Run("main_cut_and_operator_entrypoints", func(t *testing.T) {
		var pin struct {
			Commit string
			Rows   []string
		}
		readJSON(mainPin, &pin)
		if pin.Commit != cut || len(pin.Rows) != 12 {
			t.Fatalf("wrong cutoff: %+v", pin)
		}
		for _, row := range pin.Rows {
			require("docs/ROADMAP.md", row)
		}
		for pr := 316; pr <= 324; pr++ {
			require(mainPin, fmt.Sprintf("done (#%d)", pr))
			require("docs/ops/README.md", fmt.Sprintf("#%d", pr))
		}
		require(doc, cut, "including updater 17 (#323)", "Handover ran no new live acceptance")
		// Current entrypoints move with the latest cut; this cut's evidence stays frozen.
		require(register, "Follow-up 18 decisions register", "through #324")
	})
	t.Run("SECOPS21_first_delivered_restore_and_AT30", func(t *testing.T) {
		p := restore + "SECOPS-21.json"
		wantValue(p, `"closed"`, "request_status")
		wantValue(p, `"passed"`, "status")
		var guards map[string]bool
		if err := json.Unmarshal(value(p, "closure_guards"), &guards); err != nil || len(guards) != 10 {
			t.Fatalf("closure guards: %v %v", guards, err)
		}
		for name, passed := range guards {
			if !passed {
				t.Errorf("closure guard failed: %s", name)
			}
		}
		wantValue(p, "1", "first_delivered_offer", "arm_count")
		wantValue(p, `"15"`, "first_delivered_offer", "retained_before_restore", "node_accepted_desired")
		wantValue(p, `"19"`, "first_delivered_offer", "capture", "first", "generation")
		wantValue(p, "200", "first_delivered_offer", "capture", "first", "http_status")
		wantValue(p, "true", "first_delivered_offer", "capture", "first", "write_complete")
		wantValue(p, `"19"`, "first_delivered_offer", "capture", "local", "accepted_generation")
		wantValue(p, `"19"`, "first_delivered_offer", "capture", "local", "applied_generation")
		wantValue(p, "202", "first_delivered_offer", "capture", "ack", "http_status")
		wantValue(p, `"19"`, "first_delivered_offer", "central_receipt", "applied_generation")
		if string(value(p, "first_delivered_offer", "capture", "first", "body_sha256")) != string(value(p, "first_delivered_offer", "capture", "local", "envelope_sha256")) || string(value(p, "first_delivered_offer", "capture", "ack", "report_id")) != string(value(p, "first_delivered_offer", "central_receipt", "report_id")) {
			t.Fatal("delivery, durable store and central receipt no longer join")
		}
		p = restore + "AT30.json"
		wantValue(p, `"passed"`, "status")
		for _, tc := range []struct{ metric, current, delta string }{
			{"Pre-replay database RPO (s)", "92.163101", "-14.286899"},
			{"Total outage RTO (s)", "81.016113709", "6.486113709"},
		} {
			wantValue(p, tc.current, "development_host_deltas", tc.metric, "follow_up_13_seconds")
			wantValue(p, tc.delta, "development_host_deltas", tc.metric, "delta_seconds")
		}
		require(archive+"secops13.md", "Follow-up 12 archive", "byte-identical to PR #301", "Post-replay service usage RPO (s) | 0 | 0")
		require(doc, "SECOPS-21 on the assembled stack", "19 > retained 15", "harness-proved on the assembled stack, not a live-fleet guarantee", "RPO 92.163 s (−14.287 s)", "RTO 81.016 s (+6.486 s)", "published rounded comparison", "4 > 3 proof is separate")
	})
	t.Run("members9_F801_closed_F901_still_failing", func(t *testing.T) {
		p := fixture + "11e305fe-6360-4dbd-95b1-67274aa4988c-AT09.json"
		wantValue(p, `"fail"`, "status")
		var r struct {
			Steps []struct{ Name, Status, Actual string }
		}
		readJSON(p, &r)
		failures := 0
		for _, step := range r.Steps {
			if step.Status != "fail" {
				continue
			}
			failures++
			switch step.Name {
			case "D-067 revision 11 provisional lease observation":
				var v struct {
					Match bool `json:"all_match_revision_11"`
				}
				if err := json.Unmarshal([]byte(step.Actual), &v); err != nil || v.Match {
					t.Fatal("F9-01 predicate was promoted", err)
				}
			case "same-generation D-067 restart recovery and lease sequence":
				var v map[string]json.RawMessage
				if err := json.Unmarshal([]byte(step.Actual), &v); err != nil {
					t.Fatal(err)
				}
				for _, key := range []string{"desired_before", "desired_after", "applied_before", "applied_after"} {
					if string(v[key]) != `"30"` {
						t.Errorf("generation changed: %s = %s", key, v[key])
					}
				}
			default:
				t.Errorf("unexpected AT09 failure %s", step.Name)
			}
		}
		if failures != 2 {
			t.Fatalf("AT09 failing subchecks = %d", failures)
		}
		files, err := filepath.Glob(filepath.Join(root, fixture, "69f92f49-73dc-4779-8101-dfb6a802a96b-*"))
		if err != nil || len(files) != 13 {
			t.Fatalf("members8 adoption inventory %d: %v", len(files), err)
		}
		for _, file := range files {
			name := strings.TrimPrefix(filepath.Base(file), "69f92f49-73dc-4779-8101-dfb6a802a96b-")
			if readOwnedDoc(t, root, fixture+filepath.Base(file)) != readOwnedDoc(t, root, "docs/e2e/at/members/history/follow-up-8/"+name) {
				t.Errorf("members8 adoption changed %s", name)
			}
		}
		require(mainPin, "15.728602 s", "21.373814 s", "all 13 handover snapshots matching")
		require(doc, "F8-01 in the original generation-30 sequence", "AT09 is failing on F9-01", "Agent-core 27 must confirm the composition or fix a reporter race", "members 10", "D-067 revision-11 predicate to D-126")
	})
	t.Run("D126_D128_cumulative_report_and_value_equality", func(t *testing.T) {
		var report struct {
			Reasons []string `json:"forwarding_block_reasons"`
		}
		readJSON(archive+"mixed-observation.json", &report)
		if len(report.Reasons) != 2 || !slices.Contains(report.Reasons, "safety_check_failed") || !slices.Contains(report.Reasons, "awaiting_member_grants") {
			t.Fatalf("mixed reasons lost: %v", report.Reasons)
		}
		var evidence struct {
			Checks []struct {
				Name string
				Code string `json:"error_code"`
			}
			Errors []struct{ Stage, Code string }
		}
		readJSON(archive+"mixed-observation.json", &evidence)
		checks := 0
		for _, check := range evidence.Checks {
			if slices.Contains([]string{"lease", "qos", "source_binding"}, check.Name) {
				checks++
				if check.Code != "TC_FOREIGN_FILTER" {
					t.Errorf("%s code = %s", check.Name, check.Code)
				}
			}
		}
		lease := false
		for _, e := range evidence.Errors {
			lease = lease || e.Stage == "lease" && e.Code == "AWAITING_MEMBER_GRANTS"
		}
		if checks != 3 || !lease {
			t.Fatal("per-check priority and stage-specific cause lost")
		}
		require("internal/nodes/observations_followup32_test.go", "TestD126_MixedObservationEvidenceUnchanged")
		require("internal/nodes/httpwire/followup32_integration_test.go", "TestD126_MixedObservationReceiptAndProjections")
		require(doc, "safety failure > provisional / awaiting > ok", "stage's own cause", "202 Accepted", "keeps publication blocked", "replay returns the stored ACK", "complete JSON-value equality, byte-identical", "JCS output and the same SHA-256", "literal whitespace", "key order are not retained", "no schema change is planned", "D-113 is unchanged", "No revision-15 first-delivery marker", "nodes.release_fetcher = fixture", "dev-tag worker", "design advice, not implemented", "SQL shortcut stays")
	})
	t.Run("netlab19_unattributed_native_and_producer_timing", func(t *testing.T) {
		require(archive+"native-attribution.md", "No single transport cause is established", "11,011/64,064-byte", "no MSS rewriting at the observed node hops", "zero tc drops", "neither reproduces nor disproves", "omitting ordinary seq/ack/SACK", "underlay candidate family")
		require(archive+"producer-timing.md", "TimingEvidence()", "detached snapshot", "failed Apply timing record", "decimal JSON strings", "local diagnostic sibling of D-106", "17 losses", "missing/ambiguous matches remain unproved")
		require("node/enforce/timing_test.go", "TestD104_ISOF603_TimingSurvivesFailure", "TestD104_ISOF603_TimingRecoveryAndLiveRead", "TestD104_ISOF603_NoInventedStages")
		require(doc, "Handshake MSS is unchanged", "zero tc drops", "neither reproduces nor", "No single transport cause is established", "ServingEvidence.TimingEvidence()", "clear isolation's 17 historical renewal losses", "Isolation 9", "D-120's AT10 allowance does not apply", "IPv6Block by design")
	})
	t.Run("D125_producer_and_merged_updater17", func(t *testing.T) {
		require(archive+"agent26.md", "preserves the", "UUID's spelling", "never just relabels", "original checkpoint bytes", "removes checkpoint evidence", "malformed", "four existing members")
		require("node/agent/followup26_test.go", "TestD125_Rule1FreshBlockCheckpointWithLatchIdentity", "TestD125_Rule3SecondBlockQuietClosedEpochSameWatermarks", "TestD125_Rule3FinalDoneCannotRelabel", "TestD125_Rule4WithdrawalRemovesCheckpointBeforeResume", "TestD125_Rule5RestartPresentIdenticalRepublish", "TestD125_Rule5MalformedLatchPublishesNothingAndStaysBlocked")
		require("node/updater/followup17_test.go", "TestD125_RuleCorrelatedFreshPostBlockAccounting", "TestD125_RuleCorrelatedLateWithinBound", "TestD125_RulePreviousBlockIdentity", "TestD125_RuleMalformedIdentity", "TestD125_RuleCorrelatedPriorBootRejected", "TestD125_RuleByteEqualUUIDSpelling")
		require(doc, "every checkpoint names the latch identity it finalized against", "MeterCheckpoint.BarrierID", "barrier_id", "omitempty", "opaque spelling", "every new", "same-identity restart", "is merged (#323) at this cutoff", "byte-equal", "not yet proved")
	})
	t.Run("D118_active_locking_and_cancelled_ACK", func(t *testing.T) {
		require(mainPin, "D-118 active", "adoption", "usage.db.lock", "until final sampling", "done (#312)", "done (#320)")
		require("node/agent/followup26_test.go", "TestD118_LedgerOwnershipWaitsForPreviousLoops")
		require("node/agent/followup25_usage_test.go", "TestD118_ENR05_VerifiedReenrollmentHandsOffFinalSample", "TestD118_UnchangedIncarnationDoesNotRetire")
		require(doc, "real gap in #312", "RetirePreviousAt", "node lock", "usage.db.lock", "lifetime of the meter / uploader / store", "one body digest with one accepted event", "Runtime unchanged by that resume")
		require(register, "Active in agent-core follow-up 25 (#312)", "#320 fixes adoption under the node lock", "#324 passes live retired-stream hand-off")
		for _, line := range strings.Split(readOwnedDoc(t, root, register), "\n") {
			if strings.HasPrefix(line, "| D-118 |") && (strings.Contains(line, "inactive") || strings.Contains(line, "activation pending")) {
				t.Fatal("D-118 current register drift returned")
			}
		}
	})
	t.Run("D127_bounded_corrections_and_migration_order", func(t *testing.T) {
		require(archive+"fleet12.md", "one** correction", "same", "incarnation", "D-111 pre-health", "whose ACK applied a null-baseline seed", "every third distinct", "00192", "UUID `result_id`", "outcome_sequence", "Down fails atomically", "Deploy 00192 before the new control binary")
		require("internal/fleetupdate/installer_followup12_integration_test.go", "TestD127_SuccessAfterSeedMovesBothVersions", "TestD127_RollbackAfterSuccessMovesBothVersions", "TestD127_ExactReplayOfEitherOutcomeReturnsStoredAck", "TestD127_ThirdFlipIsAckConflict", "TestD127_OtherSecondEvidenceIsAckConflict", "TestD127_CorrectionRespectsCentralGenerationHighWater", "TestD127_Migration192RoundTripPreservesEvidenceAndBounds")
		require("internal/fleetupdate/installer_followup12_http_integration_test.go", "TestD127_CorrectionHTTPReplaysBothStoredAcks")
		require("docs/handover/11-database-and-retention.md", "00192_installer_outcome_corrections.sql", "outcome_sequence bounded to 1..2", "Apply before control; Down refuses once corrections exist")
		require(doc, "applied D-111 pre-health seed", "under the unchanged generation fence", "first receipt stays", "original ACK flags", "Replay of either payload returns its stored ACK", "third payload", "ACK_CONFLICT", "Deploy migration 00192 before the new control binary", "payload-derived", "outcome_sequence", "1..2", "Down", "harness-proved on PostgreSQL tests", "not yet run")
	})
	t.Run("usage5_live_retired_stream_and_same_key_replay", func(t *testing.T) {
		for n := 14; n <= 20; n++ {
			p := fmt.Sprintf("%s170ea3e3-AT%d.json", fixture, n)
			wantValue(p, `"170ea3e3"`, "run_id")
			wantValue(p, `"pass"`, "status")
		}
		p := fixture + "170ea3e3-AT20.json"
		wantValue(p, "true", "observations", "before_plus_held_equals_after")
		wantValue(p, "true", "observations", "reinstall_original_payload_bytes_equal")
		wantValue(p, "false", "observations", "reinstall_usage_files_byte_equal")
		for key, want := range map[string]string{"origin_central_ack": "2", "current_ack_before_origin_replay": "0", "current_ack_after_origin_replay": "0", "held_total_bytes": "7782"} {
			wantValue(p, strconv.Quote(want), "observations", key)
		}
		wantValue(p, `"33750"`, "observations", "member_period_before", "totals", "total_bytes")
		wantValue(p, `"41532"`, "observations", "member_period_after", "totals", "total_bytes")
		wantValue(p, "403", "observations", "retired_certificate_refusal", "http_status")
		var batches []struct {
			Key    string          `json:"idempotency_key"`
			Stored json.RawMessage `json:"stored_response"`
			Replay json.RawMessage `json:"replayed_response"`
		}
		if err := json.Unmarshal(value(p, "observations", "agent_batches_and_stored_replays"), &batches); err != nil || len(batches) != 1 {
			t.Fatalf("stored replay inventory: %v", err)
		}
		for _, batch := range batches {
			a, errA := canonjson.Canonicalize(batch.Stored)
			b, errB := canonjson.Canonicalize(batch.Replay)
			if batch.Key == "" || errA != nil || errB != nil || string(a) != string(b) {
				t.Fatal("same-key replay changed stored canonical ACK")
			}
		}
		require(archive+"usage5.md", "Follow-up 4", "immutable payloads", "independent ACKs", "No ledger rows or retirement timestamps are edited")
		require(doc, "AT14–AT20 pass in live run 170ea3e3", "controlled installer exit", "immutable", "tail across reinstall", "AT20 same-key replay", "33,750 + 7,782 = 41,532", "usage DB/WAL", "file hashes can change", "ENR-05 extension, not the DNS case")
	})
	t.Run("handover17_r17_archive_and_process_only_D124", func(t *testing.T) {
		var manifest any
		readJSON(archive+"scope-review17.json", &manifest)
		count := 0
		paths := map[string]bool{}
		var walk func(any)
		walk = func(v any) {
			switch v := v.(type) {
			case map[string]any:
				if path, ok := v["path"].(string); ok && v["sha256"] != nil {
					count++
					paths[path] = true
				}
				for _, x := range v {
					walk(x)
				}
			case []any:
				for _, x := range v {
					walk(x)
				}
			}
		}
		walk(manifest)
		if count != 242 || len(paths) != 141 {
			t.Fatalf("#319 pins %d/%d", count, len(paths))
		}
		require(mainPin, "11 subtests incl. `post_cutoff_secops_notice`", "through #313", "242 SHA-256 entries across 141 paths")
		require("docs/reviews/r17-rev14-d104-d117-review.md", "grok/grok-4.7", "62f24e6d", "### M-1 (Medium)", "r16 dispositions re-run", "## Not examined", "activation as pending")
		require("docs/reviews/README.md", "dcabc41e7d0c12725b1a7090425e2831faa3a12e", "3e2078550ddf24d365881dcee1399da474ae642bf76899f484a8c8ec60d787f7", "r16 H-1 / M-1 / M-2 / L-1 were re-run and held", "Fleet-update 12 (#322)")
		require(register, "D-124", "one-resume ownership extension", "internal/nodes/publication_evidence_test.go", "fixture-only", "Usage #324 also refreshes its deadline fixture")
		previous := -1
		for n := 98; n <= 128; n++ {
			pos := strings.Index(readOwnedDoc(t, root, register), fmt.Sprintf("| D-%03d |", n))
			if pos <= previous {
				t.Fatalf("decision %d missing or out of order", n)
			}
			previous = pos
		}
		for _, line := range strings.Split(readOwnedDoc(t, root, register), "\n") {
			if strings.HasPrefix(line, "| D-124 |") && (!strings.Contains(line, "Orchestration process only") || !strings.Contains(line, "No operator guarantee")) {
				t.Fatal("D-124 promoted")
			}
		}
		if strings.Contains(readOwnedDoc(t, root, doc), "D-124") {
			t.Fatal("D-124 belongs only in the decision log")
		}
	})
	t.Run("followup18_live_document_hashes", func(t *testing.T) {
		var manifest struct {
			Followup struct {
				Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
			} `json:"handover_follow_up_18"`
		}
		readJSON("docs/e2e/at/secops/scope-review.json", &manifest)
		if len(manifest.Followup.Documents) != 12 {
			t.Fatalf("live inventory = %d", len(manifest.Followup.Documents))
		}
		for _, pin := range manifest.Followup.Documents {
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, pin.Path)))
			if hex.EncodeToString(sum[:]) != pin.SHA256 {
				t.Errorf("live hash stale: %s", pin.Path)
			}
		}
	})

	t.Run("unpromoted_acceptance_and_D098_D109_integrity", func(t *testing.T) {
		for _, at := range []string{"AT29", "AT31"} {
			wantValue(archive+"cf81f6f8-4ccf-43d2-aae2-10121f6b169d-"+at+".json", `"blocked"`, "status")
		}
		checkHandoverRecord(t, root, "docs/e2e/at/enrollment/history/59ef4bf8-7d4a-490f-a74d-6b54394375e0/AT07.json", "59ef4bf8-7d4a-490f-a74d-6b54394375e0", "blocked", 36, 5, 0)
		require(archive+"updater17.md", "84b6a342", "does not contain PR #307", "PR #310", "Devstack 19 owns the serving-node rerun", "Installer 6 separately consumes")
		require(doc, "pre-#307 binary source `84b6a342`", "#307's F9-03 exit codes", "#310's fail-closed handshake", "post-upgrade re-enrollment seed `HASH_MISMATCH`", "pre-#307 extractor", "native MTU 1280 in both families", "Five AT07 rows", "Enrollment 8", "Independently clocked guest / 60-peer candidate hardware", "Real Windows, macOS, iOS and Android clients", "physical arm64 and power-loss qualification")
		var pins []struct {
			Path   string `json:"path"`
			Source string `json:"source"`
			Commit string `json:"source_commit"`
			SHA256 string `json:"sha256"`
		}
		readJSON(fixture+"followup18-snapshots.json", &pins)
		if len(pins) != 26 {
			t.Fatalf("snapshot count %d", len(pins))
		}
		seen := map[string]bool{}
		for _, pin := range pins {
			isReview := pin.Path == "docs/reviews/r17-rev14-d104-d117-review.md"
			commit := cut
			if isReview {
				commit = "dcabc41e7d0c12725b1a7090425e2831faa3a12e"
			}
			if pin.Source == "" || pin.Commit != commit || seen[pin.Path] {
				t.Fatalf("invalid provenance %+v", pin)
			}
			seen[pin.Path] = true
			if !isReview && !strings.Contains(pin.Path, "/history/") && !regexp.MustCompile(`/[0-9a-f]{8}(?:-|[0-9a-f-]*-)AT|/[0-9a-f-]+-SECOPS-21`).MatchString(pin.Path) {
				t.Fatalf("mutable evidence %s", pin.Path)
			}
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, pin.Path)))
			if hex.EncodeToString(sum[:]) != pin.SHA256 {
				t.Errorf("pinned bytes changed: %s", pin.Path)
			}
		}
		for _, path := range []string{"docs/e2e/at/usage/AT20.json", "docs/e2e/at/members/AT09.json", "docs/poc/G2-f802-native-attribution-2026-10-31.md", "docs/node/followup26-updater-2026-10-31.md"} {
			if claimTokenAllowed(path, "some prose") {
				t.Errorf("D-109 accepts living source %s", path)
			}
		}
	})
}

// TestHandover_Followup19Claims preserves the #337 cutoff. D-098 measurements
// read immutable snapshots; D-109 implementation pins name existing Go tests.
func TestHandover_Followup19Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "57a8cc9fc2ddb8178d3166642536dad6a1c3b95b"
	const fixture = "tests/handover/testdata/"
	const archive = fixture + "history/follow-up-19/"
	const doc = "docs/handover/follow-up-19.md"
	// Keep the #337 register and entrypoints frozen; follow-up 20 owns current claims.
	const frozen = fixture + "history/follow-up-20/"
	const register = frozen + "register19.md"
	readJSON := func(t *testing.T, path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	type claim struct {
		path   string
		tokens []string
	}
	for _, tc := range []struct {
		name   string
		claims []claim
	}{
		{"agent27_composition_and_timing", []claim{
			{archive + "agent27-members.md", []string{"single synchronous decision", "no HTTP contract", "sole stage=lease error", "TC_FOREIGN_FILTER", "A later clean read cannot rewrite the queued report"}},
			{archive + "agent27-isolation.md", []string{"already emits the qualifying observation", "Every actual Engine Apply", "live VerifyServing and blocked audit", "ServingEvidence.TimingEvidence()", "including errors", "12 losses", "10 loss windows", "32.271094", "51.114216", "physical arm64"}},
			{"node/agent/followup27_test.go", []string{"TestD126_AT09_CapturedProvisionalSafetyComposition", "TestD087_AT10_FixedGenerationProvisionalToGranted"}},
			{doc, []string{"deliberate single-decision composition", "no reporter race", "safety failure > provisional / awaiting > ok", "no producer fix was needed", "12/1,518", "32.271/51.114 ms", "17 older losses"}},
		}},
		{"installer6_correlated_success", []claim{
			{archive + "installer6.md", []string{"ENR-05 serving acceptance", "pre-#307 extractor", "exact", "inference", "FileMeter", "barrier_id", "13/13", "No live-tree repair"}},
			{"tests/installer/handoff_test.go", []string{"TestD089_RealSeedThenJournaledUpgrade"}},
			{doc, []string{"13/13 privileged tests", "post-switch revocation", "without tree repair", "exact mismatching entry"}},
		}},
		{"handover18_preserved_cut", []claim{
			{fixture + "followup19-main.json", []string{"done (#327)", "12 subtests, 26 snapshots", "exact r17 archive", "D-124 to D-128"}},
			{archive + "register18.md", []string{"through #324", "Follow-up 18 decisions register", "| D-124 |", "| D-128 |"}},
			{doc, []string{"Handover 18 (#327)", "12 subtests / 26 snapshots", "D-124–D-128 stay"}},
		}},
		{"members10_D129_and_remaining_failures", []claim{
			{archive + "members10.md", []string{"F9-01 is closed", "29 → 29", "38 / 5 / 0", "121 / 0 / 1", "80 / 0 / 1", "one unreadable withdrawal sample", "changed positive answer before", "F8-02 remains unattributed"}},
			{archive + "members10-requests.md", []string{"### F10-01", "### F10-02", "### F10-03", "234 samples", "read_ok=false", "TTL 10"}},
			{archive + "D129-PR328.txt", []string{"D-129", "desired == applied", "whatever generation the unchanged sequence yields", "30 is not a contract value"}},
			{doc, []string{"38/5/0", "29 → 29", "30 is not a contract value", "test-acceptance definition", "4/4", "121/0/1", "80/0/1", "dns-publish 9 / devstack 20"}},
		}},
		{"enrollment8_to_9_ENR07_closed", []claim{
			{"docs/e2e/at/enrollment/history/follow-up-8/summary.md", []string{"501", "43", "controlled_reinstall"}},
			{archive + "enrollment9.md", []string{"662 pass / 0 blocked / 0 failed", "48 pass / 0 blocked / 0 failed", "All five", "2724 bytes", "ACK **3**", "Physical arm64 and OS reboot remain unmeasured"}},
			{archive + "enrollment9-gap.md", []string{"planBaseline", "planSwitch", "15:11:08.150045Z", "15:11:08.150046Z", "BeforePlusHeldByteEquality", "OriginalPayloadAndHistoricalAttribution", "AgentSameKeyStoredResultReplay", "OriginACKAndIndependentCurrentACK", "ACK 1", "ACK 2"}},
			{fixture + "followup19-main.json", []string{"freeze withdrawal 187 ms", "51 byte-identical files"}},
			{doc, []string{"501/0/1", "43/1/2", "shared", "AT07 process exit", "planBaseline", "ENR-07", "662/0/0", "48/0/0", "550 + 358 = 908 bytes", "1 µs stored interval", "2,724 bytes / ACK 3", "187 ms", "51-file"}},
		}},
		{"D130_both_halves_and_safe_rollout", []claim{
			{archive + "usage-agent5.md", []string{"UTC whole-microsecond endpoints", "eventInterval", "planBaseline", "planSample", "planClockGap", "planSwitch", "Already sealed 1 ns events are never rewritten"}},
			{archive + "usage-central5.md", []string{"interval_order", "UTC().Truncate(time.Microsecond)", "usage_events_interval_check", "usage_samples_check", "usage_ledger_gaps_check1", "<=", "No", "constraint is disabled or bypassed", "Down fails atomically", "retryable `503 DEPENDENCY_UNAVAILABLE`", "next normal retry accepts the same sealed"}},
			{"node/internal/meter/precision_test.go", []string{"TestD130_AllIntervalPaths", "TestENR07_FirstGapSurvivesMicrosecondPersistence", "TestD130_LegacySealedGapUnchanged"}},
			{"internal/usage/interval_precision_integration_test.go", []string{"TestMTR03_ENR07_D130MigrationRetryACKAndStoredReplay", "TestMTR04_D130NanosecondSamplesAndRebuild", "TestMTR03_D130CanonicalIntervalOrderStillStrict"}},
			{doc, []string{"width ≥ 1 µs", "all four former 1 ns sites", "strict", "interval_order", "00193_usage_events_interval_precision.sql", "never bypassed", "Down refuses equal-endpoint", "503 DEPENDENCY_UNAVAILABLE", "without operator action"}},
			{"docs/handover/11-database-and-retention.md", []string{"00193_usage_events_interval_precision.sql", "before ingestion resumes", "Down refuses equal-endpoint rows", "canonical", "spool edits"}},
		}},
		{"devstack19_closures_and_retained_failures", []claim{
			{archive + "devstack19.md", []string{"14 / 0 / 0", "Run d6aaa798", "barrier_id", "13 / 0 / 1", "7/0/0", "15/0/1", "12,012", "RCODE 0", "SOA TTL 5", "all 15 row agreements", "no", "`agent.Decide` call", "central lease dependency seam"}},
			{archive + "devstack19-retention.md", []string{"All eight canonical payloads", "retained byte-for-byte"}},
			{doc, []string{"14/0/0", "pre-#307 binary caveat", "13/0/1", "7/0/0", "15/0/1", "12,012 bytes", "8/8", "15 agreements", "process fact, not a product claim"}},
		}},
		{"isolation9_open_product_defect", []claim{
			{archive + "isolation9.md", []string{"8 / 1 / 0", "2 / 1 / 0", "10 / 0 / 0", "5 / 1 / 1", "562 observation POSTs", "zero qualifying accepted reports", "4,767 / 5,021 / 5,025 ms", "Apply 955", "956", "957", "958", "GENERATION_MISMATCH", "generation-0 baseline", "Apply 961", "log is empty", "3,289 snapshots", "80/80", "557/557", "six assertions", "namespace"}},
			{doc, []string{"ISO-F6-03 OPEN", "AT13 5/1/1 FAIL", "real open product defect, not a harness limitation", "8.1 s before the deadline", "96 early drops / 65 positives / 41 final drops", "95 early drops outside", "writer is not identified", "not on main at #337", "3 losses / 1,396 probes", "54/294/1315", "36.090627", "50.196105", "no D-120 tolerance", "clock-namespace proof"}},
		}},
		{"wire_nodes33_D131_boundary", []claim{
			{archive + "wire-nodes33.md", []string{"httpwire.Open", "WithTestOnlyGrantEligibilityReaderFactory(func(pgx.Tx) leases.EntitlementReader)", "once in", "nil reader fails", "leases.DBEntitlementReader", "desiredstate.NewDBEntitlementReader(tx)", "no production", "configuration, environment switch or HTTP endpoint", "Each renewal samples the flag once", "does not revoke existing", "devstack 20 / isolation 10"}},
			{"internal/nodes/httpwire/followup33_integration_test.go", []string{"TestD087_AT10_ProductionConstructorAndHTTPFixedGenerationGrants", "TestD087_AT29_HTTPNilGrantReaderFailsClosed"}},
			{doc, []string{"D-131", "sampled once per renewal transaction", "nil reader fails closed", "no config / environment / HTTP surface", "bindings and generation never move", "devstack 20 → isolation 10 is not on main at #337"}},
		}},
		{"netlab20_immutable_external_pins", []claim{
			{archive + "netlab20.md", []string{"200 retained-file pins", "history/<snapshot>/", "under `docs/e2e/runs/`", "three archive findings are resolved", "unchanged digests", "no", "exceptions"}},
			{fixture + "followup19-main.json", []string{"200 / 200", "20 in-memory regression cases"}},
			{"tests/netlab/evidence_test.go", []string{"TestD098_EvidencePinValidation", "TestCommittedEvidenceFiles_ValidAndHygieneClean"}},
			{doc, []string{"200/200 pins", "20 regression cases", "history/<snapshot>/", "UUID run directory under docs/e2e/runs/", "unchanged digests"}},
		}},
		{"updates10_D132_and_blocked_regular_AT", []claim{
			{archive + "updates10.md", []string{"all 23 checks", "D-110", "D-111", "D-125", "D-127", "active release identity is inconsistent", "bin/fvpn-agent", "result_url", "TEST ONLY HTTP stimulus", "HASH_MISMATCH", "DOWNGRADE", "409", "supplies current pass claims"}},
			{archive + "D132-PR337.txt", []string{"D-132", "one-file exception", "internal/fleetupdate/d062_integration_test.go", "living", "history/follow-up-9/", "D062a/D062c", "f13 items"}},
			{"node/agent/followup24_test.go", []string{"TestUPD05_CandidateIdentityAndHealth"}},
			{fixture + "followup17-main.json", []string{"candidate release identity matches the running executable", "done (#305)"}},
			{"internal/fleetupdate/d062_integration_test.go", []string{"TestD062a_PrecheckFailureHarnessDelivery", "TestD062c_RollbackHarnessRequiresRestoredHealthEmptyAllowlist"}},
			{doc, []string{"23/23", "Regular AT23 is blocked", "#305", "#283", "no earlier pass is carried forward", "F10-01 → devstack 20", "F10-02 → installer 7", "result_url", "docs/e2e/at/<lane>/history/<follow-up>/", "fleet-update", "13 owns the follow-up"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.claims {
				requireHandoverClaimTokens(t, root, c.path, c.tokens...)
			}
		})
	}
	t.Run("cutoff_rows_and_current_entrypoints", func(t *testing.T) {
		var pin struct {
			Commit string
			Cutoff int `json:"cutoff_pr"`
			Rows   []string
		}
		readJSON(t, fixture+"followup19-main.json", &pin)
		if pin.Commit != cut || pin.Cutoff != 337 || len(pin.Rows) != 13 {
			t.Fatalf("wrong cutoff/inventory: %+v", pin)
		}
		for _, row := range pin.Rows {
			requireHandoverClaimTokens(t, root, "docs/ROADMAP.md", row)
		}
		for pr := 325; pr <= 337; pr++ {
			requireHandoverClaimTokens(t, root, fixture+"followup19-main.json", fmt.Sprintf("done (#%d)", pr))
			requireHandoverClaimTokens(t, root, "docs/ops/README.md", fmt.Sprintf("#%d", pr))
		}
		requireHandoverClaimTokens(t, root, doc, cut, "Handover ran no new live acceptance")
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			snapshot := frozen + "entrypoint-" + strings.ReplaceAll(path, "/", "_") + ".txt"
			lines := strings.Split(readOwnedDoc(t, root, snapshot), "\n")
			if len(lines) < 3 || !strings.Contains(lines[2], "Current cut") || !strings.Contains(lines[2], "follow-up-19.md") {
				t.Errorf("stale entrypoint: %s", path)
			}
		}
	})
	t.Run("frozen_acceptance_counts_and_no_promotion", func(t *testing.T) {
		for _, tc := range []struct {
			path, run, status   string
			pass, blocked, fail int
		}{
			{"docs/e2e/at/enrollment/history/follow-up-8/AT06.json", "f27a81a0-183e-46b4-85a8-304b2f64b635", "blocked", 501, 0, 1},
			{"docs/e2e/at/enrollment/history/follow-up-8/AT07.json", "f27a81a0-183e-46b4-85a8-304b2f64b635", "blocked", 43, 1, 2},
			{"docs/e2e/at/enrollment/history/891ee9b8-bf01-4583-a47a-9542c8b04c8a/AT06.json", "891ee9b8-bf01-4583-a47a-9542c8b04c8a", "pass", 662, 0, 0},
			{"docs/e2e/at/enrollment/history/891ee9b8-bf01-4583-a47a-9542c8b04c8a/AT07.json", "891ee9b8-bf01-4583-a47a-9542c8b04c8a", "pass", 48, 0, 0},
			{archive + "isolation9-AT13.json", "2be2797b-7a28-4e9f-a1fe-5fffa4c1b047", "fail", 5, 1, 1},
		} {
			checkHandoverRecord(t, root, tc.path, tc.run, tc.status, tc.pass, tc.blocked, tc.fail)
		}
		checkHandoverRecord(t, root, archive+"members10-AT09.json", "2be2d32f-bfd9-46f5-bf29-afaa6617ae94", "blocked", -1, 0, 0)
		var members struct {
			Steps []struct{ Name, Status, Actual string }
		}
		readJSON(t, archive+"members10-AT09.json", &members)
		counts := map[string]int{}
		restartRows := 0
		for _, step := range members.Steps {
			counts[step.Status]++
			if step.Name == "same-generation D-067 restart recovery and lease sequence" {
				restartRows++
				var values map[string]json.RawMessage
				if err := json.Unmarshal([]byte(step.Actual), &values); err != nil {
					t.Fatal(err)
				}
				for _, key := range []string{"desired_before", "desired_after", "applied_before", "applied_after"} {
					if string(values[key]) != `"29"` {
						t.Errorf("restart %s = %s", key, values[key])
					}
				}
			}
		}
		if restartRows != 1 {
			t.Fatalf("restart rows %d", restartRows)
		}
		if counts["pass"] != 38 || counts["blocked"] != 5 || counts["fail"] != 0 || len(members.Steps) != 43 {
			t.Fatal(counts)
		}
		for n := 23; n <= 27; n++ {
			checkHandoverRecord(t, root, fmt.Sprintf("%supdates10-AT%d.json", archive, n), "695f977c-b587-4768-88f1-0514f3a4f1ac", "blocked", -1, 0, 0)
		}
		checkHandoverRecord(t, root, archive+"updates10-installer.json", "695f977c-b587-4768-88f1-0514f3a4f1ac", "pass", -1, 0, 0)
		var installer struct {
			Observations []struct{ Name, Detail string }
		}
		readJSON(t, archive+"updates10-installer.json", &installer)
		passed := map[string]bool{}
		for _, o := range installer.Observations {
			if strings.HasPrefix(o.Name, "check_") {
				passed[o.Name] = true
				if o.Detail != "true" {
					t.Errorf("failed installer check %s: %s", o.Name, o.Detail)
				}
			}
		}
		if len(passed) != 23 {
			t.Fatalf("installer checks = %d", len(passed))
		}
	})
	t.Run("ISO_F603_exact_failure_and_renewal_counts", func(t *testing.T) {
		var expiry struct {
			Status string
			Early  struct {
				Count   int
				Outside int    `json:"outside_guard"`
				First   string `json:"first_ingress_offset_ns"`
			} `json:"early_dropout"`
			Final struct {
				Count     int
				Positives int `json:"post_deadline_positives"`
			} `json:"final_expiry_drops"`
		}
		readJSON(t, archive+"isolation9-expiry.json", &expiry)
		if expiry.Status != "fail" || expiry.Early.Count != 96 || expiry.Early.Outside != 95 || expiry.Early.First != "-8098843780" || expiry.Final.Count != 41 || expiry.Final.Positives != 0 {
			t.Fatalf("expiry changed: %+v", expiry)
		}
		var deadline struct {
			Passed           bool
			Positives, Drops int
			Errors           []string
		}
		readJSON(t, archive+"isolation9-deadline.json", &deadline)
		if deadline.Passed || deadline.Positives != 65 || deadline.Drops != 137 || !slices.Contains(deadline.Errors, "later positive after first drop") {
			t.Fatalf("deadline promoted: %+v", deadline)
		}
		var renewal struct {
			Probes, Delivered int
			LossCount         int `json:"loss_count"`
			Install           int `json:"loss_windows_overlapping_install"`
			Guard             int `json:"ingress_in_outer_guard"`
			Losses            []struct {
				Packet struct {
					ID int `json:"packet_id"`
				}
			}
		}
		readJSON(t, archive+"isolation9-continuity.json", &renewal)
		if renewal.Probes != 1396 || renewal.Delivered != 1393 || renewal.LossCount != 3 || renewal.Install != 3 || renewal.Guard != 3 || len(renewal.Losses) != 3 {
			t.Fatalf("renewal changed: %+v", renewal)
		}
		for i, id := range []int{54, 294, 1315} {
			if renewal.Losses[i].Packet.ID != id {
				t.Fatalf("loss ID changed: %+v", renewal.Losses)
			}
		}
	})
	t.Run("current_decisions_and_findings", func(t *testing.T) {
		s := readOwnedDoc(t, root, register)
		previous := -1
		for n := 98; n <= 132; n++ {
			pos := strings.Index(s, fmt.Sprintf("| D-%03d |", n))
			if pos <= previous {
				t.Fatalf("missing/out-of-order D-%03d", n)
			}
			previous = pos
		}
		requireHandoverClaimTokens(t, root, register, "| ENR-07 | CLOSED |", "| Members F9-01 | CLOSED |", "| ISO-F6-03 | OPEN — real product defect |", "| Updates F10-01 | OPEN |", "| Updates F10-02 | OPEN |", "| Members F10-02 | OPEN |", "| Members F10-03 | OPEN |", "| F8-02 / native MTU 1280 | OPEN — unattributed |", "AT29 / AT31 resources", "OS reboot, power-loss", "Agent-core 28 in flight, not on main at #337")
		for _, line := range strings.Split(s, "\n") {
			if strings.HasPrefix(line, "| D-124 |") && (!strings.Contains(line, "Orchestration process only") || !strings.Contains(line, "No operator guarantee")) {
				t.Fatal("D-124 promoted")
			}
		}
	})
	t.Run("D098_D109_provenance_and_external_pin_policy", func(t *testing.T) {
		var pins []struct {
			Path, Source, SHA256 string
			Commit               string `json:"source_commit"`
			SourceSHA            string `json:"source_sha256"`
			Offset, Length       int
		}
		readJSON(t, fixture+"followup19-snapshots.json", &pins)
		if len(pins) != 37 {
			t.Fatalf("pin inventory %d", len(pins))
		}
		seen := map[string]bool{}
		for _, pin := range pins {
			if seen[pin.Path] || !strings.Contains(pin.Path, "/history/") || pin.Source == "" {
				t.Fatalf("mutable or duplicate pin: %+v", pin)
			}
			seen[pin.Path] = true
			raw := readOwnedDoc(t, root, pin.Path)
			sum := sha256.Sum256([]byte(raw))
			if hex.EncodeToString(sum[:]) != pin.SHA256 {
				t.Errorf("snapshot changed: %s", pin.Path)
			}
			if pin.Source == "commit-message" {
				if pin.Length != len(raw) || pin.Offset <= 0 || len(pin.SourceSHA) != 64 || !slices.Contains([]string{"609ea71559375abdbb2546f47ff8a09cc41b2195", "fe816a8922a255772d5e88a9df9390ade303dc4f"}, pin.Commit) {
					t.Errorf("invalid merged PR excerpt: %+v", pin)
				}
			} else if pin.Commit != cut {
				t.Errorf("wrong source cut: %+v", pin)
			}
		}
		// The frozen source manifest must never point back to another lane's root.
		var sources struct {
			Files []struct{ Path, SHA256 string }
		}
		readJSON(t, archive+"netlab20-pins.json", &sources)
		if len(sources.Files) != 6 {
			t.Fatalf("source pin count %d", len(sources.Files))
		}
		for _, pin := range sources.Files {
			if !strings.Contains(pin.Path, "/history/") && !regexp.MustCompile(`^docs/e2e/runs/.*/[0-9a-f]{8}-[0-9a-f-]{27}/`).MatchString(pin.Path) {
				t.Errorf("mutable external pin: %s", pin.Path)
			}
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, pin.Path)))
			if hex.EncodeToString(sum[:]) != pin.SHA256 {
				t.Errorf("external archive changed: %s", pin.Path)
			}
		}
		for _, path := range []string{"docs/e2e/at/members/AT09.json", "docs/e2e/at/isolation/AT13.json", "docs/e2e/at/updates/AT23.json", "docs/node/followup27-isolation-2026-10-31.md", "docs/nodes/followup33-d087-seam-2026-10-31.md"} {
			if claimTokenAllowed(path, "some prose") {
				t.Errorf("D-109 permits mutable prose: %s", path)
			}
		}
	})
}

// TestHandover_Followup20Claims fixes the #344 cutoff. D-098 / D-109 / D-132
// keep evidence reads in immutable owned snapshots, never another lane's root.
func TestHandover_Followup20Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "96ec0630e4834e13a3a14d65553ba8108bb5da55"
	const fixture = "tests/handover/testdata/"
	const archive = fixture + "history/follow-up-20/"
	const doc = "docs/handover/follow-up-20.md"
	// Preserve all original assertions against the #344 register and entrypoints.
	const frozen = fixture + "history/follow-up-21/"
	const register = frozen + "register20.md"
	readJSON := func(t *testing.T, path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	type claim struct {
		path   string
		tokens []string
	}
	for _, tc := range []struct {
		name   string
		claims []claim
	}{
		{"usage6_D130_measurement", []claim{
			{archive + "usage6.md", []string{"Follow-up 6", "71b1a160", "ACK 1 before producing counted event 2", "1066 upload + 1513 download = 2579", "already sealed pre-D-130 1 ns retry/recovery belongs to enrollment 9", "D-118", "8/8 canonical origin"}},
			{archive + "usage5-archive.md", []string{"170ea3e3", "eba86d44", "64865721", "014daad8", "SHA256SUMS"}},
			{"node/internal/meter/precision_test.go", []string{"TestD130_AllIntervalPaths", "TestENR07_FirstGapSurvivesMicrosecondPersistence", "TestD130_LegacySealedGapUnchanged"}},
			{"tests/e2e/at/usage/live_test.go", []string{"TestATUsage_Live"}},
			{doc, []string{"AT14–AT20 pass", "planBaseline unknown-continuity gap", "ACK 1 before counted traffic", "skipped nor re-minted", "1,066 + 1,513 = 2,579 bytes", "59 files", "Enrollment 9 owns sealed-event recovery", "usage 6 measures newly minted", "not §19's DNS AT20"}},
		}},
		{"handover19_frozen", []claim{
			{fixture + "followup20-main.json", []string{"done (#339)", "16 subtests, 13 pinned roadmap rows, 37 immutable inputs", "migration 00193 rollout / Down", "follow-up 18 register frozen"}},
			{archive + "register19.md", []string{"through #337", "| D-124 |", "| D-132 |", "Agent-core 28 in flight, not on main at #337"}},
			{doc, []string{"Handover 19 remains frozen", "16 subtests / 13 pinned roadmap rows /", "37 immutable inputs", "Follow-up 18 remains frozen", "Down refuses equal-endpoint rows"}},
			{"docs/handover/11-database-and-retention.md", []string{"00193_usage_events_interval_precision.sql", "before ingestion resumes", "Down refuses equal-endpoint rows"}},
		}},
		{"fleet13_D132_owned_fixtures", []claim{
			{archive + "fleet13.md", []string{"all 23", "D-110 job ID", "D-127 / r17 M-2", "TEST ONLY HTTP stimulus", "ETag stays", "empty precheck record now fails explicitly", "scans every file"}},
			{archive + "fleet13-d062.txt", []string{"// D-132 source: docs/e2e/at/updates/history/follow-up-9/result-delivery.json; SHA-256 df19c0968d3e60e420fef36754f4b9c946f771279846fce29ec33450240f7d55.", "// D-132 source: docs/e2e/at/updates/history/follow-up-9/rollback-delivery-finding.json; SHA-256 c67f2d85cd75e494310e64a1f135cc1d67781fd4e97876190937bda428e1485c.", "testdata/d062-harness-result-delivery.json", "testdata/d062-harness-rollback-delivery-finding.json", "len(fixture.Results) == 0", "missing harness precheck result"}},
			{archive + "fleet13-paths.txt", []string{"fs.WalkDir", "path.Clean(reference)", "/history/", "mutable AT evidence reference"}},
			{"internal/fleetupdate/evidence_paths_test.go", []string{"TestD132_EvidenceReferencesUseImmutableHistory"}},
			{"internal/fleetupdate/d062_integration_test.go", []string{"TestD062a_PrecheckFailureHarnessDelivery", "TestD062c_RollbackHarnessRequiresRestoredHealthEmptyAllowlist"}},
			{doc, []string{"D-132 lane follow-up", "byte-identical", "one-line archive-path / SHA-256", "empty", "before indexing", "No product code, contract, migration or dependency changed"}},
		}},
		{"installer7_reporting_and_origin_question", []claim{
			{archive + "installer7.md", []string{"F10-02 is fixed", "validated HTTPS control origin", "13/13 PASS", "before host commands", "distinct", "wrong", "401", "config hash is", "unchanged"}},
			{archive + "installer7-config.md", []string{"--control-url", "--server-ca", "Exactly the supplied control origin", "install --allow-loopback", "exits 64 before host changes", "same-release install rerun repairs", "preserves", "unrelated updater fields"}},
			{"packaging/installer/config_test.go", []string{"TestUPD02_ReportingInputsFailBeforeInstall", "TestUPD02_ReportingRepairPreservesIdentityAndOtherConfig"}},
			{"tests/installer/handoff_test.go", []string{"TestD089_RealSeedThenJournaledUpgrade"}},
			{doc, []string{"Updates F10-02 FIXED (#341)", "POST /api/v1/agent/update-results", "enrollment server CA", "exactly the control origin", "TEST ONLY", "install --allow-loopback", "before host changes", "same-release reruns repair reporting fields", "13/13", "release host distinct from the", "control origin would be refused", "decision, not a defect, until updates 11 reports"}},
		}},
		{"updater18_nil_sink_root_integrity_and_F1801", []claim{
			{archive + "updater18.md", []string{"Engine.Flush", "VALIDATION_FAILED", "result_url_missing", "Receipt bytes/status remain pending", "repair sends once", "whole-tree", "40/40 iterations", "F18-01", "1/1/2", "UUID", "D-132"}},
			{"node/updater/followup18_test.go", []string{"TestUPD05_MissingResultURLKeepsReceiptPending", "TestUPD05_ResultConfigPolicyAndServerTrust", "TestAT25_ReleaseRootCandidateSwitchAndHealth"}},
			{doc, []string{"explicit `Engine.Flush`", "VALIDATION_FAILED", "result_url_missing", "identical receipt once", "whole-root integrity", "40/40", "immutable UUID-run pins only", "F18-01 remains pending with devstack 21", "1 / 1 / 2", "historical 13/0/1 campaign stays failed"}},
		}},
		{"devstack20_D087_composition", []claim{
			{archive + "devstack20.md", []string{"168 SHA-256 pins", "79e59c20", "7 / 0 / 0", "4,112 ms", "5ab1f11c", "3,980 ms", "47 original JSON body SHA-256 hashes", "103 original node timing records", "does not close assembled"}},
			{archive + "devstack20-up.txt", []string{"if [ \"${FVPN_E2E_GRANT_SEAM:-0}\" = 1 ]; then", "./tests/e2e/cmd/fvpn-e2e-control"}},
			{"tests/e2e/followup20_live_test.go", []string{"TestE2E_Followup20Grants"}},
			{doc, []string{"168 files", "FVPN_E2E_GRANT_SEAM=1", "deploy/devstack/up.sh:122", "SetMemberGrantsReady", "MemberGrantEvidence", "MemberGrantInvariants", "5ab1f11c", "79e59c20", "7 / 0 / 0", "3,980 / 4,112 ms", "blocker is closed", "isolation 10, pending at #344"}},
		}},
		{"devstack20_diagnostics_and_shared_update_closure", []claim{
			{archive + "devstack20.md", []string{"247 samples", "same-boot monotonic", "No unreadable sample was reproduced", "36 DNS matrix queries", "missing responder", "11 / 0 / 0", "state/meter.checkpoint", "15 / 0 / 1", "10,010 / 64,064 bytes", "exact", "drop reason is not established", "No MTU/MSS workaround"}},
			{archive + "devstack20-responses.md", []string{"D-102", "Updates 11 must consume", "tests/e2e/at/updates/live_test.go", "36/36 query traces complete", "cannot recover its missing responder"}},
			{"tests/e2e/followup20_counter_test.go", []string{"TestE2E_Followup20CounterExpiry"}},
			{"tests/e2e/followup20_regular_test.go", []string{"TestE2E_Followup20Regular"}},
			{doc, []string{"247 every-read-success counter samples", "36 complete per-query DNS traces", "Updates F10-01 shared side FIXED (#343)", "live_test.go:221/:436", "AT23–AT27", "15 / 0 / 1", "10,010 / 64,064 bytes", "No global MTU fix", "instrumented but still unattributed", "netlab 21", "docs/e2e/RESPONSES.md"}},
		}},
		{"agent28_restart_contract_and_unknown_writer", []claim{
			{archive + "agent28.md", []string{"does not identify the initial generation-0 writer", "0b191fdd", "955", "961", "Watchdog involvement is plausible, not proven", "watchdog log is empty", "signed desired/lease verification", "original lease/anchor", "independent bounded context", "retains the", "open meter epoch", "source=heartbeat", "source=baseline", "agent.log", "watchdog.log", "complete producer JSONL", "ordinary clean stop"}},
			{archive + "agent28-readme.md", []string{"Successful adoption retains the existing rules and absolute deadline", "Reboot or loss of the original clock anchor cannot adopt", "shared node lock", "queues the truthful failed verifier report", "ordinary process stop", "open meter epoch"}},
			{"node/agent/followup28_test.go", []string{"TestAT13_ISOF603_ProcessCancellationStillReadsKernel", "TestAT13_ISOF603_ExactProducerReplay955To961", "TestAT13_ISOF603_SameBootRestartKeepsRulesAndDeadline", "TestAT13_ISOF603_ControlStallKeepsVerifiedRules", "TestAT13_ISOF603_RestartWaitsForHealthyWatchdogRead", "TestAT13_ISOF603_RestartInvalidAuthorizationCloses"}},
			{"node/watchdog/followup28_test.go", []string{"TestAT13_ISOF603_HeartbeatReadPrecedesClockSample", "TestAT13_ISOF603_RefreshCancelsStaleDecision", "TestAT13_ISOF603_WatchdogBaselineReasons", "TestAT13_ISOF603_WatchdogCannotOverwriteRenewal"}},
			{doc, []string{"signed same-boot kernel adoption", "same-generation repair with the original lease", "baseline / heartbeat source kinds", "ISO-F6-03 remains OPEN", "955–961", "0b191fdd", "Boot was not the initial writer", "watchdog decision evidence was", "never retained", "that request is closed"}},
		}},
		{"D133_application_and_measured_limits", []claim{
			{archive + "agent28.md", []string{"152.125303 ms", "100 ms bound", "all four selected live gates and all four full agent suites pass", "r <= 3", "id >= 80", "Load average alone is not", "p50 / p90 / p99 / max", "all completed guards", "Failed operations remain separate", "25.492805", "24.757228", "29.006529", "27.931510", "38.074182", "40.343014", "51.672900", "50.464151", "43.600958", "41.244924", "52.596424", "57.924611", "two drops before D", "D−63.277263", "do not request a tolerance", "host", "contention for this delivery"}},
			{archive + "D133-PR344.txt", []string{"12:4x", "load ≈ 16", "A/B/A/B", "Decision (D-133 case 3)", "attributed to host contention", "No tolerance, bracket or predicate change"}},
			{"node/agent/followup21_netlab_test.go", []string{"TestD085_LiveReadNetlab"}},
			{doc, []string{"D-133 was applied to this delivery", "152.125 ms > 100 ms", "16", "A, B, A, B", "four live gates and four full agent suites", "0df1716d", "036a4829", "r ≤ 3 / idle ≥ 80%", "25.49 / 24.76", "29.01 / 27.93", "38.07 / 40.34", "51.67 / 50.46", "measured\nlimits remain open, not defects of record", "D−63.3 ms", "52.596 ms", "B1 57.92 ms", "No one requested a", "12:4x EDT", "14:0x EDT"}},
		}},
		{"operator_changes_and_bilingual_notice", []claim{
			{frozen + "entrypoint-docs_handover_operator-guide.md.txt", []string{"--control-url", "--server-ca", "exactly the control origin", "install --allow-loopback", "TEST ONLY", "result_url_missing", "VALIDATION_FAILED", "identical pending receipt once", "same absolute deadline", "open", "OS reboot or a lost anchor cannot adopt", "failed verifier report before repairing"}},
			{frozen + "entrypoint-docs_handover_deploy.md.txt", []string{"result_url_missing", "identical receipt", "TEST ONLY", "same-release rerun", "original valid lease", "owner\ndecision pending updates 11"}},
			{frozen + "entrypoint-docs_handover_member-guide.md.txt", []string{"同一次作業系統開機", "不會延長原到期時間", "原有收據", "TEST ONLY", "same OS boot", "deadline is unchanged", "original receipt", "ISO-F6-03 remains", "full AT09"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.claims {
				requireHandoverClaimTokens(t, root, c.path, c.tokens...)
			}
		})
	}
	t.Run("cutoff_roadmap_and_entrypoints", func(t *testing.T) {
		var pin struct {
			Commit string
			Cutoff int `json:"cutoff_pr"`
			Rows   []string
		}
		readJSON(t, fixture+"followup20-main.json", &pin)
		if pin.Commit != cut || pin.Cutoff != 344 || len(pin.Rows) != 7 {
			t.Fatalf("wrong cutoff/inventory: %+v", pin)
		}
		for _, row := range pin.Rows {
			requireHandoverClaimTokens(t, root, "docs/ROADMAP.md", row)
		}
		for pr := 338; pr <= 344; pr++ {
			requireHandoverClaimTokens(t, root, fixture+"followup20-main.json", fmt.Sprintf("done (#%d)", pr))
			requireHandoverClaimTokens(t, root, frozen+"entrypoint-docs_ops_README.md.txt", fmt.Sprintf("#%d", pr))
		}
		requireHandoverClaimTokens(t, root, doc, cut, "Handover ran no new live acceptance", "D-098 / D-109 / D-132")
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			snapshot := frozen + "entrypoint-" + strings.ReplaceAll(path, "/", "_") + ".txt"
			lines := strings.Split(readOwnedDoc(t, root, snapshot), "\n")
			if len(lines) < 4 || !strings.Contains(lines[2], "Current cut") || !strings.Contains(lines[2], "follow-up-20.md") || !strings.Contains(lines[3], "#337") {
				t.Errorf("stale entrypoint: %s", path)
			}
		}
	})
	t.Run("usage6_exact_fresh_gap_counted_event_and_replay", func(t *testing.T) {
		for n := 14; n <= 20; n++ {
			checkHandoverRecord(t, root, fmt.Sprintf("%s71b1a160-AT%d.json", archive, n), "71b1a160", "pass", -1, 0, 0)
		}
		var r struct{ Observations map[string]json.RawMessage }
		readJSON(t, archive+"71b1a160-AT20.json", &r)
		decode := func(key string, dst any) {
			t.Helper()
			if err := json.Unmarshal(r.Observations[key], dst); err != nil {
				t.Fatalf("%s: %v", key, err)
			}
		}
		type retained struct {
			Unchanged bool `json:"canonical_payload_unchanged"`
			Count     int  `json:"central_count_by_payload_sha256"`
			Event     struct {
				Seq     string `json:"event_seq"`
				Start   string `json:"interval_start"`
				End     string `json:"interval_end"`
				Quality string
				Gap     struct{ Reason string }
				Records []struct {
					Up   string `json:"upload_bytes"`
					Down string `json:"download_bytes"`
				}
			}
		}
		var gap, counted retained
		decode("d130_fresh_gap", &gap)
		decode("d130_counted_event", &counted)
		if !gap.Unchanged || gap.Count != 1 || gap.Event.Seq != "1" || gap.Event.Quality != "gap" || gap.Event.Gap.Reason != "unknown" || len(gap.Event.Records) != 0 || gap.Event.Start != "2026-10-31T16:05:33.253516Z" || gap.Event.End != "2026-10-31T16:05:33.253517Z" {
			t.Fatalf("fresh gap changed: %+v", gap)
		}
		if !counted.Unchanged || counted.Count != 1 || counted.Event.Seq != "2" || len(counted.Event.Records) != 1 || counted.Event.Records[0].Up != "1066" || counted.Event.Records[0].Down != "1513" {
			t.Fatalf("counted event changed: %+v", counted)
		}
		for key, want := range map[string]string{"d130_gap_ack": "1", "d130_current_ack_after_replays": "2", "d130_origin_ack_after_current_replays": "2"} {
			var got string
			decode(key, &got)
			if got != want {
				t.Errorf("%s = %s, want %s", key, got, want)
			}
		}
		var cli struct {
			ACK string `json:"acked_through"`
		}
		decode("current_cli_after_handoff", &cli)
		if cli.ACK != "1" {
			t.Fatalf("local gap ACK %s", cli.ACK)
		}
		var counters struct {
			Up   string `json:"delta_upload_bytes"`
			Down string `json:"delta_download_bytes"`
		}
		decode("d130_counted_transfer", &counters)
		if counters.Up != "1066" || counters.Down != "1513" {
			t.Fatalf("kernel counters %+v", counters)
		}
		var exact bool
		decode("d130_exact_counter_and_attribution_equality", &exact)
		if !exact {
			t.Fatal("attribution equality failed")
		}
		var batches []struct {
			Equal  bool `json:"stored_response_canonical_byte_equal"`
			Stored struct {
				ACK string `json:"highest_contiguous_seq"`
			} `json:"stored_response"`
		}
		decode("d130_current_batches_and_stored_replays", &batches)
		if len(batches) != 2 || !batches[0].Equal || !batches[1].Equal || batches[0].Stored.ACK != "1" || batches[1].Stored.ACK != "2" {
			t.Fatalf("stored replays %+v", batches)
		}
	})
	t.Run("immutable_usage5_and_devstack19_archives", func(t *testing.T) {
		for _, tc := range []struct {
			manifest, base string
			count          int
		}{
			{"docs/e2e/at/usage/history/follow-up-5/SHA256SUMS", "docs/e2e/at/usage/history/follow-up-5/", 59},
			{archive + "devstack19-archive.sha256", "", 168},
		} {
			lines := strings.Split(strings.TrimSpace(readOwnedDoc(t, root, tc.manifest)), "\n")
			if len(lines) != tc.count {
				t.Fatalf("%s: %d pins", tc.manifest, len(lines))
			}
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) != 2 {
					t.Fatalf("invalid archive pin %q", line)
				}
				if _, err := hex.DecodeString(fields[0]); err != nil || len(fields[0]) != 64 {
					t.Fatalf("invalid archive digest %q", fields[0])
				}
				// The owned, hash-pinned manifest preserves all 168 archival
				// entries. D-098 permits external reads only for history or
				// run-named files, not the campaign's root summary documents.
				if tc.base == "" && !regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).MatchString(fields[1]) {
					continue
				}
				sum := sha256.Sum256([]byte(readOwnedDoc(t, root, tc.base+fields[1])))
				if hex.EncodeToString(sum[:]) != fields[0] {
					t.Errorf("archive changed: %s", fields[1])
				}
			}
		}
	})
	t.Run("devstack20_immutable_run_verdicts", func(t *testing.T) {
		for _, tc := range []struct{ file, run, status string }{
			{"5ab1f11c-member-grants-fixed-generation.json", "5ab1f11c-a36b-4b77-bc70-85d902c93da6", "pass"},
			{"79e59c20-member-grants-fixed-generation.json", "79e59c20-f289-4679-9b37-01f060f2d376", "pass"},
			{"977efa83-finite-expiry-reader-diagnostics.json", "977efa83-e3bd-4ced-b548-dbc9763ed266", "pass"},
			{"d6b01926-update-good.json", "d6b01926-de6f-47e4-803b-ae3ea5c603df", "pass"},
			{"cce1a688-dns-negative.json", "cce1a688-6af5-483c-bd67-2d167c51c04b", "pass"},
			{"cce1a688-native-matrix-inputs.json", "cce1a688-6af5-483c-bd67-2d167c51c04b", "fail"},
		} {
			checkHandoverRecord(t, root, archive+tc.file, tc.run, tc.status, -1, 0, 0)
		}
		for _, tc := range []struct {
			file         string
			milliseconds int
		}{
			{"5ab1f11c-member-grants-fixed-generation.json", 3980},
			{"79e59c20-member-grants-fixed-generation.json", 4112},
		} {
			var r struct {
				Observations []struct{ Name, Detail string }
			}
			readJSON(t, archive+tc.file, &r)
			found := 0
			for _, o := range r.Observations {
				if o.Name != "proof" {
					continue
				}
				found++
				var proof struct {
					Distinct     bool `json:"distinct_reports"`
					Empty        bool `json:"empty_lease_report_join"`
					Equal        bool `json:"invariants_byte_equal"`
					Populated    bool `json:"populated_lease_report_join"`
					Milliseconds int  `json:"true_to_serving_receipt_ms"`
				}
				if err := json.Unmarshal([]byte(o.Detail), &proof); err != nil {
					t.Fatal(err)
				}
				if !proof.Distinct || !proof.Empty || !proof.Equal || !proof.Populated || proof.Milliseconds != tc.milliseconds {
					t.Fatalf("grant proof changed: %+v", proof)
				}
			}
			if found != 1 {
				t.Fatalf("proof count %d", found)
			}
		}
	})
	t.Run("D133_register_and_open_findings", func(t *testing.T) {
		s := readOwnedDoc(t, root, register)
		previous := -1
		for n := 98; n <= 133; n++ {
			pos := strings.Index(s, fmt.Sprintf("| D-%03d |", n))
			if pos <= previous {
				t.Fatalf("missing/out-of-order D-%03d", n)
			}
			previous = pos
		}
		requireHandoverClaimTokens(t, root, register,
			"A failed live timing gate is attributed before a delivery is accepted", "unchanged main vs the branch", "A, B, A, B", "runnable queue and idle, never by load average", "p50 / p90 / p99 / max over completed guards", "failed operations separately", "host state per leg", "lane fixes its own regression", "outside my ownership", "owner with both distributions", "both pass under equal quiet conditions", "host contention", "A product change never merges on an ownership claim alone", "12:4x EDT", "14:0x EDT", "no product value changed",
			"| Updates F10-02 | CLOSED |", "| Updates F10-01 shared side | CLOSED |", "| D-087 central seam | CLOSED |", "| D-087 assembled AT10 | OPEN |", "| ISO-F6-03 | OPEN — real product defect |", "| Members F10-04 | CLOSED under D-129 |", "| Members F10-02 | OPEN |", "| Members F10-03 | OPEN |", "| F8-02 / native MTU 1280 | OPEN — unattributed |", "| F18-01 | OPEN — classification confirmation |", "| D-133 measured limits | OPEN — not defects of record |", "| Dependabot #22 | OPEN — dependency hygiene |")
		for _, stale := range []string{"Agent-core 28 in flight", "| Updates F10-02 | OPEN |", "| Updates F10-01 | OPEN |", "devstack 20 → isolation 10 not on main"} {
			if strings.Contains(s, stale) {
				t.Errorf("stale current finding %q", stale)
			}
		}
	})
	t.Run("pending_acceptance_is_not_promoted", func(t *testing.T) {
		requireHandoverClaimTokens(t, root, doc, "Updates 11, isolation 10, members 11 and", "dns-publish 9 are running but have no results on main at #344", "Regular AT23–AT27 and the distinct artifact-origin question", "Members F10-02 / F10-03 re-measurements", "closed under D-129", "complete AT09 row is still pending", "devstack 21", "netlab 21", "Native MTU 1280 remains open under D-123", "AT11 real platform routes", "AT13 full continuous fault matrix", "AT29 / AT31 resources", "physical arm64 execution, OS reboot", "power loss and real platform clients", "no earlier", "AT23–AT27 pass is carried forward")
	})
	t.Run("dependabot22_hygiene_not_product_defect", func(t *testing.T) {
		var a struct {
			Number     int
			State      string
			Dependency struct {
				Package, Scope string
				Manifest       string `json:"manifest_path"`
			}
			Advisory struct {
				ID       string `json:"ghsa_id"`
				Severity string
			}
			Range string `json:"vulnerable_version_range"`
			Fixed string `json:"first_patched_version"`
		}
		readJSON(t, archive+"dependabot-22.json", &a)
		if a.Number != 22 || a.State != "open" || a.Dependency.Package != "minimatch" || a.Dependency.Scope != "development" || a.Dependency.Manifest != "apps/portal/package-lock.json" || a.Advisory.ID != "GHSA-g3hv-75wg-m249" || a.Advisory.Severity != "medium" || a.Range != "< 11.0.1" || a.Fixed != "11.0.1" {
			t.Fatalf("alert changed: %+v", a)
		}
		requireHandoverClaimTokens(t, root, archive+"portal-minimatch.txt", `"version": "3.1.5"`, `"dev": true`)
		requireHandoverClaimTokens(t, root, doc, "Dependabot #22 OPEN", "minimatch < 11.0.1", "GHSA-g3hv-75wg-m249", "owner **portal**", "dependency-hygiene item", "not a product defect until the portal", "reports reachability")
	})
	t.Run("D098_D109_D132_immutable_provenance", func(t *testing.T) {
		var pins []struct {
			Path, Source, SHA256 string
			Commit               string `json:"source_commit"`
			SourceSHA            string `json:"source_sha256"`
			Observed             string `json:"observed_at"`
			Offset, Length       int
		}
		readJSON(t, fixture+"followup20-snapshots.json", &pins)
		if len(pins) != 38 {
			t.Fatalf("pin inventory %d", len(pins))
		}
		seen := map[string]bool{}
		for _, pin := range pins {
			if seen[pin.Path] || !strings.HasPrefix(pin.Path, archive) || pin.Source == "" {
				t.Fatalf("invalid snapshot: %+v", pin)
			}
			seen[pin.Path] = true
			raw := readOwnedDoc(t, root, pin.Path)
			sum := sha256.Sum256([]byte(raw))
			if hex.EncodeToString(sum[:]) != pin.SHA256 {
				t.Errorf("snapshot changed: %s", pin.Path)
			}
			if pin.Length != 0 && (pin.Length != len(raw) || pin.Offset < 0 || len(pin.SourceSHA) != 64) {
				t.Errorf("invalid excerpt: %+v", pin)
			}
			switch pin.Source {
			case "commit-message":
				if !strings.HasPrefix(pin.Commit, "60270cd6") || pin.Length == 0 {
					t.Errorf("invalid D-133 merge excerpt: %+v", pin)
				}
			case "https://api.github.com/repos/TED-H-com/Freedom-VPN/dependabot/alerts/22":
				if pin.Observed == "" || pin.Commit != "" {
					t.Error("API capture must be dated separately from main")
				}
			default:
				if pin.Commit != cut {
					t.Errorf("wrong main cut: %+v", pin)
				}
			}
		}
		for _, path := range []string{"docs/e2e/at/usage/AT20.json", "docs/e2e/at/updates/AT23.json", "docs/e2e/RESPONSES.md", "docs/node/followup28-iso-f6-03-2026-10-31.md", "docs/updates/node-followup18-2026-10-31.md"} {
			if claimTokenAllowed(path, "living prose") {
				t.Errorf("mutable token pin allowed: %s", path)
			}
		}
	})
}

// TestHandover_Followup21Claims fixes #345–#351. Current acceptance claims
// use pinned roadmap rows; raw measurements are read only from owner history.
// Source prose excerpts are immutable owned snapshots of the exact main cut.
func TestHandover_Followup21Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "6d1e320dd7dee7b8a520f116a347e9f6f2d28d8e"
	const fixture = "tests/handover/testdata/"
	const archive = fixture + "history/follow-up-21/"
	const doc = "docs/handover/follow-up-21.md"
	const frozen = fixture + "history/follow-up-22/"
	const register = frozen + "register21.md"
	readJSON := func(t *testing.T, path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	type claim struct {
		path   string
		tokens []string
	}
	for _, tc := range []struct {
		name   string
		claims []claim
	}{
		{"dns9_defaults_and_inherent_inflight", []claim{
			{archive + "dns9-backend.md", []string{"F9-HIT", "F9-OTHER", "F9-ROUND", "F9-INFLIGHT", "4194304", "0 / 0", "false", "10 seconds", "`[]`", "about 9.35 seconds", "before assigning mechanism", "completion times", "Historical F10-03 stays failed and unattributed", "maximum cache lifetime, not a minimum retention or cross-query consistency promise"}},
			{doc, []string{"AdGuard Home v0.107.59", "cache_size: 4194304", "cache_ttl_min: 0", "cache_ttl_max: 0", "cache_optimistic: false", "upstream_timeout: 10 s", "fallback_dns: []", "F9-HIT", "F9-OTHER", "F9-ROUND", "F9-INFLIGHT", "new → old within 1.3 s", "inherent insertion ordering", "no exposed cache, mode or timeout setting fixes it", "TTL is a maximum", "lifetime", "stays failed / unattributed", "blocked_response_ttl remains the owner"}},
		}},
		{"dns9_attribution_and_mandatory_gate", []claim{
			{archive + "dns9-backend.md", []string{"An upstream exchange alone proves a fetch, not a switch of responder", "inferred_hit_no_authority_exchange", "Negative policy answers may be generated locally", "production `blocked_response_ttl` decision remains open"}},
			{archive + "dns9-pr-gate.txt", []string{"  unit:", "Pinned resolver positive cache and TTL (AT20)", "run: make -f mk/dns.mk dns-positive-cache"}},
			{"tests/netlab/dns/positive_cache_test.go", []string{"TestAT20_PositiveCacheTTL"}},
			{doc, []string{"BACKEND §2.1.5", "outstanding-request", "completion times", "mandatory **dns-positive-cache**", "not fallback", "Latency alone does not identify the cause"}},
		}},
		{"updates11_regular_campaign_and_artifact_origin", []claim{
			{fixture + "followup21-main.json", []string{"regular **AT23–AT27 5/5 pass**", "daddad24", "installer 27/27", "AT26 physical rollback", "AT27's mandatory witness", "allowed_artifact_origins grants private / CGNAT exceptions", "cached hand-off and uncached seed both succeed", "no hosting or policy change"}},
			{"tests/e2e/at/updates/live_test.go", []string{"TestUpdatesFollowup"}},
			{doc, []string{"27 / 27", "regular AT23–AT27 pass 5 / 5", "without an adapter", "AT26 physically", "AT27 requires that same-run physical rollback witness", "artifact-origin question is closed by measurement, no rejection", "private / CGNAT exceptions only", "needs no exception", "No private-mirror deployment"}},
		}},
		{"updates11_reporting_and_handoffs", []claim{
			{fixture + "followup21-main.json", []string{"wrapper refuses loopback exit 64", "generated result_url = the authenticated split-listener proxy", "13 result POSTs = 11 curl stimuli + one automatic success + one automatic failure", "result_url_missing never observed", "F18-01 stays the intended stopped-producer 1/1/2 negative", "F11-01 (devstack: installer_exports.go loopback opt-in", "F11-02 (updater: node-updater.md Offline installation paragraph)"}},
			{doc, []string{"64", "TEST ONLY --allow-loopback", "13 result POSTs", "11 stimuli + 1 automatic success + 1 automatic failure", "result_url_missing never observed", "F18-01 stays the D-125 negative", "1 / 1 / 2", "F11-01 → devstack 21", "F11-02 → updater 19", "39 files"}},
		}},
		{"handover20_frozen", []claim{
			{fixture + "followup21-main.json", []string{"done (#347)", "18 subtests / 7 pinned roadmap rows / 38 immutable inputs", "register D-133"}},
			{archive + "register20.md", []string{"through #344", "| D-087 assembled AT10 | OPEN |", "| Updates F10-01 shared side | CLOSED |", "| D-133 |"}},
			{"tests/handover/handover_test.go", []string{"TestHandover_Followup20Claims"}},
			{doc, []string{"#338–#344 at cutoff #344", "18 subtests / 7 pinned rows /\n38 immutable inputs", "source manifests and existing snapshots remain unchanged", "original assertions"}},
		}},
		{"members11_new_passes_preserve_historical_failures", []claim{
			{fixture + "followup21-main.json", []string{"Run 7edb263f on #343 / #344 / #336", "186 / 186", "AT20 blocked 81 / 0 / 0, 39 / 39 traces", "historical F10-02 / F10-03 failures stay failed and unattributed", "F10-04 CLOSED under D-129", "35 files, D-098"}},
			{"tests/e2e/at/members/counter_diagnostics_test.go", []string{"TestAT02_AT20_SharedDiagnosticSerialization"}},
			{doc, []string{"7edb263f", "186 / 186", "81 / 0 / 0", "39 / 39 traces", "failed / unattributed", "35-file follow-up 10 archive", "F10-04 closed under D-129", "30 is not a contract value"}},
		}},
		{"members11_AT02_new_readiness_and_failed_sibling", []claim{
			{fixture + "followup21-main.json", []string{"full AT02 fails 121 / 0 / 2", "generation 10 with zero member grants / AWAITING", "generation-13 failed Applies, HEALTH_CHECK_FAILED cause unattributed"}},
			{"tests/e2e/at/members/live_test.go", []string{"TestATMembers_Live"}},
			{doc, []string{"Full AT02 fails 121 / 0 / 2", "Members F11-02", "generation 10", "zero member grants / AWAITING", "members-readiness gate", "Members F11-03", "generation-13 failed Applies", "HEALTH_CHECK_FAILED", "unattributed → agent-core 29 /\n  devstack 21 / netlab 21"}},
		}},
		{"members11_AT09_IPv6_recurrence_and_AT01_gap", []claim{
			{fixture + "followup21-main.json", []string{"AT09 fails 37 / 5 / 1", "30 → 30", "serving 6.03 s", "first counted byte 11.40 s", "all three provisional reports pass D-126", "13,013 / 64,064 bytes in 12,010 ms", "F8-02 recurring over an IPv6 underlay", "distinct from devstack 20's IPv4 capture", "five member packets, no node / origin receipt", "2 pass / 6 blocked / 2 fail"}},
			{"tests/e2e/at/members/provisional_test.go", []string{"TestAT09_D126CapturedProvisionalSequence"}},
			{doc, []string{"Fresh AT09 fails 37 / 5 / 1", "30 → 30", "6.03 s", "11.40 s", "three provisional reports\npass D-126", "13,013 / 64,064 bytes in\n12,010 ms", "direct P2P IPv6 underlay", "F8-02 recurring", "IPv4", "five member packets\nwith no node / origin receipt → devstack 21", "2 pass / 6 blocked / 2 fail"}},
		}},
		{"portal19_alert_not_closable", []claim{
			{archive + "portal19-advisory.md", []string{"scope: development", "restored byte-for-byte", "npm audit fix", "resolved every path to 11.0.3", "clean `npm ci`", "jsx-a11y/label-has-associated-control", "@eslint/config-array", "@eslint/eslintrc", "minimatch@3.1.5", "minimatch@9.0.9", "no path into the\nbuilt production portal bundle", "No alert dismissal was performed"}},
			{fixture + "followup21-main.json", []string{"portal-verify 74 / 74 and portal-e2e 9 / 9", "eslint 9.39.5", "owner disposition requested"}},
			{doc, []string{"Dependabot #22 cannot be closed from the lockfile", "GHSA-g3hv-75wg-m249", "^11.0.1 → 11.0.3", "byte-identical", "no production-bundle path", "portal-verify 74 / 74", "portal-e2e 9 / 9", "Owner disposition\npending", "dependency-hygiene item, not a product defect"}},
		}},
		{"isolation10_real_D087_transition", []claim{
			{fixture + "followup21-main.json", []string{"Run bfcc3880 on baseline 96ec0630", "AT10 PASS 9 / 0 / 0 on the real D-087 transition", "false → natural signed empty lease → HTTP 202 ready / passed observation with forwarding false and awaiting_member_grants → true → distinct serving receipt joined to a new populated signed lease → counted native forwarding", "962.448 ms", "generation 13 and all 297 wire hashes byte-identical", "4,767 / 5,021 / 5,025 ms"}},
			{"tests/e2e/at/isolation/producer_timing_test.go", []string{"TestD087_OriginalWireAndAcceptedTransition"}},
			{doc, []string{"bfcc3880", "96ec0630", "AT10 PASS 9 / 0 / 0 on the real D-087 transition", "false → natural signed empty lease → HTTP 202", "awaiting_member_grants", "→ true → distinct serving receipt", "→ counted native forwarding", "962.448 ms", "generation 13", "297 wire hashes", "4,767 / 5,021 / 5,025 ms"}},
		}},
		{"isolation10_D104_not_historical_attribution", []claim{
			{fixture + "followup21-main.json", []string{"AT12 PASS 10 / 0 / 0", "AT13 BLOCKED 6 / 1 / 0", "161 positives / 41 drops", "D−99.655 ms", "D−63.565 ms", "one pre-D ingress loss counted", "historical ISO-F6-03 failure (96 / 65 / 41) stays failed and its writer unattributed", "AT11 BLOCKED 2 / 1 / 0"}},
			{"tests/e2e/at/isolation/deadline_trace_test.go", []string{"TestAT13_D104StrictBoundaryAndMissingEvidence"}},
			{doc, []string{"AT12 PASS 10 / 0 / 0", "AT13 BLOCKED 6 / 1 / 0", "161 positives / 41 drops", "D−99.655 ms", "D−63.565 ms", "one pre-D ingress loss\ncounted", "failure (96 / 65 / 41) stays failed", "unattributed", "AT11 BLOCKED 2 / 1 / 0", "New restart\nproofs cannot retrospectively identify it"}},
		}},
		{"isolation10_distributions_failed_attempts_and_limits", []claim{
			{fixture + "followup21-main.json", []string{"1,396 probes / 30 losses", "D-120 unchanged", "ISO-F6-01 6 / 6", "D-106 654 / 654", "31.09 / 40.53 / 48.85 / 52.93 ms outer", "16.11 / 22.32 / 27.70 / 30.62 ms install over 968 guards", "independent sample after agent-core 28's A/B", "49 failed installs incl. Apply 712 at 215.891 ms", "D-133", "f9 archived 86 / 86"}},
			{doc, []string{"1,396 probes / 30 losses", "D-120 unchanged", "ISO-F6-01 6 / 6", "D-106 654 / 654", "Completed guards (968)", "31.09 | 40.53 | 48.85 | 52.93", "16.11 | 22.32 | 27.70 | 30.62", "independent sample after agent-core 28's A/B", "49 failed installs", "Apply 712 at 215.891 ms", "14:1x chain-gate load → agent-core 29 under D-133", "host contention is not\nestablished as its cause", "12 → 13", "86 / 86"}},
		}},
		{"updater19_config_consumption_and_unmeasured_remainder", []claim{
			{archive + "updater19.md", []string{"TestUPD05_InstallerReportingConfigShape", "non-nil `HTTPSink`", "server trust is distinct from the", "No field divergence", "Physical arm64", "Filesystem/power-loss durability", "Real fleet rollout", "production central", "30 s wait / 100 ms poll", "37 updater and 8 updatejournal test files"}},
			{"node/updater/followup19_test.go", []string{"TestUPD05_InstallerReportingConfigShape"}},
			{"node/updater/followup18_test.go", []string{"TestUPD05_MissingResultURLKeepsReceiptPending"}},
			{doc, []string{"physical arm64, power loss, fleet\nrollout, production central, worst-case 30 s / 100 ms bounds", "HTTPSink", "endpoint-server CA", "result_url_missing is unreachable\nthere", "absent-result_url pending/repair negative remains unchanged", "No reporting-field divergence", "D-132 audit has zero\ndocs/e2e reads"}},
		}},
		{"updater19_offline_documentation_correction", []claim{
			{archive + "updater19-offline.md", []string{"initial and same-release installs retain both the verified", "without fetching the artifact", "only when the cache is absent", "does **not** fall back to downloading", "HASH_MISMATCH", "SIZE_MISMATCH", "ARTIFACT_INVALID"}},
			{doc, []string{"Updates F11-02 fixed", "verified archive", "absence chooses the guarded signed-URL download", "Corrupt/unreadable\narchives fail closed", "adds no runtime behavior"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.claims {
				requireHandoverClaimTokens(t, root, c.path, c.tokens...)
			}
		})
	}
	t.Run("cutoff_roadmap_and_current_entrypoints", func(t *testing.T) {
		var pin struct {
			Commit string
			Cutoff int `json:"cutoff_pr"`
			Rows   []string
		}
		readJSON(t, fixture+"followup21-main.json", &pin)
		if pin.Commit != cut || pin.Cutoff != 351 || len(pin.Rows) != 7 {
			t.Fatalf("wrong cut/inventory: %+v", pin)
		}
		seen := map[int]bool{}
		for _, row := range pin.Rows {
			requireHandoverClaimTokens(t, root, "docs/ROADMAP.md", row)
			for pr := 345; pr <= 351; pr++ {
				if strings.HasSuffix(row, fmt.Sprintf("| done (#%d) |", pr)) {
					if seen[pr] {
						t.Fatalf("duplicate PR #%d", pr)
					}
					seen[pr] = true
				}
			}
		}
		if len(seen) != 7 {
			t.Fatalf("wrong merged PRs: %v", seen)
		}
		for pr := 345; pr <= 351; pr++ {
			requireHandoverClaimTokens(t, root, frozen+"entrypoint-docs_ops_README.md.txt", fmt.Sprintf("#%d", pr))
		}
		requireHandoverClaimTokens(t, root, doc, cut, "Handover ran no new live acceptance", "D-098 / D-109 / D-132", "netlab 21 and agent-core 29 were not on main", "results are excluded")
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			snapshot := frozen + "entrypoint-" + strings.ReplaceAll(path, "/", "_") + ".txt"
			lines := strings.Split(readOwnedDoc(t, root, snapshot), "\n")
			if len(lines) < 4 || !strings.Contains(lines[2], "Current cut") || !strings.Contains(lines[2], "follow-up-21.md") || !strings.Contains(lines[3], "#344") {
				t.Errorf("stale current entrypoint: %s", path)
			}
		}
	})
	t.Run("findings_register_no_new_decision", func(t *testing.T) {
		s := readOwnedDoc(t, root, register)
		old := readOwnedDoc(t, root, archive+"register20.md")
		rows := regexp.MustCompile(`(?m)^\| D-[0-9]{3} \|.*$`)
		if !slices.Equal(rows.FindAllString(s, -1), rows.FindAllString(old, -1)) {
			t.Fatal("decision rows changed; D-133 must remain the latest")
		}
		requireHandoverClaimTokens(t, root, register, "No new decision: D-133 remains the latest", "| Members F10-04 | CLOSED under D-129 |", "| Updates F10-01 | CLOSED |", "| Artifact-origin question | CLOSED by measurement |", "| Updates F11-02 | CLOSED |", "| D-087 assembled AT10 | CLOSED |", "| F9-INFLIGHT | OPEN — inherent backend finding, documented |", "| Members F11-01 | OPEN — transport gap |", "| Members F11-02 | OPEN — readiness gate |", "| Members F11-03 | OPEN — cause unattributed |", "| Updates F11-01 | OPEN — shared wrapper |", "| ISO-F6-03 | OPEN — real product defect |", "| F8-02 / native MTU 1280 | OPEN — unattributed |", "| F18-01 | OPEN — classification confirmation |", "| Dependabot #22 | OPEN — dependency hygiene |", "| D-133 measured limits | OPEN — not defects of record |", "| Members F10-02 | HISTORICAL FAIL — unattributed |", "| Members F10-03 | HISTORICAL FAIL — unattributed |")
		requireHandoverClaimTokens(t, root, doc, "both underlays", "two D-133 measured limits", "52.596 ms", "blocked_response_ttl", "commercial feed approval", "AT11 real platform routes", "AT13 full fault matrix including physical\n  arm64", "AT29 / AT31 resources", "QUIC / PMTU\n  tooling", "no tolerance requested")
		for _, stale := range []string{"pending updates 11", "| D-087 assembled AT10 | OPEN |", "| Updates F11-02 | OPEN", "Portal owns reachability assessment", "have no results at this cutoff"} {
			if strings.Contains(s, stale) {
				t.Errorf("stale current disposition: %s", stale)
			}
		}
	})
	t.Run("operator_changes_and_bilingual_notice", func(t *testing.T) {
		requireHandoverClaimTokens(t, root, frozen+"entrypoint-docs_handover_operator-guide.md.txt", "Operator changes at #351", "F9-INFLIGHT", "10 s upstream timeout", "TTL is a maximum lifetime", "blocked_response_ttl", "owner's decision", "initial and same-release installs retain the verified archive", "Only an absent archive chooses the guarded signed-URL download", "corrupt or\nunreadable archive fails closed", "not offline\nenrollment", "Dependabot #22", "no production-bundle path", "11.0.3 override breaks", "dependency-hygiene item, not a product defect", "no\nrejection")
		requireHandoverClaimTokens(t, root, frozen+"entrypoint-docs_handover_deploy.md.txt", "private / CGNAT exceptions only", "no rejection occurred", "Initial and same-release installs retain the verified archive", "only absence chooses the guarded", "Corrupt archives fail closed", "Dependabot #22")
		requireHandoverClaimTokens(t, root, frozen+"entrypoint-docs_handover_member-guide.md.txt", "#351 截止", "TTL 是快取最長有效時間", "可能覆蓋較新的", "數值仍由營運負責人決定", "保留已驗證", "不代表可以離線註冊", "開發工具相依套件警示仍待負責人處置", "Members do not need to reinstall clients or change security settings", "DNS TTL is a maximum cache", "older outstanding", "Node installs retain a verified archive", "does not\nprovide offline enrollment", "no path into the production bundle was found")
	})
	t.Run("owner_archives_byte_identical", func(t *testing.T) {
		check := func(path, want string) {
			t.Helper()
			if !strings.Contains(path, "/history/") || !filepath.IsLocal(path) {
				t.Fatalf("D-098 / D-132 rejects archive path: %s", path)
			}
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
			if hex.EncodeToString(sum[:]) != want {
				t.Errorf("archive changed: %s", path)
			}
		}
		for _, tc := range []struct {
			base  string
			count int
		}{
			{"docs/e2e/at/members/history/follow-up-10/", 35},
			{"docs/e2e/at/updates/history/follow-up-10/", 39},
		} {
			lines := strings.Split(strings.TrimSpace(readOwnedDoc(t, root, tc.base+"SHA256SUMS")), "\n")
			if len(lines) != tc.count {
				t.Fatalf("%s inventory %d", tc.base, len(lines))
			}
			seen := map[string]bool{}
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) != 2 || !filepath.IsLocal(fields[1]) || filepath.Base(fields[1]) != fields[1] || seen[fields[1]] {
					t.Fatalf("invalid archive entry: %q", line)
				}
				seen[fields[1]] = true
				check(tc.base+fields[1], fields[0])
			}
		}
		const base = "docs/e2e/at/isolation/history/follow-up-9/"
		var pin struct {
			Files []struct{ Path, SHA256 string }
		}
		readJSON(t, base+"archive.json", &pin)
		if len(pin.Files) != 86 {
			t.Fatalf("isolation archive inventory %d", len(pin.Files))
		}
		seen := map[string]bool{}
		for _, f := range pin.Files {
			if !filepath.IsLocal(f.Path) || seen[f.Path] {
				t.Fatalf("invalid archive entry: %s", f.Path)
			}
			seen[f.Path] = true
			check(base+f.Path, f.SHA256)
		}
		t.Log("D-098: members 35/35, updates 39/39, isolation 86/86 archive digests match")
	})
	t.Run("historical_failures_and_attempts_stay_separate", func(t *testing.T) {
		for file, failedStep := range map[string]string{"AT02.json": "persistent withdrawal member-a3", "AT20.json": "positive/negative cache before TTL"} {
			path := "docs/e2e/at/members/history/follow-up-10/" + file
			checkHandoverRecord(t, root, path, "2be2d32f-bfd9-46f5-bf29-afaa6617ae94", "fail", -1, 0, 0)
			var r struct {
				Steps []struct{ Name, Status string }
			}
			readJSON(t, path, &r)
			found := false
			for _, step := range r.Steps {
				if step.Name == failedStep && step.Status == "fail" {
					found = true
				}
			}
			if !found {
				t.Fatalf("historical failure missing: %s", failedStep)
			}
		}
		checkHandoverRecord(t, root, "docs/e2e/at/isolation/history/follow-up-9/AT13.json", "2be2797b-7a28-4e9f-a1fe-5fffa4c1b047", "fail", 5, 1, 1)
		for _, path := range []string{"docs/e2e/at/isolation/history/follow-up-10-attempt-1/archive.json", "docs/e2e/at/isolation/history/follow-up-10-attempt-2/archive.json"} {
			var pin struct {
				Files []struct{ Path, SHA256 string }
			}
			readJSON(t, path, &pin)
			want := 92
			if strings.Contains(path, "attempt-2") {
				want = 98
			}
			if len(pin.Files) != want {
				t.Fatalf("%s: %d files", path, len(pin.Files))
			}
			for _, f := range pin.Files {
				if !filepath.IsLocal(f.Path) {
					t.Fatalf("invalid attempt entry: %s", f.Path)
				}
				sum := sha256.Sum256([]byte(readOwnedDoc(t, root, filepath.Join(filepath.Dir(path), f.Path))))
				if hex.EncodeToString(sum[:]) != f.SHA256 {
					t.Errorf("attempt changed: %s/%s", path, f.Path)
				}
			}
		}
	})
	t.Run("D098_D109_D132_immutable_inputs", func(t *testing.T) {
		var pins []struct {
			Path, Source, SHA256 string
			Commit               string `json:"source_commit"`
			SourceSHA            string `json:"source_sha256"`
			Offset, Length       int
		}
		readJSON(t, fixture+"followup21-snapshots.json", &pins)
		if len(pins) != 13 {
			t.Fatalf("snapshot inventory %d", len(pins))
		}
		seen := map[string]bool{}
		for _, pin := range pins {
			if seen[pin.Path] || !strings.HasPrefix(pin.Path, archive) || pin.Source == "" || pin.Commit != cut {
				t.Fatalf("invalid immutable snapshot: %+v", pin)
			}
			seen[pin.Path] = true
			// Provenance is metadata only. Never dereference a living AT root.
			if strings.HasPrefix(pin.Source, "docs/e2e/at/") && !strings.Contains(pin.Source, "/history/") {
				t.Fatalf("living AT source is not a permitted input: %s", pin.Source)
			}
			raw := readOwnedDoc(t, root, pin.Path)
			sum := sha256.Sum256([]byte(raw))
			if hex.EncodeToString(sum[:]) != pin.SHA256 {
				t.Errorf("snapshot changed: %s", pin.Path)
			}
			if pin.Length != 0 && (pin.Length != len(raw) || pin.Offset < 0 || len(pin.SourceSHA) != 64) {
				t.Errorf("invalid exact excerpt: %+v", pin)
			}
		}
		for _, path := range []string{"docs/e2e/at/members/F11-01-forwarding.json", "docs/e2e/at/members/F11-02-capture.json", "docs/e2e/at/isolation/summary.md", "docs/e2e/at/updates/installer-handoff.json", "docs/e2e/at/updates/installer-delivery-transport.json", "docs/dns/BACKEND.md", "docs/portal/DEV.md", "docs/updates/node-followup19-2026-10-31.md"} {
			if claimTokenAllowed(path, "living prose") {
				t.Errorf("D-109 permits mutable token pin: %s", path)
			}
		}
	})
	t.Run("followup20_original_inventory_frozen", func(t *testing.T) {
		for path, want := range map[string]string{
			"docs/handover/follow-up-20.md":                     "cd1cb05eac979753e558552fe3bf8f4ee8c65313532d4bbf2750fdaca2e0ab26",
			"tests/handover/testdata/followup20-main.json":      "68204d08db207e2b3f73b72c762a42585ae45e8a7574396ae4f4381cdb06aeb3",
			"tests/handover/testdata/followup20-snapshots.json": "2c4a6bb6140f502070665aca82da030eeafd43e3d4aa5645740001cfc4543533",
		} {
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
			if hex.EncodeToString(sum[:]) != want {
				t.Errorf("follow-up 20 changed: %s", path)
			}
		}
		// Only redirect mutable register/entrypoint reads, preserving all 18 cases.
		source := readOwnedDoc(t, root, "tests/handover/handover_test.go")
		start := strings.Index(source, "func TestHandover_Followup20Claims(")
		if start < 0 {
			t.Fatal("missing prior claim function")
		}
		end := strings.Index(source[start:], "func TestHandover_Followup21Claims(")
		if end < 0 {
			t.Fatal("missing claim functions")
		}
		prior := source[start : start+end]
		groups := regexp.MustCompile(`\{"[a-zA-Z0-9_]+", \[\]claim\{`).FindAllString(prior, -1)
		subtests := regexp.MustCompile(`t\.Run\("`).FindAllString(prior, -1)
		if len(groups)+len(subtests) != 18 {
			t.Fatalf("follow-up 20 cases: %d", len(groups)+len(subtests))
		}
	})
}

// TestHandover_Followup22Claims fixes #352–#356. D-098 / D-109 / D-132
// permit dated prose snapshots, roadmap rows and archived/run-ID evidence only.
func TestHandover_Followup22Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "45209609d33d567c5f04d49441799cdd3efac59b"
	const fixture = "tests/handover/testdata/"
	const archive = fixture + "history/follow-up-22/"
	const mainPin = fixture + "followup22-main.json"
	const doc = "docs/handover/follow-up-22.md"
	const frozen = fixture + "history/follow-up-23/"
	const register = frozen + "register22.md"
	readJSON := func(t *testing.T, path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	type claim struct {
		path   string
		tokens []string
	}
	for _, tc := range []struct {
		name   string
		claims []claim
	}{
		{"devstack21_loopback_fix_and_measurement", []claim{
			{mainPin, []string{"installerSpec.AllowLoopback", "exact local `ControlURL` gate", "TEST ONLY, default false elsewhere", "33b9621a 3 / 0 / 0", "certificate-free POST 403", "authenticated invalid body 400"}},
			{archive + "devstack21-response-1.md", []string{"requires both that boolean and the exact local origin", "unchanged configuration, not a", "installer 27/27", "5/5 without the former reporting adapter"}},
			{"tests/e2e/installer_loopback_test.go", []string{"TestENR02_InstallerFixtureLoopbackOptInScope"}},
			{"tests/e2e/followup21_live_test.go", []string{"TestE2E_Followup21Installer"}},
			{doc, []string{"Updates F11-01 fixed", "installerSpec.AllowLoopback", "exact local `ControlURL`", "default false elsewhere", "33b9621a", "3 / 0 / 0", "exit **0**", "unchanged", "**403**", "**400**", "**27 / 27**", ".24 collision"}},
		}},
		{"devstack21_dns_lifecycle_not_inflight_reproduction", []claim{
			{mainPin, []string{"start before dispatch", "outstanding requests", "joins late completions by identity", "cba88bbc 13 / 0 / 0", "36 / 36 complete query traces", "85 distinct authority requests", "70 identities", "4 KiB", "4 MiB"}},
			{archive + "devstack21-response-2.md", []string{"BACKEND §2.1.5", "completion times", "does not reproduce F9-INFLIGHT", "Missing history", "Hit wording stays inferred"}},
			{"tests/e2e/followup21_live_test.go", []string{"TestE2E_Followup21DNS"}},
			{"tests/e2e/dns_trace_test.go", []string{"TestAT20_EarlierAuthorityCompletionAcrossQueries"}},
			{"tests/e2e/internal/dnsrelay/trace_test.go", []string{"TestAT20_OutstandingLifecycleAndMissingReceipts", "TestAT20_RestartRetainsStreamsAndPendingRequest"}},
			{doc, []string{"start before\ndispatch", "late completions by identity", "BACKEND §2.1.5", "**4 KiB**", "**4 MiB**", "**13 / 0 / 0**", "**36 / 36**", "**85**", "**70**", "does not reproduce F9-INFLIGHT"}},
		}},
		{"devstack21_historical_stage_and_D102_answers", []claim{
			{archive + "devstack21-members.md", []string{"twelve generation-13", "install_ok=false", "Verification records", "neither\nresource exhaustion nor injection causality is established", "184.488440", "304.391743", "AT01 stays blocked"}},
			{archive + "devstack21-response-3.md", []string{"before node observable inner ingress", "All eight failed Applies", "causation is not established", "forwardingGate", "zero grants", "Keep the original generation-10 failed transfer"}},
			{archive + "devstack21-response-4.md", []string{"1/1/2", "CHECKPOINT_FAILED", "old 13/0/1 campaign stays failed", "D-125"}},
			{archive + "devstack21.md", []string{"origin slot `.24`", "first DNS run 99027e5d", "intentional fallback restarts incomplete"}},
			{doc, []string{"D-102", "members 11 / updates 11 / dns-publish 9", "after inner member egress and before observable inner node ingress", "all eight failures", "100 ms", "install_ok=false", "no verification", "failure stage, not its cause", "D-125 negative\n1 / 1 / 2"}},
		}},
		{"installer8_reconciliation_and_unchanged_gaps", []claim{
			{archive + "installer8.md", []string{"No installer behavior changed", "All rows agree", "literal JSON keys", "No test reads living acceptance evidence", "per distinct retained archive", "Offline installation paragraph", "F10-01 is **CLOSED**", "Distinct-endpoint CA and private-mirror", "Physical arm64", "power loss", "production signing/distribution/rollout"}},
			{mainPin, []string{"Privileged suite 13 / 13", "#342", "#351", "renderUpdater", "same-release reporting repair"}},
			{"packaging/installer/followup8_test.go", []string{"TestUPD05_RenderedReportingConfigShape"}},
			{"node/updater/followup19_test.go", []string{"TestUPD05_InstallerReportingConfigShape"}},
			{doc, []string{"no installer behaviour change", "Field-by-field", "#342", "#351", "literal JSON", "reads no living AT evidence", "per distinct retained archive", "F10-01 CLOSED", "#343", "#346", "13 / 13", "Distinct-endpoint CA / private-mirror", "physical arm64, power loss", "production rollout remain unmeasured"}},
		}},
		{"handover21_inventory_and_cut_frozen", []claim{
			{mainPin, []string{"21 subtests / 7 exact roadmap rows", "13 immutable owned snapshots", "350 owner-archive files", "done (#354)"}},
			{"tests/handover/handover_test.go", []string{"TestHandover_Followup21Claims"}},
			{doc, []string{"#345–#351", "6d1e320d", "21 subtests / 7 pinned\nrows / 13 immutable snapshots / 350 owner-archive files", "earlier redirected reads remain unchanged"}},
		}},
		{"netlab21_ipv6_mechanism_and_native_boundary", []claim{
			{mainPin, []string{"10,010 / 64,064 bytes in 12,012.403 ms", "19 reverse outer IPv6 packets of 1,360 bytes", "36 × 1,152-byte, one 928-byte and eleven 144-byte", "zero kernel drops in all four captures", "no ICMPv6 PTB", "4.479 ms", "one mechanism explains both underlays"}},
			{archive + "netlab21.md", []string{"receive-link rejection a\nsupported inference", "no kernel drop-reason trace", "No per-datagram capture exists for that members row", "after tunnel-endpoint underlay egress, before member", "no NetBird ICE negotiation"}},
			{"tests/netlab/scenario_f21_test.go", []string{"TestF21_AT09_IPv6Underlay"}},
			{doc, []string{"10,010 / 64,064 bytes in 12,012.403 ms", "**19**", "1,360 bytes", "36 × 1,152", "one 928", "eleven 144", "zero kernel\ndrops", "no ICMPv6 PTB", "4.479 ms", "One mechanism explains both underlays", "1,340-byte", "20 extra header bytes", "supported inference", "devstack 22", "No global MTU / MSS fix"}},
		}},
		{"netlab21_expiry_and_full_guard_distributions", []claim{
			{mainPin, []string{"[D−101 ms, D]", "26 / 7", "D−100.730 ms", "161 / 41", "101 completed reload guards", "32.480 / 41.400 / 45.396 / 52.675", "16.698 / 21.783 / 28.473 / 30.818", "historical 52.596 ms maximum is not a ceiling"}},
			{archive + "netlab21.md", []string{"100 ms expiry reserve is intentional", "1 ms truncation", "never calls release", "closed transaction", "No tolerance, D-120 extension or new acceptance pass"}},
			{doc, []string{"100 ms", "1 ms", "[D−101 ms, D]", "literal zero pre-D loss is\nfalse", "26 / 7", "D−100.730 ms", "161 / 41", "101 completed guards", "32.480 | 41.400 | 45.396 | 52.675", "16.698 | 21.783 | 28.473 | 30.818", "52.596 ms", "never opens the guard", "D-104 / D-120 / AT13 / AT29 predicates are unchanged"}},
		}},
		{"netlab21_real_rollback_and_unconfirmed_route", []claim{
			{mainPin, []string{"200 ms hold", "304.815 / 280.395 / 285.297 ms", "34.4 / 28.3 / 24.0 ms", "397 ingresses / 224 egresses / 173 losses", "150 outside", "native AT01 route is outside this fixture"}},
			{archive + "netlab21.md", []string{"one post-release verifier read failure per trial", "200 ms explicit baseline observation hold", "no automatic 100 ms or one-guard recovery ceiling", "independent monotonic bracket", "not the assertion that every loss fits a successful guard"}},
			{"tests/netlab/scenario_f21_test.go", []string{"TestF21_AT02_RollbackSibling"}},
			{doc, []string{"200 ms hold", "304.815 / 280.395 / 285.297 ms", "34.4 / 28.3 / 24.0 ms", "397 ingresses / 224 egresses / 173\nlosses", "150 outside", "not historical F11-03 outage durations", "member tunnel egress →\nnode ingress", "native AT01 route is outside this fixture"}},
		}},
		{"agent29_negative_contention_reproduction", []claim{
			{mainPin, []string{"Controlled CPU contention does NOT reproduce", "quiet / loaded / quiet / loaded", "384 on 24 logical CPUs at 0 % idle", "40 loaded Applies per intervention", "53.133 ms", "184.078 ms outer guard", "not** a causal conclusion", "loaded meter / restart / expiry failures are retained, not diagnosed"}},
			{archive + "agent29.md", []string{"negative reproductions, not passing tests", "All six failed", "historical\ncause", "S4", "53.133468", "L2", "TestLabExternalMeterReset", "TestLab04_RestartKeepsVerifiedLease"}},
			{doc, []string{"do NOT reproduce the install overrun", "384 on 24 logical CPUs at 0 % idle", "40 loaded\nApplies per intervention", "53.133 ms", "negative reproductions, not passing tests", "184.078 ms outer guard", "152 ms", "49 failed installs", "not a causal conclusion", "Loaded meter / restart / expiry failures remain failed and undiagnosed"}},
		}},
		{"agent29_opt_in_host_context_preserves_contract", []claim{
			{mainPin, []string{"1,874 Apply host brackets", "outside the measured Apply operation", "every existing row / sequence rule / D-106 digest input unchanged"}},
			{archive + "agent29.md", []string{"test_mode=true", "test_timing_path", "host.before_apply", "host.after_apply", "ok=false", "outside enforcement's measured operation", "Missing exit fields are unknown", "D-106\ncanonical digest rules are preserved"}},
			{"node/agent/measurement_host_test.go", []string{"TestF29_HostSampleAccountingAndUnavailable", "TestF29_ApplyHostContextIsOptInAndDoesNotChangeTiming"}},
			{doc, []string{"TEST ONLY host context", "sanitized returned command\nstatus", "outside the measured Apply operation", "D-106 digest inputs are unchanged", "1,874 Apply host brackets"}},
		}},
		{"agent29_baseline_semantics_and_S2_counterexample", []claim{
			{mainPin, []string{"Rollback is the node-wide baseline by design, not the previous generation", "repeated failures keep the baseline indefinitely", "100 ms bound is an install budget, not an outage ceiling", "successful Apply AND successful live verifier", "member_path_unconfigured", "not every `verification_HEALTH_CHECK_FAILED` baseline", "AT02's admitted positive control stays failed"}},
			{archive + "agent29.md", []string{"It does **not** restore the old generation", "authorization that this generation is revoking", "both dropped at baseline", "not F11-03 install rollback", "AT02 remains failed"}},
			{doc, []string{"node-wide generation-0 baseline by design, not the previous\ngeneration", "next successful Apply", "indefinitely", "100 ms bound is an install budget, not an outage ceiling", "member_path_unconfigured", "verification_HEALTH_CHECK_FAILED", "S2 is not the historical F11-03", "AT02's admitted positive control stays failed at this cutoff"}},
		}},
		{"agent29_no_D133_trigger_and_routed_requests", []claim{
			{mainPin, []string{"No D-133 A/B invoked", "no Apply / rollback ordering, timeout, policy or production forwarding change", "four Sources links", "history/follow-up-9/", "netlab 22", "members 12 / isolation 11"}},
			{archive + "agent29.md", []string{"command runtime from runnable delay/cancellation", "failure_stage", "host_context", "cause_status", "four follow-up 28 source links", "byte-identical follow-up 9"}},
			{doc, []string{"No **D-133 A/B** was invoked", "no Apply / rollback ordering, timeout, policy or", "production forwarding change", "four Sources links", "history/follow-up-9/", "executor command-lifecycle evidence → netlab 22", "→ D-135", "members 12 / isolation 11"}},
		}},
		{"D134_D135_decisions_and_operator_scope", []claim{
			{register, []string{"| D-134 | Decided 16:0x EDT", "| D-135 | 17:2x EDT", "provisional — pending the owner's confirmation", "no previous-generation restoration", "revocation always preserved", "baseline→recovery ending at the next successful Apply", "every loss counted", "reported, never waived", "separate enforcement design under D-133"}},
			{frozen + "entrypoint-docs_handover_operator-guide.md.txt", []string{"Operator changes at #356", "[D−101 ms, D]", "never extra authorization after D", "Count every\npre-D and renewal loss", "52.596 ms", "provisional, pending the owner's confirmation", "Apply closes every member on that node until the next successful Apply", "100 ms bound is an install budget, not an outage", "never waive it", "Historical AT02\nremains failed"}},
			{doc, []string{"D-134 (16:0x EDT)", "D-135 (17:2x EDT, provisional — pending the\nowner's confirmation)", "reported, never waived", "Owner decision — confirm provisional D-135", "no historical failed result becomes a pass"}},
		}},
		{"findings_and_remaining_owner_decisions", []claim{
			{register, []string{"| Updates F11-01 | CLOSED — fix + measurement |", "| Updates F10-01 | CLOSED |", "| F8-02 / native MTU 1280 | OPEN — common mechanism established |", "| Members F11-01 | OPEN — bounded transport gap |", "| Members F11-03 | OPEN — stage established, cause unproven |", "| Install-overrun cause | OPEN — unproven |", "| F9-INFLIGHT | OPEN", "| Members F11-02 | OPEN — readiness gate |", "| F18-01 | INTENDED D-125 NEGATIVE |", "| ISO-F6-03 | OPEN", "writer unattributed", "| Dependabot #22 | OPEN", "blocked_response_ttl", "D-135 confirmation is an explicit\nowner decision"}},
			{doc, []string{"expectation rewrite under provisional D-135 pending members 12", "F18-01** intended D-125 negative", "Every historical failure stays\nfailed and unattributed", "Physical arm64, power loss, production rollout", "not expanded §19 acceptance"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.claims {
				requireHandoverClaimTokens(t, root, c.path, c.tokens...)
			}
		})
	}
	t.Run("fixed_cutoff_and_current_entrypoints", func(t *testing.T) {
		var pin struct {
			Commit string
			Cutoff int `json:"cutoff_pr"`
			Rows   []string
		}
		readJSON(t, mainPin, &pin)
		if pin.Commit != cut || pin.Cutoff != 356 || len(pin.Rows) != 5 {
			t.Fatalf("wrong cutoff: %+v", pin)
		}
		seen := map[int]bool{}
		for _, row := range pin.Rows {
			requireHandoverClaimTokens(t, root, "docs/ROADMAP.md", row)
			for pr := 352; pr <= 356; pr++ {
				if strings.HasSuffix(row, fmt.Sprintf("| done (#%d) |", pr)) {
					if seen[pr] {
						t.Fatalf("duplicate PR #%d", pr)
					}
					seen[pr] = true
				}
			}
		}
		if len(seen) != 5 {
			t.Fatalf("wrong PRs: %v", seen)
		}
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			snapshot := frozen + "entrypoint-" + strings.ReplaceAll(path, "/", "_") + ".txt"
			lines := strings.Split(readOwnedDoc(t, root, snapshot), "\n")
			if len(lines) < 4 || !strings.Contains(lines[2], "Current cut") || !strings.Contains(lines[2], "follow-up-22.md") || !strings.Contains(lines[2], "#356") || !strings.Contains(lines[3], "#351") {
				t.Errorf("stale entrypoint: %s", path)
			}
		}
		requireHandoverClaimTokens(t, root, doc, cut, "Handover ran no new live acceptance", "D-098 / D-109 / D-132", "Devstack 22,", "members 12, isolation 11, updates 12 and netlab 22 were not merged", "results are excluded")
		for pr := 352; pr <= 356; pr++ {
			requireHandoverClaimTokens(t, root, frozen+"entrypoint-docs_ops_README.md.txt", fmt.Sprintf("#%d", pr))
		}
	})
	t.Run("owner_archives_114_and_82_byte_identical", func(t *testing.T) {
		check := func(path, want string) {
			t.Helper()
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
			if hex.EncodeToString(sum[:]) != want {
				t.Errorf("archive changed: %s", path)
			}
		}
		const base = "docs/e2e/runs/2026-10-31/followup20/"
		prose := map[string]string{"README.md": "devstack20-overview.md", "capture-cce1a688/README.md": "devstack20-capture-overview.md", "gates.md": "devstack20-gates.md", "scenarios.md": "devstack20-scenarios.md"}
		lines := strings.Split(strings.TrimSpace(readOwnedDoc(t, root, archive+"devstack20-archive.sha256")), "\n")
		if len(lines) != 114 {
			t.Fatalf("devstack inventory %d", len(lines))
		}
		seen := map[string]bool{}
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) != 2 || !filepath.IsLocal(fields[1]) || !strings.HasPrefix(fields[1], base) || seen[fields[1]] {
				t.Fatalf("invalid archive entry: %q", line)
			}
			path := fields[1]
			seen[path] = true
			if frozen, ok := prose[strings.TrimPrefix(path, base)]; ok {
				path = archive + frozen
			} else if !regexp.MustCompile(`[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}`).MatchString(path) {
				t.Fatalf("unscoped run file: %s", path)
			}
			check(path, fields[0])
		}
		const netlab = "docs/poc/G2-followup20-evidence/"
		var manifest struct {
			Files []struct {
				Path, SHA256 string
				Bytes        int
			}
		}
		readJSON(t, netlab+"archive.json", &manifest)
		if len(manifest.Files) != 82 {
			t.Fatalf("netlab inventory %d", len(manifest.Files))
		}
		seen = map[string]bool{}
		for _, p := range manifest.Files {
			if !filepath.IsLocal(p.Path) || seen[p.Path] {
				t.Fatalf("invalid netlab archive entry: %s", p.Path)
			}
			seen[p.Path] = true
			check(netlab+p.Path, p.SHA256)
			if len(readOwnedDoc(t, root, netlab+p.Path)) != p.Bytes {
				t.Errorf("archive length: %s", p.Path)
			}
		}
	})
	t.Run("UUID_scoped_installer_and_dns_records", func(t *testing.T) {
		type record struct {
			RunID        string `json:"run_id"`
			Status       string
			Observations []struct{ Name, Detail string }
		}
		load := func(path, id string) map[string]string {
			t.Helper()
			var r record
			readJSON(t, path, &r)
			if r.RunID != id || r.Status != "pass" {
				t.Fatalf("wrong run/status: %+v", r)
			}
			m := map[string]string{}
			for _, o := range r.Observations {
				if _, ok := m[o.Name]; ok {
					t.Fatalf("duplicate observation %s", o.Name)
				}
				m[o.Name] = o.Detail
			}
			return m
		}
		const base = "docs/e2e/runs/2026-10-31/followup21/"
		const installer = "33b9621a-41d7-49fb-bc27-1c3f05e4ab44"
		m := load(base+installer+"/installer-loopback-wrapper.json", installer)
		var install struct {
			Exit   int
			Status string
		}
		if err := json.Unmarshal([]byte(m["install"]), &install); err != nil {
			t.Fatal(err)
		}
		if install.Exit != 0 || install.Status != "installed" {
			t.Fatalf("install %+v", install)
		}
		var config struct {
			URL      string   `json:"result_url"`
			Origins  []string `json:"allowed_artifact_origins"`
			Loopback bool     `json:"allow_loopback"`
		}
		if err := json.Unmarshal([]byte(m["generated_reporting"]), &config); err != nil {
			t.Fatal(err)
		}
		if config.URL != "https://127.0.0.1:9444/api/v1/agent/update-results" || !config.Loopback || !slices.Equal(config.Origins, []string{"https://127.0.0.1:9444"}) {
			t.Fatalf("reporting shape %+v", config)
		}
		if m["config_sha256_before"] != "14a8975a8a36df9558b9b957e07613476e43de351281eadaae4657e860f06164" || m["config_sha256_after"] != m["config_sha256_before"] {
			t.Fatal("config changed across endpoint probes")
		}
		var auth map[string]string
		if err := json.Unmarshal([]byte(m["endpoint_authentication"]), &auth); err != nil {
			t.Fatal(err)
		}
		if auth["without_client_certificate"] != "403" || auth["with_client_certificate_invalid_body"] != "400" {
			t.Fatalf("endpoint authentication %+v", auth)
		}
		const dns = "cba88bbc-a7bf-49f8-9dab-3182123d02aa"
		m = load(base+dns+"/dns-history-integrity.json", dns)
		var counts map[string]int
		if err := json.Unmarshal([]byte(m["diagnostic_counts"]), &counts); err != nil {
			t.Fatal(err)
		}
		for k, want := range map[string]int{"complete": 36, "queries": 36, "distinct_authority_requests": 85, "requests_retained_across_queries": 70} {
			if counts[k] != want {
				t.Errorf("%s=%d want %d", k, counts[k], want)
			}
		}
	})
	t.Run("D098_D109_D132_snapshot_integrity_and_rejected_roots", func(t *testing.T) {
		var pins []struct {
			Path, Source, SHA256 string
			Commit               string `json:"source_commit"`
			SourceSHA            string `json:"source_sha256"`
			Offset, Length       int
		}
		readJSON(t, fixture+"followup22-snapshots.json", &pins)
		if len(pins) != 22 {
			t.Fatalf("snapshot inventory %d", len(pins))
		}
		seen := map[string]bool{}
		for _, p := range pins {
			if seen[p.Path] || !strings.HasPrefix(p.Path, archive) || !filepath.IsLocal(p.Path) || p.Commit != cut || len(p.SourceSHA) != 64 || p.Offset < 0 {
				t.Fatalf("invalid snapshot: %+v", p)
			}
			seen[p.Path] = true
			if strings.HasPrefix(p.Source, "docs/e2e/at/") || strings.HasPrefix(p.Source, "docs/poc/G2-followup21-evidence/") || strings.HasSuffix(p.Source, ".json") {
				t.Fatalf("living evidence source: %s", p.Source)
			}
			raw := readOwnedDoc(t, root, p.Path)
			sum := sha256.Sum256([]byte(raw))
			if hex.EncodeToString(sum[:]) != p.SHA256 || len(raw) != p.Length {
				t.Errorf("snapshot changed: %s", p.Path)
			}
		}
		for _, path := range []string{"docs/e2e/at/members/F11-01-forwarding.json", "docs/e2e/at/members/F11-03-capture.json", "docs/e2e/at/isolation/summary.md", "docs/poc/G2-followup21-evidence/analysis.json", "docs/node/followup29-install-contention-2026-10-31.json", "docs/node/followup29-install-contention-2026-10-31.md"} {
			if claimTokenAllowed(path, "living source prose") {
				t.Errorf("mutable token pin permitted: %s", path)
			}
		}
	})
	t.Run("followup21_original_assertions_and_decisions_frozen", func(t *testing.T) {
		for path, want := range map[string]string{
			"docs/handover/follow-up-21.md":                     "5985ca5618e973fcb3c862c084f59ef6e5dd414db9f51148322636aefbb206bd",
			"tests/handover/testdata/followup21-main.json":      "0057dbe122876a1c18c5a47aa61b5a306f200ffbf8274c8e6e7076b011201d33",
			"tests/handover/testdata/followup21-snapshots.json": "d6009c26bfbccc3d63d7113abe9753d3bb7e6d13f44a57654143cf0010c7a777",
		} {
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
			if hex.EncodeToString(sum[:]) != want {
				t.Errorf("prior cut changed: %s", path)
			}
		}
		source := readOwnedDoc(t, root, "tests/handover/handover_test.go")
		start := strings.Index(source, "func TestHandover_Followup21Claims(")
		end := strings.Index(source, "// TestHandover_Followup22Claims fixes")
		if start < 0 || end <= start {
			t.Fatal("missing claim function boundary")
		}
		prior := source[start:end]
		// Reverse only the permitted frozen-input redirects, then compare the
		// entire original function. Assertions, subtests and older redirects
		// must remain byte-identical to the #356 starting main.
		prior = strings.ReplaceAll(prior, "\tconst frozen = fixture + \"history/follow-up-22/\"\n", "")
		prior = strings.ReplaceAll(prior, "const register = frozen + \"register21.md\"", "const register = \"docs/handover/README.md\"")
		prior = strings.ReplaceAll(prior, "root, frozen+\"entrypoint-docs_ops_README.md.txt\", fmt.Sprintf", "root, \"docs/ops/README.md\", fmt.Sprintf")
		prior = strings.ReplaceAll(prior, "snapshot := frozen + \"entrypoint-\" + strings.ReplaceAll(path, \"/\", \"_\") + \".txt\"\n\t\t\tlines := strings.Split(readOwnedDoc(t, root, snapshot), \"\\n\")", "lines := strings.Split(readOwnedDoc(t, root, path), \"\\n\")")
		for _, name := range []string{"operator-guide", "deploy", "member-guide"} {
			prior = strings.ReplaceAll(prior, "root, frozen+\"entrypoint-docs_handover_"+name+".md.txt\",", "root, \"docs/handover/"+name+".md\",")
		}
		sum := sha256.Sum256([]byte(strings.TrimSpace(prior)))
		if hex.EncodeToString(sum[:]) != "0c940b5ff321f1abb6b025677949a26cf5a1594a99d0877bba1d061146ae1f43" {
			t.Fatal("follow-up 21 assertions changed beyond frozen-input redirects")
		}
		rows := regexp.MustCompile(`(?m)^\| D-[0-9]{3} \|.*$`)
		old := rows.FindAllString(readOwnedDoc(t, root, archive+"register21.md"), -1)
		current := rows.FindAllString(readOwnedDoc(t, root, register), -1)
		if len(current) != len(old)+2 || !slices.Equal(current[:len(old)], old) {
			t.Fatal("earlier decisions changed; only D-134 / D-135 may be added")
		}
	})
}

// Current document integrity is deliberately separate from immutable claims.
func TestHandover_Followup22LiveDocumentIntegrity(t *testing.T) {
	root := repoRoot(t)
	var manifest struct {
		Followup struct {
			Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
		} `json:"handover_follow_up_22"`
	}
	if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Followup.Documents) != 14 {
		t.Fatalf("live document inventory %d", len(manifest.Followup.Documents))
	}
	for _, p := range manifest.Followup.Documents {
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, p.Path)))
		if hex.EncodeToString(sum[:]) != p.SHA256 {
			t.Errorf("stale live digest: %s", p.Path)
		}
	}
}

// TestHandover_Followup23Claims fixes #357–#362. Current lane results use
// roadmap rows; D-098 / D-109 / D-132 forbid living evidence-root inputs.
func TestHandover_Followup23Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "bc011b6f0fb8324d78ad5540c698a15f568fa079"
	const fixture = "tests/handover/testdata/"
	const archive = fixture + "history/follow-up-23/"
	const frozen = fixture + "history/follow-up-24/"
	const mainPin = fixture + "followup23-main.json"
	const doc = "docs/handover/follow-up-23.md"
	const register = frozen + "register23.md"
	readJSON := func(t *testing.T, path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	type claim struct {
		path   string
		tokens []string
	}
	for _, tc := range []struct {
		name   string
		claims []claim
	}{
		{"month_boundary_incident_and_fixed_clock", []claim{
			{mainPin, []string{"from 2026-11-01 00:00 UTC", "stopped every chain at 23:09 EDT", "2026-11-01T00:00:00Z", "product validation unchanged", "87 test files, 1 fix, 23 candidates kept", "Six other-lane findings routed", "usage AT live cases a / b and hand-off, secops PITR and rate classes, members usage trace"}},
			{archive + "agent30-handoff.md", []string{"period_id 2026-10 does not overlap the interval", "four other chains were stopped before any push", "checkpoint verification and store timestamps", "[2026-10-31T23:59:00Z, 2026-10-31T23:59:10Z)", "period comes from its own start"}},
			{"node/internal/fakecontrol/usage_test.go", []string{"TestUsageCheckpointRoundTrip"}},
			{doc, []string{"PeriodID: \"2026-10\"", "first post-boundary\ngate run failed", "23:23 EDT", "internal/usage/ingest.go", "overlap predicate is unchanged", "87 test files", "one fix", "Six other-lane findings", "never pairs a calendar literal with the wall clock", "half-open interval belongs to the month of its start"}},
		}},
		{"handover22_frozen_inventory", []claim{
			{mainPin, []string{"done (#358)", "19 subtests / 5 exact roadmap rows / 22 immutable snapshots", "114-file devstack 20 and 82-file netlab 20", "D-134", "D-135"}},
			{"tests/handover/handover_test.go", []string{"TestHandover_Followup22Claims"}},
			{doc, []string{"#352–#356", "45209609", "19 subtests / 5 rows /\n22 snapshots", "114-file devstack-20", "82-file netlab-20", "earlier assertions remain intact"}},
		}},
		{"updates12_regular_wrapper_closure_and_retained_attempt", []claim{
			{mainPin, []string{"all 43 root files of run daddad24", "2babf9b6", "shared", "explicit TEST ONLY loopback opt-in", "regular AT23–AT27 PASS 5 / 5", "installer 27 / 27", "13 result POSTs", "uncached-origin fallback exits 0", "F11-01 CLOSED", "F11-02 CLOSED", "#351", "1 / 1 / 2", "path bug", "history/follow-up-12-first-attempt/", "0 requests"}},
			{"tests/e2e/at/updates/live_test.go", []string{"TestUpdatesFollowup"}},
			{"tests/e2e/at/updates/archive_test.go", []string{"TestD132_Followup11ArchivePins", "TestD098_Followup12FirstAttemptArchivePins"}},
			{doc, []string{"daddad24", "43 files", "2babf9b6", "regular AT23–AT27 5 / 5", "27 / 27", "#352 supplies the fix", "#359 supplies the regular-run", "F11-02 CLOSED", "D-125 negative\n1 / 1 / 2", "harness path bug", "no pass row is carried across runs", "0 requests"}},
		}},
		{"isolation11_host_context_and_results", []claim{
			{mainPin, []string{"all 100 root files of run bfcc3880", "49 historical failed installs", "host and verification **unavailable**", "cause_status=unproven", "279 historical renewal losses", "edd986c4", "AT10 PASS 9 / 0 / 0", "379 ms", "AT12 PASS 10 / 0 / 0", "AT11 2 / 1 / 0", "AT13 6 / 1 / 0", "2 carried operator requests"}},
			{"tests/e2e/at/isolation/producer_timing_test.go", []string{"TestD098_Followup10ArchivePins", "TestISO_F6_03_ProducerTimingCorrelation", "TestD087_OriginalWireAndAcceptedTransition"}},
			{doc, []string{"bfcc3880", "100 files", "49\nhistorical failed installs", "host / verification unavailable", "cause_status=unproven", "279 historical renewal losses", "edd986c4", "AT10 PASS 9 / 0 / 0", "AT12 PASS 10 / 0 / 0", "AT11 2 / 1 / 0", "AT13 6 / 1 / 0", "2 carried operator requests: AT11, AT13"}},
		}},
		{"isolation11_D104_losses_and_distributions", []claim{
			{mainPin, []string{"D-104 row passes 161 / 41", "D−99.872 ms", "D−62.925 ms", "one pre-D ingress loss counted", "1,396 probes / 18 losses", "7,885 producer records, 952 Applies", "0 install overruns", "7 failed Applies", "foreign-tc rejections before install", "945 guards", "31.55 / 41.57 / 48.39 / 56.77", "16.52 / 22.59 / 28.10 / 30.35", "D-106 639 / 639"}},
			{doc, []string{"D-104 passes 161 / 41", "D−99.872 ms", "D−62.925 ms", "1,396 probes / 18 losses", "every pre-D and renewal loss is counted", "952 Applies / 0 install overruns", "7 failed Applies", "ISO-F6-01 foreign-tc pre-install", "945 completed guards", "31.55 | 41.57 | 48.39 | 56.77", "16.52 | 22.59 | 28.10 | 30.35", "639 / 639"}},
		}},
		{"devstack22_native_ipv6_and_sender_control", []claim{
			{mainPin, []string{"all 63 follow-up 21 evidence files", "9b3a89e7", "13 / 0 / 2", "16,016 / 64,064 bytes in 12,011 ms", "15 reverse 1,360-byte", "all 52 upload fragment pairs", "no PTB", "64,064 bytes in 1 ms three times by sender fragmentation", "no reactive PMTU discovery proven", "155 pins"}},
			{archive + "devstack22.md", []string{"ConstrainSelectedNativePeer1280", "asymmetric receive-link fixture artifact", "does **not** test reactive PTB discovery"}},
			{archive + "devstack22-analysis.md", []string{"16,016 / 64,064 bytes in 12,011 ms", "64,064 bytes in 1 ms", "44", "not measured reactive PMTU discovery", "not\na captured kernel drop reason"}},
			{"tests/e2e/followup18_native_test.go", []string{"TestE2E_Followup22Native"}},
			{doc, []string{"63 evidence files", "followup21-archive.sha256", "followup21-archive/", "155 capture pins", "16,016 / 64,064 bytes in 12,011 ms", "15 reverse\n1,360-byte outer packets", "No PTB", "ConstrainSelectedNativePeer1280", "sender fragmentation", "Reactive PTB discovery was not measured"}},
		}},
		{"devstack22_five_SYN_pattern_exports_and_two_requests", []claim{
			{mainPin, []string{"five-SYN pattern reproduced", "loopback source", "no WireGuard transport data", "member inner egress → transport-data emission", "27 final captures", "2 requests"}},
			{archive + "devstack22-response.md", []string{"StartNativeCapture", "Sample", "Stop", "No structurally\nvalid WireGuard type-4", "handshake response reaching member underlay does not prove kernel/session\nacceptance", "NetBird 0.82.0", "does not replay all members 11 history"}},
			{"tests/e2e/native_capture_test.go", []string{"TestAT01_CaptureMissingStatsNeverMeansZeroDrops"}},
			{doc, []string{"five-SYN pattern is reproduced", "loopback source", "no WireGuard type-4 transport data", "Cause remains open", "StartNativeCapture", "Sample", "Stop", "2 requests", "members F11-01 adoption", "trace → devstack 23"}},
		}},
		{"members12_F1102_closed_F1103_blocked", []claim{
			{mainPin, []string{"all 43 root files of run 7edb263f", "b98b78ee", "0 pass / 4 blocked / 4 fail", "468 / 0 / 14", "AT01 / AT09 not re-run", "F11-02 closed by fresh rows", "generations 10 / 21", "F11-03 blocked — no failed Apply observed", "all 89 AT02 Applies succeeded", "3,976 attempts / 21 losses", "13 inside completed guards and 8 outside"}},
			{"tests/e2e/at/members/evidence_test.go", []string{"TestD098_Followup11Archive"}},
			{"tests/e2e/at/members/readiness_test.go", []string{"TestAT02_F12ReadinessRejectsEmptyLeaseAndLaterWithdrawal"}},
			{doc, []string{"7edb263f", "b98b78ee", "0 pass / 4 blocked / 4 fail", "468 / 0 / 14", "AT01 / AT09 were not re-run", "F11-02 CLOSED by fresh populated-grant rows", "historical failure stays failed", "F11-03 BLOCKED", "no failed Apply observed", "89 Applies succeeded"}},
		}},
		{"members12_new_findings_and_unqualified_reader", []claim{
			{mainPin, []string{"236.970 ms", "policy_missing", "9 losses", "F12-02", "1 MiB", "11 readiness failures", "monitor failure", "not yet live-qualified", "F12-03", "SOA TTL 2", "1.229 s", "F12-04", "SERVFAIL after 5,001 ms", "positive receipts from both resolvers", "12 requests routed"}},
			{archive + "members12-handoff.md", []string{"7b949b68", "streams to disk", "not live qualification", "monitor-expired-before-refund", "No repeat assembled run", "exact cause unproven", "responder/delivery cause unattributed"}},
			{"tests/e2e/at/members/timing_stream_test.go", []string{"TestAT02_F12TimingStreamExceedsDiagnosticLimit"}},
			{"tests/e2e/at/members/dns_attribution_checks_test.go", []string{"TestAT20_F12NegativePrecisionAndOutstandingCompletion"}},
			{doc, []string{"F12-01", "236.970 ms", "nine sibling losses", "F12-02", "1 MiB timing-export limit", "11 readiness failures", "7b949b68", "not yet live-qualified", "F12-03", "1.229 s", "F12-04", "SERVFAIL after 5,001 ms", "both resolvers", "12 requests routed"}},
		}},
		{"D136_provisional_semantic_and_bracket_for_all_baselines", []claim{
			{"node/agent/followup15_test.go", []string{"TestD067_StaleGenerationKeepsPolicyMissing"}},
			{"tests/e2e/at/members/apply_attribution_checks_test.go", []string{"TestAT02_F12RecoveryRequiresNextApplyAndCountsEveryLoss"}},
			{register, []string{"| D-136 | 23:15, 2026-10-31", "provisional — to be confirmed together with D-135", "leaseGenerationStale", "policy_missing", "Designed fail-closed behaviour, not a defect", "every node-wide baseline transition (failed Apply or stale lease)", "baseline → next successful Apply, every loss counted, recovery required, never waived", "agent-core design work under D-133 with an adversarial review"}},
			{frozen + "entrypoint-docs_handover_operator-guide.md.txt", []string{"Operator changes at #362", "D-136 stale lease — provisional", "a generation change can close every member on a node", "policy_missing", "until renewal and the next successful Apply", "recovery required, never waived"}},
			{doc, []string{"2026-10-31 — provisional, to be confirmed together with D-135", "D-135 stays provisional", "leaseGenerationStale", "every node-wide baseline transition", "baseline → next successful Apply", "every loss counted", "recovery\nrequired", "never waived", "adversarial review"}},
		}},
		{"D137_provisional_scope_black_hole_is_never_pass", []claim{
			{register, []string{"| D-137 | 23:15, 2026-10-31", "**provisional**", "spec line 365", "§19 AT09", "sender-learnable 1280 path", "reactive PTB from a router hop", "PMTU black hole", "never a pass", "historical F8-02 rows stay failed", "No global MTU / MSS clamp", "owner hardening option"}},
			{frozen + "entrypoint-docs_handover_operator-guide.md.txt", []string{"D-137 AT09 scope — provisional", "sender-learnable", "reactive PTB from a router hop", "never a pass", "lower\ntunnel MTU or an MSS clamp is an owner hardening option"}},
			{doc, []string{"D-137 (23:15 — provisional)", "spec line **365**", "sender-learnable 1280 path", "reactive PTB from\na router hop", "PMTU black\nhole", "never a pass", "Historical\nF8-02 rows stay failed", "No global MTU / MSS clamp", "owner hardening option", "does not establish the reactive-PTB branch"}},
		}},
		{"findings_unchanged_gaps_and_owner_requests", []claim{
			{register, []string{"| Updates F11-01 | CLOSED — fix + regular-run result", "| Members F11-02 | CLOSED — fresh populated-grant rows", "| Members F11-03 | BLOCKED — no failed Apply observed", "| Members F12-01 | CLASSIFIED — provisional D-136", "| Members F12-02 | OPEN", "| Members F12-03 | OPEN", "| Members F12-04 | OPEN", "| Install-overrun cause | OPEN — unproven", "| F9-INFLIGHT | OPEN", "| F18-01 | INTENDED D-125 NEGATIVE", "| ISO-F6-03 | OPEN", "writer unattributed", "| Dependabot #22 | OPEN", "blocked_response_ttl", "Every historical failure stays failed and unattributed"}},
			{doc, []string{"Owner decision 1: confirm D-135 + D-136 together", "Owner decision 2: confirm provisional D-137 scope", "lower-tunnel-MTU / MSS-clamp hardening option", "2026-11-01", "No outside-owned pin needed changing", "not expanded §19 acceptance"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.claims {
				requireHandoverClaimTokens(t, root, c.path, c.tokens...)
			}
		})
	}
	t.Run("cutoff_all_six_rows_and_entrypoints", func(t *testing.T) {
		var pin struct {
			Commit string
			Cutoff int `json:"cutoff_pr"`
			Rows   []string
			After  []int    `json:"merged_after_cutoff"`
			Absent []string `json:"absent_at_start"`
		}
		readJSON(t, mainPin, &pin)
		if pin.Commit != cut || pin.Cutoff != 362 || len(pin.Rows) != 6 || len(pin.After) != 0 || !slices.Equal(pin.Absent, []string{"w4-at-security-ops follow-up 14", "w1-netlab follow-up 22", "w4-at-members follow-up 13", "w3-devstack-e2e follow-up 23", "w2-agent-core follow-up 31", "w4-at-usage follow-up 7"}) {
			t.Fatalf("wrong starting cutoff: %+v", pin)
		}
		seen := map[int]bool{}
		for _, row := range pin.Rows {
			requireHandoverClaimTokens(t, root, "docs/ROADMAP.md", row)
			for pr := 357; pr <= 362; pr++ {
				if strings.HasSuffix(row, fmt.Sprintf("| done (#%d) |", pr)) {
					if seen[pr] {
						t.Fatalf("duplicate PR #%d", pr)
					}
					seen[pr] = true
				}
			}
		}
		if len(seen) != 6 {
			t.Fatalf("wrong PR set %v", seen)
		}
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			snapshot := frozen + "entrypoint-" + strings.ReplaceAll(path, "/", "_") + ".txt"
			lines := strings.Split(readOwnedDoc(t, root, snapshot), "\n")
			if len(lines) < 4 || !strings.Contains(lines[2], "Current cut") || !strings.Contains(lines[2], "follow-up-23.md") || !strings.Contains(lines[2], "#362") || !strings.Contains(lines[3], "#356") {
				t.Errorf("stale entrypoint: %s", path)
			}
		}
		for pr := 357; pr <= 362; pr++ {
			requireHandoverClaimTokens(t, root, frozen+"entrypoint-docs_ops_README.md.txt", fmt.Sprintf("#%d", pr))
		}
		requireHandoverClaimTokens(t, root, doc, cut, "#362, after #361", "Security-ops 14 was\nnot merged", "absent from main at start", "Handover ran no new live acceptance", "D-098 / D-109 / D-132")
	})
	t.Run("owner_archives_and_run_captures_421_files", func(t *testing.T) {
		check := func(path, want string) {
			t.Helper()
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
			if hex.EncodeToString(sum[:]) != want {
				t.Errorf("archive drift: %s", path)
			}
		}
		for path, digest := range map[string]string{
			"docs/e2e/at/updates/history/follow-up-11/SHA256SUMS":               "de8cebaa6e882a08d1f5a456f2512c0601058fa97cf15c3609ca4ae14a3b49b2",
			"docs/e2e/at/updates/history/follow-up-12-first-attempt/SHA256SUMS": "f20b4b62fb91286f84054473f379c8f9c76fbe27edc526d146990d317c0146ce",
			"docs/e2e/at/members/history/follow-up-11/SHA256SUMS":               "c7efec30c99c414c62d0aa739c0a62e8a1eee2bfa1016c647d71063c6b1b4f7d",
			"docs/e2e/at/isolation/history/follow-up-10/archive.json":           "034ad6ac8763c134207bd294f22f22e5c5c78937f48c444455e00daf4295ac59",
		} {
			check(path, digest)
		}
		total := 0
		for _, item := range []struct {
			base  string
			count int
		}{
			{"docs/e2e/at/updates/history/follow-up-11/", 43},
			{"docs/e2e/at/updates/history/follow-up-12-first-attempt/", 17},
			{"docs/e2e/at/members/history/follow-up-11/", 43},
		} {
			lines := strings.Split(strings.TrimSpace(readOwnedDoc(t, root, item.base+"SHA256SUMS")), "\n")
			if len(lines) != item.count {
				t.Fatalf("%s inventory %d", item.base, len(lines))
			}
			seen := map[string]bool{}
			for _, line := range lines {
				f := strings.Fields(line)
				if len(f) != 2 || !filepath.IsLocal(f[1]) || seen[f[1]] {
					t.Fatalf("invalid archive pin %q", line)
				}
				seen[f[1]] = true
				check(item.base+f[1], f[0])
				total++
			}
		}
		const isolation = "docs/e2e/at/isolation/history/follow-up-10/"
		var m struct {
			RunID string `json:"run_id"`
			Files []struct{ Path, SHA256 string }
		}
		readJSON(t, isolation+"archive.json", &m)
		if m.RunID != "bfcc3880-dc02-485d-990c-13811d6e8835" || len(m.Files) != 100 {
			t.Fatal("wrong isolation archive")
		}
		seen := map[string]bool{}
		for _, p := range m.Files {
			if !filepath.IsLocal(p.Path) || seen[p.Path] {
				t.Fatalf("invalid isolation path %s", p.Path)
			}
			seen[p.Path] = true
			check(isolation+p.Path, p.SHA256)
			total++
		}
		uuidPath := regexp.MustCompile(`/(?:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})/`)
		for _, item := range []struct {
			name  string
			count int
		}{{"devstack21-archive.sha256", 63}, {"devstack22-captures.sha256", 155}} {
			lines := strings.Split(strings.TrimSpace(readOwnedDoc(t, root, archive+item.name)), "\n")
			if len(lines) != item.count {
				t.Fatalf("%s inventory %d", item.name, len(lines))
			}
			seen = map[string]bool{}
			for _, line := range lines {
				f := strings.Fields(line)
				if len(f) != 2 || !filepath.IsLocal(f[1]) || seen[f[1]] {
					t.Fatalf("invalid devstack pin %q", line)
				}
				seen[f[1]] = true
				path := f[1]
				const base = "docs/e2e/runs/2026-10-31/"
				if item.name == "devstack21-archive.sha256" && (path == base+"followup21/README.md" || path == base+"followup21/gates.md" || path == base+"followup21/scenarios.md") {
					path = archive + "devstack21-" + filepath.Base(path)
				} else if !(strings.HasPrefix(path, base+"followup21/") && uuidPath.MatchString(path)) && !strings.HasPrefix(path, base+"followup21-archive/") && !(strings.HasPrefix(path, base+"followup22/") && uuidPath.MatchString(path)) {
					t.Fatalf("unscoped run file %s", path)
				}
				check(path, f[0])
				total++
			}
		}
		if total != 421 {
			t.Fatalf("archive total %d", total)
		}
	})
	t.Run("D098_D109_D132_immutable_sources_and_rejected_roots", func(t *testing.T) {
		var pins []struct {
			Path, Source, SHA256 string
			Commit               string `json:"source_commit"`
			Kind                 string `json:"source_kind"`
			SourceSHA            string `json:"source_sha256"`
			Offset, Length       int
		}
		readJSON(t, fixture+"followup23-snapshots.json", &pins)
		if len(pins) != 18 {
			t.Fatalf("snapshot inventory %d", len(pins))
		}
		seen := map[string]bool{}
		for _, p := range pins {
			if seen[p.Path] || !strings.HasPrefix(p.Path, archive) || !filepath.IsLocal(p.Path) || len(p.Commit) != 40 || len(p.SourceSHA) != 64 || p.Offset < 0 {
				t.Fatalf("invalid snapshot %+v", p)
			}
			seen[p.Path] = true
			if strings.HasPrefix(p.Source, "docs/e2e/at/") {
				t.Fatalf("living evidence source %s", p.Source)
			}
			if p.Kind == "file" && p.Commit != cut {
				t.Fatal("wrong file snapshot commit")
			}
			if p.Kind != "file" && p.Kind != "commit_message" {
				t.Fatal("unknown source kind")
			}
			raw := readOwnedDoc(t, root, p.Path)
			sum := sha256.Sum256([]byte(raw))
			if hex.EncodeToString(sum[:]) != p.SHA256 || len(raw) != p.Length {
				t.Errorf("snapshot changed: %s", p.Path)
			}
		}
		for _, path := range []string{"docs/e2e/at/members/F12-01-policy-missing.json", "docs/e2e/at/members/F12-02-stream-limit.json", "docs/e2e/at/members/F12-03-negative-cache.json", "docs/e2e/at/members/F12-04-native-servfail.json", "docs/e2e/at/isolation/summary.md", "docs/e2e/at/updates/follow-up-12.md", "node/agent/decide.go", "internal/usage/ingest.go"} {
			if claimTokenAllowed(path, "living implementation or evidence token") {
				t.Errorf("mutable pin allowed: %s", path)
			}
		}
	})
	t.Run("followup22_original_assertions_and_decisions_preserved", func(t *testing.T) {
		for path, want := range map[string]string{
			"docs/handover/follow-up-22.md":                     "75a867f182d16337c2767de319d499240448d3c667ea5dd8b8f679e848aa3e56",
			"tests/handover/testdata/followup22-main.json":      "8dfec5e6cbbb5bfd182d346642a33996be1ece486132638cb247cd12b5ebd0de",
			"tests/handover/testdata/followup22-snapshots.json": "9b3d8ca862f1dcc59cde1f8f18a1426cf818551f11af53122aaf7c872d74b125",
		} {
			sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
			if hex.EncodeToString(sum[:]) != want {
				t.Errorf("prior cut changed: %s", path)
			}
		}
		source := readOwnedDoc(t, root, "tests/handover/handover_test.go")
		start := strings.Index(source, "func TestHandover_Followup22Claims(")
		end := strings.Index(source[start:], "// Current document integrity")
		if start < 0 || end < 0 {
			t.Fatal("missing original function boundary")
		}
		prior := source[start : start+end]
		prior = strings.ReplaceAll(prior, "\tconst frozen = fixture + \"history/follow-up-23/\"\n", "")
		prior = strings.ReplaceAll(prior, "const register = frozen + \"register22.md\"", "const register = \"docs/handover/README.md\"")
		prior = strings.ReplaceAll(prior, "{frozen + \"entrypoint-docs_handover_operator-guide.md.txt\", []string{\"Operator changes at #356\"", "{\"docs/handover/operator-guide.md\", []string{\"Operator changes at #356\"")
		prior = strings.ReplaceAll(prior, "root, frozen+\"entrypoint-docs_ops_README.md.txt\", fmt.Sprintf", "root, \"docs/ops/README.md\", fmt.Sprintf")
		prior = strings.ReplaceAll(prior, "snapshot := frozen + \"entrypoint-\" + strings.ReplaceAll(path, \"/\", \"_\") + \".txt\"\n\t\t\tlines := strings.Split(readOwnedDoc(t, root, snapshot), \"\\n\")", "lines := strings.Split(readOwnedDoc(t, root, path), \"\\n\")")
		sum := sha256.Sum256([]byte(strings.TrimSpace(prior)))
		if hex.EncodeToString(sum[:]) != "7e37bf3c2feef91868eacabe0936528d92f7aebda56f3c13f34b024980957703" {
			t.Fatal("follow-up 22 assertions changed beyond frozen-input redirects")
		}
		rows := regexp.MustCompile(`(?m)^\| D-[0-9]{3} \|.*$`)
		old := rows.FindAllString(readOwnedDoc(t, root, archive+"register22.md"), -1)
		current := rows.FindAllString(readOwnedDoc(t, root, register), -1)
		if len(current) != len(old)+2 || !slices.Equal(current[:len(old)], old) {
			t.Fatal("earlier decisions changed; only D-136 / D-137 may be added")
		}
	})
}

func TestHandover_Followup23LiveDocumentIntegrity(t *testing.T) {
	root := repoRoot(t)
	var manifest struct {
		Followup struct {
			Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
		} `json:"handover_follow_up_23"`
	}
	if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Followup.Documents) != 15 {
		t.Fatalf("live document inventory %d", len(manifest.Followup.Documents))
	}
	for _, p := range manifest.Followup.Documents {
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, p.Path)))
		if hex.EncodeToString(sum[:]) != p.SHA256 {
			t.Errorf("stale live digest: %s", p.Path)
		}
	}
}

func TestHandover_Followup24Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "5bd751026073444087b32ac2a7115f8566104f8b"
	const fixture = "tests/handover/testdata/"
	const archive = fixture + "history/follow-up-24/"
	const frozen = fixture + "history/follow-up-25/"
	const mainPin = fixture + "followup24-main.json"
	const doc = "docs/handover/follow-up-24.md"
	const register = frozen + "register24.md"
	readJSON := func(t *testing.T, path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	checkHash := func(t *testing.T, path, want string) {
		t.Helper()
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
		if hex.EncodeToString(sum[:]) != want {
			t.Errorf("immutable file drift: %s", path)
		}
	}
	type claim struct {
		path   string
		tokens []string
	}
	for _, tc := range []struct {
		name   string
		claims []claim
	}{
		{"secops14_offline_interval_periods_no_live_promotion", []claim{
			{mainPin, []string{"done (#363)", "usagePeriodForStart", "44 one-second intervals", "2026→2027 year boundary", "per retained period", "Product ingestion unchanged", "No live suite ran; recorded rows unchanged"}},
			{archive + "secops14-handoff.md", []string{"their own interval start", "restore_test.go", "3,000 bytes", "2026-11-01T00:00:05Z", "2026-11-15T12:00:00Z", "2027-01-01T00:00:00Z", "No live suite ran", "full AT29 / AT31 remain blocked"}},
			{"tests/e2e/at/secops/usage_period_test.go", []string{"TestMTR03_UsagePeriodOverlapsInterval"}},
			{doc, []string{"usagePeriodForStart", "per retained period", "Product ingestion unchanged; no live suite ran; recorded rows unchanged", "SECOPS-21 closed, AT30 pass"}},
		}},
		{"handover23_previous_cut_and_month_boundary", []claim{
			{mainPin, []string{"done (#364)", "#362 at the fixed cutoff bc011b6f", "16 subtests / 6 exact roadmap rows / 18 immutable snapshots / 421", "month-boundary incident", "D-136 and D-137 as provisional", "#363 merged after the cutoff"}},
			{"tests/handover/handover_test.go", []string{"TestHandover_Followup23Claims"}},
			{doc, []string{"#357–#362 at bc011b6f", "16 subtests / 6 rows / 18 snapshots /\n421 file checks", "never pair a calendar literal with\nthe wall clock", "only those input redirects changed"}},
		}},
		{"netlab22_opt_in_and_two_windows", []claim{
			{mainPin, []string{"done (#365)", "132 files", "Opt-in TEST ONLY", "two comparison windows", "qualified R-B2 agent leg never ran", "3 requests"}},
			{archive + "netlab22-note.md", []string{"WithCommandLifecycleForTest(ctx)", "original `cmd.Run`", "TEST ONLY trace", "90-minute qualification window", "Requests 1\nand 2 belong to agent-core follow-up 31", "neither R-B2 population\nhas a qualified completed result"}},
			{"node/enforce/command_lifecycle_test.go", []string{"TestD133_CommandLifecycleOutcomes", "TestD133_CommandLifecycleDisabledAndDetached", "TestD133_AT02_CommandLifecycleJoinsApplyAndRollback"}},
			{doc, []string{"WithCommandLifecycleForTest", "node/enforce/command_lifecycle.go", "WithApplyCaptureForTest", "node/enforce/apply_capture.go", "off unless a test", "two 90-minute windows", "qualified second B agent leg", "Requests **1–2 → agent-core 31**", "docs/poc/G2-followup22-resume-evidence/"}},
		}},
		{"D138_provisional_acceptance_qualified_pool_not_envelope", []claim{
			{archive + "netlab22-acceptance.md", []string{"Orchestrator decision D-138 (provisional)", "one context lookup", "no B-only failure in either window", "TestLab04_RestartKeepsVerifiedLease", "4 on A and 4 on B", "interleave with A across windows", "two-leg-envelope rule is withdrawn as underpowered", "pool qualified legs per arm across windows"}},
			{register, []string{"| D-138 |", "03:0x EST, 2026-11-01, orchestrator — provisional", "D-133 itself unchanged", "two-leg envelope rule is withdrawn as underpowered", "Pool every qualified leg per arm across windows", "only when it repeats in direction in each window and the code can explain it", "Owner overturn option"}},
			{doc, []string{"D-133 itself is unchanged", "4 on A and 4 on B", "pools every qualified leg per arm across windows", "only when it repeats in direction in each window and the code\ncan explain it", "Excluded legs stay excluded"}},
		}},
		{"netlab_D135_assessment_losses_never_ceiling", []claim{
			{archive + "netlab22-note.md", []string{"no previous Model or previous-generation restoration", "does **not** claim", "infallible cleanup", "312.900556 /\n323.540915 / 319.004140 ms", "185 losses", "167 outside", "424 ingresses and zero egresses"}},
			{doc, []string{"no infallible cleanup", "312.900556 / 323.540915 /\n319.004140 ms", "185 admitted losses", "167 outside guards", "zero denied-source egress", "not an outage ceiling"}},
		}},
		{"usage7_archive_period_authorities_and_November", []claim{
			{mainPin, []string{"done (#366)", "71b1a160", "19 files", "summary's returned period", "each record's period from its own interval start", "held events", "b8a6e9e1", "AT14–AT20 pass", "31,103 bytes", "41,417 bytes", "2,580 bytes", "0 requests"}},
			{archive + "usage7-handoff.md", []string{"own interval start", "summary's returned period", "immutable\nsnapshot", "2026-13", "2026-12-31 → 2027-01-01", "not §19's DNS AT20"}},
			{"tests/e2e/at/usage/period_boundaries_test.go", []string{"TestAT15AT17AT20_LivePeriodsAtUTCBoundaries", "TestAT17_SummaryPeriodRejectsInvalidMonthAndBounds", "TestAT20_HeldPeriodUsesMemberTrafficAcrossMonths"}},
			{doc, []string{"history/follow-up-6/71b1a160/", "summary's returned period", "immutable snapshot", "b8a6e9e1: AT14–AT20 pass", "not a live midnight experiment", "0 requests"}},
		}},
		{"devstack23_routed_D137_scope", []claim{
			{mainPin, []string{"done (#367)", "separate native router hop", "one PTB", "learns MTU 1280", "DF=0", "fragmented in transit into 68", "3 requests"}},
			{archive + "devstack23-native.md", []string{"64,064 / 64,064 | 1 ms", "learned MTU 1280", "all **DF=false**", "68", "not proof of IPv4 DF/ICMP discovery", "Historical F8-02 remains failed"}},
			{"tests/e2e/followup23_live_test.go", []string{"TestE2E_Followup23Native", "TestAT09_RouterSlotsDoNotOverlap"}},
			{doc, []string{"PrepareNativeRouter1280", "NativeRouter1280.Select", "StartCapture", "tests/e2e/native_underlay.go", "IPv6 passes by PTB with learned\nPMTU 1280", "IPv4 passes by DF=0\nfragmentation in transit", "Members adoption is pending"}},
			{register, []string{"| D-137 |", "**provisional**", "IPv6 PTB with learned PMTU 1280", "IPv4 DF=0 fragmentation in transit", "never a pass", "owner hardening option"}},
		}},
		{"F1101_proxy_trace_and_one_SYN_second_data_point", []claim{
			{archive + "devstack23-native.md", []string{"NetBird **0.82.0**", "RedirectAs", "`::1`", "no resulting member type-4", "Kernel input rejection and\nhandshake acceptance are not directly instrumented", "No kernel drop reason"}},
			{archive + "members13-observations.md", []string{"first selection succeeds with one SYN", "Type-4 transport data appears before a loopback-source IPv6 response", "neither historical cause nor a fix is inferred"}},
			{doc, []string{"F11-01 traced to the pinned NetBird 0.82.0 client's loopback proxy", "`::1`-source handshake response", "Devstack 24 is assigned to locate the drop", "second data point,\n  no cause or repair inferred"}},
			{register, []string{"Members F11-01", "traced to loopback proxy, drop not located", "Devstack 24 locating the drop; no result in this cut"}},
		}},
		{"devstack_IDNA_F1204_and_retained_parser_failure", []claim{
			{archive + "devstack23-dns.md", []string{"Positive IDNA wire fixture", "xn--bcher-kva.example.invalid", "IDNA A UDP", "IDNA A TCP", "does **not** reproduce", "unattributed / unproven", "without rewriting the original record or its failed verdict", "StartNativeDNSCapture"}},
			{"tests/e2e/followup23_live_test.go", []string{"TestE2E_Followup23DNS"}},
			{doc, []string{"positive IDNA", "F12-04 not reproduced", "Members must instrument\nits original query next", "a26d9b0f: 14 / 0 / 3", "9bb2098e: 11 / 0 / 1", "docs/e2e/followup23-captures.sha256", "3 requests"}},
		}},
		{"members13_AT02_brackets_no_failed_Apply", []claim{
			{mainPin, []string{"done (#368)", "52 files", "4182dd3d", "AT03 and AT08 pass, AT02 and AT04 fail, six blocked", "616 / 7 / 3", "all 61 baseline calls", "21 sibling losses", "219.395 ms", "95 successful Applies and none failed", "10 requests"}},
			{archive + "members13-observations.md", []string{"other 58", "nine losses", "14.828914 ms", "never summed as unique losses", "no-loss row stays FAIL"}},
			{"tests/e2e/at/members/evidence_test.go", []string{"TestD098_Followup12Archive"}},
			{"tests/e2e/at/members/apply_attribution_checks_test.go", []string{"TestAT02_F13D136HistoricalPolicyMissingCountsLosses"}},
			{doc, []string{"history/follow-up-12/", "2 pass (AT03, AT08) / 6 blocked / 2 fail", "61 baseline calls", "21 sibling losses counted", "219.395 ms with nine losses", "95 Applies succeeded, none\n  failed", "F11-03 stays blocked", "10 requests"}},
		}},
		{"members13_F1301_parser_exit_zero_cause_unproven", []claim{
			{mainPin, []string{"F13-01", "two `nft_parse` failures with exit 0", "cause unproven"}},
			{archive + "members13-observations.md", []string{"two separate failed persistent-withdrawal checks", "command exit 0", "Raw nft output/parser errors were not retained", "cause is unproven"}},
			{archive + "members13-routing.md", []string{"F13-01 → devstack follow-up 24 retains the raw bytes and parser error", "tests/e2e/internal/counterread", "only if that evidence shows a defect"}},
			{doc, []string{"AT04 fails on new F13-01", "command exit 0", "Devstack 24 is assigned to retain both", "Both failed rows stay failed"}},
			{register, []string{"| Members F13-01 | OPEN — cause unproven", "No result imported"}},
		}},
		{"members13_stream_qualification_quiet_window_and_AT09", []claim{
			{archive + "members13-observations.md", []string{"22 readiness checks", "12 monitor stream checks", "2,455,378 bytes", "F12-03 quiet window passes", "Historical TTL 2", "PMTU black-hole limitation rows, never PASS", "Reactive PTB remains pending devstack 23"}},
			{"tests/e2e/at/members/stream_qualification_test.go", []string{"TestAT04_F13MonitorPastDiagnosticLimit"}},
			{"tests/e2e/at/members/negative_window_test.go", []string{"TestAT20_F13NegativeWindowReservesTTLPrecision"}},
			{doc, []string{"F12-02's streamed reader live-qualified", "F12-03's quiet window passes", "known-PMTU 1280 passes", "limitation rows, never PASS", "historical row stays failed"}},
			{register, []string{"| Members F12-02 | LIVE-QUALIFIED — historical failures stay failed", "| Members F12-03 | QUIET WINDOW PASSES — historical row stays failed", "| Members F12-04 | OPEN — not reproduced by #367"}},
		}},
		{"agent31_D136_correction_and_F1201_unproven", []claim{
			{mainPin, []string{"done (#369)", "verified deny-all candidate applies only when an Apply error is retained", "installs the baseline and ACKs, renews, then Applies, verifies and heartbeats", "no finite outage ceiling"}},
			{archive + "agent31-observations.md", []string{"inside\n   `if e.applyError != nil`", "ordinary\n   successful stale-lease path above performs baseline calls", "There is no finite recovery/outage ceiling", "11→13; stale-lease path consistent with code; no failed Apply", "cause_status=unproven"}},
			{"node/agent/followup31_test.go", []string{"TestF31_StaleGenerationBaselinePrecedesRenewalWithoutCandidateApply"}},
			{doc, []string{"04:3x EST, 2026-11-01", "verified deny-all candidate is\nused only when an Apply error is retained", "node-wide baseline and ACKs, renews, then Applies, verifies and\nheartbeats", "No finite\noutage ceiling", "D-136's fail-closed reading and D-135's bracket rule remain"}},
			{register, []string{"| D-136 |", "provisional — to be confirmed together with D-135", "policy_missing baseline during generation 11→13; stale-lease path consistent with code; no failed Apply", "cause_status=unproven"}},
		}},
		{"agent31_80_transitions_two_distributions_every_loss", []claim{
			{archive + "agent31-observations.md", []string{"177.901491 | 192.802074 | 201.859702 | 201.859702", "173.766526 | 187.813686 | 195.601257 | 195.601257", "76 inside\ncompleted guards, 436 outside", "66 inside completed guards / 435 outside", "80 transitions and all 1,013 sibling losses", "not pooled into a synthetic bound", "not an A,B,A,B comparison"}},
			{doc, []string{"80 desired-first generation changes", "177.901491 | 192.802074 | 201.859702 | 201.859702", "173.766526 | 187.813686 | 195.601257 | 195.601257", "512 (76 / 436)", "501 (66 / 435)", "1,013 losses", "not A/B, not pooled into a bound"}},
		}},
		{"TestLab04_cancellation_cause_bounded_local_completion", []claim{
			{archive + "agent31-observations.md", []string{"25 passes /\n5 failures", "3,599.377190–3,599.448139 s", "cancellation during Apply reaches its rollback", "control retains the original caller cancellation/budget", "at most ten seconds", "genuine injected Apply/tc\nfailure still withdraws/quarantines", "30/30 foreground repetitions", "requires and records confirmed graceful exit", "40.839979 ms", "not a measured attribution of scheduler\ncontention or proof of the exact historical cancellation point"}},
			{"node/agent/followup31_restart_test.go", []string{"TestF31_AT13_CancellationAfterLiveReadKeepsRestartAuthorization", "TestF31_AT13_CancellationDuringLocalWorkKeepsRestartAuthorization", "TestF31_AT13_CancellationDoesNotMaskRealApplyFailure"}},
			{"tests/netlab/agent/scenario_test.go", []string{"TestLab04_RestartKeepsVerifiedLease"}},
			{doc, []string{"failed 5 of 30", "3,599 s from expiry", "caller cancellation after SIGTERM manufactured", "bounded\nlocal work, at most 10 s", "control I/O stays cancellable", "Real Apply/tc\nfailures still fail closed", "recorded\ngraceful exit", "Historical failed rows stay failed"}},
			{frozen + "entrypoint-docs_handover_operator-guide.md.txt", []string{"up to **10 s**", "while control I/O stops", "Real failures still fail closed", "recorded graceful\nexit"}},
		}},
		{"agent31_TEST_ONLY_caller_and_review_only_design", []claim{
			{mainPin, []string{"only under `TestMode` and `TestTimingPath`", "intersection serving, for the r18 review only"}},
			{archive + "agent31-observations.md", []string{"No live timing threshold failed", "conditional D-133 attribution run was\nnot triggered"}},
			{archive + "agent31-recommendation.md", []string{"Recommend **(b) for adversarial design review**", "retaining **(c) in\nproduction**", "proposal, not approval"}},
			{"node/agent/followup31_test.go", []string{"TestF31_CommandLifecycleRequiresBothTimingGates"}},
			{doc, []string{"TestMode and TestTimingPath", "intersection serving for r18 review only", "generation-serving behaviour is unchanged", "no D-133 run"}},
		}},
		{"findings_unchanged_boundaries_owner_decisions_and_handoffs", []claim{
			{register, []string{"| Install-overrun cause | OPEN — unproven", "| F9-INFLIGHT | OPEN", "| F18-01 | INTENDED D-125 NEGATIVE", "| ISO-F6-03 | OPEN", "writer unattributed", "Dependabot #22 open for owner disposition", "blocked_response_ttl owner value", "Every historical failure stays failed", "D-135 + D-136 confirmation", "D-137 hardening option", "D-138 overturn option"}},
			{doc, []string{"D-135, D-136, D-137 and D-138 stay\nprovisional", "Owner decision 1: confirm D-135 + D-136 together", "Owner decision 2: D-137 hardening option", "Owner decision 3: D-138 overturn option", "Already assigned, not new requests (2026-11-01)", "tests/e2e/internal/counterread/read.go", "node/enforce/parse_nft.go", "Members next follow-up", "StartNativeDNSCapture", "No outside-owned pin needed changing", "No new\nproduction value"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.claims {
				requireHandoverClaimTokens(t, root, c.path, c.tokens...)
			}
		})
	}
	t.Run("cutoff_seven_rows_entrypoints_and_no_post_cut_results", func(t *testing.T) {
		var pin struct {
			Commit string
			Cutoff int `json:"cutoff_pr"`
			Rows   []string
			After  []int    `json:"merged_after_cutoff"`
			Absent []string `json:"absent_at_start"`
		}
		readJSON(t, mainPin, &pin)
		if pin.Commit != cut || pin.Cutoff != 369 || len(pin.Rows) != 7 || len(pin.After) != 0 || !slices.Equal(pin.Absent, []string{"grok review r18", "w3-devstack-e2e follow-up 24"}) {
			t.Fatalf("wrong cutoff: %+v", pin)
		}
		seen := map[int]bool{}
		for _, row := range pin.Rows {
			requireHandoverClaimTokens(t, root, "docs/ROADMAP.md", row)
			for pr := 363; pr <= 369; pr++ {
				if strings.HasSuffix(row, fmt.Sprintf("| done (#%d) |", pr)) {
					if seen[pr] {
						t.Fatalf("duplicate PR #%d", pr)
					}
					seen[pr] = true
				}
			}
		}
		if len(seen) != 7 {
			t.Fatalf("wrong PR set: %v", seen)
		}
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			snapshot := frozen + "entrypoint-" + strings.ReplaceAll(path, "/", "_") + ".txt"
			lines := strings.Split(readOwnedDoc(t, root, snapshot), "\n")
			if len(lines) < 4 || !strings.Contains(lines[2], "Current cut") || !strings.Contains(lines[2], "follow-up-24.md") || !strings.Contains(lines[2], "#369") || !strings.Contains(lines[3], "#362") {
				t.Errorf("stale entrypoint: %s", path)
			}
		}
		for pr := 363; pr <= 369; pr++ {
			requireHandoverClaimTokens(t, root, frozen+"entrypoint-docs_ops_README.md.txt", fmt.Sprintf("#%d", pr))
		}
		requireHandoverClaimTokens(t, root, doc, cut, "#363–#369", "Nothing was merged after the cutoff at task start", "r18 (agent-core 31's design note and shutdown change)", "no results from either enter this cut", "Handover ran no new live acceptance", "D-098 / D-109 / D-132")
	})
	t.Run("owner_archives_and_run_captures_357_files", func(t *testing.T) {
		for path, digest := range map[string]string{
			"docs/poc/G2-followup21-evidence/archive/archive.json":      "04ef9244ba0d59a5624b990ab1da0da10d3c42f1835ad26795b909db307957d9",
			"docs/e2e/at/usage/history/follow-up-6/71b1a160/SHA256SUMS": "a8f572f1f2f819ac83e17a323f938fd04d5e98f58e62b5c7cd495370763bc994",
			"docs/e2e/at/members/history/follow-up-12/SHA256SUMS":       "ee01fe65afa4887673a59749ab85fb1075283b91df65154ffd209d83ad461c3a",
		} {
			checkHash(t, path, digest)
		}
		total := 0
		for _, item := range []struct {
			base  string
			count int
		}{
			{"docs/e2e/at/usage/history/follow-up-6/71b1a160/", 19},
			{"docs/e2e/at/members/history/follow-up-12/", 52},
		} {
			lines := strings.Split(strings.TrimSpace(readOwnedDoc(t, root, item.base+"SHA256SUMS")), "\n")
			if len(lines) != item.count {
				t.Fatalf("%s inventory %d", item.base, len(lines))
			}
			seen := map[string]bool{}
			for _, line := range lines {
				f := strings.Fields(line)
				if len(f) != 2 || !filepath.IsLocal(f[1]) || seen[f[1]] {
					t.Fatalf("invalid archive pin: %q", line)
				}
				seen[f[1]] = true
				checkHash(t, item.base+f[1], f[0])
				total++
			}
		}
		const netlab = "docs/poc/G2-followup21-evidence/archive/"
		var inventory struct {
			Source string `json:"source_commit"`
			Files  []struct {
				Path, SHA256 string
				Bytes        int
			}
		}
		readJSON(t, netlab+"archive.json", &inventory)
		if inventory.Source != "8f74bc8b884e3a9a69924f791c7938e3dea09908" || len(inventory.Files) != 132 {
			t.Fatal("wrong netlab archive")
		}
		seen := map[string]bool{}
		for _, f := range inventory.Files {
			if !filepath.IsLocal(f.Path) || (!strings.HasPrefix(f.Path, "retained/") && f.Path != "notes/G2-followup21-2026-10-31.md") || seen[f.Path] {
				t.Fatalf("invalid netlab pin: %s", f.Path)
			}
			seen[f.Path] = true
			checkHash(t, netlab+f.Path, f.SHA256)
			if len(readOwnedDoc(t, root, netlab+f.Path)) != f.Bytes {
				t.Errorf("netlab size drift: %s", f.Path)
			}
			total++
		}
		lines := strings.Split(strings.TrimSpace(readOwnedDoc(t, root, archive+"devstack23-captures.sha256")), "\n")
		if len(lines) != 154 {
			t.Fatalf("devstack inventory %d", len(lines))
		}
		uuidPath := regexp.MustCompile(`^docs/e2e/runs/2026-11-01/followup23/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/`)
		seen = map[string]bool{}
		for _, line := range lines {
			f := strings.Fields(line)
			if len(f) != 2 || !filepath.IsLocal(f[1]) || seen[f[1]] {
				t.Fatalf("invalid run pin: %q", line)
			}
			seen[f[1]] = true
			path := f[1]
			switch path {
			case "docs/e2e/runs/2026-11-01/followup23/README.md":
				path = archive + "devstack23.md"
			case "docs/e2e/runs/2026-11-01/followup23/gates.md":
				path = archive + "devstack23-gates.md"
			default:
				if !uuidPath.MatchString(path) {
					t.Fatalf("unscoped run pin: %s", path)
				}
			}
			checkHash(t, path, f[0])
			total++
		}
		if total != 357 {
			t.Fatalf("archive total %d", total)
		}
	})
	t.Run("D098_D109_D132_twenty_one_snapshots_and_rejected_living_roots", func(t *testing.T) {
		var pins []struct {
			Path, Source, SHA256 string
			Commit               string `json:"source_commit"`
			Kind                 string `json:"source_kind"`
			SourceSHA            string `json:"source_sha256"`
			Offset, Length       int
		}
		readJSON(t, fixture+"followup24-snapshots.json", &pins)
		if len(pins) != 21 {
			t.Fatalf("snapshot count %d", len(pins))
		}
		seen := map[string]bool{}
		for _, pin := range pins {
			if seen[pin.Path] || !filepath.IsLocal(pin.Path) || !strings.HasPrefix(pin.Path, archive) || len(pin.Commit) != 40 || len(pin.SourceSHA) != 64 || pin.Offset < 0 {
				t.Fatalf("invalid snapshot: %+v", pin)
			}
			seen[pin.Path] = true
			if strings.HasPrefix(pin.Source, "docs/e2e/at/") {
				t.Fatalf("living source: %s", pin.Source)
			}
			if pin.Kind == "file" {
				if pin.Commit != cut {
					t.Fatal("wrong snapshot source commit")
				}
			} else if pin.Kind != "commit_message" {
				t.Fatalf("unknown source kind: %s", pin.Kind)
			}
			checkHash(t, pin.Path, pin.SHA256)
			if len(readOwnedDoc(t, root, pin.Path)) != pin.Length {
				t.Errorf("snapshot size drift: %s", pin.Path)
			}
		}
		for _, path := range []string{
			"docs/e2e/at/members/F13-01-counter-parse.json", "docs/e2e/at/members/F13-AT02-recovery.json", "docs/e2e/at/members/F13-F12-02-stream-qualification.json",
			"docs/e2e/at/usage/AT14.json", "docs/e2e/at/usage/AT20.json", "docs/e2e/at/usage/RUN.md",
			"node/agent/engine.go", "node/enforce/command_lifecycle.go", "tests/e2e/native_underlay.go",
		} {
			if claimTokenAllowed(path, "mutable implementation or evidence token") {
				t.Errorf("mutable token pin allowed: %s", path)
			}
		}
		requireHandoverClaimTokens(t, root, doc, "Members 13's F13 roots", "usage 7's root records remain living evidence", "never by reading or copying those roots")
	})
	t.Run("followup23_assertions_and_prior_decisions_preserved", func(t *testing.T) {
		for path, want := range map[string]string{
			"docs/handover/follow-up-23.md":                     "647c9d4e23ed2f8d7964fab7118444c14843ae9a422f96b0a67cffa5bc8276d9",
			"tests/handover/testdata/followup23-main.json":      "e54a71a4b283b6213eb866b80eea0b1b1022bc3410add1e43d3b4f2badfbba2d",
			"tests/handover/testdata/followup23-snapshots.json": "561eb723fa77d5c3faa8bae1eb49817e714a5e0de304880c89e29087b58b378c",
		} {
			checkHash(t, path, want)
		}
		source := readOwnedDoc(t, root, "tests/handover/handover_test.go")
		start := strings.Index(source, "func TestHandover_Followup23Claims(")
		if start < 0 {
			t.Fatal("missing prior function")
		}
		end := strings.Index(source[start:], "\nfunc TestHandover_Followup23LiveDocumentIntegrity")
		if end < 0 {
			t.Fatal("missing prior function end")
		}
		prior := source[start : start+end]
		prior = strings.ReplaceAll(prior, "\tconst frozen = fixture + \"history/follow-up-24/\"\n", "")
		prior = strings.ReplaceAll(prior, "const register = frozen + \"register23.md\"", "const register = \"docs/handover/README.md\"")
		prior = strings.ReplaceAll(prior, "{frozen + \"entrypoint-docs_handover_operator-guide.md.txt\", []string{", "{\"docs/handover/operator-guide.md\", []string{")
		prior = strings.ReplaceAll(prior, "root, frozen+\"entrypoint-docs_ops_README.md.txt\", fmt.Sprintf", "root, \"docs/ops/README.md\", fmt.Sprintf")
		prior = strings.ReplaceAll(prior, "snapshot := frozen + \"entrypoint-\" + strings.ReplaceAll(path, \"/\", \"_\") + \".txt\"\n\t\t\tlines := strings.Split(readOwnedDoc(t, root, snapshot), \"\\n\")", "lines := strings.Split(readOwnedDoc(t, root, path), \"\\n\")")
		sum := sha256.Sum256([]byte(strings.TrimSpace(prior)))
		if hex.EncodeToString(sum[:]) != "4ee710548a07604c4fdd93a8e9603bce473be39a23b9eca861683099643597b9" {
			t.Fatal("follow-up 23 assertions changed beyond frozen-input redirects")
		}
		rows := regexp.MustCompile(`(?m)^\| D-[0-9]{3} \|.*$`)
		old := rows.FindAllString(readOwnedDoc(t, root, archive+"register23.md"), -1)
		current := rows.FindAllString(readOwnedDoc(t, root, register), -1)
		if len(old) < 2 || len(current) != len(old)+1 || !slices.Equal(current[:len(old)-2], old[:len(old)-2]) {
			t.Fatal("only D-136 / D-137 correction and D-138 addition allowed")
		}
	})
	t.Log("follow-up 24: 21 subtests / 7 exact roadmap rows / 21 immutable snapshots / 357 archive and run file checks")
}

// Current owned-document integrity is intentionally separate from immutable claims.
func TestHandover_Followup24LiveDocumentIntegrity(t *testing.T) {
	root := repoRoot(t)
	var manifest struct {
		Followup struct {
			Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
		} `json:"handover_follow_up_24"`
	}
	if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Followup.Documents) != 16 {
		t.Fatalf("live document inventory %d", len(manifest.Followup.Documents))
	}
	for _, p := range manifest.Followup.Documents {
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, p.Path)))
		if hex.EncodeToString(sum[:]) != p.SHA256 {
			t.Errorf("stale live digest: %s", p.Path)
		}
	}
}

// Follow-up 25 pins the #373 cut; mutable AT roots are never evidence inputs.
func TestHandover_Followup25Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "9cd7a0531819d17163d81c524bc5539bb3edc014"
	const fixture = "tests/handover/testdata/"
	const archive = fixture + "history/follow-up-25/"
	const doc = "docs/handover/follow-up-25.md"
	const archive26 = fixture + "history/follow-up-26/"
	const register = archive26 + "register25.md"
	const mainPin = fixture + "followup25-main.json"
	readJSON := func(t *testing.T, path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	checkHash := func(t *testing.T, path, want string) {
		t.Helper()
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
		if hex.EncodeToString(sum[:]) != want {
			t.Errorf("immutable file drift: %s", path)
		}
	}
	type claim struct {
		path   string
		tokens []string
	}
	for _, tc := range []struct {
		name   string
		claims []claim
	}{
		{"devstack24_located_boundary_not_proven_kernel_cause", []claim{
			{"tests/handover/testdata/followup25-main.json", []string{"done (#370)", "cacad79c", "Ip6InHdrErrors 0→1", "ingress 1, prerouting 0, input 0", "no drop tracepoint captured"}},
			{"tests/handover/testdata/history/follow-up-25/devstack24-analysis.md", []string{"after member netdev ingress and before\nIPv6 prerouting", "Ip6InHdrErrors 0→1", "not a captured kernel drop tracepoint", "not an observed nft policy drop"}},
			{"docs/handover/follow-up-25.md", []string{"dropped before IPv6 prerouting", "Ip6InHdrErrors 0→1", "consistent with kernel receive\nvalidation", "no\ndrop tracepoint"}},
		}},
		{"F1101_success_timeout_and_owner_options", []claim{
			{"tests/handover/testdata/followup25-main.json", []string{"one SYN in 9 ms", "did not reproduce in four complete attempts", "d556673b", "kept failed and unattributed", "NetBird 0.82.0 stays pinned"}},
			{"tests/handover/testdata/history/follow-up-25/devstack24-analysis.md", []string{"Five SYNs, 5004 ms timeout", "four completed attempts", "NB_WG_KERNEL_DISABLED=true", "conflicts with the agreed kernel-WireGuard baseline", "Not an established IPv6 fix", "none applied"}},
			{"docs/handover/follow-up-25.md", []string{"succeeded with one SYN in 9 ms", "did not reproduce in four completed\nattempts", "failed and unattributed", "upstream\nNetBird fix and a qualified pin", "userspace WireGuard against the agreed\nkernel baseline", "IPv4 preference", "NetBird 0.82.0 stays pinned"}},
		}},
		{"members_F1301_private_raw_bytes_cause_unproven", []claim{
			{"tests/handover/testdata/followup25-main.json", []string{"F13-01", "exit-0 parse rejection", "raw bytes and the exact parser error", "private mode-0600", "public evidence carries digest"}},
			{"tests/handover/testdata/history/follow-up-25/devstack24-analysis.md", []string{"exact failed command bytes and parser text", "Historical F13-01 and F10-02 are still\nunattributed", "no new parser defect is inferred"}},
			{"tests/e2e/internal/counterread/read_test.go", []string{"TestAT04_ParseFailureEvidence", "TestAT04_ParseRetentionFailureAndLines"}},
			{"tests/e2e/followup24_live_test.go", []string{"TestE2E_Followup24Native"}},
			{"docs/handover/follow-up-25.md", []string{"Members F13-01", "exit-0 parse rejection", "raw bytes and\nexact parser error privately", "mode-0600", "Public evidence carries a **digest**", "cause unproven", "failed AT04 rows stay failed"}},
		}},
		{"handover24_prior_cut_remains_frozen", []claim{
			{"tests/handover/testdata/followup25-main.json", []string{"done (#371)", "#369 at the fixed cutoff 5bd75102", "21 subtests / 7 exact roadmap rows / 21 immutable snapshots / 357", "every earlier claims test stays true"}},
			{"docs/handover/follow-up-25.md", []string{"#363–#369 at 5bd75102", "21 subtests / 7 rows / 21 snapshots /\n357 file checks", "only their input redirects changed"}},
			{"tests/handover/handover_test.go", []string{"TestHandover_Followup24Claims"}},
		}},
		{"updates13_results_and_distinct_F1301_exit_gap", []claim{
			{"tests/handover/testdata/followup25-main.json", []string{"done (#372)", "2babf9b6", "759d244a", "#369's tree", "AT23–AT27 PASS 5 / 5", "installer 27 / 27", "All 11 updater agent stops", "stored lease kept", "AT23's first exit unobserved"}},
			{"tests/handover/testdata/history/follow-up-25/updates13-handoff.md", []string{"Five stopped agents", "normal exit status 0", "five calls find the agent already stopped", "AT23's first", "unobserved", "No claim that every agent exited gracefully", "run 2babf9b6 never captured stop-command durations"}},
			{"tests/handover/testdata/history/follow-up-25/updates13-routing.md", []string{"Updates F13-01 (this lane's numbering; not members' F13-01)", "devstack follow-up 25", "TEST ONLY", "SIGKILL", "row stays unobserved"}},
			{"tests/e2e/at/updates/stopobserve/main_test.go", []string{"TestUPD05_StopObservationDistinguishesExitKillAndReap"}},
			{"docs/handover/follow-up-25.md", []string{"AT23–AT27 5 / 5", "27 / 27", "11 updater agent stop commands returned success", "the stored lease kept", "AT23's first agent\nexit is unobserved", "updates F13-01", "distinct from **members F13-01**", "stage brackets cannot replace them"}},
		}},
		{"updates13_archive_and_retained_startup", []claim{
			{"tests/handover/testdata/followup25-main.json", []string{"44 follow-up 12 root files", "history/follow-up-12/", "history/follow-up-13-startup/", "configurable port slot"}},
			{"tests/handover/testdata/history/follow-up-25/updates13-handoff.md", []string{"make exit 2 in 24.395 s", "No regular test or installer test started", "kept default\n42000", "slot 54000", "no failed AT predicate was retried"}},
			{"docs/e2e/at/updates/history/follow-up-13-startup/startup.json", []string{"\"live_target_exit\": 2", "\"regular_test_started\": false", "\"installer_test_started\": false"}},
			{"tests/e2e/at/updates/archive_test.go", []string{"TestD098_Followup12ArchivePins"}},
			{"docs/handover/follow-up-25.md", []string{"44 root files", "pre-test port collision", "configurable port slot", "default 42000", "54000", "No failed AT predicate was retried"}},
		}},
		{"isolation12_new_AT10_block_not_prior_pass", []claim{
			{"tests/handover/testdata/followup25-main.json", []string{"done (#373)", "edd986c4", "103 follow-up 11 root files", "1c25ee44", "AT12 PASS 10 / 0 / 0", "AT10 BLOCKED 8 / 1 / 0", "assertion 6 is blocked", "kernel denial held", "generation-30 TC_FOREIGN_FILTER report", "AT11 2 / 1 / 0 and AT13 6 / 1 / 0"}},
			{"tests/handover/testdata/history/follow-up-25/isolation12-handoff.md", []string{"AT10 passed 9 / 0 / 0 in run edd986c4 (#360)", "No generation-30 TC_FOREIGN_FILTER report", "after removal", "never upgraded to a pass"}},
			{"tests/handover/testdata/history/follow-up-25/isolation12-routing.md", []string{"agent-core follow-up 32", "missing sixth assertion is not substituted", "operator resources"}},
			{"tests/e2e/at/isolation/producer_timing_test.go", []string{"TestD098_Followup11ArchivePins"}},
			{"docs/handover/follow-up-25.md", []string{"PASS 10 / 0 / 0", "BLOCKED 8 / 1 / 0", "BLOCKED 2 / 1 / 0", "BLOCKED 6 / 1 / 0", "ISO-F6-01 assertion 6", "Kernel denial held", "no generation-30\nTC_FOREIGN_FILTER report reached central", "AT10's earlier pass (#360) is history; this run's\nblock stands"}},
		}},
		{"isolation12_commands_lossless_stream_and_missing_losses", []claim{
			{"tests/handover/testdata/followup25-main.json", []string{"per-Apply `commands`", "956 Applies, 53,754 commands", "seven failed pre-install Applies", "949 completed guards", "outer p99 49.8 ms", "losses not captured there, never zero", "lossless gzip"}},
			{"tests/handover/testdata/history/follow-up-25/isolation12-handoff.md", []string{"All original producer bytes remain in the lossless gzip", "cause unproven", "loss and independent egress there are unavailable, never zero"}},
			{"docs/handover/follow-up-25.md", []string{"per-Apply `commands` records", "956 Applies / 53,754\ncommands", "949 completed\nguards", "49.8 ms", "producer stream is a lossless gzip", "losses were not\ncaptured and must not be called zero"}},
		}},
		{"r18_five_findings_open_and_owner_sequence", []claim{
			{"docs/reviews/r18-generation-change-review.md", []string{"### M-1 (Medium)", "### M-2 (Medium)", "### M-3 (Medium)", "### L-1 (Low)", "### L-2 (Low)", "14.021469915s", "TimeoutStopSec set=false", "40 stale-lease steps", "sampled command resources", "sibling ceil from 1000000 to 5000000 bps"}},
			{"docs/handover/follow-up-25.md", []string{"r18 M-1", "r18 M-2", "r18 M-3", "r18 L-1", "r18 L-2", "control observation POST after SIGTERM", "shutdown stacks past the **10 s** bound", "unit has no `TimeoutStopSec`", "no cap or distinct alarm", "TEST ONLY command sample ignores the install deadline", "rule does not compare stricter member/node ceilings", "agent-core 32 (running)", "OPEN → netlab", "after agent-core 32"}},
			{"docs/reviews/README.md", []string{"all five findings remain OPEN", "installer sets the value after agent-core 32", "wire-nodes handles central's stale-hold reading after agent-core 32", "No proof files were adopted"}},
		}},
		{"r18_what_held_is_review_not_acceptance", []claim{
			{"docs/reviews/r18-generation-change-review.md", []string{"Reviewing model: `grok/grok-4.7`", "Date: 2026-11-01", "## What held", "## Not examined", "No Postgres database was opened", "The lab was not.", "The product still baselines."}},
			{"docs/handover/follow-up-25.md", []string{"What held is the review's statement, not an acceptance result", "grant forwarded on any reviewed stale path", "observation keys out of production", "torn checkpoints fail closed", "no live lab or PostgreSQL suite", "did not\nimplement intersection serving", "No review proof files are adopted here"}},
		}},
		{"decisions_no_addition_D136_correction_provisional", []claim{
			{register, []string{"No new decision since D-138. D-135, D-136, D-137 and D-138 stay provisional.", "04:3x EST", "verified deny-all candidate is used only when an Apply error is retained", "No finite outage ceiling", "D-135 + D-136 confirmation", "D-137 hardening option", "D-138 overturn option", "F11-01 repair direction"}},
			{"docs/handover/follow-up-25.md", []string{"No new decision since D-138. D-135, D-136, D-137 and D-138 stay provisional.", "04:3x EST, 2026-11-01 correction", "baseline→ACK→renewal→Apply→verify→heartbeat", "no finite outage ceiling"}},
		}},
		{"findings_every_historical_failure_retained", []claim{
			{register, []string{"| Members F11-01 | OPEN — drop located by #370", "| Members F13-01 | OPEN — raw bytes kept by #370; cause unproven", "| Updates F13-01 | OPEN — exit status unobserved", "| ISO-F6-01 assertion 6 | BLOCKED — observation gap", "Every other follow-up 24 finding is unchanged", "Every historical failure stays failed", "Install-overrun cause remains unproven", "F9-INFLIGHT", "F18-01", "ISO-F6-03 writer unattributed", "Dependabot #22 open for owner disposition", "blocked_response_ttl owner value"}},
			{"docs/handover/follow-up-25.md", []string{"Every other follow-up 24 finding is\nunchanged", "Every historical failure stays failed", "first-selection/userspace-WireGuard evidence"}},
		}},
		{"dated_owner_decisions_and_outside_scope_handoffs", []claim{
			{"docs/handover/follow-up-25.md", []string{"### Requests to orchestrator", "2026-11-01 — **Owner decision 1: D-135 + D-136 confirmation", "Owner decision 2: D-137 hardening option", "Owner decision 3: D-138 overturn option", "Owner decision 4: F11-01 repair direction", "Already assigned, not new requests (2026-11-01)", "node/agent/observe.go", "node/enforce/command_lifecycle.go", "deploy/node/systemd/fvpn-agent.service", "internal/nodes/observations.go", "internal/nodes/publication_evidence.go", "deploy/devstack/agent-service.sh", "deploy/devstack/entrypoint-node.sh", "Members 14", "These outside-owned items stop with their owners", "No outside-owned pin needed changing", "No new production value is selected"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.claims {
				requireHandoverClaimTokens(t, root, c.path, c.tokens...)
			}
		})
	}
	t.Run("cutoff_four_rows_and_seven_entrypoints", func(t *testing.T) {
		var pin struct {
			Commit string
			Cutoff int `json:"cutoff_pr"`
			Rows   []string
			After  []int    `json:"merged_after_cutoff"`
			Absent []string `json:"absent_at_start"`
		}
		readJSON(t, mainPin, &pin)
		if pin.Commit != cut || pin.Cutoff != 373 || len(pin.Rows) != 4 || len(pin.After) != 0 || !slices.Equal(pin.Absent, []string{"w2-agent-core follow-up 32", "w4-at-members follow-up 14", "w3-devstack-e2e follow-up 25"}) {
			t.Fatalf("wrong cutoff: %+v", pin)
		}
		seen := map[int]bool{}
		for _, row := range pin.Rows {
			requireHandoverClaimTokens(t, root, "docs/ROADMAP.md", row)
			for pr := 370; pr <= 373; pr++ {
				if strings.HasSuffix(row, fmt.Sprintf("| done (#%d) |", pr)) {
					if seen[pr] {
						t.Fatalf("duplicate PR #%d", pr)
					}
					seen[pr] = true
				}
			}
		}
		if len(seen) != 4 {
			t.Fatalf("wrong PR set: %v", seen)
		}
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			lines := strings.Split(readOwnedDoc(t, root, archive26+"entrypoint-"+strings.ReplaceAll(path, "/", "_")+".txt"), "\n")
			if len(lines) < 4 || !strings.Contains(lines[2], "Current cut") || !strings.Contains(lines[2], "follow-up-25.md") || !strings.Contains(lines[2], "#373") || !strings.Contains(lines[3], "#369") {
				t.Errorf("stale entrypoint: %s", path)
			}
		}
		for pr := 370; pr <= 373; pr++ {
			requireHandoverClaimTokens(t, root, "docs/ops/README.md", fmt.Sprintf("#%d", pr))
		}
		requireHandoverClaimTokens(t, root, doc, cut, "#370–#373", "Nothing was merged after the cutoff\nat task start", "Agent-core 32, members 14 and devstack 25 are running", "no results from those lanes enter this cut", "Handover ran no new live acceptance", "no new operator guidance", "D-098 / D-109 / D-132")
	})
	t.Run("owner_archives_and_devstack_captures_270_files", func(t *testing.T) {
		total := 0
		for _, tc := range []struct {
			manifest, base, digest string
			count                  int
		}{
			{"docs/e2e/at/updates/history/follow-up-12/SHA256SUMS", "docs/e2e/at/updates/history/follow-up-12/", "75f5cb80931eef16399517042cfaaa599906240ff32b466ecd78ed70d6caca18", 44},
			{"docs/e2e/at/updates/history/follow-up-13-startup/SHA256SUMS", "docs/e2e/at/updates/history/follow-up-13-startup/", "1a04b4e32b8b53cda2a447f5945c9736da346e1a3b3573b90d350335657e8bc8", 1},
			{archive + "devstack24-captures.sha256", "", "07fa0da2ee3ff1c5cfebb3b58dcfe49c55fe34334652454457299e21586bf286", 122},
		} {
			checkHash(t, tc.manifest, tc.digest)
			lines := strings.Split(strings.TrimSpace(readOwnedDoc(t, root, tc.manifest)), "\n")
			if len(lines) != tc.count {
				t.Fatalf("manifest count %s: %d", tc.manifest, len(lines))
			}
			seen := map[string]bool{}
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) != 2 || len(fields[0]) != 64 || !filepath.IsLocal(fields[1]) || seen[fields[1]] {
					t.Fatalf("bad manifest row: %s", line)
				}
				if tc.base == "" && !strings.HasPrefix(fields[1], "docs/e2e/runs/2026-11-01/followup24/") {
					t.Fatalf("non-run capture: %s", fields[1])
				}
				seen[fields[1]] = true
				checkHash(t, tc.base+fields[1], fields[0])
				total++
			}
		}
		const iso = "docs/e2e/at/isolation/history/follow-up-11/"
		checkHash(t, iso+"archive.json", "b0e057dccad21b7e69cb444c4204fdb607f8cb24fdaf08a9cb6ece89ba9a4ef7")
		var inventory struct {
			Run   string `json:"run_id"`
			Files []struct{ Path, SHA256 string }
		}
		readJSON(t, iso+"archive.json", &inventory)
		if inventory.Run != "edd986c4-16d2-4aab-b92f-baa19f8bd829" || len(inventory.Files) != 103 {
			t.Fatalf("isolation archive drift: %s / %d", inventory.Run, len(inventory.Files))
		}
		for _, file := range inventory.Files {
			if !filepath.IsLocal(file.Path) {
				t.Fatalf("invalid archive path: %s", file.Path)
			}
			checkHash(t, iso+file.Path, file.SHA256)
			total++
		}
		if total != 270 {
			t.Fatalf("archive/run files = %d", total)
		}
	})
	t.Run("D098_D109_D132_immutable_snapshots_and_exact_r18", func(t *testing.T) {
		var pins []struct {
			Path, Source, SHA256 string
			Commit               string `json:"source_commit"`
			Kind                 string `json:"source_kind"`
			SourceSHA            string `json:"source_sha256"`
			Offset, Length       int
		}
		readJSON(t, fixture+"followup25-snapshots.json", &pins)
		if len(pins) != 15 {
			t.Fatalf("snapshot count %d", len(pins))
		}
		seen := map[string]bool{}
		for _, pin := range pins {
			isReview := pin.Path == "docs/reviews/r18-generation-change-review.md"
			if seen[pin.Path] || !filepath.IsLocal(pin.Path) || (!strings.HasPrefix(pin.Path, archive) && !isReview) || len(pin.Commit) != 40 || len(pin.SourceSHA) != 64 || pin.Offset < 0 {
				t.Fatalf("invalid snapshot: %+v", pin)
			}
			seen[pin.Path] = true
			if strings.HasPrefix(pin.Source, "docs/e2e/at/") {
				t.Fatalf("living source: %s", pin.Source)
			}
			switch pin.Kind {
			case "file":
				if isReview {
					if pin.Commit != "1a65598fe6ebc5ba394284857ece59d0a572458b" || pin.Source != ".agent/REVIEW.md" || pin.Offset != 0 || pin.SourceSHA != pin.SHA256 {
						t.Fatal("r18 must be the whole original report")
					}
				} else if pin.Commit != cut {
					t.Fatal("wrong snapshot source commit")
				}
			case "commit_message":
				if pin.Source != "PR #372 merge message" && pin.Source != "PR #373 merge message" {
					t.Fatalf("wrong PR source: %s", pin.Source)
				}
			default:
				t.Fatalf("unknown snapshot kind: %s", pin.Kind)
			}
			checkHash(t, pin.Path, pin.SHA256)
			if len(readOwnedDoc(t, root, pin.Path)) != pin.Length {
				t.Errorf("snapshot size drift: %s", pin.Path)
			}
		}
		checkHash(t, "docs/reviews/r18-generation-change-review.md", "f266a0705292975f5416bda5a082a3f7e779f45e82cb9b9ce54843ce0cb8e33d")
		for _, path := range []string{"docs/e2e/at/updates/AT23.json", "docs/e2e/at/updates/agent-stops.json", "docs/e2e/at/isolation/AT10.json", "docs/e2e/at/isolation/iso-f6-01-observation-gap.json", "docs/e2e/at/isolation/producer-operations.jsonl.gz", "docs/e2e/RESPONSES.md", "node/agent/observe.go"} {
			if claimTokenAllowed(path, "mutable implementation or evidence token") {
				t.Errorf("mutable token pin allowed: %s", path)
			}
		}
		requireHandoverClaimTokens(t, root, doc, "Updates 13 and\nisolation 12 root records are living roots", "never reads or copies those\nroots")
	})
	t.Run("followup24_assertions_and_all_other_findings_preserved", func(t *testing.T) {
		source := readOwnedDoc(t, root, "tests/handover/handover_test.go")
		start := strings.Index(source, "func TestHandover_Followup24Claims(")
		if start < 0 {
			t.Fatal("missing prior function")
		}
		end := strings.Index(source[start:], "// Current owned-document integrity")
		if end < 0 {
			t.Fatal("missing prior function end")
		}
		prior := source[start : start+end]
		prior = strings.ReplaceAll(prior, "\tconst frozen = fixture + \"history/follow-up-25/\"\n", "")
		prior = strings.ReplaceAll(prior, "const register = frozen + \"register24.md\"", "const register = \"docs/handover/README.md\"")
		prior = strings.ReplaceAll(prior, "{frozen + \"entrypoint-docs_handover_operator-guide.md.txt\", []string{", "{\"docs/handover/operator-guide.md\", []string{")
		prior = strings.ReplaceAll(prior, "root, frozen+\"entrypoint-docs_ops_README.md.txt\", fmt.Sprintf", "root, \"docs/ops/README.md\", fmt.Sprintf")
		prior = strings.ReplaceAll(prior, "snapshot := frozen + \"entrypoint-\" + strings.ReplaceAll(path, \"/\", \"_\") + \".txt\"\n\t\t\tlines := strings.Split(readOwnedDoc(t, root, snapshot), \"\\n\")", "lines := strings.Split(readOwnedDoc(t, root, path), \"\\n\")")
		sum := sha256.Sum256([]byte(strings.TrimSpace(prior)))
		if hex.EncodeToString(sum[:]) != "2934c2e8a0d8acca11c367a966589c5e5264cd9aaa185b5d486942f46c7ad76c" {
			t.Fatal("follow-up 24 assertions changed beyond frozen-input redirects")
		}
		old := readOwnedDoc(t, root, archive+"register24.md")
		current := readOwnedDoc(t, root, register)
		decisions := regexp.MustCompile(`(?m)^\| D-[0-9]{3} \|.*$`)
		if !slices.Equal(decisions.FindAllString(old, -1), decisions.FindAllString(current, -1)) {
			t.Fatal("no new or changed decision since D-138")
		}
		oldFindings := strings.Split(strings.Split(old, "## Findings register at the follow-up 24 cutoff")[1], "\nEvery historical failure")[0]
		currentFindings := strings.Split(strings.Split(current, "## Findings register at the follow-up 25 cutoff")[1], "\nEvery historical failure")[0]
		count := 0
		for _, row := range strings.Split(oldFindings, "\n") {
			if !strings.HasPrefix(row, "| ") || strings.HasPrefix(row, "| Finding |") {
				continue
			}
			count++
			if strings.HasPrefix(row, "| Members F11-01 |") || strings.HasPrefix(row, "| Members F13-01 |") {
				continue
			}
			if !strings.Contains(currentFindings, row) {
				t.Errorf("previous finding changed: %s", row)
			}
		}
		if count != 30 {
			t.Fatalf("previous finding count %d", count)
		}
		added := 0
		for _, row := range strings.Split(currentFindings, "\n") {
			if strings.HasPrefix(row, "| ") && !strings.HasPrefix(row, "| Finding |") {
				added++
			}
		}
		if added != count+7 {
			t.Fatalf("expected seven new findings, got %d", added-count)
		}
	})

	t.Run("prior_artifacts_and_no_new_operator_guidance", func(t *testing.T) {
		checkHash(t, "docs/handover/follow-up-24.md", "6ad3d9738155536861a56c98fc120dfce5025293d1805d4b60676f4ad6804485")
		checkHash(t, "tests/handover/testdata/followup24-main.json", "606adc66f1d7924658567b8fc3da313ed6d39c364e58de3babbf720971af0f25")
		checkHash(t, "tests/handover/testdata/followup24-snapshots.json", "3d9a4b7372420707c1fbfd636fcff49270f27b7717d5b5b4aaf7deed6479aa73")
		checkHash(t, "tests/handover/testdata/followup25-main.json", "9f9488e21aae1cd84455ba309f1e52893d1853e8ae55bb87e3a42fc2f8dd5b2b")
		checkHash(t, "tests/handover/testdata/followup25-snapshots.json", "9e4039c9d90dbdb323d15f2d1ea1b2c500d472091db65f7f24d1168885334c9d")
		for path, want := range map[string]string{
			"README.md":                       "33cd7c4c211ac070f3f72f24ec5f257667bc4195825418f4a421bf7b21cbe223",
			"docs/handover/00-index.md":       "92c1ca425f333cc83e7b32d8d05cad19cdd970a201e7cf4e59deb4de5f5cd612",
			"docs/handover/deploy.md":         "f9f2195ba6678ebef7802ad25de2dd56769fe7ddf09e1d8aea9f3f30184b971f",
			"docs/handover/operator-guide.md": "744d3c6763f5d0b4d9ba1ced7a4e3b6193183b954425d245005e552a358217e9",
			"docs/handover/member-guide.md":   "ca06333f288dcbeb51c300c1c91011dc53d08b8abaaf1faf4b75970bef3f8e18",
			"docs/handover/restore-drill.md":  "cc30c854377f1183709e39251d7af9a7a8b48181fe49df201630772a01273db8",
		} {
			content := strings.ReplaceAll(strings.ReplaceAll(readBeforeFollowup29Note(t, root, path), followup28Diagnostics, ""), followup26InstallerNote+"\n\n", "")
			lines := strings.Split(content, "\n")
			if len(lines) < 4 {
				t.Fatalf("short entrypoint: %s", path)
			}
			sum := sha256.Sum256([]byte(strings.Join(lines[4:], "\n")))
			if hex.EncodeToString(sum[:]) != want {
				t.Errorf("operator content changed beyond entrypoint: %s", path)
			}
		}
	})
	t.Log("follow-up 25: 18 subtests / 4 exact roadmap rows / 15 immutable snapshots / 270 archive and run file checks")
}

// Live integrity deliberately stays separate from frozen acceptance claims.
func TestHandover_Followup25LiveDocumentIntegrity(t *testing.T) {
	root := repoRoot(t)
	var manifest struct {
		Followup struct {
			Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
		} `json:"handover_follow_up_25"`
	}
	if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Followup.Documents) != 18 {
		t.Fatalf("live document inventory %d", len(manifest.Followup.Documents))
	}
	for _, p := range manifest.Followup.Documents {
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, p.Path)))
		if hex.EncodeToString(sum[:]) != p.SHA256 {
			t.Errorf("stale live digest: %s", p.Path)
		}
	}
}

// The sole operator-content addition in this cut; prior body assertions remove only this line.
const followup26InstallerNote = "2026-11-01 — Installer follow-up 9 ([#378 roadmap row](../ROADMAP.md), [dated hand-off](../node/install-followup9-2026-11-01.md)): `fvpn-agent.service` sets `TimeoutStopSec=15s`; watchdog `Before=fvpn-agent.service` keeps it alive through a joint agent stop. Binary replacement waits up to 20 s per unit; a stop error or timeout journals `stop-failed` and rolls back before touching the candidate, `current` or units."

// Follow-up 26 pins the #382 cut without reading members/isolation living roots.
func TestHandover_Followup26Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "9dde5eeb60a6cb6b9b10974c3100ef1ea1352698"
	const fixture = "tests/handover/testdata/"
	const archive = fixture + "history/follow-up-26/"
	const doc = "docs/handover/follow-up-26.md"
	const register = fixture + "history/follow-up-27/register26.md"
	readJSON := func(t *testing.T, path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	checkHash := func(t *testing.T, path, want string) {
		t.Helper()
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
		if hex.EncodeToString(sum[:]) != want {
			t.Errorf("immutable file drift: %s", path)
		}
	}
	type claim struct {
		path   string
		tokens []string
	}
	for _, tc := range []struct {
		name   string
		claims []claim
	}{
		{"previous_cut_18_subtests_and_r18_archive", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"done (#374)", "18 subtests / 4 exact roadmap rows / 15 immutable snapshots / 270", "r18-generation-change-review.md"}},
			{"docs/handover/follow-up-26.md", []string{"#370–#373 at 9cd7a053", "18 subtests / 4 exact roadmap rows /\n15 immutable snapshots / 270"}},
			{"tests/handover/handover_test.go", []string{"TestHandover_Followup25Claims"}},
		}},
		{"members14_archive_and_AT02_losses", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"done (#375)", "all 61 follow-up 13 root files", "4182dd3d", "258ba504", "2 pass (AT03, AT08)", "58 baselines paired to recovery", "21 losses counted"}},
			{"docs/handover/follow-up-26.md", []string{"AT03 and AT08\npass", "AT02 fails", "58 baselines\npaired to recovery, 21 losses"}},
		}},
		{"members14_F1301_F1401_and_AT09_order", []claim{
			{"tests/handover/testdata/history/follow-up-26/members14-routing.md", []string{"F14-01 → devstack follow-up 26", "no common cause with F11-01 assumed"}},
			{"tests/handover/testdata/followup26-main.json", []string{"four duplicated identical chain definitions", "raw bytes private, public digest", "confounded, reordered in 682d4434, not live-qualified", "New F14-01", "failed after valid readiness"}},
			{"docs/handover/follow-up-26.md", []string{"AT04 fails", "four duplicated identical chain definitions", "AT09's relay row is confounded by harness order", "682d4434", "not live-qualified", "D-077 re-grant forwarding attempt failed after valid\nreadiness", "cause unproven"}},
		}},
		{"agent32_M1_M2_cancellable_and_shared_budget", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"done (#376)", "no observation POST after the caller cancels", "one local deadline per Step", "worst case 13 s"}},
			{"tests/handover/testdata/history/follow-up-26/agent32-handoff.md", []string{"Every\nStep observation POST uses that budget", "at most 13 seconds", "cooperative bound", "one\ndeadline"}},
			{"node/agent/review_r18_generation_test.go", []string{"TestReviewR18_Q4_ControlRequestAfterSIGTERM", "TestReviewR18_Q4_ShutdownExceedsLocalBound"}},
			{"docs/handover/follow-up-26.md", []string{"No observation POST after the caller cancels", "One local deadline per Step", "cooperative SIGTERM-to-exit worst case 13 s"}},
		}},
		{"agent32_M3_alarm_and_L2_ceilings", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"every stale-lease hold observation carries", "policy_missing", "STALE_GENERATION", "after 3 Steps or 30 s", "any higher or missing ceiling"}},
			{"tests/handover/testdata/history/follow-up-26/agent32-handoff.md", []string{"three Steps or 30 elapsed\nseconds", "The baseline persists", "A higher ceiling fails the leg"}},
			{"node/agent/review_r18_generation_test.go", []string{"TestReviewR18_Q5_BaselineHoldHasNoCapOrDistinctAlarm", "TestReviewR18_Q2_AcceptanceGateMissesQoSCeiling"}},
			{"node/agent/followup32_test.go", []string{"TestF32_AT13_StaleHoldMonotonicAlarmAndRecovery", "TestF32_AT13_StaleCodeSurvivesSafetyPrecedence"}},
			{"docs/handover/follow-up-26.md", []string{"policy_missing plus STALE_GENERATION at stage lease", "3 Steps or 30 s", "any higher or missing ceiling"}},
		}},
		{"agent32_ISOF601_rejected_candidate", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"observationState", "omitted the rejected candidate generation", "seven-attempt regression"}},
			{"tests/handover/testdata/history/follow-up-26/agent32-handoff.md", []string{"omitting the rejected candidate", "seven failed pre-install\nApplies", "stage=apply_candidate", "verified=false"}},
			{"node/agent/followup32_test.go", []string{"TestF32_ISOF601_RejectedCandidateSurvivesObservationBackoff"}},
			{"docs/handover/follow-up-26.md", []string{"observationState", "omitted the rejected candidate generation", "no retrospective attribution to #369"}},
		}},
		{"netlab23_L1_fixed_and_duplicate_origin_unproven", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"done (#377)", "r18 L-1 fixed", "0 / 3,300", "*enforce.DuplicateChainError", "duplicate_chain", "no historical row reclassified"}},
			{"node/enforce/review_r18_lifecycle_test.go", []string{"TestReviewR18_EnabledSampleIgnoresInstallDeadline"}},
			{"node/enforce/followup23_listing_test.go", []string{"TestF13_01_DuplicateChainKind"}},
			{"docs/handover/follow-up-26.md", []string{"0 / 3,300 duplicated-chain listings", "origin unproven", "No historical row is reclassified"}},
		}},
		{"installer9_unit_and_binary_replace_stop_failure", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"done (#378)", "TimeoutStopSec=15s", "Before=fvpn-agent.service", "20 s per-unit stop deadline", "stop-failed"}},
			{"tests/handover/testdata/history/follow-up-26/installer9-handoff.md", []string{"TimeoutStopSec=15s", "Before=fvpn-agent.service", "20 s", "before extracting the candidate", "stop-failed"}},
			{"packaging/installer/followup9_test.go", []string{"TestPKG01_AT07ShippedAgentStopBudget", "TestPKG01_StopWaitsThroughUnitBudget", "TestPKG01_AT25ReplaceStopFailureIsJournaled"}},
			{"docs/handover/follow-up-26.md", []string{"20 s per-unit stop deadline", "candidate, `current` or units", "No other new operator guidance"}},
		}},
		{"agent33_design_only_limits_and_reference", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"done (#379)", "7 tests, 558 table subtests", "intersection_constraints", "baselines before the final full Apply", "every feature-enabled restart", "Option (c) stays in production"}},
			{"tests/handover/testdata/history/follow-up-26/agent33-design.md", []string{"Design only, pending the recorded D-139 representation decision", "intersection_constraints", "Absence means unsupported, hence baseline", "Feature-enabled restart still baselines/reacquires", "An optimized full promotion without that baseline is **not** part"}},
			{"node/agent/followup33_intersection_spec_test.go", []string{"TestF33_ISO02_IntersectionIdentityTable", "TestF33_FUP02_FUP04_IntersectionCeilings", "TestF33_ENT02_IntersectionDeadlines", "TestF33_ISO01_ISO04_IntersectionSets", "TestF33_ENT01_IntersectionIdentityConjunction", "TestF33_ENT02_IntersectionMultipleDevicesAndOrder", "TestF33_ISO02_IntersectionAmbiguityAndBounds"}},
			{"docs/handover/follow-up-26.md", []string{"7 tests, 558 table subtests", "No uninterrupted AT02\ntransition is promised", "r19 running", "not an acceptance result", "Option (c) stays in production"}},
		}},
		{"isolation13_new_AT10_pass_old_block_and_alarm_limit", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"done (#380)", "1c25ee44 (108 files)", "38074149", "AT10 PASS 9 / 0 / 0", "ISO-F6-01 6 / 6", "assertion 6 met by accepted reports", "AT12 PASS 10 / 0 / 0", "alarm not exercised", "Historical 1c25ee44 stays blocked"}},
			{"tests/e2e/at/isolation/producer_timing_test.go", []string{"TestD098_Followup12ArchivePins"}},
			{"docs/handover/follow-up-26.md", []string{"AT11", "BLOCKED 2 / 1 / 0", "AT13", "BLOCKED 6 / 1 / 0", "the alarm was not exercised", "1c25ee44's block stays historical"}},
		}},
		{"devstack25_closed_node_signature_and_corrected_prior_state", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"done (#381)", "32 predeclared fresh attempts", "two successful kernel IPv6 requests", "four already sampled generation 0 / serving=false before the request", "1d253f8d had an earlier true heartbeat", "0c27cac0 went from generation 5 / serving=true to 0 / false", "closure cause is unproven"}},
			{"tests/handover/testdata/history/follow-up-26/devstack25-analysis.md", []string{"| kernel | ipv4 | 8 | 7 | 1 |", "| kernel | ipv6 | 8 | 7 | 1 |", "| userspace | ipv4 | 8 | 7 | 1 |", "| userspace | ipv6 | 8 | 5 | 3 |", "not a general forwarding fix", "five SYNs", "generation 0 / serving=false after the request"}},
			{"docs/handover/follow-up-26.md", []string{"32 attempts: 26 pass / 6 timeout", "timed out 3 / 8", "Four already sampled that state before the request", "The cause is unproven"}},
		}},
		{"updates_receipt_delivered_historical_AT23_retained", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"TEST ONLY parent-owned agent wait receipt", "D-102 answer for updates"}},
			{"tests/handover/testdata/history/follow-up-26/devstack25-analysis.md", []string{"receipt keyed by a unique launch ID", "historical first AT23 exit remains\nunobserved", "adopt the D-102 readback"}},
			{"tests/e2e/internal/agentwait/wait_test.go", []string{"TestAT23_WaitStatusSurvivesReap"}},
			{"docs/handover/follow-up-26.md", []string{"TEST ONLY agent wait receipt was delivered", "D-102 answer for\nupdates", "AT23's historical unobserved exit stays unobserved", "Updates F13-01 is distinct from members F13-01"}},
		}},
		{"harness_stop_path_is_TEST_ONLY_not_product_finding", []claim{
			{"tests/handover/testdata/history/follow-up-26/isolation13-routing.md", []string{"SIGKILL / exit 137", "10 s (ordinary) and 20 s (clone fixture)", "harness signal forwarding"}},
			{"tests/handover/testdata/history/follow-up-26/devstack25-analysis.md", []string{"five-second escalation threshold", "live SIGKILL is harness-injected"}},
			{"tests/handover/testdata/history/follow-up-26/devstack25-routing.md", []string{"align the harness stop path with the shipped 15 s budget", "container SIGTERM forwarding"}},
			{"docs/handover/follow-up-26.md", []string{"TEST ONLY harness stop-path gap", "SIGKILL / exit 137", "about 5 s", "a harness gap, not a product finding", "routed to devstack 26"}},
		}},
		{"nodes34_recognition_classification_retention_escalation", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"done (#382)", "whatever the applied ACK says", "joined only to its own signed lease", "only the agent's two dependent qos / source_binding checks yield", "Migration 00194", "One transactional admin audit escalation"}},
			{"tests/handover/testdata/history/follow-up-26/nodes34-handoff.md", []string{"It does not compare the applied ACK to the desired", "same node's current incarnation", "Every unrelated failed check wins", "00194_node_generation_holds.sql", "exactly one", "node.generation_hold.escalated", "central does not need to detect a process restart"}},
			{"internal/nodes/generation_hold_test.go", []string{"TestAT29_M3_HoldClassificationOrder", "TestAT29_M3_SummaryStrictAndLossless"}},
			{"internal/nodes/generation_hold_integration_test.go", []string{"TestAT29_M3_HoldRecognitionDoesNotDependOnACK", "TestAT29_M3_CurrentIncarnationCannotBorrowOldLease", "TestAT29_M3_RestartsRetainIncidentAndCentralDeadline", "TestAT29_M3_AlarmRollbackAndMalformedSummary"}},
			{"docs/handover/follow-up-26.md", []string{"Migration 00194 retains incidents across agent boots", "one `node.generation_hold.escalated` audit escalation per incident", "no AT\npass is claimed here", "not yet been exercised in an assembled run"}},
		}},
		{"D139_provisional_design_track_and_grounds", []claim{
			{"tests/handover/testdata/followup26-main.json", []string{"D-139 design track", "173.8 / 195.6 ms", "501 sibling losses counted"}},
			{register, []string{"| D-139 |", "06:00 EST, 2026-11-01, orchestrator — provisional", "only as a design track", "before contracts, wire-nodes, netlab or agent-core implement anything", "58 brackets and 21 losses", "only path to an unaffected sibling without a requirement change"}},
			{"docs/handover/follow-up-26.md", []string{"D-139 — orchestrator, provisional, 2026-11-01 06:00 EST", "one recorded decision approves or\nrejects its representation", "D-135, D-136 (including its 04:3x correction), D-137 and D-138 stay\nprovisional and unchanged", "No finite outage ceiling"}},
		}},
		{"dated_owner_decisions_and_outside_scope_routing", []claim{
			{"docs/handover/follow-up-26.md", []string{"### Requests to orchestrator", "2026-11-01 — **Owner decision 1: D-135 + D-136 confirmation", "Owner decision 2: D-137 hardening option", "Owner decision 3: D-138 overturn option", "Owner decision 4: D-139 overturn option", "Owner decision 5: F11-01 repair direction", "devstack 26's closed-node investigation", "owner decision, not a recommendation", "Already assigned, not new requests (2026-11-01)", "F14-01", "deploy/devstack/agent-service.sh", "deploy/devstack/entrypoint-node.sh", "tests/e2e/at/updates/stopobserve/main.go", "Isolation and members", "No outside-owned pin needed changing"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.claims {
				requireHandoverClaimTokens(t, root, c.path, c.tokens...)
			}
		})
	}
	t.Run("cutoff_nine_rows_correction_and_pending_lanes", func(t *testing.T) {
		var pin struct {
			Commit     string
			Cutoff     int    `json:"cutoff_pr"`
			Correction string `json:"roadmap_correction"`
			Rows       []string
			Context    []string `json:"context_rows"`
			After      []int    `json:"merged_after_cutoff"`
			Absent     []string `json:"absent_at_start"`
		}
		readJSON(t, fixture+"followup26-main.json", &pin)
		if pin.Commit != cut || pin.Cutoff != 382 || pin.Correction != "5f4b455c" || len(pin.Rows) != 9 || len(pin.After) != 0 || !slices.Equal(pin.Absent, []string{"r19 adversarial review", "w4-at-isolation follow-up 14", "w4-at-members follow-up 15", "w3-devstack-e2e follow-up 26"}) {
			t.Fatalf("wrong cutoff: %+v", pin)
		}
		for i, row := range pin.Rows {
			if !strings.HasSuffix(row, fmt.Sprintf("| done (#%d) |", 374+i)) {
				t.Fatalf("wrong PR order: %s", row)
			}
			roadmapPath := "docs/ROADMAP.md"
			if strings.HasSuffix(row, "| done (#381) |") {
				roadmapPath = fixture + "history/follow-up-27/roadmap381.md"
			}
			requireHandoverClaimTokens(t, root, roadmapPath, row)
		}
		if len(pin.Context) != 1 || !strings.HasSuffix(pin.Context[0], "| done (#369) |") {
			t.Fatal("wrong historical grounds")
		}
		requireHandoverClaimTokens(t, root, "docs/ROADMAP.md", pin.Context[0])
		requireHandoverClaimTokens(t, root, doc, cut, "#374–#382", "5f4b455c", "Nothing\nwas merged after the cutoff at task start", "r19, isolation 14, members 15 and\ndevstack 26 were running", "none of their results enters this cut", "ran no new live acceptance", "D-098 / D-109 / D-132")
	})
	t.Run("D098_archives_and_devstack_run_1075_files", func(t *testing.T) {
		total := 0
		for _, tc := range []struct {
			manifest, base, digest string
			count                  int
		}{
			{"docs/e2e/at/members/history/follow-up-13/SHA256SUMS", "docs/e2e/at/members/history/follow-up-13/", "76df367a1f43a6df45989d35cfed1eff3036a49043e98cd4dd02e3682d2a98d0", 61},
			{archive + "devstack25-captures.sha256", "", "840969ecbc5ec10148286386dd1b4e94a152d5e6aeccfc356171432df8f2cb89", 906},
		} {
			checkHash(t, tc.manifest, tc.digest)
			lines := strings.Split(strings.TrimSpace(readOwnedDoc(t, root, tc.manifest)), "\n")
			if len(lines) != tc.count {
				t.Fatalf("manifest count %s: %d", tc.manifest, len(lines))
			}
			seen := map[string]bool{}
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) != 2 || len(fields[0]) != 64 || !filepath.IsLocal(fields[1]) || seen[fields[1]] {
					t.Fatalf("bad manifest row: %s", line)
				}
				if tc.base == "" && !strings.HasPrefix(fields[1], "docs/e2e/runs/2026-11-01/followup25/") {
					t.Fatalf("non-run capture: %s", fields[1])
				}
				seen[fields[1]] = true
				checkHash(t, tc.base+fields[1], fields[0])
				total++
			}
		}
		const iso = "docs/e2e/at/isolation/history/follow-up-12/"
		checkHash(t, iso+"archive.json", "ac6e5ab497b86be6568d7df93a9c813124d9ffecc497eb24c5959098dbbe5c3e")
		var inventory struct {
			Run   string `json:"run_id"`
			Files []struct{ Path, SHA256 string }
		}
		readJSON(t, iso+"archive.json", &inventory)
		if inventory.Run != "1c25ee44-1872-4cce-a9f5-d52491d87095" || len(inventory.Files) != 108 {
			t.Fatalf("wrong isolation archive: %s / %d", inventory.Run, len(inventory.Files))
		}
		seen := map[string]bool{}
		for _, f := range inventory.Files {
			if !filepath.IsLocal(f.Path) || seen[f.Path] {
				t.Fatalf("invalid archive path: %s", f.Path)
			}
			seen[f.Path] = true
			checkHash(t, iso+f.Path, f.SHA256)
			total++
		}
		if total != 1075 {
			t.Fatalf("archive/run files: %d", total)
		}
	})
	t.Run("D098_D109_D132_17_snapshots_never_living_roots", func(t *testing.T) {
		checkHash(t, fixture+"followup26-main.json", "dd46d62b5ab3ba9a5f3ac521a2f7673a1c0456796f0bb725cd061a4255c11f27")
		checkHash(t, fixture+"followup26-snapshots.json", "1e8d0728cdb2d249c7d92b695f180ea565d414dbb1b338467f5ed089e7a1842b")
		var pins []struct {
			Path, Source, SHA256 string
			Commit               string `json:"source_commit"`
			Kind                 string `json:"source_kind"`
			SourceSHA            string `json:"source_sha256"`
			Offset, Length       int
		}
		readJSON(t, fixture+"followup26-snapshots.json", &pins)
		if len(pins) != 17 {
			t.Fatalf("snapshot count %d", len(pins))
		}
		seen := map[string]bool{}
		for _, pin := range pins {
			if seen[pin.Path] || !filepath.IsLocal(pin.Path) || !strings.HasPrefix(pin.Path, archive) || len(pin.Commit) != 40 || len(pin.SourceSHA) != 64 || pin.Offset < 0 || pin.Length <= 0 {
				t.Fatalf("invalid snapshot: %+v", pin)
			}
			seen[pin.Path] = true
			if strings.HasPrefix(pin.Source, "docs/e2e/at/") {
				t.Fatalf("living AT source: %s", pin.Source)
			}
			switch pin.Kind {
			case "file":
				if pin.Commit != cut {
					t.Fatal("wrong snapshot source commit")
				}
				owned := pin.Source == "README.md" || strings.HasPrefix(pin.Source, "docs/handover/") || strings.HasPrefix(pin.Source, "docs/ops/")
				dated := strings.HasSuffix(pin.Source, "-2026-11-01.md") || strings.HasPrefix(pin.Source, "docs/e2e/runs/2026-11-01/followup25/")
				if !owned && !dated && pin.Source != "docs/e2e/followup25-captures.sha256" {
					t.Fatalf("non-immutable source: %s", pin.Source)
				}
			case "commit_message":
				if pin.Source != "PR #375 merge message" && pin.Source != "PR #380 merge message" && pin.Source != "PR #381 merge message" {
					t.Fatalf("wrong PR source: %s", pin.Source)
				}
			default:
				t.Fatalf("unknown source kind: %s", pin.Kind)
			}
			checkHash(t, pin.Path, pin.SHA256)
			if len(readOwnedDoc(t, root, pin.Path)) != pin.Length {
				t.Errorf("snapshot size drift: %s", pin.Path)
			}
		}
		for _, path := range []string{"docs/e2e/at/members/AT02.json", "docs/e2e/at/members/F14-01.json", "docs/e2e/at/isolation/AT10.json", "docs/e2e/at/isolation/summary.md", "docs/e2e/at/isolation/producer-operations.jsonl.gz", "docs/e2e/RESPONSES.md", "deploy/devstack/agent-service.sh", "node/agent/observe.go"} {
			if claimTokenAllowed(path, "mutable implementation or evidence token") {
				t.Errorf("mutable pin allowed: %s", path)
			}
		}
		requireHandoverClaimTokens(t, root, doc, "root records are living until members 15 and\nisolation 14 archive them", "cite by roadmap row / PR only", "never copy or\npin those living roots")
	})
	t.Run("previous_assertions_decisions_and_other_findings_preserved", func(t *testing.T) {
		source := readOwnedDoc(t, root, "tests/handover/handover_test.go")
		start := strings.Index(source, "func TestHandover_Followup25Claims(")
		if start < 0 {
			t.Fatal("missing prior function")
		}
		end := strings.Index(source[start:], "// Live integrity deliberately")
		if end < 0 {
			t.Fatal("missing prior function end")
		}
		prior := restoreFollowup29Assertions(source[start : start+end])
		prior = strings.ReplaceAll(prior, "strings.ReplaceAll(readOwnedDoc(t, root, path), followup28Diagnostics, \"\")", "readOwnedDoc(t, root, path)")
		prior = strings.ReplaceAll(prior, "\tconst archive26 = fixture + \"history/follow-up-26/\"\n", "")
		prior = strings.ReplaceAll(prior, "const register = archive26 + \"register25.md\"", "const register = \"docs/handover/README.md\"")
		prior = strings.ReplaceAll(prior, "{register, []string{", "{\"docs/handover/README.md\", []string{")
		prior = strings.ReplaceAll(prior, "archive26+\"entrypoint-\"+strings.ReplaceAll(path, \"/\", \"_\")+\".txt\"", "path")
		prior = strings.ReplaceAll(prior, "content := strings.ReplaceAll(readOwnedDoc(t, root, path), followup26InstallerNote+\"\\n\\n\", \"\")\n\t\t\tlines := strings.Split(content, \"\\n\")", "lines := strings.Split(readOwnedDoc(t, root, path), \"\\n\")")
		sum := sha256.Sum256([]byte(strings.TrimSpace(prior)))
		if hex.EncodeToString(sum[:]) != "04ea7d0299aecdf9c1f43a7e23f8943491345c87d31e8ec0b1f3440bac7126f4" {
			t.Fatal("follow-up 25 assertions changed beyond frozen-input redirects and the sole operator note")
		}
		checkHash(t, "docs/handover/follow-up-25.md", "8103922e0f29d3008b909589831240c3794bee6ca3d490f3d780a851aba7da2d")
		checkHash(t, fixture+"followup25-main.json", "9f9488e21aae1cd84455ba309f1e52893d1853e8ae55bb87e3a42fc2f8dd5b2b")
		checkHash(t, fixture+"followup25-snapshots.json", "9e4039c9d90dbdb323d15f2d1ea1b2c500d472091db65f7f24d1168885334c9d")
		checkHash(t, "docs/reviews/r18-generation-change-review.md", "f266a0705292975f5416bda5a082a3f7e779f45e82cb9b9ce54843ce0cb8e33d")
		old := readOwnedDoc(t, root, archive+"register25.md")
		current := readOwnedDoc(t, root, register)
		decisions := regexp.MustCompile(`(?m)^\| D-[0-9]{3} \|.*$`)
		before, after := decisions.FindAllString(old, -1), decisions.FindAllString(current, -1)
		if len(after) != len(before)+1 || !slices.Equal(before, after[:len(before)]) || !strings.HasPrefix(after[len(before)], "| D-139 |") {
			t.Fatal("decisions changed beyond new D-139")
		}
		oldFindings := strings.Split(strings.Split(old, "## Findings register at the follow-up 25 cutoff")[1], "\nEvery historical failure")[0]
		currentFindings := strings.Split(strings.Split(current, "## Findings register at the follow-up 26 cutoff")[1], "\nEvery historical failure")[0]
		changed := map[string]bool{"Members F11-01": true, "Members F13-01": true, "Updates F13-01": true, "ISO-F6-01 assertion 6": true, "r18 M-1": true, "r18 M-2": true, "r18 M-3": true, "r18 L-1": true, "r18 L-2": true}
		count := 0
		for _, row := range strings.Split(oldFindings, "\n") {
			if !strings.HasPrefix(row, "| ") || strings.HasPrefix(row, "| Finding |") {
				continue
			}
			count++
			name := strings.TrimSpace(strings.Split(row, "|")[1])
			if !changed[name] && !strings.Contains(currentFindings, row) {
				t.Errorf("previous finding changed: %s", row)
			}
		}
		if count != 37 {
			t.Fatalf("previous finding count %d", count)
		}
		added := 0
		for _, row := range strings.Split(currentFindings, "\n") {
			if strings.HasPrefix(row, "| ") && !strings.HasPrefix(row, "| Finding |") {
				added++
			}
		}
		if added != 39 {
			t.Fatalf("expected F14-01 and harness gap additions, total %d", added)
		}
		requireHandoverClaimTokens(t, root, register, "Every other follow-up 25 finding is unchanged", "Every historical failure stays failed", "| Members F14-01 | OPEN", "| Harness stop-path gap | TEST ONLY", "| r18 M-2 | FIXED on both sides — #376 + #378", "| r18 M-3 | FIXED on both sides — #376 + #382", "| ISO-F6-01 assertion 6 | FIXED", "| Updates F13-01 | RECEIPT DELIVERED")
	})
	t.Run("seven_entrypoints_and_only_one_operator_note", func(t *testing.T) {
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			if path != "docs/handover/operator-guide.md" && strings.Contains(readOwnedDoc(t, root, path), followup26InstallerNote) {
				t.Errorf("installer note outside the operator guide: %s", path)
			}
			lines := strings.Split(readOwnedDoc(t, root, fixture+"history/follow-up-27/entrypoint-"+strings.ReplaceAll(path, "/", "_")+".txt"), "\n")
			if len(lines) < 4 || !strings.Contains(lines[2], "Current cut") || !strings.Contains(lines[2], "follow-up-26.md") || !strings.Contains(lines[2], "#382") || !strings.Contains(lines[3], "#373") {
				t.Errorf("stale entrypoint: %s", path)
			}
		}
		operator := readOwnedDoc(t, root, "docs/handover/operator-guide.md")
		start := strings.Index(operator, "### 3.1 ")
		if start < 0 {
			t.Fatal("missing operator section 3.1")
		}
		end := strings.Index(operator[start:], "### 3.2 ")
		if end < 0 || strings.Count(operator, followup26InstallerNote) != 1 || !strings.Contains(operator[start:start+end], followup26InstallerNote) {
			t.Fatal("sole installer note missing or outside section 3.1")
		}
		// The prior body hash checks (with exactly this line removed) prove no other operator content changed.
		for pr := 374; pr <= 382; pr++ {
			requireHandoverClaimTokens(t, root, "docs/ops/README.md", fmt.Sprintf("#%d", pr))
		}
		requireHandoverClaimTokens(t, root, "docs/reviews/README.md", "r18 dispositions at the follow-up 26 cutoff", "Fixed on both sides #376 + #378", "Fixed on both sides #376 + #382", "not yet exercised in an assembled run", "reference passing is not an acceptance result")
	})
	t.Log("follow-up 26: 21 subtests / 9 exact roadmap rows / 17 immutable snapshots / 1075 archive and run file checks")
}

// Live owned-document integrity remains separate from historical acceptance.
func TestHandover_Followup26LiveDocumentIntegrity(t *testing.T) {
	root := repoRoot(t)
	var manifest struct {
		Followup struct {
			Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
		} `json:"handover_follow_up_26"`
	}
	if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Followup.Documents) != 19 {
		t.Fatalf("live document inventory %d", len(manifest.Followup.Documents))
	}
	for _, p := range manifest.Followup.Documents {
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, p.Path)))
		if hex.EncodeToString(sum[:]) != p.SHA256 {
			t.Errorf("stale live digest: %s", p.Path)
		}
	}
}

// Follow-up 27 freezes the #383–#385 cut. Living isolation/member roots,
// including compressed captures, are never inputs to these acceptance claims.
func TestHandover_Followup27Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "37db5405ee70730fe6a709d983f95f2460136c34"
	const fixture = "tests/handover/testdata/"
	const archive = fixture + "history/follow-up-27/"
	const doc = "docs/handover/follow-up-27.md"
	const register = fixture + "history/follow-up-28/register27.md"
	readJSON := func(t *testing.T, path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	checkHash := func(t *testing.T, path, want string) {
		t.Helper()
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
		if hex.EncodeToString(sum[:]) != want {
			t.Errorf("immutable file drift: %s", path)
		}
	}
	type claim struct {
		path   string
		tokens []string
	}
	for _, tc := range []struct {
		name   string
		claims []claim
	}{
		{"previous_cut_and_all_21_subtests", []claim{
			{"tests/handover/testdata/followup27-main.json", []string{"done (#384)", "21 subtests / 9 exact roadmap rows / 17 immutable snapshots / 1,075", "5f4b455c"}},
			{"docs/handover/follow-up-27.md", []string{"#374–#382 at 9dde5eeb", "5f4b455c", "21\nsubtests / 9 exact roadmap rows / 17 immutable snapshots / 1,075", "Every earlier claims test\nstays true"}},
			{"tests/handover/handover_test.go", []string{"TestHandover_Followup26Claims"}},
		}},
		{"isolation14_archive_and_case_results", []claim{
			{"tests/handover/testdata/followup27-main.json", []string{"done (#383)", "38074149 (112 root files)", "56d20530 on #382", "AT10 PASS 9 / 0 / 0", "ISO-F6-01 6 / 6", "AT12 PASS 10 / 0 / 0", "AT11 2 / 1 / 0 blocked on operator rows", "AT13 FAIL 5 / 1 / 1"}},
			{"docs/handover/follow-up-27.md", []string{"112 root files", "PASS 9 / 0 / 0", "ISO-F6-01 6 / 6", "PASS 10 / 0 / 0", "BLOCKED 2 / 1 / 0", "FAIL 5 / 1 / 1"}},
			{"tests/e2e/at/isolation/producer_timing_test.go", []string{"TestD098_Followup13ArchivePins"}},
		}},
		{"isolation_F1401_undecided_failed_boundary", []claim{
			{"tests/handover/testdata/followup27-main.json", []string{"isolation F14-01", "positive leaving at or after D−101 ms", "last positive left at D−162.220 ms", "packet 161 (scheduled D−101 ms)", "ingress at D−97.638 ms", "(D−162.2, D−97.6] ms, undecided and failed"}},
			{"docs/handover/follow-up-27.md", []string{"Isolation F14-01 is distinct from members F14-01", "D−162.220 ms", "Packet 161", "D−97.638 ms", "(D−162.2, D−97.6] ms", "undecided and failed"}},
			{"tests/e2e/at/isolation/deadline_trace_test.go", []string{"TestAT13_D104StrictBoundaryAndMissingEvidence"}},
		}},
		{"isolation14_bootstrap_join_not_populated_hold", []claim{
			{"tests/handover/testdata/followup27-main.json", []string{"one bootstrap hold (zero-member lease", "joined exactly by report, lease and incident", "generation_matched", "no populated hold was produced", "alarm / escalation, active-stop retention"}},
			{"docs/handover/follow-up-27.md", []string{"bootstrap hold (zero-member lease)", "generation_matched", "No populated hold was produced", "alarm, escalation and active-stop retention are not exercised"}},
			{"tests/e2e/at/isolation/producer_timing_test.go", []string{"TestAT29_M3CentralHoldEvidence"}},
		}},
		{"members15_AT02_all_losses_and_unproven_gap", []claim{
			{"tests/handover/testdata/followup27-main.json", []string{"done (#385)", "all 70 follow-up 14 root files of run 258ba504", "e80bb592", "2 pass (AT03, AT08) / 6 blocked / 2 fail", "644 / 6 / 2 subchecks", "93 successful Applies, 0 failed", "65 baselines paired to recovery", "4,131 attempts, 277 without egress", "249 absent at node ingress", "1108–1356", "5.0975 s", "zero capture drops; cause unproven", "28 ingress-without-egress (12 inside guards / 16 outside)"}},
			{"docs/handover/follow-up-27.md", []string{"70 root files", "2 pass / 6\nblocked / 2 fail", "644 / 6 / 2 subchecks", "AT02 fails", "93 successful Applies, 0 failed; 65 baselines paired to\nrecovery", "4,131 attempts", "277 have no egress", "249 never reached node ingress", "1108–1356", "5.0975 s", "zero capture drops", "cause unproven", "28 reached ingress without egress", "12 inside guards / 16 outside", "Every loss counts"}},
		}},
		{"members15_first_selection_and_three_serving_samples", []claim{
			{"tests/handover/testdata/followup27-main.json", []string{"AT01's F11-01 first selection fails after proven serving readiness (generation 5; five member SYNs, zero node / origin packets)", "three D-077 re-grants pass", "F14-01 stays failed historically"}},
			{"tests/handover/testdata/history/follow-up-27/members15-serving-routing.md", []string{"generation 5", "five member SYNs, zero node / origin packets", "249-sequence gap", "three samples of a signature distinct from devstack 25", "devstack follow-up 27 (collected)", "hypothesis, to be tested and not asserted"}},
			{"docs/handover/follow-up-27.md", []string{"generation 5", "five member SYNs, zero node or origin packets", "Three\nD-077 re-grants pass", "Members F14-01 stays failed historically", "three\nsamples of serving readiness proven, member packets absent at node ingress", "distinct from devstack 25's closed-node timeouts", "shared cause is unproven", "devstack 27 (collected)"}},
		}},
		{"members15_AT09_order_and_D137_failed_limitation", []claim{
			{"tests/handover/testdata/followup27-main.json", []string{"682d4434's routed-after-relay order is live-qualified", "receive-only dual MTU-1280 row fails (12,012 bytes in 12,011 ms)"}},
			{"docs/handover/follow-up-27.md", []string{"682d4434's routed-after-relay order is live-qualified", "receive-only dual MTU-1280 row fails: 12,012 bytes in 12,011 ms", "D-137 limitation, not upgraded or waived", "Historical F8-02 stays failed"}},
			{register, []string{"| F8-02 / native MTU 1280 | OPEN", "12,012 bytes in 12,011 ms", "Not upgraded or waived"}},
		}},
		{"members15_gzip_and_plain_resume_index", []claim{
			{"tests/handover/testdata/followup27-main.json", []string{"Large evidence as lossless gzip"}},
			{"tests/handover/testdata/history/follow-up-27/members15-index-resume.md", []string{"2,551-byte navigation document", "e80bb592-27ed-4ab5-a8c3-6433e7278938", "FAIL, 142 / 0 / 1 subchecks", "all 277 losses", "both original capture links", "both SHA-256 and byte-count pairs", "gzip -dc", "excludes its own bytes from the capture pins", "readers reject the explicit document-index marker", "original compressed and uncompressed pins", "AT02.json.gz", "index and capture coexisting", "not a pinned capture or a new observation"}},
			{"docs/handover/follow-up-27.md", []string{"lossless gzip with compressed and uncompressed\nSHA-256", "AT02.md is a 2,551-byte navigation index", "FAIL, 142 / 0 / 1 subchecks", "excluded from the capture pins", "reject the document index", "operator guide's existing link remains useful and unchanged", "No replacement\ntarget is requested"}},
			{"tests/e2e/at/members/evidence_io_test.go", []string{"TestEvidenceGzipLosslessAndPins"}},
		}},
		{"r19_metadata_and_review_boundary", []claim{
			{"docs/reviews/r19-intersection-design-review.md", []string{"Branch: `r19-intersection-design-review`", "Reviewing model: `grok/grok-4.7`", "agent-core follow-up 33, #379", "Product code, the design note, and the reference were not changed.", "## What held", "## Not examined"}},
			{"docs/handover/follow-up-27.md", []string{"07:47 EST", "6048e099", "41,050 bytes", "14caf0193b256d6f090e8d53ff77e0d5a5c9dab4d025794a6812e15e83228913", "Only the report is archived; proof files remain on the review branch", "not assembled acceptance results"}},
			{"docs/reviews/README.md", []string{"| `r19-intersection-design-review`", "r19 dispositions at the follow-up 27 cutoff", "6048e099984ee03b69fec961309cd3eabe0e2b65", "No proof files were adopted"}},
		}},
		{"r19_routed_findings_not_fixed", []claim{
			{"docs/reviews/r19-intersection-design-review.md", []string{"### H-1 (High)", "established download survives a new destination denial", "### M-2 (Medium)", "ACK copies the new desired document's policy refs", "### L-2 (Low)", "conntrack attempt is counted after the delete"}},
			{"docs/handover/follow-up-27.md", []string{"r19 H-1", "Routed; in progress — netlab 24", "r19 M-2", "Routed; in progress — agent-core 34", "production reachability unknown", "r19 L-2", "conntrack attempt is persisted after the delete"}},
			{register, []string{"| r19 H-1 | ROUTED; IN PROGRESS — netlab 24", "| r19 M-2 | ROUTED; IN PROGRESS — agent-core 34", "production reachability unknown", "| r19 L-2 | ROUTED; IN PROGRESS — agent-core 34", "Not fixed at this cut"}},
		}},
		{"r19_design_only_conditions", []claim{
			{"docs/reviews/r19-intersection-design-review.md", []string{"### H-2 (High)", "### M-1 (Medium)", "### M-3 (Medium)", "### M-4 (Medium)", "### M-5 (Medium)", "### M-6 (Medium)", "### M-7 (Medium)", "### M-8 (Medium)", "### L-1 (Low)", "### L-3 (Low)", "### L-4 (Low)"}},
			{"docs/handover/follow-up-27.md", []string{"r19 H-2, M-1, M-3–M-8, L-1, L-3, L-4", "Design-only", "D-141 reconsideration conditions; no implementation or closure claimed"}},
			{register, []string{"| r19 design-only | D-141 RECONSIDERATION CONDITIONS", "H-2, M-1, M-3–M-8, L-1, L-3 and L-4"}},
		}},
		{"D140_boundary_schedule_only_no_retry_waiver", []claim{
			{"tests/handover/testdata/history/follow-up-27/isolation14-routing.md", []string{"**D-140** (orchestrator, provisional)", "predicate stays byte-identical", "no positive after the first drop, first drop by D+100 ms", "only the boundary probe schedule", "member-send → node-ingress latency around D−101 ms", "schedule and latencies recorded", "straddles D−101 ms stays failed", "run 56d20530's failure stays failed"}},
			{"docs/handover/follow-up-27.md", []string{"D-140 — orchestrator, provisional, 2026-11-01 07:42 EST", "tests/e2e/at/isolation/deadline_trace.go", "byte-identical", "boundary probe schedule", "member-send → node-ingress latency around D−101 ms", "the schedule and latencies recorded", "Run 56d20530 stays failed; no retry waiver", "isolation 15"}},
			{register, []string{"| D-140 |", "07:42 EST, 2026-11-01, orchestrator — provisional", "no retry waiver"}},
		}},
		{"D141_one_representation_decision_and_cost", []claim{
			{"tests/handover/testdata/history/follow-up-27/members15-D141.md", []string{"recorded decision is **D-141**", "option (b) is not implemented and option (c) stays in production"}},
			{"docs/reviews/r19-intersection-design-review.md", []string{"qualifying share for S on today's documents is 0 of 80", "p50=173766526 ns p99=195601257 ns", "DesiredState has no intersection witness", "Ship the v2 heartbeat", "no partial-service cause exists", "baseline on 422", "E needs its own nft transaction", "promotion baseline is the same order"}},
			{"docs/handover/follow-up-27.md", []string{"D-141 — orchestrator, provisional, 2026-11-01 07:5x EST", "one\nD-139 representation decision", "is not implemented on this evidence; option (c) stays in production", "D-139's design track is closed by D-141", "test-only reference stay on main as history", "0 / 80 merged restrictive changes qualify", "173.8 / 195.6 ms p50 / p99", "a signed intersection\nwitness, a v2 heartbeat, a partial-service cause, baseline-on-422 and an nft-only\nrevocation transaction", "a desired-state intersection\nwitness", "a measured qualifying share above zero whose promotion costs less\nthan (c)", "r19's design-only findings closed", "AT02's sibling outage on a restrictive generation change rests on the owner's\nD-135 / D-136 confirmation or a requirement change"}},
			{register, []string{"| D-139 |", "now closed by D-141", "| D-141 |", "07:5x EST, 2026-11-01, orchestrator — provisional", "D-135, D-136, D-137 and D-138 stay provisional and unchanged"}},
			{"tests/handover/testdata/history/follow-up-26/agent33-design.md", []string{"Design only", "intersection_constraints", "Feature-enabled restart still baselines/reacquires"}},
			{"node/agent/followup33_intersection_spec_test.go", []string{"TestF33_ISO02_IntersectionIdentityTable"}},
		}},
		{"owner_decisions_and_outside_scope_routing", []claim{
			{"docs/handover/follow-up-27.md", []string{"### Requests to orchestrator", "2026-11-01 — **Owner decision 1: D-135 + D-136 confirmation", "Owner decision 2: D-137 hardening option", "Owner decision 3: D-138 overturn option", "Owner decision 4: D-140 overturn option", "Owner decision 5: D-141 overturn option", "Owner decision 6: F11-01 repair direction", "devstack 26", "Already assigned, not new requests (2026-11-01)", "routed; in progress", "These outside-owned items stop with their owners", "no outside-owned pin needed changing", "Six owner-decision requests", "no operator procedure", "Every historical failure\nstays failed"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.claims {
				requireHandoverClaimTokens(t, root, c.path, c.tokens...)
			}
		})
	}
	t.Run("cutoff_three_exact_rows_and_pending_lanes", func(t *testing.T) {
		var pin struct {
			Commit string
			Cutoff int `json:"cutoff_pr"`
			Rows   []string
			After  []int    `json:"merged_after_cutoff"`
			Absent []string `json:"absent_at_start"`
		}
		readJSON(t, fixture+"followup27-main.json", &pin)
		if pin.Commit != cut || pin.Cutoff != 385 || len(pin.Rows) != 3 || !slices.Equal(pin.After, []int{386}) || !slices.Equal(pin.Absent, []string{"w3-devstack-e2e follow-up 26", "w1-netlab follow-up 24", "w2-agent-core follow-up 34"}) {
			t.Fatalf("wrong cutoff: %+v", pin)
		}
		for i, row := range pin.Rows {
			if !strings.HasSuffix(row, fmt.Sprintf("| done (#%d) |", 383+i)) {
				t.Fatalf("wrong PR order: %s", row)
			}
			requireHandoverClaimTokens(t, root, "docs/ROADMAP.md", row)
		}
		requireHandoverClaimTokens(t, root, doc, cut, "#383–#385", "Nothing was merged after the cutoff at task start", "Merged after the cutoff: #386", "Devstack 26, netlab 24\nand agent-core 34 were running", "their results do not enter this cut", "Handover ran no new live acceptance", "D-098 / D-109 / D-132")
	})
	t.Run("D098_owner_archives_182_files", func(t *testing.T) {
		const iso = "docs/e2e/at/isolation/history/follow-up-13/"
		checkHash(t, iso+"archive.json", "7a1bf68d1136c15aa0f2997425c1c0f4b12a3c24440c43cd5a6348f60a973e5c")
		var inventory struct {
			Run   string `json:"run_id"`
			Files []struct{ Path, SHA256 string }
		}
		readJSON(t, iso+"archive.json", &inventory)
		if inventory.Run != "38074149-1137-4782-80fd-8669c36c6fcb" || len(inventory.Files) != 112 {
			t.Fatalf("wrong isolation archive: %s / %d", inventory.Run, len(inventory.Files))
		}
		total := 0
		seen := map[string]bool{}
		for _, f := range inventory.Files {
			if !filepath.IsLocal(f.Path) || seen[f.Path] || len(f.SHA256) != 64 {
				t.Fatalf("bad isolation archive entry: %+v", f)
			}
			seen[f.Path] = true
			checkHash(t, iso+f.Path, f.SHA256)
			total++
		}
		const members = "docs/e2e/at/members/history/follow-up-14/"
		checkHash(t, members+"SHA256SUMS", "73490a61b2503ae261307056085b92d567a104ee2e9a16315c8c2c0a1764ae58")
		rows := strings.Split(strings.TrimSpace(readOwnedDoc(t, root, members+"SHA256SUMS")), "\n")
		if len(rows) != 70 {
			t.Fatalf("members archive count %d", len(rows))
		}
		seen = map[string]bool{}
		for _, row := range rows {
			fields := strings.Fields(row)
			if len(fields) != 2 || len(fields[0]) != 64 || !filepath.IsLocal(fields[1]) || seen[fields[1]] {
				t.Fatalf("bad members archive entry: %s", row)
			}
			seen[fields[1]] = true
			checkHash(t, members+fields[1], fields[0])
			total++
		}
		if total != 182 {
			t.Fatalf("archive files = %d", total)
		}
	})
	t.Run("D098_D109_D132_14_snapshots_and_exact_r19", func(t *testing.T) {
		checkHash(t, fixture+"followup27-main.json", "14c1e6c4f05fcee8ecce00b9ad3193f4c36698b7c16788b43a8a7ad6d041a6fc")
		checkHash(t, fixture+"followup27-snapshots.json", "e2769ac1dd73367f8c2c1fc5f695145138d76accacf3c37cc02c69a9744096b8")
		var pins []struct {
			Path, Source, SHA256 string
			Commit               string `json:"source_commit"`
			Kind                 string `json:"source_kind"`
			SourceSHA            string `json:"source_sha256"`
			Offset, Length       int
		}
		readJSON(t, fixture+"followup27-snapshots.json", &pins)
		if len(pins) != 14 {
			t.Fatalf("snapshot count %d", len(pins))
		}
		seen := map[string]bool{}
		for _, pin := range pins {
			isReview := pin.Path == "docs/reviews/r19-intersection-design-review.md"
			if seen[pin.Path] || !filepath.IsLocal(pin.Path) || (!strings.HasPrefix(pin.Path, archive) && !isReview) || len(pin.Commit) != 40 || len(pin.SourceSHA) != 64 || pin.Offset < 0 || pin.Length <= 0 {
				t.Fatalf("invalid snapshot: %+v", pin)
			}
			seen[pin.Path] = true
			if strings.HasPrefix(pin.Source, "docs/e2e/at/") {
				t.Fatalf("living AT source: %s", pin.Source)
			}
			switch pin.Kind {
			case "file":
				if isReview {
					if pin.Commit != "6048e099984ee03b69fec961309cd3eabe0e2b65" || pin.Source != ".agent/REVIEW.md" || pin.Offset != 0 || pin.Length != 41050 || pin.SourceSHA != pin.SHA256 {
						t.Fatal("r19 must be the whole original report")
					}
				} else if pin.Commit != cut || (pin.Source != "README.md" && pin.Source != "docs/ROADMAP.md" && !strings.HasPrefix(pin.Source, "docs/handover/") && !strings.HasPrefix(pin.Source, "docs/ops/")) {
					t.Fatalf("wrong owned snapshot source: %+v", pin)
				}
			case "commit_message":
				commits := map[string]string{"PR #383 merge message": "9f2b564b40a410624ca371062132061e5688be88", "PR #385 merge message": "a3bce68d86e22c44100f696b809246f76302964b"}
				if commits[pin.Source] != pin.Commit {
					t.Fatalf("wrong PR source: %+v", pin)
				}
			default:
				t.Fatalf("unknown snapshot kind: %s", pin.Kind)
			}
			checkHash(t, pin.Path, pin.SHA256)
			if len(readOwnedDoc(t, root, pin.Path)) != pin.Length {
				t.Errorf("snapshot size drift: %s", pin.Path)
			}
		}
		checkHash(t, "docs/reviews/r19-intersection-design-review.md", "14caf0193b256d6f090e8d53ff77e0d5a5c9dab4d025794a6812e15e83228913")
		for _, path := range []string{"docs/e2e/at/isolation/AT13.json", "docs/e2e/at/isolation/follow-up-14-findings.json", "docs/e2e/at/isolation/generation-holds-central.json.gz", "docs/e2e/at/isolation/producer-operations.jsonl.gz", "docs/e2e/at/members/AT02.md", "docs/e2e/at/members/AT02.json.gz", "docs/e2e/at/members/AT02.md.gz", "docs/e2e/at/members/AT02.md.gz.sha256.json", "docs/e2e/at/members/F15-AT02-recovery.json.gz", "docs/e2e/at/members/F15-AT02-policy-missing.json.gz", "tests/e2e/at/isolation/deadline_trace.go", "node/agent/observe.go"} {
			if claimTokenAllowed(path, "mutable implementation or evidence token") {
				t.Errorf("mutable pin allowed: %s", path)
			}
		}
		requireHandoverClaimTokens(t, root, doc, "56d20530", "e80bb592", "gzip files included", "isolation 15", "members 16", "cite by roadmap row / PR only", "claims test never reads those roots")
	})
	t.Run("followup26_assertions_decisions_and_other_findings_preserved", func(t *testing.T) {
		source := readOwnedDoc(t, root, "tests/handover/handover_test.go")
		start := strings.Index(source, "func TestHandover_Followup26Claims(")
		if start < 0 {
			t.Fatal("missing prior function")
		}
		end := strings.Index(source[start:], "// Live owned-document integrity remains separate")
		if end < 0 {
			t.Fatal("missing prior function end")
		}
		prior := restoreFollowup29Assertions(source[start : start+end])
		prior = strings.ReplaceAll(prior, followup28LegacyNormalization, "")
		prior = strings.ReplaceAll(prior, "const register = fixture + \"history/follow-up-27/register26.md\"", "const register = \"docs/handover/README.md\"")
		prior = strings.ReplaceAll(prior, "\n\t\t\t{register, []string{", "\n\t\t\t{\"docs/handover/README.md\", []string{")
		prior = strings.ReplaceAll(prior, "fixture+\"history/follow-up-27/entrypoint-\"+strings.ReplaceAll(path, \"/\", \"_\")+\".txt\"", "path")
		prior = strings.ReplaceAll(prior, "\t\t\troadmapPath := \"docs/ROADMAP.md\"\n\t\t\tif strings.HasSuffix(row, \"| done (#381) |\") {\n\t\t\t\troadmapPath = fixture + \"history/follow-up-27/roadmap381.md\"\n\t\t\t}\n\t\t\trequireHandoverClaimTokens(t, root, roadmapPath, row)", "\t\t\trequireHandoverClaimTokens(t, root, \"docs/ROADMAP.md\", row)")
		sum := sha256.Sum256([]byte(strings.TrimSpace(prior)))
		if hex.EncodeToString(sum[:]) != "1c5c5fb8392fe806ec65fd042b0f7f83c723b8ca528ea32d0d845918f8bc85d7" {
			t.Fatal("follow-up 26 assertions changed beyond frozen-input redirects")
		}
		checkHash(t, "docs/handover/follow-up-26.md", "07c369f855d3282d0a48966bff5bbb54524cb4bb0ef0c32a54f5f832160447ea")
		checkHash(t, fixture+"followup26-main.json", "dd46d62b5ab3ba9a5f3ac521a2f7673a1c0456796f0bb725cd061a4255c11f27")
		checkHash(t, fixture+"followup26-snapshots.json", "1e8d0728cdb2d249c7d92b695f180ea565d414dbb1b338467f5ed089e7a1842b")
		old := readOwnedDoc(t, root, archive+"register26.md")
		current := readOwnedDoc(t, root, register)
		decisions := regexp.MustCompile(`(?m)^\| D-[0-9]{3} \|.*$`)
		before, after := decisions.FindAllString(old, -1), decisions.FindAllString(current, -1)
		if len(after) != len(before)+2 || !slices.Equal(before[:len(before)-1], after[:len(before)-1]) || !strings.HasPrefix(after[len(before)-1], "| D-139 |") || !strings.Contains(after[len(before)-1], "now closed by D-141") || !strings.HasPrefix(after[len(before)], "| D-140 |") || !strings.HasPrefix(after[len(before)+1], "| D-141 |") {
			t.Fatal("decisions changed beyond D-139 closure and new D-140 / D-141")
		}
		oldFindings := strings.Split(strings.Split(old, "## Findings register at the follow-up 26 cutoff")[1], "\nEvery historical failure")[0]
		newFindings := strings.Split(strings.Split(current, "## Findings register at the follow-up 27 cutoff")[1], "\nEvery historical failure")[0]
		changed := map[string]bool{"F8-02 / native MTU 1280": true, "Members F11-01": true}
		count := 0
		for _, row := range strings.Split(oldFindings, "\n") {
			if !strings.HasPrefix(row, "| ") || strings.HasPrefix(row, "| Finding |") {
				continue
			}
			count++
			name := strings.TrimSpace(strings.Split(row, "|")[1])
			if !changed[name] && !strings.Contains(newFindings, row) {
				t.Errorf("previous finding changed: %s", row)
			}
		}
		if count != 39 {
			t.Fatalf("previous finding count %d", count)
		}
		added := 0
		for _, row := range strings.Split(newFindings, "\n") {
			if strings.HasPrefix(row, "| ") && !strings.HasPrefix(row, "| Finding |") {
				added++
			}
		}
		if added != 45 {
			t.Fatalf("expected six new finding rows, total %d", added)
		}
		requireHandoverClaimTokens(t, root, register, "Every other follow-up 26 finding is unchanged", "Every historical failure stays failed", "Members F14-01 stays failed historically", "| Isolation F14-01 | FAIL", "| Members 15 serving-node signature | OPEN", "| Harness stop-path gap | TEST ONLY", "| r18 M-3 | FIXED on both sides", "| Updates F13-01 | RECEIPT DELIVERED")
	})
	t.Run("seven_entrypoints_no_new_operator_guidance", func(t *testing.T) {
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			lines := strings.Split(readOwnedDoc(t, root, fixture+"history/follow-up-28/entrypoint-"+strings.ReplaceAll(path, "/", "_")+".txt"), "\n")
			if len(lines) < 4 || !strings.Contains(lines[2], "Current cut") || !strings.Contains(lines[2], "follow-up-27.md") || !strings.Contains(lines[2], "#385") || !strings.Contains(lines[3], "#382") {
				t.Errorf("stale entrypoint: %s", path)
			}
			requireHandoverClaimTokens(t, root, fixture+"history/follow-up-28/entrypoint-"+strings.ReplaceAll(path, "/", "_")+".txt", "No new operator guidance")
		}
		for path, want := range map[string]string{
			"README.md":                       "33cd7c4c211ac070f3f72f24ec5f257667bc4195825418f4a421bf7b21cbe223",
			"docs/handover/00-index.md":       "92c1ca425f333cc83e7b32d8d05cad19cdd970a201e7cf4e59deb4de5f5cd612",
			"docs/handover/deploy.md":         "f9f2195ba6678ebef7802ad25de2dd56769fe7ddf09e1d8aea9f3f30184b971f",
			"docs/handover/operator-guide.md": "34f56826a6080df72788aa90e436209476f493d4b628fb5a65515c66b7f3f17d",
			"docs/handover/member-guide.md":   "ca06333f288dcbeb51c300c1c91011dc53d08b8abaaf1faf4b75970bef3f8e18",
			"docs/handover/restore-drill.md":  "cc30c854377f1183709e39251d7af9a7a8b48181fe49df201630772a01273db8",
		} {
			lines := strings.Split(strings.ReplaceAll(readBeforeFollowup29Note(t, root, path), followup28Diagnostics, ""), "\n")
			if len(lines) < 4 {
				t.Fatalf("short entrypoint: %s", path)
			}
			sum := sha256.Sum256([]byte(strings.Join(lines[4:], "\n")))
			if hex.EncodeToString(sum[:]) != want {
				t.Errorf("operator content changed beyond entrypoint: %s", path)
			}
		}
		for pr := 383; pr <= 385; pr++ {
			requireHandoverClaimTokens(t, root, fixture+"history/follow-up-28/entrypoint-docs_ops_README.md.txt", fmt.Sprintf("#%d", pr))
		}
		requireHandoverClaimTokens(t, root, fixture+"history/follow-up-28/entrypoint-docs_ops_README.md.txt", "Follow-up 27 acceptance snapshot", "routed; in progress", "production reachability unknown", "Historical failures stay failed")
	})
	t.Log("follow-up 27: 19 subtests / 3 exact roadmap rows / 14 immutable snapshots / 182 owner archive file checks")
}

// Live owned-document integrity is separate from frozen acceptance claims.
func TestHandover_Followup27LiveDocumentIntegrity(t *testing.T) {
	root := repoRoot(t)
	var manifest struct {
		Followup struct {
			Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
		} `json:"handover_follow_up_27"`
	}
	if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Followup.Documents) != 21 {
		t.Fatalf("live document inventory %d", len(manifest.Followup.Documents))
	}
	for _, p := range manifest.Followup.Documents {
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, p.Path)))
		if hex.EncodeToString(sum[:]) != p.SHA256 {
			t.Errorf("live document hash drift: %s", p.Path)
		}
	}
}

// Exact source normalization keeps older whole-function assertion hashes intact.
const followup28LegacyNormalization = "\t\tprior = strings.ReplaceAll(prior, \"strings.ReplaceAll(readOwnedDoc(t, root, path), followup28Diagnostics, \\\"\\\")\", \"readOwnedDoc(t, root, path)\")\n"

const followup28Diagnostics = "**2026-11-01 — Node diagnostics (#386).** Adding a trace, or another foreign\nnft table or rule, to a node closes that node by design when it fails the\nfail-closed verifier: `ISOLATION_POLICY_INVALID` makes the agent install the\ndeny baseline until the rule is removed and serving verification succeeds.\nThe TEST ONLY unscoped `meta nftrace set 1` in a foreign `mangle` table caused\nthis in devstack 25; it was harness interference, not a product defect.\nDiagnose from the member side, or capture packets without touching the node's\nruleset. See the [dated devstack 26 analysis](../e2e/runs/2026-11-01/followup26/analysis.md).\n\n**Known traceroute behaviour (#389).** The fail-closed download source guard\ndrops RELATED ICMP errors (fragmentation needed or time exceeded) whose outer\nsource is in the spec deny list, even when the quoted connection is allowed.\nThat list includes 10/8, 172.16/12, 192.168/16 and 100.64/10. A member's\ntraceroute therefore shows no reply for on-path hops with those addresses.\nNo exception exists; the [owner question remains open](follow-up-28.md).\nThis records current behaviour and promises no fix.\n\n"

// TestHandover_Followup28Claims keeps the #386–#390 cut independent of living AT roots.
func TestHandover_Followup28Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "aae6763f9dd662091835f9219205b882e4382bfe"
	const fixture = "tests/handover/testdata/"
	const frozen = fixture + "history/follow-up-28/"
	const doc = "docs/handover/follow-up-28.md"
	const register = fixture + "history/follow-up-29/register28.md"
	const rows = fixture + "followup28-main.json"
	readJSON := func(t *testing.T, path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	checkHash := func(t *testing.T, path, want string) {
		t.Helper()
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
		if hex.EncodeToString(sum[:]) != want {
			t.Errorf("immutable file drift: %s", path)
		}
	}
	type claim struct {
		path   string
		tokens []string
	}
	for _, tc := range []struct {
		name   string
		claims []claim
	}{
		{"previous_cut_counts_and_381_no_action", []claim{
			{rows, []string{"done (#387)", "37db5405", "19 subtests / 3 exact roadmap rows / 14 immutable snapshots / 182 owner archive file checks", "history/follow-up-27/roadmap381.md"}},
			{doc, []string{"#383–#385 at 37db5405", "19 subtests / 3 exact roadmap rows /\n14 immutable snapshots / 182 owner archive file checks", "#381 row is byte-identical", "400d5cf91506f8b5fba942afa833924d839a16a9aabb8dddd9a5a80250eccbbc", "no further action"}},
			{"tests/handover/handover_test.go", []string{"TestHandover_Followup27Claims"}},
		}},
		{"devstack26_closed_node_is_harness_interference", []claim{
			{rows, []string{"done (#386)", "TEST ONLY node nft tracer", "meta nftrace set 1", "ISOLATION_POLICY_INVALID", "kernel generation 0 while central applied generation stays 5", "recovers at the same generation after removal"}},
			{frozen + "devstack26-closure.md", []string{"unscoped foreign `mangle`", "unchanged verifier", "verification_ISOLATION_POLICY_INVALID", "not a demonstrated product defect"}},
			{doc, []string{"TEST ONLY node tracer", "deny baseline", "harness artifact, not a product defect", "retroactively determine every historical timeout"}},
			{"tests/e2e/kernel_observer_test.go", []string{"TestAT01_HistoricalNodeTraceRejectedByServingVerifier"}},
		}},
		{"devstack26_native_delay_not_historical_causality", []claim{
			{rows, []string{"corrected member-only observation 4 / 0", "F14-style passive re-grants 17 / 1", "af718646", "5.671 s", "258ba504's cause stays unproven"}},
			{frozen + "devstack26-closure.md", []string{"loopback-source type-2 response", "handshake 0", "5.671 seconds", "does not retroactively assign 258ba504 a cause"}},
			{doc, []string{"member-only observation is 4 / 0", "re-grants are 17 / 1", "5.671 s", "native handshake / delivery delay of about five seconds", "`::1` mechanism", "now unblocked\nby #386"}},
			{register, []string{"| Members F11-01 | OPEN — native handshake / delivery delay", "Historical d556673b stays unattributed"}},
		}},
		{"devstack26_duplicate_chain_and_stop_budget", []claim{
			{rows, []string{"duplicate_chain", "errors.As", "ambiguous_listing", "supervisor 15 s after TERM", "PID1 forwards TERM / INT", "25 s compose grace"}},
			{frozen + "devstack26-stop.md", []string{"`errors.As`", "`duplicate_chain`", "monotonic 15 seconds", "TERM/INT handler", "parent receipt is required"}},
			{doc, []string{"`duplicate_chain`", "15 s budget", "PID1 TERM forwarding", "parent wait\nreceipt records the agent outcome"}},
			{"tests/e2e/internal/counterread/read_test.go", []string{"TestAT04_ParseFailureEvidence", "TestAT04_WrappedDuplicateClassification"}},
			{register, []string{"| Harness stop-path gap | FIXED — TEST ONLY, #386; adoption #390", "Historical SIGKILL / exit-137"}},
		}},
		{"r19_M2_reachable_applied_provenance_fixed", []claim{
			{rows, []string{"done (#388)", "M-2 is production-reachable", "due renewal signs lease h", "leaves g installed and serving", "ACK of g carrying h's live refs", "minimal applied snapshot", "committed atomically with the ACK", "reported only when it matches the applied generation"}},
			{frozen + "agent34.md", []string{"M-2 is reachable", "ErrBusy", "one SQLite transaction", "missing or mismatched", "null"}},
			{doc, []string{"r19 M-2 fixed; production-reachable", "minimal applied snapshot committed atomically\nwith the ACK", "only when they match the applied generation", "legacy refs stay null"}},
			{"node/agent/followup34_test.go", []string{"TestF34_ReportBeforeNewApplyOnBusyLock", "TestF34_AppliedSnapshotSurvivesNewDesiredAndRestart", "TestF34_MissingOrMismatchedSnapshotDoesNotBorrowNewDesired"}},
			{"node/agent/review_r19_intersection_test.go", []string{"TestReviewR19_Q3_ReportStampsLiveDesiredRefs"}},
		}},
		{"r19_L2_pre_Apply_reservation_and_explicit_cost", []claim{
			{rows, []string{"r19 L-2 fixed", "reserved in one transaction before Apply", "pre-delete failure consumes the reservation", "12-attempt bound", "1.1 ms (p99 1.59 ms)", "Steps with queued deletes", "12 / 12 pass with 6 qualified legs", "cost explicit"}},
			{frozen + "agent34-acceptance.md", []string{"accepted", "1.1 ms", "1.59 ms", "12 / 12", "6 legs qualified", "D-138", "price of a crash-safe 12-attempt bound"}},
			{doc, []string{"r19 L-2 fixed", "reserved\nin one transaction before Apply", "about 1.1 ms (p99 1.59 ms) before the restricting\ninstall on Steps with queued deletes", "cost\nexplicit"}},
			{"node/agent/followup34_test.go", []string{"TestF34_ReservationFailurePreventsApply", "TestF34_GenerationChangeKeepsPendingReservationCount"}},
			{"node/agent/review_r19_intersection_test.go", []string{"TestReviewR19_Q4_ConntrackAttemptIsCountedAfterTheDelete", "TestReviewR19_Q5_ConntrackCountSurvivesRestartAndClearResetsIt"}},
		}},
		{"r19_H1_source_guard_and_exact_DNS_exception", []claim{
			{rows, []string{"done (#389)", "r19 H-1 fixed", "current denied, host or resolver address", "before any generic established/related", "exact designated-resolver TCP/UDP source-port-53 replies stay ahead", "member, serving-probe and baseline-probe paths"}},
			{frozen + "netlab24.md", []string{"exact member\nand probe resolver replies", "current source denials", "then generic probe/member accepts", "wrong direction", "extra narrowing predicates"}},
			{doc, []string{"r19 H-1 fixed", "denied, host\nor resolver address", "before any generic established / related accept", "TCP / UDP source-port-53 replies stay ahead of the\nguard", "member, serving-probe and\nbaseline-probe paths"}},
			{"node/enforce/review_r19_intersection_test.go", []string{"TestReviewR19_Q4_CurrentRenderDoesNotRestrictEstablishedDownload"}},
			{"node/enforce/followup24_test.go", []string{"TestAT13_DownloadSourceVerification"}},
			{"tests/netlab/review_r19_destination_test.go", []string{"TestReviewR19_Q4_EstablishedDownloadSurvivesDestinationDeny"}},
			{"tests/netlab/followup24_test.go", []string{"TestAT13_ProbeDownloadDestinationDeny"}},
		}},
		{"ctdenyd_content_free_diagnostic_never_usage", []claim{
			{rows, []string{"content-free `diagnostic_v1` counter `ctdenyd`", "never enters usage", "Upload diagnostics keep their meanings"}},
			{frozen + "netlab24.md", []string{"no remote address", "`diagnostic_v1`, never device usage", "upload-only `ctdeny`, `cthost`, and `ctbadns` meanings are preserved"}},
			{doc, []string{"content-free", "`diagnostic_v1` counter `ctdenyd`", "never usage"}},
			{"node/enforce/followup24_test.go", []string{"TestMTR05_DownloadDenialIsDiagnostic"}},
		}},
		{"D138_ABAB_and_D120_two_failures_retained", []claim{
			{rows, []string{"A,B,A,B", "A2 on unchanged `main` failed D-120", "lost final datagram with no recorded successor", "stays failed"}},
			{frozen + "netlab24-acceptance.md", []string{"no B-only failure", "One quiet-window series cannot attribute", "D-138", "Accepted: H-1"}},
			{frozen + "netlab24-routing.md", []string{"Netlab follow-up 25", "A2 on `main` and the earlier branch attempt", "measurement-end question", "clause (iii) unchanged"}},
			{doc, []string{"A,B,A,B accepted under D-138; no B-only failure", "A2's D-120 failure\non unchanged main stays failed", "measurement-end gap", "netlab 25", "both failed runs stay failed"}},
		}},
		{"RELATED_ICMP_open_owner_question_not_decided", []claim{
			{frozen + "netlab24-icmp.md", []string{"SpecDeniedV4", "10/8, 172.16/12, 192.168/16, 100.64/10", "RELATED ICMP", "fragmentation-needed or time-exceeded", "quoted connection is allowed", "invented no exception", "question goes to the owner"}},
			{frozen + "netlab24.md", []string{"Spec question, unresolved", "does not invent it or claim private-router PMTU compatibility"}},
			{doc, []string{"Open owner question — RELATED ICMP, not a defect or a decision", "node/enforce/prefixes.go", "10/8, 172.16/12,\n192.168/16 and 100.64/10", "fragmentation needed\nor time exceeded", "No exception exists", "No fix is promised"}},
			{register, []string{"| RELATED ICMP outer source | OPEN OWNER QUESTION", "Not a defect, not decided, no fix promised"}},
		}},
		{"updates13_archive_and_historical_gap", []claim{
			{rows, []string{"done (#390)", "run 759d244a (48 root files)", "Updates F13-01 stays recorded for run 759d244a"}},
			{"docs/e2e/at/updates/history/follow-up-13/ARCHIVE.md", []string{"All 48 files", "759d244a-2dc4-460c-9f82-e37ce0ac7560", "historical missing first AT23 exit", "later receipts\ncannot reconstruct it"}},
			{doc, []string{"48 root files", "historical gap for run 759d244a; it was not filled", "unrelated to members\nF13-01"}},
			{"tests/e2e/at/updates/archive_test.go", []string{"TestD098_Followup13ArchivePins"}},
		}},
		{"updates14_new_run_receipts_and_living_root_boundary", []claim{
			{rows, []string{"Run 2704600f on #387's tree", "AT23–AT27 PASS 5 / 5", "installer 27 / 27", "13 result POSTs", "physical AT26 rollback", "uncached-fallback probe", "All 11 stop commands returned 0", "six agent terminations each have their own parent receipt", "exit 0, AT23's first included, a new sample", "five repeated stops reuse the prior receipt", "No SIGKILL attempt", "stored lease present at every stop and start", "Devstack 25's parent wait receipt and devstack 26's 15 s stop path adopted"}},
			{doc, []string{"Run 2704600f on #387's tree", "AT23–AT27 PASS 5 / 5; installer 27 / 27; 13 result POSTs", "AT26 physical rollback", "uncached-fallback probe", "All 11 stop commands returned 0", "Six agent terminations", "AT23's first as a new sample", "five repeated stops reuse the prior receipt", "No SIGKILL attempt", "stored lease was present at every stop and start", "living until updates 15 archives them", "roadmap row / PR only (D-132)"}},
			{"tests/e2e/at/updates/stopobserve/main_test.go", []string{"TestUPD05_StopObservationDistinguishesExitKillAndReap", "TestUPD05_ParentReceiptSurvivesReapRepeatAndRestart"}},
		}},
		{"decisions_provisional_and_seven_owner_requests", []claim{
			{doc, []string{"No new decision since D-141", "D-135, D-136, D-137, D-138, D-140 and D-141 stay\nprovisional", "### Requests to orchestrator", "2026-11-01 — **Owner decision 1: D-135 + D-136 confirmation", "Owner decision 2: D-137 hardening option", "Owner decision 3: D-138 overturn option", "Owner decision 4: D-140 overturn option", "Owner decision 5: D-141 overturn option", "Owner decision 6: F11-01 repair direction and upstream report", "Owner question 7: RELATED ICMP", "Seven owner requests", "no\noutside-owned pin needed changing"}},
			{register, []string{"No new decision since D-141", "D-135, D-136, D-137, D-138, D-140 and D-141 stay\nprovisional"}},
		}},
		{"remaining_findings_and_assigned_owners", []claim{
			{doc, []string{"Members 15's serving-node signature → devstack 27", "cause\nunproven", "design-only findings\nstay recorded under D-141", "Every other open\nfinding of follow-up 27 is unchanged", "Every historical failure stays failed", "usage 8, devstack 28 and agent-core\n35"}},
			{frozen + "netlab24-routing.md", []string{"usage follow-up 8", "devstack follow-up 28", "agent-core follow-up 35"}},
			{register, []string{"| r19 H-1 | FIXED — #389", "| r19 M-2 | FIXED — #388; production-reachable", "| r19 L-2 | FIXED — #388", "| Members 15 serving-node signature | OPEN", "Routed to devstack 27, measuring", "| r19 design-only | D-141 RECONSIDERATION CONDITIONS", "no populated hold was\nproduced"}},
			{"docs/reviews/README.md", []string{"r19 dispositions at the follow-up 28 cutoff", "Fixed — #389", "Fixed — #388", "Design-only findings stay recorded under provisional D-141"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.claims {
				requireHandoverClaimTokens(t, root, c.path, c.tokens...)
			}
		})
	}
	t.Run("cutoff_five_final_rows_and_381_byte_identity", func(t *testing.T) {
		var pin struct {
			Commit string
			Cutoff int `json:"cutoff_pr"`
			Rows   []string
			After  []int    `json:"merged_after_cutoff"`
			Absent []string `json:"absent_at_start"`
		}
		readJSON(t, rows, &pin)
		if pin.Commit != cut || pin.Cutoff != 390 || len(pin.Rows) != 5 || len(pin.After) != 0 || !slices.Equal(pin.Absent, []string{"w3-devstack-e2e follow-up 27", "w4-at-isolation follow-up 15"}) {
			t.Fatalf("wrong cutoff: %+v", pin)
		}
		for i, row := range pin.Rows {
			if !strings.HasSuffix(row, fmt.Sprintf("| done (#%d) |", 386+i)) {
				t.Fatalf("wrong PR order: %s", row)
			}
			requireHandoverClaimTokens(t, root, "docs/ROADMAP.md", row)
		}
		old := readOwnedDoc(t, root, fixture+"history/follow-up-27/roadmap381.md")
		checkHash(t, fixture+"history/follow-up-27/roadmap381.md", "400d5cf91506f8b5fba942afa833924d839a16a9aabb8dddd9a5a80250eccbbc")
		var found string
		for _, row := range strings.Split(readOwnedDoc(t, root, "docs/ROADMAP.md"), "\n") {
			if strings.HasSuffix(row, "| done (#381) |") {
				found = row + "\n"
			}
		}
		if found != old {
			t.Fatal("#381 live row differs from frozen snapshot")
		}
		requireHandoverClaimTokens(t, root, doc, cut, "#386–#390", "Nothing was merged after the cutoff at task start", "Devstack 27 and isolation\n15 were running", "Handover ran no new\nlive acceptance", "D-098 / D-109 / D-132")
	})
	t.Run("D098_1644_owner_archive_and_run_files", func(t *testing.T) {
		total := 0
		for _, manifest := range []struct {
			path, prefix, hash string
			count              int
			fullPath           bool
		}{
			{"docs/e2e/followup26-captures.sha256", "docs/e2e/runs/2026-11-01/followup26/", "064d59a3c1f7398b1e6bc5ffd86246d469502cc947b2d5d0db8e03138acbd272", 987, true},
			{"docs/e2e/at/updates/history/follow-up-13/SHA256SUMS", "docs/e2e/at/updates/history/follow-up-13/", "ee4ef48632b56cadbd138a1d00d398e418b978447b3e8b5332c04d3c6d5358d4", 48, false},
		} {
			checkHash(t, manifest.path, manifest.hash)
			lines := strings.Split(strings.TrimSpace(readOwnedDoc(t, root, manifest.path)), "\n")
			if len(lines) != manifest.count {
				t.Fatalf("wrong inventory count: %s / %d", manifest.path, len(lines))
			}
			seen := map[string]bool{}
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) != 2 || len(fields[0]) != 64 || !filepath.IsLocal(fields[1]) || seen[fields[1]] {
					t.Fatalf("invalid manifest entry: %s", line)
				}
				seen[fields[1]] = true
				path := manifest.prefix + fields[1]
				if manifest.fullPath {
					path = fields[1]
					if !strings.HasPrefix(path, manifest.prefix) {
						t.Fatalf("outside dated run: %s", path)
					}
				}
				checkHash(t, path, fields[0])
				total++
			}
		}
		const netlab = "docs/poc/G2-followup24-evidence/"
		checkHash(t, netlab+"retention-pins.json", "ccbb579ff6c205dc8a5fab159c7acf06be201d4efa7656fbb82562357904b0ee")
		var inventory struct {
			Files []struct {
				Path, SHA256 string
				Bytes        int
			}
		}
		readJSON(t, netlab+"retention-pins.json", &inventory)
		if len(inventory.Files) != 609 {
			t.Fatalf("netlab inventory %d", len(inventory.Files))
		}
		seen := map[string]bool{}
		for _, f := range inventory.Files {
			if !filepath.IsLocal(f.Path) || seen[f.Path] || len(f.SHA256) != 64 {
				t.Fatalf("invalid netlab entry: %+v", f)
			}
			seen[f.Path] = true
			checkHash(t, netlab+f.Path, f.SHA256)
			if len(readOwnedDoc(t, root, netlab+f.Path)) != f.Bytes {
				t.Errorf("netlab size drift: %s", f.Path)
			}
			total++
		}
		if total != 1644 {
			t.Fatalf("archive and run file count %d", total)
		}
	})
	t.Run("D098_D109_D132_16_snapshots_no_living_roots", func(t *testing.T) {
		checkHash(t, rows, "f51a4bb2323b7fe71e8d5af82b108ffc3c7a664e3b78f0cb0991b234b7e606f1")
		checkHash(t, fixture+"followup28-snapshots.json", "e3129b05f907d3d0c3376f4b14ea9d012039867e07f2acc4cc6eb0d8182010bf")
		var pins []struct {
			Path, Source, SHA256 string
			Commit               string `json:"source_commit"`
			Kind                 string `json:"source_kind"`
			SourceSHA            string `json:"source_sha256"`
			Offset, Length       int
		}
		readJSON(t, fixture+"followup28-snapshots.json", &pins)
		if len(pins) != 16 {
			t.Fatalf("snapshot count %d", len(pins))
		}
		seen := map[string]bool{}
		for _, p := range pins {
			if !filepath.IsLocal(p.Path) || !strings.HasPrefix(p.Path, frozen) || seen[p.Path] || len(p.Commit) != 40 || len(p.SourceSHA) != 64 || p.Offset < 0 || p.Length <= 0 {
				t.Fatalf("bad snapshot: %+v", p)
			}
			seen[p.Path] = true
			if strings.HasPrefix(p.Source, "docs/e2e/at/") {
				t.Fatalf("living AT snapshot source: %s", p.Source)
			}
			switch p.Kind {
			case "file":
				allowed := p.Source == "README.md" || strings.HasPrefix(p.Source, "docs/handover/") || p.Source == "docs/ops/README.md" || p.Source == "docs/node/followup34-applied-refs-conntrack-reservation-2026-11-01.md" || p.Source == "docs/poc/G2-followup24-2026-11-01.md" || p.Source == "docs/e2e/runs/2026-11-01/followup26/analysis.md"
				if p.Commit != cut || !allowed {
					t.Fatalf("wrong snapshot source: %+v", p)
				}
			case "commit_message":
				commits := map[string]string{"PR #388 merge message": "7981ff26f298736da9b1871536a0575a8a99cefa", "PR #389 merge message": "3483aa6205ec3902f81fa40deb474adb0ef79241"}
				if commits[p.Source] != p.Commit {
					t.Fatalf("wrong PR hand-off: %+v", p)
				}
			default:
				t.Fatalf("unknown source kind: %s", p.Kind)
			}
			checkHash(t, p.Path, p.SHA256)
			if len(readOwnedDoc(t, root, p.Path)) != p.Length {
				t.Errorf("snapshot size drift: %s", p.Path)
			}
		}
		for _, path := range []string{"docs/e2e/at/updates/run.json", "docs/e2e/at/updates/agent-stops.json", "docs/e2e/at/updates/AT23.json", "docs/e2e/at/updates/summary.md", "docs/e2e/at/updates/restart-leases.json", "docs/e2e/at/updates/result-delivery.json"} {
			if claimTokenAllowed(path, "living result") {
				t.Errorf("living root allowed: %s", path)
			}
		}
	})
	t.Run("prior_assertions_and_unchanged_decisions_findings", func(t *testing.T) {
		source := readOwnedDoc(t, root, "tests/handover/handover_test.go")
		start := strings.Index(source, "func TestHandover_Followup27Claims(")
		if start < 0 {
			t.Fatal("missing prior function")
		}
		end := strings.Index(source[start:], "// Live owned-document integrity is separate from frozen acceptance claims.")
		if end < 0 {
			t.Fatal("missing prior function boundary")
		}
		prior := restoreFollowup29Assertions(source[start : start+end])
		prior = strings.ReplaceAll(prior, "\t\tprior = strings.ReplaceAll(prior, followup28LegacyNormalization, \"\")\n", "")
		prior = strings.ReplaceAll(prior, "const register = fixture + \"history/follow-up-28/register27.md\"", "const register = \"docs/handover/README.md\"")
		prior = strings.ReplaceAll(prior, "\n\t\t\t{register, []string{", "\n\t\t\t{\"docs/handover/README.md\", []string{")
		prior = strings.ReplaceAll(prior, "fixture+\"history/follow-up-28/entrypoint-\"+strings.ReplaceAll(path, \"/\", \"_\")+\".txt\"", "path")
		prior = strings.ReplaceAll(prior, "fixture+\"history/follow-up-28/entrypoint-docs_ops_README.md.txt\"", "\"docs/ops/README.md\"")
		prior = strings.ReplaceAll(prior, "strings.ReplaceAll(readOwnedDoc(t, root, path), followup28Diagnostics, \"\")", "readOwnedDoc(t, root, path)")
		sum := sha256.Sum256([]byte(strings.TrimSpace(prior)))
		if hex.EncodeToString(sum[:]) != "9cf1aa0336349d02e209893693a0c58a417db82254082a8dfbf914f62fc8f0b1" {
			t.Fatal("follow-up 27 assertions changed beyond frozen-input redirects / exact note exclusion")
		}
		checkHash(t, "docs/handover/follow-up-27.md", "74f0dec53f7da87425811797cea67a8fb2944ce8ee719da25a7772cfcdf3bee4")
		checkHash(t, fixture+"followup27-main.json", "14c1e6c4f05fcee8ecce00b9ad3193f4c36698b7c16788b43a8a7ad6d041a6fc")
		checkHash(t, fixture+"followup27-snapshots.json", "e2769ac1dd73367f8c2c1fc5f695145138d76accacf3c37cc02c69a9744096b8")
		old, current := readOwnedDoc(t, root, frozen+"register27.md"), readOwnedDoc(t, root, register)
		decisions := regexp.MustCompile(`(?m)^\| D-[0-9]{3} \|.*$`)
		if !slices.Equal(decisions.FindAllString(old, -1), decisions.FindAllString(current, -1)) {
			t.Fatal("no new or changed decision since D-141")
		}
		oldFindings := strings.Split(strings.Split(old, "## Findings register at the follow-up 27 cutoff")[1], "\nEvery historical failure")[0]
		newFindings := strings.Split(strings.Split(current, "## Findings register at the follow-up 28 cutoff")[1], "\nEvery historical failure")[0]
		changed := map[string]bool{"Members F11-01": true, "Updates F13-01": true, "Harness stop-path gap": true, "Members 15 serving-node signature": true, "r19 H-1": true, "r19 M-2": true, "r19 L-2": true}
		count := 0
		for _, row := range strings.Split(oldFindings, "\n") {
			if !strings.HasPrefix(row, "| ") || strings.HasPrefix(row, "| Finding |") {
				continue
			}
			count++
			name := strings.TrimSpace(strings.Split(row, "|")[1])
			if !changed[name] && !strings.Contains(newFindings, row) {
				t.Errorf("unaffected finding changed: %s", row)
			}
		}
		if count != 45 {
			t.Fatalf("old findings %d", count)
		}
		if got := len(regexp.MustCompile(`(?m)^\| [^|]+ \|`).FindAllString(newFindings, -1)) - 1; got != 47 {
			t.Fatalf("current findings %d", got)
		}
	})
	t.Run("seven_entrypoints_and_exact_operator_diagnostics", func(t *testing.T) {
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			lines := strings.Split(readOwnedDoc(t, root, fixture+"history/follow-up-29/entrypoint-"+strings.ReplaceAll(path, "/", "_")+".txt"), "\n")
			if len(lines) < 4 || !strings.Contains(lines[2], "Current cut") || !strings.Contains(lines[2], "follow-up-28.md") || !strings.Contains(lines[2], "#390") || !strings.Contains(lines[3], "follow-up-27.md") || !strings.Contains(lines[3], "#385") {
				t.Errorf("stale entrypoint: %s", path)
			}
		}
		op := readOwnedDoc(t, root, "docs/handover/operator-guide.md")
		a, b, note := strings.Index(op, "### 3.4 "), strings.Index(op, "### 3.5 "), strings.Index(op, followup28Diagnostics)
		if a < 0 || note <= a || b <= note || strings.Count(op, followup28Diagnostics) != 1 {
			t.Fatal("diagnostic note missing, repeated or outside node diagnostics")
		}
		requireHandoverClaimTokens(t, root, "docs/handover/operator-guide.md", followup28Diagnostics)
		for pr := 386; pr <= 390; pr++ {
			requireHandoverClaimTokens(t, root, fixture+"history/follow-up-29/entrypoint-docs_ops_README.md.txt", fmt.Sprintf("#%d", pr))
		}
		requireHandoverClaimTokens(t, root, fixture+"history/follow-up-29/entrypoint-docs_ops_README.md.txt", "Follow-up 28 acceptance snapshot", "RELATED ICMP question is open, not a defect or decision")
	})
	t.Log("follow-up 28: 19 subtests / 5 exact roadmap rows / 16 immutable snapshots / 1644 archive and run file checks")
}

// Live integrity remains separate from the immutable acceptance cut.
func TestHandover_Followup28LiveDocumentIntegrity(t *testing.T) {
	root := repoRoot(t)
	var manifest struct {
		Followup struct {
			Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
		} `json:"handover_follow_up_28"`
	}
	if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Followup.Documents) != 22 {
		t.Fatalf("live document inventory %d", len(manifest.Followup.Documents))
	}
	for _, p := range manifest.Followup.Documents {
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, p.Path)))
		if hex.EncodeToString(sum[:]) != p.SHA256 {
			t.Errorf("live document hash drift: %s", p.Path)
		}
	}
}

// Only this exact new operator note is excluded by older body-hash checks.
const followup29Configuration = "**2026-11-01 — Known local configuration behaviour at #396.** After a restart\nwith changed `extra_denied_v4` or `host_v4`, same-boot adoption can keep the node\nserving on the previously installed `denied_v4` / `host_v4` until its first\nsuccessful Apply. A larger deny set can therefore leave the old, smaller set\nin force. A busy node lock or control timeout can delay Apply; the verifier\ndoes not read these sets' elements. SIGHUP does not reload these fields.\nThe [agent-core 35 dated hand-off](../node/followup35-diagnostic-and-restart-2026-11-01.md)\nprovides no safe procedure and no maximum adoption-to-Apply delay. The fix is\nrouted to agent-core 36 (refuse adoption on changed sets), then netlab 26\n(verifier content check); it is open at this cutoff. No workaround is promised.\n\n"

func readBeforeFollowup29Note(t *testing.T, root, path string) string {
	t.Helper()
	return strings.ReplaceAll(readOwnedDoc(t, root, path), followup29Configuration, "")
}

// Reverse only this cut's input redirects / exact-note exclusion before checking
// earlier whole-function hashes. Their expected hashes and assertions stay intact.
func restoreFollowup29Assertions(source string) string {
	return strings.NewReplacer(
		"readBeforeFollowup29Note(t, root, path)", "readOwnedDoc(t, root, path)",
		"restoreFollowup29Assertions(source[start : start+end])", "source[start : start+end]",
		`const register = fixture + "history/follow-up-29/register28.md"`, `const register = "docs/handover/README.md"`,
		`fixture+"history/follow-up-29/entrypoint-"+strings.ReplaceAll(path, "/", "_")+".txt"`, "path",
		`fixture+"history/follow-up-29/entrypoint-docs_ops_README.md.txt"`, `"docs/ops/README.md"`,
	).Replace(source)
}

// TestHandover_Followup29Claims pins #391–#396, never current AT root evidence.
func TestHandover_Followup29Claims(t *testing.T) {
	root := repoRoot(t)
	const cut = "d640ce96d9145e57a60e04e5875e049070f30b96"
	const fixture = "tests/handover/testdata/"
	const frozen = fixture + "history/follow-up-29/"
	const doc = "docs/handover/follow-up-29.md"
	const register = "docs/handover/README.md"
	const rows = fixture + "followup29-main.json"
	const agent = "docs/node/followup35-diagnostic-and-restart-2026-11-01.md"
	const dev = "docs/e2e/runs/2026-11-01/followup27/"
	const netlab = "docs/poc/G2-followup25-evidence/"
	readJSON := func(t *testing.T, path string, dst any) {
		t.Helper()
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, path)), dst); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	checkHash := func(t *testing.T, path, want string) {
		t.Helper()
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
		if hex.EncodeToString(sum[:]) != want {
			t.Errorf("immutable file drift: %s", path)
		}
	}
	type claim struct {
		path   string
		tokens []string
	}
	for _, tc := range []struct {
		name   string
		claims []claim
	}{
		{"previous_cut_19_subtests_47_findings_seven_requests", []claim{
			{rows, []string{"done (#395)", "19 subtests, five final rows, 16 immutable snapshots, 1,644 archived owner-file checks", "Findings register at 47 rows", "7 requests (owner decisions)"}},
			{doc, []string{"#386–#390 at aae6763f", "19 subtests / 5 exact roadmap rows /\n16 immutable snapshots / 1,644 archive and run file checks", "47 rows", "seven requests are owner decisions"}},
			{"tests/handover/handover_test.go", []string{"TestHandover_Followup28Claims"}},
		}},
		{"usage8_nonzero_diagnostic_and_complete_download_order", []claim{
			{rows, []string{"done (#391)", "80 packets / 80,000 B", "identical samples with and without it, and none from it alone", "complete statements", "`nd_` replies", "`denied_v4`, `host_v4` and `resolver_v4` source guards", "`d_` admission", "no live run owed", "b8a6e9e1"}},
			{doc, []string{"request 1 is closed as coverage", "no samples from it alone", "Complete rendered\nstatements", "There was no live run", "final #391 roadmap row (D-132)"}},
			{"tests/e2e/at/usage/at14_16_node_test.go", []string{"TestAT14_ForwardAdmissionMetering_Mechanism"}},
		}},
		{"isolation14_archive_and_isolation15_run_boundary", []claim{
			{rows, []string{"done (#392)", "archive of run 56d20530 (117 root files and subtrees)", "Run 0b9e4510", "AT10 PASS 9 / 0 / 0 (ISO-F6-01 6 / 6)", "AT12 PASS 10 / 0 / 0", "AT11 and AT13 blocked on operator rows"}},
			{doc, []string{"0b9e4510 on #388's tree", "follow-up-14/archive.json", "follow-up-15-stream/archive.json", "final #392 roadmap row", "living until isolation 16 archives them (D-132)"}},
			{"tests/e2e/at/isolation/producer_timing_test.go", []string{"TestD098_Followup14ArchivePins"}},
		}},
		{"D140_evaluator_pass_is_not_qualification", []claim{
			{rows, []string{"byte-identical `EvaluateDeadline` passes", "last positive D−98.630 ms, first drop D−97.598 ms", "packet 171's egress followed packet 172's ingress by 3.821 µs", "row is not qualified", "Isolation F14-01 stays failed"}},
			{doc, []string{"no-interleaving", "failed qualification", "whole-node row stays **unqualified**", "Isolation F14-01 (run 56d20530) stays failed", "next sample belongs to\nisolation 16"}},
			{"tests/e2e/at/isolation/producer_timing_test.go", []string{"TestAT13_D140InterleavingQualification"}},
			{"tests/e2e/at/isolation/deadline_trace_test.go", []string{"TestAT13_D104StrictBoundaryAndMissingEvidence"}},
			{register, []string{"| Isolation 15 D-140 sample | UNQUALIFIED — failed qualification", "| Isolation F14-01 | FAIL"}},
		}},
		{"isolation15_parent_signal_receipt_and_no_hold", []claim{
			{rows, []string{"three agent exits 0", "terminal source's agent ended by signal 15, cause unproven", "No stale-lease hold formed", "populated hold is not exercised"}},
			{doc, []string{"signal 15, with no exit code", "cause is unproven and routed to agent-core 36", "not exercise the populated hold"}},
			{register, []string{"| Isolation 15 signal-15 receipt | OPEN — cause unproven", "Not relabelled cooperative"}},
		}},
		{"agent35_all_diagnostic_consumers_without_production_change", []claim{
			{rows, []string{"done (#393)", "meter selection, reset matcher, persisted readings, usage snapshot and observation consumers covered", "no production change"}},
			{agent, []string{"Production code is unchanged", "Raw Apply persistence", "Usage persistence and reporting", "No path from `ctdenyd` to member usage was found"}},
			{"node/internal/meter/followup35_test.go", []string{"TestAT14_F35_CtdenydUsageSelection", "TestAT16_F35_CtdenydDoesNotChangePersistedUsageOrEpoch"}},
			{"node/agent/followup35_test.go", []string{"TestAT16_F35_CtdenydResetAndReportConsumers", "TestAT14_F35_CtdenydCounterSamples", "TestAT16_F35_CtdenydApplyPersistenceDoesNotBecomeUsage"}},
			{doc, []string{"meter selection, the reset matcher", "usage snapshot and observation consumers", "No production change"}},
		}},
		{"agent35_same_boot_adoption_is_production_reachable", []claim{
			{rows, []string{"never reads `denied_v4` / `host_v4` contents", "serving can resume on the old, smaller sets until that Apply", "busy lock or control timeout can delay", "TestAT13_F35_RestartFirstApplyRendersConfiguredDenyGrowth/same_boot_adoption"}},
			{agent, []string{"first full Apply after restart renders the new sets", "no guarantee that serving waits for that Apply", "does **not compare denied/host set contents", "current unsafe pre-Apply serving behavior", "No new maximum adoption-to-Apply"}},
			{"node/agent/followup35_restart_test.go", []string{"TestAT13_F35_RestartFirstApplyRendersConfiguredDenyGrowth"}},
			{doc, []string{"New open, production-reachable finding", "first successful Apply", "first Apply's render proof", "no safe procedure"}},
			{register, []string{"| Agent-core 35 same-boot adoption | OPEN — production-reachable, #393"}},
		}},
		{"two_step_route_and_unconfirmed_live_drift", []claim{
			{rows, []string{"routed to agent-core 36 then netlab 26"}},
			{doc, []string{"agent-core 36 refuses adoption on changed\nsets, then netlab 26 adds the verifier's content check", "open at\n#396", "New unconfirmed finding", "set changed\nlive under a serving agent go undetected", "netlab 26 is proving that reading"}},
			{register, []string{"| Agent-core 35 live set drift | UNCONFIRMED", "No confirmed live-drift or traffic result"}},
		}},
		{"devstack27_bounded_matching_503_without_signed_mutation", []claim{
			{rows, []string{"done (#394)", "ArmMatchingLeaseHold", "MatchingLeaseHold", "ReleaseMatchingLeaseHold", "GenerationHoldSnapshot", "LocalLeaseReadback", "never edits, replays or synthesizes a signed document"}},
			{dev + "analysis.md", []string{"TEST ONLY proxy", "returning prompt 503s for successful renewal responses matching that generation", "No signed state, lease or\nreport is forged, replayed or synthesized"}},
			{doc, []string{"503 only to successful matching lease-renewal responses", "For a bounded time", "TEST ONLY"}},
			{"tests/e2e/cmd/fvpn-e2e-nodeconfig/lease_hold_test.go", []string{"TestAT13_MatchingLeaseTransportHold"}},
		}},
		{"devstack27_alarm_escalates_once_then_generation_matched", []claim{
			{rows, []string{"13 / 0 / 0", "two members", "50.000298 s", "Step-3 alarm", "9.887 s (alarm-triggered)", "generation_matched"}},
			{dev + "hold-fixture.md", []string{"13 pass / 0 blocked / 0 fail", "50.000298 s", "9.887144 s", "Step 3", "the reported alarm triggered escalation", "not the age fallback", "no second audit", "two original runners stay **11 / 0 / 1**"}},
			{doc, []string{"Step-3 report carried the alarm", "not the 30 s fallback", "Central escalated **once**", "Recovery ended the incident **`generation_matched`**", "earlier fixture\nrunner failures stay failed"}},
		}},
		{"hold_fixture_does_not_pass_consumer_rows", []claim{
			{dev + "hold-fixture.md", []string{"unrelated-failure precedence", "agent-stop retention", "distinct later incident", "sole-member `members_revoked` closure", "None of those rows is passed"}},
			{doc, []string{"unrelated-failure precedence,\nactive-stop retention, recovery inside a full row, a distinct later incident,\nand sole-member `members_revoked` closure"}},
			{register, []string{"| r18 M-3 | FIXED on both sides — #376 + #382; fixture proof #394", "Consumer rows remain open"}},
		}},
		{"serving_campaign_counts_anomalies_and_unproven_members_causes", []claim{
			{rows, []string{"8 / 8 first selections", "58,619 sends with zero missing node ingress", "183 ingress-without-origin losses counted", "IPv6 `::1`-source type-4 data", "type-2 responses", "no historical F15-F11-01 / F15-AT02 / F14-01 cause assigned"}},
			{dev + "analysis.md", []string{"Neither historical five-SYN failure nor the five-second\nmissing-ingress gap reproduced", "F15-F11-01, F15-AT02 and archived F14-01 remain byte-identical", "183 ingress-without-origin\nlosses are retained", "eight type-4 data messages", "another type-2 response"}},
			{doc, []string{"request 2", "owner's **F11-01 evidence**", "their files are byte-identical and their rows are unchanged", "No cause is assigned"}},
			{register, []string{"| Members 15 serving-node signature | OPEN", "cause unproven", "#394 request 2 adds IPv6 ::1-source"}},
		}},
		{"D120_measurement_end_guard_successor_and_watchdog", []claim{
			{rows, []string{"done (#396)", "stream ends only after the final generation-1100 guard bracket closes", "at most one more TCP / UDP cycle", "15 s parent watchdog fails incomplete evidence"}},
			{netlab + "overview.md", []string{"Only then does it request the stream boundary", "record exactly one more TCP/UDP cycle", "failed successor remains", "15-second watchdog", "Expiry fails the run and preserves incomplete evidence"}},
			{doc, []string{"only after the final guard bracket closes", "at most\none successor cycle after a lost tail", "15 s parent watchdog fails an incomplete measurement"}},
		}},
		{"D120_unchanged_analyzer_and_every_historical_failure", []claim{
			{rows, []string{"`classify_d120`, its tests", "byte-identical", "(iii): next datagram missing"}},
			{netlab + "overview.md", []string{"entire existing test file, byte-identical", "generation 1087, sequence 57", "A2, the earlier branch attempt, and `gate2-replay` remain failed"}},
			{netlab + "a2-after-verdict.txt", []string{"1087", "57", "(iii): next datagram missing"}},
			{doc, []string{"classify_d120", "byte-identical", "A2, the earlier branch attempt and `gate2-replay` stay failed"}},
			{register, []string{"| Netlab D-120 missing successor | MEASUREMENT END FIXED — #396; failed runs retained"}},
		}},
		{"D120_one_quiet_live_run_offline_only_lost_tail", []claim{
			{rows, []string{"One predeclared live run on a quiet host", "100 reloads, 101 verified Applies", "zero losses, zero extra cycles"}},
			{netlab + "overview.md", []string{"zero TCP timeouts", "no lost tail and used zero extra cycles", "offline stream-loop proofs", "13,646,265 original bytes", "720,796 compressed bytes", "Both hashes are in the manifest"}},
			{doc, []string{"One live run on a quiet host", "zero UDP\nlosses, zero TCP timeouts and zero extra cycles", "proven offline only", "SHA-256 and length for both forms", "13,646,265 → 720,796 bytes"}},
		}},
		{"unchanged_decisions_and_seven_owner_requests", []claim{
			{doc, []string{"No decision was added after D-141 on main", "D-135, D-136, D-137, D-138,\nD-140 and D-141 stay provisional", "### Requests to orchestrator", "Owner decision 1: D-135 + D-136 confirmation", "Owner decision 2: D-137 hardening option", "Owner decision 3: D-138 overturn option", "Owner decision 4: D-140 overturn option", "Owner decision 5: D-141 overturn option", "Owner decision 6: F11-01 repair direction and upstream report", "Owner question 7: RELATED ICMP", "Seven owner\nrequests", "No outside-owned pin needed changing"}},
			{register, []string{"No new decision since D-141", "D-135, D-136, D-137, D-138, D-140 and D-141 stay\nprovisional", "Every other follow-up 28 finding is unchanged", "Every historical failure stays failed"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.claims {
				requireHandoverClaimTokens(t, root, c.path, c.tokens...)
			}
		})
	}
	t.Run("six_final_rows_fixed_cut_and_no_later_results", func(t *testing.T) {
		var pin struct {
			Commit string
			Cutoff int `json:"cutoff_pr"`
			Rows   []string
			After  []int    `json:"merged_after_cutoff"`
			Absent []string `json:"absent_at_start"`
		}
		readJSON(t, rows, &pin)
		if pin.Commit != cut || pin.Cutoff != 396 || len(pin.Rows) != 6 || len(pin.After) != 0 || !slices.Equal(pin.Absent, []string{"w2-agent-core follow-up 36", "w4-at-isolation follow-up 16", "w4-at-members follow-up 16", "w3-devstack-e2e follow-up 28", "w1-netlab follow-up 26"}) {
			t.Fatalf("wrong cutoff: %+v", pin)
		}
		for i, row := range pin.Rows {
			if !strings.HasSuffix(row, fmt.Sprintf("| done (#%d) |", 391+i)) {
				t.Fatalf("nonfinal or misordered row: %s", row)
			}
			requireHandoverClaimTokens(t, root, "docs/ROADMAP.md", row)
		}
		requireHandoverClaimTokens(t, root, doc, cut, "#391–#396", "Nothing was merged after the cutoff at task start", "their results are outside\nthis cut", "Handover ran no new live acceptance", "D-098 / D-109 / D-132")
	})
	t.Run("eight_snapshots_and_no_living_evidence_inputs", func(t *testing.T) {
		checkHash(t, rows, "16ba19c231e1f05953d33004e144171a78a87b294ddf43ccc18a482d3307a166")
		checkHash(t, fixture+"followup29-snapshots.json", "757f857fd1cd52b05283376d605abbd9c3bbed1710d2117ee5f09c0ad314fce3")
		var pins []struct {
			Path, Source, SHA256 string
			Commit               string `json:"source_commit"`
			SourceSHA            string `json:"source_sha256"`
			Offset, Length       int
		}
		readJSON(t, fixture+"followup29-snapshots.json", &pins)
		if len(pins) != 8 {
			t.Fatalf("snapshot count %d", len(pins))
		}
		seen := map[string]bool{}
		for _, p := range pins {
			allowed := p.Source == "README.md" || strings.HasPrefix(p.Source, "docs/handover/") || p.Source == "docs/ops/README.md"
			if !allowed || !filepath.IsLocal(p.Path) || !strings.HasPrefix(p.Path, frozen) || seen[p.Path] || p.Commit != cut || len(p.SourceSHA) != 64 || p.Offset < 0 || p.Length <= 0 {
				t.Fatalf("invalid snapshot provenance: %+v", p)
			}
			seen[p.Path] = true
			checkHash(t, p.Path, p.SHA256)
			if len(readOwnedDoc(t, root, p.Path)) != p.Length {
				t.Errorf("snapshot length: %s", p.Path)
			}
		}
		for _, path := range []string{"docs/e2e/at/usage/AT14.json", "docs/e2e/at/usage/summary.md", "docs/e2e/at/isolation/summary.md", "docs/e2e/at/isolation/boundary-review.json", "docs/e2e/at/isolation/stop-receipt-review.json", "docs/e2e/at/isolation/AT13.json", "docs/e2e/RESPONSES.md"} {
			if claimTokenAllowed(path, "living evidence") {
				t.Errorf("D-132 living root allowed: %s", path)
			}
		}
	})
	t.Run("D098_731_archive_files_and_lossless_compressed_forms", func(t *testing.T) {
		checkHash(t, fixture+"followup29-evidence.json", "c04ef7a1966dfd554ae76447bc256323fe4cbbc6dc17aaee9783877ee69aaa8b")
		var inventory struct {
			Commit    string `json:"source_commit"`
			Documents []struct {
				Path, SHA256 string
				Bytes        int
			}
			Archives []struct {
				Path, Prefix, Format, SHA256 string
				Count                        int
			}
		}
		readJSON(t, fixture+"followup29-evidence.json", &inventory)
		if inventory.Commit != cut || len(inventory.Documents) != 4 || len(inventory.Archives) != 4 {
			t.Fatal("wrong evidence inventory")
		}
		for _, d := range inventory.Documents {
			if !claimTokenAllowed(d.Path, "immutable evidence") {
				t.Fatalf("invalid dated evidence %s", d.Path)
			}
			checkHash(t, d.Path, d.SHA256)
			if len(readOwnedDoc(t, root, d.Path)) != d.Bytes {
				t.Errorf("dated evidence length: %s", d.Path)
			}
		}
		total := 0
		for _, a := range inventory.Archives {
			checkHash(t, a.Path, a.SHA256)
			type entry struct {
				Path, SHA256       string
				Bytes              *int64
				CompressedBytes    *int64 `json:"compressed_bytes"`
				UncompressedSHA256 string `json:"uncompressed_sha256"`
				UncompressedBytes  *int64 `json:"uncompressed_bytes"`
				DecompressedSHA256 string `json:"decompressed_sha256"`
				DecompressedBytes  *int64 `json:"decompressed_bytes"`
			}
			var manifest struct{ Files []entry }
			if a.Format == "sha256" {
				for _, line := range strings.Split(strings.TrimSpace(readOwnedDoc(t, root, a.Path)), "\n") {
					fields := strings.Fields(line)
					if len(fields) != 2 || !strings.HasPrefix(fields[1], a.Prefix) {
						t.Fatalf("bad manifest line: %s", line)
					}
					manifest.Files = append(manifest.Files, entry{Path: strings.TrimPrefix(fields[1], a.Prefix), SHA256: fields[0]})
				}
			} else if a.Format == "json" {
				readJSON(t, a.Path, &manifest)
			} else {
				t.Fatalf("unknown manifest format %s", a.Format)
			}
			if len(manifest.Files) != a.Count {
				t.Fatalf("archive count %s: %d", a.Path, len(manifest.Files))
			}
			seen := map[string]bool{}
			for _, f := range manifest.Files {
				if !filepath.IsLocal(f.Path) || seen[f.Path] || len(f.SHA256) != 64 {
					t.Fatalf("bad archive entry: %+v", f)
				}
				seen[f.Path] = true
				path := a.Prefix + f.Path
				checkHash(t, path, f.SHA256)
				b := []byte(readOwnedDoc(t, root, path))
				for _, size := range []*int64{f.Bytes, f.CompressedBytes} {
					if size != nil && int64(len(b)) != *size {
						t.Errorf("archive length: %s", path)
					}
				}
				want, size := f.UncompressedSHA256, f.UncompressedBytes
				if f.DecompressedSHA256 != "" {
					want, size = f.DecompressedSHA256, f.DecompressedBytes
				}
				if want != "" {
					if size == nil || !strings.HasSuffix(path, ".gz") {
						t.Fatalf("incomplete gzip pin: %s", path)
					}
					reader, err := gzip.NewReader(bytes.NewReader(b))
					if err != nil {
						t.Fatal(err)
					}
					h := sha256.New()
					n, copyErr := io.Copy(h, reader)
					closeErr := reader.Close()
					if copyErr != nil || closeErr != nil || n != *size || hex.EncodeToString(h.Sum(nil)) != want {
						t.Errorf("lossless gzip drift: %s (%v, %v)", path, copyErr, closeErr)
					}
				}
				if path == netlab+"live-attempt1/TestAT10_AT13_ReloadRevokeBaseline/at10-apply.jsonl.gz" && (len(b) != 720796 || size == nil || *size != 13646265) {
					t.Fatal("Apply JSONL both-form lengths changed")
				}
				total++
			}
		}
		if total != 731 {
			t.Fatalf("archive/run check count %d", total)
		}
	})
	t.Run("hold_exports_are_Stack_methods_on_main", func(t *testing.T) {
		// Structural identifier checks, not mutable implementation-text pins.
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(root, "tests/e2e/internal/labstack/labstack.go"), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		alias := false
		ast.Inspect(file, func(n ast.Node) bool {
			if typ, ok := n.(*ast.TypeSpec); ok && typ.Name.Name == "Stack" && typ.Assign.IsValid() {
				if sel, ok := typ.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Stack" {
					if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "e2e" {
						alias = true
					}
				}
			}
			return true
		})
		if !alias {
			t.Fatal("labstack.Stack alias missing")
		}
		found := map[string]bool{}
		for _, path := range []string{"tests/e2e/stack_exports.go", "tests/e2e/generation_hold.go"} {
			file, err := parser.ParseFile(fset, filepath.Join(root, path), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
					continue
				}
				star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
				if !ok {
					continue
				}
				typ, ok := star.X.(*ast.Ident)
				if ok && typ.Name == "Stack" {
					found[fn.Name.Name] = true
				}
			}
		}
		for _, name := range []string{"ArmMatchingLeaseHold", "MatchingLeaseHold", "ReleaseMatchingLeaseHold", "GenerationHoldSnapshot", "LocalLeaseReadback"} {
			if !found[name] {
				t.Errorf("missing Stack export %s", name)
			}
		}
	})
	t.Run("prior_assertions_47_to_51_findings_and_decisions_unchanged", func(t *testing.T) {
		source := readOwnedDoc(t, root, "tests/handover/handover_test.go")
		start := strings.Index(source, "func TestHandover_Followup28Claims(")
		if start < 0 {
			t.Fatal("missing prior function")
		}
		end := strings.Index(source[start:], "// Live integrity remains separate")
		if end < 0 {
			t.Fatal("missing prior function boundary")
		}
		prior := restoreFollowup29Assertions(source[start : start+end])
		sum := sha256.Sum256([]byte(strings.TrimSpace(prior)))
		if hex.EncodeToString(sum[:]) != "29dd7c2674a18577c527ccdd25ab23d1411557d19895f679b1329c30e717ba9f" {
			t.Fatal("follow-up 28 assertions changed beyond frozen-input redirects / exact-note exclusion")
		}
		checkHash(t, "docs/handover/follow-up-28.md", "8f47177abd4cca06b38454b8c6a9fd5a8dfc791e7bafb532fc9b440365bdd2df")
		old, current := readOwnedDoc(t, root, frozen+"register28.md"), readOwnedDoc(t, root, register)
		decisions := regexp.MustCompile(`(?m)^\| D-[0-9]{3} \|.*$`)
		before, after := decisions.FindAllString(old, -1), decisions.FindAllString(current, -1)
		if len(after) != 44 || !slices.Equal(before, after) || !strings.HasPrefix(after[len(after)-1], "| D-141 |") {
			t.Fatal("decisions changed or a decision added after D-141")
		}
		oldFindings := strings.Split(strings.Split(old, "## Findings register at the follow-up 28 cutoff")[1], "\nEvery historical failure")[0]
		newFindings := strings.Split(strings.Split(current, "## Findings register at the follow-up 29 cutoff")[1], "\nEvery historical failure")[0]
		changed := map[string]bool{"Members F11-01": true, "r18 M-3": true, "Isolation F14-01": true, "Members 15 serving-node signature": true, "Netlab D-120 missing successor": true}
		count := 0
		for _, row := range strings.Split(oldFindings, "\n") {
			if !strings.HasPrefix(row, "| ") || strings.HasPrefix(row, "| Finding |") {
				continue
			}
			count++
			name := strings.TrimSpace(strings.Split(row, "|")[1])
			if !changed[name] && !strings.Contains(newFindings, row) {
				t.Errorf("unaffected finding changed: %s", row)
			}
		}
		if count != 47 {
			t.Fatalf("prior findings %d", count)
		}
		if got := len(regexp.MustCompile(`(?m)^\| [^|]+ \|`).FindAllString(newFindings, -1)) - 1; got != 51 {
			t.Fatalf("current findings %d", got)
		}
	})
	t.Run("seven_current_entrypoints_and_configuration_warning", func(t *testing.T) {
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			lines := strings.Split(readOwnedDoc(t, root, path), "\n")
			if len(lines) < 4 || !strings.Contains(lines[2], "Current cut") || !strings.Contains(lines[2], "follow-up-29.md") || !strings.Contains(lines[2], "#396") || !strings.Contains(lines[3], "follow-up-28.md") || !strings.Contains(lines[3], "#390") {
				t.Errorf("stale entrypoint: %s", path)
			}
		}
		op := readOwnedDoc(t, root, "docs/handover/operator-guide.md")
		a, b, note := strings.Index(op, "### 3.2 "), strings.Index(op, "### 3.3 "), strings.Index(op, followup29Configuration)
		if a < 0 || note <= a || b <= note || strings.Count(op, followup29Configuration) != 1 {
			t.Fatal("configuration note absent, repeated or outside node configuration")
		}
		requireHandoverClaimTokens(t, root, "docs/handover/operator-guide.md", followup29Configuration)
		for pr := 391; pr <= 396; pr++ {
			requireHandoverClaimTokens(t, root, "docs/ops/README.md", fmt.Sprintf("#%d", pr))
		}
		requireHandoverClaimTokens(t, root, "docs/ops/README.md", "Follow-up 29 acceptance snapshot", "No later lane result is included")
	})
	t.Log("follow-up 29: 22 subtests / 6 exact roadmap rows / 8 immutable snapshots / 731 archive and run file checks / 51 findings")
}

// Live integrity is separate from frozen acceptance evidence.
func TestHandover_Followup29LiveDocumentIntegrity(t *testing.T) {
	root := repoRoot(t)
	var manifest struct {
		Followup struct {
			Documents []struct{ Path, SHA256 string } `json:"reviewed_documents"`
		} `json:"handover_follow_up_29"`
	}
	if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Followup.Documents) != 23 {
		t.Fatalf("live document inventory %d", len(manifest.Followup.Documents))
	}
	for _, p := range manifest.Followup.Documents {
		sum := sha256.Sum256([]byte(readOwnedDoc(t, root, p.Path)))
		if hex.EncodeToString(sum[:]) != p.SHA256 {
			t.Errorf("live document hash drift: %s", p.Path)
		}
	}
}
