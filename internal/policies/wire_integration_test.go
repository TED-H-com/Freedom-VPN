//go:build integration

package policies_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/TED-H-com/Freedom-VPN/apps/control"
	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	membersruntime "github.com/TED-H-com/Freedom-VPN/internal/members/runtime"
	"github.com/TED-H-com/Freedom-VPN/internal/policies/policiestest"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/internal/usage/usagetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

type httpStack struct {
	public http.Handler
	agent  http.Handler
	adminP auth.Principal
	memP   auth.Principal
	agentP auth.Principal
	disp   *worker.Application
}

func newHTTPStack(t *testing.T, pool *pgxpool.Pool, adminID, memberID, nodeID, incID uuid.UUID) *httpStack {
	t.Helper()
	c := config.Defaults()
	c.PublicOrigin = "https://portal.example.invalid"
	c.BrowserRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	c.AgentRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	c.IngressRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	c.WorkerAttempts = 3
	c.PoliciesConfigFile = policiestest.New(t).Path
	c.UsageConfigFile = usagetest.New(t).Path

	deps := control.Dependencies{Config: c, Pool: pool}
	var handlerSets []any
	for _, dom := range control.Domains() {
		if dom.Name != "control-core" && dom.Name != "policies" && dom.Name != "usage" {
			continue
		}
		built, err := dom.Build(t.Context(), deps)
		if err != nil {
			t.Fatalf("build control domain %s: %v", dom.Name, err)
		}
		handlerSets = append(handlerSets, built.Handlers...)
	}
	handlers, err := httpapi.Compose(handlerSets...)
	if err != nil {
		t.Fatalf("httpapi.Compose: %v", err)
	}

	adminP := auth.Principal{
		Role:            auth.Admin,
		ID:              adminID,
		MFAVerified:     true,
		CSRFToken:       "TEST-ONLY-ADMIN-CSRF",
		AuthenticatedAt: time.Now().UTC(),
	}
	memP := auth.Principal{
		Role:            auth.Member,
		ID:              memberID,
		CSRFToken:       "TEST-ONLY-MEMBER-CSRF",
		AuthenticatedAt: time.Now().UTC(),
	}
	agentP := auth.Principal{
		Role:          auth.Agent,
		NodeID:        nodeID,
		IncarnationID: incID,
	}

	commands := &httpapi.Commands{Pool: pool, Retention: time.Hour}
	pubOpts := httpapi.Options{
		Config: c,
		Sessions: auth.SessionAuthenticatorFunc(func(r *http.Request) (auth.Principal, error) {
			if strings.HasPrefix(r.URL.Path, "/api/v1/member/") {
				return memP, nil
			}
			return adminP, nil
		}),
		Commands: commands,
	}
	pubAPI, err := httpapi.New(handlers, pubOpts)
	if err != nil {
		t.Fatalf("httpapi.New public: %v", err)
	}

	agentOpts := pubOpts
	agentOpts.AgentListener = true
	agentOpts.Agents = auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) {
		return agentP, nil
	})
	agentAPI, err := httpapi.New(handlers, agentOpts)
	if err != nil {
		t.Fatalf("httpapi.New agent: %v", err)
	}

	disp, err := worker.Open(
		t.Context(),
		worker.Dependencies{Config: c, Pool: pool},
		worker.NewPoliciesDomain(nil, nil),
		worker.NewUsageDomain(nil),
	)
	if err != nil {
		t.Fatalf("worker.Open: %v", err)
	}

	return &httpStack{
		public: httpapi.Transport(nil, pubAPI),
		agent:  httpapi.Transport(nil, agentAPI),
		adminP: adminP,
		memP:   memP,
		agentP: agentP,
		disp:   disp,
	}
}

func (s *httpStack) doAdmin(t *testing.T, method, path, body, idemKey, ifMatch string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		if idemKey == "" {
			idemKey = uuid.NewString()
		}
		if idemKey != "OMIT" {
			r.Header.Set("Idempotency-Key", idemKey)
		}
		r.Header.Set("Origin", "https://portal.example.invalid")
		r.Header.Set("X-CSRF-Token", s.adminP.CSRFToken)
	}
	if ifMatch != "" {
		r.Header.Set("If-Match", ifMatch)
	}
	w := httptest.NewRecorder()
	s.public.ServeHTTP(w, r)
	return w
}

func (s *httpStack) doMember(t *testing.T, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	s.public.ServeHTTP(w, r)
	return w
}

