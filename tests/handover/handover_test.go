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
		"one `/48` may hold every slot except a reserve of one",
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
		"one `/48` may hold every slot except a reserve of one",
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
		"Two clones behind the **same** IPv4 `/24` or IPv6 `/48` compatibility domain are not detected by this check.",
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
		"In each `anonymous` table (`ingress[anonymous]` and `anonClass`), one `/48` may hold every slot except a reserve of one (`max_rate_principals` defaults to `10000`, so `9999` `/64`s; a cap of one still allows that slot). A new `/64` in a `/48` that already has keys returns `429 RATE_LIMITED` with `Retry-After: 1` once only that reserve remains, without inserting or evicting, and another `/48` or an IPv4 address can take the reserve. On the agent mTLS listener, no verified chain is `401 UNAUTHENTICATED` before the limiter; a verified leaf keys ingress by its `urn:fvpn:node:<uuid>` URI SAN (`node:<uuid>`), so every still-usable serial of a node shares one burst during renewal overlap, and `sha256(leaf.Raw)` is the key only for a verified leaf without that SAN (`gates[class]` is taken before `AuthenticateAgent`, and a failed authentication has spent the token). When the table is full, eviction walks from the tail to the first bucket idle at full tokens or idle for more than an hour, bounded by `max_rate_principals`; a table with no eligible bucket returns `429`.",
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
		"one `/48` may hold every slot except a reserve of one",
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
		if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/restore-reconcile.json")), &evidence); err != nil {
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
		require("docs/handover/operator-guide.md", "docs/node/install.md", "already_current", "check --json", "update-staging/pending-release.json", "does not move `current`", "reenroll --token-file", "identity.retired/<utc>/", "keeps `usage.db`", "image-prepare", "firstboot", "`--token` on argv", "exit `64`", "`latest` is refused", "journal-first", "`0` success, `1` failed, `2` drifted, `3` busy, `4` unsupported platform, `5` dependency, `64` usage", "hardware clone boot", "ENR-02", "D-080", "no merged roadmap row")
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
			if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/isolation/"+id+".json")), &record); err != nil {
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
		require("docs/ops/README.md", "#237", "#246", "**subset passed, case blocked**", "Usage follow-up 3 re-run in progress as of 2026-10-30 EDT", "29 pass / 0 blocked / 0 fail", "16 pass / 0 blocked / 0 fail", "121.88 s")
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
		require("docs/reviews/r15-d072-d075-review.md", "dfafae0f985a5209ccf380afb9c615987f2bd775", "3929d4ca5ec00703e1d2dd9ae6306f9acf5aa3b445f288be31d4ee50acffbbbb", "grok/grok-4.7", "Routing in progress — 2026-10-30 EDT", "**H-1**", "**M-6**", "**L-1**", "not yet adopted on main")
		require("docs/reviews/README.md", "r14-d067-d070-review.md", "r15-d072-d075-review.md", "8 (1 High, 6 Medium, 1 Low)")
	})
	t.Run("reviewed_document_hashes", func(t *testing.T) {
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
