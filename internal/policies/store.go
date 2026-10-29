package policies

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"context"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

const (
	defaultSigningKeyRef = "policy-key-v1-test-only" // TEST ONLY - NOT A SECRET
	defaultBundleTTL     = 24 * time.Hour
)

var defaultUpstreamDNS = []string{"https://dns.example.com/dns-query"}

// Store manages policy_definitions, immutable policy_revisions, DNS bundle
// compilation, and atomic policy activation (§14.2, §14.5, §15.3).
type Store struct {
	pool          *pgxpool.Pool
	now           func() time.Time
	signingKeyRef string
	upstreamDNS   []string
	releaseSvc    *releases.Service
}

// StoreOption configures a Store.
type StoreOption func(*Store)

// WithClock sets a custom clock for deterministic tests.
func WithClock(now func() time.Time) StoreOption {
	return func(s *Store) {
		if now != nil {
			s.now = now
		}
	}
}

// WithSigningKeyRef sets the signing key reference recorded on new policy revisions.
func WithSigningKeyRef(keyRef string) StoreOption {
	return func(s *Store) {
		if strings.TrimSpace(keyRef) != "" {
			s.signingKeyRef = strings.TrimSpace(keyRef)
		}
	}
}

// WithUpstreamDNS configures the upstream DNS servers embedded into compiled
// DNS bundles on dns_profile activation.
func WithUpstreamDNS(upstreams []string) StoreOption {
	return func(s *Store) {
		if len(upstreams) > 0 {
			s.upstreamDNS = append([]string(nil), upstreams...)
		}
	}
}

// WithReleaseService injects the release service used for release_support
// policy activation (`releases.ApplySupportTx`).
func WithReleaseService(rs *releases.Service) StoreOption {
	return func(s *Store) {
		if rs != nil {
			s.releaseSvc = rs
		}
	}
}

