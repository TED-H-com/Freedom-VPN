//go:build integration

package desiredstate

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type restoreFixture struct {
	t           *testing.T
	pool        *pgxpool.Pool
	compiler    *Compiler
	usage       *usage.Service
	nodes       *nodes.Service
	node, inc   uuid.UUID
	now         time.Time
	oldReport   api.ObservationReport
	oldEnvelope []byte
}

func newRestoreFixture(t *testing.T) *restoreFixture {
	t.Helper()
	f := &restoreFixture{t: t, pool: dbtest.New(t), now: time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)}
	f.node, f.inc, _, _ = seedCompilerTestNode(t, t.Context(), f.pool, uuid.Nil, api.PublicationIntentUnpublished, nil)
	signer, _ := testDesiredStateSignerAndTrustStore(t, f.now)
	f.compiler = NewCompiler(f.pool, signer, WithClock(func() time.Time { return f.now }),
		WithEntitlementReader(NewFakeEntitlementReader()), WithPolicyReader(NewFakePolicyReader(PolicySnapshot{})))
	setNodeGenerationHistory(t, t.Context(), f.pool, f.node, f.inc, 10, 10)
	if _, err := f.compiler.Compile(t.Context(), f.node); err != nil {
		t.Fatal(err)
	}
	setNodeGenerationHistory(t, t.Context(), f.pool, f.node, f.inc, 11, 11, 12, 13)
	if err := f.pool.QueryRow(t.Context(), `SELECT envelope FROM desired_state_envelopes WHERE node_id=$1`, f.node).Scan(&f.oldEnvelope); err != nil {
		t.Fatal(err)
	}
	f.nodes = nodes.NewService(f.pool, nodes.WithClock(func() time.Time { return f.now }))
	f.oldReport = f.report(11)
	f.submit(f.oldReport)
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	f.usage, err = usage.NewService(usage.Config{Pool: f.pool,
		LedgerEpochSigner: usage.NewLedgerEpochSigner("test-restore-epoch", key), Clock: func() time.Time { return f.now }})
	if err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Minute)
	return f
}

func (f *restoreFixture) report(generation uint64) api.ObservationReport {
	f.t.Helper()
	raw, err := os.ReadFile("../../contracts/examples/valid/ObservationReport.json")
	if err != nil {
		f.t.Fatal(err)
	}
	var r api.ObservationReport
	if err := json.Unmarshal(raw, &r); err != nil {
		f.t.Fatal(err)
	}
	r.ReportId, r.NodeId, r.NodeIncarnationId = uuid.New(), f.node, f.inc
	r.ObservedAt = f.now
	r.CurrentReleaseId.SetNull()
	r.LastGoodReleaseId.SetNull()
	r.ProvisioningState = api.ProvisioningStateApplying
	r.SafetyStatus, r.DataHealth = api.SafetyStatusUnknown, api.DataHealthUnknown
	r.Checks, r.Errors, r.ForwardingBlockReasons = []api.NodeCheck{}, []api.NodeError{}, []api.ForwardingBlockReason{}
	r.Netbird.PeerId.SetNull()
	r.Netbird.Backend = api.ObservationReportNetbirdBackendKernel
	r.AppliedConfigGeneration.Set(decstr.U64(generation))
	return r
}

func (f *restoreFixture) submit(r api.ObservationReport) {
	f.t.Helper()
	if _, err := f.nodes.SubmitObservation(f.t.Context(), f.node, f.inc, r); err != nil {
		f.t.Fatal(err)
	}
}

func (f *restoreFixture) freeze() {
	f.t.Helper()
	if err := f.usage.FreezeIngest(f.t.Context(), nodes.SystemActorUUID.String(), "TEST ONLY restore"); err != nil {
		f.t.Fatal(err)
	}
}

func (f *restoreFixture) begin() {
	f.t.Helper()
	f.now = f.now.Add(time.Minute)
	if _, err := f.usage.BeginRecovery(f.t.Context(), usage.RecoveryRequest{
		Actor: nodes.SystemActorUUID.String(), Reason: "TEST ONLY restore", RestoredTo: f.now.Add(-time.Hour),
	}); err != nil {
		f.t.Fatal(err)
	}
}

