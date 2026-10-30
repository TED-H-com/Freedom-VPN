//go:build e2e

package enrollment

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

type check struct {
	AT     string `json:"at"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail any    `json:"detail"`
	At     string `json:"observed_at"`
}

type probe struct {
	ctx        context.Context
	stack      *labstack.Stack
	admin      labstack.BrowserSession
	client     *http.Client
	checks     []check
	secrets    [][]byte
	resultPath string
}

func (p *probe) note(at, name, status string, detail any) {
	p.checks = append(p.checks, check{at, name, status, detail, time.Now().UTC().Format(time.RFC3339Nano)})
	// Incremental private spool survives an interrupted live test. Only typed,
	// allowlisted observations are published by the records-first runner.
	raw, err := json.MarshalIndent(p.checks, "", "  ")
	if err != nil {
		panic("cannot serialize safe probe evidence")
	}
	if err = os.WriteFile(p.resultPath+".tmp", raw, 0600); err != nil {
		panic("cannot save probe evidence")
	}
	if err = os.Rename(p.resultPath+".tmp", p.resultPath); err != nil {
		panic("cannot commit probe evidence")
	}
}

func (p *probe) expect(at, name string, ok bool, detail any) bool {
	status := "fail"
	if ok {
		status = "pass"
	}
	p.note(at, name, status, detail)
	return ok
}

func TestAT06_AT07_Live(t *testing.T) {
	if os.Getenv("FVPN_E2E") != "1" {
		t.Skip("requires fresh owned stack through at-enrollment target")
	}
	if os.Getenv("FVPN_E2E_PREFIX") != "w4-enroll" || os.Getenv("FVPN_E2E_PORT_BASE") != "12000" {
		t.Fatal("refusing a stack outside the assigned binding")
	}
	path := os.Getenv("FVPN_AT_ENROLLMENT_PROBE")
	if path == "" {
		t.Fatal("records-first runner required")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Minute)
	defer cancel()
	p := &probe{ctx: ctx, resultPath: path}
	stack, err := labstack.Open()
	if err != nil {
		p.note("AT06", "stack_open", "blocked", "ENR-01: prepared stack unavailable")
		return
	}
	defer stack.Close()
	p.stack = stack
	admin, err := stack.Login(ctx, "e2e-admin", "admin")
	if err != nil {
		p.note("AT06", "admin_login", "blocked", "authenticated admin prerequisite unavailable")
		return
	}
	p.admin = admin
	p.client = &http.Client{Transport: admin.Client.Transport, Timeout: 30 * time.Second}
	baseline, err := labstack.PrepareServingBaseline(ctx, admin.Client, admin.CSRF, false)
	if err != nil {
		p.note("AT06", "serving_baseline", "blocked", "signed serving baseline unavailable")
		return
	}
	p.note("AT06", "serving_baseline", "pass", map[string]any{"release_digests": baseline.Digests, "support_revision": baseline.SupportRevision})
	t.Run("AT06", func(t *testing.T) { p.at06(t) })
	t.Run("AT07", func(t *testing.T) { p.at07(t) })
	p.scanLogs()
	p.scanEvidence()
	for _, c := range p.checks {
		if c.Status == "fail" {
			t.Errorf("%s/%s failed; see safe evidence", c.AT, c.Name)
		}
	}
	// Missing required checks are rejected by run.py even when these partial
	// product probes pass. Never use this test's exit alone as an AT verdict.
}

func (p *probe) call(method, path string, body any) (labstack.Response, error) {
	return p.stack.Call(p.ctx, p.admin, method, path, body, map[string]string{"Idempotency-Key": uuid.NewString()})
}

type issued struct {
	token      string
	enrollment api.Enrollment
}

func (p *probe) issue(name string, ttl int) (issued, bool) {
	resp, err := p.call("POST", "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name, ExpiresInSeconds: &ttl})
	var out api.EnrollmentCreated
	if !p.expect("AT06", "issue_"+name, err == nil && resp.Status == 201 && json.Unmarshal(resp.Body, &out) == nil && out.SecretAvailable, safeResponse(resp, err)) {
		return issued{}, false
	}
	token, err := out.Token.Get()
	if err != nil || token == "" {
		p.note("AT06", "issue_secret", "fail", "one-time token absent")
		return issued{}, false
	}
	p.secrets = append(p.secrets, []byte(token))
	p.expect("AT06", "issue_no_store_"+name, resp.Header.Get("Cache-Control") == "no-store", resp.Header.Get("Cache-Control"))
	return issued{token, out.Enrollment}, true
}

func (p *probe) holder(t *testing.T) holder {
	t.Helper()
	h, err := newHolder()
	if err != nil {
		t.Fatal("key/proof generation failed")
	}
	der, err := x509.MarshalECPrivateKey(h.key)
	if err != nil {
		t.Fatal("key scan material unavailable")
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(h.key)
	if err != nil {
		t.Fatal("key scan material unavailable")
	}
	// PEM wraps base64, so scan individual sufficiently long lines as well.
	scalar, err := h.key.Bytes()
	if err != nil {
		t.Fatal("key scan material unavailable")
	}
	for _, encoded := range [][]byte{der, pkcs8, scalar} {
		p.secrets = append(p.secrets, []byte(base64.StdEncoding.EncodeToString(encoded)), []byte(fmt.Sprintf("%x", encoded)))
		block := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded})
		for _, line := range bytes.Split(block, []byte("\n")) {
			if len(line) >= 40 && !bytes.HasPrefix(line, []byte("-----")) {
				p.secrets = append(p.secrets, line)
			}
		}
	}
	p.note("AT06", "csr_proof", "pass", map[string]any{"spki_sha256": h.fingerprint, "proof_input_sha256": h.proofHash, "request_sha256": hash(h.body), "incarnation_id": h.incarnation, "csr_self_signed": true})
	return h
}

func post(ctx context.Context, client *http.Client, url, token string, body []byte) (labstack.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return labstack.Response{}, err
	}
	req.Header.Set("Authorization", "Enrollment "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return labstack.Response{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &env)
	return labstack.Response{Status: resp.StatusCode, Header: resp.Header, Body: raw, Code: env.Error.Code}, err
}

func safeResponse(r labstack.Response, err error) map[string]any {
	// Never export arbitrary error messages, headers or HTTP response bodies.
	return map[string]any{"http_status": r.Status, "error_code": r.Code, "transport_error": err != nil}
}

func (p *probe) at06(t *testing.T) {
	url := p.stack.PublicURL() + "/api/v1/agent/enroll"
	h := p.holder(t)
	one, ok := p.issue("replay", 900)
	if !ok {
		return
	}
	drop := &dropSuccess{base: p.client.Transport}
	lostClient := &http.Client{Transport: drop, Timeout: 30 * time.Second}
	resp, err := post(p.ctx, lostClient, url, one.token, h.body)
	if !p.expect("AT06", "response_dropped", errors.Is(err, errDropped) && drop.dropped && resp.Status == 0, map[string]any{"observer_http_status": drop.status, "client_received_response": resp.Status != 0, "request_sha256": hash(h.body)}) {
		return
	}
	resp, err = post(p.ctx, p.client, url, one.token, h.body)
	var result api.EnrollResponse
	valid := err == nil && resp.Status == 200 && json.Unmarshal(resp.Body, &result) == nil
	p.expect("AT06", "same_holder_retry", valid && result.NodeId == drop.node && result.NodeIncarnationId == drop.incarnation && result.NodeIncarnationId == h.incarnation && hash([]byte(result.CertificatePem)) == drop.certificateHash, map[string]any{"response": safeResponse(resp, err), "node_id": result.NodeId, "incarnation_id": result.NodeIncarnationId, "certificate_sha256": hash([]byte(result.CertificatePem)), "identical_request_sha256": hash(h.body)})
	p.expect("AT06", "retry_no_store", resp.Header.Get("Cache-Control") == "no-store", resp.Header.Get("Cache-Control"))
	p.binding(one, h)
	// At this point this fresh stack has exactly one API-reserved Node.
	list, listErr := p.call("GET", "/api/v1/admin/nodes?limit=100", nil)
	var nodes api.NodeList
	valid = listErr == nil && list.Status == 200 && json.Unmarshal(list.Body, &nodes) == nil
	p.expect("AT06", "one_node", valid && len(nodes.Items) == 1 && nodes.Items[0].NodeId == one.enrollment.NodeId && nodes.NextCursor.IsNull(), map[string]any{"response": safeResponse(list, listErr), "node_count": len(nodes.Items)})
	p.current("AT06", one.enrollment.NodeId, h.incarnation)
	p.audit(one)

	// A complete request missing proof exercises more than a schema-invalid {}.
	var noProof map[string]any
	if json.Unmarshal(h.body, &noProof) != nil {
		t.Fatal("request decode failed")
	}
	delete(noProof, "proof")
	missingProof, err := json.Marshal(noProof)
	if err != nil {
		t.Fatal("request encode failed")
	}
	resp, err = post(p.ctx, p.client, url, one.token, missingProof)
	p.expect("AT06", "token_only", err == nil && resp.Status == 422 && resp.Code == "VALIDATION_FAILED" && !bytes.Contains(resp.Body, []byte("certificate_pem")), safeResponse(resp, err))
	// A structurally valid but false proof must also be refused.
	noProof["proof"] = base64.StdEncoding.EncodeToString(make([]byte, 64))
	badProof, err := json.Marshal(noProof)
	if err != nil {
		t.Fatal("request encode failed")
	}
	resp, err = post(p.ctx, p.client, url, one.token, badProof)
	p.expect("AT06", "invalid_proof", err == nil && resp.Status == 422 && resp.Code == "VALIDATION_FAILED" && !bytes.Contains(resp.Body, []byte("certificate_pem")), safeResponse(resp, err))
	resp, err = post(p.ctx, p.client, "https://agent.example.invalid:9443/api/v1/agent/bootstrap", one.token, nil)
	// TLS client-certificate refusal is not an application error code. Accept
	// only the explicit TLS alert; arbitrary network failure is not denial proof.
	tlsDenied := err != nil && strings.Contains(err.Error(), "tls: certificate required")
	p.expect("AT06", "bootstrap_requires_mtls", tlsDenied || (err == nil && resp.Status == 401 && resp.Code == "UNAUTHENTICATED"), map[string]any{"response": safeResponse(resp, err), "tls_certificate_required": tlsDenied})

	race, ok := p.issue("race", 900)
	if !ok {
		return
	}
	holders := []holder{p.holder(t), p.holder(t)}
	responses := make([]labstack.Response, 2)
	errs := make([]error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range holders {
		wg.Go(func() { <-start; responses[i], errs[i] = post(p.ctx, p.client, url, race.token, holders[i].body) })
	}
	close(start)
	wg.Wait()
	wins, denied, winner := 0, 0, -1
	for i, response := range responses {
		if errs[i] == nil && response.Status == 201 {
			wins++
			winner = i
		}
		if errs[i] == nil && response.Status == 409 && response.Code == "ENROLLMENT_KEY_MISMATCH" {
			denied++
		}
	}
	p.expect("AT06", "different_key_race", wins == 1 && denied == 1 && holders[0].fingerprint != holders[1].fingerprint, []any{safeResponse(responses[0], errs[0]), safeResponse(responses[1], errs[1])})
	if wins == 1 {
		var won api.EnrollResponse
		p.expect("AT06", "race_winner_identity", json.Unmarshal(responses[winner].Body, &won) == nil && won.NodeId == race.enrollment.NodeId && won.NodeIncarnationId == holders[winner].incarnation,
			map[string]any{"node_id": won.NodeId, "incarnation_id": won.NodeIncarnationId})
		p.binding(race, holders[winner])
		p.current("AT06", race.enrollment.NodeId, holders[winner].incarnation)
	}
	p.audit(race)

	expired, ok := p.issue("expiry", 60)
	if !ok {
		return
	}
	revoked, ok := p.issue("revocation", 900)
	if !ok {
		return
	}
	resp, err = p.call("POST", "/api/v1/admin/enrollments/"+revoked.enrollment.EnrollmentId.String()+"/revoke", api.ReasonRequest{Reason: "TEST ONLY AT06"})
	if p.expect("AT06", "revoke_unused", err == nil && resp.Status == 200, safeResponse(resp, err)) {
		resp, err = post(p.ctx, p.client, url, revoked.token, p.holder(t).body)
		p.expect("AT06", "revoked_token", err == nil && resp.Status == 401 && resp.Code == "ENROLLMENT_TOKEN_REVOKED", safeResponse(resp, err))
	}
	// Natural expiration; never change host clock or an enrollment row.
	wait := time.Until(expired.enrollment.ExpiresAt.Add(time.Second))
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-p.ctx.Done():
			return
		}
	}
	resp, err = post(p.ctx, p.client, url, expired.token, p.holder(t).body)
	p.expect("AT06", "expired_token", err == nil && resp.Status == 401 && resp.Code == "ENROLLMENT_TOKEN_EXPIRED", safeResponse(resp, err))
}

func (p *probe) binding(token issued, h holder) {
	r, err := p.call("GET", "/api/v1/admin/enrollments/"+token.enrollment.EnrollmentId.String(), nil)
	var e api.Enrollment
	valid := err == nil && r.Status == 200 && json.Unmarshal(r.Body, &e) == nil
	fp, fpErr := e.BoundPublicKeySha256.Get()
	inc, incErr := e.NodeIncarnationId.Get()
	p.expect("AT06", "consumed_binding", valid && fpErr == nil && incErr == nil && e.State == "consumed" && fp == h.fingerprint && inc == h.incarnation, map[string]any{"response": safeResponse(r, err), "state": e.State, "bound_spki_sha256": fp, "incarnation_id": inc})
}

func (p *probe) current(at string, node, incarnation uuid.UUID) api.NodeDetail {
	r, err := p.call("GET", "/api/v1/admin/nodes/"+node.String(), nil)
	var detail api.NodeDetail
	valid := err == nil && r.Status == 200 && json.Unmarshal(r.Body, &detail) == nil
	inc, incErr := detail.CurrentIncarnationId.Get()
	p.expect(at, "current_incarnation", valid && incErr == nil && inc == incarnation && detail.NodeId == node, map[string]any{"response": safeResponse(r, err), "node_id": detail.NodeId, "current_incarnation_id": inc})
	if valid {
		var fields map[string]json.RawMessage
		if json.Unmarshal(r.Body, &fields) == nil {
			peer, peerErr := detail.NetbirdPeerId.Get()
			p.note(at, "node_api_surface", "pass", map[string]any{"operation": "getNode", "field_names": fieldNames(fields), "node_id": node, "current_incarnation_id": inc, "peer_bound": peerErr == nil && peer != ""})
		}
	}
	return detail
}

func fieldNames[T any](fields map[string]T) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (p *probe) audit(token issued) {
	r, err := p.call("GET", "/api/v1/admin/audit-events?resource_type=enrollment&resource_id="+token.enrollment.EnrollmentId.String()+"&limit=100", nil)
	var events api.AuditEventList
	if !p.expect("AT06", "audit_read", err == nil && r.Status == 200 && json.Unmarshal(r.Body, &events) == nil && events.NextCursor.IsNull(), safeResponse(r, err)) {
		return
	}
	consumed, matched := 0, false
	var observed []map[string]any
	for _, event := range events.Items {
		if event.Action == "enrollment.consumed" {
			consumed++
		}
		after, e := event.Changes.After.Get()
		before, _ := event.Changes.Before.Get()
		observed = append(observed, map[string]any{"action": event.Action, "result": event.Result,
			"resource": event.Resource, "audit_event_id": event.AuditEventId, "request_id": event.RequestId,
			"operation_id": event.OperationId, "before_fields": fieldNames(before), "after_fields": fieldNames(after)})
		if e == nil {
			if value, ok := after["token_hash"].(string); ok && value == hash([]byte(token.token)) {
				matched = true
			}
		}
	}
	p.note("AT06", "enrollment_audit_fields", "pass", observed)
	p.expect("AT06", "single_consumption_audit", consumed == 1, map[string]any{"consumed_events": consumed, "resource_id": token.enrollment.EnrollmentId})
	status := "blocked"
	if matched {
		status = "pass"
	}
	p.note("AT06", "audit_token_hash", status, map[string]any{"api_hash_matches": matched, "request": "ENR-02", "owner": "pki-enroll", "operation": "listAuditEvents", "field": "changes.after.token_hash", "resource_id": token.enrollment.EnrollmentId})
}

func (p *probe) at07(t *testing.T) {
	first, err := p.stack.Enroll(p.ctx, p.admin, "TEST ONLY AT07 first")
	if err != nil {
		p.note("AT07", "first_boot", "blocked", "normal Enroll prerequisite failed")
		return
	}
	if err = p.stack.StartSecondExit(p.ctx); err != nil {
		p.note("AT07", "second_boot", "blocked", "StartSecondExit prerequisite failed")
		return
	}
	second, err := p.stack.EnrollAt(p.ctx, p.admin, "TEST ONLY AT07 second", "node2")
	if err != nil {
		p.note("AT07", "second_boot", "blocked", "normal EnrollAt prerequisite failed")
		return
	}
	p.current("AT07", first.NodeID, first.IncarnationID)
	p.current("AT07", second.NodeID, second.IncarnationID)
	images := make([]string, 2)
	for i, slot := range []string{"node", "node2"} {
		// Image provenance only. MachineIdentityFingerprints is the required
		// devstack export; do not emulate it by reading live identity files.
		image, e := exec.CommandContext(p.ctx, "docker", "inspect", "--format={{.Image}}", "fvpn-w4-enroll-"+slot).Output()
		if e != nil {
			t.Fatal("image digest unavailable")
		}
		images[i] = strings.TrimSpace(string(image))
	}
	p.expect("AT07", "distinct_api_first_boot", first.NodeID != second.NodeID && first.IncarnationID != second.IncarnationID && images[0] == images[1] && strings.HasPrefix(images[0], "sha256:"), map[string]any{"node_ids": []uuid.UUID{first.NodeID, second.NodeID}, "incarnation_ids": []uuid.UUID{first.IncarnationID, second.IncarnationID}, "image_digests": images})
	p.nodeAudit(first.NodeID)
	p.nodeAudit(second.NodeID)
	// PauseAgent resumes durable identity; it is not an installer rerun or a
	// usage-ACK hold. No unsupported mutation is substituted here.
}

func (p *probe) nodeAudit(node uuid.UUID) {
	r, err := p.call("GET", "/api/v1/admin/audit-events?resource_type=node&resource_id="+node.String()+"&limit=100", nil)
	var events api.AuditEventList
	if !p.expect("AT07", "node_audit_read", err == nil && r.Status == 200 && json.Unmarshal(r.Body, &events) == nil && events.NextCursor.IsNull(), safeResponse(r, err)) {
		return
	}
	observed := make([]map[string]any, 0, len(events.Items))
	for _, event := range events.Items {
		after, _ := event.Changes.After.Get()
		before, _ := event.Changes.Before.Get()
		observed = append(observed, map[string]any{"action": event.Action, "resource": event.Resource,
			"before_fields": fieldNames(before), "after_fields": fieldNames(after)})
	}
	p.note("AT07", "node_audit_fields", "pass", observed)
}

func (p *probe) scanEvidence() {
	raw, err := json.Marshal(p.checks)
	p.scanStream("current_probe_evidence", raw, err)
	root := filepath.Dir(filepath.Dir(p.stack.StateDir()))
	count := 0
	err = filepath.WalkDir(filepath.Join(root, "docs/e2e/at/enrollment"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		p.scanStream("evidence/"+strings.TrimPrefix(path, root+string(os.PathSeparator)), raw, readErr)
		count++
		return readErr
	})
	p.expect("AT06", "evidence_set_scan", err == nil && count > 0, map[string]any{"files_scanned": count, "read_error": err != nil})
}

func (p *probe) scanLogs() {
	for _, service := range []string{"control", "worker", "node", "node2"} {
		if service == "node2" {
			if _, err := os.Stat(filepath.Join(p.stack.StateDir(), "node2")); os.IsNotExist(err) {
				continue
			}
		}
		raw, err := exec.CommandContext(p.ctx, "docker", "logs", "fvpn-w4-enroll-"+service).CombinedOutput()
		p.scanStream(service+"/console", raw, err)
		if service == "node" || service == "node2" {
			for _, file := range []string{"agent.log", "watchdog.log", "netbird-service.log", "netbird-up.log", "resolver-proxy.log"} {
				raw, err = exec.CommandContext(p.ctx, "docker", "exec", "fvpn-w4-enroll-"+service, "cat", "/var/lib/vpn-node/e2e/"+file).Output()
				p.scanStream(service+"/"+file, raw, err)
			}
		}
	}
}

func (p *probe) scanStream(name string, raw []byte, err error) {
	if err != nil {
		p.note("AT06", "secret_scan_"+name, "blocked", "stream unavailable; not counted as zero matches")
		return
	}
	hits := 0
	for _, secret := range p.secrets {
		if len(secret) >= 16 {
			hits += bytes.Count(raw, secret)
		}
	}
	// This sentinel also catches a private key unrelated to our generated keys.
	hits += bytes.Count(raw, []byte("PRIVATE KEY-----"))
	p.expect("AT06", "secret_scan_"+name, hits == 0, map[string]any{"bytes_scanned": len(raw), "matches": hits, "needles": len(p.secrets)})
}
