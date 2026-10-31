//go:build e2e

package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// waitForUploadedTraffic observes the real agent's batch. Only AT15 takes over
// the sequence stream for deliberate protocol faults, after pausing the agent.
func (s *liveSuite) waitForUploadedTraffic(service string, nodeID, incID, deviceID uuid.UUID, after ...time.Time) api.UsageEvent {
	since := time.Unix(0, 0).UTC()
	if len(after) > 0 {
		since = after[0]
	}
	var event api.UsageEvent
	// Product defaults: 10s sampling plus 60s upload cadence, with scheduling margin.
	if !s.waitFor(90*time.Second, func() bool {
		var raw []byte
		err := s.pool.QueryRow(s.ctx, `
			SELECT p.payload FROM usage_events e
			JOIN usage_event_payloads p ON p.event_id=e.id
			JOIN usage_samples u ON u.event_id=e.id
			WHERE e.node_id=$1 AND e.incarnation_id=$2 AND u.device_id=$3
			  AND e.received_at>$4 AND u.upload_bytes>0 AND u.download_bytes>0
			ORDER BY e.event_seq DESC LIMIT 1
		`, nodeID, incID, deviceID, since).Scan(&raw)
		snap := s.inspectNodeUsageSQLite(service, deviceID)
		return err == nil && json.Unmarshal(raw, &event) == nil && snap.OutboxCount == 0 && snap.AckedThrough >= uint64(event.EventSeq)
	}) {
		s.t.Fatalf("%s real agent did not upload and ACK counted traffic for %s", service, deviceID)
	}
	return event
}

func (s *liveSuite) captureFirstServing() {
	var raw []byte
	var received time.Time
	err := s.pool.QueryRow(s.ctx, `SELECT report_payload, received_at FROM node_observations
		WHERE node_id=$1 AND incarnation_id=$2 AND safety_status='passed'
		AND report_payload->>'member_forwarding_enabled'='true' AND received_at>$3
		ORDER BY received_at ASC LIMIT 1`, s.nodeID, s.incarnationID, s.d067BringUp.ReadyObservationReceivedAt).Scan(&raw, &received)
	if err != nil {
		s.t.Fatalf("first serving observation: %v", err)
	}
	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		s.t.Fatal(err)
	}
	// Read-only, inside the node network namespace through the lane's lab exec
	// helper. The harness never edits a foreign nft table.
	rules, code, err := s.execInContainer("node", "nft", "-j", "list", "ruleset")
	if err != nil || code != 0 {
		s.t.Fatalf("read first serving ruleset: %v", err)
	}
	plain, code, err := s.execInContainer("node", "nft", "list", "ruleset")
	if err != nil || code != 0 {
		s.t.Fatalf("read first serving ruleset text: %v", err)
	}
	rs, err := enforce.ParseRuleset(rules)
	if err != nil {
		s.t.Fatal(err)
	}
	loopback := enforce.LoopbackEvidence(rs)
	dockerNAT := strings.Contains(string(rules), "DOCKER_OUTPUT") || strings.Contains(string(rules), "DOCKER_POSTROUTING")
	classification := "no foreign rules; Docker DNS NAT absent; #240 lab cleanup made this node serve; D-079 present but not exercised"
	if dockerNAT || len(rs.Foreign) != 0 {
		classification = fmt.Sprintf("Docker DNS NAT present=%t; foreign chains=%d; D-079 loopback_scoped entries=%d", dockerNAT, len(rs.Foreign), len(loopback))
	}
	imageSource, _ := os.ReadFile(filepath.Join(s.root, ".agent/usage-image-source.txt"))
	imageMain, _ := os.ReadFile(filepath.Join(s.root, ".agent/usage-image-main.txt"))
	hb, _, _ := s.execInContainer("node", "cat", "/run/vpn-node/heartbeat")
	s.firstServing = map[string]any{
		"image_build_commit":               strings.TrimSpace(string(imageSource)),
		"image_origin_main_commit":         strings.TrimSpace(string(imageMain)),
		"agent_core_followup20_in_image":   false,
		"observation":                      report,
		"received_at":                      received.UTC().Format(time.RFC3339Nano),
		"heartbeat":                        strings.TrimSpace(string(hb)),
		"nft_ruleset_excerpt":              string(plain),
		"foreign_chains":                   rs.Foreign,
		"loopback_evidence":                loopback,
		"docker_dns_nat_present":           dockerNAT,
		"foreign_rule_classification":      classification,
		"observation_classification_field": "absent: ObservationReport has no foreign-rule classification field on this tree; classification above is enforce.LoopbackEvidence over the live read-back",
	}
	s.sharedCmds = append(s.sharedCmds, CommandRecord{
		Command:  "lab execInContainer(node): nft list ruleset; nft -j list ruleset; first serving observation",
		ExitCode: 0, At: time.Now().UTC().Format(time.RFC3339),
		Output: fmt.Sprintf("report_id=%v; %s\n%s", report["report_id"], classification, plain),
	})
	if !s.bootstrapTransfer.Counted || !strings.Contains(string(hb), "serving 1") {
		s.t.Fatalf("serving bootstrap lacks counted flow: %+v heartbeat=%s", s.bootstrapTransfer, hb)
	}
	s.t.Logf("first serving report=%v; %s; counted=%t", report["report_id"], classification, s.bootstrapTransfer.Counted)
}