func (f *restoreFixture) complete() {
	f.t.Helper()
	f.now = f.now.Add(time.Minute)
	if err := f.usage.CompleteRecovery(f.t.Context(), nodes.SystemActorUUID.String()); err != nil {
		f.t.Fatal(err)
	}
}

func (f *restoreFixture) assertHeld() {
	f.t.Helper()
	for range 2 {
		out, err := f.compiler.Compile(f.t.Context(), f.node)
		var hold *apierr.Error
		if !errors.As(err, &hold) || hold.Code != apierr.DependencyUnavailable || !hold.IsRetryable() || hold.Message != usage.RestoreObservationStatus || out.Generation != 0 {
			f.t.Fatalf("compile hold = (%+v, %v)", out, err)
		}
	}
	var desired string
	var count int
	var raw []byte
	if err := f.pool.QueryRow(f.t.Context(), `SELECT n.desired_config_generation::text,
		(SELECT count(*) FROM desired_states WHERE node_id=n.id), e.envelope
		FROM nodes n JOIN desired_state_envelopes e ON e.node_id=n.id WHERE n.id=$1`, f.node).Scan(&desired, &count, &raw); err != nil {
		f.t.Fatal(err)
	}
	if desired != "11" || count != 3 || !bytes.Equal(raw, f.oldEnvelope) {
		f.t.Fatalf("hold changed desired state: desired=%s rows=%d envelope unchanged=%v", desired, count, bytes.Equal(raw, f.oldEnvelope))
	}
	detail, _, err := f.nodes.GetNode(f.t.Context(), f.node)
	if err != nil || len(detail.Drift) != 1 {
		f.t.Fatalf("node hold view: %+v, %v", detail.Drift, err)
	}
	if message, err := detail.Drift[0].Observed.Get(); err != nil || message != usage.RestoreObservationStatus {
		f.t.Fatalf("node hold status = %q, %v", message, err)
	}
	var id uuid.UUID
	if err := f.pool.QueryRow(f.t.Context(), `SELECT id FROM operations WHERE resource_id=$1 AND principal=$2 ORDER BY created_at DESC LIMIT 1`,
		f.node, usage.RestoreObservationPrincipal).Scan(&id); err != nil {
		f.t.Fatal(err)
	}
	op, err := nodes.GetOperation(f.t.Context(), f.pool, id)
	if err != nil || op.Status != api.OperationStatusPending || len(op.Steps) != 1 {
		f.t.Fatalf("hold operation = %+v, %v", op, err)
	}
	if message, err := op.Steps[0].Message.Get(); err != nil || message != usage.RestoreObservationStatus {
		f.t.Fatalf("operation hold status = %q, %v", message, err)
	}
}

func (f *restoreFixture) assertFirstOffer() {
	f.t.Helper()
	out, err := f.compiler.Compile(f.t.Context(), f.node)
	if err != nil || out.Generation != 15 || uint64(out.Envelope.Signed.ConfigGeneration) != 15 {
		f.t.Fatalf("first post-restore offer = %d, %v; want 15", out.Generation, err)
	}
	// node/internal/agentstore/state.go: ErrStaleGeneration rejects an offer
	// older than the retained generation. SECOPS-21 requires the FIRST offer
	// after restore to satisfy that contract, without a rejected 12/13/14 first.
	if !agentAcceptsDesired(14, true, out.Generation) {
		f.t.Fatal("first offer would hit ErrStaleGeneration")
	}
	var offers int
	if err := f.pool.QueryRow(f.t.Context(), `SELECT count(*) FROM desired_states WHERE node_id=$1 AND generation>13`, f.node).Scan(&offers); err != nil || offers != 1 {
		f.t.Fatalf("post-restore offers = %d, %v; want exactly one", offers, err)
	}
	detail, _, err := f.nodes.GetNode(f.t.Context(), f.node)
	if err != nil || len(detail.Drift) != 0 {
		f.t.Fatalf("hold did not clear: %+v, %v", detail.Drift, err)
	}
	var pending int
	if err := f.pool.QueryRow(f.t.Context(), `SELECT count(*) FROM operations WHERE resource_id=$1 AND principal=$2 AND status<>'succeeded'`, f.node, usage.RestoreObservationPrincipal).Scan(&pending); err != nil || pending != 0 {
		f.t.Fatalf("pending hold operations = %d, %v", pending, err)
	}
}

