package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	statusActive     = "active"
	statusWithdrawn  = "withdrawn"
	statusSuperseded = "superseded"
)

// Config is the resolver environment mixed into every document (§12.5, D-028).
// Upstreams come from deployment configuration. An empty list is rejected.
// Two activations compiled with the same Config and the same rules match.
type Config struct {
	Upstreams    []string
	BootstrapDNS []string
}

// Publication is the stored document plus the generation desired state references.
type Publication struct {
	DefinitionID      uuid.UUID
	ProfileRevisionID uuid.UUID
	ProfileCode       string
	PolicyGeneration  decstr.U64
	DocumentHash      string
	Canonical         []byte
	Document          dnspublish.Document
	Status            string
}

// Publisher compiles dns_profile activations and stores the canonical document.
type Publisher struct {
	pool *pgxpool.Pool
	cfg  Config
}

// NewPublisher builds a publisher. An empty upstream list is an error.
// pool may be nil only for tests that reject the event before a query.
func NewPublisher(pool *pgxpool.Pool, cfg Config) (*Publisher, error) {
	if len(cfg.Upstreams) == 0 {
		return nil, errors.New("dns upstreams are required")
	}
	cfg.Upstreams = append([]string(nil), cfg.Upstreams...)
	if cfg.BootstrapDNS != nil {
		cfg.BootstrapDNS = append([]string(nil), cfg.BootstrapDNS...)
	}
	return &Publisher{pool: pool, cfg: cfg}, nil
}

// NewHandler is the worker callback for dns.publish (§14.5).
func NewHandler(pool *pgxpool.Pool, cfg Config) (func(context.Context, db.ClaimedEvent) (string, error), error) {
	p, err := NewPublisher(pool, cfg)
	if err != nil {
		return nil, err
	}
	return p.Handle, nil
}

// Handle compiles or withdraws the publication described by event.
func (p *Publisher) Handle(ctx context.Context, event db.ClaimedEvent) (string, error) {
	if event.EventType != dnspublish.EventDNSPublish && event.EventType != "policy.activated" {
		return "", fmt.Errorf("unexpected event %q", event.EventType)
	}
	var body activation
	if err := json.Unmarshal(event.Payload, &body); err != nil {
		return "", fmt.Errorf("decode dns publish event: %w", err)
	}
	if body.Type != "" && body.Type != string(api.PolicyTypeDnsProfile) {
		return "dns-publish:ignored:" + body.Type, nil
	}
	if p == nil || p.pool == nil {
		return "", errors.New("dns publisher is not configured")
	}
	defID, err := uuid.Parse(body.DefinitionID)
	if err != nil {
		return "", fmt.Errorf("definition_id: %w", err)
	}
	revID, err := uuid.Parse(body.RevisionID)
	if err != nil {
		return "", fmt.Errorf("revision_id: %w", err)
	}
	gen, err := decstr.Parse(body.Generation)
	if err != nil {
		return "", fmt.Errorf("generation: %w", err)
	}
	if body.Action == statusWithdrawn || body.Action == "withdraw" {
		if err := p.withdraw(ctx, defID, revID, gen, body.OperationID); err != nil {
			return "", err
		}
		return "dns-withdrawn:" + body.Generation, nil
	}
	pub, err := p.activate(ctx, defID, revID, gen, body.OperationID)
	if err != nil {
		return "", err
	}
	return "dns-document:" + pub.DocumentHash, nil
}

type activation struct {
	DefinitionID string `json:"definition_id"`
	Type         string `json:"type"`
	RevisionID   string `json:"revision_id"`
	Generation   string `json:"generation"`
	OperationID  string `json:"operation_id"`
	Action       string `json:"action"`
}

func (p *Publisher) activate(ctx context.Context, defID, revID uuid.UUID, gen decstr.U64, operationID string) (Publication, error) {
	var pub Publication
	err := db.InTx(ctx, p.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		in, code, err := loadProfile(ctx, tx, defID, revID)
		if err != nil {
			return err
		}
		in.Upstreams = p.cfg.Upstreams
		in.BootstrapDNS = p.cfg.BootstrapDNS
		compiled, err := dnspublish.Compile(in)
		if err != nil {
			return err
		}
		if err := insertDocument(ctx, tx, defID, revID, code, gen, compiled); err != nil {
			return err
		}
		advanced, err := advanceHead(ctx, tx, defID, revID, gen, compiled.Hash, statusActive)
		if err != nil {
			return err
		}
		if err := writeAudit(ctx, tx, operationID, defID, revID, "dns.publish", api.AuditEventResultSucceeded, gen, compiled.Hash, statusActive, advanced); err != nil {
			return err
		}
		if err := noteOperation(ctx, tx, operationID, gen, compiled.Hash, statusActive); err != nil {
			return err
		}
		pub = Publication{
			DefinitionID: defID, ProfileRevisionID: revID, ProfileCode: code,
			PolicyGeneration: gen, DocumentHash: compiled.Hash, Canonical: compiled.Bytes,
			Document: compiled.Document, Status: statusActive,
		}
		return nil
	})
	return pub, err
}

