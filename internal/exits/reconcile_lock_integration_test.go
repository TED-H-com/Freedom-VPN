//go:build integration

package exits

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (f *reconcileFixture) seedAt(nodeID, exitID uuid.UUID, deployment, intent string) {
	f.t.Helper()
	incID := uuid.New()
	fp := sha256.Sum256([]byte("fp-" + exitID.String()))
	f.exec(`INSERT INTO nodes(id,lifecycle_state,provisioning_state,control_status,data_health,safety_status) VALUES($1,'registered','ready','online','healthy','passed')`, nodeID)
	f.exec(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,deployment_id,peer_id,os,arch) VALUES($1,$2,$3,$4,$5,'ubuntu','amd64')`, incID, nodeID, fp[:], deployment, "peer-"+exitID.String())
	f.exec(`UPDATE nodes SET current_incarnation_id=$2, resource_version=resource_version+1 WHERE id=$1`, nodeID, incID)
	f.exec(`INSERT INTO exits(id,node_id,display_name,region,publication_intent,publication_observed) VALUES($1,$2,'TEST ONLY exit','TEST ONLY',$3,'absent')`, exitID, nodeID, intent)
}

func (f *reconcileFixture) seedExitBinding(exitID uuid.UUID, deployment, externalID string, desired []byte) {
	f.t.Helper()
	f.exec(`INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash,sync_state) VALUES('exit',$1,'netbird',$2,$3,$4,'pending')`, exitID, deployment, externalID, desired)
}

func (f *reconcileFixture) bindingText(exitID uuid.UUID) string {
	f.t.Helper()
	var text string
	if err := f.pool.QueryRow(f.ctx, `SELECT b::text FROM external_bindings b WHERE resource_type='exit' AND resource_id=$1`, exitID).Scan(&text); err != nil {
		f.t.Fatal(err)
	}
	return text
}

func waitForWaiter(t *testing.T, ctx context.Context, pool *pgxpool.Pool, blocker int32, done <-chan error) int32 {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for {
		var waiter int32
		err := pool.QueryRow(ctx, `SELECT pid FROM pg_stat_activity WHERE datname=current_database() AND $1::int = ANY(pg_blocking_pids(pid)) ORDER BY pid LIMIT 1`, blocker).Scan(&waiter)
		if err == nil {
			return waiter
		}
		if !errors.Is(err, pgx.ErrNoRows) && ctx.Err() == nil {
			t.Fatal(err)
		}
		select {
		case recErr := <-done:
			t.Fatalf("finished before a lock wait on %d: %v", blocker, recErr)
		case <-ctx.Done():
			t.Fatalf("timed out waiting for a lock wait on %d: %v", blocker, ctx.Err())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func blockingPIDs(ctx context.Context, pool *pgxpool.Pool, pid int32) ([]int32, error) {
	var ids []int32
	err := pool.QueryRow(ctx, `SELECT pg_blocking_pids($1::int)`, pid).Scan(&ids)
	return ids, err
}

func databaseDeadlocks(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()
	for _, conn := range pool.AcquireAllIdle(ctx) {
		_, err := conn.Exec(ctx, `SELECT pg_stat_force_next_flush()`)
		conn.Release()
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT deadlocks FROM pg_stat_database WHERE datname=current_database()`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func withdrawRoute(exitID uuid.UUID) ObservedExitState {
	return ObservedExitState{Exists: true, Enabled: false, ExternalID: "nb-route-" + exitID.String(), PeerID: "peer"}
}

