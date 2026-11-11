package handover_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

// Keep complete scope integrity in the Go workflow, which runs on every PR.
func TestHandover_ScopeReviewEveryPin(t *testing.T) {
	root := repoRoot(t)
	var scope any
	if err := json.Unmarshal([]byte(readOwnedDoc(t, root, "docs/e2e/at/secops/scope-review.json")), &scope); err != nil {
		t.Fatalf("decode scope-review.json: %v", err)
	}

	count := 0
	var walk func(any)
	walk = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			rawPath, hasPath := value["path"]
			rawHash, hasHash := value["sha256"]
			if hasPath && hasHash {
				count++
				path, pathOK := rawPath.(string)
				want, hashOK := rawHash.(string)
				if !pathOK || !hashOK || path == "" || want == "" {
					t.Fatal("scope-review.json pin path and sha256 must be nonempty strings")
				}
				sum := sha256.Sum256([]byte(readOwnedDoc(t, root, path)))
				if got := hex.EncodeToString(sum[:]); got != want {
					t.Errorf("%s SHA-256 mismatch: got %s, want %s", path, got, want)
				}
			}
			for _, child := range value {
				walk(child)
			}
		case []any:
			for _, child := range value {
				walk(child)
			}
		}
	}
	walk(scope)
	if count == 0 {
		t.Fatal("scope-review.json has no path/sha256 pairs")
	}
	t.Logf("Scope review pins: %d", count)
}
