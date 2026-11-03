//go:build e2e

package members

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

// Read the serving namespace before opening the original bulk connection. Only
// the member's general accepts and their immediate predecessors are published;
// the complete table has a byte pin. No kernel rule is added or changed here.
func (l *live) d146Rules(device uuid.UUID) map[string]any {
	start := time.Now().UTC()
	raw, err := l.inside(l.nodeContainer(), "nft", "-j", "list", "table", "inet", "fvpn")
	end := time.Now().UTC()
	out := map[string]any{"available": false, "serving": false, "started_at": start, "ended_at": end, "node_role": l.nodeContainer(), "node_incarnation": l.inc}
	if err != nil {
		return out
	}
	sum := sha256.Sum256(raw)
	out["table_sha256"] = hex.EncodeToString(sum[:])
	state, err := enforce.ParseState(raw)
	if err != nil {
		return out
	}
	out["generation"] = fmt.Sprint(state.Generation)
	var table struct {
		NFTables []struct {
			Rule json.RawMessage `json:"rule"`
		} `json:"nftables"`
	}
	if json.Unmarshal(raw, &table) != nil {
		return out
	}
	var rules []json.RawMessage
	pairs := map[string]any{}
	key := strings.ReplaceAll(device.String(), "-", "")
	for _, entry := range table.NFTables {
		var rule struct {
			Chain string           `json:"chain"`
			Expr  []map[string]any `json:"expr"`
		}
		if json.Unmarshal(entry.Rule, &rule) != nil || rule.Chain != "forward" {
			continue
		}
		if len(rule.Expr) == 6 && len(rules) > 0 {
			_, accept := rule.Expr[5]["accept"]
			for direction, name := range map[string]string{"upload": "u_" + key, "download": "d_" + key} {
				if accept && rule.Expr[3]["counter"] == name {
					pairs[direction] = map[string]any{"clamp_index": len(rules) - 1, "accept_index": len(rules), "clamp": rules[len(rules)-1], "accept": entry.Rule}
				}
			}
		}
		rules = append(rules, entry.Rule)
	}
	out["pairs"], out["available"] = pairs, true
	var observation []byte
	err = l.pool.QueryRow(l.ctx, `SELECT jsonb_build_object('report_id',o.report_id,'received_at',o.received_at,'generation',o.applied_generation::text,'forwarding',o.report_payload->'member_forwarding_enabled','reasons',o.report_payload->'forwarding_block_reasons') FROM nodes n JOIN LATERAL (SELECT * FROM node_observations WHERE node_id=n.id AND incarnation_id=n.current_incarnation_id ORDER BY received_at DESC LIMIT 1) o ON true WHERE n.id=$1 AND n.desired_config_generation=n.applied_config_generation AND o.applied_generation=n.applied_config_generation AND o.applied_generation=$2 AND o.received_at>clock_timestamp()-interval '15 seconds' AND o.report_payload->>'member_forwarding_enabled'='true' AND o.report_payload->'forwarding_block_reasons'='[]'::jsonb`, l.node, int64(state.Generation)).Scan(&observation)
	out["serving_read_ok"] = err == nil
	if err == nil {
		out["serving"] = true
		out["accepted_observation"] = json.RawMessage(observation)
	}
	return out
}

func d146Upload(got labstack.FixtureProbeResult, device uuid.UUID) bool {
	// The unmodified echo fixture verifies each byte at the origin by requiring
	// all 64 exact echoes. Write completion alone does not prove delivery.
	return got.Available && got.Bytes == 64064 && got.Endpoint != nil &&
		got.Endpoint.WriteBytes-int64(len("MEMBER "+device.String()+"\n")) == 64064 &&
		got.Endpoint.Lines == 64 && got.Endpoint.ReadBytes == 64387 && got.Endpoint.Error == "none"
}

func (l *live) d146Qualification(r *record, capture *nativeCaptureSession, data map[string]any) bool {
	if capture == nil {
		return false
	}
	input := filepath.Join(capture.directory, "d146-input.json")
	raw, err := json.Marshal(data)
	if err == nil {
		err = os.WriteFile(input, raw, 0600)
	}
	if err != nil {
		r.block("D-146 original input retention unavailable")
		return false
	}
	dest := filepath.Join(l.root, "docs/e2e/at/members/history/follow-up-18-captures", l.runID, capture.label+"-d146")
	cmd := exec.CommandContext(l.ctx, "python3", "-B", "tests/e2e/at/members/d146_evidence.py", "--input", input, "--private", capture.directory, "--out", dest)
	cmd.Dir = l.root
	value, err := cmd.Output()
	var result struct {
		Qualified bool `json:"qualified"`
	}
	if err != nil || json.Unmarshal(value, &result) != nil {
		r.block("D-146 evidence analysis unavailable; original row remains unqualified")
		return false
	}
	status := "blocked"
	if result.Qualified {
		status = "pass"
	}
	r.Steps = append(r.Steps, step{Name: data["row"].(string) + " D-146 clamp evidence", Status: status,
		Expected: "new handshake after MTU/rules; both MSS advertisements above 1160 arrive as 1160; both byte predicates; every reverse datagram fits 1280 and arrives; no PTB at any point",
		Actual:   string(value), Command: "read-only serving rules and original captures; header/hash joins; no retry", At: time.Now().UTC().Format(time.RFC3339)})
	return result.Qualified
}
