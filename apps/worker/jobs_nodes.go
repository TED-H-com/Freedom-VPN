package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
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
func NodeJobs(r *httpwire.Runtime, nb *netbird.Client, fetch releases.Fetcher, attempts int, publicationAdapter ...exits.NetBirdAdapter) Registry {
	return nodeJobs(r, nb, fetch, attempts, r.DesiredStateCompiler(), publicationAdapter...)
}

func nodeJobs(r *httpwire.Runtime, nb *netbird.Client, fetch releases.Fetcher, attempts int, compiler *desiredstate.Compiler, publicationAdapter ...exits.NetBirdAdapter) Registry {
	jobs := Registry{}
	jobs.Register(nodelib.EventGenerationHoldCheck, func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		if err := nodelib.ValidateGenerationHoldEvent(event); err != nil {
			return "", err
		}
		err := r.Nodes.ReconcileGenerationHolds(commandtx.WithClaim(ctx, event), event.AggregateID)
		return "generation-hold:" + event.AggregateID.String(), err
	})
	jobs.Register("node.resolver_documents.reconcile", func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		ctx = commandtx.WithClaim(ctx, event)
		if err := r.ReconcileResolverDocuments(ctx); err != nil {
			return "", err
		}
		err := commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			return enqueueScheduledNodeJob(ctx, tx, event.EventType, event.ID, time.Now().Add(resolverReconcilePeriod(r.Config.ReconcileSeconds)))
		})
		return "resolver-reconciliation:" + event.ID.String(), err
	})
	jobs.Register("node.peer.discover", nodePeerDiscovery(r, nb, attempts))
	var publication exits.NetBirdAdapter = httpwire.NewPublicationAdapter(r, nb)
	if len(publicationAdapter) > 1 {
		panic("at most one publication adapter")
	}
	if len(publicationAdapter) == 1 {
		publication = publicationAdapter[0]
	}
	pub := exits.NewService(r.Pool, compiler, publication)
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
		revoking := op.Kind == api.OperationKindNodeRevoke || op.Kind == api.OperationKindNodeRetire
		if op.Resource.Type != api.ResourceTypeNode || op.Resource.Id != event.AggregateID || (!revoking && op.Kind != api.OperationKindNodePublish && op.Kind != api.OperationKindNodeUnpublish && op.Kind != api.OperationKindNodeDisable) {
			return "", errors.New("publication operation does not match event")
		}
		switch op.Status {
		case api.OperationStatusSucceeded, api.OperationStatusSuperseded, api.OperationStatusPartiallySucceeded:
			return op.OperationId.String(), nil
		}
		executor := pub
		if revoking {
			// Cleanup runs inside the withdrawal adapter before the executor can
			// finalize this operation, retaining the same claim and retry budget.
			executor = exits.NewService(r.Pool, compiler, httpwire.NewRevocationAdapter(r, nb, publication, op))
		}
		op, err = executor.ExecuteOperation(ctx, payload.OperationID)
		if err != nil {
			return "", err
		}
		if op.Status == api.OperationStatusFailed {
			// The executor records failure as an operation, not a Go error.
			// §14.5 / D-040: running means another retry remains. At or past
			// the dispatcher's budget, preserve failed status and its evidence
			// so dead-lettering cannot leave an operation claiming progress.
			if event.Attempt < attempts {
				err = commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
					_, e := tx.Exec(ctx, `UPDATE operations SET status='running',resource_version=coalesce(resource_version,0)+1 WHERE id=$1 AND status='failed'`, op.OperationId)
					return e
				})
				if err != nil {
					return "", err
				}
			}
			return "", errors.New("publication execution failed")
		}
		if op.Status != api.OperationStatusSucceeded && op.Status != api.OperationStatusSuperseded && op.Status != api.OperationStatusPartiallySucceeded {
			if event.Attempt >= attempts {
				if _, err := executor.ExhaustGatedPublish(ctx, op.OperationId, op.Resource.Id); err != nil {
					return "", err
				}
			}
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
			if e = r.Pool.QueryRow(ctx, `SELECT n.lifecycle_state='registered' AND n.current_incarnation_id=$2 AND i.revoked_at IS NULL AND NOT EXISTS(SELECT 1 FROM node_identity_endpoints x WHERE x.incarnation_id=i.id AND x.mapping_frozen) FROM nodes n JOIN node_incarnations i ON i.node_id=n.id AND i.id=$2 WHERE n.id=$1`, p.NodeID, p.IncarnationID).Scan(&active); e != nil {
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
			if err := enqueueNodePeerDiscovery(ctx, tx, p.NodeID, p.IncarnationID, p.OperationID); err != nil {
				return err
			}
			return scheduleNodeReconcile(ctx, tx, event, r.Config.ReconcileSeconds)
		})
		return "bootstrap:" + p.IncarnationID.String(), e
	})
	jobs.Register("node.peer.observe", func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		var payload struct {
			ReportID uuid.UUID `json:"report_id"`
		}
		if e := json.Unmarshal(event.Payload, &payload); e != nil {
			return "", e
		}
		var raw []byte
		var op uuid.UUID
		var boundPeer *string
		e := r.Pool.QueryRow(ctx, `SELECT o.report_payload,e.operation_id,i.peer_id FROM node_observations o JOIN node_enrollments e ON e.incarnation_id=o.incarnation_id JOIN nodes n ON n.id=o.node_id JOIN node_incarnations i ON i.id=o.incarnation_id WHERE o.incarnation_id=$1 AND o.report_id=$2 AND n.current_incarnation_id=i.id AND n.lifecycle_state='registered' AND i.revoked_at IS NULL AND NOT EXISTS(SELECT 1 FROM node_identity_endpoints x WHERE x.incarnation_id=i.id AND x.mapping_frozen)`, event.AggregateID, payload.ReportID).Scan(&raw, &op, &boundPeer)
		if errors.Is(e, pgx.ErrNoRows) {
			// Revoked/frozen incarnations cannot resume discovery or binding.
			return "observation:" + payload.ReportID.String(), nil
		}
		if e != nil {
			return "", e
		}
		var report api.ObservationReport
		if e = json.Unmarshal(raw, &report); e != nil {
			return "", e
		}
		peerID, peerErr := report.Netbird.PeerId.Get()
		if peerErr != nil && boundPeer != nil {
			peerID, peerErr = *boundPeer, nil
		}
		var peer netbird.Peer
		if peerErr == nil {
			if nb == nil {
				return "", errors.New("NetBird adapter required")
			}
			fencedCtx := netbird.WithDiscoveryFence(ctx, func(ctx context.Context) error {
				return commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
					return discoveryActive(ctx, tx, r, nodeDiscovery{NodeID: report.NodeId, IncarnationID: report.NodeIncarnationId, OperationID: op})
				})
			})
			peer, e = nb.EnsureNodePeer(fencedCtx, netbird.NodePeer{NodeID: report.NodeId.String(), IncarnationID: report.NodeIncarnationId.String(), PeerID: peerID})
			if errors.Is(e, errDiscoveryInactive) {
				return "observation:" + payload.ReportID.String(), nil
			}
			if e != nil {
				return "", e
			}
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
			// A baseline ACK is independent of management peer discovery (§6.1).
			// Only confirmed peer read-back can bind identity or advance that step.
			if peerErr == nil {
				if _, e := tx.Exec(ctx, `UPDATE node_incarnations SET deployment_id=$2,peer_id=$3 WHERE id=$1 AND revoked_at IS NULL`, report.NodeIncarnationId, r.Config.NetBirdDeployment, peer.ID); e != nil {
					return e
				}
				if e := stepSucceeded(ctx, tx, op, "peer_readback", "observation:"+payload.ReportID.String()); e != nil {
					return e
				}
			}
			if peerErr != nil {
				if err := enqueueNodePeerDiscovery(ctx, tx, report.NodeId, report.NodeIncarnationId, op); err != nil {
					return err
				}
			}
			var ready bool
			applied, _ := report.AppliedConfigGeneration.Get()
			if e := tx.QueryRow(ctx, `SELECT safety_status='passed' AND applied_config_generation>=desired_config_generation AND $3::numeric>0 AND $3::numeric>=desired_config_generation AND last_observed_at=$2 FROM nodes WHERE id=$1`, report.NodeId, report.ObservedAt, applied.String()).Scan(&ready); e != nil {
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
	reconcile := func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		ctx = commandtx.WithClaim(ctx, event)
		if e := r.Nodes.ReconcileGenerationHolds(ctx, uuid.Nil); e != nil {
			return "", e
		}
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
	return enqueueScheduledNodeJob(ctx, tx, "node.reconcile", exclude, available)
}
func enqueueScheduledNodeJob(ctx context.Context, tx pgx.Tx, eventType string, exclude uuid.UUID, available time.Time) error {
	var exists bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM outbox_events WHERE event_type=$2 AND delivered_at IS NULL AND dead_lettered_at IS NULL AND id<>$1)`, exclude, eventType).Scan(&exists); e != nil {
		return e
	}
	if exists {
		return nil
	}
	raw, _ := canonjson.Marshal(map[string]int{"schema_version": 1})
	return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: uuid.MustParse(releases.FleetID), AggregateType: "fleet_config", AggregateVersion: "1", EventType: eventType, SchemaVersion: 1, Payload: raw, AvailableAt: available})
}

func resolverReconcilePeriod(seconds int) time.Duration {
	return time.Duration(max(1, min(seconds, 60))) * time.Second
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
		compiler := runtime.DesiredStateCompiler()
		return Built{
			Jobs: nodeJobs(runtime, adapter, fetch, d.Config.WorkerAttempts, compiler),
			// D-026: all decorators run before member/device jobs, independent
			// of domain build order. Share the configured signer, TTL and DNS.
			Context: func(ctx context.Context) context.Context { return membersruntime.WithCompiler(ctx, compiler) },
			Runners: []func(context.Context) error{func(ctx context.Context) error { return RunNodeSchedule(ctx, runtime) }},
		}, nil
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
		return commandtx.InTx(ctx, runtime.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			if err := enqueueNodeReconcile(ctx, tx, uuid.Nil, time.Now()); err != nil {
				return err
			}
			return enqueueScheduledNodeJob(ctx, tx, "node.resolver_documents.reconcile", uuid.Nil, time.Now())
		})
	}
	ticker := time.NewTicker(resolverReconcilePeriod(runtime.Config.ReconcileSeconds))
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

type nodeDiscovery struct {
	NodeID        uuid.UUID `json:"node_id"`
	IncarnationID uuid.UUID `json:"incarnation_id"`
	OperationID   uuid.UUID `json:"operation_id"`
}

// Callers hold the fleet lock. A heartbeat cannot reset an exhausted retry
// budget: one durable discovery event belongs to each enrollment attempt.
func enqueueNodePeerDiscovery(ctx context.Context, tx pgx.Tx, node, incarnation, op uuid.UUID) error {
	raw, err := canonjson.Marshal(nodeDiscovery{node, incarnation, op})
	if err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM outbox_events WHERE event_type='node.peer.discover' AND payload->>'incarnation_id'=$1)`, incarnation.String()).Scan(&exists); err != nil || exists {
		return err
	}
	return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateType: "node", AggregateID: node, AggregateVersion: "1", EventType: "node.peer.discover", SchemaVersion: 1, Payload: raw})
}

