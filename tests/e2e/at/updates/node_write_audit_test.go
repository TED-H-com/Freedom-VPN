package updates

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// approvedNodeWriteSites lists every node resource write in tests/e2e/at/updates.
// Each site must route through Stack.WriteNode with a non-nil intended-state predicate.
var approvedNodeWriteSites = map[string]string{
	"installer_test.go:retire": "DELETE /api/v1/admin/nodes/{node} via Stack.WriteNode with retiredNode (lifecycle_state=retired) and fixture_retirement_node_write receipt",
}

// nodeWriteAuditExceptions lists every non-WriteNode HTTP or agent-POST site in
// tests/e2e/at/updates with the exact reason it is not a node resource write.
var nodeWriteAuditExceptions = map[string]string{
	"authority_test.go:conflictBounds:nodePOST":          "agent mTLS POST /api/v1/agent/update-results conflict probe",
	"authority_test.go:failedRequiresCause:nodePOST":     "agent mTLS POST /api/v1/agent/update-results error_code validation probe",
	"authority_test.go:observationAuthority:nodePOST":    "agent mTLS POST /api/v1/agent/observations authority probe",
	"installer_receipts_test.go:resultAcceptance:submit": "agent mTLS POST /api/v1/agent/update-results receipt acceptance probe",
	"installer_receipts_test.go:resultCorrection:submit": "agent mTLS POST /api/v1/agent/update-results receipt correction probe",
	"installer_test.go:central:call":                     "read-only GET /api/v1/admin/nodes/{node} for ETag and observed-state readback",
	"installer_test.go:installerHandoff:call":            "POST /api/v1/admin/enrollments (enrollment creation, not a node resource write)",
	"installer_test.go:installerHandoff:submit":          "agent mTLS POST /api/v1/agent/update-results failure receipt probe",
	"lab_test.go:call:Do":                                "session.call internal HTTP transport dispatch; every caller of session.call is audited by AST",
	"lab_test.go:call:NewRequestWithContext":             "session.call internal HTTP request constructor; every caller of session.call is audited by AST",
	"lab_test.go:enroll:call":                            "POST /api/v1/admin/enrollments (enrollment creation, not a node resource write)",
	"lab_test.go:login:Do":                               "session bootstrap GET /auth/session after portal OIDC login",
	"lab_test.go:login:NewRequestWithContext":            "session bootstrap GET /auth/session after portal OIDC login",
	"lab_test.go:putObject:Do":                           "release artifact origin upload PUT /e2e/object (not a central node resource write)",
	"lab_test.go:putObject:NewRequestWithContext":        "release artifact origin upload PUT /e2e/object (not a central node resource write)",
	"live_test.go:createBatch:call":                      "GET /api/v1/admin/releases and POST /api/v1/admin/update-batches (release and batch endpoints)",
	"live_test.go:target:call":                           "read-only GET /api/v1/admin/update-batches/{id}",
	"preresult_test.go:preResultInjection:nodePOST":      "agent mTLS POST /api/v1/agent/observations and /api/v1/agent/update-results pre-result probe",
	"preresult_test.go:recordedInjection:nodePOST":       "agent mTLS POST /api/v1/agent/observations post-result probe",
	"preresult_test.go:unexpectedAudit:call":             "read-only GET /api/v1/admin/audit-events",
	"restated_test.go:offlineBatch:nodePOST":             "agent mTLS POST /api/v1/agent/observations offline catch-up probe",
}

