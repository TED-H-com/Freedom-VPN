package contracts

import (
	"net/http"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/go-cmp/cmp"

	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

// specEndpoints is the endpoint table of §14.2 to §14.4, relative to /api/v1.
var specEndpoints = []string{
	// §14.2 Admin (23)
	"GET /admin/nodes",
	"POST /admin/enrollments",
	"GET /admin/enrollments/{enrollment_id}",
	"POST /admin/enrollments/{enrollment_id}/revoke",
	"GET /admin/nodes/{node_id}",
	"PATCH /admin/nodes/{node_id}",
	"PUT /admin/nodes/{node_id}/publication",
	"POST /admin/nodes/{node_id}/revoke",
	"DELETE /admin/nodes/{node_id}",
	"GET /admin/members",
	"GET /admin/members/{member_id}",
	"PATCH /admin/members/{member_id}",
	"POST /admin/members/{member_id}/grants",
	"GET /admin/policies",
	"POST /admin/policies",
	"POST /admin/policies/{definition_id}/activate",
	"GET /admin/releases",
	"POST /admin/releases",
	"POST /admin/update-batches",
	"GET /admin/update-batches/{batch_id}",
	"POST /admin/update-batches/{batch_id}/retry",
	"GET /admin/operations/{operation_id}",
	"GET /admin/audit-events",
	// §14.3 Member (8)
	"GET /member/me",
	"GET /member/devices",
	"DELETE /member/devices/{device_id}",
	"GET /member/exits",
	"GET /member/usage",
	"POST /member/checkout-sessions",
	"POST /member/billing-portal-sessions",
	"GET /member/operations/{operation_id}",
	// §14.4 Agent and webhook (9)
	"POST /agent/enroll",
	"POST /agent/certificates/renew",
	"GET /agent/desired-state",
	"POST /agent/observations",
	"POST /agent/usage-batches",
	"GET /agent/usage-checkpoint",
	"POST /agent/update-results",
	"POST /agent/lease-renewals",
	"POST /webhooks/{provider}",
}

// specStatuses holds the success statuses listed in §14.2 to §14.4.
var specStatuses = map[string][]string{
	"GET /admin/nodes": {"200"}, "POST /admin/enrollments": {"201"}, "GET /admin/enrollments/{enrollment_id}": {"200"},
	"POST /admin/enrollments/{enrollment_id}/revoke": {"200"}, "GET /admin/nodes/{node_id}": {"200"},
	"PATCH /admin/nodes/{node_id}": {"200", "202"}, "PUT /admin/nodes/{node_id}/publication": {"202"},
	"POST /admin/nodes/{node_id}/revoke": {"202"}, "DELETE /admin/nodes/{node_id}": {"202"}, "GET /admin/members": {"200"},
	"GET /admin/members/{member_id}": {"200"}, "PATCH /admin/members/{member_id}": {"200", "202"},
	"POST /admin/members/{member_id}/grants": {"202"}, "GET /admin/policies": {"200"}, "POST /admin/policies": {"201"},
	"POST /admin/policies/{definition_id}/activate": {"202"}, "GET /admin/releases": {"200"}, "POST /admin/releases": {"201", "202"},
	"POST /admin/update-batches": {"202"}, "GET /admin/update-batches/{batch_id}": {"200"},
	"POST /admin/update-batches/{batch_id}/retry": {"202"}, "GET /admin/operations/{operation_id}": {"200"},
	"GET /admin/audit-events": {"200"}, "GET /member/me": {"200"}, "GET /member/devices": {"200"},
	"DELETE /member/devices/{device_id}": {"202"}, "GET /member/exits": {"200"}, "GET /member/usage": {"200"},
	"POST /member/checkout-sessions": {"201", "202"}, "POST /member/billing-portal-sessions": {"201"},
	"GET /member/operations/{operation_id}": {"200"}, "POST /agent/enroll": {"200", "201"},
	"POST /agent/certificates/renew": {"201"}, "GET /agent/desired-state": {"200", "304"}, "POST /agent/observations": {"202"},
	"POST /agent/usage-batches": {"200"}, "GET /agent/usage-checkpoint": {"200"}, "POST /agent/update-results": {"202"},
	"POST /agent/lease-renewals": {"200"}, "POST /webhooks/{provider}": {"200", "202"},
}

type opInfo struct {
	key    string // "METHOD /path"
	method string
	path   string
	item   *openapi3.PathItem
	op     *openapi3.Operation
}

func operations(t *testing.T) []opInfo {
	doc := loadDoc(t)
	var ops []opInfo
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			ops = append(ops, opInfo{key: method + " " + path, method: method, path: path, item: item, op: op})
		}
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].key < ops[j].key })
	return ops
}

