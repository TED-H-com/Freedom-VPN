package installer

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// Install identifies the host, verifies one named release and enrolls
// through fvpn-agent (§6.3, PKG01, ENR03). A second run with the same
// archive reports already_current and does not touch identity, the usage
// store or the update journal (AT07). A newer archive replaces binaries
// and units only.
func Install(ctx context.Context, env *Env, req InstallRequest) (*Result, error) {
	if env == nil {
		env = &Env{}
	}
	if err := validateInstall(&req); err != nil {
		return nil, err
	}
	p, err := Identify(env)
	if err != nil {
		return nil, err
	}
	if err := checkTools(env); err != nil {
		return nil, err
	}
	if err := probeWireGuard(ctx, env); err != nil {
		return nil, err
	}
	if err := requireSystemd(env); err != nil {
		return nil, err
	}
	l := env.layout()
	tempToken := ""
	if !identityPresent(l) {
		if req.TokenFile == "" {
			line, err := promptToken()
			if err != nil {
				return nil, err
			}
			path, err := writeTokenTemp(os.TempDir(), line)
			if err != nil {
				return nil, err
			}
			tempToken = path
			req.TokenFile = path
		}
		if err := checkTokenFile(req.TokenFile); err != nil {
			if tempToken != "" {
				_ = shredFile(tempToken)
			}
			return nil, err
		}
	} else if req.TokenFile != "" {
		if _, statErr := os.Lstat(req.TokenFile); statErr == nil {
			if err := checkTokenFile(req.TokenFile); err != nil {
				return nil, err
			}
		}
	}
	defer func() {
		if tempToken != "" {
			_ = shredFile(tempToken)
		}
	}()

	verified, err := VerifyArchive(req.ManifestPath, req.ArtifactPath, req.TrustPath, p.Arch, env.now())
	if err != nil {
		return nil, err
	}
	if path, lookErr := env.look("netbird"); lookErr == nil && path != "" {
		out, verr := env.output(ctx, "netbird", "version")
		if verr != nil || !netbirdVersionOK(string(out)) {
			return nil, errf(ExitDependency, "NETBIRD_VERSION", "installed netbird is not %s", NetBirdPin)
		}
	}

	lk, err := tryLock(l.LockPath)
	if err != nil {
		if errors.Is(err, errBusy) {
			return nil, errf(ExitBusy, "busy", "update lock is held")
		}
		return nil, err
	}
	released := false
	release := func() {
		if !released {
			_ = lk.Release()
			released = true
		}
	}
	defer release()

	if err := prepareJournal(env); err != nil {
		return nil, err
	}
	cur, has := readInstalled(l)
	id := verified.Manifest.ReleaseId.String()
	if has && cur.ReleaseID == id && cur.ManifestSHA256 != verified.SHA {
		return nil, errf(ExitFailed, "IMMUTABLE", "release %s is already installed with a different manifest", id)
	}
	if has && cur.ReleaseID != id && seqCmp(cur.ReleaseSequence, verified.Manifest.ReleaseSequence.String()) >= 0 {
		return nil, errf(ExitFailed, "DOWNGRADE", "refusing to replace release %s with %s", cur.ReleaseID, id)
	}
	same := has && cur.ReleaseID == id && cur.ManifestSHA256 == verified.SHA
	if same && unitsMatch(l, id) && modesMatch(l) && agentConfigMatch(l, req, verified) {
		release()
		node, _ := readNode(l)
		return &Result{
			Status: statusAlreadyCurrent, ReleaseID: id, ManifestSHA256: verified.SHA,
			IncarnationID: node.NodeIncarnationID,
			Message:       "already_current",
		}, nil
	}

	j, err := openJournal(env)
	if err != nil {
		return nil, err
	}
	fail := func(e error) (*Result, error) {
		_ = j.Rollback(ctx)
		return nil, e
	}
	if err := makeDirs(j, l); err != nil {
		return fail(err)
	}
	if req.FaultAfter == "directories" {
		_ = j.Rollback(ctx)
		return nil, errf(ExitFailed, "rolled_back", "fault after directories")
	}
	installedPkg, err := ensureNetbird(ctx, env, j)
	if err != nil {
		return fail(err)
	}
	if has && !same {
		for _, unit := range stopOrder {
			if err := j.Note("stop", unit, unit, "stopped for binary replace"); err != nil {
				return fail(err)
			}
			_ = env.services().Stop(ctx, unit)
		}
	}
	dest := l.releaseDir(id)
	if err := j.MkdirAll(dest, 0o755); err != nil {
		return fail(err)
	}
	if err := extractRelease(req.ArtifactPath, dest, p.Arch, verified, l); err != nil {
		return fail(err)
	}
	if _, err := os.Lstat(l.CurrentLink); err != nil || !same {
		rel := "releases/" + id
		if err := j.Symlink(l.CurrentLink, rel); err != nil {
			return fail(err)
		}
	}
	if err := installUnits(j, l, id); err != nil {
		return fail(err)
	}
	if err := env.services().DaemonReload(ctx); err != nil {
		return fail(err)
	}
	for _, unit := range requiredUnits {
		if err := env.services().Enable(ctx, unit); err != nil {
			return fail(err)
		}
	}
	if err := writePublic(j, l, req); err != nil {
		return fail(err)
	}
	agentRaw, err := renderAgent(l, req, verified.Manifest)
	if err != nil {
		return fail(err)
	}
	if err := j.WriteFile(l.agentConfig(), agentRaw, 0o644); err != nil {
		return fail(err)
	}
	nb, err := marshalIndent(netbirdConfig{ManagementURL: req.NetbirdManagementURL, InterfaceName: req.MemberIngress})
	if err != nil {
		return fail(err)
	}
	if err := j.WriteFile(l.netbirdFile(), nb, 0o644); err != nil {
		return fail(err)
	}
	fb, err := marshalIndent(firstbootConfig{TokenFile: l.firstbootToken()})
	if err != nil {
		return fail(err)
	}
	if err := j.WriteFile(l.firstbootFile(), fb, 0o644); err != nil {
		return fail(err)
	}
	if err := initIsolation(ctx, env, j, req.MemberIngress, req.Uplink); err != nil {
		return fail(err)
	}
	if !identityPresent(l) {
		if err := env.run(ctx, l.agentBin(), "enroll", "-config", l.agentConfig(), "-token-file", req.TokenFile); err != nil {
			return fail(errf(ExitFailed, "failed", "enrollment failed: %s", oneLine(err.Error())))
		}
		if tempToken == "" {
			_ = shredFile(req.TokenFile)
		}
	}
	if err := writeUpdater(j, env, p, verified.Manifest); err != nil {
		return fail(err)
	}
	st := installedState{
		ReleaseID: id, ManifestSHA256: verified.SHA,
		ReleaseSequence:         verified.Manifest.ReleaseSequence.String(),
		ProductVersion:          verified.Manifest.ProductVersion,
		Arch:                    p.Arch,
		InstalledAt:             env.now().UTC().Format(time.RFC3339),
		NetbirdPackageInstalled: installedPkg || (has && cur.NetbirdPackageInstalled),
	}
	raw, err := marshalIndent(st)
	if err != nil {
		return fail(err)
	}
	if err := j.WriteFile(l.installedFile(), raw, 0o644); err != nil {
		return fail(err)
	}
	if err := j.Seal(); err != nil {
		return fail(err)
	}
	release()
	for _, unit := range startOrder {
		if err := j.Note("start", unit, unit, "started"); err != nil {
			_ = j.Rollback(ctx)
			return nil, err
		}
		if err := env.services().Start(ctx, unit); err != nil {
			_ = j.Rollback(ctx)
			return nil, errf(ExitFailed, "failed", "starting %s failed: %s", unit, oneLine(err.Error()))
		}
	}
	if err := j.Finish(); err != nil {
		return nil, err
	}
	node, _ := readNode(l)
	status := statusInstalled
	if has {
		status = statusRepaired
		if !same {
			status = statusInstalled
		}
	}
	return &Result{
		Status: status, ReleaseID: id, ManifestSHA256: verified.SHA,
		IncarnationID: node.NodeIncarnationID,
	}, nil
}

