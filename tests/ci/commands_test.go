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