func TestAPI01_EndpointCountIs40(t *testing.T) {
	ops := operations(t)
	if len(ops) != 40 {
		t.Fatalf("got %d operations, want 40 (§14.1)", len(ops))
	}
	if len(specEndpoints) != 40 {
		t.Fatalf("spec table has %d entries", len(specEndpoints))
	}
	var got []string
	for _, o := range ops {
		got = append(got, o.key)
	}
	want := append([]string(nil), specEndpoints...)
	sort.Strings(want)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("operations differ from §14.2-§14.4 (-want +got):\n%s", diff)
	}
}

func TestAPI01_SuccessStatusesMatchSpec(t *testing.T) {
	for _, o := range operations(t) {
		var got []string
		for code := range o.op.Responses.Map() {
			if code[0] == '2' || code == "304" {
				got = append(got, code)
			}
		}
		sort.Strings(got)
		if diff := cmp.Diff(specStatuses[o.key], got); diff != "" {
			t.Errorf("%s success statuses (-spec +openapi):\n%s", o.key, diff)
		}
	}
}

func TestAPI01_OperationIDsTagsAndSpecRefs(t *testing.T) {
	idRE := regexp.MustCompile(`^[a-z][A-Za-z0-9]+$`)
	reqRE := regexp.MustCompile(`^[A-Z]+[0-9]{2}$`)
	seen := map[string]bool{}
	for _, o := range operations(t) {
		id := o.op.OperationID
		if !idRE.MatchString(id) || seen[id] {
			t.Errorf("%s: bad or duplicate operationId %q", o.key, id)
		}
		seen[id] = true
		var wantTag string
		switch {
		case strings.HasPrefix(o.path, "/admin/"):
			wantTag = "admin"
		case strings.HasPrefix(o.path, "/member/"):
			wantTag = "member"
		case strings.HasPrefix(o.path, "/agent/"):
			wantTag = "agent"
		case strings.HasPrefix(o.path, "/webhooks/"):
			wantTag = "webhook"
		}
		if len(o.op.Tags) != 1 || o.op.Tags[0] != wantTag {
			t.Errorf("%s: tags %v, want [%s]", o.key, o.op.Tags, wantTag)
		}
		if s, _ := o.op.Extensions["x-fvpn-spec"].(string); !strings.HasPrefix(s, "§14.") {
			t.Errorf("%s: x-fvpn-spec %q must cite §14.x", o.key, s)
		}
		reqs, _ := o.op.Extensions["x-fvpn-requirements"].([]any)
		if len(reqs) == 0 {
			t.Errorf("%s: missing x-fvpn-requirements", o.key)
		}
		for _, r := range reqs {
			if s, _ := r.(string); !reqRE.MatchString(s) {
				t.Errorf("%s: bad requirement id %v", o.key, r)
			}
		}
		if o.op.Description == "" || o.op.Summary == "" {
			t.Errorf("%s: missing summary or description", o.key)
		}
	}
}

func securityNames(op *openapi3.Operation) []string {
	var names []string
	if op.Security == nil {
		return nil
	}
	for _, req := range *op.Security {
		var group []string
		for name := range req {
			group = append(group, name)
		}
		sort.Strings(group)
		names = append(names, strings.Join(group, "+"))
	}
	return names
}

func isUnsafe(method string) bool {
	return method != http.MethodGet && method != http.MethodHead
}

