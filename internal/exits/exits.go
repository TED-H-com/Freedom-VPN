// Package exits implements exit publication transactions, asynchronous worker
// execution with NetBird adapter read-back, and external binding drift reconciliation (§5.2, §7.2, §14.2, §15.3, §17.4, §18.3).
package exits

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Default NetBird provider and deployment identifiers.
const (
	ProviderNetBird     = "netbird"
	DefaultDeploymentID = "default"
)

// PublishExitSpec describes the desired NetBird route/group configuration for an exit (§5.2, §7.2).
type PublishExitSpec struct {
	DeploymentID   string
	ExitID         uuid.UUID
	NodeID         uuid.UUID
	PeerID         string
	DisplayName    string
	Region         string
	MemberGroupIDs []uuid.UUID
	ExistingRef    string
	DesiredHash    []byte
}

// PublishedRouteEvidence contains the external identifier returned by NetBird after publishing an exit.
type PublishedRouteEvidence struct {
	ExternalID  string
	PeerID      string
	Enabled     bool
	GroupIDs    []string
	ObservedRef string
}

// ObservedExitState represents the read-back state of an exit route from NetBird (§5.2, §7.2, §17.4).
type ObservedExitState struct {
	Exists       bool
	ExternalID   string
	PeerID       string
	Enabled      bool
	GroupIDs     []string
	ObservedHash []byte
	ObservedRef  string
}

// NetBirdAdapter abstracts the central NetBird management API for exit publication,
// withdrawal, and read-back reconciliation (§7.2).
type NetBirdAdapter interface {
	PublishExit(ctx context.Context, spec PublishExitSpec) (PublishedRouteEvidence, error)
	WithdrawExit(ctx context.Context, deploymentID string, exitID uuid.UUID, externalID string) (string, error)
	ReadExitState(ctx context.Context, deploymentID string, exitID uuid.UUID, externalID string) (ObservedExitState, error)
}

// FakeNetBirdAdapter is a thread-safe in-memory NetBirdAdapter for unit and integration tests.
// It tracks routes by external ID and counts PublishExit/WithdrawExit calls to prove idempotency.
type FakeNetBirdAdapter struct {
	mu            sync.Mutex
	routes        map[string]ObservedExitState
	exitToRouteID map[uuid.UUID]string
	publishCalls  int
	withdrawCalls int
	publishErr    error
	readbackErr   error
	withdrawErr   error
}

// NewFakeNetBirdAdapter constructs a FakeNetBirdAdapter.
func NewFakeNetBirdAdapter() *FakeNetBirdAdapter {
	return &FakeNetBirdAdapter{
		routes:        make(map[string]ObservedExitState),
		exitToRouteID: make(map[uuid.UUID]string),
	}
}

// SetPublishError configures PublishExit to fail with err when non-nil.
func (f *FakeNetBirdAdapter) SetPublishError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.publishErr = err
}

// SetReadbackError configures ReadExitState to fail with err when non-nil.
func (f *FakeNetBirdAdapter) SetReadbackError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.readbackErr = err
}

// SetWithdrawError configures WithdrawExit to fail with err when non-nil.
func (f *FakeNetBirdAdapter) SetWithdrawError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.withdrawErr = err
}

// InjectRouteState mutates the in-memory NetBird route state directly (for simulating external drift).
func (f *FakeNetBirdAdapter) InjectRouteState(exitID uuid.UUID, state ObservedExitState) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if state.ExternalID != "" {
		f.exitToRouteID[exitID] = state.ExternalID
		f.routes[state.ExternalID] = state
	}
}

// RouteCount returns the number of distinct NetBird routes created in FakeNetBirdAdapter.
func (f *FakeNetBirdAdapter) RouteCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.routes)
}

// PublishCalls returns the total number of PublishExit invocations.
func (f *FakeNetBirdAdapter) PublishCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.publishCalls
}

// WithdrawCalls returns the total number of WithdrawExit invocations.
func (f *FakeNetBirdAdapter) WithdrawCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.withdrawCalls
}

// PublishExit creates or updates an exit route in FakeNetBirdAdapter.
// If spec.ExistingRef is non-empty (or the exit already has a bound route), it updates in place without duplicating.
func (f *FakeNetBirdAdapter) PublishExit(_ context.Context, spec PublishExitSpec) (PublishedRouteEvidence, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.publishCalls++
	if f.publishErr != nil {
		return PublishedRouteEvidence{}, f.publishErr
	}

	extID := spec.ExistingRef
	if extID == "" {
		if existing, ok := f.exitToRouteID[spec.ExitID]; ok {
			extID = existing
		} else {
			extID = "nb-route-" + spec.ExitID.String()
		}
	}

	groups := make([]string, 0, len(spec.MemberGroupIDs))
	for _, gid := range spec.MemberGroupIDs {
		groups = append(groups, gid.String())
	}
	sort.Strings(groups)

	obsRef := fmt.Sprintf("netbird:route:%s:enabled:true", extID)
	state := ObservedExitState{
		Exists:       true,
		ExternalID:   extID,
		PeerID:       spec.PeerID,
		Enabled:      true,
		GroupIDs:     groups,
		ObservedHash: append([]byte(nil), spec.DesiredHash...),
		ObservedRef:  obsRef,
	}
	f.routes[extID] = state
	f.exitToRouteID[spec.ExitID] = extID

	return PublishedRouteEvidence{
		ExternalID:  extID,
		PeerID:      spec.PeerID,
		Enabled:     true,
		GroupIDs:    groups,
		ObservedRef: obsRef,
	}, nil
}

// WithdrawExit disables/removes the exit route in FakeNetBirdAdapter.
func (f *FakeNetBirdAdapter) WithdrawExit(_ context.Context, _ string, exitID uuid.UUID, externalID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.withdrawCalls++
	if f.withdrawErr != nil {
		return "", f.withdrawErr
	}
	extID := externalID
	if extID == "" {
		extID = f.exitToRouteID[exitID]
	}
	if extID == "" {
		extID = "nb-route-" + exitID.String()
	}
	st := f.routes[extID]
	st.Exists = false
	st.Enabled = false
	st.ExternalID = extID
	st.GroupIDs = nil
	st.ObservedRef = fmt.Sprintf("netbird:route:%s:withdrawn", extID)
	f.routes[extID] = st
	return st.ObservedRef, nil
}