func (p *Publisher) withdraw(ctx context.Context, defID, revID uuid.UUID, gen decstr.U64, operationID string) error {
	return db.InTx(ctx, p.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var hash string
		err := tx.QueryRow(ctx, `
			SELECT document_sha256 FROM dns_resolver_documents
			WHERE profile_revision_id = $1 AND definition_id = $2 AND policy_generation = $3::numeric`,
			revID, defID, gen.String(),
		).Scan(&hash)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("no resolver document for revision %s", revID)
			}
			return err
		}
		advanced, err := advanceHead(ctx, tx, defID, revID, gen, hash, statusWithdrawn)
		if err != nil {
			return err
		}
		if err := writeAudit(ctx, tx, operationID, defID, revID, "dns.withdraw", api.AuditEventResultSucceeded, gen, hash, statusWithdrawn, advanced); err != nil {
			return err
		}
		return noteOperation(ctx, tx, operationID, gen, hash, statusWithdrawn)
	})
}

// GetByRevision returns the stored canonical document for a policy revision.
// Desired state policy_refs carry this revision id (§12.5).
func (p *Publisher) GetByRevision(ctx context.Context, revID uuid.UUID) (Publication, error) {
	return p.getByRevision(ctx, revID)
}

func (p *Publisher) getByRevision(ctx context.Context, revID uuid.UUID) (Publication, error) {
	var pub Publication
	var genStr string
	err := p.pool.QueryRow(ctx, `
		SELECT d.definition_id, d.profile_revision_id, d.profile_code, d.policy_generation::text,
		       d.document_sha256, d.canonical_bytes,
		       CASE
		         WHEN h.profile_revision_id = d.profile_revision_id THEN h.status
		         WHEN h.definition_id IS NULL THEN 'active'
		         ELSE 'superseded'
		       END
		FROM dns_resolver_documents d
		LEFT JOIN dns_publication_heads h ON h.definition_id = d.definition_id
		WHERE d.profile_revision_id = $1`, revID).Scan(
		&pub.DefinitionID, &pub.ProfileRevisionID, &pub.ProfileCode, &genStr,
		&pub.DocumentHash, &pub.Canonical, &pub.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Publication{}, fmt.Errorf("resolver document %s not found", revID)
		}
		return Publication{}, err
	}
	switch pub.Status {
	case statusActive, statusWithdrawn, statusSuperseded:
	default:
		return Publication{}, fmt.Errorf("resolver document %s has status %s", revID, pub.Status)
	}
	gen, err := decstr.Parse(genStr)
	if err != nil {
		return Publication{}, err
	}
	pub.PolicyGeneration = gen
	if err := json.Unmarshal(pub.Canonical, &pub.Document); err != nil {
		return Publication{}, err
	}
	sum, raw, err := dnspublish.Hash(pub.Document)
	if err != nil {
		return Publication{}, err
	}
	if sum != pub.DocumentHash || string(raw) != string(pub.Canonical) {
		return Publication{}, errors.New("stored resolver document does not match its hash")
	}
	return pub, nil
}

// Head returns the current publication for a definition.
func (p *Publisher) Head(ctx context.Context, defID uuid.UUID) (Publication, error) {
	var revID uuid.UUID
	err := p.pool.QueryRow(ctx, `SELECT profile_revision_id FROM dns_publication_heads WHERE definition_id = $1`, defID).Scan(&revID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Publication{}, fmt.Errorf("dns publication %s not found", defID)
		}
		return Publication{}, err
	}
	return p.getByRevision(ctx, revID)
}