var errDiscoveryInactive = errors.New("node discovery identity inactive")

func discoveryActive(ctx context.Context, tx pgx.Tx, r *httpwire.Runtime, p nodeDiscovery) error {
	var active bool
	err := tx.QueryRow(ctx, `SELECT n.current_incarnation_id=i.id AND n.lifecycle_state='registered' AND i.revoked_at IS NULL AND e.state<>'revoked' AND NOT EXISTS(SELECT 1 FROM node_identity_endpoints x WHERE x.incarnation_id=i.id AND x.mapping_frozen)
		FROM nodes n JOIN node_incarnations i ON i.node_id=n.id AND i.id=$2 JOIN node_enrollments e ON e.incarnation_id=i.id AND e.operation_id=$3 WHERE n.id=$1 FOR UPDATE OF n,i,e`, p.NodeID, p.IncarnationID, p.OperationID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !active) {
		return errDiscoveryInactive
	}
	if err != nil {
		return err
	}
	return r.PKI.CheckFence(ctx, pki.NodeIdentity{NodeID: p.NodeID, IncarnationID: p.IncarnationID})
}

func nodePeerDiscovery(r *httpwire.Runtime, nb *netbird.Client, attempts int) Handler {
	return func(ctx context.Context, event db.ClaimedEvent) (string, error) {
		var p nodeDiscovery
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return "", err
		}
		if p.NodeID == uuid.Nil || p.OperationID == uuid.Nil || p.IncarnationID == uuid.Nil || event.AggregateID != p.NodeID || event.AggregateType != "node" {
			return "", errors.New("invalid peer discovery event")
		}
		ctx = commandtx.WithClaim(ctx, event)
		check := func(ctx context.Context) error {
			return commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error { return discoveryActive(ctx, tx, r, p) })
		}
		if err := check(ctx); err != nil {
			if errors.Is(err, errDiscoveryInactive) {
				return p.OperationID.String(), nil
			}
			return "", err
		}
		if nb == nil {
			return "", errors.New("NetBird adapter required")
		}
		// Recheck after acquiring the adapter lock and before storing its binding.
		// Revocation uses the same lock; cleanup includes bindings whose central
		// projection was interrupted. No external call runs in a DB transaction.
		peer, discoveryErr := nb.DiscoverNodePeer(netbird.WithDiscoveryFence(ctx, check), p.OperationID.String(), netbird.NodePeer{NodeID: p.NodeID.String(), IncarnationID: p.IncarnationID.String()})
		if errors.Is(discoveryErr, errDiscoveryInactive) {
			return p.OperationID.String(), nil
		}
		conflict := netbird.IsKind(discoveryErr, netbird.Conflict)
		terminal := conflict || event.Attempt >= attempts
		if discoveryErr != nil && !terminal {
			return "", discoveryErr
		}
		err := commandtx.InTx(ctx, r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			if err := discoveryActive(ctx, tx, r, p); err != nil {
				return err
			}
			if discoveryErr != nil {
				return recordDiscoveryFailure(ctx, tx, p, discoveryErr, conflict)
			}
			updated, err := tx.Exec(ctx, `UPDATE node_incarnations SET deployment_id=$2,peer_id=$3 WHERE id=$1 AND (peer_id IS NULL OR peer_id=$3)`, p.IncarnationID, r.Config.NetBirdDeployment, peer.ID)
			if err != nil {
				return err
			}
			if updated.RowsAffected() != 1 {
				return errors.New("incarnation peer changed during discovery")
			}
			evidence := uuid.Must(uuid.NewV7())
			if err := audit.Append(ctx, tx, db.AuditEvent{ID: evidence, Actor: nodelib.SystemActorUUID.String(), ActorType: "system", Action: "node.peer_discovered", Resource: "node:" + p.NodeID.String(), OperationID: &p.OperationID, Result: "succeeded"}, nil, map[string]any{"incarnation_id": p.IncarnationID, "peer_id": peer.ID}, "incarnation_id", "peer_id"); err != nil {
				return err
			}
			return stepSucceeded(ctx, tx, p.OperationID, "peer_readback", "audit_event:"+evidence.String())
		})
		if errors.Is(err, errDiscoveryInactive) {
			return p.OperationID.String(), nil
		}
		if err != nil {
			return "", err
		}
		if discoveryErr != nil && !conflict {
			return "", discoveryErr // dispatcher dead-letters exhausted transient work
		}
		// A conflict receipt means terminal classification was persisted, not a
		// successful binding. The failed operation and frozen incarnation say so.
		return p.OperationID.String(), nil
	}
}