// ReadExitState returns the current in-memory route state for exitID/externalID.
func (f *FakeNetBirdAdapter) ReadExitState(_ context.Context, _ string, exitID uuid.UUID, externalID string) (ObservedExitState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.readbackErr != nil {
		return ObservedExitState{}, f.readbackErr
	}
	extID := externalID
	if extID == "" {
		extID = f.exitToRouteID[exitID]
	}
	if extID == "" {
		return ObservedExitState{
			Exists:      false,
			ObservedRef: fmt.Sprintf("netbird:exit:%s:absent", exitID),
		}, nil
	}
	st, ok := f.routes[extID]
	if !ok {
		return ObservedExitState{
			Exists:      false,
			ExternalID:  extID,
			ObservedRef: fmt.Sprintf("netbird:route:%s:absent", extID),
		}, nil
	}
	return st, nil
}

// Service manages exit publication commands, worker execution, and NetBird reconciliation.
type Service struct {
	pool     *pgxpool.Pool
	compiler *desiredstate.Compiler
	netbird  NetBirdAdapter
	now      func() time.Time
}

// Option configures a Service.
type Option func(*Service)

// WithClock overrides the clock used by Service.
func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

// NewService constructs an exits Service.
func NewService(pool *pgxpool.Pool, compiler *desiredstate.Compiler, netbird NetBirdAdapter, opts ...Option) *Service {
	s := &Service{
		pool:     pool,
		compiler: compiler,
		netbird:  netbird,
		now:      func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SetPublicationParams holds inputs for PUT /admin/nodes/{id}/publication (§5.2, §14.2).
type SetPublicationParams struct {
	NodeID         uuid.UUID
	Principal      string
	IdempotencyKey uuid.UUID
	IfMatch        string
	RequestID      *uuid.UUID
	Request        api.PublicationRequest
}

// SetPublication writes the exit's publication_intent, increments resource_version,
// advances desired_config_generation once, creates a pending operation with steps
// (`agent_policy`, `netbird_route_readback`, `data_path_probe`), enqueues an outbox event,
// and appends an audit event in one atomic transaction (§5.2 step 1, §14.2, §15.5.5).
// It opens a transaction, manages idempotency, and delegates to SetPublicationTx.
func (s *Service) SetPublication(ctx context.Context, p SetPublicationParams) (api.Operation, string, error) {
	if p.IdempotencyKey == uuid.Nil {
		return api.Operation{}, "", apierr.New(apierr.MalformedRequest, "Idempotency-Key header is required")
	}
	if _, err := nodes.ParseIfMatch(p.IfMatch); err != nil {
		return api.Operation{}, "", err
	}
	if strings.TrimSpace(p.Request.Reason) == "" {
		return api.Operation{}, "", apierr.Validation("reason is required",
			apierr.FieldError{Field: "/reason", Reason: "required"},
		)
	}
	if _, err := publicationOperationKind(p.Request.PublicationIntent); err != nil {
		return api.Operation{}, "", err
	}

	_, _, principal := nodes.ResolveAdminActor(ctx, p.Principal)

	requestFields := map[string]any{
		"node_id":            p.NodeID.String(),
		"publication_intent": string(p.Request.PublicationIntent),
		"reason":             p.Request.Reason,
	}
	// Preserve pre-D-057 replay hashes when membership was omitted. An
	// explicit empty replacement is a distinct command from omission.
	if p.Request.GroupIds != nil {
		requestFields["group_ids"] = *p.Request.GroupIds
	}
	reqCanon, err := canonjson.Marshal(requestFields)
	if err != nil {
		return api.Operation{}, "", err
	}
	reqHashArr := sha256.Sum256(reqCanon)
	keyHash := sha256.Sum256([]byte(p.IdempotencyKey.String()))
	scope := db.IdempotencyScope{
		Principal:   principal,
		Method:      "PUT",
		Route:       "/api/v1/admin/nodes/" + p.NodeID.String() + "/publication",
		KeyHash:     keyHash[:],
		RequestHash: reqHashArr[:],
	}

	var (
		outOp   api.Operation
		outETag string
	)
	err = db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		rec, err := db.BeginIdempotent(ctx, tx, scope, 0)
		if err != nil {
			if errors.Is(err, db.ErrIdempotencyConflict) {
				return apierr.New(apierr.IdempotencyConflict, "Idempotency-Key reused with a different request payload")
			}
			return err
		}
		if !rec.New {
			if rec.OperationID == nil {
				return apierr.New(apierr.IdempotencyConflict, "request with this Idempotency-Key is currently in progress")
			}
			_, etag, err := nodes.GetNodeTx(ctx, tx, p.NodeID)
			if err != nil {
				return err
			}
			op, err := nodes.GetOperationTx(ctx, tx, *rec.OperationID)
			if err != nil {
				return err
			}
			outOp = op
			outETag = etag
			return nil
		}

		op, etag, err := s.SetPublicationTx(ctx, tx, p)
		if err != nil {
			return err
		}

		nodeIDCopy := p.NodeID
		if err := db.CompleteIdempotent(ctx, tx, rec.ID, db.IdempotencyResult{
			OperationID:    &op.OperationId,
			ResourceType:   "node",
			ResourceID:     &nodeIDCopy,
			ResponseStatus: 202,
		}); err != nil {
			return err
		}

		outOp = op
		outETag = etag
		return nil
	})
	if err != nil {
		return api.Operation{}, "", err
	}
	return outOp, outETag, nil
}