func TestReconcileNetBird_ConcurrentDriftAppend(t *testing.T) {
	f := newReconcileFixture(t)
	exitID := uuid.MustParse("00000000-0000-4000-8000-0000000000a1")
	nodeID := f.seedOrdered(exitID, "nb-main", string(api.PublicationIntentPublished))
	f.nb.InjectRouteState(exitID, withdrawRoute(exitID))
	foreign := api.StateDrift{
		Field:    "desired_state_restore_observation",
		Desired:  presence.NewNullableWithValue("held-desired"),
		Observed: presence.NewNullableWithValue("held-observed"),
	}
	raw, err := json.Marshal(foreign)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(f.ctx, 30*time.Second)
	defer cancel()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	holder := int32(tx.Conn().PgConn().PID())
	var heldVersion string
	if err = tx.QueryRow(ctx, `UPDATE nodes SET drift = drift || jsonb_build_array($2::jsonb), resource_version = resource_version + 1 WHERE id = $1 RETURNING resource_version::text`, nodeID, raw).Scan(&heldVersion); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, recErr := f.svc.ReconcileNetBird(ctx)
		done <- recErr
	}()
	waitForWaiter(t, ctx, f.pool, holder, done)
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	after, drift := f.nodeState(nodeID)
	versionPlus(t, heldVersion, after, 1)
	if len(drift) != 2 || drift[0].Field != foreign.Field || drift[1].Field != driftFieldRoute {
		t.Fatalf("drift=%+v", drift)
	}
	if fieldText(t, drift[0].Desired) != "held-desired" || fieldText(t, drift[0].Observed) != "held-observed" {
		t.Fatalf("foreign=%+v", drift[0])
	}
	if fieldText(t, drift[1].Desired) != "published" || fieldText(t, drift[1].Observed) != "withdrawn" {
		t.Fatalf("route=%+v", drift[1])
	}
}

func TestReconcileNetBird_ConcurrentDriftRemoval(t *testing.T) {
	f := newReconcileFixture(t)
	exitID := uuid.MustParse("00000000-0000-4000-8000-0000000000a2")
	nodeID := f.seedOrdered(exitID, "nb-main", string(api.PublicationIntentPublished))
	f.nb.InjectRouteState(exitID, withdrawRoute(exitID))
	foreign := api.StateDrift{
		Field:    "desired_state_restore_observation",
		Desired:  presence.NewNullableWithValue("remove-desired"),
		Observed: presence.NewNullableWithValue("remove-observed"),
	}
	f.setDrift(nodeID, []api.StateDrift{foreign})
	ctx, cancel := context.WithTimeout(f.ctx, 30*time.Second)
	defer cancel()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	holder := int32(tx.Conn().PgConn().PID())
	var heldVersion string
	if err = tx.QueryRow(ctx, `UPDATE nodes SET drift = coalesce((
		SELECT jsonb_agg(elem ORDER BY ord)
		FROM jsonb_array_elements(drift) WITH ORDINALITY AS t(elem, ord)
		WHERE coalesce(elem->>'field','') <> $2
	), '[]'::jsonb), resource_version = resource_version + 1
	WHERE id = $1 RETURNING resource_version::text`, nodeID, foreign.Field).Scan(&heldVersion); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, recErr := f.svc.ReconcileNetBird(ctx)
		done <- recErr
	}()
	waitForWaiter(t, ctx, f.pool, holder, done)
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	after, drift := f.nodeState(nodeID)
	versionPlus(t, heldVersion, after, 1)
	if len(drift) != 1 || drift[0].Field != driftFieldRoute {
		t.Fatalf("drift=%+v", drift)
	}
	if fieldText(t, drift[0].Desired) != "published" || fieldText(t, drift[0].Observed) != "withdrawn" {
		t.Fatalf("route=%+v", drift[0])
	}
}

