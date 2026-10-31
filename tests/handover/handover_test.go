// Package handover_test verifies that the owned handover and operations
// Markdown documents stay synchronized with the repository layout, Make
// targets, built fvpn-* binaries, migration inventory, and PostgreSQL schema.
package handover_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	// Revision 12 advanced the count from 47 to 49 (listAdminDevices, authorizeMemberDevice). This test still pins follow-up 7's fleet-config decisions.
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
	if opCount != 49 {
		t.Fatalf("contracts/openapi.yaml operation count = %d, want 49", opCount)
	}
	if _, ok := spec.Paths["/admin/fleet-config"]; !ok {
		t.Fatal("contracts/openapi.yaml missing /admin/fleet-config path")
	}
	fcProp := spec.Components.Schemas["FleetConfigUpdate"].Properties["max_offline_lease_seconds"]
	if fcProp.Minimum != 60 || fcProp.Maximum != 86400 {
		t.Fatalf("FleetConfigUpdate.max_offline_lease_seconds bounds = %d..%d, want 60..86400", fcProp.Minimum, fcProp.Maximum)
	}

	// 2. Verify operator-guide.md fleet-config section, the wired-operation claim, and PR #173..#182 catch-up claims.
	operatorGuide := readOwnedDoc(t, root, "docs/handover/operator-guide.md")
	for _, token := range []string{
		"All 49 are wired in `apps/control` on `main`",
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
		"All 49 are wired in `apps/control` on `main`",
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

	// 1. Verify revision 11 schema elements still present after revision 13 advanced info.version to 1.9.0:
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
	if spec.Info.Version != "1.9.0" {
		t.Fatalf("contracts/openapi.yaml info.version = %q, want 1.9.0", spec.Info.Version)
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
		"OpenAPI `1.9.0`, revision 13",
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
		"When a `node.publish` operation stays `running` with `agent_policy` or `data_path_probe` in `status: \"pending\"`, `error_code: \"NODE_NOT_SERVICEABLE\"`, and `message: \"member_forwarding_unconfirmed\"`, central is waiting for the node to apply the latest compiled configuration generation and report passing serving checks (`exits.publication_observed` remains unchanged, or `\"withdrawn\"` if an empty-exception route was withdrawn first); the step clears automatically on the next qualifying observation and worker pass, and transitions to `status: \"failed\"` (`publication_observed: \"error\"`) only if the worker retry budget (`worker_attempts` = 10 attempts, ~4.3–8.5 minutes of jittered exponential backoff from 1s to 300s) is exhausted, at which point you should verify node health and re-issue `PUT /api/v1/admin/nodes/{node_id}/publication`.",
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
		content := readOwnedDoc(t, root, path)
		for _, token := range tokens {
			if !strings.Contains(content, token) {
				t.Errorf("%s missing follow-up 13 claim %q", path, token)
			}
		}
	}
	t.Run("restore_authorization_contract_and_implementation", func(t *testing.T) {
		for _, path := range []string{"docs/handover/operator-guide.md", "docs/handover/deploy.md", "docs/handover/restore-drill.md"} {
			require(path, "GET /api/v1/admin/devices?authorization_unconfirmed=true", "POST /api/v1/admin/members/{member_id}/devices/{device_id}/authorize", "authorization_unconfirmed_since", "Idempotency-Key", "CSRF", "member.device_authorize", "409 STATE_CONFLICT", "409 DEVICE_LIMIT_REACHED", "authorization_unconfirmed: false")
		}
		require("docs/handover/operator-guide.md", "existing `member.device_authorize` operation", "after the worker sync", "device.authorization.external_withdrawal", "device.authorize", "only path that re-grants", "always", "optional in the contract", "no request body", "#247")
		require("docs/handover/member-guide.md", "需營運人員重新授權", "needs operator re-authorization", "authorization_unconfirmed: true", "always emits", "absent means the server does not project it yet")
		require("internal/members/httpadapter/reads.go", "func (h Handlers) ListAdminDevices")
		require("internal/members/httpadapter/commands.go", "func (h Handlers) AuthorizeMemberDevice")
		require("internal/members/httpadapter/admin_devices_integration_test.go", "func TestD077_AdminDeviceListAndReauthorize", "member device omitted authorization_unconfirmed", "re-authorization did not restore the paid group")
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
		require("db/migrations/"+migration, "ADD COLUMN usage_outbox JSONB", "ADD COLUMN usage_outbox_observed_at TIMESTAMPTZ")
		for _, path := range []string{"docs/handover/00-index.md", "docs/handover/11-database-and-retention.md", "docs/handover/operator-guide.md"} {
			require(path, migration)
		}
		require("docs/handover/operator-guide.md", "usage_outbox: null", "never absent", "ETag", "informational", "meter.Health", "EvaluateGate", "pure reads", "until a verified recovery checkpoint", "ApplyCheckpoint", "central_acked_through")
		require("internal/nodes/httpwire/handlers_integration_test.go", "func TestSECOPS19_D077_HTTPNodeDetailUsageOutboxETagAndDomainColumn")
		require("internal/nodes/observation_readiness_integration_test.go", "func TestSECOPS19_D077_NodeDetailUsageOutboxProjectionAndMigration")
		require("node/internal/usagestore/status_test.go", "func TestOutboxStatusIsPure")
		require("docs/node/usage-upload.md", "A batch response never changes the ledger epoch", "meter.Health")
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
		require("docs/node/install.md", "update-staging/pending-release.json", "already_current", "## Exit codes")
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
		require("docs/e2e/runs/2026-10-31/followup14/run.md", "pass=29 blocked=0 fail=0", "pass=16 blocked=0 fail=0")
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
}

// TestHandover_Followup14Claims checks the dated main boundary, implemented
// contracts and the owner measurements. Numeric evidence is always a history
// snapshot or run-ID-named fixture (D-098), never a lane's current record.
func TestHandover_Followup14Claims(t *testing.T) {
	root := repoRoot(t)
	require := func(path string, tokens ...string) {
		t.Helper()
		content := readOwnedDoc(t, root, path)
		for _, token := range tokens {
			if !strings.Contains(content, token) {
				t.Errorf("%s missing follow-up 14 claim %q", path, token)
			}
		}
	}
	const op = "docs/handover/operator-guide.md"
	const ops = "docs/ops/README.md"
	const fixtures = "tests/handover/testdata/"
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
		require("db/migrations/00190_serving_leases_clamp.sql", "NEW.expires_at < OLD.expires_at", "to_jsonb(NEW)-'expires_at'")
		require("internal/nodes/httpwire/review_r15_clamp_integration_test.go", "TestReviewR15_StoredLeaseClampGuardProof")
	})
	t.Run("D085_live_gate_D090_agent_containment_D087_schedule", func(t *testing.T) {
		require(op, "before reconciliation, after Apply before the ACK, and before the heartbeat", "min(1 s, min(heartbeat, poll)/2)", "52.6 ms per serving step", "never acknowledges the offered generation", "Member links are administratively down until a clean verified Apply", "re-containment on the next step is proven, uninterrupted containment is not", "D-100", "quarantine_<interface>", "never deletes foreign tc objects", "D-096")
		require("node/agent/serving.go", "return min(time.Second, e.stepInterval()/2)", "e.verifyCurrentServing(ctx)", "e.beat()")
		require("node/agent/followup21_test.go", "TestD085_ISO01_BetweenApplyInjection", "TestD085_InjectionAfterApplyBeforeAck", "TestD085_ReadTimeoutAndCanceledStep", "TC_FOREIGN_FILTER")
		require(fixtures+"d085-2df26aca-measurement.txt", "52.6 ms added per normal serving step", "3 reads, 30 steps")
		require("node/enforce/quarantine.go", "func Quarantine(", "quarantine_")
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
		for _, path := range []string{"deploy/node/systemd/fvpn-updater.service", "deploy/node/systemd/fvpn-updater-recover.service"} {
			unit := readOwnedDoc(t, root, path)
			if strings.Contains(unit, "refresh-boot-id") {
				t.Errorf("%s still refreshes boot id", path)
			}
			for _, line := range strings.Split(unit, "\n") {
				if strings.HasPrefix(line, "ReadWritePaths=") && strings.Contains(line, "/etc/vpn-node") {
					t.Errorf("%s allows config writes", path)
				}
			}
		}
		require("docs/handover/calibration.md", "fs.inotify.max_user_instances", "`128` was exhausted")
		require("docs/handover/deploy.md", "ENR-06", "/var/run/netbird.sock", "installer follow-up 3", "21 pass / 1 fail")
	})
	t.Run("portal16_and_D091_restore_review", func(t *testing.T) {
		for _, path := range []string{op, "docs/handover/deploy.md", "docs/handover/restore-drill.md"} {
			require(path, "Portal follow-up 16 (#255)", "/admin/devices", "neutral receipt", "paginated read-back", "usage-outbox")
		}
		require(op, "D-091 (#263)", "no published exit carries this member", "node_identity_endpoints.mapping_frozen", "registered-node fallback reading `checks.applied_policy_refs` is deleted", "owner's Step 6")
		require("apps/worker/review_r15_q9_fup_integration_test.go", "TestReviewR15_UnpublishDropsTheCarryingExitAndApplyingSticks", "TestReviewR15_FrozenCarryingExitKeepsFairUseApplying")
		require("docs/usage/restore-runbook.md", "### Step 6:", "409 STATE_CONFLICT", "409 DEVICE_LIMIT_REACHED")
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
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fmt.Sprintf(fixtures+run+"-AT%02d.json", id))), &r); err != nil {
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
		require(fixtures+run+"-summary.md", "471 pass / 0 fail", "36.997 s", "41.729 s")
		require(fixtures+run+"-D077.md", "STATE_CONFLICT", "DEVICE_LIMIT_REACHED", "fence_after=0", "fence_count=1")
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
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fixtures+run+"-restore-reconcile.json")), &e); err != nil {
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
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fixtures+run+"-"+id+".json")), &result); err != nil {
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
		require(fixtures+run+"-AT30.md", "95.507504", "93.532554563", "106.868537734")

		for _, path := range []string{op, "docs/handover/deploy.md", "docs/handover/restore-drill.md"} {
			require(path, run, "shared development host", "95.507504 s", "93.532554563 s", "106.868537734 s", "#242", "#221", "SECOPS-21 remains open", "pending fresh observation after restore", "before starting either service", "D-099", "first new offer `15`", "applied ACK")
		}
		require("docs/policies/2026-10-31-secops-21-handover.md", "cachedStateSigner.Sign", "Do not close SECOPS-21 from the compiler tests alone")
		require("internal/desiredstate/restore_integration_test.go", "TestAT30_D095_FirstOfferWaitsForFreshCurrentIncarnation", "TestAT30_D095_CompleteRecoveryDoesNotReleaseUnreportedNode")
	})
	t.Run("r15_adopted_proofs_and_coming_decisions", func(t *testing.T) {
		review := "docs/reviews/r15-d072-d075-review.md"
		require(review, "Owner fixes adopted — 2026-10-31 EDT", "#254", "#259", "#263", "#256", "#262", "#252", "#233", "r16 has not run")
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
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/isolation/history/follow-up-6-bootstrap-only/provisional-recovery.raw.json")), &window); err != nil {
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
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, fmt.Sprintf(fixtures+run+"-AT%d.json", id))), &r); err != nil {
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
		require(fixtures+run+"-recovery-action.json", "latched TC_FOREIGN_FILTER", "docker restart")
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
			if !strings.HasPrefix(pin.Path, fixtures) || !runName.MatchString(filepath.Base(pin.Path)) || len(pin.Commit) != 40 {
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
