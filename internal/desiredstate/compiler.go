package desiredstate

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Outbox event types consumed by the desired-state recompilation worker (§5.2, §9, §11, §13).
const (
	EventMemberEntitlementChanged   = "member.entitlement_changed"
	EventDeviceAuthorized           = "device.authorized"
	EventDeviceRevoked              = "device.revoked"
	EventMemberSuspended            = "member.suspended"
	EventPolicyActivated            = "policy.activated"
	EventFUPMemberGenerationChanged = "fup.member_generation_changed"
	EventNodePublicationChanged     = "node.publication_changed"
	EventReleaseTargetChanged       = "release.target_changed"
	EventNodeUpdated                = nodes.EventNodeUpdated
	EventNodeRevoked                = nodes.EventNodeRevoked
	EventNodeRetired                = nodes.EventNodeRetired
)

// Compiler compiles, signs, persists, and serves per-node desired-state envelopes (§5, §14.4, §14.7).
type Compiler struct {
	documentInputs  func(context.Context, pgx.Tx, *api.DesiredState, []uuid.UUID) error
	documentIssued  func(context.Context, pgx.Tx, api.DesiredState) error
	pool            *pgxpool.Pool
	signer          Signer
	entitlements    EntitlementReader
	policies        PolicyReader
	now             func() time.Time
	envelopeTTL     time.Duration
	dnsResolverIPv4 []string
	dnsResolverIPv6 []string
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

// WithDNSResolvers configures the deployment DNS resolver addresses stamped into compiled
// desired-state documents (§8.1, §12.2). Resolver addresses are deployment configuration
// supplied at construction (or later a published resolver document from w3-dns-publish),
// never read from dns_profile payloads and with no production default.
func WithDNSResolvers(ipv4, ipv6 []string) Option {
	return func(c *Compiler) {
		c.dnsResolverIPv4 = append([]string(nil), ipv4...)
		c.dnsResolverIPv6 = append([]string(nil), ipv6...)
	}
}

// WithDocumentInputs supplies transaction-bound fields before hashing/signing and
// records issued references after persistence in the same transaction (D-056/D-058).
func WithDocumentInputs(resolve func(context.Context, pgx.Tx, *api.DesiredState, []uuid.UUID) error, issued func(context.Context, pgx.Tx, api.DesiredState) error) Option {
	return func(c *Compiler) { c.documentInputs, c.documentIssued = resolve, issued }
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

// OmittedMember records a member omitted from a compiled desired state due to a data error (such as missing plan rate).
type OmittedMember struct {
	MemberID uuid.UUID
	Reason   string
}

// CompiledDesiredState holds a compiled and signed DesiredStateEnvelope together with its HTTP ETag.
type CompiledDesiredState struct {
	Envelope       api.DesiredStateEnvelope
	ETag           string
	Generation     uint64
	OmittedMembers []OmittedMember
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
	if len(c.dnsResolverIPv4) > 0 || len(c.dnsResolverIPv6) > 0 {
		snap.DNSResolverIPv4 = append([]string(nil), c.dnsResolverIPv4...)
		snap.DNSResolverIPv6 = append([]string(nil), c.dnsResolverIPv6...)
	}
	if snap.DNSResolverIPv4 == nil {
		snap.DNSResolverIPv4 = []string{}
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
	var omittedMembers []OmittedMember
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
			// A plan without a rate is a data error: omit the member from the compiled state
			// and record the reason in the compile result and log (§11.2).
			if m.PlanDownBps <= 0 || m.PlanUpBps <= 0 {
				omittedMembers = append(omittedMembers, OmittedMember{
					MemberID: m.MemberID,
					Reason:   "plan_missing_rate",
				})
				slog.WarnContext(ctx, "desiredstate: omitting member with missing plan rate",
					"node_id", nodeID,
					"member_id", m.MemberID,
					"reason", "plan_missing_rate",
				)
				continue
			}

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
			rateValidUntil := validUntil
			if m.OverrideValidUntil != nil && m.OverrideValidUntil.UTC().Before(rateValidUntil) {
				rateValidUntil = m.OverrideValidUntil.UTC()
			}

			mode := m.Mode
			if mode == "" {
				mode = api.MemberRatePolicyModeEnforce
			}
			reason := m.Reason
			if reason == "" {
				reason = api.MemberRatePolicyReasonNormal
			}
			prio := m.Priority
			if prio < 0 {
				prio = 0
			} else if prio > 7 {
				prio = 7
			}
			fupGen := m.FairUsePolicyGen
			if !m.HasFairUseGen && fupGen == 0 {
				fupGen = snap.FairUseGeneration
			}

			eval := &fup.EvaluationResult{
				MemberID:   m.MemberID,
				PeriodID:   snap.AccountingPeriod.PeriodId,
				Mode:       api.FairUsePayloadModeEnforce,
				Generation: nodes.ParseU64(fupGen),
				ValidFrom:  validFrom,
				ValidUntil: rateValidUntil,
			}
			if mode == api.MemberRatePolicyModeShadow {
				// Shadow mode has null tier rate; only plan_rate and node_member_rate_cap apply (§11.2, §11.5).
				eval.Mode = api.FairUsePayloadModeShadow
			} else {
				eval.EnforcedTierIndex = m.TierIndex
				if m.FairUseTierDownBps > 0 {
					d := m.FairUseTierDownBps
					eval.EnforcedDownBps = &d
				}
				if m.FairUseTierUpBps > 0 {
					u := m.FairUseTierUpBps
					eval.EnforcedUpBps = &u
				}
				eval.EnforcedPriority = &prio
				eval.EnforcedReason = &reason
			}

			planRate := &api.RatePair{
				DownBps: m.PlanDownBps,
				UpBps:   m.PlanUpBps,
			}
			memberRatePolicies = append(memberRatePolicies, fup.BuildNodeMemberRatePolicy(eval, planRate, nodeCapacityRatePair(nodeCap)))

			bindings := append([]MemberDeviceBinding(nil), m.Bindings...)
			sort.Slice(bindings, func(i, j int) bool {
				return bindings[i].DeviceID.String() < bindings[j].DeviceID.String()
			})
			for _, b := range bindings {
				// Device revocation removes peer binding immediately (§10.2).
				if b.Revoked {
					continue
				}
				bValidFrom := b.ValidFrom.UTC()
				bValidUntil := b.ValidUntil.UTC()
				if m.Suspended && bValidUntil.After(validUntil) {
					bValidUntil = validUntil
				}
				if !bValidUntil.After(bValidFrom) {
					continue
				}
				if !m.Suspended && !bValidUntil.After(issuedAt) {
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
					ValidFrom:       bValidFrom,
					ValidUntil:      bValidUntil,
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
		DesiredReleaseId:         presence.FromPointer(snap.DesiredReleaseID),
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
		NodeCapacity:       presence.FromPointer(nodeCap),
		AccountingPeriod:   snap.AccountingPeriod,
		DeviceBindings:     deviceBindings,
		MemberRatePolicies: memberRatePolicies,
		Update:             presence.FromPointer(snap.UpdateIntent),
	}

	if c.documentInputs != nil {
		if err := c.documentInputs(ctx, tx, &desired, snap.PolicyRefs); err != nil {
			return CompiledDesiredState{}, err
		}
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
		err := tx.QueryRow(ctx, `SELECT id FROM releases WHERE id = $1`, *snap.DesiredReleaseID).Scan(&rid)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return CompiledDesiredState{}, err
		}
		if err == nil {
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

	afterAudit, _ := json.Marshal(map[string]any{
		"desired_state_id":          desiredStateRowID.String(),
		"desired_config_generation": strconv.FormatUint(nextGen, 10),
		"binding_revision":          strconv.FormatUint(maxBindingRev, 10),
		"dns_generation":            strconv.FormatUint(snap.DNSGeneration, 10),
	})
	if err := db.AppendAudit(ctx, tx, db.AuditEvent{
		Actor:         nodes.SystemActorUUID.String(),
		ActorType:     "system",
		Action:        "node.desired_state_compiled",
		Resource:      "node:" + nodeID.String(),
		Result:        "succeeded",
		SchemaVersion: 1,
		Before:        []byte(`{}`),
		After:         afterAudit,
	}); err != nil {
		return CompiledDesiredState{}, err
	}

	if err := nodes.QueuePublicationGateTx(ctx, tx, nodeID, issuedAt, true); err != nil {
		return CompiledDesiredState{}, err
	}

	if c.documentIssued != nil {
		if err := c.documentIssued(ctx, tx, desired); err != nil {
			return CompiledDesiredState{}, err
		}
	}

	return CompiledDesiredState{
		Envelope:       dsEnv,
		ETag:           etag,
		Generation:     nextGen,
		OmittedMembers: omittedMembers,
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
			lifecycleState string
			incarnationID  *uuid.UUID
			incRevokedAt   *time.Time
		)
		err := tx.QueryRow(ctx, `
			SELECT n.lifecycle_state, n.current_incarnation_id, inc.revoked_at
			FROM nodes n
			LEFT JOIN node_incarnations inc ON inc.id = n.current_incarnation_id
			WHERE n.id = $1
			FOR UPDATE OF n`, nodeID).Scan(&lifecycleState, &incarnationID, &incRevokedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierr.Newf(apierr.ResourceNotFound, "node %s not found", nodeID)
			}
			return err
		}
		if lifecycleState == string(api.LifecycleStateRevoked) || lifecycleState == string(api.LifecycleStateRetired) {
			return apierr.Newf(apierr.IdentityRevoked, "node %s is %s", nodeID, lifecycleState)
		}
		if incarnationID == nil || *incarnationID == uuid.Nil {
			return apierr.Newf(apierr.StateConflict, "node %s has no active incarnation", nodeID)
		}
		if incRevokedAt != nil {
			return apierr.New(apierr.IdentityRevoked, "node incarnation has been revoked")
		}

		var (
			envBytes     []byte
			prevIssuedAt time.Time
		)
		err = tx.QueryRow(ctx, `
			SELECT envelope, issued_at
			FROM desired_state_envelopes
			WHERE node_id = $1 AND incarnation_id = $2
			FOR UPDATE`, nodeID, *incarnationID).Scan(&envBytes, &prevIssuedAt)
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
		if err := nodes.CheckPublicationDelivery(ctx, c.pool, compiled.Envelope.Signed); err != nil {
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

	var env api.DesiredStateEnvelope
	if err := json.Unmarshal(envBytes, &env); err != nil {
		return GetDesiredStateResult{}, err
	}
	if err := nodes.CheckPublicationDelivery(ctx, c.pool, env.Signed); err != nil {
		return GetDesiredStateResult{}, err
	}
	if etagMatches(ifNoneMatch, etag) {
		return GetDesiredStateResult{NotModified: true, ETag: etag}, nil
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
// A fair_use activation with no node ids, and fup.member_generation_changed, select only the
// published exits carrying the affected members. A second delivery does not compile again when
// the stored envelope already matches that Fair-Use state.
func (c *Compiler) RecompileBatch(ctx context.Context, events []db.ClaimedEvent) ([]CompiledDesiredState, error) {
	if len(events) == 0 {
		return nil, nil
	}

	affectedNodes := map[uuid.UUID]bool{}
	fupOnly := map[uuid.UUID]bool{}
	fupScopes := map[uuid.UUID]*fupMemberScope{}
	recompileAllActive := false

	for _, ev := range events {
		switch ev.EventType {
		case EventNodePublicationChanged, EventNodeUpdated, EventNodeRevoked, EventNodeRetired:
			if ev.AggregateID != uuid.Nil {
				noteAffected(affectedNodes, fupOnly, ev.AggregateID, false)
			}
			var payload struct {
				NodeID string `json:"node_id"`
			}
			if json.Unmarshal(ev.Payload, &payload) == nil && payload.NodeID != "" {
				if nid, err := uuid.Parse(payload.NodeID); err == nil {
					noteAffected(affectedNodes, fupOnly, nid, false)
				}
			}
		case EventPolicyActivated, EventReleaseTargetChanged:
			var payload struct {
				NodeID  string   `json:"node_id"`
				NodeIDs []string `json:"node_ids"`
				Type    string   `json:"type"`
			}
			_ = json.Unmarshal(ev.Payload, &payload)
			if payload.NodeID != "" {
				if nid, err := uuid.Parse(payload.NodeID); err == nil {
					noteAffected(affectedNodes, fupOnly, nid, false)
				}
			}
			for _, raw := range payload.NodeIDs {
				if nid, err := uuid.Parse(raw); err == nil {
					noteAffected(affectedNodes, fupOnly, nid, false)
				}
			}
			// A named node list, even if it does not parse, is not a fleet-wide activation.
			if payload.NodeID != "" || len(payload.NodeIDs) > 0 {
				break
			}
			// fair_use activations name no node. Recompile the exits that serve members,
			// not every registered node. Other policy types keep the fleet-wide path.
			if ev.EventType == EventPolicyActivated && payload.Type == string(api.PolicyTypeFairUse) {
				ids, err := c.exitsCarryingMembers(ctx, nil)
				if err != nil {
					return nil, err
				}
				for _, id := range ids {
					noteAffected(affectedNodes, fupOnly, id, true)
					if fupOnly[id] {
						noteFUPScope(fupScopes, id, nil)
					}
				}
			} else {
				recompileAllActive = true
			}
		case EventFUPMemberGenerationChanged:
			var payload struct {
				MemberID string `json:"member_id"`
			}
			if err := json.Unmarshal(ev.Payload, &payload); err != nil {
				return nil, fmt.Errorf("desiredstate: decode fup.member_generation_changed: %w", err)
			}
			memberID, err := uuid.Parse(payload.MemberID)
			if err != nil {
				return nil, fmt.Errorf("desiredstate: fup.member_generation_changed member_id: %w", err)
			}
			ids, err := c.exitsCarryingMembers(ctx, []uuid.UUID{memberID})
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				noteAffected(affectedNodes, fupOnly, id, true)
				if fupOnly[id] {
					noteFUPScope(fupScopes, id, &memberID)
				}
			}
		case EventMemberEntitlementChanged, EventDeviceAuthorized, EventDeviceRevoked, EventMemberSuspended:
			var payload struct {
				MemberID             string   `json:"member_id"`
				GroupIDs             []string `json:"group_ids"`
				ExitGroupIDs         []string `json:"exit_group_ids"`
				PreviousExitGroupIDs []string `json:"previous_exit_group_ids"`
				NodeIDs              []string `json:"node_ids"`
			}
			_ = json.Unmarshal(ev.Payload, &payload)
			for _, raw := range payload.NodeIDs {
				if nid, err := uuid.Parse(raw); err == nil {
					noteAffected(affectedNodes, fupOnly, nid, false)
				}
			}
			rawGroups := append([]string(nil), payload.GroupIDs...)
			rawGroups = append(rawGroups, payload.ExitGroupIDs...)
			rawGroups = append(rawGroups, payload.PreviousExitGroupIDs...)
			// D-052 F-02: old producers omitted removed groups. Also recompile
			// each latest envelope that still grants this member access; the new
			// entitlement snapshot will remove its bindings and rate policies.
			if member, err := uuid.Parse(payload.MemberID); err == nil {
				rows, err := c.pool.Query(ctx, `SELECT e.node_id FROM desired_state_envelopes e JOIN nodes n ON n.id=e.node_id AND n.current_incarnation_id=e.incarnation_id
				 WHERE EXISTS(SELECT 1 FROM jsonb_array_elements(e.envelope->'signed'->'device_bindings') v WHERE v->>'member_id'=$1)
				 OR EXISTS(SELECT 1 FROM jsonb_array_elements(e.envelope->'signed'->'member_rate_policies') v WHERE v->>'member_id'=$1)`, member.String())
				if err != nil {
					return nil, err
				}
				ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
				if err != nil {
					return nil, err
				}
				for _, id := range ids {
					noteAffected(affectedNodes, fupOnly, id, false)
				}
			}
			if len(rawGroups) > 0 {
				var gids []uuid.UUID
				for _, raw := range rawGroups {
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
						noteAffected(affectedNodes, fupOnly, nid, false)
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
			noteAffected(affectedNodes, fupOnly, nid, false)
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
		if fupOnly[nid] {
			var filter []uuid.UUID
			if sc := fupScopes[nid]; sc != nil && !sc.all {
				for id := range sc.ids {
					filter = append(filter, id)
				}
			}
			current, err := c.fairUseEnvelopeCurrent(ctx, nid, filter)
			if err != nil {
				return compiled, err
			}
			if current {
				continue
			}
		}
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

// fupMemberScope is the set of members a Fair-Use-only recompile must refresh.
// all is a fleet fair_use activation. ids is one or more generation changes.
type fupMemberScope struct {
	all bool
	ids map[uuid.UUID]struct{}
}

func noteAffected(affected, fupOnly map[uuid.UUID]bool, id uuid.UUID, fromFUP bool) {
	if id == uuid.Nil {
		return
	}
	if !fromFUP {
		affected[id] = true
		delete(fupOnly, id)
		return
	}
	if !affected[id] {
		affected[id] = true
		fupOnly[id] = true
	}
}

func noteFUPScope(scopes map[uuid.UUID]*fupMemberScope, id uuid.UUID, member *uuid.UUID) {
	sc := scopes[id]
	if sc == nil {
		sc = &fupMemberScope{ids: map[uuid.UUID]struct{}{}}
		scopes[id] = sc
	}
	if member == nil {
		sc.all = true
		return
	}
	sc.ids[*member] = struct{}{}
}

// exitsCarryingMembers returns published, registered exits whose groups overlap a
// serving entitlement, plus published exits whose current envelope already names
// one of those members. memberIDs nil selects every serving member.
func (c *Compiler) exitsCarryingMembers(ctx context.Context, memberIDs []uuid.UUID) ([]uuid.UUID, error) {
	if c.pool == nil {
		return nil, errors.New("desiredstate: database required to select fair-use exits")
	}
	asOf := time.Now().UTC()
	if c.now != nil {
		asOf = c.now().UTC()
	}
	allMembers := memberIDs == nil
	if memberIDs == nil {
		memberIDs = []uuid.UUID{}
	}
	rows, err := c.pool.Query(ctx, `
		WITH serving AS (
			SELECT m.id AS member_id, ent.projection->'exit_group_ids' AS groups
			FROM members m
			JOIN entitlements ent ON ent.member_id = m.id
			WHERE m.status = 'active'
			  AND NOT m.admin_suspended
			  AND NOT ent.admin_suspended
			  AND ent.state NOT IN ('pending_payment','suspended','expired','closed')
			  AND ent.valid_from <= $3
			  AND $3 < ent.effective_service_end
			  AND ($1::bool OR m.id = ANY($2::uuid[]))
		)
		SELECT DISTINCT e.node_id
		FROM exits e
		JOIN nodes n ON n.id = e.node_id
		JOIN exit_groups eg ON eg.exit_id = e.id
		JOIN serving s ON EXISTS (
			SELECT 1 FROM jsonb_array_elements_text(s.groups) g(id) WHERE g.id = eg.group_id::text
		)
		WHERE e.publication_intent = 'published'
		  AND n.lifecycle_state = 'registered'
		  AND n.current_incarnation_id IS NOT NULL
		UNION
		SELECT DISTINCT n.id
		FROM desired_state_envelopes env
		JOIN nodes n ON n.id = env.node_id AND n.current_incarnation_id = env.incarnation_id
		JOIN exits e ON e.node_id = n.id
		WHERE e.publication_intent = 'published'
		  AND n.lifecycle_state = 'registered'
		  AND EXISTS (
			SELECT 1
			FROM jsonb_array_elements(COALESCE(env.envelope->'signed'->'member_rate_policies', '[]'::jsonb)) v
			WHERE $1::bool OR (v->>'member_id')::uuid = ANY($2::uuid[])
		  )`, allMembers, memberIDs, asOf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// fairUseEnvelopeCurrent reports whether nodeID's stored envelope already carries
// the fleet fair-use generation and the member rate generation and mode Compile
// would write. memberFilter empty checks every entitled member on the exit.
// A matching envelope is not compiled again, so an outbox replay does not bump
// desired_config_generation.
func (c *Compiler) fairUseEnvelopeCurrent(ctx context.Context, nodeID uuid.UUID, memberFilter []uuid.UUID) (bool, error) {
	if c.pool == nil {
		return false, errors.New("desiredstate: database required to read fair-use envelope")
	}
	asOf := time.Now().UTC()
	if c.now != nil {
		asOf = c.now().UTC()
	}
	var raw []byte
	err := c.pool.QueryRow(ctx, `
		SELECT env.envelope
		FROM desired_state_envelopes env
		JOIN nodes n ON n.id = env.node_id AND n.current_incarnation_id = env.incarnation_id
		WHERE env.node_id = $1`, nodeID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var env api.DesiredStateEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return false, nil
	}

	polReader := c.policies
	if polReader == nil {
		polReader = NewDBPolicyReader(c.pool)
	}
	snap, err := polReader.ReadPolicySnapshot(ctx, nodeID, asOf)
	if err != nil {
		return false, err
	}
	fleetGen := snap.FairUseGeneration
	if fleetGen == 0 {
		fleetGen = 1
	}
	if uint64(env.Signed.PolicyRefs.FupGeneration) != fleetGen {
		return false, nil
	}

	rows, err := c.pool.Query(ctx, `
		SELECT eg.group_id
		FROM exits e
		JOIN exit_groups eg ON eg.exit_id = e.id
		WHERE e.node_id = $1 AND e.publication_intent = 'published'`, nodeID)
	if err != nil {
		return false, err
	}
	var groupIDs []uuid.UUID
	for rows.Next() {
		var gid uuid.UUID
		if err := rows.Scan(&gid); err != nil {
			rows.Close()
			return false, err
		}
		groupIDs = append(groupIDs, gid)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return false, err
	}
	rows.Close()

	entReader := c.entitlements
	if entReader == nil {
		entReader = NewDBEntitlementReader(c.pool)
	}
	var entitled []MemberEntitlement
	if len(groupIDs) > 0 {
		entitled, err = entReader.ListEntitledMembersForGroups(ctx, groupIDs, asOf)
		if err != nil {
			return false, err
		}
	}
	filter := map[uuid.UUID]struct{}{}
	for _, id := range memberFilter {
		filter[id] = struct{}{}
	}
	byMember := make(map[uuid.UUID]api.MemberRatePolicy, len(env.Signed.MemberRatePolicies))
	for _, pol := range env.Signed.MemberRatePolicies {
		byMember[pol.MemberId] = pol
	}
	want := map[uuid.UUID]struct {
		gen  uint64
		mode api.MemberRatePolicyMode
	}{}
	for _, m := range entitled {
		if m.PlanDownBps <= 0 || m.PlanUpBps <= 0 {
			continue
		}
		if len(filter) > 0 {
			if _, ok := filter[m.MemberID]; !ok {
				continue
			}
		}
		gen := m.FairUsePolicyGen
		if !m.HasFairUseGen && gen == 0 {
			gen = fleetGen
		}
		mode := m.Mode
		if mode == "" {
			mode = api.MemberRatePolicyModeEnforce
		}
		want[m.MemberID] = struct {
			gen  uint64
			mode api.MemberRatePolicyMode
		}{gen, mode}
	}
	if len(filter) > 0 {
		for id := range filter {
			if _, ok := want[id]; ok {
				continue
			}
			// Compile would omit a member this exit no longer serves.
			if _, stillThere := byMember[id]; stillThere {
				return false, nil
			}
		}
	}
	for id, w := range want {
		pol, ok := byMember[id]
		if !ok || uint64(pol.Generation) != w.gen || pol.Mode != w.mode {
			return false, nil
		}
	}
	return true, nil
}

// nodeCapacityRatePair converts an api.NodeCapacity into the directional per-member rate cap
// passed to fup.BuildNodeMemberRatePolicy (§11.2, §11.3).
func nodeCapacityRatePair(nodeCap *api.NodeCapacity) *api.RatePair {
	if nodeCap == nil {
		return nil
	}
	downCap := nodeCap.DownBps
	if v, err := nodeCap.MemberDownBpsCap.Get(); err == nil && v > 0 {
		if downCap <= 0 || v < downCap {
			downCap = v
		}
	}
	upCap := nodeCap.UpBps
	if v, err := nodeCap.MemberUpBpsCap.Get(); err == nil && v > 0 {
		if upCap <= 0 || v < upCap {
			upCap = v
		}
	}
	if downCap <= 0 && upCap <= 0 {
		return nil
	}
	if downCap <= 0 {
		downCap = math.MaxInt64
	}
	if upCap <= 0 {
		upCap = math.MaxInt64
	}
	return &api.RatePair{
		DownBps: downCap,
		UpBps:   upCap,
	}
}
