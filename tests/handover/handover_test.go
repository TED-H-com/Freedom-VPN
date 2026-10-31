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
		if name == "r16-rev13-d089-d099-review.md" || name == "r17-rev14-d104-d117-review.md" {
			// D-098 forbids reformatting the raw report to our older table schema.
			// Followup15Claims / Followup18Claims verify SHA-256 and review metadata.
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
	const register = "docs/handover/README.md"
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
			{"docs/handover/operator-guide.md", []string{"--control-url", "--server-ca", "exactly the control origin", "install --allow-loopback", "TEST ONLY", "result_url_missing", "VALIDATION_FAILED", "identical pending receipt once", "same absolute deadline", "open", "OS reboot or a lost anchor cannot adopt", "failed verifier report before repairing"}},
			{"docs/handover/deploy.md", []string{"result_url_missing", "identical receipt", "TEST ONLY", "same-release rerun", "original valid lease", "owner\ndecision pending updates 11"}},
			{"docs/handover/member-guide.md", []string{"同一次作業系統開機", "不會延長原到期時間", "原有收據", "TEST ONLY", "same OS boot", "deadline is unchanged", "original receipt", "ISO-F6-03 remains", "full AT09"}},
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
			requireHandoverClaimTokens(t, root, "docs/ops/README.md", fmt.Sprintf("#%d", pr))
		}
		requireHandoverClaimTokens(t, root, doc, cut, "Handover ran no new live acceptance", "D-098 / D-109 / D-132")
		for _, path := range []string{"README.md", "docs/handover/00-index.md", "docs/handover/deploy.md", "docs/handover/operator-guide.md", "docs/handover/member-guide.md", "docs/handover/restore-drill.md", "docs/ops/README.md"} {
			lines := strings.Split(readOwnedDoc(t, root, path), "\n")
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
			{"docs/e2e/followup19-archive.sha256", "", 168},
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
		if len(pins) != 37 {
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
