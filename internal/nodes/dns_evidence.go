package nodes

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func validateDNSPolicyEvidence(report api.ObservationReport) error {
	evidence, err := report.DnsPolicyEvidence.Get()
	if err != nil {
		return nil
	}
	seen := make(map[string]bool, len(evidence.Feeds))
	for _, feed := range evidence.Feeds {
		if seen[feed.FeedCode] {
			return apierr.Validation("Duplicate DNS feed evidence.", apierr.FieldError{Field: "/dns_policy_evidence/feeds", Reason: "feed_code values must be unique"})
		}
		seen[feed.FeedCode] = true
	}
	for _, check := range report.Checks {
		if check.Name == api.NodeCheckNameDnsPolicy && check.Status == api.HealthCheckStatusPassed && evidence.Outcome != api.DnsPolicyEvidenceOutcomeApplied {
			return apierr.Validation("DNS policy check contradicts reported apply evidence.", apierr.FieldError{Field: "/dns_policy_evidence/outcome", Reason: "only applied evidence can pass dns_policy"})
		}
	}
	return nil
}

// DNSPolicyEvidence reads the report selected by the current incarnation's
// observed state, never the latest desired policy or a late stale report (D-056).
func (s *Service) DNSPolicyEvidence(ctx context.Context, node uuid.UUID) (presence.Nullable[api.DnsPolicyEvidence], error) {
	return dnsPolicyEvidence(ctx, s.pool, node)
}

// GetNodeWithDNSEvidence reads the node view and its report from one snapshot.
func (s *Service) GetNodeWithDNSEvidence(ctx context.Context, node uuid.UUID) (detail api.NodeDetail, evidence presence.Nullable[api.DnsPolicyEvidence], etag string, err error) {
	err = db.InTx(ctx, s.pool, db.TxOptions{TxOptions: pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}}, func(tx pgx.Tx) error {
		var e error
		detail, etag, e = GetNodeTx(ctx, tx, node)
		if e != nil {
			return e
		}
		evidence, e = dnsPolicyEvidence(ctx, tx, node)
		return e
	})
	return
}

func dnsPolicyEvidence(ctx context.Context, q Querier, node uuid.UUID) (presence.Nullable[api.DnsPolicyEvidence], error) {
	out := presence.NewNullNullable[api.DnsPolicyEvidence]()
	var raw []byte
	err := q.QueryRow(ctx, `SELECT o.report_payload->'dns_policy_evidence'
		FROM nodes n JOIN node_observations o ON o.node_id=n.id AND o.incarnation_id=n.current_incarnation_id
		WHERE n.id=$1 AND o.observed_at=n.last_observed_at
		ORDER BY o.received_at DESC,o.id DESC LIMIT 1`, node).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && len(raw) == 0) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(raw, &out)
	return out, err
}