// NewStore creates a new policy Store backed by pool.
func NewStore(pool *pgxpool.Pool, opts ...StoreOption) *Store {
	s := &Store{
		pool:          pool,
		now:           func() time.Time { return time.Now().UTC() },
		signingKeyRef: defaultSigningKeyRef,
		upstreamDNS:   defaultUpstreamDNS,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Store) releaseService() (*releases.Service, error) {
	if s.releaseSvc != nil {
		return s.releaseSvc, nil
	}
	trust, err := envelope.NewTrustStore()
	if err != nil {
		return nil, err
	}
	v, err := releases.NewValidator(trust, []string{"https://example.invalid"})
	if err != nil {
		return nil, err
	}
	return &releases.Service{Pool: s.pool, Validator: v}, nil
}

// runInTx runs fn inside callerTx if httpapi.Transaction(ctx) is present, or
// opens a new db.InTx otherwise. managedIdem is true only when Store owns the
// transaction (i.e. not wrapped by httpapi.Commands middleware).
func (s *Store) runInTx(ctx context.Context, opts db.TxOptions, fn func(tx pgx.Tx, managedIdem bool) error) error {
	if callerTx, err := httpapi.Transaction(ctx); err == nil && callerTx != nil {
		return fn(callerTx, false)
	}
	return db.InTx(ctx, s.pool, opts, func(tx pgx.Tx) error {
		return fn(tx, true)
	})
}

func requestUUIDFromContext(ctx context.Context) uuid.UUID {
	if raw, ok := apierr.RequestIDFromContext(ctx); ok {
		if id, err := uuid.Parse(raw); err == nil && id != uuid.Nil {
			return id
		}
	}
	if id, err := uuid.NewV7(); err == nil {
		return id
	}
	return uuid.New()
}

func principalScope(principal string, actorID uuid.UUID) string {
	if p := strings.TrimSpace(principal); p != "" {
		return p
	}
	return "admin:" + actorID.String()
}

// CreateRevisionInput carries the parameters for POST /admin/policies (§14.2).
type CreateRevisionInput struct {
	DefinitionID   *uuid.UUID
	Code           *string
	Type           api.PolicyType
	SchemaVersion  int
	Reason         string
	Payload        json.RawMessage
	ActorID        uuid.UUID
	SigningKeyRef  string
	IdempotencyKey *uuid.UUID
	Principal      string
}

// RevisionRecord represents an immutable stored policy revision.
type RevisionRecord struct {
	RevisionID    uuid.UUID
	DefinitionID  uuid.UUID
	Code          string
	Type          api.PolicyType
	Revision      decstr.U64
	SchemaVersion int
	Payload       json.RawMessage
	PayloadHash   [32]byte
	PayloadSHA256 string
	SigningKeyRef string
	Reason        string
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
	Validated     *ValidatedPayload
}

// ToAPI converts RevisionRecord to the generated api.PolicyRevision union.
func (r *RevisionRecord) ToAPI() (api.PolicyRevision, error) {
	var out api.PolicyRevision
	env := map[string]any{
		"revision_id":    r.RevisionID.String(),
		"definition_id":  r.DefinitionID.String(),
		"type":           string(r.Type),
		"revision":       r.Revision.String(),
		"schema_version": r.SchemaVersion,
		"payload":        json.RawMessage(r.Payload),
		"payload_sha256": r.PayloadSHA256,
		"reason":         r.Reason,
		"created_by":     r.CreatedBy.String(),
		"created_at":     r.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}

// Summary converts RevisionRecord to api.PolicyRevisionSummary.
func (r *RevisionRecord) Summary() api.PolicyRevisionSummary {
	return api.PolicyRevisionSummary{
		RevisionId:    openapi_types.UUID(r.RevisionID),
		Revision:      r.Revision,
		SchemaVersion: r.SchemaVersion,
		PayloadSha256: r.PayloadSHA256,
		Reason:        r.Reason,
		CreatedBy:     openapi_types.UUID(r.CreatedBy),
		CreatedAt:     r.CreatedAt.UTC(),
	}
}

// CreateRevision validates the policy payload (JSON Schema + type-specific
// semantics + fixed isolation baseline), computes its RFC 8785 canonical JSON
// SHA-256, and stores an immutable revision row with UNIQUE(definition_id, revision).
func (s *Store) CreateRevision(ctx context.Context, in CreateRevisionInput) (*RevisionRecord, error) {
	if in.ActorID == uuid.Nil {
		return nil, newValidationError("/actor_id", ReasonSchemaInvalid, "actor_id is required")
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" || len(in.Reason) > 500 {
		return nil, newValidationError("/reason", ReasonSchemaInvalid, "reason is required and must be 1..500 characters")
	}
	if in.DefinitionID == nil {
		if in.Code == nil || !codePattern.MatchString(*in.Code) {
			return nil, newValidationError("/code", ReasonSchemaInvalid, "code is required when definition_id is omitted and must match ^[a-z0-9][a-z0-9_-]{0,63}$")
		}
	} else if *in.DefinitionID == uuid.Nil {
		return nil, newValidationError("/definition_id", ReasonSchemaInvalid, "definition_id must not be nil UUID")
	}

	validated, err := ValidatePayload(in.Type, in.SchemaVersion, in.Payload, ValidationOptions{
		ReferenceTime: s.now(),
	})
	if err != nil {
		return nil, err
	}

	keyRef := strings.TrimSpace(in.SigningKeyRef)
	if keyRef == "" {
		keyRef = s.signingKeyRef
	}

	var out *RevisionRecord
	reqID := requestUUIDFromContext(ctx)
	err = s.runInTx(ctx, db.TxOptions{}, func(tx pgx.Tx, managedIdem bool) error {
		var idemRecord db.IdempotencyRecord
		if managedIdem && in.IdempotencyKey != nil {
			scope, err := buildCreateIdempotencyScope(in, validated.PayloadHash)
			if err != nil {
				return err
			}
			idemRecord, err = db.BeginIdempotent(ctx, tx, scope, 0)
			if err != nil {
				if errors.Is(err, db.ErrIdempotencyConflict) {
					return apierr.Wrap(err, apierr.IdempotencyConflict, "Idempotency-Key was already used with a different request payload")
				}
				return err
			}
			if !idemRecord.New {
				if idemRecord.ResourceID == nil || idemRecord.ResponseStatus == nil {
					return apierr.New(apierr.StateConflict, "an identical policy creation request is still in progress")
				}
				rec, err := getRevisionTx(ctx, tx, *idemRecord.ResourceID)
				if err != nil {
					return err
				}
				out = rec
				return nil
			}
		}

		// Perform DB reference and baseline checks per policy type.
		if err := s.validateReferencesTx(ctx, tx, validated); err != nil {
			return err
		}

		defID, defCode, err := resolveOrCreateDefinitionTx(ctx, tx, in.DefinitionID, in.Code, in.Type)
		if err != nil {
			return err
		}

		var nextRevStr string
		err = tx.QueryRow(ctx,
			`SELECT (coalesce(max(revision), 0) + 1)::text FROM policy_revisions WHERE definition_id = $1`,
			defID,
		).Scan(&nextRevStr)
		if err != nil {
			return fmt.Errorf("compute next policy revision: %w", err)
		}
		nextRev, err := decstr.Parse(nextRevStr)
		if err != nil {
			return fmt.Errorf("parse next policy revision %q: %w", nextRevStr, err)
		}

		revID, err := uuid.NewV7()
		if err != nil {
			revID = uuid.New()
		}

		var createdAt time.Time
		err = tx.QueryRow(ctx,
			`INSERT INTO policy_revisions (
				id, definition_id, type, revision, schema_version,
				payload, payload_hash, signing_key_ref, created_by, reason
			) VALUES ($1, $2, $3, $4::numeric, $5, $6::jsonb, $7, $8, $9, $10)
			RETURNING created_at`,
			revID,
			defID,
			string(in.Type),
			nextRev.String(),
			in.SchemaVersion,
			validated.CanonicalJSON,
			validated.PayloadHash[:],
			keyRef,
			in.ActorID.String(),
			in.Reason,
		).Scan(&createdAt)
		if err != nil {
			if errors.Is(err, db.ErrUniqueViolation) {
				return apierr.Wrap(err, apierr.StateConflict, "concurrent policy revision conflict")
			}
			return fmt.Errorf("insert policy_revision: %w", err)
		}

		// Sync plans and plan_revisions tables when type is "plan" (§14.2, §15.2).
		if in.Type == api.PolicyTypePlan && validated.Plan != nil {
			if err := syncPlanRevisionTx(ctx, tx, defID, defCode, revID, nextRev, in.SchemaVersion, validated.Plan, validated.CanonicalJSON); err != nil {
				return err
			}
		}

		afterJSON, err := json.Marshal(map[string]any{
			"revision_id":    revID.String(),
			"definition_id":  defID.String(),
			"type":           string(in.Type),
			"code":           defCode,
			"revision":       nextRev.String(),
			"schema_version": in.SchemaVersion,
			"payload_sha256": validated.PayloadSHA256,
			"reason":         in.Reason,
		})
		if err != nil {
			return err
		}
		if err := db.AppendAudit(ctx, tx, db.AuditEvent{
			Actor:         in.ActorID.String(),
			ActorType:     string(api.AuditEventActorTypeAdmin),
			Action:        "policy.revision.create",
			Resource:      string(api.ResourceTypePolicyRevision) + ":" + revID.String(),
			RequestID:     &reqID,
			Result:        string(api.AuditEventResultSucceeded),
			SchemaVersion: 1,
			Before:        []byte(`{}`),
			After:         afterJSON,
		}); err != nil {
			return fmt.Errorf("append audit for policy revision create: %w", err)
		}

		if managedIdem && in.IdempotencyKey != nil {
			if err := db.CompleteIdempotent(ctx, tx, idemRecord.ID, db.IdempotencyResult{
				ResourceType:   "policy_revision",
				ResourceID:     &revID,
				ResponseStatus: 201,
			}); err != nil {
				return err
			}
		}

		out = &RevisionRecord{
			RevisionID:    revID,
			DefinitionID:  defID,
			Code:          defCode,
			Type:          in.Type,
			Revision:      nextRev,
			SchemaVersion: in.SchemaVersion,
			Payload:       validated.CanonicalJSON,
			PayloadHash:   validated.PayloadHash,
			PayloadSHA256: validated.PayloadSHA256,
			SigningKeyRef: keyRef,
			Reason:        in.Reason,
			CreatedBy:     in.ActorID,
			CreatedAt:     createdAt.UTC(),
			Validated:     validated,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func resolveOrCreateDefinitionTx(ctx context.Context, tx pgx.Tx, defID *uuid.UUID, code *string, policyType api.PolicyType) (uuid.UUID, string, error) {
	if defID != nil {
		var id uuid.UUID
		var dbType, dbCode string
		err := tx.QueryRow(ctx,
			`SELECT id, type, code FROM policy_definitions WHERE id = $1 FOR UPDATE`,
			*defID,
		).Scan(&id, &dbType, &dbCode)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return uuid.Nil, "", apierr.Newf(apierr.ResourceNotFound, "policy definition %s not found", *defID)
			}
			return uuid.Nil, "", err
		}
		if dbType != string(policyType) {
			return uuid.Nil, "", newValidationError("/type", ReasonUnknownPolicyType, fmt.Sprintf("definition %s has type %q, cannot create %q revision", *defID, dbType, policyType))
		}
		if code != nil && *code != dbCode {
			return uuid.Nil, "", newValidationError("/code", ReasonSchemaInvalid, fmt.Sprintf("definition %s has code %q, got %q", *defID, dbCode, *code))
		}
		return id, dbCode, nil
	}

	newID, err := uuid.NewV7()
	if err != nil {
		newID = uuid.New()
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO policy_definitions (id, type, code)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (type, code) DO NOTHING`,
		newID,
		string(policyType),
		*code,
	); err != nil {
		return uuid.Nil, "", fmt.Errorf("insert policy_definition: %w", err)
	}
	var id uuid.UUID
	var dbCode string
	// Lock definition row to serialize revision numbering without executing a
	// no-op UPDATE that would violate F-14 (strictly increasing resource_version).
	if err := tx.QueryRow(ctx,
		`SELECT id, code FROM policy_definitions WHERE type = $1 AND code = $2 FOR UPDATE`,
		string(policyType),
		*code,
	).Scan(&id, &dbCode); err != nil {
		return uuid.Nil, "", fmt.Errorf("lock policy_definition: %w", err)
	}
	return id, dbCode, nil
}

func syncPlanRevisionTx(ctx context.Context, tx pgx.Tx, defID uuid.UUID, code string, revID uuid.UUID, rev decstr.U64, schemaVersion int, plan *api.PlanPayload, canonicalJSON []byte) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO plans (id, code) VALUES ($1, $2)
		 ON CONFLICT (code) DO NOTHING`,
		defID, code,
	)
	if err != nil {
		return fmt.Errorf("sync plans row: %w", err)
	}
	var planID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM plans WHERE code = $1 FOR UPDATE`, code).Scan(&planID); err != nil {
		return fmt.Errorf("lookup plan by code %q: %w", code, err)
	}
	exitGroupIDs := make([]uuid.UUID, 0, len(plan.ExitGroupIds))
	for _, gid := range plan.ExitGroupIds {
		exitGroupIDs = append(exitGroupIDs, uuid.UUID(gid))
	}
	fupRevID := uuid.UUID(plan.FairUseRevisionId)
	_, err = tx.Exec(ctx,
		`INSERT INTO plan_revisions (
			id, plan_id, revision, device_limit, exit_group_ids,
			fup_policy_revision_id, provider, provider_price_ref, schema_version, payload
		) VALUES ($1, $2, $3::numeric, $4, $5, $6, $7, $8, $9, $10::jsonb)`,
		revID, planID, rev.String(), plan.DeviceLimit, exitGroupIDs,
		fupRevID, plan.Price.Provider, plan.Price.ProviderPriceRef, schemaVersion, canonicalJSON,
	)
	if err != nil {
		return fmt.Errorf("sync plan_revisions row: %w", err)
	}
	return nil
}

// validateReferencesTx verifies that foreign references in a policy payload
// exist and do not violate the fixed isolation baseline.
func (s *Store) validateReferencesTx(ctx context.Context, tx pgx.Tx, v *ValidatedPayload) error {
	switch v.Type {
	case api.PolicyTypePlan:
		p := v.Plan
		// Every exit_group_id must exist in member_groups (baseline guard: a plan
		// cannot grant an exit group that does not exist).
		for i, gid := range p.ExitGroupIds {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM member_groups WHERE id = $1)`, uuid.UUID(gid)).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return newValidationError(
					fmt.Sprintf("/payload/exit_group_ids/%d", i),
					ReasonPlanUnknownExitGroup,
					fmt.Sprintf("exit_group_id %s does not exist (cannot grant non-existent exit group)", uuid.UUID(gid)),
				)
			}
		}

		// fair_use_revision_id must exist with type = 'fair_use'.
		var fupType string
		err := tx.QueryRow(ctx, `SELECT type FROM policy_revisions WHERE id = $1`, uuid.UUID(p.FairUseRevisionId)).Scan(&fupType)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return newValidationError("/payload/fair_use_revision_id", ReasonPlanMissingFairUseRef, fmt.Sprintf("fair_use_revision_id %s does not exist", uuid.UUID(p.FairUseRevisionId)))
			}
			return err
		}
		if fupType != string(api.PolicyTypeFairUse) {
			return newValidationError("/payload/fair_use_revision_id", ReasonPlanMissingFairUseRef, fmt.Sprintf("revision %s has type %q, want fair_use", uuid.UUID(p.FairUseRevisionId), fupType))
		}

		// dns_profile_revision_id must exist with type = 'dns_profile'.
		var dnsType string
		err = tx.QueryRow(ctx, `SELECT type FROM policy_revisions WHERE id = $1`, uuid.UUID(p.DnsProfileRevisionId)).Scan(&dnsType)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return newValidationError("/payload/dns_profile_revision_id", ReasonPlanMissingDNSProfile, fmt.Sprintf("dns_profile_revision_id %s does not exist", uuid.UUID(p.DnsProfileRevisionId)))
			}
			return err
		}
		if dnsType != string(api.PolicyTypeDnsProfile) {
			return newValidationError("/payload/dns_profile_revision_id", ReasonPlanMissingDNSProfile, fmt.Sprintf("revision %s has type %q, want dns_profile", uuid.UUID(p.DnsProfileRevisionId), dnsType))
		}

	case api.PolicyTypeDnsProfile:
		_, _, err := s.buildAndValidateDNSProfileTx(ctx, tx, 1, *v.DNSProfile)
		if err != nil {
			return err
		}

	case api.PolicyTypeReleaseSupport:
		// If the releases table has registered releases, verify that all referenced
		// release IDs exist in releases.
		var releaseTableCount int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM releases`).Scan(&releaseTableCount); err != nil {
			return err
		}
		if releaseTableCount > 0 {
			p := v.ReleaseSupport
			checkIDs := func(ids []openapi_types.UUID, fieldPrefix string) error {
				for i, id := range ids {
					var exists bool
					if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM releases WHERE id = $1)`, uuid.UUID(id)).Scan(&exists); err != nil {
						return err
					}
					if !exists {
						return newValidationError(
							fmt.Sprintf("%s/%d", fieldPrefix, i),
							ReasonReleaseUnknownID,
							fmt.Sprintf("release_id %s does not exist in registered releases", uuid.UUID(id)),
						)
					}
				}
				return nil
			}
			if err := checkIDs(p.MinimumServingVersion.ServingReleaseIds, "/payload/minimum_serving_version/serving_release_ids"); err != nil {
				return err
			}
			if err := checkIDs(p.ApprovedReleaseIds, "/payload/approved_release_ids"); err != nil {
				return err
			}
			if err := checkIDs(p.RevokedReleaseIds, "/payload/revoked_release_ids"); err != nil {
				return err
			}
		}
	}
	return nil
}

