package desiredstate

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Outbox event types consumed by the desired-state recompilation worker (§5.2, §9, §11, §13).
const (
	EventMemberEntitlementChanged = "member.entitlement_changed"
	EventDeviceAuthorized         = "device.authorized"
	EventDeviceRevoked            = "device.revoked"
	EventMemberSuspended          = "member.suspended"
	EventPolicyActivated          = "policy.activated"
	EventNodePublicationChanged   = "node.publication_changed"
	EventReleaseTargetChanged     = "release.target_changed"
	EventNodeUpdated              = nodes.EventNodeUpdated
	EventNodeRevoked              = nodes.EventNodeRevoked
	EventNodeRetired              = nodes.EventNodeRetired
)

// Compiler compiles, signs, persists, and serves per-node desired-state envelopes (§5, §14.4, §14.7).
type Compiler struct {
	pool         *pgxpool.Pool
	signer       Signer
	entitlements EntitlementReader
	policies     PolicyReader
	now          func() time.Time
	envelopeTTL  time.Duration
}

// Option configures a Compiler.
type Option func(*Compiler)

// WithClock overrides the clock used by Compiler.
func WithClock(now func() time.Time) Option {
	return func(c *Compiler) {
		if now != nil {
			c.now = now
		}
	}
}

// WithEnvelopeTTL sets the validity window duration for signed desired-state envelopes.
func WithEnvelopeTTL(ttl time.Duration) Option {
	return func(c *Compiler) {
		if ttl > 0 {
			c.envelopeTTL = ttl
		}
	}
}

// WithEntitlementReader sets a custom EntitlementReader (such as FakeEntitlementReader in tests).
func WithEntitlementReader(r EntitlementReader) Option {
	return func(c *Compiler) {
		if r != nil {
			c.entitlements = r
		}
	}
}

// WithPolicyReader sets a custom PolicyReader (such as FakePolicyReader in tests).
func WithPolicyReader(r PolicyReader) Option {
	return func(c *Compiler) {
		if r != nil {
			c.policies = r
		}
	}
}

