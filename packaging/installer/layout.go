package installer

import (
	"os"
	"path/filepath"
	"strings"
)

// Layout is the on-node path set (AGENTS.md §4, §13.5). Tests relocate it
// with WithRoot. Production uses DefaultLayout.
type Layout struct {
	EtcDir        string
	VarDir        string
	IdentityDir   string
	StateDir      string
	UpdateJournal string
	StagingDir    string
	InstallDir    string
	RetiredDir    string
	OptDir        string
	ReleasesDir   string
	CurrentLink   string
	LockPath      string
	RunDir        string
	SystemdDir    string
	MachineID     string
	DBusMachineID string
	SSHDir        string
	NetbirdLib    string
	NetbirdEtc    string
	OSRelease     string
	ProcBootID    string
	AptList       string
	Keyring       string
	NetnsName     string
	NFTTable      string
}

// DefaultLayout is the v1 path set.
func DefaultLayout() Layout {
	return Layout{
		EtcDir:        "/etc/vpn-node",
		VarDir:        "/var/lib/vpn-node",
		IdentityDir:   "/var/lib/vpn-node/identity",
		StateDir:      "/var/lib/vpn-node/state",
		UpdateJournal: "/var/lib/vpn-node/update-journal",
		StagingDir:    "/var/lib/vpn-node/update-staging",
		InstallDir:    "/var/lib/vpn-node/install",
		RetiredDir:    "/var/lib/vpn-node/identity.retired",
		OptDir:        "/opt/vpn-node",
		ReleasesDir:   "/opt/vpn-node/releases",
		CurrentLink:   "/opt/vpn-node/current",
		LockPath:      "/var/lib/vpn-node/update.lock",
		RunDir:        "/run/vpn-node",
		SystemdDir:    "/etc/systemd/system",
		MachineID:     "/etc/machine-id",
		DBusMachineID: "/var/lib/dbus/machine-id",
		SSHDir:        "/etc/ssh",
		NetbirdLib:    "/var/lib/netbird",
		NetbirdEtc:    "/etc/netbird",
		OSRelease:     "/etc/os-release",
		ProcBootID:    "/proc/sys/kernel/random/boot_id",
		AptList:       "/etc/apt/sources.list.d/netbird.list",
		Keyring:       "/usr/share/keyrings/netbird-archive-keyring.gpg",
		NetnsName:     "fvpn",
		NFTTable:      "fvpn",
	}
}

// WithRoot prefixes every absolute path. The netns name and nft table
// stay as they are (they are not filesystem paths). An empty root returns l.
func (l Layout) WithRoot(root string) Layout {
	if root == "" || root == "/" {
		return l
	}
	root = filepath.Clean(root)
	l.EtcDir = joinRoot(root, l.EtcDir)
	l.VarDir = joinRoot(root, l.VarDir)
	l.IdentityDir = joinRoot(root, l.IdentityDir)
	l.StateDir = joinRoot(root, l.StateDir)
	l.UpdateJournal = joinRoot(root, l.UpdateJournal)
	l.StagingDir = joinRoot(root, l.StagingDir)
	l.InstallDir = joinRoot(root, l.InstallDir)
	l.RetiredDir = joinRoot(root, l.RetiredDir)
	l.OptDir = joinRoot(root, l.OptDir)
	l.ReleasesDir = joinRoot(root, l.ReleasesDir)
	l.CurrentLink = joinRoot(root, l.CurrentLink)
	l.LockPath = joinRoot(root, l.LockPath)
	l.RunDir = joinRoot(root, l.RunDir)
	l.SystemdDir = joinRoot(root, l.SystemdDir)
	l.MachineID = joinRoot(root, l.MachineID)
	l.DBusMachineID = joinRoot(root, l.DBusMachineID)
	l.SSHDir = joinRoot(root, l.SSHDir)
	l.NetbirdLib = joinRoot(root, l.NetbirdLib)
	l.NetbirdEtc = joinRoot(root, l.NetbirdEtc)
	l.OSRelease = joinRoot(root, l.OSRelease)
	l.ProcBootID = joinRoot(root, l.ProcBootID)
	l.AptList = joinRoot(root, l.AptList)
	l.Keyring = joinRoot(root, l.Keyring)
	return l
}

func joinRoot(root, path string) string {
	if path == "" {
		return ""
	}
	return filepath.Join(root, strings.TrimPrefix(path, "/"))
}

func (l Layout) releaseDir(id string) string {
	return filepath.Join(l.ReleasesDir, id)
}

func (l Layout) agentBin() string { return filepath.Join(l.CurrentLink, "fvpn-agent") }

func (l Layout) updaterBin() string { return filepath.Join(l.CurrentLink, "fvpn-updater") }

func (l Layout) unitSourceDir(releaseID string) string {
	return filepath.Join(l.releaseDir(releaseID), "lib", "systemd")
}

func (l Layout) installedFile() string {
	return filepath.Join(l.EtcDir, "installed.json")
}

func (l Layout) agentConfig() string { return filepath.Join(l.EtcDir, "agent.json") }

func (l Layout) updaterConfig() string {
	return filepath.Join(l.EtcDir, "updater.json")
}

func (l Layout) firstbootFile() string {
	return filepath.Join(l.EtcDir, "firstboot.json")
}

func (l Layout) netbirdFile() string { return filepath.Join(l.EtcDir, "netbird.json") }

func (l Layout) releaseTrust() string {
	return filepath.Join(l.EtcDir, "release-trust.json")
}

func (l Layout) serverCA() string { return filepath.Join(l.EtcDir, "ca.pem") }

func (l Layout) trustBundle() string { return filepath.Join(l.EtcDir, "trust.json") }

func (l Layout) ownedFile() string { return filepath.Join(l.InstallDir, "owned.json") }

func (l Layout) journalFile() string {
	return filepath.Join(l.InstallDir, "journal.json")
}

func (l Layout) pendingUpgrade() string {
	return filepath.Join(l.StagingDir, "pending-release.json")
}

func (l Layout) usageDB() string { return filepath.Join(l.StateDir, "usage.db") }

func (l Layout) nodeFile() string { return filepath.Join(l.IdentityDir, "node.json") }

func (l Layout) firstbootToken() string {
	return filepath.Join(l.VarDir, "firstboot.token")
}

func (l Layout) desiredTrust() string {
	return filepath.Join(l.EtcDir, "desired-trust.json")
}

func (l Layout) blobDir() string {
	return filepath.Join(l.InstallDir, "blobs")
}

func (l Layout) servicesFile() string {
	return filepath.Join(l.InstallDir, "services.json")
}

// owns reports whether path is inside a tree the installer is allowed to
// create or remove (§6.3). Rollback refuses every other path.
func (l Layout) owns(path string) bool {
	path = filepath.Clean(path)
	roots := []string{
		l.EtcDir, l.VarDir, l.OptDir, l.RunDir, l.SystemdDir,
		l.NetbirdLib, l.NetbirdEtc, l.SSHDir,
	}
	files := []string{l.AptList, l.Keyring, l.MachineID, l.DBusMachineID}
	for _, r := range roots {
		if r == "" {
			continue
		}
		r = filepath.Clean(r)
		if path == r || strings.HasPrefix(path, r+string(os.PathSeparator)) {
			return true
		}
	}
	for _, f := range files {
		if f != "" && path == filepath.Clean(f) {
			return true
		}
	}
	return false
}
