package updater

import (
	"context"
	"sort"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// acceptDesired persists the envelope's config_generation when it is not
// older than the last applied one, and unions a present signed revocation
// member into the monotonic set. The unsigned agent list is then combined
// with that set for this job. An absent member does not change the set.
func (e *Engine) acceptDesired(ctx context.Context, work *Work) error {
	if work.PolicyGeneration == 0 && !work.PolicyKnown {
		return permanent(apierr.CheckpointFailed, "policy generation is not applied")
	}
	var ids []string
	if work.SignedRevoked != nil {
		ids = make([]string, 0, len(*work.SignedRevoked))
		for _, id := range *work.SignedRevoked {
			ids = append(ids, id.String())
		}
	}
	applied, accepted, err := e.Journal.AcceptDesired(ctx, work.PolicyGeneration, work.SignedRevoked != nil, ids)
	if err != nil {
		return err
	}
	if !accepted {
		// An older desired state must not overwrite a newer one, and must
		// not put a revoked release back into service (§13.4, §15.3, §17.2).
		return permanent(apierr.StaleGeneration, "config generation")
	}
	work.PolicyGeneration = applied
	work.PolicyKnown = true
	signed, err := e.signedRevokedIDs(ctx)
	if err != nil {
		return err
	}
	work.Revoked = unionUUIDs(work.Revoked, signed)
	return nil
}

func (e *Engine) signedRevokedIDs(ctx context.Context) ([]uuid.UUID, error) {
	if e.Journal == nil {
		return nil, nil
	}
	raw, err := e.Journal.RevokedReleases(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, permanent(apierr.SignatureInvalid, "revoked release id")
		}
		out = append(out, id)
	}
	return out, nil
}

