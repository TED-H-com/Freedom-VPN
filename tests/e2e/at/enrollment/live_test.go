//go:build e2e

package enrollment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
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
	digests    [][]byte // lowercase token SHA-256; allowed in logs and admin reads
	derived    [][]byte // other token-derived encodings; must be absent
	reads      [][]byte // admin read and revoke bodies, never creation responses
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
	// "join" is a second fresh stack. JoinEnrolledIdentity needs node2 while it
	// is still waiting; the acceptance stack uses that slot for EnrollAt.
	if os.Getenv("FVPN_AT_ENROLLMENT_PHASE") == "join" {
		p.peerUniqueness(t)
	} else {
		t.Run("AT06", func(t *testing.T) { p.at06(t) })
		t.Run("AT07", func(t *testing.T) { p.at07(t) })
	}
	join := os.Getenv("FVPN_AT_ENROLLMENT_PHASE") == "join"
	p.scanLogs(!join)
	if !join || len(p.reads) > 0 {
		p.scanAdminReads()
	}
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
	return p.callHeaders(method, path, body, nil)
}

func (p *probe) callHeaders(method, path string, body any, headers map[string]string) (labstack.Response, error) {
	h := map[string]string{"Idempotency-Key": uuid.NewString()}
	for k, v := range headers {
		h[k] = v
	}
	var last labstack.Response
	var err error
	for attempt := 0; attempt < 6; attempt++ {
		last, err = p.stack.Call(p.ctx, p.admin, method, path, body, h)
		if err != nil || last.Status != http.StatusTooManyRequests || !waitRetry(p.ctx, last.Header.Get("Retry-After")) {
			return last, err
		}
	}
	return last, err
}

