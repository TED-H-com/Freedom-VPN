package updater

import (
	"os"
	"path/filepath"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
)

// Layout is the node directory split (§13.5, AGENTS.md §4). Program files,
// identity and durable state never share a directory. Tests point every
// field at a temp directory.
type Layout struct {
	Releases string `json:"releases"`
	Current  string `json:"current"`
	Identity string `json:"identity"`
	State    string `json:"state"`
	Journal  string `json:"journal"`
	Lock     string `json:"lock"`
	Staging  string `json:"staging"`
	Trust    string `json:"trust"`
}

// DefaultLayout is the v1 path set. Every field may be overridden.
func DefaultLayout() Layout {
	return Layout{
		Releases: "/opt/vpn-node/releases",
		Current:  "/opt/vpn-node/current",
		Identity: "/var/lib/vpn-node/identity",
		State:    "/var/lib/vpn-node/state",
		Journal:  "/var/lib/vpn-node/update-journal",
		Lock:     "/var/lib/vpn-node/update.lock",
		Staging:  "/var/lib/vpn-node/update-staging",
		Trust:    "/etc/vpn-node/release-trust.json",
	}
}

// Ensure creates the directories. Identity, state, journal and staging are
// mode 0700; the releases directory is 0755.
func (l Layout) Ensure() error {
	// Validate before creating any child paths under a pre-existing journal.
	if err := healthattest.CheckDir(l.Journal); err != nil && !os.IsNotExist(err) {
		return err
	}
	dirs := []struct {
		path string
		mode os.FileMode
	}{
		{l.Releases, 0o755},
		{filepath.Dir(l.Current), 0o755},
		{l.Identity, 0o700},
		{l.State, 0o700},
		{l.Journal, 0o700},
		{l.Staging, 0o700},
		{filepath.Dir(l.Lock), 0o700},
		{filepath.Dir(l.Trust), 0o755},
	}
	seen := map[string]bool{}
	for _, d := range dirs {
		if d.path == "" || d.path == "." || seen[d.path] {
			continue
		}
		seen[d.path] = true
		if err := os.MkdirAll(d.path, d.mode); err != nil {
			return err
		}
	}
	return healthattest.CheckDir(l.Journal)
}

func (l Layout) releaseDir(id string) string {
	return filepath.Join(l.Releases, id)
}

// PersistentDirs are the trees an archive must never write (§13.5, §13.7).
func (l Layout) PersistentDirs() []string {
	return []string{l.Identity, l.State, l.Journal}
}

// currentID is the release id the current symlink points at, or "".
func (l Layout) currentID() (string, error) {
	t, err := fsutil.ReadSymlink(l.Current)
	if err != nil || t == "" {
		return "", err
	}
	if !filepath.IsAbs(t) {
		t = filepath.Join(filepath.Dir(l.Current), t)
	}
	return filepath.Base(filepath.Clean(t)), nil
}

// relativeRelease is the symlink target for id, relative to current's parent,
// so the pointer stays valid if the tree is relocated (§13.5).
func (l Layout) relativeRelease(id string) (string, error) {
	return filepath.Rel(filepath.Dir(l.Current), l.releaseDir(id))
}
