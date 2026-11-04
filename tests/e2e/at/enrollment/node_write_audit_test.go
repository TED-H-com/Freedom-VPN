package enrollment

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

// No node-write exception: this lane has no deliberate stale-node-ETag test.
// Dispatchers are exceptions only at their transport boundary; their callers
// remain audited. The lost-response fixture must retain its single-shot fault.
var nodeWriteAuditExceptions = map[string]string{
	"live_test.go:call:callHeaders":                                "API dispatcher; every call site is audited",
	"live_test.go:callHeaders:Call":                                "API dispatcher; every callHeaders site is audited",
	"live_test.go:post:NewRequestWithContext":                      "anonymous agent POST dispatcher; every post URL is audited",
	"live_test.go:post:Do":                                         "anonymous agent POST transport; every post URL is audited",
	"proof.go:RoundTrip:RoundTrip":                                 "lost-enrollment-response fault transport; request constructors and callers are audited",
	"proof_test.go:TestAT06_ResponseActuallyLostBeforeCaller:Post": "loopback-only lost-response test; two deliberate single-shot calls measure response loss and recovery",
}

func auditNodeWrites(filename string, source any) ([]string, map[string]bool, error) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, filename, source, 0)
	if err != nil {
		return nil, nil, err
	}
	failures := []string{}
	exceptions := map[string]bool{}
	// Retain all assignments, including var/const declarations. An ambiguous
	// method or destination is unknown and fails closed for a possible write.
	bindings := map[string][]ast.Expr{}
	ast.Inspect(file, func(n ast.Node) bool {
		bind := func(names []ast.Expr, values []ast.Expr) {
			for i, left := range names {
				if id, ok := left.(*ast.Ident); ok && i < len(values) {
					bindings[id.Name] = append(bindings[id.Name], values[i])
				}
			}
		}
		switch n := n.(type) {
		case *ast.AssignStmt:
			bind(n.Lhs, n.Rhs)
		case *ast.ValueSpec:
			for i, name := range n.Names {
				if i < len(n.Values) {
					bindings[name.Name] = append(bindings[name.Name], n.Values[i])
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
		case *ast.ParenExpr:
			return value(e.X, seen)
		case *ast.BinaryExpr:
			if e.Op == token.ADD {
				return value(e.X, seen) + value(e.Y, seen)
			}
		case *ast.Ident:
			if !seen[e.Name] && len(bindings[e.Name]) > 0 {
				seen[e.Name] = true
				defer delete(seen, e.Name)
				v := value(bindings[e.Name][0], seen)
				for _, other := range bindings[e.Name][1:] {
					if value(other, seen) != v {
						return "?"
					}
				}
				return v
			}
		case *ast.SelectorExpr:
			if pkg, ok := e.X.(*ast.Ident); ok && pkg.Name == "http" {
				return map[string]string{"MethodGet": "GET", "MethodHead": "HEAD", "MethodPatch": "PATCH", "MethodPut": "PUT", "MethodPost": "POST", "MethodDelete": "DELETE"}[e.Sel.Name]
			}
		case *ast.CallExpr:
			if sel, ok := e.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "PublicURL" {
				return "https://control.example.invalid"
			}
		}
		return "?"
	}
	var names func(ast.Expr, map[string]bool) []string
	names = func(e ast.Expr, seen map[string]bool) []string {
		switch e := e.(type) {
		case *ast.SelectorExpr:
			return []string{e.Sel.Name}
		case *ast.ParenExpr:
			return names(e.X, seen)
		case *ast.Ident:
			if !seen[e.Name] {
				seen[e.Name] = true
				defer delete(seen, e.Name)
				var out []string
				for _, binding := range bindings[e.Name] {
					out = append(out, names(binding, seen)...)
				}
				return out
			}
		}
		return nil
	}
	for _, decl := range file.Decls {
		function := "<global>"
		if fn, ok := decl.(*ast.FuncDecl); ok {
			function = fn.Name.Name
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, name := range names(call.Fun, map[string]bool{}) {
				key := filepath.Base(filename) + ":" + function + ":" + name
				if _, ok := nodeWriteAuditExceptions[key]; ok {
					exceptions[key] = true
					continue
				}
				method, path := -1, -1
				switch name {
				case "call", "callHeaders":
					method, path = 0, 1
				case "post":
					path = 2
				case "Call":
					method, path = 2, 3
				case "NewRequest":
					method, path = 0, 1
				case "NewRequestWithContext":
					method, path = 1, 2
				case "WriteNode":
					if filepath.Base(filename) != "node_write_test.go" || function != "writeNode" {
						failures = append(failures, fs.Position(call.Pos()).String()+": use receipt-embedding writeNode")
					}
					continue
				case "Do", "Post", "PostForm", "RoundTrip":
					failures = append(failures, fs.Position(call.Pos()).String()+": unaudited HTTP transport")
					continue
				default:
					continue
				}
				if path >= len(call.Args) {
					failures = append(failures, fs.Position(call.Pos()).String()+": unknown dispatcher signature")
					continue
				}
				m := "POST"
				if method >= 0 {
					m = value(call.Args[method], map[string]bool{})
				}
				p := value(call.Args[path], map[string]bool{})
				if m == "GET" || m == "HEAD" {
					continue
				}
				p = strings.TrimPrefix(p, "https://control.example.invalid")
				known := false
				for _, prefix := range []string{"/api/v1/admin/enrollments", "/api/v1/admin/groups", "/api/v1/admin/members/", "/api/v1/agent/enroll", "https://agent.example.invalid:9443/api/v1/agent/bootstrap"} {
					if strings.HasPrefix(p, prefix) {
						known = true
					}
				}
				if !known || strings.Contains(p, "/admin/nodes") {
					failures = append(failures, fmt.Sprintf("%s: bypass %s %s", fs.Position(call.Pos()), m, p))
				}
			}
			return true
		})
	}
	return failures, exceptions, nil
}

func TestF10_NodeWriteAudit(t *testing.T) {
	used := map[string]bool{}
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		failures, exceptions, err := auditNodeWrites(path, nil)
		if err != nil {
			return err
		}
		for _, failure := range failures {
			t.Error(failure)
		}
		for exception := range exceptions {
			used[exception] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for site, reason := range nodeWriteAuditExceptions {
		if !used[site] {
			t.Error("stale audit exception", site)
		}
		t.Log(site + ": " + reason)
	}
}

func TestF10_NodeWriteAuditRejectsBypasses(t *testing.T) {
	for _, body := range []string{
		`p.call("PATCH","/api/v1/admin/nodes/"+id,body)`,
		`path:="/api/v1/admin/nodes/"+id; alias:=path; p.callHeaders("POST",alias+"/revoke",body,nil)`,
		`p.callHeaders("PUT",unknown,body,nil)`,
		`stack.Call(ctx,b,"DELETE","/api/v1/admin/nodes/"+id,nil,nil)`,
		`http.NewRequest("POST","/api/v1/admin/nodes/"+id,nil)`,
		`http.NewRequestWithContext(ctx,http.MethodPatch,path,body)`,
		`stack.WriteNode(ctx,b,"PATCH",path,path,body,etag,predicate,202)`,
		`send:=p.callHeaders; send("PATCH",path,body,nil)`,
		`var path="/api/v1/admin/nodes/"+id; const method="PUT"; p.call(method,path,body)`,
		`method:="PATCH"; method="GET"; p.call(method,path,body)`,
		`client.Do(req)`,
		`p.post(ctx,client,"/api/v1/admin/nodes/"+id,token,body)`,
	} {
		failures, _, err := auditNodeWrites("injected.go", "package enrollment; func bypass(){"+body+"}")
		if err != nil || len(failures) == 0 {
			t.Fatalf("audit missed %s: %v %v", body, failures, err)
		}
	}
	for _, body := range []string{
		`p.call("GET",unknown,nil)`,
		`p.call(http.MethodGet,"/api/v1/admin/nodes/"+id,nil)`,
		`p.call("POST","/api/v1/admin/enrollments",body)`,
	} {
		failures, _, err := auditNodeWrites("injected.go", "package enrollment; func allowed(){"+body+"}")
		if err != nil || len(failures) != 0 {
			t.Fatalf("audit rejected read/non-node command %s: %v %v", body, failures, err)
		}
	}
}
