package ci

import (
	"reflect"
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
