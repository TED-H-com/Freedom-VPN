//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func (r *runner) publish(ctx context.Context, admin *session, nodeID uuid.UUID) {
	rec := Record{ID: "publish", Title: "Publish the enrolled exit", Spec: []string{"§5.2", "§14.2"}, Status: StatusBlocked}
	if nodeID == uuid.Nil {
		nodeID = r.nodeID
	}
	if nodeID == uuid.Nil {
		nodeID = r.enrolledID
	}
	if nodeID == uuid.Nil {
		rec.Observations = append(rec.Observations, Observation{Name: "node", Detail: "absent"})
		r.record(rec)
		return
	}
	// Wait for the agent's accepted baseline report, not the worker's ready
	// projection: the latter is precisely the central gate this scenario tests.
	conn, dbErr := r.lab.db(ctx)
	if dbErr == nil {
		defer conn.Close(ctx)
		wait, cancel := context.WithTimeout(ctx, 90*time.Second)
		ready := waitUntil(wait, 2*time.Second, func() bool {
			var ready bool
			err := conn.QueryRow(wait, `SELECT EXISTS(SELECT 1 FROM node_observations
				WHERE node_id=$1 AND applied_generation>0 AND safety_status='passed'
				AND report_payload->>'provisioning_state'='ready')`, nodeID).Scan(&ready)
			return err == nil && ready
		})
		cancel()
		rec.Observations = append(rec.Observations, Observation{Name: "agent_baseline_observed", Detail: boolText(ready)})
		var snapshot []byte
		err := conn.QueryRow(ctx, `SELECT jsonb_build_object(
			'node', jsonb_build_object('provisioning_state', n.provisioning_state,
				'safety_status', n.safety_status, 'data_health', n.data_health,
				'desired_config_generation', n.desired_config_generation::text,
				'applied_config_generation', n.applied_config_generation::text,
				'peer_id', i.peer_id),
			'observation', (SELECT jsonb_build_object(
				'provisioning_state', o.report_payload->'provisioning_state',
				'safety_status', o.safety_status, 'data_health', o.data_health,
				'applied_config_generation', o.applied_generation::text,
				'peer_id', o.report_payload->'netbird'->'peer_id',
				'checks', o.report_payload->'checks') FROM node_observations o
				WHERE o.node_id=n.id ORDER BY received_at DESC LIMIT 1),
			'peer_observe_events', (SELECT count(*) FROM outbox_events
				WHERE aggregate_id=i.id AND event_type='node.peer.observe'))
			FROM nodes n JOIN node_incarnations i ON i.id=n.current_incarnation_id WHERE n.id=$1`, nodeID).Scan(&snapshot)
		if err == nil {
			rec.Observations = append(rec.Observations, Observation{Name: "stored_rows", Detail: string(snapshot)})
		} else {
			rec.Observations = append(rec.Observations, Observation{Name: "stored_rows", Detail: "query failed"})
		}
	} else {
		rec.Observations = append(rec.Observations, Observation{Name: "stored_rows", Detail: "database unavailable"})
	}
	detail, err := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/nodes/"+nodeID.String(), nil, nil)
	if err != nil || detail.Status != 200 {
		rec.Status = StatusFail
		rec.Observations = append(rec.Observations, Observation{Name: "get", Detail: httpDetail(detail, err)})
		r.record(rec)
		return
	}
	noteNode(&rec, detail.Body)
	etag := detail.Header.Get("ETag")
	put, err := admin.call(ctx, "PUT", r.lab.public+"/api/v1/admin/nodes/"+nodeID.String()+"/publication", api.PublicationRequest{
		PublicationIntent: api.PublicationIntentPublished, Reason: "e2e acceptance publish",
	}, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": etag})
	rec.Observations = append(rec.Observations, Observation{Name: "put", Detail: httpDetail(put, err)})
	if err != nil {
		rec.Status = StatusFail
		r.record(rec)
		return
	}
	if put.Status != 202 {
		rec.Observations = append(rec.Observations, Observation{Name: "put_error", Detail: apiErrorText(put.Body)})
		r.record(rec)
		return
	}
	opID := operationID(put.Body)
	rec.Observations = append(rec.Observations, Observation{Name: "operation_id", Detail: emptyDash(opID)})
	if opID == "" {
		rec.Observations = append(rec.Observations, Observation{Name: "put_body", Detail: clipText(string(put.Body), 400)})
		r.record(rec)
		return
	}
	wait, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	var last opView
	var failedText string
	ok := waitUntil(wait, 2*time.Second, func() bool {
		again, err := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/operations/"+opID, nil, nil)
		if err != nil || again.Status != 200 {
			return false
		}
		var view opView
		if json.Unmarshal(again.Body, &view) != nil {
			return false
		}
		last = view
		if text := failedStep(view); text != "" && failedText == "" {
			failedText = text
		}
		return view.Status == "succeeded"
	})
	rec.Observations = append(rec.Observations, Observation{Name: "operation_status", Detail: emptyDash(last.Status)})
	rec.Observations = append(rec.Observations, Observation{Name: "operation_kind", Detail: emptyDash(last.Kind)})
	if last.Error != nil {
		rec.Observations = append(rec.Observations, Observation{Name: "operation_error", Detail: strings.TrimSpace(last.Error.Code + " " + last.Error.Message)})
	}
	for _, step := range last.Steps {
		detail := step.Status
		if step.ErrorCode != nil && *step.ErrorCode != "" {
			detail += " " + *step.ErrorCode
		}
		if step.Message != nil && *step.Message != "" {
			detail += " " + *step.Message
		}
		rec.Observations = append(rec.Observations, Observation{Name: "step_" + step.Name, Detail: emptyDash(strings.TrimSpace(detail))})
	}
	again, againErr := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/nodes/"+nodeID.String(), nil, nil)
	rec.Observations = append(rec.Observations, Observation{Name: "node_after", Detail: httpDetail(again, againErr)})
	if againErr == nil && again.Status == 200 {
		noteNode(&rec, again.Body)
	}
	policies, bindings, intent, generation := r.desiredMemberView()
	rec.Observations = append(rec.Observations, Observation{Name: "desired_publication_intent", Detail: emptyDash(intent)})
	rec.Observations = append(rec.Observations, Observation{Name: "desired_generation", Detail: emptyDash(generation)})
	rec.Observations = append(rec.Observations, Observation{Name: "member_rate_policies", Detail: itoa(policies)})
	rec.Observations = append(rec.Observations, Observation{Name: "device_bindings", Detail: itoa(bindings)})
	chain := r.lab.forwardChain()
	rec.Observations = append(rec.Observations, Observation{Name: "forward_generation", Detail: itoa(chain.Generation)})
	rec.Observations = append(rec.Observations, Observation{Name: "member_accept_rules", Detail: itoa(chain.MemberAccept)})
	if failedText == "" {
		failedText = failedStep(last)
	}
	if ok && last.Status == "succeeded" {
		observed := nodeField(again.Body, "publication_observed")
		if observed == "active" {
			rec.Status = StatusPass
			rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "node.publish succeeded and publication_observed is active"})
		} else {
			rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "operation succeeded but publication_observed is " + emptyDash(observed)})
		}
	} else if failedText != "" {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "publication did not succeed: " + failedText})
	} else {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "publication did not reach succeeded within the poll bound"})
	}
	r.record(rec)
}