func waitRetry(ctx context.Context, header string) bool {
	wait := time.Second
	if n, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && n > 0 && n <= 30 {
		wait = time.Duration(n) * time.Second
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func (p *probe) keep(r labstack.Response) {
	if len(r.Body) > 0 {
		p.reads = append(p.reads, append([]byte(nil), r.Body...))
	}
}

type issued struct {
	token      string
	digest     string
	enrollment api.Enrollment
}

func (p *probe) rememberToken(token string) {
	p.secrets = append(p.secrets, []byte(token))
	digest := hash([]byte(token))
	p.digests = append(p.digests, []byte(digest))
	sum := sha256Sum(token)
	p.derived = append(p.derived,
		[]byte(strings.ToUpper(digest)),
		[]byte(base64.StdEncoding.EncodeToString(sum[:])),
		[]byte(base64.RawURLEncoding.EncodeToString(sum[:])),
	)
}

func sha256Sum(token string) [32]byte {
	return sha256.Sum256([]byte(token))
}

func (p *probe) issue(name string, ttl int) (issued, bool) {
	return p.issueOn(name, ttl, nil)
}

func (p *probe) issueOn(name string, ttl int, node *uuid.UUID) (issued, bool) {
	req := api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name, ExpiresInSeconds: &ttl, NodeId: node}
	resp, err := p.call("POST", "/api/v1/admin/enrollments", req)
	var out api.EnrollmentCreated
	ok := err == nil && resp.Status == 201 && json.Unmarshal(resp.Body, &out) == nil && out.SecretAvailable
	if node != nil {
		ok = ok && out.Enrollment.NodeId == *node
	}
	if !p.expect("AT06", "issue_"+name, ok, safeResponse(resp, err)) {
		return issued{}, false
	}
	token, err := out.Token.Get()
	if err != nil || token == "" {
		p.note("AT06", "issue_secret", "fail", "one-time token absent")
		return issued{}, false
	}
	p.rememberToken(token)
	digest := hash([]byte(token))
	got := ""
	if out.Enrollment.TokenSha256 != nil {
		got = *out.Enrollment.TokenSha256
	}
	// The creation response is the one response allowed to carry the plaintext.
	// It is not an admin read and is not added to the read scan set.
	p.expect("AT06", "token_sha256_create", got == digest && bytes.Contains(resp.Body, []byte(digest)) && bytes.Contains(resp.Body, []byte(token)), map[string]any{
		"digest": digest, "matches_create": got == digest, "enrollment_id": out.Enrollment.EnrollmentId, "node_id": out.Enrollment.NodeId,
	})
	p.expect("AT06", "issue_no_store_"+name, resp.Header.Get("Cache-Control") == "no-store", resp.Header.Get("Cache-Control"))
	p.readDigest(out.Enrollment.EnrollmentId, digest, token)
	return issued{token, digest, out.Enrollment}, true
}

func (p *probe) readDigest(id uuid.UUID, digest, token string) {
	r, err := p.call("GET", "/api/v1/admin/enrollments/"+id.String(), nil)
	p.keep(r)
	var e api.Enrollment
	valid := err == nil && r.Status == 200 && json.Unmarshal(r.Body, &e) == nil
	got := ""
	if e.TokenSha256 != nil {
		got = *e.TokenSha256
	}
	plain := bytes.Contains(r.Body, []byte(token))
	p.expect("AT06", "token_sha256_get", valid && got == digest && !plain, map[string]any{
		"digest": digest, "matches_get": got == digest, "plaintext_absent": !plain, "enrollment_id": id, "state": e.State,
	})
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
	var last labstack.Response
	var err error
	for attempt := 0; attempt < 4; attempt++ {
		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return labstack.Response{}, err
		}
		req.Header.Set("Authorization", "Enrollment "+token)
		req.Header.Set("Content-Type", "application/json")
		var resp *http.Response
		resp, err = client.Do(req)
		if err != nil {
			return labstack.Response{}, err
		}
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		var env struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(raw, &env)
		last = labstack.Response{Status: resp.StatusCode, Header: resp.Header, Body: raw, Code: env.Error.Code}
		err = readErr
		if err != nil || last.Status != http.StatusTooManyRequests || !waitRetry(ctx, last.Header.Get("Retry-After")) {
			return last, err
		}
	}
	return last, err
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
	p.keep(list)
	var nodes api.NodeList
	valid = listErr == nil && list.Status == 200 && json.Unmarshal(list.Body, &nodes) == nil
	p.expect("AT06", "one_node", valid && len(nodes.Items) == 1 && nodes.Items[0].NodeId == one.enrollment.NodeId && nodes.NextCursor.IsNull(), map[string]any{"response": safeResponse(list, listErr), "node_count": len(nodes.Items)})
	p.current("AT06", one.enrollment.NodeId, h.incarnation)
	p.oneIncarnation("AT06", one.enrollment.NodeId, h.incarnation, one.enrollment.EnrollmentId)
	p.audit(one, 1)

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
		p.oneIncarnation("AT06", race.enrollment.NodeId, holders[winner].incarnation, race.enrollment.EnrollmentId)
	}
	p.audit(race, 1)

	expired, ok := p.issue("expiry", 60)
	if !ok {
		return
	}
	revoked, ok := p.issue("revocation", 900)
	if !ok {
		return
	}
	// The reserved Node exists before any enrollment is consumed.
	p.emptyIncarnations(revoked.enrollment.NodeId)
	resp, err = p.call("POST", "/api/v1/admin/enrollments/"+revoked.enrollment.EnrollmentId.String()+"/revoke", api.ReasonRequest{Reason: "TEST ONLY AT06"})
	p.keep(resp)
	var revokedEnrollment api.Enrollment
	revokedParsed := err == nil && resp.Status == 200 && json.Unmarshal(resp.Body, &revokedEnrollment) == nil
	revokedDigest := revokedEnrollment.TokenSha256 != nil && *revokedEnrollment.TokenSha256 == revoked.digest
	revokedPlain := bytes.Contains(resp.Body, []byte(revoked.token))
	if p.expect("AT06", "revoke_unused", revokedParsed, safeResponse(resp, err)) {
		p.expect("AT06", "token_sha256_revoke", revokedDigest && !revokedPlain && revokedEnrollment.State == api.EnrollmentStateRevoked, map[string]any{
			"digest": revoked.digest, "matches_revoke": revokedDigest, "plaintext_absent": !revokedPlain, "enrollment_id": revoked.enrollment.EnrollmentId, "state": revokedEnrollment.State,
		})
		p.readDigest(revoked.enrollment.EnrollmentId, revoked.digest, revoked.token)
		p.audit(revoked, 0)
		resp, err = post(p.ctx, p.client, url, revoked.token, p.holder(t).body)
		p.expect("AT06", "revoked_token", err == nil && resp.Status == 401 && resp.Code == "ENROLLMENT_TOKEN_REVOKED", safeResponse(resp, err))
	}
	p.incarnationHistory(t, one, h.incarnation)
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
	p.keep(r)
	digestOK := e.TokenSha256 != nil && *e.TokenSha256 == token.digest
	plain := bytes.Contains(r.Body, []byte(token.token))
	p.expect("AT06", "consumed_binding", valid && fpErr == nil && incErr == nil && e.State == "consumed" && fp == h.fingerprint && inc == h.incarnation, map[string]any{"response": safeResponse(r, err), "state": e.State, "bound_spki_sha256": fp, "incarnation_id": inc})
	p.expect("AT06", "token_sha256_get", valid && digestOK && !plain, map[string]any{"digest": token.digest, "matches_get": digestOK, "plaintext_absent": !plain, "enrollment_id": token.enrollment.EnrollmentId, "state": e.State})
}

