package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

// CheckReport is the read-only view check prints. The usage store is a
// presence bit only; its contents are never read (§6.3).
type CheckReport struct {
	Status         string            `json:"status"`
	Baseline       *BaselineResult   `json:"baseline"`
	PendingRelease *pendingRelease   `json:"pending_release,omitempty"`
	OSID           string            `json:"os_id"`
	OSVersion      string            `json:"os_version"`
	Arch           string            `json:"arch"`
	Kernel         string            `json:"kernel"`
	ReleaseID      string            `json:"release_id,omitempty"`
	ManifestSHA256 string            `json:"manifest_sha256,omitempty"`
	ReleaseSource  string            `json:"release_source,omitempty"`
	InstallRecord  *installedState   `json:"install_record,omitempty"`
	Updater        *UpdaterStatus    `json:"updater,omitempty"`
	Units          map[string]string `json:"units"`
	Directories    []DirStatus       `json:"directories"`
	Lock           LockStatus        `json:"lock"`
	Identity       IdentityStatus    `json:"identity"`
	UsageStore     bool              `json:"usage_store_present"`
	Netbird        NetbirdStatus     `json:"netbird"`
	Marker         MarkerStatus      `json:"installer_marker"`
	Drifts         []string          `json:"drifts"`
}

// UpdaterStatus is the read-only journal snapshot's relevant subset. A failed
// attempt need not change current; recovery_required is always drift.
type UpdaterStatus struct {
	CurrentReleaseID string `json:"current_release_id"`
	State            string `json:"state"`
	RecoveryRequired bool   `json:"recovery_required"`
	SeededBaseline   *struct {
		ReleaseID      string `json:"release_id"`
		ManifestSHA256 string `json:"manifest_sha256"`
	} `json:"seeded_baseline,omitempty"`
}

// DirStatus is one directory mode check.
type DirStatus struct {
	Path     string `json:"path"`
	Mode     string `json:"mode,omitempty"`
	Expected string `json:"expected"`
	OK       bool   `json:"ok"`
}

// LockStatus reports the holder file. check does not take the flock.
type LockStatus struct {
	Held      bool   `json:"held"`
	PID       int    `json:"pid,omitempty"`
	BootID    string `json:"boot_id,omitempty"`
	StartTime int64  `json:"start_time,omitempty"`
}

// IdentityStatus names the incarnation when identity exists. It has no key material.
type IdentityStatus struct {
	Present       bool   `json:"present"`
	IncarnationID string `json:"incarnation_id,omitempty"`
	NodeID        string `json:"node_id,omitempty"`
}

// NetbirdStatus is the client version and the daemon's reported state.
// A management disconnect is reported and is not, by itself, drift.
type NetbirdStatus struct {
	Version string `json:"version,omitempty"`
	Daemon  string `json:"daemon"`
}