func hasParam(op *openapi3.Operation, in, name string, required bool) bool {
	for _, p := range op.Parameters {
		if p.Value != nil && p.Value.In == in && strings.EqualFold(p.Value.Name, name) {
			return !required || p.Value.Required
		}
	}
	return false
}

// TestSEC01_SecurityPerRole checks the authentication model of §14 and §16.1.
func TestSEC01_SecurityPerRole(t *testing.T) {
	for _, o := range operations(t) {
		sec := securityNames(o.op)
		switch {
		case strings.HasPrefix(o.path, "/admin/"):
			want := "adminSession"
			if isUnsafe(o.method) {
				want = "adminSession+csrfToken"
			}
			if len(sec) != 1 || sec[0] != want {
				t.Errorf("%s: security %v, want [%s]", o.key, sec, want)
			}
		case strings.HasPrefix(o.path, "/member/"):
			want := "memberSession"
			if isUnsafe(o.method) {
				want = "csrfToken+memberSession"
			}
			if len(sec) != 1 || sec[0] != want {
				t.Errorf("%s: security %v, want [%s]", o.key, sec, want)
			}
		case o.path == "/agent/enroll":
			if len(sec) != 1 || sec[0] != "enrollmentToken" {
				t.Errorf("%s: security %v", o.key, sec)
			}
		case strings.HasPrefix(o.path, "/agent/"):
			if o.op.Security == nil || len(*o.op.Security) != 0 || o.op.Extensions["x-fvpn-auth"] != "agent-mtls" {
				t.Errorf("%s: agent endpoints declare security: [] and x-fvpn-auth: agent-mtls", o.key)
			}
		case strings.HasPrefix(o.path, "/webhooks/"):
			if len(sec) != 1 || sec[0] != "providerWebhookSignature" {
				t.Errorf("%s: security %v", o.key, sec)
			}
		}
	}
	doc := loadDoc(t)
	if _, ok := doc.Extensions["x-fvpn-mutual-tls"]; !ok {
		t.Error("missing top-level x-fvpn-mutual-tls")
	}
	if s := doc.Components.SecuritySchemes["enrollmentToken"].Value; s.In != "header" || s.Name != "Authorization" {
		t.Error("enrollment token must be the Authorization header")
	}
}

// TestAPI01_IdempotencyAndPreconditions checks §14.5 header requirements.
func TestAPI01_IdempotencyAndPreconditions(t *testing.T) {
	ifMatch := map[string]bool{
		"PATCH /admin/nodes/{node_id}": true, "PUT /admin/nodes/{node_id}/publication": true,
		"POST /admin/nodes/{node_id}/revoke": true, "DELETE /admin/nodes/{node_id}": true,
		"PATCH /admin/members/{member_id}": true, "POST /admin/policies/{definition_id}/activate": true,
	}
	for _, o := range operations(t) {
		adminOrMember := strings.HasPrefix(o.path, "/admin/") || strings.HasPrefix(o.path, "/member/")
		if adminOrMember && isUnsafe(o.method) && !hasParam(o.op, "header", "Idempotency-Key", true) {
			t.Errorf("%s: side-effecting command must require Idempotency-Key", o.key)
		}
		if !isUnsafe(o.method) && hasParam(o.op, "header", "Idempotency-Key", false) {
			t.Errorf("%s: GET must not take Idempotency-Key", o.key)
		}
		if got := hasParam(o.op, "header", "If-Match", true); got != ifMatch[o.key] {
			t.Errorf("%s: If-Match required=%v, want %v", o.key, got, ifMatch[o.key])
		}
		if (o.op.Responses.Status(428) != nil) != ifMatch[o.key] || (o.op.Responses.Status(412) != nil) != ifMatch[o.key] {
			t.Errorf("%s: 412/428 responses must accompany If-Match", o.key)
		}
	}
	ds := loadDoc(t).Paths.Value("/agent/desired-state").Get
	if !hasParam(ds, "header", "If-None-Match", false) || ds.Responses.Status(304) == nil {
		t.Error("desired-state must support If-None-Match and 304")
	}
}