// NewCompiler constructs a desired-state Compiler.
func NewCompiler(pool *pgxpool.Pool, signer Signer, opts ...Option) *Compiler {
	c := &Compiler{
		pool:        pool,
		signer:      signer,
		now:         func() time.Time { return time.Now().UTC() },
		envelopeTTL: DefaultEnvelopeTTL,
	}
	if pool != nil {
		c.entitlements = NewDBEntitlementReader(pool)
		c.policies = NewDBPolicyReader(pool)
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// CompiledDesiredState holds a compiled and signed DesiredStateEnvelope together with its HTTP ETag.
type CompiledDesiredState struct {
	Envelope   api.DesiredStateEnvelope
	ETag       string
	Generation uint64
}

// Compile compiles a new monotonically increasing config_generation for nodeID,
// signs the DesiredState with the desired-state key purpose, and persists it (§5, §14.7).
func (c *Compiler) Compile(ctx context.Context, nodeID uuid.UUID) (CompiledDesiredState, error) {
	var out CompiledDesiredState
	err := db.InTx(ctx, c.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var err error
		out, err = c.CompileTx(ctx, tx, nodeID)
		return err
	})
	return out, err
}

// CompileTx compiles and persists a new desired-state generation inside an existing transaction.
func (c *Compiler) CompileTx(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID) (CompiledDesiredState, error) {
	if c.signer == nil {
		return CompiledDesiredState{}, errors.New("desiredstate: signer is required")
	}

	var (
		lifecycleState       string
		currentIncarnationID *uuid.UUID
		currentGenStr        string
		desiredReleaseUUID   *uuid.UUID
		capacityBytes        []byte
	)
	err := tx.QueryRow(ctx, `
		SELECT
			lifecycle_state,
			current_incarnation_id,
			desired_config_generation::text,
			desired_release_id,
			capacity
		FROM nodes
		WHERE id = $1
		FOR UPDATE`, nodeID).Scan(
		&lifecycleState,
		&currentIncarnationID,
		&currentGenStr,
		&desiredReleaseUUID,
		&capacityBytes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CompiledDesiredState{}, apierr.Newf(apierr.ResourceNotFound, "node %s not found", nodeID)
		}
		return CompiledDesiredState{}, err
	}
	if currentIncarnationID == nil || *currentIncarnationID == uuid.Nil {
		return CompiledDesiredState{}, apierr.Newf(apierr.StateConflict, "node %s has no active incarnation", nodeID)
	}
	incarnationID := *currentIncarnationID

	currentGen, _ := strconv.ParseUint(currentGenStr, 10, 64)
	var maxStoredGenStr string
	if err := tx.QueryRow(ctx, `
		SELECT coalesce(max(generation), 0)::text
		FROM desired_states
		WHERE node_id = $1 AND incarnation_id = $2`, nodeID, incarnationID).Scan(&maxStoredGenStr); err != nil {
		return CompiledDesiredState{}, err
	}
	maxStoredGen, _ := strconv.ParseUint(maxStoredGenStr, 10, 64)
	baseGen := currentGen
	if maxStoredGen > baseGen {
		baseGen = maxStoredGen
	}
	nextGen := baseGen + 1

	// Read exit publication_intent and exit_groups (§15.3).
	var (
		exitID       *uuid.UUID
		pubIntentStr string
	)
	err = tx.QueryRow(ctx, `
		SELECT id, publication_intent
		FROM exits
		WHERE node_id = $1`, nodeID).Scan(&exitID, &pubIntentStr)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return CompiledDesiredState{}, err
	}
	pubIntent := api.PublicationIntentUnpublished
	if pubIntentStr != "" {
		pubIntent = api.PublicationIntent(pubIntentStr)
	}
	if lifecycleState != string(api.LifecycleStateRegistered) {
		pubIntent = api.PublicationIntentDisabled
	}

	var groupIDs []uuid.UUID
	if exitID != nil {
		rows, err := tx.Query(ctx, `SELECT group_id FROM exit_groups WHERE exit_id = $1 ORDER BY group_id ASC`, *exitID)
		if err != nil {
			return CompiledDesiredState{}, err
		}
		for rows.Next() {
			var gid uuid.UUID
			if err := rows.Scan(&gid); err != nil {
				rows.Close()
				return CompiledDesiredState{}, err
			}
			groupIDs = append(groupIDs, gid)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return CompiledDesiredState{}, err
		}
		rows.Close()
	}

	var nodeCap *api.NodeCapacity
	if len(capacityBytes) > 0 {
		var parsed api.NodeCapacity
		if err := json.Unmarshal(capacityBytes, &parsed); err == nil {
			nodeCap = &parsed
		}
	}

	issuedAt := c.now().UTC()
	expiresAt := issuedAt.Add(c.envelopeTTL)

	// Read policy snapshot (use transaction-aware DB reader if default DBPolicyReader is in use).
	polReader := c.policies
	if _, isDB := polReader.(*DBPolicyReader); isDB || polReader == nil {
		polReader = NewDBPolicyReader(tx)
	}
	snap, err := polReader.ReadPolicySnapshot(ctx, nodeID, issuedAt)
	if err != nil {
		return CompiledDesiredState{}, err
	}
	if snap.AccountingPeriod.PeriodId == "" {
		snap.AccountingPeriod = DefaultAccountingPeriod(issuedAt)
	}
	if len(snap.DNSResolverIPv4) == 0 {
		snap.DNSResolverIPv4 = []string{DefaultDNSResolverIPv4}
	}
	if snap.DNSResolverIPv6 == nil {
		snap.DNSResolverIPv6 = []string{}
	}
	if snap.SupportedReleasePolicyID == uuid.Nil {
		snap.SupportedReleasePolicyID = DefaultSupportedReleasePolicyUUID
	}
	if snap.ServingReleaseIDs == nil {
		snap.ServingReleaseIDs = []uuid.UUID{}
	}
	if snap.AuthorizationGeneration == 0 {
		snap.AuthorizationGeneration = 1
	}
	if snap.DNSGeneration == 0 {
		snap.DNSGeneration = 1
	}
	if snap.FairUseGeneration == 0 {
		snap.FairUseGeneration = 1
	}

	// Read entitled members and device bindings for this exit's member groups.
	entReader := c.entitlements
	if _, isDB := entReader.(*DBEntitlementReader); isDB || entReader == nil {
		entReader = NewDBEntitlementReader(tx)
	}

	deviceBindings := []api.DeviceBinding{}
	memberRatePolicies := []api.MemberRatePolicy{}
	var maxBindingRev uint64

	// Only include member forwarding bindings when the exit's publication_intent is published
	// and the node is in lifecycle_state=registered (§5.1, §5.2 step 4).
	if pubIntent == api.PublicationIntentPublished && lifecycleState == string(api.LifecycleStateRegistered) && len(groupIDs) > 0 {
		entitled, err := entReader.ListEntitledMembersForGroups(ctx, groupIDs, issuedAt)
		if err != nil {
			return CompiledDesiredState{}, err
		}
		sort.Slice(entitled, func(i, j int) bool {
			return entitled[i].MemberID.String() < entitled[j].MemberID.String()
		})

		for _, m := range entitled {
			validFrom := m.ValidFrom.UTC()
			if validFrom.IsZero() {
				validFrom = snap.AccountingPeriod.PeriodStart
			}
			validUntil := m.EffectiveServiceEnd.UTC()
			if m.Suspended {
				// Member suspension clamps forwarding_valid_until (ValidUntil) so the node immediately halts forwarding (§9.2, §11.2).
				clamped := issuedAt
				if m.ClampedUntil != nil && m.ClampedUntil.Before(clamped) {
					clamped = m.ClampedUntil.UTC()
				}
				if validUntil.After(clamped) {
					validUntil = clamped
				}
			}

			planDown := m.PlanDownBps
			if planDown <= 0 {
				planDown = DefaultPlanRateBps
			}
			planUp := m.PlanUpBps
			if planUp <= 0 {
				planUp = DefaultPlanRateBps
			}
			tierDown := m.FairUseTierDownBps
			if tierDown <= 0 {
				tierDown = planDown
			}
			tierUp := m.FairUseTierUpBps
			if tierUp <= 0 {
				tierUp = planUp
			}

			// §11.2 step 3: effective_rate = min(plan_rate, fair_use_tier_rate, node_member_rate_cap)
			effDown := minI64(planDown, tierDown)
			effUp := minI64(planUp, tierUp)
			if nodeCap != nil && nodeCap.MemberDownBpsCap != nil && *nodeCap.MemberDownBpsCap > 0 {
				effDown = minI64(effDown, *nodeCap.MemberDownBpsCap)
			}
			if nodeCap != nil && nodeCap.MemberUpBpsCap != nil && *nodeCap.MemberUpBpsCap > 0 {
				effUp = minI64(effUp, *nodeCap.MemberUpBpsCap)
			}

			prio := m.Priority
			if prio < 0 {
				prio = 0
			} else if prio > 7 {
				prio = 7
			}
			fupGen := m.FairUsePolicyGen
			if fupGen == 0 {
				fupGen = snap.FairUseGeneration
			}
			mode := m.Mode
			if mode == "" {
				mode = api.MemberRatePolicyModeEnforce
			}
			reason := m.Reason
			if reason == "" {
				reason = api.MemberRatePolicyReasonNormal
			}

			memberRatePolicies = append(memberRatePolicies, api.MemberRatePolicy{
				MemberId:   m.MemberID,
				Generation: nodes.ParseU64(fupGen),
				PeriodId:   snap.AccountingPeriod.PeriodId,
				Mode:       mode,
				Reason:     reason,
				TierIndex:  m.TierIndex,
				DownBps:    &effDown,
				UpBps:      &effUp,
				Priority:   prio,
				ValidFrom:  validFrom,
				ValidUntil: validUntil,
			})

			if m.Suspended {
				continue
			}

			bindings := append([]MemberDeviceBinding(nil), m.Bindings...)
			sort.Slice(bindings, func(i, j int) bool {
				return bindings[i].DeviceID.String() < bindings[j].DeviceID.String()
			})
			for _, b := range bindings {
				// Device revocation removes peer binding immediately (§10.2).
				if b.Revoked {
					continue
				}
				if !b.ValidUntil.After(issuedAt) {
					continue
				}
				if b.BindingRevision > maxBindingRev {
					maxBindingRev = b.BindingRevision
				}
				v4 := append([]string(nil), b.AllowedIPv4...)
				if v4 == nil {
					v4 = []string{}
				}
				v6 := append([]string(nil), b.AllowedIPv6...)
				if v6 == nil {
					v6 = []string{}
				}
				deviceBindings = append(deviceBindings, api.DeviceBinding{
					MemberId:        m.MemberID,
					DeviceId:        b.DeviceID,
					PeerId:          b.NetbirdPeerID,
					AllowedIpv4:     v4,
					AllowedIpv6:     v6,
					BindingRevision: nodes.ParseU64(b.BindingRevision),
					ValidFrom:       b.ValidFrom.UTC(),
					ValidUntil:      b.ValidUntil.UTC(),
				})
			}
		}
	}

	desired := api.DesiredState{
		SchemaVersion:            api.DesiredStateSchemaVersion(1),
		NodeId:                   nodeID,
		NodeIncarnationId:        incarnationID,
		ConfigGeneration:         nodes.ParseU64(nextGen),
		BindingRevision:          nodes.ParseU64(maxBindingRev),
		IssuedAt:                 issuedAt,
		ExpiresAt:                expiresAt,
		PublicationIntent:        pubIntent,
		DesiredReleaseId:         snap.DesiredReleaseID,
		SupportedReleasePolicyId: snap.SupportedReleasePolicyID,
		ServingReleaseIds:        snap.ServingReleaseIDs,
		PolicyRefs: api.PolicyRefs{
			AuthorizationGeneration: nodes.ParseU64(snap.AuthorizationGeneration),
			DnsGeneration:           nodes.ParseU64(snap.DNSGeneration),
			FupGeneration:           nodes.ParseU64(snap.FairUseGeneration),
		},
		SafetyProfile:        DefaultSafetyProfile,
		LeaseRenewalEndpoint: api.DesiredStateLeaseRenewalEndpoint(DefaultLeaseRenewalEndpoint),
		Dns: api.DnsServiceConfig{
			ResolverIpv4: snap.DNSResolverIPv4,
			ResolverIpv6: snap.DNSResolverIPv6,
		},
		NodeCapacity:       nodeCap,
		AccountingPeriod:   snap.AccountingPeriod,
		DeviceBindings:     deviceBindings,
		MemberRatePolicies: memberRatePolicies,
		Update:             snap.UpdateIntent,
	}

	dsEnv, etag, configHash, err := c.signDesiredState(ctx, desired)
	if err != nil {
		return CompiledDesiredState{}, err
	}

	policyRefs := snap.PolicyRefs
	if policyRefs == nil {
		policyRefs = []uuid.UUID{}
	}
	releaseUUID := desiredReleaseUUID
	if releaseUUID == nil && snap.DesiredReleaseID != nil {
		var rid uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM releases WHERE id = $1`, *snap.DesiredReleaseID).Scan(&rid); err == nil {
			releaseUUID = &rid
		}
	}

	var desiredStateRowID uuid.UUID
	const insertDesired = `
		INSERT INTO desired_states (
			node_id, incarnation_id, generation, policy_refs,
			binding_revision, release_id, config_hash, schema_version, signed_at
		) VALUES (
			$1, $2, $3::numeric, $4,
			$5::numeric, $6, $7, 1, $8
		) RETURNING id`
	if err := tx.QueryRow(ctx, insertDesired,
		nodeID, incarnationID, strconv.FormatUint(nextGen, 10), policyRefs,
		strconv.FormatUint(maxBindingRev, 10), releaseUUID, configHash, issuedAt,
	).Scan(&desiredStateRowID); err != nil {
		return CompiledDesiredState{}, err
	}

	envJSON, err := json.Marshal(dsEnv)
	if err != nil {
		return CompiledDesiredState{}, err
	}

	const upsertEnvelope = `
		INSERT INTO desired_state_envelopes (
			desired_state_id, node_id, incarnation_id, generation,
			etag, envelope, issued_at, expires_at
		) VALUES (
			$1, $2, $3, $4::numeric,
			$5, $6::jsonb, $7, $8
		)
		ON CONFLICT (node_id, incarnation_id) DO UPDATE SET
			desired_state_id = EXCLUDED.desired_state_id,
			generation = EXCLUDED.generation,
			etag = EXCLUDED.etag,
			envelope = EXCLUDED.envelope,
			issued_at = EXCLUDED.issued_at,
			expires_at = EXCLUDED.expires_at`
	if _, err := tx.Exec(ctx, upsertEnvelope,
		desiredStateRowID, nodeID, incarnationID, strconv.FormatUint(nextGen, 10),
		etag, envJSON, issuedAt, expiresAt,
	); err != nil {
		return CompiledDesiredState{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE nodes
		SET desired_config_generation = $2::numeric,
		    resource_version = resource_version + 1
		WHERE id = $1`, nodeID, strconv.FormatUint(nextGen, 10)); err != nil {
		return CompiledDesiredState{}, err
	}

	return CompiledDesiredState{
		Envelope:   dsEnv,
		ETag:       etag,
		Generation: nextGen,
	}, nil
}

