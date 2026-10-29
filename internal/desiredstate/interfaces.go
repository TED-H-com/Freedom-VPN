// Package desiredstate compiles, signs, stores, and serves per-node desired-state
// envelopes and handles outbox-driven recompilation (§5, §8.1, §9.2, §10.2, §11.2, §13.2, §14.4, §14.7).
package desiredstate

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Default policy constants and recommended defaults (§5, §8.1, §14.7).
const (
	DefaultSafetyProfile           = "internet-only-v1"
	TestOnlyDefaultDNSResolverIPv4 = "100.64.0.53" // TEST ONLY default for FakePolicyReader
	DefaultLeaseRenewalEndpoint    = "/api/v1/agent/lease-renewals"
	DefaultEnvelopeTTL             = 24 * time.Hour
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
	OverrideValidUntil  *time.Time
	ValidFrom           time.Time
	EffectiveServiceEnd time.Time
	PlanDownBps         int64
	PlanUpBps           int64
	FairUseTierDownBps  int64
	FairUseTierUpBps    int64
	TierIndex           int
	Priority            int
	FairUsePolicyGen    uint64
	HasFairUseGen       bool
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
	if len(snap.DNSResolverIPv4) == 0 {
		snap.DNSResolverIPv4 = []string{TestOnlyDefaultDNSResolverIPv4}
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

// ListEntitledMembersForGroups queries PostgreSQL for members assigned to groupIDs,
// their plan rate limits, their per-period fair-use state, and their active device bindings (§9.2, §10.2, §11.2).
func (r *DBEntitlementReader) ListEntitledMembersForGroups(ctx context.Context, groupIDs []uuid.UUID, asOf time.Time) ([]MemberEntitlement, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	asOf = asOf.UTC()
	period := DefaultAccountingPeriod(asOf)

	defaultFUPMode := api.MemberRatePolicyModeEnforce
	var (
		activeFUPGenStr  string
		activeFUPPayload []byte
	)
	err := r.q.QueryRow(ctx, `
		SELECT greatest(pd.generation, pr.revision)::text, pr.payload
		FROM policy_definitions pd
		JOIN policy_revisions pr ON pr.id = pd.active_revision_id
		WHERE pd.type = 'fair_use' AND pd.active_revision_id IS NOT NULL
		ORDER BY pd.id ASC
		LIMIT 1`).Scan(&activeFUPGenStr, &activeFUPPayload)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil && len(activeFUPPayload) > 0 {
		var fupPol struct {
			Mode        string     `json:"mode"`
			EffectiveAt *time.Time `json:"effective_at"`
		}
		if json.Unmarshal(activeFUPPayload, &fupPol) == nil {
			if fupPol.Mode == string(api.FairUsePayloadModeShadow) ||
				(fupPol.EffectiveAt != nil && !fupPol.EffectiveAt.IsZero() && asOf.Before(fupPol.EffectiveAt.UTC())) {
				defaultFUPMode = api.MemberRatePolicyModeShadow
			}
		}
	}

	serving, err := entitlement.ListServingMembersForGroups(ctx, r.q, groupIDs, asOf)
	if err != nil {
		return nil, err
	}
	if len(serving) == 0 {
		return nil, nil
	}

	members := make([]MemberEntitlement, 0, len(serving))
	memberIdx := make(map[uuid.UUID]int, len(serving))
	memberIDs := make([]uuid.UUID, 0, len(serving))

	for _, sm := range serving {
		var planDown, planUp int64
		if sm.PlanRateLimit != nil {
			planDown = sm.PlanRateLimit.DownBps
			planUp = sm.PlanRateLimit.UpBps
		}
		m := MemberEntitlement{
			MemberID:            sm.MemberID,
			GroupIDs:            append([]uuid.UUID(nil), sm.ExitGroupIDs...),
			Suspended:           false,
			ValidFrom:           sm.ValidFrom.UTC(),
			EffectiveServiceEnd: sm.EffectiveServiceEnd.UTC(),
			PlanDownBps:         planDown,
			PlanUpBps:           planUp,
			FairUseTierDownBps:  0,
			FairUseTierUpBps:    0,
			TierIndex:           0,
			Priority:            1,
			FairUsePolicyGen:    0,
			HasFairUseGen:       false,
			Mode:                defaultFUPMode,
			Reason:              api.MemberRatePolicyReasonNormal,
		}
		memberIdx[sm.MemberID] = len(members)
		members = append(members, m)
		memberIDs = append(memberIDs, sm.MemberID)
	}

	const fupQuery = `
		SELECT
			fms.member_id,
			fms.mode,
			fms.enforced_tier_index,
			fms.enforced_down_bps,
			fms.enforced_up_bps,
			fms.enforced_priority,
			fms.enforced_reason,
			fms.generation::text,
			fms.override_valid_until,
			coalesce(pr.payload, '{}'::jsonb)
		FROM fup_member_states fms
		LEFT JOIN policy_revisions pr ON pr.id = fms.policy_revision_id
		WHERE fms.member_id = ANY($1)
		  AND fms.period_id = $2
		ORDER BY fms.member_id ASC`

	fupRows, err := r.q.Query(ctx, fupQuery, memberIDs, period.PeriodId)
	if err != nil {
		return nil, err
	}
	defer fupRows.Close()

	for fupRows.Next() {
		var (
			memberID            uuid.UUID
			modeStr             string
			enforcedTierIdx     int
			enforcedDownBps     *int64
			enforcedUpBps       *int64
			enforcedPrio        *int
			enforcedReason      *string
			genStr              string
			overrideValidUntil  *time.Time
			policyRevPayloadRaw []byte
		)
		if err := fupRows.Scan(
			&memberID,
			&modeStr,
			&enforcedTierIdx,
			&enforcedDownBps,
			&enforcedUpBps,
			&enforcedPrio,
			&enforcedReason,
			&genStr,
			&overrideValidUntil,
			&policyRevPayloadRaw,
		); err != nil {
			return nil, err
		}
		idx, ok := memberIdx[memberID]
		if !ok {
			continue
		}
		gen, _ := strconv.ParseUint(genStr, 10, 64)
		members[idx].FairUsePolicyGen = gen
		members[idx].HasFairUseGen = true

		isShadow := modeStr == string(api.MemberRatePolicyModeShadow)
		if !isShadow && len(policyRevPayloadRaw) > 0 {
			var revPol struct {
				Mode        string     `json:"mode"`
				EffectiveAt *time.Time `json:"effective_at"`
			}
			if json.Unmarshal(policyRevPayloadRaw, &revPol) == nil {
				if revPol.Mode == string(api.FairUsePayloadModeShadow) ||
					(revPol.EffectiveAt != nil && !revPol.EffectiveAt.IsZero() && asOf.Before(revPol.EffectiveAt.UTC())) {
					isShadow = true
				}
			}
		}

		if isShadow {
			// §11.2, §11.5: Shadow mode or before effective_at compiles mode=shadow with null tier rate.
			members[idx].Mode = api.MemberRatePolicyModeShadow
			members[idx].FairUseTierDownBps = 0
			members[idx].FairUseTierUpBps = 0
			members[idx].TierIndex = 0
			members[idx].Reason = api.MemberRatePolicyReasonNormal
		} else {
			members[idx].Mode = api.MemberRatePolicyModeEnforce
			members[idx].TierIndex = enforcedTierIdx
			if enforcedDownBps != nil && *enforcedDownBps > 0 {
				members[idx].FairUseTierDownBps = *enforcedDownBps
			} else {
				members[idx].FairUseTierDownBps = 0
			}
			if enforcedUpBps != nil && *enforcedUpBps > 0 {
				members[idx].FairUseTierUpBps = *enforcedUpBps
			} else {
				members[idx].FairUseTierUpBps = 0
			}
			if enforcedPrio != nil && *enforcedPrio >= 0 && *enforcedPrio <= 7 {
				members[idx].Priority = *enforcedPrio
			}
			if enforcedReason != nil && *enforcedReason != "" {
				members[idx].Reason = api.MemberRatePolicyReason(*enforcedReason)
			}
			if overrideValidUntil != nil {
				u := overrideValidUntil.UTC()
				members[idx].OverrideValidUntil = &u
			}
		}
	}
	if err := fupRows.Err(); err != nil {
		return nil, err
	}
	fupRows.Close()

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

// ReadPolicySnapshot reads active fair-use, DNS publication heads, and release support policies from PostgreSQL.
func (r *DBPolicyReader) ReadPolicySnapshot(ctx context.Context, nodeID uuid.UUID, asOf time.Time) (PolicySnapshot, error) {
	snap := PolicySnapshot{
		AuthorizationGeneration:  1,
		FairUseGeneration:        1,
		DNSGeneration:            0,
		DNSResolverIPv4:          []string{},
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
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PolicySnapshot{}, err
	}

	// Check node-specific desired_release_id override first.
	var nodeDesiredRel *uuid.UUID
	err = r.q.QueryRow(ctx, `SELECT desired_release_id FROM nodes WHERE id = $1`, nodeID).Scan(&nodeDesiredRel)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PolicySnapshot{}, err
	}
	if nodeDesiredRel != nil {
		desiredRelUUID = nodeDesiredRel
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
		  AND pd.type <> 'dns_profile'
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
		case string(api.PolicyTypeFairUse):
			if gen > snap.FairUseGeneration {
				snap.FairUseGeneration = gen
			}
		case string(api.PolicyTypeReleaseSupport):
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
	pRows.Close()

	// PolicySnapshot.DNSGeneration and dns_profile policy_refs come from the published head
	// in dns_publication_heads, not from the policy revision; with no published head it is 0 (§12.4, §12.5).
	hRows, err := r.q.Query(ctx, `
		SELECT profile_revision_id, policy_generation::text
		FROM dns_publication_heads
		WHERE status = 'active'
		ORDER BY definition_id ASC`)
	if err != nil {
		return PolicySnapshot{}, err
	}
	defer hRows.Close()
	for hRows.Next() {
		var (
			revID  uuid.UUID
			genStr string
		)
		if err := hRows.Scan(&revID, &genStr); err != nil {
			return PolicySnapshot{}, err
		}
		gen, _ := strconv.ParseUint(genStr, 10, 64)
		snap.PolicyRefs = append(snap.PolicyRefs, revID)
		if gen > snap.DNSGeneration {
			snap.DNSGeneration = gen
		}
	}
	if err := hRows.Err(); err != nil {
		return PolicySnapshot{}, err
	}
	hRows.Close()

	return snap, nil
}