// SetPublicationTx executes the publication change inside the caller's transaction (§5.2 step 1, §14.2, §15.5.5).
//
// Invariants enforced:
//   - Locks `nodes` before `exits` (`FOR UPDATE OF n, e`, nodes before exits) and checks `If-Match` under the row lock.
//   - Verifies serving readiness when `publication_intent == published`.
//   - Writes `exits.publication_intent`, bumps `nodes.resource_version` and `exits.resource_version`, and advances `nodes.desired_config_generation` exactly once per accepted change.
//   - Creates a pending operation (`node.publish` with `agent_policy`, `netbird_route_readback`, `data_path_probe`; `node.unpublish`; `node.disable`).
//   - Enqueues the `node.publication_changed` outbox event and appends an `exit:<uuid>` audit event through `audit.NewEvent`/`audit.AppendEvent`.
func (s *Service) SetPublicationTx(ctx context.Context, tx pgx.Tx, p SetPublicationParams) (api.Operation, string, error) {
	expectedVer, err := nodes.ParseIfMatch(p.IfMatch)
	if err != nil {
		return api.Operation{}, "", err
	}
	reason := strings.TrimSpace(p.Request.Reason)
	if reason == "" {
		return api.Operation{}, "", apierr.Validation("reason is required",
			apierr.FieldError{Field: "/reason", Reason: "required"},
		)
	}
	opKind, err := publicationOperationKind(p.Request.PublicationIntent)
	if err != nil {
		return api.Operation{}, "", err
	}

	actorUUID, actorType, principal := nodes.ResolveAdminActor(ctx, p.Principal)
	reqID := p.RequestID
	if reqID == nil {
		if rawReqID, ok := apierr.RequestIDFromContext(ctx); ok {
			if parsed, pErr := uuid.Parse(rawReqID); pErr == nil && parsed != uuid.Nil {
				reqID = &parsed
			}
		}
	} else if *reqID == uuid.Nil {
		reqID = nil
	}

	// Lock nodes before exits (`FOR UPDATE OF n, e`, §15.3, docs/db/README.md).
	var nodeExists bool
	if err := tx.QueryRow(ctx, `SELECT true FROM nodes n WHERE n.id = $1 FOR UPDATE OF n`, p.NodeID).Scan(&nodeExists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Operation{}, "", apierr.Newf(apierr.ResourceNotFound, "node %s not found", p.NodeID)
		}
		return api.Operation{}, "", err
	}

	var (
		currentVerStr string
		currentGenStr string
		exitID        uuid.UUID
		oldIntent     string
		oldObserved   string
	)
	if err := tx.QueryRow(ctx, `
		SELECT n.resource_version::text, n.desired_config_generation::text, e.id, e.publication_intent, e.publication_observed
		FROM nodes n
		JOIN exits e ON e.node_id = n.id
		WHERE n.id = $1
		FOR UPDATE OF n, e`, p.NodeID).Scan(&currentVerStr, &currentGenStr, &exitID, &oldIntent, &oldObserved); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Operation{}, "", apierr.Newf(apierr.StateConflict, "node %s has no exit record", p.NodeID)
		}
		return api.Operation{}, "", err
	}

	// Check If-Match under the row lock (§14.5).
	currentVer, err := strconv.ParseUint(currentVerStr, 10, 64)
	if err != nil {
		return api.Operation{}, "", err
	}
	if currentVerStr != expectedVer {
		return api.Operation{}, "", apierr.New(apierr.ResourceVersionConflict, "If-Match version does not match current resource_version").
			WithDetail("current_resource_version", currentVerStr)
	}

	nodeDetail, _, err := nodes.GetNodeTx(ctx, tx, p.NodeID)
	if err != nil {
		return api.Operation{}, "", err
	}

	// When setting intent=published, verify the node meets the serving baseline (§5.1, §14.2).
	if p.Request.PublicationIntent == api.PublicationIntentPublished {
		var appliedGen uint64
		if gen, err := nodeDetail.ConfigGeneration.Applied.Get(); err == nil {
			appliedGen = uint64(gen)
		}
		var currentReleaseID *uuid.UUID
		if id, err := nodeDetail.Releases.CurrentReleaseId.Get(); err == nil {
			currentReleaseID = &id
		}
		if err := nodes.CheckPublishable(nodes.ServingReadiness{
			LifecycleState:          nodeDetail.LifecycleState,
			ProvisioningState:       nodeDetail.ProvisioningState,
			PublicationIntent:       p.Request.PublicationIntent,
			ControlStatus:           nodeDetail.ControlStatus,
			SafetyStatus:            nodeDetail.SafetyStatus,
			DataHealth:              nodeDetail.DataHealth,
			DesiredConfigGeneration: uint64(nodeDetail.ConfigGeneration.Desired),
			AppliedConfigGeneration: appliedGen,
			CurrentReleaseID:        currentReleaseID,
		}); err != nil {
			return api.Operation{}, "", err
		}
	}

	groups := nodeDetail.MemberGroupIds
	if p.Request.GroupIds != nil {
		groups, err = replacePublicationGroups(ctx, tx, exitID, *p.Request.GroupIds)
		if err != nil {
			return api.Operation{}, "", err
		}
	}

	now := s.now().UTC()
	currentGen, err := strconv.ParseUint(currentGenStr, 10, 64)
	if err != nil {
		return api.Operation{}, "", err
	}
	nextGen := currentGen + 1
	nextGenStr := strconv.FormatUint(nextGen, 10)
	newVer := strconv.FormatUint(currentVer+1, 10)

	// Advance desired_config_generation once and bump resource_version on nodes.
	if _, err := tx.Exec(ctx, `
		UPDATE nodes
		SET desired_config_generation = $2::numeric,
		    resource_version = $3::numeric,
		    updated_at = $4
		WHERE id = $1`,
		p.NodeID, nextGenStr, newVer, now); err != nil {
		return api.Operation{}, "", err
	}

	// Update exits.publication_intent (which lives only on exits, §15.3).
	if _, err := tx.Exec(ctx, `
		UPDATE exits
		SET publication_intent = $1,
		    resource_version = resource_version + 1
		WHERE id = $2`,
		string(p.Request.PublicationIntent), exitID); err != nil {
		return api.Operation{}, "", err
	}

	op, err := nodes.CreateOperation(ctx, tx, nodes.CreateOperationParams{
		Kind:              string(opKind),
		Principal:         principal,
		ResourceType:      "node",
		ResourceID:        p.NodeID,
		ResourceVersion:   &newVer,
		RequestID:         reqID,
		DesiredGeneration: &nextGen,
		Now:               now,
	})
	if err != nil {
		return api.Operation{}, "", err
	}

	payload, _ := json.Marshal(map[string]any{
		"node_id":            p.NodeID.String(),
		"exit_id":            exitID.String(),
		"operation_id":       op.OperationId.String(),
		"publication_intent": string(p.Request.PublicationIntent),
		"desired_generation": nextGenStr,
		"resource_version":   newVer,
		"occurred_at":        now.Format(time.RFC3339Nano),
	})
	if err := db.EnqueueOutbox(ctx, tx, db.OutboxEvent{
		AggregateID:      p.NodeID,
		AggregateType:    "node",
		AggregateVersion: newVer,
		EventType:        desiredstate.EventNodePublicationChanged,
		SchemaVersion:    1,
		Payload:          payload,
	}); err != nil {
		return api.Operation{}, "", err
	}

	auditEv, err := audit.NewEvent(audit.Entry{
		Actor: audit.Actor{
			Type: api.AuditEventActorType(actorType),
			ID:   actorUUID,
		},
		Action: string(opKind),
		Resource: api.ResourceRef{
			Type: api.ResourceTypeExit,
			Id:   exitID,
		},
		Result:      api.AuditEventResultSucceeded,
		OperationID: &op.OperationId,
		RequestID:   reqID,
		Before: map[string]any{
			"publication_intent":   oldIntent,
			"publication_observed": oldObserved,
			"resource_version":     currentVerStr,
			"group_ids":            nodeDetail.MemberGroupIds,
		},
		After: map[string]any{
			"publication_intent":   string(p.Request.PublicationIntent),
			"publication_observed": oldObserved,
			"operation_id":         op.OperationId.String(),
			"reason":               reason,
			"resource_version":     newVer,
			"group_ids":            groups,
		},
		Fields: []string{
			"publication_intent",
			"publication_observed",
			"operation_id",
			"reason",
			"resource_version",
			"group_ids",
		},
	})
	if err != nil {
		return api.Operation{}, "", err
	}
	if err := audit.AppendEvent(ctx, tx, auditEv); err != nil {
		return api.Operation{}, "", err
	}

	return op, nodes.FormatETag(newVer), nil
}

