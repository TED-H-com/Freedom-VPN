package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	nodelib "github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// NodeJobs is injectable for composition and tests. External calls all go through
// the NetBird adapter; it verifies remote state before returning success.
func NodeJobs(r *httpwire.Runtime, nb *netbird.Client, fetch releases.Fetcher, publicationAdapter ...exits.NetBirdAdapter) Registry {
	jobs := Registry{}
	var publication exits.NetBirdAdapter = httpwire.NewPublicationAdapter(r, nb)
	if len(publicationAdapter) > 1 {
		panic("at most one publication adapter")
	}
	if len(publicationAdapter) == 1 {
		publication = publicationAdapter[0]
	}
	pub := exits.NewService(r.Pool, r.DesiredStateCompiler(), publication)
	jobs.Register(desiredstate.EventNodePublicationChanged, func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		var payload struct {
			OperationID uuid.UUID `json:"operation_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return "", err
		}
		if payload.OperationID == uuid.Nil || event.AggregateType != "node" {
			return "", errors.New("invalid publication event")
		}
		ctx = commandtx.WithClaim(ctx, event)
		if err := commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(pgx.Tx) error { return nil }); err != nil {
			return "", err
		}
		op, err := nodelib.GetOperation(ctx, r.Pool, payload.OperationID)
		if err != nil {
			return "", err
		}
		if op.Resource.Type != api.ResourceTypeNode || op.Resource.Id != event.AggregateID || (op.Kind != api.OperationKindNodePublish && op.Kind != api.OperationKindNodeUnpublish && op.Kind != api.OperationKindNodeDisable) {
			return "", errors.New("publication operation does not match event")
		}
		switch op.Status {
		case api.OperationStatusSucceeded, api.OperationStatusSuperseded, api.OperationStatusPartiallySucceeded:
			return op.OperationId.String(), nil
		}
		op, err = pub.ExecuteOperation(ctx, payload.OperationID)
		if err != nil {
			return "", err
		}
		if op.Status == api.OperationStatusFailed {
			// The executor records failure as an operation, not a Go error.
			// Keep it running while D-040 owns retries; retain the failed step
			// and error evidence rather than acknowledging the outbox event.
			err = commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
				_, e := tx.Exec(ctx, `UPDATE operations SET status='running',resource_version=coalesce(resource_version,0)+1 WHERE id=$1 AND status='failed'`, op.OperationId)
				return e
			})
			if err != nil {
				return "", err
			}
			return "", errors.New("publication execution requires retry")
		}
		if op.Status != api.OperationStatusSucceeded && op.Status != api.OperationStatusSuperseded && op.Status != api.OperationStatusPartiallySucceeded {
			return "", errors.New("publication execution incomplete")
		}
		return op.OperationId.String(), nil
	})
	jobs.Register("release.verify", func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		ctx = commandtx.WithClaim(ctx, event)
		if e := r.Releases.VerifyRelease(ctx, event.AggregateID, fetch); e != nil {
			return "", e
		}
		return "release_verification:" + event.AggregateID.String(), nil
	})
	jobs.Register("node.provision.requested", func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		var p enrollment.ProvisionEvent
		if e := json.Unmarshal(event.Payload, &p); e != nil {
			return "", e
		}
		if p.SchemaVersion != 1 || !p.OneOff || p.UsageLimit != 1 || p.Ephemeral || p.ExpiresInSeconds != 86400 || p.GroupRole != "staging" || p.IncarnationID != event.AggregateID {
			return "", errors.New("invalid provisioning event")
		}
		ctx = commandtx.WithClaim(ctx, event)
		var exists bool
		e := r.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM node_bootstrap_secrets WHERE incarnation_id=$1 AND operation_id=$2)`, p.IncarnationID, p.OperationID).Scan(&exists)
		if e != nil {
			return "", e
		}
		if !exists {
			var active bool
			if e = r.Pool.QueryRow(ctx, `SELECT n.lifecycle_state='registered' AND n.current_incarnation_id=$2 AND i.revoked_at IS NULL FROM nodes n JOIN node_incarnations i ON i.node_id=n.id AND i.id=$2 WHERE n.id=$1`, p.NodeID, p.IncarnationID).Scan(&active); e != nil {
				return "", e
			}
			if !active {
				return "", errors.New("provisioning identity no longer active")
			}
			if e = r.PKI.CheckFence(ctx, pki.NodeIdentity{NodeID: p.NodeID, IncarnationID: p.IncarnationID}); e != nil {
				return "", e
			}
			if nb == nil {
				return "", errors.New("NetBird adapter required")
			}
			key, e := nb.CreateNodeSetupKey(ctx, p.OperationID.String())
			if e != nil {
				return "", e
			}
			if !key.SecretAvailable {
				return "", errors.New("bootstrap response lost; explicit enrollment recovery required")
			}
			if e = r.Enrollment.StoreBootstrap(ctx, pki.NodeIdentity{NodeID: p.NodeID, IncarnationID: p.IncarnationID}, p.OperationID, key.ID, key.Secret.Reveal(), time.Now().Add(time.Duration(r.Config.BootstrapSeconds)*time.Second)); e != nil {
				return "", e
			}
		}
		e = commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			if e := stepSucceeded(ctx, tx, p.OperationID, "netbird_setup", "bootstrap:"+p.IncarnationID.String()); e != nil {
				return e
			}
			return scheduleNodeReconcile(ctx, tx, event, r.Config.ReconcileSeconds)
		})
		return "bootstrap:" + p.IncarnationID.String(), e
	})
	jobs.Register("node.peer.observe", func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		if nb == nil {
			return "", errors.New("NetBird adapter required")
		}
		var payload struct {
			ReportID uuid.UUID `json:"report_id"`
		}
		if e := json.Unmarshal(event.Payload, &payload); e != nil {
			return "", e
		}
		var raw []byte
		var op uuid.UUID
		e := r.Pool.QueryRow(ctx, `SELECT o.report_payload,e.operation_id FROM node_observations o JOIN node_enrollments e ON e.incarnation_id=o.incarnation_id JOIN nodes n ON n.id=o.node_id JOIN node_incarnations i ON i.id=o.incarnation_id WHERE o.incarnation_id=$1 AND o.report_id=$2 AND n.current_incarnation_id=i.id AND n.lifecycle_state='registered' AND i.revoked_at IS NULL AND NOT EXISTS(SELECT 1 FROM node_identity_endpoints x WHERE x.incarnation_id=i.id AND x.mapping_frozen)`, event.AggregateID, payload.ReportID).Scan(&raw, &op)
		if e != nil {
			return "", e
		}
		var report api.ObservationReport
		if e = json.Unmarshal(raw, &report); e != nil {
			return "", e
		}
		peerID, e := report.Netbird.PeerId.Get()
		if e != nil {
			return "", e
		}
		peer, e := nb.EnsureNodePeer(ctx, netbird.NodePeer{NodeID: report.NodeId.String(), IncarnationID: report.NodeIncarnationId.String(), PeerID: peerID})
		if e != nil {
			return "", e
		}
		ctx = commandtx.WithClaim(ctx, event)
		e = commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			var active bool
			if e := tx.QueryRow(ctx, `SELECT current_incarnation_id=$2 AND lifecycle_state='registered' AND NOT EXISTS(SELECT 1 FROM node_identity_endpoints x WHERE x.incarnation_id=$2 AND x.mapping_frozen) FROM nodes WHERE id=$1 FOR UPDATE`, report.NodeId, report.NodeIncarnationId).Scan(&active); e != nil {
				return e
			}
			if !active {
				return errors.New("identity changed during read-back")
			}
			if e := r.PKI.CheckFence(ctx, pki.NodeIdentity{NodeID: report.NodeId, IncarnationID: report.NodeIncarnationId}); e != nil {
				return e
			}
			if _, e := tx.Exec(ctx, `UPDATE node_incarnations SET deployment_id=$2,peer_id=$3 WHERE id=$1 AND revoked_at IS NULL`, report.NodeIncarnationId, r.Config.NetBirdDeployment, peer.ID); e != nil {
				return e
			}
			if e := stepSucceeded(ctx, tx, op, "peer_readback", "observation:"+payload.ReportID.String()); e != nil {
				return e
			}
			var ready bool
			if e := tx.QueryRow(ctx, `SELECT safety_status='passed' AND applied_config_generation>0 AND applied_config_generation>=desired_config_generation AND last_observed_at=$2 FROM nodes WHERE id=$1`, report.NodeId, report.ObservedAt).Scan(&ready); e != nil {
				return e
			}
			if ready && report.ProvisioningState == api.ProvisioningStateReady {
				if _, e := tx.Exec(ctx, `UPDATE nodes SET provisioning_state='ready',resource_version=resource_version+1 WHERE id=$1`, report.NodeId); e != nil {
					return e
				}
				return stepSucceeded(ctx, tx, op, "node_policy", "observation:"+payload.ReportID.String())
			}
			return nil
		})
		return "observation:" + payload.ReportID.String(), e
	})
	// Compatibility: enrollment.RevokeNode still emits both withdrawal events.
	// Remove only when that producer and its peer/key revocation are migrated.
	withdraw := func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		var p struct {
			NodeID        uuid.UUID  `json:"node_id"`
			IncarnationID *uuid.UUID `json:"incarnation_id"`
			OperationID   uuid.UUID  `json:"operation_id"`
		}
		if e := json.Unmarshal(event.Payload, &p); e != nil {
			return "", e
		}
		if p.NodeID == uuid.Nil || p.OperationID == uuid.Nil || nb == nil {
			return "", errors.New("invalid withdrawal event or adapter")
		}
		var exit uuid.UUID
		if e := r.Pool.QueryRow(ctx, `SELECT id FROM exits WHERE node_id=$1`, p.NodeID).Scan(&exit); e != nil {
			return "", e
		}
		if e := nb.WithdrawExit(ctx, exit.String()); e != nil {
			return "", e
		}
		step := "route_withdrawal"
		if event.EventType == "node.peer.withdraw" {
			step = "peer_withdrawal"
			keyRows, e := r.Pool.Query(ctx, `SELECT e.operation_id FROM node_enrollments e JOIN node_incarnations i ON i.id=e.incarnation_id WHERE e.node_id=$1 AND i.revoked_at IS NOT NULL`, p.NodeID)
			if e != nil {
				return "", e
			}
			keys, e := pgx.CollectRows(keyRows, pgx.RowTo[uuid.UUID])
			if e != nil {
				return "", e
			}
			for _, key := range keys {
				if e = nb.RevokeNodeSetupKey(ctx, key.String()); e != nil {
					return "", e
				}
			}
			rows, e := r.Pool.Query(ctx, `SELECT id,peer_id FROM node_incarnations WHERE node_id=$1 AND revoked_at IS NOT NULL`, p.NodeID)
			if e != nil {
				return "", e
			}
			type peer struct {
				id     uuid.UUID
				remote *string
			}
			var peers []peer
			for rows.Next() {
				var p peer
				if e = rows.Scan(&p.id, &p.remote); e != nil {
					rows.Close()
					return "", e
				}
				peers = append(peers, p)
			}
			e = rows.Err()
			rows.Close()
			if e != nil {
				return "", e
			}
			for _, peer := range peers {
				if peer.remote != nil {
					binding := "node:" + p.NodeID.String() + ":" + peer.id.String()
					if _, e = nb.EnsureNodePeer(ctx, netbird.NodePeer{NodeID: p.NodeID.String(), IncarnationID: peer.id.String(), PeerID: *peer.remote}); e != nil && !netbird.IsKind(e, netbird.NotFound) {
						return "", e
					}
					if e = nb.DeletePeer(ctx, binding); e != nil {
						return "", e
					}
				}
			}
		}
		ctx = commandtx.WithClaim(ctx, event)
		e := commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			if _, e := tx.Exec(ctx, `UPDATE exits SET publication_observed='withdrawn',resource_version=resource_version+1 WHERE id=$1`, exit); e != nil {
				return e
			}
			if _, e := tx.Exec(ctx, `UPDATE outbox_events SET delivery_receipt=$2 WHERE id=$1`, event.ID, "withdrawal:"+event.ID.String()); e != nil {
				return e
			}
			return stepSucceeded(ctx, tx, p.OperationID, step, "withdrawal:"+event.ID.String())
		})
		return "withdrawal:" + event.ID.String(), e
	}
	jobs.Register("node.routes.withdraw", withdraw)
	jobs.Register("node.peer.withdraw", withdraw)
	reconcile := func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		ctx = commandtx.WithClaim(ctx, event)
		if e := r.Fleet.Reconcile(ctx); e != nil {
			return "", e
		}
		if e := r.Enrollment.ExpireBootstrap(ctx); e != nil {
			return "", e
		}
		e := commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error { return scheduleNodeReconcile(ctx, tx, event, r.Config.ReconcileSeconds) })
		return "reconciliation:" + event.ID.String(), e
	}
	for _, name := range []string{"node.reconcile", "fleet.update_created", "fleet.first_sync_created", "fleet.update_retry", "fleet.release_support_changed", "node.updated"} {
		jobs.Register(name, reconcile)
	}
	return jobs
}
func stepSucceeded(ctx context.Context, tx pgx.Tx, op uuid.UUID, step, ref string) error {
	var status string
	if e := tx.QueryRow(ctx, `SELECT steps->$2->>'status' FROM operations WHERE id=$1`, op, step).Scan(&status); e != nil {
		return e
	}
	if status == "succeeded" {
		return nil
	}
	if status == "failed" {
		if e := operations.Transition(ctx, tx, op, step, api.OperationStepStatusRunning, nil, nil, nil); e != nil {
			return e
		}
	}
	return operations.Transition(ctx, tx, op, step, api.OperationStepStatusSucceeded, &operations.Evidence{Reference: ref, ObservedAt: time.Now().UTC()}, nil, nil)
}
func scheduleNodeReconcile(ctx context.Context, tx pgx.Tx, event db.ClaimedEvent, seconds int) error {
	if seconds <= 0 {
		return errors.New("node reconciliation period required")
	}
	return enqueueNodeReconcile(ctx, tx, event.ID, time.Now().Add(time.Duration(seconds)*time.Second))
}
func enqueueNodeReconcile(ctx context.Context, tx pgx.Tx, exclude uuid.UUID, available time.Time) error {
	var exists bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM outbox_events WHERE event_type='node.reconcile' AND delivered_at IS NULL AND dead_lettered_at IS NULL AND id<>$1)`, exclude).Scan(&exists); e != nil {
		return e
	}
	if exists {
		return nil
	}
	raw, _ := canonjson.Marshal(map[string]int{"schema_version": 1})
	return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: uuid.MustParse(releases.FleetID), AggregateType: "fleet_config", AggregateVersion: "1", EventType: "node.reconcile", SchemaVersion: 1, Payload: raw, AvailableAt: available})
}

// nodes builds once from the process-owned pool/configuration (D-026). The
// dispatcher is the only outbox claimant; the runner only seeds durable work.
func nodes() Domain {
	return Domain{Name: "nodes", Build: func(ctx context.Context, d Dependencies) (Built, error) {
		cfg, err := config.LoadNodes(d.Config.NodesConfigFile)
		if err != nil {
			return Built{}, &config.SettingError{Name: "nodes_config_file", Err: err}
		}
		runtime, err := httpwire.Open(ctx, d.Pool, cfg, d.Logger)
		if err != nil {
			return Built{}, &config.SettingError{Name: "nodes_config_file", Err: err}
		}
		adapter, err := netbird.New(netbird.Config{BaseURL: cfg.NetBirdURL, DeploymentID: cfg.NetBirdDeployment, Token: netbird.TokenFile(cfg.NetBirdTokenFile), Bindings: nodelib.NetBirdBindings{Pool: d.Pool}, Timeout: time.Duration(cfg.FetchSeconds) * time.Second})
		if err != nil {
			return Built{}, &config.SettingError{Name: "nodes_config_file", Err: err}
		}
		fetch, err := releases.NewHTTPSFetcher(runtime.Releases.Validator, time.Duration(cfg.FetchSeconds)*time.Second)
		if err != nil {
			return Built{}, &config.SettingError{Name: "nodes_config_file", Err: err}
		}
		return Built{Jobs: NodeJobs(runtime, adapter, fetch), Runners: []func(context.Context) error{func(ctx context.Context) error { return RunNodeSchedule(ctx, runtime) }}}, nil
	}}
}

// RunNodeSchedule seeds immediately and repairs a missing reconciliation chain.
// Fleet locking coalesces concurrent workers and the chain's delayed successors.
// A returned error is supervised; cancellation never leaves a child goroutine.
func RunNodeSchedule(ctx context.Context, runtime *httpwire.Runtime) error {
	if runtime == nil || runtime.Pool == nil || runtime.Config.ReconcileSeconds < 1 {
		return errors.New("node schedule configuration required")
	}
	seed := func() error {
		return commandtx.InTx(ctx, runtime.Pool, db.TxOptions{}, func(tx pgx.Tx) error { return enqueueNodeReconcile(ctx, tx, uuid.Nil, time.Now()) })
	}
	ticker := time.NewTicker(time.Duration(runtime.Config.ReconcileSeconds) * time.Second)
	defer ticker.Stop()
	for {
		if err := seed(); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
