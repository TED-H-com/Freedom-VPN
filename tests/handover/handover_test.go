// Package handover_test verifies that the owned handover and operations
// Markdown documents stay synchronized with the repository layout, Make
// targets, built fvpn-* binaries, migration inventory, and PostgreSQL schema.
package handover_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// ownedDocs lists the handover, ops, and root Markdown files owned by
// w4-handover. Note: docs/handover/restore-drill.md and docs/ops/backups.md
// are owned by the backups task and excluded from edits this round.
var ownedDocs = []string{
	"README.md",
	"docs/handover/README.md",
	"docs/handover/deploy.md",
	"docs/handover/calibration.md",
	"docs/handover/operator-guide.md",
	"docs/handover/member-guide.md",
	"docs/ops/README.md",
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
	cmdBuildPathRE := regexp.MustCompile(`\./((?:apps|node|internal|packaging|tests)/[A-Za-z0-9_./-]+)`)

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
		"tests/e2e/cmd/fvpn-*",
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
		"fvpn-enrollment-v1":     "protocol-domain-separation-label",
		"fvpn-release-trust-v1":  "protocol-domain-separation-label",
		"fvpn-portal-prod":       "example-oidc-client-id",
		"fvpn-portal-test":       "example-oidc-client-id",
		"fvpn-enroll-peer":       "netbird-correlation-group-prefix (fvpn-enroll-peer-<hash>)",
		"fvpn-unpublished-nodes": "netbird-shared-staging-group-name",
		"fvpn-node":              "netbird-setup-key-prefix (fvpn-node-<node_id>-<incarnation_id>)",
		"fvpn-updater-recover":   "systemd-oneshot-unit (fvpn-updater-recover.service)",
	}

	tokenRE := regexp.MustCompile(`\b(fvpn-[a-z0-9-]+)(\.service)?\b`)
	checked := 0
	for _, doc := range ownedDocs {
		content := readOwnedDoc(t, root, doc)
		for _, m := range tokenRE.FindAllStringSubmatch(content, -1) {
			name := m[1]
			suffix := m[2]
			if suffix == ".service" {
				// Systemd service units on the node live in node/agent/systemd/
				// (central units fvpn-control.service and fvpn-worker.service are
				// inline unit files in deploy.md corresponding to built binaries).
				if _, ok := binaries[name]; ok {
					checked++
					continue
				}
				unitPath := filepath.Join(root, "node/agent/systemd", name+".service")
				if _, err := os.Stat(unitPath); err != nil {
					t.Errorf("%s: systemd unit %s.service neither matches a binary nor exists at %s", doc, name, unitPath)
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

func TestHandover_MigrationInventory(t *testing.T) {
	root := repoRoot(t)
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
	wantCount := len(wantIDs)
	wantFirst := wantIDs[0]
	wantLast := wantIDs[len(wantIDs)-1]

	// 1. Check migration count, range, and last-ID claims across owned docs.
	countRE := regexp.MustCompile("(?:`?([0-9]+)`?\\s+(?:forward\\s+)?migrations|count\\s+`?([0-9]+)`?)")
	rangeRE := regexp.MustCompile("`(00[0-9]{3})`[–-]`(00[0-9]{3})`")
	lastIDRE := regexp.MustCompile("(?:last ID\\s+`(00[0-9]{3})`|through\\s+`(00[0-9]{3})(?:_[a-z0-9_]+\\.sql)?`)")
	for _, doc := range ownedDocs {
		content := readOwnedDoc(t, root, doc)
		for _, m := range countRE.FindAllStringSubmatch(content, -1) {
			numStr := m[1]
			if numStr == "" {
				numStr = m[2]
			}
			gotCount, err := strconv.Atoi(numStr)
			if err != nil {
				t.Fatalf("%s: parse migration count %q: %v", doc, numStr, err)
			}
			if gotCount != wantCount {
				t.Errorf("%s: migration count claim is %d, want %d (from db/migrations/)", doc, gotCount, wantCount)
			}
		}
		for _, m := range rangeRE.FindAllStringSubmatch(content, -1) {
			if m[1] != wantFirst || m[2] != wantLast {
				t.Errorf("%s: migration range claim is %s..%s, want %s..%s", doc, m[1], m[2], wantFirst, wantLast)
			}
		}
		for _, m := range lastIDRE.FindAllStringSubmatch(content, -1) {
			gotLast := m[1]
			if gotLast == "" {
				gotLast = m[2]
			}
			if gotLast != wantLast {
				t.Errorf("%s: migration last-ID claim is %s, want %s (from db/migrations/)", doc, gotLast, wantLast)
			}
		}
	}

	// 2. Check the fvpn-migrate status block in docs/handover/deploy.md.
	deployMD := readOwnedDoc(t, root, "docs/handover/deploy.md")
	statusBlockRE := regexp.MustCompile("(?s)```text\\ncurrent_database=\"fvpn\"\\n(.*?)```")
	m := statusBlockRE.FindStringSubmatch(deployMD)
	if m == nil {
		t.Fatal("docs/handover/deploy.md: missing fvpn-migrate status output block")
	}
	var gotIDs []string
	for _, line := range strings.Split(strings.TrimSpace(m[1]), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != "applied" {
			t.Fatalf("docs/handover/deploy.md: unexpected status line %q", line)
		}
		gotIDs = append(gotIDs, fields[0])
	}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Errorf("docs/handover/deploy.md: fvpn-migrate status IDs = %v, want %v", gotIDs, wantIDs)
	}
}

type sqlBlock struct {
	Doc            string
	Index          int
	ClusterRoleDDL bool
	Raw            string
}

func extractSQLBlocks(t testing.TB, root string) []sqlBlock {
	t.Helper()
	sqlFenceRE := regexp.MustCompile("(?s)```sql\\n(.*?)```")
	var out []sqlBlock
	for _, doc := range ownedDocs {
		content := readOwnedDoc(t, root, doc)
		matches := sqlFenceRE.FindAllStringSubmatch(content, -1)
		for i, m := range matches {
			body := strings.TrimSpace(m[1])
			clusterDDL := strings.HasPrefix(body, "-- [mechanical-check: cluster-role-ddl]")
			out = append(out, sqlBlock{
				Doc:            doc,
				Index:          i + 1,
				ClusterRoleDDL: clusterDDL,
				Raw:            body,
			})
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
	for _, b := range blocks {
		if b.ClusterRoleDDL {
			t.Logf("[non-mechanical: cluster-role-ddl] %s SQL block #%d explicitly marked as superuser/cluster role DDL", b.Doc, b.Index)
			continue
		}
		if strings.Contains(b.Raw, "CREATE DATABASE") || strings.Contains(b.Raw, "CREATE ROLE") {
			t.Errorf("%s SQL block #%d contains cluster DDL without -- [mechanical-check: cluster-role-ddl] marker", b.Doc, b.Index)
		}
	}
}
