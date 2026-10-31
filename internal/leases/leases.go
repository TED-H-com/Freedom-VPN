// Package leases implements serving lease calculation, signing, replay protection,
// and persistence for exit nodes (§14.4, §15.3, §16.3, AT29).
package leases

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultMaxOfflineLeaseSeconds is the recommended default offline serving lease cap (24h, §16.3).
const DefaultMaxOfflineLeaseSeconds = 86400

// ProvisionalGrantlessLeaseSeconds is the central expiry backstop (60s, D-067)
// when RenewTx issues a grant-less lease for a published exit whose
// provisioning_state is not yet ready. A ready node keeps the configured
// max_offline_lease_seconds window (D-065).
const ProvisionalGrantlessLeaseSeconds = 60

// Signer signs a canonical JSON payload for a given key purpose (`lease`).
type Signer interface {
	Sign(ctx context.Context, purpose envelope.Purpose, signed any) (envelope.Envelope, error)
}

// MemberLeaseInput represents a member's entitlement window and authorized devices for lease calculation (§16.3).
type MemberLeaseInput struct {
	MemberID            uuid.UUID
	GroupIDs            []uuid.UUID
	EntitlementRevision uint64
	AdminSuspended      bool
	ValidFrom           time.Time
	EffectiveServiceEnd time.Time
	DeviceIDs           []uuid.UUID
}

// EntitlementReader reads member entitlement deadlines for a node's exit groups.
type EntitlementReader interface {
	ListLeaseEntitlementsForGroups(ctx context.Context, q nodes.Querier, groupIDs []uuid.UUID, asOf time.Time) ([]MemberLeaseInput, error)
}

// FakeEntitlementReader is an in-memory EntitlementReader for unit and integration tests.
type FakeEntitlementReader struct {
	mu      sync.RWMutex
	members []MemberLeaseInput
}

// NewFakeEntitlementReader constructs a FakeEntitlementReader.
func NewFakeEntitlementReader(members ...MemberLeaseInput) *FakeEntitlementReader {
	return &FakeEntitlementReader{
		members: append([]MemberLeaseInput(nil), members...),
	}
}

// SetMembers replaces the in-memory member list.
func (f *FakeEntitlementReader) SetMembers(members ...MemberLeaseInput) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.members = append([]MemberLeaseInput(nil), members...)
}