// Check reads the host and returns exit 0 when healthy and ExitDrifted
// when the layout does not match. It does not change the host.
func Check(ctx context.Context, env *Env) (*CheckReport, error) {
	if env == nil {
		env = &Env{}
	}
	l := env.layout()
	p, err := Identify(env)
	rep := &CheckReport{
		Status: statusDrifted,
		OSID:   p.OSID, OSVersion: p.OSVersion, Arch: p.Arch, Kernel: p.Kernel,
		Units: map[string]string{}, Drifts: []string{},
	}
	if err != nil {
		rep.Drifts = append(rep.Drifts, "platform")
	}
	inst, has := readInstalled(l)
	rep.Marker, err = readMarker(ctx, env)
	// A hookless marker is ephemeral and can be absent after reboot. Its
	// absence is informational; a legacy or malformed marker is drift.
	if err != nil || rep.Marker.LegacyOwner {
		rep.Drifts = append(rep.Drifts, "installer_marker")
	}
	rep.Baseline = readBaseline(l)
	if has {
		rep.InstallRecord = &inst
		rep.ReleaseSource = "installed.json"
		rep.ReleaseID = inst.ReleaseID
		rep.ManifestSHA256 = inst.ManifestSHA256
	} else {
		rep.Drifts = append(rep.Drifts, "not_installed")
	}
	if st, err := readUpdaterStatus(ctx, env, rep.Baseline.Status == "seeded"); err != nil {
		rep.Drifts = append(rep.Drifts, "updater_journal")
	} else if st != nil {
		rep.Updater = st
		if st.RecoveryRequired {
			rep.Drifts = append(rep.Drifts, "updater_recovery_required")
		}
		if st.CurrentReleaseID != "" {
			rep.ReleaseSource = "updater_journal"
			rep.ReleaseID = st.CurrentReleaseID
			if rep.ReleaseID != inst.ReleaseID {
				// status does not expose every release's manifest digest. Keep
				// the old digest solely in install_record, never label it current.
				rep.ManifestSHA256 = ""
			}
			if seed := st.SeededBaseline; seed != nil && seed.ReleaseID == rep.ReleaseID {
				rep.ManifestSHA256 = seed.ManifestSHA256
			}
		} else if rep.Baseline.Status == "seeded" {
			rep.Drifts = append(rep.Drifts, "updater_journal")
		}
	}
	for _, d := range dirModes(l) {
		st, sterr := os.Stat(d.path)
		ds := DirStatus{Path: d.path, Expected: formatMode(d.mode)}
		if sterr != nil || !st.IsDir() {
			ds.OK = false
			rep.Drifts = append(rep.Drifts, "directory "+d.path)
		} else {
			ds.Mode = formatMode(st.Mode().Perm())
			ds.OK = st.Mode().Perm() == d.mode
			if !ds.OK {
				rep.Drifts = append(rep.Drifts, "mode "+d.path)
			}
		}
		rep.Directories = append(rep.Directories, ds)
	}
	if has {
		if !unitsMatch(l, rep.ReleaseID) {
			rep.Drifts = append(rep.Drifts, "units")
		}
		target, err := filepath.EvalSymlinks(l.CurrentLink)
		if err != nil || target != l.releaseDir(rep.ReleaseID) {
			rep.Drifts = append(rep.Drifts, "current")
		}
		for _, unit := range expectActive {
			state, _ := env.services().Active(ctx, unit)
			rep.Units[unit] = state
			if state != "active" && state != "activating" {
				rep.Drifts = append(rep.Drifts, "unit "+unit+" "+state)
			}
		}
		up, _ := env.services().Active(ctx, UnitUpdater)
		rep.Units[UnitUpdater] = up
		if up == "failed" {
			rep.Drifts = append(rep.Drifts, "unit "+UnitUpdater+" failed")
		}
		if !identityPresent(l) {
			rep.Drifts = append(rep.Drifts, "identity")
		}
	}
	if rep.Baseline.Status == "failed" || rep.Baseline.Status == "unreadable" {
		rep.Drifts = append(rep.Drifts, "baseline")
	}
	if raw, err := os.ReadFile(l.pendingUpgrade()); err == nil {
		var pending pendingRelease
		if json.Unmarshal(raw, &pending) != nil {
			rep.Drifts = append(rep.Drifts, "pending_release")
		} else {
			rep.PendingRelease = &pending
		}
	} else if !os.IsNotExist(err) {
		rep.Drifts = append(rep.Drifts, "pending_release")
	}
	rep.Lock = readLock(l)
	if n, nerr := readNode(l); nerr == nil && n.NodeIncarnationID != "" {
		rep.Identity = IdentityStatus{Present: true, IncarnationID: n.NodeIncarnationID, NodeID: n.NodeID}
	}
	if st, sterr := os.Stat(l.usageDB()); sterr == nil && st.Mode().IsRegular() {
		rep.UsageStore = true
	}
	rep.Netbird = readNetbird(ctx, env)
	if rep.Netbird.Version == "" {
		rep.Drifts = append(rep.Drifts, "netbird")
	} else if !netbirdVersionOK(rep.Netbird.Version) {
		rep.Drifts = append(rep.Drifts, "netbird_version")
	}
	if rep.Netbird.Daemon == "inactive" || rep.Netbird.Daemon == "missing" {
		if has {
			rep.Drifts = append(rep.Drifts, "netbird_daemon")
		}
	}
	if len(rep.Drifts) == 0 {
		rep.Status = statusHealthy
		return rep, nil
	}
	return rep, errf(ExitDrifted, "drifted", "%s", strings.Join(rep.Drifts, ", "))
}

func readUpdaterStatus(ctx context.Context, env *Env, seeded bool) (*UpdaterStatus, error) {
	path := filepath.Join(env.layout().UpdateJournal, "journal.db")
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) && !seeded {
			return nil, nil // Older updater without a seeded baseline.
		}
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	raw, err := env.output(ctx, env.layout().updaterBin(), "status", "--journal", path)
	if err != nil {
		return nil, err
	}
	var st UpdaterStatus
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, err
	}
	if st.CurrentReleaseID != "" {
		id, err := uuid.Parse(st.CurrentReleaseID)
		if err != nil || id == uuid.Nil || id.String() != st.CurrentReleaseID {
			return nil, fmt.Errorf("invalid current release in updater status")
		}
	}
	return &st, nil
}