// TestAPI01_ResponsesUseEnvelopeAndHeaders checks error envelopes, request
// IDs, Location on accepted operations and Retry-After.
func TestAPI01_ResponsesUseEnvelopeAndHeaders(t *testing.T) {
	for _, o := range operations(t) {
		for code, ref := range o.op.Responses.Map() {
			r := ref.Value
			if _, ok := r.Headers["X-Request-ID"]; !ok {
				t.Errorf("%s %s: missing X-Request-ID", o.key, code)
			}
			if code[0] == '4' || code[0] == '5' {
				mt := r.Content.Get("application/json")
				if mt == nil || mt.Schema == nil || mt.Schema.Ref != "#/components/schemas/ErrorEnvelope" {
					t.Errorf("%s %s: error response must use ErrorEnvelope", o.key, code)
				}
			}
			if code == "429" || code == "503" {
				if _, ok := r.Headers["Retry-After"]; !ok {
					t.Errorf("%s %s: missing Retry-After", o.key, code)
				}
			}
			if mt := r.Content.Get("application/json"); mt != nil && mt.Schema != nil && mt.Schema.Ref == "#/components/schemas/Operation" {
				if code != "202" && code != "200" {
					t.Errorf("%s %s: operations are returned with 202", o.key, code)
				}
				if code == "202" {
					if _, ok := r.Headers["Location"]; !ok {
						t.Errorf("%s %s: accepted operation without Location", o.key, code)
					}
				}
			}
			if code == "201" {
				if _, ok := r.Headers["Location"]; !ok && !strings.HasPrefix(o.path, "/agent/") && !strings.HasPrefix(o.path, "/member/") {
					t.Errorf("%s 201: missing Location", o.key)
				}
			}
		}
		for _, code := range []int{400, 401, 429, 500, 503} {
			if code == 400 && !isUnsafe(o.method) && len(o.op.Parameters) == 0 {
				continue // nothing the caller sends can be malformed
			}
			if o.op.Responses.Status(code) == nil {
				t.Errorf("%s: missing %d response", o.key, code)
			}
		}
	}
}

// TestAPI01_CursorPagination checks cursor and limit (default 50, max 100).
func TestAPI01_CursorPagination(t *testing.T) {
	doc := loadDoc(t)
	limit := doc.Components.Parameters["Limit"].Value.Schema.Value
	if limit.Default != 50.0 || limit.Max == nil || *limit.Max != 100 || limit.Min == nil || *limit.Min != 1 {
		t.Fatalf("limit must default to 50 with maximum 100: %+v", limit)
	}
	for _, o := range operations(t) {
		if o.method != http.MethodGet {
			continue
		}
		mt := o.op.Responses.Status(200).Value.Content.Get("application/json")
		s := mt.Schema.Value
		_, hasItems := s.Properties["items"]
		_, hasTargets := s.Properties["targets"]
		if !hasItems && !hasTargets {
			continue
		}
		if !hasParam(o.op, "query", "cursor", false) || !hasParam(o.op, "query", "limit", false) {
			t.Errorf("%s: list without cursor/limit", o.key)
		}
		if _, ok := s.Properties["next_cursor"]; !ok {
			t.Errorf("%s: list without next_cursor", o.key)
		}
	}
}

// TestAPI01_ErrorCodesMatchApierr keeps the ErrorCode enum identical to the
// pkg/apierr table.
func TestAPI01_ErrorCodesMatchApierr(t *testing.T) {
	var enum []string
	for _, v := range loadDoc(t).Components.Schemas["ErrorCode"].Value.Enum {
		enum = append(enum, v.(string))
	}
	var codes []string
	for _, s := range apierr.All() {
		codes = append(codes, string(s.Code))
	}
	sort.Strings(enum)
	sort.Strings(codes)
	if diff := cmp.Diff(codes, enum); diff != "" {
		t.Fatalf("ErrorCode enum differs from pkg/apierr (-apierr +openapi):\n%s", diff)
	}
}