// rollbackIfRevoked treats a currently installed release named by the
// persisted signed set as unhealthy. It restores the newest retained
// release that is not revoked. When none exists, member forwarding stays
// off with recovery_required and the revoked release does not pass the
// gate. An empty signed set does nothing, which is today's behaviour.
func (e *Engine) rollbackIfRevoked(ctx context.Context, c claimed) error {
	signed, err := e.signedRevokedIDs(ctx)
	if err != nil || len(signed) == 0 {
		return err
	}
	p, err := e.Journal.Pointers(ctx)
	if err != nil {
		return err
	}
	cur, err := uuid.Parse(p.CurrentRelease)
	if err != nil || !containsID(signed, cur) {
		return nil
	}
	target, code := e.pickRollback(ctx, c, p.CurrentRelease, nil)
	if target == "" {
		return e.blockRevoked(ctx, c, code)
	}
	if err := e.restoreRelease(ctx, target); err != nil {
		return e.blockRevoked(ctx, c, apierr.RollbackFailed)
	}
	if e.Gate != nil {
		if _, err := e.enforce(ctx, target, c, updatejournal.OutcomeRolledBack); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) blockRevoked(ctx context.Context, c claimed, code apierr.Code) error {
	if code == "" {
		code = apierr.ReleaseNotApproved
	}
	if err := e.noteBlocked(ctx, c, updatejournal.OutcomeRecoveryRequired); err != nil {
		return err
	}
	if c.Intent.JobId != uuid.Nil {
		p, _ := e.Journal.Pointers(ctx)
		schema, _ := e.Journal.StateSchema(ctx)
		started := c.Intent.IssuedAt
		in := resultInput{
			Claimed: c, State: api.UpdateResultStateFailed, Stage: api.UpdateStageRollback,
			Code: code, Current: p.CurrentRelease, Previous: p.CurrentRelease,
			Recovery: true, Started: started, Finished: started,
			Schema: schemaAtLeast1(schemaOr(schema, e.Platform.StateSchema)),
		}
		if err := e.enqueueResult(ctx, in); err != nil {
			return err
		}
	}
	return ErrMembersBlocked
}

// holdIfRevoked blocks forwarding when boot recovery finds a signed-revoked
// release still current. It does not clear a revocation (§17.2).
func (e *Engine) holdIfRevoked(ctx context.Context) error {
	signed, err := e.signedRevokedIDs(ctx)
	if err != nil || len(signed) == 0 {
		return err
	}
	p, err := e.Journal.Pointers(ctx)
	if err != nil {
		return err
	}
	cur, err := uuid.Parse(p.CurrentRelease)
	if err != nil || !containsID(signed, cur) {
		return nil
	}
	rec, err := e.Journal.GateRecord(ctx)
	if err != nil {
		return err
	}
	rec.ForwardingBlocked = true
	rec.TerminalOutcome = updatejournal.OutcomeRecoveryRequired
	if err := e.Journal.SetGate(ctx, rec); err != nil {
		return err
	}
	if e.Gate != nil {
		if err := e.Gate.Block(ctx); err != nil {
			return wrapCode(apierr.UpdateBusy, err, true)
		}
	}
	return nil
}

// pickRollback chooses the newest retained release that passes §13.9 and is
// not revoked. Newest is release_sequence, then installed_at, then id.
// exclude is the candidate that just failed, or the revoked release being
// left. A revoked id is never returned.
func (e *Engine) pickRollback(ctx context.Context, c claimed, exclude string, candidate *api.ReleaseManifest) (string, apierr.Code) {
	signed, err := e.signedRevokedIDs(ctx)
	if err != nil {
		return "", apierr.RollbackFailed
	}
	c.Revoked = unionUUIDs(c.Revoked, signed)
	ids, err := e.Journal.ListReleases(ctx)
	if err != nil {
		return "", apierr.RollbackFailed
	}
	type ranked struct {
		id  string
		seq decstr.U64
		at  time.Time
	}
	var ranks []ranked
	for _, id := range ids {
		if id == "" || id == exclude {
			continue
		}
		rel, err := e.Journal.GetRelease(ctx, id)
		if err != nil {
			continue
		}
		var seq decstr.U64
		if manifest, err := VerifyManifest(rel.ManifestEnvelope, rel.ManifestSHA256, e.ReleaseTrust, e.now()); err == nil {
			seq = manifest.ReleaseSequence
		}
		ranks = append(ranks, ranked{id: id, seq: seq, at: rel.InstalledAt})
	}
	sort.Slice(ranks, func(i, j int) bool {
		if ranks[i].seq != ranks[j].seq {
			return ranks[i].seq > ranks[j].seq
		}
		if !ranks[i].at.Equal(ranks[j].at) {
			return ranks[i].at.After(ranks[j].at)
		}
		return ranks[i].id > ranks[j].id
	})
	var first apierr.Code
	for _, r := range ranks {
		in := e.evalLastGood(ctx, r.id, c, candidate)
		plan := DecideRollback(in)
		if plan.Restore {
			return r.id, ""
		}
		if first == "" {
			first = plan.Code
		}
	}
	if first == "" {
		first = apierr.RollbackFailed
	}
	return "", first
}

// restoreRelease points current at id after the tree and signature checks
// have passed. last_good moves only when the restored release is healthy,
// so a revoked release is not left as the rollback target.
func (e *Engine) restoreRelease(ctx context.Context, id string) error {
	rel, err := e.Layout.relativeRelease(id)
	if err != nil {
		return permanent(apierr.RollbackFailed, "release path")
	}
	if err := fsutil.SyncTree(e.Layout.releaseDir(id)); err != nil {
		return permanent(apierr.RollbackFailed, "sync")
	}
	if err := fsutil.SwitchSymlink(e.Layout.Current, rel); err != nil {
		return permanent(apierr.RollbackFailed, "symlink")
	}
	if err := e.Journal.SetPointers(ctx, updatejournal.Pointers{CurrentRelease: id}); err != nil {
		return err
	}
	if e.Services != nil {
		_ = e.Services.Stop(ctx)
		if err := e.Services.Start(ctx, e.Layout.releaseDir(id)); err != nil {
			return permanent(apierr.StartFailed, "rollback")
		}
	}
	if err := e.snapshotIdentity(); err != nil {
		return err
	}
	if err := e.snapshotPointers(ctx); err != nil {
		return err
	}
	_, ok := e.runHealth(ctx, e.viewFor(ctx, id, e.manifestOf(ctx, id)))
	if !ok {
		return permanent(apierr.RollbackFailed, "restored release")
	}
	if err := e.Journal.SetPointers(ctx, updatejournal.Pointers{LastGood: id}); err != nil {
		return err
	}
	return e.snapshotPointers(ctx)
}

func unionUUIDs(a, b []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(a)+len(b))
	out := make([]uuid.UUID, 0, len(a)+len(b))
	for _, id := range append(append([]uuid.UUID{}, a...), b...) {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// postedResultCode is the UpdateResult error_code. Artifact hash, signature,
// size and archive validation failures are reported as ARTIFACT_INVALID.
// The OpError that drives retry is unchanged, and this code is not an HTTP
// status (the catalog status is zero).
func postedResultCode(code apierr.Code) apierr.Code {
	switch code {
	case apierr.HashMismatch, apierr.SizeMismatch, apierr.ValidationFailed, apierr.SignatureInvalid, apierr.KeyUntrusted:
		return apierr.ArtifactInvalid
	default:
		return code
	}
}