func TestAT30_D095_FirstOfferWaitsForFreshCurrentIncarnation(t *testing.T) {
	f := newRestoreFixture(t)
	f.freeze()
	f.assertHeld() // frozen, not only reconciling
	f.begin()
	f.assertHeld()
	// A duplicate received after freeze keeps its original receipt timestamp.
	f.submit(f.oldReport)
	f.assertHeld()
	// A different incarnation's post-freeze observation cannot release this node.
	other := uuid.New()
	if _, err := f.pool.Exec(t.Context(), `INSERT INTO node_incarnations
		(id,node_id,public_key_fingerprint,os,arch,revoked_at)
		VALUES($1,$2,decode(repeat('ab',32),'hex'),'ubuntu','amd64',now())`, other, f.node); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(t.Context(), `INSERT INTO node_observations
		(node_id,incarnation_id,report_id,payload_hash,observed_at,received_at,applied_generation,
		schema_version,checks,control_status,data_health,safety_status)
		VALUES($1,$2,$3,decode(repeat('cd',32),'hex'),$4,$4,99,1,'{}','online','unknown','unknown')`,
		f.node, other, uuid.New(), f.now); err != nil {
		t.Fatal(err)
	}
	f.assertHeld()
	// An outbox retry must expose the hold, not count it as a compiled node.
	batch, err := f.compiler.RecompileBatch(t.Context(), []db.ClaimedEvent{{EventType: EventNodeUpdated, AggregateID: f.node, Payload: []byte(`{"node_id":"` + f.node.String() + `"}`)}})
	if len(batch) != 0 || err == nil {
		t.Fatalf("held batch = %d, %v", len(batch), err)
	}
	f.now = f.now.Add(time.Second)
	f.submit(f.report(14))
	f.assertFirstOffer()
}

func TestAT30_D095_CompleteRecoveryDoesNotReleaseUnreportedNode(t *testing.T) {
	for _, visited := range []bool{false, true} {
		t.Run(map[bool]string{false: "never compiled during recovery", true: "already held"}[visited], func(t *testing.T) {
			f := newRestoreFixture(t)
			f.freeze()
			f.begin()
			if visited {
				f.assertHeld()
			}
			f.complete()
			f.assertHeld()
			f.now = f.now.Add(time.Second)
			f.submit(f.report(14))
			f.assertFirstOffer()
		})
	}
}

func TestAT30_D095_FreezeBoundarySurvivesBeginAndComplete(t *testing.T) {
	f := newRestoreFixture(t)
	f.freeze()
	// Equal central receipt time is not after freeze, regardless of node time.
	r := f.report(11)
	r.ObservedAt = f.now.Add(time.Hour)
	f.submit(r)
	f.assertHeld()
	f.now = f.now.Add(time.Second)
	fresh := f.report(14)
	fresh.ObservedAt = r.ObservedAt.Add(time.Second)
	f.submit(fresh)
	// Repeating freeze must not move the cutoff past an already-fresh report.
	f.now = f.now.Add(time.Second)
	f.freeze()
	f.begin()
	f.complete()
	f.assertFirstOffer()
}

func TestAT30_D095_NormalWithoutRestoreUnchanged(t *testing.T) {
	f := newRestoreFixture(t)
	out, err := f.compiler.Compile(t.Context(), f.node)
	if err != nil || out.Generation != 14 {
		t.Fatalf("normal compile = %d, %v; want existing max+1 behavior", out.Generation, err)
	}
}

func TestAT30_D095_DirectBeginRecoveryAlsoHolds(t *testing.T) {
	f := newRestoreFixture(t)
	f.begin()
	f.assertHeld()
	f.now = f.now.Add(time.Second)
	f.submit(f.report(14))
	f.assertFirstOffer()
}
