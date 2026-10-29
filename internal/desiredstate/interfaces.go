// Package desiredstate compiles, signs, stores, and serves per-node desired-state
// envelopes and handles outbox-driven recompilation (§5, §8.1, §9.2, §10.2, §11.2, §13.2, §14.4, §14.7).
package desiredstate

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Default policy constants and recommended defaults (§5, §8.1, §14.7).
const (
	DefaultSafetyProfile        = "internet-only-v1"
	DefaultDNSResolverIPv4      = "100.64.0.53"
	DefaultLeaseRenewalEndpoint = "/api/v1/agent/lease-renewals"
	DefaultEnvelopeTTL          = 24 * time.Hour
	DefaultPlanRateBps          = int64(100_000_000) // TEST ONLY default when plan payload omits rate
)

// DefaultSupportedReleasePolicyUUID is used when no release_support policy revision is active yet.
var DefaultSupportedReleasePolicyUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// SigningKey holds an Ed25519 private key and its key_id and allowed purpose (§16.2).
type SigningKey struct {
	KeyID      string
	Purpose    envelope.Purpose
	PrivateKey ed25519.PrivateKey
}

// Signer signs a canonical JSON payload for a given key purpose (`desired-state`, `lease`, etc.).
type Signer interface {
	Sign(ctx context.Context, purpose envelope.Purpose, signed any) (envelope.Envelope, error)
}

// KeySigner implements Signer using one or more Ed25519 private keys via pkg/envelope.
type KeySigner struct {
	Keys []SigningKey
}

// NewKeySigner creates a Signer backed by the provided Ed25519 private keys.
func NewKeySigner(keys ...SigningKey) *KeySigner {
	return &KeySigner{Keys: keys}
}

// Sign signs signed for purpose using pkg/envelope.Sign.
func (s *KeySigner) Sign(_ context.Context, purpose envelope.Purpose, signed any) (envelope.Envelope, error) {
	if !purpose.Valid() {
		return envelope.Envelope{}, fmt.Errorf("invalid signing purpose %q", purpose)
	}
	var env envelope.Envelope
	signedCount := 0
	for _, k := range s.Keys {
		if k.Purpose != "" && k.Purpose != purpose {
			continue
		}
		if signedCount == 0 {
			var err error
			env, err = envelope.Sign(signed, k.PrivateKey, k.KeyID)
			if err != nil {
				return envelope.Envelope{}, err
			}
		} else {
			if err := env.AddSignature(k.PrivateKey, k.KeyID); err != nil {
				return envelope.Envelope{}, err
			}
		}
		signedCount++
	}
	if signedCount == 0 {
		return envelope.Envelope{}, fmt.Errorf("no signing key configured for purpose %q", purpose)
	}
	return env, nil
}

// MemberDeviceBinding represents a member device's authorized overlay binding (§10.2, §14.7).
type MemberDeviceBinding struct {
	DeviceID        uuid.UUID
	NetbirdPeerID   string
	AllowedIPv4     []string
	AllowedIPv6     []string
	BindingRevision uint64
	ValidFrom       time.Time
	ValidUntil      time.Time
	Revoked         bool
}

// MemberEntitlement represents a member's entitlement, group membership, rate limits,
// and device bindings as consumed by the desired-state compiler (§9.2, §10.2, §11.2, §14.7).
type MemberEntitlement struct {
	MemberID            uuid.UUID
	GroupIDs            []uuid.UUID
	Suspended           bool
	ClampedUntil        *time.Time
	ValidFrom           time.Time
	EffectiveServiceEnd time.Time
	PlanDownBps         int64
	PlanUpBps           int64
	FairUseTierDownBps  int64
	FairUseTierUpBps    int64
	TierIndex           int
	Priority            int
	FairUsePolicyGen    uint64
	Mode                api.MemberRatePolicyMode
	Reason              api.MemberRatePolicyReason
	Bindings            []MemberDeviceBinding
}

// EntitlementReader reads entitled members and their device bindings for a set of member groups.
type EntitlementReader interface {
	ListEntitledMembersForGroups(ctx context.Context, groupIDs []uuid.UUID, asOf time.Time) ([]MemberEntitlement, error)
}

