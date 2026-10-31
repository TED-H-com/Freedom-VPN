package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type membersContextKey struct{}

// WithMembers supplies immutable dependencies to the shared dispatcher. The
// dispatcher still owns global claims, fencing, backoff and dead-lettering.
func WithMembers(ctx context.Context, s *membersruntime.Services) context.Context {
	return context.WithValue(ctx, membersContextKey{}, s)
}
func members() Domain {
	return Domain{Name: "members", Build: func(_ context.Context, d Dependencies) (Built, error) {
		cfg, e := config.LoadMembersForIssuer(d.Config.MembersConfigFile, d.Config.OIDCIssuer)
		if e != nil {
			return Built{}, e
		}
		identity, e := devices.NewIdentityDiscovery(d.Config.OIDCIssuer, d.Metrics, d.Logger)
		if e != nil {
			return Built{}, &config.SettingError{Name: "oidc_issuer", Err: e}
		}
		// The nodes domain supplies its configured compiler through WithCompiler
		// before dispatch (D-026); building members does not depend on its order.
		services, e := membersruntime.New(d.Pool, cfg, identity, nil)
		if e != nil {
			return Built{}, &config.SettingError{Name: "members_config_file", Err: e}
		}
		if services.HTTP.Devices != nil {
			services.HTTP.Devices.Logger = d.Logger
			if e = services.HTTP.Devices.RegisterRestoreFences(d.Metrics); e != nil {
				return Built{}, e
			}
		}
		return Built{Jobs: MemberJobs(), Context: func(ctx context.Context) context.Context { return WithMembers(ctx, services) }, Runners: []func(context.Context) error{func(ctx context.Context) error { return RunMemberSchedule(ctx, services) }}}, nil
	}}
}

