package release

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTree(t *testing.T, root, name string) string {
	t.Helper()
	dir := filepath.Join(root, "tree-"+name)
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("component " + name + "\n")
	if err := os.WriteFile(filepath.Join(dir, "bin", "fvpn-agent"), body, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A later mtime must not affect the archive.
	past := time.Date(1999, 1, 2, 3, 4, 5, 0, time.UTC)
	_ = os.Chtimes(filepath.Join(dir, "README"), past, past)
	return dir
}

func TestPackageReproducible(t *testing.T) {
	root := t.TempDir()
	tree := writeTree(t, root, "one")
	a := filepath.Join(root, "a.tar.zst")
	b := filepath.Join(root, "b.tar.zst")
	when := time.Unix(1_700_000_000, 0).UTC()
	if err := PackageDir(tree, a, when); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour)
	_ = os.Chtimes(filepath.Join(tree, "bin", "fvpn-agent"), future, future)
	if err := PackageDir(tree, b, when); err != nil {
		t.Fatal(err)
	}
	ab, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	bb, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ab, bb) {
		t.Fatalf("archives differ: %d vs %d bytes", len(ab), len(bb))
	}
	headers, err := ReadArchive(a)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, h := range headers {
		names = append(names, h.Name)
		if h.Uid != 0 || h.Gid != 0 || h.Uname != "root" || h.Gname != "root" {
			t.Fatalf("owner of %s: uid %d gid %d %s %s", h.Name, h.Uid, h.Gid, h.Uname, h.Gname)
		}
		if !h.ModTime.Equal(when) {
			t.Fatalf("mtime of %s = %s", h.Name, h.ModTime)
		}
	}
	if len(names) < 2 || names[0] > names[1] {
		t.Fatalf("entries not sorted: %v", names)
	}
}

func TestPackageRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	tree := writeTree(t, root, "link")
	if err := os.Symlink("fvpn-agent", filepath.Join(tree, "bin", "alias")); err != nil {
		t.Fatal(err)
	}
	err := PackageDir(tree, filepath.Join(root, "out.tar.zst"), time.Unix(0, 0))
	if err == nil {
		t.Fatal("symlink was packaged")
	}
}
