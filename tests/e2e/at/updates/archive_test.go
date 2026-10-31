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
	checkArchivePins(t, "follow-up-9", 65)
}

func TestD132_Followup10ArchivePins(t *testing.T) {
	checkArchivePins(t, "follow-up-10", 39)
}

func TestD132_Followup11ArchivePins(t *testing.T) {
	checkArchivePins(t, "follow-up-11", 43)
}

func TestD098_Followup12FirstAttemptArchivePins(t *testing.T) {
	checkArchivePins(t, "follow-up-12-first-attempt", 17)
}

func checkArchivePins(t *testing.T, followup string, count int) {
	t.Helper()
	root := filepath.Join("..", "..", "..", "..", "docs/e2e/at/updates/history", followup)
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
	if len(rows) != count {
		t.Fatalf("archive inventory changed: %d", len(rows))
	}
}