const (
	utcPattern = `^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$`
	decPattern = `^(0|[1-9][0-9]*)$`
	posPattern = `^[1-9][0-9]*$`
	decImport  = "github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

type schemaVisit struct {
	path string
	name string // property name, or "" for non-properties
	s    *openapi3.Schema
}

// walkSchemas visits every inline schema of the components (refs are
// visited at their definition).
func walkSchemas(t *testing.T, fn func(v schemaVisit)) {
	doc := loadDoc(t)
	var walk func(path, name string, ref *openapi3.SchemaRef)
	walk = func(path, name string, ref *openapi3.SchemaRef) {
		if ref == nil || ref.Value == nil {
			return
		}
		if ref.Ref != "" && path != "" && !strings.HasSuffix(path, "#") {
			return
		}
		s := ref.Value
		fn(schemaVisit{path, name, s})
		for pn, p := range s.Properties {
			walk(path+"/"+pn, pn, p)
		}
		walk(path+"/items", "", s.Items)
		for i, x := range s.AllOf {
			walk(path+"/allOf/"+string(rune('0'+i)), name, x)
		}
		for i, x := range s.OneOf {
			walk(path+"/oneOf/"+string(rune('0'+i)), name, x)
		}
		if s.AdditionalProperties.Schema != nil {
			walk(path+"/additionalProperties", "", s.AdditionalProperties.Schema)
		}
	}
	names := make([]string, 0, len(doc.Components.Schemas))
	for n := range doc.Components.Schemas {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		walk(n+"#", "", &openapi3.SchemaRef{Value: doc.Components.Schemas[n].Value})
	}
}

// TestMTR03_DecimalStringConvention enforces the §10.4 / §14.1 integer
// precision contract on every field.
func TestMTR03_DecimalStringConvention(t *testing.T) {
	decName := regexp.MustCompile(`(_bytes|_seq|generation|revision|resource_version)$`)
	count := 0
	walkSchemas(t, func(v schemaVisit) {
		s := v.s
		goType, _ := s.Extensions["x-go-type"].(string)
		isDec := s.Pattern == decPattern || s.Pattern == posPattern
		if isDec || goType == "decstr.U64" {
			count++
			imp, _ := s.Extensions["x-go-type-import"].(map[string]any)
			if !s.Type.Is("string") || s.MaxLength == nil || *s.MaxLength != 20 || goType != "decstr.U64" || imp["path"] != decImport || !isDec {
				t.Errorf("%s: decimal string must be type string, pattern %s, maxLength 20, x-go-type decstr.U64 with import", v.path, decPattern)
			}
		}
		if v.name != "" && decName.MatchString(v.name) && !isDec && len(s.AllOf) == 0 {
			t.Errorf("%s: field %q must be a decimal string", v.path, v.name)
		}
	})
	if count < 70 {
		t.Fatalf("only %d decimal string fields found; walker broken?", count)
	}
}

// TestAPI01_IDTimeAndMoneyConventions enforces UUID, date-time and money
// conventions of §14.1.
func TestAPI01_IDTimeAndMoneyConventions(t *testing.T) {
	opaqueIDs := map[string]bool{"key_id": true, "peer_id": true, "netbird_peer_id": true, "build_id": true, "os_id": true,
		"profile_id": true, "migration_id": true, "period_id": true, "provider_price_ref": true}
	timeNames := regexp.MustCompile(`(_at|^valid_from|^valid_until|_start|_end|^paid_through|^grace_until|_valid_until)$`)
	walkSchemas(t, func(v schemaVisit) {
		s := v.s
		if v.name == "" || len(s.AllOf) > 0 || s.Type == nil {
			return
		}
		if strings.HasSuffix(v.name, "_id") && !opaqueIDs[v.name] && s.Type.Is("string") && s.Format != "uuid" {
			t.Errorf("%s: %q must be format uuid", v.path, v.name)
		}
		if timeNames.MatchString(v.name) && !strings.HasSuffix(v.name, "_seq") && s.Type.Is("string") && s.Format != "date-time" {
			t.Errorf("%s: %q must be format date-time", v.path, v.name)
		}
		if s.Format == "date-time" && s.Pattern != utcPattern {
			t.Errorf("%s: date-time fields must be UTC with Z (pattern %s)", v.path, utcPattern)
		}
		if v.name == "amount_minor" && (!s.Type.Is("integer") || s.Format != "int64") {
			t.Errorf("%s: amount_minor must be int64", v.path)
		}
		if s.Type.Is("number") {
			t.Errorf("%s: floating point numbers are not allowed", v.path)
		}
		if s.Type.Is("integer") && len(s.Enum) == 0 && (s.Max == nil || *s.Max > 9007199254740991 || s.Min == nil) {
			t.Errorf("%s: integers must be bounded within ±(2^53-1)", v.path)
		}
	})
}

// TestAPI01_RequestBodiesAreClosed checks that request bodies only accept
// allowlisted fields (§14.1).
func TestAPI01_RequestBodiesAreClosed(t *testing.T) {
	doc := loadDoc(t)
	for _, o := range operations(t) {
		if o.op.RequestBody == nil || o.path == "/webhooks/{provider}" {
			continue
		}
		ref := o.op.RequestBody.Value.Content.Get("application/json").Schema
		name := strings.TrimPrefix(ref.Ref, "#/components/schemas/")
		s := doc.Components.Schemas[name].Value
		variants := []*openapi3.Schema{s}
		if len(s.OneOf) > 0 {
			variants = nil
			for _, v := range s.OneOf {
				variants = append(variants, v.Value)
			}
		}
		for _, v := range variants {
			if v.AdditionalProperties.Has == nil || *v.AdditionalProperties.Has {
				t.Errorf("%s: request body %s must set additionalProperties: false", o.key, name)
			}
		}
	}
}

// TestUPD04_EnumsMatchSpec pins state enums to §5, §9.2, §13.2 and §14.5.
func TestUPD04_EnumsMatchSpec(t *testing.T) {
	want := map[string][]string{
		"LifecycleState":      {"enrolling", "registered", "revoked", "retired"},
		"ProvisioningState":   {"pending", "applying", "ready", "failed"},
		"PublicationIntent":   {"unpublished", "published", "disabled"},
		"PublicationObserved": {"absent", "applying", "active", "withdrawn", "error"},
		"ControlStatus":       {"online", "stale", "offline"},
		"DataHealth":          {"unknown", "healthy", "degraded", "failed"},
		"SafetyStatus":        {"unknown", "passed", "failed", "expired"},
		"EntitlementState":    {"pending_payment", "active", "cancel_at_period_end", "past_due", "grace", "expired", "suspended", "closed"},
		"OperationStatus":     {"pending", "running", "succeeded", "partially_succeeded", "failed", "superseded"},
		"UpdateBatchState":    {"accepted", "waiting", "running", "completed", "completed_with_exceptions"},
		"PolicyType":          {"plan", "fair_use", "dns_profile", "domain_rule_set", "threat_feed", "release_support"},
		"UsageQuality":        {"complete", "gap", "boundary_estimated"},
		"MeteringVersion":     {"forward_admission_v1"},
	}
	doc := loadDoc(t)
	for name, values := range want {
		var got []string
		for _, v := range doc.Components.Schemas[name].Value.Enum {
			got = append(got, v.(string))
		}
		if diff := cmp.Diff(values, got); diff != "" {
			t.Errorf("%s (-spec +openapi):\n%s", name, diff)
		}
	}
	// §5 update states are all present in UpdateJobState.
	job := map[string]bool{}
	for _, v := range doc.Components.Schemas["UpdateJobState"].Value.Enum {
		job[v.(string)] = true
	}
	for _, s := range []string{"queued", "waiting_online", "downloading", "verifying", "applying", "checking", "succeeded", "failed", "rolled_back", "superseded"} {
		if !job[s] {
			t.Errorf("UpdateJobState lacks %s", s)
		}
	}
	// Batch counts are the §13.10.4 keys.
	counts := doc.Components.Schemas["UpdateBatchCounts"].Value
	var keys []string
	for k := range counts.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	wantKeys := []string{"already_current", "applying", "cancelled", "checking", "downloading", "failed", "queued", "rolled_back", "succeeded", "superseded", "verifying", "waiting_online"}
	if diff := cmp.Diff(wantKeys, keys); diff != "" {
		t.Errorf("UpdateBatchCounts keys:\n%s", diff)
	}
}
