package fleetupdate

import (
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
)

func TestD132_EvidenceReferencesUseImmutableHistory(t *testing.T) {
	// Scan all package files, including fixtures and comments, without an
	// integration tag or database. Split the prefix to avoid matching this rule.
	references := regexp.MustCompile("docs/" + "e2e/[^\\s\"'`<>)]*")
	err := fs.WalkDir(os.DirFS("."), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		for line, text := range strings.Split(string(raw), "\n") {
			for _, reference := range references.FindAllString(text, -1) {
				if !strings.Contains(path.Clean(reference), "/history/") {
					t.Errorf("%s:%d: mutable AT evidence reference %q; own a fixture or cite an immutable history archive", name, line+1, reference)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
