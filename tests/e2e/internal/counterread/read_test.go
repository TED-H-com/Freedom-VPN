package counterread

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"testing"
)

func TestAT02_ReadFailureCategories(t *testing.T) {
	for _, v := range []struct{ stderr, want string }{{"Operation not permitted", "permission"}, {"Permission denied", "permission"}, {"Resource temporarily unavailable", "transient_io"}, {"Input/output error", "transient_io"}, {"No such file or directory", "nft_read"}} {
		if got := Category(t.Context(), errors.New("exit 1"), v.stderr); got != v.want {
			t.Fatalf("%s: %s", v.stderr, got)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if Category(ctx, errors.New("killed"), "") != "timeout" {
		t.Fatal("deadline lost")
	}
	if Category(t.Context(), nil, "") != "none" {
		t.Fatal("successful read misclassified")
	}
}

// Synthetic failures exercise the real enforcement parser and exact private
// retention. No retry replaces the failed observation (F13-01 / AT04 / SEC02).
func TestAT04_ParseFailureEvidence(t *testing.T) {
	for _, tc := range []struct{ name, raw, class, kind string }{
		{"empty", " \n", "empty", "empty_document"},
		{"truncated", `{"nftables":[`, "truncated", "unexpected_eof"},
		{"interleaved", "Warning: TEST ONLY diagnostic\n{\"nftables\":[]}\n", "interleaved", "diagnostic_and_json"},
		{"unexpected", `{"nftables":"TEST ONLY private value"}`, "unexpected_format", "json_type"},
		{"semantic", `{"nftables":[{"chain":{"family":"inet","table":"fvpn","name":"TEST ONLY private name","prio":"private-value"}}]}`, "unexpected_format", "parser_rejected"},
		{"syntax", `{"nftables":!]}`, "unexpected_format", "json_syntax"},
		{"duplicate_exact", `{"nftables":[{"chain":{"family":"inet","table":"fvpn","name":"TEST ONLY private name"}},{"chain":{"family":"inet","table":"fvpn","name":"TEST ONLY private name"}}]}`, "ambiguous_listing", "duplicate_chain"},
		{"duplicate_different", `{"nftables":[{"chain":{"family":"inet","table":"fvpn","name":"TEST ONLY private name","prio":0}},{"chain":{"family":"inet","table":"fvpn","name":"TEST ONLY private name","prio":1}}]}`, "ambiguous_listing", "duplicate_chain"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := enforce.ParseState([]byte(tc.raw))
			if err == nil {
				t.Fatal("fixture did not fail real parser")
			}
			dir := t.TempDir()
			calls := 0
			result := read(t.Context(), uuid.New(), dir, func() (Heartbeat, TimingSnapshot) { return Heartbeat{}, TimingSnapshot{} }, func(context.Context) ([]byte, error) { calls++; return []byte(tc.raw), nil })
			got := result.Diagnostic.Parse
			if calls != 1 || result.Diagnostic.Exit != 0 || result.Diagnostic.Stage != "nft_parse" || result.Diagnostic.Category != "parse" || result.Present || got == nil {
				t.Fatalf("failed sample lost: %+v", result)
			}
			if got.Class != tc.class || got.ParserErrorKind != tc.kind || !got.Retained || got.Bytes != len(tc.raw) || got.SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(tc.raw))) {
				t.Fatalf("metadata: %+v", got)
			}
			for name, want := range map[string]string{"output.raw": tc.raw, "parser-error.txt": err.Error()} {
				path := filepath.Join(dir, got.ID, name)
				raw, e := os.ReadFile(path)
				if e != nil || string(raw) != want {
					t.Fatal("private bytes changed")
				}
				info, e := os.Stat(path)
				if e != nil || info.Mode().Perm() != 0600 {
					t.Fatal("unsafe file mode")
				}
			}
			data, _ := json.Marshal(got)
			if bytes.Contains(data, []byte("private value")) || bytes.Contains(data, []byte("private name")) || bytes.Contains(data, []byte("private-value")) {
				t.Fatal("private parser content leaked")
			}
			again := retainParseFailure(dir, []byte(tc.raw), err)
			if again.ID == got.ID || !again.Retained {
				t.Fatal("samples not independent")
			}
		})
	}
}

func TestAT04_ParseRetentionFailureAndLines(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(dir, []byte("TEST ONLY"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		raw   string
		lines int
	}{{"", 0}, {"x", 1}, {"x\n", 1}, {"x\ny", 2}, {"\n\n", 2}} {
		got := retainParseFailure(dir, []byte(tc.raw), errors.New("TEST ONLY parser error"))
		if got.Retained || got.RetentionError == "" || got.Lines != tc.lines {
			t.Fatalf("metadata: %+v", got)
		}
	}
}

func TestAT04_WrappedDuplicateClassification(t *testing.T) {
	_, err := enforce.ParseState([]byte(`{"nftables":[{"chain":{"family":"inet","table":"fvpn","name":"forward"}},{"chain":{"family":"inet","table":"fvpn","name":"forward"}}]}`))
	class, kind := parseClass([]byte(`{}`), fmt.Errorf("TEST ONLY wrapper: %w", err))
	if class != "ambiguous_listing" || kind != "duplicate_chain" {
		t.Fatalf("%s %s", class, kind)
	}
}
