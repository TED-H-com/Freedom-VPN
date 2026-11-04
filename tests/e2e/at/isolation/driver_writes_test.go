package isolation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// Inspect every HTTP PATCH/PUT call, including new sites. Only the existing
// fleet-config method is exempt, with its complete body pinned to the baseline.
func nodeWriteBypasses(source []byte) (int, []string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "driver.go.txt", source, 0)
	if err != nil {
		return 0, nil, err
	}
	writes := 0
	var bypasses []string
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == "putFleet" {
			digest := sha256.Sum256(source[fn.Pos()-1 : fn.End()-1])
			if hex.EncodeToString(digest[:]) != "ad0402cec8106d536bee22e9769e13fc9c31e2e17e1e961987da7ca7268beb7b" {
				bypasses = append(bypasses, "fleet exception changed; review HTTP audit")
			}
		}
		ast.Inspect(fn, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			method, ok := call.Args[1].(*ast.BasicLit)
			if !ok {
				return true
			}
			verb, _ := strconv.Unquote(method.Value)
			if verb != "PATCH" && verb != "PUT" {
				return true
			}
			if sel.Sel.Name == "writeNode" {
				writes++
			} else if fn.Name.Name != "putFleet" {
				bypasses = append(bypasses, fn.Name.Name+":"+verb)
			}
			return true
		})
	}
	return writes, bypasses, nil
}

func TestF19_02_AllDriverNodeWritesUseSharedHelper(t *testing.T) {
	source, err := os.ReadFile("driver.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	writes, bypasses, err := nodeWriteBypasses(source)
	if err != nil || writes != 6 || len(bypasses) != 0 {
		t.Fatalf("writes=%d bypasses=%v err=%v", writes, bypasses, err)
	}
	// Prove the audit catches a new bypass rather than just counting helpers.
	mutated := append(append([]byte(nil), source...), []byte(`
func escapedCapacity() { admin.call(ctx, "PATCH", nodePath, capacity, headers) }
func escapedPublication() { admin.call(ctx, "PUT", nodePath+"/publication", body, headers) }
`)...)
	_, bypasses, err = nodeWriteBypasses(mutated)
	if err != nil || len(bypasses) != 2 {
		t.Fatalf("missed bypass: %v %v", bypasses, err)
	}
}

func TestAPI01_AssistNodeWriteUsesSharedHelper(t *testing.T) {
	source, err := os.ReadFile("assist.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	writes, bypasses, err := nodeWriteBypasses(source)
	if err != nil || writes != 1 || len(bypasses) != 0 {
		t.Fatalf("writes=%d bypasses=%v err=%v", writes, bypasses, err)
	}
}

func TestF20_EarlyAbortCloneClosure(t *testing.T) {
	raw, err := os.ReadFile("testdata/early-abort.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Clone bool `json:"clone_creation_reached"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	row := UncreatedCloneClosure("fvpn-w4-isolation-node-clone", fixture.Clone)
	if row["status"] != "not created" || row["ok"] != false || row["stage"] != "clone_creation" || row["reason"] != "not created: setup exited before clone creation" {
		t.Fatal(row)
	}
	if UncreatedCloneClosure("fvpn-w4-isolation-node-clone", true) != nil {
		t.Fatal("created clone must use actual collection")
	}
	// Check the live loop consumes the stage guard before any install/exec.
	source, _ := os.ReadFile("driver.go.txt")
	closure := strings.Split(strings.Split(string(source), "func (x *isolationRun) archiveReceiver()")[1], "x.extra(\"receiver-closure.json\"")[0]
	if strings.Index(closure, "iso.UncreatedCloneClosure") >= strings.Index(closure, "x.install(container)") || !strings.Contains(closure, "closure = append(closure, absent)\n\t\t\tcontinue") {
		t.Fatal("uncreated clone collection is not bypassed")
	}
}

func TestF20_EarlyAbortReviewers(t *testing.T) {
	out, err := exec.Command("python3", "-B", "early_abort_test.py").CombinedOutput()
	if err != nil {
		t.Fatalf("reviewers: %v\n%s", err, out)
	}
	t.Log(string(out))
}

func TestF19_02_SuccessfulSetupKeepsPublicationReceipt(t *testing.T) {
	record := &Record{}
	check := &Check{Name: "serving setup"}
	note := PublicationNote(check, record)
	const trace = `{"outcome":"accepted","attempts":[{"status":412},{"status":202}]}`
	note("node_write", trace)
	note("publication_put", "202")
	if len(check.Observations) != 2 || len(record.Observations) != 1 || record.Observations[0].Name != "node_write" || record.Observations[0].Detail != trace {
		t.Fatal("setup receipt must survive without adding an acceptance check")
	}
	if len(record.Checks) != 0 {
		t.Fatal("receipt retention must not add or promote a check")
	}
}
