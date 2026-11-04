package usage

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

// Dispatch boundaries are the only exceptions; every caller of those helpers is
// audited below. No node-write exception is needed in this lane: every node
// mutation in tests/e2e/at/usage goes through liveSuite.nodeMutation ->
// Stack.WriteNode, and the expected-error precondition test in evalAT19 targets
// /api/v1/admin/policies/{id}/activate rather than /api/v1/admin/nodes.
var nodeWriteAuditExceptions = map[string]string{
	"live_test.go:apiCall:doHTTP":                            "portal API dispatcher; every apiCall caller is audited directly",
	"live_test.go:doHTTP:NewRequestWithContext":              "portal HTTP transport constructor; every doHTTP caller is audited directly",
	"live_test.go:doHTTP:Do":                                 "portal HTTP transport execution; every doHTTP caller is audited directly",
	"live_test.go:agentCall:agentCallWithClient":             "mTLS agent dispatcher; every agentCall caller is audited directly",
	"live_test.go:agentCallWithClient:NewRequestWithContext": "mTLS agent (:9443) transport constructor; every agentCallWithClient caller is audited directly",
	"live_test.go:agentCallWithClient:Do":                    "mTLS agent (:9443) transport execution; every agentCallWithClient caller is audited directly",
	"live_helpers_test.go:originProbe:Do":                    "origin data-plane probe transport execution; NewRequestWithContext in originProbe is audited directly against https://releases.example.invalid/e2e/traffic",
}

func auditNodeWrites(filename string, source any) ([]string, error) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, filename, source, 0)
	if err != nil {
		return nil, err
	}
	var failures []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		bindings := map[string]ast.Expr{}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch stmt := n.(type) {
			case *ast.AssignStmt:
				for i, left := range stmt.Lhs {
					if id, ok := left.(*ast.Ident); ok && i < len(stmt.Rhs) {
						bindings[id.Name] = stmt.Rhs[i]
					}
				}
			case *ast.ValueSpec:
				for i, id := range stmt.Names {
					if i < len(stmt.Values) {
						bindings[id.Name] = stmt.Values[i]
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
			case *ast.SelectorExpr:
				if x, ok := e.X.(*ast.Ident); ok && x.Name == "http" {
					switch e.Sel.Name {
					case "MethodGet":
						return "GET"
					case "MethodHead":
						return "HEAD"
					case "MethodPost":
						return "POST"
					case "MethodPut":
						return "PUT"
					case "MethodPatch":
						return "PATCH"
					case "MethodDelete":
						return "DELETE"
					}
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
			methodIdx, pathIdx := -1, -1
			switch name {
			case "apiCall", "doHTTP", "agentCallWithClient":
				methodIdx, pathIdx = 1, 2
			case "agentCall":
				methodIdx, pathIdx = 0, 1
			case "Call":
				methodIdx, pathIdx = 2, 3
			case "NewRequest":
				methodIdx, pathIdx = 0, 1
			case "NewRequestWithContext":
				methodIdx, pathIdx = 1, 2
			case "WriteNode":
				if filepath.Base(filename) != "live_test.go" || fn.Name.Name != "nodeMutation" {
					failures = append(failures, fs.Position(call.Pos()).String()+": node write must retain receipt through nodeMutation")
				}
				return true
			case "Do", "Post", "PostForm":
				failures = append(failures, fs.Position(call.Pos()).String()+": unaudited HTTP transport")
				return true
			default:
				return true
			}
			if pathIdx >= len(call.Args) {
				return true
			}
			m := value(call.Args[methodIdx], map[string]bool{})
			p := value(call.Args[pathIdx], map[string]bool{})
			if m == "GET" || m == "HEAD" {
				return true
			}
			known := false
			for _, prefix := range []string{
				"/api/v1/admin/policies",
				"/api/v1/admin/groups",
				"/api/v1/admin/members/",
				"/api/v1/admin/enrollments",
				"/api/v1/agent/usage-batches",
				"/api/v1/agent/observations",
				"https://portal.example.invalid:8443/api/v1/agent/enroll",
				"https://releases.example.invalid/e2e/traffic",
			} {
				if strings.HasPrefix(p, prefix) {
					known = true
					break
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

func TestNodeWriteAudit(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		failures, parseErr := auditNodeWrites(path, nil)
		if parseErr != nil {
			return parseErr
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

func TestNodeWriteAuditRejectsBypasses(t *testing.T) {
	for _, body := range []string{
		`s.apiCall(s.admin, http.MethodPatch, "/api/v1/admin/nodes/"+id, body, nil)`,
		`path := "/api/v1/admin/nodes/" + id; alias := path; s.apiCall(s.admin, http.MethodPost, alias+"/revoke", body, nil)`,
		`s.doHTTP(s.admin, http.MethodPut, "https://portal.example.invalid:8443/api/v1/admin/nodes/"+id+"/publication", body, nil)`,
		`s.apiCall(s.admin, http.MethodPut, unknownPath, body, nil)`,
		`s.hook.Call(ctx, b, http.MethodDelete, "/api/v1/admin/nodes/"+id, nil, nil)`,
		`http.NewRequest(http.MethodPost, "/api/v1/admin/nodes/"+id+"/revoke", nil)`,
		`http.NewRequestWithContext(ctx, http.MethodPatch, "https://portal.example.invalid:8443/api/v1/admin/nodes/"+id, nil)`,
		`client.Do(req)`,
		`s.hook.WriteNode(ctx, b, http.MethodPatch, path, path, body, etag, predicate, 202)`,
	} {
		violations, err := auditNodeWrites("injected.go", "package usage; func bypass() { "+body+" }")
		if err != nil || len(violations) == 0 {
			t.Fatalf("audit missed %s: violations=%v err=%v", body, violations, err)
		}
	}
}
