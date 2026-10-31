package updates

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestD098_Followup9ArchivePins(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "docs/e2e/at/updates/history/follow-up-9")
	raw, err := os.ReadFile(filepath.Join(root, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSpace(string(raw)), "\n")
	for _, row := range rows {
		fields := strings.SplitN(row, "  ", 2)
		if len(fields) != 2 || !filepath.IsLocal(fields[1]) {
			t.Fatalf("invalid pin %q", row)
		}
		b, err := os.ReadFile(filepath.Join(root, fields[1]))
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(b)); got != fields[0] {
			t.Fatalf("archive drift: %s", fields[1])
		}
	}
	if len(rows) != 65 {
		t.Fatalf("archive inventory changed: %d", len(rows))
	}
}