func loadProfile(ctx context.Context, tx pgx.Tx, defID, revID uuid.UUID) (dnspublish.Input, string, error) {
	var (
		gotDef  uuid.UUID
		typ     string
		payload []byte
	)
	err := tx.QueryRow(ctx, `
		SELECT definition_id, type, payload FROM policy_revisions WHERE id = $1`, revID,
	).Scan(&gotDef, &typ, &payload)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dnspublish.Input{}, "", fmt.Errorf("policy revision %s not found", revID)
		}
		return dnspublish.Input{}, "", err
	}
	if gotDef != defID {
		return dnspublish.Input{}, "", fmt.Errorf("revision %s does not belong to definition %s", revID, defID)
	}
	if typ != string(api.PolicyTypeDnsProfile) {
		return dnspublish.Input{}, "", fmt.Errorf("revision %s has type %s", revID, typ)
	}
	var profile api.DnsProfilePayload
	if err := json.Unmarshal(payload, &profile); err != nil {
		return dnspublish.Input{}, "", err
	}
	var extra struct {
		SafeSearch *bool `json:"safe_search"`
	}
	_ = json.Unmarshal(payload, &extra)
	safe := false
	if extra.SafeSearch != nil {
		safe = *extra.SafeSearch
	}
	rules, err := loadRules(ctx, tx, profile)
	if err != nil {
		return dnspublish.Input{}, "", err
	}
	feeds, err := loadFeeds(ctx, tx, profile)
	if err != nil {
		return dnspublish.Input{}, "", err
	}
	return dnspublish.Input{
		ProfileCode:       profile.ProfileCode,
		BlockResponse:     string(profile.BlockResponse),
		BlockedTTLSeconds: profile.BlockedTtlSeconds,
		SafeSearch:        safe,
		Rules:             rules,
		Feeds:             feeds,
	}, profile.ProfileCode, nil
}

func loadRules(ctx context.Context, tx pgx.Tx, profile api.DnsProfilePayload) ([]dnspublish.Rule, error) {
	var rules []dnspublish.Rule
	for _, id := range profile.DomainRuleSetRevisionIds {
		revID := uuid.UUID(id)
		var typ string
		var payload []byte
		err := tx.QueryRow(ctx, `SELECT type, payload FROM policy_revisions WHERE id = $1`, revID).Scan(&typ, &payload)
		if err != nil {
			return nil, fmt.Errorf("domain rule set %s: %w", revID, err)
		}
		if typ != string(api.PolicyTypeDomainRuleSet) {
			return nil, fmt.Errorf("revision %s is %s, not a domain rule set", revID, typ)
		}
		var set api.DomainRuleSetPayload
		if err := json.Unmarshal(payload, &set); err != nil {
			return nil, err
		}
		for _, rule := range set.Rules {
			if scoped, err := rule.ProfileId.Get(); err == nil && scoped != "" && scoped != profile.ProfileCode {
				continue
			}
			action := string(dnspolicy.ActionBlock)
			if rule.Action == api.DomainRuleActionAllow {
				action = string(dnspolicy.ActionAllow)
			}
			rules = append(rules, dnspublish.Rule{
				RuleID:            uuid.UUID(rule.RuleId).String(),
				Domain:            rule.Domain,
				IncludeSubdomains: rule.IncludeSubdomains,
				Action:            action,
				Reason:            rule.Reason,
			})
		}
	}
	return rules, nil
}

func loadFeeds(ctx context.Context, tx pgx.Tx, profile api.DnsProfilePayload) ([]dnspublish.Feed, error) {
	var feeds []dnspublish.Feed
	for _, id := range profile.ThreatFeedRevisionIds {
		revID := uuid.UUID(id)
		var typ string
		var payload []byte
		err := tx.QueryRow(ctx, `SELECT type, payload FROM policy_revisions WHERE id = $1`, revID).Scan(&typ, &payload)
		if err != nil {
			return nil, fmt.Errorf("threat feed %s: %w", revID, err)
		}
		if typ != string(api.PolicyTypeThreatFeed) {
			return nil, fmt.Errorf("revision %s is %s, not a threat feed", revID, typ)
		}
		var feed api.ThreatFeedPayload
		if err := json.Unmarshal(payload, &feed); err != nil {
			return nil, err
		}
		cats := make([]string, len(feed.Categories))
		for i, c := range feed.Categories {
			cats[i] = string(c)
		}
		feeds = append(feeds, dnspublish.Feed{
			FeedCode:      feed.FeedCode,
			ContentSHA256: feed.ContentSha256,
			Format:        string(feed.Format),
			Categories:    cats,
		})
	}
	return feeds, nil
}