// ListLeaseEntitlementsForGroups filters in-memory members by groupIDs.
func (f *FakeEntitlementReader) ListLeaseEntitlementsForGroups(_ context.Context, _ nodes.Querier, groupIDs []uuid.UUID, _ time.Time) ([]MemberLeaseInput, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if len(groupIDs) == 0 {
		return nil, nil
	}
	allowed := make(map[uuid.UUID]bool, len(groupIDs))
	for _, gid := range groupIDs {
		allowed[gid] = true
	}
	var out []MemberLeaseInput
	for _, m := range f.members {
		matched := false
		for _, mg := range m.GroupIDs {
			if allowed[mg] {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		cp := m
		cp.GroupIDs = append([]uuid.UUID(nil), m.GroupIDs...)
		cp.DeviceIDs = append([]uuid.UUID(nil), m.DeviceIDs...)
		out = append(out, cp)
	}
	return out, nil
}

// DBEntitlementReader reads member entitlements and authorized devices from PostgreSQL.
type DBEntitlementReader struct{}

// ListLeaseEntitlementsForGroups queries PostgreSQL for active entitlements assigned to groupIDs.
func (DBEntitlementReader) ListLeaseEntitlementsForGroups(ctx context.Context, q nodes.Querier, groupIDs []uuid.UUID, asOf time.Time) ([]MemberLeaseInput, error) {
	return queryLeaseEntitlementsForGroups(ctx, q, groupIDs, asOf, false)
}

func queryLeaseEntitlementsForGroups(
	ctx context.Context,
	q nodes.Querier,
	groupIDs []uuid.UUID,
	asOf time.Time,
	lockForShare bool,
) ([]MemberLeaseInput, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	asOf = asOf.UTC()

	var (
		serving []entitlement.ServingMember
		err     error
	)
	if lockForShare {
		serving, err = entitlement.ListServingMembersForGroupsForShare(ctx, q, groupIDs, asOf)
	} else {
		serving, err = entitlement.ListServingMembersForGroups(ctx, q, groupIDs, asOf)
	}
	if err != nil {
		return nil, err
	}
	if len(serving) == 0 {
		return nil, nil
	}

	list := make([]MemberLeaseInput, 0, len(serving))
	idxByMember := make(map[uuid.UUID]int, len(serving))
	memberIDs := make([]uuid.UUID, 0, len(serving))

	for _, sm := range serving {
		idxByMember[sm.MemberID] = len(list)
		list = append(list, MemberLeaseInput{
			MemberID:            sm.MemberID,
			GroupIDs:            append([]uuid.UUID(nil), sm.ExitGroupIDs...),
			EntitlementRevision: uint64(sm.Revision),
			AdminSuspended:      false,
			ValidFrom:           sm.ValidFrom.UTC(),
			EffectiveServiceEnd: sm.EffectiveServiceEnd.UTC(),
			DeviceIDs:           []uuid.UUID{},
		})
		memberIDs = append(memberIDs, sm.MemberID)
	}

	dRows, err := q.Query(ctx, `
		SELECT md.member_id, md.id
		FROM member_devices md
		WHERE md.member_id = ANY($1)
		  AND md.status = 'authorized'
		  AND md.revoked_at IS NULL
		ORDER BY md.member_id ASC, md.id ASC`, memberIDs)
	if err != nil {
		return nil, err
	}
	defer dRows.Close()

	for dRows.Next() {
		var mid, did uuid.UUID
		if err := dRows.Scan(&mid, &did); err != nil {
			return nil, err
		}
		if idx, ok := idxByMember[mid]; ok {
			list[idx].DeviceIDs = append(list[idx].DeviceIDs, did)
		}
	}
	if err := dRows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// ComputeMemberLeaseEnd computes the exact member lease expiration timestamp according to §16.3:
//
//	member_lease_end = min(signed_server_time + max_offline_lease, effective_service_end, node_lease_end)
func ComputeMemberLeaseEnd(
	signedServerTime time.Time,
	maxOfflineLease time.Duration,
	effectiveServiceEnd time.Time,
	nodeLeaseEnd time.Time,
) time.Time {
	offlineCap := signedServerTime.UTC().Add(maxOfflineLease)
	end := offlineCap
	if effectiveServiceEnd.UTC().Before(end) {
		end = effectiveServiceEnd.UTC()
	}
	if nodeLeaseEnd.UTC().Before(end) {
		end = nodeLeaseEnd.UTC()
	}
	return end
}

// Service issues and renews signed serving leases for exit nodes (§14.4, §16.3).
type Service struct {
	pool         *pgxpool.Pool
	signer       Signer
	entitlements EntitlementReader
	now          func() time.Time
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

// WithEntitlementReader sets a custom EntitlementReader.
func WithEntitlementReader(r EntitlementReader) Option {
	return func(s *Service) {
		if r != nil {
			s.entitlements = r
		}
	}
}

// NewService constructs a lease Service.
func NewService(pool *pgxpool.Pool, signer Signer, opts ...Option) *Service {
	s := &Service{
		pool:         pool,
		signer:       signer,
		entitlements: DBEntitlementReader{},
		now:          func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Renew processes `POST /agent/lease-renewals` (§14.4, §16.3).
//
// Invariants enforced:
//   - Authenticated mTLS identity (authNodeID, authIncarnationID) must match the request and active incarnation.
//   - Revoked or retired nodes/incarnations are rejected with 403 IDENTITY_REVOKED.
//   - Replaying the same renewal_request_id returns the exact same signed lease and expires_at without restarting TTL.
//   - An unpublished or disabled node receives a lease with zero member grants (`members: []`).
//   - Every member grant satisfies `member_lease_end = min(signed_server_time + max_offline_lease, effective_service_end, node_lease_end)`.
//   - Lease issuance never modifies any entitlement row or deadline (`paid_through`, `effective_service_end`, grants, grace).
func (s *Service) Renew(
	ctx context.Context,
	authNodeID uuid.UUID,
	authIncarnationID uuid.UUID,
	req api.LeaseRenewalRequest,
) (api.ServingLeaseEnvelope, error) {
	if s.signer == nil {
		return api.ServingLeaseEnvelope{}, errors.New("leases: signer is required")
	}
	if authNodeID == uuid.Nil || authIncarnationID == uuid.Nil {
		return api.ServingLeaseEnvelope{}, apierr.New(apierr.Unauthenticated, "authenticated node and incarnation identity required")
	}
	if req.NodeId != authNodeID || req.NodeIncarnationId != authIncarnationID {
		return api.ServingLeaseEnvelope{}, apierr.New(apierr.IncarnationMismatch, "lease request node_id or node_incarnation_id does not match mTLS identity")
	}
	if req.RenewalRequestId == uuid.Nil {
		return api.ServingLeaseEnvelope{}, apierr.New(apierr.MalformedRequest, "renewal_request_id is required")
	}

	var outEnv api.ServingLeaseEnvelope
	err := db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var err error
		outEnv, err = s.RenewTx(ctx, tx, authNodeID, authIncarnationID, req)
		return err
	})
	if err != nil {
		return api.ServingLeaseEnvelope{}, err
	}
	return outEnv, nil
}

// RenewTx processes `POST /agent/lease-renewals` inside the caller's transaction,
// holding FOR SHARE row locks on nodes, members, and entitlements until commit (§14.4, §16.3).
func (s *Service) RenewTx(
	ctx context.Context,
	tx pgx.Tx,
	authNodeID uuid.UUID,
	authIncarnationID uuid.UUID,
	req api.LeaseRenewalRequest,
) (api.ServingLeaseEnvelope, error) {
	if s.signer == nil {
		return api.ServingLeaseEnvelope{}, errors.New("leases: signer is required")
	}
	if authNodeID == uuid.Nil || authIncarnationID == uuid.Nil {
		return api.ServingLeaseEnvelope{}, apierr.New(apierr.Unauthenticated, "authenticated node and incarnation identity required")
	}
	if req.NodeId != authNodeID || req.NodeIncarnationId != authIncarnationID {
		return api.ServingLeaseEnvelope{}, apierr.New(apierr.IncarnationMismatch, "lease request node_id or node_incarnation_id does not match mTLS identity")
	}
	if req.RenewalRequestId == uuid.Nil {
		return api.ServingLeaseEnvelope{}, apierr.New(apierr.MalformedRequest, "renewal_request_id is required")
	}

	var outEnv api.ServingLeaseEnvelope
	// Lock fleet_config before nodes even for standalone Renew/RenewTx callers.
	// check_node_lease later takes FOR SHARE on the same fleet row (§16.3).
	maxOfflineSec := DefaultMaxOfflineLeaseSeconds
	if err := tx.QueryRow(ctx, `SELECT max_offline_lease_seconds FROM fleet_config LIMIT 1 FOR SHARE`).Scan(&maxOfflineSec); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return api.ServingLeaseEnvelope{}, err
	}
	if maxOfflineSec <= 0 {
		maxOfflineSec = DefaultMaxOfflineLeaseSeconds
	}
	maxOfflineDur := time.Duration(maxOfflineSec) * time.Second

	// 1. Verify node and incarnation identity and revocation status.
	var (
		lifecycleState       string
		provisioningState    string
		currentIncarnationID *uuid.UUID
		desiredGenStr        string
	)
	err := tx.QueryRow(ctx, `
		SELECT lifecycle_state, provisioning_state, current_incarnation_id, desired_config_generation::text
		FROM nodes
		WHERE id = $1
		FOR SHARE`, authNodeID).Scan(&lifecycleState, &provisioningState, &currentIncarnationID, &desiredGenStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.ServingLeaseEnvelope{}, apierr.Newf(apierr.ResourceNotFound, "node %s not found", authNodeID)
		}
		return api.ServingLeaseEnvelope{}, err
	}
	if lifecycleState == string(api.LifecycleStateRevoked) || lifecycleState == string(api.LifecycleStateRetired) {
		return api.ServingLeaseEnvelope{}, apierr.Newf(apierr.IdentityRevoked, "node %s is %s", authNodeID, lifecycleState)
	}
	if lifecycleState != string(api.LifecycleStateRegistered) {
		return api.ServingLeaseEnvelope{}, apierr.Newf(apierr.NodeNotServiceable, "node %s is not registered (state=%s)", authNodeID, lifecycleState)
	}

	var (
		incNodeID     uuid.UUID
		revokedAt     *time.Time
		mappingFrozen bool
	)
	err = tx.QueryRow(ctx, `
		SELECT node_id, revoked_at,
		       EXISTS(SELECT 1 FROM node_identity_endpoints WHERE incarnation_id=$1 AND mapping_frozen)
		FROM node_incarnations
		WHERE id = $1`, authIncarnationID).Scan(&incNodeID, &revokedAt, &mappingFrozen)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.ServingLeaseEnvelope{}, apierr.New(apierr.IncarnationMismatch, "incarnation is not registered")
		}
		return api.ServingLeaseEnvelope{}, err
	}
	if incNodeID != authNodeID {
		return api.ServingLeaseEnvelope{}, apierr.New(apierr.IncarnationMismatch, "incarnation belongs to a different node")
	}
	if revokedAt != nil {
		return api.ServingLeaseEnvelope{}, apierr.New(apierr.IdentityRevoked, "node incarnation has been revoked")
	}
	if currentIncarnationID == nil || *currentIncarnationID != authIncarnationID {
		return api.ServingLeaseEnvelope{}, apierr.New(apierr.IncarnationMismatch, "node incarnation is not active")
	}

	// 2. Check replay by (node_id, incarnation_id, renewal_request_id) (§16.3: replay never restarts TTL).
	// D-088: preserve the original signature even after the stored expiry is
	// clamped. Central validity checks use the row, not this replayed deadline.
	var existingEnvBytes []byte
	err = tx.QueryRow(ctx, `
		SELECT envelope
		FROM serving_leases
		WHERE node_id = $1 AND incarnation_id = $2 AND renewal_request_id = $3`,
		authNodeID, authIncarnationID, req.RenewalRequestId).Scan(&existingEnvBytes)
	if err == nil && len(existingEnvBytes) > 2 {
		if err := json.Unmarshal(existingEnvBytes, &outEnv); err != nil {
			return api.ServingLeaseEnvelope{}, err
		}
		return outEnv, nil
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return api.ServingLeaseEnvelope{}, err
	}

	// 3. Bound the lease by the locked fleet offline maximum (§16.3).
	issuedAt := s.now().UTC()
	nodeLeaseEnd := issuedAt.Add(maxOfflineDur)

	// 4. Read exit publication_intent and exit_groups.
	var (
		exitID       *uuid.UUID
		pubIntentStr string
	)
	err = tx.QueryRow(ctx, `
		SELECT id, publication_intent
		FROM exits
		WHERE node_id = $1`, authNodeID).Scan(&exitID, &pubIntentStr)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return api.ServingLeaseEnvelope{}, err
	}

	var groupIDs []uuid.UUID
	if exitID != nil {
		rows, err := tx.Query(ctx, `SELECT group_id FROM exit_groups WHERE exit_id = $1 ORDER BY group_id ASC`, *exitID)
		if err != nil {
			return api.ServingLeaseEnvelope{}, err
		}
		for rows.Next() {
			var gid uuid.UUID
			if err := rows.Scan(&gid); err != nil {
				rows.Close()
				return api.ServingLeaseEnvelope{}, err
			}
			groupIDs = append(groupIDs, gid)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return api.ServingLeaseEnvelope{}, err
		}
		rows.Close()
	}

	grants := []api.MemberLease{}
	var dbMemberLeases []dbMemberLease
	var maxAuthGen uint64 = 1

	publishedAndRegistered := pubIntentStr == string(api.PublicationIntentPublished) &&
		lifecycleState == string(api.LifecycleStateRegistered)
	nodeReady := provisioningState == string(api.ProvisioningStateReady) && !mappingFrozen
	includeMembers := publishedAndRegistered && nodeReady && len(groupIDs) > 0

	// D-067 central backstop: when includeMembers is false on a published exit because
	// provisioning_state is not ready or mapping is frozen (D-084), issue the grant-less lease with ExpiresAt 60s
	// after issued_at so an agent that waits for the renewal lead still renews promptly.
	// Never shorten the lease of a ready node.
	if publishedAndRegistered && !nodeReady {
		provisionalDur := time.Duration(ProvisionalGrantlessLeaseSeconds) * time.Second
		if provisionalDur < maxOfflineDur {
			nodeLeaseEnd = issuedAt.Add(provisionalDur)
		}
	}

	// Only issue member grants when exit publication_intent == 'published' and node is registered & ready (§16.3).
	if includeMembers {
		entitled, err := s.entitlements.ListLeaseEntitlementsForGroups(ctx, tx, groupIDs, issuedAt)
		if err != nil {
			return api.ServingLeaseEnvelope{}, err
		}
		grants, dbMemberLeases, maxAuthGen = buildLeaseGrants(entitled, issuedAt, maxOfflineDur, nodeLeaseEnd)
	}

	// Sign nodes.desired_config_generation; client applied_config_generation is an observation input only (§16.3).
	cfgGen, _ := strconv.ParseUint(desiredGenStr, 10, 64)

	leaseID := uuid.New()
	leaseDoc := api.ServingLease{
		SchemaVersion:           api.ServingLeaseSchemaVersion(1),
		LeaseId:                 leaseID,
		NodeId:                  authNodeID,
		NodeIncarnationId:       authIncarnationID,
		ConfigGeneration:        nodes.ParseU64(cfgGen),
		AuthorizationGeneration: nodes.ParseU64(maxAuthGen),
		IssuedAt:                issuedAt,
		ExpiresAt:               nodeLeaseEnd,
		MaxOfflineSeconds:       maxOfflineSec,
		Members:                 grants,
	}

	var (
		keyID       string
		payloadHash []byte
		envJSON     []byte
	)
	outEnv, keyID, payloadHash, envJSON, err = s.signLeaseDoc(ctx, leaseDoc)
	if err != nil {
		return api.ServingLeaseEnvelope{}, err
	}

	// Lock eligible member and entitlement rows (FOR SHARE) and hold the locks until commit.
	// If the eligible set differs after signing, discard the signature, rebuild the document, and sign again (§16.3).
	if includeMembers {
		var lockedEntitled []MemberLeaseInput
		if _, isDB := s.entitlements.(DBEntitlementReader); isDB {
			lockedEntitled, err = queryLeaseEntitlementsForGroups(ctx, tx, groupIDs, issuedAt, true)
		} else {
			lockedEntitled, err = s.entitlements.ListLeaseEntitlementsForGroups(ctx, tx, groupIDs, issuedAt)
		}
		if err != nil {
			return api.ServingLeaseEnvelope{}, err
		}
		lockedGrants, lockedDBMemberLeases, lockedMaxAuthGen := buildLeaseGrants(lockedEntitled, issuedAt, maxOfflineDur, nodeLeaseEnd)
		if !sameLeaseGrants(dbMemberLeases, lockedDBMemberLeases) {
			grants = lockedGrants
			dbMemberLeases = lockedDBMemberLeases
			maxAuthGen = lockedMaxAuthGen
			leaseDoc.AuthorizationGeneration = nodes.ParseU64(maxAuthGen)
			leaseDoc.Members = grants
			outEnv, keyID, payloadHash, envJSON, err = s.signLeaseDoc(ctx, leaseDoc)
			if err != nil {
				return api.ServingLeaseEnvelope{}, err
			}
		}
	}

	grantRefs := make([]uuid.UUID, 0, len(dbMemberLeases))
	for _, ml := range dbMemberLeases {
		grantRefs = append(grantRefs, ml.grantRefID)
	}

	const insertLease = `
		INSERT INTO serving_leases (
			id, node_id, incarnation_id, policy_generation,
			authorization_generation, member_grant_refs,
			issued_at, expires_at, key_id, payload_hash,
			renewal_request_id, envelope
		) VALUES (
			$1, $2, $3, $4::numeric,
			$5::numeric, $6,
			$7, $8, $9, $10,
			$11, $12::jsonb
		)`
	if _, err := tx.Exec(ctx, insertLease,
		leaseID, authNodeID, authIncarnationID, strconv.FormatUint(cfgGen, 10),
		strconv.FormatUint(maxAuthGen, 10), grantRefs,
		issuedAt, nodeLeaseEnd, keyID, payloadHash,
		req.RenewalRequestId, envJSON,
	); err != nil {
		return api.ServingLeaseEnvelope{}, err
	}

	// Persist serving_lease_members for every member in the final signed grant list in the same transaction;
	// any failed member-row insert fails the whole renewal (F-11, §15.3, §16.3).
	for _, ml := range dbMemberLeases {
		if _, err := tx.Exec(ctx, `
			INSERT INTO serving_lease_members (
				lease_id, member_id, entitlement_revision, valid_from, expires_at
			) VALUES (
				$1, $2, $3::numeric, $4, $5
			)`,
			leaseID, ml.memberID, strconv.FormatUint(ml.revision, 10), ml.validFrom, ml.expiresAt,
		); err != nil {
			return api.ServingLeaseEnvelope{}, err
		}
	}

	afterJSON, _ := json.Marshal(map[string]any{
		"lease_id":                 leaseID.String(),
		"config_generation":        strconv.FormatUint(cfgGen, 10),
		"authorization_generation": strconv.FormatUint(maxAuthGen, 10),
		"member_grants":            len(dbMemberLeases),
	})
	if err := db.AppendAudit(ctx, tx, db.AuditEvent{
		Actor:         authNodeID.String(),
		ActorType:     string(api.AuditEventActorTypeNode),
		Action:        "node.lease_renewed",
		Resource:      "node:" + authNodeID.String(),
		Result:        string(api.AuditEventResultSucceeded),
		SchemaVersion: 1,
		Before:        []byte(`{}`),
		After:         afterJSON,
	}); err != nil {
		return api.ServingLeaseEnvelope{}, err
	}

	return outEnv, nil
}