// buildAndValidateDNSProfileTx loads the referenced domain_rule_set and
// threat_feed revisions, checks cross-revision rule conflicts and profile_id
// targeting, and validates the combined draft via internal/dnspolicy.
func (s *Store) buildAndValidateDNSProfileTx(ctx context.Context, tx pgx.Tx, bundleRevision uint64, p api.DnsProfilePayload) (*dnspolicy.Profile, *dnspolicy.SourceRegistry, error) {
	seenRuleIDs := make(map[string]uuid.UUID)
	seenDomains := make(map[string]uuid.UUID)
	var dnsRules []dnspolicy.Rule

	for i, revID := range p.DomainRuleSetRevisionIds {
		uid := uuid.UUID(revID)
		var revType string
		var rawPayload []byte
		err := tx.QueryRow(ctx, `SELECT type, payload FROM policy_revisions WHERE id = $1`, uid).Scan(&revType, &rawPayload)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil, newValidationError(
					fmt.Sprintf("/payload/domain_rule_set_revision_ids/%d", i),
					ReasonDNSUnknownRuleSetRevision,
					fmt.Sprintf("domain_rule_set revision %s not found", uid),
				)
			}
			return nil, nil, err
		}
		if revType != string(api.PolicyTypeDomainRuleSet) {
			return nil, nil, newValidationError(
				fmt.Sprintf("/payload/domain_rule_set_revision_ids/%d", i),
				ReasonDNSUnknownRuleSetRevision,
				fmt.Sprintf("policy revision %s has type %q, want domain_rule_set", uid, revType),
			)
		}
		var rs api.DomainRuleSetPayload
		if err := json.Unmarshal(rawPayload, &rs); err != nil {
			return nil, nil, err
		}
		if err := ValidateDomainRuleSet(rs); err != nil {
			return nil, nil, err
		}
		for _, r := range rs.Rules {
			ridStr := uuid.UUID(r.RuleId).String()
			if profile, err := r.ProfileId.Get(); err == nil && profile != p.ProfileCode {
				return nil, nil, newValidationError(
					fmt.Sprintf("/payload/domain_rule_set_revision_ids/%d", i),
					ReasonDNSProfileMismatch,
					fmt.Sprintf("domain rule %q targets profile %q, cannot be included in profile %q", ridStr, profile, p.ProfileCode),
				)
			}
			if prevSet, dup := seenRuleIDs[ridStr]; dup {
				return nil, nil, newConflictError(
					fmt.Sprintf("/payload/domain_rule_set_revision_ids/%d", i),
					ReasonDNSDuplicateRuleID,
					fmt.Sprintf("duplicate rule_id %q across domain_rule_set revisions %s and %s", ridStr, prevSet, uid),
				)
			}
			seenRuleIDs[ridStr] = uid

			norm, err := dnspolicy.NormalizeDomain(r.Domain)
			if err != nil {
				return nil, nil, newValidationError("/payload/domain_rule_set_revision_ids", ReasonDNSDomainNotCanonical, err.Error())
			}
			if prevSet, dup := seenDomains[norm.Canonical]; dup {
				return nil, nil, newConflictError(
					fmt.Sprintf("/payload/domain_rule_set_revision_ids/%d", i),
					ReasonDNSSameLayerConflict,
					fmt.Sprintf("conflicting or duplicate rule for domain %q across domain_rule_set revisions %s and %s", norm.Canonical, prevSet, uid),
				)
			}
			seenDomains[norm.Canonical] = uid

			layer := dnspolicy.LayerAdminBlock
			action := dnspolicy.ActionBlock
			if r.Action == api.DomainRuleActionAllow {
				layer = dnspolicy.LayerAdminAllow
				action = dnspolicy.ActionAllow
			}
			dnsRules = append(dnsRules, dnspolicy.Rule{
				RuleID:            ridStr,
				ProfileID:         p.ProfileCode,
				Domain:            norm.Canonical,
				IncludeSubdomains: r.IncludeSubdomains,
				Action:            action,
				Layer:             layer,
				Reason:            r.Reason,
				Revision:          r.Revision.String(),
			})
		}
	}

	var dnsSources []dnspolicy.ListSource
	var enabledSourceIDs []string
	registry := dnspolicy.NewDefaultSourceRegistry()
	seenFeedCodes := make(map[string]uuid.UUID, len(p.ThreatFeedRevisionIds))

	for i, revID := range p.ThreatFeedRevisionIds {
		uid := uuid.UUID(revID)
		var revType, revNumStr string
		var rawPayload []byte
		err := tx.QueryRow(ctx, `SELECT type, revision::text, payload FROM policy_revisions WHERE id = $1`, uid).Scan(&revType, &revNumStr, &rawPayload)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil, newValidationError(
					fmt.Sprintf("/payload/threat_feed_revision_ids/%d", i),
					ReasonDNSUnknownThreatFeedRev,
					fmt.Sprintf("threat_feed revision %s not found", uid),
				)
			}
			return nil, nil, err
		}
		if revType != string(api.PolicyTypeThreatFeed) {
			return nil, nil, newValidationError(
				fmt.Sprintf("/payload/threat_feed_revision_ids/%d", i),
				ReasonDNSUnknownThreatFeedRev,
				fmt.Sprintf("policy revision %s has type %q, want threat_feed", uid, revType),
			)
		}
		var tf api.ThreatFeedPayload
		if err := json.Unmarshal(rawPayload, &tf); err != nil {
			return nil, nil, err
		}
		if err := ValidateThreatFeed(tf); err != nil {
			return nil, nil, err
		}
		if prevRev, dup := seenFeedCodes[tf.FeedCode]; dup {
			return nil, nil, newConflictError(
				fmt.Sprintf("/payload/threat_feed_revision_ids/%d", i),
				ReasonDNSInvalidProfile,
				fmt.Sprintf("duplicate feed_code %q across threat_feed revisions %s and %s", tf.FeedCode, prevRev, uid),
			)
		}
		seenFeedCodes[tf.FeedCode] = uid

		attr, _ := tf.Attribution.Get()
		var srcFormat dnspolicy.SourceFormat
		switch tf.Format {
		case api.ThreatFeedPayloadFormatDomainList:
			srcFormat = dnspolicy.SourceFormatPlain
		case api.ThreatFeedPayloadFormatHosts:
			srcFormat = dnspolicy.SourceFormatHosts
		case api.ThreatFeedPayloadFormatAdblock:
			srcFormat = dnspolicy.SourceFormatAdBlock
		default:
			srcFormat = dnspolicy.SourceFormatPlain
		}

		quality := dnspolicy.SourceQualityMetrics{
			LastUpdate:      tf.ValidFrom,
			SmokeTestResult: dnspolicy.SmokeTestPassed,
		}
		ls := dnspolicy.ListSource{
			SourceID:          tf.FeedCode,
			Name:              tf.Provider + " " + tf.FeedCode,
			Version:           revNumStr,
			ContentSHA256:     tf.ContentSha256,
			License:           tf.LicenseName,
			LicenseURL:        tf.LicenseUrl,
			LicenseStatus:     dnspolicy.LicenseCommercialApproved,
			PermittedUse:      tf.CommercialUseBasis,
			DistributionNotes: tf.CommercialUseBasis,
			AttributionNotes:  attr,
			URL:               tf.SourceUrl,
			Format:            srcFormat,
			Category:          dnspolicy.SourceCategory(tf.Categories[0]),
			UpdateCadence:     fmt.Sprintf("%ds", tf.UpdateIntervalSeconds),
			EnabledByDefault:  true,
			Quality:           quality,
			LastGood: &dnspolicy.SourceSnapshot{
				Version:       revNumStr,
				ContentSHA256: tf.ContentSha256,
				Quality:       quality,
				UpdatedAt:     tf.ValidFrom,
			},
		}
		if err := registry.Register(ls); err != nil {
			return nil, nil, newValidationError(
				fmt.Sprintf("/payload/threat_feed_revision_ids/%d", i),
				ReasonDNSInvalidProfile,
				fmt.Sprintf("register threat feed %q: %v", tf.FeedCode, err),
			)
		}
		dnsSources = append(dnsSources, ls)
		enabledSourceIDs = append(enabledSourceIDs, tf.FeedCode)
	}

	if bundleRevision == 0 {
		bundleRevision = 1
	}
	profName := p.DisplayName.ZhTW
	if p.DisplayName.En != nil && *p.DisplayName.En != "" {
		profName = *p.DisplayName.En
	}
	prof := dnspolicy.Profile{
		ProfileID:         p.ProfileCode,
		Name:              profName,
		Revision:          fmt.Sprintf("%d", bundleRevision),
		Rules:             dnsRules,
		EnabledSources:    enabledSourceIDs,
		Sources:           dnsSources,
		BlockMode:         dnspolicy.BlockingModeNXDOMAIN,
		BlockedTTLSeconds: uint32(p.BlockedTtlSeconds),
	}
	limits := dnspolicy.DefaultDraftLimits()
	if err := dnspolicy.ValidateDraft(prof, registry, &limits); err != nil {
		return nil, nil, newValidationError("/payload", ReasonDNSInvalidProfile, err.Error())
	}
	return &prof, registry, nil
}