func publicationOperationKind(intent api.PublicationIntent) (api.OperationKind, error) {
	switch intent {
	case api.PublicationIntentPublished:
		return api.OperationKindNodePublish, nil
	case api.PublicationIntentUnpublished:
		return api.OperationKindNodeUnpublish, nil
	case api.PublicationIntentDisabled:
		return api.OperationKindNodeDisable, nil
	default:
		return "", apierr.Validation(
			fmt.Sprintf("invalid publication_intent %q", intent),
			apierr.FieldError{Field: "/publication_intent", Reason: "enum"},
		)
	}
}

// ExecuteOperation runs or retries the worker job for a node publication/withdrawal operation
// (`node.publish`, `node.unpublish`, `node.disable`, `node.revoke`, `node.retire`) (§5.2, §18.3).
func (s *Service) ExecuteOperation(ctx context.Context, opID uuid.UUID) (api.Operation, error) {
	op, err := nodes.GetOperation(ctx, s.pool, opID)
	if err != nil {
		return api.Operation{}, err
	}
	if op.Status == api.OperationStatusSucceeded || op.Status == api.OperationStatusSuperseded {
		return op, nil
	}

	unlock, err := s.lockPublication(ctx, op.Resource.Id)
	if err != nil {
		return api.Operation{}, err
	}
	defer unlock()
	op, err = nodes.GetOperation(ctx, s.pool, opID)
	if err != nil {
		return api.Operation{}, err
	}
	if op.Status == api.OperationStatusSucceeded || op.Status == api.OperationStatusSuperseded {
		return op, nil
	}

	nodeID := op.Resource.Id
	switch string(op.Kind) {
	case nodes.OpKindPublish:
		return s.executePublish(ctx, opID, nodeID)
	case nodes.OpKindUnpublish, nodes.OpKindDisable, nodes.OpKindRevoke, nodes.OpKindRetire:
		return s.executeWithdraw(ctx, opID, nodeID)
	default:
		if s.compiler != nil {
			if _, err := s.compiler.Compile(ctx, nodeID); err != nil {
				return api.Operation{}, err
			}
		}
		return nodes.GetOperation(ctx, s.pool, opID)
	}
}