func recordDiscoveryFailure(ctx context.Context, tx pgx.Tx, p nodeDiscovery, cause error, freeze bool) error {
	reason := netbird.DiscoveryReason(cause)
	code := api.ErrorCodeDEPENDENCYUNAVAILABLE
	if freeze {
		code = api.ErrorCodeSTATECONFLICT
		raw, err := canonjson.Marshal(map[string]any{"incarnation_id": p.IncarnationID, "reason": reason})
		if err != nil {
			return err
		}
		hash, err := canonjson.Hash(raw)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO node_identity_endpoints(incarnation_id,endpoint_hash,last_seen_at,mapping_frozen) VALUES($1,$2,clock_timestamp(),true) ON CONFLICT(incarnation_id) DO UPDATE SET mapping_frozen=true WHERE NOT node_identity_endpoints.mapping_frozen`, p.IncarnationID, hash[:]); err != nil {
			return err
		}
		if err := nodelib.CloseInactiveGenerationHoldsTx(ctx, tx, p.NodeID, time.Now().UTC()); err != nil {
			return err
		}
	}
	var rawStep []byte
	if err := tx.QueryRow(ctx, `SELECT steps->'peer_readback' FROM operations WHERE id=$1 FOR UPDATE`, p.OperationID).Scan(&rawStep); err != nil {
		return err
	}
	var step nodelib.StoredStep
	if err := json.Unmarshal(rawStep, &step); err != nil {
		return err
	}
	if !step.Required {
		return errors.New("discovery step is not required")
	}
	codeString := string(code)
	step.Status, step.ErrorCode, step.ErrorMessage = "failed", &codeString, &reason
	step.ObservedRef, step.ObservedAt, step.UpdatedAt = "", nil, time.Now().UTC()
	rawStep, err := canonjson.Marshal(step)
	if err != nil {
		return err
	}
	// Pending sibling steps must not leave exhausted discovery claiming progress
	// (§14.5 / D-040). Set step failure and terminal status in one guarded write.
	if _, err := tx.Exec(ctx, `UPDATE operations SET status='failed',completed_at=clock_timestamp(),error_code=$2,error_message=$3,steps=jsonb_set(steps,'{peer_readback}',$4::jsonb),observed_refs=observed_refs-'peer_readback',resource_version=coalesce(resource_version,0)+1 WHERE id=$1 AND status NOT IN ('succeeded','superseded') AND steps->'peer_readback'->>'status'<>'succeeded'`, p.OperationID, code, reason, rawStep); err != nil {
		return err
	}
	errorsJSON, err := canonjson.Marshal([]api.NodeError{{Code: code, Message: reason, Stage: "peer_discovery", ObservedAt: time.Now().UTC()}})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE nodes SET provisioning_state='failed',safety_status=CASE WHEN $3 THEN 'failed' ELSE safety_status END,member_forwarding_observed=false,recent_errors=$2::jsonb,resource_version=resource_version+1 WHERE id=$1`, p.NodeID, errorsJSON, freeze); err != nil {
		return err
	}
	return audit.Append(ctx, tx, db.AuditEvent{ID: uuid.Must(uuid.NewV7()), Actor: nodelib.SystemActorUUID.String(), ActorType: "system", Action: "node.peer_discovery_failed", Resource: "node:" + p.NodeID.String(), OperationID: &p.OperationID, Result: "failed"}, nil, map[string]any{"incarnation_id": p.IncarnationID, "reason": reason}, "incarnation_id", "reason")
}