type dbMemberLease struct {
	memberID            uuid.UUID
	revision            uint64
	effectiveServiceEnd time.Time
	validFrom           time.Time
	expiresAt           time.Time
	grantRefID          uuid.UUID
}

func buildLeaseGrants(
	entitled []MemberLeaseInput,
	issuedAt time.Time,
	maxOfflineDur time.Duration,
	nodeLeaseEnd time.Time,
) ([]api.MemberLease, []dbMemberLease, uint64) {
	sorted := append([]MemberLeaseInput(nil), entitled...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].MemberID.String() < sorted[j].MemberID.String()
	})
	grants := []api.MemberLease{}
	var dbMemberLeases []dbMemberLease
	var maxAuthGen uint64 = 1

	for _, m := range sorted {
		if m.AdminSuspended {
			continue
		}
		if !m.EffectiveServiceEnd.UTC().After(issuedAt) {
			continue
		}
		validFrom := issuedAt
		if m.ValidFrom.UTC().After(validFrom) {
			validFrom = m.ValidFrom.UTC()
		}
		memberLeaseEnd := ComputeMemberLeaseEnd(issuedAt, maxOfflineDur, m.EffectiveServiceEnd, nodeLeaseEnd)
		if !memberLeaseEnd.After(validFrom) {
			continue
		}

		rev := m.EntitlementRevision
		if rev > maxAuthGen {
			maxAuthGen = rev
		}

		grants = append(grants, api.MemberLease{
			MemberId:            m.MemberID,
			EntitlementRevision: nodes.ParseU64(rev),
			EffectiveServiceEnd: m.EffectiveServiceEnd.UTC(),
			LeaseEnd:            memberLeaseEnd,
		})
		dbMemberLeases = append(dbMemberLeases, dbMemberLease{
			memberID:            m.MemberID,
			revision:            rev,
			effectiveServiceEnd: m.EffectiveServiceEnd.UTC(),
			validFrom:           validFrom,
			expiresAt:           memberLeaseEnd,
			grantRefID:          m.MemberID,
		})
	}
	return grants, dbMemberLeases, maxAuthGen
}

