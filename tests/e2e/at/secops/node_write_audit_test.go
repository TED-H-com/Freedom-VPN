//go:build e2e

package secops

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The runtime dispatcher also rejects indirect node paths, so concatenating a
// path outside the AST's local alias analysis cannot silently bypass WriteNode.
func nodeMutation15(method, path string) bool {
	return method != "GET" && method != "HEAD" && strings.Contains(path, "/api/v1/admin/nodes/")
}

func nodeBypasses15(src string) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "probe.go", src, 0)
	if err != nil {
		return nil, err
	}
	var bad []string
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		values := map[string]string{}
		var value func(ast.Expr) string
		value = func(e ast.Expr) string {
			switch x := e.(type) {
			case *ast.BasicLit:
				v, _ := strconv.Unquote(x.Value)
				return v
			case *ast.Ident:
				return values[x.Name]
			case *ast.BinaryExpr:
				return value(x.X) + value(x.Y)
			case *ast.ParenExpr:
				return value(x.X)
			}
			return ""
		}
		// Fixed-point propagation catches intermediate aliases and concatenation.
		for n := 0; n < 20; n++ {
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if a, ok := node.(*ast.AssignStmt); ok {
					for i, l := range a.Lhs {
						if id, ok := l.(*ast.Ident); ok && i < len(a.Rhs) {
							if v := value(a.Rhs[i]); v != "" {
								values[id.Name] = v
							}
						}
					}
				}
				if v, ok := node.(*ast.ValueSpec); ok {
					for i, id := range v.Names {
						if i < len(v.Values) {
							values[id.Name] = value(v.Values[i])
						}
					}
				}
				return true
			})
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name == "callRaw" && fn.Name.Name != "call" && fn.Name.Name != "roles" {
				bad = append(bad, fn.Name.Name+": unrestricted dispatcher")
			}
			method, path := -1, -1
			switch sel.Sel.Name {
			case "call", "callRaw":
				method, path = 1, 2
			case "Call":
				method, path = 2, 3
			case "NewRequest":
				method, path = 0, 1
			case "NewRequestWithContext":
				method, path = 1, 2
			case "Post", "Put", "Patch", "Delete":
				method, path = -1, 0
			}
			if path < 0 || path >= len(call.Args) {
				return true
			}
			m := "POST"
			if method >= 0 {
				m = value(call.Args[method])
			}
			if nodeMutation15(m, value(call.Args[path])) {
				bad = append(bad, fn.Name.Name+": node write via "+sel.Sel.Name)
			}
			if m != "GET" && m != "HEAD" && value(call.Args[path]) == "" {
				// The only unresolved dispatchers are audited below. New dynamic
				// write paths fail the audit until their non-node scope is proved.
				switch fn.Name.Name {
				case "call", "callRaw", "roles", "otherBodyBoundaries", "grantCommand15":
				default:
					bad = append(bad, fn.Name.Name+": unaudited dynamic write path")
				}
			}
			return true
		})
	}
	return bad, nil
}

func TestAPI01_NodeWritesUseSharedHelper(t *testing.T) {
	// Exceptions: liveProbe.roles sends the OpenAPI role-negative matrix and
	// asserts 401; liveProbe.call is the guarded non-node dispatcher. Neither
	// is an accepted node mutation. The member stale-ETag 412 stays untouched.
	for path, tokens := range map[string][]string{
		"live_test.go":       {"if nodeMutation15(method, path)", "node writes require Stack.WriteNode"},
		"grant_standing.go":  {`!strings.HasPrefix(path, "/api/v1/admin/members/")`, "member command path required"},
		"proxyprobe/main.go": {`in.Path != "/api/v1/agent/enroll"`},
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, token := range tokens {
			if !strings.Contains(string(raw), token) {
				t.Errorf("%s: dispatcher guard missing: %s", path, token)
			}
		}
	}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		bad, err := nodeBypasses15(string(b))
		if err != nil {
			return err
		}
		for _, v := range bad {
			t.Errorf("%s: %s", path, v)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPI01_NodeWriteAuditRejectsBypasses(t *testing.T) {
	for _, source := range []string{
		`func bad(){ p.call(admin,"PATCH","/api/v1/admin/nodes/"+id,body,nil) }`,
		`func bad(){ path:="/api/v1/admin/nodes/"+id; alias:=path; p.call(admin,"PUT",alias+"/publication",body,nil) }`,
		`func bad(){ http.NewRequest("POST","/api/v1/admin/nodes/"+id+"/revoke",nil) }`,
		`func bad(){ p.callRaw(admin,method,path,body,nil) }`,
		`func bad(){ p.call(admin,"POST",computedNodePath(),body,nil) }`,
	} {
		bad, err := nodeBypasses15("package probe;" + source)
		if err != nil || len(bad) == 0 {
			t.Fatalf("bypass accepted: %s (%v)", source, err)
		}
	}
	for _, source := range []string{
		`func good(){ p.writeNode("capacity",admin,"PATCH",path,path,body,etag,labstack.NodeFieldsMatch(body)) }`,
		`func good(){ p.call(admin,"GET","/api/v1/admin/nodes/"+id,nil,nil) }`,
		`func negative(){ p.call(admin,"PATCH","/api/v1/admin/members/"+id,body,stale) }`,
	} {
		bad, err := nodeBypasses15("package probe;" + source)
		if err != nil || len(bad) != 0 {
			t.Fatalf("allowed call rejected: %v %v", bad, err)
		}
	}
}
