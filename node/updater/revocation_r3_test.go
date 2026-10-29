package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func TestUPD_RevokedCurrentRollsBack(t *testing.T) {
	w := newWorld(t)
	w.releaseSeq = "1"
	older := w.makeRelease("older", 1, []int{1})
	w.releaseSeq = "2"
	cur := w.makeRelease("cur", 1, []int{1})
	w.keep(older)
	w.seed(cur)
	work := w.work(cur, "7", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{older.id, cur.id}
	work.PolicyGeneration = decstr.MustParse("18")
	work.PolicyKnown = true
	revoked := []uuid.UUID{cur.id}
	work.SignedRevoked = &revoked
	w.setWork(work)
	err := w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.ReleaseNotApproved || asOp(err).Retry {
		t.Fatal(err)
	}
	w.mustCurrent(older.id)
	blocked, _, unblocks := w.gate.snapshot()
	if blocked || unblocks == 0 {
		t.Fatalf("gate blocked %v unblocks %d", blocked, unblocks)
	}
	on, why := w.eng.Journal.BlocksForwarding(context.Background())
	if on {
		t.Fatalf("revoked release still blocks after rollback: %s", why)
	}
	assertRevoked(t, w, cur.id)
	res := w.oneResult()
	if res.PolicyGeneration.String() != "18" || res.Generation.String() != "7" {
		t.Fatalf("policy %s generation %s", res.PolicyGeneration, res.Generation)
	}
	if requiredValue(t, res.ErrorCode) != api.ErrorCode(apierr.ReleaseNotApproved) {
		t.Fatalf("code %s", requiredValue(t, res.ErrorCode))
	}
	if requiredValue(t, res.RecoveryRequired) {
		t.Fatal("recovery_required after a safe rollback")
	}
}

func TestUPD_RevokedRollbackTargetSkipped(t *testing.T) {
	w := newWorld(t)
	w.releaseSeq = "1"
	older := w.makeRelease("older", 1, []int{1})
	w.releaseSeq = "2"
	last := w.makeRelease("last", 1, []int{1})
	w.releaseSeq = "3"
	next := w.makeRelease("next", 1, []int{1})
	w.keep(older)
	w.seed(last)
	w.health.fail[next.id.String()] = true
	work := w.work(next, "9", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{older.id, last.id, next.id}
	work.PolicyGeneration = decstr.MustParse("5")
	work.PolicyKnown = true
	revoked := []uuid.UUID{last.id}
	work.SignedRevoked = &revoked
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(older.id)
	p, err := w.eng.Journal.Pointers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if p.CurrentRelease == last.id.String() || p.LastGood == last.id.String() {
		t.Fatalf("revoked release selected current %s last_good %s", p.CurrentRelease, p.LastGood)
	}
	if p.LastGood != older.id.String() {
		t.Fatalf("last_good %s", p.LastGood)
	}
	res := w.oneResult()
	if res.State != api.UpdateResultStateRolledBack || requiredValue(t, res.RecoveryRequired) {
		t.Fatalf("state %s recovery %v", res.State, requiredValue(t, res.RecoveryRequired))
	}
}

func TestUPD_NoNonRevokedReleaseBlocksForwarding(t *testing.T) {
	w := newWorld(t)
	cur := w.makeRelease("cur", 1, []int{1})
	w.seed(cur)
	work := w.work(cur, "4", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{cur.id}
	work.PolicyKnown = true
	revoked := []uuid.UUID{cur.id}
	work.SignedRevoked = &revoked
	w.setWork(work)
	if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	blocked, _, _ := w.gate.snapshot()
	if !blocked {
		t.Fatal("gate opened for a revoked release")
	}
	on, why := w.eng.Journal.BlocksForwarding(context.Background())
	if !on || why != "update recovery_required" {
		t.Fatalf("blocked %v why %q", on, why)
	}
	res := w.oneResult()
	if !requiredValue(t, res.RecoveryRequired) || res.MemberForwardingEnabled {
		t.Fatalf("recovery %v forwarding %v", requiredValue(t, res.RecoveryRequired), res.MemberForwardingEnabled)
	}
	// The symlink may still name the revoked release. It must not pass the gate.
	w.mustCurrent(cur.id)
	enabled, err := w.eng.enforce(context.Background(), cur.id.String(), w.eng.claimFrom(work), updatejournal.OutcomeRolledBack)
	if err != nil || enabled {
		t.Fatalf("revoked release passed the gate enabled %v err %v", enabled, err)
	}
}

func TestUPD_LegacyEnvelopeDoesNotPersistOrRollBack(t *testing.T) {
	node := Node{ID: uuid.Must(uuid.NewV7()), Incarnation: uuid.Must(uuid.NewV7())}
	now := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
	rel := uuid.Must(uuid.NewV7())
	ds := sampleDesired(node, rel, now)
	if ds.RevokedReleaseIds != nil {
		t.Fatal("sample desired state carries a signed revocation member")
	}
	raw, trust := signState(t, ds, envelope.PurposeDesiredState, now)
	unsigned := uuid.Must(uuid.NewV7())
	got, err := WorkFromEnvelope(raw, trust, node, now, []uuid.UUID{unsigned})
	if err != nil {
		t.Fatal(err)
	}
	if got.SignedRevoked != nil || len(got.Revoked) != 1 || got.Revoked[0] != unsigned || got.PolicyGeneration.String() != "9" {
		t.Fatalf("legacy work revoked %v signed %v policy %s", got.Revoked, got.SignedRevoked, got.PolicyGeneration)
	}

	w := newWorld(t)
	w.releaseSeq = "1"
	older := w.makeRelease("older", 1, []int{1})
	w.releaseSeq = "2"
	cur := w.makeRelease("cur", 1, []int{1})
	w.keep(older)
	w.seed(cur)
	work := w.work(cur, "3", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{older.id, cur.id}
	work.Revoked = []uuid.UUID{cur.id}
	w.setWork(work)
	err = w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.ReleaseNotApproved {
		t.Fatal(err)
	}
	w.mustCurrent(cur.id)
	ids, err := w.eng.Journal.RevokedReleases(context.Background())
	if err != nil || len(ids) != 0 {
		t.Fatalf("persisted signed set %v err %v", ids, err)
	}

	w.releaseSeq = "4"
	next := w.makeRelease("next", 1, []int{1})
	again := w.work(next, "8", api.PublicationIntentPublished)
	again.Serving = []uuid.UUID{cur.id, next.id}
	again.Revoked = nil
	w.setWork(again)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(next.id)
}

func TestUPD_SignedRevocationIsMonotonic(t *testing.T) {
	w := newWorld(t)
	cur := w.makeRelease("cur", 1, []int{1})
	w.releaseSeq = "5"
	next := w.makeRelease("next", 1, []int{1})
	w.seed(cur)
	stay := w.work(cur, "2", api.PublicationIntentPublished)
	stay.PolicyGeneration = decstr.MustParse("5")
	stay.PolicyKnown = true
	stay.Serving = []uuid.UUID{cur.id, next.id}
	revoked := []uuid.UUID{next.id}
	stay.SignedRevoked = &revoked
	w.setWork(stay)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	assertRevoked(t, w, next.id)

	legacy := w.work(next, "3", api.PublicationIntentPublished)
	legacy.PolicyGeneration = decstr.MustParse("6")
	legacy.PolicyKnown = true
	legacy.Serving = []uuid.UUID{cur.id, next.id}
	legacy.Revoked = nil
	legacy.SignedRevoked = nil
	w.setWork(legacy)
	err := w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.ReleaseNotApproved {
		t.Fatal(err)
	}
	w.mustCurrent(cur.id)
	assertRevoked(t, w, next.id)

	empty := []uuid.UUID{}
	blank := w.work(next, "4", api.PublicationIntentPublished)
	blank.PolicyGeneration = decstr.MustParse("7")
	blank.PolicyKnown = true
	blank.Serving = legacy.Serving
	blank.SignedRevoked = &empty
	w.setWork(blank)
	err = w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.ReleaseNotApproved {
		t.Fatal(err)
	}
	assertRevoked(t, w, next.id)

	older := w.work(next, "8", api.PublicationIntentPublished)
	older.PolicyGeneration = decstr.MustParse("4")
	older.PolicyKnown = true
	older.Serving = legacy.Serving
	clear := []uuid.UUID{}
	older.SignedRevoked = &clear
	w.setWork(older)
	err = w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.StaleGeneration {
		t.Fatalf("older config generation: %v", err)
	}
	w.mustCurrent(cur.id)
	assertRevoked(t, w, next.id)
}

func TestUPD_ArtifactInvalidAndAppliedPolicyGeneration(t *testing.T) {
	w := newWorld(t)
	err := w.eng.enqueueResult(context.Background(), resultInput{Claimed: claimed{Work: Work{
		Intent: api.UpdateIntent{Generation: decstr.MustParse("7")},
	}}})
	if err == nil {
		t.Fatal("posted a result before the applied policy generation was known")
	}
	if box, _ := w.eng.Journal.Outbox(context.Background()); len(box) != 0 {
		t.Fatalf("outbox %d", len(box))
	}

	t.Run("hash", func(t *testing.T) {
		w := newWorld(t)
		old := w.makeRelease("old", 1, []int{1})
		next := w.makeRelease("new", 1, []int{1})
		w.seed(old)
		bad := bytes.Repeat([]byte("x"), len(next.amd))
		w.mu.Lock()
		w.files["/a/"+next.id.String()+"/amd64"] = fileSpec{body: bad}
		w.mu.Unlock()
		work := w.work(next, "7", api.PublicationIntentPublished)
		work.Serving = []uuid.UUID{old.id, next.id}
		work.PolicyGeneration = decstr.MustParse("18")
		work.PolicyKnown = true
		w.setWork(work)
		w.eng.Hooks.AfterPhase = func(name string) error {
			if name != "verifying" {
				return nil
			}
			downloaded := work
			downloaded.PolicyGeneration = decstr.MustParse("99")
			w.setWork(downloaded)
			return nil
		}
		err := w.run()
		if asOp(err) == nil || asOp(err).Code != apierr.HashMismatch || asOp(err).Retry {
			t.Fatal(err)
		}
		res := w.oneResult()
		if requiredValue(t, res.ErrorCode) != api.ErrorCode(apierr.ArtifactInvalid) {
			t.Fatalf("code %s", requiredValue(t, res.ErrorCode))
		}
		if res.PolicyGeneration.String() != "18" || res.Generation.String() != "7" {
			t.Fatalf("policy %s generation %s", res.PolicyGeneration, res.Generation)
		}
	})

	t.Run("size", func(t *testing.T) {
		w := newWorld(t)
		old := w.makeRelease("old", 1, []int{1})
		next := w.makeRelease("new", 1, []int{1})
		w.seed(old)
		w.mu.Lock()
		spec := w.files["/a/"+next.id.String()+"/amd64"]
		spec.body = append(spec.body, []byte("too-long")...)
		w.files["/a/"+next.id.String()+"/amd64"] = spec
		w.mu.Unlock()
		work := w.work(next, "7", api.PublicationIntentPublished)
		work.Serving = []uuid.UUID{old.id, next.id}
		work.PolicyGeneration = decstr.MustParse("18")
		work.PolicyKnown = true
		w.setWork(work)
		err := w.run()
		if asOp(err) == nil || asOp(err).Code != apierr.SizeMismatch || asOp(err).Retry {
			t.Fatal(err)
		}
		if requiredValue(t, w.oneResult().ErrorCode) != api.ErrorCode(apierr.ArtifactInvalid) {
			t.Fatalf("code %s", requiredValue(t, w.oneResult().ErrorCode))
		}
	})

	t.Run("archive", func(t *testing.T) {
		w := newWorld(t)
		old := w.makeRelease("old", 1, []int{1})
		w.seed(old)
		next := w.makeRelease("bad", 1, []int{1})
		replaceAmd(t, w, next, hostileTar(t, "symlink"))
		work := w.work(next, "11", api.PublicationIntentPublished)
		work.Serving = []uuid.UUID{old.id, next.id}
		work.PolicyGeneration = decstr.MustParse("18")
		work.PolicyKnown = true
		w.mu.Lock()
		body := w.files["/m/"+next.id.String()].body
		w.mu.Unlock()
		sum := sha256Hex(body)
		work.Intent.ManifestSha256 = sum
		w.setWork(work)
		err := w.run()
		if asOp(err) == nil || asOp(err).Code != apierr.ValidationFailed || asOp(err).Retry {
			t.Fatal(err)
		}
		if requiredValue(t, w.oneResult().ErrorCode) != api.ErrorCode(apierr.ArtifactInvalid) {
			t.Fatalf("code %s", requiredValue(t, w.oneResult().ErrorCode))
		}
	})

	t.Run("signature", func(t *testing.T) {
		w := newWorld(t)
		old := w.makeRelease("old", 1, []int{1})
		next := w.makeRelease("new", 1, []int{1})
		w.seed(old)
		bad, sum := resignOther(t, next.env)
		w.mu.Lock()
		w.files["/m/"+next.id.String()] = fileSpec{body: bad}
		w.mu.Unlock()
		work := w.work(next, "9", api.PublicationIntentPublished)
		work.Intent.ManifestSha256 = sum
		work.Serving = []uuid.UUID{old.id, next.id}
		work.PolicyGeneration = decstr.MustParse("18")
		work.PolicyKnown = true
		w.setWork(work)
		err := w.run()
		op := asOp(err)
		if op == nil || (op.Code != apierr.SignatureInvalid && op.Code != apierr.KeyUntrusted) || op.Retry {
			t.Fatal(err)
		}
		if requiredValue(t, w.oneResult().ErrorCode) != api.ErrorCode(apierr.ArtifactInvalid) {
			t.Fatalf("code %s", requiredValue(t, w.oneResult().ErrorCode))
		}
		res := w.oneResult()
		if res.PolicyGeneration.String() != "18" || res.Generation.String() != "9" {
			t.Fatalf("policy %s generation %s", res.PolicyGeneration, res.Generation)
		}
	})
}

func TestUPD_SupersededResultKeepsAppliedGeneration(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	first := w.makeRelease("first", 1, []int{1})
	second := w.makeRelease("second", 1, []int{1})
	w.seed(old)
	w1 := w.work(first, "20", api.PublicationIntentPublished)
	w1.Serving = []uuid.UUID{old.id, first.id, second.id}
	w1.PolicyGeneration = decstr.MustParse("18")
	w1.PolicyKnown = true
	w2 := w.work(second, "21", api.PublicationIntentPublished)
	w2.Serving = w1.Serving
	w2.PolicyGeneration = decstr.MustParse("99")
	w2.PolicyKnown = true
	w.setWork(w1)
	w.eng.Hooks.AfterPhase = func(name string) error {
		if name == "downloading" {
			w.src.set(w2)
		}
		return nil
	}
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	box, err := w.eng.Journal.Outbox(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var sawOld, sawNew bool
	for _, ent := range box {
		var res api.UpdateResult
		if err := json.Unmarshal(ent.Payload, &res); err != nil {
			t.Fatal(err)
		}
		switch res.State {
		case api.UpdateResultStateSuperseded:
			sawOld = true
			if res.PolicyGeneration.String() != "18" || res.Generation.String() != "20" {
				t.Fatalf("superseded policy %s generation %s", res.PolicyGeneration, res.Generation)
			}
		case api.UpdateResultStateSucceeded:
			sawNew = true
			if res.PolicyGeneration.String() != "99" || res.Generation.String() != "21" {
				t.Fatalf("succeeded policy %s generation %s", res.PolicyGeneration, res.Generation)
			}
		}
	}
	if !sawOld || !sawNew {
		t.Fatalf("outcomes %v", w.outcomes())
	}
}

func assertRevoked(t *testing.T, w *world, id uuid.UUID) {
	t.Helper()
	ids, err := w.eng.Journal.RevokedReleases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range ids {
		if got == id.String() {
			return
		}
	}
	t.Fatalf("release %s not in persisted set %v", id, ids)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hexEncode(sum[:])
}

// resignOther re-signs a release envelope with a key the node does not trust.
// The returned hash is of the canonical bytes the updater downloads.
func resignOther(t *testing.T, envBytes []byte) ([]byte, string) {
	t.Helper()
	other, err := release.GenerateKey("other-release-key")
	if err != nil {
		t.Fatal(err)
	}
	signed, err := canonjson.Canonicalize(envBytes)
	if err != nil {
		t.Fatal(err)
	}
	var env envelope.Envelope
	if err := json.Unmarshal(signed, &env); err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(env.Signed, &obj); err != nil {
		t.Fatal(err)
	}
	badEnv, err := envelope.Sign(obj, other.Private(), other.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	bad, err := badEnv.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bad)
	return bad, hexEncode(sum[:])
}