// PolicySnapshot holds fleet-wide and node-specific policy inputs for the desired-state compiler.
type PolicySnapshot struct {
	AuthorizationGeneration  uint64
	FairUseGeneration        uint64
	DNSGeneration            uint64
	DNSResolverIPv4          []string
	DNSResolverIPv6          []string
	DesiredReleaseID         *uuid.UUID
	SupportedReleasePolicyID uuid.UUID
	ServingReleaseIDs        []uuid.UUID
	PolicyRefs               []uuid.UUID
	AccountingPeriod         api.AccountingPeriod
	UpdateIntent             *api.UpdateIntent
}

// PolicyReader provides fleet and node policy inputs to the desired-state compiler.
type PolicyReader interface {
	ReadPolicySnapshot(ctx context.Context, nodeID uuid.UUID, asOf time.Time) (PolicySnapshot, error)
}

// DefaultAccountingPeriod returns the UTC calendar-month accounting period containing t (§10.1, §14.7).
func DefaultAccountingPeriod(t time.Time) api.AccountingPeriod {
	utc := t.UTC()
	start := time.Date(utc.Year(), utc.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return api.AccountingPeriod{
		PeriodId:    fmt.Sprintf("%04d-%02d", start.Year(), int(start.Month())),
		PeriodStart: start,
		PeriodEnd:   end,
	}
}

// FakeEntitlementReader is an in-memory EntitlementReader for unit and integration tests.
type FakeEntitlementReader struct {
	mu      sync.RWMutex
	members []MemberEntitlement
}

// NewFakeEntitlementReader constructs a FakeEntitlementReader.
func NewFakeEntitlementReader(members ...MemberEntitlement) *FakeEntitlementReader {
	return &FakeEntitlementReader{
		members: append([]MemberEntitlement(nil), members...),
	}
}

// SetMembers replaces the in-memory member entitlement list.
func (f *FakeEntitlementReader) SetMembers(members ...MemberEntitlement) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.members = append([]MemberEntitlement(nil), members...)
}