func sameLeaseGrants(a, b []dbMemberLease) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].memberID != b[i].memberID ||
			a[i].revision != b[i].revision ||
			!a[i].effectiveServiceEnd.Equal(b[i].effectiveServiceEnd) ||
			!a[i].validFrom.Equal(b[i].validFrom) ||
			!a[i].expiresAt.Equal(b[i].expiresAt) {
			return false
		}
	}
	return true
}

func (s *Service) signLeaseDoc(
	ctx context.Context,
	leaseDoc api.ServingLease,
) (api.ServingLeaseEnvelope, string, []byte, []byte, error) {
	leaseCanon, err := canonjson.Marshal(leaseDoc)
	if err != nil {
		return api.ServingLeaseEnvelope{}, "", nil, nil, err
	}
	payloadHashArr := sha256.Sum256(leaseCanon)
	rawEnv, err := s.signer.Sign(ctx, envelope.PurposeLease, leaseDoc)
	if err != nil {
		return api.ServingLeaseEnvelope{}, "", nil, nil, err
	}
	sigs := make([]api.Signature, 0, len(rawEnv.Signatures))
	keyID := "lease-key"
	for i, sig := range rawEnv.Signatures {
		if i == 0 && sig.KeyID != "" {
			keyID = sig.KeyID
		}
		sigs = append(sigs, api.Signature{
			Algorithm:    api.SignatureAlgorithm(sig.Algorithm),
			KeyId:        sig.KeyID,
			SignatureB64: sig.SignatureB64,
		})
	}
	outEnv := api.ServingLeaseEnvelope{
		Signed:     leaseDoc,
		Signatures: sigs,
	}
	envJSON, err := json.Marshal(outEnv)
	if err != nil {
		return api.ServingLeaseEnvelope{}, "", nil, nil, err
	}
	return outEnv, keyID, payloadHashArr[:], envJSON, nil
}
