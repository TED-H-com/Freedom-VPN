package release

import (
	"os"
	"path/filepath"
)

// WriteNewTrustUpdate publishes checked public data as a new mode-0644 file.
// A hard link is the atomic no-replace commit: unlike Rename it also refuses
// dangling symlinks and races creating the final name. All fallible content
// writes, chmod, sync and close happen before the final name becomes visible.
func WriteNewTrustUpdate(path string, canonical []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".release-trust-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err := f.Write(append(append([]byte(nil), canonical...), '\n')); err != nil {
		return err
	}
	if err := f.Chmod(0o644); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Link(f.Name(), path)
}
