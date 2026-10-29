package worker

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"os"
	"time"
)

// NodeJobs is injectable for composition and tests. External calls all go through
// the NetBird adapter; it verifies remote state before returning success.
func NodeJobs(r *httpwire.Runtime, nb *netbird.Client, fetch releases.Fetcher) Registry {
	jobs := Registry{}
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
	var exists bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM outbox_events WHERE event_type='node.reconcile' AND delivered_at IS NULL AND dead_lettered_at IS NULL AND id<>$1)`, event.ID).Scan(&exists); e != nil {
		return e
	}
	if exists {
		return nil
	}
	raw, _ := canonjson.Marshal(map[string]int{"schema_version": 1})
	return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: uuid.MustParse(releases.FleetID), AggregateType: "fleet_config", AggregateVersion: "1", EventType: "node.reconcile", SchemaVersion: 1, Payload: raw, AvailableAt: time.Now().Add(time.Duration(seconds) * time.Second)})
}

// The current Jobs() framework has no pool/lifecycle injection. Each dispatched
// node job opens a bounded connection pool from the same secret-file setting.
func nodeJobs() Registry {
	out := Registry{}
	for _, name := range []string{"release.verify", "node.provision.requested", "node.routes.withdraw", "node.peer.withdraw", "node.reconcile", "fleet.update_created", "fleet.first_sync_created", "fleet.update_retry", "fleet.release_support_changed", "node.updated"} {
		out.Register(name, func(ctx context.Context, event db.ClaimedEvent) (string, error) {
			dsn, e := config.ReadSecret(os.Getenv("FVPN_DATABASE_URL_FILE"))
			if e != nil {
				return "", e
			}
			pool, e := db.Open(ctx, dsn, db.OpenOptions{})
			if e != nil {
				return "", e
			}
			defer pool.Close()
			r, e := httpwire.Open(ctx, pool, nil)
			if e != nil {
				return "", e
			}
			nb, e := netbird.New(netbird.Config{BaseURL: r.Config.NetBirdURL, DeploymentID: r.Config.NetBirdDeployment, Token: netbird.TokenFile(r.Config.NetBirdTokenFile), Bindings: nodes.NetBirdBindings{Pool: pool}, Timeout: time.Duration(r.Config.FetchSeconds) * time.Second})
			if e != nil {
				return "", e
			}
			fetch, e := releases.NewHTTPSFetcher(r.Releases.Validator, time.Duration(r.Config.FetchSeconds)*time.Second)
			if e != nil {
				return "", e
			}
			return NodeJobs(r, nb, fetch)[event.EventType](ctx, event)
		})
	}
	return out
}