// MemberJobs returns a fresh registry; Open merges it and rejects overlap.
// D-031 makes checkout synchronous. D-038 leaves legacy session events in
// disposable development databases to the observable unknown-event path.
func MemberJobs() Registry {
	r := Registry{}
	for _, name := range []string{"member.entitlement_changed", "member.grant_revoked", "member.suspended", "device.authorized", "device.revoked", "netbird.device_sync", "billing.inbox_received", "billing.reconcile", "members.reconcile", "devices.discover"} {
		r.Register(name, memberJob)
	}
	return r
}
func memberJob(ctx context.Context, event db.ClaimedEvent) (string, error) {
	s, _ := ctx.Value(membersContextKey{}).(*membersruntime.Services)
	if s == nil || s.HTTP.Members == nil || s.HTTP.Devices == nil || s.HTTP.Billing == nil {
		return "", errors.New("member worker dependencies required")
	}
	if event.SchemaVersion != 1 {
		return "", errors.New("unsupported member event schema")
	}
	pool := s.HTTP.Members.Pool
	switch event.EventType {
	case "billing.inbox_received":
		var p struct {
			ID uuid.UUID `json:"inbox_id"`
		}
		if json.Unmarshal(event.Payload, &p) != nil || p.ID != event.AggregateID {
			return "", errors.New("invalid inbox event")
		}
		if e := s.HTTP.Billing.ProcessInbox(ctx, p.ID); e != nil {
			return "", e
		}
		return "billing_inbox:" + p.ID.String(), nil
	case "netbird.device_sync", "device.authorized", "device.revoked":
		var p devices.Event
		if json.Unmarshal(event.Payload, &p) != nil || p.DeviceID != event.AggregateID || p.MemberID == uuid.Nil || p.DeviceVersion.String() != event.AggregateVersion {
			return "", errors.New("invalid device event")
		}
		var matches bool
		if e := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM member_devices WHERE id=$1 AND member_id=$2 AND deployment_id=$3 AND peer_id=$4 AND resource_version>=$5::numeric)`, p.DeviceID, p.MemberID, p.DeploymentID, p.PeerID, p.DeviceVersion.String()).Scan(&matches); e != nil {
			return "", e
		}
		if !matches {
			return "", errors.New("device event mismatch")
		}
		syncErr := s.HTTP.Devices.SyncDevice(ctx, p.DeviceID)
		if e := errors.Join(syncErr, recompileFencedMember(ctx, s, p.MemberID)); e != nil {
			return "", e
		}
		if event.EventType == "netbird.device_sync" {
			return "device_observation:" + p.DeviceID.String(), nil
		}
	case "member.entitlement_changed", "member.grant_revoked", "member.suspended":
		var p entitlement.Projection
		if json.Unmarshal(event.Payload, &p) != nil || p.MemberID != event.AggregateID || p.Revision.String() != event.AggregateVersion {
			return "", errors.New("invalid entitlement event")
		}
		var current bool
		if e := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM entitlements WHERE member_id=$1 AND revision>=$2::numeric)`, p.MemberID, p.Revision.String()).Scan(&current); e != nil {
			return "", e
		}
		if !current {
			return "", errors.New("future entitlement event")
		}
		// Grant withdrawal uses this same path, including when a future grant's
		// removal leaves the projection unchanged. SyncMember observes the
		// member.grant_revoke netbird step only after every device read-back;
		// fence fanout must finish before the job's policy receipt is acknowledged.
		syncErr := s.HTTP.Devices.SyncMember(ctx, p.MemberID)
		if e := errors.Join(syncErr, recompileFencedMember(ctx, s, p.MemberID)); e != nil {
			return "", e
		}
	case "billing.reconcile":
		var p scheduledPage
		if json.Unmarshal(event.Payload, &p) != nil || p.Run == uuid.Nil {
			return "", errors.New("invalid reconciliation event")
		}
		after, n, e := s.HTTP.Billing.Reconcile(ctx, p.Run, p.After, 100)
		if e != nil {
			return "", e
		}
		if n == 100 {
			p.After = after
			if e = enqueueMemberWork(ctx, s, event.EventType, p); e != nil {
				return "", e
			}
		}
		return "outbox_event:" + event.ID.String(), nil
	case "members.reconcile":
		var p scheduledPage
		if json.Unmarshal(event.Payload, &p) != nil || p.Run == uuid.Nil {
			return "", errors.New("invalid member sweep")
		}
		if _, e := s.HTTP.Members.Entitlements.Sweep(ctx, pool, 100); e != nil {
			return "", e
		}
		rows, e := pool.Query(ctx, `SELECT id FROM members WHERE id>$1 ORDER BY id LIMIT 100`, p.After)
		if e != nil {
			return "", e
		}
		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return "", e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return "", e
		}
		for _, id := range ids {
			// Include never-projected first-login accounts and time-only changes.
			if e = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error { _, e := s.HTTP.Members.Entitlements.Recompute(ctx, tx, id); return e }); e != nil {
				return "", e
			}
			syncErr := s.HTTP.Devices.SyncMember(ctx, id)
			if e = errors.Join(syncErr, recompileFencedMember(ctx, s, id)); e != nil {
				return "", e
			}
		}
		if len(ids) == 100 {
			p.After = ids[len(ids)-1]
			if e = enqueueMemberWork(ctx, s, event.EventType, p); e != nil {
				return "", e
			}
		}
		return "outbox_event:" + event.ID.String(), nil
	case "devices.discover":
		if s.Adapter == nil {
			return "", errors.New("NetBird discovery adapter required")
		}
		if e := s.Adapter.Discover(ctx, s.HTTP.Devices); e != nil {
			return "", e
		}
		return "outbox_event:" + event.ID.String(), nil
	default:
		return "", errors.New("unowned member event")
	}
	// Fan out policy work before acknowledging member/device intent. Compilation
	// is persisted desired state; only node ACKs may complete the nodes step.
	compiler := s.Compiler
	if compiler == nil {
		compiler = membersruntime.CompilerFromContext(ctx)
	}
	if compiler == nil {
		return "", errors.New("member desired-state compiler required")
	}
	policyEvent := event
	if event.EventType == "member.grant_revoked" {
		// The projection did not change, but the new withdrawal receipt still
		// needs the ordinary policy/read-back path for this member and its groups.
		policyEvent.EventType = "member.entitlement_changed"
	}
	compiled, e := compiler.RecompileBatch(ctx, []db.ClaimedEvent{policyEvent})
	if e != nil {
		return "", e
	}
	raw, e := canonjson.Marshal(map[string]any{"event_id": event.ID, "compiled_nodes": len(compiled)})
	if e != nil {
		return "", e
	}
	// One immutable receipt audit per outbox ID, including empty affected sets.
	resource, e := entitlement.ResourceType(event.AggregateType)
	if e != nil {
		return "", e
	}
	e = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var exists bool
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,121))`, event.ID.String()); e != nil {
			return e
		}
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_events WHERE id=$1)`, event.ID).Scan(&exists); e != nil {
			return e
		}
		if exists {
			return nil
		}
		return db.AppendAudit(ctx, tx, db.AuditEvent{ID: event.ID, Actor: "00000000-0000-0000-0000-000000000003", ActorType: "system", Action: "member.policy_compiled", Resource: string(resource) + ":" + event.AggregateID.String(), SchemaVersion: 1, Before: []byte(`{}`), After: raw, Result: "succeeded"})
	})
	if e != nil {
		return "", e
	}
	return "audit_event:" + event.ID.String(), nil
}

