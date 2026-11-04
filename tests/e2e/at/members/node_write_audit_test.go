package members

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Dispatch boundaries are the only exceptions. Their callers are audited
// below, including fenceCall's expected-error tests (which remain direct).
// No node-write exception is currently needed: this lane has no deliberate
// stale-node-ETag test. A future one must be explicitly reviewed here.
var nodeWriteAuditExceptions = map[string]string{
	"live_test.go:api:request":                   "API dispatcher; every api call is audited",
	"live_test.go:request:NewRequestWithContext": "HTTP transport; every request call is audited",
	"live_test.go:request:Do":                    "HTTP transport; every request call is audited",
	"reauthorization_test.go:fenceCall:api":      "device fence dispatcher; every fenceCall is audited, including expected errors",
	"devices_billing_test.go:nb:request":         "NetBird dispatcher with fixed NetBird origin, not the control node API",
}

func auditNodeWrites(filename string, source any) ([]string, error) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, filename, source, 0)
	if err != nil {
		return nil, err
	}
	failures := []string{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		bindings := map[string]ast.Expr{}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if a, ok := n.(*ast.AssignStmt); ok {
				for i, left := range a.Lhs {
					if id, ok := left.(*ast.Ident); ok && i < len(a.Rhs) {
						bindings[id.Name] = a.Rhs[i]
					}
				}
			}
			return true
		})
		var value func(ast.Expr, map[string]bool) string
		value = func(e ast.Expr, seen map[string]bool) string {
			switch e := e.(type) {
			case *ast.BasicLit:
				if e.Kind == token.STRING {
					s, _ := strconv.Unquote(e.Value)
					return s
				}
			case *ast.BinaryExpr:
				if e.Op == token.ADD {
					return value(e.X, seen) + value(e.Y, seen)
				}
			case *ast.Ident:
				if !seen[e.Name] {
					seen[e.Name] = true
					v := value(bindings[e.Name], seen)
					delete(seen, e.Name)
					return v
				}
			case *ast.CallExpr:
				if id, ok := e.Fun.(*ast.Ident); ok && id.Name == "authorizePath" {
					return "/api/v1/admin/members/?/devices/?/authorize"
				}
			}
			return "?"
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			name := sel.Sel.Name
			key := filepath.Base(filename) + ":" + fn.Name.Name + ":" + name
			if _, ok := nodeWriteAuditExceptions[key]; ok {
				return true
			}
			method, path := -1, -1
			switch name {
			case "api", "request":
				method, path = 1, 2
			case "fenceCall":
				method, path = 3, 4
			case "Call":
				method, path = 2, 3
			case "NewRequest":
				method, path = 0, 1
			case "NewRequestWithContext":
				method, path = 1, 2
			case "WriteNode":
				if filepath.Base(filename) != "identity_nodes_test.go" || fn.Name.Name != "nodeMutation" {
					failures = append(failures, fs.Position(call.Pos()).String()+": node write must retain receipt through nodeMutation")
				}
				return true
			case "Do", "Post", "PostForm":
				// Probe transports are read-only; their request construction is checked.
				if strings.Contains(filepath.ToSlash(filename), "cmd/probe/") && name == "Do" {
					return true
				}
				failures = append(failures, fs.Position(call.Pos()).String()+": unaudited HTTP transport")
				return true
			default:
				return true
			}
			if path >= len(call.Args) {
				return true
			}
			m, p := value(call.Args[method], map[string]bool{}), value(call.Args[path], map[string]bool{})
			if m == "GET" || m == "HEAD" {
				return true
			}
			// Only known non-node API prefixes may bypass WriteNode. Unknown dynamic
			// destinations fail closed; concatenations/aliases are resolved above.
			known := false
			for _, prefix := range []string{"/api/v1/admin/members/", "/api/v1/admin/policies", "/api/v1/admin/groups", "/api/v1/admin/enrollments", "/api/v1/member/", "/api/v1/webhooks/stripe", "https://api.stripe.com/e2e/", "https://agent.example.invalid:9443/api/v1/agent/usage-batches", "/auth/", "https://releases.example.invalid/", "https://portal.example.invalid:8443/api/v1/billing/"} {
				if strings.HasPrefix(p, prefix) {
					known = true
				}
			}
			if !known || strings.Contains(p, "/admin/nodes") {
				failures = append(failures, fmt.Sprintf("%s: bypass %s %s", fs.Position(call.Pos()), m, p))
			}
			return true
		})
	}
	return failures, nil
}
func TestF20_NodeWriteAudit(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		failures, e := auditNodeWrites(path, nil)
		if e != nil {
			return e
		}
		for _, f := range failures {
			t.Error(f)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for site, reason := range nodeWriteAuditExceptions {
		t.Log(site + ": " + reason)
	}
}
func TestF20_NodeWriteAuditRejectsBypasses(t *testing.T) {
	for _, body := range []string{
		`l.api(l.admin,"PATCH","/api/v1/admin/nodes/"+id,body,nil)`,
		`path:="/api/v1/admin/nodes/"+id; alias:=path; l.api(l.admin,"POST",alias+"/revoke",body,nil)`,
		`l.request(l.admin,"PUT",unknown,body,nil)`,
		`stack.Call(ctx,b,"DELETE","/api/v1/admin/nodes/"+id,nil,nil)`,
		`http.NewRequest("POST","/api/v1/admin/nodes/"+id,nil)`,
		`stack.WriteNode(ctx,b,"PATCH",path,path,body,etag,predicate,202)`,
	} {
		violations, err := auditNodeWrites("injected.go", "package members; func bypass(){"+body+"}")
		if err != nil || len(violations) == 0 {
			t.Fatalf("audit missed %s: %v %v", body, violations, err)
		}
	}
}
