package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// CheckReport is the read-only view check prints. The usage store is a
// presence bit only; its contents are never read (§6.3).
type CheckReport struct {
	Status         string            `json:"status"`
	OSID           string            `json:"os_id"`
	OSVersion      string            `json:"os_version"`
	Arch           string            `json:"arch"`
	Kernel         string            `json:"kernel"`
	ReleaseID      string            `json:"release_id,omitempty"`
	ManifestSHA256 string            `json:"manifest_sha256,omitempty"`
	Units          map[string]string `json:"units"`
	Directories    []DirStatus       `json:"directories"`
	Lock           LockStatus        `json:"lock"`
	Identity       IdentityStatus    `json:"identity"`
	UsageStore     bool              `json:"usage_store_present"`
	Netbird        NetbirdStatus     `json:"netbird"`
	Drifts         []string          `json:"drifts"`
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
	if has {
		rep.ReleaseID = inst.ReleaseID
		rep.ManifestSHA256 = inst.ManifestSHA256
	} else {
		rep.Drifts = append(rep.Drifts, "not_installed")
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
		if !unitsMatch(l, inst.ReleaseID) {
			rep.Drifts = append(rep.Drifts, "units")
		}
		target, _ := os.Readlink(l.CurrentLink)
		if target != "releases/"+inst.ReleaseID {
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
	status, serr := env.output(ctx, "netbird", "status")
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
		fmt.Fprintf(&b, "manifest_sha256: %s\n", r.ManifestSHA256)
	}
	fmt.Fprintf(&b, "identity: present=%t incarnation=%s\n", r.Identity.Present, r.Identity.IncarnationID)
	fmt.Fprintf(&b, "usage_store_present: %t\n", r.UsageStore)
	fmt.Fprintf(&b, "netbird: version=%s daemon=%s\n", r.Netbird.Version, r.Netbird.Daemon)
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