type opView struct {
	Status string `json:"status"`
	Kind   string `json:"kind"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Steps []struct {
		Name      string  `json:"name"`
		Status    string  `json:"status"`
		ErrorCode *string `json:"error_code"`
		Message   *string `json:"message"`
	} `json:"steps"`
}

func operationID(raw []byte) string {
	var doc struct {
		OperationID string `json:"operation_id"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	return doc.OperationID
}

func failedStep(view opView) string {
	var parts []string
	for _, step := range view.Steps {
		if step.Status != "failed" {
			continue
		}
		item := step.Name
		if step.ErrorCode != nil && *step.ErrorCode != "" {
			item += " " + *step.ErrorCode
		}
		if step.Message != nil && *step.Message != "" {
			item += " " + *step.Message
		}
		parts = append(parts, strings.TrimSpace(item))
	}
	if view.Error != nil && (view.Error.Code != "" || view.Error.Message != "") {
		parts = append(parts, strings.TrimSpace(view.Error.Code+" "+view.Error.Message))
	}
	return strings.Join(parts, "; ")
}

func noteNode(rec *Record, raw []byte) {
	var doc struct {
		ControlStatus       string `json:"control_status"`
		SafetyStatus        string `json:"safety_status"`
		DataHealth          string `json:"data_health"`
		ProvisioningState   string `json:"provisioning_state"`
		LifecycleState      string `json:"lifecycle_state"`
		PublicationIntent   string `json:"publication_intent"`
		PublicationObserved string `json:"publication_observed"`
		ConfigGeneration    struct {
			Desired string `json:"desired"`
			Applied any    `json:"applied"`
		} `json:"config_generation"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return
	}
	rec.Observations = append(rec.Observations,
		Observation{Name: "control_status", Detail: emptyDash(doc.ControlStatus)},
		Observation{Name: "safety_status", Detail: emptyDash(doc.SafetyStatus)},
		Observation{Name: "data_health", Detail: emptyDash(doc.DataHealth)},
		Observation{Name: "provisioning_state", Detail: emptyDash(doc.ProvisioningState)},
		Observation{Name: "lifecycle_state", Detail: emptyDash(doc.LifecycleState)},
		Observation{Name: "publication_intent", Detail: emptyDash(doc.PublicationIntent)},
		Observation{Name: "publication_observed", Detail: emptyDash(doc.PublicationObserved)},
		Observation{Name: "config_desired", Detail: emptyDash(doc.ConfigGeneration.Desired)},
		Observation{Name: "config_applied", Detail: emptyDash(appliedText(doc.ConfigGeneration.Applied))},
	)
	for _, ch := range doc.Checks {
		rec.Observations = append(rec.Observations, Observation{Name: "check_" + ch.Name, Detail: emptyDash(ch.Status)})
	}
}

func appliedText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		raw, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return strings.Trim(string(raw), `"`)
	}
}

