package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
// supplied one. Process checks compare the release tree. Persistent data
// checks the identity directory and the immutable pointers and state-schema
// record, not the mutable state directory (D-034c). Isolation, DNS and the
// tunnel probe fail closed: a config file cannot mark them passed (§13.9).
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
	if v.ExpectedIdentity == "" {
		return api.HealthCheckStatusFailed, nil
	}
	id, err := fsutil.TreeHash(v.IdentityDir)
	if err != nil || id != v.ExpectedIdentity {
		return api.HealthCheckStatusFailed, nil
	}
	// Callers that do not supply the immutable record are not asking for the
	// pointers check. The mutable state directory is never hashed (D-034c).
	if v.ImmutableRecord == "" {
		return api.HealthCheckStatusPassed, nil
	}
	b, err := os.ReadFile(v.ImmutableRecord)
	if err != nil || v.ExpectedState == "" || digestBytes(b) != v.ExpectedState {
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

func (e *Engine) evalLastGood(ctx context.Context, id string, c claimed, candidate *api.ReleaseManifest) RollbackInput {
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
		// A non-empty allow list is a fail-closed intersection with the
		// serving set (D-034e). An empty list adds no extra restriction.
		if candidate != nil && len(candidate.Rollback.AllowedReleaseIds) > 0 && !containsID(candidate.Rollback.AllowedReleaseIds, uid) {
			in.InServingSet = false
		}
	}
	cur, serr := e.Journal.StateSchema(ctx)
	// Schema 0 means no migration has been recorded. A migrated schema is
	// readable only when it still equals last_good. The candidate manifest
	// must not override that.
	in.SchemaReadable = serr == nil && (cur == 0 || cur == rel.StateSchemaTo)
	return in
}

// snapshotIdentity records the identity tree hash. It is taken after
// migration and before services start, and is not refreshed on rollback.
func (e *Engine) snapshotIdentity() error {
	idHash, err := fsutil.TreeHash(e.Layout.Identity)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(e.Layout.Journal+"/identity.snap", []byte(idHash), 0o600)
}

// snapshotPointers records the pointers and the state-schema marker. The
// mutable state directory is not part of the record (D-034c).
func (e *Engine) snapshotPointers(ctx context.Context) error {
	body, err := e.immutableBody(ctx)
	if err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomic(e.Layout.Journal+"/immutable.record", body, 0o600); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(e.Layout.Journal+"/immutable.snap", []byte(digestBytes(body)), 0o600)
}

func (e *Engine) immutableBody(ctx context.Context) ([]byte, error) {
	p, err := e.Journal.Pointers(ctx)
	if err != nil {
		return nil, err
	}
	schema, err := e.Journal.StateSchema(ctx)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("current=%s\nlast_good=%s\nschema=%d\n", p.CurrentRelease, p.LastGood, schema)), nil
}

func (e *Engine) writeImmutableLive(ctx context.Context) (string, error) {
	body, err := e.immutableBody(ctx)
	if err != nil {
		return "", err
	}
	path := e.Layout.Journal + "/immutable.live"
	if err := fsutil.WriteFileAtomic(path, body, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func digestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (e *Engine) readSnap(name string) string {
	b, err := os.ReadFile(e.Layout.Journal + "/" + name)
	if err != nil {
		return ""
	}
	return string(b)
}

func (e *Engine) viewFor(ctx context.Context, releaseID string, manifest *api.ReleaseManifest) ReleaseView {
	live, _ := e.writeImmutableLive(ctx)
	return ReleaseView{
		ReleaseID:        releaseID,
		Dir:              e.Layout.releaseDir(releaseID),
		Manifest:         manifest,
		IdentityDir:      e.Layout.Identity,
		StateDir:         e.Layout.State,
		ExpectedIdentity: e.readSnap("identity.snap"),
		ExpectedState:    e.readSnap("immutable.snap"),
		ImmutableRecord:  live,
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
