package ci

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMakeGoCommandExtraction(t *testing.T) {
	commands := `go test -tags=e2e -o ./generated.test ./tests/e2e/at/usage/... ./pkg/new
for pkg in ./one ./two; do go build -o ./bin/output $pkg; done
go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags integration ./internal/example/...`
	requests, tests := map[string]map[string]bool{}, map[string]map[string]bool{}
	if err := parseGoCommands(commands, requests, tests); err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"":            {"./one", "./two"},
		"e2e":         {"./pkg/new", "./tests/e2e/at/usage/..."},
		"integration": {"./internal/example/..."},
	}
	for tags, packages := range want {
		if got := sortedKeys(requests[tags]); !reflect.DeepEqual(got, packages) {
			t.Errorf("tags %q: got %v, want %v", tags, got, packages)
		}
	}
	if got := sortedKeys(tests["e2e"]); !reflect.DeepEqual(got, want["e2e"]) {
		t.Errorf("tested packages = %v, want %v", got, want["e2e"])
	}
	if err := parseGoCommands("go test $unknown", requests, tests); err == nil {
		t.Error("unresolved test package should fail closed")
	}
}

func TestGofmtInputExtraction(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	tracked, err := trackedFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, command, form string
		want                []string
	}{
		{"git pathspec", `test -z "$(gofmt -l $(git ls-files --cached --others --exclude-standard 'pkg/decstr/*.go'))"`, "git ls-files", []string{"pkg/decstr/decstr.go", "pkg/decstr/decstr_test.go"}},
		{"rg directories", `gofmt -l $(rg --files pkg/decstr pkg/canonjson -g '*.go')`, "rg --files", []string{"pkg/canonjson/canonjson.go", "pkg/canonjson/canonjson_test.go", "pkg/canonjson/review_test.go", "pkg/decstr/decstr.go", "pkg/decstr/decstr_test.go"}},
		{"literal paths and shell glob", `gofmt -l pkg/decstr pkg/canonjson/*test.go pkg/envelope/envelope.go`, "paths", []string{"pkg/canonjson/canonjson_test.go", "pkg/canonjson/review_test.go", "pkg/decstr/decstr.go", "pkg/decstr/decstr_test.go", "pkg/envelope/envelope.go"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sets, err := gofmtInputs(root, tt.command, tracked)
			if err != nil || len(sets) != 1 {
				t.Fatalf("sets = %v, error = %v", sets, err)
			}
			if sets[0].form != tt.form || !reflect.DeepEqual(sets[0].files, tt.want) {
				t.Errorf("got %s %v; want %s %v", sets[0].form, sets[0].files, tt.form, tt.want)
			}
		})
	}
	sets, err := gofmtInputs(root, `gofmt -l $(git ls-files --cached --others --exclude-standard 'pkg/*.go')`, tracked)
	if err != nil || len(sets) != 1 || !contains(sets[0].files, "pkg/decstr/decstr.go") {
		t.Errorf("git pathspec star must cross directories: %v, %v", sets, err)
	}
	withNested := append(append([]string(nil), tracked...), "pkg/decstr/nested/extra.go")
	sets, err = gofmtInputs(root, `gofmt -l pkg/decstr/*.go`, withNested)
	if err != nil || len(sets) != 1 || contains(sets[0].files, "pkg/decstr/nested/extra.go") {
		t.Errorf("shell glob star must stop at a slash: %v, %v", sets, err)
	}
	for _, command := range []string{`gofmt -l $(find pkg -name '*.go')`, `gofmt -l $files`, `gofmt -w pkg/decstr`} {
		if _, err := gofmtInputs(root, command, tracked); err == nil || !strings.Contains(err.Error(), "gofmt") {
			t.Errorf("unsupported command %q: %v", command, err)
		}
	}
	sets, err = gofmtInputs(root, `echo 'gofmt required'; echo "gofmt needed: $(gofmt -l pkg/decstr)"; gofmt -l pkg/decstr`, tracked)
	if err != nil || len(sets) != 2 {
		t.Errorf("commands in shell and quoted substitutions: %v, %v", sets, err)
	}
	if _, err := trackedFiles(t.TempDir()); err == nil || !strings.Contains(err.Error(), "needs a git checkout") {
		t.Errorf("missing checkout error = %v", err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestIndirectWiringRequiresReview(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "apps/portal"), 0o700); err != nil {
		t.Fatal(err)
	}
	script := []byte("dynamic build helper\n")
	if err := os.WriteFile(filepath.Join(root, "helper.sh"), script, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "apps/portal/package.json")
	if err := os.WriteFile(manifest, []byte(`{"scripts":{"build":"build-command"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	in := indirect{pins: map[string]string{"helper.sh": fmt.Sprintf("%x", sha256.Sum256(script))}, npm: map[string]string{"build": "build-command"}}
	if problems, err := checkWiring(root, in); err != nil || len(problems) != 0 {
		t.Fatalf("unchanged wiring: %v, %v", problems, err)
	}
	if err := os.WriteFile(filepath.Join(root, "helper.sh"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if problems, err := checkWiring(root, in); err != nil || len(problems) != 1 || !strings.Contains(problems[0], "helper.sh") {
		t.Fatalf("changed helper: %v, %v", problems, err)
	}
	if err := os.WriteFile(filepath.Join(root, "helper.sh"), script, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"scripts":{"build":"build-command","prebuild":"new-command"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if problems, err := checkWiring(root, in); err != nil || len(problems) != 1 || !strings.Contains(problems[0], "package.json") {
		t.Fatalf("new lifecycle script: %v, %v", problems, err)
	}
}