func insertDocument(ctx context.Context, tx pgx.Tx, defID, revID uuid.UUID, code string, gen decstr.U64, compiled dnspublish.Compiled) error {
	tag, err := tx.Exec(ctx, `
		INSERT INTO dns_resolver_documents (
			profile_revision_id, definition_id, profile_code, policy_generation,
			document_sha256, document, canonical_bytes
		) VALUES ($1, $2, $3, $4::numeric, $5, $6::jsonb, $7)
		ON CONFLICT (profile_revision_id) DO NOTHING`,
		revID, defID, code, gen.String(), compiled.Hash, string(compiled.Bytes), compiled.Bytes,
	)
	if err != nil {
		return fmt.Errorf("store resolver document: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var existing string
	var existingGen string
	if err := tx.QueryRow(ctx, `
		SELECT document_sha256, policy_generation::text FROM dns_resolver_documents WHERE profile_revision_id = $1`, revID,
	).Scan(&existing, &existingGen); err != nil {
		return err
	}
	if existing != compiled.Hash || existingGen != gen.String() {
		return fmt.Errorf("resolver document for %s is already stored with a different hash or generation", revID)
	}
	return nil
}

func advanceHead(ctx context.Context, tx pgx.Tx, defID, revID uuid.UUID, gen decstr.U64, hash, status string) (bool, error) {
	var curGen string
	var curStatus string
	var curRev uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT policy_generation::text, status, profile_revision_id
		FROM dns_publication_heads WHERE definition_id = $1`, defID,
	).Scan(&curGen, &curStatus, &curRev)
	if errors.Is(err, pgx.ErrNoRows) {
		if status != statusActive {
			return false, fmt.Errorf("cannot withdraw dns profile %s before it is published", defID)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO dns_publication_heads (
				definition_id, profile_revision_id, policy_generation, document_sha256, status
			) VALUES ($1, $2, $3::numeric, $4, $5)`,
			defID, revID, gen.String(), hash, status,
		)
		return err == nil, err
	}
	if err != nil {
		return false, err
	}
	cur, err := decstr.Parse(curGen)
	if err != nil {
		return false, err
	}
	if gen < cur {
		// A replay of an older activation must not replace a newer publication.
		return false, nil
	}
	if gen == cur && curStatus == status && curRev == revID {
		return false, nil
	}
	if gen == cur && status == statusActive && curStatus == statusWithdrawn {
		return false, fmt.Errorf("dns publication %s generation %s is withdrawn", defID, gen)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE dns_publication_heads
		SET profile_revision_id = $2,
		    policy_generation = $3::numeric,
		    document_sha256 = $4,
		    status = $5,
		    resource_version = resource_version + 1
		WHERE definition_id = $1`,
		defID, revID, gen.String(), hash, status,
	)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() != 1 {
		return false, fmt.Errorf("dns publication head %s was not updated", defID)
	}
	return true, nil
}

func writeAudit(ctx context.Context, tx pgx.Tx, operationID string, defID, revID uuid.UUID, action string, result api.AuditEventResult, gen decstr.U64, hash, status string, advanced bool) error {
	var op, requestID *uuid.UUID
	if operationID != "" {
		id, err := uuid.Parse(operationID)
		if err != nil {
			return fmt.Errorf("operation_id: %w", err)
		}
		op = &id
		var req *uuid.UUID
		err = tx.QueryRow(ctx, `SELECT request_id FROM operations WHERE id = $1`, id).Scan(&req)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if req != nil && *req != uuid.Nil {
			requestID = req
		}
	}
	head := "unchanged"
	if advanced {
		head = "advanced"
	}
	// The definition owns the publication head. The revision owns the document,
	// so it is a second subject rather than a bare id inside the definition row.
	after := map[string]any{
		"definition_id":       defID.String(),
		"profile_revision_id": revID.String(),
		"policy_generation":   gen.String(),
		"document_sha256":     hash,
		"status":              status,
		"head":                head,
	}
	fields := []string{"definition_id", "profile_revision_id", "policy_generation", "document_sha256", "status", "head"}
	for _, resource := range []api.ResourceRef{
		{Type: api.ResourceTypePolicyDefinition, Id: defID},
		{Type: api.ResourceTypePolicyRevision, Id: revID},
	} {
		event, err := audit.NewEvent(audit.Entry{
			Actor:       audit.SystemActor,
			Action:      action,
			Resource:    resource,
			Result:      result,
			OperationID: op,
			RequestID:   requestID,
			After:       after,
			Fields:      fields,
		})
		if err != nil {
			return err
		}
		if err := audit.AppendEvent(ctx, tx, event); err != nil {
			return err
		}
	}
	return nil
}

func noteOperation(ctx context.Context, tx pgx.Tx, operationID string, gen decstr.U64, hash, status string) error {
	if strings.TrimSpace(operationID) == "" {
		return nil
	}
	id, err := uuid.Parse(operationID)
	if err != nil {
		return err
	}
	evidence, err := json.Marshal(map[string]string{
		"policy_generation": gen.String(),
		"document_sha256":   hash,
		"status":            status,
		"observed_at":       time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return err
	}
	// Leave the operation pending: desired_state_compile is still the policies
	// step. Record the document so the center can tell desired from applied (§14.5).
	_, err = tx.Exec(ctx, `
		UPDATE operations
		SET observed_refs = jsonb_set(observed_refs, '{dns_document}', $2::jsonb, true),
		    updated_at = clock_timestamp()
		WHERE id = $1 AND status IN ('pending', 'running')`,
		id, string(evidence),
	)
	return err
}