func TestReconcileNetBird_LockOrderWithFreeze(t *testing.T) {
	f := newReconcileFixture(t)
	var mode string
	if err := f.pool.QueryRow(f.ctx, `SELECT mode FROM usage_ledger_state`).Scan(&mode); err != nil || mode != "normal" {
		t.Fatalf("ledger mode %q: %v", mode, err)
	}
	nodeLow := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	nodeHigh := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	exitLow := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	exitHigh := uuid.MustParse("00000000-0000-4000-8000-000000000020")
	if bytes.Compare(nodeLow[:], nodeHigh[:]) >= 0 || bytes.Compare(exitLow[:], exitHigh[:]) >= 0 {
		t.Fatal("fixture ids are not ordered")
	}
	const deployment = "nb-main"
	// The lower exit id sits on the higher node id, so exit order locks the
	// higher node first. FreezeIngest locks nodes by id, low then high.
	f.seedAt(nodeHigh, exitLow, deployment, string(api.PublicationIntentPublished))
	f.seedAt(nodeLow, exitHigh, deployment, string(api.PublicationIntentPublished))
	f.nb.InjectRouteState(exitLow, withdrawRoute(exitLow))
	f.nb.InjectRouteState(exitHigh, withdrawRoute(exitHigh))
	hash := bytes.Repeat([]byte{0x21}, 32)
	f.seedExitBinding(exitLow, deployment, "nb-route-"+exitLow.String(), hash)
	f.seedExitBinding(exitHigh, deployment, "nb-route-"+exitHigh.String(), hash)
	usageSvc, err := usage.NewService(usage.Config{Pool: f.pool, Clock: func() time.Time {
		return time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(f.ctx, 45*time.Second)
	defer cancel()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	holder := int32(tx.Conn().PgConn().PID())
	var locked uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT id FROM external_bindings WHERE resource_type='exit' AND resource_id=$1 AND provider='netbird' AND deployment_id=$2 FOR UPDATE`, exitLow, deployment).Scan(&locked); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, recErr := f.svc.ReconcileNetBird(ctx)
		done <- recErr
	}()
	reconcilePID := waitForWaiter(t, ctx, f.pool, holder, done)
	if reconcilePID == holder {
		t.Fatalf("reconcile pid %d is the binding holder", reconcilePID)
	}
	before := databaseDeadlocks(t, ctx, f.pool)
	freezeDone := make(chan error, 1)
	go func() {
		freezeDone <- usageSvc.FreezeIngest(ctx, "00000000-0000-4000-8000-0000000000f1", "TEST ONLY freeze")
	}()
	freezePID := waitForWaiter(t, ctx, f.pool, reconcilePID, freezeDone)
	if freezePID == holder || freezePID == reconcilePID {
		t.Fatalf("freeze pid %d collides with holder %d or reconcile %d", freezePID, holder, reconcilePID)
	}
	sawFreeze := false
	sample := func() {
		blockers, sampleErr := blockingPIDs(ctx, f.pool, reconcilePID)
		if sampleErr != nil {
			return
		}
		for _, pid := range blockers {
			if pid == freezePID {
				sawFreeze = true
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	sample()
	ticker := time.NewTicker(2 * time.Millisecond)
	defer ticker.Stop()
	waitCtx, waitCancel := context.WithTimeout(ctx, 20*time.Second)
	defer waitCancel()
	var recErr, frzErr error
	recFinished, frzFinished := false, false
	for !recFinished || !frzFinished {
		select {
		case recErr = <-done:
			recFinished = true
			sample()
		case frzErr = <-freezeDone:
			frzFinished = true
			sample()
		case <-ticker.C:
			sample()
		case <-waitCtx.Done():
			t.Fatalf("timed out reconcile_done=%t freeze_done=%t saw_freeze=%t: %v", recFinished, frzFinished, sawFreeze, waitCtx.Err())
		}
	}
	if recErr != nil {
		t.Fatal(recErr)
	}
	if frzErr != nil {
		t.Fatal(frzErr)
	}
	if sawFreeze {
		t.Fatal("reconcile waited on freeze")
	}
	after := databaseDeadlocks(t, f.ctx, f.pool)
	if after != before {
		t.Fatalf("deadlocks %d -> %d", before, after)
	}
}

func TestReconcileNetBird_SkipsStaleIntent(t *testing.T) {
	f := newReconcileFixture(t)
	changed := uuid.MustParse("00000000-0000-4000-8000-000000000031")
	stable := uuid.MustParse("00000000-0000-4000-8000-000000000032")
	changedNode := f.seedOrdered(changed, "nb-main", string(api.PublicationIntentUnpublished))
	stableNode := f.seedOrdered(stable, "nb-main", string(api.PublicationIntentPublished))
	marker := []api.StateDrift{{
		Field:    "capacity_report",
		Desired:  presence.NewNullableWithValue("stay"),
		Observed: presence.NewNullableWithValue("stay"),
	}}
	f.setDrift(changedNode, marker)
	version, _ := f.nodeState(changedNode)
	f.seedExitBinding(changed, "nb-main", "nb-changed", bytes.Repeat([]byte{0x44}, 32))
	bindingBefore := f.bindingText(changed)
	f.nb.InjectRouteState(changed, ObservedExitState{Exists: true, Enabled: true, ExternalID: "nb-changed", PeerID: "peer"})
	f.nb.InjectRouteState(stable, ObservedExitState{Exists: true, Enabled: false, ExternalID: "nb-stable", PeerID: "peer"})
	f.nb.SetPageReadHook(func(string, []ExitPageRef) {
		f.nb.SetPageReadHook(nil)
		f.exec(`UPDATE exits SET publication_intent='disabled', resource_version=resource_version+1 WHERE id=$1`, changed)
	})
	items, err := f.svc.ReconcileNetBird(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ExitID != stable || items[0].NodeID != stableNode {
		t.Fatalf("first results=%+v", items)
	}
	after, drift := f.nodeState(changedNode)
	if after != version || len(drift) != 1 || drift[0].Field != "capacity_report" || fieldText(t, drift[0].Desired) != "stay" {
		t.Fatalf("stale exit written version %s -> %s drift=%+v", version, after, drift)
	}
	if got := f.bindingText(changed); got != bindingBefore {
		t.Fatalf("binding changed\nbefore %s\nafter  %s", bindingBefore, got)
	}
	var intent string
	if err = f.pool.QueryRow(f.ctx, `SELECT publication_intent FROM exits WHERE id=$1`, changed).Scan(&intent); err != nil || intent != string(api.PublicationIntentDisabled) {
		t.Fatalf("intent %q: %v", intent, err)
	}
	items, err = f.svc.ReconcileNetBird(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, item := range items {
		if item.ExitID != changed {
			continue
		}
		found = true
		if len(item.Drift) != 1 || item.Drift[0].Field != driftFieldRoute || fieldText(t, item.Drift[0].Desired) != "disabled" || fieldText(t, item.Drift[0].Observed) != "published" {
			t.Fatalf("second drift=%+v", item.Drift)
		}
	}
	if !found {
		t.Fatalf("second results=%+v", items)
	}
	after, drift = f.nodeState(changedNode)
	versionPlus(t, version, after, 1)
	if len(drift) != 2 || drift[0].Field != "capacity_report" || drift[1].Field != driftFieldRoute || fieldText(t, drift[1].Desired) != "disabled" || fieldText(t, drift[1].Observed) != "published" {
		t.Fatalf("stored drift=%+v", drift)
	}
}

// reconcileBlockedByHook runs ReconcileNetBird and, from its page-read hook,
// runs arm in a second transaction that stays open. The hook returns before
// the reconcile's write transaction starts. This returns once pg_blocking_pids
// shows the reconcile blocked by that transaction. The caller commits holder.
func (f *reconcileFixture) reconcileBlockedByHook(ctx context.Context, arm func(pgx.Tx) error) (items *[]ReconcileItem, done <-chan error, holder pgx.Tx) {
	f.t.Helper()
	out := &[]ReconcileItem{}
	type hold struct {
		tx  pgx.Tx
		pid int32
		err error
	}
	held := make(chan hold, 1)
	f.nb.SetPageReadHook(func(string, []ExitPageRef) {
		f.nb.SetPageReadHook(nil)
		tx, err := f.pool.Begin(ctx)
		if err != nil {
			held <- hold{err: err}
			return
		}
		if err = arm(tx); err != nil {
			held <- hold{tx: tx, err: err}
			return
		}
		held <- hold{tx: tx, pid: int32(tx.Conn().PgConn().PID())}
	})
	errCh := make(chan error, 1)
	go func() {
		got, err := f.svc.ReconcileNetBird(ctx)
		*out = got
		errCh <- err
	}()
	var h hold
	select {
	case h = <-held:
	case err := <-errCh:
		f.t.Fatalf("reconcile finished before the page hook held its lock: %v", err)
	case <-ctx.Done():
		f.t.Fatal(ctx.Err())
	}
	if h.err != nil {
		if h.tx != nil {
			_ = h.tx.Rollback(context.Background())
		}
		f.t.Fatal(h.err)
	}
	holder = h.tx
	released := false
	defer func() {
		if !released && holder != nil {
			_ = holder.Rollback(context.Background())
		}
	}()
	waitForWaiter(f.t, ctx, f.pool, h.pid, errCh)
	released = true
	return out, errCh, holder
}

func routeElements(drift []api.StateDrift) []api.StateDrift {
	var out []api.StateDrift
	for _, el := range drift {
		switch el.Field {
		case driftFieldRoute, driftFieldConfig, driftFieldEvidence:
			out = append(out, el)
		}
	}
	return out
}

func TestReconcileNetBird_SkipsIntentChangedDuringNodeLock(t *testing.T) {
	f := newReconcileFixture(t)
	exitID := uuid.MustParse("00000000-0000-4000-8000-0000000000b8")
	nodeID := f.seedOrdered(exitID, "nb-main", string(api.PublicationIntentUnpublished))
	marker := []api.StateDrift{{
		Field:    "capacity_report",
		Desired:  presence.NewNullableWithValue("stay"),
		Observed: presence.NewNullableWithValue("stay"),
	}}
	f.setDrift(nodeID, marker)
	version, _ := f.nodeState(nodeID)
	f.seedExitBinding(exitID, "nb-main", "nb-lock", bytes.Repeat([]byte{0x44}, 32))
	bindingBefore := f.bindingText(exitID)
	f.nb.InjectRouteState(exitID, ObservedExitState{Exists: true, Enabled: true, ExternalID: "nb-lock", PeerID: "peer"})
	ctx, cancel := context.WithTimeout(f.ctx, 45*time.Second)
	defer cancel()
	items, done, holder := f.reconcileBlockedByHook(ctx, func(tx pgx.Tx) error {
		var locked bool
		if err := tx.QueryRow(ctx, `SELECT true FROM nodes n WHERE n.id = $1 FOR UPDATE OF n`, nodeID).Scan(&locked); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE nodes SET resource_version = resource_version + 1 WHERE id = $1`, nodeID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE exits SET publication_intent = $2, resource_version = resource_version + 1 WHERE id = $1`, exitID, string(api.PublicationIntentDisabled))
		return err
	})
	defer holder.Rollback(context.Background())
	if err := holder.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(*items) != 0 {
		t.Fatalf("first results=%+v", *items)
	}
	after, drift := f.nodeState(nodeID)
	versionPlus(t, version, after, 1)
	if len(drift) != 1 || drift[0].Field != "capacity_report" || len(routeElements(drift)) != 0 {
		t.Fatalf("drift=%+v", drift)
	}
	if got := f.bindingText(exitID); got != bindingBefore {
		t.Fatalf("binding changed\nbefore %s\nafter  %s", bindingBefore, got)
	}
	var intent string
	if err := f.pool.QueryRow(ctx, `SELECT publication_intent FROM exits WHERE id=$1`, exitID).Scan(&intent); err != nil || intent != string(api.PublicationIntentDisabled) {
		t.Fatalf("intent %q: %v", intent, err)
	}
	secondCtx, secondCancel := context.WithTimeout(f.ctx, 30*time.Second)
	defer secondCancel()
	second, err := f.svc.ReconcileNetBird(secondCtx)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, item := range second {
		if item.ExitID != exitID {
			continue
		}
		found = true
		if len(item.Drift) != 1 || item.Drift[0].Field != driftFieldRoute || fieldText(t, item.Drift[0].Desired) != "disabled" || fieldText(t, item.Drift[0].Observed) != "published" {
			t.Fatalf("second drift=%+v", item.Drift)
		}
	}
	if !found {
		t.Fatalf("second results=%+v", second)
	}
	after, drift = f.nodeState(nodeID)
	versionPlus(t, version, after, 2)
	if len(drift) != 2 || drift[0].Field != "capacity_report" || drift[1].Field != driftFieldRoute || fieldText(t, drift[1].Desired) != "disabled" || fieldText(t, drift[1].Observed) != "published" {
		t.Fatalf("stored drift=%+v", drift)
	}
}

func TestReconcileNetBird_SkipsBindingChangedDuringWrite(t *testing.T) {
	f := newReconcileFixture(t)
	exitID := uuid.MustParse("00000000-0000-4000-8000-0000000000b9")
	const deployment = "nb-main"
	nodeID := f.seedOrdered(exitID, deployment, string(api.PublicationIntentPublished))
	f.setDrift(nodeID, []api.StateDrift{{
		Field:    "capacity_report",
		Desired:  presence.NewNullableWithValue("stay"),
		Observed: presence.NewNullableWithValue("stay"),
	}})
	version, _ := f.nodeState(nodeID)
	seedDesired := bytes.Repeat([]byte{0x11}, 32)
	seedObserved := bytes.Repeat([]byte{0x21}, 32)
	seedVerified := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	const seedErr = "seed-last-error"
	f.seedExitBinding(exitID, deployment, "nb-bind", seedDesired)
	f.exec(`UPDATE external_bindings SET observed_hash=$2, last_verified_at=$3, last_error=$4 WHERE resource_type='exit' AND resource_id=$1`, exitID, seedObserved, seedVerified, seedErr)
	heldDesired := bytes.Repeat([]byte{0x31}, 32)
	heldObserved := bytes.Repeat([]byte{0x32}, 32)
	const heldSync = "synced"
	observed := withdrawRoute(exitID)
	observed.ExternalID = "nb-bind"
	observed.ObservedHash = bytes.Repeat([]byte{0x44}, 32)
	f.nb.InjectRouteState(exitID, observed)
	ctx, cancel := context.WithTimeout(f.ctx, 45*time.Second)
	defer cancel()
	items, done, holder := f.reconcileBlockedByHook(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE external_bindings
			SET desired_hash = $2,
			    observed_hash = $3,
			    sync_state = $4
			WHERE resource_type = 'exit'
			  AND resource_id = $1
			  AND provider = $5
			  AND deployment_id = $6`,
			exitID, heldDesired, heldObserved, heldSync, ProviderNetBird, deployment)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errors.New("binding update matched no row")
		}
		return nil
	})
	defer holder.Rollback(context.Background())
	if err := holder.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var (
		desired      []byte
		observedHash []byte
		syncState    string
		verified     *time.Time
		lastErr      *string
	)
	if err := f.pool.QueryRow(ctx, `
		SELECT desired_hash, observed_hash, sync_state, last_verified_at, last_error
		FROM external_bindings
		WHERE resource_type = 'exit' AND resource_id = $1 AND provider = $2 AND deployment_id = $3`,
		exitID, ProviderNetBird, deployment).Scan(&desired, &observedHash, &syncState, &verified, &lastErr); err != nil {
		t.Fatal(err)
	}
	verifiedOK := verified != nil && verified.Equal(seedVerified)
	errOK := lastErr != nil && *lastErr == seedErr
	if !bytes.Equal(desired, heldDesired) || !bytes.Equal(observedHash, heldObserved) || syncState != heldSync || !verifiedOK || !errOK {
		t.Fatalf("binding desired=%x observed=%x sync=%s verified=%v last_error=%v", desired, observedHash, syncState, verified, lastErr)
	}
	after, drift := f.nodeState(nodeID)
	if len(routeElements(drift)) != 0 || len(drift) != 1 || drift[0].Field != "capacity_report" {
		t.Fatalf("drift=%+v", drift)
	}
	versionPlus(t, version, after, 0)
	if len(*items) != 0 {
		t.Fatalf("results=%+v", *items)
	}
}
