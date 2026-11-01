//go:build e2e

package e2e

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread"
	"github.com/google/uuid"
)

// The synthetic docker executable stands in for the container boundary; it
// returns a failed read, then the two private files. No nft or network mutation.
func TestAT04_CounterFailureExport(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			root := t.TempDir()
			bin := filepath.Join(root, "docker")
			raw := []byte("TEST ONLY private raw\n")
			parser := []byte("TEST ONLY exact parser error")
			id := uuid.NewString()
			result := counterread.Result{Diagnostic: counterread.Diagnostic{Stage: "nft_parse", Category: "parse", Parse: &counterread.ParseFailure{ID: id, Class: "unexpected_format", Retained: true, Bytes: len(raw), SHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), Lines: 1}}}
			data, _ := json.Marshal(result)
			for name, b := range map[string][]byte{"response": data, "output.raw": raw, "parser-error.txt": parser} {
				if e := os.WriteFile(filepath.Join(root, name), b, 0600); e != nil {
					t.Fatal(e)
				}
			}
			script := `#!/bin/sh
case "$3" in
 /usr/local/bin/fvpn-e2e-nodeconfig) cat "$FVPN_TEST_COUNTER_EXPORT/response" ;;
 cat) test "$FVPN_TEST_COUNTER_EXPORT_FAIL" != 1 || exit 1
      case "$4" in
       */output.raw) cat "$FVPN_TEST_COUNTER_EXPORT/output.raw" ;;
       */parser-error.txt) cat "$FVPN_TEST_COUNTER_EXPORT/parser-error.txt" ;;
       *) exit 2 ;;
      esac ;;
 *) exit 3 ;;
esac
`
			if e := os.WriteFile(bin, []byte(script), 0700); e != nil {
				t.Fatal(e)
			}
			t.Setenv("PATH", root+":"+os.Getenv("PATH"))
			t.Setenv("FVPN_TEST_COUNTER_EXPORT", root)
			if fail {
				t.Setenv("FVPN_TEST_COUNTER_EXPORT_FAIL", "1")
			} else {
				t.Setenv("FVPN_TEST_COUNTER_EXPORT_FAIL", "0")
			}
			s := &Stack{lab: &lab{root: root, node: "fvpn-e2e-node"}}
			got, err := s.DeviceByteCounters(t.Context(), uuid.New())
			if err == nil || got.Diagnostic.Category != "parse" || got.Diagnostic.Exit != 0 || got.Diagnostic.Parse == nil {
				t.Fatalf("failed read changed: %+v %v", got, err)
			}
			if got.Diagnostic.Parse.Exported == fail {
				t.Fatalf("export result: %+v", got.Diagnostic.Parse)
			}
			if fail {
				if got.Diagnostic.Parse.ExportError == "" {
					t.Fatal("missing export error")
				}
				return
			}
			for name, want := range map[string][]byte{"output.raw": raw, "parser-error.txt": parser} {
				path := filepath.Join(root, ".agent/tmp/counterread", id, name)
				b, e := os.ReadFile(path)
				if e != nil || string(b) != string(want) {
					t.Fatal("private evidence lost")
				}
				info, e := os.Stat(path)
				if e != nil || info.Mode().Perm() != 0600 {
					t.Fatal("private evidence mode")
				}
			}
		})
	}
}
