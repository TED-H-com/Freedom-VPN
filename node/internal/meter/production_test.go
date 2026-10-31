package meter

import (
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
	"time"
)

// D-112: this guards the actual production construction site, so adding an
// interval override there requires consciously revisiting the central bound.
// Deployment order: raise central sampling_interval_bound_seconds first,
// then raise the meter interval, keeping it at or below that bound.
func TestD112_ProductionSamplingInterval(t *testing.T) {
	if DefaultSampleInterval != 10*time.Second {
		t.Fatalf("production default %s exceeds or changes D-112's 10s contract", DefaultSampleInterval)
	}
	s, err := usagestore.Open(filepath.Join(t.TempDir(), "usage.db"), nodeID, incID, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	m, err := New(Config{NodeID: nodeID, IncarnationID: incID, Store: s, Source: &fakeSrc{obs: observe("boot")}})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if m.interval != DefaultSampleInterval {
		t.Fatalf("default resolves to %s", m.interval)
	}
	f, err := parser.ParseFile(token.NewFileSet(), "../../agent/usage.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "New" {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "meter" {
			return true
		}
		found++
		cfg, ok := call.Args[0].(*ast.CompositeLit)
		if !ok {
			t.Error("review production meter configuration against D-112")
			return true
		}
		for _, el := range cfg.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				t.Error("production config must use named fields")
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if ok && key.Name == "Interval" {
				t.Error("production now overrides Interval: verify <= central bound and update this guard")
			}
		}
		return true
	})
	if found != 1 {
		t.Fatalf("found %d production meter constructors, want 1", found)
	}
}