// ActivateRevisionInput carries the parameters for POST /admin/policies/{id}/activate (§14.2, §14.5).
type ActivateRevisionInput struct {
	DefinitionID    uuid.UUID
	RevisionID      uuid.UUID
	ExpectedVersion string // Required If-Match header value (428 when empty, 412 on mismatch)
	EffectiveAt     *time.Time
	Reason          string
	ActorID         uuid.UUID
	IdempotencyKey  *uuid.UUID
	Principal       string
}

// ActivateResult contains the outcome of ActivateRevision.
type ActivateResult struct {
	DefinitionID    uuid.UUID
	RevisionID      uuid.UUID
	Generation      decstr.U64
	ResourceVersion decstr.U64
	NoOp            bool
	Operation       api.Operation
	CompiledBundle  *dnspolicy.CompiledBundle
}

// ActivateRevision atomically activates a policy revision under If-Match
// optimistic concurrency (`db.UpdateWithVersion`), increments the definition's
// `generation`, records the `operations` row, `audit_events` row, and
// `outbox_events` row triggering desired-state recompilation in one transaction.
//
// Activating the already-active revision is an idempotent no-op that does not
// increment generation. Activating an older revision is allowed only as an
// explicit new activation with the current If-Match resource_version, which
// produces a new higher generation (never replaying an old generation, §11.5, AT19).
func (s *Store) ActivateRevision(ctx context.Context, in ActivateRevisionInput) (*ActivateResult, error) {
	if in.DefinitionID == uuid.Nil {
		return nil, newValidationError("/definition_id", ReasonSchemaInvalid, "definition_id is required")
	}
	if in.RevisionID == uuid.Nil {
		return nil, newValidationError("/revision_id", ReasonSchemaInvalid, "revision_id is required")
	}
	if in.ActorID == uuid.Nil {
		return nil, newValidationError("/actor_id", ReasonSchemaInvalid, "actor_id is required")
	}
	if strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 500 {
		return nil, newValidationError("/reason", ReasonSchemaInvalid, "reason is required and must be 1..500 characters")
	}

	if strings.TrimSpace(in.ExpectedVersion) == "" {
		return nil, apierr.New(apierr.PreconditionRequired, "If-Match header is required to activate a policy revision")
	}
	expectedVersion := cleanETag(in.ExpectedVersion)

	effectiveAt := s.now()
	if in.EffectiveAt != nil && !in.EffectiveAt.IsZero() {
		effectiveAt = in.EffectiveAt.UTC()
	}

	var out *ActivateResult
	reqID := requestUUIDFromContext(ctx)
	err := s.runInTx(ctx, db.TxOptions{}, func(tx pgx.Tx, managedIdem bool) error {
		var idemRecord db.IdempotencyRecord
		if managedIdem && in.IdempotencyKey != nil {
			scope, err := buildActivateIdempotencyScope(in)
			if err != nil {
				return err
			}
			idemRecord, err = db.BeginIdempotent(ctx, tx, scope, 0)
			if err != nil {
				if errors.Is(err, db.ErrIdempotencyConflict) {
					return apierr.Wrap(err, apierr.IdempotencyConflict, "Idempotency-Key was already used with a different activation request")
				}
				return err
			}
			if !idemRecord.New {
				if idemRecord.OperationID == nil || idemRecord.ResponseStatus == nil {
					return apierr.New(apierr.StateConflict, "activation request is still in progress")
				}
				op, err := getOperationTx(ctx, tx, *idemRecord.OperationID)
				if err != nil {
					return err
				}
				var curGenStr, curVerStr string
				if err := tx.QueryRow(ctx, `SELECT generation::text, resource_version::text FROM policy_definitions WHERE id = $1`, in.DefinitionID).Scan(&curGenStr, &curVerStr); err != nil {
					return err
				}
				curGen, _ := decstr.Parse(curGenStr)
				curVer, _ := decstr.Parse(curVerStr)
				out = &ActivateResult{
					DefinitionID:    in.DefinitionID,
					RevisionID:      in.RevisionID,
					Generation:      curGen,
					ResourceVersion: curVer,
					NoOp:            true,
					Operation:       op,
				}
				return nil
			}
		}

		// Lock the policy definition row.
		var defType, defCode string
		var activeRevID *uuid.UUID
		var curGenStr, curVerStr string
		err := tx.QueryRow(ctx,
			`SELECT type, code, active_revision_id, generation::text, resource_version::text
			 FROM policy_definitions
			 WHERE id = $1
			 FOR UPDATE`,
			in.DefinitionID,
		).Scan(&defType, &defCode, &activeRevID, &curGenStr, &curVerStr)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierr.Newf(apierr.ResourceNotFound, "policy definition %s not found", in.DefinitionID)
			}
			return err
		}

		if _, err := decstr.Parse(expectedVersion); err != nil || curVerStr != expectedVersion {
			return apierr.Wrap(
				db.ErrVersionConflict,
				apierr.ResourceVersionConflict,
				"policy definition resource_version does not match If-Match",
			).WithDetail("current_resource_version", curVerStr)
		}

		// Verify the target revision belongs to this definition and re-validate it.
		revRec, err := getRevisionTx(ctx, tx, in.RevisionID)
		if err != nil {
			return err
		}
		if revRec.DefinitionID != in.DefinitionID {
			return apierr.Newf(apierr.ResourceNotFound, "revision %s does not belong to definition %s", in.RevisionID, in.DefinitionID)
		}
		validated, err := ValidatePayload(revRec.Type, revRec.SchemaVersion, revRec.Payload, ValidationOptions{})
		if err != nil {
			return err
		}
		if err := s.validateReferencesTx(ctx, tx, validated); err != nil {
			return err
		}

		curGen, err := decstr.Parse(curGenStr)
		if err != nil {
			return err
		}
		curVer, err := decstr.Parse(curVerStr)
		if err != nil {
			return err
		}

		// Same revision activated twice is a no-op: do not increment generation
		// or enqueue duplicate recompilation work.
		if activeRevID != nil && *activeRevID == in.RevisionID {
			op, err := getOrCreateNoOpActivateOperationTx(ctx, tx, in, reqID)
			if err != nil {
				return err
			}
			if managedIdem && in.IdempotencyKey != nil {
				opID := uuid.UUID(op.OperationId)
				if err := db.CompleteIdempotent(ctx, tx, idemRecord.ID, db.IdempotencyResult{
					ResourceType:   "policy_definition",
					ResourceID:     &in.DefinitionID,
					OperationID:    &opID,
					ResponseStatus: 202,
				}); err != nil {
					return err
				}
			}
			out = &ActivateResult{
				DefinitionID:    in.DefinitionID,
				RevisionID:      in.RevisionID,
				Generation:      curGen,
				ResourceVersion: curVer,
				NoOp:            true,
				Operation:       op,
			}
			return nil
		}

		newGen := curGen + 1
		newVerStr, err := db.UpdateWithVersion(ctx, tx, "policy_definitions", in.DefinitionID, expectedVersion, map[string]any{
			"active_revision_id": in.RevisionID,
			"generation":         newGen.String(),
		})
		if err != nil {
			if errors.Is(err, db.ErrVersionConflict) {
				return apierr.Wrap(err, apierr.ResourceVersionConflict, "policy definition resource_version conflict").
					WithDetail("current_resource_version", curVerStr)
			}
			return err
		}
		newVer, err := decstr.Parse(newVerStr)
		if err != nil {
			return err
		}

		// Perform type-specific activation work inside the same transaction.
		var compiledBundle *dnspolicy.CompiledBundle
		switch revRec.Type {
		case api.PolicyTypePlan:
			if _, err := tx.Exec(ctx,
				`UPDATE plans SET active_revision_id = $1, resource_version = resource_version + 1 WHERE code = $2`,
				in.RevisionID, defCode,
			); err != nil {
				return fmt.Errorf("update plans active_revision_id: %w", err)
			}

		case api.PolicyTypeDnsProfile:
			bundle, err := s.compileAndStoreDNSBundleTx(ctx, tx, in.RevisionID, uint64(newGen), *validated.DNSProfile)
			if err != nil {
				return err
			}
			compiledBundle = bundle

		case api.PolicyTypeReleaseSupport:
			rs, err := s.releaseService()
			if err != nil {
				return err
			}
			if err := rs.ApplySupportTx(ctx, tx, in.ActorID, *validated.ReleaseSupport, in.Reason); err != nil {
				return err
			}
		}

		// Create the asynchronous operation row (§14.5, F-10: starts pending with a required step).
		opID, err := uuid.NewV7()
		if err != nil {
			opID = uuid.New()
		}
		stepsJSON := []byte(`{"desired_state_compile":{"required":true,"status":"pending"}}`)
		var opCreatedAt time.Time
		if err := tx.QueryRow(ctx,
			`INSERT INTO operations (id, kind, status, principal, resource_type, resource_id, request_id, resource_version, desired_generation, steps)
			 VALUES ($1, 'policy.activate', 'pending', $2, 'policy_definition', $3, $4, $5::numeric, $6::numeric, $7::jsonb)
			 RETURNING created_at`,
			opID, principalScope(in.Principal, in.ActorID), in.DefinitionID, reqID, newVer.String(), newGen.String(), stepsJSON,
		).Scan(&opCreatedAt); err != nil {
			return fmt.Errorf("insert policy.activate operation: %w", err)
		}
		opUpdatedAt := opCreatedAt

		// Record audit event.
		beforeMap := map[string]any{
			"generation":       curGen.String(),
			"resource_version": curVer.String(),
		}
		if activeRevID != nil {
			beforeMap["active_revision_id"] = activeRevID.String()
		}
		beforeJSON, _ := json.Marshal(beforeMap)

		afterMap := map[string]any{
			"active_revision_id": in.RevisionID.String(),
			"revision":           revRec.Revision.String(),
			"generation":         newGen.String(),
			"resource_version":   newVer.String(),
			"effective_at":       effectiveAt.Format(time.RFC3339Nano),
			"operation_id":       opID.String(),
			"reason":             in.Reason,
		}
		afterJSON, _ := json.Marshal(afterMap)

		if err := db.AppendAudit(ctx, tx, db.AuditEvent{
			Actor:         in.ActorID.String(),
			ActorType:     string(api.AuditEventActorTypeAdmin),
			Action:        "policy.activate",
			Resource:      string(api.ResourceTypePolicyDefinition) + ":" + in.DefinitionID.String(),
			RequestID:     &reqID,
			OperationID:   &opID,
			Result:        string(api.AuditEventResultSucceeded),
			SchemaVersion: 1,
			Before:        beforeJSON,
			After:         afterJSON,
		}); err != nil {
			return fmt.Errorf("append audit for policy.activate: %w", err)
		}

		// Enqueue outbox event to trigger desired-state recompilation.
		outboxPayload, err := canonjson.Marshal(map[string]any{
			"definition_id":    in.DefinitionID.String(),
			"type":             defType,
			"code":             defCode,
			"revision_id":      in.RevisionID.String(),
			"revision":         revRec.Revision.String(),
			"generation":       newGen.String(),
			"resource_version": newVer.String(),
			"operation_id":     opID.String(),
			"effective_at":     effectiveAt.Format(time.RFC3339Nano),
		})
		if err != nil {
			return err
		}
		if err := db.EnqueueOutbox(ctx, tx, db.OutboxEvent{
			AggregateID:      in.DefinitionID,
			AggregateType:    "policy_definition",
			AggregateVersion: newGen.String(),
			EventType:        "policy.activated",
			SchemaVersion:    1,
			Payload:          outboxPayload,
		}); err != nil {
			return fmt.Errorf("enqueue policy.activated outbox event: %w", err)
		}

		if managedIdem && in.IdempotencyKey != nil {
			if err := db.CompleteIdempotent(ctx, tx, idemRecord.ID, db.IdempotencyResult{
				ResourceType:   "policy_definition",
				ResourceID:     &in.DefinitionID,
				OperationID:    &opID,
				ResponseStatus: 202,
			}); err != nil {
				return err
			}
		}

		out = &ActivateResult{
			DefinitionID:    in.DefinitionID,
			RevisionID:      in.RevisionID,
			Generation:      newGen,
			ResourceVersion: newVer,
			NoOp:            false,
			Operation: api.Operation{
				OperationId: openapi_types.UUID(opID),
				Kind:        api.OperationKind("policy.activate"),
				Status:      api.OperationStatusPending,
				RequestId:   openapi_types.UUID(reqID),
				Resource: api.ResourceRef{
					Type: api.ResourceType("policy_definition"),
					Id:   openapi_types.UUID(in.DefinitionID),
				},
				DesiredGeneration: presence.NewNullableWithValue(newGen),
				ResourceVersion:   presence.NewNullableWithValue(newVer),
				CreatedAt:         opCreatedAt.UTC(),
				UpdatedAt:         opUpdatedAt.UTC(),
				CompletedAt:       presence.NewNullNullable[time.Time](),
				Error: presence.NewNullNullable[struct {
					Code    api.ErrorCode `json:"code"`
					Message string        `json:"message"`
				}](),
				ObservedGeneration: presence.NewNullNullable[decstr.U64](),
				Steps: []api.OperationStep{
					{
						Name:       "desired_state_compile",
						Required:   true,
						Status:     api.OperationStepStatusPending,
						Message:    presence.NewNullNullable[string](),
						ObservedAt: presence.NewNullNullable[time.Time](),
						ErrorCode:  presence.NewNullNullable[api.ErrorCode](),
					},
				},
			},
			CompiledBundle: compiledBundle,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) compileAndStoreDNSBundleTx(ctx context.Context, tx pgx.Tx, profileRevID uuid.UUID, bundleRev uint64, p api.DnsProfilePayload) (*dnspolicy.CompiledBundle, error) {
	prof, registry, err := s.buildAndValidateDNSProfileTx(ctx, tx, bundleRev, p)
	if err != nil {
		return nil, err
	}

	now := s.now()
	limits := dnspolicy.DefaultDraftLimits()
	bundle, err := dnspolicy.CompileBundle(*prof, dnspolicy.CompileOptions{
		ValidFrom:         now,
		ValidUntil:        now.Add(defaultBundleTTL),
		UpstreamDNS:       s.upstreamDNS,
		BlockedTTLSeconds: uint32(p.BlockedTtlSeconds),
		Registry:          registry,
		Limits:            &limits,
	})
	if err != nil {
		return nil, newValidationError("/payload", ReasonDNSInvalidProfile, fmt.Sprintf("compile DNS bundle: %v", err))
	}

	bundleJSON, err := canonjson.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("marshal compiled DNS bundle: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO dns_compiled_bundles (
			profile_revision_id, profile_code, bundle_revision,
			bundle_sha256, valid_from, valid_until, bundle
		) VALUES ($1, $2, $3::numeric, $4, $5, $6, $7::jsonb)
		ON CONFLICT (profile_revision_id) DO NOTHING`,
		profileRevID,
		p.ProfileCode,
		bundle.Revision,
		bundle.SHA256,
		bundle.ValidFrom,
		bundle.ValidUntil,
		bundleJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("store compiled DNS bundle: %w", err)
	}
	return bundle, nil
}

// GetCompiledDNSBundle retrieves the compiled DNS bundle stored when a
// dns_profile revision was activated.
func (s *Store) GetCompiledDNSBundle(ctx context.Context, profileRevID uuid.UUID) (*dnspolicy.CompiledBundle, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx,
		`SELECT bundle FROM dns_compiled_bundles WHERE profile_revision_id = $1`,
		profileRevID,
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierr.Newf(apierr.ResourceNotFound, "compiled DNS bundle for revision %s not found", profileRevID)
		}
		return nil, err
	}
	var b dnspolicy.CompiledBundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// GetRevision fetches a single immutable policy revision by ID.
func (s *Store) GetRevision(ctx context.Context, revisionID uuid.UUID) (*RevisionRecord, error) {
	var rec *RevisionRecord
	err := s.runInTx(ctx, db.TxOptions{TxOptions: pgx.TxOptions{AccessMode: pgx.ReadOnly}}, func(tx pgx.Tx, _ bool) error {
		var e error
		rec, e = getRevisionTx(ctx, tx, revisionID)
		return e
	})
	return rec, err
}

func getRevisionTx(ctx context.Context, tx pgx.Tx, revisionID uuid.UUID) (*RevisionRecord, error) {
	var rec RevisionRecord
	var defType, revStr, createdByStr string
	var hashSlice []byte
	err := tx.QueryRow(ctx,
		`SELECT r.id, r.definition_id, d.code, r.type, r.revision::text, r.schema_version,
		        r.payload, r.payload_hash, r.signing_key_ref, r.created_by, r.reason, r.created_at
		 FROM policy_revisions r
		 JOIN policy_definitions d ON d.id = r.definition_id
		 WHERE r.id = $1`,
		revisionID,
	).Scan(
		&rec.RevisionID,
		&rec.DefinitionID,
		&rec.Code,
		&defType,
		&revStr,
		&rec.SchemaVersion,
		&rec.Payload,
		&hashSlice,
		&rec.SigningKeyRef,
		&createdByStr,
		&rec.Reason,
		&rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierr.Newf(apierr.ResourceNotFound, "policy revision %s not found", revisionID)
		}
		return nil, err
	}
	rec.Type = api.PolicyType(defType)
	rec.Revision, _ = decstr.Parse(revStr)
	copy(rec.PayloadHash[:], hashSlice)
	rec.PayloadSHA256 = hex.EncodeToString(hashSlice)
	if uid, err := uuid.Parse(createdByStr); err == nil {
		rec.CreatedBy = uid
	}
	rec.CreatedAt = rec.CreatedAt.UTC()
	return &rec, nil
}

// GetDefinition fetches a single PolicyDefinition with its active_revision,
// latest_revision, and revisions summary list (newest first).
func (s *Store) GetDefinition(ctx context.Context, definitionID uuid.UUID) (*api.PolicyDefinition, error) {
	var out *api.PolicyDefinition
	err := s.runInTx(ctx, db.TxOptions{TxOptions: pgx.TxOptions{AccessMode: pgx.ReadOnly}}, func(tx pgx.Tx, _ bool) error {
		def, err := getDefinitionTx(ctx, tx, definitionID)
		if err != nil {
			return err
		}
		out = &def
		return nil
	})
	return out, err
}

// ListDefinitionsFilter specifies filtering and cursor pagination for GET /admin/policies (§14.2).
type ListDefinitionsFilter struct {
	Type   *api.PolicyType
	Cursor string
	Limit  int
}

// ListDefinitions implements GET /admin/policies (§14.2).
func (s *Store) ListDefinitions(ctx context.Context, filter ListDefinitionsFilter) (*api.PolicyDefinitionList, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var cursorID *uuid.UUID
	if strings.TrimSpace(filter.Cursor) != "" {
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(filter.Cursor))
		if err != nil {
			return nil, apierr.New(apierr.MalformedRequest, "invalid pagination cursor")
		}
		id, err := uuid.Parse(string(raw))
		if err != nil {
			return nil, apierr.New(apierr.MalformedRequest, "invalid pagination cursor UUID")
		}
		cursorID = &id
	}

	var out api.PolicyDefinitionList
	err := s.runInTx(ctx, db.TxOptions{TxOptions: pgx.TxOptions{AccessMode: pgx.ReadOnly}}, func(tx pgx.Tx, _ bool) error {
		query := `SELECT id FROM policy_definitions WHERE 1=1`
		var args []any
		if filter.Type != nil {
			args = append(args, string(*filter.Type))
			query += fmt.Sprintf(` AND type = $%d`, len(args))
		}
		if cursorID != nil {
			args = append(args, *cursorID)
			query += fmt.Sprintf(` AND id > $%d`, len(args))
		}
		args = append(args, limit+1)
		query += fmt.Sprintf(` ORDER BY id ASC LIMIT $%d`, len(args))

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()

		var nextCursor *string
		if len(ids) > limit {
			lastID := ids[limit-1]
			ids = ids[:limit]
			c := base64.RawURLEncoding.EncodeToString([]byte(lastID.String()))
			nextCursor = &c
		}

		items := make([]api.PolicyDefinition, 0, len(ids))
		for _, id := range ids {
			def, err := getDefinitionTx(ctx, tx, id)
			if err != nil {
				return err
			}
			items = append(items, def)
		}
		out = api.PolicyDefinitionList{
			Items:      items,
			NextCursor: presence.FromPointer(nextCursor),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func getDefinitionTx(ctx context.Context, tx pgx.Tx, definitionID uuid.UUID) (api.PolicyDefinition, error) {
	var def api.PolicyDefinition
	var defID uuid.UUID
	var defType, defCode, genStr, verStr string
	var activeRevID *uuid.UUID
	var createdAt, updatedAt time.Time

	err := tx.QueryRow(ctx,
		`SELECT id, type, code, active_revision_id, generation::text, resource_version::text, created_at, updated_at
		 FROM policy_definitions
		 WHERE id = $1`,
		definitionID,
	).Scan(&defID, &defType, &defCode, &activeRevID, &genStr, &verStr, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return def, apierr.Newf(apierr.ResourceNotFound, "policy definition %s not found", definitionID)
		}
		return def, err
	}

	gen, _ := decstr.Parse(genStr)
	ver, _ := decstr.Parse(verStr)

	rows, err := tx.Query(ctx,
		`SELECT id FROM policy_revisions WHERE definition_id = $1 ORDER BY revision DESC`,
		definitionID,
	)
	if err != nil {
		return def, err
	}
	var revIDs []uuid.UUID
	for rows.Next() {
		var rid uuid.UUID
		if err := rows.Scan(&rid); err != nil {
			rows.Close()
			return def, err
		}
		revIDs = append(revIDs, rid)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return def, err
	}
	rows.Close()

	summaries := make([]api.PolicyRevisionSummary, 0, len(revIDs))
	var latestAPIRec *api.PolicyRevision
	var activeAPIRec *api.PolicyRevision

	for i, rid := range revIDs {
		rec, err := getRevisionTx(ctx, tx, rid)
		if err != nil {
			return def, err
		}
		summaries = append(summaries, rec.Summary())
		if i == 0 {
			apiRev, err := rec.ToAPI()
			if err != nil {
				return def, err
			}
			latestAPIRec = &apiRev
		}
		if activeRevID != nil && rid == *activeRevID {
			apiRev, err := rec.ToAPI()
			if err != nil {
				return def, err
			}
			activeAPIRec = &apiRev
		}
	}

	var apiActiveRevID *openapi_types.UUID
	if activeRevID != nil {
		u := openapi_types.UUID(*activeRevID)
		apiActiveRevID = &u
	}

	def = api.PolicyDefinition{
		DefinitionId:     openapi_types.UUID(defID),
		Type:             api.PolicyType(defType),
		Code:             defCode,
		ActiveRevisionId: presence.FromPointer(apiActiveRevID),
		ActiveRevision:   activeAPIRec,
		LatestRevision:   latestAPIRec,
		Revisions:        summaries,
		Generation:       gen,
		ResourceVersion:  ver,
		CreatedAt:        createdAt.UTC(),
		UpdatedAt:        updatedAt.UTC(),
	}
	return def, nil
}

// GetOperation fetches an operation by ID within the current transaction (or a read-only transaction).
func (s *Store) GetOperation(ctx context.Context, opID uuid.UUID) (api.Operation, error) {
	var op api.Operation
	err := s.runInTx(ctx, db.TxOptions{TxOptions: pgx.TxOptions{AccessMode: pgx.ReadOnly}}, func(tx pgx.Tx, _ bool) error {
		var e error
		op, e = getOperationTx(ctx, tx, opID)
		return e
	})
	return op, err
}

func getOperationTx(ctx context.Context, tx pgx.Tx, opID uuid.UUID) (api.Operation, error) {
	var op api.Operation
	var kind, status, resType string
	var resID, reqID *uuid.UUID
	var resVerStr, desiredGenStr, observedGenStr *string
	var createdAt, updatedAt time.Time
	var completedAt *time.Time
	err := tx.QueryRow(ctx,
		`SELECT kind, status, resource_type, resource_id, request_id, resource_version::text, desired_generation::text, observed_generation::text, created_at, updated_at, completed_at
		 FROM operations WHERE id = $1`,
		opID,
	).Scan(&kind, &status, &resType, &resID, &reqID, &resVerStr, &desiredGenStr, &observedGenStr, &createdAt, &updatedAt, &completedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return op, apierr.New(apierr.ResourceNotFound, "Operation not found.")
		}
		return op, err
	}
	var refID openapi_types.UUID
	if resID != nil {
		refID = openapi_types.UUID(*resID)
	}
	var apiReqID openapi_types.UUID
	if reqID != nil {
		apiReqID = openapi_types.UUID(*reqID)
	}
	var resVer, desiredGen, observedGen *decstr.U64
	if resVerStr != nil {
		if v, err := decstr.Parse(*resVerStr); err == nil {
			resVer = &v
		}
	}
	if desiredGenStr != nil {
		if g, err := decstr.Parse(*desiredGenStr); err == nil {
			desiredGen = &g
		}
	}
	if observedGenStr != nil {
		if g, err := decstr.Parse(*observedGenStr); err == nil {
			observedGen = &g
		}
	}
	stepStatus := api.OperationStepStatusPending
	if status == string(api.OperationStatusSucceeded) {
		stepStatus = api.OperationStepStatusSucceeded
	} else if status == string(api.OperationStatusFailed) {
		stepStatus = api.OperationStepStatusFailed
	}
	return api.Operation{
		OperationId: openapi_types.UUID(opID),
		Kind:        api.OperationKind(kind),
		Status:      api.OperationStatus(status),
		RequestId:   apiReqID,
		Resource: api.ResourceRef{
			Type: api.ResourceType(resType),
			Id:   refID,
		},
		DesiredGeneration: presence.FromPointer(desiredGen),
		CreatedAt:         createdAt.UTC(),
		UpdatedAt:         updatedAt.UTC(),
		CompletedAt:       presence.FromPointer(completedAt),
		Error: presence.NewNullNullable[struct {
			Code    api.ErrorCode `json:"code"`
			Message string        `json:"message"`
		}](),
		ObservedGeneration: presence.FromPointer(observedGen),
		ResourceVersion:    presence.FromPointer(resVer),
		Steps: []api.OperationStep{
			{
				Name:       "desired_state_compile",
				Required:   true,
				Status:     stepStatus,
				Message:    presence.NewNullNullable[string](),
				ObservedAt: presence.FromPointer(completedAt),
				ErrorCode:  presence.NewNullNullable[api.ErrorCode](),
			},
		},
	}, nil
}

func getOrCreateNoOpActivateOperationTx(ctx context.Context, tx pgx.Tx, in ActivateRevisionInput, reqID uuid.UUID) (api.Operation, error) {
	var existingID uuid.UUID
	err := tx.QueryRow(ctx,
		`SELECT id FROM operations
		 WHERE resource_type = 'policy_definition' AND resource_id = $1 AND kind = 'policy.activate'
		 ORDER BY created_at DESC LIMIT 1`,
		in.DefinitionID,
	).Scan(&existingID)
	if err == nil {
		return getOperationTx(ctx, tx, existingID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return api.Operation{}, err
	}

	opID, err := uuid.NewV7()
	if err != nil {
		opID = uuid.New()
	}
	var createdAt time.Time
	if err := tx.QueryRow(ctx,
		`INSERT INTO operations (id, kind, status, principal, resource_type, resource_id, request_id, steps)
		 VALUES ($1, 'policy.activate', 'pending', $2, 'policy_definition', $3, $4, '{"desired_state_compile":{"required":true,"status":"pending"}}'::jsonb)
		 RETURNING created_at`,
		opID, principalScope(in.Principal, in.ActorID), in.DefinitionID, reqID,
	).Scan(&createdAt); err != nil {
		return api.Operation{}, err
	}
	return api.Operation{
		OperationId: openapi_types.UUID(opID),
		Kind:        api.OperationKind("policy.activate"),
		Status:      api.OperationStatusPending,
		RequestId:   openapi_types.UUID(reqID),
		Resource: api.ResourceRef{
			Type: api.ResourceType("policy_definition"),
			Id:   openapi_types.UUID(in.DefinitionID),
		},
		CreatedAt:         createdAt.UTC(),
		UpdatedAt:         createdAt.UTC(),
		CompletedAt:       presence.NewNullNullable[time.Time](),
		DesiredGeneration: presence.NewNullNullable[decstr.U64](),
		Error: presence.NewNullNullable[struct {
			Code    api.ErrorCode `json:"code"`
			Message string        `json:"message"`
		}](),
		ObservedGeneration: presence.NewNullNullable[decstr.U64](),
		ResourceVersion:    presence.NewNullNullable[decstr.U64](),
		Steps: []api.OperationStep{
			{
				Name:       "desired_state_compile",
				Required:   true,
				Status:     api.OperationStepStatusPending,
				Message:    presence.NewNullNullable[string](),
				ObservedAt: presence.NewNullNullable[time.Time](),
				ErrorCode:  presence.NewNullNullable[api.ErrorCode](),
			},
		},
	}, nil
}

func buildCreateIdempotencyScope(in CreateRevisionInput, payloadHash [32]byte) (db.IdempotencyScope, error) {
	principal := strings.TrimSpace(in.Principal)
	if principal == "" {
		principal = in.ActorID.String()
	}
	keyHash := sha256.Sum256([]byte(in.IdempotencyKey.String()))
	defStr := ""
	if in.DefinitionID != nil {
		defStr = in.DefinitionID.String()
	}
	codeStr := ""
	if in.Code != nil {
		codeStr = *in.Code
	}
	reqBytes, err := canonjson.Marshal(map[string]any{
		"definition_id":  defStr,
		"code":           codeStr,
		"type":           string(in.Type),
		"schema_version": in.SchemaVersion,
		"reason":         in.Reason,
		"payload_sha256": hex.EncodeToString(payloadHash[:]),
	})
	if err != nil {
		return db.IdempotencyScope{}, err
	}
	reqHash := sha256.Sum256(reqBytes)
	return db.IdempotencyScope{
		Principal:   principal,
		Method:      "POST",
		Route:       "/api/v1/admin/policies",
		KeyHash:     keyHash[:],
		RequestHash: reqHash[:],
	}, nil
}

func buildActivateIdempotencyScope(in ActivateRevisionInput) (db.IdempotencyScope, error) {
	principal := strings.TrimSpace(in.Principal)
	if principal == "" {
		principal = in.ActorID.String()
	}
	keyHash := sha256.Sum256([]byte(in.IdempotencyKey.String()))
	effStr := ""
	if in.EffectiveAt != nil {
		effStr = in.EffectiveAt.UTC().Format(time.RFC3339Nano)
	}
	reqBytes, err := canonjson.Marshal(map[string]any{
		"definition_id": in.DefinitionID.String(),
		"revision_id":   in.RevisionID.String(),
		"effective_at":  effStr,
		"reason":        in.Reason,
	})
	if err != nil {
		return db.IdempotencyScope{}, err
	}
	reqHash := sha256.Sum256(reqBytes)
	return db.IdempotencyScope{
		Principal:   principal,
		Method:      "POST",
		Route:       fmt.Sprintf("/api/v1/admin/policies/%s/activate", in.DefinitionID),
		KeyHash:     keyHash[:],
		RequestHash: reqHash[:],
	}, nil
}

func cleanETag(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "W/") {
		return s
	}
	s = strings.Trim(s, `"`)
	return strings.TrimSpace(s)
}