func formatMode(m os.FileMode) string {
	return fmt.Sprintf("%04o", m.Perm())
}

func readLock(l Layout) LockStatus {
	raw, err := os.ReadFile(l.LockPath)
	if err != nil || len(raw) == 0 {
		return LockStatus{}
	}
	var h struct {
		PID       int    `json:"pid"`
		BootID    string `json:"boot_id"`
		StartTime int64  `json:"start_time"`
	}
	if json.Unmarshal(raw, &h) != nil || h.PID <= 0 {
		return LockStatus{}
	}
	alive := false
	err = unix.Kill(h.PID, 0)
	if err == nil || err == unix.EPERM {
		alive = true
	}
	return LockStatus{Held: alive, PID: h.PID, BootID: h.BootID, StartTime: h.StartTime}
}

func readNetbird(ctx context.Context, env *Env) NetbirdStatus {
	if _, err := env.look("netbird"); err != nil {
		return NetbirdStatus{Daemon: "missing"}
	}
	out, err := env.output(ctx, "netbird", "version")
	st := NetbirdStatus{Daemon: "unknown"}
	if err == nil {
		st.Version = strings.TrimSpace(string(out))
	}
	status, serr := env.output(ctx, "netbird", "status", "--daemon-addr", env.layout().netbirdAddr())
	if serr != nil {
		state, _ := env.services().Active(ctx, UnitNetbird)
		st.Daemon = state
		return st
	}
	text := string(status)
	switch {
	case strings.Contains(text, "Management: Connected"):
		st.Daemon = "connected"
	case strings.Contains(text, "Daemon status: Connected"), strings.Contains(text, "Management: Disconnected"):
		st.Daemon = "disconnected"
	default:
		state, _ := env.services().Active(ctx, UnitNetbird)
		if state != "" {
			st.Daemon = state
		}
	}
	return st
}

// Text renders a check report without a stack and without store contents.
func (r *CheckReport) Text() string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "status: %s\n", r.Status)
	fmt.Fprintf(&b, "os: %s %s %s\n", r.OSID, r.OSVersion, r.Arch)
	fmt.Fprintf(&b, "kernel: %s\n", r.Kernel)
	if r.ReleaseID != "" {
		fmt.Fprintf(&b, "release_id: %s\n", r.ReleaseID)
		fmt.Fprintf(&b, "release_source: %s\n", r.ReleaseSource)
		if r.ManifestSHA256 != "" {
			fmt.Fprintf(&b, "manifest_sha256: %s\n", r.ManifestSHA256)
		}
	}
	if r.InstallRecord != nil {
		fmt.Fprintf(&b, "install_record: release=%s manifest_sha256=%s\n", r.InstallRecord.ReleaseID, r.InstallRecord.ManifestSHA256)
	}
	if r.Baseline != nil {
		fmt.Fprintf(&b, "baseline: %s\n", r.Baseline.Status)
	}
	if r.PendingRelease != nil {
		fmt.Fprintf(&b, "pending_manifest_path: %s\npending_request_id: %s\n", r.PendingRelease.ManifestPath, r.PendingRelease.RequestID)
	}
	fmt.Fprintf(&b, "identity: present=%t incarnation=%s\n", r.Identity.Present, r.Identity.IncarnationID)
	fmt.Fprintf(&b, "usage_store_present: %t\n", r.UsageStore)
	fmt.Fprintf(&b, "netbird: version=%s daemon=%s\n", r.Netbird.Version, r.Netbird.Daemon)
	fmt.Fprintf(&b, "installer_marker: table=inet %s present=%t legacy_owner=%t\n", r.Marker.Table, r.Marker.Present, r.Marker.LegacyOwner)
	if r.Updater != nil {
		fmt.Fprintf(&b, "updater: state=%s recovery_required=%t\n", r.Updater.State, r.Updater.RecoveryRequired)
	}
	fmt.Fprintf(&b, "lock_held: %t\n", r.Lock.Held)
	for _, u := range []string{UnitWatchdog, UnitAgent, UnitNetbird, UnitRecover, UnitTimer, UnitNetbirdUp, UnitUpdater} {
		if s, ok := r.Units[u]; ok {
			fmt.Fprintf(&b, "unit %s: %s\n", u, s)
		}
	}
	if len(r.Drifts) == 0 {
		b.WriteString("drifts: none\n")
	} else {
		fmt.Fprintf(&b, "drifts: %s\n", strings.Join(r.Drifts, ", "))
	}
	return b.String()
}
