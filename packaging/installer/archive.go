package installer

import (
	"os"
	"path/filepath"

	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
)

// Retain a verified archive for the updater's offline seed path. Copy through
// a private file and recheck before publishing; never read a whole archive
// into memory or replace a source file while it is being read.
func retainArchive(l Layout, j *Journal, src, arch string, v Verified) error {
	var sum string
	for _, a := range v.Manifest.Artifacts {
		if string(a.Architecture) == arch {
			sum = a.Sha256
		}
	}
	if sum == "" {
		return errf(ExitFailed, "failed", "archive architecture is missing")
	}
	dst := filepath.Join(l.StagingDir, sum+".tar.zst")
	if st, err := os.Lstat(dst); err == nil && st.Mode().IsRegular() && release.CheckArtifact(v.Manifest, arch, dst) == nil {
		return nil
	}
	write := func() error {
		tmp := dst + "." + fsutil.RandHex(8)
		defer os.Remove(tmp)
		if err := fsutil.CopyFile(src, tmp, 0o600); err != nil {
			return err
		}
		if err := release.CheckArtifact(v.Manifest, arch, tmp); err != nil {
			return errf(ExitFailed, "failed", "retained archive check failed")
		}
		if err := os.Rename(tmp, dst); err != nil {
			return err
		}
		return fsutil.SyncDir(l.StagingDir)
	}
	if j != nil {
		return j.replaceFile(dst, 0o600, write)
	}
	return write()
}
