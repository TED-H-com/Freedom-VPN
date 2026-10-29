package updater

import (
	"context"
	"os"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

const (
	maxCheckAttempts    = 8
	maxRollbackAttempts = 3
)

// NodeChecks is the stock health adapter used when the agent has not
// supplied one. Process and persistent-data checks compare tree hashes.
// Isolation, DNS and the tunnel probe fail closed: a config file cannot
// mark them passed (§13.9).
type NodeChecks struct{}

func (NodeChecks) Processes(_ context.Context, v ReleaseView) (api.HealthCheckStatus, error) {
	if v.ExpectedTree == "" || v.Dir == "" {
		return api.HealthCheckStatusFailed, nil
	}
	got, err := fsutil.TreeHash(v.Dir)
	if err != nil || got != v.ExpectedTree {
		return api.HealthCheckStatusFailed, nil
	}
	return api.HealthCheckStatusPassed, nil
}

func (NodeChecks) PersistentData(_ context.Context, v ReleaseView) (api.HealthCheckStatus, error) {
	if v.ExpectedIdentity == "" || v.ExpectedState == "" {
		return api.HealthCheckStatusFailed, nil
	}
	id, err := fsutil.TreeHash(v.IdentityDir)
	if err != nil || id != v.ExpectedIdentity {
		return api.HealthCheckStatusFailed, nil
	}
	st, err := fsutil.TreeHash(v.StateDir)
	if err != nil || st != v.ExpectedState {
		return api.HealthCheckStatusFailed, nil
	}
	return api.HealthCheckStatusPassed, nil
}

func (NodeChecks) Isolation(context.Context, ReleaseView) (api.HealthCheckStatus, error) {
	return api.HealthCheckStatusFailed, nil
}

func (NodeChecks) DNSPolicy(context.Context, ReleaseView) (api.HealthCheckStatus, error) {
	return api.HealthCheckStatusFailed, nil
}

func (NodeChecks) TunnelProbe(context.Context, ReleaseView) (api.HealthCheckStatus, error) {
	return api.HealthCheckStatusFailed, nil
}

func (e *Engine) runHealth(ctx context.Context, v ReleaseView) (api.UpdateHealth, bool) {
	h := api.UpdateHealth{
		Processes:        api.HealthCheckStatusNotRun,
		IdentityAndState: api.HealthCheckStatusNotRun,
		InternetOnly:     api.HealthCheckStatusNotRun,
		DnsPolicy:        api.HealthCheckStatusNotRun,
		TunnelProbe:      api.HealthCheckStatusNotRun,
	}
	if e.Health == nil {
		h.Processes = api.HealthCheckStatusFailed
		return h, false
	}
	var err error
	if h.Processes, err = e.Health.Processes(ctx, v); err != nil {
		h.Processes = api.HealthCheckStatusFailed
	}
	if h.IdentityAndState, err = e.Health.PersistentData(ctx, v); err != nil {
		h.IdentityAndState = api.HealthCheckStatusFailed
	}
	if h.InternetOnly, err = e.Health.Isolation(ctx, v); err != nil {
		h.InternetOnly = api.HealthCheckStatusFailed
	}
	if h.DnsPolicy, err = e.Health.DNSPolicy(ctx, v); err != nil {
		h.DnsPolicy = api.HealthCheckStatusFailed
	}
	if h.TunnelProbe, err = e.Health.TunnelProbe(ctx, v); err != nil {
		h.TunnelProbe = api.HealthCheckStatusFailed
	}
	return h, healthPassed(h)
}

func (e *Engine) evalLastGood(ctx context.Context, id string, c claimed) RollbackInput {
	var in RollbackInput
	if id == "" {
		return in
	}
	rel, err := e.Journal.GetRelease(ctx, id)
	if err != nil {
		return in
	}
	in.HasLastGood = true
	sum, err := fsutil.TreeHash(e.Layout.releaseDir(id))
	in.TreeOK = err == nil && sum == rel.TreeHash
	_, err = VerifyManifest(rel.ManifestEnvelope, rel.ManifestSHA256, e.ReleaseTrust, e.now())
	in.SignatureOK = err == nil
	uid, perr := parseUUID(id)
	if perr == nil {
		in.InServingSet = containsID(c.Serving, uid)
		in.Revoked = containsID(c.Revoked, uid)
	}
	cur, serr := e.Journal.StateSchema(ctx)
	// Schema 0 means no migration has been recorded. A migrated schema is
	// readable when it still equals last_good, or when this transaction
	// was allowed to migrate because the manifest promised readability.
	in.SchemaReadable = serr == nil && (cur == 0 || cur == rel.StateSchemaTo)
	return in
}

func (e *Engine) snapshotPersistent() error {
	idHash, err := fsutil.TreeHash(e.Layout.Identity)
	if err != nil {
		return err
	}
	stHash, err := fsutil.TreeHash(e.Layout.State)
	if err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomic(e.Layout.Journal+"/identity.snap", []byte(idHash), 0o600); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(e.Layout.Journal+"/state.snap", []byte(stHash), 0o600)
}

func (e *Engine) readSnap(name string) string {
	b, err := os.ReadFile(e.Layout.Journal + "/" + name)
	if err != nil {
		return ""
	}
	return string(b)
}

func (e *Engine) viewFor(ctx context.Context, releaseID string, manifest *api.ReleaseManifest) ReleaseView {
	return ReleaseView{
		ReleaseID:        releaseID,
		Dir:              e.Layout.releaseDir(releaseID),
		Manifest:         manifest,
		IdentityDir:      e.Layout.Identity,
		StateDir:         e.Layout.State,
		ExpectedIdentity: e.readSnap("identity.snap"),
		ExpectedState:    e.readSnap("state.snap"),
		ExpectedTree:     e.treeOf(ctx, releaseID),
	}
}

func (e *Engine) treeOf(ctx context.Context, id string) string {
	rel, err := e.Journal.GetRelease(ctx, id)
	if err != nil {
		return ""
	}
	return rel.TreeHash
}

func bumpCounter(path string) (int, error) {
	n := 0
	if b, err := os.ReadFile(path); err == nil {
		for _, c := range b {
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
	}
	n++
	if err := fsutil.WriteFileAtomic(path, []byte(itoa(n)), 0o600); err != nil {
		return 0, err
	}
	return n, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func (e *Engine) openEpoch(ctx context.Context, txnID string) (string, error) {
	path := e.Layout.Journal + "/epoch-" + txnID
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		return string(trimSpace(b)), nil
	}
	if e.Meter == nil {
		return "", permanent(apierr.CheckpointFailed, "no meter")
	}
	id, err := e.Meter.OpenEpoch(ctx)
	if err != nil {
		return "", wrapCode(apierr.CheckpointFailed, err, true)
	}
	if _, err := parseUUID(id); err != nil {
		return "", permanent(apierr.CheckpointFailed, "meter epoch id")
	}
	if err := fsutil.WriteFileAtomic(path, []byte(id), 0o600); err != nil {
		return "", err
	}
	return id, nil
}

func trimSpace(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return b
}

// clock helper kept next to health so boot recovery uses one now.
func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}
