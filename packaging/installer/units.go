package installer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
)

// rewriteUnit replaces production prefixes so a relocated test layout can
// run the same unit text. Longer paths are replaced first.
func rewriteUnit(content string, l Layout) string {
	d := DefaultLayout()
	pairs := [][2]string{
		{d.IdentityDir, l.IdentityDir},
		{d.UpdateJournal, l.UpdateJournal},
		{d.StagingDir, l.StagingDir},
		{d.CurrentLink, l.CurrentLink},
		{d.ReleasesDir, l.ReleasesDir},
		{d.NetbirdRunDir, l.NetbirdRunDir},
		{d.NetbirdLib, l.NetbirdLib},
		{d.NetbirdEtc, l.NetbirdEtc},
		{d.EtcDir, l.EtcDir},
		{d.VarDir, l.VarDir},
		{d.RunDir, l.RunDir},
		{d.OptDir, l.OptDir},
		{d.MachineID, l.MachineID},
		{d.firstbootToken(), l.firstbootToken()},
	}
	sort.Slice(pairs, func(i, j int) bool { return len(pairs[i][0]) > len(pairs[j][0]) })
	for _, p := range pairs {
		if p[0] == "" || p[0] == p[1] {
			continue
		}
		content = strings.ReplaceAll(content, p[0], p[1])
	}
	return content
}

func installUnits(j *Journal, l Layout, releaseID string) error {
	srcDir := l.unitSourceDir(releaseID)
	for _, name := range requiredUnits {
		raw, err := os.ReadFile(filepath.Join(srcDir, name))
		if err != nil {
			return errf(ExitFailed, "failed", "release is missing unit %s", name)
		}
		body := rewriteUnit(string(raw), l)
		if err := j.WriteFile(filepath.Join(l.SystemdDir, name), []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func unitsMatch(l Layout, releaseID string) bool {
	srcDir := l.unitSourceDir(releaseID)
	for _, name := range requiredUnits {
		raw, err := os.ReadFile(filepath.Join(srcDir, name))
		if err != nil {
			return false
		}
		got, err := os.ReadFile(filepath.Join(l.SystemdDir, name))
		if err != nil {
			return false
		}
		if string(got) != rewriteUnit(string(raw), l) {
			return false
		}
	}
	return true
}

func dirModes(l Layout) []struct {
	path string
	mode os.FileMode
} {
	return []struct {
		path string
		mode os.FileMode
	}{
		{l.EtcDir, 0o755},
		{l.VarDir, 0o700},
		{l.IdentityDir, 0o700},
		{l.StateDir, 0o700},
		{l.UpdateJournal, 0o700},
		{l.StagingDir, 0o700},
		{l.InstallDir, 0o700},
		{l.OptDir, 0o755},
		{l.ReleasesDir, 0o755},
		{l.RunDir, 0o755},
	}
}

func modesMatch(l Layout) bool {
	for _, d := range dirModes(l) {
		st, err := os.Stat(d.path)
		if err != nil || !st.IsDir() || st.Mode().Perm() != d.mode {
			return false
		}
	}
	return true
}

func extractRelease(artifact, dest, arch string, v Verified, l Layout) error {
	if _, err := os.Stat(filepath.Join(dest, "fvpn-agent")); err == nil {
		return nil
	}
	f, err := os.Open(artifact)
	if err != nil {
		return err
	}
	defer f.Close()
	var max int64
	for _, a := range v.Manifest.Artifacts {
		if string(a.Architecture) == arch && string(a.OsId) == release.BaselineOSID {
			max = int64(a.InstalledSizeBytes.Uint64())
		}
	}
	if max <= 0 {
		return errf(ExitFailed, "failed", "manifest has no installed size")
	}
	err = updater.ExtractArchive(f, dest, updater.ExtractOptions{
		MaxBytes: max,
		Deny:     []string{l.IdentityDir, l.StateDir, l.UpdateJournal},
	})
	if err != nil {
		_ = os.RemoveAll(dest)
		return errf(ExitFailed, "failed", "extract failed")
	}
	for _, name := range []string{"fvpn-agent", "fvpn-updater", "fvpn-watchdog", "fvpn-node-install"} {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			_ = os.RemoveAll(dest)
			return errf(ExitFailed, "failed", "release is missing %s", name)
		}
	}
	for _, name := range requiredUnits {
		if _, err := os.Stat(filepath.Join(dest, "lib", "systemd", name)); err != nil {
			_ = os.RemoveAll(dest)
			return errf(ExitFailed, "failed", "release is missing unit %s", name)
		}
	}
	return nil
}