func TestUPD02_NodeWriteAudit(t *testing.T) {
	violations, seenWrites, seenExceptions := auditUpdatesDir(t, ".")
	for site := range approvedNodeWriteSites {
		if !seenWrites[site] {
			violations = append(violations, fmt.Sprintf("approved WriteNode site %q was not found in AST", site))
		}
	}
	for site := range nodeWriteAuditExceptions {
		if !seenExceptions[site] {
			violations = append(violations, fmt.Sprintf("audit exception %q was not found in AST", site))
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("unapproved node write or HTTP dispatch in tests/e2e/at/updates:\n%s", strings.Join(violations, "\n"))
	}
}

func TestUPD02_NodeWriteAuditRejectsBypasses(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "direct session.call DELETE on admin nodes",
			src: `package updates_test
func (h *handoffRun) retire(node string) {
	_, _ = h.s.admin.call(h.s.ctx, "DELETE", h.s.r.lab.public+"/api/v1/admin/nodes/"+node, nil, nil)
}`,
		},
		{
			name: "direct session.call PATCH on admin nodes via local var",
			src: `package updates_test
func bypass(s *session, id string) {
	path := "/api/v1/admin/nodes/" + id
	_, _ = s.call(nil, "PATCH", path, nil, nil)
}`,
		},
		{
			name: "direct Stack.Call POST on admin nodes",
			src: `package updates_test
func bypass(stack *Stack, b BrowserSession, id string) {
	_, _ = stack.Call(nil, b, "POST", "/api/v1/admin/nodes/"+id+":revoke", nil, nil)
}`,
		},
		{
			name: "raw http.NewRequestWithContext outside exceptions",
			src: `package updates_test
import "net/http"
func bypass() {
	_, _ = http.NewRequestWithContext(nil, "DELETE", "/api/v1/admin/nodes/x", nil)
}`,
		},
		{
			name: "raw http.Client.Do outside exceptions",
			src: `package updates_test
import "net/http"
func bypass(c *http.Client, req *http.Request) {
	_, _ = c.Do(req)
}`,
		},
		{
			name: "unapproved WriteNode site",
			src: `package updates_test
func unapproved(stack *Stack, b BrowserSession, id string) {
	_ = stack.WriteNode(nil, b, "DELETE", "/api/v1/admin/nodes/"+id, "/api/v1/admin/nodes/"+id, nil, "\"1\"", retiredNode, 202)
}`,
		},
		{
			name: "approved WriteNode site with nil predicate",
			src: `package updates_test
func (h *handoffRun) retire(node string) {
	_ = h.stack.WriteNode(h.s.ctx, nil, "DELETE", "/api/v1/admin/nodes/"+node, "/api/v1/admin/nodes/"+node, nil, "\"1\"", nil, 202)
}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "installer_test.go", tc.src, 0)
			if err != nil {
				t.Fatalf("parse snippet: %v", err)
			}
			var violations []string
			seenWrites := map[string]bool{}
			seenExceptions := map[string]bool{}
			auditUpdatesASTFile(fset, "installer_test.go", file, &violations, seenWrites, seenExceptions)
			if len(violations) == 0 {
				t.Fatalf("expected violation for %s, got none", tc.name)
			}
		})
	}
}

func auditUpdatesDir(t *testing.T, root string) ([]string, map[string]bool, map[string]bool) {
	t.Helper()
	fset := token.NewFileSet()
	var violations []string
	seenWrites := map[string]bool{}
	seenExceptions := map[string]bool{}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || d.Name() == "node_write_audit_test.go" {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = d.Name()
		}
		auditUpdatesASTFile(fset, filepath.ToSlash(rel), file, &violations, seenWrites, seenExceptions)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return violations, seenWrites, seenExceptions
}

func auditUpdatesASTFile(fset *token.FileSet, fileName string, file *ast.File, violations *[]string, seenWrites, seenExceptions map[string]bool) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		fnName := fn.Name.Name
		vars := map[string]string{}
		boolVars := map[string]bool{}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if assign, ok := n.(*ast.AssignStmt); ok {
				for i, lhs := range assign.Lhs {
					id, ok := lhs.(*ast.Ident)
					if !ok || i >= len(assign.Rhs) {
						continue
					}
					if s := resolveStringExpr(assign.Rhs[i], vars); s != "" {
						vars[id.Name] = s
					}
					if isRetiredPredicateExpr(assign.Rhs[i], boolVars) {
						boolVars[id.Name] = true
					}
				}
			}
			if spec, ok := n.(*ast.ValueSpec); ok {
				for i, id := range spec.Names {
					if i < len(spec.Values) {
						if s := resolveStringExpr(spec.Values[i], vars); s != "" {
							vars[id.Name] = s
						}
						if isRetiredPredicateExpr(spec.Values[i], boolVars) {
							boolVars[id.Name] = true
						}
					}
				}
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			selName := calledIdentName(call.Fun)
			if selName == "" {
				return true
			}
			pos := fset.Position(call.Pos())
			loc := fmt.Sprintf("%s:%d (%s)", fileName, pos.Line, fnName)

			if selName == "WriteNode" {
				siteKey := fmt.Sprintf("%s:%s", fileName, fnName)
				seenWrites[siteKey] = true
				if _, ok := approvedNodeWriteSites[siteKey]; !ok {
					*violations = append(*violations, fmt.Sprintf("%s: unapproved WriteNode call site %q", loc, siteKey))
					return true
				}
				if len(call.Args) < 8 || !isRetiredPredicateExpr(call.Args[7], boolVars) {
					*violations = append(*violations, fmt.Sprintf("%s: WriteNode must pass retiredNode / NodeFieldsMatch predicate", loc))
				}
				return true
			}

			switch selName {
			case "call", "Call":
				methodArg, urlArg := callMethodAndURL(selName, call.Args, vars)
				siteKey := fmt.Sprintf("%s:%s:%s", fileName, fnName, selName)
				if strings.Contains(urlArg, "/admin/nodes") && methodArg != "GET" {
					*violations = append(*violations, fmt.Sprintf("%s: direct %s(%q, %q) on /admin/nodes must use Stack.WriteNode", loc, selName, methodArg, urlArg))
					return true
				}
				if _, ok := nodeWriteAuditExceptions[siteKey]; !ok {
					*violations = append(*violations, fmt.Sprintf("%s: unreviewed %s(%q, %q) site %q", loc, selName, methodArg, urlArg, siteKey))
					return true
				}
				seenExceptions[siteKey] = true

			case "NewRequest", "NewRequestWithContext", "Do", "Post", "PostForm", "Head", "nodePOST", "submit":
				siteKey := fmt.Sprintf("%s:%s:%s", fileName, fnName, selName)
				for _, arg := range call.Args {
					if s := resolveStringExpr(arg, vars); strings.Contains(s, "/admin/nodes") {
						*violations = append(*violations, fmt.Sprintf("%s: raw %s targeting /admin/nodes (%q)", loc, selName, s))
						return true
					}
				}
				if _, ok := nodeWriteAuditExceptions[siteKey]; !ok {
					*violations = append(*violations, fmt.Sprintf("%s: unreviewed %s site %q", loc, selName, siteKey))
					return true
				}
				seenExceptions[siteKey] = true
			}
			return true
		})
	}
}

func calledIdentName(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.SelectorExpr:
		return v.Sel.Name
	case *ast.Ident:
		return v.Name
	default:
		return ""
	}
}

func callMethodAndURL(name string, args []ast.Expr, vars map[string]string) (string, string) {
	if name == "Call" && len(args) >= 4 {
		return resolveStringExpr(args[2], vars), resolveStringExpr(args[3], vars)
	}
	if name == "call" && len(args) >= 3 {
		return resolveStringExpr(args[1], vars), resolveStringExpr(args[2], vars)
	}
	return "", ""
}

func resolveStringExpr(expr ast.Expr, vars map[string]string) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			s, err := strconv.Unquote(v.Value)
			if err == nil {
				return s
			}
			return strings.Trim(v.Value, "`\"")
		}
	case *ast.BinaryExpr:
		if v.Op == token.ADD {
			return resolveStringExpr(v.X, vars) + resolveStringExpr(v.Y, vars)
		}
	case *ast.Ident:
		switch v.Name {
		case "MethodGet":
			return "GET"
		case "MethodPost":
			return "POST"
		case "MethodPut":
			return "PUT"
		case "MethodPatch":
			return "PATCH"
		case "MethodDelete":
			return "DELETE"
		}
		return vars[v.Name]
	case *ast.SelectorExpr:
		switch v.Sel.Name {
		case "MethodGet":
			return "GET"
		case "MethodPost":
			return "POST"
		case "MethodPut":
			return "PUT"
		case "MethodPatch":
			return "PATCH"
		case "MethodDelete":
			return "DELETE"
		}
	case *ast.CallExpr:
		if len(v.Args) > 0 {
			var parts []string
			for _, arg := range v.Args {
				if s := resolveStringExpr(arg, vars); s != "" {
					parts = append(parts, s)
				}
			}
			return strings.Join(parts, "")
		}
	}
	return ""
}

func isRetiredPredicateExpr(expr ast.Expr, boolVars map[string]bool) bool {
	switch v := expr.(type) {
	case *ast.Ident:
		if v.Name == "nil" {
			return false
		}
		if v.Name == "retiredNode" {
			return true
		}
		return boolVars[v.Name]
	case *ast.CallExpr:
		return calledIdentName(v.Fun) == "NodeFieldsMatch"
	default:
		return false
	}
}