func (p *probe) current(at string, node, incarnation uuid.UUID) api.NodeDetail {
	r, err := p.call("GET", "/api/v1/admin/nodes/"+node.String(), nil)
	p.keep(r)
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

func (p *probe) audit(token issued, expectConsumed int) {
	r, err := p.call("GET", "/api/v1/admin/audit-events?resource_type=enrollment&resource_id="+token.enrollment.EnrollmentId.String()+"&limit=100", nil)
	p.keep(r)
	var events api.AuditEventList
	if !p.expect("AT06", "audit_read", err == nil && r.Status == 200 && json.Unmarshal(r.Body, &events) == nil && events.NextCursor.IsNull(), safeResponse(r, err)) {
		return
	}
	consumed, createdOK := 0, false
	plain := bytes.Contains(r.Body, []byte(token.token))
	var stray []string
	var observed []map[string]any
	for _, event := range events.Items {
		if event.Action == "enrollment.consumed" {
			consumed++
		}
		after, afterErr := event.Changes.After.Get()
		before, _ := event.Changes.Before.Get()
		observed = append(observed, map[string]any{"action": event.Action, "result": event.Result,
			"resource": event.Resource, "audit_event_id": event.AuditEventId, "request_id": event.RequestId,
			"operation_id": event.OperationId, "before_fields": fieldNames(before), "after_fields": fieldNames(after)})
		_, has := after["token_sha256"]
		value, isString := after["token_sha256"].(string)
		if afterErr != nil {
			has, isString = false, false
		}
		if event.Action == "enrollment.created" {
			createdOK = has && isString && value == token.digest
		} else if has {
			stray = append(stray, event.Action)
		}
	}
	p.note("AT06", "enrollment_audit_fields", "pass", observed)
	if expectConsumed == 1 {
		p.expect("AT06", "single_consumption_audit", consumed == 1, map[string]any{"consumed_events": consumed, "resource_id": token.enrollment.EnrollmentId})
	}
	// Revision 10 (PR #195) puts the stored digest on enrollment.created only.
	p.expect("AT06", "audit_token_hash", createdOK && len(stray) == 0 && !plain, map[string]any{
		"api_hash_matches": createdOK, "field": "changes.after.token_sha256", "operation": "listAuditEvents",
		"digest": token.digest, "plaintext_absent": !plain, "digest_only_on_created": len(stray) == 0,
		"stray_actions": stray, "resource_id": token.enrollment.EnrollmentId,
		"changed_on_main": "contract revision 10 PR #195 Enrollment.token_sha256",
	})
}

func (p *probe) listIncarnations(node uuid.UUID, query string) (api.NodeIncarnationList, labstack.Response, error) {
	r, err := p.call("GET", "/api/v1/admin/nodes/"+node.String()+"/incarnations"+query, nil)
	p.keep(r)
	var page api.NodeIncarnationList
	if err != nil || r.Status != 200 {
		return page, r, err
	}
	if e := json.Unmarshal(r.Body, &page); e != nil {
		return page, r, e
	}
	return page, r, nil
}

func throughNull(v api.NodeIncarnationUsage) bool {
	return !v.Through.IsSpecified() || v.Through.IsNull()
}

func sameUsage(a, b api.NodeIncarnationUsage) bool {
	if a.UploadBytes != b.UploadBytes || a.DownloadBytes != b.DownloadBytes || throughNull(a) != throughNull(b) {
		return false
	}
	if throughNull(a) {
		return true
	}
	at, aerr := a.Through.Get()
	bt, berr := b.Through.Get()
	return aerr == nil && berr == nil && at.Equal(bt)
}

func usageView(u api.NodeIncarnationUsage) map[string]any {
	view := map[string]any{"upload_bytes": u.UploadBytes.String(), "download_bytes": u.DownloadBytes.String(), "through_null": throughNull(u)}
	if !throughNull(u) {
		if t, err := u.Through.Get(); err == nil {
			view["through"] = t.UTC().Format(time.RFC3339Nano)
		}
	}
	return view
}

func newestFirst(items []api.NodeIncarnation) bool {
	for i := 1; i < len(items); i++ {
		a, b := items[i-1], items[i]
		if a.CreatedAt.After(b.CreatedAt) {
			continue
		}
		if a.CreatedAt.Equal(b.CreatedAt) && bytes.Compare(a.IncarnationId[:], b.IncarnationId[:]) > 0 {
			continue
		}
		return false
	}
	return true
}

func (p *probe) oneIncarnation(at string, node, wantInc, wantEnrollment uuid.UUID) bool {
	page, r, err := p.listIncarnations(node, "")
	limited, limResp, limErr := p.listIncarnations(node, "?limit=1")
	nodeResp, nodeErr := p.call("GET", "/api/v1/admin/nodes/"+node.String(), nil)
	p.keep(nodeResp)
	var detail api.NodeDetail
	currentOK := nodeErr == nil && nodeResp.Status == 200 && json.Unmarshal(nodeResp.Body, &detail) == nil
	current, currentErr := detail.CurrentIncarnationId.Get()
	var item api.NodeIncarnation
	if len(page.Items) == 1 {
		item = page.Items[0]
	}
	enrResp, enrErr := p.call("GET", "/api/v1/admin/enrollments/"+item.EnrollmentId.String(), nil)
	p.keep(enrResp)
	var enrollment api.Enrollment
	enrOK := enrErr == nil && enrResp.Status == 200 && json.Unmarshal(enrResp.Body, &enrollment) == nil
	bound, boundErr := enrollment.NodeIncarnationId.Get()
	reasonNull := item.RetireReason.IsSpecified() && item.RetireReason.IsNull()
	retiredNull := item.RetiredAt.IsSpecified() && item.RetiredAt.IsNull()
	onePage := err == nil && r.Status == 200 && len(page.Items) == 1 && page.NextCursor.IsNull() && newestFirst(page.Items) && len(page.Items) <= 50
	limitPage := limErr == nil && limResp.Status == 200 && len(limited.Items) == 1 && limited.Items[0].IncarnationId == wantInc && limited.NextCursor.IsNull() && len(limited.Items) <= 100
	expectedEnrollment := wantEnrollment
	if expectedEnrollment == uuid.Nil {
		expectedEnrollment = item.EnrollmentId
	}
	ids := currentOK && currentErr == nil && current == wantInc && item.IncarnationId == wantInc && item.Current && item.State == api.NodeIncarnationStateActive && reasonNull && retiredNull
	link := enrOK && boundErr == nil && bound == wantInc && item.EnrollmentId == expectedEnrollment && enrollment.EnrollmentId == expectedEnrollment && expectedEnrollment != uuid.Nil
	return p.expect(at, "incarnation_uniqueness", onePage && limitPage && ids && link, map[string]any{
		"operation": "listNodeIncarnations", "node_id": node, "incarnation_id": item.IncarnationId,
		"enrollment_id": item.EnrollmentId, "count": len(page.Items), "current": item.Current, "state": item.State,
		"matches_get_node": current == wantInc, "matches_get_enrollment": bound == wantInc,
		"default_items_at_most_50": len(page.Items) <= 50, "limit_1_items": len(limited.Items),
		"next_cursor_null": page.NextCursor.IsNull() && limited.NextCursor.IsNull(), "usage": usageView(item.Usage),
		"changed_on_main": "contract revision 10 PR #200 listNodeIncarnations",
	})
}

func (p *probe) emptyIncarnations(node uuid.UUID) {
	page, r, err := p.listIncarnations(node, "")
	p.expect("AT06", "incarnation_history_empty", err == nil && r.Status == 200 && len(page.Items) == 0 && page.NextCursor.IsNull(), map[string]any{
		"operation": "listNodeIncarnations", "node_id": node, "count": len(page.Items), "http_status": r.Status,
	})
}

func (p *probe) incarnationHistory(t *testing.T, first issued, oldInc uuid.UUID) {
	t.Helper()
	node := first.enrollment.NodeId
	missing := uuid.New()
	missingResp, missingErr := p.call("GET", "/api/v1/admin/nodes/"+missing.String()+"/incarnations", nil)
	p.keep(missingResp)
	notFound := missingErr == nil && missingResp.Status == 404 && missingResp.Code == "RESOURCE_NOT_FOUND"

	before, beforeResp, beforeErr := p.listIncarnations(node, "")
	beforeOK := beforeErr == nil && beforeResp.Status == 200 && len(before.Items) == 1 && before.Items[0].IncarnationId == oldInc
	var usageBefore api.NodeIncarnationUsage
	if beforeOK {
		usageBefore = before.Items[0].Usage
	}
	name := "reinstall"
	ttl := 900
	blockedResp, blockedErr := p.call("POST", "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name, ExpiresInSeconds: &ttl, NodeId: &node})
	p.keep(blockedResp)
	blocked := blockedErr == nil && blockedResp.Status == 409 && blockedResp.Code == "NODE_INCARNATION_ACTIVE"

	revResp, revErr := p.revokeNode(node)
	revoked := revErr == nil && revResp.Status == 202
	mid, midResp, midErr := p.listIncarnations(node, "")
	var midItem api.NodeIncarnation
	if len(mid.Items) == 1 {
		midItem = mid.Items[0]
	}
	midReason, midReasonErr := midItem.RetireReason.Get()
	midOK := revoked && midErr == nil && midResp.Status == 200 && len(mid.Items) == 1 && midItem.IncarnationId == oldInc &&
		midItem.State == api.NodeIncarnationStateRetired && midReasonErr == nil && midReason == api.NodeIncarnationRetireReasonRevocation &&
		sameUsage(midItem.Usage, usageBefore)

	second, issuedOK := p.issueOn("reinstall", 900, &node)
	if !issuedOK {
		p.expect("AT06", "incarnation_history", false, map[string]any{
			"unknown_node_404": notFound, "active_blocks_reinstall": blocked, "node_revoked": revoked,
			"retired_before_replacement": midOK, "reinstall_issued": false, "node_id": node,
		})
		return
	}
	h := p.holder(t)
	resp, err := post(p.ctx, p.client, p.stack.PublicURL()+"/api/v1/agent/enroll", second.token, h.body)
	var result api.EnrollResponse
	enrolled := err == nil && resp.Status == 201 && json.Unmarshal(resp.Body, &result) == nil && result.NodeId == node && result.NodeIncarnationId == h.incarnation && result.NodeIncarnationId != oldInc
	if enrolled {
		p.binding(second, h)
		p.audit(second, 1)
	}
	page, pageResp, pageErr := p.listIncarnations(node, "")
	wide, wideResp, wideErr := p.listIncarnations(node, "?limit=100")
	head, headResp, headErr := p.listIncarnations(node, "?limit=1")
	cursorRound, duplicates := false, false
	var tail api.NodeIncarnationList
	var tailResp labstack.Response
	var tailErr error
	if headErr == nil && headResp.Status == 200 && len(head.Items) == 1 && !head.NextCursor.IsNull() {
		cursor, cursorErr := head.NextCursor.Get()
		if cursorErr == nil && cursor != "" && len(cursor) <= 512 {
			tail, tailResp, tailErr = p.listIncarnations(node, "?limit=1&cursor="+url.QueryEscape(cursor))
			seen := map[uuid.UUID]int{}
			for _, item := range append(append([]api.NodeIncarnation{}, head.Items...), tail.Items...) {
				seen[item.IncarnationId]++
			}
			duplicates = false
			for _, n := range seen {
				if n != 1 {
					duplicates = true
				}
			}
			cursorRound = tailErr == nil && tailResp.Status == 200 && len(tail.Items) == 1 && tail.NextCursor.IsNull() && !duplicates && len(seen) == 2
		}
	}
	var fresh, previous api.NodeIncarnation
	if len(page.Items) == 2 {
		fresh, previous = page.Items[0], page.Items[1]
	}
	prevReason, prevReasonErr := previous.RetireReason.Get()
	freshReasonNull := fresh.RetireReason.IsSpecified() && fresh.RetireReason.IsNull()
	history := pageErr == nil && pageResp.Status == 200 && len(page.Items) == 2 && page.NextCursor.IsNull() && len(page.Items) <= 50 && newestFirst(page.Items) &&
		fresh.IncarnationId == h.incarnation && fresh.Current && fresh.State == api.NodeIncarnationStateActive && freshReasonNull && fresh.EnrollmentId == second.enrollment.EnrollmentId &&
		previous.IncarnationId == oldInc && !previous.Current && previous.State == api.NodeIncarnationStateRetired && prevReasonErr == nil && prevReason == api.NodeIncarnationRetireReasonReinstall &&
		sameUsage(previous.Usage, usageBefore)
	wideOK := wideErr == nil && wideResp.Status == 200 && len(wide.Items) == 2 && len(wide.Items) <= 100
	p.expect("AT06", "incarnation_history", notFound && blocked && midOK && enrolled && history && wideOK && cursorRound, map[string]any{
		"operation": "listNodeIncarnations", "node_id": node, "unknown_node_404": notFound,
		"active_blocks_reinstall": blocked, "revoke_http_status": revResp.Status,
		"retired_reason_before_replacement": string(midReason), "usage_before": usageView(usageBefore),
		"usage_after_retirement": usageView(previous.Usage), "usage_unchanged": sameUsage(previous.Usage, usageBefore),
		"count": len(page.Items), "newest_incarnation_id": fresh.IncarnationId, "newest_current": fresh.Current,
		"previous_incarnation_id": previous.IncarnationId, "previous_state": previous.State, "previous_retire_reason": string(prevReason),
		"default_items_at_most_50": len(page.Items) <= 50, "limit_100_items": len(wide.Items),
		"cursor_round_trip_no_duplicates": cursorRound, "enroll_response": safeResponse(resp, err),
		"changed_on_main": "contract revision 10 PR #200 listNodeIncarnations; reinstall is createEnrollment node_id after revokeNode",
	})
}

func (p *probe) revokeNode(node uuid.UUID) (labstack.Response, error) {
	var last labstack.Response
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		got, gerr := p.call("GET", "/api/v1/admin/nodes/"+node.String(), nil)
		p.keep(got)
		if gerr != nil || got.Status != 200 {
			return got, gerr
		}
		last, err = p.callHeaders("POST", "/api/v1/admin/nodes/"+node.String()+"/revoke", api.ReasonRequest{Reason: "TEST ONLY AT06 reinstall"}, map[string]string{"If-Match": got.Header.Get("ETag")})
		p.keep(last)
		if err != nil {
			continue
		}
		if last.Status == 412 && attempt == 0 {
			continue
		}
		return last, nil
	}
	return last, err
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
	firstOne := p.oneIncarnation("AT07", first.NodeID, first.IncarnationID, uuid.Nil)
	secondOne := p.oneIncarnation("AT07", second.NodeID, second.IncarnationID, uuid.Nil)
	// These live nodes are not reinstalled here. Retirement, retained usage and
	// cursor behavior are measured on the AT06 node through the public
	// revokeNode + node_id enrollment + agent enroll path. Counted usage and
	// the real installer rerun still need devstack follow-up 13.
	p.expect("AT07", "incarnation_history", firstOne && secondOne, map[string]any{
		"operation": "listNodeIncarnations", "node_ids": []uuid.UUID{first.NodeID, second.NodeID},
		"each_count": 1, "retirement_measured_on": "AT06 incarnation_history",
		"changed_on_main": "contract revision 10 PR #200 listNodeIncarnations",
	})
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
	p.machineIdentity()
	// The clone fixture stops and resumes node. node2 stays the clean second exit.
	p.clonedIdentity()
	// PauseAgent resumes durable identity; it is not an installer rerun or a
	// usage-ACK hold. No unsupported mutation is substituted here.
}

func hex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func fingerprintView(f e2e.IdentityFingerprints) map[string]string {
	return map[string]string{
		"machine_id_sha256": f.MachineID, "agent_spki_sha256": f.AgentSPKI, "netbird_public_key_sha256": f.NetBirdPublicKey,
	}
}

func fingerprintsComplete(f e2e.IdentityFingerprints) bool {
	return hex64(f.MachineID) && hex64(f.AgentSPKI) && hex64(f.NetBirdPublicKey)
}

func (p *probe) machineIdentity() {
	first, err1 := p.stack.MachineIdentityFingerprints(p.ctx, "node")
	second, err2 := p.stack.MachineIdentityFingerprints(p.ctx, "node2")
	ok := err1 == nil && err2 == nil && fingerprintsComplete(first) && fingerprintsComplete(second)
	distinct := ok && first.MachineID != second.MachineID && first.AgentSPKI != second.AgentSPKI && first.NetBirdPublicKey != second.NetBirdPublicKey
	detail := map[string]any{
		"changed_on_main": "devstack follow-up 12 PR #204 MachineIdentityFingerprints",
		"node":            fingerprintView(first), "node2": fingerprintView(second), "reads_ok": err1 == nil && err2 == nil,
	}
	p.expect("AT07", "machine_identity", ok, detail)
	p.expect("AT07", "distinct_first_boot", distinct, detail)
}

func (p *probe) clonedIdentity() {
	const changed = "devstack follow-up 12 PR #204 StartClonedIdentityFixture"
	fixture, err := p.stack.StartClonedIdentityFixture(p.ctx, "node")
	if err != nil || fixture == nil {
		p.expect("AT07", "cloned_identity_conflict", false, map[string]any{"started": false, "changed_on_main": changed})
		return
	}
	read := func() (e2e.CloneObservation, bool) {
		obs, obsErr := fixture.Observe(p.ctx)
		return obs, obsErr == nil
	}
	var obs e2e.CloneObservation
	ok := false
	deadline := time.Now().Add(75 * time.Second)
	for {
		obs, ok = read()
		if ok && obs.SameIdentity && obs.SourceStage == "enrolled" && obs.CloneStage == "enrolled" {
			break
		}
		if time.Now().After(deadline) || p.ctx.Err() != nil || !waitRetry(p.ctx, "1") {
			break
		}
	}
	view := func(o e2e.CloneObservation) map[string]any {
		return map[string]any{
			"started": true, "read_ok": ok, "same_identity": o.SameIdentity,
			"source_stage": o.SourceStage, "clone_stage": o.CloneStage,
			"source": fingerprintView(o.Source), "clone": fingerprintView(o.Clone),
			"endpoint_rows": o.EndpointRows, "conflicts": o.Conflicts, "mapping_frozen": o.MappingFrozen,
			"changed_on_main": changed,
		}
	}
	if !ok || !obs.SameIdentity || obs.SourceStage != "enrolled" || obs.CloneStage != "enrolled" {
		p.expect("AT07", "cloned_identity_conflict", false, view(obs))
		return
	}
	freeze := time.Now().Add(65 * time.Second)
	for obs.Conflicts == 0 || !obs.MappingFrozen {
		if time.Now().After(freeze) || p.ctx.Err() != nil || !waitRetry(p.ctx, "2") {
			break
		}
		if next, good := read(); good {
			obs, ok = next, true
		}
	}
	detail := view(obs)
	// The export's own live record leaves this blocked: ObserveEndpoint has no
	// production caller, so zero conflict rows are a product gap, not a missed hook.
	if obs.Conflicts > 0 && obs.MappingFrozen {
		p.expect("AT07", "cloned_identity_conflict", true, detail)
		return
	}
	detail["owner"] = "PKI/enrollment + node identity adapter"
	detail["request"] = "ENR-08"
	detail["gap"] = "StartClonedIdentityFixture.Observe read node_identity_conflicts and mapping_frozen; both stayed unset"
	p.note("AT07", "cloned_identity_conflict", "blocked", detail)
}

func (p *probe) peerUniqueness(t *testing.T) {
	t.Helper()
	const changed = "devstack follow-up 12 PR #204 JoinEnrolledIdentity and EnrollmentPeerInventory"
	fail := func(step string) {
		p.expect("AT06", "peer_uniqueness", false, map[string]any{"step": step, "changed_on_main": changed})
	}
	if err := p.stack.StartSecondExit(p.ctx); err != nil {
		fail("start_second_exit")
		return
	}
	one, issued := p.issue("join", 900)
	if !issued {
		fail("issue")
		return
	}
	a, errA := p.stack.GenerateEnrollmentIdentity(p.ctx, "node2", "key-a")
	b, errB := p.stack.GenerateEnrollmentIdentity(p.ctx, "node2", "key-b")
	if errA != nil || errB != nil {
		fail("generate")
		return
	}
	if p.stack.StageEnrollmentToken(p.ctx, "node2", a, one.token) != nil || p.stack.StageEnrollmentToken(p.ctx, "node2", b, one.token) != nil {
		fail("stage_token")
		return
	}
	type attempt struct {
		in     e2e.EnrolledIdentityInput
		result e2e.EnrolledIdentityResult
		err    error
	}
	done := make(chan attempt, 2)
	for _, in := range []e2e.EnrolledIdentityInput{a, b} {
		go func(in e2e.EnrolledIdentityInput) {
			in.DiscardResponse = true
			in.Join = false
			out, e := p.stack.JoinEnrolledIdentity(p.ctx, "node2", in)
			done <- attempt{in, out, e}
		}(in)
	}
	var winner attempt
	won, lost, broken := 0, 0, 0
	race := make([]map[string]any, 0, 2)
	for range 2 {
		v := <-done
		race = append(race, map[string]any{
			"http_status": v.result.EnrollHTTP, "error_code": v.result.ErrorCode,
			"request_sha256": v.result.RequestSHA256, "discarded": v.result.ResponseDiscarded, "transport_error": v.err != nil,
		})
		if v.err != nil {
			broken++
			continue
		}
		if v.result.EnrollHTTP == 201 || v.result.EnrollHTTP == 200 {
			won++
			winner = v
		} else if v.result.EnrollHTTP == 409 {
			lost++
		}
	}
	if broken != 0 || won != 1 || lost != 1 {
		p.expect("AT06", "peer_uniqueness", false, map[string]any{"step": "race", "accepted": won, "rejected": lost, "transport_errors": broken, "race": race, "changed_on_main": changed})
		return
	}
	winner.in.DiscardResponse = false
	winner.in.Join = true
	replay, err := p.stack.JoinEnrolledIdentity(p.ctx, "node2", winner.in)
	if err != nil || !replay.Joined || replay.RequestSHA256 != winner.result.RequestSHA256 || replay.EnrollHTTP != 200 {
		p.expect("AT06", "peer_uniqueness", false, map[string]any{
			"step": "replay_join", "race": race, "enroll_http": replay.EnrollHTTP, "bootstrap_http": replay.BootstrapHTTP,
			"error_code": replay.ErrorCode, "joined": replay.Joined, "request_matches": replay.RequestSHA256 == winner.result.RequestSHA256,
			"changed_on_main": changed,
		})
		return
	}
	wait, cancel := context.WithTimeout(p.ctx, 90*time.Second)
	defer cancel()
	var inventory e2e.EnrollmentPeers
	var invErr error
	for {
		inventory, invErr = p.stack.EnrollmentPeerInventory(wait, p.admin, one.enrollment.EnrollmentId)
		if invErr == nil && inventory.Complete && inventory.Peers == 1 {
			break
		}
		if wait.Err() != nil || !waitRetry(wait, "1") {
			break
		}
	}
	again, againErr := p.stack.EnrollmentPeerInventory(p.ctx, p.admin, one.enrollment.EnrollmentId)
	same := againErr == nil && again.Peers == 1 && again.SetupKeyUses == 1 && again.IncarnationID == inventory.IncarnationID && again.Complete
	pass := invErr == nil && inventory.Complete && inventory.Peers == 1 && inventory.SetupKeys == 1 && inventory.SetupKeyUses == 1 &&
		inventory.ProjectedPeerPresent && inventory.IncarnationID == replay.IncarnationID && inventory.NodeID == replay.NodeID && same
	p.expect("AT06", "peer_uniqueness", pass, map[string]any{
		"changed_on_main": changed, "operation": "EnrollmentPeerInventory",
		"enrollment_id": one.enrollment.EnrollmentId, "node_id": replay.NodeID, "incarnation_id": replay.IncarnationID,
		"enroll_http": replay.EnrollHTTP, "bootstrap_http": replay.BootstrapHTTP, "joined": replay.Joined,
		"race": race, "peers": inventory.Peers, "setup_keys": inventory.SetupKeys, "setup_key_uses": inventory.SetupKeyUses,
		"correlation_groups": inventory.CorrelationGroups, "projected_peer_present": inventory.ProjectedPeerPresent,
		"complete": inventory.Complete, "enrollment_state": inventory.EnrollmentState,
		"second_read_peers": again.Peers, "second_read_setup_key_uses": again.SetupKeyUses, "second_read_complete": again.Complete,
		"idempotent_one_peer": same,
	})
}

func (p *probe) nodeAudit(node uuid.UUID) {
	r, err := p.call("GET", "/api/v1/admin/audit-events?resource_type=node&resource_id="+node.String()+"&limit=100", nil)
	p.keep(r)
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

func (p *probe) countSensitive(raw []byte) (hits, digestHits, derived int) {
	for _, secret := range p.secrets {
		if len(secret) >= 16 {
			hits += bytes.Count(raw, secret)
		}
	}
	hits += bytes.Count(raw, []byte("PRIVATE KEY-----"))
	for _, digest := range p.digests {
		digestHits += bytes.Count(raw, digest)
	}
	for _, other := range p.derived {
		if len(other) >= 16 {
			derived += bytes.Count(raw, other)
		}
	}
	return hits, digestHits, derived
}

func (p *probe) scanAdminReads() {
	raw := bytes.Join(p.reads, nil)
	hits, digestHits, derived := p.countSensitive(raw)
	p.expect("AT06", "secret_scan_admin_reads", len(p.reads) > 0 && hits == 0 && derived == 0 && digestHits > 0, map[string]any{
		"bodies": len(p.reads), "bytes_scanned": len(raw), "plaintext_or_key_matches": hits,
		"digest_occurrences": digestHits, "other_token_derived": derived,
	})
}

func (p *probe) scanLogs(strict bool) {
	for _, service := range []string{"control", "worker", "node", "node2"} {
		if service == "node2" {
			if _, err := os.Stat(filepath.Join(p.stack.StateDir(), "node2")); os.IsNotExist(err) {
				continue
			}
		}
		raw, err := exec.CommandContext(p.ctx, "docker", "logs", "fvpn-w4-enroll-"+service).CombinedOutput()
		if err != nil && !strict {
			continue
		}
		p.scanStream(service+"/console", raw, err)
		if service == "node" || service == "node2" {
			for _, file := range []string{"agent.log", "watchdog.log", "netbird-service.log", "netbird-up.log", "resolver-proxy.log"} {
				raw, err = exec.CommandContext(p.ctx, "docker", "exec", "fvpn-w4-enroll-"+service, "cat", "/var/lib/vpn-node/e2e/"+file).Output()
				// The join cycle never reaches the entrypoint's post-enroll logs.
				// A missing file there is not a zero-hit result and is not one of
				// the acceptance stack's required streams.
				if err != nil && !strict {
					continue
				}
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
	hits, digestHits, derived := p.countSensitive(raw)
	// The lowercase SHA-256 digest is allowed. Plaintext and every other
	// token-derived encoding stay negative patterns.
	p.expect("AT06", "secret_scan_"+name, hits == 0 && derived == 0, map[string]any{
		"bytes_scanned": len(raw), "matches": hits + derived, "needles": len(p.secrets),
		"digest_occurrences": digestHits, "other_token_derived": derived,
	})
}