// executePublish executes the §5.2 step 2-3 publish workflow:
// 1. Ensures desired state is compiled at >= desired_generation.
// 2. Confirms node's isolation (safety_status=passed), readiness, and policy ACK (applied_config_generation >= desired_generation).
// 3. Calls NetBirdAdapter.PublishExit (consulting external_bindings first so retries never create duplicate routes) and reads back.
// 4. Confirms data_path_probe and marks steps and operation succeeded (publication_observed = 'active').
// Any failure keeps publication_intent='published' and records specific failure details for retry.
func (s *Service) executePublish(ctx context.Context, opID uuid.UUID, nodeID uuid.UUID) (api.Operation, error) {
	now := s.now().UTC()

	nodeDetail, _, err := s.NodeService().GetNode(ctx, nodeID)
	if err != nil {
		return api.Operation{}, err
	}
	op, err := nodes.GetOperation(ctx, s.pool, opID)
	if err != nil {
		return api.Operation{}, err
	}
	var desiredGen uint64
	if gen, err := op.DesiredGeneration.Get(); err == nil {
		desiredGen = uint64(gen)
	}
	if s.compiler != nil && uint64(nodeDetail.ConfigGeneration.Desired) < desiredGen {
		compiled, cErr := s.compiler.Compile(ctx, nodeID)
		if cErr != nil {
			return s.failPublishStep(ctx, opID, nodeID, nodes.StepAgentPolicy, string(apierr.DependencyUnavailable), "desired_state_unavailable")
		}
		desiredGen = compiled.Generation
		nodeDetail, _, err = s.NodeService().GetNode(ctx, nodeID)
		if err != nil {
			return api.Operation{}, err
		}
	}

	// A later compile may have advanced beyond this operation's initial target.
	desiredGen = max(desiredGen, uint64(nodeDetail.ConfigGeneration.Desired))

	var appliedGen uint64
	appliedGenStr := "0"
	if gen, err := nodeDetail.ConfigGeneration.Applied.Get(); err == nil {
		appliedGen = uint64(gen)
		appliedGenStr = gen.String()
	}

	// Step 1: Confirm node isolation, readiness, online control status, and policy ACK (§5.2 step 2).
	var currentReleaseID *uuid.UUID
	if id, err := nodeDetail.Releases.CurrentReleaseId.Get(); err == nil {
		currentReleaseID = &id
	}
	if err := nodes.CheckPublishable(nodes.ServingReadiness{
		LifecycleState:            nodeDetail.LifecycleState,
		ProvisioningState:         nodeDetail.ProvisioningState,
		PublicationIntent:         nodeDetail.PublicationIntent,
		ControlStatus:             nodeDetail.ControlStatus,
		SafetyStatus:              nodeDetail.SafetyStatus,
		DataHealth:                nodeDetail.DataHealth,
		DesiredConfigGeneration:   desiredGen,
		AppliedConfigGeneration:   appliedGen,
		CurrentReleaseID:          currentReleaseID,
		RequireOnlineAndACKForRun: true,
	}); err != nil {
		cause := "readiness_unconfirmed"
		if appliedGen < desiredGen {
			cause = "stale_generation"
		}
		return s.failGatedPublish(ctx, opID, nodeID, nodes.StepAgentPolicy, string(apierr.NodeNotServiceable), err.Error(), cause)
	}

	if err := s.markStepSucceeded(ctx, opID, nodes.StepAgentPolicy, fmt.Sprintf("node:%s:gen:%s:safety:passed", nodeID, appliedGenStr), now); err != nil {
		return api.Operation{}, err
	}

	// §5.2: route attachment follows the current generation's data-path gate,
	// including the first member-bearing generation after an empty publication.
	cause, err := s.publicationGateCause(ctx, nodeID, desiredGen)
	if err != nil {
		return api.Operation{}, err
	}
	if cause != "" {
		return s.failGatedPublish(ctx, opID, nodeID, nodes.StepDataPathProbe, string(apierr.NodeNotServiceable), "member_forwarding_unconfirmed", cause)
	}

	// Step 2: Consult external_bindings and publish route via NetBird adapter, then read back (§5.2 step 2-3).
	exitID, exitErr := nodeDetail.ExitId.Get()
	if exitErr != nil {
		return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.StateConflict), "node has no exit record")
	}

	peerID := ""
	if value, err := nodeDetail.NetbirdPeerId.Get(); err == nil {
		peerID = value
	}
	deploymentID, err := s.lookupDeploymentID(ctx, nodeID)
	if err != nil {
		return api.Operation{}, err
	}
	existingExtID, _ := s.lookupExternalBinding(ctx, exitID, deploymentID)

	desiredHash, err := computeExitDesiredHash(exitID, peerID, true, nodeDetail.MemberGroupIds)
	if err != nil {
		return api.Operation{}, err
	}

	if s.netbird == nil {
		return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), "netbird adapter not configured")
	}

	dispName := ""
	if value, err := nodeDetail.Name.Get(); err == nil {
		dispName = value
	}
	regName := ""
	if value, err := nodeDetail.Region.Get(); err == nil {
		regName = value
	}

	pubEv, err := s.netbird.PublishExit(ctx, PublishExitSpec{
		DeploymentID:   deploymentID,
		ExitID:         exitID,
		NodeID:         nodeID,
		PeerID:         peerID,
		DisplayName:    dispName,
		Region:         regName,
		MemberGroupIDs: nodeDetail.MemberGroupIds,
		ExistingRef:    existingExtID,
		DesiredHash:    desiredHash,
	})
	if err != nil {
		cause := PublicationFailureCause(err, "publish")
		_ = s.recordBindingError(ctx, exitID, deploymentID, existingExtID, desiredHash, cause)
		return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), cause)
	}

	readback, err := s.netbird.ReadExitState(ctx, deploymentID, exitID, pubEv.ExternalID)
	if err != nil {
		cause := PublicationFailureCause(err, "readback")
		_ = s.recordBindingError(ctx, exitID, deploymentID, pubEv.ExternalID, desiredHash, cause)
		return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), cause)
	}
	if !readback.Exists || !readback.Enabled {
		msg := "route_disabled"
		if !readback.Exists {
			msg = "route_missing"
		}
		_ = s.recordBindingError(ctx, exitID, deploymentID, pubEv.ExternalID, desiredHash, msg)
		return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), msg)
	}

	obsHash := readback.ObservedHash
	if len(obsHash) == 0 {
		obsHash = desiredHash
	}
	if err := s.upsertExternalBinding(ctx, exitID, deploymentID, readback.ExternalID, desiredHash, obsHash, "synced", now); err != nil {
		return api.Operation{}, err
	}

	refVal := readback.ObservedRef
	if refVal == "" {
		refVal = fmt.Sprintf("netbird:route:%s", readback.ExternalID)
	}
	if err := s.markStepSucceeded(ctx, opID, nodes.StepNetBirdRouteReadback, refVal, now); err != nil {
		return api.Operation{}, err
	}

	// A compile or a newer observation can arrive during management I/O. Close
	// the route again if that changed the gate; never finalize stale evidence.
	cause, err = s.publicationGateCause(ctx, nodeID, desiredGen)
	if err != nil {
		return api.Operation{}, err
	}
	if cause != "" {
		return s.failGatedPublish(ctx, opID, nodeID, nodes.StepDataPathProbe, string(apierr.NodeNotServiceable), "member_forwarding_unconfirmed", cause)
	}

	if err := s.markStepSucceeded(ctx, opID, nodes.StepDataPathProbe, fmt.Sprintf("node:%s:probe:passed", nodeID), now); err != nil {
		return api.Operation{}, err
	}

	// Complete operation and mark exit publication_observed = 'active'.
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			UPDATE operations
			SET status = 'succeeded',
			    completed_at = $2,
			    observed_generation = $3::numeric,
			    error_code = NULL,
			    error_message = NULL,
			    resource_version = coalesce(resource_version, 0) + 1
			WHERE id = $1`, opID, now, appliedGenStr); err != nil {
			return err
		}
		return s.publicationObservedTx(ctx, tx, opID, nodeID, "active", "")
	})
	if err != nil {
		return api.Operation{}, err
	}
	return nodes.GetOperation(ctx, s.pool, opID)
}

// executeWithdraw executes the §5.2 step 4 unpublish/disable/revoke/retire workflow:
// 1. Withdraws member forwarding on the node first by compiling a new desired-state generation.
// 2. Checks if the node has acknowledged forwarding withdrawal (applied_config_generation >= desired_generation and member_forwarding_observed == false).
// 3. Even if the node does not respond (offline/stale/revoked/retired), withdraws the NetBird route and peer permissions centrally and reads back!
// 4. Marks operation succeeded if node ACKed, or partially_succeeded if NetBird route is withdrawn but node withdrawal is unconfirmed (§5.2 step 4, §18.3).
func (s *Service) executeWithdraw(ctx context.Context, opID uuid.UUID, nodeID uuid.UUID) (api.Operation, error) {
	now := s.now().UTC()

	nodeDetail, _, err := s.NodeService().GetNode(ctx, nodeID)
	if err != nil {
		return api.Operation{}, err
	}
	op, err := nodes.GetOperation(ctx, s.pool, opID)
	if err != nil {
		return api.Operation{}, err
	}
	var desiredGen uint64
	if gen, err := op.DesiredGeneration.Get(); err == nil {
		desiredGen = uint64(gen)
	}

	// Step 1: Compile new desired-state generation withdrawing member forwarding if node has an incarnation.
	if s.compiler != nil && (nodeDetail.CurrentIncarnationId.IsSpecified() && !nodeDetail.CurrentIncarnationId.IsNull()) && uint64(nodeDetail.ConfigGeneration.Desired) < desiredGen {
		if compiled, cErr := s.compiler.Compile(ctx, nodeID); cErr == nil {
			desiredGen = compiled.Generation
			nodeDetail, _, _ = s.NodeService().GetNode(ctx, nodeID)
		}
	}

	var appliedGen uint64
	appliedGenStr := "0"
	if gen, err := nodeDetail.ConfigGeneration.Applied.Get(); err == nil {
		appliedGen = uint64(gen)
		appliedGenStr = gen.String()
	}

	forwarding, forwardingErr := nodeDetail.MemberForwardingObserved.Get()
	nodeConfirmed := nodeDetail.ControlStatus == api.ControlStatusOnline &&
		appliedGen >= desiredGen &&
		forwardingErr == nil && !forwarding

	if nodeConfirmed {
		if err := s.markStepSucceeded(ctx, opID, nodes.StepAgentPolicy, fmt.Sprintf("node:%s:gen:%s:forwarding:false", nodeID, appliedGenStr), now); err != nil {
			return api.Operation{}, err
		}
	}

	// Step 2: Withdraw NetBird route and peer permissions centrally even if the node did not respond (§5.2 step 4).
	exitID, exitErr := nodeDetail.ExitId.Get()
	if exitErr != nil {
		return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.StateConflict), "node has no exit record")
	}

	deploymentID, err := s.lookupDeploymentID(ctx, nodeID)
	if err != nil {
		return api.Operation{}, err
	}
	existingExtID, _ := s.lookupExternalBinding(ctx, exitID, deploymentID)

	if s.netbird == nil {
		return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), "netbird adapter not configured")
	}

	withdrawRef, err := s.netbird.WithdrawExit(ctx, deploymentID, exitID, existingExtID)
	if err != nil {
		return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), fmt.Sprintf("netbird withdraw failed: %v", err))
	}
	readback, err := s.netbird.ReadExitState(ctx, deploymentID, exitID, existingExtID)
	if err != nil {
		return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), fmt.Sprintf("netbird withdraw read-back failed: %v", err))
	}
	if readback.Exists && readback.Enabled {
		return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), "netbird read-back showed route still enabled after withdrawal")
	}

	peerID := ""
	if value, err := nodeDetail.NetbirdPeerId.Get(); err == nil {
		peerID = value
	}
	desiredHash, _ := computeExitDesiredHash(exitID, peerID, false, nil)
	extIDToStore := existingExtID
	if extIDToStore == "" {
		extIDToStore = "nb-route-" + exitID.String()
	}
	if err := s.upsertExternalBinding(ctx, exitID, deploymentID, extIDToStore, desiredHash, desiredHash, "synced", now); err != nil {
		return api.Operation{}, err
	}

	if withdrawRef == "" {
		withdrawRef = fmt.Sprintf("netbird:route:%s:withdrawn", extIDToStore)
	}
	if err := s.markStepSucceeded(ctx, opID, nodes.StepNetBirdRouteReadback, withdrawRef, now); err != nil {
		return api.Operation{}, err
	}

	// Step 3: Finalize operation status.
	err = db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			UPDATE exits
			SET publication_observed = 'withdrawn',
			    resource_version = resource_version + 1
			WHERE id = $1`, exitID); err != nil {
			return err
		}

		if nodeConfirmed {
			if _, err := tx.Exec(ctx, `
				UPDATE operations
				SET status = 'succeeded',
				    completed_at = $2,
				    observed_generation = $3::numeric,
				    error_code = NULL,
				    error_message = NULL,
				    resource_version = coalesce(resource_version, 0) + 1
				WHERE id = $1`, opID, now, appliedGenStr); err != nil {
				return err
			}
			return nil
		}

		// Node has not responded; record unconfirmed node withdrawal on agent_policy step
		// and mark operation partially_succeeded (§5.2 step 4, §18.3).
		var stepsBytes []byte
		if err := tx.QueryRow(ctx, `SELECT steps FROM operations WHERE id = $1 FOR UPDATE`, opID).Scan(&stepsBytes); err != nil {
			return err
		}
		var steps map[string]nodes.StoredStep
		_ = json.Unmarshal(stepsBytes, &steps)
		if st, ok := steps[nodes.StepAgentPolicy]; ok && st.Status != string(api.OperationStepStatusSucceeded) {
			errCode := string(apierr.DependencyUnavailable)
			errMsg := "central NetBird route withdrawn; node agent withdrawal unconfirmed"
			st.Status = string(api.OperationStepStatusFailed)
			st.ErrorCode = &errCode
			st.ErrorMessage = &errMsg
			st.UpdatedAt = now
			steps[nodes.StepAgentPolicy] = st
		}
		newStepsJSON, _ := json.Marshal(steps)
		_, err := tx.Exec(ctx, `
			UPDATE operations
			SET status = 'partially_succeeded',
			    steps = $2::jsonb,
			    error_code = $3,
			    error_message = $4,
			    resource_version = coalesce(resource_version, 0) + 1
			WHERE id = $1`,
			opID, newStepsJSON, string(apierr.DependencyUnavailable), "central NetBird route withdrawn; node agent withdrawal unconfirmed")
		return err
	})
	if err != nil {
		return api.Operation{}, err
	}
	return nodes.GetOperation(ctx, s.pool, opID)
}