// ListEntitledMembersForGroups filters in-memory members by group membership.
func (f *FakeEntitlementReader) ListEntitledMembersForGroups(_ context.Context, groupIDs []uuid.UUID, _ time.Time) ([]MemberEntitlement, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if len(groupIDs) == 0 {
		return nil, nil
	}
	allowedGroups := make(map[uuid.UUID]bool, len(groupIDs))
	for _, gid := range groupIDs {
		allowedGroups[gid] = true
	}
	var out []MemberEntitlement
	for _, m := range f.members {
		matched := false
		for _, mg := range m.GroupIDs {
			if allowedGroups[mg] {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		cp := m
		cp.GroupIDs = append([]uuid.UUID(nil), m.GroupIDs...)
		cp.Bindings = append([]MemberDeviceBinding(nil), m.Bindings...)
		out = append(out, cp)
	}
	return out, nil
}

// FakePolicyReader is an in-memory PolicyReader for unit and integration tests.
type FakePolicyReader struct {
	mu          sync.RWMutex
	snapshot    PolicySnapshot
	nodeUpdates map[uuid.UUID]*api.UpdateIntent
}

// NewFakePolicyReader constructs a FakePolicyReader with sensible test defaults.
func NewFakePolicyReader(snap PolicySnapshot) *FakePolicyReader {
	if snap.AuthorizationGeneration == 0 {
		snap.AuthorizationGeneration = 1
	}
	if snap.FairUseGeneration == 0 {
		snap.FairUseGeneration = 1
	}
	if snap.DNSGeneration == 0 {
		snap.DNSGeneration = 1
	}
	if len(snap.DNSResolverIPv4) == 0 {
		snap.DNSResolverIPv4 = []string{DefaultDNSResolverIPv4}
	}
	if snap.SupportedReleasePolicyID == uuid.Nil {
		snap.SupportedReleasePolicyID = DefaultSupportedReleasePolicyUUID
	}
	return &FakePolicyReader{
		snapshot:    snap,
		nodeUpdates: make(map[uuid.UUID]*api.UpdateIntent),
	}
}

// SetSnapshot updates the policy snapshot.
func (f *FakePolicyReader) SetSnapshot(snap PolicySnapshot) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.snapshot = snap
}

// SetNodeUpdateIntent sets or clears the update intent for a specific node.
func (f *FakePolicyReader) SetNodeUpdateIntent(nodeID uuid.UUID, intent *api.UpdateIntent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if intent == nil {
		delete(f.nodeUpdates, nodeID)
	} else {
		cp := *intent
		f.nodeUpdates[nodeID] = &cp
	}
}

// ReadPolicySnapshot returns the current PolicySnapshot for nodeID.
func (f *FakePolicyReader) ReadPolicySnapshot(_ context.Context, nodeID uuid.UUID, asOf time.Time) (PolicySnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	snap := f.snapshot
	snap.DNSResolverIPv4 = append([]string(nil), f.snapshot.DNSResolverIPv4...)
	snap.DNSResolverIPv6 = append([]string(nil), f.snapshot.DNSResolverIPv6...)
	snap.ServingReleaseIDs = append([]uuid.UUID(nil), f.snapshot.ServingReleaseIDs...)
	snap.PolicyRefs = append([]uuid.UUID(nil), f.snapshot.PolicyRefs...)
	if snap.AccountingPeriod.PeriodId == "" {
		snap.AccountingPeriod = DefaultAccountingPeriod(asOf)
	}
	if u, ok := f.nodeUpdates[nodeID]; ok && u != nil {
		cp := *u
		snap.UpdateIntent = &cp
	}
	return snap, nil
}

// DBEntitlementReader reads member entitlements, group assignments, and device bindings from PostgreSQL.
type DBEntitlementReader struct {
	q nodes.Querier
}

// NewDBEntitlementReader constructs a DBEntitlementReader backed by q.
func NewDBEntitlementReader(q nodes.Querier) *DBEntitlementReader {
	return &DBEntitlementReader{q: q}
}

// ListEntitledMembersForGroups queries PostgreSQL for members assigned to groupIDs and their active device bindings.
func (r *DBEntitlementReader) ListEntitledMembersForGroups(ctx context.Context, groupIDs []uuid.UUID, asOf time.Time) ([]MemberEntitlement, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	asOf = asOf.UTC()

	const memberQuery = `
		SELECT DISTINCT
			m.id,
			m.status,
			(m.admin_suspended OR e.admin_suspended) AS is_suspended,
			e.valid_from,
			e.effective_service_end,
			coalesce(pr.payload, '{}'::jsonb)
		FROM members m
		JOIN entitlements e ON e.member_id = m.id
		LEFT JOIN subscriptions sub ON sub.member_id = m.id AND sub.is_active = true
		LEFT JOIN plan_revisions pr ON pr.id = sub.plan_revision_id
		JOIN assignments a ON a.group_id = ANY($1)
			AND a.valid_from <= $2
			AND (a.valid_until IS NULL OR a.valid_until > $2)
			AND (a.member_id = m.id OR (sub.plan_revision_id IS NOT NULL AND a.plan_revision_id = sub.plan_revision_id))
		WHERE m.status IN ('active', 'suspended')
		  AND e.valid_from IS NOT NULL
		  AND e.effective_service_end IS NOT NULL
		ORDER BY m.id ASC`

	rows, err := r.q.Query(ctx, memberQuery, groupIDs, asOf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []MemberEntitlement
	memberIdx := map[uuid.UUID]int{}
	var memberIDs []uuid.UUID

	for rows.Next() {
		var (
			memberID       uuid.UUID
			memberStatus   string
			adminSuspended bool
			validFrom      time.Time
			serviceEnd     time.Time
			planPayload    []byte
		)
		if err := rows.Scan(&memberID, &memberStatus, &adminSuspended, &validFrom, &serviceEnd, &planPayload); err != nil {
			return nil, err
		}
		suspended := memberStatus == "suspended" || adminSuspended
		if !suspended && !serviceEnd.After(asOf) {
			continue
		}

		planDown := DefaultPlanRateBps
		planUp := DefaultPlanRateBps
		prio := 1
		if len(planPayload) > 0 {
			var parsed struct {
				DownBps   int64 `json:"down_bps"`
				UpBps     int64 `json:"up_bps"`
				Priority  int   `json:"priority"`
				RateLimit *struct {
					DownBps int64 `json:"down_bps"`
					UpBps   int64 `json:"up_bps"`
				} `json:"rate_limit"`
			}
			if err := json.Unmarshal(planPayload, &parsed); err == nil {
				if parsed.DownBps > 0 {
					planDown = parsed.DownBps
				} else if parsed.RateLimit != nil && parsed.RateLimit.DownBps > 0 {
					planDown = parsed.RateLimit.DownBps
				}
				if parsed.UpBps > 0 {
					planUp = parsed.UpBps
				} else if parsed.RateLimit != nil && parsed.RateLimit.UpBps > 0 {
					planUp = parsed.RateLimit.UpBps
				}
				if parsed.Priority >= 0 && parsed.Priority <= 7 {
					prio = parsed.Priority
				}
			}
		}

		var clamped *time.Time
		if suspended {
			clampedTime := asOf
			if serviceEnd.Before(clampedTime) {
				clampedTime = serviceEnd
			}
			clamped = &clampedTime
		}

		m := MemberEntitlement{
			MemberID:            memberID,
			GroupIDs:            append([]uuid.UUID(nil), groupIDs...),
			Suspended:           suspended,
			ClampedUntil:        clamped,
			ValidFrom:           validFrom.UTC(),
			EffectiveServiceEnd: serviceEnd.UTC(),
			PlanDownBps:         planDown,
			PlanUpBps:           planUp,
			FairUseTierDownBps:  planDown,
			FairUseTierUpBps:    planUp,
			Priority:            prio,
			FairUsePolicyGen:    1,
			Mode:                api.MemberRatePolicyModeEnforce,
			Reason:              api.MemberRatePolicyReasonNormal,
		}
		memberIdx[memberID] = len(members)
		members = append(members, m)
		memberIDs = append(memberIDs, memberID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if len(memberIDs) == 0 {
		return nil, nil
	}

	const bindingQuery = `
		SELECT
			db.member_id,
			db.device_id,
			md.peer_id,
			host(db.overlay_ip),
			family(db.overlay_ip),
			db.binding_revision,
			db.valid_from,
			db.valid_until,
			(md.status <> 'authorized' OR md.revoked_at IS NOT NULL) AS is_revoked
		FROM device_bindings db
		JOIN member_devices md ON md.id = db.device_id
		WHERE db.member_id = ANY($1)
		  AND db.valid_from <= $2
		  AND db.valid_until > $2
		ORDER BY db.member_id ASC, db.device_id ASC, db.binding_revision DESC`

	bRows, err := r.q.Query(ctx, bindingQuery, memberIDs, asOf)
	if err != nil {
		return nil, err
	}
	defer bRows.Close()

	for bRows.Next() {
		var (
			memberID  uuid.UUID
			deviceID  uuid.UUID
			peerID    string
			overlayIP string
			ipFamily  int
			rev       uint64
			validFrom time.Time
			validTo   time.Time
			revoked   bool
		)
		if err := bRows.Scan(&memberID, &deviceID, &peerID, &overlayIP, &ipFamily, &rev, &validFrom, &validTo, &revoked); err != nil {
			return nil, err
		}
		idx, ok := memberIdx[memberID]
		if !ok {
			continue
		}
		var v4, v6 []string
		if ipFamily == 6 {
			v6 = []string{overlayIP}
		} else {
			v4 = []string{overlayIP}
		}
		members[idx].Bindings = append(members[idx].Bindings, MemberDeviceBinding{
			DeviceID:        deviceID,
			NetbirdPeerID:   peerID,
			AllowedIPv4:     v4,
			AllowedIPv6:     v6,
			BindingRevision: rev,
			ValidFrom:       validFrom.UTC(),
			ValidUntil:      validTo.UTC(),
			Revoked:         revoked,
		})
	}
	if err := bRows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}

// DBPolicyReader reads policy revisions and fleet configuration from PostgreSQL.
type DBPolicyReader struct {
	q nodes.Querier
}

// NewDBPolicyReader constructs a DBPolicyReader backed by q.
func NewDBPolicyReader(q nodes.Querier) *DBPolicyReader {
	return &DBPolicyReader{q: q}
}

// ReadPolicySnapshot reads active fair-use, DNS, and release support policies from PostgreSQL.
func (r *DBPolicyReader) ReadPolicySnapshot(ctx context.Context, nodeID uuid.UUID, asOf time.Time) (PolicySnapshot, error) {
	snap := PolicySnapshot{
		AuthorizationGeneration:  1,
		FairUseGeneration:        1,
		DNSGeneration:            1,
		DNSResolverIPv4:          []string{DefaultDNSResolverIPv4},
		DNSResolverIPv6:          []string{},
		SupportedReleasePolicyID: DefaultSupportedReleasePolicyUUID,
		ServingReleaseIDs:        []uuid.UUID{},
		AccountingPeriod:         DefaultAccountingPeriod(asOf),
	}

	var (
		desiredRelUUID  *uuid.UUID
		supportedRelIDs []uuid.UUID
	)
	err := r.q.QueryRow(ctx, `
		SELECT desired_release_id, supported_release_ids
		FROM fleet_config
		LIMIT 1`).Scan(&desiredRelUUID, &supportedRelIDs)
	if err != nil && err != pgx.ErrNoRows {
		return PolicySnapshot{}, err
	}

	// Check node-specific desired_release_id override first.
	var nodeDesiredRel *uuid.UUID
	if err := r.q.QueryRow(ctx, `SELECT desired_release_id FROM nodes WHERE id = $1`, nodeID).Scan(&nodeDesiredRel); err == nil && nodeDesiredRel != nil {
		desiredRelUUID = nodeDesiredRel
	}
	if desiredRelUUID == nil {
		var rid uuid.UUID
		if err := r.q.QueryRow(ctx, `SELECT id FROM releases WHERE state = 'approved' ORDER BY release_sequence DESC LIMIT 1`).Scan(&rid); err == nil {
			desiredRelUUID = &rid
		}
	}
	if desiredRelUUID != nil {
		snap.DesiredReleaseID = desiredRelUUID
		if len(supportedRelIDs) > 0 {
			snap.ServingReleaseIDs = append([]uuid.UUID(nil), supportedRelIDs...)
		} else {
			snap.ServingReleaseIDs = []uuid.UUID{*desiredRelUUID}
		}
	} else if len(supportedRelIDs) > 0 {
		snap.ServingReleaseIDs = append([]uuid.UUID(nil), supportedRelIDs...)
	}

	pRows, err := r.q.Query(ctx, `
		SELECT pd.type, pd.active_revision_id, greatest(pd.generation, pr.revision)::text, pr.payload
		FROM policy_definitions pd
		JOIN policy_revisions pr ON pr.id = pd.active_revision_id
		WHERE pd.active_revision_id IS NOT NULL
		ORDER BY pd.type ASC, pd.id ASC`)
	if err != nil {
		return PolicySnapshot{}, err
	}
	defer pRows.Close()

	for pRows.Next() {
		var (
			pType   string
			revID   uuid.UUID
			genStr  string
			payload []byte
		)
		if err := pRows.Scan(&pType, &revID, &genStr, &payload); err != nil {
			return PolicySnapshot{}, err
		}
		gen, _ := strconv.ParseUint(genStr, 10, 64)
		snap.PolicyRefs = append(snap.PolicyRefs, revID)
		switch pType {
		case "fair_use":
			if gen > snap.FairUseGeneration {
				snap.FairUseGeneration = gen
			}
		case "dns_block":
			if gen > snap.DNSGeneration {
				snap.DNSGeneration = gen
			}
			var dnsPayload struct {
				ResolverIPv4 []string `json:"resolver_ipv4"`
				ResolverIPv6 []string `json:"resolver_ipv6"`
			}
			if json.Unmarshal(payload, &dnsPayload) == nil && len(dnsPayload.ResolverIPv4) > 0 {
				snap.DNSResolverIPv4 = dnsPayload.ResolverIPv4
				snap.DNSResolverIPv6 = dnsPayload.ResolverIPv6
			}
		case "release_support":
			snap.SupportedReleasePolicyID = revID
			var relPol api.ReleaseSupportPayload
			if json.Unmarshal(payload, &relPol) == nil && len(relPol.MinimumServingVersion.ServingReleaseIds) > 0 {
				snap.ServingReleaseIDs = relPol.MinimumServingVersion.ServingReleaseIds
			}
		}
	}
	if err := pRows.Err(); err != nil {
		return PolicySnapshot{}, err
	}

	return snap, nil
}
