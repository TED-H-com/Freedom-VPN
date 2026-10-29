// Package fsutil holds the durable filesystem primitives the node updater
// relies on (§13.5): fsync of files and directories, atomic pointer switch by
// rename inside one directory, same-filesystem checks and free-space queries.
//
// A plain rename is not durable across power loss until the parent directory
// is synced; every helper here syncs what it changed. The exact flush order
// still has to be measured on the selected filesystem (§13.5, 待實測).
package fsutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"
)

// SyncDir fsyncs a directory so that entries created, renamed or removed in
// it survive a crash.
func SyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return fmt.Errorf("fsync dir %s: %w", filepath.Base(dir), err)
	}
	return nil
}

// SyncFile fsyncs a file by path.
func SyncFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// WriteFileAtomic writes data to path through a temp file in the same
// directory, fsyncs the file, renames it into place and fsyncs the directory.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = tmp.Close()
			_ = os.Remove(name)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	ok = true
	return SyncDir(dir)
}

// SwitchSymlink atomically points link at target (§13.5, §13.6 step 10). A
// new symlink is created next to link and renamed over it, which is atomic
// within one directory and therefore one filesystem; the parent directory is
// then fsynced. target is stored verbatim (use a relative target so the tree
// can be relocated).
func SwitchSymlink(link, target string) error {
	dir := filepath.Dir(link)
	tmp := filepath.Join(dir, "."+filepath.Base(link)+".next-"+randHex(6))
	if err := os.Symlink(target, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return SyncDir(dir)
}

// ReadSymlink returns the target of link, or "" when link does not exist.
func ReadSymlink(link string) (string, error) {
	t, err := os.Readlink(link)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return t, err
}

// SameFilesystem reports whether a and b live on the same device. Atomic
// rename is only promised within one filesystem (§13.5).
func SameFilesystem(a, b string) (bool, error) {
	da, err := device(a)
	if err != nil {
		return false, err
	}
	db, err := device(b)
	if err != nil {
		return false, err
	}
	return da == db, nil
}

// CopyFile copies src to dst (created exclusively) and fsyncs dst.
func CopyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// IsNoSpace reports whether err is a disk-full or quota error.
func IsNoSpace(err error) bool {
	return errors.Is(err, syscall.ENOSPC) || errors.Is(err, syscall.EDQUOT)
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand never fails on supported platforms
	}
	return hex.EncodeToString(b)
}

// TreeHash is the SHA-256 of a sorted inventory of root. The digest covers
// relative paths, permission bits and file bytes, so a rollback can prove
// the tree still matches the release that was installed (§13.5, §13.9).
// A symlink is an error. Release trees contain regular files and directories
// only (D-034a). The current pointer is a symlink outside the release tree.
func TreeHash(root string) (string, error) {
	var lines []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		perm := info.Mode().Perm()
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("fsutil: %s is a symlink", rel)
		case info.IsDir():
			lines = append(lines, fmt.Sprintf("d %04o %s", perm, rel))
		case info.Mode().IsRegular():
			sum, err := hashFile(path)
			if err != nil {
				return err
			}
			lines = append(lines, fmt.Sprintf("f %04o %s %s", perm, rel, sum))
		default:
			return fmt.Errorf("fsutil: %s has unsupported file type", rel)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(lines)
	h := sha256.New()
	for _, ln := range lines {
		_, _ = io.WriteString(h, ln)
		_, _ = h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SyncTree fsyncs every regular file and every directory under root, files
// first and directories after their children, so a tree is durable before
// the current pointer moves (§13.5, §13.6 step 10).
func SyncTree(root string) error {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if info.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		if info.Mode().IsRegular() {
			return SyncFile(path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := SyncDir(dirs[i]); err != nil {
			return err
		}
	}
	return nil
}

// RandHex returns n random bytes in hex, for unique temp names.
func RandHex(n int) string { return randHex(n) }