// ReconcileItem describes the reconciliation result for a single exit.
type ReconcileItem struct {
	NodeID uuid.UUID
	ExitID uuid.UUID
	Drift  []api.StateDrift
}

// ReconcileNetBird compares observed NetBird routes/groups against external_bindings
// and exit publication_intent, recording any drift on external_bindings and nodes.drift (§17.4 item 3).
// It never alters exits.publication_intent (EXIT03).
func (s *Service) ReconcileNetBird(ctx context.Context) ([]ReconcileItem, error) {
	if s.netbird == nil {
		return nil, errors.New("exits: netbird adapter is required for reconciliation")
	}
	now := s.now().UTC()

	rows, err := s.pool.Query(ctx, `
		SELECT
			e.id,
			e.node_id,
			e.publication_intent,
			coalesce(inc.deployment_id, $1),
			coalesce(inc.peer_id, ''),
			coalesce(eb.external_id, ''),
			eb.desired_hash
		FROM exits e
		JOIN nodes n ON n.id = e.node_id
		LEFT JOIN node_incarnations inc ON inc.id = n.current_incarnation_id
		LEFT JOIN external_bindings eb
		  ON eb.resource_type = 'exit'
		 AND eb.resource_id = e.id
		 AND eb.provider = $2
		ORDER BY e.id ASC`, DefaultDeploymentID, ProviderNetBird)
	if err != nil {
		return nil, err
	}
	type exitRow struct {
		exitID       uuid.UUID
		nodeID       uuid.UUID
		intent       string
		deploymentID string
		peerID       string
		externalID   string
		desiredHash  []byte
	}
	var items []exitRow
	for rows.Next() {
		var r exitRow
		if err := rows.Scan(&r.exitID, &r.nodeID, &r.intent, &r.deploymentID, &r.peerID, &r.externalID, &r.desiredHash); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	var results []ReconcileItem
	for _, item := range items {
		obs, err := s.netbird.ReadExitState(ctx, item.deploymentID, item.exitID, item.externalID)
		if err != nil {
			return results, err
		}

		drift := []api.StateDrift{}
		wantPublished := item.intent == string(api.PublicationIntentPublished)
		if wantPublished {
			if !obs.Exists || !obs.Enabled {
				des := "published"
				obv := "withdrawn"
				drift = append(drift, api.StateDrift{
					Field:    "netbird_route",
					Desired:  presence.NewNullableWithValue(des),
					Observed: presence.NewNullableWithValue(obv),
				})
			} else if len(item.desiredHash) > 0 && len(obs.ObservedHash) > 0 && !bytes.Equal(item.desiredHash, obs.ObservedHash) {
				des := fmt.Sprintf("%x", item.desiredHash[:8])
				obv := fmt.Sprintf("%x", obs.ObservedHash[:8])
				drift = append(drift, api.StateDrift{
					Field:    "netbird_route_config",
					Desired:  presence.NewNullableWithValue(des),
					Observed: presence.NewNullableWithValue(obv),
				})
			}
		} else {
			if obs.Exists && obs.Enabled {
				des := item.intent
				obv := "published"
				drift = append(drift, api.StateDrift{
					Field:    "netbird_route",
					Desired:  presence.NewNullableWithValue(des),
					Observed: presence.NewNullableWithValue(obv),
				})
			}
		}

		driftJSON, _ := json.Marshal(drift)
		syncState := "synced"
		var lastErr *string
		if len(drift) > 0 {
			syncState = "drifted"
			msg := fmt.Sprintf("drift on %s", drift[0].Field)
			lastErr = &msg
		}

		err = db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `
				UPDATE nodes
				SET drift = $2::jsonb,
				    resource_version = resource_version + 1
				WHERE id = $1`, item.nodeID, driftJSON); err != nil {
				return err
			}
			if item.externalID != "" {
				if _, err := tx.Exec(ctx, `
					UPDATE external_bindings
					SET sync_state = $3,
					    observed_hash = coalesce($4, observed_hash),
					    last_verified_at = $5,
					    last_error = $6
					WHERE resource_type = 'exit'
					  AND resource_id = $1
					  AND provider = $2`,
					item.exitID, ProviderNetBird, syncState, obs.ObservedHash, now, lastErr); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return results, err
		}

		results = append(results, ReconcileItem{
			NodeID: item.nodeID,
			ExitID: item.exitID,
			Drift:  drift,
		})
	}
	return results, nil
}