func nodeField(raw []byte, name string) string {
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	s, _ := doc[name].(string)
	return s
}

func apiErrorText(raw []byte) string {
	var doc struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &doc) != nil || doc.Error.Code == "" {
		return clipText(string(raw), 500)
	}
	text := doc.Error.Code + " " + doc.Error.Message
	if len(doc.Error.Details) > 0 {
		extra, err := json.Marshal(doc.Error.Details)
		if err == nil {
			text += " " + string(extra)
		}
	}
	return clipText(text, 800)
}

func (r *runner) desiredMemberView() (policies, bindings int, intent, generation string) {
	raw, err := os.ReadFile(filepath.Join(r.lab.state, "node", "var", "state", "desired-state.json"))
	if err != nil || len(raw) == 0 {
		raw, err = r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/state/desired-state.json")
	}
	if err != nil || len(raw) == 0 {
		return 0, 0, "", ""
	}
	var env struct {
		Signed struct {
			PublicationIntent  string            `json:"publication_intent"`
			ConfigGeneration   string            `json:"config_generation"`
			MemberRatePolicies []json.RawMessage `json:"member_rate_policies"`
			DeviceBindings     []json.RawMessage `json:"device_bindings"`
		} `json:"signed"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return 0, 0, "", ""
	}
	return len(env.Signed.MemberRatePolicies), len(env.Signed.DeviceBindings), env.Signed.PublicationIntent, env.Signed.ConfigGeneration
}