// recompileFencedMember uses the same fanout as device revocation before the
// triggering job is acknowledged. A sync can commit a fence and then fail, so
// callers run this even on errors. The persisted fence and signed bindings also
// recover work after a crash; no transient "new fence" flag is required.
func recompileFencedMember(ctx context.Context, s *membersruntime.Services, member uuid.UUID) error {
	rows, err := s.HTTP.Members.Pool.Query(ctx, `
		SELECT e.node_id FROM desired_state_envelopes e
		JOIN nodes n ON n.id=e.node_id AND n.current_incarnation_id=e.incarnation_id
		WHERE EXISTS (
			SELECT 1 FROM jsonb_array_elements(e.envelope->'signed'->'device_bindings') b
			JOIN device_authorization_fences f ON f.device_id::text=b->>'device_id'
			JOIN member_devices d ON d.id=f.device_id
			WHERE d.member_id=$1
		) ORDER BY e.node_id`, member)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil || len(ids) == 0 {
		return err
	}
	compiler := s.Compiler
	if compiler == nil {
		compiler = membersruntime.CompilerFromContext(ctx)
	}
	if compiler == nil {
		return errors.New("member desired-state compiler required")
	}
	// This is an in-process invalidation hint, not a device status transition or
	// a new revoke operation. Explicit node IDs include old groups still bound
	// in signed state and avoid recompiling exits that already dropped the peer.
	payload, err := canonjson.Marshal(map[string]any{"node_ids": ids})
	if err != nil {
		return err
	}
	_, err = compiler.RecompileBatch(ctx, []db.ClaimedEvent{{OutboxEvent: db.OutboxEvent{EventType: desiredstate.EventDeviceRevoked, Payload: payload}}})
	return err
}

type scheduledPage struct {
	Run   uuid.UUID `json:"run_id"`
	After uuid.UUID `json:"after"`
}

func enqueueMemberWork(ctx context.Context, s *membersruntime.Services, name string, p scheduledPage) error {
	raw, e := canonjson.Marshal(p)
	if e != nil {
		return e
	}
	key, e := canonjson.Marshal(map[string]any{"event": name, "page": p})
	if e != nil {
		return e
	}
	hash, e := canonjson.Hash(key)
	if e != nil {
		return e
	}
	id, _ := uuid.FromBytes(hash[:16])
	id[6] = id[6]&15 | 0x80
	id[8] = id[8]&63 | 0x80
	return db.InTx(ctx, s.HTTP.Members.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,122))`, id.String()); e != nil {
			return e
		}
		var exists bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM outbox_events WHERE id=$1)`, id).Scan(&exists); e != nil {
			return e
		}
		if exists {
			return nil
		}
		return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{ID: id, AggregateID: p.Run, AggregateType: "member_maintenance", AggregateVersion: "1", EventType: name, SchemaVersion: 1, Payload: raw})
	})
}

// RunMemberSchedule only enqueues owned work; it never claims another area's
// outbox rows. The members domain registers this runner with Open. Failed
// enqueue returns to the process supervisor, not a silent loop.
func RunMemberSchedule(ctx context.Context, s *membersruntime.Services) error {
	if s == nil || s.HTTP.Members == nil || s.ReconcileInterval < time.Second || s.DeviceInterval < time.Second {
		return errors.New("member schedule configuration required")
	}
	schedule := func(name string, interval time.Duration) error {
		raw, e := canonjson.Marshal([]string{name, time.Now().UTC().Truncate(interval).Format(time.RFC3339)})
		if e != nil {
			return e
		}
		hash, e := canonjson.Hash(raw)
		if e != nil {
			return e
		}
		id, _ := uuid.FromBytes(hash[:16])
		id[6] = id[6]&15 | 0x80
		id[8] = id[8]&63 | 0x80
		return enqueueMemberWork(ctx, s, name, scheduledPage{Run: id})
	}
	for _, v := range []struct {
		name     string
		interval time.Duration
	}{{"billing.reconcile", s.ReconcileInterval}, {"members.reconcile", s.DeviceInterval}, {"devices.discover", s.DeviceInterval}} {
		if e := schedule(v.name, v.interval); e != nil {
			return e
		}
	}
	billingTick := time.NewTicker(s.ReconcileInterval)
	defer billingTick.Stop()
	devicesTick := time.NewTicker(s.DeviceInterval)
	defer devicesTick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-billingTick.C:
			if e := schedule("billing.reconcile", s.ReconcileInterval); e != nil {
				return e
			}
		case <-devicesTick.C:
			for _, name := range []string{"members.reconcile", "devices.discover"} {
				if e := schedule(name, s.DeviceInterval); e != nil {
					return fmt.Errorf("schedule member work: %w", e)
				}
			}
		}
	}
}