// NodeService returns a nodes.Service sharing the same pool and clock.
func (s *Service) NodeService() *nodes.Service {
	return nodes.NewService(s.pool, nodes.WithClock(s.now))
}

func (s *Service) markStepSucceeded(ctx context.Context, opID uuid.UUID, stepName string, refValue string, now time.Time) error {
	return commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var (
			status     string
			stepsBytes []byte
			refsBytes  []byte
		)
		if err := tx.QueryRow(ctx, `
			SELECT status, steps, observed_refs
			FROM operations
			WHERE id = $1
			FOR UPDATE`, opID).Scan(&status, &stepsBytes, &refsBytes); err != nil {
			return err
		}

		var steps map[string]nodes.StoredStep
		if err := json.Unmarshal(stepsBytes, &steps); err != nil {
			return err
		}
		var refs map[string]string
		if len(refsBytes) > 0 {
			_ = json.Unmarshal(refsBytes, &refs)
		}
		if refs == nil {
			refs = map[string]string{}
		}

		st := steps[stepName]
		st.Status = string(api.OperationStepStatusSucceeded)
		st.ObservedRef = stepName
		st.ObservedAt = &now
		st.ErrorCode = nil
		st.ErrorMessage = nil
		st.Reason = ""
		st.UpdatedAt = now
		steps[stepName] = st
		refs[stepName] = refValue

		stepsJSON, _ := json.Marshal(steps)
		refsJSON, _ := json.Marshal(refs)

		if status == string(api.OperationStatusPending) || status == string(api.OperationStatusFailed) || status == string(api.OperationStatusPartiallySucceeded) {
			_, err := tx.Exec(ctx, `
				UPDATE operations
				SET status = 'running',
				    steps = $2::jsonb,
				    observed_refs = $3::jsonb,
				    resource_version = coalesce(resource_version, 0) + 1
				WHERE id = $1`, opID, stepsJSON, refsJSON)
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE operations
			SET steps = $2::jsonb,
			    observed_refs = $3::jsonb,
			    resource_version = coalesce(resource_version, 0) + 1
			WHERE id = $1`, opID, stepsJSON, refsJSON)
		return err
	})
}

func (s *Service) failPublishStep(
	ctx context.Context,
	opID uuid.UUID,
	nodeID uuid.UUID,
	stepName string,
	errCode string,
	errMsg string,
	gateReason ...string,
) (api.Operation, error) {
	return s.failPublishStepOutcome(ctx, opID, nodeID, stepName, errCode, errMsg, false, gateReason...)
}

func (s *Service) failPublishStepOutcome(ctx context.Context, opID, nodeID uuid.UUID, stepName, errCode, errMsg string, confirmedWithdrawn bool, gateReason ...string) (api.Operation, error) {
	now := s.now().UTC()
	err := commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var (
			status     string
			stepsBytes []byte
		)
		if err := tx.QueryRow(ctx, `
			SELECT status, steps
			FROM operations
			WHERE id = $1
			FOR UPDATE`, opID).Scan(&status, &stepsBytes); err != nil {
			return err
		}
		var steps map[string]nodes.StoredStep
		_ = json.Unmarshal(stepsBytes, &steps)
		if steps == nil {
			steps = map[string]nodes.StoredStep{}
		}
		st := steps[stepName]
		st.Status = string(api.OperationStepStatusFailed)
		st.ErrorCode = &errCode
		st.ErrorMessage = &errMsg
		if len(gateReason) != 0 {
			st.Reason = gateReason[0]
		}
		st.UpdatedAt = now
		steps[stepName] = st
		stepsJSON, _ := json.Marshal(steps)

		// Transition through running if currently pending/partially_succeeded/failed.
		if status != string(api.OperationStatusRunning) && status != string(api.OperationStatusFailed) {
			if _, err := tx.Exec(ctx, `
				UPDATE operations
				SET status = 'running',
				    resource_version = coalesce(resource_version, 0) + 1
				WHERE id = $1`, opID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE operations
			SET status = 'failed',
			    steps = $2::jsonb,
			    error_code = $3,
			    error_message = $4,
			    resource_version = coalesce(resource_version, 0) + 1
			WHERE id = $1`, opID, stepsJSON, errCode, errMsg); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE exits
			SET publication_observed = CASE WHEN $2::boolean THEN publication_observed ELSE 'error' END,
			    resource_version = resource_version + 1
			WHERE node_id = $1`, nodeID, confirmedWithdrawn); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return api.Operation{}, err
	}
	return nodes.GetOperation(ctx, s.pool, opID)
}

func (s *Service) lookupDeploymentID(ctx context.Context, nodeID uuid.UUID) (string, error) {
	var depID *string
	err := s.pool.QueryRow(ctx, `
		SELECT inc.deployment_id
		FROM nodes n
		LEFT JOIN node_incarnations inc ON inc.id = n.current_incarnation_id
		WHERE n.id = $1`, nodeID).Scan(&depID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if depID != nil && *depID != "" {
		return *depID, nil
	}
	return DefaultDeploymentID, nil
}

func (s *Service) lookupExternalBinding(ctx context.Context, exitID uuid.UUID, deploymentID string) (string, error) {
	var extID string
	err := s.pool.QueryRow(ctx, `
		SELECT external_id
		FROM external_bindings
		WHERE resource_type = 'exit'
		  AND resource_id = $1
		  AND provider = $2
		  AND deployment_id = $3`,
		exitID, ProviderNetBird, deploymentID).Scan(&extID)
	if err != nil {
		return "", err
	}
	return extID, nil
}

func (s *Service) upsertExternalBinding(
	ctx context.Context,
	exitID uuid.UUID,
	deploymentID string,
	externalID string,
	desiredHash []byte,
	observedHash []byte,
	syncState string,
	now time.Time,
) error {
	const q = `
		INSERT INTO external_bindings (
			resource_type, resource_id, provider, deployment_id,
			external_id, desired_hash, observed_hash, sync_state,
			last_observed_at, last_verified_at, last_error
		) VALUES (
			'exit', $1, $2, $3,
			$4, $5, $6, $7,
			$8, $8, NULL
		)
		ON CONFLICT (resource_type, resource_id, provider, deployment_id) DO UPDATE SET
			external_id = EXCLUDED.external_id,
			desired_hash = EXCLUDED.desired_hash,
			observed_hash = EXCLUDED.observed_hash,
			sync_state = EXCLUDED.sync_state,
			last_observed_at = EXCLUDED.last_observed_at,
			last_verified_at = EXCLUDED.last_verified_at,
			last_error = NULL`
	_, err := s.pool.Exec(ctx, q,
		exitID, ProviderNetBird, deploymentID,
		externalID, desiredHash, observedHash, syncState, now,
	)
	return err
}

func (s *Service) recordBindingError(
	ctx context.Context,
	exitID uuid.UUID,
	deploymentID string,
	externalID string,
	desiredHash []byte,
	errMsg string,
) error {
	if externalID == "" {
		return nil
	}
	const q = `
		INSERT INTO external_bindings (
			resource_type, resource_id, provider, deployment_id,
			external_id, desired_hash, sync_state, last_error
		) VALUES (
			'exit', $1, $2, $3,
			$4, $5, 'error', $6
		)
		ON CONFLICT (resource_type, resource_id, provider, deployment_id) DO UPDATE SET
			desired_hash = EXCLUDED.desired_hash,
			sync_state = 'error',
			last_error = EXCLUDED.last_error`
	_, err := s.pool.Exec(ctx, q, exitID, ProviderNetBird, deploymentID, externalID, desiredHash, errMsg)
	return err
}

func computeExitDesiredHash(exitID uuid.UUID, peerID string, enabled bool, groupIDs []uuid.UUID) ([]byte, error) {
	groups := make([]string, 0, len(groupIDs))
	for _, gid := range groupIDs {
		groups = append(groups, gid.String())
	}
	sort.Strings(groups)
	canon, err := canonjson.Marshal(map[string]any{
		"exit_id":   exitID.String(),
		"peer_id":   peerID,
		"enabled":   enabled,
		"group_ids": groups,
	})
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canon)
	return sum[:], nil
}