func (s *httpStack) doAgent(t *testing.T, method, path, body, idemKey string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if idemKey != "" {
		r.Header.Set("Idempotency-Key", idemKey)
	}
	w := httptest.NewRecorder()
	s.agent.ServeHTTP(w, r)
	return w
}

func extractRevisionAndDefinitionIDs(t *testing.T, raw []byte) (revID, defID uuid.UUID) {
	t.Helper()
	var m struct {
		RevisionID   uuid.UUID `json:"revision_id"`
		DefinitionID uuid.UUID `json:"definition_id"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal revision response: %v (body=%s)", err, string(raw))
	}
	return m.RevisionID, m.DefinitionID
}

func TestPoliciesHTTP_AllSixPolicyTypesActivateIdempotencyWorkerAndAudit(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	adminID := uuid.New()
	memberID := uuid.New()
	nodeID := uuid.New()
	incID := uuid.New()

	st := newHTTPStack(t, pool, adminID, memberID, nodeID, incID)
	now := time.Now().UTC()

	// Seed member_groups and releases for plan and release_support validation.
	exitGroupID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups (id, code, name) VALUES ($1, 'tw-std', 'Taiwan Standard')`, exitGroupID); err != nil {
		t.Fatalf("insert member_groups: %v", err)
	}
	rel1, rel2 := uuid.New(), uuid.New()
	for i, rid := range []uuid.UUID{rel1, rel2} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO releases (id, version, release_sequence, manifest_schema_version, manifest, manifest_hash, signature, signing_key_ref, component_versions)
			 VALUES ($1, $2, $3, 1, '{}'::jsonb, decode(repeat('01', 32), 'hex'), decode(repeat('02', 64), 'hex'), 'test-key', '{}'::jsonb)`,
			rid, "TEST-ONLY-"+rid.String()[:8], i+1,
		); err != nil {
			t.Fatalf("insert releases: %v", err)
		}
	}

	createBody := func(code string, pType api.PolicyType, payload any, reason string) string {
		b, err := json.Marshal(map[string]any{
			"code":           code,
			"type":           string(pType),
			"schema_version": 1,
			"payload":        payload,
			"reason":         reason,
		})
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	// 1. domain_rule_set
	rsPayload := api.DomainRuleSetPayload{
		Rules: []api.DomainRule{
			{
				RuleId:            openapi_types.UUID(uuid.New()),
				ProfileId:         presence.NewNullableWithValue("standard"),
				Domain:            "ads.example.com",
				IncludeSubdomains: true,
				Action:            api.DomainRuleActionBlock,
				Reason:            "admin_rule",
				Revision:          decstr.U64(1),
			},
		},
	}
	idemRS := uuid.NewString()
	rsBody := createBody("http-rules", api.PolicyTypeDomainRuleSet, rsPayload, "HTTP create domain_rule_set")
	wRS := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", rsBody, idemRS, "")
	if wRS.Code != http.StatusCreated {
		t.Fatalf("create domain_rule_set status=%d body=%s", wRS.Code, wRS.Body.String())
	}
	rsRevID, rsDefID := extractRevisionAndDefinitionIDs(t, wRS.Body.Bytes())

	// Idempotent replay with same key and payload returns 201 with same revision_id.
	wRSReplay := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", rsBody, idemRS, "")
	if wRSReplay.Code != http.StatusCreated {
		t.Fatalf("replay domain_rule_set status=%d body=%s", wRSReplay.Code, wRSReplay.Body.String())
	}
	rsReplayRevID, _ := extractRevisionAndDefinitionIDs(t, wRSReplay.Body.Bytes())
	if rsReplayRevID != rsRevID {
		t.Fatalf("replay revision_id=%s want=%s", rsReplayRevID, rsRevID)
	}

	// Same key with different payload returns 409 IDEMPOTENCY_CONFLICT.
	rsBodyDiff := createBody("http-rules", api.PolicyTypeDomainRuleSet, rsPayload, "Different reason")
	wRSConflict := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", rsBodyDiff, idemRS, "")
	if wRSConflict.Code != http.StatusConflict {
		t.Fatalf("expected 409 IDEMPOTENCY_CONFLICT, got %d body=%s", wRSConflict.Code, wRSConflict.Body.String())
	}

	// 2. threat_feed
	tfPayload := api.ThreatFeedPayload{
		Provider:              "Example Threat Intel",
		FeedCode:              "http-malware",
		LicenseName:           "Commercial Threat Feed License",
		LicenseUrl:            "https://feeds.example.com/license",
		CommercialUseBasis:    "Licensed for commercial SaaS/VPN redistribution (TEST ONLY).",
		Attribution:           presence.FromPointer(strPtr("Threat intel by Example")),
		SourceUrl:             "https://feeds.example.com/malware.txt",
		Format:                api.ThreatFeedPayloadFormatDomainList,
		Categories:            []api.ThreatFeedPayloadCategories{api.ThreatFeedPayloadCategoriesMalware},
		ContentSha256:         "7777777777777777777777777777777777777777777777777777777777777777",
		MaxSizeBytes:          decstr.U64(1048576),
		ValidFrom:             now.Add(-1 * time.Hour),
		ValidUntil:            now.Add(30 * 24 * time.Hour),
		UpdateIntervalSeconds: 3600,
	}
	wTF := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", createBody("http-malware", api.PolicyTypeThreatFeed, tfPayload, "HTTP create threat_feed"), "", "")
	if wTF.Code != http.StatusCreated {
		t.Fatalf("create threat_feed status=%d body=%s", wTF.Code, wTF.Body.String())
	}
	tfRevID, tfDefID := extractRevisionAndDefinitionIDs(t, wTF.Body.Bytes())

	// 3. dns_profile
	dnsPayload := api.DnsProfilePayload{
		ProfileCode:              "standard",
		DisplayName:              api.LocalizedText{ZhTW: "標準防護", En: strPtr("Standard")},
		DomainRuleSetRevisionIds: []openapi_types.UUID{openapi_types.UUID(rsRevID)},
		ThreatFeedRevisionIds:    []openapi_types.UUID{openapi_types.UUID(tfRevID)},
		BlockResponse:            api.DnsProfilePayloadBlockResponseNxdomain,
		BlockedTtlSeconds:        60,
	}
	wDNS := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", createBody("standard", api.PolicyTypeDnsProfile, dnsPayload, "HTTP create dns_profile"), "", "")
	if wDNS.Code != http.StatusCreated {
		t.Fatalf("create dns_profile status=%d body=%s", wDNS.Code, wDNS.Body.String())
	}
	dnsRevID, dnsDefID := extractRevisionAndDefinitionIDs(t, wDNS.Body.Bytes())

	// 4. fair_use
	fupPayload := validFairUsePayload()
	wFUP := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", createBody("http-fup", api.PolicyTypeFairUse, fupPayload, "HTTP create fair_use"), "", "")
	if wFUP.Code != http.StatusCreated {
		t.Fatalf("create fair_use status=%d body=%s", wFUP.Code, wFUP.Body.String())
	}
	fupRevID, fupDefID := extractRevisionAndDefinitionIDs(t, wFUP.Body.Bytes())

	// 5. plan
	planPayload := api.PlanPayload{
		DisplayName:  api.LocalizedText{ZhTW: "標準方案", En: strPtr("Standard")},
		DeviceLimit:  3,
		ExitGroupIds: []openapi_types.UUID{openapi_types.UUID(exitGroupID)},
		Price: api.PlanPrice{
			AmountMinor:      30000, // TEST ONLY
			Currency:         "TWD",
			BillingInterval:  api.PlanPriceBillingIntervalMonth,
			Provider:         "test-provider",
			ProviderPriceRef: "price_TEST_ONLY_http_std",
		},
		RateLimit:            presence.NewNullNullable[api.RatePair](),
		FairUseRevisionId:    openapi_types.UUID(fupRevID),
		DnsProfileRevisionId: openapi_types.UUID(dnsRevID),
	}
	wPlan := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", createBody("http-plan", api.PolicyTypePlan, planPayload, "HTTP create plan"), "", "")
	if wPlan.Code != http.StatusCreated {
		t.Fatalf("create plan status=%d body=%s", wPlan.Code, wPlan.Body.String())
	}
	planRevID, planDefID := extractRevisionAndDefinitionIDs(t, wPlan.Body.Bytes())

	// 6. release_support
	relPayload := api.ReleaseSupportPayload{
		ApprovedReleaseIds: []openapi_types.UUID{openapi_types.UUID(rel2)},
		RevokedReleaseIds:  []openapi_types.UUID{},
		MinimumServingVersion: struct {
			ServingReleaseIds []openapi_types.UUID `json:"serving_release_ids"`
		}{
			ServingReleaseIds: []openapi_types.UUID{openapi_types.UUID(rel1), openapi_types.UUID(rel2)},
		},
	}
	wRel := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", createBody("http-rel-support", api.PolicyTypeReleaseSupport, relPayload, "HTTP create release_support"), "", "")
	if wRel.Code != http.StatusCreated {
		t.Fatalf("create release_support status=%d body=%s", wRel.Code, wRel.Body.String())
	}
	relRevID, relDefID := extractRevisionAndDefinitionIDs(t, wRel.Body.Bytes())

	// Test If-Match preconditions on ActivatePolicyRevision:
	// - Missing If-Match -> 428
	// - Stale If-Match -> 412
	actBodyFn := func(revID uuid.UUID) string {
		b, _ := json.Marshal(api.PolicyActivateRequest{
			RevisionId: openapi_types.UUID(revID),
			Reason:     "HTTP activate revision",
		})
		return string(b)
	}
	w428 := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies/"+rsDefID.String()+"/activate", actBodyFn(rsRevID), "", "")
	if w428.Code != http.StatusPreconditionRequired {
		t.Fatalf("expected 428 on missing If-Match, got %d body=%s", w428.Code, w428.Body.String())
	}
	w412 := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies/"+rsDefID.String()+"/activate", actBodyFn(rsRevID), "", `"99"`)
	if w412.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected 412 on stale If-Match, got %d body=%s", w412.Code, w412.Body.String())
	}

	// Activate all 6 definitions with If-Match: "1".
	var opIDs []uuid.UUID
	for _, pair := range []struct {
		defID uuid.UUID
		revID uuid.UUID
	}{
		{rsDefID, rsRevID},
		{tfDefID, tfRevID},
		{dnsDefID, dnsRevID},
		{fupDefID, fupRevID},
		{planDefID, planRevID},
		{relDefID, relRevID},
	} {
		idemAct := uuid.NewString()
		actPath := "/api/v1/admin/policies/" + pair.defID.String() + "/activate"
		wAct := st.doAdmin(t, http.MethodPost, actPath, actBodyFn(pair.revID), idemAct, `"1"`)
		if wAct.Code != http.StatusAccepted {
			t.Fatalf("activate %s status=%d body=%s", pair.defID, wAct.Code, wAct.Body.String())
		}
		var op api.Operation
		if err := json.Unmarshal(wAct.Body.Bytes(), &op); err != nil {
			t.Fatalf("unmarshal operation: %v", err)
		}
		if op.Status != api.OperationStatusPending {
			t.Fatalf("expected pending operation on 202 response, got %s", op.Status)
		}

		// Replay same Idempotency-Key returns 202 with same operation_id.
		wActReplay := st.doAdmin(t, http.MethodPost, actPath, actBodyFn(pair.revID), idemAct, `"1"`)
		if wActReplay.Code != http.StatusAccepted {
			t.Fatalf("replay activate status=%d body=%s", wActReplay.Code, wActReplay.Body.String())
		}
		var opReplay api.Operation
		if err := json.Unmarshal(wActReplay.Body.Bytes(), &opReplay); err != nil {
			t.Fatalf("unmarshal replay operation: %v", err)
		}
		if opReplay.OperationId != op.OperationId {
			t.Fatalf("replay operation_id=%s want=%s", opReplay.OperationId, op.OperationId)
		}
		opIDs = append(opIDs, uuid.UUID(op.OperationId))
	}

	// Verify release_support activation ran releases.ApplySupportTx and updated fleet_config.
	var policyMode string
	var supportedIDs, approvedIDs []uuid.UUID
	var supportGen string
	if err := pool.QueryRow(ctx, `SELECT supported_release_policy, supported_release_ids, approved_release_ids, support_generation::text FROM fleet_config WHERE id = '00000000-0000-0000-0000-000000000001'`).Scan(&policyMode, &supportedIDs, &approvedIDs, &supportGen); err != nil {
		t.Fatalf("query fleet_config: %v", err)
	}
	if policyMode != "allowlist" || len(supportedIDs) != 2 || len(approvedIDs) != 1 || approvedIDs[0] != rel2 || supportGen != "1" {
		t.Fatalf("expected fleet_config updated from ApplySupportTx, got policy=%s supported=%v approved=%v gen=%s", policyMode, supportedIDs, approvedIDs, supportGen)
	}

	// Verify GET /api/v1/admin/policies returns all 6 active definitions.
	wList := st.doAdmin(t, http.MethodGet, "/api/v1/admin/policies?limit=10", "", "", "")
	if wList.Code != http.StatusOK {
		t.Fatalf("GET /admin/policies status=%d body=%s", wList.Code, wList.Body.String())
	}
	var list api.PolicyDefinitionList
	if err := json.Unmarshal(wList.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal PolicyDefinitionList: %v", err)
	}
	if len(list.Items) != 6 {
		t.Fatalf("expected 6 policy definitions, got %d", len(list.Items))
	}

	for _, definition := range list.Items {
		if definition.Type != api.PolicyTypeThreatFeed && !definition.ThreatFeedStatus.IsNull() {
			t.Fatalf("non-feed %s has a refresh status", definition.Type)
		}
	}

	// Run worker dispatcher to process all outbox events (6 policy.activated + 1 fleet.release_support_changed)
	// and transition operations to succeeded.
	for i := 0; i < 10; i++ {
		worked, err := st.disp.Once(ctx)
		if err != nil {
			t.Fatalf("worker Once[%d]: err=%v", i, err)
		}
		if !worked {
			break
		}
	}

	for _, opID := range opIDs {
		wOp := st.doAdmin(t, http.MethodGet, "/api/v1/admin/operations/"+opID.String(), "", "", "")
		if wOp.Code != http.StatusOK {
			t.Fatalf("GET /admin/operations/%s status=%d body=%s", opID, wOp.Code, wOp.Body.String())
		}
		var op api.Operation
		if err := json.Unmarshal(wOp.Body.Bytes(), &op); err != nil {
			t.Fatalf("unmarshal operation: %v", err)
		}
		if op.Status != api.OperationStatusSucceeded {
			t.Fatalf("expected operation %s to be succeeded after worker job, got %s", opID, op.Status)
		}
	}

	// Verify GET /api/v1/admin/audit-events over HTTP returns the policy creation and activation audit rows.
	wAudit := st.doAdmin(t, http.MethodGet, "/api/v1/admin/audit-events?limit=50", "", "", "")
	if wAudit.Code != http.StatusOK {
		t.Fatalf("GET /admin/audit-events status=%d body=%s", wAudit.Code, wAudit.Body.String())
	}
	var auditList api.AuditEventList
	if err := json.Unmarshal(wAudit.Body.Bytes(), &auditList); err != nil {
		t.Fatalf("unmarshal AuditEventList: %v", err)
	}
	if len(auditList.Items) < 12 {
		t.Fatalf("expected at least 12 audit events, got %d", len(auditList.Items))
	}
}

func TestAT18_HTTPShadowVsEnforceMemberRatePolicyAndOverrides(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	adminID := uuid.New()
	memberID := uuid.New()
	deviceID := uuid.New()
	nodeID := uuid.New()
	incID := uuid.New()

	st := newHTTPStack(t, pool, adminID, memberID, nodeID, incID)
	base := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	// Seed member, entitlement, member_device, device_binding, and registered node + incarnation.
	if _, err := pool.Exec(ctx, `INSERT INTO members (id, status) VALUES ($1, 'active')`, memberID); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entitlements (member_id, revision, device_limit, valid_from, effective_service_end)
		VALUES ($1, 1, 5, $2, $3)
	`, memberID, base.Add(-24*time.Hour), base.Add(30*24*time.Hour)); err != nil {
		t.Fatalf("insert entitlement: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO member_devices (id, member_id, deployment_id, peer_id, status)
		VALUES ($1, $2, 'dep-at18', 'peer-at18-dev', 'authorized')
	`, deviceID, memberID); err != nil {
		t.Fatalf("insert member_device: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO device_bindings (device_id, member_id, overlay_scope, overlay_ip, binding_revision, valid_from, valid_until, peer_ref)
		VALUES ($1, $2, 'overlay-main', '100.64.0.10', 1, $3, $4, 'peer-at18-dev')
	`, deviceID, memberID, base.Add(-24*time.Hour), base.Add(30*24*time.Hour)); err != nil {
		t.Fatalf("insert device_bindings: %v", err)
	}
	fp := make([]byte, 32)
	fp[0] = 0x42
	if _, err := pool.Exec(ctx, `INSERT INTO nodes (id, lifecycle_state, provisioning_state, applied_config_generation) VALUES ($1, 'registered', 'ready', 0)`, nodeID); err != nil {
		t.Fatalf("insert node: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO node_incarnations (id, node_id, public_key_fingerprint, deployment_id, peer_id, os, arch) VALUES ($1, $2, $3, 'dep-at18', 'peer-at18', 'ubuntu', 'amd64')`, incID, nodeID, fp); err != nil {
		t.Fatalf("insert node_incarnations: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE nodes SET current_incarnation_id = $2, resource_version = resource_version + 1 WHERE id = $1`, nodeID, incID); err != nil {
		t.Fatalf("set current_incarnation_id: %v", err)
	}

	// 1. Create and activate shadow fair_use policy over HTTP.
	shadowPolicy := validFairUsePayload()
	shadowPolicy.Mode = api.FairUsePayloadModeShadow
	shadowPolicy.EffectiveAt = base.Add(-time.Hour)
	createReqBytes, _ := json.Marshal(map[string]any{
		"code":           "at18-fup",
		"type":           "fair_use",
		"schema_version": 1,
		"payload":        shadowPolicy,
		"reason":         "AT18 shadow rollout",
	})
	wShadowCreate := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", string(createReqBytes), "", "")
	if wShadowCreate.Code != http.StatusCreated {
		t.Fatalf("create shadow FUP status=%d body=%s", wShadowCreate.Code, wShadowCreate.Body.String())
	}
	shadowRevID, fupDefID := extractRevisionAndDefinitionIDs(t, wShadowCreate.Body.Bytes())

	actBytes, _ := json.Marshal(api.PolicyActivateRequest{
		RevisionId: openapi_types.UUID(shadowRevID),
		Reason:     "Activate shadow FUP",
	})
	wShadowAct := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies/"+fupDefID.String()+"/activate", string(actBytes), "", `"1"`)
	if wShadowAct.Code != http.StatusAccepted {
		t.Fatalf("activate shadow FUP status=%d body=%s", wShadowAct.Code, wShadowAct.Body.String())
	}

	// Drain policy.activated outbox event.
	_, _ = st.disp.Once(ctx)

	// 2. Submit a usage batch over POST /api/v1/agent/usage-batches with 600 GB (crosses Tier 1 threshold 500 GB).
	ev1, err := usage.WithComputedHash(api.UsageEvent{
		SchemaVersion:     api.UsageEventSchemaVersionN1,
		NodeId:            openapi_types.UUID(nodeID),
		NodeIncarnationId: openapi_types.UUID(incID),
		MeterEpochId:      openapi_types.UUID(uuid.New()),
		EventSeq:          decstr.U64(1),
		IntervalStart:     base,
		IntervalEnd:       base.Add(5 * time.Minute),
		MeteringVersion:   api.MeteringVersion(usage.MeteringVersionForwardAdmissionV1),
		Quality:           api.UsageQualityComplete,
		Records: []api.UsageRecord{
			{
				MemberId:        openapi_types.UUID(memberID),
				DeviceId:        openapi_types.UUID(deviceID),
				BindingRevision: decstr.U64(1),
				PeriodId:        "2026-10",
				UploadBytes:     decstr.U64(200_000_000_000),
				DownloadBytes:   decstr.U64(400_000_000_000),
			},
		},
	})
	if err != nil {
		t.Fatalf("WithComputedHash: %v", err)
	}
	batchBytes, _ := json.Marshal(api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(nodeID),
		NodeIncarnationId: openapi_types.UUID(incID),
		LedgerEpoch:       presence.NewNullNullable[openapi_types.UUID](),
		Events:            []api.UsageEvent{ev1},
	})
	wBatch := st.doAgent(t, http.MethodPost, "/api/v1/agent/usage-batches", string(batchBytes), "")
	if wBatch.Code != http.StatusOK {
		t.Fatalf("POST /agent/usage-batches status=%d body=%s", wBatch.Code, wBatch.Body.String())
	}

	// Drain fup.evaluate outbox event.
	_, _ = st.disp.Once(ctx)

	// Verify in shadow mode:
	// - GET /api/v1/member/usage shows FairUse.State == normal, TierIndex == 0
	// - BuildNodeMemberRatePolicy returns normal plan rate (200 Mbps down / 100 Mbps up)
	wMemShadow := st.doMember(t, http.MethodGet, "/api/v1/member/usage?period_id=2026-10")
	if wMemShadow.Code != http.StatusOK {
		t.Fatalf("GET /member/usage shadow status=%d body=%s", wMemShadow.Code, wMemShadow.Body.String())
	}
	var muShadow api.MemberUsage
	if err := json.Unmarshal(wMemShadow.Body.Bytes(), &muShadow); err != nil {
		t.Fatalf("unmarshal MemberUsage shadow: %v", err)
	}
	if muShadow.FairUse.State != api.FairUseStatusStateNormal || muShadow.Totals.TotalBytes != decstr.U64(600_000_000_000) {
		t.Fatalf("unexpected shadow MemberUsage: state=%s total=%s", muShadow.FairUse.State, muShadow.Totals.TotalBytes)
	}

	polServices, err := worker.NewPoliciesServices(pool, policiestest.New(t).Policies, nil)
	if err != nil {
		t.Fatalf("worker.NewPoliciesServices: %v", err)
	}
	fupEngine := polServices.FUPEngine
	evalShadow, err := fupEngine.Compute(ctx, fup.ComputeInput{
		MemberID: memberID,
		PeriodID: "2026-10",
		Now:      base.Add(6 * time.Minute),
	})
	if err != nil {
		t.Fatalf("fupEngine.Compute shadow: %v", err)
	}
	planRate := &api.RatePair{DownBps: 200_000_000, UpBps: 100_000_000}
	rateShadow := fup.BuildNodeMemberRatePolicy(evalShadow, planRate, nil)
	shadowDown, _ := rateShadow.DownBps.Get()
	if rateShadow.Reason != api.MemberRatePolicyReasonNormal || shadowDown != 200_000_000 {
		t.Fatalf("expected normal rate in shadow mode, got reason=%s down=%d", rateShadow.Reason, shadowDown)
	}

	// 3. Create and activate enforce fair_use policy over HTTP.
	enforcePolicy := shadowPolicy
	enforcePolicy.Mode = api.FairUsePayloadModeEnforce
	enforceCreateBytes, _ := json.Marshal(map[string]any{
		"definition_id":  fupDefID.String(),
		"type":           "fair_use",
		"schema_version": 1,
		"payload":        enforcePolicy,
		"reason":         "AT18 promote to enforce mode",
	})
	wEnforceCreate := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", string(enforceCreateBytes), "", "")
	if wEnforceCreate.Code != http.StatusCreated {
		t.Fatalf("create enforce FUP status=%d body=%s", wEnforceCreate.Code, wEnforceCreate.Body.String())
	}
	enforceRevID, _ := extractRevisionAndDefinitionIDs(t, wEnforceCreate.Body.Bytes())

	actEnforceBytes, _ := json.Marshal(api.PolicyActivateRequest{
		RevisionId: openapi_types.UUID(enforceRevID),
		Reason:     "Activate enforce FUP",
	})
	wEnforceAct := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies/"+fupDefID.String()+"/activate", string(actEnforceBytes), "", `"2"`)
	if wEnforceAct.Code != http.StatusAccepted {
		t.Fatalf("activate enforce FUP status=%d body=%s", wEnforceAct.Code, wEnforceAct.Body.String())
	}

	evalEnforce, err := fupEngine.Compute(ctx, fup.ComputeInput{
		MemberID: memberID,
		PeriodID: "2026-10",
		Now:      base.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("fupEngine.Compute enforce: %v", err)
	}
	if evalEnforce.Watermark == nil {
		t.Fatalf("expected LedgerWatermark on EvaluationResult")
	}
	rateEnforce := fup.BuildNodeMemberRatePolicy(evalEnforce, planRate, nil)
	enforceDown, _ := rateEnforce.DownBps.Get()
	enforceUp, _ := rateEnforce.UpBps.Get()
	if rateEnforce.Reason != api.MemberRatePolicyReasonFairUseTier || enforceDown != 50_000_000 || enforceUp != 25_000_000 {
		t.Fatalf("expected Tier 1 rate (50M/25M, reason=fair_use_tier) in enforce mode, got %+v", rateEnforce)
	}

	// Verify GET /api/v1/member/usage now reflects adjusted state and Applying=true (fail-closed before node ACK).
	wMemEnforce := st.doMember(t, http.MethodGet, "/api/v1/member/usage?period_id=2026-10")
	if wMemEnforce.Code != http.StatusOK {
		t.Fatalf("GET /member/usage enforce status=%d body=%s", wMemEnforce.Code, wMemEnforce.Body.String())
	}
	var muEnforce api.MemberUsage
	if err := json.Unmarshal(wMemEnforce.Body.Bytes(), &muEnforce); err != nil {
		t.Fatalf("unmarshal MemberUsage enforce: %v", err)
	}
	if muEnforce.FairUse.State != api.FairUseStatusStateAdjusted || !muEnforce.FairUse.Applying {
		t.Fatalf("expected state=adjusted applying=true before node ACK, got state=%s applying=%v", muEnforce.FairUse.State, muEnforce.FairUse.Applying)
	}

	// Record node observation with safety_status='passed', control_status='online', and fup_generation="1".
	obsHash := make([]byte, 32)
	if _, err := pool.Exec(ctx, `
		INSERT INTO node_observations (
			node_id, incarnation_id, report_id, payload_hash, observed_at,
			applied_generation, schema_version, checks, control_status, data_health, safety_status
		) VALUES (
			$1, $2, $3, $4, $5,
			1, 1, '{"applied_policy_refs": {"fup_generation": "1"}}'::jsonb, 'online', 'healthy', 'passed'
		)
	`, nodeID, incID, uuid.New(), obsHash, base.Add(11*time.Minute)); err != nil {
		t.Fatalf("insert node_observations: %v", err)
	}

	wMemAcked := st.doMember(t, http.MethodGet, "/api/v1/member/usage?period_id=2026-10")
	if wMemAcked.Code != http.StatusOK {
		t.Fatalf("GET /member/usage after ACK status=%d body=%s", wMemAcked.Code, wMemAcked.Body.String())
	}
	var muAcked api.MemberUsage
	if err := json.Unmarshal(wMemAcked.Body.Bytes(), &muAcked); err != nil {
		t.Fatalf("unmarshal MemberUsage after ACK: %v", err)
	}
	if muAcked.FairUse.Applying {
		t.Fatalf("expected applying=false after serving node ACKed fup_generation 1")
	}
}

type noopMembersCompiler struct{}

func (noopMembersCompiler) RecompileBatch(context.Context, []db.ClaimedEvent) ([]desiredstate.CompiledDesiredState, error) {
	return nil, nil
}

func TestPoliciesAndUsageCompositionSeam(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	polFix := policiestest.New(t)
	useFix := usagetest.New(t)

	validCfg := config.Defaults()
	validCfg.PoliciesConfigFile = polFix.Path
	validCfg.UsageConfigFile = useFix.Path

	// 1. Missing policies_config_file, usage_config_file, or signing_config_file fails with SettingError naming the setting.
	noPolCfg := validCfg
	noPolCfg.PoliciesConfigFile = ""
	_, err := control.NewPoliciesDomain(nil).Build(ctx, control.Dependencies{Config: noPolCfg})
	var se *config.SettingError
	if !errors.As(err, &se) || se.Name != "policies_config_file" {
		t.Fatalf("control domain policies with missing policies_config_file: err=%v, want SettingError(policies_config_file)", err)
	}

	noUsageCfg := validCfg
	noUsageCfg.UsageConfigFile = ""
	_, err = control.NewUsageDomain(nil, nil, nil).Build(ctx, control.Dependencies{Config: noUsageCfg})
	se = nil
	if !errors.As(err, &se) || se.Name != "usage_config_file" {
		t.Fatalf("control domain usage with missing usage_config_file: err=%v, want SettingError(usage_config_file)", err)
	}

	badSignUsage := useFix.Usage
	badSignUsage.SigningConfigFile = ""
	_, err = control.NewUsageDomain(&badSignUsage, &polFix.Policies, nil).Build(ctx, control.Dependencies{Config: validCfg})
	se = nil
	if !errors.As(err, &se) || se.Name != "signing_config_file" {
		t.Fatalf("control domain usage with missing signing_config_file: err=%v, want SettingError(signing_config_file)", err)
	}

	// 2. Invalid policies/usage settings fail startup with config.SettingError naming the setting.
	badPolCfg := polFix.Policies
	badPolCfg.FUPBatchSize = -1
	if _, err := worker.NewPoliciesDomain(&badPolCfg, nil).Build(ctx, worker.Dependencies{Config: validCfg, Pool: pool}); err == nil {
		t.Fatal("expected SettingError for negative fup_batch_size")
	} else {
		var se *config.SettingError
		if !errors.As(err, &se) || se.Name != "fup_batch_size" {
			t.Fatalf("err=%v, want SettingError(fup_batch_size)", err)
		}
	}

	badUsageCfg := useFix.Usage
	badUsageCfg.RawUsageRetentionDays = 0
	if _, err := worker.NewUsageDomain(&badUsageCfg).Build(ctx, worker.Dependencies{Config: validCfg, Pool: pool}); err == nil {
		t.Fatal("expected SettingError for zero raw_usage_retention_days")
	} else {
		var se *config.SettingError
		if !errors.As(err, &se) || se.Name != "raw_usage_retention_days" {
			t.Fatalf("err=%v, want SettingError(raw_usage_retention_days)", err)
		}
	}

	// 3. Worker Built.Context decorates both WithFUPCompiler and membersruntime.WithCompiler.
	built, err := worker.NewPoliciesDomain(nil, noopMembersCompiler{}).Build(ctx, worker.Dependencies{Config: validCfg, Pool: pool})
	if err != nil {
		t.Fatalf("NewPoliciesDomain.Build: %v", err)
	}
	decorated := built.Context(ctx)
	if worker.FUPCompilerFromContext(decorated) == nil {
		t.Fatal("expected FUPCompilerFromContext to be non-nil on decorated worker context")
	}
	if membersruntime.CompilerFromContext(decorated) == nil {
		t.Fatal("expected membersruntime.CompilerFromContext to be non-nil on decorated worker context")
	}
}