// ReSign re-signs the current desired-state document for nodeID without incrementing
// config_generation, updating the envelope validity window and HTTP ETag (§14.7).
func (c *Compiler) ReSign(ctx context.Context, nodeID uuid.UUID) (CompiledDesiredState, error) {
	if c.signer == nil {
		return CompiledDesiredState{}, errors.New("desiredstate: signer is required")
	}

	var out CompiledDesiredState
	err := db.InTx(ctx, c.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var (
			incarnationID *uuid.UUID
			envBytes      []byte
			prevIssuedAt  time.Time
		)
		err := tx.QueryRow(ctx, `
			SELECT n.current_incarnation_id, dse.envelope, dse.issued_at
			FROM nodes n
			JOIN desired_state_envelopes dse
			  ON dse.node_id = n.id AND dse.incarnation_id = n.current_incarnation_id
			WHERE n.id = $1
			FOR UPDATE OF n, dse`, nodeID).Scan(&incarnationID, &envBytes, &prevIssuedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				out, err = c.CompileTx(ctx, tx, nodeID)
				return err
			}
			return err
		}

		var prevEnv api.DesiredStateEnvelope
		if err := json.Unmarshal(envBytes, &prevEnv); err != nil {
			return err
		}

		issuedAt := c.now().UTC()
		if !issuedAt.After(prevIssuedAt.UTC()) {
			issuedAt = prevIssuedAt.UTC().Add(time.Second)
		}
		expiresAt := issuedAt.Add(c.envelopeTTL)

		desired := prevEnv.Signed
		desired.IssuedAt = issuedAt
		desired.ExpiresAt = expiresAt

		newEnv, newETag, _, err := c.signDesiredState(ctx, desired)
		if err != nil {
			return err
		}
		newEnvJSON, err := json.Marshal(newEnv)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			UPDATE desired_state_envelopes
			SET etag = $1,
			    envelope = $2::jsonb,
			    issued_at = $3,
			    expires_at = $4
			WHERE node_id = $5 AND incarnation_id = $6`,
			newETag, newEnvJSON, issuedAt, expiresAt, nodeID, *incarnationID,
		); err != nil {
			return err
		}

		out = CompiledDesiredState{
			Envelope:   newEnv,
			ETag:       newETag,
			Generation: uint64(desired.ConfigGeneration),
		}
		return nil
	})
	return out, err
}

func (c *Compiler) signDesiredState(ctx context.Context, desired api.DesiredState) (api.DesiredStateEnvelope, string, []byte, error) {
	desiredCanon, err := canonjson.Marshal(desired)
	if err != nil {
		return api.DesiredStateEnvelope{}, "", nil, err
	}
	configHashArr := sha256.Sum256(desiredCanon)
	rawEnv, err := c.signer.Sign(ctx, envelope.PurposeDesiredState, desired)
	if err != nil {
		return api.DesiredStateEnvelope{}, "", nil, err
	}
	sigs := make([]api.Signature, 0, len(rawEnv.Signatures))
	for _, s := range rawEnv.Signatures {
		sigs = append(sigs, api.Signature{
			Algorithm:    api.SignatureAlgorithm(s.Algorithm),
			KeyId:        s.KeyID,
			SignatureB64: s.SignatureB64,
		})
	}
	dsEnv := api.DesiredStateEnvelope{
		Signed:     desired,
		Signatures: sigs,
	}
	envCanon, err := canonjson.Marshal(dsEnv)
	if err != nil {
		return api.DesiredStateEnvelope{}, "", nil, err
	}
	envHashArr := sha256.Sum256(envCanon)
	etag := fmt.Sprintf(`"ds-%s-%x"`, desired.ConfigGeneration.String(), envHashArr[:12])
	return dsEnv, etag, configHashArr[:], nil
}

// GetDesiredStateResult holds the response for GET /agent/desired-state (§14.4, §14.7).
type GetDesiredStateResult struct {
	NotModified bool
	ETag        string
	Envelope    *api.DesiredStateEnvelope
}

// GetDesiredState resolves the node and incarnation strictly from the authenticated
// mTLS identity (`authNodeID`, `authIncarnationID`), checks `If-None-Match` for 304,
// and returns the signed DesiredStateEnvelope (§14.4, §14.7).
func (c *Compiler) GetDesiredState(
	ctx context.Context,
	authNodeID uuid.UUID,
	authIncarnationID uuid.UUID,
	ifNoneMatch string,
) (GetDesiredStateResult, error) {
	if authNodeID == uuid.Nil || authIncarnationID == uuid.Nil {
		return GetDesiredStateResult{}, apierr.New(apierr.Unauthenticated, "authenticated node and incarnation identity required")
	}

	var (
		lifecycleState       string
		currentIncarnationID *uuid.UUID
	)
	err := c.pool.QueryRow(ctx, `
		SELECT lifecycle_state, current_incarnation_id
		FROM nodes
		WHERE id = $1`, authNodeID).Scan(&lifecycleState, &currentIncarnationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GetDesiredStateResult{}, apierr.Newf(apierr.ResourceNotFound, "node %s not found", authNodeID)
		}
		return GetDesiredStateResult{}, err
	}
	if lifecycleState == string(api.LifecycleStateRevoked) || lifecycleState == string(api.LifecycleStateRetired) {
		return GetDesiredStateResult{}, apierr.Newf(apierr.IdentityRevoked, "node %s is %s", authNodeID, lifecycleState)
	}

	var (
		incNodeID uuid.UUID
		revokedAt *time.Time
	)
	err = c.pool.QueryRow(ctx, `
		SELECT node_id, revoked_at
		FROM node_incarnations
		WHERE id = $1`, authIncarnationID).Scan(&incNodeID, &revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GetDesiredStateResult{}, apierr.New(apierr.IncarnationMismatch, "incarnation is not registered")
		}
		return GetDesiredStateResult{}, err
	}
	if incNodeID != authNodeID {
		return GetDesiredStateResult{}, apierr.New(apierr.IncarnationMismatch, "incarnation does not belong to authenticated node")
	}
	if revokedAt != nil {
		return GetDesiredStateResult{}, apierr.New(apierr.IdentityRevoked, "node incarnation has been revoked")
	}
	if currentIncarnationID == nil || *currentIncarnationID != authIncarnationID {
		return GetDesiredStateResult{}, apierr.New(apierr.IncarnationMismatch, "node incarnation is not active")
	}

	var (
		etag     string
		envBytes []byte
	)
	err = c.pool.QueryRow(ctx, `
		SELECT etag, envelope
		FROM desired_state_envelopes
		WHERE node_id = $1 AND incarnation_id = $2`,
		authNodeID, authIncarnationID).Scan(&etag, &envBytes)
	if errors.Is(err, pgx.ErrNoRows) {
		compiled, err := c.Compile(ctx, authNodeID)
		if err != nil {
			return GetDesiredStateResult{}, err
		}
		if etagMatches(ifNoneMatch, compiled.ETag) {
			return GetDesiredStateResult{
				NotModified: true,
				ETag:        compiled.ETag,
			}, nil
		}
		envCopy := compiled.Envelope
		return GetDesiredStateResult{
			NotModified: false,
			ETag:        compiled.ETag,
			Envelope:    &envCopy,
		}, nil
	} else if err != nil {
		return GetDesiredStateResult{}, err
	}

	if etagMatches(ifNoneMatch, etag) {
		return GetDesiredStateResult{
			NotModified: true,
			ETag:        etag,
		}, nil
	}

	var env api.DesiredStateEnvelope
	if err := json.Unmarshal(envBytes, &env); err != nil {
		return GetDesiredStateResult{}, err
	}
	return GetDesiredStateResult{
		NotModified: false,
		ETag:        etag,
		Envelope:    &env,
	}, nil
}

func etagMatches(ifNoneMatch, currentETag string) bool {
	if strings.TrimSpace(ifNoneMatch) == "" || currentETag == "" {
		return false
	}
	for _, part := range strings.Split(ifNoneMatch, ",") {
		cand := strings.TrimSpace(part)
		if cand == "*" || cand == currentETag {
			return true
		}
	}
	return false
}

// RecompileBatch coalesces a burst of outbox events into a deduplicated set of affected
// nodes and recompiles each affected node once without regressing config_generation (§5, §14.7).
func (c *Compiler) RecompileBatch(ctx context.Context, events []db.ClaimedEvent) ([]CompiledDesiredState, error) {
	if len(events) == 0 {
		return nil, nil
	}

	affectedNodes := map[uuid.UUID]bool{}
	recompileAllActive := false

	for _, ev := range events {
		switch ev.EventType {
		case EventNodePublicationChanged, EventNodeUpdated, EventNodeRevoked, EventNodeRetired:
			if ev.AggregateID != uuid.Nil {
				affectedNodes[ev.AggregateID] = true
			}
			var payload struct {
				NodeID string `json:"node_id"`
			}
			if json.Unmarshal(ev.Payload, &payload) == nil && payload.NodeID != "" {
				if nid, err := uuid.Parse(payload.NodeID); err == nil {
					affectedNodes[nid] = true
				}
			}
		case EventPolicyActivated, EventReleaseTargetChanged:
			var payload struct {
				NodeID  string   `json:"node_id"`
				NodeIDs []string `json:"node_ids"`
			}
			_ = json.Unmarshal(ev.Payload, &payload)
			if payload.NodeID != "" {
				if nid, err := uuid.Parse(payload.NodeID); err == nil {
					affectedNodes[nid] = true
				}
			}
			for _, raw := range payload.NodeIDs {
				if nid, err := uuid.Parse(raw); err == nil {
					affectedNodes[nid] = true
				}
			}
			if payload.NodeID == "" && len(payload.NodeIDs) == 0 {
				recompileAllActive = true
			}
		case EventMemberEntitlementChanged, EventDeviceAuthorized, EventDeviceRevoked, EventMemberSuspended:
			var payload struct {
				MemberID string   `json:"member_id"`
				GroupIDs []string `json:"group_ids"`
				NodeIDs  []string `json:"node_ids"`
			}
			_ = json.Unmarshal(ev.Payload, &payload)
			for _, raw := range payload.NodeIDs {
				if nid, err := uuid.Parse(raw); err == nil {
					affectedNodes[nid] = true
				}
			}
			if len(payload.GroupIDs) > 0 {
				var gids []uuid.UUID
				for _, raw := range payload.GroupIDs {
					if gid, err := uuid.Parse(raw); err == nil {
						gids = append(gids, gid)
					}
				}
				if len(gids) > 0 {
					nids, err := c.findNodesForGroups(ctx, gids)
					if err != nil {
						return nil, err
					}
					for _, nid := range nids {
						affectedNodes[nid] = true
					}
				}
			} else if len(payload.NodeIDs) == 0 {
				recompileAllActive = true
			}
		}
	}

	if recompileAllActive {
		rows, err := c.pool.Query(ctx, `
			SELECT id
			FROM nodes
			WHERE current_incarnation_id IS NOT NULL
			  AND lifecycle_state IN ('registered', 'revoked')
			ORDER BY id ASC`)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var nid uuid.UUID
			if err := rows.Scan(&nid); err != nil {
				rows.Close()
				return nil, err
			}
			affectedNodes[nid] = true
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	orderedNodeIDs := make([]uuid.UUID, 0, len(affectedNodes))
	for nid := range affectedNodes {
		orderedNodeIDs = append(orderedNodeIDs, nid)
	}
	sort.Slice(orderedNodeIDs, func(i, j int) bool {
		return orderedNodeIDs[i].String() < orderedNodeIDs[j].String()
	})

	var compiled []CompiledDesiredState
	for _, nid := range orderedNodeIDs {
		res, err := c.Compile(ctx, nid)
		if err != nil {
			var apiErr *apierr.Error
			if errors.As(err, &apiErr) && (apiErr.HTTPStatus() == 404 || apiErr.HTTPStatus() == 409) {
				continue
			}
			return compiled, err
		}
		compiled = append(compiled, res)
	}
	return compiled, nil
}

func (c *Compiler) findNodesForGroups(ctx context.Context, groupIDs []uuid.UUID) ([]uuid.UUID, error) {
	rows, err := c.pool.Query(ctx, `
		SELECT DISTINCT e.node_id
		FROM exits e
		JOIN exit_groups eg ON eg.exit_id = e.id
		JOIN nodes n ON n.id = e.node_id
		WHERE eg.group_id = ANY($1)
		  AND n.current_incarnation_id IS NOT NULL`, groupIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nids []uuid.UUID
	for rows.Next() {
		var nid uuid.UUID
		if err := rows.Scan(&nid); err != nil {
			return nil, err
		}
		nids = append(nids, nid)
	}
	return nids, rows.Err()
}

func minI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