func makeDirs(j *Journal, l Layout) error {
	for _, d := range dirModes(l) {
		if err := j.MkdirAll(d.path, d.mode); err != nil {
			return err
		}
		// An existing directory keeps the mode it had. A re-run repairs it (§6.3).
		if err := j.Chmod(d.path, d.mode); err != nil {
			return err
		}
	}
	return nil
}

func writePublic(j *Journal, l Layout, req InstallRequest) error {
	if err := copyInto(j, req.ServerCA, l.serverCA()); err != nil {
		return err
	}
	if err := copyInto(j, req.TrustBundle, l.trustBundle()); err != nil {
		return err
	}
	if err := copyInto(j, req.TrustPath, l.releaseTrust()); err != nil {
		return err
	}
	desired := req.DesiredTrustPath
	if desired == "" {
		// The updater refuses to open without a desired-state trust file.
		// Until the operator supplies that root, the release trust is copied
		// so recover can load a store. A desired state signed by another key
		// stays rejected.
		desired = req.TrustPath
	}
	return copyInto(j, desired, l.desiredTrust())
}

func copyInto(j *Journal, src, dst string) error {
	if src == dst {
		return nil
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		return errf(ExitFailed, "failed", "cannot read input file")
	}
	return j.WriteFile(dst, raw, 0o644)
}

func agentConfigMatch(l Layout, req InstallRequest, v Verified) bool {
	want, err := renderAgent(l, req, v.Manifest)
	if err != nil {
		return false
	}
	got, err := os.ReadFile(l.agentConfig())
	if err != nil {
		return false
	}
	return string(got) == string(want)
}

func seqCmp(current, next string) int {
	a, errA := decstr.Parse(current)
	b, errB := decstr.Parse(next)
	if errA != nil || errB != nil {
		if current == next {
			return 0
		}
		if current < next {
			return -1
		}
		return 1
	}
	if a.Uint64() < b.Uint64() {
		return -1
	}
	if a.Uint64() > b.Uint64() {
		return 1
	}
	return 0
}
