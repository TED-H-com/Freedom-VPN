package ci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const modulePath = "github.com/TED-H-com/Freedom-VPN/"

type workflow struct {
	On   map[string]yaml.Node `yaml:"on"`
	Jobs map[string]struct {
		Steps []struct {
			Run string `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

// These are the inputs hidden behind scripts or test processes. Packages in
// make recipes are deliberately NOT enumerated here: they are read from make.
// Source roots are scanned for new literal commands and repository paths.
type indirect struct {
	packages []string
	files    []string // exact files or directories ending in /**
	sources  []string // files or trees containing the indirect references
	fixture  string   // source copied into a temporary module
}

var stackPackages = []string{
	"./tests/e2e/cmd/fvpn-e2e-prepare", "./apps/control/cmd/fvpn-control",
	"./apps/worker/cmd/fvpn-worker", "./node/agent/cmd/fvpn-agent",
	"./node/watchdog/cmd/fvpn-watchdog", "./node/updater/cmd/fvpn-updater",
	"./tests/e2e/cmd/fvpn-e2e-idp", "./tests/e2e/cmd/fvpn-e2e-payment",
	"./tests/e2e/cmd/fvpn-e2e-nodeconfig", "./tests/e2e/cmd/fvpn-e2e-origin",
	"./tests/e2e/cmd/fvpn-e2e-nbbootstrap", "./tests/e2e/cmd/fvpn-e2e-control",
}

var usagePackages = []string{
	"./internal/usage/cmd/fvpn-usage-admin", "./node/internal/meter",
	"./node/internal/usagestore", "./node/internal/usageupload",
}

func together(lists ...[]string) []string {
	var out []string
	for _, list := range lists {
		out = append(out, list...)
	}
	return out
}

var indirectInputs = map[string]map[string]indirect{
	"at-members": {
		"verify":         {packages: []string{"./tests/e2e/at/members/cmd/probe"}, files: []string{"tests/e2e/at/members/**", "docs/e2e/at/members/**", "deploy/devstack/probe-portal.mjs", "docs/dns/BACKEND.md", "docs/poc/G2-f802-mtu1280-2026-10-31.md", "contracts/examples/valid/ObservationReport.json", "contracts/examples/policies/fair-use.json"}, sources: []string{"tests/e2e/at/members"}},
		"assigned-stack": {packages: together(stackPackages, []string{"./tests/e2e/at/members/cmd/probe"}), files: []string{"mk/e2e.mk", "deploy/devstack/**", "deploy/backup/postgres/**", "apps/portal/**", "tests/e2e/at/members/**", "docs/e2e/at/members/**", "docs/dns/BACKEND.md", "docs/poc/G2-f802-mtu1280-2026-10-31.md", "contracts/examples/valid/ObservationReport.json", "contracts/examples/policies/fair-use.json"}, sources: []string{"deploy/devstack", "tests/e2e/at/members"}},
	},
	"at-usage": {
		"verify":         {packages: usagePackages, files: []string{"tests/e2e/at/usage/**", "docs/e2e/at/usage/**"}, sources: []string{"tests/e2e/at/usage"}},
		"assigned-stack": {packages: together(stackPackages, usagePackages), files: []string{"mk/e2e.mk", "deploy/devstack/**", "deploy/backup/postgres/**", "apps/portal/**", "tests/e2e/at/usage/**", "docs/e2e/at/usage/**"}, sources: []string{"deploy/devstack", "tests/e2e/at/usage"}},
	},
	"at-secops": {
		"component-checks": {packages: []string{"./tests/e2e/at/secops/leaseprobe", "./tests/e2e/at/secops/proxyprobe", "./internal/usage/cmd/fvpn-usage-admin"}, files: []string{"tests/e2e/at/secops/**", "docs/e2e/at/secops/**", "node/agent/observe.go", "node/agent/sync.go", "node/agent/serving.go", "deploy/devstack/nginx.conf", "deploy/backup/restore-pitr.sh", "deploy/backup/post-restore-checklist.sql", "deploy/backup/backup.sh", "contracts/examples/policies/fair-use.json"}, sources: []string{"tests/e2e/at/secops"}},
	},
	"backup": {
		"lint-and-unit": {packages: []string{"./internal/db/cmd/fvpn-migrate"}, files: []string{"deploy/backup/**", "tests/backup/**", "docs/ops/backups.md", "docs/handover/restore-drill.md"}, sources: []string{"tests/backup", "deploy/backup"}},
		"pitr-drill":    {packages: []string{"./internal/db/cmd/fvpn-migrate"}, files: []string{"deploy/backup/**", "tests/backup/**", "docs/ops/backups.md", "docs/handover/restore-drill.md"}, sources: []string{"tests/backup", "deploy/backup"}},
	},
	"contracts": {
		"contracts": {files: []string{"contracts/**", "docs/contracts/**", "internal/policies/schemas/**"}, sources: []string{"contracts"}},
	},
	"dns": {
		"unit":        {packages: []string{"./internal/dnspolicy/cmd/bootstrap", "./tests/netlab/dns/cmd/cache-authority"}, files: []string{"deploy/dev/adguard/**", "tests/netlab/dns/**", "docs/dns/BACKEND.md", "docs/control/config.example.json", "contracts/examples/valid/ResolverDocument.json", "contracts/examples/valid/ResolverDocumentRef.json", "contracts/examples/valid/DnsPolicyEvidence.json"}, sources: []string{"tests/netlab/dns"}},
		"integration": {packages: []string{"./internal/dnspolicy/cmd/bootstrap"}, files: []string{"deploy/dev/adguard/**"}, sources: []string{"deploy/dev/adguard"}},
		"publish":     {files: []string{"contracts/examples/valid/ObservationReport.json", "contracts/examples/valid/ResolverDocument.json", "contracts/examples/valid/ResolverDocumentRef.json", "contracts/examples/valid/DnsPolicyEvidence.json", "contracts/examples/valid/ThreatFeedPayload.default-null-attribution.json", "contracts/examples/valid/PolicyRevisionCreate.feed-default-null-attribution.json"}},
		"lab":         {packages: []string{"./internal/dnspolicy/cmd/bootstrap", "./tests/netlab/dns/cmd/cache-authority"}, files: []string{"deploy/dev/adguard/**", "tests/netlab/dns/**", "docs/dns/BACKEND.md"}, sources: []string{"tests/netlab/dns"}},
	},
	"fleetupdate": {"verify": {files: []string{"Makefile", "mk/**", "contracts/examples/valid/ObservationReport.json", "contracts/examples/valid/UpdateResult.installer.json", "contracts/vectors/release-manifest-signing.json"}}},
	"installer": {
		"verify": {files: []string{"tests/installer/**", "deploy/node/**"}, sources: []string{"tests/installer/control-build.py"}, fixture: "tests/installer/cmd/fvpn-w5-control"},
	},
	"portal": {
		"portal": {packages: []string{"./apps/control/cmd/fvpn-control", "./apps/worker/cmd/fvpn-worker"}, files: []string{"apps/portal/**", "contracts/vectors/release-manifest-signing.json", "internal/fleetupdate/testdata/at26-direct-rollback.json", "internal/netbird/testdata/**"}, sources: []string{"apps/portal/e2e/dev-stack.mjs", "apps/portal/playwright.config.ts"}},
	},
}

var (
	makeLine   = regexp.MustCompile(`\bmake\s+((?:-f\s+[^\s]+\s+)?[a-zA-Z0-9_.-]+(?:\s+[a-zA-Z0-9_.-]+)*)`)
	goCommand  = regexp.MustCompile(`\bgo\s+(test|build|vet|run|install)\b([^;\n]*)`)
	goTags     = regexp.MustCompile(`(?:^|\s)-tags(?:=|\s+)([a-zA-Z0-9_,]+)`)
	localPkg   = regexp.MustCompile(`(?:^|[^a-zA-Z0-9_])((?:\./)[a-zA-Z0-9_./-]*)`)
	forLoop    = regexp.MustCompile(`\bfor\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+in\s+([^;]+);`)
	quotedPkg  = regexp.MustCompile(`["'` + "`" + `](\./[a-zA-Z0-9_./-]+)["'` + "`" + `]`)
	quotedPath = regexp.MustCompile(`["'` + "`" + `]((?:\.\./|(?:contracts|internal|pkg|node|apps|deploy|tests|docs|mk)/)[a-zA-Z0-9_./*+-]+)`)
	quotedGo   = regexp.MustCompile(`["']go["']\s*,\s*\[?\s*["'](?:build|run|install|test)["']`)
	scriptPath = regexp.MustCompile(`\b((?:tests|deploy|contracts|apps)/[a-zA-Z0-9_./-]+\.(?:sh|py|mjs|ts))\b`)
)

func addRequest(dst map[string]map[string]bool, tags string, pkgs ...string) {
	if dst[tags] == nil {
		dst[tags] = make(map[string]bool)
	}
	for _, p := range pkgs {
		if p != "" {
			dst[tags][p] = true
		}
	}
}

// Commands are taken from make -n, including literal loop lists used by build
// and coverage recipes. A new, unrecognized variable package fails closed.
func parseGoCommands(text string, dst, tests map[string]map[string]bool) error {
	text = strings.ReplaceAll(text, "\\\n", " ")
	loops := map[string][]string{}
	for _, loop := range forLoop.FindAllStringSubmatch(text, -1) {
		for _, p := range localPkg.FindAllStringSubmatch(loop[2], -1) {
			loops[loop[1]] = append(loops[loop[1]], p[1])
		}
	}
	for _, command := range goCommand.FindAllStringSubmatch(text, -1) {
		args := command[2]
		tags := ""
		if m := goTags.FindStringSubmatch(args); m != nil {
			tags = m[1]
		}
		var pkgs []string
		skipValue := false
		for _, field := range strings.Fields(args) {
			field = strings.Trim(field, "\"'()\\")
			if skipValue {
				skipValue = false
				continue
			}
			switch field {
			case "-o", "-coverprofile", "-cpuprofile", "-memprofile", "-trace", "-outputdir", "-overlay", "-modfile":
				skipValue = true
				continue
			}
			if strings.HasPrefix(field, "./") {
				pkgs = append(pkgs, field)
			}
		}
		for variable, values := range loops {
			if regexp.MustCompile(`\$\{?` + regexp.QuoteMeta(variable) + `\}?\b`).MatchString(args) {
				pkgs = append(pkgs, values...)
			}
		}
		if len(pkgs) == 0 && command[1] != "run" {
			return fmt.Errorf("cannot derive package arguments from go %s%s", command[1], args)
		}
		addRequest(dst, tags, pkgs...)
		if command[1] == "test" {
			addRequest(tests, tags, pkgs...)
		}
	}
	return nil
}

// The selected targets have no recursive-make or '+' recipes. Recheck this
// before dry-running: make executes those recipes even with -n. The only
// make-level shell functions currently in these files are read-only id calls.
func safeMake(root, file string, targets []string) error {
	files := []string{file}
	if file == "Makefile" {
		more, err := filepath.Glob(filepath.Join(root, "mk", "*.mk"))
		if err != nil {
			return err
		}
		for _, path := range more {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}
	}
	for _, name := range files {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		s := strings.ReplaceAll(strings.ReplaceAll(string(b), "$(shell id -u)", ""), "$(shell id -g)", "")
		if strings.Contains(s, "$(shell") || strings.Contains(s, "${shell") {
			return fmt.Errorf("%s contains an unaudited make shell function", name)
		}
		if strings.Contains(s, "$(file ") || strings.Contains(s, "${file ") || strings.Contains(s, "$(eval ") || regexp.MustCompile(`(?m)^[^#\n]*\s!=`).MatchString(s) {
			return fmt.Errorf("%s contains unaudited make expansion", name)
		}
		for _, line := range strings.Split(s, "\n") {
			if regexp.MustCompile(`^\s*(-?include|sinclude)\s+`).MatchString(line) && line != "include $(wildcard mk/*.mk)" {
				return fmt.Errorf("%s contains an unaudited make include", name)
			}
		}
		if regexp.MustCompile(`(?m)^\t[@-]*\+`).MatchString(s) {
			return fmt.Errorf("%s contains an always-executed recipe", name)
		}
	}
	// Traverse literal prerequisites too: recursive recipes in prerequisites
	// also execute under -n. Unknown variable prerequisites require review.
	type rule struct {
		deps      []string
		recursive bool
	}
	rules := map[string]rule{}
	ruleLine := regexp.MustCompile(`^([a-zA-Z0-9_.-]+):\s*([^#]*)`)
	for _, name := range files {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		lines := strings.Split(string(b), "\n")
		active := ""
		for _, line := range lines {
			if match := ruleLine.FindStringSubmatch(line); match != nil {
				active = match[1]
				r := rules[active]
				r.deps = append(r.deps, strings.Fields(match[2])...)
				rules[active] = r
			} else if len(line) > 0 && line[0] != '\t' && !strings.HasPrefix(line, "#") {
				active = ""
			}
			if active != "" && strings.HasPrefix(line, "\t") && (strings.Contains(line, "$(MAKE)") || strings.Contains(line, "${MAKE}")) {
				r := rules[active]
				r.recursive = true
				rules[active] = r
			}
		}
	}
	seen := map[string]bool{}
	var visit func(string) error
	visit = func(target string) error {
		if seen[target] {
			return nil
		}
		seen[target] = true
		r, ok := rules[target]
		if !ok {
			return nil
		}
		if r.recursive {
			return fmt.Errorf("recursive make in selected target %s", target)
		}
		for _, dep := range r.deps {
			if strings.ContainsAny(dep, "$%") {
				return fmt.Errorf("unaudited prerequisite %q in %s", dep, target)
			}
			if dep != "|" {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, target := range targets {
		if err := visit(target); err != nil {
			return err
		}
	}
	return nil
}

func recordScripts(text string, scripts map[string]bool) {
	for _, match := range scriptPath.FindAllStringSubmatch(text, -1) {
		scripts[match[1]] = true
	}
}

func commandsFromStep(root, run string, requests, tests map[string]map[string]bool, files, scripts map[string]bool) error {
	recordScripts(run, scripts)
	if err := parseGoCommands(run, requests, tests); err != nil {
		return err
	}
	for _, match := range makeLine.FindAllStringSubmatch(run, -1) {
		fields := strings.Fields(match[1])
		makefile := "Makefile"
		if len(fields) >= 2 && fields[0] == "-f" {
			makefile, fields = fields[1], fields[2:]
		}
		if len(fields) == 0 {
			return fmt.Errorf("no make target in %q", match[0])
		}
		if err := safeMake(root, makefile, fields); err != nil {
			return err
		}
		args := append([]string{"-n", "-f", makefile}, fields...)
		cmd := exec.Command("make", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("make dry run %q: %w: %s", match[0], err, out)
		}
		files[makefile] = true
		recordScripts(string(out), scripts)
		if err := parseGoCommands(string(out), requests, tests); err != nil {
			return fmt.Errorf("%s: %w", match[0], err)
		}
	}
	return nil
}

type listedPackage struct {
	Dir             string
	ImportPath      string
	EmbedFiles      []string
	TestEmbedFiles  []string
	XTestEmbedFiles []string
	TestGoFiles     []string
	XTestGoFiles    []string
}

type inventory struct {
	dirs  map[string]bool
	files map[string]bool
}

func listPackages(root, tags, arch, proxy string, pkgs []string) (inventory, error) {
	result := inventory{dirs: map[string]bool{}, files: map[string]bool{}}
	if len(pkgs) == 0 {
		return result, nil
	}
	args := []string{"list", "-deps", "-test", "-json"}
	if tags != "" {
		args = append(args, "-tags="+tags)
	}
	args = append(args, pkgs...)
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "GOPROXY="+proxy, "GOSUMDB=off")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return result, fmt.Errorf("go %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var p listedPackage
		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			return result, err
		}
		rel, err := filepath.Rel(root, p.Dir)
		if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			continue
		}
		rel = filepath.ToSlash(rel)
		result.dirs[rel] = true
		for _, f := range together(p.EmbedFiles, p.TestEmbedFiles, p.XTestEmbedFiles) {
			result.files[rel+"/"+filepath.ToSlash(f)] = true
		}
	}
	return result, nil
}

// Only the explicitly tested packages execute their tests. Dependency tests
// in go list -deps -test are compiled for type analysis, not executed.
func testSources(root, tags, proxy string, pkgs []string) ([]string, error) {
	if len(pkgs) == 0 {
		return nil, nil
	}
	args := []string{"list", "-json"}
	if tags != "" {
		args = append(args, "-tags="+tags)
	}
	args = append(args, pkgs...)
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "GOPROXY="+proxy, "GOSUMDB=off")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var files []string
	for {
		var p listedPackage
		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		for _, f := range together(p.TestGoFiles, p.XTestGoFiles) {
			file, err := filepath.Rel(root, filepath.Join(p.Dir, f))
			if err == nil && !strings.HasPrefix(file, "..") {
				files = append(files, file)
			}
		}
	}
	return files, nil
}

func fixtureImports(root, dir string) ([]string, error) {
	entries, err := filepath.Glob(filepath.Join(root, dir, "*.go"))
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("fixture %s has no Go source", dir)
	}
	var imports []string
	for _, file := range entries {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		for _, spec := range parsed.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, err
			}
			if strings.HasPrefix(path, modulePath) {
				imports = append(imports, "./"+strings.TrimPrefix(path, modulePath))
			}
		}
	}
	return imports, nil
}

func sourceFiles(root string, sources []string) ([]string, error) {
	files := map[string]bool{}
	for _, source := range sources {
		p := filepath.Join(root, source)
		err := filepath.WalkDir(p, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" || entry.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			switch filepath.Ext(path) {
			case ".go", ".sh", ".mjs":
				files[path] = true
			case ".ts":
				if !strings.HasSuffix(path, ".d.ts") {
					files[path] = true
				}
			case ".py":
				if path == p || strings.HasPrefix(entry.Name(), "test_") {
					files[path] = true
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return sortedKeys(files), nil
}

// Literal scanning is intentionally conservative; computed paths and shell
// indirection still need the explicit inventory above and human review.
func scanSources(root string, in indirect, knownDirs, knownFiles map[string]bool) ([]string, error) {
	files, err := sourceFiles(root, in.sources)
	if err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, p := range in.packages {
		known[p] = true
	}
	var problems []string
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
				continue
			}
			if goCommand.MatchString(line) || quotedGo.MatchString(line) || strings.Contains(line, "build_one ./") {
				end := min(i+5, len(lines))
				window := strings.Join(lines[i:end], "\n")
				pkgs := quotedPkg.FindAllStringSubmatch(window, -1)
				for _, p := range localPkg.FindAllStringSubmatch(line, -1) {
					pkgs = append(pkgs, []string{"", p[1]})
				}
				for _, p := range pkgs {
					candidate := strings.TrimSuffix(p[1], "/")
					if !known[candidate] && !knownDirs[strings.TrimPrefix(candidate, "./")] {
						if info, err := os.Stat(filepath.Join(root, candidate)); err == nil && info.IsDir() {
							problems = append(problems, fmt.Sprintf("%s:%d: runtime package %s is missing from indirect table", filepath.ToSlash(rel), i+1, candidate))
						}
					}
				}
			}
			for _, match := range quotedPath.FindAllStringSubmatch(line, -1) {
				candidate := match[1]
				if strings.Contains(candidate, "*") {
					continue
				}
				path := filepath.Join(root, candidate)
				if strings.HasPrefix(candidate, "../") {
					path = filepath.Join(filepath.Dir(file), candidate)
				}
				info, err := os.Stat(path)
				if err != nil || info.IsDir() {
					continue
				}
				path, err = filepath.Rel(root, path)
				if err != nil || strings.HasPrefix(path, "..") {
					continue
				}
				path = filepath.ToSlash(path)
				// This literal is a process-name comparison, not a file read.
				if filepath.ToSlash(rel) == "tests/e2e/at/members/admission.py" && path == "mk/at-isolation.mk" && strings.Contains(line, "term in matches[p]") {
					continue
				}
				ok, err := workflowMatch(in.files, path)
				if err != nil {
					return nil, err
				}
				if !ok && !knownFiles[path] && !knownDirs[filepath.ToSlash(filepath.Dir(path))] {
					problems = append(problems, fmt.Sprintf("%s:%d: repository path %s is missing from indirect table", filepath.ToSlash(rel), i+1, path))
				}
			}
		}
	}
	return problems, nil
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestWorkflowPaths(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	// A file-only module proxy lets go list use the local download cache while
	// guaranteeing that this test never contacts a proxy or checksum server.
	cacheCmd := exec.Command("go", "env", "GOMODCACHE")
	cacheCmd.Dir = root
	cachePath, err := cacheCmd.Output()
	if err != nil {
		t.Fatalf("go env GOMODCACHE: %v", err)
	}
	proxy := (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Join(strings.TrimSpace(string(cachePath)), "cache", "download"))}).String()
	cache := map[string]inventory{}
	names := make([]string, 0, len(indirectInputs))
	for name := range indirectInputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := ".github/workflows/" + name + ".yml"
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		var wf workflow
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if _, ok := wf.On["pull_request"]; !ok {
			t.Errorf("%s: missing pull_request", name)
			continue
		}
		for job, def := range wf.Jobs {
			in, ok := indirectInputs[name][job]
			if !ok {
				t.Errorf("%s job %s: missing indirect input inventory", name, job)
				continue
			}
			requests := map[string]map[string]bool{}
			tests := map[string]map[string]bool{}
			scripts := map[string]bool{}
			files := map[string]bool{path: true}
			for _, step := range def.Steps {
				if err := commandsFromStep(root, step.Run, requests, tests, files, scripts); err != nil {
					t.Errorf("%s job %s: %v", name, job, err)
				}
			}
			addRequest(requests, "", in.packages...)
			if in.fixture != "" {
				imports, err := fixtureImports(root, in.fixture)
				if err != nil {
					t.Errorf("%s job %s: %v", name, job, err)
				} else {
					addRequest(requests, "installerfixture", imports...)
				}
				files[in.fixture+"/__ci_filter_probe__.go"] = true
			}
			for _, item := range in.files {
				if strings.HasSuffix(item, "/**") {
					files[strings.TrimSuffix(item, "**")+"__ci_filter_probe__"] = true
					base := strings.TrimSuffix(item, "/**")
					err := filepath.WalkDir(filepath.Join(root, base), func(path string, entry os.DirEntry, err error) error {
						if err != nil {
							return err
						}
						if entry.IsDir() && (entry.Name() == "node_modules" || entry.Name() == ".git" || entry.Name() == "dist") {
							return filepath.SkipDir
						}
						rel, err := filepath.Rel(root, path)
						if err != nil {
							return err
						}
						rel = filepath.ToSlash(rel)
						if entry.IsDir() {
							files[rel+"/__ci_filter_probe__"] = true
						} else {
							files[rel] = true
						}
						return nil
					})
					if err != nil {
						t.Errorf("%s job %s input %s: %v", name, job, item, err)
					}
				} else {
					files[item] = true
				}
			}
			for script := range scripts {
				covered := false
				for _, source := range in.sources {
					if script == source || strings.HasPrefix(script, strings.TrimSuffix(source, "/")+"/") {
						covered = true
						break
					}
				}
				if !covered {
					for _, event := range []string{"pull_request", "push"} {
						if _, ok := wf.On[event]; ok {
							t.Errorf("%s job %s event %s: script %s is missing from indirect source inventory", name, job, event, script)
						}
					}
				}
				files[script] = true
				in.sources = append(in.sources, script)
			}
			if len(requests) > 0 {
				files["go.mod"], files["go.sum"] = true, true
			}
			dirs := map[string]bool{}
			for tags, set := range requests {
				pkgs := sortedKeys(set)
				if len(pkgs) == 0 {
					continue
				}
				for _, arch := range []string{"amd64", "arm64"} {
					key := arch + "\x00" + tags + "\x00" + strings.Join(pkgs, "\x00")
					listed, ok := cache[key]
					if !ok {
						listed, err = listPackages(root, tags, arch, proxy, pkgs)
						if err != nil {
							t.Errorf("%s job %s: %v", name, job, err)
							continue
						}
						cache[key] = listed
					}
					for dir := range listed.dirs {
						dirs[dir] = true
					}
					for file := range listed.files {
						files[file] = true
					}
				}
				t.Logf("inventory %s/%s tags=%q packages=%s", name, job, tags, strings.Join(pkgs, ","))
			}
			for dir := range dirs {
				files[dir+"/__ci_filter_probe__.go"] = true
				entries, err := os.ReadDir(filepath.Join(root, dir))
				if err != nil {
					t.Errorf("%s job %s: %v", name, job, err)
					continue
				}
				for _, entry := range entries {
					if !entry.IsDir() {
						files[dir+"/"+entry.Name()] = true
					}
				}
			}
			for tags, set := range tests {
				sources, err := testSources(root, tags, proxy, sortedKeys(set))
				if err != nil {
					t.Errorf("%s job %s: %v", name, job, err)
				} else {
					in.sources = append(in.sources, sources...)
				}
			}
			problems, err := scanSources(root, in, dirs, files)
			if err != nil {
				t.Errorf("%s job %s: %v", name, job, err)
			}
			for _, problem := range problems {
				for _, event := range []string{"pull_request", "push"} {
					if _, ok := wf.On[event]; ok {
						t.Errorf("%s job %s event %s: %s", name, job, event, problem)
					}
				}
			}
			t.Logf("inventory %s/%s directories=%d", name, job, len(dirs))
			if os.Getenv("CI_FILTER_INVENTORY") == "1" {
				t.Logf("inventory-dirs %s/%s %s", name, job, strings.Join(sortedKeys(dirs), ","))
			}
			for _, event := range []string{"pull_request", "push"} {
				node, exists := wf.On[event]
				if !exists {
					continue
				}
				var filter eventFilter
				if err := node.Decode(&filter); err != nil {
					t.Errorf("%s event %s: %v", name, event, err)
					continue
				}
				if len(filter.Paths) > 0 && len(filter.PathsIgnore) > 0 {
					t.Errorf("%s job %s event %s: paths and paths-ignore cannot coexist", name, job, event)
					continue
				}
				patterns := filter.Paths
				if len(filter.PathsIgnore) > 0 {
					patterns = filter.PathsIgnore
				}
				compiled, err := compilePatterns(patterns)
				if err != nil {
					t.Errorf("%s job %s event %s: %v", name, job, event, err)
					continue
				}
				for _, file := range sortedKeys(files) {
					included := len(patterns) == 0 || matches(compiled, file)
					if len(filter.PathsIgnore) > 0 {
						included = !matches(compiled, file)
					}
					if !included {
						t.Errorf("%s job %s event %s: filter misses %s", name, job, event, file)
					}
				}
			}
		}
		for job := range indirectInputs[name] {
			if _, ok := wf.Jobs[job]; !ok {
				t.Errorf("%s: inventoried job %s is missing", name, job)
			}
		}
	}
}
