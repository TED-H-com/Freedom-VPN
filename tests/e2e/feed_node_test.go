//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (r *runner) threatFeedNode(ctx context.Context, admin *session) {
	rec := Record{ID: "threat-feed-node", Title: "Issued resolver document reaches the node resolver", Spec: []string{"§12.5", "§19 AT22", "D-061"}, Status: StatusBlocked}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	mode, _ := r.lab.docker(r.lab.node, "stat", "-c", "%u:%a", "/etc/vpn-node/resolver-credentials.json")
	note("credential_owner_mode", strings.TrimSpace(string(mode)))
	note("dns_config", "agent-run only; lab admin 127.0.0.1:8080 and DNS 127.0.0.1:53; AdGuard shares node network namespace")
	conn, err := r.lab.db(ctx)
	if err != nil {
		note("database", "unavailable")
		return
	}
	defer conn.Close(ctx)
	var doc uuid.UUID
	var hash, generation string
	var size int
	err = conn.QueryRow(ctx, `SELECT id,document_sha256,policy_generation::text,octet_length(canonical_bytes) FROM dns_resolver_documents WHERE profile_revision_id=$1 ORDER BY policy_generation DESC LIMIT 1`, r.profileID).Scan(&doc, &hash, &generation, &size)
	if err != nil {
		note("document", "unavailable")
		return
	}
	note("document_id", doc.String())
	note("content_sha256", hash)
	note("generation", generation)
	note("size_bytes", itoa(size))
	wait, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	var reference json.RawMessage
	issued := waitUntil(wait, time.Second, func() bool {
		var envelope struct {
			Signed struct {
				DNS struct {
					Reference json.RawMessage `json:"resolver_document"`
				} `json:"dns"`
			}
		}
		if json.Unmarshal(r.desiredRaw(), &envelope) != nil {
			return false
		}
		reference = envelope.Signed.DNS.Reference
		return len(reference) > 0 && string(reference) != "null"
	})
	note("desired_resolver_reference_issued", boolText(issued))
	body, _ := r.lab.docker(r.lab.node, "wget", "--server-response", "--timeout=5", "--tries=1", "-qO-", "--content-on-error", "--ca-certificate=/etc/vpn-node/ca.pem", "--certificate=/var/lib/vpn-node/identity/cert.pem", "--private-key=/var/lib/vpn-node/identity/key.pem", r.lab.agent+"/api/v1/agent/resolver-documents/"+doc.String())
	// Keep the exact HTTP status/error envelope; document contents contain rules.
	for _, line := range strings.Split(string(body), "\n") {
		if strings.Contains(line, "HTTP/") || strings.Contains(line, `"error"`) {
			note("resolver_document_response", strings.TrimSpace(line))
		}
	}
	detail, e := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/nodes/"+r.nodeID.String(), nil, nil)
	note("admin_node", httpDetail(detail, e))
	var node struct {
		Evidence json.RawMessage `json:"dns_policy_evidence"`
	}
	_ = json.Unmarshal(detail.Body, &node)
	note("dns_policy_evidence", emptyDash(string(node.Evidence)))
	probe, _ := r.lab.docker(r.lab.node, "fvpn-e2e-nodeconfig", "dns-probe")
	note("resolver_readback", strings.TrimSpace(string(probe)))
	if !issued {
		note("owner", "w3-wire-nodes / internal/desiredstate: compiler emits resolver addresses but no resolver_document reference; D-056 issuance is missing")
		note("tampered_document_case", "blocked by the same missing issued reference; no last-good node apply exists to preserve. No signed state or projector row is fabricated.")
		return
	}
	note("owner", "w2-agent-core and w3-dns-publish: inspect apply evidence; tampered-document last-good proof still required before pass")
}
